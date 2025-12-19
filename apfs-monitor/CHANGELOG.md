# APFS Monitor - Changelog

All notable changes to this project will be documented in this file.

---

## [1.1.0] - 2025-12-18

### Added
- **Boot container auto-detection**: Automatically detects and monitors only the boot volume's APFS container
- New `-container` flag: Manually specify which container to monitor (e.g., `-container disk3`)
- `getBootContainer()` function: Parses `diskutil info /` to identify boot container
- Smart container filtering: Ignores non-boot APFS containers (external drives, secondary volumes)

### Changed
- Monitoring behavior now defaults to boot container only (previously monitored all containers)
- Log output now shows which container is being monitored: "Auto-detected boot container: disk3"
- Config struct extended with `containerFilter` field

### Fixed
- **Issue**: Alerts triggered on arbitrary non-system APFS containers (external drives, data volumes)
- **Impact**: False positives from external drives filling up, causing alert fatigue
- **Solution**: Auto-detect boot container via `diskutil info /` and filter to only that container

### Technical Details
- Boot container detected by parsing "APFS Container" field from `diskutil info /`
- Fallback to "Part of Whole" field if APFS Container not found
- Container filtering applied in `checkAndReport()` before threshold checks
- Manual override available via `-container` flag for edge cases

### Migration Notes
- Existing installations will automatically adopt boot-only monitoring on next restart
- No configuration changes required
- To monitor all containers (old behavior), use `-container ""` (not recommended)

---

## [1.0.0] - 2025-12-12

### Added
- Initial release of APFS container space monitoring daemon
- Go-based monitoring application (`main.go`)
- LaunchDaemon configuration for automatic startup
- macOS notification support via osascript
- Configurable warning and critical thresholds
- Configurable check interval
- Logging to file or stdout
- One-time check mode and continuous daemon mode
- Installation script (`install.sh`)
- Uninstallation script (`uninstall.sh`)
- Update script (`update-daemon.sh`)
- Test/verification script (`test.sh`)
- Comprehensive documentation:
  - MASTER-DOCUMENTATION.md (complete reference)
  - SUMMARY.md (quick overview)
  - README.md (tool documentation)
  - QUICKSTART.md (30-second setup)
  - QUICK-REFERENCE.md (command reference card)
  - CHANGELOG.md (this file)

### Technical Details
- Monitors APFS container free space at physical container level
- Parses `diskutil apfs list` output using regex
- Checks 5 APFS containers on system
- Default thresholds: 30 GB warning, 20 GB critical
- Default check interval: 5 minutes
- Auto-restart on crash via launchd KeepAlive
- Runs at boot via launchd RunAtLoad

### Fixed
- Parser regex patterns corrected during initial deployment
  - Changed from "Capacity Consumed" to "Capacity In Use By Volumes"
  - Changed from "Capacity Free" to "Capacity Not Allocated"
  - Fixed container name pattern to match "Container diskN" format
  - Fixed volume pattern to match "+-> Volume" format

### Current Status
- Deployed and running as system daemon
- Monitoring 5 APFS containers
- Logging to /var/log/apfs-monitor.log
- Boot container (disk3) at 96.6% capacity (31.54 GB free)

### Known Issues
- None currently

### Future Enhancements Under Consideration
- Email notification support
- Webhook support for external monitoring
- Prometheus metrics endpoint
- Historical tracking and trend analysis
- Predictive alerts (time to threshold)
- Per-volume breakdown in alerts
- Automatic cleanup integration
- Web dashboard
- Slack/Discord notifications

---

## Version History

- **1.1.0** (2025-12-18): Boot container auto-detection, smart filtering
- **1.0.0** (2025-12-12): Initial release

---

## Update Process

When updating the monitor:

1. Make changes to `main.go`
2. Update this CHANGELOG.md
3. Increment version number in code (if implementing versioning)
4. Run update script: `sudo bash update-daemon.sh`
5. Verify: `tail -20 /var/log/apfs-monitor.log`
6. Test: `/usr/local/bin/apfs-monitor`

---

## Notes

This changelog follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) principles.

Version numbering: MAJOR.MINOR.PATCH
- MAJOR: Breaking changes
- MINOR: New features, backwards compatible
- PATCH: Bug fixes, backwards compatible
