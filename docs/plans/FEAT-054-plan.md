---
id: PLAN-032
feature: FEAT-054
title: "Implementation Plan for @now/@timeNow/@dateNow/@today Literals"
status: complete
created: 2025-12-09
completed: 2025-12-09
---

# Implementation Plan: FEAT-054 Datetime Now Literals

## Overview
Replace the `now()` builtin with datetime literal syntax: `@now`, `@timeNow`, `@dateNow`, and `@today`. This aligns with Parsley's literal-based approach and provides granular access to current datetime, time-only, and date-only values.

## Prerequisites
- [x] FEAT-054 spec finalized
- [x] Understand existing datetime literal parsing in lexer

## Tasks

### Task 1: Add Token Types
**Files**: `pkg/parsley/lexer/token/token.go`
**Estimated effort**: Small

Steps:
1. Add `DATETIME_NOW` token type constant
2. Add `TIME_NOW` token type constant
3. Add `DATE_NOW` token type constant

Tests:
- N/A (unit tests in lexer task)

---

### Task 2: Add Lexer Recognition
**Files**: `pkg/parsley/lexer/lexer.go`
**Estimated effort**: Medium

Steps:
1. Locate `@` prefix handling code
2. Add `matchKeyword()` helper if not exists
3. Check for `now`, `timeNow`, `dateNow`, `today` after `@`
4. Return appropriate token type
5. Ensure these are checked before other `@` literal parsing (dates, durations, etc.)

Tests:
- `@now` produces DATETIME_NOW token
- `@timeNow` produces TIME_NOW token
- `@dateNow` produces DATE_NOW token
- `@today` produces DATE_NOW token
- `@2025-12-09` still works (not matched as keyword)
- `@1d` still works (duration)

---

### Task 3: Add AST Node
**Files**: `pkg/parsley/ast/ast.go`
**Estimated effort**: Small

Steps:
1. Add `DatetimeNowLiteral` struct with Token and Kind fields
2. Implement `expressionNode()` method
3. Implement `TokenLiteral()` method
4. Implement `String()` method

Tests:
- N/A (tested via parser)

---

### Task 4: Add Parser Support
**Files**: `pkg/parsley/parser/parser.go`
**Estimated effort**: Small

Steps:
1. Register prefix parser for `DATETIME_NOW`
2. Register prefix parser for `TIME_NOW`
3. Register prefix parser for `DATE_NOW`
4. Each parser creates `DatetimeNowLiteral` with appropriate Kind

Tests:
- Parse `@now` to DatetimeNowLiteral with Kind="datetime"
- Parse `@timeNow` to DatetimeNowLiteral with Kind="time"
- Parse `@dateNow` to DatetimeNowLiteral with Kind="date"
- Parse `@today` to DatetimeNowLiteral with Kind="date"

---

### Task 5: Add Evaluator Support
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Medium

Steps:
1. Add case for `*ast.DatetimeNowLiteral` in main `Eval()` switch
2. Implement `evalDatetimeNowLiteral()` function
3. Handle Kind="datetime" → full datetime dictionary
4. Handle Kind="time" → time-only dictionary
5. Handle Kind="date" → date-only dictionary
6. Ensure dictionaries have correct `kind` field for type system

Tests:
- `@now` returns datetime dictionary with year/month/day/hour/minute/second/kind
- `@timeNow` returns time dictionary with hour/minute/second/kind
- `@dateNow` returns date dictionary with year/month/day/kind
- `@today` returns same as `@dateNow`
- Kind fields are correct strings

---

### Task 6: Deprecate `now()` Builtin
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small

Steps:
1. Locate `now()` builtin in `getBuiltins()`
2. Add deprecation warning log/output
3. Keep function working for backward compatibility
4. Add comment noting deprecation

Tests:
- `now()` still works
- `now()` produces deprecation warning (if warnings implemented)

---

### Task 7: Add Comprehensive Tests
**Files**: `pkg/parsley/tests/datetime_now_test.go` (new file)
**Estimated effort**: Medium

Steps:
1. Create new test file for datetime now literals
2. Test each literal variant
3. Test dictionary structure and kind fields
4. Test datetime methods work on results (`.format()`, `.toISO()`)
5. Test comparisons (`@today == @dateNow`)
6. Test arithmetic (`@2025-12-25 - @today`)
7. Test edge cases

Tests:
- All examples from FEAT-054 spec
- Comparison operations
- Arithmetic operations
- Method calls on results

---

### Task 8: Update Documentation
**Files**: `docs/parsley/reference.md`, `docs/parsley/CHEATSHEET.md`
**Estimated effort**: Small

Steps:
1. Add `@now`, `@timeNow`, `@dateNow`, `@today` to datetime literals section
2. Mark `now()` as deprecated in builtins section
3. Add examples showing new syntax
4. Add migration guide for `now()` → `@now`

Tests:
- N/A (documentation)

---

## Validation Checklist
- [ ] All tests pass: `make test`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] Documentation updated (reference.md, CHEATSHEET.md)
- [ ] BACKLOG.md updated with deferrals (if any)
- [ ] All four literal forms work
- [ ] Existing datetime literals still work
- [ ] `now()` still works (deprecated)

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| | Task 1 | ⬜ Not Started | — |
| | Task 2 | ⬜ Not Started | — |
| | Task 3 | ⬜ Not Started | — |
| | Task 4 | ⬜ Not Started | — |
| | Task 5 | ⬜ Not Started | — |
| | Task 6 | ⬜ Not Started | — |
| | Task 7 | ⬜ Not Started | — |
| | Task 8 | ⬜ Not Started | — |

## Deferred Items
Items to add to BACKLOG.md after implementation:
- Consider removing `now()` in next major version
- Consider `@tomorrow`, `@yesterday` shortcuts (if demand exists)
