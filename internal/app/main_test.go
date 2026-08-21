package app

import (
	"os"
	"testing"
	"time"

	"github.com/litelensapp/litelens/internal/kube"
	"github.com/litelensapp/litelens/internal/updater"
)

// TestMain stubs the updater package's Homebrew-managed-install probe for the
// whole package test run so checkForUpdate/CheckForUpdate tests aren't
// coupled to the real test machine's actual Homebrew/Caskroom state (a dev
// machine with litelens installed via `brew install` would otherwise make
// updater.Check short-circuit before ever reaching the mock release server).
//
// It also shrinks kube.ResyncStaggerInterval: at the production value
// (300ms), with ~30 resources registered in NewFactoryHandle, the
// last-registered resource ("events") wouldn't start syncing for ~9s —
// this package's informer_sync_gating tests call NewFactoryHandle and wait
// on events sync directly, so that cost was paid on every test run.
//
// It also shrinks updater.RetryBackoffSchedule (production: 5s, 10s) — this
// package's checkForUpdate retry-exhaustion tests otherwise take 15s each.
func TestMain(m *testing.M) {
	restoreHomebrew := updater.SetCheckHomebrewManagedForTest(func() bool { return false })
	kube.ResyncStaggerInterval = time.Millisecond
	updater.RetryBackoffSchedule = []time.Duration{time.Millisecond, time.Millisecond}
	code := m.Run()
	restoreHomebrew()
	os.Exit(code)
}
