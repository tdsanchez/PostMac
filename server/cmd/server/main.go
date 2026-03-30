// server — browser-native file server artifact builder
//
// Hard fork of mediatunnel (2026-03-24).
//
// Produces a single self-contained HTML file. When opened in any modern
// browser, the artifact prompts the user to pick specific files. It groups
// them by file extension and renders a navigable tag wordcloud.
//
// No backend. No install. No ports. The browser is the runtime.
//
// Filesystem bridge: File System Access API (showOpenFilePicker)
// File serving:      Blob URLs from FileSystemFileHandle.getFile()
// Phase 2 (future):  Service Worker intercepts /file/ via MessageChannel
//
// Usage:
//   server --output server.html
//   server --output server.html --title "Web Archive"
//   server --output server.html --serve 8080

package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	outputPath := flag.String("output", "server.html", "output HTML file")
	title      := flag.String("title", "Server", "page title")
	servePort  := flag.Int("serve", 0, "serve artifact via HTTP on this port (development)")
	wasmPath   := flag.String("wasm", "", "path to viewer.wasm for Publisher-format save (auto-resolved)")
	flag.Parse()

	wc2js := readWordcloud2JS()
	if wc2js == "" {
		fmt.Fprintln(os.Stderr, "error: wordcloud2.min.js not found in assets/")
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "wordcloud2.js: %s\n", formatSize(int64(len(wc2js))))

	wasmExecJS := readWasmExecJS()

	// Auto-resolve mdrender.wasm (C/md4c — replaces Go viewer.wasm)
	if *wasmPath == "" {
		candidate := filepath.Join(repoRoot(), "assets", "mdrender.wasm")
		if _, err := os.Stat(candidate); err == nil {
			*wasmPath = candidate
		}
	}

	// Load + compress mdrender.wasm for injection into saved Publisher artifacts
	var viewerWasmB64 string
	if *wasmPath != "" {
		wasmBytes, err := os.ReadFile(*wasmPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: cannot read mdrender.wasm: %v\n", err)
		} else {
			compressed, err := gzipCompress(wasmBytes)
			if err != nil {
				viewerWasmB64 = base64.StdEncoding.EncodeToString(wasmBytes)
			} else {
				viewerWasmB64 = base64.StdEncoding.EncodeToString(compressed)
				fmt.Fprintf(os.Stderr, "mdrender.wasm: %s → %s (%.0f%% reduction)\n",
					formatSize(int64(len(wasmBytes))),
					formatSize(int64(len(compressed))),
					100*(1-float64(len(compressed))/float64(len(wasmBytes))))
			}
		}
	}

	// Gzip+base64 encode wasm_exec.js for injection into saved Publisher artifacts
	// wasm_exec.js no longer needed — mdrender.wasm is C, not Go
	var wasmExecB64 string
	_ = wasmExecJS

	// Extract Publisher htmlTemplate from sibling source, gzip+base64 encode it
	publisherTmplB64 := loadPublisherTemplateBaked()

	// Load codec.wasm (C gzip codec), gzip+base64 for bootstrap delivery
	codecWasmB64 := loadCodecWasm()

	swActive   := "false"
	swRegister := ""
	if *servePort != 0 {
		swActive   = "true"
		swRegister = swRegisterSnippet
	}

	html := serverTemplate
	html = strings.Replace(html, "{{TITLE}}", *title, -1)
	html = strings.Replace(html, "{{WORDCLOUD2_JS}}", wc2js, 1)
	html = strings.Replace(html, "{{WASM_EXEC_JS}}", wasmExecJS, 1)
	html = strings.Replace(html, "{{VIEWER_WASM_B64}}", viewerWasmB64, 1)
	html = strings.Replace(html, "{{WASM_EXEC_B64}}", wasmExecB64, 1)
	html = strings.Replace(html, "{{PUBLISHER_TMPL_B64}}", publisherTmplB64, 1)
	html = strings.Replace(html, "{{CODEC_WASM_B64}}", codecWasmB64, 1)
	html = strings.Replace(html, "{{SW_ACTIVE}}", swActive, 1)
	html = strings.Replace(html, "{{SW_REGISTER}}", swRegister, 1)

	if *outputPath != "" {
		out, err := os.Create(*outputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error creating output: %v\n", err)
			os.Exit(1)
		}
		w := bufio.NewWriterSize(out, 4*1024*1024)
		fmt.Fprint(w, html)
		if err := w.Flush(); err != nil {
			out.Close()
			fmt.Fprintf(os.Stderr, "write error: %v\n", err)
			os.Exit(1)
		}
		out.Close()
		info, _ := os.Stat(*outputPath)
		fmt.Printf("server: %s (%.1f MB)\n", *outputPath, float64(info.Size())/1048576)
	}

	if *servePort != 0 {
		serveURL := fmt.Sprintf("http://localhost:%d", *servePort)
		fmt.Fprintf(os.Stderr, "serving: %s\n", serveURL)

		mux := http.NewServeMux()
		mux.HandleFunc("/sw.js", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/javascript")
			w.Header().Set("Service-Worker-Allowed", "/")
			fmt.Fprint(w, swScript)
		})
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, html)
		})

		go func() {
			time.Sleep(150 * time.Millisecond)
			exec.Command("open", "-a", "Waterfox", serveURL).Run()
		}()

		fmt.Fprintf(os.Stderr, "ctrl+c to stop\n")
		if err := http.ListenAndServe(fmt.Sprintf(":%d", *servePort), mux); err != nil {
			fmt.Fprintf(os.Stderr, "serve error: %v\n", err)
			os.Exit(1)
		}
	}
}

// swScript — minimal Service Worker for development mode.
// Phase 2 will add /file/ intercept via MessageChannel to page.
const swScript = `
self.addEventListener('install',  () => self.skipWaiting());
self.addEventListener('activate', e  => e.waitUntil(clients.claim()));
// Phase 2: intercept /file/ requests, broker to page via MessageChannel
`

const swRegisterSnippet = `
if ('serviceWorker' in navigator) {
  navigator.serviceWorker.register('/sw.js', { scope: '/' })
    .then(r  => console.log('Server SW:', r.scope))
    .catch(e => console.warn('Server SW failed:', e));
}
`

// ── Utilities (shared with mediatunnel lineage) ───────────────────────────────

func readWordcloud2JS() string {
	candidate := filepath.Join(repoRoot(), "assets", "wordcloud2.min.js")
	if data, err := os.ReadFile(candidate); err == nil {
		return string(data)
	}
	return ""
}

func readWasmExecJS() string {
	goroot, _ := exec.Command("go", "env", "GOROOT").Output()
	root := strings.TrimSpace(string(goroot))
	for _, p := range []string{
		filepath.Join(root, "misc", "wasm", "wasm_exec.js"),
		filepath.Join(root, "lib",  "wasm", "wasm_exec.js"),
	} {
		if data, err := os.ReadFile(p); err == nil {
			return string(data)
		}
	}
	return ""
}

// loadCodecWasm loads assets/codec.wasm, gzip-compresses it, and returns
// a base64 string for bootstrap delivery in the artifact.
func loadCodecWasm() string {
	p := filepath.Join(repoRoot(), "assets", "codec.wasm")
	data, err := os.ReadFile(p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: codec.wasm not found (%v) — C codec unavailable\n", err)
		return ""
	}
	gz, err := gzipCompress(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: compress codec.wasm: %v\n", err)
		return ""
	}
	fmt.Fprintf(os.Stderr, "codec.wasm: %s → %s (%.0f%% reduction)\n",
		formatSize(int64(len(data))),
		formatSize(int64(len(gz))),
		100*(1-float64(len(gz))/float64(len(data))))
	return base64.StdEncoding.EncodeToString(gz)
}

// loadPublisherTemplateBaked reads the Publisher htmlTemplate from
// cmd/publisher/main.go at build time, gzip-compresses it, and returns
// a base64 string for injection into the Server artifact.
// Publisher stays the source of truth — no duplication.
func loadPublisherTemplateBaked() string {
	src := filepath.Join(repoRoot(), "cmd", "publisher", "main.go")
	data, err := os.ReadFile(src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: cannot read publisher/main.go: %v — save will be unavailable\n", err)
		return ""
	}
	s := string(data)
	marker := "const htmlTemplate = `"
	start := strings.Index(s, marker)
	if start < 0 {
		fmt.Fprintln(os.Stderr, "warning: htmlTemplate not found in publisher/main.go — save will be unavailable")
		return ""
	}
	start += len(marker)
	rest := s[start:]
	// Template ends with </html>\n` — find last occurrence of newline+backtick
	end := strings.LastIndex(rest, "\n`")
	if end < 0 {
		fmt.Fprintln(os.Stderr, "warning: could not find end of htmlTemplate — save will be unavailable")
		return ""
	}
	tmpl := rest[:end+1]
	gz, err := gzipCompress([]byte(tmpl))
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: compress publisher template: %v\n", err)
		return ""
	}
	fmt.Fprintf(os.Stderr, "publisher template: %s → %s (%.0f%% reduction)\n",
		formatSize(int64(len(tmpl))),
		formatSize(int64(len(gz))),
		100*(1-float64(len(gz))/float64(len(tmpl))))
	return base64.StdEncoding.EncodeToString(gz)
}

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

// ── HTML Template ─────────────────────────────────────────────────────────────
// Placeholders: {{TITLE}} (×2), {{WORDCLOUD2_JS}}, {{WASM_EXEC_JS}},
//               {{VIEWER_WASM_B64}}, {{WASM_EXEC_B64}}, {{PUBLISHER_TMPL_B64}},
//               {{SW_ACTIVE}}, {{SW_REGISTER}}
const serverTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{TITLE}}</title>
<style>
* { box-sizing: border-box; margin: 0; padding: 0; }
html, body { height: 100%; }
body { background: #111; color: #ddd; font-family: Helvetica, Arial, sans-serif; font-size: 13px; display: flex; flex-direction: column; }

/* ── Landing ── */
#landing {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  flex: 1; gap: 20px; padding: 40px;
}
#landing h1 { font-size: 24px; font-weight: 400; color: #888; }
#landing p  { font-size: 13px; color: #444; text-align: center; max-width: 340px; line-height: 1.7; }
#open-btn {
  background: #1a2a1a; border: 1px solid #3a6a3a; color: #4ade80;
  padding: 10px 32px; border-radius: 5px; font-size: 14px; cursor: pointer;
}
#open-btn:hover { background: #243424; border-color: #4a8a4a; }
#open-hint { font-size: 11px; color: #333; text-align: center; max-width: 340px; line-height: 1.8; }

/* ── Passphrase input ── */
#passphrase {
  background: #1a1a1a; border: 1px solid #2a2a2a; color: #ccc;
  padding: 3px 8px; border-radius: 4px; font-size: 11px; width: 160px; outline: none; flex-shrink: 0;
}
#passphrase:focus { border-color: #555; }
#passphrase::placeholder { color: #333; }

/* ── Save button ── */
#save-btn {
  background: #1a1a1a; border: 1px solid #2a2a2a; color: #666;
  padding: 4px 10px; border-radius: 4px; font-size: 11px; cursor: pointer; flex-shrink: 0;
}
#save-btn:hover { border-color: #555; color: #aaa; }
#save-btn:disabled { color: #333; border-color: #222; cursor: default; }

/* ── Progress ── */
#progress {
  display: none; flex-direction: column; align-items: center; justify-content: center;
  flex: 1; gap: 12px;
}
#progress-label { color: #555; font-size: 13px; }
#progress-count { color: #4a9eff; font-size: 11px; }

/* ── Main UI ── */
#app { display: none; flex-direction: column; flex: 1; overflow: hidden; }
#header {
  padding: 8px 16px; color: #444; font-size: 11px;
  border-bottom: 1px solid #1a1a1a; display: flex; align-items: center; gap: 12px; flex-shrink: 0;
}
#header-title  { color: #666; font-size: 13px; flex: 1; }
#folder-name   { color: #4a9eff; }
#file-total    { color: #333; }
#search-bar    {
  padding: 6px 16px 7px; display: flex; align-items: center; gap: 8px;
  border-bottom: 1px solid #1a1a1a; flex-shrink: 0;
}
#search {
  background: #1a1a1a; border: 1px solid #2a2a2a; color: #ccc;
  padding: 4px 10px; border-radius: 3px; font-size: 12px; width: 220px; outline: none;
}
#search:focus { border-color: #555; }
#search-info  { color: #333; font-size: 11px; }
#layout { display: flex; flex: 1; overflow: hidden; }

/* ── Cloud panel ── */
#cloud-panel { flex: 3; overflow-y: auto; position: relative; min-width: 120px; }
#canvas { display: none; cursor: pointer; }

/* ── Drag divider ── */
#divider {
  width: 5px; background: #1a1a1a; cursor: col-resize; flex-shrink: 0;
  transition: background 0.1s;
}
#divider:hover, #divider.dragging { background: #3a6a3a; }

/* ── Detail panel ── */
#detail-panel {
  display: flex; flex-direction: column;
  overflow: hidden; min-width: 200px;
}
#detail-header  { padding: 10px 14px 8px; border-bottom: 1px solid #1a1a1a; flex-shrink: 0; }
#active-tag     { color: #4a9eff; font-size: 14px; font-weight: 500; }
#tag-count      { color: #333; font-size: 11px; margin-top: 2px; }

/* ── File list ── */
#file-list { flex: 1; overflow-y: auto; }
.fitem {
  padding: 7px 14px; border-bottom: 1px solid #161616; cursor: pointer;
  display: flex; flex-direction: column; gap: 2px;
}
.fitem:hover { background: #161616; }
.fitem.active { background: #1a2a1a; border-left: 2px solid #4ade80; }
.fitem-name { color: #bbb; font-size: 12px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.fitem-path { color: #333; font-size: 10px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }

/* ── Viewer ── */
#viewer-panel {
  border-top: 1px solid #1a1a1a; display: flex; flex-direction: column;
  flex-shrink: 0; height: 0; transition: height 0.15s ease; overflow: hidden;
}
#viewer-panel.open { height: 55%; }
#viewer-bar {
  padding: 6px 14px; border-bottom: 1px solid #1a1a1a; display: flex;
  align-items: center; gap: 8px; flex-shrink: 0;
}
#viewer-fname { color: #555; font-size: 10px; flex: 1; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
#open-tab-btn {
  background: #1a1a1a; border: 1px solid #2a2a2a; color: #666;
  padding: 3px 10px; border-radius: 3px; font-size: 11px; cursor: pointer; flex-shrink: 0;
}
#open-tab-btn:hover { border-color: #555; color: #aaa; }
#viewer-close {
  background: none; border: none; color: #333; font-size: 14px; cursor: pointer; flex-shrink: 0;
}
#viewer-close:hover { color: #888; }
#viewer-frame { flex: 1; border: none; background: #fff; }
</style>
</head>
<body>

<!-- Landing -->
<div id="landing">
  <h1>{{TITLE}}</h1>
  <p>Pick files to browse and explore. Select all you want — ctrl-A for everything in a folder. Groups by file type.</p>
  <button id="open-btn" onclick="openFiles()">Open Files</button>
  <input id="file-input" type="file" multiple style="display:none" onchange="onFileInput(this)">
  <p id="open-hint">
    Runs entirely in this tab.<br>
    No upload. No server. No install.
  </p>
</div>

<!-- Progress -->
<div id="progress">
  <div id="progress-label">Scanning&hellip;</div>
  <div id="progress-count"></div>
</div>

<!-- Main app -->
<div id="app">
  <div id="header">
    <span id="header-title">{{TITLE}}</span>
    <span id="folder-name"></span>
    <span id="file-total"></span>
    <input id="passphrase" type="password" placeholder="passphrase (optional)" title="Set a passphrase to encrypt the saved artifact">
    <button id="save-btn" onclick="saveAsPublisher()" title="Save all scanned files as a Publisher artifact">save</button>
  </div>
  <div id="search-bar">
    <input id="search" type="text" placeholder="search tags&hellip;" oninput="filterCloud()">
    <span id="search-info"></span>
  </div>
  <div id="layout">
    <div id="cloud-panel">
      <canvas id="canvas"></canvas>
    </div>
    <div id="divider"></div>
    <div id="detail-panel">
      <div id="detail-header">
        <div id="active-tag">select a tag</div>
        <div id="tag-count"></div>
      </div>
      <div id="file-list"></div>
      <div id="viewer-panel">
        <div id="viewer-bar">
          <span id="viewer-fname"></span>
          <button id="open-tab-btn" onclick="openInTab()">open in tab</button>
          <button id="viewer-close" onclick="closeViewer()" title="close">&times;</button>
        </div>
        <iframe id="viewer-frame" sandbox="allow-same-origin allow-scripts"></iframe>
      </div>
    </div>
  </div>
</div>

<script>
{{WORDCLOUD2_JS}}
</script>
<script>
{{WASM_EXEC_JS}}
</script>
<script>
const SW_ACTIVE = {{SW_ACTIVE}};

// ── State ─────────────────────────────────────────────────────────────────────
let dirHandle    = null;  // kept for directory-mode compat
let selectionName = 'selection';
let fileHandles  = new Map();   // path → FileSystemFileHandle
let blobCache    = new Map();   // path → blob URL
let activeBlob   = null;        // currently viewed blob URL
let INDEX        = null;        // {paths, tags, freqs}
let ALL_WORDS    = [];
let CURRENT_PATHS = [];
let CURRENT_POS   = -1;

// ── File access ───────────────────────────────────────────────────────────────
async function openFiles() {
  if (window.showOpenFilePicker) {
    let handles;
    try {
      handles = await window.showOpenFilePicker({ multiple: true });
    } catch(e) {
      if (e.name !== 'AbortError') console.error('showOpenFilePicker:', e);
      return;
    }
    if (!handles.length) return;
    showProgress();
    try {
      await buildIndexFromHandles(handles);
    } catch(e) {
      document.getElementById('progress-label').textContent = 'Error: ' + e.message;
      document.getElementById('progress-count').textContent = e.stack || '';
      console.error('buildIndex:', e);
    }
  } else {
    // Fallback: classic file input (Firefox, Safari)
    document.getElementById('file-input').click();
  }
}

async function onFileInput(input) {
  if (!input.files || !input.files.length) return;
  // Wrap File objects to match FileSystemFileHandle interface
  const handles = Array.from(input.files).map(function(f) {
    return { name: f.name, getFile: function() { return Promise.resolve(f); } };
  });
  showProgress();
  try {
    await buildIndexFromHandles(handles);
  } catch(e) {
    document.getElementById('progress-label').textContent = 'Error: ' + e.message;
    document.getElementById('progress-count').textContent = e.stack || '';
    console.error('buildIndex:', e);
  }
}

function showProgress() {
  document.getElementById('landing').style.display   = 'none';
  document.getElementById('progress').style.display  = 'flex';
}

// ── Index build from file handles ─────────────────────────────────────────────
async function buildIndexFromHandles(handles) {
  fileHandles.clear();
  blobCache.clear();

  const paths  = [];
  const tagMap = {};

  for (let i = 0; i < handles.length; i++) {
    const handle = handles[i];
    const name   = handle.name;
    const path   = name;

    // Tag = file extension (without dot), or 'other'
    const dot = name.lastIndexOf('.');
    const tag = dot >= 0 ? name.substring(dot + 1).toLowerCase() : 'other';

    const idx = paths.length;
    paths.push(path);
    fileHandles.set(path, handle);
    if (!tagMap[tag]) tagMap[tag] = [];
    tagMap[tag].push(idx);

    if ((i + 1) % 50 === 0) {
      document.getElementById('progress-count').textContent =
        (i + 1).toLocaleString() + ' / ' + handles.length.toLocaleString() + ' files';
      await new Promise(r => setTimeout(r, 0));
    }
  }

  if (paths.length === 0) {
    document.getElementById('progress-label').textContent = 'No files selected.';
    return;
  }

  // Virtual ALL tag — every file
  tagMap['ALL'] = paths.map(function(_, i) { return i; });

  const freqs = Object.entries(tagMap)
    .filter(function(e) { return e[0] !== 'ALL'; })
    .sort(function(a, b) { return b[1].length - a[1].length; })
    .map(function(e)     { return [e[0], e[1].length]; });

  // Prepend ALL at a fixed display weight so it doesn't dwarf everything else
  const allWeight = Math.max.apply(null, freqs.map(function(f){return f[1];})) || 1;
  freqs.unshift(['ALL', allWeight]);

  INDEX = { paths, tags: tagMap, freqs };
  ALL_WORDS = freqs;

  selectionName = handles.length === 1
    ? handles[0].name
    : handles.length + ' files';

  document.getElementById('progress').style.display  = 'none';
  document.getElementById('app').style.display       = 'flex';
  document.getElementById('folder-name').textContent = selectionName;
  document.getElementById('file-total').textContent  = paths.length.toLocaleString() + ' files';

  drawCloud(ALL_WORDS);
}

// ── Wordcloud ─────────────────────────────────────────────────────────────────
var canvas = document.getElementById('canvas');

function drawCloud(words) {
  if (!words.length) return;

  const panel = document.getElementById('cloud-panel');
  canvas.style.display = 'block';
  canvas.width  = panel.clientWidth  || 600;
  canvas.height = Math.max(400, Math.floor((panel.clientHeight || 600) * 0.9));

  const ctx = canvas.getContext('2d');
  ctx.fillStyle = '#111';
  ctx.fillRect(0, 0, canvas.width, canvas.height);

  const maxFreq = Math.max.apply(null, words.map(function(w) { return w[1]; }));

  WordCloud(canvas, {
    list: words,
    gridSize: 6,
    weightFactor: function(size) {
      var logScale = maxFreq > 1 ? Math.log(size + 1) / Math.log(maxFreq + 1) : 1;
      return Math.max(12, Math.min(logScale * canvas.width / 5, 80));
    },
    fontFamily: 'Helvetica, Arial, sans-serif',
    color: function(word) {
      if (word === 'ALL') return '#4ade80';
      var blues = ['#4a9eff','#7ecfff','#88bbff','#5ab4ff','#3388ee','#66aaff'];
      return blues[Math.floor(Math.random() * blues.length)];
    },
    backgroundColor: '#111',
    rotateRatio: 0.1,
    minSize: 10,
    drawOutOfBound: false,
    shuffle: true,
    click: function(item) { selectTag(item[0]); },
    hover: function(item) { canvas.style.cursor = item ? 'pointer' : 'default'; }
  });
}

function filterCloud() {
  const q = document.getElementById('search').value.toLowerCase().trim();
  if (!ALL_WORDS.length) return;
  const filtered = q
    ? ALL_WORDS.filter(function(w) { return w[0].toLowerCase().includes(q); })
    : ALL_WORDS;
  document.getElementById('search-info').textContent =
    q ? filtered.length + ' match' + (filtered.length === 1 ? '' : 'es') : '';
  drawCloud(filtered);
}

// ── Tag selection + file list ─────────────────────────────────────────────────
function selectTag(tag) {
  if (!INDEX) return;
  const indices = INDEX.tags[tag] || [];
  CURRENT_PATHS = indices.map(function(i) { return INDEX.paths[i]; });
  CURRENT_POS   = CURRENT_PATHS.length ? 0 : -1;

  document.getElementById('active-tag').textContent = tag;
  document.getElementById('tag-count').textContent  =
    CURRENT_PATHS.length.toLocaleString() + ' file' + (CURRENT_PATHS.length === 1 ? '' : 's');

  renderFileList(CURRENT_PATHS);
}

function renderFileList(paths) {
  const list = document.getElementById('file-list');
  list.innerHTML = '';
  paths.forEach(function(p, i) {
    const div  = document.createElement('div');
    div.className = 'fitem' + (i === CURRENT_POS ? ' active' : '');
    const name = p.split('/').pop();
    const dir  = p.includes('/') ? p.substring(0, p.lastIndexOf('/')) : '';
    div.innerHTML =
      '<div class="fitem-name">' + esc(name) + '</div>' +
      (dir ? '<div class="fitem-path">' + esc(dir) + '</div>' : '');
    div.onclick = function() { openFile(p, i); };
    list.appendChild(div);
  });
}

// ── File viewer ───────────────────────────────────────────────────────────────
async function openFile(path, listIndex) {
  CURRENT_POS = listIndex;
  // Update active state in list
  document.querySelectorAll('.fitem').forEach(function(el, i) {
    el.classList.toggle('active', i === listIndex);
  });

  const handle = fileHandles.get(path);
  if (!handle) return;

  // Reuse cached blob URL or create new one
  let url = blobCache.get(path);
  if (!url) {
    const file = await handle.getFile();
    url = URL.createObjectURL(file);
    blobCache.set(path, url);
  }

  // Show viewer panel
  const panel = document.getElementById('viewer-panel');
  panel.classList.add('open');
  document.getElementById('viewer-fname').textContent = path;
  document.getElementById('viewer-frame').src = url;
  activeBlob = url;
}

async function openInTab() {
  if (!activeBlob) return;
  window.open(activeBlob, '_blank');
}

function closeViewer() {
  const panel = document.getElementById('viewer-panel');
  panel.classList.remove('open');
  document.getElementById('viewer-frame').src = '';
  activeBlob = null;
  document.querySelectorAll('.fitem').forEach(function(el) {
    el.classList.remove('active');
  });
}

// ── Keyboard navigation ───────────────────────────────────────────────────────
document.addEventListener('keydown', function(e) {
  if (e.target.tagName === 'INPUT') return;
  if (!CURRENT_PATHS.length) return;
  let dir = 0;
  if (e.key === 'ArrowDown'  || e.key === 'j') dir =  1;
  if (e.key === 'ArrowUp'    || e.key === 'k') dir = -1;
  if (!dir) return;
  e.preventDefault();
  const next = Math.max(0, Math.min(CURRENT_PATHS.length - 1, CURRENT_POS + dir));
  openFile(CURRENT_PATHS[next], next);
  const items = document.querySelectorAll('.fitem');
  if (items[next]) items[next].scrollIntoView({ block: 'nearest' });
});

// ── Utilities ─────────────────────────────────────────────────────────────────
function esc(s) {
  return (s || '').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
}

// ── C Codec (gzip via zlib, compiled to WASM) ─────────────────────────────────
const CODEC_WASM_B64 = "{{CODEC_WASM_B64}}";
let CODEC = null; // set by initCodec() on boot

async function initCodec() {
  if (!CODEC_WASM_B64) return;
  try {
    // Bootstrap: decompress codec.wasm via browser-native DecompressionStream
    const raw   = atob(CODEC_WASM_B64);
    const bytes = new Uint8Array(raw.length);
    for (let i = 0; i < raw.length; i++) bytes[i] = raw.charCodeAt(i);
    const wasmBytes = await browserDecompress(bytes);

    // Instantiate — only import needed is emscripten_resize_heap
    const {instance} = await WebAssembly.instantiate(wasmBytes, {
      env: {
        emscripten_resize_heap: function(requestedSize) {
          const mem = instance.exports.memory;
          const needed = Math.ceil(requestedSize / 65536);
          const current = mem.buffer.byteLength / 65536;
          if (needed <= current) return 1;
          try { mem.grow(needed - current); return 1; } catch(e) { return 0; }
        }
      }
    });

    const ex = instance.exports;

    function heapSet(data) {
      const ptr = ex.malloc(data.length);
      new Uint8Array(ex.memory.buffer, ptr, data.length).set(data);
      return ptr;
    }
    function heapGetU32(ptr) {
      return new DataView(ex.memory.buffer).getUint32(ptr, /*little-endian*/true);
    }

    CODEC = {
      compress: function(data) {
        const srcPtr    = heapSet(data);
        const outLenPtr = ex.malloc(4);
        const dstPtr    = ex.compress_buf(srcPtr, data.length, outLenPtr);
        ex.free(srcPtr);
        if (!dstPtr) { ex.free(outLenPtr); throw new Error('codec compress failed'); }
        const outLen = heapGetU32(outLenPtr);
        ex.free(outLenPtr);
        const result = new Uint8Array(ex.memory.buffer, dstPtr, outLen).slice();
        ex.free_buf(dstPtr);
        return result;
      },
      decompress: function(data) {
        // Read ISIZE from gzip trailer (last 4 bytes, little-endian)
        const dv    = new DataView(data.buffer, data.byteOffset, data.byteLength);
        const isize = dv.getUint32(data.byteLength - 4, true);
        const maxOut = Math.max(isize, 64) + 64;
        const srcPtr    = heapSet(data);
        const outLenPtr = ex.malloc(4);
        const dstPtr    = ex.decompress_buf(srcPtr, data.length, maxOut, outLenPtr);
        ex.free(srcPtr);
        if (!dstPtr) { ex.free(outLenPtr); throw new Error('codec decompress failed'); }
        const outLen = heapGetU32(outLenPtr);
        ex.free(outLenPtr);
        const result = new Uint8Array(ex.memory.buffer, dstPtr, outLen).slice();
        ex.free_buf(dstPtr);
        return result;
      }
    };
    console.log('codec.wasm ready');
  } catch(e) {
    console.warn('codec.wasm failed to load, using browser fallbacks:', e);
  }
}

// Raw decompression (used by initCodec bootstrap — no CODEC dependency)
async function browserDecompress(bytes) {
  const ds = new DecompressionStream('gzip');
  const writer = ds.writable.getWriter();
  writer.write(bytes);
  writer.close();
  const chunks = [];
  const reader = ds.readable.getReader();
  while (true) {
    const {done, value} = await reader.read();
    if (done) break;
    chunks.push(value);
  }
  const out = new Uint8Array(chunks.reduce(function(s,c){return s+c.length;},0));
  let off = 0; for (const c of chunks) { out.set(c, off); off += c.length; }
  return out;
}

// ── Publisher save ────────────────────────────────────────────────────────────
const VIEWER_WASM_B64     = "{{VIEWER_WASM_B64}}";
const WASM_EXEC_B64       = "{{WASM_EXEC_B64}}";
const PUBLISHER_TMPL_B64  = "{{PUBLISHER_TMPL_B64}}";

function mimeFor(path) {
  const ext = path.split('.').pop().toLowerCase();
  const map = {
    jpg:'image/jpeg', jpeg:'image/jpeg', png:'image/png', gif:'image/gif',
    webp:'image/webp', tif:'image/tiff', tiff:'image/tiff',
    pdf:'application/pdf', mp4:'video/mp4', mov:'video/quicktime',
    webm:'video/webm', mp3:'audio/mpeg', m4a:'audio/mp4',
    md:'text/markdown', html:'text/html', htm:'text/html', txt:'text/plain',
    svg:'image/svg+xml'
  };
  return map[ext] || 'application/octet-stream';
}

function shouldCompress(mime) {
  return mime.startsWith('text/') || mime === 'application/json' || mime === 'image/svg+xml';
}

async function gzipCompressBytes(data) {
  const bytes = data instanceof Uint8Array ? data : new TextEncoder().encode(data);
  if (CODEC) return CODEC.compress(bytes);
  // Fallback: browser CompressionStream
  const cs = new CompressionStream('gzip');
  const writer = cs.writable.getWriter();
  writer.write(bytes);
  writer.close();
  const chunks = [];
  const reader = cs.readable.getReader();
  while (true) {
    const {done, value} = await reader.read();
    if (done) break;
    chunks.push(value);
  }
  const out = new Uint8Array(chunks.reduce(function(s,c){return s+c.length;}, 0));
  let off = 0;
  for (const c of chunks) { out.set(c, off); off += c.length; }
  return out;
}

function bytesToBase64(bytes) {
  let s = '';
  for (let i = 0; i < bytes.length; i++) s += String.fromCharCode(bytes[i]);
  return btoa(s);
}

async function gzipDecompress(b64) {
  const raw = atob(b64);
  const bytes = new Uint8Array(raw.length);
  for (let i = 0; i < raw.length; i++) bytes[i] = raw.charCodeAt(i);
  if (CODEC) return CODEC.decompress(bytes);
  return browserDecompress(bytes);
}

async function encryptManifest(jsonStr, passphrase) {
  const salt = crypto.getRandomValues(new Uint8Array(16));
  const iv   = crypto.getRandomValues(new Uint8Array(12));
  const keyMaterial = await crypto.subtle.importKey(
    'raw', new TextEncoder().encode(passphrase), 'PBKDF2', false, ['deriveKey']);
  const key = await crypto.subtle.deriveKey(
    { name: 'PBKDF2', salt, iterations: 100000, hash: 'SHA-256' },
    keyMaterial, { name: 'AES-GCM', length: 256 }, false, ['encrypt']);
  const ct = await crypto.subtle.encrypt(
    { name: 'AES-GCM', iv }, key, new TextEncoder().encode(jsonStr));
  return JSON.stringify({
    encrypted: 'true',
    salt: btoa(String.fromCharCode(...salt)),
    iv:   btoa(String.fromCharCode(...iv)),
    data: bytesToBase64(new Uint8Array(ct))
  });
}

async function saveAsPublisher() {
  if (!PUBLISHER_TMPL_B64) {
    alert('Publisher template not available — rebuild Server binary with viewer.wasm present.');
    return;
  }
  if (!fileHandles.size) {
    alert('No files scanned yet — open a folder first.');
    return;
  }

  const btn = document.getElementById('save-btn');
  btn.disabled = true;
  btn.textContent = 'building\u2026';

  try {
    // Build PATH_TAGS reverse map
    const pathTagMap = {};
    if (INDEX) {
      for (const [tag, indices] of Object.entries(INDEX.tags)) {
        for (const idx of indices) {
          const p = INDEX.paths[idx];
          if (!pathTagMap[p]) pathTagMap[p] = [];
          pathTagMap[p].push(tag);
        }
      }
    }

    // Read + encode all files
    const files = [];
    let id = 1;
    let done = 0;
    const total = fileHandles.size;
    for (const [path, handle] of fileHandles) {
      const file  = await handle.getFile();
      const bytes = new Uint8Array(await file.arrayBuffer());
      const mime  = mimeFor(path);
      let content, compressed;
      if (shouldCompress(mime)) {
        const gz = await gzipCompressBytes(bytes);
        content    = bytesToBase64(gz);
        compressed = true;
      } else {
        content    = bytesToBase64(bytes);
        compressed = false;
      }
      files.push({
        id:         id++,
        name:       path.split('/').pop(),
        path:       path,
        mime:       mime,
        tags:       pathTagMap[path] || [],
        comment:    '',
        created:    Math.floor(file.lastModified / 1000),
        size:       file.size,
        content:    content,
        compressed: compressed
      });
      done++;
      if (done % 20 === 0) {
        btn.textContent = 'building\u2026 ' + done + '/' + total;
        await new Promise(r => setTimeout(r, 0)); // yield
      }
    }

    const manifest = {
      title:        selectionName || 'Server Export',
      published_at: Math.floor(Date.now() / 1000),
      server_url:   '',
      paths:        INDEX ? INDEX.paths : [],
      files:        files
    };

    btn.textContent = 'compressing\u2026';
    const passphrase = document.getElementById('passphrase').value.trim();
    let manifestJSON = JSON.stringify(manifest);
    if (passphrase) {
      btn.textContent = 'encrypting\u2026';
      manifestJSON = await encryptManifest(manifestJSON, passphrase);
    }

    // Decompress Publisher template + wasm_exec
    const tmplBytes    = await gzipDecompress(PUBLISHER_TMPL_B64);
    const tmplText     = new TextDecoder().decode(tmplBytes);
    const wasmExecText = WASM_EXEC_B64
      ? new TextDecoder().decode(await gzipDecompress(WASM_EXEC_B64))
      : '';

    // String-replace placeholders (same as Go publisher does)
    const title = manifest.title;
    let html = tmplText;
    html = html.split('{{TITLE}}').join(title);
    html = html.replace('{{DATA}}',         manifestJSON);
    html = html.replace('{{WASM_B64}}',     VIEWER_WASM_B64);
    html = html.replace('{{WASM_EXEC_JS}}', wasmExecText);
    html = html.replace('{{CODEC_B64}}',    CODEC_WASM_B64);
    html = html.replace('{{RING_NAV}}',     '');

    // Download
    const blob     = new Blob([html], {type: 'text/html'});
    const url      = URL.createObjectURL(blob);
    const a        = document.createElement('a');
    a.href         = url;
    a.download     = title.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g,'') + '.html';
    a.click();
    URL.revokeObjectURL(url);

    btn.textContent = 'saved';
    setTimeout(function() { btn.textContent = 'save'; btn.disabled = false; }, 2000);
  } catch(e) {
    console.error('saveAsPublisher:', e);
    btn.textContent = 'error — see console';
    btn.disabled = false;
  }
}

// ── Draggable divider ─────────────────────────────────────────────────────────
(function() {
  var divider    = document.getElementById('divider');
  var cloudPanel = document.getElementById('cloud-panel');
  var layout     = document.getElementById('layout');
  var dragging   = false;
  var startX     = 0;
  var startW     = 0;

  divider.addEventListener('mousedown', function(e) {
    // First drag: lock current flex-computed width to px, then take over
    if (cloudPanel.style.flex !== 'none') {
      cloudPanel.style.width = cloudPanel.clientWidth + 'px';
      cloudPanel.style.flex  = 'none';
    }
    dragging = true;
    startX   = e.clientX;
    startW   = cloudPanel.clientWidth;
    divider.classList.add('dragging');
    document.body.style.cursor     = 'col-resize';
    document.body.style.userSelect = 'none';
    e.preventDefault();
  });

  document.addEventListener('mousemove', function(e) {
    if (!dragging) return;
    var delta = e.clientX - startX;
    var newW  = Math.max(120, Math.min(startW + delta, layout.clientWidth - 200 - 5));
    cloudPanel.style.width = newW + 'px';
  });

  document.addEventListener('mouseup', function() {
    if (!dragging) return;
    dragging = false;
    divider.classList.remove('dragging');
    document.body.style.cursor     = '';
    document.body.style.userSelect = '';
    if (typeof drawCloud === 'function' && ALL_WORDS.length) drawCloud(ALL_WORDS);
  });
})();

// ── Boot ──────────────────────────────────────────────────────────────────────
initCodec(); // load C codec in background; compress/decompress fall back gracefully

// ── SW registration (development mode) ───────────────────────────────────────
{{SW_REGISTER}}
</script>
</body>
</html>`
