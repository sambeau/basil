---
id: FEAT-063
title: "Site-Wide CSS/JS Auto-Bundle"
status: draft
priority: medium
created: 2025-12-11
author: "@human + AI"
---

# FEAT-063: Site-Wide CSS/JS Auto-Bundle

## Summary

Basil automatically discovers and bundles all `.css` and `.js` files from the `handlers/` tree into single served files (`/__site.css` and `/__site.js`). This allows developers to co-locate component styles and scripts with their Parsley code while still benefiting from browser caching and clean HTML output.

## User Story

As a developer, I want my component CSS and JavaScript files to be automatically bundled and served so that I can organize my code by feature/component without littering my HTML with multiple `<link>` and `<script>` tags.

## Acceptance Criteria

### Core Functionality
- [ ] All `.css` files under `handlers/` are discovered and concatenated
- [ ] All `.js` files under `handlers/` are discovered and concatenated
- [ ] Files in `public/` are excluded (served separately for third-party libs)
- [ ] Hidden files (starting with `.`) are excluded
- [ ] Bundle served at `/__site.css?v={hash}` and `/__site.js?v={hash}`
- [ ] Hash computed from concatenated content for cache busting

### File Order
- [ ] Depth-first traversal of directory tree
- [ ] Alphabetical ordering within each directory level
- [ ] Deterministic order across restarts

### Tags
- [ ] `<Css/>` tag emits `<link rel="stylesheet" href="/__site.css?v={hash}">`
- [ ] `<Script/>` tag emits `<script src="/__site.js?v={hash}"></script>`

### Dev Mode
- [ ] Source file comments included in output (showing which file each section came from)
- [ ] Bundle regenerated on any `.css`/`.js` file change
- [ ] Hash updated on regeneration

### Production Mode
- [ ] No source comments (just concatenated content)
- [ ] Bundle generated once at startup
- [ ] SIGHUP triggers regeneration
- [ ] Long cache headers (`Cache-Control: public, max-age=31536000`)

## Design Decisions

- **Query string for cache busting**: Simpler than filename hashing, still effective
- **Exclude `public/`**: Third-party libraries should be explicitly linked before the bundle, giving developers control over load order
- **Include everything else**: No underscore-prefix exclusion or manifest files—simpler mental model
- **Site-wide bundle**: Simpler than per-route bundles; most sites share styles across pages anyway
- **Depth-first alphabetical order**: Predictable, allows developers to control cascade by folder structure

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Specification

### Bundle URLs

```
/__site.css?v=a1b2c3d4
/__site.js?v=e5f6g7h8
```

The hash is the first 8 characters of SHA-256 of the concatenated content.

### Discovery Algorithm

```
1. Walk handlers/ directory recursively (depth-first)
2. For each directory (alphabetically):
   a. Collect .css files (alphabetically)
   b. Collect .js files (alphabetically)
   c. Recurse into subdirectories (alphabetically)
3. Skip hidden files (.*) and public/ folder
4. Store ordered lists: cssFiles, jsFiles
5. Compute hash from concatenated content
```

### Example File Order

Given this structure:
```
handlers/
├── base.css              # 1
├── utils.js              # JS-1
├── components/
│   ├── button/
│   │   ├── button.css    # 2
│   │   └── button.js     # JS-2
│   └── card/
│       └── card.css      # 3
├── pages/
│   └── about/
│       └── about.css     # 4
└── parts/
    └── header/
        └── header.css    # 5
```

CSS order: base.css → button.css → card.css → about.css → header.css
JS order: utils.js → button.js

### Tag Output

**`<Css/>`**
```html
<link rel="stylesheet" href="/__site.css?v=a1b2c3d4">
```

**`<Script/>`**
```html
<script src="/__site.js?v=e5f6g7h8"></script>
```

### Dev Mode Output (CSS example)

```css
/* ══════════════════════════════════════════════════════════════
   handlers/base.css
   ══════════════════════════════════════════════════════════════ */
body {
    margin: 0;
    font-family: system-ui, sans-serif;
}

/* ══════════════════════════════════════════════════════════════
   handlers/components/button/button.css
   ══════════════════════════════════════════════════════════════ */
.button {
    padding: 0.5rem 1rem;
    border-radius: 4px;
}
```

### Production Mode Output

```css
body {
    margin: 0;
    font-family: system-ui, sans-serif;
}
.button {
    padding: 0.5rem 1rem;
    border-radius: 4px;
}
```

### HTTP Response Headers

```
Content-Type: text/css; charset=utf-8  (or application/javascript)
Cache-Control: public, max-age=31536000
ETag: "a1b2c3d4"
```

### Typical Page Usage

```parsley
<html>
<head>
    <meta charset="utf-8"/>
    <title>{title}</title>
    <link rel="stylesheet" href="/static/reset.css"/>      // Third-party from public/ (via static: config)
    <link rel="stylesheet" href="/static/vendor/lib.css"/> // Third-party from public/
    <Css/>                                                  // Your bundled styles (last)
</head>
<body>
    {children}
    
    <script src="/static/vendor/chart.js"></script>  // Third-party from public/
    <Script/>                                         // Your bundled scripts (last)
</body>
</html>
```

Note: The `/static/` prefix depends on your `basil.yaml` configuration:
```yaml
static:
  - path: /static/
    root: ./public
```

### Integration with `<Page/>`

The `<Page/>` component in `std/basil` could optionally auto-include these tags:

```parsley
// std/basil Page component
export Page = fn({title, children}) {
    <html>
    <head>
        <meta charset="utf-8"/>
        <title>{title}</title>
        <Css/>
    </head>
    <body>
        {children}
        <Script/>
    </body>
    </html>
}
```

Or leave it explicit for developer control.

### Affected Components

- `server/server.go` — Add AssetBundle struct, discovery on startup
- `server/handler.go` — Serve `/__site.css` and `/__site.js` routes
- `server/watcher.go` — Watch `.css`/`.js` files, trigger bundle regeneration
- `pkg/parsley/evaluator/evaluator.go` — Implement `<Css/>` and `<Script/>` tags

### Edge Cases

1. **No CSS/JS files**: Tags emit nothing (empty href would be invalid)
2. **Empty files**: Include in bundle (may be placeholders)
3. **Binary files with .css/.js extension**: Include (garbage in, garbage out)
4. **Symlinks**: Follow symlinks (standard Go filepath.Walk behavior)
5. **Encoding**: All files assumed UTF-8
6. **Very large bundles**: No size limit (developer responsibility)

### Cache Invalidation

**Dev Mode:**
1. File watcher detects `.css` or `.js` change in `handlers/`
2. Rebuild asset bundle (re-walk, re-concatenate, re-hash)
3. LiveReload triggers page refresh (existing mechanism)
4. New hash in URL causes browser to fetch fresh bundle

**Production Mode:**
1. Bundle built once at startup
2. SIGHUP signal triggers rebuild (same as script cache clear)
3. New requests get new hash

## Future Considerations

- **Minification**: Could add optional CSS/JS minification in production
- **Source maps**: Could generate source maps for debugging
- **Per-route bundles**: Could scope bundles to route subtrees if needed
- **Import tracking**: Could analyze Parsley imports to only include used CSS/JS

## Related

- Depends on: None
- Related: `docs/design/DESIGN-asset-bundling.md` (discussion document)
