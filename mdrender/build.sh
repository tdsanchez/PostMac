#!/usr/bin/env bash
# build.sh — compile mdrender.c + md4c to mdrender.wasm via Emscripten
#
# Output: ../publisher/assets/mdrender.wasm and ../server/assets/mdrender.wasm
# Requires: emcc (brew install emscripten)

set -euo pipefail
cd "$(dirname "$0")"

echo "Building mdrender.wasm..."

emcc mdrender.c md4c.c md4c-html.c entity.c \
  -o mdrender.js \
  -O2 \
  --no-entry \
  -sEXPORTED_FUNCTIONS="['_render_markdown','_result_len','_free_result','_malloc','_free']" \
  -sEXPORTED_RUNTIME_METHODS="['cwrap']" \
  -sALLOW_MEMORY_GROWTH=1 \
  -sENVIRONMENT=web

for DEST in ../publisher/assets ../server/assets; do
  mkdir -p "$DEST"
  cp mdrender.wasm "$DEST/mdrender.wasm"
  RAW=$(wc -c < "$DEST/mdrender.wasm")
  GZ=$(gzip -c "$DEST/mdrender.wasm" | wc -c)
  echo "mdrender.wasm → $DEST/mdrender.wasm (${RAW} bytes raw, ${GZ} bytes gzipped)"
done
echo "Done."
