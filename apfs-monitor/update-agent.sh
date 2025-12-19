#!/bin/bash
set -e
cd /Users/tdsanchez/apfs-monitor
go build -o apfs-monitor main.go
sudo cp apfs-monitor /usr/local/bin/
launchctl unload ~/Library/LaunchAgents/com.local.apfs-monitor.plist
launchctl load ~/Library/LaunchAgents/com.local.apfs-monitor.plist
echo "Agent updated and restarted"
tail -10 ~/Library/Logs/apfs-monitor.log
