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
ygglazy --version

# Run configurator
sudo ygglazy

# Get help
ygglazy --help
```

## Documentation

- 📚 [Installation Guide](docs/INSTALL.md) - Detailed installation and troubleshooting
- 📝 [Changelog](docs/CHANGELOG.md) - Full history of changes
- 🔄 [Migration Guide](docs/MIGRATION.md) - Upgrading from older versions

## How to get utility:

### Windows
* **Download** latest "ygglazy-VER-ARCH.exe" for your cpu architecture from [Releases](https://github.com/Y-Akamirsky/ygg-lazy-cli/releases/latest)
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
sudo ygglazy-uninstall
```

- **Or** you can use the standalone uninstall script:
```bash
curl -sL https://raw.githubusercontent.com/Y-Akamirsky/ygg-lazy-cli/main/uninstall.sh | sudo bash
```
    
### MacOS

#### Option 1: Homebrew (Recommended)
```bash
brew tap Y-Akamirsky/ygg-lazy-cli
brew install ygglazy
sudo ygglazy
```

#### Option 2: Download Binary
1. **Download** the latest release for your Mac:
   - **Intel Mac**: `ygglazy-darwin-amd64`
   - **Apple Silicon**: `ygglazy-darwin-arm64`
   
   From [Releases](https://github.com/Y-Akamirsky/ygg-lazy-cli/releases/latest)

2. **Remove quarantine attribute** (macOS security):
   ```bash
   xattr -d com.apple.quarantine ygglazy-darwin-*
   ```

3. **Make executable and install**:
   ```bash
   chmod +x ygglazy-darwin-*
   sudo mv ygglazy-darwin-* /usr/local/bin/ygglazy
   ```

4. **Run**:
   ```bash
   sudo ygglazy
   ```

> See [Installation Guide](docs/INSTALL.md) for troubleshooting.
    
### BSD (FreeBSD, OpenBSD, NetBSD)
1. **Download** the latest release for your BSD system:
   - **FreeBSD**: `ygglazy-freebsd-amd64`
   - **OpenBSD**: `ygglazy-openbsd-amd64`
   - **NetBSD**: `ygglazy-netbsd-amd64`
   
   From [Releases](https://github.com/Y-Akamirsky/ygg-lazy-cli/releases/latest)

2. **Make executable and install**:
   ```bash
   chmod +x ygglazy-*-amd64
   # FreeBSD/NetBSD
   sudo mv ygglazy-*-amd64 /usr/local/bin/ygglazy
   # OpenBSD
   doas mv ygglazy-*-amd64 /usr/local/bin/ygglazy
   ```

3. **Run**:
   ```bash
   # FreeBSD/NetBSD
   sudo ygglazy
   # OpenBSD
   doas ygglazy
   ```

> See [Installation Guide](docs/INSTALL.md) for pkg/ports installation details.

## Contribute
- Yo! Pull-requests & issues are welcome! Please open an issue if you have problems with YggLazy CLI.

## Built With
- Go
- good intentions
- friends pain...
