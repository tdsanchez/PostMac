# APFS Monitor - Quick Reference Card

## Status Check (30 seconds)

```bash
# Is it running?
sudo launchctl list | grep apfs-monitor

# What's it saying?
tail -20 /var/log/apfs-monitor.log

# Check now (manual)
/usr/local/bin/apfs-monitor
```

---

## Emergency Space Cleanup (5 minutes)

```bash
# 1. Delete TM snapshots (20-50GB usually)
tmutil deletelocalsnapshots /

# 2. Clear caches (5-20GB usually)
sudo rm -rf /Library/Caches/*
rm -rf ~/Library/Caches/*

# 3. Empty trash
rm -rf ~/.Trash/*

# 4. Check result
/usr/local/bin/apfs-monitor
```

---

## Daemon Management

```bash
# Start
sudo launchctl load /Library/LaunchDaemons/com.local.apfs-monitor.plist

# Stop
sudo launchctl unload /Library/LaunchDaemons/com.local.apfs-monitor.plist

# Restart
sudo launchctl unload /Library/LaunchDaemons/com.local.apfs-monitor.plist
sudo launchctl load /Library/LaunchDaemons/com.local.apfs-monitor.plist

# Status
sudo launchctl list | grep apfs-monitor
```

---

## Update After Code Changes

```bash
cd /Users/tdsanchez/apfs-monitor
sudo bash update-daemon.sh
```

---

## Adjust Thresholds

1. Edit: `sudo nano /Library/LaunchDaemons/com.local.apfs-monitor.plist`
2. Change values under `-warning` and `-critical`
3. Restart daemon (see Daemon Management above)

**Recommended for 994GB container:**
- Warning: 40-50 GB
- Critical: 25-30 GB

---

## Diagnostic Commands

```bash
# Container-level space (accurate)
diskutil apfs list

# Volume-level space (MISLEADING!)
df -h

# Time Machine status
tmutil status

# Find large files
sudo du -h -d 1 /System/Volumes/Data | sort -h | tail -20

# List TM snapshots
tmutil listlocalsnapshots /
```

---

## Important Files

```
Binary:       /usr/local/bin/apfs-monitor
Config:       /Library/LaunchDaemons/com.local.apfs-monitor.plist
Logs:         /var/log/apfs-monitor.log
Source:       /Users/tdsanchez/apfs-monitor/
Update:       /Users/tdsanchez/apfs-monitor/update-daemon.sh
```

---

## Current Thresholds

- **Warning**: 30 GB
- **Critical**: 20 GB
- **Interval**: 5 minutes

---

## When Alert Fires

1. Don't panic - you have warning time
2. Run emergency cleanup (see above)
3. Check what's using space: `diskutil apfs list`
4. Verify fixed: `/usr/local/bin/apfs-monitor`

---

## Uninstall

```bash
cd /Users/tdsanchez/apfs-monitor
./uninstall.sh
```

---

**For complete documentation, see MASTER-DOCUMENTATION.md**
