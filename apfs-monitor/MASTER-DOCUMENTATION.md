# APFS Space Monitoring - Complete Documentation

## Executive Summary

**Problem**: APFS containers share physical space across multiple volumes. When one volume fills, it can starve others and prevent Time Machine snapshots, causing system instability. This has occurred ~10 times since 2019, costing ~3 hours per incident.

**Solution**: Go-based monitoring daemon that checks APFS container free space every 5 minutes and sends alerts before critical thresholds are reached.

**Status**: ✅ Installed and running as of 2025-12-12

---

## System State (2025-12-12)

### APFS Containers on System

```
disk3  (Boot Container - 994.7 GB)
├─ Free: 31.54 GB (3.4%)
├─ Used: 960.8 GB (96.6%)
└─ Volumes:
   ├─ System (0xFF):   11.2 GB
   ├─ Preboot:          8.0 GB
   ├─ Recovery:         2.0 GB
   ├─ Data:           520.3 GB (encrypted, main user data)
   ├─ VM:               0.0 GB
   ├─ Publica:        206.0 GB (unencrypted)
   └─ Sandboxa:       212.9 GB

disk7  (187.65 GB free, 95.0% used)
disk9  (173.48 GB free, 63.6% used)
disk14 (876.31 GB free, 76.5% used)
disk22 (5683.49 GB free, 61.9% used)
```

### Critical Risk

**disk3 is at 96.6% capacity** - only 1.54 GB above warning threshold of 30 GB.

---

## The APFS Problem Explained

### What Happened (Latest Incident: 2025-12-12)

1. Attempted to move ~/Music (~210GB) from encrypted Data volume to unencrypted Publica volume
2. Publica showed 232GB available but operation failed with "not enough space"
3. Used `ditto` command as workaround
4. `ditto` filled the entire container, exhausting all free space
5. Time Machine snapshots failed (require container-level free space)
6. System became unresponsive
7. Recovery: delete backed-up data, remove partition, 3 hours lost

### Root Cause

APFS uses container-based architecture:
- Multiple volumes in one container share the same physical storage pool
- macOS Finder shows per-volume "available" space (misleading)
- This doesn't account for space needed by other volumes in the container
- When container fills, all volumes are affected simultaneously
- Time Machine needs container-level free space for snapshots

### Why This Keeps Happening

**Human cognitive limits**: If users must maintain a mental model of how every APFS volume interacts with every other volume in a shared container, the filesystem has failed its purpose of abstracting complexity.

**Solution**: Automate monitoring so computers manage this complexity, not humans.

### Why HFS+ Was Better for This

- Predictable space reporting (what you see is what you get)
- Volume isolation (each volume has discrete space allocation)
- No hidden sharing mechanisms
- File operations behave predictably

**Decision**: Reverted significant storage to HFS+ for predictability and performance.

---

## Monitoring Solution

### What It Does

- Monitors APFS container free space at the physical container level (not misleading volume level)
- Checks every 5 minutes
- Sends macOS notifications when thresholds breached
- Logs all status checks
- Runs automatically at boot
- Restarts automatically if it crashes

### Current Configuration

- **Warning Threshold**: 30 GB free
- **Critical Threshold**: 20 GB free
- **Check Interval**: 5 minutes
- **Log File**: `/var/log/apfs-monitor.log`
- **Error Log**: `/var/log/apfs-monitor.err`

### Installation Status

```
Binary:        /usr/local/bin/apfs-monitor (installed)
LaunchDaemon:  /Library/LaunchDaemons/com.local.apfs-monitor.plist (loaded)
Status:        Running (PID varies)
Logs:          /var/log/apfs-monitor.log (active)
```

### Files Created

#### Repository Files (/Users/tdsanchez/apfs-monitor/)

**Source Code:**
```
main.go                              # Go daemon source code
go.mod                               # Go module file
.gitignore                           # Git ignore patterns
```

**Documentation:**
```
MASTER-DOCUMENTATION.md              # Complete reference (this file)
SUMMARY.md                           # Quick overview
README.md                            # Tool documentation
QUICKSTART.md                        # 30-second setup guide
QUICK-REFERENCE.md                   # Command cheat sheet
CHANGELOG.md                         # Version history
INCIDENT-REPORT.md                   # 2025-12-12 incident analysis
INDEX.md                             # Documentation index
SYSTEM-CARD-NOTES.md                 # Model behavior notes
TEST-LOG.md                          # Testing documentation
```

**Scripts:**
```
install.sh                           # One-command installation
uninstall.sh                         # Clean removal
update-daemon.sh                     # Update running daemon
test.sh                              # Verification script
set-thresholds-100-50.sh             # Set thresholds to 100/50 GB
```

**Configuration:**
```
com.local.apfs-monitor.plist         # LaunchDaemon template
```

#### Installed System Files

**Binary:**
```
/usr/local/bin/apfs-monitor          # Compiled Go binary (2.6 MB)
```

**LaunchDaemon:**
```
/Library/LaunchDaemons/com.local.apfs-monitor.plist
                                     # Active daemon configuration
                                     # Owner: root:wheel
                                     # Permissions: 644
```

**Log Files:**
```
/var/log/apfs-monitor.log            # Status and check logs
/var/log/apfs-monitor.err            # Error output (stderr)
/var/log/apfs-monitor.out            # Standard output (stdout)
```

#### External Documentation

**User Home Directory:**
```
/Users/tdsanchez/apfs-space-issue-documentation.md
                                     # Original incident report

/Users/tdsanchez/APFS-MONITORING-INDEX.md
                                     # Quick access documentation index
```

#### Git Repository

**Status:** Initialized with trunk-based development workflow
**Initial commit:** 0117cee
**Branch:** master
**Files tracked:** 19 files, 2369 lines

---

## Usage Guide

### Check Current Status

```bash
# View recent log entries
tail -20 /var/log/apfs-monitor.log

# Watch logs in real-time
tail -f /var/log/apfs-monitor.log

# Check daemon is running
sudo launchctl list | grep apfs-monitor

# Run manual check
/usr/local/bin/apfs-monitor
```

### Daemon Management

```bash
# Start daemon
sudo launchctl load /Library/LaunchDaemons/com.local.apfs-monitor.plist

# Stop daemon
sudo launchctl unload /Library/LaunchDaemons/com.local.apfs-monitor.plist

# Restart daemon
sudo launchctl unload /Library/LaunchDaemons/com.local.apfs-monitor.plist
sudo launchctl load /Library/LaunchDaemons/com.local.apfs-monitor.plist

# Check daemon status
sudo launchctl list | grep apfs-monitor
```

### Update Monitor After Code Changes

```bash
cd /Users/tdsanchez/apfs-monitor
sudo bash update-daemon.sh
```

### Adjust Thresholds

Edit `/Library/LaunchDaemons/com.local.apfs-monitor.plist`:

```xml
<string>-warning</string>
<string>30</string>      <!-- Change this value (GB) -->
<string>-critical</string>
<string>20</string>      <!-- Change this value (GB) -->
```

Then reload daemon:
```bash
sudo launchctl unload /Library/LaunchDaemons/com.local.apfs-monitor.plist
sudo launchctl load /Library/LaunchDaemons/com.local.apfs-monitor.plist
```

**Recommended for your 994GB container**:
- Warning: 40-50 GB
- Critical: 25-30 GB

### Change Check Interval

Edit plist and change:
```xml
<string>-interval</string>
<string>5m</string>      <!-- Options: 1m, 5m, 10m, 30m, 1h, etc. -->
```

---

## When You Get An Alert

### Warning Alert (< 30 GB free)

You have time to clean up proactively. Don't panic.

1. **Check what's using space**:
   ```bash
   diskutil apfs list
   sudo du -h -d 1 /System/Volumes/Data | sort -h | tail -20
   ```

2. **Safe cleanup options** (no backup needed first):
   ```bash
   # Delete local Time Machine snapshots (already backed up)
   tmutil listlocalsnapshots /
   tmutil deletelocalsnapshots /

   # Clear system caches (regeneratable)
   sudo rm -rf /Library/Caches/*
   rm -rf ~/Library/Caches/*

   # Empty trash
   rm -rf ~/.Trash/*

   # Docker cleanup (if applicable)
   docker system prune -a
   ```

3. **Verify space freed**:
   ```bash
   /usr/local/bin/apfs-monitor
   ```

### Critical Alert (< 20 GB free)

Immediate action required. System is at risk.

1. **Stop any large file operations immediately**
2. **Run emergency cleanup** (all commands from Warning section)
3. **Consider removing unnecessary APFS volumes**
4. **Move large files to external storage**
5. **Delete old downloads, large email attachments**

### Nuclear Option

If space is critically low and nothing else works:
```bash
# List all local snapshots
tmutil listlocalsnapshots /

# Delete ALL local snapshots
sudo tmutil deletelocalsnapshots /

# This can free 20-50GB instantly
```

---

## Diagnostic Commands

### Check APFS Container Status

```bash
# Full container details
diskutil apfs list

# List physical disks and partitions
diskutil list

# Container-level space (accurate)
diskutil apfs list | grep -E "(Container disk|Capacity)"

# Volume-level space (MISLEADING - don't trust this)
df -h
```

### Check Time Machine Status

```bash
# Current status
tmutil status

# List local snapshots
tmutil listlocalsnapshots /

# Delete specific snapshot
tmutil deletelocalsnapshots YYYY-MM-DD-HHMMSS

# Delete all local snapshots
tmutil deletelocalsnapshots /
```

### Find Large Files

```bash
# Top-level directories by size
sudo du -h -d 1 /System/Volumes/Data | sort -h | tail -20

# Find files larger than 1GB
sudo find /System/Volumes/Data -type f -size +1G 2>/dev/null

# Find largest files in home directory
du -ha ~/ | sort -h | tail -50
```

---

## Technical Details

### How the Monitor Works

1. Executes `diskutil apfs list` every 5 minutes
2. Parses output using regex patterns:
   - Container name: `Container (disk\d+)`
   - Used space: `Capacity In Use By Volumes:\s+([\d]+)\s+B`
   - Free space: `Capacity Not Allocated:\s+([\d]+)\s+B`
3. Converts bytes to GB
4. Compares against thresholds
5. Logs status and sends notifications if needed
6. Repeats

### Why Go?

- Fast compilation
- Single static binary (no dependencies)
- Cross-platform
- Low memory footprint
- Excellent standard library for regex and command execution

### LaunchDaemon Configuration

```xml
RunAtLoad:     true  (starts at boot)
KeepAlive:     true  (auto-restart on crash)
ThrottleInterval: 300 seconds (prevent rapid restart loops)
```

### Log Rotation

Logs will grow over time. To prevent unbounded growth:

```bash
# Check log size
ls -lh /var/log/apfs-monitor.log

# Manually rotate if needed
sudo mv /var/log/apfs-monitor.log /var/log/apfs-monitor.log.old
sudo touch /var/log/apfs-monitor.log
sudo launchctl unload /Library/LaunchDaemons/com.local.apfs-monitor.plist
sudo launchctl load /Library/LaunchDaemons/com.local.apfs-monitor.plist
```

Consider setting up periodic log rotation via cron or newsyslog.

---

## Troubleshooting

### No Log Output

```bash
# Check if daemon is running
sudo launchctl list | grep apfs-monitor

# Check error log
cat /var/log/apfs-monitor.err

# Run manually to see output
/usr/local/bin/apfs-monitor

# Rebuild and reinstall
cd /Users/tdsanchez/apfs-monitor
sudo bash update-daemon.sh
```

### Daemon Not Starting

```bash
# Check plist syntax
plutil -lint /Library/LaunchDaemons/com.local.apfs-monitor.plist

# Check permissions
ls -la /Library/LaunchDaemons/com.local.apfs-monitor.plist
# Should be: -rw-r--r-- root wheel

# Check binary permissions
ls -la /usr/local/bin/apfs-monitor
# Should be: -rwxr-xr-x root admin

# Try loading manually
sudo launchctl load -w /Library/LaunchDaemons/com.local.apfs-monitor.plist

# Check system log for errors
log show --predicate 'subsystem == "com.apple.launchd"' --last 1h | grep apfs
```

### Notifications Not Appearing

```bash
# Check notification settings
# System Preferences > Notifications > Script Editor (osascript)

# Test notification manually
osascript -e 'display notification "Test" with title "Test Title"'

# Disable notifications (edit plist)
<string>-notify</string>
<string>false</string>
```

### `getcwd` Errors During Installation

These occur when your shell is in a directory with filesystem issues. Solution:

```bash
cd /tmp
# Then run commands
```

### Parser Not Working

Verify `diskutil apfs list` output format hasn't changed:

```bash
diskutil apfs list | head -50
```

If format changed, update regex patterns in `main.go` and rebuild.

---

## Maintenance

### Weekly

Check logs for any anomalies:
```bash
tail -100 /var/log/apfs-monitor.log
```

### Monthly

1. Verify daemon is still running
2. Check log file size
3. Review space trends across containers

### After macOS Updates

1. Verify daemon still running: `sudo launchctl list | grep apfs-monitor`
2. Test manually: `/usr/local/bin/apfs-monitor`
3. Check for `diskutil` output format changes

### Before Major System Changes

1. Note current space levels
2. Ensure Time Machine has recent backup
3. Consider temporarily lowering thresholds for extra safety

---

## Uninstallation

```bash
cd /Users/tdsanchez/apfs-monitor
./uninstall.sh

# Manually remove logs if desired
sudo rm /var/log/apfs-monitor.*

# Remove project directory
rm -rf /Users/tdsanchez/apfs-monitor
```

---

## Historical Context

### Incidents Since macOS Catalina (2019)

- **Frequency**: ~10 occurrences over 6 years
- **Time per incident**: ~3 hours
- **Total time lost**: ~30 hours
- **Pattern**: APFS container exhaustion preventing Time Machine snapshots
- **Trigger**: Large file operations between volumes in same container

### Evolution of Mitigation

1. **2019-2024**: Manual vigilance (failed 10 times)
2. **2024**: Reverted some storage to HFS+ for predictability
3. **2025**: Implemented automated monitoring (this solution)

### Design Philosophy Shift

From: "Users should understand APFS container architecture"
To: "Computers should manage APFS complexity automatically"

---

## Prevention Best Practices

### Do's

- ✅ Monitor container-level space, not volume-level
- ✅ Keep containers under 85-90% capacity
- ✅ Maintain 30-50GB free for snapshots and operations
- ✅ Use HFS+ for volumes where predictability matters
- ✅ Delete local TM snapshots regularly (they're redundant)
- ✅ Clear caches periodically
- ✅ Use `rsync` or `cp` for large file operations (they fail cleanly)

### Don'ts

- ❌ Don't trust Finder's "Available" space for APFS volumes
- ❌ Don't use `ditto` for large transfers on APFS (no pre-flight checks)
- ❌ Don't fill containers beyond 90% capacity
- ❌ Don't create unnecessary volumes in the same container
- ❌ Don't rely on manual monitoring (humans forget)

---

## Future Enhancements

Potential improvements to monitoring tool:

- [ ] Email notifications
- [ ] Webhook support for external monitoring systems
- [ ] Prometheus metrics endpoint
- [ ] Historical space usage tracking and graphing
- [ ] Predictive alerts ("will hit threshold in X days")
- [ ] Per-volume breakdown in alerts
- [ ] Integration with cleanup tools (automatic cache clearing)
- [ ] Web dashboard
- [ ] Slack/Discord notifications
- [ ] Automatic snapshot cleanup when threshold approached

---

## Quick Reference Card

### Emergency Commands

```bash
# Check space NOW
/usr/local/bin/apfs-monitor

# Free space FAST
tmutil deletelocalsnapshots /
sudo rm -rf /Library/Caches/*
rm -rf ~/Library/Caches/*

# Check daemon status
sudo launchctl list | grep apfs-monitor

# View logs
tail -f /var/log/apfs-monitor.log
```

### Important Paths

```
Monitor binary:  /usr/local/bin/apfs-monitor
Daemon config:   /Library/LaunchDaemons/com.local.apfs-monitor.plist
Logs:            /var/log/apfs-monitor.log
Source code:     /Users/tdsanchez/apfs-monitor/
Update script:   /Users/tdsanchez/apfs-monitor/update-daemon.sh
```

### Key Thresholds (Current)

```
Warning:  30 GB free
Critical: 20 GB free
Interval: 5 minutes
```

---

## Conclusion

This monitoring solution automates the vigilance required to prevent APFS container space exhaustion. By checking every 5 minutes and alerting before critical thresholds, it provides the early warning needed to avoid 3-hour recovery sessions.

**Goal achieved**: Computer manages APFS complexity, not human.

---

## Document Metadata

- **Created**: 2025-12-12
- **Author**: Automated documentation via Claude Code
- **Version**: 1.0
- **System**: macOS with APFS
- **Status**: Production deployed

---

## See Also

- `README.md` - Detailed tool documentation
- `QUICKSTART.md` - 30-second installation guide
- `apfs-space-issue-documentation.md` - Full incident report
- `main.go` - Source code with inline comments
