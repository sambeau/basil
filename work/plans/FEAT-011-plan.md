# FEAT-011 Implementation Plan

**Feature**: Basil namespace for Parsley environment  
**Status**: Complete  
**Created**: 2025-12-01

## Overview
Move all Basil-injected variables into a `basil.*` namespace and add response header/status control.

## Tasks

### Phase 1: Core Implementation
- [x] **Task 1.1**: Create `buildBasilContext()` function in handler.go
  - Build nested `basil` object with http.request, http.response, auth, sqlite
  - Replace individual `setEnvVar` calls with single `basil` injection
  
- [x] **Task 1.2**: Update response handling in `writeResponse()`
  - Read `basil.http.response.status` from environment after execution
  - Read `basil.http.response.headers` and apply to response
  - Return value still used as body

- [x] **Task 1.3**: Handle database injection
  - Inject as `basil.sqlite` instead of `db`
  - Used `ast.ObjectLiteralExpression` to wrap DBConnection for Dictionary storage

### Phase 2: Update Examples
- [x] **Task 2.1**: Update `examples/hello/` handlers
  - No changes needed - hello example doesn't use request/db variables

- [x] **Task 2.2**: Update `examples/auth/` handlers  
  - Changed `request.user` to `basil.auth.user` in all handlers

### Phase 3: Documentation & Testing
- [x] **Task 3.1**: Add/update tests
  - Updated database tests to use `basil.sqlite`
  - Existing tests pass with new namespace
  
- [x] **Task 3.2**: Update documentation
  - Updated basil-quick-start.md
  - Updated authentication.md

## Files Modified
| File | Changes |
|------|---------|
| `server/handler.go` | New `buildBasilContext()`, `extractResponseMeta()`, response handling |
| `server/database_test.go` | Updated scripts to use `basil.sqlite` |
| `examples/auth/handlers/*.pars` | Changed `request.user` to `basil.auth.user` |
| `docs/guide/basil-quick-start.md` | Updated all examples to new namespace |
| `docs/guide/authentication.md` | Updated auth examples |

## Progress Log
| Date | Task | Notes |
|------|------|-------|
| 2025-12-01 | Tasks 1.1-1.3 | Core implementation complete |
| 2025-12-01 | Tasks 2.1-2.2 | Examples updated |
| 2025-12-01 | Tasks 3.1-3.2 | Tests and docs updated |

## Definition of Done
- [x] All tasks complete
- [x] Tests pass
- [x] Examples work
- [x] Documentation updated
- [x] `make check` passes
