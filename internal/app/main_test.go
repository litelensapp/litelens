package app

import (
	"os"
	"testing"

	"github.com/litelensapp/litelens/internal/updater"
)

// TestMain stubs the updater package's Homebrew-managed-install probe for the
// whole package test run so checkForUpdate/CheckForUpdate tests aren't
// coupled to the real test machine's actual Homebrew/Caskroom state (a dev
// machine with litelens installed via `brew install` would otherwise make
// updater.Check short-circuit before ever reaching the mock release server).
func TestMain(m *testing.M) {
	restore := updater.SetCheckHomebrewManagedForTest(func() bool { return false })
	code := m.Run()
	restore()
	os.Exit(code)
}
