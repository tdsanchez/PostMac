#!/bin/bash
set -e

echo "Switching from LaunchDaemon (root) to LaunchAgent (user)..."

echo "Unloading root daemon..."
sudo launchctl unload /Library/LaunchDaemons/com.local.apfs-monitor.plist

echo "Removing root daemon plist..."
sudo rm /Library/LaunchDaemons/com.local.apfs-monitor.plist

echo "Creating user log directory..."
mkdir -p ~/Library/Logs

echo "Installing LaunchAgent plist..."
cp com.local.apfs-monitor-agent.plist ~/Library/LaunchAgents/com.local.apfs-monitor.plist

echo "Loading LaunchAgent (user-level)..."
launchctl load ~/Library/LaunchAgents/com.local.apfs-monitor.plist

echo ""
echo "Switch complete!"
echo ""
echo "Logs now at:"
echo "  ~/Library/Logs/apfs-monitor.log"
echo ""
echo "Check status:"
echo "  launchctl list | grep apfs-monitor"
echo ""
echo "View logs:"
echo "  tail -f ~/Library/Logs/apfs-monitor.log"
