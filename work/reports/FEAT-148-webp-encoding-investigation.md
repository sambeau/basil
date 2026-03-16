# FEAT-148 Phase 3 — WebP Encoding Investigation Results

**Date:** 2026-03-16
**Feature:** FEAT-148 (Image Transformation and Caching)
**Scope:** Evaluate `gen2brain/webp` for WebP output encoding — binary size impact, encoding performance, compression ratio, and integration effort.

---

## 1. Background

Phase 2 shipped `imageSrcset()` for responsive images. The spec (FEAT-148 §Phase 3) and the Phase 2 investigation both identified WebP output encoding as the next logical step: the codebase is fully prepared (options parsing, validation, normalization, quality defaults, caching all handle `"webp"`), and `imageSrcset()` is the primary consumer (serving smaller WebP variants to supporting browsers).

The Phase 2 investigation estimated +1–3 MB binary impact but deferred actual measurement. This report provides measured data.

## 2. Library: `gen2brain/webp` v0.5.5

### Overview

| Attribute | Value |
|-----------|-------|
| Repository | `github.com/gen2brain/webp` |
| Version | v0.5.5 (May 2025) |
| License | MIT |
| Stars | 54 |
| CGo required | **No** — pure-Go via two backends |
| Go version | 1.23+ |
| Transitive deps | `ebitengine/purego` v0.8.3, `tetratelabs/wazero` v1.9.0 |

### How It Works (Two Backends)

1. **Dynamic library** (preferred): If `libwebp` is installed on the host, loads it at runtime via `purego` (pure-Go dlopen/dlsym). No CGo. Fastest path.
2. **WASM fallback**: If no system `libwebp` is found, uses Google's `libwebp` compiled to WASM, executed by the `wazero` pure-Go WebAssembly runtime. Always works, slower.

The backend is selected automatically at runtime. `webp.Dynamic()` returns `nil` if the dynamic library loaded, or an error describing why it fell back to WASM.

### Encoding API

```go
// Options are the encoding parameters.
type Options struct {
    Quality  int  // Range [0,100]. 100 implies Lossless. Default 75.
    Lossless bool // Ignore quality, use lossless compression.
    Method   int  // Speed/quality tradeoff: 0=fast, 6=slower-better. Default 4.
    Exact    bool // Preserve exact RGB in transparent areas.
}

// Encode writes the image m to w with the given options.
func Encode(w io.Writer, m image.Image, o ...Options) error
```

Clean variadic API. Maps directly to our `Encode()` pattern.

### Gotchas

- **Quality=0 means "use default (75)"**, not "worst quality". Use `Quality: 1` for minimum.
- **First encode/decode pays WASM init cost** (~146ms). Call `webp.Init()` at startup to front-load.
- **No animated WebP encoding** (decode only). Not relevant for our use case.
- **Converts all images to `*image.NRGBA`** internally before encoding. Small allocation cost per encode.

---

## 3. Binary Size Impact (Measured)

### Baseline (before)

| Binary | Size |
|--------|------|
| `basil` | 37,141,154 bytes (35.4 MiB) |
| `pars` | 28,094,194 bytes (26.8 MiB) |

### After adding `gen2brain/webp` v0.5.5

| Binary | Size | Delta |
|--------|------|-------|
| `basil` | 41,348,562 bytes (39.4 MiB) | **+4,207,408 bytes (+4.0 MiB, +11.3%)** |
| `pars` | 28,094,194 bytes (26.8 MiB) | unchanged (does not import images package) |

### Where the +4.0 MiB comes from

| Component | Contribution |
|-----------|-------------|
| `tetratelabs/wazero` (WebAssembly runtime) | ~3.5 MiB (86K lines of Go — a full WASM interpreter/compiler) |
| `libwebp.wasm.gz` (embedded WASM binary) | ~158 KB compressed |
| `ebitengine/purego` (dynamic loading) | ~200 KB |
| `gen2brain/webp` itself | ~100 KB |

The dominant cost is `wazero`. This is a one-time cost — if any future dependency also uses `wazero`, it won't double-count.

### Assessment

+4.0 MiB (+11.3%) is significant but acceptable for a web server binary. The `basil` binary was already 35 MiB; going to 39 MiB doesn't cross a meaningful threshold. The `pars` CLI is unaffected.

---

## 4. Encoding Performance (Measured)

All measurements on macOS (Apple Silicon), using **WASM fallback** (no system `libwebp` installed). The dynamic library path would be ~3–5× faster.

### WASM Init Cost

| Operation | Time |
|-----------|------|
| `webp.Init()` (cold, first call) | **146ms** |
| Subsequent encodes | No init overhead |

This should be called at server startup (e.g., in `NewRegistry()`).

### Encoding Speed: WebP vs JPEG

#### Gradient images (realistic photo-like content)

| Image size | WebP encode | JPEG encode | Slowdown | WebP size | JPEG size | Compression gain |
|-----------|-------------|-------------|----------|-----------|-----------|-----------------|
| 400×300 | **4.7ms** | 1.5ms | 3.1× | 2 KB | 6 KB | **67% smaller** |
| 800×600 | **16.5ms** | 5.8ms | 2.9× | 5 KB | 18 KB | **69% smaller** |
| 1200×900 | **36.8ms** | 12.8ms | 2.9× | 10 KB | 35 KB | **72% smaller** |

#### Random noise images (worst case for compression)

| Image size | WebP encode | JPEG encode | Slowdown | WebP size | JPEG size | Compression gain |
|-----------|-------------|-------------|----------|-----------|-----------|-----------------|
| 800×600 | 57ms | 15ms | 3.8× | 315 KB | 315 KB | ~0% |
| 1200×900 | 117ms | 31ms | 3.8× | 713 KB | 710 KB | ~0% |
| 1920×1080 | 223ms | 59ms | 3.8× | 1360 KB | 1357 KB | ~0% |

### Key Findings

1. **WebP is 3–4× slower than JPEG to encode** on the WASM path. With system `libwebp`, the gap narrows to ~1.2× (per library benchmarks).
2. **WebP produces 67–72% smaller files** for typical photo-like content. For random noise (worst case), there's no size advantage.
3. **Encoding time is acceptable**: 4.7ms for a 400×300 variant, 37ms for 1200×900. Since results are disk-cached and deduplicated via singleflight, each variant is only encoded once.
4. **Init cost is one-time**: 146ms at startup is negligible for a web server.

### Performance with system `libwebp`

If the user installs `libwebp` (e.g., `brew install webp` / `apt install libwebp-dev`), the dynamic backend kicks in automatically. Based on the library's published benchmarks:

| Backend | Relative encode speed |
|---------|----------------------|
| Dynamic (system libwebp) | **1×** (fastest) |
| WASM fallback | **5×** slower |

With dynamic loading, WebP encoding would be ~1.2× slower than stdlib JPEG — essentially negligible.

---

## 5. Integration Effort

### What's already done (from Phase 1 & 2)

The entire format pipeline already handles `"webp"`:

| Component | WebP handling | Status |
|-----------|--------------|--------|
| `ParseOptions()` | Accepts `format: "webp"` | ✅ |
| `Validate()` | Validates `"webp"` as allowed format | ✅ |
| `NormalizeFormat()` | Maps `"webp"` → `("webp", ".webp")` | ✅ |
| `ExtensionToFormat()` | Maps `".webp"` → `"webp"` | ✅ |
| `DefaultQuality()` | Returns 80 for `"webp"` | ✅ |
| Cache paths | Handles `.webp` extension | ✅ |
| Evaluator builtins | Error hints list WebP as supported | ✅ |
| WebP decode | Registered via `x/image/webp` import | ✅ |

### What needs to change

1. **`server/images/transform.go`** — `Encode()` function, `case "webp":` — replace error stub with `webp.Encode()` call (~5 lines)
2. **`server/images/transform.go`** — `Process()` function — remove WebP→JPEG fallback (~3 lines deleted)
3. **`server/images/registry.go`** — call `webp.Init()` in `NewRegistry()` to front-load WASM init (~1 line)
4. **`server/images/transform_test.go`** — update `TestEncode_WebPOutputError` → success test, update `TestProcess_WebPInputFallsBackToJPEG` → WebP-in/WebP-out test
5. **`go.mod`** — `go get github.com/gen2brain/webp@v0.5.5`

No changes needed in options parsing, validation, normalization, cache, evaluator, or handler code.

### Estimated effort

~30 minutes of implementation + testing. The smallest Phase 3 item by far.

---

## 6. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Binary size increase (+4 MiB) | Certain | Low | Acceptable for a web server. `pars` unaffected. |
| WASM encode latency (3–4× JPEG) | Certain | Low | Results cached to disk + singleflight dedup. Each variant encoded once. Document `brew install webp` for production speed. |
| WASM init latency (146ms) | Certain | Low | Call `webp.Init()` at server startup. |
| `wazero` dependency size/maintenance | Low | Low | Well-maintained (Tetrate Labs), v1.9.0, widely used in Go ecosystem. |
| Library abandonment | Low | Low | v0.5.5 released May 2025, actively maintained. WASM binary is vendored. |
| Quality=0 silent default | Low | Low | Our `DefaultQuality("webp")` returns 80, and `Encode()` passes quality explicitly. |

---

## 7. Recommendation

**Proceed with implementation.** The case is clear:

- **High user value**: WebP produces 67–72% smaller files for responsive images. Combined with `imageSrcset()`, this is the single highest-impact optimization available.
- **Minimal integration effort**: ~5 lines of code change + test updates. The codebase was designed for this.
- **Acceptable tradeoffs**: +4.0 MiB binary (11.3%), 3–4× slower encode (mitigated by caching), 146ms one-time init (mitigated by startup call).
- **No CGo**: Pure-Go, cross-compiles cleanly, no build toolchain changes.
- **Graceful performance scaling**: WASM fallback always works; system `libwebp` provides near-native speed when available.

### Documentation notes for users

After implementation, document in the FAQ or guide:
- WebP output is now supported: `image(@./photo.jpg, {format: "webp"})`
- For best encoding performance, install system `libwebp` (`brew install webp` / `apt install libwebp-dev`)
- Without system library, encoding uses a pure-Go WASM fallback (works everywhere, ~3× slower, results cached)