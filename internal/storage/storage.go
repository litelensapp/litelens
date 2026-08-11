// Package storage resolves the on-disk directory LiteLens uses for persistent
// app data (settings, installed plugins), which defaults to ~/.litelens.
package storage

import (
	"os"
	"path/filepath"
)

// Dir returns ~/.litelens, joined with any additional path elements.
func Dir(elem ...string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.ExpandEnv("$HOME")
	}
	return filepath.Join(append([]string{home, ".litelens"}, elem...)...)
}
