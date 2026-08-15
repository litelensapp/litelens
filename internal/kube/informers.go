package kube

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/litelensapp/litelens/internal/lib/debouncer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// stopEntry pairs a stop channel with a sync.Once so the channel is closed at most once.
type stopEntry struct {
	ch   chan struct{}
	once sync.Once
}

// FactoryHandle wraps a SharedInformerFactory with per-informer stop channels.
type FactoryHandle struct {
	Factory      informers.SharedInformerFactory
	stopChannels map[string]*stopEntry
	globalStop   chan struct{}
	globalOnce   sync.Once
	forbidden    sync.Map // map[string]struct{} — resource keys that failed to sync
	debouncers   []*debouncer.Debouncer
	synced       map[string]chan struct{}
	syncedOnce   map[string]*sync.Once
}

// StopResource closes the per-resource stop channel (at most once), records the
// resource as forbidden, and calls onForbidden — unless Stop() has already fired.
// Exported for testing edge cases.
func (h *FactoryHandle) StopResource(resource string, onForbidden func(string)) {
	if e, ok := h.stopChannels[resource]; ok {
		e.once.Do(func() { close(e.ch) })
	}
	h.forbidden.Store(resource, struct{}{})
	select {
	case <-h.globalStop:
		// Handle already stopped; skip emitting the event.
	default:
		onForbidden(resource)
	}
}

// NewFactoryHandle creates a factory, pre-registers all informers, wires up
// forbidden-error detection, starts each informer on its own stop channel,
// and returns the handle immediately. Each resource's cache is warmed
// asynchronously; callers should wait on GetSyncedChan(resource) before
// reading from a resource's lister to ensure the cache is warm.
// onForbidden is called once per resource key when a 403 is detected;
// keys match the ViewType strings used by the frontend (e.g. "ingresses").
// Call Stop() when the context is no longer active.
func NewFactoryHandle(cs kubernetes.Interface, onForbidden func(resource string)) *FactoryHandle {
	factory := informers.NewSharedInformerFactory(cs, 30*time.Second)

	h := &FactoryHandle{
		Factory:      factory,
		stopChannels: make(map[string]*stopEntry),
		globalStop:   make(chan struct{}),
		synced:       make(map[string]chan struct{}),
		syncedOnce:   make(map[string]*sync.Once),
	}

	type entry struct {
		inf      cache.SharedIndexInformer
		resource string
	}

	// Order mirrors NAV_CORE in frontend/src/app/clusters/navConfig.ts (top-to-bottom,
	// group-by-group) so the stagger delay below lines up with how soon each resource
	// is likely to be viewed/needed after connecting, instead of an arbitrary order.
	informerList := []entry{
		{factory.Core().V1().Namespaces().Informer(), "namespaces"},
		{factory.Core().V1().Nodes().Informer(), "nodes"},
		{factory.Core().V1().Pods().Informer(), "pods"},
		{factory.Apps().V1().Deployments().Informer(), "deployments"},
		{factory.Apps().V1().DaemonSets().Informer(), "daemonsets"},
		{factory.Apps().V1().StatefulSets().Informer(), "statefulsets"},
		{factory.Apps().V1().ReplicaSets().Informer(), "replicasets"},
		{factory.Batch().V1().Jobs().Informer(), "jobs"},
		{factory.Batch().V1().CronJobs().Informer(), "cronjobs"},
		{factory.Core().V1().ConfigMaps().Informer(), "configmaps"},
		{factory.Core().V1().Secrets().Informer(), "secrets"},
		{factory.Core().V1().ResourceQuotas().Informer(), "resourcequotas"},
		{factory.Core().V1().LimitRanges().Informer(), "limitranges"},
		{factory.Autoscaling().V2().HorizontalPodAutoscalers().Informer(), "hpa"},
		{factory.Policy().V1().PodDisruptionBudgets().Informer(), "pdbs"},
		{factory.Scheduling().V1().PriorityClasses().Informer(), "priorityclasses"},
		{factory.Coordination().V1().Leases().Informer(), "leases"},
		{factory.Admissionregistration().V1().ValidatingWebhookConfigurations().Informer(), "validatingwebhookconfigs"},
		{factory.Core().V1().Services().Informer(), "services"},
		{factory.Discovery().V1().EndpointSlices().Informer(), "endpointslices"},
		{factory.Core().V1().Endpoints().Informer(), "endpoints"},
		{factory.Networking().V1().Ingresses().Informer(), "ingresses"},
		{factory.Networking().V1().IngressClasses().Informer(), "ingressclasses"},
		{factory.Networking().V1().NetworkPolicies().Informer(), "networkpolicies"},
		{factory.Core().V1().PersistentVolumeClaims().Informer(), "pvcs"},
		{factory.Core().V1().PersistentVolumes().Informer(), "pvs"},
		{factory.Storage().V1().StorageClasses().Informer(), "storageclasses"},
		{factory.Core().V1().ServiceAccounts().Informer(), "serviceaccounts"},
		{factory.Rbac().V1().ClusterRoles().Informer(), "clusterroles"},
		{factory.Rbac().V1().Roles().Informer(), "roles"},
		{factory.Rbac().V1().ClusterRoleBindings().Informer(), "clusterrolebindings"},
		{factory.Rbac().V1().RoleBindings().Informer(), "rolebindings"},
		{factory.Core().V1().Events().Informer(), "events"},
	}

	// Allocate per-resource stop channels, sync channels, sync.Once, wire error handlers.
	// NOTE: infToResource is read-only after this loop; no lock needed.
	infToResource := make(map[cache.SharedIndexInformer]string, len(informerList))
	for _, e := range informerList {
		se := &stopEntry{ch: make(chan struct{})}
		h.stopChannels[e.resource] = se
		h.synced[e.resource] = make(chan struct{})
		h.syncedOnce[e.resource] = &sync.Once{}

		resource := e.resource // capture for closure
		//nolint:errcheck — only fails if already started, which it isn't yet
		e.inf.SetWatchErrorHandler(func(_ *cache.Reflector, err error) {
			if strings.Contains(err.Error(), "is forbidden") {
				h.StopResource(resource, onForbidden)
			}
		})
		infToResource[e.inf] = resource
	}

	// Start each informer on its own stop channel, staggering start times so the
	// informers' 30s resync tickers (which start when Run() is called and persist
	// for the informer's lifetime) don't all land on the same instant — otherwise
	// every resource's UPDATE events burst through the frontend's event handlers
	// simultaneously every 30s, which can stall the UI on large clusters.
	const resyncStagger = 300 * time.Millisecond
	for i, e := range informerList {
		inf := e.inf
		ch := h.stopChannels[e.resource].ch
		delay := time.Duration(i) * resyncStagger
		go func() {
			if delay > 0 {
				select {
				case <-time.After(delay):
				case <-ch:
					return
				}
			}
			inf.Run(ch)
		}()
	}

	// Start async per-resource sync goroutines. Each waits for its own resource's
	// cache sync with a 30s timeout. If timeout is exceeded, the resource is
	// marked forbidden. WatchErrorHandler remains the primary detector for
	// watch-phase 403s.
	for _, e := range informerList {
		resource := e.resource
		inf := e.inf
		go func() {
			stopOrTimeout := make(chan struct{})
			go func() {
				timer := time.NewTimer(30 * time.Second)
				defer timer.Stop()
				select {
				case <-h.stopChannels[resource].ch:
				case <-timer.C:
				}
				close(stopOrTimeout)
			}()
			cache.WaitForCacheSync(stopOrTimeout, inf.HasSynced)
			if !inf.HasSynced() {
				log.Printf("kube/informers: cache sync timed out for %q — marking forbidden", resource)
				h.StopResource(resource, onForbidden)
			}
			h.syncedOnce[resource].Do(func() { close(h.synced[resource]) })
		}()
	}

	return h
}

// GetSyncedChan returns a channel that closes once the given resource's initial
// cache sync completes (or times out/fails). Callers should <-chan before
// reading from that resource's lister. Unknown resources return an
// already-closed channel (nothing to wait for).
func (h *FactoryHandle) GetSyncedChan(resource string) <-chan struct{} {
	if h == nil {
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	if ch, ok := h.synced[resource]; ok {
		return ch
	}
	ch := make(chan struct{})
	close(ch)
	return ch
}

// IsForbidden reports whether the given resource key was denied access
// (either via a 403 during watch or a failed initial cache sync).
func (h *FactoryHandle) IsForbidden(resource string) bool {
	if h == nil {
		return false
	}
	_, ok := h.forbidden.Load(resource)
	return ok
}

// RegisterDebouncer records a debouncer for lifecycle management.
func (h *FactoryHandle) RegisterDebouncer(d *debouncer.Debouncer) {
	h.debouncers = append(h.debouncers, d)
}

// Stop shuts down all per-resource informers and the global sync goroutine.
// It also stops all registered debouncers before stopping the factory.
func (h *FactoryHandle) Stop() {
	for _, d := range h.debouncers {
		d.Stop()
	}
	h.globalOnce.Do(func() { close(h.globalStop) })
	for _, se := range h.stopChannels {
		se.once.Do(func() { close(se.ch) })
	}
}
