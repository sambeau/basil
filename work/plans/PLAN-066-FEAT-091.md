---
id: PLAN-066
feature: FEAT-091
title: "Implementation Plan for Record Type"
status: draft
created: 2026-01-15
---

# Implementation Plan: FEAT-091 Record Type

## Overview

Implement the **Record type** for Parsley — a typed wrapper around data that carries its schema and validation state. This is a multi-phase feature spanning lexer, parser, evaluator, and template compilation.

**Total estimated effort:** Large (4-6 weeks)
**Risk level:** Medium-High (touches multiple subsystems)

## Prerequisites

- [ ] Review existing `@schema` implementation in Query DSL
- [ ] Understand current Table type implementation
- [ ] Review Dictionary type and its methods

## Architecture Overview

```
@schema User {...}          ← Schema object (exists, needs extensions)
       │
       ├── User({...})      ← Record (NEW TYPE)
       │      │
       │      ├── data: Dictionary
       │      ├── schema: Schema
       │      ├── errors: Dictionary
       │      └── validated: Boolean
       │
       └── User([...])      ← TypedTable (extends Table)
              │
              └── rows: []Record
```

**Key insight:** Record IS-A Dictionary for data access. This enables compatibility with spread, JSON encoding, and existing functions.

---

## Phase 1: Core Record Type

**Estimated effort:** Large (1-2 weeks)
**Risk:** Medium (new type, but isolated)
**Dependencies:** None

### 1.1 Create Record Object Type

**Files:** `pkg/parsley/object/object.go`

Add new object type:

```go
type Record struct {
    Schema    *Schema
    Data      *Dictionary
    Errors    *Dictionary  // {field: {code: string, message: string}}
    Validated bool
}

func (r *Record) Type() ObjectType { return RECORD_OBJ }
func (r *Record) Inspect() string { ... }
```

**Steps:**
1. Add `RECORD_OBJ` constant to ObjectType enum
2. Implement `Record` struct with Schema, Data, Errors, Validated fields
3. Implement `Type()` and `Inspect()` methods
4. Implement dictionary-like access (IS-A Dictionary semantics)

**Commit:** `feat(parsley): add Record object type`

---

### 1.2 Make Schema Callable

**Files:** `pkg/parsley/evaluator/evaluator.go`

Enable `Schema({...})` syntax to create Records:

```go
func (e *Evaluator) evalCallExpression(call *ast.CallExpression, env *Environment) Object {
    // ... existing code ...
    
    // Check if callee is a Schema
    if schema, ok := function.(*object.Schema); ok {
        return e.createRecordFromSchema(schema, args, env)
    }
}
```

**Steps:**
1. Add case in `evalCallExpression` for Schema callee
2. Implement `createRecordFromSchema(schema, args, env)`
3. Handle dict argument → Record
4. Handle array argument → Table (Phase 3)
5. Apply default values from schema
6. Apply type casting based on field types
7. Filter unknown fields (whitelist)
8. Set `validated = false`

**Commit:** `feat(parsley): make Schema callable to create Records`

---

### 1.3 Implement Record Validation

**Files:** `pkg/parsley/evaluator/record_validation.go` (new)

Create validation engine:

```go
func (e *Evaluator) validateRecord(record *object.Record) *object.Record {
    errors := make(map[string]object.Object)
    
    for _, field := range record.Schema.Fields {
        if err := e.validateField(record, field); err != nil {
            errors[field.Name] = err
        }
    }
    
    return &object.Record{
        Schema:    record.Schema,
        Data:      record.Data,
        Errors:    object.NewDictionary(errors),
        Validated: true,
    }
}
```

**Validation rules (in order):**
1. Type casting (if needed)
2. Required check (null/missing → REQUIRED)
3. Type check (wrong type → TYPE)
4. Format check for validated types (email, url, etc. → FORMAT)
5. Constraint checks (min, max → MIN_LENGTH, MAX_LENGTH, MIN_VALUE, MAX_VALUE)
6. Enum check (not in set → ENUM)

**Edge cases:**
- Empty string `""` passes `required`
- Zero `0` passes `required`
- False `false` passes `required`
- Only first error per field is stored

**Commit:** `feat(parsley): implement Record validation engine`

---

### 1.4 Implement Record Methods

**Files:** `pkg/parsley/evaluator/methods_record.go` (new)

| Method | Implementation |
|--------|----------------|
| `validate()` | Call `validateRecord()`, return new Record |
| `update(dict)` | Merge data, call `validateRecord()` |
| `errors()` | Return `record.Errors` |
| `error(field)` | Return `errors[field].message` or null |
| `errorCode(field)` | Return `errors[field].code` or null |
| `errorList()` | Convert errors dict to array |
| `isValid()` | Return `validated && len(errors) == 0` |
| `hasError(field)` | Return `field in errors` |
| `schema()` | Return `record.Schema` |
| `data()` | Return `record.Data` |
| `keys()` | Return `record.Schema.Fields.map(f => f.Name)` |
| `withError(field, msg)` | Add error with code "CUSTOM" |
| `withError(field, code, msg)` | Add error with custom code |

**Commit:** `feat(parsley): implement Record methods`

---

### 1.5 Dictionary Compatibility

**Files:** `pkg/parsley/evaluator/evaluator.go`

Ensure Record works as Dictionary:

1. **Spread syntax:** `{...record}` expands data fields only
2. **JSON encoding:** `json.encode(record)` encodes data only
3. **Function arguments:** Functions accepting Dictionary accept Record
4. **Dot access:** `record.fieldName` returns data value

**Steps:**
1. Update spread handling to check for Record type
2. Update JSON encoder to use `record.Data`
3. Record already has dictionary-like access via `Data` field

**Commit:** `feat(parsley): ensure Record dictionary compatibility`

---

### 1.6 Implement Error Messages

**Files:** `pkg/parsley/evaluator/record_validation.go`

Create error message generator using field titles:

```go
var errorTemplates = map[string]string{
    "REQUIRED":   "{title} is required",
    "TYPE":       "{title} must be a {type}",
    "FORMAT":     "{title} is not a valid {type}",
    "ENUM":       "{title} must be one of: {values}",
    "MIN_LENGTH": "{title} must be at least {min} characters",
    "MAX_LENGTH": "{title} must be at most {max} characters",
    "MIN_VALUE":  "{title} must be at least {min}",
    "MAX_VALUE":  "{title} must be at most {max}",
}
```

**Title resolution:**
1. Use explicit `title` from metadata
2. Fall back to titlecase of field name (`firstName` → `First Name`)

**Commit:** `feat(parsley): implement validation error messages`

---

### 1.7 Phase 1 Tests

**Files:** `pkg/parsley/tests/record_test.go` (new)

Test categories:
- Record creation (TEST-REC-001 through TEST-REC-007)
- Validation (TEST-VAL-001 through TEST-VAL-021)
- Error handling (TEST-ERR-001 through TEST-ERR-009)
- Methods (TEST-MTD-001 through TEST-MTD-010)
- Dictionary compatibility (TEST-DICT-001 through TEST-DICT-003)

**Commit:** `test(parsley): add Record type tests`

---

## Phase 2: Schema Metadata

**Estimated effort:** Medium (3-5 days)
**Risk:** Low (additive)
**Dependencies:** Phase 1

### 2.1 Parse Pipe Syntax

**Files:** `pkg/parsley/parser/parser.go`

Extend schema field parsing:

```parsley
name: string(min: 2, required) | {title: "Full Name", placeholder: "Enter name"}
```

**Grammar:**
```
field_def = identifier ":" type_spec [ "|" metadata ]
metadata  = "{" [ key_value { "," key_value } ] "}"
```

**Steps:**
1. After parsing type_spec, check for `|` token
2. If present, parse dictionary literal as metadata
3. Store metadata in `SchemaField.Metadata`

**Commit:** `feat(parsley): parse pipe syntax for schema metadata`

---

### 2.2 Implement Schema Methods

**Files:** `pkg/parsley/evaluator/methods_schema.go`

| Method | Implementation |
|--------|----------------|
| `title(field)` | Return metadata.title or titlecase(field) |
| `placeholder(field)` | Return metadata.placeholder or null |
| `meta(field, key)` | Return metadata[key] or null |
| `fields()` | Return array of field names |
| `visibleFields()` | Return fields where hidden != true |
| `enumValues(field)` | Return enum values or empty array |

**Title case function:**
```go
func toTitleCase(s string) string {
    // "firstName" → "First Name"
    // "email" → "Email"
}
```

**Commit:** `feat(parsley): implement Schema metadata methods`

---

### 2.3 Implement Record.format()

**Files:** `pkg/parsley/evaluator/methods_record.go`

Format values based on schema hints:

| Format | Implementation |
|--------|----------------|
| `date` | `time.Format("Jan 2, 2006")` |
| `datetime` | `time.Format("Jan 2, 2006 3:04 PM")` |
| `currency` | Format with currency symbol and commas |
| `percent` | Multiply by 100, add % |
| `number` | Add thousands separators |

**Commit:** `feat(parsley): implement Record.format() method`

---

### 2.4 Phase 2 Tests

**Files:** `pkg/parsley/tests/schema_metadata_test.go` (new)

Test categories:
- Pipe syntax parsing
- Schema methods (title, placeholder, meta, fields, visibleFields, enumValues)
- Title case fallback
- Format methods (date, datetime, currency, percent, number)

**Commit:** `test(parsley): add Schema metadata tests`

---

## Phase 3: Table Integration

**Estimated effort:** Medium (3-5 days)
**Risk:** Low (extends existing Table)
**Dependencies:** Phase 1, Phase 2

### 3.1 Schema with Array Creates Table

**Files:** `pkg/parsley/evaluator/evaluator.go`

Extend Schema callable to handle arrays:

```go
func (e *Evaluator) createRecordFromSchema(schema, args, env) Object {
    arg := args[0]
    
    switch v := arg.(type) {
    case *object.Dictionary:
        return e.createSingleRecord(schema, v)
    case *object.Array:
        return e.createTypedTable(schema, v)
    }
}
```

**Steps:**
1. Check if argument is Array
2. Create Table with schema attached
3. Each row becomes a Record (unvalidated)

**Commit:** `feat(parsley): Schema([...]) creates typed Table`

---

### 3.2 Implement .as() Method

**Files:** `pkg/parsley/evaluator/methods_dictionary.go`, `methods_table.go`

Add `.as(Schema)` to Dictionary and Table:

```parsley
{name: "Alice"}.as(User)        // → Record
table(data).as(User)            // → TypedTable
```

**Commit:** `feat(parsley): add .as(Schema) method to Dictionary and Table`

---

### 3.3 Implement Table Validation Methods

**Files:** `pkg/parsley/evaluator/methods_table.go`

| Method | Implementation |
|--------|----------------|
| `validate()` | Validate all rows, return new Table |
| `isValid()` | Return true if all rows valid |
| `errors()` | Return `[{row, field, code, message}]` |
| `validRows()` | Filter to valid rows only |
| `invalidRows()` | Filter to invalid rows only |
| `schema()` | Return Table's schema |

**Commit:** `feat(parsley): implement Table validation methods`

---

### 3.4 Row Access Returns Record

**Files:** `pkg/parsley/evaluator/evaluator.go`

Ensure `table[n]` returns Record:

```go
func (e *Evaluator) evalIndexExpression(table, index) Object {
    if typedTable, ok := table.(*object.TypedTable); ok {
        row := typedTable.Rows[idx]
        return row // Already a Record
    }
}
```

**Commit:** `feat(parsley): table[n] returns Record for typed Tables`

---

### 3.5 Phase 3 Tests

**Files:** `pkg/parsley/tests/typed_table_test.go` (new)

Test categories (TEST-TBL-001 through TEST-TBL-008):
- Schema with array creates Table
- Table rows are Records
- Table validation methods
- Row access returns Record

**Commit:** `test(parsley): add typed Table tests`

---

## Phase 4: Form Binding

**Estimated effort:** Large (1-2 weeks)
**Risk:** Medium-High (template compilation changes)
**Dependencies:** Phase 1, Phase 2

### 4.1 Parse @record Attribute

**Files:** `pkg/parsley/parser/parser.go`

Recognize `<form @record={expr}>`:

**Steps:**
1. When parsing tag attributes, check for `@record`
2. Parse expression in braces
3. Store in AST as special attribute

**Commit:** `feat(parsley): parse @record attribute on form elements`

---

### 4.2 Establish Form Context

**Files:** `pkg/parsley/evaluator/evaluator.go`

Track form context during evaluation:

```go
type FormContext struct {
    Record *object.Record
}

func (e *Evaluator) evalFormElement(form *ast.TagExpression, env) Object {
    record := e.Eval(form.RecordExpr, env)
    
    // Push form context
    ctx := &FormContext{Record: record.(*object.Record)}
    e.formContextStack = append(e.formContextStack, ctx)
    defer e.popFormContext()
    
    // Evaluate children with context
    return e.evalChildren(form.Children, env)
}
```

**Commit:** `feat(parsley): establish form context for @record`

---

### 4.3 Implement @field Input Rewriting

**Files:** `pkg/parsley/evaluator/form_binding.go` (new)

Rewrite `<input @field="name"/>` to full input:

```go
func (e *Evaluator) rewriteFieldInput(input *ast.TagExpression) *ast.TagExpression {
    field := input.Attributes["@field"]
    record := e.currentFormContext().Record
    schema := record.Schema
    
    // Build attributes
    attrs := map[string]Object{
        "name": field,
        "value": record.Data.Get(field),
    }
    
    // Add validation attributes from schema
    if constraint := schema.GetConstraint(field, "required"); constraint {
        attrs["required"] = TRUE
        attrs["aria-required"] = "true"
    }
    if min := schema.GetConstraint(field, "min"); min != nil {
        if schema.IsStringType(field) {
            attrs["minlength"] = min
        } else {
            attrs["min"] = min
        }
    }
    // ... max, type, etc.
    
    // Add ARIA attributes
    hasError := record.HasError(field)
    attrs["aria-invalid"] = hasError ? "true" : "false"
    if hasError {
        attrs["aria-describedby"] = field + "-error"
    }
    
    return newTagExpression("input", attrs)
}
```

**Type derivation:**

| Schema Type | HTML Type |
|-------------|-----------|
| `email` | `email` |
| `url` | `url` |
| `phone` | `tel` |
| `int` | `number` |
| `date` | `date` |
| `datetime` | `datetime-local` |
| `time` | `time` |

**Commit:** `feat(parsley): implement @field input rewriting`

---

### 4.4 Checkbox and Radio Binding

**Files:** `pkg/parsley/evaluator/form_binding.go`

Handle special input types:

```go
// Checkbox: type="checkbox" binds checked to boolean
if inputType == "checkbox" {
    attrs["checked"] = record.Data.Get(field) // boolean
}

// Radio: type="radio" binds checked to equality
if inputType == "radio" {
    value := input.Attributes["value"]
    attrs["checked"] = record.Data.Get(field) == value
}
```

**Commit:** `feat(parsley): implement checkbox and radio binding`

---

### 4.5 Implement Label Component

**Files:** `pkg/parsley/evaluator/form_components.go` (new)

Handle `<Label @field="x"/>` and `<Label @field="x">...</Label>`:

```go
func (e *Evaluator) evalLabelComponent(label *ast.TagExpression) Object {
    field := label.Attributes["@field"]
    tag := label.Attributes["@tag"] || "label"
    record := e.currentFormContext().Record
    title := record.Schema.Title(field)
    
    if label.IsSelfClosing {
        // <Label @field="x"/> → <label for="x">Title</label>
        return newTagExpression(tag, {"for": field}, title)
    } else {
        // <Label @field="x">...</Label> → <label>Title...children...</label>
        children := append([]Object{title}, e.evalChildren(label.Children)...)
        return newTagExpression(tag, {}, children)
    }
}
```

**Commit:** `feat(parsley): implement Label component`

---

### 4.6 Implement Error Component

**Files:** `pkg/parsley/evaluator/form_components.go`

Handle `<Error @field="x"/>`:

```go
func (e *Evaluator) evalErrorComponent(error *ast.TagExpression) Object {
    field := error.Attributes["@field"]
    tag := error.Attributes["@tag"] || "span"
    record := e.currentFormContext().Record
    
    if !record.HasError(field) {
        return NULL // Render nothing
    }
    
    return newTagExpression(tag, {
        "id": field + "-error",
        "class": "error",
        "role": "alert",
    }, record.Error(field))
}
```

**Commit:** `feat(parsley): implement Error component`

---

### 4.7 Implement Meta Component

**Files:** `pkg/parsley/evaluator/form_components.go`

Handle `<Meta @field="x" @key="help"/>`:

```go
func (e *Evaluator) evalMetaComponent(meta *ast.TagExpression) Object {
    field := meta.Attributes["@field"]
    key := meta.Attributes["@key"]
    tag := meta.Attributes["@tag"] || "span"
    record := e.currentFormContext().Record
    
    value := record.Schema.Meta(field, key)
    if value == nil {
        return NULL // Render nothing
    }
    
    return newTagExpression(tag, {}, value)
}
```

**Commit:** `feat(parsley): implement Meta component`

---

### 4.8 Implement Select Component

**Files:** `pkg/parsley/evaluator/form_components.go`

Handle `<Select @field="x"/>`:

```go
func (e *Evaluator) evalSelectComponent(sel *ast.TagExpression) Object {
    field := sel.Attributes["@field"]
    placeholder := sel.Attributes["placeholder"] || record.Schema.Placeholder(field) || ""
    record := e.currentFormContext().Record
    
    options := []Object{
        newTagExpression("option", {"value": ""}, placeholder),
    }
    
    for _, value := range record.Schema.EnumValues(field) {
        selected := record.Data.Get(field) == value
        options = append(options, newTagExpression("option", {
            "value": value,
            "selected": selected,
        }, value))
    }
    
    return newTagExpression("select", {
        "name": field,
        "aria-invalid": record.HasError(field) ? "true" : "false",
        "aria-describedby": record.HasError(field) ? field + "-error" : nil,
    }, options)
}
```

**Commit:** `feat(parsley): implement Select component`

---

### 4.9 Record Shorthand Methods

**Files:** `pkg/parsley/evaluator/methods_record.go`

Add convenience methods:

```go
// record.title(field) → schema.title(field)
// record.placeholder(field) → schema.meta(field, "placeholder")
// record.meta(field, key) → schema.meta(field, key)
// record.enumValues(field) → schema.enumValues(field)
```

**Commit:** `feat(parsley): add Record shorthand methods`

---

### 4.10 Phase 4 Tests

**Files:** `pkg/parsley/tests/form_binding_test.go` (new)

Test categories (TEST-FORM-001 through TEST-FORM-020):
- @record establishes context
- @field binds value and adds attributes
- ARIA attributes
- Checkbox and radio binding
- Label, Error, Meta, Select components

**Commit:** `test(parsley): add form binding tests`

---

## Phase 5: Database Integration

**Estimated effort:** Medium (3-5 days)
**Risk:** Medium (touches Query DSL)
**Dependencies:** Phase 1, Phase 3

### 5.1 Query Returns Records

**Files:** `pkg/parsley/evaluator/query.go`

Modify query evaluation to return Records:

```go
func (e *Evaluator) evalQuery(query *ast.QueryExpression) Object {
    // ... execute query ...
    
    binding := query.Binding
    if binding.Schema != nil {
        // Check if projection is subset of schema
        if e.isProjectionSubsetOfSchema(query.Projection, binding.Schema) {
            // Return Record(s)
            if query.IsSingle {
                return e.rowToRecord(row, binding.Schema, true) // validated=true
            } else {
                return e.rowsToTypedTable(rows, binding.Schema, true)
            }
        }
    }
    
    // Fall back to Dictionary/Table
    return e.rowsToDictionaries(rows)
}
```

**Commit:** `feat(parsley): queries return Records when schema available`

---

### 5.2 Explicit Record Terminals

**Files:** `pkg/parsley/parser/parser.go`, `evaluator/query.go`

Add `?!->` and `??!->` terminals:

```parsley
@query(Users ?!-> name, email)   // Record or error
@query(Users ??!-> name, email)  // Table of Records or error
```

**Steps:**
1. Parse `?!->` and `??!->` as new terminal types
2. In evaluator, verify projection ⊆ schema.fields()
3. Error if any column not in schema

**Commit:** `feat(parsley): add explicit Record terminals ?!-> and ??!->`

---

### 5.3 Auto-Validation on Query

**Files:** `pkg/parsley/evaluator/query.go`

Records from queries are auto-validated:

```go
func (e *Evaluator) rowToRecord(row, schema, autoValidated) *object.Record {
    return &object.Record{
        Schema:    schema,
        Data:      rowToDict(row),
        Errors:    object.NewDictionary(nil), // Empty
        Validated: autoValidated,             // true for query results
    }
}
```

**Commit:** `feat(parsley): auto-validate Records from queries`

---

### 5.4 Insert Validation

**Files:** `pkg/parsley/evaluator/query.go`

Validate before insert:

```go
func (e *Evaluator) evalInsert(insert *ast.InsertExpression) Object {
    data := e.Eval(insert.Data, env)
    
    // If data is Record, validate it
    if record, ok := data.(*object.Record); ok {
        validated := e.validateRecord(record)
        if !validated.IsValid() {
            return e.newValidationError(validated.Errors)
        }
        data = validated.Data
    }
    
    // ... execute insert ...
}
```

**Commit:** `feat(parsley): validate Records on @insert`

---

### 5.5 Relations Return Dictionaries

**Files:** `pkg/parsley/evaluator/query.go`

Ensure eager-loaded relations are plain Dictionaries:

```go
func (e *Evaluator) loadRelation(parent, relation) Object {
    rows := e.executeRelationQuery(parent, relation)
    // Always return Dictionary, not Record
    return e.rowsToDictionaries(rows)
}
```

**Commit:** `feat(parsley): relations return Dictionaries not Records`

---

### 5.6 Phase 5 Tests

**Files:** `pkg/parsley/tests/record_db_test.go` (new)

Test categories (TEST-DB-001 through TEST-DB-008):
- Query returns Record
- Query returns Table of Records
- Auto-validation on query
- Projection auto-detect
- Explicit terminals error on non-subset
- Insert validation
- Relations return Dictionaries

**Commit:** `test(parsley): add Record database integration tests`

---

## Documentation Updates

### After Phase 1
- [ ] `docs/parsley/reference.md` — Add Record type section
- [ ] `docs/parsley/CHEATSHEET.md` — Add Record creation and validation

### After Phase 2
- [ ] `docs/parsley/reference.md` — Add Schema metadata section
- [ ] Update `docs/guide/query-dsl.md` — Document pipe syntax

### After Phase 4
- [ ] `docs/guide/forms.md` (new) — Complete form binding guide
- [ ] `docs/guide/README.md` — Link to forms guide
- [ ] `docs/guide/faq.md` — Add form-related FAQs

### After Phase 5
- [ ] Update `docs/guide/query-dsl.md` — Document Record return types
- [ ] `examples/` — Add form validation example

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Breaking existing @schema | Phase 1 tests verify backward compatibility |
| Dictionary compatibility issues | Explicit IS-A relationship, comprehensive tests |
| Form context leaks | Stack-based context with defer cleanup |
| Query DSL regressions | Existing query tests plus new Record tests |
| Performance with large tables | Lazy validation option in Phase 5 (future) |

---

## Progress Log

| Date | Phase | Status | Notes |
|------|-------|--------|-------|
| 2026-01-15 | - | Planning | Plan created |
| 2026-01-16 | Phase 1 | Complete | Core Record type (commit a82f3e9) |
| 2026-01-16 | Phase 2 | Complete | Schema metadata (commit ff608c8) |
| 2026-01-16 | Phase 3 | Complete | Table integration (commit c327009) |
| 2026-01-16 | Phase 4 | Complete | Form binding: @record context, @field input rewriting, Label/Error/Meta/Select components |

---

## Verification

Use the checklists in FEAT-091 Section 12 to verify each phase:
- Phase 1: P1-001 through P1-031
- Phase 2: P2-001 through P2-013
- Phase 3: P3-001 through P3-011
- Phase 4: P4-001 through P4-023
- Phase 5: P5-001 through P5-010
