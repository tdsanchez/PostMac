# corpus-navigator

Queries a live media-server's API and builds a fully self-contained HTML corpus artifact: a tag wordcloud, click-to-gallery viewer, per-file tag display, and tag writeback. The artifact is portable and works offline — tag writes require the server to be running, browsing does not.

Built to solve the problem of making a tagged image corpus navigable without being tied to a running server. The artifact IS the front end for the corpus it represents.

## Usage

```bash
corpus-navigator --server http://localhost:8898 --output corpus.html
corpus-navigator --server http://localhost:8898 --output corpus.html --title "VFP Corpus"
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--server` | required | media-server base URL |
| `--output` | `corpus.html` | output HTML path |
| `--title` | `Corpus Navigator` | page title |

## What it does

1. Fetches all tags from `/api/alltags`
2. Fetches file paths for each tag from `/api/filelist?category=<tag>` (up to 20 concurrent requests)
3. Builds a deduped path index and tag→path-indices manifest
4. Serializes, gzip-compresses, base64-encodes the manifest
5. Injects everything into a self-contained HTML template with wordcloud2.js inline

The manifest structure:
```json
{ "paths": [...], "tags": { "tagname": [idx,...] }, "freqs": [[tag, count],...] }
```

The browser decompresses and builds reverse lookup tables at boot (O(1) tag reads, fully offline after that).

## Important: do not use cache.db

All `media-server-arm64` instances share `~/.media-server-conf/cache.db` with no port isolation — whichever instance last ran owns it. corpus-navigator always queries the live API. Never pass a `--cache` flag; there isn't one by design.

## Tag writeback

When `--server` is embedded and the server is reachable, the artifact POSTs to `/api/addtag` and `/api/removetag`. If the server is down, browsing still works — tag writes fail silently with an alert.

## Output size

A corpus of 320k images with ~2k tags compresses to roughly 7.5MB. A small curated corpus (a few thousand images, ~200 tags) typically comes out under 2MB.

## Build

```bash
go build -o corpus-navigator ./cmd/corpus/
```

Requires `assets/wordcloud2.min.js`. The binary is not tracked in git.
