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
  - publicUrl
  - csrf
  - server
  - handler
  - basil
---

# Server Globals

Every Basil handler runs with a handful of globals already in scope — no `import` required. On top of the two [Parsley globals](../../parsley/manual/fundamentals/globals.md) (`@env` and `@args`, available everywhere), the server adds the request's input (`@params`) plus two functions for asset URLs and CSRF protection.

> `@params` is re-read per request, so a value captured at module scope always reflects the current request.

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

> **`@env` and `@args` are Parsley globals, not Basil-specific.** They work the same in the `pars` CLI and in handlers, so they're documented in the Parsley manual under [Globals](../../parsley/manual/fundamentals/globals.md). In a request handler `@args` is normally empty.

## Server Functions

### `publicUrl(path)`

**Server-only.** Take a private file sitting on disk next to your handler and give it a public URL the browser can fetch — without moving the file into your [public directory](#the-public-directory).

**Arguments:**
- `path` (path literal or `String`) — the file to publish, relative to the current handler

**Returns:** `String` — a public URL that serves the file, with a content hash in the path

```parsley
let logoUrl = publicUrl(@./assets/logo.svg)
<img src={logoUrl} alt="Logo"/>
// logoUrl is something like: /__p/3f9a1c2b7d4e6f80.svg
```

The file itself never moves — `@./assets/logo.svg` stays exactly where it is, private and beside your handler code. `publicUrl()` just makes that one file reachable at the returned URL.

**How it works:**

1. Basil reads the file and computes a short hash of its **contents**.
2. It records a mapping — `hash → this file on disk` — in an in-memory registry. Nothing is copied; the original file is served in place.
3. It returns a URL of the form `/__p/{hash}.{ext}`. When a browser requests that URL, Basil looks the hash up and streams the file straight from disk.

The registry is rebuilt each time the server (re)starts, and each file is hashed once and cached (by modification time and size) so repeated calls in a request are cheap.

**The content hash and caching.** Because the hash is derived from the file's contents, every version of the file gets its own distinct URL. This lets Basil serve the file with long-lived cache headers: the browser can keep a copy **forever**, since that exact URL will never point at different bytes. Edit the file and its hash — and therefore its URL — changes, so the next page render links to the new URL and the browser fetches the fresh version. In other words, the newest version is cached aggressively and old versions simply stop being referenced; you never have to think about stale caches or query-string `?v=` tricks.

**Errors:**
- `state` error — if called outside a Basil server handler
- `security` error — if the path resolves outside the handler's root directory (path-traversal attempts are rejected)
- `IO-0001` — if the file cannot be read

There is also a size ceiling: files over 100 MB are rejected (put those in the public directory instead), and files over 10 MB log a warning.

#### The public directory

Basil also has an ordinary **public directory** for static files — CSS, third-party scripts, images you're happy to expose as-is. Set its path in your site config (default `./public`):

```yaml
# basil.yaml
public_dir: ./public
```

Anything under `public_dir` is served at the web root, and Parsley rewrites paths under it to URLs automatically — `@./public/images/foo.png` becomes `/images/foo.png`. See [Configuration → `public_dir`](configuration.md#public-dir).

**When to use which:**

- **Static files you're fine exposing** — drop them in `public_dir` and reference them directly (or via [`asset()`](../../parsley/manual/features/file-io.md#assets), which just strips the `public_dir` prefix to form the URL). No hashing, no server registration.
- **A private file next to a handler** that you want to publish on demand, with automatic cache-safe versioning — use `publicUrl()`. It leaves the file where it is and serves it through the hashed `/__p/` URL described above.

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

## Other Server Globals

Two more globals are injected by the server but documented on their own pages:

- `@DB` — the configured database connection, shared across handlers. See [Database](database.md).
- `@SEARCH` — the full-text search index. See [Search](search.md).

## See Also

- [Parsley Globals](../../parsley/manual/fundamentals/globals.md) — `@env` and `@args`, available everywhere Parsley runs
- [@basil/http](http.md) — the full request/response context (`request.query`, headers, cookies)
- [@basil/auth](auth.md) — the authenticated user and session
- [Authentication](authentication.md) — configuring auth, roles, and protected paths
