# Design Document: `@api` Grammar Construct (v2 — Schema-Bound)

**Status:** Proposal (alternative to v1)  
**Created:** 2026-03-03  
**Authors:** Human + AI collaboration  
**Related:** `DESIGN-api-grammar.md` (v1), `std-api-discussion.md` (Dec 2025), `1.0-SHIP-REVIEW.md` §4/§8

---

## Context

This is a companion to the v1 `@api` design document. Where v1 proposes a minimal grammar construct for declaring API endpoints with explicit handler functions, this document explores the other end of the spectrum: **what if `@api` could compose with `@schema` and `db.bind()` to generate full CRUD APIs with validation, pagination, auth, and rate limiting — using building blocks Parsley already has?**

The December 2025 `std-api-discussion.md` explored this idea as a library (`api.expose()`). This document re-examines it as a grammar construct that composes with existing Parsley primitives.

---

## What Already Exists

Before designing anything new, it's worth recognising how much is already built:

| Capability | Where | Status |
|-----------|-------|--------|
| Schema declaration | `@schema User { name: string, ... }` | ✅ Grammar construct, first-class value |
| Field validation | `ValidateSchemaField()` — required, min/max, pattern, enum, type | ✅ Built into schema evaluation |
| Type coercion | `castFieldValue()` — string→int, string→bool, etc. | ✅ Built into Record creation |
| Database binding | `db.bind(Schema, "tablename")` → `TableBinding` | ✅ Full CRUD: all/find/where/insert/update/delete/save |
| Auto-pagination | `getPagination()` reads `?limit=` and `?offset=` from request | ✅ Built into `TableBinding.all()` |
| Query options | `orderBy`, `select`, `limit`, `offset` | ✅ Parsed from dict argument |
| Soft deletes | `db.bind(Schema, "table", {softDelete: "deleted_at"})` | ✅ Built into TableBinding |
| Record type | `Schema({...})` → validated Record with `.errors`, `.valid?` | ✅ First-class type |
| API method dispatch | `GET`→`get`, `GET /:id`→`getById`, `POST`→`post`, etc. | ✅ In `server/api.go` |
| Auth wrappers | `AuthWrappedFunction` with `public/auth/admin/roles` | ✅ Read by `enforceAuth()` |
| Error model | `apiFailError("HTTP-404", msg, 404)` → unified error with status/code/message | ✅ Caught by try/catch, rendered as JSON |
| Rate limiting | Per-route, by IP or user, configurable requests/window | ✅ In `server/api.go` |
| Redirect | `Redirect{URL, Status}` | ✅ First-class type |
| Subpath/ID extraction | `extractID(subPath)` | ✅ In `server/api.go` |

The gap isn't capability — it's **composition**. These pieces exist but there's no way to wire them together declaratively. You have to write the glue by hand in every API file.

---

## The Idea

`@api` composes `@schema` + `db.bind()` + auth + metadata into a single declaration. The grammar handles the glue. You supply the schema, the binding, and the policy. Parsley generates the handlers.

### Level 0: Handwritten handlers (v1 design)

```parsley
@api {
  auth: public
  get: fn(req) { db.query("SELECT * FROM users") }
  getById: fn(req) { db.query("SELECT * FROM users WHERE id = ?", req.params.id) }
  post: fn(req) { db.insert(users, req.body) }
}
```

You write every handler. The `@api` block carries metadata and enables dispatch. This is the v1 proposal — always available, always an option.

### Level 1: Schema-bound resource

```parsley
let db = basil.db

@schema Todo {
  id: id
  title: string(min: 1, max: 200)
  done: boolean
  createdAt: datetime(auto: true, readonly: true)
}

let Todos = db.bind(Todo, "todos")

@api {
  resource: Todos
  auth: required
}
```

That's it. A full CRUD API with:
- `GET /` → paginated list (using `Todos.all()` with auto-pagination)
- `GET /:id` → single record (using `Todos.find(id)`, 404 if missing)
- `POST /` → create (validates against `Todo` schema, returns record or validation errors)
- `PUT /:id` → update (validates, returns updated record)
- `DELETE /:id` → delete (returns 204)
- Automatic request body validation via the schema
- Automatic pagination via query params (`?limit=20&offset=0`)
- Auth enforced on all endpoints
- Rate limiting at defaults (60/min)

### Level 2: Customised resource

```parsley
@api {
  resource: Todos
  auth: required
  cache: 30s

  // Override just the parts you need to customise
  get: fn(req) {
    Todos.where({done: false}, {orderBy: "createdAt desc"})
  }

  // post, put, delete auto-generated from resource
  // getById auto-generated from resource
}
```

Explicit handlers override auto-generated ones. You customise what you need; the rest is handled.

### Level 3: Hooks instead of full overrides

```parsley
@api {
  resource: Todos
  auth: required

  beforeCreate: fn(req, data) {
    data.update({ownerId: req.user.id})
  }

  beforeUpdate: fn(req, id, data) {
    if (req.user.role != "admin") {
      let todo = Todos.find(id)
      if (todo.ownerId != req.user.id) { forbidden("Not your todo") }
    }
    data
  }

  afterDelete: fn(req, id) {
    log("deleted todo", id, "by", req.user.id)
  }
}
```

Hooks let you inject business logic without replacing the entire handler. The auto-generated handler calls the hook at the right point.

---

## How `resource:` Works

When `@api` sees a `resource:` property, it auto-generates handlers for any HTTP method not explicitly provided. The generation is mechanical — it's doing exactly what you'd write by hand, using the `TableBinding` methods that already exist.

### Auto-generated `get` (list)

Equivalent to:
```parsley
fn(req) {
  Todos.all()
  // pagination is automatic — TableBinding.all() already reads ?limit and ?offset
}
```

### Auto-generated `getById`

Equivalent to:
```parsley
fn(req) {
  let result = Todos.find(req.params.id)
  if (!result) { notFound() }
  result
}
```

### Auto-generated `post` (create)

Equivalent to:
```parsley
fn(req) {
  let record = Todo(req.body)       // schema validates + coerces
  if (!record.valid?) {
    badRequest({errors: record.errors})
  }
  Todos.insert(record)
}
```

### Auto-generated `put` (update)

Equivalent to:
```parsley
fn(req) {
  let existing = Todos.find(req.params.id)
  if (!existing) { notFound() }
  let updated = existing.update(req.body)  // re-validates
  if (!updated.valid?) {
    badRequest({errors: updated.errors})
  }
  Todos.update(req.params.id, updated)
}
```

### Auto-generated `delete`

Equivalent to:
```parsley
fn(req) {
  let existing = Todos.find(req.params.id)
  if (!existing) { notFound() }
  Todos.delete(req.params.id)
  null  // 204 No Content
}
```

### Key point: no magic

Each generated handler is simple, predictable, and does exactly what the equivalent hand-written code would do. The auto-generation is a convenience, not an abstraction. If the auto-generated version doesn't suit your needs, you replace it with an explicit handler — no escape hatch needed because you were never locked in.

---

## The `APIResource` Value (Extended from v1)

```go
type APIResource struct {
    // Metadata (same as v1)
    Auth      string            // "public", "auth", "admin", "roles"
    Roles     []string          // for auth: "roles"
    Cache     time.Duration     // response cache TTL
    CSRF      bool              // require CSRF validation
    RateLimit *Dictionary       // {requests: N, window: "1m"}

    // Handlers — explicit or auto-generated
    Handlers  map[string]Object // "get", "getById", "post", "put", "patch", "delete"

    // Resource binding (v2 addition)
    Resource  *TableBinding     // if present, auto-generates missing handlers
    Schema    *DSLSchema        // extracted from Resource.DSLSchema

    // Hooks (v2 addition)
    Hooks     map[string]Object // "beforeCreate", "afterCreate", etc.

    // Nested routing (same as v1)
    Routes    *Dictionary

    Env       *Environment
}
```

When the evaluator processes an `@api` block with `resource:`, it:

1. Extracts the `TableBinding` and its `DSLSchema`
2. For each standard method (`get`, `getById`, `post`, `put`, `delete`), checks if an explicit handler was provided
3. If not, generates one using the `TableBinding` methods
4. If hooks are present, wraps the generated handler to call them at the appropriate point

---

## Hooks

Hooks run at defined points in the auto-generated handler lifecycle. They receive the request context and the relevant data, and can modify it or abort with an error.

| Hook | When | Arguments | Return |
|------|------|-----------|--------|
| `beforeCreate` | After validation, before insert | `(req, record)` | modified record, or error |
| `afterCreate` | After insert | `(req, record)` | ignored (side effects only) |
| `beforeUpdate` | After validation, before update | `(req, id, record)` | modified record, or error |
| `afterUpdate` | After update | `(req, record)` | ignored |
| `beforeDelete` | Before delete | `(req, id)` | error to abort, or null to proceed |
| `afterDelete` | After delete | `(req, id)` | ignored |
| `beforeList` | Before query | `(req, options)` | modified options (e.g., add where clause) |
| `beforeGet` | Before find | `(req, id)` | error to abort, or null to proceed |

### Owner scoping via hooks

The December discussion mentioned `access: "owner"` as a high-level concept. With hooks, this is explicit and composable:

```parsley
@api {
  resource: Todos
  auth: required

  beforeList: fn(req, opts) {
    opts.update({where: {ownerId: req.user.id}})
  }

  beforeCreate: fn(req, data) {
    data.update({ownerId: req.user.id})
  }

  beforeUpdate: fn(req, id, data) {
    let todo = Todos.find(id)
    if (todo.ownerId != req.user.id) { forbidden() }
    data
  }

  beforeDelete: fn(req, id) {
    let todo = Todos.find(id)
    if (todo.ownerId != req.user.id) { forbidden() }
  }
}
```

This is more lines than `access: "owner"`, but it's transparent — you can read exactly what happens. And it composes: you can combine owner scoping with role checks, soft deletes, audit logging, or any other business logic.

If the owner-scoping pattern turns out to be common enough, it could become a helper:

```parsley
let ownerScoped = import @basil/owner

@api {
  resource: Todos
  auth: required
  ...ownerScoped("ownerId")
}
```

But that's sugar, not core. The hooks are the primitive.

---

## Filesystem Binding (Same as v1)

The filesystem (site mode) or YAML (routes mode) provides the URL prefix. The `@api` block provides the dispatch. This is unchanged from v1:

```
site/
  api/
    todos/
      index.pars    ← contains @api { resource: Todos, ... }
```

- `GET /api/todos/` → `get` (list)
- `GET /api/todos/abc123` → `getById` (subpath `/abc123` → id `abc123`)
- `POST /api/todos/` → `post` (create)
- `PUT /api/todos/abc123` → `put` (update)
- `DELETE /api/todos/abc123` → `delete`

The server evaluates the file, gets an `APIResource`, inspects the type, and dispatches accordingly — identical in both site mode and routes mode.

---

## Validation & Error Responses

Because `@api` knows the schema, validation is automatic. The auto-generated `post` and `put` handlers validate request bodies against the schema and return structured errors.

### Successful creation

```
POST /api/todos
Content-Type: application/json

{"title": "Buy milk", "done": false}
```

```json
HTTP 201 Created

{"id": "abc123", "title": "Buy milk", "done": false, "createdAt": "2026-03-03T10:00:00Z"}
```

### Validation failure

```
POST /api/todos
Content-Type: application/json

{"title": "", "done": "not a boolean"}
```

```json
HTTP 400 Bad Request

{
  "code": "HTTP-400",
  "message": "Validation failed",
  "status": 400,
  "errors": [
    {"field": "title", "code": "VAL-MIN-LENGTH", "message": "title must be at least 1 character"},
    {"field": "done", "code": "VAL-TYPE", "message": "done must be a boolean"}
  ]
}
```

This uses the existing `ValidateSchemaFields()` machinery and the unified error model. No new infrastructure needed.

---

## Pagination

Auto-generated `get` (list) uses `TableBinding.all()`, which already reads `?limit=` and `?offset=` from the request query string via `getPagination()`. Defaults: limit 20, max 100.

The response includes pagination metadata:

```json
GET /api/todos?limit=10&offset=20

{
  "data": [...],
  "pagination": {
    "limit": 10,
    "offset": 20,
    "total": 47
  }
}
```

The `total` field requires a `COUNT(*)` query — the auto-generated handler would call `Todos.count()` alongside `Todos.all()`. This is an optional enhancement; the minimal version just returns the array.

### Wrapping convention

This raises a question: should API list endpoints return a bare array or a wrapper object?

**Option A: Bare array** (current behaviour)
```json
[{"id": "1", "title": "Buy milk"}, ...]
```
Simple. But no room for pagination metadata.

**Option B: Envelope**
```json
{"data": [...], "pagination": {...}}
```
More useful. Industry standard. But adds a layer.

**Recommendation:** Auto-generated list handlers use the envelope. Explicit handlers return whatever they want (bare array, custom envelope, etc.). The `writeAPIResponse` code already handles both shapes.

---

## Rate Limiting

Configured per-`@api` block, using the existing rate limiter:

```parsley
@api {
  resource: Todos
  auth: public
  rateLimit: {requests: 120, window: "1m"}
}
```

Default: 60 requests/minute per IP (authenticated: per user). This is already implemented in `server/api.go` — the `@api` block just needs to carry the configuration.

---

## Cache

```parsley
@api {
  resource: Todos
  auth: public
  cache: 30s
}
```

The server sets `Cache-Control` headers on GET responses. Write operations (`POST`, `PUT`, `DELETE`) invalidate the cache. Since Basil is single-process with an in-memory cache, invalidation is instant.

---

## Nested Resources

For relationships like `/api/users/:id/todos`:

```parsley
// site/api/users/index.pars

@api {
  resource: Users
  auth: required

  routes: {
    todos: import ./todos.pars
  }
}
```

```parsley
// site/api/users/todos.pars

@api {
  auth: required

  // The parent ID comes through the subpath
  get: fn(req) {
    Todos.where({userId: req.params.parentId})
  }
}
```

Or in site mode, the filesystem does this naturally:

```
site/
  api/
    users/
      index.pars          ← @api { resource: Users }
      [id]/
        todos/
          index.pars      ← @api { resource scoped to parent }
```

**Note:** Nested resource routing is a v1.1 concern. The mechanism (subpath splitting, `routes:` dict) already exists in `dispatchModule`. The main design question is how to pass parent IDs through the chain.

---

## Full Example: Blog API

```parsley
// site/api/posts/index.pars

let db = basil.db

@schema Post {
  id: id
  title: string(min: 1, max: 300)
  slug: slug
  body: string(min: 1)
  status: enum["draft", "published", "archived"]
  authorId: string(readonly: true)
  publishedAt: datetime?
  createdAt: datetime(auto: true, readonly: true)
  updatedAt: datetime(auto: true, readonly: true)
}

let Posts = db.bind(Post, "posts")

@api {
  auth: required
  cache: 60s
  rateLimit: {requests: 100, window: "1m"}

  // Custom list: only published posts for non-admins
  get: fn(req) {
    if (req.user.role == "admin") {
      Posts.all({orderBy: "createdAt desc"})
    } else {
      Posts.where({status: "published"}, {orderBy: "publishedAt desc"})
    }
  }

  // getById: auto-generated (find by id, 404 if missing)
  // delete: auto-generated (find, delete, 204)

  beforeCreate: fn(req, data) {
    data.update({authorId: req.user.id})
  }

  beforeUpdate: fn(req, id, data) {
    let post = Posts.find(id)
    if (post.authorId != req.user.id && req.user.role != "admin") {
      forbidden("Can only edit your own posts")
    }
    // Auto-set publishedAt when status changes to published
    if (data.status == "published" && post.status != "published") {
      data.update({publishedAt: @now})
    }
    data
  }
}
```

Lines of code: ~40. What you get: a complete blog post API with auth, validation, owner scoping, publish timestamps, admin override, pagination, rate limiting, and caching.

---

## Comparison: v1 vs v2 for the Same Endpoint

### v1 (explicit handlers only)

```parsley
let db = basil.db
let Todos = db.bind(Todo, "todos")

@api {
  auth: required
  cache: 30s

  get: fn(req) {
    Todos.all()
  }

  getById: fn(req) {
    let t = Todos.find(req.params.id)
    if (!t) { notFound() }
    t
  }

  post: fn(req) {
    let record = Todo(req.body)
    if (!record.valid?) { badRequest({errors: record.errors}) }
    Todos.insert(record)
  }

  put: fn(req) {
    let existing = Todos.find(req.params.id)
    if (!existing) { notFound() }
    let updated = existing.update(req.body)
    if (!updated.valid?) { badRequest({errors: updated.errors}) }
    Todos.update(req.params.id, updated)
  }

  delete: fn(req) {
    let existing = Todos.find(req.params.id)
    if (!existing) { notFound() }
    Todos.delete(req.params.id)
  }
}
```

~30 lines. Explicit, readable, no magic.

### v2 (resource-bound)

```parsley
let db = basil.db
let Todos = db.bind(Todo, "todos")

@api {
  resource: Todos
  auth: required
  cache: 30s
}
```

~5 lines. Same result. Every auto-generated handler does exactly what the v1 version does.

### v2 with customisation

```parsley
@api {
  resource: Todos
  auth: required
  cache: 30s

  // Override list to filter
  get: fn(req) {
    Todos.where({done: false})
  }

  // Everything else auto-generated
}
```

~8 lines. You customise what you need. The rest is handled.

---

## Design Principles

### 1. Progressive disclosure

Level 0 (`@api` with explicit handlers) works without knowing about resource binding. Level 1 (`resource:`) works without knowing about hooks. Level 2 (hooks) works without knowing about nested resources. Each layer is optional.

### 2. No magic — just defaults

Every auto-generated handler is equivalent to code you could write yourself. There's no hidden query rewriting, no implicit middleware, no framework magic. If you want to know what `resource: Todos` does for `POST`, the answer is: "it validates with the schema, inserts with the binding, returns the record or validation errors." That's it.

### 3. Escape hatch is the main hatch

If an auto-generated handler doesn't do what you want, you replace it with an explicit function. You don't configure the auto-generator — you just write the code. This is the Parsley way: code is the configuration.

### 4. Compose, don't configure

Rather than adding flags like `access: "owner"` or `pagination: {max: 50}` to the `@api` block, the design uses composition: hooks for business logic, explicit handlers for custom queries, schema constraints for validation. The `@api` block is small because the primitives are powerful.

### 5. The file declares what it is

Same as v1: the server evaluates the file, inspects the return value, and dispatches accordingly. No YAML annotation, no path convention, no external metadata. Works identically in site mode and routes mode.

---

## What's New in the Grammar (vs v1)

| Addition | Purpose |
|----------|---------|
| `resource:` property | Points to a `TableBinding`, enables auto-generation |
| `beforeX` / `afterX` hooks | Lifecycle hooks for auto-generated handlers |
| Envelope response format | `{data: [...], pagination: {...}}` for list endpoints |

Everything else — the `@api` keyword, the metadata properties, the handler functions, the dispatch mechanism — is identical to v1.

---

## What's New in the Runtime (vs What Exists)

Surprisingly little:

| Need | Existing | New |
|------|----------|-----|
| Schema validation | `ValidateSchemaFields()`, `Record.valid?`, `Record.errors` | Wire into auto-generated POST/PUT |
| CRUD operations | `TableBinding.all/find/insert/update/delete` | Call from auto-generated handlers |
| Pagination | `getPagination()` reads `?limit=` and `?offset=` | Add `count()` call for total; envelope format |
| Auth enforcement | `enforceAuth()` reads `AuthWrappedFunction` metadata | Read from `APIResource.Auth` instead |
| Rate limiting | `enforceRateLimit()` | Read config from `APIResource.RateLimit` |
| Error responses | `apiFailError()` → JSON with status/code/message | Add validation error array format |
| Cache headers | Fragment cache exists | Add `Cache-Control` header from `APIResource.Cache` |
| Hook execution | — | New: wrap auto-generated handlers with hook calls |

The heaviest implementation work is the auto-generation logic in the evaluator (creating the handler functions from the `TableBinding`) and the hook wrapping. The server-side dispatch changes are minimal — it's the same dispatch as v1, just with handlers that were generated rather than hand-written.

---

## What This Deliberately Doesn't Do

### No auto-routing / `api.expose()`

The December discussion proposed `api.expose(todos, {prefix: "/api/todos"})` — a function call that magically registers routes. This design rejects that approach. Routes come from the filesystem or YAML. The `@api` block declares the handler, not the route. This keeps routing predictable and visible.

### No OpenAPI generation (yet)

The `@api` block has enough metadata (schema, methods, auth) to generate an OpenAPI spec. But that's a future extension, not a v1 concern.

### No relationship auto-loading

`@schema` supports `via` relationships, but `@api` doesn't auto-join related data. If you want to include a user's posts in the response, write a custom `getById` handler. Lazy relationship loading is a rabbit hole.

### No query parameter filtering

The December discussion proposed `?done=false&title_like=foo` auto-filtering. This design omits it for v1. Use `beforeList` hooks or explicit `get` handlers for filtering. Auto-filtering from query params has security implications (exposing internal field names, enabling expensive queries) that need careful design.

### No auto-migration

Schema changes don't auto-migrate the database. `db.createTable(Schema)` handles initial creation. Migrations are a separate problem.

---

## Migration from `@std/api`

Same as v1:

1. `@api` grammar is additive — `@std/api` continues to work
2. Auth wrappers (`api.public()`, `api.auth()`, etc.) are superseded by `auth:` property
3. Error helpers (`notFound()`, `badRequest()`, etc.) become builtins or move to `@basil/http`
4. `type: api` in YAML becomes optional (server detects from return value)

---

## Open Questions

### 1. Should `resource:` accept a raw `@schema` without binding?

```parsley
@api {
  resource: Todo           // schema, not binding — no database
  auth: public
  get: fn(req) { ... }     // must be explicit, no DB to auto-generate from
}
```

This would give you validation on POST/PUT (the schema can validate `req.body`) without database operations. The auto-generated handlers would only cover `post` (validate-only, returns validated record) and not `get`/`delete`/etc.

**Recommendation:** Not for v1. Keep `resource:` strictly for `TableBinding`. Schema-only validation can use `Todo(req.body)` in explicit handlers.

### 2. How do auto-generated POST handlers return the created record?

Options:
- Return the input record (what was validated and inserted)
- Re-query the database to get auto-generated fields (id, createdAt)
- Return the result of `Todos.insert()` (which currently returns `lastInsertId` for SQLite)

**Recommendation:** The insert should return the full record including server-generated fields. This may require `Todos.insert()` to do an insert-then-find, or for `@api` auto-generation to do the two-step. This aligns with user expectations from other frameworks.

### 3. PATCH vs PUT

Currently `@api` can have both `put` and `patch`. For auto-generated handlers:
- `put` could mean "full replacement" (all fields required)
- `patch` could mean "partial update" (only provided fields)

The existing `TableBinding.update()` does partial updates (only writes provided columns). So auto-generated `put` and `patch` would behave the same. Is that confusing?

**Recommendation:** Auto-generate `put` only. If users want `patch` semantics, they can add an explicit handler. Or we generate both with the same behaviour, since partial-update-on-PUT is common in practice.

### 4. How does `beforeList` modify query options?

`beforeList` receives an options dict and returns a modified one. But `TableBinding.all()` takes its options as a method argument, while filtering (`where`) is a separate method. The hook may need to switch between `all()` and `where()` depending on whether conditions are added.

**Recommendation:** The auto-generated `get` handler should use `where({}, opts)` as the base (where with no conditions = all), so `beforeList` can add conditions naturally:

```parsley
beforeList: fn(req, opts) {
  opts.update({where: {ownerId: req.user.id}, orderBy: "createdAt desc"})
}
```

This may require extending `TableBinding.where()` to accept options the same way `.all()` does. Small change.

### 5. v1 or v2 for 1.0?

These aren't mutually exclusive. v2 is a superset of v1 — every `resource:` API can also be written with explicit handlers. The question is scope:

- **v1 only for 1.0**: Grammar keyword, metadata, dispatch, explicit handlers. ~2 weeks.
- **v1 + v2 for 1.0**: Add `resource:` auto-generation and hooks. ~3–4 weeks.
- **v1 for 1.0, v2 for 1.1**: Ship the grammar, prove the dispatch model, add resource binding after.

**Recommendation:** v1 for 1.0, v2 for 1.1. The grammar and dispatch model are the hard parts. Once those work, adding `resource:` is incremental. Shipping v1 first also lets real usage inform the hook design.

---

## Summary

| Aspect | v1 (explicit) | v2 (resource-bound) |
|--------|---------------|---------------------|
| Handler authoring | Write every function | Auto-generated from `TableBinding` |
| Lines for CRUD | ~30 | ~5 |
| Customisation | Replace the handler | Override specific handlers, or use hooks |
| Validation | Manual (`Schema(req.body)`) | Automatic on POST/PUT |
| Pagination | Manual (`Todos.all()` handles it) | Automatic with envelope |
| Business logic | In the handler | In hooks (`beforeCreate`, etc.) |
| Complexity | Minimal — just a metadata container | Moderate — auto-generation + hooks |
| New infrastructure | Grammar + dispatch | Grammar + dispatch + auto-gen + hooks |
| Dependency on existing code | Low | High (composes `@schema` + `TableBinding`) |

The core principle is the same as v1: **the file declares what it is.** v2 just lets the declaration do more work.

---

*This document presents an alternative design. It is not a replacement for v1 — rather, v2 extends v1 with resource binding. The recommendation is to implement v1 first (grammar, dispatch, metadata) and layer v2 on top once the foundation is proven.*