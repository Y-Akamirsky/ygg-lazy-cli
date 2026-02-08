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
GO_VERSION="1.25.6"

PREFIX="/usr/local"
BIN_PATH="$PREFIX/bin/$APP_NAME"
ICON_PATH="$PREFIX/share/icons/ygglazycli.svg"
DESKTOP_PATH="/usr/share/applications/ygg-lazy-cli.desktop"

REAL_USER="${SUDO_USER:-$USER}"
REAL_HOME="$(getent passwd "$REAL_USER" | cut -d: -f6)"

CACHE_BASE="${XDG_CACHE_HOME:-$REAL_HOME/.cache}"
CACHE_DIR="$CACHE_BASE/$APP_NAME-installer"
GO_ROOT="$CACHE_DIR/go-$GO_VERSION"
BUILD_DIR="$CACHE_DIR/build"

ICON_URL="https://raw.githubusercontent.com/$REPO/main/linux-install/ygglazycli.svg"
DESKTOP_URL="https://raw.githubusercontent.com/$REPO/main/linux-install/ygg-lazy-cli.desktop"

### UTILS ###
log() { echo "→ $*"; }
die() { echo "✗ $*" >&2; exit 1; }
cleanup_on_error() {
  if [[ -d "$CACHE_DIR" ]]; then
    log "Cleaning up cache directory due to error"
    rm -rf "$CACHE_DIR"
  fi
}
trap cleanup_on_error ERR

### CLEANUP PHASE ###
cleanup_old_bin() {
  if [ -f /usr/local/bin/ygg-lazy-cli ]; then
    echo "Removing old binary (overcomplicated name)..."
    rm /usr/local/bin/ygg-lazy-cli
  fi
  if [ -f /usr/local/bin/ygg-lazy-cli-uninstall ]; then
    echo "Removing old uninstaller (overcomplicated name)..."
    rm /usr/local/bin/ygg-lazy-cli-uninstall
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

  log "Installing binary"
  install -Dm755 "$BUILD_DIR/$APP_NAME" "$BIN_PATH"

  log "Installing icon"
  install -Dm644 "$CACHE_DIR/ygglazycli.svg" "$ICON_PATH" 2>/dev/null || true

  log "Installing desktop entry"
  install -Dm644 "$CACHE_DIR/ygg-lazy-cli.desktop" "$DESKTOP_PATH" 2>/dev/null || true

  # Generate uninstall script
  log "Creating uninstaller"
  cat > /usr/local/bin/ygglazy-uninstall << 'UNINSTALL_EOF'
#!/bin/bash

# Uninstaller for YggLazy-cli

if [ "$EUID" -ne 0 ]; then
  echo "It requires root"
  exit 1
fi

echo "=== Uninstalling YggLazy-cli ==="

# Remove binary
if [ -f /usr/local/bin/ygg-lazy-cli ]; then
  echo "Removing old binary..."
  rm /usr/local/bin/ygg-lazy-cli
fi
if [ -f /usr/local/bin/ygglazy-uninstall ]; then
  echo "Removing binary..."
  rm /usr/local/bin/ygglazy
fi

# Remove icon
if [ -f /usr/local/share/icons/ygglazycli.svg ]; then
  echo "Removing icon..."
  rm /usr/local/share/icons/ygglazycli.svg
fi

# Remove desktop file
if [ -f /usr/share/applications/ygg-lazy-cli.desktop ]; then
  echo "Removing menu shortcut..."
  rm /usr/share/applications/ygg-lazy-cli.desktop
fi

# Update desktop database
update-desktop-database /usr/share/applications/ 2>/dev/null

# Remove this uninstaller
echo "Removing uninstaller..."
rm /usr/local/bin/ygglazy-uninstall

echo "Done! YggLazy-cli has been uninstalled."
UNINSTALL_EOF

  chmod +x /usr/local/bin/ygglazy-uninstall

  command -v update-desktop-database >/dev/null && \
    update-desktop-database /usr/share/applications || true

  log "Installation complete"

  # Clean up temp script if it was created from pipe
  [[ -n "${TEMP_SCRIPT_CLEANUP:-}" ]] && rm -f "$TEMP_SCRIPT_CLEANUP"

  exit 0
fi


### USER PHASE ###
log "Preparing build environment"
mkdir -p "$CACHE_DIR" "$BUILD_DIR"

### GO TOOLCHAIN ###
need_go=true
if command -v go >/dev/null 2>&1; then
  if go version | grep -q "go$GO_VERSION"; then
    need_go=false
  fi
fi

if $need_go; then
  log "Fetching Go $GO_VERSION locally"
  # Clean up any previous Go installation attempts
  rm -rf "$GO_ROOT" "$CACHE_DIR/go"
  mkdir -p "$CACHE_DIR"

  curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" \
    | tar -xz -C "$CACHE_DIR"
  mv "$CACHE_DIR/go" "$GO_ROOT"
  export GOROOT="$GO_ROOT"
  export PATH="$GOROOT/bin:$PATH"
fi

command -v go >/dev/null || die "Go toolchain not available"

### BUILD ###
log "Cloning repository"
rm -rf "$BUILD_DIR"
git clone "https://github.com/$REPO.git" "$BUILD_DIR"

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
  export TEMP_SCRIPT_CLEANUP="$SCRIPT_PATH"
  exec sudo -E bash "$SCRIPT_PATH" --install "$BUILD_DIR" "$CACHE_DIR" "$SCRIPT_PATH"
elif command -v systemd-run >/dev/null; then
  export TEMP_SCRIPT_CLEANUP="$SCRIPT_PATH"
  exec systemd-run --quiet --pipe --wait --pty \
    bash "$SCRIPT_PATH" --install "$BUILD_DIR" "$CACHE_DIR" "$SCRIPT_PATH"
elif command -v doas >/dev/null; then
  export TEMP_SCRIPT_CLEANUP="$SCRIPT_PATH"
  exec doas bash "$SCRIPT_PATH" --install "$BUILD_DIR" "$CACHE_DIR" "$SCRIPT_PATH"
else
  die "No supported privilege escalation method found"
fi
