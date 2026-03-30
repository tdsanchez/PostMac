// corpus — self-contained offline-first corpus navigator
//
// Queries a live media-server's API to build a fully self-contained HTML artifact:
// a wordcloud of human tags, click-to-gallery, per-file tag display, and tag
// writeback to the live server when available.
//
// Replicates the core navigation + tagging workflow of media-server (8898)
// as a portable file. No server required for browsing. Tag writes go to
// --server; read-only offline otherwise.
//
// NOTE: Does NOT read cache.db. The media-server-arm64 instances share a single
// ~/.media-server-conf/cache.db with no port isolation — it reflects whichever
// instance last ran. Query the live API instead.
//
// Usage:
//   corpus --server http://localhost:8898 --output corpus.html
//   corpus --server http://localhost:8898 --output corpus.html --title "VFP Corpus"
//
// Data embedded:
//   { "paths": [...], "tags": { "tagname": [idx,...] }, "freqs": [[tag,count],...] }

package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type CorpusManifest struct {
	Paths []string           `json:"paths"`
	Tags  map[string][]int   `json:"tags"`  // tag → path indices
	Freqs [][2]interface{}   `json:"freqs"` // [[tag, count], ...] sorted desc
}

func main() {
	outputPath := flag.String("output", "corpus.html", "output HTML file")
	title      := flag.String("title", "Corpus Navigator", "page title")
	serverURL  := flag.String("server", "", "media-server base URL (required, e.g. http://localhost:8898)")
	flag.Parse()

	if *serverURL == "" {
		fmt.Fprintln(os.Stderr, "error: --server is required (e.g. --server http://localhost:8898)")
		os.Exit(1)
	}

	base := strings.TrimRight(*serverURL, "/")
	fmt.Fprintf(os.Stderr, "server: %s\n", base)

	// Build path list and tag index from live server API
	manifest, err := buildManifestFromAPI(base)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error building manifest: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "paths: %d, tags: %d\n", len(manifest.Paths), len(manifest.Tags))

	// Serialize + compress
	jsonBytes, err := json.Marshal(manifest)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshaling: %v\n", err)
		os.Exit(1)
	}
	compressed, err := gzipCompress(jsonBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error compressing: %v\n", err)
		os.Exit(1)
	}
	dataB64 := base64.StdEncoding.EncodeToString(compressed)
	fmt.Fprintf(os.Stderr, "manifest: %s → %s (%.0f%% reduction)\n",
		formatSize(int64(len(jsonBytes))),
		formatSize(int64(len(compressed))),
		100*(1-float64(len(compressed))/float64(len(jsonBytes))))

	// Load wordcloud2.js
	wc2js := readWordcloud2JS()
	if wc2js == "" {
		fmt.Fprintln(os.Stderr, "error: wordcloud2.min.js not found in assets/")
		os.Exit(1)
	}

	out, err := os.Create(*outputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating output: %v\n", err)
		os.Exit(1)
	}
	defer out.Close()

	w := bufio.NewWriterSize(out, 8*1024*1024)
	html := corpusTemplate
	html = strings.Replace(html, "{{TITLE}}", *title, 2)
	html = strings.Replace(html, "{{TAG_COUNT}}", fmt.Sprintf("%d", len(manifest.Tags)), 1)
	html = strings.Replace(html, "{{PATH_COUNT}}", fmt.Sprintf("%d", len(manifest.Paths)), 1)
	html = strings.Replace(html, "{{DATA_B64}}", dataB64, 1)
	html = strings.Replace(html, "{{SERVER_URL}}", *serverURL, 1)
	html = strings.Replace(html, "{{WORDCLOUD2_JS}}", wc2js, 1)
	fmt.Fprint(w, html)
	if err := w.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "write error: %v\n", err)
		os.Exit(1)
	}

	info, _ := os.Stat(*outputPath)
	fmt.Printf("corpus: %s (%d tags, %d files, %.1f MB)\n",
		*outputPath, len(manifest.Tags), len(manifest.Paths), float64(info.Size())/1048576)
}

// buildManifestFromAPI fetches tag→paths from a live media-server using its API.
// Uses /api/alltags to get tag list, then /api/filelist?category=<tag> for each tag.
// Concurrent fetches (up to 20 at a time) to keep total time under a few seconds.
func buildManifestFromAPI(base string) (*CorpusManifest, error) {
	// 1. Fetch tag list
	resp, err := http.Get(base + "/api/alltags")
	if err != nil {
		return nil, fmt.Errorf("GET /api/alltags: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("read /api/alltags: %w", err)
	}
	var tagNames []string
	if err := json.Unmarshal(body, &tagNames); err != nil {
		return nil, fmt.Errorf("parse /api/alltags: %w", err)
	}
	fmt.Fprintf(os.Stderr, "tags: %d\n", len(tagNames))

	// 2. Fetch file paths for each tag concurrently (semaphore: 20 concurrent)
	type tagResult struct {
		tag   string
		paths []string
		err   error
	}
	results := make([]tagResult, len(tagNames))
	sem := make(chan struct{}, 20)
	var wg sync.WaitGroup

	for i, tag := range tagNames {
		wg.Add(1)
		go func(i int, tag string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			u := base + "/api/filelist?category=" + url.QueryEscape(tag)
			r, err := http.Get(u)
			if err != nil {
				results[i] = tagResult{tag: tag, err: err}
				return
			}
			b, err := io.ReadAll(r.Body)
			r.Body.Close()
			if err != nil {
				results[i] = tagResult{tag: tag, err: err}
				return
			}
			var paths []string
			if err := json.Unmarshal(b, &paths); err != nil {
				results[i] = tagResult{tag: tag, err: fmt.Errorf("parse filelist for %q: %w", tag, err)}
				return
			}
			results[i] = tagResult{tag: tag, paths: paths}
		}(i, tag)
	}
	wg.Wait()

	// 3. Build manifest
	pathIndex := map[string]int{}
	paths     := []string{}
	tagMap    := map[string][]int{}

	type tagEntry struct {
		name  string
		count int
	}
	var entries []tagEntry

	for _, res := range results {
		if res.err != nil {
			fmt.Fprintf(os.Stderr, "warning: %v\n", res.err)
			continue
		}
		if len(res.paths) == 0 {
			continue
		}
		indices := make([]int, 0, len(res.paths))
		for _, p := range res.paths {
			idx, ok := pathIndex[p]
			if !ok {
				idx = len(paths)
				pathIndex[p] = idx
				paths = append(paths, p)
			}
			indices = append(indices, idx)
		}
		tagMap[res.tag] = indices
		entries = append(entries, tagEntry{res.tag, len(indices)})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].count > entries[j].count
	})

	freqs := make([][2]interface{}, len(entries))
	for i, e := range entries {
		freqs[i] = [2]interface{}{e.name, e.count}
	}

	return &CorpusManifest{
		Paths: paths,
		Tags:  tagMap,
		Freqs: freqs,
	}, nil
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

func readWordcloud2JS() string {
	candidate := filepath.Join(repoRoot(), "assets", "wordcloud2.min.js")
	if data, err := os.ReadFile(candidate); err == nil {
		return string(data)
	}
	return ""
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

// corpusTemplate — self-contained corpus navigator artifact.
// Placeholders: {{TITLE}} (×2), {{TAG_COUNT}}, {{PATH_COUNT}},
//               {{DATA_B64}}, {{SERVER_URL}}, {{WORDCLOUD2_JS}}
const corpusTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>{{TITLE}}</title>
<style>
* { box-sizing: border-box; margin: 0; padding: 0; }
body { background: #111; color: #ddd; font-family: Helvetica, Arial, sans-serif; font-size: 13px; display: flex; flex-direction: column; height: 100vh; }
#header { padding: 10px 20px 6px; color: #555; font-size: 12px; flex-shrink: 0; }
#search-bar {
  padding: 6px 20px 8px; display: flex; align-items: center; gap: 8px;
  border-bottom: 1px solid #1e1e1e; flex-shrink: 0;
}
#search { background: #1a1a1a; border: 1px solid #2e2e2e; color: #ccc;
  padding: 4px 10px; border-radius: 3px; font-size: 12px; width: 220px; outline: none; }
#search:focus { border-color: #555; }
#search-results { color: #444; font-size: 11px; }
#layout { display: flex; flex: 1; overflow: hidden; }
#cloud-panel { flex: 3; overflow-y: auto; position: relative; }
#loading { color: #555; font-size: 13px; padding: 40px 20px; text-align: center; }
#canvas { display: none; margin: 0 auto; cursor: pointer; }
#detail-panel {
  flex: 2; border-left: 1px solid #1e1e1e; display: flex; flex-direction: column;
  overflow: hidden; min-width: 300px;
}
#detail-header { padding: 10px 14px; border-bottom: 1px solid #1e1e1e; flex-shrink: 0; }
#active-tag { color: #4a9eff; font-size: 14px; font-weight: 500; }
#tag-count { color: #444; font-size: 11px; margin-top: 2px; }
#tag-actions { margin-top: 8px; display: flex; gap: 6px; }
#tag-btn {
  display: none; background: #1a1a1a; border: 1px solid #2e2e2e; color: #666;
  padding: 3px 10px; border-radius: 3px; font-size: 11px; cursor: pointer;
}
#viewer { flex: 1; display: flex; align-items: center; justify-content: center;
  background: #0a0a0a; overflow: hidden; position: relative; }
#viewer img { max-width: 100%; max-height: 100%; object-fit: contain; display: block;
  transition: opacity 0.12s ease; }
#viewer-empty { color: #2a2a2a; font-size: 13px; }
#nav { display: flex; align-items: center; justify-content: center; gap: 12px;
  padding: 6px 14px; border-top: 1px solid #1a1a1a; flex-shrink: 0; }
#nav button { background: #1a1a1a; border: 1px solid #2e2e2e; color: #666;
  padding: 3px 10px; border-radius: 3px; font-size: 14px; cursor: pointer; }
#nav button:hover { color: #aaa; border-color: #555; }
#nav-pos { color: #444; font-size: 11px; min-width: 80px; text-align: center; }
#file-info { padding: 8px 14px 10px; border-top: 1px solid #1a1a1a; flex-shrink: 0; }
#viewer-fname { color: #555; font-size: 10px; margin-bottom: 6px;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
#viewer-tags { display: flex; gap: 4px; flex-wrap: wrap; margin-bottom: 6px; min-height: 20px; }
.ftag { background: #1a2a1a; color: #4a9eff; font-size: 11px; padding: 2px 7px;
  border-radius: 10px; cursor: pointer; }
.ftag:hover { background: #2a3a2a; }
#add-tag-row { display: flex; gap: 6px; }
#add-tag-input { background: #1a1a1a; border: 1px solid #2e2e2e; color: #ccc;
  padding: 4px 8px; border-radius: 3px; font-size: 11px; flex: 1; outline: none; }
#add-tag-input:focus { border-color: #4a7a4a; }
#add-tag-btn { background: #1a2a1a; border: 1px solid #3a5a3a; color: #4ade80;
  padding: 4px 10px; border-radius: 3px; font-size: 11px; cursor: pointer; }
</style>
</head>
<body>
<div id="header">
  {{TITLE}} &mdash; {{TAG_COUNT}} tags &mdash; {{PATH_COUNT}} tagged files &mdash; click any tag to browse
</div>
<div id="search-bar">
  <input id="search" type="text" placeholder="search tags&hellip;" oninput="filterCloud()">
  <span id="search-results"></span>
</div>
<div id="layout">
  <div id="cloud-panel">
    <div id="loading">decompressing&hellip;</div>
    <canvas id="canvas"></canvas>
  </div>
  <div id="detail-panel">
    <div id="detail-header">
      <div id="active-tag">select a tag</div>
      <div id="tag-count"></div>
      <div id="tag-actions">
        <button id="tag-btn" onclick="openInViewer()">open in viewer</button>
      </div>
    </div>
    <div id="viewer">
      <span id="viewer-empty">select a tag</span>
      <img id="viewer-img" src="" style="display:none">
    </div>
    <div id="nav">
      <button onclick="navigate(-1)">&#8592;</button>
      <span id="nav-pos"></span>
      <button onclick="navigate(1)">&#8594;</button>
    </div>
    <div id="file-info">
      <div id="viewer-fname"></div>
      <div id="viewer-tags"></div>
      <div id="add-tag-row">
        <input id="add-tag-input" type="text" placeholder="add tag&hellip;">
        <button id="add-tag-btn" onclick="addTagToFile()">add</button>
      </div>
    </div>
  </div>
</div>

<script>
{{WORDCLOUD2_JS}}
</script>
<script>
const DATA_B64   = "{{DATA_B64}}";
const SERVER_URL = "{{SERVER_URL}}";
var PORT = SERVER_URL ? SERVER_URL.replace(/\/$/, '') : null;

var INDEX          = null;
var PATH_INDEX     = {};   // path → array index (built at boot)
var PATH_TAGS      = {};   // array index → [tag, ...] (built at boot)
var ALL_WORDS      = [];
var FILTERED       = [];
var CURRENT_INDICES = [];  // path indices for selected tag
var CURRENT_POS    = 0;    // position within CURRENT_INDICES

// ── Decompress ────────────────────────────────────────────────────────────────
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

// ── Tag cloud ─────────────────────────────────────────────────────────────────
var canvas = document.getElementById('canvas');

function drawCloud(words) {
  var maxFreq = words.length ? Math.max.apply(null, words.map(function(w) { return w[1]; })) : 1;
  var ctx = canvas.getContext('2d');
  ctx.fillStyle = '#111';
  ctx.fillRect(0, 0, canvas.width, canvas.height);

  if (!words.length) {
    ctx.fillStyle = '#333'; ctx.font = '14px Helvetica'; ctx.textAlign = 'center';
    ctx.fillText('no tags match', canvas.width / 2, canvas.height / 2);
    return;
  }

  WordCloud(canvas, {
    list: words,
    gridSize: 4,
    weightFactor: function(size) {
      // Log scale + hard cap: prevents dominant tags (65k files) from
      // consuming all canvas space and silently dropping mid-size tags
      var logScale = maxFreq > 1 ? Math.log(size + 1) / Math.log(maxFreq + 1) : 1;
      return Math.max(13, Math.min(logScale * canvas.width / 4, 72));
    },
    fontFamily: 'Helvetica, Arial, sans-serif',
    color: function() {
      var blues = ['#4a9eff','#7ecfff','#88bbff','#5ab4ff','#3388ee'];
      return blues[Math.floor(Math.random() * blues.length)];
    },
    backgroundColor: '#111',
    rotateRatio: 0.1,
    minSize: 10,
    drawOutOfBound: false,
    shuffle: false,
    click: function(item) { selectTag(item[0]); },
    hover: function(item) { canvas.style.cursor = item ? 'pointer' : 'default'; }
  });
}

function filterCloud() {
  var q = document.getElementById('search').value.trim().toLowerCase();
  FILTERED = q
    ? ALL_WORDS.filter(function(w) { return w[0].toLowerCase().includes(q); })
    : ALL_WORDS;
  document.getElementById('search-results').textContent =
    q ? FILTERED.length + ' match' + (FILTERED.length === 1 ? '' : 'es') : '';
  drawCloud(FILTERED);
}

// ── Single image viewer ───────────────────────────────────────────────────────
function selectTag(tag) {
  document.getElementById('active-tag').textContent = tag;
  CURRENT_INDICES = INDEX.tags[tag] || [];
  CURRENT_POS = 0;
  document.getElementById('tag-count').textContent =
    CURRENT_INDICES.length.toLocaleString() + ' file' + (CURRENT_INDICES.length === 1 ? '' : 's');
  document.getElementById('tag-btn').style.display = PORT ? 'inline-block' : 'none';
  showCurrent();
}

function showCurrent() {
  if (!CURRENT_INDICES.length) {
    document.getElementById('viewer-empty').style.display = '';
    document.getElementById('viewer-img').style.display = 'none';
    document.getElementById('nav-pos').textContent = '';
    document.getElementById('viewer-fname').textContent = '';
    document.getElementById('viewer-tags').innerHTML = '';
    return;
  }
  var pathIdx = CURRENT_INDICES[CURRENT_POS];
  var path = INDEX.paths[pathIdx];

  document.getElementById('viewer-empty').style.display = 'none';
  var img = document.getElementById('viewer-img');
  img.style.display = 'block';
  img.src = PORT ? PORT + '/file/' + encodeURIComponent(path) : '';

  document.getElementById('nav-pos').textContent =
    (CURRENT_POS + 1) + ' / ' + CURRENT_INDICES.length.toLocaleString();
  document.getElementById('viewer-fname').textContent = path.split('/').pop();

  var tags = (PATH_TAGS[pathIdx] || []).slice().sort();
  var tagsEl = document.getElementById('viewer-tags');
  tagsEl.innerHTML = '';
  tags.forEach(function(t) {
    var span = document.createElement('span');
    span.className = 'ftag';
    span.textContent = t;
    span.title = 'click to remove';
    span.onclick = function() { removeTag(t); };
    tagsEl.appendChild(span);
  });
}

function navigate(delta) {
  if (!CURRENT_INDICES.length) return;
  CURRENT_POS = (CURRENT_POS + delta + CURRENT_INDICES.length) % CURRENT_INDICES.length;
  showCurrent();
}

function openInViewer() {
  var tag = document.getElementById('active-tag').textContent;
  if (PORT) window.open(PORT + '/view/All?tag=' + encodeURIComponent(tag), '_blank');
}

function currentPath() {
  if (!CURRENT_INDICES.length) return null;
  return INDEX.paths[CURRENT_INDICES[CURRENT_POS]];
}

function addTagToFile() {
  var tag = document.getElementById('add-tag-input').value.trim();
  var path = currentPath();
  if (!tag || !path) return;
  var pathIdx = CURRENT_INDICES[CURRENT_POS];
  if (!PATH_TAGS[pathIdx]) PATH_TAGS[pathIdx] = [];
  if (PATH_TAGS[pathIdx].indexOf(tag) === -1) PATH_TAGS[pathIdx].push(tag);
  document.getElementById('add-tag-input').value = '';
  if (PORT) {
    fetch(PORT + '/api/addtag', {
      method: 'POST', headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({ filePath: path, tag: tag })
    }).catch(function() { alert('tag write failed — is server running?'); });
  }
  showCurrent();
}

function removeTag(tag) {
  var path = currentPath();
  if (!path) return;
  var pathIdx = CURRENT_INDICES[CURRENT_POS];
  if (PATH_TAGS[pathIdx]) {
    PATH_TAGS[pathIdx] = PATH_TAGS[pathIdx].filter(function(t) { return t !== tag; });
  }
  if (PORT) {
    fetch(PORT + '/api/removetag', {
      method: 'POST', headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({ filePath: path, tag: tag })
    }).catch(function() {});
  }
  showCurrent();
}

document.addEventListener('keydown', function(e) {
  if (document.activeElement === document.getElementById('add-tag-input')) {
    if (e.key === 'Enter') addTagToFile();
    return;
  }
  if (e.key === 'ArrowRight' || e.key === 'ArrowDown') navigate(1);
  if (e.key === 'ArrowLeft'  || e.key === 'ArrowUp')   navigate(-1);
});

// ── Boot ──────────────────────────────────────────────────────────────────────
async function boot() {
  try {
    const bytes = await gzipDecompress(DATA_B64);
    INDEX = JSON.parse(new TextDecoder().decode(bytes));

    // Build reverse lookup tables (O(1) tag reads, fully offline)
    INDEX.paths.forEach(function(p, i) { PATH_INDEX[p] = i; });
    Object.keys(INDEX.tags).forEach(function(tag) {
      INDEX.tags[tag].forEach(function(i) {
        if (!PATH_TAGS[i]) PATH_TAGS[i] = [];
        PATH_TAGS[i].push(tag);
      });
    });

    ALL_WORDS = (INDEX.freqs || []).map(function(f) { return [f[0], f[1]]; });
    FILTERED  = ALL_WORDS;

    document.getElementById('loading').style.display = 'none';
    canvas.style.display = 'block';
    var panel = document.getElementById('cloud-panel');
    canvas.width  = panel.offsetWidth - 20;
    canvas.height = panel.offsetHeight - 20;

    drawCloud(FILTERED);
  } catch(e) {
    document.getElementById('loading').textContent = 'error: ' + e.message;
    console.error(e);
  }
}

boot();
</script>
</body>
</html>`
