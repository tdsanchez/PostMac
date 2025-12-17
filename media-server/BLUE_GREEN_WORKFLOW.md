# Blue/Green Workflow Architecture

> **Purpose**: Document the multi-tier repository architecture and blue/green deployment strategy
> **Last Updated**: 2025-12-13
> **Status**: Active development pattern

---

## Overview

This repository follows a **blue/green workflow** designed to separate private development from public releases while maintaining the value of trunk-based development and AI-assisted collaboration.

**Core principle**: Keep full development history private (Blue), release sanitized versions publicly (Green), while demonstrating AI-native development methodology.

---

## Three-Tier Architecture

```
┌─────────────────────────────────────────┐
│   [Parent Project]                      │
│   (Private development workspace)       │
│   Location: [Unknown/Not yet linked]    │
└──────────────┬──────────────────────────┘
               │
               │ (git submodule)
               │
               ▼
┌─────────────────────────────────────────┐
│   media-server-internal (BLUE)          │
│   This repository - Private dev         │
│   Location: /Users/tdsanchez/dev/       │
│           media-server-internal         │
└──────────────┬──────────────────────────┘
               │
               │ (sanitized export)
               │
               ▼
┌─────────────────────────────────────────┐
│   PostMac (GREEN)                       │
│   Public GitHub repository              │
│   Location: github.com/tdsanchez/PostMac│
└─────────────────────────────────────────┘
```

---

## Environment Definitions

### Blue Environment (Private Development)

**Location**: `/Users/tdsanchez/dev/media-server-internal` (this repository)

**Characteristics**:
- ✅ Full git history preserved
- ✅ Personal paths and context (`/Volumes/Terminator`, etc.)
- ✅ Work-in-progress features and experiments
- ✅ Claude collaboration sessions with full context
- ✅ Trunk-based development optimized for LLM reasoning
- ✅ Real error messages with actual paths
- ✅ Personal testing notes and observations
- ✅ **Never pushed to GitHub or any public host**

**Purpose**:
- Primary development workspace
- Where features are designed, implemented, and tested
- Where documentation evolves organically with code
- Full context for AI collaboration without sanitization overhead

**Git Strategy**:
- Local repository only (or private remote if needed)
- Linear history (no feature branches)
- Natural language commit messages optimized for LLM context
- All documentation committed with code changes

---

### Green Environment (Public Release)

**Location**: `github.com/tdsanchez/PostMac` (or similar - not yet created)

**Characteristics**:
- ✅ Sanitized paths and personal identifiers
- ✅ Curated git history (cherry-picked or cleaned)
- ✅ Public-facing documentation
- ✅ Generic examples that remain semantically meaningful
- ✅ Demonstrates methodology without exposing personal context
- ✅ README optimized for external audience
- ✅ **Publicly accessible on GitHub**

**Purpose**:
- Demonstrate AI-native development methodology
- Share working application as proof of concept
- Prove vendor limitations (Apple's 10k file limit) are artificial
- Provide template for others to replicate methodology
- Philosophy: "PostMac" - tools for users abandoned by Apple/Microsoft

**Git Strategy**:
- Public GitHub repository
- May have rewritten/cleaned history OR fresh start with attribution
- Focus on demonstrating methodology, not full development journey
- Cross-referenced documentation system intact but sanitized

---

### Parent Project (Integration Layer)

**Location**: Unknown - not yet integrated (as of 2025-12-13)

**Suspected Characteristics**:
- Likely another private development workspace
- Contains multiple tools/projects beyond media-server
- May follow similar blue/green pattern
- media-server-internal intended as git submodule

**Status**: 🔴 **Unclear** - needs investigation

**Questions to resolve**:
- What is the parent project?
- Is media-server already linked as submodule?
- Does parent also follow blue/green workflow?
- Is parent purely organizational or also released publicly?

**Investigation needed**:
```bash
# Check if we're inside a parent repo
git rev-parse --show-superproject-working-tree

# Check remote configuration
git remote -v

# Look at directory structure
ls -la ..

# Check for .gitmodules in parent
cat ../.gitmodules 2>/dev/null
```

---

## Blue/Green Workflow: Data Flow

### Development Flow (Blue → Green)

```
1. [BLUE] Feature development
   ├─ Design with Claude
   ├─ Implement incrementally
   ├─ Test with real data/paths
   ├─ Document in system cards
   └─ Commit to local git history

2. [BLUE] Pre-release preparation
   ├─ Run PRE_RELEASE_CHECKLIST.md protocol
   ├─ Sanitize paths (see sanitization map below)
   ├─ Review documentation for personal info
   ├─ Test sanitized version still makes sense
   └─ Tag commit as "release candidate"

3. [BLUE → GREEN] Export/Sanitization
   ├─ Clone or export specific commits
   ├─ Apply sanitization map globally
   ├─ Verify all personal identifiers removed
   ├─ Rebuild to test functionality
   └─ Create clean git history (optional)

4. [GREEN] Public release
   ├─ Push to GitHub
   ├─ Verify documentation renders correctly
   ├─ Check no personal data leaked
   └─ Monitor for issues

5. [GREEN] Maintenance
   ├─ Bug reports from public → track in Blue
   ├─ Fix in Blue environment
   ├─ Re-sanitize and export to Green
   └─ Update public repository
```

---

## Sanitization Map (Blue → Green)

### Path Replacements

| Blue (Private) | Green (Public) | Context |
|---------------|----------------|---------|
| `/Volumes/Terminator/*` | `/Volumes/External/media` | Personal volume name |
| `/Volumes/Classica/*` | `/Volumes/External/media` | Personal library |
| `/Volumes/Suprex/*` | `/Volumes/External/media` | Personal volume |
| `/Users/tdsanchez/*` | `/Users/user/*` or removed | Personal user path |
| `DigiKam Fixed/` | `photos/` | Personal folder name |
| Any personal folder names | Generic equivalents | Maintain semantic meaning |

### Content Replacements

| Category | Action |
|----------|--------|
| Offensive language | Remove or rephrase (see PRE_RELEASE_CHECKLIST.md) |
| Personal email | Replace with `user@example.com` |
| Credentials/secrets | **NEVER commit - rotate if found** |
| Error messages with paths | Sanitize paths but keep error structure |
| Testing notes with personal context | Generalize or remove |
| Git commit messages | Review for personal info (may rewrite history) |

---

## Why This Architecture?

### Problem Being Solved

**Without blue/green separation**:
- ❌ Can't collaborate freely with Claude (sanitization overhead every commit)
- ❌ Can't share methodology without exposing personal data
- ❌ Manual sanitization is error-prone and tedious
- ❌ Either stay private (no value sharing) OR overshare (privacy/security risk)

**With blue/green separation**:
- ✅ Develop freely with full context in Blue
- ✅ Share methodology and value in Green
- ✅ Systematic sanitization process (PRE_RELEASE_CHECKLIST.md)
- ✅ Privacy protected, value shared
- ✅ Git history stays private but methodology demonstrated publicly

### Microsoft/GitHub Concern Addressed

**Concern**: "Not sucked up by Microsoft" - GitHub is owned by Microsoft

**Solution**:
- Blue environment: Local git, never pushed to GitHub (or private self-hosted git)
- Green environment: Only sanitized, public-ready content goes to GitHub
- Full development history stays private
- Microsoft/GitHub only sees what you choose to share

**Result**: Control over your data while still participating in open source community

---

## Integration with Parent Project

### Current Status (2025-12-13)

🔴 **Unknown** - User indicated parent project exists but integration status unclear ("I think [not integrated yet]")

### Possible Integration Scenarios

#### Scenario A: Not Yet Linked
- media-server-internal exists standalone
- Parent project exists but hasn't added this as submodule yet
- **Action needed**: Investigate parent structure, link as submodule

#### Scenario B: Already Linked
- This repo already cloned as submodule in parent
- Working from within parent project structure
- **Action needed**: Verify with `git rev-parse --show-superproject-working-tree`

#### Scenario C: Independent Development
- No parent project integration needed
- media-server standalone blue environment
- **Action needed**: Clarify project relationships

### Questions to Resolve

1. **Parent project identity**: What is it? Where is it?
2. **Integration method**: Git submodule? Monorepo? Separate clones?
3. **Parent workflow**: Does parent also follow blue/green pattern?
4. **Release coordination**: If parent releases publicly, how does media-server fit?
5. **Development location**: Are we working inside parent or standalone?

---

## Practical Workflow Examples

### Adding a New Feature

**Blue Environment (Development)**:
```bash
# Working in media-server-internal (Blue)
cd /Users/tdsanchez/dev/media-server-internal

# Develop feature with Claude
# - Use real paths in examples
# - Test with actual data at /Volumes/Terminator/media
# - Commit with natural language messages

git add .
git commit -m "feat: add beaming feature for rapid file organization

Implements [B] keyboard shortcut to move/copy files to predefined
target directory. Tested with /Volumes/Terminator/media →
/Volumes/Classica/sorted workflow.

Uses existing cache update mechanism, integrates with FSEvents."

# Feature complete in Blue - ready when needed for Green release
```

**Green Environment (Public Release)**:
```bash
# When ready to release publicly
cd /Users/tdsanchez/dev/media-server-internal

# Run pre-release checklist
# - Sanitize all paths
# - Review documentation
# - Test sanitized version

# Create or update public repository
cd /tmp
git clone git@github.com:tdsanchez/PostMac.git
cd PostMac

# Cherry-pick sanitized commits or apply patches
# Git history clean, paths sanitized, ready for public

git push origin main
```

### Receiving Bug Reports from Green

**Public user reports issue on GitHub**:
1. Note issue in Blue environment (BUGS.md)
2. Reproduce with Blue environment data/paths
3. Fix in Blue, test thoroughly
4. Run pre-release checklist
5. Export sanitized fix to Green
6. Update public repository
7. Close GitHub issue with reference to fix

**Blue and Green stay synchronized but independent.**

---

## Benefits of This Approach

### For Development (Blue)
- ✅ Zero sanitization overhead during active development
- ✅ Full context available for AI collaboration
- ✅ Real paths and data for testing
- ✅ Natural commit messages without censoring
- ✅ Experimentation without public scrutiny

### For Sharing (Green)
- ✅ Methodology demonstrated publicly
- ✅ Privacy and security maintained
- ✅ Professional presentation
- ✅ No personal identifiers leaked
- ✅ Value delivered to community

### For Methodology
- ✅ Proves AI-native development works at scale
- ✅ Living documentation system demonstrated
- ✅ Trunk-based + LLM collaboration validated
- ✅ Template for others to replicate
- ✅ Addresses real privacy concerns while remaining open

---

## Trade-offs and Limitations

### Maintenance Overhead
- ❌ Must sanitize before each public release
- ❌ Two repositories to maintain (Blue + Green)
- ❌ Risk of sanitization mistakes (mitigated by checklist)
- ⚖️ **Judgment**: Worth it for privacy + sharing benefits

### History Divergence
- ❌ Blue and Green git histories differ
- ❌ Can't easily merge between them
- ❌ Green may have rewritten/cleaned history
- ⚖️ **Judgment**: Acceptable - they serve different purposes

### Documentation Duplication
- ❌ Some docs need sanitized versions in Green
- ❌ Updates must be applied to both environments
- ❌ Risk of documentation drift
- ⚖️ **Judgment**: Mitigated by systematic approach (PRE_RELEASE_CHECKLIST.md)

### Public Contribution Complexity
- ❌ Can't accept PRs to Green (breaks linear history)
- ❌ Contributors can't see full Blue development context
- ❌ Methodology demonstration, not collaborative project
- ⚖️ **Judgment**: Intentional - see README "Contributing" section

---

## Related Documentation

- **[PRE_RELEASE_CHECKLIST.md](./PRE_RELEASE_CHECKLIST.md)** - Systematic sanitization protocol before Green releases
- **[🌐 Open Source Vision.md](./🌐%20Open%20Source%20Vision.md)** - High-level philosophy and goals for public release
- **[🧠 Development Methodology.md](./🧠%20Development%20Methodology.md)** - AI-native development process used in Blue
- **[PROJECT_OVERVIEW.md](./PROJECT_OVERVIEW.md)** - Technical system card for the application
- **[README.md](./README.md)** - Public-facing description (Green-ready)

---

## Next Steps

### Immediate Actions Needed

1. **Clarify parent project integration**:
   - [ ] Identify parent project location
   - [ ] Verify submodule status
   - [ ] Document parent's blue/green strategy (if applicable)

2. **Verify Blue environment isolation**:
   - [ ] Confirm no remotes pointing to public Git hosts
   - [ ] Check `.git/config` for accidental GitHub references
   - [ ] Ensure local-only development

3. **Prepare for first Green release**:
   - [ ] Create GitHub repository (PostMac or similar)
   - [ ] Run full PRE_RELEASE_CHECKLIST.md protocol
   - [ ] Test sanitized version thoroughly
   - [ ] Push initial release

### Long-term Workflow

- Develop all features in Blue
- Release to Green periodically (semantic milestones or feature completeness)
- Maintain documentation coherence across both environments
- Use parent project for integration (once clarified)

---

*This document is part of the Blue environment system card library. It will be sanitized before inclusion in Green releases.*

**Last Updated**: 2025-12-13
**Status**: Active - clarification needed on parent project integration
