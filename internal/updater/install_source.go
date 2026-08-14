package updater

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	InstallSourceHomebrew = "homebrew"
	InstallSourceApt      = "apt"
	InstallSourceWinget   = "winget"
	InstallSourceManual   = "manual"
)

// DetectInstallSource reports which channel the running binary was installed
// through, so the self-updater (and the Settings UI) can distinguish it from
// a manual scripts/install.sh or raw-binary install. Best-effort: falls back
// to InstallSourceManual whenever detection is inconclusive.
func DetectInstallSource() string {
	exe, err := os.Executable()
	if err != nil {
		return InstallSourceManual
	}

	switch runtime.GOOS {
	case "linux":
		if IsAptManagedInstall(exe) {
			return InstallSourceApt
		}
	case "darwin":
		if realPath, err := filepath.EvalSymlinks(exe); err == nil && IsHomebrewCaskroomPath(realPath) {
			return InstallSourceHomebrew
		}
	case "windows":
		if isWingetManagedPath(exe) {
			return InstallSourceWinget
		}
	}

	return InstallSourceManual
}

// IsHomebrewCaskroomPath reports whether realPath (the symlink-resolved
// executable path) lives inside a Homebrew Caskroom, i.e. /Applications/*.app
// was installed as a Homebrew cask symlink rather than a real copy (which is
// what scripts/install.sh and the cask's own postflight-signed copy produce).
func IsHomebrewCaskroomPath(realPath string) bool {
	return strings.Contains(realPath, "/Caskroom/")
}

// isWingetManagedPath reports whether path looks like a winget-managed
// portable-exe install, which lands under
// %LOCALAPPDATA%\Microsoft\WinGet\Packages\... with a launch shim under
// %LOCALAPPDATA%\Microsoft\WinGet\Links\....
func isWingetManagedPath(path string) bool {
	lower := strings.ToLower(path)
	return strings.Contains(lower, `winget\packages`) || strings.Contains(lower, `winget\links`)
}
