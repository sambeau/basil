---
id: FEAT-034
title: "std/api - API Module for Basil"
status: implemented
priority: high
created: 2025-12-06
updated: 2025-12-08
implemented: 2025-12-08
author: "@human"
---

# FEAT-034: std/api - API Module for Basil

## Summary

Add a comprehensive API module (`std/api`) to Basil that provides schema-based validation, database table binding, automatic CRUD operations, authentication wrappers, and sensible defaults for building JSON APIs. This turns Basil from an HTML-only server into a capable API platform while maintaining its "batteries included, secure by default" philosophy.

## Implementation Status

### Completed ✅
- Core schema type factories (std/schema)
- Schema validation and define functions
- ID generation module (std/id) with ULID, UUID, NanoID, CUID
- Table binding with CRUD methods (SQLite) and schema validation/ID autogen
- API route mapping + nested routes, JSON response handling, custom status/headers
- Auth wrapper functions (std/api): public, adminOnly, roles, auth (enforced server-side, auth-by-default)
- API error helpers: notFound, forbidden, badRequest, unauthorized, conflict, serverError
- Rate limiting defaults (60 req/min per user/IP, per-route override) and pagination defaults (limit=20, max=100)

### Deferred / Revisit ⏸️
- `schema.sanitize()` (not implemented)
- Schema composition via `++` merge
- Role resolution beyond session data: no role column lookup/custom resolver yet; admin/roles wrappers currently deny unless role is present
- `fail(code, message)` extension not implemented
- Extending table helpers via `++` composition (not surfaced yet)

## User Story

As a Basil developer, I want to build JSON APIs with minimal boilerplate so that I can serve mobile apps, SPAs, and third-party integrations alongside my HTML pages without switching frameworks.

## Acceptance Criteria

### Core Schema System (`std/schema`)
- [x] `schema.define(name, fields)` creates a named schema object
- [x] Built-in types: `string`, `email`, `url`, `phone`, `integer`, `number`, `boolean`, `enum`, `date`, `datetime`, `money`, `array`, `object`, `id`
- [x] Types include built-in validation, sanitization, and documentation
- [x] `Schema.validate(data)` returns `{valid, errors}` (method-style API)
- [ ] `schema.sanitize(schema, data)` returns sanitized data without validation
- [ ] Schema composition via `++` merge operator

*Notes*: Validation is in place; sanitization/composition remain open.

### Table Binding
- [x] `schema.table(schema, db, tableName)` returns query helper object
- [x] Query methods: `all()`, `find(id)`, `where(conditions)`, `insert(data)`, `update(id, data)`, `delete(id)`
- [x] Automatic ID generation on insert (using `std/id`)
- [x] Schema validation on insert/update
- [ ] Extensible via `++` merge for custom query methods (not exposed yet)

*Notes*: Implementation is Go-side `TableBinding` (SQLite-only) with parameterized SQL and identifier validation.

### API Routes
- [x] Module exports map to HTTP methods (`get`, `post`, `getById`, `put`, `patch`, `delete`)
- [x] `routes` export for nested route composition
- [x] Dictionary return = JSON response, HTML string = HTML response
- [x] Custom status/headers via response object

*Notes*: API detection via `type: api` or `/api/` convention; `req.params.id` populated for `getById`/mutations.

### Authentication Wrappers
- [x] All handlers require auth by default (server enforces; wrappers set metadata)
- [x] `public(fn)` wrapper opts out of auth
- [x] `adminOnly(fn)` wrapper requires admin role  
- [x] `roles(roleList, fn)` wrapper requires specific roles
- [x] `auth(fn)` wrapper for custom auth options
- [x] `req.user` available in authenticated handlers (id/name/email)
- [ ] Convention-based role lookup (configurable `roleColumn`)
- [ ] Optional custom role resolver function

*Notes*: Admin/roles enforcement currently denies when role data absent; role resolution needs follow-up.

### ID Generation (`std/id`)
- [x] `id.new()` returns default sortable ID (ULID)
- [x] `id.uuid()` / `id.uuidv4()` for UUID v4
- [x] `id.uuidv7()` for time-sortable UUID v7
- [x] `id.nanoid()` / `id.nanoid(length)` for NanoID
- [x] `id.cuid()` for CUID2

### Sensible Defaults
- [x] Rate limiting: 60 req/min default, configurable per-route via `export rateLimit`
- [x] Pagination: 20 items default, 100 max, configurable via query params; limit=0 disables cap
- [x] Auth required by default
- [x] Consistent error format: `{error: {code, message, field?}}`
- [x] SQL injection protection via parameterized queries and identifier validation

### Error Handling
- [ ] `fail(code, message)` two-argument form for HTTP errors (not implemented)
- [ ] `HTTP-*` error codes map to HTTP status codes (only via APIError helpers)
- [x] Error helpers: `notFound(msg)`, `forbidden(msg)`, `badRequest(msg)`, `unauthorized(msg)`, `conflict(msg)`, `serverError(msg)`

## Design Decisions

- **Wrapper functions, not method decorators**: Parsley functions don't have methods, so auth uses `public(fn)` not `fn.public()`. This is explicit and works with current Parsley.

- **Schema-first design**: Schemas are the foundation. Define once, get validation, sanitization, query helpers, and documentation everywhere. No separate validation layer.

- **Auth ON by default**: Every API handler requires authentication unless explicitly wrapped with `public()`. Users opt out of security, not into it.

- **Convention over configuration for roles**: Planned `roleColumn`/resolver support deferred; current enforcement denies admin/roles when role data is absent.

- **`++` for composition**: Use Parsley's existing merge operator for extending schemas and query helpers rather than inventing new syntax.

- **ULID-like default IDs**: `id.new()` returns sortable, URL-safe, compact IDs by default. UUID available when interoperability requires it.

- **fail() for HTTP errors**: Extend Parsley's existing `fail()` function to accept error codes. `HTTP-*` prefix triggers HTTP response mapping.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### New Modules to Create

**pkg/parsley/stdlib/stdlib_schema.go:**
- `schema.define()`, `schema.sanitize()`
- Type functions: `schema.string()`, `schema.email()`, etc.
- `Schema.validate(data)` method on schema objects
- `schema.table()` for database binding
- `schema.id()` type

**pkg/parsley/stdlib/stdlib_id.go:**
- `id.new()` - ULID-like default
- `id.uuid()`, `id.uuidv4()`, `id.uuidv7()`
- `id.nanoid()`, `id.cuid()`

**pkg/parsley/stdlib/stdlib_api.go:**
- `public()`, `adminOnly()`, `roles()` wrappers
- Error helpers: `notFound()`, `forbidden()`, etc.
- Auth metadata handling

**server/api.go:**
- API route registration from module exports
- Auth enforcement based on wrapper metadata
- Rate limiting per route
- Pagination handling
- JSON response formatting

## Affected Components

- `pkg/parsley/evaluator/evaluator.go` — exported `CallWithEnv`/`ExportsToDict`, TableBinding dispatch, DB helpers reused
- `pkg/parsley/evaluator/stdlib_schema.go` — `schema.table` binding
- `pkg/parsley/evaluator/stdlib_schema_table_binding.go` — TableBinding implementation (SQLite-only, parameterized SQL)
- `pkg/parsley/evaluator/stdlib_api.go` — auth wrappers, APIError helpers
- `server/api.go` — API routing, auth enforcement, rate limiting, response serialization
- `server/server.go` — route setup, default rate limiter
- `server/ratelimit.go` — in-memory token bucket

### Dependencies

- Depends on: Existing auth system (FEAT-004), SQLite support (FEAT-002)
- Blocks: None

### Edge Cases & Constraints

1. **Mixed auth methods** — Handler-level wrappers override module-level defaults
2. **Role lookup on every request** — Consider caching user roles in session
3. **Schema validation errors** — Return 400 with structured error array
4. **Rate limit scope** — Per-IP for public, per-user for authenticated
5. **Pagination limits** — Clamp user-provided limits to max (100)
6. **ID collision** — ULID includes random component, collision probability negligible

## Implementation Phases

### Phase 1: Core Schema System
- `std/schema` module with types and validation
- Manual validation in handlers
- No table binding yet

### Phase 2: ID Generation
- `std/id` module
- All ID formats
- Integration with `schema.id()` type

### Phase 3: Table Binding
- `schema.table()` implementation
- Query helper methods
- Auto ID generation on insert

### Phase 4: API Routes
- Module-to-route mapping
- JSON response handling
- Route composition with `routes` export

### Phase 5: Authentication
- Auth wrappers (`public`, `adminOnly`, `roles`)
- `req.user` population
- Role resolution (convention + custom)

### Phase 6: Sensible Defaults
- Rate limiting
- Pagination
- Error formatting
- `fail()` extension

## Configuration

```yaml
api:
  rateLimit:
    default: 60        # requests per minute
    window: 1m
  pagination:
    default: 20
    max: 100
  auth:
    roleColumn: role   # Column name in users table
    # Or: roleResolver: ./config/auth.pars
```

## Related

- Design: `docs/design/api-design-summary.md`
- Plan: `docs/plans/FEAT-034-plan.md` (to be created)
- Auth: FEAT-004
- Database: FEAT-002
