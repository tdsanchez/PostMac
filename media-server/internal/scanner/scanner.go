package scanner

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/tdsanchez/PostMac/media-server/internal/cache"
	"github.com/tdsanchez/PostMac/media-server/internal/config"
	"github.com/tdsanchez/PostMac/media-server/internal/models"
	"github.com/tdsanchez/PostMac/media-server/internal/state"
)

// ScanDirectory scans the serve directory using double-buffered state (no blocking)
func ScanDirectory(serveDir string) error {
	// Get inactive state buffer
	inactive := state.GetInactiveState()

	// Build into inactive state (no locks held!)
	if err := ScanDirectoryInto(serveDir, inactive); err != nil {
		return err
	}

	// Atomic swap when complete
	state.SwapState(inactive)
	return nil
}

// ScanDirectoryInto scans the serve directory and populates the provided state
// This function does NOT acquire locks - it builds into the provided state buffer
func ScanDirectoryInto(serveDir string, targetState *state.AppState) error {
	// Initialize target state
	targetState.FilesByTag = make(map[string][]models.FileInfo)
	targetState.AllFiles = make([]models.FileInfo, 0)
	tagSet := make(map[string]bool)
	filesByTag := targetState.FilesByTag

	// Track files by directory path for hierarchical categories
	filesByDir := make(map[string][]models.FileInfo)
	rootFiles := []models.FileInfo{}

	// Recursive walker function
	err := filepath.Walk(serveDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error accessing path %s: %v", path, err)
			return nil // Continue walking despite errors
		}

		// Skip directories themselves (we'll process them later)
		if info.IsDir() {
			// Skip hidden directories
			if info.Name() != filepath.Base(serveDir) && strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file type is supported
		ext := strings.ToLower(filepath.Ext(path))
		if !config.SupportedExts[ext] {
			return nil
		}

		// Get relative path and directory
		relPath, _ := filepath.Rel(serveDir, path)
		relDir := filepath.Dir(relPath)

		// Get tags and metadata
		tags := GetMacOSTags(path)
		comment := GetMacOSComment(path)

		fileInfo := models.FileInfo{
			Name:    info.Name(),
			Path:    path,
			RelPath: relPath,
			Tags:    tags,
			Comment: comment,
			Created: getBirthTime(info),
			Size:    info.Size(),
		}

		// Add to master list
		targetState.AllFiles = append(targetState.AllFiles, fileInfo)

		// Track by directory for hierarchical categories
		if relDir == "." {
			// Root level file
			rootFiles = append(rootFiles, fileInfo)
		} else {
			// Subdirectory file
			filesByDir[relDir] = append(filesByDir[relDir], fileInfo)
		}

		// Add to file type category
		typeCategory := config.GetFileTypeCategory(info.Name())
		filesByTag[typeCategory] = append(filesByTag[typeCategory], fileInfo)

		// Add to tag categories
		if len(tags) == 0 {
			filesByTag["Untagged"] = append(filesByTag["Untagged"], fileInfo)
		} else {
			for _, tag := range tags {
				// Skip if tag matches the type category to avoid duplicates
				if tag != typeCategory {
					filesByTag[tag] = append(filesByTag[tag], fileInfo)
					tagSet[tag] = true
				}
			}
		}

		return nil
	})

	if err != nil {
		log.Printf("Error walking directory: %v", err)
		return err
	}

	// Create "All" category from all files in the library
	allFilesList := targetState.AllFiles
	sort.Slice(allFilesList, func(i, j int) bool {
		return allFilesList[i].Created.After(allFilesList[j].Created)
	})
	filesByTag["All"] = allFilesList

	// Create hierarchical folder categories
	for dirPath, files := range filesByDir {
		// Sort files by creation time
		sort.Slice(files, func(i, j int) bool {
			return files[i].Created.After(files[j].Created)
		})

		// Create category name with folder emoji
		// Use the relative path to show hierarchy
		categoryName := "📁 " + dirPath
		filesByTag[categoryName] = files
	}

	// Create parent folder aggregations for ALL levels (not just top-level)
	// This ensures intermediate folders without direct files still have valid categories
	parentAggregates := make(map[string][]models.FileInfo)
	for dirPath, files := range filesByDir {
		parts := strings.Split(dirPath, "/")

		// Build aggregations for each parent level
		// e.g., for "a/b/c/d", create aggregations for "a", "a/b", and "a/b/c"
		for i := 1; i < len(parts); i++ {
			parentPath := "📁 " + strings.Join(parts[:i], "/")
			parentAggregates[parentPath] = append(parentAggregates[parentPath], files...)
		}
	}

	// Add or merge parent aggregations into filesByTag
	for parentPath, aggregatedFiles := range parentAggregates {
		if existing, ok := filesByTag[parentPath]; ok {
			// Parent folder exists with its own files, append aggregated files from subdirs
			filesByTag[parentPath] = append(existing, aggregatedFiles...)
		} else {
			// No files directly in parent, just use aggregated files from subdirs
			filesByTag[parentPath] = aggregatedFiles
		}

		// Sort the combined list
		sort.Slice(filesByTag[parentPath], func(i, j int) bool {
			return filesByTag[parentPath][i].Created.After(filesByTag[parentPath][j].Created)
		})
	}

	// Sort all other categories by creation time
	for tag := range filesByTag {
		if tag != "All" && !strings.HasPrefix(tag, "📁") {
			files := filesByTag[tag]
			sort.Slice(files, func(i, j int) bool {
				return files[i].Created.After(files[j].Created)
			})
		}
	}

	// Build tag list
	allTags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		allTags = append(allTags, tag)
	}
	sort.Strings(allTags)
	targetState.AllTags = allTags

	return nil
}

// UpdateFileTagsInMemory updates the in-memory data structures when tags change
func UpdateFileTagsInMemory(relPath string, newTags []string) {
	state.LockData()
	defer state.UnlockData()

	allFiles := state.GetAllFiles()
	filesByTag := state.GetFilesByTag()

	// Update allFiles master list
	for i := range allFiles {
		if allFiles[i].RelPath == relPath {
			oldTags := allFiles[i].Tags
			allFiles[i].Tags = newTags

			// Remove file from old tag categories in inverted index
			for _, oldTag := range oldTags {
				if files, ok := filesByTag[oldTag]; ok {
					for j, f := range files {
						if f.RelPath == relPath {
							filesByTag[oldTag] = append(files[:j], files[j+1:]...)
							break
						}
					}
				}
			}

			// Add file to new tag categories in inverted index
			for _, newTag := range newTags {
				filesByTag[newTag] = append(filesByTag[newTag], allFiles[i])
			}

			// Update tags in ALL other categories (including subdirectory and type categories)
			for categoryName, files := range filesByTag {
				for j := range files {
					if files[j].RelPath == relPath {
						filesByTag[categoryName][j].Tags = newTags
					}
				}
			}

			// Rebuild allTags list (excluding system categories like "All", type icons, subdirs)
			tagSet := make(map[string]bool)
			for tag := range filesByTag {
				if tag != "All" && tag != "Untagged" && !strings.HasPrefix(tag, "📷") && !strings.HasPrefix(tag, "🎬") && !strings.HasPrefix(tag, "📄") && !strings.HasPrefix(tag, "📝") && !strings.HasPrefix(tag, "🌐") && !strings.HasPrefix(tag, "📃") && !strings.HasPrefix(tag, "📦") && !strings.HasPrefix(tag, "📁") {
					tagSet[tag] = true
				}
			}
			allTags := make([]string, 0, len(tagSet))
			for tag := range tagSet {
				allTags = append(allTags, tag)
			}
			sort.Strings(allTags)
			state.SetAllTags(allTags)

			break
		}
	}
}

// RemoveFileFromMemory removes a file from all in-memory data structures and cache
func RemoveFileFromMemory(relPath string) {
	state.LockData()
	defer state.UnlockData()

	allFiles := state.GetAllFiles()
	filesByTag := state.GetFilesByTag()

	// Remove from allFiles master list
	for i := range allFiles {
		if allFiles[i].RelPath == relPath {
			// Remove from slice
			allFiles = append(allFiles[:i], allFiles[i+1:]...)
			state.SetAllFiles(allFiles)
			break
		}
	}

	// Remove from all tag categories in filesByTag
	for categoryName, files := range filesByTag {
		for j := range files {
			if files[j].RelPath == relPath {
				filesByTag[categoryName] = append(files[:j], files[j+1:]...)
				break
			}
		}
	}

	// Remove from cache if available
	cache := state.GetCache()
	if cache != nil {
		if err := cache.DeleteFile(relPath); err != nil {
			log.Printf("⚠️ Warning: Failed to remove file from cache: %v", err)
		}
	}
}

// getBirthTime gets the creation time (birth time) of a file on macOS
func getBirthTime(info os.FileInfo) time.Time {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		// Fallback to ModTime if we can't get birth time
		return info.ModTime()
	}
	// On macOS, Birthtimespec contains the file creation time
	return time.Unix(stat.Birthtimespec.Sec, stat.Birthtimespec.Nsec)
}

// LoadOrScanDirectory loads file index from cache or performs full scan
func LoadOrScanDirectory(serveDir string) (*cache.Cache, error) {
	log.Println("🔍 Checking for cache...")
	
	// Open or create cache
	c, err := cache.New(serveDir)
	if err != nil {
		log.Printf("⚠️  Failed to open cache, performing full scan: %v", err)
		if err := ScanDirectory(serveDir); err != nil {
			return nil, err
		}
		return c, nil
	}

	// Check if we have a recent cache
	lastScan, totalFiles, totalTags, err := c.GetScanMetadata()
	if err != nil {
		log.Printf("⚠️  Failed to read cache metadata: %v", err)
		if err := ScanDirectory(serveDir); err != nil {
			return nil, err
		}
		// Save to cache after scan
		SaveToCache(c)
		return c, nil
	}

	// If cache is empty or very old (> 7 days), do a full scan
	cacheAge := time.Since(lastScan)
	if totalFiles == 0 || cacheAge > 7*24*time.Hour {
		log.Printf("📊 Cache is %v old or empty, performing full scan...", cacheAge.Round(time.Second))
		if err := ScanDirectory(serveDir); err != nil {
			return nil, err
		}
		// Save to cache after scan
		SaveToCache(c)
		return c, nil
	}

	// Load from cache
	log.Printf("⚡ Loading from cache (last scan: %v ago, %d files, %d tags)...",
		cacheAge.Round(time.Second), totalFiles, totalTags)

	files, err := c.LoadFiles()
	if err != nil {
		log.Printf("⚠️  Failed to load from cache: %v", err)
		log.Println("📊 Performing full scan...")
		if err := ScanDirectory(serveDir); err != nil {
			return nil, err
		}
		SaveToCache(c)
		return c, nil
	}

	// Reconstruct full paths from relative paths
	for i := range files {
		files[i].Path = filepath.Join(serveDir, files[i].RelPath)
	}

	// Get inactive state buffer
	inactive := state.GetInactiveState()

	// Rebuild in-memory structures from cached files
	buildInMemoryStructuresInto(files, inactive)

	// Atomic swap to make new state active
	state.SwapState(inactive)

	log.Printf("✅ Loaded %d files from cache", len(files))
	return c, nil
}

// buildInMemoryStructuresInto rebuilds the in-memory file index from cached files
func buildInMemoryStructuresInto(files []models.FileInfo, targetState *state.AppState) {
	filesByTag := make(map[string][]models.FileInfo)
	tagSet := make(map[string]bool)
	filesByDir := make(map[string][]models.FileInfo)

	// Process files and build indexes
	for _, file := range files {
		// Add to all files list
		targetState.AllFiles = append(targetState.AllFiles, file)

		// Get type category first (needed for tag deduplication check)
		category := config.GetFileTypeCategory(file.Name)

		// Track tags (skip if tag matches type category to avoid duplicates)
		if len(file.Tags) == 0 {
			filesByTag["Untagged"] = append(filesByTag["Untagged"], file)
		} else {
			for _, tag := range file.Tags {
				tagSet[tag] = true
				if tag != category {
					filesByTag[tag] = append(filesByTag[tag], file)
				}
			}
		}

		// Track by directory
		dirPath := filepath.Dir(file.RelPath)
		if dirPath == "." {
			dirPath = ""
		}
		filesByDir[dirPath] = append(filesByDir[dirPath], file)

		// Add to type category
		filesByTag[category] = append(filesByTag[category], file)
	}

	// Create "All" category
	allFilesList := targetState.AllFiles
	sort.Slice(allFilesList, func(i, j int) bool {
		return allFilesList[i].Created.After(allFilesList[j].Created)
	})
	filesByTag["All"] = allFilesList

	// Create hierarchical folder categories
	for dirPath, dirFiles := range filesByDir {
		if dirPath == "" {
			continue
		}
		sort.Slice(dirFiles, func(i, j int) bool {
			return dirFiles[i].Created.After(dirFiles[j].Created)
		})
		categoryName := "📁 " + dirPath
		filesByTag[categoryName] = dirFiles
	}

	// Create parent folder aggregations
	parentAggregates := make(map[string][]models.FileInfo)
	for dirPath, dirFiles := range filesByDir {
		if dirPath == "" {
			continue
		}
		parts := strings.Split(dirPath, "/")
		for i := 1; i < len(parts); i++ {
			parentPath := "📁 " + strings.Join(parts[:i], "/")
			parentAggregates[parentPath] = append(parentAggregates[parentPath], dirFiles...)
		}
	}

	// Merge parent aggregations
	for parentPath, aggregatedFiles := range parentAggregates {
		if existing, ok := filesByTag[parentPath]; ok {
			filesByTag[parentPath] = append(existing, aggregatedFiles...)
		} else {
			filesByTag[parentPath] = aggregatedFiles
		}
	}

	// Sort all categories by creation time
	for tag := range filesByTag {
		sort.Slice(filesByTag[tag], func(i, j int) bool {
			return filesByTag[tag][i].Created.After(filesByTag[tag][j].Created)
		})
	}

	// Set filesByTag
	targetState.FilesByTag = filesByTag

	// Set all tags
	allTags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		allTags = append(allTags, tag)
	}
	sort.Strings(allTags)
	targetState.AllTags = allTags
}

// SaveToCache saves the current in-memory state to the cache (exported for API handlers)
func SaveToCache(c *cache.Cache) {
	log.Println("💾 Saving scan results to cache...")

	allFiles := state.GetAllFiles()
	allTags := state.GetAllTags()

	if err := c.SaveFiles(allFiles, len(allTags)); err != nil {
		log.Printf("⚠️  Failed to save to cache: %v", err)
		return
	}

	log.Println("✅ Cache saved successfully")
}
