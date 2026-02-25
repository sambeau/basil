---
id: PLAN-101
feature: FEAT-120
title: "Implementation Plan: Remove print/println/printf Builtins"
status: complete
created: 2025-01-21
---

# Implementation Plan: FEAT-120

## Overview

Remove the `print()`, `println()`, and `printf()` builtin functions from Parsley and add helpful error messages that teach the expression-based output model. This is a breaking change for 1.0 to establish correct idioms.

## Prerequisites

- [x] FEAT-120 spec finalized
- [x] Understand `PrintValue` type and where it's handled
- [x] Identify all files using print/println/printf

## Tasks

### Task 1: Add Special-Case Error for Removed Print Functions

**Files**: `pkg/parsley/errors/errors.go`, `pkg/parsley/evaluator/eval_infix.go`
**Estimated effort**: Medium

Steps:
1. Add new error code `UNDEF-0020` in `pkg/parsley/errors/errors.go` for "removed print function" with a detailed template explaining expression-based output
2. Create `NewRemovedPrintError(name string)` helper function in `pkg/parsley/errors/errors.go` that:
   - Uses the new error code
   - Adds hints showing the correct pattern (expression-based output)
   - Suggests `log()` for debugging use cases
3. Modify `evalIdentifier()` in `pkg/parsley/evaluator/eval_infix.go` to check for `print`, `println`, `printf` BEFORE the generic undefined identifier error and return the specialized error

Error message format (from spec):
```
Error: Unknown function 'print'

Parsley uses expression-based output — values are returned, not printed.

Instead of:
    print(value)
    print(<div>hello</div>)

Write:
    value
    <div>hello</div>

The last expression in a block becomes its output.

For debugging (console output), use:
    log("debug:", value)
```

Tests:
- Test that `print("hello")` returns the specialized error with hints
- Test that `println("hello")` returns the specialized error
- Test that `printf("hello")` returns the specialized error
- Test "did you mean" suggests `log()` for debugging

---

### Task 2: Remove Print Builtins from Evaluator

**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small

Steps:
1. Remove the `print` builtin function definition from `getBuiltins()` (~line 3524-3529)
2. Remove the `println` builtin function definition from `getBuiltins()` (~line 3530-3542)
3. Note: `printf` doesn't appear to exist as a separate builtin (only in metadata), verify this

Tests:
- Verify `print` is no longer in `getBuiltins()` map
- Verify `println` is no longer in `getBuiltins()` map

---

### Task 3: Remove PrintValue Type and Handling

**Files**: `pkg/parsley/evaluator/evaluator.go`, `pkg/parsley/evaluator/eval_control_flow.go`
**Estimated effort**: Medium

Steps:
1. Remove `PRINT_VALUE_OBJ` constant from ObjectType definitions (~line 84)
2. Remove `PrintValue` struct definition (~lines 218-227)
3. Remove `PrintValue` handling in `evalProgram()` (~lines 5047-5059)
4. Remove `PrintValue` handling in `evalBlockStatement()` (~lines 5094-5106)
5. Remove `PrintValue` handling in `evalInterpolationBlock()` (~lines 5143-5155)
6. Remove `PrintValue` handling in `evalForExpression()` in eval_control_flow.go (~lines 160-165)
7. Remove `PrintValue` handling in `evalForDictExpression()` in eval_control_flow.go (~lines 307-312)

Tests:
- Full test suite passes after removal
- No references to `PrintValue` or `PRINT_VALUE_OBJ` remain in codebase

---

### Task 4: Remove Print Metadata from Introspection

**Files**: `pkg/parsley/evaluator/introspect.go`
**Estimated effort**: Small

Steps:
1. Remove `"print"` entry from `BuiltinMetadata` map (~line 416)
2. Remove `"println"` entry from `BuiltinMetadata` map (~line 417)
3. Remove `"printf"` entry from `BuiltinMetadata` map (~line 418)

Tests:
- `builtins()` function no longer lists print/println/printf
- `describe(print)` returns undefined error (not builtin info)

---

### Task 5: Convert Test Files to Expression Style

**Files**: `pkg/parsley/tests/replace_function_test.pars`
**Estimated effort**: Medium

Steps:
1. Rewrite `replace_function_test.pars` to use expression-based output instead of `print()`:
   - Replace `print("=== Section ===")` with `"=== Section ===\n"`
   - Replace `print(result)` with `result` followed by `"\n"`
   - Replace `print("\n...")` with `"\n..."`
   - Ensure test still validates the same functionality

Tests:
- Run `go test ./...` to verify test file executes correctly
- Output should be equivalent to before

---

### Task 6: Convert Example Files

**Files**: 
- `examples/parsley/reference/21-builtins.pars`
- `examples/parsley/temp/test_table_builtin.pars`
- `examples/parsley/temp/test_table_error1.pars`
- `examples/parsley/temp/test_table_error2.pars`

**Estimated effort**: Small

Steps:
1. Update `21-builtins.pars`:
   - Remove section 7.2 Output that shows `print("hello")` and `println(" world")`
   - Or replace with `log()` example for debugging output
2. Update `test_table_builtin.pars`:
   - Replace all `print(...)` calls with expression output or `log()` for debug
3. Update `test_table_error1.pars`:
   - Replace `print("This should not print...")` with log or remove
4. Update `test_table_error2.pars`:
   - Replace `print("This should not print")` with log or remove

Tests:
- All example files run without error using `pars`

---

### Task 7: Update VS Code Extension Test File

**Files**: `.vscode-extension/test/syntax-test.pars`
**Estimated effort**: Small

Steps:
1. Remove or update the commented print/println/printf lines (~lines 247-250)
2. Keep syntax highlighting test comprehensive but without print references

Tests:
- File still demonstrates all valid Parsley syntax features

---

### Task 8: Update Documentation - CHEATSHEET.md

**Files**: `docs/parsley/CHEATSHEET.md`
**Estimated effort**: Medium

Steps:
1. Reorder gotchas to make "No print() function" the #1 gotcha (currently #1 mentions output but could be clearer)
2. Update section 1 "Output Functions" (~lines 9-20) to explicitly state:
   - `print()` does NOT exist
   - Expression-based output is the idiom
   - `log()` is for debugging
3. Add clear before/after examples showing the mental model shift
4. Remove any other references to print/println throughout the document

Tests:
- Manual review of documentation accuracy

---

### Task 9: Update Documentation - parsley.instructions.md

**Files**: `.github/instructions/parsley.instructions.md`
**Estimated effort**: Small

Steps:
1. The file already has correct guidance under "Output is `log()`, not `print()`"
2. Strengthen the warning - make it even more prominent
3. Add explanation of expression-based output model
4. Add example showing the correct pattern for template output

Tests:
- Manual review of AI guidance effectiveness

---

### Task 10: Update Documentation - copilot-instructions.md

**Files**: `.github/copilot-instructions.md`
**Estimated effort**: Small

Steps:
1. Add explicit rule about Parsley's expression-based output model
2. Reference the fact that `print()` doesn't exist
3. Point to `log()` for debugging

Tests:
- Manual review

---

### Task 11: Update contrib/highlightjs README

**Files**: `contrib/highlightjs/README.md`
**Estimated effort**: Small

Steps:
1. Update the "Usage in HTML" example that shows `print(greeting)` - change to expression style or `log()`
2. Update "Basic Syntax" example that shows `print(greet(name))` - change to expression style
3. Update any other print references in examples

Tests:
- Manual review of README examples

---

### Task 12: Add "Did You Mean" for log() Suggestion

**Files**: `pkg/parsley/errors/errors.go`
**Estimated effort**: Small

Steps:
1. Ensure the specialized print error includes a "Did you mean `log()`?" hint
2. This should be part of Task 1, but verify it's working correctly

Tests:
- Error for `print("debug")` includes hint suggesting `log()`

---

## Validation Checklist

- [x] All tests pass: `go test ./...`
- [x] Build succeeds: `make build`
- [x] Linter passes: `golangci-lint run` (pre-existing issues only)
- [x] `print("hello")` returns helpful error message
- [x] `println("hello")` returns helpful error message  
- [x] `printf("hello")` returns helpful error message
- [x] Error message includes "did you mean `log()`?" hint
- [x] No references to `PrintValue` in codebase
- [x] All example files run without error
- [x] Documentation updated and accurate
- [ ] Manual AI testing shows improved guidance (deferred to human)

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-01-21 | Task 1: Add special-case error | ✅ Complete | Added UNDEF-0020 error code and NewRemovedPrintError helper |
| 2025-01-21 | Task 2: Remove print builtins | ✅ Complete | Removed print, println, printf from getBuiltins() |
| 2025-01-21 | Task 3: Remove PrintValue type | ✅ Complete | Removed type and all 5 handling locations |
| 2025-01-21 | Task 4: Remove introspection metadata | ✅ Complete | Removed from BuiltinMetadata map |
| 2025-01-21 | Task 5: Convert test files | ✅ Complete | Updated replace_function_test.pars, print_test.go, render_test.go, help_test.go |
| 2025-01-21 | Task 6: Convert example files | ✅ Complete | Updated 4 files to use log() or expression style |
| 2025-01-21 | Task 7: Update VS Code test | ✅ Complete | Removed print references from syntax-test.pars |
| 2025-01-21 | Task 8: Update CHEATSHEET.md | ✅ Complete | Made "No print()" the #1 gotcha with clear examples |
| 2025-01-21 | Task 9: Update parsley.instructions.md | ✅ Complete | Strengthened guidance about expression-based output |
| 2025-01-21 | Task 10: Update copilot-instructions.md | ✅ Complete | Added critical warning about no print() |
| 2025-01-21 | Task 11: Update highlightjs README | ✅ Complete | Fixed 3 examples to use expression style |
| 2025-01-21 | Task 12: Verify "did you mean" hint | ✅ Complete | Error message includes log() suggestion |

## Recommended Task Order

1. **Task 1** - Add the error handling first (so we have good errors when we remove things)
2. **Task 2** - Remove builtins from getBuiltins()
3. **Task 3** - Remove PrintValue type and all handling code
4. **Task 4** - Remove introspection metadata
5. **Task 5-7** - Update test and example files (these will fail until above is done)
6. **Task 8-11** - Update documentation
7. **Task 12** - Final verification of error hints

## Deferred Items

Items to add to work/BACKLOG.md after implementation:
- None anticipated - this is a clean removal

## Files Summary

### Core Changes (Go)
- `pkg/parsley/errors/errors.go` - Add UNDEF-0020 and helper
- `pkg/parsley/evaluator/eval_infix.go` - Add special-case detection
- `pkg/parsley/evaluator/evaluator.go` - Remove builtins, PrintValue type, handling
- `pkg/parsley/evaluator/eval_control_flow.go` - Remove PrintValue handling
- `pkg/parsley/evaluator/introspect.go` - Remove metadata entries

### Test/Example Files (Parsley)
- `pkg/parsley/tests/replace_function_test.pars` - Rewrite to expression style
- `examples/parsley/reference/21-builtins.pars` - Remove print examples
- `examples/parsley/temp/test_table_builtin.pars` - Convert to log/expressions
- `examples/parsley/temp/test_table_error1.pars` - Convert to log/expressions
- `examples/parsley/temp/test_table_error2.pars` - Convert to log/expressions
- `.vscode-extension/test/syntax-test.pars` - Remove print references

### Documentation (Markdown)
- `docs/parsley/CHEATSHEET.md` - Major update to gotchas
- `.github/instructions/parsley.instructions.md` - Strengthen guidance
- `.github/copilot-instructions.md` - Add output model rule
- `contrib/highlightjs/README.md` - Fix examples