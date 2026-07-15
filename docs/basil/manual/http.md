---
id: man-bas-http
title: "@basil/http"
system: basil
type: stdlib
name: http
created: 2026-07-15
version: 0.2.0
author: Basil Team
keywords:
  - http
  - request
  - response
  - route
  - method
  - cookies
  - headers
  - server
  - handler
  - basil
---

# @basil/http

The HTTP request context for Basil handlers: the incoming `request`, a mutable `response` for status/headers/cookies, and shortcuts for the matched `route` and `method`.

```parsley
let {request, response, route, method} = import @basil/http
```

> ⚠️ **Server-only.** These exports are available only inside Basil request handlers. All four are dynamic accessors that always return the current request's values, even when imported at module scope.

## request

Access to the current HTTP request.

**Type:** `Dictionary`

| Property | Type | Description |
|---|---|---|
| `method` | `String` | HTTP method: `"GET"`, `"POST"`, `"PUT"`, `"DELETE"`, … |
| `path` | `String` | Full request path (e.g. `"/users/123"`) |
| `route` | `String` | The matched route portion after the handler mount point |
| `query` | `Dictionary` | Query-string parameters (e.g. `?name=foo` → `{name: "foo"}`) |
| `headers` | `Dictionary` | Request headers (lowercase keys) |
| `body` | `String` \| `Dictionary` | Request body (for `POST`/`PUT`); JSON bodies are auto-parsed |
| `form` | `Dictionary` | Form data (for form-encoded `POST`/`PUT`/`PATCH`) |
| `files` | `Dictionary` | Uploaded files (for `multipart/form-data` requests) |

```parsley
let {request} = import @basil/http
request.method                  // "GET"
request.query.page ?? 1         // query param with a default
```

> For most input handling, prefer the [`@params`](globals.md#params) global — it merges `query` and `form` into one dictionary.

## response

Set response status, headers, and cookies.

**Type:** `Dictionary` (mutable)

| Property | Type | Description |
|---|---|---|
| `status` | `Integer` | HTTP status code (default: `200`) |
| `headers` | `Dictionary` | Response headers to set |
| `cookies` | `Array` | Cookies to set (array of cookie dictionaries) |

```parsley
let {response} = import @basil/http
response.status = 201
response.headers["X-Custom"] = "value"
response.cookies = [{name: "session", value: "abc123", httpOnly: true}]
```

**Cookie dictionary properties:**

| Property | Type | Description |
|---|---|---|
| `name` | `String` (required) | Cookie name |
| `value` | `String` (required) | Cookie value |
| `path` | `String` | Cookie path (default: `"/"`) |
| `domain` | `String` | Cookie domain |
| `maxAge` | `Integer` | Max age in seconds |
| `expires` | `String` | Expiry date |
| `httpOnly` | `Boolean` | HTTP-only flag |
| `secure` | `Boolean` | Secure flag |
| `sameSite` | `String` | SameSite policy: `"Strict"`, `"Lax"`, `"None"` |

## route

The matched route path — the portion of the request path after the handler's mount point.

**Type:** `String`

```parsley
let {route} = import @basil/http
// Handler mounted at /users, request to /users/123
route                           // "123"
```

## method

The current HTTP method — a shortcut for `request.method`.

**Type:** `String`

```parsley
let {method} = import @basil/http
if (method == "POST") {
    // handle form submission
}
```

## See Also

- [Server Globals](globals.md) — `@params` for merged query + form input
- [@basil/auth](auth.md) — the authenticated user and session
- [@basil/api](api.md) — redirects and HTTP error responses
- [Routing](routing.md) — how requests map to handlers
