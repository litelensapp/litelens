# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

Litelens — a desktop Kubernetes dashboard built with [Wails v2](https://wails.io) (Go backend + React/TypeScript frontend, native webview shell, no Electron). Root module `github.com/litelensapp/litelens`. pnpm workspace with `frontend` and `design-system` as packages. Plugins are separate Go modules, launched as subprocesses and driven over gRPC — never compiled into the main binary. No plugins currently ship; the plugin host (`internal/plugin`, marketplace UI, tray registry) is generic infrastructure for plugins installed at runtime.

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
- **`internal/dto`** — leaf package, pure DTO type definitions (no imports of `kube`/`app`). One type per K8s resource kind; **fields must use `string` for timestamps, never `time.Time`** — Wails can't generate TS bindings for it.
- **`internal/plugin`** — generic plugin host: discovery, download/verify, process lifecycle (`loader.go`), gRPC handshake. Talks to plugin subprocesses ONLY through `Invoke(pluginID, method, payloadJSON)` — see Plugin architecture below.
- **`internal/updater`** — self-update (checks GitHub releases, downloads, swaps binary); split into `authenticated_updater.go` (private repo, token-based) and `unauthenticated_updater.go` (public repo, driven by a `manifest.json` release artifact — the single source of truth for per-OS/arch download filenames + SHA256, see `manifest.go`).
- **`internal/config`** — app config (`~/.litelens/settings.json`, via `internal/storage`).
- **`internal/storage`** — resolves `~/.litelens`, the single on-disk directory for all persistent app data (settings, installed plugins). Leaf package (no internal deps); `internal/config` and `internal/app` both depend on it.

**Package dependency direction** (no cycles): `dto` ← `kube/resources` ← `kube` ← `app`; `store` ← `config` ← `app`, `store` ← `app`. `dto` and `store` are the leaves.

### Core backend patterns

- **Watch, don't poll.** `SharedInformerFactory` caches K8s objects; `lister.Xxxs(ns).List(...)` reads from the in-memory cache, zero API round trips. `kube.NewFactoryHandle` blocks (≤30s) until every informer's initial LIST has synced before `Connect()` unblocks the frontend's first queries.
- **Detail reads** use `Get<Resource>ByName(namespace, name)` → `lister.Xxxs(ns).Get(name)`, never `ListXxx().find(...)`.
- **Push updates via Wails events**, not polling: Go emits (`runtime.EventsEmit(a.ctx, "pods:update", pods)`), the frontend's per-resource `useXxxUpdateEvents` hook subscribes and the owning data-access hook merges the pushed payload over its `useQuery` result locally (no global cache-write hook).
- **Forbidden-resource (403) handling**: informer 403s emit `resource:forbidden` once per resource per connection; frontend's `useCatchForbiddenResources` toasts when the user is on/navigates to that view.
- Adding a new Go method requires manually adding it to `frontend/src/api/resources.ts`'s re-export list (Wails bindings are auto-generated into `frontend/wailsjs/`, but that file is hand-maintained) — same for any new DTO type needing hand-added entries in `frontend/wailsjs/go/models.ts` if you're not running `wails dev`/`wails generate module` to regenerate.
- `wails generate` mutates `go.mod` — never run it as a shortcut for other tasks.

### Plugin architecture

Plugins are **fully standalone Go modules** (e.g. `plugins/<name>` — own `go.mod` under `github.com/litelensapp/litelens/plugins/<name>`). Each plugin ships its own frontend bundle too (`plugins/<name>/frontend`, its own workspace package).

- **Shared contract via `packages/core` module:** a nested Go module (`github.com/litelensapp/litelens/packages/core`, own `go.mod`) exporting the plugin-host contract: protobuf service definitions (`packages/core/pb/`), data transfer objects (`packages/core/dto/`), and kubeconfig loading utilities (`packages/core/kube/`). Plugins depend on this module (versioned as `packages/core/vX.Y.Z`) instead of hand-copying these packages. See `packages/core/README.md` for versioning and update workflow. **Status:** the host's `internal/` currently maintains parallel copies; Phase 4 of the architecture plan will migrate the host to import `packages/core` as a normal dependency.
- The plugin binary is launched as a subprocess by `internal/plugin`, emits a one-line JSON `READY` handshake (`grpcPort`, `pid`, version) on stdout, then serves gRPC.
- The main app never calls plugin methods directly — everything crosses the boundary via the generic `pb.PluginServer` contract (`GetCapabilities` / `SetClusterContext` / `Invoke(method, payloadJson)`), dispatched inside the plugin's own `internal/server/grpc.go`.
- Plugin frontend bundles are loaded dynamically and resolve `react`/`react-dom`/`@litelens/design-system`/`@litelens/core`/`@tanstack/react-query` as **bare specifiers against the host's own module instances** via an import map (`frontend/index.html`) + verbatim-copied shim files in `frontend/public/vendor/` — this avoids shipping duplicate instances that would break Context (React) or query client isolation.
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
