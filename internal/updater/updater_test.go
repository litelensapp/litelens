package updater

import "testing"

// TestMain stubs checkHomebrewManaged for the whole package test run so
// Check()'s tests aren't coupled to the actual test machine's real
// Homebrew/Caskroom state (see the checkHomebrewManaged doc comment).
// isHomebrewInstalled itself is still exercised directly and unaffected,
// since its tests call it with injected statFn/findBrew/checkBrewList
// functions rather than going through this override.
func TestMain(m *testing.M) {
	checkHomebrewManaged = func() bool { return false }
	m.Run()
}
