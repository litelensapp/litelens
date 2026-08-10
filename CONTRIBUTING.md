# Contributing to LiteLens

Welcome to LiteLens! We're excited you want to contribute.

## About LiteLens

LiteLens is a lightweight Kubernetes dashboard built with [Wails v2](https://wails.io). It combines a Go backend with a React/TypeScript frontend running in a native webview shell. The repository is structured around a pnpm workspace with the following key directories:

- **`internal/`** — Go backend implementation (Kubernetes client, business logic, app state)
- **`frontend/`** — React + TypeScript frontend application
- **`design-system/`** — Shared UI kit published as an npm package (`@litelens/design-system`)

## Development Setup

### Local environment

Install dependencies and prepare your environment:

```bash
# Install Node dependencies (pnpm workspace)
pnpm install

# Install Go dependencies
go mod download
```

### Running the app

For hot-reload development of the full desktop app (Go backend + React frontend):

```bash
wails dev
```

For frontend-only development (Vite dev server, useful for pure UI work):

```bash
pnpm dev
```

## Building

### Full application build

```bash
pnpm build:app
```

This runs the build script (`scripts/build.sh`), wraps `wails build`, and on macOS applies ad-hoc code signing.

### Design system only

```bash
pnpm build:ds
```

Builds the `@litelens/design-system` package — required before a frontend build if design-system code has changed.

### Frontend only (without Wails binary)

```bash
pnpm build:app:fe
```

Equivalent to `pnpm build:ds && <frontend build>`.

## Testing

### Go backend tests

```bash
# Run all Go tests with race detector
pnpm test:be

# With coverage
pnpm test:be:coverage

# Single test by name
go test -race -run TestName ./internal/app/...
```

### Frontend tests (Vitest)

```bash
# Run all frontend tests
pnpm test:fe

# With coverage
pnpm test:fe:coverage

# Single test file
pnpm --filter litelens-frontend exec vitest run path/to/File.test.tsx
```

### Design system tests

```bash
pnpm test:ds
```

## Lint & Format

These checks must pass before submitting a PR:

```bash
# Format all code (Prettier: TS, TSX, JS, JSON, CSS, Markdown, YAML)
pnpm format

# Lint TypeScript/React code
pnpm lint

# Lint Go
go vet ./internal/...
go tool staticcheck ./internal/...
```

## Test cluster on local

### Minikube

```sh
minikube addons enable metrics-server
```

### Docker Desktop

```sh
kubectl config use-context docker-desktop
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
kubectl patch deployment metrics-server -n kube-system \
    --type=json \
    -p='[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]'
kubectl rollout status deployment/metrics-server -n kube-system

kubectl top nodes
```

## Branching & Pull Requests

- Fork the repository or create a feature branch off `main`
- Keep PRs focused on a single feature or bug fix
- Describe what you changed and why in the PR description
- Reference any related GitHub Issues

## Developer Certificate of Origin

All commits must be signed off using:

```bash
git commit -s -m "Your commit message"
```

This appends a `Signed-off-by` trailer, certifying that you wrote the code and have the right to contribute it to the project. See the `DCO` file for the full Developer Certificate of Origin v1.1 text. We use DCO for contribution provenance tracking — no CLA required.

## Code Style

### TypeScript & React

Files in `frontend/src` and `design-system/src` use:

- **Imports:** Relative imports within the same top-level directory (`frontend` or `design-system`); `@/...` alias reserved for cross-top-level imports only
- **Icon buttons:** Must include `aria-label` attribute for WCAG 2.1 Level A compliance (enforced by the `icon-button-aria-label/check` ESLint rule)
- **Components:** Before writing custom UI, check if shadcn already has it:

  ```bash
  cd design-system && pnpm run ui:add <component>
  ```

- **Status badges:** Use `Badge`'s `success`/`warning`/`destructive`/`ghost` CVA variants; other colors use `variant="outline"` + className
- **Toasts:** Use `toast.custom(() => renderErrorToast/renderSuccessToast({...}), { style: TOAST_STYLE })` from `design-system/src/components/toasts`; never use `toast.error()` or `toast.success()`
- **CTA buttons:** Reuse shared components (`ResourceModificationButton`, `ResourceDeletionButton`, `ResourceRestartButton`, `ResourceScaleButton`, `ResourceBulkDeletionButton`, `ResourceCreationButton`) instead of inlining icon+dropdown markup

### Go

Files under `internal/` follow:

- Hexagonal + Onion architecture (dependency flows inward; `dto` is the leaf, no cycles)
- Watch-based updates, not polling (use `SharedInformerFactory` and listers for zero-copy reads from in-memory cache)
- Push updates via Wails events, not polling
- Detail reads use `GetXxxByName(namespace, name)` → `lister.Xxxs(ns).Get(name)`, never `ListXxx().find(...)`
- DTO fields use `string` for timestamps, never `time.Time` (Wails TS bindings limitation)

## Reporting Issues

- **Bugs:** Use GitHub Issues with details about reproduction, expected vs. actual behavior, and your environment (OS, K8s version)
- **Features:** Use GitHub Issues to propose new capabilities

## License

Contributions to LiteLens are licensed under the Apache License 2.0 (see `LICENSE` file). By contributing, you agree to license your contribution under the same terms.

---

Thanks for contributing! Please reach out with questions or feedback.
