# Design Investigation: FEAT-148 Phase 2 — Image Transform Enhancements

**Date:** 2026-06-28
**Status:** Investigation Complete
**Results:** `work/reports/FEAT-148-phase2-investigation-results.md`
**Related:**
- `work/specs/FEAT-148.md` — Feature specification (Phase 1 implemented, Phase 2 bullet points)
- `work/plans/FEAT-148-plan.md` — Phase 1 implementation plan (complete)
- `work/BACKLOG.md` — Backlog items #130–#133
- `server/images/` — Phase 1 implementation
- `pkg/parsley/evaluator/image.go` — `image()` and `imageInfo()` builtins

---

## 1. Purpose

Phase 1 of FEAT-148 (image transformation and caching) is implemented and reviewed. Phase 2 is defined in the spec as five bullet points with minimal detail. This document captures what needs to be investigated and designed before Phase 2 can be planned and implemented.

Each Phase 2 item is assessed for complexity, user value, open questions, and investigation tasks.

---

## 2. Phase 2 Items

### 2.1 `imageSrcset()` — Responsive Image Generation

**Backlog:** #130
**Complexity:** High
**User Value:** High — responsive images are expected in modern web development

#### What the spec says

> `imageSrcset(path, style, scales)` builtin: generate `srcset` attribute string for responsive images

#### Open questions

1. **What is `scales`?** Two competing models:
   - **Width descriptors:** explicit pixel widths like `[320, 640, 1280]` → `srcset="img-320.jpg 320w, img-640.jpg 640w, img-1280.jpg 1280w"`
   - **Pixel density descriptors:** multipliers like `[1, 2, 3]` applied to the style's base width → `srcset="img-300.jpg 1x, img-600.jpg 2x, img-900.jpg 3x"`
   - Or both, selected by argument type or option?

2. **Return value:** Just the `srcset` attribute value string? Or a dict with `{src, srcset, sizes, width, height}` to generate a complete `<img>` tag? A dict is more useful but more opinionated.

3. **`sizes` attribute:** `srcset` with width descriptors is useless without a corresponding `sizes` attribute. Should `imageSrcset()` also generate `sizes`? If so, how does the developer specify breakpoints?

4. **Interaction with `image()`:** Does `imageSrcset()` call `image()` N times internally (going through the registry for each variant), or does it use a batch API on the registry? The former is simpler and reuses existing code. The latter could enable optimizations (single source load, multiple resizes).

5. **Default scales:** Should there be sensible defaults if `scales` is omitted? e.g., `[0.5x, 1x, 2x]` of the style's base width?

6. **Naming:** `imageSrcset` is long. Alternatives: `srcset()`, `imageSet()`, `responsiveImage()`. Consider what reads best in a Parsley template.

#### Investigation tasks

- [x] Survey how other frameworks handle this (Next.js `<Image>`, Eleventy image plugin, Hugo image processing) — what API patterns work well?
- [x] Prototype two API shapes (width-descriptor vs density-descriptor) with real Parsley template examples and evaluate readability
- [x] Determine whether a batch registry API is needed or if N×`Transform()` is acceptable for typical scale counts (3–5 variants)
- [x] Decide on return type: string vs dict

---

### 2.2 Blur Placeholder Generation

**Backlog:** #131
**Complexity:** Medium
**User Value:** Medium — nice progressive loading UX, especially for hero images and galleries

#### What the spec says

> `{blur: true}` option producing a tiny blurred image for progressive loading

#### Open questions

1. **Return format:** The whole point of a blur placeholder is avoiding a network round-trip. If `image(@./photo.jpg, {blur: true})` returns a `/__img/` URL, the browser still makes a request. Should it instead return a `data:` URI (base64-encoded inline image)? That's a different return type from the normal `image()` string URL.

2. **Separate builtin or option?** Three possibilities:
   - `image(@./photo.jpg, {blur: true})` — returns data URI instead of URL (changes return semantics based on option, which is surprising)
   - `image(@./photo.jpg, {blur: true})` — returns URL to a tiny blurred image (simple, but defeats the purpose)
   - `imageBlur(@./photo.jpg)` or `imageInfo(@./photo.jpg).blur` — separate function or field that always returns a data URI

3. **Parameters:** What size should the blur placeholder be? Typical LQIP (Low Quality Image Placeholder) is ~20–40px wide, heavily blurred, JPEG at quality 20. Should the size/blur radius be configurable or just "good defaults"?

4. **CSS integration:** Blur placeholders are typically used with a CSS `background-image` on a container, then swapped out when the full image loads. Should we provide any guidance or helper for this pattern?

#### Investigation tasks

- [x] Decide: data URI vs URL (data URI is more useful but changes the function's return type contract)
- [x] Test `disintegration/imaging` `Blur()` function: resize to 20px wide → blur σ=10 → encode JPEG q=20 → base64. Measure output size (should be <1KB)
- [x] Prototype the API: which of the three approaches above reads best in real Parsley templates?
- [x] Determine whether `imageInfo()` is the right home for this (returns it alongside metadata) or if it belongs on `image()`

---

### 2.3 Dominant Color Extraction

**Backlog:** #133
**Complexity:** Low
**User Value:** Low — useful for placeholder backgrounds and design systems, but niche

#### What the spec says

> Dominant color extraction in `imageInfo()`: `{color: "#2a4f3b"}`

#### Open questions

1. **Algorithm:** Simple average of all pixels? Weighted center crop? K-means clustering (pick the largest cluster)? Average is fast but gives muddy results on diverse images. K-means gives better results but is slower.

2. **Library vs hand-rolled:** Is there a pure-Go library for dominant color extraction, or do we write a simple implementation? A basic "resize to 1×1 pixel, read the color" approach using `imaging.Resize` is trivial and might be good enough.

3. **Performance:** This only runs when `imageInfo()` is called, and `imageInfo()` currently reads just the image header. Adding color extraction means decoding the full image. Should the `color` field be opt-in (e.g., `imageInfo(@./photo.jpg, {color: true})`)?

4. **Format:** Hex string (`"#2a4f3b"`) is the obvious choice. Should it also return RGB components?

#### Investigation tasks

- [x] Evaluate pure-Go color extraction libraries (search for `dominant color`, `color palette`, `color quantization`)
- [x] Test the "resize to 1×1" shortcut: `imaging.Resize(img, 1, 1, imaging.Lanczos)` → read pixel → format as hex. Compare quality against a simple average and a k-means approach on 5–10 diverse images
- [x] Decide: always compute (adds full decode cost to `imageInfo()`) vs opt-in parameter
- [ ] Benchmark: how much time does full decode + resize-to-1×1 add vs header-only `DecodeConfig`? *(deferred — dominant color deferred to Phase 3)*

---

### 2.4 Smart Crop

**Backlog:** #132
**Complexity:** Medium
**User Value:** Medium — useful for user-uploaded content where subject position varies

#### What the spec says

> Smart crop via focal point detection: `{crop: "smart"}`

#### Open questions

1. **Library:** The spec mentions `muesli/smartcrop`. Status unknown — is it maintained? Pure Go? What's the quality and performance?

2. **Accuracy:** Smart crop algorithms vary wildly in quality. Face detection is the gold standard but requires ML models (heavy). `muesli/smartcrop` uses an edge/saturation/skin-tone heuristic — good enough for most cases?

3. **Performance:** Smart crop needs to analyze the full image to find the focal point. How does this compare to the ~100–300ms Phase 1 transform cost? Is it acceptable as a one-time cached cost?

4. **Fallback:** What happens when smart crop can't find a focal point? Fall back to center crop silently? Log a warning?

5. **Interaction with existing crop:** Currently `crop: "center"` uses `imaging.Fill` with `imaging.Center`. `crop: "smart"` would need to compute a focal point, then use a custom anchor. Does this fit cleanly into the existing `Transform()` function?

#### Investigation tasks

- [x] Evaluate `muesli/smartcrop`: check maintenance status, dependencies (must be pure Go, no CGo), star count, last commit, open issues
- [ ] Test `muesli/smartcrop` on 5–10 diverse images (portraits, landscapes, product shots, group photos): assess crop quality subjectively *(deferred — smart crop deferred to Phase 3)*
- [ ] Benchmark: time to compute focal point on a typical 12MP JPEG *(deferred — smart crop deferred to Phase 3)*
- [x] Check if there are alternative pure-Go smart crop libraries
- [x] Prototype integration: how does the focal point translate to an `imaging.Fill` anchor? Does `muesli/smartcrop` return coordinates that map to `imaging`'s anchor system, or do we need manual sub-image extraction?

---

### 2.5 Sharpen on Downscale

**Backlog:** Not separately tracked (included in Phase 2 spec bullet)
**Complexity:** Low
**User Value:** Low — subtle quality improvement that most users won't notice

#### What the spec says

> Sharpen on downscale: subtle unsharp mask when reducing image dimensions significantly

#### Open questions

1. **Threshold:** At what downscale ratio should sharpening kick in? 50% reduction? 2:1? Any downscale?

2. **Amount:** `disintegration/imaging` has `Sharpen(img, sigma)`. What sigma produces a subtle, natural-looking result? Typical values are 0.3–0.8 for web images.

3. **Opt-out:** Should this be automatic (always sharpen on downscale) or opt-in (`{sharpen: true}`)? Automatic is the "just works" approach but could surprise users. Opt-in is safer but less "batteries included."

4. **Interaction with explicit sharpen option:** Phase 2 also planned a `Sharpen` field on `TransformOptions`. If auto-sharpen exists, what happens when the user also specifies `{sharpen: 0.5}`? Override? Stack?

#### Investigation tasks

- [x] Test `imaging.Sharpen()` with sigma values 0.3, 0.5, 0.8 on 3–5 downscaled images. Compare visual quality (before/after, side by side)
- [x] Decide: automatic vs opt-in
- [x] Define the threshold and default sigma
- [x] This is simple enough that it may not need a full design doc — could go straight to implementation after the above tests

---

## 3. Cross-Cutting Concerns

### WebP Encoding (Deferred from Phase 1)

Phase 1 deferred `gen2brain/webp` due to binary size concerns. Phase 2 features don't strictly require WebP encoding, but `imageSrcset()` would benefit from it (serving WebP to supporting browsers is a key responsive image optimization).

**Investigation tasks:**
- [x] Measure binary size impact of adding `gen2brain/webp` (build with and without, compare) — baseline recorded; actual with-webp measurement deferred to `imageSrcset()` implementation
- [ ] Benchmark WebP encode performance via WASM on typical transforms *(deferred — WebP encoding deferred until imageSrcset() is ready)*
- [x] Decide: include in Phase 2 or continue deferring? — **continue deferring**, measure when `imageSrcset()` is implemented

### `imageInfo()` Metadata Caching

The Phase 1 review noted that `imageInfo()` results are not cached in memory. Phase 2 adds dominant color extraction, which requires full image decode — making the caching question more pressing.

**Investigation tasks:**
- [x] Decide: add a simple `sync.Map` cache keyed on absolute path + modtime? Or defer until there's evidence of a real performance problem? — **add cache**, low-risk and becomes important when dominant color/blur are added to `imageInfo()`

---

## 4. Recommended Priority

Based on complexity and user value:

| Priority | Item | Rationale |
|----------|------|-----------|
| 1 | `imageSrcset()` | Highest user value; most complex; shapes the API surface |
| 2 | Blur placeholder | Good UX improvement; medium complexity; depends on API decision |
| 3 | Sharpen on downscale | Low complexity; can ship quickly after a small tuning exercise |
| 4 | Dominant color | Low complexity but niche value; easy to add to `imageInfo()` |
| 5 | Smart crop | Depends on external library quality; evaluate before committing |

---

## 5. Next Steps

1. ~~**Pick which items to include in Phase 2** — not all five need to ship together~~ ✅ Done — see results report
2. ~~**Run the investigation tasks** for selected items~~ ✅ Done — see `work/reports/FEAT-148-phase2-investigation-results.md`
3. **Write API proposals** for `imageSrcset()` and `imageBlur()` with Parsley template examples
4. **Update the spec** (`work/specs/FEAT-148.md` Phase 2 section) with concrete acceptance criteria
5. **Create an implementation plan** (`work/plans/FEAT-148-phase2-plan.md`) once the design is settled