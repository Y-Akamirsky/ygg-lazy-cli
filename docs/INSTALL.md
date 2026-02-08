# Installation Guide

## Linux - Quick Install

```bash
curl -sL https://raw.githubusercontent.com/Y-Akamirsky/ygg-lazy-cli/main/install.sh | bash
```

**Note:** Installation takes 1-5 minutes as the program is compiled on your device for maximum compatibility.

### What the Installer Does

1. Checks for Go (installs automatically if needed)
2. Downloads source code
3. Compiles the program on your device
4. Installs to system
5. Cleans up temporary files

## macOS Installation

### Option 1: Homebrew (Recommended)

```bash
brew tap Y-Akamirsky/ygg-lazy-cli
brew install ygglazy
```

**Run:**
```bash
sudo ygglazy
```

That's it! Homebrew handles everything including dependencies and PATH setup.

### Option 2: Download Pre-built Binary

1. **Download** the appropriate binary for your Mac:
   - **Intel Mac**: Download `ygglazy-darwin-amd64`
   - **Apple Silicon (M1/M2/M3)**: Download `ygglazy-darwin-arm64`
   
   From [Releases](https://github.com/Y-Akamirsky/ygg-lazy-cli/releases/latest)

2. **Remove macOS quarantine attribute**:
   ```bash
   xattr -d com.apple.quarantine ~/Downloads/ygglazy-darwin-*
   ```
   
   > **Why?** macOS marks downloaded files as quarantined. This command removes the restriction.

3. **Install**:
   ```bash
   chmod +x ~/Downloads/ygglazy-darwin-*
   sudo mv ~/Downloads/ygglazy-darwin-* /usr/local/bin/ygglazy
   ```

4. **Run**:
   ```bash
   sudo ygglazy
   ```

### Option 3: Build from Source

```bash
# Install dependencies
brew install go git

# Clone and build
git clone https://github.com/Y-Akamirsky/ygg-lazy-cli.git
cd ygg-lazy-cli
go build -ldflags="-s -w" -trimpath -o ygglazy

# Install
sudo mv ygglazy /usr/local/bin/
```

### macOS Troubleshooting

**"Cannot be opened because the developer cannot be verified"**
```bash
# Remove quarantine attribute
xattr -d com.apple.quarantine /usr/local/bin/ygglazy
```

**Command not found after install**
```bash
# Check if /usr/local/bin is in PATH
echo $PATH | grep /usr/local/bin

# If not, add to your shell config:
echo 'export PATH="/usr/local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

## BSD Installation

### FreeBSD

1. **Download**:
   ```bash
   fetch https://github.com/Y-Akamirsky/ygg-lazy-cli/releases/latest/download/ygglazy-freebsd-amd64
   ```

2. **Install**:
   ```bash
   chmod +x ygglazy-freebsd-amd64
   sudo mv ygglazy-freebsd-amd64 /usr/local/bin/ygglazy
   ```

3. **Run**:
   ```bash
   sudo ygglazy
   ```

**Alternative: Build from source**
```bash
# Install dependencies
sudo pkg install go git

# Clone and build
git clone https://github.com/Y-Akamirsky/ygg-lazy-cli.git
cd ygg-lazy-cli
go build -ldflags="-s -w" -trimpath -o ygglazy

# Install
sudo mv ygglazy /usr/local/bin/
```

### OpenBSD

1. **Download**:
   ```bash
   ftp https://github.com/Y-Akamirsky/ygg-lazy-cli/releases/latest/download/ygglazy-openbsd-amd64
   ```

2. **Install**:
   ```bash
   chmod +x ygglazy-openbsd-amd64
   doas mv ygglazy-openbsd-amd64 /usr/local/bin/ygglazy
   ```

3. **Run**:
   ```bash
   doas ygglazy
   ```

**Alternative: Build from source**
```bash
# Install dependencies
doas pkg_add go git

# Clone and build
git clone https://github.com/Y-Akamirsky/ygg-lazy-cli.git
cd ygg-lazy-cli
go build -ldflags="-s -w" -trimpath -o ygglazy

# Install
doas mv ygglazy /usr/local/bin/
```

### NetBSD

1. **Download**:
   ```bash
   ftp https://github.com/Y-Akamirsky/ygg-lazy-cli/releases/latest/download/ygglazy-openbsd-amd64
   ```

2. **Install**:
   ```bash
   chmod +x ygglazy-openbsd-amd64
   su -c 'mv ygglazy-openbsd-amd64 /usr/local/bin/ygglazy'
   ```

3. **Run**:
   ```bash
   su -c ygglazy
   # or
   sudo ygglazy
   ```

**Alternative: Build from source**
```bash
# Install dependencies (as root)
pkgin install go git

# Clone and build (as user)
git clone https://github.com/Y-Akamirsky/ygg-lazy-cli.git
cd ygg-lazy-cli
go build -ldflags="-s -w" -trimpath -o ygg-lazy-cli

# Install (as root)
su -c 'mv ygglazy /usr/local/bin/'
```

## Windows Installation

1. **Download** `ygglazy-windows-amd64.exe` or `ygglazy-windows-386.exe` from [Releases](https://github.com/Y-Akamirsky/ygg-lazy-cli/releases/latest)

2. **Run as Administrator** (right-click → "Run as administrator")

3. **Use** the interactive menu

## Usage

### Linux/BSD
```bash
# Run (requires root)
sudo ygglazy        # Linux, FreeBSD, NetBSD
doas ygglazy        # OpenBSD

# Show version (no root needed)
ygglazy --version

# Show help
ygglazy --help
```

### macOS
```bash
# Run (requires sudo)
sudo ygglazy

# Show version
ygglazy --version
```

### Windows
- Right-click → "Run as administrator"
- Use PowerShell or CMD

## Uninstall

### Linux (installed via script)
```bash
sudo ygglazy-uninstall
```

### macOS/BSD (manual install)
```bash
sudo rm /usr/local/bin/ygglazy        # FreeBSD, NetBSD, macOS
doas rm /usr/local/bin/ygglazy        # OpenBSD
```

### Windows
- Delete the .exe file

## Troubleshooting

### Linux: Missing git

```bash
# Debian/Ubuntu
sudo apt install git

# Fedora
sudo dnf install git

# Arch
sudo pacman -S git

# Alpine
sudo apk add git

# Void
sudo xbps-install -S git
```

### Linux: Program doesn't appear in menu

```bash
sudo update-desktop-database /usr/share/applications/
```

### macOS: "developer cannot be verified"

```bash
xattr -d com.apple.quarantine /usr/local/bin/ygglazy
```

### BSD: Missing dependencies

```bash
# FreeBSD
sudo pkg install go git

# OpenBSD
doas pkg_add go git

# NetBSD
su -c 'pkgin install go git'
```

## Why Compile on Device? (Linux)

On older PCs, pre-compiled binaries may fail due to CPU instruction incompatibility. Compiling on your device solves this.

## Manual Build from Source (All Platforms)

If automatic installation doesn't work:

```bash
# 1. Install Go (if not installed)
# Linux:
curl -sSL https://git.io/g-install | sh -s
source ~/.bashrc
g install latest

# macOS (Homebrew):
brew install go

# FreeBSD:
sudo pkg install go

# OpenBSD:
doas pkg_add go

# NetBSD:
su -c 'pkgin install go'

# 2. Download and compile
git clone https://github.com/Y-Akamirsky/ygg-lazy-cli.git
cd ygg-lazy-cli
go build -ldflags="-s -w" -trimpath -o ygglazy

# 3. Install
sudo cp ygglazy /usr/local/bin/        # Linux, macOS, FreeBSD, NetBSD
doas cp ygglazy /usr/local/bin/        # OpenBSD
sudo chmod +x /usr/local/bin/ygglazy
```

## Platform-Specific Notes

### macOS
- **Homebrew installation of Yggdrasil** is recommended: `brew install yggdrasil-go`
- Config location: `/usr/local/etc/yggdrasil.conf` or `/opt/homebrew/etc/yggdrasil.conf`
- Service management via `launchctl`

### FreeBSD
- Install Yggdrasil: `sudo pkg install yggdrasil` or via ports
- Config location: `/usr/local/etc/yggdrasil.conf`
- Service management: `sudo service yggdrasil start`

### OpenBSD
- Install Yggdrasil: `doas pkg_add yggdrasil`
- Config location: `/etc/yggdrasil.conf`
- Service management: `doas rcctl start yggdrasil`

### NetBSD
- Install Yggdrasil: `su -c 'pkgin install yggdrasil'`
- Config location: `/etc/yggdrasil.conf`
- Service management: `su -c '/etc/rc.d/yggdrasil start'`

## Support

If you have issues, create an [issue on GitHub](https://github.com/Y-Akamirsky/ygg-lazy-cli/issues) with:
- OS and version (`uname -a` or `cat /etc/os-release`)
- Architecture (`uname -m`)
- Error message
- Installation method used
