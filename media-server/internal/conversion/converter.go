package conversion

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tdsanchez/PostMac/media-server/internal/state"
)

// ConvertToHTML converts RTF or WebArchive files to HTML using textutil
func ConvertToHTML(sourcePath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(sourcePath))

	cache := state.GetConversionCache()
	if cached, ok := cache.Load(sourcePath); ok {
		if cachedPath, ok := cached.(string); ok {
			if _, err := os.Stat(cachedPath); err == nil {
				return cachedPath, nil
			}
		}
	}

	tmpFile, err := os.CreateTemp("", "converted-*.html")
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	var cmd *exec.Cmd

	if ext == ".rtf" || ext == ".webarchive" {
		cmd = exec.Command("textutil", "-convert", "html", sourcePath, "-output", tmpPath)
	} else {
		return "", fmt.Errorf("unsupported conversion type: %s", ext)
	}

	if err := cmd.Run(); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("conversion failed: %v", err)
	}

	cache.Store(sourcePath, tmpPath)
	return tmpPath, nil
}
