/*
 * mdrender.c — Markdown → HTML via md4c, compiled to WebAssembly.
 *
 * Replaces viewer.wasm (Go runtime + goldmark, 5.8MB) with a focused
 * C implementation (~100KB). Same bootstrap delivery pattern as codec.wasm:
 * gzip+base64 in artifact, decompressed via bootstrap, instantiated once.
 *
 * Exported functions:
 *   render_markdown(ptr, len) → result_ptr
 *     Renders CommonMark + GFM extensions to HTML.
 *     Returns pointer to null-terminated HTML string.
 *     Caller must free with free_result().
 *
 *   result_len(ptr) → uint32
 *     Returns byte length of result (excluding null terminator).
 *
 *   free_result(ptr)
 *     Frees memory returned by render_markdown.
 *
 *   malloc / free — exposed for JS-side buffer management.
 *
 * Build:
 *   bash build.sh
 */

#include "md4c-html.h"
#include <stdlib.h>
#include <string.h>
#include <stdint.h>

/* ── Output buffer ────────────────────────────────────────────────────────── */

typedef struct {
    char*  data;
    size_t len;
    size_t cap;
} Buffer;

static void buf_append(const MD_CHAR* text, MD_SIZE size, void* userdata) {
    Buffer* b = (Buffer*)userdata;
    if (b->len + size + 1 > b->cap) {
        size_t newcap = b->cap ? b->cap * 2 : 4096;
        while (newcap < b->len + size + 1) newcap *= 2;
        b->data = (char*)realloc(b->data, newcap);
        b->cap  = newcap;
    }
    memcpy(b->data + b->len, text, size);
    b->len += size;
    b->data[b->len] = '\0';
}

/* ── Public API ───────────────────────────────────────────────────────────── */

/*
 * render_markdown — convert Markdown to HTML.
 * input: pointer to UTF-8 markdown text in WASM heap
 * len:   byte length of input
 * Returns pointer to null-terminated HTML string (caller must free_result).
 * Returns NULL on error.
 */
char* render_markdown(const char* input, uint32_t len) {
    Buffer b = {0};

    /* MD_DIALECT_GITHUB = tables + strikethrough + tasklists + permissive autolinks */
    unsigned flags = MD_DIALECT_GITHUB;

    int r = md_html(input, (MD_SIZE)len, buf_append, &b, flags, 0);
    if (r != 0) {
        free(b.data);
        return NULL;
    }

    /* Ensure null-terminated even on empty output */
    if (!b.data) {
        b.data = (char*)malloc(1);
        b.data[0] = '\0';
        b.len = 0;
    }

    return b.data;
}

/*
 * result_len — byte length of a render_markdown result.
 */
uint32_t result_len(const char* ptr) {
    if (!ptr) return 0;
    return (uint32_t)strlen(ptr);
}

/*
 * free_result — release memory returned by render_markdown.
 */
void free_result(char* ptr) {
    free(ptr);
}
