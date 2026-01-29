#!/bin/bash

# Check if user is root
if [ "$EUID" -ne 0 ]; then
  echo "Please run the installer with sudo (required for copying to /usr/local/bin)"
  exit 1
fi

BIN_DIR=/usr/local/bin/ygg-lazy-cli
ICON_DIR=/usr/local/share/icons
ICON=ygglazycli.svg
DESKTOP_DIR=/usr/share/applications/ygg-lazy-cli.desktop

REPO="Y-Akamirsky/ygg-lazy-cli"
ICON_URL="https://raw.githubusercontent.com/$REPO/main/linux-install/ygglazycli.svg"
DESKTOP_URL="https://raw.githubusercontent.com/$REPO/main/linux-install/ygg-lazy-cli.desktop"

# Temporary directory for building
BUILD_DIR=$(mktemp -d)
trap "rm -rf $BUILD_DIR" EXIT

# Get actual user info
ACTUAL_USER="${SUDO_USER:-$USER}"
ACTUAL_HOME=$(eval echo ~$ACTUAL_USER)

# Flag to track if we installed 'g'
G_INSTALLED_BY_SCRIPT=0

echo "=== Checking old version ==="

# Check if old version exists and remove it
if [ -f "$BIN_DIR" ]; then
  echo "Old version found, removing..."
  rm "$BIN_DIR"
fi

echo "=== Preparing build environment ==="

# Function to check Go version
check_go_version() {
  if command -v go >/dev/null 2>&1; then
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    REQUIRED_VERSION="1.25.6"

    # Simple version comparison (works for major.minor)
    if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" = "$REQUIRED_VERSION" ]; then
      echo "Found Go version $GO_VERSION (sufficient)"
      return 0
    else
      echo "Found Go version $GO_VERSION (insufficient, need >= $REQUIRED_VERSION)"
      return 1
    fi
  else
    echo "Go is not installed"
    return 1
  fi
}

# Function to install 'g' and latest Go
install_go_with_g() {
  echo "Installing Go using 'g' utility..."

  # Check if 'g' is already installed
  if command -v g >/dev/null 2>&1; then
    echo "'g' utility is already installed, using it..."
  else
    echo "Installing 'g' utility..."
    # Install 'g' for the current user (non-root)
    # Force bash execution regardless of user's default shell
    # Use IGNORE_QUESTIONS=1 for non-interactive installation

    su $ACTUAL_USER -c 'export IGNORE_QUESTIONS=1; curl -sSL https://git.io/g-install | bash' || {
      echo "Error: Failed to install 'g' utility"
      exit 1
    }

    G_INSTALLED_BY_SCRIPT=1

    # Set PATH to include g
    export GOPATH="$ACTUAL_HOME/go"
    export GOROOT="$ACTUAL_HOME/.go"
    export PATH="$GOPATH/bin:$PATH"
  fi

  # Set up 'g' paths
  export GOPATH="$ACTUAL_HOME/go"
  export GOROOT="$ACTUAL_HOME/.go"
  export PATH="$GOPATH/bin:$PATH"

  # Install latest Go version using 'g'
  echo "Installing latest Go version..."
  su $ACTUAL_USER -c 'export PATH="$HOME/.go/bin:$PATH"; bash -c "g install latest"' || {
    echo "Error: Failed to install Go using 'g'"
    exit 1
  }

  echo "Go installed successfully via 'g'"
}

# Check if Go is available and sufficient
if ! check_go_version; then
  install_go_with_g
fi

# Ensure Go is in PATH
if ! command -v go >/dev/null 2>&1; then
  export GOPATH="$ACTUAL_HOME/go"
  export GOROOT="$ACTUAL_HOME/.go"
  export PATH="$GOPATH/bin:$PATH"
fi

# Verify Go is now available
if ! command -v go >/dev/null 2>&1; then
  echo "Error: Go is still not available after installation attempt"
  exit 1
fi

echo "=== Downloading source code ==="

# Download source code
cd "$BUILD_DIR"
echo "Cloning repository..."
git clone "https://github.com/$REPO.git" project || {
  echo "Error: Failed to clone repository"
  exit 1
}

# Change ownership of BUILD_DIR to actual user
chown -R $ACTUAL_USER:$(id -gn $ACTUAL_USER) "$BUILD_DIR"

cd project

echo "=== Compiling YggLazy-cli ==="

# Compile with compatibility flags to avoid CPU instruction issues
echo "Compiling may take a while..."

# Set up environment and compile as user
su $ACTUAL_USER -c "cd '$BUILD_DIR/project' && \
  export GOPATH='$GOPATH' && \
  export GOROOT='$GOROOT' && \
  export PATH='$PATH' && \
  export CGO_ENABLED=0 && \
  go build -ldflags='-s -w' -trimpath -o ygg-lazy-cli ." || {
  echo "Error: Compilation failed"
  exit 1
}

echo "Compilation successful!"

echo "=== Installing YggLazy-cli ==="

# Copy compiled binary
echo "Installing binary..."
cp "$BUILD_DIR/project/ygg-lazy-cli" "$BIN_DIR"
chmod +x "$BIN_DIR"

# Download icon
echo "Installing icon..."
mkdir -p "$ICON_DIR"
curl -L -o "$ICON_DIR/$ICON" "$ICON_URL" || {
  echo "Warning: Failed to download icon"
}

# Download .desktop file
echo "Creating menu shortcut..."
curl -L -o "$DESKTOP_DIR" "$DESKTOP_URL" || {
  echo "Warning: Failed to download .desktop file"
}

# Update desktop database (to show icon immediately)
update-desktop-database /usr/share/applications/ 2>/dev/null

# Clean up 'g' if it was installed by this script
if [ $G_INSTALLED_BY_SCRIPT -eq 1 ]; then
  echo "=== Cleaning up ==="
  echo "Removing 'g' utility (installed by this script)..."
  ACTUAL_USER="${SUDO_USER:-$USER}"
  ACTUAL_HOME=$(eval echo ~$ACTUAL_USER)

  # Remove 'g' and its configuration
  rm -rf "$ACTUAL_HOME/.go"

  # Remove 'g' shell configuration from profile files
  for profile_file in "$ACTUAL_HOME/.bashrc" "$ACTUAL_HOME/.zshrc" "$ACTUAL_HOME/.profile"; do
    if [ -f "$profile_file" ]; then
      # Remove lines added by g-install
      sed -i '/# g-install/,+2d' "$profile_file" 2>/dev/null
      sed -i '/export GOROOT/d' "$profile_file" 2>/dev/null
      sed -i '/export GOPATH/d' "$profile_file" 2>/dev/null
      sed -i '/export PATH.*\.go/d' "$profile_file" 2>/dev/null
    fi
  done

  echo "'g' utility has been removed"
fi

# Generate uninstall script
echo "Creating uninstaller..."
cat > /usr/local/bin/ygg-lazy-cli-uninstall << 'UNINSTALL_EOF'
#!/bin/bash

# Uninstaller for YggLazy-cli

if [ "$EUID" -ne 0 ]; then
  echo "Please run the uninstaller with sudo"
  exit 1
fi

echo "=== Uninstalling YggLazy-cli ==="

# Remove binary
if [ -f /usr/local/bin/ygg-lazy-cli ]; then
  echo "Removing binary..."
  rm /usr/local/bin/ygg-lazy-cli
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
rm /usr/local/bin/ygg-lazy-cli-uninstall

echo "Done! YggLazy-cli has been uninstalled."
UNINSTALL_EOF

chmod +x /usr/local/bin/ygg-lazy-cli-uninstall

echo ""
echo "===================================="
echo "Done! YggLazy-cli has been installed."
echo "Look for YggLazy-cli in your applications menu."
echo "Or type 'sudo ygg-lazy-cli' in the terminal."
echo ""
echo "To uninstall later, run: sudo ygg-lazy-cli-uninstall"
echo "===================================="
