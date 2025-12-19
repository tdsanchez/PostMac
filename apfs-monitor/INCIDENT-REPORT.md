# APFS Container Space Exhaustion Issue

## Problem Summary

APFS volumes within the same container share a common physical space pool. When one volume fills up, it can prevent other volumes from functioning properly and block Time Machine snapshots from being created, even if those other volumes appear to have free space.

**This antipattern requires constant mental modeling of inter-volume dependencies. If users must manually track how every APFS volume interacts with every other volume in the container, why have a computer at all?**

## Date of Latest Occurrence

2025-12-12

## What Happened This Time

### Initial State
- Music folder location: `~/Music` (on encrypted APFS volume)
- Music folder size: ~210GB
- Target volume: Unencrypted APFS volume on same physical NVMe device
- Target volume available space (reported): 232GB
- Expected result: Move should succeed with ~22GB remaining

### The Failure
1. macOS GUI reported "not enough space" despite 232GB > 210GB
2. Attempted workaround: Used `ditto` command to copy the data
3. Result: `ditto` filled BOTH the target volume AND the entire physical container
4. System became unresponsive ("dead in the water")
5. Time Machine backups failed due to insufficient space for snapshots

### Recovery
- Deleted large amounts of backed-up data to free container space
- Decided to remove one of 3 co-partitions on boot NVMe (in addition to boot and data partitions)
  - This is an "easy win" for freeing space
  - But represents additional complexity that shouldn't be necessary
- Time Machine backups resumed after sufficient space was freed
- Estimated time lost: ~3 hours
- **Impact**: Complete derailment from actual work

## Root Cause

### APFS Architecture Issue
APFS uses a container-based architecture where:
- Multiple volumes exist within a single APFS container
- All volumes share the same physical storage pool
- Free space is dynamically allocated from the shared pool
- Each volume can grow until the entire container is full

### Why This Happens
1. **GUI reporting inaccuracy**: macOS Finder shows per-volume "available" space, but this doesn't account for:
   - Space reserved for snapshots
   - Space needed by other volumes in the container
   - Container-level overhead

2. **ditto behavior**: The `ditto` command doesn't perform pre-flight space checks properly in APFS environments. It begins copying and continues until the physical container is exhausted.

3. **Snapshot requirements**: Time Machine requires free space at the container level to create APFS snapshots. When the container fills, snapshot creation fails.

## Historical Context

- **Frequency**: ~10 occurrences since macOS Catalina (2019)
- **Impact per occurrence**: ~3 hours of recovery time
- **Total impact**: ~30 hours over 6 years
- **Pattern**: This is a known antipattern in APFS usage

## Workaround: Reverting to HFS+

Due to both this space management issue and performance concerns, significant storage has been reverted to HFS+ formatting.

### Why HFS+ Instead of APFS

1. **Predictable space reporting**: HFS+ volumes report actual available space accurately
2. **Volume isolation**: Each HFS+ volume has its own discrete space allocation - no hidden sharing
3. **Performance**: Better performance characteristics for certain workloads
4. **Mental overhead**: No need to maintain a mental model of how every volume in a container interacts with others
5. **Operational simplicity**: File operations behave predictably - if space is shown, space is actually available

### The Philosophical Problem

APFS was designed to be "modern" and provide flexibility through dynamic space allocation. However:
- The flexibility creates unpredictability
- Users cannot trust reported space availability
- Simple file operations can cascade into system-wide failures
- Recovery requires deep understanding of container architecture

**If a filesystem requires users to constantly think about internal volume interactions, it has failed its primary purpose: abstracting storage complexity away from users.**

## Commands to Diagnose

### Check APFS container structure
```bash
diskutil list
```

### Check APFS container space usage
```bash
diskutil apfs list
```

### Check Time Machine status
```bash
tmutil status
```

### Check for APFS snapshots
```bash
tmutil listlocalsnapshots /
```

## Prevention Strategies

### Manual Prevention
1. Always check container-level free space, not just volume-level
2. Never fill a volume beyond 85-90% of total container capacity
3. Maintain at least 20-30GB free at container level for snapshots
4. Use `rsync` or `cp` with verbose output instead of `ditto` for large transfers
5. **Consider HFS+ for volumes where predictability matters more than APFS features**

### Automated Prevention (Proposed)
Create a monitoring daemon that:
- Runs at system startup
- Periodically checks APFS container free space
- Sends alerts when free space drops below threshold
- Can be triggered via cron or launchd
- Implemented in Go for cross-platform compatibility and low overhead
- **Goal**: Computer manages complexity so humans don't have to

## Resolution (2025-12-12)

### Implemented Solution

✅ **Go-based monitoring daemon deployed**
- Installed at: `/usr/local/bin/apfs-monitor`
- Running as: launchd daemon (com.local.apfs-monitor)
- Checking: Every 5 minutes
- Logging to: `/var/log/apfs-monitor.log`
- Status: Active and monitoring 5 APFS containers

### Configuration
- Warning threshold: 30 GB free
- Critical threshold: 20 GB free
- Notifications: macOS native notifications enabled
- Auto-start: Yes (via launchd)
- Auto-restart on crash: Yes

### Current State
- Boot container (disk3): 31.54 GB free (96.6% used)
- Still close to warning threshold - partition removal in progress
- All other containers healthy

### Planned Actions
1. ✅ Deploy monitoring daemon (COMPLETE)
2. 🔄 Remove unnecessary partition from boot container (IN PROGRESS)
3. ⏳ Adjust thresholds after space freed
4. ⏳ Evaluate additional volumes for HFS+ conversion

### Documentation Created
- `/Users/tdsanchez/apfs-monitor/MASTER-DOCUMENTATION.md` - Complete reference
- `/Users/tdsanchez/apfs-monitor/README.md` - Tool documentation
- `/Users/tdsanchez/apfs-monitor/QUICKSTART.md` - Installation guide
- This file - Incident report and analysis

### Success Metrics
- **Prevention**: Alerts before space critical
- **Early warning**: 5-minute check interval
- **Automation**: No manual vigilance required
- **Reliability**: Auto-restart, logging, notifications
- **Goal**: Prevent future 3-hour recovery sessions

### Lessons Applied
1. ✅ Automated monitoring instead of manual vigilance
2. ✅ Computer manages complexity, not human
3. ✅ Container-level monitoring (not misleading volume-level)
4. ✅ Early warning system with configurable thresholds
5. ✅ Comprehensive documentation for future reference

## Technical Details

### APFS Container vs Volume
- **Container**: Physical storage allocation unit
- **Volume**: Logical filesystem within a container
- **Key insight**: Volumes share container space dynamically
- **Problem**: This sharing is opaque to users and poorly reported by the OS

### Space Reporting Commands
```bash
# Container-level space
diskutil apfs list | grep -A 20 "Container disk"

# Volume-level space (misleading!)
df -h

# Actual available space calculation needed
# Total Container Free - Sum(All Volume Reserves) - Snapshot Reserve
```

## References

- Apple Developer Documentation: APFS
- Time Machine snapshot requirements
- Historical incidents: 2019-present (~10 occurrences)

## Lessons Learned

1. **Never trust GUI space reporting** for APFS volumes
2. **ditto is dangerous** for large transfers on APFS
3. **Container-level monitoring is essential** for APFS health
4. **Automation is necessary** - manual vigilance has failed 10 times
5. **Prevention > Recovery** - 3 hours per incident is too costly
6. **HFS+ remains viable** for scenarios requiring predictable space management
7. **Complexity should be managed by computers, not users** - if a filesystem requires constant mental modeling, it's a design failure
