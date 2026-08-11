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
  # All LiteLens app data (settings.json, plugins) now lives under ~/.litelens.
  # Also remove old settings location for thoroughness.
  APP_DATA_DIR="$HOME/.litelens"

  # Remove old settings location (OS-specific config dir)
  case "$OS" in
    Darwin)
      rm -rf "$HOME/Library/Application Support/litelens"
      ;;
    *)
      rm -rf "${XDG_CONFIG_HOME:-$HOME/.config}/litelens"
      ;;
  esac

  # Remove consolidated app data directory
  rm -rf "$APP_DATA_DIR"
  success "LiteLens persistent data removed"
fi
