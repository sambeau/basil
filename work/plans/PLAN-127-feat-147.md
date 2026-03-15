---
id: PLAN-127
feature: FEAT-147
title: "Implementation Plan for 1.0 Ship Review §12 Fixes"
status: complete
created: 2026-03-15
---

# Implementation Plan: FEAT-147

## Overview
Address the six should-fix items from the §12 investigation of the 1.0 Ship Review. All tasks are independent and can be executed in any order. Most are small (< 10 lines of code) except the CHANGELOG catchup which requires reviewing 30+ commits.

## Prerequisites
- [x] §12 investigation complete — all 10 areas reviewed
- [x] Ship review status table current
- [x] Git working tree clean

## Tasks

### Task 1: CHANGELOG Catchup
**Files**: `CHANGELOG.md`
**Estimated effort**: Medium (reviewing 30+ commits)

The CHANGELOG's last entry is `[1.0.0-alpha.1] - 2026-02-26`. All post-alpha.1 work needs documenting.

Steps:
1. Review `git log --oneline` from alpha.1 to HEAD
2. Add an `[Unreleased]` section above `[1.0.0-alpha.1]`
3. Document under `### Added`, `### Changed`, `### Fixed` subsections:
   - FEAT-137: Method registry migration (all types now use declarative MethodRegistry; `pars describe` works for all types)
   - FEAT-142: Meta component and Page restructure
   - FEAT-143: Accessibility components (SkipLink, VisuallyHidden, LiveRegion, FocusTrap)
   - FEAT-144: DataTable redesign
   - FEAT-145: `<field/>` tag and `record.fieldProps()`
   - FEAT-146: Consistent string coercion for DateTime, Duration, Unit in tables/templates
   - BUG-025: Short-circuit evaluation for `&&` and `||` operators
   - Prelude component fixes (9 bugs across `.length()`, `.format("iso")`, `.floor()`)
   - Config YAML consistency (`cors.maxAge` → `cors.max_age`, `sqlite` → `database.path`, `HttpOnly` → `HTTPOnly`)
   - Stdlib namespace moves (`@std/dev` → `@basil/log`, `@std/html` → `@basil/html`, `@std/api` → `@basil/api`)
   - `@std/mdDoc` renamed to `@std/mddoc`
   - Prelude smoke test added
4. Follow the existing CHANGELOG format and style

Tests:
- N/A (documentation only)

---

### Task 2: Remove Search Debug Prints
**Files**: `server/search.go`
**Estimated effort**: Small

Several `fmt.Printf("[DEBUG]...")` calls were left in `parseSearchOptions` and will print to stdout in production.

Steps:
1. Find all `fmt.Printf("[DEBUG]` calls in `server/search.go`
2. Remove them (they are developer debugging artifacts, not intentional logging)
3. Check if `fmt` import can be removed or is still used elsewhere in the file
4. Run tests: `go test ./server/...`

Tests:
- Existing search tests must still pass

---

### Task 3: Add Text File Size Limit in Search Scanner
**Files**: `server/search/scanner.go`
**Estimated effort**: Small

PDF and DOCX extractors have `MaxPDFSize`/`MaxDOCXSize` (50MB) guards. Markdown/HTML files have no size check before `os.ReadFile()`, which could OOM on a multi-GB `.md` file.

Steps:
1. Add `MaxTextFileSize = 50 * 1024 * 1024` constant (or use existing pattern from `extract_pdf.go`/`extract_docx.go`)
2. In `ScanFolder()`, before `os.ReadFile(path)` for text files, call `os.Stat(path)` and check `fi.Size() > MaxTextFileSize`
3. If oversized, append to `scanErrors` with descriptive message and `continue` to next file
4. Run tests: `go test ./server/search/...`

Tests:
- Existing scanner tests must still pass
- No new test needed (testing with a 50MB+ file is impractical in unit tests; the pattern matches existing PDF/DOCX guards which also lack size-specific tests)

---

### Task 4: Git Handler Path Traversal Guard
**Files**: `server/git.go`, `server/git_test.go`
**Estimated effort**: Small

The `GitHandler.ServeHTTP` strips `/.git` prefix and delegates to `go-git-http` without checking for `..` path traversal. The library's regex routing mitigates this in practice, but there's no deliberate guard.

Steps:
1. In `GitHandler.ServeHTTP`, after the `strings.TrimPrefix(r.URL.Path, "/.git")` line, add a path traversal check
2. If the trimmed path contains `..`, return 400 Bad Request
3. Add test case `TestGitHandler_PathTraversal` that sends a request with `/.git/../../../etc/passwd` and expects 400
4. Run tests: `go test ./server/...`

Tests:
- `TestGitHandler_PathTraversal` — traversal path returns 400
- Existing git tests still pass

---

### Task 5: Document Rate Limiter Limitations
**Files**: `docs/guide/api-table-binding.md`
**Estimated effort**: Small

The in-memory nature of the API rate limiter is intentional (per FEAT-034 design doc) but not documented for users. The docs mention "rate limiting (60 req/min)" without noting it's ephemeral/single-instance.

Steps:
1. Find the rate limiting mention in `docs/guide/api-table-binding.md`
2. Add a brief note: the rate limiter is in-memory, resets on server restart, and is per-instance (not shared across multiple processes)
3. Note this is intentional for the target use case (single-process deployment)

Tests:
- N/A (documentation only)

---

### Task 6: Remove Stale CLI Documentation
**Files**: `docs/guide/faq.md`, `docs/parsley/CHEATSHEET.md`
**Estimated effort**: Small

`pars migrate-let-var` was removed (FEAT-128) but two docs still reference it.

Steps:
1. In `docs/guide/faq.md` (around L66–80), remove or replace the `migrate-let-var` Q&A section. Replace with a note that `let`/`var` syntax was removed in 1.0 and the migration tool is no longer needed (or remove the section entirely if the question is no longer relevant)
2. In `docs/parsley/CHEATSHEET.md` (around L83), remove the `migrate-let-var` reference
3. Scan for any other references: `grep -r "migrate-let-var" docs/`

Tests:
- N/A (documentation only)

---

## Execution Order

All tasks are independent. Recommended order for efficient execution:

1. **Tasks 2, 3, 4** (code changes) — do these first so tests can run
2. **Task 1** (CHANGELOG) — requires reviewing git history
3. **Tasks 5, 6** (doc fixes) — quick edits
4. **Final**: Update ship review `work/reports/1.0-SHIP-REVIEW.md` §12 statuses
5. **Final**: Commit all changes, run full test suite

## Validation Checklist
- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `make dev`
- [ ] No `[DEBUG]` output from search subsystem
- [ ] Git handler rejects `..` paths with 400
- [ ] CHANGELOG covers all post-alpha.1 changes
- [ ] `grep -r "migrate-let-var" docs/` returns no results
- [ ] `grep -r 'fmt.Printf.*DEBUG' server/search.go` returns no results
- [ ] Ship review §12 statuses updated

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-03-15 | Task 2: Search debug prints | ✅ Complete | Removed 4 `fmt.Printf("[DEBUG]...")` from `parseSearchOptions` |
| 2026-03-15 | Task 3: Search text file size limit | ✅ Complete | Added `MaxTextFileSize` (50MB) guard in `scanner.go`; reuses existing `fileInfo` from `d.Info()` |
| 2026-03-15 | Task 4: Git path traversal guard | ✅ Complete | Added `strings.Contains(path, "..")` check returning 400; added `TestGitHandler_PathTraversal` with 3 test cases |
| 2026-03-15 | Task 1: CHANGELOG catchup | ✅ Complete | Added [Unreleased] section with Added/Changed/Fixed covering FEAT-137, FEAT-142–146, BUG-025, prelude fixes, namespace moves, config, search/git hardening |
| 2026-03-15 | Task 5: Rate limiter docs | ✅ Complete | Added inline note to `docs/guide/api-table-binding.md` about in-memory/single-instance nature |
| 2026-03-15 | Task 6: Stale CLI docs | ✅ Complete | Removed `migrate-let-var` from `docs/guide/faq.md` and `docs/parsley/CHEATSHEET.md`; grep confirms zero remaining references |
| 2026-03-15 | Final: Update ship review | ✅ Complete | All §12 areas marked investigated, checklist updated, decision log and specs table updated |

## Deferred Items
Items confirmed as post-1.0 during the §12 investigation (no backlog entry needed — already tracked in ship review):
- LiveReload tests — dev-mode only, zero production risk
- WebAuthn `FinishRegistration`/`FinishLogin` handler tests — upstream lib handles crypto
- Error message catalog migration — ~51 `fmt.Errorf` calls, incremental work
- Query DSL backlog cleanup — items #2–#4 already implemented, backlog stale
- Search `context.Context` support for query timeouts
- Search benchmark tests (already backlog #37)
- Git auth rate limiting (spec FEAT-035 lists it but not implemented)
- Git `go-git-http` library replacement (unmaintained since 2016)
- Rate limiter bucket eviction (slow memory growth under high-cardinality IPs)