---
id: FEAT-051
title: "Standard Prelude"
status: tracking
priority: high
created: 2025-12-08
author: "@copilot"
tracking:
  - FEAT-056
  - FEAT-057
  - FEAT-058
  - FEAT-059
---

# FEAT-051: Standard Prelude

> **📋 TRACKING ISSUE**: This feature has been split into smaller, independently implementable features:
>
> | Feature | Title | Status | Description |
> |---------|-------|--------|-------------|
> | [FEAT-056](FEAT-056.md) | Prelude Infrastructure | draft | Embed system, startup parsing, asset serving |
> | [FEAT-057](FEAT-057.md) | DevTools in Parsley | draft | File-based routing for `/__/` pages |
> | [FEAT-058](FEAT-058.md) | HTML Components in Prelude | draft | `std/html` loads from prelude |
> | [FEAT-059](FEAT-059.md) | Error Pages | draft | 404, 500, dev error pages |
>
> **Implementation order**: FEAT-056 → (FEAT-057, FEAT-058, FEAT-059 can be parallel)

## Summary

Embed a collection of Parsley source files, JavaScript, and static assets into the Basil binary. This "prelude" provides HTML components, DevTools UI, error pages, and future admin interface—all written in Parsley and human-editable in the source tree. The prelude enables file-based routing under `/__/` so new pages can be added without Go code changes.

## User Story

As a Basil maintainer, I want the DevTools UI, error pages, and HTML components written in Parsley so that I can iterate on them quickly without modifying Go code, while still shipping a single binary.

## Acceptance Criteria

### Core Infrastructure
- [ ] All files under `prelude/` are embedded into the binary via `//go:embed`
- [ ] `.pars` files are parsed at server startup into cached ASTs
- [ ] Parse errors in prelude cause server startup to fail (fail-fast)
- [ ] Single binary deployment - no external files needed

### File-Based Routing for `/__/`
- [ ] `/__/` routes to `prelude/devtools/index.pars`
- [ ] `/__/db` routes to `prelude/devtools/db.pars`
- [ ] `/__/{path}` looks for `prelude/devtools/{path}.pars` or `prelude/devtools/{path}/index.pars`
- [ ] New pages can be added by creating `.pars` files without Go changes
- [ ] 404 for unknown paths under `/__/`

### Asset Serving
- [ ] `/__/js/{file}` serves files from `prelude/js/`
- [ ] `/__/css/{file}` serves files from `prelude/css/` (future)
- [ ] `/__/public/{file}` serves files from `prelude/public/` (future)
- [ ] Proper Content-Type headers based on file extension
- [ ] Immutable caching for versioned assets (FEAT-050)

### Prelude Environment
- [ ] Prelude pages receive a special environment with Basil metadata
- [ ] `basil.version` - current version string
- [ ] `basil.commit` - git commit short hash
- [ ] `basil.dev` - always `true` for `/__/` pages
- [ ] DevTools pages receive additional context (tables, logs, etc.)

## Design Decisions

- **Parsed at startup**: All `.pars` files parsed once when server starts. No lazy loading—simpler and fails fast on errors.
- **File-based routing**: `/__/` uses site-mode-style routing within prelude. Adding a page = adding a file.
- **Single embed directive**: One `//go:embed prelude/**/*` captures everything—Parsley, JS, CSS, images.
- **Inline SVG for icons**: No `/__/public/` initially. Icons are inline SVG in Parsley components. `/__/public/` added when needed (admin interface).
- **Prelude is not user-extensible**: This is Basil's internal code, not a plugin system. Users can't add to the prelude.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Prelude Structure

```
prelude/
├── js/                           # JavaScript assets
│   └── basil.js                  # Component enhancements (FEAT-050)
├── css/                          # CSS assets (future)
│   └── devtools.css
├── public/                       # Static assets (future, for admin)
│   └── ...
├── components/                   # HTML Components (std/html)
│   ├── form.pars
│   ├── text_field.pars
│   ├── select_field.pars
│   ├── button.pars
│   ├── data_table.pars
│   └── ...
├── devtools/                     # DevTools pages (/__/)
│   ├── index.pars                # /__/
│   ├── db.pars                   # /__/db
│   ├── db_table.pars             # /__/db/{table}
│   ├── logs.pars                 # /__/logs
│   ├── env.pars                  # /__/env
│   └── help/
│       ├── index.pars            # /__/help
│       ├── sessions.pars         # /__/help/sessions
│       └── whats-new.pars        # /__/whats-new
└── errors/                       # Error pages
    ├── 404.pars
    ├── 500.pars
    └── dev_error.pars
```

### Embedding

```go
// server/prelude.go

import "embed"

//go:embed prelude/**/*
var preludeFS embed.FS
```

The `**/*` glob captures all files recursively, regardless of extension.

### Startup Parsing

```go
// server/prelude.go

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
        
        // Store with relative path as key: "devtools/index.pars"
        key := strings.TrimPrefix(path, "prelude/")
        preludeASTs[key] = program
        
        return nil
    })
}
```

Called from `server.New()`:

```go
func New(cfg *config.Config) (*Server, error) {
    // ... existing init ...
    
    if err := initPrelude(); err != nil {
        return nil, fmt.Errorf("initializing prelude: %w", err)
    }
    
    // ...
}
```

### DevTools Handler (File-Based Routing)

```go
// server/devtools.go

func (s *Server) handleDevTools(w http.ResponseWriter, r *http.Request) {
    // Only in dev mode
    if !s.config.Server.Dev {
        http.NotFound(w, r)
        return
    }
    
    path := strings.TrimPrefix(r.URL.Path, "/__/")
    if path == "" {
        path = "index"
    }
    
    // Try exact match, then index.pars in directory
    candidates := []string{
        "devtools/" + path + ".pars",
        "devtools/" + path + "/index.pars",
    }
    
    var program *ast.Program
    for _, candidate := range candidates {
        if p, ok := preludeASTs[candidate]; ok {
            program = p
            break
        }
    }
    
    if program == nil {
        s.handlePrelude404(w, r)
        return
    }
    
    // Create environment with devtools context
    env := s.createDevToolsEnv(r, path)
    
    // Evaluate
    result := evaluator.Eval(program, env)
    
    // Handle errors
    if err, ok := result.(*evaluator.Error); ok {
        s.logError("prelude error: %s", err.Inspect())
        http.Error(w, "Internal error", 500)
        return
    }
    
    // Write HTML response
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    fmt.Fprint(w, result.Inspect())
}
```

### DevTools Environment

```go
func (s *Server) createDevToolsEnv(r *http.Request, path string) *evaluator.Environment {
    env := evaluator.NewEnvironment()
    
    // Basil metadata
    basilMeta := &evaluator.Dictionary{
        Pairs: map[string]ast.Expression{
            "version": lit(main.Version),
            "commit":  lit(main.Commit),
            "dev":     lit(true),
        },
    }
    
    // DevTools-specific context
    devtools := &evaluator.Dictionary{
        Pairs: map[string]ast.Expression{
            "path": lit(path),
        },
    }
    
    // Add data based on which page
    switch {
    case strings.HasPrefix(path, "db"):
        devtools.Pairs["tables"] = s.getTableList()
        devtools.Pairs["query"] = s.createQueryFunction()
    case path == "logs":
        devtools.Pairs["entries"] = s.getLogEntries()
    case path == "env":
        devtools.Pairs["config"] = s.getConfigForDisplay()
    }
    
    basilMeta.Pairs["devtools"] = &ast.ObjectLiteralExpression{Obj: devtools}
    env.Set("basil", basilMeta)
    
    return env
}
```

### Asset Handler

```go
// server/prelude.go

func (s *Server) handlePreludeAsset(w http.ResponseWriter, r *http.Request) {
    // Determine asset type from path
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
    
    // Set content type
    contentType := mime.TypeByExtension(path.Ext(filename))
    if contentType == "" {
        contentType = "application/octet-stream"
    }
    w.Header().Set("Content-Type", contentType)
    
    // Caching (immutable for versioned files)
    if strings.Contains(filename, ".") && len(filename) > 10 {
        // Likely has hash in name
        w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
    } else {
        w.Header().Set("Cache-Control", "public, max-age=3600")
    }
    
    w.Write(data)
}
```

### Route Registration

```go
// server/server.go - in setupRoutes()

// DevTools (file-based routing)
mux.HandleFunc("/__/", s.handleDevTools)

// Prelude assets
mux.HandleFunc("/__/js/", s.handlePreludeAsset)
mux.HandleFunc("/__/css/", s.handlePreludeAsset)
mux.HandleFunc("/__/public/", s.handlePreludeAsset)
```

### Example Prelude Page

```parsley
// prelude/devtools/index.pars

fn DevToolsIndex() {
    <!DOCTYPE html>
    <html lang="en">
    <head>
        <meta charset="utf-8"/>
        <meta name="viewport" content="width=device-width, initial-scale=1"/>
        <title>Basil Dev Tools</title>
        <style>
            // ... styles ...
        </style>
    </head>
    <body>
        <div class="container">
            <h1><span class="brand">BASIL</span> Dev Tools</h1>
            <p class="version">v{basil.version} ({basil.commit})</p>
            
            <ul class="tools-list">
                <li><a href="/__/db">Database Browser</a></li>
                <li><a href="/__/logs">Dev Logs</a></li>
                <li><a href="/__/env">Environment</a></li>
                <li><a href="/__/help">Help</a></li>
            </ul>
        </div>
    </body>
    </html>
}

DevToolsIndex()
```

### HTML Components in Prelude

```parsley
// prelude/components/text_field.pars

export fn TextField(props) {
    let {name, label, type, value, hint, error, required} = props
    let inputId = "field-" ++ name
    let hintId = if (hint) { inputId ++ "-hint" } else { null }
    let errorId = if (error) { inputId ++ "-error" } else { null }
    
    let describedBy = [hintId, errorId].filter(fn(x) { x != null }).join(" ")
    
    <div class="field">
        <label for={inputId}>
            {label}
            if (required) {
                <span class="field-required" aria-hidden="true">*</span>
            }
        </label>
        <input 
            type={type ?? "text"}
            id={inputId}
            name={name}
            value={value ?? ""}
            required={required}
            aria-required={required}
            aria-describedby={if (describedBy != "") { describedBy } else { null }}
            aria-invalid={error != null}
        />
        if (hint) {
            <p id={hintId} class="field-hint">{hint}</p>
        }
        if (error) {
            <p id={errorId} class="field-error" role="alert">{error}</p>
        }
    </div>
}
```

Components are loaded via `std/html` import, which pulls from preludeASTs.

### Affected Components

- `server/prelude.go` — New file: embed, parse, asset serving
- `server/server.go` — Call `initPrelude()`, register routes
- `server/devtools.go` — Refactor to use file-based routing
- `server/errors.go` — Use prelude error pages
- `pkg/parsley/evaluator/stdlib_html.go` — New file: load components from prelude

### Implementation Phases

**Phase 1: Infrastructure + DevTools**
- Embed system
- Startup parsing with fail-fast
- File-based routing for `/__/`
- Convert DevTools pages to Parsley
- `/__/js/` serving

**Phase 2: HTML Components**
- Convert component specs to Parsley in `prelude/components/`
- `std/html` module that loads from prelude
- FEAT-046 completion

**Phase 3: Error Pages**
- Simple initial versions
- Dev error page with full details
- Production error pages (minimal)

**Phase 4: Future (Post 1.0)**
- `/__/public/` for admin assets
- Admin interface
- Online help / what's new

### Edge Cases

1. **Parse error in prelude**: Server fails to start with clear error message
2. **Missing page**: Return prelude 404 page (or fallback to simple 404)
3. **Error in prelude page**: Log error, return simple 500 (avoid infinite recursion)
4. **Binary size**: Prelude adds ~50-100KB to binary (acceptable)

### Security Considerations

- `/__/` routes only available in dev mode (existing behavior)
- Prelude code is trusted (part of Basil distribution)
- No user input interpreted as Parsley code
- Asset paths sanitized to prevent directory traversal

## Related

- **FEAT-050**: Built-in JavaScript Assets (now part of prelude)
- **FEAT-046**: HTML Components (consumes prelude components)
- **Design doc**: `work/design/html-components.md`
