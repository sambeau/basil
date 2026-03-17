---
id: FEAT-149
title: "Content-Aware Image Operations (Smart Crop & Seam Carving)"
status: draft
priority: medium
created: 2026-03-16
author: "@human / @ai"
related: FEAT-148
---

# FEAT-149: Content-Aware Image Operations (Smart Crop & Seam Carving)

## Summary

Basil's image pipeline (FEAT-148) supports basic transforms: resize, crop-to-center, format conversion, sharpening, and blur placeholders. This feature adds two content-aware image operations: **smart crop** (analyse an image for high-interest regions — including faces — and crop to preserve them) and **seam carving** (resize by removing low-interest pixel paths rather than uniformly scaling). Both are implemented in pure Go with zero new external dependencies, built as sub-packages inside the existing `server/images/` pipeline.

## User Story

As a Basil developer, I want to write `image(@./photo.jpg, {width: 400, height: 300, crop: "smart"})` and have the crop automatically focus on the most important part of the image — especially faces — so that thumbnails and hero images look good without me manually specifying a crop region for every image.

As a Basil developer, I want to write `image(@./photo.jpg, {width: 800, scale: "smart"})` and have the image intelligently shrink by removing unimportant areas, so that I can change an image's aspect ratio without distorting the subject.

## Motivation

Center-cropping is the most common source of bad thumbnails on the web. A portrait with the subject off-center gets cropped to show a blank wall. A group photo loses half the people. A landscape cuts the horizon awkwardly. Every CMS user has experienced this.

Smart crop solves the 90% case: detect faces and high-interest regions, score candidate crops, pick the best one. The user writes `crop: "smart"` and gets a good result. For the remaining 10%, a `focal` option lets the user specify where to focus.

Seam carving addresses a different but related problem: changing an image's aspect ratio without distorting its subject. Rather than uniformly squishing pixels, seam carving removes connected paths of low-energy pixels — stretching or compressing the background while leaving the subject intact.

### Design Principles

- **Zero-config by default**: `{crop: "smart"}` should produce the right result without any configuration
- **Face-aware from day one**: heuristic-only cropping has well-documented failure modes, particularly around skin-tone bias — face detection is included, not deferred
- **Pure Go, no new dependencies**: the PICO face detector is reimplemented from the paper (~200 lines); the only external artefact is a 234 KB pre-trained cascade file (MIT-licensed, embedded via `go:embed`)
- **The Parsley aesthetic**: simple, minimal, complete, composable
- **Cached like everything else**: both operations are one-time costs per unique transform, cached by the existing disk cache

## Acceptance Criteria

### Phase 1: Smart Crop with Face Detection

#### 1a. Smart Crop Core

- [ ] `{crop: "smart"}` option in `image()` builtin: analyse the image and crop to the most interesting region
- [ ] Heuristic scoring pipeline evaluating candidate crop rectangles on four signals:
  - Edge density (Sobel filter)
  - Colour saturation
  - Skin-tone proximity (with awareness of diverse skin tones — not a single reference colour)
  - Composition rules (rule-of-thirds placement, subject margin, edge clutter avoidance — informed by GenCrop paper V1–V5 violation criteria)
- [ ] Two-pass analysis architecture:
  - Pass 1: face detection at 640px on longest side (reliable detection of faces ≥ 10% of image height)
  - Pass 2: heuristic scoring at 256px on longest side (fast candidate search)
- [ ] Candidate crop coordinates mapped back to original image space for full-resolution cropping
- [ ] Smart crop requires both `width` and `height` (an aspect ratio is needed to evaluate crops) — error if only one dimension is specified with `crop: "smart"`
- [ ] Falls back gracefully to center crop if analysis produces no clear winner (e.g., uniform image)

#### 1b. Face Detection (PICO)

- [ ] PICO face detection algorithm reimplemented from [Markuš et al., 2014](https://arxiv.org/pdf/1305.4537) — not copied from Pigo source
- [ ] Pigo's `facefinder` cascade file (234 KB, MIT-licensed) embedded via `go:embed` — zero runtime configuration
- [ ] `CASCADE_LICENSE` file alongside the cascade binary with full MIT attribution to Endre Simo / Pigo
- [ ] Face detections converted to boost regions in the scoring pipeline (heavily weighted — a detected face dominates heuristic signals)
- [ ] Non-maximum suppression (IoU clustering) to merge overlapping face detections
- [ ] PICO parameters: `MinSize: 30, MaxSize: 640, ShiftFactor: 0.1, ScaleFactor: 1.1`
- [ ] No pupil localisation, no facial landmarks, no rotation detection — only face bounding boxes are needed for cropping

#### 1c. Focal Point Option

- [ ] `focal: {x, y}` option: normalised coordinates (0–1), converted to a boost region centred on that point
- [ ] `focal: {x, y, w, h}` option: normalised rectangle, used directly as a boost region
- [ ] Focal point feeds the same boost scoring pipeline as face detection — no separate codepath
- [ ] Focal point is optional — if omitted, face detection + heuristics determine the crop
- [ ] Error if `focal` is specified without `crop: "smart"`

#### 1d. Integration

- [ ] `TransformOptions.Crop` accepts `"smart"` in addition to existing `"center"` and `""`
- [ ] `TransformOptions.Focal` field added (normalised point or rectangle)
- [ ] `ParseOptions()` updated to parse `focal` dict from Parsley options
- [ ] `Validate()` updated: `crop: "smart"` requires both width and height; `focal` requires `crop: "smart"`
- [ ] `Canonical()` updated to include focal point in cache key
- [ ] `Transform()` dispatches to smart crop when `opts.Crop == "smart"`
- [ ] Cached by existing `CacheKey()` mechanism — no caching changes needed
- [ ] Works with all existing output formats (JPEG, PNG, WebP, GIF)

#### 1e. Testing

- [ ] Curated test image set (~30–50 images) covering: portraits (various skin tones), group photos, landscapes, product shots, food, animals, edge cases
- [ ] Comparison grid generation: center crop vs smart crop for 2–3 target aspect ratios per test image
- [ ] Human-reviewed "golden" crop rectangles for regression testing (IoU > 0.8 tolerance)
- [ ] Unit tests for PICO face detector: known face images produce detections, non-face images produce none
- [ ] Unit tests for Sobel filter, heuristic scoring functions
- [ ] Integration test: `image(@./photo.jpg, {width: 400, height: 300, crop: "smart"})` produces a cached, serveable image
- [ ] Integration test: `focal` option shifts crop toward the specified point

### Phase 2: Seam Carving

#### 2a. Seam Carving Core

- [ ] `{scale: "smart"}` option in `image()` builtin: resize by removing low-energy seams
- [ ] Sobel energy map computation (shared with smart crop)
- [ ] Dynamic programming seam finder: minimum-energy vertical seam (top to bottom) and horizontal seam (left to right)
- [ ] Backward energy function (gradient magnitude of pixels being removed)
- [ ] Width reduction: iteratively remove vertical seams until target width is reached
- [ ] Height reduction: iteratively remove horizontal seams until target height is reached
- [ ] Width+height reduction: reduce in both dimensions (reduce the larger delta first)
- [ ] No enlargement (seam insertion) in v1 — Basil already refuses to upscale
- [ ] Graceful limits: warn or fall back to standard resize if reduction exceeds 30% in either dimension (artefact threshold)

#### 2b. Integration

- [ ] `TransformOptions.Scale` field added (accepts `"smart"` or `""`)
- [ ] `ParseOptions()` updated to parse `scale` option
- [ ] `Validate()` updated: `scale: "smart"` requires at least one dimension specified; cannot be combined with `crop`
- [ ] `Canonical()` updated to include scale mode in cache key
- [ ] `Transform()` dispatches to seam carving when `opts.Scale == "smart"`
- [ ] Cached by existing `CacheKey()` mechanism

#### 2c. Testing

- [ ] Unit tests for energy map computation
- [ ] Unit tests for seam finding: verify seam follows low-energy path on a known synthetic image
- [ ] Visual comparison tests: seam-carved results vs standard resize on 10+ diverse images
- [ ] Integration test: `image(@./photo.jpg, {width: 800, scale: "smart"})` produces a cached image at the correct dimensions
- [ ] Test that `scale: "smart"` + `crop: "smart"` produces a validation error

### Phase 3: Nice to Have

- [ ] Forward energy for seam carving (Rubinstein et al., 2008) — reduces artefacts on strong edges
- [ ] Seam insertion (enlargement) — duplicate low-energy seams to grow an image
- [ ] Protect/remove masks for seam carving: `{scale: "smart", protect: {x, y, w, h}}` to mark regions that should never be carved
- [ ] Object detection beyond faces (would require a different cascade or approach)
- [ ] Focal point auto-suggestion: `imageInfo(@./photo.jpg)` returns a `focal` field with the detected best focal point

## Design Decisions

### Reimplemented PICO, Not Pigo as Dependency

**Decision**: Reimplement the PICO detection runtime from the [paper](https://arxiv.org/pdf/1305.4537) (~200 lines), rather than importing `esimov/pigo` as a Go dependency.

**Rationale**: Pigo's `core/` package has zero external dependencies and is only ~280 lines, so importing it would work. But at ~200 lines for the parts we need, reimplementing from the paper is barely more work and gives us full control: no rotation detection, no pupil localisation, no landmarks — just face bounding boxes for crop boost regions. The algorithm is trivially simple (binary tree walks comparing pixel intensities). We bundle Pigo's MIT-licensed cascade file for the trained model data, which is the genuinely hard part to reproduce.

### Build Inside Basil, Not Standalone Library

**Decision**: Build as sub-packages inside `server/images/` rather than a separate repository.

**Rationale**: `{crop: "smart"}` slots into the same `Transform()` function that handles `{crop: "center"}`. The face detector is purpose-built for cropping boost regions, not a general-purpose library. For ~500–800 lines of total code, a separate repo creates more overhead (versioning, CI, `go.mod` coordination) than it removes. Sub-package boundaries (`server/images/smartcrop/`, `server/images/seamcarve/`) are clean enough to extract later if needed.

### Built from Scratch, Not muesli/smartcrop

**Decision**: Build the smart crop scorer from scratch rather than using `muesli/smartcrop`.

**Rationale**: We're going beyond what `muesli/smartcrop` offers — it implements heuristic-only scoring (no face detection) and inherits smartcrop.js's single-reference-colour skin-tone bias. Building from scratch lets us incorporate PICO face detection, GenCrop composition rules, and design the skin-tone heuristic with diverse skin tones in mind from the start. The implementation is small enough (~200 lines of scoring) that writing our own is comparable effort to understanding, integrating, and extending someone else's code.

### Two-Pass Analysis Architecture

**Decision**: Run face detection at 640px and heuristic scoring at 256px, not a single resolution for both.

**Rationale**: PICO needs faces to be at least ~30–40px for reliable detection. At 256px, a face at 10% of image height (one person in a group of 10) maps to ~26px — unreliable. At 640px the same face is ~64px — reliable. Heuristics (edge density, saturation, composition) don't need pixel precision, so 256px is sufficient for scoring. The two-pass approach costs ~15–40ms total, well within budget for a one-time cached operation.

### Face Detection Included from Day One

**Decision**: Ship smart crop with PICO face detection from the start, not as a later enhancement.

**Rationale**: Heuristic-only smart cropping has well-documented failure modes — particularly the skin-tone reference colour bias (tuned to medium-light Caucasian skin in smartcrop.js), which is both a quality and inclusivity problem. Face detection sidesteps this entirely by detecting faces via geometry, not colour. Adding PICO is only ~3–5 days of extra work and ~200 lines + 234 KB. The boost architecture keeps face detection cleanly separated from heuristic scoring.

### Seam Carving: Backward Energy, Reduction Only

**Decision**: Implement backward energy (not forward energy) and width/height reduction only (no enlargement) in v1.

**Rationale**: Backward energy is simpler (the energy map is the same Sobel gradient used for smart crop scoring) and produces good results for modest reductions (< 20–30%). Forward energy adds ~50 lines and changes the DP recurrence — worth doing if visual testing reveals artefacts on strong-edge images, but not needed for v1. Enlargement (seam insertion) has a narrow use case in web image serving (Basil already refuses to upscale), produces lower-quality results, and roughly doubles the implementation surface.

### Focal Point API: Normalised Coordinates

**Decision**: Focal points use normalised (0–1) coordinates, supporting both a point `{x, y}` and a rectangle `{x, y, w, h}`.

**Rationale**: Users don't know source image dimensions when writing templates. Normalised coordinates are stable across image resizes. A point is converted internally to a small boost region; a rectangle is used directly. Both feed the same boost scoring pipeline as face detection — no separate codepath. The point form is the "just tell me where to focus" API; the rectangle form is the "I know exactly what region matters" API.

### User-Facing Option Name: `"smart"` for Both Crop and Scale

**Decision**: Use `"smart"` as the user-facing option value for both `crop: "smart"` (smart crop) and `scale: "smart"` (seam carving), even though the underlying algorithms are completely different.

**Rationale**: Users don't know what "seam carving" is — `scale: "seam"` is a leaky abstraction that exposes an implementation detail. `scale: "smart"` communicates the intent: "resize this image intelligently." The two options are mutually exclusive (`crop` and `scale` cannot be combined), so there is no ambiguity — `crop: "smart"` means content-aware cropping, `scale: "smart"` means content-aware scaling. Internally, the package name remains `seamcarve/` and all algorithm references remain "seam carving" — only the user-facing option string changes.

### GenCrop Paper Findings as Heuristic Specification

**Decision**: Use the five violation criteria from the [GenCrop paper](https://arxiv.org/abs/2312.12080) (AAAI 2024) as a specification for composition scoring heuristics.

**Rationale**: The paper's ML pipeline (Stable Diffusion, BLIP-2, YOLOv8, 24.9M-parameter model) is far too heavy for our use case, but its analysis of what makes a professional crop is research-backed and directly actionable as heuristics: don't cut through joints (2.5% margin), get negative space right, avoid edge clutter, respect rule-of-thirds, maintain balance. These rules are cheap to implement and validated against real professional photography data.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Architecture Overview

```
image(@./photo.jpg, {width: 400, height: 300, crop: "smart"})
  │
  ▼
ParseOptions()  →  TransformOptions{Crop: "smart", Width: 400, Height: 300}
  │
  ▼
Transform(img, opts)
  │
  ├─ crop == "smart" ──▶ smartcrop.BestCrop(img, 400, 300, boosts)
  │                         │
  │                         ├─ Pass 1: Resize to 640px → PICO face detect
  │                         │          → face rectangles → boost regions
  │                         │
  │                         ├─ Pass 2: Resize to 256px → Sobel + saturation
  │                         │          + skin-tone + composition scoring
  │                         │
  │                         ├─ Score candidate crops (grid search)
  │                         │   weighted by: boosts (100×) + heuristics
  │                         │
  │                         └─ Return best crop.Rectangle
  │                         │
  │                    imaging.Crop(img, bestCrop)
  │
  ├─ scale == "smart" ──▶ seamcarve.Resize(img, 800, 0)
  │                         │
  │                         ├─ Compute Sobel energy map
  │                         ├─ DP: find minimum-energy seam
  │                         ├─ Remove seam
  │                         └─ Repeat until target width
  │
  └─ (existing paths: center crop, fit, resize)
```

### Affected Components

| File | Change | Description |
|------|--------|-------------|
| `server/images/options.go` | **MODIFY** | Add `Scale`, `Focal` fields to `TransformOptions`; update `ParseOptions()`, `Validate()`, `Canonical()` to handle `crop: "smart"`, `scale: "smart"`, `focal` dict |
| `server/images/options_test.go` | **MODIFY** | Tests for new options parsing, validation, canonical key generation |
| `server/images/transform.go` | **MODIFY** | Add smart crop and seam carving dispatch in `Transform()` |
| `server/images/transform_test.go` | **MODIFY** | Integration tests for smart crop and seam carving transforms |
| `server/images/smartcrop/smartcrop.go` | **NEW** | `BestCrop(img, width, height, opts)` — top-level API, two-pass analysis, candidate search |
| `server/images/smartcrop/score.go` | **NEW** | Heuristic scoring: edge density, saturation, skin-tone, composition, boost regions |
| `server/images/smartcrop/sobel.go` | **NEW** | Sobel edge detection filter (~50 lines) |
| `server/images/smartcrop/detect.go` | **NEW** | PICO face detection runtime: cascade unpacking, `classifyRegion`, `RunCascade`, `ClusterDetections` (~200 lines) |
| `server/images/smartcrop/cascade.go` | **NEW** | Cascade file loader with `go:embed` (~40 lines) |
| `server/images/smartcrop/facefinder` | **NEW** | Embedded PICO cascade binary (234 KB, from Pigo, MIT-licensed) |
| `server/images/smartcrop/CASCADE_LICENSE` | **NEW** | MIT attribution for cascade file (Endre Simo / Pigo) |
| `server/images/smartcrop/smartcrop_test.go` | **NEW** | Scoring tests, face detection tests, integration tests, golden crop regression tests |
| `server/images/seamcarve/seamcarve.go` | **NEW** | `Resize(img, width, height)` — seam finding + removal loop |
| `server/images/seamcarve/energy.go` | **NEW** | Sobel energy map for seam carving (same algorithm as smartcrop, separate instance) |
| `server/images/seamcarve/seamcarve_test.go` | **NEW** | Energy map tests, seam path tests, visual comparison tests |

### New Package: `server/images/smartcrop/`

```
server/images/smartcrop/
├── smartcrop.go        # BestCrop() — two-pass analysis + candidate search
├── score.go            # ScoreCandidate() — heuristic + boost scoring
├── sobel.go            # Sobel() — edge detection filter
├── detect.go           # DetectFaces() — PICO runtime
├── cascade.go          # loadCascade() — go:embed + binary unpacking
├── facefinder          # pre-trained cascade (234 KB)
├── CASCADE_LICENSE     # MIT notice (Endre Simo / Pigo)
└── smartcrop_test.go
```

### New Package: `server/images/seamcarve/`

```
server/images/seamcarve/
├── seamcarve.go        # Resize() — iterative seam removal
├── energy.go           # EnergyMap() — Sobel gradient magnitude
└── seamcarve_test.go
```

### TransformOptions Changes

```go
// Updated TransformOptions (additions shown)
type TransformOptions struct {
	Width           int          // (existing)
	Height          int          // (existing)
	Crop            string       // (existing — add "smart")
	Quality         int          // (existing)
	Format          string       // (existing)
	Sharpen         float64      // (existing)
	SharpenDisabled bool         // (existing)
	Scale           string       // NEW: "" or "smart"
	Focal           *FocalRegion // NEW: optional focal point/region
}

// FocalRegion represents a user-specified region of interest in normalised (0–1) coordinates.
// If W and H are zero, it's a point (expanded to a small boost region internally).
// If W and H are non-zero, it's a rectangle used directly as a boost region.
type FocalRegion struct {
	X float64 // 0–1, left to right
	Y float64 // 0–1, top to bottom
	W float64 // 0–1, width (0 = point)
	H float64 // 0–1, height (0 = point)
}
```

### Smart Crop Internal Types

```go
// BoostRegion is a weighted rectangular region in original image coordinates.
// Face detections and focal points are both converted to BoostRegions.
type BoostRegion struct {
	Rect   image.Rectangle
	Weight float64
}

// CropCandidate is a scored candidate crop rectangle.
type CropCandidate struct {
	Rect  image.Rectangle
	Score float64
}

// BestCrop analyses the image and returns the best crop rectangle.
func BestCrop(img image.Image, targetWidth, targetHeight int, focal *FocalRegion) image.Rectangle
```

### PICO Detector Internal Types

```go
// Cascade holds the unpacked decision tree data from the cascade file.
type Cascade struct {
	treeCodes     []int8
	treePred      []float32
	treeThreshold []float32
	treeDepth     uint32
	treeNum       uint32
}

// Detection represents a single face detection.
type Detection struct {
	Row   int
	Col   int
	Scale int
	Score float32
}

// DetectFaces runs the PICO cascade on a grayscale image and returns clustered detections.
func DetectFaces(gray []uint8, rows, cols int) []Detection
```

### Builtin Signatures

#### `image(path, options)` — updated

Existing signature, new options accepted:

```parsley
// Smart crop — requires both width and height
image(@./photo.jpg, {width: 400, height: 300, crop: "smart"})

// Smart crop with focal point
image(@./photo.jpg, {width: 400, height: 300, crop: "smart", focal: {x: 0.3, y: 0.5}})

// Smart crop with focal region
image(@./photo.jpg, {width: 400, height: 300, crop: "smart", focal: {x: 0.2, y: 0.3, w: 0.4, h: 0.5}})

// Seam carving — width reduction
image(@./photo.jpg, {width: 800, scale: "smart"})

// Seam carving — both dimensions
image(@./photo.jpg, {width: 800, height: 500, scale: "smart"})
```

### Two-Pass Analysis Detail

| Pass | Resolution | Purpose | Time |
|------|-----------|---------|------|
| Face detection | 640px longest side | PICO cascade: `MinSize: 30, MaxSize: 640, ShiftFactor: 0.1, ScaleFactor: 1.1` | ~10–30ms |
| Heuristic scoring | 256px longest side | Edge density + saturation + skin-tone + composition scoring over candidate crop grid | ~2–5ms |

Face rectangles from Pass 1 are mapped to original image coordinates (`× originalSize/640`), then scaled down to 256px coordinates for Pass 2 scoring as boost regions.

#### Face detection reliability by resolution

| Face as % of Image | Face Size at 640px | Detectable? |
|---|---|---|
| 20% (1–2 people) | ~128px | ✓ reliable |
| 10% (group of ~10) | ~64px | ✓ reliable |
| 7% (group of ~15) | ~45px | ✓ reliable |
| 5% (crowd of ~20) | ~32px | ~ marginal |

### Scoring Heuristics (from GenCrop research)

The five violation criteria from the GenCrop paper, implemented as scoring penalties:

1. **V1 — Subject margin**: penalise crops that cut within 2.5% of a boost region's bounds (don't crop through joints/extremities)
2. **V2 — Context preservation**: penalise crops that bisect secondary high-interest regions at the frame edge
3. **V3 — Negative space balance**: penalise crops that are too tight around the subject or that place the subject too far from center/thirds
4. **V4 — Edge clutter**: penalise crops with high-energy content at the frame edges (partial objects are distracting)
5. **V5 — Composition balance**: boost crops where the primary interest region aligns with rule-of-thirds intersections

### Scoring Weights

Approximate starting weights (subject to tuning during implementation):

| Signal | Weight | Notes |
|--------|--------|-------|
| Face boost region | 100.0 | Dominant — a detected face overrides heuristics |
| Focal point boost | 100.0 | Same pipeline as face detection |
| Edge density | 0.2 | Detail/structure signal |
| Saturation | 0.1 | Colour interest signal |
| Skin-tone | 0.5 | Lower than smartcrop.js's 1.8 — faces are detected explicitly, so skin-tone is a secondary signal |
| Composition (V3/V5) | 0.3 | Rule-of-thirds, negative space |
| Edge clutter penalty (V4) | -0.2 | Penalise partial objects at crop edges |
| Subject margin penalty (V1) | -0.5 | Penalise crops that clip boost regions |

### Seam Carving Algorithm

```
function Resize(img, targetWidth, targetHeight):
    while img.width > targetWidth:
        energy = SobelEnergyMap(img)
        seam = FindMinVerticalSeam(energy)    // DP, top to bottom
        img = RemoveVerticalSeam(img, seam)

    while img.height > targetHeight:
        energy = SobelEnergyMap(img)
        seam = FindMinHorizontalSeam(energy)  // DP, left to right
        img = RemoveHorizontalSeam(img, seam)

    return img

function FindMinVerticalSeam(energy):
    // DP: M[i][j] = energy[i][j] + min(M[i-1][j-1], M[i-1][j], M[i-1][j+1])
    // Backtrack from min of bottom row to find seam path
    // Returns []int — one column index per row
```

### Performance Expectations

| Operation | Expected Cost | Cacheable? |
|-----------|--------------|------------|
| Smart crop — full pipeline (detect + score + crop + encode) | ~100–300ms | Yes |
| Smart crop — analysis only (detect + score) | ~15–40ms | Yes (part of pipeline) |
| Seam carving — 10% width reduction on 2000px image | ~200–500ms | Yes |
| Seam carving — 30% width reduction on 2000px image | ~500–1500ms | Yes |

All operations are one-time costs per unique transform, cached by Basil's existing disk cache. Subsequent requests for the same transform serve the cached file directly.

### Security

- **Path traversal**: same rules as existing `image()` builtin — path must be within handler root
- **Cascade file**: embedded via `go:embed`, not loaded from user-supplied path
- **Resource limits**: existing `MaxSourceSize` (50MB) and `WarnSourceSize` (10MB) apply to smart crop and seam carving inputs
- **Seam carving DoS**: extreme reductions (> 30%) should warn or fall back to standard resize, as seam carving is O(width × height × seams_removed)
- **No user-supplied cascade files**: the embedded `facefinder` cascade is the only supported model

### Edge Cases & Constraints

1. **No faces detected** — falls back to heuristic-only scoring (edge + saturation + composition). Still better than center crop for most images.
2. **Uniform image (solid colour, gradient)** — all candidate crops score similarly. Falls back to center crop. This is correct behaviour.
3. **Very small source image** (< 256px) — skip downscaling, run analysis at original resolution. PICO may not detect faces below 30px, but the image is small enough that center crop is acceptable.
4. **Source image smaller than target crop** — existing behaviour applies: clamp target dimensions to source, no upscaling.
5. **`crop: "smart"` with only width or only height** — validation error. Smart crop needs an aspect ratio to evaluate candidate rectangles.
6. **`scale: "smart"` with `crop` specified** — validation error. These are mutually exclusive operations.
7. **`focal` without `crop: "smart"`** — validation error. Focal points only make sense in the context of smart crop scoring.
8. **Seam carving > 30% reduction** — log a warning, proceed with seam carving but results may have visible artefacts. Do not silently fall back (the user explicitly requested seam carving).
9. **GIF input with smart crop** — works on the first frame only (same as existing resize behaviour).
10. **Concurrent smart crop requests for same image** — existing `singleflight` deduplication in the registry handles this.

### Dependencies

- **Depends on**: FEAT-148 (image transformation pipeline, `server/images/`, `TransformOptions`, disk cache, `/__img/` handler)
- **Blocks**: Nothing — this is additive
- **Related**: FEAT-148 Phase 3 (`{crop: "smart"}` was deferred there pending investigation); Backlog item #132 (smart crop)

## Implementation Notes

*Added during/after implementation*

### Suggested Implementation Order

1. **Sobel filter** (`smartcrop/sobel.go`) — smallest, most testable component, shared by both features
2. **PICO face detector** (`smartcrop/detect.go`, `smartcrop/cascade.go`) — independent of scoring, testable in isolation
3. **Heuristic scoring** (`smartcrop/score.go`) — edge, saturation, skin-tone, composition
4. **Smart crop candidate search** (`smartcrop/smartcrop.go`) — ties together detection + scoring + boost pipeline
5. **Basil integration for smart crop** — `TransformOptions` changes, `Transform()` dispatch
6. **Quality tuning** — curated test images, weight adjustment, golden crop regression tests
7. **Seam carving core** (`seamcarve/energy.go`, `seamcarve/seamcarve.go`) — energy map + DP + seam removal
8. **Basil integration for seam carving** — `TransformOptions` changes, `Transform()` dispatch
9. **Seam carving tests** — visual comparison, dimension verification

### Cascade File Provenance

The `facefinder` cascade binary is sourced from the [Pigo](https://github.com/esimov/pigo) repository by Endre Simo, licensed under the MIT License. The file encodes a pre-trained PICO decision tree cascade for frontal face detection. It was trained using the PICO training framework on academic face datasets. We redistribute this file under its MIT license terms with full attribution in `CASCADE_LICENSE`.

We do **not** copy any of Pigo's Go source code. The detection runtime is reimplemented from the [original PICO paper](https://arxiv.org/pdf/1305.4537) (Markuš et al., 2014). The cascade binary format is simple (flat `int8` codes + `float32` predictions/thresholds) and documented in the paper.

### Quality Benchmarking Approach

1. Curate ~30–50 test images from Unsplash/Pexels covering diverse content types and skin tones
2. Generate comparison grids: center crop vs smart crop at 1:1, 16:9, 3:4 aspect ratios
3. Human review flags failures → record "golden" crop rectangle as expected output
4. `go test` regression: verify IoU > 0.8 against golden crops
5. Optionally validate against subset of [FCDB dataset](http://personal.ie.cuhk.edu.hk/~ccloy/downloads_flickr_crop.html) (academic ground truth)

## Related

- Plan: `work/plans/FEAT-149-plan.md` (PLAN-130) — Implementation plan
- Feature spec: `work/specs/FEAT-148.md` — Image Transformation and Caching (this feature extends Phase 3 smart crop)
- Design investigation: `work/design/DESIGN-content-aware-image-ops.md` — full exploration, feasibility analysis, face detection research, all design decisions
- Design investigation (Phase 2): `work/design/DESIGN-image-transform-phase2.md` — Phase 2 image enhancements where smart crop was first identified and deferred
- Backlog: `work/BACKLOG.md` item #132 — smart crop
- PICO paper: [Object Detection with Pixel Intensity Comparisons Organized in Decision Trees](https://arxiv.org/pdf/1305.4537) — face detection algorithm
- GenCrop paper: [Learning Subject-Aware Cropping by Outpainting Professional Photos](https://arxiv.org/abs/2312.12080) — composition heuristic specification
- Seam carving: [Seam Carving for Content-Aware Image Resizing](https://en.wikipedia.org/wiki/Seam_carving) — Avidan & Shamir, 2007
- Pigo: [github.com/esimov/pigo](https://github.com/esimov/pigo) — MIT-licensed cascade file source
- Existing implementation: `server/images/transform.go`, `server/images/options.go`
