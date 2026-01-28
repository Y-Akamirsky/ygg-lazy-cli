# Installation Guide

## Quick Install

```bash
curl -sL https://raw.githubusercontent.com/Y-Akamirsky/ygg-lazy-cli/main/install.sh | sudo bash
```

**Note:** Installation takes 1-5 minutes as the program is compiled on your device for maximum compatibility.

## What the Installer Does

1. Checks for Go (installs automatically if needed)
2. Downloads source code
3. Compiles the program on your device
4. Installs to system
5. Cleans up temporary files

## Usage

```bash
# Run (requires sudo)
sudo ygg-lazy-cli

# Show version
ygg-lazy-cli --version

# Show help
ygg-lazy-cli --help
```

## Uninstall

```bash
sudo ygg-lazy-cli-uninstall
```

## Troubleshooting

### Go doesn't install automatically

```bash
# Install Go manually via 'g' utility
curl -sSL https://git.io/g-install | sh -s
source ~/.bashrc
g install latest

# Then retry installation
curl -sL https://raw.githubusercontent.com/Y-Akamirsky/ygg-lazy-cli/main/install.sh | sudo bash
```

### Missing git

```bash
# Debian/Ubuntu
sudo apt install git

# Fedora
sudo dnf install git

# Arch
sudo pacman -S git
```

### Program doesn't appear in menu

```bash
sudo update-desktop-database /usr/share/applications/
```

## Why Compile on Device?

On older distributions (Debian 12, Slackware), pre-compiled binaries may fail due to CPU instruction incompatibility. Compiling on your device solves this.

## Manual Installation

If automatic script doesn't work:

```bash
# 1. Install Go
curl -sSL https://git.io/g-install | sh -s
source ~/.bashrc
g install latest

# 2. Download and compile
git clone https://github.com/Y-Akamirsky/ygg-lazy-cli.git
cd ygg-lazy-cli
CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o ygg-lazy-cli .

# 3. Install
sudo cp ygg-lazy-cli /usr/local/bin/
sudo chmod +x /usr/local/bin/ygg-lazy-cli
```

## Support

If you have issues, create an [issue on GitHub](https://github.com/Y-Akamirsky/ygg-lazy-cli/issues) with:
- Distribution version (`cat /etc/os-release`)
- Architecture (`uname -m`)
- Error message