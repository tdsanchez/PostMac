package handlers

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/tdsanchez/PostMac/internal/config"
)

// HandleFile serves individual media files with proper MIME types
func HandleFile(w http.ResponseWriter, r *http.Request) {
	absPath := strings.TrimPrefix(r.URL.Path, "/file/")
	log.Printf("🔍 HandleFile: original path from URL: %s", absPath)

	// URL decode the path
	absPath, err := url.QueryUnescape(absPath)
	if err != nil {
		log.Printf("❌ HandleFile: URL decode failed: %v", err)
		http.Error(w, "Invalid path encoding", http.StatusBadRequest)
		return
	}
	log.Printf("🔍 HandleFile: after URL decode: %s", absPath)

	// Fix for browser normalizing /file//Volumes/ to /file/Volumes/ (or /file//Users/ to /file/Users/)
	// If path starts with "Volumes/" or "Users/" but isn't absolute, it's a file that lost its leading /
	if !filepath.IsAbs(absPath) && (strings.HasPrefix(absPath, "Volumes/") || strings.HasPrefix(absPath, "Users/")) {
		absPath = "/" + absPath
		log.Printf("🔍 HandleFile: restored leading /: %s", absPath)
	}

	cleanPath := filepath.Clean(absPath)

	file, err := os.Open(cleanPath)
	if err != nil {
		log.Printf("❌ HandleFile: os.Open failed: %v", err)
		http.NotFound(w, r)
		return
	}
	log.Printf("✅ HandleFile: successfully opened file: %s", cleanPath)
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ext := strings.ToLower(filepath.Ext(cleanPath))
	if config.TextExts[ext] {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		io.Copy(w, file)
		return
	}

	// Set explicit MIME types for media files
	switch ext {
	case ".mp4":
		w.Header().Set("Content-Type", "video/mp4")
	case ".mov":
		w.Header().Set("Content-Type", "video/quicktime")
	case ".m4v":
		w.Header().Set("Content-Type", "video/x-m4v")
	case ".avi":
		w.Header().Set("Content-Type", "video/x-msvideo")
	case ".mkv":
		w.Header().Set("Content-Type", "video/x-matroska")
	case ".webm":
		w.Header().Set("Content-Type", "video/webm")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	case ".webp":
		w.Header().Set("Content-Type", "image/webp")
	case ".pdf":
		w.Header().Set("Content-Type", "application/pdf")
	}

	http.ServeContent(w, r, stat.Name(), stat.ModTime(), file)
}
