---
id: PLAN-022
feature: FEAT-034
title: "Implementation Plan for std/api Module"
status: in-progress
created: 2025-12-06
updated: 2025-01-18
---

# Implementation Plan: FEAT-034 (std/api Module)

## Overview

Implement the `std/api` module for Basil, providing schema-based validation, database table binding, authentication wrappers, ID generation, and sensible defaults for building JSON APIs.

## Progress Summary

| Phase | Status | Notes |
|-------|--------|-------|
| Phase 1: Core Schema System | ✅ DONE | Type factories, define, validate |
| Phase 2: ID Generation | ✅ DONE | ULID, UUID, NanoID, CUID |
| Phase 3: Table Binding | ⏸️ DEFERRED | Complex, needs db integration |
| Phase 4: API Routes | ⏸️ DEFERRED | Needs Basil handler changes |
| Phase 5: Authentication | ✅ DONE | Auth wrappers, error helpers |
| Phase 6: Sensible Defaults | ⏸️ DEFERRED | Rate limiting, pagination |

## Implementation Notes

### Completed Work (2025-01-18)

**Files Created:**
- `pkg/parsley/evaluator/stdlib_schema.go` - Schema types and validation
- `pkg/parsley/evaluator/stdlib_id.go` - ID generation (ULID, UUID, NanoID, CUID)
- `pkg/parsley/evaluator/stdlib_api.go` - Auth wrappers and error helpers
- `pkg/parsley/tests/stdlib_schema_test.go` - Schema tests
- `pkg/parsley/tests/stdlib_id_test.go` - ID tests  
- `pkg/parsley/tests/stdlib_api_test.go` - API tests

**Files Modified:**
- `pkg/parsley/evaluator/stdlib_table.go` - Registered schema, id, api modules
- `pkg/parsley/evaluator/evaluator.go` - Added AuthWrappedFunction support, API_ERROR_OBJ type

**Key Decisions:**
- Auth uses wrapper pattern: `public(fn(req) {...})` not method style
- APIError is its own type (API_ERROR_OBJ) not ERROR_OBJ to avoid halting execution
- ID default is ULID (time-sortable, 26 chars, Crockford Base32)

## Prerequisites

- [x] Existing auth system working (FEAT-004)
- [x] SQLite database support working (FEAT-002)
- [x] Review `work/design/api-design-summary.md` for full context

## Phase 1: Core Schema System ✅ COMPLETED

### Task 1.1: Schema Type Functions
**Files**: `pkg/parsley/stdlib/stdlib_schema.go`
**Estimated effort**: Large

Steps:
1. Create `stdlib_schema.go` with module registration
2. Implement type factory functions:
   - `schema.string({required, min, max, pattern})`
   - `schema.email({required})`
   - `schema.url({required, protocols})`
   - `schema.phone({required})`
   - `schema.integer({required, min, max})`
   - `schema.number({required, min, max})`
   - `schema.boolean({default})`
   - `schema.enum({values, default})`
   - `schema.date({required, min, max})`
   - `schema.datetime({required})`
   - `schema.money({required, currency})`
   - `schema.array({of, min, max})`
   - `schema.object({properties})`
3. Each type returns a dictionary with: `type`, `validate`, `sanitize`, `required`, options
4. Register module as `std/schema`

Tests:
- Each type function returns correct structure
- Options are properly stored
- Type composition with `++` works

---

### Task 1.2: Schema Definition
**Files**: `pkg/parsley/stdlib/stdlib_schema.go`
**Estimated effort**: Medium

Steps:
1. Implement `schema.define(name, fields)`:
   - Takes schema name (string) and fields (dictionary of type specs)
   - Returns schema object with name, fields, and helper methods
2. Schema object structure:
   ```
   {
     __schema__: true,
     name: "Todo",
     fields: {title: {...}, done: {...}},
     validate: fn(data) {...},
     sanitize: fn(data) {...}
   }
   ```

Tests:
- `schema.define()` creates valid schema object
- Schema has `name` and `fields` properties
- Schema is identifiable via `__schema__` marker

---

### Task 1.3: Validation Function
**Files**: `pkg/parsley/stdlib/stdlib_schema.go`
**Estimated effort**: Medium

Steps:
1. Implement `Schema.validate(data)` method on schema objects:
   - Iterate over schema fields
   - Apply each field's validation rules
   - Collect errors into array
   - Return `{valid: bool, errors: [...] or []}`
2. Error format:
   ```
   {field: "email", code: "FORMAT", message: "Invalid email format", value: "bad"}
   ```
3. Validation rules by type:
   - `required`: field must exist and not be null/empty
   - `min`/`max`: length for strings, value for numbers
   - `pattern`: regex match for strings
   - `values`: enum membership

Tests:
- Valid data returns `{value, errors: null}`
- Invalid data returns errors array
- Multiple errors collected
- Sanitization applied before validation

---

### Task 1.4: Sanitization Function
**Files**: `pkg/parsley/stdlib/stdlib_schema.go`
**Estimated effort**: Small

Steps:
1. Implement `schema.sanitize(schema, data)`:
   - Apply sanitization without validation
   - Return sanitized data
2. Sanitization by type:
   - `string`: trim whitespace
   - `email`: trim + lowercase
   - `url`: trim
   - `integer`/`number`: coerce from string if possible
   - `boolean`: coerce from string ("true"/"false", "1"/"0")

Tests:
- String trimming works
- Email lowercasing works
- Type coercion works
- Unknown fields passed through or stripped (decide)

---

## Phase 2: ID Generation

### Task 2.1: ID Module
**Files**: `pkg/parsley/stdlib/stdlib_id.go`
**Estimated effort**: Medium

Steps:
1. Create `stdlib_id.go` with module registration
2. Implement ID generators:
   - `id.new()` — ULID-like (use `github.com/oklog/ulid` or similar)
   - `id.uuid()` / `id.uuidv4()` — UUID v4 (use `github.com/google/uuid`)
   - `id.uuidv7()` — UUID v7
   - `id.nanoid()` / `id.nanoid(length)` — NanoID
   - `id.cuid()` — CUID2
3. Register module as `std/id`

Tests:
- Each generator returns string of correct format
- `id.new()` is sortable (later IDs sort after earlier)
- `id.nanoid(10)` returns 10-char string
- Generated IDs are unique (generate 1000, check no duplicates)

---

### Task 2.2: Schema ID Type
**Files**: `pkg/parsley/stdlib/stdlib_schema.go`
**Estimated effort**: Small

Steps:
1. Add `schema.id({format})` type function
2. `format` options: `"default"`, `"uuid"`, `"uuidv7"`, `"nanoid"`, `"cuid"`
3. Validation: check format matches expected pattern
4. Generation: call appropriate `std/id` function

Tests:
- `schema.id()` validates ULID format
- `schema.id({format: "uuid"})` validates UUID format
- ID generation on insert (tested in Phase 3)

---

## Phase 3: Table Binding

### Task 3.1: Table Binding Function
**Files**: `pkg/parsley/stdlib/stdlib_schema.go`
**Estimated effort**: Large

Steps:
1. Implement `schema.table(schema, db, tableName)`:
   - Takes schema, database connection, table name
   - Returns query helper object
2. Query helper structure:
   ```
   {
     __table__: true,
     schema: <schema>,
     db: <db>,
     table: "todos",
     all: fn() {...},
     find: fn(id) {...},
     where: fn(conditions) {...},
     insert: fn(data) {...},
     update: fn(id, data) {...},
     delete: fn(id) {...}
   }
   ```
3. Each method uses parameterized queries

Tests:
- `schema.table()` returns object with all methods
- Methods are callable functions

---

### Task 3.2: Query Methods Implementation
**Files**: `pkg/parsley/stdlib/stdlib_schema.go`
**Estimated effort**: Large

Steps:
1. Implement `all()`:
   - `SELECT * FROM {table}`
   - Return array of dictionaries
2. Implement `find(id)`:
   - `SELECT * FROM {table} WHERE id = ?`
   - Return dictionary or null
3. Implement `where(conditions)`:
   - Build WHERE clause from conditions dict
   - `SELECT * FROM {table} WHERE col1 = ? AND col2 = ?`
   - Return array
4. Implement `insert(data)`:
   - Validate data against schema
   - Generate ID if not provided
   - `INSERT INTO {table} (...) VALUES (...)`
   - Return inserted record with ID
5. Implement `update(id, data)`:
   - Validate data against schema (partial)
   - `UPDATE {table} SET ... WHERE id = ?`
   - Return updated record
6. Implement `delete(id)`:
   - `DELETE FROM {table} WHERE id = ?`
   - Return `{affected: 1}` or similar

Tests:
- `all()` returns all rows
- `find()` returns single row or null
- `where()` filters correctly
- `insert()` creates row with generated ID
- `update()` modifies row
- `delete()` removes row
- SQL injection prevented (test with malicious input)

---

## Phase 4: API Routes

### Task 4.1: Module Export Mapping
**Files**: `server/api.go` (new), `server/handler.go`
**Estimated effort**: Large

Steps:
1. Create `server/api.go` for API-specific handling
2. When loading API module, map exports to routes:
   - `get` → GET `/resource`
   - `post` → POST `/resource`
   - `getById` → GET `/resource/:id`
   - `put` → PUT `/resource/:id`
   - `patch` → PATCH `/resource/:id`
   - `delete` → DELETE `/resource/:id`
3. Extract `:id` parameter into `req.params.id`
4. Handle `routes` export for nested routing

Tests:
- Exports map to correct HTTP methods
- Path parameters extracted
- Nested routes work

---

### Task 4.2: JSON Response Handling
**Files**: `server/api.go`, `server/handler.go`
**Estimated effort**: Medium

Steps:
1. Detect API routes (under `/api/` prefix or configured)
2. If handler returns dictionary/array, serialize as JSON
3. Set `Content-Type: application/json`
4. Handle custom status/headers in response object

Tests:
- Dictionary return → JSON response
- Array return → JSON array response
- HTML string return → HTML response
- Custom status codes work

---

## Phase 5: Authentication

### Task 5.1: Auth Wrappers
**Files**: `pkg/parsley/stdlib/stdlib_api.go` (new)
**Estimated effort**: Medium

Steps:
1. Create `stdlib_api.go` with module registration
2. Implement `public(fn)`:
   - Returns wrapped function with `__auth__: "public"` metadata
3. Implement `adminOnly(fn)`:
   - Returns wrapped function with `__auth__: "admin"` metadata
4. Implement `roles(roleList, fn)`:
   - Returns wrapped function with `__auth__: {roles: [...]}` metadata
5. Wrapper functions are still callable (delegate to inner fn)

Tests:
- `public(fn)` returns callable function
- Wrapped function has `__auth__` property
- Inner function is called correctly

---

### Task 5.2: Auth Enforcement
**Files**: `server/api.go`
**Estimated effort**: Medium

Steps:
1. Before calling handler, check auth metadata
2. If no metadata or auth required:
   - Verify session/API key
   - Populate `req.user`
   - Return 401 if not authenticated
3. If `__auth__: "public"`:
   - Skip auth check
4. If `__auth__: "admin"` or `__auth__: {roles: [...]}`:
   - Verify auth + check role
   - Return 403 if wrong role

Tests:
- Unauthenticated request to protected route → 401
- Authenticated request succeeds
- `public()` routes work without auth
- `adminOnly()` routes require admin role
- Wrong role → 403

---

### Task 5.3: Role Resolution
**Files**: `server/api.go`, `config/config.go`
**Estimated effort**: Medium

Steps:
1. Add `api.auth.roleColumn` config option (default: `"role"`)
2. On auth check, look up user's role from users table
3. Optional: `api.auth.roleResolver` for custom Parsley function
4. Cache role in session to avoid repeated lookups

Tests:
- Role column lookup works
- Custom role resolver works
- Role cached in session

---

## Phase 6: Sensible Defaults

### Task 6.1: Rate Limiting
**Files**: `server/api.go`, `server/ratelimit.go` (new)
**Estimated effort**: Medium

Steps:
1. Create simple in-memory rate limiter
2. Default: 60 requests per minute per IP
3. For authenticated: per user ID
4. Module can override: `export rateLimit = {requests: 100, window: @1m}`
5. Return 429 when limit exceeded

Tests:
- Rate limit enforced
- Per-route override works
- 429 response on limit exceeded

---

### Task 6.2: Pagination
**Files**: `pkg/parsley/stdlib/stdlib_schema.go`
**Estimated effort**: Small

Steps:
1. `all()` and `where()` accept pagination options
2. Default limit: 20, max: 100
3. Accept `?limit=N&offset=M` query params
4. Clamp limit to max

Tests:
- Default pagination works
- Custom limit respected
- Limit clamped to max
- Offset works

---

### Task 6.3: Error Helpers
**Files**: `pkg/parsley/stdlib/stdlib_api.go`
**Estimated effort**: Small

Steps:
1. Implement error helper functions:
   - `notFound(msg)` → `fail("HTTP-404", msg)`
   - `forbidden(msg)` → `fail("HTTP-403", msg)`
   - `badRequest(msg)` → `fail("HTTP-400", msg)`
   - `unauthorized(msg)` → `fail("HTTP-401", msg)`
   - `conflict(msg)` → `fail("HTTP-409", msg)`

Tests:
- Each helper calls `fail` with correct code
- Error propagates correctly

---

### Task 6.4: Extend fail() for Error Codes
**Files**: `pkg/parsley/evaluator/evaluator.go`, `pkg/parsley/builtins/builtins.go`
**Estimated effort**: Medium

Steps:
1. Modify `fail()` to accept optional second argument
2. `fail(message)` — current behavior, code defaults to `USER-0001`
3. `fail(code, message)` — custom code
4. Store code in error object
5. In API handler, check for `HTTP-*` prefix and map to status

Tests:
- `fail("msg")` works as before
- `fail("HTTP-404", "msg")` sets code
- API handler returns correct HTTP status

---

## Validation Checklist

- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] Documentation updated
- [ ] `work/design/api-design-summary.md` reflects final implementation
- [ ] BACKLOG.md updated with deferrals

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| — | — | — | — |

## Deferred Items

Items to add to BACKLOG.md after implementation:
- API key scopes — Not MVP, add later
- Table.groupBy(), Table.join() — Complex, defer
- OAuth2/OIDC providers — After core API stable
- OpenAPI spec generation — Nice to have, not MVP
