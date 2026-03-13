# FEAT-140: Remove Per-Request Module Cache Clearing from API Handler

## Summary

Remove the `evaluator.ClearModuleCache()` call from `server/api.go` that currently runs on every API request. Investigation confirms this is unnecessary because the `DynamicAccessor` pattern already ensures request-scoped values (`@params`, `method`, `request`, `session`, `auth`, `user`) are resolved fresh for each request.

## Motivation

### Current Behavior
- `server/api.go:46` calls `evaluator.ClearModuleCache()` on **every** API request
- This clears all cached modules, forcing re-import and re-evaluation on subsequent requests
- Page handlers (`handler.go`) do **not** clear the module cache and work correctly

### Why Remove It
1. **Unnecessary**: The `DynamicAccessor` pattern in `@basil/http` and `@basil/auth` already handles request-scoped values correctly
2. **Inconsistent**: Page handlers don't clear the cache, yet work fine
3. **Performance**: While negligible for simple handlers, cache clearing adds overhead for handlers with many module imports
4. **Simplicity**: Less code, clearer architecture

### Evidence
- All existing tests pass with cache clearing disabled
- New comprehensive tests (`server/params_freshness_test.go`) verify params remain fresh across multiple requests without cache clearing
- The `DynamicAccessor.Resolve()` method uses the **current execution environment**, not cached values

## Technical Details

### How DynamicAccessor Works

When a module imports `@basil/http`:
```parsley
let {params, method} = import @basil/http
```

The `params` and `method` values are `DynamicAccessor` objects, not actual values. When accessed inside a function:

1. `ApplyFunctionWithEnv` copies `@params` and `BasilCtx` from the **caller's environment** to the function's execution environment
2. The `DynamicAccessor.Resolve(env)` method is called with this execution environment
3. The resolver walks the environment chain to find `@params`, which was set fresh by the handler

### Key Code Paths

**Environment propagation** (`eval_expressions.go:73-95`):
```go
func ApplyFunctionWithEnv(fn Object, args []Object, env *Environment) Object {
    extendedEnv := extendFunctionEnv(fn, args)
    if env != nil {
        extendedEnv.BasilCtx = env.BasilCtx      // Fresh from caller
        if params, ok := env.Get("@params"); ok {
            extendedEnv.Set("@params", params)   // Fresh from caller
        }
    }
}
```

**DynamicAccessor resolution** (`stdlib_table.go:242-251`):
```go
"params": &DynamicAccessor{
    Name: "params",
    Resolver: func(e *Environment) Object {
        if val, ok := e.Get("@params"); ok {
            return ensureObject(val)
        }
        return NULL
    },
},
```

### What Could Go Wrong (And Why It Doesn't)

**Concern**: User captures `params` at module scope
```parsley
let {params} = import @basil/http
let orderBy = params.orderBy  // Captured at module load time
```

**Protection**: This pattern produces error `UNDEF-0010` because `@params` is not available at module scope during import. The error message guides users to use params inside functions.

## Scope

### In Scope
- Remove `ClearModuleCache()` call from `server/api.go`
- Update comments to explain why module cache is preserved
- Verify all existing tests pass

### Out of Scope
- Changes to page handler behavior (already doesn't clear cache)
- Changes to `DynamicAccessor` implementation
- Changes to hot-reload behavior (`ReloadScripts` should still clear cache)
- Changes to file watcher behavior (`InvalidateModule` should still work)

## Acceptance Criteria

1. [x] `ClearModuleCache()` removed from `api.go` ServeHTTP method
2. [x] Comment added explaining why module cache is preserved for API requests
3. [x] All existing server tests pass
4. [x] All params freshness tests pass (`server/params_freshness_test.go`)
5. [x] All evaluator module cache tests pass (`pkg/parsley/tests/module_cache_test.go`)
6. [x] Hot-reload still works (verified via `TestHotReloadClearsModuleCache`)

## Risks

### Low Risk
- **Unknown user module patterns**: If any user has modules that incorrectly cache request data, they would see stale values. However:
  - The error system catches the most common mistake (module-scope capture)
  - Page handlers already work this way without issues
  - The pattern is documented in the safe module patterns guide

### Mitigation
- The change is easily reversible if issues are discovered
- Comprehensive test coverage validates the core scenarios

## Dependencies

- Investigation complete: #116 (backlog item)
- Tests added: `server/params_freshness_test.go` (commit dd557bc)
- Analysis documented: `work/reports/BASIL-PERFORMANCE-ANALYSIS-2026-03-10.md`

## References

- Backlog item #116: Review per-request module cache clearing
- Performance analysis: `work/reports/BASIL-PERFORMANCE-ANALYSIS-2026-03-10.md`
- Test file: `server/params_freshness_test.go`
- DynamicAccessor implementation: `pkg/parsley/evaluator/stdlib_table.go:84-102`
- Environment propagation: `pkg/parsley/evaluator/eval_expressions.go:73-95`
