# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

LiteLens — a desktop Kubernetes dashboard built with [Wails v2](https://wails.io) (Go backend + React/TypeScript frontend, native webview shell, no Electron). Root module `github.com/gknguyen/litelens`. pnpm workspace with `frontend`, `design-system`, and `plugins/helm/frontend` as packages. Plugins (e.g. Helm) are separate Go modules, launched as subprocesses and driven over gRPC — never compiled into the main binary.

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
pnpm build:plugin:helm   # build the Helm plugin (standalone Go module + its own frontend)
```

### Lint / format / vet (run after every change, before reporting done)

```bash
pnpm format              # prettier --write across ts/tsx/js/json/css/md/yml
pnpm lint                 # eslint frontend/src design-system/src
go vet ./internal/...
go tool staticcheck ./internal/...
```

### Tests

```bash
pnpm test:be              # go test -race -v ./internal/...
pnpm test:be:coverage
pnpm test:fe               # vitest run (frontend)
pnpm test:fe:coverage
pnpm test:ds                # design-system vitest
pnpm test:plugin:helm:be    # cd plugins/helm && go test -race -v ./...
pnpm test:plugin:helm:fe
```

Single Go test: `go test -race -run TestName ./internal/app/...`
Single frontend test: `pnpm --filter litelens-frontend exec vitest run path/to/File.test.tsx`

### Local test cluster

```bash
minikube addons enable metrics-server
# or, for Docker Desktop's docker-desktop context, patch metrics-server with --kubelet-insecure-tls (see README.md)
```

## Architecture

### Backend (`internal/`)

- **`internal/app`** — one file per Wails-bound method group (`pod.go`, `deployment.go`, `node.go`, …), plus `app.go` (the `App` struct, `Connect`/`Disconnect`, event wiring). Every exported method on `App` is auto-bound to TypeScript by Wails (`main.go` binds only `*App` — `bindList := []any{a}`; nothing else is exposed directly).
- **`internal/kube`** — client/informer plumbing: per-context `*kubernetes.Clientset` cache (mutex-guarded map), `clientcmd` loading rules (respects `$KUBECONFIG`), `SharedInformerFactory` handles, node/pod metrics fetch.
- **`internal/kube/resources`** — per-resource-type `ListXxx`/`toXxx` conversion logic (raw K8s object → DTO).
- **`internal/dto`** — leaf package, pure DTO type definitions (no imports of `kube`/`app`). One type per K8s resource kind; **fields must use `string` for timestamps, never `time.Time`** — Wails can't generate TS bindings for it.
- **`internal/plugin`** — generic plugin host: discovery, download/verify, process lifecycle (`loader.go`), gRPC handshake. Talks to plugin subprocesses ONLY through `Invoke(pluginID, method, payloadJSON)` — see Plugin architecture below.
- **`internal/updater`** — self-update (checks GitHub releases, downloads, swaps binary).
- **`internal/config`** — app config (`~/.config/litelens/settings.json`).

**Package dependency direction** (no cycles): `dto` ← `kube/resources` ← `kube` ← `app`. `dto` is the leaf.

### Core backend patterns

- **Watch, don't poll.** `SharedInformerFactory` caches K8s objects; `lister.Xxxs(ns).List(...)` reads from the in-memory cache, zero API round trips. `kube.NewFactoryHandle` blocks (≤30s) until every informer's initial LIST has synced before `Connect()` unblocks the frontend's first queries.
- **Detail reads** use `Get<Resource>ByName(namespace, name)` → `lister.Xxxs(ns).Get(name)`, never `ListXxx().find(...)`.
- **Push updates via Wails events**, not polling: Go emits (`runtime.EventsEmit(a.ctx, "pods:update", pods)`), the frontend's per-resource `useXxxUpdateEvents` hook subscribes and the owning data-access hook merges the pushed payload over its `useQuery` result locally (no global cache-write hook).
- **Forbidden-resource (403) handling**: informer 403s emit `resource:forbidden` once per resource per connection; frontend's `useCatchForbiddenResources` toasts when the user is on/navigates to that view.
- Adding a new Go method requires manually adding it to `frontend/src/api/resources.ts`'s re-export list (Wails bindings are auto-generated into `frontend/wailsjs/`, but that file is hand-maintained) — same for any new DTO type needing hand-added entries in `frontend/wailsjs/go/models.ts` if you're not running `wails dev`/`wails generate module` to regenerate.
- `wails generate` mutates `go.mod` — never run it as a shortcut for other tasks.

### Plugin architecture (Helm)

Plugins are **fully standalone Go modules** (e.g. `plugins/helm` — `go.mod` for `github.com/gknguyen/litelens/plugins/helm`, zero dependency on the root module: local copies of `dto`, the plugin gRPC `pb` package, and `kube.LoadingRules`). Each plugin ships its own frontend bundle too (`plugins/helm/frontend`, workspace package `@litelens/helm-plugin-frontend`).

- The plugin binary is launched as a subprocess by `internal/plugin`, emits a one-line JSON `READY` handshake (`grpcPort`, `pid`, version) on stdout, then serves gRPC.
- The main app never calls plugin methods directly — everything crosses the boundary via the generic `pb.PluginServer` contract (`GetCapabilities` / `SetClusterContext` / `Invoke(method, payloadJson)`), dispatched inside the plugin's own `internal/server/grpc.go`.
- Plugin frontend bundles are loaded dynamically and resolve `react`/`react-dom`/`@litelens/design-system`/`@tanstack/react-query` as **bare specifiers against the host's own module instances** via an import map (`frontend/index.html`) + verbatim-copied shim files in `frontend/public/vendor/` — this avoids shipping a second React/query-client instance that would break Context.
- Plugins are installed at runtime from a marketplace (GitHub release manifests); nothing plugin-specific is compiled into `main.go`.

### Frontend (`frontend/src`)

- **`app/`** — all feature code. `app/clusters/modules/<group>/<resource>/` holds one folder per K8s resource (e.g. `workloads/pods`, `configs/secrets`, `networks/services`), each split into `components/`, `hooks/data-access/`, `hooks/data-mutation/`, `api/`. `app/clusters/shared/` holds cross-resource hooks/components/trays. `app/marketplace`, `app/settings`, `app/updater`, `app/about` are top-level non-cluster modules.
- **`design-system/`** (repo-root package, `@litelens/design-system`) — publishable UI kit. `atoms/` = shadcn-generated primitives wrapping `@base-ui/react` (NOT Radix — `TooltipTrigger` etc. use the `render` prop, not `asChild`). `components/` = composite/business components (buttons, drawers, modals, tables, toasts). Never import `design-system` → `app` except the one sanctioned exception (unified tray family registry, injected via props).
- **Per-cluster query isolation**: every TanStack Query hook's key includes `context` (cluster name); switching clusters calls `queryClient.removeQueries` and `MainLayout` remounts (`key={activeContext}`).
- **Code splitting**: every resource view + heavy widget (xterm, detail drawers) is `React.lazy`-loaded.
- All resource views follow one shape: header (title, count, namespace select, search) → `<Table>` → empty-state row.

### Unified tray system

Three tray "families" (modification/YAML-edit, pod logs+exec, Helm chart install) share one bottom-tray shell (`UnifiedTrayProvider`/`UnifiedTrayShell`, mounted once in `MainLayout.tsx`) via a discriminated-union tab type and a family→content-component registry. Open a tab from anywhere with `useUnifiedTray().openTab(family, params)`. To add a new resource's modification tray: add the kind to `ModificationResourceKind`, create `<Resource>ModificationTray.tsx` (mirror `NamespaceModificationTray.tsx`), register it in `modificationTrayRegistry.tsx` — no changes needed to the shell itself. See `.claude/memory/unified_tray_architecture.md` and `.claude/memory/modification_tray_architecture.md` for full detail.

### DTO/detail-drawer conventions

Each resource has a `dto.<Type>` (Go) and matching TS interface. List views use lean fields; detail drawers add `CreatedAt`, `Labels`, `Annotations`, `ManagedFields`, etc. Detail drawers render via `ResourceDetailDrawer` with an `isLoading`/empty two-gate pattern. Cross-resource references (Namespace, ClusterRole, owning workload, etc.) always render as `<ResourceLink onClick={...}>` wired to `MainLayoutContext`'s `onToggle<Resource>Detail`, never plain text.

## Project-specific conventions

- **Before building any custom UI primitive**, check shadcn first: `cd design-system && pnpm run ui:add <component>`. Only hand-roll if no shadcn/Base UI equivalent exists.
- **Status badges** use `Badge`'s `success`/`warning`/`destructive`/`ghost` `cva` variants for red/green/amber/muted; other colors still use `variant="outline"` + manual className.
- **Never use `toast.error()`/`toast.success()`** — use `toast.custom(() => renderErrorToast/renderSuccessToast({...}), { style: TOAST_STYLE })` from `design-system/src/components/toasts`.
- Reuse the shared CTA button components (`ResourceModificationButton`, `ResourceDeletionButton`, `ResourceRestartButton`, `ResourceScaleButton`, `ResourceBulkDeletionButton`, `ResourceCreationButton`) instead of inlining icon+dropdown/tooltip markup for edit/delete/restart/scale/bulk-delete/create actions.
- Icon-only buttons require `aria-label` (WCAG 2.1 Level A) — enforced by the custom `icon-button-aria-label/check` ESLint rule (`eslint.rules/icon-button-aria-label.js`).
- Imports: files within the same top-level dir (`design-system`, or a given `app/...` module) use **relative imports**, not the `@/...` alias — the alias is reserved for cross-top-level imports.
- After any dev task: run `pnpm format`, `pnpm lint`, `go vet`, `go build` before reporting done.

## Memory

`.claude/memory/` contains extensive project memory (architecture decisions, file structure, gotchas, feature history) auto-maintained across sessions — check `.claude/memory/MEMORY.md` for the index when you need history/rationale beyond what's summarized here.
