---
id: FEAT-143
title: "Prelude Component Styling Strategy"
status: complete
priority: high
created: 2026-06-15
updated: 2026-03-15
author: "@human"
---

# FEAT-143: Prelude Component Styling Strategy

## Summary

Adopt Pico CSS as the recommended styling framework for Prelude components, remove all embedded CSS from components, and provide a small supplement CSS file for components Pico doesn't cover (toasts, pagination, error summary, skip link). Components output semantic HTML with accessibility attributes that works unstyled and looks polished with Pico.

## Current Status

**✅ COMPLETE**

All components implemented with correct Parsley syntax. Post-implementation audit issues have been resolved.

| Area | Status |
|------|--------|
| Design direction | ✅ Complete |
| Component structure | ✅ Complete |
| Supplement CSS | ✅ Complete |
| Documentation | ✅ Complete |
| Parsley correctness | ✅ Complete |
| Verification script | ✅ Complete |

## User Story

As a developer building web applications with Basil, I want Prelude components that output semantic HTML without embedded styles, so that I can use any CSS framework (or none) while getting accessible, well-structured markup out of the box.

## Problem Statement

Current Prelude components have several styling issues:

1. **Embedded CSS** — `SkipLink` contains inline `<style>` tags, violating separation of concerns
2. **Custom classes** — Components use custom classes (`.field`, `.field-hint`, `.breadcrumb`) that require custom CSS
3. **Framework lock-in** — Users must either use our undocumented CSS or write their own
4. **Inconsistent patterns** — Different components use different patterns for similar concepts (errors, hints)
5. **No recommended path** — No clear guidance on how to style Prelude components

## Design Decisions

### DECIDED: All-In on Standard Pico CSS

**Decision:** Use unmodified Pico CSS from CDN or npm. Do not fork or modify Pico.

**Rationale:**
- Users can upgrade Pico independently
- No maintenance burden for Basil team
- Familiar to existing Pico users
- Pico already used in Basil devtools (`/__/css/pico.min.css`)
- Supplement file is minimal (~50 lines)

### DECIDED: Use `data-*` Attributes for Semantic Variants

**Decision:** Use `data-type="success"` instead of `.toast-success` for styling hooks.

**Rationale:**
- Keeps classes available for user customization
- Semantic meaning embedded in attribute
- Easy to style with attribute selectors
- Works in Pico's classless mode

### DECIDED: Native HTML5 Accordion

**Decision:** Use `<details name="group">` for exclusive accordions.

**Rationale:**
- No JavaScript required
- Browser handles open/close logic
- Widely supported (Chrome 120+, Firefox 130+, Safari 17.2+)
- Progressive enhancement — works without `name` in older browsers (just not exclusive)

### DECIDED: Supplement CSS Location

**Decision:** Ship `basil-supplement.css` in `examples/css/` with documentation.

**Rationale:**
- Keeps it optional and visible
- Users can copy into their projects or reference directly
- Not bundled into core — users choose their styling approach

### DECIDED: Target Pico CSS with Classes

**Decision:** Design components for Pico CSS with classes (the full-featured mode). Components may use classes where beneficial.

**Rationale:**
- Full Pico feature set available (button variants, grid classes, etc.)
- Classless Pico still works for basic styling — just won't pick up class-based enhancements
- Simpler implementation — no need to avoid classes or create `data-*` workarounds
- Users wanting minimal setup can use classless; users wanting full control use classes

---

## Acceptance Criteria

### Phase 1: Foundation (No Breaking Changes)

- [x] `examples/css/basil-supplement.css` exists with styles for skip-link, toasts, pagination, error-summary
- [x] `examples/css/README.md` documents how to use Pico + supplement
- [x] New `Dialog` component created using `<dialog><article>` pattern
- [x] New `Details` component created using native `<details><summary>` pattern
- [x] New `Accordion` component created using `<details name="...">` pattern
- [x] Documentation explains Pico CSS setup for user projects

### Phase 2: Migrate Existing Components

- [x] `TextField` uses `<small>` for hints/errors instead of `<p>` with custom classes
- [x] `TextareaField` updated to match `TextField` pattern
- [x] `SelectField` updated to match `TextField` pattern
- [x] `Breadcrumb` uses `aria-label="breadcrumb"` (Pico convention) and removes custom classes
- [x] `SkipLink` removes inline `<style>`, uses `class="skip-link"` for supplement CSS
- [x] `Page` changes `id="main"` default from `<body>` to `null` (users put `id` on `<main>`)
- [x] `Form` removes `.form` class (optional, Pico styles `<form>` directly)

### Phase 3: New Components

- [x] `Toast` component outputs `<article role="status|alert" data-type="...">`
- [x] `Toasts` container component outputs `<aside aria-live="polite">`
- [x] `Pagination` component outputs semantic nav with `aria-label="Pagination"`
- [x] `ErrorSummary` component outputs `<aside role="alert">` with field links

### Phase 4: Parsley Correctness (NEW — BLOCKING)

- [ ] Fix invalid `{...attrs}` spread syntax → `...attrs` in 9 component files
- [ ] Fix reversed `for` loop variable ordering in 6 component files
- [ ] Fix pagination range precedence bug (`start..end + 1` → `start..end`)
- [ ] All components pass `pars --check` syntax validation
- [ ] All components render correctly when evaluated with sample props

### Phase 5: Testing (NEW — REQUIRED)

- [ ] Integration tests added to `pkg/parsley/tests/` for new components
- [ ] Verification script created and passes
- [ ] Edge cases tested (empty arrays, null values, boundary conditions)

### Testing Requirements

For each component, verify:

- [ ] Works unstyled (readable, functional)
- [ ] Works with Pico classless mode
- [ ] Works with Pico class mode
- [ ] Works with Pico + supplement
- [ ] ARIA attributes are correct
- [ ] Keyboard navigation works
- [ ] Screen reader announces correctly
- [ ] Dark mode works (via `color-scheme` or Pico's theme)
- [ ] Mobile responsive

### Documentation Requirements

- [x] Pico CSS setup guide added to docs
- [x] Each new component documented with props, examples, and accessibility notes
- [x] Migration guide for existing users
- [x] FAQ entry for "How do I style Prelude components?"
- [ ] Spec examples corrected to use valid Parsley syntax

---

## Appendix: Parsley Correctness Fixes

A post-implementation audit identified blocking Parsley syntax errors. These MUST be fixed before the feature is complete.

### Issue 1: Invalid Tag Spread Syntax

**Problem:** Used `{...attrs}` (JSX syntax) instead of `...attrs` (Parsley syntax).

**Affected files (9):**
- `accordion.pars`, `breadcrumb.pars`, `details.pars`, `dialog.pars`
- `error_summary.pars`, `pagination.pars`, `skip_link.pars`, `toast.pars`, `toasts.pars`

**Fix:** Replace `{...attrs}` with `...attrs`

**Verification:**
```bash
# Should error:
pars -e 'let a = {x: 1}; <div {...a}>"test"</div>'

# Should work:
pars -e 'let a = {x: 1}; <div ...a>"test"</div>'
```

### Issue 2: Reversed `for` Loop Variable Ordering

**Problem:** Used `for (item, idx in items)` but Parsley uses `for (idx, item in items)`.

**Affected files (6):**
- `accordion.pars`: `for (item, i in items)` → `for (i, item in items)`
- `breadcrumb.pars`: `for (item, idx in items)` → `for (idx, item in items)`
- `checkbox_group.pars`: `for (opt, idx in options)` → `for (idx, opt in options)`
- `radio_group.pars`: `for (opt, idx in options)` → `for (idx, opt in options)`
- `data_table.pars`: Two loops need fixing

**Verification:**
```bash
pars -e 'for (i, item in ["a", "b", "c"]) { i + ":" + item }'
# Correct output: ["0:a", "1:b", "2:c"]
```

### Issue 3: Pagination Range Precedence Bug

**Problem:** `start..end + 1` parses as `(start..end) + 1` causing type error.

**Location:** `pagination.pars` line 45

**Fix:** Change to `start..end` (range is inclusive)

**Verification:**
```bash
pars -e '1..5 + 1'  # Error: Type mismatch
pars -e '1..5'      # [1, 2, 3, 4, 5] - inclusive
```

### Correct Parsley Patterns

For reference, these are the verified correct patterns:

| Pattern | ❌ Wrong | ✅ Correct |
|---------|----------|-----------|
| Tag spread | `<div {...attrs}>` | `<div ...attrs>` |
| For loop (array) | `for (item, idx in arr)` | `for (idx, item in arr)` |
| Conditional attr | `attr={cond && "value"}` | `attr={if (cond) "value" else null}` |
| Default value | `fn({x = 5})` | `fn({x}) { let val = x ?? 5 }` |

### Verification Checklist

After fixes are applied:

```bash
# 1. Syntax check all files
for f in server/prelude/components/*.pars; do
    pars --check "$f" || echo "FAIL: $f"
done

# 2. Test accordion renders correctly
pars -r -e '{Accordion} = import @basil/html; <Accordion name="test" items={[{title: "Q1", content: "A1"}]}/>'

# 3. Test breadcrumb positions are 1-based
pars -r -e '{Breadcrumb} = import @basil/html; <Breadcrumb items={[{label: "Home", href: "/"}, {label: "About"}]}/>' | grep 'content="1"'

# 4. Test pagination doesn't error
pars -r -e '{Pagination} = import @basil/html; <Pagination current={3} total={100} perPage={10} href="/p?page={page}"/>'
```

---

## Technical Context

### Design Document

See `work/design/DESIGN-prelude-pico-compatibility.md` for full component mappings, HTML patterns, and Parsley implementations.

### Related Files

- Supplement CSS: `examples/css/basil-supplement.css`
- Supplement README: `examples/css/README.md`
- Pico in devtools: `server/prelude/devtools/components/page.pars`
- **Review document:** `work/reports/STANDARD-PRELUDE-REVIEW.md` (Appendix D has full fix details)

### Dependencies

- Depends on: None
- Blocks: None
- Related: FEAT-051 (Standard Prelude), FEAT-142 (Meta Component)

---

## Component Specifications

> **Note:** The examples below have been corrected to use valid Parsley syntax.

### New Components

#### Dialog

**File:** `server/prelude/components/dialog.pars`

**Props:**
| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `id` | string | required | Dialog ID for targeting |
| `title` | string | — | Optional header title |
| `footer` | any | — | Footer content (buttons) |
| `contents` | any | — | Dialog body content |

**Parsley Implementation:**
```parsley
export Dialog = fn({id, title, contents, footer, ...attrs}) {
    <dialog id={id} ...attrs>
        <article>
            if (title) {
                <header>
                    <button 
                        aria-label="Close" 
                        rel="prev"
                        onclick="this.closest('dialog').close()"
                    />
                    <h2>title</h2>
                </header>
            }
            contents
            if (footer) {
                <footer>footer</footer>
            }
        </article>
    </dialog>
}
```

---

#### Details

**File:** `server/prelude/components/details.pars`

**Props:**
| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `title` | string | required | Summary text |
| `open` | boolean | `false` | Initially expanded |
| `name` | string | — | Group name for exclusive accordion |
| `contents` | any | — | Expandable content |

**Parsley Implementation:**
```parsley
export Details = fn({title, open, name, contents, ...attrs}) {
    <details open={open} name={name} ...attrs>
        <summary>title</summary>
        contents
    </details>
}
```

---

#### Accordion

**File:** `server/prelude/components/accordion.pars`

**Props:**
| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `name` | string | required | Group name (enables exclusive behavior) |
| `items` | array | required | Array of `{title, content, open?}` |

**Parsley Implementation:**
```parsley
export Accordion = fn({name, items, ...attrs}) {
    if (items == null || items.length() == 0) {
        null
    } else {
        for (i, item in items) {
            <details name={name} open={item.open ?? (i == 0)} ...attrs>
                <summary>item.title</summary>
                item.content
            </details>
        }
    }
}
```

---

#### Toast

**File:** `server/prelude/components/toast.pars`

**Props:**
| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `message` | string | required | Toast message |
| `type` | string | `"info"` | One of: `info`, `success`, `warning`, `error` |
| `dismissible` | boolean | `true` | Show dismiss button |

**Parsley Implementation:**
```parsley
export Toast = fn({message, type, dismissible, ...attrs}) {
    let toastType = type ?? "info"
    let canDismiss = dismissible ?? true
    let role = if (toastType == "error") "alert" else "status"
    
    <article role={role} data-type={toastType} ...attrs>
        <p>message</p>
        if (canDismiss) {
            <button aria-label="Dismiss" onclick="this.parentElement.remove()"/>
        }
    </article>
}
```

---

#### Toasts (Container)

**File:** `server/prelude/components/toasts.pars`

**Props:**
| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `position` | string | `"top-right"` | Position: `top-right`, `top-left`, `top-center`, `bottom-right`, `bottom-left`, `bottom-center` |
| `contents` | any | — | Toast children |

**Parsley Implementation:**
```parsley
export Toasts = fn({position, contents, ...attrs}) {
    <aside 
        id="toasts" 
        aria-live="polite" 
        aria-label="Notifications"
        data-position={position ?? "top-right"}
        ...attrs
    >
        contents
    </aside>
}
```

---

#### Pagination

**File:** `server/prelude/components/pagination.pars`

**Props:**
| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `current` | integer | required | Current page number |
| `total` | integer | required | Total item count |
| `perPage` | integer | `20` | Items per page |
| `href` | string | required | URL template with `{page}` placeholder |
| `window` | integer | `2` | Pages to show around current |
| `showFirst` | boolean | `true` | Show first/last buttons |
| `showPrev` | boolean | `true` | Show prev/next buttons |
| `labels` | object | see below | Button labels |

**Parsley Implementation:**
```parsley
let {max, min} = import @std/math

export Pagination = fn({current, total, perPage, href, window, showFirst, showPrev, labels, ...attrs}) {
    let itemsPerPage = perPage ?? 20
    let totalPages = ((total - 1) / itemsPerPage).floor() + 1

    if (totalPages <= 1) {
        null
    } else {
        let pageWindow = window ?? 2
        let showFirstLast = showFirst ?? true
        let showPrevNext = showPrev ?? true
        let navLabels = labels ?? {first: "«", prev: "‹", next: "›", last: "»"}

        let pageUrl = fn(n) { href.replace("{page}", n + "") }

        let start = max(1, current - pageWindow)
        let end = min(totalPages, current + pageWindow)

        <nav aria-label="Pagination" ...attrs>
            <ul>
                // First/prev buttons, page numbers, next/last buttons
                // ... (see full implementation in component file)
            </ul>
        </nav>
    }
}
```

---

#### ErrorSummary

**File:** `server/prelude/components/error_summary.pars`

**Props:**
| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `title` | string | `"There is a problem"` | Summary heading |
| `errors` | array | required | Array of `{field, message}` |
| `id` | string | `"error-summary"` | Element ID |

**Parsley Implementation:**
```parsley
export ErrorSummary = fn({title, errors, id, ...attrs}) {
    if (errors == null || errors.length() == 0) {
        null
    } else {
        let summaryId = id ?? "error-summary"
        let summaryTitle = title ?? "There is a problem"

        <aside
            role="alert"
            aria-labelledby={summaryId + "-title"}
            tabindex="-1"
            id={summaryId}
            ...attrs
        >
            <header>
                <h2 id={summaryId + "-title"}>summaryTitle</h2>
            </header>
            <ul>
                for (err in errors) {
                    <li>
                        <a href={"#" + err.field}>err.message</a>
                    </li>
                }
            </ul>
        </aside>
    }
}
```

---

## Related

- **Design:** `work/design/DESIGN-prelude-pico-compatibility.md`
- **Review:** `work/reports/STANDARD-PRELUDE-REVIEW.md`
- **Plan:** `work/plans/FEAT-143-plan.md`
- **Prior art:** FEAT-051 (Standard Prelude), FEAT-142 (Meta Component)
