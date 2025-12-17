package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/tdsanchez/PostMac/internal/handlers"
	"github.com/tdsanchez/PostMac/internal/persistence"
	"github.com/tdsanchez/PostMac/internal/scanner"
	"github.com/tdsanchez/PostMac/internal/state"
	"github.com/tdsanchez/PostMac/internal/watcher"
)

//go:embed main_template.html main_template.js index_template.html gallery_template.html
var embeddedFiles embed.FS

func init() {
	rand.Seed(time.Now().UnixNano())
	state.Initialize()
}

func main() {
	port := flag.String("port", "8080", "Port to serve on")
	dir := flag.String("dir", ".", "Directory to serve")
	flag.Parse()

	var err error
	serveDir, err := filepath.Abs(*dir)
	if err != nil {
		log.Fatalf("Invalid directory: %v", err)
	}

	if _, err := os.Stat(serveDir); os.IsNotExist(err) {
		log.Fatalf("Directory does not exist: %s", serveDir)
	}

	state.SetServeDir(serveDir)

	// Load from cache or scan directory
	dbCache, err := scanner.LoadOrScanDirectory(serveDir)
	if err != nil {
		log.Fatalf("Failed to load/scan directory: %v", err)
	}
	defer dbCache.Close()

	fmt.Printf("✅ Found %d media files\n", state.GetFileCount())
	fmt.Printf("✅ Found %d tag categories\n\n", state.GetCategoryCount())

	// Set cache for persistence layer
	state.SetCache(dbCache)

	// Start filesystem watcher for auto-rescan
	fsWatcher, err := watcher.New(serveDir, dbCache)
	if err != nil {
		log.Printf("⚠️  Warning: Failed to start filesystem watcher: %v", err)
		log.Println("   Auto-rescan disabled, but manual rescan button still available")
	} else {
		fsWatcher.Start()
	}

	// Set embedded files for handlers
	handlers.SetEmbeddedFiles(embeddedFiles)

	// Register routes
	http.HandleFunc("/", handlers.HandleRoot)
	http.HandleFunc("/tag/", handlers.HandleTag)
	http.HandleFunc("/view/", handlers.HandleViewer)
	http.HandleFunc("/viewer.js", handlers.HandleViewerJS)
	http.HandleFunc("/file/", handlers.HandleFile)
	http.HandleFunc("/api/addtag", handlers.HandleAddTag)
	http.HandleFunc("/api/removetag", handlers.HandleRemoveTag)
	http.HandleFunc("/api/batchaddtag", handlers.HandleBatchAddTag)
	http.HandleFunc("/api/alltags", handlers.HandleGetAllTags)
	http.HandleFunc("/api/filelist", handlers.HandleGetFileList)
	http.HandleFunc("/api/comment", handlers.HandleUpdateComment)
	http.HandleFunc("/api/shutdown", handlers.HandleShutdown)
	http.HandleFunc("/api/rescan", handlers.HandleRescan)
	http.HandleFunc("/api/scanstatus", handlers.HandleScanStatus)
	http.HandleFunc("/api/deletefile", handlers.HandleDeleteFile)
	http.HandleFunc("/api/metadata/", handlers.HandleMetadata)
	http.HandleFunc("/api/quicklook", handlers.HandleQuickLook)
	http.HandleFunc("/api/convert/", handlers.HandleConvert)

	addr := ":" + *port
	url := "http://localhost" + addr

	fmt.Printf("🚀 Media Server Started\n")
	fmt.Printf("📂 Serving: %s\n", serveDir)
	fmt.Printf("🌐 URL: %s\n", url)
	fmt.Printf("⏹  Press Ctrl+C to stop\n\n")

	go func() {
		time.Sleep(500 * time.Millisecond)
		serverReady := state.GetServerReady()
		serverReady <- true
	}()

	// Start background batch write processor
	go persistence.StartBatchProcessor()

	log.Fatal(http.ListenAndServe(addr, nil))
}
