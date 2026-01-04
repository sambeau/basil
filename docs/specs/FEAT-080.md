---
id: FEAT-080
title: "Decouple @DB from Request Context"
status: implemented
priority: high
created: 2026-01-04
implemented: 2026-01-04
author: "@human"
---

# FEAT-080: Decouple @DB from Request Context

## Summary

Move `@DB` resolution from the per-request `BasilCtx` to a server-level environment field. This allows modules to use `@DB` at load time (module scope), enabling schema bindings to be created once at startup rather than per-request.

## Problem Statement

Currently, `@DB` is only available inside request handlers because:

1. `@DB` resolves by looking up `env.BasilCtx["sqlite"]`
2. `BasilCtx` is only populated when a request arrives (in `buildBasilContext()`)
3. Module cache is cleared per-request so modules can see fresh `basil.http.request` values

**But this is architecturally wrong:**

- The database connection (`s.db`) is opened at server startup via `initSQLite()`
- The database is **server infrastructure**, not request state
- The per-request `BasilCtx` conflates:
  - **Server resources**: database, session store, fragment cache
  - **Request context**: HTTP method, URL, headers, cookies, auth user

This prevents the common pattern of binding schemas at module scope:

```parsley
// models.pars — THIS SHOULD WORK but currently fails
let {schema} = import @std/schema

@schema User { id: int, name: string, email: string }

// ❌ ERROR: "@DB is only available in Basil server handlers"
let Users = schema.table(User, @DB, "users")

export Users
```

## Solution

Separate database from request context by adding a server-level database field to `Environment`.

### Current Architecture

```
Request arrives
    ↓
buildBasilContext(r, db, ...) → BasilCtx dict with sqlite, http, auth, session
    ↓
env.BasilCtx = basilObj
    ↓
ClearModuleCache() — forces modules to re-evaluate per request
    ↓
Eval(program, env) — @DB resolves from env.BasilCtx["sqlite"]
```

### Proposed Architecture

```
Server starts
    ↓
initSQLite() → s.db exists
    ↓
Handlers registered with env.ServerDB = s.db  ← NEW

Request arrives
    ↓
buildBasilContext(r, ...) → BasilCtx with http, auth, session (NO sqlite)
    ↓
env.BasilCtx = basilObj
    ↓
Module cache NOT cleared for @DB (only for request-dependent modules)
    ↓
Eval(program, env) — @DB resolves from env.ServerDB first
```

## Design

### New Environment Field

```go
type Environment struct {
    // ... existing fields ...
    
    // ServerDB holds the server's database connection.
    // Set at handler registration time, available to all modules.
    // Resolves @DB literal when BasilCtx is not available.
    ServerDB     *DBConnection
}
```

### Updated @DB Resolution

```go
func evalDBLiteral(node *ast.DBLiteral, env *Environment) Object {
    // 1. Try server-level database first (available at module load time)
    if env.ServerDB != nil {
        return env.ServerDB
    }
    
    // 2. Fall back to BasilCtx["sqlite"] (backward compatible)
    if env.BasilCtx != nil {
        if basilDict, ok := env.BasilCtx.(*Dictionary); ok {
            if sqliteExpr, ok := basilDict.Pairs["sqlite"]; ok {
                // ... existing resolution logic ...
            }
        }
    }
    
    // 3. Error if neither available
    return &Error{
        Class:   ErrorClass("state"),
        Message: "@DB is only available in Basil server context",
        Hints:   []string{"Run inside a Basil handler or module with a configured database"},
    }
}
```

### Environment Propagation

When modules are loaded, `ServerDB` must propagate:

```go
// In importModule()
moduleEnv := NewEnvironment()
moduleEnv.ServerDB = env.ServerDB  // ← ADD THIS
moduleEnv.Filename = absPath
// ... rest of setup ...
```

### Server Handler Setup

```go
// In handler.go, before evaluating the script
env := evaluator.NewEnvironment()
env.Filename = h.scriptPath

// Set server-level database (available to modules at load time)
if h.server.db != nil {
    env.ServerDB = evaluator.NewManagedDBConnection(h.server.db, h.server.dbDriver)
}

// Build request-specific context (http, auth, session - NOT sqlite)
basilObj := buildBasilContext(r, h.route, reqCtx, h.route.PublicDir, ...)
env.BasilCtx = basilObj
```

### Remove sqlite from BasilCtx

Update `buildBasilContext()` to NOT include sqlite:

```go
func buildBasilContext(r *http.Request, route config.Route, reqCtx map[string]interface{}, 
    publicDir string, ...) evaluator.Object {
    
    // Build the basil namespace (sqlite removed - now at env.ServerDB)
    basilMap := map[string]interface{}{
        "http": map[string]interface{}{...},
        "auth": authCtx,
        "context": map[string]interface{}{},
        "public_dir": publicDir,
        "csrf": map[string]interface{}{...},
    }
    // ... NO sqlite here ...
}
```

## Module Caching Strategy

### Current: Clear All Per-Request

```go
// handler.go
evaluator.ClearModuleCache()  // Clears ALL modules every request
```

### Proposed: Selective Clearing

Modules that only use server resources (`@DB`, schemas) should be cached across requests. Only modules that access request context (`basil.http.request`) need re-evaluation.

**Option A: Don't clear cache at all**
- Modules are cached indefinitely
- `basil.http.request` accessed via environment, not cached in module
- Simplest, but requires modules to not store request values at module scope

**Option B: Mark request-dependent modules**
- Track which modules access `BasilCtx` at evaluation time
- Only clear those modules per-request
- More complex, preserves current flexibility

**Recommendation: Option A** with documentation that modules should not store `basil.http.*` values at module scope. This matches how most web frameworks work.

## Acceptance Criteria

- [x] `@DB` resolves from `env.ServerDB` when available
- [x] `env.ServerDB` is set before handler evaluation
- [x] `env.ServerDB` propagates to imported modules
- [x] Modules can use `@DB` at module scope (not just in functions)
- [x] `schema.table()` works at module scope with `@DB`
- [x] Backward compatible: existing handlers using `@DB` still work
- [x] Error message updated: "Basil server context" not "Basil server handlers"

## Test Cases

### Module-Scope @DB (New)
```parsley
// test_module.pars
let db = @DB  // Should work at module scope
export db
```

### Module-Scope Schema Binding (New)
```parsley
// models.pars
let {schema} = import @std/schema

@schema User { id: int, name: string }
let Users = schema.table(User, @DB, "users")

export Users
```

### Handler Using Module (New)
```parsley
// handler.pars
let {Users} = import @./models

// Use pre-bound table
let users = Users.all()
<ul>
    {for (u in users) { <li>{u.name}</li> }}
</ul>
```

### Backward Compatibility
```parsley
// Existing code should still work
let user = @DB <=?=> "SELECT * FROM users WHERE id = 1"
```

### Standalone Parsley (No Server)
```parsley
// pars script without server should error clearly
let db = @DB  // Error: "@DB is only available in Basil server context"
```

## Migration

**No breaking changes.** All existing code continues to work.

New capability: `@DB` available at module scope in Basil handlers.

## Implementation Notes

1. **Thread safety**: `ServerDB` is read-only after server init, no locking needed
2. **Connection management**: `ServerDB` uses same `*DBConnection` as before, lifecycle unchanged
3. **Multiple databases**: Future work could add `env.ServerDBs["name"]` for named connections
4. **HUP handling**: On config reload, create new `ServerDB`, clear module cache

## Related

- FEAT-079: Query DSL (depends on this for module-scope bindings)
- FEAT-034: Schema Validation (schema.table uses @DB)
- FEAT-078: TableBinding API (TableBinding creation at module scope)
