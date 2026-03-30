package watcher

import (
	"log"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/tdsanchez/PostMac/internal/cache"
	"github.com/tdsanchez/PostMac/internal/config"
	"github.com/tdsanchez/PostMac/internal/scanner"
	"github.com/tdsanchez/PostMac/internal/state"
)

// Watcher monitors filesystem changes and triggers automatic rescans
type Watcher struct {
	fsWatcher     *fsnotify.Watcher
	dbCache       *cache.Cache
	debounceTimer *time.Timer
	eventQueue    chan bool
	stopChan      chan bool
	watchedPaths  []string // Original stdin paths for rescans
}

// NewFromPaths creates a new filesystem watcher that monitors parent directories of the given paths.
// Uses recursive FSEvents watching on minimal ancestor directories to avoid per-directory
// kqueue file descriptor exhaustion when running multiple instances on large libraries.
func NewFromPaths(paths []string, dbCache *cache.Cache) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		fsWatcher:    fsWatcher,
		dbCache:      dbCache,
		eventQueue:   make(chan bool, 100),
		stopChan:     make(chan bool),
		watchedPaths: paths,
	}

	// Count total unique parent dirs for logging
	parentDirs := make(map[string]bool)
	for _, p := range paths {
		parentDirs[filepath.Dir(p)] = true
	}
	totalParentDirs := len(parentDirs)

	// Find the minimal set of ancestor directories that covers all parent dirs.
	// Example: if files span /Volumes/X/photos/2020/Jan and /Volumes/X/photos/2020/Feb,
	// we only need to watch /Volumes/X/photos — one recursive watch covers both.
	ancestors := findCommonAncestors(paths)

	// Watch each minimal ancestor directory. On macOS, each fsWatcher.Add()
	// call costs 1 open file descriptor (kqueue). By watching only top-level
	// ancestor directories instead of every unique parent directory, we reduce
	// FD usage from O(unique_dirs) to O(volume_roots) — typically 2–10 FDs
	// rather than thousands. Events from deeply nested subdirectories are not
	// captured by this watch (kqueue is non-recursive), but the background
	// freshness scanner provides a reliable safety net for those cases.
	watchedCount := 0
	for _, dir := range ancestors {
		if err := fsWatcher.Add(dir); err != nil {
			log.Printf("⚠️  Failed to watch %s: %v", dir, err)
		} else {
			watchedCount++
		}
	}

	log.Printf("📡 Filesystem watcher: %d recursive watches covering %d parent directories",
		watchedCount, totalParentDirs)

	return w, nil
}

// findCommonAncestors computes the minimal set of directories that covers all
// parent directories of the given paths. Sorted so shorter (higher-level) paths
// are processed first; any dir whose prefix is already in the set is skipped.
func findCommonAncestors(paths []string) []string {
	parentDirs := make(map[string]bool)
	for _, p := range paths {
		parentDirs[filepath.Dir(p)] = true
	}

	sorted := make([]string, 0, len(parentDirs))
	for d := range parentDirs {
		sorted = append(sorted, d)
	}
	sort.Strings(sorted)

	var minimal []string
	for _, d := range sorted {
		covered := false
		for _, ancestor := range minimal {
			if strings.HasPrefix(d, ancestor+"/") {
				covered = true
				break
			}
		}
		if !covered {
			minimal = append(minimal, d)
		}
	}
	return minimal
}

// Start begins monitoring filesystem events
func (w *Watcher) Start() {
	go w.watchEvents()
	go w.debouncedRescan()
	log.Println("✅ Filesystem watcher started (auto-rescan enabled)")
}

// watchEvents monitors filesystem events and queues rescans
func (w *Watcher) watchEvents() {
	for {
		select {
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}

			// Ignore noise
			if w.shouldIgnoreEvent(event) {
				continue
			}

			// Log event for debugging
			log.Printf("📝 FS event: %s %s", event.Op, event.Name)

			// If this is a CREATE event for a supported file, add it to stdin paths
			if event.Op&fsnotify.Create != 0 {
				ext := strings.ToLower(filepath.Ext(event.Name))
				if config.SupportedExts[ext] {
					log.Printf("📥 Adding new file to scan list: %s", event.Name)
					state.AddStdinPath(event.Name)
				}
			}

			// Queue rescan (non-blocking)
			select {
			case w.eventQueue <- true:
			default:
				// Queue full, skip this event
			}

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			log.Printf("⚠️  Filesystem watcher error: %v", err)

		case <-w.stopChan:
			return
		}
	}
}

// debouncedRescan handles debouncing and triggering rescans
func (w *Watcher) debouncedRescan() {
	const debounceDelay = 3 * time.Second

	for {
		select {
		case <-w.eventQueue:
			// Reset or create debounce timer
			if w.debounceTimer != nil {
				w.debounceTimer.Stop()
			}

			w.debounceTimer = time.AfterFunc(debounceDelay, func() {
				w.triggerRescan()
			})

		case <-w.stopChan:
			if w.debounceTimer != nil {
				w.debounceTimer.Stop()
			}
			return
		}
	}
}

// triggerRescan performs the actual filesystem rescan
func (w *Watcher) triggerRescan() {
	// Don't trigger if already scanning
	if state.IsScanning() {
		log.Println("⏭️  Skipping auto-rescan (scan already in progress)")
		return
	}

	log.Println("🔄 Auto-rescan triggered by filesystem changes...")

	state.SetScanning(true)

	// Get stdin paths if they were provided at startup
	stdinPaths := state.GetStdinPaths()

	// Perform scan using stdin paths
	if err := scanner.ProcessPaths(stdinPaths); err != nil {
		log.Printf("❌ Auto-rescan failed: %v", err)
		state.SetScanning(false)
		return
	}

	// Save to cache
	if w.dbCache != nil {
		scanner.SaveToCache(w.dbCache)
	}

	state.SetScanCompleted()
	log.Println("✅ Auto-rescan completed")
}

// shouldIgnoreEvent filters out events we don't care about
func (w *Watcher) shouldIgnoreEvent(event fsnotify.Event) bool {
	name := filepath.Base(event.Name)

	// Ignore macOS metadata files
	if name == ".DS_Store" || strings.HasPrefix(name, "._") {
		return true
	}

	// Ignore temporary files
	if strings.HasSuffix(name, ".tmp") || strings.HasSuffix(name, "~") {
		return true
	}

	// Ignore hidden files (except directories, which we handle in Walk)
	if strings.HasPrefix(name, ".") {
		return true
	}

	// Ignore Photos library internals
	if strings.Contains(event.Name, ".photoslibrary/") {
		return true
	}

	// Only care about Write, Create, Remove, Rename
	if event.Op&fsnotify.Write == 0 &&
		event.Op&fsnotify.Create == 0 &&
		event.Op&fsnotify.Remove == 0 &&
		event.Op&fsnotify.Rename == 0 {
		return true
	}

	return false
}

// Stop stops the filesystem watcher
func (w *Watcher) Stop() {
	log.Println("🛑 Stopping filesystem watcher...")
	close(w.stopChan)
	w.fsWatcher.Close()
}
