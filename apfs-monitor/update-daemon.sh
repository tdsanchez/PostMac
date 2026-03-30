#!/bin/bash
set -e
cd /Users/tdsanchez/apfs-monitor
go build -o apfs-monitor main.go
sudo cp /Users/tdsanchez/apfs-monitor/apfs-monitor /usr/local/bin/
sudo launchctl unload /Library/LaunchDaemons/com.local.apfs-monitor.plist
sudo launchctl load /Library/LaunchDaemons/com.local.apfs-monitor.plist
echo "Daemon updated and restarted"
tail -20 /var/log/apfs-monitor.log
