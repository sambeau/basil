---
id: FEAT-012
title: "Remove path() builtin function"
status: done
priority: medium
created: 2025-12-02
author: "@human"
---

# FEAT-012: Remove path() builtin function

## Summary
Remove the `path()` builtin function from Parsley since `@({...})` path template syntax provides identical functionality, and "path" is a commonly needed variable name in web contexts.

## User Story
As a Parsley developer building web applications, I want to use `path` as a variable name (e.g., for URL paths) without shadowing or conflicting with a builtin function.

## Acceptance Criteria
- [x] `path()` builtin is removed from Parsley
- [x] `path` can be used as a variable name without issues
- [x] Existing path functionality still works via `@({...})` syntax
- [x] Tests updated to use `@` syntax
- [x] Any internal uses of `path()` updated to use `@({...})`

## Migration Guide

| Before | After |
|--------|-------|
| `path("/usr/bin")` | `@/usr/bin` or `@({"/usr/bin"})` |
| `path(someVar)` | `@({someVar})` |
| `path("/users/" + name)` | `@(/users/{name})` |

## Design Decisions

### Why Remove?
1. **Redundancy**: `@({...})` does everything `path()` does
2. **Name collision**: `path` is extremely common in web code (URL paths, file paths, route paths)
3. **Consistency**: Other literals use `@` prefix (`@/path`, `@https://...`, `@2024-01-01`)

### Breaking Change
This is a breaking change. Any code using `path()` will need to be updated. However:
- Parsley is pre-1.0, breaking changes are acceptable
- The migration is mechanical and straightforward
- The `@({...})` syntax has been available, so this is just removing the redundant builtin

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Code
- `pkg/parsley/evaluator/evaluator.go` — Remove `path` from builtins map (~line 4241)
- Any tests that use `path()` builtin
- Documentation/examples

### Implementation Notes
The `path()` builtin at line 4241-4256 simply:
1. Takes a string argument
2. Parses it with `parsePathString()`
3. Returns a path dictionary via `pathToDict()`

This is exactly what `@({...})` does via `evalPathTemplateLiteral()`.

## Related
- FEAT-011: basil namespace (introduced `basil.http.request.path`)
