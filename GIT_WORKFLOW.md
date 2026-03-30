# Git Workflow - Dev Tree with Submodules

**Purpose**: Guide for working with parent dev tree repo and subproject git repositories
**Structure**: Parent repo at `/Users/tdsanchez/dev` with git submodules for each project
**Last Updated**: 2025-12-12

---

## Repository Structure

```
/Users/tdsanchez/dev/              # Parent repo (this level)
├── .git/                          # Parent repo git database
├── .gitmodules                    # Submodule configuration
├── README.md                      # Dev tree documentation
├── 🧠 DEVELOPMENT_METHODOLOGY.md
├── PROJECT_INDEX.md
├── QUICK-REFERENCE.md
├── media_server/                  # Git submodule
│   └── .git/                      # Independent git repo
└── apfs-monitor/                  # Git submodule
    └── .git/                      # Independent git repo
```

### What This Means

**Parent Repo Tracks**:
- Dev tree level documentation (README.md, etc.)
- Which commit/version of each subproject is current
- Metadata about subprojects (.gitmodules)

**Submodule Repos Track**:
- Their own source code
- Their own project-specific documentation
- Their own independent git history

**Benefits**:
- Each project maintains independent version control
- Parent repo provides portfolio-level organization
- Can work on projects independently
- Clean separation of concerns

---

## Common Workflows

### Working on a Subproject (e.g., media_server)

```bash
# 1. Navigate to subproject
cd /Users/tdsanchez/dev/media_server

# 2. Work normally - it's a regular git repo
git status
git log
git add changed_files
git commit -m "Add new feature"

# 3. The parent repo notices the submodule changed
cd /Users/tdsanchez/dev
git status
# Output: modified:   media_server (new commits)

# 4. Update parent repo to track new subproject commit
git add media_server
git commit -m "Update media_server to include [feature]"
```

**Key Point**: Subprojects are independent repos. Work in them normally.

### Updating Dev Tree Documentation

```bash
# 1. Navigate to parent repo
cd /Users/tdsanchez/dev

# 2. Edit documentation
nano README.md
nano PROJECT_INDEX.md

# 3. Commit to parent repo
git add README.md PROJECT_INDEX.md
git commit -m "Update documentation for new project"
```

**Key Point**: Dev tree docs live in parent repo, project docs live in submodules.

### Adding a New Project

```bash
# 1. Create and initialize new project
cd /Users/tdsanchez/dev
mkdir new-project
cd new-project
git init
# ... create project files ...
git add .
git commit -m "Initial commit for new-project"

# 2. Add as submodule in parent repo
cd /Users/tdsanchez/dev
git submodule add ./new-project new-project
git add .gitmodules new-project
git commit -m "Add new-project as submodule"

# 3. Update dev tree documentation
nano PROJECT_INDEX.md
nano README.md
git add PROJECT_INDEX.md README.md
git commit -m "Document new-project in dev tree"
```

### Checking Status Across All Repos

```bash
# Parent repo status
cd /Users/tdsanchez/dev
git status

# Check each submodule status
git submodule foreach 'git status'

# Or check individually
cd media_server && git status
cd ../apfs-monitor && git status
```

---

## Git Submodule Commands

### Viewing Submodules

```bash
# List all submodules
git submodule

# Show submodule details
cat .gitmodules

# Check submodule status (shows current commit)
git submodule status
```

### Updating Submodules (After Others' Changes)

If someone else updated a subproject and you're pulling changes:

```bash
# Update parent repo
git pull

# Update all submodules to tracked commits
git submodule update --init --recursive

# Or update submodules to latest from their own repos
git submodule update --remote
```

### Working with Submodule Branches

```bash
# Enter submodule
cd media_server

# Create and work on a feature branch
git checkout -b new-feature
# ... make changes ...
git commit -m "Implement feature"

# Switch back to main development
git checkout main
git merge new-feature

# Parent repo notices the change
cd ..
git add media_server
git commit -m "Update media_server with new feature"
```

---

## Common Scenarios

### Scenario 1: Fix a Bug in media_server

```bash
# Work in submodule
cd media_server
git checkout -b fix-bug
# ... fix bug ...
git add .
git commit -m "Fix random mode bug"
git checkout main
git merge fix-bug
git branch -d fix-bug

# Update parent repo
cd ..
git add media_server
git commit -m "Update media_server: fix random mode bug"
```

### Scenario 2: Add New Dev Tree Documentation

```bash
cd /Users/tdsanchez/dev
nano NEW_GUIDE.md
git add NEW_GUIDE.md
git commit -m "Add guide for [topic]"
```

### Scenario 3: Document Changes Across Multiple Projects

```bash
# Make changes in subprojects first
cd media_server
# ... make changes, commit ...
cd ../apfs-monitor
# ... make changes, commit ...

# Update parent repo to track new commits
cd /Users/tdsanchez/dev
git add media_server apfs-monitor
git commit -m "Update both projects: [description]"

# Update dev tree docs to reflect changes
nano PROJECT_INDEX.md
git add PROJECT_INDEX.md
git commit -m "Update project index with recent changes"
```

### Scenario 4: Clone This Repo Somewhere Else

```bash
# Clone parent repo
git clone /path/to/source/dev /path/to/destination

# Initialize and update submodules
cd /path/to/destination
git submodule init
git submodule update

# Or in one command
git clone --recursive /path/to/source/dev /path/to/destination
```

---

## Important Concepts

### Submodules Are References, Not Copies

The parent repo doesn't store the subproject files - it stores:
1. The path to the submodule (media_server)
2. The commit hash the parent is tracking
3. The URL/path to the submodule's git repo

When you `git add media_server`, you're updating which commit the parent tracks.

### Each Repo Has Its Own History

```bash
# Parent repo history (dev tree level changes)
cd /Users/tdsanchez/dev
git log

# media_server history (project changes)
cd media_server
git log

# apfs-monitor history (project changes)
cd ../apfs-monitor
git log
```

These are **completely independent** histories.

### Submodule Commits Are Tracked by Parent

```bash
# Check what commit parent is tracking
cd /Users/tdsanchez/dev
git submodule status

# Output shows: commit hash, path, and branch
# +abc123def456 media_server (main)
# +789ghi012jkl apfs-monitor (master)
```

---

## Best Practices

### 1. Commit Subproject Changes First

Always commit changes in the subproject before updating the parent:

```bash
# ✅ Good workflow
cd media_server
git commit -m "Add feature"
cd ..
git add media_server
git commit -m "Update media_server with feature"

# ❌ Bad workflow
cd media_server
# ... make changes but don't commit ...
cd ..
git add media_server  # Nothing to update!
```

### 2. Keep Documentation in Sync

When you update a subproject significantly:
1. Commit changes in subproject
2. Update parent to track new commit
3. Update PROJECT_INDEX.md or README.md if needed
4. Commit documentation changes to parent

### 3. Use Descriptive Commit Messages

In subprojects:
```bash
git commit -m "Fix pagination bug in gallery view

- Issue: Gallery froze with 100k+ files
- Solution: Implement server-side pagination
- Impact: All large categories now usable"
```

In parent repo:
```bash
git commit -m "Update media_server to v1.5 with pagination

Tracks commit abc123 which adds gallery pagination for
handling 100k+ file categories."
```

### 4. Check Status Before Committing

```bash
# Always check what's changed
git status

# In parent repo, shows:
# - Changed documentation files (tracked by parent)
# - Modified submodules (new commits in subproject)

# Stage and commit appropriately
```

---

## Troubleshooting

### "modified: media_server (modified content)"

**Meaning**: The subproject has uncommitted changes.

**Solution**:
```bash
cd media_server
git status        # See what changed
git add .
git commit -m "Commit message"
cd ..
git add media_server  # Update parent to track new commit
git commit -m "Update media_server"
```

### "modified: media_server (new commits)"

**Meaning**: The subproject has new commits, parent repo not updated yet.

**Solution**:
```bash
git add media_server
git commit -m "Update media_server to latest version"
```

### Submodule Directory is Empty

**Meaning**: Cloned parent repo but submodules not initialized.

**Solution**:
```bash
git submodule init
git submodule update

# Or retroactively:
git submodule update --init --recursive
```

### Accidentally Deleted Submodule

**To remove a submodule properly**:
```bash
git submodule deinit media_server
git rm media_server
rm -rf .git/modules/media_server
git commit -m "Remove media_server submodule"
```

**To restore if accidentally deleted**:
```bash
git submodule update --init media_server
```

---

## Daily Workflow Examples

### Morning: Check What Changed

```bash
cd /Users/tdsanchez/dev

# Check parent repo
git status
git log --oneline -5

# Check subprojects
git submodule foreach 'git status'
git submodule foreach 'git log --oneline -3'
```

### Working Session: Fix Something in media_server

```bash
# Load context
cd media_server
cat PROJECT_OVERVIEW.md | head -50
git log --oneline -5

# Make changes
nano cmd/media-server/main.go

# Test
go build && ./media-server

# Commit in subproject
git add cmd/media-server/main.go
git commit -m "Fix cache invalidation bug"

# Update parent
cd ..
git add media_server
git commit -m "media_server: fix cache invalidation"
```

### End of Day: Review What Was Done

```bash
cd /Users/tdsanchez/dev

# See parent repo changes
git log --oneline --since="1 day ago"

# See subproject changes
cd media_server
git log --oneline --since="1 day ago"

cd ../apfs-monitor
git log --online --since="1 day ago"
```

---

## Advanced Topics

### Pushing Submodules to Remote (If You Add Remotes Later)

```bash
# Add remote to parent
cd /Users/tdsanchez/dev
git remote add origin git@github.com:user/dev-tree.git
git push -u origin main

# Add remotes to subprojects
cd media_server
git remote add origin git@github.com:user/media-server.git
git push -u origin main

cd ../apfs-monitor
git remote add origin git@github.com:user/apfs-monitor.git
git push -u origin master

# In .gitmodules, update URLs to point to real remotes
```

### Working with Submodule Branches

```bash
# Check which branch submodule is on
cd media_server
git branch

# Switch submodule to specific branch
git checkout feature-branch

# Parent tracks whatever commit is checked out
cd ..
git add media_server
git commit -m "Track media_server feature-branch"
```

### Diffing Across Repos

```bash
# See what changed in parent
cd /Users/tdsanchez/dev
git diff

# See what changed in specific submodule
git diff media_server
# Shows: Subproject commit abc...def

# See actual code changes in submodule
cd media_server
git diff
```

---

## Integration with Development Methodology

This git structure supports the AI-native development workflow:

### Session Start
```bash
# Load dev tree context
cd /Users/tdsanchez/dev
cat README.md PROJECT_INDEX.md

# Load specific project context
cd media_server
cat PROJECT_OVERVIEW.md
git log --oneline -10
```

### Note-Taking Mode
```bash
# Document findings in subproject
cd media_server
nano PROJECT_OVERVIEW.md
git add PROJECT_OVERVIEW.md
git commit -m "Document pagination performance analysis"

# Update parent
cd ..
git add media_server
git commit -m "media_server: add performance documentation"
```

### Code-Making Mode
```bash
# Implement in subproject
cd media_server
nano internal/handlers/pages.go
go build && ./test.sh
git add internal/handlers/pages.go
git commit -m "Implement pagination for large galleries"

# Update parent
cd ..
git add media_server
git commit -m "media_server: implement pagination (handles 168k+ files)"
```

### Session End
```bash
# Review all changes
cd /Users/tdsanchez/dev
git log --oneline --since="1 day ago"
git submodule foreach 'git log --oneline --since="1 day ago"'

# Ensure everything committed
git status
git submodule foreach 'git status'
```

---

## Quick Reference

### Common Commands

```bash
# Status
git submodule status              # Show submodule commits
git submodule foreach 'git status'  # Check each submodule

# Updates
git submodule update --init       # Initialize submodules
git submodule update --remote     # Update to latest

# Working
cd submodule && git commit        # Commit in submodule
cd .. && git add submodule        # Update parent tracking
git commit -m "Update submodule"  # Commit parent

# Adding new
git submodule add ./path name     # Add new submodule
```

---

## Why This Structure Works

**For This Use Case**:
- ✅ Independent development of each tool
- ✅ Portfolio-level organization and documentation
- ✅ Each project maintains own git history
- ✅ Parent tracks "current stable" versions
- ✅ Clean separation: dev tree docs vs project docs

**Supports AI-Native Workflow**:
- ✅ Context loading at appropriate level
- ✅ Token efficiency (load parent OR subproject docs, not both)
- ✅ Clear documentation boundaries
- ✅ Git history as design rationale at each level

---

**Last Updated**: 2025-12-12
**Parent Repo**: /Users/tdsanchez/dev
**Submodules**: media_server, apfs-monitor
**Methodology**: AI-native development with comprehensive system cards
