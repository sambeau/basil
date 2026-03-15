# Design: Prelude 1.0 Components

**Date:** 2025-01-14
**Status:** Draft
**Related:** 
- `work/reports/STANDARD-PRELUDE-REVIEW.md` §6, §14
- `work/design/server-enhanced-components.md`

---

## 1. Overview

This document covers new components needed for the standard prelude (`@basil/html`) before or shortly after 1.0. These fill gaps identified in the prelude review — components that every CRUD application needs but are currently missing.

### 1.1 Components Covered

| Component | Priority | Complexity | Notes |
|-----------|----------|------------|-------|
| `Pagination` | High (1.0) | Low | Every list page needs it |
| `Toast` / `Toasts` | High (1.0) | Low-Medium | Form feedback, flash messages |
| `Dialog` | Medium (1.1) | Low | Native `<dialog>`, no heavy JS |
| `Details` / `Accordion` | Medium (1.1) | Low | Native `<details>`, zero JS |
| `ErrorSummary` | Medium (1.1) | Low | Accessibility best practice |
| `FileField` | Medium (1.1) | Medium-High | Deferred — complex, needs design |

### 1.2 Design Principles

1. **Server-rendered first** — Components work without JavaScript where possible
2. **Progressive enhancement** — JS adds polish (animations, auto-dismiss) but isn't required
3. **Accessible by default** — ARIA attributes, keyboard navigation, screen reader support
4. **Composable** — Components work together and with user components
5. **Minimal API** — Few required props, sensible defaults

---

## 2. Pagination

### 2.1 Purpose

Render page navigation for lists. Pure server component — no JavaScript required.

### 2.2 API

```parsley
<Pagination 
    current={page}            // Current page number (required)
    total={totalItems}        // Total item count (required)
    perPage={20}              // Items per page (default: 20)
    href="/products?page={page}"  // URL template with {page} placeholder
    
    // Optional
    window={2}                // Pages to show around current (default: 2)
    showFirst={true}          // Show first/last buttons (default: true)
    showPrev={true}           // Show prev/next buttons (default: true)
    labels={{                 // Custom labels
        first: "«",
        prev: "‹",
        next: "›",
        last: "»",
        page: "Page {n}"      // For aria-label
    }}
    class="my-pagination"
    ...attrs
/>
```

### 2.3 Usage Examples

```parsley
// Basic usage
let page = (request.query.page ?? "1").toInt()
let perPage = 20
let {rows, total} = db.paginate("SELECT * FROM products", page, perPage) // ⚠️ this is not Parsley syntax

<DataTable data={rows}/>
<Pagination current={page} total={total} perPage={perPage} href="/products?page={page}"/>

// With Part-based navigation (no full page reload)
<Pagination 
    current={page} 
    total={total} 
    href="/products?page={page}"
    hx-boost="true" // ⚠️ why HTMX syntax here?
    hx-target="#product-list"// ⚠️ why HTMX syntax here?
/>
```

### 2.4 Output HTML

```html
<nav aria-label="Pagination" class="pagination">
    <a href="/products?page=1" aria-label="First page">«</a>
    <a href="/products?page=2" aria-label="Previous page">‹</a>
    
    <a href="/products?page=1" aria-label="Page 1">1</a>
    <a href="/products?page=2" aria-label="Page 2">2</a>
    <span aria-current="page" class="pagination-current">3</span>
    <a href="/products?page=4" aria-label="Page 4">4</a>
    <a href="/products?page=5" aria-label="Page 5">5</a>
    <span class="pagination-ellipsis" aria-hidden="true">…</span>
    <a href="/products?page=20" aria-label="Page 20">20</a>
    
    <a href="/products?page=4" aria-label="Next page">›</a>
    <a href="/products?page=20" aria-label="Last page">»</a>
</nav>
```

### 2.5 Implementation Notes

```parsley
export Pagination = fn({
    current,
    total,
    perPage = 20,
    href,
    window = 2,
    showFirst = true,
    showPrev = true,
    labels = {},
    class,
    ...attrs
}) {
    let totalPages = (total / perPage).ceil()
    if (totalPages <= 1) { return null }
    
    let first = labels.first ?? "«"
    let prev = labels.prev ?? "‹"
    let next = labels.next ?? "›"
    let last = labels.last ?? "»"
    
    let makeHref = fn(page) {
        href.replace("{page}", page.toString())
    }
    
    // Calculate visible page range
    let startPage = max(1, current - window)
    let endPage = min(totalPages, current + window)
    
    let paginationClass = ["pagination", class].filter(fn(c) { c }).join(" ")
    
    <nav aria-label="Pagination" class={paginationClass} {...attrs}>
        // First/Prev buttons
        if (showFirst and current > 1) {
            <a href={makeHref(1)} aria-label="First page">{first}</a>
        }
        if (showPrev and current > 1) {
            <a href={makeHref(current - 1)} aria-label="Previous page">{prev}</a>
        }
        
        // Page numbers with ellipsis
        if (startPage > 1) {
            <a href={makeHref(1)} aria-label="Page 1">"1"</a>
            if (startPage > 2) {
                <span class="pagination-ellipsis" aria-hidden="true">"…"</span>
            }
        }
        
        for (p in range(startPage, endPage + 1)) {
            if (p == current) {
                <span aria-current="page" class="pagination-current">{p}</span>
            } else {
                <a href={makeHref(p)} aria-label={"Page " + p}>{p}</a>
            }
        }
        
        if (endPage < totalPages) {
            if (endPage < totalPages - 1) {
                <span class="pagination-ellipsis" aria-hidden="true">"…"</span>
            }
            <a href={makeHref(totalPages)} aria-label={"Page " + totalPages}>{totalPages}</a>
        }
        
        // Next/Last buttons
        if (showPrev and current < totalPages) {
            <a href={makeHref(current + 1)} aria-label="Next page">{next}</a>
        }
        if (showFirst and current < totalPages) {
            <a href={makeHref(totalPages)} aria-label="Last page">{last}</a>
        }
    </nav>
}
```

### 2.6 CSS

⚠️ We shouldn’t supply css, just un-styled HTML

```css
.pagination {
    display: flex;
    gap: 0.25rem;
    align-items: center;
    justify-content: center;
    margin: 1rem 0;
}

.pagination a,
.pagination span {
    padding: 0.5rem 0.75rem;
    border-radius: 0.25rem;
    text-decoration: none;
}

.pagination a:hover {
    background: var(--pagination-hover-bg, #f3f4f6);
}

.pagination-current {
    background: var(--pagination-current-bg, #3b82f6);
    color: var(--pagination-current-color, white);
}

.pagination-ellipsis {
    color: var(--text-muted, #6b7280);
}
```

### 2.7 Decisions Needed

#### DECISION: `href` Template vs Callback

**Question:** How should page URLs be generated?

**Options:**
- **A) String template with `{page}`** — `href="/products?page={page}"`
- **B) Callback function** — `href={fn(page) { "/products?page=" + page }}`
- **C) Both** — Template if string, call if function

**Recommendation:** Option A for simplicity. The `{page}` placeholder covers 99% of cases. Users needing complex URLs can build the entire nav themselves.

**Status:** ⏳ Needs decision

---

#### DECISION: Zero-Based vs One-Based Pages

**Question:** Should `current` be 1-based (first page is 1) or 0-based (first page is 0)?

**Options:**
- **A) 1-based** — Matches URL conventions (`?page=1`), human-friendly
- **B) 0-based** — Matches array indexing, offset calculations

**Recommendation:** Option A — 1-based. URLs almost universally use `?page=1` for the first page. Database `OFFSET` calculation is `(page - 1) * perPage`, which is simple enough.

**Status:** ⏳ Needs decision

---

## 3. Toast / Toasts

### 3.1 Purpose

Display flash messages and notifications. Server-rendered with JavaScript enhancement for auto-dismiss and animations.

### 3.2 API

```parsley
// In layout — renders all flash messages
<Toasts 
    position="top-right"      // Position (default: "top-right")
    duration={5000}           // Auto-dismiss after ms (default: 5000, 0 = no auto-dismiss)
/>

// In handlers — set flash messages (existing API)
basil.session.flash("success", "User saved successfully!")
basil.session.flash("error", "Failed to delete item")
basil.session.flash("info", "Your session will expire in 5 minutes")
basil.session.flash("warning", "This action cannot be undone")

// Manual toast (for non-flash use cases)
<Toast type="success" message="Saved!" dismissible={true}/>
```

### 3.3 Usage Examples

```parsley
// In layout (renders flash messages automatically)
<Page title="My App">
    <Toasts position="top-right"/>
    <Nav/>
    <main>{contents}</main>
</Page>

// In a handler
export POST = fn(request) {
    let result = saveUser(request.body)
    if (result.ok) {
        basil.session.flash("success", "User saved!")
    } else {
        basil.session.flash("error", result.error)
    }
    redirect("/users")
}
```

### 3.4 Output HTML

```html
<div class="toasts toasts-top-right" aria-live="polite" aria-atomic="true">
    <div class="toast toast-success" role="alert" data-duration="5000">
        <span class="toast-icon" aria-hidden="true">✓</span>
        <span class="toast-message">User saved successfully!</span>
        <button class="toast-dismiss" aria-label="Dismiss" type="button">×</button>
    </div>
</div>
```

### 3.5 Implementation Notes

```parsley
export Toasts = fn({
    position = "top-right",
    duration = 5000,
    class,
    ...attrs
}) {
    let flashes = basil.session.flashes() ?? []
    if (flashes.len() == 0) { return null }
    
    let icons = {
        success: "✓",
        error: "✗",
        warning: "⚠",
        info: "ℹ"
    }
    
    let containerClass = ["toasts", "toasts-" + position, class].filter(fn(c) { c }).join(" ")
    
    <div class={containerClass} aria-live="polite" aria-atomic="true" {...attrs}>
        for (flash in flashes) {
            <div 
                class={"toast toast-" + flash.type} 
                role="alert"
                data-duration={duration}
            >
                <span class="toast-icon" aria-hidden="true">{icons[flash.type] ?? ""}</span>
                <span class="toast-message">{flash.message}</span>
                <button class="toast-dismiss" aria-label="Dismiss" type="button">"×"</button>
            </div>
        }
    </div>
}

export Toast = fn({
    type = "info",
    message,
    dismissible = true,
    duration,
    class,
    ...attrs
}) {
    let icons = {
        success: "✓",
        error: "✗", 
        warning: "⚠",
        info: "ℹ"
    }
    
    let toastClass = ["toast", "toast-" + type, class].filter(fn(c) { c }).join(" ")
    
    <div 
        class={toastClass} 
        role="alert"
        data-duration={duration}
        {...attrs}
    >
        <span class="toast-icon" aria-hidden="true">{icons[type] ?? ""}</span>
        <span class="toast-message">{message}</span>
        if (dismissible) {
            <button class="toast-dismiss" aria-label="Dismiss" type="button">"×"</button>
        }
    </div>
}
```

### 3.6 JavaScript Enhancement

```javascript
// In basil.js
document.addEventListener('click', (e) => {
    if (e.target.closest('.toast-dismiss')) {
        const toast = e.target.closest('.toast');
        toast.classList.add('toast-dismissing');
        setTimeout(() => toast.remove(), 300);
    }
});

// Auto-dismiss
document.querySelectorAll('.toast[data-duration]').forEach(toast => {
    const duration = parseInt(toast.dataset.duration);
    if (duration > 0) {
        let timeout = setTimeout(() => dismissToast(toast), duration);
        
        // Pause on hover
        toast.addEventListener('mouseenter', () => clearTimeout(timeout));
        toast.addEventListener('mouseleave', () => {
            timeout = setTimeout(() => dismissToast(toast), duration);
        });
    }
});

function dismissToast(toast) {
    toast.classList.add('toast-dismissing');
    setTimeout(() => toast.remove(), 300);
}
```

### 3.7 CSS

⚠️ We shouldn’t supply css for styling, just un-styled HTML. Where CSS should be considered is where layout is vital. However I’d rather we looked at ways to do this without styling being proscribed.


```css
.toasts {
    position: fixed;
    z-index: 1000;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    padding: 1rem;
    pointer-events: none;
}

.toasts > * {
    pointer-events: auto;
}

.toasts-top-right { top: 0; right: 0; }
.toasts-top-left { top: 0; left: 0; }
.toasts-bottom-right { bottom: 0; right: 0; }
.toasts-bottom-left { bottom: 0; left: 0; }
.toasts-top-center { top: 0; left: 50%; transform: translateX(-50%); }
.toasts-bottom-center { bottom: 0; left: 50%; transform: translateX(-50%); }

.toast {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.75rem 1rem;
    border-radius: 0.5rem;
    background: white;
    box-shadow: 0 4px 6px -1px rgb(0 0 0 / 0.1);
    animation: toast-in 0.3s ease-out;
}

.toast-dismissing {
    animation: toast-out 0.3s ease-in forwards;
}

@keyframes toast-in {
    from { opacity: 0; transform: translateY(-1rem); }
    to { opacity: 1; transform: translateY(0); }
}

@keyframes toast-out {
    from { opacity: 1; transform: translateY(0); }
    to { opacity: 0; transform: translateY(-1rem); }
}

.toast-success { border-left: 4px solid #22c55e; }
.toast-error { border-left: 4px solid #ef4444; }
.toast-warning { border-left: 4px solid #f59e0b; }
.toast-info { border-left: 4px solid #3b82f6; }

.toast-icon {
    font-size: 1.25rem;
}

.toast-success .toast-icon { color: #22c55e; }
.toast-error .toast-icon { color: #ef4444; }
.toast-warning .toast-icon { color: #f59e0b; }
.toast-info .toast-icon { color: #3b82f6; }

.toast-dismiss {
    margin-left: auto;
    background: none;
    border: none;
    cursor: pointer;
    opacity: 0.5;
    font-size: 1.25rem;
    line-height: 1;
}

.toast-dismiss:hover {
    opacity: 1;
}
```

### 3.8 Decisions Needed

#### DECISION: Flash Message API

**Question:** How does `Toasts` access flash messages?

**Options:**
- **A) `basil.session.flashes()`** — Returns array, clears them (current assumption)
- **B) Explicit prop** — `<Toasts messages={basil.session.flashes()}/>`
- **C) Both** — Auto-read if no prop, use prop if provided

**Recommendation:** Option C — Auto-read for convenience, prop for testing/custom use.

**Status:** ⏳ Needs decision

---

#### DECISION: Position Values

**Question:** What position values should be supported?

**Options:**
- **A) Six positions** — top-right, top-left, top-center, bottom-right, bottom-left, bottom-center
- **B) Four positions** — top-right, top-left, bottom-right, bottom-left
- **C) Two positions** — top-right (default), top-left

**Recommendation:** Option A — All six are easy to implement and cover all use cases.

**Status:** ⏳ Needs decision

---

## 4. Dialog

### 4.1 Purpose

Modal dialogs using native `<dialog>` element. Minimal JavaScript for polish.

### 4.2 API

```parsley
<Dialog 
    id="confirm-delete"       // Required for targeting
    title="Confirm Delete"    // Dialog title
    modal={true}              // Modal (default) vs non-modal
    closeOnBackdrop={true}    // Close when clicking backdrop (default: true)
>
    <p>"Are you sure you want to delete this item?"</p>
    <div class="dialog-actions">
        <Button onclick="this.closest('dialog').close()">"Cancel"</Button>
        <Button type="submit" form="delete-form" variant="danger">"Delete"</Button>
    </div>
</Dialog>

// Trigger
<Button onclick="document.getElementById('confirm-delete').showModal()">"Delete"</Button>
```

### 4.3 Output HTML

```html
<dialog id="confirm-delete" class="dialog" data-close-on-backdrop="true">
    <header class="dialog-header">
        <h2 class="dialog-title">Confirm Delete</h2>
        <button type="button" class="dialog-close" aria-label="Close" 
                onclick="this.closest('dialog').close()">×</button>
    </header>
    <div class="dialog-content">
        <p>Are you sure you want to delete this item?</p>
        <div class="dialog-actions">
            <button onclick="this.closest('dialog').close()">Cancel</button>
            <button type="submit" form="delete-form" class="btn-danger">Delete</button>
        </div>
    </div>
</dialog>
```

### 4.4 Implementation Notes

```parsley
export Dialog = fn({
    id,
    title,
    modal = true,
    closeOnBackdrop = true,
    class,
    contents,
    ...attrs
}) {
    let dialogClass = ["dialog", class].filter(fn(c) { c }).join(" ")
    
    <dialog 
        id={id} 
        class={dialogClass}
        data-close-on-backdrop={if (closeOnBackdrop) { "true" } else { null }}
        {...attrs}
    >
        if (title) {
            <header class="dialog-header">
                <h2 class="dialog-title">{title}</h2>
                <button 
                    type="button" 
                    class="dialog-close" 
                    aria-label="Close"
                    onclick="this.closest('dialog').close()"
                >"×"</button>
            </header>
        }
        <div class="dialog-content">
            {contents}
        </div>
    </dialog>
}
```

### 4.5 JavaScript Enhancement

```javascript
// Backdrop click to close
document.addEventListener('click', (e) => {
    const dialog = e.target.closest('dialog[data-close-on-backdrop="true"]');
    if (dialog && e.target === dialog) {
        dialog.close();
    }
});

// Auto-focus first focusable element
document.addEventListener('open', (e) => {
    if (e.target.tagName === 'DIALOG') {
        const focusable = e.target.querySelector(
            'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
        );
        if (focusable) focusable.focus();
    }
}, true);
```

### 4.6 CSS

⚠️ We shouldn’t supply css for styling, just un-styled HTML. Where CSS should be considered is where layout is vital. However I’d rather we looked at ways to do this without styling being proscribed.

```css
.dialog {
    border: none;
    border-radius: 0.5rem;
    padding: 0;
    max-width: 32rem;
    box-shadow: 0 25px 50px -12px rgb(0 0 0 / 0.25);
}

.dialog::backdrop {
    background: rgb(0 0 0 / 0.5);
}

.dialog-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 1rem 1.5rem;
    border-bottom: 1px solid var(--border-color, #e5e7eb);
}

.dialog-title {
    margin: 0;
    font-size: 1.125rem;
}

.dialog-close {
    background: none;
    border: none;
    font-size: 1.5rem;
    cursor: pointer;
    opacity: 0.5;
    line-height: 1;
}

.dialog-close:hover {
    opacity: 1;
}

.dialog-content {
    padding: 1.5rem;
}

.dialog-actions {
    display: flex;
    gap: 0.75rem;
    justify-content: flex-end;
    margin-top: 1.5rem;
}
```

### 4.7 Decisions Needed

#### DECISION: Naming — `Dialog` vs `Modal`

**Question:** What should the component be called?

**Options:**
- **A) `Dialog`** — Matches HTML element name (`<dialog>`)
- **B) `Modal`** — More common term in UI frameworks
- **C) Both** — `Dialog` with `Modal` as alias

**Recommendation:** Option A — `Dialog` is accurate and matches the HTML. "Modal" is a property of how the dialog is shown (`showModal()` vs `show()`), not the component itself.

**Status:** ⏳ Needs decision

---

#### DECISION: Non-Modal Support

**Question:** Should we support non-modal dialogs (`dialog.show()` instead of `dialog.showModal()`)?

**Options:**
- **A) Modal only** — Simpler, covers most use cases
- **B) Support both** — `modal` prop controls behavior
- **C) Separate component** — `Dialog` for modal, `Popover` for non-modal

**Recommendation:** Option B — The `modal` prop provides flexibility without complexity. Non-modal dialogs are useful for tooltips and dropdowns.

**Status:** ⏳ Needs decision

---

## 5. Details / Accordion

### 5.1 Purpose

Disclosure widgets using native `<details>`/`<summary>`. Zero JavaScript for single disclosures; minimal JS for exclusive accordion behavior.

### 5.2 API

```parsley
// Single disclosure
<Details summary="More information" open={false}>
    <p>"Hidden content revealed when expanded."</p>
</Details>

// Accordion (only one open at a time)
<Accordion>
    <Details summary="Section 1">"Content for section 1"</Details>
    <Details summary="Section 2">"Content for section 2"</Details>
    <Details summary="Section 3">"Content for section 3"</Details>
</Accordion>

// Accordion with one initially open
<Accordion>
    <Details summary="Section 1" open={true}>"Content 1"</Details>
    <Details summary="Section 2">"Content 2"</Details>
</Accordion>
```

### 5.3 Output HTML

```html
<!-- Single Details -->
<details class="details">
    <summary class="details-summary">More information</summary>
    <div class="details-content">
        <p>Hidden content revealed when expanded.</p>
    </div>
</details>

<!-- Accordion -->
<div class="accordion" data-exclusive="true">
    <details class="details accordion-item">
        <summary class="details-summary">Section 1</summary>
        <div class="details-content">Content for section 1</div>
    </details>
    <details class="details accordion-item">
        <summary class="details-summary">Section 2</summary>
        <div class="details-content">Content for section 2</div>
    </details>
</div>
```

### 5.4 Implementation Notes

```parsley
export Details = fn({
    summary,
    open = false,
    class,
    contents,
    ...attrs
}) {
    let detailsClass = ["details", class].filter(fn(c) { c }).join(" ")
    
    <details class={detailsClass} open={if (open) { true } else { null }} {...attrs}>
        <summary class="details-summary">{summary}</summary>
        <div class="details-content">
            {contents}
        </div>
    </details>
}

export Accordion = fn({
    exclusive = true,
    class,
    contents,
    ...attrs
}) {
    let accordionClass = ["accordion", class].filter(fn(c) { c }).join(" ")
    
    <div 
        class={accordionClass} 
        data-exclusive={if (exclusive) { "true" } else { null }}
        {...attrs}
    >
        {contents}
    </div>
}
```

### 5.5 JavaScript Enhancement (Accordion Only)

```javascript
// Exclusive accordion behavior
document.addEventListener('toggle', (e) => {
    if (e.target.tagName !== 'DETAILS') return;
    if (!e.target.open) return;
    
    const accordion = e.target.closest('[data-exclusive="true"]');
    if (!accordion) return;
    
    // Close other open details in the same accordion
    accordion.querySelectorAll('details[open]').forEach(details => {
        if (details !== e.target) {
            details.open = false;
        }
    });
}, true);
```

### 5.6 CSS

```css
.details {
    border: 1px solid var(--border-color, #e5e7eb);
    border-radius: 0.5rem;
    margin-bottom: 0.5rem;
}

.details-summary {
    padding: 1rem;
    cursor: pointer;
    font-weight: 500;
    list-style: none;
}

.details-summary::-webkit-details-marker {
    display: none;
}

.details-summary::before {
    content: "▸";
    margin-right: 0.5rem;
    transition: transform 0.2s;
}

.details[open] .details-summary::before {
    transform: rotate(90deg);
}

.details-content {
    padding: 0 1rem 1rem;
}

.accordion {
    display: flex;
    flex-direction: column;
}

.accordion .details {
    border-radius: 0;
    margin-bottom: 0;
    border-bottom: none;
}

.accordion .details:first-child {
    border-radius: 0.5rem 0.5rem 0 0;
}

.accordion .details:last-child {
    border-radius: 0 0 0.5rem 0.5rem;
    border-bottom: 1px solid var(--border-color, #e5e7eb);
}
```

### 5.7 Decisions Needed

#### DECISION: Single vs Two Components

**Question:** Should `Details` and `Accordion` be separate components, or one component with a mode prop?

**Options:**
- **A) Two components** — `Details` for single, `Accordion` as wrapper for exclusive groups
- **B) One component** — `<Details exclusive={true}>` wraps children in accordion mode
- **C) Three components** — `Details`, `Accordion`, and `AccordionItem`

**Recommendation:** Option A — Two components is cleaner. `Details` maps directly to `<details>`, `Accordion` is a wrapper that adds exclusive behavior. No need for `AccordionItem` since `Details` works inside `Accordion`.

**Status:** ⏳ Needs decision

---

## 6. ErrorSummary

### 6.1 Purpose

Display form validation errors in an accessible summary at the top of a form. Follows GOV.UK Design System pattern.

### 6.2 API

```parsley
// With record (auto-derives errors)
<ErrorSummary record={user}/>

// Manual mode
<ErrorSummary 
    errors={{
        email: "Email is required",
        password: "Password must be at least 8 characters"
    }}
    title="There are problems with your form"
/>

// Array of errors (for non-field errors)
<ErrorSummary 
    errors={["Please fix the errors below", "Session expired"]}
    title="Error"
/>
```

### 6.3 Output HTML

```html
<div class="error-summary" role="alert" aria-labelledby="error-summary-title">
    <h2 id="error-summary-title" class="error-summary-title">
        There are problems with your form
    </h2>
    <ul class="error-summary-list">
        <li>
            <a href="#field-email-input">Email — is required</a>
        </li>
        <li>
            <a href="#field-password-input">Password — must be at least 8 characters</a>
        </li>
    </ul>
</div>
```

### 6.4 Implementation Notes

```parsley
export ErrorSummary = fn({
    record,
    errors,
    title = "There is a problem",
    class,
    ...attrs
}) {
    // Derive errors from record if provided
    let errorList = if (record) {
        record.errorList()  // Returns [{field: "email", message: "is required"}, ...]
    } else if (errors.isDict()) {
        errors.entries().map(fn(e) { {field: e.key, message: e.value} })
    } else if (errors.isArray()) {
        errors.map(fn(msg) { {field: null, message: msg} })
    } else {
        []
    }
    
    if (errorList.len() == 0) { return null }
    
    // Get field titles from schema if available
    let schema = if (record) { record.schema() } else { null }
    let getTitle = fn(field) {
        if (schema) {
            schema.title(field) ?? field.replace("_", " ").toTitleCase()
        } else {
            field.replace("_", " ").toTitleCase()
        }
    }
    
    let summaryClass = ["error-summary", class].filter(fn(c) { c }).join(" ")
    let titleId = "error-summary-title-" + (record.id() ?? "form")
    
    <div class={summaryClass} role="alert" aria-labelledby={titleId} {...attrs}>
        <h2 id={titleId} class="error-summary-title">{title}</h2>
        <ul class="error-summary-list">
            for (err in errorList) {
                <li>
                    if (err.field) {
                        <a href={"#field-" + err.field + "-input"}>
                            {getTitle(err.field)} " — " {err.message}
                        </a>
                    } else {
                        {err.message}
                    }
                </li>
            }
        </ul>
    </div>
}
```

### 6.5 CSS

⚠️ We shouldn’t supply css for styling, just un-styled HTML. Where CSS should be considered is where layout is vital. However I’d rather we looked at ways to do this without styling being proscribed.

```css
.error-summary {
    padding: 1rem 1.5rem;
    margin-bottom: 1.5rem;
    border: 4px solid var(--error-color, #d4351c);
    border-radius: 0;
}

.error-summary-title {
    margin: 0 0 0.5rem;
    font-size: 1.125rem;
}

.error-summary-list {
    margin: 0;
    padding-left: 1.25rem;
}

.error-summary-list li {
    margin-bottom: 0.25rem;
}

.error-summary-list a {
    color: var(--error-color, #d4351c);
    font-weight: 500;
}
```

### 6.6 Decisions Needed

#### DECISION: Link Target ID Format

**Question:** What ID format should error links target?

**Options:**
- **A) `#field-{name}-input`** — Matches `@field` generated IDs
- **B) `#{name}`** — Simple, assumes user sets id manually
- **C) Configurable** — `fieldIdTemplate` prop

**Recommendation:** Option A — Consistent with `@field` and `<field/>` tag output. Users with custom IDs can use `render` functions or manual error lists.

**Status:** ⏳ Needs decision

---

#### DECISION: Auto-Focus

**Question:** Should the ErrorSummary auto-focus when it appears?

**Options:**
- **A) Yes** — Focus the summary on page load if it has errors
- **B) No** — Let the page flow naturally
- **C) Configurable** — `autoFocus` prop

**Recommendation:** Option C — Auto-focus is good for form validation flows but intrusive in other contexts. Default to `true` when inside a form, `false` otherwise.

**Status:** ⏳ Needs decision

---

## 7. FileField (Deferred)

`FileField` is more complex than the other components and needs its own design document. Key considerations:

- Basic file input vs drag-drop with preview
- Upload endpoint contract
- Progress indication
- Multi-file support
- File type and size validation
- Thumbnail previews for images
- Integration with form submission

**Recommendation:** Defer to 1.1 or create separate `DESIGN-file-field.md`.

---

## 8. Implementation Order

### Phase 1: Pure Server Components (1.0)
1. **Pagination** — No JS required, high value
2. **Details** — Native `<details>`, zero JS for single use

### Phase 2: Server + Light JS (1.0 or early 1.1)
3. **Toasts** — Server renders, JS adds polish
4. **ErrorSummary** — Server rendered, optional focus JS
5. **Accordion** — Needs toggle event listener for exclusive mode

### Phase 3: Native Element Wrappers (1.1)
6. **Dialog** — Native `<dialog>`, light JS for backdrop click

### Phase 4: Complex Components (1.1+)
7. **FileField** — Needs separate design

**Total estimated effort:** 8-12 hours for phases 1-3

---

## 9. Testing Strategy

Each component needs:
1. **Render tests** — Correct HTML output for various prop combinations
2. **Accessibility tests** — ARIA attributes, roles, keyboard navigation
3. **Edge cases** — Empty states, null values, missing props
4. **Integration tests** — Works with real data (flash messages, records, Tables)

Example test structure:

```go
func TestPagination(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        contains []string
    }{
        {
            name: "basic pagination",
            input: `<Pagination current={3} total={100} perPage={10} href="/p?page={page}"/>`,
            contains: []string{
                `aria-label="Pagination"`,
                `aria-current="page"`,
                `href="/p?page=2"`,
                `href="/p?page=4"`,
            },
        },
        {
            name: "single page hides pagination",
            input: `<Pagination current={1} total={5} perPage={10} href="/p?page={page}"/>`,
            contains: []string{}, // Should return null
        },
    }
    // ... test execution
}
```

---

## 10. CSS Bundle Considerations

All CSS for these components should be added to the prelude CSS bundle. Estimated additions:

| Component | CSS Lines | Notes |
|-----------|-----------|-------|
| Pagination | ~30 | Flexbox layout, hover states |
| Toasts | ~60 | Positioning, animations, variants |
| Dialog | ~40 | Native dialog styling, backdrop |
| Details/Accordion | ~50 | Summary styling, transitions |
| ErrorSummary | ~20 | GOV.UK-style error box |
| **Total** | ~200 | |

The CSS should use CSS custom properties for theming consistency with the rest of the prelude.