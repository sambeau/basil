# Design: HTTP Fetch Improvements

## Status

**Stage:** Approved — Ready for Implementation  
**Created:** 2026-03-16  
**Author:** @sam, @copilot  
**Source:** `work/parsley/design/plan-httpFetchImprovements.prompt.md`, `work/parsley/design/Fetch defaults and options.md`  
**Relates to:** Backlog (STDLIB review §7 — "HTTP client: No external API calling")

---

## 1. Overview

### 1.1 Motivation

Parsley's I/O operators (`<=/=` fetch, `=/=>` write) already support HTTP requests — GET with format decoding, POST/PUT/PATCH with body serialization, custom headers, timeouts, redirects, and error capture. However, the current HTTP client experience has ergonomic gaps compared to the SFTP subsystem and the language's design goals:

1. **No reusable client configuration** — Auth headers, base URLs, and timeouts must be repeated on every request.
2. **No HTTP POST/PUT/PATCH tests** — The write operator for HTTP is implemented but untested.
3. **Method accessor tests missing** — `.get`, `.post`, `.put`, `.patch`, `.delete` accessors are implemented but not exercised in the test suite.

This design documents the current state, confirms what's already done, and defines the remaining work to bring the HTTP client to a complete, well-tested feature.

### 1.2 Design Principles

From the original design notes:

- **Arrow direction matches data flow** — The payload (significant data) determines the operator direction.
- **Sensible defaults** — `<=/=` defaults to GET, `=/=>` defaults to POST.
- **Method shortcuts** — `.put`, `.patch`, `.delete` property accessors on request handles (not method calls — avoids precedence ambiguity).
- **Reusable formatters** — Create configured formatters once, use across multiple requests.
- **Full response metadata** — Access status, headers, URL, errors when needed.

---

## 2. Current State (Already Implemented)

### 2.1 Fetch Operator (`<=/=`) — GET Requests

Fully implemented in `eval_network_io.go`. Supports JSON, YAML, text, lines, bytes formats with error capture:

```parsley
// Basic fetch with error capture
let {data, error} = <=/= JSON(url("https://api.example.com/users"))

// Status code access
let {data, error, status} = <=/= JSON(url("https://api.example.com/users"))

// Different formats
let {data, error} = <=/= YAML(url("https://api.example.com/config"))
let {data, error} = <=/= text(url("https://api.example.com/hello"))
let {data, error} = <=/= lines(url("https://api.example.com/log"))
```

**Tests:** 9 tests in `eval_network_io_test.go` covering JSON, text, YAML, lines, bytes, 404 status, error capture, invalid URL, and redirects.

### 2.2 Write Operator (`=/=>`) — POST/PUT/PATCH Requests

Implemented in `eval_file_io.go` (`evalHTTPWrite`) and dispatched from `eval_network_io.go` (`evalRemoteWriteStatement`). The write operator:

- Detects HTTP request dictionaries and delegates to `evalHTTPWrite`
- Defaults to POST if no method is set
- Preserves PUT/PATCH if set via method accessor
- Auto-serializes dict/array bodies to JSON
- Returns a response typed dictionary

```parsley
// POST (default)
let response = {name: "Alice"} =/=> JSON(url("https://api.example.com/users"))

// PUT via method accessor
let response = {name: "Alice"} =/=> JSON(url("https://api.example.com/users/1")).put

// PATCH via method accessor
let response = {email: "new@example.com"} =/=> JSON(url("https://api.example.com/users/1")).patch
```

**Tests:** None for HTTP writes. Only SFTP write tests exist.

### 2.3 HTTP Method Accessors

Implemented in `evaluator.go` (`evalDotExpression`). Property access on request dictionaries maps to HTTP methods:

| Accessor | HTTP Method | Implemented |
|----------|-------------|-------------|
| `.get` | GET | ✅ |
| `.post` | POST | ✅ |
| `.put` | PUT | ✅ |
| `.patch` | PATCH | ✅ |
| `.delete` | DELETE | ✅ |

Also implemented: `setRequestMethod` in `eval_network_io.go` clones the request dict with the new method.

```parsley
// DELETE via fetch operator (no payload — the request is the message)
let result <=/= JSON(url("https://api.example.com/users/1")).delete
```

**Tests:** None.

### 2.4 Response Typed Dictionary

Implemented. Responses use the `__` prefix typed dictionary pattern:

```parsley
{
  __type: "response",
  __format: "json",
  __data: [...],
  __response: {
    status: 200,
    statusText: "OK",
    ok: true,
    url: "https://api.example.com/users",
    headers: {"content-type": "application/json"},
    error: null
  }
}
```

Auto-unwrap is implemented in `evalDotExpression` — property access on a response dict delegates to `__data`. The `isResponseDict` check and `makeResponseTypedDict` factory are in place.

### 2.5 `fetchUrlContentFull` — Core HTTP Engine

The shared HTTP engine in `eval_network_io.go` handles all HTTP operations:

- Configurable method (GET/POST/PUT/PATCH/DELETE)
- Request body serialization (string, dict→JSON, array→JSON)
- Custom headers via `headers` dict
- Configurable timeout (default 30s)
- Redirect following with final URL capture
- Response header capture
- Format-aware response parsing (json, yaml, text, lines, bytes)
- Test client injection via `testHTTPClient`

---

## 3. Arrow Direction Design

The direction of the I/O arrow matches the direction of the **significant data** (the payload), not the HTTP request direction:

| Method | Operator | Accessor | Rationale |
|--------|----------|----------|-----------|
| GET | `<=/=` | `.get` | Server sends data to client — data flows left |
| POST | `=/=>` | `.post` | Client sends payload to server — data flows right |
| PUT | `=/=>` | `.put` | Client sends payload to server — data flows right |
| PATCH | `=/=>` | `.patch` | Client sends payload to server — data flows right |
| DELETE | `<=/=` | `.delete` | The request itself is the significant message — no payload |

DELETE uses the fetch operator because the important information is the *request to delete*, not a payload being sent. The response (confirmation/error) flows back to the client.

---

## 4. Remaining Work

### 4.1 Phase 1: Test Coverage (Should-Fix)

Add tests for all implemented-but-untested HTTP functionality:

**HTTP Write Tests** (in `eval_network_io_test.go`):
- `TestHTTPWrite_POST_JSON` — POST dict body, verify it arrives as JSON
- `TestHTTPWrite_PUT_JSON` — PUT via `.put` accessor
- `TestHTTPWrite_PATCH_JSON` — PATCH via `.patch` accessor  
- `TestHTTPWrite_POST_DefaultMethod` — Verify POST is the default write method
- `TestHTTPWrite_ResponseStatus` — Verify response status/headers are returned
- `TestHTTPWrite_ErrorCapture` — Network failure during write

**HTTP Method Accessor Tests**:
- `TestFetch_DELETE` — DELETE via `.delete` accessor on fetch
- `TestFetch_MethodAccessor_Post` — `.post` changes method on request dict
- `TestFetch_MethodAccessor_Put` — `.put` changes method on request dict

**Estimated effort:** 2–3 hours. Requires extending `testenv` to capture incoming request method/body for verification.

### 4.2 Phase 2: Reusable Formatters (Post-1.0)

Allow format factories without a URL argument, creating a pre-configured "client" that remembers headers, timeout, and other options:

```parsley
// Create a reusable API client with auth headers
let api = JSON({
    headers: {
        "Authorization": "Bearer " + token,
        "X-API-Key": apiKey
    },
    timeout: 30
})

// Reuse across requests — options are preserved
let users <=/= api(url("https://api.example.com/users"))
let posts <=/= api(url("https://api.example.com/posts"))
let result = {name: "New"} =/=> api(url("https://api.example.com/users"))
```

**Requirements:**
1. `JSON()` / `YAML()` / `text()` etc. with no URL argument returns a "partial formatter" object
2. Partial formatter is callable with a URL, producing a request dict
3. Options from creation are merged with per-request options

**Complexity:** High — requires a new object type (or typed dict) that implements the call interface, plus option merging semantics.

**Open questions:**
- How do per-request options override client-level options? (Last-writer-wins? Deep merge for headers?)
- Should the client object be mutable (add headers later) or immutable (create a new one)?
- Should there be a base URL feature? e.g., `let api = JSON({baseUrl: "https://api.example.com"})` then `api(url("/users"))`

### 4.3 Phase 3: Future Enhancements (Post-1.0)

These are ideas from the original design notes, not yet designed in detail:

- **Form data encoding** — `multipart/form-data` for file uploads
- **URL-encoded bodies** — `application/x-www-form-urlencoded`
- **Streaming responses** — For chunked/streaming HTTP responses
- **Retry logic** — Configurable retry with backoff
- **Cookie jar** — Session cookies across requests

---

## 5. Design Decisions

| # | Decision | Rationale |
|---|----------|-----------|
| 1 | DELETE uses `<=/=` with `.delete` accessor | The request is the message; no payload needed. Consistent with arrow-direction-matches-data-flow principle. |
| 2 | Response is a typed dictionary with `__` prefix | Follows the existing pattern for datetime, duration, regex. `__` prefix prevents collision with JSON data fields. |
| 3 | Data auto-unwraps on property access | Iteration, indexing, and property access on a response delegate to `.__data`. Metadata is accessible via `.__response` or destructuring. |
| 4 | Method accessors are properties, not method calls | `.put` not `.put()` — avoids precedence ambiguity ("does the put happen after the network call?") and matches SFTP format accessor pattern (`.json`, `.text`). |
| 5 | `=/=>` defaults to POST | POST is the de-facto standard for sending data to a server. PUT/PATCH require explicit accessor. |
| 6 | Error is always a string, data is null on error | Clear signal: if `error` is non-null, `data` is null. Human-readable error message. Network errors vs HTTP errors are distinguished by presence of `status`. |
| 7 | Append operator `=/=>>` rejects HTTP targets | HTTP has no append semantic. Clear error message directs users to local file append (`==>>`). |

---

## 6. Implementation Priority

| Phase | Description | Status | Priority | Effort |
|-------|-------------|--------|----------|--------|
| — | Fetch operator (GET) | ✅ Done | — | — |
| — | Write operator (POST/PUT/PATCH) | ✅ Done | — | — |
| — | Method accessors | ✅ Done | — | — |
| — | Response typed dict + auto-unwrap | ✅ Done | — | — |
| 1 | Test coverage for HTTP write + accessors | 🔲 Todo | Should-fix (pre or post 1.0) | 2–3 hours |
| 2 | Reusable formatters | 🔲 Todo | Post-1.0 | High |
| 3 | Future enhancements | 🔲 Todo | Post-1.0 | TBD |

---

## 7. Related Documents

- `work/parsley/design/Fetch defaults and options.md` — Original arrow-direction and method-shortcut reasoning
- `work/parsley/design/plan-httpFetchImprovements.prompt.md` — Original implementation plan (superseded by this doc)
- `work/parsley/INVENTORY-operators.md` — Operator inventory including `<=/=` and `=/=>`
- `work/reports/STDLIB-1.0-RELEASE-REVIEW.md` §7 — HTTP client gap assessment
- `pkg/parsley/evaluator/eval_network_io.go` — Core implementation
- `pkg/parsley/evaluator/eval_file_io.go` — `evalHTTPWrite` implementation
- `pkg/parsley/evaluator/evaluator.go` — Method accessor dispatch in `evalDotExpression`
