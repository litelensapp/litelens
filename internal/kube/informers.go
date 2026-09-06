package kube

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/litelensapp/litelens/internal/lib/debouncer"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	listersappsv1 "k8s.io/client-go/listers/apps/v1"
	listersautoscalingv2 "k8s.io/client-go/listers/autoscaling/v2"
	listersbatchv1 "k8s.io/client-go/listers/batch/v1"
	listerscoordinationv1 "k8s.io/client-go/listers/coordination/v1"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	listersdiscoveryv1 "k8s.io/client-go/listers/discovery/v1"
	listersnetworkingv1 "k8s.io/client-go/listers/networking/v1"
	listerspolicyv1 "k8s.io/client-go/listers/policy/v1"
	listersrbacv1 "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/litelensapp/litelens/internal/kube/nsscope"
)

// FactoryHandle's per-resource typed methods (RescopeXxx/SetXxxEventHandler/
// XxxLister) and the initScopedResources bootstrap live in resources.go.

// stopEntry pairs a stop channel with a sync.Once so the channel is closed at most once.
type stopEntry struct {
	ch   chan struct{}
	once sync.Once
}

// ResyncJitterStep is the per-resource-type increment added to the base 30s
// resync period (via informers.WithCustomResyncConfig) so every resource's
// periodic full re-list doesn't land on the same wall-clock instant forever
// after Connect(). Exported so tests can shrink it if a test ever needs to
// assert on resync behavior directly (none currently do).
var ResyncJitterStep = time.Second

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

	cs          kubernetes.Interface
	onForbidden func(resource string)

	// The resources below are managed separately from the generic ones above
	// (not in stopChannels/synced/syncedOnce) because their informer(s) can
	// be rebuilt at runtime by their RescopeXxx method to trade one
	// cluster-wide LIST for several namespace-scoped ones, via the generic
	// nsscope engine. scoped indexes all of them by resource key for the
	// bookkeeping methods below (StopResource/GetSyncedChan/Stop); the typed
	// fields exist alongside it so Rescope/Lister methods can return
	// resource-specific types.
	scoped map[string]nsscopeResource

	pod         *nsscope.ScopedResource[listerscorev1.PodLister]
	deployment  *nsscope.ScopedResource[listersappsv1.DeploymentLister]
	daemonset   *nsscope.ScopedResource[listersappsv1.DaemonSetLister]
	statefulset *nsscope.ScopedResource[listersappsv1.StatefulSetLister]
	replicaset  *nsscope.ScopedResource[listersappsv1.ReplicaSetLister]
	job         *nsscope.ScopedResource[listersbatchv1.JobLister]
	cronjob     *nsscope.ScopedResource[listersbatchv1.CronJobLister]

	configmap      *nsscope.ScopedResource[listerscorev1.ConfigMapLister]
	secret         *nsscope.ScopedResource[listerscorev1.SecretLister]
	resourcequota  *nsscope.ScopedResource[listerscorev1.ResourceQuotaLister]
	limitrange     *nsscope.ScopedResource[listerscorev1.LimitRangeLister]
	hpa            *nsscope.ScopedResource[listersautoscalingv2.HorizontalPodAutoscalerLister]
	pdb            *nsscope.ScopedResource[listerspolicyv1.PodDisruptionBudgetLister]
	lease          *nsscope.ScopedResource[listerscoordinationv1.LeaseLister]
	service        *nsscope.ScopedResource[listerscorev1.ServiceLister]
	endpointslice  *nsscope.ScopedResource[listersdiscoveryv1.EndpointSliceLister]
	endpoint       *nsscope.ScopedResource[listerscorev1.EndpointsLister]
	ingress        *nsscope.ScopedResource[listersnetworkingv1.IngressLister]
	networkpolicy  *nsscope.ScopedResource[listersnetworkingv1.NetworkPolicyLister]
	pvc            *nsscope.ScopedResource[listerscorev1.PersistentVolumeClaimLister]
	serviceaccount *nsscope.ScopedResource[listerscorev1.ServiceAccountLister]
	role           *nsscope.ScopedResource[listersrbacv1.RoleLister]
	rolebinding    *nsscope.ScopedResource[listersrbacv1.RoleBindingLister]
	event          *nsscope.ScopedResource[listerscorev1.EventLister]
}

// StopResource closes the per-resource stop channel (at most once), records the
// resource as forbidden, and calls onForbidden — unless Stop() has already fired.
// Exported for testing edge cases.
func (h *FactoryHandle) StopResource(resource string, onForbidden func(string)) {
	if sr, ok := h.scoped[resource]; ok {
		sr.Stop()
	} else if e, ok := h.stopChannels[resource]; ok {
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
	// Order mirrors NAV_CORE in frontend/src/app/clusters/navConfig.ts (top-to-bottom,
	// group-by-group). It no longer gates start time (see resyncConfig below) — it's
	// kept only because resyncTypes below reuses it to assign each type a distinct
	// resync period.
	resyncTypes := []metav1.Object{
		&corev1.Namespace{},
		&corev1.Node{},
		&corev1.Pod{},
		&appsv1.Deployment{},
		&appsv1.DaemonSet{},
		&appsv1.StatefulSet{},
		&appsv1.ReplicaSet{},
		&batchv1.Job{},
		&batchv1.CronJob{},
		&corev1.ConfigMap{},
		&corev1.Secret{},
		&corev1.ResourceQuota{},
		&corev1.LimitRange{},
		&autoscalingv2.HorizontalPodAutoscaler{},
		&policyv1.PodDisruptionBudget{},
		&schedulingv1.PriorityClass{},
		&coordinationv1.Lease{},
		&admissionregistrationv1.ValidatingWebhookConfiguration{},
		&corev1.Service{},
		&discoveryv1.EndpointSlice{},
		&corev1.Endpoints{},
		&networkingv1.Ingress{},
		&networkingv1.IngressClass{},
		&networkingv1.NetworkPolicy{},
		&corev1.PersistentVolumeClaim{},
		&corev1.PersistentVolume{},
		&storagev1.StorageClass{},
		&corev1.ServiceAccount{},
		&rbacv1.ClusterRole{},
		&rbacv1.Role{},
		&rbacv1.ClusterRoleBinding{},
		&rbacv1.RoleBinding{},
		&corev1.Event{},
	}
	// Give every resource type a distinct resync period (30s, 31s, 32s, ...)
	// instead of staggering when each informer starts. This keeps informers'
	// periodic full re-lists from landing on the same wall-clock instant
	// forever after (which would otherwise burst every resource's UPDATE
	// events through the frontend simultaneously every 30s on large
	// clusters), without delaying any resource's *initial* sync — every
	// informer now starts immediately, all at once.
	resyncConfig := make(map[metav1.Object]time.Duration, len(resyncTypes))
	for i, obj := range resyncTypes {
		resyncConfig[obj] = 30*time.Second + time.Duration(i)*ResyncJitterStep
	}
	factory := informers.NewSharedInformerFactoryWithOptions(cs, 30*time.Second,
		informers.WithCustomResyncConfig(resyncConfig))

	h := &FactoryHandle{
		Factory:      factory,
		stopChannels: make(map[string]*stopEntry),
		globalStop:   make(chan struct{}),
		synced:       make(map[string]chan struct{}),
		syncedOnce:   make(map[string]*sync.Once),
		cs:           cs,
		onForbidden:  onForbidden,
	}

	type entry struct {
		inf      cache.SharedIndexInformer
		resource string
	}

	// Every namespaced resource type is deliberately absent here — their
	// informer(s) are managed by their RescopeXxx method instead, so the
	// namespace filter known at Connect time (or set later via
	// SetActiveNamespaces) can avoid a cluster-wide LIST. See
	// nsscope.NewPodsResource and its siblings. Only cluster-scoped
	// resources (no namespace to scope by) remain generic below.
	informerList := []entry{
		{factory.Core().V1().Namespaces().Informer(), "namespaces"},
		{factory.Core().V1().Nodes().Informer(), "nodes"},
		{factory.Scheduling().V1().PriorityClasses().Informer(), "priorityclasses"},
		{factory.Admissionregistration().V1().ValidatingWebhookConfigurations().Informer(), "validatingwebhookconfigs"},
		{factory.Networking().V1().IngressClasses().Informer(), "ingressclasses"},
		{factory.Core().V1().PersistentVolumes().Informer(), "pvs"},
		{factory.Storage().V1().StorageClasses().Informer(), "storageclasses"},
		{factory.Rbac().V1().ClusterRoles().Informer(), "clusterroles"},
		{factory.Rbac().V1().ClusterRoleBindings().Informer(), "clusterrolebindings"},
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

	// Start every informer on its own stop channel immediately — the per-type
	// resync jitter configured above (resyncConfig) is what keeps their
	// periodic re-lists from landing simultaneously, so there's no need to
	// delay any resource's initial LIST+WATCH.
	for _, e := range informerList {
		inf := e.inf
		ch := h.stopChannels[e.resource].ch
		go func() {
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

	// Bootstrap each nsscope-managed resource (pods + workloads); see
	// FactoryHandle.initScopedResources in resources.go.
	h.initScopedResources()

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
	if sr, ok := h.scoped[resource]; ok {
		return sr.SyncedChan()
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
	for _, sr := range h.scoped {
		sr.Stop()
	}
}
