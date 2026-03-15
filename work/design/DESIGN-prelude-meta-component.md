# Design: Prelude Meta Component

**Status:** Revised  
**Date:** 2025-06-15  
**Revised:** 2025-03-15  
**Related:** STANDARD-PRELUDE-REVIEW.md §7.4, FEAT-143  
**Implements:** Prelude Review Decision #1  
**Spec:** FEAT-142

---

## 1. Overview

This document covers the redesign of the prelude `Head` component into `Meta` — a focused component for SEO and social media metadata that composes cleanly with `Page`.

### 1.1 Problem Statement

The current `Head` component has a structural problem:

1. `Head` generates its own `<head>` wrapper tag
2. `Page` also generates a `<head>` section
3. Therefore, `Head` cannot be nested inside `Page` — they're mutually exclusive
4. Both components include `<CSS/>`, risking duplicate asset bundles
5. The relationship between them is confusing to users

### 1.2 Decision

**Rename `Head` → `Meta` and remove the `<head>` wrapper.**

The `Meta` component will output only meta/link tags (Open Graph, Twitter Cards, favicons, canonical URL, etc.) for insertion into `Page`'s `head` prop. This creates a clean composition model:

- `Page` owns document structure (html, head, body, asset bundles)
- `Meta` owns SEO/social metadata (optional, composable)

### 1.3 Parsley Idioms (from FEAT-143)

This design uses correct Parsley syntax validated during FEAT-143:

- **String concatenation:** Use `+` not `++` (which merges arrays)
- **Conditionals:** Use concise `if (cond) expr else expr` for single expressions
- **Spread:** Use `...attrs` not `{...attrs}`
- **Required props:** Use `fail()` for validation errors

---

## 2. Design

### 2.1 Component Responsibilities

| Component | Responsibility |
|-----------|---------------|
| `Page` | Complete HTML document: doctype, html, head (with charset, viewport, title, CSS, OG/Twitter basics), body (with SkipLink, content, JS) |
| `Meta` | SEO/social metadata only: image, canonical URL, favicons, robots directives, article metadata |

### 2.2 Key Design Decision: Title/Description Ownership

**Problem:** `Meta` doesn't have access to `Page`'s title, but Open Graph requires `og:title`.

**Solution:** `Page` outputs `og:title`, `og:description`, `twitter:title`, `twitter:description` from its own props. `Meta` focuses on the *additional* metadata that `Page` doesn't handle (image, url, type, author, favicons).

This means:
- Simple pages get basic OG/Twitter for free — just use `<Page title="..." description="...">`
- `Meta` is only needed for advanced metadata (images, article dates, etc.)

### 2.3 Usage Patterns

**Simple page (most common):**

```parsley
<Page lang="en" title="About Us" description="Learn about our company">
    <main id="main">
        <h1>"About Us"</h1>
        "Content here..."
    </main>
</Page>
```

No `Meta` needed — `Page` handles `<title>`, `<meta name="description">`, and the OG/Twitter equivalents.

**Page with full social metadata:**

```parsley
<Page lang="en" title="My Blog Post" description="A deep dive into patterns" head={
    <Meta 
        image="/og/functional-patterns.png"
        url="https://example.com/posts/functional-patterns"
        type="article"
        author="Sam Phillips"
        published={post.publishedAt}
        twitter="@samphillips"
    />
}>
    <article>
        (post.content)
    </article>
</Page>
```

**Page with Meta plus additional head content:**

```parsley
<Page lang="en" title="Dashboard" head={
    <>
        <Meta noIndex={true}/>
        <link rel="preconnect" href="https://api.example.com"/>
        <script src="/js/charts.js" defer/>
    </>
}>
    <main id="main">"Dashboard content"</main>
</Page>
```

### 2.4 Meta Component Props

| Prop | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `image` | string | No | — | Open Graph/Twitter image URL |
| `url` | string | No | — | Canonical URL (also used for og:url) |
| `type` | string | No | `"website"` | og:type — use `"article"` for blog posts |
| `author` | string | No | — | Author name (for articles) |
| `published` | datetime | No | — | Published date (for articles) |
| `modified` | datetime | No | — | Modified date (for articles) |
| `twitter` | string | No | — | Twitter handle (with @) for twitter:site and twitter:creator |
| `favicon` | string | No | `"/favicon.ico"` | Favicon path |
| `faviconSvg` | string | No | `"/favicon.svg"` | SVG favicon path |
| `appleTouchIcon` | string | No | `"/apple-touch-icon.png"` | Apple touch icon path |
| `noIndex` | boolean | No | `false` | Add `<meta name="robots" content="noindex, nofollow">` |
| `contents` | any | No | — | Additional meta/link tags to include |

### 2.5 Output Structure

`Meta` outputs only tags, no wrapper:

```html
<!-- Robots (if noIndex) -->
<meta name="robots" content="noindex, nofollow"/>

<!-- Author -->
<meta name="author" content="..."/>

<!-- Canonical URL -->
<link rel="canonical" href="..."/>
<meta property="og:url" content="..."/>

<!-- Open Graph -->
<meta property="og:type" content="article"/>
<meta property="og:image" content="..."/>
<meta property="article:published_time" content="..."/>
<meta property="article:modified_time" content="..."/>
<meta property="article:author" content="..."/>

<!-- Twitter Card -->
<meta name="twitter:card" content="summary_large_image"/>
<meta name="twitter:site" content="@..."/>
<meta name="twitter:creator" content="@..."/>
<meta name="twitter:image" content="..."/>

<!-- Favicons -->
<link rel="icon" href="/favicon.ico" sizes="any"/>
<link rel="icon" href="/favicon.svg" type="image/svg+xml"/>
<link rel="apple-touch-icon" href="/apple-touch-icon.png"/>

<!-- Additional content -->
...
```

**Not output by Meta** (handled by Page):
- `og:title`, `og:description`
- `twitter:title`, `twitter:description`

---

## 3. Implementation

### 3.1 New Meta Component

**File:** `server/prelude/components/meta.pars`

```parsley
// Meta - SEO and social media metadata tags
// 
// Composes with Page's head prop to add social sharing metadata.
// Note: og:title, og:description, twitter:title, twitter:description
// are output by Page from its title/description props.
//
// Usage:
//   <Page title="My Post" description="About things" head={
//       <Meta 
//           image="/og.png"
//           url="https://example.com/post"
//           type="article"
//           published={post.date}
//       />
//   }>
//       "content"
//   </Page>

export Meta = fn({
    image, url, type, author, published, modified,
    twitter, favicon, faviconSvg, appleTouchIcon, noIndex, contents
}) {
    let ogType = type ?? "website"
    
    // Robots directive
    if (noIndex) {
        <meta name="robots" content="noindex, nofollow"/>
    }
    
    // Author
    if (author) {
        <meta name="author" content={author}/>
    }
    
    // Canonical URL
    if (url) {
        <link rel="canonical" href={url}/>
        <meta property="og:url" content={url}/>
    }
    
    // Open Graph type
    <meta property="og:type" content={ogType}/>
    
    // Open Graph image
    if (image) {
        <meta property="og:image" content={image}/>
        <meta name="twitter:image" content={image}/>
    }
    
    // Twitter Card type
    <meta name="twitter:card" content={if (image) "summary_large_image" else "summary"}/>
    
    // Twitter handle
    if (twitter) {
        <meta name="twitter:site" content={twitter}/>
        <meta name="twitter:creator" content={twitter}/>
    }
    
    // Article-specific metadata
    if (ogType == "article") {
        if (published) {
            <meta property="article:published_time" content={published.iso}/>
        }
        if (modified) {
            <meta property="article:modified_time" content={modified.iso}/>
        }
        if (author) {
            <meta property="article:author" content={author}/>
        }
    }
    
    // Favicons
    <link rel="icon" href={favicon ?? "/favicon.ico"} sizes="any"/>
    <link rel="icon" href={faviconSvg ?? "/favicon.svg"} type="image/svg+xml"/>
    <link rel="apple-touch-icon" href={appleTouchIcon ?? "/apple-touch-icon.png"}/>
    
    // Additional content
    if (contents) {
        (contents)
    }
}

// Deprecated alias for backward compatibility
export Head = Meta
```

### 3.2 Updated Page Component

**File:** `server/prelude/components/page.pars`

```parsley
// Page - Complete HTML document wrapper
// Generates <!DOCTYPE html>, <html>, <head>, <body> with SkipLink and asset bundles
// Compatible with Pico CSS
//
// Automatically outputs Open Graph and Twitter Card metadata from title/description.
// Use Meta component in head prop for additional social metadata (image, author, etc.)
//
// Usage:
//   <Page lang="en" title="My Site" description="About my site">
//       <main id="main">Content here</main>
//   </Page>
//
// With Meta for social metadata:
//   <Page lang="en" title="Blog Post" description="A great post" head={
//       <Meta image="/og.png" type="article" published={post.date}/>
//   }>
//       <article>(post.content)</article>
//   </Page>

export Page = fn({lang, title, description, class, id, head, noBasilJS, contents}) {
    if (title == null || title == "") {
        fail("Page requires a title")
    }
    
    "<!DOCTYPE html>\n"
    <html lang={lang ?? "en"}>
    <head>
        <meta charset="UTF-8"/>
        <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
        <title>title</title>
        
        // Basic meta description
        if (description) {
            <meta name="description" content={description}/>
        }
        
        // Open Graph title/description
        <meta property="og:title" content={title}/>
        if (description) {
            <meta property="og:description" content={description}/>
        }
        
        // Twitter title/description
        <meta name="twitter:title" content={title}/>
        if (description) {
            <meta name="twitter:description" content={description}/>
        }
        
        // CSS bundle
        <CSS/>
        
        // Additional head content (Meta component, custom tags, etc.)
        if (head) {
            (head)
        }
    </head>
    <body class={class} id={id}>
        <SkipLink/>
        (contents)
        <Javascript/>
        if (!noBasilJS) {
            <BasilJS/>
        }
    </body>
    </html>
}
```

### 3.3 Migration Steps

1. Create `meta.pars` with new `Meta` component
2. Add `Head` as alias in `meta.pars` for backward compatibility
3. Update `page.pars`:
   - Add title validation with `fail()`
   - Add og:title, og:description, twitter:title, twitter:description output
   - Update comments to document new behavior
4. Delete old `head.pars`
5. Update prelude exports to include `Meta` and keep `Head` alias
6. Update documentation and examples

---

## 4. Validation

### 4.1 Required Props

Following the pattern established in FEAT-143 for `Iframe` and `Abbr`:

- **Page:** `title` is required — fails with clear error if missing
- **Meta:** No required props — all metadata is optional

### 4.2 Parsley Syntax Checklist

Per FEAT-143 learnings, verify:

- [ ] String concatenation uses `+` not `++`
- [ ] Conditionals use concise form: `if (cond) expr else expr`
- [ ] No `{...attrs}` — use `...attrs` for spread
- [ ] Validation uses `fail("message")` not `throw`
- [ ] All files pass `pars --check`

---

## 5. Testing

### 5.1 Test Cases

1. **Page without Meta** — Verify og:title and twitter:title are output from Page.title
2. **Page with description** — Verify og:description and twitter:description are output
3. **Page without title** — Verify `fail()` error is raised
4. **Page with Meta** — Verify Meta outputs image, url, type, favicons correctly
5. **Article type** — Verify article:published_time, article:author are output
6. **noIndex** — Verify robots meta tag is output
7. **Head alias** — Verify `<Head>` still works (deprecated but functional)
8. **No duplicate tags** — Verify og:title isn't duplicated when using Meta
9. **Datetime handling** — Verify published/modified dates use `.iso` format

### 5.2 Verification Script Addition

Add to `scripts/verify-prelude.sh`:

```bash
# Test Page outputs og:title
check_expr "Page og:title output" \
    '... Page component test ...' \
    'og:title'

# Test Meta image output
check_expr "Meta image output" \
    '... Meta component test ...' \
    'og:image'
```

---

## 6. Documentation Updates

### 6.1 Files to Update

1. **Prelude guide** — Document `Meta` component and composition with `Page`
2. **Component reference** — Add `Meta`, update `Page`, deprecate `Head`
3. **Examples** — Update any examples using `Head` to use the new pattern

### 6.2 Deprecation Notice

Add to `Head` documentation:

> **Deprecated:** Use `<Meta>` inside `<Page>`'s `head` prop instead.
> 
> ```parsley
> // Old (deprecated):
> <Head title="..." description="..." image="..."/>
> 
> // New:
> <Page title="..." description="..." head={
>     <Meta image="..."/>
> }>
>     "content"
> </Page>
> ```

---

## 7. Timeline

| Task | Effort | Dependencies |
|------|--------|--------------|
| Create `meta.pars` | 30 min | None |
| Update `page.pars` | 30 min | None |
| Delete `head.pars`, update exports | 15 min | After above |
| Add to verification script | 15 min | After implementation |
| Add tests | 45 min | After implementation |
| Update documentation | 1 hour | After tests pass |

**Total: ~3 hours**

---

## 8. Changes from Original Design

This revision incorporates lessons from FEAT-143:

| Area | Original | Revised |
|------|----------|---------|
| String concat | Used `++` | Uses `+` |
| Conditionals | `if (x) { y } else { z }` | `if (x) y else z` |
| Validation | None | `fail()` for required title |
| Testing | Manual | Integrated with verify-prelude.sh |

---

## 9. References

- **Spec:** `work/specs/FEAT-142.md`
- **FEAT-143 learnings:** Parsley syntax corrections, validation patterns
- **Parsley manual:** `docs/parsley/manual/fundamentals/`
- **Prelude review:** `work/reports/STANDARD-PRELUDE-REVIEW.md` §7.4