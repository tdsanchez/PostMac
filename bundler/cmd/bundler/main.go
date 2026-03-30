// bundler — self-contained HTML document packager
//
// Cousin of publisher. Reads file paths from stdin and builds a single
// self-contained HTML file with all content stored as gzip+base64 inline.
// Artifacts are intentionally large and hard to crawl — the document IS
// the delivery mechanism.
//
// Markdown is rendered client-side via marked.js (39KB, no WASM required).
// Unsupported file types are flagged at build time and shown as placeholders.
//
// Usage:
//   cat paths.txt | bundler --output bundle.html --title "My Bundle"
//   find /some/dir | bundler --output docs.html
//
// Flags:
//   --output  path to write HTML file (default: bundle.html)
//   --title   document title (default: "Published")
//   --cache   path to cache.db (default: ~/.media-server-conf/cache.db)
//
// Compression:
//   text/* file contents are gzip-compressed before base64 encoding.
//   Images and PDFs are skipped (already internally compressed).
//   Browser decompresses using DecompressionStream('gzip') — zero deps.

package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// PublishFile is the in-memory representation of one published item.
type PublishFile struct {
	ID         int64    `json:"id"`
	Name       string   `json:"name"`
	AbsPath    string   `json:"path"`
	MIME       string   `json:"mime"`
	Tags       []string `json:"tags"`
	Comment    string   `json:"comment"`
	Created    int64    `json:"created"`    // unix seconds
	Size       int64    `json:"size"`       // original file size
	Content    string   `json:"content"`    // base64(gzip(data)) or base64(data)
	Compressed bool     `json:"compressed"` // true if content is gzip compressed
}

// PublishManifest is the top-level JSON blob embedded in the HTML.
type PublishManifest struct {
	Title       string        `json:"title"`
	PublishedAt int64         `json:"published_at"`
	ServerURL   string        `json:"server_url,omitempty"`
	Paths       []string      `json:"paths,omitempty"`
	Files       []PublishFile `json:"files"`
}

func main() {
	outputPath := flag.String("output", "bundle.html", "output HTML file path")
	title      := flag.String("title", "Published", "magazine title")
	cachePath  := flag.String("cache", "", "path to cache.db (default: ~/.media-server-conf/cache.db)")
	serverURL  := flag.String("server", "", "media-server base URL for live tag editing (e.g. http://localhost:9192)")
	passphrase := flag.String("passphrase", "", "encrypt document with this passphrase (AES-256-GCM + PBKDF2)")
	mode       := flag.String("mode", "magazine", "output mode: magazine | wordcloud")
	indexPath  := flag.String("index", "", "path to index.json (required for --mode wordcloud)")
	ringURL    := flag.String("ring", "", "URL of ring.json index for ← prev | index | next → nav")
	artifactURL := flag.String("url", "", "this artifact's own URL (used to locate it in ring.json)")
	flag.Parse()

	if *mode == "wordcloud" {
		if err := wordcloudMode(*indexPath, *outputPath, *title, *serverURL); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Load marked.js for JS-native markdown rendering
	var markedJS string
	markedPath := filepath.Join(repoRoot(), "assets", "marked.min.js")
	if markedBytes, err := os.ReadFile(markedPath); err == nil {
		markedJS = string(markedBytes)
		fmt.Fprintf(os.Stderr, "marked.js: %s\n", formatSize(int64(len(markedBytes))))
	} else {
		fmt.Fprintln(os.Stderr, "warning: marked.min.js not found — markdown will fall back to <pre>")
	}

	// Resolve cache.db
	resolvedCache := *cachePath
	if resolvedCache == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			resolvedCache = filepath.Join(home, ".media-server-conf", "cache.db")
		}
	}

	var cacheDB *sql.DB
	if resolvedCache != "" {
		db, err := sql.Open("sqlite3", resolvedCache+"?mode=ro&_foreign_keys=on")
		if err == nil {
			if pingErr := db.Ping(); pingErr == nil {
				cacheDB = db
				defer cacheDB.Close()
				fmt.Fprintf(os.Stderr, "cache: using %s\n", resolvedCache)
			}
		}
		if cacheDB == nil {
			fmt.Fprintf(os.Stderr, "cache: not available, proceeding without metadata enrichment\n")
		}
	}

	// Read file paths from stdin
	var files []PublishFile
	var nextID int64 = 1
	var totalRaw, totalPacked int64

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	for scanner.Scan() {
		path := strings.TrimSpace(scanner.Text())
		if path == "" {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skip (stat): %s\n", path)
			continue
		}
		if info.IsDir() {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skip (read): %s\n", path)
			continue
		}

		mimeType := mimeForFile(path)
		if !isRenderableMIME(mimeType) {
			fmt.Fprintf(os.Stderr, "unsupported type: %s (%s) — will show placeholder\n", filepath.Base(path), mimeType)
		}
		content, compressed := packContent(data, mimeType)

		pf := PublishFile{
			ID:         nextID,
			Name:       filepath.Base(path),
			AbsPath:    path,
			MIME:       mimeType,
			Tags:       []string{},
			Created:    info.ModTime().Unix(),
			Size:       info.Size(),
			Content:    content,
			Compressed: compressed,
		}
		nextID++

		if cacheDB != nil {
			enrichFromCache(cacheDB, path, &pf)
		}

		packedBytes := len(content) * 3 / 4 // approximate decoded size
		totalRaw += info.Size()
		totalPacked += int64(packedBytes)

		flag := ""
		if compressed {
			flag = " [gz]"
		}
		fmt.Fprintf(os.Stderr, "added: %s (%s, %s%s)\n", pf.Name, mimeType, formatSize(pf.Size), flag)
		files = append(files, pf)
	}

	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "no files — nothing to publish")
		os.Exit(1)
	}

	var paths []string
	for _, f := range files {
		paths = append(paths, f.AbsPath)
	}
	manifest := PublishManifest{
		Title:       *title,
		PublishedAt: time.Now().Unix(),
		ServerURL:   *serverURL,
		Paths:       paths,
		Files:       files,
	}

	ringNav := fetchRingNav(*ringURL, *artifactURL)

	if err := writeHTML(*outputPath, manifest, markedJS, *passphrase, ringNav); err != nil {
		fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
		os.Exit(1)
	}

	info, _ := os.Stat(*outputPath)
	renderMode := "JS-only"
	if markedJS != "" {
		renderMode = "JS+marked.js"
	}
	fmt.Printf("published: %s (%d files, %s, %.1f MB)\n",
		*outputPath, len(files), renderMode, float64(info.Size())/1048576)
}

// packContent compresses text content with gzip; skips already-compressed formats.
func packContent(data []byte, mimeType string) (b64 string, compressed bool) {
	if shouldCompress(mimeType) {
		if gz, err := gzipCompress(data); err == nil && len(gz) < len(data) {
			return base64.StdEncoding.EncodeToString(gz), true
		}
	}
	return base64.StdEncoding.EncodeToString(data), false
}

// shouldCompress returns true for MIME types that benefit from gzip.
func shouldCompress(mimeType string) bool {
	switch {
	case strings.HasPrefix(mimeType, "text/"):
		return true
	case mimeType == "application/json":
		return true
	case mimeType == "image/svg+xml":
		return true
	// Skip: image/jpeg, image/png, image/webp, application/pdf, video/*, audio/*
	// These are already compressed internally.
	default:
		return false
	}
}

// gzipCompress compresses data with gzip best compression.
func gzipCompress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// enrichFromCache pulls tags, comment, and birth time from cache.db.
func enrichFromCache(db *sql.DB, absPath string, pf *PublishFile) {
	var fileID int64
	var comment sql.NullString
	var osBirthTime sql.NullInt64

	row := db.QueryRow(`SELECT id, comment, os_birth_time FROM files WHERE abs_path = ?`, absPath)
	if err := row.Scan(&fileID, &comment, &osBirthTime); err != nil {
		return
	}
	if comment.Valid && comment.String != "" {
		pf.Comment = comment.String
	}
	if osBirthTime.Valid && osBirthTime.Int64 > 0 {
		pf.Created = osBirthTime.Int64
	}

	rows, err := db.Query(`SELECT tag_name FROM tags WHERE file_id = ?`, fileID)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err == nil {
			pf.Tags = append(pf.Tags, tag)
		}
	}
}

// fetchRingNav fetches ring.json and returns a nav HTML string for prev/next navigation.
// Returns "" if ringURL or artifactURL is empty, fetch fails, or artifact not found in ring.
func fetchRingNav(ringURL, artifactURL string) string {
	if ringURL == "" || artifactURL == "" {
		return ""
	}

	type RingArtifact struct {
		URL   string `json:"url"`
		Title string `json:"title"`
		Date  string `json:"date"`
	}
	type RingIndex struct {
		Artifacts []RingArtifact `json:"artifacts"`
	}

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(ringURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ring: fetch failed: %v — skipping nav\n", err)
		return ""
	}
	defer resp.Body.Close()

	var ring RingIndex
	if err := json.NewDecoder(resp.Body).Decode(&ring); err != nil {
		fmt.Fprintf(os.Stderr, "ring: parse failed: %v — skipping nav\n", err)
		return ""
	}

	// Sort by date ascending
	sort.Slice(ring.Artifacts, func(i, j int) bool {
		return ring.Artifacts[i].Date < ring.Artifacts[j].Date
	})

	// Find current artifact index
	cur := -1
	for i, a := range ring.Artifacts {
		if a.URL == artifactURL {
			cur = i
			break
		}
	}
	if cur == -1 {
		fmt.Fprintf(os.Stderr, "ring: artifact URL not found in ring.json — skipping nav\n")
		return ""
	}

	fmt.Fprintf(os.Stderr, "ring: found at position %d of %d\n", cur+1, len(ring.Artifacts))

	n := len(ring.Artifacts)
	prev := &ring.Artifacts[(cur-1+n)%n]
	next := &ring.Artifacts[(cur+1)%n]

	var sb strings.Builder
	sb.WriteString(`<nav id="ring-nav" style="background:#1a1a1a;border-top:1px solid #2a2a2a;padding:10px 16px 28px;font-family:monospace;font-size:12px;color:#666;display:flex;gap:16px;align-items:center;position:fixed;bottom:0;left:0;right:0;z-index:9999;">`)
	fmt.Fprintf(&sb, `<a href="%s" style="color:#888;text-decoration:none;" title="%s">← %s</a>`, prev.URL, prev.Title, truncate(prev.Title, 40))
	fmt.Fprintf(&sb, `<a href="%s" style="color:#555;text-decoration:none;margin:0 auto;">index</a>`, ringURL)
	fmt.Fprintf(&sb, `<a href="%s" style="color:#888;text-decoration:none;" title="%s">%s →</a>`, next.URL, next.Title, truncate(next.Title, 40))
	sb.WriteString(`</nav>`)
	return sb.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

// writeHTML injects all data into the HTML template.
func writeHTML(outputPath string, manifest PublishManifest, markedJS, passphrase, ringNav string) error {
	jsonBytes, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	var dataJSON string
	if passphrase != "" {
		enc, err := encryptManifest(jsonBytes, passphrase)
		if err != nil {
			return fmt.Errorf("encrypt: %w", err)
		}
		dataJSON = string(enc)
		fmt.Fprintf(os.Stderr, "encrypted: AES-256-GCM + PBKDF2-SHA256 (100k iterations)\n")
	} else {
		dataJSON = string(jsonBytes)
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	defer out.Close()

	w := bufio.NewWriterSize(out, 8*1024*1024)

	html := htmlTemplate
	html = strings.Replace(html, "{{TITLE}}", manifest.Title, 2)
	html = strings.Replace(html, "{{DATA}}", dataJSON, 1)
	html = strings.Replace(html, "{{MARKED_JS}}", markedJS, 1)
	html = strings.Replace(html, "{{RING_NAV}}", ringNav, 1)

	fmt.Fprint(w, html)
	return w.Flush()
}

// pbkdf2SHA256 derives a key using PBKDF2-HMAC-SHA256 (no external deps).
func pbkdf2SHA256(password, salt []byte, iter, keyLen int) []byte {
	prf := hmac.New(sha256.New, password)
	hashLen := prf.Size()
	numBlocks := (keyLen + hashLen - 1) / hashLen
	dk := make([]byte, 0, numBlocks*hashLen)
	T := make([]byte, hashLen)
	U := make([]byte, hashLen)
	var counter [4]byte
	for block := 1; block <= numBlocks; block++ {
		binary.BigEndian.PutUint32(counter[:], uint32(block))
		prf.Reset()
		prf.Write(salt)
		prf.Write(counter[:])
		copy(T, prf.Sum(nil))
		copy(U, T)
		for n := 2; n <= iter; n++ {
			prf.Reset()
			prf.Write(U)
			sum := prf.Sum(nil)
			copy(U, sum)
			for x := range T {
				T[x] ^= U[x]
			}
		}
		dk = append(dk, T...)
	}
	return dk[:keyLen]
}

// encryptManifest encrypts jsonBytes with AES-256-GCM, key derived via PBKDF2-SHA256.
// Returns a JSON envelope: {"encrypted":"true","salt":"...","iv":"...","data":"..."}.
func encryptManifest(jsonBytes []byte, passphrase string) ([]byte, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("rand salt: %w", err)
	}
	iv := make([]byte, 12)
	if _, err := rand.Read(iv); err != nil {
		return nil, fmt.Errorf("rand iv: %w", err)
	}
	key := pbkdf2SHA256([]byte(passphrase), salt, 100000, 32)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	ciphertext := gcm.Seal(nil, iv, jsonBytes, nil)
	envelope := map[string]string{
		"encrypted": "true",
		"salt":      base64.StdEncoding.EncodeToString(salt),
		"iv":        base64.StdEncoding.EncodeToString(iv),
		"data":      base64.StdEncoding.EncodeToString(ciphertext),
	}
	return json.Marshal(envelope)
}


// repoRoot walks up from the executable to find go.mod.
func repoRoot() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	dir := filepath.Dir(exe)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "."
}

func mimeForFile(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".pdf":
		return "application/pdf"
	case ".mp4":
		return "video/mp4"
	case ".mov":
		return "video/quicktime"
	case ".webm":
		return "video/webm"
	case ".mp3":
		return "audio/mpeg"
	case ".m4a":
		return "audio/mp4"
	case ".md", ".markdown":
		return "text/markdown"
	case ".html", ".htm":
		return "text/html"
	case ".txt":
		return "text/plain"
	case ".webarchive":
		return "application/x-webarchive"
	default:
		t := mime.TypeByExtension(ext)
		if t != "" {
			return t
		}
		return "application/octet-stream"
	}
}

func isRenderableMIME(mime string) bool {
	switch {
	case mime == "text/markdown", mime == "text/html", mime == "text/plain":
		return true
	case mime == "application/pdf":
		return true
	case strings.HasPrefix(mime, "image/jpeg"), strings.HasPrefix(mime, "image/png"),
		strings.HasPrefix(mime, "image/gif"), strings.HasPrefix(mime, "image/webp"),
		strings.HasPrefix(mime, "image/svg"):
		return true
	case strings.HasPrefix(mime, "video/"), strings.HasPrefix(mime, "audio/"):
		return true
	}
	return false
}

func formatSize(n int64) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1048576:
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(n)/1048576)
	}
}

// readWordcloud2JS finds wordcloud2.min.js in the assets/ directory.
func readWordcloud2JS() string {
	candidate := filepath.Join(repoRoot(), "assets", "wordcloud2.min.js")
	if data, err := os.ReadFile(candidate); err == nil {
		return string(data)
	}
	return ""
}

// wordcloudMode builds a self-contained wordcloud HTML artifact from index.json.
func wordcloudMode(indexPath, outputPath, title, serverURL string) error {
	if indexPath == "" {
		return fmt.Errorf("--index is required for --mode wordcloud")
	}
	indexBytes, err := os.ReadFile(indexPath)
	if err != nil {
		return fmt.Errorf("read index: %w", err)
	}
	fmt.Fprintf(os.Stderr, "index: %s\n", formatSize(int64(len(indexBytes))))

	// Parse just enough to get stats for the header
	var idx struct {
		Paths        []json.RawMessage `json:"paths"`
		Freqs        []json.RawMessage `json:"freqs"`
		UntaggedCount int              `json:"untagged_count"`
	}
	if err := json.Unmarshal(indexBytes, &idx); err != nil {
		return fmt.Errorf("parse index: %w", err)
	}
	fileCount    := len(idx.Paths)
	untaggedCount := idx.UntaggedCount

	// Gzip-compress the full index JSON
	compressed, err := gzipCompress(indexBytes)
	if err != nil {
		return fmt.Errorf("compress index: %w", err)
	}
	indexB64 := base64.StdEncoding.EncodeToString(compressed)
	fmt.Fprintf(os.Stderr, "index compressed: %s → %s (%.0f%% reduction)\n",
		formatSize(int64(len(indexBytes))),
		formatSize(int64(len(compressed))),
		100*(1-float64(len(compressed))/float64(len(indexBytes))))

	// Load wordcloud2.min.js
	wc2js := readWordcloud2JS()
	if wc2js == "" {
		return fmt.Errorf("wordcloud2.min.js not found in assets/ — run: curl -sL https://cdnjs.cloudflare.com/ajax/libs/wordcloud2.js/1.2.2/wordcloud2.min.js -o assets/wordcloud2.min.js")
	}
	fmt.Fprintf(os.Stderr, "wordcloud2.js: %s\n", formatSize(int64(len(wc2js))))

	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer out.Close()

	w := bufio.NewWriterSize(out, 8*1024*1024)
	html := wordcloudTemplate
	html = strings.Replace(html, "{{TITLE}}", title, 2)
	html = strings.Replace(html, "{{FILE_COUNT}}", fmt.Sprintf("%d", fileCount), 1)
	html = strings.Replace(html, "{{UNTAGGED_COUNT}}", fmt.Sprintf("%d", untaggedCount), 1)
	html = strings.Replace(html, "{{INDEX_B64}}", indexB64, 1)
	html = strings.Replace(html, "{{SERVER_URL}}", serverURL, 1)
	html = strings.Replace(html, "{{WORDCLOUD2_JS}}", wc2js, 1)
	fmt.Fprint(w, html)
	if err := w.Flush(); err != nil {
		return err
	}

	info, _ := os.Stat(outputPath)
	fmt.Printf("wordcloud: %s (%d images, %.1f MB)\n", outputPath, fileCount, float64(info.Size())/1048576)
	return nil
}

// wordcloudTemplate — self-contained offline wordcloud artifact.
// Placeholders: {{TITLE}} (×2), {{FILE_COUNT}}, {{UNTAGGED_COUNT}},
//               {{INDEX_B64}}, {{SERVER_URL}}, {{WORDCLOUD2_JS}}
const wordcloudTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>{{TITLE}}</title>
<style>
* { box-sizing: border-box; margin: 0; padding: 0; }
body { background: #111; color: #ddd; font-family: Helvetica, Arial, sans-serif; font-size: 13px; }
#header { padding: 12px 20px 6px; color: #555; font-size: 12px; line-height: 1.9; }
#controls {
  padding: 6px 20px 10px;
  display: flex; align-items: center; gap: 8px; flex-wrap: wrap;
  border-bottom: 1px solid #1e1e1e;
}
#controls label { color: #666; font-size: 11px; }
.filter-btn {
  background: #1a1a1a; border: 1px solid #2e2e2e; color: #666;
  padding: 3px 11px; border-radius: 3px; font-size: 11px; cursor: pointer;
}
.filter-btn:hover { border-color: #555; color: #aaa; }
.filter-btn.active { background: #1e2d45; border-color: #4a9eff; color: #4a9eff; }
#oversized-toggle { accent-color: #4a9eff; cursor: pointer; margin-right: 2px; }
#filter-info { color: #2e2e2e; font-size: 11px; margin-left: auto; }
#loading { color: #555; font-size: 13px; padding: 40px 20px; text-align: center; }
#canvas { display: block; margin: 0 auto; cursor: pointer; }
#results { display: none; border-top: 1px solid #1e1e1e; padding: 16px 20px; }
#results-header { font-size: 12px; color: #777; margin-bottom: 12px; }
.word-label  { color: #f5a623; font-size: 14px; }
.tag-label   { color: #4a9eff; font-size: 14px; }
.vi-label    { color: #88bbff; font-size: 14px; }
#gallery {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 4px;
}
.thumb {
  position: relative; aspect-ratio: 1;
  overflow: hidden; background: #1a1a1a; border-radius: 2px;
}
.thumb a { display: block; width: 100%; height: 100%; }
.thumb img {
  width: 100%; height: 100%; object-fit: cover; display: block;
  transition: opacity 0.1s;
}
.thumb img:hover { opacity: 0.7; cursor: pointer; }
.thumb.untagged-outlier { outline: 2px solid #f97316; }
#gallery-note { color: #333; font-size: 11px; margin-top: 10px; }
#nav-indicator { display:none; color:#444; font-size:11px; margin-left:16px; vertical-align:middle; }
#nav-pos { color:#666; }
#server-badge {
  font-size: 10px; padding: 2px 8px; border-radius: 3px; margin-left: 8px;
  background: #1a2a1a; border: 1px solid #2a4a2a; color: #4a8a4a;
}
#server-badge.offline { background: #2a1a1a; border-color: #4a2a2a; color: #8a4a4a; }
</style>
</head>
<body>
<div id="header">
  {{TITLE}} &mdash; {{FILE_COUNT}} images &mdash;
  <span style="color:#88bbff">light blue = vi- visual</span> &nbsp;
  <span style="color:#4ade80">green = vi- + one property</span> &nbsp;
  <span style="color:#f5a623">orange = vi- + two properties</span> &nbsp;
  <span style="color:#4a9eff">blue = Finder tag</span> &mdash;
  {{UNTAGGED_COUNT}} untagged &mdash; click any word to open gallery
  <span id="server-badge">checking server…</span>
</div>

<div id="controls">
  <label>max group size:</label>
  <button class="filter-btn" data-max="Infinity">all</button>
  <button class="filter-btn" data-max="50000">≤50k</button>
  <button class="filter-btn" data-max="20000">≤20k</button>
  <button class="filter-btn" data-max="5000">≤5k</button>
  <button class="filter-btn active" data-max="2000">≤2k</button>
  <button class="filter-btn" data-max="500">≤500</button>
  <button class="filter-btn" data-max="200">≤200</button>
  &nbsp;&nbsp;
  <label>
    <input type="checkbox" id="oversized-toggle" checked>
    hide OVERSIZED_ groups
  </label>
  <span id="filter-info"></span>
</div>

<div id="loading">decompressing index…</div>
<canvas id="canvas" style="display:none"></canvas>

<div id="results">
  <div id="results-header">
    <span id="active-word" class="word-label"></span>
    &mdash; <span id="result-count"></span>
    <span id="nav-indicator">&larr;&rarr;&nbsp; <span id="nav-pos"></span></span>
    &nbsp;&nbsp;
    <button id="browse-btn" style="
      display:none; background:#1a1a1a; border:1px solid #2e2e2e; color:#666;
      padding:3px 12px; border-radius:3px; font-size:11px; cursor:pointer;
      margin-left:8px; vertical-align:middle;">browse</button>
    <button id="tag-btn" style="
      display:none; background:#1a1a1a; border:1px solid #2e2e2e; color:#666;
      padding:3px 12px; border-radius:3px; font-size:11px; cursor:pointer;
      margin-left:4px; vertical-align:middle;">tag this group</button>
  </div>
  <div id="gallery"></div>
  <div id="gallery-note"></div>
</div>

<script>
{{WORDCLOUD2_JS}}
</script>
<script>
const INDEX_B64  = "{{INDEX_B64}}";
const SERVER_URL = "{{SERVER_URL}}";

var ALL_WORDS   = [];
var taggedWords = new Set();
var INDEX       = null;
var GALLERY_LIMIT = 1500;

// PORT/TAG_PORT derived from SERVER_URL when available
var PORT     = SERVER_URL ? SERVER_URL.replace(/\/$/, '') : null;
var TAG_PORT = PORT;

// ── Decompression (same as publisher magazine viewer) ─────────────────────────
async function gzipDecompress(b64) {
  const compressed = Uint8Array.from(atob(b64), c => c.charCodeAt(0));
  const ds = new DecompressionStream('gzip');
  const writer = ds.writable.getWriter();
  writer.write(compressed);
  writer.close();
  const chunks = [];
  const reader = ds.readable.getReader();
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    chunks.push(value);
  }
  const total = chunks.reduce((s, c) => s + c.length, 0);
  const out = new Uint8Array(total);
  let off = 0;
  for (const c of chunks) { out.set(c, off); off += c.length; }
  return out;
}

// ── Arrow-key navigation state ────────────────────────────────────────────────
var navList  = [];
var navIndex = -1;

var canvas     = document.getElementById('canvas');
var filterInfo = document.getElementById('filter-info');

var currentMax    = 2000;
var hideOversized = true;

function applyFilter() {
  return ALL_WORDS.filter(function(w) {
    var word  = w[0];
    var count = w[1];
    if (hideOversized && word.startsWith('OVERSIZED_')) return false;
    if (count > currentMax) return false;
    return true;
  });
}

function drawCloud() {
  var words   = applyFilter();
  var maxFreq = words.length
    ? Math.max.apply(null, words.map(function(w) { return w[1]; }))
    : 1;

  var hidden = ALL_WORDS.length - words.length;
  filterInfo.textContent = hidden > 0
    ? '(' + hidden.toLocaleString() + ' group' + (hidden === 1 ? '' : 's') + ' hidden)'
    : '';

  var ctx = canvas.getContext('2d');
  ctx.fillStyle = '#111';
  ctx.fillRect(0, 0, canvas.width, canvas.height);

  if (!words.length) {
    ctx.fillStyle = '#333';
    ctx.font = '16px Helvetica';
    ctx.textAlign = 'center';
    ctx.fillText('No groups match current filter', canvas.width / 2, canvas.height / 2);
    return;
  }

  WordCloud(canvas, {
    list: words,
    gridSize: 6,
    weightFactor: function(size) {
      return Math.max(10, size / maxFreq * canvas.width / 10);
    },
    fontFamily: 'Helvetica, Arial, sans-serif',
    color: function(word) {
      if (word.startsWith('vi-vht-') || word.startsWith('vi-vha-') || word.startsWith('vi-vsh-') ||
          word.startsWith('vi-p2-') || word.startsWith('vi-x-')) {
        var ambers = ['#f5a623','#f59f00','#fb923c','#f97316','#ea8c00'];
        return ambers[Math.floor(Math.random() * ambers.length)];
      } else if (word.startsWith('vi-vh-') || word.startsWith('vi-vt-') ||
                 word.startsWith('vi-va-') || word.startsWith('vi-vs-') ||
                 word.startsWith('vi-p3-') || word.startsWith('vi-p-')) {
        var greens = ['#4ade80','#22c55e','#86efac','#34d399','#6ee7b7'];
        return greens[Math.floor(Math.random() * greens.length)];
      } else if (word.startsWith('vi-')) {
        var blues = ['#4a9eff','#7ecfff','#88bbff','#5ab4ff','#3388ee','#66aaff'];
        return blues[Math.floor(Math.random() * blues.length)];
      } else if (taggedWords.has(word)) {
        var lbl = ['#4a9eff','#7ecfff','#a0d8ff'];
        return lbl[Math.floor(Math.random() * lbl.length)];
      } else {
        var teals = ['#22d3ee','#06b6d4','#38bdf8','#67e8f9','#0ea5e9'];
        return teals[Math.floor(Math.random() * teals.length)];
      }
    },
    backgroundColor: '#111',
    rotateRatio: 0.15,
    minSize: 8,
    drawOutOfBound: false,
    shuffle: false,
    click: function(item) {
      var word    = item[0];
      var indices = INDEX.index[word] || [];
      var paths   = indices.map(function(i) { return INDEX.paths[i]; });
      showGallery(word, paths);
    },
    hover: function(item) {
      canvas.style.cursor = item ? 'pointer' : 'default';
    }
  });
  buildNavList();
}

// ── Gallery ───────────────────────────────────────────────────────────────────

function showGallery(word, paths) {
  var isVi     = word.startsWith('vi-');
  var isTagged = taggedWords.has(word);
  var labelCls = isVi ? 'vi-label' : (isTagged ? 'tag-label' : 'word-label');

  var el = document.getElementById('active-word');
  el.textContent = word;
  el.className   = labelCls;

  document.getElementById('result-count').textContent =
    paths.length.toLocaleString() + ' image' + (paths.length === 1 ? '' : 's');

  var gallery = document.getElementById('gallery');
  gallery.innerHTML = '';
  document.getElementById('gallery-note').textContent = '';

  var display  = paths.slice(0, GALLERY_LIMIT);
  var groupTag = isTagged ? word : null;

  display.forEach(function(p) {
    var div = document.createElement('div');
    div.className = 'thumb';
    if (PORT) {
      var imgUrl  = PORT + '/file/'      + encodeURIComponent(p);
      var viewUrl = PORT + '/view/All?file=' + encodeURIComponent(p);
      div.innerHTML =
        '<a href="' + viewUrl + '" target="_blank">' +
        '<img src="' + imgUrl + '" loading="lazy" title="' + p.split('/').pop() + '">' +
        '</a>';
      if (groupTag && TAG_PORT) {
        fetch(TAG_PORT + '/api/filetags?path=' + encodeURIComponent(p))
          .then(function(r) { return r.json(); })
          .then(function(data) {
            if ((data.tags || []).indexOf(groupTag) === -1) {
              div.classList.add('untagged-outlier');
            }
          })
          .catch(function() {});
      }
    } else {
      // Offline: show filename only
      div.innerHTML = '<div style="padding:6px;font-size:9px;color:#555;word-break:break-all;">' +
        p.split('/').pop() + '</div>';
    }
    gallery.appendChild(div);
  });

  if (paths.length > GALLERY_LIMIT) {
    document.getElementById('gallery-note').textContent =
      'Showing ' + GALLERY_LIMIT.toLocaleString() + ' of ' +
      paths.length.toLocaleString() + (PORT ? ' — open in viewer for full set' : '');
  }

  document.getElementById('browse-btn').style.display = (isVi && PORT) ? 'inline-block' : 'none';
  document.getElementById('browse-btn').onclick = function() {
    window.open(TAG_PORT + '/viewer?cluster=' + encodeURIComponent(word), '_blank');
  };

  var tagBtn = document.getElementById('tag-btn');
  tagBtn.style.display = PORT ? 'inline-block' : 'none';
  tagBtn.disabled      = false;
  if (taggedWords.has(word)) {
    tagBtn.textContent       = '\u2713 tagged';
    tagBtn.style.color       = '#4a9eff';
    tagBtn.style.borderColor = '#4a9eff';
  } else {
    tagBtn.textContent       = 'tag this group';
    tagBtn.style.color       = '#666';
    tagBtn.style.borderColor = '#2e2e2e';
  }
  tagBtn.onclick = function() { tagThisGroup(word, paths); };

  for (var i = 0; i < navList.length; i++) {
    if (navList[i].word === word) { navIndex = i; break; }
  }
  updateNavIndicator();

  document.getElementById('results').style.display = 'block';
  document.getElementById('results').scrollIntoView({ behavior: 'smooth', block: 'start' });
}

// ── Tag writeback (requires PORT) ─────────────────────────────────────────────

function tagThisGroup(word, paths) {
  if (!PORT) return;
  var btn = document.getElementById('tag-btn');
  btn.disabled    = true;
  btn.textContent = 'tagging ' + paths.length.toLocaleString() + ' files\u2026';

  fetch(PORT + '/api/batchaddtag', {
    method:  'POST',
    headers: {'Content-Type': 'application/json'},
    body:    JSON.stringify({ tag: word, filePaths: paths })
  })
  .then(function(r) { return r.json(); })
  .then(function(data) {
    var n = data.count || paths.length;
    btn.textContent       = '\u2713 tagged ' + n.toLocaleString() + ' files';
    btn.style.color       = '#4a9eff';
    btn.style.borderColor = '#4a9eff';
    taggedWords.add(word);
  })
  .catch(function() {
    btn.textContent = 'error \u2014 is media server running?';
    btn.disabled    = false;
  });
}

// ── Arrow-key navigation ──────────────────────────────────────────────────────

function buildNavList() {
  var viWords = ALL_WORDS.filter(function(w) { return w[0].startsWith('vi-'); });
  var sorted  = viWords.slice().sort(function(a, b) { return a[1] - b[1]; });
  navList = sorted.map(function(w) {
    var indices = INDEX.index[w[0]] || [];
    return { word: w[0], paths: indices.map(function(i) { return INDEX.paths[i]; }) };
  });
  var current = document.getElementById('active-word').textContent;
  navIndex = -1;
  if (current) {
    for (var i = 0; i < navList.length; i++) {
      if (navList[i].word === current) { navIndex = i; break; }
    }
  }
  updateNavIndicator();
}

function updateNavIndicator() {
  var el  = document.getElementById('nav-indicator');
  var pos = document.getElementById('nav-pos');
  if (navList.length === 0) { el.style.display = 'none'; return; }
  el.style.display = 'inline';
  pos.textContent  = navIndex >= 0
    ? (navIndex + 1) + '\u202f/\u202f' + navList.length
    : '\u2014\u202f/\u202f' + navList.length;
}

document.addEventListener('keydown', function(e) {
  if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') return;
  if (!navList.length) return;
  if (e.key === 'ArrowLeft') {
    navIndex = navIndex <= 0 ? navList.length - 1 : navIndex - 1;
  } else if (e.key === 'ArrowRight') {
    navIndex = navIndex < 0 ? 0 : (navIndex >= navList.length - 1 ? 0 : navIndex + 1);
  } else { return; }
  e.preventDefault();
  var item = navList[navIndex];
  showGallery(item.word, item.paths);
  var rendered = applyFilter().some(function(w) { return w[0] === item.word; });
  if (!rendered) {
    for (var i = 0; i < ALL_WORDS.length; i++) {
      if (ALL_WORDS[i][0] === item.word) {
        var count = ALL_WORDS[i][1];
        var presets = [200, 500, 2000, 5000, 20000, 50000, Infinity];
        for (var p = 0; p < presets.length; p++) {
          if (presets[p] >= count) {
            currentMax = presets[p];
            document.querySelectorAll('.filter-btn').forEach(function(b) {
              b.classList.toggle('active', parseFloat(b.dataset.max) === currentMax);
            });
            drawCloud();
            break;
          }
        }
        break;
      }
    }
  }
});

document.querySelectorAll('.filter-btn').forEach(function(btn) {
  btn.addEventListener('click', function() {
    document.querySelectorAll('.filter-btn').forEach(function(b) { b.classList.remove('active'); });
    btn.classList.add('active');
    currentMax = parseFloat(btn.dataset.max);
    drawCloud();
  });
});

document.getElementById('oversized-toggle').addEventListener('change', function(e) {
  hideOversized = e.target.checked;
  drawCloud();
});

// ── Boot: async decompress index, then render ─────────────────────────────────
async function boot() {
  try {
    const bytes = await gzipDecompress(INDEX_B64);
    INDEX = JSON.parse(new TextDecoder().decode(bytes));
    ALL_WORDS   = INDEX.freqs || [];
    taggedWords = new Set(ALL_WORDS.filter(function(w) { return w[2]; }).map(function(w) { return w[0]; }));

    document.getElementById('loading').style.display = 'none';
    canvas.style.display = 'block';
    canvas.width  = Math.min(window.innerWidth - 40, 1400);
    canvas.height = Math.floor(canvas.width * 0.55);
    drawCloud();

    // Server status badge
    var badge = document.getElementById('server-badge');
    if (PORT) {
      fetch(PORT + '/api/alltags', { signal: AbortSignal.timeout(3000) })
        .then(function() {
          badge.textContent = 'server live';
          badge.className   = '';
        })
        .catch(function() {
          badge.textContent = 'server offline — browse only';
          badge.className   = 'offline';
          PORT = null; TAG_PORT = null;
        });
    } else {
      badge.textContent = 'offline — browse only';
      badge.className   = 'offline';
    }
  } catch(e) {
    document.getElementById('loading').textContent = 'error loading index: ' + e.message;
    console.error(e);
  }
}

boot();
</script>
</body>
</html>`

// htmlTemplate — self-contained viewer with gzip decompression support.
// Placeholders: {{TITLE}} (×2), {{DATA}}, {{WASM_EXEC_JS}}, {{WASM_B64}}
const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{TITLE}}</title>
<style>
* { box-sizing: border-box; margin: 0; padding: 0; }
html, body { height: 100%; }
body {
  background: #111;
  color: #ddd;
  font-family: -apple-system, Helvetica, Arial, sans-serif;
  font-size: 13px;
  display: flex;
  flex-direction: column;
}
#header {
  background: #1a1a1a;
  border-bottom: 1px solid #2a2a2a;
  padding: 10px 16px;
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}
#header h1 { font-size: 14px; font-weight: 500; color: #aaa; flex: 1; }
#search {
  background: #222;
  border: 1px solid #333;
  color: #ccc;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
  width: 180px;
}
#search:focus { outline: none; border-color: #555; }
#layout { display: flex; flex: 1; overflow: hidden; }
#sidebar {
  width: 240px;
  min-width: 180px;
  background: #161616;
  border-right: 1px solid #222;
  overflow-y: auto;
  flex-shrink: 0;
}
#tag-bar {
  padding: 8px;
  border-bottom: 1px solid #222;
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
.tag-btn {
  background: #222;
  border: 1px solid #333;
  color: #999;
  padding: 2px 7px;
  border-radius: 10px;
  font-size: 10px;
  cursor: pointer;
}
.tag-btn:hover { background: #2a2a2a; color: #ccc; }
.tag-btn.active { background: #2a4a2a; border-color: #4a8a4a; color: #8fbc8f; }
#file-list { padding: 4px 0; }
.file-item {
  padding: 7px 12px;
  cursor: pointer;
  border-bottom: 1px solid #1e1e1e;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.file-item:hover { background: #1e1e1e; }
.file-item.active { background: #1a2a1a; border-left: 2px solid #4a8a4a; }
.file-name { color: #ccc; font-size: 12px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.file-meta { color: #555; font-size: 10px; }
.file-tags { display: flex; gap: 3px; flex-wrap: wrap; margin-top: 2px; }
.file-tag { background: #1e2e1e; color: #6a9a6a; font-size: 9px; padding: 1px 5px; border-radius: 8px; }
#content { flex: 1; overflow-y: auto; padding: 24px; background: #111; }
#content-inner { max-width: 860px; margin: 0 auto; }
.content-header { margin-bottom: 16px; border-bottom: 1px solid #222; padding-bottom: 12px; }
.content-title { font-size: 15px; color: #ccc; font-weight: 500; margin-bottom: 4px; }
.content-meta { font-size: 11px; color: #555; }
.content-comment { font-size: 12px; color: #888; font-style: italic; margin-top: 6px; }
.content-tags { display: flex; gap: 4px; flex-wrap: wrap; margin-top: 8px; align-items: center; }
.content-tag { background: #1e2e1e; color: #6a9a6a; font-size: 10px; padding: 2px 6px; border-radius: 10px; display: inline-flex; align-items: center; gap: 3px; }
.tag-rm { color: #555; cursor: pointer; font-size: 11px; line-height: 1; }
.tag-rm:hover { color: #e06c75; }
.tag-add-btn { background: none; border: 1px dashed #3a6a3a; color: #4a7a4a; font-size: 10px; padding: 2px 7px; border-radius: 10px; cursor: pointer; user-select: none; }
.tag-add-btn:hover { border-color: #5a9a5a; color: #6aaa6a; }
#tag-input-box { margin-top: 8px; display: flex; gap: 8px; align-items: center; }
#tag-input-field { background: #1a1a1a; border: 1px solid #4a7a4a; color: #ccc; padding: 4px 9px; border-radius: 4px; font-size: 12px; width: 180px; outline: none; }
#tag-input-field:focus { border-color: #6aaa6a; }
img.media { max-width: 100%; display: block; border-radius: 4px; }
embed.media { width: 100%; height: 80vh; border: none; border-radius: 4px; }
video.media, audio.media { width: 100%; }
pre.media {
  background: #1a1a1a;
  padding: 16px;
  border-radius: 4px;
  color: #ccc;
  font-family: 'SF Mono', 'Menlo', monospace;
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}
iframe.media { width: 100%; height: 80vh; border: 1px solid #222; border-radius: 4px; background: #fff; }
#empty { color: #555; text-align: center; margin-top: 80px; font-size: 14px; }
.md-body { color: #e6edf3; line-height: 1.7; font-size: 14px; }
.md-body h1,.md-body h2,.md-body h3,.md-body h4,.md-body h5,.md-body h6 {
  margin-top: 24px; margin-bottom: 12px; font-weight: 600; line-height: 1.3;
  border-bottom: 1px solid #21262d; padding-bottom: 6px; color: #ccc;
}
.md-body h1 { font-size: 1.8em; } .md-body h2 { font-size: 1.4em; } .md-body h3 { font-size: 1.2em; }
.md-body p { margin-bottom: 14px; }
.md-body a { color: #58a6ff; text-decoration: none; }
.md-body a:hover { text-decoration: underline; }
.md-body code {
  background: #1e1e1e; padding: 2px 5px; border-radius: 4px;
  font-family: 'SF Mono', Menlo, monospace; font-size: 0.85em; color: #f0883e;
}
.md-body pre { background: #161b22; padding: 14px; border-radius: 6px; overflow-x: auto; margin-bottom: 14px; }
.md-body pre code { background: none; padding: 0; color: #e6edf3; }
.md-body ul,.md-body ol { padding-left: 2em; margin-bottom: 14px; }
.md-body li { margin-bottom: 4px; }
.md-body blockquote { border-left: 3px solid #30363d; padding-left: 14px; margin-left: 0; color: #8b949e; font-style: italic; }
.md-body table { border-collapse: collapse; width: 100%; margin-bottom: 14px; }
.md-body th,.md-body td { border: 1px solid #30363d; padding: 7px 12px; text-align: left; }
.md-body th { background: #161b22; font-weight: 600; }
.md-body hr { border: 0; border-top: 1px solid #21262d; margin: 20px 0; }
.md-body img { max-width: 100%; border-radius: 4px; }
.md-body del { color: #8b949e; }
#wasm-status { font-size: 10px; color: #555; }
#count-label { font-size: 11px; color: #555; }
#menu-btn {
  display: none; background: none; border: 1px solid #333; color: #888;
  padding: 4px 9px; border-radius: 4px; font-size: 14px; cursor: pointer;
  flex-shrink: 0; line-height: 1;
}
#menu-btn:hover { border-color: #555; color: #ccc; }
@media (max-width: 768px) {
  #menu-btn { display: block; }
  #search { width: 80px; }
  #save-btn { padding: 4px 7px; font-size: 10px; }
  #wasm-status { display: none; }
  #layout { flex-direction: column; }
  #sidebar {
    width: 100%; max-height: 0; overflow: hidden;
    border-right: none; border-bottom: 1px solid #222;
    transition: max-height 0.2s ease;
  }
  #sidebar.open { max-height: 50vh; overflow-y: auto; }
  #content { padding: 14px; }
  #content-inner { max-width: 100%; }
  .file-item { padding: 11px 14px; }
  .tag-btn { padding: 5px 11px; font-size: 11px; }
  .md-body { font-size: 15px; }
  img.media { border-radius: 2px; }
}
#save-btn {
  background: #1a1a1a;
  border: 1px solid #2a2a2a;
  color: #666;
  padding: 4px 10px;
  border-radius: 4px;
  font-size: 11px;
  cursor: pointer;
  flex-shrink: 0;
}
#save-btn:hover { border-color: #555; color: #aaa; }
#lock-screen {
  display: none; position: fixed; inset: 0; background: #111; z-index: 9999;
  flex-direction: column; align-items: center; justify-content: center; gap: 16px;
}
#lock-screen .lock-icon { font-size: 52px; }
#lock-screen .lock-title { font-size: 16px; color: #888; }
#passphrase-input {
  background: #1a1a1a; border: 1px solid #444; color: #ccc;
  padding: 10px 16px; border-radius: 6px; font-size: 16px; width: 280px; outline: none;
}
#passphrase-input:focus { border-color: #6aaa6a; }
#lock-btn {
  background: #2a4a2a; border: 1px solid #4a8a4a; color: #8fbc8f;
  padding: 8px 28px; border-radius: 6px; font-size: 14px; cursor: pointer;
}
#lock-btn:hover { background: #3a6a3a; }
#lock-error { color: #e06c75; font-size: 12px; min-height: 16px; }
</style>
</head>
<body>
<div id="lock-screen">
  <div class="lock-icon">🔒</div>
  <div class="lock-title">This document is encrypted.</div>
  <input id="passphrase-input" type="password" placeholder="Enter passphrase…"
    onkeydown="if(event.key==='Enter')unlock()">
  <button id="lock-btn" onclick="unlock()">Unlock</button>
  <div id="lock-error"></div>
</div>
<div id="header">
  <button id="menu-btn" onclick="document.getElementById('sidebar').classList.toggle('open')" title="Toggle file list">☰</button>
  <h1>{{TITLE}}</h1>
  <span id="wasm-status"></span>
  <input id="search" type="text" placeholder="search…" oninput="filterFiles()">
  <span id="count-label"></span>
  <button id="save-btn" onclick="saveArtifact()" title="Save a copy of this artifact">save a copy</button>
</div>
<div id="layout">
  <div id="sidebar">
    <div id="tag-bar"></div>
    <div id="file-list"></div>
  </div>
  <div id="content">
    <div id="content-inner"><div id="empty">Select a file to view</div></div>
  </div>
</div>
<script>
{{MARKED_JS}}
</script>
<script>
const MANIFEST_RAW = {{DATA}};

let DATA = null;
let activeTag = null;
let activeID = null;
let searchQuery = '';

// Decompress gzip-compressed base64 data using browser-native DecompressionStream.
async function gzipDecompress(b64) {
  const compressed = Uint8Array.from(atob(b64), c => c.charCodeAt(0));
  const ds = new DecompressionStream('gzip');
  const writer = ds.writable.getWriter();
  writer.write(compressed);
  writer.close();
  const chunks = [];
  const reader = ds.readable.getReader();
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    chunks.push(value);
  }
  const total = chunks.reduce((s, c) => s + c.length, 0);
  const out = new Uint8Array(total);
  let off = 0;
  for (const c of chunks) { out.set(c, off); off += c.length; }
  return out;
}

// Decode a file's content, decompressing if needed.
async function decodeContent(f) {
  if (f.compressed) {
    const bytes = await gzipDecompress(f.content);
    return new TextDecoder().decode(bytes);
  }
  return atob(f.content);
}

// Decode content as raw bytes (for binary data URIs).
async function decodeBytes(f) {
  if (f.compressed) {
    return await gzipDecompress(f.content);
  }
  return Uint8Array.from(atob(f.content), c => c.charCodeAt(0));
}


function allTags() {
  const set = new Set();
  DATA.files.forEach(f => (f.tags || []).forEach(t => set.add(t)));
  return [...set].sort();
}

function filteredFiles() {
  return DATA.files.filter(f => {
    if (activeTag && !(f.tags || []).includes(activeTag)) return false;
    if (searchQuery) {
      const q = searchQuery.toLowerCase();
      if (!f.name.toLowerCase().includes(q) &&
          !(f.comment||'').toLowerCase().includes(q) &&
          !(f.tags||[]).join(' ').toLowerCase().includes(q)) return false;
    }
    return true;
  });
}

function renderSidebar() {
  const tags = allTags();
  const tagBar = document.getElementById('tag-bar');
  tagBar.innerHTML = '';
  tags.forEach(t => {
    const btn = document.createElement('span');
    btn.className = 'tag-btn' + (activeTag === t ? ' active' : '');
    btn.textContent = t;
    btn.onclick = () => { activeTag = activeTag === t ? null : t; renderSidebar(); };
    tagBar.appendChild(btn);
  });
  const files = filteredFiles();
  const list = document.getElementById('file-list');
  list.innerHTML = '';
  document.getElementById('count-label').textContent = files.length + ' / ' + DATA.files.length;
  files.forEach(f => {
    const item = document.createElement('div');
    item.className = 'file-item' + (f.id === activeID ? ' active' : '');
    item.onclick = () => openFile(f.id);
    const d = new Date(f.created * 1000);
    const ds = d.getFullYear() + '-' + String(d.getMonth()+1).padStart(2,'0') + '-' + String(d.getDate()).padStart(2,'0');
    let tagsHTML = f.tags && f.tags.length
      ? '<div class="file-tags">' + f.tags.map(t => '<span class="file-tag">'+esc(t)+'</span>').join('') + '</div>'
      : '';
    item.innerHTML =
      '<div class="file-name">' + esc(f.name) + '</div>' +
      '<div class="file-meta">' + esc(ds) + ' · ' + formatSize(f.size) + '</div>' +
      tagsHTML;
    list.appendChild(item);
  });
}

async function openFile(id) {
  activeID = id;
  const f = DATA.files.find(x => x.id === id);
  if (!f) return;
  renderSidebar();

  const d = new Date(f.created * 1000);
  const ds = d.getFullYear() + '-' + String(d.getMonth()+1).padStart(2,'0') + '-' + String(d.getDate()).padStart(2,'0');
  let tagsHTML = '';
  if ((f.tags && f.tags.length) || DATA.server_url) {
    const chips = (f.tags || []).map(t =>
      '<span class="content-tag" data-tag="'+esc(t)+'" data-fid="'+f.id+'">' +
        esc(t) +
        (DATA.server_url ? ' <span class="tag-rm" title="remove">×</span>' : '') +
      '</span>'
    ).join('');
    const addBtn = DATA.server_url
      ? '<span class="tag-add-btn" data-fid="'+f.id+'">+ tag</span>'
      : '';
    tagsHTML = '<div class="content-tags">' + chips + addBtn + '</div>';
  }
  let commentHTML = f.comment ? '<div class="content-comment">' + esc(f.comment) + '</div>' : '';

  let mediaHTML = '';

  if (f.mime.startsWith('image/')) {
    const dataURI = 'data:' + f.mime + ';base64,' + f.content;
    mediaHTML = '<img class="media" src="' + dataURI + '">';
  } else if (f.mime === 'application/pdf') {
    const dataURI = 'data:' + f.mime + ';base64,' + f.content;
    mediaHTML = '<embed class="media" src="' + dataURI + '" type="application/pdf">';
  } else if (f.mime.startsWith('video/') || f.mime.startsWith('audio/')) {
    const dataURI = 'data:' + f.mime + ';base64,' + f.content;
    const tag = f.mime.startsWith('video/') ? 'video' : 'audio';
    mediaHTML = '<' + tag + ' class="media" src="' + dataURI + '" controls></' + tag + '>';
  } else if (f.mime === 'text/html') {
    const raw = await decodeContent(f);
    mediaHTML = '<iframe class="media" srcdoc="' + raw.replace(/"/g, '&quot;') + '"></iframe>';
  } else if (f.mime === 'text/markdown') {
    const raw = await decodeContent(f);
    mediaHTML = typeof marked !== 'undefined'
      ? '<div class="md-body">' + marked.parse(raw) + '</div>'
      : '<pre class="media">' + esc(raw) + '</pre>';
  } else {
    mediaHTML = '<div class="media" style="display:flex;align-items:center;justify-content:center;height:200px;color:#888;font-size:14px;">Unsupported file type: ' + esc(f.mime) + '</div>';
  }

  document.getElementById('content-inner').innerHTML =
    '<div class="content-header">' +
      '<div class="content-title">' + esc(f.name) + '</div>' +
      '<div class="content-meta">' + esc(ds) + ' · ' + formatSize(f.size) + ' · ' + esc(f.mime) + '</div>' +
      commentHTML + tagsHTML +
    '</div>' + mediaHTML;
}

function filterFiles() {
  searchQuery = document.getElementById('search').value;
  renderSidebar();
}

function formatSize(n) {
  if (n < 1024) return n + ' B';
  if (n < 1048576) return (n/1024).toFixed(1) + ' KB';
  return (n/1048576).toFixed(1) + ' MB';
}

function esc(s) {
  return (s||'').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
}

// --- Tag editing (requires DATA.server_url) ---

async function addTag(fileId, tag) {
  tag = tag.trim();
  if (!tag) return;
  const f = DATA.files.find(x => x.id === fileId);
  if (!f || (f.tags || []).includes(tag)) return;
  f.tags = [...(f.tags || []), tag];
  if (activeID === fileId) openFile(fileId);
  renderSidebar();
  try {
    await fetch(DATA.server_url + '/api/addtag', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({filePath: f.path, tag})
    });
  } catch(e) {
    f.tags = f.tags.filter(t => t !== tag);
    if (activeID === fileId) openFile(fileId);
    console.error('addtag failed:', e);
  }
}

async function removeTag(fileId, tag) {
  const f = DATA.files.find(x => x.id === fileId);
  if (!f) return;
  const prev = f.tags || [];
  f.tags = prev.filter(t => t !== tag);
  if (activeID === fileId) openFile(fileId);
  renderSidebar();
  try {
    await fetch(DATA.server_url + '/api/removetag', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({filePath: f.path, tag})
    });
  } catch(e) {
    f.tags = prev;
    if (activeID === fileId) openFile(fileId);
    console.error('removetag failed:', e);
  }
}

function showTagInput(fileId) {
  const old = document.getElementById('tag-input-box');
  if (old) { old.remove(); return; }
  const box = document.createElement('div');
  box.id = 'tag-input-box';
  const opts = allTags().map(t => '<option value="'+esc(t)+'">').join('');
  box.innerHTML =
    '<datalist id="tag-dl">'+opts+'</datalist>' +
    '<input id="tag-input-field" type="text" placeholder="tag name…" list="tag-dl" autocomplete="off">' +
    '<span style="font-size:10px;color:#555">enter · esc to cancel</span>';
  const hdr = document.querySelector('.content-header');
  if (hdr) hdr.appendChild(box);
  const inp = document.getElementById('tag-input-field');
  inp.focus();
  inp.addEventListener('keydown', function(ev) {
    ev.stopPropagation();
    if (ev.key === 'Enter') { const v = inp.value.trim(); box.remove(); if (v) addTag(fileId, v); }
    else if (ev.key === 'Escape') { box.remove(); }
  });
}

// Event delegation for tag chip × and + tag button
document.getElementById('content').addEventListener('click', function(e) {
  const rm = e.target.closest('.tag-rm');
  if (rm) {
    const chip = rm.closest('.content-tag');
    if (chip) removeTag(parseInt(chip.dataset.fid), chip.dataset.tag);
    return;
  }
  const ab = e.target.closest('.tag-add-btn');
  if (ab) { showTagInput(parseInt(ab.dataset.fid)); }
});

document.addEventListener('keydown', function(e) {
  if (document.activeElement === document.getElementById('search')) return;
  if (document.activeElement && document.activeElement.id === 'tag-input-field') return;
  if (e.key === 't' && DATA.server_url && activeID !== null) {
    e.preventDefault();
    showTagInput(activeID);
    return;
  }
  let dir = 0;
  if (e.key === 'ArrowDown' || e.key === 'ArrowRight' || e.key === 'j') dir = 1;
  if (e.key === 'ArrowUp'   || e.key === 'ArrowLeft'  || e.key === 'k') dir = -1;
  if (!dir) return;
  e.preventDefault();
  const files = filteredFiles();
  if (!files.length) return;
  const idx = files.findIndex(f => f.id === activeID);
  const next = (idx + dir + files.length) % files.length;
  openFile(files[next].id);
  requestAnimationFrame(() => {
    const active = document.querySelector('.file-item.active');
    if (active) active.scrollIntoView({ block: 'nearest' });
  });
});

// --- Save a copy ---

function saveArtifact() {
  const title = (DATA && DATA.title) || document.title || 'artifact';
  const filename = title.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '') + '.html';
  const html = '<!DOCTYPE html>\n' + document.documentElement.outerHTML;
  const blob = new Blob([html], { type: 'text/html' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

// --- Encryption / boot ---

async function unlock() {
  const passphrase = document.getElementById('passphrase-input').value;
  const errEl = document.getElementById('lock-error');
  errEl.textContent = '';
  try {
    const salt = Uint8Array.from(atob(MANIFEST_RAW.salt), c => c.charCodeAt(0));
    const iv   = Uint8Array.from(atob(MANIFEST_RAW.iv),   c => c.charCodeAt(0));
    const ct   = Uint8Array.from(atob(MANIFEST_RAW.data), c => c.charCodeAt(0));
    const keyMaterial = await crypto.subtle.importKey(
      'raw', new TextEncoder().encode(passphrase), 'PBKDF2', false, ['deriveKey']);
    const key = await crypto.subtle.deriveKey(
      { name: 'PBKDF2', salt, iterations: 100000, hash: 'SHA-256' },
      keyMaterial, { name: 'AES-GCM', length: 256 }, false, ['decrypt']);
    const decrypted = await crypto.subtle.decrypt({ name: 'AES-GCM', iv }, key, ct);
    DATA = JSON.parse(new TextDecoder().decode(decrypted));
    document.getElementById('lock-screen').style.display = 'none';
    startViewer();
  } catch(e) {
    errEl.textContent = 'Wrong passphrase.';
    document.getElementById('passphrase-input').select();
  }
}

function startViewer() {
  renderSidebar();
  if (DATA.files.length > 0) openFile(DATA.files[0].id);
}

async function boot() {
  if (MANIFEST_RAW.encrypted === 'true') {
    document.getElementById('lock-screen').style.display = 'flex';
    document.getElementById('passphrase-input').focus();
  } else {
    DATA = MANIFEST_RAW;
    startViewer();
  }
}

boot();
</script>
{{RING_NAV}}
</body>
</html>
`
