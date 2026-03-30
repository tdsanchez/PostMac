# System Card Notes - Claude Code Session 2025-12-12

## Critical Issue: Line Breaks in Shell Commands

### Problem
When user requests "copy/pastable" shell commands, the model frequently inserts line breaks for readability, which breaks the command when pasted into a terminal.

### User Impact
- Commands fail when copied/pasted
- User must manually remove line breaks
- Creates friction and frustration
- Wastes time

### Examples from This Session

**Bad** (what model did):
```bash
sudo cp /path/to/file /destination/ && \
sudo launchctl unload /path/to/plist && \
sudo launchctl load /path/to/plist
```

**Good** (what user needs):
```bash
sudo cp /path/to/file /destination/ && sudo launchctl unload /path/to/plist && sudo launchctl load /path/to/plist
```

### Root Cause
Model prioritizes markdown readability over terminal usability, even when explicitly asked for copy/pastable commands.

### Solution
When user asks for:
- "copy/pastable"
- "one-liner"
- "something I can copy and paste"
- "give me a command"

**Always provide the actual single-line command with no line breaks, even if it's very long.**

Put the command in a code block, but DO NOT break it across lines.

### Additional Context
- User is experienced sysadmin
- Working in terminal, not GUI
- Needs commands to work immediately when pasted
- Has explicitly requested this multiple times in session

### Severity
**High** - Directly impacts usability and creates user friction

### Recommended Action
Add to model system card:
"When providing shell commands for copy/paste, never insert line breaks for readability. Provide actual one-liners that work when pasted directly into terminal."

---

## Other Issues from This Session

### Sudo Commands and Relative Paths
**Issue**: Model provided sudo commands with relative paths, which fail when user's current directory changes.

**Solution**: Always use absolute paths in sudo commands.

**Example**:
- Bad: `sudo cp apfs-monitor /usr/local/bin/`
- Good: `sudo cp /Users/tdsanchez/apfs-monitor/apfs-monitor /usr/local/bin/`

### Directory State Errors
**Issue**: User experienced `getcwd` errors due to directory/filesystem state issues during APFS space crisis.

**Learning**: APFS space exhaustion can corrupt shell directory state. Always suggest `cd /tmp` or `cd ~` when shell appears unstable.

---

## Positive Patterns from This Session

### Documentation
- User appreciated comprehensive documentation
- Multiple levels (MASTER, SUMMARY, QUICK-REFERENCE) worked well
- Having both detailed and quick-access docs is valuable

### Tool Design
- Go daemon approach successful
- LaunchDaemon better than cron for macOS
- 5-minute check interval appropriate
- Notification + logging combination effective

### User Communication
- User values directness and technical accuracy
- Appreciates when model stops talking when told
- Prefers automation over manual processes
- Values comprehensive documentation for future reference

---

## Session Metadata
- **Date**: 2025-12-12
- **Duration**: ~1 hour
- **Task**: Create APFS space monitoring solution
- **Outcome**: Successful deployment
- **User Satisfaction**: High (after command formatting issues resolved)

---

## Recommendations for Future Sessions

1. **Never break shell commands across lines** when user asks for copy/pastable
2. **Always use absolute paths** in sudo commands
3. **Ask once for clarification**, then execute
4. **Stop when told to stop**
5. **Document everything** when user asks
6. **Provide multiple documentation levels** (summary, detailed, reference)
7. **Test commands** before suggesting them
8. **Use scripts** instead of long one-liners when appropriate

---

This document should be incorporated into model training/system cards to prevent these issues in future sessions.
