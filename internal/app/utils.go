package app

import (
	kmeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/cache"
)

// nsFromObj extracts the namespace from an informer event object or deletion tombstone.
func nsFromObj(obj any) string {
	if d, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		obj = d.Obj
	}
	if o, err := kmeta.Accessor(obj); err == nil {
		return o.GetNamespace()
	}
	return ""
}

func (a *App) isActive(ctx string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.activeContext == ctx
}

// tryClaimConnectSeq reports whether seq is newer than every Connect call
// seen so far and, if so, claims it as the current one. A stale or duplicate
// seq (<=  the highest already claimed) is rejected without side effects, so
// the caller can bail out before doing any of Connect's slow client-building
// or informer-syncing work. See Connect.
func (a *App) tryClaimConnectSeq(seq int64) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if seq <= a.activeContextSeq {
		return false
	}
	a.activeContextSeq = seq
	return true
}
