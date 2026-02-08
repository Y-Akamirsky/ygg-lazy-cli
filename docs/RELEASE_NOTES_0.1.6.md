# YggLazy-cli v0.1.6c

**Release Date:** February 8, 2026

## 🎉 What's New

### Name in terminal was changed to `ygglazy`
- Way easier to use:
  - `ygglazy` instead of `ygg-lazy-cli`
  - `ygglazy -i` instead of `ygg-lazy-cli --ygginstall`

### New short flag
- `-i` - Install Yggdrasil (like --ygginstall)

## 🐛 Bug Fixes

- None

## 📥 Install / Upgrade

**Linux:**
```bash
curl -sL https://raw.githubusercontent.com/Y-Akamirsky/ygg-lazy-cli/main/install.sh | sudo bash
```

**Windows:** Download `.exe` from [Releases](https://github.com/Y-Akamirsky/ygg-lazy-cli/releases/tag/v0.1.5)

**MacOS:**
```bash
brew tap Y-Akamirsky/ygg-lazy-cli
brew install ygglazy
```

## 🗑️ Uninstall

```bash
sudo ygg-lazy-cli-uninstall
```

## 📚 Documentation

- [INSTALL.md](../INSTALL.md) - Installation guide
- [CHANGELOG.md](../CHANGELOG.md) - Full changelog
- [MIGRATION.md](../MIGRATION.md) - Upgrade guide

## ⚠️ Breaking Changes

**Name was changed!** Highly recommended to uninstall old version:
**Linux:**
```bash
sudo ygg-lazy-cli-uninstall
```
**MacOS:**
```bash
brew uninstall ygg-lazy-cli
```
Then install new one:
**Linux:**
```bash
curl -sL https://raw.githubusercontent.com/Y-Akamirsky/ygg-lazy-cli/main/install.sh | sudo bash
```
**MacOS:**
```bash
brew tap Y-Akamirsky/ygg-lazy-cli
brew install ygglazy
```
**Windows:** 
Just remove old .exe lol and download new one from [Releases](https://github.com/Y-Akamirsky/ygg-lazy-cli/releases/)

**BSD:**
```bash
sudo rm -f /usr/local/bin/ygglazy
# Or:
doas rm -f /usr/local/bin/ygglazy
```
**Changelog**: [0.1.6-b...v0.1.6-c](https://github.com/Y-Akamirsky/ygg-lazy-cli/compare/0.1.6-b...0.1.6-c)
