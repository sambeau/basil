# Smart Crop Test Images

This directory contains test images for tuning and validating the smart crop algorithm (FEAT-149).

## Directory Structure

```
smartcrop/
├── faces/          # Portrait and group photos (8-10 images)
├── landscapes/     # Scenic images with horizons/focal points (6-8 images)
├── objects/        # Product shots and centered subjects (4-6 images)
└── edge-cases/     # Challenging images for algorithm stress testing (6-8 images)
```

## Image Requirements

### General
- **Format**: JPEG preferred (smaller size, representative of real-world usage)
- **Resolution**: At least 800x600px; ideally 1200-2000px on longest side
- **Total count**: 25-50 images across all categories

### By Category

#### `faces/`
Images featuring human faces for testing face detection and portrait cropping.

- `face-centered.jpg` — Single face, centered in frame
- `face-offcenter.jpg` — Single face, rule-of-thirds positioning
- `face-group.jpg` — Multiple faces (3-5 people)
- `face-small.jpg` — Small face in larger scene (person in landscape)
- `face-diverse-*.jpg` — Faces with diverse skin tones (critical for bias testing)

#### `landscapes/`
Scenic images for testing composition scoring and edge detection.

- `landscape-horizon.jpg` — Clear horizon line
- `landscape-focal.jpg` — Strong focal point (lighthouse, lone tree, etc.)
- `landscape-busy.jpg` — Cluttered scene with many elements
- `landscape-minimal.jpg` — Minimalist scene with single subject

#### `objects/`
Product-style shots for testing subject detection without faces.

- `object-centered.jpg` — Centered subject, clean background
- `object-edge.jpg` — Subject near frame edge
- `object-contrast.jpg` — High contrast subject
- `object-multiple.jpg` — Multiple objects in frame

#### `edge-cases/`
Challenging images to stress-test the algorithm.

- `edge-texture.jpg` — Highly textured image (fabric, foliage)
- `edge-vertical-lines.jpg` — Strong vertical lines (architecture, trees)
- `edge-low-contrast.jpg` — Low contrast, muted colors
- `edge-uniform.jpg` — Nearly uniform image (sky, wall)
- `edge-center-bad.jpg` — Image where center crop would fail badly

## Sourcing Images

### Recommended Sources (Free, Permissive Licenses)

1. **Unsplash** (https://unsplash.com) — High quality, free for commercial use
2. **Pexels** (https://pexels.com) — Similar to Unsplash
3. **Your own photos** — No licensing concerns

### Naming Convention

Use descriptive, lowercase names with hyphens:
```
category-description.jpg
```

Examples:
- `face-woman-laughing.jpg`
- `landscape-mountain-sunset.jpg`
- `object-coffee-cup.jpg`

### Important: Diverse Representation

For face detection testing, ensure diversity in:
- Skin tones (light, medium, dark)
- Age ranges
- Lighting conditions
- Face angles (frontal, 3/4 profile)

This is critical for detecting bias in the face detection algorithm.

## Usage

After adding images, run the comparison grid generator:

```bash
go run ./scripts/smartcrop-compare/main.go
```

This generates side-by-side comparisons in `testdata/images/smartcrop/output/` (gitignored).

## Output Directory

The `output/` subdirectory is gitignored and contains:
- Comparison grids (center vs smart crop)
- Debug visualizations (Sobel maps, face detection boxes)
- Golden crop annotations (after human review)

## License

All test images should be:
- Your own work, OR
- Licensed under a permissive license (Unsplash, Pexels, CC0, etc.)

Do NOT commit copyrighted images without proper licensing.