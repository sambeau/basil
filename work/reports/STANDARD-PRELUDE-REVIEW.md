# Standard Prelude Review

**Date:** 2026-03-15
**Status:** Report (decisions resolved)
**Purpose:** Critical review of the standard HTML prelude for Basil 1.0 readiness
**Companion documents:**
- `work/reports/STDLIB-1.0-RELEASE-REVIEW.md`
- `work/reports/STDLIB-1.0-ACTION-PLAN.md`
- `work/design/server-enhanced-components.md`
- `work/design/sortable-list-component.md`
**Design documents (from this review):**
- `work/design/DESIGN-typed-value-formatting.md` — Decisions #2 and #3 (objectToString changes, record.fieldProps)
- `work/design/DESIGN-prelude-meta-component.md` — Decision #1 (Head → Meta rename)

---

## Executive Summary

The standard HTML prelude (`@basil/html`) contains 26 components across 7 categories. The component quality is generally high — accessibility patterns are solid, form components follow consistent conventions, and the progressive enhancement philosophy is well-executed. However, several structural and strategic issues need addressing before 1.0:

1. **Namespacing** — Component names will clash with user-defined components (especially `Page`, `Form`, `Button`, `Nav`). We need a strategy.
2. **Flexibility gaps** — `Page` and `Head` are too rigid for real-world use; extending them is awkward.
3. **Missing components** — Pagination, Toasts, and FileField are conspicuously absent for the target audience.
4. **Inconsistencies** — Some components use prop spreading (`...attrs`), others enumerate every prop explicitly. This affects extensibility.
5. **`@field` form binding is disconnected** — The powerful `@record`/`@field` system in `eval_tags.go` and `form_binding.go` doesn't interact with prelude components. Two parallel form systems exist.
6. **Localisation** — Time components are good; everything else (numbers, currency, dates outside time components) has no localisation story.
7. **JavaScript** — `basil.js` is lean (~290 lines) but could do more with custom elements for toast dismissal, pagination, and form validation feedback.

**Overall verdict:** The prelude is a strong foundation. With the changes proposed in this report, it can be an excellent 1.0 release. The work is mostly additive — improving what exists rather than rewriting it.

**FEAT-143 verification addendum (2026-03-15):** A later implementation review of the Pico CSS compatibility work confirms that the design direction in this report was broadly correct, but also found several blocking Parsley-level correctness issues that this review did not catch because it was primarily strategic and architectural rather than an executable-language audit. In particular, FEAT-143's new and updated `.pars` components include invalid tag spread syntax (`{...attrs}` instead of `...attrs`), incorrect `for` loop index/value ordering in several components, a pagination range bug, and spec/design examples that were copied into implementation despite not matching current Parsley idioms. This means FEAT-143 should not be treated as "done" merely because the components exist; Parsley correctness and focused component tests are required. See Appendix D for the detailed reconciliation.

---

## Decisions Summary

Three blocking design decisions have been resolved. These were required before design documents could be written.

| Decision | Resolution | Section |
|----------|------------|---------|
| **Head / Page relationship** | **Option A: Separate** — Rename `Head` → `Meta`, strip `<head>` wrapper, use inside `Page`'s `head` prop. Rationale: separation of concerns, composability, manageable prop counts. | §7.4 |
| **Form field abstraction** | **Layered approach** — Four levels: (1) `<field name="x"/>` for terse complete fields, (2) `@field` attributes for custom structure, (3) `fieldProps()` method for component library authors, (4) manual props. Each level serves distinct use cases without overlap. | §9.3 |
| **Typed value formatting** | **Change both `objectToString()` and `objectToPrintString()`** — Call `.medium()` on `money`, `datetime`, `unit`, `duration`. Breaking change acceptable pre-1.0. Goal: human-readable output by default. | §9.5 |

---

## Table of Contents

1. [Easy Namespacing](#1-easy-namespacing)
2. [Consistent Naming](#2-consistent-naming)
3. [Fit-for-Purpose](#3-fit-for-purpose)
4. [Modern Standards](#4-modern-standards)
5. [Accessibility Standards](#5-accessibility-standards)
6. [Common Usage](#6-common-usage)
7. [Flexibility, Customisation, and Enhancibility](#7-flexibility-customisation-and-enhancibility)
8. [Internationalisation and Localisation](#8-internationalisation-and-localisation)
9. [Fit With New Parsley Features](#9-fit-with-new-parsley-features)
10. [JavaScript Enhancements](#10-javascript-enhancements)
11. [Parsley Language Enhancements](#11-parsley-language-enhancements)
12. [Previously Discussed Components](#12-previously-discussed-components)
13. [Component-by-Component Review](#13-component-by-component-review)
14. [Proposed New Components](#14-proposed-new-components)
15. [Summary of Recommendations](#15-summary-of-recommendations)
16. [FEAT-143 Verification Addendum](#16-feat-143-verification-addendum)

---

## 1. Easy Namespacing

### The Problem

The prelude exports names like `Page`, `Form`, `Button`, `Nav`, `A`, `Img`, `Time` — exactly the names a user would choose for their own components. This is by design (guessable names), but it creates an immediate practical problem: the first thing most developers do is create their own `<Page/>` wrapper, and it clashes.

The current import mechanism is:

```parsley
{Page, Form, TextField} = import @basil/html
```

But in practice, prelude components are auto-available (they're evaluated into the shared prelude environment), so there's no explicit import — and therefore no opportunity to rename at the import site.

### Analysis of Options

| Option | Syntax | Pros | Cons |
|--------|--------|------|------|
| **A. Prefix convention** | `<B.Page/>` or `<Basil.Page/>` | Clear provenance, no language change | Verbose, ugly, needs dot-access on components |
| **B. Lowercase prefix** | `<b-page/>` or `<basil-page/>` | Web-component style, familiar to frontend devs | Breaks Parsley convention (components are PascalCase), looks like HTML custom elements |
| **C. Sigil/namespace import** | `html.Page` via `import @basil/html as html` | Explicit, Parsley-idiomatic | Requires `as` import syntax to be comfortable |
| **D. Opt-out auto-import** | User components shadow prelude | Zero boilerplate for common case | Silent shadowing can be confusing; no way to access original |
| **E. Explicit re-export** | Prelude defines `BasilPage`, user aliases to `Page` | No clash, user controls naming | Prelude names become ugly; defeats "guessable" goal |
| **F. Qualified access always available** | `<@basil.Page/>` or `<html:Page/>` | Always accessible even when shadowed | New syntax needed; XML namespace feel |

### Recommendation

**Option D (current behaviour) + Option C (qualified fallback) + documentation guidance.**

The current auto-import with shadowing is actually the right default for Parsley's aesthetic. If a user defines their own `Page`, they probably *want* to replace the prelude's `Page`. The problem is when they want to *use* the prelude's `Page` inside their own `Page`.

The fix is to ensure `import @basil/html as html` works and gives `html.Page`, `html.Form`, etc. This is already possible with Parsley's existing import/destructuring syntax:

```parsley
// User's page.pars
let html = import @basil/html

export Page = fn({title, description, contents}) {
    <html.Page lang="en" title={title} description={description}>
        <MyNav/>
        <main id="content">
            (contents)
        </main>
        <MyFooter/>
    </html.Page>
}
```

**What's needed:**
1. Verify that `import @basil/html` returns a dictionary that supports dot-access for all 26 components.
2. Document this pattern prominently in the prelude guide.
3. Consider whether a shorter alias is warranted (e.g., `@basil/html` could also be accessible as `@html` with a deprecation path, or we document `let h = import @basil/html` as the convention).

**Additional idea — `@std` alias:** Since `@basil/html` is the most-used import, consider whether it could also be available as `@html` or `@ui`. This is purely ergonomic.

### What We Should NOT Do

- Add a prefix to all component names (`BasilPage`, `BPage`). This makes the 90% case (no clash) ugly for the 10% case (clash).
- Introduce XML-style namespaces (`<html:Page/>`). This is over-engineering and alien to Parsley's aesthetic.
- Remove auto-import. The zero-config experience of `<Page>` just working is a key selling point.

---

## 2. Consistent Naming

### Current State

| Convention | Examples | Consistent? |
|------------|----------|-------------|
| PascalCase for components | `Page`, `TextField`, `DataTable` | ✅ Yes |
| HTML element name as base | `Form` → `<form>`, `Nav` → `<nav>` | ✅ Mostly |
| Suffix for field types | `TextField`, `SelectField`, `TextareaField` | ⚠️ Inconsistent — `Checkbox` and `Button` have no `Field` suffix |
| Group suffix | `RadioGroup`, `CheckboxGroup` | ✅ Yes |
| Semantic naming | `SkipLink`, `SrOnly`, `DataTable` | ✅ Yes |

### Issues Found

1. **`Checkbox` vs `CheckboxField`** — Standalone `Checkbox` lacks the `Field` suffix that `TextField`, `SelectField`, and `TextareaField` have. It should be `CheckboxField` for consistency, with `Checkbox` kept as a deprecated alias.

2. **`A` is too terse** — While `<A>` mirrors `<a>`, it's cryptic and hard to search for. `Link` would be far more guessable and searchable. `<A>` should become a deprecated alias for `<Link>`.

3. **`SrOnly` abbreviation** — "Sr" is not universally obvious. However, `.sr-only` is an established CSS convention (Bootstrap, Tailwind), so this is acceptable. Document it well.

4. **`DataTable` vs `Table`** — `DataTable` is correct because it's not a raw `<table>` wrapper — it's an opinionated data display component. Good name.

5. **`Img` vs `Image`** — `Img` mirrors the HTML tag, but `Image` would be more guessable for someone who doesn't think in HTML tag names. Borderline — keep `Img` but consider adding `Image` as an alias.

6. **`Abbr` vs `Abbreviation`** — Same as `Img`. Keep `Abbr` (HTML convention) but document it well.

### Naming Principles (Proposed)

For the documentation and for future components:

1. **PascalCase always** — `TextField`, not `textField` or `text_field`.
2. **Form inputs get the `Field` suffix** — `TextField`, `SelectField`, `FileField`, `NumberField`. Exception: `Button` (it's not a field).
3. **Groups get the `Group` suffix** — `RadioGroup`, `CheckboxGroup`.
4. **Use the most guessable English word** — `Link` over `A`, `Image` over `Img` (as aliases).
5. **Compound names describe the component's purpose** — `DataTable` (not just `Table`), `SkipLink` (not just `Skip`), `SrOnly` (established convention).

### Recommended Changes

| Current | Proposed | Action |
|---------|----------|--------|
| `A` | `Link` (primary), `A` (alias) | Rename with alias |
| `Checkbox` | `CheckboxField` (primary), `Checkbox` (alias) | Rename with alias |
| `Img` | Keep `Img`, add `Image` alias | Add alias |
| All others | No change | — |

---

## 3. Fit-for-Purpose

### What "Fit-for-Purpose" Means for Basil

The prelude should provide:
- A quick, correct default for every common HTML pattern
- Built-in accessibility without the developer needing to know ARIA
- Safety defaults (CSRF, `rel="noopener"`, lazy loading)
- Enough flexibility to customise without replacing entirely

### Assessment by Category

#### Form Components — Excellent ✅

The form components (`TextField`, `TextareaField`, `SelectField`, `RadioGroup`, `CheckboxGroup`, `Checkbox`, `Button`, `Form`) are the strongest part of the prelude.

**What they do well:**
- Consistent `name`/`label`/`hint`/`error`/`required` prop pattern across all field types
- Proper `for`/`id` association with auto-generated IDs
- `aria-describedby` links hint and error text to the input
- `aria-invalid` and `role="alert"` for error states
- Prop spreading via `...inputAttrs` — user can pass any HTML attribute
- CSRF auto-injection in `Form`
- Submit button disable-on-submit via `basil.js`

**What's missing:**
- No `FileField` — file uploads are extremely common for the target audience (SMB sites with image uploads, document forms). This is the only genuinely missing form component; `NumberField` and `HiddenField` are not needed because `<TextField type="number" min={0} max={100} step={1}/>` and `<input type="hidden" name="x" value="y"/>` already work via prop spreading and raw HTML respectively — there is nothing to enhance.

**Consistency bug (see also §13 `TextField`):**
- `TextField` uses `required={if (required) "required" else null}` (string value) while other components use `required={required}` (boolean). HTML output differs: `required="required"` vs `required="true"`. Should be normalised to the boolean form.

#### Layout Components — Good but Rigid ⚠️

`Page` and `Head` work for simple cases but are difficult to extend.

**`Page` issues:**
- `head` prop for additional `<head>` content exists but ordering is unclear — does `<CSS/>` come before or after user's head content? (Currently: CSS first, then user content — correct for overrides, but user might want to add CSS *before* the bundle)
- No `bodyAttrs` or `htmlAttrs` prop for adding attributes to `<body>` or `<html>` (e.g., `data-theme="dark"`, `dir="rtl"`)
- `id` defaults to `"main"` on `<body>` — this conflicts with the common pattern of having a `<main id="main">` inside the body. The body shouldn't have `id="main"` by default.
- No `footer` or `afterBody` slot for scripts that must come at the very end
- `lang` defaults to `"en"` silently — should this be explicit or configurable globally?

**`Head` issues:**
- `Head` is actually a standalone component that generates its own `<head>` tag, but `Page` also generates a `<head>`. These are two different things with confusingly similar names.
- `Head` includes `<CSS/>` but `Page` also includes `<CSS/>` — if you use `Head` inside `Page`, you might get double CSS bundles.
- The relationship between `Page` and `Head` needs clarification: are they alternatives or composable?

**Recommendation:** `Head` should be renamed to something like `Meta` or `SEO` (since it's really about Open Graph, Twitter Cards, favicons, and SEO meta tags) and should NOT generate its own `<head>` wrapper — it should just output the meta tags for insertion into `Page`'s `head` prop. Or alternatively, `Page` should accept the same props that `Head` has (`image`, `url`, `twitter`, etc.) and delegate to `Head` internally.

#### Navigation — Good ✅

`Nav`, `Breadcrumb`, `SkipLink` — all correct and well-implemented. `Breadcrumb` even has Schema.org structured data markup, which is excellent.

**Minor issue:** `SkipLink` has its CSS inlined via a `<style>` tag. This means every page that uses `SkipLink` (which is every page via `Page`) gets a `<style>` block. This should be moved to the CSS bundle or at minimum deduplicated.

#### Media — Good ✅

`Img`, `Iframe`, `Figure`, `Blockquote` — all solid.

**`Img` note:** Defaults to `loading="lazy"` and `decoding="async"`, which is correct for most images. However, above-the-fold hero images should NOT be lazy-loaded (it hurts LCP — Largest Contentful Paint). Consider a `priority` or `eager` prop that sets `loading="eager"` and `decoding="auto"` and optionally adds `fetchpriority="high"`.

**`Iframe` note:** The component enumerates props explicitly rather than using prop spreading. This means users can't add `data-*` attributes or other arbitrary attributes. Should use `...iframeAttrs` pattern.

#### Time Components — Excellent ✅

`Time`, `LocalTime`, `TimeRange`, `RelativeTime` — these are some of the best components in the prelude. The progressive enhancement pattern (server renders UTC fallback, client-side custom elements enhance with local timezone) is exactly right.

**One concern:** `LocalTime`, `TimeRange`, and `RelativeTime` all require `basil.js` (via custom elements). The `noBasilJS` prop on `Page` would break them silently. Should document this dependency clearly, or have the time components detect missing JS and degrade gracefully (they already do — the server fallback remains — but a console warning would help developers debug).

#### Data — Needs Rethinking ⚠️

`DataTable` is functional but its design predates the Parsley `Table` type, and it shows. The current API requires the user to decompose their data into parallel arrays:

```parsley
<DataTable caption="Users" columns={["Name", "Email", "Role"]} rows={users} keys={["name", "email", "role"]}/>
```

But if `users` is already a `Table`, it knows its `.columns` and `.rows` — the user is being asked to re-specify structure that the data already carries. Meanwhile, `Table` already has a `.toHTML(footer?)` method that renders a basic `<table>` (with optional footer and colspan support), but without any of the accessibility attributes that `DataTable` provides (`scope="col"`, `scope="row"`, `<caption>`, semantic class names).

**The two features overlap but neither is complete:**

| Capability | `Table.toHTML()` | `<DataTable/>` |
|-----------|-----------------|----------------|
| Knows columns automatically | ✅ From `Table.Columns` | ❌ Must pass `columns` and `keys` props |
| Knows rows automatically | ✅ From `Table.Rows` | ❌ Must pass `rows` prop |
| `<caption>` | ❌ | ✅ |
| `scope="col"` / `scope="row"` | ❌ | ✅ |
| `<thead>` / `<tbody>` | ✅ | ✅ |
| `<tfoot>` with aggregates | ✅ Footer with colspan | ❌ |
| Schema-aware (types) | ✅ `Table.Schema` available | ❌ |
| Cell formatting | ❌ Calls `objectToString()` | ❌ Outputs raw values |
| Empty state | ❌ | ❌ |
| Custom cell rendering | ❌ | ❌ |
| CSS classes / ID | ❌ | ✅ |
| Prop spreading | ❌ (Go string output) | ❌ (enumerates props) |

**The right design:** `DataTable` should primarily be a richer, accessible way to render a Parsley `Table` object. The `Table` type handles data preparation (`.where()`, `.orderBy()`, `.select()`, `.limit()`, `.offset()`); `DataTable` handles presentation. They should compose naturally:

```parsley
let users = db.query("SELECT name, email, role FROM users")
    .orderBy("name")
    .limit(20)

// Simple — derive everything from the Table
<DataTable data={users} caption="User list"/>

// Override column headers (Table columns are field names, not display names)
<DataTable data={users} caption="User list" columns={["Name", "Email", "Role"]}/>

// Keep manual mode as fallback for raw arrays
<DataTable rows={arrayData} columns={["Name", "Email"]} keys={["name", "email"]}/>
```

**What enhancements matter, in priority order:**

1. **Accept `Table` directly** — derive columns/rows automatically; keep current `rows`/`columns`/`keys` props as manual fallback for raw arrays of dictionaries
2. **Empty state** — a message when there are no rows (`empty` prop, default: "No data")
3. **Cell formatting based on built-in types** — Parsley's built-in types already carry formatting intelligence. When a cell value is a `money` object, `DataTable` should call `.medium()` automatically (→ "£49.99"); when it's a `datetime`, call `.medium()` (→ "Mar 15, 2025") or wrap in `<LocalTime>` for client-side locale enhancement; when it's a `unit`, call `.medium()` (→ "5.00kg"). For schema-bound tables, the schema's field types provide a second signal (`money`, `date`, `datetime`, `percent`) even when the raw cell value is a plain number or string — `record.format(field)` already implements this mapping. Plain strings and numbers render as-is. A per-column `format` prop (`format={{price: "currency", created: "date"}}`) should exist as an *override* for when auto-detection isn't enough, not as the primary mechanism. Similarly, column alignment can be auto-derived: `money` and numeric values default to right-alignment, everything else to left.
4. **Custom cell rendering** — a `render` prop with per-column render functions for links, buttons, badges: `render={{email: fn(val, row) { <A href={"mailto:" ++ val}>{val}</A> }}}`
5. **Footer/summary row** — `Table` already has `.sum()`, `.avg()`, `.count()`. A `footer` prop could accept a dict of column→aggregate or column→value: `footer={{salary: users.avg("salary").format("currency")}}`
6. **Column alignment** — `align` prop: `align={{price: "right", id: "center"}}`
7. **Scrolling** — CSS concern, not component concern. A `scrollable` boolean prop that wraps the table in a `<div style="overflow-x: auto">` is sufficient.
8. **Pagination** — better as a separate `<Pagination>` component that composes alongside `DataTable`, since `Table` already has `.limit()` and `.offset()` for the data side. Building pagination into `DataTable` would conflate data preparation with presentation.
9. **Sorting** — the unused `sortable` prop should be removed for now. Client-side sorting is complex JS; server-side sorting is a `Table.orderBy()` concern, not a presentation concern. Revisit in 1.2.

**What this means for `Table.toHTML()`:** Once `DataTable` accepts a `Table` directly, `Table.toHTML()` becomes the quick-and-dirty option (CLI output, dev tools, debugging) while `<DataTable>` is the production-quality option (accessible, styled, extensible). Both have clear value; they stop overlapping.

#### Utility — Adequate ✅

`SrOnly`, `Abbr`, `A`/`Link`, `Icon` — all fine. `Icon` correctly uses `aria-hidden="true"` for decorative icons and provides an optional `label` for meaningful icons.

---

## 4. Modern Standards

### What's Good

- **`loading="lazy"`** on `Img` and `Iframe` — correct modern default
- **`decoding="async"`** on `Img` — good for performance
- **Custom elements** for time components (`<local-time>`, `<relative-time>`, `<time-range>`) — modern and progressive
- **`rel="noopener noreferrer"`** on external links — security best practice
- **Schema.org `BreadcrumbList`** structured data — SEO best practice
- **CSS `field-sizing: content`** detection in `basil.js` with polyfill fallback for textarea auto-resize

### What's Missing or Outdated

1. **No `fetchpriority` support** — `Img` should support `fetchpriority="high"` for hero/LCP images. This is a relatively new but well-supported attribute.

2. **No `<picture>` / responsive images component** — Modern web development uses `<picture>` with `<source>` for art direction and `srcset`/`sizes` on `<img>` for resolution switching. A `Picture` or `ResponsiveImg` component would be valuable.

3. **No `<dialog>` component** — The HTML `<dialog>` element is now well-supported (baseline 2022) and provides native modal behaviour with proper focus trapping, backdrop, and `Escape` key handling. A `Dialog` component would be a significant addition.

4. **No `popover` attribute support** — The Popover API (baseline 2024) provides native tooltip/popover behaviour without JavaScript. A `Popover` component or attribute support on `Button` would be forward-looking.

5. **No `<details>`/`<summary>` component** — Native disclosure widget, no JS needed. Very useful for FAQs, accordions, and collapsible sections. An `Accordion` or `Details` component would be a quick win.

6. **No `color-scheme` meta tag** — For dark mode support, `<meta name="color-scheme" content="light dark">` should be an option in `Page`.

7. **No `Content-Security-Policy` helpers** — While this is more of a server concern, the prelude could help by generating nonces for inline scripts/styles.

8. **`SkipLink` uses inline styles** — Should use a class with CSS in the bundle, not inline `<style>`. Inline styles conflict with strict Content Security Policies.

### Recommendations by Priority

| Item | Priority | Effort |
|------|----------|--------|
| `fetchpriority` on `Img` | High (1.0) | 5 min |
| Move `SkipLink` CSS to bundle | High (1.0) | 15 min |
| `Dialog` component | Medium (1.1) | 2-3 hours |
| `Details`/`Accordion` component | Medium (1.1) | 1 hour |
| `color-scheme` meta in `Page` | Low (1.1) | 10 min |
| `Picture`/responsive image component | Low (1.2) | 2 hours |
| Popover support | Low (1.2) | 2 hours |

---

## 5. Accessibility Standards

### What's Excellent

The prelude has genuinely good accessibility. This is one of its strongest selling points.

- **Form components**: Proper `<label>` + `for`/`id` association, `aria-required`, `aria-invalid`, `aria-describedby` linking hints and errors, `role="alert"` on error messages, `<fieldset>`/`<legend>` for groups.
- **Navigation**: `SkipLink` auto-included in `Page`, `aria-label` on `<nav>`, `aria-current="page"` on breadcrumb current item.
- **Images**: `alt` attribute always present (defaults to `""` for decorative), `title` required on `Iframe`.
- **Icons**: `aria-hidden="true"` with optional `sr-only` label.
- **Tables**: `scope="col"` and `scope="row"` on header cells, `<caption>` support.
- **Time**: `datetime` attribute for machine-readable dates, `title` for full UTC time.

### What's Missing or Could Be Better

1. **No `aria-live` region component** — Beyond `role="alert"` on errors, there's no general-purpose live region component for dynamic content updates. A `LiveRegion` or `Announce` component would be useful for SPA-like updates via Parts.

2. **No focus management utilities** — When a Part updates content, focus can be lost. A `FocusTrap` component (for modals/dialogs) and guidance on focus management after updates would help.

3. **Error summary component** — Best practice for form accessibility is to provide an error summary at the top of the form listing all errors with links to the relevant fields. This is standard in GOV.UK Design System and similar. A `FormErrors` or `ErrorSummary` component would be valuable.

4. **`required` indicator is visual-only** — The asterisk `*` is `aria-hidden="true"` (correct), but the `aria-required` attribute is the machine-readable equivalent (also present — good). However, the text " *" is visually appended after the label text, which means screen readers announce "Email" while sighted users see "Email *". Some accessibility guidelines recommend also having the word "required" visible or in an `sr-only` span. This is debatable — current implementation is acceptable but could be improved.

5. **`Breadcrumb` separator** — The separator `" / "` is inside the visible list but has `aria-hidden="true"` — correct. Good implementation.

6. **Skip link target** — `SkipLink` targets `#main` by default, and `Page` sets `id="main"` on `<body>`. This means the skip link jumps to the body, not to the main content. It should target a `<main>` element inside the body. This is a bug.

7. **No `role="status"` component** — For success messages, loading states, and other non-urgent updates, `role="status"` (equivalent to `aria-live="polite"`) is appropriate. Currently only `role="alert"` (equivalent to `aria-live="assertive"`) is used, which is too aggressive for non-error states.

8. **`Button` toggle accessibility** — The toggle button pattern (`data-toggle`) sets `aria-expanded` and `aria-controls`, which is correct. But it initialises `aria-expanded="false"` regardless of the target's actual state. If the target starts visible, this would be wrong.

### Recommendations

| Item | Priority | Effort |
|------|----------|--------|
| Fix skip link target (`<main>` not `<body>`) | 🔴 High (1.0) | 15 min |
| `ErrorSummary` component | Medium (1.1) | 2 hours |
| `LiveRegion` component | Low (1.1) | 1 hour |
| Fix `Button` toggle initial state | Low (1.0) | 15 min |

---

## 6. Common Usage

### What the Target Audience Needs

The target audience is SMB web developers, hobbyists, educators, and PHP/Rails refugees. Their most common needs:

| Need | Covered? | Component |
|------|----------|-----------|
| Page layout | ✅ | `Page` |
| Navigation | ✅ | `Nav`, `Breadcrumb` |
| Contact/login forms | ✅ | `Form`, `TextField`, `Button` |
| Data tables | ✅ | `DataTable` |
| Images | ✅ | `Img`, `Figure` |
| Embeds (YouTube, maps) | ✅ | `Iframe` |
| Links | ✅ | `A` |
| **File uploads** | ❌ | Missing |
| **Flash messages** | ❌ | Missing |
| **Pagination** | ❌ | Missing |
| **Modals/dialogs** | ❌ | Missing |
| **Accordions/FAQs** | ❌ | Missing |
| **Alerts/notifications** | ❌ | Missing |
| **Tabs** | ❌ | Missing |
| **Cards** | ❌ | Could be user-built easily |
| **Loading states** | ❌ | `Skeleton` was designed but not implemented |

### Gap Analysis: What Competitors Provide

| Component | Rails (ViewComponent) | Laravel (Blade) | Django | Basil |
|-----------|----------------------|-----------------|--------|-------|
| Form fields | ✅ | ✅ | ✅ | ✅ |
| Flash/Toast | ✅ | ✅ | ✅ | ❌ |
| Pagination | ✅ | ✅ | ✅ | ❌ |
| File upload | ✅ (Active Storage) | ✅ (Livewire) | ✅ | ❌ |
| Modal/Dialog | ❌ (gem) | ❌ (Alpine) | ❌ | ❌ |
| Data table | ❌ (gem) | ❌ (Livewire) | ✅ (admin) | ✅ |
| Breadcrumb | ❌ (gem) | ❌ (package) | ❌ | ✅ |
| Accordion | ❌ | ❌ | ❌ | ❌ |

Basil is actually ahead of Rails/Laravel/Django in some areas (breadcrumbs with Schema.org, accessible form components, time localisation). But the missing Pagination and Flash/Toast components are glaring — every CRUD app needs them.

### Recommended Additions for 1.0 or 1.1

| Component | Priority | Rationale |
|-----------|----------|-----------|
| `Pagination` | 🔴 High (1.0 or 1.1) | Every list page needs it |
| `Toast` / `Flash` | 🔴 High (1.0 or 1.1) | Every form submission needs feedback |
| `FileField` | 🟡 Medium (1.1) | Common for SMB sites (logos, documents) |
| `Dialog` | 🟡 Medium (1.1) | Uses native `<dialog>`, no heavy JS |
| `Details` / `Accordion` | 🟡 Medium (1.1) | Uses native `<details>`, zero JS |
| `ErrorSummary` | 🟡 Medium (1.1) | Accessibility best practice for forms |
| `Tabs` | 🟢 Low (1.2) | Requires JS, less common in server-rendered apps |

---

## 7. Flexibility, Customisation, and Enhancibility

This is where the prelude has the most room for improvement. Several design patterns need to be established and applied consistently.

### Issue 1: Prop Spreading Inconsistency

Some components use the `...rest` spreading pattern, others enumerate every prop:

| Pattern | Components |
|---------|-----------|
| **Spreads rest props** ✅ | `TextField`, `TextareaField`, `SelectField`, `RadioGroup`, `CheckboxGroup`, `Button`, `Form`, `A`, `Img` |
| **Enumerates all props** ❌ | `Iframe`, `Figure`, `Blockquote`, `Nav`, `Abbr`, `SrOnly`, `SkipLink`, `DataTable`, `Time`, `LocalTime`, `TimeRange`, `RelativeTime`, `Icon`, `Checkbox`, `Page`, `Head` |

Components that enumerate props cannot receive arbitrary HTML attributes. This means a user cannot add `data-*` attributes, `style`, `role`, or any attribute the component author didn't anticipate.

**Recommendation:** All components should use the `...rest` spreading pattern. This is a mechanical change — destructure known props, spread the rest onto the primary HTML element.

### Issue 2: Class Merging

The current pattern for class merging is:

```parsley
class={"field" ++ if (class) { " " ++ class } else { "" }}
```

This has two problems. First, it uses `++` which in Parsley creates an **array**, not a concatenated string — `"field" ++ " my-class"` produces `["field", " my-class"]`, not `"field my-class"`. It works by accident (the array is coerced to a string when rendered as an HTML attribute), but the intent is string concatenation, which should use `+`. Second, it's verbose and repeated identically in every component.

A shorter, correct idiom already exists using `++` deliberately as array construction:

```parsley
class={("field" ++ class).join(" ")}
```

This builds an array `["field", class]` and joins with spaces. Since `join()` skips null and empty string elements (**BUG-024**, now fixed), this produces `"field"` when `class` is null and `"field my-custom"` when `class` is `"my-custom"` — exactly the right behaviour with no conditional logic.

This is already clean enough that `cx()` becomes a nice-to-have rather than essential. Two improvements:

1. **Helper function**: Create a `cx()` or `classes()` utility that merges class names, filtering nulls:

   ```parsley
   // Instead of the current (buggy) pattern:
   class={"field" ++ if (class) { " " ++ class } else { "" }}
   // Or the correct and concise:
   class={("field" ++ class).join(" ")}
   // Or with a helper for multi-class cases:
   class={cx("field", class)}
   ```

2. **Convention**: Document that the component's base class is always the first class. Users can then target it with CSS: `.field { ... }` for all fields, `.field.my-custom { ... }` for specific ones.

### Issue 3: `Page` Extensibility

`Page` is the most important component and the most difficult to extend. Current pain points:

**Adding meta tags, fonts, scripts, CSS:**
The `head` prop accepts arbitrary content, but:
- There's no way to control ordering relative to `<CSS/>` and other built-in head content
- There's no `beforeCSS` or `afterCSS` slot
- Multiple `head` contributions from nested components can't be aggregated

**Adding body attributes:**
No way to add `data-theme`, `dir`, `hx-boost`, or other attributes to `<body>` or `<html>`.

**Adding scripts at the end:**
No way to add scripts after `<Javascript/>` and `<BasilJS/>`.

**Proposed `Page` interface:**

```parsley
<Page 
    lang="en" 
    dir="ltr"
    title="My Site"
    description="About things"
    
    // Head content slots (ordered)
    headStart={<link rel="preconnect" href="https://fonts.googleapis.com"/>}
    head={<link rel="stylesheet" href="https://fonts.googleapis.com/css?family=Roboto"/>}
    
    // Body attributes
    bodyClass="theme-dark"
    bodyAttrs={{data-theme: "dark"}}
    
    // Content slots  
    beforeContent={<MyNav/>}
    afterContent={<MyFooter/>}
    
    // Script slots
    scripts={<script src="/custom.js"/>}
    
    // Feature flags
    noBasilJS={false}
    colorScheme="light dark"
>
    <main id="content">
        "Page content"
    </main>
</Page>
```

This is more complex than the current interface, but every prop is optional — the simple case remains simple:

```parsley
<Page title="Hello">
    "content"
</Page>
```

### Issue 4: `Head` / `Page` Relationship

> **DECIDED:** Option A — Rename `Head` → `Meta`, strip the `<head>` wrapper, use inside `Page`'s `head` prop.

As noted in §3, `Head` and `Page` both generate `<head>` content, and their relationship is confusing. `Head` generates its own `<head>` tag, so it can't be nested inside `Page`. Two options were considered:

**Option A — Separate components (CHOSEN):** Rename `Head` → `Meta`, strip the `<head>` wrapper so it just outputs meta/link tags, use inside `Page`'s `head` prop.

**Option B — Merge into `Page`:** Absorb `Head`'s props directly into `Page`, eliminating `Head` entirely.

**Decision rationale:**

1. **Separation of concerns is cleaner.** `Page` is about document structure (html/head/body, skip links, asset bundles). `Meta` is about SEO/social metadata (Open Graph, Twitter Cards, favicons). These are conceptually distinct responsibilities.

2. **Composability wins.** With Option A, users who don't need social metadata get a lean `Page`. Users who need it compose `<Meta/>` inside the `head` prop. Option B forces every `Page` to accept 10+ props that most pages won't use.

3. **Prop count matters for DX.** A `Page` component with 15-20 props is overwhelming. Keeping `Page` focused (~7 props) and having `Meta` (~12 props) as opt-in keeps both manageable.

4. **Real-world usage patterns.** Most pages in a site don't need custom OG images/article metadata — those are for blog posts and landing pages. Having a separate `Meta` component lets those special pages opt in without cluttering the common case.

**Implementation:**

```parsley
// Simple page (most common)
<Page lang="en" title="About Us" description="Learn about our company">
    "content"
</Page>

// Blog post with full social metadata
<Page lang="en" title="My Post" head={
    <Meta 
        description="A deep dive into..."
        image="/og/my-post.png"
        url="https://example.com/posts/my-post"
        type="article"
        published={post.date}
        twitter="@handle"
    />
}>
    post.content
</Page>
```

**Changes required:**
- Rename `head.pars` component from `Head` to `Meta`
- Remove the `<head>...</head>` wrapper — output only the meta/link tags
- Keep `<CSS/>` call inside `Page`, not `Meta`
- Update documentation and examples

### Issue 5: DataTable Customisation

`DataTable` accepts `columns` (display names), `rows` (data), and `keys` (which fields to show). But there's no way to:

- Format cell values (e.g., dates, currency)
- Add links to cells
- Add action buttons to rows
- Custom-render a cell

**Proposed enhancement — render callback:**

```parsley
<DataTable 
    columns={["Name", "Email", "Actions"]}
    rows={users}
    keys={["name", "email"]}
    render={{
        email: fn(value, row) { <A href={"mailto:" ++ value}>{value}</A> },
        _actions: fn(_, row) { 
            <A href={"/users/" ++ row.id ++ "/edit"}>"Edit"</A>
        }
    }}
/>
```

The `render` prop is a dictionary mapping column keys to render functions. A special `_actions` key (or any key not in `keys`) adds a virtual column. This follows React-table and similar patterns but stays within Parsley's functional style.

---

## 8. Internationalisation and Localisation

### Current State

The prelude's i18n story is:

| Feature | Status | Mechanism |
|---------|--------|-----------|
| Time/date localisation | ✅ Good | `<local-time>`, `<relative-time>` custom elements use `Intl.DateTimeFormat` |
| Time range localisation | ✅ Good | `<time-range>` custom element uses `Intl.DateTimeFormat` |
| Number formatting | ❌ Missing | No component |
| Currency formatting | ❌ Missing | No component (though Parsley has Money type) |
| Text direction (RTL) | ❌ Missing | No `dir` attribute support on `Page` |
| Language attribute | ⚠️ Partial | `Page` has `lang` prop, defaults to `"en"` |
| String translation | ❌ Missing | No i18n system |
| Locale-aware sorting | ❌ Missing | — |

### What the Browser Can Do for Us

The key insight is that the browser knows the user's locale via `navigator.language` (and the `Accept-Language` header). Basil can leverage this in two ways:

1. **Client-side (custom elements):** Already done for time components. The same pattern works for numbers and currency.
2. **Server-side (Accept-Language header):** Basil has access to the request headers and could parse `Accept-Language` to provide server-rendered localised content.

### Proposed Components

#### `<LocalNumber>` — Client-side number formatting

```parsley
<LocalNumber value={1234567.89}/>
// Server fallback: "1234567.89"
// Client renders (en-US): "1,234,567.89"
// Client renders (de-DE): "1.234.567,89"

<LocalNumber value={0.15} style="percent"/>
// Server fallback: "15%"
// Client renders (en-US): "15%"
// Client renders (fr-FR): "15 %"
```

**JavaScript:** ~20 lines using `Intl.NumberFormat`.

#### `<LocalCurrency>` — Client-side currency formatting

```parsley
<LocalCurrency value={#49.99GBP}/>
// Server fallback: "£49.99" (from money.medium())
// Client renders (en-US): "£49.99" or "GBP 49.99"
// Client renders (de-DE): "49,99 £"

<LocalCurrency value={price} currency="USD"/>
// For non-Money values, specify currency explicitly
```

**JavaScript:** ~20 lines using `Intl.NumberFormat` with `style: "currency"`.

**Integration with Parsley `money` type:** The component should type-check its `value` prop. When it receives a `money` value (which already carries `.amount`, `.currency`, and `.scale`):
- Extract the currency code automatically — no separate `currency` prop needed
- Use `value.medium()` for the server-rendered fallback (→ "£49.99"), giving a good default before JS enhances
- Pass `value.amount`, `value.currency`, and `value.scale` as `data-` attributes on the custom element so the client-side JS can re-render with `Intl.NumberFormat` using the user's locale

When `value` is a plain number, the `currency` prop is required (current design). This doesn't restrict — plain numbers still work — but rewards using the `money` type with better ergonomics and a formatted server fallback instead of a bare number.

The `money` type also has `.short()` (→ "£5K"), `.long()` (→ "£4,999.00"), and `.full()` (→ "four thousand nine hundred and ninety-nine pounds") which the component could expose via a `style` prop for different display contexts.

#### `<LocalDate>` — Alias for `<LocalTime>` with date-only defaults

```parsley
<LocalDate value={post.publishedAt}/>
// Equivalent to <LocalTime datetime={post.publishedAt} format="date"/>
```

This is purely ergonomic — `<LocalTime format="date">` works but `<LocalDate>` is more discoverable.

Like the time components (see §13 note on `datetime` type awareness), `<LocalDate>` should accept a Parsley `datetime` value directly — extracting `.iso` for the attribute and using `.medium()` for the server-rendered fallback — as well as plain date strings.

### RTL / Direction Support

`Page` should accept a `dir` prop:

```parsley
<Page lang="ar" dir="rtl" title="موقعي">
    "content"
</Page>
```

This is a one-line addition to the `<html>` tag output.

### Accept-Language on the Server

Basil could make the browser's preferred language available via `request.locale` or `request.language`:

```parsley
let locale = request.language ?? "en"
<Page lang={locale}>
    // Server-rendered content can use locale for formatting
    price.format({locale: locale})
</Page>
```

This is more of a Basil server feature than a prelude feature, but it would enable server-side localisation without JavaScript.

### What We Should NOT Do for 1.0

- Build a full translation/i18n framework (e.g., message catalogs, plural rules). This is a significant feature that should be a post-1.0 effort.
- Provide locale-aware sorting (complex, Unicode-dependent).
- Attempt to translate UI labels or error messages — keep them in English for 1.0, document how users can customise.

### Recommendations

| Item | Priority | Effort |
|------|----------|--------|
| `dir` prop on `Page` | 🔴 High (1.0) | 5 min |
| `LocalNumber` component + JS | 🟡 Medium (1.1) | 1 hour |
| `LocalCurrency` component + JS | 🟡 Medium (1.1) | 1 hour |
| `LocalDate` alias | 🟢 Low (1.1) | 15 min |
| `request.language` server feature | 🟡 Medium (1.1) | 2 hours |

---

## 9. Fit With New Parsley Features

### The Two Form Systems Problem

Basil now has two parallel ways to build forms:

**System 1: Prelude Components (Parsley-level)**

```parsley
<Form action="/users" method="POST">
    <TextField name="email" label="Email" type="email" required={true} value={form.email} error={errors.email}/>
    <TextField name="name" label="Name" value={form.name} error={errors.name}/>
    <Button type="submit">"Save"</Button>
</Form>
```

**System 2: `@record`/`@field` Binding (evaluator-level)**

```parsley
@schema User {
    name: string | {placeholder: "Your name"} | required
    email: email | {placeholder: "you@example.com"} | required
}

let user = User({name: form.name, email: form.email})

<form @record={user} method="POST" action="/users">
    <label @field="name"/>
    <input @field="name"/>
    <error @field="name"/>
    
    <label @field="email"/>
    <input @field="email"/>
    <error @field="email"/>
    
    <button type="submit">"Save"</button>
</form>
```

These systems don't interact. The `@field` system generates attributes at the evaluator level (in Go code: `form_binding.go`, `form_components.go`, `form_autocomplete.go`), while prelude components are Parsley functions that know nothing about `@record` or `@field`.

### The Opportunity

The prelude components should be able to accept a record and field name and delegate to the `@field` system's intelligence:

```parsley
// Ideal: prelude components + schema power
@schema User {
    name: string | {placeholder: "Your name"} | required
    email: email | {placeholder: "you@example.com"} | required  
}

let user = User({name: form.name, email: form.email})

<Form action="/users" record={user}>
    <TextField field="name"/>
    <TextField field="email"/>
    <Button type="submit">"Save"</Button>
</Form>
```

In this model:
- `<Form record={user}>` sets the form context (like `<form @record={user}>`)
- `<TextField field="name"/>` reads from the record's schema to determine: label (from field name), type (from schema type), required (from constraints), placeholder (from metadata), autocomplete (from `form_autocomplete.go` logic), value (from record data), error (from record validation errors)
- The developer can still override any of these: `<TextField field="name" label="Full Name"/>`

### What's Needed

> **DECIDED:** Implement `record.fieldProps(name, options?)` — a hybrid approach with hardwired type mappings plus metadata overrides.

This requires a bridge between the prelude component system and the form binding system. Three options were considered:

**Option A: Expose form binding helpers as builtins** — Add hidden builtins like `__formFieldAttrs(record, "email")` that prelude components call internally.

**Option B: Make `@field` work on prelude components** — Extend the `@field` system to recognise prelude component tags.

**Option C: Record method that returns form metadata** — Add `record.fieldProps(name)` that returns a props dictionary.

**Decision: Hybrid Option C with type-aware defaults**

Option B was ruled out (mixes evaluator-level and component-level concerns). Between A and C, Option C was chosen because:

1. **Keep intelligence in Parsley, not hidden Go builtins.** A method on `record` is easier to test, document, and debug. Users can inspect what `.fieldProps()` returns; they can't see what hidden builtins do.

2. **Transparency and composability.** Users see exactly what's being passed and can override individual props:
   ```parsley
   let props = user.fieldProps("email")
   log("Field props:", props)  // Debuggable
   <TextField ...props label="Work Email"/>  // Override label
   ```

3. **Prelude components stay simple.** `TextField` just takes props — it doesn't need to know about records. The intelligence is at the call site.

4. **Builds on existing infrastructure.** The `record` type already has `.title()`, `.placeholder()`, `.error()`, `.hasError()`, `.format()`, `.enumValues()`. Adding `.fieldProps()` is a thin wrapper, not new infrastructure.

**The hybrid approach: hardwired type mappings + metadata overrides**

The `fieldProps()` method combines:
- **Hardwired defaults for known schema types** (input type, autocomplete, inputmode, pattern)
- **Metadata overrides** for customisation
- **Automatic assembly** of label, placeholder, value, error, required from existing methods

**Hardwired type mappings (implemented in Go):**

| Schema Type | HTML `type` | `inputmode` | `autocomplete` | Notes |
|-------------|-------------|-------------|----------------|-------|
| `string` | `text` | — | — | |
| `email` | `email` | `email` | `email` | |
| `url` | `url` | `url` | `url` | |
| `phone` | `tel` | `tel` | `tel` | |
| `integer` | `number` | `numeric` | — | |
| `number`/`float` | `text` | `decimal` | — | `type="number"` has UX issues |
| `boolean` | `checkbox` | — | — | |
| `money` | `text` | `decimal` | — | Needs custom parsing |
| `date` | `date` | — | — | Native date picker |
| `datetime` | `datetime-local` | — | — | Native datetime picker |
| `unit` | `text` | `decimal` | — | Needs custom parsing |
| `enum` | — | — | — | Returns `type: "select"` + `options` |

**Metadata-driven customisation:**

For fields where the hardwired default isn't right, schema metadata overrides:

```parsley
@schema User {
    // Override input type via metadata
    age: integer | {inputType: "text", inputmode: "numeric"}
    
    // Custom autocomplete
    street: string | {autocomplete: "street-address"}
    
    // Pattern for validation
    postcode: string | {pattern: "[A-Z]{1,2}[0-9][0-9A-Z]?\\s?[0-9][A-Z]{2}"}
}
```

**Example usage:**

```parsley
@schema Product {
    name: string | {title: "Product Name", placeholder: "Enter name..."}
    price: money | {currency: "GBP", title: "Price"}
    status: enum("draft", "active", "archived") | {title: "Status"}
    email: email | {title: "Contact Email"}
}

let product = Product({name: "Widget", price: 4999, status: "draft", email: "sam@example.com"})

product.fieldProps("name")
// → {name: "name", type: "text", label: "Product Name", placeholder: "Enter name...", value: "Widget", required: true}

product.fieldProps("price")
// → {name: "price", type: "text", inputmode: "decimal", label: "Price", value: "49.99", required: true}

product.fieldProps("status")
// → {name: "status", type: "select", label: "Status", value: "draft", options: ["draft", "active", "archived"], required: true}

product.fieldProps("email")
// → {name: "email", type: "email", autocomplete: "email", label: "Contact Email", value: "sam@example.com", required: true}

// Usage with prelude components
<TextField ...product.fieldProps("name")/>
<TextField ...product.fieldProps("email") class="wide"/>
<SelectField ...product.fieldProps("status")/>
```

**Implementation notes:**
- Implement as a method on the `Record` type in Go (`methods_record.go`)
- Call existing methods (`.title()`, `.placeholder()`, `.error()`, `.enumValues()`, etc.) internally
- Support optional second argument for call-site overrides: `product.fieldProps("name", {class: "large"})`
- Return a dictionary that can be spread into prelude components

**Supported metadata keys for `fieldProps()` override:**

| Metadata key | Purpose | Example |
|--------------|---------|---------|
| `title` | Label text | `{title: "Full Name"}` |
| `placeholder` | Input placeholder | `{placeholder: "Enter..."}` |
| `inputType` | Override HTML input type | `{inputType: "text"}` |
| `inputmode` | Override inputmode attribute | `{inputmode: "numeric"}` |
| `autocomplete` | Override autocomplete hint | `{autocomplete: "street-address"}` |
| `pattern` | HTML pattern attribute | `{pattern: "[0-9]+"}` |
| `min`, `max`, `step` | Numeric constraints | `{min: 0, max: 100}` |
| `format` | Display format hint (for `.format()`) | `{format: "currency"}` |
| `currency` | Currency code for money fields | `{currency: "GBP"}` |

### Table ↔ DataTable

See §3 (Data — Needs Rethinking) for the full analysis. In short: `DataTable`'s API predates the `Table` type and ignores it. The user is forced to decompose a `Table` into parallel `columns`/`rows`/`keys` arrays that the `Table` already carries. Meanwhile `Table.toHTML()` renders a bare `<table>` without the accessibility attributes (`scope`, `<caption>`) that `DataTable` provides.

The right split is: `Table` handles data preparation (`.where()`, `.orderBy()`, `.limit()`, `.offset()`); `DataTable` handles accessible, styled presentation. `DataTable` should accept a `Table` directly via a `data` prop and derive columns/rows automatically, with the current manual props kept as a fallback for raw arrays. `Table.toHTML()` remains the quick-and-dirty option for CLI/debugging output.

The most valuable enhancements — in priority order — are: accept `Table` directly, empty state, cell formatting based on value/schema types, custom cell render functions, footer/summary rows, and column alignment. Sorting and pagination should remain separate concerns (`Table.orderBy()` and a standalone `<Pagination>` component respectively).

### Schema Validation ↔ Form Errors

The `@schema` validation system produces structured errors. The prelude form components accept `error` as a string prop. These should integrate:

```parsley
let user = User(form)
let errors = user.validate()

// Current: manual error extraction
<TextField name="email" label="Email" error={errors.email?.message}/>

// Ideal: pass record, component extracts errors automatically
<TextField field="email" record={user}/>
```

### Built-in `.toHTML()` Methods vs Prelude Components

Four built-in types have `.toHTML()` methods: `table`, `array`, `dictionary`, and `markdown`. These overlap with and complement the prelude in different ways. The key question is: do they work well together, and should the prelude's lessons feed back into the built-in methods?

#### Current State

| Type | `.toHTML()` produces | Accessibility | Type-aware formatting | Prelude equivalent |
|------|---------------------|---------------|----------------------|-------------------|
| `table` | `<table>` with `<thead>/<tbody>/<tfoot>` | ❌ No `scope`, no `<caption>` | ❌ Uses `objectToString()` — raw values | `<DataTable/>` |
| `array` | `<ul>` or `<ol>` with `<li>` | ❌ No ARIA | ❌ Uses `objectToPrintString()` — raw values | None |
| `dictionary` | `<dl>/<dt>/<dd>` (or `<table>` via `table: true` option) | ❌ No ARIA | ❌ Uses `objectToPrintString()` — raw values | None |
| `markdown` | Full HTML from markdown AST | N/A (content-dependent) | ✅ Full markdown rendering | None (different purpose) |

Meanwhile, types with rich formatting intelligence have **no** `.toHTML()` at all:

| Type | Has `.toHTML()`? | Formatting methods available |
|------|-----------------|------------------------------|
| `money` | ❌ | `.medium()` → "£4,999.00", `.short()` → "£5K", `.long()`, `.full()` |
| `datetime` | ❌ | `.medium()` → "Mar 15, 2025", `.short()` → "3/15/25", `.long()`, `.full()` |
| `unit` | ❌ | `.medium()` → "5.00kg", `.short()` → "5kg", `.long()` → "5.00 kilograms", `.full()` → "5.00 kilograms (11.0 lb)" |
| `record` | ❌ | `.format(field)` — schema-aware formatting for date, currency, percent, number |
| `duration` | ❌ | `.medium()` → "2 hours", `.short()` → "2h", `.long()` → "2 hours 30 minutes" |

#### The Formatting Gap in `.toHTML()`

> **DECIDED:** Change both `objectToString()` and `objectToPrintString()` to call `.medium()` on typed values. This is a breaking change acceptable before 1.0.

The most significant finding: **none of the `.toHTML()` methods use the formatting intelligence of typed cell values.** When a `money` value appears in a table cell, `table.toHTML()` calls `objectToString()`, which falls through to `Money.Inspect()` → "£4999.00" (no thousands separator, no locale awareness). Compare:

| Context | `money(499900, "GBP")` renders as |
|---------|-----------------------------------|
| `table.toHTML()` cell | "£4999.00" (raw `Inspect()`) |
| Parsley tag content `<td>(price)</td>` | "£4999.00" (same path) |
| `price.medium()` | "£ 4,999.00" (properly formatted) |
| Prelude `<DataTable/>` cell | "£4999.00" (same raw path — no type awareness yet) |

The same applies to `datetime` values in table cells: they render as "2025-03-15" (ISO date string) rather than "Mar 15, 2025" (`.medium()`), and `unit` values render as "5kg" (`.short()` equivalent from `Inspect()`) rather than the arguably more appropriate "5.00kg" (`.medium()`).

**Decision: Change the default formatting for typed values**

The goal is "Parsley templates should produce human-readable output by default with zero effort." Since we're pre-1.0, backward compatibility is not a concern — break and fix.

**Changes to implement:**

| Function | Current behaviour | New behaviour |
|----------|-------------------|---------------|
| `objectToString()` | Falls through to `Inspect()` | Calls `.medium()` for `money`, `datetime`, `unit`, `duration` |
| `objectToPrintString()` | Falls through to `Inspect()` | Calls `.medium()` for `money`, `datetime`, `unit`, `duration` |

**Impact:**

| Type | Before | After |
|------|--------|-------|
| `money(499900, "GBP")` | "£4999.00" | "£4,999.00" |
| `date("2025-03-15")` | "2025-03-15" | "Mar 15, 2025" |
| `unit(5, "kg")` | "5kg" | "5.00kg" |
| `duration` | varies | "2 hours 30 minutes" |

This affects:
- Tag content: `<td>(price)</td>` → now renders "£4,999.00"
- `.toHTML()` methods: table cells, array items, dict values
- String coercion contexts

**For users who need ISO/raw format:**
- Use `.iso` property for datetimes: `<time datetime={post.date.iso}>`
- Use `.inspect()` or specific format methods for other types

#### Do They Complement Each Other?

**Yes, but with a clear division of labour:**

- **`.toHTML()` methods** are quick-and-dirty: useful for debugging, CLI output, REPL exploration, and one-off rendering where you want *something* without thinking about presentation. They produce valid HTML but not *good* HTML.
- **Prelude components** are production-quality: accessible, styled, extensible, and designed for real web pages.

This division is correct and should be preserved. The two should **not** converge — making `table.toHTML()` produce the same output as `<DataTable/>` would make the method too complex and opinionated for its purpose as a quick utility.

#### What the Prelude Teaches Us About `.toHTML()`

The prelude's quality highlights specific gaps in the built-in methods that are worth addressing without changing their purpose:

**1. `table.toHTML()` should use typed formatting for cell values**

This is the lowest-hanging fruit. When `objectToString()` encounters a `Money`, `datetime`, `duration`, or `unit` value, it should call `.medium()` (or equivalent) instead of falling through to `Inspect()`. This doesn't change the structure of the output — it just makes cell values readable:

```
// Before: table.toHTML() renders money as "£4999.00"
// After:  table.toHTML() renders money as "£4,999.00"
// Before: table.toHTML() renders datetime as "2025-03-15"  
// After:  table.toHTML() renders datetime as "Mar 15, 2025"
```

This is a change to `objectToString()` in `stdlib_table.go` (and possibly `objectToPrintString()` in `eval_string_conversions.go`), not to `table.toHTML()` itself. It would benefit all `.toHTML()` methods and Parsley tag content rendering (`<td>(price)</td>`) simultaneously.

**2. `table.toHTML()` should add minimal accessibility attributes**

Adding `scope="col"` to `<th>` elements in `<thead>` is a one-line change that brings the output closer to accessibility standards without adding complexity. A `<caption>` would require an API change (optional argument), which is a lighter lift than a full `<DataTable/>` redesign but still worth considering.

**3. `dict.toHTML({table: true})` has a bug**

The option parsing code does `tableExpr.(Object)` instead of `Eval(tableExpr, env)`, so the `table: true` option is silently ignored — the output is always `<dl>`. This should be fixed independently of the prelude review.

**4. `record` should consider a `.toHTML()` method**

A `record` has schema metadata (field titles, types, format hints) that would make its `.toHTML()` output significantly richer than `dictionary.toHTML()`. It could render as a `<dl>` where `<dt>` uses `schema.title(field)` instead of the raw field name, and `<dd>` uses `record.format(field)` for type-aware value display. This would be a natural complement to the form-focused prelude components — forms are for *input*, `.toHTML()` is for *display*:

```parsley
let user = User({name: "Sam", email: "sam@example.com", role: "admin"})

// Quick display (debugging, admin panels, detail views):
user.toHTML()
// → <dl><dt>Name</dt><dd>Sam</dd><dt>Email</dt><dd>sam@example.com</dd><dt>Role</dt><dd>admin</dd></dl>

// Production display: use prelude components or custom Parsley templates
```

#### Recommendations

| Item | Priority | Effort | Impact | Status |
|------|----------|--------|--------|--------|
| Fix `objectToString`/`objectToPrintString` to call `.medium()` on `money`, `datetime`, `unit`, `duration` | 🔴 High (1.0) | 30 min | All `.toHTML()` methods and tag content rendering benefit simultaneously | **DECIDED** |
| Fix `dict.toHTML({table: true})` option parsing bug | 🟡 Medium (1.0) | 10 min | Restores documented functionality | |
| Add `scope="col"` to `table.toHTML()` `<th>` elements | 🟡 Medium (1.1) | 5 min | Minimal accessibility improvement | |
| Add `record.toHTML()` with schema-aware field titles and formatted values | 🟢 Low (1.1) | 1-2 hours | Useful for quick data display | |

#### Time Component Pre-existing Bugs

During this analysis, two pre-existing bugs were found in the time prelude components:

1. **`LocalTime`, `TimeRange`, `RelativeTime`** call `.format("iso")` on datetime values, but `datetime.format()` only accepts `short`, `medium`, `long`, `full` — not `"iso"`. The correct approach is to use the `.iso` property: `datetime.iso`.
2. **`RelativeTime`** calls `.relative()` on datetime values, but no `.relative()` method exists on the `datetime` type.

These components will error at runtime when passed `datetime` objects. This should be tracked as a separate bug fix.

---

## 10. JavaScript Enhancements

### Current `basil.js` Capabilities (~290 lines)

| Feature | Lines | Mechanism |
|---------|-------|-----------|
| Form confirm dialog | ~3 | `data-confirm` attribute |
| Auto-submit on change | ~2 | `data-autosubmit` attribute |
| Character counter | ~6 | `data-counter` attribute |
| Toggle visibility | ~8 | `data-toggle` attribute |
| Copy to clipboard | ~12 | `data-copy` attribute |
| Submit button disable | ~3 | All forms |
| Textarea auto-resize | ~5 | `data-autoresize` + CSS fallback |
| Focus first error | ~2 | `aria-invalid` detection |
| `<local-time>` | ~30 | Custom element + `Intl.DateTimeFormat` |
| `<time-range>` | ~40 | Custom element + `Intl.DateTimeFormat` |
| `<relative-time>` | ~80 | Custom element + `Intl.RelativeTimeFormat` |

**Total:** ~290 lines, estimated ~5KB minified. This is lean and well-designed.

### Proposed Additions

#### Toast/Flash Dismissal and Animation (~50 lines)

```javascript
// Auto-dismiss toasts after duration
// Pause on hover
// Manual dismiss
// CSS animation for enter/exit
class ToastElement extends HTMLElement {
    connectedCallback() { /* ... */ }
}
```

This pairs with the proposed `Toast` component.

#### Client-side Form Validation Feedback (~40 lines)

```javascript
// Show/hide error messages on blur (client-side pre-validation)
// Pair with server-side validation, don't replace it
document.querySelectorAll('[data-validate]').forEach(field => {
    field.addEventListener('blur', () => {
        if (!field.checkValidity()) {
            // Show browser-native validation message alongside our error element
        }
    })
})
```

This leverages the browser's built-in constraint validation API (`checkValidity()`, `validationMessage`) to give immediate feedback while the prelude's `error` prop handles server-side validation.

#### `<local-number>` and `<local-currency>` Custom Elements (~40 lines)

```javascript
class LocalNumberElement extends HTMLElement {
    connectedCallback() {
        const value = parseFloat(this.getAttribute('value'))
        if (isNaN(value)) return
        const style = this.getAttribute('style') || 'decimal'
        this.textContent = new Intl.NumberFormat(navigator.language, { style }).format(value)
    }
}
```

#### Dialog Auto-Focus and Close (~20 lines)

```javascript
// For <dialog> elements: auto-focus first input, close on Escape, close on backdrop click
document.querySelectorAll('dialog[data-auto]').forEach(dialog => {
    dialog.addEventListener('click', e => {
        if (e.target === dialog) dialog.close()
    })
})
```

#### Accordion `<details>` Exclusive Mode (~15 lines)

```javascript
// Only one <details> in a group open at a time (accordion behavior)
document.querySelectorAll('[data-accordion]').forEach(group => {
    group.querySelectorAll('details').forEach(detail => {
        detail.addEventListener('toggle', () => {
            if (detail.open) {
                group.querySelectorAll('details[open]').forEach(other => {
                    if (other !== detail) other.removeAttribute('open')
                })
            }
        })
    })
})
```

### Bundle Size Projection

| Feature | Est. Lines | Est. Minified |
|---------|-----------|---------------|
| Current `basil.js` | ~290 | ~5KB |
| Toast dismissal | ~50 | ~1KB |
| Form validation | ~40 | ~0.8KB |
| Number/currency elements | ~40 | ~0.8KB |
| Dialog helpers | ~20 | ~0.4KB |
| Accordion exclusive | ~15 | ~0.3KB |
| **Total** | **~455** | **~8.3KB** |

Well under the 10KB target mentioned in the server-enhanced-components design doc.

### What Should NOT Be in `basil.js`

- Full form validation library (use browser's constraint validation API)
- Animation library (use CSS transitions/animations)
- AJAX/fetch library (use Parts system)
- State management (server is the source of truth)
- SortableJS or other large dependencies (document as "bring your own")

---

## 11. Parsley Language Enhancements

### Class Merging Utility

As mentioned in §7, class merging is repeated verbatim in every component:

```parsley
class={"field" ++ if (class) { " " ++ class } else { "" }}
```

This pattern has a latent bug: `++` on strings creates an **array**, not a concatenated string. It happens to work because the array is coerced when rendered as an attribute value, but the intent is string concatenation (`+`). All 12 components that use this pattern should be fixed.

Additionally, `array.join()` does not skip null elements — it converts them to `""` and still inserts the separator, producing trailing spaces and double spaces. This is tracked as **BUG-024**. Once that's fixed, the correct idiomatic Parsley becomes simply:

```parsley
class={("field" ++ class).join(" ")}
// ++ builds array ["field", class], join skips null, produces "field my-class" or "field"
```

A `cx()` built-in would still be slightly more ergonomic for multi-class cases:

```parsley
class={cx("field", class, if (error) { "field-error" })}
// cx() joins non-null, non-empty strings with spaces
```

`cx()` (from the React ecosystem's `classnames` / `clsx` pattern) is simple enough to be a builtin or a prelude utility function.

### Slot/Children Pattern

Parsley's `contents` prop works well for single-slot components:

```parsley
<Card>"content here"</Card>
// contents = "content here"
```

But multi-slot components (like `Page` with `head`, `beforeContent`, `afterContent`) rely on passing rendered content as props, which is less ergonomic:

```parsley
<Page title="Test" head={<meta name="theme-color" content="#000"/>}>
    "content"
</Page>
```

This works but can get unwieldy with complex head content. A named-slot syntax would help, though this is a significant language addition and probably out of scope for 1.0:

```parsley
// Hypothetical future syntax (NOT a 1.0 proposal):
<Page title="Test">
    <slot:head>
        <meta name="theme-color" content="#000"/>
        <link rel="preconnect" href="https://fonts.googleapis.com"/>
    </slot:head>
    "main content"
</Page>
```

For now, the `head` prop approach is sufficient.

### Spread in Component Tags

The `...rest` prop spreading already works:

```parsley
export TextField = fn(props) {
    let {name, label, ...inputAttrs} = props
    <input ...inputAttrs/>
}
```

This is good. Ensure it's well-documented and consistently used across all components.

### Conditional Attribute Rendering

**Verified:** Conditional attributes must use the `if (x) "value" else null` pattern:

```parsley
aria-required={if (required) "true" else null}
```

This is the canonical Parsley idiom. **Neither `&&` nor `??` provides a shorter alternative:**

- `required && "true"` → returns `true` or `false` (the boolean), not the string — Parsley's `&&` does not short-circuit to return the right-hand value like JavaScript
- `required ?? "true"` → only checks for `null`, so `false ?? "true"` returns `false`, not `"true"`

The `if/else` pattern is correct and should be used consistently across all components.

---

## 12. Previously Discussed Components

### SortableList (Drag-and-Drop)

**Status:** Fully designed in two documents:
- `work/design/sortable-list-component.md` — Detailed component design
- `work/design/sortable-lists.md` — HTML5 drag-and-drop design
- `work/specs/FEAT-073.md` — Full specification with fractional ranking

**Summary:** `<SortableList>` and `<SortableItem>` components powered by SortableJS, with `@std/sortable` backend module for fractional ranking. Requires SortableJS as external dependency.

**Assessment for 1.0:** Too complex for 1.0. External JS dependency conflicts with the "single binary" philosophy. Good candidate for 1.2 or as a documented pattern.

### WYSIWYG Text Editor

**Status:** Mentioned in the priority matrix of `work/design/server-enhanced-components.md` as P3 (defer):

> **WYSIWYG** | ⭐⭐⭐ | Very High complexity | Editor library | 🔴 P3 - Document pattern

**Assessment:** Very high complexity, requires a large external editor library (TipTap, ProseMirror, Quill, etc.). Not appropriate for the prelude — should be a documented pattern showing how to integrate a third-party editor with Basil's form system.

### SearchField / ComboField

**Status:** Fully designed in `work/design/server-enhanced-components.md` §1, with a prototype plan. This is the flagship example of the "server+client pattern" where the server generates a temporary endpoint for the search function.

**Assessment:** The temporary endpoint concept is architecturally significant and not yet implemented. The component itself is valuable (every app with a database needs search), but the infrastructure isn't ready for 1.0. Good candidate for 1.1.

### Pagination

**Status:** Designed in `work/design/server-enhanced-components.md` §3. Marked as P0 (Do Now) in the priority matrix but never implemented.

**Assessment:** Pure server component, no JS required. Very high value. Should be prioritised for 1.0 or 1.1 at the latest.

### Toasts / Flash Messages

**Status:** Designed in `work/design/server-enhanced-components.md` §4. Marked as P0 (Do Now) but never implemented.

**Assessment:** Low complexity (~50 lines JS), very high value. Should be in 1.0 or 1.1.

### Skeleton Loaders

**Status:** Designed in `work/design/server-enhanced-components.md` §5. Marked as P0 (Do Now) but never implemented.

**Assessment:** Very low complexity (pure CSS), useful for Part lazy loading. Good candidate for 1.1.

### CommandPalette / Spotlight

**Status:** Mentioned in priority matrix as P3 (defer). No detailed design.

**Assessment:** Cool but not essential. Defer to 2.0.

### InfiniteScroll

**Status:** Designed in `work/design/server-enhanced-components.md` §7.

**Assessment:** Medium complexity, depends on Parts system. Defer to 1.2.

### FileField with Upload Progress

**Status:** Designed in `work/design/server-enhanced-components.md` §8.

**Assessment:** High complexity (JS for progress/drag-drop), but a basic `FileField` without progress is simple. Basic version for 1.1, enhanced version for 1.2.

### LiveForm / Auto-Save

**Status:** Designed in `work/design/server-enhanced-components.md` §9.

**Assessment:** Medium complexity, depends on Parts system and debounce. Defer to 1.2.

### Modal with Part Content

**Status:** Designed in `work/design/server-enhanced-components.md` §6.

**Assessment:** A basic `Dialog` component (using native `<dialog>`) is simple and doesn't need Part integration. Part-loaded dialog content can come later. Basic `Dialog` for 1.1.

---

## 13. Component-by-Component Review

### `Page`

**File:** `server/prelude/components/page.pars`

| Aspect | Assessment |
|--------|-----------|
| Prop spreading | ❌ No — enumerates all props |
| Class support | ⚠️ On `<body>` only, no `<html>` class |
| Accessibility | ✅ SkipLink included, lang attribute |
| Extensibility | ⚠️ `head` prop exists but limited |
| Modern standards | ⚠️ Missing `color-scheme`, no `dir`, no `htmlAttrs` |

**Issues:**
1. `id` defaults to `"main"` on `<body>` — conflicts with `<main id="main">` pattern. Should default to `null`.
2. No `dir` attribute on `<html>` for RTL support.
3. No `bodyAttrs` or `htmlAttrs` for arbitrary attributes.
4. No `scripts` slot for additional scripts after basil.js.
5. `<CSS/>` and `head` ordering — user content goes after CSS, which is correct for overrides but not documented.

**Recommended changes:**
- Remove default `id="main"` from `<body>` (or change to `null`)
- Add `dir` prop on `<html>`
- Add `bodyAttrs` prop for spread onto `<body>`
- Add `scripts` prop for content after `<BasilJS/>`
- Add `colorScheme` prop → `<meta name="color-scheme">`

### `Head`

**File:** `server/prelude/components/head.pars`

| Aspect | Assessment |
|--------|-----------|
| Completeness | ✅ Excellent — OG, Twitter, favicons, canonical, noIndex |
| Name clarity | ❌ Confusing overlap with `Page`'s `<head>` |
| Usage | ⚠️ Generates own `<head>` tag — can't be nested inside `Page` |

**Recommended changes:**
- Rename to `Meta` or `SEO` to clarify its purpose
- Make it output meta tags without the `<head>` wrapper, so it can be used inside `Page`'s `head` prop
- OR absorb its functionality into `Page` props

### `TextField`

**File:** `server/prelude/components/text_field.pars`

| Aspect | Assessment |
|--------|-----------|
| Prop spreading | ✅ `...inputAttrs` |
| Accessibility | ✅ Excellent |
| Consistency | ⚠️ Uses `required="required"` (string), others use `required={required}` (boolean) |

**Issues:**
1. `required={if (required) "required" else null}` — should be `required={required}` for consistency with other components. HTML boolean attributes don't need a string value.
2. Uses `.length()` method call on `describedByParts` while other components use `.length` property — inconsistency (though both may work in Parsley).

### `TextareaField`

**File:** `server/prelude/components/textarea_field.pars`

| Aspect | Assessment |
|--------|-----------|
| Prop spreading | ✅ `...textareaAttrs` |
| Features | ✅ Character counter, auto-resize |
| Accessibility | ✅ Counter linked via `aria-describedby` |

**Issues:**
1. Counter references `maxlength` as a bare identifier on line `textareaValue.length " / " maxlength` — but `maxlength` is destructured from `props` via `...textareaAttrs`, not explicitly extracted. This might reference `props.maxlength` or be undefined. Should verify.

### `SelectField`

**File:** `server/prelude/components/select_field.pars`

| Aspect | Assessment |
|--------|-----------|
| Prop spreading | ✅ `...selectAttrs` |
| Features | ✅ Simple values AND objects, placeholder, autosubmit |
| Accessibility | ✅ Good |

**Issues:**
1. Option type detection uses `opt.type == "dictionary"` — this works for Parsley's type system but should be documented.
2. No `disabled` options support (individual options can't be disabled).
3. No optgroup support.
4. **Schema enum integration** — When the form-binding bridge (§9) is implemented, `SelectField` should be able to auto-populate its options from `schema.enumValues(field)` or `record.enumValues(field)`. A schema like `role: enum("admin", "editor", "viewer")` already carries the option list; requiring the developer to re-specify it manually as `options={["admin", "editor", "viewer"]}` is the same kind of redundancy as `DataTable` requiring manual `columns`/`keys`. With `record` awareness, `<SelectField field="role" record={user}/>` could derive options, label, value, and error automatically.

### `RadioGroup`

**File:** `server/prelude/components/radio_group.pars`

| Aspect | Assessment |
|--------|-----------|
| Prop spreading | ✅ `...fieldsetAttrs` |
| Accessibility | ✅ Excellent — fieldset/legend, only first radio required |
| Consistency | ✅ Matches CheckboxGroup pattern |

**No issues found.** This is one of the best-implemented components.

**Schema note:** Like `SelectField`, `RadioGroup` would benefit from `schema.enumValues(field)` integration. Enum fields with a small number of options are a natural fit for radio buttons, and the schema already carries the option list. With `record` awareness: `<RadioGroup field="role" record={user}/>` would derive options, label, selected value, and error from the record.

### `CheckboxGroup`

**File:** `server/prelude/components/checkbox_group.pars`

| Aspect | Assessment |
|--------|-----------|
| Prop spreading | ✅ `...fieldsetAttrs` |
| Accessibility | ✅ Good |
| Name convention | ✅ Uses `name[]` for multiple values |

**No significant issues.**

### `Checkbox`

**File:** `server/prelude/components/checkbox.pars`

| Aspect | Assessment |
|--------|-----------|
| Prop spreading | ❌ Enumerates all props |
| Naming | ⚠️ Should be `CheckboxField` for consistency |
| Accessibility | ✅ Good |

**Issues:**
1. No prop spreading — can't add `data-*` or arbitrary attributes.
2. Name should follow the `*Field` convention.

### `Button`

**File:** `server/prelude/components/button.pars`

| Aspect | Assessment |
|--------|-----------|
| Prop spreading | ✅ `...buttonAttrs` |
| Defaults | ✅ `type="button"` (not "submit") — correct |
| Features | ✅ Toggle, copy, confirm |

**Issues:**
1. Toggle `aria-expanded` initialises to `"false"` — should check target's actual visibility state.

### `Form`

**File:** `server/prelude/components/form.pars`

| Aspect | Assessment |
|--------|-----------|
| Prop spreading | ✅ `...formAttrs` |
| CSRF | ✅ Auto-injected |
| Accessibility | ⚠️ No `novalidate` option for server-validation-only forms |

**Issues:**
1. No integration with `@record` system — see §9.
2. No `enctype` handling for file uploads (important for `FileField`).
3. The `confirm` feature depends on `basil.js` — should document this.

### `A` (Link)

**File:** `server/prelude/components/a.pars`

| Aspect | Assessment |
|--------|-----------|
| Prop spreading | ✅ `...anchorAttrs` |
| Security | ✅ `rel="noopener noreferrer"` for external |
| Naming | ⚠️ Should be `Link` |

**Issues:**
1. Name `A` is hard to search for and not guessable. Should be `Link`.
2. `external` prop auto-detection could be smarter — detect `https://` or `http://` in `href` automatically:
   ```parsley
   let isExternal = external || target == "_blank" || (href && href.startsWith("http"))
   ```

### `Nav`

**File:** `server/prelude/components/nav.pars`

| Aspect | Assessment |
|--------|-----------|
| Prop spreading | ❌ Enumerates `id`, `class`, `label` only |
| Simplicity | ✅ Minimal and correct |

**Issues:**
1. No prop spreading — can't add `data-*` attributes or `role`.

### `Breadcrumb`

**File:** `server/prelude/components/breadcrumb.pars`

| Aspect | Assessment |
|--------|-----------|
| Schema.org markup | ✅ Excellent |
| Accessibility | ✅ `aria-label`, `aria-current="page"`, `<ol>` structure |
| Prop spreading | ❌ Enumerates props |

**Minor issue:** `itemscope=""` uses empty string — HTML boolean attributes should not need a value. Verify Parsley's handling of boolean attributes.

### `Img`

**File:** `server/prelude/components/img.pars`

| Aspect | Assessment |
|--------|-----------|
| Prop spreading | ✅ `...imgAttrs` |
| Defaults | ✅ `loading="lazy"`, `decoding="async"` |
| Accessibility | ✅ `alt` defaults to empty string for decorative |

**Issues:**
1. No `fetchpriority` support for hero/LCP images.
2. No `priority` convenience prop that sets `loading="eager"`, `decoding="auto"`, `fetchpriority="high"`.
3. `alt` defaulting to `""` is correct for decorative images, but if the developer forgets `alt`, the image is treated as decorative rather than erroring. Consider a dev-mode warning.

### `Iframe`

**File:** `server/prelude/components/iframe.pars`

| Aspect | Assessment |
|--------|-----------|
| Prop spreading | ❌ Enumerates all props |
| Defaults | ✅ `loading="lazy"` |
| Accessibility | ✅ `title` required |

**Issues:**
1. No prop spreading — can't add `data-*` attributes.
2. Should use `...iframeAttrs` pattern like other components.

### `Figure`

**File:** `server/prelude/components/figure.pars`

| Aspect | Assessment |
|--------|-----------|
| Simplicity | ✅ Correct and minimal |
| Prop spreading | ❌ Enumerates `id`, `class`, `caption` only |

### `Blockquote`

**File:** `server/prelude/components/blockquote.pars`

| Aspect | Assessment |
|--------|-----------|
| Semantics | ✅ `<figure>` + `<blockquote>` + `<figcaption>` + `<cite>` |
| Prop spreading | ❌ Enumerates props |

### `SkipLink`

**File:** `server/prelude/components/skip_link.pars`

| Aspect | Assessment |
|--------|-----------|
| Accessibility | ✅ Correct pattern |
| CSS | ⚠️ Inline `<style>` — should be in CSS bundle |
| Target | ⚠️ Defaults to `#main` which is on `<body>` — should target `<main>` element |

### `SrOnly`

**File:** `server/prelude/components/sr_only.pars`

| Aspect | Assessment |
|--------|-----------|
| Simplicity | ✅ Minimal |
| CSS | ⚠️ Relies on `.sr-only` class being defined — where? |

**Issue:** The component outputs `class="sr-only"` but doesn't define the CSS. Where is `.sr-only` defined? It needs to be in the CSS bundle or inline. If it's not defined, the component is broken.

### `Abbr`

**File:** `server/prelude/components/abbr.pars`

| Aspect | Assessment |
|--------|-----------|
| Simplicity | ✅ Correct |
| Prop spreading | ❌ Enumerates props |

### `Icon`

**File:** `server/prelude/components/icon.pars`

| Aspect | Assessment |
|--------|-----------|
| Accessibility | ✅ `aria-hidden`, optional sr-only label |
| Implementation | ⚠️ CSS-class-based icons (`icon-{name}`) — requires user to set up icon font/CSS |

**Issue:** The component assumes a CSS icon system (FontAwesome, custom icon font) where `.icon-search` displays the search icon. This is a reasonable assumption but should be documented. No built-in icons are provided.

### `DataTable`

**File:** `server/prelude/components/data_table.pars`

| Aspect | Assessment |
|--------|-----------|
| Accessibility | ✅ `scope="col"`, `scope="row"`, `<caption>` |
| Extensibility | ❌ No cell render customisation |
| Fit with `Table` type | ❌ Ignores it — requires manual decomposition into `columns`/`rows`/`keys` |
| Features | ⚠️ `sortable` prop unused |

**Core issue:** `DataTable`'s API predates the Parsley `Table` type and doesn't use it. Users must unpack a `Table` into parallel arrays (`columns`, `rows`, `keys`) that the `Table` already carries. See §3 for the full analysis and proposed redesign.

**Summary of needed changes:**
1. Accept a `Table` directly via a `data` prop — derive columns and rows automatically
2. Add empty state (`empty` prop, default: "No data")
3. Add cell formatting based on value/schema types (`format` prop)
4. Add custom cell rendering (`render` prop with per-column functions)
5. Add footer/summary row support (`footer` prop)
6. Remove unused `sortable` prop (sorting is a `Table.orderBy()` concern, not a presentation concern)
7. Keep current `columns`/`rows`/`keys` props as manual fallback for raw arrays

### Time Components (`Time`, `LocalTime`, `TimeRange`, `RelativeTime`)

| Aspect | Assessment |
|--------|-----------|
| Progressive enhancement | ✅ Excellent |
| Accessibility | ✅ `datetime`, `title`, `aria-live` |
| Client-side | ✅ Proper `Intl` API usage |
| Prop spreading | ❌ All enumerate props |
| Built-in type awareness | ⚠️ Not `datetime`-aware |

**These are the highest quality components in the prelude.** The only issues are the lack of prop spreading on the custom elements (can't add `data-*` or `style` attributes) and the lack of `datetime` type awareness.

**`datetime` type awareness:** Parsley has a first-class `datetime` type (returned by `date()`, `datetime()`, database queries, etc.) with `.iso`, `.medium()`, `.short()`, `.long()`, and `.fmt(style, locale)` methods. Currently users must extract the ISO string manually:

```parsley
// Current: user extracts .iso
<LocalTime datetime={post.publishedAt.iso}/>

// Better: component detects datetime type automatically
<LocalTime datetime={post.publishedAt}/>
```

When `datetime` receives a `datetime` object, the component should:
- Extract `.iso` for the `datetime` HTML attribute
- Use `.medium()` for the server-rendered fallback text (→ "Mar 15, 2025 10:30 AM") instead of the raw ISO string
- Continue to accept plain strings for backward compatibility

This doesn't restrict usage — strings still work — but gives a real ergonomic and rendering win since database queries and the `date()`/`datetime()` builtins return `datetime` objects. The same applies to `TimeRange` (which takes two datetime values) and `RelativeTime`.

---

## 14. Proposed New Components

### `Pagination` (Priority: High)

Based on the design in `work/design/server-enhanced-components.md`:

```parsley
<Pagination
    current={page}
    total={totalItems}
    perPage={20}
    href="/products?page={page}"
    window={2}
/>
```

**Output:** `<nav aria-label="Pagination">` with `<a>` links, current page marked with `aria-current="page"`, ellipsis for gaps, first/last/prev/next buttons with `aria-label`.

**Effort:** 2-3 hours (pure server component, no JS)

### `Toast` / `Flash` (Priority: High)

```parsley
// In page layout:
<Toasts position="top-right" duration={5000}/>

// Reads from basil.session.flash automatically
```

**Output:** `<div class="toasts" aria-live="polite">` with individual toast elements.

**JS needed:** ~50 lines for auto-dismiss, hover pause, manual dismiss.

**Effort:** 2-3 hours (component + JS)

### `Dialog` (Priority: Medium)

```parsley
<Dialog id="confirm-delete" title="Confirm Delete">
    <p>"Are you sure?"</p>
    <Button onclick="this.closest('dialog').close()">"Cancel"</Button>
    <Button type="submit" form="delete-form">"Delete"</Button>
</Dialog>

<Button onclick="document.getElementById('confirm-delete').showModal()">"Delete"</Button>
```

Uses native `<dialog>` element — no JS library needed. `basil.js` adds backdrop click-to-close and auto-focus.

**Effort:** 1-2 hours

### `Details` / `Accordion` (Priority: Medium)

```parsley
// Single disclosure:
<Details summary="More information">
    <p>"Hidden content here"</p>
</Details>

// Accordion (exclusive — only one open):
<Accordion>
    <Details summary="Section 1">"Content 1"</Details>
    <Details summary="Section 2">"Content 2"</Details>
</Accordion>
```

Uses native `<details>`/`<summary>` — zero JS for single disclosure, ~15 lines JS for exclusive accordion mode.

**Effort:** 1 hour

### `ErrorSummary` (Priority: Medium)

```parsley
// Manual mode — flat errors dict:
<ErrorSummary errors={validationErrors} title="There are problems with your form"/>

// Record-aware mode — derives everything from the record:
<ErrorSummary record={user}/>
```

When given a `record`, the component can derive the full error summary automatically:
- Call `record.errorList()` to get all validation errors as an array
- Call `record.schema().title(field)` to get the display label for each field (→ "Email" instead of "email")
- Generate `href="#field-{name}-input"` links using the schema field names

The flat `errors` prop remains as a fallback for non-schema usage.

**Output:**

```html
<div class="error-summary" role="alert" aria-labelledby="error-summary-title">
    <h2 id="error-summary-title">There are problems with your form</h2>
    <ul>
        <li><a href="#field-email-input">Email — is required</a></li>
        <li><a href="#field-name-input">Name — must be at least 2 characters</a></li>
    </ul>
</div>
```

Links jump to the relevant field. Follows GOV.UK Design System pattern.

**Effort:** 1-2 hours (record-aware mode adds minimal complexity since `record.errorList()` and `schema.title()` already exist)

### `FileField` (Priority: Medium)

Basic version (no upload progress):

```parsley
<FileField name="avatar" label="Profile Photo" accept="image/*"/>
```

Enhanced version (with drag-drop and preview, later):

```parsley
<FileField 
    name="attachments" 
    label="Upload Files"
    accept="image/*,.pdf"
    multiple={true}
    dragDrop={true}
    preview={true}
/>
```

Basic version: 1 hour. Enhanced version: 3-4 hours.

### `LocalNumber` / `LocalCurrency` (Priority: Medium)

See §8. Custom elements with `Intl.NumberFormat`.

**Effort:** 1 hour each

---

## 15. Summary of Recommendations

### Tier 1: Must Do for 1.0

| # | Item | Effort | Section | Status |
|---|------|--------|---------|--------|
| 1 | Fix `Page` `id="main"` default on `<body>` (change to `null`) | 5 min | §13 | |
| 2 | Fix `SkipLink` target — should point to `<main>` not `<body>` | 15 min | §5 | |
| 3 | Add `dir` prop to `Page` for RTL support | 5 min | §8 | |
| 4 | Add prop spreading to all components that lack it | 1-2 hours | §7 | |
| 5 | Rename `A` → `Link` (keep `A` as alias) | 15 min | §2 | |
| 6 | Add `fetchpriority`/`priority` support to `Img` | 15 min | §4 | |
| 7 | Move `SkipLink` CSS from inline `<style>` to CSS bundle | 15 min | §4 | |
| 8 | Normalise `required` attribute handling across all form components | 30 min | §13 | |
| 9 | Verify and document `import @basil/html as x` pattern for namespacing | 30 min | §1 | |
| 10 | Rename `Head` → `Meta`, strip `<head>` wrapper, use inside `Page.head` prop | 1-2 hours | §7 | **DECIDED** |
| 11 | Verify `.sr-only` CSS is actually defined somewhere | 15 min | §13 | |
| 12 | Fix `objectToString`/`objectToPrintString` to call `.medium()` on `money`, `datetime`, `unit`, `duration` values | 30 min | §9 | **DECIDED** |
| 13 | Fix `dict.toHTML({table: true})` option parsing bug (expression not evaluated) | 10 min | §9 | |
| 14 | Fix time component bugs: `.format("iso")` → `.iso`, missing `.relative()` method | 30 min | §9 | |
| 15 | Fix `++` operator misuse in class merging across all 12 components (should be `+` or use `cx()`) | 30 min | §7, §11 | |
| 16 | FEAT-143: Replace invalid tag spread syntax `{...attrs}` with Parsley spread syntax `...attrs` across all affected components and docs | 20-30 min | §16 | |
| 17 | FEAT-143: Fix reversed `for (item, i in items)` / `for (item, idx in items)` loops to Parsley-correct `for (i, item in items)` form | 20-30 min | §16 | |
| 18 | FEAT-143: Fix pagination page range logic (`start..end`, not `start..end + 1`) and verify edge cases | 15 min | §16 | |
| 19 | FEAT-143: Add focused Parsley tests for new/updated prelude components so syntax/runtime errors are caught by CI | 1-2 hours | §16 | |
| 20 | FEAT-143: Correct spec/design examples that currently show non-idiomatic or invalid Parsley patterns | 30-45 min | §16 | |

**Estimated total: 7-11 hours**

### Tier 2: Should Do for 1.0 or 1.1

| # | Item | Effort | Section |
|---|------|--------|---------|
| 16 | `Pagination` component | 2-3 hours | §14 |
| 17 | `Toast`/`Flash` component + JS | 2-3 hours | §14 |
| 18 | `Dialog` component (native `<dialog>`) | 1-2 hours | §14 |
| 19 | `Details`/`Accordion` component | 1 hour | §14 |
| 20 | `ErrorSummary` component (with `record`-aware mode) | 1-2 hours | §14 |
| 21 | Rename `Checkbox` → `CheckboxField` (keep alias) | 15 min | §2 |
| 22 | Add `bodyAttrs`, `scripts` props to `Page` | 1 hour | §7 |
| 23 | `DataTable` redesign: accept `Table` directly, auto-format typed cell values (`money`/`datetime`/`unit` via `.medium()`), empty state, remove `sortable` | 2-3 hours | §3, §9, §13 |
| 24 | `cx()` class merging utility | 30 min | §11 |
| 25 | Add `scope="col"` to `table.toHTML()` `<th>` elements | 5 min | §9 |

**Estimated total: 12-17 hours**

### Tier 3: Post-1.0 Enhancements (1.1-1.2)

| # | Item | Effort | Section | Status |
|---|------|--------|---------|--------|
| 26 | `DataTable` per-column `format` override prop (for when auto-detection isn't enough) | 1-2 hours | §3 | |
| 27 | `DataTable` custom cell render functions (`render` prop) | 2-3 hours | §3 | |
| 28 | `DataTable` footer/summary row support (`footer` prop) | 1-2 hours | §3 | |
| 29 | `FileField` (basic) | 1 hour | §14 | |
| 30 | `LocalNumber` / `LocalCurrency` components + JS (with `money` type awareness) | 2 hours | §8 | |
| 31 | Implement `record.fieldProps(name, options?)` with hardwired type mappings + metadata overrides | 4-6 hours | §9 | **DECIDED** |
| 32 | Time components: accept `datetime` objects directly (auto-extract `.iso`, use `.medium()` for fallback) | 1 hour | §13 | |
| 33 | `record.toHTML()` with schema-aware field titles and formatted values | 1-2 hours | §9 | |
| 34 | Auto-detect external links in `Link` component | 15 min | §13 | |
| 35 | `request.language` server feature | 2 hours | §8 | |
| 36 | `Skeleton` component (pure CSS) | 1 hour | §12 | |
| 37 | `SearchField` prototype | 4-8 hours | §12 | |

### Tier 4: Defer to 2.0 or Never

| Item | Reason |
|------|--------|
| WYSIWYG editor | Too complex, external dependency |
| CommandPalette | Niche, high complexity |
| SortableList | External JS dependency, complex server integration |
| Full i18n/translation framework | Massive scope, needs dedicated design |
| Push-Parts (SSE) | Infrastructure change, not a component |
| Named slots syntax | Language change, out of scope |

---

## Appendix A: Component Prop Spreading Audit

Components that need `...rest` spreading added:

| Component | Primary Element | Change Needed |
|-----------|----------------|---------------|
| `Checkbox` | `<div>` wrapper | Add `...rest` to wrapper or input |
| `Nav` | `<nav>` | Add `...navAttrs` |
| `Breadcrumb` | `<nav>` | Add `...navAttrs` |
| `Iframe` | `<iframe>` | Add `...iframeAttrs` |
| `Figure` | `<figure>` | Add `...figureAttrs` |
| `Blockquote` | `<figure>` | Add `...blockquoteAttrs` |
| `SkipLink` | `<a>` | Low priority (internal component) |
| `SrOnly` | `<span>` | Add `...spanAttrs` |
| `Abbr` | `<abbr>` | Add `...abbrAttrs` |
| `Icon` | `<span>` | Add `...iconAttrs` |
| `DataTable` | `<table>` | Add `...tableAttrs` |
| `Time` | `<time>` | Add `...timeAttrs` |
| `LocalTime` | `<local-time>` | Add `...attrs` |
| `TimeRange` | `<time-range>` | Add `...attrs` |
| `RelativeTime` | `<relative-time>` | Add `...attrs` |

## Appendix B: Design Document Cross-References

| Document | Relevance to This Review |
|----------|-------------------------|
| `work/design/server-enhanced-components.md` | Pagination, Toast, Skeleton, SearchField, FileField, SortableList, Modal, InfiniteScroll, LiveForm, CommandPalette, WYSIWYG designs |
| `work/design/sortable-list-component.md` | Detailed SortableList component design |
| `work/design/sortable-lists.md` | HTML5 drag-and-drop approach |
| `work/specs/FEAT-051.md` | Standard Prelude umbrella spec |
| `work/specs/FEAT-058.md` | HTML Components in Prelude spec |
| `work/specs/FEAT-073.md` | SortableList with fractional ranking spec |
| `work/reports/STDLIB-1.0-RELEASE-REVIEW.md` | Overall stdlib readiness |
| `work/reports/STDLIB-1.0-ACTION-PLAN.md` | Prioritised action items |
| `work/BACKLOG.md` | #12 (form target partial updates), #17 (locale standardisation), #21 (form validation), #69 (autofocus metadata), #70 (inputmode metadata) |

## Appendix C: Backlog Items Relevant to This Review

| Backlog # | Item | Relevance |
|-----------|------|-----------|
| #12 | Form `target=` partial updates (Turbo-style) | Would enhance `Form` component with AJAX submission |
| #17 | Standardise locale support across stdlib | Directly relevant to §8 (i18n) |
| #21 | Form validation/sanitization | Relevant to §9 (form binding bridge) |
| #69 | `autofocus` metadata for form binding | Would enhance `@field` → prelude bridge |
| #70 | `inputmode` metadata for form binding | Would enhance mobile experience for form fields |

## Appendix D: FEAT-143 Implementation Fix Strategy

**Status:** Blocking issues identified — implementation not yet correct  
**Last verified:** 2026-03-15

### Executive Summary

The FEAT-143 implementation (Pico CSS compatibility) is strategically sound but contains blocking Parsley syntax errors that prevent components from running correctly. This appendix provides a concrete fix strategy with verification commands.

### Blocking Issues

| Issue | Severity | Files Affected | Fix Complexity |
|-------|----------|----------------|----------------|
| Invalid `{...attrs}` spread syntax | 🔴 Critical | 9 files | Simple find/replace |
| Reversed `for` loop variable ordering | 🔴 Critical | 6 files | Careful refactor |
| Pagination range precedence bug | 🔴 Critical | 1 file | One-line fix |

### Issue 1: Invalid Tag Spread Syntax

**Problem:** JSX-style `{...attrs}` was used instead of Parsley's `...attrs`.

**Verification (confirms the bug):**
```bash
# This should error:
pars -e 'let a = {x: 1}; <div {...a}>"test"</div>'
# Error: unexpected '...'

# This works:
pars -e 'let a = {x: 1}; <div ...a>"test"</div>'
# Output: <div x="1">test</div>
```

**Affected files:**
| File | Line | Current | Fix |
|------|------|---------|-----|
| `accordion.pars` | 11 | `{...attrs}` | `...attrs` |
| `breadcrumb.pars` | 11 | `{...attrs}` | `...attrs` |
| `details.pars` | 7 | `{...attrs}` | `...attrs` |
| `dialog.pars` | 7 | `{...attrs}` | `...attrs` |
| `error_summary.pars` | 19 | `{...attrs}` | `...attrs` |
| `pagination.pars` | 25 | `{...attrs}` | `...attrs` |
| `skip_link.pars` | 8 | `{...attrs}` | `...attrs` |
| `toast.pars` | 12 | `{...attrs}` | `...attrs` |
| `toasts.pars` | 14 | `{...attrs}` | `...attrs` |

**Fix command:**
```bash
cd server/prelude/components
for f in accordion.pars breadcrumb.pars details.pars dialog.pars \
         error_summary.pars pagination.pars skip_link.pars toast.pars toasts.pars; do
    sed -i '' 's/{\.\.\.attrs}/...attrs/g' "$f"
done
```

**Verification (confirms the fix):**
```bash
# Each file should parse without error:
for f in server/prelude/components/*.pars; do
    pars --check "$f" 2>&1 || echo "FAIL: $f"
done
```

### Issue 2: Reversed `for` Loop Variable Ordering

**Problem:** Parsley's two-variable `for` loop is `for (index, value in array)`, not `for (value, index in array)`.

**Verification (demonstrates correct behavior):**
```bash
pars -e 'for (i, item in ["a", "b", "c"]) { i + ":" + item }'
# Output: ["0:a", "1:b", "2:c"]

pars -e 'for (item, i in ["a", "b", "c"]) { i + ":" + item }'
# Output: ["a:0", "b:1", "c:2"]  ← WRONG: item is the index!
```

**Affected files and fixes:**

| File | Line | Current | Fix |
|------|------|---------|-----|
| `accordion.pars` | 10 | `for (item, i in items)` | `for (i, item in items)` |
| `breadcrumb.pars` | 17 | `for (item, idx in items)` | `for (idx, item in items)` |
| `checkbox_group.pars` | 41 | `for (opt, idx in options)` | `for (idx, opt in options)` |
| `radio_group.pars` | 38 | `for (opt, idx in options)` | `for (idx, opt in options)` |
| `data_table.pars` | 11 | `for (col, idx in columns)` | `for (idx, col in columns)` |
| `data_table.pars` | 19 | `for (key, idx in keys)` | `for (idx, key in keys)` |

**Important:** After fixing variable names, verify that the logic still makes sense. The variables are used for:
- `accordion.pars`: `i == 0` check for default open state ✓
- `breadcrumb.pars`: `idx == (items.length() - 1)` for last item, `idx > 0` for separator, `idx + 1` for position ✓
- `checkbox_group.pars`: `fieldId ++ "-" ++ idx` for unique IDs ✓
- `radio_group.pars`: `idx == 0 && required` for first-only required, `fieldId ++ "-" ++ idx` for IDs ✓
- `data_table.pars`: `idx == 0` for row scope on first column ✓

**Verification (confirms correct output):**
```bash
# Test accordion renders items correctly:
pars -r -e '
let Accordion = fn({name, items}) {
    for (i, item in items) {
        <details name={name} open={i == 0}>
            <summary>item.title</summary>
            item.content
        </details>
    }
}
<Accordion name="faq" items={[{title: "Q1", content: "A1"}, {title: "Q2", content: "A2"}]}/>
'
# Should output: <details name="faq" open>... (first open, second not)

# Test breadcrumb position numbering:
pars -r -e '
for (idx, item in [{label: "Home"}, {label: "Products"}]) {
    <li><meta itemprop="position" content={idx + 1}/> item.label</li>
}
'
# Should output position 1, 2 (not 0, 1)
```

### Issue 3: Pagination Range Precedence Bug

**Problem:** `start..end + 1` is parsed as `(start..end) + 1` due to operator precedence, causing a type error.

**Verification (demonstrates the bug):**
```bash
pars -e '1..5 + 1'
# Error: Type mismatch: array + integer

pars -e '1..(5 + 1)'
# Output: [1, 2, 3, 4, 5, 6]
```

**Location:** `pagination.pars` line 45

**Current code:**
```parsley
for (n in start..end + 1) {
```

**Analysis:** The `..` operator is inclusive, so `1..5` produces `[1, 2, 3, 4, 5]`. If the intent is to include `end` in the range, `start..end` is sufficient. If the intent is to go one past `end`, use `start..(end + 1)`.

**Fix:** Change to `for (n in start..end) {` (since `end` is already calculated as `min(totalPages, current + pageWindow)` and should be inclusive).

**Verification:**
```bash
# Test pagination range logic:
pars -e '
let start = 2
let end = 5
for (n in start..end) { n }
'
# Should output: [2, 3, 4, 5]
```

### Component-by-Component Verification

After applying all fixes, verify each component renders correctly:

```bash
#!/bin/bash
# Save as verify-prelude.sh

PRELUDE="server/prelude/components"

echo "=== Verifying Prelude Components ==="

# 1. Syntax check all files
echo -e "\n--- Syntax Check ---"
for f in $PRELUDE/*.pars; do
    if ! pars --check "$f" 2>/dev/null; then
        echo "FAIL: $f"
        exit 1
    fi
done
echo "All files pass syntax check"

# 2. Test accordion
echo -e "\n--- Accordion ---"
pars -r -e '
{Accordion} = import @basil/html
<Accordion name="test" items={[{title: "One", content: "First"}, {title: "Two", content: "Second"}]}/>
' | grep -q 'name="test"' && echo "OK: Accordion" || echo "FAIL: Accordion"

# 3. Test breadcrumb  
echo -e "\n--- Breadcrumb ---"
pars -r -e '
{Breadcrumb} = import @basil/html
<Breadcrumb items={[{label: "Home", href: "/"}, {label: "About"}]}/>
' | grep -q 'itemprop="position" content="2"' && echo "OK: Breadcrumb positions correct" || echo "FAIL: Breadcrumb"

# 4. Test pagination
echo -e "\n--- Pagination ---"
pars -r -e '
{Pagination} = import @basil/html
<Pagination current={3} total={100} perPage={10} href="/p?page={page}"/>
' | grep -q 'aria-current="page"' && echo "OK: Pagination" || echo "FAIL: Pagination"

# 5. Test toast
echo -e "\n--- Toast ---"
pars -r -e '
{Toast} = import @basil/html
<Toast message="Hello" type="success"/>
' | grep -q 'data-type="success"' && echo "OK: Toast" || echo "FAIL: Toast"

# 6. Test dialog
echo -e "\n--- Dialog ---"
pars -r -e '
{Dialog} = import @basil/html
<Dialog id="test" title="Test Dialog">"Content"</Dialog>
' | grep -q '<dialog id="test">' && echo "OK: Dialog" || echo "FAIL: Dialog"

# 7. Test error_summary
echo -e "\n--- ErrorSummary ---"
pars -r -e '
{ErrorSummary} = import @basil/html
<ErrorSummary errors={[{field: "email", message: "Invalid"}]}/>
' | grep -q 'role="alert"' && echo "OK: ErrorSummary" || echo "FAIL: ErrorSummary"

echo -e "\n=== Verification Complete ==="
```

### Integration Test Requirements

Add these test cases to `pkg/parsley/tests/`:

```go
// prelude_components_test.go

func TestAccordionRendersCorrectly(t *testing.T) {
    code := `
{Accordion} = import @basil/html
<Accordion name="faq" items={[
    {title: "Q1", content: "A1"},
    {title: "Q2", content: "A2"}
]}/>
`
    result := evalToString(t, code)
    
    // First item should be open by default
    assert.Contains(t, result, `<details name="faq" open>`)
    // Second item should not be open
    assert.Contains(t, result, `<details name="faq">`)
    assert.NotContains(t, result, `<details name="faq" open><details name="faq" open>`)
}

func TestBreadcrumbPositionsAreOneBased(t *testing.T) {
    code := `
{Breadcrumb} = import @basil/html
<Breadcrumb items={[{label: "Home", href: "/"}, {label: "Products", href: "/products"}, {label: "Shoes"}]}/>
`
    result := evalToString(t, code)
    
    // Positions should be 1, 2, 3 (not 0, 1, 2)
    assert.Contains(t, result, `content="1"`)
    assert.Contains(t, result, `content="2"`)
    assert.Contains(t, result, `content="3"`)
    assert.NotContains(t, result, `content="0"`)
}

func TestPaginationRangeIncludesEnd(t *testing.T) {
    code := `
{Pagination} = import @basil/html
<Pagination current={5} total={100} perPage={10} href="/p?page={page}" window={2}/>
`
    result := evalToString(t, code)
    
    // With current=5, window=2, should show pages 3,4,5,6,7
    assert.Contains(t, result, `>3</a>`)
    assert.Contains(t, result, `>4</a>`)
    assert.Contains(t, result, `aria-current="page">5</a>`)
    assert.Contains(t, result, `>6</a>`)
    assert.Contains(t, result, `>7</a>`)
}

func TestRadioGroupGeneratesUniqueIds(t *testing.T) {
    code := `
{RadioGroup} = import @basil/html
<RadioGroup name="size" label="Size" options={["S", "M", "L"]} value="M"/>
`
    result := evalToString(t, code)
    
    // IDs should be field-size-0, field-size-1, field-size-2
    assert.Contains(t, result, `id="field-size-0"`)
    assert.Contains(t, result, `id="field-size-1"`)
    assert.Contains(t, result, `id="field-size-2"`)
}
```

### Parsley Idiom Reference (Quick Guide)

For future prelude development, these are the verified correct Parsley patterns:

| Pattern | ❌ Wrong | ✅ Correct |
|---------|----------|-----------|
| Tag spread | `<div {...attrs}>` | `<div ...attrs>` |
| For loop (array) | `for (item, idx in arr)` | `for (idx, item in arr)` |
| For loop (dict) | `for (val, key in dict)` | `for (key, val in dict)` |
| Conditional attr | `attr={cond && "value"}` | `attr={if (cond) "value" else null}` |
| Default value | `fn({x = 5})` | `fn({x}) { let val = x ?? 5 ... }` |
| String concat | `"a" ++ "b"` (creates array) | `"a" + "b"` |
| Class merging | `"base" ++ if (c) " " ++ c else ""` | `("base" ++ class).join(" ")` |
| Range inclusive | `1..5` → `[1,2,3,4,5]` | (this is correct) |
| Range + arithmetic | `1..5 + 1` (error) | `1..(5 + 1)` |

### Checklist for Completion

- [ ] Fix `{...attrs}` → `...attrs` in 9 files
- [ ] Fix `for` loop ordering in 6 files
- [ ] Fix pagination range in 1 file
- [ ] Run `pars --check` on all prelude `.pars` files
- [ ] Run verification script (above)
- [ ] Add integration tests to `pkg/parsley/tests/`
- [ ] Update FEAT-143 spec examples to use correct syntax
- [ ] Update design doc examples to use correct syntax

### Process Improvements

To prevent similar issues in future:

1. **Verify examples before committing:** Run `pars -e "..."` or `pars -r -e "..."` on any Parsley code in specs/docs
2. **Add prelude smoke tests:** CI should evaluate each prelude component with sample props
3. **Document Parsley vs JavaScript differences:** The cheatsheet at `docs/parsley/CHEATSHEET.md` should prominently cover these gotchas
4. **Review checklist item:** "Did you test the Parsley examples in this PR?"
