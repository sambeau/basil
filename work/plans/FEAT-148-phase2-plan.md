---
id: PLAN-129
feature: FEAT-148
title: "Implementation Plan for Image Transform Enhancements (Phase 2)"
status: complete
created: 2026-06-28
---

# Implementation Plan: FEAT-148 Phase 2 — Image Transform Enhancements

## Overview

Phase 2 adds four capabilities to the existing image transformation system: automatic sharpening on downscale, blur placeholder generation (`imageBlur()`), responsive image generation (`imageSrcset()`), and in-memory caching for `imageInfo()`. These build on the Phase 1 foundation in `server/images/` and `pkg/parsley/evaluator/image.go`.

**Investigation complete.** All design decisions are resolved — see `work/reports/FEAT-148-phase2-investigation-results.md` and `work/design/DESIGN-image-transform-phase2.md`.

**Estimated total effort:** ~600 lines of implementation + ~500 lines of tests.

## Design Decisions Driving This Plan

1. **Sharpen is automatic on downscale** (σ=0.5 default, opt-out via `{sharpen: false}`, override via `{sharpen: N}`).
2. **`imageBlur()` is a separate builtin** — always returns `data:image/jpeg;base64,...`, never a URL. Fixed parameters: 20px wide, σ=10, JPEG q=20.
3. **`imageSrcset()` returns a dict** `{src, srcset, width, height}` — width descriptors by default, density descriptors via optional 4th arg `"x"`.
4. **`imageSrcset()` calls `image()` N times** — no batch API needed at typical scale counts (3–5 variants).
5. **`imageInfo()` gets a `sync.Map` cache** keyed on absolute path + modtime.
6. **Dominant color, smart crop, and WebP encoding deferred to Phase 3.**

## Current Code Reference

| File | Key Structures | Phase 2 Changes |
|------|---------------|-----------------|
| `server/images/options.go` | `TransformOptions{Width, Height, Crop, Quality, Format}`, `ParseOptions()`, `Validate()`, `Canonical()`, `CacheKey()` | Add `Sharpen float64`, `SharpenDisabled bool`; update `ParseOptions`, `Validate`, `Canonical` |
| `server/images/transform.go` | `Load()`, `Transform()`, `Encode()`, `Process()`, `GetInfo()`, `SourceHash()`, `ImageInfo` | Add `imaging.Sharpen()` call after resize in `Transform()`; add `GenerateBlurPlaceholder()` |
| `server/images/registry.go` | `Registry{byHash, cache, group, ...}`, `Transform()`, `doTransform()`, `Info()`, `Clear()` | Add `BlurPlaceholder()` method; add `sync.Map` info cache; apply default sharpen σ=0.5 |
| `server/images/handler.go` | `Handler`, `ServeHTTP()` | No changes (blur returns data URI, not served via handler) |
| `pkg/parsley/evaluator/image.go` | `evalImage()`, `evalImageInfo()`, path helpers | Add `evalImageSrcset()`, `evalImageBlur()`; pass `sharpen` option through to registry |

## Prerequisites

- [x] Phase 1 implementation complete and tested
- [x] Investigation complete — all design decisions resolved
- [x] Spec updated with Phase 2 acceptance criteria (`work/specs/FEAT-148.md`)
- [ ] Feature branch created: `feat/FEAT-148-phase2`

## Tasks

### Task 1: Sharpen on Downscale — Options

**Files**: `server/images/options.go`
**Estimated effort**: Small (~40 lines)

Extend `TransformOptions` and its parsing/validation/canonicalization to support the `sharpen` option.

Steps:
1. Add fields to `TransformOptions`:
   ```go
   Sharpen         float64 // Sharpen sigma (0 = use default, >0 = explicit)
   SharpenDisabled bool    // Explicitly disabled via {sharpen: false}
   ```
2. Update `ParseOptions()` to handle the `sharpen` key:
   - `false` (boolean) → `SharpenDisabled = true`
   - `true` (boolean) or omitted → leave both zero (registry applies default)
   - Number (float/int) → `Sharpen = N`
3. Update `Validate()`:
   - Reject negative sigma: `if t.Sharpen < 0 { return error }`
   - `SharpenDisabled` and `Sharpen > 0` are mutually exclusive — reject if both set
4. Update `Canonical()` to include sharpen state:
   - Disabled: `|s=off`
   - Default (zero, not disabled): `|s=0` (registry will fill in 0.5)
   - Explicit: `|s=0.8`

Tests:
- Parse `{sharpen: false}` → `SharpenDisabled = true`
- Parse `{sharpen: true}` → both zero (default behavior)
- Parse `{sharpen: 0.8}` → `Sharpen = 0.8`
- Parse `{}` (omitted) → both zero
- Validate rejects `{sharpen: -1}`
- Canonical produces different keys for disabled vs default vs explicit

---

### Task 2: Sharpen on Downscale — Transform & Registry

**Files**: `server/images/transform.go`, `server/images/registry.go`
**Estimated effort**: Small (~30 lines)

Apply `imaging.Sharpen()` after resize when the image was downscaled.

Steps:
1. In `transform.go` `Transform()`: after resize, if `opts.Sharpen > 0` and the output is smaller than the source in both dimensions, call `imaging.Sharpen(img, opts.Sharpen)`
2. In `registry.go` `Transform()`: after `ParseOptions` and before `Validate`, apply default sharpen:
   ```go
   if parsed.Sharpen == 0 && !parsed.SharpenDisabled {
       parsed.Sharpen = 0.5 // default sigma for downscale sharpening
   }
   ```
3. The sharpen value flows through `Canonical()` → `CacheKey()` — different sharpen settings automatically produce different cache entries

Tests:
- Transform with sharpen: verify output differs from unsharpened (pixel comparison or hash)
- Transform without downscale: verify sharpen is NOT applied even if sigma is set
- Registry applies default σ=0.5 when sharpen is not specified
- Registry does not apply sharpen when `SharpenDisabled = true`
- Cache key differs between sharpened and unsharpened variants

---

### Task 3: Blur Placeholder — `GenerateBlurPlaceholder()`

**Files**: `server/images/transform.go`
**Estimated effort**: Small (~40 lines)

Add the core LQIP pipeline as a standalone function.

Steps:
1. Add function:
   ```go
   func GenerateBlurPlaceholder(sourcePath string) (string, error)
   ```
2. Pipeline:
   - `Load(sourcePath)` — full image decode
   - `imaging.Resize(img, 20, 0, imaging.Lanczos)` — 20px wide, preserve aspect
   - `imaging.Blur(resized, 10)` — Gaussian blur σ=10
   - `Encode(blurred, "jpeg", 20)` — JPEG quality 20
   - Base64 encode → return `"data:image/jpeg;base64," + encoded`
3. No edge padding needed at LQIP scale (verified by investigation)

Tests:
- Output starts with `"data:image/jpeg;base64,"`
- Output length is reasonable (~800–900 chars)
- Decoding the base64 payload produces a valid JPEG
- Error on non-existent path
- Error on unsupported format (e.g., SVG)

---

### Task 4: Blur Placeholder — Registry & Builtin

**Files**: `server/images/registry.go`, `pkg/parsley/evaluator/image.go`
**Estimated effort**: Medium (~80 lines)

Wire blur into the registry (with caching) and expose as a Parsley builtin.

Steps:
1. In `registry.go`, add `BlurPlaceholder(sourcePath string) (string, error)`:
   - Compute cache key: `SHA256(sourceHash + "|blur")[:16]`
   - Check disk cache (store the data URI string as a text file, or store the JPEG bytes and reconstruct data URI on read)
   - On miss: call `GenerateBlurPlaceholder()`, write result to cache
   - Use `singleflight` to deduplicate concurrent requests for the same source
2. Add `BlurPlaceholder` method to the `ImageRegistrar` interface (or duck-type via `map[string]any` like existing methods)
3. In `image.go`, add `NewImageBlurBuiltin()` → `evalImageBlur()`:
   - 1 arg: path (same resolution + security as `image()`)
   - Call `env.ImageRegistry.BlurPlaceholder(absPath)`
   - Return the data URI string
4. Register in the evaluator's builtin table

Tests:
- `imageBlur(@./photo.jpg)` returns data URI string
- Same path called twice → cache hit (second call is faster or identical result)
- Security: path outside root → error
- Missing registry → clear error
- Invalid arg count → arity error

---

### Task 5: `imageSrcset()` — Builtin Implementation

**Files**: `pkg/parsley/evaluator/image.go`, `server/images/registry.go`
**Estimated effort**: Large (~150 lines)

The most complex task. Generates multiple image variants and returns a dict.

Steps:
1. In `image.go`, add `NewImageSrcsetBuiltin()` → `evalImageSrcset()`:
   - 3–4 args: `(path, style, widths)` or `(path, style, scales, "x")`
   - `path`: resolve + security check (same as `image()`)
   - `style`: Parsley dict → `map[string]any` (same as `image()` options)
   - `widths`/`scales`: Parsley array → `[]int`
   - 4th arg `"x"` (optional string): switches to density descriptor mode

2. Width descriptor mode (default):
   - For each width in `widths`: call `image(path, style_with_width)` where the style's `width` is overridden
   - Get image info for aspect ratio calculation
   - Build srcset string: `"/__img/a.jpg 400w, /__img/b.jpg 800w, /__img/c.jpg 1200w"`
   - `src`: the variant matching `style.width` (or middle width if no base width)
   - `width`/`height`: dimensions of the largest variant

3. Density descriptor mode (`"x"` 4th arg):
   - Requires `style.width` to be set (error otherwise — need a base width to multiply)
   - For each scale in `scales`: compute `pixel_width = style.width * scale`
   - Call `image()` for each computed width
   - Build srcset string: `"/__img/a.jpg 1x, /__img/b.jpg 2x, /__img/c.jpg 3x"`
   - `src`: the 1x variant
   - `width`/`height`: dimensions of the 1x variant

4. Clamp widths exceeding source dimensions (no upscaling):
   - Call `imageInfo()` to get source dimensions
   - Filter out widths > source width
   - Deduplicate (clamped widths may collapse)

5. Validation:
   - `widths`/`scales` must be a non-empty array of positive numbers
   - Density mode requires `style.width` to be set
   - 4th arg, if present, must be the string `"x"`

6. Return Parsley dict:
   ```
   {src: "/__img/abc.jpg", srcset: "/__img/... 400w, ...", width: 1200, height: 900}
   ```

7. Register in the evaluator's builtin table

Tests:
- Width descriptor mode: 3 widths → dict with correct `src`, `srcset` (3 entries), `width`, `height`
- Density descriptor mode: `[1, 2, 3]` + `"x"` → dict with `1x, 2x, 3x` descriptors
- Upscale clamping: width > source → clamped to source width, deduped
- Empty widths array → error
- Non-positive width → error
- Density mode without `style.width` → error
- Invalid 4th arg → error
- Security: path outside root → error
- Missing registry → clear error

---

### Task 6: `imageInfo()` Memory Cache

**Files**: `server/images/registry.go`
**Estimated effort**: Small (~40 lines)

Add in-memory caching for `imageInfo()` results to avoid redundant disk reads.

Steps:
1. Add field to `Registry`:
   ```go
   infoCache sync.Map // key: "absPath|modtime" → value: map[string]any
   ```
2. In `Info()`:
   - `os.Stat(sourcePath)` to get modtime
   - Build cache key: `absPath + "|" + modtime.UnixNano()`
   - Check `infoCache.Load(key)` → return cached result if found
   - On miss: call `GetInfo()`, store result, return
3. In `Clear()`: add `r.infoCache = sync.Map{}` (or `Range` + `Delete`)
4. Dev mode: always check modtime (already implicit in the key — if file changes, modtime changes, key misses)

Tests:
- Second call for same path returns cached result (verify with mock or timing)
- File modification (different modtime) → cache miss
- `Clear()` purges the info cache
- Concurrent access is safe (sync.Map handles this)

---

### Task 7: Evaluator Registration & Integration

**Files**: `pkg/parsley/evaluator/image.go`, `pkg/parsley/evaluator/evaluator.go` (or wherever builtins are registered)
**Estimated effort**: Small (~30 lines)

Wire the new builtins into the evaluator so they're available in Parsley code.

Steps:
1. Ensure `NewImageBlurBuiltin()` and `NewImageSrcsetBuiltin()` are registered alongside existing `NewImageBuiltin()` and `NewImageInfoBuiltin()`
2. Verify the `ImageRegistrar` interface (or duck-type pattern) supports the new `BlurPlaceholder()` method — if the existing pattern uses `map[string]any` at the interface boundary, add the new method in the same style
3. Verify server integration: `server/server.go`, `server/handler.go`, `server/api.go` pass the registry to the evaluator environment — the new builtins should pick it up automatically if the registry already satisfies the interface

Tests:
- Integration test: `.pars` template using `imageBlur()` renders data URI
- Integration test: `.pars` template using `imageSrcset()` renders dict with expected keys
- Integration test: `image()` with `{sharpen: false}` produces a valid URL
- All existing image tests still pass

---

## Implementation Order

The tasks have these dependencies:

```
Task 1 (Sharpen Options) ──► Task 2 (Sharpen Transform+Registry)
                                          │
Task 3 (Blur Pipeline) ──► Task 4 (Blur Registry+Builtin)
                                          │
                                          ▼
                              Task 5 (imageSrcset Builtin)
                                          │
Task 6 (imageInfo Cache) ────────────────┤
                                          ▼
                              Task 7 (Registration & Integration)
```

**Recommended order** (sharpen first — lowest complexity, validates the option-extension pattern):

1. **Task 1** — Sharpen options (extend `TransformOptions`)
2. **Task 2** — Sharpen transform + registry (apply after resize)
3. **Task 3** — Blur pipeline (`GenerateBlurPlaceholder`)
4. **Task 4** — Blur registry + builtin (`imageBlur()`)
5. **Task 6** — `imageInfo()` memory cache (independent, can be done in parallel with Tasks 3–4)
6. **Task 5** — `imageSrcset()` builtin (most complex, benefits from sharpen + cache being in place)
7. **Task 7** — Registration and integration wiring

**Commit points:**
- After Tasks 1+2: sharpen feature complete, all tests pass
- After Tasks 3+4: blur feature complete, all tests pass
- After Task 6: info cache complete, all tests pass
- After Tasks 5+7: srcset feature complete, full integration, all tests pass

## Task Order & Commits

| Commit | Tasks | Message | Gate |
|--------|-------|---------|------|
| 1 | 1, 2 | `feat(images): add automatic sharpen on downscale` | `go test ./server/images/... ./pkg/parsley/...` |
| 2 | 3, 4 | `feat(images): add imageBlur() LQIP builtin` | `go test ./server/images/... ./pkg/parsley/...` |
| 3 | 6 | `feat(images): add imageInfo() memory cache` | `go test ./server/images/...` |
| 4 | 5, 7 | `feat(images): add imageSrcset() responsive image builtin` | `go test ./...` |
| 5 | — | `test(images): run benchmarks and verify no regressions` | `make bench-compare` |

## Estimated Total Effort

| Component | Lines (est.) |
|-----------|-------------|
| `server/images/options.go` additions | ~40 |
| `server/images/transform.go` additions | ~70 |
| `server/images/registry.go` additions | ~120 |
| `pkg/parsley/evaluator/image.go` additions | ~200 |
| Integration wiring | ~20 |
| **Tests** | ~500 |
| **Total** | **~950** |

## Validation Checklist

- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `make build`
- [ ] Benchmarks checked: `make bench-compare` — no regressions > 5%
- [ ] Manual test: `image(@./photo.jpg, {sharpen: false})` returns valid URL
- [ ] Manual test: `image(@./photo.jpg)` with downscale applies sharpen by default
- [ ] Manual test: `imageBlur(@./photo.jpg)` returns `data:image/jpeg;base64,...`
- [ ] Manual test: `imageSrcset(@./photo.jpg, {width: 800}, [400, 800, 1200])` returns dict with `src`, `srcset`, `width`, `height`
- [ ] Manual test: `imageSrcset(@./icon.jpg, {width: 64}, [1, 2, 3], "x")` returns density descriptors
- [ ] Cache key isolation: sharpened vs unsharpened variants produce different cached files
- [ ] Dev mode: modify source image, verify blur placeholder regenerates

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| `imageSrcset()` argument parsing complexity | Medium | Medium | Start with width descriptors only; add density mode as second step within Task 5 |
| Sharpen alters existing cached images | Low | Low | Sharpen state is in cache key — old cache entries remain valid (no sharpen in key = old behavior). New transforms get sharpened variants with new cache keys. |
| Blur placeholder size exceeds expectations | Low | Low | Investigation measured ~600 bytes consistently. Pipeline parameters are fixed. |
| `sync.Map` memory growth for info cache | Low | Low | Each entry is tiny (~100 bytes). `Clear()` resets. Bounded by number of unique images in project. |

## Deferred Items

Items remaining for Phase 3 (not in scope for this plan):

| Item | Reason | Ready State |
|------|--------|-------------|
| Dominant color extraction | Low user value; 1×1 resize approach ready when needed | Investigation complete, approach documented |
| Smart crop (`{crop: "smart"}`) | Only library (`muesli/smartcrop`) is dormant with quality issues | Investigation complete, integration path documented |
| WebP output encoding | Codebase prepared (~4 lines to add); measure binary impact when `imageSrcset()` is shipping | Baseline binary sizes recorded |
| Batch registry API | N×`Transform()` is fine for 3–5 variants; optimize if profiling shows decode bottleneck | Architecture assessment complete |
| `imageInfo()` dominant color opt-in | Depends on info cache (Phase 2d) + full decode cost justification | Blocked on user demand |

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-06-28 | Investigation | ✅ Complete | See `work/reports/FEAT-148-phase2-investigation-results.md` |
| 2026-06-28 | Design decisions | ✅ Complete | See `work/design/DESIGN-image-transform-phase2.md` |
| 2026-06-28 | Spec updated | ✅ Complete | Phase 2 acceptance criteria in `work/specs/FEAT-148.md` |
| 2026-06-28 | Implementation plan | ✅ Complete | This document |
| 2026-06-28 | Tasks 1 & 2 | ✅ Complete | Sharpen on downscale (options, transform, registry) |
| 2026-06-28 | Tasks 3 & 4 | ✅ Complete | imageBlur() LQIP builtin with caching |
| 2026-06-28 | Task 6 | ✅ Complete | imageInfo() memory cache (sync.Map, path+modtime key) |
| 2026-06-28 | Tasks 5 & 7 | ✅ Complete | imageSrcset() builtin (width & density modes) |