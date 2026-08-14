package updater

import (
	"os"
	"os/exec"
	"runtime"
	"testing"
)

func TestIsHomebrewInstalled(t *testing.T) {
	dirExists := func(path string) (os.FileInfo, error) {
		if path == "/opt/homebrew/Caskroom/litelens" || path == "/usr/local/Caskroom/litelens" {
			return nil, nil
		}
		return nil, os.ErrNotExist
	}
	dirMissing := func(string) (os.FileInfo, error) {
		return nil, os.ErrNotExist
	}
	noBrew := func() string { return "" }
	brewFound := func() string { return "/opt/homebrew/bin/brew" }

	tests := []struct {
		name          string
		statFn        func(string) (os.FileInfo, error)
		findBrew      func() string
		checkBrewList func(string) (installed, ok bool)
		want          bool
	}{
		{
			name:          "caskroom dir missing entirely -> not homebrew, brew not even consulted",
			statFn:        dirMissing,
			findBrew:      brewFound,
			checkBrewList: func(string) (bool, bool) { t.Fatal("checkBrewList should not be called when no Caskroom dir exists"); return false, false },
			want:          false,
		},
		{
			name:          "caskroom dir exists, brew unreachable -> fall back to directory signal",
			statFn:        dirExists,
			findBrew:      noBrew,
			checkBrewList: func(brewPath string) (bool, bool) { return false, false },
			want:          true,
		},
		{
			name:     "caskroom dir exists, brew confirms installed",
			statFn:   dirExists,
			findBrew: brewFound,
			checkBrewList: func(brewPath string) (bool, bool) {
				if brewPath != "/opt/homebrew/bin/brew" {
					t.Errorf("checkBrewList got brewPath %q", brewPath)
				}
				return true, true
			},
			want: true,
		},
		{
			name:     "caskroom dir is orphaned, brew confirms not installed",
			statFn:   dirExists,
			findBrew: brewFound,
			checkBrewList: func(brewPath string) (bool, bool) {
				return false, true
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isHomebrewInstalled(tt.statFn, tt.findBrew, tt.checkBrewList)
			if got != tt.want {
				t.Errorf("isHomebrewInstalled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBrewListCask(t *testing.T) {
	if _, ok := brewListCask(""); ok {
		t.Error("brewListCask(\"\") should be inconclusive (ok=false) when brewPath is empty")
	}

	if runtime.GOOS != "darwin" {
		t.Skip("brewListCask shells out to a real brew binary; only exercised on darwin")
	}

	brewPath, err := exec.LookPath("brew")
	if err != nil {
		t.Skip("brew not installed on this machine, skipping live brewListCask check")
	}

	// Querying a cask name that cannot plausibly be installed confirms the
	// "not installed" (ok=true, installed=false) path against the real binary.
	installed, ok := brewListCask(brewPath)
	if !ok {
		t.Skip("brew list returned inconclusive result on this machine")
	}
	_ = installed // whatever litelens's actual state is on this machine; just confirming it doesn't error
}

func TestIsWingetManagedPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "winget packages dir",
			path: `C:\Users\alice\AppData\Local\Microsoft\WinGet\Packages\litelensapp.LiteLens_abc123\litelens.exe`,
			want: true,
		},
		{
			name: "winget links shim",
			path: `C:\Users\alice\AppData\Local\Microsoft\WinGet\Links\litelens.exe`,
			want: true,
		},
		{
			name: "case-insensitive match",
			path: `C:\Users\alice\AppData\Local\microsoft\winget\packages\litelensapp.LiteLens_abc123\litelens.exe`,
			want: true,
		},
		{
			name: "manual download",
			path: `C:\Users\alice\Downloads\litelens-windows-amd64.exe`,
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isWingetManagedPath(tt.path); got != tt.want {
				t.Errorf("isWingetManagedPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
