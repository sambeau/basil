---
id: PLAN-034
feature: FEAT-059
title: "Implementation Plan for Error Pages in Prelude"
status: complete
created: 2025-12-09
completed: 2025-12-09
---

# Implementation Plan: FEAT-059 Error Pages in Prelude

## Overview

Convert error page rendering from plain HTTP responses to Parsley files in the prelude. This provides:
- Consistent, branded error pages
- Detailed errors in dev mode
- Minimal, user-friendly errors in production
- Fail-safe fallback if error page itself fails

## Prerequisites

- [x] FEAT-056 (Prelude Infrastructure) implemented
- [x] Prelude system supports .pars file parsing

## Tasks

### Task 1: Create Error Page Templates ✅
**Status**: COMPLETE
**Files**: `server/prelude/errors/{404,500,dev_error}.pars`

Completed:
1. ✅ Created `server/prelude/errors/` directory
2. ✅ Created `404.pars` - Simple 404 page with gradient purple background
3. ✅ Created `500.pars` - Simple 500 page with gradient pink/red background
4. ✅ Created `dev_error.pars` - Detailed dev error page with conditional sections
5. ✅ Fixed Parsley syntax (`let fn() {}` instead of `fn() {}`)
6. ✅ Fixed conditional rendering (wrapped `if` in `{...}` for interpolation)
7. ✅ Updated prelude embed directive to include `prelude/errors/*`

Tests:
- ✅ Pages parse successfully during server startup (verified in existing tests)

---

### Task 2: Implement Error Environment Builder ✅
**Status**: COMPLETE
**Files**: `server/errors.go`

Completed:
1. ✅ Created `createErrorEnv(r, code, err)` method
2. ✅ Builds environment with `error.code`, `error.message`
3. ✅ In dev mode, adds `error.details`, `error.stack`, `error.request`
4. ✅ Adds `basil.version` metadata
5. ✅ Uses `parsley.ToParsley()` for correct type conversion

Tests:
- ✅ `TestCreateErrorEnv` - validates environment creation

---

### Task 3: Implement Prelude Error Renderer ✅
**Status**: COMPLETE
**Files**: `server/errors.go`

Completed:
1. ✅ Created `renderPreludeError(w, r, code, err)` method
2. ✅ Selects error page based on code (404, 500, dev_error)
3. ✅ Creates error environment using `createErrorEnv()`
4. ✅ Evaluates Parsley AST with error environment
5. ✅ Returns false on failure for fallback handling
6. ✅ Sets proper status code and Content-Type
7. ✅ Created `handle404(w, r)` and `handle500(w, r, err)` helper methods

Tests:
- ✅ `TestRenderPreludeError_404` - renders 404 page
- ✅ `TestRenderPreludeError_500` - renders 500 page
- ✅ `TestHandle404` - validates 404 handler
- ✅ `TestHandle500` - validates 500 handler

---

### Task 4: Integrate with Existing Error Handlers ✅
**Status**: COMPLETE
**Files**: `server/site.go`, `server/handler.go`, `server/server.go`

Completed:
1. ✅ Updated site handler 404 path to use `s.handle404(w, r)`
2. ✅ Updated server.go 404 fallback to use `s.handle404(w, r)`
3. ✅ Updated production mode error handlers to use `s.handle500(w, r, err)`
4. ✅ Added request parameter to error handling methods
5. ✅ Updated all tests to match new signatures
6. ✅ Verified fallback to prelude error pages works

Tests:
- ✅ All existing tests updated and passing
- ✅ Integration verified through existing test suite

---

## Validation Checklist

- [x] All new tests pass
- [x] Build succeeds
- [x] Integrate with existing error handlers
- [x] All tests updated for new error pages
- [ ] Manual test: Navigate to non-existent page, see branded 404
- [ ] Manual test: Trigger error in dev mode, see detailed error page
- [ ] Manual test: Trigger error in prod mode, see generic 500 page
- [ ] Manual test: Trigger error in dev mode, see detailed error
- [ ] Manual test: Error in error page shows plain text fallback

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| | Task 1: Error Page Templates | ⬜ Not Started | — |
| | Task 2: Error Environment | ⬜ Not Started | — |
| | Task 3: Prelude Renderer | ⬜ Not Started | — |
| | Task 4: Integration | ⬜ Not Started | — |

## Deferred Items

Items to add to BACKLOG.md after implementation:
- Custom error pages per route - allow routes to specify custom error handlers
- Error page themes - multiple visual styles for error pages
- Localized error messages - i18n support for error pages
