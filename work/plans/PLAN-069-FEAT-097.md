---
id: PLAN-069
feature: FEAT-097
title: "Implementation Plan for Form Autocomplete Metadata"
status: draft
created: 2026-01-19
---

# Implementation Plan: FEAT-097

## Overview
Add autocomplete support to form input binding. Auto-derive values from field types and names, with explicit metadata override.

## Prerequisites
- [x] Design document reviewed: `work/design/FORM_AUTOCOMPLETE.md`
- [x] Feature spec created: `work/specs/FEAT-097.md`
- [x] Form binding working (FEAT-091)

## Tasks

### Task 1: Add autocomplete pattern map
**Files**: `pkg/parsley/evaluator/form_autocomplete.go` (new file)
**Estimated effort**: Small

Steps:
1. Create new file `form_autocomplete.go`
2. Define `autocompletePatterns` map (field name → autocomplete value)
3. Add `getAutocomplete(fieldName, fieldType string, metadata map[string]Object) string` function
4. Implement priority: explicit metadata > field name > type > empty

Tests:
- Type-based: `email` → `"email"`, `phone` → `"tel"`, `url` → `"url"`
- Field name: `firstName` → `"given-name"`, `password` → `"current-password"`
- Case-insensitive: `FIRSTNAME`, `FirstName`, `firstname` all → `"given-name"`
- Explicit override wins
- Unknown field returns empty string

---

### Task 2: Integrate with input field binding
**Files**: `pkg/parsley/evaluator/form_binding.go`
**Estimated effort**: Small

Steps:
1. Import/call `getAutocomplete()` in `evalFieldBinding()`
2. Add `autocomplete` attribute to output if value is non-empty
3. Handle both self-closing `<input @field/>` and regular `<input @field>`

Tests:
- `<input @field="email"/>` includes `autocomplete="email"`
- `<input @field="firstName"/>` includes `autocomplete="given-name"`
- `<input @field="unknownField"/>` has no autocomplete attribute
- Explicit metadata `| {autocomplete: "off"}` produces `autocomplete="off"`

---

### Task 3: Support select and textarea
**Files**: `pkg/parsley/evaluator/form_components.go`
**Estimated effort**: Small

Steps:
1. Add autocomplete to `evalSelectComponent()` output
2. Add autocomplete to textarea binding (if exists)
3. Use same `getAutocomplete()` logic

Tests:
- `<select @field="country"/>` with `| {autocomplete: "country-name"}` works
- Select with enum type and autocomplete metadata

---

### Task 4: Add unit tests
**Files**: `pkg/parsley/evaluator/form_autocomplete_test.go` (new file)
**Estimated effort**: Medium

Steps:
1. Create test file
2. Test `getAutocomplete()` function directly
3. Test full pattern map coverage
4. Test edge cases (empty, nil metadata, case variations)

Test cases:
- All type-based defaults
- All field name patterns from spec
- Mixed case field names
- Explicit metadata override
- Explicit "off" value
- Compound values like "shipping street-address"
- Empty/nil metadata handling

---

### Task 5: Update documentation
**Files**: 
- `docs/parsley/manual/builtins/record.md`
- `docs/parsley/reference.md`
- `docs/parsley/CHEATSHEET.md`
**Estimated effort**: Small

Steps:
1. Add "Autocomplete" subsection to record.md Form Binding section
2. Document auto-derivation behavior
3. Document explicit override syntax
4. Add examples (login, registration, checkout)
5. Update reference.md form binding section
6. Add brief note to CHEATSHEET.md

---

## Validation Checklist
- [x] All tests pass: `make test`
- [x] Build succeeds: `make build`
- [x] Linter passes: `golangci-lint run`
- [x] Documentation updated
- [x] work/BACKLOG.md updated with deferrals (if any)

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-01-19 | Task 1: Pattern map | ✅ Complete | Created form_autocomplete.go with 60+ patterns |
| 2026-01-19 | Task 2: Input binding | ✅ Complete | Added to buildInputAttributes() |
| 2026-01-19 | Task 3: Select/textarea | ✅ Complete | Added to evalSelectComponent() and textarea handling |
| 2026-01-19 | Task 4: Unit tests | ✅ Complete | form_autocomplete_test.go + integration tests |
| 2026-01-19 | Task 5: Documentation | ✅ Complete | record.md, reference.md, CHEATSHEET.md |
- Consider `autofocus` metadata support — Related UX feature, separate scope
- Consider `inputmode` metadata support — Related mobile keyboard hints
