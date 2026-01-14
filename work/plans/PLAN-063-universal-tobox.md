---
id: PLAN-063
feature: FEAT-088
title: "Implementation Plan for Universal toBox() Method"
status: complete
created: 2026-01-14
---

# Implementation Plan: FEAT-088

## Overview
Extend `toBox()` method from Tables to arrays, dictionaries, and scalar values. Phase 1 covers core functionality; Phase 2 adds polish options.

## Prerequisites
- [x] FEAT-087 Builtin Table complete (Table.toBox() exists)
- [x] Design spec approved (FEAT-088)

## Tasks

### Task 1: Extract Box-Drawing Utilities
**Files**: `pkg/parsley/evaluator/eval_box.go` (new)
**Estimated effort**: Small

Extract shared box-drawing logic from `tableToBox()` into reusable utilities.

Steps:
1. Create `eval_box.go` with box-drawing character constants
2. Create `BoxStyle` struct with style presets (single, double, ascii, rounded)
3. Create helper functions: `boxHLine()`, `boxRow()`, `boxWidth()` (handles unicode)
4. Create `BoxRenderer` struct to manage state (widths, style, alignment)

No tests needed - internal utilities tested via integration.

---

### Task 2: Implement Scalar toBox()
**Files**: `pkg/parsley/evaluator/methods.go`, `pkg/parsley/evaluator/eval_box.go`
**Estimated effort**: Small

Add `toBox()` method to String, Integer, Float, Boolean, Null.

Steps:
1. Add case in `evalStringMethod()` for "toBox"
2. Add case in `evalIntegerMethod()` for "toBox"
3. Add case in `evalFloatMethod()` for "toBox"
4. Add case in `evalBooleanMethod()` for "toBox"
5. Add `nullToBox()` function for null values
6. All delegate to common `scalarToBox(value string) string`

Tests:
- `42.toBox()` → single box with "42"
- `"hello".toBox()` → single box with "hello"
- `true.toBox()` → single box with "true"
- `null.toBox()` → single box with "null"
- `3.14.toBox()` → single box with "3.14"

---

### Task 3: Implement Array toBox() - Vertical
**Files**: `pkg/parsley/evaluator/methods.go`, `pkg/parsley/evaluator/eval_box.go`
**Estimated effort**: Medium

Add `toBox()` method to Array with vertical layout (default).

Steps:
1. Add case in `evalArrayMethod()` for "toBox"
2. Implement `arrayToBox(arr *Array, args []Object, env *Environment) Object`
3. Parse options dict: `{direction: "vertical"|"horizontal", align: "left"|"right"|"center"}`
4. For vertical: render each element in its own row
5. For nested complex values: use `objectToString()` for inline representation

Tests:
- `["a", "b", "c"].toBox()` → vertical stack
- `[1, 2, 3].toBox()` → vertical stack of numbers
- `[].toBox()` → empty box or "(empty)"
- `["single"].toBox()` → single item box
- `[{name: "Alice"}].toBox()` → dict shown as inline string

---

### Task 4: Implement Array toBox() - Horizontal
**Files**: `pkg/parsley/evaluator/eval_box.go`
**Estimated effort**: Small

Add horizontal layout option for arrays.

Steps:
1. Parse `{direction: "horizontal"}` from options
2. Render all elements in a single row with separators
3. Calculate column widths for proper alignment

Tests:
- `["a", "b", "c"].toBox({direction: "horizontal"})` → single row
- `[1, 2, 3].toBox({direction: "horizontal"})` → single row

---

### Task 5: Implement Array toBox() - Grid
**Files**: `pkg/parsley/evaluator/eval_box.go`
**Estimated effort**: Medium

Auto-detect array-of-arrays and render as grid.

Steps:
1. Detect if all elements are arrays (uniform grid)
2. Calculate column widths across all rows
3. Handle jagged arrays: pad shorter rows with empty cells
4. Render as proper grid with internal separators

Tests:
- `[[1,2,3], [4,5,6]].toBox()` → 2x3 grid
- `[[1,2], [3,4,5]].toBox()` → jagged grid, shorter row padded
- `[["a"], ["b"], ["c"]].toBox()` → 3x1 grid

---

### Task 6: Implement Dictionary toBox()
**Files**: `pkg/parsley/evaluator/methods.go`, `pkg/parsley/evaluator/eval_box.go`
**Estimated effort**: Medium

Add `toBox()` method to Dictionary.

Steps:
1. Add case in `evalDictionaryMethod()` for "toBox"
2. Implement `dictToBox(dict *Dictionary, args []Object, env *Environment) Object`
3. Default: two-column layout (key | value)
4. Option `{keys: true}`: render keys only in single row
5. Preserve key order (dictionaries are ordered in Parsley)
6. For nested complex values: use inline representation

Tests:
- `{a: 1, b: 2}.toBox()` → two-column key-value
- `{name: "Alice", age: 30}.toBox()` → two-column
- `{}.toBox()` → empty box
- `{a: 1}.toBox({keys: true})` → single row with just "a"
- `{x: [1,2], y: {z: 3}}.toBox()` → nested shown inline

---

### Task 7: Implement Alignment Option
**Files**: `pkg/parsley/evaluator/eval_box.go`
**Estimated effort**: Small

Add alignment support to all toBox variants.

Steps:
1. Parse `{align: "left"|"right"|"center"}` from options
2. Update `boxRow()` helper to handle alignment
3. Apply to all value types

Tests:
- `[1, 22, 333].toBox({align: "right"})` → right-aligned numbers
- `{a: 1, bb: 22}.toBox({align: "center"})` → centered values

---

### Task 8: Refactor Table.toBox() to Use Shared Utilities
**Files**: `pkg/parsley/evaluator/stdlib_table.go`, `pkg/parsley/evaluator/eval_box.go`
**Estimated effort**: Small

Refactor existing `tableToBox()` to use the new shared box utilities.

Steps:
1. Update `tableToBox()` to use `BoxRenderer`
2. Ensure backward compatibility (no options changes)
3. Remove duplicate box-drawing code from stdlib_table.go

Tests:
- Existing Table.toBox() tests must still pass
- No new tests needed

---

### Task 9: Documentation
**Files**: `docs/parsley/reference.md`, `docs/parsley/manual/builtins/` (new files?)
**Estimated effort**: Small

Document the new toBox() method.

Steps:
1. Add toBox() to Array methods in reference.md
2. Add toBox() to Dictionary methods in reference.md
3. Add toBox() to scalar type sections
4. Add examples for each type
5. Document options: direction, align, keys

Tests: N/A (documentation)

---

## Phase 2 Tasks (Deferred to Later)

### Task P2-1: Style Options
Add `{style: "single"|"double"|"ascii"|"rounded"}` option.

### Task P2-2: Title Option
Add `{title: "My Data"}` option for titled boxes.

### Task P2-3: Width Control
Add `{maxWidth: N}` option with ellipsis truncation.

---

## Validation Checklist
- [x] All tests pass: `go test ./...`
- [x] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [x] Documentation updated
- [ ] work/BACKLOG.md updated with Phase 2 items

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-01-14 | Task 1: Box utilities | ✅ | Created eval_box.go with BoxRenderer |
| 2026-01-14 | Task 2: Scalar toBox | ✅ | String, Integer, Float, Boolean, Null |
| 2026-01-14 | Task 3: Array vertical | ✅ | Default direction |
| 2026-01-14 | Task 4: Array horizontal | ✅ | `{direction: "horizontal"}` |
| 2026-01-14 | Task 5: Array grid | ✅ | Auto-detect or `{direction: "grid"}` |
| 2026-01-14 | Task 6: Dictionary toBox | ✅ | Key-value pairs, `{keys: true}` option |
| 2026-01-14 | Task 7: Alignment | ✅ | `{align: "left"|"right"|"center"}` |
| 2026-01-14 | Task 8: Refactor Table | ✅ | Table.toBox() uses shared BoxRenderer |
| 2026-01-14 | Task 9: Documentation | ✅ | Updated reference.md |

## Deferred Items
Items to add to work/BACKLOG.md after implementation:
- Phase 2 style options (`{style: "double"}`) — Polish, not MVP
- Phase 2 title option (`{title: "..."}`) — Polish, not MVP
- Phase 2 maxWidth truncation — Polish, not MVP
- Color support (`{color: true}`) — Complexity, terminal compatibility issues
- Schema-driven formatting — Significant complexity, unclear requirements
