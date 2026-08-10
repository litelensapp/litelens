---
name: development-workflow
description: After every dev task run pnpm format + pnpm lint before reporting done; sync auto memory after every codebase change without waiting to be asked
metadata:
  node_type: memory
  type: feedback
  originSessionId: 4c9d20f2-fae5-4639-a340-3b791fa9bae3
---

After every development task, always run:

```bash
pnpm format
pnpm lint

go vet
go build
```

**Why:** Prettier formats all TS/JS/JSON/CSS/MD/YAML; ESLint catches issues. Never skip — run before reporting a task as done.

**How to apply:** These commands are the mandatory finish step for any frontend or full-stack change in this project.

## Unit tests (frontend)

Vitest is set up in `frontend/`. Run from repo root:

```bash
pnpm --filter litelens-frontend test        # run once
pnpm --filter litelens-frontend test:watch  # watch mode
```

Or from `frontend/` directly: `pnpm test`. Test files live in `src/hooks/__tests__/`.
