---
id: FEAT-050
title: "Built-in JavaScript Assets"
status: superseded
priority: medium
created: 2025-12-08
author: "@copilot"
superseded-by: FEAT-056
---

# FEAT-050: Built-in JavaScript Assets

> **⚠️ SUPERSEDED**: This feature has been superseded by [FEAT-056](FEAT-056.md) (Prelude Infrastructure), which provides a more general solution that includes JavaScript asset serving as part of the prelude system.

## Summary

Serve built-in JavaScript assets from `/__/js/` path for HTML component enhancements. A single `basil.js` file provides progressive enhancement behaviors (form protection, toggles, counters, etc.) with proper caching via content-based version hashing.

## User Story

As a Basil developer using HTML components, I want the required JavaScript to be automatically available and efficiently cached so that my forms and interactive elements work without manual asset setup.

## Acceptance Criteria

- [ ] `/__/js/basil.{hash}.js` serves the component JavaScript
- [ ] Hash is derived from git commit (short hash) for cache busting
- [ ] Response includes `Cache-Control: public, max-age=31536000, immutable`
- [ ] Content-Type is `application/javascript`
- [ ] Works in both dev and production modes
- [ ] HTML components can emit `<script type="module" src="/__/js/basil.{hash}.js">`
- [ ] Multiple script tags for same URL only execute once (ES module behavior)

## Design Decisions

- **Path `/__/js/`**: Consistent with other Basil internal routes (`/__/db`, `/__/logs`)
- **Single file**: ~55 lines total, likely cached after first page load
- **Version hash**: Git commit short hash (already available via `main.Commit`)
- **`type="module"`**: Guarantees single execution even with duplicate script tags
- **Immutable caching**: Hash changes when content changes, so cache forever
- **Embedded in binary**: No external files to deploy

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### URL Format

```
/__/js/basil.{commit}.js
```

Example: `/__/js/basil.c0f1c82.js`

The commit hash is the 7-character short hash from `git rev-parse --short HEAD`, already available as `main.Commit` at build time.

### JavaScript Content

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

### Server Implementation

```go
// server/js_assets.go

//go:embed assets/basil.js
var basilJS []byte

// jsAssetHash is computed at startup from the embedded content or commit
var jsAssetHash string

func init() {
    // Use commit hash if available, otherwise hash the content
    if main.Commit != "" {
        jsAssetHash = main.Commit
    } else {
        h := sha256.Sum256(basilJS)
        jsAssetHash = hex.EncodeToString(h[:])[:7]
    }
}

func (s *Server) handleJSAsset(w http.ResponseWriter, r *http.Request) {
    // Expected path: /__/js/basil.{hash}.js
    expected := fmt.Sprintf("basil.%s.js", jsAssetHash)
    requested := strings.TrimPrefix(r.URL.Path, "/__/js/")
    
    if requested != expected {
        // Wrong hash or unknown file - 404
        http.NotFound(w, r)
        return
    }
    
    w.Header().Set("Content-Type", "application/javascript")
    w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
    w.Write(basilJS)
}

// JSAssetURL returns the versioned URL for basil.js
func JSAssetURL() string {
    return fmt.Sprintf("/__/js/basil.%s.js", jsAssetHash)
}
```

### Usage in HTML Components

Components that need JavaScript emit the script tag:

```go
// In component rendering
fmt.Fprintf(w, `<script type="module" src="%s"></script>`, server.JSAssetURL())
```

Or in Parsley component:

```parsley
<script type="module" src={basil.js.url}/>
```

The `type="module"` ensures:
1. Browser only executes the script once (even with duplicate tags)
2. Script runs after DOM is ready (deferred by default)
3. Strict mode enabled

### Route Registration

```go
// In server.go setupRoutes()
mux.HandleFunc("/__/js/", s.handleJSAsset)
```

### Affected Components

- `server/js_assets.go` — New file: embedded JS and handler
- `server/server.go` — Register `/__/js/` route
- `cmd/basil/main.go` — Expose `Commit` variable for hash

### Edge Cases

1. **No commit hash** (dev build): Fall back to content hash
2. **Unknown file requested**: Return 404
3. **Wrong hash**: Return 404 (forces refresh to get new URL)
4. **Multiple script tags**: ES modules handle deduplication

### Future Considerations

- Could add source maps for debugging (`basil.{hash}.js.map`)
- Could minify in production builds
- Could add other asset types (CSS) using same pattern
- Components could register which features they need, only include used code

## Related

- **Parent**: FEAT-046 (HTML Components) - consumes this infrastructure
- **Design doc**: `docs/design/html-components.md`
