#!/bin/bash
sudo sed -i '' -e 's|<string>30</string>|<string>100</string>|' -e 's|<string>20</string>|<string>50</string>|' /Library/LaunchDaemons/com.local.apfs-monitor.plist
sudo launchctl unload /Library/LaunchDaemons/com.local.apfs-monitor.plist
sudo launchctl load /Library/LaunchDaemons/com.local.apfs-monitor.plist
/usr/local/bin/apfs-monitor
