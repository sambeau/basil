---
id: PLAN-002
feature: FEAT-002
title: "Implementation Plan for Basil Web Server (Phase 1)"
status: draft
created: 2025-11-30
---

# Implementation Plan: FEAT-002 Phase 1 (MVP)

## Overview
Implement the core Basil web server with HTTPS support, config loading, static file serving, and Parsley script execution. This is the minimum viable product that allows serving dynamic Parsley-generated content.

## Prerequisites
- [x] FEAT-003: Parsley `WithDB()` option (v0.15.4) — For Phase 2, not blocking Phase 1
- [ ] Go 1.21+ installed
- [ ] Design decisions finalized in FEAT-002 spec

## Tasks

### Task 1: Project Structure & Dependencies
**Files**: `go.mod`, `main.go`
**Estimated effort**: Small

Steps:
1. Update `go.mod` with dependencies (parsley, yaml.v3, autocert)
2. Create minimal `main.go` with `run()` pattern (Mat Ryer style)
3. Set up CLI flags (--config, --dev, --port, --version, --help)

Tests:
- `go build` succeeds
- `--help` prints usage
- `--version` prints version

---

### Task 2: Configuration Loading
**Files**: `config/config.go`, `config/load.go`
**Estimated effort**: Medium

Steps:
1. Define `Config` struct matching YAML schema
2. Implement config file resolution (flag → ENV → ./basil.yaml → ~/.config/basil/)
3. Implement ENV variable interpolation (`${VAR:-default}`)
4. Add validation for required fields

Tests:
- Load config from explicit path
- Load config from current directory
- ENV interpolation works
- Missing required field returns error
- Config file not found returns helpful error

---

### Task 3: Server Core
**Files**: `server.go`, `routes.go`
**Estimated effort**: Medium

Steps:
1. Create `NewServer(config) http.Handler` constructor
2. Set up router with middleware chain
3. Implement graceful shutdown with context
4. Add health check endpoint (`/health`)

Tests:
- Server starts and responds to requests
- Graceful shutdown completes pending requests
- Health endpoint returns 200

---

### Task 4: Static File Handler
**Files**: `handlers/static.go`
**Estimated effort**: Small

Steps:
1. Create handler for static directory pass-through
2. Support both directory roots and single file mappings
3. Return 404 for directory requests (no listing)
4. Set appropriate Content-Type headers

Tests:
- Serves files from configured directories
- Returns 404 for directories
- Returns 404 for non-existent files
- Correct Content-Type for common file types

---

### Task 5: Parsley Handler
**Files**: `handlers/parsley.go`
**Estimated effort**: Medium

Steps:
1. Create handler that evaluates Parsley scripts
2. Build request object (method, path, query, headers, body, cookies)
3. Parse response dictionary from Parsley (status, headers, body)
4. Handle proxy headers (X-Forwarded-For, X-Forwarded-Proto) when configured

Tests:
- Evaluates script and returns response
- Passes request data correctly
- Handles missing/malformed response dictionary
- Respects proxy headers when enabled

---

### Task 6: Request Logging Middleware
**Files**: `middleware/logging.go`
**Estimated effort**: Small

Steps:
1. Create logging middleware with configurable output
2. Log: timestamp, method, path, status, duration, client IP
3. Support JSON and text formats
4. Respect log level configuration

Tests:
- Logs requests in configured format
- Includes all required fields
- Respects log level

---

### Task 7: TLS/HTTPS Support
**Files**: `internal/tls/tls.go`
**Estimated effort**: Medium

Steps:
1. Implement automatic TLS with `autocert` (Let's Encrypt)
2. Support manual certificate configuration
3. Implement `--dev` mode (HTTP on localhost only)
4. Add redirect from HTTP to HTTPS in production

Tests:
- Dev mode serves HTTP on localhost
- Manual certs load correctly
- ACME configuration validates email

---

### Task 8: Integration & CLI
**Files**: `main.go`
**Estimated effort**: Small

Steps:
1. Wire everything together in `run()`
2. Handle signals for graceful shutdown
3. Print startup message with listening address
4. Add example config file

Tests:
- Full integration test with test config
- Server starts in dev mode
- Graceful shutdown on SIGTERM

---

## Validation Checklist
- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `go build -o basil .`
- [ ] Linter passes: `golangci-lint run`
- [ ] Dev mode works: `./basil --dev`
- [ ] Example config loads correctly
- [ ] Documentation updated
- [ ] BACKLOG.md updated with deferrals (if any)

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| | Task 1: Project Structure | ⬜ Not Started | — |
| | Task 2: Config Loading | ⬜ Not Started | — |
| | Task 3: Server Core | ⬜ Not Started | — |
| | Task 4: Static Handler | ⬜ Not Started | — |
| | Task 5: Parsley Handler | ⬜ Not Started | — |
| | Task 6: Logging Middleware | ⬜ Not Started | — |
| | Task 7: TLS Support | ⬜ Not Started | — |
| | Task 8: Integration | ⬜ Not Started | — |

## Deferred to Phase 2
- Database connection management (FEAT-003 complete, ready to use)
- Response caching
- Script caching/compilation
- Hot reload
- Multipart form parsing
- Security headers middleware

## Deferred to Phase 3
- Route-based authentication
- Passkey/WebAuthn
- Admin interface
