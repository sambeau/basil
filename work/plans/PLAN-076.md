---
id: PLAN-076
feature: N/A (Code Cleanup)
title: "Dead Code Removal & Deduplication - Quick Wins"
status: complete
created: 2026-01-28
completed: 2026-01-28
---

# Implementation Plan: Code Cleanup Quick Wins

## Overview

Implement the low-risk improvements identified in `work/reports/CODE-ANALYSIS-2025-01.md`:
1. Delete 7 obsolete functions from `server/errors.go`
2. Deduplicate `tableMin`/`tableMax` in evaluator
3. Deduplicate `ServeCSS`/`ServeJS` in bundle server
4. Deduplicate typed dict checks in PLN serializer

## Prerequisites
- [x] Code analysis report completed (CODE-ANALYSIS-2025-01.md)
- [x] Verified functions are truly unused via spec cross-reference

## Tasks

### Task 1: Delete Obsolete Error Functions
**Files**: `server/errors.go`
**Estimated effort**: Small

These 7 functions were part of the old HTML error page system, now replaced by the Parsley template (`dev_error.pars`):

Functions to delete:
1. `makeRelativePath` 
2. `makeMessageRelative`
3. `improveErrorMessage`
4. `renderDevErrorPage`
5. `getSourceContext`
6. `highlightParsley`
7. `escapeForCodeDisplay`

Steps:
1. Search for any remaining callers (should be none)
2. Delete each function
3. Remove any orphaned imports
4. Run tests to confirm no breakage

Tests:
- `go test ./server/...` passes
- Dev error page still renders correctly

---

### Task 2: Deduplicate tableMin/tableMax
**Files**: `pkg/parsley/evaluator/stdlib_table.go`
**Estimated effort**: Small

The `tableMin` and `tableMax` functions are ~35 lines each and differ only in the comparison operator (`<` vs `>`).

Steps:
1. Create shared `tableExtreme(args, env, isMin bool)` function
2. Refactor `tableMin` to call `tableExtreme(args, env, true)`
3. Refactor `tableMax` to call `tableExtreme(args, env, false)`
4. Verify behavior unchanged

Tests:
- Existing `table.min()` tests pass
- Existing `table.max()` tests pass
- Add edge case: empty table, single element, all equal values

---

### Task 3: Deduplicate ServeCSS/ServeJS
**Files**: `server/bundle.go`
**Estimated effort**: Small

`ServeCSS` and `ServeJS` are ~40 lines each and differ only in the Content-Type header.

Steps:
1. Create private `serveBundle(w, r, contentType, content, hash)` function
2. Refactor `ServeCSS` to call `serveBundle` with `"text/css; charset=utf-8"`
3. Refactor `ServeJS` to call `serveBundle` with `"application/javascript; charset=utf-8"`
4. Verify caching headers still work correctly

Tests:
- `go test ./server/...` passes
- Manual test: CSS/JS bundles serve with correct headers

---

### Task 4: Deduplicate Typed Dict Checks
**Files**: `pkg/parsley/pln/serializer.go`
**Estimated effort**: Small

Three identical functions: `isDatetimeDict`, `isPathDict`, `isURLDict`.

Steps:
1. Create `isTypedDict(obj Object, typeName string) bool`
2. Refactor each function to call `isTypedDict(obj, "datetime")` etc.
3. Verify PLN serialization unchanged

Tests:
- Existing PLN serialization tests pass
- Test: datetime, path, URL objects serialize correctly

---

## Validation Checklist
- [x] All tests pass: `go test ./...`
- [x] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] Deadcode reduced (re-run `deadcode ./...`)
- [ ] Dupl clones reduced (re-run `dupl -t 75 .`)

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-01-28 | Task 1: Delete obsolete errors.go functions | ✅ Complete | Deleted ~573 lines of obsolete code (DevError struct, renderDevErrorPage, etc.). Kept SourceLine struct (used by Server.getSourceContext). Removed orphaned imports. |
| 2026-01-28 | Task 2: Deduplicate tableMin/tableMax | ✅ Complete | Created `tableExtreme(t, args, env, findMin)` helper with bool parameter for comparison direction |
| 2026-01-28 | Task 3: Deduplicate ServeCSS/ServeJS | ✅ Complete | Created `serveBundle(w, r, contentType, content, hash)` helper |
| 2026-01-28 | Task 4: Deduplicate typed dict checks | ✅ Complete | Created `isTypedDict(obj, typeName)` helper |
| 2026-01-28 | Validation: Fix test failures | ✅ Complete | Added initPrelude() calls to tests, fixed prelude embed directive, fixed template null regex match bugs, added fallback Location display in dev_error.pars |

## Deferred Items
Items to add to work/BACKLOG.md after implementation:
- Deduplicate `evalDirComputedProperty`/`evalFileComputedProperty` — Medium risk, needs more careful testing
- Expose markdown helpers via `@std/markdown` module — Requires new feature spec
