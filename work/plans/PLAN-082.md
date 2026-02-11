---
id: PLAN-082
feature: FEAT-106
title: "Implementation Plan for CLI Enhancements: Inline Evaluation and Syntax Check"
status: complete
created: 2026-02-09
---

# Implementation Plan: FEAT-106

## Overview
Add `-e/--eval` flag for inline code evaluation and `--check` flag for syntax validation to the `pars` CLI. These features enable quick testing, scripting, and CI/CD integration without creating temporary files.

## Prerequisites
- [x] Feature specification approved (FEAT-106)
- [x] Existing `cmd/pars/main.go` structure reviewed
- [x] Parser and evaluator APIs understood

## Tasks

### Task 1: Add Flag Definitions
**Files**: `cmd/pars/main.go`
**Estimated effort**: Small

Steps:
1. Add `-e/--eval` string flags in the flag declarations section
2. Add `--check` boolean flag in the flag declarations section
3. Update `printHelp()` to document both new flags with usage examples
4. Add examples section showing common usage patterns

Tests:
- Manual: `pars -h` shows new flags
- Manual: `pars --help` shows new flags

---

### Task 2: Implement Inline Evaluation Mode (`-e`)
**Files**: `cmd/pars/main.go`
**Estimated effort**: Medium

Steps:
1. Create `executeInline(code string, args []string, prettyPrint bool)` function
   - Build security policy using existing `buildSecurityPolicy()`
   - Create lexer with code string and filename `<eval>`
   - Parse the program and handle structured errors
   - Create environment with `evaluator.NewEnvironmentWithArgs(args)`
   - Set `env.Filename = "<eval>"` and `env.Security = policy`
   - Evaluate and handle errors
   - Print result if non-null using `evaluator.ObjectToPrintString()`
   - Apply pretty-print formatting if enabled
   - Exit with code 1 on errors, 0 on success
2. Update `main()` to check for `-e/--eval` flag before file execution
   - Prefer `-e` over `--eval` if both are set
   - Pass remaining `flag.Args()` as script arguments
   - Pass pretty-print flag setting
3. Ensure `-e` works with all existing security flags

Tests:
- Simple expression: `pars -e "1 + 2"` outputs `3`
- String output: `pars -e '"hello"'` outputs `hello`
- Println: `pars -e 'println("hi")'` outputs `hi`
- Null result: `pars -e 'let x = 1'` outputs nothing
- With args: `pars -e '@args' foo bar` outputs `["foo", "bar"]`
- With pretty-print: `pars -e '"<div>test</div>"' -pp` formats HTML
- Long flag: `pars --eval "1 + 2"` outputs `3`
- Syntax error: `pars -e "1 +"` shows error and exits 1
- Runtime error: `pars -e "1 / 0"` shows error and exits 1
- Security flags work: `pars -e 'read("/etc/passwd")' --no-read` denied

---

### Task 3: Implement Syntax Check Mode (`--check`)
**Files**: `cmd/pars/main.go`
**Estimated effort**: Medium

Steps:
1. Create `checkFiles(files []string) int` function
   - Iterate through all provided files
   - For each file:
     - Read content with `os.ReadFile()`, return 2 on file error
     - Create lexer with `lexer.NewWithFilename()`
     - Parse with `parser.New()` and `ParseProgram()`
     - Check for structured errors with `p.StructuredErrors()`
     - Print errors using existing `printStructuredErrors()`
   - Return 0 if all files valid, 1 if any syntax errors, 2 on file errors
2. Update `main()` to check for `--check` flag
   - Check must come after subcommand check but before `-e` check
   - Verify at least one file argument provided (error if none)
   - Call `checkFiles()` and exit with returned code
3. Ensure no output on success (Unix convention)
4. Ensure errors go to stderr

Tests:
- Valid file: `pars --check valid.pars` exits 0 with no output
- Invalid syntax: `pars --check invalid.pars` shows errors, exits 1
- File not found: `pars --check missing.pars` shows error, exits 2
- Multiple valid: `pars --check a.pars b.pars` exits 0 with no output
- Mixed results: `pars --check valid.pars invalid.pars` shows errors for invalid, exits 1
- No files: `pars --check` shows error message, exits 2
- Shell glob: `pars --check *.pars` works (shell expands)

---

### Task 4: Integration and Priority Handling
**Files**: `cmd/pars/main.go`
**Estimated effort**: Small

Steps:
1. Update `main()` dispatch logic with correct priority order:
   - Subcommand check (existing: `fmt`)
   - Flag parsing (existing)
   - Help/version flags (existing)
   - **NEW**: `-e/--eval` flag → `executeInline()`
   - **NEW**: `--check` flag → `checkFiles()` + exit
   - File argument → `executeFile()` (existing)
   - No args → REPL (existing)
2. Document edge case behavior:
   - `-e` with file args: file args become `@args` for evaluated code
   - `--check` with `-e`: undefined behavior, `--check` requires file args
3. Ensure proper error exit codes throughout

Tests:
- Priority: `pars -e "1" file.pars` evaluates code, ignores file
- Args pass-through: `pars -e '@args' foo bar` gets `["foo", "bar"]`
- Mutual exclusivity: `-e` and `--check` together shows error or ignores one

---

### Task 5: Documentation and Help Text
**Files**: `cmd/pars/main.go`
**Estimated effort**: Small

Steps:
1. Update `printHelp()` function:
   - Add "Evaluation Options" section after "Display Options"
   - Document `-e, --eval <code>` with description
   - Document `--check` with description
   - Add usage line: `pars -e "code" [args...]`
   - Add usage line: `pars --check <file>...`
2. Add examples section at the end:
   - Inline eval example
   - Syntax check example
   - Combined with other flags example

Tests:
- `pars -h` shows new flags and examples
- `pars --help` shows new flags and examples

---

## Validation Checklist
- [x] All tests pass: `make test` (parsley tests all pass, server test failures pre-existing)
- [x] Build succeeds: `make build`
- [x] Linter passes: `golangci-lint run` (no new issues in cmd/pars)
- [x] Manual testing of all test cases above completed
- [x] Help text includes new flags and examples
- [x] Exit codes are correct (0=success, 1=syntax/runtime error, 2=file error)
- [x] Security flags work with `-e`
- [x] work/BACKLOG.md updated with deferrals (if any)
- [x] FEAT-106.md updated with implementation notes

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-02-09 | Task 1 | ✅ Complete | Flags added and help text updated |
| 2026-02-09 | Task 2 | ✅ Complete | executeInline() implemented with full security support |
| 2026-02-09 | Task 3 | ✅ Complete | checkFiles() implemented with proper exit codes |
| 2026-02-09 | Task 4 | ✅ Complete | Dispatch logic updated, refactored to switch statement |
| 2026-02-09 | Task 5 | ✅ Complete | Help text includes new flags and examples |

## Deferred Items
Items to add to work/BACKLOG.md after implementation:
- Multiple `-e` flags support (e.g., `pars -e "let x=1" -e "x+1"`) — Not needed for MVP, can concatenate with semicolons
- `pars --check -` for stdin syntax checking — Useful for editor integration, defer to future iteration
- `--check` with `--json` output for structured error reporting — Nice for tooling, not required initially

## Notes
- The implementation preserves all existing CLI behavior
- Security flags apply to `-e` mode just like file execution mode
- Exit codes follow Unix conventions: 0=success, 1=user error (syntax/runtime), 2=system error (file not found)
- Pretty-print works with `-e` for HTML formatting
- Remaining args after `-e "code"` become `@args` in the evaluated code, enabling inline scripts with arguments