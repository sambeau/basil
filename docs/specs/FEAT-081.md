---
id: FEAT-081
title: "Rich Schema Types for Query DSL"
status: implemented
priority: medium
created: 2026-01-05
implemented: 2026-01-05
author: "@human"
---

# FEAT-081: Rich Schema Types for Query DSL

## Summary

Extend the `@schema` DSL to support additional types beyond basic primitives, including validated string types (`email`, `url`, `phone`, `slug`), `enum` types with CHECK constraints, and type constraints (`min`, `max`, `unique`). This brings the `@schema` DSL closer to feature parity with `@std/schema` while generating appropriate database schema definitions.

## User Story

As a developer, I want to declare rich field types in my `@schema` definitions so that the database schema includes appropriate constraints and validations are applied on insert/update operations.

## Acceptance Criteria

### Phase 1: Validated String Types
- [x] `email` type stores as TEXT, validates format on insert/update
- [x] `url` type stores as TEXT, validates format on insert/update
- [x] `phone` type stores as TEXT, validates format on insert/update
- [x] `slug` type for URL-safe strings
- [x] Validation errors are clear and include field name

### Phase 2: Enum Types
- [x] `enum("a", "b", "c")` syntax parses correctly
- [x] Creates TEXT column with CHECK constraint
- [x] Validation on insert/update ensures value is in allowed set
- [x] Error messages list allowed values

### Phase 3: Type Constraints
- [x] `string(min: 1, max: 100)` validates length
- [x] `int(min: 0, max: 150)` validates range
- [x] `unique: true` adds UNIQUE constraint to column
- [x] Constraints generate CHECK constraints where possible

### Phase 4: Additional Types (Optional)
- [x] `slug` type for URL-safe strings
- [x] `bigint` for 64-bit integers (recognized, maps to INTEGER)
- [ ] `decimal(precision, scale)` for precise decimals (deferred)

## Design Decisions

- **Store as TEXT, validate in Parsley**: Validated types like `email`, `url`, `phone` store as TEXT in the database. Validation happens in the Parsley layer during `@insert` and `@update` operations. This keeps the database schema simple and portable across SQLite/PostgreSQL.

- **Reuse `@std/schema` validators**: The validation logic already exists in `stdlib_schema.go`. The DSL schema types should delegate to these same validators for consistency.

- **CHECK constraints for enums**: Rather than creating PostgreSQL ENUM types (which have migration issues), use `CHECK (column IN ('a', 'b', 'c'))` which works on both SQLite and PostgreSQL.

- **Constraint syntax**: Use parentheses with named options: `string(min: 1, max: 100)`. This mirrors the `@std/schema` API and is consistent with function call syntax.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Type Mapping Reference

| DSL Type | Parsley Type | SQLite | PostgreSQL | Validator |
|----------|--------------|--------|------------|-----------|
| `int` | Integer | `INTEGER` | `INTEGER` | type check |
| `string` | String | `TEXT` | `TEXT` | type check |
| `bool` | Boolean | `INTEGER` | `BOOLEAN` | type check |
| `float` | Float | `REAL` | `REAL` | type check |
| `datetime` | String/Dict | `TEXT` | `TIMESTAMP` | format check |
| `date` | String | `TEXT` | `DATE` | `YYYY-MM-DD` |
| `time` | String | `TEXT` | `TIME` | `HH:MM:SS` |
| `email` | String | `TEXT` | `TEXT` | email regex |
| `url` | String | `TEXT` | `TEXT` | URL regex |
| `phone` | String | `TEXT` | `TEXT` | phone regex |
| `enum(...)` | String | `TEXT` + CHECK | `TEXT` + CHECK | value in set |
| `uuid` | String | `TEXT` | `UUID` | UUID regex |
| `ulid` | String | `TEXT` | `TEXT` | ULID regex |
| `json` | Dict/Array | `TEXT` | `JSONB` | JSON parse |
| `money` | Integer | `INTEGER` | `INTEGER` | type check |
| `slug` | String | `TEXT` | `TEXT` | slug regex |
| `bigint` | Integer | `INTEGER` | `BIGINT` | type check |

### Affected Components

- `pkg/parsley/ast/ast.go` — Extend `SchemaField` to store type options and enum values
- `pkg/parsley/parser/parser.go` — Parse `type(options)` and `enum("a", "b")` syntax
- `pkg/parsley/evaluator/stdlib_dsl_schema.go` — Store type metadata, update `buildCreateTableSQL`
- `pkg/parsley/evaluator/stdlib_dsl_query.go` — Add validation hooks in `evalInsertExpression` and `evalUpdateExpression`

### Syntax Examples

```parsley
@schema User {
    id: int
    name: string(min: 1, max: 100)
    email: email(unique: true)
    phone: phone
    website: url
    role: enum("admin", "user", "guest")
    age: int(min: 0, max: 150)
    balance: money
    created_at: datetime
}
```

Generated SQL (SQLite):
```sql
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY,
    name TEXT CHECK(length(name) >= 1 AND length(name) <= 100),
    email TEXT UNIQUE,
    phone TEXT,
    website TEXT,
    role TEXT CHECK(role IN ('admin', 'user', 'guest')),
    age INTEGER CHECK(age >= 0 AND age <= 150),
    balance INTEGER,
    created_at TEXT
)
```

### Validation Flow

1. **On `@insert`**: Before building SQL, validate each field value:
   - Check type matches
   - For `email`/`url`/`phone`: run regex validation
   - For `enum`: check value in allowed set
   - For constrained types: check min/max/length
   - If any fail: return validation error with field details

2. **On `@update`**: Same validation for fields being updated

3. **Error format**:
```parsley
{
    error: "VALIDATION_ERROR",
    message: "Validation failed",
    fields: [
        {field: "email", code: "FORMAT", message: "Invalid email format"},
        {field: "role", code: "ENUM", message: "Value must be one of: admin, user, guest"}
    ]
}
```

### Parser Changes

Current `parseSchemaField`:
```
name: type
name: type via foreign_key
name: [type] via foreign_key
```

Extended syntax:
```
name: type
name: type(option: value, ...)
name: enum("value1", "value2", ...)
name: type via foreign_key
```

### AST Changes

```go
type SchemaField struct {
    Name       *Identifier
    TypeName   string
    IsArray    bool
    ForeignKey string
    // New fields:
    TypeOptions map[string]Expression  // {min: 1, max: 100, unique: true}
    EnumValues  []string               // ["admin", "user", "guest"]
}
```

### Edge Cases & Constraints

1. **Enum with special characters** — Enum values must be valid SQL strings (escape quotes)
2. **Null handling** — Constraints only apply to non-null values unless `required: true`
3. **Migration** — Changing constraints requires ALTER TABLE (out of scope)
4. **PostgreSQL UUID** — Could use native UUID type for `uuid` fields on PostgreSQL
5. **Constraint names** — SQLite doesn't support named constraints, PostgreSQL does

### Dependencies

- Depends on: `db.createTable()` (just implemented)
- Blocks: None

## Implementation Notes

*To be added during implementation*

## Related

- `@std/schema` module: `pkg/parsley/evaluator/stdlib_schema.go`
- `db.createTable()`: `pkg/parsley/evaluator/evaluator.go`
- `buildCreateTableSQL()`: `pkg/parsley/evaluator/stdlib_dsl_schema.go`
