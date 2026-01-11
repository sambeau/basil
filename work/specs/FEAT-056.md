---
id: FEAT-056
title: "Prelude Infrastructure"
status: implemented
priority: high
created: 2025-12-09
implemented: 2025-12-09
author: "@copilot"
supersedes: FEAT-050
part-of: FEAT-051
---

# FEAT-056: Prelude Infrastructure

## Summary

Create the foundational infrastructure for embedding Parsley source files and static assets into the Basil binary. This includes the embed system, startup parsing, and asset serving for `/__/js/`, `/__/css/`, and `/__/public/` paths.

This feature extracts the infrastructure portion of FEAT-051 and supersedes FEAT-050 (Built-in JavaScript Assets) by providing a more general solution.

## User Story

As a Basil maintainer, I want a system that embeds Parsley files and static assets into the binary so that future features (DevTools, HTML components, error pages) can be written in Parsley and shipped as a single binary.

## Acceptance Criteria

### Embedding
- [x] All files under `prelude/` are embedded via `//go:embed`
- [x] `.pars` files are parsed at server startup into cached ASTs
- [x] Parse errors in prelude cause server startup to fail (fail-fast)
- [x] Single binary deployment - no external files needed

### Asset Serving
- [x] `/__/js/{file}` serves files from `prelude/js/`
- [x] `/__/css/{file}` serves files from `prelude/css/` (empty initially)
- [x] `/__/public/{file}` serves files from `prelude/public/` (empty initially)
- [x] Proper Content-Type headers based on file extension
- [x] Versioned assets (with hash in filename) get immutable caching
- [x] `JSAssetURL()` returns versioned URL for `basil.js`

### JavaScript Assets (from FEAT-050)
- [x] `prelude/js/basil.js` contains component enhancement JavaScript
- [x] Hash derived from git commit (short hash) for cache busting
- [x] Response includes `Cache-Control: public, max-age=31536000, immutable`
- [x] HTML components can emit `<script type="module" src="...">`

## Design Decisions

- **Single embed directive**: One `//go:embed prelude/**/*` captures everything
- **Parsed at startup**: All `.pars` files parsed once, fail-fast on errors
- **Git commit hash**: Use `main.Commit` for versioning, fallback to content hash
- **Prelude is not user-extensible**: Internal Basil code only

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Directory Structure

```
prelude/
├── js/
│   └── basil.js              # Component enhancements
├── css/                       # Empty initially
└── public/                    # Empty initially
```

Future features (FEAT-057, 058, 059) will add more directories.

### Embedding

```go
// server/prelude.go

import "embed"

//go:embed prelude/**/*
var preludeFS embed.FS
```

### Startup Parsing

```go
var preludeASTs map[string]*ast.Program

func initPrelude() error {
    preludeASTs = make(map[string]*ast.Program)
    
    return fs.WalkDir(preludeFS, "prelude", func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return err
        }
        if d.IsDir() || !strings.HasSuffix(path, ".pars") {
            return nil
        }
        
        source, err := preludeFS.ReadFile(path)
        if err != nil {
            return fmt.Errorf("reading %s: %w", path, err)
        }
        
        l := lexer.New(string(source))
        p := parser.New(l)
        program := p.ParseProgram()
        
        if len(p.Errors()) > 0 {
            return fmt.Errorf("parse error in %s: %v", path, p.Errors())
        }
        
        key := strings.TrimPrefix(path, "prelude/")
        preludeASTs[key] = program
        return nil
    })
}
```

### Asset Handler

```go
func (s *Server) handlePreludeAsset(w http.ResponseWriter, r *http.Request) {
    var dir string
    switch {
    case strings.HasPrefix(r.URL.Path, "/__/js/"):
        dir = "js"
    case strings.HasPrefix(r.URL.Path, "/__/css/"):
        dir = "css"
    case strings.HasPrefix(r.URL.Path, "/__/public/"):
        dir = "public"
    default:
        http.NotFound(w, r)
        return
    }
    
    filename := strings.TrimPrefix(r.URL.Path, "/__/"+dir+"/")
    filepath := "prelude/" + dir + "/" + filename
    
    data, err := preludeFS.ReadFile(filepath)
    if err != nil {
        http.NotFound(w, r)
        return
    }
    
    contentType := mime.TypeByExtension(path.Ext(filename))
    if contentType == "" {
        contentType = "application/octet-stream"
    }
    w.Header().Set("Content-Type", contentType)
    
    // Immutable caching for versioned files
    if isVersionedAsset(filename) {
        w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
    } else {
        w.Header().Set("Cache-Control", "public, max-age=3600")
    }
    
    w.Write(data)
}
```

### JavaScript Content (basil.js)

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

### JSAssetURL Helper

```go
var jsAssetHash string

func init() {
    if main.Commit != "" {
        jsAssetHash = main.Commit
    } else {
        h := sha256.Sum256(basilJS)
        jsAssetHash = hex.EncodeToString(h[:])[:7]
    }
}

func JSAssetURL() string {
    return fmt.Sprintf("/__/js/basil.%s.js", jsAssetHash)
}
```

### Route Registration

```go
// server/server.go - in setupRoutes()
mux.HandleFunc("/__/js/", s.handlePreludeAsset)
mux.HandleFunc("/__/css/", s.handlePreludeAsset)
mux.HandleFunc("/__/public/", s.handlePreludeAsset)
```

### Affected Files

- `server/prelude.go` — New file: embed, parse, asset serving
- `server/server.go` — Call `initPrelude()`, register asset routes

## Related

- **Supersedes**: FEAT-050 (Built-in JavaScript Assets)
- **Part of**: FEAT-051 (Standard Prelude) — this is the infrastructure phase
- **Enables**: FEAT-057 (DevTools in Parsley), FEAT-058 (HTML Components in Prelude), FEAT-059 (Error Pages)
