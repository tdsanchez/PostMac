# server

Produces a single self-contained HTML artifact (`server.html`). When opened in any modern browser, it lets you pick local files, browse them by tag (file extension), and save the selection as an encrypted or unencrypted Publisher-format artifact. No backend. No install. No ports.

The artifact IS the tool. `server.html` is committed to the repo and distributed directly. The Go binary exists only to rebuild it.

## Usage

```bash
# Build the artifact
server --output server.html

# Development mode — serves on a local port, registers a Service Worker
server --output server.html --serve 8080
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--output` | `server.html` | output path |
| `--title` | `Server` | page title |
| `--serve` | 0 | dev HTTP port (also registers SW) |
| `--wasm` | auto | path to mdrender.wasm |

## What the artifact does

1. User opens `server.html` in a browser
2. Clicks "Open Files" → `showOpenFilePicker` (Chrome/Edge) or `<input type=file>` fallback (Firefox/Safari/Waterfox)
3. Files are grouped by extension and rendered as a tag wordcloud
4. Click a tag → file list → iframe viewer
5. Optional: fill in a passphrase (leave empty for unencrypted)
6. Click "Save" → downloads a Publisher-format `.html` artifact containing all selected files

The saved artifact is fully self-contained: mdrender.wasm, all file content, wordcloud, gallery, tag browser — everything in one file.

## What's baked in at build time

| Asset | Source | How embedded |
|---|---|---|
| wordcloud2.js | `assets/wordcloud2.min.js` | inline script |
| codec.wasm | `assets/codec.wasm` | gzip+base64 |
| mdrender.wasm | `assets/mdrender.wasm` | gzip+base64 (for saved artifacts) |
| Publisher template | `cmd/publisher/main.go` | gzip+base64 (for Save) |

Publisher stays the source of truth for the viewer format. No duplication.

## Compression

The C codec (`assets/codec.wasm`, built via Emscripten from `cmd/codec/codec.c`) handles gzip at runtime in the browser. Bootstrap: codec.wasm itself arrives as gzip+base64 and is decompressed via `DecompressionStream` — the one permanent use of the browser fallback. After that, all compress/decompress calls go through `CODEC.compress()` / `CODEC.decompress()` (C, via WASM).

## Encryption

Passphrase field in the artifact header. Leave empty → unencrypted. Fill → AES-256-GCM with PBKDF2-SHA256 key derivation (100,000 iterations, random 16-byte salt). Web Crypto handles everything. The saved artifact uses Publisher's existing lock screen — no extra code needed.

## Browser compatibility

| Browser | File picker | codec.wasm | Save |
|---|---|---|---|
| Chrome / Edge | `showOpenFilePicker` | yes | yes |
| Firefox / Safari / Waterfox | `<input type=file>` fallback | yes | yes |

## Lineage

```
media-server-arm64 (Go, native)
  └── mediatunnel (hard fork — WASM experiment)
        └── server (hard fork — browser-native, no backend)
```

## Build

```bash
go build -o server ./cmd/server/
./server --output server.html
```

To rebuild the C codec:
```bash
bash cmd/codec/build.sh   # requires emscripten
```

The Go binary is not tracked in git. `server.html` is tracked — it's the deliverable.
