---
id: PLAN-122
feature: FEAT-143
title: "Implementation Plan: Prelude Component Styling Strategy"
status: draft
created: 2026-06-15
---

# Implementation Plan: FEAT-143

## Overview

Implement a Pico CSS-compatible styling strategy for Prelude components. This involves creating new components (Dialog, Details, Accordion, Toast, Pagination, ErrorSummary), updating existing components to output semantic HTML without embedded CSS, and documenting the recommended styling approach.

## Prerequisites

- [x] Spec reviewed and approved: `work/specs/FEAT-143-prelude-component-styling.md`
- [x] Design document complete: `work/design/DESIGN-prelude-pico-compatibility.md`
- [x] Supplement CSS created: `examples/css/basil-supplement.css`
- [x] Supplement README created: `examples/css/README.md`

## Phase 1: Foundation (No Breaking Changes)

### Task 1.1: Create Details component
**Files**: `server/prelude/components/details.pars` (new)
**Estimated effort**: 15 min

Steps:
1. Create `details.pars` with `Details` component
2. Props: `title` (required), `open` (default false), `name` (optional for accordion grouping), `contents`
3. Output: `<details>` with `<summary>` containing title
4. Export from prelude

Tests:
- Basic details renders correctly
- `open` attribute works
- `name` attribute passes through for accordion behavior
- Additional attrs spread correctly

---

### Task 1.2: Create Accordion component
**Files**: `server/prelude/components/accordion.pars` (new)
**Estimated effort**: 20 min

Steps:
1. Create `accordion.pars` with `Accordion` component
2. Props: `name` (required), `items` (array of `{title, content, open?}`)
3. Output: Multiple `<details name={name}>` elements
4. First item open by default unless specified
5. Export from prelude

Tests:
- Renders multiple details with same `name`
- First item is open by default
- Custom `open` in items works
- Empty items array returns null

---

### Task 1.3: Create Dialog component
**Files**: `server/prelude/components/dialog.pars` (new)
**Estimated effort**: 25 min

Steps:
1. Create `dialog.pars` with `Dialog` component
2. Props: `id` (required), `title` (optional), `footer` (optional), `contents`
3. Output: `<dialog><article>` with optional `<header>` and `<footer>`
4. Close button uses `rel="prev"` and inline onclick for Pico compatibility
5. Export from prelude

Tests:
- Basic dialog renders with article wrapper
- Title creates header with close button
- Footer renders in footer element
- Dialog without title has no header
- Id attribute is required and passes through

---

### Task 1.4: Update prelude exports for Phase 1 components
**Files**: `server/prelude/prelude.go` or equivalent export file
**Estimated effort**: 15 min

Steps:
1. Add `Details` to prelude exports
2. Add `Accordion` to prelude exports
3. Add `Dialog` to prelude exports
4. Verify components are accessible via `@prelude`

Tests:
- `import @prelude` provides Details, Accordion, Dialog
- No naming conflicts with existing exports

---

### Task 1.5: Document Pico CSS setup
**Files**: `docs/guide/styling.md` (new or update existing)
**Estimated effort**: 30 min

Steps:
1. Create/update styling guide with Pico CSS recommendation
2. Document minimal setup (classless)
3. Document recommended setup (with classes)
4. Document supplement CSS usage
5. Show example Page component with Pico links

Tests:
- Documentation is accurate
- Code examples work when copy-pasted

---

## Phase 2: Migrate Existing Components

### Task 2.1: Update TextField to use `<small>` for hints/errors
**Files**: `server/prelude/components/text_field.pars`
**Estimated effort**: 20 min

Steps:
1. Change hint output from `<p class="field-hint">` to `<small>`
2. Change error output from `<p class="field-error">` to `<small>`
3. Remove wrapper div class (`.field`)
4. Keep all ARIA attributes (`aria-invalid`, `aria-describedby`, `aria-required`)
5. Test with Pico CSS

Tests:
- Hint renders as `<small>` with correct id
- Error renders as `<small>` with correct id
- `aria-describedby` still references correct ids
- Pico inherits validation color from `aria-invalid`

---

### Task 2.2: Update TextareaField to match TextField pattern
**Files**: `server/prelude/components/textarea_field.pars`
**Estimated effort**: 15 min

Steps:
1. Apply same changes as TextField (small for hints/errors)
2. Remove custom classes
3. Keep ARIA attributes

Tests:
- Mirrors TextField behavior
- Works with Pico CSS

---

### Task 2.3: Update SelectField to match TextField pattern
**Files**: `server/prelude/components/select_field.pars`
**Estimated effort**: 15 min

Steps:
1. Apply same changes as TextField (small for hints/errors)
2. Remove custom classes
3. Keep ARIA attributes

Tests:
- Mirrors TextField behavior
- Works with Pico CSS

---

### Task 2.4: Update SkipLink to remove inline CSS
**Files**: `server/prelude/components/skip_link.pars`
**Estimated effort**: 15 min

Steps:
1. Remove inline `<style>` tag entirely
2. Add `class="skip-link"` to the anchor element
3. Verify supplement CSS has matching `.skip-link` styles

Tests:
- No `<style>` tag in output
- Has `class="skip-link"`
- Works correctly with supplement CSS (hidden until focused)

---

### Task 2.5: Update Breadcrumb for Pico compatibility
**Files**: `server/prelude/components/breadcrumb.pars`
**Estimated effort**: 20 min

Steps:
1. Change `aria-label="Breadcrumb"` to lowercase (Pico convention)
2. Remove custom classes (`.breadcrumb`, `.breadcrumb-list`, `.breadcrumb-item`)
3. Ensure structure is `<nav aria-label="breadcrumb"><ul><li>...`
4. Keep `aria-current="page"` on current item

Tests:
- Uses Pico-compatible aria-label
- Works styled with Pico CSS
- Current page still marked correctly

---

### Task 2.6: Update Page to fix body id default
**Files**: `server/prelude/components/page.pars`
**Estimated effort**: 10 min

Steps:
1. Change `id` prop default from `"main"` to `null`
2. Document that users should put `id="main"` on their `<main>` element
3. Skip link still defaults to `#main` target

Tests:
- Body has no id by default
- Custom id still works when provided
- Documentation updated

---

### Task 2.7: Update Form to remove .form class (optional)
**Files**: `server/prelude/components/form.pars`
**Estimated effort**: 5 min

Steps:
1. Remove `.form` class from form element
2. Pico styles `<form>` directly

Tests:
- Form still works without class
- Pico styles apply correctly

---

## Phase 3: New Components

### Task 3.1: Create Toast component
**Files**: `server/prelude/components/toast.pars` (new)
**Estimated effort**: 20 min

Steps:
1. Create `toast.pars` with `Toast` component
2. Props: `message` (required), `type` (default "info"), `dismissible` (default true)
3. Output: `<article role="status|alert" data-type={type}>`
4. Use `role="alert"` for error type, `role="status"` for others
5. Dismiss button uses inline onclick
6. Export from prelude

Tests:
- Info toast has `role="status"`
- Error toast has `role="alert"`
- `data-type` attribute set correctly
- Dismiss button works
- Non-dismissible toast has no button

---

### Task 3.2: Create Toasts container component
**Files**: `server/prelude/components/toasts.pars` (new)
**Estimated effort**: 15 min

Steps:
1. Create `toasts.pars` with `Toasts` component
2. Props: `position` (default "top-right"), `contents`
3. Output: `<aside id="toasts" aria-live="polite" aria-label="Notifications" data-position={position}>`
4. Export from prelude

Tests:
- Container has correct ARIA attributes
- Position passed as data-position
- Contents render inside

---

### Task 3.3: Create Pagination component
**Files**: `server/prelude/components/pagination.pars` (new)
**Estimated effort**: 45 min

Steps:
1. Create `pagination.pars` with `Pagination` component
2. Props: `current`, `total`, `perPage`, `href`, `window`, `showFirst`, `showPrev`, `labels`
3. Import `max`, `min` from `@std/math`
4. Calculate total pages
5. Output semantic nav with `aria-label="Pagination"`
6. Use `aria-current="page"` for current page
7. Use `aria-hidden="true"` for ellipsis
8. Use `aria-label` on prev/next/first/last buttons
9. Export from prelude

Tests:
- Returns null for single page
- Current page marked with aria-current
- Ellipsis rendered with aria-hidden
- Window calculation correct
- First/last buttons conditional
- URL template replacement works

---

### Task 3.4: Create ErrorSummary component
**Files**: `server/prelude/components/error_summary.pars` (new)
**Estimated effort**: 25 min

Steps:
1. Create `error_summary.pars` with `ErrorSummary` component
2. Props: `title` (default "There is a problem"), `errors` (array of {field, message}), `id`
3. Output: `<aside role="alert" tabindex="-1">` with header and ul
4. Links to field ids via `href={"#" ++ err.field}`
5. Return null for empty/null errors
6. Export from prelude

Tests:
- Returns null for empty errors
- Returns null for null errors
- Links point to correct field ids
- Has role="alert" and tabindex="-1"
- Title is customizable

---

### Task 3.5: Update prelude exports for Phase 3 components
**Files**: `server/prelude/prelude.go` or equivalent export file
**Estimated effort**: 10 min

Steps:
1. Add `Toast` to prelude exports
2. Add `Toasts` to prelude exports
3. Add `Pagination` to prelude exports
4. Add `ErrorSummary` to prelude exports

Tests:
- All new components accessible via `@prelude`

---

## Phase 4: Testing and Documentation

### Task 4.1: Add component integration tests
**Files**: `pkg/parsley/tests/prelude_components_test.go` or similar
**Estimated effort**: 1 hour

Steps:
1. Add tests for Dialog HTML output
2. Add tests for Details/Accordion HTML output
3. Add tests for Toast/Toasts HTML output
4. Add tests for Pagination HTML output (multiple scenarios)
5. Add tests for ErrorSummary HTML output
6. Verify ARIA attributes in all tests

Tests:
- All new components have test coverage
- Edge cases tested (empty arrays, null values)
- ARIA attributes verified

---

### Task 4.2: Create example page demonstrating all components
**Files**: `examples/prelude-components/` (new directory)
**Estimated effort**: 45 min

Steps:
1. Create example app showing all Pico-styled components
2. Include Dialog, Accordion, Toasts, Pagination, ErrorSummary
3. Include form with TextField, TextareaField, SelectField
4. Show both success and error states
5. Document how to run the example

Tests:
- Example runs successfully
- All components render correctly with Pico CSS
- Demonstrates supplement CSS usage

---

### Task 4.3: Document all new components
**Files**: `docs/prelude/components/` (multiple files)
**Estimated effort**: 1.5 hours

Steps:
1. Document Dialog: props, examples, accessibility notes
2. Document Details/Accordion: props, examples, native behavior
3. Document Toast/Toasts: props, examples, positioning
4. Document Pagination: props, examples, URL templates
5. Document ErrorSummary: props, examples, focus management
6. Update component index

Tests:
- All props documented
- Examples are runnable
- Accessibility considerations noted

---

### Task 4.4: Add migration guide
**Files**: `docs/guide/migration/pico-styling.md` (new)
**Estimated effort**: 30 min

Steps:
1. Document changes to TextField/TextareaField/SelectField
2. Document SkipLink CSS requirement
3. Document Page body id change
4. Document Breadcrumb aria-label change
5. Provide before/after examples

Tests:
- Migration steps are clear
- Breaking changes clearly identified

---

### Task 4.5: Update FAQ
**Files**: `docs/guide/faq.md`
**Estimated effort**: 15 min

Steps:
1. Add "How do I style Prelude components?" entry
2. Reference Pico CSS recommendation
3. Link to styling guide and supplement CSS

Tests:
- FAQ entry is helpful and accurate

---

## Validation Checklist

- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `make build`
- [ ] New components render correctly
- [ ] Existing components still work (backward compatible where possible)
- [ ] Components work unstyled
- [ ] Components work with Pico classless
- [ ] Components work with Pico + classes
- [ ] Components work with Pico + supplement
- [ ] ARIA attributes correct on all components
- [ ] Documentation complete for all new components
- [ ] Migration guide complete
- [ ] FAQ updated
- [ ] No performance regressions: `make bench-compare`

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-06-15 | Task 1.1: Create Details component | ✅ Complete | Native HTML5 `<details>` |
| 2026-06-15 | Task 1.2: Create Accordion component | ✅ Complete | Uses `name` attribute for exclusive behavior |
| 2026-06-15 | Task 1.3: Create Dialog component | ✅ Complete | Pico-compatible `<dialog><article>` pattern |
| 2026-06-15 | Task 1.4: Update prelude exports (Phase 1) | ✅ Complete | Details, Accordion, Dialog registered |
| 2026-06-15 | Task 2.1: Update TextField | ✅ Complete | Uses `<small>` for hints/errors |
| 2026-06-15 | Task 2.2: Update TextareaField | ✅ Complete | Same pattern as TextField |
| 2026-06-15 | Task 2.3: Update SelectField | ✅ Complete | Same pattern as TextField |
| 2026-06-15 | Task 2.4: Update SkipLink | ✅ Complete | Removed inline CSS, uses class |
| 2026-06-15 | Task 2.5: Update Breadcrumb | ✅ Complete | Lowercase aria-label, removed classes |
| 2026-06-15 | Task 2.6: Update Page | ✅ Complete | Removed default body id |
| 2026-06-15 | Task 2.7: Update Form | ✅ Complete | Removed .form class |
| 2026-06-15 | Task 3.1: Create Toast component | ✅ Complete | role="status\|alert", data-type |
| 2026-06-15 | Task 3.2: Create Toasts container | ✅ Complete | aria-live="polite", data-position |
| 2026-06-15 | Task 3.3: Create Pagination component | ✅ Complete | Window-based, aria-current |
| 2026-06-15 | Task 3.4: Create ErrorSummary component | ✅ Complete | role="alert", tabindex="-1" |
| 2026-06-15 | Task 3.5: Update prelude exports (Phase 3) | ✅ Complete | All 4 components registered |
| | Task 4.1: Add component integration tests | ⏳ Pending | |
| | Task 4.2: Create example page | ⏳ Pending | |
| | Task 4.3: Document all new components | ⏳ Pending | |
| | Task 4.4: Add migration guide | ⏳ Pending | |
| | Task 4.5: Update FAQ | ⏳ Pending | |

## Deferred Items

Items to add to `work/BACKLOG.md` after implementation:

- Optional modal animation JS — Pico examples include animation helpers
- Toast auto-dismiss JS — Timer-based dismissal for toasts
- ErrorSummary auto-focus JS — Focus summary on form validation failure
- Pagination with parts — Document how to use `.parts` files for partial page updates when paginating