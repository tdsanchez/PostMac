package handlers

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/tdsanchez/PostMac/media-server/internal/cache"
	"github.com/tdsanchez/PostMac/media-server/internal/config"
	"github.com/tdsanchez/PostMac/media-server/internal/conversion"
	"github.com/tdsanchez/PostMac/media-server/internal/metadata"
	"github.com/tdsanchez/PostMac/media-server/internal/models"
	"github.com/tdsanchez/PostMac/media-server/internal/persistence"
	"github.com/tdsanchez/PostMac/media-server/internal/scanner"
	"github.com/tdsanchez/PostMac/media-server/internal/state"
)

var embeddedFiles embed.FS

// SetEmbeddedFiles sets the embedded file system for handlers
func SetEmbeddedFiles(fs embed.FS) {
	embeddedFiles = fs
}

// HandleAddTag handles adding a tag to a single file
func HandleAddTag(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var op models.TagOperation
	if err := json.NewDecoder(r.Body).Decode(&op); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get current tags from in-memory data (NOT from disk) - lock-free
	current := state.GetCurrent()
	allFiles := current.AllFiles
	var currentTags []string
	for i := range allFiles {
		if allFiles[i].RelPath == op.FilePath {
			currentTags = make([]string, len(allFiles[i].Tags))
			copy(currentTags, allFiles[i].Tags)
			break
		}
	}

	// Check if tag already exists
	for _, t := range currentTags {
		if t == op.Tag {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "tags": currentTags})
			return
		}
	}

	// Add new tag
	newTags := append(currentTags, op.Tag)

	// Update in-memory data (instant UI response)
	scanner.UpdateFileTagsInMemory(op.FilePath, newTags)

	// Queue disk write for batched persistence (relative path)
	persistence.QueueDiskWrite(op.FilePath, newTags)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "tags": newTags})
}

// HandleBatchAddTag handles adding a tag to multiple files
func HandleBatchAddTag(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var op models.BatchTagOperation
	if err := json.NewDecoder(r.Body).Decode(&op); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	successCount := 0
	// Lock-free state access (double-buffered)
	current := state.GetCurrent()
	allFiles := current.AllFiles

	for _, relPath := range op.FilePaths {
		// Get current tags from in-memory data
		var currentTags []string
		for i := range allFiles {
			if allFiles[i].RelPath == relPath {
				currentTags = make([]string, len(allFiles[i].Tags))
				copy(currentTags, allFiles[i].Tags)
				break
			}
		}

		// Check if tag already exists
		tagExists := false
		for _, t := range currentTags {
			if t == op.Tag {
				tagExists = true
				break
			}
		}

		if !tagExists {
			// Add new tag
			newTags := append(currentTags, op.Tag)

			// Update in-memory data
			scanner.UpdateFileTagsInMemory(relPath, newTags)

			// Queue disk write for later (use relative path)
			persistence.QueueDiskWrite(relPath, newTags)
		}
		successCount++
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"count":   successCount,
	})
}

// HandleRemoveTag handles removing a tag from a file
func HandleRemoveTag(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var op models.TagOperation
	if err := json.NewDecoder(r.Body).Decode(&op); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get current tags from in-memory data - lock-free
	current := state.GetCurrent()
	allFiles := current.AllFiles
	var currentTags []string
	for i := range allFiles {
		if allFiles[i].RelPath == op.FilePath {
			currentTags = make([]string, len(allFiles[i].Tags))
			copy(currentTags, allFiles[i].Tags)
			break
		}
	}

	// Remove the tag
	newTags := []string{}
	for _, t := range currentTags {
		if t != op.Tag {
			newTags = append(newTags, t)
		}
	}

	// Update in-memory data
	scanner.UpdateFileTagsInMemory(op.FilePath, newTags)

	// Queue disk write for later (use relative path)
	persistence.QueueDiskWrite(op.FilePath, newTags)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "tags": newTags})
}

// HandleGetAllTags returns all available tags
func HandleGetAllTags(w http.ResponseWriter, r *http.Request) {
	// Lock-free state access (double-buffered)
	current := state.GetCurrent()
	allTags := current.AllTags

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(allTags)
}

// HandleGetFileList returns all file paths for a given category
func HandleGetFileList(w http.ResponseWriter, r *http.Request) {
	// Get category from query parameter
	category := r.URL.Query().Get("category")
	if category == "" {
		http.Error(w, "Missing category parameter", http.StatusBadRequest)
		return
	}

	// Decode URL-encoded category name
	decodedCategory, err := url.QueryUnescape(category)
	if err != nil {
		http.Error(w, "Invalid category parameter", http.StatusBadRequest)
		return
	}

	// Get files for the category - lock-free
	current := state.GetCurrent()
	filesByTag := current.FilesByTag
	files, exists := filesByTag[decodedCategory]

	if !exists {
		http.Error(w, "Category not found", http.StatusNotFound)
		return
	}

	// Build array of relative paths
	filePaths := make([]string, len(files))
	for i, f := range files {
		filePaths[i] = f.RelPath
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filePaths)
}

// HandleUpdateComment updates a Finder comment for a file
func HandleUpdateComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		FilePath string `json:"filepath"`
		Comment  string `json:"comment"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Construct full path
	serveDir := state.GetServeDir()
	fullPath := filepath.Join(serveDir, req.FilePath)

	// Update comment on disk immediately
	if err := scanner.SetMacOSComment(fullPath, req.Comment); err != nil {
		log.Printf("Error setting comment for %s: %v", fullPath, err)
		http.Error(w, "Failed to update comment", http.StatusInternalServerError)
		return
	}

	// Update comment in cache
	if cache := state.GetCache(); cache != nil {
		if err := cache.UpdateFileComment(req.FilePath, req.Comment); err != nil {
			log.Printf("Warning: Failed to update cache for %s: %v", req.FilePath, err)
		}
	}

	// Update comment in memory across ALL categories
	state.LockData()
	defer state.UnlockData()

	allFiles := state.GetAllFiles()
	filesByTag := state.GetFilesByTag()

	// Update in allFiles
	for i := range allFiles {
		if allFiles[i].RelPath == req.FilePath {
			allFiles[i].Comment = req.Comment
		}
	}

	// Update in ALL categories (including subdirectory and type categories)
	for categoryName, files := range filesByTag {
		for j := range files {
			if files[j].RelPath == req.FilePath {
				filesByTag[categoryName][j].Comment = req.Comment
			}
		}
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Comment updated successfully",
	})
}

// HandleShutdown gracefully shuts down the server
func HandleShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Println("🛑 Shutdown requested from UI")

	// Send success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Server shutting down...",
	})

	// Graceful shutdown after response sent
	go func() {
		time.Sleep(500 * time.Millisecond)
		log.Println("👋 Server shutdown complete")
		os.Exit(0)
	}()
}

// HandleRescan triggers an incremental scan in the background
func HandleRescan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if scan is already in progress
	if state.IsScanning() {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Scan already in progress",
		})
		return
	}

	log.Println("🔄 Rescan requested from UI")

	// Send immediate response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Scan started...",
	})

	// Trigger scan in background
	go func() {
		serveDir := state.GetServeDir()
		dbCache := state.GetCache()

		state.SetScanning(true)
		log.Println("📊 Starting incremental scan...")

		// Perform scan
		if err := scanner.ScanDirectory(serveDir); err != nil {
			log.Printf("❌ Scan failed: %v", err)
			state.SetScanning(false)
			return
		}

		// Save to cache
		if dbCache != nil {
			scanner.SaveToCache(dbCache.(*cache.Cache))
		}

		state.SetScanCompleted()
		log.Println("✅ Scan completed")
	}()
}

// HandleScanStatus returns the current scan state
func HandleScanStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	isScanning, completed := state.GetScanState()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"isScanning": isScanning,
		"completed":  completed,
	})

	// Auto-clear completed flag after UI reads it
	if completed {
		state.ClearScanCompleted()
	}
}

// HandleDeleteFile moves a file to Trash and removes it from cache
func HandleDeleteFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		FilePath string `json:"filePath"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	serveDir := state.GetServeDir()
	fullPath := filepath.Join(serveDir, req.FilePath)
	cleanPath := filepath.Clean(fullPath)

	// Security check: ensure path is within serve directory
	if !strings.HasPrefix(cleanPath, serveDir) {
		log.Printf("❌ Delete forbidden: path outside serve directory: %s", cleanPath)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Forbidden: path outside serve directory",
		})
		return
	}

	// Check if file exists
	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		log.Printf("❌ Delete failed: file not found: %s", cleanPath)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "File not found",
		})
		return
	}

	// Move file to Trash using osascript (macOS-specific)
	script := fmt.Sprintf(`
		set filepath to POSIX file "%s"
		tell application "Finder"
			move filepath to trash
		end tell
	`, cleanPath)

	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		log.Printf("❌ Delete failed: osascript error: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to move file to Trash",
		})
		return
	}

	log.Printf("🗑️ File moved to Trash: %s", req.FilePath)

	// Remove file from in-memory state and cache
	scanner.RemoveFileFromMemory(req.FilePath)

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "File moved to Trash",
	})
}

// HandleMetadata returns EXIF and file metadata
func HandleMetadata(w http.ResponseWriter, r *http.Request) {
	relPath := strings.TrimPrefix(r.URL.Path, "/api/metadata/")
	relPath, _ = url.QueryUnescape(relPath)

	serveDir := state.GetServeDir()
	fullPath := filepath.Join(serveDir, relPath)

	cleanPath := filepath.Clean(fullPath)
	if !strings.HasPrefix(cleanPath, serveDir) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	meta, err := metadata.GetFileMetadata(cleanPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(meta)
}

// HandleQuickLook opens a file in QuickLook and reveals in Finder
func HandleQuickLook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.RevealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	serveDir := state.GetServeDir()
	fullPath := filepath.Join(serveDir, req.FilePath)
	cleanPath := filepath.Clean(fullPath)

	if !strings.HasPrefix(cleanPath, serveDir) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	go func() {

		cmd := exec.Command("sh", "-c", fmt.Sprintf("open -R '%s' > /dev/null 2>&1", cleanPath))
		cmd2 := exec.Command("qlmanage", "-p", cleanPath)
		
		cmd.Run()
		time.Sleep(500 * time.Millisecond)

		cmd2.Run()
		time.Sleep(1500 * time.Millisecond)

		script := `tell application "System Events" to set frontmost of first process whose name is "qlmanage" to true`
        activateCmd := exec.Command("osascript", "-e", script)
        activateCmd.Run()
        time.Sleep(1500 * time.Millisecond)
        
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// HandleConvert converts RTF or WebArchive files to HTML
func HandleConvert(w http.ResponseWriter, r *http.Request) {
	relPath := strings.TrimPrefix(r.URL.Path, "/api/convert/")
	relPath, _ = url.QueryUnescape(relPath)

	serveDir := state.GetServeDir()
	fullPath := filepath.Join(serveDir, relPath)

	cleanPath := filepath.Clean(fullPath)
	if !strings.HasPrefix(cleanPath, serveDir) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	ext := strings.ToLower(filepath.Ext(cleanPath))
	if !config.ConvertibleExts[ext] {
		http.Error(w, "File type not convertible", http.StatusBadRequest)
		return
	}

	htmlPath, err := conversion.ConvertToHTML(cleanPath)
	if err != nil {
		log.Printf("Conversion error: %v", err)
		http.Error(w, fmt.Sprintf("Conversion failed: %v", err), http.StatusInternalServerError)
		return
	}

	htmlFile, err := os.Open(htmlPath)
	if err != nil {
		http.Error(w, "Failed to open converted file", http.StatusInternalServerError)
		return
	}
	defer htmlFile.Close()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.Copy(w, htmlFile)
}
