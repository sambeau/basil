---
id: FEAT-078
title: "TableBinding Extended Query Methods"
status: implemented
priority: medium
created: 2026-01-02
implemented: 2026-01-04
author: "@copilot"
---

# FEAT-078: TableBinding Extended Query Methods

## Summary

Extend the TableBinding API with optional query options (orderBy, select, limit/offset), aggregation methods (count, sum, avg, min, max), and convenience methods (first, last, exists, findBy). All additions are optional parameters or new methods — existing code continues to work unchanged. Each method = one SQL query.

## Motivation

The current TableBinding API covers basic CRUD but lacks:
- Sorting without raw SQL
- Column selection (always `SELECT *`)
- Aggregations without raw SQL
- Convenience methods like `first()` and `last()`

These are common operations that shouldn't require dropping to raw SQL.

## Design Principles

1. **Backward compatible** — All existing code works unchanged
2. **One method = one query** — No hidden query chains
3. **Optional parameters** — New functionality via optional second argument
4. **Matches Table API** — Consistent with in-memory Table where sensible

---

## Specification

### 1. Extend `all()` with Options

**Current signature:** `all()`

**Extended signature:** `all(options?)`

```parsley
// Current (unchanged)
Users.all()

// Extended with options
Users.all({orderBy: "name"})
Users.all({orderBy: "created_at", order: "desc"})
Users.all({orderBy: [["age", "desc"], ["name", "asc"]]})
Users.all({select: ["id", "name", "email"]})
Users.all({limit: 10, offset: 20})  // Override auto-pagination
```

#### Options Dictionary

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `orderBy` | string \| array | none | Column name or array of `[col, dir]` pairs |
| `order` | `"asc"` \| `"desc"` | `"asc"` | Direction when `orderBy` is string |
| `select` | array of strings | all columns | Column names to return |
| `limit` | integer | auto (20) | Max rows (overrides auto-pagination) |
| `offset` | integer | auto (0) | Skip rows |

#### SQL Generation

```parsley
Users.all({orderBy: "name", order: "desc", limit: 10})
// → SELECT * FROM users ORDER BY name DESC LIMIT 10 OFFSET 0

Users.all({select: ["id", "name"], orderBy: [["age", "desc"], ["name", "asc"]]})
// → SELECT id, name FROM users ORDER BY age DESC, name ASC LIMIT 20 OFFSET 0
```

---

### 2. Extend `where()` with Options

**Current signature:** `where(conditions)`

**Extended signature:** `where(conditions, options?)`

```parsley
// Current (unchanged)
Users.where({role: "admin"})

// Extended with options
Users.where({role: "admin"}, {orderBy: "name"})
Users.where({active: true}, {select: ["id", "name"], limit: 5})
```

Same options as `all()` except no auto-pagination (returns all matches by default).

---

### 3. Aggregation Methods

New methods that return a single value (not an array).

#### `count(conditions?)`

```parsley
Users.count()                    // SELECT COUNT(*) FROM users
Users.count({role: "admin"})     // SELECT COUNT(*) FROM users WHERE role = ?
```

Returns: integer

#### `sum(column, conditions?)`

```parsley
Users.sum("balance")                   // SELECT SUM(balance) FROM users
Users.sum("balance", {active: true})   // SELECT SUM(balance) FROM users WHERE active = ?
```

Returns: number (or null if no rows)

#### `avg(column, conditions?)`

```parsley
Users.avg("age")                       // SELECT AVG(age) FROM users
Users.avg("age", {role: "admin"})      // SELECT AVG(age) FROM users WHERE role = ?
```

Returns: number (or null if no rows)

#### `min(column, conditions?)`

```parsley
Users.min("created_at")                // SELECT MIN(created_at) FROM users
Users.min("score", {active: true})     // SELECT MIN(score) FROM users WHERE active = ?
```

Returns: value (or null if no rows)

#### `max(column, conditions?)`

```parsley
Users.max("score")                     // SELECT MAX(score) FROM users
Users.max("updated_at", {role: "admin"}) // SELECT MAX(updated_at) FROM users WHERE role = ?
```

Returns: value (or null if no rows)

---

### 4. Convenience Methods

#### `first(n?, options?)`

Get first record(s) ordered by primary key.

```parsley
Users.first()                         // ORDER BY id ASC LIMIT 1
Users.first(5)                        // ORDER BY id ASC LIMIT 5
Users.first({orderBy: "created_at"})  // ORDER BY created_at ASC LIMIT 1
Users.first(3, {orderBy: "score", order: "desc"})  // ORDER BY score DESC LIMIT 3
```

Returns: single record (or null) when no `n`, array when `n` specified.

#### `last(n?, options?)`

Get last record(s) ordered by primary key (reversed).

```parsley
Users.last()                          // ORDER BY id DESC LIMIT 1
Users.last(5)                         // ORDER BY id DESC LIMIT 5
Users.last({orderBy: "created_at"})   // ORDER BY created_at DESC LIMIT 1
```

Returns: single record (or null) when no `n`, array when `n` specified.

#### `exists(conditions)`

Check if any matching record exists. More efficient than `where()` for existence checks.

```parsley
Users.exists({email: "alice@example.com"})  // SELECT 1 FROM users WHERE email = ? LIMIT 1
```

Returns: boolean (`true` if at least one match, `false` otherwise)

#### `findBy(conditions, options?)`

Like `where()` but returns first match or null (not an array).

```parsley
Users.findBy({email: "alice@example.com"})
// → SELECT * FROM users WHERE email = ? LIMIT 1
// Returns: {id: ..., email: ..., ...} or null

Users.findBy({role: "admin"}, {orderBy: "created_at"})
// → SELECT * FROM users WHERE role = ? ORDER BY created_at ASC LIMIT 1
```

Returns: single record or null

---

## Implementation Notes

### Column Name Validation

All column names in `orderBy` and `select` must pass the existing identifier regex: `^[A-Za-z_][A-Za-z0-9_]*$`

### SQL Building

Extend `executeAll` and `executeWhere` in `stdlib_schema_table_binding.go` to accept options and build SQL clauses:

```go
func buildOrderByClause(options *Dictionary) (string, error) {
    // Handle string: "name" → "ORDER BY name ASC"
    // Handle string+order: "name", "desc" → "ORDER BY name DESC"  
    // Handle array: [["age", "desc"], ["name", "asc"]] → "ORDER BY age DESC, name ASC"
}

func buildSelectClause(options *Dictionary) (string, error) {
    // Default: "*"
    // With select: ["id", "name"] → "id, name"
}
```

### New Methods

Add to `evalTableBindingMethod`:
- `count` → `executeCount`
- `sum` → `executeSum`
- `avg` → `executeAvg`
- `min` → `executeMin`
- `max` → `executeMax`
- `first` → `executeFirst`
- `last` → `executeLast`
- `exists` → `executeExists`
- `findBy` → `executeFindBy`

---

## Test Cases

### all() with options
- `all({orderBy: "name"})` returns sorted results
- `all({orderBy: "name", order: "desc"})` returns reverse sorted
- `all({orderBy: [["a", "desc"], ["b", "asc"]]})` multi-column sort
- `all({select: ["id", "name"]})` returns only specified columns
- `all({limit: 5, offset: 10})` overrides pagination
- Invalid column name in orderBy → error
- Invalid column name in select → error

### where() with options
- `where({x: 1}, {orderBy: "y"})` filters and sorts
- `where({x: 1}, {select: ["a", "b"]})` filters and projects

### Aggregations
- `count()` returns total count
- `count({role: "admin"})` returns filtered count
- `sum("balance")` returns sum
- `sum("balance", {active: true})` returns filtered sum
- `avg("age")` returns average
- `min("created_at")` returns minimum
- `max("score")` returns maximum
- Aggregation on empty table → returns null (except count → 0)

### Convenience methods
- `first()` returns single record or null
- `first(5)` returns array of up to 5 records
- `last()` returns last record by id
- `exists({email: "x"})` returns true/false
- `findBy({email: "x"})` returns single record or null

---

## Migration

No migration needed. All changes are additive.

---

## Future Considerations

### Predicate-based where (deferred)

Support function argument for client-side filtering:

```parsley
// SQL filter + client-side refinement
Users.where({active: true}, fn(row) { strings.hasPrefix(row.name, "A") })
```

This would fetch all `active=true` rows, then filter client-side. Matches in-memory Table API but has performance implications that need documentation.

### groupBy for aggregations (deferred)

```parsley
Users.count({}, {groupBy: "role"})  // → {admin: 5, user: 100}
```

Adds complexity; defer to Query Builder investigation.

---

## Related

- FEAT-034: Original TableBinding implementation
- In-memory Table API in `stdlib_table.go`
