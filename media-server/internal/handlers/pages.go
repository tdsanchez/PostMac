package handlers

import (
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

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

// Helper function map for templates
func getTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"hasSuffix": func(s, suffix string) bool {
			return strings.HasSuffix(strings.ToLower(s), strings.ToLower(suffix))
		},
		"urlEncode": func(s string) string {
			return url.QueryEscape(s)
		},
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
		"isTextFile":             config.IsTextFile,
		"isConvertibleFile":      config.IsConvertibleFile,
		"isHTMLFile":             config.IsHTMLFile,
		"parseFolderBreadcrumbs": parseFolderBreadcrumbs,
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

// HandleTag serves the gallery page for a specific tag
func HandleTag(w http.ResponseWriter, r *http.Request) {
	tag := strings.TrimPrefix(r.URL.Path, "/tag/")
	tag, _ = url.QueryUnescape(tag)

	// Lock-free state access (double-buffered)
	current := state.GetCurrent()
	filesByTag := current.FilesByTag
	files, ok := filesByTag[tag]

	// Get child folders if this is a folder category
	childFolders := getChildFolders(tag, filesByTag)

	if !ok {
		http.NotFound(w, r)
		return
	}

	// Pagination parameters
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 200 // Default page size
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	// Calculate pagination
	totalFiles := len(files)
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

	data := struct {
		Tag          string
		Files        []models.FileInfo
		Count        int
		TotalFiles   int
		ChildFolders []SubfolderInfo
		Page         int
		Limit        int
		TotalPages   int
		StartIdx     int
		EndIdx       int
	}{
		Tag:          tag,
		Files:        paginatedFiles,
		Count:        len(paginatedFiles),
		TotalFiles:   totalFiles,
		ChildFolders: childFolders,
		Page:         page,
		Limit:        limit,
		TotalPages:   totalPages,
		StartIdx:     startIdx + 1, // 1-indexed for display
		EndIdx:       endIdx,
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

// HandleViewer serves the single file viewer page
func HandleViewer(w http.ResponseWriter, r *http.Request) {
	// Extract tag from URL path
	tag := strings.TrimPrefix(r.URL.Path, "/view/")
	tag, _ = url.QueryUnescape(tag)

	// Get file path from query parameter
	filepath := r.URL.Query().Get("file")
	if filepath == "" {
		http.NotFound(w, r)
		return
	}
	filepath, _ = url.QueryUnescape(filepath)

	// Lock-free state access (double-buffered)
	current := state.GetCurrent()
	filesByTag := current.FilesByTag
	files, ok := filesByTag[tag]

	if !ok {
		http.NotFound(w, r)
		return
	}

	// Find the file in this category's file list
	index := -1
	for i, f := range files {
		if f.RelPath == filepath {
			index = i
			break
		}
	}

	if index == -1 {
		http.NotFound(w, r)
		return
	}

	currentFile := files[index]
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
	}{
		Tag:          tag,
		File:         currentFile,
		Index:        index + 1,
		Total:        len(files),
		PrevFile:     prevFile,
		NextFile:     nextFile,
		AllFilePaths: allFilePaths,
	}

	// Process the JavaScript template
	jsTmpl, err := template.New("viewerjs").Funcs(funcMap).Parse(jsTemplateStr)
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
		Tag        string
		File       models.FileInfo
		Index      int
		Total      int
		PrevFile   models.FileInfo
		NextFile   models.FileInfo
		JavaScript template.JS
	}{
		Tag:        tag,
		File:       currentFile,
		Index:      index + 1,
		Total:      len(files),
		PrevFile:   prevFile,
		NextFile:   nextFile,
		JavaScript: template.JS(processedJS),
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Template execute error: %v", err)
	}
}
