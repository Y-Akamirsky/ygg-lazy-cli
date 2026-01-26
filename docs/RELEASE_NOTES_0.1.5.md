# YggLazy-cli v0.1.5

**Release Date:** January 26, 2025

## 🎉 What's New

### Enhanced Peer Testing
- Tests **100 peers** (up from 25) with **20 concurrent workers**
- **5 ping attempts** per peer for better accuracy
- Advanced metrics: min/max latency, jitter, stability score
- Real-time progress display

### Dead Peer Management
- **"Check Active Peers Status"** - View real-time connection status
- **"Remove Dead Peers"** - Auto-detect and remove failed connections

### IPv6 Support
- Fixed parsing of IPv6 addresses like `tcp://[2001:db8::1]:12345`
- Improved config file bracket detection

### Installation
- Auto-generated uninstaller: `sudo ygg-lazy-cli-uninstall`
- Standalone uninstall script available

### CLI Improvements
- `--version` / `-v` flag (no sudo needed)
- `--help` / `-h` flag

## 🐛 Bug Fixes

- Fixed IPv6 address parsing in config
- Fixed jitter calculation (proper standard deviation)
- Fixed dead peer detection JSON parsing
- Fixed flag parsing order for --version

## 📥 Install / Upgrade

**Linux:**
```bash
curl -sL https://raw.githubusercontent.com/Y-Akamirsky/ygg-lazy-cli/main/install.sh | sudo bash
```

**Windows:** Download `.exe` from [Releases](https://github.com/Y-Akamirsky/ygg-lazy-cli/releases/tag/v0.1.5)

## 🗑️ Uninstall

```bash
sudo ygg-lazy-cli-uninstall
```

## 📚 Documentation

- [INSTALL.md](../INSTALL.md) - Installation guide
- [CHANGELOG.md](../CHANGELOG.md) - Full changelog
- [MIGRATION.md](../MIGRATION.md) - Upgrade guide

## ⚠️ Breaking Changes

**None** - Fully backward compatible with v0.1.4

---

**Full Changelog**: [v0.1.4...v0.1.5](https://github.com/Y-Akamirsky/ygg-lazy-cli/compare/v0.1.4...v0.1.5)