#!/usr/bin/env bash

# If running from pipe, save to temp file and re-execute
if [[ ! -f "$0" ]] || [[ "$0" == "bash" ]] || [[ "$0" == "/dev/fd/"* ]] || [[ "$0" == "/proc/"* ]] || [[ "$0" == "-bash" ]]; then
  if [[ "${REEXEC_DONE:-}" != "1" ]]; then
    TEMP_SCRIPT=$(mktemp)
    cat > "$TEMP_SCRIPT"
    chmod +x "$TEMP_SCRIPT"
    export REEXEC_DONE=1
    exec bash "$TEMP_SCRIPT" "$@"
  fi
fi

set -euo pipefail

### CONFIG ###
APP_NAME="ygglazy"
REPO="Y-Akamirsky/ygg-lazy-cli"
REPO_URL="https://github.com/${REPO}.git"
BRANCH="dev"
# Latest Go stable as of 2026-11-14
GO_VERSION="1.27.4"
# Actual sha256 for go1.27.4.linux-amd64.tar.gz from https://go.dev/dl/
GO_SHA256_AMD64="7226736c2339719cda28fbf3a54d71aa63bcefe376e98cef00034268dab24840"

PREFIX="/usr/local"
BIN_PATH="${PREFIX}/bin/${APP_NAME}"
ICON_PATH="${PREFIX}/share/icons/ygglazycli.svg"
DESKTOP_PATH="${PREFIX}/share/applications/ygg-lazy-cli.desktop"

REAL_USER="${SUDO_USER:-$USER}"
REAL_HOME="$(getent passwd "$REAL_USER" | cut -d: -f6)"

CACHE_BASE="${XDG_CACHE_HOME:-$REAL_HOME/.cache}"
CACHE_DIR="$CACHE_BASE/$APP_NAME-installer"
GO_ROOT="$CACHE_DIR/go-$GO_VERSION"
BUILD_DIR="$CACHE_DIR/build"

DESKTOP_URL="https://raw.githubusercontent.com/$REPO/$BRANCH/linux-install/ygg-lazy-cli.desktop"
ICON_URL="https://raw.githubusercontent.com/$REPO/$BRANCH/linux-install/ygglazycli.svg"
UNINSTALL_URL="https://raw.githubusercontent.com/$REPO/$BRANCH/uninstall.sh"

### UTILS ###
log() { echo "→ $*"; }
die() { echo "✗ $*" >&2; exit 1; }

cleanup_cache() {
  if [[ -d "$CACHE_DIR" ]]; then
    log "Cleaning up cache directory"
    rm -rf "$CACHE_DIR"
  fi
}
trap cleanup_cache ERR

### CLEANUP PHASE ###
cleanup_old_bin() {
  if [[ -f /usr/local/bin/ygg-lazy-cli ]]; then
    log "Removing old binary (legacy name)..."
    rm -f /usr/local/bin/ygg-lazy-cli
  fi
  if [[ -f /usr/local/bin/ygg-lazy-cli-uninstall ]]; then
    log "Removing old uninstaller (legacy name)..."
    rm -f /usr/local/bin/ygg-lazy-cli-uninstall
  fi
}

### ROOT ESCALATION (INSTALL PHASE) ###
if [[ "${1:-}" == "--install" ]]; then
  [[ $EUID -eq 0 ]] || die "Install phase requires root"

  BUILD_DIR="$2"
  CACHE_DIR="$3"
  SCRIPT_PATH="$4"

  [[ -x "$BUILD_DIR/$APP_NAME" ]] || die "Binary not found: $BUILD_DIR/$APP_NAME"

  cleanup_old_bin

  log "Installing binary to $BIN_PATH"
  install -Dm755 "$BUILD_DIR/$APP_NAME" "$BIN_PATH"

  log "Installing icon to $ICON_PATH"
  install -Dm644 "$CACHE_DIR/ygglazycli.svg" "$ICON_PATH" 2>/dev/null || true

  log "Installing desktop entry to $DESKTOP_PATH"
  install -Dm644 "$CACHE_DIR/ygg-lazy-cli.desktop" "$DESKTOP_PATH" 2>/dev/null || true

  # Install standalone uninstaller
  log "Installing uninstaller"
  install -Dm755 "$CACHE_DIR/uninstall.sh" "/usr/local/bin/${APP_NAME}-uninstall" 2>/dev/null || {
    log "Creating uninstaller from template"
    cat > "/usr/local/bin/${APP_NAME}-uninstall" << 'UNINSTALL_EOF'
#!/bin/bash

# Uninstaller for YggLazy-cli
# Installed by install.sh

if [[ $EUID -ne 0 ]]; then
  echo "Please run the uninstaller with sudo" >&2
  exit 1
fi

echo "=== Uninstalling YggLazy-cli ==="

BINARY="/usr/local/bin/ygglazy"
ICON="/usr/local/share/icons/ygglazycli.svg"
DESKTOP="/usr/local/share/applications/ygg-lazy-cli.desktop"
UNINSTALLER="/usr/local/bin/ygglazy-uninstall"

REMOVED=0

# Remove binary
if [[ -f "$BINARY" ]]; then
  echo "✓ Removing binary: $BINARY"
  rm -f "$BINARY"
  ((REMOVED++))
else
  echo "- Binary not found: $BINARY"
fi

# Remove icon
if [[ -f "$ICON" ]]; then
  echo "✓ Removing icon: $ICON"
  rm -f "$ICON"
  ((REMOVED++))
else
  echo "- Icon not found: $ICON"
fi

# Remove desktop file
if [[ -f "$DESKTOP" ]]; then
  echo "✓ Removing desktop shortcut: $DESKTOP"
  rm -f "$DESKTOP"
  ((REMOVED++))
else
  echo "- Desktop file not found: $DESKTOP"
fi

# Remove this uninstaller
if [[ -f "$UNINSTALLER" ]]; then
  echo "✓ Removing uninstaller: $UNINSTALLER"
  rm -f "$UNINSTALLER"
  ((REMOVED++))
fi

# Update desktop database
echo "Updating desktop database..."
update-desktop-database /usr/local/share/applications/ 2>/dev/null || true

echo ""
if [[ $REMOVED -eq 0 ]]; then
  echo "No YggLazy-cli files were found (already removed?)"
else
  echo "Done! Removed $REMOVED file(s)."
fi
UNINSTALL_EOF
    chmod +x "/usr/local/bin/${APP_NAME}-uninstall"
  }

  command -v update-desktop-database >/dev/null && \
    update-desktop-database /usr/local/share/applications || true

  log "Installation complete"
  exit 0
fi


### USER PHASE ###
log "Preparing build environment"
mkdir -p "$CACHE_DIR" "$BUILD_DIR"

### FETCH STANDALONE UNINSTALLER ###
log "Fetching standalone uninstaller"
curl -fsSL -o "$CACHE_DIR/uninstall.sh" \
  "$UNINSTALL_URL" || true

### GO TOOLCHAIN ###
need_go=true
if command -v go >/dev/null 2>&1; then
  if go version | grep -q "go$GO_VERSION"; then
    need_go=false
  fi
fi

if $need_go; then
  log "Fetching Go $GO_VERSION locally"
  rm -rf "$GO_ROOT" "$CACHE_DIR/go"
  mkdir -p "$CACHE_DIR"

  GO_TARBALL="go${GO_VERSION}.linux-amd64.tar.gz"
  GO_URL="https://go.dev/dl/${GO_TARBALL}"

  log "Downloading $GO_URL"
  curl -fsSL -o "$CACHE_DIR/$GO_TARBALL" "$GO_URL"

  # Verify checksum (optional but recommended)
  if command -v sha256sum >/dev/null; then
    EXPECTED="$GO_SHA256_AMD64"
    ACTUAL=$(sha256sum "$CACHE_DIR/$GO_TARBALL" | cut -d' ' -f1)
    if [[ "$ACTUAL" != "$EXPECTED" ]]; then
      die "Go tarball checksum mismatch! Expected $EXPECTED, got $ACTUAL"
    fi
    log "Checksum verified"
  fi

  tar -xzf "$CACHE_DIR/$GO_TARBALL" -C "$CACHE_DIR"
  mv "$CACHE_DIR/go" "$GO_ROOT"
  export GOROOT="$GO_ROOT"
  export PATH="$GOROOT/bin:$PATH"
fi

command -v go >/dev/null || die "Go toolchain not available"

### BUILD ###
log "Cloning repository (branch: $BRANCH)"
rm -rf "$BUILD_DIR"
git clone --branch "$BRANCH" --depth 1 "$REPO_URL" "$BUILD_DIR"

log "Building $APP_NAME"
cd "$BUILD_DIR"
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$APP_NAME"

### FETCH ASSETS ###
log "Fetching desktop assets"
curl -fsSL -o "$CACHE_DIR/ygglazycli.svg" "$ICON_URL" || true
curl -fsSL -o "$CACHE_DIR/ygg-lazy-cli.desktop" "$DESKTOP_URL" || true

### ESCALATE ONCE ###
SCRIPT_PATH="$(realpath "$0")"

# If already root, just run install phase directly
if [[ $EUID -eq 0 ]]; then
  log "Running with root privileges"
  exec bash "$SCRIPT_PATH" --install "$BUILD_DIR" "$CACHE_DIR" "$SCRIPT_PATH"
fi

log "Requesting privileges to install system-wide"

if command -v sudo >/dev/null; then
  exec sudo -E bash "$SCRIPT_PATH" --install "$BUILD_DIR" "$CACHE_DIR" "$SCRIPT_PATH"
elif command -v systemd-run >/dev/null; then
  exec systemd-run --quiet --pipe --wait --pty \
    bash "$SCRIPT_PATH" --install "$BUILD_DIR" "$CACHE_DIR" "$SCRIPT_PATH"
elif command -v doas >/dev/null; then
  exec doas bash "$SCRIPT_PATH" --install "$BUILD_DIR" "$CACHE_DIR" "$SCRIPT_PATH"
else
  die "No supported privilege escalation method found (sudo, systemd-run, doas)"
fi
