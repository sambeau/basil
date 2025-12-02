---
id: FEAT-011
title: "Basil namespace for Parsley environment"
status: done
priority: high
created: 2025-12-01
author: "@human"
---

# FEAT-011: Basil namespace for Parsley environment

## Summary
Move all Basil-injected variables into a `basil` namespace to avoid naming conflicts with user-defined variables and make the source of values explicit.

## User Story
As a Parsley script author, I want Basil-provided variables under a clear namespace so that I don't accidentally overwrite them and can easily identify what comes from the framework.

## Acceptance Criteria
- [x] All Basil-injected values are under `basil.*` namespace
- [x] `basil.http.request` contains incoming request data
- [x] `basil.http.response` can set status code and headers (not body)
- [x] Return value is always used as response body
- [x] `basil.auth.required` indicates if route requires auth
- [x] `basil.auth.user` contains authenticated user or nil
- [x] `basil.sqlite` provides database connection (when configured)
- [x] Old top-level variables (`request`, `method`, `path`, `query`, `db`) are removed
- [x] Examples updated to use new namespace
- [x] Documentation updated

## Design Decisions

### Namespace Structure
```parsley
basil.http.request      # {method, path, query, headers, host, body, form, files}
basil.http.response     # {status, headers} - for setting status code, cookies, etc.

basil.auth.required     # bool - is auth required for this route?
basil.auth.user         # {id, name, email, created} or nil

basil.sqlite            # DBConnection object (when configured)
# Future: basil.postgres, basil.mysql
```

### Response Handling
- **Return value = body** (always)
- **`basil.http.response` = metadata** (status code, headers, cookies)
- Both work together:
  ```parsley
  # Set a cookie and custom status, return HTML body
  basil.http.response.headers["Set-Cookie"] = "session=abc123; HttpOnly"
  basil.http.response.status = 201
  
  <html>
    <body>Account created!</body>
  </html>
  ```
- For redirects, the body is typically empty but could contain a message:
  ```parsley
  basil.http.response.status = 302
  basil.http.response.headers["Location"] = "/dashboard"
  
  "Redirecting..."  # Optional body (usually ignored by browser)
  ```

### Database Naming
- Use `basil.sqlite` instead of generic `db`
- Future databases will be `basil.postgres`, `basil.mysql`, etc.
- Makes it explicit which database type is configured

### Breaking Change
- This is a breaking change for existing scripts
- No deprecation period - clean break at version boundary

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `server/handler.go` — Inject `basil` object instead of top-level vars
- `server/handler.go` — Check `basil.http.response` after script execution
- `examples/hello/` — Update to use new namespace
- `examples/auth/` — Update to use new namespace
- `docs/guide/` — Update documentation

### Current Injected Variables (to be replaced)
```go
setEnvVar(env, "request", reqCtx)
setEnvVar(env, "method", r.Method)
setEnvVar(env, "path", r.URL.Path)
setEnvVar(env, "query", queryToMap(r.URL.Query()))
env.Set("db", conn)
```

### New Structure
```go
basil := map[string]interface{}{
    "http": map[string]interface{}{
        "request": reqCtx,  // {method, path, query, headers, host, body, form, files}
        "response": map[string]interface{}{
            "status": 200,
            "headers": map[string]interface{}{},
        },
    },
    "auth": map[string]interface{}{
        "required": route.Auth == "required",
        "user": userMap,    // or nil
    },
    "sqlite": conn,         // DBConnection or nil
}
setEnvVar(env, "basil", basil)
```

### Response Handling
After script execution:
1. Read `basil.http.response.status` - use as HTTP status code (default 200)
2. Read `basil.http.response.headers` - add to response headers
3. Use script return value as response body (always)

### Edge Cases & Constraints
1. `basil.http.response` may be partially set (e.g., just status) - merge with defaults
2. Database connection only added if configured
3. Auth user only added if auth middleware ran

## Implementation Notes
*Added during/after implementation*

## Related
- Plan: `docs/plans/FEAT-011-plan.md`
