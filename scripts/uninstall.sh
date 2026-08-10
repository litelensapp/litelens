#!/usr/bin/env bash
set -euo pipefail

GREEN='\033[0;32m'; NC='\033[0m'
success() { echo -e "${GREEN}✓${NC} $*"; }

OS="$(uname -s)"

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
