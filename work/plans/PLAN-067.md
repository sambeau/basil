---
id: PLAN-067
feature: FEAT-094
title: "Implementation Plan for Schema Enhancements"
status: complete
created: 2026-01-17
completed: 2026-01-17
---

# Implementation Plan: FEAT-094 Schema Enhancements

## Overview

Implement schema enhancements covering:
1. **String pattern constraint** — Regex validation for strings
2. **ID type clarification** — `id` as alias for `ulid`, explicit `uuid`/`ulid`/`int`/`bigint` types
3. **Money currency metadata** — Display formatting via `record.format()`

## Implementation Summary

All phases completed successfully:

- **Phase 1 (Pattern Constraint)**: Added `Pattern` and `PatternSource` fields to `DSLSchemaField`, implemented `validatePattern()` function, added `PATTERN` error code, and integrated pattern into form binding via HTML `pattern` attribute.

- **Phase 2 (ID Types)**: Added `id` type as alias for `ulid` at schema parse time (SPEC-ID-002), updated SQL generation for `uuid(auto)`/`ulid(auto)`/`int(auto)`/`bigint(auto)` with proper PRIMARY KEY handling, and implemented auto ID generation in `buildInsertSQL`.

- **Phase 3 (Money Currency)**: Enhanced `record.format()` to check money type with `currency` metadata and use `formatCurrencyWithLocale()` for proper currency symbol formatting.

## Behavioral Changes (Breaking)

The `id` type is now an alias for `ulid`. This means:
- `id: id` without `auto` expects values in valid ULID format (26 characters, base32)
- `id: id(auto)` auto-generates ULID on insert
- Existing code using `id: id` with arbitrary string IDs should change to `id: string` or use valid ULID values

Updated tests:
- `schema_mutation_test.go`: Changed `id: id` to `id: string` for tests using arbitrary string IDs
- `record_db_test.go`: Changed `id: uuid` to `id: uuid(auto)` for tests expecting auto-generation

## Files Modified

- `pkg/parsley/evaluator/stdlib_dsl_schema.go` — Pattern parsing, id→ulid alias, SQL generation
- `pkg/parsley/evaluator/record_validation.go` — Pattern validation, PATTERN error code
- `pkg/parsley/evaluator/form_binding.go` — HTML pattern attribute output
- `pkg/parsley/evaluator/stdlib_dsl_query.go` — Auto ULID/UUID generation on insert
- `pkg/parsley/evaluator/methods_record.go` — Currency metadata in format()
- `pkg/parsley/evaluator/stdlib_schema_table_binding.go` — Auto flag check in generateID()
- `pkg/parsley/tests/record_test.go` — New tests for pattern, id alias, currency
- `pkg/parsley/tests/schema_mutation_test.go` — Updated for id type changes
- `pkg/parsley/tests/record_db_test.go` — Updated for uuid type changes

## Documentation Updated

- `docs/parsley/manual/builtins/schema.md` — Pattern constraint, ID types, currency metadata
- `docs/parsley/reference.md` — Added pattern to constraints table
- `docs/parsley/CHEATSHEET.md` — Added gotcha #11 for id type requiring auto

## Prerequisites

- [x] FEAT-091 Record Type core implementation complete
- [x] FEAT-094 specification complete
- [x] Read `pkg/parsley/evaluator/dsl_schema.go` for schema field parsing
- [x] Read `pkg/parsley/evaluator/record_validation.go` for validation logic
- [x] Read `pkg/parsley/evaluator/record_methods.go` for `record.format()`

---

## Phase 1: String Pattern Constraint

### Task 1.1: Add Pattern Fields to DSLSchemaField

**Files**: `pkg/parsley/evaluator/dsl_schema.go`
**Estimated effort**: Small
**Status**: ✅ Complete

Steps:
1. Add `Pattern *regexp.Regexp` field to `DSLSchemaField` struct
2. Add `PatternSource string` field to store original regex string (for HTML)
3. Export both in the field info dictionary

Tests:
- Verify `fields` accessor includes pattern info when present

---

### Task 1.2: Parse Pattern Constraint

**Files**: `pkg/parsley/evaluator/dsl_schema.go`
**Estimated effort**: Medium
**Status**: ✅ Complete

Steps:
1. In `parseDSLFieldOptions()`, handle `pattern` key
2. Expect a Regex object as the value
3. Compile and store in `Pattern`; store source in `PatternSource`
4. Produce parse error if regex is invalid

Tests:
- `string(pattern: /^[a-z]+$/)` parses correctly
- `string(pattern: "not-a-regex")` produces error
- Pattern combined with other constraints: `string(min: 1, pattern: /^[a-z]+$/)`

---

### Task 1.3: Implement Pattern Validation

**Files**: `pkg/parsley/evaluator/record_validation.go`
**Estimated effort**: Medium

Steps:
1. Add `validatePattern()` function per spec
2. Empty strings pass pattern validation (use `required` for non-empty)
3. Error code: `PATTERN`
4. Error message: `"{title} does not match the required format"`
5. Call from main validation loop

Tests:
- `"Alice"` matches `/^[A-Za-z]+$/` → valid
- `"Alice123"` doesn't match → PATTERN error
- `""` matches any pattern → valid (empty passes)
- Combined: `string(min: 1, pattern: /^[A-Z]/)` — empty fails MIN_LENGTH first

---

### Task 1.4: Pattern in Form Binding

**Files**: `pkg/parsley/evaluator/record_form.go` (or similar)
**Estimated effort**: Small

Steps:
1. In `@field` rendering, check for `PatternSource`
2. Emit `pattern="{source}"` attribute on `<input>`
3. Convert Go regex to JS-compatible where possible (basic patterns)
4. Log warning if unconvertible (e.g., Go-specific features)

Tests:
- `<input pattern="^[a-z]+$">` generated for simple patterns
- Complex patterns handled gracefully

---

## Phase 2: ID Type Clarification

### Task 2.1: Add `id` Type Alias

**Files**: `pkg/parsley/evaluator/dsl_schema.go`
**Estimated effort**: Small

Steps:
1. In field type normalization, expand `id` → `ulid`
2. Early in processing, before validation/SQL generation

Tests:
- `@schema T { id: id(auto) }` → `T.fields.id.type` is `"ulid"`
- `id` without `auto` validates as ULID format

---

### Task 2.2: Validate UUID Format

**Files**: `pkg/parsley/evaluator/record_validation.go`
**Estimated effort**: Small

Steps:
1. Add UUID format validation regex: `/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i`
2. Apply when type is `uuid` and NOT `auto`
3. Error code: `FORMAT`
4. Error message: `"{title} must be a valid UUID"`

Tests:
- `550e8400-e29b-41d4-a716-446655440000` → valid
- `not-a-uuid` → FORMAT error
- `uuid(auto)` field → validation skipped

---

### Task 2.3: Validate ULID Format

**Files**: `pkg/parsley/evaluator/record_validation.go`
**Estimated effort**: Small

Steps:
1. Add ULID format validation regex: `/^[0-7][0-9A-HJKMNP-TV-Z]{25}$/`
2. Apply when type is `ulid` and NOT `auto`
3. Error code: `FORMAT`
4. Error message: `"{title} must be a valid ULID"`

Tests:
- `01ARZ3NDEKTSV4RRFFQ69G5FAV` → valid
- `not-a-ulid` → FORMAT error
- `ulid(auto)` field → validation skipped

---

### Task 2.4: Update SQL Generation for ID Types

**Files**: `pkg/parsley/evaluator/database.go` (or `createTable` location)
**Estimated effort**: Medium

Steps:
1. Handle `uuid(auto)`:
   - SQLite: `TEXT PRIMARY KEY`
   - PostgreSQL: `UUID PRIMARY KEY DEFAULT gen_random_uuid()`
2. Handle `ulid(auto)`:
   - SQLite: `TEXT PRIMARY KEY`
   - PostgreSQL: `TEXT PRIMARY KEY`
3. Handle `int(auto)`:
   - SQLite: `INTEGER PRIMARY KEY` (implicit autoincrement)
   - PostgreSQL: `SERIAL PRIMARY KEY`
4. Handle `bigint(auto)`:
   - SQLite: `INTEGER PRIMARY KEY`
   - PostgreSQL: `BIGSERIAL PRIMARY KEY`

Tests:
- `db.createTable(Schema, "table")` generates correct DDL for each type
- SQLite and PostgreSQL variants tested

---

### Task 2.5: ULID/UUID Generation on Insert

**Files**: `pkg/parsley/evaluator/database_insert.go` (or insert handler)
**Estimated effort**: Medium

Steps:
1. For `ulid(auto)` fields: generate ULID at insert time if not provided
2. For `uuid(auto)` fields: generate UUID v4 at insert time if not provided
3. Ensure generated value is returned in the result record

Tests:
- Insert without ID → ID generated and returned
- Insert with ID (non-auto) → provided ID used

---

## Phase 3: Money Currency Metadata

### Task 3.1: Enhance `record.format()` for Currency

**Files**: `pkg/parsley/evaluator/record_methods.go`
**Estimated effort**: Medium

Steps:
1. In `format(field)`, check if field type is `money`
2. Check for `currency` metadata key
3. Format value with currency symbol and decimals:
   - USD: `$1,234.56`
   - EUR: `€1.234,56`
   - JPY: `¥1,234` (no decimals)
4. Check for optional `format` metadata for custom patterns
5. Default to USD if no currency specified

Tests:
- `{price: 1999}` with `currency: "USD"` → `"$19.99"`
- `{price: 1999}` with `currency: "EUR"` → `"€19,99"`
- `{price: 5000}` with `currency: "JPY"` → `"¥5,000"`
- No currency metadata → default USD formatting

---

### Task 3.2: Currency Metadata Accessibility

**Files**: `pkg/parsley/evaluator/dsl_schema.go`
**Estimated effort**: Small

Steps:
1. Ensure `currency` metadata is accessible via `schema.meta(field, "currency")`
2. Ensure `format` metadata is accessible via `schema.meta(field, "format")`

Tests:
- `Schema.meta("price", "currency")` returns `"USD"`
- `Schema.meta("price", "format")` returns custom format if set

---

## Phase 4: Documentation

### Task 4.1: Update Schema Manual Page

**Files**: `docs/parsley/manual/builtins/schema.md`
**Estimated effort**: Medium

Steps:
1. Add section for `pattern` constraint with examples
2. Document `id` type alias (mention it equals `ulid`)
3. Document all ID types: `uuid`, `ulid`, `int`, `bigint` with `auto`
4. Add currency metadata section with `format()` examples
5. Update the constraints table

---

### Task 4.2: Update Language Reference

**Files**: `docs/parsley/reference.md`
**Estimated effort**: Small

Steps:
1. Add `pattern` to constraint table in Schema Literals section
2. Add `id` to type table with note that it's alias for `ulid`
3. Ensure all ID types documented

---

### Task 4.3: Update Cheatsheet

**Files**: `docs/parsley/CHEATSHEET.md`
**Estimated effort**: Small

Steps:
1. Add pattern constraint syntax
2. Note: `id` = `ulid` (Parsley's opinionated default)
3. Note: Currency is metadata, not constraint

---

### Task 4.4: Update FEAT-091 Feature Mapping Table

**Files**: `work/specs/FEAT-091.md`
**Estimated effort**: Small

Steps:
1. Verify the feature mapping table (§1.4) is up to date with new features
2. Add any missing entries

---

## Validation Checklist

- [ ] All tests pass: `make test`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] Schema manual page updated
- [ ] Language reference updated
- [ ] Cheatsheet updated
- [ ] FEAT-094 spec status updated to `implemented`
- [ ] work/BACKLOG.md updated with deferrals (if any)

---

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-01-17 | Plan created | ✅ Complete | — |
| | Task 1.1 | ⬜ Not started | — |
| | Task 1.2 | ⬜ Not started | — |
| | Task 1.3 | ⬜ Not started | — |
| | Task 1.4 | ⬜ Not started | — |
| | Task 2.1 | ⬜ Not started | — |
| | Task 2.2 | ⬜ Not started | — |
| | Task 2.3 | ⬜ Not started | — |
| | Task 2.4 | ⬜ Not started | — |
| | Task 2.5 | ⬜ Not started | — |
| | Task 3.1 | ⬜ Not started | — |
| | Task 3.2 | ⬜ Not started | — |
| | Task 4.1 | ⬜ Not started | — |
| | Task 4.2 | ⬜ Not started | — |
| | Task 4.3 | ⬜ Not started | — |
| | Task 4.4 | ⬜ Not started | — |

---

## Deferred Items

Items to add to work/BACKLOG.md after implementation:
- Typed arrays `array(items: string)` — Complexity, Parsley arrays are heterogeneous
- Nested object types — Defer to V2; use `json` type for now
- `serial` type alias — `int(auto)` is clear enough
- Advanced currency formatting (locale-aware) — Consider i18n package later
