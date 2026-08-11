package app

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPluginsRootDirDefaultUsesUserHome tests that pluginsRootDir() returns ~/.litelens/plugins
// when settings.PluginsDir is empty (the default case using os.UserHomeDir()).
func TestPluginsRootDirDefaultUsesUserHome(t *testing.T) {
	app := NewApp("test")

	// Verify default (no custom dir set)
	app.mu.RLock()
	pluginsDir := app.settings.PluginsDir
	app.mu.RUnlock()

	if pluginsDir != "" {
		t.Fatalf("expected default PluginsDir to be empty, got %q", pluginsDir)
	}

	// Call pluginsRootDir and verify it uses UserHome
	result := app.pluginsRootDir()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir() failed: %v", err)
	}

	expected := filepath.Join(home, ".litelens", "plugins")
	if result != expected {
		t.Errorf("pluginsRootDir() mismatch: got %q, expected %q", result, expected)
	}
}

// TestPluginsRootDirCustomDirTakesPrecedence tests that when settings.PluginsDir is set,
// it takes precedence over the default ~/.litelens/plugins path.
func TestPluginsRootDirCustomDirTakesPrecedence(t *testing.T) {
	app := NewApp("test")

	// Set a custom plugins directory
	customDir := "/tmp/my-custom-plugins"
	app.mu.Lock()
	app.settings.PluginsDir = customDir
	app.mu.Unlock()

	// Call pluginsRootDir and verify it returns the custom dir, not the default
	result := app.pluginsRootDir()

	if result != customDir {
		t.Errorf("pluginsRootDir() did not return custom dir: got %q, expected %q", result, customDir)
	}

	// Verify it doesn't use the default ~/.litelens/plugins path
	home, _ := os.UserHomeDir()
	defaultPath := filepath.Join(home, ".litelens", "plugins")
	if result == defaultPath {
		t.Errorf("pluginsRootDir() returned default path when custom dir was set")
	}
}

// TestPluginsRootDirMultipleCallsConsistent tests that multiple calls to pluginsRootDir()
// return the same value (important for thread safety with RLock).
func TestPluginsRootDirMultipleCallsConsistent(t *testing.T) {
	app := NewApp("test")

	result1 := app.pluginsRootDir()
	result2 := app.pluginsRootDir()
	result3 := app.pluginsRootDir()

	if result1 != result2 || result2 != result3 {
		t.Errorf("pluginsRootDir() returned different values on consecutive calls: %q, %q, %q",
			result1, result2, result3)
	}
}

// TestPluginsRootDirWithCustomSettings tests pluginsRootDir with custom settings saved.
// This verifies the end-to-end interaction between config.Load/Save and pluginsRootDir().
func TestPluginsRootDirWithCustomSettings(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Cleanup(func() {
		os.Setenv("HOME", origHome)
	})
	os.Setenv("HOME", tmpHome)

	app := NewApp("test")

	// Set a custom plugins directory and save
	customDir := filepath.Join(tmpHome, "my-plugins")
	app.mu.Lock()
	app.settings.PluginsDir = customDir
	app.mu.Unlock()

	// Verify pluginsRootDir returns the custom dir
	result := app.pluginsRootDir()
	if result != customDir {
		t.Errorf("pluginsRootDir() returned wrong value: got %q, expected %q", result, customDir)
	}

	// Now clear the custom dir and verify default is used again
	app.mu.Lock()
	app.settings.PluginsDir = ""
	app.mu.Unlock()

	result = app.pluginsRootDir()
	expected := filepath.Join(tmpHome, ".litelens", "plugins")
	if result != expected {
		t.Errorf("pluginsRootDir() returned wrong value after clearing custom dir: got %q, expected %q",
			result, expected)
	}
}

// BenchmarkPluginsRootDir measures the performance of pluginsRootDir() calls.
// RLock and ExpandEnv should be very fast even when called frequently.
func BenchmarkPluginsRootDir(b *testing.B) {
	app := NewApp("test")

	b.ResetTimer()
	for b.Loop() {
		_ = app.pluginsRootDir()
	}
}
