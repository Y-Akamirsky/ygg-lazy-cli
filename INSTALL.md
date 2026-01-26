# Installation & Uninstallation Guide

## Installation

### Linux

#### Quick Install (Recommended)
```bash
curl -sL https://raw.githubusercontent.com/Y-Akamirsky/ygg-lazy-cli/main/install.sh | sudo bash
```

#### Manual Install
1. Download the installer:
   ```bash
   wget https://raw.githubusercontent.com/Y-Akamirsky/ygg-lazy-cli/main/install.sh
   ```

2. Make it executable:
   ```bash
   chmod +x install.sh
   ```

3. Run with sudo:
   ```bash
   sudo ./install.sh
   ```

#### What Gets Installed

The installer will:
- Download the correct binary for your architecture (amd64/arm64)
- Install to `/usr/local/bin/ygg-lazy-cli`
- Add application icon to `/usr/local/share/icons/ygglazycli.svg`
- Create desktop shortcut at `/usr/share/applications/ygg-lazy-cli.desktop`
- Generate an uninstaller at `/usr/local/bin/ygg-lazy-cli-uninstall`

After installation, you can:
- Launch from terminal: `ygg-lazy-cli` (requires sudo for config access)
- Find in your applications menu as "YggLazy-cli"

---

## Uninstallation

### Method 1: Auto-generated Uninstaller (Recommended)

If you installed with the install.sh script, an uninstaller was automatically created:

```bash
sudo ygg-lazy-cli-uninstall
```

This will remove all installed files and clean up the system.

### Method 2: Standalone Uninstall Script

If the auto-generated uninstaller is not available (e.g., you installed an older version):

```bash
curl -sL https://raw.githubusercontent.com/Y-Akamirsky/ygg-lazy-cli/main/uninstall.sh | sudo bash
```

Or download and run manually:
```bash
wget https://raw.githubusercontent.com/Y-Akamirsky/ygg-lazy-cli/main/uninstall.sh
chmod +x uninstall.sh
sudo ./uninstall.sh
```

### Method 3: Manual Removal

Remove the files manually:

```bash
sudo rm /usr/local/bin/ygg-lazy-cli
sudo rm /usr/local/share/icons/ygglazycli.svg
sudo rm /usr/share/applications/ygg-lazy-cli.desktop
sudo rm /usr/local/bin/ygg-lazy-cli-uninstall  # if exists
sudo update-desktop-database /usr/share/applications/
```

---

## Troubleshooting

### Installation Issues

**Problem**: "Architecture not supported"
- **Solution**: Currently only x86_64 (amd64) and aarch64 (arm64) are supported. Other architectures coming soon.

**Problem**: "Permission denied"
- **Solution**: Make sure you run the installer with `sudo`

**Problem**: "curl: command not found"
- **Solution**: Install curl first:
  - Debian/Ubuntu: `sudo apt install curl`
  - Fedora/RHEL: `sudo dnf install curl`
  - Arch: `sudo pacman -S curl`

### Uninstallation Issues

**Problem**: "Uninstaller not found"
- **Solution**: Use Method 2 (standalone uninstall script) or Method 3 (manual removal)

**Problem**: "Some files not found during uninstall"
- **Solution**: This is normal - it means those files were already removed or the installation was incomplete. The uninstaller will skip missing files.

---

## Verification

### Check Installation
```bash
# Check binary exists and is executable
which ygg-lazy-cli
# Should output: /usr/local/bin/ygg-lazy-cli

# Check version
ygg-lazy-cli --version

# Check if desktop entry exists
ls -l /usr/share/applications/ygg-lazy-cli.desktop

# Check if icon exists
ls -l /usr/local/share/icons/ygglazycli.svg
```

### Check Uninstallation
```bash
# Verify all files are removed
ls /usr/local/bin/ygg-lazy-cli 2>/dev/null && echo "Binary still exists" || echo "Binary removed ✓"
ls /usr/local/share/icons/ygglazycli.svg 2>/dev/null && echo "Icon still exists" || echo "Icon removed ✓"
ls /usr/share/applications/ygg-lazy-cli.desktop 2>/dev/null && echo "Desktop file still exists" || echo "Desktop file removed ✓"
```

---

## Notes

- **Root Access**: Installation and uninstallation require root/sudo privileges
- **Config Files**: Yggdrasil configuration files (`/etc/yggdrasil.conf`) are NOT removed during uninstallation
- **Updates**: To update, simply run the install script again - it will detect and remove the old version first
- **Multiple Architectures**: The installer automatically detects your CPU architecture and downloads the correct binary

---

## Platform Support

| Platform | Status | Architectures |
|----------|--------|---------------|
| Linux    | ✅ Supported | x86_64, aarch64 |
| Windows  | ✅ Supported | Manual install from releases |
| macOS    | 🔜 Coming soon | - |
| BSD      | 🔜 Coming soon | - |

---

## Getting Help

If you encounter issues:
1. Check this troubleshooting guide
2. Open an issue on [GitHub](https://github.com/Y-Akamirsky/ygg-lazy-cli/issues)
3. Include:
   - Your OS and architecture (`uname -a`)
   - Error messages
   - Steps to reproduce