# Bug Tracking

> **Last Updated**: 2025-12-16
> **Purpose**: Track bugs discovered during testing and development
> **Status**: Active testing in progress, preparing for public release
> **Current Bug Count**: 6 open (3 High, 2 Medium, 1 Critical, 1 Low) + 3 fixed

---

## Active Bugs

### Template for New Bugs

```markdown
### Bug #N: [Short Description]

**Status**: 🔴 Open | 🟡 In Progress | 🟢 Fixed | ⚫ Won't Fix

**Severity**: Critical | High | Medium | Low

**Discovered**: YYYY-MM-DD

**Affects**: [Component/Feature]

**Symptoms**:
- What's happening that shouldn't be?
- What error messages appear?

**Steps to Reproduce**:
1. Step one
2. Step two
3. Expected vs Actual

**Expected Behavior**:
What should happen

**Actual Behavior**:
What actually happens

**Environment**:
- Browser: [Chrome/Safari/Firefox]
- File count: [if relevant]
- Category: [if relevant]

**Related Code**:
- `file.go:123` - relevant location
- `template.html:456` - related code

**Notes**:
Any additional context, workarounds, or investigation notes

**Resolution** (when fixed):
- Fix implemented: [commit hash]
- Changes made: [description]
```

---

## High Priority Bugs (Pre-Release Blockers)

### Bug #4: Tag editing UX issues - focus management and exit behavior

**Status**: ✅ **FIXED** (2025-12-16)

**Severity**: High (Pre-Release Blocker) → RESOLVED

**Priority**: ~~**HIGHEST** - Must fix before public release~~ → **FIXED**

**Discovered**: 2025-12-16 (documented from startup.txt)

**Fixed**: 2025-12-16 (commit e858307)

**Affects**: Tag editing in both gallery and single file views

**Symptoms**:
- Tag edit field does not reliably capture or maintain focus
- Escape key to close tag editing requires field to be in focus first
- If tag edit field not in focus, must click into field then press Escape (two-step process)
- Escape key behavior varies by browser
- Entering a matched tag and pressing Enter does not exit tag editing mode
- Tag editing implementation differs between gallery and single file views
- Overall UX is "very not usable" (user feedback)

**Steps to Reproduce**:
1. Press 'T' to open tag editing field
2. Click elsewhere on page (field loses focus)
3. Press Escape - nothing happens (field still visible)
4. Click back into tag editing field
5. Press Escape - now field closes
6. Alternative: Type tag name, press Enter on autocomplete match - field remains open

**Expected Behavior**:
- Tag edit mode should "dominate focus" - either field has focus or mode is deactivated
- Escape key should close tag editing regardless of focus state
- Arrow keys (in addition to Escape) should cancel tag edit mode
- When autocomplete match selected with Enter, tag editing mode should exit
- Consistent behavior across gallery and single file views
- Single-step exit mechanism

**Actual Behavior**:
- Focus management is inconsistent
- Escape requires field focus (two-step process to exit)
- Enter on matched tag adds tag but keeps field open
- Different implementations in gallery vs single file view
- Browser-dependent Escape key side effects

**Environment**:
- All browsers (behavior varies by browser)
- Both gallery and single file views
- Tag autocomplete interaction

**Related Code**:
- `cmd/media-server/main_template.html` - Single file view tag editing
- `cmd/media-server/main_template.js` - Tag editing event handlers (lines 318-376)
- `cmd/media-server/gallery_template.html` - Gallery view tag editing
- Tag editing implementation differs between views (needs unification)

**Proposed Fixes** (from startup.txt):
1. **Arrow keys cancel tag edit mode** - Easiest win, quick implementation
2. **Focus dominance** - If tag edit mode active but field not focused, deactivate mode automatically
3. **Enter exits after tag match** - When autocomplete match selected, add tag and close field
4. **Unified implementation** - Extract tag editing component, use same code in both views
5. **Browser-agnostic exit** - Handle Escape consistently across browsers

**Implementation Notes**:
- Quick win: Arrow key cancellation can be implemented first
- Medium effort: Focus dominance requires event listeners on document
- Larger refactor: Unifying tag editing across views
- Related to gallery/single view tag drift issue in PROJECT_OVERVIEW.md

**User Impact**:
- Daily workflow friction - tag editing is core feature
- Muscle memory interrupted by focus management issues
- Reduces efficiency of rapid tagging workflow

**Priority Justification**:
User explicitly stated this is higher priority than other bugs. Tag editing is a primary workflow, and poor UX in core features damages project credibility.

**Resolution**:
- **Fix implemented**: commit e858307 (2025-12-16)
- **Changes made**: Created `exitTagMode()` helper function in `cmd/media-server/main_template.js`
  - Clears input field, autocomplete, and resets selectedIndex
  - Removes 'active' class from tag input container
  - Called from: Enter handler (auto-closes after adding tag), Escape handler, Arrow key handlers (closes before navigating)
- **Result**: Tag editor now closes automatically after adding tag with Enter. Arrow keys close editor and navigate in one action. No clicking required to dismiss editor.
- **Testing**: Confirmed working - smooth keyboard-driven tag editing workflow restored
- **See**: TAG_EDIT_CHANGE_GEMINI.md for implementation design

---

### Bug #5: File rescanning performance - blocks server during scan

**Status**: ✅ **FIXED** (2025-12-16)

**Severity**: High (Pre-Release Blocker) → RESOLVED

**Priority**: ~~**HIGH** - Must fix before public release~~ → **FIXED**

**Discovered**: 2025-12-16 (documented from startup.txt)

**Fixed**: 2025-12-16 (commit 7c651c5)

**Affects**: Rescan button, FSEvents auto-rescan, overall server responsiveness

**Symptoms**:
- Auto-rescanning of files makes service unusable for large file sets
- Server becomes unresponsive during scan operations
- Manual rescan via button blocks user interaction
- Intended transparent UX is not achieved
- "Not what we set out to do" - architectural issue vs intended design

**Steps to Reproduce**:
1. Have large file collection (100k+ files)
2. Click rescan button OR trigger FSEvents auto-rescan
3. Observe server becomes unresponsive
4. Try to navigate during scan - requests hang or timeout
5. UX degrades significantly during scan

**Expected Behavior**:
- Scanning should be transparent to UX
- Server should remain responsive during scan operations
- User can continue browsing while scan runs in background
- Progress indication without blocking interaction

**Actual Behavior**:
- Scan blocks server threads
- Requests queue up or timeout during scan
- Navigation and browsing unavailable during scan
- Auto-rescan (FSEvents) causes unexpected UX freezes

**Environment**:
- All browsers
- Particularly bad with large file sets (100k+)
- Both manual rescan button and FSEvents auto-rescan affected

**Related Code**:
- `internal/scanner/scanner.go` - ScanDirectory function (blocking filesystem walk)
- `internal/watcher/watcher.go` - FSEvents auto-rescan trigger
- `internal/handlers/api.go` - /api/rescan endpoint (lines 659-686)
- `cmd/media-server/main.go` - Scan on startup

**Root Cause Analysis**:
- Scanning uses blocking `filepath.Walk` on main goroutine
- Even though scan runs in goroutine, state locks block request handlers
- State lock contention: scan holds write lock, handlers wait for read locks
- Large file sets mean long lock hold times
- Background scan still impacts foreground UX

**Proposed Solutions**:
1. **Chunked scanning** - Release state lock periodically during scan
2. **Copy-on-write state** - Build new state, atomic swap when complete
3. **Separate read/write state** - Handlers read from stable state, scan builds new state
4. **Incremental updates** - Update state in smaller batches
5. **Priority queuing** - User requests take priority over scan updates

**Implementation Complexity**:
- Quick fix: Add lock release points in scan loop
- Better fix: Copy-on-write state management
- Best fix: Separate read/write state with atomic swap

**User Impact**:
- Daily workflow disruption - rescans happen frequently
- Auto-rescan (FSEvents) causes unexpected freezes
- Users learn to avoid rescan button = defeats the purpose

**Priority Justification**:
User explicitly stated this is high priority pre-release blocker. Auto-rescan feature becomes a liability rather than asset if it degrades UX.

**Solution Implemented** ✅:
- **Approach**: Double-buffered state with atomic.Value for lock-free reads (#3 from proposed solutions)
- **Architecture**: Two complete AppState instances (stateA, stateB), atomic swap pattern
- **Implementation**:
  - `internal/state/state.go` - Added AppState struct, double-buffer, GetCurrent() lock-free API
  - `internal/scanner/scanner.go` - Builds into inactive buffer, no locks during scan
  - `internal/handlers/*.go` - All handlers converted to lock-free GetCurrent()
- **Testing**: 10 parallel requests during 30s scan (169,902 files)
- **Results**:
  - Before: 82,000ms request latency (blocked)
  - After: 13-50ms request latency (lock-free)
  - **1,600x performance improvement**
- **Documentation**: `feature_change_docs/TWO_STATE_BUFFERING.md`
- **Commit**: 7c651c5
- **Status**: ✅ RESOLVED - Server remains fully responsive during rescans

---

## Medium Priority Bugs (Post-Testing Discovery)

### Bug #6: 'C' key sometimes triggers link copy instead of comment editing

**Status**: 🔴 Open

**Severity**: Medium

**Discovered**: 2025-12-16 (documented from startup.txt)

**Affects**: Single file viewer - 'C' keyboard shortcut

**Symptoms**:
- In some cases when pressing 'C' in single file view, creates a copy of the link instead of enabling comment editing
- This is a relic of an old feature that needs to be removed
- Inconsistent behavior - sometimes works correctly, sometimes triggers copy

**Steps to Reproduce**:
- Intermittent - specific trigger conditions unclear
- Press 'C' in single file view
- Sometimes: Comment editing activates (correct)
- Sometimes: Link copy happens (incorrect)

**Expected Behavior**:
- 'C' key should ONLY trigger comment editing mode
- No link copying functionality should exist

**Actual Behavior**:
- Old link copy feature still triggers in some cases
- Conflicts with comment editing

**Related Code**:
- `cmd/media-server/main_template.js` - Keyboard event handlers
- Comment editing handler around line 439-546
- May have old event listener or key binding still active

**Fix Required**:
- Remove all link copy functionality
- Ensure 'C' key only triggers comment editing
- Clean up any legacy code related to link copying

**Priority**: Medium - doesn't block core workflows but causes confusion

---

### Bug #7: Arrow key navigation doesn't work on server start until category selected

**Status**: 🔴 Open

**Severity**: Medium

**Discovered**: 2025-12-16 (documented from startup.txt)

**Affects**: Initial page load, keyboard navigation

**Symptoms**:
- When server starts, it selects first DOM object representing a file on page
- Arrow key navigation doesn't work until user manually navigates into a category
- After diving into a category, arrow keys work correctly

**Steps to Reproduce**:
1. Start server
2. Load homepage (category view)
3. First file/category card is visually selected
4. Press arrow keys (↑↓←→)
5. Nothing happens - no navigation
6. Click into a category to view gallery
7. Arrow keys now work correctly

**Expected Behavior**:
- Arrow key navigation should work immediately on homepage
- Selected card should respond to keyboard navigation
- Consistent behavior from initial load

**Actual Behavior**:
- Visual selection appears (first item selected)
- Keyboard navigation doesn't activate until after category interaction
- Two-state behavior: broken initially, works after interaction

**Related Code**:
- `cmd/media-server/index_template.html` - Homepage category cards, keyboard handler
- `cmd/media-server/gallery_template.html` - Gallery view keyboard handler
- Auto-select implementation (lines 336-339 in gallery)
- Homepage keyboard navigation (arrow key handling)

**Root Cause (Hypothesis)**:
- Event listeners not properly attached on initial page load
- Focus state not properly initialized
- JavaScript initialization timing issue

**Priority**: Medium - workaround exists (click into category first)

---

### Bug #8: Random mode finds non-existent files after deletions (auto-scan not updating)

**Status**: 🔴 Open

**Severity**: High

**Discovered**: 2025-12-16 (documented from startup.txt)

**Affects**: Random mode navigation, file deletion interaction with auto-scan

**Symptoms**:
- "Extremely frequently" when in random mode, next file does not exist
- Occurs after deleting files externally or via 'X' key
- Auto-scan (FSEvents) not correctly updating state when files are deleted
- Random mode tries to navigate to cached paths that no longer exist

**Steps to Reproduce**:
1. Enable random mode in single file viewer
2. Delete some files (either via 'X' key or externally via Finder)
3. Navigate randomly using '→' arrow
4. Frequently encounter "file not found" or 404 errors
5. File paths in random pool not updated after deletions

**Expected Behavior**:
- When file deleted, auto-scan detects change
- Deleted file removed from all category caches
- Random mode pool updated immediately
- Never tries to navigate to deleted files

**Actual Behavior**:
- Deleted files remain in random pool
- Navigation attempts fail with 404 or errors
- FSEvents triggers rescan but cache not properly updated
- Or: cache updated but localStorage/client-side cache stale

**Related Code**:
- `internal/watcher/watcher.go` - FSEvents auto-rescan
- `internal/cache/cache.go` - Cache update on file deletion
- `internal/handlers/api.go` - DeleteFile endpoint (lines 315-366)
- `cmd/media-server/main_template.js` - Random navigation (lines 87-129)
- localStorage file list cache

**Root Cause Analysis (Hypothesis)**:
1. FSEvents triggers rescan but timing issue with cache update
2. Client-side localStorage cache not invalidated when rescan completes
3. DeleteFile API updates cache but doesn't broadcast to other clients/tabs
4. Race condition between deletion and cache update

**Potential Fixes**:
1. Ensure cache invalidation on successful delete
2. Broadcast cache invalidation event to client (auto-refresh localStorage)
3. Improve FSEvents → cache → client update chain
4. Add validation: skip non-existent files in random mode gracefully

**User Impact**:
- "Extremely frequent" issue (user's words)
- Breaks random mode workflow
- Requires manual rescan to fix
- Frustrating UX - broken navigation

**Priority**: High - core feature (random mode) frequently broken

**Related to**: Bug #5 (rescanning performance) - both involve scan/cache coherence

---

### Bug #1: Sort mode only sorts current page, not entire category

**Status**: 🔴 Open

**Severity**: High

**Discovered**: 2025-12-13

**Affects**: Gallery view - Sort functionality (S key)

**Symptoms**:
- Pressing 'S' to cycle through sort modes only reorders files visible on current page
- Sort does not affect the full category's file set
- Pagination shows different files on each page, but they're not globally sorted

**Steps to Reproduce**:
1. Navigate to a category with more files than page size (e.g., "All" with 168k files, 200 per page)
2. Press 'S' to toggle sort mode (name → date → size → random)
3. Observe that only the 200 files on current page are reordered
4. Navigate to page 2 - files are not sorted relative to page 1

**Expected Behavior**:
- Sort should operate on the entire category's file set
- Backend should re-sort the full list and return page 1 of sorted results
- Pagination should show sorted slices (files 1-200, 201-400, etc. of sorted set)

**Actual Behavior**:
- Sort only reorders files in current DOM/page
- Backend pagination returns original order
- Global sort order not preserved across pages

**Environment**:
- Any browser
- Affects all categories with pagination (200+ files)
- Sort modes: name, date, size, random

**Related Code**:
- `cmd/media-server/gallery_template.html` - Client-side sort logic (S key handler)
- `internal/handlers/pages.go:277-358` - Backend pagination in HandleTag
- Gallery JavaScript handles sort locally without server round-trip

**Notes**:
- This is a pagination implementation issue - sort was designed before pagination
- Fix requires either:
  1. Server-side sort with query param (`?sort=name&order=asc`)
  2. Client-side fetch of full file list (defeats pagination benefits)
  3. Hybrid: cache full list client-side, sort locally, paginate in DOM
- Related to Bug #2 (random mode scope issue)

---

### Bug #2: Random mode in single file view only randomizes within current page files

**Status**: ✅ **FIXED** (2025-12-16)

**Severity**: High → RESOLVED

**Discovered**: 2025-12-13

**Fixed**: 2025-12-16 (commit bd81103)

**Affects**: Single file viewer - Random mode navigation

**Symptoms**:
- When navigating from gallery to single file view, toggling random mode (R key)
- Random navigation only picks from files visible on the gallery page (e.g., 200 files)
- Does not randomize across entire category (e.g., 168k files)

**Steps to Reproduce**:
1. Navigate to large category (e.g., "All" with 168k files)
2. Gallery shows page 1 (files 1-200 of 168,331)
3. Click on a file to enter single file view
4. Press 'R' to enable random mode
5. Press '→' (right arrow) to navigate randomly
6. Observe that random files are only from the 200 files on page 1

**Expected Behavior**:
- Random mode should pick from all 168,331 files in "All" category
- Each random navigation could land on any file in the full set
- Should use `/api/filelist` endpoint to get complete category file list

**Actual Behavior**:
- Random navigation constrained to files from current gallery page
- `allFilePaths` array only contains ~200 paths (current page)
- Effectively "random within page" instead of "random within category"

**Environment**:
- Any browser
- Affects categories with pagination (200+ files)
- Single file viewer with random mode enabled

**Related Code**:
- `cmd/media-server/main_template.js:87-129` - Random navigation logic
- `internal/handlers/pages.go:448-452` - Server sends empty allFilePaths array
- `internal/handlers/api.go:193-228` - `/api/filelist` endpoint exists but may not be used correctly
- Gallery template populates localStorage cache with page files only

**Notes**:
- This is related to Bug #1 - both are pagination scope issues
- The `/api/filelist` endpoint was added to solve serialization bottleneck (see NEXT_CYCLE_IMPROVEMENTS.md)
- Fix likely requires:
  1. Ensure single file viewer fetches full category file list from `/api/filelist`
  2. Verify localStorage cache stores complete list, not just current page
  3. May need to update gallery's localStorage caching logic
- According to NEXT_CYCLE_IMPROVEMENTS.md, this should already work - needs investigation

**Investigation Needed**:
- Check if gallery view's localStorage cache is storing full list or just current page
- Verify `/api/filelist` endpoint is being called when entering random mode
- Review commit 21e1ef5 (localStorage caching implementation) for potential issues

**Resolution**:
- **Fix implemented**: commit bd81103 (2025-12-16)
- **Root cause**: Gallery template cached only current page's 200 files instead of fetching full category from `/api/filelist`
- **Changes made**: Modified `cmd/media-server/gallery_template.html` to:
  - Make `initializeItems()` async
  - Fetch complete file list from `/api/filelist?category=X` on page load
  - Cache all 169k+ paths in localStorage (not just current page)
  - Added fallback to current page if API fetch fails
- **Result**: Random mode now picks from entire category (169k files), not just 200
- **Testing**: Confirmed working after clearing stale localStorage cache
- **Note**: Stale cache from before fix needed manual clearing - future enhancement could add cache versioning
- **See**: PAGE_NAV_FIX_CLAUDE.md and PAGE_NAV_ANALYSIS_GEMINI.md for complete analysis

---

### Bug #3: File deletion (X key) shows false failure with delay, but actually succeeds

**Status**: 🔴 Open

**Severity**: Critical

**Discovered**: 2025-12-13

**Affects**: Single file viewer - File deletion modal dialog (X key)

**Symptoms**:
- Pressing 'X' key triggers delete confirmation modal with two CSS buttons
- OS default key "ring" (focus indicator) appears inconsistently - sometimes visible, sometimes not
- When using standard interaction (click or space bar):
  - Long delay occurs (several seconds)
  - GUI reports deletion failure
  - **BUT the file is actually deleted successfully** (false negative error)
- Using third-party accessibility tool (Shortcat) to invoke button works instantly with no delay

**Steps to Reproduce**:
1. Navigate to any file in single file viewer
2. Press 'X' key to delete file
3. Modal dialog appears with "Delete" and "Cancel" buttons
4. Observe whether OS focus ring appears on default button (inconsistent)
5. Click "Delete" button OR press Space to activate default
6. Long delay (several seconds)
7. GUI shows error/failure message
8. Check filesystem - file is actually deleted despite error message

**Expected Behavior**:
- Modal appears with clear default button focus
- Clicking/pressing Space activates delete immediately
- Backend returns success status
- GUI shows success message
- File is deleted and cache updated
- Navigation proceeds to next file

**Actual Behavior**:
- Modal focus state inconsistent (ring appears/doesn't appear randomly)
- Long delay (2-5 seconds) after clicking/pressing Space
- GUI reports "Deletion failed" or shows error
- File IS actually deleted (backend operation succeeded)
- Cache may or may not update correctly
- False negative error breaks user trust in UI

**Workaround**:
- Using Shortcat (third-party accessibility tool) to invoke button works perfectly:
  - No delay
  - Instant response
  - No false errors
  - 100% reliable

**Environment**:
- Browser: Safari/Chrome (test across browsers)
- macOS version: [needs testing]
- File types: All types affected
- Reproducibility: Intermittent but frequent

**Related Code**:
- `cmd/media-server/main_template.html` - Modal dialog rendering
- `cmd/media-server/main_template.js` - X key handler, delete API call
- `internal/handlers/api.go` - `/api/deletefile` endpoint
- Modal CSS and focus management

**Root Cause Analysis** (hypotheses):

1. **Race condition - frontend timeout vs backend success**:
   - Frontend sets aggressive timeout (e.g., 2 seconds)
   - Backend delete operation takes 3-5 seconds (macOS Trash move is slow)
   - Frontend times out and shows error before backend responds
   - Backend completes successfully but response is ignored

2. **Modal focus management issues**:
   - Browser focus state confused when modal appears
   - Event handlers attached before DOM fully ready
   - Focus ring visibility tied to focus state (explains inconsistency)
   - Click/keypress events delayed or queued

3. **Async/await promise handling bug**:
   - Delete API call not properly awaited
   - Error handling catches timeout exception but file already deleted
   - UI state updated pessimistically before backend confirms

4. **Cache update timing**:
   - Delete succeeds on filesystem
   - Cache update happens asynchronously
   - UI checks cache before update completes
   - Reports "file still exists" when cache is stale

5. **macOS Trash operation latency**:
   - Moving file to Trash via osascript is slow (3-5 seconds)
   - Backend doesn't stream progress
   - Frontend assumes failure when no immediate response

**Why Shortcat works**:
- Bypasses browser event system entirely
- Uses macOS accessibility APIs directly
- May trigger click at OS level before JavaScript handlers
- Avoids modal focus state issues
- Possible that it fires event BEFORE frontend timeout logic kicks in

**Investigation Needed**:
1. Measure actual backend `/api/deletefile` response time
2. Check frontend timeout settings in fetch/API call
3. Review promise/async handling in delete function
4. Add logging to track: button click → API call → backend start → trash operation → backend response → frontend receives
5. Test across browsers (Safari, Chrome, Firefox) to isolate browser-specific issues
6. Examine modal rendering and focus management code
7. Check if macOS version affects Trash operation speed

**Potential Fixes**:
1. **Increase frontend timeout**: Allow 10-15 seconds for Trash operation
2. **Add loading indicator**: Show progress during delete operation
3. **Fix error handling**: Check actual backend response, don't assume timeout = failure
4. **Improve modal focus**: Ensure default button gets focus reliably
5. **Add retry logic**: If timeout occurs, poll backend to check if operation succeeded
6. **Stream progress**: Backend sends progress updates during Trash operation
7. **Cache invalidation**: Force cache refresh before checking deletion status

**Notes**:
- This is a **critical UX issue** - false error messages destroy user trust
- Intermittent focus ring suggests browser focus state bug
- Shortcat workaround proves the backend works - this is purely a frontend timing/error handling issue
- The fact that delete succeeds but reports failure is worse than if it simply failed
- User avoids this feature entirely due to unreliability - major workflow impact

**Priority**: Critical - False errors are worse than no errors, breaks user confidence in the system

---

### Bug #9: Enter key doesn't navigate into categories from root index view

**Status**: 🔴 Open

**Severity**: Low

**Discovered**: 2025-12-16

**Affects**: Index/root view (http://localhost:XXXX/)

**Symptoms**:
- At media server root/index page (category listing), pressing Enter key does nothing
- Enter key navigation works correctly in gallery views (categories) and single file viewer
- Inconsistent keyboard navigation behavior between root and other views

**Steps to Reproduce**:
1. Navigate to server root: http://localhost:9191/
2. Categories are displayed (All, Photos, Videos, etc.)
3. Use arrow keys to navigate between categories (works)
4. Press Enter to navigate into selected category
5. **Nothing happens** - Enter key is non-functional

**Expected Behavior**:
- Arrow keys navigate between categories in root view
- Enter key activates selected category (navigates to /tag/CategoryName)
- Consistent keyboard-only navigation across all views

**Actual Behavior**:
- Arrow keys work in root view
- Enter key does nothing in root view
- Must click category to navigate into it
- Enter key DOES work in gallery and single file views

**Workaround**:
- Click on category instead of using Enter key
- Or navigate directly via URL

**Environment**:
- All browsers
- Root index view only (not galleries or file viewer)

**Related Code**:
- `cmd/media-server/index_template.html` - Root view category listing
- Likely missing Enter key handler in root view JavaScript
- Compare with gallery/viewer Enter key handlers which work correctly

**Notes**:
- Low priority - clicking works fine
- Affects keyboard-only navigation users
- Inconsistency in UX across views
- Quick fix: Add Enter key handler to root view matching other views

**Priority**: Low - Workaround exists (clicking), but keyboard nav should be consistent

---

## Recently Fixed Bugs

_Bugs will be moved here once resolved, for reference_

---

## Won't Fix / By Design

_Behaviors that seem like bugs but are intentional design decisions_

---

