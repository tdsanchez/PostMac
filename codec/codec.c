/*
 * codec.c — gzip compress/decompress via zlib, compiled to WebAssembly.
 *
 * Bootstrap layer: codec.wasm is delivered via existing gzip+base64 scheme.
 * Once instantiated, all subsequent compress/decompress routes through here.
 *
 * Wire format: gzip (windowBits 15+16), compatible with existing artifacts.
 * Old artifacts decompress via DecompressionStream fallback.
 * New artifacts decompress via codec.wasm.
 *
 * Exported functions:
 *   compress_buf(src, src_len, out_len_ptr) → dst_ptr | 0
 *   decompress_buf(src, src_len, max_out, out_len_ptr) → dst_ptr | 0
 *   gzip_isize(src, src_len) → uint32 (uncompressed size from gzip trailer)
 *   free_buf(ptr)
 *
 * All pointers are WASM linear memory offsets.
 * Caller allocates via malloc (exported), frees via free_buf.
 *
 * Build:
 *   emcc codec.c -o codec.js -sUSE_ZLIB=1 -O2 --no-entry \
 *     -sEXPORTED_FUNCTIONS="['_compress_buf','_decompress_buf','_gzip_isize','_free_buf','_malloc','_free']" \
 *     -sEXPORTED_RUNTIME_METHODS="['cwrap']" \
 *     -sALLOW_MEMORY_GROWTH=1
 *   # then take codec.wasm alongside codec.js
 */

#include <zlib.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>

/*
 * compress_buf — gzip-compress src[0..src_len).
 * Writes compressed length to *out_len.
 * Returns pointer to compressed bytes (caller must free_buf), or NULL on error.
 */
uint8_t* compress_buf(const uint8_t* src, uint32_t src_len, uint32_t* out_len) {
    z_stream s;
    memset(&s, 0, sizeof(s));
    /* windowBits = 15+16 → gzip format */
    if (deflateInit2(&s, Z_BEST_COMPRESSION, Z_DEFLATED, 15|16, 8, Z_DEFAULT_STRATEGY) != Z_OK)
        return NULL;

    uLong bound = deflateBound(&s, src_len);
    uint8_t* dst = (uint8_t*)malloc(bound);
    if (!dst) { deflateEnd(&s); return NULL; }

    s.next_in   = (Bytef*)src;
    s.avail_in  = src_len;
    s.next_out  = dst;
    s.avail_out = bound;

    int r = deflate(&s, Z_FINISH);
    deflateEnd(&s);
    if (r != Z_STREAM_END) { free(dst); return NULL; }

    *out_len = (uint32_t)s.total_out;
    return dst;
}

/*
 * decompress_buf — gzip-decompress src[0..src_len).
 * max_out: upper bound on uncompressed size (use gzip_isize + padding).
 * Writes decompressed length to *out_len.
 * Returns pointer to decompressed bytes (caller must free_buf), or NULL on error.
 */
uint8_t* decompress_buf(const uint8_t* src, uint32_t src_len, uint32_t max_out, uint32_t* out_len) {
    z_stream s;
    memset(&s, 0, sizeof(s));
    /* windowBits = 15+32 → auto-detect gzip or zlib */
    if (inflateInit2(&s, 15|32) != Z_OK)
        return NULL;

    uint8_t* dst = (uint8_t*)malloc(max_out);
    if (!dst) { inflateEnd(&s); return NULL; }

    s.next_in   = (Bytef*)src;
    s.avail_in  = src_len;
    s.next_out  = dst;
    s.avail_out = max_out;

    int r = inflate(&s, Z_FINISH);
    inflateEnd(&s);
    if (r != Z_STREAM_END) { free(dst); return NULL; }

    *out_len = (uint32_t)s.total_out;
    return dst;
}

/*
 * gzip_isize — read the ISIZE field from a gzip stream's trailer.
 * ISIZE is the original (uncompressed) size mod 2^32, stored in the
 * last 4 bytes of the gzip file, little-endian.
 * Use this to size the output buffer for decompress_buf.
 */
uint32_t gzip_isize(const uint8_t* src, uint32_t src_len) {
    if (src_len < 4) return 0;
    const uint8_t* p = src + src_len - 4;
    return (uint32_t)p[0]
         | ((uint32_t)p[1] << 8)
         | ((uint32_t)p[2] << 16)
         | ((uint32_t)p[3] << 24);
}

/*
 * free_buf — release a buffer returned by compress_buf or decompress_buf.
 */
void free_buf(void* ptr) {
    free(ptr);
}
