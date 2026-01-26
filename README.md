# YggLazy CLI
### Lazy way to configure Yggdrasil Network!

![Version](https://img.shields.io/badge/version-0.1.5-blue) ![License](https://img.shields.io/badge/license-GPL3-orange)

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
    curl -sL https://raw.githubusercontent.com/Y-Akamirsky/ygg-lazy-cli/main/install.sh | sudo bash
    ```
- **Press** Enter.
- **Little** wait...
- **Use** it!

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
- Coming soon...
    
### BSD
- Coming soon...
    
## What's New in v0.1.5

- ✨ Enhanced peer testing with advanced metrics
- 🔧 New "Check Active Peers Status" and "Remove Dead Peers" features
- 🌐 Fixed IPv6 address parsing
- 📦 Auto-generated uninstaller
- ℹ️ `--version` and `--help` flags
- 📚 Comprehensive documentation (INSTALL.md, CHANGELOG.md, MIGRATION.md)

See [CHANGELOG.md](CHANGELOG.md) for full details.

## Contribute
- Yo! Pull-requests & issues are welcome! Please open an issue if you have problems with YggLazy CLI.

## Built With
- Go
- good intentions
- friends pain...
