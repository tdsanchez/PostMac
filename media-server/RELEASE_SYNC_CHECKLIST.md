# Release Sync Checklist

> **Purpose**: Document differences between internal development repo and public PostMac/media-server release
> **Last Updated**: 2025-12-17
> **Status**: Internal fork maintained independently, sync for Green releases only

---

## Repository Strategy

**Internal Development**: `/Users/tdsanchez/dev/media-server-internal` (this directory)
- Continue development here with no module path changes
- Keep internal-only documentation and tooling
- Maintain as working development environment

**Public Release**: `/Users/tdsanchez/dev/PostMac/media-server`
- Sync code changes for each Green release
- Apply module path corrections during sync
- Exclude internal-only files

---

## Required Changes When Syncing Internal → PostMac

### 1. Module Path Correction

**File**: `go.mod`

```diff
- module github.com/tdsanchez/PostMac
+ module github.com/tdsanchez/PostMac/media-server
```

**Also update dependency organization**:
```go
// PostMac version (correct for nested repo structure)
require (
    github.com/fsnotify/fsnotify v1.9.0
    github.com/mattn/go-sqlite3 v1.14.32
    github.com/pkg/xattr v0.4.12
    github.com/rwcarlsen/goexif v0.0.0-20190401172101-9e8deecbddbd
    howett.net/plist v1.0.1
)

require golang.org/x/sys v0.13.0 // indirect
```

### 2. Import Path Updates (11 Files)

**Pattern**: Add `/media-server` to all internal imports

```diff
- "github.com/tdsanchez/PostMac/internal/handlers"
+ "github.com/tdsanchez/PostMac/media-server/internal/handlers"
```

**Files requiring import path updates**:
1. `cmd/media-server/main.go`
2. `internal/cache/cache.go`
3. `internal/conversion/converter.go`
4. `internal/handlers/api.go`
5. `internal/handlers/file.go`
6. `internal/handlers/pages.go`
7. `internal/metadata/metadata.go`
8. `internal/persistence/writer.go`
9. `internal/scanner/scanner.go`
10. `internal/state/state.go`
11. `internal/watcher/watcher.go`

**Automated fix command**:
```bash
# In PostMac/media-server directory after copying files
find . -type f -name "*.go" -exec sed -i '' 's|github.com/tdsanchez/PostMac/internal/|github.com/tdsanchez/PostMac/media-server/internal/|g' {} +
```

### 3. Dependency Cleanup

**File**: `go.sum`

Run after import path updates:
```bash
go mod tidy
```

This will clean up any stale dependency entries (like the extra golang.org/x/sys version in internal).

---

## Files to EXCLUDE from Public Release

**Internal-only documentation** (do not sync to PostMac):
- `SCRUBBING_REPORT_2025-12-16.md` - Pre-release scrubbing notes
- `SESSIONS.md` - Development session notes
- `TIDDLYWIKI_INTEGRATION_PLAN.md` - Future integration planning
- `TW_INTEGRATION_ANALYSIS_GEMINI.md` - Analysis documents
- `RELEASE_SYNC_CHECKLIST.md` - This file
- `media-server_` - Temporary/backup files

**Internal-only tooling** (do not sync to PostMac):
- `tag-sync.js` - Development utility
- `deploy-tag-sync-dash-fix.sh` - Deployment scripts

---

## Files Expected to be IDENTICAL

These should require no changes when syncing:
- `README.md` ✅
- `BUGS.md` ✅
- All `.html` templates
- All `.js` frontend files
- All Go code logic (only import paths change)
- Build scripts (`build_server.sh`, etc.)
- Deployment docs (`COLIMA_DEPLOYMENT.md`, etc.)

---

## Sync Procedure for Next Green Release

### Phase 1: Prepare Internal Repo
1. Ensure all features complete and tested
2. Update BUGS.md with latest status
3. Review git log for all changes since last release
4. Commit any pending changes

### Phase 2: Copy to PostMac
```bash
# From internal directory
rsync -av --exclude='SCRUBBING_REPORT*' \
         --exclude='SESSIONS.md' \
         --exclude='TIDDLYWIKI_*' \
         --exclude='TW_INTEGRATION_*' \
         --exclude='RELEASE_SYNC_CHECKLIST.md' \
         --exclude='tag-sync.js' \
         --exclude='deploy-tag-sync-dash-fix.sh' \
         --exclude='media-server_*' \
         --exclude='.git' \
         ./ ../PostMac/media-server/
```

### Phase 3: Fix Module Paths (in PostMac directory)
```bash
cd ../PostMac/media-server

# Update go.mod module path
sed -i '' 's|^module github.com/tdsanchez/PostMac$|module github.com/tdsanchez/PostMac/media-server|' go.mod

# Update all import paths in Go files
find . -type f -name "*.go" -exec sed -i '' 's|github.com/tdsanchez/PostMac/internal/|github.com/tdsanchez/PostMac/media-server/internal/|g' {} +

# Clean up dependencies
go mod tidy

# Verify it builds
go build -o media-server cmd/media-server/main.go
```

### Phase 4: Commit and Tag Release
```bash
# In PostMac/media-server directory
git add .
git commit -m "Sync from internal: [describe changes]

- Feature X
- Bug fix Y
- Performance improvement Z

🤖 Synced from internal development repo for Green release"

# Tag the release
git tag -a v0.X.0-green -m "Green Release X: [title]"
git push origin main --tags
```

### Phase 5: Verify
1. Build succeeds in PostMac directory
2. Run basic smoke tests
3. Verify documentation accuracy
4. Check GitHub rendering of README

---

## Notes

- **Why two repos?** Internal repo optimized for daily development with flat module path. Public repo follows GitHub nested structure convention.
- **Why not use branches?** This methodology uses trunk-based development. The two repos represent different deployment contexts, not feature branches.
- **Module path in internal**: Keeping `github.com/tdsanchez/PostMac` (without `/media-server`) in internal repo to avoid constant import path updates during development.
- **Frequency**: Sync only for Green releases (major milestones), not for every commit.

---

## Verification Commands

After sync, verify in PostMac directory:

```bash
# Check module path
grep "^module" go.mod
# Should show: module github.com/tdsanchez/PostMac/media-server

# Check import paths
grep -r "github.com/tdsanchez/PostMac/internal" --include="*.go" .
# Should show NO results (all should include /media-server)

# Verify imports are correct
grep -r "github.com/tdsanchez/PostMac/media-server/internal" --include="*.go" . | wc -l
# Should show ~30-40 import lines

# Build test
go build -o media-server cmd/media-server/main.go && echo "✅ Build successful"
```

---

## Change Log

**2025-12-17**: Initial checklist created after comparison of internal vs PostMac repos
- Documented module path differences
- Identified 11 files requiring import updates
- Listed internal-only files to exclude
- Created automated sync procedure
