# FEAT-143 Implementation Audit Report

**Date:** 2026-06-15
**Auditor:** AI Assistant
**Status:** 🔴 **FAILED** — Critical bugs prevent components from functioning

## Executive Summary

The FEAT-143 implementation has significant issues that prevent the new components from working at runtime. While the structural approach is sound and the design decisions are good, three categories of Parsley syntax errors — verified against the reference manual and `pars -e` — mean that every new component and several updated components will fail when invoked.

**Overall Assessment:**
- ⚠️ **Implementation Completeness:** ~85% (acceptance criteria mostly addressed, tests missing)
- 🔴 **Code Correctness:** Critical — 3 runtime-breaking bugs across all new components
- ❌ **Test Coverage:** None — no component-specific tests were written
- ✅ **Documentation:** Good — styling guide, FAQ, migration guide all present
- ⚠️ **Spec Accuracy:** Spec diverges from implementation and contains invalid Parsley syntax

---

## Section 1: Critical Bugs

### 1.1 🔴 `{...attrs}` is invalid tag spread syntax

**Severity:** Critical — runtime error on every component invocation
**Scope:** All 9 FEAT-143 component files

Parsley's tag attribute spread syntax is `...attrs` (no braces). The `{...attrs}` form causes a runtime error: `unexpected '...'`.

**Reference:**

From `docs/parsley/reference.md` §8.6:

```parsley
let attrs = {class: "btn", id: "submit"}
<button ...attrs>"Submit"</button>
```

**Verification:**

```
$ pars -r -e 'let attrs = {class: "test"}; <div ...attrs>"ok"</div>'
<div class="test">ok</div>

$ pars -r -e 'let attrs = {class: "test"}; <div {...attrs}>"ok"</div>'
Runtime error: unexpected '...'
```

**Affected files (all FEAT-143):**

| File | Line | Code |
|------|------|------|
| `details.pars` | 7 | `<details open={open} name={name} {...attrs}>` |
| `accordion.pars` | 11 | `<details name={name} open={...} {...attrs}>` |
| `dialog.pars` | 7 | `<dialog id={id} {...attrs}>` |
| `toast.pars` | 12 | `<article role={role} data-type={toastType} {...attrs}>` |
| `toasts.pars` | 14 | `{...attrs}` |
| `pagination.pars` | 25 | `<nav aria-label="Pagination" {...attrs}>` |
| `error_summary.pars` | 19 | `{...attrs}` |
| `skip_link.pars` | 8 | `<a href={href} class="skip-link" {...attrs}>` |
| `breadcrumb.pars` | 11 | `{...attrs}` |

**Fix:** Replace `{...attrs}` with `...attrs` in all tag positions.

**Note:** The pre-existing components (`form.pars`, `text_field.pars`, `img.pars`, etc.) correctly use `...inputAttrs` / `...formAttrs` without braces. The `{...attrs}` variant was introduced by FEAT-143.

**Root cause:** The design document (`DESIGN-prelude-pico-compatibility.md`) contains `{...attrs}` throughout its Parsley examples. These were copied into the spec and then into the implementation without verification.

---

### 1.2 🔴 `for (item, i in items)` has swapped parameter order

**Severity:** Critical — runtime error when components are rendered
**Scope:** `accordion.pars` (FEAT-143), plus pre-existing `breadcrumb.pars`, `checkbox_group.pars`, `radio_group.pars`, `data_table.pars`

Parsley's `for` with two parameters is **index first, value second**:

From `docs/parsley/reference.md` §3.2:

```parsley
for (i, n in nums) { `{i}: {n}` }
// ["0: 1", "1: 2", "2: 3", "3: 4", "4: 5"]
```

Confirmed by evaluator source (`pkg/parsley/evaluator/eval_control_flow.go` L139–142):

```go
if paramCount == 2 {
    // Two parameters: index and element
    args = []Object{&Integer{Value: int64(idx)}, elem}
}
```

And by test (`pkg/parsley/tests/for_indexing_test.go` L191–194):

```go
`for(index, value in [7, 8, 9]) { index }`,
`[0, 1, 2]`,
```

**Verification:**

```
$ pars -e 'for (item, i in ["a", "b", "c"]) { {item: item, i: i} }'
[{item: 0, i: "a"}, {item: 1, i: "b"}, {item: 2, i: "c"}]
```

**FEAT-143 affected file:**

| File | Line | Code | Problem |
|------|------|------|---------|
| `accordion.pars` | 10–11 | `for (item, i in items) { ... item.open ?? (i == 0) }` | `item` is index (integer), `item.open` errors; `i` is the dict, `i == 0` errors |

**Pre-existing files with same bug (not introduced by FEAT-143):**

| File | Line | Code | Problem |
|------|------|------|---------|
| `breadcrumb.pars` | 17–18 | `for (item, idx in items) { ... idx == (items.length() - 1) }` | `idx` is a dict, comparison to integer errors |
| `checkbox_group.pars` | 41 | `for (opt, idx in options)` | `opt` is index, `idx` is value — all uses are swapped |
| `radio_group.pars` | 38 | `for (opt, idx in options)` | Same as checkbox_group |
| `data_table.pars` | 11, 19 | `for (col, idx in columns)` / `for (key, idx in keys)` | Same pattern |

**Fix for FEAT-143:** Change `for (item, i in items)` to `for (i, item in items)` in `accordion.pars`.

**Note on pre-existing bugs:** The Breadcrumb was already broken before FEAT-143. FEAT-143 preserved the broken parameter order when updating Breadcrumb and introduced the same pattern in Accordion. All 5 affected files should be fixed.

---

### 1.3 🔴 Pagination range `start..end + 1` — precedence error and off-by-one

**Severity:** Critical — runtime error
**Scope:** `pagination.pars` line 48

```parsley
for (n in start..end + 1) {
```

**Two problems:**

1. **Operator precedence:** `start..end + 1` is parsed as `(start..end) + 1`. The `..` binds tighter than `+`, producing an array, then `+ 1` tries to add an integer to an array → runtime error: `Type mismatch: array + integer`.

2. **Off-by-one (even if parenthesized):** Parsley's `..` is inclusive on both ends (`1..5` → `[1,2,3,4,5]`). Using `start..(end + 1)` would include one page beyond the window. The correct code is `start..end`.

**Verification:**

```
$ pars -e 'let start = 1; let end = 5; for (n in start..end + 1) { n }'
Runtime error: Type mismatch: array + integer

$ pars -e 'let start = 1; let end = 5; for (n in start..(end + 1)) { n }'
[1, 2, 3, 4, 5, 6]

$ pars -e 'let start = 1; let end = 5; for (n in start..end) { n }'
[1, 2, 3, 4, 5]
```

**Fix:** Change `start..end + 1` to `start..end`.

---

## Section 2: Spec and Design Document Accuracy

### 2.1 🟡 Default parameters in function signatures (invalid Parsley)

The spec and design document contain function signatures with default parameter values. Parsley does not support this.

From `docs/parsley/manual/fundamentals/functions.md`:

> Parsley does not support default parameter values. Missing arguments leave the parameter unbound. [...] **No default parameters** — use `??` inside the body if you need defaults: `let x = arg ?? "default"`.

**Spec instances (4):**

| Spec Location | Invalid Code |
|---------------|-------------|
| Details spec | `fn({title, open = false, contents, ...attrs})` |
| Toast spec | `fn({message, type = "info", dismissible = true, ...attrs})` |
| Toasts spec | `fn({position = "top-right", contents, ...attrs})` |
| SkipLink spec | `fn({href = "#main", text = "Skip to main content", ...attrs})` |

**Design doc instances (3):**

| Design Doc Location | Invalid Code |
|--------------------|-------------|
| §4.2 Details | `fn({title, open = false, contents, ...attrs})` |
| §4.5 Toasts | `fn({position = "top-right", contents, ...attrs})` |
| §4.7 SkipLink | `fn({href = "#main", text = "Skip to main content", ...attrs})` |

**Implementation status:** The actual `.pars` files correctly use `??` for defaults inside the function body, so the running code handles this correctly. But the spec and design doc are misleading.

**Fix:** Update spec and design doc examples to use `??` pattern matching the implementations.

### 2.2 🟡 Spec TextField differs from implementation

The spec describes a cleaner destructured signature:

```parsley
// Spec version
export TextField = fn({name, label, type = "text", value, error, hint, required = false, ...attrs}) {
    let inputId = attrs.id ?? name
```

The implementation uses the `fn(props)` style with manual destructuring:

```parsley
// Actual version
export TextField = fn(props) {
    let {name, label, type, value, hint, error, required, id, ...inputAttrs} = props
    let inputId = id ?? name
```

The implementation is superior: the `fn(props)` pattern allows `...inputAttrs` to collect remaining props for spreading onto the `<input>` element. The spec version would also fail due to the default parameters issue.

### 2.3 🟡 Spec SkipLink uses different prop names

The spec specifies `href` and `text` as prop names:

```parsley
export SkipLink = fn({href = "#main", text = "Skip to main content", ...attrs}) {
```

The implementation uses `target` and `text`, preserving backward compatibility:

```parsley
export SkipLink = fn({target, text, ...attrs}) {
    let href = target ?? "#main"
```

The implementation is correct — it avoids a breaking change for existing users who pass `target`.

---

## Section 3: Acceptance Criteria Checklist

### Phase 1: Foundation

| Criterion | Status | Notes |
|-----------|--------|-------|
| `examples/css/basil-supplement.css` exists | ✅ | Covers skip-link, toasts, pagination, error-summary, .sr-only |
| `examples/css/README.md` documents usage | ✅ | Comprehensive |
| Dialog component created | 🔴 | File exists but broken (`{...attrs}`) |
| Details component created | 🔴 | File exists but broken (`{...attrs}`) |
| Accordion component created | 🔴 | File exists but broken (`{...attrs}` + swapped `for` params) |
| Pico CSS setup documented | ✅ | `docs/guide/styling.md` |

### Phase 2: Migrate Existing Components

| Criterion | Status | Notes |
|-----------|--------|-------|
| TextField uses `<small>` | ✅ | Correct, uses `...inputAttrs` (pre-existing correct spread) |
| TextareaField updated | ✅ | Same pattern as TextField |
| SelectField updated | ✅ | Same pattern as TextField |
| Breadcrumb Pico-compatible | 🔴 | `{...attrs}` broken + pre-existing swapped `for` params |
| SkipLink no inline CSS | 🔴 | `{...attrs}` broken |
| Page body id fixed | ✅ | Default changed to `null` |
| Form `.form` class removed | ✅ | Uses pre-existing correct `...formAttrs` |

### Phase 3: New Components

| Criterion | Status | Notes |
|-----------|--------|-------|
| Toast component | 🔴 | `{...attrs}` broken |
| Toasts container | 🔴 | `{...attrs}` broken |
| Pagination component | 🔴 | `{...attrs}` broken + range `start..end + 1` broken |
| ErrorSummary component | 🔴 | `{...attrs}` broken |

### Testing Requirements

| Criterion | Status | Notes |
|-----------|--------|-------|
| Works unstyled | ❌ | Cannot verify — components error at runtime |
| Works with Pico classless | ❌ | Cannot verify |
| Works with Pico + classes | ❌ | Cannot verify |
| Works with Pico + supplement | ❌ | Cannot verify |
| ARIA attributes correct | ⚠️ | Structurally correct in source, but untested |
| Keyboard navigation | ❌ | Not tested |
| Screen reader announces correctly | ❌ | Not tested |
| Dark mode works | ❌ | Not tested |
| Mobile responsive | ❌ | Not tested |

### Documentation Requirements

| Criterion | Status | Notes |
|-----------|--------|-------|
| Pico CSS setup guide | ✅ | `docs/guide/styling.md` — comprehensive |
| Component docs with props/examples | ✅ | Covered in styling guide |
| Migration guide | ✅ | Included in styling guide |
| FAQ entry | ✅ | Added to `docs/guide/faq.md` |

### Test Coverage

| Criterion | Status | Notes |
|-----------|--------|-------|
| Unit tests for HTML structure | ❌ | None created |
| Unit tests for ARIA attributes | ❌ | None created |
| Unit tests for conditional rendering | ❌ | None created |
| Unit tests for edge cases | ❌ | None created |
| Integration test pages | ❌ | None created |

---

## Section 4: Additional Findings

### 4.1 ⚠️ Supplement CSS has overly broad selectors

```css
#toasts,
[data-position] {
    position: fixed;
    ...
}
```

The `[data-position]` selector matches **any** element with a `data-position` attribute anywhere in the document, not just toast containers. This could cause unintended styling on unrelated elements.

**Fix:** Scope to `#toasts[data-position]` or `aside[data-position]`.

### 4.2 ⚠️ Checkbox, CheckboxGroup, RadioGroup not updated

The design document (`DESIGN-prelude-pico-compatibility.md` §7, lines 854–855) lists:
- `checkbox.pars` — "Verify Pico checkbox pattern"
- `radio_group.pars` — "Verify Pico radio pattern"

These were not in the spec acceptance criteria, so omitting them is defensible. However, it creates an inconsistency: `TextField`, `TextareaField`, and `SelectField` now use `<small>` for hints/errors and have no wrapper divs, while `Checkbox`, `CheckboxGroup`, and `RadioGroup` still use `<p class="field-hint">`, `<p class="field-error">`, and wrapper divs with custom classes.

These components also have:
- Pre-existing swapped `for` parameter order (CheckboxGroup, RadioGroup)
- `.length` instead of `.length()` (all three)
- Custom classes that won't be styled by Pico

### 4.3 ⚠️ Spec marked complete despite missing tests

The spec status was changed to `complete` and the plan marks tasks 4.1 (integration tests) and 4.2 (example page) as "Deferred". However, the spec's acceptance criteria explicitly list testing requirements with checkboxes. The spec should either remain `draft`/`in-progress` until tests exist, or the acceptance criteria should be amended to note the deferral.

### 4.4 ℹ️ Documentation Parsley syntax in migration section

In `docs/guide/styling.md`, the migration section uses Parsley comments to describe HTML output:

```parsley
// Output is now:
// <label for="email">Email</label>
// <input type="email" id="email" name="email"/>
// <small id="email-hint">We'll never share this</small>
```

This could confuse users who try to run it as code. Consider presenting expected HTML output in a separate block or as prose.

---

## Section 5: Registrations

Component registrations in `pkg/parsley/evaluator/stdlib_html.go` are correct:

| Component | `htmlModuleMeta` | `componentFiles` | Status |
|-----------|-----------------|-------------------|--------|
| Details | ✅ | ✅ | Registered |
| Accordion | ✅ | ✅ | Registered |
| Dialog | ✅ | ✅ | Registered |
| Toast | ✅ | ✅ | Registered |
| Toasts | ✅ | ✅ | Registered |
| ErrorSummary | ✅ | ✅ | Registered |
| Pagination | ✅ | ✅ | Registered |

All 7 new components are properly registered in both metadata and file lists.

---

## Section 6: What's Good

Despite the bugs, the implementation gets many things right:

1. **Design decisions are sound** — Pico CSS, semantic HTML, `data-*` attributes, native `<details>` accordion
2. **ARIA attributes** — Correct roles, aria-labels, aria-current, aria-live, tabindex throughout
3. **Progressive enhancement** — Dialog uses native `<dialog>`, accordion uses native `name` attribute
4. **Form field updates** — TextField, TextareaField, SelectField correctly use `<small>` and `??` defaults
5. **Documentation** — The styling guide, FAQ entry, and migration guide are comprehensive and well-written
6. **Supplement CSS** — Well-structured, uses Pico CSS custom properties for theming
7. **ErrorSummary** — Good accessibility pattern with `role="alert"`, `tabindex="-1"`, and field links
8. **Pagination** — Good algorithm design with configurable window, labels, and URL templates

---

## Section 7: Recommended Actions

### Must Fix (Blocking)

1. **Replace `{...attrs}` with `...attrs`** in all 9 affected component files
2. **Fix Accordion `for` parameter order** — change `for (item, i in items)` to `for (i, item in items)`
3. **Fix Pagination range** — change `start..end + 1` to `start..end`
4. **Fix Breadcrumb `for` parameter order** — change `for (item, idx in items)` to `for (idx, item in items)` (pre-existing bug, now exposed)
5. **Add basic component tests** — at minimum verify each component renders without error

### Should Fix

6. **Fix pre-existing swapped `for` params** in `checkbox_group.pars`, `radio_group.pars`, `data_table.pars`
7. **Scope supplement CSS** `[data-position]` selector to `#toasts[data-position]`
8. **Update Checkbox/CheckboxGroup/RadioGroup** to match Pico pattern (use `<small>`, remove custom classes, fix `.length` → `.length()`)
9. **Update spec** to match actual implementations (prop names, function signatures, remove default params)
10. **Update design doc** to fix `{...attrs}` and default parameter syntax

### Should Consider

11. **Revert spec status** from `complete` to `in-progress` until bugs are fixed and tests exist
12. **Add CI test** that runs `pars -e` against component examples to catch syntax issues early

---

## Section 8: Root Cause Analysis

The bugs all share a common root cause: **the design document's Parsley examples were not verified against the language before being implemented.**

The `{...attrs}` syntax, default parameters in destructuring, and `for (item, i in ...)` ordering are all patterns from JavaScript/JSX that don't apply to Parsley. They were written into the design doc, copied into the spec, and then implemented faithfully — but never tested.

The absence of component tests meant these runtime errors were never caught. The existing test suite passes because:
- Server tests don't render individual components with test data
- The `go test ./...` suite verifies Go code compilation and component registration, not Parsley evaluation
- No integration tests exercise the component templates

**Prevention:** Before implementing Parsley code from spec/design documents, verify examples with `pars -e` or `pars -r -e`. The `.github/copilot-instructions.md` already mandates this ("Verify examples work — run `pars -e "code"` before committing doc changes") but it was not followed during implementation.