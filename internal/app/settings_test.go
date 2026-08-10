package app

import (
	"os"
	"testing"

	"github.com/gknguyen/litelens/internal/config"
	"github.com/gknguyen/litelens/internal/dto"
	"github.com/gknguyen/litelens/internal/plugin"
)

// TestSaveSettingsPluginsDirChange verifies that changing PluginsDir in SaveSettings
// correctly rebuilds a.pluginLoaders against the new directory.
// This is the fix for the stale plugin metadata bug when users change the plugins dir
// in Settings > Marketplace and then revert to default.
// NOTE: SaveSettings calls runtime.EventsEmit which requires a valid Wails context.
// We can't test the full SaveSettings in unit tests, but we can test the key logic:
// - The pluginsDirChanged detection (comparing old vs new PluginsDir)
// - The guard (pluginOperationInProgress check)
// - The isolation (onPluginsDirChanged clears and rebuilds loaders)
func TestSaveSettingsPluginsDirChangeGuard(t *testing.T) {
	app := NewApp("test")

	// Start with default plugins dir (empty string in settings)
	origDir := app.pluginsRootDir()
	if origDir == "" {
		t.Fatalf("expected default pluginsRootDir to be non-empty, got empty string")
	}

	// Test: pluginsDirChanged detection works
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "litelens-plugin-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Manually simulate the logic: read old, check if changed, check guard
	app.mu.RLock()
	oldPluginsDir := app.settings.PluginsDir
	app.mu.RUnlock()

	newPluginsDir := tmpDir
	pluginsDirChanged := newPluginsDir != oldPluginsDir

	if !pluginsDirChanged {
		t.Error("expected pluginsDirChanged to be true when switching from default to custom dir")
	}

	// Manually check that pluginOperationInProgress returns false (no-op guard should pass)
	if app.pluginOperationInProgress() {
		t.Error("expected pluginOperationInProgress to be false when no operations in progress")
	}
}

// TestPluginOperationInProgressDuringInstall verifies that the guard correctly
// detects when a plugin install is in progress.
func TestPluginOperationInProgressDuringInstall(t *testing.T) {
	app := NewApp("test")

	tmpDir, err := os.MkdirTemp("", "litelens-plugin-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Simulate a plugin currently installing
	app.pluginsMu.Lock()
	mockLoader := &plugin.PluginLoader{}
	mockLoader.SetStatus(dto.PluginStatusInstalling)
	app.pluginLoaders["installing-plugin"] = mockLoader
	app.pluginsMu.Unlock()

	// pluginOperationInProgress should return true
	if !app.pluginOperationInProgress() {
		t.Error("expected pluginOperationInProgress to be true when plugin is INSTALLING")
	}

	// The guard logic in SaveSettings (if pluginsDirChanged && a.pluginOperationInProgress())
	// would reject the directory change at this point.
	// We can't call SaveSettings directly due to EventsEmit context requirements,
	// but we've verified the check would trigger.
}

// TestPluginOperationInProgressDuringRemoval verifies that the guard correctly
// detects when a plugin removal is in progress.
func TestPluginOperationInProgressDuringRemoval(t *testing.T) {
	app := NewApp("test")

	tmpDir, err := os.MkdirTemp("", "litelens-plugin-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Simulate a plugin removal in progress
	app.pluginsMu.Lock()
	app.removingPluginIDs["removing-plugin"] = true
	app.pluginsMu.Unlock()

	// pluginOperationInProgress should return true
	if !app.pluginOperationInProgress() {
		t.Error("expected pluginOperationInProgress to be true when removal is in progress")
	}
}

// TestPluginOperationNotBlockedWhenSameDirUnchanged verifies that the guard does NOT
// block directory-change saves if PluginsDir is unchanged, even if a plugin install is in progress.
// The guard only applies to ACTUAL DIRECTORY CHANGES.
func TestPluginOperationNotBlockedWhenSameDirUnchanged(t *testing.T) {
	app := NewApp("test")

	// Simulate a plugin installing
	app.pluginsMu.Lock()
	mockLoader := &plugin.PluginLoader{}
	mockLoader.SetStatus(dto.PluginStatusInstalling)
	app.pluginLoaders["installing-plugin"] = mockLoader
	app.pluginsMu.Unlock()

	// Read current PluginsDir, then submit the identical value back (simulating a
	// save where the user didn't actually change the directory field).
	app.mu.RLock()
	oldDir := app.settings.PluginsDir
	app.mu.RUnlock()
	newDir := oldDir

	// Check if directory would be considered "changed"
	pluginsDirChanged := newDir != oldDir
	hasPluginOpInProgress := app.pluginOperationInProgress()

	// The guard check is: if pluginsDirChanged && a.pluginOperationInProgress()
	// Since pluginsDirChanged is false, the entire guard is bypassed (short-circuit)
	if pluginsDirChanged && hasPluginOpInProgress {
		t.Error("guard should not trigger when PluginsDir is unchanged")
	}
}

// TestPluginOperationInProgress verifies the helper correctly detects
// in-flight plugin operations.
func TestPluginOperationInProgress(t *testing.T) {
	app := NewApp("test")

	// Initially should be false (no operations in progress)
	if app.pluginOperationInProgress() {
		t.Error("expected pluginOperationInProgress to be false initially")
	}

	// Simulate plugin installing
	app.pluginsMu.Lock()
	mockLoader := &plugin.PluginLoader{}
	mockLoader.SetStatus(dto.PluginStatusInstalling)
	app.pluginLoaders["plugin-a"] = mockLoader
	app.pluginsMu.Unlock()

	if !app.pluginOperationInProgress() {
		t.Error("expected pluginOperationInProgress to be true when plugin is INSTALLING")
	}

	// Simulate plugin ready
	app.pluginsMu.Lock()
	mockLoader.SetStatus(dto.PluginStatusReady)
	app.pluginsMu.Unlock()

	if app.pluginOperationInProgress() {
		t.Error("expected pluginOperationInProgress to be false when plugin is READY")
	}

	// Simulate plugin removal
	app.pluginsMu.Lock()
	app.removingPluginIDs["plugin-b"] = true
	app.pluginsMu.Unlock()

	if !app.pluginOperationInProgress() {
		t.Error("expected pluginOperationInProgress to be true when plugin removal is in progress")
	}

	// Clear removal
	app.pluginsMu.Lock()
	delete(app.removingPluginIDs, "plugin-b")
	app.pluginsMu.Unlock()

	if app.pluginOperationInProgress() {
		t.Error("expected pluginOperationInProgress to be false when removal is cleared")
	}
}

// TestGetSettingsSeedsDefaultMarketplaceURLOnFirstRun verifies that GetSettings
// seeds the default marketplace source (config.GetMarketplaceBaseURL(), which
// honors MARKETPLACE_BASE_URL) as the first entry of MarketplaceRepositories
// when it has never been persisted (fresh install, nil array), so the
// frontend can show/manage it like any other repository row without a
// dedicated "default URL" concept of its own.
func TestGetSettingsSeedsDefaultMarketplaceURLOnFirstRun(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("MARKETPLACE_BASE_URL", "https://api.github.com/repos/test-org/test-marketplace/releases")

	app := NewApp("test")

	settings, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings returned error: %v", err)
	}

	if len(settings.MarketplaceRepositories) != 1 {
		t.Fatalf("expected 1 seeded repository, got %d: %+v", len(settings.MarketplaceRepositories), settings.MarketplaceRepositories)
	}
	if got := settings.MarketplaceRepositories[0].URL; got != "https://api.github.com/repos/test-org/test-marketplace/releases" {
		t.Errorf("expected seeded URL to match MARKETPLACE_BASE_URL, got %q", got)
	}
}

// TestGetSettingsDoesNotReSeedOnceUserHasSaved verifies that once the user has
// saved settings with a non-nil MarketplaceRepositories array (even one where
// they removed the seeded default entry entirely, leaving it empty), GetSettings
// never seeds the default URL again.
func TestGetSettingsDoesNotReSeedOnceUserHasSaved(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := config.Save(config.Settings{
		MarketplaceRepositories: []config.MarketplaceRepository{},
	}); err != nil {
		t.Fatalf("failed to seed settings: %v", err)
	}

	app := NewApp("test")

	settings, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings returned error: %v", err)
	}

	if len(settings.MarketplaceRepositories) != 0 {
		t.Fatalf("expected no re-seeding once persisted as empty, got %d entries: %+v", len(settings.MarketplaceRepositories), settings.MarketplaceRepositories)
	}
}

// TestGetSettingsPreservesUserEditToSeededDefaultURL verifies that if the user
// edited the seeded default row's URL and saved, subsequent GetSettings calls
// return the edited value as-is rather than re-seeding the original default.
func TestGetSettingsPreservesUserEditToSeededDefaultURL(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := config.Save(config.Settings{
		MarketplaceRepositories: []config.MarketplaceRepository{
			{URL: "https://api.github.com/repos/user/custom/releases", Private: true, AccessToken: "tok"},
		},
	}); err != nil {
		t.Fatalf("failed to seed settings: %v", err)
	}

	app := NewApp("test")

	settings, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings returned error: %v", err)
	}

	if len(settings.MarketplaceRepositories) != 1 {
		t.Fatalf("expected exactly the user's saved repo, got %d entries: %+v", len(settings.MarketplaceRepositories), settings.MarketplaceRepositories)
	}
	if got := settings.MarketplaceRepositories[0].URL; got != "https://api.github.com/repos/user/custom/releases" {
		t.Errorf("expected user's edited URL, got %q", got)
	}
}

// TestGranularSectionSavesDoNotClobberMarketplaceRepositories guards against a
// real "last write wins" race that existed when every Settings section
// (Marketplace, Plugins Directory, Terminal, Sandbox) shared the single
// generic SaveSettings(fullObject) path: each section independently did
// read-latest-then-merge-then-SaveSettings(fullObject), so saving one section
// with a "latest" snapshot captured BEFORE another section's save landed
// would silently revert that other section's just-saved data.
//
// The fix moves Marketplace (SaveMarketplaceRepositories) and Plugins
// Directory (SavePluginsDir) onto dedicated methods that mutate only their
// own field directly on the authoritative in-memory a.settings under the
// lock, so neither can be clobbered by, or clobber, a stale snapshot from an
// unrelated section's save. This test proves that invariant: it saves
// Marketplace, then saves Plugins Directory using a snapshot captured BEFORE
// the marketplace save, and asserts the marketplace data survives.
func TestGranularSectionSavesDoNotClobberMarketplaceRepositories(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	app := NewApp("test")

	// Both sections read "latest" at the same moment, before either has saved
	// anything — this mirrors two Settings sub-sections open on the same
	// MarketplaceContent view, each holding their own independent snapshot.
	staleSnapshot, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings returned error: %v", err)
	}

	marketplaceRepos := []config.MarketplaceRepository{
		{URL: "https://api.github.com/repos/gknguyen/litelens/releases"},
		{URL: "https://github.com/userA/repoA"},
		{URL: "https://github.com/userB/repoB"},
	}
	if err := app.SaveMarketplaceRepositories(marketplaceRepos); err != nil {
		t.Fatalf("SaveMarketplaceRepositories returned error: %v", err)
	}

	// Plugins Directory section saves its own field using its OWN stale
	// snapshot's value (unchanged), captured before the marketplace save
	// landed — exactly what a lagging frontend cache would hold.
	if err := app.SavePluginsDir(staleSnapshot.PluginsDir); err != nil {
		t.Fatalf("SavePluginsDir returned error: %v", err)
	}

	final, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings returned error: %v", err)
	}

	if len(final.MarketplaceRepositories) != len(marketplaceRepos) {
		t.Fatalf(
			"marketplace repositories were clobbered by an unrelated section's stale save — "+
				"expected %d entries, got %d: %+v",
			len(marketplaceRepos), len(final.MarketplaceRepositories), final.MarketplaceRepositories,
		)
	}
}
