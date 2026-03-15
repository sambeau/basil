---
id: PLAN-123
feature: FEAT-142
title: "Implementation Plan: Meta Component and Page Restructure"
status: ready
created: 2026-03-15
updated: 2026-03-15
---

# Implementation Plan: FEAT-142

## Overview

Restructure the prelude's `Head` and `Page` components to fix their broken composition model. Create a new `Meta` component for SEO/social metadata that composes cleanly inside `Page`'s `head` prop.

## Current Status

**Ready for implementation.**

| Phase | Status | Notes |
|-------|--------|-------|
| Phase 1: Create Meta | 🟡 Ready | New component |
| Phase 2: Update Page | 🟡 Ready | Add OG/Twitter output, validation |
| Phase 3: Cleanup | 🟡 Ready | Delete Head, update exports |
| Phase 4: Testing | 🟡 Ready | Verification and tests |
| Phase 5: Documentation | 🟡 Ready | Update docs, add deprecation notice |

## Prerequisites

- [x] Spec approved: `work/specs/FEAT-142.md`
- [x] Design document complete: `work/design/DESIGN-prelude-meta-component.md`
- [x] FEAT-143 complete (Parsley correctness patterns established)

---

## Phase 1: Create Meta Component

### Task 1.1: Create meta.pars
**Status:** 🟡 Ready  
**File:** `server/prelude/components/meta.pars` (new)  
**Estimated effort:** 30 min

Create the new `Meta` component with correct Parsley syntax:

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

**Verification:**
```bash
pars --check server/prelude/components/meta.pars
```

---

## Phase 2: Update Page Component

### Task 2.1: Update page.pars
**Status:** 🟡 Ready  
**File:** `server/prelude/components/page.pars`  
**Estimated effort:** 30 min

Update `Page` to:
1. Validate that `title` is provided using `fail()`
2. Output `og:title` and `twitter:title` from `title` prop
3. Output `og:description` and `twitter:description` from `description` prop

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

**Verification:**
```bash
pars --check server/prelude/components/page.pars
```

---

## Phase 3: Cleanup

### Task 3.1: Delete head.pars
**Status:** 🟡 Ready  
**File:** `server/prelude/components/head.pars` (delete)  
**Estimated effort:** 5 min

Delete the old `head.pars` file. Its functionality is now split between:
- `page.pars` — document structure, basic OG/Twitter
- `meta.pars` — additional SEO/social metadata

**Command:**
```bash
rm server/prelude/components/head.pars
```

### Task 3.2: Update prelude exports
**Status:** 🟡 Ready  
**File:** `server/prelude/prelude.go` (or equivalent export file)  
**Estimated effort:** 10 min

Update the prelude to:
1. Export `Meta` from `meta.pars`
2. Export `Head` as alias (from `meta.pars`) for backward compatibility
3. Remove old `Head` export that pointed to `head.pars`

**Verification:**
```bash
go build ./...
```

---

## Phase 4: Testing

### Task 4.1: Syntax verification
**Status:** 🟡 Ready  
**Estimated effort:** 5 min

Verify all component files pass syntax check:

```bash
for f in server/prelude/components/*.pars; do
    pars --check "$f" || echo "FAIL: $f"
done
```

### Task 4.2: Update verify-prelude.sh
**Status:** 🟡 Ready  
**File:** `scripts/verify-prelude.sh`  
**Estimated effort:** 15 min

Add tests for Meta and updated Page:

```bash
# Test Page outputs og:title (requires server context, may need integration test)
echo -n "Page title validation: "
# ... test that Page without title fails

# Test Meta outputs og:image
echo -n "Meta image: "
# ... test Meta component
```

### Task 4.3: Run full test suite
**Status:** 🟡 Ready  
**Estimated effort:** 5 min

```bash
go test ./...
./scripts/verify-prelude.sh
```

### Task 4.4: Manual smoke test
**Status:** 🟡 Ready  
**Estimated effort:** 15 min

Test in a real Basil app:

1. Simple Page (no Meta):
```parsley
<Page lang="en" title="Test" description="A test page">
    <main id="main">"Hello"</main>
</Page>
```
Verify output contains: `og:title`, `og:description`, `twitter:title`, `twitter:description`

2. Page with Meta:
```parsley
<Page lang="en" title="Blog Post" description="A post" head={
    <Meta image="/og.png" type="article" author="Test"/>
}>
    <main id="main">"Content"</main>
</Page>
```
Verify output contains: article metadata, image tags, favicons

3. Page without title:
```parsley
<Page lang="en">
    <main id="main">"No title"</main>
</Page>
```
Verify: `fail()` error is raised

---

## Phase 5: Documentation

### Task 5.1: Update component documentation
**Status:** 🟡 Ready  
**Estimated effort:** 30 min

Update docs to document:
- `Meta` component and its props
- `Page` now outputs OG/Twitter metadata automatically
- Composition pattern (`Meta` inside `Page.head`)

### Task 5.2: Add deprecation notice for Head
**Status:** 🟡 Ready  
**Estimated effort:** 15 min

Add to documentation:

> **Deprecated:** `Head` is deprecated. Use `<Meta>` inside `<Page>`'s `head` prop instead.
> 
> ```parsley
> // Old (deprecated):
> <Head title="..." description="..." image="..."/>
> 
> // New:
> <Page title="..." description="..." head={
>     <Meta image="..."/>
> }>
>     <main id="main">"content"</main>
> </Page>
> ```

### Task 5.3: Update examples
**Status:** 🟡 Ready  
**Estimated effort:** 15 min

Search for any examples using `<Head>` and update to new pattern.

```bash
grep -r "<Head" examples/ docs/
```

---

## Validation Checklist

**Phase 1-2 (Implementation):**
- [ ] `meta.pars` created with correct Parsley syntax
- [ ] `page.pars` updated with title validation and OG/Twitter output
- [ ] All files pass `pars --check`

**Phase 3 (Cleanup):**
- [ ] `head.pars` deleted
- [ ] Prelude exports updated
- [ ] Build succeeds: `go build ./...`

**Phase 4 (Testing):**
- [ ] `go test ./...` passes
- [ ] `scripts/verify-prelude.sh` passes
- [ ] Manual smoke test passes

**Phase 5 (Documentation):**
- [ ] `Meta` component documented
- [ ] `Page` OG/Twitter behavior documented
- [ ] Deprecation notice added for `Head`
- [ ] Examples updated

---

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-03-15 | Plan created | ✅ | Based on revised design |

---

## Deferred Items (Post-FEAT-142)

Add to `work/BACKLOG.md` after completion:

- **JSON-LD structured data** — Add support for schema.org JSON-LD in Meta
- **Multiple images** — Support array of images for og:image
- **Locale support** — Add og:locale and og:locale:alternate

---

## References

- **Spec:** `work/specs/FEAT-142.md`
- **Design:** `work/design/DESIGN-prelude-meta-component.md`
- **FEAT-143:** Parsley correctness patterns (spread, for-loops, string concat)
- **Prelude review:** `work/reports/STANDARD-PRELUDE-REVIEW.md` §7.4