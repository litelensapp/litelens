# @litelens/core

React hooks for litelens plugin frontends. This is the frontend counterpart to the Go `packages/core` module — see `packages/core/README.md` for the plugin-host contract as a whole.

## Purpose

Plugin frontends run inside the host app's own module instances (React, React DOM, TanStack Query, `@litelens/design-system`) rather than bundling their own. `@litelens/core` is the host-level API surface plugins use to reach into that shared runtime — cluster context, navigation, resource links, the unified tray, etc.

## How it resolves at runtime

This package is a **type/documentation surface only** — every export in `src/index.ts` throws if actually called. The real implementations live in the host app (`frontend/src/expose/`) and are substituted at runtime:

1. The host's import map (`frontend/index.html`) resolves the bare specifier `@litelens/core` to a generated vendor shim (`frontend/public/vendor/litelens/core.js`), not to this package's `dist/`.
2. That shim reads off `window.__LITELENS_VENDOR__.core`, which `frontend/src/expose/index.tsx` populates with the host's real hook implementations before any plugin bundle is dynamically imported.
3. Plugin code imports `@litelens/core` normally (`import { useClusterWideAPI } from "@litelens/core"`) and, at runtime, transparently gets the host's implementation — never the stub in this package.

So this package exists to give plugin authors real types, JSDoc, and editor autocomplete during development, while the host controls the actual behavior. If a hook's signature changes on the host side (`frontend/src/expose/hooks/`), update the matching declaration here to keep them in sync — there is no automated check that they match.

## Contents

- **`src/clusterWideAPI.ts`** (exported as `clusterWideAPI`) — capabilities scoped to the single-cluster view. See "Using `clusterWideAPI`" below.
- **`src/appWideAPI.ts`** (exported as `appWideAPI`) — capabilities with no single-cluster-view constraint. See "Using `appWideAPI`" below.
- **`src/types/`** — shared TypeScript types re-exported alongside the API namespaces above (`api`, `nav`, `resources`, `tray`).
- **`src/build/tsupPreset.ts`** (exported as `@litelens/core/tsup-preset`) — shared tsup build config for plugin frontends: Node-only build tooling, kept out of the `.` export above so it never ships in the browser bundle the vendor shim substitutes for. See "Shared plugin build config" below.

## Using `clusterWideAPI`

Everything under `clusterWideAPI` is only valid inside the single-cluster view (i.e. a component rendered within `MainLayout`'s subtree). Calling the hooks from an app-wide screen (Settings, Marketplace) throws.

- **`useExposeProperties()`** — hook returning cluster-scoped state: `activeContext`, `activeNamespaces`, `activeResource`, `availableNamespaces`, `resourceLinks` (open a built-in resource's detail drawer, e.g. `resourceLinks.pod(namespace, name)`), and `unifiedTray` (`{ openTab }` for the bottom tray).
- **`useExposeMethods()`** — hook returning cluster-scoped actions, currently `onNavigateToView(view)`.
- **`registerViews(pluginId, configs)`** — registers your plugin's main view component(s), one per resource you own. Each `config.name` must match the `view` value used in the nav entry you register via `registerNavEntry` — the host mounts whichever registered view's `name` equals the currently active resource. Optionally pass a per-view `stylesheet` (a `Promise` from a CSS import) that loads only when that view mounts.
- **`registerNavEntry(pluginId, navEntry)`** — registers your plugin's sidebar entry (a single item or a group of items, see `NavEntry` in `src/types/nav.ts`).
- **`registerTrayFamilies(pluginId, families)`** — registers content components for tray families your plugin owns, keyed by an arbitrary family name your own `openTab` calls reference.
- **`registerEvents(pluginId, handlers)`** — subscribes your plugin to the host's plugin event bus, keyed by event name.

`registerViews`, `registerNavEntry`, `registerTrayFamilies`, and `registerEvents` are typically all called once, at module scope, in your plugin's entry file — that way your view, nav entry, tray content, and event handlers are all available as soon as the host dynamically imports your bundle, with no component needing to mount first. Re-registering under the same `pluginId` replaces the previous registration; the host also unregisters everything automatically when the plugin is uninstalled.

```ts
import { clusterWideAPI } from "@litelens/core";
import { HelmChartListView } from "./views/HelmChartListView";
import { HELM_NAV_ENTRY } from "./nav";
import { HELM_TRAY_FAMILIES } from "./tray";

const PLUGIN_ID = "helm";

clusterWideAPI.registerViews(PLUGIN_ID, [{ name: "helm-charts", component: HelmChartListView }]);
clusterWideAPI.registerNavEntry(PLUGIN_ID, HELM_NAV_ENTRY);
clusterWideAPI.registerTrayFamilies(PLUGIN_ID, HELM_TRAY_FAMILIES);
clusterWideAPI.registerEvents(PLUGIN_ID, {
  "helm:release-updated": (payload) => {
    appWideAPI.getQueryClient().invalidateQueries({ queryKey: ["helm", "releases"] });
  },
});
```

Inside a mounted view component:

```tsx
function HelmChartListView() {
  const { activeContext, resourceLinks } = clusterWideAPI.useExposeProperties();
  const { onNavigateToView } = clusterWideAPI.useExposeMethods();
  // ...
}
```

## Using `appWideAPI`

- **`registerStylesheets(pluginId, stylesheets)`** — registers your plugin's global stylesheet(s) (e.g. compiled Tailwind output), loaded once at the app level regardless of which of your views is active. Call this once at module scope, alongside the `clusterWideAPI` registration calls above — don't use per-view `stylesheet` on `registerViews` for CSS that should apply across all of your views.
- **`getQueryClient()`** — returns the host's singleton `QueryClient`. Use this from code that isn't a mounted React component (e.g. the module-scope `registerEvents` handler above) and therefore can't call `useQueryClient()`.

## Shared plugin build config

`@litelens/core/tsup-preset` exports `createPluginTsupConfig({ pluginRoot })`, the tsup config every plugin frontend needs: externalizes the host-shared runtime deps (`PLUGIN_SHARED_EXTERNALS`), inlines Tailwind-compiled CSS as text via a local Tailwind CLI pass, and optionally emits a bundle-analysis report to `dist/stats/` when `ANALYZE=true`. A plugin's own `tsup.config.ts` becomes:

```ts
import { createPluginTsupConfig } from "@litelens/core/tsup-preset";
import path from "node:path";
import { fileURLToPath } from "node:url";

export default createPluginTsupConfig({
  pluginRoot: path.dirname(fileURLToPath(import.meta.url)),
});
```

Pass `entry` or `external` to override the entrypoint or add plugin-specific externals. Unlike the `.` export, this subpath pulls in real Node dependencies (`tsup`, `esbuild-visualizer`, `@tailwindcss/cli`) — they're declared as `dependencies` of this package (not `devDependencies`) so consuming plugins get them transitively.

## Versioning

Versioned alongside the host app, not as a stable long-term SDK — expect breaking changes as the plugin host API evolves.

## Build

```bash
pnpm --filter @litelens/core build   # tsup, outputs to dist/
```
