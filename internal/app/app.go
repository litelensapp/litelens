package app

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/litelensapp/litelens/internal/config"
	"github.com/litelensapp/litelens/internal/dto"
	"github.com/litelensapp/litelens/internal/kube"
	"github.com/litelensapp/litelens/internal/plugin"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/remotecommand"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
)

// metricsFetchTimeout bounds queries to metrics-server (cluster-local call on desktop-app IPC path).
// Long enough to tolerate normal latency; short enough to keep the UI responsive when metrics-server
// is unavailable or slow.
const metricsFetchTimeout = 3 * time.Second

// apiReadTimeout bounds direct clientset Get/List calls that bypass the informer cache
// (e.g. Get*YAML methods). These hit the live API server rather than a local cache.
const apiReadTimeout = 10 * time.Second

// apiMutationTimeout bounds Create/Update/Patch/Delete calls, which may wait on
// etcd persistence or admission webhooks and so get more headroom than reads.
const apiMutationTimeout = 30 * time.Second

// App struct
type App struct {
	ctx               context.Context
	version           string
	settings          config.Settings
	clients           map[string]*kubernetes.Clientset
	factories         map[string]*kube.FactoryHandle
	metricsClients    map[string]*metricsclient.Clientset
	activeContext     string
	mu                sync.RWMutex
	portForwards      map[string]dto.PortForward
	pfMu              sync.RWMutex
	restConfigs       map[string]*rest.Config
	pfCancels         map[string]context.CancelFunc
	logCancels        map[string]context.CancelFunc
	logSeqs           map[string]uint64
	execCancels       map[string]context.CancelFunc
	execResizeChans   map[string]chan remotecommand.TerminalSize
	streamMu          sync.Mutex
	pluginLoaders     map[string]*plugin.PluginLoader
	removingPluginIDs map[string]bool // tracks plugins being removed to prevent concurrent installs
	// pluginsMu guards pluginLoaders and removingPluginIDs. Lock ordering: never
	// hold pluginsMu while acquiring mu (mu may be taken first and released, then
	// pluginsMu taken separately, but not nested) — mu-guarded helpers like
	// pluginsRootDir() are called from within pluginsMu critical sections
	// (e.g. InstallPlugin), so acquiring mu while already holding pluginsMu is
	// fine, but the reverse (acquiring pluginsMu while already holding mu) would
	// risk lock-order inversion and must be avoided.
	pluginsMu sync.RWMutex
}

// NewApp creates a new App application struct
func NewApp(version string) *App {
	s, _ := config.Load()
	resolveLoginShellPATH(s.ShellPath)
	return &App{
		version:           version,
		settings:          s,
		clients:           make(map[string]*kubernetes.Clientset),
		factories:         make(map[string]*kube.FactoryHandle),
		metricsClients:    make(map[string]*metricsclient.Clientset),
		portForwards:      make(map[string]dto.PortForward),
		restConfigs:       make(map[string]*rest.Config),
		pfCancels:         make(map[string]context.CancelFunc),
		logCancels:        make(map[string]context.CancelFunc),
		logSeqs:           make(map[string]uint64),
		execCancels:       make(map[string]context.CancelFunc),
		execResizeChans:   make(map[string]chan remotecommand.TerminalSize),
		pluginLoaders:     make(map[string]*plugin.PluginLoader),
		removingPluginIDs: make(map[string]bool),
	}
}

// Startup is called when the app starts. The context is saved
// so we can call the runtime methods.
// Wails guarantees Startup completes before DomReady/frontend JS runs, so
// restoreInstalledPlugins must stay synchronous here — moving it into a
// goroutine would let an early GetInstalledPlugin poll race ahead of it and
// see NOT_INSTALLED for an already-installed plugin.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.restoreInstalledPlugins()
	go a.checkForUpdate()
}

// DomReady is called once the frontend DOM is loaded and the native window is visible.
func (a *App) DomReady(_ context.Context) {
	enableFullscreenButton()
}

// Shutdown is called by Wails when the app is quitting (menu Quit, window
// close, ⌘Q). It gracefully terminates every running plugin subprocess so
// they don't leak as orphaned processes — without this, a plugin process
// keeps running (and its lock file keeps pointing at a live PID) after the
// app exits, which a future session can then mistakenly reuse via
// PluginLoader.Launch()'s stale-lock reuse path, even after the plugins
// directory has since changed.
func (a *App) Shutdown(_ context.Context) {
	a.pluginsMu.Lock()
	defer a.pluginsMu.Unlock()
	for id, loader := range a.pluginLoaders {
		if err := loader.Shutdown(); err != nil {
			log.Printf("plugin %q shutdown on app quit failed: %v", id, err)
		}
	}
}

// GetVersion returns the current application version.
func (a *App) GetVersion() string {
	return a.version
}

func (a *App) emitConnectStatus(contextName, message string) {
	wailsruntime.EventsEmit(a.ctx, "connect:status", map[string]string{
		"context": contextName,
		"message": message,
	})
}

// Connect builds (or reuses) a clientset for the given context, probes the API
// server, and marks the context active. The ping runs outside the lock so a
// slow or unreachable cluster never blocks other goroutines reading the cache.
// NewFactoryHandle blocks until every informer's initial LIST has populated
// its cache, so activeContext is only set — and the frontend's first
// List*/Get* calls only unblocked — once listers are warm.
func (a *App) Connect(contextName string) error {
	a.emitConnectStatus(contextName, "Loading cluster configuration...")

	a.mu.RLock()
	cs, exists := a.clients[contextName]
	proxy := a.settings.ClusterProxies[contextName]
	httpProxy := proxy.HttpProxy
	httpsProxy := proxy.HttpsProxy
	kubeconfigPaths := a.settings.KubeconfigPaths
	a.mu.RUnlock()

	var rc *rest.Config
	if !exists {
		a.emitConnectStatus(contextName, "Building API client...")
		var err error
		cs, rc, err = kube.NewClientset(contextName, httpProxy, httpsProxy, kubeconfigPaths)
		if err != nil {
			a.emitConnectStatus(contextName, "Failed to build API client: "+err.Error())
			return err
		}
		a.emitConnectStatus(contextName, "API client ready")
	}

	// Always verify the API server is reachable before marking connected.
	a.emitConnectStatus(contextName, "Verifying API server connectivity...")
	if err := kube.Ping(cs); err != nil {
		a.emitConnectStatus(contextName, "Cannot reach API server: "+err.Error())
		return err
	}
	a.emitConnectStatus(contextName, "API server reachable")

	a.mu.Lock()
	if _, recheck := a.clients[contextName]; !recheck {
		a.clients[contextName] = cs
	}
	if rc != nil {
		a.restConfigs[contextName] = rc
	}

	// Stop any existing factory for this context before creating a new one.
	if old, ok := a.factories[contextName]; ok {
		old.Stop()
	}

	a.emitConnectStatus(contextName, "Starting informers...")
	var forbiddenOnce sync.Map
	h := kube.NewFactoryHandle(cs, func(resource string) {
		if _, loaded := forbiddenOnce.LoadOrStore(resource, true); !loaded {
			wailsruntime.EventsEmit(a.ctx, "resource:forbidden", resource)
		}
	})
	a.factories[contextName] = h
	a.activeContext = contextName

	// Register event handlers for live updates.
	isCtx := func() bool { return a.isActive(contextName) }
	debLeases := config.NewDebouncer(config.DefaultDebounceInterval, a.emitLeases, isCtx)
	debEvents := config.NewDebouncer(config.DefaultDebounceInterval, a.emitEvents, isCtx)
	debEndpoints := config.NewDebouncer(config.DefaultDebounceInterval, a.emitEndpoints, isCtx)
	debEndpointSlices := config.NewDebouncer(config.DefaultDebounceInterval, a.emitEndpointSlices, isCtx)
	debPods := config.NewDebouncer(config.DefaultDebounceInterval, a.emitPods, isCtx)
	debDeployments := config.NewDebouncer(config.DefaultDebounceInterval, a.emitDeployments, isCtx)
	debDaemonSets := config.NewDebouncer(config.DefaultDebounceInterval, a.emitDaemonSets, isCtx)
	debReplicaSets := config.NewDebouncer(config.DefaultDebounceInterval, a.emitReplicaSets, isCtx)
	debStatefulSets := config.NewDebouncer(config.DefaultDebounceInterval, a.emitStatefulSets, isCtx)
	debJobs := config.NewDebouncer(config.DefaultDebounceInterval, a.emitJobs, isCtx)
	debCronJobs := config.NewDebouncer(config.DefaultDebounceInterval, a.emitCronJobs, isCtx)
	debConfigMaps := config.NewDebouncer(config.DefaultDebounceInterval, a.emitConfigMaps, isCtx)
	debSecrets := config.NewDebouncer(config.DefaultDebounceInterval, a.emitSecrets, isCtx)
	debResourceQuotas := config.NewDebouncer(config.DefaultDebounceInterval, a.emitResourceQuotas, isCtx)
	debLimitRanges := config.NewDebouncer(config.DefaultDebounceInterval, a.emitLimitRanges, isCtx)
	debHPAs := config.NewDebouncer(config.DefaultDebounceInterval, a.emitHPAs, isCtx)
	debPodDisruptionBudgets := config.NewDebouncer(config.DefaultDebounceInterval, a.emitPodDisruptionBudgets, isCtx)
	debIngresses := config.NewDebouncer(config.DefaultDebounceInterval, a.emitIngresses, isCtx)
	debNetworkPolicies := config.NewDebouncer(config.DefaultDebounceInterval, a.emitNetworkPolicies, isCtx)
	debIngressClasses := config.NewDebouncer(config.DefaultDebounceInterval, func(_ string) { a.emitIngressClasses() }, isCtx)
	debValidatingWebhookConfigs := config.NewDebouncer(config.DefaultDebounceInterval, func(_ string) { a.emitValidatingWebhookConfigs() }, isCtx)
	debPersistentVolumeClaims := config.NewDebouncer(config.DefaultDebounceInterval, a.emitPersistentVolumeClaims, isCtx)
	debPersistentVolumes := config.NewDebouncer(config.DefaultDebounceInterval, func(_ string) { a.emitPersistentVolumes() }, isCtx)
	debStorageClasses := config.NewDebouncer(config.DefaultDebounceInterval, func(_ string) { a.emitStorageClasses() }, isCtx)
	debServices := config.NewDebouncer(config.DefaultDebounceInterval, a.emitServices, isCtx)
	debNodes := config.NewDebouncer(config.DefaultDebounceInterval, func(_ string) { a.emitNodes() }, isCtx)
	debNamespaces := config.NewDebouncer(config.DefaultDebounceInterval, func(_ string) { a.emitNamespaces() }, isCtx)
	debServiceAccounts := config.NewDebouncer(config.DefaultDebounceInterval, a.emitServiceAccounts, isCtx)
	debClusterRoles := config.NewDebouncer(config.DefaultDebounceInterval, func(_ string) { a.emitClusterRoles() }, isCtx)
	debRoles := config.NewDebouncer(config.DefaultDebounceInterval, a.emitRoles, isCtx)
	debClusterRoleBindings := config.NewDebouncer(config.DefaultDebounceInterval, func(_ string) { a.emitClusterRoleBindings() }, isCtx)
	debRoleBindings := config.NewDebouncer(config.DefaultDebounceInterval, a.emitRoleBindings, isCtx)
	debPriorityClasses := config.NewDebouncer(config.DefaultDebounceInterval, func(_ string) { a.emitPriorityClasses() }, isCtx)

	// Register all debouncers with the factory for lifecycle management
	h.RegisterDebouncer(debLeases)
	h.RegisterDebouncer(debEvents)
	h.RegisterDebouncer(debEndpoints)
	h.RegisterDebouncer(debEndpointSlices)
	h.RegisterDebouncer(debPods)
	h.RegisterDebouncer(debDeployments)
	h.RegisterDebouncer(debDaemonSets)
	h.RegisterDebouncer(debReplicaSets)
	h.RegisterDebouncer(debStatefulSets)
	h.RegisterDebouncer(debJobs)
	h.RegisterDebouncer(debCronJobs)
	h.RegisterDebouncer(debConfigMaps)
	h.RegisterDebouncer(debSecrets)
	h.RegisterDebouncer(debResourceQuotas)
	h.RegisterDebouncer(debLimitRanges)
	h.RegisterDebouncer(debHPAs)
	h.RegisterDebouncer(debPodDisruptionBudgets)
	h.RegisterDebouncer(debIngresses)
	h.RegisterDebouncer(debNetworkPolicies)
	h.RegisterDebouncer(debIngressClasses)
	h.RegisterDebouncer(debValidatingWebhookConfigs)
	h.RegisterDebouncer(debPersistentVolumeClaims)
	h.RegisterDebouncer(debPersistentVolumes)
	h.RegisterDebouncer(debStorageClasses)
	h.RegisterDebouncer(debServices)
	h.RegisterDebouncer(debNodes)
	h.RegisterDebouncer(debNamespaces)
	h.RegisterDebouncer(debServiceAccounts)
	h.RegisterDebouncer(debClusterRoles)
	h.RegisterDebouncer(debRoles)
	h.RegisterDebouncer(debClusterRoleBindings)
	h.RegisterDebouncer(debRoleBindings)
	h.RegisterDebouncer(debPriorityClasses)

	h.Factory.Core().V1().Pods().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debPods.Trigger(nsFromObj(obj))
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debPods.Trigger(nsFromObj(new))
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debPods.Trigger(nsFromObj(obj))
			}
		},
	})
	h.Factory.Apps().V1().Deployments().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debDeployments.Trigger(nsFromObj(obj))
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debDeployments.Trigger(nsFromObj(new))
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debDeployments.Trigger(nsFromObj(obj))
			}
		},
	})
	h.Factory.Apps().V1().DaemonSets().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debDaemonSets.Trigger(nsFromObj(obj))
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debDaemonSets.Trigger(nsFromObj(new))
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debDaemonSets.Trigger(nsFromObj(obj))
			}
		},
	})
	h.Factory.Apps().V1().ReplicaSets().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debReplicaSets.Trigger(nsFromObj(obj))
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debReplicaSets.Trigger(nsFromObj(new))
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debReplicaSets.Trigger(nsFromObj(obj))
			}
		},
	})
	h.Factory.Apps().V1().StatefulSets().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debStatefulSets.Trigger(nsFromObj(obj))
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debStatefulSets.Trigger(nsFromObj(new))
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debStatefulSets.Trigger(nsFromObj(obj))
			}
		},
	})
	h.Factory.Batch().V1().Jobs().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debJobs.Trigger(nsFromObj(obj))
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debJobs.Trigger(nsFromObj(new))
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debJobs.Trigger(nsFromObj(obj))
			}
		},
	})
	h.Factory.Batch().V1().CronJobs().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debCronJobs.Trigger(nsFromObj(obj))
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debCronJobs.Trigger(nsFromObj(new))
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debCronJobs.Trigger(nsFromObj(obj))
			}
		},
	})
	h.Factory.Core().V1().ConfigMaps().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debConfigMaps.Trigger(nsFromObj(obj))
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debConfigMaps.Trigger(nsFromObj(new))
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debConfigMaps.Trigger(nsFromObj(obj))
			}
		},
	})
	h.Factory.Core().V1().Secrets().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debSecrets.Trigger(nsFromObj(obj))
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debSecrets.Trigger(nsFromObj(new))
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debSecrets.Trigger(nsFromObj(obj))
			}
		},
	})
	h.Factory.Core().V1().ResourceQuotas().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debResourceQuotas.Trigger(nsFromObj(obj))
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debResourceQuotas.Trigger(nsFromObj(new))
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debResourceQuotas.Trigger(nsFromObj(obj))
			}
		},
	})
	h.Factory.Core().V1().LimitRanges().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debLimitRanges.Trigger(nsFromObj(obj))
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debLimitRanges.Trigger(nsFromObj(new))
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debLimitRanges.Trigger(nsFromObj(obj))
			}
		},
	})
	h.Factory.Autoscaling().V2().HorizontalPodAutoscalers().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debHPAs.Trigger(nsFromObj(obj))
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debHPAs.Trigger(nsFromObj(new))
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debHPAs.Trigger(nsFromObj(obj))
			}
		},
	})
	h.Factory.Policy().V1().PodDisruptionBudgets().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debPodDisruptionBudgets.Trigger(nsFromObj(obj))
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debPodDisruptionBudgets.Trigger(nsFromObj(new))
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debPodDisruptionBudgets.Trigger(nsFromObj(obj))
			}
		},
	})
	h.Factory.Networking().V1().Ingresses().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debIngresses.Trigger(nsFromObj(obj))
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debIngresses.Trigger(nsFromObj(new))
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debIngresses.Trigger(nsFromObj(obj))
			}
		},
	})
	h.Factory.Networking().V1().NetworkPolicies().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debNetworkPolicies.Trigger(nsFromObj(obj))
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debNetworkPolicies.Trigger(nsFromObj(new))
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debNetworkPolicies.Trigger(nsFromObj(obj))
			}
		},
	})
	h.Factory.Networking().V1().IngressClasses().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debIngressClasses.Trigger("")
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debIngressClasses.Trigger("")
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debIngressClasses.Trigger("")
			}
		},
	})
	h.Factory.Admissionregistration().V1().ValidatingWebhookConfigurations().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debValidatingWebhookConfigs.Trigger("")
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debValidatingWebhookConfigs.Trigger("")
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debValidatingWebhookConfigs.Trigger("")
			}
		},
	})
	h.Factory.Core().V1().PersistentVolumeClaims().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debPersistentVolumeClaims.Trigger(nsFromObj(obj))
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debPersistentVolumeClaims.Trigger(nsFromObj(new))
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debPersistentVolumeClaims.Trigger(nsFromObj(obj))
			}
		},
	})
	h.Factory.Core().V1().PersistentVolumes().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debPersistentVolumes.Trigger("")
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debPersistentVolumes.Trigger("")
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debPersistentVolumes.Trigger("")
			}
		},
	})
	h.Factory.Storage().V1().StorageClasses().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debStorageClasses.Trigger("")
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debStorageClasses.Trigger("")
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debStorageClasses.Trigger("")
			}
		},
	})
	h.Factory.Core().V1().Endpoints().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debEndpoints.Trigger(nsFromObj(obj))
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debEndpoints.Trigger(nsFromObj(new))
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debEndpoints.Trigger(nsFromObj(obj))
			}
		},
	})
	h.Factory.Core().V1().Services().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debServices.Trigger(nsFromObj(obj))
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debServices.Trigger(nsFromObj(new))
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debServices.Trigger(nsFromObj(obj))
			}
		},
	})
	h.Factory.Core().V1().Nodes().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debNodes.Trigger("")
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debNodes.Trigger("")
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debNodes.Trigger("")
			}
		},
	})
	h.Factory.Core().V1().Namespaces().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debNamespaces.Trigger("")
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debNamespaces.Trigger("")
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debNamespaces.Trigger("")
			}
		},
	})
	h.Factory.Core().V1().ServiceAccounts().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debServiceAccounts.Trigger(nsFromObj(obj))
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debServiceAccounts.Trigger(nsFromObj(new))
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debServiceAccounts.Trigger(nsFromObj(obj))
			}
		},
	})
	h.Factory.Rbac().V1().ClusterRoles().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debClusterRoles.Trigger("")
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debClusterRoles.Trigger("")
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debClusterRoles.Trigger("")
			}
		},
	})
	h.Factory.Rbac().V1().Roles().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debRoles.Trigger(nsFromObj(obj))
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debRoles.Trigger(nsFromObj(new))
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debRoles.Trigger(nsFromObj(obj))
			}
		},
	})
	h.Factory.Rbac().V1().ClusterRoleBindings().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debClusterRoleBindings.Trigger("")
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debClusterRoleBindings.Trigger("")
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debClusterRoleBindings.Trigger("")
			}
		},
	})
	h.Factory.Rbac().V1().RoleBindings().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debRoleBindings.Trigger(nsFromObj(obj))
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debRoleBindings.Trigger(nsFromObj(new))
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debRoleBindings.Trigger(nsFromObj(obj))
			}
		},
	})
	h.Factory.Scheduling().V1().PriorityClasses().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debPriorityClasses.Trigger("")
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debPriorityClasses.Trigger("")
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debPriorityClasses.Trigger("")
			}
		},
	})
	h.Factory.Coordination().V1().Leases().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debLeases.Trigger(nsFromObj(obj))
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debLeases.Trigger(nsFromObj(new))
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debLeases.Trigger(nsFromObj(obj))
			}
		},
	})
	h.Factory.Core().V1().Events().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debEvents.Trigger(nsFromObj(obj))
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debEvents.Trigger(nsFromObj(new))
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debEvents.Trigger(nsFromObj(obj))
			}
		},
	})
	h.Factory.Discovery().V1().EndpointSlices().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if a.isActive(contextName) {
				debEndpointSlices.Trigger(nsFromObj(obj))
			}
		},
		UpdateFunc: func(old, new any) {
			if a.isActive(contextName) {
				debEndpointSlices.Trigger(nsFromObj(new))
			}
		},
		DeleteFunc: func(obj any) {
			if a.isActive(contextName) {
				debEndpointSlices.Trigger(nsFromObj(obj))
			}
		},
	})

	// Create (or replace) metrics client for this context.
	if mc, err := kube.NewMetricsClientForContext(contextName, httpProxy, httpsProxy, a.settings.KubeconfigPaths); err == nil {
		a.metricsClients[contextName] = mc
	}
	a.mu.Unlock()
	a.emitConnectStatus(contextName, "Connected")
	go a.prewarmRestoredPlugins(contextName)
	return nil
}

// IsResourceForbidden reports whether the given resource is known to be forbidden
// (403) in the currently active cluster context.
func (a *App) IsResourceForbidden(resource string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	h, ok := a.factories[a.activeContext]
	if !ok {
		return false
	}
	return h.IsForbidden(resource)
}
