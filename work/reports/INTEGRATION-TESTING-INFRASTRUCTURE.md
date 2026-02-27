# Integration Testing Infrastructure

**Date:** 2026-02-27
**Status:** Complete
**Author:** AI Assistant

---

## Background

Basil has excellent unit test coverage for its core language (Parsley) and most server logic. However, a class of features cannot be meaningfully tested without external services:

- **Fetch** (`<=/=` operator) — requires an HTTP/HTTPS server to respond to
- **SFTP** — requires an SSH daemon with file system access
- **Git server** — requires a Git HTTP endpoint with auth
- **Email (SMTP)** — requires a mail server to accept and capture messages
- **Database drivers** — SQLite is already in-process; Postgres/MySQL require a server

Currently, tests for these features either do not exist, test only the happy path with mocks at a very shallow level, or carry comments like:

> "We can't easily test the SFTP cache with a mock connection because the health check requires a real SSH client."

This is a gap that must be addressed before v1.0. As Basil adds more integrations (the email notification API in FEAT-084, the completed SFTP driver, the Fetch operator), the untested surface area grows.

A secondary but important constraint: **AI coding agents must be able to run the full test suite with a single command.** Any solution that requires out-of-band setup (Docker, external accounts, running a daemon first) effectively means agents cannot run integration tests at all, which defeats the purpose.

---

## Guiding Principles

1. **Local first.** Tests must run with `go test ./...` and no other setup. No Docker, no external services, no credentials.
2. **Fast.** Integration servers must start in under 100ms. Tests that take seconds per run will be skipped.
3. **Self-cleaning.** Servers bind to `:0` (random port), use temp directories, and are torn down via `t.Cleanup()`.
4. **Use existing dependencies where possible.** Basil already depends on `go-git-http`, `pkg/sftp`, and `golang.org/x/crypto/ssh`. Reuse these rather than adding new transitive deps.
5. **Good enough, not perfect.** We are not testing the third-party libraries themselves. We are testing Basil's integration layer.

---

## Strategy

The recommended approach is a shared **`testenv` Go package** at `basil/testenv/`. It is not a binary — it is an importable test helper. Each test suite calls `testenv.Start(t, ...)` to get a struct of live addresses and credentials for whatever services that test needs.

```
// In any *_test.go file:
env := testenv.Start(t, testenv.WithHTTPS(), testenv.WithSFTP())
// env.HTTPSURL  — e.g. "https://127.0.0.1:52341"
// env.SFTPAddr  — e.g. "127.0.0.1:52342"
// t.Cleanup is registered automatically; servers stop when test ends
```

This is the same pattern as Go's own `net/http/httptest` and `database/sql` test helpers. It requires no changes to `go test` invocations and is fully transparent to CI.

---

## Rejected Alternatives

### Docker / docker compose

**Rejected.** Docker is great for reproducing environments but is hostile to the agent-assisted workflow that is central to Basil's development. Specific problems:

- Requires Docker Desktop (macOS license concerns) or a Docker-compatible runtime to be installed and running
- Container startup time for Postgres is 3–10 seconds; MySQL is longer
- AI agents cannot manage container lifecycle during a coding session
- Health-check timing is fiddly and flaky
- Cannot easily introspect state (e.g. "what email was just sent?")
- Adds a mandatory out-of-band step before `go test` works

Docker may be appropriate as a *CI-only* strategy for real Postgres/MySQL if those drivers need testing against a real server. It is not appropriate as the primary integration test strategy.

### Live servers / third-party accounts

**Rejected.** Live servers break offline development entirely, are unavailable to AI agents, introduce flakiness from network conditions and rate limits, and risk leaking test data into production systems. Third-party accounts (Supabase, SendGrid, etc.) add credential management overhead and external dependencies that have nothing to do with the correctness of Basil's code.

### Shallow mocks at the evaluator boundary

**Rejected as the sole strategy.** Mocking the SFTP client interface, for example, tests that Basil calls the right methods — but it does not test that the connection is established correctly, that auth works, that the path handling is right, or that errors are propagated correctly. Mocks have their place at the unit level but cannot replace integration tests for network services.

---

## Recommended Implementation

### Phase 1 — HTTP/HTTPS and SMTP (Recommended for v1.0)

These two cover the Fetch operator and the upcoming email notification API (FEAT-084). Both are trivial to implement.

#### 1a. Fake HTTPS server

**Library:** `net/http/httptest` (Go standard library — zero new dependencies)

`httptest.NewTLSServer(handler)` starts a real HTTPS server on a random port with a self-signed certificate. The test client is pre-configured to trust it. This is already the standard Go way to test HTTP clients.

**What it enables:**
- Testing the `<=/=` fetch operator against a real TLS endpoint
- Testing redirect handling, error status codes, timeouts
- Testing JSON/CSV/text fetch formats end-to-end
- Testing that Basil's HTTP client correctly rejects untrusted certificates in production mode

**Effort:** Very small. `httptest.NewTLSServer` is one line. The testenv wrapper adds fixture helpers (serve JSON, serve CSV, serve a file, return 4xx/5xx).

#### 1b. Fake SMTP server

**Library:** `github.com/emersion/go-smtp` (~500 stars, actively maintained, permissive BSD licence)

This library provides a simple SMTP server that can be embedded in a test process. The backend is a Go interface — implementing it with an in-memory message store is around 50 lines.

**What it enables:**
- Testing that `basil.email.send()` (FEAT-084) actually delivers a message
- Asserting on the To, From, Subject, and body of sent email
- Testing that failed sends are reported correctly
- Testing rate limiting (50/hr, 200/day per the FEAT-084 spec)
- Future: testing email verification flows end-to-end

**New dependency:** `github.com/emersion/go-smtp` — test-only, not compiled into the Basil binary.

**Effort:** Small. ~100 lines for the testenv wrapper including the in-memory backend and a `Messages()` helper to assert on sent mail.

---

### Phase 2 — SFTP/SSH (Recommended for v1.0)

SFTP is already fully implemented in the evaluator. The connection cache, error types, and file handle logic are all there. What does not exist is any test that actually connects to an SFTP server. The comment in `connection_cache_test.go` explicitly flags this gap.

#### Fake SFTP server

**Libraries:**
- `github.com/gliderlabs/ssh` — SSH server framework (already a transitive dependency via `golang.org/x/crypto/ssh`)
- `github.com/pkg/sftp` — already a direct Basil dependency (used in the client)

`pkg/sftp` includes an SFTP server implementation (the `sftp.NewRequestServer` API) that can be attached to any `io.ReadWriteCloser`. Combined with `gliderlabs/ssh` for the SSH transport, a fully functional in-process SFTP server can be built in under 150 lines. It serves files from a `t.TempDir()` directory.

**What it enables:**
- End-to-end tests of the SFTP connection cache (filling the gap flagged in `connection_cache_test.go`)
- Testing SFTP auth (password, key-based)
- Testing file read/write/list operations
- Testing connection reuse and TTL eviction
- Testing error paths: missing files, permission denied, broken connections

**New dependency:** `github.com/gliderlabs/ssh` — test-only. `pkg/sftp` server mode is already available (it's in the same package as the client).

**Effort:** Medium. ~150 lines for the SSH/SFTP test server, plus test cases for the existing evaluator code.

---

### Phase 3 — Postgres (Recommended post-v1.0 if needed)

#### Embedded Postgres

**Library:** `github.com/fergusstrange/embedded-postgres`

This library downloads a real Postgres binary on first use (cached in `~/.embedded-postgres-go/`), extracts it to a temp directory, and manages its lifecycle programmatically. It is the cleanest way to get a real Postgres server without Docker.

**What it enables:**
- Testing Postgres-specific SQL behaviour that SQLite does not cover
- Testing the Postgres connection string parsing
- Testing Postgres-specific error codes and their mapping to Basil errors

**Why defer:** The Go `database/sql` interface is consistent across drivers. If the SQLite integration tests pass, Postgres-specific bugs are likely to be in SQL dialect differences or connection string handling — both narrow, low-risk areas. The embedded-postgres download (~30MB) also adds first-run latency that is best accepted only when the tests are actually needed.

**MySQL:** Skip indefinitely. MySQL and MariaDB cover a small portion of Basil's user base and the `database/sql` driver interface means MySQL-specific bugs would be extremely unusual. Revisit only if a concrete MySQL bug is reported.

---

## Package Structure

```
basil/
└── testenv/
    ├── testenv.go        // Start(), options, Env struct
    ├── https.go          // fake HTTPS server (httptest wrapper + fixtures)
    ├── smtp.go           // fake SMTP server (go-smtp + in-memory backend)
    ├── sftp.go           // fake SFTP server (gliderlabs/ssh + pkg/sftp)
    └── postgres.go       // embedded Postgres (Phase 3, behind build tag)
```

The package is only imported in `_test.go` files, so none of the test server code is compiled into the Basil binary.

---

## Summary Table

| Service | Phase | Library | New Dep? | Effort | What it tests |
|---------|-------|---------|----------|--------|---------------|
| HTTPS | 1 | `net/http/httptest` (stdlib) | No | Very small | Fetch operator, TLS, redirects, error codes |
| SMTP | 1 | `go-smtp` | Yes (test-only) | Small | Email send, rate limiting, delivery errors |
| SFTP/SSH | 2 | `gliderlabs/ssh` + `pkg/sftp` server | Yes (test-only) | Medium | Connection cache, file ops, auth, error paths |
| Postgres | 3 | `embedded-postgres` | Yes (test-only) | Medium | Postgres dialect, connection strings |
| MySQL | — | — | — | — | Skip |
| Git HTTP | — | `go-git-http` (existing dep) | No | Small | Auth wrapping (use `httptest`) |

---

## Recommended Next Steps

1. Create feature spec **FEAT-132** covering Phase 1 (HTTPS + SMTP testenv) and Phase 2 (SFTP testenv).
2. Add a backlog item for Phase 3 (embedded Postgres) noting the trigger condition: "a Postgres-specific bug is reported or the Postgres driver changes significantly."
3. Update `AGENTS.md` once `testenv` exists: note that integration tests requiring external services should use `testenv.Start(t, ...)` and that `go test ./...` is always sufficient.