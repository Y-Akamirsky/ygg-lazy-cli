# Migration Guide: v0.1.4 → v0.1.5

This guide helps existing users upgrade from YggLazy-cli v0.1.4 to v0.1.5.

## Quick Upgrade

### Linux

Simply run the installer again - it will detect and upgrade automatically:

```bash
curl -sL https://raw.githubusercontent.com/Y-Akamirsky/ygg-lazy-cli/main/install.sh | sudo bash
```

The installer will:
1. Detect the old version
2. Remove it automatically
3. Install the new version
4. Create an uninstaller for future use

### Windows

1. Close any running instances of YggLazy-cli
2. Download the new version from [Releases](https://github.com/Y-Akamirsky/ygg-lazy-cli/releases/tag/v0.1.5)
3. Replace the old executable
4. Run as Administrator

---

## What Changes for You

### ✅ No Breaking Changes

Your existing configuration files and peer lists are **100% compatible**. Nothing breaks.

### 🎉 New Features You'll See

#### 1. Enhanced Peer Testing (Automatic)

When you use "Auto-select Peers", you'll notice:
- Testing takes longer (~1-2 minutes instead of ~15 seconds)
- More peers are tested (100 vs 25)
- More detailed statistics (min/max/jitter/stability)
- Better peer quality overall

**What to do:** Nothing! Just enjoy better peers.

#### 2. New Menu Options

Two new options in the main menu:
- **"Check Active Peers Status"** - See real-time connection status
- **"Remove Dead Peers"** - Automatically clean up failed connections

**What to do:** Try them out after adding peers!

#### 3. IPv6 Addresses Work Now

If you had issues with IPv6 peers like `tcp://[2001:db8::1]:12345`, they now work correctly.

**What to do:** 
- If you manually removed IPv6 peers due to bugs, you can add them back
- Use "Auto-select Peers" to find new IPv6 peers

#### 4. Version Flag

You can now check the version without sudo:

```bash
ygg-lazy-cli --version
# or
ygg-lazy-cli -v
```

**What to do:** Use this to verify you upgraded successfully!

---

## Recommended Post-Upgrade Steps

### Step 1: Verify Installation

```bash
# Check version (should show 0.1.5)
ygg-lazy-cli --version

# Check uninstaller was created
ls -l /usr/local/bin/ygg-lazy-cli-uninstall
```

### Step 2: Clean Up Dead Peers (Optional but Recommended)

If you've been using v0.1.4 for a while, you might have accumulated dead peers:

```bash
sudo ygg-lazy-cli
```

Then:
1. Select "Check Active Peers Status" - See which peers are Down
2. Select "Remove Dead Peers" - Clean them up automatically
3. Restart service when prompted

### Step 3: Test the New Peer Selection (Optional)

Want to try the improved peer testing?

```bash
sudo ygg-lazy-cli
```

Then:
1. Select "Auto-select Peers (Best Latency)"
2. Wait for testing to complete (~1-2 minutes)
3. Review the detailed statistics
4. Add the recommended peers
5. Wait 2-3 minutes after restart
6. Use "Check Active Peers Status" to verify
7. Use "Remove Dead Peers" if needed

---

## Troubleshooting Upgrade Issues

### Problem: "Old version still shows"

**Solution:**
```bash
# Check which version is running
which ygg-lazy-cli
ygg-lazy-cli --version

# If still old version, reinstall
curl -sL https://raw.githubusercontent.com/Y-Akamirsky/ygg-lazy-cli/main/install.sh | sudo bash
```

### Problem: "Can't find uninstaller"

**Symptom:** `/usr/local/bin/ygg-lazy-cli-uninstall` doesn't exist

**Solution:** This is normal if you upgraded from v0.1.4. The uninstaller is created during installation. You can:
1. Run the installer again to create it
2. Or use the standalone uninstall script when needed

### Problem: "IPv6 peers still not working"

**Solution:**
```bash
# Verify you're on v0.1.5
ygg-lazy-cli --version

# Check your config
sudo ygg-lazy-cli
# Select "View Configured Peers"

# If IPv6 peers look correct, restart Yggdrasil
sudo systemctl restart yggdrasil
```

### Problem: "Peer testing seems slower"

**This is expected!** v0.1.5 tests 100 peers with 5 attempts each for much better accuracy.

- v0.1.4: 25 peers × 3 attempts = 75 connections (~15 seconds)
- v0.1.5: 100 peers × 5 attempts = 500 connections (~60-120 seconds)

**Result:** Much better peer quality and stability!

---

## Configuration Compatibility

### Config Files

✅ **No changes needed** to `/etc/yggdrasil.conf`

Your existing configuration works as-is.

### Peer Format

✅ **All formats still supported**:
- IPv4: `tcp://192.168.1.1:12345`
- Hostnames: `tls://node.example.com:443`
- IPv6: `tcp://[2001:db8::1]:54321` (now works better!)

### Installation Paths

✅ **Same paths** (no changes):
- Binary: `/usr/local/bin/ygg-lazy-cli`
- Icon: `/usr/local/share/icons/ygglazycli.svg`
- Desktop: `/usr/share/applications/ygg-lazy-cli.desktop`

**New in v0.1.5:**
- Uninstaller: `/usr/local/bin/ygg-lazy-cli-uninstall` (auto-generated)

---

## New Workflow Recommendations

### Old Workflow (v0.1.4)
```
1. Auto-select Peers
2. Add 3 peers
3. Restart service
4. Hope for the best 🤞
```

### New Workflow (v0.1.5)
```
1. Auto-select Peers
2. Add 3-10 peers (more options now!)
3. Restart service
4. Wait 2-3 minutes ⏱️
5. Check Active Peers Status ✅
6. Remove Dead Peers if any 🗑️
7. Enjoy stable connections! 🎉
```

---

## Rollback (If Needed)

If you need to go back to v0.1.4 for any reason:

### Linux

```bash
# Uninstall v0.1.5
sudo ygg-lazy-cli-uninstall

# Install v0.1.4
# (Download and run old installer or build from v0.1.4 tag)
```

**Note:** Your config files are untouched, so rollback is safe.

---

## Performance Impact

### Testing Time
- **Old:** ~15 seconds
- **New:** ~60-120 seconds
- **Why:** Testing 4x more peers with better accuracy

### Resource Usage
- **CPU:** Slightly higher during testing (20 concurrent workers)
- **Memory:** Minimal increase (~10-20MB)
- **Network:** More connections during testing, but spread over time

### Runtime Performance
- **No impact** on normal operation
- Only affects the "Auto-select Peers" feature

---

## FAQ

### Q: Do I need to reconfigure anything?

**A:** No! Just upgrade and everything works.

### Q: Will my existing peers stop working?

**A:** No! Your configured peers are not touched.

### Q: Should I re-add all my peers?

**A:** Not necessary, but you might want to use "Remove Dead Peers" to clean up any failed connections.

### Q: Can I use both v0.1.4 and v0.1.5?

**A:** No, only one version can be installed at a time. The installer automatically replaces the old version.

### Q: What if I customized the installation paths?

**A:** Manual installations might need manual upgrade. Use the same paths you used originally.

### Q: Do I need to update anything on Windows?

**A:** Just replace the .exe file. No config changes needed.

---

## Getting Help

If you encounter issues during or after migration:

1. **Check this guide** - Most common issues are covered
2. **Check INSTALL.md** - Installation/troubleshooting reference
3. **Open an issue** - [GitHub Issues](https://github.com/Y-Akamirsky/ygg-lazy-cli/issues)
4. **Include:**
   - Your OS and version
   - Old version you're upgrading from
   - New version you're trying to install
   - Output of `ygg-lazy-cli --version`
   - Any error messages

---

## Summary

✅ **Safe upgrade** - No breaking changes  
✅ **Better peers** - More thorough testing  
✅ **New features** - Dead peer management, version flags  
✅ **Bug fixes** - IPv6 support, better parsing  
✅ **Easy uninstall** - Auto-generated uninstaller  

**Upgrade time:** ~2 minutes  
**Recommended downtime:** None (config untouched)  
**Risk level:** Low (fully backward compatible)

Enjoy YggLazy-cli v0.1.5! 🎉