package watcher

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/tdsanchez/PostMac/media-server/internal/cache"
	"github.com/tdsanchez/PostMac/media-server/internal/scanner"
	"github.com/tdsanchez/PostMac/media-server/internal/state"
)

// Watcher monitors filesystem changes and triggers automatic rescans
type Watcher struct {
	fsWatcher     *fsnotify.Watcher
	serveDir      string
	dbCache       *cache.Cache
	debounceTimer *time.Timer
	eventQueue    chan bool
	stopChan      chan bool
}

// New creates a new filesystem watcher
func New(serveDir string, dbCache *cache.Cache) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		fsWatcher:  fsWatcher,
		serveDir:   serveDir,
		dbCache:    dbCache,
		eventQueue: make(chan bool, 100), // Buffer to prevent blocking
		stopChan:   make(chan bool),
	}

	// Add root directory to watcher
	if err := fsWatcher.Add(serveDir); err != nil {
		return nil, err
	}

	// Add all subdirectories recursively
	if err := w.addSubdirectories(serveDir); err != nil {
		log.Printf("Warning: Error adding subdirectories to watch: %v", err)
	}

	log.Printf("📡 Filesystem watcher initialized for: %s", serveDir)

	return w, nil
}

// addSubdirectories recursively adds all subdirectories to the watcher
func (w *Watcher) addSubdirectories(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors, continue walking
		}

		// Skip if not a directory
		if !info.IsDir() {
			return nil
		}

		// Skip hidden directories
		if strings.HasPrefix(info.Name(), ".") && path != root {
			return filepath.SkipDir
		}

		// Skip Photos libraries
		if strings.HasSuffix(path, ".photoslibrary") {
			return filepath.SkipDir
		}

		// Add directory to watcher
		if err := w.fsWatcher.Add(path); err != nil {
			log.Printf("Warning: Failed to watch %s: %v", path, err)
		}

		return nil
	})
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

	// Perform scan
	if err := scanner.ScanDirectory(w.serveDir); err != nil {
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
