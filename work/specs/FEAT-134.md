---
id: FEAT-134
title: "Query DSL Cross-Database Compatibility (Postgres & MySQL)"
status: draft
priority: high
created: 2026-02-28
author: "@human"
---

# FEAT-134: Query DSL Cross-Database Compatibility

## Summary

The Query DSL generates PostgreSQL-flavored SQL exclusively. This works for SQLite (which accepts `$1` placeholders) and mostly works for PostgreSQL, but has a live bug in soft delete and a misleading `lastId` return. MySQL is completely broken — every parameterized query fails at runtime.

This spec covers three phases:
- **Phase 1:** Fix the Postgres-breaking bugs (soft delete timestamp, `lastId` behavior)
- **Phase 2:** Harden Postgres support with tests and boolean normalization
- **Phase 3:** Full MySQL support (placeholders, `RETURNING` alternatives, upsert syntax, `CHAR_LENGTH`)

Phases 1–2 are required for 1.0. Phase 3 can ship in 1.0 or be deferred to 1.1 depending on schedule.

## User Story

As a developer using Basil with PostgreSQL, I want the Query DSL to produce correct SQL for my database so that soft deletes, upserts, and raw SQL execute correctly without silent failures.

As a developer using Basil with MySQL, I want the Query DSL to produce correct SQL for my database so that I can use `@query`, `@insert`, `@update`, and `@delete` the same way I would with SQLite or Postgres.

## Audit Source

All findings from `work/reports/QUERY-DSL-CROSS-DB-AUDIT.md` (2025-07-27).

---

## Phase 1: PostgreSQL Fixes (Required for 1.0)

### P1-1. Soft delete generates SQLite-specific `datetime('now')`

**Location:** `pkg/parsley/evaluator/stdlib_dsl_query.go`, `buildDeleteSQL` ~L2904

**Bug:** Soft delete path emits:
```sql
UPDATE users SET deleted_at = datetime('now')
```

`datetime('now')` is a SQLite function. It fails on PostgreSQL and MySQL.

**Fix:** Add a driver-aware current-timestamp helper and use it in `buildDeleteSQL`. The `binding.DB.Driver` is available at the call site.

```go
func currentTimestampSQL(driver string) string {
    if driver == "sqlite" {
        return "datetime('now')"
    }
    return "CURRENT_TIMESTAMP"
}
```

`CURRENT_TIMESTAMP` is standard SQL and works on both PostgreSQL and MySQL. SQLite also supports it, but `datetime('now')` is the existing idiomatic form so we keep it for SQLite.

**Acceptance criteria:**
- [ ] `buildDeleteSQL` uses `currentTimestampSQL(binding.DB.Driver)` instead of hardcoded `datetime('now')`
- [ ] Unit test: `buildDeleteSQL` with `Driver: "postgres"` produces SQL containing `CURRENT_TIMESTAMP`
- [ ] Unit test: `buildDeleteSQL` with `Driver: "sqlite"` still produces SQL containing `datetime('now')`

---

### P1-2. `LastInsertId()` silently returns 0 on PostgreSQL

**Location:** `pkg/parsley/evaluator/eval_database.go`, `evalExecuteStatement` ~L183 and `evalDatabaseExecute` ~L332

**Bug:** The raw SQL execute operators (`<=!=>`) call:
```go
affected, _ := result.RowsAffected()
lastId, _ := result.LastInsertId()
```

`lib/pq` does not support `LastInsertId()` — it always returns an error, which is silently discarded. The result dict contains `{affected: N, lastId: 0}`, misleading Postgres users into thinking no ID was generated.

**Fix:** Check the error from `LastInsertId()`. If it fails, set `lastId` to `null` in the result dictionary instead of `0`.

```go
affected, _ := result.RowsAffected()
lastIdVal, lastIdErr := result.LastInsertId()

// Build result dict
pairs := map[string]ast.Expression{
    "affected": intLiteralExpr(affected),
}
if lastIdErr == nil {
    pairs["lastId"] = intLiteralExpr(lastIdVal)
} else {
    pairs["lastId"] = nullExpr()
}
```

This applies to both `evalExecuteStatement` (statement form) and `evalDatabaseExecute` (infix expression form).

**Acceptance criteria:**
- [ ] `evalExecuteStatement`: when `LastInsertId()` returns an error, `lastId` is `null` (not `0`)
- [ ] `evalDatabaseExecute`: same fix applied
- [ ] Unit test: mock a `sql.Result` that errors on `LastInsertId()`, verify result dict has `lastId: null`
- [ ] Existing SQLite tests still pass (SQLite's `LastInsertId()` returns successfully)

---

## Phase 2: Postgres Hardening (Required for 1.0)

### P2-1. Thread `driver` into `buildDeleteSQL`

**Context:** `buildDeleteSQL` currently receives `(node, binding, env)`. It needs `binding.DB.Driver` for the timestamp fix (P1-1). This is already available via `binding.DB.Driver` — no signature change needed, just use it internally.

However, future phases (Phase 3) will need `driver` threaded into `buildConditionSQL`, `buildInsertSQL`, `buildUpdateSQL`, `buildInClause`, etc. Phase 2 adds the `driver` parameter to the core SQL-building functions proactively, even though Phases 1–2 only need it in `buildDeleteSQL`. This avoids a second pass of signature changes in Phase 3.

**Functions to update:**
- `buildSelectSQL(node, binding, env)` — add `driver` (from `binding.DB.Driver`)
- `buildInsertSQL(node, binding, env)` — add `driver`
- `buildInsertSQLForBatch(node, binding, env)` — add `driver`
- `buildUpdateSQL(node, binding, env)` — add `driver`
- `buildDeleteSQL(node, binding, env)` — add `driver`
- `buildConditionSQL(cond, env, paramIdx)` — add `driver`
- `buildConditionSQLWithCTEs(cond, env, paramIdx, cteNames)` — add `driver`
- `buildInClause(column, value, paramIdx)` — add `driver`
- `buildCTESQL(cte, env, paramIdx, cteNames)` — add `driver`
- `buildCorrelatedSubquerySQL(subquery, outerTableName, env, paramIdx)` — add `driver`
- `buildCorrelatedConditionSQL(cond, outerTableName, env, paramIdx)` — add `driver`
- `buildCorrelatedConditionWhereClause(cond, cf, tableName, env, paramIdx)` — add `driver`
- `buildJoinSubquerySQL(cf, outerTableAlias, env, paramIdx)` — add `driver`
- `buildJoinConditionSQL(cond, outerAlias, joinAlias, env, paramIdx)` — add `driver`
- `buildSubqueryCondition(column, operator, subquery, env, paramIdx)` — add `driver`
- `buildComputedFieldSQL(cf, outerTableName, env, paramIdx)` — add `driver`
- `loadHasManyRelation` — access driver from `parentBinding.DB.Driver`
- `loadBelongsToRelation` — access driver from binding

For Phase 2, all these functions receive `driver` but only `buildDeleteSQL` changes behavior. The parameter is passed through everywhere else unchanged. Phase 3 will activate the driver-aware behavior in the remaining functions.

**Acceptance criteria:**
- [ ] All SQL-building functions accept a `driver string` parameter
- [ ] All call sites pass `binding.DB.Driver` (or propagate from caller)
- [ ] All existing tests pass with no behavior change (SQLite driver)
- [ ] No `driver` parameter is left unused at the function level (it should at minimum be passed to child functions)

---

### P2-2. Add Postgres integration test coverage

**Context:** All existing DSL query tests use `@sqlite(":memory:")`. There is zero test coverage for PostgreSQL. While we can't run a live Postgres instance in unit tests, we can test the SQL generation layer by calling `buildSelectSQL`, `buildInsertSQL`, etc. directly with `Driver: "postgres"` and asserting the generated SQL strings.

**Approach:** Add a new test file `pkg/parsley/tests/dsl_query_postgres_test.go` (or a section in existing `dsl_query_test.go`) that:
1. Creates a `TableBinding` with `DB: &DBConnection{Driver: "postgres"}` and a DSL schema
2. Calls the SQL-building functions directly
3. Asserts the generated SQL is valid PostgreSQL (contains `$1` placeholders, `CURRENT_TIMESTAMP` for soft delete, etc.)

This does NOT require a running Postgres instance — it's SQL string assertion.

**Test cases:**
- SELECT with conditions → `$1, $2` placeholders
- INSERT with values → `$1, $2` placeholders
- INSERT with RETURNING → contains `RETURNING *`
- UPDATE with conditions → `$1, $2` placeholders
- UPDATE with RETURNING → contains `RETURNING *`
- DELETE (hard) with conditions → `$1, $2` placeholders
- DELETE (soft) → contains `CURRENT_TIMESTAMP` (not `datetime('now')`)
- Upsert → contains `ON CONFLICT ... EXCLUDED`
- IN clause → `$1, $2, $3` placeholders
- BETWEEN → `$1` and `$2` placeholders

**Acceptance criteria:**
- [ ] At least 10 SQL-generation tests asserting Postgres-correct SQL
- [ ] Soft delete test confirms `CURRENT_TIMESTAMP` for Postgres driver
- [ ] All tests run without a Postgres instance (pure SQL string tests)

---

## Phase 3: MySQL Support (1.0 or 1.1 — TBD)

### P3-1. `$1` placeholders → `?` for MySQL

**Location:** Every `fmt.Sprintf("$%d", *paramIdx)` call in `stdlib_dsl_query.go` (~20 occurrences) and `loadHasManyRelation`/`loadBelongsToRelation`.

**Problem:** MySQL uses positional `?` placeholders. `go-sql-driver/mysql` does not translate `$N` to `?`. Every parameterized DSL query crashes on MySQL.

**Fix:** Add a placeholder helper:

```go
func sqlPlaceholder(driver string, idx int) string {
    if driver == "mysql" {
        return "?"
    }
    return fmt.Sprintf("$%d", idx)
}
```

Replace all `fmt.Sprintf("$%d", *paramIdx)` calls with `sqlPlaceholder(driver, *paramIdx)`.

The `paramIdx` counter is still incremented for MySQL — it tracks the parameter's position in the `params` slice. Only the emitted SQL text changes.

**Acceptance criteria:**
- [ ] All `fmt.Sprintf("$%d", ...)` calls in `stdlib_dsl_query.go` replaced with `sqlPlaceholder()`
- [ ] `loadHasManyRelation` and `loadBelongsToRelation` use `sqlPlaceholder()`
- [ ] SQL-generation tests for MySQL driver confirm `?` placeholders throughout
- [ ] Existing SQLite and Postgres tests unchanged

---

### P3-2. `RETURNING` clause — error on MySQL

**Location:** `buildInsertSQL`, `buildInsertSQLForBatch`, `buildUpdateSQL`, `buildDeleteSQL`

**Problem:** MySQL has no `RETURNING` clause. `INSERT ... RETURNING *`, `UPDATE ... RETURNING *`, and `DELETE ... RETURNING *` all fail.

**Fix:** When `driver == "mysql"` and the terminal requests returning data (`Type == "one"` or `Type == "many"`), return a clear error instead of generating invalid SQL:

```go
if driver == "mysql" && node.Terminal != nil &&
    (node.Terminal.Type == "one" || node.Terminal.Type == "many") {
    return "", nil, &Error{
        Message: "@insert ?-> / ??-> (RETURNING) is not supported on MySQL. Use @insert . and query separately.",
        Class:   ClassDatabase,
        Code:    "DB-0018",
    }
}
```

Same pattern for `@update` and `@delete`.

**Design decision:** Error rather than emulation. Emulating `RETURNING` (INSERT + SELECT, or pre-SELECT + mutation in a transaction) adds complexity, implicit transactions, and edge cases (concurrent modifications, non-auto-increment PKs). A clear error with a workaround hint is more honest and maintainable. Emulation can be added later if user demand warrants it.

**Acceptance criteria:**
- [ ] `buildInsertSQL` returns error for MySQL + returning terminal
- [ ] `buildInsertSQLForBatch` returns error for MySQL + returning terminal
- [ ] `buildUpdateSQL` returns error for MySQL + returning terminal
- [ ] `buildDeleteSQL` returns error for MySQL + returning terminal
- [ ] Error message includes actionable workaround
- [ ] Non-returning terminals (`.`, `.-> count`) continue to work on MySQL
- [ ] New error code `DB-0018` registered in error catalog

---

### P3-3. Upsert syntax for MySQL

**Location:** `buildInsertSQL` (~L2362), `buildInsertSQLForBatch` (~L2550)

**Problem:** DSL generates PostgreSQL/SQLite upsert syntax:
```sql
INSERT INTO users (...) VALUES (...)
ON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name, email = EXCLUDED.email
```

MySQL uses:
```sql
INSERT INTO users (...) VALUES (...)
ON DUPLICATE KEY UPDATE name = VALUES(name), email = VALUES(email)
```

**Fix:** Driver-aware upsert generation:

```go
if len(node.UpsertKey) > 0 {
    if driver == "mysql" {
        sql.WriteString(" ON DUPLICATE KEY UPDATE ")
        var updates []string
        for _, col := range columns {
            updates = append(updates, fmt.Sprintf("%s = VALUES(%s)", col, col))
        }
        sql.WriteString(strings.Join(updates, ", "))
    } else {
        sql.WriteString(" ON CONFLICT (")
        sql.WriteString(strings.Join(node.UpsertKey, ", "))
        sql.WriteString(") DO UPDATE SET ")
        var updates []string
        for _, col := range columns {
            updates = append(updates, fmt.Sprintf("%s = EXCLUDED.%s", col, col))
        }
        sql.WriteString(strings.Join(updates, ", "))
    }
}
```

**Note:** MySQL's `ON DUPLICATE KEY` uses the table's unique/primary key implicitly — the `UpsertKey` fields are not specified in the SQL (MySQL infers from the table definition). The DSL's `upsertKey` is still needed to know which columns to exclude from the UPDATE set in more sophisticated scenarios, but for the basic `col = VALUES(col)` pattern, the key fields don't appear in the MySQL SQL.

**Acceptance criteria:**
- [ ] MySQL driver generates `ON DUPLICATE KEY UPDATE col = VALUES(col)`
- [ ] PostgreSQL/SQLite driver unchanged (`ON CONFLICT ... EXCLUDED`)
- [ ] Both `buildInsertSQL` and `buildInsertSQLForBatch` updated
- [ ] SQL-generation test for MySQL upsert

---

### P3-4. `length()` → `CHAR_LENGTH()` for MySQL

**Location:** `pkg/parsley/evaluator/stdlib_dsl_schema.go`, `buildCreateTableSQL` ~L709

**Problem:** CHECK constraints use `length(col)` which returns **byte length** on MySQL, not character length. For UTF-8 text, `LENGTH('café')` = 5 but `CHAR_LENGTH('café')` = 4. String length constraints would be wrong for non-ASCII data.

**Fix:** Driver-aware length function:

```go
func sqlLengthFunc(driver string) string {
    if driver == "mysql" {
        return "CHAR_LENGTH"
    }
    return "length"
}
```

Use in CHECK constraint generation:

```go
lenFn := sqlLengthFunc(driver)
checks = append(checks, fmt.Sprintf("%s(%s) >= %d AND %s(%s) <= %d",
    lenFn, name, *field.MinLength, lenFn, name, *field.MaxLength))
```

**Acceptance criteria:**
- [ ] MySQL DDL uses `CHAR_LENGTH()` in CHECK constraints
- [ ] SQLite/Postgres DDL unchanged (`length()`)
- [ ] SQL-generation test for MySQL CHECK constraint with string length

---

### P3-5. MySQL SQL-generation test suite

**Approach:** Same pattern as P2-2, but with `Driver: "mysql"`. Create or extend a test file that calls SQL-building functions with a MySQL-driver `TableBinding` and asserts the generated SQL.

**Test cases:**
- SELECT with conditions → `?` placeholders
- INSERT with values → `?` placeholders
- INSERT with returning terminal → returns error `DB-0018`
- INSERT (non-returning) → works, `?` placeholders
- UPDATE with conditions → `?` placeholders
- UPDATE with returning terminal → returns error `DB-0018`
- DELETE (hard) with conditions → `?` placeholders
- DELETE (soft) → contains `CURRENT_TIMESTAMP` (not `datetime('now')`)
- DELETE with returning terminal → returns error `DB-0018`
- Upsert → `ON DUPLICATE KEY UPDATE ... VALUES(...)`
- IN clause → `?, ?, ?` placeholders
- BETWEEN → `?` and `?` placeholders
- CREATE TABLE with string length → `CHAR_LENGTH`

**Acceptance criteria:**
- [ ] At least 13 SQL-generation tests asserting MySQL-correct SQL
- [ ] All placeholder tests confirm `?` not `$N`
- [ ] RETURNING terminal tests confirm error with helpful message
- [ ] All tests run without a MySQL instance (pure SQL string tests)

---

## Design Decisions

- **`CURRENT_TIMESTAMP` over `NOW()`:** Both work on Postgres and MySQL. `CURRENT_TIMESTAMP` is standard SQL (SQL-92). Preferred for portability. SQLite keeps `datetime('now')` since it's the existing idiomatic form.
- **Error over emulation for MySQL `RETURNING`:** Emulating `RETURNING` (INSERT + SELECT, or SELECT + mutation in transaction) introduces implicit behavior, concurrency edge cases, and non-obvious transaction semantics. A clear error with a workaround is more honest. Can revisit if users request emulation.
- **Thread `driver` in Phase 2, activate in Phase 3:** Avoids two rounds of signature changes. Phase 2 is a mechanical refactor that enables Phase 3 to be a series of small, focused behavior changes.
- **`lastId: null` over omitting the key:** Returning `null` rather than omitting `lastId` from the result dict means user code that accesses `result.lastId` gets `null` instead of an undefined-variable error. Less surprising.
- **SQL string tests over integration tests:** Testing SQL generation by asserting output strings doesn't require running database instances. This gives fast, reliable CI coverage. Live integration tests with embedded Postgres/MySQL are deferred (see backlog #113).

---

## Technical Context

### Affected Components
- `pkg/parsley/evaluator/stdlib_dsl_query.go` — All SQL-building functions (placeholder, RETURNING, upsert, soft delete timestamp)
- `pkg/parsley/evaluator/eval_database.go` — `LastInsertId()` handling in raw SQL operators
- `pkg/parsley/evaluator/stdlib_dsl_schema.go` — `buildCreateTableSQL` CHECK constraints (`length` vs `CHAR_LENGTH`)
- `pkg/parsley/tests/dsl_query_test.go` — New Postgres/MySQL SQL-generation tests

### Dependencies
- Depends on: FEAT-133 (database connection management) — already complete
- Blocks: 1.0 release sign-off for database support

### Edge Cases & Constraints
1. **MySQL `ON DUPLICATE KEY` doesn't take explicit conflict columns** — MySQL infers from unique indexes. The DSL's `upsertKey` is not emitted in the MySQL SQL but is still parsed and validated.
2. **MySQL 5.7 does not support CTEs** — CTEs require MySQL 8.0+. This is acceptable; MySQL 5.7 is EOL (October 2023).
3. **SQLite `RETURNING` requires 3.35.0+** — The DSL uses `RETURNING` unconditionally for SQLite. The old-style methods deliberately avoided it. If users hit old SQLite versions, this is a separate issue (existing behavior, not introduced by this spec).
4. **`lib/pq` `LastInsertId()` error is driver-specific** — The fix checks the error return generically, so it works regardless of which driver fails. No driver-name check needed.

## Related
- Audit: `work/reports/QUERY-DSL-CROSS-DB-AUDIT.md`
- FEAT-133: Database connection management (pool lifecycle, `db.Ping()` removal)
- Backlog #113: Embedded Postgres for integration tests (deferred)
- Plan: TBD — to be created when implementation begins