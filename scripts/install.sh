#!/usr/bin/env bash
set -euo pipefail

REPO="litelensapp/litelens"
BIN_NAME="litelens"
INSTALL_DIR="/usr/local/bin"
DESKTOP_DIR="$HOME/.local/share/applications"

# Base URL for all release lookups (releases/latest, releases/tags/{tag},
# releases/download/..., releases/assets/...). Mirrors GetReleasesBaseURL in
# internal/config/env.go — overridable via the same APP_VERSION_RELEASES_BASE_URL
# environment variable. Defaults to the public github.com host; if REPO is
# private and LITELENS_ACCESS_TOKEN is required to access it, set this to
# https://api.github.com/repos/${REPO} instead (the public github.com host
# rejects Bearer tokens on private-repo release URLs).
RELEASES_BASE_URL="${APP_VERSION_RELEASES_BASE_URL:-https://github.com/${REPO}}"

# ── colours ──────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
info()    { echo -e "${GREEN}→${NC} $*"; }
warn()    { echo -e "${YELLOW}⚠${NC}  $*"; }
error()   { echo -e "${RED}✗${NC}  $*" >&2; exit 1; }
success() { echo -e "${GREEN}✓${NC} $*"; }

# ── jq availability check ────────────────────────────────────────────────────
command -v jq &>/dev/null || error "jq is required but not installed. Install it and re-run."

# ── detect OS + arch ─────────────────────────────────────────────────────────
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
  Linux)
    case "$ARCH" in
      x86_64)  PLATFORM_OS="linux"; PLATFORM_ARCH="amd64" ;;
      *)        error "Unsupported Linux architecture: $ARCH" ;;
    esac
    ;;
  Darwin)
    case "$ARCH" in
      arm64)   PLATFORM_OS="darwin"; PLATFORM_ARCH="arm64" ;;
      x86_64)  PLATFORM_OS="darwin"; PLATFORM_ARCH="amd64" ;;
      *)        error "Unsupported macOS architecture: $ARCH" ;;
    esac
    ;;
  *)
    error "Unsupported OS: $OS (Windows users: download the .exe from GitHub Releases)"
    ;;
esac

# ── auth ─────────────────────────────────────────────────────────────────────
# For private repos, export LITELENS_ACCESS_TOKEN and also point
# RELEASES_BASE_URL at the API host (see its definition above) in
# ~/.bashrc or ~/.zshrc:
#   export LITELENS_ACCESS_TOKEN=ghp_xxx
#   export APP_VERSION_RELEASES_BASE_URL=https://api.github.com/repos/litelensapp/litelens
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
    # Surface GitHub's own explanation instead of guessing — a 401/403 can mean
    # an invalid/expired token, missing scopes, rate limiting, or an org policy
    # rejecting the token (e.g. fine-grained PAT lifetime limits), and each of
    # those needs a different fix.
    local gh_message
    gh_message=$(echo "$body" | grep -o '"message": *"[^"]*"' | head -1 | sed -E 's/"message": *"//; s/"$//')
    if [[ -n "$gh_message" ]]; then
      error "GitHub API returned $http_code: $gh_message"
    else
      error "GitHub API returned $http_code — token is invalid or missing required permissions."
    fi
  elif [[ "$http_code" == "404" ]]; then
    if [[ -z "$LITELENS_ACCESS_TOKEN" ]]; then
      error "GitHub API returned 404 — check the repo name or that the release/tag exists.\nIf this is a private repo, export a token and the API base URL, then re-run:\n  export LITELENS_ACCESS_TOKEN=ghp_xxx\n  export APP_VERSION_RELEASES_BASE_URL=https://api.github.com/repos/${REPO}"
    else
      error "GitHub API returned 404 — check the repo name or that the release/tag exists."
    fi
  elif [[ "$http_code" != "200" ]]; then
    error "GitHub API request failed with HTTP $http_code"
  fi

  echo "$body"
}

# ── setup temp directory ────────────────────────────────────────────────────
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

# ── resolve version ───────────────────────────────────────────────────────────
# Priority: positional arg > VERSION env var > latest release
VERSION="${1:-${VERSION:-}}"

if [[ -n "$VERSION" ]]; then
  TAG="$VERSION"
  info "Installing version: $TAG"
elif [[ -n "$LITELENS_ACCESS_TOKEN" ]]; then
  info "Fetching latest release..."
  TAG=$(gh_curl "${RELEASES_BASE_URL}/releases/latest" \
    | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
  [[ -z "$TAG" ]] && error "Could not resolve latest release."
  info "Latest version: $TAG"
else
  # Public repo, no token: resolve the latest tag via the releases page
  # redirect instead of the GitHub API. The unauthenticated API is capped at
  # 60 requests/hour *per source IP*, which is shared by everyone behind the
  # same NAT/office network/CI runner pool — easy to exhaust without ever
  # having made 60 requests yourself, and it fails with a 403 that looks
  # identical to a bad token. The releases page redirect isn't rate-limited.
  info "Fetching latest release..."
  FINAL_URL=$(curl -fsSL "${CURL_OPTS[@]+"${CURL_OPTS[@]}"}" -o /dev/null -w '%{url_effective}' \
    "${RELEASES_BASE_URL}/releases/latest") \
    || error "Could not reach ${RELEASES_BASE_URL}/releases/latest"
  TAG="${FINAL_URL##*/tag/}"
  [[ -z "$TAG" || "$TAG" == "$FINAL_URL" ]] && error "Could not resolve latest release for ${REPO}."
  info "Latest version: $TAG"
fi

# ── resolve release metadata + manifest ───────────────────────────────────────
# manifest.json is the single source of truth for the artifact filename and
# SHA256 checksum for each platform — generated from the actual build matrix
# output, avoiding hardcoded per-platform guesses drifting from what was built.
RELEASE_JSON=""
ASSET_ID=""
if [[ -n "$LITELENS_ACCESS_TOKEN" ]]; then
  info "Resolving release..."
  RELEASE_JSON=$(gh_curl "${RELEASES_BASE_URL}/releases/tags/${TAG}")

  # Extract asset id — tracks last seen "id" value, stops at matching "name"
  MANIFEST_ASSET_ID=$(echo "$RELEASE_JSON" | awk -v art="manifest.json" '
    /"id":/ { id=$0; gsub(/[^0-9]/, "", id); last_id=id }
    /"name":/ && index($0, art) { print last_id; exit }
  ')
  [[ -z "$MANIFEST_ASSET_ID" ]] && error "manifest.json not found in release ${TAG}. Check the tag name and repo."

  curl -fsSL "${CURL_OPTS[@]+"${CURL_OPTS[@]}"}" \
    -H "Authorization: Bearer ${LITELENS_ACCESS_TOKEN}" \
    -H "Accept: application/octet-stream" \
    "${RELEASES_BASE_URL}/releases/assets/${MANIFEST_ASSET_ID}" \
    -o "$TMP_DIR/manifest.json" \
    || error "Could not download manifest for release ${TAG}."
else
  info "Resolving asset from manifest..."
  curl -fsSL "${CURL_OPTS[@]+"${CURL_OPTS[@]}"}" \
    "${RELEASES_BASE_URL}/releases/download/${TAG}/manifest.json" \
    -o "$TMP_DIR/manifest.json" \
    || error "Could not download manifest for release ${TAG}."
fi

# Use --arg to pass PLATFORM_OS/PLATFORM_ARCH as JSON strings to jq, preventing
# injection if these values were ever derived from untrusted sources.
ARTIFACT=$(jq -r --arg os "$PLATFORM_OS" --arg arch "$PLATFORM_ARCH" \
  '.artifacts[] | select(.os == $os and .arch == $arch) | .filename' \
  "$TMP_DIR/manifest.json")
[[ -z "$ARTIFACT" || "$ARTIFACT" == "null" ]] && \
  error "Platform ${PLATFORM_OS}/${PLATFORM_ARCH} not found in release manifest for ${TAG}."

EXPECTED_SHA256=$(jq -r --arg os "$PLATFORM_OS" --arg arch "$PLATFORM_ARCH" \
  '.artifacts[] | select(.os == $os and .arch == $arch) | .sha256' \
  "$TMP_DIR/manifest.json")
[[ -z "$EXPECTED_SHA256" || "$EXPECTED_SHA256" == "null" ]] && \
  error "No SHA256 checksum in release manifest for ${PLATFORM_OS}/${PLATFORM_ARCH}; refusing to install an unverified binary."

# ── resolve asset ID (required for private repo downloads only) ──────────────
# Public downloads use direct github.com/.../releases/download URLs below and
# never need the API, so skip this (and its share of the rate limit) entirely
# when there's no token.
if [[ -n "$LITELENS_ACCESS_TOKEN" ]]; then
  ASSET_ID=$(echo "$RELEASE_JSON" | awk -v art="$ARTIFACT" '
    /"id":/ { id=$0; gsub(/[^0-9]/, "", id); last_id=id }
    /"name":/ && index($0, art) { print last_id; exit }
  ')

  [[ -z "$ASSET_ID" ]] && error "Asset '${ARTIFACT}' not found in release ${TAG}. Check the tag name and repo."
fi

# ── download ─────────────────────────────────────────────────────────────────
info "Downloading $ARTIFACT..."
if [[ -n "$LITELENS_ACCESS_TOKEN" ]]; then
  # Private repo: use the GitHub API assets endpoint with Accept: application/octet-stream
  curl -fsSL "${CURL_OPTS[@]+"${CURL_OPTS[@]}"}" --progress-bar \
    -H "Authorization: Bearer ${LITELENS_ACCESS_TOKEN}" \
    -H "Accept: application/octet-stream" \
    "${RELEASES_BASE_URL}/releases/assets/${ASSET_ID}" \
    -o "$TMP_DIR/$ARTIFACT"
else
  # Public repo: direct download URL
  curl -fsSL "${CURL_OPTS[@]+"${CURL_OPTS[@]}"}" --progress-bar \
    "${RELEASES_BASE_URL}/releases/download/${TAG}/${ARTIFACT}" \
    -o "$TMP_DIR/$ARTIFACT" \
    || error "Could not download ${ARTIFACT} for release ${TAG}. Check the tag name and repo."
fi

# ── verify checksum ──────────────────────────────────────────────────────────
# Verified against the SHA256 already resolved from manifest.json above —
# fail-closed behavior (refusing to install without a checksum) is enforced
# where EXPECTED_SHA256 is read from the manifest.
info "Verifying checksum..."
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
