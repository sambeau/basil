---
id: FEAT-028
title: "API Routes"
status: draft
priority: medium
created: 2025-12-05
author: "@human"
---

# FEAT-028: API Routes

## Summary

Provide a standardized way to create JSON API endpoints in Basil, with automatic content-type handling, structured error responses, authentication that returns 401 instead of redirects, and other API-specific concerns.

## Motivation

Currently, Basil is optimized for HTML page rendering. Developers who want to create API endpoints must:
- Manually call `JSON()` and set content-type headers
- Handle errors manually (or get HTML error pages)
- Deal with authentication redirects instead of 401 responses
- Implement their own CORS, rate limiting, etc.

A standardized API route system would let developers focus on business logic while Basil handles the boilerplate.

## User Story

As a developer building a web application with Basil, I want to create API endpoints that automatically return JSON, handle errors properly, and integrate with Basil's authentication system, so that I can quickly build APIs without reinventing common patterns.

## Design Questions

### 1. Route Declaration

How should API routes be declared in `basil.yaml`?

**Option A: Explicit `type` field**
```yaml
routes:
  - path: /api/users
    handler: handlers/api/users.pars
    type: api  # or "json"
```

**Option B: Convention-based (path prefix)**
```yaml
api_prefix: /api  # All routes under /api are API routes
routes:
  - path: /api/users
    handler: handlers/api/users.pars
```

**Option C: Handler return type detection**
Return a dictionary/array → JSON. Return HTML/string → HTML.

### 2. JSON Output

How should handlers return JSON?

**Option A: Automatic serialization**
```parsley
// Just return data - Basil serializes to JSON
{
  users: db.query("SELECT * FROM users")
}
```

**Option B: Explicit JSON wrapper**
```parsley
JSON({
  users: db.query("SELECT * FROM users")
})
```

**Option C: Response object**
```parsley
response({
  body: { users: users },
  status: 200,
  headers: { "X-Custom": "value" }
})
```

### 3. Error Handling

How should errors be returned?

**Automatic (from FEAT-023 structured errors):**
```json
{
  "error": {
    "code": "UNDEF-0001",
    "message": "identifier not found: foo",
    "line": 5,
    "column": 10
  }
}
```

**Questions:**
- Should dev mode include line/column? (Security concern in production)
- Should there be a standard error envelope?
- How to handle application-level errors (not Parsley errors)?

### 4. Authentication

How should `auth: required` work for API routes?

**HTML routes:** Redirect to login page
**API routes:** Return 401 with JSON error

```json
{
  "error": {
    "code": "AUTH-0001", 
    "message": "Authentication required"
  }
}
```

### 5. HTTP Methods

Should API routes support method-specific handlers?

**Option A: Single handler, check method**
```parsley
let { method } = basil.http.request

if (method == "GET") {
  // list users
} else if (method == "POST") {
  // create user
}
```

**Option B: Method-specific handlers in config**
```yaml
routes:
  - path: /api/users
    type: api
    handlers:
      GET: handlers/api/users/list.pars
      POST: handlers/api/users/create.pars
```

**Option C: Export-based routing**
```parsley
// handlers/api/users.pars
export fn GET() { /* list */ }
export fn POST() { /* create */ }
```

### 6. Request Body Parsing

How should JSON request bodies be handled?

```parsley
let body = basil.http.request.json  // Parsed JSON body
let raw = basil.http.request.body   // Raw string
```

### 7. CORS

Should Basil handle CORS automatically for API routes?

```yaml
routes:
  - path: /api/*
    type: api
    cors:
      origins: ["https://example.com"]
      methods: ["GET", "POST"]
```

### 8. Rate Limiting

Should Basil provide built-in rate limiting?

```yaml
routes:
  - path: /api/*
    type: api
    rate_limit: 100/minute
```

## Scope Considerations

**Phase 1 (MVP):**
- `type: api` route declaration
- Automatic JSON serialization
- JSON error responses (using FEAT-023 structure)
- 401 for auth failures instead of redirect

**Phase 2:**
- Method-specific handlers
- JSON body parsing
- CORS support

**Phase 3:**
- Rate limiting
- Request validation/schemas
- OpenAPI spec generation?

## Technical Notes

### Leveraging FEAT-023

Structured errors make JSON error responses trivial:
```go
if errObj != nil {
    parsleyErr := errObj.ToParsleyError()
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusInternalServerError)
    json.NewEncoder(w).Encode(map[string]any{
        "error": parsleyErr,
    })
}
```

### Content-Type Detection

For automatic response type, check if result is:
- `*evaluator.String` containing HTML → `text/html`
- `*evaluator.Dictionary` or `*evaluator.Array` → `application/json`

## Related

- FEAT-006: Dev Mode Error Display (HTML errors) — now this feature handles API errors
- FEAT-023: Structured Error Objects — enables JSON error serialization
- Existing: `basil.http.request` context object

## Open Questions

1. Should API routes be a separate handler type or just configuration on existing handlers?
2. How opinionated should Basil be about REST conventions?
3. Should there be a standard response envelope? (`{ data: ..., error: ..., meta: ... }`)
4. WebSocket support? (Probably separate feature)
