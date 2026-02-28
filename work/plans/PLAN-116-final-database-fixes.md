---
id: PLAN-116
feature: FEAT-136
title: "Implementation Plan for Final Database Fixes"
status: done
created: 2026-02-28
---

# Implementation Plan: FEAT-136 Final Database Fixes

## Overview

Five database issues discovered during the FEAT-135 quality audit. Ordered by dependency and severity:

1. **Transaction-aware execution helpers** (Issue 1) — foundation for everything else
2. **Real manual transaction methods** (Issue 2) — depends on Issue 1 helpers
3. **Harmonize `<=??=>` return type** (Issue 3) — independent
4. **Missing `rows.Err()` check** (Issue 4) — independent, quick
5. **Block comment EOF edge case** (Issue 5) — independent, trivial

Issues 1 and 2 must be implemented together (they interact). Issues 3–5 are independent of each other and of 1–2.

## Prerequisites

- [x] FEAT-135 merged (scanner/rewriter code exists)
- [x] `@transaction` DSL already works correctly (reference implementation in `stdlib_dsl_query.go`)
- [x] `TableBinding.query()`/`.exec()` already implement the Tx-aware pattern (reference in `stdlib_schema_table_binding.go`)
- [ ] Clean working tree (`git status`)

## Error Code Inventory

Before starting, confirm available error codes. Current DB error landscape:

| Code | Used For | Location |
|------|----------|----------|
| DB-0006 | No transaction in progress | `errors.go` (catalog) |
| DB-0007 | Already in a transaction | `errors.go` (catalog) |
| DB-0013 | Nested transaction rejected | `stdlib_dsl_query.go` (hardcoded) |
| DB-0014 | Failed to begin transaction | `stdlib_dsl_query.go` (hardcoded) |
| DB-0015 | Transaction commit failed | `stdlib_dsl_query.go` (hardcoded) |
| DB-0016 | Cannot update: no PK | `errors.go` (catalog) |
| DB-0017 | Cannot delete: no PK | `errors.go` (catalog) |
| DB-0018 | RETURNING not supported on MySQL | `errors.go` (catalog) |

**Plan**: Reuse DB-0014 for `begin()` failure, DB-0015 for `commit()` failure. Add DB-0019 for `rollback()` failure. Optionally register DB-0013–DB-0015 in the error catalog to consolidate hardcoded errors (nice-to-have, not required).

## Tasks

### Task 1: Add `connQuery`/`connExec`/`connQueryRow` helpers

**Files**: `pkg/parsley/evaluator/eval_database.go`
**Estimated effort**: Small

These helpers route database calls through the active transaction when present, or fall back to the raw `*sql.DB`. This is the foundation for Issues 1 and 2.

Steps:
1. Add three helper functions at the top of `eval_database.go` (after imports, before `evalQueryOneStatement`):
   ```go
   func connQuery(conn *DBConnection, query string, args ...any) (*sql.Rows, error) {
       if conn.Tx != nil {
           return conn.Tx.Query(query, args...)
       }
       return conn.DB.Query(query, args...)
   }

   func connExec(conn *DBConnection, query string, args ...any) (sql.Result, error) {
       if conn.Tx != nil {
           return conn.Tx.Exec(query, args...)
       }
       return conn.DB.Exec(query, args...)
   }

   func connQueryRow(conn *DBConnection, query string, args ...any) *sql.Row {
       if conn.Tx != nil {
           return conn.Tx.QueryRow(query, args...)
       }
       return conn.DB.QueryRow(query, args...)
   }
   ```

2. Replace all direct `conn.DB.Query(` calls with `connQuery(conn,` in:
   - `evalQueryOneStatement` (L41)
   - `evalQueryManyStatement` (L106)
   - `evalDatabaseQueryOne` (L347)
   - `evalDatabaseQueryMany` (L398)

3. Replace all direct `conn.DB.Exec(` calls with `connExec(conn,` in:
   - `evalExecuteStatement` (L175)
   - `evalDatabaseExecute` (L453)

4. In `pkg/parsley/evaluator/eval_method_dispatch.go`, replace:
   - `conn.DB.Exec(sql)` in the `"execute"` case (~L99) with `connExec(conn, sql)`
   - `conn.DB.QueryRow(...)` in the `"lastInsertId"` case (~L111) with `connQueryRow(conn, ...)`

5. Verify build: `go build ./pkg/parsley/...`

Tests (deferred to Task 3 — tested together with manual transactions):
- Helpers are internal; tested indirectly through integration tests

---

### Task 2: Make `begin()`/`commit()`/`rollback()` manage a real `sql.Tx`

**Files**: `pkg/parsley/evaluator/eval_method_dispatch.go`
**Estimated effort**: Small

Steps:
1. Rewrite the `"begin"` case:
   ```go
   case "begin":
       if len(args) != 0 {
           return newArityError("begin", len(args), 0)
       }
       if conn.InTransaction {
           return newDatabaseStateError("DB-0007")
       }
       tx, txErr := conn.DB.Begin()
       if txErr != nil {
           conn.LastError = txErr.Error()
           return newDatabaseError("DB-0014", txErr)
       }
       conn.Tx = tx
       conn.InTransaction = true
       return &Boolean{Value: true}
   ```

2. Rewrite the `"commit"` case:
   ```go
   case "commit":
       if len(args) != 0 {
           return newArityError("commit", len(args), 0)
       }
       if !conn.InTransaction {
           return newDatabaseStateError("DB-0006")
       }
       if commitErr := conn.Tx.Commit(); commitErr != nil {
           conn.Tx = nil
           conn.InTransaction = false
           conn.LastError = commitErr.Error()
           return newDatabaseError("DB-0015", commitErr)
       }
       conn.Tx = nil
       conn.InTransaction = false
       return &Boolean{Value: true}
   ```

3. Rewrite the `"rollback"` case:
   ```go
   case "rollback":
       if len(args) != 0 {
           return newArityError("rollback", len(args), 0)
       }
       if !conn.InTransaction {
           return newDatabaseStateError("DB-0006")
       }
       if rbErr := conn.Tx.Rollback(); rbErr != nil {
           conn.Tx = nil
           conn.InTransaction = false
           conn.LastError = rbErr.Error()
           return newDatabaseError("DB-0019", rbErr)
       }
       conn.Tx = nil
       conn.InTransaction = false
       return &Boolean{Value: true}
   ```

4. Register DB-0019 in `pkg/parsley/errors/errors.go`:
   ```go
   "DB-0019": {
       Class:    ClassDatabase,
       Template: "Transaction rollback failed: {{.GoError}}",
   },
   ```

5. Check whether `newDatabaseError` accepts the code + `error` signature already used by DB-0014/DB-0015 in `@transaction`. If those are hardcoded `&Error{}` literals, use the same pattern for consistency. If a catalog-based helper exists, use that.

6. Verify build: `go build ./pkg/parsley/...`

---

### Task 3: Integration tests for transactions (Issues 1 + 2)

**Files**: `pkg/parsley/tests/database_test.go`
**Estimated effort**: Medium

Add a new test function `TestDatabaseTransactions` with subtests. All tests use `:memory:` SQLite.

Steps:
1. Add test: **Manual begin/commit** — `db.begin()`, INSERT via `<=!=>`, `db.commit()`, SELECT via `<=?=>` → row present.

2. Add test: **Manual begin/rollback** — `db.begin()`, INSERT via `<=!=>`, `db.rollback()`, SELECT via `<=?=>` → null (row not present).

3. Add test: **begin() twice errors** — `db.begin()` then `db.begin()` → error containing `DB-0007`.

4. Add test: **commit() without begin errors** — `db.commit()` → error containing `DB-0006`.

5. Add test: **rollback() without begin errors** — `db.rollback()` → error containing `DB-0006`.

6. Add test: **@transaction + raw SQL operators** — `@transaction(db)` block with INSERT via `<=!=>`, verify row visible inside block via `<=?=>`, then test that error in block triggers rollback (row not present after).

7. Add test: **Infix operators use Tx** — same as test 1 but using infix form `db <=!=> ...` and `db <=?=> ...` inside manual `begin()`/`commit()`.

8. Add test: **conn.execute() uses Tx** — `db.begin()`, `db.execute("INSERT ...")`, `db.rollback()`, SELECT → row not present.

9. Run tests: `go test ./pkg/parsley/tests/ -run TestDatabaseTransactions -v`

---

### Task 4: Harmonize `<=??=>` return type (Issue 3)

**Files**: `pkg/parsley/evaluator/eval_database.go`
**Estimated effort**: Small

Steps:
1. In `evalDatabaseQueryMany` (~L386–440), change the results accumulator from `[]Object` to `[]*Dictionary`:
   ```go
   var results []*Dictionary
   ```

2. Change the append to use the `*Dictionary` directly (no cast needed — `rowToDict` returns `*Dictionary`):
   ```go
   resultDict := rowToDict(columns, values, env)
   results = append(results, resultDict)
   ```

3. Change the return from `&Array{Elements: results}` to:
   ```go
   return &Table{Rows: results, Columns: columns}
   ```

4. Search for any existing tests that assert the infix `<=??=>` returns an `Array` and update them to expect `Table`:
   ```bash
   grep -n "<=\?\?=>" pkg/parsley/tests/database_test.go
   ```

5. Add a test in `TestSQLiteQueries` or a new `TestQueryManyReturnType` function:
   - Statement form `let rows <=??=> db <SQL>...</SQL>` → verify `rows` is a `Table` with `.columns`
   - Infix form `db <=??=> <SQL>...</SQL>` → verify result is a `Table` with `.columns`
   - Both should support index access (`result[0]`) for backward compatibility

6. Run tests: `go test ./pkg/parsley/tests/ -run TestSQLite -v`

---

### Task 5: Add `rows.Err()` check to single-row query functions (Issue 4)

**Files**: `pkg/parsley/evaluator/eval_database.go`
**Estimated effort**: Small

Steps:
1. In `evalQueryOneStatement`, add `rows.Err()` check in two places:

   a. After the "no rows" branch (`!rows.Next()`), before returning NULL:
   ```go
   if !rows.Next() {
       if rowsErr := rows.Err(); rowsErr != nil {
           conn.LastError = rowsErr.Error()
           return newDatabaseError("DB-0002", rowsErr)
       }
       return assignQueryResult(node.Names, NULL, env, node.IsLet)
   }
   ```

   b. After successful `rows.Scan()`, before returning the result dict:
   ```go
   if scanErr := rows.Scan(valuePtrs...); scanErr != nil {
       conn.LastError = scanErr.Error()
       return newDatabaseError("DB-0004", scanErr)
   }

   if rowsErr := rows.Err(); rowsErr != nil {
       conn.LastError = rowsErr.Error()
       return newDatabaseError("DB-0002", rowsErr)
   }

   resultDict := rowToDict(columns, values, env)
   ```

2. Apply the same two changes to `evalDatabaseQueryOne`.

3. Verify build: `go build ./pkg/parsley/...`
4. Run existing tests to ensure no regressions: `go test ./pkg/parsley/tests/ -run TestSQLite -v`

Tests:
- The `rows.Err()` path is difficult to trigger in integration tests (requires a connection error mid-iteration). This is a defensive correctness fix verified by code inspection and existing test non-regression. No new test needed — the Go docs mandate this pattern.

---

### Task 6: Fix block comment EOF edge case (Issue 5)

**Files**: `pkg/parsley/evaluator/eval_sql_named_params.go`
**Estimated effort**: Small

Steps:
1. In `scanSQLNamedParams`, fix the block comment loop (~L72):

   **Before:**
   ```go
   for i+1 < n {
       if sql[i] == '*' && sql[i+1] == '/' {
           i += 2
           break
       }
       i++
   }
   ```

   **After:**
   ```go
   for i < n {
       if i+1 < n && sql[i] == '*' && sql[i+1] == '/' {
           i += 2
           break
       }
       i++
   }
   ```

2. In `rewriteNamedParams`, fix the block comment loop (~L220):

   **Before:**
   ```go
   for i+1 < n {
       out.WriteByte(sql[i])
       if sql[i] == '*' && sql[i+1] == '/' {
           out.WriteByte(sql[i+1])
           i += 2
           break
       }
       i++
   }
   ```

   **After:**
   ```go
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

3. Add unit tests in `pkg/parsley/evaluator/eval_sql_named_params_test.go`:

   a. `TestScanSQLNamedParams_UnterminatedBlockComment` — `scanSQLNamedParams("SELECT /* unterminated")` → no panic, `hasNamed=false`, `hasPositional=false`

   b. `TestScanSQLNamedParams_UnterminatedBlockCommentWithParam` — `scanSQLNamedParams("SELECT /* comment with :x")` → `hasNamed=false` (`:x` inside comment)

   c. `TestScanSQLNamedParams_TerminatedBlockComment` — `scanSQLNamedParams("SELECT /* comment */ :id FROM t")` → `names=["id"]` (regression test)

   d. `TestRewriteNamedParams_UnterminatedBlockComment` — rewrite `"SELECT /* unterminated"` → output equals input (all chars preserved)

   e. `TestRewriteNamedParams_UnterminatedBlockCommentLastChar` — rewrite `"SELECT /*x"` → output equals `"SELECT /*x"` (the `x` is not lost)

4. Run: `go test ./pkg/parsley/evaluator/ -run TestScanSQLNamedParams_Unterminated -v`
5. Run: `go test ./pkg/parsley/evaluator/ -run TestRewriteNamedParams_Unterminated -v`

---

### Task 7: Run full test suite and commit

**Files**: All changed files
**Estimated effort**: Small

Steps:
1. Run full test suite: `go test ./pkg/parsley/...`
2. Run linter: `golangci-lint run`
3. Fix any issues
4. Commit with: `fix(database): honour transactions in raw SQL operators and manual begin/commit/rollback`
   - Or split into multiple commits by task if preferred

---

### Task 8: Update spec and documentation

**Files**: `work/specs/FEAT-136.md`, `work/parsley/design/Database Implementation Status.md`, `docs/parsley/manual/features/database.md`
**Estimated effort**: Small

Steps:
1. Update `FEAT-136.md` status from `draft` to `implemented`
2. Check all acceptance criteria boxes
3. Update `Database Implementation Status.md` to note:
   - Transaction support is now consistent across all operators
   - Manual `begin()`/`commit()`/`rollback()` are real
   - `<=??=>` returns `Table` in both forms
4. If the database manual page mentions `begin()`/`commit()`/`rollback()`, verify the documentation is accurate (it should now be correct since these methods actually work)
5. Update this plan's status to `done`
6. Update `work/ID_COUNTER.md` (already done at plan creation)

---

## Task Order & Commits

```
Task 1 (helpers) ──┐
                    ├──► Task 3 (transaction tests) ──► Task 7 (validate & commit)
Task 2 (begin/commit) ─┘

Task 4 (return type) ─────────────────────────────────► Task 7

Task 5 (rows.Err) ────────────────────────────────────► Task 7

Task 6 (block comment) ───────────────────────────────► Task 7

Task 8 (docs) ── after Task 7
```

**Suggested commit sequence:**
1. `fix(database): add transaction-aware connQuery/connExec/connQueryRow helpers` (Tasks 1+2)
2. `test(database): add transaction integration tests` (Task 3)
3. `fix(database): harmonize <=??=> infix return type to Table` (Task 4)
4. `fix(database): add rows.Err() check to single-row query functions` (Task 5)
5. `fix(sql): fix block comment EOF edge case in scanner/rewriter` (Task 6)
6. `docs(database): update FEAT-136 spec and implementation status` (Task 8)

Tasks 4, 5, and 6 are independent and can be done in any order or in parallel.

## Validation Checklist

- [ ] All tests pass: `go test ./pkg/parsley/...`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] Transaction integration tests pass (new)
- [ ] `<=??=>` return type tests pass (new)
- [ ] Block comment unit tests pass (new)
- [ ] No regressions in existing database tests
- [ ] `FEAT-136.md` status updated to `implemented`
- [ ] `Database Implementation Status.md` updated

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Issue 3 (Table return type) breaks user code | Low — statement form already returns Table; infix form is less commonly used | Table supports index access, so `result[0]` still works |
| Transaction helpers miss a call site | Medium — queries would silently bypass Tx | Grep for `conn.DB.Query`, `conn.DB.Exec`, `conn.DB.QueryRow` after changes to verify zero remaining direct calls |
| `begin()` error handling differs from `@transaction` | Low — cosmetic | Reuse same error codes (DB-0014, DB-0015) for consistency |

## Deferred Items

Items to add to `work/BACKLOG.md` after implementation:
- Consider refactoring `TableBinding.query()`/`.exec()` to use the new `connQuery`/`connExec` helpers (reduces code duplication, but not urgent — the inline logic is correct)
- Consider registering DB-0013/DB-0014/DB-0015 in the error catalog (currently hardcoded in `stdlib_dsl_query.go`) for consistency

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-02-28 | Task 1: connQuery/connExec helpers | ✅ Complete | commit `d552765` |
| 2026-02-28 | Task 2: Real begin/commit/rollback | ✅ Complete | commit `d552765` — DB-0019 added; DB-0013/14/15 registered in catalog |
| 2026-02-28 | Task 3: Transaction integration tests | ✅ Complete | commit `d1f2a65` — 8 subtests in TestDatabaseTransactionIntegration |
| 2026-02-28 | Task 4: Harmonize return type | ✅ Complete | commit `d1f2a65` — evalDatabaseQueryMany now returns Table |
| 2026-02-28 | Task 5: rows.Err() check | ✅ Complete | commit `d552765` — added to evalQueryOneStatement and evalDatabaseQueryOne |
| 2026-02-28 | Task 6: Block comment EOF fix | ✅ Complete | commit `1d01662` — 7 new unit tests |
| 2026-02-28 | Task 7: Full validation | ✅ Complete | All tests pass; no new lint issues |
| 2026-02-28 | Task 8: Docs & status | ✅ Complete | FEAT-136 marked implemented; PLAN-116 marked done |