---
id: man-pars-std-api
title: "@std/api"
system: parsley
type: stdlib
name: api
created: 2026-02-06
version: 0.2.0
author: Basil Team
keywords:
  - api
  - HTTP
  - auth
  - authentication
  - authorization
  - error
  - redirect
  - handler
  - server
  - basil
---

# @std/api

HTTP API utilities for Basil handlers. Provides auth wrappers for route-level access control, error helpers that map to HTTP status codes, and a redirect helper.

```parsley
let api = import @std/api
```

> ⚠️ This module is designed for use inside Basil server handlers. Auth wrappers are consumed by the Basil router — they have no effect in standalone Parsley scripts.

## Auth Wrappers

Auth wrappers decorate handler functions with authentication metadata. The Basil router reads this metadata to enforce access control before the handler runs.

| Function | Args | Description |
|---|---|---|
| `public(fn)` | function | No authentication required |
| `adminOnly(fn)` | function | Requires `admin` role |
| `roles(roles, fn)` | array, function | Requires any of the specified roles |
| `auth(fn)` | function | Requires any authenticated user |

```parsley
let api = import @std/api

// Public route — anyone can access
let listProducts = api.public(fn(req) {
    @query(Products ??-> *)
})

// Admin only
let deleteUser = api.adminOnly(fn(req) {
    @delete(Users | id == {req.params.id} .)
    {ok: true}
})

// Specific roles
let editArticle = api.roles(["editor", "admin"], fn(req) {
    @update(Articles | id == {req.params.id} |< title: req.body.title .)
    {ok: true}
})

// Any logged-in user
let getProfile = api.auth(fn(req) {
    @query(Users | id == {req.user.id} ?-> *)
})
```

### Options

`public` and `auth` accept an optional first argument for configuration:

```parsley
let handler = api.public({cors: true}, fn(req) {
    // ...
})
```

## Error Helpers

Error helpers return special objects that the Basil server converts to HTTP error responses with the appropriate status code. Each accepts an optional message string.

| Function | HTTP Status | Default Message |
|---|---|---|
| `notFound(msg?)` | 404 | "Not found" |
| `forbidden(msg?)` | 403 | "Forbidden" |
| `badRequest(msg?)` | 400 | "Bad request" |
| `unauthorized(msg?)` | 401 | "Unauthorized" |
| `conflict(msg?)` | 409 | "Conflict" |
| `serverError(msg?)` | 500 | "Internal server error" |

```parsley
fn getUser(req) {
    let user = @query(Users | id == {req.params.id} ?-> *)
    if (user == null) {
        return api.notFound("User not found")
    }
    user
}
```

```parsley
fn createUser(req) {
    check req.body.email else api.badRequest("Email is required")

    let existing = @query(Users | email == {req.body.email} ?-> exists)
    if (existing) {
        return api.conflict("A user with this email already exists")
    }

    @insert(Users |< name: req.body.name |< email: req.body.email ?-> *)
}
```

The error response body is a JSON dictionary:

```parsley
// api.notFound("User not found") produces:
// { "error": { "code": "HTTP-404", "message": "User not found" } }
```

## Redirect Helper

| Function | Args | Description |
|---|---|---|
| `redirect(url, status?)` | string or path, integer? | HTTP redirect (default: 302 Found) |

```parsley
fn handleLogin(req) {
    // ... authenticate ...
    return api.redirect("/dashboard")
}

fn handleOldUrl(req) {
    return api.redirect("/new-url", 301)     // permanent redirect
}
```

The status must be a 3xx code. Common values:

| Status | Meaning |
|---|---|
| 301 | Moved Permanently |
| 302 | Found (default) |
| 303 | See Other |
| 307 | Temporary Redirect |
| 308 | Permanent Redirect |

## See Also

- [HTTP & Networking](../features/network.md) — fetch operator and HTTP requests
- [Error Handling](../fundamentals/errors.md) — Parsley error model
- [Database](../features/database.md) — database connections for handler queries
- [Query DSL](../features/query-dsl.md) — declarative database queries