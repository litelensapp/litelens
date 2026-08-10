# litelens

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

Detects your OS and architecture, downloads the specified release from GitHub, installs the binary, and on Linux creates a `.desktop` entry so the app appears in the launcher.

**Linux system dependencies** (Debian/Ubuntu — required by the WebKit runtime):

```sh
sudo apt-get install libgtk-3-0 libwebkit2gtk-4.1-0
```

### Uninstall

```sh
curl -fsSL "https://raw.githubusercontent.com/litelensapp/litelens/main/scripts/uninstall.sh" | bash
```

Outputs a native binary to `build/bin/litelens` (macOS: `build/bin/litelens.app`).

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for setup, testing, and PR guidelines. Please also review our [Code of Conduct](CODE_OF_CONDUCT.md) and [Security Policy](SECURITY.md).
