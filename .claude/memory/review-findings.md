# Review findings log

Auto-maintained by the `fix` agent-team pipeline (Phase 5). Entries mark ✅ once confirmed fixed by a later QA/review pass.

## 2026-08-11 — useCheckForUpdate silent-failure fix (commit 4b73bdd + follow-up)

- **HIGH** ✅ — React mutation hooks that call a Wails-bound async method and show toasts on completion must guard against concurrent invocation (e.g. `if (checkingForUpdate) return;` before starting a new call). Without it, rapid double-clicks can let a shared "toast already shown" ref/flag from one in-flight call suppress or race the toast from another. Fixed in `frontend/src/app/about/hooks/useCheckForUpdate.ts`.
- **MEDIUM** (not fixed, accepted as-is) — Wails event payloads emitted for events the frontend only keys off by name (not payload contents) are harmless to keep; don't strip them reflexively during review — future consumers may want the payload.
- **MEDIUM** (not fixed, accepted as-is) — Mocks under `frontend/src/__mocks__/wailsjs/` for methods that now return `error` (rejecting promise) only resolve by default; if adding tests for the error path of a Wails-bound method, override the mock per-test rather than changing the shared mock's default behavior.
