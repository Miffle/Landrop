#!/usr/bin/env sh
# Landrop installer
# Usage: sh <(wget -O - https://raw.githubusercontent.com/Miffle/Landrop/main/install.sh)
#    or: sh <(curl -fsSL https://raw.githubusercontent.com/Miffle/Landrop/main/install.sh)

set -e

REPO="Miffle/Landrop"
INSTALL_DIR="${LANDROP_DIR:-$HOME/landrop}"
BIN_NAME="landrop"
API_URL="https://api.github.com/repos/${REPO}/releases/latest"

# ---------- helpers ----------
info()  { printf '\033[1;34m[landrop]\033[0m %s\n' "$*"; }
ok()    { printf '\033[1;32m[landrop]\033[0m %s\n' "$*"; }
err()   { printf '\033[1;31m[landrop] ERROR:\033[0m %s\n' "$*" >&2; exit 1; }

need() {
  command -v "$1" >/dev/null 2>&1 || err "Required tool not found: $1"
}

# ---------- detect OS / arch ----------
detect_target() {
  OS="$(uname -s)"
  ARCH="$(uname -m)"

  case "$OS" in
    Linux) OS="linux" ;;
    Darwin) err "macOS is not supported yet." ;;
    MINGW*|MSYS*|CYGWIN*) err "Use the .exe from the GitHub Releases page on Windows." ;;
    *) err "Unsupported OS: $OS" ;;
  esac

  case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    armv7*|armhf) ARCH="armv7" ;;
    *) err "Unsupported architecture: $ARCH" ;;
  esac

  TARGET="${OS}-${ARCH}"
  info "Detected target: $TARGET"
}

# ---------- fetch latest release ----------
fetch_release() {
  need wget

  info "Fetching latest release info from GitHub..."
  RELEASE_JSON="$(wget -qO- "$API_URL")" || err "Failed to fetch release info. Check your internet connection."

  TAG="$(printf '%s' "$RELEASE_JSON" | grep '"tag_name"' | head -1 | cut -d'"' -f4)"
  [ -n "$TAG" ] || err "Could not parse release tag."

  info "Latest release: $TAG"

  # Build expected asset name
  ASSET_NAME="${BIN_NAME}-${TARGET}"
  DOWNLOAD_URL="$(printf '%s' "$RELEASE_JSON" | grep "browser_download_url" | grep "${ASSET_NAME}" | head -1 | cut -d'"' -f4)"
  [ -n "$DOWNLOAD_URL" ] || err "No binary found for target '${TARGET}' in release ${TAG}."
}

# ---------- install ----------
do_install() {
  mkdir -p "$INSTALL_DIR"

  BINARY_PATH="${INSTALL_DIR}/${BIN_NAME}"
  TEMP_PATH="${BINARY_PATH}.tmp"

  info "Downloading $ASSET_NAME..."
  wget -qO "$TEMP_PATH" "$DOWNLOAD_URL" || err "Download failed."

  chmod +x "$TEMP_PATH"
  mv "$TEMP_PATH" "$BINARY_PATH"

  ok "Installed to $BINARY_PATH"
}

# ---------- optional systemd service ----------
setup_systemd() {
  if ! command -v systemctl >/dev/null 2>&1; then
    return
  fi

  printf '\033[1;34m[landrop]\033[0m Install as systemd service? [Y/n] '
  read -r REPLY
  case "$REPLY" in
   [Nn]*)
        info "Skipped systemd setup."
        ;;
   *)
      SERVICE_FILE="/etc/systemd/system/landrop.service"
      info "Writing $SERVICE_FILE (requires sudo)..."
      sudo tee "$SERVICE_FILE" >/dev/null <<EOF
[Unit]
Description=Landrop file transfer server
After=network.target

[Service]
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/${BIN_NAME}
Restart=on-failure
User=$(whoami)

[Install]
WantedBy=multi-user.target
EOF
      sudo systemctl daemon-reload
      sudo systemctl enable --now landrop
      ok "Service enabled and started."
      ;;
  esac
}

# ---------- add to PATH hint ----------
path_hint() {
  case ":$PATH:" in
    *":${INSTALL_DIR}:"*) ;;
    *)
      info "Add Landrop to your PATH:"
      printf "  export PATH=\"\$PATH:%s\"\n" "$INSTALL_DIR"
      ;;
  esac
}

# ---------- main ----------
detect_target
fetch_release
do_install
setup_systemd
path_hint

ok "Done! Run: ${INSTALL_DIR}/${BIN_NAME}"
info "Open http://localhost:6437 in your browser."
