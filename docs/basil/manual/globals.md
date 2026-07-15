---
id: man-bas-globals
title: "Server Globals"
system: basil
type: reference
name: globals
created: 2026-07-15
version: 0.2.0
author: Basil Team
keywords:
  - globals
  - params
  - env
  - args
  - publicUrl
  - csrf
  - server
  - handler
  - basil
---

# Server Globals

Every Basil handler runs with a handful of globals already in scope — no `import` required. They give you the request's input (`@params`), the process environment (`@env`), and command-line arguments (`@args`), plus two server functions for asset URLs and CSRF protection.

> These globals are re-read per request, so a value captured at module scope always reflects the current request.

## `@params`

**Server-only.** Merged URL query parameters and form data.

**Type:** `Dictionary`

Query-string parameters and POST form data are merged into one dictionary, with form data taking precedence on conflicts. This is the value you usually want — it unifies both ways a client can send input.

```parsley
// URL: /search?q=hello
// Or POST form: q=hello
@params.q                       // "hello"
@params["page"] ?? 1            // default to 1 if missing
```

> Use `@params` instead of `request.query` (from [@basil/http](http.md)) when you want unified access to both query strings and form submissions. Reach for `request.query` only when you specifically need the query string alone.

## `@env`

Environment variables dictionary. Works in both the `pars` CLI and the Basil server.

**Type:** `Dictionary`

```parsley
@env.HOME                       // "/Users/alice"
@env["DATABASE_URL"]            // connection string
@env.PATH                       // system PATH
```

> Environment variables are read once at startup and are read-only.

## `@args`

Command-line arguments array. Primarily useful in `pars` CLI scripts.

**Type:** `Array` of `String`

```parsley
// pars script.pars arg1 arg2
@args[0]                        // "arg1"
@args[1]                        // "arg2"
@args.length()                  // 2
```

> In Basil server context `@args` is typically empty — it carries the CLI script's arguments, of which a request handler has none.

## Server Functions

### `publicUrl(path)`

**Server-only.** Register a file and get its content-hashed public URL, for cache-busting.

**Arguments:**
- `path` (path literal or `String`) — the file to register

**Returns:** `String` — a public URL with a content hash in the filename

```parsley
let logoUrl = publicUrl(@./assets/logo.svg)
<img src={logoUrl} alt="Logo"/>
// Produces: /assets/logo-a1b2c3d4.svg
```

**How it works:**

1. The file's content is hashed.
2. The file is copied into the public assets directory with the hash in its filename.
3. The hashed URL is returned, so browsers cache it indefinitely and re-fetch only when the content changes.

**Errors:**
- `state` error — if called outside a Basil server context
- `security` error — if the path is outside the handler's directory
- `IO-0001` — if the file cannot be read

**Security:** the path must be within the handler's root directory; path-traversal attempts are rejected.

**vs. `asset()`:** Parsley's core [`asset()`](../../parsley/manual/features/file-io.md#assets) builtin only strips the `public_dir` prefix to turn a path into a web URL — no hashing, no server required. Use `publicUrl()` when you want content-hashed, cache-busting URLs.

### CSRF Token

Access the CSRF token through the request context to protect state-changing forms.

**Available via:** `basil.csrf.token`

```parsley
<form method="post">
    <input type="hidden" name="_csrf" value={basil.csrf.token}/>
    // form fields...
    <button type="submit">Submit</button>
</form>
```

**How it works:**

1. Basil generates a unique CSRF token per session.
2. The token is stored in a cookie and exposed as `basil.csrf.token`.
3. `POST`/`PUT`/`DELETE` requests must include it as the `_csrf` parameter.
4. Basil validates the token and rejects mismatched requests.

> CSRF protection is automatic for state-changing HTTP methods — always include the token in your forms.

## See Also

- [@basil/http](http.md) — the full request/response context (`request.query`, headers, cookies)
- [@basil/auth](auth.md) — the authenticated user and session
- [Authentication](authentication.md) — configuring auth, roles, and protected paths
