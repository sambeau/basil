---
id: PLAN-115
feature: FEAT-135
title: "Named Parameters in <SQL> Tags"
status: done
created: 2026-02-28
---

# Implementation Plan: FEAT-135 Named Parameters in `<SQL>` Tags

## Overview

Add `:name` style named parameter support to `<SQL>` tags. Users write `:param_name` in SQL text, Parsley validates each against the tag's attributes at evaluation time, and rewrites to driver-native placeholders (`$N` or `?`) at query execution time. Bare `?` placeholders continue to work for backward compatibility.

**Architecture**: Split across two phases of execution:
1. **Tag evaluation** (`evalSQLTag`): Detect whether the SQL uses `:name` or `?` style. Validate `:name` references against attributes. Store a `mode` flag (`"named"` or `"positional"`) in the result dictionary.
2. **Query execution** (`extractSQLAndParams`): If mode is `"named"`, rewrite `:name` → driver placeholder and build positional params array. If `"positional"`, use existing declaration-order logic.

**Estimated total effort:** ~3 hours

## Prerequisites

- [x] FEAT-134 (cross-database placeholders) — complete
- [x] Declaration-order fix committed (`4462a6d`)
- [x] Spec: `work/specs/FEAT-135.md`

## Current Code Reference

Key functions and their locations:

```
evalSQLTag          eval_tags.go:1085     — builds {sql, params} dict from <SQL> tag
extractSQLAndParams eval_database.go:213  — unpacks {sql, params} dict at query time
dictToNamedParams   eval_database.go:255  — converts params dict → positional []any
sqlPlaceholder      stdlib_dsl_query.go:22 — returns $N or ? per driver
newSQLError         eval_errors.go:572    — creates SQL error from catalog
SQL error catalog   errors/errors.go:1258 — SQL-0001 through SQL-0004
```

All callers of `extractSQLAndParams` have a `*DBConnection` (and thus `.Driver`) in scope:
- `evalQueryOneStatement` → `conn` at L22
- `evalQueryManyStatement` → `conn` at L88
- `evalExecuteStatement` → `conn` at L157
- `evalDatabaseQueryOne` → `conn` at L305
- `evalDatabaseQueryMany` → `conn` at L356
- `evalDatabaseExecute` → `conn` at L411

---

## Tasks

### Task 1: Add SQL scanner helper — `scanSQLNamedParams`

**Files:** `pkg/parsley/evaluator/eval_database.go`
**Effort:** Medium (45 min)
**Commit point:** Yes — after unit tests pass

Write a pure function that scans a SQL string and extracts named parameter info. This is the core parsing logic, isolated for easy testing.

```go
type sqlParamMode int

const (
    sqlParamNone       sqlParamMode = iota // no placeholders found
    sqlParamPositional                      // bare ? found
    sqlParamNamed                           // :name found
)

type sqlScanResult struct {
    Mode       sqlParamMode
    NamedParams []string   // ordered list of :name occurrences (may contain duplicates)
    HasQuestion bool       // true if any bare ? found
    HasNamed    bool       // true if any :name found
}
```

Steps:

1. Add `scanSQLNamedParams(sql string) sqlScanResult` to `eval_database.go`.

2. The scanner walks the SQL string character by character, tracking state:
   - **Inside single-quoted string** (`'...'`): skip everything, handle `''` escape (SQL standard doubled quote).
   - **Inside dollar-quoted string** (`$$...$$`): skip everything until matching `$$`. Track the dollar-quote tag for `$tag$...$tag$` syntax.
   - **`::` sequence**: skip both colons (Postgres type cast). When we see `:` and the next char is also `:`, consume both and continue.
   - **`:` followed by letter/underscore**: this is a named param. Read the identifier (`[a-zA-Z_][a-zA-Z0-9_]*`), append to `NamedParams`, set `HasNamed = true`.
   - **`:` followed by digit or other**: not a param, skip.
   - **`?`** (outside quotes): set `HasQuestion = true`.
   - **`--` line comment**: skip to end of line.
   - **`/*` block comment**: skip to `*/`.

3. After scanning, set `Mode`:
   - If `HasNamed && HasQuestion` → leave as-is (caller will error)
   - If `HasNamed` → `sqlParamNamed`
   - If `HasQuestion` → `sqlParamPositional`
   - Otherwise → `sqlParamNone`

Tests (unit tests in `pkg/parsley/evaluator/sql_named_params_test.go`):

| Test | Input | Expected |
|------|-------|----------|
| No params | `SELECT * FROM users` | Mode=None |
| Bare ? | `SELECT * FROM users WHERE id = ?` | Mode=Positional |
| Single :name | `SELECT * FROM users WHERE id = :id` | Mode=Named, params=["id"] |
| Multiple :names | `INSERT INTO users (name, age) VALUES (:name, :age)` | params=["name", "age"] |
| Repeated :name | `WHERE a LIKE :term OR b LIKE :term` | params=["term", "term"] |
| Postgres :: cast | `SELECT id::text FROM users WHERE id = :id` | params=["id"] only |
| Single-quoted skip | `WHERE name = ':not_a_param'` | Mode=None |
| Doubled quote skip | `WHERE name = 'it''s :not_a_param'` | Mode=None |
| Dollar-quoted skip | `$$body with :stuff$$` | Mode=None |
| Tagged dollar quote | `$fn$body with :stuff$fn$` | Mode=None |
| Mixed ? and :name | `WHERE id = :id AND age > ?` | HasNamed=true, HasQuestion=true |
| Line comment skip | `SELECT * -- :not_a_param\nFROM users WHERE id = :id` | params=["id"] |
| Block comment skip | `SELECT * /* :not_a_param */ FROM users WHERE id = :id` | params=["id"] |
| Colon then digit | `SELECT * FROM t WHERE x = :1` | Mode=None (`:1` not valid identifier) |
| Underscore start | `WHERE x = :_private` | params=["_private"] |

---

### Task 2: Add `rewriteNamedParams` helper

**Files:** `pkg/parsley/evaluator/eval_database.go`
**Effort:** Medium (30 min)
**Commit point:** Yes — with Task 1 tests

Write the function that rewrites `:name` → driver placeholder and builds the positional params array.

```go
func rewriteNamedParams(sql string, scan sqlScanResult, paramsDict *Dictionary, env *Environment, driver string) (string, []any, *Error)
```

Steps:

1. Add to `eval_database.go`, below `scanSQLNamedParams`.

2. First, validate all named params have matching attributes:
   ```go
   // Deduplicate names for validation
   seen := make(map[string]bool)
   for _, name := range scan.NamedParams {
       if !seen[name] {
           if _, exists := paramsDict.Pairs[name]; !exists {
               // Build list of available attribute names for error message
               return "", nil, newSQLError("SQL-0005", map[string]any{
                   "Name": name, "Available": strings.Join(availableKeys, ", "),
               })
           }
           seen[name] = true
       }
   }
   ```

3. Evaluate each unique param value once, cache in a `map[string]any`:
   ```go
   evaluated := make(map[string]any, len(seen))
   for name := range seen {
       expr := paramsDict.Pairs[name]
       val := Eval(expr, env)
       if isError(val) { return error }
       evaluated[name] = objectToGoValue(val)
   }
   ```

4. Walk the SQL string again (same state machine as scanner), but this time build a new string:
   - Copy everything verbatim except `:name` tokens
   - Replace each `:name` with `sqlPlaceholder(driver, paramIdx)` and append `evaluated[name]` to the params slice
   - Increment `paramIdx` for each replacement

5. Return the rewritten SQL and positional params.

Tests (in same test file, can use table-driven):

| Test | SQL | Props | Driver | Expected SQL | Expected Params |
|------|-----|-------|--------|-------------|-----------------|
| Basic | `WHERE id = :id` | `{id: 42}` | postgres | `WHERE id = $1` | [42] |
| Basic | `WHERE id = :id` | `{id: 42}` | mysql | `WHERE id = ?` | [42] |
| Basic | `WHERE id = :id` | `{id: 42}` | sqlite | `WHERE id = $1` | [42] |
| Multi | `VALUES (:name, :age)` | `{name: "Alice", age: 30}` | postgres | `VALUES ($1, $2)` | ["Alice", 30] |
| Repeated | `WHERE a = :x OR b = :x` | `{x: 5}` | postgres | `WHERE a = $1 OR b = $2` | [5, 5] |
| With :: | `SELECT id::text WHERE id = :id` | `{id: 1}` | postgres | `SELECT id::text WHERE id = $1` | [1] |
| Missing attr | `WHERE id = :id` | `{}` | sqlite | error SQL-0005 | — |

---

### Task 3: Register error codes

**Files:** `pkg/parsley/errors/errors.go`
**Effort:** Small (5 min)
**Commit point:** Combined with Task 4

Add two new error codes after `SQL-0004`:

```go
"SQL-0005": {
    Class:    ClassType,
    Template: "Unknown parameter ':{{.Name}}' in <SQL> tag — no matching attribute. Available: {{.Available}}",
},
"SQL-0006": {
    Class:    ClassParse,
    Template: "Cannot mix positional ? and named :param placeholders in <SQL> tag",
},
```

---

### Task 4: Update `evalSQLTag` — detect mode and validate

**Files:** `pkg/parsley/evaluator/eval_tags.go`
**Effort:** Medium (30 min)
**Commit point:** Yes — after all evaluator tests pass

Modify `evalSQLTag` to scan the SQL content and validate named params against attributes. The SQL is **not** rewritten here — that happens at query execution time when the driver is known.

Steps:

1. After trimming the SQL, call `scanSQLNamedParams(trimmedSQL)`.

2. If `scan.HasNamed && scan.HasQuestion`, return `SQL-0006` error.

3. If `scan.Mode == sqlParamNamed`:
   - Validate each name in `scan.NamedParams` exists in the props dict. If not, return `SQL-0005`.
   - Add `"mode"` key to result dictionary: `resultPairs["mode"] = &ast.StringLiteral{Value: "named"}`
   - Still store params as a keyed dictionary (same as today) — the keys are needed at rewrite time.

4. If `scan.Mode == sqlParamPositional` or `sqlParamNone`:
   - Keep existing behavior (params in declaration order, no mode key or `mode = "positional"`).

5. The result dictionary shape:
   - **Named mode**: `{sql: "...with :name...", params: {name: val, age: val}, mode: "named"}`
   - **Positional mode**: `{sql: "...with ?...", params: {name: val, age: val}}` (unchanged)

---

### Task 5: Update `extractSQLAndParams` — rewrite at execution time

**Files:** `pkg/parsley/evaluator/eval_database.go`
**Effort:** Medium (30 min)
**Commit point:** Yes — after all evaluator tests pass

Modify `extractSQLAndParams` to accept a `driver string` parameter and handle named-mode rewriting.

Steps:

1. Change signature:
   ```go
   func extractSQLAndParams(queryObj Object, env *Environment, driver string) (string, []any, *Error)
   ```

2. Update all 6 call sites to pass `conn.Driver`:
   - `evalQueryOneStatement` L32: `extractSQLAndParams(queryObj, env, conn.Driver)`
   - `evalQueryManyStatement` L98: same
   - `evalExecuteStatement` L167: same
   - `evalDatabaseQueryOne` L312: same
   - `evalDatabaseQueryMany` L363: same
   - `evalDatabaseExecute` L418: same

3. In the dictionary branch, after extracting `sqlStr` and `paramsDict`:
   ```go
   // Check for named parameter mode
   modeStr := ""
   if modeExpr, hasMode := dict.Pairs["mode"]; hasMode {
       modeObj := Eval(modeExpr, env)
       if s, ok := modeObj.(*String); ok {
           modeStr = s.Value
       }
   }

   if modeStr == "named" && paramsDict != nil {
       scan := scanSQLNamedParams(sqlStr.Value)
       return rewriteNamedParams(sqlStr.Value, scan, paramsDict, env, driver)
   }

   // Existing positional logic
   ```

4. The existing positional path (declaration-order `dictToNamedParams`) remains untouched.

---

### Task 6: Integration tests

**Files:** `pkg/parsley/tests/database_test.go`
**Effort:** Medium (30 min)
**Commit point:** Yes — final commit

Add integration tests to `TestSQLTag` that exercise the full pipeline (parse → evaluate → execute) using SQLite in-memory.

Tests:

1. **Named param basic insert + query**
   ```parsley
   let InsertUser = fn(props) {
       <SQL name={props.name} age={props.age}>
           INSERT INTO tag_users (name, age) VALUES (:name, :age)
       </SQL>
   }
   ```
   Insert with `name="Alice" age={30}`, query back, verify `name="Alice"` and `age=30`.

2. **Named param attribute order irrelevant**
   Same as above but write attributes as `age={props.age} name={props.name}` — same correct result. (This is the key regression test.)

3. **Repeated named param**
   ```parsley
   <SQL term={props.term}>
       SELECT * FROM tag_users WHERE name LIKE :term OR name = :term
   </SQL>
   ```
   Verify correct results with a matching term.

4. **Postgres :: cast not treated as param**
   ```parsley
   let query = <SQL id={1}>
       SELECT id, name FROM tag_users WHERE id = :id
   </SQL>
   query.sql
   ```
   Verify `:id` is present in `sql` and the SQL string does not contain `::` confusion. (Full `::` test is in unit tests; this just checks the tag produces the right dictionary.)

5. **Missing attribute error**
   ```parsley
   let query = <SQL>
       SELECT * FROM tag_users WHERE id = :id
   </SQL>
   ```
   Verify returns error with code `SQL-0005`.

6. **Mixed placeholder error**
   ```parsley
   let query = <SQL id={1}>
       SELECT * FROM tag_users WHERE id = :id AND age > ?
   </SQL>
   ```
   Verify returns error with code `SQL-0006`.

7. **Bare ? backward compatibility**
   ```parsley
   <SQL id={props.id}>
       SELECT * FROM tag_users WHERE id = ?
   </SQL>
   ```
   Verify existing positional behavior still works.

8. **Named params with string literal containing colon**
   ```parsley
   <SQL name={props.name}>
       SELECT * FROM tag_users WHERE name = :name AND bio LIKE '%:not_a_param%'
   </SQL>
   ```
   Verify only `:name` is treated as a param (note: `%:not_a_param%` is outside quotes in SQL — this test validates that the scanner only matches `:identifier` preceded by certain characters. Adjust test if needed to use a quoted string: `'text with :colon'`).

---

### Task 7: Update documentation

**Files:**
- `docs/parsley/manual/features/database.md`
- `docs/basil/reference.md`
- `docs/parsley/manual/fundamentals/tags.md`
- `docs/parsley/manual/features/security.md` (if it mentions placeholder style)
- `work/parsley/design/Database Implementation Status.md`

**Effort:** Small (15 min)
**Commit point:** Yes — final docs commit

Steps:

1. **`database.md`** — Update the `<SQL>` Tag section:
   - Lead with `:name` syntax as the primary/recommended approach
   - Show basic example, repeated param example, multi-column insert
   - Keep `?` as documented alternative for simple cases
   - Remove the "attribute order matters" callout (it no longer matters for `:name`)
   - Keep the callout but narrow it to `?` mode only

2. **`reference.md`** — Update `<SQL>` Tag Attributes bullet:
   - `All attributes are treated as query parameters. Use :name placeholders to bind by name (recommended), or ? for positional binding in declaration order.`

3. **`tags.md`** — Update the SQL section similarly.

4. **`Database Implementation Status.md`** — Mark §1 ("Named Parameters Not Fully Implemented") as ✅ complete, referencing FEAT-135.

---

### Task 8: Update ID counter

**Files:** `work/ID_COUNTER.md`
**Effort:** Trivial
**Commit point:** Combined with Task 7

Update PLAN counter: Next ID → 116, Last Allocated → PLAN-115.

_(Already done when this plan file was created.)_

---

## Task Order & Commits

```
Task 1 + 2: Scanner + rewriter helpers + unit tests
  └─ commit: feat(sql-tag): add named parameter scanner and rewriter

Task 3 + 4: Error codes + evalSQLTag validation
  └─ commit: feat(sql-tag): detect and validate :name params in evalSQLTag

Task 5: extractSQLAndParams driver-aware rewriting
  └─ commit: feat(sql-tag): rewrite :name to driver placeholders at execution time

Task 6: Integration tests
  └─ commit: test(sql-tag): integration tests for named parameters

Task 7 + 8: Documentation + ID counter
  └─ commit: docs(sql-tag): document :name parameter syntax
```

Tasks 1–2 can be developed together (scanner + rewriter are tested in isolation). Tasks 3–5 form the core wiring. Task 6 is the end-to-end validation. Task 7 is cosmetic.

## Validation Checklist

- [ ] All existing tests pass: `go test ./pkg/parsley/...`
- [ ] New unit tests pass for scanner and rewriter
- [ ] New integration tests pass for full SQL tag pipeline
- [ ] Build succeeds: `make dev`
- [ ] Linter passes: `golangci-lint run`
- [ ] Documentation updated (all 4 files)
- [ ] `Database Implementation Status.md` updated
- [ ] `work/BACKLOG.md` updated with deferrals (if any)

## Risk Assessment

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Scanner misses an edge case in SQL quoting | Medium | Comprehensive unit tests for single quotes, dollar quotes, doubled quotes, comments |
| Existing tests break from signature change | Low | `extractSQLAndParams` signature change is mechanical; all 6 call sites are in one file |
| Performance regression from double scan | Very Low | SQL strings are small; two linear scans is negligible. Could optimize to single pass later if needed |
| `lib/pq` rejects `$N` placeholders | Very Low | Already used by the Query DSL (FEAT-134) — confirmed working |

## Deferred Items

Items to add to `work/BACKLOG.md` after implementation:

- **`params={...props}` spread syntax in `<SQL>` tags** — The original design showed `<SQL params={...props}>`. Spread in tag attributes now works (added earlier), but the `params=` attribute pattern was never built. With `:name` params, spreading all of `props` into the attribute list is the natural equivalent and already works: `<SQL ...props>`. Verify this works and document it, or defer to a future item.
- **Single-pass scanner optimization** — Currently scans twice (once in `evalSQLTag` for validation, once in `rewriteNamedParams` for rewriting). Could be combined into a single pass. Not worth the complexity now.

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| | Task 1-2: Scanner + rewriter | ⬜ Not started | |
| | Task 3-4: Errors + evalSQLTag | ⬜ Not started | |
| | Task 5: extractSQLAndParams | ⬜ Not started | |
| | Task 6: Integration tests | ⬜ Not started | |
| | Task 7-8: Docs + ID | ⬜ Not started | |