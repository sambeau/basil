---
id: FEAT-143
title: "Prelude Component Styling Strategy"
status: complete
priority: high
created: 2026-06-15
completed: 2026-06-15
author: "@human"
---

# FEAT-143: Prelude Component Styling Strategy

## Summary

Adopt Pico CSS as the recommended styling framework for Prelude components, remove all embedded CSS from components, and provide a small supplement CSS file for components Pico doesn't cover (toasts, pagination, error summary, skip link). Components output semantic HTML with accessibility attributes that works unstyled and looks polished with Pico.

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

- [ ] `examples/css/basil-supplement.css` exists with styles for skip-link, toasts, pagination, error-summary
- [ ] `examples/css/README.md` documents how to use Pico + supplement
- [ ] New `Dialog` component created using `<dialog><article>` pattern
- [ ] New `Details` component created using native `<details><summary>` pattern
- [ ] New `Accordion` component created using `<details name="...">` pattern
- [ ] Documentation explains Pico CSS setup for user projects

### Phase 2: Migrate Existing Components

- [ ] `TextField` uses `<small>` for hints/errors instead of `<p>` with custom classes
- [ ] `TextareaField` updated to match `TextField` pattern
- [ ] `SelectField` updated to match `TextField` pattern
- [ ] `Breadcrumb` uses `aria-label="Breadcrumb"` (Pico convention) and removes custom classes
- [ ] `SkipLink` removes inline `<style>`, uses `class="skip-link"` for supplement CSS
- [ ] `Page` changes `id="main"` default from `<body>` to `null` (users put `id` on `<main>`)
- [ ] `Form` removes `.form` class (optional, Pico styles `<form>` directly)

### Phase 3: New Components

- [ ] `Toast` component outputs `<article role="status|alert" data-type="...">`
- [ ] `Toasts` container component outputs `<aside aria-live="polite">`
- [ ] `Pagination` component outputs semantic nav with `aria-label="Pagination"`
- [ ] `ErrorSummary` component outputs `<aside role="alert">` with field links

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

- [ ] Pico CSS setup guide added to docs
- [ ] Each new component documented with props, examples, and accessibility notes
- [ ] Migration guide for existing users
- [ ] FAQ entry for "How do I style Prelude components?"

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Design Document

See `work/design/DESIGN-prelude-pico-compatibility.md` for full component mappings, HTML patterns, and Parsley implementations.

### Related Files

- Supplement CSS: `examples/css/basil-supplement.css` (already created)
- Supplement README: `examples/css/README.md` (already created)
- Pico in devtools: `server/prelude/devtools/components/page.pars`

### Dependencies

- Depends on: None
- Blocks: None
- Related: FEAT-051 (Standard Prelude), FEAT-142 (Meta Component)

---

## Component Specifications

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

**HTML Output:**
```html
<dialog id="confirm">
    <article>
        <header>
            <button aria-label="Close" rel="prev" onclick="this.closest('dialog').close()"></button>
            <h2>Title</h2>
        </header>
        <!-- contents -->
        <footer><!-- footer --></footer>
    </article>
</dialog>
```

**Parsley Implementation:**
```parsley
export Dialog = fn({id, title, contents, footer, ...attrs}) {
    <dialog id={id} {...attrs}>
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
export Details = fn({title, open = false, contents, ...attrs}) {
    <details open={open} {...attrs}>
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
    for (item, i in items) {
        <details name={name} open={item.open ?? (i == 0)} {...attrs}>
            <summary>item.title</summary>
            item.content
        </details>
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
export Toast = fn({message, type = "info", dismissible = true, ...attrs}) {
    let role = if (type == "error") { "alert" } else { "status" }
    
    <article role={role} data-type={type} {...attrs}>
        <p>message</p>
        if (dismissible) {
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
export Toasts = fn({position = "top-right", contents, ...attrs}) {
    <aside 
        id="toasts" 
        aria-live="polite" 
        aria-label="Notifications"
        data-position={position}
        {...attrs}
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
| `labels` | object | `{first: "«", prev: "‹", next: "›", last: "»"}` | Button labels |

**Parsley Implementation:**
```parsley
let {max, min} = import @std/math

export Pagination = fn({
    current,
    total,
    perPage = 20,
    href,
    window = 2,
    showFirst = true,
    showPrev = true,
    labels = {first: "«", prev: "‹", next: "›", last: "»"},
    ...attrs
}) {
    let totalPages = ((total - 1) / perPage).floor() + 1
    if (totalPages <= 1) { null }
    else {
        let pageUrl = fn(n) { href.replace("{page}", n ++ "") }
        
        <nav aria-label="Pagination" {...attrs}>
            <ul>
                if (showFirst && current > 1) {
                    <li><a href={pageUrl(1)} aria-label="First page">labels.first</a></li>
                }
                if (showPrev && current > 1) {
                    <li><a href={pageUrl(current - 1)} aria-label="Previous page">labels.prev</a></li>
                }
                
                let start = max(1, current - window)
                let end = min(totalPages, current + window)
                
                if (start > 1) {
                    <li><a href={pageUrl(1)}>"1"</a></li>
                    if (start > 2) {
                        <li><span aria-hidden="true">"…"</span></li>
                    }
                }
                
                for (n in start..end) {
                    <li>
                        if (n == current) {
                            <a href={pageUrl(n)} aria-current="page">n</a>
                        } else {
                            <a href={pageUrl(n)}>n</a>
                        }
                    </li>
                }
                
                if (end < totalPages) {
                    if (end < totalPages - 1) {
                        <li><span aria-hidden="true">"…"</span></li>
                    }
                    <li><a href={pageUrl(totalPages)}>totalPages</a></li>
                }
                
                if (showPrev && current < totalPages) {
                    <li><a href={pageUrl(current + 1)} aria-label="Next page">labels.next</a></li>
                }
                if (showFirst && current < totalPages) {
                    <li><a href={pageUrl(totalPages)} aria-label="Last page">labels.last</a></li>
                }
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
| `errors` | array | required | Array of `{field, message}` objects |
| `id` | string | `"error-summary"` | Element ID for focus management |

**Parsley Implementation:**
```parsley
export ErrorSummary = fn({
    title = "There is a problem",
    errors,
    id = "error-summary",
    ...attrs
}) {
    if (errors == null || errors.length() == 0) { null }
    else {
        <aside 
            role="alert" 
            aria-labelledby={id ++ "-title"} 
            tabindex="-1" 
            id={id}
            {...attrs}
        >
            <header>
                <h2 id={id ++ "-title"}>title</h2>
            </header>
            <ul>
                for (err in errors) {
                    <li>
                        <a href={"#" ++ err.field}>err.message</a>
                    </li>
                }
            </ul>
        </aside>
    }
}
```

---

### Existing Component Changes

#### TextField (Update)

**Changes:**
- Use `<small>` instead of `<p>` for hint/error text
- Remove custom classes (`.field`, `.field-hint`, `.field-error`)
- Keep all ARIA attributes

**Updated Implementation:**
```parsley
export TextField = fn({
    name,
    label,
    type = "text",
    value,
    error,
    hint,
    required = false,
    ...attrs
}) {
    let inputId = attrs.id ?? name
    let hasError = error != null && error != ""
    
    let describedBy = if (hint && hasError) {
        inputId ++ "-hint " ++ inputId ++ "-error"
    } else if (hint) {
        inputId ++ "-hint"
    } else if (hasError) {
        inputId ++ "-error"
    } else {
        null
    }
    
    <label for={inputId}>label</label>
    <input
        type={type}
        id={inputId}
        name={name}
        value={value}
        required={required}
        aria-invalid={if (hasError) { "true" } else { null }}
        aria-describedby={describedBy}
        {...attrs}
    />
    if (hint && !hasError) {
        <small id={inputId ++ "-hint"}>hint</small>
    }
    if (hasError) {
        <small id={inputId ++ "-error"}>error</small>
    }
}
```

---

#### SkipLink (Update)

**Changes:**
- Remove inline `<style>` tag
- Add `class="skip-link"` for supplement CSS targeting

**Updated Implementation:**
```parsley
export SkipLink = fn({href = "#main", text = "Skip to main content", ...attrs}) {
    <a href={href} class="skip-link" {...attrs}>text</a>
}
```

---

### Supplement CSS

The supplement CSS (`examples/css/basil-supplement.css`) covers:

1. **Skip Link** — Visually hidden until focused
2. **Toast Container** — Fixed positioning with `data-position` variants
3. **Toast Types** — Border colors for `data-type` variants
4. **Pagination** — Flexbox layout for nav list
5. **Error Summary** — Alert border and focus styles
6. **Screen Reader Only** — `.sr-only` utility class

---

## Test Strategy

### Unit Tests

For each component, create tests in `pkg/parsley/tests/` that verify:

1. **HTML Structure** — Output matches expected semantic structure
2. **ARIA Attributes** — Required accessibility attributes are present
3. **Conditional Rendering** — Optional elements render correctly based on props
4. **Edge Cases** — Empty arrays, null values, missing optional props

### Integration Tests

Create example pages that demonstrate:

1. **Unstyled rendering** — Components are usable without CSS
2. **Pico classless** — Components look correct with Pico classless
3. **Pico + supplement** — Full styling with all features

### Accessibility Tests

Manual testing checklist:
- [ ] Keyboard navigation (Tab, Enter, Escape)
- [ ] Screen reader announcements (VoiceOver, NVDA)
- [ ] Focus management (dialogs, error summaries)
- [ ] Color contrast in light/dark modes

---

## Migration Guide

### For Existing Users

**SkipLink:**
```parsley
// Before: Worked but had inline CSS
<SkipLink/>

// After: Add supplement CSS to your page
// In <head>: <link rel="stylesheet" href="/css/basil-supplement.css"/>
<SkipLink/>
```

**TextField:**
```parsley
// Before (still works, just outputs cleaner HTML)
<TextField name="email" label="Email" error="Invalid email"/>

// No code changes needed — output changes from <p> to <small>
```

**Page body id:**
```parsley
// Before: Page set id="main" on <body> by default

// After: Add id="main" to your <main> element explicitly
<Page title="My Page">
    <main id="main">
        "Content"
    </main>
</Page>
```

---

## Implementation Priority

| Phase | Components | Effort | Breaking Changes |
|-------|------------|--------|------------------|
| 1 | Dialog, Details, Accordion, Docs | 4 hours | None |
| 2 | TextField, SkipLink, Breadcrumb, Page | 3 hours | Minor (HTML output) |
| 3 | Toast, Toasts, Pagination, ErrorSummary | 4 hours | None |

**Total estimated effort:** ~11 hours

---

## Related

- Design doc: `work/design/DESIGN-prelude-pico-compatibility.md`
- Supplement CSS: `examples/css/basil-supplement.css`
- Parent feature: FEAT-051 (Standard Prelude)
- Related: FEAT-142 (Meta Component and Page Restructure)
- Pico CSS: https://picocss.com/docs