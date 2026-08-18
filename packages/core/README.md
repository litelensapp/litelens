# litelens-core

Shared Go module providing the plugin-host contract: protobuf service definitions, data transfer objects (DTOs), and kubeconfig loading utilities.

## Purpose

This module formalizes packages that plugins (`litelens-plugins`) previously hand-copied and kept manually in sync with the host. Instead of duplicate copies, plugins now import these packages as a real versioned dependency of this module, ensuring correctness and reducing maintenance burden.

## Contents

- **`pb/`** — protobuf service definitions (`plugin.proto`) and generated Go code (`plugin.pb.go`, `plugin_grpc.pb.go`). Defines the gRPC service contract (`GetCapabilities`, `SetClusterContext`, `Invoke`) between the host and plugin subprocesses.
- **`dto/`** — data transfer objects (all 36 DTO types from the host's internal schema). Defines the shape of K8s resource data passed between host and plugin frontends. All timestamp fields use `string` type (not `time.Time`), matching Wails binding conventions.
- **`kube/`** — utility for loading kubeconfig files. Contains only `LoadingRules(paths []string)`, which returns a client-go `ClientConfigLoadingRules` instance for the given kubeconfig paths (or defaults to the standard KUBECONFIG env var / `~/.kube/config` if none supplied).

## Package layout

One Go package per concern (`pb/`, `dto/`, `kube/`) deliberately leaves room for future host-utility packages to be added as sibling packages later — e.g. a menu bar/settings integration package — without requiring a restructure. Future packages should follow the same pattern: one package, one concern, documented in this README.

## Versioning

This module is tagged independently using Go's subdirectory-module convention:

- **Tag format:** `packages/core/vX.Y.Z` (e.g. `packages/core/v0.1.0`, `packages/core/v1.2.3`) — the tag prefix matches this module's path relative to the repo root, per Go's subdirectory-module convention.
- **In dependency declarations:** `go get github.com/litelensapp/litelens/packages/core@packages/core/v0.1.0`.
- The host repo's main module (`github.com/litelensapp/litelens`) has its own separate tags (`v1.2.0`, etc.) and release cadence — they are independent.

## Source-of-truth decision

**The host's `internal/` packages (`internal/plugin/pb/`, `internal/dto/`, `internal/kube`) are currently hand-maintained parallel copies.** Future phases (Phase 4, not part of this provisioning) will decide whether:

1. The host's `internal/` code imports `core/` directly as a normal module dependency, or
2. `core/` is kept as the source of truth, with updates propagated _back_ to `internal/` copies via an automated sync step.

This phase does **not** perform that migration — it only provisions the `core/` module standalone. Phase 4 will make the dependency direction choice explicit and implement the corresponding workflow.

### Intentional divergence: `PluginLockFile.Timestamp` field type

`core/dto/plugin.go`'s `PluginLockFile.Timestamp` field is intentionally typed as `string` (RFC3339 format) to match the Wails binding convention used throughout `core/dto`. This diverges from `internal/dto/plugin.go`, where the same field is currently typed as `time.Time` — a pre-existing bug in `internal/` that violates the project convention documented in CLAUDE.md ("fields must use `string` for timestamps, never `time.Time`"). The `internal/dto/plugin.go` bug is out of scope for this provisioning phase and should be fixed as a separate, dedicated follow-up (see the plan's Phase 1 notes).

## Use by plugins

Once this module is versioned and tagged (e.g. `packages/core/v0.1.0`), plugin repositories (e.g. `litelens-plugins/plugins/helm/`) depend on it by adding to their `go.mod`:

```go
require github.com/litelensapp/litelens/packages/core v0.1.0  // for subdirectory tag packages/core/v0.1.0
```

Then import from the module normally:

```go
import (
    "github.com/litelensapp/litelens/packages/core/pb"
    "github.com/litelensapp/litelens/packages/core/dto"
    "github.com/litelensapp/litelens/packages/core/kube"
)
```

Previously, plugins hand-copied these files and manually kept them in sync. This module eliminates that burden.

## CI/CD

This module has its own CI job (the `core` job in `.github/workflows/job-build-check.yml`) that runs `go vet`, `go build`, and `staticcheck` independently, since it is a separate Go module (`go.mod`) with its own dependency tree. The root module's `go.mod` does not govern this module's build.
