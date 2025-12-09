---
id: PLAN-034
feature: FEAT-059
title: "Implementation Plan for Error Pages in Prelude"
status: draft
created: 2025-12-09
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

### Task 1: Create Error Page Templates
**Files**: `server/prelude/errors/{404,500,dev_error}.pars`
**Estimated effort**: Small

Steps:
1. Create `server/prelude/errors/` directory
2. Create `404.pars` - Simple, user-friendly "Not Found" page
3. Create `500.pars` - Simple, user-friendly "Server Error" page
4. Create `dev_error.pars` - Detailed error page with stack trace, request info

Tests:
- Verify Parsley syntax is valid
- Verify pages parse during server startup

---

### Task 2: Implement Error Environment Builder
**Files**: `server/errors.go`
**Estimated effort**: Small

Steps:
1. Create `createErrorEnv(r, code, err)` function
2. Build environment with `error.code`, `error.message`
3. In dev mode, add `error.details`, `error.stack`, `error.request`
4. Add `basil.version`, `basil.dev` metadata

Tests:
- `TestCreateErrorEnv_Production` - minimal error data
- `TestCreateErrorEnv_Dev` - detailed error data
- `TestCreateErrorEnv_NoError` - handles nil error

---

### Task 3: Implement Prelude Error Renderer
**Files**: `server/errors.go`
**Estimated effort**: Medium

Steps:
1. Create `renderPreludeError(w, r, code, err)` function
2. Select error page based on code and dev mode
3. Create error environment
4. Evaluate Parsley AST
5. Handle errors in error page (fallback to plain text)
6. Set proper status code and Content-Type

Tests:
- `TestRenderPreludeError_404` - renders 404 page
- `TestRenderPreludeError_500` - renders 500 page
- `TestRenderPreludeError_DevMode` - renders dev error page
- `TestRenderPreludeError_Fallback` - handles error in error page

---

### Task 4: Integrate with Existing Error Handlers
**Files**: `server/errors.go`, `server/handler.go`
**Estimated effort**: Small

Steps:
1. Update `handleError()` to use `renderPreludeError()`
2. Update `handle404()` to use prelude 404 page
3. Ensure fallback to plain HTTP errors still works
4. Test integration with existing error paths

Tests:
- Integration test: Request non-existent route gets 404 page
- Integration test: Handler error gets dev error page (dev mode)
- Integration test: Handler error gets 500 page (prod mode)

---

## Validation Checklist

- [ ] All tests pass: `make test`
- [ ] Build succeeds: `make build`
- [ ] Manual test: Navigate to non-existent page, see branded 404
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
