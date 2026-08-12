// Package storage resolves the on-disk directory LiteLens uses for persistent
// app data (settings, installed plugins). In development mode (when SetDevMode(true)
// is called), it returns build/storage relative to the current working directory.
// Otherwise it defaults to ~/.litelens.
package storage

import (
	"log"
	"os"
	"path/filepath"
)

var devMode bool

// SetDevMode enables or disables development mode. When enabled, Dir() resolves
// to build/storage relative to the current working directory instead of ~/.litelens.
func SetDevMode(mode bool) {
	devMode = mode
}

// Dir returns the LiteLens storage directory. In development mode, this is
// build/storage relative to the current working directory; otherwise it is
// ~/.litelens. Both are joined with any additional path elements.
func Dir(elem ...string) string {
	if devMode {
		return devBuildDir(elem...)
	}
	return homeDir(elem...)
}

func devBuildDir(elem ...string) string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Printf("storage: failed to get working directory: %v; falling back to ~/.litelens", err)
		return homeDir(elem...)
	}
	return filepath.Join(append([]string{cwd, "build", "storage"}, elem...)...)
}

func homeDir(elem ...string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.ExpandEnv("$HOME")
	}
	return filepath.Join(append([]string{home, ".litelens"}, elem...)...)
}
