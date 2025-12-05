---
id: PLAN-017
feature: FEAT-027
title: "Implementation Plan for Collection Insert Methods"
status: complete
created: 2025-12-05
---

# Implementation Plan: FEAT-027 Collection Insert Methods

## Overview
Add 8 insert/append methods across arrays, dictionaries, and tables to enable positional insertion of elements in Parsley collections.

## Prerequisites
- [x] FEAT-026: Ordered Dictionaries (completed 2025-12-04)
- [x] Design decisions documented in spec

## Tasks

### Task 1: Array `insert` Method
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Small

Steps:
1. Add `"insert"` to `arrayMethods` slice
2. Add `case "insert":` in `evalArrayMethod`
3. Validate args: exactly 2 (index, value)
4. Handle negative indices (convert to positive)
5. Bounds check index
6. Create new array with element inserted before index
7. Return new array (immutable)

Tests (`pkg/parsley/tests/array_insert_test.go`):
- Insert at beginning (index 0)
- Insert in middle
- Insert at end (index == length)
- Negative index (-1 = before last)
- Out-of-bounds error
- Insert into empty array

---

### Task 2: Dictionary `insertAfter` and `insertBefore` Methods
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Medium

Steps:
1. Add `"insertAfter"`, `"insertBefore"` to `dictionaryMethods` slice (create if missing)
2. Add cases in `evalDictionaryMethod`
3. Validate args: exactly 3 (existingKey, newKey, value)
4. Error if existingKey doesn't exist
5. Error if newKey already exists
6. Create new dictionary with new key inserted at correct position
7. Copy all key-value pairs, inserting new one at right position

Tests (`pkg/parsley/tests/dict_insert_test.go`):
- `insertAfter` - insert after first key
- `insertAfter` - insert after middle key
- `insertAfter` - insert after last key
- `insertBefore` - insert before first key
- `insertBefore` - insert before middle key
- `insertBefore` - insert before last key
- Error: existing key not found
- Error: new key already exists
- Verify order is preserved

---

### Task 3: Table Row Methods (`appendRow`, `insertRowAt`)
**Files**: `pkg/parsley/evaluator/stdlib_table.go`
**Estimated effort**: Medium

Steps:
1. Add `case "appendRow":` in `EvalTableMethod`
2. Add `case "insertRowAt":` in `EvalTableMethod`
3. Validate row is a dictionary with matching columns
4. For `appendRow`: create new table with row appended
5. For `insertRowAt`: validate index, insert row at position
6. Update method list in error message

Tests (`pkg/parsley/tests/table_row_insert_test.go`):
- `appendRow` - add row to end
- `appendRow` - add to empty table
- `insertRowAt(0, row)` - insert at beginning
- `insertRowAt` - insert in middle
- `insertRowAt(len, row)` - insert at end (same as append)
- Error: row missing columns
- Error: row has extra columns (decide: error or ignore)
- Error: index out of bounds

---

### Task 4: Table Column Methods (`appendCol`, `insertColAfter`, `insertColBefore`)
**Files**: `pkg/parsley/evaluator/stdlib_table.go`
**Estimated effort**: Large

Steps:
1. Add three cases in `EvalTableMethod`
2. For each method, handle two signatures:
   - Values array: `appendCol(name, [v1, v2, v3])`
   - Function: `appendCol(name, fn(row) { ... })`
3. Validate values array length matches row count
4. For function: iterate rows, call function with row dict, collect results
5. Create new table with column inserted at correct position
6. Update method list in error message

Tests (`pkg/parsley/tests/table_col_insert_test.go`):
- `appendCol` with values array
- `appendCol` with function
- `insertColAfter` with values
- `insertColAfter` with function
- `insertColBefore` with values
- `insertColBefore` with function
- Error: values array length mismatch
- Error: existing column not found
- Error: new column name already exists
- Function receives correct row data

---

### Task 5: Documentation
**Files**: `docs/parsley/reference.md`, `docs/parsley/CHEATSHEET.md`
**Estimated effort**: Small

Steps:
1. Add array `insert` method to reference.md
2. Add dictionary `insertAfter`/`insertBefore` methods
3. Add table row methods section
4. Add table column methods section
5. Add examples for each method
6. Update CHEATSHEET.md if any gotchas discovered

---

### Task 6: Final Validation
**Files**: All
**Estimated effort**: Small

Steps:
1. Run full test suite
2. Run linter
3. Build both binaries
4. Manual testing with example scripts
5. Update spec status to complete

---

## Validation Checklist
- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] Documentation updated
- [ ] BACKLOG.md updated with deferrals (if any)

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-12-05 | Task 1: Array insert | ✅ Complete | — |
| 2025-12-05 | Task 2: Dict insert | ✅ Complete | — |
| 2025-12-05 | Task 3: Table rows | ✅ Complete | — |
| 2025-12-05 | Task 4: Table cols | ✅ Complete | — |
| 2025-12-05 | Task 5: Tests | ✅ Complete | 4 test files |
| 2025-12-05 | Task 6: Docs | ✅ Complete | reference.md updated |
| 2025-12-05 | Task 7: Validation | ✅ Complete | All tests pass |

## Deferred Items
Items to add to BACKLOG.md after implementation:
- *None anticipated*

## Implementation Order
Recommended order (simplest to most complex):
1. **Array insert** — Simplest, establishes pattern
2. **Dictionary insert** — Builds on ordered dict work
3. **Table row methods** — Table infrastructure
4. **Table column methods** — Most complex (function support)
5. **Documentation** — After all code complete
6. **Validation** — Final step

## Estimated Total Effort
- Small tasks: 3 × ~30 min = 1.5 hrs
- Medium tasks: 2 × ~1 hr = 2 hrs  
- Large tasks: 1 × ~2 hrs = 2 hrs
- **Total**: ~5-6 hours
