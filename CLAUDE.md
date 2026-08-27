# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

Litelens — a desktop Kubernetes dashboard built with [Wails v2](https://wails.io) (Go backend + React/TypeScript frontend, native webview shell, no Electron). Root module `github.com/litelensapp/litelens`. pnpm workspace with `frontend` and `design-system` as packages. Plugins are separate Go modules, launched as subprocesses, driven over HTTP (business logic) plus a native gRPC pub/sub channel (host↔plugin push events) — never compiled into the main binary. No plugins currently ship; the plugin host (`internal/plugin`, marketplace UI, tray registry) is generic infrastructure for plugins installed at runtime.

## Commands

### Dev

```bash
wails dev              # hot-reload desktop app (Go + Vite)
pnpm dev                # frontend only (vite), for pure UI work
```

### Build

```bash
pnpm build:app          # full app build (bash scripts/build.sh; wraps `wails build`, macOS ad-hoc codesigns)
pnpm build:ds            # build @litelens/design-system package (needed before frontend build)
pnpm build:app:fe        # build:ds + frontend build only (no Wails binary)
```

### Lint / format / vet (run after every change, before reporting done)

```bash
pnpm format              # prettier --write across ts/tsx/js/json/css/md/yml
pnpm lint:fe              # eslint frontend/src design-system/src
pnpm lint:be              # go vet + staticcheck for ./internal/... and the packages/core module
```

### Tests

```bash
pnpm test:be              # go test -race -v ./internal/...
pnpm test:be:coverage
pnpm test:fe               # vitest run (frontend)
pnpm test:fe:coverage
pnpm test:ds                # design-system vitest
```

Single Go test: `go test -race -run TestName ./internal/app/...`
Single frontend test: `pnpm --filter litelens-frontend exec vitest run path/to/File.test.tsx`

### Local test cluster

```bash
minikube addons enable metrics-server
# or, for Docker Desktop's docker-desktop context, patch metrics-server with --kubelet-insecure-tls (see CONTRIBUTING.md)
```

## Architecture

### Backend (`internal/`)

- **`internal/app`** — one file per Wails-bound method group (`pod.go`, `deployment.go`, `node.go`, …), plus `app.go` (the `App` struct, `Connect`/`Disconnect`, event wiring). Every exported method on `App` is auto-bound to TypeScript by Wails (`main.go` binds only `*App` — `bindList := []any{a}`; nothing else is exposed directly).
- **`internal/kube`** — client/informer plumbing: per-context `*kubernetes.Clientset` cache (mutex-guarded map), `clientcmd` loading rules (respects `$KUBECONFIG`), `SharedInformerFactory` handles, node/pod metrics fetch.
- **`internal/kube/resources`** — per-resource-type `ListXxx`/`toXxx` conversion logic (raw K8s object → DTO).
- **`packages/core/kube/dto`** — leaf package, pure DTO type definitions (no imports of `kube`/`app`), imported by `internal/` as a Go module dependency (see Plugin architecture below). One type per K8s resource kind; **fields must use `string` for timestamps, never `time.Time`** — Wails can't generate TS bindings for it. `internal/dto` no longer exists — it was removed once every importer moved to `packages/core/kube/dto`.
- **`internal/plugin`** — generic plugin host: discovery, download/verify, process lifecycle (`loader.go`). Business-logic calls to a plugin go over plain HTTP to the plugin's own local server (`App.GetPluginBackendAddr(pluginID)` resolves the `127.0.0.1:<port>` address, relaunching/health-checking as needed); push-style host↔plugin events (cluster context, namespace filter changes) go over a native gRPC pub/sub channel instead (types from `packages/core/pb`) — see Plugin architecture below.
- **`internal/updater`** — self-update (checks GitHub releases, downloads, swaps binary); split into `authenticated_updater.go` (private repo, token-based) and `unauthenticated_updater.go` (public repo, driven by a `manifest.json` release artifact — the single source of truth for per-OS/arch download filenames + SHA256, see `manifest.go`).
- **`internal/config`** — app config (`~/.litelens/settings.json`, via `internal/storage`).
- **`internal/storage`** — resolves `~/.litelens`, the single on-disk directory for all persistent app data (settings, installed plugins). Leaf package (no internal deps); `internal/config` and `internal/app` both depend on it.

**Package dependency direction** (no cycles): `packages/core/kube/dto` ← `kube/resources` ← `kube` ← `app`; `store` ← `config` ← `app`, `store` ← `app`. `packages/core/kube/dto` and `store` are the leaves.

### Core backend patterns

- **Watch, don't poll.** `SharedInformerFactory` caches K8s objects; `lister.Xxxs(ns).List(...)` reads from the in-memory cache, zero API round trips. `kube.NewFactoryHandle` blocks (≤30s) until every informer's initial LIST has synced before `Connect()` unblocks the frontend's first queries.
- **Detail reads** use `Get<Resource>ByName(namespace, name)` → `lister.Xxxs(ns).Get(name)`, never `ListXxx().find(...)`.
- **Push updates via Wails events**, not polling: Go emits (`runtime.EventsEmit(a.ctx, "pods:update", pods)`), the frontend's per-resource `useXxxUpdateEvents` hook subscribes and the owning data-access hook merges the pushed payload over its `useQuery` result locally (no global cache-write hook).
- **Forbidden-resource (403) handling**: informer 403s emit `resource:forbidden` once per resource per connection; frontend's `useCatchForbiddenResources` toasts when the user is on/navigates to that view.
- Adding a new Go method requires manually adding it to `frontend/src/api/resources.ts`'s re-export list (Wails bindings are auto-generated into `frontend/wailsjs/`, but that file is hand-maintained) — same for any new DTO type needing hand-added entries in `frontend/wailsjs/go/models.ts` if you're not running `wails dev`/`wails generate module` to regenerate.
- `wails generate` mutates `go.mod` — never run it as a shortcut for other tasks.

### Plugin architecture

Plugins are **fully standalone Go modules** (e.g. in `litelens-plugins` repo, own `go.mod` under `github.com/litelensapp/litelens-plugins/plugins/<name>`). Each plugin ships its own frontend bundle too (`plugins/<name>/frontend`, its own workspace package). The host (`internal/`) and all plugins share a common contract via the `packages/core` nested Go module.

**Go module structure:**

- **`packages/core/` (nested module):** a sibling to `internal/`, with its own `go.mod` (`github.com/litelensapp/litelens/packages/core`), versioned via git tags shaped `packages/core/vX.Y.Z` — first published as `packages/core/v1.7.0`. Exports the plugin-host contract: protobuf service definitions (`packages/core/pb/`), data transfer objects (`packages/core/kube/dto/` — leaf package, no heavy deps), kubeconfig loading utilities (`packages/core/kube.LoadingRules` only, not full client/informer machinery), and shared host↔plugin pub/sub constants (`packages/core/util.EventTopic*` — was `packages/core/pluginsdk` before the 2026-08-27 rename).
- **Host import:** `internal/` imports `packages/core/{pb,kube,kube/dto,util}` via `go.mod`'s `require github.com/litelensapp/litelens/packages/core vX.Y.Z`, but a permanent `replace github.com/litelensapp/litelens/packages/core => ./packages/core` directive makes the host always build against local source — `packages/core` is the host's own extension surface (same pairing as `@litelens/core` resolved via frontend `workspace:*`), so host and core always ship from the same commit and the `require` version pin doesn't gate the host build. No root `go.work` file (the bare `replace` line is enough). This replaces parallel hand-maintained copies in `internal/`.
- **Plugin import:** plugins depend on `packages/core/` as a normal versioned Go dependency (e.g. `go get github.com/litelensapp/litelens/packages/core@vX.Y.Z` — the module path already includes `packages/core`, so the `@` suffix is just `vX.Y.Z`, not `packages/core/vX.Y.Z`), avoiding the old manual-sync cost of hand-copying DTO and proto files.

**Frontend packages:**

- **`@litelens/core` (pnpm workspace package):** provisioned at `packages/core/frontend/`, exports host-level React hooks (starting with `useResourceLinks`) for plugin frontends to import. Resolved via import map (`frontend/index.html`) and vendor shims (`frontend/public/vendor/litelens/core.js`), same pattern as `@litelens/design-system`. Versioned alongside the host app, not as a stable long-term SDK. See `packages/core/README.md` for rationale and versioning.

**Plugin subprocess model:**

- Plugin binary is launched as a subprocess by `internal/plugin`, emits a one-line JSON `READY` handshake (`httpPort`, plus pid/version) on stdout, then serves its own local HTTP server for business-logic calls.
- The main app still manages process lifecycle (spawn, health checks, restart detection). Host↔plugin push events use a single native gRPC pub/sub service (`packages/core/pb`'s `Plugin` service: `Subscribe(topic) → stream PubSubMessage` / `Publish(topic, payload)`), hosted by `internal/api/grpc` (`hostgrpc.GRPCServerConfig`) — the host publishes on topics like `cluster.context`/`namespaces.active` (`packages/core/util.EventTopic*`), plugins may only publish under their own `plugins.<pluginID>.*` prefix. Plugins authenticate to the host's gRPC server with a per-launch token (`internal/plugin`'s `TokenManager`); the older per-purpose unary RPCs (`GetCapabilities`/`SetClusterContext`/`Invoke`/separate watch streams) have been fully replaced by this generic pub/sub contract.
- Plugin frontend bundles are loaded dynamically and resolve `react`/`react-dom`/`@litelens/design-system`/`@litelens/core`/`@tanstack/react-query` as bare specifiers against the host's own module instances via import map + vendor shims — avoids duplicate instances that would break React Context or query client isolation.
- Plugins are installed at runtime from a marketplace (GitHub release manifests); nothing plugin-specific is compiled into `main.go`.

### Frontend (`frontend/src`)

- **`app/`** — all feature code. `app/clusters/modules/<group>/<resource>/` holds one folder per K8s resource (e.g. `workloads/pods`, `configs/secrets`, `networks/services`), each split into `components/`, `hooks/data-access/`, `hooks/data-mutation/`, `api/`. `app/clusters/shared/` holds cross-resource hooks/components/trays. `app/marketplace`, `app/settings`, `app/updater`, `app/about` are top-level non-cluster modules.
- **`design-system/`** (repo-root package, `@litelens/design-system`) — publishable UI kit. `atoms/` = shadcn-generated primitives wrapping `@base-ui/react` (NOT Radix — `TooltipTrigger` etc. use the `render` prop, not `asChild`). `components/` = composite/business components (buttons, drawers, modals, tables, toasts). Never import `design-system` → `app` except the one sanctioned exception (unified tray family registry, injected via props).
- **Per-cluster query isolation**: every TanStack Query hook's key includes `context` (cluster name); switching clusters calls `queryClient.removeQueries` and `MainLayout` remounts (`key={activeContext}`).
- **Code splitting**: every resource view + heavy widget (xterm, detail drawers) is `React.lazy`-loaded.
- All resource views follow one shape: header (title, count, namespace select, search) → `<Table>` → empty-state row.

### Unified tray system

Two built-in tray "families" (modification/YAML-edit, pod logs+exec) share one bottom-tray shell, and plugin-owned families can be merged in at runtime (see `usePluginTrayRegistry`) (`UnifiedTrayProvider`/`UnifiedTrayShell`, mounted once in `MainLayout.tsx`) via a discriminated-union tab type and a family→content-component registry. Open a tab from anywhere with `useUnifiedTray().openTab(family, params)`. To add a new resource's modification tray: add the kind to `ModificationResourceKind`, create `<Resource>ModificationTray.tsx` (mirror `NamespaceModificationTray.tsx`), register it in `modificationTrayRegistry.tsx` — no changes needed to the shell itself. See `.claude/memory/unified_tray_architecture.md` and `.claude/memory/modification_tray_architecture.md` for full detail.

### DTO/detail-drawer conventions

Each resource has a `dto.<Type>` (Go) and matching TS interface. List views use lean fields; detail drawers add `CreatedAt`, `Labels`, `Annotations`, `ManagedFields`, etc. Detail drawers render via `ResourceDetailDrawer` with an `isLoading`/empty two-gate pattern. Cross-resource references (Namespace, ClusterRole, owning workload, etc.) always render as `<ResourceLink onClick={...}>` wired to `MainLayoutContext`'s `onToggle<Resource>Detail`, never plain text.

## Project-specific conventions

- Imports: files within the same top-level dir (`design-system`, or a given `app/...` module) use **relative imports**, not the `@/...` alias — the alias is reserved for cross-top-level imports.
- Full UI conventions (shadcn-first rule, status badge variants, toast helpers, shared CTA button components, icon-button `aria-label` rule) are in `.claude/memory/component_guidelines.md` — check it before hand-rolling any of these.

## Memory

`.claude/memory/` contains extensive project memory (architecture decisions, file structure, gotchas, feature history) auto-maintained across sessions — check `.claude/memory/MEMORY.md` for the index when you need history/rationale beyond what's summarized here.
