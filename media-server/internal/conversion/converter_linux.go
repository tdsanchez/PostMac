//go:build !darwin

package conversion

import (
	"fmt"
	"os"

	"github.com/tdsanchez/PostMac/internal/state"
)

// ConvertToHTML is a Linux stub. RTF/WebArchive conversion via textutil is
// macOS-only. Pandoc or LibreOffice can be wired in here when needed.
func ConvertToHTML(sourcePath string) (string, error) {
	cache := state.GetConversionCache()
	if cached, ok := cache.Load(sourcePath); ok {
		if cachedPath, ok := cached.(string); ok {
			if _, err := os.Stat(cachedPath); err == nil {
				return cachedPath, nil
			}
		}
	}

	return "", fmt.Errorf("RTF/WebArchive conversion not supported on Linux (no textutil equivalent installed)")
}
