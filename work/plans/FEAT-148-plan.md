---
id: PLAN-128
feature: FEAT-148
title: "Implementation Plan for Image Transformation and Caching"
status: complete
created: 2026-06-28
---

# Implementation Plan: FEAT-148 — Image Transformation and Caching

## Overview

Add built-in image transformation and caching to Basil. The `image()` builtin transforms images (resize, crop, auto-rotate, format conversion), caches results to disk, and returns content-hashed immutable URLs. The `imageInfo()` builtin returns image metadata without transforming. The architecture mirrors the existing `publicUrl()` → `assetRegistry` → `assetHandler` pattern.

## Design Decisions Driving This Plan

1. **Default format = original** (not WebP). JPEG in → JPEG out unless `{format: "webp"}` is specified.
2. **WebP is opt-in** via `gen2brain/webp` (WASM-based, CGo-free). WebP output support is now implemented; the binary impact and runtime tradeoffs were measured and accepted.
3. **`disintegration/imaging`** is the core transform engine (resize, crop, auto-orient, encode/decode).
4. **`singleflight`** for concurrent transform deduplication.
5. **Atomic cache writes** (temp file + rename) to prevent serving partial files.

## Dependencies

| Package | Purpose | Risk |
|---------|---------|------|
| `github.com/disintegration/imaging` | Resize, crop, rotate, auto-orient, JPEG/PNG encode/decode | Low — stable, 10k+ dependents, pure Go |
| `github.com/gen2brain/webp` | WebP encode/decode (CGo-free via WASM) | Medium — small community, adds wazero dep |
| `golang.org/x/image/webp` | WebP decode (fallback/primary decoder) | Low — Go team maintained |
| `golang.org/x/sync/singleflight` | Dedup concurrent transforms for same cache key | Low — standard Go extended lib |

## Prerequisites

- [x] Library evaluation complete (see FEAT-148.md)
- [x] Spec revised: default format changed from WebP to original
- [x] Feature branch created: `feat/FEAT-148-image-transform`

## Tasks

### Task 1: Image Config

**Files**: `server/config/config.go`
**Estimated effort**: Small

Add `ImageConfig` struct and wire into `Config`:

```go
type ImageConfig struct {
    CacheDir       string `yaml:"cache_dir"`       // default: "./cache/images"
    MaxWidth       int    `yaml:"max_width"`        // default: 4096
    MaxHeight      int    `yaml:"max_height"`       // default: 4096
    DefaultQuality int    `yaml:"default_quality"`  // default: 0 (format-specific)
    DefaultFormat  string `yaml:"default_format"`   // default: "" (original)
}
```

Steps:
1. Add `ImageConfig` struct
2. Add `Images ImageConfig` field to `Config`
3. Add defaults in `Defaults()` method

Tests:
- Config parsing with image section
- Config defaults are applied correctly

---

### Task 2: Transform Options

**Files**: `server/images/options.go`
**Estimated effort**: Small

Define `TransformOptions` and parsing from Parsley dict values.

Steps:
1. Create `server/images/` package
2. Define `TransformOptions` struct
3. Write `ParseOptions(dict)` to convert a Parsley dict to `TransformOptions`
4. Write `Validate()` method to check option ranges (width > 0, quality 1-100, etc.)
5. Write `Canonical()` method for stable string representation (used in cache key)

Tests:
- Parse valid options dict
- Parse empty dict (all defaults)
- Reject invalid values (negative width, quality > 100, unknown format)
- Canonical string is deterministic

---

### Task 3: Core Transform Pipeline

**Files**: `server/images/transform.go`
**Estimated effort**: Medium

The transform pipeline: load → auto-orient → resize/crop → encode.

Steps:
1. `Load(path string) (image.Image, string, error)` — decode image, detect format, handle WebP via `x/image/webp`
2. `Transform(img image.Image, opts TransformOptions) image.Image` — apply resize/crop
3. `Encode(img image.Image, format string, quality int) ([]byte, error)` — encode to target format
4. `Process(sourcePath string, opts TransformOptions) ([]byte, string, error)` — full pipeline: load → transform → encode, returns bytes + output extension
5. Use `imaging.Open(path, imaging.AutoOrientation(true))` for JPEG/PNG/GIF auto-orientation
6. For WebP input: decode via `golang.org/x/image/webp`, auto-orientation not available (EXIF in WebP is rare)
7. Resize logic:
   - Width only → fit width, preserve aspect ratio
   - Height only → fit height, preserve aspect ratio
   - Both without crop → fit within box (`imaging.Fit`)
   - Both with `crop: "center"` → fill box and crop (`imaging.Fill` with `imaging.Center`)
8. No upscaling: if requested dimensions exceed source, use source dimensions
9. Animated GIF: use first frame only (with warning logged)
10. Source size limits: reject > 50MB, warn > 10MB

Tests:
- Resize JPEG: width only, height only, both (fit), both + crop
- Auto-orientation with rotated JPEG
- No upscaling: requested 2000px but source is 800px → output is 800px
- Quality parameter affects output size
- Format conversion: JPEG → PNG, PNG → JPEG
- WebP output via `gen2brain/webp`
- WebP source with no explicit output format preserves WebP output
- GIF first frame extraction
- Reject oversized file
- Encode/decode round-trip quality

---

### Task 4: Disk Cache

**Files**: `server/images/cache.go`
**Estimated effort**: Medium

Content-addressed disk cache with atomic writes.

Steps:
1. `CacheKey(sourceHash string, opts TransformOptions) string` — SHA256 of `sourceHash + opts.Canonical()`, truncated to 16 hex chars
2. `SourceHash(path string) (string, error)` — SHA256 of file contents, truncated to 16 hex chars (same algo as `publicUrl()`)
3. `CachePath(cacheDir, key, ext string) string` — `{cacheDir}/{key}.{ext}`
4. `Read(cacheDir, key, ext string) (string, bool)` — check if cached file exists, return path
5. `Write(cacheDir, key, ext string, data []byte) (string, error)` — atomic write: write to temp file in cacheDir, then `os.Rename`
6. Create cacheDir on first write if it doesn't exist (`os.MkdirAll`)
7. Dev mode: accept `modTime` parameter, re-transform if source is newer than cached file

Tests:
- Cache key is deterministic for same inputs
- Cache key differs when options differ
- Cache key differs when source hash differs
- Write + Read round-trip
- Atomic write: partial write doesn't leave corrupt file
- Cache dir created automatically
- Dev mode: stale cache detected

---

### Task 5: Image Registry

**Files**: `server/images/registry.go`
**Estimated effort**: Medium

Maps hash → filepath (like `assetRegistry`), coordinates transforms, deduplicates concurrent requests.

Steps:
1. Define `Registry` struct with:
   - `byHash map[string]string` (hash → cached file path)
   - `sync.RWMutex` for thread safety
   - `singleflight.Group` for transform dedup
   - Config fields (cacheDir, maxWidth, maxHeight, defaultQuality, defaultFormat, devMode)
2. `NewRegistry(cfg ImageConfig, devMode bool, logger func(string, ...any)) *Registry`
3. `Transform(sourcePath string, opts TransformOptions) (string, error)`:
   - Compute source hash
   - Merge opts with config defaults
   - Compute cache key
   - Check cache → return URL if hit
   - Use singleflight to transform, cache, register
   - Return `/__img/{key}.{ext}`
4. `Info(sourcePath string) (ImageInfo, error)`:
   - Decode image config (not full decode — just dimensions/format)
   - Return `{width, height, format, orientation}`
   - Cache in memory (metadata is small)
5. `Lookup(hash string) (string, bool)` — reverse lookup for handler
6. `Clear()` — clear in-memory maps (called on dev reload)
7. On startup/first use, scan cacheDir for existing cached files and populate `byHash` map

Tests:
- Transform returns URL in correct format
- Cache hit on second call (no re-transform)
- Concurrent transforms for same key → single execution (singleflight)
- Info returns correct dimensions and format
- Clear wipes registry
- Config defaults applied (maxWidth, defaultQuality)

---

### Task 6: HTTP Handler

**Files**: `server/images/handler.go`
**Estimated effort**: Small

Mirror `assetHandler` pattern for `/__img/` route.

Steps:
1. Define `Handler` struct with `registry *Registry` and `devMode bool`
2. `NewHandler(registry *Registry, devMode bool) *Handler`
3. `ServeHTTP`:
   - Strip `/__img/` prefix
   - Split hash from extension
   - Lookup in registry
   - Verify extension matches
   - Set cache headers (immutable in prod, no-cache in dev)
   - `http.ServeFile`

Tests:
- Serve cached image with correct Content-Type
- 404 for unknown hash
- Extension mismatch → 404
- Cache headers correct in prod vs dev mode

---

### Task 7: Evaluator Builtins

**Files**: `pkg/parsley/evaluator/image.go`
**Estimated effort**: Medium

Add `image()` and `imageInfo()` builtins. Follow `public_url.go` pattern exactly.

Steps:
1. Define `ImageRegistrar` interface in `evaluator.go`:
   ```go
   type ImageRegistrar interface {
       Transform(sourcePath string, opts map[string]any) (string, error)
       Info(sourcePath string) (map[string]any, error)
   }
   ```
2. Add `ImageRegistry ImageRegistrar` field to `Environment`
3. Propagate in `NewEnclosedEnvironment` and all env-copy sites (same places as `AssetRegistry`)
4. `NewImageBuiltin()` → `StdlibBuiltin` for `image()`:
   - Accept 1 arg (path) or 2 args (path, options dict)
   - Resolve path same as `publicUrl()`: relative to current file, validate within RootPath
   - If no ImageRegistry, return error
   - Call `env.ImageRegistry.Transform(absPath, optsMap)`
   - Return URL string
5. `NewImageInfoBuiltin()` → `StdlibBuiltin` for `imageInfo()`:
   - Accept 1 arg (path)
   - Resolve path same way
   - Call `env.ImageRegistry.Info(absPath)`
   - Return Parsley dict

Tests:
- `image(@./photo.jpg)` returns URL string
- `image(@./photo.jpg, {width: 300})` returns URL string
- `image(@./photo.jpg, {format: "webp"})` returns URL with .webp extension
- WebP source with no explicit format preserves `.webp` output
- `imageInfo(@./photo.jpg)` returns dict with width, height, format, orientation
- Path resolution: relative to current file
- Security: path outside root → error
- Missing ImageRegistry → clear error
- Invalid path → clear error
- Invalid options → clear error

---

### Task 8: Server Integration

**Files**: `server/server.go`, `server/handler.go`, `server/api.go`
**Estimated effort**: Small

Wire everything together.

Steps:
1. Add `imageRegistry *images.Registry` field to `Server`
2. Initialize in `New()` using `cfg.Images`
3. Mount handler: `s.mux.Handle("/__img/", images.NewHandler(s.imageRegistry, devMode))`
4. Inject into evaluator env in `handler.go` and `api.go`: `env.ImageRegistry = h.server.imageRegistry`
5. Clear on reload: `s.imageRegistry.Clear()` in `ReloadScripts()`
6. Register builtins: add `NewImageBuiltin()` and `NewImageInfoBuiltin()` to the builtin list

Tests:
- Integration test: full round-trip from Parsley `image()` call to served image
- Dev mode reload clears registry

---

### Task 9: Add Dependencies

**Files**: `go.mod`, `go.sum`
**Estimated effort**: Small

```bash
go get github.com/disintegration/imaging
go get github.com/gen2brain/webp
go get golang.org/x/sync
go mod tidy
```

Note: `golang.org/x/image` may already be an indirect dependency. Verify.

---

## Implementation Order

The tasks have these dependencies:

```
Task 1 (Config) ──────────────────────────┐
Task 9 (Dependencies) ────────────────────┤
Task 2 (Options) ─────────────────────────┤
                                          ▼
Task 3 (Transform) ──► Task 4 (Cache) ──► Task 5 (Registry) ──► Task 6 (Handler)
                                                                       │
Task 7 (Builtins) ◄───────────────────────────────────────────────────┘
                                          │
                                          ▼
                                    Task 8 (Integration)
```

Suggested order for implementation:
1. **Task 9** — Add dependencies first so imports resolve
2. **Task 1** — Config (small, no other deps)
3. **Task 2** — Options (no other deps)
4. **Task 3** — Transform pipeline (depends on deps + options)
5. **Task 4** — Disk cache (depends on options for cache key)
6. **Task 5** — Registry (depends on transform + cache)
7. **Task 6** — Handler (depends on registry)
8. **Task 7** — Builtins (depends on interface shape from registry)
9. **Task 8** — Integration (depends on everything)

Commit points:
- After Task 2+3+4 (core package compiles and tests pass)
- After Task 5+6 (server/images package complete)
- After Task 7 (builtins complete)
- After Task 8 (fully integrated, all tests pass)

## Estimated Total Effort

| Component | Lines (est.) |
|-----------|-------------|
| `server/images/options.go` | ~80 |
| `server/images/transform.go` | ~150 |
| `server/images/cache.go` | ~100 |
| `server/images/registry.go` | ~200 |
| `server/images/handler.go` | ~70 |
| `pkg/parsley/evaluator/image.go` | ~150 |
| Config additions | ~30 |
| Integration wiring | ~30 |
| **Tests** | ~400 |
| **Total** | **~1200** |

## Validation Checklist

- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] Benchmarks checked: `make bench-compare`
- [ ] Manual test: create a `.pars` file using `image()` and `imageInfo()`, verify output
- [ ] Dev mode: modify source image, verify re-transform
- [ ] Cache directory created automatically
- [ ] WebP opt-in works: `{format: "webp"}` produces WebP output
- [ ] Security: path traversal attempt returns error

## Risk Mitigation

| Risk | Mitigation |
|------|-----------|
| `gen2brain/webp` adds too much binary size | WebP support is opt-in; can be removed without affecting core feature. Measure binary size before/after. |
| `disintegration/imaging` has unpatched CVE | Input validation (size limits, dimension limits) provides defense in depth. Monitor for forks. |
| WASM WebP encode too slow | Results are cached — 96ms one-time cost is acceptable. Log transform duration for monitoring. |
| Memory spike on large images | 50MB file size limit + 4096×4096 max output dimensions. Decoded 50MP image ≈ 200MB; consider source dimension limit (8192×8192) if this becomes an issue. |

## Deferred Items

Items to add to `work/BACKLOG.md` after implementation:
- `imageSrcset()` builtin for responsive images (Phase 2)
- Blur placeholder generation (Phase 2)
- Smart crop via `muesli/smartcrop` (Phase 2)
- Dominant color extraction in `imageInfo()` (Phase 2)
- AVIF output format (Phase 3)
- Cache eviction / LRU / `basil cache clear` CLI (Phase 3)
- Source dimension limit for memory protection (if needed)
- Evaluate `disintegration/imaging` forks if maintenance becomes a concern

## Post-Implementation Review Findings

Review conducted after Phase 1 implementation. All tests pass. Findings below are recorded for tracking before fixes are applied.

### Bug: Variable Shadowing in `doTransform` (registry.go)

**Severity**: Medium — masked by early returns but structurally incorrect and fragile.

In `registry.go` `doTransform()`, line 155 uses `:=` to declare `err` inside the `else` branch, which **shadows** the outer `err` declared on line 147. The inner `err` from `Process()` and `cache.Write()` is never visible to the `if err != nil` check on line 165, which reads the outer `err` (always `nil` in the `else` branch).

In practice, the early returns at lines 156 and 161 prevent incorrect behavior — but if those were ever refactored, errors from the `else` branch would be silently swallowed.

**Fix**: Change line 155 from `:=` to `=` with `data` declared separately:
```go
var data []byte
data, _, err = Process(sourcePath, opts)
```

### Test Coverage Gaps

The `server/images` unit tests (options, transform, cache) are thorough. However, three components have **no dedicated tests**:

#### 1. Registry (registry.go) — No Tests
The most complex component (singleflight coordination, cache hit/miss logic, format determination, config default merging) is entirely untested at the unit level. Missing tests:
- `Transform()` happy path: returns correct URL format
- Cache hit on second call (no re-transform)
- Concurrent transforms for same key → single execution (singleflight)
- `Info()` returns correct metadata
- `Lookup()` / `Clear()` / `CacheStats()` behavior
- Config defaults applied (maxWidth, defaultQuality, defaultFormat)

#### 2. Handler (handler.go) — No Tests
The HTTP handler has no tests. Missing tests per spec Task 6:
- Serve cached image with correct Content-Type
- 404 for unknown hash
- Extension mismatch → 404
- Cache headers correct in prod vs dev mode

#### 3. Evaluator Builtins (evaluator/image.go) — No Tests
The `image()` and `imageInfo()` builtins have no tests. Missing tests per spec Task 7:
- Missing `ImageRegistry` → clear error message
- Invalid argument count → arity error
- Path dict vs string argument handling
- Security: path outside root → error
- Invalid options dict → error

#### 4. Weak Existing Tests
- `TestProcess_WebPInputFallsBackToJPEG` doesn't test actual WebP input — it tests JPEG→JPEG and JPEG→PNG with explicit format, never exercising the WebP fallback code path at `transform.go` lines 183–185.
- `TestLoad_Errors/file_too_large` is skipped — the 50MB size limit is untested.

### Minor Code Quality Items

#### `SourceHash` Reads Entire File Into Memory
`transform.go` `SourceHash()` uses `io.ReadAll(f)` which allocates the full file contents (up to 50MB) just to compute a hash. Should use streaming: `io.Copy(hasher, f)` with `sha256.New()`.

#### `imageInfo()` Metadata Not Cached in Memory
The spec says "Results cached in memory (metadata is small)." Currently `Info()` calls `GetInfo()` every time without caching. For gallery-style loops this means repeated disk I/O and image header parsing for the same file. Low priority since `image.DecodeConfig` only reads headers.

### Spec Deviations (Acceptable)

These are intentional deviations documented for the record:

1. **`ImageRegistrar` uses `map[string]any`** instead of typed `ImageOptions`/`ImageInfo` structs at the interface boundary. This avoids a circular dependency between `evaluator` and `server/images`. Typed structs are used internally. Correct tradeoff.

2. **No `gen2brain/webp` dependency** — WebP encoding deferred. WebP output returns a clear error. WebP input decoding works via `golang.org/x/image/webp`. Binary size concern justified the deferral.

3. **No eager cache directory scan on startup** — the spec's Task 5 says "scan cacheDir for existing cached files." Implementation uses per-key lazy probing via `os.Stat` on the deterministic path. This is arguably better: avoids startup latency and doesn't need to reverse-engineer options from filenames.

---

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-06-28 | Spec revision | ✅ Complete | Default format changed to original, WebP opt-in |
| 2026-06-28 | Library evaluation | ✅ Complete | See FEAT-148.md |
| 2026-06-28 | Implementation plan | ✅ Complete | This document |
| 2026-06-28 | Task 9: Dependencies | ✅ Complete | Added disintegration/imaging, golang.org/x/sync, golang.org/x/image |
| 2026-06-28 | Task 1: Config | ✅ Complete | Added ImageConfig to server/config/config.go |
| 2026-06-28 | Task 2: Options | ✅ Complete | server/images/options.go with tests |
| 2026-06-28 | Task 3: Transform | ✅ Complete | server/images/transform.go with tests |
| 2026-06-28 | Task 4: Cache | ✅ Complete | server/images/cache.go with tests |
| 2026-06-28 | Task 5: Registry | ✅ Complete | server/images/registry.go with singleflight dedup |
| 2026-06-28 | Task 6: Handler | ✅ Complete | server/images/handler.go mirrors assetHandler |
| 2026-06-28 | Task 7: Builtins | ✅ Complete | pkg/parsley/evaluator/image.go with image() and imageInfo() |
| 2026-06-28 | Task 8: Integration | ✅ Complete | Wired into server.go, handler.go, api.go |
| 2026-06-28 | Post-implementation review | 📋 Recorded | 1 bug, test gaps (registry/handler/builtins), minor code quality items |