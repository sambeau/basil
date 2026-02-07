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

The default HTTP method is GET. To use other methods, construct a request dictionary with a `method` field. The simplest way is to use a URL handle and set properties:

```parsley
// POST with JSON body
let createUser = {
    url: @https://api.example.com/users,
    method: "POST",
    format: "json",
    body: {name: "Alice", email: "alice@example.com"},
    headers: {
        Authorization: "Bearer token123"
    }
}
let response <=/= createUser
```

### Request Dictionary Fields

| Field | Type | Default | Description |
|---|---|---|---|
| `url` | URL | required | Target URL |
| `method` | string | `"GET"` | HTTP method (GET, POST, PUT, DELETE) |
| `format` | string | `"text"` | Response format (json, yaml, text, lines, bytes) |
| `body` | any | none | Request body (dictionaries/arrays auto-serialized as JSON) |
| `headers` | dictionary | none | Custom request headers |
| `timeout` | integer | `30000` | Timeout in milliseconds |

When `body` is a dictionary or array, it is automatically JSON-encoded and `Content-Type` is set to `application/json` (unless you override it in headers).

### Examples

```parsley
// PUT
let updateUser = {
    url: @https://api.example.com/users/123,
    method: "PUT",
    format: "json",
    body: {name: "Alice Smith"}
}
let {data, error} <=/= updateUser

// DELETE
let deleteUser = {
    url: @https://api.example.com/users/123,
    method: "DELETE",
    format: "json"
}
let {data, error} <=/= deleteUser
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

Use file handles with the SFTP connection to read and write remote files:

```parsley
// Read a remote JSON file
let config <=/= JSON(sftp, "/etc/app/config.json")

// Write to a remote file
"new content" ==> text(sftp, "/var/data/output.txt")
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
- **Format-aware handles** — `JSON(@https://...)` auto-parses the response. No manual `response.json()` step.
- **Error capture pattern** — `{data, error}` destructuring catches network failures without try/catch blocks.
- **No async/await** — fetch is synchronous. There are no promises or callbacks.
- **Auto-serialization** — dictionary and array request bodies are automatically JSON-encoded.

## See Also

- [URLs](../builtins/urls.md) — URL literals, interpolation, and properties
- [File I/O](file-io.md) — file read/write operators (same pattern as fetch)
- [Error Handling](../fundamentals/errors.md) — `{data, error}` pattern and catchable errors
- [Security Model](security.md) — network security and SSRF prevention