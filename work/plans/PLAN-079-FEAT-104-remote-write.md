---
id: PLAN-079
feature: FEAT-104
title: "Implementation Plan: Remote Write Operator (=/=> and =/=>>)"
status: complete
created: 2026-02-07
---

# Implementation Plan: FEAT-104 Remote Write Operator

## Overview

Implement the `=/=>` (remote write) and `=/=>>` (remote append) operators across the full compiler pipeline — lexer, parser, AST, evaluator — and enforce the breaking change that `==>` / `==>>` no longer accept network targets. Then update all tests, documentation, and examples.

The spec (`work/specs/FEAT-104.md`) contains the full behavioral specification, grammar, test plan, and documentation plan. This plan maps those requirements to ordered implementation tasks.

## Prerequisites

- [x] FEAT-104 spec approved (`work/specs/FEAT-104.md`)
- [x] Design docs reviewed (`plan-sftpSupport.md`, `plan-httpFetchImprovements.prompt.md`)
- [ ] Feature branch created: `feat/FEAT-104-remote-write-operator`

## Tasks

### Task 1: Add Tokens to Lexer

**Files:** `pkg/parsley/lexer/lexer.go`
**Estimated effort:** Small
**Spec ref:** Grammar → Tokens; Technical Context → Lexer

Steps:
1. Add `REMOTE_WRITE` and `REMOTE_APPEND` token type constants after the existing `APPEND_TO` constant (around L85)
2. Add `String()` cases for both token types (around L275)
3. In `NextToken()`, in the `'='` case, add lookahead check for `=/=>` and `=/=>>` BEFORE the existing `==` / `==>` / `==>>` checks. The `=` followed by `/` is unique and won't conflict:
   - Check `peekChar() == '/'` && `peekCharN(2) == '='` && `peekCharN(3) == '>'`
   - If `peekChar()` after consuming those is `>`, produce `REMOTE_APPEND`; otherwise `REMOTE_WRITE`
   - Use the reference code from spec section "Implementation Reference → Lexer"
4. Add `REMOTE_WRITE` and `REMOTE_APPEND` to `tokenTypeToReadableName` in `parser.go` (around L2839)

Tests (spec Layer 1):
- L1.1: `=/=>` tokenises as `REMOTE_WRITE`
- L1.2: `=/=>>` tokenises as `REMOTE_APPEND`
- L1.3: No ambiguity — `=`, `==`, `==>`, `==>>`, `=>`, `= /regex/` all unchanged
- L1.4: Correct line/column on token

**Validate:** `go test ./pkg/parsley/lexer/...`

---

### Task 2: Add AST Node

**Files:** `pkg/parsley/ast/ast.go`
**Estimated effort:** Small
**Spec ref:** Grammar → AST Node; Technical Context → AST node

Steps:
1. Add `RemoteWriteStatement` struct after the existing `WriteStatement` (around L1060):
   - Fields: `Token lexer.Token`, `Value Expression`, `Target Expression`, `Append bool`
2. Implement interface methods: `statementNode()`, `TokenLiteral()`, `String()`
   - `String()` should output `value =/=> target;` or `value =/=>> target;`
   - Use the reference code from spec

No tests needed for this task alone — parser tests in Task 3 will exercise the node.

**Validate:** `go build ./pkg/parsley/...`

---

### Task 3: Add Parser Support

**Files:** `pkg/parsley/parser/parser.go`
**Estimated effort:** Small
**Spec ref:** Grammar → Precedence; Technical Context → Parser

Steps:
1. In `parseExpressionStatement()`, after the existing `WRITE_TO` / `APPEND_TO` block (around L898), add a check for `REMOTE_WRITE` / `REMOTE_APPEND`:
   - Consume the operator token
   - Create `ast.RemoteWriteStatement` with `Value: expr`, `Append` based on token type
   - Parse target expression at `LOWEST` precedence
   - Consume optional semicolon
   - Use the reference code from spec
2. Add `REMOTE_WRITE` → `'=/=>'` and `REMOTE_APPEND` → `'=/=>>'` to `tokenTypeToReadableName`

Tests (spec Layer 2):
- P2.1: Basic `=/=>` parsing — verify `RemoteWriteStatement` node, `Value`, `Target`, `Append: false`
- P2.2: Basic `=/=>>` parsing — verify `Append: true`
- P2.3: `String()` round-trip — `x =/=> y` → `x =/=> y;`
- P2.4: Semicolon handling — both with and without

**Validate:** `go test ./pkg/parsley/...`

---

### Task 4: Add Evaluator — `evalRemoteWriteStatement`

**Files:** `pkg/parsley/evaluator/eval_network_io.go`, `pkg/parsley/evaluator/evaluator.go`
**Estimated effort:** Medium
**Spec ref:** Behavioral Specification (all subsections); Technical Context → Evaluator — new handler

Steps:
1. In `evaluator.go`, add `case *ast.RemoteWriteStatement` to the `Eval()` switch, calling `evalRemoteWriteStatement(node, env)`
2. In `eval_network_io.go`, add the `evalRemoteWriteStatement` function:
   - Eval `node.Value` and `node.Target`
   - Check for `*SFTPFileHandle` → dispatch to existing `evalSFTPWrite(handle, value, node.Append, env)`
   - Check for request dict (`isRequestDict`) → if `node.Append`, error ("no HTTP append"); otherwise dispatch to existing `evalHTTPWrite(reqDict, value, env)`
   - Check for file dict (`isFileDict`) → error with message suggesting `==>` or `==>>`
   - Anything else → type error
   - Use the reference code from spec, including the `op` variable for correct operator name in error messages

Tests (spec Layers 3–5):
- E3.1–E3.4: HTTP POST/PUT/PATCH via `=/=>` with mock server
- E3.5: Custom headers
- E3.6: HTTP error handling (500, connection refused)
- E3.7: Error capture patterns (let binding, assignment, destructuring)
- E4.1: `=/=>` rejects local file handles
- E4.2: `=/=>` rejects non-handle types (int, string, array, plain dict)
- E5.1: `=/=>>` rejects HTTP targets
- E5.2: `=/=>>` rejects local file handles

**Validate:** `go test ./pkg/parsley/...`

---

### Task 5: Breaking Change — `==>` Rejects Network Targets

**Files:** `pkg/parsley/evaluator/eval_file_io.go`
**Estimated effort:** Small
**Spec ref:** Behavioral Specification → Breaking Change to `==>` and `==>>` ; Technical Context → Evaluator — modify `evalWriteStatement`

Steps:
1. In `evalWriteStatement`, add rejection checks BEFORE the existing `isRequestDict` and `*SFTPFileHandle` dispatch:
   - If target is `isRequestDict`: return error `"operator ==> is for local file writes; use =/=> for network writes"`
   - If target is `*SFTPFileHandle` and `!node.Append`: return error `"operator ==> is for local file writes; use =/=> for network writes"`
   - If target is `*SFTPFileHandle` and `node.Append`: return error `"operator ==>> is for local file appends; use =/=>> for remote appends"`
2. Remove the existing `evalHTTPWrite` dispatch code that is now unreachable (the `isRequestDict` check around L175 and the `evalSFTPWrite` check around L168)

Tests (spec Layer 6):
- E6.1: `==>` rejects HTTP request dicts (3 cases: plain, .put, text format)
- E6.2: `==>` rejects SFTP handles (skipped — no SFTP server)
- E6.3: `==>>` rejects SFTP handles (skipped — no SFTP server)
- E6.4: `==>` still works for local files (regression — text, JSON, append, stdout)

**Validate:** `go test ./pkg/parsley/...` — existing `write_operator_test.go` must still pass

---

### Task 6: Update SFTP Tests

**Files:** `pkg/parsley/tests/sftp_test.go`
**Estimated effort:** Small
**Spec ref:** Test Plan → Layer 7

Steps:
1. Restructure the skipped SFTP tests that use `=/=>` and `=/=>>` syntax:
   - Split each test function into **parse tests** (un-skipped) and **integration tests** (remain skipped)
   - Parse tests verify the input doesn't produce a parse error — they may produce a runtime error (no SFTP server) but should not produce a *syntax* error
2. Specifically un-skip and adapt parse-level cases from:
   - `TestSFTPWriteOperatorSyntax` — verify `=/=>` parses
   - `TestSFTPAppendOperatorSyntax` — verify `=/=>>` parses
   - `TestSFTPErrorCapturePattern` (write case) — verify parse
   - `TestSFTPFormatEncoding` — verify parse
3. Add new tests for `==>` / `==>>` rejection with SFTP targets (E6.2, E6.3 — these can test the error message string without needing an actual SFTP server if we mock the handle creation)

**Validate:** `go test ./pkg/parsley/tests/...`

---

### Task 7: Verify Example File

**Files:** `examples/parsley/sftp_demo.pars`
**Estimated effort:** Small
**Spec ref:** Test Plan → Layer 8

Steps:
1. Run the lexer/parser against `sftp_demo.pars` and verify it produces no parse errors
2. If `pars --check` or similar exists, use that; otherwise write a quick test that reads the file and calls `parser.ParseProgram()` verifying no parser errors
3. Fix any issues in the example file if the syntax has drifted from the implemented grammar

**Validate:** No parse errors from `sftp_demo.pars`

---

### Task 8: Update Syntax Highlighting

**Files:** `contrib/highlightjs/`
**Estimated effort:** Small
**Spec ref:** Documentation Plan → D5

Steps:
1. Find the operator pattern list in the highlight.js grammar file
2. Add `=/=>` and `=/=>>` alongside existing `<=/=`, `==>`, `==>>`, `<==`
3. Update the README if it lists operators

**Validate:** Visual inspection of the pattern list

---

### Task 9: Update Documentation — Network Manual Page

**Files:** `docs/parsley/manual/features/network.md`
**Estimated effort:** Medium
**Spec ref:** Documentation Plan → D1 (D1.1 through D1.5)

Steps:
1. **D1.1** — Replace the "### The Write Operator (`==>`)" section with "### The Remote Write Operator (`=/=>`)" using the draft content from the spec
2. **D1.2** — Add "### Error Handling for Remote Writes" section after the new remote write section
3. **D1.3** — Update the "### Method Accessors" section to include the accessor → operator mapping table
4. **D1.4** — Update the SFTP section: replace `==>` with `=/=>` in examples, add `=/=>>` append example
5. **D1.5** — Add "Local vs network writes" bullet to "Key Differences" section
6. Review all remaining `==>` references in the file — any that show network targets must be updated to `=/=>`

**Validate:** Read through the page end-to-end; all examples should be consistent

---

### Task 10: Update Documentation — Cheatsheet

**Files:** `docs/parsley/CHEATSHEET.md`
**Estimated effort:** Small
**Spec ref:** Documentation Plan → D2 (D2.1, D2.2)

Steps:
1. **D2.1** — In the "### HTTP Requests" section, add `=/=>` POST/PUT/PATCH examples after the existing `<=/=` GET examples
2. **D2.2** — Add a new numbered gotcha "### N. Local vs Network Write Operators" to the "## 🚨 Major Gotchas" section, showing the ❌/✅ pattern from the spec

**Validate:** Search entire file for `==>` with network URLs — none should remain

---

### Task 11: Update Documentation — Reference Pages

**Files:** `docs/parsley/reference.md`, `docs/basil/reference.md`, `docs/parsley/manual/builtins/urls.md`
**Estimated effort:** Medium
**Spec ref:** Documentation Plan → D3, D4, D7

Steps:
1. **D3.1** — `docs/parsley/reference.md` section 2.11 (Precedence Table): add `=/=>` and `=/=>>` to I/O operators row
2. **D3.2** — `docs/parsley/reference.md` section 6.12 (File Operations): add network write examples
3. **D4.1** — `docs/basil/reference.md`: add sections 3.5 (Remote Write) and 3.6 (Remote Append) after existing 3.4 (Fetch URL), using the draft content from the spec. Renumber existing 3.5 (Error Capture Pattern) to 3.7
4. **D4.2** — `docs/basil/reference.md` Appendix A: add feature availability row for `=/=>` / `=/=>>`
5. **D4.3** — `docs/basil/reference.md` section 1.4 (SFTP): replace `==>` with `=/=>` in example
6. **D7** — `docs/parsley/manual/builtins/urls.md`: update "See Also" to include `=/=>` remote write

**Validate:** Search all three files for `==>` with network/SFTP targets — none should remain

---

### Task 12: Update CHANGELOG

**Files:** `CHANGELOG.md`
**Estimated effort:** Small
**Spec ref:** Documentation Plan → D6

Steps:
1. Add a "### Breaking Changes" entry documenting the `==>` / `==>>` rejection of network targets
2. Add an "### Added" entry for the `=/=>` and `=/=>>` operators
3. Use the draft wording from the spec

**Validate:** Read the entries for clarity and accuracy

---

### Task 13: Final Validation & Cleanup

**Files:** All
**Estimated effort:** Small
**Spec ref:** Test Plan → Layer 9; Acceptance Criteria (all)

Steps:
1. Run `make check` (build + test)
2. Run `go test ./...` and verify all tests pass (spec I9.1–I9.4)
3. Run `golangci-lint run` if available
4. Grep the entire codebase for `==>` adjacent to network URLs (`==>.*@https`, `==>.*@sftp`, `==>.*url(`) and fix any remaining occurrences
5. Walk through every acceptance criterion checkbox in FEAT-104 and verify it's met
6. Remove any temporary files created during development
7. Update this plan's progress log

**Validate:** `make check` passes; all acceptance criteria checked off

---

## Task Dependency Order

```
Task 1 (Lexer)
  └→ Task 2 (AST)
       └→ Task 3 (Parser)
            ├→ Task 4 (Evaluator — new operator)
            │    └→ Task 5 (Breaking change to ==>)
            │         ├→ Task 6 (SFTP tests)
            │         └→ Task 7 (Example file)
            └→ Task 8 (Highlight.js) — independent of evaluator
Tasks 9–12 (Documentation) — independent of each other, depend on Tasks 1–5
Task 13 (Final validation) — depends on all above
```

Tasks 9, 10, 11, 12 can be done in parallel once the implementation (Tasks 1–7) is complete.

## Estimated Total Effort

| Task | Effort | Depends on |
|------|--------|------------|
| 1. Lexer tokens | Small | — |
| 2. AST node | Small | 1 |
| 3. Parser support | Small | 2 |
| 4. Evaluator — new operator | Medium | 3 |
| 5. Breaking change to `==>` | Small | 4 |
| 6. SFTP tests | Small | 5 |
| 7. Example file verification | Small | 3 |
| 8. Highlight.js | Small | — |
| 9. Docs — network.md | Medium | 5 |
| 10. Docs — cheatsheet | Small | 5 |
| 11. Docs — reference pages | Medium | 5 |
| 12. CHANGELOG | Small | 5 |
| 13. Final validation | Small | All |
| **Total** | **~1 day** | |

## Validation Checklist

- [x] All tests pass: `go test ./...`
- [x] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [x] No `==>` with network targets anywhere in codebase (grep check)
- [x] `sftp_demo.pars` parses without error
- [x] All FEAT-104 acceptance criteria checked off
- [x] Documentation updated (network.md, CHEATSHEET.md, reference.md ×2, urls.md, highlightjs, CHANGELOG)
- [ ] work/BACKLOG.md updated with deferrals (if any)

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-02-07 | Task 1: Lexer tokens | ✅ Done | REMOTE_WRITE, REMOTE_APPEND tokens added |
| 2026-02-07 | Task 2: AST node | ✅ Done | RemoteWriteStatement with Append flag |
| 2026-02-07 | Task 3: Parser support | ✅ Done | parseExpressionStatement extended |
| 2026-02-07 | Task 4: Evaluator — new operator | ✅ Done | evalRemoteWriteStatement in eval_network_io.go |
| 2026-02-07 | Task 5: Breaking change to ==> | ✅ Done | evalWriteStatement rejects HTTP/SFTP targets |
| 2026-02-07 | Task 6: SFTP tests | ✅ Done | Parse-level tests unskipped; integration tests remain skipped |
| 2026-02-07 | Task 7: Example file | ✅ Done | sftp_demo.pars updated and parses cleanly |
| 2026-02-07 | Task 8: Highlight.js | ✅ Done | Added =\/=>>? to operator regex |
| 2026-02-07 | Task 9: Docs — network.md | ✅ Done | Remote write section, error handling, SFTP, key differences |
| 2026-02-07 | Task 10: Docs — cheatsheet | ✅ Done | HTTP examples and gotcha #10 added |
| 2026-02-07 | Task 11: Docs — reference pages | ✅ Done | basil ref (3.5, 3.6, appendix), parsley ref (precedence, examples), urls.md |
| 2026-02-07 | Task 12: CHANGELOG | ✅ Done | Breaking change + Added entries |
| 2026-02-07 | Task 13: Final validation | ✅ Done | Builds pass, tests pass, grep clean |

## Deferred Items

Items to add to work/BACKLOG.md after implementation:
- Assignment-capture patterns for statement-level operators (e.g., `result = data =/=> target`) are not supported by the parser. The spec describes error-capture patterns but they require parser-level changes to support assignment around statement-level operators. This applies equally to `==>`.
- `golangci-lint run` not yet verified (tool not confirmed available in environment)
- SFTP integration tests remain skipped (require live SFTP server)