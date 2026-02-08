# Migration Guide: v0.1.5,v0.1.6-b → v0.1.6-c

This guide helps existing users upgrade from YggLazy-cli v0.1.5,v0.1.6-b to v0.1.6-c.

## Quick Upgrade

**Name was changed!** Highly recommended to uninstall old version before installing new one:

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
