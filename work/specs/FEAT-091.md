---
id: FEAT-091
title: "Record Type"
status: draft
priority: high
created: 2026-01-15
author: "@copilot"
supersedes: FEAT-002
design: DESIGN-record-type-v3.md
---

# FEAT-091: Record Type for Parsley

## Summary

Implement the **Record type** — a typed wrapper around data that carries its schema and validation state. The primary motivation is form handling, but records have broader applications including API validation, configuration, and bulk data import.

## User Story

As a Parsley developer, I want to define schemas for my data and have automatic validation, type casting, and form binding so that I can build forms with less boilerplate and catch errors early.

## Acceptance Criteria

- [ ] Schema definition with constraints and metadata (pipe syntax)
- [ ] Record creation from schema with defaults and type casting
- [ ] Validation with standard error codes and messages
- [ ] Form binding with `@field`, `<Label>`, `<Error>`, `<Meta>`, `<Select>`
- [ ] Table validation with bulk error reporting
- [ ] Database integration with auto-validated query results
- [ ] All tests pass
- [ ] Documentation updated

---

## Specification

This specification defines the **Record type** for Parsley with sufficient precision to:

1. **Guide Implementation:** Unambiguous behavior for all features
2. **Generate Documentation:** Accurate reference material for users
3. **Enable Testing:** Comprehensive test case derivation
4. **Verify Completeness:** Checklist-based implementation verification

---

## Table of Contents

1. [Definitions](#1-definitions)
2. [Type System](#2-type-system)
3. [Schema Specification](#3-schema-specification)
4. [Record Specification](#4-record-specification)
5. [Table Specification](#5-table-specification)
6. [Validation Specification](#6-validation-specification)
7. [Form Binding Specification](#7-form-binding-specification)
8. [Database Integration Specification](#8-database-integration-specification)
9. [Error Catalog](#9-error-catalog)
10. [Implementation Phases](#10-implementation-phases)
11. [Test Requirements](#11-test-requirements)
12. [Verification Checklist](#12-verification-checklist)

---

## 1. Definitions

### 1.1 Terminology

| Term | Definition |
|------|------------|
| **Schema** | A structural definition specifying field names, types, constraints, and metadata |
| **Record** | An immutable data container bound to a schema, carrying data and validation state |
| **Table** | An ordered collection of Records sharing the same schema |
| **Field** | A named element within a schema with a type and optional constraints |
| **Constraint** | A validation rule applied to a field (e.g., `required`, `min`, `max`) |
| **Metadata** | Display and behavior hints attached to a field (e.g., `title`, `placeholder`) |
| **Validation** | The process of checking data against schema constraints |
| **Error** | A validation failure with a code and human-readable message |

### 1.2 Notation

- `→` denotes "produces" or "returns"
- `⊆` denotes "is a subset of"
- `∀` denotes "for all"
- `∃` denotes "there exists"
- `|` in code denotes the metadata pipe operator
- `@` prefix denotes Parsley directives

### 1.3 Conformance

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD", "SHOULD NOT", "RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be interpreted as described in RFC 2119.

---

## 2. Type System

### 2.1 Record Type Identity

A Record:
- **IS-A** Dictionary for data access purposes
- **HAS-A** Schema reference
- **HAS-A** Validation state (validated: boolean)
- **HAS-A** Error collection

### 2.2 Type Relationships

```
Dictionary ←──── Record
                   │
                   ├── schema: Schema
                   ├── validated: Boolean
                   └── errors: Dictionary<String, Error>

Table ←──── TypedTable
               │
               └── schema: Schema
```

### 2.3 Immutability Invariant

**INV-001:** Records MUST be immutable. All mutation operations MUST return new Record instances.

**INV-002:** The original Record MUST remain unchanged after any mutation operation.

### 2.4 Dictionary Compatibility

**SPEC-DC-001:** A Record MUST be usable in any context where a Dictionary is expected.

**SPEC-DC-002:** Spread syntax (`{...record}`) MUST expand only data fields (not metadata).

**SPEC-DC-003:** JSON encoding of a Record MUST encode only data fields.

**SPEC-DC-004:** Passing a Record to a function expecting a Dictionary MUST succeed.

---

## 3. Schema Specification

### 3.1 Schema Declaration Syntax

#### 3.1.1 Named Declaration (Canonical)

```
@schema <Identifier> {
    <field-list>
}
```

**SPEC-SCH-001:** Named schema declarations MUST create a binding in the current scope.

#### 3.1.2 Assignment Form

```
let <Identifier> = @schema {
    <field-list>
}
```

**SPEC-SCH-002:** Assignment form MUST be semantically equivalent to named declaration.

### 3.2 Field Definition Syntax

```
<field-name>: <type>[(<constraints>)] [| {<metadata>}]
```

#### 3.2.1 Field Name

**SPEC-FLD-001:** Field names MUST be valid Parsley identifiers.

**SPEC-FLD-002:** Field names MUST be unique within a schema.

#### 3.2.2 Type Specification

**SPEC-TYP-001:** Both bare types (`int`) and called types (`int()`) MUST be valid.

**SPEC-TYP-002:** The following base types MUST be supported:

| Type | Go Equivalent | Description |
|------|---------------|-------------|
| `int` | `int64` | Integer |
| `bigint` | `int64` | 64-bit integer (same as int) |
| `float` | `float64` | Floating-point number |
| `string` | `string` | Text string |
| `text` | `string` | Alias for string |
| `bool` | `bool` | Boolean |
| `datetime` | `time.Time` | Date and time |
| `date` | `time.Time` | Date only |
| `time` | `time.Time` | Time only |
| `money` | `decimal.Decimal` | Monetary value |
| `decimal` | `decimal.Decimal` | Arbitrary precision decimal |
| `json` | `interface{}` | JSON value |

**SPEC-TYP-003:** The following validated string types MUST be supported:

| Type | Validation | Description |
|------|------------|-------------|
| `email` | RFC 5322 format | Email address |
| `url` | http/https URL | Web URL |
| `phone` | Digits, +, -, spaces, parens | Phone number |
| `slug` | `^[a-z0-9-]+$` | URL-safe slug |
| `uuid` | UUID format | UUID string |
| `ulid` | ULID format | ULID string |

#### 3.2.3 Enum Type

```
enum(<value1>, <value2>, ...)
```

**SPEC-ENUM-001:** Enum values MUST be string literals.

**SPEC-ENUM-002:** Validation MUST fail if value is not in the allowed set.

**SPEC-ENUM-003:** Empty string MUST NOT be a valid enum value unless explicitly listed.

### 3.3 Constraint Specification

**SPEC-CON-001:** The following constraints MUST be supported:

| Constraint | Applies To | Type | Description |
|------------|------------|------|-------------|
| `required` | All | Boolean | Field must have a non-null value |
| `min` | `string` | Integer | Minimum string length |
| `max` | `string` | Integer | Maximum string length |
| `min` | `int`, `float` | Number | Minimum numeric value |
| `max` | `int`, `float` | Number | Maximum numeric value |
| `default` | All | Value | Default value when missing or null |
| `unique` | All | Boolean | Database uniqueness (DB phase only) |

**SPEC-CON-002:** Multiple constraints MUST be comma-separated within parentheses.

**SPEC-CON-003:** Constraint order MUST NOT affect semantics.

### 3.4 Metadata Specification

#### 3.4.1 Pipe Syntax

```
<type>(<constraints>) | {<key>: <value>, ...}
```

**SPEC-META-001:** The pipe (`|`) MUST separate type/constraints from metadata.

**SPEC-META-002:** Metadata MUST be a dictionary literal.

**SPEC-META-003:** Metadata MUST accept any key-value pairs (open dictionary).

#### 3.4.2 Core Metadata Keys (V1)

**SPEC-META-004:** The following metadata keys MUST have defined semantics:

| Key | Type | Purpose | Consumers |
|-----|------|---------|-----------|
| `title` | String | Human-readable field label | Forms, tables, errors |
| `placeholder` | String | Input placeholder text | Forms |
| `format` | String | Display format hint | Tables, formatting |
| `hidden` | Boolean | Exclude from default display | Tables, forms |
| `help` | String | Descriptive help text | Forms |

#### 3.4.3 Title Resolution

**SPEC-META-005:** Title resolution MUST follow this precedence:
1. Explicit `title` in metadata
2. Field name converted to title case (e.g., `firstName` → `First Name`)

### 3.5 Schema Methods

**SPEC-SCH-MTD-001:** Schemas MUST implement the following methods:

| Method | Signature | Returns | Description |
|--------|-----------|---------|-------------|
| `title` | `(field: String) → String` | String | Field title per SPEC-META-005 |
| `placeholder` | `(field: String) → String?` | String or null | Field placeholder |
| `meta` | `(field: String, key: String) → Any` | Any or null | Any metadata value |
| `fields` | `() → Array<String>` | Array | All field names |
| `visibleFields` | `() → Array<String>` | Array | Fields where `hidden != true` |
| `enumValues` | `(field: String) → Array<String>` | Array | Enum values (empty if not enum) |

---

## 4. Record Specification

### 4.1 Record Creation

#### 4.1.1 Schema Call Syntax

```
<Schema>({<field>: <value>, ...})
```

**SPEC-REC-001:** Calling a schema with a dictionary MUST return a Record.

**SPEC-REC-002:** The returned Record MUST have `validated = false`.

**SPEC-REC-003:** Default values MUST be applied during creation.

**SPEC-REC-004:** Fields not in the schema MUST be silently filtered out.

**SPEC-REC-005:** Type casting MUST be applied per field types.

#### 4.1.2 `.as()` Syntax

```
<dictionary>.as(<Schema>)
```

**SPEC-REC-006:** `.as(Schema)` on a dictionary MUST return a Record.

**SPEC-REC-007:** Behavior MUST be identical to `Schema(dictionary)`.

### 4.2 Default Value Application

**SPEC-DEF-001:** Default values MUST be applied when:
- Field is missing from input dictionary
- Field value is `null`

**SPEC-DEF-002:** Default values MUST NOT be applied when:
- Field has any non-null value (including empty string, zero, false)

**SPEC-DEF-003:** Default application MUST occur at creation time, not validation time.

### 4.3 Type Casting

**SPEC-CAST-001:** The following casts MUST be applied automatically:

| From | To | Rule |
|------|----|------|
| `"123"` | `int` | Parse as integer |
| `"3.14"` | `float` | Parse as float |
| `"true"`, `"false"` | `bool` | Parse as boolean |
| `"TRUE"`, `"FALSE"` | `bool` | Parse as boolean (case-insensitive) |
| `"1"`, `"0"` | `bool` | Parse as boolean |
| `1`, `0` | `bool` | Convert to boolean |

**SPEC-CAST-002:** Failed casts MUST produce a `TYPE` error during validation.

### 4.4 Field Access

**SPEC-ACC-001:** Data fields MUST be accessible via dot notation: `record.fieldName`

**SPEC-ACC-002:** Accessing a field not in schema MUST return `null`.

**SPEC-ACC-003:** Metadata MUST NOT be accessible via dot notation.

### 4.5 Record Methods

**SPEC-REC-MTD-001:** Records MUST implement the following methods:

| Method | Signature | Returns | Description |
|--------|-----------|---------|-------------|
| `validate` | `() → Record` | Record | Returns new Record with validation performed |
| `update` | `(dict: Dictionary) → Record` | Record | Merges fields and auto-revalidates |
| `errors` | `() → Dictionary` | `{field: {code, message}}` | All field errors |
| `error` | `(field: String) → String?` | String or null | Error message for field |
| `errorCode` | `(field: String) → String?` | String or null | Error code for field |
| `errorList` | `() → Array` | `[{field, code, message}]` | Errors as array |
| `isValid` | `() → Boolean` | Boolean | True if validated AND no errors |
| `hasError` | `(field: String) → Boolean` | Boolean | True if field has error |
| `schema` | `() → Schema` | Schema | The bound schema |
| `data` | `() → Dictionary` | Dictionary | Plain dictionary of all data |
| `keys` | `() → Array<String>` | Array | Field names |
| `withError` | `(field, msg) → Record` | Record | Add custom error |
| `withError` | `(field, code, msg) → Record` | Record | Add custom error with code |
| `title` | `(field: String) → String` | String | Shorthand for `schema().title(field)` |
| `placeholder` | `(field: String) → String?` | String or null | Shorthand for `meta(field, "placeholder")` |
| `meta` | `(field, key: String) → Any` | Any | Shorthand for `schema().meta(field, key)` |
| `enumValues` | `(field: String) → Array` | Array | Enum values for field |
| `format` | `(field: String) → String` | String | Value formatted per schema hints |

### 4.6 `isValid()` Semantics

**SPEC-VALID-001:** `isValid()` MUST return `false` if:
- Record has not been validated (`validated = false`)
- Record has any errors (`len(errors) > 0`)

**SPEC-VALID-002:** `isValid()` MUST return `true` if:
- Record has been validated (`validated = true`) AND
- Record has no errors (`len(errors) = 0`)

### 4.7 `update()` Semantics

**SPEC-UPD-001:** `update(dict)` MUST:
1. Create a new Record with merged data
2. Automatically validate the new Record
3. Return the new Record

**SPEC-UPD-002:** The original Record MUST remain unchanged.

**SPEC-UPD-003:** Validation MUST occur even if the original was unvalidated.

### 4.8 `withError()` Semantics

**SPEC-ERR-001:** `withError(field, msg)` MUST:
1. Create a new Record with the error added
2. NOT trigger revalidation
3. Use error code `"CUSTOM"` when code not provided

**SPEC-ERR-002:** `withError(field, code, msg)` MUST use the provided code.

**SPEC-ERR-003:** Adding an error to a field with existing error MUST replace the error.

### 4.9 `format()` Semantics

**SPEC-FMT-001:** The following format strings MUST be supported:

| Format | Example Input | Example Output |
|--------|---------------|----------------|
| `"date"` | `@2025-01-15` | "Jan 15, 2025" |
| `"datetime"` | `@2025-01-15T14:30:00Z` | "Jan 15, 2025 2:30 PM" |
| `"currency"` | `52000` | "$52,000.00" |
| `"percent"` | `0.15` | "15%" |
| `"number"` | `1234567` | "1,234,567" |

**SPEC-FMT-002:** If no format is specified, MUST return string representation.

**SPEC-FMT-003:** If format is unrecognized, MUST return string representation.

---

## 5. Table Specification

### 5.1 Table Creation

#### 5.1.1 Schema Call with Array

```
<Schema>([{...}, {...}, ...])
```

**SPEC-TBL-001:** Calling a schema with an array MUST return a Table.

**SPEC-TBL-002:** Each element MUST become a Record in the Table.

**SPEC-TBL-003:** Records MUST have `validated = false`.

#### 5.1.2 Table Literal

```
@table(<Schema>) [
    {...},
    {...}
]
```

**SPEC-TBL-004:** Table literal syntax MUST bind schema at compile time.

#### 5.1.3 `.as()` Syntax

```
table(<array>).as(<Schema>)
```

**SPEC-TBL-005:** `.as(Schema)` on a table MUST return a typed Table.

### 5.2 Table Methods

**SPEC-TBL-MTD-001:** Typed Tables MUST implement the following methods:

| Method | Signature | Returns | Description |
|--------|-----------|---------|-------------|
| `validate` | `() → Table` | Table | Bulk validate all rows |
| `isValid` | `() → Boolean` | Boolean | True if ALL rows valid |
| `errors` | `() → Array` | `[{row, field, code, message}]` | All errors with row index |
| `validRows` | `() → Table` | Table | Rows that passed validation |
| `invalidRows` | `() → Table` | Table | Rows that failed validation |
| `schema` | `() → Schema` | Schema | The bound schema |

### 5.3 Table Error Shape

**SPEC-TBL-ERR-001:** Table errors MUST include row index:

```javascript
[
    {row: 0, field: "email", code: "FORMAT", message: "Invalid email format"},
    {row: 2, field: "name", code: "REQUIRED", message: "Name is required"}
]
```

**SPEC-TBL-ERR-002:** Row indices MUST be zero-based.

### 5.4 Row Access

**SPEC-TBL-ROW-001:** `table[n]` MUST return the Record at index n.

**SPEC-TBL-ROW-002:** Individual rows MUST have the standard Record error shape.

---

## 6. Validation Specification

### 6.1 Validation Process

**SPEC-VAL-001:** `record.validate()` MUST:
1. Check all constraint rules against data
2. Populate errors for failed constraints
3. Set `validated = true`
4. Return a new Record

**SPEC-VAL-002:** Original Record MUST remain unchanged.

### 6.2 Constraint Validation Rules

#### 6.2.1 Required

**SPEC-VAL-REQ-001:** A field fails `required` if:
- Field is missing from data
- Field value is `null`
- Field value is undefined

**SPEC-VAL-REQ-002:** Empty string (`""`) MUST NOT fail `required`.

**SPEC-VAL-REQ-003:** Zero (`0`) MUST NOT fail `required`.

**SPEC-VAL-REQ-004:** False (`false`) MUST NOT fail `required`.

#### 6.2.2 String Length

**SPEC-VAL-LEN-001:** `min` on string MUST fail if `len(value) < min`.

**SPEC-VAL-LEN-002:** `max` on string MUST fail if `len(value) > max`.

**SPEC-VAL-LEN-003:** Length constraints MUST be checked on non-null values only.

#### 6.2.3 Numeric Range

**SPEC-VAL-RNG-001:** `min` on number MUST fail if `value < min`.

**SPEC-VAL-RNG-002:** `max` on number MUST fail if `value > max`.

**SPEC-VAL-RNG-003:** Range constraints MUST be checked on non-null values only.

#### 6.2.4 Format Validation

**SPEC-VAL-FMT-001:** Validated types (`email`, `url`, etc.) MUST check format.

**SPEC-VAL-FMT-002:** Format MUST be checked on non-null, non-empty values only.

#### 6.2.5 Enum Validation

**SPEC-VAL-ENUM-001:** Value MUST be in the declared enum set.

**SPEC-VAL-ENUM-002:** Enum check MUST be case-sensitive.

### 6.3 Validation Order

**SPEC-VAL-ORD-001:** Validation MUST check in this order:
1. Type casting (if needed)
2. Required check
3. Type check
4. Format check (for validated types)
5. Constraint checks (min, max, enum)

**SPEC-VAL-ORD-002:** Validation MUST stop at first failure for each field.

### 6.4 Error Shape

**SPEC-VAL-ERR-001:** Record errors MUST have this shape:

```javascript
{
    "fieldName": {
        "code": "ERROR_CODE",
        "message": "Human-readable message"
    }
}
```

**SPEC-VAL-ERR-002:** Only one error per field MUST be stored.

---

## 7. Form Binding Specification

### 7.1 Form Context

```
<form @record={<expression>}>
    ...
</form>
```

**SPEC-FORM-001:** `@record` MUST establish a form context in the AST.

**SPEC-FORM-002:** The expression MUST evaluate to a Record.

**SPEC-FORM-003:** Form binding elements MUST only work within a form context.

### 7.2 Input Binding

```
<input @field="<fieldName>"/>
```

#### 7.2.1 Rewriting Rules

**SPEC-INPUT-001:** `<input @field="x"/>` MUST be rewritten to:

```html
<input name="x"
       value={record.x}
       [required]
       [minlength="n"]
       [maxlength="n"]
       [min="n"]
       [max="n"]
       [type="..."]
       aria-invalid={record.hasError("x")}
       aria-describedby={record.hasError("x") ? "x-error" : null}
       [aria-required="true"]/>
```

**SPEC-INPUT-002:** HTML validation attributes MUST be derived from schema constraints:

| Schema Constraint | HTML Attribute |
|-------------------|----------------|
| `required` | `required` |
| `min` (string) | `minlength` |
| `max` (string) | `maxlength` |
| `min` (number) | `min` |
| `max` (number) | `max` |

**SPEC-INPUT-003:** `type` attribute MUST be derived from schema type:

| Schema Type | HTML Type |
|-------------|-----------|
| `email` | `email` |
| `url` | `url` |
| `phone` | `tel` |
| `int` | `number` |
| `date` | `date` |
| `datetime` | `datetime-local` |
| `time` | `time` |

**SPEC-INPUT-004:** Explicit `type` attribute in source MUST override derived type.

#### 7.2.2 Checkbox Binding

**SPEC-INPUT-CHK-001:** `<input @field="x" type="checkbox"/>` MUST:
- Bind `checked` attribute to boolean value: `checked={record.x}`
- Include standard `name` and ARIA attributes

**SPEC-INPUT-CHK-002:** The field SHOULD be of type `bool`.

#### 7.2.3 Radio Button Binding

**SPEC-INPUT-RAD-001:** `<input @field="x" type="radio" value="v"/>` MUST:
- Bind `checked` to equality check: `checked={record.x == "v"}`
- Include standard `name` and ARIA attributes

**SPEC-INPUT-RAD-002:** The field SHOULD be of type `enum`.

### 7.3 ARIA Accessibility

**SPEC-ARIA-001:** All inputs MUST include:

| Attribute | Value | Condition |
|-----------|-------|-----------|
| `aria-invalid` | `true` or `false` | Always present |
| `aria-describedby` | `"{field}-error"` | When error exists |
| `aria-required` | `true` | When field is required |

**SPEC-ARIA-002:** `aria-invalid` MUST be explicitly `"false"` (not omitted) when valid.

### 7.4 Label Component

#### 7.4.1 Self-Closing Form

```
<Label @field="x"/>
```

**SPEC-LABEL-001:** Self-closing `<Label>` MUST produce:

```html
<label for="x">{record.title("x")}</label>
```

#### 7.4.2 Tag-Pair Form

```
<Label @field="x">
    {children}
</Label>
```

**SPEC-LABEL-002:** Tag-pair `<Label>` MUST produce:

```html
<label>
    {record.title("x")}
    {children}
</label>
```

**SPEC-LABEL-003:** Tag-pair form MUST NOT include `for` attribute.

#### 7.4.3 Tag Override

```
<Label @field="x" @tag="span"/>
```

**SPEC-LABEL-004:** `@tag` prop MUST change the output element type.

**SPEC-LABEL-005:** Default tag MUST be `label`.

### 7.5 Error Component

```
<Error @field="x"/>
```

**SPEC-ERROR-001:** `<Error>` MUST produce when error exists:

```html
<span id="x-error" class="error" role="alert">
    {record.error("x")}
</span>
```

**SPEC-ERROR-002:** `<Error>` MUST produce nothing when no error exists.

**SPEC-ERROR-003:** `@tag` prop MUST change the output element type.

**SPEC-ERROR-004:** Default tag MUST be `span`.

**SPEC-ERROR-005:** `id` MUST be `"{field}-error"` to match `aria-describedby`.

**SPEC-ERROR-006:** `role="alert"` MUST always be present.

### 7.6 Meta Component

```
<Meta @field="x" @key="help"/>
```

**SPEC-META-CMP-001:** `<Meta>` MUST produce when metadata exists:

```html
<span>{record.meta("x", "help")}</span>
```

**SPEC-META-CMP-002:** `<Meta>` MUST produce nothing when metadata doesn't exist.

**SPEC-META-CMP-003:** `@tag` prop MUST change the output element type.

**SPEC-META-CMP-004:** Default tag MUST be `span`.

**SPEC-META-CMP-005:** `@key` MUST be required.

### 7.7 Select Component

```
<Select @field="x"/>
```

**SPEC-SELECT-001:** `<Select>` MUST produce:

```html
<select name="x"
        aria-invalid={record.hasError("x")}
        aria-describedby={record.hasError("x") ? "x-error" : null}>
    <option value="">{record.placeholder("x")}</option>
    {for each value in record.enumValues("x"):}
        <option value="{value}" selected={record.x == value}>{value}</option>
    {end for}
</select>
```

**SPEC-SELECT-002:** If `placeholder` prop is provided, it MUST override schema placeholder.

**SPEC-SELECT-003:** If no placeholder exists, empty option MUST still be present with empty text.

**SPEC-SELECT-004:** Field SHOULD be of type `enum`.

---

## 8. Database Integration Specification

### 8.1 Query Return Types

#### 8.1.1 Auto-Detect Mode

**SPEC-DB-001:** `@query(Table ?-> *)` MUST return a Record.

**SPEC-DB-002:** `@query(Table ??-> *)` MUST return a Table of Records.

**SPEC-DB-003:** `@query(Table ?-> a, b)` MUST return:
- Record if `{a, b} ⊆ schema.fields()`
- Dictionary otherwise

**SPEC-DB-004:** `@query(Table ??-> a, b)` MUST return:
- Table of Records if `{a, b} ⊆ schema.fields()`
- Table of Dictionaries otherwise

#### 8.1.2 Explicit Mode

**SPEC-DB-005:** `@query(Table ?!-> a, b)` MUST:
- Return Record if `{a, b} ⊆ schema.fields()`
- Error if any column not in schema

**SPEC-DB-006:** `@query(Table ??!-> a, b)` MUST:
- Return Table of Records if `{a, b} ⊆ schema.fields()`
- Error if any column not in schema

### 8.2 Auto-Validation on Query

**SPEC-DB-VAL-001:** Records from queries MUST have `validated = true`.

**SPEC-DB-VAL-002:** Records from queries MUST have `errors = {}` (empty).

**SPEC-DB-VAL-003:** Rationale: Data from database is trusted; it passed validation on insert.

### 8.3 Partial Record Validation

**SPEC-DB-PART-001:** When validating partial Records (subset of fields):
- `required` MUST NOT be enforced on missing fields
- Constraints on present fields MUST be enforced
- Metadata MUST be available for present fields

### 8.4 Relations

**SPEC-DB-REL-001:** Eager-loaded relations (`with`) MUST return plain Dictionaries.

**SPEC-DB-REL-002:** Relations MUST NOT be validated as part of parent Record.

**SPEC-DB-REL-003:** Explicit `.as(Schema)` MAY be used to convert relation to Record.

### 8.5 Insert Validation

**SPEC-DB-INS-001:** `@insert` MUST validate Records before insertion.

**SPEC-DB-INS-002:** Validation MUST occur even if Record was previously validated.

**SPEC-DB-INS-003:** Batch inserts MUST validate all rows.

---

## 9. Error Catalog

### 9.1 Validation Error Codes

| Code | Trigger | Message Template |
|------|---------|------------------|
| `REQUIRED` | Required field missing or null | `"{title} is required"` |
| `TYPE` | Type mismatch or cast failure | `"{title} must be a {type}"` |
| `FORMAT` | Invalid format for validated type | `"{title} is not a valid {type}"` |
| `ENUM` | Value not in enum set | `"{title} must be one of: {values}"` |
| `MIN_LENGTH` | String shorter than min | `"{title} must be at least {min} characters"` |
| `MAX_LENGTH` | String longer than max | `"{title} must be at most {max} characters"` |
| `MIN_VALUE` | Number less than min | `"{title} must be at least {min}"` |
| `MAX_VALUE` | Number greater than max | `"{title} must be at most {max}"` |
| `CUSTOM` | Added via `withError()` without code | User-provided message |

### 9.2 Message Generation

**SPEC-MSG-001:** Messages MUST use field title (per SPEC-META-005).

**SPEC-MSG-002:** Messages MUST start with capital letter.

**SPEC-MSG-003:** Messages MUST NOT end with period.

---

## 10. Implementation Phases

### Phase 1: Core Record Type

**Scope:** Record creation, validation, error handling

**Deliverables:**
- [ ] Record struct with schema, data, errors, validated flag
- [ ] `Schema({...})` → Record (unvalidated, with defaults)
- [ ] `record.validate()` → Record (validated)
- [ ] `record.update({...})` → Record (auto-revalidated)
- [ ] `record.isValid()` method
- [ ] `record.errors()` method
- [ ] `record.error(field)` method
- [ ] `record.errorCode(field)` method
- [ ] `record.errorList()` method
- [ ] `record.hasError(field)` method
- [ ] `record.withError(field, msg)` method
- [ ] `record.withError(field, code, msg)` method
- [ ] `record.data()` method
- [ ] `record.keys()` method
- [ ] `record.schema()` method
- [ ] Error shape: `{field: {code, message}}`
- [ ] Validation: required, min/max length, min/max value
- [ ] Validation: type casting
- [ ] Validation: format (email, url, phone, slug, uuid, ulid)
- [ ] Validation: enum
- [ ] Default value application
- [ ] Field filtering (whitelist)
- [ ] Dictionary compatibility (spread, JSON, functions)

### Phase 2: Schema Metadata

**Scope:** Metadata parsing and access

**Deliverables:**
- [ ] Pipe syntax parsing in schema definition
- [ ] `schema.title(field)` method
- [ ] `schema.placeholder(field)` method
- [ ] `schema.meta(field, key)` method
- [ ] `schema.fields()` method
- [ ] `schema.visibleFields()` method
- [ ] `schema.enumValues(field)` method
- [ ] Title case fallback for missing title
- [ ] `record.format(field)` method
- [ ] Format: date, datetime, currency, percent, number

### Phase 3: Table Integration

**Scope:** Typed tables with bulk validation

**Deliverables:**
- [ ] `Schema([...])` → Table
- [ ] `@table(Schema) [...]` literal
- [ ] `{...}.as(Schema)` → Record
- [ ] `table(data).as(Schema)` → Table
- [ ] `table.validate()` method
- [ ] `table.isValid()` method
- [ ] `table.errors()` with row indices
- [ ] `table.validRows()` method
- [ ] `table.invalidRows()` method
- [ ] `table.schema()` method
- [ ] Row access: `table[n]` → Record

### Phase 4: Form Binding

**Scope:** HTML form generation and binding

**Deliverables:**
- [ ] `<form @record={...}>` context
- [ ] `<input @field="..."/>` rewriting
- [ ] HTML validation attributes from schema
- [ ] HTML type from schema type
- [ ] Checkbox binding (boolean → checked)
- [ ] Radio binding (enum + value → checked)
- [ ] ARIA attributes: aria-invalid, aria-describedby, aria-required
- [ ] `<Label @field="..."/>` self-closing form
- [ ] `<Label @field="...">...</Label>` tag-pair form
- [ ] `<Label @tag="..."/>` tag override
- [ ] `<Error @field="..."/>` component
- [ ] `<Error @tag="..."/>` tag override
- [ ] `<Meta @field="..." @key="..."/>` component
- [ ] `<Meta @tag="..."/>` tag override
- [ ] `<Select @field="..."/>` component
- [ ] `<Select placeholder="..."/>` custom placeholder
- [ ] `record.title(field)` shorthand
- [ ] `record.placeholder(field)` shorthand
- [ ] `record.meta(field, key)` shorthand
- [ ] `record.enumValues(field)` shorthand

### Phase 5: Database Integration

**Scope:** Query return types and validation

**Deliverables:**
- [ ] Queries return Records when table has schema
- [ ] Auto-validation on query return
- [ ] Projection auto-detect (Record if columns ⊆ schema)
- [ ] `?!->` explicit Record terminal
- [ ] `??!->` explicit Table of Records terminal
- [ ] Error on non-schema column with explicit terminal
- [ ] Batch insert validation
- [ ] Partial record validation (missing fields)
- [ ] Relations return Dictionaries (not Records)

---

## 11. Test Requirements

### 11.1 Unit Test Categories

Each specification requirement (SPEC-*) MUST have corresponding tests.

#### 11.1.1 Record Creation Tests

```
TEST-REC-001: Schema call with dictionary creates Record
TEST-REC-002: Created Record has validated=false
TEST-REC-003: Default values applied on creation
TEST-REC-004: Unknown fields filtered out
TEST-REC-005: Type casting applied during creation
TEST-REC-006: .as() method creates Record
TEST-REC-007: .as() equivalent to Schema() call
```

#### 11.1.2 Validation Tests

```
TEST-VAL-001: validate() returns new Record
TEST-VAL-002: validate() sets validated=true
TEST-VAL-003: Original Record unchanged after validate()
TEST-VAL-004: Required fails for null
TEST-VAL-005: Required fails for missing
TEST-VAL-006: Required passes for empty string
TEST-VAL-007: Required passes for zero
TEST-VAL-008: Required passes for false
TEST-VAL-009: min length fails for short string
TEST-VAL-010: max length fails for long string
TEST-VAL-011: min value fails for small number
TEST-VAL-012: max value fails for large number
TEST-VAL-013: Enum fails for invalid value
TEST-VAL-014: Enum passes for valid value
TEST-VAL-015: Email format validation
TEST-VAL-016: URL format validation
TEST-VAL-017: Phone format validation
TEST-VAL-018: Slug format validation
TEST-VAL-019: Type cast string to int
TEST-VAL-020: Type cast string to bool
TEST-VAL-021: Type cast failure produces TYPE error
```

#### 11.1.3 Error Handling Tests

```
TEST-ERR-001: errors() returns correct shape
TEST-ERR-002: error(field) returns message
TEST-ERR-003: error(field) returns null when no error
TEST-ERR-004: errorCode(field) returns code
TEST-ERR-005: errorList() returns array format
TEST-ERR-006: withError() adds error
TEST-ERR-007: withError() does not revalidate
TEST-ERR-008: withError() with code uses provided code
TEST-ERR-009: withError() without code uses CUSTOM
```

#### 11.1.4 Method Tests

```
TEST-MTD-001: isValid() false when not validated
TEST-MTD-002: isValid() false when errors exist
TEST-MTD-003: isValid() true when validated and no errors
TEST-MTD-004: update() merges data
TEST-MTD-005: update() auto-revalidates
TEST-MTD-006: update() returns new Record
TEST-MTD-007: data() returns plain dictionary
TEST-MTD-008: keys() returns field names
TEST-MTD-009: schema() returns bound schema
TEST-MTD-010: hasError() returns boolean
TEST-MTD-011: title() shorthand works
TEST-MTD-012: placeholder() shorthand works
TEST-MTD-013: meta() shorthand works
TEST-MTD-014: enumValues() returns enum values
TEST-MTD-015: enumValues() returns empty for non-enum
TEST-MTD-016: format() with date format
TEST-MTD-017: format() with currency format
```

#### 11.1.5 Dictionary Compatibility Tests

```
TEST-DICT-001: Spread syntax expands data only
TEST-DICT-002: JSON encoding encodes data only
TEST-DICT-003: Function accepting dict accepts Record
```

#### 11.1.6 Table Tests

```
TEST-TBL-001: Schema with array creates Table
TEST-TBL-002: Table rows are Records
TEST-TBL-003: table.validate() validates all rows
TEST-TBL-004: table.isValid() returns false if any invalid
TEST-TBL-005: table.errors() includes row index
TEST-TBL-006: table.validRows() returns valid only
TEST-TBL-007: table.invalidRows() returns invalid only
TEST-TBL-008: table[n] returns Record
```

#### 11.1.7 Form Binding Tests

```
TEST-FORM-001: @record establishes context
TEST-FORM-002: @field binds value
TEST-FORM-003: @field adds validation attributes
TEST-FORM-004: @field adds ARIA attributes
TEST-FORM-005: aria-invalid explicitly false when valid
TEST-FORM-006: Checkbox binds checked to boolean
TEST-FORM-007: Radio binds checked to equality
TEST-FORM-008: Label self-closing generates for attribute
TEST-FORM-009: Label tag-pair wraps children
TEST-FORM-010: Label @tag changes element
TEST-FORM-011: Error renders when error exists
TEST-FORM-012: Error renders nothing when no error
TEST-FORM-013: Error @tag changes element
TEST-FORM-014: Error has role="alert"
TEST-FORM-015: Meta renders when metadata exists
TEST-FORM-016: Meta renders nothing when no metadata
TEST-FORM-017: Meta @tag changes element
TEST-FORM-018: Select generates options from enum
TEST-FORM-019: Select includes placeholder option
TEST-FORM-020: Select custom placeholder overrides
```

#### 11.1.8 Database Integration Tests

```
TEST-DB-001: Query with * returns Record
TEST-DB-002: Query with ??-> returns Table of Records
TEST-DB-003: Query Record has validated=true
TEST-DB-004: Projection subset returns Record
TEST-DB-005: Projection non-subset returns Dictionary
TEST-DB-006: Explicit ?!-> errors on non-subset
TEST-DB-007: Relations return Dictionaries
TEST-DB-008: Insert validates Record
```

### 11.2 Integration Test Scenarios

```
SCENARIO-001: Complete form submission flow
SCENARIO-002: Edit existing record flow
SCENARIO-003: Bulk CSV import with validation
SCENARIO-004: API request validation
SCENARIO-005: Form with all input types
SCENARIO-006: Form with custom validation
```

---

## 12. Verification Checklist

### 12.1 Phase 1 Checklist

| ID | Requirement | Implemented | Tested | Documented |
|----|-------------|-------------|--------|------------|
| P1-001 | Record struct with schema, data, errors, validated | ☐ | ☐ | ☐ |
| P1-002 | Schema({...}) returns Record | ☐ | ☐ | ☐ |
| P1-003 | Created Record has validated=false | ☐ | ☐ | ☐ |
| P1-004 | Default values applied at creation | ☐ | ☐ | ☐ |
| P1-005 | Unknown fields filtered | ☐ | ☐ | ☐ |
| P1-006 | Type casting on creation | ☐ | ☐ | ☐ |
| P1-007 | record.validate() returns new Record | ☐ | ☐ | ☐ |
| P1-008 | record.validate() sets validated=true | ☐ | ☐ | ☐ |
| P1-009 | record.update() merges and revalidates | ☐ | ☐ | ☐ |
| P1-010 | record.isValid() semantics correct | ☐ | ☐ | ☐ |
| P1-011 | record.errors() returns correct shape | ☐ | ☐ | ☐ |
| P1-012 | record.error(field) returns message | ☐ | ☐ | ☐ |
| P1-013 | record.errorCode(field) returns code | ☐ | ☐ | ☐ |
| P1-014 | record.errorList() returns array | ☐ | ☐ | ☐ |
| P1-015 | record.hasError(field) returns boolean | ☐ | ☐ | ☐ |
| P1-016 | record.withError(field, msg) works | ☐ | ☐ | ☐ |
| P1-017 | record.withError(field, code, msg) works | ☐ | ☐ | ☐ |
| P1-018 | record.data() returns plain dict | ☐ | ☐ | ☐ |
| P1-019 | record.keys() returns field names | ☐ | ☐ | ☐ |
| P1-020 | record.schema() returns schema | ☐ | ☐ | ☐ |
| P1-021 | Validation: required | ☐ | ☐ | ☐ |
| P1-022 | Validation: min/max length | ☐ | ☐ | ☐ |
| P1-023 | Validation: min/max value | ☐ | ☐ | ☐ |
| P1-024 | Validation: enum | ☐ | ☐ | ☐ |
| P1-025 | Validation: email format | ☐ | ☐ | ☐ |
| P1-026 | Validation: url format | ☐ | ☐ | ☐ |
| P1-027 | Validation: phone format | ☐ | ☐ | ☐ |
| P1-028 | Validation: slug format | ☐ | ☐ | ☐ |
| P1-029 | Spread syntax works | ☐ | ☐ | ☐ |
| P1-030 | JSON encoding works | ☐ | ☐ | ☐ |
| P1-031 | Record immutability | ☐ | ☐ | ☐ |

### 12.2 Phase 2 Checklist

| ID | Requirement | Implemented | Tested | Documented |
|----|-------------|-------------|--------|------------|
| P2-001 | Pipe syntax parsing | ☐ | ☐ | ☐ |
| P2-002 | schema.title(field) | ☐ | ☐ | ☐ |
| P2-003 | schema.placeholder(field) | ☐ | ☐ | ☐ |
| P2-004 | schema.meta(field, key) | ☐ | ☐ | ☐ |
| P2-005 | schema.fields() | ☐ | ☐ | ☐ |
| P2-006 | schema.visibleFields() | ☐ | ☐ | ☐ |
| P2-007 | schema.enumValues(field) | ☐ | ☐ | ☐ |
| P2-008 | Title case fallback | ☐ | ☐ | ☐ |
| P2-009 | record.format() date | ☐ | ☐ | ☐ |
| P2-010 | record.format() datetime | ☐ | ☐ | ☐ |
| P2-011 | record.format() currency | ☐ | ☐ | ☐ |
| P2-012 | record.format() percent | ☐ | ☐ | ☐ |
| P2-013 | record.format() number | ☐ | ☐ | ☐ |

### 12.3 Phase 3 Checklist

| ID | Requirement | Implemented | Tested | Documented |
|----|-------------|-------------|--------|------------|
| P3-001 | Schema([...]) creates Table | ☐ | ☐ | ☐ |
| P3-002 | @table(Schema) [...] literal | ☐ | ☐ | ☐ |
| P3-003 | {...}.as(Schema) creates Record | ☐ | ☐ | ☐ |
| P3-004 | table(data).as(Schema) creates Table | ☐ | ☐ | ☐ |
| P3-005 | table.validate() | ☐ | ☐ | ☐ |
| P3-006 | table.isValid() | ☐ | ☐ | ☐ |
| P3-007 | table.errors() with row indices | ☐ | ☐ | ☐ |
| P3-008 | table.validRows() | ☐ | ☐ | ☐ |
| P3-009 | table.invalidRows() | ☐ | ☐ | ☐ |
| P3-010 | table.schema() | ☐ | ☐ | ☐ |
| P3-011 | table[n] returns Record | ☐ | ☐ | ☐ |

### 12.4 Phase 4 Checklist

| ID | Requirement | Implemented | Tested | Documented |
|----|-------------|-------------|--------|------------|
| P4-001 | <form @record={...}> context | ☐ | ☐ | ☐ |
| P4-002 | <input @field="..."/> rewriting | ☐ | ☐ | ☐ |
| P4-003 | HTML validation attributes | ☐ | ☐ | ☐ |
| P4-004 | HTML type derivation | ☐ | ☐ | ☐ |
| P4-005 | Checkbox binding | ☐ | ☐ | ☐ |
| P4-006 | Radio binding | ☐ | ☐ | ☐ |
| P4-007 | aria-invalid attribute | ☐ | ☐ | ☐ |
| P4-008 | aria-describedby attribute | ☐ | ☐ | ☐ |
| P4-009 | aria-required attribute | ☐ | ☐ | ☐ |
| P4-010 | <Label @field="..."/> self-closing | ☐ | ☐ | ☐ |
| P4-011 | <Label @field="...">...</Label> tag-pair | ☐ | ☐ | ☐ |
| P4-012 | <Label @tag="..."/> override | ☐ | ☐ | ☐ |
| P4-013 | <Error @field="..."/> component | ☐ | ☐ | ☐ |
| P4-014 | <Error @tag="..."/> override | ☐ | ☐ | ☐ |
| P4-015 | Error role="alert" | ☐ | ☐ | ☐ |
| P4-016 | <Meta @field="..." @key="..."/> | ☐ | ☐ | ☐ |
| P4-017 | <Meta @tag="..."/> override | ☐ | ☐ | ☐ |
| P4-018 | <Select @field="..."/> | ☐ | ☐ | ☐ |
| P4-019 | <Select placeholder="..."/> | ☐ | ☐ | ☐ |
| P4-020 | record.title() shorthand | ☐ | ☐ | ☐ |
| P4-021 | record.placeholder() shorthand | ☐ | ☐ | ☐ |
| P4-022 | record.meta() shorthand | ☐ | ☐ | ☐ |
| P4-023 | record.enumValues() shorthand | ☐ | ☐ | ☐ |

### 12.5 Phase 5 Checklist

| ID | Requirement | Implemented | Tested | Documented |
|----|-------------|-------------|--------|------------|
| P5-001 | Query ?-> * returns Record | ✓ | ✓ | ☐ |
| P5-002 | Query ??-> * returns Table | ✓ | ✓ | ☐ |
| P5-003 | Query Record auto-validated | ✓ | ✓ | ☐ |
| P5-004 | Projection auto-detect | ☐ | ☐ | ☐ |
| P5-005 | ?!-> explicit terminal | ☐ | ☐ | ☐ |
| P5-006 | ??!-> explicit terminal | ☐ | ☐ | ☐ |
| P5-007 | Error on non-schema column | ☐ | ☐ | ☐ |
| P5-008 | Batch insert validation | ✓ | ☐ | ☐ |
| P5-009 | Partial record validation | ☐ | ☐ | ☐ |
| P5-010 | Relations return Dictionaries | ☐ | ☐ | ☐ |

---

## Appendix A: Grammar Snippets

### A.1 Schema Declaration

```ebnf
schema_decl     = "@schema" identifier "{" field_list "}" ;
schema_expr     = "@schema" "{" field_list "}" ;
field_list      = field_def { ("," | NEWLINE) field_def } ;
field_def       = identifier ":" type_spec [ "|" metadata ] ;
type_spec       = type_name [ "(" constraint_list ")" ] ;
type_name       = "int" | "bigint" | "float" | "string" | "text" | "bool"
                | "datetime" | "date" | "time" | "money" | "decimal" | "json"
                | "email" | "url" | "phone" | "slug" | "uuid" | "ulid"
                | enum_type ;
enum_type       = "enum" "(" string_literal { "," string_literal } ")" ;
constraint_list = constraint { "," constraint } ;
constraint      = "required" | "unique"
                | "min" ":" (integer | string)
                | "max" ":" (integer | string)
                | "default" ":" literal ;
metadata        = "{" [ key_value { "," key_value } ] "}" ;
key_value       = identifier ":" expression ;
```

### A.2 Form Binding

```ebnf
form_element    = "<form" "@record" "=" "{" expression "}" attributes ">" 
                  content "</form>" ;
input_binding   = "<input" "@field" "=" string attributes "/>" ;
label_binding   = "<Label" "@field" "=" string [ "@tag" "=" string ] "/>"
                | "<Label" "@field" "=" string [ "@tag" "=" string ] ">"
                  content "</Label>" ;
error_binding   = "<Error" "@field" "=" string [ "@tag" "=" string ] "/>" ;
meta_binding    = "<Meta" "@field" "=" string "@key" "=" string 
                  [ "@tag" "=" string ] "/>" ;
select_binding  = "<Select" "@field" "=" string [ "placeholder" "=" string ] "/>" ;
```

---

## Appendix B: Error Message Templates

| Code | Template |
|------|----------|
| `REQUIRED` | `"{title} is required"` |
| `TYPE` | `"{title} must be a {type}"` |
| `FORMAT` | `"{title} is not a valid {type}"` |
| `ENUM` | `"{title} must be one of: {values}"` |
| `MIN_LENGTH` | `"{title} must be at least {min} characters"` |
| `MAX_LENGTH` | `"{title} must be at most {max} characters"` |
| `MIN_VALUE` | `"{title} must be at least {min}"` |
| `MAX_VALUE` | `"{title} must be at most {max}"` |

---

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2025-01-15 | - | Initial specification from DESIGN-record-type-v3.md |
