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

// IsHomebrewCaskroomPath reports whether the running executable was installed
// via Homebrew Cask by checking for the existence of the known Homebrew
// Caskroom directory for the litelens cask. The executable itself may live in
// /Applications (a real copy, not a symlink), so we check for the Caskroom
// directory independently. Works on both Apple Silicon and Intel macOS.
// The realPath parameter is kept for API compatibility but not used;
// detection is based on filesystem checks.
func IsHomebrewCaskroomPath(realPath string) bool {
	return isHomebrewCaskroomPathChecked(os.Stat)
}

// isHomebrewCaskroomPathChecked is a testable variant that accepts a stat
// function, allowing tests to mock filesystem checks without requiring actual
// Homebrew installation.
func isHomebrewCaskroomPathChecked(statFn func(string) (os.FileInfo, error)) bool {
	// Check for both known Homebrew Caskroom paths on macOS.
	// Apple Silicon: /opt/homebrew/Caskroom/litelens
	// Intel: /usr/local/Caskroom/litelens
	armPath := "/opt/homebrew/Caskroom/litelens"
	intelPath := "/usr/local/Caskroom/litelens"

	_, errARM := statFn(armPath)
	_, errIntel := statFn(intelPath)

	return errARM == nil || errIntel == nil
}

// isWingetManagedPath reports whether path looks like a winget-managed
// portable-exe install, which lands under
// %LOCALAPPDATA%\Microsoft\WinGet\Packages\... with a launch shim under
// %LOCALAPPDATA%\Microsoft\WinGet\Links\....
func isWingetManagedPath(path string) bool {
	lower := strings.ToLower(path)
	return strings.Contains(lower, `winget\packages`) || strings.Contains(lower, `winget\links`)
}
