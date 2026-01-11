---
id: PLAN-027
feature: FEAT-041
title: "Implementation Plan for publicUrl() Asset Function"
status: draft
created: 2025-12-07
---

# Implementation Plan: FEAT-041 publicUrl()

## Overview
Implement `publicUrl()` function that makes private files accessible via content-hashed public URLs. Files remain in place; Basil maintains an in-memory registry mapping hashes to file paths.

## Prerequisites
- [x] Spec complete (FEAT-041)
- [x] Design decisions finalized (content hash, lazy caching, size limits)

## Tasks

### Task 1: Create Asset Registry
**Files**: `server/assets.go` (new)
**Estimated effort**: Medium

Steps:
1. Create `assetRegistry` struct with:
   - `byHash map[string]string` (hash → filepath)
   - `cache map[string]assetEntry` (filepath → cached hash info)
   - `sync.RWMutex` for thread safety
2. Implement `assetEntry` struct (hash, modTime, size)
3. Implement `NewAssetRegistry()` constructor
4. Implement `Register(filepath string) (string, error)`:
   - Stat file, check size limits (warn >10MB, error >100MB)
   - Check cache for existing hash (compare modTime, size)
   - If cache miss/stale: read file, compute SHA256, update cache
   - Return `/__p/{hash}.{ext}` URL
5. Implement `Lookup(hash string) (string, bool)` for serving
6. Implement `Clear()` for server reload

Tests:
- Register file returns correct URL format
- Same file content returns same hash
- Modified file returns new hash
- Cache hit doesn't re-read file (mock or timing test)
- Size warning logged for >10MB file
- Size error returned for >100MB file
- Thread-safe concurrent registration

---

### Task 2: Add HTTP Handler for /__p/ Routes
**Files**: `server/assets.go`, `server/server.go`
**Estimated effort**: Small

Steps:
1. Implement `ServeHTTP` on asset registry (or separate handler)
2. Extract hash from URL path `/__p/{hash}.{ext}`
3. Lookup filepath in registry
4. Return 404 if not found
5. Set cache headers: `Cache-Control: public, max-age=31536000, immutable`
6. Serve file with correct Content-Type
7. Wire handler into server mux for `/__p/` prefix

Tests:
- Registered asset serves correctly
- Unknown hash returns 404
- Cache headers present on response
- Content-Type matches file extension

---

### Task 3: Implement publicUrl() Builtin
**Files**: `pkg/parsley/evaluator/builtins.go`, `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Medium

Steps:
1. Add `publicUrl` to builtins map
2. Implement `evalPublicUrl(args []Object, env *Environment) Object`:
   - Validate single path argument
   - Resolve path relative to current file (using env.Filename)
   - Security check: ensure resolved path within handler root
   - Get asset registry from environment (via BasilCtx or new field)
   - Call registry.Register()
   - Return URL string or error
3. Add `AssetRegistry` field to Environment struct
4. Wire registry into environment in handler setup

Tests:
- `publicUrl(@./icon.svg)` returns URL string
- Relative path resolved correctly
- Path outside handler root returns error
- Non-existent file returns error
- Non-path argument returns type error

---

### Task 4: Wire Registry into Server
**Files**: `server/server.go`, `server/handler.go`, `server/api.go`
**Estimated effort**: Small

Steps:
1. Add `assetRegistry *assetRegistry` field to Server struct
2. Initialize registry in `New()`
3. Pass registry to environment in `parsleyHandler.ServeHTTP()`
4. Pass registry to environment in `apiHandler.ServeHTTP()`
5. Clear registry in `ReloadScripts()` (alongside other caches)

Tests:
- Server starts with registry initialized
- Handler has access to registry
- Reload clears registry

---

### Task 5: Documentation
**Files**: `docs/parsley/reference.md`, `docs/parsley/CHEATSHEET.md`, `docs/guide/basil-quick-start.md`
**Estimated effort**: Small

Steps:
1. Add `publicUrl()` to reference.md under Utility Functions
2. Add example to CHEATSHEET.md in appropriate section
3. Add brief mention in basil-quick-start.md (optional, for component assets)
4. Update FEAT-041 spec with implementation notes

Tests:
- Documentation renders correctly
- Examples are accurate

---

## Validation Checklist
- [x] All tests pass: `make check`
- [x] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [x] Documentation updated
- [ ] Manual test: create component with asset, verify URL works
- [ ] Manual test: modify asset, verify new URL generated
- [ ] Manual test: large file warning appears
- [ ] BACKLOG.md updated with deferrals (if any)

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-12-07 | Task 1: Asset Registry | ✅ Complete | server/assets.go |
| 2025-12-07 | Task 2: HTTP Handler | ✅ Complete | In assets.go |
| 2025-12-07 | Task 3: publicUrl() Builtin | ✅ Complete | public_url.go |
| 2025-12-07 | Task 4: Server Wiring | ✅ Complete | |
| 2025-12-07 | Task 5: Documentation | ✅ Complete | reference.md, CHEATSHEET.md |

## Deferred Items
Items to add to BACKLOG.md after implementation:
- DevTools integration (`/__/assets` page showing registry stats) — Not MVP
- Disk-persisted hash cache for faster cold starts — Optimization
- `publicUrl()` for URLs (fetch remote, cache locally) — Different use case

## Test File Structure

```
server/
├── assets.go           # New: asset registry
├── assets_test.go      # New: registry tests
└── server.go           # Modified: add registry

pkg/parsley/
├── evaluator/
│   ├── builtins.go     # Modified: add publicUrl
│   └── evaluator.go    # Modified: AssetRegistry field
└── tests/
    └── public_url_test.go  # New: builtin tests
```

## Implementation Order

1. **Task 1** (Asset Registry) — Core data structure, can be tested in isolation
2. **Task 2** (HTTP Handler) — Depends on Task 1
3. **Task 4** (Server Wiring) — Depends on Tasks 1, 2
4. **Task 3** (publicUrl Builtin) — Depends on Task 4 (needs registry in env)
5. **Task 5** (Documentation) — After implementation complete
