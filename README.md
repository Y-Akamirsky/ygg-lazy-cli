# YggLazy CLI
### Lazy way to configure Yggdrasil Network!

![Version](https://img.shields.io/badge/version-0.1.6-blue) ![License](https://img.shields.io/badge/license-GPL3-orange)

![Logo](linux-install/ygglazycli.svg)

## Features

- 🚀 **Fast peer testing** - Tests 100 peers with 20 concurrent workers
- 📊 **Advanced metrics** - Latency, jitter, stability scoring
- 🔧 **Dead peer management** - Automatic detection and removal
- 📦 **Easy install/uninstall** - One-command installation with auto-generated uninstaller
- ℹ️ **Version flags** - `--version` and `--help` support

## Quick Start

```bash
# Check version (no sudo needed)
ygg-lazy-cli --version

# Run configurator
sudo ygg-lazy-cli

# Get help
ygg-lazy-cli --help
```

## Documentation

- 📚 [Installation Guide](docs/INSTALL.md) - Detailed installation and troubleshooting
- 📝 [Changelog](docs/CHANGELOG.md) - Full history of changes
- 🔄 [Migration Guide](docs/MIGRATION.md) - Upgrading from older versions

## How to get utility:

### Windows
* **Download** latest "ygg-lazy-cli-VER-ARCH.exe" for your cpu architecture from [Releases](https://github.com/Y-Akamirsky/ygg-lazy-cli/releases/latest)
* **Start** executable as Administrator.
* **Use** it!
    
### Linux
- **Open** a terminal (yep, again terminal), and copy paste command below.
    ```bash
    curl -sL https://raw.githubusercontent.com/Y-Akamirsky/ygg-lazy-cli/main/install.sh | bash
    ```
- **Press** Enter.
- **Wait** for compilation - the program will be built on your device for maximum compatibility!
- **Use** it!

> **Note**: Installation takes 1-5 minutes - the program is compiled on your device for maximum compatibility.  
> Details: [Russian guide](docs/INSTALL_RU.md) | [English guide](docs/INSTALL.md)

#### Uninstalling on Linux
- The installer automatically creates an uninstaller. **To remove YggLazy-cli:**
```bash
sudo ygg-lazy-cli-uninstall
```

- **Or** you can use the standalone uninstall script:
```bash
curl -sL https://raw.githubusercontent.com/Y-Akamirsky/ygg-lazy-cli/main/uninstall.sh | sudo bash
```
    
### MacOS

#### Option 1: Homebrew (Recommended)
```bash
brew tap Y-Akamirsky/ygg-lazy-cli
brew install ygg-lazy-cli
sudo ygg-lazy-cli
```

#### Option 2: Download Binary
1. **Download** the latest release for your Mac:
   - **Intel Mac**: `ygg-lazy-cli-darwin-amd64`
   - **Apple Silicon (M1/M2/M3)**: `ygg-lazy-cli-darwin-arm64`
   
   From [Releases](https://github.com/Y-Akamirsky/ygg-lazy-cli/releases/latest)

2. **Remove quarantine attribute** (macOS security):
   ```bash
   xattr -d com.apple.quarantine ygg-lazy-cli-darwin-*
   ```

3. **Make executable and install**:
   ```bash
   chmod +x ygg-lazy-cli-darwin-*
   sudo mv ygg-lazy-cli-darwin-* /usr/local/bin/ygg-lazy-cli
   ```

4. **Run**:
   ```bash
   sudo ygg-lazy-cli
   ```

> See [Installation Guide](docs/INSTALL.md) for troubleshooting.
    
### BSD (FreeBSD, OpenBSD, NetBSD)
1. **Download** the latest release for your BSD system:
   - **FreeBSD**: `ygg-lazy-cli-freebsd-amd64`
   - **OpenBSD**: `ygg-lazy-cli-openbsd-amd64`
   - **NetBSD**: `ygg-lazy-cli-netbsd-amd64`
   
   From [Releases](https://github.com/Y-Akamirsky/ygg-lazy-cli/releases/latest)

2. **Make executable and install**:
   ```bash
   chmod +x ygg-lazy-cli-*-amd64
   # FreeBSD/NetBSD
   sudo mv ygg-lazy-cli-*-amd64 /usr/local/bin/ygg-lazy-cli
   # OpenBSD
   doas mv ygg-lazy-cli-*-amd64 /usr/local/bin/ygg-lazy-cli
   ```

3. **Run**:
   ```bash
   # FreeBSD/NetBSD
   sudo ygg-lazy-cli
   # OpenBSD
   doas ygg-lazy-cli
   ```

> See [Installation Guide](docs/INSTALL.md) for pkg/ports installation details.

## Contribute
- Yo! Pull-requests & issues are welcome! Please open an issue if you have problems with YggLazy CLI.

## Built With
- Go
- good intentions
- friends pain...
