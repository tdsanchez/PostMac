# PostMac: Media Server

> **Part of the PostMac Project**
> Tools and mindset for Mac users alienated by Apple and Microsoft's recent abandonment of PCs as tools

---

## What Is This?

A high-performance media server for macOS that manages hundreds of thousands of files using filesystem tags, SQLite caching, and AI-assisted development methodology.

**Built to replace DEVONthink**: When commercial tools blame "APFS limitations" for blocking threads and killing performance, you build your own tool that handles 350k files with lock-free concurrency and sub-50ms response times.

**Built to prove a point**: Vendor "technical limitations" are usually architectural decisions. Same hardware, same APFS, 1,600x better performance.

---

## The PostMac Project

**PostMac** (camelCase as a tip of the hat to Steve Jobs) is a self-prototyping AI development effort delivering tools and mindset to Mac users who feel abandoned.

**Created by someone who spent 40+ years on Apple platforms** - This isn't reactionary criticism from an outsider. This is a veteran Apple developer recognizing that the platform has fundamentally changed direction, and building an exit strategy.

**The current landscape**:
- **Apple's trajectory**: Dumbing down macOS for iOS convergence, artificial performance limits, ecosystem lock-in
- **Microsoft's trajectory**: Abandoning Windows as a professional tool, cloud-first mentality, subscription everything
- **The gap**: Power users who want **computers as tools**, not consumption devices

**PostMac's mission**: Help users (and the creator) exit the Apple platform by demonstrating:
1. **Tools you need can be built** - AI-assisted development makes personal tools viable
2. **Vendor limits are often artificial** - This server proves Apple's "10k file limit" is BS
3. **You're not trapped** - Build what you need, escape ecosystem lock-in
4. **The methodology works** - AI-native development is production-ready today
5. **Platform independence is achievable** - Your data, your tools, your control

---

## Quick Start

### Requirements
- macOS (uses extended attributes, FSEvents, osascript)
- Go 1.20+
- SQLite3

### Build and Run
```bash
go build -o media-server cmd/media-server/main.go
./media-server --dir=/Volumes/External/media --port=8080
```

Open http://localhost:8080

---

## Key Features

- ✅ **Handles 350k+ files** (tested) - Apple claims this is impossible, but here we are
- ✅ **1-2 second load times** - SQLite cache eliminates repeated filesystem scans
- ✅ **macOS tag integration** - Uses native filesystem tags, not proprietary database
- ✅ **Keyboard-driven workflow** - Built for "graybeards" who want efficiency
- ✅ **Pagination** - 200 files per page, configurable
- ✅ **Random mode** - Navigate thousands of files randomly for visual variety
- ✅ **FSEvents auto-sync** - Detects external file changes automatically
- ✅ **No vendor lock-in** - Your data stays in standard formats

---

## Documentation

This repository follows an unconventional documentation approach because it's part of a methodology demonstration.

**System Cards (Two Levels)**:
1. **[../.system_card.md](../.system_card.md)** - **Project system card** (behavioral guidelines for Claude Code)
2. **[PROJECT_OVERVIEW.md](./PROJECT_OVERVIEW.md)** - **Application system card** (technical documentation)

**Start here**:
3. **[HOW_TO_USE_THIS_REPOSITORY.md](./HOW_TO_USE_THIS_REPOSITORY.md)** - Why this repo is different
4. **[🧠 Development Methodology.md](./🧠%20Development%20Methodology.md)** - AI-native development paradigm
5. **[BUGS.md](./BUGS.md)** - Real-world testing issues

**Key insight**: Documentation is operational infrastructure, not afterthought. System cards are maintained with the same rigor as code.

---

## Why This Repository Looks Weird

This isn't a typical open source project. It's a **demonstration of AI-native development** that happens to include a working media server.

**Intentionally unconventional**:
- No feature branches (trunk-based development optimized for LLM reasoning)
- Issues tracked in-repo, not GitHub Issues (cross-referenced documentation graph)
- No semantic versioning (the methodology is the product, not the binary)
- Documentation evolved with code (living system cards, not stale docs)
- Cross-referencing protocol (docs form coherent knowledge graph)

**Why?** Because this demonstrates how development will work for the rest of the decade. LLMs + system cards + git = new paradigm.

See [HOW_TO_USE_THIS_REPOSITORY.md](./HOW_TO_USE_THIS_REPOSITORY.md) for full explanation.

---

## Why This Exists: DEVONthink's Performance Failure

**The Problem**: DEVONthink's database operations block threads and kill performance with large file sets

**What they blame**: "APFS architecture limitations", filesystem performance, macOS constraints

**The reality**: Architectural decisions, not technical limits

**The proof** (from this server's logs, right now):
```
📝 FS event: WRITE DEVONthink-5.dtMeta [... 20,000+ rapid events ...]
🔄 Auto-rescan triggered by filesystem changes...
✅ Auto-rescan completed (14,780 files scanned)
Response times: 13-50ms (during scan)
```

**Same APFS filesystem DEVONthink struggles with**:
- This server: 20,000 FS events → Auto-rescan → Zero blocking → 13-50ms latency
- DEVONthink: Similar operations → Thread blocking → Performance death

**The difference**: Lock-free concurrency (double-buffered state with atomic.Value) vs blocking I/O patterns

**Measured improvement**: 1,600x better latency (82,000ms → 13-50ms) during filesystem rescans

This isn't about better hardware or newer APIs. It's about **building it right**.

---

## The "Vendor Limits Are BS" Proof

**Apple claims**: Finder and Photos can't handle more than ~10k files efficiently due to "APFS architecture limitations"

**DEVONthink implies**: Database performance limited by filesystem constraints

**Reality**:
- This tool loads **350,000 files in 2 seconds** on standard Mac hardware
- Same APFS filesystem vendors claim "can't handle it"
- Same hardware, zero performance issues
- Lock-free concurrency during auto-rescan
- Pagination enables browsing millions of files

**Conclusion**: Vendor limitations are **architectural decisions**, not technical constraints. When you can build a better tool in a few AI-assisted development sessions, the "impossible" becomes routine.

DEVONthink's blocking behavior is a choice. Apple's 10k file limit is a choice. **You're not trapped** - build what you need.

---

## Architecture Highlights

- **Go backend** - Fast, compiled, cross-platform
- **SQLite cache** - 14x faster startup (1 second vs 14 seconds for 168k files)
- **macOS integration** - Extended attributes, Finder comments, FSEvents, QuickLook
- **No framework bloat** - Vanilla JavaScript, no React/Vue/etc
- **Optimized for scale** - Tested with 350k files, designed for millions

See [PROJECT_OVERVIEW.md](./PROJECT_OVERVIEW.md) for complete architecture documentation.

---

## PostMac Philosophy: Platform Exit Strategy

**PCs as Tools, Not Consumption Devices**

The original promise of personal computers: **amplify human capability through programmable tools**

Apple and Microsoft have abandoned this:
- Apple: iOS-ification of macOS, walled gardens, artificial limits
- Microsoft: Cloud dependency, subscription models, consumer focus

**Why this matters from a 40-year veteran**:

I've used and developed on Apple hardware for more than 40 years. This isn't about nostalgia or refusing to adapt - it's about recognizing when a platform has fundamentally abandoned its core users.

**The trajectory is clear**:
- 1980s-2000s: Apple built tools for creators and power users
- 2010s: iOS success began reshaping macOS philosophy
- 2020s: "Pro" users increasingly treated as legacy burden
- Result: Artificial limits, performance degradation, ecosystem lock-in prioritized over capability

**PostMac is an exit strategy** - for me and others:
- Build the tools you need (AI makes this viable for individuals)
- Escape ecosystem lock-in (portable data, open formats, standard technologies)
- Prove vendor limits are artificial (this server is the proof)
- Share the methodology (AI-native development for everyone)
- **Enable platform independence** (your data, your tools, your control)

This isn't about staying on macOS while complaining. It's about **building the bridge to leave**.

When someone with 40 years on the platform says "it's time to go," that's not reactionary - it's recognition that the platform you invested in no longer serves you.

**Steve Jobs would understand** - PostMac is named in his honor because he believed in tools that empower individuals, not corporations that extract rent. The current Apple leadership has inverted that philosophy.

---

## Contributing

**Don't.**

Not because contributions aren't welcome, but because **forking misses the point**.

This repo demonstrates a **personal development workflow** optimized for AI collaboration. The methodology only makes sense for individuals or very small teams. Accepting PRs would break the linear git history that makes LLM reasoning work.

**This is an exit strategy demonstration** - the value isn't in the specific tool, it's in showing that **you can build your own exit**.

**Instead of contributing here**:
1. Use this as a template for your own AI-native development
2. Build your own tools to replace the vendor software trapping you
3. Create your own system cards and living documentation
4. Share your results and experiences
5. **Build your own bridge out of the Apple ecosystem**

The methodology is portable. This specific repo is not.

**If you're also leaving Apple** after years on the platform, this project demonstrates:
- Vendor lock-in can be broken (your data in standard formats)
- Performance "limitations" are often artificial (proof by construction)
- AI-native development enables individual tool creation
- You don't need vendor permission to build better tools

---

## License

[Choose appropriate license - MIT, Apache 2.0, GPL, etc.]

---

## Contact

GitHub: [@tdsanchez](https://github.com/tdsanchez)
Project: [PostMac](https://github.com/tdsanchez/PostMac)

---

*Built with AI-native development methodology. Documentation generated and maintained using LLM collaboration. This is intentional.*
