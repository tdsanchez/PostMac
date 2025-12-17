package handlers

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/tdsanchez/PostMac/internal/config"
	"github.com/tdsanchez/PostMac/internal/state"
)

// HandleFile serves individual media files with proper MIME types
func HandleFile(w http.ResponseWriter, r *http.Request) {
	relPath := strings.TrimPrefix(r.URL.Path, "/file/")
	serveDir := state.GetServeDir()
	fullPath := filepath.Join(serveDir, relPath)

	cleanPath := filepath.Clean(fullPath)
	if !strings.HasPrefix(cleanPath, serveDir) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	file, err := os.Open(cleanPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
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
