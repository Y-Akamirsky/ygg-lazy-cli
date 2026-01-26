# Changelog

All notable changes to YggLazy-cli will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.5] - 2025-01-26

### 🎉 Major Features

#### Peer Testing & Validation
- **Multi-threaded peer testing** - Now tests 100 peers with 20 concurrent workers (was 25 peers sequentially)
- **Multiple ping attempts** - 5 attempts per peer with 150ms delay between attempts for accurate measurements
- **Advanced latency metrics**:
  - Average, minimum, and maximum latency
  - Jitter calculation (standard deviation)
  - Stability score (coefficient of variation)
  - Smart ranking by combined latency + stability score
- **Real-time progress display** - Shows tested count, found peers, last latency, and jitter
- **Detailed peer statistics** - Min/max/jitter for each peer with stability classification (excellent/good/fair/unstable)

#### Dead Peer Management
- **New: "Check Active Peers Status"** menu option - View real-time status of all peers via `yggdrasilctl getPeers`
- **New: "Remove Dead Peers"** feature:
  - Automatically detects peers with "Down" status
  - Shows error messages for each dead peer (EOF, i/o timeout, etc.)
  - Distinguishes between config peers and temporary peers
  - Batch removal with confirmation prompt
  - Smart restart prompt after removal

#### IPv6 Address Support
- **Fixed IPv6 parsing bug** - Now correctly handles IPv6 addresses in square brackets like `tcp://[2001:db8::1]:12345`
- **Improved regex pattern** - Better matching for IPv6, IPv4, and hostname formats
- **Proper bracket detection** - Searches for properly formatted closing brackets (`\n  ]`) instead of the first `]`

#### Installation & Uninstallation
- **Auto-generated uninstaller** - `install.sh` now creates `/usr/local/bin/ygg-lazy-cli-uninstall`
- **Standalone uninstall script** - `uninstall.sh` can be run independently
- **Detailed uninstall output** - Shows each file being removed with status
- **Smart cleanup** - Updates desktop database after install/uninstall
- **Installation verification** - Comprehensive INSTALL.md documentation

#### Version Management
- **New: `--version` / `-v` flag** - Show version without sudo
- **New: `--help` / `-h` flag** - Display usage information
- **Centralized version constant** - Single source of truth for version number
- **Enhanced version output** - Shows YggLazy-cli version, Go version, and OS/Arch

### ✨ Improvements

#### User Experience
- **Better progress indicators** - Real-time updates during peer testing with ✓/✗ symbols
- **Color-coded output** - Green for success, red for errors, yellow for warnings, cyan for info
- **Detailed error messages** - Shows specific error for each failed peer (timeout, EOF, etc.)
- **Testing summary** - Statistics after peer testing (total tested, found, best latency, best stability)
- **Warning messages** - Informs user about limitations and next steps

#### Performance
- **20 concurrent workers** - Much faster peer testing (10x+ speedup)
- **100 peers tested** - Double the previous amount for better selection
- **Smart timeout handling** - 3-second timeout for initial connection, 2-second for response
- **Efficient parsing** - Improved JSON parsing from `yggdrasilctl` output

#### Code Quality
- **Better error handling** - More descriptive error messages and graceful degradation
- **Type safety** - Proper struct definitions for peer data
- **Documentation** - Comprehensive inline comments explaining complex logic
- **Modular functions** - Separated concerns for better maintainability

### 🐛 Bug Fixes

- **Fixed IPv6 bracket parsing** - Config parser now correctly handles IPv6 addresses
- **Fixed jitter calculation** - Now uses proper standard deviation (sqrt of variance)
- **Fixed stability filtering** - Relaxed criteria from 0.5 to allow more peers
- **Fixed dead peer detection** - Now correctly parses `getPeers` JSON array structure
- **Fixed config peer matching** - Properly matches URIs between config and active peers
- **Fixed flag parsing order** - Flags now parsed before requiring sudo for `--version`/`--help`

### 📚 Documentation

#### New Files
- **INSTALL.md** - Comprehensive installation/uninstallation guide with troubleshooting
- **CHANGELOG.md** - This file, tracking all changes
- **uninstall.sh** - Standalone uninstallation script

#### Updated Files
- **README.md** - Added uninstallation instructions and links
- **install.sh** - Now generates uninstaller and provides better feedback

### 🔧 Technical Changes

#### Peer Structure Enhancement
```go
type Peer struct {
    URI             string
    Latency         time.Duration
    MinLatency      time.Duration
    MaxLatency      time.Duration
    Jitter          time.Duration
    Stability       float64
    YggdrasilStatus bool
}
```

#### New Functions
- `pingPeerDetailed()` - Comprehensive peer testing with statistics
- `checkActivePeersStatus()` - Display real-time peer status
- `removeDeadPeers()` - Intelligent dead peer removal
- Custom `flag.Usage()` - Better help output

#### Modified Functions
- `autoAddPeers()` - Multi-threaded testing with detailed metrics
- `getConfigPeers()` - Fixed IPv6 bracket handling
- `addPeersToConfig()` - Improved closing bracket detection
- `removePeersFromConfig()` - Better pattern matching
- `printBanner()` - Uses version constant

### 🗑️ Removed

- Unreliable Yggdrasil protocol handshake checking (replaced with post-install verification)
- Overly strict stability filtering (stability > 0.5)
- Single-threaded peer testing

### 📊 Statistics

- **990 lines added**, 121 lines removed
- 5 files changed significantly
- 2 new documentation files
- 1 new uninstall script

### 🔄 Migration Notes

#### From v0.1.4 to v0.1.5

**No breaking changes** - All configurations and data remain compatible.

**New workflow recommended:**
1. Use "Auto-select Peers" to add best peers
2. Restart Yggdrasil service
3. Wait 2-3 minutes
4. Use "Check Active Peers Status" to verify
5. Use "Remove Dead Peers" to clean up

**Uninstallation:**
- Old versions: Manual removal required
- v0.1.5+: Use `sudo ygg-lazy-cli-uninstall` or `uninstall.sh`

---

## [0.1.4] - 2025-01-25

### Initial release with basic features
- Interactive menu system
- Auto-select peers by latency
- Manual peer selection
- View/add/remove peers
- Node status display
- Service control
- Yggdrasil auto-installation
- Cross-platform support (Windows/Linux)

---

## Links

- [Repository](https://github.com/Y-Akamirsky/ygg-lazy-cli)
- [Issues](https://github.com/Y-Akamirsky/ygg-lazy-cli/issues)
- [Releases](https://github.com/Y-Akamirsky/ygg-lazy-cli/releases)
