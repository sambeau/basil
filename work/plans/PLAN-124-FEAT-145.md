---
id: PLAN-124
feature: FEAT-145
title: "Implementation Plan for Typed Value Formatting and Field Abstraction"
status: active
created: 2026-06-15
---

# Implementation Plan: FEAT-145

## Overview

Implement typed value formatting, form field abstraction methods, and table column props. This plan covers five phases: objectToString() changes, fieldProps() method, `<field/>` tag, columnProps() method, and documentation.

## Prerequisites

- [ ] Design doc reviewed: `work/design/DESIGN-typed-value-formatting.md`
- [ ] Feature spec approved: `work/specs/FEAT-145.md`
- [ ] Understand existing `@field` attribute system in `eval_form_context.go`

## Phase 1: Typed Value Formatting (2-3 hours)

### Task 1.1: Modify objectToString()
**Files**: `pkg/parsley/evaluator/eval_string_conversions.go`
**Estimated effort**: 1 hour

Steps:
1. Add case for `*Money` — call `moneyMedium()` and extract string value
2. Add case for `*Datetime` — call `datetimeMedium()` and extract string value
3. Add case for `*Duration` — call `durationMedium()` and extract string value
4. Add case for `*Unit` — call `unitMedium()` and extract string value
5. Fall back to `Inspect()` if medium() returns non-string

Tests:
- `<td>{money(499900, "GBP")}</td>` → `<td>£4,999.00</td>`
- `<td>{datetime("2025-03-15T14:30:00")}</td>` → `<td>Mar 15, 2025, 2:30 PM</td>`
- `<td>{duration(9000)}</td>` → `<td>2 hours 30 minutes</td>`
- `<td>{unit(5, "kg")}</td>` → `<td>5.00 kg</td>`

---

### Task 1.2: Modify objectToPrintString()
**Files**: `pkg/parsley/evaluator/eval_string_conversions.go`
**Estimated effort**: 30 min

Steps:
1. Apply same changes as Task 1.1 to `objectToPrintString()`
2. Ensure consistency between both functions

Tests:
- Same test cases as Task 1.1 but via print contexts

---

### Task 1.3: Verify raw format access
**Files**: `pkg/parsley/evaluator/` (tests only)
**Estimated effort**: 30 min

Steps:
1. Verify `.iso` property on datetime still returns ISO format
2. Verify `.short()` on duration/unit still works
3. Verify `.inspect()` returns programmer-friendly format
4. Add regression tests for these access patterns

Tests:
- `datetime("2025-03-15").iso` → `"2025-03-15"`
- `duration(9000).short()` → `"2h 30m"`
- `unit(5, "kg").short()` → `"5kg"`
- `money(499900, "GBP").inspect()` → programmer format

---

## Phase 2: fieldProps() Method (3-4 hours)

### Task 2.1: Create type mapping helpers
**Files**: `pkg/parsley/evaluator/form_helpers.go` (new or existing)
**Estimated effort**: 1 hour

Steps:
1. Create `inputTypeForSchemaType(schemaType string) string` helper
2. Create `inputModeForSchemaType(schemaType string) string` helper
3. Create `autocompleteForSchemaType(schemaType string) string` helper
4. Implement mappings per design doc:
   - email → email/email/email
   - url → url/url/url
   - phone → tel/tel/tel
   - integer → number/numeric/—
   - boolean → checkbox/—/—
   - date → date/—/—
   - datetime → datetime-local/—/—
   - money → text/decimal/—
   - enum → select/—/—

Tests:
- Unit tests for each mapping function
- All schema types covered

---

### Task 2.2: Implement recordFieldProps()
**Files**: `pkg/parsley/evaluator/methods_record.go`
**Estimated effort**: 2 hours

Steps:
1. Add method registration: `"fieldProps": {Fn: recordMethodFieldProps, Arity: "1-2", ...}`
2. Extract field name from first argument (required string)
3. Build result dictionary with:
   - `name` — field name
   - `type` — from schema via mapping helpers
   - `label` — from `.title()` or titlecased field name
   - `placeholder` — from `.placeholder()` if available
   - `value` — from record data, formatted for input (ISO for dates, decimal for money)
   - `required` — from schema constraint
   - `error` — from `.error()` if present
   - `autocomplete`, `inputmode` — from mapping helpers
   - `options` — for enums, array of allowed values
4. Handle optional second argument (overrides dictionary)
5. Merge overrides (second arg wins)

Tests:
- Basic email field returns correct props
- Money field value formatted as decimal string
- Date field value formatted as ISO
- Enum field includes options array
- Override argument merges correctly
- Non-existent field returns minimal props

---

### Task 2.3: Add value formatting for inputs
**Files**: `pkg/parsley/evaluator/methods_record.go`
**Estimated effort**: 30 min

Steps:
1. Create `formatValueForInput(value Object, schemaType string) Object` helper
2. Money → decimal string (e.g., "49.99" not 4999)
3. Date → ISO date string
4. Datetime → ISO local datetime string
5. Unit → numeric value only
6. Others → as-is

Tests:
- Money value: `{amount: 4999, currency: "GBP"}` → `"49.99"`
- Date value: datetime object → `"2025-03-15"`
- Datetime value: datetime object → `"2025-03-15T14:30"`

---

### Task 2.4: Update pars describe
**Files**: `pkg/parsley/evaluator/describe.go` or equivalent
**Estimated effort**: 30 min

Steps:
1. Add `fieldProps` to record type description
2. Document parameters and return value

Tests:
- `pars describe record` shows fieldProps method

---

## Phase 3: `<field/>` Tag (3-4 hours)

### Task 3.1: Create field tag handler
**Files**: `pkg/parsley/evaluator/form_field_tag.go` (new)
**Estimated effort**: 1 hour

Steps:
1. Create `evalFieldTag(node *ast.TagLiteral, propsStr string, env *Environment) Object`
2. Parse `name` attribute (required)
3. Get form context from environment (error if not in `@record` form)
4. Get record and schema from form context

Tests:
- Error when name not provided
- Error when not inside form @record context

---

### Task 3.2: Implement field output structure
**Files**: `pkg/parsley/evaluator/form_field_tag.go`
**Estimated effort**: 1.5 hours

Steps:
1. Build wrapper div with class (default "field")
2. Build label element with `for` attribute and text
3. Build input element with all attributes from schema
4. Build error span (only if error exists) with `role="alert"`
5. Handle props: `as`, `class`, `id`, `label`, `placeholder`, `help`
6. Apply ARIA attributes: `aria-required`, `aria-invalid`, `aria-describedby`

Tests:
- Basic field outputs correct structure
- Custom class applied to wrapper
- Label override works
- Help text rendered when provided
- Error span only rendered when error exists
- ARIA attributes correct

---

### Task 3.3: Implement checkbox special case
**Files**: `pkg/parsley/evaluator/form_field_tag.go`
**Estimated effort**: 30 min

Steps:
1. Detect boolean type from schema
2. For boolean: render input before label
3. Add `field--checkbox` class to wrapper
4. Handle checked attribute

Tests:
- Boolean field has input before label
- Boolean field has `field--checkbox` class
- Checked attribute present when value is true

---

### Task 3.4: Wire into tag evaluation
**Files**: `pkg/parsley/evaluator/eval_tags.go`
**Estimated effort**: 30 min

Steps:
1. Add case for `"field"` tag name in tag evaluation
2. Route to `evalFieldTag()`
3. Ensure self-closing syntax works: `<field name="email"/>`

Tests:
- `<field name="email"/>` routes to correct handler
- Non-self-closing `<field name="email"></field>` also works (or errors appropriately)

---

## Phase 4: columnProps() Method (2-3 hours)

### Task 4.1: Create alignment/format helpers
**Files**: `pkg/parsley/evaluator/table_helpers.go` (new or existing)
**Estimated effort**: 30 min

Steps:
1. Create `alignmentForType(schemaType string) string`:
   - money, integer, float, duration, unit → "right"
   - boolean → "center"
   - others → "left"
2. Create `formatForType(schemaType string) string`:
   - money → "currency"
   - date → "date"
   - datetime → "datetime"
   - duration → "duration"
   - unit → "unit"
   - boolean → "boolean"
   - others → "" (empty)

Tests:
- Unit tests for each helper function

---

### Task 4.2: Implement tableColumnProps()
**Files**: `pkg/parsley/evaluator/methods_table.go`
**Estimated effort**: 1.5 hours

Steps:
1. Add method registration: `"columnProps": {Fn: tableMethodColumnProps, Arity: "1", ...}`
2. Extract column name from argument (required string)
3. Build result dictionary:
   - `name` — column name
   - `label` — from schema title or titlecased name
   - `type` — schema type (if available)
   - `align` — from alignmentForType()
   - `format` — from formatForType() (only if non-empty)
4. Handle tables without schema (minimal props)

Tests:
- Schema-bound table returns full props
- Money column has align=right, format=currency
- Boolean column has align=center
- String column has align=left
- Table without schema returns minimal props

---

### Task 4.3: Update pars describe
**Files**: `pkg/parsley/evaluator/describe.go` or equivalent
**Estimated effort**: 30 min

Steps:
1. Add `columnProps` to table type description
2. Document parameters and return value

Tests:
- `pars describe table` shows columnProps method

---

## Phase 5: Documentation (1-2 hours)

### Task 5.1: Update form binding documentation
**Files**: `docs/parsley/manual/forms.md` or equivalent
**Estimated effort**: 45 min

Steps:
1. Document four abstraction levels (Level 1-4)
2. Add examples for each level
3. Explain when to use each level
4. Add fieldProps() usage examples

---

### Task 5.2: Add migration guide
**Files**: `docs/parsley/manual/` or `CHANGELOG.md`
**Estimated effort**: 30 min

Steps:
1. Document typed value formatting change
2. Explain how to access raw/ISO formats
3. Note that existing @field code is unchanged

---

### Task 5.3: Document columnProps() for DataTable
**Files**: `docs/parsley/manual/tables.md` or equivalent
**Estimated effort**: 30 min

Steps:
1. Document columnProps() method
2. Show integration pattern with DataTable
3. Explain relationship to fieldProps()

---

## Validation Checklist

- [ ] All tests pass: `go test ./pkg/parsley/...`
- [ ] Build succeeds: `make build`
- [ ] Benchmarks checked: `make bench-compare`
- [ ] Documentation updated
- [ ] `pars describe record` shows fieldProps
- [ ] `pars describe table` shows columnProps
- [ ] All FEAT-145 acceptance criteria checked
- [ ] FEAT-144 unblocked (Parts A and C complete)

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| | Task 1.1: objectToString() | | |
| | Task 1.2: objectToPrintString() | | |
| | Task 1.3: Verify raw access | | |
| | Task 2.1: Type mapping helpers | | |
| | Task 2.2: recordFieldProps() | | |
| | Task 2.3: Value formatting | | |
| | Task 2.4: pars describe record | | |
| | Task 3.1: Field tag handler | | |
| | Task 3.2: Field output structure | | |
| | Task 3.3: Checkbox special case | | |
| | Task 3.4: Wire into eval_tags | | |
| | Task 4.1: Alignment/format helpers | | |
| | Task 4.2: tableColumnProps() | | |
| | Task 4.3: pars describe table | | |
| | Task 5.1: Form binding docs | | |
| | Task 5.2: Migration guide | | |
| | Task 5.3: columnProps docs | | |

## Implementation Order

Recommended order to minimize conflicts and enable incremental testing:

1. **Phase 1** (Tasks 1.1-1.3) — Foundational, no dependencies, testable immediately
2. **Task 2.1** (type mapping helpers) — Shared by fieldProps and field tag
3. **Task 2.2-2.4** (fieldProps) — Uses helpers from 2.1
4. **Task 4.1** (alignment/format helpers) — Independent
5. **Task 4.2-4.3** (columnProps) — Uses helpers from 4.1, unblocks FEAT-144
6. **Phase 3** (Tasks 3.1-3.4) — Uses helpers from 2.1, can be done in parallel with Phase 4
7. **Phase 5** (documentation) — Final step after all functionality complete

## Dependencies

This feature **blocks FEAT-144** (DataTable Redesign):
- FEAT-144 requires Part A (objectToString changes) for typed value display
- FEAT-144 requires Part C (columnProps) for schema-aware column metadata

Complete Phase 1 and Phase 4 first to unblock DataTable work.