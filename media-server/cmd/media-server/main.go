package main

import (
	"bufio"
	"embed"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/tdsanchez/PostMac/internal/handlers"
	"github.com/tdsanchez/PostMac/internal/persistence"
	"github.com/tdsanchez/PostMac/internal/scanner"
	"github.com/tdsanchez/PostMac/internal/state"
	"github.com/tdsanchez/PostMac/internal/watcher"
)

//go:embed main_template.html main_template.js index_template.html gallery_template.html train_template.html
var embeddedFiles embed.FS

func init() {
	rand.Seed(time.Now().UnixNano())
	state.Initialize()
}

func main() {
	port := flag.String("port", "8080", "Port to serve on")
	noWatch := flag.Bool("no-watch", false, "Disable filesystem watcher")
	useStdin := flag.Bool("stdin", false, "Read file paths from stdin (one absolute path per line)")
	flag.Parse()

	// Read stdin paths (required for incremental scanning mode)
	var stdinPaths []string
	if *useStdin {
		log.Println("📥 Reading file paths from stdin...")
		stdinScanner := bufio.NewScanner(os.Stdin)
		for stdinScanner.Scan() {
			path := strings.TrimSpace(stdinScanner.Text())
			if path != "" {
				stdinPaths = append(stdinPaths, path)
			}
		}
		if err := stdinScanner.Err(); err != nil {
			log.Fatalf("Error reading from stdin: %v", err)
		}
		log.Printf("📥 Read %d file paths from stdin\n", len(stdinPaths))

		// Store stdin paths in state for use in rescans
		state.SetStdinPaths(stdinPaths)
	}

	// Load from cache or process stdin paths
	dbCache, err := scanner.LoadOrScan(stdinPaths, *port)
	if err != nil {
		log.Fatalf("Failed to load/scan: %v", err)
	}
	defer dbCache.Close()

	fmt.Printf("✅ Found %d media files\n", state.GetFileCount())
	fmt.Printf("✅ Found %d tag categories\n\n", state.GetCategoryCount())

	// Set cache for persistence layer
	state.SetCache(dbCache)

	// Start filesystem watcher for auto-rescan (unless disabled)
	if !*noWatch && len(stdinPaths) > 0 {
		fsWatcher, err := watcher.NewFromPaths(stdinPaths, dbCache)
		if err != nil {
			log.Printf("⚠️  Warning: Failed to start filesystem watcher: %v", err)
			log.Println("   Auto-rescan disabled, but manual rescan button still available")
		} else {
			fsWatcher.Start()
		}
	} else if *noWatch {
		log.Println("📡 Filesystem watcher disabled (--no-watch flag)")
		log.Println("   Use the Rescan button to manually refresh the library")
	}

	// Set embedded files for handlers
	handlers.SetEmbeddedFiles(embeddedFiles)

	// Register routes
	http.HandleFunc("/", handlers.HandleRoot)
	http.HandleFunc("/tag/", handlers.HandleTag)
	http.HandleFunc("/view/", handlers.HandleViewer)
	http.HandleFunc("/train", handlers.HandleTraining)
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
	http.HandleFunc("/api/metadata", handlers.HandleMetadata)
	http.HandleFunc("/api/quicklook", handlers.HandleQuickLook)
	http.HandleFunc("/api/convert/", handlers.HandleConvert)
	http.HandleFunc("/api/search", handlers.HandleSearch)
	http.HandleFunc("/api/log-invalid-path", handlers.HandleLogInvalidPath)
	http.HandleFunc("/api/datedecision", handlers.HandleSaveDateDecision)
	http.HandleFunc("/api/datestats", handlers.HandleGetDateStats)
	http.HandleFunc("/api/datepredict", handlers.HandleGetDatePrediction)
	http.HandleFunc("/api/scan-progress", handlers.HandleScanProgress)

	addr := ":" + *port
	url := "http://localhost" + addr

	fmt.Printf("🚀 Media Server Started\n")
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
