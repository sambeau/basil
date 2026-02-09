---
id: man-pars-urls
title: URLs
system: parsley
type: builtins
name: urls
created: 2026-02-06
version: 0.2.0
author: Basil Team
keywords:
  - url
  - URL literal
  - interpolated URL
  - scheme
  - host
  - query
  - fragment
  - origin
  - href
  - fetch
---

# URLs

URL values represent web addresses. Like paths, they are first-class objects (not strings) with typed properties and methods. URLs are created from literals prefixed with `@` followed by a scheme.

## Literals

```parsley
@https://example.com
@http://localhost:3000
@https://api.example.com/v1/users?page=1
```

The `@` prefix followed by `http://` or `https://` creates a URL value.

## Interpolated URLs

Use `@(...)` with `{expr}` placeholders for dynamic URLs:

```parsley
let id = 123
@(https://api.example.com/users/{id})
// https://api.example.com/users/123

let version = "v2"
let resource = "posts"
@(https://api.example.com/{version}/{resource})
// https://api.example.com/v2/posts
```

## url() Builtin

Create a URL from a string — useful when the entire URL is dynamic:

```parsley
let u = url("https://example.com")
```

Prefer literals for static URLs — they're validated at parse time.

## Properties

| Property | Type | Description |
|---|---|---|
| `.scheme` | string | URL scheme (`"http"`, `"https"`, etc.) |
| `.host` | string | Hostname |
| `.port` | integer | Port number (`0` if not specified) |
| `.path` | array | Path segments as array |
| `.query` | dictionary | Query parameters as dictionary |
| `.fragment` | string | Fragment identifier (after `#`) |

```parsley
let u = @https://example.com:8080/api/users?page=1&limit=10#section
u.scheme                         // "https"
u.host                           // "example.com"
u.port                           // 8080
u.query                          // {page: "1", limit: "10"}
u.fragment                       // "section"
```

## Methods

### .origin()

Returns the scheme, host, and port as a string:

```parsley
let u = @https://example.com:8080/api/users
u.origin()                       // "https://example.com:8080"
```

### .pathname()

Returns the path portion as a string:

```parsley
let u = @https://example.com/api/users
u.pathname()                     // "/api/users"
```

### .href()

Returns the full URL as a string:

```parsley
let u = @https://example.com/api?page=1
u.href()                         // "https://example.com/api?page=1"
```

### .search()

Returns the query string (including `?`), or empty string if no query:

```parsley
let u = @https://example.com/api?page=1&limit=10
u.search()                       // "?page=1&limit=10"

let u2 = @https://example.com/api
u2.search()                      // ""
```

### .toDict() / .inspect()

```parsley
let u = @https://example.com/path
u.toDict()                       // {scheme: "https", host: "example.com", ...}
u.inspect()                      // includes __type: "url"
```

## URL Arithmetic

Use `+` to append path segments:

```parsley
let base = @https://api.example.com/v1
let full = base + "/users"
// https://api.example.com/v1/users
```

## URLs as File Handle Sources

URLs work as sources for file handle constructors, enabling HTTP fetches:

```parsley
let data <== JSON(@https://api.example.com/users.json)
let page <== text(@https://example.com/page.html)
```

The read operator `<==` performs an HTTP GET when given a URL-based handle. See [File I/O](../features/file-io.md) for details.

## Fetch Operator

The `<=/=` operator performs an HTTP fetch from a URL handle:

```parsley
let response <=/= @https://api.example.com/data
```

`<=/=` is also a true expression, so it can appear on the right side of an assignment or anywhere a value is expected:

```parsley
let response = <=/= JSON(@https://api.example.com/data)
let {data, error} = <=/= JSON(@https://api.example.com/data)
```

The remote write operator `=/=>` works the same way — it sends data and returns a response:

```parsley
let result = payload =/=> JSON(@https://api.example.com/items)
```

See [HTTP & Networking](../features/network.md) for full details on fetch and remote write expressions.

## Key Differences from Other Languages

- **URLs are objects, not strings** — they have typed properties (`.scheme`, `.host`, `.query` as a dictionary) and methods. No manual string parsing needed.
- **Interpolation uses `{expr}`** — `@(https://api.com/{version}/users)`, not template string syntax.
- **Query params are a dictionary** — access `u.query.page` directly instead of parsing query strings.
- **URLs double as fetch sources** — pass a URL to a file handle constructor and use `<==` to fetch remote data. No separate `fetch()` function needed.

## See Also

- [Paths](paths.md) — filesystem path literals and manipulation
- [File I/O](../features/file-io.md) — reading from URL-based file handles
- [Operators](../fundamentals/operators.md) — `+` URL joining, `<==` read, `<=/=` fetch, `=/=>` remote write
- [HTTP & Networking](../features/network.md) — HTTP requests and responses