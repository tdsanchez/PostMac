# Personal Tool Suite

Personal tools built to solve real problems that no commercial product would address. AI-native development, trunk-based git, 12-factor philosophy. Every tool is a single binary or a single HTML file.

**tl;dr**: It's more like a gist than a repo.

---

## Philosophy

Apple's tools fail at scale. Finder chokes at ~10k files. Photos.app doesn't handle millions. APFS hides container-level space from users. These tools break through those limits — not by working around Apple's APIs but by ignoring them entirely and going lower.

The development model: natural language as development interface, git history as rationale, living documentation as operational infrastructure. Tools get built for an audience of one because that's now economically viable.

> "When personal tools are trivial to build, ecosystem lock-in loses its power."

---

## Tools

### media-server

Web-based media library manager for organizing and tagging large file collections. Tested at 168k+ files, architected for 5M+. Started manually (nohup or Ansible). Serves a browser UI for tagging via extended attributes, scanning, and cache-accelerated loads.

- Go, single binary
- SQLite cache (14x faster startup vs. cold scan)
- FSEvents for cache coherence

### publisher

Reads file paths from stdin. Produces a single self-contained HTML artifact containing all files as gzip+base64, a C/WASM markdown renderer (md4c via Emscripten), and a browser-native viewer. No server required. Optional AES-256-GCM encryption with PBKDF2 key derivation. Optional ring navigation for linked artifact collections.

```bash
find /some/dir -name "*.md" | publisher --output docs.html --title "My Docs"
```

See [publisher/README.md](publisher/README.md).

### bundler

Cousin of publisher. Same manifest format, same encryption, same ring nav — but uses marked.js instead of WASM for markdown rendering. Simpler dependency chain; suitable when WASM is more than needed.

```bash
cat paths.txt | bundler --output bundle.html --title "Bundle"
```

See [bundler/README.md](bundler/README.md).

### corpus-navigator

Queries a live media-server's API and builds a self-contained HTML corpus artifact: tag wordcloud, click-to-gallery, per-file tags, and tag writeback. The artifact is portable and offline-capable. Built to answer the question: what happens when the server goes away but the corpus needs to stay navigable?

```bash
corpus-navigator --server http://localhost:8898 --output corpus.html --title "VFP Corpus"
```

See [corpus-navigator/README.md](corpus-navigator/README.md).

### server

Produces `server.html` — a self-contained HTML artifact that IS the tool. Open it in a browser: pick files, browse by tag (file extension), optionally encrypt, save as a Publisher-format artifact. No backend. No install. No port.

`server.html` is committed and distributed directly. The Go binary rebuilds it.

```bash
./server --output server.html
open server.html
```

See [server/README.md](server/README.md).

---

## Simmering

A few more things under active development — not documented here yet, but worth knowing exist:

- **WASM viewer** — in-browser rendering experiments; feeds improvements back into publisher's mdrender.wasm pipeline
- **mediatunnel** — WASM-based filesystem bridge; ancestor to server
- **string-art** — computational art generator using the same corpus tooling

---

## Common Themes

**Go, single binary.** Fast compilation, no runtime dependency, great stdlib. Binaries are gitignored (arm64); rebuild from source.

**Self-contained HTML artifacts.** The artifact format is the delivery mechanism. gzip+base64 content, `DecompressionStream` for decompression, Web Crypto for encryption — all browser-native, zero JS library dependencies beyond wordcloud2 and marked.

**Trunk-based.** No feature branches. Always integrated. Git log is the design rationale.

**macOS-native integration.** Extended attributes for tags, FSEvents for cache coherence.

## Building

```bash
cd <tool>
go build -o <binary> ./cmd/<name>/
```

WASM assets (`assets/codec.wasm`, `assets/mdrender.wasm`) are tracked in git as build inputs. Rebuilding them requires Emscripten:

```bash
bash cmd/codec/build.sh      # → assets/codec.wasm
bash cmd/mdrender/build.sh   # → assets/mdrender.wasm
```

## License

Use freely. No warranty. Battle-tested on production systems; your mileage may vary.
