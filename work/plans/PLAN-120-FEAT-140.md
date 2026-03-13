# PLAN-120: Remove Per-Request Module Cache Clearing (FEAT-140)

## Overview

| Field | Value |
|-------|-------|
| Feature | FEAT-140 |
| Created | 2026-03-13 |
| Status | Ready |
| Complexity | Low |
| Risk | Low |

## Summary

Remove the unnecessary `evaluator.ClearModuleCache()` call from `server/api.go`. Investigation confirms the `DynamicAccessor` pattern already ensures request-scoped values remain fresh without clearing the module cache on every request.

## Prerequisites

- [x] Investigation complete (#116)
- [x] Params freshness tests added (`server/params_freshness_test.go`)
- [x] Analysis documented in performance report

## Implementation Steps

### Step 1: Remove ClearModuleCache Call

**File**: `server/api.go`

**Change**: Remove line 46 (`evaluator.ClearModuleCache()`) and add explanatory comment.

**Before**:
```go
func (h *apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	program, err := h.cache.getAST(h.scriptPath)
	if err != nil {
		h.server.logError("failed to load script: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	evaluator.ClearModuleCache()

	reqCtx := buildAPIRequestContext(r, h.route)
```

**After**:
```go
func (h *apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	program, err := h.cache.getAST(h.scriptPath)
	if err != nil {
		h.server.logError("failed to load script: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Module cache is preserved across requests for performance.
	// Request-scoped values (@params, method, session, auth, user) are handled
	// by DynamicAccessor which resolves fresh values from the current environment.
	// See: pkg/parsley/evaluator/stdlib_table.go (DynamicAccessor implementation)

	reqCtx := buildAPIRequestContext(r, h.route)
```

### Step 2: Run Tests

```bash
# Run all server tests
go test ./server/... -count=1

# Run specific params freshness tests
go test ./server -run "ParamsFreshness" -v

# Run module cache tests
go test ./pkg/parsley/tests -run "ModuleCache|DynamicAccessor|Params" -v
```

### Step 3: Manual Verification

1. Start a test server with an API route that uses `@basil/http.params`
2. Make multiple requests with different query parameters
3. Verify each request returns correct (fresh) values
4. Test hot-reload still works (modify handler file, verify changes take effect)

## Verification Checklist

- [ ] `go test ./server/...` passes
- [ ] `go test ./pkg/parsley/...` passes
- [ ] Params freshness tests pass without cache clearing
- [ ] Hot-reload still clears cache on file change (via `ReloadScripts`)
- [ ] File watcher still invalidates changed modules (via `InvalidateModule`)

## Rollback Plan

If issues are discovered after merge:

1. Restore the `ClearModuleCache()` call in `api.go`
2. Single line change, easily reversible

## Test Coverage

### Existing Tests (must pass)
- `TestAPIRouteMapping` - API routing with params
- `TestModuleCacheClear` - Cache clearing functionality
- `TestDynamicAccessorInCachedModule` - DynamicAccessor with cached modules
- `TestParamsDynamicAccessorInModule` - Params via @basil/http

### New Tests (already added)
- `TestAPIParamsFreshnessAcrossRequests` - 5 sequential requests with different params
- `TestAPIParamsFreshnessWithMethod` - GET/POST method detection stays fresh
- `TestPageHandlerParamsFreshness` - Page handler params (reference, already works)
- `TestAPIParamsWithAtParamsDirect` - Direct @params usage in functions

## Commit Message

```
feat(server): remove unnecessary per-request module cache clearing

Remove ClearModuleCache() call from api.go ServeHTTP. Investigation
confirms this is unnecessary because:

1. DynamicAccessor pattern resolves request-scoped values fresh from
   the current execution environment, not from cached values
2. ApplyFunctionWithEnv propagates @params and BasilCtx from the
   caller's environment to function execution environments
3. Page handlers already don't clear the cache and work correctly
4. All tests pass without cache clearing

Benefits:
- Consistent behavior between API and page handlers
- Potential performance improvement for module-heavy handlers
- Simpler code path

The module cache is still cleared appropriately by:
- ReloadScripts() for hot-reload
- InvalidateModule() for file watcher changes

Closes: FEAT-140
Related: #116 (backlog investigation item)
```

## Time Estimate

| Task | Estimate |
|------|----------|
| Code change | 5 minutes |
| Test verification | 10 minutes |
| Manual testing | 10 minutes |
| **Total** | ~25 minutes |

## Notes

- This is a low-risk change because:
  - All tests pass without the cache clearing
  - Page handlers already work without it
  - The change is easily reversible
  - Comprehensive test coverage validates the core scenarios
- Performance impact is negligible for simple handlers but could matter for handlers importing many modules with expensive initialization