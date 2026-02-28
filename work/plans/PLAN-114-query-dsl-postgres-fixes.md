---
id: PLAN-114
feature: FEAT-134
title: "Query DSL Postgres Fixes & Driver Threading (Phases 1–2)"
status: complete
created: 2026-02-28
---

# Implementation Plan: FEAT-134 Phases 1–2

## Overview

Fix two live bugs that break PostgreSQL (soft delete timestamp, silent `LastInsertId`), then thread `driver string` through all SQL-building functions to enable Phase 3 (MySQL support) without a second refactor pass. Add SQL-generation tests for Postgres.

**Estimated total effort:** ~2 hours

## Prerequisites

- [x] FEAT-133 (database connection management) — merged
- [x] Audit report: `work/reports/QUERY-DSL-CROSS-DB-AUDIT.md`
- [x] Spec: `work/specs/FEAT-134.md`

## Call Graph Reference

The SQL-building functions form a call tree rooted at five top-level builders. Each receives `binding *TableBinding`, so `binding.DB.Driver` is always available at the entry point. The `driver` parameter must be threaded down through the entire tree.

```
evalQueryExpression
  └─ buildSelectSQL(node, binding, env)                          ← entry, has binding.DB.Driver
       ├─ buildCTESQL(cte, env, paramIdx, cteNames)
       │    └─ buildConditionNodeSQLWithCTEs → buildConditionSQLWithCTEs
       │         ├─ buildSubqueryCondition → buildConditionNodeSQL → ...
       │         └─ buildInClause
       ├─ buildComputedFieldSQL(cf, outerTable, env, paramIdx)
       │    └─ buildCorrelatedSubquerySQL
       │         └─ buildCorrelatedConditionSQL → buildCorrelatedCondition
       ├─ buildJoinSubquerySQL(cf, outerAlias, env, paramIdx)
       │    └─ buildJoinConditionSQL → buildJoinCondition
       ├─ buildCorrelatedConditionWhereClause(cond, cf, table, env, paramIdx)
       │    └─ buildCorrelatedSubquerySQL (same as above)
       ├─ buildConditionNodeSQLWithCTEs → buildConditionSQLWithCTEs
       │    ├─ buildSubqueryCondition → buildConditionNodeSQL
       │    │    ├─ buildConditionSQL
       │    │    │    ├─ buildSubqueryCondition (recursive)
       │    │    │    └─ buildInClause
       │    │    └─ buildConditionGroupSQL → buildConditionNodeSQL (recursive)
       │    └─ buildInClause
       └─ buildConditionGroupSQLWithCTEs → buildConditionNodeSQLWithCTEs (recursive)

evalInsertExpression
  └─ buildInsertSQL(node, binding, env)                          ← entry

evalBatchInsert
  └─ buildInsertSQLForBatch(node, binding, env)                  ← entry

evalUpdateExpression
  └─ buildUpdateSQL(node, binding, env)                          ← entry
       └─ buildConditionSQL(cond, env, paramIdx)
            ├─ buildSubqueryCondition
            └─ buildInClause

evalDeleteExpression
  └─ buildDeleteSQL(node, binding, env)                          ← entry
       └─ buildConditionSQL (same as above)

loadHasManyRelation  ← has parentBinding.DB.Driver
  └─ buildConditionNodeSQL (hardcoded $1 at L2071, then delegates)

loadBelongsToRelation  ← has binding via relatedBinding
  └─ hardcoded $1 at L2163
```

---

## Tasks

### Task 1: Add `currentTimestampSQL` helper and fix soft delete (P1-1)

**Files:** `pkg/parsley/evaluator/stdlib_dsl_query.go`
**Effort:** Small (15 min)
**Commit point:** Yes — after tests pass

This is the live Postgres bug. Fix it first so the rest of the plan is pure refactoring.

Steps:

1. Add a package-level helper near the top of `stdlib_dsl_query.go` (after imports):
   ```go
   // currentTimestampSQL returns the driver-appropriate SQL expression for "now".
   func currentTimestampSQL(driver string) string {
       if driver == "sqlite" {
           return "datetime('now')"
       }
       return "CURRENT_TIMESTAMP"
   }
   ```

2. In `buildDeleteSQL` (~L2904), replace:
   ```go
   sql.WriteString(" = datetime('now')")
   ```
   with:
   ```go
   sql.WriteString(" = ")
   sql.WriteString(currentTimestampSQL(binding.DB.Driver))
   ```

3. Run `go test ./pkg/parsley/...` — all existing tests should pass since they use SQLite driver.

Tests (added in Task 5):
- `buildDeleteSQL` with `Driver: "postgres"` → SQL contains `CURRENT_TIMESTAMP`
- `buildDeleteSQL` with `Driver: "sqlite"` → SQL contains `datetime('now')`

---

### Task 2: Fix `LastInsertId()` returning 0 on Postgres (P1-2)

**Files:** `pkg/parsley/evaluator/eval_database.go`
**Effort:** Small (15 min)
**Commit point:** Yes — after tests pass

Steps:

1. In `evalExecuteStatement` (~L183–195), replace:
   ```go
   affected, _ := result.RowsAffected()
   lastId, _ := result.LastInsertId()

   resultDict := &Dictionary{
       Pairs: map[string]ast.Expression{
           "affected": &ast.IntegerLiteral{
               Token: lexer.Token{Type: lexer.INT, Literal: strconv.FormatInt(affected, 10)},
               Value: affected,
           },
           "lastId": &ast.IntegerLiteral{
               Token: lexer.Token{Type: lexer.INT, Literal: strconv.FormatInt(lastId, 10)},
               Value: lastId,
           },
       },
       Env: env,
   }
   ```
   with:
   ```go
   affected, _ := result.RowsAffected()
   lastIdVal, lastIdErr := result.LastInsertId()

   pairs := map[string]ast.Expression{
       "affected": &ast.IntegerLiteral{
           Token: lexer.Token{Type: lexer.INT, Literal: strconv.FormatInt(affected, 10)},
           Value: affected,
       },
   }
   if lastIdErr == nil {
       pairs["lastId"] = &ast.IntegerLiteral{
           Token: lexer.Token{Type: lexer.INT, Literal: strconv.FormatInt(lastIdVal, 10)},
           Value: lastIdVal,
       }
   } else {
       pairs["lastId"] = &ast.Identifier{
           Token: lexer.Token{Type: lexer.IDENT, Literal: "null"},
           Value: "null",
       }
   }

   resultDict := &Dictionary{
       Pairs: pairs,
       Env:   env,
   }
   ```

2. Apply the identical pattern to `evalDatabaseExecute` (~L329–344), which is the infix expression version of the same operation.

3. Run `go test ./pkg/parsley/...` — existing SQLite tests pass because SQLite's `LastInsertId()` succeeds.

Tests (added in Task 5):
- Existing `<=!=>` tests should still pass (SQLite `LastInsertId` succeeds → integer)
- New test with a mock/stub `sql.Result` that errors on `LastInsertId()` → verifies `lastId` is `null`

---

### Task 3: Thread `driver string` through all SQL-building functions (P2-1)

**Files:** `pkg/parsley/evaluator/stdlib_dsl_query.go`
**Effort:** Medium (45 min)
**Commit point:** Yes — after tests pass

This is the largest task but is purely mechanical. No behavior changes — `driver` is passed through but only `buildDeleteSQL` (from Task 1) uses it. Phase 3 will activate it in the other functions.

**Strategy:** Work bottom-up from leaf functions to root functions. This way, each function's callees already have the `driver` parameter when you update its signature.

#### Step 1 — Leaf functions (no SQL-building callees)

Add `driver string` parameter to:

| Function | Line | New signature |
|----------|------|---------------|
| `buildInClause` | L1574 | `buildInClause(column string, value Object, paramIdx *int, driver string)` |
| `buildSubqueryCondition` | L1504 | `buildSubqueryCondition(column string, operator string, subquery *ast.QuerySubquery, env *Environment, paramIdx *int, driver string)` |

`buildSubqueryCondition` calls `buildConditionNodeSQL` internally (L1523) — that will be updated in Step 2.

#### Step 2 — Condition functions (mid-level)

Add `driver string` parameter to:

| Function | Line | Notes |
|----------|------|-------|
| `buildConditionSQL` | L1334 | Calls `buildSubqueryCondition`, `buildInClause` — update those call sites |
| `buildConditionNodeSQL` | L1058 | Calls `buildConditionSQL`, `buildConditionGroupSQL` |
| `buildConditionGroupSQL` | L1086 | Calls `buildConditionNodeSQL` |
| `buildConditionSQLWithCTEs` | L1179 | Calls `buildSubqueryCondition`, `buildInClause` |
| `buildConditionNodeSQLWithCTEs` | L1118 | Calls `buildConditionSQLWithCTEs`, `buildConditionGroupSQLWithCTEs` |
| `buildConditionGroupSQLWithCTEs` | L1146 | Calls `buildConditionNodeSQLWithCTEs` |

#### Step 3 — Correlated/Join condition functions

Add `driver string` parameter to:

| Function | Line | Notes |
|----------|------|-------|
| `buildCorrelatedCondition` | L913 | Leaf for correlated conditions, emits `$N` placeholders |
| `buildCorrelatedConditionSQL` | L877 | Calls `buildCorrelatedCondition` |
| `buildCorrelatedSubquerySQL` | L630 | Calls `buildCorrelatedConditionSQL` |
| `buildCorrelatedConditionWhereClause` | L995 | Calls `buildCorrelatedSubquerySQL` |
| `buildJoinCondition` | L795 | Leaf for join conditions |
| `buildJoinConditionSQL` | L758 | Calls `buildJoinCondition` |

#### Step 4 — Composite/CTE functions

Add `driver string` parameter to:

| Function | Line | Notes |
|----------|------|-------|
| `buildComputedFieldSQL` | L588 | Calls `buildCorrelatedSubquerySQL` |
| `buildJoinSubquerySQL` | L709 | Calls `buildJoinConditionSQL` |
| `buildCTESQL` | L499 | Calls `buildConditionNodeSQLWithCTEs` |

#### Step 5 — Top-level build functions

Update the five entry-point functions to extract `driver` from `binding.DB.Driver` and pass it to all children:

| Function | Line | Change |
|----------|------|--------|
| `buildSelectSQL` | L109 | Add `driver := binding.DB.Driver` at top, pass to all child calls |
| `buildInsertSQL` | L2298 | Same — only needs `driver` for placeholder (Phase 3) |
| `buildInsertSQLForBatch` | L2523 | Same |
| `buildUpdateSQL` | L2652 | Same, calls `buildConditionSQL` |
| `buildDeleteSQL` | L2893 | Already uses `binding.DB.Driver` from Task 1, extract to local `driver` var, pass to `buildConditionSQL` |

#### Step 6 — Non-DSL callers

Update `loadHasManyRelation` (L2060) and `loadBelongsToRelation` (L2146) which also build SQL:

- `loadHasManyRelation`: hardcodes `$1` at L2071 — leave as-is for now (Phase 3 will change it). Pass `parentBinding.DB.Driver` to `buildConditionNodeSQL` at L2082.
- `loadBelongsToRelation`: hardcodes `$1` at L2163 — leave as-is for now (Phase 3).

#### Verification

After all signature changes:
1. `go build ./...` — must compile with no errors
2. `go test ./pkg/parsley/...` — all existing tests pass (behavior unchanged; all tests use SQLite)

---

### Task 4: Add `sqlPlaceholder` helper (prep for Phase 3)

**Files:** `pkg/parsley/evaluator/stdlib_dsl_query.go`
**Effort:** Small (5 min)
**Commit point:** Combined with Task 3

Add a package-level helper alongside `currentTimestampSQL`:

```go
// sqlPlaceholder returns the driver-appropriate parameter placeholder.
// PostgreSQL/SQLite use $1, $2, etc. MySQL uses ? for all positions.
func sqlPlaceholder(driver string, idx int) string {
    if driver == "mysql" {
        return "?"
    }
    return fmt.Sprintf("$%d", idx)
}
```

This function is **not yet called** — it's placed here so Phase 3 can simply replace `fmt.Sprintf("$%d", *paramIdx)` with `sqlPlaceholder(driver, *paramIdx)` everywhere. No behavior change. Include it in the Task 3 commit.

---

### Task 5: Postgres SQL-generation tests (P2-2)

**Files:** `pkg/parsley/tests/dsl_query_postgres_sql_test.go` (new file)
**Effort:** Medium (45 min)
**Commit point:** Yes — final commit

Create a new test file that tests SQL generation directly by constructing AST nodes and `TableBinding` structs with `Driver: "postgres"`, calling the build functions, and asserting the generated SQL strings. No running Postgres instance required.

#### Test helper

```go
func makeTestBinding(driver string, tableName string, softDeleteCol string) *evaluator.TableBinding {
    return &evaluator.TableBinding{
        DB: &evaluator.DBConnection{
            Driver: driver,
        },
        TableName:        tableName,
        SoftDeleteColumn: softDeleteCol,
    }
}
```

Note: `TableBinding`, `DBConnection`, and the build functions are package-internal to `evaluator`. The test file should be in `pkg/parsley/evaluator/` (not `tests/`) to access unexported functions. Name it `dsl_query_postgres_sql_test.go`.

**Revised file:** `pkg/parsley/evaluator/dsl_query_postgres_sql_test.go` (new file)

#### Test cases

| # | Test name | What it asserts |
|---|-----------|-----------------|
| 1 | `TestPostgres_SoftDelete_UsesCurrentTimestamp` | `buildDeleteSQL` with `Driver:"postgres"`, soft delete column → SQL contains `CURRENT_TIMESTAMP`, not `datetime('now')` |
| 2 | `TestSQLite_SoftDelete_UsesDatetime` | `buildDeleteSQL` with `Driver:"sqlite"`, soft delete column → SQL contains `datetime('now')` |
| 3 | `TestPostgres_SelectCondition_UsesDollarPlaceholders` | `buildSelectSQL` with `Driver:"postgres"`, one equality condition → SQL contains `$1` |
| 4 | `TestPostgres_InsertValues_UsesDollarPlaceholders` | `buildInsertSQL` with `Driver:"postgres"` → SQL contains `$1, $2` |
| 5 | `TestPostgres_InsertReturning_HasReturningClause` | `buildInsertSQL` with terminal type `"one"` → SQL contains `RETURNING *` |
| 6 | `TestPostgres_UpdateCondition_UsesDollarPlaceholders` | `buildUpdateSQL` with `Driver:"postgres"` → SQL contains `SET name = $1 ... WHERE id = $2` |
| 7 | `TestPostgres_UpdateReturning_HasReturningClause` | `buildUpdateSQL` with terminal type `"one"` → SQL contains `RETURNING *` |
| 8 | `TestPostgres_DeleteHard_UsesDollarPlaceholders` | `buildDeleteSQL` with `Driver:"postgres"`, no soft delete, one condition → `$1` placeholder |
| 9 | `TestPostgres_DeleteReturning_HasReturningClause` | `buildDeleteSQL` with terminal type `"one"` → SQL contains `RETURNING` |
| 10 | `TestPostgres_Upsert_UsesOnConflictExcluded` | `buildInsertSQL` with upsert key → SQL contains `ON CONFLICT ... EXCLUDED` |
| 11 | `TestPostgres_InClause_UsesDollarPlaceholders` | `buildInClause` with `Driver:"postgres"` and 3-element array → `$1, $2, $3` |
| 12 | `TestPostgres_Between_UsesDollarPlaceholders` | `buildConditionSQL` with `between` operator and `Driver:"postgres"` → `$1` AND `$2` |
| 13 | `TestLastInsertId_Error_ReturnsNull` | Create a mock `sql.Result` where `LastInsertId()` returns error → result dict has `lastId` evaluating to `null` |
| 14 | `TestLastInsertId_Success_ReturnsInteger` | Create a mock `sql.Result` where `LastInsertId()` returns 42 → result dict has `lastId` = 42 |

Tests 1–12 construct minimal AST nodes (just enough fields populated for the SQL builder to work) and call the builder functions directly. Tests 13–14 test the result-dict construction logic from Task 2 (may need a small helper to build the dict and evaluate the `lastId` key).

---

### Task 6: Update ID counter

**Files:** `work/ID_COUNTER.md`
**Effort:** Trivial
**Commit point:** Combined with Task 5

Update PLAN counter:
```
| Plan | PLAN | 115 | PLAN-114 (2026-02-28) |
```

---

## Task Order & Commits

| Order | Task | Branch | Commit message |
|-------|------|--------|----------------|
| 1 | Task 1 (soft delete fix) | `feat/FEAT-134-postgres-fixes` | `fix(dsl): use CURRENT_TIMESTAMP for soft delete on Postgres` |
| 2 | Task 2 (lastInsertId fix) | same branch | `fix(dsl): return null for lastId when driver doesn't support it` |
| 3 | Tasks 3+4 (driver threading + helper) | same branch | `refactor(dsl): thread driver param through SQL-building functions` |
| 4 | Tasks 5+6 (tests + ID counter) | same branch | `test(dsl): add Postgres SQL-generation tests for FEAT-134` |

After all 4 commits, run `make check`. If green, ready for merge to `main`.

---

## Validation Checklist

- [ ] `go build ./...` succeeds
- [ ] `go test ./pkg/parsley/...` passes (all existing + new tests)
- [ ] `make check` passes
- [ ] `golangci-lint run` reports no new issues
- [ ] Soft delete SQL verified for both `sqlite` and `postgres` drivers
- [ ] `lastId` returns `null` when `LastInsertId()` errors
- [ ] `lastId` returns integer when `LastInsertId()` succeeds
- [ ] All SQL-building functions accept `driver string` parameter
- [ ] `sqlPlaceholder` and `currentTimestampSQL` helpers exist and are tested
- [ ] No remaining hardcoded `datetime('now')` in DSL query builders
- [ ] FEAT-134 spec status updated to `in-progress`
- [ ] `work/ID_COUNTER.md` updated

## Deferred Items

Items to add to `work/BACKLOG.md` after implementation:

- **Phase 3 MySQL activation** — `sqlPlaceholder` helper exists but is not yet called. All functions have `driver` parameter but only `buildDeleteSQL` uses it for behavior. Phase 3 (FEAT-134 P3-1 through P3-5) activates MySQL-specific behavior. Estimated ~5 hours.
- **`loadHasManyRelation` / `loadBelongsToRelation` hardcoded `$1`** — These two functions build SQL inline with `fmt.Sprintf("... $1", ...)` at L2071 and L2163 respectively. Phase 3 should convert these to use `sqlPlaceholder()`. Left unchanged in this plan because they require `driver` to flow from the parent binding (which it now does via the updated `buildConditionNodeSQL` calls, but the inline `$1` strings need manual conversion).

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-02-28 | Task 1: soft delete fix | ✅ Complete | `currentTimestampSQL` helper added; `buildDeleteSQL` uses it |
| 2026-02-28 | Task 2: lastInsertId fix | ✅ Complete | Both `evalExecuteStatement` and `evalDatabaseExecute` updated |
| 2026-02-28 | Task 3+4: driver threading + `sqlPlaceholder` | ✅ Complete | All 18 SQL-building functions updated; zero raw `$%d` calls remain |
| 2026-02-28 | Task 5+6: tests + ID counter | ✅ Complete | 19 tests in `dsl_query_postgres_sql_test.go`; all pass; zero new lint issues |