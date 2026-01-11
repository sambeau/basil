---
id: PLAN-010
feature: FEAT-018
title: "Implementation Plan for Standard Library Table Module"
status: complete
created: 2025-12-02
completed: 2025-01-03
---

# Implementation Plan: FEAT-018 (Standard Library Table Module)

## Overview
Implement the Table module as the first standard library addition to Parsley. This establishes the `@std/` import pattern and provides SQL-like operations on arrays of dictionaries.

## Prerequisites
- [x] Spec approved (FEAT-018)
- [ ] Design decision: How to register stdlib modules in evaluator

## Tasks

### Task 1: Standard Library Import Infrastructure
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small

Create the infrastructure for `@std/` imports. The evaluator needs to recognize `@std/module` paths and return built-in module objects instead of loading from filesystem.

Steps:
1. Add `stdlibModules` map in evaluator (module name → loader function)
2. Modify `evalImportExpression` to check for `@std/` prefix
3. If `@std/` prefix, look up module in registry and call loader
4. Return error for unknown stdlib modules

Tests:
- Import unknown `@std/nonexistent` returns clear error
- `@std/table` resolves (after Task 2)

---

### Task 2: Table Object Type
**Files**: `pkg/parsley/object/object.go`
**Estimated effort**: Medium

Define the Table object type that wraps an array and exposes methods.

Steps:
1. Add `TABLE` to ObjectType constants
2. Create `Table` struct with `Rows` field ([]Object, each a Dict)
3. Implement `Type()` → `TABLE`
4. Implement `Inspect()` → `Table(N rows)`
5. Add `Copy()` method for immutability

Tests:
- Table type identification
- Inspect output format
- Copy creates independent instance

---

### Task 3: Table Constructor and Module Export
**Files**: `pkg/parsley/evaluator/stdlib_table.go` (new file)
**Estimated effort**: Medium

Implement the Table constructor function and wire it up as a stdlib module.

Steps:
1. Create new file `stdlib_table.go`
2. Implement `tableConstructor` builtin that:
   - Validates input is array
   - Validates all elements are dicts (or allows empty array)
   - Returns new Table object
3. Create `loadTableModule()` that returns Dict with `Table` key
4. Register in stdlib map

Tests:
- `Table([])` returns empty table
- `Table([{a: 1}, {b: 2}])` succeeds
- `Table("not array")` returns error
- `Table([1, 2, 3])` returns error (elements not dicts)
- Import and destructure: `{Table} = import(@std/table)`

---

### Task 4: Table.rows Property
**Files**: `pkg/parsley/evaluator/stdlib_table.go`
**Estimated effort**: Small

Implement the `rows` property to extract the underlying array.

Steps:
1. Add `rows` as a property access on Table objects
2. Returns copy of internal array (immutability)

Tests:
- `Table(arr).rows` returns array
- Modifying returned array doesn't affect table

---

### Task 5: Table.where() Method
**Files**: `pkg/parsley/evaluator/stdlib_table.go`
**Estimated effort**: Medium

Implement filtering with predicate function.

Steps:
1. Add `where` method to Table
2. Accept function argument (predicate)
3. Iterate rows, call predicate with row as `it`
4. Collect rows where predicate returns truthy
5. Return new Table with filtered rows

Tests:
- `Table(data).where({it.age > 18})` filters correctly
- Empty result returns empty Table
- Non-function argument returns error
- Original table unchanged

---

### Task 6: Table.orderBy() Method
**Files**: `pkg/parsley/evaluator/stdlib_table.go`
**Estimated effort**: Medium

Implement sorting by column(s).

Steps:
1. Add `orderBy` method to Table
2. Handle single column: `orderBy("name")`
3. Handle direction: `orderBy("name", "desc")`
4. Handle multi-column: `orderBy(["a", "b"])` or `orderBy([["a", "asc"], ["b", "desc"]])`
5. Use stable sort to preserve order of equal elements
6. Return new Table with sorted rows

Tests:
- Single column ascending (default)
- Single column descending
- Multi-column sort
- Mixed directions
- Missing column returns error
- String vs number comparison works correctly

---

### Task 7: Table.select() Method
**Files**: `pkg/parsley/evaluator/stdlib_table.go`
**Estimated effort**: Small

Implement column projection.

Steps:
1. Add `select` method to Table
2. Accept array of column names
3. Create new rows with only specified columns
4. Missing columns get null value
5. Return new Table

Tests:
- Select subset of columns
- Select with non-existent column (includes as null)
- Empty columns array returns rows with no keys
- Preserves row order

---

### Task 8: Table.limit() Method
**Files**: `pkg/parsley/evaluator/stdlib_table.go`
**Estimated effort**: Small

Implement row limiting with optional offset.

Steps:
1. Add `limit` method to Table
2. `limit(n)` - first n rows
3. `limit(n, offset)` - n rows starting at offset
4. Handle out-of-bounds gracefully
5. Return new Table

Tests:
- `limit(5)` returns first 5
- `limit(5, 10)` returns rows 10-14
- `limit(100)` on 10-row table returns all 10
- `limit(5, 100)` on 10-row table returns empty
- Negative values return error

---

### Task 9: Aggregation Methods
**Files**: `pkg/parsley/evaluator/stdlib_table.go`
**Estimated effort**: Medium

Implement sum, avg, count, min, max.

Steps:
1. Add `count()` - returns integer row count
2. Add `sum(column)` - sum of numeric values, skip non-numeric
3. Add `avg(column)` - average of numeric values
4. Add `min(column)` - minimum value (works for strings too)
5. Add `max(column)` - maximum value
6. Handle empty tables: count=0, sum=0, avg=null, min/max=null

Tests:
- Each aggregation with valid data
- Empty table handling
- Non-existent column returns error
- Mixed types in column (skip non-numeric for sum/avg)
- String min/max comparison

---

### Task 10: Table.toHTML() Method
**Files**: `pkg/parsley/evaluator/stdlib_table.go`
**Estimated effort**: Medium

Render table as HTML.

Steps:
1. Add `toHTML()` method
2. Generate `<table><thead><tr><th>...</th></tr></thead><tbody>...</tbody></table>`
3. Column order: from first row's keys, or select order if select() was used
4. HTML-escape all values
5. Handle empty table (empty tbody)
6. Return string

Tests:
- Basic HTML structure
- HTML escaping of special chars (`<`, `>`, `&`, `"`)
- Empty table produces valid HTML
- Column order preserved from select()
- Null values render as empty cells

---

### Task 11: Table.toCSV() Method
**Files**: `pkg/parsley/evaluator/stdlib_table.go`
**Estimated effort**: Medium

Render table as CSV string.

Steps:
1. Add `toCSV()` method
2. First row is header (column names)
3. Quote values containing comma, quote, or newline
4. Escape quotes by doubling them
5. Use CRLF line endings (RFC 4180)
6. Return string

Tests:
- Basic CSV output
- Values with commas are quoted
- Values with quotes are quoted and escaped
- Values with newlines are quoted
- Empty table produces header only
- Column order matches toHTML()

---

### Task 12: Method Dispatch for Table Objects
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small

Wire up method calls on Table objects to the implementations.

Steps:
1. Add case for `*object.Table` in method call evaluation
2. Dispatch to appropriate method based on name
3. Handle unknown method error

Tests:
- All methods callable via dot notation
- Unknown method returns clear error
- Method chaining works

---

## Validation Checklist
- [x] All tests pass: `go test ./...`
- [x] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] Documentation updated (reference.md, CHEATSHEET.md)
- [ ] Example added to docs/guide/
- [ ] BACKLOG.md updated with deferrals

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-01-03 | Task 1: Stdlib infrastructure | ✅ Complete | Added @std/ handling in evalImport, created stdlib registry pattern |
| 2025-01-03 | Task 2: Table object type | ✅ Complete | Added TABLE_OBJ constant, Table struct with Rows/Columns |
| 2025-01-03 | Task 3: Constructor & module | ✅ Complete | Created stdlib_table.go with TableConstructor |
| 2025-01-03 | Task 4: rows property | ✅ Complete | Added property dispatch in evalDotExpression |
| 2025-01-03 | Task 5: where() | ✅ Complete | Filtering with fn(row) predicate |
| 2025-01-03 | Task 6: orderBy() | ✅ Complete | Single/multi column, asc/desc support |
| 2025-01-03 | Task 7: select() | ✅ Complete | Column projection |
| 2025-01-03 | Task 8: limit() | ✅ Complete | Row limiting with optional offset |
| 2025-01-03 | Task 9: Aggregations | ✅ Complete | sum, avg, count, min, max |
| 2025-01-03 | Task 10: toHTML() | ✅ Complete | Clean HTML with escaping |
| 2025-01-03 | Task 11: toCSV() | ✅ Complete | RFC 4180 compliant |
| 2025-01-03 | Task 12: Method dispatch | ✅ Complete | evalTableMethod in stdlib_table.go |

## Implementation Notes

### Import Workaround
The lexer only recognizes path literals starting with `@/`, `@./`, `@~/`, or `@../`. For stdlib imports, use string syntax: `import("std/table")`. A future enhancement should add `@std/` to the lexer.

### Files Created/Modified
- **New**: `pkg/parsley/evaluator/stdlib_table.go` - Complete Table module implementation (~650 lines)
- **Modified**: `pkg/parsley/evaluator/evaluator.go`:
  - Added TABLE_OBJ constant
  - Added Table struct
  - Added @std/ handling in evalImport
  - Added case *Table in method dispatch
  - Added Table property handling in evalDotExpression
  - Added case *StdlibBuiltin in applyFunction
  - Modified evalDictDestructuringAssignment for StdlibModuleDict
- **New**: `pkg/parsley/tests/stdlib_table_test.go` - Comprehensive tests for all Table functionality

## Deferred Items
Items to add to BACKLOG.md after implementation:
- `groupBy(column)` — Complex aggregation, needs design for return type
- `join(table, column)` — SQL joins, needs careful design
- Column transforms — `transform(col, fn)`, `addColumn(name, fn)`
- `distinct()` — Deduplication
- `first()` / `last()` — Single row access
- `toJSON()` — JSON output
- `fromCSV(string)` — CSV parsing into Table
