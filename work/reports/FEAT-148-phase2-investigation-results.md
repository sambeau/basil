# FEAT-148 Phase 2 — Investigation Results

**Date:** 2026-06-28
**Status:** Complete
**Investigator:** AI Agent
**Source:** `work/design/DESIGN-image-transform-phase2.md` (investigation tasks)

---

## Table of Contents

1. [imageSrcset() — Framework Survey & API Design](#1-imagesrcset--framework-survey--api-design)
2. [Blur Placeholder — LQIP Prototyping](#2-blur-placeholder--lqip-prototyping)
3. [Dominant Color Extraction — Libraries & Techniques](#3-dominant-color-extraction--libraries--techniques)
4. [Smart Crop — Library Evaluation](#4-smart-crop--library-evaluation)
5. [Sharpen on Downscale — Tuning Results](#5-sharpen-on-downscale--tuning-results)
6. [WebP Encoding — Binary Size Baseline](#6-webp-encoding--binary-size-baseline)
7. [Current Architecture Assessment](#7-current-architecture-assessment)
8. [Recommendations](#8-recommendations)

---

## 1. imageSrcset() — Framework Survey & API Design

### Framework Comparison

| Feature | Next.js `<Image>` | Eleventy Image | Hugo |
|---|---|---|---|
| **Width specification** | Global config arrays (`deviceSizes` + `imageSizes`) | Per-call `widths` array | Per-call `.Resize "300x"` |
| **Default widths** | `[640, 750, 828, 1080, 1200, 1920, 2048, 3840]` + `[32, 48, 64, 96, 128, 256, 384]` | `["auto"]` (original only) | None |
| **Width descriptors (`w`)** | ✅ (when `sizes` present) | ✅ (always) | ✅ (manual) |
| **Density descriptors (`x`)** | ✅ (when no `sizes`) | ❌ | ✅ (manual) |
| **Descriptor auto-selection** | ✅ based on `sizes` prop | ❌ always `w` | N/A (manual) |
| **srcset generation** | Automatic | Automatic | Manual template loop |
| **sizes generation** | Manual (developer provides) | Manual (developer provides) | Manual |
| **API return type** | React component / props object (`getImageProps`) | Metadata object or HTML string | Single image resource object |
| **Multi-format (`<picture>`)** | Server-side content negotiation | ✅ `<picture>` with `<source>` per format | Manual |
| **Prevents upscaling** | Yes | Yes | No |

### Key Takeaways for Parsley

1. **Width descriptors are the primary mode.** All three frameworks default to `w` descriptors for responsive images. Density descriptors (`1x`/`2x`) are secondary, used only for fixed-size images.

2. **Explicit width arrays are the ergonomic sweet spot.** Eleventy's `widths: [200, 600, "auto"]` is the most composable pattern. Next.js's global config is less flexible. Hugo's fully-manual approach is too low-level.

3. **`"auto"` as a sentinel for original width** is useful (Eleventy pattern) — avoids forcing the user to know original dimensions.

4. **Returning a data object is more flexible than returning markup.** Next.js's `getImageProps()` and Eleventy's default metadata object both let callers build custom markup. Parsley already has native HTML composition, so data is the right level.

5. **`sizes` should be user-provided.** No framework auto-generates `sizes` — it depends on CSS layout context.

6. **Prevent upscaling by default** — both Next.js and Eleventy filter out widths larger than the source image.

### API Shape Recommendation

**Width-descriptor mode** (primary), with density-descriptor mode as an optional variant:

```parsley
// Width descriptors — the common case
imageSrcset(@./photo.jpg, {width: 800, quality: 80}, [400, 800, 1200])
// → {
//     src: "/__img/abc123.jpg",
//     srcset: "/__img/def456.jpg 400w, /__img/abc123.jpg 800w, /__img/ghi789.jpg 1200w",
//     width: 1200,
//     height: 900
//   }

// Density descriptors — for fixed-size images
imageSrcset(@./icon.jpg, {width: 64}, [1, 2, 3], "x")
// → {
//     src: "/__img/abc123.jpg",
//     srcset: "/__img/abc123.jpg 1x, /__img/def456.jpg 2x, /__img/ghi789.jpg 3x",
//     width: 64,
//     height: 64
//   }
```

**Return type: dict** with `{src, srcset, width, height}`. This gives Parsley templates full control over markup composition. The `sizes` attribute is the developer's responsibility.

**Internally:** Call `image()` N times (one per scale). The batch registry optimization is possible but not needed at typical scale counts (3–5 variants). See §7 for batch feasibility.

### Open Decision: Default Scales

If `scales` is omitted, use sensible defaults:
- Width mode: `[0.5x, 1x, 2x]` of the style's base width (i.e., `[400, 800, 1600]` for `width: 800`)
- Or a fixed set like `[640, 1024, 1920]`

Recommendation: require explicit scales for Phase 2; add smart defaults later based on usage patterns.

---

## 2. Blur Placeholder — LQIP Prototyping

### Measurements

Tested with `disintegration/imaging`: resize 800×600 source → 20×15px, blur, encode JPEG q=20, base64.

| Sigma | JPEG Bytes | Data URI Length | Pipeline Time |
|-------|-----------|-----------------|---------------|
| 5     | 620 B     | 851 chars       | ~3.5 ms       |
| 10    | 608 B     | 835 chars       | ~3.2 ms       |
| 15    | 605 B     | 831 chars       | ~3.1 ms       |

### Findings

- **All variants well under the 1 KB target** (~600 bytes). Higher sigma = slightly smaller (more uniform → better JPEG compression), but the difference is marginal.
- **Pipeline cost is negligible** — ~3 ms for resize + blur + encode. This is a one-time cached cost.
- **Sigma 10 is the sweet spot** — good visual blur for a placeholder with no meaningful size penalty vs sigma 5.

### Edge Feathering Investigation

**Question:** Does `imaging.Blur()` pull in imaginary/transparent pixels from beyond the image boundary, creating white or feathered edges?

**How `imaging.Blur()` handles edges:** The library clamps the Gaussian kernel window to the image bounds — it never samples beyond the edge. It then renormalizes the kernel weights so they sum to 1 over the clamped range. This is mathematically equivalent to infinite edge replication: the outermost pixel is treated as if it extends forever outward.

**Consequence:** On opaque images (JPEG), there is no transparency feathering. However, the clamped kernel over-weights the edge pixel relative to what a "true" continuation of the image content would contribute. On a gradient or non-uniform image, this biases edge pixels slightly toward their own color.

#### Measured edge deviation (red→blue horizontal gradient, 200×150, σ=10)

| Method | Left edge pixel (orig 255,0,0) | Max deviation | Mean deviation (outer 3px) |
|--------|-------------------------------|---------------|---------------------------|
| Direct blur (no padding) | (245, 0, 9) | 10 | 4.26 |
| Padded blur, 1× radius | (250, 0, 5) | 5 | 2.02 |
| Padded blur, 2× radius | (250, 0, 5) | 5 | 2.02 |

#### Pad-then-crop technique

Pre-padding the image with edge-replicated pixels before blurring, then cropping back to original dimensions, eliminates the kernel truncation bias:

```
radius = ceil(sigma × 3)
1. Pad image by `radius` pixels on each side (edge replication)
2. Blur the padded image
3. Crop back to original dimensions
```

**1× radius is sufficient.** The Gaussian kernel extends `ceil(σ × 3)` pixels. With 1× radius of padding, the kernel has full "runway" at every edge pixel — the clamping effect is completely eliminated. 2× radius produced identical results (no additional improvement).

#### Impact at LQIP thumbnail scale

At 20×15 pixels (the LQIP target size), edge deviation was **zero** for both direct and padded blur. At that scale, σ=10 is so large relative to the image dimensions that the entire image becomes a near-uniform color field — edge truncation effects are lost in the overall smoothing.

#### Transparent images (PNG with alpha)

For images with transparent edges, `imaging.Blur()` uses alpha-premultiplied weighting (`RGB × alpha × kernel_weight`). This causes opacity to feather inward from transparent regions. Padding with transparent pixels would not help here (it would make it worse). However, since the LQIP pipeline encodes to JPEG (which discards alpha), this is a non-issue for the intended use case.

#### Recommendation

- **For LQIP (20px thumbnails):** No padding needed — deviation is zero at this scale.
- **For any future larger-scale blur feature:** Use the 1× radius pad-then-crop technique. The cost is modest (blurring a slightly larger image) and eliminates the edge bias entirely.
- **Implementation note:** If padding is added later, gate it behind a size threshold (e.g., only pad when `min(width, height) > radius × 2`) to avoid the overhead on tiny thumbnails where it provides no benefit.

### API Decision: Data URI vs URL

**Recommendation: data URI.** The whole point of LQIP is avoiding a network round-trip. A URL defeats the purpose.

### API Decision: Separate Function vs Option

Three options were evaluated:

| Approach | Pros | Cons |
|---|---|---|
| `image(@./photo.jpg, {blur: true})` → data URI | One function | Changes return type based on option (surprising) |
| `image(@./photo.jpg, {blur: true})` → URL | Consistent return type | Defeats LQIP purpose |
| `imageBlur(@./photo.jpg)` → data URI | Clear contract, always returns data URI | New builtin |

**Recommendation: `imageBlur(@./photo.jpg)`** — a dedicated function with a clear contract. It always returns a `data:image/jpeg;base64,...` string. No parameters needed beyond the path (size/blur/quality are internal defaults: 20px wide, sigma 10, JPEG q20).

Alternatively, add a `blur` field to `imageInfo()` return dict, computed lazily: `imageInfo(@./photo.jpg).blur` → data URI string. This is elegant but conflates metadata with image generation. **Defer this decision until `imageInfo()` caching is resolved** (see §7).

---

## 3. Dominant Color Extraction — Libraries & Techniques

### Library Survey

| Library | Stars | Last Commit | Pure Go | Algorithm | API | Verdict |
|---|---|---|---|---|---|---|
| `cenkalti/dominantcolor` | 126 | Jul 2024 | ✅ (dep: `x/image`) | K-Means (Chromium port) | `Find(img) color.RGBA` | **Top pick** |
| `marekm4/color-extractor` | 131 | Jul 2023 | ✅ (zero deps) | Fixed 8-bucket RGB octant | `ExtractColors(img) []color.Color` | Fast but coarse |
| `EdlinOrg/prominentcolor` | 183 | Feb 2022 | Mostly (uses `nfnt/resize`) | K-Means++ | `Kmeans(img) []ColorItem` | Stale, heavy deps |
| `joshdk/quantize` | 40 | Nov 2017 | ✅ | MMCQ Median Cut | `Image(img, depth) []color.Color` | Abandoned |
| `RobCherry/vibrant` | 27 | ~2017 | ✅ | Android Palette port | Material Design swatches | Wrong tool (extracts Vibrant/Muted, not dominant) |
| `esimov/colorquant` | 91 | Jan 2021 | ✅ | Leptonica-based | Produces paletted images | Wrong tool (image-to-image, not extraction) |

### Resize-to-1×1 Technique

Tested `imaging.Resize(img, 1, 1, imaging.Lanczos)` on known inputs:

| Input | Expected | Got | Match |
|-------|----------|-----|-------|
| Solid red | `#ff0000` | `#ff0000` | ✅ Exact |
| 75% blue + 25% green | ~`#003ebf` | `#0032bf` | ✅ Close |
| Black→white gradient | ~`#808080` | `#7f7f7f` | ✅ Within rounding |

**Pros:** Zero new dependencies, one line of code, deterministic, fast.
**Cons:** Produces a weighted average, not a true dominant color. Multi-region images (60% blue sky + 40% green grass) yield teal/cyan rather than blue. K-Means would correctly return blue.

### Recommendation

**Start with 1×1 resize** (zero deps, already have `imaging`). It's sufficient for the primary use case: generating a CSS `background-color` placeholder. If users report quality issues with diverse images, upgrade to `cenkalti/dominantcolor` later — it has the cleanest API (`Find(img) color.RGBA`) and is ported from Chromium's battle-tested algorithm.

### Opt-In vs Always Compute

**Recommendation: opt-in.** Currently `imageInfo()` reads only headers (`image.DecodeConfig`). Adding color extraction requires full image decode — a significant cost increase. Use `imageInfo(@./photo.jpg, {color: true})` to gate it, or always compute if `imageInfo()` gets a memory cache (see §7).

---

## 4. Smart Crop — Library Evaluation

### `muesli/smartcrop`

| Attribute | Value |
|---|---|
| Stars | ~1,900 |
| License | MIT |
| Last human commit | Jan 2022 (~3.5 years dormant) |
| Last release | v0.3.0 (April 2019 — 6+ years old) |
| Open issues | 6 (oldest from 2014, some about quality) |
| Pure Go | ✅ (deps: `nfnt/resize`, `golang.org/x/image`) |
| Algorithm | Edge detection + skin tone + saturation + rule-of-thirds scoring |
| API | `analyzer.FindBestCrop(img, w, h) (image.Rectangle, error)` |

**Algorithm details:** Pre-scales image to 400px, runs Laplacian edge detection on CIE lightness, detects skin tones and saturation, generates candidate crops via brute-force grid search, scores with weighted sum + rule-of-thirds bonus.

### Alternative Libraries

| Library | Stars | Status | Approach | Verdict |
|---|---|---|---|---|
| `esimov/caire` | ~10,500 | Active (v1.5.0, Apr 2025) | Seam carving | **Not a fit** — modifies image, doesn't return crop rect. Heavy deps (GUI framework, face detection) |
| `region23/batch_smartcrop` | 5 | Dead (2015) | Wraps `muesli/smartcrop` | Dead |
| `titpetric/smartcrop` | 3 | Dead (2019) | Wraps `muesli/smartcrop` | Dead |

**`muesli/smartcrop` is the only viable pure-Go smart crop library.** There are no alternatives.

### Integration with `disintegration/imaging`

⚠️ **`imaging.Fill()` is NOT compatible** with externally-computed crop rectangles. `Fill()` uses its own anchor system (`imaging.Center`, `imaging.Top`, etc.) internally.

Integration pattern would be:

```go
// 1. Compute focal point
topCrop, _ := analyzer.FindBestCrop(img, targetW, targetH)

// 2. Extract the region
cropped := imaging.Crop(img, topCrop)

// 3. Scale to exact target size
final := imaging.Resize(cropped, targetW, targetH, imaging.Lanczos)
```

The `nfnt/resize` dependency can be replaced with a custom resizer using the library's `Resizer` interface — avoiding the unmaintained `nfnt/resize`:

```go
type imagingResizer struct{}

func (r imagingResizer) Resize(img image.Image, width, height uint) image.Image {
    return imaging.Resize(img, int(width), int(height), imaging.Lanczos)
}

analyzer := smartcrop.NewAnalyzer(imagingResizer{})
```

### Risks

- **Dormant maintenance** — pin to commit SHA or vendor
- **Known quality issues** — issue #55 ("results differ from JS lib"), issue #33 ("sometimes gives wrong result")
- **`nfnt/resize` unmaintained** — mitigated by supplying custom resizer (see above)

### Recommendation

**Defer smart crop to Phase 3 or later.** The only library is dormant with known quality issues. The feature is useful but risky — invest engineering time in `imageSrcset()` and blur placeholders first. If smart crop is needed, vendor `muesli/smartcrop` with the custom `imaging`-based resizer.

---

## 5. Sharpen on Downscale — Tuning Results

### Measurements

Tested `imaging.Sharpen(img, sigma)` at three sigma values on two downscale ratios:

| Downscale | Sigma | Sharpen Time |
|-----------|-------|-------------|
| 400×300 (50%) | 0.3 | 1.26 ms |
| 400×300 (50%) | 0.5 | 1.19 ms |
| 400×300 (50%) | 0.8 | 1.30 ms |
| 200×150 (75%) | 0.3 | 0.37 ms |
| 200×150 (75%) | 0.5 | 0.38 ms |
| 200×150 (75%) | 0.8 | 0.38 ms |

### Findings

- **Sharpen cost scales with pixel count** — ~1.2 ms for 400×300, ~0.4 ms for 200×150. Very cheap and negligible vs the overall transform cost (100–300ms).
- **Sigma value has no measurable impact on performance** — only on visual output.
- **Sigma 0.5 is the recommended default** — enough to recover edge detail lost to Lanczos resampling without introducing halos. Standard recommendation for post-downscale sharpening.

### Decisions

| Question | Decision | Rationale |
|---|---|---|
| Automatic vs opt-out? | **Automatic** (opt-out via `{sharpen: false}`) | Downscale sharpening is universally-accepted best practice; images should "just work" and look good without the developer needing to know about it. Aligns with Basil's "zero-config by default, overridable when needed" principle. |
| Threshold? | **Any downscale** (when both source dimensions > target) | The cost is negligible; simplifies logic |
| Default sigma? | **0.5** | Industry standard for web images |
| Interaction with explicit values? | Omitted or `true` → sigma 0.5; `{sharpen: 0.8}` → sigma 0.8; `{sharpen: false}` → disabled | Boolean/omitted uses default, number overrides, false disables |

### Implementation Path

This is simple enough to go straight to implementation:
1. Add `Sharpen` (float64) and `SharpenDisabled` (bool) fields to `TransformOptions`
2. Registry applies default sigma 0.5 when `Sharpen == 0 && !SharpenDisabled` (same pattern as default quality)
3. In `Transform()`, after resize, if `opts.Sharpen > 0 && !opts.SharpenDisabled` and image was downscaled, apply `imaging.Sharpen(img, opts.Sharpen)`
4. `ParseOptions()`: `sharpen: false` → `SharpenDisabled=true`; `sharpen: true` or omitted → use default; `sharpen: N` → explicit sigma
5. Update `Canonical()`, `Validate()`

---

## 6. WebP Encoding — Binary Size Baseline

### Current State

| Binary | Size |
|--------|------|
| `basil` | 35 MB (37,089,442 bytes) |
| `pars` | 27 MB (28,094,162 bytes) |

- **WebP decode: ✅ supported** via `golang.org/x/image/webp` (registered as blank import)
- **WebP encode: ❌ not supported** — explicit error placeholder in `Encode()` function
- **`gen2brain/webp`: not a dependency** — not in `go.mod` or `go.sum`
- **Codebase is fully prepared** for WebP encoding — options parsing, validation, format normalization, quality defaults (80), config fields, and cache paths all handle `"webp"` already

### What Adding WebP Encoding Would Require

1. `go get github.com/gen2brain/webp`
2. Replace the error stub in `Encode()` `case "webp":` (~4 lines of code)
3. Remove the WebP→JPEG fallback in `Process()`
4. Update `TestEncode_WebPOutputError` to become a success test

### Estimated Binary Size Impact

`gen2brain/webp` is a pure-Go WebP encoder (no CGo). Typical pure-Go WebP encoders add **1–3 MB** to binaries. Actual measurement requires adding the dependency and rebuilding.

### Recommendation

**Defer WebP encoding measurement to when `imageSrcset()` is implemented.** The srcset feature is where WebP output matters most (serving WebP to supporting browsers). Measure the actual impact at that point. The codebase is already prepared, so adding it is a small diff.

---

## 7. Current Architecture Assessment

### TransformOptions (Current)

```go
type TransformOptions struct {
    Width   int    // Target width (0 = auto)
    Height  int    // Target height (0 = auto)
    Crop    string // "center" or "" (no crop)
    Quality int    // 1-100, 0 = format default
    Format  string // "webp", "jpeg", "png", or ""
}
```

### Adding New Crop Modes

**Fits cleanly.** `Crop` is already a string enum. Adding `"smart"`, `"top"`, `"bottom"`, etc. requires:
- Extend `Validate()` allowed values
- Add new branch in `Transform()` alongside the `crop == "center"` check
- `Canonical()` already includes crop in cache key — different modes auto-produce different cache keys
- `imaging.Fill()` already supports `imaging.Top`, `imaging.Bottom`, `imaging.Left`, `imaging.Right`, etc. — basic directional crops are trivial

For smart crop specifically, a different code path is needed (see §4).

### imageInfo() — Header-Only Today

`imageInfo()` uses `image.DecodeConfig()` — reads only image headers, does not decode pixels. Adding dominant color or blur would require full decode, making caching important.

### Batch Transform Feasibility

**Not currently supported, but straightforward to add.** Key observations:
- Each `image()` call loads from disk independently — 5 variants = 5 full decodes
- `Load()`, `Transform()`, and `Encode()` in `transform.go` are already separate, composable functions
- A `TransformBatch()` could call `Load()` once, then loop over `Transform()` + `Encode()` per variant
- The saved cost is disk I/O + decode time per additional variant (significant for large images)
- **Verdict:** Not needed for Phase 2 launch (3–5 variants is fine with N×`Transform()`). Add batch if profiling shows decode is a bottleneck.

### imageInfo() Caching

**Recommendation: add `sync.Map` cache keyed on absolute path + modtime.** This is low-risk, low-complexity, and becomes important if dominant color or blur are added to `imageInfo()`. Without caching, calling `imageInfo()` in a gallery loop decodes headers N times. With color extraction, it would decode the full image N times.

---

## 8. Recommendations

### Phase 2 Scope (Recommended)

Ship these three items in Phase 2:

| Priority | Item | Complexity | Ready to Implement? |
|----------|------|-----------|-------------------|
| 1 | `imageSrcset()` | High | Needs API proposal (design decisions resolved above) |
| 2 | `imageBlur()` / blur placeholder | Medium | Yes — parameters determined (20px, σ10, q20, data URI) |
| 3 | Sharpen on downscale | Low | Yes — parameters determined (automatic, σ0.5 default, opt-out via `{sharpen: false}`) |

Defer to Phase 3:

| Item | Reason |
|------|--------|
| Dominant color | Low user value; start with 1×1 resize (zero deps), upgrade to `cenkalti/dominantcolor` if needed |
| Smart crop | Only library (`muesli/smartcrop`) is dormant with known quality issues; evaluate when there's user demand |
| WebP encoding | Measure binary impact when `imageSrcset()` is ready; the codebase is prepared |

### Key Design Decisions Resolved

| Decision | Resolution |
|---|---|
| `imageSrcset()` return type | Dict: `{src, srcset, width, height}` |
| `imageSrcset()` descriptor mode | Width descriptors by default; density via optional 4th arg |
| `imageSrcset()` internal impl | N×`image()` calls (no batch API needed yet) |
| Blur placeholder return type | Data URI (`data:image/jpeg;base64,...`) |
| Blur placeholder API | Separate `imageBlur()` function (clear contract, always data URI) |
| Blur parameters | 20px wide, sigma 10, JPEG quality 20 (~600 bytes output) |
| Sharpen mode | Automatic on downscale (σ0.5 default); `{sharpen: false}` to disable, `{sharpen: N}` to override |
| Sharpen threshold | Any downscale (cost is negligible) |
| Dominant color approach | 1×1 resize (zero deps) initially; `cenkalti/dominantcolor` upgrade path |
| Dominant color opt-in | Opt-in parameter on `imageInfo()` |
| Smart crop library | `muesli/smartcrop` (only option), but deferred |

### Next Steps

1. **Write API proposals** for `imageSrcset()` and `imageBlur()` with Parsley template examples
2. **Update `work/specs/FEAT-148.md`** Phase 2 section with concrete acceptance criteria based on these findings
3. **Create `work/plans/FEAT-148-phase2-plan.md`** implementation plan
4. **Implement sharpen first** (lowest complexity, fastest to ship, validates the option-extension pattern)
5. **Implement `imageBlur()`** next (parameters fully determined, simple pipeline)
6. **Implement `imageSrcset()`** last (most complex, shapes the API surface)
7. **Add `imageInfo()` caching** as a cross-cutting improvement before dominant color is added