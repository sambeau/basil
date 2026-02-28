---
id: FEAT-136
title: "Final Database Fixes"
status: draft
priority: high
created: 2026-02-28
author: "@ai"
---

# FEAT-136: Final Database Fixes

## Summary

A quality audit of the database subsystem during FEAT-135 (named parameters) uncovered five pre-existing issues ranging from broken transaction semantics to minor edge cases. This spec groups them into a single feature so they can be fixed, tested, and shipped together.

| # | Issue | Severity | Introduced by FEAT-135? |
|---|-------|----------|------------------------|
| 1 | Raw SQL operators ignore `conn.Tx` (transactions broken) | 🔴 Bug | No — pre-existing |
| 2 | Manual `begin()`/`commit()`/`rollback()` are no-ops | 🔴 Bug | No — pre-existing |
| 3 | `<=??=>` returns `Table` vs `Array` depending on syntax form | ⚠️ Inconsistency | No — pre-existing |
| 4 | `evalQueryOneStatement` missing `rows.Err()` check | ⚠️ Minor | No — pre-existing |
| 5 | Block comment EOF edge case in scanner/rewriter | ⚠️ Trivial | Yes — same class as `$$` fix |

## User Story

As a developer using Parsley's database features, I want transactions and query operators to work correctly and consistently so that my data mutations are safe, my queries behave the same regardless of syntax form, and edge cases don't silently corrupt SQL.

## Acceptance Criteria

### Issue 1: Raw SQL operators must honour active transactions

- [ ] `evalQueryOneStatement` (`<=?=>` statement form) uses `conn.Tx` when a transaction is active
- [ ] `evalQueryManyStatement` (`<=??=>` statement form) uses `conn.Tx` when a transaction is active
- [ ] `evalExecuteStatement` (`<=!=>` statement form) uses `conn.Tx` when a transaction is active
- [ ] `evalDatabaseQueryOne` (`<=?=>` infix form) uses `conn.Tx` when a transaction is active
- [ ] `evalDatabaseQueryMany` (`<=??=>` infix form) uses `conn.Tx` when a transaction is active
- [ ] `evalDatabaseExecute` (`<=!=>` infix form) uses `conn.Tx` when a transaction is active
- [ ] `evalDBConnectionMethod` `"execute"` case uses `conn.Tx` when a transaction is active
- [ ] `evalDBConnectionMethod` `"lastInsertId"` case uses `conn.Tx` when a transaction is active
- [ ] Integration test: within `@transaction`, raw SQL operator writes are rolled back on error

### Issue 2: Manual `begin()`/`commit()`/`rollback()` must manage a real `sql.Tx`

- [ ] `conn.begin()` calls `conn.DB.Begin()` and stores the resulting `*sql.Tx` in `conn.Tx`
- [ ] `conn.commit()` calls `conn.Tx.Commit()` and nils `conn.Tx`
- [ ] `conn.rollback()` calls `conn.Tx.Rollback()` and nils `conn.Tx`
- [ ] Errors from `DB.Begin()`, `Tx.Commit()`, and `Tx.Rollback()` are returned as database errors
- [ ] `begin()` when already in a transaction still returns `DB-0007`
- [ ] `commit()`/`rollback()` when not in a transaction still returns `DB-0006`
- [ ] Integration test: `begin()` + INSERT + `rollback()` → row not present; `begin()` + INSERT + `commit()` → row present

### Issue 3: Harmonize `<=??=>` return type

- [ ] Statement form (`let rows <=??=> conn <SQL>...</SQL>`) returns `Table` (current behaviour, correct — has column metadata)
- [ ] Infix form (`conn <=??=> <SQL>...</SQL>`) returns `Table` (currently returns `Array` — must change)
- [ ] Both forms return a `Table` with `.columns` and `.rows` accessible
- [ ] Existing tests updated to reflect consistent return type
- [ ] Documentation updated if needed

### Issue 4: Add `rows.Err()` check to single-row query functions

- [ ] `evalQueryOneStatement` checks `rows.Err()` after the `rows.Next()` / `rows.Scan()` sequence
- [ ] `evalDatabaseQueryOne` checks `rows.Err()` after the `rows.Next()` / `rows.Scan()` sequence
- [ ] On `rows.Err()` failure, returns `DB-0002` error with the underlying error message

### Issue 5: Fix block comment EOF edge case in scanner and rewriter

- [ ] `scanSQLNamedParams`: unterminated `/* ... EOF` does not lose the final character
- [ ] `rewriteNamedParams`: unterminated `/* ... EOF` copies all remaining characters verbatim
- [ ] Unit tests cover unterminated block comment at EOF

## Design Decisions

- **Centralize DB execution (Issue 1)**: Rather than adding `if conn.Tx != nil` checks to every call site, introduce a helper pair — `connQuery(conn, sql, params...)` and `connExec(conn, sql, params...)` — that route to `conn.Tx` or `conn.DB` as appropriate. All raw SQL evaluator functions and `evalDBConnectionMethod` call through these helpers. `TableBinding.query()`/`.exec()` already have the correct logic and can be left as-is or refactored to use the same helpers.

- **Make manual transaction methods real (Issue 2)**: `begin()`/`commit()`/`rollback()` should manage a real `*sql.Tx`. The TODO comment in the code ("Real transaction support will be added with actual query execution") is now stale — `@transaction` already does this correctly. The manual methods should follow the same pattern. Keep the `InTransaction` bool as a convenience flag, set alongside `conn.Tx`.

- **Standardize on `Table` return type (Issue 3)**: The `Table` type carries column metadata that `Array` does not. Users who want array-like access can use `table.rows`. Returning `Table` from both forms is more useful and less surprising. This is a minor breaking change for code that pattern-matches on the return type of the infix `<=??=>`, but the statement form already returns `Table`, so most real code should be fine.

- **`rows.Err()` placement (Issue 4)**: For single-row queries, `rows.Err()` should be checked after `rows.Next()` returns false (no row) and also after `Scan()` succeeds (to catch iteration errors). The Go documentation requires checking `rows.Err()` after the loop — for single-row queries, this means checking it before returning the result.

- **Block comment fix is minimal (Issue 5)**: Change the loop condition from `i+1 < n` to `i < n`, and handle the `i == n-1` case (last char without closing `*/`) by consuming it. This matches the fix already applied for `$$` dollar-quoting.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Issue 1: Transaction-aware execution helpers

**Root cause**: The six raw SQL evaluator functions and two `evalDBConnectionMethod` cases call `conn.DB.Query()`/`conn.DB.Exec()` directly, bypassing any active `conn.Tx`.

**Affected code** — all in `pkg/parsley/evaluator/eval_database.go`:
- `evalQueryOneStatement` — L41: `conn.DB.Query(sql, params...)`
- `evalQueryManyStatement` — L106: `conn.DB.Query(sql, params...)`
- `evalExecuteStatement` — L175: `conn.DB.Exec(sql, params...)`
- `evalDatabaseQueryOne` — L347: `conn.DB.Query(sql, params...)`
- `evalDatabaseQueryMany` — L398: `conn.DB.Query(sql, params...)`
- `evalDatabaseExecute` — L453: `conn.DB.Exec(sql, params...)`

**Affected code** — `pkg/parsley/evaluator/eval_method_dispatch.go`:
- `evalDBConnectionMethod` `"execute"` case — L99: `conn.DB.Exec(sql)`
- `evalDBConnectionMethod` `"lastInsertId"` case — L111: `conn.DB.QueryRow("SELECT last_insert_rowid()")`

**Fix**: Add two helper functions to `eval_database.go`:

```go
// connQuery routes a query through the active transaction if present.
func connQuery(conn *DBConnection, sql string, params ...any) (*sql.Rows, error) {
    if conn.Tx != nil {
        return conn.Tx.Query(sql, params...)
    }
    return conn.DB.Query(sql, params...)
}

// connExec routes an exec through the active transaction if present.
func connExec(conn *DBConnection, sql string, params ...any) (sql.Result, error) {
    if conn.Tx != nil {
        return conn.Tx.Exec(sql, params...)
    }
    return conn.DB.Exec(sql, params...)
}

// connQueryRow routes a single-row query through the active transaction if present.
func connQueryRow(conn *DBConnection, sql string, params ...any) *sql.Row {
    if conn.Tx != nil {
        return conn.Tx.QueryRow(sql, params...)
    }
    return conn.DB.QueryRow(sql, params...)
}
```

Then replace all `conn.DB.Query(` → `connQuery(conn,`, `conn.DB.Exec(` → `connExec(conn,`, `conn.DB.QueryRow(` → `connQueryRow(conn,` in the affected functions.

**Reference pattern**: `TableBinding.query()` and `TableBinding.exec()` in `stdlib_schema_table_binding.go` already implement this pattern inline (lines ~1202 and ~1223).

### Issue 2: Real transaction management in manual methods

**Affected code**: `pkg/parsley/evaluator/eval_method_dispatch.go` lines 12–44.

**Current behaviour**:
```go
case "begin":
    conn.InTransaction = true
    return &Boolean{Value: true}

case "commit":
    conn.InTransaction = false
    return &Boolean{Value: true}

case "rollback":
    conn.InTransaction = false
    return &Boolean{Value: true}
```

**Fix**:
```go
case "begin":
    if conn.InTransaction {
        return newDatabaseStateError("DB-0007")
    }
    tx, err := conn.DB.Begin()
    if err != nil {
        conn.LastError = err.Error()
        return newDatabaseError("DB-0014", err) // or appropriate code
    }
    conn.Tx = tx
    conn.InTransaction = true
    return &Boolean{Value: true}

case "commit":
    if !conn.InTransaction {
        return newDatabaseStateError("DB-0006")
    }
    if err := conn.Tx.Commit(); err != nil {
        conn.Tx = nil
        conn.InTransaction = false
        conn.LastError = err.Error()
        return newDatabaseError("DB-0015", err)
    }
    conn.Tx = nil
    conn.InTransaction = false
    return &Boolean{Value: true}

case "rollback":
    if !conn.InTransaction {
        return newDatabaseStateError("DB-0006")
    }
    if err := conn.Tx.Rollback(); err != nil {
        conn.Tx = nil
        conn.InTransaction = false
        conn.LastError = err.Error()
        return newDatabaseError("DB-0016", err)
    }
    conn.Tx = nil
    conn.InTransaction = false
    return &Boolean{Value: true}
```

Note: Check which error codes are appropriate. `DB-0015` is already used for commit failure in `@transaction`. May need a new code for rollback failure, or reuse a generic code.

### Issue 3: Harmonize `<=??=>` return type

**Affected code**: `evalDatabaseQueryMany` in `eval_database.go` (infix form), around line 438.

**Current**: Returns `&Array{Elements: results}`.
**Statement form** (`evalQueryManyStatement`): Returns `&Table{Rows: results, Columns: columns}`.

**Fix**: Change `evalDatabaseQueryMany` to:
```go
// Convert []Object results to []*Dictionary for Table
var dictResults []*Dictionary
for _, r := range results {
    if d, ok := r.(*Dictionary); ok {
        dictResults = append(dictResults, d)
    }
}
return &Table{Rows: dictResults, Columns: columns}
```

Or better, accumulate `[]*Dictionary` from the start (the `rowToDict` call already returns `*Dictionary`), matching the pattern used in `evalQueryManyStatement`.

### Issue 4: Missing `rows.Err()` check

**Affected functions**:
- `evalQueryOneStatement` (line ~77) — returns `resultDict` without checking `rows.Err()`
- `evalDatabaseQueryOne` (line ~383) — same

**Fix**: After `rows.Scan()` succeeds and before returning, add:
```go
if rowsErr := rows.Err(); rowsErr != nil {
    conn.LastError = rowsErr.Error()
    return newDatabaseError("DB-0002", rowsErr)
}
```

Note: For the "no rows" path (`!rows.Next()`), `rows.Err()` should also be checked — `rows.Next()` returning false could be due to an error, not just an empty result set.

### Issue 5: Block comment EOF edge case

**Affected code**: `pkg/parsley/evaluator/eval_sql_named_params.go`

**Scanner** (`scanSQLNamedParams`, around line 72):
```go
// Current (buggy):
for i+1 < n {
    if sql[i] == '*' && sql[i+1] == '/' {
        i += 2
        break
    }
    i++
}

// Fixed:
for i < n {
    if i+1 < n && sql[i] == '*' && sql[i+1] == '/' {
        i += 2
        break
    }
    i++
}
```

**Rewriter** (`rewriteNamedParams`, around line 220):
```go
// Current (buggy):
for i+1 < n {
    out.WriteByte(sql[i])
    if sql[i] == '*' && sql[i+1] == '/' {
        out.WriteByte(sql[i+1])
        i += 2
        break
    }
    i++
}

// Fixed:
for i < n {
    if i+1 < n && sql[i] == '*' && sql[i+1] == '/' {
        out.WriteByte(sql[i])
        out.WriteByte(sql[i+1])
        i += 2
        break
    }
    out.WriteByte(sql[i])
    i++
}
```

### Dependencies

- **Internal**: Issues 1 and 2 interact — once `begin()` creates a real `conn.Tx`, the helpers from Issue 1 automatically route queries through it. These two should be implemented together.
- **FEAT-135**: The scanner/rewriter code touched in Issue 5 was introduced/modified by FEAT-135. No conflict expected.
- **`@transaction` DSL**: Already works correctly via `stdlib_dsl_query.go`. No changes needed there.

### Test Plan

| # | Test | Description |
|---|------|-------------|
| 1a | Transaction: raw SQL sees Tx writes | Within `@transaction`, INSERT via `<=!=>`, then SELECT via `<=?=>` — row visible |
| 1b | Transaction: rollback undoes raw SQL writes | Within `@transaction`, INSERT via `<=!=>`, error triggers rollback, SELECT after → no row |
| 1c | Transaction: infix operators also use Tx | Same as 1a/1b but using infix `conn <=!=> ...` and `conn <=?=> ...` |
| 2a | Manual begin/commit | `conn.begin()`, INSERT, `conn.commit()`, SELECT → row present |
| 2b | Manual begin/rollback | `conn.begin()`, INSERT, `conn.rollback()`, SELECT → row not present |
| 2c | Manual begin error: already in tx | `conn.begin()` twice → `DB-0007` error |
| 2d | Manual commit error: not in tx | `conn.commit()` without begin → `DB-0006` error |
| 2e | Begin creates real Tx | After `conn.begin()`, `conn.Tx` is non-nil (unit test) |
| 3a | Statement `<=??=>` returns Table | Verify `.type()` is `"table"` and `.columns` is accessible |
| 3b | Infix `<=??=>` returns Table | Verify `.type()` is `"table"` and `.columns` is accessible (currently returns Array) |
| 3c | Table from both forms is usable | Both results support `.rows`, `.filter()`, etc. |
| 4a | `rows.Err()` on query-one statement | Synthetic error scenario (if feasible) or code review verification |
| 4b | No-row path checks `rows.Err()` | Query returning 0 rows doesn't mask an iteration error |
| 5a | Unterminated block comment in scanner | `scanSQLNamedParams("SELECT /* unterminated")` — no panic, no lost chars |
| 5b | Unterminated block comment in rewriter | `rewriteNamedParams("SELECT /* unterminated :x", ...)` — output includes all original chars |
| 5c | Terminated block comment still works | `SELECT /* comment */ :id` — `:id` still detected |

### Edge Cases & Constraints

1. **Nested transactions** — `@transaction` already rejects nested `@transaction` blocks (`conn.Tx != nil` check). With Issue 2 fixed, `begin()` inside `@transaction` (or vice versa) will also be rejected by the `InTransaction` / `conn.Tx != nil` guard. Verify this works correctly.
2. **Connection methods inside `@transaction`** — `conn.execute()` and `conn.lastInsertId()` inside `@transaction` currently bypass the Tx. After Issue 1, they'll use the Tx. Verify no regressions.
3. **Issue 3 breaking change** — Code that does `let arr = conn <=??=> query; arr[0]` will still work because `Table` supports index access. Code that checks `typeof(result) == "array"` will break. This is acceptable — the statement form already returns `Table`, so the inconsistency is the real bug.
4. **Error code conflicts** — Verify that any new error codes for rollback failure don't collide with existing codes. Check `pkg/parsley/errors/` catalog.

## Related

- Plan: `work/plans/PLAN-116-final-database-fixes.md`
- Discovered during: FEAT-135 (named parameters) quality audit
- Design docs: `work/parsley/design/Database Design.md`, `work/parsley/design/Database Implementation Status.md`
- Existing correct pattern: `TableBinding.query()`/`.exec()` in `stdlib_schema_table_binding.go`
- Transaction DSL: `evalTransactionExpression` in `stdlib_dsl_query.go`
- Backlog: Consider adding an item for `connQueryRow` helper if `lastInsertId` needs it