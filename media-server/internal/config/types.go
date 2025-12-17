package config

import (
	"path/filepath"
	"strings"
)

// File type extension maps
var (
	SupportedExts = map[string]bool{
		".gif": true, ".jpg": true, ".jpeg": true, ".mp4": true,
		".png": true, ".tif": true, ".tiff": true, ".webp": true,
		".mov": true, ".avi": true, ".mkv": true, ".m4v": true,
		".pdf": true,
		".go": true, ".sh": true, ".mod": true, ".sum": true,
		".txt": true, ".md": true, ".json": true, ".yaml": true, ".yml": true,
		".html": true, ".htm": true, ".rtf": true, ".webarchive": true,
	}

	TextExts = map[string]bool{
		".go": true, ".sh": true, ".mod": true, ".sum": true,
		".txt": true, ".md": true, ".json": true, ".yaml": true, ".yml": true,
	}

	ConvertibleExts = map[string]bool{
		".rtf":        true,
		".webarchive": true,
	}

	HTMLExts = map[string]bool{
		".html": true,
		".htm":  true,
	}
)

// IsTextFile checks if a file is a text file based on extension
func IsTextFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return TextExts[ext]
}

// IsConvertibleFile checks if a file can be converted to HTML
func IsConvertibleFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ConvertibleExts[ext]
}

// IsHTMLFile checks if a file is an HTML file
func IsHTMLFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return HTMLExts[ext]
}

// GetFileTypeCategory returns the category for a file based on its extension
func GetFileTypeCategory(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	switch {
	case ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp" || ext == ".tif" || ext == ".tiff":
		return "📷 Images"
	case ext == ".mp4" || ext == ".mov" || ext == ".avi" || ext == ".mkv" || ext == ".m4v":
		return "🎬 Videos"
	case ext == ".pdf":
		return "📄 PDFs"
	case TextExts[ext]:
		return "📝 Text Files"
	case ext == ".html" || ext == ".htm":
		return "🌐 HTML"
	case ConvertibleExts[ext]:
		return "📃 Documents"
	default:
		return "📦 Other"
	}
}

// GetCategoryPriority returns priority order for categories
// Lower numbers = higher priority (sorted first)
func GetCategoryPriority(categoryName string) int {
	// Subdirectories (📁) come first
	if strings.HasPrefix(categoryName, "📁 ") {
		return 0
	}

	switch categoryName {
	case "All":
		return 1
	case "📷 Images":
		return 2
	case "🎥 Videos":
		return 3
	case "🎵 Audio":
		return 4
	case "Untagged":
		return 5
	default:
		// User tags have lowest priority
		return 10
	}
}
