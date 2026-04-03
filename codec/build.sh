#!/usr/bin/env bash
# build.sh — compile codec.c to codec.wasm via Emscripten
#
# Output: ../publisher/assets/codec.wasm and ../server/assets/codec.wasm
# The .wasm is embedded in Server/Publisher artifacts via gzip+base64 bootstrap.
#
# Requires: emcc (brew install emscripten)

set -euo pipefail
cd "$(dirname "$0")"

echo "Building codec.wasm..."

emcc codec.c \
  -o codec.js \
  -sUSE_ZLIB=1 \
  -O2 \
  --no-entry \
  -sEXPORTED_FUNCTIONS="['_compress_buf','_decompress_buf','_gzip_isize','_free_buf','_malloc','_free']" \
  -sEXPORTED_RUNTIME_METHODS="['cwrap']" \
  -sALLOW_MEMORY_GROWTH=1 \
  -sENVIRONMENT=web

for DEST in ../publisher/assets ../server/assets; do
  mkdir -p "$DEST"
  cp codec.wasm "$DEST/codec.wasm"
  SIZE=$(wc -c < "$DEST/codec.wasm")
  echo "codec.wasm → $DEST/codec.wasm ($(( SIZE / 1024 )) KB)"
done
echo "Done."
