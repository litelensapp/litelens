---
name: go-work-removal-todo
description: go.work at repo root is a temporary shim until packages/core is first tagged — remove it once that happens
metadata:
  node_type: memory
  type: project
  originSessionId: d92fb7a1-c0a5-4c8c-9225-630bbe759b97
  modified: 2026-08-18T16:31:38.693Z
---

Root `go.work` (created during Phase 4 of [[plugin-architecture-inversion]]) is an interim mechanism only: it lets `internal/` import `packages/core/{pb,dto,kube}` from the local checkout because no `packages/core/vX.Y.Z` git tag has ever been published yet — Go can't resolve a versioned module dependency without one.

**Why:** `packages/core` is a separate nested Go module (own `go.mod`), so without a tag, `go build ./internal/...` can't resolve `github.com/litelensapp/litelens/packages/core/...` as a normal dependency. `go.work` substitutes the on-disk directory instead, with no changes to either `go.mod`.

**How to apply:** once `.github/workflows/job-publish-app-core-packages.yml`'s `publish-be` job has run at least once (gated behind repo var `vars.CORE_PACKAGE_RELEASED == 'true'` in `cd.yml`) and a `packages/core/vX.Y.Z` tag exists:
1. Delete `go.work` and `go.work.sum` from the repo root.
2. In root `go.mod`, run `go get github.com/litelensapp/litelens/packages/core@packages/core/vX.Y.Z` to add a normal pinned dependency.
3. Re-run `go build ./... && go vet ./internal/... && pnpm test:be` to confirm the switch didn't break anything.

Until then, `go.work` carries a `// TODO` comment pointing back to this same removal sequence.
