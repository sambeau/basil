---
id: PLAN-038
feature: FEAT-062
title: "Implementation Plan for Parts V1.1: Auto-Refresh and Lazy Loading"
status: draft
created: 2025-12-10
---

# Implementation Plan: FEAT-062

## Overview

Add auto-refresh (`part-refresh`) and lazy loading (`part-load`) capabilities to the Parts system. These are client-side features that extend the existing JavaScript runtime without requiring server changes.

## Prerequisites

- [x] FEAT-061 (Parts V1) complete
- [x] JavaScript runtime exists in `server/handler.go`
- [ ] Review current runtime to understand integration points

## Tasks

### Task 1: Extend `<Part/>` Component to Accept New Attributes
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small

Steps:
1. Update `renderPartComponent` to recognize `part-refresh`, `part-load`, `part-load-threshold` props
2. Generate corresponding `data-part-refresh`, `data-part-load`, `data-part-load-threshold` attributes on wrapper div
3. Ensure values are properly serialized (numbers as strings)

Tests:
- Part with `part-refresh={5000}` generates `data-part-refresh="5000"`
- Part with `part-load="loaded"` generates `data-part-load="loaded"`
- Part with `part-load-threshold={200}` generates `data-part-load-threshold="200"`
- Attributes work in combination

---

### Task 2: Implement Auto-Refresh in JavaScript Runtime
**Files**: `server/handler.go` (partsRuntimeScript function)
**Estimated effort**: Medium

Steps:
1. Add `refreshIntervals` WeakMap to track active timers
2. Implement `startAutoRefresh(part)` function:
   - Parse `data-part-refresh` for interval
   - Enforce 100ms minimum
   - Create `setInterval` that calls `refresh()`
   - Store timer ID in WeakMap
3. Implement `stopAutoRefresh(part)` function:
   - Clear interval from WeakMap
4. Add Page Visibility API listener:
   - Stop all refresh timers when tab hidden
   - Restart when tab visible
5. Reset refresh timer after manual interaction in `updatePart()`
6. Initialize auto-refresh for Parts without `part-load` on page load

Tests:
- Manual: Part with `part-refresh={2000}` updates every 2 seconds
- Manual: Refresh pauses when switching tabs
- Manual: Clicking a `part-click` button resets the timer
- Manual: Interval less than 100ms is clamped to 100ms

---

### Task 3: Implement Lazy Loading in JavaScript Runtime
**Files**: `server/handler.go` (partsRuntimeScript function)
**Estimated effort**: Medium

Steps:
1. Add `lazyParts` WeakMap to track loaded state
2. Implement `initLazyLoading()` function:
   - Query all `[data-part-load]` elements
   - Create IntersectionObserver per part (for individual thresholds)
   - On intersection: load view, mark as loaded, stop observing
   - Start auto-refresh after load completes (if configured)
3. Respect `data-part-load-threshold` for rootMargin
4. Call `initLazyLoading()` on DOMContentLoaded
5. Re-initialize lazy loading for nested Parts after parent refresh

Tests:
- Manual: Part with `part-load="loaded"` shows empty until scrolled into view
- Manual: Scrolling Part into view triggers load
- Manual: Threshold causes early loading before Part enters viewport
- Manual: Nested lazy Parts work independently

---

### Task 4: Handle Combined Auto-Refresh + Lazy Loading
**Files**: `server/handler.go` (partsRuntimeScript function)
**Estimated effort**: Small

Steps:
1. Ensure auto-refresh only starts after lazy load completes
2. Modify initialization to skip auto-refresh for lazy Parts
3. Start auto-refresh in lazy load callback after `refresh()` completes

Tests:
- Manual: Part with both attributes loads lazily, then starts auto-refresh
- Manual: Auto-refresh doesn't run for invisible lazy Parts

---

### Task 5: Add Tests for New Attributes
**Files**: `pkg/parsley/evaluator/part_component_test.go`
**Estimated effort**: Small

Steps:
1. Add test for `part-refresh` attribute generation
2. Add test for `part-load` attribute generation
3. Add test for `part-load-threshold` attribute generation
4. Add test for combined attributes

Tests:
- Unit tests verify correct HTML output

---

### Task 6: Update Example
**Files**: `examples/parts/handlers/`
**Estimated effort**: Small

Steps:
1. Create `clock.part` demonstrating auto-refresh
2. Create `lazy-content.part` demonstrating lazy loading
3. Update `index.pars` to showcase both features
4. Add CSS for loading indicator visibility

Tests:
- Example runs and demonstrates all V1.1 features

---

### Task 7: Update Documentation
**Files**: Multiple documentation files
**Estimated effort**: Medium

Steps:
1. `docs/guide/parts.md` - Add sections for auto-refresh and lazy loading with examples
2. `docs/parsley/reference.md` - Add `part-refresh`, `part-load`, `part-load-threshold` to Part attributes
3. `docs/parsley/CHEATSHEET.md` - Add gotchas for new attributes
4. `docs/guide/faq.md` - Add FAQ entries for auto-refresh and lazy loading

Tests:
- Documentation is accurate and complete

---

## Validation Checklist
- [ ] All tests pass: `make test`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] Example works: Run `examples/parts/` and verify auto-refresh and lazy loading
- [ ] Documentation updated
- [ ] BACKLOG.md updated with deferrals (if any)
- [ ] FEAT-062 spec updated with implementation notes

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| — | Task 1: Component Attributes | ⬜ Not started | — |
| — | Task 2: Auto-Refresh | ⬜ Not started | — |
| — | Task 3: Lazy Loading | ⬜ Not started | — |
| — | Task 4: Combined Handling | ⬜ Not started | — |
| — | Task 5: Unit Tests | ⬜ Not started | — |
| — | Task 6: Example | ⬜ Not started | — |
| — | Task 7: Documentation | ⬜ Not started | — |

## Deferred Items
Items to add to BACKLOG.md after implementation:
- Custom lazy-load placeholders — Keep V1.1 simple
- Pause refresh when scrolled out of view — Would require tracking visibility per Part
- Multiple refresh intervals (fast/slow) — Complex, wait for user demand
- Refresh on reconnect — Nice-to-have, defer to V1.2
