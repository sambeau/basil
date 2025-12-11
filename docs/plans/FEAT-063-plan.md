---
id: PLAN-039
feature: FEAT-063
title: "Implementation Plan for Site-Wide CSS/JS Auto-Bundle"
status: draft
created: 2025-12-11
---

# Implementation Plan: FEAT-063

## Overview

Implement automatic discovery and bundling of CSS and JS files from the `handlers/` directory tree. Files are concatenated in depth-first alphabetical order, served at `/__site.css` and `/__site.js` with content-hash query strings for cache busting. New `<Css/>` and `<Script/>` tags emit the appropriate `<link>` and `<script>` elements.

## Prerequisites

- [x] Spec written (FEAT-063)
- [x] Understand existing asset registry (`server/assets.go`)
- [x] Understand watcher mechanism (`server/watcher.go`)
- [x] Understand tag evaluation (`pkg/parsley/evaluator/evaluator.go`)

## Tasks

### Task 1: Create Bundle Type and Discovery in `server/bundle.go`

**Files:** `server/bundle.go` (new)  
**Estimated effort:** Medium

Steps:
1. Create `AssetBundle` struct:
   ```go
   type AssetBundle struct {
       mu       sync.RWMutex
       cssFiles []string  // ordered file paths
       jsFiles  []string  // ordered file paths
       cssHash  string    // first 8 chars of SHA-256
       jsHash   string
       cssContent []byte  // concatenated content
       jsContent  []byte
       devMode  bool
       handlersDir string
   }
   ```
2. Implement `NewAssetBundle(handlersDir string, devMode bool) *AssetBundle`
3. Implement `(b *AssetBundle) Rebuild() error`:
   - Walk `handlersDir` depth-first, alphabetically
   - Skip hidden files (`.*`) and `public/` folder
   - Collect `.css` and `.js` files in order
   - Concatenate content (with dev mode comments)
   - Compute hashes
4. Implement `(b *AssetBundle) CSSUrl() string` → `/__site.css?v={hash}` (empty string if no CSS)
5. Implement `(b *AssetBundle) JSUrl() string` → `/__site.js?v={hash}` (empty string if no JS)
6. Implement `(b *AssetBundle) ServeCSS(w, r)` and `(b *AssetBundle) ServeJS(w, r)`

Tests:
- Discovery finds CSS/JS files in correct order
- Hidden files are excluded
- Hash changes when content changes
- Dev mode includes source comments
- Production mode excludes comments

---

### Task 2: Integrate Bundle into Server

**Files:** `server/server.go`  
**Estimated effort:** Small

Steps:
1. Add `assetBundle *AssetBundle` field to `Server` struct
2. Initialize bundle in `New()` after config is loaded:
   ```go
   s.assetBundle = NewAssetBundle(handlersDir, cfg.Server.Dev)
   if err := s.assetBundle.Rebuild(); err != nil {
       return nil, fmt.Errorf("building asset bundle: %w", err)
   }
   ```
3. Pass bundle reference to evaluator environment (for `<Css/>` and `<Script/>` tags)

Tests:
- Server starts with bundle initialized
- Bundle is accessible from handler context

---

### Task 3: Add HTTP Routes for Bundles

**Files:** `server/handler.go`  
**Estimated effort:** Small

Steps:
1. Register routes in `setupRoutes()`:
   ```go
   s.mux.HandleFunc("/__site.css", s.serveSiteCSS)
   s.mux.HandleFunc("/__site.js", s.serveSiteJS)
   ```
2. Implement handler methods that delegate to `assetBundle.ServeCSS/ServeJS`
3. Set appropriate headers:
   - `Content-Type: text/css; charset=utf-8` or `application/javascript; charset=utf-8`
   - `Cache-Control: public, max-age=31536000` (production)
   - `Cache-Control: no-cache` (dev mode)
   - `ETag: "{hash}"`

Tests:
- Routes respond with correct content type
- ETag matches hash
- Cache headers correct for dev vs production

---

### Task 4: Implement `<Css/>` and `<Script/>` Tags

**Files:** `pkg/parsley/evaluator/evaluator.go`  
**Estimated effort:** Medium

Steps:
1. Add `AssetBundle` interface to `Environment`:
   ```go
   type AssetBundler interface {
       CSSUrl() string
       JSUrl() string
   }
   ```
2. Add `AssetBundle AssetBundler` field to `Environment` struct
3. In `evalCustomTag()`, add special handling for `Css` and `Script`:
   ```go
   case "Css":
       if env.AssetBundle == nil || env.AssetBundle.CSSUrl() == "" {
           return &String{Value: ""}  // No CSS bundle
       }
       return &String{Value: fmt.Sprintf(`<link rel="stylesheet" href="%s">`, env.AssetBundle.CSSUrl())}
   case "Script":
       if env.AssetBundle == nil || env.AssetBundle.JSUrl() == "" {
           return &String{Value: ""}  // No JS bundle
       }
       return &String{Value: fmt.Sprintf(`<script src="%s"></script>`, env.AssetBundle.JSUrl())}
   ```
4. Copy `AssetBundle` to child environments in `NewChildEnvironment()`

Tests:
- `<Css/>` emits correct `<link>` tag with hash
- `<Script/>` emits correct `<script>` tag with hash
- Tags emit nothing when bundle is empty/nil

---

### Task 5: Add Bundle to Handler Context

**Files:** `server/handler.go`  
**Estimated effort:** Small

Steps:
1. In `parsleyHandler.createEnvironment()`, set `env.AssetBundle = s.assetBundle`
2. Ensure bundle is passed through all handler types (route handlers, site handlers)

Tests:
- `<Css/>` works in route handlers
- `<Css/>` works in site (filesystem) handlers

---

### Task 6: Add Watcher Support for Bundle Regeneration

**Files:** `server/watcher.go`  
**Estimated effort:** Small

Steps:
1. In `handleFileChange()`, when `.css` or `.js` changes:
   ```go
   case ".css", ".js":
       // Check if file is under handlers/ (not static)
       if w.isHandlerFile(path) {
           w.logInfo("bundle asset changed: %s", path)
           if err := w.server.assetBundle.Rebuild(); err != nil {
               w.logError("failed to rebuild bundle: %v", err)
           }
       } else {
           w.logInfo("static file changed: %s", path)
       }
   ```
2. Ensure live reload is triggered (already happens via `changeSeq`)

Tests:
- Modifying CSS in handlers/ triggers bundle rebuild
- Modifying CSS in public/ does not trigger rebuild
- Browser reloads with new bundle hash

---

### Task 7: Add SIGHUP Handler for Production Rebuild

**Files:** `server/server.go`  
**Estimated effort:** Small

Steps:
1. In existing SIGHUP handler (if any), add bundle rebuild:
   ```go
   if s.assetBundle != nil {
       if err := s.assetBundle.Rebuild(); err != nil {
           s.logError("failed to rebuild bundle: %v", err)
       } else {
           s.logInfo("asset bundle rebuilt")
       }
   }
   ```

Tests:
- SIGHUP triggers bundle rebuild
- New requests get new hash

---

### Task 8: Add Unit Tests

**Files:** `server/bundle_test.go` (new)  
**Estimated effort:** Medium

Test cases:
- `TestAssetBundle_Discovery` — files found in correct order
- `TestAssetBundle_DepthFirstOrder` — nested directories processed correctly
- `TestAssetBundle_ExcludesHidden` — dotfiles excluded
- `TestAssetBundle_ExcludesPublic` — public/ folder excluded (if in handlers/)
- `TestAssetBundle_HashComputation` — hash is first 8 chars of SHA-256
- `TestAssetBundle_DevModeComments` — dev mode includes source paths
- `TestAssetBundle_ProductionNoComments` — production omits comments
- `TestAssetBundle_EmptyBundle` — no CSS/JS returns empty URL
- `TestAssetBundle_Rebuild` — content changes update hash

---

### Task 9: Add Integration Tests

**Files:** `pkg/parsley/tests/bundle_tags_test.go` (new)  
**Estimated effort:** Small

Test cases:
- `TestCssTag_EmitsLink` — `<Css/>` produces `<link>` element
- `TestScriptTag_EmitsScript` — `<Script/>` produces `<script>` element
- `TestCssTag_NoBundle` — `<Css/>` with no bundle emits nothing
- `TestCssTag_EmptyBundle` — `<Css/>` with empty bundle emits nothing

---

### Task 10: Update Documentation

**Files:** `docs/parsley/reference.md`, `docs/parsley/CHEATSHEET.md`, `basil.example.yaml`  
**Estimated effort:** Small

Steps:
1. Document `<Css/>` and `<Script/>` tags in reference
2. Add to cheatsheet under "Built-in Tags"
3. Add example in `docs/guide/` showing co-located CSS usage
4. Note in basil.example.yaml about automatic bundling

---

## Validation Checklist

- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] Manual test: Create CSS in handlers/, verify bundle served
- [ ] Manual test: Modify CSS, verify hash changes
- [ ] Manual test: `<Css/>` emits correct `<link>` tag
- [ ] Documentation updated
- [ ] BACKLOG.md updated with deferrals (if any)

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| — | Task 1: Bundle type | ⬜ Not started | — |
| — | Task 2: Server integration | ⬜ Not started | — |
| — | Task 3: HTTP routes | ⬜ Not started | — |
| — | Task 4: Css/Script tags | ⬜ Not started | — |
| — | Task 5: Handler context | ⬜ Not started | — |
| — | Task 6: Watcher support | ⬜ Not started | — |
| — | Task 7: SIGHUP handler | ⬜ Not started | — |
| — | Task 8: Unit tests | ⬜ Not started | — |
| — | Task 9: Integration tests | ⬜ Not started | — |
| — | Task 10: Documentation | ⬜ Not started | — |

## Deferred Items

Items to add to BACKLOG.md after implementation:
- **Minification** — Add optional CSS/JS minification for production bundles
- **Source maps** — Generate source maps for debugging bundled JS
- **Per-route bundles** — Scope bundles to route subtrees for code splitting
- **Import tracking** — Analyze Parsley imports to only include used CSS/JS

## Design Notes

### File Order Algorithm

Depth-first, alphabetical within each level:

```
handlers/
├── base.css              # 1 (root files first, alphabetically)
├── utils.js              # JS-1
├── components/           # then subdirs alphabetically
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

Implementation uses `filepath.WalkDir` with custom sorting.

### Bundle Interface

The `AssetBundler` interface allows the evaluator to remain independent of server implementation:

```go
type AssetBundler interface {
    CSSUrl() string  // Returns "" if no CSS files
    JSUrl() string   // Returns "" if no JS files
}
```

This also enables mock implementations for testing.
