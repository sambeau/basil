---
id: FEAT-057
title: "DevTools in Parsley"
status: draft
priority: medium
created: 2025-12-09
author: "@copilot"
depends-on: FEAT-056
part-of: FEAT-051
---

# FEAT-057: DevTools in Parsley

## Summary

Convert the existing DevTools UI (`/__/` routes) from Go templates to Parsley files using file-based routing. New DevTools pages can be added by creating `.pars` files without Go code changes.

## User Story

As a Basil maintainer, I want DevTools pages written in Parsley so that I can iterate on the UI quickly without modifying Go code, while having a consistent experience with the rest of the Parsley ecosystem.

## Acceptance Criteria

### File-Based Routing
- [ ] `/__/` routes to `prelude/devtools/index.pars`
- [ ] `/__/db` routes to `prelude/devtools/db.pars`
- [ ] `/__/{path}` looks for `prelude/devtools/{path}.pars` or `prelude/devtools/{path}/index.pars`
- [ ] New pages can be added by creating `.pars` files without Go changes
- [ ] 404 for unknown paths under `/__/`

### DevTools Environment
- [ ] DevTools pages receive special environment with Basil metadata
- [ ] `basil.version` - current version string
- [ ] `basil.commit` - git commit short hash  
- [ ] `basil.dev` - always `true` for `/__/` pages
- [ ] Database pages receive table list and query function
- [ ] Logs page receives log entries
- [ ] Env page receives config for display

### Page Conversion
- [ ] Convert `/__/` index page to Parsley
- [ ] Convert `/__/db` database browser to Parsley
- [ ] Convert `/__/db/{table}` table view to Parsley
- [ ] Convert `/__/logs` dev logs to Parsley
- [ ] Convert `/__/env` environment view to Parsley

## Design Decisions

- **File-based routing**: Similar to site-mode routing within `prelude/devtools/`
- **Pre-parsed ASTs**: Uses ASTs from FEAT-056 infrastructure
- **DevTools only in dev mode**: Existing behavior preserved
- **Fallback to simple 404**: If prelude 404 page fails, use plain text

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Directory Structure

```
prelude/
├── devtools/
│   ├── index.pars              # /__/
│   ├── db.pars                 # /__/db
│   ├── db_table.pars           # /__/db/{table}
│   ├── logs.pars               # /__/logs
│   ├── env.pars                # /__/env
│   └── help/
│       ├── index.pars          # /__/help
│       └── sessions.pars       # /__/help/sessions
└── ...
```

### DevTools Handler

```go
// server/devtools.go

func (s *Server) handleDevTools(w http.ResponseWriter, r *http.Request) {
    if !s.config.Server.Dev {
        http.NotFound(w, r)
        return
    }
    
    path := strings.TrimPrefix(r.URL.Path, "/__/")
    if path == "" {
        path = "index"
    }
    
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
    
    env := s.createDevToolsEnv(r, path)
    result := evaluator.Eval(program, env)
    
    if err, ok := result.(*evaluator.Error); ok {
        s.logError("prelude error: %s", err.Inspect())
        http.Error(w, "Internal error", 500)
        return
    }
    
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    fmt.Fprint(w, result.Inspect())
}
```

### DevTools Environment

```go
func (s *Server) createDevToolsEnv(r *http.Request, path string) *evaluator.Environment {
    env := evaluator.NewEnvironment()
    
    basilMeta := map[string]interface{}{
        "version": main.Version,
        "commit":  main.Commit,
        "dev":     true,
    }
    
    devtools := map[string]interface{}{
        "path": path,
    }
    
    switch {
    case strings.HasPrefix(path, "db"):
        devtools["tables"] = s.getTableList()
        devtools["query"] = s.createQueryFunction()
    case path == "logs":
        devtools["entries"] = s.getLogEntries()
    case path == "env":
        devtools["config"] = s.getConfigForDisplay()
    }
    
    basilMeta["devtools"] = devtools
    env.Set("basil", basilMeta)
    
    return env
}
```

### Example Page (index.pars)

```parsley
fn DevToolsIndex() {
    <!DOCTYPE html>
    <html lang="en">
    <head>
        <meta charset="utf-8"/>
        <meta name="viewport" content="width=device-width, initial-scale=1"/>
        <title>Basil Dev Tools</title>
        <style>
            body { font-family: system-ui; max-width: 800px; margin: 2rem auto; }
            .brand { color: #228B22; }
            .version { color: #666; font-size: 0.9em; }
            .tools-list { list-style: none; padding: 0; }
            .tools-list li { margin: 0.5rem 0; }
            .tools-list a { font-size: 1.2em; }
        </style>
    </head>
    <body>
        <h1><span class="brand">BASIL</span> Dev Tools</h1>
        <p class="version">v{basil.version} ({basil.commit})</p>
        
        <ul class="tools-list">
            <li><a href="/__/db">Database Browser</a></li>
            <li><a href="/__/logs">Dev Logs</a></li>
            <li><a href="/__/env">Environment</a></li>
            <li><a href="/__/help">Help</a></li>
        </ul>
    </body>
    </html>
}

DevToolsIndex()
```

### Route Registration

```go
// server/server.go - replaces existing devtools handler
mux.HandleFunc("/__/", s.handleDevTools)
```

### Affected Files

- `server/devtools.go` — Refactor to use file-based routing
- `server/server.go` — Update route registration
- `prelude/devtools/*.pars` — New DevTools pages

## Related

- **Depends on**: FEAT-056 (Prelude Infrastructure)
- **Part of**: FEAT-051 (Standard Prelude)
