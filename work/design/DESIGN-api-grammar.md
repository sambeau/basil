# Design Document: `@api` Grammar Construct

**Status:** Proposal  
**Created:** 2026-03-03  
**Authors:** Human + AI collaboration  
**Related:** `work/reports/1.0-SHIP-REVIEW.md` §4, §8; `work/design/std-api-discussion.md`

---

## Problem

There are three separate problems that converge on the same solution:

### 1. Site mode can't declare API endpoints

Site mode (filesystem routing) has no way to say "this file is an API endpoint." In routes mode, you write `type: api` in YAML. In site mode, you're stuck — every file is treated as an HTML page handler.

### 2. Route metadata lives outside the handler

Auth, caching, CSRF, and route type are all declared in `basil.yaml` — external to the `.pars` file that actually handles the request. This means:

- The file doesn't know what it is
- The same file behaves differently depending on which YAML points at it
- Site mode can't express any of this (no YAML per route)

### 3. `@std/api` conflates declaration with runtime

The current `@std/api` module does two unrelated things:

1. **Declares metadata** — `api.public(fn)`, `api.auth(fn)`, `api.roles(["admin"], fn)` wrap functions with auth annotations
2. **Creates runtime errors** — `api.notFound()`, `api.badRequest()`, `api.redirect()` produce error/redirect objects

Thing 1 is structural declaration pretending to be a function call. Thing 2 is genuine runtime behaviour. They shouldn't be in the same place.

---

## Insight

Parsley already solves this class of problem. `@schema` declares what data looks like. `@query` declares what a database operation looks like. In both cases:

- The declaration is **a grammar construct**, not a library call
- It produces a **first-class value** that the runtime inspects
- The file **doesn't need external metadata** — the value tells you what it is

An API endpoint is the same kind of thing: a structural declaration of what this handler responds to, how it authenticates, and what functions handle each HTTP method.

---

## Proposal: `@api` as a Grammar Construct

### The simplest case

```parsley
// site/api/users/index.pars

@api {
  get: fn(req) {
    db.query("SELECT * FROM users")
  }
}
```

The file evaluates to an `APIResource` value. The server — whether site mode or routes mode — evaluates the file, sees the result type, and dispatches using API semantics (method routing, JSON responses, etc).

### With metadata

```parsley
@api {
  auth: public
  cache: 5m

  get: fn(req) {
    db.query("SELECT * FROM users")
  }

  post: fn(req) {
    db.insert(users, req.body)
  }
}
```

Auth, cache, and CSRF are declared *inside the handler file*, not in YAML. The server reads them from the `APIResource` value.

### With ID-based dispatch

```parsley
@api {
  auth: required

  get: fn(req) {
    db.query("SELECT * FROM users")
  }

  getById: fn(req) {
    db.query("SELECT * FROM users WHERE id = ?", req.params.id)
  }

  delete: fn(req) {
    db.query("DELETE FROM users WHERE id = ?", req.params.id)
  }
}
```

This uses the same `getById` convention that already exists in routes mode. The dispatch rules are unchanged.

---

## How Routing Binds to `@api`

This is the key question: how does a URL like `GET /api/users/abc123` end up calling `getById` with `abc123`?

**Answer: the same way it does today — the filesystem (or YAML) provides the path prefix, and the remainder is the subpath.**

### Site mode

```
site/
  api/
    users/
      index.pars    ← contains @api { get: ..., getById: ... }
```

1. Request: `GET /api/users/abc123`
2. `findHandler` walks the filesystem, finds `site/api/users/index.pars`
3. `subpath = "/abc123"` (the part of the URL not consumed by the filesystem walk)
4. Server evaluates the file → gets an `APIResource`
5. Server sees it's an API resource → uses API dispatch:
   - `extractID("/abc123")` → `hasID=true, id="abc123"`
   - `mapMethodToExport("GET", true)` → `"getById"`
   - Calls `getById(req)` with `req.params.id = "abc123"`

For the collection: `GET /api/users/` → subpath is empty → `get`.

**The filesystem does the routing. The `@api` block does the dispatch.** The subpath mechanism already exists in site mode (`findHandler` returns it). It's just not used for API dispatch today.

### Routes mode

```yaml
routes:
  - path: /api/users
    handler: ./handlers/users.pars
```

1. Request: `GET /api/users/abc123`
2. YAML route matches `/api/users`
3. `subpath = "/abc123"` (same extraction as today)
4. Server evaluates the file → gets an `APIResource`
5. Same dispatch as above

The YAML no longer needs `type: api`. The server discovers the type from the return value. `type: api` can remain as an optional performance hint (skip HTML rendering setup) but is not required.

### Single-file handler

Even a file served directly (no site structure, single YAML route) works identically. The `@api` value carries everything the server needs.

**The binding is always: infrastructure provides the path prefix → `@api` provides the dispatch.** This is true regardless of whether "infrastructure" means filesystem, YAML, or anything else.

---

## The `APIResource` Value

Like `@schema` produces a `DSLSchema`, `@api` produces an `APIResource`. It's a first-class Parsley value.

### Properties

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `auth` | string | `"auth"` | `"public"`, `"auth"`, `"admin"` |
| `roles` | array | `[]` | Required roles (when `auth` is `"roles"`) |
| `cache` | duration | `0` | Response cache TTL |
| `csrf` | boolean | `false` | Require CSRF token validation |
| `rateLimit` | dict | `{requests: 60, window: "1m"}` | Rate limiting config |
| `get` | function | — | `GET /collection` handler |
| `getById` | function | — | `GET /collection/:id` handler |
| `post` | function | — | `POST /collection` handler |
| `put` | function | — | `PUT /collection/:id` handler |
| `patch` | function | — | `PATCH /collection/:id` handler |
| `delete` | function | — | `DELETE /collection/:id` handler |
| `routes` | dict | — | Nested sub-resource routing |

### Auth inheritance

The top-level `auth` applies to all handlers in the block. Individual handlers can override:

```parsley
@api {
  auth: required

  // Inherits auth: required
  get: fn(req) { ... }

  // Override: this endpoint is public
  getById: api.public(fn(req) { ... })
}
```

Wait — that's the library pattern creeping back in. Better:

```parsley
@api {
  auth: required
  get: fn(req) { ... }
  getById: fn(req) { ... }

  // Per-handler overrides via a nested block?
  // This needs more thought. See Open Questions.
}
```

For v1, top-level auth applying uniformly to all handlers in the block is probably sufficient. Per-handler override is a v1.1 concern.

---

## Nested Resources

The `routes` property enables sub-resource routing, matching the existing `routes` export convention:

```parsley
// site/api/users/index.pars

@api {
  auth: required

  get: fn(req) { db.query("SELECT * FROM users") }
  getById: fn(req) { db.query("SELECT * FROM users WHERE id = ?", req.params.id) }

  routes: {
    posts: import ./posts.pars
  }
}
```

`GET /api/users/abc123/posts` → dispatches to `posts.pars` with appropriate subpath. This is identical to how `routes` works in the current `apiHandler.dispatchModule`.

---

## What Happens to `@std/api`?

It splits into two pieces:

### 1. The `@api` grammar (this proposal)

Replaces: `api.public()`, `api.auth()`, `api.roles()`, `api.adminOnly()`

These are metadata declarations. They become properties of the `@api` block.

### 2. Error/redirect builtins

`notFound()`, `badRequest()`, `forbidden()`, `unauthorized()`, `conflict()`, `serverError()`, `redirect()` — these are runtime operations. They could become:

**Option A: Top-level builtins** (like `len()`, `range()`)
```parsley
notFound("User not found")
redirect("/login")
```

**Option B: Stay in a module** (renamed to `@basil/http` or similar)
```parsley
let {notFound, redirect} = import @basil/http
notFound("User not found")
```

**Option C: Methods on the request object**
```parsley
get: fn(req) {
  if (!user) { req.notFound("User not found") }
}
```

Option A is the most minimal. These are common enough to justify builtins, and they're unambiguous in meaning. Option B is fine if we prefer to keep the global namespace small. Option C is a stretch.

**Recommendation:** Option A for the most common ones (`notFound`, `redirect`, `badRequest`), with the full set available via `import @basil/http`.

---

## Server-Side Changes

### Unified dispatch in site mode

`siteHandler.serveWithHandler` currently always creates a `parsleyHandler`. The change:

```
func (h *siteHandler) serveWithHandler(...) {
    // 1. Evaluate the file
    result := evaluate(handlerPath)

    // 2. Check what came back
    switch result.(type) {
    case *APIResource:
        // Dispatch as API: method routing, JSON, auth from metadata
        h.serveAPI(w, r, result, subpath)
    default:
        // Dispatch as page: render HTML (current behaviour)
        h.servePage(w, r, result)
    }
}
```

### Routes mode becomes simpler

In routes mode, `isAPIRoute()` currently checks YAML `type: api` or path prefix `/api/`. With `@api`, the server can instead:

1. Evaluate the file
2. If the result is an `APIResource`, use API dispatch
3. Otherwise, use page dispatch

`type: api` in YAML remains supported (for backwards compat and as a signal to skip HTML setup) but is no longer required.

### Auth/cache/CSRF from the value

Instead of reading auth from YAML config:

```go
// Before: from YAML
authMode := route.Auth

// After: from the APIResource value
authMode := apiResource.Auth  // "public", "auth", "admin", etc
cacheTTL := apiResource.Cache
csrfRequired := apiResource.CSRF
```

This closes all five gaps from §8 of the ship review:

| Gap | Resolution |
|-----|-----------|
| Per-route cache TTL | `cache: 5m` in `@api` block |
| Auth per route | `auth: required` in `@api` block |
| CSRF middleware | `csrf: true` in `@api` block |
| Per-route public_dir | Not applicable to API routes |
| Route type (API) | The `@api` value *is* the type declaration |

---

## Grammar

### Lexer

New token: `API_LITERAL` for `@api`, added to `detectAtLiteralType` alongside `@schema`, `@query`, etc.

### Parser

```
@api { <property_list> }
```

Where `property_list` is a comma-or-newline-separated list of `key: expression` pairs. Structurally identical to a dictionary literal, but produces an `APIDeclaration` AST node.

Known keys are parsed with awareness (e.g., `auth` expects a string, `cache` expects a duration, handler keys expect functions), but unknown keys are preserved for forward compatibility.

### AST

```go
type APIDeclaration struct {
    Token      lexer.Token
    Properties map[string]Expression  // auth, cache, csrf, get, post, etc.
}
```

### Evaluator

Evaluating an `APIDeclaration` produces an `APIResource` object:

```go
type APIResource struct {
    Auth      string                // "public", "auth", "admin", "roles"
    Roles     []string              // for auth: "roles"
    Cache     time.Duration         // response cache TTL
    CSRF      bool                  // require CSRF validation
    RateLimit *Dictionary           // {requests: N, window: "1m"}
    Handlers  map[string]Object     // "get", "getById", "post", etc.
    Routes    *Dictionary           // nested sub-resources
    Env       *Environment          // captured environment
}

func (a *APIResource) Type() ObjectType { return API_RESOURCE_OBJ }
func (a *APIResource) Inspect() string  { return "@api{...}" }
```

---

## Examples

### Minimal public JSON endpoint

```parsley
// site/api/health/index.pars

@api {
  auth: public
  get: fn(req) { {status: "ok", timestamp: @now} }
}
```

### Full CRUD resource

```parsley
// site/api/todos/index.pars

let db = basil.db

@schema Todo {
  title: string(min: 1, max: 200)
  done: boolean
}

let Todos = db.bind(Todo, "todos")

@api {
  auth: required
  cache: 30s

  get: fn(req) {
    Todos.all()
  }

  getById: fn(req) {
    let todo = Todos.find(req.params.id)
    if (!todo) { notFound("Todo not found") }
    todo
  }

  post: fn(req) {
    Todos.insert(req.body)
  }

  put: fn(req) {
    Todos.update(req.params.id, req.body)
  }

  delete: fn(req) {
    Todos.delete(req.params.id)
  }
}
```

### Mixed site with pages and APIs

```
site/
  index.pars              ← page: returns HTML
  about/
    index.pars            ← page: returns HTML
  api/
    users/
      index.pars          ← API: returns @api { ... }
    health/
      index.pars          ← API: returns @api { ... }
  dashboard/
    index.pars            ← page: returns HTML
```

No configuration needed. The server evaluates each file and dispatches based on what it returns. Pages return HTML. APIs return `@api` resources. The filesystem provides the URL structure.

### Routes mode (backwards compatible)

```yaml
# basil.yaml
routes:
  - path: /
    handler: ./handlers/index.pars
  - path: /api/users
    handler: ./handlers/api/users.pars
    # type: api    ← no longer required, but still accepted
```

```parsley
// handlers/api/users.pars — works identically to site mode version
@api {
  auth: public
  get: fn(req) { db.query("SELECT * FROM users") }
}
```

---

## What This Doesn't Do

This proposal deliberately excludes:

1. **Auto-generated CRUD from schema** — No `api.expose(schema)` magic. You write the handler functions. This keeps it predictable and debuggable.

2. **OpenAPI generation** — Possible future extension (the `@api` block has enough metadata), but not in scope.

3. **Request validation** — Use `@schema` and `.validate()` in your handler functions. The `@api` block doesn't auto-validate request bodies.

4. **Database binding** — `@api` doesn't know about databases. Use `db.bind()` or raw queries in your handlers.

5. **Per-handler auth overrides** — v1 applies auth uniformly to the whole block. Per-handler overrides are a v1.1 concern (see Open Questions).

These are conscious scope limits. Each could be added later without changing the grammar.

---

## Migration Path

### Phase 1: Add `@api` grammar (non-breaking)

- Add `API_LITERAL` token, parser rule, `APIResource` type
- Teach `siteHandler` and `parsleyHandler` to recognise `APIResource` results
- `@std/api` continues to work unchanged

### Phase 2: Move error helpers to builtins or `@basil/http`

- `notFound()`, `badRequest()`, etc. become builtins or move to `@basil/http`
- `@std/api` re-exports them for backwards compatibility with a deprecation warning

### Phase 3: Deprecate `@std/api` wrapper functions

- `api.public(fn)`, `api.auth(fn)`, etc. emit deprecation warnings
- Guide users to use `@api { auth: public, ... }` instead
- Remove in next major version

### Phase 4: Make `type: api` in YAML optional

- Server detects API routes from return value
- YAML `type: api` becomes a documentation hint, not a requirement

---

## Open Questions

### 1. Per-handler auth overrides

Should individual handlers within an `@api` block be able to override the block-level auth? If so, what's the syntax?

Possible approaches:
- **Nested blocks:** `get: { auth: public, handler: fn(req) { ... } }`
- **Metadata dict:** `get: {auth: "public"} fn(req) { ... }` (new syntax)
- **Defer to v1.1:** Block-level auth is sufficient for most cases. A handler that needs different auth can be a separate file.

### 2. Should `@page` exist too?

Site mode page handlers could benefit from the same metadata:

```parsley
@page {
  auth: required
  cache: 5m
}

<h1>"Dashboard"</h1>
```

But this is awkward — the `@page` block doesn't wrap the HTML output. It's metadata floating above it. Two alternatives:

- **Convention:** If a file returns an `@api`, it's an API. Everything else is a page. Page-level metadata uses a different mechanism (front-matter? a `@meta` block?).
- **Unified:** `@handler { type: page, auth: required, cache: 5m, render: fn(req) { <h1>"hi"</h1> } }` — but this forces page handlers into a function wrapper, which breaks the current "the file IS the output" simplicity.

**Recommendation:** Don't add `@page` for v1. Focus `@api` on solving the API dispatch problem. Page-level auth/cache in site mode can be addressed separately (it's a smaller gap — most pages use global protected_paths config).

### 3. How does `pars describe` work with `@api`?

`pars describe` currently introspects types and modules. An `@api` resource could be introspectable:

```
$ pars describe api
@api resource

Properties:
  auth       string     Authentication level ("public", "auth", "admin", "roles")
  cache      duration   Response cache TTL
  csrf       boolean    CSRF token validation
  rateLimit  dict       Rate limiting configuration

Handlers:
  get        function   Handle GET /collection
  getById    function   Handle GET /collection/:id
  post       function   Handle POST /collection
  put        function   Handle PUT /collection/:id
  patch      function   Handle PATCH /collection/:id
  delete     function   Handle DELETE /collection/:id
```

### 4. What about CORS?

CORS is currently server-level config in `basil.yaml`. Should `@api` be able to declare per-endpoint CORS?

```parsley
@api {
  cors: { origins: ["https://example.com"], methods: ["GET", "POST"] }
  get: fn(req) { ... }
}
```

Probably not for v1 — server-level CORS is sufficient for most apps. But the design accommodates it: any key in the `@api` block can carry metadata.

### 5. Dictionary convention vs grammar keyword?

An alternative to a new grammar keyword: use a regular dictionary with a special `__type` marker:

```parsley
{
  __type: "api"
  auth: "public"
  get: fn(req) { ... }
}
```

**Pros:** No grammar changes. The server already inspects dictionary types.
**Cons:** Fragile (typo in `__type` fails silently). Not introspectable. Doesn't follow the `@schema`/`@query` pattern. Feels like a workaround.

**Recommendation:** Use the grammar keyword. It's consistent with the language's existing approach to structural declarations, and it gives the parser/evaluator full knowledge of the construct for error checking and introspection.

---

## Summary

| Aspect | Current (`@std/api`) | Proposed (`@api`) |
|--------|---------------------|-------------------|
| Declaration style | Library function calls | Grammar construct |
| Where metadata lives | YAML config + wrapper functions | Inside the `.pars` file |
| Site mode support | ❌ No API dispatch | ✅ Server inspects return value |
| Routes mode support | ✅ Via `type: api` in YAML | ✅ Auto-detected from return value |
| Auth declaration | `api.public(fn)` wrapper | `auth: public` property |
| Cache declaration | YAML `cache: 5m` | `cache: 5m` property |
| CSRF declaration | YAML/middleware only | `csrf: true` property |
| Introspection | Limited (`pars describe` can't see wrappers) | Full (`APIResource` is a typed value) |
| Error helpers | Bundled in same module | Separate (builtins or `@basil/http`) |
| New grammar | None | One `@` keyword |
| Breaking changes | None | None (additive; old module still works during migration) |

The core principle: **the file declares what it is. The server respects that.**