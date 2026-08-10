package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"github.com/gknguyen/litelens/internal/config"
	"github.com/gknguyen/litelens/internal/dto"
	"github.com/gknguyen/litelens/internal/plugin"
	"github.com/gknguyen/litelens/internal/plugin/pb"
)

// pluginsRootDir returns the directory all installed plugins live under. If a custom
// plugins directory is configured in settings, returns that; otherwise returns ~/.litelens/plugins.
func (a *App) pluginsRootDir() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.settings.PluginsDir != "" {
		return a.settings.PluginsDir
	}
	return filepath.Join(os.ExpandEnv("$HOME"), ".litelens", "plugins")
}

// PluginAssetDir resolves the actual on-disk directory for an installed
// plugin, for wiring the plugin asset HTTP handler (internal/plugin.NewPluginAssetHandler)
// in main.go. Derived from the loader's binary path rather than
// pluginInstallDir(pluginID) — restoreInstalledPlugins intentionally allows
// the on-disk directory name to differ from the plugin ID (e.g. a local dev
// build under plugins/helm/.output/ with id "helm"), so reconstructing the
// path from pluginsRootDir()+pluginID would 404 for that case. Same
// resolution getInstalledPluginInfo uses.
func (a *App) PluginAssetDir(pluginID string) (string, bool) {
	a.pluginsMu.RLock()
	loader, ok := a.pluginLoaders[pluginID]
	a.pluginsMu.RUnlock()
	if !ok {
		return "", false
	}
	return filepath.Dir(loader.BinaryPath()), true
}

// pluginInstallDir returns the install directory for a single plugin: {pluginsRootDir}/{pluginID}.
func (a *App) pluginInstallDir(pluginID string) string {
	return filepath.Join(a.pluginsRootDir(), pluginID)
}

// restoreInstalledPlugins discovers and restores plugin loaders from disk.
// It runs synchronously during Startup to ensure pluginLoaders is populated
// before any frontend requests arrive. For each plugin directory found,
// it validates the binary is present and executable and the metadata file
// holds a well-formed checksum, then marks the plugin as READY so
// GetInstalledPlugin reports correctly.
func (a *App) restoreInstalledPlugins() {
	pluginsRoot := a.pluginsRootDir()

	// If plugins directory doesn't exist, return silently (first run).
	entries, err := os.ReadDir(pluginsRoot)
	if err != nil {
		// Directory doesn't exist or can't be read; silently skip.
		return
	}

	for _, entry := range entries {
		// Only process directories.
		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()

		// Validate directory name to prevent path traversal.
		if !plugin.ValidPluginID(dirName) {
			continue
		}

		// Read and parse metadata from .plugin-metadata.json FIRST.
		// Legacy formats (flat schema, .bundle-sha256-only) are no longer supported.
		metadataPath := filepath.Join(pluginsRoot, dirName, dto.PluginMetadataFile)
		metadataData, metadataErr := os.ReadFile(metadataPath)
		if metadataErr != nil {
			// No metadata file found; skip this directory.
			continue
		}

		var metadata dto.PluginMetadata
		if err := json.Unmarshal(metadataData, &metadata); err != nil {
			// Failed to parse metadata; skip this directory.
			continue
		}

		// Extract plugin ID from metadata. This is the sole source of truth for identity.
		pluginID := metadata.ID
		if pluginID == "" || !plugin.ValidPluginID(pluginID) {
			// Missing or invalid ID field; skip this directory.
			continue
		}

		// Determine the binary base name from metadata; fall back to old hardcoded convention
		// when the field is absent (pre-existing installs).
		binaryName := metadata.Assets.BinaryName
		if binaryName == "" {
			binaryName = fmt.Sprintf("plugin-%s", pluginID)
		}
		if runtime.GOOS == dto.WindowsGOOS {
			binaryName += ".exe"
		}
		binaryPath := filepath.Join(pluginsRoot, dirName, binaryName)

		// Binary must exist, and on non-Windows must be executable.
		binInfo, binErr := os.Stat(binaryPath)
		if binErr != nil {
			continue
		}
		if runtime.GOOS != dto.WindowsGOOS && binInfo.Mode()&0111 == 0 {
			continue
		}

		// Validate bundle checksum format.
		if !plugin.BundleChecksumHexRe.MatchString(metadata.Bundle.SHA256) {
			// Invalid checksum; skip this directory.
			continue
		}

		// Both binary and metadata exist and look valid. Create a loader and mark it READY.
		loader := plugin.NewPluginLoader(pluginID, binaryPath)
		loader.SetStatus(dto.PluginStatusReady)

		// Add to pluginLoaders map if not already present.
		a.pluginsMu.Lock()
		if _, exists := a.pluginLoaders[pluginID]; !exists {
			a.pluginLoaders[pluginID] = loader
		}
		a.pluginsMu.Unlock()
	}
}

// prewarmRestoredPlugins launches any installed-but-not-yet-launched plugin
// loaders now that a cluster context is active. restoreInstalledPlugins marks
// restored plugins READY without launching them (the kubeconfig needed for
// Launch isn't known until a context connects), so without this the first
// proxied RPC (e.g. ListHelmCharts) would pay the launch/handshake cost
// synchronously on the Wails IPC path. Runs in the background so a slow or
// failing launch never blocks Connect.
func (a *App) prewarmRestoredPlugins(contextName string) {
	kubeconfigPath, err := a.GetContextKubeconfigPath(contextName)
	if err != nil {
		return
	}

	a.pluginsMu.RLock()
	loaders := make([]*plugin.PluginLoader, 0, len(a.pluginLoaders))
	for _, loader := range a.pluginLoaders {
		loaders = append(loaders, loader)
	}
	a.pluginsMu.RUnlock()

	for _, loader := range loaders {
		if loader.Status() == dto.PluginStatusReady && loader.GetClient() == nil {
			_ = loader.Launch(context.Background(), kubeconfigPath)
		}
	}
}

// pluginOperationInProgress reports whether any installed plugin is currently
// installing, or any removal is in flight — used to block plugins-directory
// changes that would race with a concurrent install/remove.
func (a *App) pluginOperationInProgress() bool {
	a.pluginsMu.RLock()
	defer a.pluginsMu.RUnlock()
	for _, loader := range a.pluginLoaders {
		if loader.Status() == dto.PluginStatusInstalling {
			return true
		}
	}
	return len(a.removingPluginIDs) > 0
}

// onPluginsDirChanged rebuilds a.pluginLoaders after the configured plugins
// directory changes: it gracefully shuts down every loader created against
// the OLD directory (their baked-in binaryPath is now stale) and rescans the
// NEW root via restoreInstalledPlugins. Callers must invoke this only after
// confirming (via pluginOperationInProgress) that no install/removal is
// in flight, and must not hold a.mu when calling this (restoreInstalledPlugins
// -> pluginsRootDir() acquires a.mu.RLock()).
func (a *App) onPluginsDirChanged() {
	a.pluginsMu.Lock()
	for _, loader := range a.pluginLoaders {
		if err := loader.Shutdown(); err != nil {
			log.Printf("plugin shutdown during directory change failed: %v", err)
		}
	}
	a.pluginLoaders = make(map[string]*plugin.PluginLoader)
	a.pluginsMu.Unlock()

	a.restoreInstalledPlugins()
}

// GetInstalledPlugins returns the full status info (as returned by GetInstalledPlugin)
// for every installed plugin, sorted by plugin ID for deterministic output. Each entry
// additionally carries PluginID so callers don't need a separate per-ID round trip.
// Returns an empty slice (not nil) for JSON serialization to produce [] not null.
func (a *App) GetInstalledPlugins() []dto.InstalledPlugin {
	a.pluginsMu.RLock()
	ids := make([]string, 0, len(a.pluginLoaders))
	for id := range a.pluginLoaders {
		ids = append(ids, id)
	}
	a.pluginsMu.RUnlock()

	sort.Strings(ids)

	result := make([]dto.InstalledPlugin, 0, len(ids))
	for _, id := range ids {
		info := a.getInstalledPluginInfo(id)
		info.PluginID = id
		result = append(result, info)
	}
	return result
}

// GetInstalledPlugin returns the status of a plugin by ID.
func (a *App) GetInstalledPlugin(pluginID string) dto.InstalledPlugin {
	// Validate pluginID to prevent path traversal
	if !plugin.ValidPluginID(pluginID) {
		return dto.InstalledPlugin{
			Status: dto.PluginStatusNotInstalled.String(),
			Error:  "invalid plugin ID",
		}
	}

	return a.getInstalledPluginInfo(pluginID)
}

// getInstalledPluginInfo builds the status DTO for a single plugin ID. Callers are
// responsible for validating pluginID beforehand (GetInstalledPlugins only ever
// passes IDs sourced from a.pluginLoaders, which are already validated on insert).
func (a *App) getInstalledPluginInfo(pluginID string) dto.InstalledPlugin {
	a.pluginsMu.RLock()
	loader, ok := a.pluginLoaders[pluginID]
	a.pluginsMu.RUnlock()

	if !ok {
		return dto.InstalledPlugin{
			Status:   dto.PluginStatusNotInstalled.String(),
			Progress: 0,
		}
	}

	status := loader.Status()
	progress := loader.Progress()
	errMsg := ""
	if status == dto.PluginStatusCrashed {
		errMsg = loader.LastError()
	}

	// Resolve the plugin's actual on-disk directory from the loader's binary
	// path rather than assuming pluginInstallDir(pluginID) — restoreInstalledPlugins
	// intentionally allows the directory name to differ from the plugin ID
	// (e.g. a local dev build under plugins/helm/.output/ with id "helm"), so
	// re-deriving the path from pluginID here would silently miss the metadata.
	pluginDir := filepath.Dir(loader.BinaryPath())

	// Read metadata from disk if it exists
	metadataFile := filepath.Join(pluginDir, dto.PluginMetadataFile)
	bundleChecksum := "0000000000000000000000000000000000000000000000000000000000000000"
	installedVersion := ""
	var manifest dto.Manifest
	var releaseTag, installedAt string

	if data, err := os.ReadFile(metadataFile); err == nil {
		var metadata dto.PluginMetadata
		if err := json.Unmarshal(data, &metadata); err == nil {
			bundleChecksum = metadata.Bundle.SHA256
			installedVersion = plugin.NormalizeVersion(metadata.ReleaseTag)
			manifest = metadata.Manifest
			releaseTag = metadata.ReleaseTag
			installedAt = metadata.InstalledAt
		}
	}

	var size int64
	if status == dto.PluginStatusReady || status == dto.PluginStatusCrashed {
		size = plugin.DirSize(pluginDir)
	}

	return dto.InstalledPlugin{
		Manifest:         manifest,
		ReleaseTag:       releaseTag,
		InstalledAt:      installedAt,
		Status:           status.String(),
		Error:            errMsg,
		Progress:         progress,
		BundleChecksum:   bundleChecksum,
		InstalledVersion: installedVersion,
		Size:             size,
	}
}

// InstallPlugin initiates installation of a plugin by ID, optionally targeting a specific release tag,
// from a specified marketplace source. sourceURL identifies which marketplace (empty = default, otherwise a repo URL).
// If targetTag is empty, fetches the latest release. Otherwise, fetches the specified tag.
// Sets status to INSTALLING synchronously and returns immediately.
// Actual download/verification happens asynchronously.
// Plugin installation is a global metadata-level operation and succeeds regardless of
// active cluster context. Actual plugin feature usage (e.g., helmPluginClient for Helm)
// gates on active context separately, which is correct.
func (a *App) InstallPlugin(pluginID, targetTag, sourceURL string) error {
	// Validate pluginID to prevent path traversal
	if !plugin.ValidPluginID(pluginID) {
		return fmt.Errorf("invalid plugin ID: %q", pluginID)
	}

	a.pluginsMu.Lock()
	// Check if a removal is in progress for this plugin
	if a.removingPluginIDs[pluginID] {
		a.pluginsMu.Unlock()
		return fmt.Errorf("cannot install plugin %q while a removal is in progress", pluginID)
	}

	loader, ok := a.pluginLoaders[pluginID]
	if !ok {
		// Create a new loader if it doesn't exist
		// Plugin binary path will be ~/.litelens/plugins/{pluginID}/plugin-{pluginID}
		// Add .exe suffix on Windows
		binaryName := fmt.Sprintf("plugin-%s", pluginID)
		if runtime.GOOS == dto.WindowsGOOS {
			binaryName += ".exe"
		}
		binaryPath := filepath.Join(a.pluginInstallDir(pluginID), binaryName)
		loader = plugin.NewPluginLoader(pluginID, binaryPath)
		a.pluginLoaders[pluginID] = loader
	}
	a.pluginsMu.Unlock()

	// Set status to INSTALLING
	loader.SetStatus(dto.PluginStatusInstalling)

	// Resolve token and baseURL based on sourceURL. Settings.AccessToken is
	// reserved for the app's own update-check downloads and must never be used
	// as a fallback here: each marketplace repository uses its own configured
	// AccessToken only (empty for the public default marketplace, or if the
	// user left a private repo's token unset — the subsequent fetch will fail
	// naturally with an auth error in that case).
	a.mu.RLock()
	baseURL := sourceURL
	token := ""
	for _, repo := range a.settings.MarketplaceRepositories {
		if repo.URL == sourceURL {
			token = repo.AccessToken
			break
		}
	}
	a.mu.RUnlock()

	// Start async installation in a goroutine
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		// 1. Fetch release from GitHub (specific tag if provided, otherwise latest)
		assets, tag, err := plugin.FetchRelease(ctx, baseURL, token, targetTag)
		if err != nil {
			loader.SetStatusWithError(dto.PluginStatusCrashed, fmt.Sprintf("fetch release: %v", err))
			return
		}

		// 2. Fetch and parse manifest
		manifest, err := plugin.FetchManifest(ctx, assets, pluginID, token)
		if err != nil {
			loader.SetStatusWithError(dto.PluginStatusCrashed, fmt.Sprintf("fetch manifest: %v", err))
			return
		}

		// 3. Check platform compatibility
		if !plugin.IsPlatformSupported(manifest, runtime.GOOS, runtime.GOARCH) {
			errMsg := fmt.Sprintf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
			loader.SetStatusWithError(dto.PluginStatusIncompatible, errMsg)
			return
		}

		// 4. Check host version compatibility
		compatible, err := plugin.IsHostVersionCompatible(a.GetVersion(), manifest.MinimumHostVersion, manifest.MaximumHostVersion)
		if err != nil {
			loader.SetStatusWithError(dto.PluginStatusCrashed, fmt.Sprintf("version check: %v", err))
			return
		}
		if !compatible {
			errMsg := fmt.Sprintf("host version %s not compatible with plugin (requires %s to %s)",
				a.GetVersion(), manifest.MinimumHostVersion, manifest.MaximumHostVersion)
			loader.SetStatusWithError(dto.PluginStatusIncompatible, errMsg)
			return
		}

		// 5. Resolve asset names and locate URLs
		binaryAsset, bundleAsset := plugin.ResolveAssetNames(pluginID, runtime.GOOS, runtime.GOARCH)

		binaryURL, ok := assets[binaryAsset]
		if !ok {
			loader.SetStatusWithError(dto.PluginStatusCrashed, fmt.Sprintf("binary asset %q not found in release %s", binaryAsset, tag))
			return
		}

		bundleURL, ok := assets[bundleAsset]
		if !ok {
			loader.SetStatusWithError(dto.PluginStatusCrashed, fmt.Sprintf("bundle asset %q not found in release %s", bundleAsset, tag))
			return
		}

		pluginDir := a.pluginInstallDir(pluginID)

		// Compute the REAL binary name and dist dir from the manifest.
		// The loader was created with a provisional binary name before the manifest was fetched;
		// now that we have the manifest, update to use the manifest-driven names.
		realBinaryName := manifest.Assets.BinaryName
		if realBinaryName == "" {
			realBinaryName = fmt.Sprintf("plugin-%s", pluginID)
		}
		if runtime.GOOS == dto.WindowsGOOS {
			realBinaryName += ".exe"
		}
		binaryPath := filepath.Join(pluginDir, realBinaryName)
		loader.SetBinaryPath(binaryPath)

		realBundleDir := manifest.Assets.BundleDir
		if realBundleDir == "" {
			realBundleDir = "dist"
		}
		distDir := filepath.Join(pluginDir, realBundleDir)
		metadataPath := filepath.Join(pluginDir, dto.PluginMetadataFile)

		// 5b. Back up any existing installation's assets before touching them,
		// so a failed download/verify/extract below can be rolled back instead
		// of leaving a previously-working plugin half-upgraded.
		backup, err := plugin.BackupPluginAssets(pluginDir, binaryPath, distDir, metadataPath)
		if err != nil {
			loader.SetStatusWithError(dto.PluginStatusCrashed, fmt.Sprintf("backup existing assets: %v", err))
			return
		}
		installSucceeded := false
		defer func() {
			if installSucceeded {
				if err := backup.Discard(); err != nil {
					log.Printf("plugin %q: discard asset backup: %v", pluginID, err)
				}
				return
			}
			if err := backup.Restore(); err != nil {
				log.Printf("plugin %q: restore asset backup: %v", pluginID, err)
			}
		}()

		// 6. Download and verify binary
		if err := plugin.DownloadToFile(ctx, binaryURL, binaryPath, token, func(pct int) {
			// Map 0-100 binary download to 0-50 overall progress
			loader.SetProgress(pct / 2)
		}); err != nil {
			_ = os.Remove(binaryPath)
			loader.SetStatusWithError(dto.PluginStatusCrashed, fmt.Sprintf("download binary: %v", err))
			return
		}

		// Prefer platform-specific checksum from manifest.Binaries map, fall back to manifest.Binary
		platformKey := runtime.GOOS + "-" + runtime.GOARCH
		expectedSHA256 := manifest.Binary.SHA256
		if len(manifest.Binaries) > 0 {
			if platformAsset, ok := manifest.Binaries[platformKey]; ok {
				expectedSHA256 = platformAsset.SHA256
			}
		}

		if err := plugin.VerifySHA256(binaryPath, expectedSHA256); err != nil {
			_ = os.Remove(binaryPath)
			loader.SetStatusWithError(dto.PluginStatusCrashed, fmt.Sprintf("verify binary: %v", err))
			return
		}

		// Make binary executable on Unix
		if runtime.GOOS != dto.WindowsGOOS {
			if err := os.Chmod(binaryPath, 0755); err != nil {
				_ = os.Remove(binaryPath)
				loader.SetStatusWithError(dto.PluginStatusCrashed, fmt.Sprintf("chmod binary: %v", err))
				return
			}
		}

		// 7. Download and verify bundle (tar.gz archive)
		bundleArchivePath := filepath.Join(pluginDir, "dist-bundle.tar.gz.tmp")

		if err := plugin.DownloadToFile(ctx, bundleURL, bundleArchivePath, token, func(pct int) {
			// Map 0-100 bundle download to 50-100 overall progress
			loader.SetProgress(50 + pct/2)
		}); err != nil {
			_ = os.Remove(bundleArchivePath)
			loader.SetStatusWithError(dto.PluginStatusCrashed, fmt.Sprintf("download bundle: %v", err))
			return
		}

		if err := plugin.VerifySHA256(bundleArchivePath, manifest.Bundle.SHA256); err != nil {
			_ = os.Remove(bundleArchivePath)
			loader.SetStatusWithError(dto.PluginStatusCrashed, fmt.Sprintf("verify bundle: %v", err))
			return
		}

		// Remove any stale dist/ directory before extraction to avoid leftover chunk files
		// from a previous version (chunk filenames are content-hashed and change between versions).
		// Normally already gone since BackupPluginAssets moved it aside above.
		_ = os.RemoveAll(distDir)

		// Extract the archive
		if err := plugin.ExtractTarGz(bundleArchivePath, distDir); err != nil {
			_ = os.RemoveAll(distDir)
			_ = os.Remove(bundleArchivePath)
			loader.SetStatusWithError(dto.PluginStatusCrashed, fmt.Sprintf("extract bundle: %v", err))
			return
		}

		// Clean up the temporary archive file
		_ = os.Remove(bundleArchivePath)

		// 7b. Download the logo asset, if the manifest declares one. Best-effort:
		// a missing or failed logo download must not fail the install, since it's
		// cosmetic only (PluginLogo falls back to a placeholder icon on load error).
		if manifest.Assets.Logo != "" {
			logoAssetName := plugin.ResolveLogoAssetName(pluginID, manifest.Assets.Logo)
			if logoURL, ok := assets[logoAssetName]; ok {
				logoPath := filepath.Join(pluginDir, manifest.Assets.Logo)
				if err := plugin.DownloadToFile(ctx, logoURL, logoPath, token, nil); err != nil {
					log.Printf("plugin %q: download logo: %v", pluginID, err)
				}
			}
		}

		// 8. Persist metadata last, only after binary + bundle are downloaded,
		// verified, and extracted. Writing it any earlier risks clobbering a
		// previously valid installation's metadata with info for a version
		// that never finished downloading, breaking restoreInstalledPlugins.
		finalMetadata := dto.PluginMetadata{
			Manifest:    *manifest,
			ReleaseTag:  tag,
			InstalledAt: time.Now().Format(time.RFC3339),
		}
		finalMetadataJSON, err := json.Marshal(finalMetadata)
		if err != nil {
			loader.SetStatusWithError(dto.PluginStatusCrashed, fmt.Sprintf("marshal metadata: %v", err))
			return
		}
		if err := plugin.WriteMetadataAtomically(metadataPath, finalMetadataJSON); err != nil {
			loader.SetStatusWithError(dto.PluginStatusCrashed, fmt.Sprintf("persist metadata: %v", err))
			return
		}

		// Download/verify/extract/metadata all succeeded, so the old assets'
		// backup is no longer needed and will be discarded by the deferred cleanup.
		installSucceeded = true

		// 9. Conditionally launch the plugin if an active context is present
		// If no active context, defer launch to helmPluginClient lazy-launch logic.
		// This allows install to succeed as a metadata-level operation.
		a.mu.RLock()
		activeContextName := a.activeContext
		a.mu.RUnlock()

		if activeContextName != "" {
			// Active context exists; resolve kubeconfig and launch now.
			// A failure here is not install failure — the binary and metadata are
			// already valid, so fall through to READY and let helmPluginClient's
			// lazy-launch retry once a valid context is available.
			kubeconfigPath, err := a.GetContextKubeconfigPath(activeContextName)
			if err == nil {
				_ = loader.Launch(ctx, kubeconfigPath)
			}
		}
		// If no active context, skip launch; helmPluginClient will do lazy launch on first use

		loader.SetStatus(dto.PluginStatusReady)
	}()

	return nil
}

// RemovePlugin uninstalls an installed plugin by ID, removing its binary,
// bundle, metadata, and lock file from disk. The plugin process is gracefully
// shut down if running. Returns an error if the plugin is not installed or
// is currently installing. If the plugin is on disk but not in the loaders map
// (orphaned), it is still removed.
func (a *App) RemovePlugin(pluginID string) error {
	// Validate pluginID to prevent path traversal
	if !plugin.ValidPluginID(pluginID) {
		return fmt.Errorf("invalid plugin ID: %q", pluginID)
	}

	pluginDir := a.pluginInstallDir(pluginID)

	// Acquire lock to check installation status and mark removal in progress
	// atomically, so no InstallPlugin call can slip in between the check and
	// the removingPluginIDs flag being set.
	a.pluginsMu.Lock()

	// Check if an installation is already in progress for this plugin
	if a.pluginLoaders[pluginID] != nil && a.pluginLoaders[pluginID].Status() == dto.PluginStatusInstalling {
		a.pluginsMu.Unlock()
		return fmt.Errorf("cannot remove plugin %q while installation is in progress", pluginID)
	}

	// Get the loader if it exists (may be nil for orphaned directories)
	loader, loaderExists := a.pluginLoaders[pluginID]

	// Mark removal in progress before releasing the lock so a concurrent
	// InstallPlugin call sees it immediately (see the removingPluginIDs check
	// added to InstallPlugin above).
	a.removingPluginIDs[pluginID] = true
	a.pluginsMu.Unlock()

	// Check if plugin exists (either in map or on disk)
	dirExists := false
	if _, err := os.Stat(pluginDir); err == nil {
		dirExists = true
	}

	if !loaderExists && !dirExists {
		// Not in map and not on disk — truly not installed
		a.pluginsMu.Lock()
		delete(a.removingPluginIDs, pluginID)
		a.pluginsMu.Unlock()
		return fmt.Errorf("plugin %q is not installed", pluginID)
	}

	// Shutdown the plugin if a loader exists
	if loaderExists && loader != nil {
		if err := loader.Shutdown(); err != nil {
			// Log but don't fail — we still want to delete the files
			fmt.Printf("plugin shutdown warning: %v\n", err)
		}
	}

	// Delete the plugin directory recursively (works whether loader exists or not)
	var removeErr error
	if err := os.RemoveAll(pluginDir); err != nil {
		// Check if the directory still exists; if not, treat as success
		if _, statErr := os.Stat(pluginDir); os.IsNotExist(statErr) {
			removeErr = nil
		} else {
			removeErr = fmt.Errorf("removing plugin directory %q: %w", pluginDir, err)
		}
	}

	// Clean up tracking
	a.pluginsMu.Lock()
	delete(a.removingPluginIDs, pluginID)
	// Only delete the loader if it exists
	if loaderExists {
		delete(a.pluginLoaders, pluginID)
	}
	a.pluginsMu.Unlock()

	return removeErr
}

// GetPluginsFromMarketplace fetches manifests from the default marketplace and all
// user-configured repositories concurrently. It returns a merged slice of manifests
// with SourceURL set appropriately (empty string for default, repository URL for user-added).
// Errors from individual sources are aggregated in the Errors map, keyed by source identifier.
func (a *App) GetPluginsFromMarketplace() *dto.MarketplaceResult {
	a.mu.RLock()
	repositories := make([]config.MarketplaceRepository, len(a.settings.MarketplaceRepositories))
	copy(repositories, a.settings.MarketplaceRepositories)
	a.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Prepare list of sources to fetch: default (baseURL="") + user repos.
	// Settings.AccessToken is reserved for the app's own update-check downloads
	// and must never be used as a fallback here: each marketplace repository
	// uses its own configured AccessToken only (empty for the public default
	// marketplace, or if the user left a private repo's token unset).
	type source struct {
		sourceURL string // empty for default, repo URL for user-added
		token     string
	}

	// Build list of sources: default (baseURL="") + non-disabled user repos.
	sources := []source{
		{
			sourceURL: "",
			token:     "",
		},
	}
	// User-added repository sources (skip disabled ones)
	for _, repo := range repositories {
		if repo.Disabled {
			continue
		}
		sources = append(sources, source{
			sourceURL: repo.URL,
			token:     repo.AccessToken,
		})
	}

	// Fetch all sources concurrently via channels
	type fetchResult struct {
		manifests []*dto.Manifest
		sourceURL string
		errors    map[string]string // keyed sourceLabel+":release" or sourceLabel+":"+pluginID
	}

	resultsChan := make(chan fetchResult, len(sources))
	for _, src := range sources {
		go func(src source) {
			sourceLabel := src.sourceURL
			if sourceLabel == "" {
				sourceLabel = "default"
			}

			assets, _, err := plugin.FetchLatestRelease(ctx, src.sourceURL, src.token)
			if err != nil {
				resultsChan <- fetchResult{
					manifests: nil,
					sourceURL: src.sourceURL,
					errors:    map[string]string{sourceLabel + ":release": err.Error()},
				}
				return
			}

			pluginIDs := plugin.DiscoverPluginIDs(assets)
			manifests := make([]*dto.Manifest, 0, len(pluginIDs))
			srcErrors := make(map[string]string)

			for _, pluginID := range pluginIDs {
				manifest, err := plugin.FetchManifest(ctx, assets, pluginID, src.token)
				if err != nil {
					log.Printf("plugin marketplace: %s from %q: fetch manifest failed: %v", pluginID, src.sourceURL, err)
					srcErrors[sourceLabel+":"+pluginID] = err.Error()
					continue
				}
				manifest.SourceURL = src.sourceURL
				manifests = append(manifests, manifest)
			}

			resultsChan <- fetchResult{
				manifests: manifests,
				sourceURL: src.sourceURL,
				errors:    srcErrors,
			}
		}(src)
	}

	// Collect results
	allManifests := make([]*dto.Manifest, 0)
	errors := make(map[string]string)

	for range sources {
		result := <-resultsChan
		maps.Copy(errors, result.errors)
		allManifests = append(allManifests, result.manifests...)
	}

	return &dto.MarketplaceResult{
		Manifests: allManifests,
		Errors:    errors,
	}
}

// pluginClient returns a ready gRPC client for the named plugin, launching it
// lazily and syncing its active cluster context. Generic across every
// plugin — this file must never import a concrete plugin's own package.
func (a *App) pluginClient(pluginID string) (pb.PluginClient, error) {
	a.pluginsMu.RLock()
	loader, ok := a.pluginLoaders[pluginID]
	a.pluginsMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("plugin %q not installed", pluginID)
	}
	if status := loader.Status(); status != dto.PluginStatusReady {
		return nil, fmt.Errorf("plugin %q not ready (status: %s)", pluginID, status)
	}

	a.mu.RLock()
	activeContextName := a.activeContext
	a.mu.RUnlock()
	if activeContextName == "" {
		return nil, fmt.Errorf("plugin %q requires a connected cluster context", pluginID)
	}
	kubeconfigPath, err := a.GetContextKubeconfigPath(activeContextName)
	if err != nil {
		return nil, fmt.Errorf("resolve kubeconfig: %w", err)
	}

	client := loader.GetClient()
	if client == nil {
		if err := loader.Launch(context.Background(), kubeconfigPath); err != nil {
			return nil, fmt.Errorf("launch plugin %q: %w", pluginID, err)
		}
		client = loader.GetClient()
		if client == nil {
			return nil, fmt.Errorf("plugin %q client unavailable", pluginID)
		}
	}

	if _, err := client.SetClusterContext(context.Background(), &pb.SetClusterContextRequest{
		ContextName:    activeContextName,
		KubeconfigPath: kubeconfigPath,
	}); err != nil {
		return nil, fmt.Errorf("sync cluster context: %w", err)
	}

	return client, nil
}

// InvokePlugin is the single generic Wails-bound entry point the frontend
// uses to call into any installed plugin. pluginID selects the plugin (e.g.
// "helm"); method and payloadJSON/the return value are opaque strings whose
// shape only the plugin's own frontend bundle and backend agree on — this
// method must never branch on pluginID or method.
func (a *App) InvokePlugin(pluginID, method, payloadJSON string) (string, error) {
	client, err := a.pluginClient(pluginID)
	if err != nil {
		return "", err
	}
	resp, err := client.Invoke(context.Background(), &pb.InvokeRequest{
		Method:      method,
		PayloadJson: payloadJSON,
	})
	if err != nil {
		return "", err
	}
	if resp.Error != "" {
		return "", fmt.Errorf("%s", resp.Error)
	}
	return resp.PayloadJson, nil
}
