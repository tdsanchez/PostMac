//go:build darwin

package scanner

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/pkg/xattr"
	"howett.net/plist"
)

// SetMacOSTags writes macOS Finder tags directly via xattr (works natively on Darwin).
func SetMacOSTags(path string, tags []string) error {
	data, err := encodePlistTags(tags)
	if err != nil {
		return err
	}
	return xattr.Set(path, "com.apple.metadata:_kMDItemUserTags", data)
}

// GetMacOSComment reads the macOS Finder comment from a file via xattr.
func GetMacOSComment(path string) string {
	data, err := xattr.Get(path, "com.apple.metadata:kMDItemFinderComment")
	if err != nil {
		return ""
	}

	var comment string
	_, err = plist.Unmarshal(data, &comment)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(comment)
}

// SetMacOSComment writes a macOS Finder comment via osascript so Finder and
// Spotlight are properly notified.
func SetMacOSComment(path string, comment string) error {
	escapedComment := strings.ReplaceAll(comment, `"`, `\"`)
	script := fmt.Sprintf(`tell application "Finder" to set comment of (POSIX file "%s" as alias) to "%s"`, path, escapedComment)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("osascript failed: %v, output: %s", err, string(output))
	}
	return nil
}
