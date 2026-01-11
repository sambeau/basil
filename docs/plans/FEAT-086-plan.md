---
id: PLAN-057
feature: FEAT-086
title: "Implementation Plan for @params, @env, and @args globals"
status: draft
created: 2026-01-10
---

# Implementation Plan: FEAT-086

## Overview
Add `@params`, `@env`, and `@args` as built-in globals. `@env` and `@args` are Parsley-level (available in pars CLI and Basil), while `@params` is Basil-only (HTTP request context).

## Prerequisites
- [x] Design decision: Naming convention (@params, not @query)
- [x] Design decision: Merge order (POST wins over GET)
- [x] Design decision: Parsley/Basil boundary

## Tasks

### Task 1: Add @env and @args to Parsley NewEnvironment()
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small

The base Parsley environment needs `@env` populated at creation time. `@args` requires being passed in from the CLI/server.

Steps:
1. Create `NewEnvironmentWithArgs(args []string)` function
2. Populate `@env` dictionary from `os.Environ()`
3. Populate `@args` array from provided args
4. Update `NewEnvironment()` to call `NewEnvironmentWithArgs(nil)` for backwards compatibility

Tests:
- @env contains PATH environment variable
- @env returns nil for nonexistent key
- @args contains CLI arguments when provided
- @args is empty array when no args provided

---

### Task 2: Update pars CLI to pass arguments
**Files**: `cmd/pars/main.go`
**Estimated effort**: Small

The pars CLI currently doesn't pass script arguments. Update to use new environment factory.

Steps:
1. Change `evaluator.NewEnvironment()` to `evaluator.NewEnvironmentWithArgs(scriptArgs)`
2. Script args are everything after the filename: `pars script.pars arg1 arg2` → `["arg1", "arg2"]`
3. Update REPL to use empty args

Tests:
- `pars script.pars foo bar` → @args = ["foo", "bar"]
- REPL → @args = []

---

### Task 3: Add @params to Basil request scope  
**Files**: `server/handler.go`
**Estimated effort**: Medium

Add `@params` to the Parsley scope when executing handlers. This merges query string and form data with POST overwriting GET.

Steps:
1. Create `buildParams(r *http.Request) map[string]interface{}` helper
2. Merge `r.URL.Query()` (lower priority) with `r.PostForm` (higher priority)
3. Handle multi-value params (single → string, multiple → array)
4. Add params to environment via `env.Set("@params", ...)`
5. Wire into `parsleyHandler.ServeHTTP()` before script evaluation

Tests:
- GET `?name=alice` → @params.name = "alice"
- POST name=bob → @params.name = "bob"  
- GET `?name=alice` + POST name=bob → @params.name = "bob" (POST wins)
- GET `?tag=a&tag=b` → @params.tag = ["a", "b"]

---

### Task 4: Wire @env and @args into Basil server
**Files**: `server/handler.go`, `cmd/basil/main.go`
**Estimated effort**: Small

Ensure @env and @args are available in Basil handlers (inherited from Parsley).

Steps:
1. Update Basil's environment creation to use `NewEnvironmentWithArgs(serverArgs)`
2. Server args = CLI args after config file: `basil -c config.yaml --foo` → ["--foo"]
3. @env is automatically populated from Parsley layer

Tests:
- @env.HOME contains home directory
- @args contains server startup arguments

---

### Task 5: Update API handler for @params
**Files**: `server/api.go`
**Estimated effort**: Small

API handlers (`@API`) need the same @params treatment as regular handlers.

Steps:
1. Add @params to `buildAPIRequestContext()`
2. Ensure REST route params are also merged (e.g., `/users/:id` → @params.id)

Tests:
- API POST with JSON body should NOT populate @params (documented edge case)
- API route params available in @params

---

### Task 6: Documentation updates
**Files**: `docs/guide/handlers.md`, `docs/parsley/reference.md`, `docs/parsley/CHEATSHEET.md`
**Estimated effort**: Medium

Steps:
1. Document @params in handler guide with examples
2. Document @env and @args in Parsley reference
3. Add to cheatsheet as "built-in globals"
4. Update FAQ with common patterns

---

### Task 7: Update examples
**Files**: `examples/search/index.pars`, other examples
**Estimated effort**: Small

Steps:
1. Update search example to use @params instead of import
2. Scan other examples for `import @basil/http` that could use @params
3. Keep imports where full request object is needed

---

## Validation Checklist
- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] pars CLI: @env and @args work
- [ ] Basil handlers: @params, @env, @args all work
- [ ] Documentation updated
- [ ] Examples updated

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| | | | |

## Deferred Items
Items to add to BACKLOG.md after implementation:
- @files global for file uploads (currently need import @basil/http for files)
- @headers global (debatable value, less common use case)
- @method shorthand (very short, maybe not worth a global)

## Implementation Notes

### Environment.Set with Dictionary
The `env.Set()` method accepts `Object` interface. Need to convert Go maps to Parsley `Dictionary` objects:

```go
// Convert Go map to Parsley Dictionary
func mapToDict(m map[string]interface{}) *evaluator.Dictionary {
    pairs := make(map[string]evaluator.Object)
    for k, v := range m {
        pairs[k] = nativeToObject(v)
    }
    return &evaluator.Dictionary{Pairs: pairs}
}
```

### Identifier Resolution
The evaluator already handles @ prefixed identifiers (e.g., `@DB`). Adding `@params`, `@env`, `@args` follows the same pattern - they're just regular variables in the environment that happen to start with @.

