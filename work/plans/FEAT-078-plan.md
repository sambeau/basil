---
id: PLAN-051
feature: FEAT-078
title: "Implementation Plan for TableBinding Extended Query Methods"
status: complete
created: 2026-01-04
completed: 2026-02-06
---

# Implementation Plan: FEAT-078

## Overview

Extend the TableBinding API with:
1. Query options (`orderBy`, `select`, `limit/offset`) for `all()` and `where()`
2. Aggregation methods (`count`, `sum`, `avg`, `min`, `max`)
3. Convenience methods (`first`, `last`, `exists`, `findBy`)

All changes are backward compatible — existing code works unchanged.

## Prerequisites

- [x] FEAT-080 complete (@DB available at module scope)
- [x] Existing TableBinding implementation in `stdlib_schema_table_binding.go`

## Tasks

### Phase 1: Query Options Infrastructure

#### Task 1.1: Add Option Parsing Helpers
**Files**: `pkg/parsley/evaluator/stdlib_schema_table_binding.go`
**Estimated effort**: Medium

Steps:
1. Create `QueryOptions` struct to hold parsed options
2. Implement `parseQueryOptions(args []Object, startIndex int) (*QueryOptions, *Error)` to parse options dictionary
3. Implement `buildOrderByClause(opts *QueryOptions) (string, error)` for ORDER BY generation
4. Implement `buildSelectClause(opts *QueryOptions, defaultCols string) string` for column selection
5. Add column name validation using existing `identifierRegex`

```go
type QueryOptions struct {
    OrderBy    []OrderSpec  // [{Column: "name", Dir: "ASC"}, ...]
    Select     []string     // ["id", "name"] or nil for *
    Limit      *int64       // nil = use default/no limit
    Offset     *int64       // nil = 0
    NoLimit    bool         // explicit limit=0 means no limit
}

type OrderSpec struct {
    Column string
    Dir    string // "ASC" or "DESC"
}
```

Tests:
- Parse `{orderBy: "name"}` → single column ASC
- Parse `{orderBy: "name", order: "desc"}` → single column DESC
- Parse `{orderBy: [["age", "desc"], ["name", "asc"]]}` → multi-column
- Parse `{select: ["id", "name"]}` → column list
- Parse `{limit: 10, offset: 5}` → explicit pagination
- Invalid column name → error
- Invalid order direction → error

---

#### Task 1.2: Extend `all()` with Options
**Files**: `pkg/parsley/evaluator/stdlib_schema_table_binding.go`
**Estimated effort**: Small

Steps:
1. Update `executeAll` to accept optional `args` parameter
2. Parse options if provided
3. Build SQL with ORDER BY, SELECT, LIMIT/OFFSET clauses
4. Fall back to current behavior when no options

SQL pattern:
```sql
SELECT {cols} FROM {table} [ORDER BY ...] LIMIT ? OFFSET ?
```

Tests:
- `all()` unchanged behavior (SELECT *, auto-pagination)
- `all({orderBy: "name"})` sorts ascending
- `all({orderBy: "created_at", order: "desc"})` sorts descending
- `all({orderBy: [["a", "desc"], ["b", "asc"]]})` multi-column sort
- `all({select: ["id", "name"]})` returns only specified columns
- `all({limit: 5, offset: 10})` overrides auto-pagination

---

#### Task 1.3: Extend `where()` with Options
**Files**: `pkg/parsley/evaluator/stdlib_schema_table_binding.go`
**Estimated effort**: Small

Steps:
1. Update `executeWhere` to accept optional second argument
2. Parse options if second argument present
3. Build SQL with WHERE + ORDER BY + SELECT clauses
4. No auto-pagination for where() (current behavior)

Tests:
- `where({role: "admin"})` unchanged
- `where({role: "admin"}, {orderBy: "name"})` filters and sorts
- `where({active: true}, {select: ["id", "name"], limit: 5})` filters, projects, limits

---

### Phase 2: Aggregation Methods

#### Task 2.1: Add `count()` Method
**Files**: `pkg/parsley/evaluator/stdlib_schema_table_binding.go`
**Estimated effort**: Small

Steps:
1. Add `"count"` case to `evalTableBindingMethod`
2. Implement `executeCount(args []Object, env *Environment) Object`
3. SQL: `SELECT COUNT(*) FROM table [WHERE ...]`
4. Return `Integer` result

Tests:
- `count()` returns total row count
- `count({role: "admin"})` returns filtered count
- `count()` on empty table returns 0

---

#### Task 2.2: Add `sum()` Method
**Files**: `pkg/parsley/evaluator/stdlib_schema_table_binding.go`
**Estimated effort**: Small

Steps:
1. Add `"sum"` case to `evalTableBindingMethod`
2. Implement `executeSum(args []Object, env *Environment) Object`
3. Validate column name argument
4. SQL: `SELECT SUM(col) FROM table [WHERE ...]`
5. Return number or NULL

Tests:
- `sum("balance")` returns total
- `sum("balance", {active: true})` returns filtered sum
- `sum("balance")` on empty table returns null

---

#### Task 2.3: Add `avg()` Method
**Files**: `pkg/parsley/evaluator/stdlib_schema_table_binding.go`
**Estimated effort**: Small

Steps:
1. Add `"avg"` case to `evalTableBindingMethod`
2. Implement `executeAvg(args []Object, env *Environment) Object`
3. SQL: `SELECT AVG(col) FROM table [WHERE ...]`
4. Return float or NULL

Tests:
- `avg("age")` returns average
- `avg("age", {role: "admin"})` returns filtered average
- `avg("age")` on empty table returns null

---

#### Task 2.4: Add `min()` and `max()` Methods
**Files**: `pkg/parsley/evaluator/stdlib_schema_table_binding.go`
**Estimated effort**: Small

Steps:
1. Add `"min"` and `"max"` cases to `evalTableBindingMethod`
2. Implement `executeMin` and `executeMax` (can share logic via helper)
3. SQL: `SELECT MIN/MAX(col) FROM table [WHERE ...]`
4. Return value or NULL

Tests:
- `min("created_at")` returns earliest
- `max("score")` returns highest
- `min/max` on empty table returns null

---

### Phase 3: Convenience Methods

#### Task 3.1: Add `first()` Method
**Files**: `pkg/parsley/evaluator/stdlib_schema_table_binding.go`
**Estimated effort**: Medium

Steps:
1. Add `"first"` case to `evalTableBindingMethod`
2. Implement `executeFirst(args []Object, env *Environment) Object`
3. Parse arguments: `first()`, `first(n)`, `first(opts)`, `first(n, opts)`
4. Default ORDER BY id ASC
5. Return single record or null (no n), array (with n)

Argument parsing logic:
```
first()           → LIMIT 1, return single/null
first(5)          → LIMIT 5, return array
first({orderBy})  → LIMIT 1 with custom order, return single/null
first(5, {opts})  → LIMIT 5 with options, return array
```

Tests:
- `first()` returns single record or null
- `first(5)` returns array of up to 5
- `first({orderBy: "created_at"})` custom sort
- `first(3, {orderBy: "score", order: "desc"})` n + options

---

#### Task 3.2: Add `last()` Method
**Files**: `pkg/parsley/evaluator/stdlib_schema_table_binding.go`
**Estimated effort**: Small

Steps:
1. Add `"last"` case to `evalTableBindingMethod`
2. Implement `executeLast(args []Object, env *Environment) Object`
3. Like `first()` but default ORDER BY id DESC
4. When custom orderBy provided, reverse direction

Tests:
- `last()` returns last record by id
- `last(5)` returns last 5 records
- `last({orderBy: "created_at"})` orders DESC

---

#### Task 3.3: Add `exists()` Method
**Files**: `pkg/parsley/evaluator/stdlib_schema_table_binding.go`
**Estimated effort**: Small

Steps:
1. Add `"exists"` case to `evalTableBindingMethod`
2. Implement `executeExists(args []Object, env *Environment) Object`
3. SQL: `SELECT 1 FROM table WHERE ... LIMIT 1`
4. Return `Boolean(true/false)`

Tests:
- `exists({email: "x"})` returns true when match
- `exists({email: "nonexistent"})` returns false
- More efficient than `where().length > 0`

---

#### Task 3.4: Add `findBy()` Method
**Files**: `pkg/parsley/evaluator/stdlib_schema_table_binding.go`
**Estimated effort**: Small

Steps:
1. Add `"findBy"` case to `evalTableBindingMethod`
2. Implement `executeFindBy(args []Object, env *Environment) Object`
3. Like `where()` but adds `LIMIT 1` and returns single record or null
4. Support optional second argument for options

Tests:
- `findBy({email: "x"})` returns single record
- `findBy({email: "nonexistent"})` returns null
- `findBy({role: "admin"}, {orderBy: "created_at"})` with sort

---

### Phase 4: Documentation & Tests

#### Task 4.1: Add Comprehensive Tests
**Files**: `pkg/parsley/tests/table_binding_test.go`
**Estimated effort**: Medium

Steps:
1. Add test cases for all new options on `all()` and `where()`
2. Add test cases for each aggregation method
3. Add test cases for each convenience method
4. Test error cases (invalid column names, wrong types)

---

#### Task 4.2: Update Documentation
**Files**: `work/specs/FEAT-078.md`, `docs/parsley/reference.md`
**Estimated effort**: Small

Steps:
1. Mark FEAT-078 as implemented
2. Update reference docs with new methods
3. Add examples to cheatsheet if appropriate

---

## Validation Checklist

- [x] All tests pass: `go test ./...`
- [x] Build succeeds: `make build`
- [x] Linter passes: `gofmt` applied (golangci-lint not available)
- [x] FEAT-078 spec updated to `implemented`
- [x] Reference documentation updated

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-01-04 | Task 1.1: Option parsing helpers | ✅ Complete | QueryOptions, parseQueryOptions, buildOrderByClause, buildSelectClause |
| 2026-01-04 | Task 1.2: Extend all() | ✅ Complete | orderBy, select, limit/offset options |
| 2026-01-04 | Task 1.3: Extend where() | ✅ Complete | Optional second argument for options |
| 2026-01-04 | Task 2.1-2.4: Aggregations | ✅ Complete | count, sum, avg, min, max |
| 2026-01-04 | Task 3.1-3.4: Convenience methods | ✅ Complete | first, last, exists, findBy |
| 2026-01-04 | Task 4.1: Tests | ✅ Complete | 15 new tests, all passing |
| 2026-01-04 | Task 4.2: Documentation | ✅ Complete | Spec marked implemented |
| 2026-02-06 | Documentation update | ✅ Complete | Added comprehensive docs to reference.md for query options, aggregations, convenience methods |
| 2026-02-06 | Code formatting | ✅ Complete | Applied gofmt to all files |

## Deferred Items

Items to add to BACKLOG.md after implementation:
- `groupBy` option for aggregations — Adds complexity, defer to Query Builder
- Predicate-based `where()` with function argument — Performance implications need design

## Implementation Order Recommendation

1. **Phase 1** first — Option parsing is foundational for other features
2. **Phase 2** can be done independently after Phase 1
3. **Phase 3** depends on Phase 1 (uses options parsing)
4. **Phase 4** runs throughout

Each task is designed to be independently testable and committable.