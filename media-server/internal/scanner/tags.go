package scanner

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/pkg/xattr"
	"howett.net/plist"
)

// GetMacOSTags reads macOS Finder tags from a file
func GetMacOSTags(path string) []string {
	data, err := xattr.Get(path, "com.apple.metadata:_kMDItemUserTags")
	if err != nil {
		return nil
	}

	var tags []interface{}
	_, err = plist.Unmarshal(data, &tags)
	if err != nil {
		return nil
	}

	// Use a map to track unique tags and deduplicate
	seen := make(map[string]bool)
	result := []string{}
	for _, tag := range tags {
		if tagStr, ok := tag.(string); ok {
			parts := strings.Split(tagStr, "\n")
			tagName := parts[0]
			// Only add if we haven't seen this tag before
			if !seen[tagName] {
				seen[tagName] = true
				result = append(result, tagName)
			}
		}
	}

	return result
}

// SetMacOSTags writes macOS Finder tags to a file
func SetMacOSTags(path string, tags []string) error {
	var plistTags []interface{}
	for _, tag := range tags {
		plistTags = append(plistTags, tag)
	}

	data, err := plist.Marshal(plistTags, plist.BinaryFormat)
	if err != nil {
		return err
	}

	return xattr.Set(path, "com.apple.metadata:_kMDItemUserTags", data)
}

// GetMacOSComment reads the macOS Finder comment from a file
// Uses xattr for fast reading during scan (comments set via osascript are properly encoded as plist)
func GetMacOSComment(path string) string {
	data, err := xattr.Get(path, "com.apple.metadata:kMDItemFinderComment")
	if err != nil {
		return ""
	}

	// Decode as binary plist (comments set via Finder/osascript are encoded this way)
	var comment string
	_, err = plist.Unmarshal(data, &comment)
	if err != nil {
		// If plist decoding fails, return empty (corrupted comment data)
		return ""
	}

	return strings.TrimSpace(comment)
}

// SetMacOSComment writes a macOS Finder comment to a file using AppleScript
// This ensures Finder and Spotlight are properly notified
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
