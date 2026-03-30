# APFS Monitor - Quick Start

## TL;DR

Prevent APFS container space exhaustion from destroying your day.

## Installation (30 seconds)

```bash
cd apfs-monitor
./test.sh        # Verify it works
./install.sh     # Install as system daemon
```

Done. The monitor now runs automatically and will alert you before space issues occur.

## What You Get

- Alerts when APFS container space drops below 30GB (warning) or 20GB (critical)
- Runs every 5 minutes automatically
- macOS notifications + log file
- Starts automatically at boot

## Check It's Working

```bash
# View recent activity
tail -20 /var/log/apfs-monitor.log

# Check daemon status
sudo launchctl list | grep apfs-monitor

# Run manual check
/usr/local/bin/apfs-monitor
```

## Customize Thresholds

Edit `/Library/LaunchDaemons/com.local.apfs-monitor.plist` and change:

```xml
<string>-warning</string>
<string>30</string>      <!-- Change this -->
<string>-critical</string>
<string>20</string>      <!-- Change this -->
```

Then reload:
```bash
sudo launchctl unload /Library/LaunchDaemons/com.local.apfs-monitor.plist
sudo launchctl load /Library/LaunchDaemons/com.local.apfs-monitor.plist
```

## Uninstall

```bash
./uninstall.sh
```

## When You Get An Alert

1. **Don't panic** - you have warning time now
2. Check what's using space: `diskutil apfs list`
3. Free up space:
   - Delete old Time Machine local snapshots: `tmutil listlocalsnapshots /` then `tmutil deletelocalsnapshots <date>`
   - Clear system caches: `sudo rm -rf /System/Library/Caches/*`
   - Remove unnecessary APFS volumes
   - Move large files to external storage
4. Verify space is freed: `/usr/local/bin/apfs-monitor`

## The Goal

Never spend 3 hours recovering from APFS space exhaustion again.
