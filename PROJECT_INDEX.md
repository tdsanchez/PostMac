# Project Index - Detailed Catalog

**Purpose**: Comprehensive reference for all projects in this development tree
**Last Updated**: 2025-12-12
**Projects**: 3 active

---

## Quick Navigation

| Project | Status | Language | Purpose | Documentation |
|---------|--------|----------|---------|---------------|
| [media_server](#media_server) | Production | Go | Media library manager (100k+ files) | [Overview](media_server/PROJECT_OVERVIEW.md) |
| [apfs-monitor](#apfs-monitor) | Production | Go | APFS container space monitor | [Overview](apfs-monitor/PROJECT_OVERVIEW.md) |
| [IaaDb](#iaadb) | Development | Go | Infrastructure as a Database (K8s alternative) | [Overview](IaaDb/PROJECT_OVERVIEW.md) |

---

## media_server

### Overview
**Full Name**: Media Server - Web-Based Finder Alternative
**Status**: Production-ready (168k+ files tested, architected for 5M+)
**Language**: Go
**First Commit**: 2025-12-02
**Last Update**: 2025-12-11

### Purpose
A local web server for organizing and viewing media files using macOS filesystem tags. Provides a web-based gallery interface with hierarchical folder navigation, tag management, and metadata viewing.

**The Real Problem Being Solved**:
- Human-indexing toward 5 million files for perceptual AI training
- Generating ground truth labels through manual tagging workflow
- Creating training datasets for perceptual recognition algorithms
- Breaking through Apple's artificial "10k file limit" in Finder

### Key Features
- ✅ SQLite caching (14x faster startup: 1s vs 14s for 168k files)
- ✅ Pagination (handles 168k+ files, 200/page default)
- ✅ Tag management via macOS extended attributes
- ✅ FSEvents auto-rescan (automatic cache coherence on file changes)
- ✅ FSEvents DOM auto-refresh (browser automatically reloads after scan)
- ✅ localStorage caching (eliminates template serialization bottleneck)
- ✅ Comment display and editing (plist decoding)
- ✅ Random mode slideshow
- ✅ Video playback (gallery previews and single file view)
- ✅ File deletion with Trash support
- ✅ Keyboard shortcuts for rapid tagging
- ✅ QuickLook integration
- ✅ Full OS filesystem path display

### Architecture
```
cmd/media-server/          # Main application
  ├── main.go             # Server initialization
  ├── index_template.html # Homepage (category grid)
  ├── gallery_template.html # Gallery view (file grid)
  ├── main_template.html  # Single file viewer
  └── main_template.js    # Viewer JavaScript

internal/
  ├── config/            # File type definitions
  ├── conversion/        # Format conversion (RTF, WebArchive)
  ├── handlers/          # HTTP request handlers
  ├── metadata/          # EXIF extraction
  ├── models/            # Data structures
  ├── persistence/       # Tag persistence (batched writes)
  ├── scanner/           # Directory scanning + macOS tags
  └── state/             # Global state management
```

### Performance
- **Files tested**: 168,331 (168k) files in production
- **Expanded testing**: ~350,000 files (loads in 2 seconds)
- **Startup time**: 1 second (with cache) vs 14 seconds (without)
- **Cache size**: 43MB SQLite database
- **Page load**: <1 second for any page (pagination)
- **Proof**: 350k files in 2s proves Apple's limitations are artificial

### Real-World Use Case
**ML Dataset Preparation at Scale**:
- Tag millions of files for training perceptual AI
- Generate ground truth labels through manual workflow
- Algorithms functionally inferred from 256x256 bitmaps + tag collections
- No commercial tool supports this workflow (market too small)
- **This tool exists because LLMs make personal tools viable**

### Technology Stack
- **Backend**: Go 1.21+
- **Dependencies**:
  - github.com/pkg/xattr (macOS extended attributes)
  - github.com/rwcarlsen/goexif (EXIF parsing)
  - howett.net/plist (property list parsing)
  - github.com/mattn/go-sqlite3 (SQLite driver)
  - github.com/fsnotify/fsnotify (FSEvents integration)
- **Frontend**: Vanilla JavaScript (no frameworks)
- **Storage**: SQLite for caching, filesystem for media

### Documentation Files
- `PROJECT_OVERVIEW.md` - Complete technical system card (87KB)
- `🧠 Development Methodology.md` - AI-native development process
- `🌐 Open Source Vision.md` - High-level purpose and vision
- `NEXT_CYCLE_IMPROVEMENTS.md` - Containerization analysis
- `SESSIONS.md` - Version history and session tracking
- `COLIMA_DEPLOYMENT.md` - Container deployment guide
- `MULTIPASS_DEPLOYMENT.md` - VM deployment guide
- `ansible/README.md` - Ansible deployment automation

### Scripts
- `build_server.sh` - Build binary and launch browser
- `load_test.py` - Python load testing (mixed scenarios)
- `load_test.pl` - Perl load testing (cross-language verification)
- `debug_duplicates.sh` - Debugging helper
- `git_grep.sh` - Search across git history

### Known Limitations
- No authentication (designed for local use)
- Tag write permission errors on some files (intermittent)
- QuickLook initialization issue (workaround: navigate first)
- No multi-user support

### Recent Major Changes
- **(2025-12-11)**: localStorage caching + API endpoint eliminates serialization bottleneck
- **(2025-12-10)**: FSEvents DOM auto-refresh for seamless UX
- **(2025-12-10)**: Dual navigation paths (click vs keyboard on homepage)
- **(2025-12-10)**: Random sort mode in gallery view
- **(2025-12-10)**: Click-to-file navigation in gallery
- **(2025-12-09)**: OS filesystem path display
- **(2025-12-09)**: File deletion with Trash support
- **(2025-12-08)**: SQLite caching implementation (14x speedup)
- **(2025-12-08)**: Pagination system (handles 168k+ files)
- **(2025-12-08)**: "All" category fix (shows all files, not just root)

### Future Enhancements
- [ ] Drag-and-drop file tagging
- [ ] "[B]eaming" feature (transfer to predefined directory)
- [ ] Responsive design for tablets/mobile
- [ ] Virtual scrolling for massive galleries
- [ ] Incremental scanning with FSEvents

### How to Use
```bash
cd media_server
go build -o media-server cmd/media-server/main.go
./media-server --dir=/path/to/media --port=8080
# Open browser to http://localhost:8080
```

---

## apfs-monitor

### Overview
**Full Name**: APFS Container Space Monitor
**Status**: Production (deployed 2025-12-12, tested and validated)
**Language**: Go
**First Commit**: 2025-12-12
**Last Update**: 2025-12-12

### Purpose
A lightweight Go daemon that monitors APFS container free space and sends persistent modal alerts before Time Machine snapshots fail.

**The Real Problem Being Solved**:
- APFS containers share space across multiple volumes
- macOS Finder shows misleading per-volume space
- Container exhaustion causes Time Machine failures and system instability
- Users can't see the actual container-level free space
- **Historical impact**: ~10 incidents over 6 years, ~3 hours each, ~30 hours total lost

### Key Features
- ✅ Container-level monitoring (not misleading volume reports)
- ✅ Persistent modal alerts (stay on screen until acknowledged)
- ✅ Configurable warning/critical thresholds (100GB/50GB default)
- ✅ Auto-runs at login via LaunchAgent
- ✅ Auto-restarts on crash
- ✅ Checks every 5 minutes automatically
- ✅ Comprehensive logging
- ✅ Battle-tested (137GB test file validation)

### Architecture
```
main.go                    # Single-file implementation
  ├── main()              # Entry point, flag parsing
  ├── runDaemon()         # Continuous monitoring loop
  ├── checkAndReport()    # Single check cycle
  ├── checkThresholds()   # Compare against warning/critical
  ├── getAPFSContainers() # Execute diskutil command
  ├── parseAPFSList()     # Parse diskutil output (regex)
  └── sendNotification()  # Display persistent alert

LaunchAgent Integration:
  ~/Library/LaunchAgents/com.local.apfs-monitor.plist
  /usr/local/bin/apfs-monitor (binary)
  ~/Library/Logs/apfs-monitor.log (logs)
```

### How It Works
1. Runs `diskutil apfs list` to get container information
2. Parses output to extract "Capacity Not Allocated" (true free space)
3. Compares against thresholds (100GB warning, 50GB critical)
4. Logs status every check
5. Sends persistent modal alert if threshold breached
6. Repeats alert every 5 minutes until resolved

### Why LaunchAgent (Not LaunchDaemon)?
- ✅ Can send user notifications (macOS security requirement)
- ✅ Runs at user login (acceptable for this use case)
- ✅ Has user permissions for osascript
- ❌ Root daemons cannot send notifications (macOS blocks them)

### Technology Stack
- **Language**: Go 1.21+
- **Dependencies**: None (stdlib only)
- **macOS Integration**: diskutil, osascript (AppleScript), launchctl
- **Deployment**: LaunchAgent (user-level, not root)

### Performance
- **Memory**: ~6 MB RAM
- **CPU**: Negligible (~1 second every 5 minutes)
- **Disk**: ~10 MB (binary + logs)
- **Network**: None (local-only)

### Documentation Files
- `PROJECT_OVERVIEW.md` - Complete overview with testing details
- `README.md` - User-facing tool documentation
- `QUICKSTART.md` - 30-second installation guide
- `QUICK-REFERENCE.md` - Command cheat sheet
- `MASTER-DOCUMENTATION.md` - Complete reference manual
- `SUMMARY.md` - Executive summary
- `INCIDENT-REPORT.md` - 2025-12-12 incident analysis
- `TEST-LOG.md` - Testing methodology and results
- `SYSTEM-CARD-NOTES.md` - Model behavior observations
- `CHANGELOG.md` - Version history
- `INDEX.md` - Documentation index

### Scripts
- `install.sh` - Initial installation
- `uninstall.sh` - Clean removal
- `switch-to-agent.sh` - Convert from daemon to agent
- `update-agent.sh` - Update running agent
- `test.sh` - Verify installation
- `set-thresholds-100-50.sh` - Set thresholds

### Testing & Validation
**Test Setup** (2025-12-12):
- Created 137 GB test file to reduce free space
- Reduced from 229 GB to 94 GB (below 100 GB threshold)
- Monitored automatic detection and alerting

**Results**: ✅ **ALL TESTS PASSED**
- Detection: ✅ Working (5 minutes after space reduction)
- Logging: ✅ Working (all checks logged)
- Alerts: ✅ Working (persistent modal appears)
- Persistence: ✅ Working (stays on screen until acknowledged)
- Repetition: ✅ Working (repeats every 5 minutes)
- Resolution: ✅ Working (no alerts when space restored)

### Key Decisions
| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Use Go | Fast, simple, single binary | ✅ Easy deployment |
| Container-level monitoring | Only accurate space measure | ✅ No false positives |
| LaunchAgent vs Daemon | Notifications require user context | ✅ Alerts work |
| Modal alerts vs notifications | Need persistent warnings | ✅ Cannot be ignored |
| 5-minute interval | Balance responsiveness/overhead | ✅ Good tradeoff |
| 100GB warning / 50GB critical | Based on 1TB container | ✅ Adequate warning |

### Development History
**Timeline**:
- 2019-2024: Manual vigilance (failed ~10 times, 30+ hours lost)
- 2025-12-12 morning: APFS crisis (210 GB Music folder move)
- 2025-12-12 afternoon: Solution developed (3-hour session)
- 2025-12-12 evening: Deployed and validated

**Iterations**:
1. Initial daemon (root) - detection works, notifications blocked
2. Switch to user agent - transient notifications too brief
3. Persistent alerts - **final working solution**

### Configuration
**Current Settings**:
```
Warning Threshold:  100 GB free
Critical Threshold: 50 GB free
Check Interval:     5 minutes
Notifications:      Enabled (persistent modal alerts)
Log Location:       ~/Library/Logs/apfs-monitor.log
```

**Recommended Thresholds by Container Size**:
| Container | Warning | Critical | Rationale |
|-----------|---------|----------|-----------|
| 500 GB | 40 GB | 20 GB | 8% / 4% buffer |
| 1 TB | 100 GB | 50 GB | 10% / 5% buffer |
| 2 TB | 150 GB | 75 GB | 7.5% / 3.75% buffer |
| 4 TB | 200 GB | 100 GB | 5% / 2.5% buffer |

### How to Use
```bash
cd apfs-monitor
./test.sh        # Verify it works
bash switch-to-agent.sh  # Install as user agent

# Check status
launchctl list | grep apfs-monitor

# View logs
tail -f ~/Library/Logs/apfs-monitor.log

# Manual check
/usr/local/bin/apfs-monitor
```

### ROI Analysis
**Development Cost**: 3 hours (one-time)
**Prevented Future Incidents**: ~10 over next 6 years
**Time Saved**: ~30 hours (10 × 3 hours per incident)
**ROI**: 10x in time alone, infinite in stress reduction

---

## IaaDb

### Overview
**Full Name**: Infrastructure as a Database
**Status**: Development (PoC validation phase)
**Language**: Go
**First Commit**: 2025-12-12
**Last Update**: 2025-12-12

### Purpose
A Kubernetes alternative that uses SQL databases for state management and systemd for process orchestration, eliminating the need for container abstractions when deploying statically-compiled binaries.

**The Real Problem Being Solved**:
- Kubernetes is over-engineered for most workloads
- DBAs becoming "data scientists" need infrastructure that matches their mental model (tables, SQL)
- etcd is just a key-value store — use actual SQL with ACID guarantees
- Containers became popular because PMs mistook them for VMs, not because they solve the right problem
- Need deterministic, auditable infrastructure with every state change as a transaction

### Key Features
- ✅ HCL → SQL transformation (Python PoC validated)
- ✅ 2NF normalized database schema (resources, attributes, variables, outputs, dependencies)
- ✅ Go implementation (working)
- ✅ SQLite backend (working)
- ⏳ Systemd integration (planned)
- ⏳ Orchestration daemon (planned)
- ⏳ Service discovery (planned)
- ⏳ Web UI with dual interface (planned)

### Architecture
```
K8s Concept          IaaDb Equivalent
-----------          ----------------
etcd                 Postgres/Oracle/MySQL/SQLite
Desired state YAML   Rows in deployments table
Controllers          Systemd units + orchestrator
kubectl              SQL queries or thin CLI
Pods                 Go binary processes
```

### Technology Stack
- **Backend**: Go 1.21+
- **Dependencies**:
  - github.com/hashicorp/hcl/v2 (HCL parsing)
  - github.com/mattn/go-sqlite3 (SQLite driver)
  - github.com/lib/pq (Postgres, planned)
  - github.com/go-sql-driver/mysql (MySQL, planned)
- **Database**: SQLite (initial), Postgres/MySQL/Oracle (planned)

### Documentation Files
- `PROJECT_OVERVIEW.md` - Complete technical system card
- `README.md` - User-facing documentation
- `examples/simple.tf` - Example HCL file

### Implementation Status

**Completed**:
- [x] Conceptual validation (stream of consciousness captured)
- [x] Python PoC (Flask service)
- [x] C implementation (flex/bison parser)
- [x] Go module structure
- [x] HCL parser (using hashicorp/hcl)
- [x] SQLite backend
- [x] CLI tool (hcl2sql)
- [x] 2NF normalized schema

**In Progress**:
- [ ] Improved attribute value extraction (currently stores raw HCL structures)
- [ ] Support for nested objects and lists
- [ ] Reference detection (var.foo, resource.bar.id)

**Planned**:
- [ ] Bidirectional transformation (SQL → HCL)
- [ ] Postgres/MySQL support
- [ ] Systemd integration
- [ ] Orchestrator daemon with reconciliation loop
- [ ] Service discovery and routing
- [ ] Web UI (dual interface: K8s-style + SQL query interface)

### How to Use
```bash
cd IaaDb

# Build
go build -o hcl2sql cmd/hcl2sql/main.go

# Convert HCL to SQLite
./hcl2sql --hcl examples/simple.tf --db infrastructure.db

# Query infrastructure
sqlite3 infrastructure.db "SELECT * FROM terraform_resources;"
sqlite3 infrastructure.db "
  SELECT r.resource_type, r.resource_name, a.attribute_name, a.attribute_value
  FROM terraform_resources r
  JOIN terraform_resource_attributes a ON r.resource_id = a.resource_id;
"
```

### Target Market
- **Oracle Cloud Infrastructure customers**: DBAs who know PL/SQL but find YAML mystifying
- **Database Administrators**: Understand transactions, constraints, ACID guarantees
- **Platform Engineers**: Value simplicity over complexity, need auditability
- **Teams migrating to cloud**: Don't want to force learning Kubernetes

### Philosophy
**Containers Solve the Wrong Problem**:
- The value was reproducible artifacts (packaging)
- Not isolation (namespaces/cgroups handle that)
- For statically-compiled Go binaries: unnecessary overhead

**etcd Is a Database Pretending Not To Be**:
- Key-value store with watch semantics
- SQL provides transactions, constraints, triggers
- DBAs already understand this model

**Dual Interface Strategy**:
- Give K8s people their abstractions
- Give DB people their tables
- Same system, different lenses

### Database Schema (2NF)
```sql
terraform_resources              -- Resource metadata
terraform_resource_attributes    -- Flattened attributes
terraform_variables              -- Variable definitions
terraform_outputs                -- Output definitions
terraform_dependencies           -- Resource dependencies
```

### Known Challenges
- **HCL Expression Evaluation**: Terraform expressions need evaluation context
- **State Locking**: Need distributed locking for multi-node orchestration
- **Provider Plugins**: Initial focus on state representation, not execution
- **Market Education**: "Why not just use Kubernetes?" objection

### Development History
**Timeline**:
- 2025-12-12: Concept validation, Python PoC, C implementation
- 2025-12-12: Go implementation started
- 2025-12-12: Working HCL → SQLite transformation

**Iterations**:
1. Python PoC - Flask service with python-hcl2
2. C implementation - flex/bison parser (no JSON)
3. Go implementation - Production target with hashicorp/hcl

---

## Cross-Project Patterns

### Common Technology
- **Language**: Go across all projects
- **macOS Integration**: xattr, osascript, diskutil, launchctl
- **Deployment**: User-level agents (not root)
- **Documentation**: Comprehensive system cards

### Common Principles
- **Automation over vigilance**: Computers don't forget
- **Prevention over recovery**: Stop crises before they happen
- **Simplicity over complexity**: Single binaries, minimal dependencies
- **Users over vendors**: Build what you need, not what sells

### Shared Methodology
- **AI-native development**: LLM-assisted, conversation-driven
- **Trunk-based git**: Linear history, small commits
- **Living documentation**: System cards updated with code
- **Explicit modes**: Note-taking vs. code-making

---

## Project Comparison

| Aspect | media_server | apfs-monitor | IaaDb |
|--------|--------------|--------------|-------|
| **Complexity** | High (full-stack web app) | Low (single daemon) | High (distributed system) |
| **LOC** | ~3,000+ lines | ~500 lines | ~500+ lines (initial) |
| **Dependencies** | 5 external libraries | 0 (stdlib only) | 2 (hcl, sqlite3) |
| **Development Time** | Multiple sessions over weeks | Single 3-hour session | Initial PoC in 1 day |
| **Files Managed** | 168k+ media files | N/A (monitors APFS) | Infrastructure state |
| **Problem Space** | Apple's 10k file limit | APFS space visibility | K8s complexity |
| **Proof of Concept** | 350k files in 2s | Container monitoring works | HCL → SQL works |
| **ROI** | Enables ML work (invaluable) | 10x time saved | Potential consultancy |

---

## Development Statistics

### Total Code
- **Go**: ~3,500+ lines across both projects
- **JavaScript**: ~1,000+ lines (media_server only)
- **HTML**: ~1,500+ lines (media_server templates)
- **Shell Scripts**: ~500+ lines (both projects)

### Documentation
- **Total Documentation**: ~200+ KB across all markdown files
- **System Cards**: 2 comprehensive PROJECT_OVERVIEW.md files
- **Supporting Docs**: ~30+ markdown files
- **Git Commits**: ~50+ commits (media_server), ~10+ commits (apfs-monitor)

### Testing
- **Manual Testing**: Extensive (both projects)
- **Load Testing**: Stress-tested (media_server: 5 instances, 20 req/sec)
- **Scale Testing**: 350k files (media_server), 137GB test file (apfs-monitor)
- **Validation**: Both projects battle-tested in production

---

## Future Projects (Potential)

Ideas for additional tools in this tree:

### High Priority
- [ ] **filesystem-differ** - Compare two directory trees for sync/backup validation
- [ ] **photo-deduplicator** - Find and merge duplicate photos across libraries
- [ ] **tag-migrator** - Migrate macOS tags between filesystems/volumes

### Medium Priority
- [ ] **spotlight-indexer** - Custom Spotlight metadata importer for obscure formats
- [ ] **time-machine-optimizer** - Intelligent snapshot management and space recovery
- [ ] **bundle-analyzer** - Analyze macOS bundles (.app, .photoslibrary) for space usage

### Low Priority
- [ ] **clipboard-manager** - Persistent clipboard history with search
- [ ] **window-manager** - Keyboard-driven window management
- [ ] **notification-aggregator** - Consolidate and manage macOS notifications

---

## How to Add a New Project

1. **Create project directory**:
   ```bash
   mkdir /Users/tdsanchez/dev/new-project
   cd /Users/tdsanchez/dev/new-project
   ```

2. **Initialize with documentation**:
   - Create `PROJECT_OVERVIEW.md` (use media_server as template)
   - Create `README.md` (user-facing)
   - Create `QUICKSTART.md` (if applicable)

3. **Set up development**:
   - Initialize git: `git init`
   - Create initial structure
   - First commit with project skeleton

4. **Update dev-tree documentation**:
   - Add project to this `PROJECT_INDEX.md`
   - Update `/Users/tdsanchez/dev/README.md`
   - Document in appropriate sections

5. **Follow methodology**:
   - Use explicit operating modes
   - Maintain system card documentation
   - Use trunk-based development
   - Keep documentation synchronized with code

---

## Support & Maintenance

### For Issues
- Document issue in project's PROJECT_OVERVIEW.md "Known Issues" section
- Include: symptom, when discovered, context, potential causes, investigation needed
- Track in git commits when resolved

### For Enhancements
- Document in "Future Enhancements" section of PROJECT_OVERVIEW.md
- Prioritize: High / Medium / Low
- Include rationale and potential implementation approach

### For Questions
- Check PROJECT_OVERVIEW.md first (most comprehensive)
- Check git history for context on specific changes
- Review SESSIONS.md for development timeline (if exists)

---

## Acknowledgments

**Philosophy**: Computers should manage complexity, not users
**Methodology**: AI-native development with comprehensive system cards
**Tools**: Claude Code (Anthropic), Go, macOS native tools
**Result**: Sophisticated personal tools solving real problems

**Developed**: 2025 (media_server started 2025-12-02, apfs-monitor 2025-12-12)
**Lines of Code**: 3,500+ Go, 1,000+ JavaScript, 1,500+ HTML
**Documentation**: 200+ KB of living system cards
**Hours Saved**: 30+ (and counting)
**ML Training Dataset**: On track toward 5M tagged files

---

*When personal tools are trivial to build, ecosystem lock-in loses its power.*

---

**Last Updated**: 2025-12-12
**Active Projects**: 2
**Planned Projects**: 8+
**Methodology**: AI-native development
