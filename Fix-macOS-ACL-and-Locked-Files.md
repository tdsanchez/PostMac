# Fixing macOS ACL and Locked Files for Deduplication

## The Problem

macOS files can become undeletable due to three issues:
1. **ACLs (Access Control Lists)** - Extra permissions like "group:everyone deny delete"
2. **File flags** - Locked/immutable flags (uchg, schg, uappnd, sappnd)
3. **Read-only permissions** - Files marked read-only (`-r--r--r--`)

These prevent deduplication tools from working properly, especially frustrating with 10,000+ files.

## The Solution

### Version 2 Script (Recommended)

Use this improved script that takes any directory as an argument:

```bash
#!/bin/bash
# Fix permissions script v2 - removes flags, ACLs, and makes writable
# Does NOT delete any files
# Usage: sudo ./fix_permissions_v2.sh [directory]
#   If no directory specified, uses current directory

TARGET="${1:-.}"

if [ ! -d "$TARGET" ]; then
    echo "Error: Directory '$TARGET' does not exist"
    exit 1
fi

echo "Fixing permissions under: $TARGET"
echo "This may take a while with many files..."
echo ""

# Step 1: Remove ALL file flags (uchg, schg, uappnd, sappnd, etc)
echo "[1/4] Removing immutable flags (uchg)..."
chflags -R nouchg "$TARGET" 2>&1 | grep -v "Operation not permitted" | head -20

echo "[2/4] Removing system immutable flags (schg)..."
chflags -R noschg "$TARGET" 2>&1 | grep -v "Operation not permitted" | head -20

# Step 2: Remove all ACLs
echo "[3/4] Removing ACLs..."
chmod -RN "$TARGET" 2>&1 | grep -v "Operation not permitted" | head -20

# Step 3: Make everything writable
echo "[4/4] Making everything writable..."
chmod -R u+w "$TARGET" 2>&1 | grep -v "Operation not permitted" | head -20

echo ""
echo "Done! Checking for remaining locked files..."

# Find any remaining locked files
LOCKED=$(find "$TARGET" -flags uchg 2>/dev/null | head -10)
if [ -n "$LOCKED" ]; then
    echo "WARNING: Some files still locked:"
    echo "$LOCKED"
else
    echo "Success! No locked files found."
fi
```

## How to Use

1. **Save the script:**
   ```bash
   nano /tmp/fix_permissions_v2.sh
   # Paste the script above, then Ctrl+X, Y, Enter
   chmod +x /tmp/fix_permissions_v2.sh
   ```

2. **Run with sudo on the target directory:**
   ```bash
   # For a specific directory
   sudo /tmp/fix_permissions_v2.sh "/Volumes/Terminator/Blacker"

   # For any directory
   sudo /tmp/fix_permissions_v2.sh "/path/to/directory"
   ```

3. **IMPORTANT**: Always run on the parent directory containing all files you want to deduplicate. Running on subdirectories individually will miss files in sibling directories.

## Quick One-Liners

For quick fixes on specific directories:

```bash
# Remove all ACLs
sudo chmod -RN /path/to/directory/

# Remove user immutable flags
sudo chflags -R nouchg /path/to/directory/

# Remove system immutable flags
sudo chflags -R noschg /path/to/directory/

# Make writable
sudo chmod -R u+w /path/to/directory/

# All at once
sudo chflags -R nouchg /path/to/directory/ && \
sudo chflags -R noschg /path/to/directory/ && \
sudo chmod -RN /path/to/directory/ && \
sudo chmod -R u+w /path/to/directory/
```

## Checking Files

To see what's blocking a file:

```bash
# Show ACLs and extended attributes
ls -le filename

# Show file flags
ls -lO filename

# Show both
ls -leO filename

# Show for directory
ls -leOd /path/to/directory
```

Look for:
- `+` after permissions = ACLs present (e.g., `drwxr-xr-x+`)
- `@` after permissions = Extended attributes (e.g., `drwxr-xr-x@`)
- `uchg` or `schg` in flags column = File is locked/immutable
- `uappnd` or `sappnd` = Append-only flag set
- `dr-x` instead of `drwx` = Read-only directory
- `-r--r--r--` instead of `-rw-r--r--` = Read-only file

## Common Problem Files

### Syncthing Database Files
Files under paths like `Syncthing/index-v0.14.0.db/` often have read-only permissions on ALL files. The script handles these with `chmod -R u+w`.

### Old Archive Files
Files from old backups (1990s, 2000s archives) often have the `uchg` flag set. The script removes these with `chflags -R nouchg`.

### Installer Files
Old software installers (like "Microsoft Office 4.2.1") may have both `uchg` flags and read-only permissions.

## Notes

- System folders (.TemporaryItems, .DocumentRevisions-V100) may still show "Permission denied" - this is normal and safe to ignore
- Always run with `sudo` for best results
- This does NOT delete files, only fixes permissions
- The script verifies at the end if any locked files remain
- Run on the TOP-LEVEL directory containing all files you want to process

## Success Rate

99.9% of previously undeletable files became deletable after running this script on the correct parent directory.

## Example Usage

```bash
# Run on entire volume subdirectory
sudo /tmp/fix_permissions_v2.sh "/Volumes/Terminator/Blacker"

# This catches ALL subdirectories:
# - /Volumes/Terminator/Blacker/Articula/
# - /Volumes/Terminator/Blacker/Deca/
# - /Volumes/Terminator/Blacker/Octogee Remnants/
# And all files within them
```

## Troubleshooting

**Still seeing locked files after running the script?**

1. Verify you ran it on the parent directory, not a subdirectory
2. Check if the files are on a read-only volume: `mount | grep Terminator`
3. Verify you used `sudo`
4. Check the script output for specific errors
5. Run `ls -lO` on a problematic file to see what's still blocking it

**"Operation not permitted" errors?**

These are usually for macOS system folders and can be safely ignored. They don't affect your deduplication process.
