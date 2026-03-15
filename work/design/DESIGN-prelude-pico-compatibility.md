# Design: Prelude Pico CSS Compatibility

**Status:** Draft  
**Created:** 2025-01-14  
**Related:** DESIGN-prelude-1.0-components.md, STANDARD-PRELUDE-REVIEW.md

## 1. Executive Summary

This document defines how Prelude components should output HTML that is **fully compatible with Pico CSS** while remaining **framework-agnostic**. The approach is:

1. **All-in on standard Pico CSS** — no fork, no modifications
2. **Tiny supplement file** (~30 lines) for gaps Pico doesn't cover
3. **Components output semantic HTML** that works unstyled and looks great with Pico
4. **Zero CSS embedded in components** — all styling is external

### Why Pico CSS?

| Criterion | Pico CSS | Notes |
|-----------|----------|-------|
| Size | 3.5KB gzipped | Smallest of viable options |
| Philosophy | Semantic HTML | Matches our design principles |
| Classless version | Yes | Works without any classes |
| Maintenance | Active, MIT licensed | Low risk |
| Already in use | Yes | Basil devtools use it at `/__/css/pico.min.css` |
| Browser support | Modern browsers | Same as Parsley target |

---

## 2. Current State Analysis

### Pico CSS Already in Use

The Basil devtools already use Pico CSS, served at:
- `/__/css/pico.min.css`
- `/__/css/pico.colors.min.css`

See `server/prelude/devtools/components/page.pars`:

```parsley
<link rel="stylesheet" href="/__/css/pico.colors.min.css"/>
<link rel="stylesheet" href="/__/css/pico.min.css"/>
```

### Current Component Patterns

**TextField** (`server/prelude/components/text_field.pars`):
- Uses custom classes: `.field`, `.field-hint`, `.field-error`, `.field-required`
- Good: Already uses `aria-invalid`, `aria-describedby`, `aria-required`
- Change needed: Use `<small>` instead of `<p>` for hints/errors (Pico patterns)

**Breadcrumb** (`server/prelude/components/breadcrumb.pars`):
- Uses custom classes: `.breadcrumb`, `.breadcrumb-list`, `.breadcrumb-item`
- Good: Uses `aria-label="Breadcrumb"`, `aria-current="page"`
- Change needed: Pico expects `<nav aria-label="breadcrumb">` (lowercase)

**SkipLink** (`server/prelude/components/skip_link.pars`):
- ⚠️ Contains inline `<style>` tag — violates CSS-free principle
- Change needed: Move to supplement CSS

**Form** (`server/prelude/components/form.pars`):
- Uses custom class: `.form`
- Mostly fine — Pico styles `<form>` directly

**Page** (`server/prelude/components/page.pars`):
- Sets `id="main"` on body by default (should be on `<main>`)
- Includes `<SkipLink/>` — good, but needs CSS externalized

---

## 3. Pico CSS Coverage Analysis

### What Pico Covers (No Custom CSS Needed)

| Component | Pico Support | HTML Pattern |
|-----------|--------------|--------------|
| **Dialog/Modal** | ✅ Full | `<dialog><article>` |
| **Accordion** | ✅ Full | `<details><summary>` |
| **Forms** | ✅ Full | Semantic form elements |
| **Form validation** | ✅ Full | `aria-invalid` attribute |
| **Helper text** | ✅ Full | `<small>` below input |
| **Buttons** | ✅ Full | `<button>`, variants via class |
| **Navigation** | ✅ Full | `<nav><ul><li>` |
| **Breadcrumbs** | ✅ Full | `<nav aria-label="breadcrumb">` |
| **Tables** | ✅ Full | Semantic table elements |
| **Progress** | ✅ Full | `<progress>` |
| **Loading states** | ✅ Full | `aria-busy="true"` |

### What Pico Doesn't Cover (Supplement Required)

| Component | Gap | Supplement Needed |
|-----------|-----|-------------------|
| **Toast/Notification** | No component | Position + basic styles |
| **Pagination** | No component | Flexbox layout |
| **Error Summary** | No component | Border + focus styles |
| **Skip Link** | No `.sr-only` | Screen reader styles |

---

## 4. Component HTML Mappings

### 4.1 Dialog

**Current Design:**

```html
<dialog id="confirm" class="dialog" data-close-on-backdrop="true">
    <header class="dialog-header">
        <h2 class="dialog-title">Title</h2>
        <button class="dialog-close">×</button>
    </header>
    <div class="dialog-content">...</div>
</dialog>
```

**Pico-Compatible:**

```html
<dialog id="confirm">
    <article>
        <header>
            <button aria-label="Close" rel="prev"></button>
            <h2>Title</h2>
        </header>
        <p>Content goes here</p>
        <footer>
            <button class="secondary">Cancel</button>
            <button>Confirm</button>
        </footer>
    </article>
</dialog>
```

**Key differences:**
- `<article>` wrapper instead of custom classes
- `rel="prev"` on close button (Pico convention for positioning)
- `<footer>` for actions (Pico right-aligns content)
- No custom classes needed

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

**JavaScript (optional enhancement):**

```javascript
// From Pico examples - handles animation and backdrop click
// See: https://github.com/picocss/examples/blob/master/v2-html/js/modal.js

const isOpenClass = "modal-is-open";
const openingClass = "modal-is-opening";
const closingClass = "modal-is-closing";
const animationDuration = 400;

const openModal = (modal) => {
    document.documentElement.classList.add(isOpenClass, openingClass);
    setTimeout(() => {
        document.documentElement.classList.remove(openingClass);
    }, animationDuration);
    modal.showModal();
};

const closeModal = (modal) => {
    document.documentElement.classList.add(closingClass);
    setTimeout(() => {
        document.documentElement.classList.remove(closingClass, isOpenClass);
        modal.close();
    }, animationDuration);
};

// Backdrop click to close
document.addEventListener("click", (event) => {
    const dialog = event.target.closest("dialog[open]");
    if (dialog && event.target === dialog) {
        closeModal(dialog);
    }
});
```

---

### 4.2 Accordion

**Current Design:**

```html
<div class="accordion">
    <details class="details">
        <summary class="details-summary">Title</summary>
        <div class="details-content">Content</div>
    </details>
</div>
```

**Pico-Compatible (using native HTML5 `name` attribute):**

```html
<details name="faq">
    <summary>Question 1</summary>
    <p>Answer 1</p>
</details>
<details name="faq">
    <summary>Question 2</summary>
    <p>Answer 2</p>
</details>
```

**Key differences:**
- Native `name` attribute creates exclusive accordion (no JS needed!)
- No wrapper div required
- No custom classes needed
- Browser handles open/close logic

**Parsley Implementation:**

```parsley
// Single expandable section
export Details = fn({title, open = false, contents, ...attrs}) {
    <details open={open} {...attrs}>
        <summary>title</summary>
        contents
    </details>
}

// Exclusive accordion group (only one open at a time)
export Accordion = fn({name, items, ...attrs}) {
    for (item, i in items) {
        <details name={name} open={item.open ?? (i == 0)} {...attrs}>
            <summary>item.title</summary>
            item.content
        </details>
    }
}
```

**Button-style summary (optional):**

```html
<details>
    <summary role="button">Click me</summary>
    <p>Revealed content</p>
</details>

<!-- With variants -->
<details>
    <summary role="button" class="secondary">Secondary</summary>
    <p>Content</p>
</details>
```

---

### 4.3 Form Fields

**Current Implementation** (`server/prelude/components/text_field.pars`):

```html
<div class="field" id="field-email">
    <label for="field-email-input">
        Email
        <span class="field-required" aria-hidden="true"> *</span>
    </label>
    <input type="email" id="field-email-input" name="email" 
           aria-required="true" aria-describedby="field-email-error" 
           aria-invalid="true"/>
    <p id="field-email-hint" class="field-hint">We'll never share this</p>
    <p id="field-email-error" class="field-error" role="alert">Invalid email</p>
</div>
```


**Current Design (various patterns):**

```html
<div class="form-field">
    <label for="email">Email</label>
    <input type="email" id="email" name="email">
    <span class="error">Invalid email</span>
</div>
```

**Pico-Compatible:**

```html
<!-- Option A: Label wrapping input -->
<label>
    Email
    <input type="email" name="email" placeholder="Email" aria-invalid="true">
</label>
<small>Please enter a valid email address</small>

<!-- Option B: Label with for attribute -->
<label for="email">Email</label>
<input 
    type="email" 
    id="email" 
    name="email" 
    aria-invalid="true"
    aria-describedby="email-error"
>
<small id="email-error">Please enter a valid email address</small>
```

**Key differences:**
- `aria-invalid="true|false"` for validation states (Pico styles this)
- `<small>` for helper/error text (Pico inherits validation color)
- `aria-describedby` links input to error message
- No custom classes for basic styling

**Parsley Implementation:**

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
    
    // Build aria-describedby from hint and error IDs
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

### 4.4 Pagination (Supplement Required)

Pico doesn't have pagination, so we need minimal CSS.

**HTML Structure:**

```html
<nav aria-label="Pagination">
    <ul>
        <li><a href="?page=1" aria-label="First page">«</a></li>
        <li><a href="?page=2" aria-label="Previous page">‹</a></li>
        <li><a href="?page=1">1</a></li>
        <li><a href="?page=2">2</a></li>
        <li><a href="?page=3" aria-current="page">3</a></li>
        <li><a href="?page=4">4</a></li>
        <li><span aria-hidden="true">…</span></li>
        <li><a href="?page=10">10</a></li>
        <li><a href="?page=4" aria-label="Next page">›</a></li>
        <li><a href="?page=10" aria-label="Last page">»</a></li>
    </ul>
</nav>
```

**Key patterns:**
- Uses `<nav>` for semantics
- `aria-label="Pagination"` for screen readers
- `aria-current="page"` marks current page (no class needed)
- `aria-hidden="true"` on ellipsis
- `aria-label` on prev/next/first/last buttons

**Parsley Implementation:**

```parsley
let {max, min} = import @std/math

export Pagination = fn({
    current,
    total,
    perPage = 20,
    href,           // URL template: "/items?page={page}"
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
                // First button
                if (showFirst && current > 1) {
                    <li><a href={pageUrl(1)} aria-label="First page">labels.first</a></li>
                }
                
                // Previous button
                if (showPrev && current > 1) {
                    <li><a href={pageUrl(current - 1)} aria-label="Previous page">labels.prev</a></li>
                }
                
                // Page numbers with window
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
                
                // Next button
                if (showPrev && current < totalPages) {
                    <li><a href={pageUrl(current + 1)} aria-label="Next page">labels.next</a></li>
                }
                
                // Last button
                if (showFirst && current < totalPages) {
                    <li><a href={pageUrl(totalPages)} aria-label="Last page">labels.last</a></li>
                }
            </ul>
        </nav>
    }
}
```

---

### 4.5 Toast / Notification (Supplement Required)

Pico has no toast component. We'll use `<aside>` with ARIA roles.

**HTML Structure:**

```html
<!-- Container for multiple toasts -->
<aside id="toasts" aria-live="polite" aria-label="Notifications">
    <!-- Individual toast -->
    <article role="status" data-type="success">
        <p>Your changes have been saved.</p>
        <button aria-label="Dismiss" onclick="this.parentElement.remove()"></button>
    </article>
    
    <article role="alert" data-type="error">
        <p>Failed to save changes.</p>
        <button aria-label="Dismiss" onclick="this.parentElement.remove()"></button>
    </article>
</aside>
```

**Key patterns:**
- `<aside>` container with `aria-live="polite"` for non-urgent notifications
- `role="status"` for informational toasts
- `role="alert"` for errors (more urgent, interrupts screen readers)
- `data-type` attribute for styling hooks (not classes)
- `<article>` gets Pico's card styling automatically

**Parsley Implementation:**

```parsley
// Container component (typically placed once in Page)
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

// Individual toast
export Toast = fn({
    message,
    type = "info",    // info, success, warning, error
    dismissible = true,
    ...attrs
}) {
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

### 4.6 Error Summary (Supplement Required)

Similar to GDS pattern but simplified.

**HTML Structure:**

```html
<aside role="alert" aria-labelledby="error-summary-title" tabindex="-1" id="error-summary">
    <header>
        <h2 id="error-summary-title">There is a problem</h2>
    </header>
    <ul>
        <li><a href="#email">Enter a valid email address</a></li>
        <li><a href="#password">Password must be at least 8 characters</a></li>
    </ul>
</aside>
```

**Key patterns:**
- `role="alert"` for immediate announcement
- `tabindex="-1"` allows programmatic focus
- Links to individual field IDs
- Uses `<aside>` which Pico styles as a card

**Parsley Implementation:**

```parsley
export ErrorSummary = fn({
    title = "There is a problem",
    errors,    // Array of {field: string, message: string}
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

### 4.7 Skip Link (Supplement Required)

**Current Implementation** (`server/prelude/components/skip_link.pars`) contains inline CSS:

```parsley
<style>
    #skip { position:absolute; left:-10000px; ... }
    #skip:focus { position:static; width:auto; ... }
</style>
<a href={href} id="skip">linkText</a>
```

**Issue:** Inline `<style>` violates CSS-free component principle.


**HTML Structure:**

```html
<a href="#main" class="skip-link">Skip to main content</a>
```

**Parsley Implementation:**

```parsley
export SkipLink = fn({href = "#main", text = "Skip to main content", ...attrs}) {
    <a href={href} class="skip-link" {...attrs}>text</a>
}
```

---

## 5. Supplement CSS

This is the **only CSS we provide**. It's ~30 lines covering gaps in Pico.

```css
/* basil-supplement.css - Extends Pico CSS for Prelude components */
/* Version: 1.0.0 */

/* ===== Skip Link (Screen reader accessible) ===== */
.skip-link {
    position: absolute;
    left: -9999px;
    top: auto;
    width: 1px;
    height: 1px;
    overflow: hidden;
}
.skip-link:focus {
    position: fixed;
    top: 0;
    left: 0;
    width: auto;
    height: auto;
    padding: 1rem;
    background: var(--pico-background-color);
    z-index: 9999;
}

/* ===== Toast Container ===== */
#toasts, [data-position] {
    position: fixed;
    z-index: 1000;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    max-width: 24rem;
    pointer-events: none;
}
#toasts > *, [data-position] > * {
    pointer-events: auto;
}
[data-position="top-right"]    { top: 1rem; right: 1rem; }
[data-position="top-left"]     { top: 1rem; left: 1rem; }
[data-position="top-center"]   { top: 1rem; left: 50%; transform: translateX(-50%); }
[data-position="bottom-right"] { bottom: 1rem; right: 1rem; }
[data-position="bottom-left"]  { bottom: 1rem; left: 1rem; }
[data-position="bottom-center"]{ bottom: 1rem; left: 50%; transform: translateX(-50%); }

/* Toast type indicators (optional - uses Pico CSS variables) */
[data-type="success"] { border-left: 4px solid var(--pico-ins-color, #2a9d8f); }
[data-type="error"]   { border-left: 4px solid var(--pico-del-color, #e63946); }
[data-type="warning"] { border-left: 4px solid var(--pico-mark-background-color, #ffc107); }
[data-type="info"]    { border-left: 4px solid var(--pico-primary, #1095c1); }

/* ===== Pagination ===== */
nav[aria-label="Pagination"] ul {
    display: flex;
    flex-wrap: wrap;
    gap: 0.25rem;
    list-style: none;
    padding: 0;
    margin: 0;
}
nav[aria-label="Pagination"] a,
nav[aria-label="Pagination"] span {
    display: inline-block;
    padding: 0.5rem 0.75rem;
    text-decoration: none;
}
nav[aria-label="Pagination"] a[aria-current="page"] {
    background: var(--pico-primary);
    color: var(--pico-primary-inverse);
    border-radius: var(--pico-border-radius);
}

/* ===== Error Summary ===== */
aside[role="alert"] {
    border-left: 4px solid var(--pico-del-color, #e63946);
}
aside[role="alert"]:focus {
    outline: 2px solid var(--pico-primary-focus);
    outline-offset: 2px;
}
```

---

## 6. JavaScript Enhancements

All JavaScript is **optional**. Components work without it.

### 5.1 Modal Utilities (from Pico examples)

```javascript
// modal.js - Optional animation and backdrop click support
// Source: https://github.com/picocss/examples/blob/master/v2-html/js/modal.js

const animationDuration = 400;

export function openModal(dialog) {
    document.documentElement.classList.add("modal-is-open", "modal-is-opening");
    setTimeout(() => {
        document.documentElement.classList.remove("modal-is-opening");
    }, animationDuration);
    dialog.showModal();
}

export function closeModal(dialog) {
    document.documentElement.classList.add("modal-is-closing");
    setTimeout(() => {
        document.documentElement.classList.remove("modal-is-closing", "modal-is-open");
        dialog.close();
    }, animationDuration);
}

// Auto-setup backdrop click
document.addEventListener("click", (event) => {
    const dialog = event.target.closest("dialog[open]");
    if (dialog && event.target === dialog) {
        closeModal(dialog);
    }
});
```

### 5.2 Toast Auto-Dismiss

```javascript
// toast.js - Optional auto-dismiss for toasts

export function showToast(message, { type = "info", duration = 5000, container = "#toasts" } = {}) {
    const toast = document.createElement("article");
    toast.setAttribute("role", type === "error" ? "alert" : "status");
    toast.setAttribute("data-type", type);
    toast.innerHTML = `
        <p>${message}</p>
        <button aria-label="Dismiss"></button>
    `;
    
    toast.querySelector("button").onclick = () => toast.remove();
    
    document.querySelector(container)?.appendChild(toast);
    
    if (duration > 0) {
        setTimeout(() => {
            toast.style.opacity = "0";
            setTimeout(() => toast.remove(), 300);
        }, duration);
    }
    
    return toast;
}
```

### 5.3 Error Summary Focus

```javascript
// error-summary.js - Focus error summary on page load if present

document.addEventListener("DOMContentLoaded", () => {
    const errorSummary = document.querySelector("aside[role='alert']");
    if (errorSummary) {
        errorSummary.focus();
    }
});
```

---

## 7. Migration from Current Design

### Classes to Remove

| Current Class | Replacement |
|--------------|-------------|
| `.dialog` | None (use `<dialog><article>`) |
| `.dialog-header` | `<header>` |
| `.dialog-title` | `<h2>` |
| `.dialog-close` | `<button rel="prev">` |
| `.dialog-content` | Direct children of `<article>` |
| `.dialog-actions` | `<footer>` |
| `.details` | None (native `<details>`) |
| `.details-summary` | `<summary>` |
| `.details-content` | Direct children |
| `.accordion` | None (use `name` attribute) |
| `.pagination` | `aria-label="Pagination"` |
| `.pagination-current` | `aria-current="page"` |
| `.pagination-ellipsis` | `<span aria-hidden="true">` |
| `.toast` | `<article role="status">` |
| `.toast-success` etc | `data-type="success"` |
| `.error-summary` | `<aside role="alert">` |

### Attributes to Add

| Element | Add |
|---------|-----|
| Invalid inputs | `aria-invalid="true"` |
| Valid inputs | `aria-invalid="false"` |
| Current page | `aria-current="page"` |
| Error toast | `role="alert"` |
| Info toast | `role="status"` |
| Error summary | `role="alert" tabindex="-1"` |
| Ellipsis | `aria-hidden="true"` |
| Dialog close | `rel="prev" aria-label="Close"` |

---

### Current Components Requiring Changes

| Component | Change Required |
|-----------|----------------|
| `text_field.pars` | Use `<small>` for hints/errors, remove wrapper div classes |
| `textarea_field.pars` | Same as TextField |
| `select_field.pars` | Same as TextField |
| `checkbox.pars` | Verify Pico checkbox pattern |
| `radio_group.pars` | Verify Pico radio pattern |
| `breadcrumb.pars` | Change `aria-label` to lowercase, simplify classes |
| `skip_link.pars` | Remove inline `<style>`, use supplement CSS |
| `page.pars` | Move `id="main"` to `<main>` element, not `<body>` |
| `form.pars` | Remove `.form` class (optional) |

### New Components to Add (Pico-Compatible)

| Component | Priority | Notes |
|-----------|----------|-------|
| `dialog.pars` | High | `<dialog><article>` pattern |
| `accordion.pars` | High | Native `name` attribute |
| `details.pars` | High | Single expandable section |
| `toast.pars` | Medium | Needs supplement CSS |
| `toasts.pars` | Medium | Container component |
| `pagination.pars` | Medium | Needs supplement CSS |
| `error_summary.pars` | Medium | Needs supplement CSS |

---

## 8. File Structure

```
basil/
├── server/prelude/
│   └── components/          # Prelude components (semantic HTML only)
│       ├── dialog.pars
│       ├── accordion.pars
│       ├── pagination.pars
│       ├── toast.pars
│       ├── error-summary.pars
│       └── ...
├── examples/
│   └── css/
│       ├── README.md        # How to use with Pico
│       └── basil-supplement.css  # Our ~30 line supplement
└── static/
    └── pico/                # Served by admin pages (already exists?)
        ├── pico.min.css
        └── pico.classless.min.css
```

---

## 9. Usage Documentation

### Minimal Setup (Classless)

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta name="color-scheme" content="light dark">
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.classless.min.css">
    <link rel="stylesheet" href="/css/basil-supplement.css">
    <title>My App</title>
</head>
<body>
    <main>
        <h1>Hello World</h1>
    </main>
</body>
</html>
```

### With Classes (Recommended)

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta name="color-scheme" content="light dark">
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.min.css">
    <link rel="stylesheet" href="/css/basil-supplement.css">
    <title>My App</title>
</head>
<body>
    <main class="container">
        <h1>Hello World</h1>
    </main>
</body>
</html>
```

### In Parsley Page Component

```parsley
<Page title="My Page">
    <CSS href="https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.min.css"/>
    <CSS href="/css/basil-supplement.css"/>
    
    <main class="container">
        <h1>"My Page"</h1>
        // Content
    </main>
</Page>
```

---

## 10. Decisions

### DECIDED: All-In on Standard Pico CSS

**Decision:** Use unmodified Pico CSS from CDN or npm.

**Rationale:**
- Users can upgrade Pico independently
- No maintenance burden for us
- Familiar to Pico users
- Supplement file is tiny (~30 lines)

### DECIDED: Use `data-*` Attributes for Semantic Variants

**Decision:** Use `data-type="success"` instead of `.toast-success`.

**Rationale:**
- Keeps classes for user customization
- Semantic meaning in attribute
- Easy to style with attribute selectors
- Works in classless Pico

### DECIDED: Native HTML5 Accordion

**Decision:** Use `<details name="group">` for exclusive accordions.

**Rationale:**
- No JavaScript needed
- Browser handles open/close
- Widely supported (Chrome 120+, Firefox 130+, Safari 17.2+)
- Progressive enhancement — works without `name` in older browsers (just not exclusive)

### OPEN: Button Variants in Classless Mode

**Question:** Should we support button variants (secondary, outline) in classless mode?

**Options:**
1. Require class-based Pico for variants
2. Add `data-variant` support in supplement
3. Don't support variants in classless

**Recommendation:** Option 1 — variants are a progressive enhancement. Document that classless mode gives basic styling, class mode gives full control.

---

## 11. Testing Checklist

For each component, verify:

- [ ] Works unstyled (readable, functional)
- [ ] Works with Pico classless
- [ ] Works with Pico + classes
- [ ] Works with Pico + supplement
- [ ] ARIA attributes correct
- [ ] Keyboard navigation works
- [ ] Screen reader announces correctly
- [ ] Dark mode works (via `color-scheme`)
- [ ] Mobile responsive

---

## 12. Implementation Priority

### Phase 1: Foundation (No Breaking Changes)
1. Create `basil-supplement.css` with skip-link, toast, pagination, error-summary styles
2. Add new components: `Dialog`, `Accordion`, `Details`
3. Document Pico CSS setup for user projects

### Phase 2: Migrate Existing Components
1. Update `TextField`, `TextareaField`, `SelectField` to use `<small>`
2. Update `Breadcrumb` to use Pico pattern
3. Remove inline CSS from `SkipLink`
4. Fix `Page` body id default

### Phase 3: New Components
1. Add `Toast`, `Toasts` components
2. Add `Pagination` component  
3. Add `ErrorSummary` component

---

## 13. References

- [Pico CSS Documentation](https://picocss.com/docs)
- [Pico CSS Examples](https://github.com/picocss/examples)
- [Pico Classless Example](https://raw.githubusercontent.com/picocss/examples/refs/heads/master/v2-html-classless/index.html)
- [GDS Error Summary](https://design-system.service.gov.uk/components/error-summary/)
- [HTML `<details>` MDN](https://developer.mozilla.org/en-US/docs/Web/HTML/Element/details)
- [ARIA Authoring Practices](https://www.w3.org/WAI/ARIA/apg/)