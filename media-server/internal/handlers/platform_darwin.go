//go:build darwin

package handlers

import (
	"fmt"
	"os/exec"
	"time"
)

// trashFile moves a file to Trash via osascript/Finder.
func trashFile(path string) error {
	script := fmt.Sprintf(`
		set filepath to POSIX file "%s"
		tell application "Finder"
			move filepath to trash
		end tell
	`, path)
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

// revealAndPreview reveals and previews a file via Finder and QuickLook.
func revealAndPreview(path string) {
	cmd := exec.Command("sh", "-c", fmt.Sprintf("open -R '%s' > /dev/null 2>&1", path))
	cmd2 := exec.Command("qlmanage", "-p", path)

	cmd.Run()
	time.Sleep(500 * time.Millisecond)
	cmd2.Run()
	time.Sleep(1500 * time.Millisecond)

	script := `tell application "System Events" to set frontmost of first process whose name is "qlmanage" to true`
	exec.Command("osascript", "-e", script).Run()
	time.Sleep(1500 * time.Millisecond)
}
