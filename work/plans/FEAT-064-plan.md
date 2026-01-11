---
id: PLAN-038
feature: FEAT-064
title: "Implementation Plan for Gzip/Zstd Response Compression"
status: draft
created: 2025-12-11
---

# Implementation Plan: FEAT-064

## Overview

Add transparent HTTP response compression to Basil using `klauspost/compress/gzhttp` as a top-level middleware wrapper. This will compress all eligible responses (HTML, CSS, JavaScript, JSON, etc.) automatically.

## Prerequisites

- [x] Spec written (FEAT-064)
- [ ] Review `gzhttp` API options for configuration

## Tasks

### Task 1: Add Compression Config to `config/config.go`

**Files:** `config/config.go`  
**Estimated effort:** Small

Steps:
1. Add `CompressionConfig` struct with fields:
   - `Enabled bool` (default: true)
   - `Level string` (fastest/default/best/none)
   - `MinSize int` (default: 1024)
   - `Zstd bool` (default: false)
2. Add `Compression CompressionConfig` field to `Config` struct
3. Set defaults in `Defaults()` function

Tests:
- Config struct parses correctly from YAML
- Default values are applied

---

### Task 2: Add `go get` Dependency

**Files:** `go.mod`, `go.sum`  
**Estimated effort:** Small

Steps:
1. Run `go get github.com/klauspost/compress`
2. Run `go mod tidy`

Tests:
- Build succeeds

---

### Task 3: Create Compression Middleware in `server/compression.go`

**Files:** `server/compression.go` (new)  
**Estimated effort:** Medium

Steps:
1. Create `newCompressionHandler(handler http.Handler, cfg CompressionConfig) http.Handler`
2. Configure gzhttp options based on config:
   - `gzhttp.MinSize(cfg.MinSize)`
   - `gzhttp.CompressionLevel(level)` based on cfg.Level
3. Optionally enable Zstd if `cfg.Zstd` is true
4. Return wrapped handler or original if compression disabled

```go
package server

import (
    "net/http"
    
    "github.com/klauspost/compress/gzhttp"
    "github.com/sambeau/basil/config"
)

func newCompressionHandler(h http.Handler, cfg config.CompressionConfig) http.Handler {
    if !cfg.Enabled {
        return h
    }
    
    opts := []gzhttp.Option{
        gzhttp.MinSize(cfg.MinSize),
    }
    
    // Map level string to gzip constant
    switch cfg.Level {
    case "fastest":
        opts = append(opts, gzhttp.CompressionLevel(gzip.BestSpeed))
    case "best":
        opts = append(opts, gzhttp.CompressionLevel(gzip.BestCompression))
    case "none":
        return h // Treat "none" as disabled
    default:
        opts = append(opts, gzhttp.CompressionLevel(gzip.DefaultCompression))
    }
    
    wrapper, _ := gzhttp.NewWrapper(opts...)
    return wrapper(h)
}
```

Tests:
- Compression disabled returns original handler
- Level "none" returns original handler
- Valid config creates wrapped handler

---

### Task 4: Integrate Middleware in `server/server.go`

**Files:** `server/server.go`  
**Estimated effort:** Small

Steps:
1. Add compression as the outermost middleware (after logging, before server)
2. Insert after the request logger wrap in `Run()`:

```go
// Current order (around line 690-706):
// 1. Rate limiter
// 2. Proxy aware
// 3. Security headers
// 4. CORS
// 5. Request logging
// 6. → ADD: Compression (outermost)

// Wrap with compression (outermost - compresses all responses)
handler = newCompressionHandler(handler, s.config.Compression)

s.server = &http.Server{
```

Tests:
- Server starts with compression enabled
- Server starts with compression disabled
- Responses include `Content-Encoding: gzip` header

---

### Task 5: Add Tests in `server/compression_test.go`

**Files:** `server/compression_test.go` (new)  
**Estimated effort:** Medium

Steps:
1. Test middleware creation with various configs
2. Test response compression for eligible content types
3. Test small responses are not compressed
4. Test binary content types are not compressed
5. Integration test with full server

Test cases:
- `TestCompressionHandler_Disabled` — returns original handler
- `TestCompressionHandler_GzipResponse` — verifies Content-Encoding: gzip
- `TestCompressionHandler_SmallResponse` — responses < MinSize not compressed
- `TestCompressionHandler_BinaryContent` — images not compressed
- `TestCompressionHandler_Levels` — fastest/default/best all work

---

### Task 6: Update Documentation

**Files:** `docs/guide/basil-quick-start.md`, `basil.example.yaml`  
**Estimated effort:** Small

Steps:
1. Add `compression:` section to `basil.example.yaml` with comments
2. Document compression settings in quick-start guide
3. Note that compression is enabled by default

---

## Validation Checklist

- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] Manual test: `curl -H "Accept-Encoding: gzip" localhost:3000 | file -`
- [ ] Manual test: Response headers show `Content-Encoding: gzip`
- [ ] Documentation updated
- [ ] BACKLOG.md updated with deferrals (if any)

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-12-11 | Task 1: Config | ✅ Complete | Added CompressionConfig to config.go |
| 2025-12-11 | Task 2: Dependency | ✅ Complete | Added klauspost/compress v1.18.2 |
| 2025-12-11 | Task 3: Middleware | ✅ Complete | Implemented newCompressionHandler in server/compression.go |
| 2025-12-11 | Task 4: Integration | ✅ Complete | Integrated as outermost middleware in server.go |
| 2025-12-11 | Task 5: Tests | ✅ Complete | All tests passing in server/compression_test.go |
| 2025-12-11 | Task 6: Documentation | ✅ Complete | Updated basil.example.yaml and quick-start guide |

## Implementation Summary

FEAT-064 has been successfully implemented. All tasks completed.

### Files Changed
- `config/config.go` — Added CompressionConfig struct and defaults
- `server/compression.go` — New middleware implementation
- `server/compression_test.go` — Comprehensive test suite
- `server/server.go` — Integrated compression middleware
- `basil.example.yaml` — Added compression config section
- `docs/guide/basil-quick-start.md` — Added HTTP Compression section
- `go.mod`, `go.sum` — Added klauspost/compress dependency

### Verification
- ✅ All tests pass (`go test ./...`)
- ✅ Build succeeds (`make build`)
- ✅ Manual verification: `Content-Encoding: gzip` header present in responses
- ✅ Compression enabled by default with sensible settings

### Next Steps
Ready for commit and merge to main.

## Deferred Items

Items to add to BACKLOG.md after implementation:
- Zstd support — Enable when more browsers support it natively
- Per-route compression disable — May need `compression: false` on routes for WebSocket, SSE
- Compression statistics/metrics — Track bytes saved in dev mode
