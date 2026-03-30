# publisher

Reads file paths from stdin. Produces a single self-contained HTML file that runs a WASM markdown renderer, displays images, PDFs, video, and audio natively, and carries all content as gzip+base64 inline. No server. No install. No network required after the file is open.

This is a personal tool built to solve a real problem: distributing curated collections of mixed-format files — markdown notes, die-shot images, webarchives, PDFs — as a single portable artifact with a browser-native viewer.

## Usage

```bash
cat paths.txt | publisher --output magazine.html --title "My Collection"
find /some/dir -name "*.md" | publisher --output docs.html
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--output` | `bundle.html` | output HTML path |
| `--title` | `Published` | artifact title |
| `--cache` | `~/.media-server-conf/cache.db` | metadata cache (read-only, optional) |
| `--wasm` | auto | path to mdrender.wasm |
| `--fresh` | false | recompile mdrender.wasm before bundling |
| `--server` | — | media-server URL for live tag editing |
| `--passphrase` | — | encrypt with AES-256-GCM + PBKDF2 (100k iterations) |
| `--ring` | — | URL of ring.json for ← prev \| index \| next → nav |
| `--mode` | `magazine` | `magazine` or `wordcloud` |
| `--index` | — | index.json path (required with `--mode wordcloud`) |

## What gets embedded

One JSON manifest containing all file content and metadata, gzip+base64 encoded. The WASM binary (`mdrender.wasm`, C/md4c) is also gzip+base64 and decompressed in-browser via `DecompressionStream`. No JavaScript libraries required.

**MIME → renderer mapping:**

| Type | Renderer |
|---|---|
| `text/markdown` | mdrender.wasm (C, md4c — GFM) |
| `image/*` | `<img>` data URI |
| `application/pdf` | `<embed>` data URI |
| `text/html` | `<iframe srcdoc>` |
| `video/*`, `audio/*` | `<video>` / `<audio>` data URI |
| other | `<pre>` fallback |

Text, JSON, and SVG are gzip-compressed before base64 encoding. Images (except SVG), PDFs, and video are already compressed and are skipped.

## Metadata enrichment

If `~/.media-server-conf/cache.db` is accessible, publisher enriches each file with tags, comments, and birth time from the scanner cache. Files not in the cache get stat()-based metadata only. No cache = graceful degradation.

## Encryption

`--passphrase foo` encrypts the manifest with AES-256-GCM. Key derivation: PBKDF2-SHA256, 100,000 iterations, random 16-byte salt. The artifact shows a lock screen at open. Web Crypto handles everything — no library.

## Build

```bash
go build -o publisher ./cmd/publisher/
```

Requires `assets/mdrender.wasm`. To rebuild it:
```bash
bash cmd/mdrender/build.sh   # requires emscripten
```

## Architecture notes

Three-level hierarchy: Go source → Go binary (runs on macOS) → HTML artifact (runs in browser). The binary is the factory; the artifact is the product. The binary is not tracked in git.

`--mode wordcloud` is the Cloudseed path — builds a wordcloud artifact from an index.json rather than a file list. Same binary, different output mode.

Publisher is the source of truth for the viewer HTML template. The `server` tool bakes this template in at build time rather than duplicating it.
