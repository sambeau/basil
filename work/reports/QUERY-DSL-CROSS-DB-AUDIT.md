# Query DSL Cross-Database Audit

**Date:** 2025-07-27
**Scope:** Audit of `stdlib_dsl_query.go`, `eval_database.go`, `stdlib_schema_table_binding.go`, `stdlib_dsl_schema.go`, `sql_security.go`
**Purpose:** Determine whether the Query DSL "just works" with PostgreSQL and MySQL, or whether there are gaps blocking 1.0 sign-off
**Triggered by:** Database coupling review (FEAT-133 follow-up)

---

## Executive Summary

The Query DSL generates **PostgreSQL-flavored SQL exclusively**. It works correctly with SQLite (which accepts Postgres-style `$1` placeholders) and PostgreSQL, but **will crash at runtime on MySQL** for every parameterized query. There are also Postgres-breaking issues in the soft delete path.

| Database   | DSL Status         | Notes                                      |
|------------|--------------------|--------------------------------------------|
| SQLite     | ✅ Production-ready | All tests pass, primary development target |
| PostgreSQL | ⚠️ Mostly works    | 1 critical bug (soft delete), 1 medium     |
| MySQL      | ❌ Broken           | 4 critical issues, all parameterized queries fail |

**Recommendation:** Fix the 2 Postgres issues, ship 1.0 with SQLite + Postgres support, document MySQL as experimental, and add full MySQL support in a follow-up release.

---

## Methodology

1. Traced every SQL-generating code path in the Query DSL (`@query`, `@insert`, `@update`, `@delete`, `@transaction`)
2. Traced the old-style `TableBinding` method API (`find()`, `where()`, `all()`, `insert()`, `update()`, `delete()`, `save()`)
3. Traced `buildCreateTableSQL` and `schemaTypeToSQL` for DDL generation
4. Checked parameter placeholder syntax against each driver's requirements
5. Checked SQL dialect features (`RETURNING`, `ON CONFLICT`, `datetime()`, `length()`) against each database
6. Reviewed `objectToGoValue` and `rowToDict` for type conversion correctness
7. Reviewed connection establishment in `connectionBuiltins` (`@postgres()`, `@mysql()`)

---

## Critical Issues (🔴 Will Break at Runtime)

### C1. `$1`-style placeholders are incompatible with MySQL

**Location:** Every `buildConditionSQL`, `buildInsertSQL`, `buildUpdateSQL`, `buildDeleteSQL`, `buildInClause`, `buildCorrelatedConditionSQL`, `loadHasManyRelation`, `loadBelongsToRelation`

**Pattern:**
```go
// stdlib_dsl_query.go — appears ~20 times throughout the file
placeholder := fmt.Sprintf("$%d", *paramIdx)
*paramIdx++
```

**Impact:**
- **PostgreSQL:** ✅ `$1, $2, $3` is native syntax
- **MySQL:** ❌ MySQL uses `?` positional placeholders. `go-sql-driver/mysql` does **not** translate `$N` → `?`. Every parameterized DSL query will fail with a SQL syntax error.
- **SQLite:** ✅ `modernc.org/sqlite` accepts `$1` syntax

**Scope of breakage:** Every `@query` with conditions, every `@insert`, every `@update`, every `@delete` with conditions, every eager-loaded relation (`with`), every batch insert, every upsert, every BETWEEN, every IN clause, every correlated subquery, every CTE with conditions.

**Fix:** Thread `binding.DB.Driver` into all SQL-building functions and use a helper:
```go
func placeholder(driver string, idx int) string {
    if driver == "mysql" {
        return "?"
    }
    return fmt.Sprintf("$%d", idx)
}
```

For MySQL, the `paramIdx` counter is still needed for internal bookkeeping (to keep params in order) but the emitted placeholder is always `?`.

**Estimated effort:** 2–3 hours. Mechanical change — add `driver string` parameter to ~15 functions, replace all `fmt.Sprintf("$%d", *paramIdx)` calls.

---

### C2. `RETURNING` clause is unsupported on MySQL

**Location:** `buildInsertSQL` (L2382–2388), `buildInsertSQLForBatch` (L2569–2575), `buildUpdateSQL` (L2724–2728), `buildDeleteSQL` (L2951–2956)

**Pattern:**
```go
// buildInsertSQL, buildUpdateSQL, buildDeleteSQL — all identical pattern
if node.Terminal != nil && (node.Terminal.Type == "one" || node.Terminal.Type == "many") {
    sql.WriteString(" RETURNING ")
    if len(node.Terminal.Projection) == 0 || node.Terminal.Projection[0] == "*" {
        sql.WriteString("*")
    } else {
        sql.WriteString(strings.Join(node.Terminal.Projection, ", "))
    }
}
```

**Impact:**
- **PostgreSQL:** ✅ `RETURNING` fully supported
- **MySQL:** ❌ No `RETURNING` clause exists. Query will fail at runtime.
- **SQLite:** ✅ Supported since 3.35.0 (2021-03-12)

**Affected DSL terminals:**
- `@insert(Users) | ... ?-> *` (insert returning one row)
- `@update(Users) | ... ?-> *` / `??-> *` (update returning rows)
- `@delete(Users) | ... ?-> *` / `??-> *` (delete returning rows)

**Non-returning terminals are fine:** `@insert ... .`, `@update ... .`, `@delete ... .`, `.-> count` — these use `Exec()` not `Query()`.

**Fix options for MySQL:**
1. **INSERT returning:** Execute INSERT, then `SELECT ... WHERE id = LAST_INSERT_ID()` (only works for single-row auto-increment inserts)
2. **UPDATE/DELETE returning:** Wrap in transaction: SELECT matching rows first, then execute mutation, return the pre-selected data
3. **Simplest:** Return a clear error when `RETURNING` terminals are used with MySQL driver, explaining the limitation

**Estimated effort:** Option 3 (error): 30 minutes. Options 1–2 (emulation): 4–6 hours with edge cases.

---

### C3. `ON CONFLICT ... DO UPDATE SET col = EXCLUDED.col` is MySQL-incompatible

**Location:** `buildInsertSQL` (L2362–2377), `buildInsertSQLForBatch` (L2550–2561)

**Pattern:**
```go
if len(node.UpsertKey) > 0 {
    sql.WriteString(" ON CONFLICT (")
    sql.WriteString(strings.Join(node.UpsertKey, ", "))
    sql.WriteString(") DO UPDATE SET ")
    var updates []string
    for _, col := range columns {
        updates = append(updates, fmt.Sprintf("%s = EXCLUDED.%s", col, col))
    }
    sql.WriteString(strings.Join(updates, ", "))
}
```

**Impact:**
- **PostgreSQL:** ✅ Native syntax
- **MySQL:** ❌ MySQL uses `ON DUPLICATE KEY UPDATE col = VALUES(col)` (or `col = NEW.col` in MySQL 8.0.19+). The `ON CONFLICT` / `EXCLUDED` syntax will fail.
- **SQLite:** ✅ SQLite adopted the PostgreSQL syntax

**Fix:** Driver-aware upsert generation:
```go
if driver == "mysql" {
    sql.WriteString(" ON DUPLICATE KEY UPDATE ")
    for _, col := range columns {
        updates = append(updates, fmt.Sprintf("%s = VALUES(%s)", col, col))
    }
} else {
    sql.WriteString(" ON CONFLICT (")
    sql.WriteString(strings.Join(node.UpsertKey, ", "))
    sql.WriteString(") DO UPDATE SET ")
    for _, col := range columns {
        updates = append(updates, fmt.Sprintf("%s = EXCLUDED.%s", col, col))
    }
}
```

**Estimated effort:** 1 hour.

---

### C4. Soft delete uses SQLite-specific `datetime('now')`

**Location:** `buildDeleteSQL` (L2899–2906)

**Pattern:**
```go
if binding.SoftDeleteColumn != "" {
    sql.WriteString("UPDATE ")
    sql.WriteString(binding.TableName)
    sql.WriteString(" SET ")
    sql.WriteString(binding.SoftDeleteColumn)
    sql.WriteString(" = datetime('now')")
}
```

**Impact:**
- **PostgreSQL:** ❌ `datetime('now')` is not valid SQL. Will fail with syntax error. Use `NOW()` or `CURRENT_TIMESTAMP`.
- **MySQL:** ❌ Same — `datetime()` is not a MySQL function. Use `NOW()` or `CURRENT_TIMESTAMP`.
- **SQLite:** ✅ `datetime('now')` is the correct SQLite function

**Fix:** Driver-aware timestamp:
```go
func currentTimestampSQL(driver string) string {
    switch driver {
    case "sqlite":
        return "datetime('now')"
    default: // postgres, mysql
        return "CURRENT_TIMESTAMP"
    }
}
```

`CURRENT_TIMESTAMP` is actually standard SQL and works on all three databases, but SQLite's `datetime('now')` is idiomatic and already in use. Either approach works.

**Estimated effort:** 15 minutes.

**Note:** This also breaks PostgreSQL, not just MySQL. This is a live bug for any Postgres user with soft deletes.

---

## Medium Issues (🟡 Wrong Behavior / Edge Cases)

### M1. `LastInsertId()` silently returns 0 for PostgreSQL

**Location:** `evalExecuteStatement` in `eval_database.go` (L183–184), `evalDatabaseExecute` (L332–333)

**Pattern:**
```go
affected, _ := result.RowsAffected()
lastId, _ := result.LastInsertId()
```

**Impact:** `lib/pq` does not support `LastInsertId()` — it always returns an error (which is discarded by `_`), so `lastId` is always `0`. Users of raw SQL `<=!=>` will get `{affected: N, lastId: 0}` and may incorrectly think no ID was generated.

The DSL-level `.lastInsertId()` method on DBConnection already guards:
```go
if conn.Driver != "sqlite" {
    return newDatabaseErrorWithDriver("DB-0001", conn.Driver, ...)
}
```

But the raw SQL execute operator returns `lastId: 0` silently.

**Fix options:**
1. Return `lastId: null` when `LastInsertId()` returns an error
2. Only include `lastId` in the result dict when the driver supports it
3. Document the behavior

**Estimated effort:** 15 minutes for option 1.

---

### M2. `length()` in CHECK constraints returns byte length on MySQL

**Location:** `buildCreateTableSQL` in `stdlib_dsl_schema.go` (L709–715)

**Pattern:**
```go
checks = append(checks, fmt.Sprintf("length(%s) >= %d AND length(%s) <= %d",
    name, *field.MinLength, name, *field.MaxLength))
```

**Impact:**
- **PostgreSQL:** ✅ `length()` returns character count
- **MySQL:** ⚠️ `LENGTH()` returns **byte** count. For multi-byte UTF-8 characters, `LENGTH('café')` = 5 but `CHAR_LENGTH('café')` = 4. Constraints would be incorrect for non-ASCII data.
- **SQLite:** ✅ `length()` returns character count

**Fix:** Use `CHAR_LENGTH()` for MySQL (which also works on PostgreSQL but not SQLite).

```go
func lengthFunc(driver string) string {
    if driver == "mysql" {
        return "CHAR_LENGTH"
    }
    return "length"
}
```

**Estimated effort:** 15 minutes.

---

### M3. Boolean scanning inconsistency across drivers

**Location:** `rowToDict` in `eval_conversions.go`

Postgres returns `bool` for `BOOLEAN` columns, SQLite returns `int64` (0/1), MySQL returns `[]byte` or `int64` depending on the column type. The `rowToDict` function scans values as `any` and then converts them. This may produce inconsistent Parsley types for the same logical data depending on the backend.

**Impact:** Low — likely works in practice because Go's `database/sql` scanner normalizes most types, but untested across backends.

**Fix:** Add explicit boolean normalization in `rowToDict` for `bool`, `int64(0)`, `int64(1)` → `Boolean{Value: true/false}`.

**Estimated effort:** 30 minutes + tests.

---

## Items That Are Fine (🟢)

### DDL Generation (`buildCreateTableSQL`)
Properly driver-aware with branching for:
- `INTEGER PRIMARY KEY` vs `SERIAL PRIMARY KEY` vs `INT AUTO_INCREMENT PRIMARY KEY`
- `BIGSERIAL` vs `BIGINT AUTO_INCREMENT`
- `UUID ... DEFAULT gen_random_uuid()` for Postgres
- `BOOLEAN` vs `INTEGER` (SQLite bools)
- `TIMESTAMP` vs `TEXT` (SQLite datetimes)
- `JSONB` vs `TEXT`

### SQL Injection Protection (`sql_security.go`)
Identifier validation is database-agnostic. `isValidSQLIdentifier` uses a conservative `[a-zA-Z_][a-zA-Z0-9_]*` regex that works for all three databases.

### Transaction Handling (`evalTransactionExpression`)
Uses `database/sql`'s `Begin()`/`Commit()`/`Rollback()` which is driver-agnostic.

### Standard SQL Clauses
`SELECT`, `FROM`, `WHERE`, `ORDER BY`, `GROUP BY`, `HAVING`, `LIMIT`, `OFFSET`, `JOIN`, `LEFT JOIN`, `IN`, `NOT IN`, `BETWEEN`, `LIKE`, `IS NULL`, `IS NOT NULL` — all standard SQL, all work.

### CTEs (`WITH` Clauses)
Supported by PostgreSQL (all versions), MySQL 8.0+, SQLite 3.8.3+.

### Connection Pooling (FEAT-133)
`ConnMaxIdleTime` and `ConnMaxLifetime` are set for both Postgres and MySQL. Pool lifecycle is properly managed.

### Old-Style Methods
Correctly gated to SQLite-only via `ensureSQLite()`. Not a bug — they're a legacy API that predates multi-database support.

---

## Architecture Assessment

The DSL's SQL generation architecture is fundamentally sound:
- Clean separation between SQL building (`buildSelectSQL`, `buildInsertSQL`, etc.) and execution (`executeQueryMany`, `executeInsert`, etc.)
- `TableBinding` has access to `DB.Driver` on every path
- The `paramIdx` counter pattern works well — it just needs to emit `?` instead of `$N` for MySQL
- All SQL builders already accept `*int` paramIdx, making it easy to add a `driver string` parameter

The issues are **mechanical, not architectural**. Every fix is a matter of adding driver-awareness to functions that already have access to the driver string via `binding.DB.Driver`. No redesign is needed.

---

## Recommendation

### For 1.0: SQLite + PostgreSQL (Fix 2 Issues)

| Fix | Issue | Effort | Priority |
|-----|-------|--------|----------|
| C4  | `datetime('now')` → driver-aware timestamp | 15 min | Must-fix (breaks Postgres soft delete) |
| M1  | `LastInsertId()` → return null on error | 15 min | Should-fix (misleading for Postgres) |

With these two fixes, PostgreSQL works fully. SQLite continues to work. MySQL remains broken but is already not documented as supported by the DSL.

### For 1.1: Full MySQL Support (Fix Remaining Issues)

| Fix | Issue | Effort |
|-----|-------|--------|
| C1  | `$1` → `?` placeholders for MySQL | 2–3 hours |
| C2  | `RETURNING` emulation or error for MySQL | 30 min (error) or 4–6 hours (emulation) |
| C3  | `ON CONFLICT` → `ON DUPLICATE KEY` for MySQL | 1 hour |
| M2  | `length()` → `CHAR_LENGTH()` for MySQL | 15 min |
| M3  | Boolean scanning normalization | 30 min |

**Total for MySQL support:** ~5 hours (with error on RETURNING) or ~9 hours (with RETURNING emulation)

### Documentation Actions

1. Add a "Database Support" section to the Basil docs stating:
   - SQLite: Full support (production-ready)
   - PostgreSQL: Full support (production-ready)
   - MySQL: Connection and raw SQL work; Query DSL support coming in a future release
2. Update the Query DSL documentation to note which backends are supported
3. Add backlog items for the MySQL fixes

---

## Appendix: Files Audited

| File | What It Does | Issues Found |
|------|-------------|--------------|
| `pkg/parsley/evaluator/stdlib_dsl_query.go` | Query DSL SQL generation (SELECT, INSERT, UPDATE, DELETE, transactions) | C1, C2, C3, C4 |
| `pkg/parsley/evaluator/eval_database.go` | Raw SQL operators (`<=?=>`, `<=??=>`, `<=!=>`) | M1 |
| `pkg/parsley/evaluator/stdlib_schema_table_binding.go` | Old-style TableBinding methods | Correctly SQLite-gated |
| `pkg/parsley/evaluator/stdlib_dsl_schema.go` | Schema DDL generation (`buildCreateTableSQL`) | M2; DDL otherwise good |
| `pkg/parsley/evaluator/sql_security.go` | SQL identifier validation | No issues |
| `pkg/parsley/evaluator/eval_conversions.go` | `rowToDict` type conversion | M3 |
| `pkg/parsley/evaluator/drivers.go` | Driver registration (`lib/pq`, `go-sql-driver/mysql`) | No issues |
| `pkg/parsley/evaluator/evaluator.go` | `@postgres()`, `@mysql()` connection builtins | No issues |
| `pkg/parsley/evaluator/eval_method_dispatch.go` | DBConnection methods | `lastInsertId` correctly guarded |
| `pkg/parsley/tests/dsl_query_test.go` | DSL tests (all use SQLite `:memory:`) | No Postgres/MySQL test coverage |