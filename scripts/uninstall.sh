#!/usr/bin/env bash
set -euo pipefail

GREEN='\033[0;32m'; NC='\033[0m'
success() { echo -e "${GREEN}✓${NC} $*"; }

OS="$(uname -s)"
CLEANUP=false

for arg in "$@"; do
  case "$arg" in
    cleanup)
      CLEANUP=true
      ;;
  esac
done

if [[ "$OS" == "Darwin" ]]; then
  rm -rf /Applications/LiteLens.app
  rm -f /usr/local/bin/litelens
  success "LiteLens removed"
else
  sudo rm -f /usr/local/bin/litelens
  rm -f "$HOME/.local/share/applications/litelens.desktop"
  if command -v update-desktop-database &>/dev/null; then
    update-desktop-database "$HOME/.local/share/applications"
  fi
  success "LiteLens removed"
fi

if [[ "$CLEANUP" == "true" ]]; then
  # Settings (incl. AccessToken) live under the OS user-config dir, resolved the
  # same way Go's os.UserConfigDir() does.
  case "$OS" in
    Darwin)
      SETTINGS_DIR="$HOME/Library/Application Support/litelens"
      ;;
    *)
      SETTINGS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/litelens"
      ;;
  esac

  # Installed plugins live under ~/.litelens by default (overridable via
  # settings.PluginsDir, which is being deleted below anyway).
  DATA_DIR="$HOME/.litelens"

  rm -rf "$SETTINGS_DIR"
  rm -rf "$DATA_DIR"
  success "LiteLens persistent data removed"
fi
