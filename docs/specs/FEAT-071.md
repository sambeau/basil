---
id: FEAT-071
title: "Simplify @basil namespace for handler context"
status: completed
priority: high
created: 2025-12-15
author: "@human"
---

# FEAT-071: Simplify @basil namespace for handler context

## Summary
Replace the verbose `@std/basil` module with a new `@basil` namespace that provides direct, ergonomic access to HTTP request context in Basil handlers. Instead of `basil.http.request.query`, developers can use `import @basil/http` and access `query`, `route`, `method`, and `request` directly. Also fixes several bugs with query params and path objects.

## User Story
As a Basil developer, I want intuitive access to HTTP request data so that I can write concise handler code without navigating deep namespace hierarchies.

## Acceptance Criteria
- [x] `import @basil/http` provides `{request, response, query, route, method}`
- [x] `import @basil/auth` provides `{db, session}`
- [x] `import @basil` shows available modules via introspection (`.?`)
- [x] `import @std` shows available modules via introspection (`.?`)
- [x] Query params `?flag` (no value) return `true` instead of `""`
- [x] Path objects created via `ToParsley()` retain their type and methods (`.match()`, etc.)
- [x] `subpath` renamed to `route` in HTTP context
- [x] `import @std/basil` fails with helpful error message
- [x] Backlog item added to remove `@std/basil` error before Alpha

## Design Decisions
- **New `@basil` namespace**: Separates Basil server concerns from Parsley stdlib (`@std`). Semantically clearer: `@std` = language, `@basil` = server.
- **Direct exports from `@basil/http`**: Export `query`, `route`, `method` directly rather than requiring `request.query`. Keeps `request` for full access.
- **`route` not `subpath`**: "subpath" was hard to remember. "route" clearly indicates the matched portion of the path.
- **Query `?flag` → `true`**: Empty query values (no `=`) semantically mean "present" which is truthy. `""` was confusing.
- **Hard error for `@std/basil`**: Pre-alpha, so break things cleanly. Tests should fail to catch migrations.
- **Introspection for namespaces**: Makes discovery easy. `@basil.?` lists `http`, `auth`.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `pkg/parsley/evaluator/evaluator.go` — Add `basil/` prefix handling in `importModule()`, fix `isPathDict()` to handle `ObjectLiteralExpression`
- `pkg/parsley/evaluator/stdlib_table.go` — Add `loadBasilHTTPModule()`, `loadBasilAuthModule()`, `loadBasilRoot()`, `getBasilModules()` registry, remove `basil` from `getStdlibModules()`
- `server/handler.go` — Fix `queryToMap()` for valueless params, rename `subpath` → `route` in `buildBasilContext()`
- `pkg/parsley/evaluator/introspect.go` — Add introspection for `StdlibRoot` and new `BasilRoot` types

### Dependencies
- Depends on: None
- Blocks: None

### Edge Cases & Constraints
1. **No handler context** — When running outside Basil (CLI, tests), `@basil/http` returns empty/null values gracefully
2. **Path type detection** — `isPathDict()` currently checks for `*ast.StringLiteral` in `__type`, but `NewDictionaryFromObjects()` creates `*ast.ObjectLiteralExpression` wrapping `*String`. Must check both.
3. **Query array values** — `?foo=1&foo=2` should still work as arrays. Only `?flag` (no `=`) becomes `true`.
4. **Backwards compatibility** — None required (pre-alpha). `@std/basil` intentionally breaks.

## Implementation Notes
*Added during/after implementation*

## Related
- Plan: `docs/plans/FEAT-071-plan.md` (if created)
- Backlog: Remove `@std/basil` deprecation error before Alpha release
