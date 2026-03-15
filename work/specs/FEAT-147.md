---
id: FEAT-147
title: "1.0 Ship Review §12 Fixes"
status: in-progress
priority: high
created: 2026-03-15
updated: 2026-03-15
author: "@copilot"
plan: PLAN-127
related: FEAT-146, FEAT-144, FEAT-137
---

# FEAT-147: 1.0 Ship Review §12 Fixes

## Summary
Address the actionable findings from the §12 investigation of the 1.0 Ship Review. This covers the should-fix items discovered during the systematic review of 10 previously uninvestigated areas: search subsystem, LiveReload, Git handler, error messages, CHANGELOG, rate limiter, WebAuthn, CLI, Query DSL, and documentation accuracy.

Six items require pre-1.0 action (most are small); the remaining four are confirmed post-1.0.

## User Story
As a Basil user approaching the 1.0 release, I want the framework to not have debug output in production, stale documentation referencing removed features, undocumented limitations, or theoretical security gaps, so that the release is polished and trustworthy.

## Scope

### In Scope (Should-Fix for 1.0)
1. **CHANGELOG catchup** — Document all post-alpha.1 changes (6+ features, bug fixes, namespace moves)
2. **Search: remove debug prints** — Remove `fmt.Printf("[DEBUG]...")` from `server/search.go`
3. **Search: add text file size limit** — Add `MaxTextFileSize` guard in `server/search/scanner.go` before `os.ReadFile`
4. **Git handler: path traversal guard** — Add explicit `containsPathTraversal()` check in `GitHandler.ServeHTTP`
5. **Rate limiter: document limitations** — Add notes about in-memory/ephemeral nature to user docs
6. **Stale CLI docs** — Remove `migrate-let-var` references from FAQ and cheatsheet

### Out of Scope (Confirmed Post-1.0)
- LiveReload tests (dev-mode only, zero production risk)
- WebAuthn handler tests (upstream library handles crypto; TODO noted)
- Error message catalog migration (~51 `fmt.Errorf` calls — incremental work)
- Query DSL backlog cleanup (items #2–#4 already implemented; backlog stale)

## Acceptance Criteria

### §1: CHANGELOG Catchup
- [x] All features since alpha.1 documented: FEAT-142 (Meta), FEAT-143 (A11y), FEAT-144 (DataTable), FEAT-145 (field tag), FEAT-146 (coercion), BUG-025 (short-circuit)
- [x] Prelude component fixes documented
- [x] Namespace moves (`@std/*` → `@basil/*`) documented
- [x] Config consistency fixes documented
- [x] Method registry migration (FEAT-137) documented
- [x] CHANGELOG follows existing format and style

### §2: Search Debug Prints
- [x] All `fmt.Printf("[DEBUG]...")` removed from `server/search.go`
- [x] No other debug print statements left in production code paths
- [x] Tests still pass

### §3: Search Text File Size Limit
- [x] `MaxTextFileSize` constant defined (consistent with existing `MaxPDFSize`/`MaxDOCXSize`)
- [x] `os.ReadFile` in `scanner.go` guarded by file size check
- [x] Oversized files logged/skipped gracefully (not a crash)
- [x] Tests still pass

### §4: Git Path Traversal Guard
- [x] `GitHandler.ServeHTTP` checks for path traversal (`..`) before delegating to `go-git-http`
- [x] Traversal attempts return 400 Bad Request
- [x] Test added for path traversal rejection
- [x] Existing git tests still pass

### §5: Rate Limiter Documentation
- [x] `docs/guide/api-table-binding.md` updated with note about in-memory nature
- [x] Limitation noted: resets on restart, single-instance only
- [x] Workaround/context provided (intentional for target use case)

### §6: Stale CLI Docs
- [x] `docs/guide/faq.md` — `migrate-let-var` section removed or replaced
- [x] `docs/parsley/CHEATSHEET.md` — stale reference removed

## Design Decisions

- **Text file size limit value**: Use 50MB to match `MaxPDFSize` and `MaxDOCXSize` — consistent across all file types.
- **Path traversal check placement**: In `GitHandler.ServeHTTP` before URL rewriting and delegation. Uses the same `strings.Contains(path, "..")` pattern as `siteHandler`.
- **CHANGELOG granularity**: One `[Unreleased]` section with subsections matching the existing style (Added, Changed, Fixed). Will be moved to a version header when a release is cut.
- **Rate limiter docs**: Brief inline note rather than a full section — the limitation is inherent to the design and well-reasoned (per FEAT-034 design doc).

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Files

| File | Change |
|------|--------|
| `CHANGELOG.md` | Add [Unreleased] section with all post-alpha.1 changes |
| `server/search.go` | Remove `fmt.Printf("[DEBUG]...")` statements |
| `server/search/scanner.go` | Add `MaxTextFileSize` const, guard `os.ReadFile` |
| `server/git.go` | Add path traversal check in `ServeHTTP` |
| `server/git_test.go` | Add path traversal test |
| `docs/guide/api-table-binding.md` | Add rate limiter limitation note |
| `docs/guide/faq.md` | Remove `migrate-let-var` section |
| `docs/parsley/CHEATSHEET.md` | Remove `migrate-let-var` reference |
| `work/reports/1.0-SHIP-REVIEW.md` | Update §12 statuses |

### Dependencies
- Depends on: None (all changes are independent)
- Blocks: 1.0 release readiness

### Edge Cases
1. **Text file exactly at size limit** — should be indexed (limit is exclusive upper bound)
2. **Git path traversal with URL encoding** — Go's `net/http` normalizes `%2e%2e` to `..` before reaching the handler, so a simple string check is sufficient
3. **CHANGELOG ordering** — newest changes first within each subsection, matching existing style

## Implementation Notes
*Added during/after implementation*

## Related
- Plan: `work/plans/PLAN-127-feat-147.md`
- Ship Review: `work/reports/1.0-SHIP-REVIEW.md`
- Search subsystem: `server/search/`
- Git handler: `server/git.go`
- Rate limiter design: `work/design/FEAT-034-phases-3-6-design.md`
