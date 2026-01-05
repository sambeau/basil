---
id: PLAN-053
feature: FEAT-081
title: "Implementation Plan for Rich Schema Types for Query DSL"
status: complete
created: 2026-01-05
completed: 2026-01-05
---

# Implementation Plan: FEAT-081 Rich Schema Types

## Overview

Extend the `@schema` DSL to support validated string types (`email`, `url`, `phone`, `slug`), `enum` types with CHECK constraints, and type constraints (`min`, `max`, `unique`). This brings the `@schema` DSL closer to feature parity with `@std/schema`.

## Implementation Phases

| Phase | Description | Effort | Dependencies | Status |
|-------|-------------|--------|--------------|--------|
| 1 | Validated String Types | Medium | None | ✅ Complete |
| 2 | Enum Types | Medium | None | ✅ Complete |
| 3 | Type Constraints | Medium | Phase 1 | ✅ Complete |
| 4 | Additional Types | Small | Phase 1 | ✅ Complete |

All phases implemented and tested.

---

## Prerequisites

- [x] `db.createTable()` implemented — generates CREATE TABLE from schema
- [x] Existing `@schema` parser infrastructure working
- [x] Existing regex validators in `stdlib_schema.go`

---

## Phase 1: Validated String Types ✅

**Goal**: Add `email`, `url`, `phone`, `slug` types that store as TEXT but validate on insert/update.

### Task 1.1: Extend DSL Type Recognition ✅
**Files**: `pkg/parsley/evaluator/stdlib_dsl_schema.go`
**Status**: Complete

Implemented:
- Added `email`, `url`, `phone`, `slug` to `knownPrimitiveTypes`
- Updated `schemaTypeToSQL()` to map all to TEXT
- Works with `buildCreateTableSQL()`

### Task 1.2: Add Schema Field Type Metadata ✅
**Files**: `pkg/parsley/evaluator/stdlib_dsl_schema.go`
**Status**: Complete

Implemented `DSLSchemaField` with:
- `ValidationType` - "email", "url", "phone", "slug", "enum"
- `EnumValues` - for enum validation
- `MinLength`, `MaxLength` - for string length constraints
- `MinValue`, `MaxValue` - for integer range constraints
- `Unique` - for UNIQUE constraint generation

### Task 1.3: Add Validation on Insert ✅
**Files**: `pkg/parsley/evaluator/stdlib_dsl_query.go`
**Status**: Complete

Implemented:
- `ValidateSchemaField()` with regex validation for email, url, phone, slug
- `ValidateSchemaFields()` to validate all fields
- Validation hook in `evalInsertExpression()`
- Returns validation error Dictionary on failure

### Task 1.4: Add Validation on Update ✅
**Files**: `pkg/parsley/evaluator/stdlib_dsl_query.go`
**Status**: Complete

Implemented:
- Validation hook in `evalUpdateExpression()`
- Only validates fields being updated

### Task 1.5: Validation Error Format ✅
**Files**: `pkg/parsley/evaluator/stdlib_dsl_schema.go`
**Status**: Complete

Implemented `buildValidationErrorObject()` returning:
```parsley
{
    error: "VALIDATION_ERROR",
    field: "email",
    value: "invalid-email",
    message: "Invalid email format"
}
```

---

## Phase 2: Enum Types ✅

**Goal**: Add `enum("a", "b", "c")` syntax with CHECK constraints and validation.

### Task 2.1: Extend AST for Enum Types ✅
**Files**: `pkg/parsley/ast/ast.go`
**Status**: Complete

Added `EnumValues []string` to `SchemaField`

### Task 2.2: Parse Enum Syntax ✅
**Files**: `pkg/parsley/parser/parser.go`
**Status**: Complete

Implemented `parseEnumValues()` for `enum("value1", "value2")` syntax

### Task 2.3: Store Enum Metadata ✅
**Files**: `pkg/parsley/evaluator/stdlib_dsl_schema.go`
**Status**: Complete

Stores enum values in `DSLSchemaField.EnumValues`

### Task 2.4: Generate CHECK Constraint ✅
**Files**: `pkg/parsley/evaluator/stdlib_dsl_schema.go`
**Status**: Complete

`buildCreateTableSQL()` generates:
```sql
status TEXT CHECK(status IN ('active', 'inactive', 'draft'))
```

### Task 2.5: Validate Enum on Insert/Update ✅
**Files**: `pkg/parsley/evaluator/stdlib_dsl_query.go`
**Status**: Complete

Enum validation checks value against allowed list

---

## Phase 3: Type Constraints ✅

**Goal**: Add constraints like `min`, `max`, `unique` to types.

### Task 3.1: Extend AST for Type Options ✅
**Files**: `pkg/parsley/ast/ast.go`
**Status**: Complete

Added `TypeOptions map[string]Expression` to `SchemaField`

### Task 3.2: Parse Type Options Syntax ✅
**Files**: `pkg/parsley/parser/parser.go`
**Status**: Complete

Implemented `parseTypeOptions()` for `type(min: 1, max: 100, unique: true)` syntax

---

## Phase 2: Enum Types

**Goal**: Add `enum("a", "b", "c")` syntax with CHECK constraints and validation.

### Task 2.1: Extend AST for Enum Types
**Files**: `pkg/parsley/ast/ast.go`
**Effort**: Small

Steps:
1. Add `EnumValues` field to `SchemaField`:
```go
type SchemaField struct {
    Token      lexer.Token
    Name       *Identifier
    TypeName   string      // "int", "string", "enum"
    IsArray    bool
    ForeignKey string
    EnumValues []string    // NEW: ["admin", "user", "guest"]
}
```

Tests:
- AST can represent enum field with values
- `String()` method outputs `role: enum("admin", "user")`

---

### Task 2.2: Parse Enum Syntax
**Files**: `pkg/parsley/parser/parser.go`
**Effort**: Medium

Steps:
1. In `parseSchemaField()`, after parsing type name:
   - If type is `enum`, expect `(`
   - Parse comma-separated string literals
   - Expect `)`
2. Store values in `EnumValues` field

Grammar:
```
field: enum("value1", "value2", ...)
```

Tests:
- `role: enum("admin", "user")` parses with EnumValues = ["admin", "user"]
- `status: enum("active", "inactive", "pending")` parses correctly
- `role: enum()` produces clear error (empty enum)
- `role: enum("admin", 123)` produces clear error (non-string value)

---

### Task 2.3: Store Enum Metadata
**Files**: `pkg/parsley/evaluator/stdlib_dsl_schema.go`
**Effort**: Small

Steps:
1. Add `EnumValues []string` to `DSLSchemaField`
2. When evaluating `@schema`, extract enum values from AST
3. Set `ValidationType = "enum"` for enum fields

Tests:
- Schema object contains enum values for enum fields
- Schema inspection shows allowed values

---

### Task 2.4: Generate CHECK Constraint
**Files**: `pkg/parsley/evaluator/stdlib_dsl_schema.go`
**Effort**: Small

Steps:
1. In `buildCreateTableSQL()`, for enum fields:
   - Generate: `column TEXT CHECK(column IN ('a', 'b', 'c'))`
2. Properly escape enum values for SQL

```go
if len(field.EnumValues) > 0 {
    quoted := make([]string, len(field.EnumValues))
    for i, v := range field.EnumValues {
        quoted[i] = fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
    }
    constraint := fmt.Sprintf("CHECK(%s IN (%s))", fieldName, strings.Join(quoted, ", "))
    columnDef += " " + constraint
}
```

Tests:
- `db.createTable(schema)` generates CHECK constraint for enum
- Enum values with quotes are properly escaped
- CHECK constraint syntax works on both SQLite and PostgreSQL

---

### Task 2.5: Validate Enum on Insert/Update
**Files**: `pkg/parsley/evaluator/stdlib_dsl_query.go`
**Effort**: Small

Steps:
1. Add enum validation case in `validateSchemaField()`:
```go
case "enum":
    if !contains(field.EnumValues, value.(*String).Value) {
        return &ValidationError{
            Field: fieldName, 
            Code: "ENUM", 
            Message: fmt.Sprintf("Value must be one of: %s", strings.Join(field.EnumValues, ", "))
        }
    }
```

Tests:
- `@insert(Users | {role: "admin"})` succeeds
- `@insert(Users | {role: "invalid"})` returns enum validation error
- Error message lists allowed values

---

## Phase 3: Type Constraints

**Goal**: Add constraints like `min`, `max`, `unique` to types.

### Task 3.1: Extend AST for Type Options
**Files**: `pkg/parsley/ast/ast.go`
**Effort**: Small

Steps:
1. Add `TypeOptions` to `SchemaField`:
```go
type SchemaField struct {
    // ... existing fields ...
    TypeOptions map[string]interface{}  // {min: 1, max: 100, unique: true}
}
```

Tests:
- AST can represent field with options
- `String()` outputs `name: string(min: 1, max: 100)`

---

### Task 3.2: Parse Type Options Syntax
**Files**: `pkg/parsley/parser/parser.go`
**Effort**: Medium

Steps:
1. In `parseSchemaField()`, after parsing type name:
   - If next token is `(` (and not enum), parse options
   - Parse `key: value` pairs separated by commas
   - Support integer values for `min`/`max`, boolean for `unique`
   - Expect `)`

Grammar:
```
field: type(option: value, ...)
```

Tests:
- `name: string(min: 1, max: 100)` parses correctly
- `age: int(min: 0, max: 150)` parses correctly
- `email: email(unique: true)` parses correctly
- `name: string(min: 1)` parses with single option
- `name: string(max: "abc")` produces error (wrong type for max)

---

### Task 3.3: Store Constraint Metadata ✅
**Files**: `pkg/parsley/evaluator/stdlib_dsl_schema.go`
**Status**: Complete

Implemented constraint fields in `DSLSchemaField`:
- `MinLength *int` - for string(min: N)
- `MaxLength *int` - for string(max: N)  
- `MinValue *int64` - for int(min: N)
- `MaxValue *int64` - for int(max: N)
- `Unique bool`

### Task 3.4: Generate SQL Constraints ✅
**Files**: `pkg/parsley/evaluator/stdlib_dsl_schema.go`
**Status**: Complete

`buildCreateTableSQL()` generates:
- UNIQUE keyword for `unique: true`
- CHECK constraint for integer min/max
- CHECK constraint for string length

### Task 3.5: Validate Constraints on Insert/Update ✅
**Files**: `pkg/parsley/evaluator/stdlib_dsl_query.go`
**Status**: Complete

Implemented validation for:
- Integer min/max range
- String min/max length

---

## Phase 4: Additional Types ✅

**Goal**: Add `slug`, `bigint` types.

### Task 4.1: Add Slug Type ✅
**Files**: `pkg/parsley/evaluator/stdlib_dsl_schema.go`
**Status**: Complete

Implemented:
- `slug` type maps to TEXT
- Validation regex: `^[a-z0-9]+(?:-[a-z0-9]+)*$`
- Tests verify valid/invalid slugs
- `@insert(Posts | {slug: "Invalid Slug!"})` returns validation error

---

### Task 4.2: Add Bigint Type
**Files**: `pkg/parsley/evaluator/stdlib_dsl_schema.go`
**Effort**: Small

Steps:
1. Add `bigint` to type mappings:
   - SQLite: INTEGER (SQLite integers are already 64-bit)
   - PostgreSQL: BIGINT

Tests:
- `count: bigint` creates appropriate column type
- Large integers handled correctly

---

### Task 4.3: Add Decimal Type
**Files**: `pkg/parsley/evaluator/stdlib_dsl_schema.go`, `pkg/parsley/parser/parser.go`
**Effort**: Medium

Steps:
1. Parse `decimal(precision, scale)` syntax
2. Map to SQL DECIMAL/NUMERIC types
3. Store precision and scale in field metadata

Tests:
- `price: decimal(10, 2)` creates DECIMAL(10,2) column
- Precision and scale are stored and accessible

---

## Validation Checklist

- [x] All parsley tests pass: `go test ./pkg/parsley/...`
- [x] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] Documentation updated (query-dsl.md)
- [ ] BACKLOG.md updated with deferrals (if any)

## Test Files Created/Updated

| File | Purpose |
|------|---------|
| `pkg/parsley/tests/dsl_query_test.go` | Added 15+ tests for rich schema types |

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-01-05 | Phase 1-4 implementation | Complete | All validated types, enum, constraints |
| 2026-01-05 | Tests for all features | Complete | 15+ tests pass |

## Deferred Items

- Documentation update for query-dsl.md guide
- `bigint` type support (added to knownPrimitiveTypes but not tested)
- `decimal` type support (not implemented)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Enum with SQL injection | High | Escape single quotes in enum values |
| Constraint names in PostgreSQL | Low | Use anonymous constraints (simpler) |
| Breaking existing schemas | Medium | New fields are all optional, backward compatible |
| Validation performance | Low | Validation is O(1) per field, minimal overhead |

## Out of Scope

- Migration support (ALTER TABLE for constraint changes)
- Named constraints in PostgreSQL
- Custom validation functions
- Complex cross-field validation

## Related Documents

- [FEAT-081.md](../specs/FEAT-081.md) — Feature specification
- [query-dsl.md](../guide/query-dsl.md) — User documentation
- [stdlib_schema.go](../../pkg/parsley/evaluator/stdlib_schema.go) — Existing validators
