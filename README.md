# Development Projects - Personal Tool Suite

**Philosophy**: Build sophisticated personal tools that solve real problems which no commercial product would address.

**Development Approach**: AI-native development using LLMs, trunk-based git workflow, and living documentation as operational infrastructure.

**tl;dr**: It's more like a gist than a repo.

**Last Updated**: 2025-17-12

---

## Active Projects

### 1. media_server
**Purpose**: Web-based media library manager for organizing and tagging 100k+ files
**Status**: Production-ready (168k+ files tested, architected for 5M+)
**Language**: Go
**Key Features**:
- APFS container space monitor
- Persistent modal alerts for space warnings
- Auto-runs at login via LaunchAgent
- Battle-tested (prevents recurring 3-hour crises)

**Documentation**: See `media_server/PROJECT_OVERVIEW.md`

### 2. apfs-monitor
**Purpose**: Monitor APFS container space and prevent Time Machine snapshot failures
**Status**: Production (deployed 2025-12-12, tested and validated)
**Language**: Go
**Key Features**:
- SQLite caching (14x faster startup)
- Pagination (handles 168k+ files)
- Tag management via macOS extended attributes
- FSEvents auto-rescan for cache coherence
- localStorage optimization for instant page loads

**Documentation**: See `apfs-monitor/PROJECT_OVERVIEW.md`

---

## Common Themes

### Problems These Tools Solve

**Apple's Artificial Limitations**:
- **Finder**: Chokes at ~10k files (claimed "APFS limitations")
- **APFS Containers**: Hide true space availability from users
- **Photos.app**: Doesn't scale to millions of files
- **macOS Tools**: Optimize for consumers, not power users

**Proof of Concept**: These tools demonstrate Apple's limitations are **policy, not technical**:
- `media_server` loads 350k files in 2 seconds on same Mac hardware Apple claims "can't handle" large libraries
- `apfs-monitor` exposes container-level space data that macOS deliberately hides from Finder

**Result**: Breaking out of vendor lock-in, building tools that actually work at scale.

### Development Methodology

Both projects use **AI-native development**:

**Core Principles**:
- Natural language as development interface (replaces traditional IDEs)
- Explicit process modes (note-taking vs. code-making)
- Git history as prompt-optimized database
- System cards for instant context recovery
- Living documentation maintained with same rigor as code

**Why This Works**:
- Development time: Hours of conversational sessions vs. weeks of traditional development
- Personal value justifies personal effort (tools get built for audience-of-one)
- Sophisticated tools solving real problems can now exist without commercial backing

**For Methodology Details**: See `media_server/🧠 Development Methodology.md`

### Tech Stack Commonalities

**Backend**: Go (fast compilation, single binary, low overhead, great stdlib)
**macOS Integration**: Extended attributes (xattr), FSEvents, osascript, diskutil
**Deployment**: User-level LaunchAgents (not root daemons)
**Philosophy**: Computers should manage complexity, not users

---

## Project Documentation Standards

Each project includes:

### Core Documentation
- **PROJECT_OVERVIEW.md** - Comprehensive technical system card
  - Architecture and component responsibilities
  - Implementation details with line references
  - Known issues and performance characteristics
  - Recent changes and git history context

- **README.md** - User-facing documentation
  - Installation and usage instructions
  - Feature overview
  - Quick command reference

### Supporting Documentation
- **QUICKSTART.md** - 30-second getting started guide
- **QUICK-REFERENCE.md** - Command cheat sheet
- **MASTER-DOCUMENTATION.md** - Complete reference manual
- **CHANGELOG.md** - Version history
- **INCIDENT-REPORT.md** - Problem analysis (when applicable)
- **TEST-LOG.md** - Testing methodology and results
- **SUMMARY.md** - Executive summary

### Development Documentation
*(media_server specific, methodology reference)*
- **🧠 Development Methodology.md** - AI-native development process
- **🌐 Open Source Vision.md** - High-level purpose and vision
- **NEXT_CYCLE_IMPROVEMENTS.md** - Current cycle architectural improvements
- **SESSIONS.md** - Session tracking and version history

### Scripts
Each project includes shell scripts for common operations:
- `install.sh` / `build_server.sh` - Build and installation
- `test.sh` - Verify functionality
- `update-agent.sh` / `update-daemon.sh` - Update running services
- `uninstall.sh` - Clean removal

---

## Why These Tools Exist

### The Effort-to-Value Revolution

**Traditional development economics**:
- Building sophisticated personal tools required weeks of full-time effort
- Only justified for commercial products or large team projects
- Personal tools solving real problems remained unbuilt

**AI-assisted development economics**:
- Same sophisticated tools built through conversational sessions
- Effort measured in hours of session time, not weeks of sprint work
- **Personal value justifies personal effort**
- Tools that solve actual problems get built, even for audience-of-one

### Real-World Use Cases

**media_server**:
- Human-indexing toward 5 million files for perceptual AI training
- Generating ground truth labels through manual tagging workflow
- Creating training datasets for perceptual recognition algorithms
- Proving Apple's "10k file limit" is artificial, not technical

**apfs-monitor**:
- Preventing recurring APFS container exhaustion (10 incidents over 6 years)
- Each incident: ~3 hours of emergency cleanup
- Automation ROI: 10x in time saved, infinite in stress reduction
- Total productivity saved: ~30 hours over next 6 years

---

## Development Workflow

### Operating Modes

**Note-Taking Mode** (Documentation/Analysis):
- AI documents issues, captures requirements, analyzes problems
- NO code changes, NO file edits, NO implementations
- Outputs: Issue documentation, design prototypes, architectural analysis

**Code-Making Mode** (Implementation):
- AI makes code changes, edits files, implements solutions
- User must explicitly grant permission: "implement", "go", "dewit", etc.
- AI should ALWAYS confirm mode switch before implementing

### Git Workflow

**Trunk-based development** on `main`/`master`:
- Small, focused commits with clear messages
- Linear history enabling chronological reasoning
- Git log as complete design rationale
- No feature branches, always integrated

### Session Start Protocol

1. Load `PROJECT_OVERVIEW.md` for technical context
2. Use line references to navigate code (e.g., `HandleViewer:390`)
3. Check git history for recent changes and rationale
4. Clarify operating mode (note-taking vs. code-making)
5. Begin work

---

## Token Economics

### Why Comprehensive Documentation Matters

**Without system cards**:
- Read 10-15 files to understand architecture (~20K-40K tokens)
- Grep/search to find components
- Re-discover relationships between files every session
- Explore codebase from scratch each time

**With system cards**:
- Read PROJECT_OVERVIEW.md once (~3K-5K tokens)
- Get immediate context on architecture, file locations, line numbers
- Jump directly to relevant code sections
- Only read specific files when modifications needed

**Result**: 3K-5K tokens for context vs. 20K-40K for codebase exploration (80-90% reduction)

---

## Ecosystem Independence

Breaking out of vendor lock-in:

**Apple's Constraints**:
- Finder/Photos optimized for consumers, breaks at scale
- APFS container complexity hidden from users
- GUI tools show misleading information
- Artificial limitations to control user workflows

**This Approach**:
- Filesystem tags (portable)
- SQLite cache (open format)
- Go binaries (cross-platform capable)
- Open source (no vendor dependency)

**Future Mobility**:
- Data remains accessible outside Apple ecosystem
- No subscriptions, no lock-in
- One-time build effort, infinite runtime value

---

## For the Rest of the Decade

**Prediction**: LLMs + comprehensive system cards + git-based version control will be the dominant development model through 2030.

**Evidence**: These projects prove it's not just viable—it's already superior to traditional approaches for developers who can write and reason clearly.

**Impact**: Sophisticated tools solving real problems can now exist for audiences-of-one, enabling work (like ML dataset preparation at scale) that was previously impossible without commercial backing.

---

## Philosophy

### Complexity Should Be Managed By Computers, Not Users

APFS containers, large file sets, and filesystem quirks require cognitive overhead that should be invisible. These tools automate that mental burden so you can focus on actual work.

### Prevention > Recovery

Automated monitoring and smart architecture prevent crises rather than requiring emergency responses:
- apfs-monitor: 5-minute check vs. 3-hour emergency cleanup
- media_server: Instant cache loads vs. 14-second scans every startup

### Build What You Actually Need

When personal tools are trivial to build, you're not trapped by vendor limitations. You build what you actually need, when you need it.

---

## Project Structure

```
/Users/tdsanchez/dev/
├── README.md                          # This file
├── 🧠 DEVELOPMENT_METHODOLOGY.md      # AI-native development guide
├── PROJECT_INDEX.md                   # Detailed project catalog
├── .obsidian/                         # Obsidian vault config
├── apfs-monitor/                      # APFS container monitor
│   ├── PROJECT_OVERVIEW.md
│   ├── README.md
│   ├── main.go
│   └── [full project structure]
└── media_server/                      # Media library manager
    ├── PROJECT_OVERVIEW.md
    ├── 🧠 Development Methodology.md
    ├── 🌐 Open Source Vision.md
    ├── main.go
    └── [full project structure]
```

---

## Getting Started

### For Users

1. **Choose your tool**:
   - Need to prevent APFS space crises? → `apfs-monitor`
   - Need to manage 100k+ media files? → `media_server`

2. **Read the QUICKSTART**:
   - `apfs-monitor/QUICKSTART.md` - 2-minute installation
   - `media_server` - Build and run instructions in README

3. **Run the test script**:
   ```bash
   cd apfs-monitor && ./test.sh
   # or
   cd media_server && ./build_server.sh
   ```

### For Developers

1. **Read the methodology**:
   - `media_server/🧠 Development Methodology.md`
   - Understand AI-native development workflow
   - Learn system card approach

2. **Study an example**:
   - Pick a project
   - Read PROJECT_OVERVIEW.md
   - Examine git history
   - See how features evolved

3. **Apply the pattern**:
   - Create comprehensive system cards
   - Document architecture and decisions
   - Use explicit operating modes
   - Maintain living documentation

---

## Contributing

These are personal tools open-sourced for others facing similar challenges.

**To contribute**:
1. Fork the specific project repository
2. Read PROJECT_OVERVIEW.md to understand architecture
3. Make your changes following the methodology
4. Test thoroughly
5. Document your changes
6. Submit pull request with clear description

---

## License

**Use freely. No warranty.**

These tools are provided as-is. They have been battle-tested on production systems but your mileage may vary.

**Disclaimer**: Don't blame me if things go wrong. Use at your own risk.

---

## Acknowledgments

**Developed with**: Claude Code (Anthropic)
**Methodology**: AI-native development with system cards
**Philosophy**: Computers should manage complexity, not users
**Result**: Sophisticated personal tools that actually solve real problems

---

*"If a filesystem requires users to constantly think about internal volume interactions, it has failed its primary purpose: abstracting storage complexity away from users."*

*"When personal tools are trivial to build, ecosystem lock-in loses its power."*

---

**Last Updated**: 2025-12-12
**Projects**: 2 active (media_server, apfs-monitor)
**Lines of Go**: ~3,500+ across projects
**Problems Solved**: APFS space management, large-scale file organization
**Hours Saved**: ~30+ (and counting)
