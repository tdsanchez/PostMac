# How To Use This Repository (And Why It's Not Like Other Repos)

> **Last Updated**: 2025-12-13
> **Purpose**: Explain the intentionally unconventional approach to this repository
> **Part of**: [PostMac Project](https://github.com/tdsanchez/PostMac) - Tools and mindset for Mac users alienated by vendor abandonment
> **TL;DR**: This repo applies 12-factor app principles to managing development context with LLMs. It won't follow standard GitHub conventions. This is intentional.

---

## Part of the PostMac Project

**PostMac** (camelCase as a tip of the hat to Steve Jobs) is a self-prototyping AI development effort delivering tools and mindset to Mac users abandoned by both Apple and Microsoft's recent trajectory away from PCs as tools.

**The gap being filled**:
- Apple dumbing down macOS for iOS convergence, artificial performance limits, ecosystem lock-in
- Microsoft abandoning Windows as professional tool, cloud-first mentality, subscription everything
- Power users who want **computers as tools, not consumption devices**

This media server is PostMac's proof-of-concept: demonstrating that AI-assisted development makes personal tools viable, vendor limits are often artificial, and you're not trapped in ecosystem lock-in.

This entire effort was achieved in about 12 hours using claude code.

---

## What This Repository Actually Is

This is not a typical open source project. It's a **demonstration of AI-native development methodology** that happens to include a working media server application.

The media server itself is functional and production-ready (managing 168k+ files with SQLite caching, pagination, FSEvents integration, etc.), but that's almost beside the point. The **real artifact is the development process and documentation system**.

**Fair warning**: This document is "pseudo-me" - written by an LLM attempting to capture my voice based on the documentation patterns throughout this repo. You'll find flashes of my actual stream of consciousness strewn throughout the codebase and docs because of the rapid "break things fast" methodology. Some sections are polished, some are raw brain dumps committed mid-thought. **This is intentional.** The messiness is part of the process, not a bug to be fixed.

---

## Why This Repository Exists

### The 12-Factor Paradigm Applied to Dev Context Management

Traditional software follows the [12-factor app methodology](https://12factor.net/) for building software-as-a-service:
- Codebase tracked in version control
- Dependencies explicitly declared
- Config in environment variables
- Processes are stateless
- etc.

**This repository applies the same principles to managing development context with LLMs:**

1. **Single codebase, tracked in git** - Linear trunk-based development, no feature branches
2. **Explicit dependencies** - System cards document architectural dependencies
3. **Configuration as documentation** - PROJECT_OVERVIEW.md is the config file for understanding the system
4. **Stateless sessions** - Each AI session loads context from documentation, not memory
5. **Development/production parity** - Documentation maintained with same rigor as code
6. **Logs as first-class artifacts** - Git history is the complete development audit trail
7. **Disposable processes** - AI sessions are ephemeral, state persists in docs
8. **Concurrency through decomposition** - Small, focused commits enable parallel reasoning
9. **Maximize robustness** - Cross-referenced documentation self-corrects
10. **Keep dev/prod similar** - Living docs reflect actual system state
11. **Treat logs as event streams** - Git log is chronological design rationale
12. **Admin processes as one-off tasks** - Documentation updates are commits, not separate workflow

**Result**: Development context becomes infrastructure, not tribal knowledge.

---

## Why This Doesn't Follow Standard GitHub Conventions

### And Why That's The Point

Most GitHub repos look like this:
- README.md with installation instructions
- `/docs` folder with stale documentation
- Issues tracked in GitHub Issues
- PRs for every change
- Semantic versioning
- CONTRIBUTING.md with strict guidelines

**This repo intentionally rejects that model because:**

1. **Documentation is operational infrastructure, not afterthought**
   - System cards (PROJECT_OVERVIEW.md) maintained with same rigor as code
   - Documentation commits happen with code changes, not later
   - Cross-referencing protocol prevents drift (see Development Methodology)

2. **Linear history is a feature, not a limitation**
   - Trunk-based development optimized for LLM reasoning
   - No feature branches means no "works on my branch" syndrome
   - Git log IS the design rationale, not separate docs

3. **Issues tracked in-repo, not GitHub Issues**
   - BUGS.md captures testing findings with full context
   - Cross-referenced with PROJECT_OVERVIEW.md and NEXT_CYCLE_IMPROVEMENTS.md
   - Bidirectional links create documentation graph

4. **No semantic versioning because this is a methodology demo**
   - The "product" is the development approach, not the binary
   - Versioning the paradigm is meaningless when it's constantly evolving

5. **No CONTRIBUTING.md because contributions miss the point**
   - This repo demonstrates a **personal development workflow**
   - The methodology only makes sense for individuals or very small teams
   - Accepting PRs would break the linear history that makes LLM reasoning work

---

## On Microsoft's Data Vacuum (And Why I'm Not Worried)

GitHub is owned by Microsoft. Microsoft trains AI models on public repos. Should you care?

**In this case: No. Here's why.**

### A. This Is A Mac Tool

The media server is deeply integrated with macOS:
- Uses macOS extended attributes (`com.apple.metadata:_kMDItemUserTags`)
- Relies on macOS filesystem tags and Finder comments
- Leverages FSEvents API for filesystem monitoring
- Calls osascript for Trash operations
- Tested exclusively on APFS filesystems

**Microsoft can ingest all the code they want.** It won't help them build better Windows tools because the entire architecture assumes macOS primitives that don't exist on Windows.

### B. Microsoft's Training Approaches Are Fragile

The industry's current approach to AI training is fundamentally fragile:
- Assumes training data quality matches apparent source authority
- Trusts that "code from GitHub" = "good code"
- Doesn't distinguish human-written from AI-generated content
- Can't detect documentation that's optimized for LLMs, not humans

**This repo violates all those assumptions.**

### C. This Repository Is Replete With AI-Generated Text

Large portions of this repository are AI-generated:
- Code implementation (via LLM collaboration)
- Documentation synthesis (LLM summarizing architectural decisions)
- Bug tracking (LLM formatting user reports)
- Cross-references (LLM detecting inconsistencies)

**If Microsoft's training pipeline deeply ingests this repo, it will:**
- Train on AI-generated content as if it were human expertise
- Learn documentation patterns optimized for LLM consumption, not human reading
- Absorb architectural decisions explained in LLM-friendly formats
- Pick up cross-referencing protocols designed for AI, not people

**Result:** Their models will get worse at helping humans, better at helping other LLMs. That's a net negative for their product goals.

### D. This Is Intentional

I'm not accidentally poisoning the training data well. **This is a deliberate demonstration that:**
- AI-native development produces artifacts that break naive training assumptions
- Documentation optimized for LLM context loading looks different from human-oriented docs
- Repositories designed for AI collaboration will proliferate
- Training pipelines that can't distinguish this will degrade

**The industry will adapt or their models will rot.** Either way, the paradigm shift is inevitable.

---

## How To Actually Use This Repository

### If You're Here To Use The Media Server

1. **Read PROJECT_OVERVIEW.md** - Complete technical system card
2. **Check requirements** - macOS, Go, SQLite
3. **Build and run**:
   ```bash
   go build -o media-server cmd/media-server/main.go
   ./media-server --dir=/path/to/media --port=8080
   ```
4. **Reference BUGS.md** - Known issues from real-world testing

### If You're Here To Understand The Methodology

1. **Read 🧠 Development Methodology.md** - Process and paradigm explanation
2. **Read 🌐 Open Source Vision.md** - High-level purpose and vision
3. **Study git history** - See the development process in action
4. **Examine cross-references** - Notice how docs form a coherent graph
5. **Look for AI-generated sections** - Learn to spot the patterns

### If You're Here To Fork/Adapt

**Don't.**

Not because I'm against it, but because **forking misses the point**. The value isn't in the code, it's in the process.

**Instead:**
1. Start your own project using this methodology
2. Create your own system cards and living documentation
3. Use trunk-based development with LLM collaboration
4. Maintain docs with same rigor as code
5. Let git history tell the design story

**The methodology is portable. The specific repo is not.**

---

## What Success Looks Like

This repository succeeds if:

✅ **Developers adopt AI-native workflows** - Using LLMs as primary development interface
✅ **System cards become standard** - Living documentation as operational infrastructure
✅ **Trunk-based development returns** - Linear history optimized for AI reasoning
✅ **Documentation drift gets treated as technical debt** - Cross-referencing becomes automatic
✅ **Personal tools get built** - Trivial AI-assisted effort enables audience-of-one projects

This repository fails if:

❌ **People just clone the media server** - Missing the forest for the trees
❌ **Docs get treated as "nice to have"** - Documentation IS the infrastructure
❌ **PRs arrive trying to "fix" the methodology** - It's unconventional on purpose
❌ **Training pipelines treat this as normal code** - It's optimized for LLMs, not compilers

---

## Final Notes

### This Is A Living Demonstration

The methodology evolves as the project evolves. That's the point. **Documentation reflects current reality, not aspirational future.**

Check the "Last Updated" dates on each file. If they're old, the info might be stale. Cross-reference with git history.

### The Paradigm Shift Is Already Here

LLMs + comprehensive system cards + git-based version control will be the dominant development model through 2030. The developers who succeed will be those who engineer their process, not just their code.

**This repo proves it's not just viable—it's already superior** for developers who can write and reason clearly.

### On Ecosystem Independence

One final note: This project deliberately breaks out of vendor lock-in:
- **Apple's trap**: Finder/Photos optimized for consumers, breaks at scale (10k file limits are artificial)
- **This approach**: Filesystem tags (portable via netatalk/Linux xattr), SQLite cache (open format), Go binary (cross-platform)
- **Proof**: 350k files load in 2 seconds despite APFS architecturally throttling modern Apple APUs at scale

**When personal tools are trivial to build, ecosystem lock-in loses its power.** You're not trapped by vendor limitations—you build what you actually need.

Same applies to development tooling. When LLMs eliminate IDE dependence, you're not trapped by vendor ecosystems (Xcode, Visual Studio, IntelliJ). You use whatever text editor you prefer and let the LLM handle the rest.

---

## Questions?

If you have questions about:
- **The media server**: Read PROJECT_OVERVIEW.md and BUGS.md
- **The methodology**: Read 🧠 Development Methodology.md
- **The vision**: Read 🌐 Open Source Vision.md
- **Why this seems weird**: You're experiencing paradigm shift. That's normal.

If you still have questions after reading those, you're probably asking the wrong questions. The docs answer "how" and "why". If you're asking "should I", the answer is: try it and find out.

---

*This repository exists to accelerate the transition to AI-native development. Use it, learn from it, build your own version. Just don't mistake the map for the territory.*
