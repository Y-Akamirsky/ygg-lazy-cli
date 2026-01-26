#!/bin/bash

# Check if user is root
if [ "$EUID" -ne 0 ]; then
  echo "Please run the installer with sudo (required for copying to /usr/local/bin)"
  exit
fi

BIN_DIR=/usr/local/bin/ygg-lazy-cli
ICON_DIR=/usr/local/share/icons
ICON=ygglazycli.svg
DESKTOP_DIR=/usr/share/applications/ygg-lazy-cli.desktop

REPO="Y-Akamirsky/ygg-lazy-cli"
# Check architecture
ARCH=$(uname -m)
if [[ "$ARCH" == "x86_64" ]]; then
  BINARY_URL="https://raw.githubusercontent.com/$REPO/main/linux-install/ygg-lazy-cli-amd64"
elif [[ "$ARCH" == "aarch64" ]]; then
  BINARY_URL="https://raw.githubusercontent.com/$REPO/main/linux-install/ygg-lazy-cli-arm64"
else
  echo "Architecture $ARCH is not supported... yet."
  exit 1
fi

ICON_URL="https://raw.githubusercontent.com/$REPO/main/linux-install/ygglazycli.svg"
DESKTOP_URL="https://raw.githubusercontent.com/$REPO/main/linux-install/ygg-lazy-cli.desktop"
echo "=== Cheking old version ==="

# Check if old version exists and remove it
if [ -f "$BIN_DIR" ]; then
  echo "Old version found, removing..."
  rm "$BIN_DIR"
fi

echo "=== Installing YggLazy-cli ==="

# 1. Download a binary
echo "Downloading binary..."
curl -L -o $BIN_DIR "$BINARY_URL"
chmod +x $BIN_DIR

# 2. Download an icon
echo "Installing icon..."
mkdir -p $ICON_DIR
curl -L -o $ICON_DIR/$ICON "$ICON_URL"

# 3. Download .desktop file
echo "Creating menu shortcut..."
curl -L -o $DESKTOP_DIR "$DESKTOP_URL"

# Update desktop database (to show icon immediately)
update-desktop-database /usr/share/applications/ 2>/dev/null

echo "Done! Now look for YggLazy-cli in your applications menu."
