# APFS Container Space Monitor

A lightweight Go daemon that monitors APFS container free space and alerts you before Time Machine snapshots fail.

## The Problem This Solves

APFS volumes in the same container share physical space. When the container fills up:
- Time Machine snapshots fail
- File operations that should succeed (based on volume-reported space) fail
- System becomes unstable
- Recovery takes hours

This monitor watches container-level space and alerts you before disaster strikes.

## Features

- **Auto-detects boot container** - Monitors only the boot volume's container, ignoring external drives
- Monitors APFS container free space (not just volume space)
- Configurable warning and critical thresholds
- macOS notification integration
- Can run as one-time check or continuous daemon
- Lightweight - minimal CPU and memory usage
- Logging to file or stdout
- Optional manual container selection for advanced use cases

## Installation

### Build from source

```bash
cd apfs-monitor
go build -o apfs-monitor main.go
```

### Install to system

```bash
sudo cp apfs-monitor /usr/local/bin/
sudo chmod +x /usr/local/bin/apfs-monitor
```

## Usage

### One-time check

```bash
./apfs-monitor
```

### Run as daemon (manual)

```bash
./apfs-monitor -daemon -interval 5m -log /var/log/apfs-monitor.log
```

### Run as launchd daemon (recommended)

This will automatically start the monitor at boot and keep it running:

```bash
# Copy the plist file
sudo cp com.local.apfs-monitor.plist /Library/LaunchDaemons/

# Load the daemon
sudo launchctl load /Library/LaunchDaemons/com.local.apfs-monitor.plist

# Check status
sudo launchctl list | grep apfs-monitor
```

### View logs

```bash
tail -f /var/log/apfs-monitor.log
```

## Configuration

### Command-line flags

- `-daemon` - Run as continuous daemon (default: false, one-time check)
- `-interval duration` - Check interval in daemon mode (default: 5m)
- `-warning float` - Warning threshold in GB (default: 30.0)
- `-critical float` - Critical threshold in GB (default: 20.0)
- `-log string` - Log file path (default: stdout)
- `-notify` - Send macOS notifications (default: true)
- `-container string` - Specific APFS container to monitor (e.g., disk3). If empty, auto-detects boot container (default: auto-detect)

### Examples

Check every 10 minutes, warn at 50GB, critical at 25GB:
```bash
./apfs-monitor -daemon -interval 10m -warning 50 -critical 25
```

One-time check with custom thresholds:
```bash
./apfs-monitor -warning 40 -critical 20
```

Monitor a specific container (advanced):
```bash
./apfs-monitor -daemon -container disk5 -warning 100 -critical 50
```

## How It Works

1. Auto-detects boot container by parsing `diskutil info /` (or uses manually specified container)
2. Runs `diskutil apfs list` to get container information
3. Parses output to extract free space at container level
4. Filters to only the boot container (ignores external drives and secondary containers)
5. Compares free space against thresholds
6. Logs status and sends notifications if thresholds breached
7. Repeats at configured interval (in daemon mode)

## Thresholds

### Recommended Settings

For a typical setup with Time Machine:
- **Warning**: 30-50 GB (gives you time to clean up)
- **Critical**: 20-30 GB (urgent action needed)

Adjust based on your container size and snapshot requirements.

### Why These Numbers?

- Time Machine snapshots can consume 10-20GB or more
- System operations need headroom
- APFS performance degrades when container is >90% full
- Better to get false warnings than no warnings

## Uninstall

### Remove launchd daemon

```bash
sudo launchctl unload /Library/LaunchDaemons/com.local.apfs-monitor.plist
sudo rm /Library/LaunchDaemons/com.local.apfs-monitor.plist
```

### Remove binary

```bash
sudo rm /usr/local/bin/apfs-monitor
```

## Troubleshooting

### No notifications appearing

- Check that `-notify` is set to true (default)
- Verify you have notification permissions
- Check System Preferences > Notifications

### Daemon not starting

```bash
# Check launchd logs
sudo tail -f /var/log/system.log | grep apfs-monitor

# Manually test the binary
sudo /usr/local/bin/apfs-monitor
```

### Inaccurate space reporting

The monitor reads directly from `diskutil apfs list`, which reports container-level space. If this seems wrong:

```bash
# Manual check
diskutil apfs list

# Compare with volume-level (often misleading)
df -h
```

## Philosophy

**Computers should manage complexity, not create it.**

APFS containers require users to understand volume interactions that should be invisible. This tool automates that mental overhead so you can focus on actual work instead of babysitting filesystem quirks.

## Future Enhancements

Potential improvements:
- [ ] Email notifications
- [ ] Webhook support for integration with monitoring systems
- [ ] Prometheus metrics endpoint
- [ ] Historical space usage tracking
- [ ] Prediction of when thresholds will be reached
- [ ] Per-volume breakdown in alerts
- [ ] Integration with cleanup tools (automatic cache clearing, etc.)

## License

Use freely. No warranty. Don't blame me if APFS still ruins your day.
