package handlers

import (
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	texttemplate "text/template"
	"time"

	"github.com/tdsanchez/PostMac/internal/config"
	"github.com/tdsanchez/PostMac/internal/models"
	"github.com/tdsanchez/PostMac/internal/state"
)

// BreadcrumbSegment represents a clickable segment in a breadcrumb trail
type BreadcrumbSegment struct {
	Label string
	URL   string
}

// parseFolderBreadcrumbs splits a folder category into breadcrumb segments
// e.g., "📁 Photos/2024/Vacation" -> [{Photos, /tag/📁 Photos}, {2024, /tag/📁 Photos/2024}, ...]
func parseFolderBreadcrumbs(tag string) []BreadcrumbSegment {
	// Check if this is a folder category
	if !strings.HasPrefix(tag, "📁 ") {
		// Not a folder, return single segment
		return []BreadcrumbSegment{{Label: tag, URL: "/tag/" + url.QueryEscape(tag)}}
	}

	// Remove the folder emoji prefix
	path := strings.TrimPrefix(tag, "📁 ")
	parts := strings.Split(path, "/")

	segments := make([]BreadcrumbSegment, 0, len(parts))
	currentPath := ""

	for i, part := range parts {
		if i == 0 {
			currentPath = part
		} else {
			currentPath = currentPath + "/" + part
		}

		segments = append(segments, BreadcrumbSegment{
			Label: part,
			URL:   "/tag/" + url.QueryEscape("📁 "+currentPath),
		})
	}

	return segments
}

// jsEscape escapes a string for use inside a JavaScript single-quoted string literal.
// Used with text/template (not html/template) so no HTML escaping occurs.
func jsEscape(s string) string {
	// Escape backslashes first, then single quotes, then other special chars
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}

// Helper function map for templates
func getTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"hasSuffix": func(s, suffix string) bool {
			return strings.HasSuffix(strings.ToLower(s), strings.ToLower(suffix))
		},
		"hasPrefix": func(s, prefix string) bool {
			return strings.HasPrefix(s, prefix)
		},
		"urlEncode": func(s string) string {
			return url.QueryEscape(s)
		},
		"jsEscape": jsEscape,
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"len": func(s []BreadcrumbSegment) int {
			return len(s)
		},
		"getDir": func(file models.FileInfo) string {
			// Extract directory from full path by removing the filename
			dir := filepath.Dir(file.Path)
			return dir
		},
		"formatFileSize": func(bytes int64) string {
			const unit = 1024
			if bytes < unit {
				return strconv.FormatInt(bytes, 10) + " B"
			}
			div, exp := int64(unit), 0
			for n := bytes / unit; n >= unit; n /= unit {
				div *= unit
				exp++
			}
			return strconv.FormatFloat(float64(bytes)/float64(div), 'f', 1, 64) + " " + "KMGTPE"[exp:exp+1] + "B"
		},
		"formatDate": func(t interface{}) string {
			// Handle time.Time
			switch v := t.(type) {
			case time.Time:
				if v.IsZero() {
					return "(no date)"
				}
				// Extract date components directly (no timezone conversion)
				year, month, day := v.Date()
				hour, min, _ := v.Clock()

				// Format manually to avoid timezone conversion issues
				ampm := "AM"
				displayHour := hour
				if hour >= 12 {
					ampm = "PM"
					if hour > 12 {
						displayHour = hour - 12
					}
				}
				if displayHour == 0 {
					displayHour = 12
				}

				monthNames := []string{"", "Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
				return fmt.Sprintf("%s %d, %d %d:%02d %s", monthNames[month], day, year, displayHour, min, ampm)
			default:
				// Return empty for unknown types
				return ""
			}
		},
		"dict": func(values ...interface{}) map[string]interface{} {
			// Helper to create a dict for passing multiple values to templates
			if len(values)%2 != 0 {
				return nil
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil
				}
				dict[key] = values[i+1]
			}
			return dict
		},
		"isTextFile":             config.IsTextFile,
		"isConvertibleFile":      config.IsConvertibleFile,
		"isHTMLFile":             config.IsHTMLFile,
		"parseFolderBreadcrumbs": parseFolderBreadcrumbs,
	}
}

// sortFiles sorts a slice of FileInfo in-place according to the specified mode and direction.
// This is extracted so both HandleTag (category view) and HandleViewer (single file view)
// use identical sorting, ensuring prev/next navigation matches the category sort order.
func sortFiles(files []models.FileInfo, sortMode string, sortReversed bool) {
	if sortMode == "" {
		return
	}

	switch sortMode {
	case "name":
		sort.Slice(files, func(i, j int) bool {
			result := strings.Compare(files[i].Name, files[j].Name)
			if sortReversed {
				return result > 0
			}
			return result < 0
		})
	case "date": // Legacy alias for os_birth
		sort.Slice(files, func(i, j int) bool {
			if sortReversed {
				return files[i].OSBirthTime.Before(files[j].OSBirthTime)
			}
			return files[i].OSBirthTime.After(files[j].OSBirthTime)
		})
	case "size":
		sort.Slice(files, func(i, j int) bool {
			if sortReversed {
				return files[i].Size < files[j].Size // Smallest first
			}
			return files[i].Size > files[j].Size // Largest first
		})
	case "os_birth":
		sort.Slice(files, func(i, j int) bool {
			if sortReversed {
				return files[i].OSBirthTime.Before(files[j].OSBirthTime) // Oldest first
			}
			return files[i].OSBirthTime.After(files[j].OSBirthTime) // Newest first
		})
	case "os_mod":
		sort.Slice(files, func(i, j int) bool {
			if sortReversed {
				return files[i].OSModTime.Before(files[j].OSModTime) // Oldest first
			}
			return files[i].OSModTime.After(files[j].OSModTime) // Newest first
		})
	case "exif_create":
		sort.Slice(files, func(i, j int) bool {
			if sortReversed {
				return files[i].EXIFCreateDate.Before(files[j].EXIFCreateDate) // Oldest first
			}
			return files[i].EXIFCreateDate.After(files[j].EXIFCreateDate) // Newest first
		})
	case "exif_modify":
		sort.Slice(files, func(i, j int) bool {
			if sortReversed {
				return files[i].EXIFModifyDate.Before(files[j].EXIFModifyDate) // Oldest first
			}
			return files[i].EXIFModifyDate.After(files[j].EXIFModifyDate) // Newest first
		})
	case "random":
		// Shuffle using Fisher-Yates algorithm
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(files), func(i, j int) {
			files[i], files[j] = files[j], files[i]
		})
	}
}

// groupFoldersByTopLevel groups folder categories by their top-level directory
// and returns only top-level folder categories for the index page
func groupFoldersByTopLevel(filesByTag map[string][]models.FileInfo) map[string][]models.FileInfo {
	topLevelFolders := make(map[string][]models.FileInfo)

	for tag, files := range filesByTag {
		if !strings.HasPrefix(tag, "📁 ") {
			continue
		}

		// Extract the path without the emoji
		path := strings.TrimPrefix(tag, "📁 ")

		// Get the top-level folder name
		parts := strings.Split(path, "/")
		topLevel := "📁 " + parts[0]

		// If this IS a top-level folder, use it directly
		if len(parts) == 1 {
			topLevelFolders[topLevel] = files
		} else {
			// Otherwise, aggregate files under the top-level folder
			topLevelFolders[topLevel] = append(topLevelFolders[topLevel], files...)
		}
	}

	return topLevelFolders
}

// HandleRoot serves the homepage with category previews
func HandleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Lock-free state access (double-buffered)
	current := state.GetCurrent()
	filesByTag := current.FilesByTag
	allFiles := current.AllFiles

	// Group folders by top-level for cleaner display
	topLevelFolders := groupFoldersByTopLevel(filesByTag)

	previews := []models.CategoryPreview{}

	// Add non-folder categories (All, Types, Tags)
	for tag, files := range filesByTag {
		if !strings.HasPrefix(tag, "📁 ") && len(files) > 0 {
			randomFile := files[rand.Intn(len(files))]
			previews = append(previews, models.CategoryPreview{
				Tag:         tag,
				Count:       len(files),
				PreviewFile: randomFile,
			})
		}
	}

	// Add top-level folder categories
	for tag, files := range topLevelFolders {
		if len(files) > 0 {
			randomFile := files[rand.Intn(len(files))]
			previews = append(previews, models.CategoryPreview{
				Tag:         tag,
				Count:       len(files),
				PreviewFile: randomFile,
			})
		}
	}

	// Sort by hierarchy first (All, Types, Folders, Tags), then by popularity
	sort.Slice(previews, func(i, j int) bool {
		priorityI := config.GetCategoryPriority(previews[i].Tag)
		priorityJ := config.GetCategoryPriority(previews[j].Tag)

		// If same priority level, sort by count (popularity)
		if priorityI == priorityJ {
			return previews[i].Count > previews[j].Count
		}

		// Otherwise sort by priority
		return priorityI < priorityJ
	})

	funcMap := getTemplateFuncs()

	// Read the index template from embedded file system
	templateContent, err := embeddedFiles.ReadFile("index_template.html")
	if err != nil {
		http.Error(w, "Template file not found", http.StatusInternalServerError)
		log.Printf("Error reading embedded index template file: %v", err)
		return
	}
	tmplStr := string(templateContent)

	tmpl, err := template.New("index").Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		log.Printf("Template parse error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Previews        []models.CategoryPreview
		TotalFiles      int
		TotalCategories int
	}{
		Previews:        previews,
		TotalFiles:      len(allFiles),
		TotalCategories: len(filesByTag),
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Template execute error: %v", err)
	}
}

// SubfolderInfo represents a child folder within the current folder
type SubfolderInfo struct {
	Name  string
	Path  string // Full category path like "📁 Photos/2024"
	Count int    // Number of files in this subfolder tree
}

// TreeNode represents a node in the folder hierarchy tree
type TreeNode struct {
	Name     string
	Path     string // Full category path like "📁 Photos/2024"
	Count    int    // Number of files in this folder (not recursive)
	Children []TreeNode
	Depth    int
}

// getChildFolders finds all immediate child folders of the current folder path
func getChildFolders(currentTag string, filesByTag map[string][]models.FileInfo) []SubfolderInfo {
	// Only process folder categories
	if !strings.HasPrefix(currentTag, "📁 ") {
		return nil
	}

	currentPath := strings.TrimPrefix(currentTag, "📁 ")
	childFolders := make(map[string]*SubfolderInfo)

	// Scan all folder categories to find children
	for tag, files := range filesByTag {
		if !strings.HasPrefix(tag, "📁 ") {
			continue
		}

		path := strings.TrimPrefix(tag, "📁 ")

		// Check if this path is a child of current path
		if strings.HasPrefix(path, currentPath+"/") {
			// Get the relative path from current folder
			relativePath := strings.TrimPrefix(path, currentPath+"/")
			parts := strings.Split(relativePath, "/")

			// Only immediate children (first segment after current path)
			childName := parts[0]
			childFullPath := "📁 " + currentPath + "/" + childName

			// Aggregate file counts for this child and all its descendants
			if existing, ok := childFolders[childName]; ok {
				existing.Count += len(files)
			} else {
				childFolders[childName] = &SubfolderInfo{
					Name:  childName,
					Path:  childFullPath,
					Count: len(files),
				}
			}
		}
	}

	// Convert map to slice and sort by name
	result := make([]SubfolderInfo, 0, len(childFolders))
	for _, folder := range childFolders {
		result = append(result, *folder)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// buildFolderTree builds a hierarchical tree of all folder categories
func buildFolderTree(filesByTag map[string][]models.FileInfo) []TreeNode {
	// Collect all folder paths
	folderPaths := make(map[string]int) // path -> file count
	for tag, files := range filesByTag {
		if strings.HasPrefix(tag, "📁 ") {
			path := strings.TrimPrefix(tag, "📁 ")
			folderPaths[path] = len(files)
		}
	}

	// Build tree structure
	rootNodes := make(map[string]*TreeNode)

	// Sort paths by depth (shallowest first) to build tree correctly
	paths := make([]string, 0, len(folderPaths))
	for path := range folderPaths {
		paths = append(paths, path)
	}
	sort.Slice(paths, func(i, j int) bool {
		depthI := strings.Count(paths[i], "/")
		depthJ := strings.Count(paths[j], "/")
		if depthI != depthJ {
			return depthI < depthJ
		}
		return paths[i] < paths[j]
	})

	// Build tree by inserting each path
	for _, path := range paths {
		parts := strings.Split(path, "/")
		depth := len(parts) - 1

		node := TreeNode{
			Name:     parts[len(parts)-1],
			Path:     "📁 " + path,
			Count:    folderPaths[path],
			Children: []TreeNode{},
			Depth:    depth,
		}

		if depth == 0 {
			// Root level folder
			rootNodes[parts[0]] = &node
		} else {
			// Find parent and add as child
			parentPath := strings.Join(parts[:len(parts)-1], "/")
			parent := findNodeByPath(rootNodes, parentPath)
			if parent != nil {
				parent.Children = append(parent.Children, node)
			}
		}
	}

	// Convert map to sorted slice
	result := make([]TreeNode, 0, len(rootNodes))
	for _, node := range rootNodes {
		result = append(result, *node)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// findNodeByPath recursively searches for a node by path
func findNodeByPath(nodes map[string]*TreeNode, targetPath string) *TreeNode {
	parts := strings.Split(targetPath, "/")
	rootKey := parts[0]

	if root, ok := nodes[rootKey]; ok {
		if len(parts) == 1 {
			return root
		}
		return findNodeInChildren(&root.Children, parts[1:], 1)
	}
	return nil
}

// findNodeInChildren recursively searches children for a path
func findNodeInChildren(children *[]TreeNode, pathParts []string, depth int) *TreeNode {
	for i := range *children {
		node := &(*children)[i]
		if node.Depth == depth && node.Name == pathParts[0] {
			if len(pathParts) == 1 {
				return node
			}
			return findNodeInChildren(&node.Children, pathParts[1:], depth+1)
		}
	}
	return nil
}

// HandleTag serves the gallery page for a specific tag
func HandleTag(w http.ResponseWriter, r *http.Request) {
	tag := strings.TrimPrefix(r.URL.Path, "/tag/")
	tag, _ = url.QueryUnescape(tag)

	// Lock-free state access (double-buffered)
	current := state.GetCurrent()
	filesByTag := current.FilesByTag

	var files []models.FileInfo
	var ok bool

	// Check if this is a synthetic category (search query)
	if strings.HasPrefix(tag, "🔍 ") {
		// Extract search query
		query := strings.TrimPrefix(tag, "🔍 ")

		// Execute search
		searchResults, err := HandleSearchQuery(query)
		if err != nil {
			log.Printf("Synthetic category search error: %v", err)
			http.Error(w, "Search query failed: "+err.Error(), http.StatusBadRequest)
			return
		}

		files = searchResults
		ok = true
	} else {
		// Normal tag lookup
		files, ok = filesByTag[tag]
	}

	// Get child folders if this is a folder category
	childFolders := getChildFolders(tag, filesByTag)

	if !ok {
		http.NotFound(w, r)
		return
	}

	// Sort parameters
	sortMode := r.URL.Query().Get("sort")
	sortReversed := r.URL.Query().Get("reversed") == "true"

	// CRITICAL: Copy the slice before sorting to avoid corrupting shared state
	// The files slice from filesByTag is a reference - sorting in place would mutate global state
	filesCopy := make([]models.FileInfo, len(files))
	copy(filesCopy, files)
	files = filesCopy

	// Apply server-side sorting BEFORE pagination
	sortFiles(files, sortMode, sortReversed)

	// Pagination parameters
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	// Calculate total files first (needed for unlimited pagination)
	totalFiles := len(files)

	limit := 200 // Default page size
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			if l == -1 {
				// Magic value: unlimited (show all files)
				limit = totalFiles
			} else if l > 0 && l <= 50000 {
				limit = l
			}
		}
	}

	// Calculate pagination
	totalPages := (totalFiles + limit - 1) / limit // Ceiling division

	// Clamp page to valid range
	if page > totalPages && totalPages > 0 {
		page = totalPages
	}

	// Slice files for current page
	startIdx := (page - 1) * limit
	endIdx := startIdx + limit
	if endIdx > totalFiles {
		endIdx = totalFiles
	}

	paginatedFiles := files
	if startIdx < totalFiles {
		paginatedFiles = files[startIdx:endIdx]
	} else {
		paginatedFiles = []models.FileInfo{}
	}

	funcMap := getTemplateFuncs()

	// Read the gallery template from embedded file system
	templateContent, err := embeddedFiles.ReadFile("gallery_template.html")
	if err != nil {
		http.Error(w, "Template file not found", http.StatusInternalServerError)
		log.Printf("Error reading embedded gallery template file: %v", err)
		return
	}
	tmplStr := string(templateContent)

	tmpl, err := template.New("tag").Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		log.Printf("Template parse error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Build folder tree for sidebar navigation
	folderTree := buildFolderTree(filesByTag)

	data := struct {
		Tag          string
		Files        []models.FileInfo
		Count        int
		TotalFiles   int
		ChildFolders []SubfolderInfo
		FolderTree   []TreeNode // NEW: For sidebar navigation
		Page         int
		Limit        int
		TotalPages   int
		StartIdx     int
		EndIdx       int
		SortMode     string
		SortReversed bool
	}{
		Tag:          tag,
		Files:        paginatedFiles,
		Count:        len(paginatedFiles),
		TotalFiles:   totalFiles,
		ChildFolders: childFolders,
		FolderTree:   folderTree,
		Page:         page,
		Limit:        limit,
		TotalPages:   totalPages,
		StartIdx:     startIdx + 1, // 1-indexed for display
		EndIdx:       endIdx,
		SortMode:     sortMode,
		SortReversed: sortReversed,
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Template execute error: %v", err)
	}
}

// HandleViewerJS serves the viewer JavaScript (currently minimal)
func HandleViewerJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	http.Error(w, "// JavaScript is rendered inline", http.StatusOK)
}

// fileExistsOnDisk checks if a file exists and is accessible on the filesystem
func fileExistsOnDisk(absPath string) bool {
	cleanPath := filepath.Clean(absPath)

	// Check if file exists
	info, err := os.Stat(cleanPath)
	if err != nil {
		return false
	}

	// Must be a regular file, not a directory
	return !info.IsDir()
}

// findValidFileWithFallback attempts to find a valid file, falling back to next files if needed
// Returns the valid file info, its index, and whether a valid file was found
func findValidFileWithFallback(requestedPath string, files []models.FileInfo, requestedIndex int) (models.FileInfo, int, bool) {
	// First, try the requested file
	if requestedIndex >= 0 && requestedIndex < len(files) {
		requestedFile := files[requestedIndex]
		if fileExistsOnDisk(requestedFile.Path) {
			return requestedFile, requestedIndex, true
		}

		// Requested file doesn't exist on disk - log for RCA
		log.Printf("⚠️ MISSING FILE: path=%s (file exists in cache but not on disk)", requestedPath)
	}

	// File missing or invalid - try to find next valid file
	maxAttempts := 10
	for i := 1; i <= maxAttempts; i++ {
		// Try next file in sequence
		nextIndex := (requestedIndex + i) % len(files)
		if nextIndex < 0 || nextIndex >= len(files) {
			continue
		}

		nextFile := files[nextIndex]
		if fileExistsOnDisk(nextFile.Path) {
			// Found valid file - log the fallback
			log.Printf("✅ FALLBACK: requested=%s served=%s (skipped %d files)", requestedPath, nextFile.Path, i)
			return nextFile, nextIndex, true
		}

		log.Printf("⚠️ SKIPPED MISSING: path=%s (attempt %d/%d)", nextFile.Path, i, maxAttempts)
	}

	// No valid file found after max attempts
	log.Printf("❌ NO VALID FILES: requested=%s (checked %d files, all missing)", requestedPath, maxAttempts)
	return models.FileInfo{}, -1, false
}

// HandleViewer serves the single file viewer page
func HandleViewer(w http.ResponseWriter, r *http.Request) {
	// Extract tag from URL path
	tag := strings.TrimPrefix(r.URL.Path, "/view/")
	tag, _ = url.QueryUnescape(tag)
	// log.Printf("🔍 HandleViewer: tag=%s", tag)

	// Get file path from query parameter
	filepath := r.URL.Query().Get("file")
	if filepath == "" {
		log.Printf("❌ HandleViewer: no file parameter")
		http.NotFound(w, r)
		return
	}
	filepath, _ = url.QueryUnescape(filepath)
	log.Printf("🔍 HandleViewer: requested file=%s", filepath)

	// Lock-free state access (double-buffered)
	current := state.GetCurrent()
	filesByTag := current.FilesByTag

	var files []models.FileInfo
	var ok bool

	// Check if this is a synthetic category (search query)
	if strings.HasPrefix(tag, "🔍 ") {
		// Extract search query
		query := strings.TrimPrefix(tag, "🔍 ")

		// Execute search
		searchResults, err := HandleSearchQuery(query)
		if err != nil {
			log.Printf("Synthetic category search error in viewer: %v", err)
			http.Error(w, "Search query failed: "+err.Error(), http.StatusBadRequest)
			return
		}

		files = searchResults
		ok = true
	} else {
		// Normal tag lookup
		files, ok = filesByTag[tag]
	}

	if !ok {
		log.Printf("❌ HandleViewer: tag not found: %s", tag)
		http.NotFound(w, r)
		return
	}
	log.Printf("🔍 HandleViewer: found %d files in tag", len(files))

	// Sort parameters - must match category view sorting for consistent prev/next navigation
	sortMode := r.URL.Query().Get("sort")
	sortReversed := r.URL.Query().Get("reversed") == "true"

	// CRITICAL: Copy the slice before sorting to avoid corrupting shared state
	// The files slice from filesByTag is a reference - sorting in place would mutate global state
	filesCopy := make([]models.FileInfo, len(files))
	copy(filesCopy, files)
	files = filesCopy

	// Apply sorting to get same order as category view
	sortFiles(files, sortMode, sortReversed)

	// Find the file in this category's file list (now in sorted order)
	requestedIndex := -1
	for i, f := range files {
		if f.Path == filepath {
			requestedIndex = i
			break
		}
	}
	log.Printf("🔍 HandleViewer: requestedIndex=%d", requestedIndex)

	// Validate file and fallback to next valid file if needed
	currentFile, index, found := findValidFileWithFallback(filepath, files, requestedIndex)
	if !found {
		// No valid files found after checking multiple candidates
		log.Printf("❌ HandleViewer: no valid files found")
		http.Error(w, "No valid files available in this category", http.StatusNotFound)
		return
	}
	log.Printf("✅ HandleViewer: using file at index %d/%d: %s", index, len(files), currentFile.Path)
	prevIndex := index - 1
	nextIndex := index + 1
	if prevIndex < 0 {
		prevIndex = len(files) - 1
	}
	if nextIndex >= len(files) {
		nextIndex = 0
	}

	prevFile := files[prevIndex]
	nextFile := files[nextIndex]
	log.Printf("🔍 HandleViewer: prevFile[%d]=%s, nextFile[%d]=%s", prevIndex, prevFile.Path, nextIndex, nextFile.Path)

	funcMap := getTemplateFuncs()

	// Read the HTML template from embedded file system
	templateContent, err := embeddedFiles.ReadFile("main_template.html")
	if err != nil {
		http.Error(w, "Template file not found", http.StatusInternalServerError)
		log.Printf("Error reading embedded template file: %v", err)
		return
	}
	tmplStr := string(templateContent)

	// Read and process the JavaScript template
	jsContent, err := embeddedFiles.ReadFile("main_template.js")
	if err != nil {
		http.Error(w, "JavaScript template file not found", http.StatusInternalServerError)
		log.Printf("Error reading embedded JavaScript file: %v", err)
		return
	}
	jsTemplateStr := string(jsContent)

	// Send empty array - client will fetch from /api/filelist if needed
	// This eliminates the template serialization bottleneck for large categories (100k+ files)
	// Client-side localStorage cache (populated by gallery) will be used when available
	// Otherwise, viewer will fetch on-demand via /api/filelist when random mode is activated
	allFilePaths := []string{}

	// Create a temporary data structure for JS template processing
	jsData := struct {
		Tag          string
		File         models.FileInfo
		Index        int
		Total        int
		PrevFile     models.FileInfo
		NextFile     models.FileInfo
		AllFilePaths []string
		SortMode     string
		SortReversed bool
	}{
		Tag:          tag,
		File:         currentFile,
		Index:        index + 1,
		Total:        len(files),
		PrevFile:     prevFile,
		NextFile:     nextFile,
		AllFilePaths: allFilePaths,
		SortMode:     sortMode,
		SortReversed: sortReversed,
	}

	// Process the JavaScript template using text/template (no HTML escaping)
	jsFuncMap := texttemplate.FuncMap{
		"jsEscape":         jsEscape,
		"isTextFile":       config.IsTextFile,
		"isConvertibleFile": config.IsConvertibleFile,
	}
	jsTmpl, err := texttemplate.New("viewerjs").Funcs(jsFuncMap).Parse(jsTemplateStr)
	if err != nil {
		log.Printf("JavaScript template parse error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var jsBuffer strings.Builder
	if err := jsTmpl.Execute(&jsBuffer, jsData); err != nil {
		log.Printf("JavaScript template execute error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	processedJS := jsBuffer.String()

	tmpl, err := template.New("viewer").Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		log.Printf("Template parse error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Tag          string
		File         models.FileInfo
		Index        int
		Total        int
		PrevFile     models.FileInfo
		NextFile     models.FileInfo
		JavaScript   template.JS
		SortMode     string
		SortReversed bool
	}{
		Tag:          tag,
		File:         currentFile,
		Index:        index + 1,
		Total:        len(files),
		PrevFile:     prevFile,
		NextFile:     nextFile,
		JavaScript:   template.JS(processedJS),
		SortMode:     sortMode,
		SortReversed: sortReversed,
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Template execute error: %v", err)
	}
}

// HandleTraining serves the ML training page for date correction
func HandleTraining(w http.ResponseWriter, r *http.Request) {
	// Get current state
	currentState := state.GetCurrent()

	// Get files from "📅 Needs Date Correction" category
	files, exists := currentState.FilesByTag["📅 Needs Date Correction"]
	if !exists || len(files) == 0 {
		http.Error(w, "No files need date correction", http.StatusNotFound)
		return
	}

	// Get index from query parameter (default to 1)
	indexStr := r.URL.Query().Get("index")
	index := 1
	if indexStr != "" {
		if parsedIndex, err := strconv.Atoi(indexStr); err == nil && parsedIndex >= 1 && parsedIndex <= len(files) {
			index = parsedIndex
		}
	}

	// Get file at index (1-based)
	file := files[index-1]

	// Read the training template
	templateContent, err := embeddedFiles.ReadFile("train_template.html")
	if err != nil {
		http.Error(w, "Template file not found", http.StatusInternalServerError)
		return
	}

	// Parse template with formatDate function
	tmpl, err := template.New("train").Funcs(template.FuncMap{
		"formatDate": func(t interface{}) string {
			switch v := t.(type) {
			case time.Time:
				if v.IsZero() {
					return "(no date)"
				}
				// Extract date components directly (no timezone conversion)
				year, month, day := v.Date()
				hour, min, _ := v.Clock()

				// Format manually to avoid timezone conversion issues
				ampm := "AM"
				displayHour := hour
				if hour >= 12 {
					ampm = "PM"
					if hour > 12 {
						displayHour = hour - 12
					}
				}
				if displayHour == 0 {
					displayHour = 12
				}

				monthNames := []string{"", "Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
				return fmt.Sprintf("%s %d, %d %d:%02d %s", monthNames[month], day, year, displayHour, min, ampm)
			default:
				return ""
			}
		},
	}).Parse(string(templateContent))

	if err != nil {
		log.Printf("Training template parse error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Check for existing decision
	existingDecision := "not_chosen"
	if dbCache := state.GetCache(); dbCache != nil {
		if decision, exists, err := dbCache.GetDateDecision(file.Path); err == nil && exists {
			existingDecision = decision
		}
	}

	// Prepare data
	data := struct {
		File             models.FileInfo
		Index            int
		Total            int
		ExistingDecision string
	}{
		File:             file,
		Index:            index,
		Total:            len(files),
		ExistingDecision: existingDecision,
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Training template execute error: %v", err)
	}
}
