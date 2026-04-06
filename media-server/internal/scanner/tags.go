package scanner

import (
	"strings"

	"github.com/pkg/xattr"
	"howett.net/plist"
)

// GetMacOSTags reads macOS Finder tags from a file's xattr.
// The xattr key is macOS-specific, but xattr itself works on Linux too.
// On Linux with non-macOS files, this will simply return nil (no tags).
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

	seen := make(map[string]bool)
	result := []string{}
	for _, tag := range tags {
		if tagStr, ok := tag.(string); ok {
			parts := strings.Split(tagStr, "\n")
			tagName := parts[0]
			if !seen[tagName] {
				seen[tagName] = true
				result = append(result, tagName)
			}
		}
	}

	return result
}

// encodePlistTags encodes a slice of tag strings as a binary plist for the
// com.apple.metadata:_kMDItemUserTags xattr format.
func encodePlistTags(tags []string) ([]byte, error) {
	var plistTags []interface{}
	for _, tag := range tags {
		plistTags = append(plistTags, tag)
	}
	return plist.Marshal(plistTags, plist.BinaryFormat)
}
