---
id: FEAT-094
title: "Schema Enhancements: Pattern, ID Types, and Money Currency"
status: complete
priority: medium
created: 2026-01-17
completed: 2026-01-17
author: "@copilot"
extends: FEAT-091
plan: PLAN-067
---

# FEAT-094: Schema Enhancements

## Summary

Extend `@schema` with additional constraints and type refinements to achieve parity with `@std/schema` and improve the single-source-of-truth model. This specification covers:

1. **String pattern constraint** — Regex validation for strings
2. **ID type clarification** — Explicit `uuid`, `ulid`, `int(auto)` types; `id` as default alias
3. **Money currency metadata** — Display formatting for monetary values

## Relationship to FEAT-091

This spec **extends** FEAT-091 (Record Type). All definitions, principles, and behaviors from FEAT-091 apply. This document specifies only the new features.

## Design Principle Review

Per FEAT-091 §1.4, every schema feature flows to three destinations:

| Feature | Database (DDL) | Validation | Forms (HTML) |
|---------|----------------|------------|--------------|
| `string(pattern:/.../)` | ❌ | Regex match | `pattern` attr (best-effort) |
| `uuid(auto)` | `TEXT PRIMARY KEY` | UUID format | Hidden/readonly |
| `ulid(auto)` | `TEXT PRIMARY KEY` | ULID format | Hidden/readonly |
| `int(auto)` | `INTEGER PRIMARY KEY` | Skip on insert | Hidden/readonly |
| `bigint(auto)` | `BIGINT PRIMARY KEY` | Skip on insert | Hidden/readonly |
| `id` (alias) | Same as `ulid` | ULID format | Hidden/readonly |
| `money \| {currency}` | ❌ | ❌ | Display formatting |

---

## 1. String Pattern Constraint

### 1.1 Syntax

```parsley
@schema User {
    name: string(pattern: /^[A-Za-z\s\-']+$/),
    username: string(min: 3, max: 20, pattern: /^[a-z][a-z0-9_]*$/),
    slug: string(pattern: /^[a-z0-9]+(?:-[a-z0-9]+)*$/)
}
```

**SPEC-PAT-001:** The `pattern` constraint MUST accept a regex literal.

**SPEC-PAT-002:** The regex MUST be compiled at schema parse time. Invalid regex MUST produce a parse error.

**SPEC-PAT-003:** Empty strings MUST pass pattern validation (use `required` or `min: 1` for non-empty).

### 1.2 Validation Behavior

**SPEC-PAT-004:** Validation MUST fail if the string value does not match the pattern.

**SPEC-PAT-005:** The error code MUST be `PATTERN`.

**SPEC-PAT-006:** The error message MUST be `"{title} does not match the required format"`.

### 1.3 Database Mapping

**SPEC-PAT-007:** Pattern constraints MUST NOT generate SQL CHECK constraints.

**Rationale:** SQLite does not support regex in CHECK constraints. PostgreSQL supports it via extensions, but cross-database compatibility is preferred. Pattern validation is server-side only.

### 1.4 Form Mapping

**SPEC-PAT-008:** Pattern constraints SHOULD generate an HTML5 `pattern` attribute on `<input>` elements.

**SPEC-PAT-009:** The regex MUST be converted to JavaScript-compatible syntax when generating the HTML attribute.

**SPEC-PAT-010:** If conversion is not possible (e.g., Go-specific regex features), the `pattern` attribute SHOULD be omitted with a warning.

**Rationale:** HTML5 pattern validation uses JavaScript regex, which differs from Go regex. Best-effort conversion provides progressive enhancement; server-side validation is authoritative.

### 1.5 Implementation

Add to `DSLSchemaField` struct:

```go
type DSLSchemaField struct {
    // ... existing fields ...
    Pattern       *regexp.Regexp // compiled regex pattern
    PatternSource string         // original pattern string (for HTML)
}
```

Add to `record_validation.go`:

```go
func validatePattern(value Object, field *DSLSchemaField, title string) *RecordError {
    if field.Pattern == nil {
        return nil
    }
    str, ok := value.(*String)
    if !ok {
        return nil // Type error handled elsewhere
    }
    // Empty strings pass (use required for non-empty)
    if str.Value == "" {
        return nil
    }
    if !field.Pattern.MatchString(str.Value) {
        return &RecordError{
            Code:    "PATTERN",
            Message: fmt.Sprintf("%s does not match the required format", title),
        }
    }
    return nil
}
```

### 1.6 Test Cases

| Input | Pattern | Expected |
|-------|---------|----------|
| `"Alice"` | `/^[A-Za-z]+$/` | ✅ Valid |
| `"Alice123"` | `/^[A-Za-z]+$/` | ❌ PATTERN error |
| `""` | `/^[A-Za-z]+$/` | ✅ Valid (empty passes) |
| `"test-slug"` | `/^[a-z0-9-]+$/` | ✅ Valid |
| `"Test-Slug"` | `/^[a-z0-9-]+$/` | ❌ PATTERN error |

---

## 2. ID Type Clarification

### 2.1 Design Decision

Rather than `id(format: "uuid")`, Parsley uses **explicit ID types**:

| Type | Format | Length | Sortable | Collision Risk |
|------|--------|--------|----------|----------------|
| `uuid` | UUID v4 | 36 chars | ❌ Random | Very low |
| `ulid` | ULID | 26 chars | ✅ Time-based | Very low |
| `int` | Integer | Variable | ✅ Sequential | None (DB ensures) |
| `bigint` | 64-bit int | Variable | ✅ Sequential | None (DB ensures) |

**SPEC-ID-001:** `uuid`, `ulid`, `int`, and `bigint` MUST be valid field types.

**SPEC-ID-002:** The `id` type MUST be an alias for `ulid`.

**Rationale for `id` = `ulid`:**
- ULIDs are time-sortable (better for database indexing)
- URL-safe (no special characters)
- Don't expose business information (unlike sequential integers)
- Work in distributed systems without coordination

### 2.2 Syntax Examples

```parsley
@schema User {
    // Recommended: explicit types
    id: ulid(auto),                  // ULID primary key (preferred)
    
    // Alternatives
    id: uuid(auto),                  // UUID v4 primary key
    id: int(auto),                   // Auto-increment integer
    id: bigint(auto),                // Auto-increment 64-bit integer
    
    // Alias (same as ulid)
    id: id(auto),                    // Alias for ulid(auto)
}
```

### 2.3 Database Mapping

| Type | SQLite | PostgreSQL |
|------|--------|------------|
| `uuid(auto)` | `TEXT PRIMARY KEY` | `UUID PRIMARY KEY DEFAULT gen_random_uuid()` |
| `ulid(auto)` | `TEXT PRIMARY KEY` | `TEXT PRIMARY KEY` |
| `int(auto)` | `INTEGER PRIMARY KEY` | `SERIAL PRIMARY KEY` |
| `bigint(auto)` | `INTEGER PRIMARY KEY` | `BIGSERIAL PRIMARY KEY` |
| `id(auto)` | Same as `ulid` | Same as `ulid` |

**SPEC-ID-003:** Integer primary keys in SQLite MUST use implicit autoincrement (not `AUTOINCREMENT` keyword).

**Rationale:** SQLite's `INTEGER PRIMARY KEY` is automatically an alias for `rowid` with auto-increment behavior. The explicit `AUTOINCREMENT` keyword adds overhead and is rarely needed.

**SPEC-ID-004:** ULID generation MUST happen at insert time in the server/database layer.

**SPEC-ID-005:** UUID generation MUST happen at insert time in the server/database layer.

### 2.4 Validation Behavior

| Type | Without `auto` | With `auto` |
|------|----------------|-------------|
| `uuid` | Must be valid UUID format | Skipped on insert |
| `ulid` | Must be valid ULID format | Skipped on insert |
| `int` | Must be integer | Skipped on insert |
| `bigint` | Must be integer | Skipped on insert |
| `id` | Same as `ulid` | Same as `ulid` |

**SPEC-ID-006:** When `auto` is NOT specified, the field MUST be validated for correct format.

### 2.5 Form Behavior

**SPEC-ID-007:** Fields with `auto` constraint MUST be rendered as hidden or readonly inputs.

**SPEC-ID-008:** Fields with `auto` constraint MUST NOT be included in form field iterations by default.

### 2.6 Implementation

The `id` type alias is handled at parse time:

```go
// In schema field evaluation
if field.TypeName == "id" {
    field.TypeName = "ulid"  // Expand alias
}
```

### 2.7 Prior Art

| Framework | Integer ID | String ID |
|-----------|------------|-----------|
| Prisma | `Int @id @default(autoincrement())` | `String @id @default(uuid())` |
| Django | `AutoField` | `UUIDField(default=uuid.uuid4)` |
| Rails | `id: :integer` (default) | `id: :uuid` |
| Drizzle | `serial('id')` | `text('id').$defaultFn(ulid)` |

---

## 3. Money Currency Metadata

### 3.1 Design Decision

Currency is **metadata for display**, not a constraint. The database stores numeric values; currency symbols and formatting are presentation concerns.

### 3.2 Syntax

```parsley
@schema Product {
    price: money | {currency: "USD"},
    eurPrice: money | {currency: "EUR", format: "€#,##0.00"},
    jpyPrice: money | {currency: "JPY", format: "¥#,##0"}  // No decimals for Yen
}
```

**SPEC-CUR-001:** Currency MUST be specified as metadata, not a constraint.

**SPEC-CUR-002:** The `currency` metadata key SHOULD contain an ISO 4217 currency code.

**SPEC-CUR-003:** The `format` metadata key MAY contain a format pattern for display.

### 3.3 Database Mapping

**SPEC-CUR-004:** Currency metadata MUST NOT affect database schema generation.

**SPEC-CUR-005:** Money fields MUST be stored as `INTEGER` (cents/minor units) or `DECIMAL(19,4)`.

### 3.4 Validation Behavior

**SPEC-CUR-006:** Currency metadata MUST NOT affect validation.

**Rationale:** Validating that a value "is in USD" doesn't make sense—the value is just a number. Currency is how we *interpret* and *display* that number.

### 3.5 Form/Display Behavior

**SPEC-CUR-007:** Forms and tables MAY use currency metadata for display formatting.

**SPEC-CUR-008:** The `record.format(field)` method SHOULD apply currency formatting when available.

```parsley
let product = Product({price: 1999})  // $19.99 stored as cents

// Display formatting
product.price                         // 1999 (raw value)
product.format("price")               // "$19.99" (formatted with currency)
```

### 3.6 Implementation

Currency is purely metadata—no special handling in validation or database. The `format()` method checks for currency metadata:

```go
func formatRecordCurrency(value Object, field *DSLSchemaField) Object {
    // Get currency from metadata
    currency := "USD"  // default
    if field.Metadata != nil {
        if cur, ok := field.Metadata["currency"]; ok {
            if s, ok := cur.(*String); ok {
                currency = s.Value
            }
        }
    }
    // Format based on currency
    // ... formatting logic ...
}
```

---

## 4. Summary: What's NOT Included

Based on our discussion, these features are explicitly **not** in scope:

| Feature | Reason |
|---------|--------|
| Typed arrays `array(items: string)` | Parsley arrays are heterogeneous; adds complexity |
| Nested object types | Too complex for V1; use `json` type |
| Currency as constraint | Database doesn't know about currencies; purely display |
| `serial` type alias | `int(auto)` is clear enough; avoid redundant aliases |

---

## 5. Test Requirements

### 5.1 Pattern Tests

```parsley
// TEST-PAT-001: Valid pattern match
@schema T1 { name: string(pattern: /^[A-Z][a-z]+$/) }
T1({name: "Alice"}).validate().isValid()  // true

// TEST-PAT-002: Invalid pattern match
T1({name: "alice"}).validate().isValid()  // false
T1({name: "alice"}).validate().errorCode("name")  // "PATTERN"

// TEST-PAT-003: Empty string passes pattern
T1({name: ""}).validate().isValid()  // true (use required for non-empty)

// TEST-PAT-004: Pattern with min constraint
@schema T2 { name: string(min: 1, pattern: /^[A-Z][a-z]+$/) }
T2({name: ""}).validate().isValid()  // false (MIN_LENGTH error)
```

### 5.2 ID Type Tests

```parsley
// TEST-ID-001: id is alias for ulid
@schema T3 { id: id(auto) }
T3.fields.id.type  // "ulid"

// TEST-ID-002: uuid validation when not auto
@schema T4 { id: uuid }
T4({id: "not-a-uuid"}).validate().isValid()  // false

// TEST-ID-003: ulid validation when not auto
@schema T5 { id: ulid }
T5({id: "not-a-ulid"}).validate().isValid()  // false

// TEST-ID-004: int(auto) skips validation
@schema T6 { id: int(auto), name: string }
T6({name: "Alice"}).validate().isValid()  // true
```

### 5.3 Currency Metadata Tests

```parsley
// TEST-CUR-001: Currency in metadata
@schema T7 { price: money | {currency: "USD"} }
T7.meta("price", "currency")  // "USD"

// TEST-CUR-002: Format with currency
let p = T7({price: 1999})
p.format("price")  // "$19.99"
```

---

## 6. Implementation Phases

### Phase 1: Pattern Constraint
1. Add `Pattern` and `PatternSource` to `DSLSchemaField`
2. Parse regex in type options
3. Add `validatePattern()` in record validation
4. Add tests
5. Update form binding to emit `pattern` attribute

### Phase 2: ID Type Clarification
1. Add `id` → `ulid` alias expansion in parser
2. Ensure `uuid`, `ulid`, `int`, `bigint` all work with `auto`
3. Add UUID/ULID format validation when `auto` is not set
4. Update `createTable` SQL generation for each type
5. Add tests

### Phase 3: Money Currency Metadata
1. Enhance `record.format()` to check currency metadata
2. Add currency formatting logic (symbol, decimal places)
3. Add tests
4. Document in schema.md

---

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-01-17 | - | Initial specification |
