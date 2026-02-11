---
id: PLAN-086
feature: FEAT-113
title: "Implementation Plan: CLI -e to Output PLN by Default"
status: draft
created: 2026-02-11
---

# Implementation Plan: FEAT-113

## Overview
Change the `-e` flag to output PLN (Parsley Literal Notation) by default, matching REPL behavior. Add `--raw` / `-r` flags for when file-like output is needed.

## Prerequisites
- [x] FEAT-106 (CLI `-e` flag) — already implemented
- [x] REPL PLN output implementation exists (`ObjectToFormattedReprString`)

## Tasks

### Task 1: Add `--raw` and `-r` flags
**Files**: `cmd/pars/main.go`
**Estimated effort**: Small

Steps:
1. Add two new flag definitions after line 37:
   ```go
   rawFlag     = flag.Bool("r", false, "Output raw print string instead of PLN")
   rawLongFlag = flag.Bool("raw", false, "Output raw print string instead of PLN")
   ```
2. In `main()`, after determining `prettyPrint` (~line 77), add logic to determine `raw`:
   ```go
   raw := *rawFlag || *rawLongFlag
   ```
3. Update the `executeInline` call to pass the `raw` parameter

Tests:
- Build succeeds with new flags
- `pars -h` shows new flags (verified in Task 3)

---

### Task 2: Modify `executeInline` to support PLN output
**Files**: `cmd/pars/main.go`
**Estimated effort**: Small

Steps:
1. Update function signature to accept `raw bool` parameter:
   ```go
   func executeInline(code string, args []string, prettyPrint bool, raw bool)
   ```
2. Replace the output logic (lines 190-197) with:
   ```go
   if evaluated == nil {
       if !raw {
           fmt.Println("null")
       }
       return
   }
   
   if evaluated.Type() == evaluator.ERROR_OBJ {
       // ... existing error handling unchanged ...
   }
   
   if raw {
       // File-like behavior (current)
       if evaluated.Type() != evaluator.NULL_OBJ {
           output := evaluator.ObjectToPrintString(evaluated)
           if prettyPrint {
               output = formatter.FormatHTML(output)
           }
           fmt.Println(output)
       }
   } else {
       // REPL-like behavior (new default)
       if evaluated.Type() == evaluator.NULL_OBJ {
           fmt.Println("null")
       } else {
           fmt.Println(evaluator.ObjectToFormattedReprString(evaluated))
       }
   }
   ```

Tests:
- Default mode outputs PLN
- Raw mode outputs print string
- Null handling in both modes

---

### Task 3: Update help text
**Files**: `cmd/pars/main.go`
**Estimated effort**: Small

Steps:
1. In `printHelp()`, update the "Evaluation Options" section to document new behavior:
   ```
   Evaluation Options:
     -e, --eval <code>     Evaluate code string (outputs PLN representation)
     -r, --raw             Output raw print string instead of PLN (with -e)
     --check               Check syntax without executing (can specify multiple files)
   ```
2. Update the examples section to show PLN behavior and `--raw` usage:
   ```
   Examples:
     pars                      Start interactive REPL
     pars script.pars          Execute a Parsley script
     pars -pp page.pars        Execute and pretty-print HTML output
     pars -e "1 + 2"           Evaluate inline code (outputs: 3)
     pars -e "[1, 2, 3]"       Evaluate array (outputs: [1, 2, 3])
     pars -e "[1,2,3]" --raw   Raw output for scripting (outputs: 123)
     pars -e '@args' foo bar   Evaluate code with arguments
     --check               Check syntax without executing (can specify multiple files)
   ```

Tests:
- `pars -h` displays updated help

---

### Task 4: Add CLI integration tests
**Files**: `cmd/pars/main_test.go` (create if needed)
**Estimated effort**: Medium

Steps:
1. Create or update CLI test file with test cases from spec:

| Test Case | Command | Expected Output |
|-----------|---------|-----------------|
| Number | `pars -e "1 + 2"` | `3` |
| String | `pars -e '"hello"'` | `"hello"` |
| Array | `pars -e "[1, 2, 3]"` | `[1, 2, 3]` |
| Dictionary | `pars -e "{a: 1}"` | `{a: 1}` |
| Regex match | `pars -e '"hi" ~ /(\w+)/'` | `["hi", "hi"]` |
| Null | `pars -e "null"` | `null` |
| Raw string | `pars -e '"hello"' --raw` | `hello` |
| Raw array | `pars -e "[1,2,3]" --raw` | `123` |
| Raw HTML | `pars -e '"<b>hi</b>"' -r` | `<b>hi</b>` |
| Raw + pretty | `pars -e '"<div>x</div>"' -r -pp` | Pretty HTML |
| Raw null | `pars -e "null" --raw` | (empty) |

Tests:
- All table-driven tests pass

---

## Validation Checklist
- [ ] All tests pass: `make test`
- [ ] Build succeeds: `make dev`
- [ ] Linter passes: `golangci-lint run`
- [ ] Help text updated
- [ ] Manual testing of all test cases from spec
- [ ] work/BACKLOG.md updated with deferrals (if any)

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| | Task 1: Add flags | ⬜ Not started | — |
| | Task 2: Modify executeInline | ⬜ Not started | — |
| | Task 3: Update help text | ⬜ Not started | — |
| | Task 4: Add tests | ⬜ Not started | — |

## Deferred Items
None anticipated.

## Notes

### Key Implementation Details
- Use `evaluator.ObjectToFormattedReprString()` for PLN output (same as REPL)
- Use `evaluator.ObjectToPrintString()` for raw output (current behavior)
- REPL displays "OK" for null; `-e` should display "null" per spec
- Error handling remains unchanged

### Breaking Change
This changes the default output format for `-e`. Users relying on the current behavior in scripts should add `--raw`.