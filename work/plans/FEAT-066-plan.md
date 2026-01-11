---
id: PLAN-041
feature: FEAT-066
title: "Implementation Plan for Database Download/Upload"
status: draft
created: 2025-12-11
---

# Implementation Plan: FEAT-066

## Overview
Add download and upload functionality to the `/__/db` page, allowing developers to easily backup and restore their SQLite database during development. Uploads will include validation, automatic backup, and cache invalidation.

## Prerequisites
- [x] FEAT-066 specification approved
- [x] Existing `/__/db` page structure understood
- [x] SQLite magic bytes validation approach confirmed

## Tasks

### Task 1: Implement Database Download Handler
**Files**: `server/handler.go`
**Estimated effort**: Small

Steps:
1. Add `handleDevDBDownload` method to `devToolsHandler`
2. Read database file from `h.server.config.SQLite`
3. Set `Content-Type: application/octet-stream`
4. Set `Content-Disposition: attachment; filename="<basename>.db"`
5. Stream file to response with `http.ServeContent`

Tests:
- Test download returns correct content-type and disposition headers
- Test download returns actual database file content
- Test download fails gracefully if database file doesn't exist

---

### Task 2: Implement Database Upload Handler
**Files**: `server/handler.go`
**Estimated effort**: Medium

Steps:
1. Add `handleDevDBUpload` method to `devToolsHandler`
2. Parse multipart form with 100MB size limit (`r.ParseMultipartForm(100 << 20)`)
3. Extract uploaded file from form field "database"
4. Validate SQLite magic bytes: `53 51 4C 69 74 65 20 66 6F 72 6D 61 74 20 33 00`
5. Create timestamped backup: `<dbPath>.YYYYMMDD-HHMMSS.backup`
6. Copy uploaded file to database path
7. Call `h.server.ReloadScripts()` to invalidate caches
8. Return JSON success response with redirect

Tests:
- Test upload succeeds with valid SQLite file
- Test upload rejects non-SQLite file (magic bytes validation)
- Test upload creates timestamped backup
- Test upload replaces existing database
- Test upload invalidates caches
- Test upload enforces 100MB size limit
- Test upload fails gracefully on filesystem errors

---

### Task 3: Add Routes to Dev Tools
**Files**: `server/server.go`
**Estimated effort**: Small

Steps:
1. Add route: `GET /__/db/download` → `devTools.handleDevDBDownload`
2. Add route: `POST /__/db/upload` → `devTools.handleDevDBUpload`
3. Ensure routes are under dev mode protection

Tests:
- Test routes are registered correctly
- Test routes are dev-mode only (fail in production)

---

### Task 4: Update Database Management UI
**Files**: `server/handler.go` (database management HTML template)
**Estimated effort**: Small

Steps:
1. Add download button with link to `/__/db/download`
2. Add upload form with file input and submit button
3. Style buttons to match existing dev tools aesthetic
4. Add client-side validation (file required, show progress)
5. Add success/error message display area

Tests:
- Manual test: Download button downloads database
- Manual test: Upload form accepts and uploads database
- Manual test: Upload shows success message on completion
- Manual test: Upload shows error message on validation failure

---

### Task 5: Helper Function for File Copy
**Files**: `server/handler.go`
**Estimated effort**: Small

Steps:
1. Add `copyFile(src, dst string) error` helper function
2. Open source file for reading
3. Create destination file for writing
4. Use `io.Copy` to stream data
5. Handle errors and cleanup

Tests:
- Test copyFile succeeds with valid files
- Test copyFile returns error if source doesn't exist
- Test copyFile returns error if destination is unwritable

---

## Validation Checklist
- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `make dev`
- [ ] Linter passes: `golangci-lint run`
- [ ] Manual E2E test: Download database from `/__/db`
- [ ] Manual E2E test: Upload valid SQLite database
- [ ] Manual E2E test: Upload invalid file (should reject)
- [ ] Manual E2E test: Verify backup file created with timestamp
- [ ] Manual E2E test: Verify cache invalidation works
- [ ] Documentation: Update FAQ if needed
- [ ] BACKLOG.md updated with deferrals (if any)

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-12-11 | Plan created | ✅ Complete | — |
| 2025-12-11 | Task 1: Download handler | ✅ Complete | Added handleDevDBFileDownload |
| 2025-12-11 | Task 2: Upload handler | ✅ Complete | Added handleDevDBFileUpload with validation |
| 2025-12-11 | Task 3: Routes | ✅ Complete | Added /__/db/download and /__/db/upload |
| 2025-12-11 | Task 4: UI update | ✅ Complete | Added download button and upload form |
| 2025-12-11 | Task 5: Helper function | ✅ Complete | Added copyFile utility |
| 2025-12-11 | Task 6: Unit tests | ✅ Complete | 6 tests added, all passing |
| 2025-12-11 | Task 7: Validation | ✅ Complete | All tests pass, build succeeds |

## Deferred Items
None anticipated. If multi-file upload or backup management UI is requested, defer to future feature.
