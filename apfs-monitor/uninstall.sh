#!/bin/bash

set -e

echo "Unloading APFS monitor daemon..."
if sudo launchctl list | grep -q "com.local.apfs-monitor"; then
    sudo launchctl unload /Library/LaunchDaemons/com.local.apfs-monitor.plist
    echo "Daemon unloaded."
else
    echo "Daemon not running."
fi

echo "Removing launchd plist..."
sudo rm -f /Library/LaunchDaemons/com.local.apfs-monitor.plist

echo "Removing binary..."
sudo rm -f /usr/local/bin/apfs-monitor

echo ""
echo "Uninstallation complete!"
echo ""
echo "Log files remain in /var/log/apfs-monitor.*"
echo "Remove them manually if desired:"
echo "  sudo rm /var/log/apfs-monitor.*"
