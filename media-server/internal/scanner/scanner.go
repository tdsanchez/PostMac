package scanner

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/tdsanchez/PostMac/internal/cache"
	"github.com/tdsanchez/PostMac/internal/config"
	"github.com/tdsanchez/PostMac/internal/models"
	"github.com/tdsanchez/PostMac/internal/state"
)

// FreshnessScanner handles background freshness checking
type FreshnessScanner struct {
	cache     *cache.Cache
	progress  atomic.Int64
	total     atomic.Int64
	isRunning atomic.Bool
}

var globalFreshnessScanner *FreshnessScanner

// GetFreshnessScanner returns the global freshness scanner instance
func GetFreshnessScanner() *FreshnessScanner {
	return globalFreshnessScanner
}

// GetProgress returns current progress
func (fs *FreshnessScanner) GetProgress() (checked, total int64) {
	return fs.progress.Load(), fs.total.Load()
}

// IsRunning returns whether the scanner is running
func (fs *FreshnessScanner) IsRunning() bool {
	return fs.isRunning.Load()
}

// ProcessPaths processes stdin paths using double-buffered state (no blocking)
func ProcessPaths(paths []string) error {
	// Get inactive state buffer
	inactive := state.GetInactiveState()

	// Build into inactive state (no locks held!)
	if err := ProcessPathsInto(paths, inactive); err != nil {
		return err
	}

	// Atomic swap when complete
	state.SwapState(inactive)
	return nil
}

// ProcessPathsInto processes stdin paths, populating the provided state
// This function does NOT acquire locks - it builds into the provided state buffer
func ProcessPathsInto(paths []string, targetState *state.AppState) error {
	// Initialize target state
	targetState.FilesByTag = make(map[string][]models.FileInfo)
	targetState.AllFiles = make([]models.FileInfo, 0)
	tagSet := make(map[string]bool)
	filesByTag := targetState.FilesByTag

	// Track files by directory path for hierarchical categories
	filesByDir := make(map[string][]models.FileInfo)

	if len(paths) > 0 {
		log.Printf("📥 Processing %d paths...\n", len(paths))
		for _, path := range paths {
			// Get file info
			info, err := os.Stat(path)
			if err != nil {
				log.Printf("⚠️  Skipping path (stat error): %s - %v", path, err)
				continue
			}

			// Skip directories
			if info.IsDir() {
				continue
			}

			// Check if file type is supported
			ext := strings.ToLower(filepath.Ext(path))
			if !config.SupportedExts[ext] {
				continue
			}

			// Get tags and metadata
			tags := GetMacOSTags(path)
			comment := GetMacOSComment(path)

			// Perform date analysis (Phase 1: JPEG enrichment)
			osModTime, osBirthTime, exifCreate, exifModify, earliest, needsCorrection, largeDiscrepancy, maxDiffHours := analyzeDateMetadata(path, info)

			fileInfo := models.FileInfo{
				Name:                info.Name(),
				Path:                path, // Absolute path is the primary identifier
				Tags:                tags,
				Comment:             comment,
				Created:             getBirthTime(info),
				Size:                info.Size(),
				OSModTime:           osModTime,
				OSBirthTime:         osBirthTime,
				EXIFCreateDate:      exifCreate,
				EXIFModifyDate:      exifModify,
				EarliestDate:        earliest,
				NeedsDateCorrection: needsCorrection,
				LargeDiscrepancy:    largeDiscrepancy,
				MaxDiffHours:        maxDiffHours,
			}

			// Add to master list
			targetState.AllFiles = append(targetState.AllFiles, fileInfo)

			// Track by parent directory for hierarchical categories
			dirPath := filepath.Dir(path)
			filesByDir[dirPath] = append(filesByDir[dirPath], fileInfo)

			// Add to file type category
			typeCategory := config.GetFileTypeCategory(info.Name())
			filesByTag[typeCategory] = append(filesByTag[typeCategory], fileInfo)

			// Add to tag categories
			if len(tags) == 0 {
				filesByTag["Untagged"] = append(filesByTag["Untagged"], fileInfo)
			} else {
				for _, tag := range tags {
					if tag != typeCategory {
						filesByTag[tag] = append(filesByTag[tag], fileInfo)
						tagSet[tag] = true
					}
				}
			}

			// Add to date correction synthetic categories
			if fileInfo.NeedsDateCorrection {
				filesByTag["📅 Needs Date Correction"] = append(filesByTag["📅 Needs Date Correction"], fileInfo)

				if fileInfo.LargeDiscrepancy {
					filesByTag["🔍 Needs Review (>24h)"] = append(filesByTag["🔍 Needs Review (>24h)"], fileInfo)
				}
			}
		}
		log.Printf("✅ Processed %d files\n", len(targetState.AllFiles))
	}

	// Create "All" category from all files in the library
	allFilesList := targetState.AllFiles
	sort.Slice(allFilesList, func(i, j int) bool {
		return allFilesList[i].Created.After(allFilesList[j].Created)
	})
	filesByTag["All"] = allFilesList

	// Create tag count synthetic views
	tagCountBuckets := map[string][]models.FileInfo{
		"1 Tag":   {},
		"2 Tags":  {},
		"3 Tags":  {},
		"4 Tags":  {},
		"5 Tags":  {},
		"6+ Tags": {},
	}

	for _, file := range allFilesList {
		tagCount := len(file.Tags)
		switch {
		case tagCount == 1:
			tagCountBuckets["1 Tag"] = append(tagCountBuckets["1 Tag"], file)
		case tagCount == 2:
			tagCountBuckets["2 Tags"] = append(tagCountBuckets["2 Tags"], file)
		case tagCount == 3:
			tagCountBuckets["3 Tags"] = append(tagCountBuckets["3 Tags"], file)
		case tagCount == 4:
			tagCountBuckets["4 Tags"] = append(tagCountBuckets["4 Tags"], file)
		case tagCount == 5:
			tagCountBuckets["5 Tags"] = append(tagCountBuckets["5 Tags"], file)
		case tagCount >= 6:
			tagCountBuckets["6+ Tags"] = append(tagCountBuckets["6+ Tags"], file)
		}
	}

	// Add tag count categories to filesByTag
	for category, files := range tagCountBuckets {
		if len(files) > 0 {
			filesByTag[category] = files
		}
	}

	// Create hierarchical folder categories (use absolute paths)
	for dirPath, files := range filesByDir {
		// Sort files by creation time
		sort.Slice(files, func(i, j int) bool {
			return files[i].Created.After(files[j].Created)
		})

		// Create category name with folder emoji using absolute path
		categoryName := "📁 " + dirPath
		filesByTag[categoryName] = files
	}

	// Create parent folder aggregations for ALL levels
	parentAggregates := make(map[string][]models.FileInfo)
	for dirPath, files := range filesByDir {
		// Walk up the directory tree
		current := filepath.Dir(dirPath)
		for current != "/" && current != "." {
			parentPath := "📁 " + current
			parentAggregates[parentPath] = append(parentAggregates[parentPath], files...)
			current = filepath.Dir(current)
		}
	}

	// Add or merge parent aggregations into filesByTag
	for parentPath, aggregatedFiles := range parentAggregates {
		if existing, ok := filesByTag[parentPath]; ok {
			filesByTag[parentPath] = append(existing, aggregatedFiles...)
		} else {
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
func UpdateFileTagsInMemory(absPath string, newTags []string) {
	state.LockData()
	defer state.UnlockData()

	allFiles := state.GetAllFiles()
	filesByTag := state.GetFilesByTag()

	// Update allFiles master list
	for i := range allFiles {
		if allFiles[i].Path == absPath {
			oldTags := allFiles[i].Tags
			allFiles[i].Tags = newTags

			// Remove file from old tag categories in inverted index
			for _, oldTag := range oldTags {
				if files, ok := filesByTag[oldTag]; ok {
					for j, f := range files {
						if f.Path == absPath {
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
					if files[j].Path == absPath {
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
func RemoveFileFromMemory(absPath string) {
	state.LockData()
	defer state.UnlockData()

	allFiles := state.GetAllFiles()
	filesByTag := state.GetFilesByTag()

	// Remove from allFiles master list
	for i := range allFiles {
		if allFiles[i].Path == absPath {
			// Remove from slice
			allFiles = append(allFiles[:i], allFiles[i+1:]...)
			state.SetAllFiles(allFiles)
			break
		}
	}

	// Remove from all tag categories in filesByTag
	for categoryName, files := range filesByTag {
		for j := range files {
			if files[j].Path == absPath {
				filesByTag[categoryName] = append(files[:j], files[j+1:]...)
				break
			}
		}
	}

	// Remove from cache if available
	dbCache := state.GetCache()
	if dbCache != nil {
		if err := dbCache.DeleteFile(absPath); err != nil {
			log.Printf("⚠️ Warning: Failed to remove file from cache: %v", err)
		}
	}
}

// analyzeDateMetadata performs date analysis for JPEG files (Phase 1: read-only)
// Compares OS timestamps (mtime, btime) with EXIF dates (CreateDate, ModifyDate)
// and determines if correction is needed
func analyzeDateMetadata(filePath string, info os.FileInfo) (osModTime, osBirthTime, exifCreateDate, exifModifyDate, earliestDate time.Time, needsCorrection, largeDiscrepancy bool, maxDiffHours int) {
	// Fixed MST timezone (UTC-7, no DST) for Arizona
	mstFixed := time.FixedZone("MST", -7*3600)

	// Get OS timestamps and convert to fixed MST for consistent display
	osModTime = info.ModTime().In(mstFixed)
	osBirthTime = getBirthTime(info).In(mstFixed)

	// Check if file is JPEG
	ext := strings.ToLower(filepath.Ext(filePath))
	isJPEG := ext == ".jpg" || ext == ".jpeg"

	if !isJPEG {
		// Not a JPEG - return OS timestamps only, no analysis
		return osModTime, osBirthTime, time.Time{}, time.Time{}, time.Time{}, false, false, 0
	}

	// Parse EXIF data for JPEGs
	f, err := os.Open(filePath)
	if err != nil {
		// Can't open file - return OS timestamps only
		return osModTime, osBirthTime, time.Time{}, time.Time{}, time.Time{}, false, false, 0
	}
	defer f.Close()

	x, err := exif.Decode(f)
	if err != nil {
		// No EXIF data or parse error - return OS timestamps only
		return osModTime, osBirthTime, time.Time{}, time.Time{}, time.Time{}, false, false, 0
	}

	// Helper to parse EXIF date bypassing DST completely
	// EXIF dates are "naive" - they represent local time with no timezone/DST info
	// Arizona: Use MST (-7) always, never PDT (-6)
	parseEXIFDate := func(dateStr string) (time.Time, error) {
		// Use MST fixed offset: -7 hours = -25200 seconds
		// This avoids DST adjustments that Go incorrectly applies to Arizona historical dates
		mstFixed := time.FixedZone("MST", -7*3600)

		// Parse in fixed MST - displays correctly without DST offset
		parsed, err := time.ParseInLocation("2006:01:02 15:04:05", dateStr, mstFixed)
		if err != nil {
			return time.Time{}, err
		}

		return parsed, nil
	}

	// Extract EXIF ModifyDate (DateTime tag - when file was last modified)
	if dtTag, err := x.Get(exif.DateTime); err == nil {
		if dtStr, err := dtTag.StringVal(); err == nil {
			if parsed, err := parseEXIFDate(dtStr); err == nil {
				exifModifyDate = parsed
			}
		}
	}

	// Extract EXIF CreateDate (DateTimeOriginal - when photo was originally taken)
	if dtOrig, err := x.Get(exif.DateTimeOriginal); err == nil {
		if dtOrigStr, err := dtOrig.StringVal(); err == nil {
			if parsed, err := parseEXIFDate(dtOrigStr); err == nil {
				exifCreateDate = parsed
			}
		}
	}

	// If CreateDate not found, use DateTimeDigitized as fallback
	if exifCreateDate.IsZero() {
		if dtDig, err := x.Get(exif.DateTimeDigitized); err == nil {
			if dtDigStr, err := dtDig.StringVal(); err == nil {
				if parsed, err := parseEXIFDate(dtDigStr); err == nil {
					exifCreateDate = parsed
				}
			}
		}
	}

	// Helper function to compare dates component-wise (ignore timezone)
	dateComponents := func(t time.Time) (int, int, int, int, int, int) {
		return t.Year(), int(t.Month()), t.Day(), t.Hour(), t.Minute(), t.Second()
	}

	compareDates := func(a, b time.Time) bool {
		// Returns true if a is before b (component-wise, ignoring timezone)
		y1, m1, d1, h1, min1, s1 := dateComponents(a)
		y2, m2, d2, h2, min2, s2 := dateComponents(b)

		if y1 != y2 { return y1 < y2 }
		if m1 != m2 { return m1 < m2 }
		if d1 != d2 { return d1 < d2 }
		if h1 != h2 { return h1 < h2 }
		if min1 != min2 { return min1 < min2 }
		return s1 < s2
	}

	// Find earliest date among all 4 timestamps (component-wise comparison)
	dates := []time.Time{osModTime, osBirthTime}
	if !exifCreateDate.IsZero() {
		dates = append(dates, exifCreateDate)
	}
	if !exifModifyDate.IsZero() {
		dates = append(dates, exifModifyDate)
	}

	// Find the earliest non-zero date (using component comparison)
	for _, d := range dates {
		if !d.IsZero() {
			if earliestDate.IsZero() || compareDates(d, earliestDate) {
				earliestDate = d
			}
		}
	}

	// Component-wise date equality check (ignore timezone)
	datesEqual := func(a, b time.Time) bool {
		y1, m1, d1, h1, min1, s1 := dateComponents(a)
		y2, m2, d2, h2, min2, s2 := dateComponents(b)
		// Allow 1-second tolerance
		return y1 == y2 && m1 == m2 && d1 == d2 && h1 == h2 && min1 == min2 && (s1 == s2 || s1 == s2+1 || s1 == s2-1)
	}

	// Component-wise difference calculation (returns hours difference, approximate)
	dateDiffHours := func(a, b time.Time) int {
		y1, m1, d1, h1, min1, _ := dateComponents(a)
		y2, m2, d2, h2, min2, _ := dateComponents(b)

		// Rough calculation: year diff * 8760 + month diff * 730 + day diff * 24 + hour diff
		yearDiff := (y1 - y2) * 8760
		monthDiff := (m1 - m2) * 730
		dayDiff := (d1 - d2) * 24
		hourDiff := (h1 - h2)
		minDiff := (min1 - min2) / 60

		total := yearDiff + monthDiff + dayDiff + hourDiff + minDiff
		if total < 0 {
			total = -total
		}
		return total
	}

	// Determine if correction is needed (any date differs from earliest)
	// Also detect large discrepancies (>24 hours) that should be flagged for review
	// Calculate maximum difference between any two timestamps
	//
	// IMPORTANT: Only flag correction if EXIF dates exist. Files without EXIF data
	// (screenshots, scans, etc.) may have legitimate modification times that differ
	// from birth time - we should not touch those.
	maxDiffHours = 0
	if !earliestDate.IsZero() && (!exifCreateDate.IsZero() || !exifModifyDate.IsZero()) {
		needsCorrection = false
		largeDiscrepancy = false

		for _, d := range dates {
			if !d.IsZero() {
				// Check if dates differ (component-wise)
				if !datesEqual(d, earliestDate) {
					needsCorrection = true

					// Track maximum difference
					diff := dateDiffHours(d, earliestDate)
					if diff > maxDiffHours {
						maxDiffHours = diff
					}

					// Flag for review if difference > 24 hours (approximate)
					if diff > 24 {
						largeDiscrepancy = true
					}
				}
			}
		}
	}

	return osModTime, osBirthTime, exifCreateDate, exifModifyDate, earliestDate, needsCorrection, largeDiscrepancy, maxDiffHours
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

// LoadOrScan loads files from cache or processes stdin paths
// SQLite is the source of truth - instant startup from cache
func LoadOrScan(stdinPaths []string, port string) (*cache.Cache, error) {
	log.Println("🔍 Opening cache...")

	// Open or create cache (port-isolated: cache-PORT.db)
	c, err := cache.New(port)
	if err != nil {
		return nil, err
	}

	// Check if we have cached data
	lastScan, totalFiles, _, err := c.GetScanMetadata()
	if err != nil {
		log.Printf("⚠️  Failed to read cache metadata: %v", err)
	}

	// If we have stdin paths, process them and merge with cache
	if len(stdinPaths) > 0 {
		log.Printf("📥 Processing %d stdin paths...\n", len(stdinPaths))

		// Load existing cached files first
		var existingFiles []models.FileInfo
		if totalFiles > 0 {
			existingFiles, err = c.LoadFiles()
			if err != nil {
				log.Printf("⚠️  Failed to load from cache: %v", err)
				existingFiles = nil
			}
		}

		// Create a map of existing files for quick lookup
		existingMap := make(map[string]bool)
		for _, f := range existingFiles {
			existingMap[f.Path] = true
		}

		// Process stdin paths - check which ones need parsing
		var newFiles []models.FileInfo
		for _, path := range stdinPaths {
			// Check if already in cache with fresh data
			if cachedMtime, ok := c.GetFileMtime(path); ok {
				info, err := os.Stat(path)
				if err == nil && info.ModTime().UnixNano() == cachedMtime {
					// File is fresh in cache, skip re-parsing
					continue
				}
			}

			// Need to parse this file
			info, err := os.Stat(path)
			if err != nil {
				log.Printf("⚠️  Skipping path (stat error): %s - %v", path, err)
				continue
			}

			if info.IsDir() {
				continue
			}

			ext := strings.ToLower(filepath.Ext(path))
			if !config.SupportedExts[ext] {
				continue
			}

			tags := GetMacOSTags(path)
			comment := GetMacOSComment(path)
			osModTime, osBirthTime, exifCreate, exifModify, earliest, needsCorrection, largeDiscrepancy, maxDiffHours := analyzeDateMetadata(path, info)

			fileInfo := models.FileInfo{
				Name:                info.Name(),
				Path:                path,
				Tags:                tags,
				Comment:             comment,
				Created:             getBirthTime(info),
				Size:                info.Size(),
				OSModTime:           osModTime,
				OSBirthTime:         osBirthTime,
				EXIFCreateDate:      exifCreate,
				EXIFModifyDate:      exifModify,
				EarliestDate:        earliest,
				NeedsDateCorrection: needsCorrection,
				LargeDiscrepancy:    largeDiscrepancy,
				MaxDiffHours:        maxDiffHours,
			}

			newFiles = append(newFiles, fileInfo)

			// Upsert to cache
			if err := c.UpsertFile(fileInfo); err != nil {
				log.Printf("⚠️  Failed to cache file %s: %v", path, err)
			}
		}

		log.Printf("✅ Parsed %d new/updated files\n", len(newFiles))

		// Now process all paths to build state
		if err := ProcessPaths(stdinPaths); err != nil {
			return nil, err
		}

		// Save updated cache metadata
		SaveToCache(c)

		// Start background freshness scanner
		globalFreshnessScanner = &FreshnessScanner{cache: c}
		go globalFreshnessScanner.runFreshnessCheck(stdinPaths)

		return c, nil
	}

	// No stdin paths - load from cache for instant startup
	if totalFiles > 0 {
		cacheAge := time.Since(lastScan)
		log.Printf("⚡ Loading from cache (last scan: %v ago, %d files)...",
			cacheAge.Round(time.Second), totalFiles)

		files, err := c.LoadFiles()
		if err != nil {
			log.Printf("⚠️  Failed to load from cache: %v", err)
			return c, nil
		}

		// Get inactive state buffer
		inactive := state.GetInactiveState()

		// Rebuild in-memory structures from cached files
		buildInMemoryStructuresInto(files, inactive)

		// Atomic swap to make new state active
		state.SwapState(inactive)

		log.Printf("✅ Loaded %d files from cache", len(files))

		// Start background freshness scanner
		globalFreshnessScanner = &FreshnessScanner{cache: c}
		paths := make([]string, len(files))
		for i, f := range files {
			paths[i] = f.Path
		}
		go globalFreshnessScanner.runFreshnessCheck(paths)

		return c, nil
	}

	log.Println("📊 No cached files found, waiting for stdin paths...")
	return c, nil
}

// runFreshnessCheck validates cached files against filesystem
func (fs *FreshnessScanner) runFreshnessCheck(paths []string) {
	fs.isRunning.Store(true)
	defer fs.isRunning.Store(false)

	fs.total.Store(int64(len(paths)))
	log.Printf("🔄 Starting background freshness check for %d files...", len(paths))

	var staleCount int
	var missingCount int

	for i, path := range paths {
		fs.progress.Store(int64(i + 1))

		info, err := os.Stat(path)
		if err != nil {
			// File missing - will be hidden from UI
			missingCount++
			continue
		}

		// Check if mtime matches cached value
		cachedMtime, ok := fs.cache.GetFileMtime(path)
		if !ok {
			continue
		}

		if info.ModTime().UnixNano() != cachedMtime {
			// File changed - re-parse and update
			staleCount++
			tags := GetMacOSTags(path)
			comment := GetMacOSComment(path)
			osModTime, osBirthTime, exifCreate, exifModify, earliest, needsCorrection, largeDiscrepancy, maxDiffHours := analyzeDateMetadata(path, info)

			fileInfo := models.FileInfo{
				Name:                info.Name(),
				Path:                path,
				Tags:                tags,
				Comment:             comment,
				Created:             getBirthTime(info),
				Size:                info.Size(),
				OSModTime:           osModTime,
				OSBirthTime:         osBirthTime,
				EXIFCreateDate:      exifCreate,
				EXIFModifyDate:      exifModify,
				EarliestDate:        earliest,
				NeedsDateCorrection: needsCorrection,
				LargeDiscrepancy:    largeDiscrepancy,
				MaxDiffHours:        maxDiffHours,
			}

			fs.cache.UpsertFile(fileInfo)
		}

		// Pacing: ~100 files/second
		time.Sleep(10 * time.Millisecond)
	}

	log.Printf("✅ Freshness check complete: %d stale, %d missing", staleCount, missingCount)
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

		// Track by directory (use absolute path)
		dirPath := filepath.Dir(file.Path)
		filesByDir[dirPath] = append(filesByDir[dirPath], file)

		// Add to type category
		filesByTag[category] = append(filesByTag[category], file)

		// Add to ML training synthetic categories (if applicable)
		if file.NeedsDateCorrection {
			filesByTag["📅 Needs Date Correction"] = append(filesByTag["📅 Needs Date Correction"], file)
			if file.LargeDiscrepancy {
				filesByTag["🔍 Needs Review (>24h)"] = append(filesByTag["🔍 Needs Review (>24h)"], file)
			}
		}
	}

	// Create "All" category
	allFilesList := targetState.AllFiles
	sort.Slice(allFilesList, func(i, j int) bool {
		return allFilesList[i].Created.After(allFilesList[j].Created)
	})
	filesByTag["All"] = allFilesList

	// Create tag count synthetic views
	tagCountBuckets := map[string][]models.FileInfo{
		"1 Tag":   {},
		"2 Tags":  {},
		"3 Tags":  {},
		"4 Tags":  {},
		"5 Tags":  {},
		"6+ Tags": {},
	}

	for _, file := range allFilesList {
		tagCount := len(file.Tags)
		switch {
		case tagCount == 1:
			tagCountBuckets["1 Tag"] = append(tagCountBuckets["1 Tag"], file)
		case tagCount == 2:
			tagCountBuckets["2 Tags"] = append(tagCountBuckets["2 Tags"], file)
		case tagCount == 3:
			tagCountBuckets["3 Tags"] = append(tagCountBuckets["3 Tags"], file)
		case tagCount == 4:
			tagCountBuckets["4 Tags"] = append(tagCountBuckets["4 Tags"], file)
		case tagCount == 5:
			tagCountBuckets["5 Tags"] = append(tagCountBuckets["5 Tags"], file)
		case tagCount >= 6:
			tagCountBuckets["6+ Tags"] = append(tagCountBuckets["6+ Tags"], file)
		}
	}

	// Add tag count categories to filesByTag
	for category, catFiles := range tagCountBuckets {
		if len(catFiles) > 0 {
			filesByTag[category] = catFiles
		}
	}

	// Create hierarchical folder categories (using absolute paths)
	for dirPath, dirFiles := range filesByDir {
		sort.Slice(dirFiles, func(i, j int) bool {
			return dirFiles[i].Created.After(dirFiles[j].Created)
		})
		categoryName := "📁 " + dirPath
		filesByTag[categoryName] = dirFiles
	}

	// Create parent folder aggregations
	parentAggregates := make(map[string][]models.FileInfo)
	for dirPath, dirFiles := range filesByDir {
		// Walk up the directory tree
		current := filepath.Dir(dirPath)
		for current != "/" && current != "." {
			parentPath := "📁 " + current
			parentAggregates[parentPath] = append(parentAggregates[parentPath], dirFiles...)
			current = filepath.Dir(current)
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
