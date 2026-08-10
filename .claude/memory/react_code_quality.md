---
name: react-code-quality-patterns
description: "litelens React coding conventions — no inline render functions, useReducer for grouped state, timer cleanup via status-driven effects, accessible inputs"
metadata:
  node_type: memory
  type: project
  originSessionId: 4c9d20f2-fae5-4639-a340-3b791fa9bae3
---

### No inline render functions

Never call `renderX()` inside JSX — extract as a named component so React can reconcile by identity. Example: `AppContent` in `App.tsx` instead of `renderContent()`.

### Related state → useReducer

When 3+ state slices update together (load, save, reset), group them in a typed reducer. `SettingsView` uses `formReducer` for `{ varRows, secretRows, shellPath, terminalCopyPaste, status }`. Pure UI/nav state (`section`, `envOpen`) stays as `useState`.

### Timer cleanup via status-driven effect

Don't hold timer IDs in refs and clean them up in an unrelated effect. Instead, watch the state that drives the timer:

```tsx
useEffect(() => {
  if (form.status !== "saved") return;
  const t = setTimeout(() => dispatch({ type: "SAVE_RESET" }), 2000);
  return () => clearTimeout(t);
}, [form.status]);
```

This lets React own the lifecycle; the cleanup fires automatically on unmount or status change.

### Accessible inputs

Every `<input>` must have a visible `<label>` or `aria-label`. Don't rely on `placeholder` — it disappears when the user types. Never use `autoFocus` — it disorients screen reader and keyboard users. Custom Checkbox (base-ui renders `<button>`, not `<input>`) — use `<div>` wrapper, not `<label>`, and set `aria-label` on the Checkbox itself.

### React 19 ref-as-prop (no forwardRef)

In React 19 (`@types/react@19.2.17`), `ref` is a normal prop. Never use `forwardRef` — it's deprecated. Include `ref?: Ref<Handle>` in the props interface directly:

```tsx
export interface MyHandle { doSomething: () => void }
export interface MyProps { ...; ref?: Ref<MyHandle> }
const MyInner = ({ ref, ...props }: MyProps) => {
  useImperativeHandle(ref, () => ({ doSomething }), [doSomething])
  // ...
}
export const My = ({ ref, ...props }: MyProps) => <div><MyInner {...props} ref={ref} /></div>
```

Parent uses callback refs for dynamic instances: `ref={(handle) => { refs.current[id] = handle }}`.

### Imperative handles over prop callbacks for child→parent methods

When a parent needs to call a child function (clear, reconnect, search), expose it via `useImperativeHandle`, not via `onXxxReady` callbacks in `useEffect`. Parent holds `useRef<Record<string, Handle | null>>({})` for multiple dynamic instances. This eliminates `no-prop-callback-in-effect` and `no-pass-data-to-parent` violations.

### Never read/write `ref.current` during render

The `react-hooks/refs` ESLint rule (in `eslint.config.js`) hard-errors on any ref access in the render body — only inside effects/handlers is allowed. This rules out the common "useRef instead of useState to avoid a re-render" fix for values compared during render (e.g. the "derived state when a prop changes" pattern: `if (values !== prevValues) { ...; setEditedValues(...) }`). For that pattern, keep `prevValues` as real state — either its own `useState` (React's documented recipe) or folded into a `useReducer` alongside the derived value it gates, e.g. `HelmChartVersionTray.tsx`'s `valuesEditorReducer` combines `editedValues`+`prevValues` into one reducer so the raw `useState` count drops without touching a ref. Confirmed by hitting `Cannot access ref value during render` / `Cannot update ref during render` lint errors when attempting the ref conversion.

### Stable keys for add/remove form rows

For dynamically added/removed rows (LimitRange/ResourceQuota creation modals, ConfigMap/Secret new-entry editors), never key by array index — deletions/reorders remap index to the wrong row. Use a per-row `id: number` field seeded from a `useRef(0)` counter incremented in the add handler (ref mutation is fine here since it only happens in an event handler, not during render), and update/delete by `row.id` instead of index.

### status+error → useReducer in hooks

When a hook has both `status` (enum) and `error` (string|null) state that always update together, use `useReducer` not two `useStates`:

```ts
type State = { status: MyStatus; error: string | null }
type Action = { type: "connecting" | "active" | "closed" } | { type: "error"; error: string }
function reducer(_: State, action: Action): State { ... }
const [{ status, error }, dispatch] = useReducer(reducer, { status: "connecting", error: null })
```

Keep `onStatusChange` ref-synced so the callback never goes stale:

```ts
const onStatusChangeRef = useRef(onStatusChange);
useEffect(() => {
  onStatusChangeRef.current = onStatusChange;
}); // no dep array
```
