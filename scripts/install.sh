#!/usr/bin/env bash
set -euo pipefail

REPO="gknguyen/litelens"
BIN_NAME="litelens"
INSTALL_DIR="/usr/local/bin"
DESKTOP_DIR="$HOME/.local/share/applications"

# ── colours ──────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
info()    { echo -e "${GREEN}→${NC} $*"; }
warn()    { echo -e "${YELLOW}⚠${NC}  $*"; }
error()   { echo -e "${RED}✗${NC}  $*" >&2; exit 1; }
success() { echo -e "${GREEN}✓${NC} $*"; }

# ── detect OS + arch ─────────────────────────────────────────────────────────
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
  Linux)
    case "$ARCH" in
      x86_64)  ARTIFACT="litelens-linux-amd64.tar.gz" ;;
      *)        error "Unsupported Linux architecture: $ARCH" ;;
    esac
    ;;
  Darwin)
    case "$ARCH" in
      arm64)   ARTIFACT="litelens-darwin-arm64.zip" ;;
      x86_64)  ARTIFACT="litelens-darwin-amd64.zip" ;;
      *)        error "Unsupported macOS architecture: $ARCH" ;;
    esac
    ;;
  *)
    error "Unsupported OS: $OS (Windows users: download the .exe from GitHub Releases)"
    ;;
esac

# ── auth ─────────────────────────────────────────────────────────────────────
# For private repos, export LITELENS_ACCESS_TOKEN in ~/.bashrc or ~/.zshrc:
#   export LITELENS_ACCESS_TOKEN=ghp_xxx
LITELENS_ACCESS_TOKEN="${LITELENS_ACCESS_TOKEN:-}"

# HTTP/2 has stream teardown issues on some Linux curl builds; force HTTP/1.1
CURL_OPTS=()
[[ "$OS" == "Linux" ]] && CURL_OPTS+=(--http1.1)

gh_curl() {
  local args=(-sL "${CURL_OPTS[@]+"${CURL_OPTS[@]}"}")
  [[ -n "$LITELENS_ACCESS_TOKEN" ]] && args+=(-H "Authorization: Bearer ${LITELENS_ACCESS_TOKEN}")

  local tmp http_code body
  tmp=$(mktemp)
  http_code=$(curl "${args[@]}" -o "$tmp" -w "%{http_code}" "$@")
  body=$(cat "$tmp"); rm -f "$tmp"

  if [[ "$http_code" == "401" || "$http_code" == "403" ]]; then
    error "GitHub API returned $http_code — token is invalid or missing required permissions."
  elif [[ "$http_code" == "404" ]]; then
    if [[ -z "$LITELENS_ACCESS_TOKEN" ]]; then
      error "GitHub API returned 404 — this is a private repo.\nExport your token and re-run:\n  export LITELENS_ACCESS_TOKEN=ghp_xxx"
    else
      error "GitHub API returned 404 — check the repo name or that the release/tag exists."
    fi
  elif [[ "$http_code" != "200" ]]; then
    error "GitHub API request failed with HTTP $http_code"
  fi

  echo "$body"
}

# ── resolve version ───────────────────────────────────────────────────────────
# Priority: positional arg > VERSION env var > latest release
VERSION="${1:-${VERSION:-}}"

if [[ -n "$VERSION" ]]; then
  TAG="$VERSION"
  info "Installing version: $TAG"
else
  info "Fetching latest release..."
  TAG=$(gh_curl "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
  [[ -z "$TAG" ]] && error "Could not resolve latest release. For private repos set LITELENS_ACCESS_TOKEN."
  info "Latest version: $TAG"
fi

# ── resolve asset ID (required for private repo downloads) ───────────────────
info "Resolving asset..."
RELEASE_JSON=$(gh_curl "https://api.github.com/repos/${REPO}/releases/tags/${TAG}")

# Extract asset id — tracks last seen "id" value, stops at matching "name"
ASSET_ID=$(echo "$RELEASE_JSON" | awk -v art="$ARTIFACT" '
  /"id":/ { id=$0; gsub(/[^0-9]/, "", id); last_id=id }
  /"name":/ && index($0, art) { print last_id; exit }
')

[[ -z "$ASSET_ID" ]] && error "Asset '${ARTIFACT}' not found in release ${TAG}. Check the tag name and repo."

# ── download ─────────────────────────────────────────────────────────────────
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

info "Downloading $ARTIFACT..."
if [[ -n "$LITELENS_ACCESS_TOKEN" ]]; then
  # Private repo: use the GitHub API assets endpoint with Accept: application/octet-stream
  curl -fsSL "${CURL_OPTS[@]+"${CURL_OPTS[@]}"}" --progress-bar \
    -H "Authorization: Bearer ${LITELENS_ACCESS_TOKEN}" \
    -H "Accept: application/octet-stream" \
    "https://api.github.com/repos/${REPO}/releases/assets/${ASSET_ID}" \
    -o "$TMP_DIR/$ARTIFACT"
else
  # Public repo: direct download URL
  curl -fsSL "${CURL_OPTS[@]+"${CURL_OPTS[@]}"}" --progress-bar \
    "https://github.com/${REPO}/releases/download/${TAG}/${ARTIFACT}" \
    -o "$TMP_DIR/$ARTIFACT"
fi

# ── verify checksum ──────────────────────────────────────────────────────────
# Fail closed: if no checksum asset is published for this release, refuse to
# install rather than run an unverified binary.
info "Verifying checksum..."
CHECKSUM_ASSET_ID=$(echo "$RELEASE_JSON" | awk -v art="${ARTIFACT}.sha256" '
  /"id":/ { id=$0; gsub(/[^0-9]/, "", id); last_id=id }
  /"name":/ && index($0, art) { print last_id; exit }
')
[[ -z "$CHECKSUM_ASSET_ID" ]] && error "No SHA256 checksum published for '${ARTIFACT}' in release ${TAG}; refusing to install an unverified binary."

if [[ -n "$LITELENS_ACCESS_TOKEN" ]]; then
  curl -fsSL "${CURL_OPTS[@]+"${CURL_OPTS[@]}"}" \
    -H "Authorization: Bearer ${LITELENS_ACCESS_TOKEN}" \
    -H "Accept: application/octet-stream" \
    "https://api.github.com/repos/${REPO}/releases/assets/${CHECKSUM_ASSET_ID}" \
    -o "$TMP_DIR/$ARTIFACT.sha256"
else
  curl -fsSL "${CURL_OPTS[@]+"${CURL_OPTS[@]}"}" \
    "https://github.com/${REPO}/releases/download/${TAG}/${ARTIFACT}.sha256" \
    -o "$TMP_DIR/$ARTIFACT.sha256"
fi

EXPECTED_SHA256=$(tr -d '[:space:]' < "$TMP_DIR/$ARTIFACT.sha256")
ACTUAL_SHA256=$(shasum -a 256 "$TMP_DIR/$ARTIFACT" | awk '{print $1}')
[[ "$EXPECTED_SHA256" == "$ACTUAL_SHA256" ]] || error "Checksum mismatch for ${ARTIFACT}: expected ${EXPECTED_SHA256}, got ${ACTUAL_SHA256}"
success "Checksum verified"

# ── extract ──────────────────────────────────────────────────────────────────
info "Extracting..."
case "$ARTIFACT" in
  *.tar.gz) tar -xzf "$TMP_DIR/$ARTIFACT" -C "$TMP_DIR" ;;
  *.zip)    unzip -q "$TMP_DIR/$ARTIFACT" -d "$TMP_DIR" ;;
esac

# ── install binary ───────────────────────────────────────────────────────────
if [[ "$OS" == "Darwin" ]]; then
  APP_SRC=$(find "$TMP_DIR" -name "litelens.app" -maxdepth 3 | head -1)
  [[ -z "$APP_SRC" ]] && error "litelens.app not found in archive"

  APP_DEST="/Applications/LiteLens.app"
  info "Installing to $APP_DEST..."
  rm -rf "$APP_DEST"
  cp -r "$APP_SRC" "$APP_DEST"

  info "Applying ad-hoc code signature..."
  xattr -cr "$APP_DEST"
  codesign --force --deep --sign - "$APP_DEST"

  ln -sf "$APP_DEST/Contents/MacOS/litelens" "$INSTALL_DIR/$BIN_NAME" 2>/dev/null || true

  success "LiteLens installed to $APP_DEST"
  echo "  Open from Applications or run: open /Applications/LiteLens.app"

else
  BIN_SRC=$(find "$TMP_DIR" -name "$BIN_NAME" -type f | head -1)
  [[ -z "$BIN_SRC" ]] && error "Binary not found in archive"

  info "Installing to $INSTALL_DIR/$BIN_NAME..."
  # Copy to a staging file in the same directory, then rename it into place.
  # rename(2) is atomic and doesn't touch the destination's inode, so a
  # currently-running $BIN_NAME (e.g. self-update) keeps executing the old
  # inode instead of hitting "text file busy" or a truncated binary from
  # an in-place cp/write.
  STAGED="$INSTALL_DIR/.$BIN_NAME.new"
  if [[ -w "$INSTALL_DIR" ]]; then
    cp "$BIN_SRC" "$STAGED"
    chmod +x "$STAGED"
    mv -f "$STAGED" "$INSTALL_DIR/$BIN_NAME"
  else
    sudo cp "$BIN_SRC" "$STAGED"
    sudo chmod +x "$STAGED"
    sudo mv -f "$STAGED" "$INSTALL_DIR/$BIN_NAME"
  fi

  # Install the helper binary if present (non-blocking; skip gracefully if missing,
  # e.g. older releases that predate the helper).
  HELPER_SRC=$(find "$TMP_DIR" -name "litelens-install-helper" -type f | head -1)
  if [[ -n "$HELPER_SRC" ]]; then
    HELPER_NAME="litelens-install-helper"
    HELPER_STAGED="$INSTALL_DIR/.$HELPER_NAME.new"
    if [[ -w "$INSTALL_DIR" ]]; then
      cp "$HELPER_SRC" "$HELPER_STAGED"
      chmod +x "$HELPER_STAGED"
      mv -f "$HELPER_STAGED" "$INSTALL_DIR/$HELPER_NAME"
    else
      sudo cp "$HELPER_SRC" "$HELPER_STAGED"
      sudo chmod +x "$HELPER_STAGED"
      sudo mv -f "$HELPER_STAGED" "$INSTALL_DIR/$HELPER_NAME"
    fi
  else
    echo "⚠ litelens-install-helper not found in release archive; auto-update will be unavailable until reinstalled from a release that includes it" >&2
  fi

  # ── system dependencies (Debian/Ubuntu only) ─────────────────────────────
  if command -v dpkg &>/dev/null && [[ -f /etc/os-release ]]; then
    . /etc/os-release
    if [[ "$ID" == "ubuntu" || "$ID" == "debian" || "$ID_LIKE" == *"ubuntu"* || "$ID_LIKE" == *"debian"* ]]; then
      MISSING_PKGS=()
      for pkg in libgtk-3-0 libwebkit2gtk-4.1-0; do
        if ! dpkg -s "$pkg" &>/dev/null 2>&1; then
          MISSING_PKGS+=("$pkg")
        fi
      done
      if [[ ${#MISSING_PKGS[@]} -gt 0 ]]; then
        info "Installing missing dependencies: ${MISSING_PKGS[*]}"
        sudo apt-get update -qq
        if ! sudo apt-get install -y "${MISSING_PKGS[@]}"; then
          warn "apt-get install failed for: ${MISSING_PKGS[*]}"
          if [[ " ${MISSING_PKGS[*]} " == *" libwebkit2gtk-4.1-0 "* ]]; then
            warn "Attempting libwebkit2gtk-4.0-37 fallback (Ubuntu 22.04 and earlier do not package 4.1)..."
            sudo apt-get install -y libwebkit2gtk-4.0-37 || warn "libwebkit2gtk-4.0-37 fallback also failed; app may not run"
          fi
        else
          success "System dependencies installed"
        fi
      else
        success "System dependencies already installed"
      fi
    fi
  fi

  # ── icon (Linux) ──────────────────────────────────────────────────────────
  ICON_SRC=$(find "$TMP_DIR" -name "appicon.png" -type f | head -1)
  if [[ -n "$ICON_SRC" ]]; then
    ICON_DIR="$HOME/.local/share/icons/hicolor/256x256/apps"
    mkdir -p "$ICON_DIR"
    cp "$ICON_SRC" "$ICON_DIR/litelens.png"
    if command -v gtk-update-icon-cache &>/dev/null; then
      gtk-update-icon-cache -f -t "$HOME/.local/share/icons/hicolor" 2>/dev/null || true
    fi
  fi

  # ── desktop entry (Linux) ─────────────────────────────────────────────────
  mkdir -p "$DESKTOP_DIR"
  cat > "$DESKTOP_DIR/litelens.desktop" << EOF
[Desktop Entry]
Name=LiteLens
Comment=Kubernetes Cluster Manager
Exec=$INSTALL_DIR/$BIN_NAME
Icon=litelens
Type=Application
Categories=Development;Network;
StartupNotify=true
EOF

  if command -v update-desktop-database &>/dev/null; then
    update-desktop-database "$DESKTOP_DIR"
  fi

  success "LiteLens $TAG installed to $INSTALL_DIR/$BIN_NAME"
  echo "  Run from terminal: litelens"
  echo "  Or find it in your application launcher"
fi
