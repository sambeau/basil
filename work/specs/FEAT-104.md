---
id: FEAT-104
title: "Remote Write Operator (=/=> and =/=>>)"
status: draft
priority: high
created: 2026-02-07
author: "@human"
---

# FEAT-104: Remote Write Operator (`=/=>` and `=/=>>`)

## Summary

Implement the `=/=>` (remote write) and `=/=>>` (remote append) operators as the network-write counterparts to the `<=/=` (fetch) operator. This is a **breaking change**: `==>` will no longer accept network targets (HTTP request dicts, SFTP handles) and will be restricted to local file operations. Network writes must use `=/=>`.

These operators were part of the original language design — specified in `plan-sftpSupport.md` and `plan-httpFetchImprovements.prompt.md`, documented in the cheatsheet, and used in SFTP examples and tests — but were never implemented in the lexer, parser, or evaluator. During implementation of HTTP write support, `==>` was made polymorphic as a shortcut. This feature corrects that deviation and enforces the designed operator boundary between local and network I/O.

## Motivation

### Original Design Intent

The original design documents are explicit about this separation. From `plan-sftpSupport.md`:

> **Why Network Operators (Not File Operators)**
>
> SFTP is network I/O, not local file I/O:
> - **Data crosses network** → latency, bandwidth, failures
> - **Clear distinction** from local operations in code
> - **Matches HTTP pattern** → `<=/=` for network reads
> - **Enables append** → `=/=>>` parallel to `==>>` for local files

And `plan-httpFetchImprovements.prompt.md` defines the arrow direction principle:

> | Method | Operator | Accessor | Reason |
> |--------|----------|----------|--------|
> | GET | `<=/=` | `.get` | Server sends data to client |
> | POST | `=/=>` | `.post` | Client sends payload to server |
> | PUT | `=/=>` | `.put` | Client sends payload to server |
> | PATCH | `=/=>` | `.patch` | Client sends payload to server |
> | DELETE | `<=/=` | `.delete` | Request is the message, no payload |

### The Complete I/O Operator Table

The `/` in the middle visually signals "network" — data crosses a boundary:

| Direction | File (local) | Network (remote) |
|-----------|-------------|-------------------|
| **Read**  | `<==`       | `<=/=`            |
| **Write** | `==>`       | `=/=>`            |
| **Append**| `==>>`      | `=/=>>`           |

The read side already enforces this boundary: you cannot use `<==` to fetch a URL, and you cannot use `<=/=` to read a local file. The write side must be equally strict.

### Why Breaking

It is important to know at a glance whether a line of code is a local file operation or a network operation. Network I/O has fundamentally different failure modes (latency, timeouts, auth failures, rate limiting) and security implications (data leaving the machine). Making `==>` silently send data over the network undermines this visibility.

## User Story

As a Parsley developer, I want local file writes (`==>`) and network writes (`=/=>`) to use distinct operators so that I can see at a glance when my code is sending data over the network.

---

## Grammar

### Syntax

```
remote_write_statement  ::= expression "=/=>" expression
remote_append_statement ::= expression "=/=>>" expression
```

Both operators are statement-level (like `==>` and `==>>`) and appear in expression-statement position. The left-hand side is the **value** (data to send). The right-hand side is the **target** (network handle).

### Tokens

| Token | Literal | Name |
|-------|---------|------|
| `REMOTE_WRITE` | `=/=>` | Remote write |
| `REMOTE_APPEND` | `=/=>>` | Remote append |

### Precedence

The remote write operators have the same precedence as `==>` and `==>>`: they are parsed as statement-level constructs in `parseExpressionStatement`, not as infix operators. The left-hand expression is parsed first at `LOWEST` precedence, then the operator is consumed, then the right-hand expression is parsed at `LOWEST` precedence.

### AST Node

```
RemoteWriteStatement {
    Token   lexer.Token    // the =/=> or =/=>> token
    Value   Expression     // left side — data to write
    Target  Expression     // right side — network handle
    Append  bool           // true for =/=>>, false for =/=>
}
```

This mirrors `WriteStatement` but is a distinct node type, just as `FetchStatement` is distinct from `ReadStatement`.

---

## Behavioral Specification

### Valid Targets for `=/=>`

| Target type | Behavior |
|---|---|
| HTTP request dictionary (`isRequestDict` is true) | Sets body to value, defaults method to POST (unless already PUT or PATCH via accessor), executes HTTP request, returns response dictionary |
| SFTP file handle (`*SFTPFileHandle`) | Writes value to remote file using the handle's format, returns `NULL` on success or error object on failure |

### Valid Targets for `=/=>>`

| Target type | Behavior |
|---|---|
| HTTP request dictionary | **Error**: `operator =/=>> (remote append) is not supported for HTTP — HTTP has no append semantic` |
| SFTP file handle | Appends value to remote file using the handle's format (uses `SSH_FXF_APPEND`), returns `NULL` on success or error object on failure |

### Invalid Targets (Both Operators)

| Target type | Error message |
|---|---|
| Local file dictionary (`isFileDict` is true) | `operator =/=> is for network writes; use ==> for local file writes` |
| Any other type (string, integer, array, etc.) | `operator =/=> requires an HTTP request handle or SFTP file handle, got <TYPE>` |

Note: The error messages for `=/=>>` should read `=/=>>` instead of `=/=>`.

### HTTP Method Resolution for `=/=>`

When the target is an HTTP request dictionary:

1. The value (left side) becomes the request `body`
2. If the request dict already has `method` set to `PUT` or `PATCH` (via `.put` or `.patch` accessor), that method is preserved
3. Otherwise, the method defaults to `POST`
4. The request is executed via `fetchUrlContentFull`

This is identical to the existing `evalHTTPWrite` behavior — the same function is reused.

### HTTP Method Mapping (Full Reference)

Per the original design, the arrow direction indicates data flow:

| HTTP Method | Operator | Why |
|---|---|---|
| GET | `<=/=` | Server sends data to client (read) |
| POST | `=/=>` | Client sends payload to server (write) |
| PUT | `=/=>` with `.put` | Client sends payload to server (write) |
| PATCH | `=/=>` with `.patch` | Client sends payload to server (write) |
| DELETE | `<=/=` with `.delete` | Request is the message — no payload (read) |

### Return Values

**HTTP `=/=>`**: Returns a typed response dictionary (same structure as `evalHTTPWrite` currently returns):

```
{
    __type: "response",
    data: <parsed response body>,
    status: <integer>,
    statusText: <string>,
    ok: <boolean>,
    url: <string>,
    headers: <dictionary>,
    error: <string or null>
}
```

On network failure, returns an HTTP error object (`HTTP-0006`).

**SFTP `=/=>`**: Returns `NULL` on success. On failure, returns an error object.

**SFTP `=/=>>`**: Returns `NULL` on success. On failure, returns an error object.

### Error Capture Patterns

The `=/=>` operator is a statement that produces a value. That value can be captured via assignment:

```parsley
// Simple: capture error/response into a variable
error = data =/=> JSON(@https://api.example.com/items)

// Let binding
let response = data =/=> JSON(@https://api.example.com/items)

// Destructured response (works because HTTP =/=> returns a response dict)
let {data, error} = payload =/=> JSON(@https://api.example.com/items)

// SFTP error capture
let writeErr = config =/=> conn(@/config/app.json).json
```

### Breaking Change to `==>` and `==>>`

The `evalWriteStatement` function must be modified to reject network targets:

| Target type | Current behavior | New behavior |
|---|---|---|
| Local file dictionary | Write to file | Write to file (**unchanged**) |
| HTTP request dictionary | Execute HTTP request via `evalHTTPWrite` | **Error**: `operator ==> is for local file writes; use =/=> for network writes` |
| SFTP file handle | Write to remote file via `evalSFTPWrite` | **Error**: `operator ==> is for local file writes; use =/=> for network writes` |
| Stdin/stdout/stderr (`@-`, `@stdin`, `@stdout`, `@stderr`) | Write to stream | Write to stream (**unchanged**) |
| Any other type | Error | Error (**unchanged**) |

The same applies to `==>>` (append):

| Target type | Current behavior | New behavior |
|---|---|---|
| Local file dictionary | Append to file | Append to file (**unchanged**) |
| SFTP file handle | Append to remote file | **Error**: `operator ==>> is for local file appends; use =/=>> for remote appends` |
| HTTP request dictionary | N/A (not currently handled) | **Error**: `operator ==>> is for local file appends; use =/=>> for remote appends` |

---

## Comprehensive Examples

### HTTP POST (default method)

```parsley
let payload = {name: "Alice", age: 30}
let response = payload =/=> JSON(@https://api.example.com/users)
// Sends POST with JSON body {"name": "Alice", "age": 30}
// response is a response dictionary with .data, .status, .ok, etc.
```

### HTTP PUT (explicit method via accessor)

```parsley
let updated = {name: "Alice", age: 31}
let response = updated =/=> JSON(@https://api.example.com/users/1).put
// Sends PUT with JSON body
```

### HTTP PATCH (explicit method via accessor)

```parsley
let patch = {age: 31}
let response = patch =/=> JSON(@https://api.example.com/users/1).patch
// Sends PATCH with JSON body
```

### HTTP POST with options dictionary

```parsley
let response = payload =/=> JSON(@https://api.example.com/users, {
    headers: {Authorization: "Bearer token123"},
    timeout: 5000
})
// POST with custom headers and timeout
```

### HTTP DELETE (uses fetch — no payload)

```parsley
let result <=/= JSON(@https://api.example.com/users/1).delete
// DELETE has no body — it uses the fetch operator, not the write operator
```

### Error capture — simple

```parsley
let error = data =/=> JSON(@https://api.example.com/items)
if (error) {
    log("Write failed:", error)
}
```

### Error capture — destructured response

```parsley
let {data, error, status} = payload =/=> JSON(@https://api.example.com/items)
if (error) {
    log("HTTP error:", status, error)
} else {
    log("Created:", data.id)
}
```

### SFTP write

```parsley
let conn = SFTP(@sftp://user@host, {keyFile: @~/.ssh/id_rsa})

// Write JSON file to remote server
let writeErr = config =/=> conn(@/config/app.json).json
if (writeErr) {
    log("Failed to write:", writeErr)
}

// Write text file
"Hello, SFTP World!" =/=> conn(@/messages/hello.txt).text

// Write lines
["line1", "line2", "line3"] =/=> conn(@/logs/startup.log).lines

// Write bytes
[137, 80, 78, 71, 13, 10, 26, 10] =/=> conn(@/images/test.png).bytes
```

### SFTP append

```parsley
// Append text to remote log file
"New log entry\n" =/=>> conn(@/logs/app.log).text

// Append lines
"Additional line" =/=>> conn(@/data/list.txt).lines
```

### What breaks — `==>` rejects network targets

```parsley
// OLD (worked before, now errors):
payload ==> JSON(@https://api.example.com/users)
// ERROR: operator ==> is for local file writes; use =/=> for network writes

// OLD (worked before, now errors):
config ==> conn(@/config/app.json).json
// ERROR: operator ==> is for local file writes; use =/=> for network writes

// FIX — use =/=> instead:
payload =/=> JSON(@https://api.example.com/users)
config =/=> conn(@/config/app.json).json
```

### Local file operations — unchanged

```parsley
// These continue to work exactly as before:
data ==> JSON(@./output.json)
"Hello" ==> text(@./message.txt)
logEntry ==>> text(@./log.txt)
data ==> JSON(@-)           // stdout
```

---

## Acceptance Criteria

### Functional — New Operators
- [ ] Lexer tokenises `=/=>` as `REMOTE_WRITE`
- [ ] Lexer tokenises `=/=>>` as `REMOTE_APPEND`
- [ ] Parser produces `RemoteWriteStatement` AST node with `Append: false` for `=/=>`
- [ ] Parser produces `RemoteWriteStatement` AST node with `Append: true` for `=/=>>`
- [ ] Parser captures left-hand expression as `Value` and right-hand expression as `Target`
- [ ] `=/=>` with HTTP request dict target: sets body, defaults method to POST, executes request, returns response dict
- [ ] `=/=>` with HTTP request dict + `.put` accessor: sends PUT
- [ ] `=/=>` with HTTP request dict + `.patch` accessor: sends PATCH
- [ ] `=/=>` with SFTP file handle: writes value to remote file, returns NULL on success
- [ ] `=/=>>` with SFTP file handle: appends value to remote file, returns NULL on success
- [ ] `=/=>>` with HTTP request dict: produces error (no HTTP append)
- [ ] `=/=>` with local file handle: produces error suggesting `==>`
- [ ] `=/=>>` with local file handle: produces error suggesting `==>>`
- [ ] `=/=>` with non-handle type (string, int, etc.): produces type error
- [ ] Error capture works: `error = value =/=> target`
- [ ] Let binding works: `let response = value =/=> target`
- [ ] Destructuring works: `let {data, error} = value =/=> target`

### Functional — Breaking Change to `==>`
- [ ] `==>` with HTTP request dict target: produces error `operator ==> is for local file writes; use =/=> for network writes`
- [ ] `==>` with SFTP file handle target: produces error `operator ==> is for local file writes; use =/=> for network writes`
- [ ] `==>` with local file handle: works as before (unchanged)
- [ ] `==>` with stdin/stdout/stderr: works as before (unchanged)
- [ ] `==>>` with SFTP file handle target: produces error `operator ==>> is for local file appends; use =/=>> for remote appends`
- [ ] `==>>` with local file handle: works as before (unchanged)

---

## Test Plan

Tests are organised by layer. Each test case includes the input, expected outcome, and what it verifies the spec.

### Layer 1: Lexer Tests

**File:** `pkg/parsley/lexer/lexer_test.go` (or a new `lexer_remote_write_test.go` if preferred)

These tests verify that the lexer produces the correct token types and literals.

#### L1.1 — Tokenise `=/=>`

| # | Input | Expected Token Type | Expected Literal | Notes |
|---|---|---|---|---|
| L1.1a | `x =/=> y` | `REMOTE_WRITE` | `=/=>` | Basic remote write |
| L1.1b | `{a: 1} =/=> JSON(url)` | `REMOTE_WRITE` | `=/=>` | Dict value, format factory target |

#### L1.2 — Tokenise `=/=>>`

| # | Input | Expected Token Type | Expected Literal | Notes |
|---|---|---|---|---|
| L1.2a | `x =/=>> y` | `REMOTE_APPEND` | `=/=>>` | Basic remote append |
| L1.2b | `"text" =/=>> conn(@/log).text` | `REMOTE_APPEND` | `=/=>>` | String value, SFTP target |

#### L1.3 — No Ambiguity With Existing Tokens

| # | Input | Tokens Produced | Notes |
|---|---|---|---|
| L1.3a | `x = y` | `IDENT`, `ASSIGN`, `IDENT` | Plain assignment unchanged |
| L1.3b | `x == y` | `IDENT`, `EQ`, `IDENT` | Equality unchanged |
| L1.3c | `x ==> y` | `IDENT`, `WRITE_TO`, `IDENT` | File write unchanged |
| L1.3d | `x ==>> y` | `IDENT`, `APPEND_TO`, `IDENT` | File append unchanged |
| L1.3e | `x => y` | `IDENT`, `ARROW`, `IDENT` | Arrow unchanged |
| L1.3f | `x = /y/` | `IDENT`, `ASSIGN`, `REGEX` | Assignment + regex — not `=/=>` |
| L1.3g | `x =/=> y; a ==> b` | `REMOTE_WRITE` then `WRITE_TO` | Both operators in same input |

#### L1.4 — Token Position Tracking

| # | Input | Expected Line/Column | Notes |
|---|---|---|---|
| L1.4a | `x =/=> y` | Token at col 3 | Column of the `=` that starts `=/=>` |

### Layer 2: Parser Tests

**File:** `pkg/parsley/tests/` — new test file or added to existing parser tests.

These tests verify AST structure. They parse input and check that the resulting AST node has the correct type and fields.

#### P2.1 — Basic `=/=>` Parsing

| # | Input | Expected AST | Notes |
|---|---|---|---|
| P2.1a | `x =/=> y` | `RemoteWriteStatement{Value: Ident("x"), Target: Ident("y"), Append: false}` | Simplest form |
| P2.1b | `{a: 1} =/=> target` | `RemoteWriteStatement{Value: DictLiteral, Target: Ident("target"), Append: false}` | Dict as value |
| P2.1c | `"hello" =/=> target` | `RemoteWriteStatement{Value: StringLiteral("hello"), Target: Ident("target"), Append: false}` | String as value |
| P2.1d | `[1, 2] =/=> target` | `RemoteWriteStatement{Value: ArrayLiteral, Target: Ident("target"), Append: false}` | Array as value |
| P2.1e | `x =/=> JSON(@https://example.com)` | `RemoteWriteStatement{Value: Ident("x"), Target: CallExpression}` | Format factory call as target |
| P2.1f | `x =/=> JSON(@https://example.com).put` | `RemoteWriteStatement{Target: DotExpression}` | Method accessor on target |

#### P2.2 — Basic `=/=>>` Parsing

| # | Input | Expected AST | Notes |
|---|---|---|---|
| P2.2a | `x =/=>> y` | `RemoteWriteStatement{Value: Ident("x"), Target: Ident("y"), Append: true}` | Simplest append form |
| P2.2b | `"line\n" =/=>> target` | `RemoteWriteStatement{Append: true}` | String append |

#### P2.3 — `String()` Round-Trip

| # | Input | Expected `String()` output | Notes |
|---|---|---|---|
| P2.3a | `x =/=> y` | `x =/=> y;` | Verify AST stringification includes operator |
| P2.3b | `x =/=>> y` | `x =/=>> y;` | Verify append variant |

#### P2.4 — Semicolon Handling

| # | Input | Parses without error | Notes |
|---|---|---|---|
| P2.4a | `x =/=> y;` | Yes | Explicit semicolon |
| P2.4b | `x =/=> y` | Yes | No semicolon (implicit) |

### Layer 3: Evaluator Tests — HTTP Remote Write

**File:** `pkg/parsley/tests/` — new `remote_write_test.go`.

These tests use `httptest.NewServer` mock servers (same pattern as `fetch_test.go`).

#### E3.1 — HTTP POST (Default Method)

**Setup:** Echo server that returns `{"method": "<METHOD>", "body": "<BODY>"}`.

| # | Input | Expected | Notes |
|---|---|---|---|
| E3.1a | `{name: "Alice"} =/=> JSON(url("<server>"))` | Response dict with method=POST, body contains `name` | Default method is POST |
| E3.1b | `"hello" =/=> text(url("<server>"))` | Response with method=POST | Text format |
| E3.1c | `[1, 2, 3] =/=> JSON(url("<server>"))` | Response with method=POST, body is array | Array body |

#### E3.2 — HTTP PUT (Via Accessor)

| # | Input | Expected | Notes |
|---|---|---|---|
| E3.2a | `{name: "Alice"} =/=> JSON(url("<server>")).put` | Response dict with method=PUT | `.put` accessor |

#### E3.3 — HTTP PATCH (Via Accessor)

| # | Input | Expected | Notes |
|---|---|---|---|
| E3.3a | `{age: 31} =/=> JSON(url("<server>")).patch` | Response dict with method=PATCH | `.patch` accessor |

#### E3.4 — HTTP POST Explicit (Via Accessor)

| # | Input | Expected | Notes |
|---|---|---|---|
| E3.4a | `{name: "Alice"} =/=> JSON(url("<server>")).post` | Response dict with method=POST | `.post` accessor (redundant but valid) |

#### E3.5 — HTTP with Custom Headers

| # | Input | Expected | Notes |
|---|---|---|---|
| E3.5a | `{data: 1} =/=> JSON(url("<server>"), {headers: {Authorization: "Bearer token"}})` | Server receives Authorization header | Custom headers passed through |

#### E3.6 — HTTP Error Handling

**Setup:** Server returning 500, server returning 404, unreachable server.

| # | Input | Expected | Notes |
|---|---|---|---|
| E3.6a | `{data: 1} =/=> JSON(url("<500-server>"))` | Error object (HTTP-0006) or response with `ok: false` | Server error |
| E3.6b | `{data: 1} =/=> JSON(url("http://localhost:1"))` | Error object | Connection refused |

#### E3.7 — Error Capture Patterns (HTTP)

| # | Input | Expected | Notes |
|---|---|---|---|
| E3.7a | `let response = {a: 1} =/=> JSON(url("<server>")); response.status` | 200 (integer) | Let binding captures response |
| E3.7b | `error = {a: 1} =/=> JSON(url("<server>")); error` | Response dict | Assignment captures response |
| E3.7c | `let {data, error} = {a: 1} =/=> JSON(url("<server>")); error` | null | Destructured, no error |
| E3.7d | `let {data, error} = {a: 1} =/=> JSON(url("<bad-server>")); error != null` | true | Destructured, has error |

### Layer 4: Evaluator Tests — Target Rejection by `=/=>`

#### E4.1 — Reject Local File Handles

| # | Input | Expected Error Contains | Notes |
|---|---|---|---|
| E4.1a | `"data" =/=> JSON(@./local.json)` | `for network writes` and `use ==>` | File path target |
| E4.1b | `"data" =/=> text(@./local.txt)` | `for network writes` and `use ==>` | Text file target |

#### E4.2 — Reject Non-Handle Types

| # | Input | Expected Error Contains | Notes |
|---|---|---|---|
| E4.2a | `"data" =/=> 123` | `requires an HTTP request handle or SFTP file handle` | Integer target |
| E4.2b | `"data" =/=> "string"` | `requires an HTTP request handle or SFTP file handle` | String target |
| E4.2c | `"data" =/=> [1, 2]` | `requires an HTTP request handle or SFTP file handle` | Array target |
| E4.2d | `"data" =/=> {a: 1}` | `requires an HTTP request handle or SFTP file handle` | Plain dict (not request dict) |

### Layer 5: Evaluator Tests — `=/=>>` (Remote Append)

#### E5.1 — Reject HTTP Targets

| # | Input | Expected Error Contains | Notes |
|---|---|---|---|
| E5.1a | `"data" =/=>> JSON(url("<server>"))` | `not supported for HTTP` and `no append semantic` | HTTP has no append |
| E5.1b | `"data" =/=>> JSON(url("<server>")).post` | `not supported for HTTP` | Even with explicit method |

#### E5.2 — Reject Local File Handles

| # | Input | Expected Error Contains | Notes |
|---|---|---|---|
| E5.2a | `"data" =/=>> text(@./local.txt)` | `for network writes` or `for remote appends` and `use ==>>` | Suggest `==>>` |

#### E5.3 — SFTP Append (Skipped — requires SFTP server)

| # | Input | Notes |
|---|---|---|
| E5.3a | `"line\n" =/=>> conn(@/log.txt).text` | Verify append flag passed to `evalSFTPWrite` |

### Layer 6: Evaluator Tests — Breaking Change to `==>`

These tests verify that `==>` and `==>>` now reject network targets.

#### E6.1 — `==>` Rejects HTTP Request Dicts

**Setup:** Mock HTTP server (same as Layer 3).

| # | Input | Expected Error Contains | Notes |
|---|---|---|---|
| E6.1a | `{a: 1} ==> JSON(url("<server>"))` | `operator ==> is for local file writes` and `use =/=>` | Was previously valid |
| E6.1b | `{a: 1} ==> JSON(url("<server>")).put` | `operator ==> is for local file writes` and `use =/=>` | PUT variant |
| E6.1c | `"data" ==> text(url("<server>"))` | `operator ==> is for local file writes` and `use =/=>` | Text format |

#### E6.2 — `==>` Rejects SFTP Handles (Skipped)

| # | Input | Expected Error Contains | Notes |
|---|---|---|---|
| E6.2a | `config ==> conn(@/config.json).json` | `operator ==> is for local file writes` and `use =/=>` | SFTP write rejected |

#### E6.3 — `==>>` Rejects SFTP Handles (Skipped)

| # | Input | Expected Error Contains | Notes |
|---|---|---|---|
| E6.3a | `"log\n" ==>> conn(@/log.txt).text` | `for local file appends` and `use =/=>>` | SFTP append rejected |

#### E6.4 — `==>` Still Works for Local Files

These are regression tests — they should already pass. Run them to confirm no collateral damage.

| # | Input | Expected | Notes |
|---|---|---|---|
| E6.4a | `"hello" ==> text(@<tmpfile>)` | File contains "hello" | Local text write |
| E6.4b | `{a: 1} ==> JSON(@<tmpfile>)` | File contains JSON | Local JSON write |
| E6.4c | `"more" ==>> text(@<tmpfile>)` | File appended | Local text append |
| E6.4d | `data ==> JSON(@-)` | Writes to stdout | Stdin/stdout unchanged |

### Layer 7: SFTP Tests (Update Existing Skipped Tests)

**File:** `pkg/parsley/tests/sftp_test.go`

The existing SFTP tests in this file already use `=/=>` and `=/=>>` syntax but are skipped because the operator didn't parse. After implementation:

1. **Un-skip** `TestSFTPWriteOperatorSyntax` — these should now parse without error (they will still fail at runtime due to no SFTP server, but they should get past the parser)
2. **Un-skip** `TestSFTPAppendOperatorSyntax` — same
3. **Update** `TestSFTPErrorCapturePattern` write test — verify it parses
4. **Update** `TestSFTPFormatEncoding` — verify it parses

The tests should be restructured to distinguish:
- **Parse tests** (un-skipped): verify the syntax is accepted by the parser
- **Integration tests** (remain skipped): verify actual SFTP server interaction

### Layer 8: Example File Verification

**File:** `examples/parsley/sftp_demo.pars`

This file already uses `=/=>` and `=/=>>` syntax throughout. After implementation, verify it parses without error:

```bash
pars --check examples/parsley/sftp_demo.pars
```

(If `--check` doesn't exist, at minimum verify the lexer/parser don't error.)

### Layer 9: Integration / Smoke Tests

| # | Test | Notes |
|---|---|---|
| I9.1 | Run `go test ./...` — all existing tests pass | No regressions |
| I9.2 | Existing `write_operator_test.go` tests pass unchanged | Local file writes unaffected |
| I9.3 | Existing `fetch_test.go` tests pass unchanged | Fetch operator unaffected |
| I9.4 | Existing `stdin_test.go` tests pass unchanged | Stdin/stdout unaffected |

---

## Documentation Plan

### D1. `docs/parsley/manual/features/network.md` — Primary Changes

This is the main manual page for HTTP & Networking. It currently documents `==>` as the HTTP write operator. This needs significant revision.

#### D1.1 — Rename "The Write Operator" Section

**Current** (section "### The Write Operator (`==>`)"): Documents `==>` for HTTP writes.

**Replace with** new section "### The Remote Write Operator (`=/=>`)":

```markdown
### The Remote Write Operator (`=/=>`)

Use `=/=>` to send data to a network target. The left side is the data to send (becomes the request body). The right side is a URL handle. The method defaults to POST unless the handle specifies PUT or PATCH:

​```parsley
// POST (default)
let response = {name: "Alice"} =/=> JSON(@https://api.example.com/users)

// PUT (explicit via accessor)
let response = {name: "Alice Smith"} =/=> JSON(@https://api.example.com/users/123).put

// PATCH
let response = {age: 31} =/=> JSON(@https://api.example.com/users/123).patch
​```

The `=/=>` operator only accepts network targets (HTTP request handles or SFTP file handles). For local file writes, use `==>`.
```

#### D1.2 — Add Error Capture Examples for `=/=>`

```markdown
### Error Handling for Remote Writes

​```parsley
// Capture the full response
let response = payload =/=> JSON(@https://api.example.com/items)
if (!response.ok) {
    log("Failed:", response.status, response.error)
}

// Destructured capture
let {data, error} = payload =/=> JSON(@https://api.example.com/items)
if (error) {
    log("Error:", error)
}
​```
```

#### D1.3 — Update Method Accessors Section

Add a note that method accessors combine with `=/=>` for write methods and `<=/=` for read methods:

```markdown
### Method Accessors

| Accessor | Method | Use with |
|---|---|---|
| `.get` | GET | `<=/=` (default, rarely needed) |
| `.post` | POST | `=/=>` (default, rarely needed) |
| `.put` | PUT | `=/=>` |
| `.patch` | PATCH | `=/=>` |
| `.delete` | DELETE | `<=/=` |
```

#### D1.4 — Update SFTP Section

The SFTP section currently shows `==>` for writes. Replace with:

```markdown
### Reading and Writing via SFTP

Use the network I/O operators with SFTP connections:

​```parsley
// Read a remote JSON file (network read)
let config <=/= conn(@/etc/app/config.json).json

// Write to a remote file (network write)
"new content" =/=> conn(@/var/data/output.txt).text

// Append to a remote log (network append)
"log entry\n" =/=>> conn(@/var/log/app.log).text
​```
```

#### D1.5 — Update "Key Differences" Section

Add a bullet point:

```markdown
- **Local vs network writes** — `==>` writes to local files, `=/=>` writes to network targets. The `/` in the operator visually signals that data crosses a network boundary. This matches the read side: `<==` reads files, `<=/=` fetches URLs.
```

### D2. `docs/parsley/CHEATSHEET.md`

#### D2.1 — Update File I/O Section (### HTTP Requests)

**Current:**

```markdown
### HTTP Requests
// Simple GET
let users <=/= JSON(@https://api.example.com/users)

// POST with body
let response <=/= JSON(@https://api.example.com/users, {
    method: "POST",
    body: {name: "Alice"},
    headers: {"Authorization": "Bearer token"}
})
```

**Add** after the existing content:

```markdown
// POST with remote write operator
let response = {name: "Alice"} =/=> JSON(@https://api.example.com/users)

// PUT
let response = data =/=> JSON(@https://api.example.com/users/1).put

// PATCH
let response = patch =/=> JSON(@https://api.example.com/users/1).patch
```

#### D2.2 — Add to Operator Summary / Gotchas

Under the existing "Major Gotchas" or add a new gotcha:

```markdown
### N. Local vs Network Write Operators
​```parsley
// ❌ WRONG — ==> is for local files only
data ==> JSON(@https://api.example.com/users)
// ERROR: operator ==> is for local file writes; use =/=> for network writes

// ✅ CORRECT — use =/=> for network writes
data =/=> JSON(@https://api.example.com/users)

// ✅ CORRECT — ==> for local files
data ==> JSON(@./output.json)
​```
```

### D3. `docs/parsley/reference.md`

#### D3.1 — Operator Precedence Table (Section 2.11)

Add `=/=>` and `=/=>>` to the I/O operators row alongside `<==`, `==>`, `==>>`, `<=/=`.

#### D3.2 — File Operations Section (Section 6.12)

Add network write examples:

```markdown
// Network writes (HTTP)
{name: "Alice"} =/=> JSON(@https://api.example.com/users)            // POST
{name: "Alice"} =/=> JSON(@https://api.example.com/users/1).put      // PUT

// Network writes (SFTP)
config =/=> conn(@/config/app.json).json                              // Write
"entry\n" =/=>> conn(@/logs/app.log).text                             // Append
```

### D4. `docs/basil/reference.md`

#### D4.1 — Section 3: File I/O Operations

Add new subsection after 3.4 (Fetch URL):

```markdown
### 3.5 Remote Write (`=/=>`)

Sends data to a network target (HTTP endpoint or SFTP server).

**Grammar:** `expression =/=> expression`

​```parsley
{name: "Alice"} =/=> JSON(@https://api.example.com/users)
​```

**Arguments:**
- Left: Value to send (becomes request body for HTTP, file content for SFTP)
- Right: Network handle (HTTP request handle or SFTP file handle)

**HTTP behavior:** Defaults method to POST. Use `.put` or `.patch` accessors for other methods.
**Returns:** Response dictionary (HTTP) or NULL (SFTP success) or error

**Errors:** `HTTP-0006` (request failed)

### 3.6 Remote Append (`=/=>>`)

Appends data to a remote file via SFTP.

**Grammar:** `expression =/=>> expression`

​```parsley
"log entry\n" =/=>> conn(@/var/log/app.log).text
​```

Not supported for HTTP targets (HTTP has no append semantic).
```

#### D4.2 — Update Feature Availability Table (Appendix A)

Add row:

```markdown
| Remote Write (`=/=>`, `=/=>>`) | ✓ | ✓ | Requires `--allow-net` in pars |
```

#### D4.3 — Update SFTP Section (Section 1.4)

Replace `==>` with `=/=>` in the SFTP file operations example:

```markdown
// Write to remote file
"data" =/=> text(sftp[@./remote/file.txt])
```

### D5. `contrib/highlightjs/`

Add `=/=>` and `=/=>>` to the operator pattern list alongside `<=/=`, `==>`, `==>>`, `<==`.

### D6. CHANGELOG

Add a breaking change entry:

```markdown
### Breaking Changes

- **`==>` and `==>>` no longer accept network targets.** HTTP request dictionaries and SFTP file handles must now use the dedicated network write operators `=/=>` and `=/=>>`. Using `==>` with a network target produces a clear error message with the fix. This enforces a visible distinction between local file I/O and network I/O, matching the existing read-side separation (`<==` vs `<=/=`).

### Added

- **Remote write operator `=/=>`** — Sends data to HTTP endpoints or SFTP servers. Defaults to POST for HTTP; use `.put` or `.patch` accessors for other methods. Counterpart to the fetch operator `<=/=`.
- **Remote append operator `=/=>>`** — Appends data to remote files via SFTP. Not supported for HTTP (HTTP has no append semantic).
```

### D7. `docs/parsley/manual/builtins/urls.md`

The "See Also" section references `<=/=` fetch. Add `=/=>` remote write:

```markdown
- [Operators](../fundamentals/operators.md) — `+` URL joining, `<==` read, `<=/=` fetch, `=/=>` remote write
```

---

## Design Decisions

- **Breaking change — `==>` rejects network targets**: This enforces the original design intent from `plan-sftpSupport.md` and `plan-httpFetchImprovements.prompt.md`. The read side already enforces this boundary (`<==` vs `<=/=`); the write side must be equally strict. Network I/O should always be visually distinct from local file I/O.

- **Separate AST node (`RemoteWriteStatement`)**: Rather than overloading `WriteStatement` with a flag, a distinct node keeps the AST explicit and mirrors how `FetchStatement` is separate from `ReadStatement`.

- **`=/=>>` rejects HTTP targets**: HTTP has no meaningful "append" semantic. Rather than silently treating it as POST, produce a clear error. SFTP append (`=/=>>`) works as designed (uses `SSH_FXF_APPEND`).

- **Error capture uses assignment, not destructuring on the operator**: The pattern is `error = value =/=> target`. The `=/=>` expression returns a result (response dict or error), which gets assigned. This matches how `==>` works today.

- **DELETE uses `<=/=` not `=/=>`**: Per the original design, DELETE has no payload — the request itself is the message. So `<=/= target.delete` is correct, not `=/=>`.

- **Error messages include the fix**: Every error message from this change tells the user exactly what to do instead (e.g., "use `=/=>` for network writes"). The migration should be frictionless.

---

<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components

| File | Change |
|---|---|
| `pkg/parsley/lexer/lexer.go` | Add `REMOTE_WRITE` and `REMOTE_APPEND` token types; tokenise in `NextToken()` `=` branch |
| `pkg/parsley/ast/ast.go` | Add `RemoteWriteStatement` AST node |
| `pkg/parsley/parser/parser.go` | Parse `=/=>` and `=/=>>` in `parseExpressionStatement`; add to `tokenTypeToReadableName` |
| `pkg/parsley/evaluator/eval_network_io.go` | Add `evalRemoteWriteStatement` function |
| `pkg/parsley/evaluator/eval_file_io.go` | Modify `evalWriteStatement` to reject request dicts and SFTP handles |
| `pkg/parsley/evaluator/evaluator.go` | Add `case *ast.RemoteWriteStatement` in `Eval()` |
| `pkg/parsley/tests/remote_write_test.go` | New test file (Layers 3–6 from test plan) |
| `pkg/parsley/tests/sftp_test.go` | Un-skip parse tests, add `==>` rejection tests |
| `pkg/parsley/lexer/lexer_test.go` | Add token tests (Layer 1 from test plan) |
| `contrib/highlightjs/` | Add operators to pattern list |

### Implementation Reference

**Lexer** — The `=/=>` token starts with `=`, so it's handled in the `=` branch of `NextToken()`. Lookahead sequence: `=`, `/`, `=`, `>` for `REMOTE_WRITE`; `=`, `/`, `=`, `>`, `>` for `REMOTE_APPEND`. Check the longer token first to avoid partial matches.

```go
// In the '=' case of NextToken(), before existing == / ==> / ==>> checks:
// =/=> (remote write) and =/=>> (remote append)
if l.peekChar() == '/' && l.peekCharN(2) == '=' && l.peekCharN(3) == '>' {
    line := l.line
    col := l.column
    l.readChar() // consume '/'
    l.readChar() // consume '='
    l.readChar() // consume '>'
    if l.peekChar() == '>' {
        l.readChar() // consume second '>'
        tok = Token{Type: REMOTE_APPEND, Literal: "=/=>>", Line: line, Column: col}
    } else {
        tok = Token{Type: REMOTE_WRITE, Literal: "=/=>", Line: line, Column: col}
    }
}
```

**AST node:**

```go
type RemoteWriteStatement struct {
    Token  lexer.Token // the =/=> or =/=>> token
    Value  Expression  // the data to write (left side)
    Target Expression  // the network handle (right side)
    Append bool        // true for =/=>> (append), false for =/=> (write)
}

func (rw *RemoteWriteStatement) statementNode()       {}
func (rw *RemoteWriteStatement) TokenLiteral() string { return rw.Token.Literal }
func (rw *RemoteWriteStatement) String() string {
    var out bytes.Buffer
    out.WriteString(rw.Value.String())
    if rw.Append {
        out.WriteString(" =/=>> ")
    } else {
        out.WriteString(" =/=> ")
    }
    out.WriteString(rw.Target.String())
    out.WriteString(";")
    return out.String()
}
```

**Parser** — In `parseExpressionStatement`, after the `==>` / `==>>` block:

```go
// Check for remote write operators =/=> or =/=>>
if p.peekTokenIs(lexer.REMOTE_WRITE) || p.peekTokenIs(lexer.REMOTE_APPEND) {
    p.nextToken() // consume =/=> or =/=>>
    stmt := &ast.RemoteWriteStatement{
        Token:  p.curToken,
        Value:  expr,
        Append: p.curToken.Type == lexer.REMOTE_APPEND,
    }
    p.nextToken() // move to target expression
    stmt.Target = p.parseExpression(LOWEST)
    if p.peekTokenIs(lexer.SEMICOLON) {
        p.nextToken()
    }
    return stmt
}
```

**Evaluator — new handler** (in `eval_network_io.go`):

```go
func evalRemoteWriteStatement(node *ast.RemoteWriteStatement, env *Environment) Object {
    value := Eval(node.Value, env)
    if isError(value) {
        return value
    }

    target := Eval(node.Target, env)
    if isError(target) {
        return target
    }

    op := "=/=>"
    if node.Append {
        op = "=/=>>"
    }

    // SFTP file handle
    if sftpHandle, ok := target.(*SFTPFileHandle); ok {
        err := evalSFTPWrite(sftpHandle, value, node.Append, env)
        if err != nil {
            return err
        }
        return NULL
    }

    // HTTP request dictionary
    if reqDict, ok := target.(*Dictionary); ok && isRequestDict(reqDict) {
        if node.Append {
            return newError("operator =/=>> (remote append) is not supported for HTTP — HTTP has no append semantic")
        }
        return evalHTTPWrite(reqDict, value, env)
    }

    // Reject local file handles with helpful message
    if fileDict, ok := target.(*Dictionary); ok && isFileDict(fileDict) {
        if node.Append {
            return newError("operator =/=>> is for remote appends; use ==>> for local file appends")
        }
        return newError("operator =/=> is for network writes; use ==> for local file writes")
    }

    return newError("operator %s requires an HTTP request handle or SFTP file handle, got %s", op, strings.ToLower(string(target.Type())))
}
```

**Evaluator — modify `evalWriteStatement`** (in `eval_file_io.go`):

```go
// Add BEFORE the existing isRequestDict / SFTPFileHandle checks:

// Reject HTTP request dictionaries — must use =/=>
if reqDict, ok := target.(*Dictionary); ok && isRequestDict(reqDict) {
    return newError("operator ==> is for local file writes; use =/=> for network writes")
}

// Reject SFTP file handles — must use =/=> or =/=>>
if _, ok := target.(*SFTPFileHandle); ok {
    if node.Append {
        return newError("operator ==>> is for local file appends; use =/=>> for remote appends")
    }
    return newError("operator ==> is for local file writes; use =/=> for network writes")
}
```

### Dependencies

None — all infrastructure (`evalHTTPWrite`, `evalSFTPWrite`, `fetchUrlContentFull`, `isRequestDict`, `isFileDict`) already exists and is reused without modification.

### Edge Cases & Constraints

1. **Token ambiguity with `=` operator** — The `=` branch in the lexer already handles `==`, `==>`, `==>>`, `=>`. The new `=/=>` starts with `=` then `/`, which is currently not a valid token sequence, so there is no ambiguity. However, the check for `=/` must come before the fallback `=` assignment, and the ordering relative to `==>` matters. Since `==>` starts with `==` (two `=` chars) and `=/=>` starts with `=/` (equals then slash), they don't conflict.

2. **`= /` as assignment + division** — The sequence `x = /regex/` (assign a regex literal) starts with `= /` which looks like the start of `=/=>`. The lexer must check `peekChar() == '/'` AND `peekCharN(2) == '='` AND `peekCharN(3) == '>'` — a regex literal would have different characters at positions 2+, so there's no ambiguity.

3. **Error capture with assignment** — The pattern `error = value =/=> target` works because the parser first parses `error` as an expression, sees `=` (assignment), then parses the RHS which includes `value =/=> target`. The `=/=>` produces a `RemoteWriteStatement` that returns a value. This is the same pattern used by `==>` today.

4. **`=/=>>` on HTTP** — HTTP has no append. Error with: `operator =/=>> (remote append) is not supported for HTTP — HTTP has no append semantic`.

5. **Existing code using `==>` for HTTP** — This is the intentional breaking change. Any code like `data ==> JSON(@url)` will get a clear error message with the exact fix.

### Migration

```parsley
// Before (no longer works):
payload ==> JSON(@https://api.example.com/users)
payload ==> JSON(@https://api.example.com/users/1).put

// After:
payload =/=> JSON(@https://api.example.com/users)
payload =/=> JSON(@https://api.example.com/users/1).put
```

The error message makes the fix self-evident.

## Related

- Design docs:
  - `work/parsley/design/plan-sftpSupport.md` — Defines operator table and "Why Network Operators" rationale
  - `work/parsley/design/plan-httpFetchImprovements.prompt.md` — Defines arrow direction principle and HTTP method → operator mapping
  - `work/parsley/design/plan-fileIoApi.prompt.md` — Original file I/O design (local operators only)
- Existing operators: `<=/=` (fetch), `==>` (write), `==>>` (append)
- SFTP tests: `pkg/parsley/tests/sftp_test.go` (currently skipped, uses `=/=>`)
- SFTP example: `examples/parsley/sftp_demo.pars` (uses `=/=>`, currently broken)
- Old cheatsheet: `docs/parsley/archive/CHEATSHEET.old.md` (documents `=/=>`)