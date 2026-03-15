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

### Task 1.1: Modify objectToTemplateString()
**Files**: `pkg/parsley/evaluator/eval_string_conversions.go`
**Estimated effort**: 1 hour
**Status**: ✅ Complete

Steps:
1. ✅ Add case for `*Money` — call `moneyMedium()` and extract string value
2. ⏸️ Datetime — kept existing ISO format (`.medium()` doesn't respect datetime kinds)
3. ⏸️ Duration — kept existing format (`.medium()` returns relative time, not absolute)
4. ⏸️ Unit — kept existing format (`.medium()` adds unwanted decimal places)
5. ✅ Fall back to `Inspect()` if medium() returns non-string

**Implementation Notes**:
- Only Money formatting was changed to use `.medium()` for human-readable output
- Datetime `.medium()` doesn't properly handle datetime kinds (date-only, time-only, full datetime)
- Duration `.medium()` returns relative time like "tomorrow" instead of "1 day"
- Unit existing format is preferred by users (no forced decimal places)
- These issues should be addressed in separate tasks before enabling `.medium()` for those types

Tests:
- ✅ `"Price: " + £4999.00` → `"Price: £ 4,999.00"` (with thousands separator)
- ✅ Template interpolation: `` `<td>{m}</td>` `` → `"<td>£ 4,999.00</td>"`

---

### Task 1.2: Modify objectToPrintString()
**Files**: `pkg/parsley/evaluator/eval_string_conversions.go`
**Estimated effort**: 30 min
**Status**: ✅ Complete

Steps:
1. ✅ Apply same Money changes as Task 1.1 to `objectToPrintString()`
2. ✅ Ensure consistency between both functions

Tests:
- Same test cases as Task 1.1 but via print contexts

---

### Task 1.3: Verify raw format access
**Files**: `pkg/parsley/evaluator/` (tests only)
**Estimated effort**: 30 min
**Status**: ✅ Complete (existing tests pass)

Steps:
1. ✅ Verify `.iso` property on datetime still returns ISO format
2. ✅ Verify `.short()` on duration/unit still works
3. ✅ Verify `.inspect()` returns programmer-friendly format
4. Existing regression tests cover these patterns

Tests:
- ✅ All existing datetime/duration/unit tests pass
- ✅ `datetime("2025-03-15").iso` → `"2025-03-15"` (unchanged)
- ✅ `duration(@1d).short()` → `"1d"` (unchanged)
- ✅ `unit(5, "kg").short()` → `"5kg"` (unchanged)

---

## Phase 2: fieldProps() Method (3-4 hours)

### Task 2.1: Create type mapping helpers
**Files**: `pkg/parsley/evaluator/methods_record.go`
**Estimated effort**: 1 hour
**Status**: ✅ Complete

Steps:
1. ✅ Create `inputTypeForSchemaType(schemaType string) (inputType, inputMode, autocomplete string)` helper
2. ✅ Implement mappings per design doc:
   - email → email/email/email
   - url → url/url/url
   - phone → tel/tel/tel
   - integer, int → number/numeric/—
   - float, number → text/decimal/—
   - boolean, bool → checkbox/—/—
   - date → date/—/—
   - datetime → datetime-local/—/—
   - money → text/decimal/—
   - unit → text/decimal/—
   - password → password/—/current-password

Tests:
- ✅ Verified via manual testing with `pars -e`

---

### Task 2.2: Implement recordFieldProps()
**Files**: `pkg/parsley/evaluator/methods_record.go`
**Estimated effort**: 2 hours
**Status**: ✅ Complete

Steps:
1. ✅ Add method registration: `"fieldProps": {Fn: recordMethodFieldProps, Arity: "1-2", ...}`
2. ✅ Extract field name from first argument (required string)
3. ✅ Build result dictionary with:
   - `name` — field name
   - `type` — from schema via mapping helpers
   - `label` — from `.title()` or titlecased field name
   - `placeholder` — from `.placeholder()` if available
   - `value` — from record data, formatted for input (ISO for dates, decimal for money)
   - `required` — from schema constraint
   - `error` — from `.error()` if present
   - `autocomplete`, `inputmode` — from mapping helpers
   - `options` — for enums, array of allowed values
4. ✅ Handle optional second argument (overrides dictionary)
5. ✅ Merge overrides (second arg wins)

Tests:
- ✅ `user.fieldProps("email")` → `{name: "email", type: "email", label: "Email Address", ...}`
- ✅ Money field value formatted as decimal string ("49.99" not 4999)
- ✅ Override argument merges correctly: `fieldProps("email", {label: "Work Email"})`
- ✅ Error field included when validation fails
- ✅ Enum fields include options array

---

### Task 2.3: Add value formatting for inputs
**Files**: `pkg/parsley/evaluator/methods_record.go`
**Estimated effort**: 30 min
**Status**: ✅ Complete

Steps:
1. ✅ Create `formatValueForInput(val Object, field *DSLSchemaField) ast.Expression` helper
2. ✅ Money → decimal string (e.g., "49.99" not 4999)
3. ✅ Datetime → ISO string (removes trailing Z for datetime-local input)
4. ✅ Unit → numeric value only
5. ✅ String, Integer, Float, Boolean → as-is

Tests:
- ✅ Money value: £49.99 → `"49.99"`
- ✅ Datetime value → ISO string for input

---

### Task 2.4: Update pars describe
**Files**: `pkg/parsley/evaluator/describe.go` or equivalent
**Estimated effort**: 30 min
**Status**: ✅ Complete (auto-registered)

Steps:
1. ✅ Method auto-registered in RecordMethodRegistry with description
2. ✅ `pars describe record` shows fieldProps method

Tests:
- ✅ `pars describe record` shows `.fieldProps(arg1, arg2?) Get form field props for a field (field, overrides?)`

---

## Phase 3: `<field/>` Tag (3-4 hours)

### Task 3.1: Create field tag handler
**Files**: `pkg/parsley/evaluator/form_components.go`, `pkg/parsley/evaluator/eval_tags.go`
**Estimated effort**: 1 hour
**Status**: ✅ Complete

Steps:
1. ✅ Create `evalFieldTag(node *ast.TagLiteral, propsStr string, env *Environment) Object`
2. ✅ Parse `name` attribute (required)
3. ✅ Get form context from environment (error if not in `@record` form)
4. ✅ Get record and schema from form context
5. ✅ Wire into `evalStandardTag()` in `eval_tags.go`

Tests:
- ✅ Error FORM-0010 when name not provided
- ✅ Error FORM-0002 when not inside form @record context

---

### Task 3.2: Implement field output structure
**Files**: `pkg/parsley/evaluator/form_components.go`
**Estimated effort**: 1.5 hours
**Status**: ✅ Complete

Steps:
1. ✅ Build wrapper div with class (default "field")
2. ✅ Build label element with `for` attribute and text
3. ✅ Build input element with all attributes from schema
4. ✅ Build error span (only if error exists) with `role="alert"`
5. ✅ Handle props: `as`, `class`, `id`, `label`, `placeholder`, `help`
6. ✅ Apply ARIA attributes: `aria-required`, `aria-invalid`, `aria-describedby`
7. ✅ Support `as="textarea"` and `as="select"` (delegates to existing evalSelectComponent)
8. ✅ Constraints: minlength, maxlength, min, max, pattern

Tests:
- ✅ Basic field outputs div > label + input structure
- ✅ Custom class applied to wrapper
- ✅ Label override works
- ✅ Help text rendered as `<span class="help">`
- ✅ Error span only rendered when field has error
- ✅ ARIA attributes correct (aria-required, aria-invalid, aria-describedby)
- ✅ Textarea renders with value inside tags

---

### Task 3.3: Implement checkbox special case
**Files**: `pkg/parsley/evaluator/form_components.go`
**Estimated effort**: 30 min
**Status**: ✅ Complete

Steps:
1. ✅ Detect boolean type from schema
2. ✅ For boolean: render input before label
3. ✅ Add `field--checkbox` class to wrapper
4. ✅ Handle checked attribute

Tests:
- ✅ Boolean field has input before label
- ✅ Boolean field has `field--checkbox` class
- ✅ Checked attribute present when value is true

---

### Task 3.4: Wire into tag evaluation
**Files**: `pkg/parsley/evaluator/eval_tags.go`
**Estimated effort**: 30 min
**Status**: ✅ Complete (done as part of Task 3.1)

Steps:
1. ✅ Add case for `"field"` tag name in `evalStandardTag()`
2. ✅ Route to `evalFieldTag()`
3. ✅ Self-closing syntax works: `<field name="email"/>`

Tests:
- ✅ `<field name="email"/>` routes to correct handler

---

## Phase 4: columnProps() Method (2-3 hours)

### Task 4.1: Create alignment/format helpers
**Files**: `pkg/parsley/evaluator/stdlib_table.go`
**Estimated effort**: 30 min
**Status**: ✅ Complete

Steps:
1. ✅ Create `alignmentForSchemaType(schemaType string) string`:
   - money, integer, int, float, number, duration, unit → "right"
   - boolean, bool → "center"
   - others → "left"
2. ✅ Create `formatHintForSchemaType(schemaType string) string`:
   - money → "currency"
   - date → "date"
   - datetime → "datetime"
   - duration → "duration"
   - unit → "unit"
   - boolean, bool → "boolean"
   - others → "" (empty)

Tests:
- ✅ Verified via manual testing with `pars -e`

---

### Task 4.2: Implement tableColumnProps()
**Files**: `pkg/parsley/evaluator/stdlib_table.go`
**Estimated effort**: 1.5 hours
**Status**: ✅ Complete

Steps:
1. ✅ Add method registration: `"columnProps": {Fn: tableMethodColumnProps, Arity: "1", ...}`
2. ✅ Extract column name from argument (required string)
3. ✅ Build result dictionary:
   - `name` — column name
   - `label` — from schema title or titlecased name (uses existing `toTitleCase`)
   - `type` — schema type (if available)
   - `align` — from alignmentForSchemaType()
   - `format` — from formatHintForSchemaType() (only if non-empty)
4. ✅ Handle tables without schema (minimal props)

Tests:
- ✅ `table([{name: "Alice"}]).columnProps("name")` → `{name: "name", label: "Name", align: "left"}`
- ✅ `table([{created_at: "..."}]).columnProps("created_at")` → `{..., label: "Created At", ...}`
- ✅ Schema-bound table with money → `{align: "right", format: "currency", type: "money", ...}`
- ✅ Schema-bound table with boolean → `{align: "center", format: "boolean", type: "boolean", ...}`

---

### Task 4.3: Update pars describe
**Files**: `pkg/parsley/evaluator/describe.go` or equivalent
**Estimated effort**: 30 min
**Status**: ⏸️ Deferred (auto-registered via MethodRegistry)

Steps:
1. Method auto-registered in TableMethodRegistry with description
2. `pars describe table` should show columnProps automatically

Tests:
- `pars describe table` shows columnProps method (verify after implementation)

---

## Phase 5: Documentation (1-2 hours)

### Task 5.1: Update form binding documentation
**Files**: `docs/parsley/manual/builtins/record.md`
**Estimated effort**: 45 min
**Status**: ✅ Complete

Steps:
1. ✅ Document four abstraction levels (Level 1-4)
2. ✅ Add examples for each level
3. ✅ Explain when to use each level
4. ✅ Add fieldProps() usage examples
5. ✅ Document `<field/>` tag with all props and examples

---

### Task 5.2: Add migration guide
**Files**: `docs/parsley/CHANGES.md`
**Estimated effort**: 30 min
**Status**: ✅ Complete

Steps:
1. ✅ Document typed value formatting change (money in templates)
2. ✅ Explain how to access raw/ISO formats
3. ✅ Note that datetime/duration/unit formatting unchanged
4. ✅ Document new `<field/>` tag and `fieldProps()` method
5. ✅ Document new `columnProps()` method

---

### Task 5.3: Document columnProps() for DataTable
**Files**: `docs/parsley/manual/builtins/table.md`
**Estimated effort**: 30 min
**Status**: ✅ Complete (done in previous commit)

Steps:
1. ✅ Document columnProps() method
2. ✅ Show examples with and without schema
3. ✅ Document alignment and format mappings

---

## Validation Checklist

- [x] All tests pass: `go test ./pkg/parsley/...`
- [ ] Build succeeds: `make build`
- [ ] Benchmarks checked: `make bench-compare`
- [x] Documentation updated
- [x] `pars describe record` shows fieldProps
- [x] `pars describe table` shows columnProps
- [x] All FEAT-145 acceptance criteria checked
- [x] FEAT-144 unblocked (Parts A and C complete)

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-06-15 | Task 1.1: objectToTemplateString() | ✅ Complete | Money only; datetime/duration/unit deferred |
| 2026-06-15 | Task 1.2: objectToPrintString() | ✅ Complete | Money only; consistent with 1.1 |
| 2026-06-15 | Task 1.3: Verify raw access | ✅ Complete | All existing tests pass |
| 2026-06-15 | Task 2.1: Type mapping helpers | ✅ Complete | inputTypeForSchemaType() |
| 2026-06-15 | Task 2.2: recordFieldProps() | ✅ Complete | Full implementation with overrides |
| 2026-06-15 | Task 2.3: Value formatting | ✅ Complete | formatValueForInput() |
| 2026-06-15 | Task 2.4: pars describe record | ✅ Complete | Auto-registered via MethodRegistry |
| 2026-06-15 | Task 3.1: Field tag handler | ✅ Complete | evalFieldTag in form_components.go |
| 2026-06-15 | Task 3.2: Field output structure | ✅ Complete | Full structure with all props |
| 2026-06-15 | Task 3.3: Checkbox special case | ✅ Complete | Input before label, field--checkbox |
| 2026-06-15 | Task 3.4: Wire into eval_tags | ✅ Complete | Done as part of 3.1 |
| 2026-06-15 | Task 4.1: Alignment/format helpers | ✅ Complete | alignmentForSchemaType, formatHintForSchemaType |
| 2026-06-15 | Task 4.2: tableColumnProps() | ✅ Complete | Full implementation with schema support |
| 2026-06-15 | Task 4.3: pars describe table | ⏸️ Deferred | Auto-registered via MethodRegistry |
| 2026-06-15 | Task 5.1: Form binding docs | ✅ Complete | Added abstraction levels, `<field/>` docs |
| 2026-06-15 | Task 5.2: Migration guide | ✅ Complete | Added to docs/parsley/CHANGES.md |
| 2026-06-15 | Task 5.3: columnProps docs | ✅ Complete | Done in previous commit |

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