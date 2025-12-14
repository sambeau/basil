---
id: FEAT-063
title: "Site-Wide CSS/JS Auto-Bundle"
status: implemented
priority: medium
created: 2025-12-11
implemented: 2025-12-11
author: "@human + AI"
---

# FEAT-063: Site-Wide CSS/JS Auto-Bundle

## Summary

Basil automatically discovers and bundles all `.css` and `.js` files from the `handlers/` tree into single served files (`/__site.css` and `/__site.js`). This allows developers to co-locate component styles and scripts with their Parsley code while still benefiting from browser caching and clean HTML output.

## User Story

As a developer, I want my component CSS and JavaScript files to be automatically bundled and served so that I can organize my code by feature/component without littering my HTML with multiple `<link>` and `<script>` tags.

## Acceptance Criteria

### Core Functionality
- [x] All `.css` files under `handlers/` are discovered and concatenated
- [x] All `.js` files under `handlers/` are discovered and concatenated
- [x] Files in `public/` are excluded (served separately for third-party libs)
- [x] Hidden files (starting with `.`) are excluded
- [x] Bundle served at `/__site.css?v={hash}` and `/__site.js?v={hash}`
- [x] Hash computed from concatenated content for cache busting

### File Order
- [x] Depth-first traversal of directory tree
- [x] Alphabetical ordering within each directory level
- [x] Deterministic order across restarts

### Tags
- [x] `<CSS/>` tag emits `<link rel="stylesheet" href="/__site.css?v={hash}">`
- [x] `<Javascript/>` tag emits `<script src="/__site.js?v={hash}"></script>`

### Dev Mode
- [x] Source file comments included in output (showing which file each section came from)
- [x] Bundle regenerated on any `.css`/`.js` file change
- [x] Hash updated on regeneration

### Production Mode
- [x] No source comments (just concatenated content)
- [x] Bundle generated once at startup
- [x] SIGHUP triggers regeneration
- [x] Long cache headers (`Cache-Control: public, max-age=31536000`)

## Implementation Notes

**Commit:** a2d1575
**Date:** 2025-12-11

Successfully implemented all acceptance criteria. Key implementation details:

### Files Created
- `server/bundle.go` (265 lines) - AssetBundle type with discovery, concatenation, hashing, HTTP serving
- `server/bundle_test.go` (190 lines) - 7 comprehensive unit tests
- `pkg/parsley/tests/bundle_tags_test.go` (163 lines) - 6 integration tests for tag evaluation

### Files Modified
- `server/server.go` - Added assetBundle field, initialization, route registration
- `pkg/parsley/evaluator/evaluator.go` - AssetBundler interface, <CSS/>/<Javascript/> tag handling
- `server/handler.go` & `server/api.go` - Bundle context injection
- `server/watcher.go` - Bundle rebuild on file changes
- `docs/parsley/reference.md` - Asset Bundle Tags section
- `docs/parsley/CHEATSHEET.md` - Gotchas and usage guide
- `basil.example.yaml` - Documentation comments

### Testing
All 13 tests passing:
- Asset discovery with depth-first alphabetical ordering
- Dev mode source comments
- Production mode without comments
- Hidden file exclusion
- Empty bundle handling
- Hash computation (SHA-256, first 8 chars)
- URL generation with cache-busting
- <CSS/> tag emits link element
- <Javascript/> tag emits script element
- Empty/missing bundle cases
- Template integration

### Design Choices
- **AssetBundler interface**: Prevents circular dependencies between server and evaluator packages
- **Depth-first alphabetical**: Allows predictable CSS cascade control via folder structure
- **ETag support**: Efficient HTTP caching with 304 Not Modified responses
- **File watching integration**: Seamless dev mode experience with live reload

### Known Limitations
- No minification (deferred - could add in future)
- No source maps (deferred - could add in future)
- No per-route bundles (site-wide only)



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
1. Determine handler root directory:
   - Site mode: Parent of the site/ directory (the handler root)
   - Route mode: Common ancestor of all handler file paths
2. Walk handler root directory recursively (depth-first)
3. For each directory (alphabetically):
   a. Collect .css files (alphabetically)
   b. Collect .js files (alphabetically)
   c. Recurse into subdirectories (alphabetically)
4. Skip hidden files (.*) and the configured public directory (from public_dir config)
5. Store ordered lists: cssFiles, jsFiles
6. Compute hash from concatenated content
```

**Note**: In site mode, the entire handler root is scanned, not just the `site/` directory. This allows CSS/JS files to be organized in sibling directories like `components/`, alongside the `site/` handlers.

**Note**: The public directory name is determined from the `public_dir` configuration (e.g., if `public_dir: "./static"`, then `static/` is excluded). This prevents third-party library files from being included in the auto-bundle.

### Example File Order

Given this structure:
```
<handler root>/           # In site mode: parent of site/
├── base.css              # 1
├── utils.js              # JS-1
├── components/
│   ├── button/
│   │   ├── button.css    # 2
│   │   └── button.js     # JS-2
│   └── card/
│       └── card.css      # 3
├── site/                 # Site mode handlers directory
│   └── index.pars
├── pages/
│   └── about/
│       └── about.css     # 4
├── parts/
│   └── header/
│       └── header.css    # 5
└── public/               # Excluded (or whatever name is in public_dir config)
    └── bootstrap.css     # Not included
```

CSS order: base.css → button.css → card.css → about.css → header.css
JS order: utils.js → button.js

### Tag Output

**`<CSS/>`**
```html
<link rel="stylesheet" href="/__site.css?v=a1b2c3d4">
```

**`<Javascript/>`**
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
    <CSS/>                                                  // Your bundled styles (last)
</head>
<body>
    {children}
    
    <script src="/static/vendor/chart.js"></script>  // Third-party from public/
    <Javascript/>                                     // Your bundled scripts (last)
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
        <CSS/>
    </head>
    <body>
        {children}
        <Javascript/>
    </body>
    </html>
}
```

Or leave it explicit for developer control.

### Affected Components

- `server/server.go` — Add AssetBundle struct, discovery on startup
- `server/handler.go` — Serve `/__site.css` and `/__site.js` routes
- `server/watcher.go` — Watch `.css`/`.js` files, trigger bundle regeneration
- `pkg/parsley/evaluator/evaluator.go` — Implement `<CSS/>` and `<Javascript/>` tags

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
