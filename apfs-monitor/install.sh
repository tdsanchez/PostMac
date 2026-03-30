#!/bin/bash

set -e

echo "Building APFS Monitor..."
go build -o apfs-monitor main.go

echo "Installing binary to /usr/local/bin/..."
sudo cp apfs-monitor /usr/local/bin/
sudo chmod +x /usr/local/bin/apfs-monitor

echo "Installing launchd plist..."
sudo cp com.local.apfs-monitor.plist /Library/LaunchDaemons/
sudo chown root:wheel /Library/LaunchDaemons/com.local.apfs-monitor.plist
sudo chmod 644 /Library/LaunchDaemons/com.local.apfs-monitor.plist

echo "Creating log directory..."
sudo mkdir -p /var/log
sudo touch /var/log/apfs-monitor.log
sudo touch /var/log/apfs-monitor.err
sudo touch /var/log/apfs-monitor.out

# Unload if already loaded (for updates)
if sudo launchctl list | grep -q "com.local.apfs-monitor"; then
    echo "Unloading existing daemon..."
    sudo launchctl unload /Library/LaunchDaemons/com.local.apfs-monitor.plist
fi

echo "Loading launchd daemon..."
sudo launchctl load /Library/LaunchDaemons/com.local.apfs-monitor.plist

echo ""
echo "Installation complete!"
echo ""
echo "The APFS monitor is now running and will start automatically at boot."
echo ""
echo "View logs with:"
echo "  tail -f /var/log/apfs-monitor.log"
echo ""
echo "Check status with:"
echo "  sudo launchctl list | grep apfs-monitor"
echo ""
echo "Test immediately with:"
echo "  /usr/local/bin/apfs-monitor"
