---
id: PLAN-064
feature: FEAT-089
title: "Implementation Plan for toBox() Phase 2 Options"
status: draft
created: 2026-01-14
---

# Implementation Plan: FEAT-089

## Overview
Add style, title, and maxWidth options to the toBox() method. Color support is deferred to Phase 3.

## Prerequisites
- [x] FEAT-088 Universal toBox() complete
- [x] BoxStyle presets exist in eval_box.go

## Tasks

### Task 1: Style Option
**Files**: `pkg/parsley/evaluator/eval_box.go`
**Estimated effort**: Small

Wire up existing BoxStyle presets to the options API.

Steps:
1. Add `style` field to `parseBoxOptions()` return values
2. Parse `{style: "single"|"double"|"ascii"|"rounded"}` from options dict
3. Map style string to BoxStyle preset (BoxStyleSingle, BoxStyleDouble, etc.)
4. Pass style to BoxRenderer in all toBox functions
5. Default remains "single" (current behavior)

Tests:
- `"test".toBox({style: "double"})` → double-line borders
- `[1,2,3].toBox({style: "ascii"})` → ASCII +|- characters
- `{a:1}.toBox({style: "rounded"})` → rounded corners
- Invalid style value → error

---

### Task 2: maxWidth Option
**Files**: `pkg/parsley/evaluator/eval_box.go`
**Estimated effort**: Small

Add truncation support for long values.

Steps:
1. Add `maxWidth` field to `parseBoxOptions()` return values
2. Parse `{maxWidth: N}` from options dict (integer)
3. Create `truncateToWidth(s string, maxWidth int) string` helper
4. Apply truncation in `objectToBoxString()` before rendering
5. Handle unicode properly (truncate by runes, not bytes)
6. Ignore maxWidth ≤ 3 (can't fit char + "...")

Tests:
- `"hello world".toBox({maxWidth: 8})` → "hello..."
- `["long string here", "short"].toBox({maxWidth: 10})` → truncated
- `{key: "very long value"}.toBox({maxWidth: 12})` → value truncated
- `"short".toBox({maxWidth: 20})` → no truncation
- `"test".toBox({maxWidth: 3})` → ignored (too small)

---

### Task 3: Title Option
**Files**: `pkg/parsley/evaluator/eval_box.go`
**Estimated effort**: Medium

Add title row rendering to all box types.

Steps:
1. Add `title` field to `parseBoxOptions()` return values
2. Parse `{title: "string"}` from options dict
3. Add `Title` field to BoxRenderer struct
4. Modify `RenderSingleValue()` to include title row if set
5. Modify `RenderVerticalList()` to include title row if set
6. Modify `RenderHorizontalList()` to include title row if set
7. Modify `RenderGrid()` to include title row if set
8. Modify `RenderKeyValue()` to include title row if set
9. Modify `RenderTable()` to include title row above headers
10. Center title within box width
11. Empty string title is ignored

Title format:
```
┌─────────────────┐
│     Title       │  ← title row (centered)
├─────────────────┤  ← separator
│ content here    │
└─────────────────┘
```

Tests:
- `"hello".toBox({title: "Greeting"})` → title above value
- `[1,2,3].toBox({title: "Numbers"})` → title above list
- `{a:1}.toBox({title: "Data"})` → title above key-value
- `"x".toBox({title: ""})` → no title (empty string ignored)
- Long title expands box width

---

### Task 4: Combined Options
**Files**: `pkg/parsley/evaluator/eval_box.go`
**Estimated effort**: Small

Ensure all options work together.

Steps:
1. Verify style + title work together
2. Verify style + maxWidth work together
3. Verify title + maxWidth work together
4. Verify all three together
5. Verify with existing options (direction, align, keys)

Tests:
- `[1,2,3].toBox({style: "double", title: "Data", maxWidth: 10})`
- `{a:1}.toBox({style: "rounded", title: "Config", align: "center"})`
- `["a","b"].toBox({direction: "horizontal", style: "ascii", title: "Items"})`

---

### Task 5: Table.toBox() Options
**Files**: `pkg/parsley/evaluator/stdlib_table.go`
**Estimated effort**: Small

Extend Table.toBox() to support new options.

Steps:
1. Parse style, title, maxWidth in tableToBox()
2. Pass options to BoxRenderer
3. Apply maxWidth to cell values before rendering

Tests:
- Table with `{style: "double"}`
- Table with `{title: "Users"}`
- Table with `{maxWidth: 15}` truncates long cells

---

### Task 6: Documentation
**Files**: `docs/parsley/reference.md`
**Estimated effort**: Small

Document new options.

Steps:
1. Add style option to toBox() documentation
2. Add title option to toBox() documentation
3. Add maxWidth option to toBox() documentation
4. Add examples for each option
5. Add examples of combined options

---

## Phase 3 Tasks (Deferred)

### Task P3-1: Color Support
Add `{color: true}` option with TTY detection and type-based coloring.
Deferred due to terminal compatibility complexity.

---

## Validation Checklist
- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run` (no new issues)
- [ ] Documentation updated
- [ ] FEAT-089 spec updated with implementation notes

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| | Task 1: Style option | ⬜ | |
| | Task 2: maxWidth option | ⬜ | |
| | Task 3: Title option | ⬜ | |
| | Task 4: Combined options | ⬜ | |
| | Task 5: Table.toBox() options | ⬜ | |
| | Task 6: Documentation | ⬜ | |

## Estimated Total Effort
- Task 1: ~30 min (Small)
- Task 2: ~45 min (Small)
- Task 3: ~1.5 hr (Medium)
- Task 4: ~30 min (Small)
- Task 5: ~30 min (Small)
- Task 6: ~30 min (Small)
- **Total: ~4 hours**
