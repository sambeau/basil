---
id: FEAT-142
title: "Meta Component and Page Restructure"
status: approved
priority: high
created: 2026-06-15
updated: 2026-03-15
author: "@human"
---

# FEAT-142: Meta Component and Page Restructure

## Summary

Restructure the prelude's `Head` and `Page` components to fix their broken composition model. Rename `Head` to `Meta`, remove its `<head>` wrapper so it outputs only meta/link tags, and update `Page` to output basic Open Graph and Twitter metadata from its existing `title` and `description` props. This enables clean composition where `Page` owns document structure and `Meta` provides optional SEO/social enhancement.

## User Story

As a developer building web pages, I want to add Open Graph and Twitter Card metadata without duplicating my title and description, so that social sharing works correctly with minimal boilerplate.

## Problem Statement

The current `Head` and `Page` components have a structural problem:

1. `Head` generates its own `<head>` wrapper tag
2. `Page` also generates a `<head>` section
3. Therefore, `Head` cannot be nested inside `Page` — they're mutually exclusive
4. Both components include `<CSS/>`, risking duplicate asset bundles
5. The relationship between them is confusing to users

## Acceptance Criteria

### Meta Component (New)

- [ ] New `Meta` component created in `server/prelude/components/meta.pars`
- [ ] `Meta` outputs meta/link tags only — no `<head>` wrapper
- [ ] `Meta` supports all SEO/social props: `image`, `url`, `type`, `author`, `published`, `modified`, `twitter`, `favicon`, `faviconSvg`, `appleTouchIcon`, `noIndex`
- [ ] `Meta` does NOT output `og:title`, `og:description`, `twitter:title`, `twitter:description` (these come from `Page`)
- [ ] `Meta` outputs `og:type` (default: `"website"`)
- [ ] `Meta` outputs favicon links with sensible defaults
- [ ] `Meta` handles `datetime` objects for `published`/`modified` using `.iso` property
- [ ] `Meta` supports `contents` prop for additional arbitrary tags

### Page Component Updates

- [ ] `Page` validates that `title` is provided using `fail()`
- [ ] `Page` outputs `<meta property="og:title">` from its `title` prop
- [ ] `Page` outputs `<meta name="twitter:title">` from its `title` prop
- [ ] `Page` outputs `<meta property="og:description">` when `description` prop is provided
- [ ] `Page` outputs `<meta name="twitter:description">` when `description` prop is provided
- [ ] `Page` continues to output `<title>` and `<meta name="description">` as before
- [ ] `Page`'s `head` prop renders after the new OG/Twitter tags (allowing `Meta` to add more)

### Backward Compatibility

- [ ] `Head` exported as deprecated alias for `Meta`
- [ ] Existing `<Head>` usage continues to work (but outputs at wrong level if used standalone)

### Parsley Correctness (from FEAT-143)

- [ ] All code uses `+` for string concatenation (not `++`)
- [ ] All single-expression conditionals use concise form: `if (cond) expr else expr`
- [ ] Spread syntax uses `...attrs` (not `{...attrs}`)
- [ ] Required prop validation uses `fail("message")`
- [ ] All files pass `pars --check`

### Documentation

- [ ] `Meta` component documented in prelude guide
- [ ] `Page` documentation updated to show `og:title`/`twitter:title` auto-output
- [ ] Migration guide from `<Head>` to `<Meta>` pattern
- [ ] Deprecation notice added to `Head` documentation

## Design Decisions

1. **Page outputs OG/Twitter title/description**: Rather than requiring users to duplicate their title in both `Page` and `Meta`, `Page` now outputs the Open Graph and Twitter equivalents automatically. `Meta` focuses on the *additional* metadata (image, url, type, author).

2. **Meta has no wrapper**: `Meta` outputs raw tags so it can be composed inside `Page`'s `head` prop alongside other content like `<link>`, `<script>`, etc.

3. **Favicons in Meta, not Page**: Favicon links are SEO/branding concerns that belong with `Meta`. Users who don't need social metadata also typically don't need custom favicons.

4. **Required title validation**: Following the FEAT-143 pattern for `Iframe` and `Abbr`, `Page` uses `fail()` to enforce that `title` is provided.

5. **Deprecated alias**: `Head` is kept as an alias for backward compatibility, though using it standalone (not inside `Page.head`) will produce incorrect output.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Design Document

See `work/design/DESIGN-prelude-meta-component.md` for full design rationale and implementation code.

### Affected Components

| File | Change |
|------|--------|
| `server/prelude/components/meta.pars` | New file — `Meta` component |
| `server/prelude/components/page.pars` | Update — add title validation, add OG/Twitter output |
| `server/prelude/components/head.pars` | Delete — replaced by `meta.pars` |
| Prelude exports | Add `Meta`, keep `Head` as alias |

### Dependencies

- Depends on: FEAT-143 (Parsley correctness patterns)
- Blocks: None
- Related: FEAT-051 (Standard Prelude), FEAT-058 (HTML Components in Prelude)

### New Meta Component

```parsley
// server/prelude/components/meta.pars

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
        contents
    }
}

// Deprecated alias for backward compatibility
export Head = Meta
```

### Updated Page Component

```parsley
// server/prelude/components/page.pars

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
//       <article>post.content</article>
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
        contents
        <Javascript/>
        if (!noBasilJS) {
            <BasilJS/>
        }
    </body>
    </html>
}
```

### Meta Component Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `image` | string | — | Open Graph/Twitter image URL |
| `url` | string | — | Canonical URL (also og:url) |
| `type` | string | `"website"` | og:type — use `"article"` for blog posts |
| `author` | string | — | Author name |
| `published` | datetime | — | Published date (for articles) |
| `modified` | datetime | — | Modified date (for articles) |
| `twitter` | string | — | Twitter handle (with @) |
| `favicon` | string | `/favicon.ico` | Favicon path |
| `faviconSvg` | string | `/favicon.svg` | SVG favicon path |
| `appleTouchIcon` | string | `/apple-touch-icon.png` | Apple touch icon path |
| `noIndex` | boolean | `false` | Add robots noindex directive |
| `contents` | any | — | Additional meta/link tags |

### Page Component Props (Updated)

| Prop | Type | Default | Description | Change |
|------|------|---------|-------------|--------|
| `lang` | string | `"en"` | Language code | No change |
| `title` | string | **required** | Page title | Now validated with `fail()`, outputs og:title, twitter:title |
| `description` | string | — | Meta description | Now also outputs og:description, twitter:description |
| `class` | string | — | Body class | No change |
| `id` | string | — | Body id | No change |
| `head` | any | — | Additional head content | No change |
| `noBasilJS` | boolean | `false` | Omit basil.js | No change |
| `contents` | any | — | Body content | No change |

### Usage Examples

**Simple page (common case):**

```parsley
<Page lang="en" title="About Us" description="Learn about our company">
    <main id="main">
        <h1>"About Us"</h1>
        "Content..."
    </main>
</Page>
```

Output includes:
- `<title>About Us</title>`
- `<meta name="description" content="Learn about our company"/>`
- `<meta property="og:title" content="About Us"/>`
- `<meta property="og:description" content="Learn about our company"/>`
- `<meta name="twitter:title" content="About Us"/>`
- `<meta name="twitter:description" content="Learn about our company"/>`

**Blog post with full social metadata:**

```parsley
<Page lang="en" title="My Blog Post" description="A post about things" head={
    <Meta 
        image="/og/post.png"
        url="https://example.com/posts/my-post"
        type="article"
        author="Sam Phillips"
        published={post.publishedAt}
        twitter="@samphillips"
    />
}>
    <article>post.content</article>
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

### Test Cases

1. **Page without Meta** — Verify og:title and twitter:title are output from Page.title
2. **Page with description** — Verify og:description and twitter:description are output
3. **Page without title** — Verify `fail()` error is raised
4. **Page with Meta** — Verify Meta outputs image, url, type, favicons correctly
5. **Article type** — Verify article:published_time, article:author are output
6. **noIndex** — Verify robots meta tag is output
7. **Head alias** — Verify `<Head>` still works (deprecated but functional)
8. **No duplicate tags** — Verify og:title isn't duplicated when using Meta
9. **Datetime handling** — Verify published/modified dates use `.iso` format
10. **Syntax check** — Verify all files pass `pars --check`

### Migration Guide

```parsley
// Old pattern (deprecated):
<Head title="My Page" description="About" image="/og.png"/>

// New pattern:
<Page title="My Page" description="About" head={
    <Meta image="/og.png"/>
}>
    <main id="main">"content"</main>
</Page>
```

### Implementation Steps

1. Create `server/prelude/components/meta.pars` with `Meta` component
2. Add `Head` as deprecated alias in `meta.pars`
3. Update `server/prelude/components/page.pars`:
   - Add title validation with `fail()`
   - Add og:title, og:description, twitter:title, twitter:description output
4. Delete `server/prelude/components/head.pars`
5. Update prelude exports to include `Meta` and `Head` alias
6. Run `pars --check` on all modified files
7. Add tests for all acceptance criteria
8. Update documentation

### Effort Estimate

| Task | Effort |
|------|--------|
| Create `meta.pars` | 30 min |
| Update `page.pars` | 30 min |
| Delete `head.pars`, update exports | 15 min |
| Syntax verification | 10 min |
| Add tests | 45 min |
| Update documentation | 1 hour |
| **Total** | **~3 hours** |

## Related

- Design doc: `work/design/DESIGN-prelude-meta-component.md`
- Standard Prelude Review: `work/reports/STANDARD-PRELUDE-REVIEW.md` §7.4
- FEAT-143: Prelude Parsley correctness patterns
- Parent feature: FEAT-051 (Standard Prelude)
- Related: FEAT-058 (HTML Components in Prelude)