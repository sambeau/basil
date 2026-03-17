---
id: man-bas-images
title: Images
system: basil
type: builtin
name: images
created: 2026-06-28
version: 1.0.0-alpha.1
author: Basil Team
keywords:
  - image
  - imageInfo
  - imageBlur
  - imageSrcset
  - transform
  - resize
  - crop
  - webp
  - jpeg
  - responsive images
  - srcset
  - lqip
  - blur placeholder
---

# Images

Four builtins for transforming, optimising, and serving images from Basil handlers. Images are transformed once, cached to disk, and served at content-hashed URLs with immutable cache headers.

> **Basil-only.** These builtins require the Basil server environment. They will error in `pars` or the REPL.

```parsley
let url = image(@./hero.jpg)
let url = image(@./hero.jpg, {width: 800, format: "webp"})
let resp = imageSrcset(@./hero.jpg, {width: 800}, [400, 800, 1200])
let placeholder = imageBlur(@./hero.jpg)
let info = imageInfo(@./hero.jpg)
```

---

## image()

```parsley
image(path) → string
image(path, options) → string
```

Returns a public URL for the transformed image (e.g. `/__img/a3f2b1c4.jpg`).

**Options:**

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `width` | integer | — | Target width in pixels. Height scales proportionally. |
| `height` | integer | — | Target height in pixels. Width scales proportionally. |
| `crop` | string | `""` | `"center"` fills the exact box and crops excess. `"smart"` analyses the image for faces and high-interest regions, then crops to preserve them. Without crop, the image fits within the box. |
| `scale` | string | `""` | `"smart"` resizes using seam carving — removes or inserts low-energy pixel paths instead of uniformly scaling. Preserves important visual content when changing aspect ratio. Cannot be combined with `crop`. |
| `focal` | dict | — | Focal point or region for smart crop. `{x, y}` (normalised 0–1) or `{x, y, w, h}`. Requires `crop: "smart"`. |
| `quality` | integer | format default | 1–100. Defaults: JPEG 85, WebP 80, PNG lossless. |
| `format` | string | source format | `"jpeg"`, `"png"`, `"webp"`, `"gif"`. Default preserves the source format. |
| `sharpen` | bool or number | `true` | Sharpens after downscale (σ=0.5). `false` disables. A number sets sigma explicitly. |

```parsley
// Serve original (auto-rotated, metadata stripped)
<img src={image(@./photo.jpg)} alt="Photo"/>

// Resize to 400px wide
let thumb = image(@./photo.jpg, {width: 400})

// Crop to exact 800×600
let cover = image(@./photo.jpg, {width: 800, height: 600, crop: "center"})

// Convert to WebP
let webp = image(@./photo.jpg, {width: 800, format: "webp", quality: 75})

// Named style (define once, reuse everywhere)
let heroStyle = {width: 1200, format: "webp", quality: 80}
let hero = image(@./hero.jpg, heroStyle)

// Smart crop — automatically focuses on faces and interesting regions
let portrait = image(@./group-photo.jpg, {width: 400, height: 300, crop: "smart"})

// Smart crop with focal point — direct the crop toward a specific area
let focused = image(@./landscape.jpg, {width: 800, height: 400, crop: "smart", focal: {x: 0.3, y: 0.5}})

// Smart scale — content-aware resizing via seam carving
let narrow = image(@./banner.jpg, {width: 600, scale: "smart"})
```

**Good to know:**

- EXIF orientation is applied automatically.
- Images are never upscaled — requesting a larger width clamps to source dimensions.
- `scale: "smart"` bypasses the upscale clamp — seam carving can intelligently enlarge images by inserting pixel paths.
- Smart crop requires both `width` and `height` (it needs an aspect ratio to evaluate crop candidates).
- `crop` and `scale` are mutually exclusive — use one or the other.
- Changing more than 30% of an image's dimensions via seam carving may produce visible artifacts.
- Sharpening only applies on downscale.
- Files over 50 MB are rejected. Files over 10 MB log a warning.

**Errors:**

| Condition | Error class |
|-----------|-------------|
| Path outside handler directory | `security` |
| File not found or unsupported format | `io` |
| File exceeds 50 MB | `io` |
| Invalid option value | `argument` |
| `crop: "smart"` without both width and height | `argument` |
| `focal` without `crop: "smart"` | `argument` |
| `crop` combined with `scale` | `argument` |
| Called outside Basil server | `state` |

---

## imageInfo()

```parsley
imageInfo(path) → dict
```

Returns metadata about an image without transforming it. Results are cached in memory, so repeated calls (e.g. in a gallery loop) are fast.

**Returns:**

| Key | Type | Description |
|-----|------|-------------|
| `width` | integer | Width in pixels (EXIF-corrected for JPEG) |
| `height` | integer | Height in pixels (EXIF-corrected for JPEG) |
| `format` | string | `"jpeg"`, `"png"`, `"webp"`, or `"gif"` |
| `orientation` | string | `"landscape"`, `"portrait"`, or `"square"` |

```parsley
let info = imageInfo(@./photo.jpg)
// {width: 3024, height: 4032, format: "jpeg", orientation: "portrait"}
```

Use `imageInfo()` to set explicit dimensions on `<img>` tags and avoid layout shift:

```parsley
let info = imageInfo(@./hero.jpg)
<img
    src={image(@./hero.jpg)}
    width={info.width}
    height={info.height}
    alt="Hero"
/>
```

> **Note:** `imageInfo()` returns the **source** dimensions. If you resize with `image()`, use `imageSrcset()` instead — it returns the correct output dimensions in `resp.width` and `resp.height`.

**Gallery example:**

```parsley
let photos = fileList(@./gallery/*.jpg)
for (photo in photos) {
    let info = imageInfo(photo.path)
    let displayWidth = 400
    let displayHeight = (displayWidth * info.height) / info.width
    let url = image(photo.path, {width: displayWidth})
    <a href={image(photo.path, {width: 1200})}>
        <img
            src={url}
            width={displayWidth}
            height={displayHeight}
            alt={photo.stem}
        />
    </a>
}
```

---

## imageBlur()

```parsley
imageBlur(path) → string
```

Returns an inline `data:` URI (~600 bytes) containing a tiny blurred version of the image. Use it as a CSS background that appears instantly while the full image loads. Always outputs JPEG regardless of source format. Results are cached to disk.

```parsley
let blur = imageBlur(@./hero.jpg)
let full = image(@./hero.jpg, {width: 1200})

<div
    style={"background-image: url(" + blur + "); background-size: cover;"}
    class="hero-placeholder"
>
    <img src={full} loading=lazy alt="Hero" class="hero-full"/>
</div>
```

Pair with CSS for a fade-in effect:

```parsley
<style>
.hero-placeholder { position: relative; }
.hero-full { position: absolute; inset: 0; opacity: 0; transition: opacity 0.3s; }
.hero-full.loaded { opacity: 1; }
</style>
```

---

## imageSrcset()

```parsley
imageSrcset(path, style, widths) → dict
imageSrcset(path, style, scales, "x") → dict
```

Generates multiple resized variants and returns a dict for responsive `<img>` tags:

| Key | Type | Description |
|-----|------|-------------|
| `src` | string | URL of the default variant |
| `srcset` | string | Complete `srcset` attribute value |
| `width` | integer | Pixel width of the default variant |
| `height` | integer | Pixel height (computed from aspect ratio) |

**Width descriptor mode:**

```parsley
let resp = imageSrcset(@./hero.jpg, {format: "webp", quality: 80}, [400, 800, 1200])

<img
    src={resp.src}
    srcset={resp.srcset}
    sizes="(max-width: 600px) 100vw, 800px"
    width={resp.width}
    height={resp.height}
    alt="Hero"
/>
```

The `src` is the variant matching `style.width`, or the largest variant if `style.width` is not set.

**Density descriptor mode:**

Pass `"x"` as the fourth argument. `style.width` is required — it is the 1× base width.

```parsley
let resp = imageSrcset(@./logo.png, {width: 120, format: "png"}, [1, 2, 3], "x")

<img
    src={resp.src}
    srcset={resp.srcset}
    width={resp.width}
    height={resp.height}
    alt="Logo"
/>
```

Widths exceeding the source dimensions are clamped — no upscaling. If clamping causes duplicates, they are removed.

`sizes` is always your responsibility — it depends on your CSS layout.

---

## Supported Formats

| Format | Decode | Encode |
|--------|--------|--------|
| JPEG | ✅ | ✅ |
| PNG | ✅ | ✅ |
| GIF | ✅ (first frame) | ✅ |
| WebP | ✅ | ✅ |

WebP encoding uses a pure-Go encoder with no system dependencies. For faster encoding in production, install `libwebp` (`brew install webp` / `apt install libwebp-dev`).

---

## Configuration

```yaml
images:
  cache_dir: ./cache/images   # Default cache location
  max_width: 4000             # Reject transforms wider than this
  max_height: 4000            # Reject transforms taller than this
  default_quality: 85         # Override per-format defaults
  default_format: ""          # Force a default output format (e.g. "webp")
```

Zero-config defaults work for most projects.

---

## Security

All image paths must be within the handler's root directory. Path traversal attempts (e.g. `../../etc/passwd`) are rejected with a `security` error.

---

## See Also

- [Paths](paths.md) — path literals and `@./` syntax
- [@basil/html Img](../stdlib/html.md) — accessible `<img>` component