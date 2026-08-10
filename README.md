# litelens

## Verification

```sh
# vet & lint
go vet ./internal/... && go tool staticcheck ./internal/...
```

## Dev mode (hot-reload)

```sh
wails dev
```

## Build App

```sh
pnpm build:app
```

## Installation

### Linux / macOS

Add `LITELENS_ACCESS_TOKEN` to your shell profile once:

```sh
# ~/.bashrc or ~/.zshrc
export LITELENS_ACCESS_TOKEN=ghp_xxx
```

Then install:

#### Latest release

```sh
curl -fsSL \
  -H "Authorization: Bearer $LITELENS_ACCESS_TOKEN" \
  -H "Accept: application/vnd.github.raw+json" \
  "https://api.github.com/repos/litelensapp/litelens/contents/scripts/install.sh" \
  | LITELENS_ACCESS_TOKEN="$LITELENS_ACCESS_TOKEN" bash
```

#### Specific version

```sh
curl -fsSL \
  -H "Authorization: Bearer $LITELENS_ACCESS_TOKEN" \
  -H "Accept: application/vnd.github.raw+json" \
  "https://api.github.com/repos/litelensapp/litelens/contents/scripts/install.sh" \
  | LITELENS_ACCESS_TOKEN="$LITELENS_ACCESS_TOKEN" bash -s v1.2.0
```

> `raw.githubusercontent.com` returns 404 for private repos even when logged in — the GitHub Contents API with a token is required.
>
> Generate a token at **GitHub → Settings → Developer settings → Personal access tokens** with `Contents: Read-only` (fine-grained) or `repo` (classic) scope.

Once the repo is made public, the standard `curl ... | bash` without a token will work.

Detects your OS and architecture, downloads the specified release from GitHub, installs the binary, and on Linux creates a `.desktop` entry so the app appears in the launcher.

**Linux system dependencies** (Debian/Ubuntu — required by the WebKit runtime):

```sh
sudo apt-get install libgtk-3-0 libwebkit2gtk-4.1-0
```

### Uninstall

```sh
curl -fsSL \
  -H "Authorization: Bearer $LITELENS_ACCESS_TOKEN" \
  -H "Accept: application/vnd.github.raw+json" \
  "https://api.github.com/repos/litelensapp/litelens/contents/scripts/uninstall.sh" \
  | LITELENS_ACCESS_TOKEN="$LITELENS_ACCESS_TOKEN" bash
```

Outputs a native binary to `build/bin/litelens` (macOS: `build/bin/litelens.app`).

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

claude --resume 913764e7-f443-490a-b317-84d757c0560e
