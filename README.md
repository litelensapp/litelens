# Litelens

[![license](https://img.shields.io/github/license/litelensapp/litelens.svg)](LICENSE)
[![release](https://img.shields.io/github/v/release/litelensapp/litelens?display_name=tag&sort=semver)](https://github.com/litelensapp/litelens/releases/latest)

LiteLens is a lightweight, native desktop dashboard for managing Kubernetes
clusters. It's built with [Wails](https://wails.io) (Go backend + React
webview, no Electron), so it stays small and fast while giving you a clean,
modern watch-based UI over your cluster.

https://github.com/user-attachments/assets/7afe5c08-fb14-4ca1-ac33-4a2cbe8d2849

## Installation

### Homebrew (macOS)

LiteLens is distributed via a custom Homebrew tap (not `homebrew-core`), so
`brew search litelens` won't find it — tap it explicitly first:

```sh
brew tap litelensapp/homebrew-litelens
brew trust litelensapp/litelens/litelens
brew install litelens
```

### APT (Debian/Ubuntu)

LiteLens publishes `.deb` packages to a self-hosted APT repository, signed
with a dedicated GPG key. Pick the command block matching your Ubuntu
release:

**Ubuntu 24.04 (noble)**

```sh
curl -fsSL https://litelensapp.github.io/litelens-apt/keys/litelens-keyring.gpg | sudo gpg --dearmor -o /usr/share/keyrings/litelens-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/litelens-archive-keyring.gpg] https://litelensapp.github.io/litelens-apt noble main" | sudo tee /etc/apt/sources.list.d/litelens.list
sudo apt-get update && sudo apt-get install litelens
```

**Ubuntu 22.04 (jammy)**

```sh
curl -fsSL https://litelensapp.github.io/litelens-apt/keys/litelens-keyring.gpg | sudo gpg --dearmor -o /usr/share/keyrings/litelens-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/litelens-archive-keyring.gpg] https://litelensapp.github.io/litelens-apt jammy main" | sudo tee /etc/apt/sources.list.d/litelens.list
sudo apt-get update && sudo apt-get install litelens
```

**Ubuntu 20.04 (focal)**

```sh
curl -fsSL https://litelensapp.github.io/litelens-apt/keys/litelens-keyring.gpg | sudo gpg --dearmor -o /usr/share/keyrings/litelens-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/litelens-archive-keyring.gpg] https://litelensapp.github.io/litelens-apt focal main" | sudo tee /etc/apt/sources.list.d/litelens.list
sudo apt-get update && sudo apt-get install litelens
```

If you install via `apt`, prefer `apt upgrade` over the in-app updater — the
in-app updater detects apt-managed installs and silently defers to `apt`
rather than prompting.

### Manual (Linux + MacOS)

```sh
curl -fsSL "https://raw.githubusercontent.com/litelensapp/litelens/main/scripts/install.sh" | bash
```

## Uninstallation

### Homebrew (macOS)

```sh
brew uninstall litelensapp/litelens/litelens
```

### APT (Debian/Ubuntu)

```sh
sudo apt remove litelens
```

This preserves `~/.litelens` (settings, installed plugins). To wipe it too:

```sh
sudo apt remove --purge litelens
```

### Manual (Linux + MacOS)

```sh
curl -fsSL "https://raw.githubusercontent.com/litelensapp/litelens/main/scripts/uninstall.sh" | bash
```

This preserves `~/.litelens` (settings, installed plugins). To wipe it too:

```sh
curl -fsSL "https://raw.githubusercontent.com/litelensapp/litelens/main/scripts/uninstall.sh" | bash -s -- cleanup
```

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for setup,
testing, and PR guidelines. Please also review our
[Code of Conduct](CODE_OF_CONDUCT.md) and [Security Policy](SECURITY.md).

## License

LiteLens is licensed under the [Apache License 2.0](LICENSE).
