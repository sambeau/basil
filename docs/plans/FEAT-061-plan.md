---
id: PLAN-037
feature: FEAT-061
title: "Implementation Plan: Parts (Reloadable HTML Fragments)"
status: complete
created: 2025-12-10
completed: 2025-12-10
---

# Implementation Plan: FEAT-061 Parts

## Overview

Implement Parts — reloadable HTML fragments with multiple views. This involves:
1. Recognizing `.part` files as Part modules
2. Implementing the `<Part/>` component
3. Adding server-side Part request handling
4. Auto-injecting the JavaScript runtime

## Prerequisites

- [ ] Design document reviewed: `docs/design/DESIGN-parts.md`
- [ ] Spec approved: `docs/specs/FEAT-061.md`
- [ ] Understand existing module loading in `pkg/parsley/evaluator/`
- [ ] Understand existing component handling in evaluator
- [ ] Understand Basil server request handling in `server/`

## Tasks

### Task 1: Part Module Recognition
**Files**: `pkg/parsley/evaluator/evaluator.go`, `pkg/parsley/evaluator/module.go`
**Estimated effort**: Small

Extend module loading to recognize `.part` files as a special module type.

Steps:
1. Add `PART_MODULE` type constant alongside existing module types
2. In module resolution, detect `.part` extension
3. Load `.part` files same as `.pars` but tag as Part module
4. Store Part modules in module cache with Part flag

Tests:
- Load a `.part` file as module
- Verify exports are accessible
- Verify Part flag is set

---

### Task 2: `<Part/>` Component Implementation
**Files**: `pkg/parsley/evaluator/components.go` (or similar)
**Estimated effort**: Medium

Implement the `<Part/>` built-in component that renders Parts.

Steps:
1. Register `Part` as a built-in component
2. Extract `src` prop (required, path literal)
3. Extract `view` prop (optional, default "default")
4. Extract remaining props for view function
5. Resolve Part module from `src` path
6. Look up view function from module exports
7. Call view function with props
8. Wrap result in `<div data-part-src="..." data-part-view="..." data-part-props='...'>...</div>`
9. Track that page contains Parts (for JS injection)

Tests:
- `<Part src=@./counter.part/>` renders default view
- `<Part src=@./counter.part view="edit"/>` renders edit view
- Props passed to view function correctly
- Wrapper div has correct data attributes
- Error on missing `src`
- Error on non-existent Part file
- Error on non-existent view

---

### Task 3: Part Attribute Output
**Files**: `pkg/parsley/evaluator/html.go` (or tag evaluation)
**Estimated effort**: Small

Ensure `part-*` attributes are rendered correctly in HTML output.

Steps:
1. `part-click`, `part-submit` render as-is (not transformed)
2. `part-{propname}` attributes render with values
3. Values are properly escaped for HTML attributes

Tests:
- `<button part-click="edit">` outputs correctly
- `<button part-click="edit" part-id={123}>` outputs `part-id="123"`
- Special characters in values are escaped

---

### Task 4: Server Part Request Handler
**Files**: `server/handler.go`, `server/parts.go` (new)
**Estimated effort**: Large

Handle Part view requests from the JavaScript runtime.

Steps:
1. Create `server/parts.go` with Part request handler
2. In main handler, detect Part requests (has `_view` param + resolves to `.part` file)
3. Return 404 for direct `.part` file requests (no `_view`)
4. Parse `_view` param to get view name
5. Parse remaining query params as props
6. For POST, parse form body and merge with query params
7. Apply type coercion to props (same as form handling)
8. Load Part module
9. Look up and call view function
10. Return HTML fragment (no wrapper — JS replaces innerHTML)
11. Ensure auth/session context is available

Tests:
- `GET /_parts/counter?_view=default` returns HTML
- `GET /path/to/counter.part` returns 404
- `POST /_parts/todo?_view=save` with form body works
- Type coercion: `count=5` → number, `active=true` → boolean
- Missing view returns 404
- Auth cookies are validated

---

### Task 5: Part URL Generation
**Files**: `pkg/parsley/evaluator/components.go`, `server/parts.go`
**Estimated effort**: Medium

Generate the correct URL for Part requests in `data-part-src`.

Steps:
1. Determine URL scheme for Parts (e.g., `/_parts/` prefix or path-based)
2. Convert file path to Part URL in `<Part/>` component
3. In server, reverse the URL to file path
4. Handle both file-routed and single-handler modes
5. Ensure relative paths in nested Parts resolve correctly

Tests:
- `src=@./counter.part` from `/dashboard` → correct URL
- `src=@./parts/item.part` → correct URL
- Nested Part URLs resolve correctly
- Single-handler mode works

---

### Task 6: JavaScript Runtime Injection
**Files**: `server/handler.go`, `server/livereload.go` (or new `server/parts_js.go`)
**Estimated effort**: Medium

Auto-inject the Parts JavaScript runtime when a page contains `<Part/>`.

Steps:
1. Track whether response contains Parts (flag during render)
2. If Parts present, inject `<script>` before `</body>` or at end
3. Store JS runtime as embedded string or file
4. Ensure JS is only injected once per page
5. Consider: inject inline vs external `/basil/parts.js`

Tests:
- Page with `<Part/>` includes JS runtime
- Page without `<Part/>` does not include JS
- JS is injected only once even with multiple Parts
- JS is valid and executes without errors

---

### Task 7: Nested Parts Support
**Files**: `pkg/parsley/evaluator/components.go`
**Estimated effort**: Small

Ensure Parts containing other Parts work correctly.

Steps:
1. When rendering a Part, its content may contain `<Part/>` components
2. Each nested Part gets its own wrapper div
3. JS `init()` function handles nested Parts after refresh
4. Verify no infinite loops possible

Tests:
- Part containing Part renders correctly
- After parent refresh, child Parts are initialized
- Deeply nested Parts work (3+ levels)

---

### Task 8: Error Handling
**Files**: `server/parts.go`, JS runtime
**Estimated effort**: Small

Implement error handling for Part operations.

Steps:
1. Server: Return appropriate HTTP status codes (404, 500)
2. Server: Log errors for debugging
3. JS: On fetch error, leave old content in place
4. JS: Remove `data-part-loading` class on error

Tests:
- Network error leaves old content
- 404 response leaves old content
- 500 response leaves old content
- Loading class removed on error

---

### Task 9: Documentation
**Files**: `docs/guide/`, `docs/parsley/CHEATSHEET.md`
**Estimated effort**: Small

Document Parts for users.

Steps:
1. Add Parts section to user guide
2. Add Parts examples to cheatsheet
3. Update any relevant FAQ entries
4. Add example Part to `examples/` directory

Tests:
- Documentation renders correctly
- Examples are valid Parsley code

---

## Validation Checklist

- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] Manual test: Counter Part increments
- [ ] Manual test: Form Part submits and updates
- [ ] Manual test: Nested Parts work
- [ ] Manual test: Loading class appears during fetch
- [ ] Manual test: Animation classes work with CSS
- [ ] Documentation updated
- [ ] BACKLOG.md updated with deferrals

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-12-10 | Task 1: Part Module Recognition | ✅ Complete | Implemented in evaluator.go, validates function-only exports |
| 2025-12-10 | Task 2: `<Part/>` Component | ✅ Complete | Renders with data-part-* attributes |
| 2025-12-10 | Task 3: Part Attribute Output | ✅ Complete | data-part-src, data-part-view, data-part-props |
| 2025-12-10 | Task 4: Server Part Handler | ✅ Complete | handlePartRequest in server/parts.go |
| 2025-12-10 | Task 5: Part URL Generation | ✅ Complete | convertPathToPartURL using handler route path |
| 2025-12-10 | Task 6: JS Runtime Injection | ✅ Complete | Auto-injects when ContainsParts flag is true |
| 2025-12-10 | Task 7: Nested Parts Support | ✅ Complete | Tested with comprehensive test cases |
| 2025-12-10 | Task 8: Error Handling | ✅ Complete | Server 400/404/500, JS graceful fallback |
| 2025-12-10 | Task 9: Example | ✅ Complete | examples/parts/ with counter demo |
| 2025-12-10 | Task 10: Documentation | ✅ Complete | Updated CHEATSHEET, reference, FAQ, created parts.md guide |

**Additional Fixes:**
- Fixed JS runtime to collect part-* attributes from clicked elements (props weren't being passed)
- Fixed Part URL generation to use handler route path as base (404 errors resolved)

## Deferred Items

Items to add to BACKLOG.md after V1 implementation:

- `part-refresh={ms}` — Auto-refresh for live data (V1.1)
- `part-load="view"` — Lazy loading support (V1.1)
- Responsive Parts with media queries (V1.2)
- Target other Parts on page (V1.2)
- Part response caching — Complex cache invalidation issues
- `export error` view convention — Custom error states
- `animate` attribute for preset animations

## Implementation Order

Recommended order to minimize dependencies:

1. **Task 1**: Part Module Recognition — Foundation for everything else
2. **Task 3**: Part Attribute Output — Needed for component to work
3. **Task 2**: `<Part/>` Component — Core functionality
4. **Task 5**: Part URL Generation — Needed for server handler
5. **Task 4**: Server Part Handler — Enables runtime updates
6. **Task 6**: JS Runtime Injection — Makes Parts interactive
7. **Task 7**: Nested Parts Support — Verify composition works
8. **Task 8**: Error Handling — Polish
9. **Task 9**: Documentation — Final step

## Notes

- Start with a simple counter Part for testing throughout
- The JS runtime can be developed/tested in isolation with static HTML
- Consider adding a `--parts-debug` flag for verbose logging during development
