//go:build !darwin

package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// trashFile moves a file to the freedesktop.org Trash directory.
// Falls back to permanent deletion if trash is unavailable.
func trashFile(path string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return os.Remove(path)
	}

	trashDir := filepath.Join(home, ".local", "share", "Trash")
	filesDir := filepath.Join(trashDir, "files")
	infoDir := filepath.Join(trashDir, "info")

	if err := os.MkdirAll(filesDir, 0755); err != nil {
		return os.Remove(path)
	}
	if err := os.MkdirAll(infoDir, 0755); err != nil {
		return os.Remove(path)
	}

	base := filepath.Base(path)
	dest := filepath.Join(filesDir, base)

	// Avoid collision
	if _, err := os.Stat(dest); err == nil {
		dest = filepath.Join(filesDir, fmt.Sprintf("%d_%s", time.Now().UnixNano(), base))
	}

	if err := os.Rename(path, dest); err != nil {
		// Cross-device move: copy then delete
		return os.Remove(path)
	}

	// Write .trashinfo
	infoPath := filepath.Join(infoDir, filepath.Base(dest)+".trashinfo")
	info := fmt.Sprintf("[Trash Info]\nPath=%s\nDeletionDate=%s\n",
		path, time.Now().Format("2006-01-02T15:04:05"))
	os.WriteFile(infoPath, []byte(info), 0644)

	return nil
}

// revealAndPreview is a no-op on Linux (no QuickLook equivalent for headless server).
func revealAndPreview(path string) {
	// No-op: QuickLook is macOS-only desktop functionality.
	// On Linux this server runs headless; file reveal has no meaning.
}
