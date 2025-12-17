# Two-State Buffering Architecture

> **Date**: 2025-12-16
> **Purpose**: Eliminate request blocking during filesystem rescans
> **Status**: Implementation in progress
> **Bug**: Fixes Bug #5 (File rescanning performance)

---

## Problem Statement

### Current Blocking Behavior

**User requirement**: "If the server is rescanning files I should still be able to navigate and use it. Modal blocking is not acceptable."

**Current implementation**:
- Scanner holds write lock for entire scan duration (10+ seconds with 100k files)
- ALL request handlers block waiting for read lock
- Server becomes completely unresponsive during scans
- Auto-rescan (FSEvents) triggers frequently, causing unexpected freezes

**Code location**: `internal/scanner/scanner.go:19-21`
```go
func ScanDirectory(serveDir string) error {
    state.LockData()           // ← WRITE LOCK (blocking all requests)
    defer state.UnlockData()   // ← Held for 10+ seconds during filepath.Walk()

    // ... 178 lines of filesystem scanning ...
}
```

**Impact**:
- Handler requests queue up during scan
- Navigation stops working
- User experience degrades to unusable
- FSEvents auto-rescan becomes a liability instead of feature

---

## Solution: Double-Buffered State

### Concept

Maintain TWO complete application states in memory. While one serves requests (active), the other can be rebuilt (inactive). When scan completes, atomically swap which state is active.

**Key insight**: Go's `sync/atomic.Value` provides lock-free atomic loads, perfect for this pattern.

### Architecture

```
┌─────────────────────────────────────────────────────┐
│                  HTTP Handlers                      │
│  (HandleTag, HandleViewer, HandleRoot, etc.)        │
└─────────────────────┬───────────────────────────────┘
                      │
                      │ current := currentState.Load()  ← LOCK-FREE!
                      ↓
         ┌────────────────────────────┐
         │   atomic.Value             │
         │   (currentState)           │
         │         ↓                  │
         │    Points to:              │
         │    ┌─────────┐             │
         │    │ stateA  │ ← Active    │
         │    └─────────┘             │
         │         or                 │
         │    ┌─────────┐             │
         │    │ stateB  │ ← Active    │
         │    └─────────┘             │
         └────────────────────────────┘
                      │
         ┌────────────┴────────────┐
         ↓                         ↓
    ┌─────────┐              ┌─────────┐
    │ stateA  │              │ stateB  │
    └─────────┘              └─────────┘
         ↑                         ↑
         │                         │
    One serves requests      Other being rebuilt
    (read-only)              (no locks needed!)
         │                         │
         └─────────┬───────────────┘
                   │
            Atomic swap when
            rebuild completes
```

### State Structure

**Before** (current):
```go
// Global variables with single mutex
var (
    filesByTag      map[string][]models.FileInfo
    allFiles        []models.FileInfo
    allTags         []string
    dataMutex       sync.RWMutex  // ← THE PROBLEM
)
```

**After** (double-buffered):
```go
// Encapsulated state struct
type AppState struct {
    filesByTag map[string][]models.FileInfo
    allFiles   []models.FileInfo
    allTags    []string
}

// Double buffer
var (
    stateA       *AppState        // Buffer A
    stateB       *AppState        // Buffer B
    currentState atomic.Value     // Points to active state (lock-free reads!)
    stateMutex   sync.Mutex       // Only for swap operation
)
```

---

## Implementation Plan

### Phase 1: Refactor State Package

**File**: `internal/state/state.go`

**Changes**:
1. Define `AppState` struct
2. Replace global variables with double-buffer
3. Add `GetCurrent() *AppState` - lock-free reader
4. Add `SwapState(newState *AppState)` - atomic swap
5. Add `GetInactiveState() *AppState` - for building new state
6. Keep existing helper functions for compatibility

**New API**:
```go
// Lock-free read access
func GetCurrent() *AppState

// Build new state (returns inactive buffer)
func GetInactiveState() *AppState

// Atomic swap after build complete
func SwapState(newState *AppState)
```

### Phase 2: Refactor Scanner

**File**: `internal/scanner/scanner.go`

**Changes**:
1. Remove `state.LockData()` / `UnlockData()` calls
2. Accept `*state.AppState` parameter
3. Build into provided state instead of global
4. Create new `ScanDirectoryDoubleBuffered()` function

**New signature**:
```go
// Old (blocks global state)
func ScanDirectory(serveDir string) error

// New (builds into provided state, no blocking)
func ScanDirectoryInto(serveDir string, targetState *state.AppState) error

// High-level wrapper (handles double-buffering)
func ScanDirectoryDoubleBuffered(serveDir string) error {
    inactive := state.GetInactiveState()
    if err := ScanDirectoryInto(serveDir, inactive); err != nil {
        return err
    }
    state.SwapState(inactive)
    return nil
}
```

### Phase 3: Update Handlers

**Files**: `internal/handlers/*.go`

**Changes** (mechanical, low risk):
```go
// Before
state.RLockData()
filesByTag := state.GetFilesByTag()
state.RUnlockData()

// After
current := state.GetCurrent()  // Lock-free!
filesByTag := current.FilesByTag
```

**Affected handlers**:
- `HandleRoot`
- `HandleTag`
- `HandleViewer`
- `HandleGetFileList`
- Any other handler reading state

### Phase 4: Update Watcher

**File**: `internal/watcher/watcher.go`

**Changes**:
- Line 168: Replace `scanner.ScanDirectory()` with `scanner.ScanDirectoryDoubleBuffered()`

### Phase 5: Update Main

**File**: `cmd/media-server/main.go`

**Changes**:
- Initialize double-buffer on startup
- Use new scan function

---

## Benefits

### Performance

| Metric | Before | After |
|--------|--------|-------|
| **Request blocking during scan** | 10+ seconds | 0 seconds |
| **Lock contention** | High (RWMutex) | Zero (atomic.Value) |
| **Handler latency** | Variable (blocks on lock) | Constant (lock-free) |
| **Scan time** | 10s (held lock) | 10s (background, invisible) |

### User Experience

**Before**:
```
User: *clicks next file during auto-rescan*
Server: "..." (10 second freeze)
User: 😡 "Is it broken?"
```

**After**:
```
User: *clicks next file during auto-rescan*
Server: *serves instantly from active state*
Background: *quietly building new state*
[10 seconds later]: *atomic swap, user never noticed*
User: 😊 "It just works"
```

### Architectural

✅ **Lock-free reads**: Handlers never block
✅ **Separation of concerns**: Scan builds separate state
✅ **Atomic consistency**: Swap is all-or-nothing
✅ **No partial states**: Users never see incomplete scan
✅ **Predictable latency**: No variable lock wait times

---

## Trade-offs

### Memory Cost

**2x state in RAM**:
- 100k files ≈ 100MB per state
- Total: ~200MB for double-buffer
- Acceptable for modern systems (8GB+ RAM typical)

**Alternative considered**: "Just generate another SQLite DB"
- Pros: Lower memory (DB on disk)
- Cons: Slower (DB write + read), still need locking for DB access
- Verdict: Double-buffer is better for current scale (100k-500k files)

### Code Changes

**Medium refactor** (~200 lines changed):
- State package: ~100 lines
- Scanner: ~50 lines
- Handlers: ~50 lines (mechanical)
- Watcher: ~10 lines
- Main: ~10 lines

**Risk**: Low - changes are mechanical, not algorithmic

---

## Testing Plan

### Unit Tests

1. **State package**:
   - Verify `GetCurrent()` returns active state
   - Verify `SwapState()` atomically swaps
   - Verify concurrent `GetCurrent()` calls don't race

2. **Scanner**:
   - Verify `ScanDirectoryInto()` doesn't mutate global state
   - Verify scan results identical to current implementation

### Integration Tests

1. **Concurrent access during scan**:
   ```bash
   # Start scan in background
   curl http://localhost:8080/api/rescan &

   # Hammer server with requests during scan
   for i in {1..100}; do
     curl http://localhost:8080/tag/All &
   done
   wait

   # Verify: All requests succeed, no blocking
   ```

2. **State consistency**:
   - Verify handlers see consistent state (no torn reads)
   - Verify swap doesn't cause race conditions

3. **Memory leak check**:
   - Run multiple rescans
   - Verify old states are garbage collected
   - Monitor memory usage (should be stable)

### Performance Testing

**Load test with concurrent scan**:
```python
# Start 5 instances
# Trigger rescan on instance 1
# Measure request latency on instances 2-5 during scan
# Expected: Latency unchanged (no blocking)
```

---

## Rollback Plan

If implementation has issues:

1. **Git revert**: Single commit contains all changes
2. **Feature flag**: Could add `useDoubleBuffer` flag to toggle
3. **Fallback**: Old code preserved in git history

**Low risk**: Changes are isolated, can be reverted cleanly.

---

## Migration Notes

### Backward Compatibility

**Cache format**: Unchanged (SQLite schema identical)
**API**: Unchanged (same HTTP endpoints)
**Behavior**: Identical (except no blocking during scan)

### Deployment

**Zero downtime**: Binary replacement, no migration needed
**No config changes**: Works with existing setup
**Transparent**: Users won't notice except improved performance

---

## Related Issues

### Fixes

- **Bug #5**: File rescanning performance (HIGH priority pre-release blocker)
- **Bug #8**: Random mode finds deleted files (related - cache coherence improves)

### Enables

- **Containerization**: Makes server truly ready for production deployment
- **Auto-rescan**: FSEvents becomes useful instead of liability
- **Scalability**: Lock-free architecture scales better

### Future Enhancements

Once double-buffering is in place:
- Incremental scanning (update only changed files)
- Progressive scan (show results as they arrive)
- Concurrent scans (if multiple volumes)

---

## Code Locations

### Files to Modify

1. **`internal/state/state.go`** - Core refactor (~100 lines)
2. **`internal/scanner/scanner.go`** - Remove locks, accept state param (~50 lines)
3. **`internal/handlers/api.go`** - Update state access (~20 lines)
4. **`internal/handlers/pages.go`** - Update state access (~30 lines)
5. **`internal/watcher/watcher.go`** - Use new scan function (~10 lines)
6. **`cmd/media-server/main.go`** - Initialize double-buffer (~10 lines)

### Key Functions to Change

**State package**:
- `GetFilesByTag()` → `GetCurrent().FilesByTag`
- `GetAllFiles()` → `GetCurrent().AllFiles`
- `LockData() / UnlockData()` → Remove

**Scanner**:
- `ScanDirectory(dir)` → `ScanDirectoryInto(dir, state)`

**Handlers** (all):
- `state.RLockData()` → `current := state.GetCurrent()`

---

## Implementation Timeline

**Estimated effort**: 2-4 hours

**Breakdown**:
1. State refactor: 1 hour
2. Scanner refactor: 30 minutes
3. Handler updates: 1 hour
4. Testing: 1-2 hours
5. Documentation: 30 minutes

**Total**: ~4 hours with thorough testing

---

## Success Criteria

✅ **No request blocking during scan** (primary goal)
✅ **Lock-free handler access** (atomic.Value)
✅ **Memory usage acceptable** (<500MB for 200k files)
✅ **All tests pass** (existing + new)
✅ **No regressions** (handlers work identically)
✅ **Auto-rescan usable** (transparent to user)

---

## References

**Related Documentation**:
- `BUGS.md` - Bug #5 (File rescanning performance)
- `NEXT_CYCLE_IMPROVEMENTS.md` - Containerization analysis
- `PROJECT_OVERVIEW.md` - Current architecture

**Go Documentation**:
- `sync/atomic.Value` - Lock-free atomic loads/stores
- `sync.RWMutex` - Current (problematic) locking

**Design Pattern**:
- Double buffering (graphics/game dev pattern)
- Applied to application state instead of frame buffers

---

*Implementation proceeds in phases with testing at each stage. Rollback plan ensures low risk.*
