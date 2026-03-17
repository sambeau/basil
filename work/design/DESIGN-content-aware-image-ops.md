# Design: Content-Aware Image Operations (Smart Crop & Seam Carving)

**Date:** 2026-03-16
**Status:** Exploration / Discussion
**Author:** @sam, @copilot
**Related:**
- `work/design/DESIGN-image-transform-phase2.md` — Phase 2 image enhancements (smart crop deferred to Phase 3)
- `work/specs/FEAT-148.md` — Image transformation feature spec
- `work/BACKLOG.md` — Backlog item #132 (smart crop)

---

## 1. Overview

### 1.1 Motivation

Basil's image pipeline (FEAT-148 Phase 1) supports basic transforms: resize, crop to center, format conversion. Phase 2 identified smart crop (`{crop: "smart"}`) as a desirable feature but deferred it to Phase 3 pending investigation.

This document explores two related techniques:

1. **Content-Aware Cropping ("Smart Crop")** — Analyse an image for high-interest regions and crop to preserve them, rather than blindly cropping to center.
2. **Content-Aware Scaling ("Seam Carving")** — Resize an image by removing or inserting low-interest pixel seams, preserving high-interest content.

Both are well-understood algorithms with pure-Go implementations possible. The question is whether we can build something good enough ourselves, aligned with the Parsley aesthetic (simplicity, minimalism, completeness, composability), without taking on heavy dependencies.

### 1.2 Design Principles

- **No C dependencies** — pure Go only, no CGo, no system-installed libraries
- **No CLI tool dependencies** — must not require external binaries
- **No ML models for v1** — heuristic-based, no training data required
- **Minimal dependency surface** — prefer hand-rolled implementations over third-party libraries
- **Good enough for 90% of cases** — face detection and object recognition are v2 concerns
- **Standalone library** — reusable outside of Basil

---

## 2. Content-Aware Cropping

### 2.1 Technique

Smart cropping evaluates candidate crop rectangles and picks the one that preserves the most "interesting" content. The scoring heuristic combines:

1. **Edge detection** (Sobel filter) — high-edge areas indicate detail and structure
2. **Saturation scoring** — colourful areas are more visually interesting than flat grey
3. **Skin-tone detection** — a simple colour-range heuristic (HSV thresholds, not ML) that biases toward faces and people
4. **Rule-of-thirds weighting** — boost scores near thirds intersections, matching photographic composition

This is the approach used by [smartcrop.js](https://github.com/jwagner/smartcrop.js/), which achieves surprisingly good results with zero ML. The entire scoring algorithm fits in a few hundred lines.

### 2.2 Candidate Search

To keep computation tractable:

- **Downscale for analysis** — resize the source image to ~256px on the long side before scoring
- **Grid sampling** — evaluate crop positions on a grid (e.g., every 10–20px at analysis scale) rather than every pixel
- **Fixed scale factors** — test a small set of crop scales (e.g., 1.0, 0.9, 0.8 of the target aspect ratio)
- **Map back to original** — apply the winning crop rectangle to the full-resolution image

### 2.3 Proposed API

```parsley
// In Basil's image pipeline (query parameter):
//   /images/photo.jpg?w=400&h=300&crop=smart

// In Parsley templates:
image(@./photo.jpg, {width: 400, height: 300, crop: "smart"})
```

As a standalone Go library:

```go
// Basic usage — returns the best crop rectangle
crop := smartcrop.BestCrop(img, targetWidth, targetHeight)
cropped := imaging.Crop(img, crop)

// With options (functional options pattern, sensible defaults)
crop := smartcrop.BestCrop(img, w, h,
    smartcrop.WithBoost(rect),       // bias toward a region (e.g., user-specified focal point)
    smartcrop.WithMinScale(0.8),     // minimum crop scale relative to source
)
```

### 2.4 What This Does NOT Include (v1)

- Face detection (ML-based)
- Object detection / saliency maps
- Eye-gaze or attention prediction

These could be added as a v2 enhancement. [Pigo](https://github.com/esimov/pigo) is a pure-Go face detector based on [pixel intensity comparison](https://arxiv.org/pdf/1305.4537) that ships a ~1.2MB cascade file — a possible future integration point without breaking the "no C dependencies" rule.

---

## 3. Content-Aware Scaling (Seam Carving)

### 3.1 Technique

Seam carving resizes an image by removing (or inserting) connected paths of low-energy pixels — "seams" — rather than uniformly scaling all pixels. This preserves high-detail regions while compressing or expanding low-detail regions.

The core algorithm:

1. **Compute energy map** — gradient magnitude at each pixel (Sobel filter, same as smart crop)
2. **Find minimum-energy seam** — dynamic programming, single pass from top to bottom (vertical seam) or left to right (horizontal seam)
3. **Remove the seam** — delete that one-pixel-wide path from the image
4. **Repeat** — until the target width (or height) is reached

To enlarge an image, the process is reversed: find the lowest-energy seams and duplicate them.

The algorithm is well-defined, entirely deterministic, and requires ~200–300 lines of Go.

### 3.2 Proposed API

As a standalone Go library:

```go
// Basic usage — resize to target dimensions
result := seamcarve.Resize(img, targetWidth, targetHeight)

// With protect/remove masks
result := seamcarve.Resize(img, w, h,
    seamcarve.WithProtect(mask),    // mark regions to never carve through
    seamcarve.WithRemove(mask),     // mark regions to preferentially remove
)
```

In Basil's image pipeline:

```parsley
// Query parameter:
//   /images/photo.jpg?w=800&scale=smart

// Parsley template:
image(@./photo.jpg, {width: 800, scale: "smart"})
```

### 3.3 Limitations

- Seam carving works best when the aspect ratio change is modest (< 30% reduction). Extreme resizing produces visible artifacts.
- Images with uniform high-detail content (e.g., dense text, busy patterns) don't benefit — there are no low-energy seams to exploit.
- Vertical-only or horizontal-only carving is simpler and more predictable. Carving in both directions simultaneously is more complex and can produce artifacts.

---

## 4. Shared Infrastructure

Both techniques share a foundation:

| Component | Smart Crop | Seam Carving |
|-----------|-----------|-------------|
| Edge detection (Sobel filter) | ✓ scoring input | ✓ energy map |
| Saturation map | ✓ scoring input | — |
| Skin-tone detection | ✓ scoring input | — |
| Image downscaling | ✓ analysis scale | — |
| Dynamic programming | — | ✓ seam finding |

Both techniques share the Sobel energy-map computation, but neither depends on the other — the shared utility is a single function that can live in either package or be factored into a small helper.

---

## 5. Project Structure

### 5.1 ~~Standalone Library~~ → Build Inside Basil

> **Updated:** Originally recommended a standalone library. Revised after evaluating the integration surface with `server/images/`. See §10.1 for full reasoning.

Both features should live inside Basil as internal sub-packages of `server/images/`. The existing image pipeline (`Transform()`, `Load()`, `Encode()`, `TransformOptions`, caching, handler) is where these operations need to integrate — they're new modes within the existing transform pipeline, not standalone operations.

**Rationale for building inside Basil:**
- `{crop: "smart"}` needs to slot into the same `Transform()` function that handles `{crop: "center"}`
- The face detector is purpose-built for cropping boost regions, not a general-purpose library
- Avoids separate repo overhead (versioning, CI, releases, `go.mod` coordination) for ~500–800 lines of code
- Can reuse `disintegration/imaging` which Basil already depends on
- Integration tests exercising the full path (Parsley template → handler → smart crop → cache) are more valuable than isolated unit tests
- Sub-package boundaries allow future extraction into a standalone library if needed

**Structure inside Basil:**

```
server/images/
├── cache.go                    # (existing)
├── handler.go                  # (existing)
├── options.go                  # (existing — add crop: "smart", scale: "smart")
├── transform.go                # (existing — dispatch to smartcrop/seamcarve)
├── smartcrop/
│   ├── smartcrop.go            # scoring + candidate search
│   ├── score.go                # edge, saturation, skin-tone, composition scoring
│   ├── sobel.go                # Sobel edge detection (shared utility)
│   ├── detect.go               # PICO face detector runtime
│   ├── cascade.go              # cascade file loader + go:embed
│   ├── facefinder              # embedded cascade binary (234 KB)
│   └── CASCADE_LICENSE         # MIT attribution for Pigo's cascade file
└── seamcarve/
    ├── seamcarve.go            # seam finding + removal
    └── energy.go               # energy map (reuses Sobel concept from smartcrop)
```

### 5.2 Integration with Existing Pipeline

Smart crop and seam carving wire into the existing `Transform()` function in `server/images/transform.go`:

- `{crop: "smart"}` → runs PICO face detection + heuristic scoring → picks best crop rectangle → `imaging.Crop()`
- `{scale: "smart"}` → runs seam carving DP to target width/height
- Both produce results cached by the existing `CacheKey()` mechanism — no caching changes needed

---

## 6. Feasibility Assessment

### 6.1 Can it be done without training data?

**Mostly yes.** The algorithms themselves are entirely computational — no training step, no ML inference at runtime:

- Smart crop scoring uses per-pixel heuristics (edges, saturation, skin-tone colour ranges)
- Seam carving uses gradient energy and dynamic programming
- The PICO face detector is a simple binary tree classifier — the runtime is pure arithmetic

The one pre-trained artefact is the **cascade file** (234 KB) that encodes which pixel-pair comparisons discriminate faces from non-faces. This was trained offline by the PICO paper's authors using academic face datasets. We bundle it as a static, embedded asset — the user provides nothing and configures nothing. This is analogous to how OpenCV ships its Haar cascade XML files.

### 6.2 Effort Estimate

**Smart crop with face detection (Phase 1):**

| Component | Effort | Notes |
|-----------|--------|-------|
| Sobel energy map | < 1 day | ~50 lines, shared utility |
| Heuristic scoring (edge, saturation, skin-tone, composition) | 3–4 days | GenCrop V1–V5 rules, weight tuning |
| PICO face detector runtime | 1–2 days | ~200 lines, reimplemented from paper |
| Cascade file embedding + loader | < 1 day | ~40 lines, `go:embed` |
| Boost integration (faces → scoring) | < 1 day | ~50 lines |
| Candidate search + downscaling | 1–2 days | Grid search, scale factors |
| Basil integration (`{crop: "smart"}`) | 1 day | Wire into `Transform()`, update `TransformOptions` |
| Tests + quality tuning | 2–3 days | Diverse image set, weight adjustment |
| **Subtotal** | **~2 weeks** | Smart crop with face detection, integrated into Basil |

**Seam carving (Phase 2, when there's demand):**

| Component | Effort | Notes |
|-----------|--------|-------|
| Seam carving (core DP + removal) | 2–3 days | Backward energy, reduction only |
| Tests + Basil integration | 1–2 days | `{scale: "smart"}` in `TransformOptions` |
| **Subtotal** | **3–5 days** | |

The bulk of the smart crop effort is in heuristic weight tuning — getting results that "feel right" across diverse image types. The face detection runtime itself is straightforward.

### 6.3 Performance Expectations

| Operation | Expected Cost | Cacheable? |
|-----------|--------------|------------|
| Smart crop heuristic scoring (256px analysis scale) | ~5–20ms | Yes (same as other transforms) |
| PICO face detection (on analysis-scale image) | ~10–50ms | Yes (part of smart crop pipeline) |
| Smart crop — full pipeline (detect + score + crop) | ~100–400ms | Yes |
| Seam carving — 10% width reduction on 2000px image | ~200–500ms | Yes |
| Seam carving — 30% width reduction on 2000px image | ~500–1500ms | Yes |

Both operations are one-time costs per unique transform, cached by Basil's existing image cache.

---

## 7. Prior Art

| Project | Language | Approach | Notes |
|---------|----------|----------|-------|
| [smartcrop.js](https://github.com/jwagner/smartcrop.js/) | JavaScript | Edge + saturation + skin heuristic | Our primary reference for smart crop |
| [muesli/smartcrop](https://github.com/muesli/smartcrop) | Go | Port of smartcrop.js | Pure Go; maintenance status unknown — needs evaluation |
| [Pigo](https://github.com/esimov/pigo) | Go | Cascade classifier face detection | Pure Go; possible v2 integration |
| [ganimtron-10/seam-carving](https://github.com/ganimtron-10/seam-carving) | Go | Basic seam carving | Small reference implementation |
| [vivianhylee/seam-carving](https://github.com/vivianhylee/seam-carving) | Python | Seam carving with forward energy | Good reference for advanced energy functions |

Academic references:
- [Seam Carving for Content-Aware Image Resizing](https://en.wikipedia.org/wiki/Seam_carving) — Avidan & Shamir, 2007
- [Pixel Intensity Comparisons for Object Detection](https://arxiv.org/pdf/1305.4537) — Pigo's underlying technique
- [Content-Aware Image Retargeting: A Comprehensive Review](https://www.sciencedirect.com/science/article/abs/pii/S0165168422000433)

---

## 8. ~~Original Recommended Approach~~ (Superseded)

> **Note:** This section preserves the original phasing recommendation for context. It has been superseded by the revised phasing in §13, which reflects the decisions to build inside Basil (not as a standalone library), start with smart crop (not seam carving), and include face detection from day one.

### ~~Phase 1: Seam Carving Library~~

Start here because:
- The algorithm is precisely defined (less tuning)
- Establishes shared energy-map infrastructure
- Visually impressive — good for validating the approach

Deliverables:
- Pure-Go seam carving library with no external dependencies
- Width reduction and enlargement
- Optional protect/remove masks
- Benchmark suite

### ~~Phase 2: Smart Crop Library~~

Build on the energy-map foundation:
- Scoring heuristics (edge, saturation, skin-tone, thirds)
- Candidate crop search with configurable target aspect ratio
- Weight tuning against a diverse set of test images

Deliverables:
- Pure-Go smart crop library
- Sensible default weights
- Optional boost regions (user-specified focal points)
- Visual test suite (before/after comparisons)

### ~~Phase 3: Basil Integration~~

Wire both libraries into Basil's image transform pipeline:
- `{crop: "smart"}` for content-aware cropping
- `{scale: "smart"}` for content-aware scaling
- Cached like all other transforms

### ~~Phase 4 (Future): Enhanced Detection~~

If heuristic smart crop proves insufficient:
- Evaluate Pigo integration for face-aware cropping
- Consider a simple saliency model
- Remain pure-Go, ship any model data as embedded assets

---

## 9. Face Detection: Later Phase Discussion

### 9.1 Why Face Detection Matters

Experience with heuristic-only smart cropping (smartcrop.js and similar) shows clear limitations without face detection. The smartcrop.js architecture is instructive — skin detection has the highest heuristic weight (1.8 vs detail's 0.2 and saturation's 0.1), but its `boost` API for face detection regions dominates at 100.0, because **the author knows the heuristics aren't enough**.

Known failure modes of heuristic-only cropping:

| Scenario | Heuristic Quality | With Face Detection |
|----------|------------------|---------------------|
| Single person, clear background, medium skin tone | Good (~85%) | Excellent (~95%+) |
| Single person, dark skin tone | Poor (~60%) | Excellent (~95%+) |
| Group photos | Poor (~50%, gravitates to largest skin mass) | Good (~85%+) |
| Person in skin-toned environment (beach, wood, sand) | Poor (~40%, false positives) | Excellent (~90%+) |
| Fully clothed, small face, landscape | Very poor (~30%) | Good (~85%+) |
| Non-human subjects (animals, products) | No skin signal (edge/saturation only) | N/A |

The skin-tone heuristic has a deeper problem: the standard reference colour (`[0.78, 0.57, 0.44]` in normalised RGB, roughly a medium-light Caucasian tone) is biased. Darker and very light skin tones score lower. This is both a technical limitation and an inclusivity concern. Face detection sidesteps this entirely — it detects faces by geometry, not colour.

**Conclusion:** Heuristic-only cropping is a defensible v1 ("better than center crop"), but face detection should be planned as a near-term follow-up, not a distant v3 aspiration.

### 9.2 The PICO Algorithm (Pigo's Foundation)

Pigo implements the **Pixel Intensity Comparison-based Object (PICO)** detection algorithm from [Markuš et al., 2014](https://arxiv.org/pdf/1305.4537). The algorithm is surprisingly simple:

**How it works:**

1. **Grayscale conversion** — the image is converted to a flat `[]uint8` pixel buffer. No integral images, HOG pyramids, or other preprocessing needed.
2. **Sliding window with scale pyramid** — a detection window scans the image, starting at `MinSize`, stepping by `ShiftFactor * scale` pixels, growing by `ScaleFactor` each pass until `MaxSize`.
3. **Binary decision tree cascade** — at each window position, a cascade of binary trees classifies the region. Each tree node performs a single **pixel intensity comparison**: read two pixel values at offsets relative to the window center/scale, branch left or right based on which is brighter. That's it — no convolutions, no matrix operations, no floating-point-heavy maths.
4. **Early rejection** — each tree produces a score. If the cumulative score falls below a threshold after any tree, the window is immediately rejected (not a face). This makes rejection very fast.
5. **Clustering (NMS)** — overlapping detections are merged using Intersection over Union.

The key insight: **all the intelligence is in the cascade file**, which encodes which pixel pairs to compare, at what offsets, and what to predict at each leaf. The runtime code is just a loop that walks the trees.

### 9.3 Pigo: Size, License, and Feasibility

| Aspect | Detail |
|--------|--------|
| **License** | MIT (Copyright © 2018 Endre Simo). Fully permissive — use, modify, redistribute, sublicense freely. |
| **Core detector size** | ~280 lines of Go (`pigo.go`). The entire `core/` package is ~760 lines across 7 files. |
| **Core dependencies** | **Zero external dependencies.** Only Go standard library (`encoding/binary`, `math`, `sort`, `sync`, `unsafe`, `image`). The external deps (`disintegration/imaging`, `fogleman/gg`, `golang.org/x/term`) are used only by the CLI tool and examples. |
| **Cascade file** | `facefinder` — **234 KB** binary. Pre-trained, encodes the decision trees. Simple custom binary format (not protobuf), parsed in ~40 lines. |
| **Additional cascades** | `puploc` (pupil localisation), `lps/` (facial landmarks). Not needed for cropping — only `facefinder` matters. |

### 9.4 Can We Roll Our Own?

**Yes — and the effort is very modest.** There are three distinct questions:

#### Can we reimplement the detection runtime from the paper?

**Absolutely.** The entire Pigo `classifyRegion` function is ~30 lines. The core algorithm is:

```
for each tree in cascade:
    idx = 1
    for each depth level:
        read two pixels at offsets encoded in treeCodes
        branch left or right based on which pixel is brighter
    accumulate leaf prediction
    if cumulative score < threshold: reject (not a face)
return score
```

`RunCascade` is a standard sliding-window loop. `ClusterDetections` is textbook IoU non-maximum suppression. A competent developer could implement the complete detection runtime in a day from the paper alone, without ever looking at Pigo's source. There is nothing clever or complex about the runtime — the clever part was the training.

**Estimated size:** ~200–300 lines of Go for a clean implementation (detection + NMS + grayscale conversion). No external dependencies.

#### Can we use the trained cascade file?

**Yes.** The `facefinder` cascade file in Pigo's repository is MIT-licensed. We can freely redistribute it. Options:

1. **Bundle Pigo's cascade file** (234 KB) — simplest, proven quality, MIT-licensed
2. **Train our own** — requires implementing the training algorithm from the paper *and* a large labelled face dataset. This is significantly harder than the runtime and not worth doing. The existing cascade works.
3. **Go:embed the cascade** — ship it as an embedded asset in the library binary, so there's no file-path configuration needed at runtime

Option 1 (with `go:embed`) is the clear winner. We get face detection with zero runtime configuration.

#### Do we even need to reimplement, or could we just use Pigo?

We *could* import `pigo/core` directly — it has zero external dependencies. But there are reasons to prefer our own implementation:

- **Control** — we own the code, can trim what we don't need (rotation support, pupil localisation, landmarks), and optimise for our use case
- **Simplicity** — Pigo includes features we don't need; a purpose-built detector for smart cropping can be simpler
- **The algorithm is trivial to implement** — at ~200 lines, writing it ourselves is barely more work than integrating a dependency
- **Cascade file compatibility** — Pigo's binary format is simple enough that we can match it (or define our own loader) without difficulty

**Recommendation:** Reimplement the ~200-line runtime ourselves, bundle Pigo's MIT-licensed cascade file. This gives us face detection with zero Go dependencies and full control, while avoiding the need to train our own cascade.

### 9.5 What About the GenCrop Paper?

The [GenCrop paper](https://arxiv.org/abs/2312.12080) ("Learning Subject-Aware Cropping by Outpainting Professional Photos", AAAI 2024) proposes an ML-heavy pipeline: Stable Diffusion for outpainting, BLIP-2 for captioning, YOLOv8 for detection, and a 24.9M-parameter ResNet-50 + Transformer cropping model. This is far too heavy for our use case.

However, the paper is **extremely valuable as a specification** for what good cropping looks like. Its key findings, derived from analysing professional photography, validate heuristics we can implement cheaply:

**Five violation criteria** (what professional crops avoid):

1. **V1: Don't cut unnaturally through the subject** — never crop a person at a joint (ankle, knee, elbow, chin). Add a ~2.5% margin around subject bounds.
2. **V2: Don't cut through scene context** — don't chop companion objects, reflections, or interacting elements.
3. **V3: Get negative space right** — not too tight, not unbalanced. More space in the direction of gaze/movement.
4. **V4: Avoid edge clutter** — don't leave partially-visible distracting elements at frame edges. Include fully or exclude fully.
5. **V5: Maintain balance** — respect rule-of-thirds, symmetry, leading lines.

**Other actionable findings:**

- Subject should occupy ~10–80% of the crop area
- Professional photos cluster around 3:2 and 2:3 aspect ratios (41% and 25% of Unsplash)
- The specific cropping model architecture barely matters — the *rules* matter more than the mechanism
- Even without subject information, composition rules alone outperform prior ML methods

These findings strengthen our heuristic approach and give us a research-backed checklist for scoring candidate crops — all implementable without any ML.

### 9.6 The "Boost" Architecture

Smartcrop.js has the right architectural pattern: **separate detection from scoring**. Face detection (or any subject detector) produces rectangular "boost" regions with weights. The crop scorer weights these regions heavily (100× the heuristic signals), so a detected face completely dominates the scoring.

This means our library can be structured as:

```
Phase 1 (heuristic):  edge + saturation + skin-tone scoring → "better than center"
Phase 2 (detection):  PICO face detector → boost regions → face-aware scoring
```

The boost API is the bridge. Phase 1 works without it. Phase 2 plugs in via the same scoring pipeline. Users could even supply their own boost regions (e.g., from an external face detector, or manually specified focal points) without us changing the core scorer.

### 9.7 Recommended Approach for Face Detection Phase

| Step | Effort | Description |
|------|--------|-------------|
| Implement PICO runtime | 1–2 days | ~200 lines: sliding window, tree classifier, NMS. From the paper, not from Pigo source. |
| Bundle cascade file | < 1 hour | Embed Pigo's MIT-licensed `facefinder` (234 KB) via `go:embed`. |
| Wire into smart crop scorer | 1 day | Run face detection, convert detections to boost regions, feed into existing scoring pipeline. |
| Test and tune | 1–2 days | Validate on diverse portrait images: skin tones, group shots, small faces, clothed subjects. |
| **Total** | **3–5 days** | On top of the Phase 2 smart crop work. |

### 9.8 What We Don't Need

- **Pupil localisation** — Pigo supports it, but we don't need eye positions for cropping. The face bounding box is sufficient.
- **Facial landmarks** — same reasoning. Cropping needs "where is the face", not "where are the eyebrows".
- **Rotation detection** — Pigo supports in-plane rotation. For web images, faces are almost always upright. Skip for v1; add later if needed.
- **Training infrastructure** — we use the existing cascade file. Training is a solved problem we don't need to re-solve.

---

## 10. Open Questions — Recommendations

### 10.1 Separate Library or Inside Basil?

**Recommendation: Build inside Basil as internal packages.**

The original argument for a standalone library was reusability and clean separation. But looking at the actual shape of this work, a separate project creates more friction than it removes:

- **Basil already has `server/images/`** with `Transform()`, `Load()`, `Encode()`, `TransformOptions`, caching, and the handler pipeline. Smart crop and seam carving need to integrate tightly with this — they're not standalone operations, they're new modes within the existing transform pipeline. `{crop: "smart"}` needs to slot into the same `Transform()` function that handles `{crop: "center"}`.
- **The face detector is purpose-built for cropping.** We're not building a general-purpose face detection library — we're building just enough to generate boost regions for crop scoring. A standalone library would either be too specific to attract outside users or too general for what we actually need.
- **Dependency management overhead.** A separate repo means separate versioning, separate CI, separate releases, and `go.mod` version coordination between the library and Basil. For ~500–800 lines of code, that's more infrastructure than implementation.
- **Basil already has `disintegration/imaging`** as a dependency. Smart crop and seam carving both need basic image operations (resize, pixel access). Inside Basil, they can use `imaging` directly. A standalone library would either duplicate that dependency or introduce its own, more primitive pixel manipulation.
- **Testing is easier.** The real test of smart crop quality is "does the crop look right in the context of Basil's image pipeline?" Integration tests that exercise the full path (Parsley template → image handler → smart crop → cached response) are more valuable than unit tests on an isolated library.

**Package structure inside Basil:**

```
server/images/
├── cache.go            # (existing)
├── handler.go          # (existing)
├── options.go          # (existing — add crop: "smart", scale: "smart")
├── transform.go        # (existing — dispatch to smartcrop/seamcarve)
├── smartcrop/
│   ├── smartcrop.go    # scoring + candidate search
│   ├── score.go        # edge, saturation, skin-tone, composition scoring
│   ├── detect.go       # PICO face detector runtime
│   ├── cascade.go      # cascade file loader + go:embed
│   └── facefinder      # embedded cascade binary (234 KB)
└── seamcarve/
    ├── seamcarve.go    # seam finding + removal
    └── energy.go       # energy map (Sobel filter — shared concept with smartcrop)
```

The `smartcrop` and `seamcarve` sub-packages are still cleanly separated from each other and from the main `images` package — we get the separation benefits without the overhead of a separate project. If they ever need to be extracted into a standalone library later, the package boundaries are already clean.

**If we ever want to extract it:** moving `server/images/smartcrop/` to its own repo later is a straightforward `go mod` operation. Starting inside Basil doesn't close that door.

### 10.2 muesli/smartcrop — Use or Build from Scratch?

**Recommendation: Build from scratch.**

`muesli/smartcrop` is a Go port of smartcrop.js. While it exists and is pure Go, there are several reasons to build our own:

- **We're going beyond what it offers.** muesli/smartcrop implements the same heuristic-only approach as smartcrop.js — edge, saturation, skin-tone scoring. We plan to add face detection via PICO and composition rules from the GenCrop findings. Starting from muesli/smartcrop would mean forking and heavily modifying it rather than building cleanly.
- **The implementation is small.** The entire scoring + candidate search is a few hundred lines. Writing it ourselves, tuned for our needs, is comparable effort to understanding, integrating, and extending someone else's code.
- **We avoid the skin-tone bias.** We can design our skin-tone heuristic (or decide to de-weight it) from the start, rather than inheriting the single-reference-colour approach and patching it later.
- **We control the scoring weights.** The relative weights of edge, saturation, skin-tone, and boost signals are the most important tuning parameters. Owning the scorer means we can adjust these freely as we test on diverse images.
- **No dependency to track.** One less external module in `go.mod`.

### 10.3 Seam Carving — Reduction Only or Include Enlargement?

**Recommendation: Reduction only for v1.**

Seam carving enlargement (inserting seams) is the reverse operation: find the N lowest-energy seams, then duplicate them to grow the image. It works, but:

- **The use case is narrow.** In web image serving, you almost always want to *reduce* an image to fit a layout, not enlarge it. Basil already refuses to upscale (`if targetWidth > srcWidth { targetWidth = srcWidth }`).
- **Quality degrades faster.** Inserting seams produces visible repetition artifacts more quickly than removing them. The results are less impressive and harder to tune.
- **It roughly doubles the implementation surface** for a feature few users will need.
- **Easy to add later.** The seam-finding infrastructure is the same for both operations. Adding enlargement later is a small delta on top of the reduction implementation.

Ship reduction only. If users request enlargement, it's a straightforward follow-up.

### 10.4 Forward Energy vs Backward Energy?

**Recommendation: Start with backward energy, consider forward energy if artefacts are visible.**

The original Avidan & Shamir (2007) seam carving uses **backward energy** — the energy of pixels being removed. Rubinstein et al. (2008) improved this with **forward energy** — the energy *introduced* by removing a seam (i.e., the cost of the new pixel adjacencies created when neighbours collapse together).

Forward energy produces noticeably better results on images with strong edges, because it avoids creating new discontinuities. However:

- **Backward energy is simpler.** The energy map is just a Sobel gradient magnitude — the same computation we need for smart crop scoring. No additional logic.
- **Forward energy adds ~50 lines** to the seam-finding DP. Not huge, but it changes the recurrence relation and adds three cost terms per pixel per row.
- **For modest reductions (< 20%), the difference is subtle.** Extreme reductions show clear differences, but those are also the cases where seam carving in general starts to break down.

Implement backward energy first. If visual testing reveals artefacts on strong-edge images, forward energy is a contained upgrade to the DP step — it doesn't change the API or architecture.

### 10.5 Integration Priority — Wait for Both or Ship Incrementally?

**Recommendation: Ship smart crop first, then seam carving.**

This reverses the earlier "start with seam carving" recommendation. The reasoning:

- **Smart crop is already in the FEAT-148 spec** (`{crop: "smart"}` is a defined but deferred feature). Seam carving is not. Users are expecting smart crop.
- **Smart crop has a clearer use case.** "Crop my thumbnail to show the face" is something every CMS user understands. Seam carving is a more niche technique that requires explanation.
- **The shared infrastructure argument was overstated.** Both use a Sobel energy map, but the implementations are different: smart crop uses it as one input to a scoring function over candidate rectangles; seam carving uses it as the basis for dynamic programming over pixel paths. Sharing a `sobel()` helper function is trivial — it doesn't require building seam carving first.
- **Basil integration for smart crop is straightforward.** `TransformOptions.Crop` already accepts `"center"` — adding `"smart"` is a matter of calling the smart crop scorer before `imaging.Crop()` in `Transform()`.

The phasing becomes:
1. Smart crop with face detection → immediate user value
2. Seam carving → follows when there's demand or when someone wants to build it
3. They share the Sobel filter utility, but neither depends on the other

### 10.6 Face Detection Timing — Day One or Fast-Follow?

**Recommendation: Day one, for the reasons already discussed in §9.**

To reiterate briefly: the heuristic-only approach has well-documented failure modes, particularly around skin-tone bias. The PICO face detector adds ~200 lines and 234 KB. It's less than a week of additional work. Shipping without it and then quickly shipping it after is more total effort (two rounds of testing, two rounds of integration, release overhead) than including it from the start.

The boost architecture means face detection is cleanly additive — it doesn't complicate the heuristic scorer, it just provides an additional (dominant) signal.

### 10.7 Cascade File Provenance

**Recommendation: Bundle the file, add attribution, and document provenance.**

The `facefinder` cascade file from Pigo's repository is MIT-licensed. MIT requires only that the copyright notice and permission notice be included. The practical steps:

1. **Copy the cascade file** into `server/images/smartcrop/facefinder` (234 KB).
2. **Add a `CASCADE_LICENSE` file** alongside it with Pigo's MIT notice:
   ```
   The facefinder cascade file is derived from the Pigo project
   (https://github.com/esimov/pigo) by Endre Simo.
   Licensed under the MIT License. See below.

   Copyright (c) 2018 Endre Simo
   [full MIT text]
   ```
3. **Embed it with `go:embed`** so there's no file-path configuration at runtime.
4. **Note in THIRD_PARTY or NOTICES** (if Basil has one, or create one) that the cascade file originates from Pigo.

This is sufficient for MIT compliance. We are *not* copying Pigo's Go source code — we're reimplementing the algorithm from the paper. The cascade file is a trained data artifact, not code, but the license covers it regardless.

The cascade file's training data provenance (what face images were used to train it) is outside our control — it was trained by the PICO paper's authors using academic face datasets. This is standard practice for bundled classifiers (OpenCV ships similar cascade files). We don't need to trace the training data.

### 10.8 Non-Face Subjects

**Recommendation: Accept the limitation for v1. The heuristic fallback is good enough for most non-portrait content.**

Face detection helps portraits. For everything else, the heuristic scorer (edge density, saturation, composition rules) is the fallback. Let's assess how that fallback performs across content types:

| Content Type | Heuristic Quality | Why |
|---|---|---|
| **Landscapes** | Good | High-detail regions (trees, buildings, horizon lines) score well on edge density. Rule-of-thirds placement naturally favours interesting compositions. |
| **Product shots on plain background** | Good | Product has all the edge/saturation signal; background has none. The scorer naturally gravitates to the product. |
| **Food photography** | Good | Colourful, textured subjects against simpler plates/surfaces. Saturation and edge scoring work well. |
| **Product shots on busy background** | Fair | Competing signals. The crop may not centre on the product if the background has high edge density too. |
| **Animals** | Fair to poor | No face detection, no skin-tone signal. Depends on the animal having enough edge contrast against the background. |
| **Text/graphics** | Poor | Uniform high-detail. No clear focal point. Falls back to roughly center-weighted. |

For the cases where heuristics struggle, the **user-supplied boost region** is the escape hatch. Basil's API already supports options dicts — adding a `focal` option is straightforward:

```parsley
// Explicit focal point for non-portrait content
image(@./product.jpg, {width: 400, height: 300, crop: "smart", focal: {x: 0.3, y: 0.5}})
```

This converts to a boost region in the scoring pipeline — the same mechanism face detection uses. No new architecture needed, just a new option in `TransformOptions`.

For v1, the combination of heuristic scoring + face detection + optional focal point covers the vast majority of real-world use cases. Object detection (animals, products in busy scenes) would require a different cascade or a more complex model — that's genuinely a future concern.

---

## 11. Resolved Questions

For reference, the original open questions and their resolutions:

| # | Question | Resolution | Section |
|---|----------|-----------|---------|
| 1 | Separate library or inside Basil? | Inside Basil as `server/images/smartcrop/` and `server/images/seamcarve/` | §10.1 |
| 2 | Use muesli/smartcrop or build from scratch? | Build from scratch | §10.2 |
| 3 | Seam carving enlargement in v1? | No, reduction only | §10.3 |
| 4 | Forward energy or backward energy? | Backward for v1, forward if artefacts are visible | §10.4 |
| 5 | Integration priority? | Smart crop first (reverses earlier recommendation) | §10.5 |
| 6 | Face detection timing? | Day one | §10.6 |
| 7 | Cascade file provenance? | Bundle with attribution + CASCADE_LICENSE file | §10.7 |
| 8 | Non-face subjects? | Heuristic fallback + optional focal point option | §10.8 |

---

## 12. Remaining Open Questions — Resolved

### 12.1 Seam Carving Priority

**Decision: Build it now, as Phase 2.**

While seam carving is more niche than smart crop, the implementation is small (3–5 days), well-defined, and shares infrastructure with smart crop (Sobel energy map). Building it while we're already deep in content-aware image work avoids the context-switch cost of returning to it later. It may be a while before we revisit this area of the codebase.

### 12.2 Focal Point API

**Decision: Support both a simple normalised point and a region-of-interest rectangle.**

Two levels of the same feature, for different use cases:

```parsley
// Simple focal point — normalised coordinates (0–1), maps to a boost point
image(@./product.jpg, {width: 400, height: 300, crop: "smart", focal: {x: 0.3, y: 0.5}})

// Region of interest — normalised rectangle, maps to a boost region
image(@./product.jpg, {width: 400, height: 300, crop: "smart", focal: {x: 0.2, y: 0.3, w: 0.4, h: 0.5}})
```

**Normalised (0–1) rather than pixel-based** because:
- Users don't know the source image dimensions when writing templates
- Normalised coordinates are stable across image resizes/crops
- A point `{x, y}` is converted internally to a small boost region centred on that coordinate
- A rectangle `{x, y, w, h}` is used directly as a boost region
- Both feed into the same boost scoring pipeline that face detection uses — no separate codepath

The point form is the "just tell me where to focus" API. The rectangle form is the "I know exactly what region matters" API. Both are optional — if omitted, face detection + heuristics do their thing.

### 12.3 Quality Benchmarking

**Decision: Curated test image set with human-reviewed "golden" crops, plus a visual comparison workflow.**

There's no fully automated way to measure crop quality — it's inherently subjective. But we can make the process systematic:

1. **Curate a test set of ~30–50 images** covering the key scenarios:
   - Portraits (single person, various skin tones, various lighting)
   - Group photos (2–3 people, 5–10 people)
   - Landscapes with and without people
   - Product shots (plain and busy backgrounds)
   - Food photography
   - Animals
   - Edge cases (no clear subject, multiple competing subjects)

2. **For each image, define 2–3 target aspect ratios** (e.g., 1:1 square, 16:9 wide, 3:4 portrait) to test different crop shapes.

3. **Generate comparison grids** — for each test image, render side-by-side:
   - Center crop (baseline)
   - Smart crop (our result)
   - Optionally: manually placed "ideal" crop

4. **Human review** — you review the comparison grids and flag failures. Failures get added to a regression set with the expected crop rectangle recorded as the "golden" output.

5. **Automated regression** — once golden crops exist, `go test` can verify that the algorithm's output stays within a tolerance (e.g., IoU > 0.8 with the golden crop). This catches regressions during future changes without requiring human review every time.

Good sources for diverse, freely-licensed test images:
- [Unsplash](https://unsplash.com/) — high-quality, free license, diverse content
- [Pexels](https://www.pexels.com/) — similar to Unsplash
- The [FCDB dataset](http://personal.ie.cuhk.edu.hk/~ccloy/downloads_flickr_crop.html) (Flickr Cropping Dataset) — academic dataset with human-annotated "best crops" for ~31K images, used by the GenCrop paper and others for benchmarking

The FCDB dataset is particularly useful — it provides ground-truth crop annotations from professional photographers, which could give us an objective baseline for a subset of test images.

### 12.4 Analysis Resolution — Two-Pass Architecture

**Decision: Run face detection at 640px, heuristic scoring at 256px.**

The problem with a single 256px analysis pass is that PICO face detection needs faces to be at least ~30–40px for reliable detection. At 256px, a face that's 10% of the image height (e.g., one person in a group of 10) maps to just ~26px — right at the unreliable zone. A face at 5% (one in a crowd of 20) would be ~13px — completely undetectable.

Working backward from "reliably detect faces ≥ 10% of image height" with a 40px minimum:

| Face as % of Image | Face Size at 256px | Face Size at 640px | Detectable? |
|---|---|---|---|
| 20% (1–2 people) | ~51px ✓ | ~128px ✓ | Both |
| 10% (group of ~10) | ~26px ✗ | ~64px ✓ | 640px only |
| 7% (group of ~15) | ~18px ✗ | ~45px ✓ | 640px only |
| 5% (crowd of ~20) | ~13px ✗ | ~32px ~ | 640px marginal |

**The two-pass flow:**

1. **Pass 1: Face detection at 640px** — Decode image → resize to 640px on longest side → run PICO with `MinSize: 30, MaxSize: 640, ShiftFactor: 0.1, ScaleFactor: 1.1` → collect face rectangles → map coordinates back to original image space.

2. **Pass 2: Heuristic scoring at 256px** — Resize original to 256px on longest side → compute edge density, saturation, skin-tone, composition scoring → evaluate candidate crop rectangles → face boost regions (from Pass 1) dominate scoring where present.

**Why 640px and not larger?**
- At 640px, PICO runs in ~10–30ms — fast enough for a one-time cached operation
- Going to 1024px gives ~2.5× slower detection (quadratic in pixel count for the sliding window) for diminishing returns
- 640px reliably detects faces down to ~6% of image height, which covers group photos of ~15 people — beyond that, users should use the `focal` option
- The muesli/smartcrop library uses 400px as its prescale minimum; 640px is already more generous

**Why keep heuristic scoring at 256px?**
- Edge detection, saturation mapping, and composition scoring don't need pixel precision — they work on broad spatial patterns
- 256px is sufficient for smartcrop.js, which operates at this scale for all its heuristic analysis
- Keeping the scoring pass small means the candidate search (which evaluates many crop rectangles) stays fast

**Estimated cost for the two-pass pipeline** (on a 4000×3000 source image):

| Step | Time |
|---|---|
| Resize to 640px | ~1ms |
| PICO face detection at 640px | ~10–30ms |
| Resize to 256px | ~0.5ms |
| Heuristic scoring at 256px | ~2–5ms |
| Candidate search + scoring | ~2–5ms |
| **Total analysis** | **~15–40ms** |
| Full pipeline (load + analyse + crop + encode) | ~100–300ms |

All cached by Basil's existing image cache — the analysis cost is paid once per unique source image + crop target combination.

---

## 13. Final Phasing

### Phase 1: Smart Crop with Face Detection
*(~2 weeks)*

The primary deliverable. Build inside `server/images/smartcrop/`:

1. **Two-pass analysis architecture:**
   - Pass 1: PICO face detection at 640px analysis scale
   - Pass 2: Heuristic scoring at 256px analysis scale
2. **Sobel energy map** — shared utility, ~50 lines
3. **Heuristic scoring** — edge density, saturation, skin-tone, composition rules (GenCrop V1–V5) — ~200 lines
4. **PICO face detector** — reimplemented from paper, ~200 lines + 234 KB embedded cascade
5. **Boost integration** — face detections + optional focal point → weighted regions → scoring pipeline — ~50 lines
6. **Candidate search** — grid sample at 256px, score, pick best — ~100 lines
7. **Focal point option** — `focal: {x, y}` and `focal: {x, y, w, h}` in normalised (0–1) coordinates — ~30 lines
8. **Basil integration** — `{crop: "smart"}` in `TransformOptions`, wire into `Transform()` — ~50 lines
9. **Tests** — curated image set (~30–50 images), comparison grid generation, golden crop regression tests

### Phase 2: Seam Carving
*(3–5 days, immediately after Phase 1)*

Build inside `server/images/seamcarve/`:

1. Energy map (reuse Sobel from smart crop)
2. DP seam finding (backward energy)
3. Seam removal (width reduction only)
4. Basil integration (`{scale: "smart"}` in `TransformOptions`)
5. Tests + visual comparison

### Phase 3 (Future): Enhancements
*(if needed)*

- Forward energy for seam carving (if artefacts are visible)
- Seam insertion (enlargement)
- Object detection beyond faces
- Focal point auto-detection for non-portrait content

---

## 14. Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-03-16 | Pursue pure-Go, heuristic-based approach | Aligns with no-dependency philosophy |
| 2026-03-16 | ~~Build as standalone library~~ → Build inside Basil | Tight integration with existing `server/images/` pipeline; avoids separate project overhead; clean sub-package boundaries allow future extraction if needed |
| 2026-03-16 | ~~Start with seam carving~~ → Start with smart crop | Smart crop is already in the FEAT-148 spec; clearer user demand; shared Sobel utility is trivial to factor out |
| 2026-03-16 | ~~No face detection in v1~~ → Include from day one | Heuristic-only approach has significant, well-documented failure modes including skin-tone bias; PICO adds only ~200 lines + 234 KB |
| 2026-03-16 | Face detection is feasible to roll our own | PICO algorithm is ~200 lines of Go, cascade file is 234 KB MIT-licensed from Pigo |
| 2026-03-16 | Reimplement PICO runtime rather than importing Pigo | Core is trivially small; avoids dependency while keeping full control. Use paper as reference, not Pigo source. |
| 2026-03-16 | Bundle Pigo's trained cascade file (MIT) with attribution | Training our own cascade is unnecessary — the existing one is proven and freely redistributable. Add CASCADE_LICENSE alongside the binary. |
| 2026-03-16 | Use GenCrop paper findings as heuristic specification | The five violation criteria and composition rules are research-backed and implementable without ML |
| 2026-03-16 | Build from scratch, don't use muesli/smartcrop | We're going beyond what it offers (face detection, GenCrop rules); avoids inheriting skin-tone bias; the implementation is small enough that writing our own is comparable effort |
| 2026-03-16 | Seam carving: backward energy, reduction only | Forward energy and enlargement add complexity for marginal v1 benefit; easy to add later |
| 2026-03-16 | Non-face subjects handled by heuristic fallback + optional focal point | Heuristics work well for landscapes, food, products on plain backgrounds; focal point option is the escape hatch for edge cases; object detection is a future concern |
| 2026-03-16 | Build seam carving now (Phase 2), not deferred | Small effort (3–5 days), shares infrastructure, avoids context-switch cost of returning later |
| 2026-03-16 | Focal point API: normalised coordinates (0–1), support both point and rectangle | Users don't know source dimensions; normalised is stable across resizes; point converts to small boost region, rectangle used directly; both feed same boost pipeline as face detection |
| 2026-03-16 | Two-pass analysis: face detection at 640px, heuristic scoring at 256px | 256px is too coarse for PICO on group photos (faces < 30px); 640px detects faces down to ~6% of image height; heuristics don't need pixel precision so 256px is sufficient for scoring |
| 2026-03-16 | Quality benchmarking: curated test set + comparison grids + human review + golden crop regression tests | No fully automated quality metric; FCDB dataset provides academic ground truth for subset; golden crops enable automated regression testing after initial human review |