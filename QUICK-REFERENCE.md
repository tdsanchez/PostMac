# Quick Reference - Common Commands

**Purpose**: Fast command lookup across all projects in this dev tree
**Last Updated**: 2025-12-12

---

## Project Status Checks

### media_server
```bash
# Check if server is running
lsof -i :8080

# View recent activity
tail -f media_server/server.log

# Check cache status
ls -lh "$(cat ~/.media-server-dir)/.media-server-cache.db" 2>/dev/null || echo "No cache found"

# View git status
cd media_server && git status
```

### apfs-monitor
```bash
# Check if agent is running
launchctl list | grep apfs-monitor

# View recent logs
tail -20 ~/Library/Logs/apfs-monitor.log

# Check for errors
cat ~/Library/Logs/apfs-monitor.err

# Manual space check
/usr/local/bin/apfs-monitor
```

---

## Building & Running

### media_server
```bash
# Build from source
cd media_server
go build -o media-server cmd/media-server/main.go

# Run with default settings (port 8080, current directory)
./media-server

# Run with custom settings
./media-server --dir=/Volumes/Media --port=9090

# Build and launch browser automatically
./build_server.sh
```

### apfs-monitor
```bash
# Build from source
cd apfs-monitor
go build -o apfs-monitor main.go

# Test locally
./apfs-monitor

# Test daemon mode (foreground)
./apfs-monitor -daemon -interval 1m

# Run test suite
./test.sh
```

---

## Installation & Updates

### media_server
```bash
# Install dependencies
cd media_server
go mod download

# Rebuild after changes
go build -o media-server cmd/media-server/main.go

# No installation needed - runs from directory
```

### apfs-monitor
```bash
# Install as user agent
cd apfs-monitor
bash switch-to-agent.sh

# Update after code changes
bash update-agent.sh

# Uninstall completely
bash uninstall.sh
```

---

## Service Management

### media_server (Manual)
```bash
# Start server
cd media_server && ./media-server --dir=/path/to/media &

# Stop server (graceful)
# Use shutdown button in web UI, or:
curl -X POST http://localhost:8080/api/shutdown

# Stop server (force)
pkill -f media-server
```

### apfs-monitor (LaunchAgent)
```bash
# Load agent (start)
launchctl load ~/Library/LaunchAgents/com.local.apfs-monitor.plist

# Unload agent (stop)
launchctl unload ~/Library/LaunchAgents/com.local.apfs-monitor.plist

# Restart agent
launchctl unload ~/Library/LaunchAgents/com.local.apfs-monitor.plist
launchctl load ~/Library/LaunchAgents/com.local.apfs-monitor.plist

# Check status
launchctl list | grep apfs-monitor
```

---

## Logs & Debugging

### media_server
```bash
# Server logs (if logging to file)
tail -f media_server/server.log

# Check for errors in recent git commits
cd media_server && git log --oneline -10

# Debug duplicates
cd media_server && ./debug_duplicates.sh

# Search git history
cd media_server && ./git_grep.sh "search term"
```

### apfs-monitor
```bash
# View logs (live)
tail -f ~/Library/Logs/apfs-monitor.log

# View error log
cat ~/Library/Logs/apfs-monitor.err

# Check last 20 entries
tail -20 ~/Library/Logs/apfs-monitor.log

# View system logs for LaunchAgent
log show --predicate 'processImagePath contains "apfs-monitor"' --last 1h
```

---

## Testing

### media_server
```bash
# Python load test (mixed workload)
cd media_server
python3 load_test.py --url http://localhost:8080 --workers 5 --duration 60

# Perl load test (cross-verification)
perl load_test.pl --url http://localhost:8080 --workers 5 --duration 60

# Stress test viewer with random mode
python3 load_test.py --url http://localhost:8080 --scenario viewer-random --workers 10

# API endpoint performance
python3 load_test.py --url http://localhost:8080 --scenario api-only --requests 1000

# Maximum stress
python3 load_test.py --url http://localhost:8080 --scenario stress --workers 20
```

### apfs-monitor
```bash
# Run test suite
cd apfs-monitor
./test.sh

# Manual notification test
osascript -e 'display alert "Test" message "Testing alerts"'

# Check actual APFS space
diskutil apfs list | grep -A 3 "Container disk3"

# Compare with monitor output
/usr/local/bin/apfs-monitor
```

---

## Configuration

### media_server
```bash
# Command-line flags
./media-server --help

# Common flags
--dir=/path/to/media    # Media directory to serve
--port=8080             # Port to listen on (default 8080)

# No config file - all configuration via flags
```

### apfs-monitor
```bash
# Edit thresholds
nano ~/Library/LaunchAgents/com.local.apfs-monitor.plist
# Change <string>100</string> for warning threshold
# Change <string>50</string> for critical threshold

# Reload after changes
launchctl unload ~/Library/LaunchAgents/com.local.apfs-monitor.plist
launchctl load ~/Library/LaunchAgents/com.local.apfs-monitor.plist

# Set thresholds script
cd apfs-monitor
bash set-thresholds-100-50.sh
```

---

## Documentation Access

### media_server
```bash
# Main technical overview
open media_server/PROJECT_OVERVIEW.md

# Development methodology
open media_server/🧠\ Development\ Methodology.md

# Open source vision
open media_server/🌐\ Open\ Source\ Vision.md

# Current improvements
open media_server/NEXT_CYCLE_IMPROVEMENTS.md

# Session history
open media_server/SESSIONS.md
```

### apfs-monitor
```bash
# Main overview
open apfs-monitor/PROJECT_OVERVIEW.md

# Quick start
open apfs-monitor/QUICKSTART.md

# Complete reference
open apfs-monitor/MASTER-DOCUMENTATION.md

# Incident analysis
open apfs-monitor/INCIDENT-REPORT.md

# Testing details
open apfs-monitor/TEST-LOG.md
```

### Dev Tree Level
```bash
# Dev tree overview
open README.md

# Development methodology
open 🧠\ DEVELOPMENT_METHODOLOGY.md

# Detailed project catalog
open PROJECT_INDEX.md

# This file
open QUICK-REFERENCE.md
```

---

## Git Operations

### Common Git Commands (Both Projects)
```bash
# View recent commits
git log --oneline -10

# View changes in last commit
git diff HEAD~1 HEAD

# Search commit messages
git log --grep="random mode"

# View file history
git log --follow -- path/to/file.go

# Show commit details
git show <commit-hash>

# Compact history view
git log --graph --oneline --all
```

### media_server Specific
```bash
cd media_server

# View session patches
ls -lh *_changes/

# View recent architectural changes
git log --oneline --grep="pagination\|cache\|FSEvents"

# Check word frequency in commits
cat git_word_frequency.txt
```

### apfs-monitor Specific
```bash
cd apfs-monitor

# View complete development history
git log --oneline

# See initial implementation
git show $(git rev-list --max-parents=0 HEAD)
```

---

## Emergency Commands

### media_server Issues
```bash
# Server won't start
pkill -f media-server
cd media_server && ./media-server

# Cache corruption
rm "$(cat ~/.media-server-dir)/.media-server-cache.db"
# Restart server (will rebuild cache)

# Port already in use
lsof -ti:8080 | xargs kill -9
# Then restart server

# Memory issues
# Reduce page size in browser:
# http://localhost:8080/tag/All?limit=100
```

### apfs-monitor Issues
```bash
# Agent not running
launchctl load ~/Library/LaunchAgents/com.local.apfs-monitor.plist

# Logs not updating
launchctl unload ~/Library/LaunchAgents/com.local.apfs-monitor.plist
launchctl load ~/Library/LaunchAgents/com.local.apfs-monitor.plist

# No alerts appearing
# Verify agent runs as user (not root)
ps aux | grep apfs-monitor

# Test alert manually
osascript -e 'display alert "Test" message "Testing"'
```

### APFS Space Emergency
```bash
# Check actual container space
diskutil apfs list | grep -A 3 "Container disk3"

# Quick space recovery
rm -rf ~/Library/Caches/*
sudo rm -rf /Library/Caches/*

# Delete Time Machine local snapshots
tmutil listlocalsnapshots /
tmutil deletelocalsnapshots /

# Find large files
du -sh ~/* | sort -h | tail -10
```

---

## Development Workflow

### Starting a Session
```bash
# 1. Navigate to project
cd media_server  # or cd apfs-monitor

# 2. Load context
cat PROJECT_OVERVIEW.md | head -100

# 3. Check recent changes
git log --oneline -5

# 4. Clarify mode
# Note-taking or code-making?
```

### Making Changes
```bash
# 1. Make code changes
nano main.go

# 2. Test locally
go build -o binary main.go
./binary

# 3. Update if deployed
bash update-agent.sh  # apfs-monitor
# or rebuild server     # media_server

# 4. Update documentation
nano PROJECT_OVERVIEW.md

# 5. Commit
git add -A
git commit -m "Description of changes"
```

### Ending a Session
```bash
# 1. Ensure all changes committed
git status

# 2. Update PROJECT_OVERVIEW.md if significant changes
nano PROJECT_OVERVIEW.md

# 3. Commit documentation updates
git add PROJECT_OVERVIEW.md
git commit -m "Update documentation for [feature]"

# 4. Create session notes if major work
# (See SESSIONS.md pattern in media_server)
```

---

## Troubleshooting Decision Tree

### "Server/Agent not responding"
```bash
# 1. Is it running?
launchctl list | grep apfs-monitor  # apfs-monitor
lsof -i :8080                       # media_server

# 2. Check logs for errors
tail ~/Library/Logs/apfs-monitor.log  # apfs-monitor
tail media_server/server.log          # media_server

# 3. Restart
launchctl unload/load ...  # apfs-monitor
pkill -f media-server && ./media-server  # media_server
```

### "Unexpected behavior after update"
```bash
# 1. Check what changed
git diff HEAD~1 HEAD

# 2. Read commit message
git log -1

# 3. Review PROJECT_OVERVIEW.md "Recent Changes"
grep -A 5 "Recent Changes" PROJECT_OVERVIEW.md

# 4. Consider rollback
git log --oneline -5  # find good commit
git checkout <commit-hash> -- .
```

### "Performance degradation"
```bash
# media_server specific
# 1. Check cache size
ls -lh .media-server-cache.db

# 2. Consider rebuild
mv .media-server-cache.db .media-server-cache.db.backup
# Restart server

# 3. Check for large categories
# Use pagination with smaller page size
# http://localhost:8080/tag/CategoryName?limit=100
```

---

## URLs & Endpoints

### media_server
```bash
# Homepage (category grid)
http://localhost:8080/

# Gallery view
http://localhost:8080/tag/CategoryName
http://localhost:8080/tag/CategoryName?page=2&limit=200

# Single file viewer
http://localhost:8080/view/CategoryName?file=/path/to/file.jpg

# API endpoints
http://localhost:8080/api/alltags          # GET all tags
http://localhost:8080/api/filelist?category=X  # GET file list
http://localhost:8080/api/addtag           # POST add tag
http://localhost:8080/api/removetag        # POST remove tag
http://localhost:8080/api/comment          # POST update comment
http://localhost:8080/api/metadata/path    # GET EXIF data
http://localhost:8080/api/quicklook        # POST open QuickLook
http://localhost:8080/api/deletefile       # POST move to trash
http://localhost:8080/api/shutdown         # POST shutdown server
http://localhost:8080/api/rescan           # POST trigger rescan
http://localhost:8080/api/scanstatus       # GET scan status
```

### apfs-monitor
```bash
# No web interface - command-line only
# Manual check
/usr/local/bin/apfs-monitor

# View logs
tail -f ~/Library/Logs/apfs-monitor.log
```

---

## Keyboard Shortcuts

### media_server (Single File View)
```
←     Previous file (browser back)
→     Next file (or random if in random mode)
Esc   Return to gallery (or stop slideshow)
T     Open tag input
L     Add "❤️" tag
1-5   Add star rating (1-★ through 5-★★★★★)
C     Edit comment
Q     QuickLook preview
S     Toggle slideshow
R     Toggle random mode
X     Delete file (with confirmation)
+     Increase slideshow speed
-     Decrease slideshow speed
```

### media_server (Gallery View)
```
↑↓←→  Navigate between files
Enter Open selected file
T     Add tag to selected file
E     Edit comment on selected file
Cmd/Ctrl+Click  Multi-select
S     Toggle sort mode (name → date → size → random)
Shift+S  Reverse sort order
```

### media_server (Homepage)
```
↑↓←→  Navigate between categories
Enter Open gallery view for category
Click Open preview file for category
```

---

## Performance Optimization

### media_server
```bash
# Reduce page size for large categories
http://localhost:8080/tag/Large?limit=100

# Clear localStorage cache (browser console)
localStorage.clear()

# Rebuild SQLite cache
rm .media-server-cache.db && restart server

# Check cache statistics
sqlite3 .media-server-cache.db "SELECT COUNT(*) FROM files;"
```

### apfs-monitor
```bash
# Adjust check interval (edit plist)
# Default: 300 seconds (5 minutes)
# Range: 60s - 3600s

# Adjust thresholds based on container size
# 1 TB container: 100GB warning, 50GB critical
# 2 TB container: 150GB warning, 75GB critical
```

---

## System Requirements

### Both Projects
- macOS 10.14+ (Catalina or later)
- Go 1.20+ (for building from source)
- User-level permissions (no root required)

### media_server Specific
- Modern browser (Chrome, Safari, Firefox)
- Sufficient RAM for cache (43MB per 168k files)
- APFS filesystem (for extended attributes)

### apfs-monitor Specific
- APFS container setup
- Notification permissions enabled
- LaunchAgent support

---

## Useful Aliases (Optional)

Add to `~/.zshrc` or `~/.bashrc`:

```bash
# media_server shortcuts
alias ms-start='cd ~/dev/media_server && ./media-server &'
alias ms-stop='curl -X POST http://localhost:8080/api/shutdown'
alias ms-logs='tail -f ~/dev/media_server/server.log'
alias ms-build='cd ~/dev/media_server && go build -o media-server cmd/media-server/main.go'

# apfs-monitor shortcuts
alias apfs-status='launchctl list | grep apfs-monitor'
alias apfs-logs='tail -f ~/Library/Logs/apfs-monitor.log'
alias apfs-check='/usr/local/bin/apfs-monitor'
alias apfs-restart='launchctl unload ~/Library/LaunchAgents/com.local.apfs-monitor.plist && launchctl load ~/Library/LaunchAgents/com.local.apfs-monitor.plist'

# Development shortcuts
alias dev='cd ~/dev'
alias ms='cd ~/dev/media_server'
alias apfs='cd ~/dev/apfs-monitor'
```

---

## Additional Resources

### Project Documentation
- Each project has comprehensive PROJECT_OVERVIEW.md
- Check README.md for user-facing instructions
- Review git history for development rationale

### Development Methodology
- Read `🧠 DEVELOPMENT_METHODOLOGY.md` for AI-native workflow
- Review `PROJECT_INDEX.md` for detailed project catalog
- Study git commit messages for implementation details

### External Resources
- Go documentation: https://go.dev/doc/
- macOS launchd: `man launchd.plist`
- APFS utilities: `man diskutil`
- Extended attributes: `man xattr`

---

**Last Updated**: 2025-12-12
**Projects Covered**: media_server, apfs-monitor
**Command Categories**: 15+
**Total Commands**: 100+
