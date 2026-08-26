package app

import (
	"sync"
	"testing"
)

// TestTryClaimConnectSeq_RejectsStaleAndDuplicate verifies the ordering guard
// used by Connect: a call with a seq no higher than the highest already
// claimed is rejected, and a strictly higher seq is accepted and becomes the
// new high-water mark.
func TestTryClaimConnectSeq_RejectsStaleAndDuplicate(t *testing.T) {
	a := &App{}

	if !a.tryClaimConnectSeq(1) {
		t.Fatal("expected first claim (seq=1) to succeed")
	}
	if !a.tryClaimConnectSeq(3) {
		t.Fatal("expected higher seq (seq=3) to succeed")
	}
	if a.tryClaimConnectSeq(2) {
		t.Fatal("expected stale seq (seq=2, after seq=3 already claimed) to be rejected")
	}
	if a.tryClaimConnectSeq(3) {
		t.Fatal("expected duplicate seq (seq=3) to be rejected")
	}
	if !a.tryClaimConnectSeq(4) {
		t.Fatal("expected higher seq (seq=4) to succeed")
	}
}

// TestTryClaimConnectSeq_ConcurrentOutOfOrderCompletion reproduces the actual
// bug: rapid back-and-forth cluster switches fire concurrent Connect calls
// whose slow network/informer-sync work can finish in an order that doesn't
// match the order the user clicked them in. Without this guard, whichever
// goroutine reaches the state-mutating section of Connect last would win,
// even if it was an earlier (now-stale) click. This asserts that regardless
// of completion order, only the call carrying the numerically highest seq
// is ever left as the claimed one.
func TestTryClaimConnectSeq_ConcurrentOutOfOrderCompletion(t *testing.T) {
	a := &App{}
	const n = 50

	var wg sync.WaitGroup
	claimed := make([]bool, n+1) // index 0 unused; seq values are 1..n
	for seq := int64(1); seq <= n; seq++ {
		wg.Add(1)
		go func(seq int64) {
			defer wg.Done()
			claimed[seq] = a.tryClaimConnectSeq(seq)
		}(seq)
	}
	wg.Wait()

	a.mu.RLock()
	finalSeq := a.activeContextSeq
	a.mu.RUnlock()
	if finalSeq != n {
		t.Fatalf("expected activeContextSeq to end at the highest seq (%d), got %d", n, finalSeq)
	}

	// A later call attempted after every goroutine above has finished must
	// still correctly recognize anything <= n as stale.
	if a.tryClaimConnectSeq(n) {
		t.Fatalf("expected seq=%d to be rejected as stale once %d is already claimed", n, n)
	}
	if !a.tryClaimConnectSeq(n + 1) {
		t.Fatalf("expected seq=%d (strictly newer) to be accepted", n+1)
	}
}
