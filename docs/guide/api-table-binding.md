# Basil API + Table Binding

This document explains the new API surface for Parsley-based JSON APIs: `std/api` route helpers plus schema-aware table bindings via `schema.table`. It includes examples (minimal and advanced) that you can copy into `*.pars` files.

## What the feature is

- **API routes**: Modules exported under `/api/...` (or `type: api` routes) map HTTP methods to exports (`get`, `getById`, `post`, `put`, `patch`, `delete`). Auth defaults to **protected** unless you wrap with `api.public(...)`.
- **Auth wrappers** (`std/api`): `public`, `auth`, `adminOnly`, `roles([...])` decorate handlers and attach metadata. Server enforces it before your function runs.
- **Schema-aware table bindings** (`schema.table`): Given a schema, DB connection, and table name, you get CRUD helpers that validate input, auto-generate IDs, and clamp pagination.
- **Responses**: Dict/array → JSON, string → text/HTML, `APIError` → JSON + status, `{status, headers, body}` works for custom responses.
- **Defaults you get for free**: Auth required unless made public, rate limiting (60 req/min per user/IP), pagination defaults (`limit=20,max=100,offset=0`) on `all()`/`where()`, JSON content-type, structured errors.

## Library surface (snippets)

```parsley
{api} = import(@std/api)
{schema} = import(@std/schema)
```

**Auth wrappers**

```parsley
export get = api.public(fn(req) { {ok: true} })
export post = api.auth(fn(req) { /* requires session */ })
export delete = api.adminOnly(fn(req) { /* admin only */ })
export put = api.roles(["editor", "admin"], fn(req) { /* role-gated */ })
```

**Table binding**

```parsley
let User = schema.define("User", {
  id: schema.id(),
  email: schema.email({required: true}),
  name: schema.string({required: true}),
})

let db = SQLITE(":memory:")  // or managed basil.sqlite from server
let Users = schema.table(User, db, "users")

Users.insert({email: "a@example.com", name: "A"})  // validates + auto id
Users.find("abc")       // single row or null
Users.where({name: "A"}) // equality match, parameterized
Users.update("abc", {name: "B"})
Users.delete("abc")      // returns {affected: n}
```

## Minimal example (naive but works)

A single file API for `/api/todos` that only lists and creates todos in SQLite.

```parsley
{api} = import(@std/api)
{schema} = import(@std/schema)

let Todo = schema.define("Todo", {
  id: schema.id(),
  title: schema.string({required: true, min: 1, max: 200}),
})

let db = SQLITE(":memory:")
let _ = db <=!=> "CREATE TABLE todos (id TEXT PRIMARY KEY, title TEXT)"
let Todos = schema.table(Todo, db, "todos")

export get = api.public(fn(req) { Todos.all() })
export post = api.public(fn(req) {
  let body = req.body  // expect {title}
  let result = Todos.insert(body)
  // If validation failed, return it directly (API serializes to JSON)
  result
})
```

What you get for free here:
- Auth defaults to required, but `api.public` made both endpoints open.
- `Todos.insert` validates `title`, auto-generates ULID `id`, and returns the inserted row.
- Pagination on `Todos.all()` is clamped to `limit<=100`, default `limit=20`, `offset=0` from query params.
- JSON response + content-type without manual setting.
- Rate limit 60 req/min per IP (or per user if authenticated).

## More advanced example (users table with role-gated update)

Adds fields, role protection, and `getById`/`patch` handlers.

```parsley
{api} = import(@std/api)
{schema} = import(@std/schema)

let User = schema.define("User", {
  id: schema.id({format: "uuidv7"}),
  email: schema.email({required: true}),
  name: schema.string({required: true, max: 120}),
  role: schema.enum({values: ["user", "admin"], default: "user"}),
})

// In Basil server context, basil.sqlite is a managed connection
let db = basil.sqlite
let _ = db <=!=> "CREATE TABLE IF NOT EXISTS users (id TEXT PRIMARY KEY, email TEXT, name TEXT, role TEXT)"
let Users = schema.table(User, db, "users")

// List with pagination (query params limit/offset applied automatically)
export get = api.auth(fn(req) { Users.all() })

// Fetch one
export getById = api.auth(fn(req) {
  let user = Users.find(req.params.id)
  if (user == null) { api.notFound("User") } else { user }
})

// Create
export post = api.auth(fn(req) {
  let result = Users.insert(req.body)
  result
})

// Partial update, admin only
export patch = api.adminOnly(fn(req) {
  let updates = req.body
  let result = Users.update(req.params.id, updates)
  result
})

// Optional per-route rate limit override (stricter for this module)
export rateLimit = {requests: 30, window: @1m}
```

What you get here:
- **Auth**: all routes require login; updates require admin wrapper; `req.user` populated by server when authenticated.
- **Validation**: email/name/role enforced; update rejects `id` changes; invalid payload returns `{valid:false, errors:[...]}` with HTTP 200 by default (you can wrap with `api.badRequest` if desired).
- **ID generation**: `uuidv7` auto-generated if missing on insert.
- **Pagination**: `Users.all()` uses limit/offset query params with default 20/max 100.
- **Rate limiting**: overridden to 30 req/min per user/IP via `rateLimit` export; otherwise defaults to 60/min.
- **Safety**: all SQL is parameterized; column names validated as identifiers.

## Tips

- If you need custom error codes, return `api.error(code, status, message)` (via `APIError`) or the validation dict from `schema.validate` / table methods.
- To disable pagination caps for a one-off list, pass `limit=0` in the query string (interpreted as "no limit").
- For public read but protected write, wrap readers with `api.public` and writers with `api.auth`/`api.adminOnly`.
- Tables are SQLite-only for now; managed connection `basil.sqlite` is provided to API modules automatically.
