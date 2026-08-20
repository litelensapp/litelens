---
name: go-work-removal-todo
description: go.work shim was removed once packages/core/v1.7.0 was tagged — internal/ now imports packages/core as a normal pinned Go dependency
metadata:
  node_type: memory
  type: project
  originSessionId: d92fb7a1-c0a5-4c8c-9225-630bbe759b97
  modified: 2026-08-20T17:51:40.004Z
---

DONE (2026-08-21, branch `chore/remove-go-work-shim`): root `go.work`/`go.work.sum` deleted and the `replace github.com/litelensapp/litelens/packages/core => ./packages/core` directive removed from root `go.mod`. `go.mod` now requires `github.com/litelensapp/litelens/packages/core v1.7.0` (resolved via `go get .../packages/core@v1.7.0` — note the module path already includes `packages/core`, so the `@` version is just `vX.Y.Z`, not `packages/core/vX.Y.Z`). Confirmed with `go build ./...`, `go vet ./internal/...`, `go mod tidy`, `pnpm lint:be`, and `pnpm test:be` (all pass).

Original context, kept for history: this was an interim mechanism from Phase 4 of [[plugin-architecture-inversion]] — `packages/core` is a separate nested Go module (own `go.mod`), so without a published tag `go build ./internal/...` couldn't resolve `github.com/litelensapp/litelens/packages/core/...` as a normal dependency, and `go.work` substituted the on-disk directory instead.
