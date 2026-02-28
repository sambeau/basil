---
id: FEAT-135
title: "Named Parameters in <SQL> Tags"
status: draft
priority: high
created: 2026-02-28
author: "@human"
---

# FEAT-135: Named Parameters in `<SQL>` Tags

## Summary

The `<SQL>` tag currently binds attribute values to `?` placeholders by declaration order. This is fragile — if a user writes attributes in a different order to their placeholders, values bind to the wrong columns silently. The original database design (`work/parsley/design/Database Design.md`) intended `:name` style named parameters with automatic translation to driver-native placeholders, but this was never implemented (flagged in `work/parsley/design/Database Implementation Status.md` §1).

This feature adds named parameter support: users write `:name` in SQL text, Parsley matches each to the corresponding attribute, and rewrites to `$1`/`$2`/`?` depending on the database driver. Attribute declaration order becomes irrelevant. Bare `?` placeholders continue to work (declaration order) for backward compatibility.

## User Story

As a developer writing `<SQL>` components, I want to reference parameters by name in my SQL so that I don't have to carefully match attribute order to placeholder order, and so that my queries are self-documenting and safe across all database drivers.

## Current Behavior

```parsley
// ✅ Works — but only because attribute order happens to match SQL order
let InsertUser = fn(props) {
    <SQL age={props.age} name={props.name}>
        INSERT INTO users (age, name) VALUES (?, ?)
    </SQL>
}

// ❌ Silently wrong — name goes into age column, age into name column
let InsertUser = fn(props) {
    <SQL name={props.name} age={props.age}>
        INSERT INTO users (age, name) VALUES (?, ?)
    </SQL>
}
```

## Desired Behavior

```parsley
// ✅ Attribute order does not matter — :name references match by name
let InsertUser = fn(props) {
    <SQL name={props.name} age={props.age}>
        INSERT INTO users (age, name) VALUES (:age, :name)
    </SQL>
}

// ✅ Same param used twice — only one attribute needed
let SearchUser = fn(props) {
    <SQL term={props.term}>
        SELECT * FROM users WHERE name LIKE :term OR email LIKE :term
    </SQL>
}

// ✅ Bare ? still works (backward compatibility, declaration order)
let GetUser = fn(props) {
    <SQL id={props.id}>
        SELECT * FROM users WHERE id = ?
    </SQL>
}
```

## Acceptance Criteria

- [ ] `:name` placeholders in `<SQL>` tag content are rewritten to driver-native placeholders (`$N` for Postgres/SQLite, `?` for MySQL)
- [ ] Params array is built in the order `:name` placeholders appear in the SQL, not attribute declaration order
- [ ] A `:name` with no matching attribute produces a clear error (e.g., `SQL-0005: Unknown parameter ':age' — no matching attribute on <SQL> tag`)
- [ ] An attribute with no corresponding `:name` in the SQL is silently ignored (it may be used for other purposes or future use)
- [ ] A `:name` can appear multiple times in the SQL; the same value is bound at each position
- [ ] `::` (Postgres type cast syntax, e.g., `column::text`) is not treated as a named parameter
- [ ] `:name` inside SQL string literals (`'...'`) is not treated as a named parameter
- [ ] Bare `?` placeholders still work with declaration-order binding (backward compatibility)
- [ ] Mixing `?` and `:name` in the same query is an error (e.g., `SQL-0006: Cannot mix positional ? and named :param placeholders`)
- [ ] Works correctly for all three drivers: SQLite, Postgres, MySQL
- [ ] Existing `<SQL>` tag tests continue to pass (backward compatibility)
- [ ] New tests cover: basic named params, repeated params, Postgres `::` casts, string literal escaping, missing attribute error, mixed placeholder error, multi-driver placeholder generation

## Design Decisions

- **`:name` syntax (not `{name}` or `@name`)**: `:name` is the industry standard for named SQL parameters (used by PDO, Rails, JDBI, Spring JDBC, Go's `sqlx`). It avoids collision with `{}` (JSON functions in Postgres), `@` (Parsley sigil), and is unambiguous in SQL context. Matches the original design in `Database Design.md`.

- **Rewrite happens in `evalSQLTag`, not the lexer**: The SQL content is raw text at the lexer level. Named parameter rewriting is a semantic operation that needs access to the props dictionary and the driver, so it belongs in the evaluator when the `<SQL>` tag is processed.

- **Bare `?` still works**: Existing code uses `?` placeholders. We don't break this. The system detects which style is used and applies the appropriate binding strategy. Mixing styles is an error to prevent confusion.

- **Ignored unused attributes**: An attribute not referenced by any `:name` is not an error. This keeps the `<SQL>` tag composable — a component might pass extra props that aren't needed for every query variant.

- **Repeated `:name` duplicates the value**: If `:term` appears twice, the param value is bound twice (at positions `$1` and `$2`, or `?` and `?`). This matches how `database/sql` works — each placeholder position needs its own param entry.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Implementation Approach

The core change is in `evalSQLTag` (`pkg/parsley/evaluator/eval_tags.go`). After trimming the SQL content, before building the result dictionary:

1. **Scan** the SQL string for `:identifier` tokens, skipping:
   - `::` sequences (Postgres type casts)
   - Content inside single-quoted string literals (`'...'`)
   - Content inside `$$` dollar-quoted strings (Postgres)
2. **Check** for mixed mode: if both `?` and `:name` are found, return `SQL-0006` error
3. **If `:name` placeholders found**:
   - For each `:name`, look up the name in the evaluated props dictionary
   - If not found, return `SQL-0005` error with the parameter name
   - Replace `:name` with the driver-appropriate placeholder (`sqlPlaceholder(driver, idx)`)
   - Build the params slice in SQL-occurrence order (a repeated `:name` adds the value again)
   - The driver must be available — look it up from the environment's database connection context, or defer rewriting to `extractSQLAndParams` where the connection is known
4. **If bare `?` placeholders found** (or no placeholders): current behavior unchanged

### Driver Awareness

The `<SQL>` tag currently has no access to the database driver — it just produces a `{sql, params}` dictionary. Two approaches:

**Option A — Defer rewriting to `extractSQLAndParams`**: Keep `evalSQLTag` producing `{sql, params}` where `sql` still contains `:name` tokens and `params` is keyed by name. In `extractSQLAndParams` (which runs at query execution time when the connection/driver is known), scan for `:name` and rewrite to driver-native placeholders while building the positional params array.

**Option B — Always use `?` in rewriting**: Since `database/sql` drivers for Go universally accept `?` positional placeholders (including `lib/pq` for Postgres via its query rewriter), always rewrite `:name` → `?` and don't worry about driver. _Note: verify this works with `lib/pq` — it may require `$N` style._

**Recommendation**: Option A. It cleanly separates tag evaluation (name resolution) from SQL generation (driver-specific placeholders). The `<SQL>` tag stores `sql` with `:name` intact and `params` as a keyed dictionary. `extractSQLAndParams` does the rewrite when it has access to the driver.

### Affected Components

- `pkg/parsley/evaluator/eval_tags.go` — `evalSQLTag`: detect `:name` in SQL, validate against props, store named params
- `pkg/parsley/evaluator/eval_database.go` — `extractSQLAndParams`: rewrite `:name` → driver placeholder, build positional params array
- `pkg/parsley/errors/errors.go` — Register `SQL-0005` (unknown parameter) and `SQL-0006` (mixed placeholders)
- `pkg/parsley/tests/database_test.go` — New integration tests
- `pkg/parsley/evaluator/eval_tags_test.go` or similar — Unit tests for rewriting logic

### Error Catalog Additions

| Code | Class | Template |
|------|-------|----------|
| `SQL-0005` | `type` | `Unknown parameter ':{{.Name}}' in <SQL> tag — no matching attribute. Available: {{.Available}}` |
| `SQL-0006` | `parse` | `Cannot mix positional ? and named :param placeholders in <SQL> tag` |

### Edge Cases & Constraints

1. **Postgres `::` casts** — `SELECT id::text FROM users` must not match `:text` as a parameter. Rule: skip any `:` immediately preceded by another `:`.
2. **String literals** — `WHERE name = ':not_a_param'` must not match. Rule: track single-quote nesting while scanning.
3. **Postgres `$$` dollar quoting** — `$$body with :stuff$$` must not match. Rule: track dollar-quote state.
4. **Numeric suffixes** — `:1` or `:123` are not valid identifiers and should not match (distinguishes from potential Postgres `$1` in user SQL). Rule: `:name` requires the character after `:` to be a letter or underscore.
5. **Empty SQL** — Already handled (returns SQL-0001).
6. **No attributes + `:name` in SQL** — Returns SQL-0005 for the first unmatched name.

### Dependencies

- Depends on: FEAT-134 (cross-database placeholders — complete)
- Related: `work/parsley/design/Database Design.md` (original named params design)
- Related: `work/parsley/design/Database Implementation Status.md` §1 (gap acknowledgment)

### Test Plan

| Test | Description |
|------|-------------|
| Basic named param | `<SQL id={42}>SELECT * FROM users WHERE id = :id</SQL>` → params=[42] |
| Multiple named params | `<SQL name={"Alice"} age={30}>INSERT INTO users (name, age) VALUES (:name, :age)</SQL>` → params=["Alice", 30] |
| Attribute order irrelevant | Same as above but `age` attribute listed before `name` — same result |
| Repeated param | `<SQL term={"foo"}>... WHERE a LIKE :term OR b LIKE :term</SQL>` → params=["foo", "foo"] |
| Postgres `::` cast | `<SQL id={1}>SELECT name::text FROM users WHERE id = :id</SQL>` → only `:id` matched |
| String literal skip | `<SQL>SELECT * FROM users WHERE name = ':not_a_param'</SQL>` → no params |
| Missing attribute error | `<SQL>SELECT * FROM users WHERE id = :id</SQL>` → SQL-0005 |
| Mixed placeholder error | `<SQL id={1}>SELECT * FROM users WHERE id = :id AND age > ?</SQL>` → SQL-0006 |
| Bare `?` backward compat | `<SQL id={1}>SELECT * FROM users WHERE id = ?</SQL>` → works as before |
| Driver: Postgres | `:name` → `$1`, `$2`, ... |
| Driver: MySQL | `:name` → `?`, `?`, ... |
| Driver: SQLite | `:name` → `$1`, `$2`, ... |

## Related

- Plan: `work/plans/PLAN-115-sql-named-params.md`
- Original design: `work/parsley/design/Database Design.md`
- Gap tracking: `work/parsley/design/Database Implementation Status.md` §1, §2
- Cross-DB placeholders: `work/specs/FEAT-134.md`
- Declaration-order fix: commit `4462a6d` (fix(sql-tag): bind params in declaration order)
- Backlog item #5: "Parameterized queries for raw SQL operators" (marked complete but named params were not included)