# TiddlyWiki Integration - Architectural Design

> **Purpose**: Design media-server as TiddlyWiki backend for blog publishing from Gallery view
> **Status**: 🔵 Planning phase - architectural exploration
> **Last Updated**: 2025-12-13
> **Inspiration**: TiddlyDark macOS app, AI-assisted tag sync PoC

---

## Vision

Transform media-server into a **blog publishing platform** by:
1. Backend TiddlyWiki with media-server's existing infrastructure
2. Enable blog post creation from Gallery image selections
3. Leverage existing tag system (macOS tags ↔ media-server ↔ blog tags)
4. Preserve TiddlyWiki's editing power while adding photo-centric workflow

**Key insight**: Media-server already has all the pieces (file storage, SQLite, tags, HTTP server) - just need to add tiddler storage and TW-compatible API.

---

## Architectural Approach: Hybrid (Option C)

### Why Hybrid?

**Instead of**:
- ❌ Pure HTML TiddlyWiki (awkward persistence, hard to automate)
- ❌ Separate Node.js TW server (two servers, complex deployment)

**We build**:
- ✅ Media-server becomes TiddlyWiki backend
- ✅ Single Go server handles everything
- ✅ Unified data model (files, tags, blog posts)
- ✅ Can export to standard TW format for portability

---

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Media-Server (Go)                        │
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │
│  │   Gallery    │  │    Viewer    │  │     Blog     │    │
│  │   (Existing) │  │  (Existing)  │  │     (NEW)    │    │
│  └──────┬───────┘  └──────────────┘  └──────┬───────┘    │
│         │                                     │             │
│         │  Select images                     │             │
│         └────────────────┬──────────────────┘             │
│                          ▼                                  │
│              ┌────────────────────────┐                    │
│              │  Blog Post Creator     │                    │
│              │  - Pre-populate images │                    │
│              │  - Inherit macOS tags  │                    │
│              │  - Open TW editor      │                    │
│              └────────────────────────┘                    │
│                          │                                  │
│         ┌────────────────┴────────────────┐               │
│         ▼                                  ▼                │
│  ┌─────────────────┐              ┌─────────────────┐    │
│  │  SQLite Cache   │              │   TW API Layer  │    │
│  │  (Existing)     │              │   (NEW)         │    │
│  │                 │              │                 │    │
│  │  - files        │              │  /api/tiddlers/ │    │
│  │  - tags         │              │  /api/blog/*    │    │
│  │  - scan_meta    │              │                 │    │
│  │  - blog_posts ←─┼──────────────┤  Storage:       │    │
│  │    (NEW)        │              │  - SQLite table │    │
│  └─────────────────┘              │  - JSON files   │    │
│                                    │  - Hybrid       │    │
│                                    └─────────────────┘    │
│                          │                                  │
│                          ▼                                  │
│              ┌────────────────────────┐                    │
│              │   TiddlyWiki HTML      │                    │
│              │   Modified to use      │                    │
│              │   media-server API     │                    │
│              └────────────────────────┘                    │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## Data Model: Blog Posts as Tiddlers

### SQLite Schema (New Table)

```sql
-- Blog posts stored as TiddlyWiki-compatible tiddlers
CREATE TABLE blog_posts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    text TEXT,                    -- Markdown/TW markup content
    tags TEXT,                    -- Space-separated tag list (TW format)
    created INTEGER NOT NULL,     -- Unix timestamp (milliseconds)
    modified INTEGER NOT NULL,    -- Unix timestamp (milliseconds)
    creator TEXT,                 -- Author (optional)
    modifier TEXT,                -- Last editor (optional)
    type TEXT DEFAULT 'text/vnd.tiddlywiki',  -- Content type
    fields TEXT,                  -- JSON for custom fields
    image_refs TEXT,              -- JSON array of image paths
    published BOOLEAN DEFAULT 0,  -- Publish status
    UNIQUE(title)
);

-- Index for efficient queries
CREATE INDEX idx_blog_posts_tags ON blog_posts(tags);
CREATE INDEX idx_blog_posts_created ON blog_posts(created DESC);
CREATE INDEX idx_blog_posts_published ON blog_posts(published);
```

### Go Struct (internal/models/models.go)

```go
// BlogPost represents a TiddlyWiki tiddler stored in media-server
type BlogPost struct {
    ID         int64     `json:"id"`
    Title      string    `json:"title"`
    Text       string    `json:"text"`
    Tags       string    `json:"tags"`      // Space-separated
    Created    int64     `json:"created"`   // Milliseconds since epoch
    Modified   int64     `json:"modified"`  // Milliseconds since epoch
    Creator    string    `json:"creator,omitempty"`
    Modifier   string    `json:"modifier,omitempty"`
    Type       string    `json:"type"`
    Fields     string    `json:"fields,omitempty"`  // JSON string
    ImageRefs  []string  `json:"image_refs"`        // Parsed from JSON
    Published  bool      `json:"published"`
}

// TiddlerJSON is TiddlyWiki-compatible JSON format
type TiddlerJSON struct {
    Title    string            `json:"title"`
    Text     string            `json:"text"`
    Tags     string            `json:"tags"`
    Created  string            `json:"created"`  // TW uses string timestamps
    Modified string            `json:"modified"`
    Type     string            `json:"type"`
    Fields   map[string]string `json:"fields,omitempty"`
}
```

### Image Reference Format

**In blog post text (Markdown/TW syntax)**:
```markdown
Here's my photo blog post!

[img[/file/path/to/image1.jpg]]
[img[/file/path/to/image2.jpg]]

Or using markdown:
![](http://localhost:8080/file/path/to/image.jpg)
```

**In image_refs JSON field** (for metadata):
```json
[
  "/Volumes/External/media/photos/IMG_1234.jpg",
  "/Volumes/External/media/photos/IMG_5678.jpg"
]
```

---

## API Design: TiddlyWiki-Compatible Endpoints

### Core CRUD Operations

```
GET    /api/tiddlers              → List all tiddlers (blog posts)
GET    /api/tiddlers/:title       → Get specific tiddler
PUT    /api/tiddlers/:title       → Create/update tiddler
DELETE /api/tiddlers/:title       → Delete tiddler
```

### Media-Server Specific Extensions

```
POST   /api/blog/create           → Create blog post from Gallery selection
GET    /api/blog/posts            → List blog posts with pagination
GET    /api/blog/post/:id         → Get blog post by ID
POST   /api/blog/publish/:id      → Mark post as published
POST   /api/blog/export/:id       → Export to static HTML/TW JSON
GET    /api/blog/tags             → Get all blog tags (complement to /api/alltags)
```

### Gallery Integration

```
POST   /api/gallery/create-post   → Create blog post from selected images
  Request body:
  {
    "image_paths": ["/path/1.jpg", "/path/2.jpg"],
    "title": "My Photo Blog Post",
    "tags": ["blog", "photography", "2025"],
    "initial_text": "Optional caption..."
  }

  Response:
  {
    "post_id": 42,
    "title": "My Photo Blog Post",
    "edit_url": "/blog/edit/42"
  }
```

---

## User Workflows

### Workflow 1: Create Blog Post from Gallery

**User journey**:
1. **Gallery View** - Browse photos, find interesting set
2. **Multi-select** - Cmd+Click to select images (existing functionality)
3. **Keyboard shortcut** - Press 'B' for "Blog" (new)
   - OR click "Create Blog Post" button
4. **Modal appears**:
   ```
   ┌─────────────────────────────────────────┐
   │  Create Blog Post                       │
   ├─────────────────────────────────────────┤
   │  Title: [________________________]      │
   │                                         │
   │  Tags:  [photography] [vacation]       │
   │         (inherited from image tags)     │
   │         + Add tag                       │
   │                                         │
   │  Images: 5 selected                     │
   │  [x] IMG_1234.jpg                       │
   │  [x] IMG_5678.jpg                       │
   │  ...                                    │
   │                                         │
   │  [Cancel]  [Create & Edit]              │
   └─────────────────────────────────────────┘
   ```
5. **Create** - POST to `/api/blog/create`
6. **Redirect** - Opens blog editor at `/blog/edit/:id`

### Workflow 2: Edit Blog Post in TiddlyWiki Interface

**After creation**:
1. **TW Editor loads** at `/blog/edit/:id`
2. **Pre-populated content**:
   - Title from user input
   - Images embedded as TW image syntax
   - Tags inherited from macOS tags
   - Empty text area for narrative
3. **Edit with TW features**:
   - Rich text / markdown editing
   - WikiLinks: `[[Other Post]]`
   - Transclusion: `{{!!field}}`
   - Macros and widgets
4. **Save** - Auto-save via `/api/tiddlers/:title` (PUT)
5. **Preview** - Live preview in TW interface

### Workflow 3: Publish Blog Post

**Publishing options**:

**Option A: Static HTML Export**
```
User clicks "Publish" → Export modal:
┌─────────────────────────────────────────┐
│  Publish Blog Post                      │
├─────────────────────────────────────────┤
│  Format:                                │
│  ○ Single HTML file (portable)          │
│  ○ Multi-file (HTML + images)           │
│  ○ TiddlyWiki JSON (import elsewhere)   │
│                                         │
│  Include images:                        │
│  ○ Embed as base64 (bloats file)        │
│  ○ Copy to export directory             │
│  ○ Reference by URL (requires server)   │
│                                         │
│  Export to: [~/Desktop/blog-export/]    │
│                                         │
│  [Cancel]  [Export]                     │
└─────────────────────────────────────────┘
```

**Option B: Publish to Server Directory**
```bash
# Publishes to configured publish directory
# Structure:
/publish/
  ├── index.html           (Blog index page)
  ├── posts/
  │   ├── my-post-1.html
  │   ├── my-post-2.html
  │   └── ...
  └── images/
      ├── IMG_1234.jpg
      └── ...

# Can be served by Nginx, S3, GitHub Pages, etc.
```

**Option C: Mark as Published (Keep in Media-Server)**
- Sets `published = true` in database
- Post appears in public blog feed at `/blog`
- Still editable, unpublish option available

---

## TiddlyWiki Integration: Technical Details

### Modified TiddlyWiki HTML

**Standard TiddlyWiki**:
- Self-contained HTML file
- Data stored in HTML `<div>` elements
- Saves by rewriting entire HTML file

**Media-Server TiddlyWiki**:
- HTML shell served by media-server
- Data loaded from `/api/tiddlers` endpoint
- Saves via PUT to `/api/tiddlers/:title`
- Images served by media-server `/file/*` route

**Implementation**:
```html
<!-- Served at /blog or /blog/edit/:id -->
<!DOCTYPE html>
<html>
<head>
  <title>Media-Server Blog</title>
  <!-- TiddlyWiki core JS/CSS -->
  <script src="/static/tiddlywiki-core.js"></script>
</head>
<body>
  <script>
    // Configure TW to use media-server API
    $tw.wiki.addEventListener("th-saving-tiddler", function(event) {
      const tiddler = event.tiddler;
      fetch(`/api/tiddlers/${encodeURIComponent(tiddler.fields.title)}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(tiddler.fields)
      });
    });

    // Load tiddlers from media-server
    fetch('/api/tiddlers')
      .then(r => r.json())
      .then(tiddlers => {
        tiddlers.forEach(t => $tw.wiki.addTiddler(t));
        $tw.boot.startup();
      });
  </script>
</body>
</html>
```

### Tag Synchronization: Three-Way Sync

**Current**: macOS tags ↔ media-server (working)

**New**: macOS tags ↔ media-server ↔ blog post tags

**Synchronization rules**:
1. **Image → Blog Post** (creation):
   - Read macOS tags from selected images
   - Pre-populate blog post tags
   - User can add/remove tags in blog editor

2. **Blog Post → Images** (optional, user choice):
   - After editing blog post tags
   - Offer to sync tags back to images
   - Applies tags to all images referenced in post

3. **Round-trip** (continuous sync):
   - Changes to image tags (in Gallery) can update blog posts
   - Changes to blog tags can update images
   - Conflict resolution: User chooses priority

**Implementation strategy**:
- Start simple: One-way (images → blog) at creation time
- Add bi-directional sync in v2
- Make sync optional/user-controlled (avoid tag chaos)

---

## Gallery View: UI Changes

### New Button: "Create Blog Post"

**Location**: Gallery header, next to existing buttons

**Current header**:
```
┌────────────────────────────────────────────────────────┐
│ [≡] Category: All  |  Showing 1-200 of 168331 files   │
│ [Rescan] [Sort: Name ▼]                                │
└────────────────────────────────────────────────────────┘
```

**Updated header**:
```
┌────────────────────────────────────────────────────────┐
│ [≡] Category: All  |  Showing 1-200 of 168331 files   │
│ [Rescan] [Sort: Name ▼] [📝 Create Blog Post]         │
│                          (disabled if no selection)    │
└────────────────────────────────────────────────────────┘
```

### Selection Count Indicator

**When images selected**:
```
┌────────────────────────────────────────────────────────┐
│ [≡] Category: All  |  5 images selected                │
│ [Rescan] [Sort: Name ▼] [📝 Create Blog Post (5)]     │
└────────────────────────────────────────────────────────┘
```

### Keyboard Shortcut: 'B' for Blog

**Add to existing shortcuts** (gallery_template.html):
```javascript
// Existing: T (tag), E (edit comment), S (sort), etc.
// New:
case 'b':
case 'B':
  if (selectedItems.length > 0) {
    createBlogPostFromSelection();
  } else {
    showNotification('Select images first (Cmd+Click)');
  }
  break;
```

---

## Implementation Plan: Incremental Rollout

### Phase 1: Core Blog Storage (v1 - Minimal Viable)

**Goals**:
- Store blog posts in SQLite
- Basic CRUD API
- Simple creation from Gallery
- Minimal editor (textarea, not TW yet)

**Deliverables**:
- [ ] SQLite schema: `blog_posts` table
- [ ] Go structs: `BlogPost` model
- [ ] API endpoints: `/api/blog/*` basic CRUD
- [ ] Gallery UI: "Create Blog Post" button + modal
- [ ] Simple editor: `/blog/edit/:id` with textarea
- [ ] List view: `/blog` shows all posts

**Testing criteria**:
- Can create blog post from Gallery selection
- Images referenced correctly
- Tags inherited from macOS tags
- Can edit and save posts
- Posts persist across server restarts

**Estimated effort**: 1-2 development sessions

---

### Phase 2: TiddlyWiki Integration (v2 - Rich Editing)

**Goals**:
- Replace textarea with TiddlyWiki editor
- TW-compatible API
- Export to TW JSON format

**Deliverables**:
- [ ] TiddlyWiki-compatible API: `/api/tiddlers/*`
- [ ] Modified TW HTML served at `/blog`
- [ ] TW editor integrated (load/save via API)
- [ ] Export function: TW JSON format
- [ ] Import function: Load existing TW files

**Testing criteria**:
- Can edit with TW rich text features
- WikiLinks work between posts
- Export produces valid TW JSON
- Can import into standard TiddlyWiki Node

**Estimated effort**: 2-3 development sessions

---

### Phase 3: Publishing & Export (v3 - Production Ready)

**Goals**:
- Export to static HTML
- Publish workflow
- Image handling options

**Deliverables**:
- [ ] Static HTML export (single file)
- [ ] Multi-file export (HTML + images)
- [ ] Base64 image embedding option
- [ ] Publish directory configuration
- [ ] RSS feed generation (optional)
- [ ] Public blog view (separate from editor)

**Testing criteria**:
- Exported HTML works standalone
- Images display correctly
- Can host on GitHub Pages / S3
- Blog index page lists all published posts

**Estimated effort**: 2-3 development sessions

---

### Phase 4: Advanced Features (v4 - Power User)

**Goals**:
- Bi-directional tag sync
- TiddlyDark-style drop zone
- Advanced workflows

**Deliverables**:
- [ ] Drag-and-drop images into blog editor
- [ ] Tag sync: blog → images (optional, user-controlled)
- [ ] Batch operations (publish multiple posts)
- [ ] Blog templates (reusable post structures)
- [ ] Media library integration (browse media-server from blog)
- [ ] Version history (git-backed or SQLite)

**Testing criteria**:
- Drop zone works like TiddlyDark
- Tag changes propagate correctly
- Templates speed up post creation

**Estimated effort**: 3-4 development sessions

---

## Technical Considerations

### Image Storage Strategy

**Options**:

1. **Reference by URL** (recommended for v1):
   - Blog posts contain URLs: `http://localhost:8080/file/path/to/image.jpg`
   - Images stay in original location (media library)
   - Export copies images to publish directory
   - **Pro**: No duplication, canonical source
   - **Con**: Exported HTML needs image bundling

2. **Copy to blog directory**:
   - On blog post creation, copy images to `/blog/images/`
   - Blog posts reference local copies
   - **Pro**: Blog posts self-contained
   - **Con**: Duplication, storage waste

3. **Hybrid** (best of both):
   - Development: Reference original URLs
   - Publishing: Copy images to export directory
   - **Pro**: Best UX for both modes
   - **Con**: More complex logic

**Recommendation**: Start with #1 (reference), add #3 (hybrid) in Phase 3.

---

### TiddlyWiki Format Compatibility

**TiddlyWiki JSON tiddler format**:
```json
{
  "title": "My Blog Post",
  "text": "Content goes here\n\n[img[images/photo.jpg]]",
  "tags": "blog photography travel",
  "created": "20251213120000000",
  "modified": "20251213123000000",
  "type": "text/vnd.tiddlywiki",
  "custom-field": "value"
}
```

**Timestamp format**: `YYYYMMDDhhmmssSSS` (17 digits, millisecond precision)

**Conversion needed**:
```go
// Unix timestamp (seconds) → TW timestamp string
func unixToTWTimestamp(unix int64) string {
    t := time.Unix(unix, 0)
    return t.Format("20060102150405000")
}

// TW timestamp string → Unix timestamp
func twTimestampToUnix(tw string) int64 {
    t, _ := time.Parse("20060102150405000", tw)
    return t.Unix()
}
```

**Tag format**: Space-separated string (not array!)
```
"tags": "tag1 tag2 tag3"  // Correct
"tags": ["tag1", "tag2"]  // Wrong - need to join
```

---

### Performance Considerations

**Blog post count scaling**:
- SQLite handles millions of rows efficiently
- Pagination in list view (like Gallery: 200/page)
- Indexes on `created`, `tags`, `published` for fast queries

**Image loading**:
- Gallery already handles 350k+ files
- Blog view should paginate posts (not images)
- Full-size images lazy-loaded in editor

**Export performance**:
- Static HTML generation should be background job
- Progress indicator for large exports
- Cache exported HTML (regenerate only on change)

---

## Security & Privacy Considerations

### Published vs Draft

**Draft posts**:
- `published = false`
- Only accessible at `/blog/edit/:id` (authenticated?)
- Not listed in public blog index
- Not exported in public HTML

**Published posts**:
- `published = true`
- Visible at `/blog` public index
- Included in exports
- Can be unpublished (revert to draft)

**Authentication** (future consideration):
- Currently media-server has no auth (local use)
- Blog publishing may require simple auth
- Options: Basic auth, API token, macOS keychain integration

### Path Sanitization (Blue → Green)

**Blog posts contain image paths**:
```markdown
[img[/Volumes/Terminator/media/photo.jpg]]  ← Blue (personal path)
[img[/Volumes/External/media/photo.jpg]]     ← Green (sanitized)
```

**Pre-release checklist extension**:
- [ ] Scan blog post `text` field for personal paths
- [ ] Scan blog post `image_refs` field
- [ ] Apply sanitization map to all blog content
- [ ] Test exported HTML has no personal identifiers

**Implementation**:
```sql
-- Find blog posts with personal paths
SELECT id, title, text
FROM blog_posts
WHERE text LIKE '%/Volumes/Terminator%'
   OR text LIKE '%/Users/tdsanchez%';

-- Sanitize (run before Green export)
UPDATE blog_posts
SET text = REPLACE(text, '/Volumes/Terminator/', '/Volumes/External/media/'),
    image_refs = REPLACE(image_refs, '/Volumes/Terminator/', '/Volumes/External/media/')
WHERE /* ... */;
```

---

## Open Questions & Design Decisions

### Q1: Blog Post Title Uniqueness

**Question**: Should blog post titles be unique (like TiddlyWiki)?

**Options**:
- **A**: Unique titles (TW standard) - easier for WikiLinks
- **B**: Allow duplicates, use ID - more flexible
- **C**: Unique per date (slug + date)

**Recommendation**: Start with A (unique), can relax later.

---

### Q2: Editor Choice

**Question**: Full TiddlyWiki or custom lightweight editor?

**Options**:
- **A**: Full TW embedded - maximum power, steeper learning curve
- **B**: Markdown editor (CodeMirror, etc.) - simpler, less features
- **C**: Start with B, upgrade to A in v2

**Recommendation**: C - validate workflow with simple editor first.

---

### Q3: Multi-Author Support

**Question**: Support multiple authors (creator/modifier fields)?

**Options**:
- **A**: Single user (creator = "media-server") - simpler
- **B**: Multi-user with auth - future-proofing
- **C**: Optional creator field (user can set name)

**Recommendation**: A for v1, can add auth layer later.

---

### Q4: Git Integration

**Question**: Version blog posts with git?

**Options**:
- **A**: No git - just SQLite - simpler
- **B**: Each post as file, git tracked - version history
- **C**: Export to git on publish - hybrid

**Recommendation**: A for v1, B would be powerful for v4.

---

## Success Criteria

**Phase 1 (v1) is successful when**:
- ✅ Can select images in Gallery
- ✅ Create blog post with one click
- ✅ Edit post with title, text, tags
- ✅ Images appear in post correctly
- ✅ Tags inherited from macOS tags
- ✅ Posts persist across restarts
- ✅ Can view all posts in list

**Full integration (v4) is successful when**:
- ✅ Blog workflow feels natural (keyboard-driven)
- ✅ TiddlyWiki editing power fully available
- ✅ Export to static HTML works flawlessly
- ✅ Can publish to GitHub Pages / blog host
- ✅ Tag sync bidirectional (optional)
- ✅ Drop zone works like TiddlyDark
- ✅ Documented well enough for others to replicate

---

## Related Documentation

- **[PROJECT_OVERVIEW.md](./PROJECT_OVERVIEW.md)** - Media-server architecture
- **[🧠 Development Methodology.md](./🧠%20Development%20Methodology.md)** - AI-native development process
- **[BLUE_GREEN_WORKFLOW.md](./BLUE_GREEN_WORKFLOW.md)** - Release workflow (blog posts need sanitization)
- **[PRE_RELEASE_CHECKLIST.md](./PRE_RELEASE_CHECKLIST.md)** - Add blog post path scrubbing

---

## Next Steps

### Immediate (This Session)

1. **Review this plan** - Does architecture make sense?
2. **Decide on scope** - Start with Phase 1 (v1)?
3. **Design decisions** - Answer open questions above
4. **Commit this doc** - Add to Blue environment system cards

### Development (Next Sessions)

**If proceeding with Phase 1**:
1. Create SQLite schema migration
2. Add BlogPost model and handlers
3. Implement basic CRUD API
4. Add Gallery UI (button + modal)
5. Build simple editor view
6. Test end-to-end workflow

**Timeline estimate**: 1-2 sessions for v1, 6-10 sessions for full v4.

---

*This document is part of the Blue environment system card library. Architectural planning for TiddlyWiki integration inspired by TiddlyDark and AI-assisted tag sync PoC.*

**Status**: 🔵 Planning - awaiting design decisions and implementation approval
**Last Updated**: 2025-12-13
