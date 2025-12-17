# Pre-Release Checklist

> **Purpose**: Parametric scrubbing procedure executed before EVERY open source release
> **Last Updated**: 2025-12-13
> **Status**: Required protocol - DO NOT skip

---

## Overview

Before **ANY** public release to GitHub or other open source platforms, this repository MUST be scrubbed for:
1. Offensive or age-inappropriate language
2. Personal identifying information
3. Sensitive paths or volume names
4. Credentials or secrets
5. Any other content that shouldn't be public

**This is a parametric process** - the patterns and procedures below should be executed systematically every time.

---

## Scrubbing Protocol

### Step 1: Scan For Offensive Language

**Pattern**: Common profanity and offensive terms

```bash
# Search for offensive language (case-insensitive)
# Note: Pattern intentionally not shown to avoid including offensive terms in this document
# Use standard profanity word list or create your own pattern file
grep -rni -f /path/to/offensive-words.txt \
  --include="*.md" --include="*.go" --include="*.js" --include="*.html" \
  /path/to/repo | grep -v ".git"

# Alternative: Manual review of documentation and user-facing strings
# Focus on: error messages, comments, documentation, commit messages
```

**Action**:
- If matches found: Sanitize or remove
- Document context if technical term vs actual profanity
- Err on side of caution for public release

**Expected Result**: Zero matches

---

### Step 2: Scan For Personal Paths

**Pattern**: Personal volume names, user directories, identifying filesystem paths

```bash
# Check for personal volume names
grep -rn "/Volumes/[^/]*" --include="*.md" --include="*.go" /path/to/repo | grep -v ".git"

# Check for user home directories
grep -rn "/Users/[^/]*" --include="*.md" --include="*.go" /path/to/repo | grep -v ".git"

# Check for other identifying paths
grep -rn "/home/[^/]*" --include="*.md" --include="*.go" /path/to/repo | grep -v ".git"
```

**Sanitization Map**:
- `/Volumes/Terminator/*` → `/Volumes/External/media`
- `/Volumes/<PersonalName>/*` → `/Volumes/External/media`
- `/Users/<username>/*` → `/Users/user/*` or remove entirely
- `/home/<username>/*` → `/home/user/*` or remove entirely

**Action**:
- Replace all personal paths with generic examples
- Maintain semantic meaning (e.g., "media" for media directories)
- Keep path structure realistic (don't use `/path/to/foo`)

**Expected Result**: Zero personal identifiers in paths

---

### Step 3: Scan For Credentials and Secrets

**Pattern**: Passwords, API keys, tokens, secrets

```bash
# Check for credential patterns
grep -rni "password\|secret\|api.key\|token\|bearer\|auth.*key" \
  --include="*.md" --include="*.go" --include="*.js" --include="*.env" \
  /path/to/repo | grep -v ".git" | grep -v "Token.*Savings"

# Check for potential secrets in config files
find /path/to/repo -type f \( -name "*.env" -o -name "*.credentials" -o -name "*.secret" \) \
  ! -path "*/.git/*"
```

**Exclusions**:
- "Token" in context of "LLM tokens" or "token savings" is OK
- "Private" in systemd configs (`PrivateTmp=true`) is OK
- Generic security discussions are OK

**Action**:
- If actual credentials found: **STOP AND ROTATE THEM**
- Remove credential files
- Replace examples with obvious placeholders (`YOUR_API_KEY_HERE`)

**Expected Result**: Zero actual credentials, only contextual references

---

### Step 4: Scan For Email Addresses

**Pattern**: Email addresses that could identify individuals

```bash
# Check for email addresses
grep -rn "@" --include="*.md" --include="*.go" --include="*.js" \
  /path/to/repo | grep -v ".git" | grep -v "http" | grep -v "example.com"
```

**Exclusions**:
- SSH commands like `ubuntu@hostname` are OK if generic
- Example emails with `example.com` are OK
- Documentation references to public mailing lists are OK

**Action**:
- Replace personal emails with `user@example.com`
- Remove contributor emails unless they've explicitly consented

**Expected Result**: Zero personal email addresses

---

### Step 5: Scan For Hardcoded Personal Data

**Pattern**: Names, addresses, phone numbers, other PII

```bash
# Manual review of these sections
# - README.md author sections
# - LICENSE file (if personal)
# - Any "About" or "Contact" sections
# - Git commit author info (can't be changed easily, but be aware)
```

**Action**:
- Use pseudonyms or generic identifiers
- Remove contact info unless explicitly wanting to be contacted
- Consider using GitHub handle instead of real name

**Expected Result**: Minimal PII, intentional only

---

### Step 6: Review Recent Commits

**Pattern**: Check recent commits for anything that shouldn't be public

```bash
# Review last 10 commits
git log -10 --oneline

# Check diff of recent changes
git diff HEAD~10..HEAD
```

**Action**:
- Review commit messages for offensive language or PII
- Check actual code changes for sensitive data
- Note: Can't rewrite history easily once pushed, so catch this early

**Expected Result**: Clean commit history

---

## Sanitization Map (Parametric Replacements)

### Volume Paths
| Original Pattern | Replacement |
|-----------------|-------------|
| `/Volumes/Terminator/*` | `/Volumes/External/media` |
| `/Volumes/Classica/*` | `/Volumes/External/media` |
| `/Volumes/Suprex/*` | `/Volumes/External/media` |
| `/Volumes/<AnyPersonalName>/*` | `/Volumes/External/media` |

### User Paths
| Original Pattern | Replacement |
|-----------------|-------------|
| `/Users/tdsanchez/*` | Remove or use `/Users/user/*` |
| `/Users/<username>/*` | Remove or use `/Users/user/*` |
| `/home/<username>/*` | Remove or use `/home/user/*` |

### Application-Specific Paths
| Original Pattern | Replacement |
|-----------------|-------------|
| `DigiKam Fixed` (personal folder) | `photos` or generic name |
| `Classica` (personal library name) | `media` |
| Other personal folder names | Generic equivalents |

---

## Files That Typically Need Scrubbing

Based on 2025-12-13 scrubbing session, these files commonly contain personal data:

1. **PROJECT_OVERVIEW.md** - Example paths, error messages, cache locations
2. **MULTIPASS_DEPLOYMENT.md** - Volume mount commands
3. **COLIMA_DEPLOYMENT.md** - Docker volume mounts
4. **BUGS.md** - User-reported paths in error messages (if present)
5. **README.md** - Example commands, paths
6. **Any deployment documentation** - Tends to have real paths from testing

**Action**: Prioritize reviewing these files in every scrubbing pass.

---

## Verification Checklist

Before declaring scrubbing complete, verify:

- [ ] No offensive language found (Step 1)
- [ ] No personal volume paths (Step 2)
- [ ] No credentials or secrets (Step 3)
- [ ] No personal email addresses (Step 4)
- [ ] Minimal/intentional PII only (Step 5)
- [ ] Clean recent commit history (Step 6)
- [ ] All files in "typically needs scrubbing" list reviewed
- [ ] Sanitization map patterns applied consistently
- [ ] Generic examples remain semantically meaningful
- [ ] Documentation still makes sense after sanitization

---

## Post-Scrubbing Validation

After scrubbing, do a final validation:

```bash
# Quick smoke test - should return no matches
grep -rn "Terminator\|tdsanchez\|Classica\|Suprex" \
  --include="*.md" --include="*.go" /path/to/repo | grep -v ".git"

# Check for any remaining personal paths
grep -rn "/Volumes/[A-Z]" --include="*.md" /path/to/repo | grep -v "External"
```

**Expected**: Clean output, or only intentional exceptions

---

## Why This Is Parametric

This process is **parametric** because:

1. **Pattern-based**: Uses regex patterns that can be applied to any content
2. **Repeatable**: Same steps every time, deterministic results
3. **Extensible**: Add new patterns as discovered
4. **Automatable**: Could be scripted (though manual review recommended)
5. **Version-controlled**: This checklist itself is in git

**Every release goes through the same process.** No exceptions.

---

## When To Run This

**Required**:
- ✅ Before initial public GitHub release
- ✅ Before any major version tag/release
- ✅ After adding significant new documentation
- ✅ If you suspect personal data was committed

**Recommended**:
- Before any external sharing (blog post links, demos, etc.)
- After major refactoring that touched many files
- Periodically (quarterly?) as sanity check

---

## What To Do If Scrubbing Finds Issues

### Scenario 1: Personal Path Found
- Update sanitization map with new pattern
- Apply replacement globally
- Commit with message: `docs: sanitize personal paths for release`

### Scenario 2: Offensive Language Found
- Remove or rephrase
- Check context (technical term vs actual profanity)
- Commit with message: `docs: sanitize language for public release`

### Scenario 3: Credential Found
- **IMMEDIATELY ROTATE THE CREDENTIAL**
- Remove from all commits (may require history rewrite)
- Document in `.gitignore` to prevent future commits
- Consider: Was this pushed? If yes, assume compromised.

### Scenario 4: PII Found
- Assess: Intentional or accidental?
- If accidental: Remove and sanitize
- If intentional: Ensure consent and document reasoning

---

## Automation Considerations

**Could this be automated?** Partially.

**What can be automated**:
- Pattern scanning (grep-based searches)
- Flagging potential issues
- Applying known sanitization map replacements

**What requires human judgment**:
- Context of language (technical vs offensive)
- Semantic meaning of paths (keep examples realistic)
- PII intentionality (wanted vs accidental)
- Commit message content review

**Recommendation**: Semi-automate scanning, require human review for sanitization.

---

## Git History Deep Cleaning

### When Git History Contains Offensive Content

If the scrubbing process discovers offensive terms in git history (not just current files), you have two options:

#### Option A: Accept Historical Context (Low Risk)
- **When acceptable**: Terms appear in technical documentation context (examples, patterns, checklists)
- **Risk**: Minimal - clearly technical usage, not malicious
- **Action**: Document why it's acceptable, proceed with push
- **Example**: Grep pattern in pre-release checklist showing what to scan for

#### Option B: Clean Git History (Zero Tolerance)
- **When required**: Any offensive terms, regardless of context
- **Risk**: None - completely clean repository
- **Action**: Rewrite git history to remove offensive content

### Git History Cleaning Procedure

**IMPORTANT**: This rewrites git history. Only do this BEFORE first public push. After pushing, history rewriting breaks clones.

**Prerequisites:**
1. Create backup of entire repository
2. Identify commits containing offensive content
3. Verify you have clean replacement content

**Method 1: Reset and Recreate (Simple)**

```bash
# 1. Create backup first
cp -r . ../$(basename $(pwd))-internal

# 2. Identify problem commit hash
PROBLEM_COMMIT="abc123"  # Replace with actual hash
BEFORE_PROBLEM=$(git rev-parse ${PROBLEM_COMMIT}^)

# 3. Save commits that come AFTER the problem
git format-patch ${PROBLEM_COMMIT}..HEAD -o /tmp/good-patches

# 4. Reset to before problem commit
git reset --hard ${BEFORE_PROBLEM}

# 5. Manually create the sanitized version of changed files
# (Create clean versions without offensive content)

# 6. Commit the clean version
git add .
git commit -m "Your clean commit message"

# 7. Reapply the good patches
git am /tmp/good-patches/*.patch

# 8. Verify history is clean
git log --all -p | grep -i "offensive-pattern" || echo "CLEAN"

# 9. Force update branch (ONLY if never pushed publicly)
# git branch -f main HEAD
```

**Method 2: Interactive Rebase (Advanced)**

```bash
# 1. Create backup first
cp -r . ../$(basename $(pwd))-internal

# 2. Start interactive rebase
git rebase -i <commit-before-problem>^

# 3. Mark problem commit as 'edit'
# 4. When rebase stops at that commit:
#    - Fix the files manually
#    - git add <fixed-files>
#    - git commit --amend
#    - git rebase --continue

# 5. Verify clean
git log --all -p | grep -i "offensive-pattern" || echo "CLEAN"
```

**Method 3: Nuclear Option (Complete Reset)**

```bash
# 1. Create backup
cp -r . ../$(basename $(pwd))-internal

# 2. Save current clean state
git checkout main
cp -r . /tmp/clean-repo-state

# 3. Delete all git history
rm -rf .git

# 4. Create fresh repository
git init
git add .
git commit -m "Initial commit: Clean repository for public release"

# 5. Optionally add important historical context
# (Cherry-pick clean commits from backup if needed)
```

### Verification After Cleaning

**Required checks:**
```bash
# 1. Scan all history for offensive content
git log --all -p | grep -i "fuck\|shit\|damn" && echo "FOUND ISSUES" || echo "CLEAN"

# 2. Check commit messages
git log --all --oneline | grep -i "offensive-pattern" && echo "FOUND IN MESSAGES" || echo "CLEAN"

# 3. Verify file count matches
git ls-files | wc -l  # Should match original

# 4. Test build/functionality
# (Run your tests, build process, etc.)

# 5. Compare with backup
diff -r . ../repo-internal --exclude=.git
```

### Post-Cleaning Actions

- [ ] Verify all tests pass
- [ ] Rebuild binary and test functionality
- [ ] Update this checklist's "Last scrubbed" date
- [ ] Document what was cleaned in commit message
- [ ] Keep internal backup until public push succeeds
- [ ] After successful push, can delete internal backup (or keep for records)

### Critical Warnings

⚠️ **NEVER rewrite history after public push** - breaks all clones
⚠️ **ALWAYS backup before history rewriting** - no undo button
⚠️ **VERIFY thoroughly after cleaning** - ensure nothing broke
⚠️ **TEST the cleaned repo** - don't assume it works

---

## Notes

- This document itself will be public - don't include actual sensitive patterns here
- Update sanitization map as new personal identifiers discovered
- Keep this checklist current - if process changes, update this file
- Consider this a living document that evolves with the project

---

*Last scrubbed: 2025-12-13*
*Next scrubbing: Before next public release*
*Scrubbing lead: Repository owner*
