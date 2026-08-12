package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDir(t *testing.T) {
	tests := []struct {
		name           string
		devMode        bool
		elem           []string
		envOverride    string
		wantSuffix     string
		wantStartsWith string
	}{
		{
			name:           "production mode returns .litelens path",
			devMode:        false,
			elem:           []string{},
			envOverride:    "",
			wantSuffix:     ".litelens",
			wantStartsWith: "",
		},
		{
			name:           "production mode with elem appends correctly",
			devMode:        false,
			elem:           []string{"config", "settings.json"},
			envOverride:    "",
			wantSuffix:     filepath.Join(".litelens", "config", "settings.json"),
			wantStartsWith: "",
		},
		{
			name:           "dev mode returns build/storage path",
			devMode:        true,
			elem:           []string{},
			envOverride:    "",
			wantSuffix:     filepath.Join("build", "storage"),
			wantStartsWith: "",
		},
		{
			name:           "dev mode with elem appends correctly",
			devMode:        true,
			elem:           []string{"plugins", "list.json"},
			envOverride:    "",
			wantSuffix:     filepath.Join("build", "storage", "plugins", "list.json"),
			wantStartsWith: "",
		},
		{
			name:           "production mode with LITELENS_ROOT_DIR override",
			devMode:        false,
			elem:           []string{},
			envOverride:    "/custom/path",
			wantSuffix:     "/custom/path",
			wantStartsWith: "/custom/path",
		},
		{
			name:           "production mode with LITELENS_ROOT_DIR override and elem",
			devMode:        false,
			elem:           []string{"config", "settings.json"},
			envOverride:    "/custom/path",
			wantSuffix:     filepath.Join("/custom/path", "config", "settings.json"),
			wantStartsWith: "/custom/path",
		},
		{
			name:           "dev mode ignores LITELENS_ROOT_DIR override",
			devMode:        true,
			elem:           []string{},
			envOverride:    "/custom/path",
			wantSuffix:     filepath.Join("build", "storage"),
			wantStartsWith: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				devMode = false
				rootDirOverride = ""
			})

			if tt.envOverride != "" {
				SetRootDirOverride(tt.envOverride)
			}

			SetDevMode(tt.devMode)
			got := Dir(tt.elem...)

			if !strings.HasSuffix(got, tt.wantSuffix) {
				t.Errorf("Dir() = %q, want suffix %q", got, tt.wantSuffix)
			}

			// For dev mode, verify it's an absolute path starting with cwd
			if tt.devMode {
				cwd, err := os.Getwd()
				if err != nil {
					t.Fatalf("os.Getwd() failed: %v", err)
				}
				if !strings.HasPrefix(got, cwd) {
					t.Errorf("Dev mode Dir() = %q, want to start with cwd %q", got, cwd)
				}
			}

			// For production mode with override, verify it uses the override
			if !tt.devMode && tt.envOverride != "" {
				if tt.wantStartsWith != "" && !strings.HasPrefix(got, tt.wantStartsWith) {
					t.Errorf("Production mode with override Dir() = %q, want to start with %q", got, tt.wantStartsWith)
				}
			}

			// For production mode without override, verify it contains home directory
			if !tt.devMode && tt.envOverride == "" {
				home, err := os.UserHomeDir()
				if err != nil {
					home = os.ExpandEnv("$HOME")
				}
				if !strings.HasPrefix(got, home) {
					t.Errorf("Production mode Dir() = %q, want to start with home %q", got, home)
				}
			}
		})
	}
}

func TestDevModeIsIsolated(t *testing.T) {
	t.Cleanup(func() {
		devMode = false
	})

	// Set dev mode
	SetDevMode(true)
	path1 := Dir()

	// Reset and check it's false
	SetDevMode(false)
	path2 := Dir()

	if strings.Contains(path1, "build/storage") {
		t.Logf("Dev mode path: %s", path1)
	} else {
		t.Errorf("Expected dev mode path to contain build/storage, got %q", path1)
	}

	if strings.Contains(path2, ".litelens") {
		t.Logf("Production mode path: %s", path2)
	} else {
		t.Errorf("Expected production mode path to contain .litelens, got %q", path2)
	}

	// Verify they're different
	if path1 == path2 {
		t.Errorf("Dev mode and production mode paths should differ, but both are %q", path1)
	}
}
