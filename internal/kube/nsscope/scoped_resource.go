// Package nsscope provides a generic, resource-agnostic engine for backing a
// client-go lister with either a single cluster-wide informer or a set of
// per-namespace informers, rebuildable at runtime as a caller's namespace
// filter changes. It exists because client-go's SharedInformerFactory only
// supports a single namespace scope for its entire lifetime
// (informers.WithNamespace), with no way to scope one resource type to a
// handful of namespaces without paying for a cluster-wide LIST — see
// FactoryHandle in the parent kube package, which was the original motivating
// use case (Pods).
//
// A ScopedResource is generic over L, the resource's client-go lister
// interface (e.g. k8s.io/client-go/listers/core/v1.PodLister). Callers
// supply the resource-specific glue (how to build a namespaced informer, a
// cluster-wide informer, a single-namespace lister, and a lister that fans
// out across several namespaces) via Config; ScopedResource owns the
// concurrency-safe swap/rebuild/stop lifecycle around those hooks.
package nsscope

import (
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	kmeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/cache"
)

// DefaultMaxNamespaces is used when Config.MaxNamespaces is zero.
const DefaultMaxNamespaces = 10

// DefaultSyncTimeout is used when Config.SyncTimeout is zero.
const DefaultSyncTimeout = 30 * time.Second

// group is the set of informer(s) currently backing a ScopedResource: either
// a single cluster-wide informer (namespaces == nil) or one namespace-scoped
// informer per entry in namespaces.
type group struct {
	namespaces []string // sorted; nil = cluster-wide
	informers  []cache.SharedIndexInformer
	stop       chan struct{}
	stopOnce   sync.Once
	synced     chan struct{}
}

// Config holds the resource-specific hooks a ScopedResource needs. All
// function fields are required unless noted otherwise.
type Config[L any] struct {
	// Name identifies the resource for logging (e.g. "pods").
	Name string

	// MaxNamespaces caps how many per-namespace informers Rescope will
	// build before falling back to a single cluster-wide informer (N
	// separate LISTs cost more round trips than one cluster-wide LIST
	// beyond this point). Zero uses DefaultMaxNamespaces.
	MaxNamespaces int

	// SyncTimeout bounds how long Rescope's background sync-wait will wait
	// for a newly built group's informers to complete their initial LIST
	// before marking the resource forbidden. Zero uses DefaultSyncTimeout.
	SyncTimeout time.Duration

	// NewNamespacedInformer builds a single-namespace informer scoped to ns.
	NewNamespacedInformer func(ns string) cache.SharedIndexInformer

	// NewClusterWideInformer builds a single cluster-wide informer,
	// typically backed by a shared SharedInformerFactory so it's reused
	// across every consumer of that factory rather than duplicated.
	NewClusterWideInformer func() cache.SharedIndexInformer

	// BuildLister builds this resource's lister from one informer's
	// indexer. Used for both the cluster-wide informer and each
	// per-namespace informer.
	BuildLister func(indexer cache.Indexer) L

	// BuildMultiLister fans a set of per-namespace listers (keyed by
	// namespace) out into a single L satisfying the same lister interface.
	// Only called when Rescope is building more than one namespace-scoped
	// informer.
	BuildMultiLister func(listers map[string]L) L

	// ClearForbidden is called at the start of a Rescope call that will
	// actually rebuild the group (i.e. not a no-op), before the rebuild
	// starts, so a namespace-filter change gives a previously-forbidden
	// resource a clean slate to retry against. Optional.
	ClearForbidden func()

	// OnForbidden is called when every informer in the *current* group
	// fails to sync (watch error containing "is forbidden", or sync
	// timeout). Never called on behalf of a group that's already been
	// superseded by a later Rescope call. Optional.
	OnForbidden func(name string)
}

// ScopedResource manages a swappable set of namespace-scoped (or single
// cluster-wide) SharedIndexInformers for one resource type, rebuilt on
// demand via Rescope. Safe for concurrent use.
type ScopedResource[L any] struct {
	cfg Config[L]

	mu           sync.RWMutex
	group        *group
	lister       L
	eventHandler func(namespace string)
}

// New constructs a ScopedResource and bootstraps it with a cluster-wide
// group (equivalent to Rescope(nil)), so Lister/SyncedChan never observe an
// unset state.
func New[L any](cfg Config[L]) *ScopedResource[L] {
	if cfg.MaxNamespaces <= 0 {
		cfg.MaxNamespaces = DefaultMaxNamespaces
	}
	if cfg.SyncTimeout <= 0 {
		cfg.SyncTimeout = DefaultSyncTimeout
	}
	r := &ScopedResource[L]{cfg: cfg}
	r.Rescope(nil)
	return r
}

// Rescope (re)builds the informer(s) backing this resource to cover exactly
// the given namespaces, replacing whatever was there before. nil, an empty
// slice, or more than Config.MaxNamespaces namespaces falls back to a single
// cluster-wide informer. A no-op if the requested scope already matches the
// current one.
//
// Safe to call at any time, including concurrently with reads via
// Lister/SyncedChan — callers always see either the old group or the fully
// built new one, never a half-built one.
func (r *ScopedResource[L]) Rescope(namespaces []string) {
	target := normalize(namespaces, r.cfg.MaxNamespaces)

	r.mu.RLock()
	current := r.group
	r.mu.RUnlock()
	if current != nil && sameNamespaces(current.namespaces, target) {
		return
	}

	if r.cfg.ClearForbidden != nil {
		r.cfg.ClearForbidden()
	}

	g := r.buildGroup(target)

	r.mu.Lock()
	old := r.group
	r.group = g
	r.lister = r.buildLister(g)
	r.mu.Unlock()

	if old != nil {
		old.stopOnce.Do(func() { close(old.stop) })
	}
}

// SetEventHandler registers fn to be called with the namespace of every
// add/update/delete observed by the current and any future informer(s) this
// resource builds (i.e. it survives Rescope rebuilds).
func (r *ScopedResource[L]) SetEventHandler(fn func(namespace string)) {
	r.mu.Lock()
	r.eventHandler = fn
	r.mu.Unlock()
}

// Lister returns the current lister, scoped exactly as configured by the
// most recent Rescope call (or cluster-wide if Rescope has never been called
// directly — New bootstraps a cluster-wide default).
func (r *ScopedResource[L]) Lister() L {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.lister
}

// SyncedChan returns a channel that closes once the current group's initial
// cache sync completes (or times out/fails).
func (r *ScopedResource[L]) SyncedChan() <-chan struct{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.group != nil {
		return r.group.synced
	}
	ch := make(chan struct{})
	close(ch)
	return ch
}

// Stop stops the current group's informer(s). Safe to call multiple times
// and concurrently with Rescope.
func (r *ScopedResource[L]) Stop() {
	r.mu.RLock()
	g := r.group
	r.mu.RUnlock()
	if g != nil {
		g.stopOnce.Do(func() { close(g.stop) })
	}
}

func (r *ScopedResource[L]) buildGroup(namespaces []string) *group {
	g := &group{
		namespaces: namespaces,
		stop:       make(chan struct{}),
		synced:     make(chan struct{}),
	}

	if len(namespaces) == 0 {
		g.informers = []cache.SharedIndexInformer{r.cfg.NewClusterWideInformer()}
	} else {
		g.informers = make([]cache.SharedIndexInformer, len(namespaces))
		for i, ns := range namespaces {
			g.informers[i] = r.cfg.NewNamespacedInformer(ns)
		}
	}

	handler := cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj any) { r.fireEvent(objNamespace(obj)) },
		UpdateFunc: func(_, newObj any) { r.fireEvent(objNamespace(newObj)) },
		DeleteFunc: func(obj any) { r.fireEvent(objNamespace(obj)) },
	}
	syncFuncs := make([]cache.InformerSynced, len(g.informers))
	for i, inf := range g.informers {
		//nolint:errcheck — only fails if already started, which it isn't yet
		inf.AddEventHandler(handler)
		//nolint:errcheck — only fails if already started, which it isn't yet
		inf.SetWatchErrorHandler(func(_ *cache.Reflector, err error) {
			if strings.Contains(err.Error(), "is forbidden") {
				r.markForbidden(g)
			}
		})
		syncFuncs[i] = inf.HasSynced
		go inf.Run(g.stop)
	}

	go func() {
		stopOrTimeout := make(chan struct{})
		go func() {
			timer := time.NewTimer(r.cfg.SyncTimeout)
			defer timer.Stop()
			select {
			case <-g.stop:
			case <-timer.C:
			}
			close(stopOrTimeout)
		}()
		cache.WaitForCacheSync(stopOrTimeout, syncFuncs...)
		for _, inf := range g.informers {
			if !inf.HasSynced() {
				log.Printf("kube/nsscope: cache sync timed out for %q — marking forbidden", r.cfg.Name)
				r.markForbidden(g)
				break
			}
		}
		close(g.synced)
	}()

	return g
}

func (r *ScopedResource[L]) buildLister(g *group) L {
	if len(g.namespaces) == 0 {
		return r.cfg.BuildLister(g.informers[0].GetIndexer())
	}
	m := make(map[string]L, len(g.namespaces))
	for i, ns := range g.namespaces {
		m[ns] = r.cfg.BuildLister(g.informers[i].GetIndexer())
	}
	return r.cfg.BuildMultiLister(m)
}

// markForbidden records this resource as forbidden and notifies OnForbidden
// — but only if g is still the current group (an older, already-superseded
// group's failure shouldn't override a newer, healthy one).
func (r *ScopedResource[L]) markForbidden(g *group) {
	g.stopOnce.Do(func() { close(g.stop) })

	r.mu.RLock()
	current := r.group == g
	r.mu.RUnlock()
	if !current {
		// g was already superseded by a later Rescope call — its failure
		// (e.g. sync timeout because it got stopped mid-sync) says nothing
		// about the current group's health.
		return
	}

	if r.cfg.OnForbidden != nil {
		r.cfg.OnForbidden(r.cfg.Name)
	}
}

func (r *ScopedResource[L]) fireEvent(namespace string) {
	r.mu.RLock()
	fn := r.eventHandler
	r.mu.RUnlock()
	if fn != nil {
		fn(namespace)
	}
}

func normalize(namespaces []string, max int) []string {
	if len(namespaces) == 0 || len(namespaces) > max {
		return nil
	}
	out := append([]string(nil), namespaces...)
	sort.Strings(out)
	return out
}

func sameNamespaces(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func objNamespace(obj any) string {
	if d, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		obj = d.Obj
	}
	if o, err := kmeta.Accessor(obj); err == nil {
		return o.GetNamespace()
	}
	return ""
}
