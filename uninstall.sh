#!/bin/bash

# Uninstaller for YggLazy-cli
# This script removes all components installed by install.sh

if [ "$EUID" -ne 0 ]; then
  echo "Please run the uninstaller with sudo"
  exit 1
fi

echo "=== Uninstalling YggLazy-cli ==="
echo ""

# Define file paths
BINARY="/usr/local/bin/ygg-lazy-cli"
ICON="/usr/local/share/icons/ygglazycli.svg"
DESKTOP="/usr/share/applications/ygg-lazy-cli.desktop"
UNINSTALLER="/usr/local/bin/ygg-lazy-cli-uninstall"

REMOVED_COUNT=0

# Remove binary
if [ -f "$BINARY" ]; then
  echo "✓ Removing binary: $BINARY"
  rm "$BINARY"
  REMOVED_COUNT=$((REMOVED_COUNT + 1))
else
  echo "- Binary not found (already removed?): $BINARY"
fi

# Remove icon
if [ -f "$ICON" ]; then
  echo "✓ Removing icon: $ICON"
  rm "$ICON"
  REMOVED_COUNT=$((REMOVED_COUNT + 1))
else
  echo "- Icon not found (already removed?): $ICON"
fi

# Remove desktop file
if [ -f "$DESKTOP" ]; then
  echo "✓ Removing desktop shortcut: $DESKTOP"
  rm "$DESKTOP"
  REMOVED_COUNT=$((REMOVED_COUNT + 1))
else
  echo "- Desktop file not found (already removed?): $DESKTOP"
fi

# Remove auto-generated uninstaller if exists
if [ -f "$UNINSTALLER" ]; then
  echo "✓ Removing auto-generated uninstaller: $UNINSTALLER"
  rm "$UNINSTALLER"
  REMOVED_COUNT=$((REMOVED_COUNT + 1))
fi

# Update desktop database
echo ""
echo "Updating desktop database..."
update-desktop-database /usr/share/applications/ 2>/dev/null

echo ""
if [ $REMOVED_COUNT -eq 0 ]; then
  echo "No YggLazy-cli files were found. Either it's not installed or already removed."
else
  echo "Done! Removed $REMOVED_COUNT file(s)."
  echo "YggLazy-cli has been successfully uninstalled."
fi
