# litelens memory

## Architecture

- [Architecture decisions](architecture_decisions.md) — IPC, SharedInformerFactory caching, DTO design, package deps, Wails bindings, macOS build
- [File structure](file_structure.md) — full annotated project tree (Go backend + React frontend)
- [Unified tray architecture](unified_tray_architecture.md) — built-in tray families (modification/pod) share one shell via discriminated union + registry; plugin-owned families merge in at runtime
- [Modification tray architecture](modification_tray_architecture.md) — generic cross-resource bottom tray; how to add a new resource kind
- [Detail drawer pattern](detail_drawer_pattern.md) — DetailDrawerContext + DetailBlock; full steps for adding a new drawer end-to-end
- [go.work removal / packages/core replace](go_work_removal_todo.md) — no go.work file, but a bare `replace` for packages/core in go.mod is permanent by design (host+core ship as a pair, like @litelens/core workspace:*)
- [Wails IPC call ordering](wails_ipc_call_ordering.md) — no ordering guarantee for rapid concurrent calls to the same bound method; fix with a frontend-generated monotonic seq, not a Go-side one

## Component & UI conventions

- [Component guidelines](component_guidelines.md) — shadcn-first rule, status badge pattern, ResourceLink, CTA button components, aria-label rule
- [React code quality](react_code_quality.md) — no inline render fns, useReducer for grouped state, React 19 ref-as-prop, no ref access during render
- [Color palettes](project_color_palettes.md) — green-500 positive/active, red-500 destructive, amber=stop/pause, palette CSS token locations
- [Typography scale](typography_scale.md) — .text-h1/-h2/-h3/-body/-caption/-label utility classes, rolled out codebase-wide
- [Tailwind v4 @theme inline](tailwind_v4_theme_inline.md) — color-mix() must live directly in @theme inline, not behind an indirect var()
- [Tinted shadow utilities](tinted_shadow_utilities.md) — .shadow-subtle/-depth-1/-depth-2, Dialog=depth-2, Sheet=depth-1

## Workflow

- [Development with plugins workflow](development_with_plugins_workflow.md) — plugin repo path, `pnpm build:local` for fresh builds, verify E2E at http://localhost:34115 in dev mode

## Feedback & Conventions

- [Development workflow](development_workflow.md) — always run pnpm format + lint + go vet + go build before reporting done; sync auto memory after codebase changes
- [Review findings log](review-findings.md) — recurring issues flagged by `/agent-team --fix` Phase 5 reviews, so future fixes don't repeat them
- [LiteLens→Litelens rebrand casing](litelens_rebrand_casing.md) — what changed vs. deliberately kept (winget ID, cask slug, storage dirs); macOS .app path migration fallback
