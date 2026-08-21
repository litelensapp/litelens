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

- **`src/index.ts`** — hook declarations (currently `useClusterWideAPI`), each a stub that throws if called outside a plugin context.
- **`src/types/`** — shared TypeScript types re-exported alongside the hooks (`api`, `nav`, `resources`, `tray`).

## Versioning

Versioned alongside the host app, not as a stable long-term SDK — expect breaking changes as the plugin host API evolves.

## Build

```bash
pnpm --filter @litelens/core build   # tsup, outputs to dist/
```
