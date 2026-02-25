---
id: PLAN-102
feature: FEAT-122
title: "Implementation Plan: Swift-Style Variable Declarations (let/var)"
status: complete
created: 2025-01-21
---

# Implementation Plan: FEAT-122

## Overview

Implement Swift-style variable declaration semantics where `let` creates immutable bindings and `var` creates mutable bindings. This includes adding the `var` keyword, enforcing immutability, requiring explicit declarations, and providing a migration tool.

## Prerequisites

- [x] Design decision made (Swift-style `let`/`var`)
- [x] FEAT-122 spec approved
- [ ] Review existing test files for migration scope

## Phase 1: Add `var` Keyword (Non-Breaking)

### Task 1.1: Add VAR Token to Lexer

**Files**: `pkg/parsley/lexer/lexer.go`  
**Estimated effort**: Small

Steps:
1. Add `VAR` constant to TokenType enum (after `LET`)
2. Add `"var": VAR` to keywords map
3. Add `VAR` case to `TokenType.String()` method

Tests:
- Lexer tokenizes `var x = 5` correctly
- `var` is recognized as keyword, not identifier

---

### Task 1.2: Reserve `const` Keyword

**Files**: `pkg/parsley/lexer/lexer.go`  
**Estimated effort**: Small

Steps:
1. Add `CONST` constant to TokenType enum
2. Add `"const": CONST` to keywords map
3. Add `CONST` case to `TokenType.String()` method

Tests:
- `const` cannot be used as identifier
- Error message suggests using `let` instead

---

### Task 1.3: Update AST for Mutability Tracking

**Files**: `pkg/parsley/ast/ast.go`  
**Estimated effort**: Small

Steps:
1. Add `Mutable bool` field to `LetStatement` struct
2. Update `LetStatement.String()` to output `var` when `Mutable` is true
3. Update `LetStatement.TokenLiteral()` to return correct keyword

Tests:
- AST correctly represents `let` vs `var` statements
- String() output matches source syntax

---

### Task 1.4: Parse `var` Statements

**Files**: `pkg/parsley/parser/parser.go`  
**Estimated effort**: Medium

Steps:
1. Add `case lexer.VAR` to `parseStatement()` switch
2. Create `parseVarStatement()` or modify `parseLetStatement()` to accept mutability flag
3. Handle `var` with destructuring: `var [a, b] = arr`, `var {x, y} = obj`
4. Handle `export var x = ...`

Tests:
- `var x = 5` parses correctly
- `var [a, b] = [1, 2]` parses correctly
- `var {x, y} = obj` parses correctly
- `export var x = 5` parses correctly
- Mixing: `let x = 1; var y = 2` both parse

---

### Task 1.5: Update Environment for Immutability Tracking

**Files**: `pkg/parsley/evaluator/evaluator.go`  
**Estimated effort**: Medium

Steps:
1. Rename `letBindings` to `immutable` (or add new `immutable` map)
2. Add `SetImmutable(name string, val Object)` method
3. Add `IsImmutable(name string) bool` method
4. Update `SetLet()` to mark bindings as immutable
5. Add `SetVar()` method for mutable bindings (or use plain `Set()`)

Tests:
- `let` bindings are marked immutable
- `var` bindings are not marked immutable
- `IsImmutable()` checks current and outer scopes correctly

---

### Task 1.6: Evaluate `var` Statements

**Files**: `pkg/parsley/evaluator/eval_statements.go`  
**Estimated effort**: Medium

Steps:
1. Update `evalLetStatement()` to check `Mutable` field
2. Use `SetImmutable()` for `let`, `Set()` for `var`
3. Handle destructuring with correct mutability
4. Handle exports with correct mutability

Tests:
- `var x = 5` creates mutable binding
- `let x = 5` creates immutable binding
- Destructuring respects mutability flag

---

## Phase 2: Enforce Immutability (Breaking)

### Task 2.1: Error on Reassigning Immutable Bindings

**Files**: `pkg/parsley/evaluator/evaluator.go`  
**Estimated effort**: Medium

Steps:
1. Update `Update()` method to check `IsImmutable()`
2. Return error if attempting to reassign immutable binding
3. Create clear error message with hint to use `var`

Tests:
- `let x = 5; x = 10` produces error
- `var x = 5; x = 10` succeeds
- Error message includes original declaration location (if possible)
- Error message suggests using `var`

---

### Task 2.2: Add Error Codes for Immutability

**Files**: `pkg/parsley/errors/errors.go`  
**Estimated effort**: Small

Steps:
1. Add `ASSIGN-0003` (or similar): "cannot reassign immutable binding"
2. Add helpful hints to error template
3. Ensure `var` is in `ParsleyKeywords` list

Tests:
- Error code is correct
- Hints are helpful

---

### Task 2.3: Make Loop Variables Immutable

**Files**: `pkg/parsley/evaluator/eval_control_flow.go`  
**Estimated effort**: Small

Steps:
1. In `evalForExpression()`, bind loop variable as immutable
2. In indexed for `for (i, x in arr)`, bind both as immutable

Tests:
- `for (x in arr) { x = 99 }` produces error
- `for (i, x in arr) { i = 0 }` produces error

---

### Task 2.4: Make Function Parameters Immutable

**Files**: `pkg/parsley/evaluator/eval_expressions.go`  
**Estimated effort**: Small

Steps:
1. In `extendFunctionEnv()`, bind parameters as immutable
2. Handle destructured parameters as immutable

Tests:
- `fn(x) { x = 10 }` produces error when called
- `fn({a, b}) { a = 10 }` produces error when called

---

## Phase 3: Require Explicit Declarations

### Task 3.1: Error on Implicit Declarations

**Files**: `pkg/parsley/evaluator/evaluator.go`  
**Estimated effort**: Medium

Steps:
1. In `Update()`, if variable doesn't exist, return error instead of creating
2. Add error code for implicit declaration
3. Error message should suggest `let` or `var`

Tests:
- `x = 5` (without prior declaration) produces error
- Error suggests using `let x = 5` or `var x = 5`
- Reassignment to existing `var` still works

---

### Task 3.2: Handle `const` Keyword Error

**Files**: `pkg/parsley/parser/parser.go`  
**Estimated effort**: Small

Steps:
1. Add `case lexer.CONST` to `parseStatement()`
2. Return error: "use 'let' for constants in Parsley"

Tests:
- `const x = 5` produces helpful error
- Error suggests using `let` instead

---

## Phase 4: Migration Tool

### Task 4.1: Implement `migrate-let-var` Command ✅

**Files**: `cmd/pars/main.go`, `cmd/pars/migrate.go` (new)  
**Estimated effort**: Large

Steps:
1. ✅ Add `migrate-let-var` subcommand to CLI
2. ✅ Parse all `.pars` files in directory (with `-r` flag for recursive)
3. ✅ Build list of `let` bindings that are reassigned
4. ✅ Generate patches: `let` → `var` for reassigned bindings
5. ✅ Add `let` to implicit declarations (or `var` if reassigned later)
6. ✅ Output diff by default, apply with `-w`/`--write`
7. ✅ Add `-l` flag to list files that need migration

Tests:
- ✅ Correctly identifies reassigned `let` bindings
- ✅ Correctly adds `let` to implicit declarations
- ✅ Correctly adds `var` to implicit declarations that are later reassigned
- ✅ `-w` modifies files in place
- ✅ Without `-w`, only outputs diff
- ✅ Handles nested scopes correctly (inner function/loop scopes)
- ✅ Handles destructuring patterns

---

### Task 4.2: Add Deprecation Warnings (Optional Transitional)

**Files**: `pkg/parsley/evaluator/evaluator.go`  
**Estimated effort**: Small
**Status**: Skipped - Phase 3 already makes implicit declarations errors

Steps:
1. ~~Add warning for implicit declarations (before making them errors)~~
2. ~~Warning suggests migration tool~~

Notes: This was marked optional and is no longer needed since Phase 3 already
implemented hard errors for implicit declarations. The migration tool is the
recommended path for updating code.

---

## Phase 5: Documentation & Tests

### Task 5.1: Update Reference Documentation ✅

**Files**: `docs/parsley/reference.md`  
**Estimated effort**: Medium

Steps:
1. ✅ Update §4.1 "Let Binding" → "Variable Declarations"
2. ✅ Document `let` (immutable) and `var` (mutable)
3. ✅ Document shallow immutability semantics
4. ✅ Update examples throughout document

---

### Task 5.2: Update Cheatsheet ✅

**Files**: `docs/parsley/CHEATSHEET.md`  
**Estimated effort**: Small

Steps:
1. ✅ Update syntax comparison table
2. ✅ Add to "Major Gotchas" section (new section #2)
3. ✅ Update variable examples

---

### Task 5.3: Update FAQ ✅

**Files**: `docs/guide/faq.md`  
**Estimated effort**: Small

Steps:
1. ✅ Add "How do I declare variables?" entry
2. ✅ Add "What's the difference between let and var?" entry
3. ✅ Add migration guidance ("How do I migrate old Parsley code...")

---

### Task 5.4: Migrate Existing Test Files ✅

**Files**: `pkg/parsley/tests/*.pars`, `server/prelude/**/*.pars`  
**Estimated effort**: Medium

Steps:
1. ✅ Run migration tool on test files and prelude
2. ✅ Applied fixes to remaining files (db.pars, dev_error.pars)
3. ✅ All tests pass

---

### Task 5.5: Add New Tests for let/var Semantics ✅

**Files**: `pkg/parsley/tests/let_var_test.pars`, `pkg/parsley/evaluator/let_var_immutability_test.go`  
**Estimated effort**: Medium

Steps:
1. ✅ Test `let` immutability (both .pars and _test.go)
2. ✅ Test `var` mutability
3. ✅ Test destructuring with both
4. ✅ Test error codes and hints
5. ✅ Test loop variable and parameter immutability
6. ✅ Test implicit declaration errors
4. Test exports with both
5. Test loop variable immutability
6. Test parameter immutability
7. Test error messages

---

## Validation Checklist

- [x] All tests pass: `go test ./...` (except pre-existing server infra issues)
- [x] Build succeeds: `make build`
- [x] Linter passes: `golangci-lint run`
- [x] `let x = 5; x = 10` produces error
- [x] `var x = 5; x = 10` succeeds
- [x] `x = 5` (implicit) produces error (Phase 3)
- [x] `const x = 5` produces helpful error
- [x] `export x = 5` is immutable
- [x] `export var x = 5` is mutable
- [x] Loop variables cannot be reassigned
- [x] Function parameters cannot be reassigned
- [x] Migration tool works correctly (Phase 4)
- [x] Documentation updated (Phase 5)
- [x] REPL enforces same semantics

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-01-21 | Task 1.1: Add VAR token | ✅ Complete | Added VAR to lexer TokenType enum and keywords map |
| 2025-01-21 | Task 1.2: Reserve CONST | ✅ Complete | Added CONST as reserved keyword with helpful error |
| 2025-01-21 | Task 1.3: Update AST | ✅ Complete | Added Mutable field to LetStatement, updated String() |
| 2025-01-21 | Task 1.4: Parse var | ✅ Complete | parseLetStatement now accepts mutable flag |
| 2025-01-21 | Task 1.5: Update Environment | ✅ Complete | Added SetVar, SetVarExport, IsImmutable methods |
| 2025-01-21 | Task 1.6: Evaluate var | ✅ Complete | LetStatement uses SetVar/SetLet based on Mutable flag |
| 2025-01-21 | Task 2.1: Error on reassigning immutable | ✅ Complete | Update() now checks IsImmutable() and returns error |
| 2025-01-21 | Task 2.2: Add ASSIGN error codes | ✅ Complete | Added ASSIGN-0001, ASSIGN-0002, ASSIGN-0003 |
| 2025-01-21 | Task 2.3: Make loop variables immutable | ✅ Complete | Loop vars bound via extendFunctionEnv with SetLet |
| 2025-01-21 | Task 2.4: Make function params immutable | ✅ Complete | extendFunctionEnv uses SetLet for all params |
| 2025-01-21 | Phase 2 test fixes | ✅ Complete | Updated prelude components and test files to use var |
| 2025-01-22 | Task 3.1: Error on implicit declarations | ✅ Complete | Update() returns ASSIGN-0004 error when var doesn't exist |
| 2025-01-22 | Task 3.2: Handle const keyword error | ✅ Complete | Already done in Phase 1 |
| 2025-01-22 | Phase 3 test fixes | ✅ Complete | Updated all test files and prelude to use explicit declarations |
| 2025-01-22 | Task 4.1: Implement migrate-let-var | ✅ Complete | New subcommand with -w, -l, -r flags |
| 2025-01-22 | Task 4.2: Deprecation warnings | ⏭️ Skipped | Not needed - Phase 3 already has hard errors |
| 2025-01-22 | Task 5.1: Update reference docs | ✅ Complete | §4.1 rewritten for let/var semantics |
| 2025-01-22 | Task 5.2: Update cheatsheet | ✅ Complete | Added Major Gotcha #2, updated syntax table |
| 2025-01-22 | Task 5.3: Update FAQ | ✅ Complete | Added 3 new entries for let/var |
| 2025-01-22 | Task 5.4: Migrate test files | ✅ Complete | Fixed remaining prelude files |
| 2025-01-22 | Task 5.5: Tests for let/var | ✅ Complete | Already comprehensive from Phase 2/3 |

## Deferred Items

Items to add to work/BACKLOG.md after implementation:
- Deep immutability (frozen objects) — Complex, not needed for v1.0
- `mut` keyword for mutable parameters — Low priority, can add later if needed

## Estimated Total Effort

| Phase | Effort |
|-------|--------|
| Phase 1: Add `var` | 1 day |
| Phase 2: Enforce Immutability | 0.5 day |
| Phase 3: Require Explicit | 0.5 day |
| Phase 4: Migration Tool | 1 day |
| Phase 5: Documentation & Tests | 1 day |
| **Total** | **~4 days** |