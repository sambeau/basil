---
id: PLAN-047
feature: FEAT-047
title: "Implementation Plan for CORS Configuration"
status: draft
created: 2025-12-09
---

# Implementation Plan: FEAT-047 (CORS Configuration)

## Overview
Add CORS (Cross-Origin Resource Sharing) support to Basil with global configuration in `basil.yaml` and optional per-handler overrides via a `cors()` function in Parsley.

## Prerequisites
- [x] FEAT-028 API routes implemented (CORS primarily affects API endpoints)
- [x] Config loading infrastructure exists in `config/`
- [x] Middleware pattern established in `server/`

## Tasks

### Task 1: Add CORS config types
**Files**: `config/config.go`
**Estimated effort**: Small

Steps:
1. Add `CORSConfig` struct with fields: `Origins`, `Methods`, `Headers`, `Expose`, `Credentials`, `MaxAge`
2. Add `CORS CORSConfig` field to root `Config` struct
3. Add validation method to reject `origins: "*"` with `credentials: true`

```go
type CORSConfig struct {
    Origins     StringOrSlice `yaml:"origins"`     // "*" or list of origins
    Methods     []string      `yaml:"methods"`     // default: GET, HEAD, POST
    Headers     []string      `yaml:"headers"`     // allowed request headers
    Expose      []string      `yaml:"expose"`      // exposed response headers
    Credentials bool          `yaml:"credentials"` // allow cookies/auth
    MaxAge      Duration      `yaml:"maxAge"`      // preflight cache (0 = omit header)
}
```

Tests:
- Config parses single origin string
- Config parses origin array
- Config rejects `origins: "*"` with `credentials: true`
- Defaults applied when section missing

---

### Task 2: Parse CORS config from YAML
**Files**: `config/load.go`
**Estimated effort**: Small

Steps:
1. Handle `StringOrSlice` type for origins (can be `"*"` or `[...]`)
2. Set sensible defaults (empty = CORS disabled)
3. Call validation after loading

Tests:
- Load config with cors section
- Load config without cors section (disabled)
- Load config with duration maxAge (@24h)

---

### Task 3: Create CORS middleware
**Files**: `server/cors.go` (new)
**Estimated effort**: Medium

Steps:
1. Create `CORSMiddleware` struct holding config
2. Implement `func (m *CORSMiddleware) Handler(next http.Handler) http.Handler`
3. Check for `Origin` header presence (skip if missing = same-origin)
4. Handle OPTIONS preflight: return 204 with all CORS headers
5. For other requests: add `Access-Control-Allow-Origin` and relevant headers
6. Support dynamic origin matching (check if request origin in allowed list)

Logic:
```go
func (m *CORSMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.Handler) {
    origin := r.Header.Get("Origin")
    if origin == "" {
        next.ServeHTTP(w, r)
        return
    }

    if !m.isOriginAllowed(origin) {
        next.ServeHTTP(w, r)  // No CORS headers = browser blocks
        return
    }

    // Set allowed origin (specific origin, not "*" when credentials enabled)
    allowedOrigin := m.getAllowedOrigin(origin)
    w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)

    if m.config.Credentials {
        w.Header().Set("Access-Control-Allow-Credentials", "true")
    }

    if len(m.config.Expose) > 0 {
        w.Header().Set("Access-Control-Expose-Headers", strings.Join(m.config.Expose, ", "))
    }

    // Handle preflight
    if r.Method == http.MethodOptions {
        m.handlePreflight(w)
        return
    }

    next.ServeHTTP(w, r)
}
```

Tests:
- No Origin header → no CORS headers added
- Origin in allowed list → headers added
- Origin not in list → no headers (browser blocks)
- OPTIONS request → 204 with full preflight headers
- Credentials header only when enabled
- Wildcard origin works (without credentials)

---

### Task 4: Wire up CORS middleware in server
**Files**: `server/server.go`
**Estimated effort**: Small

Steps:
1. Initialize CORSMiddleware in `NewServer` if cors config present
2. Apply middleware to API routes (and optionally all routes based on config)
3. Ensure middleware runs early (before auth, after logging)

Tests:
- Server starts with cors config
- Server starts without cors config
- CORS headers present on API responses

---

### Task 5: Add cors() Parsley function for per-handler overrides
**Files**: `pkg/parsley/evaluator/stdlib_basil.go` or new `stdlib_cors.go`
**Estimated effort**: Medium

Steps:
1. Add `cors()` builtin that sets response headers directly
2. Support `cors(false)` to suppress CORS for a handler
3. Support `cors({origin: "...", methods: [...], ...})` for overrides
4. Store override in handler context, apply in response phase

```parsley
// Disable CORS for this handler
cors(false)

// Custom settings
cors({
    origin: basil.http.request.headers.Origin,
    credentials: true
})
```

Tests:
- `cors(false)` suppresses headers
- `cors({...})` overrides global config
- Dynamic origin from request works

---

### Task 6: Documentation
**Files**: `docs/guide/`, `docs/parsley/reference.md`
**Estimated effort**: Small

Steps:
1. Add CORS section to security/API documentation
2. Document config options in basil.yaml reference
3. Add common patterns (public API, authenticated API, multi-frontend)
4. Document `cors()` function in Parsley reference

---

## Validation Checklist
- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] Manual test: API call from different origin with/without CORS
- [ ] Manual test: Preflight request returns correct headers
- [ ] Manual test: Credentials + specific origin works
- [ ] Documentation updated
- [ ] BACKLOG.md updated with deferrals (if any)

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| — | Task 1: Config types | ⬜ Not started | — |
| — | Task 2: Config parsing | ⬜ Not started | — |
| — | Task 3: CORS middleware | ⬜ Not started | — |
| — | Task 4: Server wiring | ⬜ Not started | — |
| — | Task 5: cors() function | ⬜ Not started | — |
| — | Task 6: Documentation | ⬜ Not started | — |

## Deferred Items
Items to add to BACKLOG.md after implementation:
- Per-route CORS config in routes mode (if needed beyond cors() function)
- CORS for WebSocket upgrade requests (separate feature)
- Wildcard subdomain matching (e.g., `*.example.com`)

## Testing Strategy

### Unit Tests
- `config/config_test.go`: CORSConfig validation
- `server/cors_test.go`: Middleware logic

### Integration Tests
- Full request/response cycle with CORS headers
- Preflight handling

### Manual Testing
```bash
# Simple GET with Origin header
curl -H "Origin: https://app.example.com" http://localhost:8080/api/test -v

# Preflight request
curl -X OPTIONS \
  -H "Origin: https://app.example.com" \
  -H "Access-Control-Request-Method: DELETE" \
  http://localhost:8080/api/test -v

# With credentials (should only work with specific origin config)
curl -H "Origin: https://app.example.com" \
  -H "Cookie: session=abc" \
  http://localhost:8080/api/test -v
```

## Security Considerations
- Never allow `*` origin with credentials (browsers reject, but validate server-side)
- Log CORS rejections in dev mode for debugging
- Consider rate-limiting preflight requests (browser caching via maxAge helps)
