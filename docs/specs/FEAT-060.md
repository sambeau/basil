---
id: FEAT-060
title: "Remove basil Global (Breaking Change)"
status: draft
priority: high
created: 2025-12-09
author: "@ai"
---

# FEAT-060: Remove basil Global (Breaking Change)

## Summary
Remove the `basil` global variable and require all scripts to use `import("std/basil")` instead. This eliminates API confusion, improves code clarity, and aligns with Parsley's explicit import model.

**Breaking Change Policy**: No deprecation period. Break things, fix things. This is a clean break during the pre-1.0 window.

## User Story
As a **Parsley developer**, I want **one clear way to access the basil context** so that **I don't have to remember whether to use the global or the import, and AI assistants can provide consistent examples**.

As an **AI assistant**, I want **a single canonical way to access basil** so that **I can provide accurate code examples without confusion**.

## Acceptance Criteria
- [ ] Remove `env.SetProtected("basil", basilObj)` from server handlers
- [ ] Keep `env.BasilCtx` for `std/basil` import support
- [ ] Update all examples to use `let {basil} = import("std/basil")`
- [ ] Update all server prelude templates to use import
- [ ] Update all documentation to use import
- [ ] Update all tests that rely on basil global
- [ ] Add migration guide to CHANGELOG
- [ ] Version bump (minor: 0.x.0 → 0.y.0)

## Design Decisions

- **No deprecation warning**: Clean break, fix everything in one commit. We're pre-1.0, now is the time.
- **Import-only access**: `std/basil` is the only way. Makes intent explicit, follows Parsley conventions.
- **Keep BasilCtx**: Internal mechanism stays; only the global injection is removed.
- **Single migration commit**: All examples, docs, tests updated together for atomic change.

## Rationale

**Current Problem:**
```parsley
// Two ways to access the same object:
basil.http.request.method           // Global (implicit)
let {basil} = import("std/basil")   // Import (explicit)
basil.http.request.method
```

This creates:
1. **Confusion** - Which one should I use?
2. **Inconsistency** - Examples use both, making docs/tutorials contradictory
3. **Testing issues** - Tests must set both `env.SetProtected()` and `env.BasilCtx`
4. **Maintenance burden** - Two code paths for the same functionality

**After This Change:**
```parsley
// One way:
let {basil} = import("std/basil")
basil.http.request.method
```

Clean, explicit, consistent with Parsley's import model.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components

**Code Changes:**
- `server/handler.go` — Remove `env.SetProtected("basil", basilObj)` line (keep BasilCtx)
- `server/api.go` — Remove `env.SetProtected("basil", basilObj)` line (keep BasilCtx)
- `server/errors.go` — Remove `env.Set("basil", basilObj)` in error templates

**Server Templates (7 files):**
- `server/prelude/devtools/index.pars` — Add import, use `basil.*`
- `server/prelude/devtools/env.pars` — Add import, use `basil.*`
- `server/prelude/errors/dev_error.pars` — Add import, use `basil.*`
- `server/prelude/errors/generic_error.pars` — Check if basil used

**Examples (7 files):**
- `examples/auth/handlers/dashboard.pars` — Add import
- `examples/auth/handlers/login.pars` — Add import
- `examples/auth/handlers/signup.pars` — Add import
- `examples/auth/handlers/logout.pars` — Add import
- `examples/auth/handlers/index.pars` — Add import
- `examples/auth/handlers/page.pars` — Add import
- `examples/cors/test-api.pars` — Add import

**Documentation (3 files):**
- `docs/guide/cors.md` — Update all code examples
- `docs/guide/basil-quick-start.md` — Update examples
- `docs/guide/README.md` — Check for basil references

**Tests (minimal changes expected):**
- Most tests use `BasilCtx` directly, not the global
- `pkg/parsley/tests/cache_test.go` — `<basil.cache.Cache>` tests (may need import in test code)
- `pkg/parsley/tests/public_dir_test.go` — Helper function sets basil dict
- `pkg/parsley/tests/database_test.go` — Sets `BasilCtx`
- `pkg/parsley/tests/stdlib_table_test.go` — Tests `std/basil` import

### Dependencies
- Depends on: None
- Blocks: None

### Edge Cases & Constraints

1. **Error pages** — Must work even if script errors before import
   - Solution: Error template environment has separate setup, inject `basil` for error context only
   - Or: Error templates also use import

2. **DevTools pages** — Need basil metadata (version, commit, etc.)
   - Solution: Add import at top of each template

3. **Components** — `<basil.cache.Cache>`, `<basil.auth.Login>`, etc.
   - These are special tags, not namespace access - work differently
   - But examples using them will need the import for `basil.auth.user` checks

4. **Breaking change communication**:
   - CHANGELOG must have clear migration instructions
   - Error message if someone tries to use `basil.*` without import (already handled by "undefined identifier")

### Migration Pattern

**Before:**
```parsley
if basil.auth.user {
  <h1>Welcome, {basil.auth.user.name}!</h1>
}
```

**After:**
```parsley
let {basil} = import("std/basil")

if basil.auth.user {
  <h1>Welcome, {basil.auth.user.name}!</h1>
}
```

Simple find-and-replace in most files:
1. Check if file uses `basil.*`
2. If yes, add `let {basil} = import("std/basil")` at top
3. Done

### Removed Code

```go
// server/handler.go - DELETE THIS LINE:
env.SetProtected("basil", basilObj)

// server/api.go - DELETE THIS LINE:
env.SetProtected("basil", basilObj)

// Keep these:
env.BasilCtx = basilObj          // Used by std/basil import
// ... other env settings ...
```

## Implementation Notes
*To be added during implementation*

## Related
- Backlog item: "Remove `basil` global in favor of `std/basil` import"
- Related: FEAT-011 (Basil namespace - introduced the dual system)
