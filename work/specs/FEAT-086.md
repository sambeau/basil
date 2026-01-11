---
id: FEAT-086
title: "Built-in @params, @env, and @args globals"
status: implemented
priority: medium
created: 2026-01-10
author: "@human"
---

# FEAT-086: Built-in @params, @env, and @args globals

## Summary
Add `@params`, `@env`, and `@args` as built-in globals available without requiring imports. These follow the established pattern of factory functions like `@DB` and `@SEARCH`, reducing boilerplate.

**Parsley/Basil boundary:**
- `@env` and `@args` are **Parsley-level** — available in both `pars` CLI and Basil handlers
- `@params` is **Basil-only** — only meaningful in HTTP request context

## User Story
As a Parsley developer, I want common inputs (environment, arguments, request parameters) available without imports so that I can write scripts and handlers with less boilerplate.

## Acceptance Criteria
- [x] `@env` is available in all Parsley contexts (pars + Basil)
- [x] `@args` is available in all Parsley contexts (pars + Basil)
- [x] `@params` is available in Basil handler scopes only
- [x] `@params` contains merged query + form parameters (POST wins)
- [x] All three are standard dictionaries (dot and bracket notation)
- [x] Existing `import @basil/http` continues to work for edge cases
- [ ] Documentation updated with new globals
- [x] Examples updated to use new globals where appropriate

## Design Decisions

### Parsley vs Basil boundary
**Rationale**: 
- `@env` (environment) and `@args` (CLI arguments) are process-level, available everywhere
- `@params` (request parameters) only exists in HTTP context
- This mirrors how other languages work (e.g., Python's `os.environ` vs Flask's `request.args`)

### Naming: `@params` not `@query` or `@form`
**Rationale**: Rails and Sinatra use `params` for merged request parameters. This is the most widely recognized convention. Using `@query` would conflict conceptually with `search.query()` method calls.

### Merge order: POST wins over GET
**Rationale**: 
- POST represents intentional form submission
- GET params are visible in URLs and logs
- Matches Rails convention
- CSRF protection mitigates security concerns about parameter injection

### @args structure
**Rationale**:
- `@args` is an array of strings (like `os.Args` in Go, `sys.argv` in Python)
- For pars: `pars script.pars foo bar` → `@args = ["foo", "bar"]`
- For Basil: server startup arguments (static, not per-request)

### Keep `import @basil/http` available
**Rationale**: Edge cases may need:
- Full request object with all fields
- HTTP headers
- Request method
- Raw request body

Note: `query` export removed from `@basil/http` in favor of `@params`. Use `request.query` if you need GET params specifically.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `pkg/parsley/evaluator/` — Add @env and @args to base Parsley scope
- `server/handler.go` — Add @params to request scope
- `server/prelude.go` — May need updates for scope injection
- `cmd/pars/main.go` — Pass CLI args to Parsley scope
- `cmd/basil/main.go` — Pass server args to Parsley scope
- `docs/guide/` — Update handler and CLI documentation
- `docs/parsley/` — Document new globals
- `examples/` — Update examples to use new globals

### API Design

```parsley
// ============================================================
// PARSLEY-LEVEL (available in both pars CLI and Basil handlers)
// ============================================================

// @env: environment variables (read-only dictionary)
debug = @env.DEBUG == "true"     // Dot notation
apiKey = @env["API_KEY"]         // Bracket notation

// @args: command-line arguments (array)
// pars script.pars hello world  →  @args = ["hello", "world"]
filename = @args[0]              // First argument
allArgs = @args                  // Full array

// ============================================================
// BASIL-ONLY (only available in HTTP handler context)
// ============================================================

// @params: merged query + form (POST overwrites GET)
name = @params.name              // Dot notation
name = @params["name"]           // Bracket notation
page = @params.page ?? "1"

// Brackets required for special characters or dynamic keys
userId = @params["user-id"]      // Hyphen in key
tags = @params["tags[]"]         // Rails-style array param
value = @params[fieldName]       // Dynamic key

// For edge cases, import still works
let {request, headers, method, body} = import @basil/http
// Use request.query for GET-only params (not merged)
```

### Merge Implementation

```go
// In Parsley evaluator setup (base scope)
func setupParsleyGlobals(scope *Scope, args []string) {
    // @env - environment variables as map
    envMap := make(map[string]any)
    for _, e := range os.Environ() {
        if k, v, ok := strings.Cut(e, "="); ok {
            envMap[k] = v
        }
    }
    scope.Set("@env", envMap)
    
    // @args - CLI arguments as array
    scope.Set("@args", args)
}

// In Basil handler setup (request scope)
func buildParams(r *http.Request) map[string]any {
    params := make(map[string]any)
    
    // Query params first (lower priority)
    for key, values := range r.URL.Query() {
        if len(values) == 1 {
            params[key] = values[0]
        } else {
            params[key] = values
        }
    }
    
    // Form params second (higher priority, overwrites)
    if err := r.ParseForm(); err == nil {
        for key, values := range r.PostForm {
            if len(values) == 1 {
                params[key] = values[0]
            } else {
                params[key] = values
            }
        }
    }
    
    return params
}
```

### Multi-value handling
When a parameter appears multiple times (`?tag=a&tag=b`):
- Single value: return as string
- Multiple values: return as array

### Dependencies
- Depends on: None
- Blocks: None

### Edge Cases & Constraints

1. **File uploads** — `@params` contains form fields but not file data. Use `import @basil/http` for `files`.

2. **JSON body** — `@params` only handles `application/x-www-form-urlencoded` and `multipart/form-data`. JSON bodies need explicit parsing via `import @basil/http`.

3. **Array parameters** — `?ids[]=1&ids[]=2` returns `["1", "2"]` for `@params["ids[]"]`. Rails-style `ids[]` naming is preserved.

4. **Empty vs missing** — `?name=` sets `@params["name"]` to `""`. Missing key returns `nil`.

5. **Type coercion** — All values are strings (or arrays of strings). Numeric conversion is the handler's responsibility.

## Migration Guide

### Basil handlers
Before:
```parsley
let {query, form} = import @basil/http
name = query["name"] ?? form["name"]
// or
let {query} = import @basil/http
name = query.name
```

After:
```parsley
name = @params.name

// If you need GET params specifically (not merged):
let {request} = import @basil/http
name = request.query.name
```

### Parsley scripts (pars)
Before:
```parsley
// No standard way to access CLI args or env
```

After:
```parsley
// pars script.pars input.txt output.txt
inputFile = @args[0]
outputFile = @args[1]
verbose = @env.VERBOSE == "true"
```

## Related
- Discussion: Chat 2026-01-10 (params design discussion)
- Similar: `@DB`, `@SEARCH` factory function patterns
