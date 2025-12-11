# FEAT-064: Gzip/Zstd Response Compression

**Status:** Implemented  
**Created:** 2025-12-11  
**Author:** AI Assistant  

## Summary

Add transparent HTTP response compression to Basil using the `klauspost/compress/gzhttp` middleware. This will compress all eligible responses (HTML, CSS, JavaScript, JSON, etc.) automatically, improving page load times and reducing bandwidth usage.

## Motivation

Basil currently serves all responses uncompressed via `http.ServeFile` and direct response writers. Modern browsers universally support gzip compression, and most also support the newer Zstd algorithm which offers better compression ratios.

Adding compression:
- Reduces bandwidth usage by 60-80% for text content
- Improves page load times, especially on slower connections
- Is expected by users of modern web frameworks
- Benefits all of Basil: static files, generated HTML, API responses, bundled assets (FEAT-063)

## Design

### Middleware Approach

Use `klauspost/compress/gzhttp` as a top-level middleware wrapper around the entire HTTP handler. This provides:
- Automatic gzip compression for eligible responses
- Optional Zstd support for browsers that accept it
- BREACH attack mitigation via random padding
- Sensible defaults (1024 byte minimum, skip pre-compressed files)

### Implementation

```go
import "github.com/klauspost/compress/gzhttp"

// In server setup
handler := buildHandler()  // existing router
handler = gzhttp.GzipHandler(handler)
```

### Configuration

Add optional compression settings to `basil.yaml`:

```yaml
server:
  compression:
    enabled: true          # default: true
    level: default         # options: fastest, default, best, none
    min_size: 1024         # minimum response size to compress (bytes)
    zstd: false            # enable Zstd for supporting browsers
```

### Defaults

- **Enabled by default:** Compression should "just work" for new users
- **Minimum size:** 1024 bytes (gzhttp default) - small responses aren't worth compressing
- **Skip extensions:** `.gz`, `.zip`, `.png`, `.jpg`, `.webp`, etc. (already compressed)
- **Content types:** HTML, CSS, JavaScript, JSON, XML, plain text

### What Gets Compressed

| Content Type | Compressed |
|-------------|------------|
| `text/html` | ✅ |
| `text/css` | ✅ |
| `application/javascript` | ✅ |
| `application/json` | ✅ |
| `text/plain` | ✅ |
| `image/svg+xml` | ✅ |
| Pre-compressed files (`.gz`) | ❌ |
| Binary images (`.png`, `.jpg`) | ❌ |
| Small responses (<1024 bytes) | ❌ |

## Library Choice

**Selected:** `github.com/klauspost/compress/gzhttp`

| Criteria | klauspost/compress/gzhttp | NYTimes/gziphandler |
|----------|---------------------------|---------------------|
| Status | Active (v1.18.2, Dec 2025) | Abandoned (Feb 2019) |
| Performance | ~2x faster | Baseline |
| Memory | ~70% less | Baseline |
| Zstd Support | ✅ | ❌ |
| BREACH Mitigation | ✅ | ❌ |
| API | `gzhttp.GzipHandler(h)` | `gziphandler.GzipHandler(h)` |

## Dev Mode Considerations

- Compression remains enabled in dev mode (matches production behavior)
- Consider adding response header `X-Compression: gzip` in dev mode for debugging
- LiveReload responses should still be compressed

## Testing

1. Verify `Content-Encoding: gzip` header present for eligible responses
2. Verify small responses (<1024 bytes) are not compressed
3. Verify pre-compressed static files are served directly
4. Verify binary content types are not compressed
5. Benchmark response times with and without compression
6. Test with `curl --compressed` and browser dev tools

## Open Questions

1. Should compression be enabled by default, or opt-in?
   - **Recommendation:** Enabled by default (it's what users expect)

2. Should we support Zstd?
   - **Recommendation:** Optional, disabled by default (newer, less universal)

3. Should there be a way to disable compression for specific routes?
   - **Recommendation:** Defer to BACKLOG unless there's a clear use case

## Related

- FEAT-063: CSS/JS Auto-Bundle (bundled assets benefit from compression)

## Implementation Notes

*To be filled in during implementation*
