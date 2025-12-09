---
id: FEAT-059
title: "Error Pages in Prelude"
status: draft
priority: low
created: 2025-12-09
author: "@copilot"
depends-on: FEAT-056
part-of: FEAT-051
---

# FEAT-059: Error Pages in Prelude

## Summary

Move error page rendering from Go templates to Parsley files in the prelude. Provides different error pages for development (detailed) and production (minimal) modes.

## User Story

As a Basil maintainer, I want error pages written in Parsley so that they're consistent with the rest of the UI and easy to customize without Go code changes.

## Acceptance Criteria

### Error Pages
- [ ] `prelude/errors/404.pars` - Not Found page
- [ ] `prelude/errors/500.pars` - Internal Server Error page
- [ ] `prelude/errors/dev_error.pars` - Detailed error for dev mode

### Dev vs Production
- [ ] Dev mode shows detailed error with stack trace, request info
- [ ] Production mode shows minimal, user-friendly error
- [ ] Parsley parse/eval errors show source location in dev mode

### Error Environment
- [ ] Error pages receive error details in environment
- [ ] `error.code` - HTTP status code
- [ ] `error.message` - Error message
- [ ] `error.details` - Additional details (dev mode only)
- [ ] `error.stack` - Stack trace (dev mode only)
- [ ] `error.request` - Request info (dev mode only)

### Fallback Handling
- [ ] If error page itself fails, fall back to plain text
- [ ] No infinite recursion if error page has error

## Design Decisions

- **Simple initially**: Basic error pages, enhanced over time
- **Fail-safe**: Always have a fallback if Parsley rendering fails
- **No sensitive data in production**: Stack traces, paths only in dev mode

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Directory Structure

```
prelude/
├── errors/
│   ├── 404.pars              # Not Found
│   ├── 500.pars              # Internal Server Error
│   └── dev_error.pars        # Detailed dev error
└── ...
```

### Error Handler

```go
// server/errors.go

func (s *Server) renderError(w http.ResponseWriter, r *http.Request, code int, err error) {
    // Try to use prelude error page
    if !s.renderPreludeError(w, r, code, err) {
        // Fallback to plain text
        http.Error(w, http.StatusText(code), code)
    }
}

func (s *Server) renderPreludeError(w http.ResponseWriter, r *http.Request, code int, err error) bool {
    var pageName string
    if s.config.Server.Dev && err != nil {
        pageName = "errors/dev_error.pars"
    } else {
        pageName = fmt.Sprintf("errors/%d.pars", code)
    }
    
    program, ok := preludeASTs[pageName]
    if !ok {
        // Try generic error page
        program, ok = preludeASTs["errors/500.pars"]
        if !ok {
            return false
        }
    }
    
    env := s.createErrorEnv(r, code, err)
    result := evaluator.Eval(program, env)
    
    if _, isErr := result.(*evaluator.Error); isErr {
        // Error page failed - don't recurse
        return false
    }
    
    w.WriteHeader(code)
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    fmt.Fprint(w, result.Inspect())
    return true
}
```

### Error Environment

```go
func (s *Server) createErrorEnv(r *http.Request, code int, err error) *evaluator.Environment {
    env := evaluator.NewEnvironment()
    
    errorInfo := map[string]interface{}{
        "code":    code,
        "message": http.StatusText(code),
    }
    
    if s.config.Server.Dev && err != nil {
        errorInfo["details"] = err.Error()
        errorInfo["stack"] = getStackTrace()
        errorInfo["request"] = map[string]interface{}{
            "method": r.Method,
            "path":   r.URL.Path,
            "query":  r.URL.RawQuery,
        }
    }
    
    env.Set("error", errorInfo)
    env.Set("basil", map[string]interface{}{
        "version": main.Version,
        "dev":     s.config.Server.Dev,
    })
    
    return env
}
```

### Example Error Page (404.pars)

```parsley
fn NotFoundPage() {
    <!DOCTYPE html>
    <html lang="en">
    <head>
        <meta charset="utf-8"/>
        <title>Page Not Found</title>
        <style>
            body { 
                font-family: system-ui; 
                display: flex; 
                justify-content: center; 
                align-items: center; 
                height: 100vh; 
                margin: 0;
            }
            .error { text-align: center; }
            .error h1 { font-size: 4rem; margin: 0; color: #666; }
            .error p { color: #999; }
        </style>
    </head>
    <body>
        <div class="error">
            <h1>404</h1>
            <p>Page not found</p>
            <p><a href="/">Go home</a></p>
        </div>
    </body>
    </html>
}

NotFoundPage()
```

### Example Dev Error Page (dev_error.pars)

```parsley
fn DevErrorPage() {
    <!DOCTYPE html>
    <html lang="en">
    <head>
        <meta charset="utf-8"/>
        <title>Error {error.code}</title>
        <style>
            body { font-family: monospace; padding: 2rem; background: #1a1a1a; color: #eee; }
            .error-box { background: #2a2a2a; border-left: 4px solid #e74c3c; padding: 1rem; margin: 1rem 0; }
            .stack { background: #222; padding: 1rem; overflow-x: auto; }
            .request { background: #222; padding: 1rem; }
            h1 { color: #e74c3c; }
            pre { margin: 0; }
        </style>
    </head>
    <body>
        <h1>Error {error.code}: {error.message}</h1>
        
        if (error.details) {
            <div class="error-box">
                <strong>Details:</strong>
                <pre>{error.details}</pre>
            </div>
        }
        
        if (error.request) {
            <div class="request">
                <strong>Request:</strong>
                <pre>{error.request.method} {error.request.path}</pre>
                if (error.request.query) {
                    <pre>Query: {error.request.query}</pre>
                }
            </div>
        }
        
        if (error.stack) {
            <div class="stack">
                <strong>Stack Trace:</strong>
                <pre>{error.stack}</pre>
            </div>
        }
    </body>
    </html>
}

DevErrorPage()
```

### Affected Files

- `server/errors.go` — Use prelude for error rendering
- `prelude/errors/*.pars` — Error page implementations

## Related

- **Depends on**: FEAT-056 (Prelude Infrastructure)
- **Part of**: FEAT-051 (Standard Prelude)
