package plugin

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/litelensapp/litelens/internal/config"
	"github.com/litelensapp/litelens/packages/core/kube/dto"
)

const pluginInstallerUserAgent = "litelens-plugin-installer/1.0"

// manifestIndexAssetName is the repo-wide release asset that acts as the
// single source of truth for every plugin published in a release: it embeds
// each plugin's complete manifest directly, so no separate per-plugin
// manifest asset is needed (or published by real litelens-plugins releases).
const manifestIndexAssetName = "manifest.json"

// newGitHubAPIRequest builds a GET request with the headers GitHub's REST API
// expects, plus a Bearer token when the target repo is private.
func newGitHubAPIRequest(ctx context.Context, url, token string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", pluginInstallerUserAgent)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req, nil
}

// newAssetDownloadRequest builds a GET request for a release asset's
// browser_download_url. Private-repo assets require the same Bearer auth as
// the API endpoints (see internal/app/updater.go's performWindowsUpdate).
func newAssetDownloadRequest(ctx context.Context, url, token string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", pluginInstallerUserAgent)
	req.Header.Set("Accept", "application/octet-stream")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req, nil
}

// checkGitHubAPIResponse turns a non-200 GitHub API response into an error,
// giving a specific, actionable message when the failure is due to the
// unauthenticated rate limit (60 requests/hour per IP) rather than a generic
// "status 403" — that limit is shared by anything else on the same network
// calling GitHub's API, so it's routinely hit without the plugin fetch itself
// being at fault.
func checkGitHubAPIResponse(resp *http.Response) error {
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	if (resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests) &&
		resp.Header.Get("X-RateLimit-Remaining") == "0" {
		resetAt := "an unknown time"
		if resetUnix, err := strconv.ParseInt(resp.Header.Get("X-RateLimit-Reset"), 10, 64); err == nil {
			resetAt = time.Unix(resetUnix, 0).Local().Format("15:04:05 MST")
		}
		return fmt.Errorf("github API rate limit exceeded for this network (resets at %s); this limit is shared by anything else on your network calling GitHub's API and is unrelated to plugin availability", resetAt)
	}
	return fmt.Errorf("github API returned status %d", resp.StatusCode)
}

// FetchLatestRelease fetches the latest release from GitHub API.
// If baseURL is empty, uses config.GetMarketplaceBaseURL() (env-var/default behavior).
// If baseURL is non-empty, uses it directly (expected to be in the form: https://api.github.com/repos/owner/repo/releases).
// Token is sent as a Bearer header when non-empty, so private repos work identically.
// private indicates whether to use the authenticated API asset endpoint (asset.URL) for private repos
// or the public browser_download_url (asset.BrowserDownloadURL) for public repos.
// Returns a map of asset name -> download URL and the release tag name.
func FetchLatestRelease(ctx context.Context, baseURL, token string, private bool) (assets map[string]string, tag string, err error) {
	if baseURL == "" {
		baseURL = config.GetMarketplaceBaseURL()
	}
	url := baseURL + "/latest"

	req, err := newGitHubAPIRequest(ctx, url, token)
	if err != nil {
		return nil, "", fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("fetching release: %w", err)
	}
	defer resp.Body.Close()

	if err := checkGitHubAPIResponse(resp); err != nil {
		return nil, "", err
	}

	var release dto.GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, "", fmt.Errorf("parsing release JSON: %w", err)
	}

	assets = make(map[string]string)
	for _, asset := range release.Assets {
		if private {
			assets[asset.Name] = asset.URL
		} else {
			assets[asset.Name] = asset.BrowserDownloadURL
		}
	}

	return assets, release.TagName, nil
}

// FetchRelease fetches a specific release by tag from GitHub API.
// If baseURL is empty, uses config.GetMarketplaceBaseURL() (env-var/default behavior).
// If baseURL is non-empty, uses it directly (expected to be in the form: https://api.github.com/repos/owner/repo/releases).
// If tag is empty, behaves identically to FetchLatestRelease.
// private indicates whether to use the authenticated API asset endpoint (asset.URL) for private repos
// or the public browser_download_url (asset.BrowserDownloadURL) for public repos.
// Returns a map of asset name -> download URL and the resolved tag name.
func FetchRelease(ctx context.Context, baseURL, token, tag string, private bool) (assets map[string]string, resolvedTag string, err error) {
	if tag == "" {
		return FetchLatestRelease(ctx, baseURL, token, private)
	}

	if baseURL == "" {
		baseURL = config.GetMarketplaceBaseURL()
	}
	url := baseURL + "/tags/" + neturl.PathEscape(tag)

	req, err := newGitHubAPIRequest(ctx, url, token)
	if err != nil {
		return nil, "", fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("fetching release: %w", err)
	}
	defer resp.Body.Close()

	if err := checkGitHubAPIResponse(resp); err != nil {
		return nil, "", err
	}

	var release dto.GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, "", fmt.Errorf("parsing release JSON: %w", err)
	}

	assets = make(map[string]string)
	for _, asset := range release.Assets {
		if private {
			assets[asset.Name] = asset.URL
		} else {
			assets[asset.Name] = asset.BrowserDownloadURL
		}
	}

	return assets, release.TagName, nil
}

// FetchManifestIndex fetches and parses the repo-wide manifest.json index
// asset from a release — the single source of truth for every plugin the
// release publishes. Returns an error if the asset is missing or fails to
// fetch/parse.
func FetchManifestIndex(ctx context.Context, assets map[string]string, token string) (*dto.ManifestIndex, error) {
	url, ok := assets[manifestIndexAssetName]
	if !ok {
		return nil, fmt.Errorf("%q not found in release", manifestIndexAssetName)
	}

	req, err := newAssetDownloadRequest(ctx, url, token)
	if err != nil {
		return nil, fmt.Errorf("creating manifest index request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching manifest index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("manifest index fetch returned status %d", resp.StatusCode)
	}

	var index dto.ManifestIndex
	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		return nil, fmt.Errorf("parsing manifest index JSON: %w", err)
	}

	return &index, nil
}

// DiscoverPluginIDs derives the set of plugin IDs published in a release from
// the manifest index's embedded manifests, so the marketplace never needs a
// hardcoded list of known plugins. Returned in sorted order for determinism.
func DiscoverPluginIDs(index *dto.ManifestIndex) []string {
	ids := make([]string, 0, len(index.Plugins))
	for _, entry := range index.Plugins {
		if entry.ID != "" && ValidPluginID(entry.ID) {
			ids = append(ids, entry.ID)
		}
	}
	slices.Sort(ids)
	return ids
}

// FetchManifest returns the plugin's manifest from the manifest index — the
// index embeds each plugin's complete manifest directly, so no further asset
// fetch is needed.
func FetchManifest(pluginID string, index *dto.ManifestIndex) (*dto.Manifest, error) {
	for i := range index.Plugins {
		if index.Plugins[i].ID == pluginID {
			manifest := index.Plugins[i]
			manifest.Version = NormalizeVersion(manifest.Version)
			return &manifest, nil
		}
	}
	return nil, fmt.Errorf("plugin %q not found in manifest index", pluginID)
}

// NormalizeVersion strips a leading "v"/"V" so versions from different sources
// (manifest.json's declared version, a GitHub release tag like "v0.15.4") can
// be compared and displayed consistently as bare "x.y.z" semver.
func NormalizeVersion(version string) string {
	if len(version) > 0 && (version[0] == 'v' || version[0] == 'V') {
		return version[1:]
	}
	return version
}

// ResolveAssetNames returns the binary and bundle asset names for the given platform.
// bundleAsset is now a tar.gz archive containing the complete dist/ directory
// (index.js + chunk-*.js files with content-hashed names).
func ResolveAssetNames(pluginID, goos, goarch string) (binaryAsset, bundleAsset string) {
	binaryAsset = fmt.Sprintf("litelens-plugin-%s-%s-%s", pluginID, goos, goarch)
	if goos == "windows" {
		binaryAsset += ".exe"
	}
	bundleAsset = fmt.Sprintf("litelens-plugin-%s-frontend.tar.gz", pluginID)
	return binaryAsset, bundleAsset
}

// ResolveLogoAssetName returns the GitHub release asset name for a plugin's
// logo file, derived from the local filename manifest.Assets.Logo declares
// (e.g. "helm.svg" -> "litelens-plugin-helm-logo.svg"). Mirrors the same
// litelens-plugin-<id>-<kind> naming convention as the binary/bundle/manifest
// assets, so the installer can look it up in the assets map FetchRelease returns.
func ResolveLogoAssetName(pluginID, logoFileName string) string {
	return fmt.Sprintf("litelens-plugin-%s-logo%s", pluginID, filepath.Ext(logoFileName))
}

// IsPlatformSupported checks if the manifest supports the given platform
func IsPlatformSupported(m *dto.Manifest, goos, goarch string) bool {
	if m.OS == nil {
		return false
	}
	archs, ok := m.OS[goos]
	if !ok {
		return false
	}
	return slices.Contains(archs, goarch)
}

// IsHostVersionCompatible checks if hostVersion falls within min and max version constraints.
// Development builds (hostVersion == config.Dev, the same value the frontend's About dialog
// displays via GetVersion()) always report compatible, since the version gate only makes
// sense for semver-tagged releases.
func IsHostVersionCompatible(hostVersion, minVersion, maxVersion string) (bool, error) {
	if hostVersion == config.Dev {
		return true, nil
	}

	hostVer, err := semver.NewVersion(hostVersion)
	if err != nil {
		return false, fmt.Errorf("parsing host version %q: %w", hostVersion, err)
	}

	minVer, err := semver.NewVersion(minVersion)
	if err != nil {
		return false, fmt.Errorf("parsing minimum version %q: %w", minVersion, err)
	}

	maxVer, err := semver.NewVersion(maxVersion)
	if err != nil {
		return false, fmt.Errorf("parsing maximum version %q: %w", maxVersion, err)
	}

	// Check if hostVer is within [minVer, maxVer]
	if hostVer.LessThan(minVer) || hostVer.GreaterThan(maxVer) {
		return false, nil
	}

	return true, nil
}

// DownloadToFile downloads a file from the given URL to destPath.
// Creates parent directories if needed and atomically writes via temp file.
// token is sent as a Bearer header (required for private-repo release assets).
// onProgress is an optional callback that receives progress updates as a percentage (0-100).
// If onProgress is nil or resp.ContentLength is unknown (<=0), no progress callbacks are made.
func DownloadToFile(ctx context.Context, url, destPath, token string, onProgress func(pct int)) error {
	// Ensure parent directory exists
	parentDir := filepath.Dir(destPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("creating directory %q: %w", parentDir, err)
	}

	// Download to a temporary file in the same directory to ensure atomic move
	tempFile := destPath + ".tmp"

	// Remove any stale temp file
	_ = os.Remove(tempFile)

	req, err := newAssetDownloadRequest(ctx, url, token)
	if err != nil {
		return fmt.Errorf("creating download request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	// Write to temp file
	f, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("creating temp file %q: %w", tempFile, err)
	}
	defer f.Close()

	// Wrap resp.Body with progress tracking if ContentLength is known and onProgress is provided
	var body io.Reader = resp.Body
	if onProgress != nil && resp.ContentLength > 0 {
		body = &progressReader{
			reader:        resp.Body,
			contentLength: resp.ContentLength,
			onProgress:    onProgress,
		}
	}

	if _, err := io.Copy(f, body); err != nil {
		_ = os.Remove(tempFile)
		return fmt.Errorf("writing to temp file: %w", err)
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(tempFile)
		return fmt.Errorf("closing temp file: %w", err)
	}

	// Atomically move temp file to final location
	if err := os.Rename(tempFile, destPath); err != nil {
		_ = os.Remove(tempFile)
		return fmt.Errorf("moving temp file to %q: %w", destPath, err)
	}

	return nil
}

// AssetBackup tracks an installed plugin's existing binary, dist directory,
// and metadata file after they've been moved aside by BackupPluginAssets, so
// a failed re-download can be rolled back via Restore, or the backup can be
// discarded via Discard once the new version installs successfully.
type AssetBackup struct {
	dir          string
	binaryPath   string
	distDir      string
	metadataPath string
	hadBinary    bool
	hadDist      bool
	hadMetadata  bool
}

// BackupPluginAssets moves a plugin's existing binary, dist directory, and
// metadata file into a temporary ".backup" directory before a new version is
// downloaded. Assets that don't exist yet (fresh install) are skipped, so the
// returned backup is a no-op for Restore/Discard.
func BackupPluginAssets(pluginDir, binaryPath, distDir, metadataPath string) (*AssetBackup, error) {
	backupDir := filepath.Join(pluginDir, ".backup")
	_ = os.RemoveAll(backupDir)

	b := &AssetBackup{
		dir:          backupDir,
		binaryPath:   binaryPath,
		distDir:      distDir,
		metadataPath: metadataPath,
	}

	moveAside := func(srcPath, backupName string) (moved bool, err error) {
		if _, statErr := os.Lstat(srcPath); statErr != nil {
			return false, nil
		}
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			return false, fmt.Errorf("creating backup directory: %w", err)
		}
		if err := os.Rename(srcPath, filepath.Join(backupDir, backupName)); err != nil {
			return false, fmt.Errorf("backing up %q: %w", srcPath, err)
		}
		return true, nil
	}

	var err error
	if b.hadBinary, err = moveAside(binaryPath, filepath.Base(binaryPath)); err != nil {
		return nil, err
	}
	if b.hadDist, err = moveAside(distDir, "dist"); err != nil {
		return nil, err
	}
	if b.hadMetadata, err = moveAside(metadataPath, filepath.Base(metadataPath)); err != nil {
		return nil, err
	}

	return b, nil
}

// Restore moves backed-up assets back to their original locations, undoing
// BackupPluginAssets. Any partially-downloaded new assets at the destination
// are removed first so the restored (old) assets always win.
func (b *AssetBackup) Restore() error {
	moveBack := func(hadIt bool, backupName, destPath string, removeDest func(string) error) error {
		if !hadIt {
			return nil
		}
		_ = removeDest(destPath)
		if err := os.Rename(filepath.Join(b.dir, backupName), destPath); err != nil {
			return fmt.Errorf("restoring %q: %w", destPath, err)
		}
		return nil
	}

	if err := moveBack(b.hadBinary, filepath.Base(b.binaryPath), b.binaryPath, os.Remove); err != nil {
		return err
	}
	if err := moveBack(b.hadDist, "dist", b.distDir, os.RemoveAll); err != nil {
		return err
	}
	if err := moveBack(b.hadMetadata, filepath.Base(b.metadataPath), b.metadataPath, os.Remove); err != nil {
		return err
	}

	return os.RemoveAll(b.dir)
}

// Discard permanently deletes the backed-up assets after a successful
// install, since the previous version's files are no longer needed.
func (b *AssetBackup) Discard() error {
	if !b.hadBinary && !b.hadDist && !b.hadMetadata {
		return nil
	}
	return os.RemoveAll(b.dir)
}

// progressReader wraps an io.Reader and tracks progress with callbacks.
// It only calls onProgress when the percentage actually changes to avoid flooding callbacks.
type progressReader struct {
	reader        io.Reader
	contentLength int64
	bytesRead     int64
	lastReported  int
	onProgress    func(pct int)
}

func (pr *progressReader) Read(p []byte) (n int, err error) {
	n, err = pr.reader.Read(p)
	pr.bytesRead += int64(n)

	// Only invoke onProgress if percentage changes
	if pr.contentLength > 0 && pr.onProgress != nil {
		pct := int(pr.bytesRead * 100 / pr.contentLength)
		if pct > pr.lastReported {
			pr.lastReported = pct
			pr.onProgress(pct)
		}
	}

	return n, err
}

// VerifySHA256 verifies that a file's SHA256 hash matches the expected hex string (case-insensitive)
func VerifySHA256(path, expectedHex string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening file %q: %w", path, err)
	}
	defer f.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return fmt.Errorf("hashing file: %w", err)
	}

	actualHex := fmt.Sprintf("%x", hash.Sum(nil))
	if strings.EqualFold(actualHex, expectedHex) {
		return nil
	}

	return fmt.Errorf("SHA256 mismatch: expected %s, got %s", expectedHex, actualHex)
}

// ExtractTarGz extracts a gzip-compressed tar archive to destDir.
// Only regular files are extracted; directories, symlinks, and hard links are rejected
// for security. Zip-slip protection is mandatory: each entry's target path is verified
// to be within destDir via filepath.Rel — if relative path starts with "..", extraction fails.
func ExtractTarGz(archivePath, destDir string) error {
	// Open and decompress
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("opening archive %q: %w", archivePath, err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("decompressing gzip: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			// End of tar stream, normal completion
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}

		// Only extract regular files; reject everything else
		if hdr.Typeflag != tar.TypeReg {
			if hdr.Typeflag == tar.TypeSymlink || hdr.Typeflag == tar.TypeLink {
				return fmt.Errorf("rejecting symlink/hardlink in archive: %q", hdr.Name)
			}
			// Silently skip directories and other types
			continue
		}

		// Zip-slip protection: resolve target path and verify it's within destDir
		// BEFORE creating any directories or writing any files.
		targetPath := filepath.Join(destDir, hdr.Name)

		absDestDir, err := filepath.Abs(destDir)
		if err != nil {
			return fmt.Errorf("resolving dest directory: %w", err)
		}

		absTargetPath, err := filepath.Abs(targetPath)
		if err != nil {
			return fmt.Errorf("resolving target path for %q: %w", hdr.Name, err)
		}

		// Check if resolved target is within destDir
		relPath, err := filepath.Rel(absDestDir, absTargetPath)
		if err != nil || strings.HasPrefix(relPath, "..") {
			return fmt.Errorf("path traversal attack detected: %q would escape %q", hdr.Name, destDir)
		}

		// Only now that the path is confirmed safe, ensure parent directories exist
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("creating directory for %q: %w", targetPath, err)
		}

		// Extract file with 0644 permissions (non-executable)
		f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("creating file %q: %w", targetPath, err)
		}

		if _, err := io.Copy(f, tr); err != nil {
			f.Close()
			return fmt.Errorf("writing file %q: %w", targetPath, err)
		}

		if err := f.Close(); err != nil {
			return fmt.Errorf("closing file %q: %w", targetPath, err)
		}
	}

	return nil
}
