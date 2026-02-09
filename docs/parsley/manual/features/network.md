---
id: man-pars-network
title: HTTP & Networking
system: parsley
type: features
name: network
created: 2026-02-06
version: 0.2.0
author: Basil Team
keywords:
  - HTTP
  - fetch
  - URL
  - network
  - request
  - response
  - SFTP
  - API
  - headers
---

# HTTP & Networking

Parsley uses the **fetch operator** `<=/=` to make HTTP requests. You create a URL or request handle on the right side, and the operator fetches the content. Response data is automatically parsed based on the handle's format.

## The Fetch Operator (`<=/=`)

The fetch operator works like the file read operator (`<==`) but for network resources:

```parsley
let response <=/= JSON(@https://api.example.com/users)
```

The left side is a variable binding. The right side is a URL handle (a format function wrapping a URL literal) or a plain URL.

### Fetch as an Expression

The fetch operator can also be used as a standalone expression on the right side of an assignment. In this form, `<=/=` is a prefix operator — the result is captured into a variable:

```parsley
let response = <=/= JSON(@https://api.example.com/users)
response.data                    // the parsed content
response.ok                      // true if status 200–299
```

This is equivalent to the statement form (`let response <=/= ...`) but works anywhere an expression is expected — in function arguments, conditionals, or chained operations:

```parsley
// Use in a conditional
if ((<=/= JSON(@https://api.example.com/health)).ok) {
    log("API is up")
}

// Pass directly to a function
processUsers(<=/= JSON(@https://api.example.com/users))
```

## URL Handles

Wrap a URL literal in a format function to control how the response is parsed:

| Handle | Parses as | Description |
|---|---|---|
| `JSON(@https://...)` | dictionary/array | Parse response as JSON |
| `YAML(@https://...)` | dictionary/array | Parse response as YAML |
| `text(@https://...)` | string | Raw response body |
| `lines(@https://...)` | array | Response split into lines |
| `bytes(@https://...)` | array | Raw byte array |

```parsley
let users <=/= JSON(@https://api.example.com/users)
let readme <=/= text(@https://raw.githubusercontent.com/user/repo/main/README.md)
```

A plain URL (without a format wrapper) is fetched as text:

```parsley
let html <=/= @https://example.com
```

## Response Object

When assigned to a single variable, the fetch operator returns a **response dictionary** with metadata:

```parsley
let response <=/= JSON(@https://api.example.com/users)
response.data                    // the parsed content
response.status                  // 200
response.statusText              // "200 OK"
response.ok                      // true (status 200–299)
response.url                     // final URL (after redirects)
response.headers                 // response headers dictionary
```

The response wraps the parsed data alongside HTTP metadata. Access the data directly through dictionary destructuring or via the `data` property.

## Error Handling

Use the `{data, error}` destructuring pattern to capture network errors instead of halting:

```parsley
let {data, error} <=/= JSON(@https://api.example.com/users)
if (error) {
    log("Fetch failed: " + error)
} else {
    for (user in data) {
        user.name
    }
}
```

The error-capture pattern returns a dictionary with these fields:

| Field | Type | Description |
|---|---|---|
| `data` | varies or null | Parsed response content (null on error) |
| `error` | string or null | Error message (null on success) |
| `status` | integer | HTTP status code (0 if request failed entirely) |
| `headers` | dictionary | Response headers (empty dict if request failed) |

Without the `{data, error}` pattern, a failed fetch produces a network-class error that propagates up the call stack.

## HTTP Methods

The default HTTP method is GET. To use other methods, pass an **options dictionary** as the second argument to a format factory function, or use the **write operator** (`==>`) with **method accessors**.

### Options Dictionary

Pass `method`, `body`, `headers`, and `timeout` as a second argument to any format factory:

```parsley
// POST with JSON body
let {data, error} <=/= JSON(@https://api.example.com/users, {
    method: "POST",
    body: {name: "Alice", email: "alice@example.com"},
    headers: {Authorization: "Bearer token123"}
})
```

| Option | Type | Default | Description |
|---|---|---|---|
| `method` | string | `"GET"` | HTTP method (GET, POST, PUT, DELETE, PATCH) |
| `body` | any | none | Request body (dictionaries/arrays auto-serialized as JSON) |
| `headers` | dictionary | none | Custom request headers |
| `timeout` | integer | `30000` | Timeout in milliseconds |

When `body` is a dictionary or array, it is automatically JSON-encoded and `Content-Type` is set to `application/json` (unless you override it in headers).

### Method Accessors

Format factory handles have `.get`, `.post`, `.put`, `.patch`, and `.delete` accessors that return a new request handle with the method set:

```parsley
let api = JSON(@https://api.example.com/users)
api.get                              // GET request handle
api.post                             // POST request handle
api.put                              // PUT request handle
api.delete                           // DELETE request handle
```

| Accessor | Method | Use with |
|---|---|---|
| `.get` | GET | `<=/=` (default, rarely needed) |
| `.post` | POST | `=/=>` (default, rarely needed) |
| `.put` | PUT | `=/=>` |
| `.patch` | PATCH | `=/=>` |
| `.delete` | DELETE | `<=/=` |

### The Remote Write Operator (`=/=>`)

Use `=/=>` to send data to a network target. The left side is the data to send (becomes the request body). The right side is a URL handle. The method defaults to POST unless the handle specifies PUT or PATCH:

```parsley
// POST (default)
{name: "Alice", email: "alice@example.com"} =/=> JSON(@https://api.example.com/users)

// PUT (explicit via accessor)
{name: "Alice Smith"} =/=> JSON(@https://api.example.com/users/123).put

// PATCH
{age: 31} =/=> JSON(@https://api.example.com/users/123).patch
```

The `=/=>` operator only accepts network targets (HTTP request handles or SFTP file handles). For local file writes, use `==>`.

The append variant `=/=>>` works the same way but signals append semantics (relevant for SFTP targets):

```parsley
"log entry\n" =/=>> text(sftp, "/var/log/app.log")
```

### Remote Write as an Expression

Like fetch, the remote write operator is a true expression — it returns a response object that you can capture:

```parsley
// Capture the full response
let response = payload =/=> JSON(@https://api.example.com/items)
response.data                    // response body (parsed)
response.status                  // HTTP status code
response.ok                      // true if status 200–299
```

This works for all remote write variants (`=/=>` and `=/=>>`).

### Error Handling for Remote Writes

```parsley
// Capture the full response
let response = payload =/=> JSON(@https://api.example.com/items)
if (!response.ok) {
    log("Failed:", response.status, response.error)
}

// Destructured capture
let {data, error} = payload =/=> JSON(@https://api.example.com/items)
if (error) {
    log("Error:", error)
}
```

When using `{data, error}` destructuring on a remote write expression, the typed response is automatically converted to the legacy `{data, error, status, headers}` shape for compatibility.

### Examples

```parsley
// PUT with options dictionary
let {data, error} <=/= JSON(@https://api.example.com/users/123, {
    method: "PUT",
    body: {name: "Alice Smith"}
})

// DELETE (no body needed)
let {data, error} <=/= JSON(@https://api.example.com/users/123, {
    method: "DELETE"
})
```

## Interpolated URLs

URL literals support interpolation with `@(...)` syntax:

```parsley
let userId = 123
let user <=/= JSON(@(https://api.example.com/users/{userId}))
```

## SFTP

Parsley supports SFTP connections for reading and writing files on remote servers.

### Creating an SFTP Connection

```parsley
let sftp = @sftp("sftp://user@host:22", {
    keyFile: @~/.ssh/id_rsa
})
```

The first argument is an SFTP URL. The second (optional) argument is an options dictionary:

| Option | Type | Description |
|---|---|---|
| `keyFile` | path or string | Path to SSH private key |
| `passphrase` | string | Passphrase for encrypted key |
| `password` | string | Password authentication |
| `knownHostsFile` | path or string | Path to known_hosts file |
| `timeout` | duration | Connection timeout (default 30s) |

At least one authentication method (key file or password) must be provided.

### Reading and Writing via SFTP

Use the network I/O operators with SFTP connections:

```parsley
// Read a remote JSON file (network read)
let config <=/= JSON(sftp, "/etc/app/config.json")

// Write to a remote file (network write)
"new content" =/=> text(sftp, "/var/data/output.txt")

// Append to a remote log (network append)
"log entry\n" =/=>> text(sftp, "/var/log/app.log")
```

### Connection Methods

| Method | Returns | Description |
|---|---|---|
| `.close()` | null | Close the SFTP connection |

## Common Patterns

### Fetch and Transform

```parsley
let users <=/= JSON(@https://api.example.com/users)
let names = for (user in users) {
    user.name
}
```

### API with Authentication

```parsley
let request = {
    url: @https://api.example.com/data,
    method: "GET",
    format: "json",
    headers: {
        Authorization: "Bearer " + apiToken,
        Accept: "application/json"
    }
}
let {data, error} <=/= request
```

### Safe Fetch with Fallback

```parsley
let {data, error} <=/= JSON(@https://api.example.com/config)
let config = if (error) { defaults } else { data }
```

## Key Differences from Other Languages

- **Operator, not function** — `<=/=` replaces `fetch()` or `http.get()`. The operator syntax mirrors the file read operator `<==`, making the data flow direction clear.
- **True expressions** — both `<=/=` and `=/=>` return values, so `let r = <=/= url` and `let r = data =/=> url` work anywhere an expression is expected.
- **Format-aware handles** — `JSON(@https://...)` auto-parses the response. No manual `response.json()` step.
- **Error capture pattern** — `{data, error}` destructuring catches network failures without try/catch blocks.
- **No async/await** — fetch is synchronous. There are no promises or callbacks.
- **Auto-serialization** — dictionary and array request bodies are automatically JSON-encoded.
- **Local vs network writes** — `==>` writes to local files, `=/=>` writes to network targets. The `/` in the operator visually signals that data crosses a network boundary. This matches the read side: `<==` reads files, `<=/=` fetches URLs.

## See Also

- [URLs](../builtins/urls.md) — URL literals, interpolation, and properties
- [File I/O](file-io.md) — file read/write operators (same pattern as fetch)
- [Error Handling](../fundamentals/errors.md) — `{data, error}` pattern and catchable errors
- [Security Model](security.md) — network security and SSRF prevention