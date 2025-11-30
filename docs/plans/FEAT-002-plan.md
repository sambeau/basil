---
id: PLAN-002
feature: FEAT-002
title: "Implementation Plan for Basil Web Server (Phase 1)"
status: complete
created: 2025-11-30
---

# Implementation Plan: FEAT-002 Phase 1 (MVP)

## Overview
Implement the core Basil web server with HTTPS support, config loading, static file serving, and Parsley script execution. This is the minimum viable product that allows serving dynamic Parsley-generated content.

## Prerequisites
- [x] FEAT-003: Parsley `WithDB()` option (v0.15.4) — For Phase 2, not blocking Phase 1
- [x] Go 1.21+ installed
- [x] Design decisions finalized in FEAT-002 spec

## Progress Log
- 2025-11-30: Task 1 complete (main.go, go.mod, CLI)
- 2025-11-30: Task 2 complete (config package with tests)
- 2025-11-30: Task 3 complete (server package with graceful shutdown)
- 2025-11-30: Task 4 complete (Parsley integration with script cache)
- 2025-11-30: Task 5 complete (dev mode tested)
- 2025-11-30: Task 6 complete (basic logging)
- 2025-11-30: Task 7 complete (26 tests passing)
- 2025-11-30: Task 8 complete (docs and example)
- 2025-11-30: BUG-001 fixed (dev mode caching)
- 2025-11-30: BUG-002 fixed (module imports)
- 2025-11-30: **Phase 1 Complete** ✅

## Tasks

### Task 1: Project Structure & Dependencies ✅
**Files**: `go.mod`, `main.go`
**Status**: COMPLETE

Steps:
1. ✅ Update `go.mod` with dependencies (parsley, yaml.v3, autocert)
2. ✅ Create minimal `main.go` with `run()` pattern (Mat Ryer style)
3. ✅ Set up CLI flags (--config, --dev, --port, --version, --help)

Tests:
- ✅ `go build` succeeds
- ✅ `--help` prints usage
- ✅ `--version` prints version

---

### Task 2: Configuration Loading ✅
**Files**: `config/config.go`, `config/load.go`
**Status**: COMPLETE

Steps:
1. ✅ Define `Config` struct matching YAML schema
2. ✅ Implement config file resolution (flag → ENV → ./basil.yaml → ~/.config/basil/)
3. ✅ Implement ENV variable interpolation (`${VAR:-default}`)
4. ✅ Add validation for required fields

Tests (12 tests in config/load_test.go):
- ✅ Load config from explicit path
- ✅ Load config from current directory
- ✅ ENV interpolation works
- ✅ Missing required field returns error
- ✅ Config file not found returns helpful error

---

### Task 3: Server Core ✅
**Files**: `server/server.go`
**Status**: COMPLETE

Steps:
1. ✅ Create `New(config) (*Server, error)` constructor
2. ✅ Set up router with static and dynamic routes
3. ✅ Implement graceful shutdown with context
4. Health check endpoint deferred to Phase 2

Tests (7 tests in server/server_test.go):
- ✅ Server starts and responds to requests
- ✅ Graceful shutdown completes cleanly
- ✅ Static file serving works

---

### Task 4: Parsley Handler ✅
**Files**: `server/handler.go`
**Status**: COMPLETE

Steps:
1. ✅ Create handler that evaluates Parsley scripts
2. ✅ Build request object (method, path, query, headers)
3. ✅ Parse response (string, HTML detection, JSON map)
4. ✅ Script caching with concurrent access support
5. ✅ Security policy (read-only, restricted paths)

Tests:
- ✅ String response works
- ✅ Map response returns JSON
- ✅ Missing script returns 500

---

### Task 5: Dev Mode ✅
**Status**: COMPLETE

- ✅ `--dev` flag enables HTTP on localhost
- ✅ Port defaults to 8080 in dev mode
- ✅ Port can be overridden with --port

---

### Task 6: Logging ✅
**Status**: COMPLETE (basic)

- ✅ Info/error logging to stdout/stderr
- ✅ Script log() output captured and logged
- Structured JSON logging deferred to Phase 2

---

### Task 7: Testing ✅
**Status**: COMPLETE

- 4 tests in main_test.go
- 12 tests in config/load_test.go  
- 10 tests in server/server_test.go
- **26 total tests passing**

---

### Task 8: Documentation ✅
**Status**: COMPLETE

- ✅ Example app created (examples/hello/)
- ✅ FEAT-002 spec updated with implementation status
- ✅ Basil quick-start guide created (docs/guide/basil-quick-start.md)

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
