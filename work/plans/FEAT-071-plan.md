---
id: PLAN-045
feature: FEAT-071
title: "Implementation Plan for @basil Namespace"
status: completed
created: 2025-12-15
---

# Implementation Plan: FEAT-071

## Overview
Replace `@std/basil` with a new `@basil` namespace providing ergonomic access to HTTP context. This is a **breaking change** — `@std/basil` will error immediately, not deprecate gracefully. Pre-alpha means we break things cleanly.

## Prerequisites
- [x] FEAT-071 spec approved
- [x] Decision: Break `@std/basil` with error (not deprecation)

## Tasks

### Task 1: Add `@basil` namespace support in import resolution
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small

Steps:
1. In `importModule()`, add handling for `basil` root (like `std` root)
2. Add handling for `basil/` prefix to route to basil module loaders
3. Return `loadBasilRoot()` for bare `import @basil`
4. Route `basil/http` to `loadBasilHTTPModule()`
5. Route `basil/auth` to `loadBasilAuthModule()`

Tests:
- `import @basil` returns BasilRoot with module list
- `import @basil/http` returns module with exports
- `import @basil/auth` returns module with exports
- `import @basil/unknown` returns error

---

### Task 2: Create BasilRoot and basil module registry
**Files**: `pkg/parsley/evaluator/stdlib_table.go`
**Estimated effort**: Medium

Steps:
1. Create `BasilRoot` type (like `StdlibRoot`) for introspection
2. Create `getBasilModules()` registry function returning `{"http": loadBasilHTTPModule, "auth": loadBasilAuthModule}`
3. Create `loadBasilRoot()` returning `BasilRoot` with module names
4. Create `loadBasilModule()` to dispatch to module loaders

Tests:
- `BasilRoot.Inspect()` shows available modules
- Module loading works for valid names
- Error for invalid module names

---

### Task 3: Create `@basil/http` module
**Files**: `pkg/parsley/evaluator/stdlib_table.go`
**Estimated effort**: Medium

Steps:
1. Create `loadBasilHTTPModule(env)` function
2. Extract `request`, `response` from `env.BasilCtx`
3. Extract `query` from request (shorthand)
4. Extract `route` from request (renamed from `subpath`)
5. Extract `method` from request (shorthand)
6. Return `StdlibModuleDict` with exports: `{request, response, query, route, method}`
7. Handle missing context gracefully (CLI/test mode)

Tests:
- `let {request} = import @basil/http` works
- `let {query, route, method} = import @basil/http` works
- Shorthand values match nested values
- Works outside handler context (returns null/empty)

---

### Task 4: Create `@basil/auth` module
**Files**: `pkg/parsley/evaluator/stdlib_table.go`
**Estimated effort**: Small

Steps:
1. Create `loadBasilAuthModule(env)` function
2. Extract `db`, `session` from `env.BasilCtx`
3. Return `StdlibModuleDict` with exports: `{db, session}`
4. Handle missing context gracefully

Tests:
- `let {session} = import @basil/auth` works
- Works outside handler context

---

### Task 5: Break `@std/basil` with error
**Files**: `pkg/parsley/evaluator/stdlib_table.go`
**Estimated effort**: Small

**NOTE**: This is intentionally breaking, not deprecating. Tests should fail. Code must be updated.

Steps:
1. Remove `"basil"` entry from `getStdlibModules()` registry
2. Add special case in `loadStdlibModule()` that checks for `"basil"` and returns a clear error: `"@std/basil has been removed. Use @basil/http or @basil/auth instead."`
3. Delete `loadBasilModule()` function (the old one)

Tests:
- `import @std/basil` returns error with migration message
- Error message mentions `@basil/http` and `@basil/auth`

---

### Task 6: Fix query params `?flag` → `true`
**Files**: `server/handler.go`
**Estimated effort**: Small

Steps:
1. Find `queryToMap()` function
2. When iterating query values, check if value is `[""]` (single empty string)
3. If so, set value to `true` instead of `""`
4. Keep array handling for `?foo=1&foo=2`

Tests:
- `?flag` → `{flag: true}`
- `?flag=` → `{flag: ""}` (explicit empty)
- `?flag=value` → `{flag: "value"}`
- `?flag=1&flag=2` → `{flag: ["1", "2"]}`
- `?a&b=1&c` → `{a: true, b: "1", c: true}`

---

### Task 7: Rename `subpath` to `route`
**Files**: `server/handler.go`
**Estimated effort**: Small

Steps:
1. Find `buildBasilContext()` function
2. Change key from `"subpath"` to `"route"`
3. Update any comments referencing subpath

Tests:
- `request.route` contains the matched route portion
- `request.subpath` no longer exists (or errors)

---

### Task 8: Fix path type preservation through ToParsley
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small

Steps:
1. Find `isPathDict()` function
2. Current code checks for `*ast.StringLiteral` in `Pairs["__type"]`
3. Add fallback: also check for `*ast.ObjectLiteralExpression` containing `*String`
4. Apply same fix to `isUrlDict()` and `isFileDict()`

Tests:
- Path created in Go, passed through ToParsley, has `.match()` method
- `route.match("/users/:id")` works in handlers
- URL and File dictionaries also preserve their types

---

### Task 9: Add introspection for @basil and @std roots
**Files**: `pkg/parsley/evaluator/introspect.go`
**Estimated effort**: Medium

Steps:
1. Find introspection handling for `StdlibRoot`
2. Add case for `StdlibRoot` showing available modules
3. Add case for `BasilRoot` showing available modules
4. Format output nicely: `@std { table, math, valid, ... }`

Tests:
- `@std.?` shows module list
- `@basil.?` shows `http`, `auth`
- Output is readable and helpful

---

## Validation Checklist
- [x] All tests pass: `make test`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [x] Example apps updated to use `@basil/http`
- [x] BACKLOG.md updated with deferral item

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-12-15 | Tasks 1-9 | ✅ Complete | Implemented @basil namespace, removed @std/basil, query flags true, route rename, type detection, introspection, examples/tests updated |

## Deferred Items
Items to add to BACKLOG.md after implementation:
- **Remove `@std/basil` error before Alpha** — Currently returns helpful migration error; should be removed entirely before Alpha release so it's just "unknown module"
