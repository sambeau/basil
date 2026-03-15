---
id: FEAT-145
title: "Typed Value Formatting and Field Abstraction"
status: approved
priority: high
created: 2026-06-15
updated: 2026-06-15
author: "@human"
blocks: FEAT-144
plan: PLAN-124
---

# FEAT-145: Typed Value Formatting and Field Abstraction

## Summary

Implement three related improvements to Parsley's handling of typed values and form fields:

1. **Part A: Typed Value Formatting** — Change `objectToString()` to produce human-readable output for `money`, `datetime`, `unit`, and `duration` types by calling `.medium()` automatically.

2. **Part B: Form Field Abstraction** — Add `record.fieldProps(name)` method and `<field/>` tag for progressive form binding, from most terse to most flexible.

3. **Part C: Table Column Props** — Add `table.columnProps(name)` method to derive display metadata (label, type, alignment, format) from schemas for DataTable integration.

## User Story

As a developer building web UIs, I want typed values like money and dates to render human-readable by default, and I want a layered approach to form binding where simple forms are simple and complex forms remain possible.

## Problem Statement

**Typed Value Formatting:**
- Currently, typed values in template output show programmer-friendly formats: `2025-03-15` instead of "Mar 15, 2025", raw money amounts without thousands separators
- Users must explicitly call `.medium()` on every typed value for human-readable output
- This is tedious and error-prone

**Form Field Abstraction:**
- The `@field` attribute system works well but requires verbose markup for standard forms
- Component library authors need a bridge to extract schema metadata without reimplementing the logic
- No single-tag solution exists for rapid prototyping of standard forms

**Table Column Props:**
- DataTable needs to derive column metadata (headers, alignment) from schemas
- Currently requires ad-hoc schema lookups
- Should parallel the `fieldProps()` pattern for consistency

## Acceptance Criteria

### Part A: Typed Value Formatting

- [x] `objectToString()` calls `.medium()` on `money` values → "£ 4,999.00"
- [ ] ~~`objectToString()` calls `.medium()` on `datetime` values~~ — **Deferred**: `.medium()` doesn't respect datetime kinds
- [ ] ~~`objectToString()` calls `.medium()` on `duration` values~~ — **Deferred**: `.medium()` returns relative time
- [ ] ~~`objectToString()` calls `.medium()` on `unit` values~~ — **Deferred**: users prefer existing format
- [x] `objectToPrintString()` uses the same formatting (Money only)
- [x] Affects tag content interpolation: `<td>{price}</td>`
- [x] Affects `Table.toHTML()` cell rendering (for Money)
- [x] Affects string concatenation: `"Price: " + price`
- [x] Raw/ISO formats remain accessible via `.iso`, `.inspect()`, `.short()`

**Implementation Notes (Part A)**:
- Only Money formatting was implemented in this phase
- Datetime: `.medium()` doesn't handle datetime kinds (date-only, time-only, full datetime) — needs separate fix
- Duration: `.medium()` returns relative time ("tomorrow") not absolute ("1 day") — not suitable for string coercion
- Unit: `.medium()` adds decimal places ("12.00m" vs "12m") — existing format preferred
- These types retain their existing string conversion behavior until their `.medium()` methods are fixed

### Part B: `fieldProps()` Method

- [x] `record.fieldProps(name)` returns a dictionary of form field props
- [x] Returns `name` — field name for HTML name attribute
- [x] Returns `type` — HTML input type derived from schema type
- [x] Returns `label` — from schema `.title()` or titlecased field name
- [x] Returns `placeholder` — from schema `.placeholder()` if available
- [x] Returns `value` — current value formatted for input (ISO for dates, decimal for money)
- [x] Returns `required` — boolean from schema constraint
- [x] Returns `error` — error message from `.error()` if present
- [x] Returns `autocomplete` — HTML autocomplete hint from type
- [x] Returns `inputmode` — HTML inputmode hint from type
- [x] Returns `options` — array of values for enum types
- [x] Accepts optional second argument for overrides: `fieldProps("email", {label: "Work Email"})`
- [x] Type mappings: email→email, url→url, phone→tel, integer→number, date→date, datetime→datetime-local, boolean→checkbox, enum→select

### Part B: `<field/>` Tag

- [x] `<field name="email"/>` outputs complete field structure (div > label + input + error)
- [x] Requires `@record` context (error if not inside `<form @record={...}>`)
- [x] `name` prop is required
- [x] `as` prop overrides input type: `as="textarea"`, `as="select"`
- [x] `class` prop sets wrapper div class (default: `"field"`)
- [x] `id` prop overrides input id
- [x] `label` prop overrides label text
- [x] `placeholder` prop overrides placeholder
- [x] `help` prop adds help text span
- [x] Boolean fields render checkbox before label (special ordering)
- [x] Boolean fields add `field--checkbox` class to wrapper
- [x] Error span only rendered when field has error
- [x] Error span has `role="alert"` for accessibility
- [x] All ARIA attributes applied correctly (`aria-required`, `aria-invalid`, `aria-describedby`)

### Part C: `columnProps()` Method

- [x] `table.columnProps(name)` returns a dictionary of column display props
- [x] Returns `name` — column identifier
- [x] Returns `label` — from schema `.title()` or titlecased column name
- [x] Returns `type` — original schema type
- [x] Returns `align` — derived from type: right for numeric, center for boolean, left for text
- [x] Returns `format` — format hint: "currency", "date", "datetime", "duration", "unit", "boolean"
- [x] Works on tables without schema (minimal props: name, label, align=left)
- [x] Alignment mapping: money/integer/float/duration/unit → right, boolean → center, others → left

### Documentation

- [ ] Form binding documentation updated with all four levels
- [ ] Migration guide for typed value formatting changes
- [ ] `pars describe record` shows `fieldProps` method
- [ ] `pars describe table` shows `columnProps` method
- [ ] Component library integration guide for `fieldProps()` pattern

## Design Decisions

1. **Human-readable by default**: Since Basil is pre-1.0, we can change defaults. Templates should produce readable output without explicit formatting calls.

2. **Four abstraction levels**: Level 4 (`<field/>`), Level 3 (`@field`), Level 2 (`fieldProps()`), Level 1 (manual). Each serves a distinct use case without overlap.

3. **No `<Field>` wrapper component**: A wrapper component adds nothing over a plain `<div class="field">`. The value is in `<field/>` (outputs everything) or `@field` (full control), not a hybrid.

4. **`fieldProps()` for library authors**: The primary use case is inside component implementations, not at call sites. It bridges schema metadata to custom components.

5. **`columnProps()` parallels `fieldProps()`**: Same schema metadata, different UI concerns. Forms need input types; tables need alignment and format hints.

6. **Value formatting differs by context**: Inputs need ISO/raw formats for submission; display needs `.medium()` for readability.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Design Document

See `work/design/DESIGN-typed-value-formatting.md` for full design rationale, implementation code, and test cases.

### Affected Files

| File | Change |
|------|--------|
| `pkg/parsley/evaluator/eval_string_conversions.go` | Modify `objectToString()`, `objectToPrintString()` |
| `pkg/parsley/evaluator/methods_record.go` | Add `fieldProps` method |
| `pkg/parsley/evaluator/form_field_tag.go` | New file — `<field/>` tag handler |
| `pkg/parsley/evaluator/eval_tags.go` | Wire in `<field/>` tag |
| `pkg/parsley/evaluator/methods_table.go` | Add `columnProps` method |

### Dependencies

- Depends on: None (foundational feature)
- Blocks: FEAT-144 (DataTable Redesign) — requires Part A and Part C
- Related: FEAT-051 (Standard Prelude), existing `@field` attribute system

### Part A: objectToString() Implementation

```go
// pkg/parsley/evaluator/eval_string_conversions.go

func objectToString(obj Object) string {
    switch v := obj.(type) {
    case *Money:
        result := moneyMedium(v, nil)
        if str, ok := result.(*String); ok {
            return str.Value
        }
        return v.Inspect()
    case *Datetime:
        result := datetimeMedium(v, nil, nil)
        if str, ok := result.(*String); ok {
            return str.Value
        }
        return v.Inspect()
    case *Unit:
        result := unitMedium(v, nil)
        if str, ok := result.(*String); ok {
            return str.Value
        }
        return v.Inspect()
    case *Duration:
        result := durationMedium(v, nil)
        if str, ok := result.(*String); ok {
            return str.Value
        }
        return v.Inspect()
    // ... existing cases unchanged
    }
}
```

### Part B: fieldProps() Return Value

| Key | Type | Source | Description |
|-----|------|--------|-------------|
| `name` | string | field name | HTML `name` attribute |
| `type` | string | schema type → mapping | HTML `type` or `"select"` for enums |
| `label` | string | `.title()` or field name | Display label |
| `placeholder` | string | `.placeholder()` | Input placeholder |
| `value` | any | record data | Current value (formatted for input) |
| `required` | boolean | schema constraint | Whether required |
| `error` | string | `.error()` | Error message if present |
| `autocomplete` | string | type/metadata | HTML `autocomplete` hint |
| `inputmode` | string | type/metadata | HTML `inputmode` hint |
| `options` | array | `.enumValues()` | For enums, the allowed values |

### Part B: Type Mappings

| Schema Type | `type` | `inputmode` | `autocomplete` |
|-------------|--------|-------------|----------------|
| `string` | `text` | — | — |
| `email` | `email` | `email` | `email` |
| `url` | `url` | `url` | `url` |
| `phone` | `tel` | `tel` | `tel` |
| `integer` | `number` | `numeric` | — |
| `number`/`float` | `text` | `decimal` | — |
| `boolean` | `checkbox` | — | — |
| `money` | `text` | `decimal` | — |
| `date` | `date` | — | — |
| `datetime` | `datetime-local` | — | — |
| `unit` | `text` | `decimal` | — |
| `enum(...)` | `select` | — | — |

### Part B: `<field/>` Output Structure

```html
<!-- <field name="email"/> outputs: -->
<div class="field">
    <label for="email">Email</label>
    <input type="email" name="email" id="email" value="..." 
           required aria-required="true" aria-invalid="false" autocomplete="email"/>
    <span id="email-error" class="error" role="alert">Error message here</span>
</div>

<!-- Boolean field (<field name="active"/>) outputs: -->
<div class="field field--checkbox">
    <input type="checkbox" name="active" id="active" checked/>
    <label for="active">Active</label>
    <span id="active-error" class="error" role="alert">...</span>
</div>
```

### Part C: columnProps() Return Value

| Key | Type | Source | Description |
|-----|------|--------|-------------|
| `name` | string | column name | Column identifier |
| `label` | string | `.title()` or titlecased name | Display header |
| `type` | string | schema type | Original schema type |
| `align` | string | derived from type | `"left"`, `"right"`, or `"center"` |
| `format` | string | derived from type | Format hint for display |

### Part C: Type to Alignment Mapping

| Schema Type | Alignment | Rationale |
|-------------|-----------|-----------|
| `money` | right | Numeric, decimal alignment |
| `integer` | right | Numeric |
| `float` | right | Numeric |
| `duration` | right | Numeric-like |
| `unit` | right | Numeric with unit |
| `boolean` | center | Binary state |
| `string` | left | Text |
| `email` | left | Text |
| `url` | left | Text |
| `date` | left | Text-like |
| `datetime` | left | Text-like |
| `enum` | left | Text |
| (unknown) | left | Default |

### Test Cases

**Part A: Typed Value Formatting**

```go
func TestTypedValueFormatting(t *testing.T) {
    tests := []struct{
        input    string
        expected string
    }{
        {`<td>{money(499900, "GBP")}</td>`, `<td>£4,999.00</td>`},
        {`<td>{datetime("2025-03-15T14:30:00")}</td>`, `<td>Mar 15, 2025, 2:30 PM</td>`},
        {`<td>{duration(9000)}</td>`, `<td>2 hours 30 minutes</td>`},
        {`<td>{unit(5, "kg")}</td>`, `<td>5.00 kg</td>`},
    }
}
```

**Part B: `<field/>` Tag**

```go
func TestFieldTag(t *testing.T) {
    tests := []struct{
        name     string
        input    string
        contains []string
    }{
        {
            name: "basic email field",
            input: `
                @schema User { email: email | {title: "Email"} }
                let user = User({email: "test@example.com"})
                <form @record={user}><field name="email"/></form>
            `,
            contains: []string{
                `<div class="field">`,
                `<label for="email">Email</label>`,
                `type="email"`,
                `name="email"`,
                `value="test@example.com"`,
            },
        },
        {
            name: "checkbox field ordering",
            input: `
                @schema User { active: boolean }
                let user = User({active: true})
                <form @record={user}><field name="active"/></form>
            `,
            contains: []string{
                `class="field field--checkbox"`,
                `<input type="checkbox"`,  // Before label
            },
        },
    }
}
```

**Part C: `columnProps()`**

```go
func TestTableColumnProps(t *testing.T) {
    tests := []struct{
        name     string
        input    string
        expected map[string]interface{}
    }{
        {
            name: "money column with schema",
            input: `
                @schema Order { total: money | {title: "Order Total"} }
                let Orders = @DB.bind(Order, "orders")
                let orders = Orders.all()
                orders.columnProps("total")
            `,
            expected: map[string]interface{}{
                "name":   "total",
                "label":  "Order Total",
                "type":   "money",
                "align":  "right",
                "format": "currency",
            },
        },
        {
            name: "table without schema",
            input: `
                let t = table([{some_column: 123}])
                t.columnProps("some_column")
            `,
            expected: map[string]interface{}{
                "name":  "some_column",
                "label": "Some Column",
                "align": "left",
            },
        },
    }
}
```

### Implementation Steps

**Phase 1: Typed Value Formatting (2-3 hours)**
1. Modify `objectToString()` in `eval_string_conversions.go`
2. Modify `objectToPrintString()`
3. Add tests for string coercion contexts
4. Verify `.iso`, `.short()`, `.inspect()` alternatives work

**Phase 2: `fieldProps()` Method (3-4 hours)**
1. Implement `recordFieldProps()` in `methods_record.go`
2. Add type mapping helper functions
3. Register method
4. Add comprehensive tests
5. Update `pars describe record`

**Phase 3: `<field/>` Tag (3-4 hours)**
1. Create `form_field_tag.go`
2. Implement `evalFieldTag()`
3. Wire into tag evaluation in `eval_tags.go`
4. Add tests for all field types and props
5. Add tests for checkbox special case

**Phase 4: `columnProps()` Method (2-3 hours)**
1. Implement `tableColumnProps()` in `methods_table.go`
2. Add `alignmentForType()` and `formatForType()` helpers
3. Register method on Table type
4. Add tests for schema-bound and raw tables
5. Update `pars describe table`

**Phase 5: Documentation (1-2 hours)**
1. Update form binding documentation with all four levels
2. Add migration guide
3. Document `fieldProps()` and `columnProps()` patterns

### Effort Estimate

| Phase | Effort |
|-------|--------|
| Phase 1: Typed Value Formatting | 2-3 hours |
| Phase 2: `fieldProps()` Method | 3-4 hours |
| Phase 3: `<field/>` Tag | 3-4 hours |
| Phase 4: `columnProps()` Method | 2-3 hours |
| Phase 5: Documentation | 1-2 hours |
| **Total** | **11-16 hours** |

### Migration Notes

**For Users:**
- If relying on ISO format in output, use `.iso`: `<time datetime={date.iso}>`
- If need raw money, use `.inspect()` or explicit formatting
- Existing `@field` code continues to work unchanged
- New `<field/>` tag available for simpler forms

**For Library Authors:**
- Use `fieldProps()` inside component implementations
- Accept `record` + `field` props for schema-aware mode
- Fall back to manual props for non-schema usage

## Related

- Design doc: `work/design/DESIGN-typed-value-formatting.md`
- Standard Prelude Review: `work/reports/STANDARD-PRELUDE-REVIEW.md` §9.3, §9.5
- Blocks: FEAT-144 (DataTable Redesign) — requires Parts A and C
- Parent feature: FEAT-051 (Standard Prelude)
- Existing system: `@field` attribute handling in `eval_form_context.go`
