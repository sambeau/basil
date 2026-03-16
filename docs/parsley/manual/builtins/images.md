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

Basil provides four built-in functions for transforming, optimising, and serving images directly from Parsley handlers. Images are transformed on first use, cached to disk at content-hashed URLs, and served with immutable cache headers — no external image service or manual resizing required.

> **Basil-only.** These builtins are only available inside Basil server handlers. They will error if called from `pars` or the REPL.

```parsley
// Basic usage — auto-rotate, strip metadata, cache, return URL
let url = image(@./hero.jpg)

// Resize to 800px wide, convert to WebP
let url = image(@./hero.jpg, {width: 800, format: "webp"})

// Responsive image set
let resp = imageSrcset(@./hero.jpg, {width: 800}, [400, 800, 1200])

// Blur placeholder for progressive loading
let placeholder = imageBlur(@./hero.jpg)

// Image metadata
let info = imageInfo(@./hero.jpg)
```

---

## image()

```parsley
image(path) → string
image(path, options) → string
```

Transforms an image and returns its public URL (e.g. `/__img/a3f2b1c4.jpg`). The first call performs the transform and caches the result to disk; subsequent calls return the cached URL immediately.

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| `path` | path | Source image path. Use `@./` (relative to current file) or `@~/` (relative to project root). |
| `options` | dict | Transform options (see below). Optional. |

**Options dict:**

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `width` | integer | — | Target width in pixels. Height is scaled proportionally unless `height` is also set. |
| `height` | integer | — | Target height in pixels. Width is scaled proportionally unless `width` is also set. |
| `crop` | string | `""` | `"center"` fills the exact `width`×`height` box, cropping excess. Without `crop`, the image fits within the box (letterbox). |
| `quality` | integer | format default | Compression quality 1–100. Defaults: JPEG 85, WebP 80, PNG lossless. |
| `format` | string | source format | Output format: `"jpeg"`, `"png"`, `"webp"`, `"gif"`. Default preserves the source format (JPEG in → JPEG out). |
| `sharpen` | bool or number | `true` | `true` applies default sharpening (σ=0.5) after downscale. `false` disables. A number sets the sigma explicitly. |

**Returns:** A `string` URL beginning with `/__img/`.

**Behaviour:**

- EXIF orientation is applied automatically — a portrait photo shot with the camera rotated will display correctly.
- Images are never upscaled. Requesting a width larger than the source clamps to source dimensions.
- Sharpening is only applied on downscale, never on upscale or same-size encodes.
- Source files over 10 MB produce a server log warning. Files over 50 MB are rejected with an error.
- Concurrent requests for the same transform are deduplicated (singleflight) — the transform runs once regardless of how many requests arrive simultaneously.

**Examples:**

```parsley
// Serve original (auto-rotated, metadata stripped)
let url = image(@./photo.jpg)
<img src={url} alt="Photo"/>

// Resize to 400px wide, preserve aspect ratio
let thumb = image(@./photo.jpg, {width: 400})

// Crop to exact 800×600 box (center crop)
let cover = image(@./photo.jpg, {width: 800, height: 600, crop: "center"})

// Convert to WebP at quality 75
let webpUrl = image(@./photo.jpg, {width: 800, format: "webp", quality: 75})

// Disable auto-sharpening
let soft = image(@./photo.jpg, {width: 400, sharpen: false})

// Named style dict (define once, reuse)
let heroStyle = {width: 1200, format: "webp", quality: 80}
let hero = image(@./hero.jpg, heroStyle)
```

**Errors:**

| Condition | Error class |
|-----------|-------------|
| Path outside handler directory | `security` |
| File not found | `io` |
| Unsupported format | `io` |
| File exceeds 50 MB | `io` |
| Invalid option value | `argument` |
| Called outside Basil server | `state` |

---

## imageInfo()

```parsley
imageInfo(path) → dict
```

Returns metadata about an image without transforming it. Results are cached in memory (keyed by path and file modification time), so calling `imageInfo()` repeatedly in a loop — for example, rendering a gallery — is efficient.

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| `path` | path | Source image path. |

**Returns:** A dict:

| Key | Type | Description |
|-----|------|-------------|
| `width` | integer | Width in pixels (after EXIF auto-orientation for JPEG) |
| `height` | integer | Height in pixels (after EXIF auto-orientation for JPEG) |
| `format` | string | Detected format: `"jpeg"`, `"png"`, `"webp"`, `"gif"` |
| `orientation` | string | `"landscape"`, `"portrait"`, or `"square"` |

**Example:**

```parsley
let info = imageInfo(@./photo.jpg)
// {width: 3024, height: 4032, format: "jpeg", orientation: "portrait"}

// Useful for setting explicit width/height on img tags (avoids CLS)
let info = imageInfo(@./hero.jpg)
<img
    src={image(@./hero.jpg, {width: 1200})}
    width={info.width}
    height={info.height}
    alt="Hero"
/>
```

**Gallery example:**

```parsley
let photos = fileList(@./gallery/*.jpg)
for (photo in photos) {
    let info = imageInfo(photo.path)   // cached after first call per photo
    let url  = image(photo.path, {width: 400})
    <a href={image(photo.path, {width: 1200})}>
        <img
            src={url}
            width={400}
            height={info.orientation == "landscape" ? 267 : 533}
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

Generates a Low Quality Image Placeholder (LQIP) and returns it as an inline `data:` URI. Use it as a CSS background image that appears instantly while the full-resolution image loads.

The pipeline is: resize to 20 px wide → Gaussian blur (σ=10) → JPEG quality 20 → base64. The result is approximately 600 bytes (~835 characters as a data URI). The result is cached to disk; subsequent calls return the cached string immediately.

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| `path` | path | Source image path. |

**Returns:** A `string` in the form `"data:image/jpeg;base64,..."`. Always JPEG output regardless of source format.

**Example:**

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

**CSS pattern:**

```parsley
// Fade the real image in over the placeholder
<style>
".hero-placeholder { position: relative; }"
".hero-full { position: absolute; inset: 0; opacity: 0; transition: opacity 0.3s; }"
".hero-full.loaded { opacity: 1; }"
</style>
```

---

## imageSrcset()

```parsley
imageSrcset(path, style, widths) → dict
imageSrcset(path, style, scales, "x") → dict
```

Generates multiple resized variants of an image and returns a dict ready for use in a responsive `<img>` tag. All variants are produced via the same disk-cached, singleflight-deduped pipeline as `image()`.

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| `path` | path | Source image path. |
| `style` | dict | Base transform options (same keys as `image()` options). |
| `widths` | array of integers | Target pixel widths for width descriptor mode, e.g. `[400, 800, 1200]`. |
| `scales` | array of integers | Density multipliers for density descriptor mode, e.g. `[1, 2, 3]`. Requires `style.width` to be set. |
| `"x"` | string literal | Pass `"x"` as the fourth argument to enable density descriptor mode. |

**Returns:** A dict:

| Key | Type | Description |
|-----|------|-------------|
| `src` | string | URL of the default/base variant |
| `srcset` | string | Complete `srcset` attribute value |
| `width` | integer | Pixel width of the default variant |
| `height` | integer | Pixel height of the default variant (computed from aspect ratio) |

**Width descriptor mode:**

Generates one variant per width. The `src` is the variant matching `style.width`, or the largest variant if `style.width` is not set.

```parsley
let heroStyle = {width: 800, format: "webp", quality: 80}
let resp = imageSrcset(@./hero.jpg, heroStyle, [400, 800, 1200])

<img
    src={resp.src}
    srcset={resp.srcset}
    sizes="(max-width: 600px) 100vw, 800px"
    width={resp.width}
    height={resp.height}
    alt="Hero"
/>
```

`resp.srcset` will look like:
```
/__img/a1b2c3d4.webp 400w, /__img/e5f6a7b8.webp 800w, /__img/c9d0e1f2.webp 1200w
```

**Density descriptor mode:**

Pass `"x"` as the fourth argument and a `scales` array. `style.width` must be set — it is the 1x base width.

```parsley
let logoStyle = {width: 120, format: "png"}
let resp = imageSrcset(@./logo.png, logoStyle, [1, 2, 3], "x")

<img
    src={resp.src}
    srcset={resp.srcset}
    width={resp.width}
    height={resp.height}
    alt="Logo"
/>
```

`resp.srcset` will look like:
```
/__img/a1b2c3d4.png 1x, /__img/e5f6a7b8.png 2x, /__img/c9d0e1f2.png 3x
```

**Clamping:** Widths that exceed the source image dimensions are clamped to the source width — no upscaling occurs. If clamping causes multiple requested widths to collapse to the same pixel width, duplicates are removed.

**`sizes` is always user-provided.** The correct value depends on your CSS layout and cannot be computed automatically.

---

## Supported Formats

| Format | Input (decode) | Output (encode) |
|--------|---------------|-----------------|
| JPEG | ✅ | ✅ |
| PNG | ✅ | ✅ |
| GIF | ✅ (first frame) | ✅ |
| WebP | ✅ | ✅ |

> **WebP encoding note:** Basil uses a pure-Go WebP encoder (`gen2brain/webp`) with a WASM fallback so no system libraries are needed. The first encode in a server process incurs a ~150 ms one-time initialisation cost. Install `libwebp` on your production host for near-native encoding speed (`brew install webp` / `apt install libwebp-dev`).

---

## Configuration

Image behaviour can be tuned in `basil.yaml`:

```yaml
images:
  cache_dir: ./cache/images   # Default cache location
  max_width: 4000             # Reject requests exceeding this width
  max_height: 4000            # Reject requests exceeding this height
  default_quality: 85         # Override per-format defaults
  default_format: ""          # Force a default output format
```

Zero-config defaults are sensible for most projects.

---

## Security

All image paths are restricted to the handler's root directory. A path like `@~/../../etc/passwd` will be rejected with a `security` error. Only use path literals (`@./`) pointing to files you control.

---

## See Also

- [Paths](paths.md) — path literals and `@./` syntax
- [@basil/html `Img`](../stdlib/html.md) — accessible `<img>` component that pairs well with these builtins
- [FEAT-148 spec](../../../../work/specs/FEAT-148.md) — full specification including design decisions