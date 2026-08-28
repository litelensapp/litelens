# Uninstallation

### Homebrew (macOS)

```sh
brew uninstall litelens
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
