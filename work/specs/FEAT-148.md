---
id: FEAT-148
title: "Image Transformation and Caching"
status: implemented
priority: medium
created: 2026-06-28
author: "@human / @ai"
---

# FEAT-148: Image Transformation and Caching

## Summary

Basil should provide built-in image transformation and caching so that small-to-medium site developers never need to install a separate image library, pay for a cloud image service, or manually resize and convert images. The feature should "just work" with zero configuration, produce good-looking images by default, and be overridable when needed. Images are transformed once, cached to disk, and served via content-hashed immutable URLs.

## User Story

As a Basil developer, I want to reference images in my Parsley templates and have them automatically optimized, resized, and served with correct caching so that my site loads fast and looks good without manual image processing or third-party services.

## Motivation

Image handling is the last significant gap in Basil's "batteries included" story. Currently, a developer using Basil must either:

1. Manually resize, crop, and convert images using external tools (Photoshop, ImageMagick, FFmpeg)
2. Install and configure a separate image processing library or server (Sharp, Thumbor, Imaginary)
3. Pay for a cloud image service (Cloudflare Images, imgix, Cloudinary)

All three options contradict Basil's philosophy: a single binary that handles the common web development workflow end-to-end. A developer with a folder of photos should be able to write `image(@./photo.jpg)` and get an optimized, properly-rotated image served with correct caching headers.

### Design Principles

- **Zero-config by default**: `image(@./photo.jpg)` should produce the right result without any configuration
- **Secure by construction**: only transformations defined in Parsley code can be generated — no URL parameter manipulation
- **Transform once, serve forever**: images are transformed on first request, cached to disk, and served with immutable cache headers
- **Pure Go**: no C dependencies, no CGo — Basil remains a single static binary
- **The Parsley aesthetic**: simple, minimal, complete, composable

## Acceptance Criteria

### Phase 1: Core (1.0 or 1.1)

- [ ] `image(path)` builtin: auto-rotate (EXIF), strip metadata, cache, return hashed URL (original format preserved)
- [ ] `image(path, options)` builtin: resize (width/height/fit), set quality, choose format
- [ ] `image(path, style)` builtin: apply a named style (dict with transformation options)
- [ ] `imageInfo(path)` builtin: return `{width, height, format, orientation}` dict
- [ ] Disk cache at configurable location (default `./cache/images/`)
- [ ] `/__img/{hash}.{ext}` handler serving cached images with immutable cache headers
- [ ] Dev mode: no-cache headers, re-transform on source change
- [ ] Security: path must be within handler root (same rules as `publicUrl()`)
- [ ] Size limits: warn at 10MB source, reject at 50MB source
- [ ] Supported input formats: JPEG, PNG, GIF, WebP (decode only)
- [ ] Supported output formats: JPEG, PNG, WebP (opt-in via `{format: "webp"}`)
- [ ] Default output format: **original** (JPEG in → JPEG out, PNG in → PNG out)

### Phase 2: Lovable (1.1 or 1.2)

- [ ] `imageSrcset(path, style, scales)` builtin: generate `srcset` attribute string for responsive images
- [ ] Blur placeholder generation: `{blur: true}` option producing a tiny blurred image for progressive loading
- [ ] Dominant color extraction in `imageInfo()`: `{color: "#2a4f3b"}`
- [ ] Smart crop via focal point detection: `{crop: "smart"}`
- [ ] Sharpen on downscale: subtle unsharp mask when reducing image dimensions significantly

### Phase 3: Nice to Have (post-1.2)

- [ ] AVIF output format support
- [ ] SVG optimization (metadata stripping, minification)
- [ ] `basil cache clear` CLI subcommand
- [ ] Cache size limits / LRU eviction policy
- [ ] Color palette extraction in `imageInfo()`

## Design Decisions

### Pure Go, No libvips

**Decision**: Use pure Go image libraries. Do not depend on libvips, ImageMagick, or any C library.

**Rationale**: Basil's deployment story is "single binary, works everywhere." Adding a C dependency would require system package installation (`apt-get install libvips-dev`), break cross-compilation, and complicate Docker images. The performance difference is irrelevant because images are transformed once and cached — a 200ms transform vs. a 40ms transform is invisible when the result is served from disk cache for all subsequent requests.

**Libraries** (all pure Go, no CGo):

| Library | Purpose | Status | License |
|---------|---------|--------|---------|
| `disintegration/imaging` | Resize, crop, rotate, sharpen, blur, EXIF auto-orient | Stable, unmaintained since 2020 (5.7k ⭐, 10k+ dependents) | MIT |
| `gen2brain/webp` | WebP encode/decode (CGo-free via WASM) | Active (54 ⭐), uses wazero runtime | MIT |
| `golang.org/x/image/webp` | WebP decode only | Active (Go team) | BSD-3 |

**Evaluated and rejected:**

| Library | Reason |
|---------|--------|
| `chai2010/webp` | Requires CGo (vendored C source), abandoned |
| `kolesa-team/go-webp` | Requires CGo + system libwebp |
| `rwcarlsen/goexif` | Unnecessary — `imaging.AutoOrientation(true)` handles EXIF |
| `muesli/smartcrop` | Phase 2 — not needed for Phase 1 |

**Phase 1 dependencies:**
- `disintegration/imaging` — core transform engine (resize, crop, rotate, auto-orient, blur, sharpen, JPEG/PNG encode/decode)
- `gen2brain/webp` — opt-in WebP encoding (CGo-free via WASM + wazero). Encode is ~5x slower than native (~96ms vs ~19ms) but acceptable since results are cached. If binary size or dependency cost proves too high, WebP encoding can be deferred without affecting the rest of Phase 1.
- `golang.org/x/image` — WebP decoding (for accepting WebP source images)

### Builtins, Not URL Parameters

**Decision**: Image transformations are defined in Parsley code via `image()` builtin calls. The resulting URLs are opaque content hashes. There is no URL-parameter-based transformation API.

**Rationale**: URL-based APIs (e.g., `?w=300&h=200&q=80`) allow anyone to request arbitrary transformations, creating a denial-of-service vector (each unique parameter set generates a new cached file). Mitigations (signed URLs, allowlists) add complexity. By routing through Parsley code, only transformations the developer defines can be generated. The URL is an opaque hash — secure by construction, no signing needed.

**Tradeoff**: This means images cannot be transformed by external systems or CDNs via URL manipulation. This is acceptable for Basil's target audience. Developers who need CDN-level image transformation should use a CDN.

### Styles Defined in Parsley, Not Config

**Decision**: Image styles (reusable transformation presets) are plain Parsley dictionaries, not YAML config entries.

**Rationale**:
- Config changes require a server restart (HUP); Parsley code changes are picked up on next request (in dev mode) or next deploy
- Image styles are a *design* concern (how images look), not an *infrastructure* concern (how the server runs)
- Parsley dicts are composable — styles can be merged, extended, or computed
- Keeps `basil.yaml` focused on server configuration

**Example**:
```parsley
// Define styles as regular Parsley dictionaries
styles = {
  thumbnail: { width: 150, height: 150, crop: "center" },
  hero:      { width: 1200 },
  card:      { width: 400, height: 300, crop: "center" },
}

// Use them
<img src={image(@./photo.jpg, styles.thumbnail)} />
```

### Default Format: Original, Not WebP

**Decision**: `image(@./photo.jpg)` with no format option preserves the original format (JPEG → JPEG, PNG → PNG). WebP is available via explicit `{format: "webp"}`.

**Rationale**:
- Least surprising behavior — a JPEG stays a JPEG unless you ask for something else
- WebP encoding depends on `gen2brain/webp` (WASM-based), which adds binary size and a wazero dependency. Making it opt-in means the dependency cost is justified only when actually used.
- Developers who want WebP can trivially opt in: `image(@./photo.jpg, {format: "webp"})`
- A site-wide default can be set via config (`default_format: webp`) or via Parsley style dicts

**Previous direction**: Earlier draft defaulted to WebP output. Revised per product decision that automatic WebP conversion is not a priority, but WebP support should be available when feasible.

### Format Baked Into URL, Not Content-Negotiated

**Decision**: The output format is determined at transform time and baked into the cached filename and URL. There is no per-request `Accept` header negotiation.

**Rationale**:
- Simplest possible caching — one URL, one file, one format
- CDN-friendly — CDNs cache by URL, not by `Accept` header (Vary: Accept causes cache fragmentation)
- Predictable — the developer knows exactly what format will be served

### Disk Cache, Not Memory Cache

**Decision**: Transformed images are cached to disk, not held in memory.

**Rationale**: Images are large (100KB–5MB each). A memory cache would either be too small to be useful or consume too much RAM. Disk cache is bounded only by disk space, survives server restarts, and serving from disk via `http.ServeFile` is efficient (the OS page cache handles hot files).

### Cache Invalidation: Content-Addressed

**Decision**: Cache keys are derived from `SHA256(source_file_contents + transformation_params)`. When the source file changes, a new cache entry is created. Old entries become orphans.

**Rationale**: Content-addressed caching is simple, correct, and requires no invalidation logic. Orphan cleanup is a Phase 3 concern — developers can delete the cache directory at any time, and it rebuilds on demand.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Architecture Overview

The image system mirrors the existing `publicUrl()` → `assetRegistry` → `assetHandler` pattern:

```
Parsley code                    Server                         Client
─────────────                   ──────                         ──────
image(@./photo.jpg, opts)
  │
  ▼
evalImage()                     
  ├─ resolve path (same as publicUrl)
  ├─ security check (within handler root)
  ├─ compute cache key: SHA256(file_contents + opts_canonical)
  ├─ check disk cache
  │   ├─ HIT: return cached URL
  │   └─ MISS:
  │       ├─ load source image
  │       ├─ auto-rotate (EXIF)
  │       ├─ apply transforms (resize, crop, quality)
  │       ├─ encode to output format
  │       ├─ write to cache dir (atomic: temp file + rename)
  │       └─ register with imageRegistry
  └─ return /__img/{hash}.{ext}
                                                               GET /__img/{hash}.jpeg
                                imageHandler.ServeHTTP()           │
                                  ├─ lookup hash in registry       │
                                  ├─ serve from disk cache ───────►│
                                  └─ Cache-Control: immutable      │
```

### Affected Components

- **NEW** `server/images/` — Image transformation package (transform, cache, registry, handler)
- **NEW** `pkg/parsley/evaluator/image.go` — `image()` and `imageInfo()` builtins
- **MODIFY** `server/server.go` — Initialize image registry/cache, mount `/__img/` handler, inject into evaluator environment
- **MODIFY** `server/config/config.go` — Add `Images` config section
- **MODIFY** `pkg/parsley/evaluator/evaluator.go` — Add `ImageRegistrar` interface to `Environment`

### New Package: `server/images/`

```
server/images/
├── transform.go    # Image loading, EXIF rotation, resize, crop, format conversion
├── cache.go        # Disk cache: write, read, key computation
├── registry.go     # Hash → filepath mapping (analogous to assetRegistry)
├── handler.go      # HTTP handler for /__img/ (analogous to assetHandler)
└── options.go      # TransformOptions struct, parsing from Parsley dicts
```

### TransformOptions

```go
type TransformOptions struct {
    Width   int    // Target width (0 = auto/preserve aspect ratio)
    Height  int    // Target height (0 = auto/preserve aspect ratio)
    Crop    string // "center", "smart" (Phase 2), or "" (no crop — fit within box)
    Quality int    // 1-100, 0 = format default (JPEG: 85, WebP: 80, PNG: lossless)
    Format  string // "webp", "jpeg", "png", "original", or "" (default = "original")
    Blur    float64 // Blur radius, 0 = none (Phase 2)
    Sharpen float64 // Sharpen amount, 0 = none (Phase 2)
}
```

### Builtin Signatures

#### `image(path)` → `string`
- Auto-rotate, strip metadata, cache in original format
- Returns `/__img/{hash}.{ext}` (extension matches source format)

#### `image(path, options_dict)` → `string`
- Apply specified transformations
- `options_dict` keys: `width`, `height`, `crop`, `quality`, `format`
- Phase 2 adds: `blur`, `sharpen`
- Returns `/__img/{hash}.{ext}`

#### `image(path, style_dict)` → `string`  
- Same as above — a "style" is just a dict with the same keys
- No special style registration mechanism needed; styles are regular Parsley values

#### `imageInfo(path)` → `dict`
- Returns `{width: int, height: int, format: string, orientation: string}`
- `orientation`: `"landscape"`, `"portrait"`, or `"square"`
- Phase 2 adds: `color: string` (dominant color as hex)
- Does NOT transform the image — reads metadata only
- Results cached in memory (metadata is small)

### Cache Key Computation

```go
func cacheKey(sourceHash string, opts TransformOptions) string {
    // Canonical representation of options for stable hashing
    canonical := fmt.Sprintf("%s|w=%d|h=%d|c=%s|q=%d|f=%s",
        sourceHash, opts.Width, opts.Height, opts.Crop,
        opts.Quality, opts.Format)
    hash := sha256.Sum256([]byte(canonical))
    return hex.EncodeToString(hash[:])[:16]
}
```

The source hash is `SHA256(file_contents)[:16]`, same as `publicUrl()`. The cache key combines source identity with transformation parameters, so the same source with different options produces different cache entries.

### Disk Cache Layout

```
./cache/images/
├── a3f7b2c1e4d5f6a7.jpeg    # photo.jpg → default (auto-rotated, metadata stripped)
├── b8c9d0e1f2a3b4c5.jpeg    # photo.jpg → {width: 300}
├── c4d5e6f7a8b9c0d1.webp    # photo.jpg → {width: 300, format: "webp"}
└── ...
```

Flat directory. No subdirectories. Filenames are cache keys + output extension. Simple to enumerate, simple to delete.

### Config Addition

```yaml
# basil.yaml — all fields optional, shown with defaults
images:
  cache_dir: ./cache/images   # Where transformed images are stored
  max_width: 4096             # Maximum output width (safety limit)
  max_height: 4096            # Maximum output height (safety limit)
  default_quality: 0          # 0 = format-specific default (JPEG: 85, WebP: 80, PNG: lossless)
  default_format: ""          # "" = original format; "webp", "jpeg", "png" to override
```

### Environment Integration

Add to `pkg/parsley/evaluator/evaluator.go`:

```go
type ImageRegistrar interface {
    // Transform transforms an image with the given options, caches the result,
    // and returns the public URL (e.g., /__img/{hash}.jpeg).
    Transform(sourcePath string, opts ImageOptions) (string, error)

    // Info returns metadata about the source image without transforming it.
    Info(sourcePath string) (ImageInfo, error)
}
```

Set `env.ImageRegistry` in `server/handler.go` and `server/api.go`, same pattern as `env.AssetRegistry`.

### Handler: `/__img/`

Identical pattern to `assetHandler` at `/__p/`:
- Extract hash from URL path
- Look up filepath in registry
- Verify extension matches
- Set `Cache-Control: public, max-age=31536000, immutable` (production) or `no-cache` (dev)
- Serve via `http.ServeFile`

### Security

- **Path containment**: Same security check as `publicUrl()` — source path must be within handler root (`env.RootPath`)
- **No URL-parameter attack surface**: Transformations are defined in code, not in URLs
- **Size limits**: Reject source files >50MB, warn >10MB
- **Dimension limits**: `max_width` and `max_height` cap output dimensions (default 4096×4096)
- **No path traversal in cache**: Cache filenames are hex hashes, no user-controlled path components
- **Metadata stripping**: EXIF data (which may contain GPS coordinates, camera serial numbers, etc.) is stripped from output by default

### Performance Considerations

- **First request**: Transformation cost depends on image size and operation. Resize of a 12MP JPEG to 300px wide: ~100-300ms in pure Go. Acceptable for a one-time cost.
- **Subsequent requests**: Served from disk cache via `http.ServeFile`. OS page cache handles frequently-accessed files. Effectively zero overhead.
- **Concurrent transforms**: Multiple requests for the same uncached image should not trigger duplicate transforms. Use `singleflight` (from `golang.org/x/sync/singleflight`) or a simple per-key mutex.
- **Startup**: No work at startup. Cache is populated lazily on first access.
- **Dev mode**: Check source file modTime before serving cached version. If source changed, re-transform.
- **WebP encoding via WASM**: ~96ms per encode vs ~19ms native. Acceptable for cached one-time transforms.

### Edge Cases & Constraints

1. **Animated GIF**: Do not transform animated GIFs — serve as-is (or strip to first frame with a warning). Resizing animated GIFs frame-by-frame is complex and slow.
2. **SVG input**: Not supported in Phase 1. `image()` on an SVG should return a clear error with a hint to use `publicUrl()` instead.
3. **Very large images** (>50MP): May cause high memory usage during decode. The 50MB file size limit provides some protection, but a decoded 50MP image is ~200MB in memory. Consider a dimension limit on *source* images as well (e.g., 8192×8192 max input).
4. **Upscaling**: If requested dimensions are larger than source, do not upscale — return at source dimensions. Upscaling produces blurry results and wastes bandwidth.
5. **Aspect ratio**: When both `width` and `height` are specified without `crop`, fit within the box (preserve aspect ratio, no distortion). With `crop: "center"`, fill the box and crop excess.
6. **Cache directory missing**: Create `cache_dir` on first write. Do not fail at startup if it doesn't exist.
7. **Cache directory on read-only filesystem**: Surface a clear error on first transform attempt. `imageInfo()` should still work (it doesn't write to cache).
8. **Concurrent cache writes**: Use atomic write (write to temp file, rename) to prevent serving partial files.
9. **GIF input, non-GIF output**: When converting a GIF to JPEG or WebP, use the first frame.
10. **WebP input**: Decode via `golang.org/x/image/webp`. If `gen2brain/webp` is present, it may provide its own decoder — prefer `x/image/webp` for decode to minimize WASM overhead.

### Dependencies

- **Depends on**: FEAT-002 (static file serving, handler roots, path resolution), `publicUrl()` pattern (architecture template)
- **Blocks**: Nothing — this is additive
- **Related**: FileField design (`work/design/DESIGN-file-field.md`) — Phase 3 of FileField mentions image preview thumbnails, which could use this system

## Implementation Notes

### Post-Implementation Review (2026-06-28)

Review of Phase 1 implementation found one bug, several test coverage gaps, and minor spec deviations. Overall assessment: solid implementation that mirrors existing codebase patterns correctly.

#### Bug: Variable Shadowing in `registry.go` `doTransform`

In `server/images/registry.go` `doTransform()`, line 155 uses `:=` for the `err` variable inside the `else` branch, which **shadows** the outer `err` declared on line 147. This means the `if err != nil` check after the if/else block (line 165) always sees the outer `err`, which is never assigned in the `else` branch. In practice, the early returns at lines 156 and 161 mask the bug — but it is still incorrect and fragile. Fix: use `=` instead of `:=` on line 155 with a separate `var data []byte` declaration.

**Severity:** Low (masked by early returns, but could surface if code is refactored).
**Status:** To fix.

#### Spec Deviations (Acceptable)

1. **`ImageRegistrar` uses `map[string]any` not typed structs** — The spec suggested `ImageOptions`/`ImageInfo` types on the interface. The implementation uses `map[string]any` at the interface boundary to avoid importing `server/images` types into the evaluator package (which would create a circular dependency). Typed structs are used internally. This is the correct tradeoff.

2. **`gen2brain/webp` not included** — WebP encoding returns a clear error. WebP decoding (input) works via `golang.org/x/image/webp`. This was a deliberate decision to avoid WASM binary size overhead. Documented in conversation and plan.

3. **`imageInfo()` metadata not cached in memory** — The spec says "Results cached in memory (metadata is small)." Currently `Info()` calls `GetInfo()` every time without memoization. Low impact since `image.DecodeConfig` only reads headers, but could matter in gallery loops.

4. **No bulk cache directory scan on startup** — The spec's Task 5 says "scan cacheDir for existing cached files." The implementation uses per-key lazy probing via `os.Stat` instead. This is arguably better — avoids startup latency and doesn't require reverse-engineering options from filenames.

#### Test Coverage Gaps

1. **No Registry tests** — `registry.go` (the most complex component: singleflight, cache coordination, format determination) has zero dedicated tests. `Transform()`, `doTransform()`, `Info()`, `Lookup()`, `Clear()`, and `CacheStats()` are all untested.

2. **No Handler tests** — `handler.go` HTTP handler is untested. Missing: serve cached image, 404 for unknown hash, extension mismatch → 404, cache headers in prod vs dev.

3. **No evaluator builtin tests** — `image()` and `imageInfo()` in `evaluator/image.go` are untested. Missing: no registry → error, bad args → arity error, path dict vs string handling, security path-outside-root → error.

4. **`TestProcess_WebPInputFallsBackToJPEG` doesn't test actual WebP input** — The test uses JPEG sources with explicit format options. It does not exercise the WebP→JPEG fallback code path in `Process()`.

5. **`TestLoad_Errors` file-too-large is skipped** — The 50MB size limit check is untested.

6. **No concurrency test for singleflight dedup** — The plan lists this as a required test for Task 5.

#### Minor Code Quality Notes

- **`SourceHash` reads entire file into memory** — For a 50MB file (the max source size), this allocates 50MB just to hash. Using `io.Copy` to a `sha256.New()` writer would stream without buffering. Low priority since images are typically 1–10MB.

- **Quality default application is split** — Config defaults are applied in `Registry.Transform()`, format-specific defaults in `Encode()`. Both paths are correct but the split could confuse future maintainers.

### Suggested Implementation Order

1. `server/images/options.go` — Define `TransformOptions`, parsing from Parsley dicts
2. `server/images/transform.go` — Core transform pipeline: load → EXIF rotate → resize/crop → encode
3. `server/images/cache.go` — Disk cache: key computation, read, atomic write
4. `server/images/registry.go` — Hash-to-filepath mapping (mirror `assetRegistry`)
5. `server/images/handler.go` — HTTP handler (mirror `assetHandler`)
6. `pkg/parsley/evaluator/image.go` — `image()` and `imageInfo()` builtins
7. Integration: wire into `server.go`, `handler.go`, `api.go`, `config.go`
8. Tests: unit tests for transform, cache, options parsing; integration tests for builtins

### Library Evaluation Results

Completed 2026-06-28. See "Pure Go, No libvips" design decision above for full results.

- [x] `disintegration/imaging`: pure Go, MIT, stable (5.7k ⭐, 10k+ dependents), Lanczos resampling, built-in EXIF auto-orientation. Unmaintained since 2020 but mature. **Selected for Phase 1.**
- [x] WebP encoder: `gen2brain/webp` is CGo-free (WASM via wazero). ~5x slower encode than native, acceptable with caching. MIT license. **Selected for Phase 1 (opt-in).**
- [x] `chai2010/webp`: **Rejected** — requires CGo (vendored C source), abandoned.
- [x] `kolesa-team/go-webp`: **Rejected** — requires CGo + system libwebp.
- [x] EXIF: `imaging.AutoOrientation(true)` handles orientation. No separate EXIF library needed.
- [x] All selected libraries: MIT or BSD-3 licensed.
- [x] `golang.org/x/image/webp`: decode-only, used for WebP input support.

## Related

- Architecture template: `publicUrl()` in `pkg/parsley/evaluator/public_url.go`, `assetRegistry` / `assetHandler` in `server/assets.go`
- Config pattern: `server/config/config.go`
- FileField design: `work/design/DESIGN-file-field.md` (Phase 3: image preview thumbnails)
- Compression feature: `work/specs/FEAT-064.md` (similar "middleware infrastructure" pattern)
- Plan: `work/plans/FEAT-148-plan.md`
