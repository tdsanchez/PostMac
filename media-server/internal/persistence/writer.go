package persistence

import (
	"log"
	"path/filepath"
	"time"

	"github.com/tdsanchez/PostMac/internal/models"
	"github.com/tdsanchez/PostMac/internal/scanner"
	"github.com/tdsanchez/PostMac/internal/state"
)

// QueueDiskWrite adds a tag update to the write queue for batched persistence.
// IMPORTANT: filePath should be RELATIVE to serveDir (ProcessBatchWrites will join it).
func QueueDiskWrite(filePath string, tags []string) {
	state.LockWriteQueue()
	defer state.UnlockWriteQueue()

	writeQueue := state.GetWriteQueue()

	// Remove any existing queue item for this file (deduplication)
	for i := len(writeQueue) - 1; i >= 0; i-- {
		if writeQueue[i].FilePath == filePath {
			writeQueue = append(writeQueue[:i], writeQueue[i+1:]...)
		}
	}

	// Add new item with latest tags
	writeQueue = append(writeQueue, models.WriteQueueItem{
		FilePath:  filePath,
		Tags:      tags,
		Timestamp: time.Now(),
	})

	state.SetWriteQueue(writeQueue)
}

// GetQueueSize returns the current size of the write queue
func GetQueueSize() int {
	state.LockWriteQueue()
	defer state.UnlockWriteQueue()
	return len(state.GetWriteQueue())
}

// ProcessBatchWrites is the background goroutine that persists queued tag changes to disk.
// Runs every 5 seconds, batching multiple tag operations into efficient bulk writes.
func ProcessBatchWrites() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		state.LockWriteQueue()
		writeQueue := state.GetWriteQueue()
		if len(writeQueue) == 0 {
			state.UnlockWriteQueue()
			continue
		}

		// Copy queue and clear it atomically
		items := make([]models.WriteQueueItem, len(writeQueue))
		copy(items, writeQueue)
		state.SetWriteQueue([]models.WriteQueueItem{})
		state.UnlockWriteQueue()

		// Write to disk outside the lock (can take time with APFS)
		serveDir := state.GetServeDir()
		cache := state.GetCache()
		for _, item := range items {
			fullPath := filepath.Join(serveDir, item.FilePath)
			if err := scanner.SetMacOSTags(fullPath, item.Tags); err != nil {
				log.Printf("Error writing tags to disk for %s: %v", item.FilePath, err)
				// Re-queue on failure (will retry in next batch)
				QueueDiskWrite(item.FilePath, item.Tags)
			} else if cache != nil {
				// Also update the database cache
				if err := cache.UpdateFileTags(item.FilePath, item.Tags); err != nil {
					log.Printf("Warning: Failed to update cache for %s: %v", item.FilePath, err)
				}
			}
		}
	}
}

// FlushWriteQueue immediately writes all pending items in the queue to disk
func FlushWriteQueue() {
	state.LockWriteQueue()
	writeQueue := state.GetWriteQueue()
	if len(writeQueue) == 0 {
		state.UnlockWriteQueue()
		return
	}

	items := make([]models.WriteQueueItem, len(writeQueue))
	copy(items, writeQueue)
	state.SetWriteQueue([]models.WriteQueueItem{})
	state.UnlockWriteQueue()

	serveDir := state.GetServeDir()
	cache := state.GetCache()
	for _, item := range items {
		fullPath := filepath.Join(serveDir, item.FilePath)
		if err := scanner.SetMacOSTags(fullPath, item.Tags); err != nil {
			log.Printf("Error flushing tags to disk for %s: %v", item.FilePath, err)
			QueueDiskWrite(item.FilePath, item.Tags)
		} else if cache != nil {
			// Also update the database cache
			if err := cache.UpdateFileTags(item.FilePath, item.Tags); err != nil {
				log.Printf("Warning: Failed to update cache for %s: %v", item.FilePath, err)
			}
		}
	}
}

// StartBatchProcessor starts the background batch write processor
func StartBatchProcessor() {
	ProcessBatchWrites()
}
