---
id: PLAN-112
feature: FEAT-132
title: "Implementation Plan for Integration Testing Infrastructure (testenv)"
status: draft
created: 2026-02-27
---

# Implementation Plan: FEAT-132 (Integration Testing Infrastructure)

## Overview

This plan implements a shared `testenv` Go package that starts lightweight in-process fake servers for HTTPS, SMTP, and SFTP. This allows integration tests for the Fetch operator, email (FEAT-084), and SFTP driver to run with a plain `go test ./...` and no external setup.

The work is split into two phases matching the spec:
- **Phase 1:** HTTPS fake server + SMTP fake server, plus Fetch operator integration tests
- **Phase 2:** SFTP/SSH fake server, plus SFTP evaluator integration tests

Phase 3 (embedded Postgres) is explicitly out of scope and tracked in the backlog.

## Branch

`feat/FEAT-132-testenv`

## Prerequisites

- [ ] Working tree is clean (`git status`)
- [ ] `go test ./...` passes on the current branch before any changes
- [ ] Read `work/reports/INTEGRATION-TESTING-INFRASTRUCTURE.md`
- [ ] Read `work/specs/FEAT-132.md`

---

## Phase 1: HTTPS and SMTP

### Task 1A: Create the `testenv` package skeleton

**Files:** `testenv/testenv.go` (new)
**Estimated effort:** Small

The `testenv` package is the entry point. It defines the `Env` struct, the functional-options pattern, and the `Start` function. Individual server implementations (HTTPS, SMTP, SFTP) are added in subsequent tasks and populate fields on `Env`.

Steps:
1. Create `testenv/` directory at the repo root (alongside `server/`, `auth/`, etc.)
2. Create `testenv/testenv.go` with:
   - `package testenv`
   - `options` struct with `wantHTTPS`, `wantSMTP`, `wantSFTP` bool fields
   - `Option` functional type: `type Option func(*options)`
   - `WithHTTPS() Option`, `WithSMTP() Option`, `WithSFTP() Option` constructors
   - `Env` struct with all fields for all phases (unexported server handles, exported address/credential strings)
   - `Start(t testing.TB, opts ...Option) *Env` — calls per-server start functions based on options, returns populated `Env`
3. No server logic in this file — just the scaffold

Tests:
- `testenv/testenv_test.go`: `TestStart_NoOptions` — `Start(t)` with no options returns a non-nil `Env` without panicking
- `TestStart_AllOptions` — deferred until all servers exist; add as a TODO comment

Commit: `feat(testenv): add testenv package skeleton with options and Env struct`

---

### Task 1B: Fake HTTPS server

**Files:** `testenv/https.go` (new)
**Estimated effort:** Small

Uses `net/http/httptest.NewTLSServer` from the standard library — no new dependency.

Steps:
1. Create `testenv/https.go`
2. Implement `startHTTPS(t testing.TB, env *Env)`:
   - Create an `http.ServeMux`
   - Start `httptest.NewTLSServer(mux)`
   - Set `env.HTTPSURL = server.URL`
   - Set `env.HTTPSClient = server.Client()` (pre-configured to trust the self-signed cert)
   - Register `t.Cleanup(server.Close)`
   - Store `mux` on `env` (unexported) for use by fixture helpers
3. Implement fixture helpers on `*Env`:
   - `ServeJSON(path string, v any)` — registers a handler that marshals `v` as JSON with `Content-Type: application/json`
   - `ServeText(path string, text string)` — registers a plain-text handler
   - `ServeRedirect(path string, target string, code int)` — registers an HTTP redirect handler
   - `ServeError(path string, code int)` — registers a handler that returns the given status code with an empty body
4. Add `HTTPSURL string` and `HTTPSClient *http.Client` fields to `Env` in `testenv.go`

Tests:
- `testenv/https_test.go`:
  - `TestHTTPS_ServeJSON` — start env with HTTPS, register JSON fixture, fetch with `env.HTTPSClient`, assert body and Content-Type
  - `TestHTTPS_ServeText` — same for plain text
  - `TestHTTPS_ServeError` — register a 404 fixture, assert status code
  - `TestHTTPS_ServeRedirect` — register a redirect, follow it, assert final URL
  - `TestHTTPS_TLSRequired` — a plain `http.DefaultClient` (not `env.HTTPSClient`) should fail with a cert error, confirming TLS is real

Commit: `feat(testenv): add fake HTTPS server with JSON/text/error/redirect fixtures`

---

### Task 1C: Fake SMTP server

**Files:** `testenv/smtp.go` (new)
**Estimated effort:** Small–Medium

Requires one new test-only dependency: `github.com/emersion/go-smtp`.

Steps:
1. Add the dependency: `go get github.com/emersion/go-smtp`
2. Run `go mod tidy`
3. Create `testenv/smtp.go`
4. Implement an in-memory SMTP backend satisfying `go-smtp`'s `Backend` and `Session` interfaces:
   - `Backend.NewSession` returns a new `*smtpSession`
   - `smtpSession` accumulates `From`, `To`, `Data` (raw message bytes) per message
   - On `smtpSession.Data`, parse `Subject` from the raw headers and store a completed `*Message` on a shared (mutex-protected) slice on `Env`
5. Define `testenv.Message` struct: `From string`, `To []string`, `Subject string`, `Body string`
6. Implement `startSMTP(t testing.TB, env *Env)`:
   - Create and configure `smtp.NewServer(backend)`
   - Start server on `:0` — capture the assigned port via `net.Listen` before passing to the server, or read from `server.Addr` after `ListenAndServe`
   - Set `env.SMTPAddr`
   - Register `t.Cleanup(server.Close)`
7. Implement accessors on `*Env`:
   - `Messages() []*Message` — returns a copy of captured messages (thread-safe)
   - `LastMessage() *Message` — returns the last captured message, or nil
   - `WaitForMessage(t testing.TB, timeout time.Duration) *Message` — polls until a message arrives or timeout, calls `t.Fatal` on timeout (useful in async send tests)
8. Add `SMTPAddr string` field to `Env` in `testenv.go`

Note on port binding: `go-smtp`'s `ListenAndServe` blocks, so run it in a goroutine. Use `net.Listen("tcp", "127.0.0.1:0")` to pre-bind and get the port, then pass the listener to the server via `server.Serve(ln)`.

Tests:
- `testenv/smtp_test.go`:
  - `TestSMTP_ReceiveMessage` — connect with `net/smtp`, send a message to `env.SMTPAddr`, assert `env.LastMessage()` has correct From, To, Subject, Body
  - `TestSMTP_MultipleMessages` — send three messages, assert `len(env.Messages()) == 3`
  - `TestSMTP_WaitForMessage` — send asynchronously, assert `WaitForMessage` returns before timeout
  - `TestSMTP_NoMessages` — assert `env.LastMessage()` returns nil when nothing has been sent

Commit: `feat(testenv): add fake SMTP server with in-memory message capture`

---

### Task 1D: Fetch operator integration tests

**Files:** `pkg/parsley/evaluator/eval_network_io_test.go` (new)
**Estimated effort:** Medium

This is where Phase 1 pays off. We exercise the `<=/=` fetch operator end-to-end against the fake HTTPS server.

Before writing tests, read `fetchUrlContentFull` in `eval_network_io.go`. It constructs its own `http.Client` locally with no injection point:

```go
client := &http.Client{Timeout: timeout}
```

This means a TLS server with a self-signed cert will fail certificate validation. A small, targeted refactor is needed first.

Steps:
1. **Refactor `fetchUrlContentFull`** to accept an optional `*http.Client` parameter (or add a package-level `var testHTTPClient *http.Client` that defaults to nil and is used when set). The package-level var approach is simpler and avoids touching the function signature or callers:
   - Add `var testHTTPClient *http.Client` to `eval_network_io.go`
   - In `fetchUrlContentFull`, replace `client := &http.Client{Timeout: timeout}` with:
     ```go
     client := testHTTPClient
     if client == nil {
         client = &http.Client{Timeout: timeout}
     }
     ```
   - This is test-only coupling but is minimal and self-documenting
2. Create `eval_network_io_test.go` in `pkg/parsley/evaluator/`
3. In `TestMain` (or a helper), set `testHTTPClient = testenv.Start(...).HTTPSClient` for the test run
4. Write integration tests using Parsley source strings evaluated with `testEval(src)`:
   - `TestFetch_JSON` — serve `{"name":"basil"}` at `/data.json`, evaluate `let x <=/= jsonFile(@https://HOST/data.json)`, assert `x.name == "basil"`
   - `TestFetch_Text` — serve plain text, evaluate fetch, assert content
   - `TestFetch_404` — serve a 404, evaluate fetch with error capture `{data, error} <=/= ...`, assert `error != null`
   - `TestFetch_Redirect` — serve a redirect chain, assert final URL is the redirect target
   - `TestFetch_InvalidURL` — evaluate `<=/= jsonFile(@https://127.0.0.1:1/bad)`, assert error propagates cleanly
   - `TestFetch_ErrorCapture` — assert the `{data, error}` destructuring pattern works for both success and failure

Tests are the feature for this task — no separate tests section.

Commit: `test(evaluator): add Fetch operator integration tests using fake HTTPS server`

---

### Phase 1 Validation

- [ ] `go test ./testenv/...` passes
- [ ] `go test ./pkg/parsley/evaluator/...` passes (including new fetch tests)
- [ ] `go test ./...` passes (no regressions)
- [ ] `go mod tidy` — go.sum is clean
- [ ] `golangci-lint run` — no new issues

---

## Phase 2: SFTP/SSH

### Task 2A: Fake SFTP server

**Files:** `testenv/sftp.go` (new)
**Estimated effort:** Medium

Uses `github.com/gliderlabs/ssh` for the SSH transport and the server mode of `github.com/pkg/sftp` (already a direct dependency) for the SFTP subsystem.

Steps:
1. Add the new test-only dependency: `go get github.com/gliderlabs/ssh`
2. Run `go mod tidy`
3. Create `testenv/sftp.go`
4. Implement `startSFTP(t testing.TB, env *Env)`:
   - Generate a throw-away RSA host key with `rsa.GenerateKey` (or use `ssh.GenerateKey`) — no file needed, purely in-memory
   - Create a `gliderlabs/ssh.Server` with:
     - `PasswordHandler`: accepts `env.SFTPUser` / `env.SFTPPassword` only, rejects everything else
     - `SubsystemHandlers["sftp"]`: creates a `pkg/sftp.NewRequestServer` rooted at `t.TempDir()`
   - Listen on `127.0.0.1:0`, capture port, set `env.SFTPAddr`
   - Run `server.ListenAndServe()` in a goroutine
   - Register `t.Cleanup(server.Close)`
5. Set fixed `env.SFTPUser = "testuser"` and `env.SFTPPassword = "testpass"`
6. Implement filesystem helpers on `*Env`:
   - `SFTPWriteFile(path, content string)` — writes `content` to `filepath.Join(tempDir, path)`; creates parent dirs as needed
   - `SFTPDir(path string) string` — returns the absolute path to a subdirectory within the temp root (for setup)
7. Add `SFTPAddr`, `SFTPUser`, `SFTPPassword string` fields to `Env` in `testenv.go`

Note on `pkg/sftp` server mode: use `sftp.NewRequestServer(channel, sftp.WithRootDirectory(root))` inside the SubsystemHandler. The `channel` is the `ssh.Session` from gliderlabs.

Tests:
- `testenv/sftp_test.go` (tests the fake server itself using a real `pkg/sftp` client):
  - `TestSFTP_Connect` — dial the fake server with correct credentials, assert connection succeeds
  - `TestSFTP_BadPassword` — dial with wrong password, assert error
  - `TestSFTP_WriteAndRead` — use `SFTPWriteFile` to seed a file, read it via sftp client, assert content matches
  - `TestSFTP_List` — write two files, list the directory via sftp client, assert both names present
  - `TestSFTP_MissingFile` — stat a non-existent path, assert error

Commit: `feat(testenv): add fake SFTP server using gliderlabs/ssh and pkg/sftp`

---

### Task 2B: SFTP evaluator integration tests

**Files:** `pkg/parsley/evaluator/eval_sftp_integration_test.go` (new)
**Estimated effort:** Medium

This directly addresses the gap documented in `connection_cache_test.go` and `introspect_validation_test.go`.

Steps:
1. Create `eval_sftp_integration_test.go` in `pkg/parsley/evaluator/`
2. Add a `TestMain` or package-level setup that starts a `testenv` with SFTP and stores `env.SFTPAddr`, `env.SFTPUser`, `env.SFTPPassword` in package-level vars accessible to tests
3. Write integration tests. These evaluate Parsley source that uses `@sftp(host, user, pass)` connection literals. Use a helper `evalWithSFTP(src string) Object` that substitutes the fake server address into the source:

   - `TestSFTPEval_Connect` — evaluate `@sftp("HOST", PORT, "testuser", "testpass")`, assert result type is `SFTP_CONNECTION_OBJ`
   - `TestSFTPEval_ReadTextFile` — seed a text file via `env.SFTPWriteFile`, evaluate `<=/= conn(@/file.txt).text`, assert content
   - `TestSFTPEval_ReadJSONFile` — seed a JSON file, evaluate `<=/= conn(@/data.json).json`, assert parsed structure
   - `TestSFTPEval_ListDirectory` — seed two files, evaluate `<=/= conn(@/).dir`, assert array contains both filenames
   - `TestSFTPEval_WriteFile` — evaluate a write statement `conn(@/out.txt) ==> "hello"`, assert `env.SFTPReadFile("/out.txt") == "hello"` (once write is supported)
   - `TestSFTPEval_BadPassword` — evaluate connection with wrong password, assert error class is `network`
   - `TestSFTPEval_MissingFile` — evaluate fetch of non-existent path, assert error class is `io`
   - `TestSFTPEval_ConnectionCache` — connect twice with the same credentials, assert the second call returns the cached connection (verify `sftpCache.size()` does not increase)
   - `TestSFTPEval_CacheHealthCheck` — connect, forcibly close the underlying SSH client, connect again; assert a new connection is established (cache evicts the dead entry)

4. Update `introspect_validation_test.go`: remove `sftpconnection` and `sftpfile` from the `skippedTypes` map, or add a note that they are now covered by `eval_sftp_integration_test.go`

Tests are the feature for this task.

Commit: `test(evaluator): add SFTP integration tests covering connection, file I/O, cache, and error paths`

---

### Phase 2 Validation

- [ ] `go test ./testenv/...` passes
- [ ] `go test ./pkg/parsley/evaluator/...` passes (including new SFTP tests)
- [ ] `go test ./...` passes (no regressions)
- [ ] `go mod tidy` — go.sum is clean
- [ ] `golangci-lint run` — no new issues
- [ ] `introspect_validation_test.go` skipped-types comment is updated

---

## Final Validation Checklist

- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] No hardcoded ports anywhere in `testenv/`
- [ ] No `testenv` import in any non-`_test.go` file
- [ ] `go mod tidy` — no phantom entries in `go.sum`
- [ ] FEAT-132 acceptance criteria checked off in `work/specs/FEAT-132.md`
- [ ] Backlog updated with Phase 3 (embedded Postgres) deferred item

---

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-02-27 | 1A: testenv skeleton | ✅ Complete | `testenv/testenv.go` — Env struct, options, Start() |
| 2026-02-27 | 1B: Fake HTTPS server | ✅ Complete | `testenv/https.go` — httptest.NewTLSServer + fixtures |
| 2026-02-27 | 1C: Fake SMTP server | ✅ Complete | `testenv/smtp.go` — go-smtp + in-memory backend |
| 2026-02-27 | 1D: Fetch integration tests | ✅ Complete | `eval_network_io_test.go` — 6 tests; added testHTTPClient injection point in fetchUrlContentFull |
| 2026-02-27 | 2A: Fake SFTP server | ✅ Complete | `testenv/sftp.go` — gliderlabs/ssh + pkg/sftp request server with OS handler rooted at TempDir |
| 2026-02-27 | 2B: SFTP evaluator integration tests | ✅ Complete | `eval_sftp_integration_test.go` — 7 tests; also fixed nil *Error-in-interface bug in evalSFTPRead |

---

## Notes From Implementation

- `sftp.WithRootDirectory` does not exist in `pkg/sftp@v1.13.10`. Used `sftp.NewRequestServer` with a custom `osHandler` implementing `sftp.Handlers` (FileGet/FilePut/FileCmd/FileList interfaces) rooted at the temp directory.
- Fixed a pre-existing nil `*Error`-in-interface bug in `evalSFTPRead` (`eval_file_io.go`): `return parseJSON(...)` returned a nil `*Error` as a non-nil `Object` interface, causing a nil pointer dereference when the error was type-asserted. Unwrapped explicitly.
- The `testHTTPClient` injection var approach (package-level var, nil in production) is minimal and self-documenting. No callers changed.
- Fetch `{data, error}` pattern only captures network-level failures in the `error` field; HTTP 4xx/5xx status codes surface via `status`. Tests adjusted to match actual semantics.
- Pre-existing failures `TestDatabaseShutdown` and `TestSiteHandler_RootPath` in `server/` are unrelated to this work (confirmed on `main` before changes).

## Deferred Items

Items to add to `work/BACKLOG.md` after implementation:

- **Phase 3: Embedded Postgres** — `github.com/fergusstrange/embedded-postgres`. Trigger: a Postgres-specific bug is reported or the Postgres driver changes. See `work/reports/INTEGRATION-TESTING-INFRASTRUCTURE.md` for full rationale.
- **`testenv.WithGit()`** — fake Git HTTP server for testing the `GitHandler` auth wrapping in `server/git.go`. Low priority: `go-git-http` is already a dep and the auth layer can be tested with a plain `httptest.Server`. Add if a Git auth bug is reported.
- **Build tag for SFTP tests** — if `eval_sftp_integration_test.go` proves slow enough to disrupt the normal unit test loop, add `//go:build integration` and a `make test-integration` target. Start without it.

## Related

- Spec: `work/specs/FEAT-132.md`
- Report: `work/reports/INTEGRATION-TESTING-INFRASTRUCTURE.md`
- FEAT-084: Email notification API (Phase 1 SMTP is a prerequisite for its integration tests)
- `pkg/parsley/evaluator/eval_network_io.go` — `fetchUrlContentFull`, target of Task 1D refactor
- `pkg/parsley/evaluator/connection_cache_test.go` — documents the SFTP testing gap addressed in Task 2B
- `pkg/parsley/evaluator/introspect_validation_test.go` — lists `sftpconnection` as a skipped type; updated in Task 2B