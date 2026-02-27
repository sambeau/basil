---
id: FEAT-132
title: "Integration Testing Infrastructure (testenv)"
status: draft
priority: high
created: 2026-02-27
author: "@human"
---

# FEAT-132: Integration Testing Infrastructure (testenv)

## Summary

Basil has a class of features that cannot be meaningfully tested without live external services: the Fetch operator (`<=/=`), the SFTP driver, the Git HTTP server, and the upcoming email notification API (FEAT-084). Currently these features have no integration tests, or carry explicit comments noting that real-service testing is impossible. This spec introduces a shared `testenv` Go package that starts lightweight in-process fake servers, allowing full integration tests to run with a single `go test ./...` command and no external setup.

The package is test-only (never compiled into the Basil binary) and follows the same pattern as Go's own `net/http/httptest`: call `testenv.Start(t, ...)`, get back a struct of addresses and credentials, write assertions, and let `t.Cleanup` handle teardown.

## User Story

As a developer (human or AI agent) working on Basil, I want integration tests for network-dependent features to run locally with `go test ./...` so that I can verify correct behaviour without managing external services, Docker containers, or third-party accounts.

## Acceptance Criteria

### Phase 1: HTTPS and SMTP

- [x] `testenv` package exists at `basil/testenv/`
- [x] `testenv.Start(t, opts...)` starts requested servers, returns an `Env` struct, and registers `t.Cleanup` for teardown
- [x] All servers bind to `:0` (OS-assigned random port) — no hardcoded ports
- [x] **HTTPS:** fake server starts via `httptest.NewTLSServer`; `Env.HTTPSURL` is set; test client trusts the self-signed cert
- [x] **HTTPS fixtures:** helpers to serve JSON, plain text, a redirect, and configurable HTTP error codes (4xx/5xx)
- [x] **SMTP:** fake server starts via `go-smtp`; `Env.SMTPAddr` is set; sent messages are captured in memory
- [x] **SMTP:** `Env.Messages()` returns all captured messages with To, From, Subject, and body accessible
- [x] **SMTP:** `Env.LastMessage()` convenience helper returns the most recently received message
- [x] Integration tests added for the Fetch (`<=/=`) operator using the fake HTTPS server
- [x] Integration tests added for `basil.email.send()` (FEAT-084) using the fake SMTP server, or a placeholder test registered for when FEAT-084 lands
- [x] All new tests pass with `go test ./...`

### Phase 2: SFTP/SSH

- [x] **SFTP:** fake SSH/SFTP server starts in-process; `Env.SFTPAddr`, `Env.SFTPUser`, `Env.SFTPPassword` are set
- [x] Fake SFTP server serves files from a `t.TempDir()` directory
- [x] Test fixture helpers: `Env.SFTPWriteFile(path, content)` and `Env.SFTPReadFile(path)` for setup/assertion
- [x] Integration tests added for SFTP connection establishment, file read, file write, and directory listing
- [x] Integration tests added for SFTP connection cache (`connection_cache.go`) — filling the gap noted in `connection_cache_test.go`
- [x] Integration tests added for SFTP auth failure (wrong password returns correct Basil error class)
- [x] Integration tests added for SFTP error paths: missing file, permission denied
- [x] All new tests pass with `go test ./...`

---

### Phase 4: SFTP File Operations

Integration tests for SFTP file handle methods (`mkdir`, `rmdir`, `remove`) and connection methods (`close`).

- [ ] `mkdir()` — create a directory on the SFTP server
- [ ] `mkdir({parents: true})` — create nested directories recursively
- [ ] `rmdir()` — remove an empty directory
- [ ] `remove()` — delete a file
- [ ] `close()` — explicitly close an SFTP connection
- [ ] All new tests pass with `go test ./...`

**Known limitation:** `rmdir({recursive: true})` is parsed but not implemented (TODO in `eval_network_io.go:96`). Test should verify it returns an appropriate error or behaves as non-recursive.

---

### Phase 5: SFTP Format Coverage

Integration tests for all SFTP read and write formats beyond `.text`, `.json`, and `.dir`.

**Read formats:**
- [ ] `.lines` — read file and split by newlines into array
- [ ] `.csv` — parse CSV file with headers into array of dictionaries
- [ ] `.bytes` — read file as byte array (array of integers 0-255)
- [ ] `.file` — auto-detect format from file extension

**Write formats:**
- [ ] `.json` — write object/dictionary as JSON
- [ ] `.lines` — write array of strings joined by newlines
- [ ] `.bytes` — write array of integers as bytes

**Error cases:**
- [ ] `.csv` write — verify returns error `SFTP-0003` ("CSV write not yet implemented")
- [ ] Unknown format — verify returns appropriate error

- [ ] All new tests pass with `go test ./...`

---

### Phase 6: Fetch Format Coverage

Integration tests for Fetch operator response formats beyond `text` and `json`.

- [ ] `yaml` — parse YAML response
- [ ] `lines` — split response by newlines into array
- [ ] `bytes` — read response as byte array

- [ ] All new tests pass with `go test ./...`

---

### Phase 7: Embedded Postgres (deferred — post-v1.0 if needed)

Not in scope for this spec. Tracked in the backlog. Trigger: a Postgres-specific bug is reported, or the Postgres driver changes significantly. See `work/reports/INTEGRATION-TESTING-INFRASTRUCTURE.md` for the full rationale and recommended library (`fergusstrange/embedded-postgres`).

---

## Design Decisions

- **Package, not binary.** `testenv` is imported by `_test.go` files only. It is never compiled into the Basil binary. No CLI tooling, no separate process to start.
- **`t.Cleanup`, not `defer`.** All server teardown is registered with `t.Cleanup(func() { ... })` inside `testenv.Start`. This ensures cleanup runs even on `t.Fatal` and works correctly with `t.Parallel`.
- **Random ports (`:0`).** Servers never use hardcoded ports. This allows multiple test packages to run in parallel without conflict.
- **Stdlib first.** The HTTPS server uses `net/http/httptest` from the standard library — no new dependency. The SMTP and SFTP servers each add one test-only dependency.
- **Reuse existing deps.** `pkg/sftp` is already a direct Basil dependency and includes a server implementation. `gliderlabs/ssh` is the standard Go SSH server library. Neither is a novel addition to the module graph.
- **No Docker.** Docker is hostile to the AI-agent development workflow (requires out-of-band setup, slow container start, cannot be managed during `go test`). Rejected as the primary strategy.
- **No live servers.** External services introduce flakiness, require credentials, break offline development, and are unavailable to AI agents. Rejected entirely for automated tests.
- **MySQL skipped.** The `database/sql` driver interface is consistent across drivers. MySQL-specific bugs would be unusual, and the embedded MySQL story is poor. Revisit only if a concrete bug is reported.

## Technical Context

### Affected Components

- `testenv/testenv.go` — new file: `Start()`, functional options, `Env` struct
- `testenv/https.go` — new file: HTTPS fake server and fixture helpers
- `testenv/smtp.go` — new file: SMTP fake server, in-memory backend, message accessors
- `testenv/sftp.go` — new file: SSH/SFTP fake server, temp-dir filesystem, fixture helpers
- `pkg/parsley/evaluator/connection_cache_test.go` — add SFTP integration tests (Phase 2)
- `pkg/parsley/evaluator/eval_fetch_test.go` — new file: Fetch operator integration tests (Phase 1)

### Dependencies

New test-only dependencies (not compiled into Basil binary):

| Package | Used for | Phase |
|---------|----------|-------|
| `github.com/emersion/go-smtp` | Fake SMTP server | 1 |
| `github.com/gliderlabs/ssh` | SSH transport for fake SFTP server | 2 |

`github.com/pkg/sftp` is already a direct dependency and includes the SFTP server implementation used in Phase 2.

`net/http/httptest` is Go standard library — no new dependency.

- **Depends on:** None. This is infrastructure; it does not depend on any other feature spec.
- **Blocks:** FEAT-084 (email) integration tests depend on Phase 1 SMTP. Any future SFTP bug fixes benefit from Phase 2.

### Package Structure

```
basil/
└── testenv/
    ├── testenv.go     // Start(), WithHTTPS(), WithSMTP(), WithSFTP(), Env struct
    ├── https.go       // httptest.NewTLSServer wrapper + JSON/text/redirect/error fixtures
    ├── smtp.go        // go-smtp server + in-memory backend + Messages()/LastMessage()
    └── sftp.go        // gliderlabs/ssh + pkg/sftp server + TempDir filesystem + helpers
```

### API Shape

```go
// Start selected fake servers. Servers stop when t ends.
env := testenv.Start(t,
    testenv.WithHTTPS(),
    testenv.WithSMTP(),
    testenv.WithSFTP(),
)

// HTTPS
env.HTTPSURL               // "https://127.0.0.1:PORT"
env.ServeJSON(path, v)     // register a JSON fixture at path
env.ServeText(path, text)  // register a plain-text fixture
env.ServeError(path, code) // register a fixed HTTP error code

// SMTP
env.SMTPAddr               // "127.0.0.1:PORT"
env.Messages()             // []*testenv.Message — all captured messages
env.LastMessage()          // *testenv.Message — most recent message

// SFTP
env.SFTPAddr               // "127.0.0.1:PORT"
env.SFTPUser               // "testuser"
env.SFTPPassword           // "testpass"
env.SFTPWriteFile(path, content)
env.SFTPReadFile(path) string
```

### Edge Cases & Constraints

1. **TLS trust** — `httptest.NewTLSServer` returns a `*httptest.Server` with a `.Client()` method that returns an `*http.Client` pre-configured to trust the self-signed cert. Basil's Fetch evaluator must accept an injected client or transport for tests. If it does not already, a small refactor to accept an `http.Client` option will be needed.
2. **SFTP key vs password auth** — The fake SFTP server supports password auth only (simpler). Basil's SFTP client supports both; key-based auth remains covered by unit tests of the key-loading logic.
3. **go-smtp STARTTLS** — The fake SMTP server runs plain SMTP (no TLS) on localhost. This is appropriate for test use; production email (FEAT-084) will use TLS to real providers.
4. **Parallel tests** — Because all servers bind to `:0`, test packages using `testenv` are safe to run with `t.Parallel()`.
5. **Test build tag** — Consider a `//go:build integration` tag if the SFTP server's startup time proves disruptive to the normal unit test run. Start without it; add only if needed.

## Implementation Notes

*To be filled in during implementation.*

## Related

- Report: `work/reports/INTEGRATION-TESTING-INFRASTRUCTURE.md`
- FEAT-084: Email notification API (primary consumer of Phase 1 SMTP)
- Backlog item: Phase 3 Embedded Postgres
```

Now update the ID counter: