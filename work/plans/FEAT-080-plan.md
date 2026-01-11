---
id: PLAN-050
feature: FEAT-080
title: "Implementation Plan for Decouple @DB from Request Context"
status: draft
created: 2026-01-04
---

# Implementation Plan: FEAT-080 Decouple @DB from Request Context

## Overview

Separate the database connection from per-request context, allowing `@DB` to be available at module load time. This enables schema bindings and model definitions at module scope rather than per-request.

## Prerequisites

- [ ] Understand current `@DB` resolution path in evaluator.go
- [ ] Understand `BasilCtx` structure and lifecycle
- [ ] Understand module caching behavior

---

## Phase 1: Add ServerDB to Environment

**Goal**: Add a new environment field for server-level database connection.

### Task 1.1: Add ServerDB Field to Environment
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Effort**: Small

Steps:
1. Add `ServerDB *DBConnection` field to `Environment` struct
2. Update `NewEnvironment()` to initialize `ServerDB` to nil
3. Update `NewEnclosedEnvironment()` to copy `ServerDB` from outer

```go
type Environment struct {
    // ... existing fields ...
    ServerDB      *DBConnection   // Server-level database connection
}
```

Acceptance Criteria:
- [ ] `Environment` has `ServerDB` field
- [ ] `NewEnvironment()` initializes it to nil
- [ ] `NewEnclosedEnvironment()` copies from outer

Tests:
- New environment has nil ServerDB
- Enclosed environment inherits ServerDB from outer

---

### Task 1.2: Propagate ServerDB to Module Environments
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Effort**: Small

Steps:
1. In `importModule()`, copy `ServerDB` to module environment
2. Find all places where new environments are created and ensure propagation

Location in `importModule()`:
```go
moduleEnv := NewEnvironment()
moduleEnv.ServerDB = env.ServerDB  // ADD THIS
moduleEnv.Filename = absPath
moduleEnv.RootPath = env.RootPath
// ... rest unchanged ...
```

Acceptance Criteria:
- [ ] Imported modules receive `ServerDB` from parent environment
- [ ] Nested imports also receive `ServerDB`

Tests:
- Module import receives ServerDB
- Nested module import receives ServerDB

---

## Phase 2: Update @DB Resolution

**Goal**: Make `@DB` resolve from `ServerDB` first, then fall back to `BasilCtx`.

### Task 2.1: Update evalDBLiteral
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Effort**: Small

Steps:
1. Find `evalDBLiteral` function (around line 2360)
2. Add check for `env.ServerDB` at the start
3. Keep existing `BasilCtx` resolution as fallback
4. Update error message

Before:
```go
func evalDBLiteral(node *ast.DBLiteral, env *Environment) Object {
    // ... gets basilObj from env.BasilCtx ...
    // ... looks up basilDict.Pairs["sqlite"] ...
    // ... error if not found ...
}
```

After:
```go
func evalDBLiteral(node *ast.DBLiteral, env *Environment) Object {
    // 1. Try server-level database first
    if env.ServerDB != nil {
        return env.ServerDB
    }
    
    // 2. Fall back to BasilCtx["sqlite"] (backward compatibility)
    // ... existing code ...
    
    // 3. Updated error message
    return &Error{
        Class:   ErrorClass("state"),
        Message: "@DB is only available in Basil server context",
        Hints:   []string{"Run inside a Basil handler or module with a configured database"},
    }
}
```

Acceptance Criteria:
- [ ] `@DB` returns `ServerDB` when available
- [ ] `@DB` falls back to `BasilCtx["sqlite"]` when `ServerDB` is nil
- [ ] Error message says "Basil server context" not "Basil server handlers"

Tests:
- @DB with ServerDB set returns ServerDB
- @DB without ServerDB falls back to BasilCtx
- @DB without either returns updated error message

---

## Phase 3: Update Server Handler Setup

**Goal**: Set `ServerDB` before handler evaluation so modules can access it.

### Task 3.1: Set ServerDB in Handler
**Files**: `server/handler.go`
**Effort**: Small

Steps:
1. In `ServeHTTP()`, set `env.ServerDB` before evaluating script
2. Create DBConnection from `h.server.db` and `h.server.dbDriver`

Location (around line 234, after creating env):
```go
env := evaluator.NewEnvironment()
env.Filename = h.scriptPath

// Set server-level database (available to modules at load time)
if h.server.db != nil {
    env.ServerDB = evaluator.NewManagedDBConnection(h.server.db, h.server.dbDriver)
}
```

Acceptance Criteria:
- [ ] Handler environment has `ServerDB` set before evaluation
- [ ] `ServerDB` is nil when server has no database configured

Tests:
- Handler with database has ServerDB set
- Handler without database has nil ServerDB

---

### Task 3.2: Set ServerDB in API Handler
**Files**: `server/api.go`
**Effort**: Small

Steps:
1. Same change as 3.1 but in API handler

Location:
```go
env := evaluator.NewEnvironment()
// ... existing setup ...

// Set server-level database
if h.server.db != nil {
    env.ServerDB = evaluator.NewManagedDBConnection(h.server.db, h.server.dbDriver)
}
```

Acceptance Criteria:
- [ ] API handler environment has `ServerDB` set

Tests:
- API handler with database has ServerDB set

---

### Task 3.3: Set ServerDB in Error Handler
**Files**: `server/errors.go`
**Effort**: Small

Steps:
1. Same change in error handler's environment setup

Acceptance Criteria:
- [ ] Error handler environment has `ServerDB` set

Tests:
- Error handler with database has ServerDB set

---

### Task 3.4: Set ServerDB in DevTools Handler
**Files**: `server/devtools.go`
**Effort**: Small

Steps:
1. Same change in devtools handler's environment setup

Acceptance Criteria:
- [ ] DevTools handler environment has `ServerDB` set

Tests:
- DevTools handler with database has ServerDB set

---

## Phase 4: Remove Module Cache Clearing

**Goal**: Stop clearing module cache per-request, allowing modules to be cached with their `@DB` bindings.

### Task 4.1: Remove ClearModuleCache Call
**Files**: `server/handler.go`
**Effort**: Small

Steps:
1. Remove or comment out `evaluator.ClearModuleCache()` call
2. Add comment explaining why module cache is preserved

Before:
```go
// Clear module cache so imports see fresh basil.* values for this request
evaluator.ClearModuleCache()
```

After:
```go
// Module cache is preserved across requests.
// Modules should NOT store request-specific values (basil.http.request) at module scope.
// Server resources (@DB, schemas) are cached for performance.
```

Acceptance Criteria:
- [ ] `ClearModuleCache()` not called per-request
- [ ] Comment explains the caching strategy

Tests:
- Module imported twice in same request returns same cached module
- Module imported across requests returns cached module

---

### Task 4.2: Document Module Caching Behavior
**Files**: `docs/guide/modules.md` or similar
**Effort**: Small

Steps:
1. Add documentation about module caching
2. Warn against storing request values at module scope
3. Show correct pattern for request-dependent code

```markdown
## Module Caching

Modules are cached after first load. This means:

✅ **DO** — Define schemas and table bindings at module scope:
```parsley
// models.pars — Executed once, cached
let Users = schema.table(User, @DB, "users")
export Users
```

❌ **DON'T** — Store request values at module scope:
```parsley
// BAD — request is captured once, stale for subsequent requests
let currentUser = basil.auth.user  // WRONG!
```

✅ **DO** — Access request values inside functions:
```parsley
// GOOD — evaluated fresh each call
let getCurrentUser = fn() { basil.auth.user }
```
```

Acceptance Criteria:
- [ ] Documentation explains module caching
- [ ] Examples show correct and incorrect patterns

---

## Phase 5: Optional Cleanup

**Goal**: Remove sqlite from `BasilCtx` (optional, for cleaner architecture).

### Task 5.1: Remove sqlite from buildBasilContext
**Files**: `server/handler.go`
**Effort**: Small

Steps:
1. Remove the code that adds sqlite to `BasilCtx`
2. This is optional since `ServerDB` takes priority anyway

Before:
```go
// Add database connection if configured
if db != nil {
    conn := evaluator.NewManagedDBConnection(db, dbDriver)
    basilDict.Pairs["sqlite"] = &ast.ObjectLiteralExpression{Obj: conn}
}
```

After:
```go
// Database connection is at env.ServerDB, not in BasilCtx
// (sqlite removed - was redundant with ServerDB)
```

Acceptance Criteria:
- [ ] `BasilCtx` no longer contains sqlite
- [ ] All code still works via `ServerDB`

Tests:
- Existing @DB usage still works
- basil.sqlite is no longer available (breaking change warning)

**Note**: This task may be deferred if we want to maintain `basil.sqlite` as an alias.

---

## Validation Checklist

### Per-Task Validation
After each task:
- [ ] `go test ./...` — All tests pass
- [ ] `go build -o basil ./cmd/basil` — Build succeeds

### Final Validation
- [ ] All acceptance criteria from FEAT-080 checked off
- [ ] Module-scope @DB works (manual test)
- [ ] `schema.table()` at module scope works (manual test)
- [ ] Existing handlers still work (backward compatibility)
- [ ] Error message updated

---

## Test Plan

### Unit Tests

| Test | File | Description |
|------|------|-------------|
| TestServerDBInEnvironment | `evaluator/evaluator_test.go` | ServerDB field exists and propagates |
| TestDBLiteralWithServerDB | `tests/database_test.go` | @DB resolves from ServerDB |
| TestDBLiteralFallback | `tests/database_test.go` | @DB falls back to BasilCtx |
| TestDBLiteralError | `tests/database_test.go` | Error message updated |
| TestModuleInheritsServerDB | `tests/modules_test.go` | Imported modules get ServerDB |

### Integration Tests

| Test | Description |
|------|-------------|
| Module-scope @DB | Module that uses @DB at top level works |
| Module-scope schema.table | Model module with table binding works |
| Handler imports model | Handler uses pre-bound table from module |
| Multiple requests | Cached module works across requests |
| No database | @DB error message is clear when no DB |

### Manual Tests

1. Create `models.pars` with schema and `schema.table()` at module scope
2. Create handler that imports and uses the model
3. Make multiple requests, verify caching works
4. Verify @DB error in standalone `pars` script

---

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-01-04 | Task 1.1: Add ServerDB field | ✅ Complete | Added to Environment struct |
| 2026-01-04 | Task 1.2: NewEnclosedEnvironment | ✅ Complete | Propagates ServerDB from outer |
| 2026-01-04 | Task 1.2: importModule propagation | ✅ Complete | Modules inherit ServerDB |
| 2026-01-04 | Task 2.1: Update resolveDBLiteral | ✅ Complete | ServerDB checked first, fallback to BasilCtx |
| 2026-01-04 | Task 3.1: handler.go ServerDB | ✅ Complete | Set before evaluation |
| 2026-01-04 | Task 3.2: api.go ServerDB | ✅ Complete | Set before evaluation |
| 2026-01-04 | Task 4.1: Remove ClearModuleCache | ✅ Complete | Module cache preserved across requests |
| 2026-01-04 | Tests: database_test.go | ✅ Complete | Added ServerDB tests, updated error message |

---

## Deferred Items

Items to add to BACKLOG.md after implementation:

1. **basil.sqlite alias** — Decide if `basil.sqlite` should remain as alias for @DB
2. **Multiple databases** — Add `env.ServerDBs["name"]` for named connections
3. **Module cache invalidation** — Selective cache clearing on file change
4. **HUP signal handling** — Clear module cache and recreate ServerDB on reload

---

## Related Documents

- Specification: [FEAT-080.md](../specs/FEAT-080.md)
- Depends on: None
- Unblocks: FEAT-079 (Query DSL module-scope bindings)
- Related: FEAT-034 (Schema Validation), FEAT-078 (TableBinding API)
