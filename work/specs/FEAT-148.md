---
id: FEAT-148
title: "Image Transformation and Caching"
status: phase-2
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

### Phase 1: Core ✅ Implemented

- [x] `image(path)` builtin: auto-rotate (EXIF), strip metadata, cache, return hashed URL (original format preserved)
- [x] `image(path, options)` builtin: resize (width/height/fit), set quality, choose format
- [x] `image(path, style)` builtin: apply a named style (dict with transformation options)
- [x] `imageInfo(path)` builtin: return `{width, height, format, orientation}` dict
- [x] Disk cache at configurable location (default `./cache/images/`)
- [x] `/__img/{hash}.{ext}` handler serving cached images with immutable cache headers
- [x] Dev mode: no-cache headers, re-transform on source change
- [x] Security: path must be within handler root (same rules as `publicUrl()`)
- [x] Size limits: warn at 10MB source, reject at 50MB source
- [x] Supported input formats: JPEG, PNG, GIF, WebP (decode only)
- [x] Supported output formats: JPEG, PNG (WebP encoding deferred — see Design Decisions)
- [x] Default output format: **original** (JPEG in → JPEG out, PNG in → PNG out)

### Phase 2: Lovable

**Investigation complete** — see `work/reports/FEAT-148-phase2-investigation-results.md` and `work/design/DESIGN-image-transform-phase2.md`.

Phase 2 ships three features. Dominant color, smart crop, and WebP encoding are deferred to Phase 3 based on investigation findings (see rationale below each deferred item).

#### 2a. `imageSrcset()` — Responsive Image Generation

Generate multiple resized variants of a source image and return a dict suitable for building responsive `<img>` tags.

- [ ] `imageSrcset(path, style, widths)` builtin: generate responsive image variants
  - `path`: source image path (same rules as `image()`)
  - `style`: transform options dict (same keys as `image()` — `width`, `height`, `crop`, `quality`, `format`)
  - `widths`: array of target widths in pixels, e.g., `[400, 800, 1200]`
  - The `width` field in `style` is used as the base/default width for `src`; `widths` array generates the `srcset` variants
- [ ] Returns a dict: `{src: string, srcset: string, width: int, height: int}`
  - `src`: URL of the base variant (the `style.width` size, or the middle width if no `style.width`)
  - `srcset`: complete `srcset` attribute value with width descriptors, e.g., `"/__img/a.jpg 400w, /__img/b.jpg 800w, /__img/c.jpg 1200w"`
  - `width`: pixel width of the largest generated variant
  - `height`: pixel height of the largest generated variant (computed from aspect ratio)
- [ ] `imageSrcset(path, style, scales, "x")` density descriptor mode (optional 4th string arg):
  - `scales` is an array of density multipliers, e.g., `[1, 2, 3]`
  - Multiplied against `style.width` to produce pixel widths
  - Returns `srcset` with density descriptors: `"/__img/a.jpg 1x, /__img/b.jpg 2x, /__img/c.jpg 3x"`
- [ ] Widths exceeding source image dimensions are clamped (no upscaling), matching `image()` behavior
- [ ] Internally calls `image()` N times (once per width) — reuses existing registry, cache, and singleflight dedup
- [ ] `sizes` attribute is NOT generated — developer provides it (depends on CSS layout context)
- [ ] Error if `widths`/`scales` array is empty, not an array, or contains non-positive values

**Example usage in Parsley:**

```parsley
heroStyle = {width: 800, quality: 80}

resp = imageSrcset(@./hero.jpg, heroStyle, [400, 800, 1200])
<img
  src={resp.src}
  srcset={resp.srcset}
  sizes="(max-width: 600px) 100vw, 50vw"
  width={resp.width}
  height={resp.height}
/>
```

#### 2b. `imageBlur()` — Blur Placeholder (LQIP)

Generate a tiny blurred placeholder image as an inline data URI for progressive loading.

- [ ] `imageBlur(path)` builtin: returns a `data:image/jpeg;base64,...` string
  - `path`: source image path (same rules as `image()`)
  - Always returns a data URI — never a URL (the point is avoiding a network round-trip)
- [ ] Pipeline: resize to 20px wide (aspect ratio preserved) → Gaussian blur σ=10 → encode JPEG quality 20 → base64
  - Output size: ~600 bytes (~835 chars as data URI) — verified by investigation
  - Pipeline cost: ~3ms — negligible, one-time cached cost
- [ ] Result is cached (same content-addressed disk cache as `image()`)
- [ ] JPEG output only (alpha is discarded — appropriate for placeholder use)
- [ ] No edge feathering concern at LQIP scale — blur radius (30px) exceeds thumbnail dimensions, so kernel clamping has no visible effect (verified by investigation)
- [ ] Error if path is invalid, outside root, or not a supported image format

**Example usage in Parsley:**

```parsley
// Inline as a CSS background for progressive loading
<div
  style={"background-image: url(" + imageBlur(@./hero.jpg) + ")"}
  class="hero-placeholder"
>
  <img src={image(@./hero.jpg, {width: 1200})} loading=lazy />
</div>
```

#### 2c. Sharpen on Downscale

Apply a subtle unsharp mask when reducing image dimensions, recovering edge detail lost to resampling.

- [ ] New `sharpen` option on `image()` transform options: `{sharpen: true}` or `{sharpen: 0.5}`
  - `true` (boolean): apply default sigma of 0.5
  - Number (float): apply specified sigma (must be > 0)
  - `false` or omitted: no sharpening (default — opt-in only)
- [ ] Sharpening is applied after resize, only when the output dimensions are smaller than the source
- [ ] Uses `imaging.Sharpen(img, sigma)` from `disintegration/imaging`
  - Cost: ~1ms at 400×300, ~0.4ms at 200×150 — negligible (verified by investigation)
- [ ] `sharpen` value is included in cache key via `Canonical()` — different sharpen settings produce different cached variants
- [ ] Sharpen is independent of crop mode — works with `crop: "center"` or no crop
- [ ] Add `Sharpen` field to `TransformOptions` struct (type: `float64`, 0 = none)
- [ ] Update `ParseOptions()`, `Validate()`, `Canonical()` to handle `sharpen`
- [ ] `Validate()`: reject negative sigma values

#### 2d. `imageInfo()` Memory Cache

Cache `imageInfo()` results to avoid redundant disk reads in loops (e.g., gallery pages).

- [ ] Add `sync.Map` cache in the registry, keyed on absolute path + file modtime
  - Modtime check ensures cache invalidation when the source file changes
- [ ] Cache stores the `{width, height, format, orientation}` dict
- [ ] Dev mode: always check modtime before returning cached result
- [ ] `Clear()` method also clears the info cache

### Phase 3: Nice to Have

Carried forward from original spec, plus items deferred from Phase 2 based on investigation.

**Deferred from Phase 2 (with rationale):**

- [ ] Dominant color extraction in `imageInfo()`: `imageInfo(@./photo.jpg, {color: true})` → adds `color: "#2a4f3b"` to return dict
  - *Deferred: low user value; zero-dep approach (1×1 resize via `imaging`) is ready when needed; requires full image decode (gated behind opt-in param); depends on `imageInfo()` caching (Phase 2d) being in place*
  - Implementation: `imaging.Resize(img, 1, 1, imaging.Lanczos)` → read pixel → format as hex. Upgrade to `cenkalti/dominantcolor` (K-Means, Chromium port) if average-color quality is insufficient.
- [ ] Smart crop via focal point detection: `{crop: "smart"}`
  - *Deferred: only viable library (`muesli/smartcrop`, ~1.9k ⭐) is dormant (last human commit Jan 2022, last release 2019) with known quality issues. No alternative pure-Go libraries exist. Returns `image.Rectangle` — requires `imaging.Crop()` + `imaging.Resize()` (not compatible with `imaging.Fill()`). Evaluate when there is user demand.*
  - If adopted: vendor or fork `muesli/smartcrop`, replace its `nfnt/resize` dep with custom `imaging`-based resizer via its `Resizer` interface.
- [ ] WebP output encoding via `gen2brain/webp`
  - *Deferred: codebase is fully prepared (options parsing, validation, format normalization, quality defaults all handle "webp"). The only missing piece is the encoder call in `Encode()` (~4 lines). Binary size impact unmeasured — current basil is 35MB, pars is 27MB. Measure when `imageSrcset()` is shipping (WebP output matters most for responsive images).*

**Original Phase 3 items:**

- [ ] AVIF output format support
- [ ] SVG optimization (metadata stripping, minification)
- [ ] `basil cache clear` CLI subcommand
- [ ] Cache size limits / LRU eviction policy
- [ ] Color palette extraction in `imageInfo()`

## Design Decisions

### Phase 2 Design Decisions

#### `imageSrcset()` Returns a Dict, Not a String

**Decision**: `imageSrcset()` returns `{src, srcset, width, height}`, not just a `srcset` attribute string.

**Rationale**: A dict gives the developer everything needed to build a complete `<img>` tag — `src` for the fallback, `srcset` for responsive variants, and `width`/`height` for layout stability (preventing CLS). Parsley's native HTML composition makes it trivial to use dict fields in attributes. A bare string would require a separate call to get dimensions and the fallback `src`. Surveyed Next.js (`getImageProps()` returns a props object), Eleventy Image (returns metadata object by default), and Hugo (returns individual resource objects) — all provide structured data, not just a string.

#### `imageSrcset()` Uses Width Descriptors by Default

**Decision**: The primary mode uses width descriptors (`400w`, `800w`, `1200w`). Density descriptors (`1x`, `2x`) are available via an optional 4th argument.

**Rationale**: Width descriptors are the standard for responsive images — all three surveyed frameworks (Next.js, Eleventy, Hugo) default to or primarily use `w` descriptors. Density descriptors are a secondary mode for fixed-size images (icons, avatars). Next.js auto-selects based on whether `sizes` is present; we make it explicit with an optional `"x"` argument to keep the API predictable.

#### `imageSrcset()` Calls `image()` N Times Internally

**Decision**: `imageSrcset()` calls `image()` once per width in the scales array. No batch registry API.

**Rationale**: At typical scale counts (3–5 variants), the overhead of N separate `image()` calls is negligible — each goes through the registry's singleflight dedup and cache check. A batch API (single source load, multiple transforms) would save one image decode per additional variant, but the complexity isn't justified until profiling shows decode is a bottleneck. The existing `Load()`, `Transform()`, and `Encode()` functions in `transform.go` are already composable, so a `TransformBatch()` method could be added later without changing the public API.

#### `imageBlur()` Is a Separate Builtin, Not an Option on `image()`

**Decision**: Blur placeholder is a dedicated `imageBlur(path)` function, not `image(path, {blur: true})`.

**Rationale**: `image()` always returns a URL string (`/__img/{hash}.ext`). `imageBlur()` always returns a data URI string (`data:image/jpeg;base64,...`). These are fundamentally different return types. Overloading `image()` with an option that changes the return type would be surprising and break the contract. A separate function makes the behaviour obvious at the call site. The name `imageBlur` is consistent with the `image`/`imageInfo`/`imageSrcset` family.

#### Blur Parameters Are Fixed, Not Configurable

**Decision**: `imageBlur()` takes only a path — no size, sigma, or quality parameters.

**Rationale**: LQIP is a well-understood technique with a narrow sweet spot: 20px wide, heavy blur, low JPEG quality. Investigation confirmed the defaults produce ~600 byte payloads in ~3ms — there is nothing to tune. Exposing parameters would invite bikeshedding and produce worse results (too large = defeats the purpose, too small = artifacts). If a future use case needs configurable blur, it can be added as a separate feature.

#### Blur Edge Handling at LQIP Scale

**Decision**: No padding/crop technique is needed for the LQIP pipeline.

**Rationale**: `imaging.Blur()` uses kernel clamping (equivalent to edge replication) — it does not sample imaginary/transparent pixels beyond the image boundary. On opaque images (JPEG), there is no feathering. At LQIP scale (20×15px), the blur radius (30px from σ=10) exceeds the image dimensions, so the entire thumbnail becomes a near-uniform color field — edge truncation effects are undetectable (verified: zero deviation measured). For any future larger-scale blur feature, a 1× radius pad-then-crop technique eliminates edge bias completely (also verified). See `work/reports/FEAT-148-phase2-investigation-results.md` §2 for full measurements.

#### Sharpen Is Opt-In, Not Automatic

**Decision**: Sharpening on downscale requires `{sharpen: true}` — it is not applied automatically.

**Rationale**: Automatic sharpening could surprise users who expect pixel-identical output from deterministic transforms. Opt-in is safer and more predictable. The cost is trivial (~1ms), so there's no performance reason to gate it — the decision is purely about user expectations. `{sharpen: true}` uses a default sigma of 0.5 (industry standard for post-downscale web images); an explicit number like `{sharpen: 0.8}` overrides the default.

#### `sizes` Is Always User-Provided

**Decision**: `imageSrcset()` does not generate or accept a `sizes` attribute. The developer writes it directly in their template.

**Rationale**: The `sizes` attribute describes how wide the image will be at various viewport widths — this depends entirely on CSS layout, which the image system has no knowledge of. No surveyed framework (Next.js, Eleventy, Hugo) auto-generates `sizes`. Attempting to infer it would produce incorrect results and create a false sense of correctness.

---

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
    Width   int     // Target width (0 = auto/preserve aspect ratio)
    Height  int     // Target height (0 = auto/preserve aspect ratio)
    Crop    string  // "center", "smart" (Phase 3), or "" (no crop — fit within box)
    Quality int     // 1-100, 0 = format default (JPEG: 85, WebP: 80, PNG: lossless)
    Format  string  // "webp", "jpeg", "png", "original", or "" (default = "original")
    Sharpen float64 // Sharpen sigma, 0 = none (Phase 2: opt-in via {sharpen: true} → 0.5, or {sharpen: N})
}
```

**Phase 1 (implemented):** Width, Height, Crop ("center"), Quality, Format.
**Phase 2 (adding):** Sharpen.
**Phase 3 (planned):** Crop "smart" mode. Blur is handled by the separate `imageBlur()` builtin, not as a TransformOptions field.

### Builtin Signatures

#### `image(path)` → `string`
- Auto-rotate, strip metadata, cache in original format
- Returns `/__img/{hash}.{ext}` (extension matches source format)

#### `image(path, options_dict)` → `string`
- Apply specified transformations
- `options_dict` keys: `width`, `height`, `crop`, `quality`, `format`
- Phase 2 adds: `sharpen` (boolean `true` → sigma 0.5, or float for explicit sigma)
- Returns `/__img/{hash}.{ext}`

#### `image(path, style_dict)` → `string`  
- Same as above — a "style" is just a dict with the same keys
- No special style registration mechanism needed; styles are regular Parsley values

#### `imageSrcset(path, style, widths)` → `dict` *(Phase 2)*
- Generate multiple resized variants and return responsive image metadata
- `widths`: array of pixel widths, e.g., `[400, 800, 1200]`
- Returns `{src: string, srcset: string, width: int, height: int}`
- `srcset` uses width descriptors by default: `"/__img/a.jpg 400w, /__img/b.jpg 800w"`

#### `imageSrcset(path, style, scales, "x")` → `dict` *(Phase 2)*
- Density descriptor mode (4th arg is the string `"x"`)
- `scales`: array of density multipliers, e.g., `[1, 2, 3]`
- Multiplied against `style.width` to compute pixel widths
- `srcset` uses density descriptors: `"/__img/a.jpg 1x, /__img/b.jpg 2x"`

#### `imageBlur(path)` → `string` *(Phase 2)*
- Generate a tiny blurred placeholder image as an inline data URI
- Returns `data:image/jpeg;base64,...` (~600 bytes / ~835 chars)
- Pipeline: resize to 20px wide → Gaussian blur σ=10 → JPEG quality 20 → base64
- No options — parameters are fixed (see Phase 2 Design Decisions)

#### `imageInfo(path)` → `dict`
- Returns `{width: int, height: int, format: string, orientation: string}`
- `orientation`: `"landscape"`, `"portrait"`, or `"square"`
- Phase 2 adds: in-memory caching (keyed on path + modtime)
- Phase 3 adds: `color: string` (dominant color as hex, opt-in via `{color: true}`)
- Does NOT transform the image — reads metadata only

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

### Phase 2 Affected Components

- `server/images/options.go` — Add `Sharpen` field to `TransformOptions`, update `ParseOptions()`, `Validate()`, `Canonical()`
- `server/images/transform.go` — Apply `imaging.Sharpen()` after resize when `opts.Sharpen > 0`; add `GenerateBlurPlaceholder()` function (resize → blur → JPEG encode → base64)
- `server/images/registry.go` — Add `BlurPlaceholder(sourcePath)` method; add `sync.Map` info cache for `imageInfo()` memoization; add `TransformMultiple()` or have `imageSrcset` call `Transform()` N times
- `server/images/handler.go` — No changes expected (blur placeholders are data URIs, not served via handler)
- `pkg/parsley/evaluator/image.go` — Register `imageSrcset()` and `imageBlur()` builtins; update `image()` to pass `sharpen` option through to registry

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
- Plan (Phase 1): `work/plans/FEAT-148-plan.md`
- Design investigation (Phase 2): `work/design/DESIGN-image-transform-phase2.md`
- Investigation results (Phase 2): `work/reports/FEAT-148-phase2-investigation-results.md`
