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
