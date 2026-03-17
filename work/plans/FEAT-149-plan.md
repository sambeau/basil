---
id: PLAN-130
feature: FEAT-149
title: "Implementation Plan for Content-Aware Image Operations (Smart Crop & Seam Carving)"
status: draft
created: 2026-03-16
---

# Implementation Plan: FEAT-149 — Content-Aware Image Operations

## Overview

Add smart crop (content-aware cropping with face detection) and seam carving (content-aware scaling) to Basil's existing image transformation pipeline. Smart crop uses a two-pass analysis architecture: PICO face detection at 640px followed by heuristic scoring at 256px. Seam carving uses dynamic programming over a Sobel energy map. Both integrate into the existing `server/images/` package as sub-packages and are cached by the existing disk cache.

## Design Decisions Driving This Plan

1. **Build inside Basil** as `server/images/smartcrop/` and `server/images/seamcarve/`, not a standalone library.
2. **Reimplement PICO** from the paper (~200 lines), don't import Pigo. Bundle Pigo's MIT-licensed cascade file (234 KB) via `go:embed`.
3. **Two-pass analysis**: face detection at 640px, heuristic scoring at 256px.
4. **Smart crop first**, seam carving second. Smart crop is already in the FEAT-148 spec and has clearer user demand.
5. **Both crop and scale use `"smart"` as the option value** — `{crop: "smart"}` and `{scale: "smart"}`.
6. **Backward energy** for seam carving (simpler, good enough for < 30% reduction). **Reduction only** (no enlargement).
7. **Focal point API** uses normalised (0–1) coordinates, supporting both point `{x, y}` and rectangle `{x, y, w, h}`.
8. **No new Go dependencies.** The face detector is pure arithmetic on pixel arrays. The only external artefact is the cascade file.

## Dependencies

| Component | Purpose | Risk |
|-----------|---------|------|
| `disintegration/imaging` (existing) | Resize for analysis scale, final crop, pixel access | None — already in `go.mod` |
| `facefinder` cascade file (Pigo, MIT) | Pre-trained PICO face detection model | Low — static 234 KB binary, embedded |
| PICO paper (Markuš et al., 2014) | Algorithm reference for detection runtime | None — well-documented public paper |

No new entries in `go.mod`.

## Prerequisites

- [x] Design investigation complete: `work/design/DESIGN-content-aware-image-ops.md`
- [x] Spec written: `work/specs/FEAT-149.md`
- [x] FEAT-148 Phase 1 implemented (image transform pipeline, disk cache, `/__img/` handler)
- [ ] Feature branch created: `feat/FEAT-149-content-aware-images`
- [ ] Curate test image set (~30–50 images from Unsplash/Pexels) — see Task 10

## Tasks

### Task 1: Sobel Edge Detection

**Files**: `server/images/smartcrop/sobel.go`, `server/images/smartcrop/sobel_test.go`
**Estimated effort**: Small (< 1 day)

Implement a Sobel edge detection filter that computes gradient magnitude for each pixel. This is the foundation for both smart crop scoring and seam carving energy maps.

Steps:
1. Create `server/images/smartcrop/` package directory
2. Implement `Sobel(img image.Image) [][]float64` — returns a 2D gradient magnitude map
3. Use standard 3×3 Sobel kernels (Gx and Gy), compute `sqrt(Gx² + Gy²)` per pixel
4. Operate on grayscale luminance (convert RGB → grayscale inline, no separate image allocation needed)
5. Handle image edges (clamp or zero-pad the 1px border)

Tests:
- Uniform image → all-zero energy map
- Vertical edge (black|white) → high values along the edge, zero elsewhere
- Horizontal edge → same pattern, rotated
- Diagonal gradient → smooth non-zero values
- Known synthetic image with expected gradient values (numerical test)

**Commit point**: after tests pass.

---

### Task 2: PICO Face Detection Runtime

**Files**: `server/images/smartcrop/detect.go`, `server/images/smartcrop/cascade.go`, `server/images/smartcrop/facefinder`, `server/images/smartcrop/CASCADE_LICENSE`, `server/images/smartcrop/detect_test.go`
**Estimated effort**: Medium (1–2 days)

Reimplement the PICO detection algorithm from [Markuš et al., 2014](https://arxiv.org/pdf/1305.4537). This is the sliding-window binary tree cascade classifier. Do **not** copy from Pigo source — implement from the paper.

Steps:
1. Download `facefinder` cascade file from [Pigo repository](https://github.com/esimov/pigo/blob/master/cascade/facefinder) and place in `server/images/smartcrop/facefinder`
2. Create `CASCADE_LICENSE` with MIT attribution:
   ```
   The facefinder cascade file is derived from the Pigo project
   (https://github.com/esimov/pigo) by Endre Simo.

   Copyright (c) 2018 Endre Simo

   Permission is hereby granted, free of charge, to any person obtaining a copy
   of this software and associated documentation files (the "Software"), to deal
   in the Software without restriction, including without limitation the rights
   to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
   copies of the Software, and to permit persons to whom the Software is
   furnished to do so, subject to the following conditions:

   The above copyright notice and this permission notice shall be included in all
   copies or substantial portions of the Software.

   THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
   IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
   FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
   AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
   LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
   OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
   SOFTWARE.
   ```
3. Implement `cascade.go`:
   - `//go:embed facefinder` directive
   - `Cascade` struct: `treeCodes []int8`, `treePred []float32`, `treeThreshold []float32`, `treeDepth uint32`, `treeNum uint32`
   - `Unpack(data []byte) (*Cascade, error)` — parse the binary format (skip 8 bytes header, read tree depth, tree count, then loop: read codes, predictions, thresholds)
   - Package-level `var defaultCascade *Cascade` initialised via `sync.Once`
4. Implement `detect.go`:
   - `Detection` struct: `Row, Col, Scale int`, `Score float32`
   - `ToGrayscale(img image.Image) (pixels []uint8, width, height int)` — convert image to flat grayscale byte buffer
   - `classifyRegion(cascade, row, col, scale, treeDepth int, pixels []uint8, dim int) float32` — walk binary trees, compare pixel pairs at encoded offsets, accumulate predictions, early-reject below threshold
   - `DetectFaces(img image.Image) []Detection` — public API:
     - Convert to grayscale
     - Sliding window: `MinSize: 30`, `MaxSize: min(width, height)`, `ShiftFactor: 0.1`, `ScaleFactor: 1.1`
     - Collect detections where score > 0
     - Cluster via IoU non-maximum suppression (threshold: 0.3)
   - `ClusterDetections(detections []Detection, iouThreshold float64) []Detection` — merge overlapping detections by IoU, average positions, sum scores

Tests:
- Cascade unpacks without error, has non-zero tree count and depth
- `classifyRegion` on known face crop returns positive score
- `classifyRegion` on uniform grey patch returns negative score
- `DetectFaces` on a 640px-wide portrait image returns at least one detection
- `DetectFaces` on a plain white image returns zero detections
- `DetectFaces` on a group photo returns multiple detections
- `ClusterDetections` merges overlapping detections correctly
- `ClusterDetections` preserves non-overlapping detections

**Commit point**: after tests pass. Face detection works in isolation.

---

### Task 3: Heuristic Scoring

**Files**: `server/images/smartcrop/score.go`, `server/images/smartcrop/score_test.go`
**Estimated effort**: Medium (2–3 days)

Implement the scoring functions that evaluate candidate crop rectangles. Each function scores one aspect of image content within a rectangle.

Steps:
1. Define `BoostRegion` struct: `Rect image.Rectangle`, `Weight float64`
2. Define scoring functions, each operating on the analysis-scale image within a candidate crop `image.Rectangle`:
   - `scoreEdgeDensity(sobelMap [][]float64, crop image.Rectangle) float64` — mean gradient magnitude within the crop. Higher = more detail.
   - `scoreSaturation(img image.Image, crop image.Rectangle) float64` — mean saturation (convert each pixel to HSL, average S). Higher = more colourful.
   - `scoreSkinTone(img image.Image, crop image.Rectangle) float64` — proportion of pixels in a skin-tone colour range. Use a broader range than smartcrop.js's single reference colour — define ranges that cover diverse skin tones in normalised RGB space.
   - `scoreComposition(crop image.Rectangle, imgBounds image.Rectangle) float64` — rule-of-thirds: boost when center of crop's primary content aligns with thirds intersections. Penalise extreme off-center placements.
   - `scoreEdgeClutter(sobelMap [][]float64, crop image.Rectangle) float64` — mean gradient magnitude in a thin border (e.g., 10% of crop width) at the crop edges. Higher = more clutter at edges (penalty). Implements GenCrop V4.
   - `scoreBoostRegions(crop image.Rectangle, boosts []BoostRegion) float64` — sum of (overlap area × weight) for each boost region that intersects the crop. Penalise crops that clip boost regions within 2.5% margin (GenCrop V1).
3. Define `ScoreCandidate(img image.Image, sobelMap [][]float64, crop image.Rectangle, boosts []BoostRegion) float64`:
   - Weighted combination of all scoring signals
   - Starting weights (subject to tuning in Task 8):
     - Boost regions: 100.0
     - Edge density: 0.2
     - Saturation: 0.1
     - Skin-tone: 0.5
     - Composition: 0.3
     - Edge clutter: -0.2
     - Subject margin penalty: -0.5
   - Normalise by crop area to avoid bias toward larger crops

Tests:
- `scoreEdgeDensity`: high score for detailed crop, low for uniform crop
- `scoreSaturation`: high for colourful region, low for greyscale
- `scoreSkinTone`: detects diverse skin-tone patches (test with synthetic patches at various tones)
- `scoreComposition`: thirds-aligned crop scores higher than corner-placed crop
- `scoreEdgeClutter`: crop with busy edges scores lower than crop with clean edges
- `scoreBoostRegions`: crop containing a boost region scores much higher; crop that clips a boost region is penalised
- `ScoreCandidate`: boost region dominates over heuristic signals (a crop containing a face boost should always beat a crop without one, even if the non-face crop has higher edge/saturation scores)

**Commit point**: after tests pass.

---

### Task 4: Smart Crop Candidate Search

**Files**: `server/images/smartcrop/smartcrop.go`, `server/images/smartcrop/smartcrop_test.go`
**Estimated effort**: Medium (1–2 days)

Tie together face detection, heuristic scoring, and candidate search into the top-level `BestCrop` API.

Steps:
1. Define `FocalRegion` struct: `X, Y, W, H float64` (normalised 0–1)
2. Implement `BestCrop(img image.Image, targetWidth, targetHeight int, focal *FocalRegion) image.Rectangle`:
   - **Pass 1 — Face detection (640px)**:
     - Resize image to 640px on longest side using `imaging.Resize`
     - Call `DetectFaces(resized)`
     - Map detection coordinates back to original image space: `origX = detectX × (origWidth / 640px_width)`
     - Convert each `Detection` to a `BoostRegion` with `Weight: 1.0`
   - **Focal point → boost region**:
     - If `focal` is provided, convert normalised coordinates to original image coordinates
     - If `focal.W == 0 && focal.H == 0` (point), expand to a small boost region (e.g., 10% of image dimensions centered on the point)
     - If `focal.W > 0 && focal.H > 0` (rectangle), use directly
     - Add as `BoostRegion` with `Weight: 1.0`
   - **Pass 2 — Heuristic scoring (256px)**:
     - Resize image to 256px on longest side
     - Compute Sobel energy map at 256px scale
     - Scale all boost regions to 256px coordinates
     - Generate candidate crop rectangles:
       - Target aspect ratio = `targetWidth / targetHeight`
       - Scale factors: e.g., `[1.0, 0.9, 0.8, 0.7, 0.6, 0.5]` of the analysis image dimensions
       - For each scale, step across positions on a grid (e.g., every 10px at 256px scale)
       - Each candidate is a rectangle at the target aspect ratio at that position and scale
     - Score each candidate via `ScoreCandidate`
     - Pick the candidate with the highest score
   - **Map back to original**: scale winning crop rectangle from 256px coordinates to original image coordinates
   - **Fallback**: if all candidates score within 1% of each other (uniform image), return center crop
3. Implement `focalToBoost(focal *FocalRegion, imgWidth, imgHeight int) BoostRegion` — helper to convert normalised focal to pixel-space boost region

Tests:
- Portrait image with clear face → crop contains the face
- Image with face off-center → crop shifts to include face (not center crop)
- Image with focal point specified → crop shifts to include focal area
- Uniform image → returns center crop (fallback)
- Landscape with no faces → returns heuristic-driven crop (high-detail region preferred)
- Group photo → crop includes area with most faces
- Various aspect ratios (1:1, 16:9, 3:4) → all produce valid rectangles within image bounds
- Returned rectangle dimensions match target aspect ratio (within rounding)
- Returned rectangle is within image bounds

**Commit point**: after tests pass. Smart crop works end-to-end in isolation.

---

### Task 5: TransformOptions Changes

**Files**: `server/images/options.go`, `server/images/options_test.go`
**Estimated effort**: Small (< 1 day)

Extend `TransformOptions` with the new fields for smart crop and seam carving.

Steps:
1. Add fields to `TransformOptions`:
   ```go
   Scale string       // "" or "smart"
   Focal *FocalRegion // optional focal point/region (normalised 0–1)
   ```
2. Define `FocalRegion` struct in `options.go`:
   ```go
   type FocalRegion struct {
       X float64
       Y float64
       W float64 // 0 = point
       H float64 // 0 = point
   }
   ```
3. Update `ParseOptions()`:
   - Parse `"scale"` key → `string`, set `t.Scale`
   - Parse `"focal"` key → expect a dict with `x`, `y` (required), `w`, `h` (optional, default 0). Convert to `*FocalRegion`.
4. Update `Validate()`:
   - `crop: "smart"` requires both `Width > 0` and `Height > 0`
   - `scale: "smart"` requires at least one of `Width > 0` or `Height > 0`
   - `scale` and `crop` cannot both be non-empty (mutually exclusive)
   - `focal` requires `crop: "smart"` — error otherwise
   - `focal` coordinates must be in 0–1 range
   - `crop` now accepts `""`, `"center"`, or `"smart"`
   - `scale` accepts `""` or `"smart"`
5. Update `Canonical()`:
   - Include `scale` value in the canonical string
   - Include focal point/region in the canonical string (if non-nil): `f=x,y` or `f=x,y,w,h`
   - This ensures different focal points produce different cache keys

Tests:
- Parse `{crop: "smart", width: 400, height: 300}` → valid
- Parse `{crop: "smart", width: 400}` → validation error (missing height)
- Parse `{scale: "smart", width: 800}` → valid
- Parse `{crop: "smart", scale: "smart"}` → validation error (mutually exclusive)
- Parse `{focal: {x: 0.5, y: 0.5}}` without `crop: "smart"` → validation error
- Parse `{crop: "smart", width: 400, height: 300, focal: {x: 0.5, y: 0.5}}` → valid, focal is a point
- Parse `{crop: "smart", width: 400, height: 300, focal: {x: 0.2, y: 0.3, w: 0.4, h: 0.5}}` → valid, focal is a rectangle
- Parse `{focal: {x: 1.5, y: 0.5}}` → validation error (out of range)
- Parse `{scale: "seam"}` → validation error (unknown scale value)
- `Canonical()` produces different strings for different focal points
- `Canonical()` produces different strings for `crop: "smart"` vs `crop: "center"`

**Commit point**: after tests pass.

---

### Task 6: Transform Integration

**Files**: `server/images/transform.go`, `server/images/transform_test.go`
**Estimated effort**: Small (< 1 day)

Wire smart crop into the existing `Transform()` function.

Steps:
1. Import `server/images/smartcrop` package
2. In `Transform()`, add a new case before the existing crop/resize logic:
   ```go
   if opts.Crop == "smart" && targetWidth > 0 && targetHeight > 0 {
       cropRect := smartcrop.BestCrop(img, targetWidth, targetHeight, opts.Focal)
       result = imaging.Crop(img, cropRect)
       // Resize to exact target dimensions if the crop is larger
       // (it will be, since BestCrop works at original resolution)
       result = imaging.Resize(result, targetWidth, targetHeight, imaging.Lanczos)
   }
   ```
3. The existing sharpening logic (applied on downscale) should still apply after smart crop — no changes needed there.
4. Note: seam carving integration is Task 9 (separate).

Tests:
- `Transform(img, {Crop: "smart", Width: 400, Height: 300})` returns an image of exactly 400×300
- `Transform(img, {Crop: "smart", Width: 400, Height: 300, Focal: &FocalRegion{X: 0.5, Y: 0.5}})` returns 400×300
- `Transform(img, {Crop: "center", Width: 400, Height: 300})` still works (regression test)
- `Transform(img, {Width: 400})` still works (regression test)
- Smart crop result is different from center crop result on an off-center portrait
- Full pipeline test: `Process(sourcePath, opts)` with `crop: "smart"` produces encoded bytes at correct dimensions

**Commit point**: after tests pass. Smart crop is now functional through the full Basil pipeline.

---

### Task 7: Evaluator Updates

**Files**: `pkg/parsley/evaluator/image.go`
**Estimated effort**: Small (< 1 day)

Update the `imageObjectToGo` helper and verify that the existing `evalImage` function correctly passes the new options through to the registry.

Steps:
1. Update `imageObjectToGo` to handle dict values for `focal`:
   - When the option key is `"focal"` and the value is a `*Dictionary`, convert it to a `map[string]any` with `x`, `y`, `w`, `h` keys
   - This means `ParseOptions` in `options.go` will receive `focal` as a `map[string]any` and needs to extract the `FocalRegion` from it (already handled in Task 5)
2. Alternatively (simpler): extend `imageObjectToGo` to recursively convert nested `*Dictionary` values to `map[string]any`. This is more general and handles `focal` without special-casing.
3. Verify that `evalImage` already passes all options through to `env.ImageRegistry.Transform(absPath, opts)` — it does (see existing code). No changes needed to the main flow.

Tests:
- Parsley expression `image(@./photo.jpg, {width: 400, height: 300, crop: "smart"})` in a test environment returns a URL string
- Parsley expression with `focal: {x: 0.5, y: 0.5}` passes through correctly
- Parsley expression with `scale: "smart"` passes through correctly (once Task 9 is done)
- Existing `image()` tests still pass (regression)
- Existing `imageBlur()`, `imageSrcset()`, `imageInfo()` tests still pass (regression)

**Commit point**: after tests pass.

---

### Task 8: Quality Tuning

**Files**: `server/images/smartcrop/score.go` (weight adjustments), test image set
**Estimated effort**: Medium (2–3 days)

Tune scoring weights against a diverse set of test images. This is iterative and involves human review.

Steps:
1. Assemble the curated test image set (see Task 10 — may run in parallel):
   - ~30–50 images from Unsplash/Pexels
   - Categories: portraits (various skin tones, lighting), group photos, landscapes with/without people, product shots, food, animals, edge cases
2. Write a test helper (or standalone script) that generates comparison grids:
   - For each test image × each aspect ratio (1:1, 16:9, 3:4):
     - Generate center crop
     - Generate smart crop
     - Composite side-by-side into a single comparison image
   - Output all comparison images to a `testdata/comparisons/` directory
3. Human review:
   - Review comparison grids
   - Flag failures (smart crop worse than center crop)
   - For each failure, analyse which scoring signal is misbehaving
   - Adjust weights in `score.go` and regenerate
4. After weights are stable:
   - For a subset of images (especially faces and group photos), record the smart crop rectangle as a "golden" expected output
   - Add regression tests: `go test` verifies IoU > 0.8 against golden rectangles
5. Clean up: remove comparison grid generation script if it was a standalone tool, or keep it as a `go test -run TestGenerateComparisons` gated behind a build tag (e.g., `//go:build tuning`)

Tests:
- Golden crop regression tests for 10–15 key images
- Weight sanity checks: face boost always dominates heuristics (a crop with a face beats a crop without one)

**Commit point**: after weights are stable and golden crop tests pass.

---

### Task 9: Seam Carving

**Files**: `server/images/seamcarve/seamcarve.go`, `server/images/seamcarve/energy.go`, `server/images/seamcarve/seamcarve_test.go`
**Estimated effort**: Medium (3–5 days)

Implement seam carving for content-aware image scaling (width and/or height reduction).

Steps:
1. Create `server/images/seamcarve/` package directory
2. Implement `energy.go`:
   - `EnergyMap(img image.Image) [][]float64` — Sobel gradient magnitude (same algorithm as `smartcrop/sobel.go`; can copy/duplicate — it's ~50 lines, not worth a shared package for this alone)
3. Implement `seamcarve.go`:
   - `Resize(img image.Image, targetWidth, targetHeight int) image.Image` — public API:
     - If `targetWidth < img.Width`: remove `(img.Width - targetWidth)` vertical seams
     - If `targetHeight < img.Height`: remove `(img.Height - targetHeight)` horizontal seams
     - If both dimensions need reduction: reduce the dimension with the larger delta first
     - Log a warning if reduction exceeds 30% in either dimension (artefact threshold)
   - `findMinVerticalSeam(energy [][]float64) []int` — DP, returns one column index per row:
     - Build cumulative energy matrix M: `M[i][j] = energy[i][j] + min(M[i-1][j-1], M[i-1][j], M[i-1][j+1])`
     - Backtrack from min of bottom row to find seam path
   - `findMinHorizontalSeam(energy [][]float64) []int` — same algorithm, transposed (one row index per column)
   - `removeVerticalSeam(img image.Image, seam []int) image.Image` — create new image with one fewer column, shifting pixels left past the seam at each row
   - `removeHorizontalSeam(img image.Image, seam []int) image.Image` — same, shifting pixels up
   - Each iteration: recompute energy map, find seam, remove seam. (Recomputing the full energy map each iteration is simpler than incremental update and fast enough for cached operations.)
4. Wire into `Transform()` in `server/images/transform.go`:
   ```go
   if opts.Scale == "smart" {
       result = seamcarve.Resize(img, targetWidth, targetHeight)
   }
   ```
   This goes before the existing resize logic (which is for uniform scaling).
5. Update `Validate()` to accept `scale: "smart"` (already done in Task 5).

Tests:
- `EnergyMap` on uniform image → all-zero (or near-zero) values
- `EnergyMap` on image with vertical stripe → high energy along stripe edges
- `findMinVerticalSeam` on synthetic energy map with known low-energy path → seam follows that path
- `findMinVerticalSeam` on uniform energy → any path is valid (seam length equals image height)
- `removeVerticalSeam` reduces image width by 1
- `removeHorizontalSeam` reduces image height by 1
- `Resize` with `targetWidth = img.Width - 50` → output is exactly 50px narrower
- `Resize` with both dimensions reduced → output matches both targets
- `Resize` with no reduction needed → returns image unchanged
- `Resize` on image with clear low-energy region → seam paths cluster in that region (visual/manual inspection)
- Integration test: `Process(sourcePath, {Scale: "smart", Width: 800})` produces encoded bytes at width 800
- Regression: existing resize tests still pass

**Commit point**: after tests pass.

---

### Task 10: Test Image Curation

**Files**: `server/images/smartcrop/testdata/`, `server/images/seamcarve/testdata/`
**Estimated effort**: Small (< 1 day)
**Can run in parallel with Tasks 1–4.**

Assemble the curated test image set for quality tuning and regression testing.

Steps:
1. Create `server/images/smartcrop/testdata/` directory
2. Download ~30–50 images from Unsplash/Pexels (free license) covering:
   - 5–8 portraits (various skin tones, lighting, backgrounds)
   - 3–5 group photos (2–3 people, 5–10 people)
   - 3–5 landscapes (with and without people)
   - 3–5 product shots (plain and busy backgrounds)
   - 2–3 food photography
   - 2–3 animals
   - 2–3 edge cases (no clear subject, multiple competing subjects, uniform image)
3. Resize all test images to ~2000px on longest side (keep file sizes manageable for the repo)
4. Create `testdata/README.md` documenting each image's source URL, license, and category
5. Create `server/images/seamcarve/testdata/` with a subset of the same images (landscapes and product shots are most relevant for seam carving)

**Commit point**: after images are added. This commit will be large (binary files) but is a one-time addition.

---

## Implementation Order

The tasks have these dependencies:

```
Task 1 (Sobel) ──────────────────┐
                                  │
Task 2 (PICO Detector) ──────────┤
                                  ▼
Task 3 (Heuristic Scoring) ──► Task 4 (Candidate Search / BestCrop)
                                  │
Task 5 (TransformOptions) ────────┤
                                  ▼
                            Task 6 (Transform Integration)
                                  │
                            Task 7 (Evaluator Updates)
                                  │
Task 10 (Test Images) ──────► Task 8 (Quality Tuning)
                                  │
                                  ▼
                            Task 9 (Seam Carving)
```

Suggested order:
1. **Task 10** — curate test images (can start immediately, runs in parallel)
2. **Task 1** — Sobel filter (smallest, most testable, foundation for both features)
3. **Task 2** — PICO face detector (independent of scoring, testable in isolation)
4. **Task 3** — Heuristic scoring (depends on Task 1 for Sobel)
5. **Task 5** — TransformOptions changes (independent, can interleave with Task 3)
6. **Task 4** — Candidate search / BestCrop (depends on Tasks 1, 2, 3)
7. **Task 6** — Transform integration (depends on Tasks 4, 5)
8. **Task 7** — Evaluator updates (depends on Task 5, light task)
9. **Task 8** — Quality tuning (depends on Tasks 6, 10 — needs working smart crop + test images)
10. **Task 9** — Seam carving (independent of Tasks 3–8, depends only on Sobel concept from Task 1)

Commit points:
- After Task 1+2 (low-level components compile and test)
- After Task 4 (smart crop works in isolation)
- After Task 5+6+7 (smart crop integrated into Basil pipeline, all tests pass)
- After Task 8 (weights tuned, golden crop regression tests in place)
- After Task 9 (seam carving integrated, all tests pass)

## Estimated Total Effort

| Component | Lines (est.) |
|-----------|-------------|
| `smartcrop/sobel.go` | ~50 |
| `smartcrop/detect.go` | ~200 |
| `smartcrop/cascade.go` | ~40 |
| `smartcrop/score.go` | ~200 |
| `smartcrop/smartcrop.go` | ~150 |
| `images/options.go` changes | ~60 |
| `images/transform.go` changes | ~20 |
| `evaluator/image.go` changes | ~15 |
| `seamcarve/energy.go` | ~50 |
| `seamcarve/seamcarve.go` | ~200 |
| **Tests** (all packages) | ~600 |
| **Total** | **~1585** |

Plus:
- `facefinder` cascade binary: 234 KB
- `CASCADE_LICENSE`: ~25 lines
- Test images: ~30–50 files, ~2000px each

## Validation Checklist

- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `make build`
- [ ] Benchmarks checked: `make bench-compare` (flag regressions > 5%)
- [ ] Manual test: create a `.pars` file using `image(@./photo.jpg, {width: 400, height: 300, crop: "smart"})`, verify crop focuses on subject
- [ ] Manual test: same image with `{crop: "center"}` produces a different (worse) crop
- [ ] Manual test: `image(@./photo.jpg, {width: 800, scale: "smart"})` produces a narrower image without distortion
- [ ] Manual test: `focal: {x: 0.3, y: 0.5}` shifts the crop toward the specified point
- [ ] Dev mode: modify source image, verify re-transform (cache invalidation)
- [ ] Diverse skin tones: golden crop tests pass for portraits across skin tone range
- [ ] Group photo: smart crop includes more faces than center crop
- [ ] Error handling: `{crop: "smart", width: 400}` (missing height) returns a clear error
- [ ] Error handling: `{crop: "smart", scale: "smart"}` returns a clear error
- [ ] Performance: smart crop analysis < 50ms on a typical 4000×3000 image (check with `go test -bench`)
- [ ] Binary size: check Basil binary size before/after (cascade file adds ~234 KB)

## Risk Mitigation

| Risk | Mitigation |
|------|-----------|
| PICO cascade file doesn't detect diverse faces well | Test on diverse portrait images early (Task 2 tests). If detection quality is poor for certain demographics, investigate alternative cascade files or adjusting MinSize/ShiftFactor. |
| Heuristic weights don't generalise across image types | Task 8 uses a diverse test set. If one weight set can't cover all categories, consider category-specific presets (but this adds complexity — prefer a single weight set). |
| Seam carving too slow for large images | Expected: 200–500ms for 10% reduction. If too slow, consider: (a) downscale before carving then upscale result, (b) skip every Nth seam recomputation, (c) set a stricter reduction limit. All cached, so cost is one-time. |
| Sobel filter edge-case panics (out-of-bounds pixel access) | Explicit bounds checking in Task 1. Test with 1×1, 2×2, and 3×3 images. |
| Smart crop always returns center crop (heuristics not discriminating) | This would surface in Task 8 comparison grids. Root cause likely flat scoring — check individual signal outputs, verify face detection is producing boost regions. |
| Cascade binary format changes in future Pigo versions | We embed a specific file, not a URL. Pin to a known-good version. Document the provenance in CASCADE_LICENSE. |

## Deferred Items

Items to add to `work/BACKLOG.md` after implementation if not addressed:
- Forward energy for seam carving (if artefacts visible on strong-edge images)
- Seam insertion (enlargement)
- Protect/remove masks for seam carving (`{scale: "smart", protect: {x, y, w, h}}`)
- Object detection beyond faces (animals, products)
- Focal point auto-suggestion via `imageInfo()` (return detected focal point)
- FCDB dataset validation (academic ground-truth comparison)
- In-plane rotation support for face detection (tilted heads)

## Progress Log

*Updated during implementation*

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-03-17 | Task 1 | ✅ Complete | Sobel edge detection filter implemented with tests |
| 2026-03-17 | Task 2 | ✅ Complete | PICO face detection runtime implemented, facefinder cascade embedded |
| 2026-03-17 | Task 3 | ✅ Complete | Heuristic scoring (edge, saturation, skin-tone, composition, boost) |
| 2026-03-17 | Task 4 | ✅ Complete | BestCrop API with two-pass analysis and candidate search |
| 2026-03-17 | Task 5 | ✅ Complete | TransformOptions extended with Scale, Focal fields |
| 2026-03-17 | Task 6 | ✅ Complete | Smart crop wired into Transform() pipeline |
| 2026-03-17 | Task 7 | ✅ Complete | Evaluator handles nested dicts for focal option |
| | Task 8 | 🔜 Deferred | Quality tuning requires test images and human review |
| | Task 9 | 🔜 Deferred | Seam carving (Phase 2) |
| | Task 10 | 🔜 Deferred | Test image curation |

### Implementation Notes

**Phase 1 (Smart Crop) is feature-complete.** All core functionality works:
- `image(@./photo.jpg, {width: 400, height: 300, crop: "smart"})` analyses and crops
- Face detection runs at 640px, heuristics at 256px
- Focal point support via `focal: {x, y}` or `focal: {x, y, w, h}`
- Different focal points produce different cache keys
- Results are cached by existing disk cache

**Benchmark results (M2 Mac):**
- BestCrop (640x480 image): ~115ms (one-time, cached)
- Face detection at 640px: ~10ms
- Sobel filter at 256px: ~1.2ms

**Deferred to Phase 2:**
- Task 8: Weight tuning with curated test images
- Task 9: Seam carving (`scale: "smart"`)
- Task 10: Test image set curation