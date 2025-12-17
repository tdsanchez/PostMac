package state

import (
	"sync"
	"sync/atomic"

	"github.com/tdsanchez/PostMac/internal/models"
)

// CacheInterface defines the minimal interface needed from the cache
type CacheInterface interface {
	UpdateFileComment(relPath, comment string) error
	UpdateFileTags(relPath string, tags []string) error
	DeleteFile(relPath string) error
	Close() error
}

// AppState encapsulates all application state that can be atomically swapped
type AppState struct {
	FilesByTag map[string][]models.FileInfo
	AllFiles   []models.FileInfo
	AllTags    []string
}

var (
	serveDir string

	// Double-buffered state for lock-free reads
	stateA       *AppState
	stateB       *AppState
	currentState atomic.Value // Holds *AppState - lock-free reads!
	inactiveIdx  int          // 0=A is inactive, 1=B is inactive
	stateMutex   sync.Mutex   // Only for swap operation

	// Legacy variables (kept for compatibility during transition)
	filesByTag      map[string][]models.FileInfo // DEPRECATED: Use GetCurrent()
	allFiles        []models.FileInfo             // DEPRECATED: Use GetCurrent()
	allTags         []string                      // DEPRECATED: Use GetCurrent()
	dataMutex       sync.RWMutex                  // DEPRECATED: Will be removed

	serverReady     chan bool
	conversionCache sync.Map
	writeQueue      []models.WriteQueueItem
	writeQueueMutex sync.Mutex
	dbCache         CacheInterface
	cacheMutex      sync.RWMutex
	scanState       struct {
		isScanning bool
		completed  bool
		mu         sync.RWMutex
	}
)

// Initialize sets up the state channels and structures
func Initialize() {
	serverReady = make(chan bool, 1)

	// Initialize double-buffered state with empty state
	emptyState := &AppState{
		FilesByTag: make(map[string][]models.FileInfo),
		AllFiles:   make([]models.FileInfo, 0),
		AllTags:    make([]string, 0),
	}
	InitializeDoubleBuffer(emptyState)
}

// GetServeDir returns the current serve directory
func GetServeDir() string {
	return serveDir
}

// SetServeDir sets the serve directory
func SetServeDir(dir string) {
	serveDir = dir
}

// GetFilesByTag returns the filesByTag map (read-only access, must hold lock)
func GetFilesByTag() map[string][]models.FileInfo {
	return filesByTag
}

// SetFilesByTag replaces the entire filesByTag map
func SetFilesByTag(files map[string][]models.FileInfo) {
	filesByTag = files
}

// GetAllFiles returns all files (read-only access, must hold lock)
func GetAllFiles() []models.FileInfo {
	return allFiles
}

// SetAllFiles replaces the entire allFiles slice
func SetAllFiles(files []models.FileInfo) {
	allFiles = files
}

// AppendAllFiles adds files to the allFiles slice
func AppendAllFiles(files ...models.FileInfo) {
	allFiles = append(allFiles, files...)
}

// GetAllTags returns all tag categories
func GetAllTags() []string {
	return allTags
}

// SetAllTags replaces the entire allTags slice
func SetAllTags(tags []string) {
	allTags = tags
}

// LockData locks the data mutex for writing
func LockData() {
	dataMutex.Lock()
}

// UnlockData unlocks the data mutex
func UnlockData() {
	dataMutex.Unlock()
}

// RLockData locks the data mutex for reading
func RLockData() {
	dataMutex.RLock()
}

// RUnlockData unlocks the read lock on data mutex
func RUnlockData() {
	dataMutex.RUnlock()
}

// GetServerReady returns the server ready channel
func GetServerReady() chan bool {
	return serverReady
}

// GetConversionCache returns the conversion cache
func GetConversionCache() *sync.Map {
	return &conversionCache
}

// GetWriteQueue returns the write queue (must hold lock)
func GetWriteQueue() []models.WriteQueueItem {
	return writeQueue
}

// SetWriteQueue replaces the entire write queue
func SetWriteQueue(queue []models.WriteQueueItem) {
	writeQueue = queue
}

// AppendWriteQueue adds an item to the write queue
func AppendWriteQueue(item models.WriteQueueItem) {
	writeQueue = append(writeQueue, item)
}

// LockWriteQueue locks the write queue mutex
func LockWriteQueue() {
	writeQueueMutex.Lock()
}

// UnlockWriteQueue unlocks the write queue mutex
func UnlockWriteQueue() {
	writeQueueMutex.Unlock()
}

// GetFileCount returns the number of files
func GetFileCount() int {
	dataMutex.RLock()
	defer dataMutex.RUnlock()
	return len(allFiles)
}

// GetCategoryCount returns the number of tag categories
func GetCategoryCount() int {
	dataMutex.RLock()
	defer dataMutex.RUnlock()
	return len(filesByTag)
}

// SetCache sets the database cache
func SetCache(cache CacheInterface) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	dbCache = cache
}

// GetCache returns the database cache
func GetCache() CacheInterface {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()
	return dbCache
}

// SetScanning sets whether a scan is currently in progress
func SetScanning(scanning bool) {
	scanState.mu.Lock()
	defer scanState.mu.Unlock()
	scanState.isScanning = scanning
	if scanning {
		scanState.completed = false
	}
}

// IsScanning returns whether a scan is currently in progress
func IsScanning() bool {
	scanState.mu.RLock()
	defer scanState.mu.RUnlock()
	return scanState.isScanning
}

// SetScanCompleted marks that a scan has completed
func SetScanCompleted() {
	scanState.mu.Lock()
	defer scanState.mu.Unlock()
	scanState.isScanning = false
	scanState.completed = true
}

// GetScanState returns the current scan state
func GetScanState() (isScanning bool, completed bool) {
	scanState.mu.RLock()
	defer scanState.mu.RUnlock()
	return scanState.isScanning, scanState.completed
}

// ClearScanCompleted clears the completed flag (for UI acknowledgment)
func ClearScanCompleted() {
	scanState.mu.Lock()
	defer scanState.mu.Unlock()
	scanState.completed = false
}

// ============================================================================
// Double-Buffer State Management (New Lock-Free API)
// ============================================================================

// GetCurrent returns the current active state (lock-free read)
func GetCurrent() *AppState {
	return currentState.Load().(*AppState)
}

// InitializeDoubleBuffer sets up the double-buffered state with initial data
func InitializeDoubleBuffer(initialState *AppState) {
	stateA = initialState
	stateB = &AppState{
		FilesByTag: make(map[string][]models.FileInfo),
		AllFiles:   make([]models.FileInfo, 0),
		AllTags:    make([]string, 0),
	}
	currentState.Store(stateA)
	inactiveIdx = 1 // B is inactive initially

	// Also populate legacy variables for backward compatibility
	filesByTag = initialState.FilesByTag
	allFiles = initialState.AllFiles
	allTags = initialState.AllTags
}

// GetInactiveState returns the inactive state buffer for building new state
func GetInactiveState() *AppState {
	stateMutex.Lock()
	defer stateMutex.Unlock()

	if inactiveIdx == 0 {
		// A is inactive, reset it
		stateA = &AppState{
			FilesByTag: make(map[string][]models.FileInfo),
			AllFiles:   make([]models.FileInfo, 0),
			AllTags:    make([]string, 0),
		}
		return stateA
	} else {
		// B is inactive, reset it
		stateB = &AppState{
			FilesByTag: make(map[string][]models.FileInfo),
			AllFiles:   make([]models.FileInfo, 0),
			AllTags:    make([]string, 0),
		}
		return stateB
	}
}

// SwapState atomically swaps the active and inactive states
func SwapState(newState *AppState) {
	stateMutex.Lock()
	defer stateMutex.Unlock()

	// Store new state as current (atomic operation)
	currentState.Store(newState)

	// Flip which buffer is inactive
	inactiveIdx = 1 - inactiveIdx

	// Update legacy variables for backward compatibility
	filesByTag = newState.FilesByTag
	allFiles = newState.AllFiles
	allTags = newState.AllTags
}
