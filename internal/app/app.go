package app

import (
	"context"
	"log"
	"sync"
	"time"

	hostgrpc "github.com/litelensapp/litelens/internal/api/grpc"
	"github.com/litelensapp/litelens/internal/config"
	"github.com/litelensapp/litelens/internal/kube"
	"github.com/litelensapp/litelens/internal/lib/debouncer"
	"github.com/litelensapp/litelens/internal/plugin"
	"github.com/litelensapp/litelens/internal/updater"
	"github.com/litelensapp/litelens/packages/core/kube/dto"
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
	ctx                   context.Context
	version               string
	appSizeBytes          int64         // cached at startup, read-only afterward
	installSource         string        // set once during Startup; guarded by mu
	installSourceReady    chan struct{} // closed once installSource has been detected; see GetInstallSource
	settings              config.Settings
	clients               map[string]*kubernetes.Clientset
	factories             map[string]*kube.FactoryHandle
	metricsClients        map[string]*metricsclient.Clientset
	activeContext         string
	activeContextSeq      int64    // guarded by mu; monotonic per Connect call, see Connect
	activeNamespaces      []string // guarded by mu; empty/nil = all namespaces
	activeNamespacesSeq   int64    // guarded by mu; see SetActiveNamespaces
	mu                    sync.RWMutex
	lastUpdateCheckResult *UpdateCheckResult // guarded by mu; caches the last successful update check
	portForwards          map[string]dto.PortForward
	pfMu                  sync.RWMutex
	restConfigs           map[string]*rest.Config
	pfCancels             map[string]context.CancelFunc
	logCancels            map[string]context.CancelFunc
	logSeqs               map[string]uint64
	execCancels           map[string]context.CancelFunc
	execResizeChans       map[string]chan remotecommand.TerminalSize
	streamMu              sync.Mutex
	pluginLoaders         map[string]*plugin.PluginLoader
	removingPluginIDs     map[string]bool // tracks plugins being removed to prevent concurrent installs
	// pluginsMu guards pluginLoaders and removingPluginIDs. Lock ordering: never
	// hold pluginsMu while acquiring mu (mu may be taken first and released, then
	// pluginsMu taken separately, but not nested) — mu-guarded helpers like
	// pluginsRootDir() are called from within pluginsMu critical sections
	// (e.g. InstallPlugin), so acquiring mu while already holding pluginsMu is
	// fine, but the reverse (acquiring pluginsMu while already holding mu) would
	// risk lock-order inversion and must be avoided.
	pluginsMu     sync.RWMutex
	grpcServerCfg *hostgrpc.GRPCServerConfig
}

// NewApp creates a new App application struct
func NewApp(version string) *App {
	s, _ := config.Load()
	// resolveLoginShellPATH shells out to the user's login shell (up to a 5s
	// timeout on macOS) and NewApp runs before wails.Run even starts, so doing
	// this synchronously blocks the whole process before a window can appear.
	// It's best-effort — Setenv is skipped on failure/timeout and the app keeps
	// the original PATH — and only affects exec-credential-plugin lookups
	// (aws/gcloud) during a later Connect(), which needs user interaction and
	// so has ample time for this to finish first.
	// Skipped in tests ("test" is the sentinel version every unit test passes
	// to NewApp) — otherwise every test constructing an App piles up real
	// subprocess spawns with no benefit, which is slow and flaky under load.
	if version != "test" {
		go resolveLoginShellPATH(s.ShellPath)
	}
	return &App{
		version:            version,
		settings:           s,
		clients:            make(map[string]*kubernetes.Clientset),
		factories:          make(map[string]*kube.FactoryHandle),
		metricsClients:     make(map[string]*metricsclient.Clientset),
		portForwards:       make(map[string]dto.PortForward),
		restConfigs:        make(map[string]*rest.Config),
		pfCancels:          make(map[string]context.CancelFunc),
		logCancels:         make(map[string]context.CancelFunc),
		logSeqs:            make(map[string]uint64),
		execCancels:        make(map[string]context.CancelFunc),
		execResizeChans:    make(map[string]chan remotecommand.TerminalSize),
		pluginLoaders:      make(map[string]*plugin.PluginLoader),
		removingPluginIDs:  make(map[string]bool),
		installSourceReady: make(chan struct{}),
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
	a.appSizeBytes = getAppSizeBytes()
	a.detectInstallSource()
	a.restoreInstalledPlugins()
	a.runServer()
	go a.checkForUpdate(3)
}

// runServer starts the plugin cluster context gRPC server. This is required
// for cluster sync to work with plugins; if it fails, plugins will not
// receive context changes.
func (a *App) runServer() {
	eventEmitter := func(payload map[string]any) {
		wailsruntime.EventsEmit(a.ctx, "plugin:event", payload)
	}
	grpcCfg, err := hostgrpc.NewGRPCServerConfig(eventEmitter)
	if err != nil {
		// Log with ERROR level since plugin sync is essential infrastructure
		log.Printf("ERROR: failed to start plugin gRPC server: %v (plugins will not receive cluster context changes)", err)
	} else {
		a.grpcServerCfg = grpcCfg
		log.Printf("plugin cluster context gRPC server started on port %d", grpcCfg.Port())
	}
}

// detectInstallSource blocks up to ~2s (brew's timeout), but eliminates the
// race entirely and completes before frontend JS runs.
func (a *App) detectInstallSource() {
	source := updater.DetectInstallSource()
	a.mu.Lock()
	a.installSource = source
	a.mu.Unlock()
	close(a.installSourceReady)
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
	// Kill plugin processes before stopping the gRPC server. Each plugin
	// holds a long-lived ClusterContextWatch stream open for its entire
	// lifetime (see internal/api/grpc/server.go), and grpcServerCfg.Stop() calls
	// GracefulStop(), which blocks until every active RPC finishes. Stopping
	// the server first would deadlock: GracefulStop waits on a stream that
	// only closes when the plugin dies, but the plugin is only killed below.
	a.pluginsMu.Lock()
	for id, loader := range a.pluginLoaders {
		if err := loader.Shutdown(); err != nil {
			log.Printf("plugin %q shutdown on app quit failed: %v", id, err)
		}
	}
	a.pluginsMu.Unlock()

	if a.grpcServerCfg != nil {
		a.grpcServerCfg.Stop()
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
//
// seq is a value the frontend increments synchronously on every call (before
// the async IPC dispatch), same pattern as SetActiveNamespaces: rapid
// back-and-forth context switches launch multiple Connect calls concurrently,
// and since each does slow network/informer-sync work outside any lock that
// serializes it against the others, they can complete in an order that
// doesn't match the order the user clicked them in. Without seq, whichever
// call happens to finish last wins and silently overwrites activeContext with
// a stale (no-longer-selected) context — every List*/Get* call and the
// namespace filter would then silently keep operating on the wrong cluster
// while the UI shows the one the user actually selected.
func (a *App) Connect(contextName string, seq int64) error {
	if !a.tryClaimConnectSeq(seq) {
		return nil
	}

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

	// A newer Connect call may have already become active while this one was
	// blocked building its client and syncing informers above — don't let a
	// stale call clobber it. Stop the just-synced factory rather than leaking it.
	if seq < a.activeContextSeq {
		a.mu.Unlock()
		h.Stop()
		return nil
	}
	a.factories[contextName] = h
	a.activeContext = contextName
	// Restore this context's persisted default namespace filter (rather than
	// resetting to nil/"all namespaces") and push it to plugins unconditionally
	// on every Connect — not just on a genuine context switch. A plain
	// reconnect to the already-active context (e.g. a page reload while the
	// host process keeps running) previously fell into the "no context
	// change, no push" branch, leaving a running plugin's synced namespace
	// filter stale (e.g. left over from an earlier in-session selection) with
	// nothing to correct it: the frontend's MainLayout does asynchronously
	// re-push its restored defaults on mount, but any plugin business call
	// racing ahead of that IPC round trip would be served — and its result
	// cached indefinitely — against the stale filter. Pushing the correct
	// restored value here, synchronously within Connect() and before it
	// returns to the frontend, closes that window instead of relying on the
	// slower async re-push to win the race.
	restoredNamespaces := a.restoredNamespacesForContextLocked(contextName)
	a.activeNamespaces = restoredNamespaces

	// Push cluster context to all running plugins with HTTP backends.
	// Phase 2 design decision: "The host pushes POST on every cluster switch."
	// Unlock before resolving the kubeconfig path and pushing: GetContextKubeconfigPath
	// takes its own RLock on a.mu, and a.mu is not reentrant, so calling it while still
	// holding the write lock acquired above would deadlock.
	a.mu.Unlock()
	a.emitActiveContextToPlugins(contextName)
	a.emitActiveNamespacesToPlugins(restoredNamespaces)
	a.mu.Lock()

	// Register event handlers for live updates.
	isCtx := func() bool { return a.isActive(contextName) }
	debLeases := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitLeases() }, isCtx)
	debEvents := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitEvents() }, isCtx)
	debEndpoints := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitEndpoints() }, isCtx)
	debEndpointSlices := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitEndpointSlices() }, isCtx)
	debPods := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitPods() }, isCtx)
	debDeployments := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitDeployments() }, isCtx)
	debDaemonSets := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitDaemonSets() }, isCtx)
	debReplicaSets := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitReplicaSets() }, isCtx)
	debStatefulSets := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitStatefulSets() }, isCtx)
	debJobs := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitJobs() }, isCtx)
	debCronJobs := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitCronJobs() }, isCtx)
	debConfigMaps := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitConfigMaps() }, isCtx)
	debSecrets := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitSecrets() }, isCtx)
	debResourceQuotas := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitResourceQuotas() }, isCtx)
	debLimitRanges := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitLimitRanges() }, isCtx)
	debHPAs := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitHPAs() }, isCtx)
	debPodDisruptionBudgets := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitPodDisruptionBudgets() }, isCtx)
	debIngresses := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitIngresses() }, isCtx)
	debNetworkPolicies := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitNetworkPolicies() }, isCtx)
	debIngressClasses := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitIngressClasses() }, isCtx)
	debValidatingWebhookConfigs := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitValidatingWebhookConfigs() }, isCtx)
	debPersistentVolumeClaims := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitPersistentVolumeClaims() }, isCtx)
	debPersistentVolumes := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitPersistentVolumes() }, isCtx)
	debStorageClasses := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitStorageClasses() }, isCtx)
	debServices := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitServices() }, isCtx)
	debNodes := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitNodes() }, isCtx)
	debNamespaces := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitNamespaces() }, isCtx)
	debServiceAccounts := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitServiceAccounts() }, isCtx)
	debClusterRoles := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitClusterRoles() }, isCtx)
	debRoles := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitRoles() }, isCtx)
	debClusterRoleBindings := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitClusterRoleBindings() }, isCtx)
	debRoleBindings := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitRoleBindings() }, isCtx)
	debPriorityClasses := debouncer.NewDebouncer(debouncer.DefaultDebounceInterval, func(_ string) { a.emitPriorityClasses() }, isCtx)

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
