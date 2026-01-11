# Rails-Inspired UX Niceties

Design notes for features inspired by Ruby on Rails that improve developer and user experience without requiring JavaScript expertise.

## Overview

Rails pioneered several patterns that make web development more productive and create better user experiences. This document catalogs patterns worth considering for Basil, noting what Parsley already provides and what would need to be added.

---

## Already Covered by Parsley

These Rails features are already available in Parsley:

| Rails Feature | Parsley Equivalent |
|---------------|-------------------|
| `number_to_currency(1234.5)` | `1234.5.currency("USD")` → `"$1,234.50"` |
| `number_with_delimiter(12345)` | `12345.format()` → `"12,345"` |
| `number_to_percentage(0.65)` | `0.65.percent()` → `"65%"` |
| `time_ago_in_words(@-1d)` | `@-1d.format()` → `"yesterday"` |
| `distance_of_time_in_words(future)` | `(future - now()).format()` → `"in 2 weeks"` |
| `cycle("odd", "even")` | `for (item, i in items)` + `i % 2` |
| Date formatting (localized) | `dt.format("long", "de-DE")` |
| Money handling | Native `$12.34`, `EUR#50.00` with exact arithmetic |
| Content slots | `<Layout sidebar=<Nav/>>{children}</Layout>` |

---

## Decided: Add to Parsley

### 1. `.highlight(phrase)` - String Method

Wraps matches in `<mark>` tags with proper HTML escaping.

```parsley
"Search results for dogs".highlight("dogs")
// → "Search results for <mark>dogs</mark>"

"Find <script>".highlight("<script>")
// → "Find <mark>&lt;script&gt;</mark>" (safe!)
```

**Why it's valuable:** Safe HTML wrapping that handles escaping is fiddly to get right. Common need for search results.

**Optional:** Custom wrapper tag:
```parsley
"hello world".highlight("world", "strong")
// → "hello <strong>world</strong>"
```

### 2. `.paragraphs()` - String Method

Converts plain text with blank lines to HTML paragraphs.

```parsley
"First paragraph.\n\nSecond paragraph.".paragraphs()
// → "<p>First paragraph.</p><p>Second paragraph.</p>"

"Has <html> in it\n\nStill safe".paragraphs()
// → "<p>Has &lt;html&gt; in it</p><p>Still safe</p>"
```

**Why it's valuable:** Common pattern for displaying user-submitted text (blog comments, descriptions). Easy to get XSS wrong when doing it manually.

### 3. `.humanize()` - Number Method

Locale-aware compact number formatting.

```parsley
1234567.humanize()           // "1.2M"
1234567.humanize("de-DE")    // "1,2 Mio."
1000000000.humanize()        // "1B"
```

**Why it's valuable:** Every developer writes this, and usually gets edge cases wrong (trailing zeros, locale separators, billion vs milliard). Go's `golang.org/x/text/message` handles it correctly via CLDR data.

**Different from `.format()`:** `.format()` adds digit separators (`1,234,567`), `.humanize()` compacts (`1.2M`).

---

## Decided: NOT Adding

### `pluralize()` - Too Complex for Proper Implementation

Rails' `pluralize(5, "item")` is English-only (just adds "s").

Proper i18n pluralization requires CLDR plural rules:
- English: 2 forms (one, other)
- Russian: 3 forms (one, few, many)
- Arabic: 6 forms (zero, one, two, few, many, other)

Maintaining CLDR rule tables for 200+ locales is a significant undertaking. Better left to dedicated i18n libraries if users need it.

### `truncate()` - Trivial in Parsley

```parsley
// Rails
truncate("Hello world", 8)  // "Hello..."

// Parsley - already easy
"Hello world"[:5] + "..."   // "Hello..."
```

Not worth a method for 12 characters of code.

### `excerpt()` - Too Specialist

Extracts text around a phrase. Useful for search results but niche enough that users can build it if needed.

### Layout Slots - Already Have

Parsley's tag attributes already provide this:
```parsley
<Layout head=<title>Page</title> sidebar=<Nav/>>
  Main content
</Layout>
```

No need for Rails' `content_for` / `yield` magic.

---

## Needs More Design: Form `target=` (Turbo-Style Partial Updates)

### The Concept

When a form has `target="#element-id"`, instead of full page reload:
1. JavaScript intercepts submit
2. Sends form data via `fetch()`
3. Server returns HTML fragment
4. JS replaces target element's innerHTML
5. No page reload, instant feel

```parsley
<Form action="/comments" method=POST target="#comment-list">
  <textarea name=body/>
  <button>Post</button>
</Form>

<ul id=comment-list>
  {for (c in comments) { <li>{c.body}</li> }}
</ul>
```

### Why It's Valuable

- **No JavaScript knowledge needed** - just add `target="#id"`
- **Progressive enhancement** - works without JS (falls back to full reload)
- **Server-rendered** - no client state management
- **SPA-like feel** without SPA complexity

This is the core idea behind Rails Hotwire/Turbo, Phoenix LiveView, and htmx.

### Design Challenges for Basil

**1. How does the handler know to return a fragment vs full page?**

Options discussed:
- **Always fragment** - handler returns fragment, no-JS gets redirect
- **Check `basil.http.request.isFragment`** - explicit but verbose
- **File convention** - `_fragment.pars` files are fragments (only works for filepath routing)
- **Module-based** - some way to mark a component as "fragment-capable"

**2. How does layout wrapping work?**

Full pages go through layout (html/head/body). Fragments shouldn't. Basil needs to know which is which.

**3. Config-based vs filepath-based routing**

Filename conventions (`_*.pars`) only work for filepath-based routing. Config-based routing (`basil.yaml`) would need a different mechanism - perhaps a route property:

```yaml
routes:
  /comments:
    post: handlers/add-comment.pars
    fragment: true  # Don't wrap in layout
```

**4. The injected JavaScript**

The `<Form>` component would auto-inject ~20 lines of JS (once per page) when `target=` is present. This needs careful design:
- Where in the page does it go?
- How to avoid duplicate injection?
- Should it be configurable/optional?

### Status: Backlogged

This feature has high UX value but needs significant design work to fit Basil's architecture. Added to BACKLOG.md for future consideration.

### Idea: `.parts` File Extension

A promising naming idea: use `.parts` extension (alongside `.pars`) to denote fragment/partial files:

```
pages/
  post.pars           # Full page (wrapped in layout)
  post.parts          # Partial/fragment version (no layout)

components/
  CommentList.pars    # Regular component
  CommentList.parts   # Fragment-ready version
```

The naming fits the Parsley ecosystem nicely:
- `.pars` = Parsley (full pages/components)
- `.parts` = Parts (fragments/partials)

This could solve the "how does Basil know it's a fragment" problem elegantly - the file extension tells it. Needs more design work to determine:
- Can a single file serve both roles somehow?
- How does this interact with config-based routing?
- Does the Form `target=` automatically look for `.parts` files?

---

## Related Specs

| Feature | Spec | Status |
|---------|------|--------|
| Cookies | FEAT-043 | Spec done |
| CSRF Protection | FEAT-044 | Spec done |
| Redirect Helper | FEAT-045 | Spec done |
| Path Pattern Matching | FEAT-046 | Spec done |
| CORS Configuration | FEAT-047 | Spec done |
| Sessions & Flash | Design doc | Done |

---

## Implementation Priority

1. **Text helpers** (highlight, paragraphs, humanize) - Low effort, no dependencies
2. **Cookies** (FEAT-043) - Blocks CSRF and sessions
3. **CSRF** (FEAT-044) - Security requirement
4. **Sessions/Flash** - Depends on cookies
5. **Form target=** - High value but needs design work
