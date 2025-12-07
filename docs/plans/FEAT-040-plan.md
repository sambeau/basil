---
id: PLAN-028
feature: FEAT-040
title: "Implementation Plan for Filesystem-Based Routing"
status: complete
created: 2025-12-08
---

# Implementation Plan: FEAT-040

## Overview
Implement filesystem-based routing for Basil sites. When `site:` is configured, requests are routed to `index.pars` files using a walk-back algorithm. The remaining path after the matched handler becomes `basil.http.request.subpath` as a Path object.

## Prerequisites
- [x] FEAT-041 (publicUrl) implemented — provides `site:` mode support for publicUrl()
- [x] Path object implementation with `.segments` property
- [x] Existing handler infrastructure (scriptCache, parsleyHandler)

## Tasks

### Task 1: Add Site Config Field
**Files**: `config/config.go`, `config/load.go`
**Estimated effort**: Small

Steps:
1. Add `Site string` field to `Config` struct with `yaml:"site"` tag
2. Add mutual exclusion validation: if `site` is set, `routes` must be empty (and vice versa for non-empty routes)
3. Resolve `site` path relative to config base directory in `Load()`

Tests:
- Config with both `site:` and `routes:` should error
- Config with only `site:` should parse successfully
- Site path should be resolved to absolute path

---

### Task 2: Create Site Handler
**Files**: `server/site.go` (new)
**Estimated effort**: Large

Steps:
1. Create `siteHandler` struct with server reference, siteRoot, scriptCache
2. Implement `ServeHTTP` with walk-back routing algorithm:
   - Given URL path `/reports/2025/Q4/`, look for:
     - `{siteRoot}/reports/2025/Q4/index.pars`
     - `{siteRoot}/reports/2025/index.pars`
     - `{siteRoot}/reports/index.pars`
     - `{siteRoot}/index.pars`
   - First match wins
3. Calculate `subpath` as the portion of the URL path not consumed by the matched handler
4. Security checks:
   - Reject paths with `..` (path traversal)
   - Reject paths starting with `.` (dotfiles/hidden)
   - Clean and canonicalize paths
5. Handle trailing slash redirect: `/reports` → `/reports/` (302 redirect)

Tests:
- Walk-back finds correct handler
- Subpath calculation is correct
- 404 when no handler found
- Path traversal attempts blocked
- Dotfile requests blocked

---

### Task 3: Wire Site Handler in Server
**Files**: `server/server.go`
**Estimated effort**: Medium

Steps:
1. In `setupRoutes()`, check if `s.config.Site` is set
2. If site mode, skip normal route registration
3. Create and register site handler for catch-all route "/"
4. Asset handler (`/__p/`) and dev tools still registered normally
5. Auth routes (`/__auth/*`) and git routes (`/.git/`) still work

Tests:
- Site mode routes requests to site handler
- System routes (`/__*`) still work in site mode
- Normal route mode unchanged

---

### Task 4: Add basil.http.request.subpath
**Files**: `server/handler.go`, `server/site.go`
**Estimated effort**: Medium

Steps:
1. In site handler, pass subpath to request context
2. Update `buildBasilContext()` to include `subpath` in `http.request`
3. Create subpath as a Path object with:
   - `__type: "path"`
   - `absolute: false`
   - `segments: [...]` - the segments after the handler
4. Empty subpath (handler at exact URL) has `segments: []`

Tests:
- `basil.http.request.subpath.segments` returns correct array
- Empty subpath for exact handler match
- Multi-segment subpath works correctly

---

### Task 5: Handle Static Files in Site Mode
**Files**: `server/site.go`
**Estimated effort**: Medium

Steps:
1. Check `public_dir` config (global level)
2. Before walk-back routing, check if file exists in public_dir
3. Serve static files directly (same logic as existing root handler)
4. Only apply to non-index paths (don't serve public_dir/index.pars)

Tests:
- Static files served from public_dir
- index.pars files not served as static
- Walk-back routing still works for non-static paths

---

### Task 6: Integrate with FEAT-041 (publicUrl)
**Files**: `server/site.go`
**Estimated effort**: Small

Steps:
1. Set handler's publicDir based on matched index.pars location
2. Pass this to `buildBasilContext()` so `publicUrl()` works correctly
3. Asset handler already exists at `/__p/`, just needs correct publicDir

Tests:
- publicUrl() in site mode resolves relative to handler location
- Asset URLs work correctly

---

### Task 7: Add Comprehensive Tests
**Files**: `server/site_test.go` (new)
**Estimated effort**: Medium

Steps:
1. Test walk-back algorithm with various directory structures
2. Test subpath calculation
3. Test security (path traversal, dotfiles)
4. Test trailing slash redirects
5. Test integration with static files
6. Test error handling (no handler found)

---

### Task 8: Update Documentation
**Files**: `docs/parsley/reference.md`, `docs/parsley/CHEATSHEET.md`, `docs/specs/FEAT-040.md`
**Estimated effort**: Small

Steps:
1. Add `site:` config option to basil configuration reference
2. Document `basil.http.request.subpath` Path object
3. Update FEAT-040 spec with implementation notes
4. Add example site structure

---

## Validation Checklist
- [x] All tests pass: `go test ./...`
- [x] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [x] Documentation updated
- [x] BACKLOG.md updated with deferrals (if any)
- [x] Spec acceptance criteria checked off

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-12-08 | Planning | ✅ Complete | Created implementation plan |
| 2025-12-08 | Task 1: Config | ✅ Complete | Added Site field, mutual exclusion validation |
| 2025-12-08 | Task 2: Site Handler | ✅ Complete | Walk-back routing, security, trailing slash redirect |
| 2025-12-08 | Task 3: Wire Handler | ✅ Complete | Site handler registration in setupRoutes |
| 2025-12-08 | Task 4: Subpath | ✅ Complete | basil.http.request.subpath as Path object |
| 2025-12-08 | Task 5: Static Files | ✅ Complete | public_dir served before walk-back routing |
| 2025-12-08 | Task 6: publicUrl | ✅ Complete | Uses handler directory as publicDir |
| 2025-12-08 | Task 7: Tests | ✅ Complete | server/site_test.go with comprehensive coverage |
| 2025-12-08 | Task 8: Docs | ✅ Complete | reference.md, CHEATSHEET.md, spec updated |

## Deferred Items
Items to add to BACKLOG.md after implementation:
- Per-route caching in site mode (need cache config per index.pars)
- Auth integration in site mode (how to specify auth for different handlers)
