---
id: FEAT-002
title: "Basil Web Server"
status: in-progress
priority: high
created: 2025-11-30
author: "@sambeau"
---

# FEAT-002: Basil Web Server

## Summary
Build Basil, a focused Go web server for the Parsley programming language. Basil will be HTTPS-only, fast, secure, and simple to configure. It integrates with Parsley to render dynamic HTML while efficiently serving static files directly.

## Implementation Status

### Phase 1: Core Server (MVP) ✅
- [x] `--dev` flag for local development (HTTP on localhost)
- [x] Configuration via YAML file with ENV variable interpolation
- [x] Config file resolution (CLI flag → ENV → `./basil.yaml` → `~/.config/basil/`)
- [x] Static file serving via configured directory pass-through
- [x] Parsley script rendering with path-based routing to handlers
- [x] Parsley module imports working (security policy allows handler directory)
- [x] Request logging (basic)
- [x] Graceful shutdown
- [x] Dev mode disables script caching for live editing
- [ ] HTTPS-only server with automatic TLS (deferred to Phase 2)
- [x] Proxy mode support

### Phase 2: Enhanced Features
- [x] SQLite database connection management for Parsley
- [x] Hot reload in dev mode (watch scripts and config)
- [x] Request logging (text and JSON formats)
- [x] Multipart form parsing (passed to Parsley as dictionary)
- [x] Security headers (CSP, HSTS, etc.)
- [x] HTTPS-only server with automatic TLS
- [x] Proxy mode support
- [x] Compiled/cached Parsley scripts (AST in memory)
- [ ] Route-based caching for generated responses (configurable TTL)
- [ ] Request data validation/sanitization

### Phase 3: Administration
- [ ] Route-based authentication (Basil-managed)
- [ ] Passkey/WebAuthn authentication
- [ ] User identity passed to Parsley (id, email, etc.)
- [ ] Admin interface (built with Parsley)

## Design Decisions

- **HTTPS-only (production)**: Modern security best practice. `--dev` flag enables HTTP on localhost for development. Production uses `autocert` for automatic Let's Encrypt certificates.

- **Explicit route configuration**: Static directories are "pass-through" (served directly). All other paths route to Parsley handlers. Example: `/admin/*` → admin handler, `/*` → catch-all handler. Parsley receives the full path for its own internal routing.

- **Response dictionary**: Parsley returns `{status, headers, body}` dictionary. Basil annotates with additional headers (security, caching) before sending to client.

- **Parsley as library**: Use `github.com/sambeau/parsley/pkg/parsley` with its clean `Eval`/`EvalFile` API. Server handles request→Parsley data mapping; Parsley handles response generation.

- **Server manages resources**: SQLite connections, caches, auth, and script compilation are managed by Basil, not Parsley scripts. This ensures proper concurrency handling and resource lifecycle.

- **Auth-by-route**: Authentication is configured per-route in Basil config. Authenticated user identity (id, email, etc.) is passed to Parsley. Basil handles the auth flow; Parsley just receives verified user data.

- **Script caching**: In production, Parsley scripts are parsed once and AST cached in memory. In dev mode with hot reload, scripts are re-parsed on file change.

- **Proxy support**: Trust `X-Forwarded-For` and `X-Forwarded-Proto` headers when configured (for deployment behind nginx/Caddy).

- **Mat Ryer patterns**: Follow the Go HTTP service patterns from "How I write HTTP services in Go after 13 years" for testability and maintainability.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Parsley Library Integration

The Parsley library (`pkg/parsley`) provides a clean embedding API:

```go
import "github.com/sambeau/parsley/pkg/parsley"

// Evaluate with request data
result, err := parsley.EvalFile("handler.pars",
    parsley.WithVar("request", requestData),
    parsley.WithLogger(requestLogger),
    parsley.WithSecurity(serverPolicy),
)

// Get output
html := result.String()
```

**Key features available:**
- `WithVar(name, value)` — Pass Go values to Parsley (auto-converted)
- `WithLogger(logger)` — Custom logger for `log()`/`logLine()`
- `WithSecurity(policy)` — File system access controls
- `WithEnv(env)` — Pre-configured environment for shared state

**Type conversions (automatic):**
- Go maps → Parsley dictionaries
- Go slices → Parsley arrays
- Go primitives → Parsley primitives
- `time.Time` → Parsley DateTime dictionary

### Proposed Architecture

```
basil/
├── main.go                 # Entry point, calls run()
├── server.go               # NewServer constructor
├── routes.go               # Route definitions
├── handlers/
│   ├── static.go           # Static file handler
│   ├── parsley.go          # Parsley script handler
│   └── health.go           # Health check endpoint
├── middleware/
│   ├── logging.go          # Request logging
│   ├── security.go         # Security headers
│   └── cache.go            # Response caching
├── config/
│   ├── config.go           # Configuration types
│   └── load.go             # Config file loading
├── internal/
│   ├── tls/                # TLS/ACME management
│   └── db/                 # SQLite connection pool
└── docs/
    └── config-reference.md # Configuration documentation
```

### Request Flow

```
Client Request
      │
      ▼
┌─────────────────┐
│  TLS Termination │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Middleware    │  ← Logging, Security Headers
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│     Router      │
└────────┬────────┘
         │
    ┌────┴────┐
    │         │
    ▼         ▼
┌───────┐ ┌────────────┐
│Static │ │  Parsley   │
│Files  │ │  Handler   │
└───────┘ └─────┬──────┘
                │
                ▼
         ┌──────────────┐
         │ parsley.Eval │
         │   WithVar    │
         │   (request)  │
         └──────────────┘
```

### Request Object for Parsley

```go
requestData := map[string]interface{}{
    "method":  r.Method,
    "path":    r.URL.Path,           // Full path for Parsley routing
    "query":   queryToMap(r.URL.Query()),
    "headers": headersToMap(r.Header),
    "body":    body,                 // string or parsed form/JSON
    "cookies": cookiesToMap(r.Cookies()),
    "remote":  clientIP,             // Respects X-Forwarded-For if configured
    "user":    userInfo,             // nil if not authenticated, else {id, email, ...}
}
```

### Response Dictionary from Parsley

```parsley
// Parsley script returns:
{
    status: 200,
    headers: {
        "Content-Type": "text/html; charset=utf-8",
        "X-Custom": "value"
    },
    body: "<html>...</html>"
}
```

Basil then:
1. Sets status code
2. Merges headers (Parsley headers + Basil security headers)
3. Applies caching headers if route is cached
4. Writes body

### CLI Interface

```
basil                       # Uses default config resolution
basil --config app.yaml     # Explicit config file
basil --dev                 # Development mode (HTTP on localhost:8080)
basil --dev --port 3000     # Override port
basil --version             # Show version
basil --help                # Show help
```

### Config File Resolution

Basil finds its config in this order (first match wins):

1. `--config` flag: `basil --config ./path/to/basil.yaml`
2. `BASIL_CONFIG` env: `BASIL_CONFIG=/path/to/basil.yaml basil`
3. Current directory: `./basil.yaml`
4. XDG config: `~/.config/basil/basil.yaml`

If none found → error listing searched locations.

### Environment Variable Interpolation

Config values support `${VAR}` and `${VAR:-default}` syntax:

```yaml
server:
  port: ${BASIL_PORT:-443}
  https:
    email: ${BASIL_ACME_EMAIL}
    cert: ${BASIL_TLS_CERT:-}
    key: ${BASIL_TLS_KEY:-}

database:
  sqlite: ${BASIL_DB_PATH:-./data/app.db}

logging:
  level: ${BASIL_LOG_LEVEL:-info}
```

**Recommended ENV variables:**
- `BASIL_PORT` — Listen port (common for containers)
- `BASIL_ACME_EMAIL` — Let's Encrypt email
- `BASIL_TLS_CERT` / `BASIL_TLS_KEY` — Manual cert paths
- `BASIL_DB_PATH` — Database location
- `BASIL_LOG_LEVEL` — Override log level

### Configuration Format

```yaml
# basil.yaml
server:
  host: ""                    # Empty = all interfaces
  port: ${BASIL_PORT:-443}
  
  https:
    auto: true                # Use ACME/Let's Encrypt
    email: ${BASIL_ACME_EMAIL:-admin@example.com}
    # OR manual certificates:
    # cert: /path/to/cert.pem
    # key: /path/to/key.pem
  
  proxy:
    trusted: true             # Trust X-Forwarded-* headers
    # trusted_ips: [10.0.0.0/8]  # Optional: restrict to specific proxies

# Static file directories (pass-through, served directly)
static:
  - path: /static/
    root: ./public/static
  - path: /assets/
    root: ./public/assets
  - path: /favicon.ico
    file: ./public/favicon.ico

# Parsley route handlers (evaluated in order, first match wins)
routes:
  - path: /admin/*
    handler: ./handlers/admin.pars
    auth: required            # Must be authenticated
    
  - path: /api/*
    handler: ./handlers/api.pars
    
  - path: /*                  # Catch-all
    handler: ./handlers/site.pars

# Authentication configuration
auth:
  provider: passkey           # or "session", "jwt"
  session_ttl: 24h
  # User data passed to Parsley: {id, email, name, roles}

logging:
  level: info                 # debug, info, warn, error
  format: json                # or "text"
  output: stderr              # stderr (default), stdout, or file path
  # output: ./logs/basil.log  # File path for production
  
  # Parsley script log() output (separate from server logs)
  parsley:
    output: stderr            # stderr, stdout, file, or "response" (include in debug headers)
    # output: ./logs/parsley.log
  
database:
  sqlite: ./data/app.db       # Optional, managed by Basil

cache:
  enabled: true
  default_ttl: 60             # seconds
  scripts: true               # Cache compiled Parsley AST (production)
  routes:                     # Per-route response caching
    /api/status: 5
    /pages/*: 300
```

### Dependencies

**Required:**
- `golang.org/x/crypto/acme/autocert` — Automatic TLS certificates
- `github.com/sambeau/parsley/pkg/parsley` — Parsley language

**Parsley Library Enhancement (Dependency):**
- **FEAT-003**: `WithDB()` option for injecting server-managed database connections (required for Phase 2)

**Already available via Parsley** (transitive dependencies):
- `modernc.org/sqlite` — Pure Go SQLite (no CGO)
- `gopkg.in/yaml.v3` — YAML config parsing
- `golang.org/x/crypto` — SSH/TLS crypto primitives

**Additional for Basil:**
- `github.com/go-webauthn/webauthn` — Passkey/WebAuthn support (Phase 3)
- `github.com/fsnotify/fsnotify` — File watching for hot reload (Phase 2)

### Security Considerations

1. **Parsley sandboxing**: Use `WithSecurity()` to restrict file system access
2. **Input sanitization**: Clean query params, form data before passing to Parsley
3. **Security headers**: Default to strict CSP, HSTS, X-Frame-Options
4. **Rate limiting**: Consider for Phase 2
5. **Path traversal**: Validate static file paths strictly
6. **Auth-by-route**: Basil enforces authentication before Parsley sees the request

### Resolved Questions

1. **Development mode**: `--dev` flag enables HTTP on localhost
2. **Routing**: Explicit config—static directories pass-through, routes map to Parsley handlers. Parsley receives full path for internal routing.
3. **Response format**: Dictionary `{status, headers, body}` from Parsley
4. **Hot reload**: Yes, in dev mode—watch scripts and config for changes
5. **Proxy mode**: Yes, trust `X-Forwarded-*` headers when configured
6. **Authentication**: Basil manages auth per-route; passes verified user identity to Parsley
7. **Config resolution**: CLI flag → ENV → `./basil.yaml` → `~/.config/basil/`
8. **ENV interpolation**: Config supports `${VAR:-default}` syntax for deployment flexibility

### Remaining Investigation

1. **Passkey user identity**: What fields are available after WebAuthn? (id, email, display name, etc.)
2. **Script caching**: Can Parsley library expose parsed AST for reuse? May need `pkg/parsley` enhancement.
3. **ACME staging**: Use Let's Encrypt staging for testing to avoid rate limits

### Out of Scope (for MVP)
- WebSocket support
- HTTP/3 (QUIC)
- Clustering / horizontal scaling
- Built-in deployment tools
- Non-SQLite databases (Phase 2+)

## Related

- Discussion document: User's initial requirements
- Parsley library: `github.com/sambeau/parsley/pkg/parsley`
- Caddy docs: https://caddyserver.com/docs/
- Go HTTP patterns: Mat Ryer's blog post

## Investigation Notes

### Caddy's HTTPS-only Approach
Caddy uses automatic HTTPS via ACME protocol (Let's Encrypt). For local development, it generates self-signed certificates. This is achievable with Go's `autocert` package.

### Parsley Security Model
Parsley's `SecurityPolicy` supports:
- `AllowWriteAll` / specific write paths
- `AllowExecuteAll` / specific execute paths  
- Read restrictions via `RestrictReadPaths`

This allows Basil to sandbox Parsley scripts appropriately.

### Go HTTP Service Patterns (Mat Ryer)
Key patterns to adopt:
1. `main()` only calls `run(ctx, args, getenv)` — testable
2. `NewServer()` returns `http.Handler` — middleware-friendly
3. `routes.go` maps entire API surface — discoverable
4. Handler functions return `http.Handler` — closures for deps
5. Middleware adapter pattern — composable
6. `sync.Once` for lazy initialization — fast startup
