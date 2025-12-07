---
id: FEAT-047
title: "CORS Configuration"
status: draft
priority: high
created: 2025-12-07
author: "@human"
---

# FEAT-047: CORS Configuration

## Summary
Add Cross-Origin Resource Sharing (CORS) configuration to allow APIs to be consumed by JavaScript from different origins. CORS controls which external domains can make `fetch()` requests to Basil endpoints from browsers.

## User Story
As a Basil developer building an API, I want to configure CORS so that my frontend application on a different domain can call my API endpoints.

## Acceptance Criteria
- [ ] Global CORS config in `basil.yaml` under `cors:` key
- [ ] Supports: `origins`, `methods`, `headers`, `expose`, `credentials`, `maxAge`
- [ ] Automatically handles OPTIONS preflight requests
- [ ] `cors()` function in Parsley for per-handler overrides
- [ ] `credentials: true` requires specific origin (not `*`)
- [ ] Validates configuration and errors on invalid combinations
- [ ] Documentation with common patterns

## Design Decisions

- **Config-first with Parsley override**: Most apps need simple global config, some routes need different rules
- **Auto-handle preflight**: OPTIONS requests handled automatically when CORS enabled
- **Secure defaults**: No CORS by default (same-origin only)
- **Validation**: Error on invalid config like `credentials: true` with `origins: "*"`

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Configuration

```yaml
# basil.yaml
cors:
  # Which origins can make requests
  # Options: "*" (any), single string, or list
  origins:
    - https://app.example.com
    - https://staging.example.com
  
  # Which HTTP methods are allowed (default: GET, HEAD, POST)
  methods: [GET, POST, PUT, PATCH, DELETE]
  
  # Which request headers are allowed
  headers: [Content-Type, Authorization, X-Requested-With]
  
  # Which response headers the browser can access (beyond simple headers)
  expose: [X-Total-Count, X-Page-Count]
  
  # Allow credentials (cookies, Authorization header)
  # Cannot use with origins: "*"
  credentials: true
  
  # How long browser caches preflight response (seconds or duration)
  maxAge: 86400  # or @24h
```

### Per-Handler Override

```parsley
// Disable CORS for this handler (even if globally enabled)
cors(false)

// Override with specific settings
cors({
    origin: "*",           // Public API, no credentials
    methods: ["GET"]       // Read-only
})

// Dynamic origin (allow if in list)
let origin = basil.http.request.headers.Origin
let allowed = ["https://app1.com", "https://app2.com"]
if (origin in allowed) {
    cors({origin: origin, credentials: true})
}
```

### How CORS Works

**Simple requests** (GET, POST with simple content-types):
```
Request:
  GET /api/data HTTP/1.1
  Origin: https://app.example.com

Response:
  HTTP/1.1 200 OK
  Access-Control-Allow-Origin: https://app.example.com
  Access-Control-Allow-Credentials: true
```

**Preflight requests** (PUT, DELETE, custom headers):
```
Request (browser-initiated):
  OPTIONS /api/data HTTP/1.1
  Origin: https://app.example.com
  Access-Control-Request-Method: DELETE
  Access-Control-Request-Headers: Authorization

Response:
  HTTP/1.1 204 No Content
  Access-Control-Allow-Origin: https://app.example.com
  Access-Control-Allow-Methods: GET, POST, PUT, DELETE
  Access-Control-Allow-Headers: Authorization, Content-Type
  Access-Control-Allow-Credentials: true
  Access-Control-Max-Age: 86400
```

### Response Headers

| Header | When Set | Value |
|--------|----------|-------|
| `Access-Control-Allow-Origin` | Always if CORS enabled | Origin or `*` |
| `Access-Control-Allow-Credentials` | If `credentials: true` | `true` |
| `Access-Control-Allow-Methods` | Preflight only | Configured methods |
| `Access-Control-Allow-Headers` | Preflight only | Configured headers |
| `Access-Control-Expose-Headers` | If `expose` configured | Configured headers |
| `Access-Control-Max-Age` | Preflight only | Configured max age |

### Validation Rules

| Configuration | Valid? | Notes |
|---------------|--------|-------|
| `origins: "*"` | ✅ | Any origin |
| `origins: "*"` + `credentials: true` | ❌ | Browser rejects this |
| `origins: [...]` + `credentials: true` | ✅ | Must be specific origins |
| `methods: []` | ⚠️ | Empty = browser defaults |
| No `cors:` section | ✅ | CORS disabled (same-origin only) |

### Implementation

```go
// In server setup
type CORSConfig struct {
    Origins     []string      // or "*" for any
    Methods     []string      // default: GET, HEAD, POST
    Headers     []string      // allowed request headers
    Expose      []string      // exposed response headers
    Credentials bool          // allow cookies/auth
    MaxAge      time.Duration // preflight cache time
}

// Middleware flow:
// 1. If no Origin header → skip CORS (same-origin request)
// 2. If OPTIONS request → handle preflight, return 204
// 3. Otherwise → add CORS headers to response
```

### Affected Components
- `config/config.go` — Add CORSConfig struct and validation
- `config/load.go` — Parse cors section
- `server/cors.go` (new) — CORS middleware
- `server/server.go` — Wire up middleware
- `pkg/parsley/evaluator/builtins.go` — Add `cors()` function

### Dependencies
- Depends on: None
- Blocks: None

### Common Patterns

**Public read-only API:**
```yaml
cors:
  origins: "*"
  methods: [GET, HEAD]
```

**Authenticated API for specific frontend:**
```yaml
cors:
  origins: https://app.example.com
  methods: [GET, POST, PUT, DELETE]
  headers: [Content-Type, Authorization]
  credentials: true
```

**Multiple frontends (staging + production):**
```yaml
cors:
  origins:
    - https://app.example.com
    - https://staging.app.example.com
    - http://localhost:3000
  credentials: true
```

**Microservices (internal only):**
```yaml
# No cors: section — only same-origin requests allowed
```

## Implementation Notes
*Added during/after implementation*

## Related
- Security design doc: `docs/design/security-features.md`
