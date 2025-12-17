# Media Server - Project Overview

> **Last Updated**: 2025-12-16 (Pre-release bug prioritization, documentation cross-referencing)
> **Purpose**: Living documentation for understanding the codebase architecture
>
> **Current Status**:
> - ✅ Comment display and editing working (plist decoding implemented)
> - ✅ Random mode functional
> - ✅ **Video playback working** - Both gallery previews and single file view
> - ✅ **Pagination working** - Gallery views handle 168k+ files efficiently (200/page default)
> - ✅ **SQLite caching working** - Subsequent startups in ~1 second (14x faster!)
> - ✅ **localStorage caching working** - Client-side file list cache eliminates template serialization bottleneck
> - ✅ **Tag persistence working** - Tags survive server restarts (cache bug fixed!)
> - ✅ **"All" category fixed** - Shows all files in library instead of just root-level
> - ✅ **OS filesystem path display** - Full path visible in single file and gallery views
> - ✅ **Rescan button** - Incremental scan with fancy red→orange→green animated state transitions
> - ✅ **FSEvents auto-rescan** - Automatic cache updates on filesystem changes (3-second debounce)
> - ✅ **FSEvents DOM auto-refresh** - Browser automatically reloads after scan completes (seamless UX)
> - ✅ **"Untagged" category fixed** - Now persists across server restarts (cache loading bug fixed!)
> - ✅ **Containerization ready** - localStorage caching removes serialization bottleneck (see NEXT_CYCLE_IMPROVEMENTS.md)
> - ⚠️ QuickLook initialization issue (workaround: navigate first, then use QuickLook)
> - ⚠️ **Tag write permission errors** - Seeing intermittent "permission denied" on some files
> - 📊 Tested with 168,331 media files, 633 tag categories
> - 📊 **Expanded testing**: ~350k files at /Volumes/External/scratch (loads in 2 seconds)
> - 📊 **Stress testing**: 5 instances, 0.25s slideshow intervals, mixed 10k-100k file datasets
> - 💾 Cache: 43MB database, loads 168k files in 1 second
> - 🎯 **Proof of concept**: 350k files in 2 seconds proves Apple's limitations are artificial

## 🎯 Project Purpose

A local web server for organizing and viewing media files using macOS filesystem tags. Provides a web-based gallery interface with hierarchical folder navigation, tag management, and metadata viewing.


### Status and Roadmap

**Current Status:** Production-ready methodology demonstrated on real project (168k+ file media server)

**When Open Sourced, This Repository Will Provide:**
- Complete working application (Go-based media server)
- Comprehensive system card documentation
- Git history showing AI-assisted development in practice
- Examples of note-taking vs code-making mode sessions
- Performance optimization decisions and tradeoffs
- Real bug fixes and feature implementations

**Intended Impact:**
- Accelerate adoption of AI-native development practices
- Provide template for effective LLM collaboration
- Demonstrate that the paradigm shift is already viable
- Help developers transition from IDE-centric to conversation-driven workflows

### Personal Tools That Wouldn't Exist Otherwise

**The Effort-to-Value Revolution:**

Traditional development economics:
- Build a sophisticated media server with 168k+ file handling, SQLite caching, pagination, tag management, keyboard shortcuts?
- Weeks of full-time development effort
- Only justified for commercial products or large team projects
- Personal tools with real value remain unbuilt because effort doesn't justify outcome

AI-assisted development economics:
- Same sophisticated tool, built through conversational sessions
- Effort measured in hours of session time, not weeks of sprint work
- **Personal value justifies personal effort**
- Tools that solve actual problems get built, even for audience-of-one

**This project is an example:** A media server that has real value FOR THE CREATOR but would never have justified traditional development effort. The trivial effort investment compared to pre-LLM development makes personal tools viable.

### Real-World Use Case: ML Dataset Preparation at Scale

This media server isn't just file organization—it's a **machine learning dataset creation tool**:

**The Actual Problem Being Solved:**
- Human-indexing toward **5 million files** for perceptual AI training
- Generating ground truth labels through manual tagging workflow
- Creating training datasets for perceptual recognition algorithms
- Algorithms will be functionally inferred from 256x256 bitmaps + tag collections generated over time

**Why Existing Tools Fail (And Why This Is Intentional):**
- macOS Finder/system can only handle ~10k files at a time before performance degrades (claimed "APFS architecture limitations")
- Commercial DAM systems lock you into vendor ecosystems
- Apple's Photos app doesn't scale to millions of files
- No commercial tool supports "tag millions of files for ML training" workflow (market too small)

**Proof That Apple's Limitations Are Artificial, Not Technical:**
- **This tool loads 350,000 files in 2 seconds** (Go + SQLite cache)
- Same Mac hardware that Apple claims "can't handle" large media libraries
- Simple binary proves the hardware is perfectly capable
- **Conclusion**: Apple's limitations exist to appease media industry partners, not due to technical constraints
- The platform itself is the bottleneck—artificially hobbled to control user workflows

**What This Tool Enables:**
- ✅ Work with 168k+ files today, architected for 5M+ files eventually
- ✅ Break through macOS 10k-files-at-a-time limit via pagination and SQLite caching
- ✅ Break out of Apple's ecosystem (portable tags, open formats)
- ✅ Custom workflow optimized for ML labeling (keyboard shortcuts, batch operations, rapid tagging)
- ✅ Full control over data pipeline (no vendor lock-in)

**Why "You Can't Work With That Many Files" Is Limited Thinking:**

Skeptics say: "You can't actually work with 350k files at once—you can only interact with one file at a time."

This misses the point entirely. The value isn't in **simultaneous interaction**, it's in **corpus-level capabilities**:

1. **Navigation at scale enables pattern recognition**: Moving through 350k files reveals patterns impossible to see in smaller sets
2. **Incremental tagging builds training data**: One file at a time, over time, creates a massive labeled corpus
3. **The set itself is infrastructure**: 350k tagged files becomes the foundation for visual recognition algorithms
4. **Emergent capabilities from corpus coherence**: Search, navigate, and query across the entire set—traditional tools assume "bulk edit" is the only valuable operation on large sets

**Concrete example**: Tagging files one-at-a-time across 350k images creates ground truth labels for training perceptual recognition models. The ML model doesn't care that you tagged files sequentially—it cares that you have a coherent, labeled corpus at scale.

Traditional tools optimize for **bulk operations** (select all → apply change). This tool optimizes for **corpus-as-training-data** workflows where the set itself is the asset, not individual file manipulations.

**Why This Tool Exists:**
- No commercial product would build "ML dataset labeling for 5M personal files"
- Market size: ~1 person (maybe dozens worldwide)
- Traditional development cost: $50K-$100K+ engineering time
- LLM-assisted development cost: Conversational hours + API tokens
- **Result: Tool gets built because personal value justifies trivial effort**

This is the paradigm shift: **sophisticated personal tools that solve real problems but have no commercial market can now exist.**

### The Ecosystem Independence Angle

Breaking out of vendor lock-in:
- **Apple's trap**: Finder/Photos optimized for consumer use, breaks at scale
- **This approach**: Filesystem tags (portable), SQLite cache (open format), Go binary (cross-platform)
- **Future mobility**: Data remains accessible outside Apple ecosystem
- **No subscription**: One-time build effort, infinite runtime value

When personal tools are trivial to build, ecosystem lock-in loses its power. You're not trapped by vendor limitations—you build what you actually need.

### For the Rest of the Decade

The prediction: **LLMs + comprehensive system cards + git-based version control will be the dominant development model through 2030.**

This project exists to prove it's not just viable—it's already superior to traditional approaches for developers who can write and reason clearly.

**More importantly**: It proves that sophisticated tools solving real problems can now exist for audiences-of-one, enabling work (like ML dataset preparation at scale) that was previously impossible without commercial backing.

---

## 📁 Project Structure

```
media-server(applered)/
├── cmd/media-server/          # Main application entry point
│   ├── main.go               # Server initialization (routes, startup)
│   ├── index_template.html   # Homepage template (category grid)
│   ├── gallery_template.html # Gallery view template (file grid)
│   ├── main_template.html    # Single file viewer template
│   └── main_template.js      # Viewer JavaScript (navigation, shortcuts)
├── internal/                  # Internal packages
│   ├── config/               # File type definitions and priorities
│   │   └── types.go
│   ├── conversion/           # Format conversion (RTF, WebArchive → HTML)
│   │   └── converter.go
│   ├── handlers/             # HTTP request handlers
│   │   ├── api.go           # API endpoints (tags, metadata, etc.)
│   │   ├── file.go          # File serving with proper MIME types
│   │   └── pages.go         # Page rendering (index, gallery, viewer)
│   ├── metadata/             # EXIF and file metadata extraction
│   │   └── metadata.go
│   ├── models/               # Data structures
│   │   └── models.go
│   ├── persistence/          # Tag persistence (batched write queue)
│   │   └── writer.go
│   ├── scanner/              # Directory scanning and macOS tag I/O
│   │   ├── scanner.go       # Main scanning logic
│   │   └── tags.go          # macOS tag read/write operations
│   └── state/                # Global state management
│       └── state.go
├── go.mod & go.sum           # Go dependencies
├── media-server              # Compiled binary
└── PROJECT_OVERVIEW.md       # This file
```

## 🏗️ Architecture Overview

### Core Components

#### 1. **Entry Point** (`cmd/media-server/main.go:90`)
- Initializes application state
- Scans directory for media files on startup
- Registers HTTP routes
- Starts background batch write processor
- Embeds templates into binary for single-file deployment

#### 2. **Scanner** (`internal/scanner/scanner.go:247`)
- **Responsibility**: Builds the file index on startup
- Recursively walks directory tree
- Reads macOS extended attributes (tags and Finder comments)
- Creates multiple organizational views:
  - **All**: Root-level files only
  - **File Type Categories**: 📷 Images, 🎬 Videos, 📄 PDFs, 📝 Text Files, etc.
  - **Folder Categories**: 📁 hierarchical folder structure
  - **Tag Categories**: User-defined macOS tags
- Builds inverted index: `map[string][]FileInfo` (tag → files)
- Handles intermediate folders without direct files

**Key Functions**:
- `ScanDirectory()` - Main scanning entry point
- `UpdateFileTagsInMemory()` - Updates in-memory state when tags change
- `getBirthTime()` - Extracts file creation time on macOS

#### 3. **State Management** (`internal/state/state.go:139`)
- **Responsibility**: Thread-safe global state
- Uses `sync.RWMutex` for concurrent access
- **State Variables**:
  - `filesByTag`: Inverted index (tag → files)
  - `allFiles`: Master file list
  - `allTags`: Available user tags
  - `writeQueue`: Pending tag write operations
  - `conversionCache`: Cached HTML conversions

**Locking Pattern**:
- `RLockData()/RUnlockData()` for reads
- `LockData()/UnlockData()` for writes

#### 4. **Handlers** (`internal/handlers/`)

##### **Page Handlers** (`pages.go:452`)
- `HandleRoot()` - Homepage with category previews (line 106)
  - Groups folders by top-level
  - Sorts by priority and popularity
- `HandleTag()` - Gallery view for a specific tag/category (line 255)
  - Shows files in category
  - Displays child folders
  - Supports file selection via query param
- `HandleViewer()` - Single file viewer with prev/next navigation (line 319)
  - Renders file with keyboard shortcuts
  - Processes JavaScript template dynamically

**Helper Functions**:
- `parseFolderBreadcrumbs()` - Splits folder path into clickable segments (line 25)
- `groupFoldersByTopLevel()` - Aggregates folders for homepage (line 79)
- `getChildFolders()` - Finds immediate subfolders (line 203)

##### **API Handlers** (`api.go:374`)
- `HandleAddTag()` - Add tag to single file (line 33)
- `HandleBatchAddTag()` - Add tag to multiple files (line 81)
- `HandleRemoveTag()` - Remove tag from file (line 138)
- `HandleGetAllTags()` - Return available tags (line 182)
- `HandleGetFileList()` - Return file paths for category (line 193) **[NEW]**
- `HandleUpdateComment()` - Update Finder comment (line 192)
- `HandleMetadata()` - Return EXIF data (line 275)
- `HandleQuickLook()` - Reveal file in Finder (line 299)
- `HandleConvert()` - Convert RTF/WebArchive to HTML (line 337)
- `HandleShutdown()` - Graceful server shutdown (line 251)

**Update Pattern**:
1. Update in-memory state immediately (instant UI response)
2. Queue disk write for batched persistence
3. Return success to client

##### **File Handler** (`file.go:74`)
- Serves static files with proper MIME types
- Path traversal protection
- Special handling for text files (UTF-8 charset)

#### 5. **Persistence** (`internal/persistence/writer.go`)
- **Responsibility**: Batch write operations to disk
- Reduces I/O by grouping tag updates
- Background goroutine processes queue periodically
- Prevents excessive disk writes during UI interactions

#### 6. **Models** (`internal/models/models.go:69`)

**Core Types**:
```go
FileInfo             // File metadata with tags, comments, creation time
CategoryPreview      // Tag category with preview file and count
TagOperation         // Single tag add/remove request
BatchTagOperation    // Bulk tag operation
WriteQueueItem       // Pending disk write
FileMetadata         // EXIF and file system metadata
```

#### 7. **Configuration** (`internal/config/types.go:100`)
- Defines supported file extensions
- Maps extensions to category emojis
- Priority ordering for categories (folders first, then All, then types, then user tags)

**Category Priority** (lower = higher):
- 0: 📁 Folders
- 1: All
- 2-4: File types (Images, Videos, Audio)
- 5: Untagged
- 10: User tags

## 🎯 Single File View - Reference Implementation (GOLD STANDARD)

**Purpose**: This section documents how single file view currently works. Use this as the canonical reference when implementing pagination or fixing gallery view drift.

### Tag Interaction (main_template.html + main_template.js)

**Visual Appearance:**
- macOS "aqua" styling with proper button appearance
- Blue background (`#007AFF`) with white text
- Rounded corners (`border-radius: 14px`)
- Proper padding and minimum touch target size (`min-height: 36px`)
- Hover effects (darker blue `#0051D5`)
- Current tag highlighting (green `#34C759`)

**Interaction Methods:**
1. **Alt-click**: Remove tag from file (working correctly)
2. **Right-click**: Context menu with "Go to Tag Gallery" and "Delete Tag" options
3. **Regular click**: Navigate to tag gallery page
4. **Keyboard shortcut 'T'**: Open tag input field with autocomplete

**Tag Rendering** (`main_template.html:137-143`):
```html
<a href="/tag/{{urlEncode .}}" class="tag{{if eq . $.Tag}} current{{end}}" data-tag="{{.}}">
    {{.}}
</a>
```

**Tag Event Handlers** (`main_template.js:279-293`):
- Context menu on right-click (`contextmenu` event)
- Tag removal via context menu
- Navigation to tag gallery
- Click event propagation handling

### Comment Interaction

**Visual Appearance:**
- Dark theme styling with proper contrast
- Background: `rgba(255,255,255,0.05)`
- Hover effect: `rgba(255,255,255,0.08)`
- Empty state with italic placeholder text
- Proper text wrapping (`white-space: pre-wrap`)

**Interaction Methods:**
1. **Click on comment**: Enable editing mode (textarea appears)
2. **Ctrl+Enter / Cmd+Enter**: Save comment
3. **Escape**: Cancel editing
4. **Blur (click away)**: Auto-save comment
5. **Keyboard shortcut 'C'**: Enable comment editing

**Comment Editing** (`main_template.js:439-546`):
- Single edit state management (`isEditingComment` flag)
- Auto-save on blur
- Visual feedback during save (`comment-saving` class)
- Error handling with notifications
- Empty state detection and styling

### Keyboard Shortcuts (main_template.js:378-427)

**Navigation:**
- `←` Arrow Left: Previous file (browser back)
- `→` Arrow Right: Next file (or random if in random mode)
- `Escape`: Return to gallery (or stop slideshow if active)

**Tagging:**
- `T`: Open tag input with autocomplete
- `L`: Add "❤️" (love) tag
- `1-5`: Add star rating tags (1-★ through 5-★★★★★)

**Other:**
- `C`: Edit comment
- `Q`: QuickLook preview
- `S`: Toggle slideshow
- `R`: Toggle random mode
- `X`: Delete file (move to Trash with confirmation)
- `+`: Increase slideshow speed
- `-`: Decrease slideshow speed

**Keyboard Event Handling:**
- Proper detection of active input fields (don't intercept when editing)
- Browser shortcut exclusion (Cmd/Ctrl key combinations pass through)
- Event prevention for custom shortcuts

### Tag Autocomplete (main_template.js:318-376)

**Features:**
- Fetches all available tags from `/api/alltags`
- Filters matches as user types (case-insensitive)
- Keyboard navigation:
  - `↓` Arrow Down: Next suggestion
  - `↑` Arrow Up: Previous suggestion
  - `Enter`: Select highlighted or create new tag
  - `Escape`: Close autocomplete
- Visual highlighting of selected suggestion
- Click to select suggestion

### API Integration

**Tag Operations:**
- `POST /api/addtag` - Add tag to file
- `POST /api/removetag` - Remove tag from file
- Returns updated tag list for immediate UI refresh

**Comment Operations:**
- `POST /api/comment` - Update Finder comment
- Sends `{filepath: string, comment: string}`

**Metadata:**
- `GET /api/metadata/{filepath}` - Fetch EXIF and file metadata
- Displays dimensions, camera info, file size, dates

**Other:**
- `POST /api/quicklook` - Launch macOS QuickLook
- `POST /api/deletefile` - Move file to Trash and remove from cache
- `GET /api/alltags` - Get list of all available tags

### UI State Management

**No Framework:**
- Pure vanilla JavaScript
- Direct DOM manipulation
- Event delegation for dynamic elements
- Simple state variables (no Redux/state management library)

**State Variables:**
- `filePath`: Current file path
- `allTags`: Array of all available tags
- `contextMenuTag`: Tag for context menu action
- `isEditingComment`: Comment edit state
- `slideshowActive`, `slideshowDelay`, `randomMode`: Slideshow state

### Visual Feedback

**Notifications** (`main_template.js:171-176`):
- Toast-style notifications (top-right corner)
- Auto-dismiss after 2 seconds
- Slide-in animation
- Used for: tag added/removed, comment saved, slideshow toggle, etc.

**Slideshow Indicator:**
- Persistent indicator when slideshow active (top-left)
- Shows mode (SLIDESHOW vs RANDOM) and timing
- Orange background for visibility

### Critical Implementation Details

**Working Features to Preserve:**
1. Alt-click tag removal works correctly
2. macOS aqua button styling matches OS conventions
3. Comment editing is smooth with auto-save
4. Keyboard shortcuts don't conflict with browser
5. Autocomplete is responsive and intuitive
6. Visual feedback is immediate and clear
7. Context menu provides alternative interaction method
8. All operations update UI optimistically (instant feedback)

**Files to Reference:**
- `cmd/media-server/main_template.html` (lines 1-188)
- `cmd/media-server/main_template.js` (lines 1-617)
- CSS styles in main_template.html `<style>` block (lines 5-87)

---

## 📄 Gallery Pagination - Implementation Reference

**Purpose**: This section documents the pagination system implemented to handle large file collections (100k+ files) efficiently.

### Backend Implementation (`internal/handlers/pages.go:277-358`)

**Query Parameters:**
- `page` - Current page number (1-indexed, defaults to 1)
- `limit` - Files per page (defaults to 200, max 1000)

**Handler Logic:**
```go
// Parse pagination parameters
page := 1
limit := 200 // Default

// Calculate pagination
totalFiles := len(files)
totalPages := (totalFiles + limit - 1) / limit

// Slice files for current page
startIdx := (page - 1) * limit
endIdx := startIdx + limit
paginatedFiles := files[startIdx:endIdx]
```

**Template Data Structure:**
```go
struct {
    Tag          string
    Files        []models.FileInfo  // Paginated subset
    Count        int                // Files on current page
    TotalFiles   int                // Total files in category
    ChildFolders []SubfolderInfo
    Page         int                // Current page number
    Limit        int                // Page size
    TotalPages   int                // Total number of pages
    StartIdx     int                // First file index (1-indexed)
    EndIdx       int                // Last file index
}
```

### Frontend Implementation (`gallery_template.html`)

**Header Display (lines 107-113):**
- Shows: "Showing 1-200 of 168331 files (Page 1/842)"
- Conditional: Only shows pagination info if TotalPages > 1
- Otherwise shows simple count: "168331 files"

**Pagination Controls (lines 207-245):**
```html
<div class="pagination">
    <!-- Previous Button -->
    <a href="/tag/{{urlEncode .Tag}}?page={{sub .Page 1}}&limit={{.Limit}}">
        « Previous
    </a>

    <!-- Page Info -->
    <div class="pagination-info">Page {{.Page}} of {{.TotalPages}}</div>

    <!-- Next Button -->
    <a href="/tag/{{urlEncode .Tag}}?page={{add .Page 1}}&limit={{.Limit}}">
        Next »
    </a>

    <!-- Jump to Page -->
    <form action="/tag/{{urlEncode .Tag}}" method="get">
        <input type="number" name="page" min="1" max="{{.TotalPages}}">
        <input type="hidden" name="limit" value="{{.Limit}}">
        <button type="submit">Go</button>
    </form>

    <!-- Page Size Selector -->
    <select onchange="window.location.href='/tag/{{urlEncode .Tag}}?page=1&limit='+this.value">
        <option value="100">100</option>
        <option value="200" selected>200</option>
        <option value="500">500</option>
        <option value="1000">1000</option>
    </select>
</div>
```

**CSS Styling (lines 88-100):**
- macOS-style design matching existing UI
- Blue buttons (`#007AFF`) with hover effects
- Disabled state styling for first/last page buttons
- Responsive layout with flexbox

### Template Functions Required

**Added to `getTemplateFuncs()` (pages.go:68-70):**
- `sub` - Subtract integers (for Previous button: `{{sub .Page 1}}`)
- `add` - Add integers (for Next button: `{{add .Page 1}}`)

### Performance Characteristics

**Memory:**
- Only current page held in template (200 files vs 168k)
- No change to backend storage (full list still in memory)

**Rendering:**
- Page load time: <1 second for any page
- DOM elements per page: 200-1000 (vs 168k without pagination)
- No browser freezing or broken pipe errors

**User Experience:**
- Large categories (168k files) now fully usable
- 842 pages at 200 files/page default
- Configurable page size (100, 200, 500, 1000)
- Jump to specific page for quick navigation

### URL Examples

```
/tag/All                          # Page 1, default size (200)
/tag/All?page=2                   # Page 2, default size
/tag/All?page=1&limit=500         # Page 1, 500 files per page
/tag/%F0%9F%93%81+Partitia?page=5&limit=100  # Folder category, page 5, 100 per page
```

### Gallery Keyboard Navigation

**Auto-Select First Item (`gallery_template.html:336-339`):**
- First file in gallery automatically selected on page load
- Enables immediate keyboard navigation without clicking
- Blue highlight indicates selected file
- Designed for keyboard-driven workflow ("for graybeards")

**Keyboard Shortcuts (Gallery View):**
- `↑↓←→` Arrow Keys: Navigate between files in gallery grid
- `Enter`: Open selected file in viewer
- `T`: Add tag to selected file
- `E`: Edit comment on selected file
- `Cmd/Ctrl+Click`: Multi-select files for batch operations
- `S`: Toggle sort mode (name → date → size)
- `Shift+S`: Reverse sort order

**Implementation:**
- Auto-select runs in `initializeItems()` after DOM loads
- Uses existing `selectItem()` function for consistency
- Preserves all existing click handlers and selection logic
- Works seamlessly with pagination (first item per page selected)

### Future Enhancements

**Potential Improvements:**
- [ ] Remember user's preferred page size in localStorage
- [ ] Add keyboard shortcuts ([ ] for prev/next page)
- [ ] Show page range selector (e.g., "Pages: 1-10 | 11-20 | ...")
- [ ] Add "Jump to first/last" buttons
- [ ] URL hash preservation when navigating back from single file view

**Files to Reference:**
- `internal/handlers/pages.go` (lines 256-363) - HandleTag with pagination
- `cmd/media-server/gallery_template.html` (lines 88-100, 107-113, 207-245)
- `internal/scanner/scanner.go` (lines 108-113) - "All" category fix

---

## 🔄 Key Workflows

### Startup Flow
1. Parse command-line flags (port, directory)
2. Initialize state
3. Scan directory → build file index
4. Embed templates
5. Register HTTP routes
6. Start batch write processor
7. Listen on port

### Tag Update Flow
1. Client sends tag operation via API
2. Handler reads current tags from memory
3. Handler updates in-memory state immediately
4. Handler queues disk write operation
5. Handler returns success to client (instant UI update)
6. Background processor writes to disk later (batched)

### Page Render Flow
1. Client requests page (/, /tag/X, /view/X)
2. Handler acquires read lock on state
3. Handler reads relevant data from in-memory structures
4. Handler releases read lock
5. Handler executes template with data
6. HTML sent to client

### Folder Navigation Flow
1. Scanner creates category for each folder path (e.g., "📁 Photos/2024")
2. Scanner creates parent aggregations (e.g., "📁 Photos" includes all subfolders)
3. Gallery page shows:
   - Files directly in current folder
   - Child folder cards (with file counts)
4. Breadcrumbs parsed from category name for navigation

### Rescan Flow (Background Incremental Scan)
**User Experience**: Click "Rescan" button → Animated state transitions (red → orange → green) → Auto-reset
**Implementation**: Non-blocking background scan with fancy UI feedback

1. **Trigger** (`/api/rescan`):
   - User clicks Rescan button in gallery header
   - Frontend sends POST to `/api/rescan`
   - Backend checks if scan already in progress (returns error if yes)
   - Backend starts scan in goroutine, returns immediately
   - Frontend sets button to "scanning" state (orange, pulsing, rotating icon)

2. **Background Scan**:
   - Set `scanState.isScanning = true`
   - Call `scanner.ScanDirectory(serveDir)` (full filesystem walk)
   - Save results to cache via `scanner.SaveToCache(cache)`
   - Set `scanState.completed = true` when done

3. **Status Polling** (`/api/scanstatus`):
   - Frontend polls every 500ms while scanning
   - Backend returns `{isScanning: bool, completed: bool}`
   - When `completed: true`, frontend transitions button to green
   - Backend auto-clears completed flag after first read

4. **UI State Transitions** (CSS animations):
   - **Idle**: Red background (#FF6B47), "Rescan" text
   - **Scanning**: Orange background (#FFA500), pulsing opacity, rotating icon, "Scanning..." text
   - **Complete**: Green background (#34C759), "Scan Complete!" text, holds 3 seconds
   - **Reset**: Automatically returns to idle state

**Files involved**:
- `internal/state/state.go` - Scan state tracking
- `internal/handlers/api.go` - `/api/rescan` and `/api/scanstatus` endpoints
- `cmd/media-server/gallery_template.html` - Button UI, animations, polling logic
- `cmd/media-server/main.go` - Route registration

**Design notes**:
- ✨ Fancy because it is (red→orange→green color transitions, pulse animations, rotating icon)
- Non-blocking: user can navigate away mid-scan
- Next page load automatically uses updated data
- Fixes cache coherence issues from external file operations

### FSEvents Auto-Rescan Flow (Automatic Cache Coherence)
**User Experience**: Filesystem changes trigger automatic background rescans → Cache always fresh
**Implementation**: fsnotify package wraps macOS FSEvents API for kernel-level efficiency

1. **Initialization** (`main.go` startup):
   - Create watcher with `watcher.New(serveDir, dbCache)`
   - Recursively add all subdirectories to watch
   - Skip hidden dirs, .photoslibrary bundles
   - Start event monitoring goroutines

2. **Event Monitoring** (`watcher.watchEvents()`):
   - Listen to `fsWatcher.Events` channel
   - Filter out noise: .DS_Store, ._* metadata, temp files, hidden files
   - Only process: Write, Create, Remove, Rename operations
   - Queue rescan request (non-blocking channel)

3. **Debouncing** (`watcher.debouncedRescan()`):
   - Wait 3 seconds after last filesystem event
   - Prevents scan spam during bulk operations
   - Single rescan covers all queued changes

4. **Automatic Rescan** (`watcher.triggerRescan()`):
   - Check if scan already in progress (skip if yes)
   - Set scan state (UI shows progress via existing polling)
   - Call `scanner.ScanDirectory()` + `scanner.SaveToCache()`
   - Set completed state (UI shows green completion)

**Advantages**:
- **Zero-maintenance cache**: No manual rescan needed
- **Kernel-level efficiency**: FSEvents is macOS native, not polling
- **Smart debouncing**: Bulk operations trigger single rescan
- **UI feedback**: Existing rescan button shows auto-scan progress
- **Graceful degradation**: Falls back to manual rescan if watcher fails

**Files involved**:
- `internal/watcher/watcher.go` - Complete FSEvents implementation
- `cmd/media-server/main.go` - Watcher initialization
- Uses existing scan state from rescan button feature

**Edge cases handled**:
- Recursive directory watching (fsnotify only watches one level)
- New subdirectories created after startup
- Photos library internals ignored
- macOS metadata file noise filtered

**Current Behavior (Confirmed Working)**:
- Filesystem changes (add/delete/modify) trigger automatic rescan
- Updated data appears in cache and in-memory state almost instantaneously
- **Auto-refresh implemented**: Browser automatically reloads 2 seconds after scan completes
- **Seamless UX**: file changes → auto-scan → auto-refresh → updated view without manual intervention
- **Visual feedback**: Rescan button transitions red → orange (scanning) → green (complete) → page reload
- **Example**: Adding files to "Untagged" category via Finder automatically appears in browser view
- **Smoke Test Passed**: File duplication detected and reflected in GUI near-instantly

## 🛠️ Dependencies

```go
github.com/pkg/xattr          // Extended attributes (macOS tags)
github.com/rwcarlsen/goexif   // EXIF parsing
howett.net/plist              // Property list parsing
github.com/mattn/go-sqlite3   // SQLite database driver
github.com/fsnotify/fsnotify  // Filesystem event monitoring (FSEvents on macOS)
```

## 🎨 Supported File Types

### Media Files
- **Images**: .gif, .jpg, .jpeg, .png, .tif, .tiff, .webp
- **Videos**: .mp4, .mov, .avi, .mkv, .m4v
- **Documents**: .pdf

### Text Files
- **Code**: .go, .sh, .mod, .sum
- **Documents**: .txt, .md, .json, .yaml, .yml

### Web Files
- **HTML**: .html, .htm
- **Convertible**: .rtf, .webarchive (converted to HTML on-demand)

## 🔐 Security Considerations

- Path traversal protection in file handlers
- Clean path validation (`filepath.Clean()`)
- Prefix checking to ensure files are within serve directory
- No authentication (designed for local use)

## 🚀 Performance Optimizations

1. **In-Memory Index**: All file metadata cached on startup
2. **Batched Writes**: Tag updates grouped to reduce disk I/O
3. **Read-Write Locks**: Multiple concurrent readers allowed
4. **Embedded Templates**: No disk reads for template files
5. **Conversion Cache**: HTML conversions cached in memory

## 📝 Recent Changes (from Git History)

- **(2025-12-11)**: Implement dedicated `/api/filelist` endpoint to eliminate template serialization bottleneck **[COMPLETED]**
  - **Problem discovered**: Containerization stress testing (5 instances, 0.25s slideshow, 10k-100k files) revealed catastrophic inefficiency
  - **Root cause**: Every `/view/` request serialized entire category file list (100k+ paths) into JavaScript template
  - **Impact**: 20 req/sec load = 2M file iterations/sec, template timeouts, broken pipes, system resource thrash causing Bluetooth audio stuttering
  - **Solution**: Dedicated API service with lazy loading
    - New endpoint: `GET /api/filelist?category=X` returns JSON array of file paths (`api.go:193-228`)
    - Server sends empty `allFilePaths` array (no more template serialization)
    - Viewer JavaScript fetches from API on-demand when random mode used
    - Automatic localStorage caching for instant reuse
  - **Performance improvement**: Template execution from seconds to <100ms, instant page loads
  - **Architectural benefits**:
    - Separation of concerns (data fetching decoupled from rendering)
    - Cross-cutting infrastructure (API useful for future features)
    - Constraint shift: JavaScript engine → compiled Go (orders of magnitude improvement)
  - **Containerization**: ✅ Blocker eliminated, architecture now containerization-ready
  - **Verification**: Cross-language testing (Python json.tool validates Go's JSON output)
  - **Documentation**: Full analysis in `NEXT_CYCLE_IMPROVEMENTS.md`
  - **Files**: `internal/handlers/api.go`, `main_template.js`, `internal/handlers/pages.go`, `main.go`
  - **Commits**: `566720e` (localStorage attempt), `a5b6be0` (fallback issues), `21e1ef5` (API endpoint solution)
  - **Status**: ✅ COMPLETED - Production ready
- **(2025-12-10)**: Implement dual navigation paths for homepage category cards
  - **Feature**: Two distinct navigation flows from homepage
  - **Mouse/Click navigation**: Click category card → opens preview file in single-file view
  - **Keyboard navigation**: Arrow keys navigate, Enter opens gallery view
  - **Rationale**: Mouse users get immediate visual context, keyboard users get systematic browsing
  - **Implementation**:
    - Card href points to `/view/{category}?file={preview}` (click behavior)
    - JavaScript intercepts Enter key on focused cards to navigate to `/tag/{category}` instead
    - Arrow key navigation (↑↓←→) between category cards
    - Auto-select first category on page load
    - Selected card gets blue outline highlight
  - **UX consistency**: Homepage keyboard navigation now matches gallery view
    - Same arrow key navigation model
    - Same auto-select behavior
    - Same visual feedback (blue outline)
    - Consistent muscle memory across all views
  - **Status**: ✅ Working smoothly with consistent UX across views
  - Files: `cmd/media-server/index_template.html` (JavaScript keyboard handler, arrow navigation, selection state)
  - Commits: `23aded1` (navigation change), `8c213aa` (dual paths), `60df064` (arrow key navigation)
- **(2025-12-10)**: Implement random sort mode in gallery view
  - **Feature**: Add random and reverse-random as sort modes
  - **S key cycle**: name → date → size → random → reverse-random → (back to name)
  - **Shift-S**: Reverses sort direction for name/date/size modes
  - **Implementation**: Fisher-Yates shuffle algorithm, stores random order for consistency
  - **Visual indicators**: 🎲 SORT: RANDOM and 🔄 SORT: REVERSE-RANDOM
  - **Design goal**: Maximize visual churn/variety when browsing millions of files
  - Files: `cmd/media-server/gallery_template.html`
  - Commit: `72bfc64`
- **(2025-12-10)**: Implement click-to-file navigation in gallery view
  - **Feature**: Clicking preview image navigates directly to that file in single-file viewer
  - **Previous behavior**: Required selection + Enter or double-click
  - **New behavior**:
    - Click preview image (photo/video/icon) → navigate to that file immediately
    - Click info area (filename/tags/comment) → select for keyboard navigation
    - Enter and double-click still work as backup methods
  - **Note**: Backend already supported `?file=` parameter - frontend now utilizes it
  - Files: `cmd/media-server/gallery_template.html` (click handlers)
  - Commit: `72bfc64`
- **(2025-12-10)**: Implement FSEvents DOM auto-refresh for seamless UX
  - **Feature**: Browser automatically reloads 2 seconds after FSEvents-triggered scan completes
  - **User Experience**: Adding files externally (via Finder) → auto-scan → auto-refresh → files appear without manual intervention
  - **Visual feedback**: Rescan button transitions red → orange (scanning) → green (complete) → page reload
  - **Example use case**: Add files to "Untagged" category via Finder, browser view automatically updates
  - **Implementation**: Modified scan completion handler in gallery_template.html to reload page after 2-second completion state
  - **Impact**: Zero-maintenance cache coherence now extends to zero-maintenance UI updates
  - Files: `cmd/media-server/gallery_template.html` (lines 969-982)
  - **Also documented**: Drag-and-drop file tagging future enhancement (drop files onto tag categories for batch tagging)
- **(2025-12-09)**: Implement OS filesystem path display
  - **Feature**: Display full OS filesystem path in both single file viewer and gallery view
  - **Single file view**: Path shown below comment section in monospace font
  - **Gallery view**: Directory path shown in header next to file count
  - **Critical bug fix**: Cache loading wasn't reconstructing full Path from RelPath
  - **Solution**: Added `filepath.Join(serveDir, RelPath)` reconstruction after cache load
  - **Impact**: Users can now see actual filesystem locations for files and directories
  - **Styling**: Neutral gray monospace text (not link blue) for clear readability
  - Files: `cmd/media-server/main_template.html`, `gallery_template.html`, `internal/handlers/pages.go` (getDir function), `internal/scanner/scanner.go` (Path reconstruction)
- **(2025-12-09)**: Fix plaintext file viewer UX - navigation arrows and margins
  - **Problem**: Navigation arrows overlaid text content making files unreadable, especially near edges
  - **Solution**: Text-file-specific styling that moves arrows outside content area
  - **Implementation**:
    - Added `body.viewing-text` CSS class applied only when viewing text files
    - Arrows positioned `fixed` at screen edges (left: 0, right: 0) instead of overlaying content
    - Text content gets generous margins (80px left/right, 30px top/bottom)
    - Narrower arrow buttons (50px wide) to minimize screen real estate
  - **Impact**: Text files (including PROJECT_OVERVIEW.md) now fully readable without occlusion
  - **Scope**: Only affects text file viewing - images, videos, PDFs unchanged
  - Files: `cmd/media-server/main_template.html` (lines 50-54, 92)
- **(2025-12-09)**: Implement file deletion with Trash support
  - **New feature**: Press `X` key to delete current file with modal confirmation
  - **User flow**: Confirmation modal → Move to Trash → Remove from cache/memory → Navigate to next file
  - **Keyboard shortcuts updated**: `X` now deletes file, `R` toggles random mode (previously `X`)
  - **Backend**: New `/api/deletefile` endpoint using osascript to move files to macOS Trash
  - **Cache integration**: Automatically removes deleted files from SQLite cache and in-memory structures
  - **Safety**: Modal confirmation prevents accidental deletion, files can be recovered from Trash
  - Files: `cmd/media-server/main_template.html`, `main_template.js`, `internal/handlers/api.go`, `internal/scanner/scanner.go`, `internal/cache/cache.go`, `internal/state/state.go`
- **(2025-12-08 end of day)**: Implement SQLite caching (commit 0b7d338)
  - **Performance**: 14x faster subsequent startups (1 second vs 14 seconds for 168k files)
  - **Database**: 43MB SQLite cache stores files, tags, comments, scan metadata
  - **Features**: Automatic cache detection, 7-day staleness check, incremental tag/comment updates
  - **Impact**: Solves slow startup problem from PROJECT_OVERVIEW critical issues
  - Files: `internal/cache/cache.go`, scanner integration, persistence layer updates
- **(2025-12-08 end of session)**: Restore pagination with working videos (commit c8a71bb)
  - **Problem**: Pagination was unnecessarily removed when fixing videos
  - **Fix**: Restored paginated template from commit d44ef4c
  - **Result**: Both pagination AND videos now working together
  - **Lesson learned**: Pagination and video playback are independent - don't remove one to fix the other
- **(2025-12-08 evening)**: Restore working video playback (commit 0694599)
  - **Problem**: Video playback completely broken after pagination implementation

---

## 💾 SQLite Caching System

**Purpose**: Eliminate slow startup scans by caching file metadata in SQLite database.

### Performance Improvement

**Before caching:**
- Every startup: Full filesystem scan (~14 seconds for 168k files)
- APFS overhead: Binary tree traversal + nanosecond timestamps
- macOS throttling: Performance degradation at 100k+ files

**After caching:**
- First run: Full scan + cache save (~15 seconds total)
- Subsequent runs: Load from cache (~1 second)
- **14x speedup** for 168k file library

**Expanded Testing (2025-12-09):**
- Successfully tested with ~350k files at `/Volumes/External/scratch`
- Cache system scales to larger libraries
- Performance characteristics under evaluation with larger dataset

### Database Schema (`internal/cache/cache.go`)

```sql
-- Files table stores core file metadata
CREATE TABLE files (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    rel_path TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    size INTEGER NOT NULL,
    created INTEGER NOT NULL,  -- Unix timestamp
    comment TEXT
);

-- Tags table (normalized, one row per file-tag pair)
CREATE TABLE tags (
    file_id INTEGER NOT NULL,
    tag_name TEXT NOT NULL,
    PRIMARY KEY (file_id, tag_name),
    FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE
);

-- Scan metadata tracks last scan timestamp
CREATE TABLE scan_metadata (
    directory_path TEXT PRIMARY KEY,
    last_scan_time INTEGER NOT NULL,
    total_files INTEGER NOT NULL,
    total_tags INTEGER NOT NULL
);
```

### Cache Lifecycle

**Startup (`scanner.LoadOrScanDirectory()`):**
1. Check if `.media-server-cache.db` exists in serve directory
2. Read scan metadata to check cache freshness
3. If cache valid (< 7 days old): Load files from DB
4. If cache stale/missing: Full filesystem scan + save to DB
5. Build in-memory indexes (`filesByTag`, folder hierarchy, etc.)

**Runtime Updates:**
- Tag changes: Write to filesystem → Update cache → Update memory
- Comment changes: Write to filesystem → Update cache → Update memory
- Batched writes (5-second intervals) update cache asynchronously

**Cache Location:**
```
/Volumes/External/media/.media-server-cache.db
```
Hidden file in serve directory (43MB for 168k files)

### Implementation Details

**Loading Flow (`scanner.buildInMemoryStructures()`):**
- Read all files and tags from database
- Rebuild folder hierarchy (same logic as ScanDirectory)
- Create category indexes (All, folders, types, user tags)
- Sort by creation time (newest first)

**Saving Flow (`scanner.saveToCache()`):**
- Called after successful filesystem scan
- Writes all files and tags to database in single transaction
- Updates scan metadata with current timestamp

**Update Flow (`persistence.ProcessBatchWrites()`):**
- Batched tag writes update filesystem first
- On success: Update cache via `cache.UpdateFileTags()`
- Comments update cache via `cache.UpdateFileComment()`

### Cache Invalidation

Cache is rebuilt when:
- Database doesn't exist
- Last scan > 7 days ago
- Total file count is 0 (empty database)
- Schema error (old database format)

### Benefits

✅ **Fast startups**: 1 second vs 14 seconds for 168k files
✅ **Reliable counts**: Single source of truth, no timing dependencies
✅ **Persistent**: Survives server restarts
✅ **No external dependencies**: SQLite built into Go
✅ **Efficient updates**: Tag/comment changes update cache incrementally
✅ **Solves APFS issues**: Avoids repeated nanosecond timestamp reads

### Future Enhancements

- [ ] FSEvents integration for automatic cache updates
- [ ] Incremental scans (only scan changed files)
- [ ] Cache migration for schema changes
- [ ] Vacuum/optimize database periodically
- [ ] Cache statistics/diagnostics endpoint

  - **Fix**: Restored gold template configuration
    - Removed `.SelectedFile` template references causing errors
    - Used minimal video attributes: `controls + muted` for gallery, `controls + autoplay + muted` for viewer
  - **Critical lesson**: Video playback breaks when adding extra attributes like `loop`, `playsinline`, `preload`
- **(2025-12-08)**: Implement full pagination system and fix "All" category
  - **"All" category fix** (`internal/scanner/scanner.go:108-113`):
    - Changed from showing only root-level files to showing ALL files in library
    - Now correctly displays all 168,331 files instead of 0
  - **Pagination implementation**:
    - Backend (`internal/handlers/pages.go:277-358`): Added `?page=N&limit=M` support, array slicing, pagination metadata
    - Frontend (`gallery_template.html:88-100, 207-245`): Dynamic header, prev/next buttons, page jump, size selector
    - Default: 200 files per page (configurable: 100, 200, 500, 1000)
    - Added `sub` template function for pagination math
  - **Auto-select feature removal**:
    - Removed `SelectedFile` parameter and auto-scroll JavaScript
    - Removed `?select=` parameters from homepage links
    - Performance improvement: No expensive array searching on page load
  - **Impact**: Large categories (168k files) now usable, eliminates browser freezing
- **c76b47c (2025-12-05)**: Fix Finder comment decoding to handle plist format
  - Comments set via Finder/osascript are stored as binary plist in xattr
  - Previously reading raw bytes showed garbage like "00bplist!..."
  - Now properly decode plist in `GetMacOSComment()` to extract clean text
  - Location: `internal/scanner/tags.go:59-75`
- **af42027 (2025-12-05)**: Fix random mode in slideshow (pass all file paths to JS for true random selection)
- **9fccbdf**: Fix navigation and keyboard shortcuts for hierarchical folders
- **9b99f5e**: Fix navigation for intermediate folders without direct files
- **c51a85d**: Fix 404 error for top-level folder categories
- **2808208**: Add subfolder navigation cards in gallery view
- **005841f**: Implement hierarchical breadcrumb navigation and folder grouping

## 🧩 Extension Points

### Adding New File Type Support
1. Add extension to `config.SupportedExts` map
2. Add to appropriate category map (TextExts, ConvertibleExts, etc.)
3. Update `GetFileTypeCategory()` with emoji and category name
4. Add MIME type in `handlers/file.go` if needed

### Adding New API Endpoints
1. Define handler function in `internal/handlers/api.go`
2. Register route in `cmd/media-server/main.go`
3. Update in-memory state if needed
4. Queue disk write if persistence required

### Adding New Metadata Fields
1. Update `models.FileInfo` struct
2. Update scanner to populate field
3. Update templates to display field

## 🐛 Known Issues / TODOs

### Pre-Release Blockers (MUST FIX BEFORE PUBLIC RELEASE)

**Status**: Preparing for first public release - these issues block GitHub publication

#### Bug #4: Tag editing UX issues - focus management and exit behavior
- **Priority**: **HIGHEST** pre-release blocker
- **Issue**: Tag edit field doesn't reliably maintain focus, Escape key requires two-step process
- **Impact**: Core workflow feature unusable - "very not usable" (user feedback)
- **Quick wins**: Arrow keys to cancel, Enter to exit after tag match
- **Larger fix**: Unified tag editing between gallery and single file views
- **See**: BUGS.md Bug #4 for full details and proposed fixes
- **Related**: Tag functionality divergence between views (existing issue below)

#### Bug #5: File rescanning performance - blocks server during scan
- **Priority**: **HIGH** pre-release blocker
- **Issue**: Auto-rescanning makes service unusable for large file sets, server becomes unresponsive
- **Impact**: FSEvents auto-rescan becomes liability instead of feature
- **Root cause**: State lock contention during scan blocks request handlers
- **Proposed fixes**: Chunked scanning, copy-on-write state, or separate read/write state
- **See**: BUGS.md Bug #5 for full details and solution options
- **Related**: Cache coherence with external file operations (existing issue below)

#### Bug #8: Random mode finds non-existent files after deletions
- **Priority**: **HIGH** (related to rescanning)
- **Issue**: "Extremely frequently" random mode navigates to deleted files
- **Impact**: Core feature (random mode) frequently broken
- **Root cause**: FSEvents/cache/localStorage update chain not properly synchronized
- **See**: BUGS.md Bug #8 for full details
- **Related to**: Bug #5 (both involve scan/cache coherence)

**Additional bugs tracked in BUGS.md**: 5 more bugs (2 High, 2 Medium, 1 Critical)
**Total bug count**: 8 open bugs requiring resolution before public release

---

### Critical Architecture Changes (HIGH PRIORITY)

- [x] **~~Implement persistent filesystem cache (SQLite)~~** (COMPLETED 2025-12-08: 0b7d338)
  - ✅ **Implemented**: SQLite cache with 14x performance improvement
  - ✅ **Performance**: 168k files load in 1 second (vs 14 second scan)
  - ✅ **Database**: 43MB SQLite file with files, tags, comments, scan metadata
  - ✅ **Cache lifecycle**: Auto-detection, 7-day staleness, incremental updates
  - ✅ **Solved problems**:
    - Instant subsequent startups (load from cache vs full scan)
    - Reliable file counts (single source of truth)
    - Works with pagination (fast page-based serving)
    - Solves APFS nanosecond timestamp cache thrashing
  - **Future enhancements**:
    - [ ] FSEvents integration for automatic cache updates on file changes
    - [ ] Incremental scans (only scan changed files)
    - [ ] Cache migration for schema changes

- [x] **~~Fix All view file duplication bug~~** (RESOLVED 2025-12-08)
  - **Original symptom**: "All" view was empty (0 files) instead of showing all files in library
  - **Root cause identified**: "All" category was only populated with root-level files (`relDir == "."`), not all files
  - **Solution implemented** (`internal/scanner/scanner.go:108-113`):
    - Changed from `filesByTag["All"] = rootFiles` to `filesByTag["All"] = allFilesList`
    - "All" now correctly shows all 168,331 files in the library
  - **Testing**: No duplicates found in any tested categories (5-star, Files, Partitia, All)
  - **Status**: ✅ RESOLVED - All category now works as expected

- [x] **~~Fix "Untagged" category disappearing on cache load~~** (RESOLVED 2025-12-10)
  - **Original symptom**: "Untagged" category visible after fresh scan, disappears after server restart
  - **User report**: "there used to be an untagged category that is now gone" - bio-memory confirmed correct!
  - **Root cause identified**: `buildInMemoryStructures()` missing logic to add files with no tags to "Untagged" category
  - **Solution implemented** (`internal/scanner/scanner.go:370-372`):
    - Added `if len(file.Tags) == 0` check in `buildInMemoryStructures()`
    - Matches existing logic in `ScanDirectory()` (line 89-90)
    - Files with zero tags now properly added to "Untagged" in both scan paths
  - **Impact**: Fresh scans AND cache loads now both populate "Untagged" category correctly
  - **Status**: ✅ RESOLVED - Untagged category persists across restarts

### Active Issues

- [ ] **Sort mode only sorts current page, not entire category** (Bug #1 in BUGS.md)
  - **Symptom**: Pressing 'S' to cycle through sort modes only reorders files visible on current page
  - **When discovered**: 2025-12-13 during testing
  - **Severity**: High - breaks expected sort behavior for paginated categories
  - **Expected**: Sort should operate on entire category and return page 1 of sorted results
  - **Actual**: Sort only reorders ~200 files in current DOM, backend pagination unaware
  - **Related code**: `cmd/media-server/gallery_template.html` (S key handler), `internal/handlers/pages.go:277-358` (pagination)
  - **Root cause**: Pagination implementation issue - sort was designed before pagination
  - **Potential fix**: Server-side sort with query param (`?sort=name&order=asc`)
  - **Status**: Open, tracked in BUGS.md

- [ ] **Random mode in single file view only randomizes within current page files** (Bug #2 in BUGS.md)
  - **Symptom**: When navigating from gallery to single file view, random mode only picks from files on current gallery page (~200 files)
  - **When discovered**: 2025-12-13 during testing
  - **Severity**: High - defeats purpose of random mode for large categories
  - **Expected**: Should randomize across entire category (e.g., all 168k files in "All")
  - **Actual**: Random navigation constrained to files from current gallery page
  - **Related code**: `cmd/media-server/main_template.js:87-129`, `internal/handlers/api.go:193-228` (/api/filelist)
  - **Root cause**: Gallery localStorage cache may only store current page, not full category
  - **Investigation needed**: Verify /api/filelist endpoint usage and localStorage caching behavior
  - **Related**: NEXT_CYCLE_IMPROVEMENTS.md shows this should be fixed - indicates regression or incomplete implementation
  - **Status**: Open, tracked in BUGS.md

- [ ] **Photos .photoslibrary bundles should be skipped during scanning**
  - **Description**: Need to skip Photos .photoslibrary bundles during directory scanning
  - **When discovered**: 2025-12-09
  - **Context**: .photoslibrary packages are macOS Photos libraries that appear as directories but should be treated as single units
  - **Impact**: Scanning these bundles can cause performance issues and expose internal Photos library structure
  - **Potential implementation**: Add check in scanner to skip directories with .photoslibrary extension
  - **Related code**: `internal/scanner/scanner.go` (ScanDirectory function)
  - **Priority**: Medium - prevents scanning unnecessary internal library structures
  - **Status**: Identified, needs implementation

- [ ] **Investigate retrieving all OS user tags via Go**
  - **Description**: Explore whether there's a way to get all available macOS user tags from the system (not just tags on scanned files)
  - **When discovered**: 2025-12-09
  - **Context**: Currently we only discover tags by scanning files - may be useful to get complete tag list from macOS
  - **Use case**: Could provide autocomplete with all system tags, not just tags currently in use
  - **Investigation needed**: Research macOS APIs for retrieving complete tag list (mdfind, Spotlight APIs, system preferences)
  - **Related code**: `internal/scanner/tags.go`, `internal/handlers/api.go` (HandleGetAllTags)
  - **Priority**: Low - nice-to-have enhancement
  - **Status**: Investigation phase

- [ ] **Tag write permission errors on certain files**
  - **Symptom**: Intermittent "permission denied" errors when attempting to write tags to filesystem
  - **Example error**: `xattr.Set /Volumes/External/media/scripts/modtag7.sh com.apple.metadata:_kMDItemUserTags: permission denied`
  - **When discovered**: 2025-12-09 during active usage
  - **Context**: Attempting to adjust tag on plain text file (.sh script)
  - **File permissions**: Checked and appear normal (no obvious permission issues)
  - **Related code**: `internal/scanner/tags.go` (SetMacOSTags function), `internal/persistence/writer.go` (batch write processing)
  - **Potential causes**:
    - macOS System Integrity Protection (SIP) restrictions on certain paths
    - File locked by another process
    - Extended attribute restrictions on executable files
    - Volume-specific xattr limitations
    - Race condition with OS-level indexing (Spotlight, etc.)
  - **Impact**: Tag changes fail silently from user perspective (UI updates but filesystem doesn't)
  - **Investigation needed**:
    - Determine if issue is consistent for specific file types (.sh, executables)
    - Check if issue is path-specific (certain volumes or directories)
    - Add error surfacing to UI (currently fails silently)
    - Consider retry logic with exponential backoff
  - **Status**: Newly discovered, needs investigation and error handling improvements

- [ ] **Cache coherence with external file operations**
  - **Symptom**: File moves/deletions performed outside the media server (via Finder, command line, etc.) are not reflected in the cache until manual rescan
  - **When discovered**: 2025-12-09 during testing discussions
  - **Root cause**: SQLite cache + in-memory state are isolated from filesystem events
  - **Current behavior**:
    - Tool tries to navigate to deleted/moved file → 404 or error
    - Cache shows files that no longer exist
    - New files added externally not visible until rescan
  - **User impact**: Cache can become stale if files managed outside the tool
  - **Potential solutions**:
    1. **FSEvents integration** (macOS file system events API)
       - Monitor serve directory for changes
       - Automatically update cache when files added/removed/moved
       - Real-time cache coherence
    2. **Sync scan feature** (user-initiated)
       - Keyboard shortcut to trigger incremental scan
       - User points to changed directory via file dialog
       - Tool recursively scans just that subtree
       - Updates cache with changes
    3. **Exit scan** (automatic on shutdown)
       - Perform full or incremental scan on Ctrl+C or shutdown button
       - Ensures cache coherent with filesystem before exit
       - Trade-off: Slower shutdown
    4. **Lazy validation** (on-demand)
       - When file not found, trigger rescan of parent directory
       - Automatic recovery from cache staleness
       - May cause unexpected delays during navigation
  - **Related code**:
    - `internal/cache/cache.go` (cache operations)
    - `internal/scanner/scanner.go` (scanning logic)
    - `cmd/media-server/main.go` (shutdown handling)
  - **Priority**: Medium - affects reliability when using tool alongside other file management workflows
  - **Status**: Design phase - multiple potential solutions, needs architectural decision

- [x] **~~Plaintext file viewer - navigation arrows obscure content~~** (RESOLVED 2025-12-09)
  - **Problem**: Navigation arrows (← →) positioned inside content area overlaid text, making files unreadable
  - **Solution**: Added text-file-specific CSS mode that positions arrows outside content area at screen edges
  - **Implementation**: `body.viewing-text` class with fixed positioning, generous text margins (80px sides)
  - **Status**: ✅ RESOLVED - Text files now fully readable with arrows outside viewing area

- [ ] **Responsive design for large screens and tablet/mobile viewing**
  - **User requirements** (2025-12-06):
    1. **Large screen auto-zoom**: Images should auto-zoom to 75% of browser width on big screens
    2. **Tablet/mobile single file view**: Need different rendering scheme for iPad/tablet - current single file mode unusable on iOS because navigation elements take up most of screen and content is not visible
  - **Status**: Requirements documented, awaiting implementation approval
  - **⚠️ IMPLEMENTATION PROTOCOL**: DO NOT implement without explicit user approval
    - User must say "go:", "dewit", or similar explicit command before making changes
    - Present proposed changes first, get approval, then implement
    - Attempted implementation on 2025-12-06 broke navigation and produced poor results on iPad - changes were reverted
  - **Related code**: `cmd/media-server/main_template.html` (CSS for media-container, nav-btn, info-panel), `cmd/media-server/main_template.js` (responsive behavior)
  - **Priority**: Medium - UX improvement for specific devices

- [ ] **Navigation breaks with special characters (+, -) in filenames**
  - **Symptom**: In single file viewer, clicking next/previous arrows or using keyboard navigation fails to load files with `+` or `-` (and possibly other special characters) in their names
  - **When discovered**: 2025-12-06 during active usage
  - **Root cause**: URL encoding/decoding issue - `+` is interpreted as space in query strings, special characters not properly encoded/decoded
  - **Related code**:
    - `cmd/media-server/main_template.js:49-60` (buildURL function using encodeURIComponent)
    - `internal/handlers/pages.go:322-331` (URL parameter extraction and unescaping)
    - `cmd/media-server/main_template.js:6-7` (prevFilePath and nextFilePath template variables)
  - **Investigation needed**: Need to ensure consistent encoding throughout the chain (Go template → JavaScript → URL → Go handler)
  - **Status**: Newly discovered, needs encoding fix

- [ ] **Tag functionality divergence between single view and gallery view (UX regression)**
  - **Symptom**: Tag appearance and behavior has diverged across multiple iterations
    - **Single file view** (GOLD STANDARD): Tags have macOS "aqua" appearance, alt-click works for tag removal
    - **Gallery view** (DEGRADED): Tags lost aqua appearance, alt-click no longer works, only right-click context menu functions
  - **When discovered**: 2025-12-06 during active usage across multiple sessions
  - **Root cause - Uncertain, likely compound issue**:
    - **Development drift**: Features added/modified in one view but not synchronized to the other over multiple iterations
    - **Rendering failures**: Gallery view's inability to render large file sets may be preventing proper CSS/JS loading
    - **Not fully understood**: The exact mechanism causing the appearance/functionality loss needs investigation
    - Could be: incomplete page rendering, CSS not loading, JS event handlers failing to attach, or pure code drift
    - **Key insight**: Fixing rendering performance issues (hard limit, smart click routing, pagination) may actually restore tag functionality if incomplete rendering is breaking CSS/JS initialization
  - **Impact**: Inconsistent UX between views, users must learn different interaction patterns
  - **Investigation strategy**: Implement performance fixes first, then reassess whether tag drift persists with properly rendering pages
  - **Required action**: Unify tag implementation across both views
    - Use single file view as reference implementation
    - Extract tag component into shared template/partial if possible
    - Ensure both views support: alt-click removal, macOS aqua styling, right-click context menu, consistent keyboard shortcuts
  - **Related code**:
    - `cmd/media-server/main_template.html` + `main_template.js` (single view - GOLD STANDARD)
    - `cmd/media-server/gallery_template.html` (gallery view - needs sync)
    - Tag rendering: gallery line 165-173, single view line 137-143
    - Tag handlers: gallery line 475-488, single view line 279-293
  - **Priority**: Medium - UX consistency issue
  - **Status**: Feature drift identified, needs refactoring to unify tag component behavior
  - **⚠️ CRITICAL NOTE FOR PAGINATION**: When implementing paginated gallery view, ALL aspects of single file view tag/comment functionality MUST be preserved:
    - macOS aqua styling for tags
    - Alt-click for tag removal
    - Right-click context menu
    - Comment editing behavior
    - Keyboard shortcuts
    - Visual feedback and animations
    - See detailed single file view documentation below for complete feature inventory

- [ ] **PDF viewer navigation and focus issues**
  - **Symptom**: Navigation buttons overlap PDF content; focus issues prevent scrolling PDFs with arrow keys and using tag/comment features
  - **When discovered**: 2025-12-05 during PDF rating workflow
  - **Attempted fix (2025-12-05)**: Changed nav buttons to `position: fixed`, added iframe focus detection to keyboard handlers
  - **Result**: Made navigation significantly worse - changes were reverted
  - **Related code**: `cmd/media-server/main_template.html` (CSS for .nav-btn, .media-container), `cmd/media-server/main_template.js` (keyboard event handlers around line 378)
  - **Status**: Deferred - original behavior restored, requires more careful analysis of focus/interaction model
  - **Note**: Any future attempts must preserve existing navigation UX

- [ ] **QuickLook does not work on initial page load until navigation occurs**
  - **Symptom**: Pressing 'q' on first file does nothing
  - **Workaround**: Navigate to next file with right arrow, then QuickLook works
  - **When discovered**: 2025-12-05 during comment fix session
  - **Investigation needed**: JavaScript initialization or browser back/forward cache (bfcache)
  - **Related code**: `cmd/media-server/main_template.js` (keyboard event handlers)
  - **Status**: Reproducible, workaround exists, low priority

### Approved Changes - Implementation Queue

- [x] **Click-to-specific-file navigation in gallery view** (COMPLETED 2025-12-10)
  - ✅ Implemented: Clicking preview image navigates to that file in single-file viewer
  - ✅ Backend already supported `?file=` parameter
  - ✅ Click handlers distinguish between preview clicks (navigate) and info area clicks (select)
  - See "Recent Changes" section for details

- [x] **Random sort mode in gallery view** (COMPLETED 2025-12-10)
  - ✅ Implemented: S-key cycle includes random and reverse-random modes
  - ✅ Fisher-Yates shuffle algorithm for fair randomization
  - ✅ Visual indicators: 🎲 SORT: RANDOM and 🔄 SORT: REVERSE-RANDOM
  - See "Recent Changes" section for details

- [x] **Dual navigation paths for homepage** (COMPLETED 2025-12-10)
  - ✅ Implemented: Mouse clicks go to preview file, keyboard Enter goes to gallery view
  - ✅ Arrow key navigation added for UX consistency with gallery view
  - ✅ Working smoothly with consistent navigation across all views
  - See "Recent Changes" section for full details

### Feature Requests

- [x] **Improve homepage keyboard navigation** (COMPLETED 2025-12-10)
  - ✅ Added arrow key navigation between category cards
  - ✅ Auto-select first category on page load
  - ✅ Consistent UX with gallery view
  - See "Recent Changes" for implementation details

- [ ] **Display OS filesystem path for current location**
  - **Description**: Show actual filesystem path similar to how breadcrumbs show in-app navigation
  - **Use case**: User needs to know the actual OS path for current file/category being viewed
  - **When requested**: 2025-12-09
  - **Potential implementation**:
    - Add path display in header or info panel
    - Show full path: `/Volumes/External/media/folder/file.jpg`
    - Make path copyable (click to copy to clipboard)
    - Consider showing both breadcrumb navigation AND OS path
  - **Related code**: `cmd/media-server/main_template.html`, `gallery_template.html` (header sections)
  - **Priority**: Medium - UX enhancement for file location awareness

- [ ] **[B]eaming feature - transfer file to predefined target directory**
  - **Description**: Keyboard shortcut 'B' opens modal asking if user wants to move/copy current file to a predefined target directory
  - **Use case**: Facilitate moving files between root directories for file management workflows
  - **When requested**: 2025-12-09
  - **Requirements**:
    - User configurable target directory (settings or config file)
    - Modal confirmation with target path displayed
    - Option to move or copy
    - Update cache after operation
    - Navigate to next file after transfer
  - **Workflow**:
    1. User views file, presses 'B'
    2. Modal: "Transfer [filename] to [target_dir]? [Move] [Copy] [Cancel]"
    3. Execute operation
    4. Update cache and in-memory state
    5. Navigate to next file
  - **Potential implementation**:
    - Add `/api/transfer` endpoint
    - New keyboard handler for 'B' key
    - Configuration: Add `--target-dir` flag or config file
    - Cache update: Remove from source (if move), add to target location awareness
  - **Related code**:
    - `cmd/media-server/main_template.js` (keyboard shortcuts)
    - `internal/handlers/api.go` (new transfer endpoint)
    - `cmd/media-server/main.go` (configuration)
  - **Priority**: Medium-Low - workflow enhancement for power users

- [ ] **Drag-and-drop file tagging - drop zone for tag categories**
  - **Description**: Allow users to drag one or more files from the gallery view and drop them onto a tag category to apply that tag to all dropped files
  - **Use case**: Rapid tagging workflow - visually select files and drag to tag categories for batch tagging
  - **When requested**: 2025-12-10
  - **Requirements**:
    - Drag-and-drop support in gallery view
    - Drop zones on tag category cards (both in gallery view and homepage)
    - Works for single or multiple files
    - Visual feedback during drag (highlight drop zones)
    - Batch tag operation on drop
    - Auto-refresh after tagging to show updated tag assignments
  - **Workflow**:
    1. User selects one or more files in gallery view (Cmd/Ctrl+click for multi-select)
    2. User drags selected files over tag category card
    3. Drop zone highlights to indicate valid drop target
    4. User drops files on tag category
    5. Batch tag operation applies tag to all dropped files
    6. Visual feedback confirms tagging (notification or auto-refresh)
  - **Potential implementation**:
    - Add HTML5 drag-and-drop API to gallery items
    - Make tag category cards drop zones (`ondrop`, `ondragover` handlers)
    - Reuse existing `/api/batchaddtag` endpoint for batch tagging
    - Visual feedback: CSS classes for drag states (`.dragging`, `.drop-target`)
    - Preserve multi-selection state during drag operation
  - **Alternative implementation**:
    - Could also work in single file view: drag current file to tag badge or tag list
    - Could include folder categories as drop zones for organizing files
  - **Related code**:
    - `cmd/media-server/gallery_template.html` (drag/drop handlers)
    - `internal/handlers/api.go` (existing batchaddtag endpoint)
    - Gallery JavaScript (lines 294-1066) - selection and tag management
  - **Priority**: Medium - significant UX enhancement for tagging workflow, especially with large file collections
  - **Benefits**:
    - Faster than keyboard shortcuts for visual thinkers
    - Natural interaction model (drag = move/apply)
    - Works well with multi-select (tag many files at once)
    - Reduces cognitive load (see tag categories visually, drag files to them)

### General Limitations

- [ ] No file upload capability (read-only except tags)
- [ ] No multi-user support
- [ ] Keyboard shortcuts may conflict with browser shortcuts

### Performance Issues

**Architectural Overview**: Multiple performance issues stem from the current architecture's reliance on synchronous, in-memory filesystem scanning at startup. The following issues are interconnected and point toward a need for persistent caching with incremental updates.

**Critical APFS Performance Issue**: Apple's APFS nanosecond timestamp resolution causes severe CPU/GPU cache performance degradation when working with large file sets. The nanosecond-precision timestamps cause cache line invalidation, destroying cache performance even on modern M4 Pro hardware. This is a fundamental architectural conflict between APFS and modern CPU cache design - Apple's filesystem team and hardware team have created conflicting requirements.

**Performance degradation thresholds**:
- **1k files**: Recommended hard limit for gallery view (conservative, ensures good UX)
- **5k files**: Observable performance degradation begins (cache thrashing becomes measurable)
- **80k+ files**: Catastrophic failure (browser unresponsive, broken pipe errors, never recovers)

This affects:
- File scanning (binary tree traversal + cache thrashing)
- Template rendering (processing large file arrays)
- Browser rendering (when combined with DOM overhead)

**Broken pipe errors**: When rendering large categories (80k+ files), the server logs numerous "write tcp: broken pipe" errors as the browser abandons connections due to timeouts/unresponsiveness.

- [x] **~~Gallery view rendering fails with 20k+ files, auto-select causes problems~~** (RESOLVED 2025-12-08)
  - **Solution implemented**: Full pagination system with auto-select feature removed
  - **Implementation details**:
    1. **Pagination** (`internal/handlers/pages.go:277-358`):
       - Added `?page=N&limit=M` query parameter support
       - Default page size: 200 files (configurable: 100, 200, 500, 1000)
       - Proper array slicing for each page
       - Pagination metadata passed to template (TotalPages, StartIdx, EndIdx)
    2. **UI Controls** (`gallery_template.html:88-100, 207-245`):
       - Dynamic header: "Showing 1-200 of 168331 files (Page 1/842)"
       - Previous/Next navigation buttons
       - Jump-to-page input field
       - Per-page size dropdown selector
       - macOS-style UI design
    3. **Auto-select removal**:
       - Removed `SelectedFile` parameter from `HandleTag`
       - Removed auto-select JavaScript logic from gallery template
       - Removed `?select=` parameters from homepage category links
       - Performance improvement: No expensive array searching on page load
  - **Testing results**:
    - "All" category (168,331 files): 842 pages at 200 files/page
    - Page navigation tested and working
    - No performance issues browsing paginated views
  - **Status**: ✅ RESOLVED - Large categories now usable with pagination

- [x] **~~Browser thread blocking detection and early abort for massive categories~~** (RESOLVED 2025-12-08)
  - **Original symptom**: Clicking on category with 80k+ files causes browser to become 100% unresponsive
  - **Solution**: Pagination system prevents rendering all files at once
  - **How pagination solves this**:
    - Maximum DOM elements per page: 200-1000 (configurable)
    - Categories with 168k files now render in <1 second per page
    - No more broken pipe errors or browser freezing
    - No need for manual partitioning of large collections
  - **Status**: ✅ RESOLVED - Pagination eliminates browser thread blocking

- [ ] **Inconsistent/variable file counts with large libraries**
  - **Symptom**: File counts vary randomly across page loads, counts are close but not identical (e.g., showing 4788, then 4792, then 4785 files)
  - **When discovered**: 2025-12-06 during active usage with large file collections
  - **Root cause analysis**: Multiple potential causes indicating fundamental reliability issues:
    1. **Race conditions**: Concurrent access to `filesByTag` during scanning despite mutex locks
    2. **Incomplete scans**: macOS throttling interrupts filepath.Walk before completion
    3. **No persistence**: Every restart rescans filesystem and potentially gets different results
    4. **Timing-dependent**: Count depends on when throttling kicks in or when scan is interrupted
  - **Related code**: `internal/scanner/scanner.go` (file counting during scan), `internal/state/state.go` (concurrent access to filesByTag)
  - **Impact**: Unreliable counts undermine trust in the application, indicates deeper scan reliability issues
  - **Why this matters**: If you can't reliably count files, you can't trust the index is complete
  - **Relationship to other issues**: Direct symptom of macOS file access throttling and lack of persistent cache
  - **Proposed solution**: Implement Redis/SQLite persistent cache (addresses root causes #2, #3, #4)
  - **Status**: Critical reliability issue that indicates need for persistent scan cache

- [ ] **File scanning hits 100k file wall on macOS (Mojave+)**
  - **Symptom**: Scanning performance degrades dramatically when approaching or exceeding ~100,000 files
  - **When discovered**: 2025-12-06 during active usage with large media libraries
  - **Root cause - Compound macOS/APFS issues**:
    1. **macOS throttling**: System throttles file system access for large numbers of files since Mojave (10.14)
    2. **APFS binary tree traversal**: Filesystem must traverse binary trees for large file sets
    3. **APFS nanosecond timestamps**: Destroys CPU/GPU cache performance - nanosecond resolution causes cache line invalidation
    4. **Performance cliff at ~5k files**: Observable even on M4 Pro hardware due to cache thrashing
    5. **Go filepath.Walk**: Affected by all OS-level throttling and APFS overhead
  - **Apple architectural conflict**: APFS nanosecond file resolution fundamentally conflicts with modern CPU cache architecture
  - **Related code**: `internal/scanner/scanner.go:18-247` (ScanDirectory function using filepath.Walk)
  - **Impact**: Startup scan becomes prohibitively slow with large libraries (100k+ files), performance degradation begins at ~5k files
  - **Potential solutions**:
    - **Persistent cache layer**: Implement Redis or SQLite to cache scan results (solves both performance and reliability)
    - Use file system events (FSEvents API) to incrementally update cache
    - Implement incremental/background scanning with progress indication
    - Add file count limits or directory exclusion patterns
    - Consider alternative scanning approaches (parallel walkers, C-based scanners)
  - **Note**: Redis/SQLite caching would address both the 100k file wall AND the inconsistent count issues
  - **Status**: Known limitation on modern macOS, persistent cache layer needed for large libraries

---

### 🏗️ Architectural Recommendation: Persistent Cache Layer

The performance and reliability issues documented above share a common root cause and solution:

**Current Architecture Problems:**
1. **Synchronous scanning at startup** - Every restart requires full filesystem walk
2. **No persistence** - All metadata must be re-read from extended attributes every time
3. **macOS throttling** - OS limits prevent reliable scanning of large libraries (100k+ files)
4. **Memory-only state** - No way to recover from incomplete scans
5. **Race conditions** - Concurrent access during scanning leads to inconsistent counts

**Proposed Solution: Redis or SQLite Cache**

Implement a persistent cache layer that stores:
- File paths and metadata (size, dates, type)
- Tag associations (from extended attributes)
- Finder comments
- Folder hierarchy
- Last scan timestamps

**Benefits:**
- **Instant startup** - Load index from cache instead of filesystem scan
- **Incremental updates** - Use macOS FSEvents API to detect changes and update only modified files
- **Reliable counts** - Single source of truth, not timing-dependent
- **Better performance** - No repeated extended attribute reads
- **Graceful degradation** - Can serve requests even if scan is incomplete

**Implementation Strategy:**
1. SQLite for simplicity (single file, no external dependencies) OR Redis for performance
2. FSEvents integration for real-time file system monitoring
3. Background sync process to reconcile cache with filesystem
4. Schema: tables for files, tags, folders, with appropriate indexes
5. Fallback to direct filesystem read if cache miss
6. **Combine with pagination** - Cache enables efficient page-based queries

**Complementary Solution: Pagination**

While persistent cache addresses backend performance, pagination solves frontend rendering limits:
- **Independent of cache** - Can be implemented separately or together
- **Immediate benefit** - Makes large categories usable right away
- **Query parameters**: `?page=1&limit=200` in gallery URLs
- **Works with cache** - SQLite enables efficient `LIMIT/OFFSET` queries
- **User experience**: Clear page navigation vs. waiting for 20k+ items to render

**Combined Approach (Recommended):**
1. **Immediate**: Add hard limit (1k files) in HandleTag to prevent catastrophic browser failures and APFS cache thrashing
2. **Quick win (before pagination)**: Implement smart click routing in gallery view - distinguish image clicks from card clicks
   - Image click → viewer with `?select=` (auto-select enabled)
   - Card click → category view without select (fast, no auto-select overhead)
   - Prevents handler from being bogged down searching for file position in large category lists
   - Simple event delegation change, no template restructuring needed
3. **Short term**: Implement pagination (200-500 files per page, immediate fix for gallery rendering)
4. **Medium term**: Add SQLite cache (solves startup time, scan reliability, enables fast pagination)
5. **Both together**: Cache provides fast page queries, pagination keeps DOM manageable

**Current Workaround (Band-aid):**
- Users must manually partition large collections into smaller sub-categories using folder structure or tags
- This is unsustainable and requires manual intervention for every large category
- Hard limit + pagination will eliminate need for manual partitioning

**Why 1k file limit?**
- Conservative threshold that ensures good UX even on fast hardware (M4 Pro tested)
- Prevents APFS nanosecond timestamp cache thrashing before it becomes noticeable
- Avoids cascade failures: filesystem → template rendering → network → browser
- Can be raised to 5k-10k once pagination is implemented (users choose page size)

**Trade-offs:**
- Added complexity (cache invalidation, consistency management)
- Storage requirement (~1-10MB per 100k files, depending on metadata)
- Potential for cache staleness if external tools modify tags
- Pagination adds navigation overhead for browsing

**Priority:**
- **Pagination**: High - Quick win for gallery usability
- **Persistent Cache**: High - Fundamental architecture improvement

---

### Recently Fixed

- [x] ~~Tags not persisting across server restarts~~ (Fixed 2025-12-08: 5232dcc)
  - **Problem**: Tag changes made in one session were saved to disk and cache, but disappeared when server restarted
  - **Root cause**: Critical bug in `internal/cache/cache.go:156-160` - `LoadFiles()` used array indices (1, 2, 3...) to lookup files in `fileMap` instead of actual database IDs
  - **Impact**: Tags were loaded from SQLite correctly but lost during copy to final files slice
  - **Solution**: Track file order separately (`fileOrder[]`) and build final files slice AFTER tags are loaded, using correct database IDs
  - **Testing**: Tags now persist correctly across server restarts - Session A tag changes visible in Session B
  - **Note**: This was a day-one bug in the SQLite cache implementation (commit 0b7d338) - cache writes worked but cache reads lost tags
- [x] ~~Pagination unnecessarily removed~~ (Fixed 2025-12-08: c8a71bb)
  - **Problem**: Pagination removed when fixing videos (independent features)
  - **Solution**: Restored paginated template from d44ef4c
  - **Result**: Both pagination and videos working together
- [x] ~~Video playback completely broken~~ (Fixed 2025-12-08 evening: 0694599)
  - **Problem**: Videos not showing at all - no previews in gallery, no playback in single file view
  - **Root cause**: Template errors from `.SelectedFile` references + incorrect video attributes
  - **Solution**: Restored gold template configuration and removed `.SelectedFile` references
  - **Working config**:
    - Gallery view: Lazy-loaded videos with `controls` + `muted` only
    - Single file view: `controls` + `autoplay` + `muted` only
  - **Critical lesson**: DO NOT add extra attributes (`loop`, `playsinline`, `preload`) - they break playback
  - Files: `cmd/media-server/gallery_template.html`, `main_template.html`, `main_template.js`
- [x] ~~"All" category empty/broken~~ (Fixed 2025-12-08)
  - Changed from showing only root-level files (0 files) to showing ALL files (168,331)
  - Modified `internal/scanner/scanner.go:108-113` to use `allFilesList` instead of `rootFiles`
- [x] ~~Auto-select performance issue~~ (Fixed 2025-12-08)
  - Removed expensive auto-select feature that searched through thousands of files on page load
  - Homepage links now simple: `/tag/All` instead of `/tag/All?select=...`
  - Also removed `.SelectedFile` template references that were causing errors
- [x] ~~Random mode not working~~ (Fixed 2025-12-05: af42027)
  - Added `allFilePaths` array to JavaScript template for true random selection
- [x] ~~Finder comments showing binary plist garbage~~ (Fixed 2025-12-05: c76b47c)
  - Added plist decoding in `GetMacOSComment()` at `internal/scanner/tags.go:59-75`


## 💡 Development Tips

### Building
```bash
go build -o media-server cmd/media-server/main.go
# or use provided script
./build_server.sh
```

### Running
```bash
./media-server --dir=/path/to/media --port=8080
```

### Testing Tag Operations
Tags are stored in macOS extended attributes (`com.apple.metadata:_kMDItemUserTags`)

### Load Testing
Comprehensive load testing scripts validate architectural improvements and containerization readiness:

**Python Load Tester** (`load_test.py`):
```bash
# Standard mixed workload test
python3 load_test.py --url http://localhost:8080 --workers 5 --duration 60

# Stress test viewer with random mode (validates serialization fix)
python3 load_test.py --url http://localhost:8080 --scenario viewer-random --workers 10 --duration 30

# API endpoint performance test
python3 load_test.py --url http://localhost:8080 --scenario api-only --requests 1000

# Maximum stress test
python3 load_test.py --url http://localhost:8080 --scenario stress --workers 20 --duration 120
```

**Perl Load Tester** (`load_test.pl`):
```bash
# Same scenarios, Perl implementation
perl load_test.pl --url http://localhost:8080 --workers 5 --duration 60
perl load_test.pl --url http://localhost:8080 --scenario stress --workers 20
```

**Scenarios:**
- `mixed` - Mixed workload (homepage, gallery, viewer, API)
- `viewer-random` - Focus on viewer with random mode (stress test serialization fix)
- `gallery` - Gallery page requests with pagination
- `api-only` - API endpoint testing (/api/filelist)
- `stress` - Aggressive stress test with no delays

**Key Metrics:**
- Broken pipes (should be 0 after serialization fix)
- Response times (p50, p95, p99)
- Throughput (requests/second)
- Success rate
- Error distribution

**Cross-Language Verification:**
- Python and Perl implementations validate consistent behavior
- Different HTTP client libraries catch edge cases
- VLSI-style verification methodology

### Debugging
- Check state with browser DevTools (React DevTools not needed - vanilla JS)
- Log statements use Go's `log` package
- State mutations protected by explicit lock/unlock calls

---

## 📚 Code Reference Quick Links

| Component | File | Line | Description |
|-----------|------|------|-------------|
| Main entry | `cmd/media-server/main.go` | 28 | Server initialization |
| Route registration | `cmd/media-server/main.go` | 56-69 | HTTP routes |
| Directory scan | `internal/scanner/scanner.go` | 18 | Scanning logic |
| Folder hierarchy | `internal/scanner/scanner.go` | 127-155 | Parent folder aggregation |
| Tag update | `internal/scanner/scanner.go` | 179 | In-memory tag update |
| Homepage | `internal/handlers/pages.go` | 106 | Root handler |
| Gallery | `internal/handlers/pages.go` | 255 | Tag gallery view |
| Breadcrumbs | `internal/handlers/pages.go` | 25 | Folder path parsing |
| Add tag API | `internal/handlers/api.go` | 33 | Single tag add |
| Batch tag API | `internal/handlers/api.go` | 81 | Bulk tag add |
| State locks | `internal/state/state.go` | 71-89 | Thread-safe access |
| File types | `internal/config/types.go` | 10-34 | Extension maps |
| Category priority | `internal/config/types.go` | 78 | Sort order |

---

## 🧠 Development Methodology

This project uses an **AI-native development methodology** combining LLMs, trunk-based development, and living documentation as operational infrastructure.

**Key aspects:**
- Natural language as development interface (replaces traditional IDEs)
- Explicit process modes (note-taking vs. code-making)
- Git history as prompt-optimized database
- System cards for instant context recovery

**For complete methodology documentation, see:**
→ **[🧠 Development Methodology.md](./🧠%20Development%20Methodology.md)**

**Quick summary:**
- **Operating modes**: Note-taking (analysis/documentation) vs. Code-making (implementation)
- **Process requirements**: Explicit mode transitions, clear communication, confirmation before implementing
- **Git workflow**: Trunk-based development with small commits and natural language messages
- **Documentation approach**: Living system cards maintained with same rigor as code
- **Token efficiency**: 3K-5K tokens for context vs. 20K-40K for codebase exploration

---

## 📖 How to Use This Document

**Purpose**: This document serves as a **technical system card** for the media server codebase.

**What this document covers:**
- Architecture and component responsibilities
- Implementation details with line references
- Known issues and performance characteristics
- Recent changes and git history context

**What other documents cover:**
- **[🧠 Development Methodology.md](./🧠%20Development%20Methodology.md)** - Process and AI-native development paradigm
- **[🌐 Open Source Vision.md](./🌐%20Open%20Source%20Vision.md)** - High-level purpose and vision
- **[NEXT_CYCLE_IMPROVEMENTS.md](./NEXT_CYCLE_IMPROVEMENTS.md)** - Current cycle's architectural improvements
- **[BUGS.md](./BUGS.md)** - Active bug tracking and testing notes

**Best practices:**
1. Load PROJECT_OVERVIEW.md at session start for technical context
2. Use line references to navigate directly to code (e.g., `HandleViewer:390`)
3. Check git history for recent changes and rationale
4. Update documentation when making significant changes
5. Commit documentation updates with code changes
6. Cross-reference documents to catch inconsistencies (see Development Methodology)

**Token efficiency:** This approach reduces context loading from 40K tokens (codebase exploration) to 5K tokens (targeted reading).

---

*This document should be loaded at the start of each session to provide context about the codebase structure and architecture.*
