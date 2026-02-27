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

## Phase 2.5: Coverage Gaps

Post-implementation review identified five gaps between the FEAT-132 spec and the actual implementation. These are all small, self-contained tasks.

### Task 2.5A: SMTP placeholder test for FEAT-084

**Files:** `pkg/parsley/evaluator/eval_smtp_placeholder_test.go` (new)
**Estimated effort:** Tiny

The spec says: "Integration tests added for `basil.email.send()` (FEAT-084) using the fake SMTP server, **or a placeholder test registered for when FEAT-084 lands**." Neither exists.

Steps:
1. Create `pkg/parsley/evaluator/eval_smtp_placeholder_test.go`
2. Add a single test that starts a `testenv.WithSMTP()` env and calls `t.Skip`:

```go
func TestEmail_Send_Placeholder(t *testing.T) {
    // Verify the SMTP test infrastructure is functional.
    env := testenv.Start(t, testenv.WithSMTP())
    if env.SMTPAddr == "" {
        t.Fatal("SMTP server did not start")
    }
    t.Skip("FEAT-084 (email notification API) not yet implemented — enable this test when basil.email.send() lands")
}
```

This satisfies the spec criterion and serves as a reminder when FEAT-084 is implemented.

Commit: `test(evaluator): add SMTP placeholder test for FEAT-084 email integration`

---

### Task 2.5B: SFTP evaluator directory listing test

**Files:** `pkg/parsley/evaluator/eval_sftp_integration_test.go` (edit)
**Estimated effort:** Small

The spec requires "Integration tests added for SFTP … directory listing." There is a `TestSFTP_List` in `testenv/sftp_test.go` that tests the fake server directly, but no evaluator-level test exercising the `.dir` format accessor via Parsley syntax.

Steps:
1. Add `TestSFTPEval_ListDirectory` to `eval_sftp_integration_test.go`
2. Seed two files via `env.SFTPWriteFile`, then evaluate:

```go
func TestSFTPEval_ListDirectory(t *testing.T) {
    env := testenv.Start(t, testenv.WithSFTP())
    env.SFTPWriteFile("/listtest/alpha.txt", "a")
    env.SFTPWriteFile("/listtest/beta.txt", "b")

    src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
let {data, error} <=/= conn(@/listtest).dir
data
`)
    result := testEval(src)
    if isError(result) {
        t.Fatalf("unexpected error: %s", result.(*Error).Message)
    }
    arr, ok := result.(*Array)
    if !ok {
        t.Fatalf("expected Array, got %T: %s", result, result.Inspect())
    }
    // Check that both filenames appear in the listing.
    names := make(map[string]bool)
    for _, elem := range arr.Elements {
        names[elem.Inspect()] = true
    }
    if !names[`"alpha.txt"`] {
        t.Error("expected alpha.txt in directory listing")
    }
    if !names[`"beta.txt"`] {
        t.Error("expected beta.txt in directory listing")
    }
}
```

Note: The dir listing returns an array of dictionaries or strings depending on the evaluator implementation. Inspect the result of `pars -e` or `evalSFTPRead` with `format == "dir"` in `eval_file_io.go` to confirm the exact shape and adjust assertions accordingly. The key logic is at `eval_file_io.go` around line 272 — it builds dictionaries with `name`, `size`, `isDir`, `modified` keys.

Commit: `test(evaluator): add SFTP directory listing integration test`

---

### Task 2.5C: SFTP permission denied test

**Files:** `pkg/parsley/evaluator/eval_sftp_integration_test.go` (edit)
**Estimated effort:** Small

The spec lists "permission denied" as an explicit error path to test alongside "missing file." Only missing file was implemented.

Steps:
1. Add `TestSFTPEval_PermissionDenied` to `eval_sftp_integration_test.go`
2. Seed a file, then use `os.Chmod` to remove read permissions before attempting to read it via the evaluator:

```go
func TestSFTPEval_PermissionDenied(t *testing.T) {
    if runtime.GOOS == "windows" {
        t.Skip("permission denied semantics differ on Windows")
    }
    env := testenv.Start(t, testenv.WithSFTP())
    env.SFTPWriteFile("/secret.txt", "classified")

    // Remove read permission on the file so the SFTP server's os.OpenFile fails.
    fullPath := filepath.Join(env.SFTPRoot(), "secret.txt")
    if err := os.Chmod(fullPath, 0o000); err != nil {
        t.Fatalf("chmod failed: %v", err)
    }
    t.Cleanup(func() { _ = os.Chmod(fullPath, 0o644) }) // restore for TempDir cleanup

    src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
let {data, error} <=/= conn(@/secret.txt).text
error
`)
    result := testEval(src)
    if isError(result) {
        t.Fatalf("unexpected hard evaluator error: %s", result.(*Error).Message)
    }
    if result == NULL {
        t.Error("expected non-null error for permission denied, got null")
    }
}
```

This requires exposing the SFTP temp dir root. Add a `SFTPRoot() string` accessor to `Env`:

```go
// In testenv/testenv.go:
func (e *Env) SFTPRoot() string { return e.sftpRoot }
```

Commit: `test(evaluator): add SFTP permission denied integration test`

---

### Task 2.5D: Update `introspect_validation_test.go` skipped-types comments

**Files:** `pkg/parsley/evaluator/introspect_validation_test.go` (edit)
**Estimated effort:** Tiny

The plan (Task 2B) said: "Update `introspect_validation_test.go`: remove `sftpconnection` and `sftpfile` from the `skippedTypes` map, or add a note that they are now covered by `eval_sftp_integration_test.go`." This was not done.

Steps:
1. Update the comment block at the top (~line 26) to note that SFTP types are now covered:

```go
// SKIPPED TYPES (require external resources):
// - dbconnection: Requires database connection
// - sftpconnection: Covered by eval_sftp_integration_test.go (testenv fake SFTP server)
// - session: Requires server context
// - dev: Requires dev module setup
```

2. Update the `createTestValues` comment block (~line 179) similarly:

```go
// Types that require external resources are not included:
// - dbconnection (needs DB)
// - sftpconnection (covered by eval_sftp_integration_test.go)
// - sftpfile (covered by eval_sftp_integration_test.go)
// - session (needs server context)
// - dev (needs dev module)
```

3. In `TestSkippedTypes_Documented` (~line 450), update the reason strings for `sftpconnection` and `sftpfile` to indicate they now have integration test coverage but remain skipped here because introspection still requires a live handle:

```go
"sftpconnection": "Covered by eval_sftp_integration_test.go; skipped here because introspection needs a live handle",
"sftpfile":       "Covered by eval_sftp_integration_test.go; skipped here because introspection needs a live handle",
```

Commit: `docs(evaluator): update introspect_validation_test.go to reflect SFTP integration test coverage`

---

### Task 2.5E: Remove stale comment in `connection_cache_test.go`

**Files:** `pkg/parsley/evaluator/connection_cache_test.go` (edit)
**Estimated effort:** Tiny

Lines 300–303 say: "We can't easily test the SFTP cache with a mock connection because the health check (Getwd) requires a real SSH client." This is no longer true — `TestSFTPEval_ConnectionCache` and `TestSFTPEval_CacheHealthCheck` in `eval_sftp_integration_test.go` do exactly this.

Steps:
1. Replace the stale comment in `TestSFTPCacheIntegration` (~line 303):

```go
// Note: Full SFTP cache integration tests (including health-check eviction)
// are in eval_sftp_integration_test.go using a real fake SSH/SFTP server
// from testenv. Here we just verify the cache is initialized properly.
```

Commit: `docs(evaluator): update stale SFTP cache comment to reference integration tests`

---

### Phase 2.5 Validation

- [ ] `go test ./testenv/...` passes
- [ ] `go test ./pkg/parsley/evaluator/...` passes (including new tests)
- [ ] `go test ./...` passes (no regressions)
- [ ] `golangci-lint run` — no new issues

---

## Phase 4: SFTP File Operations

Integration tests for SFTP file handle methods (`mkdir`, `rmdir`, `remove`) and connection methods (`close`).

### Task 4A: Directory creation tests

**Files:** `pkg/parsley/evaluator/eval_sftp_integration_test.go` (edit)
**Estimated effort:** Small

Steps:
1. Add `TestSFTPEval_Mkdir` — create a directory, verify it exists via `env.SFTPReadFile` parent listing or `os.Stat` on `env.SFTPRoot()`
2. Add `TestSFTPEval_MkdirParents` — create nested directories with `{parents: true}`, verify the full path exists

```go
func TestSFTPEval_Mkdir(t *testing.T) {
    env := testenv.Start(t, testenv.WithSFTP())

    src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
conn(@/newdir).mkdir()
`)
    result := testEval(src)
    if isError(result) {
        t.Fatalf("unexpected error: %s", result.(*Error).Message)
    }

    // Verify directory was created
    info, err := os.Stat(filepath.Join(env.SFTPRoot(), "newdir"))
    if err != nil {
        t.Fatalf("directory not created: %v", err)
    }
    if !info.IsDir() {
        t.Error("expected a directory, got a file")
    }
}

func TestSFTPEval_MkdirParents(t *testing.T) {
    env := testenv.Start(t, testenv.WithSFTP())

    src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
conn(@/deep/nested/path).mkdir({parents: true})
`)
    result := testEval(src)
    if isError(result) {
        t.Fatalf("unexpected error: %s", result.(*Error).Message)
    }

    // Verify nested directory was created
    info, err := os.Stat(filepath.Join(env.SFTPRoot(), "deep/nested/path"))
    if err != nil {
        t.Fatalf("nested directory not created: %v", err)
    }
    if !info.IsDir() {
        t.Error("expected a directory, got a file")
    }
}
```

Commit: `test(evaluator): add SFTP mkdir and mkdir with parents tests`

---

### Task 4B: Directory and file removal tests

**Files:** `pkg/parsley/evaluator/eval_sftp_integration_test.go` (edit)
**Estimated effort:** Small

Steps:
1. Add `TestSFTPEval_Rmdir` — create a directory, remove it with `rmdir()`, verify it's gone
2. Add `TestSFTPEval_Remove` — create a file, remove it with `remove()`, verify it's gone
3. Add `TestSFTPEval_RmdirRecursive_NotImplemented` — verify `rmdir({recursive: true})` either works or returns an appropriate error (documents the TODO at `eval_network_io.go:96`)

```go
func TestSFTPEval_Rmdir(t *testing.T) {
    env := testenv.Start(t, testenv.WithSFTP())

    // Create directory first
    dirPath := filepath.Join(env.SFTPRoot(), "toremove")
    if err := os.Mkdir(dirPath, 0o755); err != nil {
        t.Fatalf("setup failed: %v", err)
    }

    src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
conn(@/toremove).rmdir()
`)
    result := testEval(src)
    if isError(result) {
        t.Fatalf("unexpected error: %s", result.(*Error).Message)
    }

    // Verify directory was removed
    if _, err := os.Stat(dirPath); !os.IsNotExist(err) {
        t.Error("directory still exists after rmdir")
    }
}

func TestSFTPEval_Remove(t *testing.T) {
    env := testenv.Start(t, testenv.WithSFTP())
    env.SFTPWriteFile("/deleteme.txt", "temporary content")

    src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
conn(@/deleteme.txt).remove()
`)
    result := testEval(src)
    if isError(result) {
        t.Fatalf("unexpected error: %s", result.(*Error).Message)
    }

    // Verify file was removed
    filePath := filepath.Join(env.SFTPRoot(), "deleteme.txt")
    if _, err := os.Stat(filePath); !os.IsNotExist(err) {
        t.Error("file still exists after remove")
    }
}
```

Commit: `test(evaluator): add SFTP rmdir and remove tests`

---

### Task 4C: Connection close test

**Files:** `pkg/parsley/evaluator/eval_sftp_integration_test.go` (edit)
**Estimated effort:** Small

Steps:
1. Add `TestSFTPEval_Close` — connect, close explicitly, verify subsequent operations fail with appropriate error

```go
func TestSFTPEval_Close(t *testing.T) {
    env := testenv.Start(t, testenv.WithSFTP())
    env.SFTPWriteFile("/testfile.txt", "content")

    // Connect and close
    src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
conn.close()
`)
    result := testEval(src)
    if isError(result) {
        t.Fatalf("close failed: %s", result.(*Error).Message)
    }

    // Note: After close, the connection is marked disconnected but remains
    // in the cache. A subsequent connection attempt should establish a new
    // connection. This test verifies close() doesn't error.
}
```

Commit: `test(evaluator): add SFTP connection close test`

---

### Phase 4 Validation

- [ ] `go test ./pkg/parsley/evaluator/...` passes
- [ ] All SFTP file operation methods have integration test coverage

---

## Phase 5: SFTP Format Coverage

Integration tests for all SFTP read and write formats.

### Task 5A: SFTP read format tests

**Files:** `pkg/parsley/evaluator/eval_sftp_integration_test.go` (edit)
**Estimated effort:** Medium

Steps:
1. Add `TestSFTPEval_ReadLines` — read file with `.lines` format, verify array of strings
2. Add `TestSFTPEval_ReadCSV` — read CSV file with `.csv` format, verify array of dictionaries with header keys
3. Add `TestSFTPEval_ReadBytes` — read file with `.bytes` format, verify array of integers
4. Add `TestSFTPEval_ReadFileAutoDetect` — read `.json` file with `.file` format, verify JSON is auto-parsed

```go
func TestSFTPEval_ReadLines(t *testing.T) {
    env := testenv.Start(t, testenv.WithSFTP())
    env.SFTPWriteFile("/lines.txt", "line1\nline2\nline3")

    src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
let {data, error} <=/= conn(@/lines.txt).lines
data
`)
    result := testEval(src)
    if isError(result) {
        t.Fatalf("unexpected error: %s", result.(*Error).Message)
    }
    arr, ok := result.(*Array)
    if !ok {
        t.Fatalf("expected Array, got %T", result)
    }
    if len(arr.Elements) != 3 {
        t.Errorf("expected 3 lines, got %d", len(arr.Elements))
    }
}

func TestSFTPEval_ReadCSV(t *testing.T) {
    env := testenv.Start(t, testenv.WithSFTP())
    env.SFTPWriteFile("/data.csv", "name,age\nAlice,30\nBob,25")

    src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
let {data, error} <=/= conn(@/data.csv).csv
data[0].name
`)
    result := testEval(src)
    if isError(result) {
        t.Fatalf("unexpected error: %s", result.(*Error).Message)
    }
    str, ok := result.(*String)
    if !ok {
        t.Fatalf("expected String, got %T: %s", result, result.Inspect())
    }
    if str.Value != "Alice" {
        t.Errorf("expected 'Alice', got %q", str.Value)
    }
}

func TestSFTPEval_ReadBytes(t *testing.T) {
    env := testenv.Start(t, testenv.WithSFTP())
    env.SFTPWriteFile("/binary.bin", "ABC") // bytes 65, 66, 67

    src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
let {data, error} <=/= conn(@/binary.bin).bytes
data[0]
`)
    result := testEval(src)
    if isError(result) {
        t.Fatalf("unexpected error: %s", result.(*Error).Message)
    }
    num, ok := result.(*Integer)
    if !ok {
        t.Fatalf("expected Integer, got %T", result)
    }
    if num.Value != 65 { // 'A' = 65
        t.Errorf("expected 65, got %d", num.Value)
    }
}

func TestSFTPEval_ReadFileAutoDetect(t *testing.T) {
    env := testenv.Start(t, testenv.WithSFTP())
    env.SFTPWriteFile("/auto.json", `{"key": "value"}`)

    src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
let {data, error} <=/= conn(@/auto.json).file
data.key
`)
    result := testEval(src)
    if isError(result) {
        t.Fatalf("unexpected error: %s", result.(*Error).Message)
    }
    str, ok := result.(*String)
    if !ok {
        t.Fatalf("expected String, got %T", result)
    }
    if str.Value != "value" {
        t.Errorf("expected 'value', got %q", str.Value)
    }
}
```

Commit: `test(evaluator): add SFTP read format tests (lines, csv, bytes, file)`

---

### Task 5B: SFTP write format tests

**Files:** `pkg/parsley/evaluator/eval_sftp_integration_test.go` (edit)
**Estimated effort:** Medium

Steps:
1. Add `TestSFTPEval_WriteJSON` — write dictionary with `.json` format, read back and verify
2. Add `TestSFTPEval_WriteLines` — write array with `.lines` format, verify file contents
3. Add `TestSFTPEval_WriteBytes` — write byte array with `.bytes` format, verify file contents

```go
func TestSFTPEval_WriteJSON(t *testing.T) {
    env := testenv.Start(t, testenv.WithSFTP())

    src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
{name: "test", count: 42} =/=> conn(@/output.json).json
`)
    result := testEval(src)
    if isError(result) {
        t.Fatalf("unexpected error: %s", result.(*Error).Message)
    }

    got := env.SFTPReadFile("/output.json")
    // JSON encoding adds a newline, check for key content
    if !strings.Contains(got, `"name"`) || !strings.Contains(got, `"test"`) {
        t.Errorf("unexpected JSON content: %s", got)
    }
}

func TestSFTPEval_WriteLines(t *testing.T) {
    env := testenv.Start(t, testenv.WithSFTP())

    src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
["first", "second", "third"] =/=> conn(@/lines.txt).lines
`)
    result := testEval(src)
    if isError(result) {
        t.Fatalf("unexpected error: %s", result.(*Error).Message)
    }

    got := env.SFTPReadFile("/lines.txt")
    expected := "first\nsecond\nthird\n"
    if got != expected {
        t.Errorf("expected %q, got %q", expected, got)
    }
}

func TestSFTPEval_WriteBytes(t *testing.T) {
    env := testenv.Start(t, testenv.WithSFTP())

    src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
[72, 105, 33] =/=> conn(@/bytes.bin).bytes
`)
    result := testEval(src)
    if isError(result) {
        t.Fatalf("unexpected error: %s", result.(*Error).Message)
    }

    got := env.SFTPReadFile("/bytes.bin")
    if got != "Hi!" { // 72='H', 105='i', 33='!'
        t.Errorf("expected 'Hi!', got %q", got)
    }
}
```

Commit: `test(evaluator): add SFTP write format tests (json, lines, bytes)`

---

### Task 5C: SFTP format error tests

**Files:** `pkg/parsley/evaluator/eval_sftp_integration_test.go` (edit)
**Estimated effort:** Small

Steps:
1. Add `TestSFTPEval_WriteCSV_NotImplemented` — verify `.csv` write returns error SFTP-0003
2. Add `TestSFTPEval_UnknownFormat` — verify unknown format returns appropriate error

```go
func TestSFTPEval_WriteCSV_NotImplemented(t *testing.T) {
    env := testenv.Start(t, testenv.WithSFTP())

    src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
[{a: 1}] =/=> conn(@/data.csv).csv
`)
    result := testEval(src)
    // Should return an error about CSV write not being implemented
    if !isError(result) {
        t.Fatal("expected error for CSV write, got success")
    }
    err := result.(*Error)
    if !strings.Contains(err.Message, "CSV") && !strings.Contains(err.Message, "not") {
        t.Errorf("expected CSV not implemented error, got: %s", err.Message)
    }
}

func TestSFTPEval_UnknownReadFormat(t *testing.T) {
    env := testenv.Start(t, testenv.WithSFTP())
    env.SFTPWriteFile("/test.txt", "content")

    src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
let {data, error} <=/= conn(@/test.txt).unknownformat
error
`)
    result := testEval(src)
    // Should capture error about unknown format
    if result == NULL {
        t.Error("expected error for unknown format, got null")
    }
}
```

Commit: `test(evaluator): add SFTP format error tests (csv write, unknown format)`

---

### Phase 5 Validation

- [ ] `go test ./pkg/parsley/evaluator/...` passes
- [ ] All SFTP read formats have integration test coverage
- [ ] All SFTP write formats have integration test coverage
- [ ] Known limitations (CSV write) are tested and documented

---

## Phase 6: Fetch Format Coverage

Integration tests for Fetch operator response formats.

### Task 6A: Fetch format tests

**Files:** `pkg/parsley/evaluator/eval_network_io_test.go` (edit)
**Estimated effort:** Medium

Steps:
1. Add `ServeYAML` helper to testenv (or use `ServeText` with YAML content)
2. Add `TestFetch_YAML` — fetch YAML response, verify parsed structure
3. Add `TestFetch_Lines` — fetch multi-line text, verify array of strings
4. Add `TestFetch_Bytes` — fetch binary content, verify byte array

```go
func TestFetch_YAML(t *testing.T) {
    env := testenv.Start(t, testenv.WithHTTPS())
    testHTTPClient = env.HTTPSClient
    t.Cleanup(func() { testHTTPClient = nil })

    // Serve YAML content with appropriate content-type
    env.ServeText("/config.yaml", "name: basil\nversion: 1")

    src := fmt.Sprintf(`
let {data, error} = <=/= YAML(url("%s/config.yaml"))
data.name
`, env.HTTPSURL)

    result := testEval(src)
    if isError(result) {
        t.Fatalf("unexpected error: %s", result.(*Error).Message)
    }
    str, ok := result.(*String)
    if !ok {
        t.Fatalf("expected String, got %T", result)
    }
    if str.Value != "basil" {
        t.Errorf("expected 'basil', got %q", str.Value)
    }
}

func TestFetch_Lines(t *testing.T) {
    env := testenv.Start(t, testenv.WithHTTPS())
    testHTTPClient = env.HTTPSClient
    t.Cleanup(func() { testHTTPClient = nil })

    env.ServeText("/lines.txt", "line1\nline2\nline3")

    src := fmt.Sprintf(`
let {data, error} = <=/= lines(url("%s/lines.txt"))
len(data)
`, env.HTTPSURL)

    result := testEval(src)
    if isError(result) {
        t.Fatalf("unexpected error: %s", result.(*Error).Message)
    }
    num, ok := result.(*Integer)
    if !ok {
        t.Fatalf("expected Integer, got %T", result)
    }
    if num.Value != 3 {
        t.Errorf("expected 3 lines, got %d", num.Value)
    }
}

func TestFetch_Bytes(t *testing.T) {
    env := testenv.Start(t, testenv.WithHTTPS())
    testHTTPClient = env.HTTPSClient
    t.Cleanup(func() { testHTTPClient = nil })

    env.ServeText("/binary", "ABC")

    src := fmt.Sprintf(`
let {data, error} = <=/= bytes(url("%s/binary"))
data[0]
`, env.HTTPSURL)

    result := testEval(src)
    if isError(result) {
        t.Fatalf("unexpected error: %s", result.(*Error).Message)
    }
    num, ok := result.(*Integer)
    if !ok {
        t.Fatalf("expected Integer, got %T", result)
    }
    if num.Value != 65 { // 'A' = 65
        t.Errorf("expected 65, got %d", num.Value)
    }
}
```

Commit: `test(evaluator): add Fetch format tests (yaml, lines, bytes)`

---

### Phase 6 Validation

- [ ] `go test ./pkg/parsley/evaluator/...` passes
- [ ] All Fetch response formats have integration test coverage

---

## Final Validation Checklist

- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] No hardcoded ports anywhere in `testenv/`
- [ ] No `testenv` import in any non-`_test.go` file
- [ ] `go mod tidy` — no phantom entries in `go.sum`
- [ ] FEAT-132 acceptance criteria checked off in `work/specs/FEAT-132.md`
- [ ] Backlog updated with Phase 7 (embedded Postgres) deferred item

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
| 2026-02-27 | 2.5A: SMTP placeholder test | ✅ Complete | `eval_smtp_placeholder_test.go` — `t.Skip` placeholder for FEAT-084 |
| 2026-02-27 | 2.5B: SFTP dir listing test | ✅ Complete | `TestSFTPEval_ListDirectory` — evaluator-level `.dir` format test |
| 2026-02-27 | 2.5C: SFTP permission denied test | ✅ Complete | `TestSFTPEval_PermissionDenied` — `os.Chmod` + `SFTPRoot()` accessor added to `testenv` |
| 2026-02-27 | 2.5D: introspect_validation comments | ✅ Complete | Updated all three SFTP comment locations in `introspect_validation_test.go` |
| 2026-02-27 | 2.5E: connection_cache comment | ✅ Complete | Updated stale comment in `TestSFTPCacheIntegration` |
| 2026-02-27 | 2.5F: SFTP write test | ✅ Complete | `TestSFTPEval_WriteFile` — `=/=>` operator writes content, verified via `SFTPReadFile` |

---

## Notes From Implementation

- `sftp.WithRootDirectory` does not exist in `pkg/sftp@v1.13.10`. Used `sftp.NewRequestServer` with a custom `osHandler` implementing `sftp.Handlers` (FileGet/FilePut/FileCmd/FileList interfaces) rooted at the temp directory.
- Fixed a pre-existing nil `*Error`-in-interface bug in `evalSFTPRead` (`eval_file_io.go`): `return parseJSON(...)` returned a nil `*Error` as a non-nil `Object` interface, causing a nil pointer dereference when the error was type-asserted. Unwrapped explicitly.
- The `testHTTPClient` injection var approach (package-level var, nil in production) is minimal and self-documenting. No callers changed.
- Fetch `{data, error}` pattern only captures network-level failures in the `error` field; HTTP 4xx/5xx status codes surface via `status`. Tests adjusted to match actual semantics.
- Pre-existing failures `TestDatabaseShutdown` and `TestSiteHandler_RootPath` in `server/` are unrelated to this work (confirmed on `main` before changes).

## Deferred Items

Items to add to `work/BACKLOG.md` after implementation:

- **Phase 7: Embedded Postgres** — `github.com/fergusstrange/embedded-postgres`. Trigger: a Postgres-specific bug is reported or the Postgres driver changes. See `work/reports/INTEGRATION-TESTING-INFRASTRUCTURE.md` for full rationale.
- **`testenv.WithGit()`** — fake Git HTTP server for testing the `GitHandler` auth wrapping in `server/git.go`. Low priority: `go-git-http` is already a dep and the auth layer can be tested with a plain `httptest.Server`. Add if a Git auth bug is reported.
- **Build tag for SFTP tests** — if `eval_sftp_integration_test.go` proves slow enough to disrupt the normal unit test loop, add `//go:build integration` and a `make test-integration` target. Start without it.

## Related

- Spec: `work/specs/FEAT-132.md`
- Report: `work/reports/INTEGRATION-TESTING-INFRASTRUCTURE.md`
- FEAT-084: Email notification API (Phase 1 SMTP is a prerequisite for its integration tests)
- `pkg/parsley/evaluator/eval_network_io.go` — `fetchUrlContentFull`, target of Task 1D refactor
- `pkg/parsley/evaluator/connection_cache_test.go` — stale SFTP comment to be updated in Task 2.5E
- `pkg/parsley/evaluator/introspect_validation_test.go` — stale skipped-types comments to be updated in Task 2.5D