# Litelens

[![license](https://img.shields.io/github/license/litelensapp/litelens.svg)](LICENSE)
[![release](https://img.shields.io/github/v/release/litelensapp/litelens?display_name=tag&sort=semver)](https://github.com/litelensapp/litelens/releases/latest)

LiteLens is a lightweight, native desktop dashboard for managing Kubernetes
clusters. It's built with [Wails](https://wails.io) (Go backend + React
webview, no Electron), so it stays small and fast while giving you a clean,
modern watch-based UI over your cluster.

https://github.com/user-attachments/assets/7afe5c08-fb14-4ca1-ac33-4a2cbe8d2849

## Installation

### Linux / macOS

#### Latest release

```sh
curl -fsSL "https://raw.githubusercontent.com/litelensapp/litelens/main/scripts/install.sh" | bash
```

#### Specific version

```sh
curl -fsSL "https://raw.githubusercontent.com/litelensapp/litelens/main/scripts/install.sh" | bash -s v1.2.0
```

#### Homebrew (macOS, Apple Silicon)

LiteLens is distributed via a custom Homebrew tap (not `homebrew-core`), so
`brew search litelens` won't find it — tap it explicitly first:

```sh
brew tap litelensapp/homebrew-litelens
brew install litelens
```

If you install via Homebrew, prefer `brew upgrade litelens` over the
in-app updater to keep versions in sync.

### Uninstall

```sh
curl -fsSL "https://raw.githubusercontent.com/litelensapp/litelens/main/scripts/uninstall.sh" | bash
```

#### With cleanup mode to wire all data

```sh
curl -fsSL "https://raw.githubusercontent.com/litelensapp/litelens/main/scripts/uninstall.sh" | bash -s -- cleanup
```

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for setup,
testing, and PR guidelines. Please also review our
[Code of Conduct](CODE_OF_CONDUCT.md) and [Security Policy](SECURITY.md).

## License

LiteLens is licensed under the [Apache License 2.0](LICENSE).
