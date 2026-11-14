#!/bin/bash

# Uninstaller for YggLazy-cli
# This script removes all components installed by install.sh

if [[ $EUID -ne 0 ]]; then
  echo "Please run the uninstaller with sudo" >&2
  exit 1
fi

echo "=== Uninstalling YggLazy-cli ==="
echo ""

# Define file paths (must match install.sh PREFIX=/usr/local)
OLD_BINARY="/usr/local/bin/ygg-lazy-cli"
BINARY="/usr/local/bin/ygglazy"
ICON="/usr/local/share/icons/ygglazycli.svg"
DESKTOP="/usr/local/share/applications/ygg-lazy-cli.desktop"
UNINSTALLER="/usr/local/bin/ygglazy-uninstall"

REMOVED_COUNT=0

# Remove binary (legacy name)
if [[ -f "$OLD_BINARY" ]]; then
  echo "✓ Removing old binary: $OLD_BINARY"
  rm -f "$OLD_BINARY"
  ((REMOVED_COUNT++))
else
  echo "- Old binary not found (already removed?): $OLD_BINARY"
fi

# Remove binary (current name)
if [[ -f "$BINARY" ]]; then
  echo "✓ Removing binary: $BINARY"
  rm -f "$BINARY"
  ((REMOVED_COUNT++))
else
  echo "- Binary not found (already removed?): $BINARY"
fi

# Remove icon
if [[ -f "$ICON" ]]; then
  echo "✓ Removing icon: $ICON"
  rm -f "$ICON"
  ((REMOVED_COUNT++))
else
  echo "- Icon not found (already removed?): $ICON"
fi

# Remove desktop file
if [[ -f "$DESKTOP" ]]; then
  echo "✓ Removing desktop shortcut: $DESKTOP"
  rm -f "$DESKTOP"
  ((REMOVED_COUNT++))
else
  echo "- Desktop file not found (already removed?): $DESKTOP"
fi

# Remove auto-generated uninstaller (legacy name)
if [[ -f "/usr/local/bin/ygg-lazy-cli-uninstall" ]]; then
  echo "✓ Removing legacy uninstaller"
  rm -f "/usr/local/bin/ygg-lazy-cli-uninstall"
  ((REMOVED_COUNT++))
fi

# Remove current uninstaller
if [[ -f "$UNINSTALLER" ]]; then
  echo "✓ Removing uninstaller: $UNINSTALLER"
  rm -f "$UNINSTALLER"
  ((REMOVED_COUNT++))
fi

# Update desktop database
echo ""
echo "Updating desktop database..."
update-desktop-database /usr/local/share/applications/ 2>/dev/null || true

echo ""
if [[ $REMOVED_COUNT -eq 0 ]]; then
  echo "No YggLazy-cli files were found. Either it's not installed or already removed."
else
  echo "Done! Removed $REMOVED_COUNT file(s)."
  echo "YggLazy-cli has been successfully uninstalled."
fi