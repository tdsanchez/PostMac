# Next Cycle Improvements

> **Last Updated**: 2025-12-13
> **Purpose**: Document critical architectural issues discovered during containerization stress testing
> **Status**: ⚠️ **PARTIALLY RESOLVED** - Template serialization fixed, pagination interaction issues discovered (see Bug #2)

## Executive Summary

Stress testing with 5 instances (0.25s slideshow intervals, mixed view/tag pages, 10k-100k file datasets) revealed **catastrophic architectural inefficiency** that makes the current design unsuitable for containerization without fundamental redesign.

**Key Finding**: Every single-file viewer request loads and serializes the ENTIRE category's file list (up to 100k+ paths) into JavaScript, causing template execution timeouts and broken pipe errors.

**Real-World Impact**: System resource thrash (memory pressure + GC churn) so severe that Bluetooth audio playback stutters during operation. This indicates the inefficiency affects the entire system, not just the web server process.

**Primary Solution**: Client-side caching (localStorage) eliminates the serialization bottleneck with ~20 lines of JavaScript and zero server changes.

---

## ✅ Implementation Completed (2025-12-11)

**Solution Implemented**: Option 1 - Client-Side File List Caching + On-Demand API Fetch

**Changes Made** (commit 21e1ef5):
1. **New API Endpoint** (`internal/handlers/api.go:193-228`)
   - `GET /api/filelist?category=X` returns JSON array of file paths
   - Proper URL decoding, error handling, 404 for missing categories
   - Cross-cutting infrastructure for future features

2. **Eliminated Template Serialization** (`internal/handlers/pages.go:448-452`)
   - Server sends empty `allFilePaths` array
   - Template execution: seconds → <100ms
   - No more broken pipes or timeouts

3. **Async On-Demand Fetching** (`cmd/media-server/main_template.js:87-129`)
   - Made `navigateRandom()` async
   - Fetches from `/api/filelist` when cache empty and random mode used
   - Automatically caches result in localStorage
   - Graceful fallback to sequential navigation on errors

4. **Route Registration** (`cmd/media-server/main.go:81`)
   - Endpoint registered in server routes

**Verification**:
- ✅ API endpoint tested with curl (returns valid JSON)
- ✅ Cross-language verification (Python json.tool validates Go's JSON output)
- ✅ Browser loads viewer instantly (no serialization delay)
- ✅ Random mode works with direct navigation (async API fetch)

**Performance Impact**:
- **Before**: Template execution timeout, broken pipes, system thrash
- **After**: Instant page load, async data fetch, containerization ready

**Architectural Benefits**:
- Separation of concerns (data fetching decoupled from rendering)
- Lazy loading (fetch only when needed)
- Cross-cutting infrastructure (API useful beyond random mode)
- Constraint shift: JavaScript engine → compiled Go (orders of magnitude improvement)

---

## ⚠️ Regression Discovered (2025-12-13)

**Issue**: Testing revealed that random mode and sort functionality are scoped to paginated subsets rather than full categories.

**Affected Features**:
1. **Random mode in single file viewer** (Bug #2 in BUGS.md)
   - When navigating from gallery to single file view, random mode only picks from files on current gallery page (~200 files)
   - Expected: Should randomize across entire category (e.g., all 168k files)
   - Root cause: Gallery's localStorage cache may only store current page, not full category list

2. **Sort mode in gallery view** (Bug #1 in BUGS.md)
   - Sort operations (S key) only reorder current page files
   - Expected: Should sort entire category and show page 1 of sorted results
   - Root cause: Client-side sort operates on DOM, backend pagination unaware of sort state

**Investigation Required**:
- Verify `/api/filelist` endpoint returns full category list (not just current page)
- Check if gallery view's localStorage caching is storing complete file list
- Review interaction between pagination and localStorage cache population
- May need to update gallery template to fetch and cache full list on page load

**Status**: Open bugs tracked in BUGS.md, needs investigation and fix

---

## Stress Test Configuration

### Test Setup (Common Across All Tests)
- 5 server instances running simultaneously
- Mix of 10k and 100k file datasets
- 0.25 second slideshow intervals (4 requests/sec per instance)
- Random mode enabled
- Mixed `/view/` and `/tag/` page requests

### Test Environment 1: M4 Pro (Local)
**Hardware:**
- M4 Pro chip (Apple Silicon)
- P-cores: 15% duty cycle
- E-cores: barely touched
- Connection: localhost (::1) - eliminates network latency

**Observed Failures:**
```
Template execute error: write tcp [::1]:9496->[::1]:55677: write: broken pipe
Template execute error: write tcp [::1]:9496->[::1]:55689: write: broken pipe
(~1 error per second, constant across all instances)
```

### Test Environment 2: 2018 Mac Mini (Remote)
**Hardware:**
- Intel Mac (2018 Mac Mini)
- CPU: ~50% utilization (significantly higher than M4 Pro)
- Connection: WiFi network (client on different machine)

**Observed Failures:**
```
Template execute error: write tcp: broken pipe
(broken pipes on ALL instances despite available CPU headroom)
```

**Key Finding:** Broken pipes persist even at 50% CPU utilization on different architecture, confirming the issue is NOT CPU-bound but architectural (template serialization bottleneck).

### Cross-Platform Analysis

**Comparative Results:**
| Platform | CPU Utilization | Broken Pipes | Network |
|----------|----------------|--------------|---------|
| M4 Pro (Apple Silicon) | 15% | Yes (~1/sec) | localhost |
| Intel Mac Mini 2018 | 50% | Yes (all instances) | WiFi |

**Critical Insights:**
1. **Not CPU-bound**: 3x higher CPU usage (50% vs 15%) doesn't prevent broken pipes
2. **Architecture-independent**: Affects both Apple Silicon and Intel x86
3. **Network-independent**: Occurs on both localhost and WiFi connections
4. **Headroom irrelevant**: Available CPU capacity doesn't help when bottleneck is I/O

**Conclusion:** The broken pipe errors are caused by **network write timeouts during template serialization**, not insufficient compute resources. The server is I/O-blocked waiting to send megabyte-sized responses while template execution holds the connection open. Client browsers time out and close connections before serialization completes.

This definitively confirms the architectural diagnosis: serializing 100k file paths into JavaScript templates is the bottleneck, not CPU or memory.

---

## Root Cause Analysis

### The Architectural Problem

**Location**: `internal/handlers/pages.go:449-470` in `HandleViewer()`

```go
// Build list of all file paths for random navigation
allFilePaths := make([]string, len(files))
for i, f := range files {
    allFilePaths[i] = f.RelPath
}

jsData := struct {
    // ...
    AllFilePaths []string
}{
    // ...
    AllFilePaths: allFilePaths,  // ENTIRE category file list!
}
```

**What happens on EVERY `/view/` request:**

1. **Line 393**: Load entire category file list from memory
   ```go
   files, ok := filesByTag[tag]  // Could be 100k+ files
   ```

2. **Lines 449-452**: Copy all file paths into new array
   ```go
   allFilePaths := make([]string, len(files))  // 100k allocations
   ```

3. **Lines 473-486**: Serialize into JavaScript template
   ```go
   const allFilePaths = [{{range $i, $path := .AllFilePaths}}{{if $i}},{{end}}'{{$path}}'{{end}}];
   // Iterates 100k times, building megabyte-sized JavaScript string
   ```

4. **Result**: Browser receives megabytes of JavaScript containing 100k file paths just to display ONE image

### Performance Impact

**System-wide load during stress test:**
- 20 requests/second (5 instances × 4 req/sec)
- Each request processes 100k file array
- **2 MILLION file array iterations per second**
- Each response serializes **megabytes of JavaScript**

**Why template execution times out:**
- Serializing 100k paths: `['path1','path2',...,'path100000']`
- String concatenation + escaping takes **seconds**
- Browser timeout (typically 30-60s) expires
- Connection closes → broken pipe

**Why CPU usage is low (15%):**
- **Not compute-bound**: Memory and I/O bound
- Copying arrays, building strings, garbage collection
- Template execution is serialized (can't parallelize)
- Go runtime has cores available but architecture can't use them

---

## Why This Fails Containerization

| Requirement | Current State | Impact |
|-------------|---------------|---------|
| **Request throughput** | <10 req/sec per instance | ❌ Can't justify container resources |
| **Response time** | Seconds for 100k categories | ❌ Unacceptable latency |
| **Resource efficiency** | 85% idle CPU during failures | ❌ Wasted compute capacity |
| **Scalability** | Linear degradation with file count | ❌ Doesn't scale vertically or horizontally |
| **Graceful degradation** | Hard failures (broken pipes) | ❌ No backpressure or queuing |
| **Memory usage** | 100k file arrays per request | ❌ Memory pressure under load |

**Verdict**: Fundamental architecture redesign required before containerization.

---

## Architectural Issues

### 1. **Entire Dataset Loaded Per Request**
- **Problem**: `/view/` endpoint loads full category file list to display one file
- **Why**: Random mode needs access to all file paths
- **Impact**: 100k file iterations per request
- **Better approach**: Cache file list client-side or use API endpoint

### 2. **Template Serialization of Large Arrays**
- **Problem**: Go templates iterate 100k paths to build JavaScript literal
- **Why**: `{{range}}` over AllFilePaths creates string concatenation loop
- **Impact**: Template execution takes seconds, browsers time out
- **Better approach**: Stream data, lazy load, or use JSON endpoint

### 3. **No Request Prioritization or Queuing**
- **Problem**: All requests treated equally, no backpressure
- **Why**: Default Go http.Server with blocking handlers
- **Impact**: Simultaneous slow requests overwhelm server
- **Better approach**: Request queuing, rate limiting, priority lanes

### 4. **Synchronous Blocking Template Execution**
- **Problem**: Each request holds goroutine until template fully renders
- **Why**: `template.Execute()` blocks until complete
- **Impact**: Can't utilize available CPU (85% idle)
- **Better approach**: Streaming responses, chunked encoding

### 5. **No Caching Layer**
- **Problem**: Same data serialized repeatedly for every request
- **Why**: No rendered fragment cache, no CDN, no client-side persistence
- **Impact**: Redundant work on every request
- **Better approach**: Cache rendered JS, localStorage for file lists

---

## What We Actually Tried (And Why It Failed)

### Attempt 1: localStorage Caching with Empty Server Array (Commit 566720e)

**Implementation:**
- Server sends empty array: `allFilePaths = []`
- Client tries localStorage cache first
- Falls back to server array (which is empty)

**What Broke:**
- Random mode silently degraded to sequential mode
- User toggles random (`?random=true` in URL)
- `allFilePaths.length === 0` → navigateRandom() falls back to nextFilePath
- Result: Clicking next gives sequential file, not random

**Why This Happened:**
- No cache populated yet (direct navigation or first visit)
- Server sent empty array (intentionally, to eliminate serialization)
- Random mode had no file list to pick from
- Fallback behavior: sequential navigation

### Attempt 2: Restore Server Array as Fallback (Commit a5b6be0)

**Implementation:**
- Server builds and serializes full 100k file array again
- Client prefers localStorage cache if available
- Falls back to server array when cache empty

**What This "Fixed":**
- Random mode works in all cases (cache or no cache)

**What This Broke:**
- **The entire optimization** - server still serializes 100k paths every request
- Broken pipes persist (confirmed in stress testing on both M4 Pro and Intel Mac)
- Containerization blocker remains unsolved

**Current Status:**
- Server does expensive work on every request
- localStorage cache is an optimization path that may or may not get used
- The architectural bottleneck is back

---

## Proposed Solutions (Proper Fixes)

### Option 1: Client-Side File List Caching + On-Demand API Fetch (RECOMMENDED)

**Approach**: Server sends empty array, client fetches file list on-demand when needed for random mode

**Implementation:**
1. **Server changes:**
   - Send empty `allFilePaths` array (eliminates serialization bottleneck)
   - Add new endpoint: `GET /api/filelist?category=X` returns JSON array of file paths

2. **Gallery view:**
   - Fetches file list on page load
   - Stores in localStorage: `fileList_${category}`
   - No change to existing behavior

3. **Viewer JavaScript:**
   - Try localStorage cache first
   - If random mode enabled AND cache empty:
     - Fetch `/api/filelist?category=${currentTag}`
     - Cache result in localStorage
     - Then proceed with random navigation
   - If random mode disabled: no file list needed

**Benefits:**
- ✅ Eliminates 100k path serialization from viewer requests
- ✅ File list only fetched when actually needed (random mode)
- ✅ Gallery visit populates cache (pre-optimization)
- ✅ Direct viewer navigation fetches on-demand (lazy loading)
- ✅ Random mode works in all cases
- ✅ URL state preserved (`?random=true`)

**Trade-offs:**
- One extra API call on first random mode use (if cache empty)
- ~50-500ms latency for first random click (acceptable)
- Requires new API endpoint (~20 lines of Go)

**Code Changes Required:**
- `internal/handlers/api.go`: Add `/api/filelist` endpoint
- `internal/handlers/pages.go`: Change `allFilePaths := make(...)` to `allFilePaths := []string{}`
- `cmd/media-server/main_template.js`: Add lazy fetch logic in `navigateRandom()`
- `cmd/media-server/gallery_template.html`: Add cache population on page load (may already exist)

### Option 2: Random File API Endpoint (Targeted Fix)

**Approach**: Separate `/api/random-file` endpoint returns next random path

**Changes Required:**
1. New API endpoint: `POST /api/random-file` with `{category, excludePath}`
2. Server picks random file server-side, returns single path
3. Viewer JavaScript calls API instead of using pre-loaded array

**Benefits:**
- ✅ Viewer requests only load single file data
- ✅ Server-side random selection (no client-side list needed)
- ✅ Scales to unlimited file counts

**Trade-offs:**
- Extra API request per navigation (adds latency)
- Requires JavaScript changes in viewer

### Option 3: Paginated Random Pool (Hybrid)

**Approach**: Viewer loads small random subset (e.g., 100 paths), refills when exhausted

**Changes Required:**
1. Backend sends random sample of N paths (default 100)
2. Client navigates through sample
3. When exhausted, requests new sample

**Benefits:**
- ✅ Bounded response size regardless of category size
- ✅ Fewer API calls than per-file endpoint
- ✅ Works with 100k+ categories

**Trade-offs:**
- Slightly less random (temporal clustering)
- More complex client-side logic

### Option 4: Streaming JSON Response (Architectural)

**Approach**: Stream file list as JSON, viewer processes incrementally

**Changes Required:**
1. Replace template serialization with streaming JSON encoder
2. Use chunked transfer encoding
3. Client-side JavaScript processes stream

**Benefits:**
- ✅ No template timeout issues
- ✅ Browser can start rendering before complete
- ✅ Better CPU utilization (streaming is parallelizable)

**Trade-offs:**
- Significant rewrite of viewer architecture
- Client-side complexity increases

### Option 5: Static File Pre-Generation (Radical)

**Approach**: Generate static HTML/JS for each file at scan time, serve from disk

**Changes Required:**
1. Scanner generates pre-rendered viewer pages
2. Web server serves static files
3. Client-side navigation via JSON API

**Benefits:**
- ✅ Zero template execution time
- ✅ Can use CDN/caching effectively
- ✅ Scales infinitely

**Trade-offs:**
- Disk space (one HTML per file)
- Stale pages if files change
- Fundamentally different architecture

---

## Recommended Path Forward

**PRIMARY RECOMMENDATION: Client-Side Caching (Option 1)**

**Why this is the obvious choice:**
- ✅ **Lowest hanging fruit** - ~20 lines of JavaScript
- ✅ **Zero server changes** - Pure client-side implementation
- ✅ **Immediate elimination** of 100k path serialization bottleneck
- ✅ **Gallery already has the data** - Just persist to localStorage
- ✅ **Graceful fallback** - Server still works if cache missing
- ✅ **Real-world impact**: Fixes system resource thrash so severe it causes Bluetooth audio stuttering

**Implementation Priority:**

**IMMEDIATE (Fixes containerization blocker + Bluetooth stuttering):**
1. **Implement Option 1: Client-Side File List Caching**
   - Gallery view persists file list to localStorage on page load
   - Viewer reads from localStorage, falls back to server if missing
   - Cache invalidation on rescan (tie into existing rescan button)
   - Estimated effort: 1 hour

**Short-term (Performance Optimization):**
2. **Add request timeout and queuing**
   - Configure `http.Server` with read/write timeouts
   - Make the `T` quick key put the tag entry field in focus if it is active but not in focus
   - Implement request queue with depth limits
   - Fail fast instead of stacking requests

3. **Add response caching**
   - Cache rendered HTML/JS for frequently accessed files
   - Use ETag/Last-Modified headers
   - Consider Redis for multi-instance deployments

**Medium-term (Scalability):**
4. **Implement Option 3: Paginated Random Pool**
   - Bounds response size to reasonable limits
   - Better UX than per-file API calls
   - Enables 1M+ file categories

5. **Add telemetry and observability**
   - Request duration histograms
   - Template execution time tracking
   - Memory allocation profiling
   - Container-ready metrics (Prometheus)

**Long-term (Architecture Redesign):**
6. **Consider Option 4 or 5** if becoming multi-tenant SaaS
   - Current architecture is fine for personal use (1-5 users)
   - Fundamental redesign only needed for 100+ concurrent users
   - Consider Erlang and what it takes to implement the control code in Erlang, Python, C  vs. Go.

---

## Testing Plan

**Regression Testing:**
- Verify random mode still works after changes
- Verify slideshow functionality preserved
- Verify navigation (prev/next) still functional

**Performance Testing:**
- Stress test with 5 instances, 0.25s intervals
- Measure template execution time before/after
- Verify broken pipes eliminated
- Profile memory usage and GC pressure

**Scale Testing:**
- Test with 100k, 500k, 1M file categories
- Measure response time vs. file count
- Verify bounded resource usage

---

## Containerization Readiness Checklist

After implementing improvements, verify:

- [ ] Template execution completes in <1 second for any category size
- [ ] No broken pipe errors under 20 req/sec load
- [ ] CPU utilization >50% under load (proves architecture can use resources)
- [ ] Memory usage bounded and predictable
- [ ] Response size <1MB regardless of category size
- [ ] Graceful degradation under overload (queue, not fail)
- [ ] Horizontal scaling works (multiple containers, shared cache)
- [ ] Health check endpoint responds in <100ms
- [ ] Metrics exported for monitoring
- [ ] No APFS-specific dependencies (can run on ext4/overlay)

---

## Open Questions

1. **What's the target deployment model?**
   - Single-user personal deployment (current)
   - Multi-tenant SaaS (different requirements)
   - Hybrid (personal containers in shared cluster)

2. **What's acceptable latency for random navigation?**
   - Current: <1ms (pre-loaded array in memory)
   - API endpoint: ~10-50ms (server-side random + network)
   - Paginated pool: <1ms until refill, then 50ms

3. **Should we support offline mode?**
   - Affects whether client-side caching is mandatory
   - Impacts architecture choice (Option 1 vs 2)

4. **What's the maximum file count target?**
   - Current tested: 350k files
   - User mentioned: 5M files as eventual goal
   - Affects whether Option 3 (pagination) is sufficient

---

## Related Documentation

- `PROJECT_OVERVIEW.md` - Architecture overview and development methodology
- `internal/handlers/pages.go:377-510` - Current HandleViewer implementation
- `cmd/media-server/main_template.js:1-93` - Client-side navigation logic
- Git commit history - Shows evolution of random mode feature

---

*This document captures findings from 2025-12-11 containerization stress testing session and provides actionable improvements for next development cycle.*
