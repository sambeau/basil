---
id: PLAN-033
feature: FEAT-056
title: "Implementation Plan for Prelude Infrastructure"
status: draft
created: 2025-12-09
---

# Implementation Plan: FEAT-056 Prelude Infrastructure

## Overview

Implement the foundational infrastructure for embedding Parsley source files and static assets into the Basil binary. This includes:

1. Embed system using `//go:embed`
2. Startup parsing of `.pars` files with fail-fast
3. Asset serving for `/__/js/`, `/__/css/`, `/__/public/`
4. JavaScript asset with versioned URL helper

This unblocks FEAT-057 (DevTools), FEAT-058 (HTML Components), and FEAT-059 (Error Pages).

## Prerequisites

- [x] FEAT-056 spec created and reviewed
- [x] No blocking dependencies

## Tasks

### Task 1: Create Prelude Directory Structure
**Files**: `server/prelude/js/basil.js`, `server/prelude/css/.gitkeep`, `server/prelude/public/.gitkeep`
**Estimated effort**: Small

Steps:
1. Create `server/prelude/` directory
2. Create `server/prelude/js/basil.js` with component enhancement JavaScript
3. Create empty `server/prelude/css/` and `server/prelude/public/` directories with `.gitkeep`

Tests:
- Files exist and are valid JavaScript

---

### Task 2: Implement Embed System
**Files**: `server/prelude.go`
**Estimated effort**: Medium

Steps:
1. Create `server/prelude.go` with `//go:embed prelude/*` directive
2. Implement `preludeFS` embedded filesystem variable
3. Implement `preludeASTs` map for cached ASTs
4. Implement `initPrelude()` function that:
   - Walks the embedded filesystem
   - Parses all `.pars` files
   - Returns error if any parse fails (fail-fast)
   - Stores ASTs keyed by relative path

Tests:
- `TestInitPrelude_EmptyDir` - no `.pars` files succeeds
- `TestInitPrelude_ValidPars` - valid `.pars` file parses
- `TestInitPrelude_InvalidPars` - invalid `.pars` file returns error
- `TestInitPrelude_MultiplePars` - multiple files all parsed

---

### Task 3: Implement Asset Handler
**Files**: `server/prelude.go`
**Estimated effort**: Medium

Steps:
1. Implement `handlePreludeAsset(w, r)` handler
2. Route based on path prefix (`/__/js/`, `/__/css/`, `/__/public/`)
3. Read file from embedded filesystem
4. Set Content-Type based on file extension
5. Set Cache-Control headers (immutable for versioned, short for others)
6. Handle 404 for missing files

Tests:
- `TestHandlePreludeAsset_JS` - serves JavaScript with correct headers
- `TestHandlePreludeAsset_CSS` - serves CSS with correct headers
- `TestHandlePreludeAsset_NotFound` - returns 404 for missing files
- `TestHandlePreludeAsset_Caching` - versioned files get immutable cache
- `TestHandlePreludeAsset_DirectoryTraversal` - blocks `../` attacks

---

### Task 4: Implement JSAssetURL Helper
**Files**: `server/prelude.go`
**Estimated effort**: Small

Steps:
1. Add `jsAssetHash` package variable
2. Initialize hash in `initPrelude()`:
   - Use `main.Commit` if available
   - Fall back to SHA256 of content (first 7 chars)
3. Implement `JSAssetURL() string` returning `/__/js/basil.{hash}.js`
4. Update asset handler to accept versioned filename

Tests:
- `TestJSAssetURL_WithCommit` - uses commit hash
- `TestJSAssetURL_WithoutCommit` - falls back to content hash
- `TestJSAssetURL_Format` - returns correct format

---

### Task 5: Integrate with Server
**Files**: `server/server.go`
**Estimated effort**: Small

Steps:
1. Call `initPrelude()` in `New()` function
2. Handle error if prelude init fails
3. Register asset routes in `setupRoutes()`:
   - `/__/js/` → `handlePreludeAsset`
   - `/__/css/` → `handlePreludeAsset`
   - `/__/public/` → `handlePreludeAsset`

Tests:
- `TestServerNew_PreludeInit` - server initializes with prelude
- `TestServer_AssetRoutes` - asset routes are registered
- Integration test: `TestServer_ServesBasilJS` - end-to-end JS serving

---

### Task 6: Export Prelude Access Functions
**Files**: `server/prelude.go`
**Estimated effort**: Small

Steps:
1. Implement `GetPreludeAST(path string) *ast.Program` for future use
2. Implement `HasPreludeAST(path string) bool` for checking existence
3. These will be used by FEAT-057/058/059

Tests:
- `TestGetPreludeAST_Exists` - returns AST for existing file
- `TestGetPreludeAST_NotExists` - returns nil for missing file
- `TestHasPreludeAST` - returns correct boolean

---

## Validation Checklist

- [ ] All tests pass: `make test`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] Manual test: Start server, request `/__/js/basil.{hash}.js`
- [ ] Manual test: Verify Cache-Control header is immutable
- [ ] Manual test: Verify 404 for unknown assets

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-12-09 | Task 1: Directory Structure | ✅ Complete | Created prelude/js/basil.js |
| 2025-12-09 | Task 2: Embed System | ✅ Complete | Implemented initPrelude() with fail-fast parsing |
| 2025-12-09 | Task 3: Asset Handler | ✅ Complete | handlePreludeAsset() serves /__/js/, /__/css/, /__/public/ |
| 2025-12-09 | Task 4: JSAssetURL | ✅ Complete | Version hash from commit or content |
| 2025-12-09 | Task 5: Server Integration | ✅ Complete | initPrelude() called in New(), routes registered |
| 2025-12-09 | Task 6: Export Functions | ✅ Complete | GetPreludeAST() and HasPreludeAST() |
| 2025-12-09 | Tests | ✅ Complete | 10 test functions, all passing |

## Deferred Items

Items to add to BACKLOG.md after implementation:
- Source maps for basil.js — useful for debugging but not essential for alpha
- Minification in production builds — optimization for later
- CSS asset (devtools.css) — will be added with FEAT-057

## Notes

### JavaScript Content (basil.js)

The JavaScript provides progressive enhancement for HTML components:

```javascript
// Confirm before submit
document.querySelectorAll('form[data-confirm]').forEach(f => 
  f.addEventListener('submit', e => 
    confirm(f.dataset.confirm) || e.preventDefault()))

// Auto-submit on change
document.querySelectorAll('[data-autosubmit]').forEach(el =>
  el.addEventListener('change', () => el.form.submit()))

// Character counter
document.querySelectorAll('[data-counter]').forEach(ta => {
  const counter = document.getElementById(ta.dataset.counter)
  const max = ta.maxLength
  const update = () => counter.textContent = `${ta.value.length} / ${max}`
  ta.addEventListener('input', update)
  update()
})

// Toggle visibility
document.querySelectorAll('[data-toggle]').forEach(btn => {
  const target = document.querySelector(btn.dataset.toggle)
  btn.setAttribute('aria-controls', target.id)
  btn.setAttribute('aria-expanded', !target.hidden)
  btn.addEventListener('click', () => {
    target.hidden = !target.hidden
    btn.setAttribute('aria-expanded', !target.hidden)
  })
})

// Copy to clipboard
document.querySelectorAll('[data-copy]').forEach(btn => {
  const originalText = btn.textContent
  btn.addEventListener('click', async () => {
    try {
      const text = document.querySelector(btn.dataset.copy).textContent
      await navigator.clipboard.writeText(text)
      btn.textContent = 'Copied!'
    } catch (e) {
      btn.textContent = 'Failed'
    }
    setTimeout(() => btn.textContent = originalText, 2000)
  })
})

// Disable submit button on submit
document.querySelectorAll('form').forEach(f =>
  f.addEventListener('submit', () =>
    f.querySelectorAll('[type=submit]').forEach(b => b.disabled = true)))

// Auto-resize textarea (CSS fallback)
if (!CSS.supports('field-sizing', 'content')) {
  document.querySelectorAll('[data-autoresize]').forEach(ta => {
    const resize = () => { ta.style.height = 'auto'; ta.style.height = ta.scrollHeight + 'px' }
    ta.addEventListener('input', resize)
    resize()
  })
}

// Focus first invalid field
const firstError = document.querySelector('[aria-invalid="true"]')
if (firstError) firstError.focus()
```

### Embed Considerations

- `//go:embed` must be in a Go file in the same directory as the embedded files
- Placing prelude under `server/` keeps it close to the embedding code
- Alternative: `internal/prelude/` with a separate package

### Version Hash Strategy

```go
var jsAssetHash string

func initJSHash() {
    if Commit != "" && Commit != "unknown" {
        jsAssetHash = Commit[:7]
    } else {
        // Development fallback: hash the content
        h := sha256.Sum256(preludeJS)
        jsAssetHash = hex.EncodeToString(h[:])[:7]
    }
}
```

This ensures:
- Production builds use git commit (deterministic)
- Development builds use content hash (changes when JS changes)
