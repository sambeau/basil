---
id: PLAN-036
feature: FEAT-058
title: "Implementation Plan for HTML Components in Prelude"
status: deferred
created: 2025-12-09
---

# Implementation Plan: FEAT-058 HTML Components in Prelude

## Overview
Move HTML component implementations from potential future Go code to Parsley files in the prelude. The `std/html` module will load components from `prelude/components/`, making them human-editable and easy to extend.

## Prerequisites
- [x] FEAT-056: Prelude Infrastructure (completed)
- [ ] Decision: Which components to include in MVP

## Tasks

### Task 1: Create Component Directory Structure
**Files**: `server/prelude/components/*.pars`
**Estimated effort**: Small

Steps:
1. Create `prelude/components/` directory
2. Update embed directive to include `prelude/components/*`
3. Ensure component files are parsed at startup

Tests:
- Component ASTs available via `GetPreludeAST("components/...")`

---

### Task 2: Implement TextField Component
**Files**: `server/prelude/components/text_field.pars`
**Estimated effort**: Small

Steps:
1. Create `TextField(props)` function
2. Support props: `name`, `label`, `type`, `value`, `hint`, `error`, `required`, `placeholder`
3. Generate accessible markup with proper ARIA attributes
4. Include label, input, hint text, and error message

Tests:
- Renders input with label
- Required field shows indicator
- Error state shows error message
- ARIA attributes present

---

### Task 3: Implement SelectField Component
**Files**: `server/prelude/components/select_field.pars`
**Estimated effort**: Small

Steps:
1. Create `SelectField(props)` function
2. Support props: `name`, `label`, `options`, `value`, `hint`, `error`, `required`, `placeholder`
3. Options as array of `{value, label}` or simple strings
4. Include accessible markup

Tests:
- Renders select with options
- Selected value marked
- Placeholder option when provided

---

### Task 4: Implement Button Component
**Files**: `server/prelude/components/button.pars`
**Estimated effort**: Small

Steps:
1. Create `Button(props, children)` function
2. Support props: `type`, `variant`, `disabled`, `name`, `value`
3. Variants: `primary`, `secondary`, `danger`
4. Button text from children

Tests:
- Renders button with text
- Type defaults to "button"
- Variant classes applied

---

### Task 5: Implement Form Component
**Files**: `server/prelude/components/form.pars`
**Estimated effort**: Medium

Steps:
1. Create `Form(props, children)` function
2. Support props: `action`, `method`, `confirm`, `enctype`
3. Auto-include CSRF token from `basil.http.csrf`
4. Confirmation dialog via `onsubmit` when `confirm` prop set

Tests:
- Renders form with action/method
- CSRF hidden input included
- Confirm triggers JS dialog

---

### Task 6: Implement CheckboxField Component
**Files**: `server/prelude/components/checkbox_field.pars`
**Estimated effort**: Small

Steps:
1. Create `CheckboxField(props)` function
2. Support props: `name`, `label`, `checked`, `value`, `hint`, `error`
3. Label clickable (wraps or uses `for`)

Tests:
- Renders checkbox with label
- Checked state works
- Value attribute set

---

### Task 7: Implement RadioGroup Component
**Files**: `server/prelude/components/radio_group.pars`
**Estimated effort**: Small

Steps:
1. Create `RadioGroup(props)` function
2. Support props: `name`, `label`, `options`, `value`, `hint`, `error`
3. Options as array of `{value, label}`
4. Fieldset/legend for accessibility

Tests:
- Renders radio buttons in fieldset
- Selected option checked
- Legend shows group label

---

### Task 8: Implement TextAreaField Component
**Files**: `server/prelude/components/textarea_field.pars`
**Estimated effort**: Small

Steps:
1. Create `TextAreaField(props)` function
2. Support props: `name`, `label`, `value`, `hint`, `error`, `required`, `rows`, `placeholder`
3. Similar structure to TextField

Tests:
- Renders textarea with label
- Rows attribute honored
- Value as content

---

### Task 9: Implement DataTable Component
**Files**: `server/prelude/components/data_table.pars`
**Estimated effort**: Medium

Steps:
1. Create `DataTable(props)` function
2. Support props: `data`, `columns`, `sortable`, `emptyMessage`
3. Columns define header and accessor
4. Accessible table markup

Tests:
- Renders table with headers
- Data rows rendered
- Empty state message shown

---

### Task 10: Create std/html Module Loader
**Files**: `pkg/parsley/evaluator/stdlib_html.go`
**Estimated effort**: Medium

Steps:
1. Create `loadHTMLModule()` function
2. Load each component from prelude ASTs
3. Extract exported functions by name convention
4. Register as `std/html` import

Tests:
- `import @std/html` works
- Components accessible as `html.TextField`, etc.
- Components callable with props

---

### Task 11: Add Documentation
**Files**: `docs/guide/components.md` (or similar)
**Estimated effort**: Small

Steps:
1. Document available components
2. Show usage examples
3. List all supported props for each

Tests:
- Documentation accurate

---

## Validation Checklist
- [ ] All tests pass: `make test`
- [ ] Build succeeds: `make build`
- [ ] Each component renders correctly
- [ ] Components work in real forms
- [ ] Accessibility attributes present
- [ ] Documentation complete

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| — | — | — | — |
