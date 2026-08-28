# litelens-core

Shared Go module providing the plugin-host contract: protobuf/gRPC pub/sub service definitions, DTOs, kubeconfig loading utilities, and a plugin-side gRPC client for host↔plugin event sync.

## Purpose

Plugins (`litelens-plugins`) previously hand-copied these packages and kept them manually in sync with the host. This module lets plugins import them as a real versioned dependency instead, ensuring correctness and removing that maintenance burden.

## Contents

- **`pb/`** — protobuf definitions (`plugin.proto`) and generated Go code (`plugin.pb.go`, `plugin_grpc.pb.go`). Defines the `Plugin` gRPC service's pub/sub contract between host and plugin subprocesses: `Subscribe(topic) → stream PubSubMessage` and `Publish(topic, payload)`. The host publishes on topics like `cluster.context`/`namespaces.active`; plugins may only publish under their own `plugins.<pluginID>.*` prefix.
- **`kube/dto/`** — data transfer objects (36 DTO types mirroring the host's internal schema). Defines the shape of K8s resource data passed between host and plugin frontends. All timestamp fields use `string` (not `time.Time`), matching Wails binding conventions.
- **`kube/`** — `LoadingRules(paths []string)`, which returns a client-go `ClientConfigLoadingRules` for the given kubeconfig paths (or the standard `KUBECONFIG` env var / `~/.kube/config` when none are supplied).
- **`async/`** — the plugin subprocess's outbound gRPC client to the host: dialing the host's gRPC server, authenticating with a bearer token (`NewAuthInterceptors`), and subscribing to/publishing on pub/sub topics. Includes `EventTopic` constants (`EventTopicClusterContext`, `EventTopicNamespacesActive`) and their event payload types (`ClusterContextEvent`, `ActiveNamespacesEvent`), a generic `EventRoute`/`NewRoute` for wiring a typed handler to a topic with built-in reconnect/backoff, and `Emit` for a plugin to publish its own events back to the host.
- **`util/`** — host-adjacent helpers not specific to any other package; currently `ReadAuthTokenFromStdin`, used by plugin subprocesses to read their per-launch gRPC auth token.
- **`frontend/`** (`@litelens/core`, pnpm workspace package) — TypeScript/React counterpart of this contract. See `frontend/README.md` for its API surface (`clusterWideAPI`, `appWideAPI`) and how plugin bundles resolve it against the host's own module instances at runtime.

## Use by plugins

```go
require github.com/litelensapp/litelens/packages/core vX.Y.Z
```

The samples below are taken from the `helm` plugin (`litelens-plugins/plugins/helm/internal`), the reference consumer of this module.

### `util` — read the per-launch auth token

`internal/main.go` reads the gRPC bearer token off stdin before dialing the host, then reuses it for every subsequent gRPC call:

```go
import "github.com/litelensapp/litelens/packages/core/util"

token, err := util.ReadAuthTokenFromStdin()
if err != nil {
    fmt.Fprintf(os.Stderr, "error: read auth token from stdin: %v\n", err)
    os.Exit(1)
}
authToken = token
```

### `async` — dial the host, subscribe to its events, emit your own

`internal/main.go` opens one shared `GrpcClient` for the plugin process:

```go
import coreasync "github.com/litelensapp/litelens/packages/core/async"

client := &coreasync.GrpcClient{}
if err := client.Dial(addr, authToken); err != nil {
    fmt.Fprintf(os.Stderr, "warning: failed to connect to host gRPC server: %v\n", err)
} else {
    defer client.Close()
    hostClient = client
}
```

`internal/adapters/presentations/async/dispatcher.go` wires the plugin's own handlers onto the shared `EventRoute`/`EventTopic` machinery instead of hand-rolling watch loops:

```go
import "github.com/litelensapp/litelens/packages/core/async"

func NewEventDispatcher(receiver async.EventReceiver) *EventDispatcher {
    h := NewHandler(receiver)
    return &EventDispatcher{
        routes: []async.EventRoute{
            async.NewRoute(string(async.EventTopicClusterContext), h.handleClusterContext, async.DeserializeClusterContext),
            async.NewRoute(string(async.EventTopicNamespacesActive), h.handleActiveNamespaces, async.DeserializeActiveNamespaces),
        },
    }
}
```

And `Emit` lets the plugin publish its own events back to the host, e.g. after a helm release changes (`internal/main.go`):

```go
eventEmitFn := func(ctx context.Context, eventName string, data any) {
    if hostClient != nil {
        hostClient.Emit(ctx, eventName, "helm", data)
    }
}
```

### `kube` — build a REST client getter against the active context

`internal/applications/helm/helm.go` uses `LoadingRules` to scope helm's `RESTClientGetter` to the cluster provider's active kubeconfig paths, rather than re-deriving default loading rules itself:

```go
import "github.com/litelensapp/litelens/packages/core/kube"

getter := s.getterFactory.NewRESTClientGetter(rc, kube.LoadingRules(kubeconfigPaths), &clientcmd.ConfigOverrides{CurrentContext: activeCtx})
```

### `pb` — consumed indirectly through `async`

The helm plugin never imports `pb` directly; `async.GrpcClient`/`async.EventRoute` wrap the generated `pb.PluginClient` (`Subscribe`/`Publish`) so plugin code only deals with typed events and topics. Import `pb` directly only if you need the raw gRPC client/stream types `async` doesn't expose.

### `kube/dto` — shape shared with generic K8s resource views

The helm plugin doesn't consume `kube/dto` — it works with helm release objects, not raw K8s resources. A plugin whose views render built-in resource types (pods, deployments, etc.) alongside its own would import the matching DTO instead of hand-rolling one:

```go
import "github.com/litelensapp/litelens/packages/core/kube/dto"

var pods []dto.Pod
```

See `CONTRIBUTING.md` for versioning scheme, source-of-truth history, and CI details.
