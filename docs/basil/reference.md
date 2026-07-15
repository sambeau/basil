# Basil Server Reference

> Basil's server-only extensions to Parsley — where everything lives, and what's available where.
> Verified against `pars` v0.2.0 and `basil` (January 2026).

A Basil handler runs the full [Parsley language](../parsley/manual/index.md) plus a set of
server-only globals, modules, and functions. This page is the map: each area below has its own
page with the complete detail. The [availability matrix](#appendix-a-feature-availability) at the
foot of the page shows which features work in the `pars` CLI as well as the server.

## Handler API — server-only

Globals, modules, and functions available inside Basil request handlers.

| Area | Import | Covers |
|------|--------|--------|
| [Server Globals](manual/globals.md) | *(none — always in scope)* | `@params`, `@env`, `@args`, `publicUrl()`, CSRF token |
| [@basil/http](manual/http.md) | `import @basil/http` | `request`, `response`, `route`, `method` |
| [@basil/auth](manual/auth.md) | `import @basil/auth` | `session`, `auth`, `user` |
| [@basil/api](manual/api.md) | `import @basil/api` | Redirects, HTTP error helpers, auth wrappers |
| [@basil/log](manual/log.md) | `import @basil/log` | `dev.log` and the dev panel |
| [@basil/html](manual/html.md) | `import @basil/html` | Accessible HTML form and UI components |
| [Session Management](manual/session.md) | *(via `@basil/auth`)* | `basil.session` — storage and flash messages |

## Server features

Larger Basil subsystems, each with its own manual page.

| Area | Page |
|------|------|
| Database (`@DB`) | [Database](manual/database.md) |
| Authentication | [Authentication](manual/authentication.md) · [Guide](manual/authentication-guide.md) |
| Images | [Images](manual/images.md) |
| Search | [Search](manual/search.md) · [Guide](manual/search-guide.md) |
| Parts | [Parts](manual/parts.md) · [Guide](manual/parts-guide.md) · [JavaScript API](manual/parts-js.md) |

## Parsley language

Connection literals, database operators, file I/O, format factories, and the error model are
core **Parsley** features — they behave identically whether you run them under `pars` or inside a
Basil handler. They're documented in the [Parsley Language Manual](../parsley/manual/index.md):

| Feature | Page |
|---------|------|
| Connection literals & database operators | [Parsley: Database](../parsley/manual/features/database.md) |
| File I/O operators (`<==`, `==>`, `==>>`, `<=/=`, `=/=>`) | [Parsley: File I/O](../parsley/manual/features/file-io.md) |
| Format factories (`JSON`, `CSV`, `YAML`, `MD`, `dir`, …) | [Parsley: Data Formats](../parsley/manual/features/data-formats.md) |
| HTTP & networking (`fetch`, SFTP) | [Parsley: HTTP & Networking](../parsley/manual/features/network.md) |
| Shell commands (`@shell`) | [Parsley: Shell Commands](../parsley/manual/features/commands.md) |
| Error model, classes, and codes | [Parsley: Error Handling](../parsley/manual/fundamentals/errors.md) · [Error Codes](../parsley/error-codes.md) |

---

## Appendix A: Feature Availability

| Feature | pars CLI | Basil Server | Notes |
|---------|----------|--------------|-------|
| File I/O (`<==`, `==>`, `==>>`) | ✓ | ✓ | |
| URL Fetch (`<=/=`) | ✓ | ✓ | Requires `--allow-net` in pars |
| Remote Write (`=/=>`, `=/=>>`) | ✓ | ✓ | Requires `--allow-net` in pars |
| Database operators | ✓ | ✓ | |
| Format factories | ✓ | ✓ | |
| `@env` | ✓ | ✓ | |
| `@args` | ✓ | — | Empty in server context |
| `@basil/api` | ✓ | ✓ | Returns objects in pars, handled by server |
| `@basil/log` | no-op | ✓ | Dev panel requires server; silently returns null in pars |
| `@basil/http` | — | ✓ | Request context only |
| `@basil/auth` | — | ✓ | Requires auth config |
| `@params` | — | ✓ | Query + form data |
| `@DB` | — | ✓ | Server-configured database |
| Session methods | — | ✓ | Cookie-based sessions |
| `publicUrl()` | — | ✓ | Asset registration |
| CSRF token | — | ✓ | Form protection |

## Appendix B: Type Summary

| Type | Description | Example |
|------|-------------|---------|
| `DBConnection` | Database connection handle | `@sqlite("./db.sqlite")` |
| `SFTPConnection` | SFTP connection handle | `@sftp("sftp://user@host:22", {...})` |
| `SessionModule` | Session data wrapper | `import @basil/auth` |
| `Redirect` | HTTP redirect response | `redirect("/path")` |
| `APIError` | HTTP error response | `notFound()` |
| `AuthWrappedFunction` | Auth-decorated function | `api.public(fn() {...})` |
| File Handle | File read/write handle | `JSON(@./data.json)` |
| Directory Handle | Directory listing handle | `dir(@./data)` |

## Appendix C: Format Factory Summary

Format factories are a Parsley language feature — see [Data Formats](../parsley/manual/features/data-formats.md) for full documentation.

| Factory | Read Type | Write Type | Options |
|---------|-----------|------------|---------|
| `JSON` | Dictionary/Array/… | any JSON-serializable | — |
| `CSV` | Array of Dict/Array | Array of Dict/Array | `{header: bool}` |
| `YAML` | Dictionary/Array/… | any YAML-serializable | — |
| `text` | String | String | — |
| `lines` | Array of String | Array of String | — |
| `bytes` | Array of Integer | Array of Integer | — |
| `SVG` | String (no prolog) | — | — |
| `MD` | Dict (frontmatter, content) | — | — |
| `dir` | Array of handles | — | — |
| `file` | auto-detected | auto-detected | — |
