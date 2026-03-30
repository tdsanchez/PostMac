# bundler

Cousin of publisher. Reads file paths from stdin and builds a single self-contained HTML file with all content stored as gzip+base64 inline. Markdown is rendered client-side via marked.js (39KB). No WASM required.

Built for cases where publisher is more than needed — mixed-format document bundles that don't require the full WASM renderer stack. The artifact intentionally has no external dependencies and is opaque to crawlers.

## Usage

```bash
cat paths.txt | bundler --output bundle.html --title "My Bundle"
find /some/dir | bundler --output docs.html
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--output` | `bundle.html` | output HTML path |
| `--title` | `Published` | document title |
| `--cache` | `~/.media-server-conf/cache.db` | metadata cache (read-only, optional) |
| `--server` | — | media-server URL for live tag editing |
| `--passphrase` | — | encrypt with AES-256-GCM + PBKDF2 (100k iterations) |
| `--ring` | — | URL of ring.json for prev/next ring nav (fetched at build time) |
| `--url` | — | this artifact's own URL (used to locate it in ring.json) |
| `--mode` | `magazine` | `magazine` or `wordcloud` |
| `--index` | — | index.json path (required with `--mode wordcloud`) |

## Differences from publisher

| | bundler | publisher |
|---|---|---|
| Markdown renderer | marked.js (JS, 39KB) | mdrender.wasm (C, md4c) |
| WASM dependency | none | mdrender.wasm (optional) |
| Ring nav | fetched at build time, baked in | fetched live at page load |
| Encryption | AES-256-GCM + PBKDF2 | same |
| Format | same manifest schema | same manifest schema |

Both produce the same `PublishManifest` JSON structure and support the same MIME types. The difference is renderer and delivery mechanism for ring nav.

## Compression

Text, JSON, and SVG are gzip-compressed before base64 encoding. Images, PDFs, video, and audio are skipped (already compressed). Browser decompresses via `DecompressionStream` — no library.

## Unsupported file types

Flagged at build time with a warning to stderr. Shown as a placeholder in the artifact rather than failing the build.

## Build

```bash
go build -o bundler ./cmd/bundler/
```

The binary is not tracked in git. `assets/marked.min.js` must be present at build time for markdown rendering; without it, markdown falls back to `<pre>`.
