---
id: PLAN-048
feature: FEAT-077
title: "Implementation Plan for check, stop, skip statements"
status: complete
created: 2025-12-31
completed: 2025-12-31
---

# Implementation Plan: FEAT-077 (check, stop, skip)

## Overview

Implement three new control flow statements:
- `check CONDITION else VALUE` — precondition with early exit
- `stop` — exit for loop, return accumulated results
- `skip` — skip current iteration (produce null)

## Prerequisites

- [x] Feature spec approved (FEAT-077)
- [x] Design decisions finalized

## Tasks

### Task 1: Add Lexer Tokens ✅

**Files**: `pkg/parsley/lexer/lexer.go`  
**Status**: Complete

Steps:
1. ✅ Add token types: `CHECK`, `STOP`, `SKIP`
2. ✅ Add to `TokenType.String()` switch
3. ✅ Add to `keywords` map

---

### Task 2: Add AST Nodes ✅

**Files**: `pkg/parsley/ast/ast.go`  
**Status**: Complete

Steps:
1. ✅ Add `CheckStatement` node
2. ✅ Add `StopStatement` node
3. ✅ Add `SkipStatement` node

---

### Task 3: Add Parser Support ✅

**Files**: `pkg/parsley/parser/parser.go`  
**Status**: Complete

Steps:
1. ✅ Add `parseCheckStatement()` function
2. ✅ Add `parseStopStatement()` function
3. ✅ Add `parseSkipStatement()` function
4. ✅ Add cases to `parseStatement()` switch

---

### Task 4: Add Signal Types ✅

**Files**: `pkg/parsley/evaluator/evaluator.go`  
**Status**: Complete

Steps:
1. ✅ Add `StopSignal`, `SkipSignal`, `CheckExit` types
2. ✅ Add corresponding `ObjectType` constants
3. ✅ Implement `Type()` and `Inspect()` methods

---

### Task 5-6: Implement Evaluation ✅

**Files**: `pkg/parsley/evaluator/evaluator.go`  
**Status**: Complete

Steps:
1. ✅ Add `evalCheckStatement()` function
2. ✅ Add cases in `Eval()` for new statement types
3. ✅ Update `unwrapReturnValue()` to handle signals

---

### Task 7: Handle Signals in For Loops ✅

**Files**: `pkg/parsley/evaluator/evaluator.go`  
**Status**: Complete

Steps:
1. ✅ Update `evalForExpression()` to handle stop/skip/check signals
2. ✅ Update `evalForDictExpression()` to handle signals

---

### Task 8: Handle Signals in Functions/Blocks ✅

**Files**: `pkg/parsley/evaluator/evaluator.go`  
**Status**: Complete

Steps:
1. ✅ Update `evalBlockStatement()` to bubble up signals
2. ✅ Update `evalInterpolationBlock()` to bubble up signals
3. ✅ Update `evalProgram()` to convert stop/skip to errors at top level

---

### Task 9: Add Tests ✅

**Files**: `pkg/parsley/tests/control_flow_test.go`  
**Status**: Complete

Test cases:
1. ✅ `stop` in for loop exits early with accumulated results
2. ✅ `skip` in for loop skips iteration
3. ✅ `check` passes continues execution
4. ✅ `check` fails returns else value
5. ✅ `stop`/`skip` outside loop produces error
6. ✅ `check` in function acts like early return
7. ✅ Dictionary iteration with stop/skip

---

### Task 10: Update Documentation ✅

**Files**: 
- `docs/parsley/reference.md`
- `docs/parsley/CHEATSHEET.md`

**Status**: Complete

Steps:
1. ✅ Added Control Flow section to reference.md
2. ✅ Updated CHEATSHEET.md with stop/skip examples
3. ✅ Added comparison table entries for stop/skip/check

---

## Summary

Implementation complete! All 10 tasks finished successfully.

### Files Changed
- `pkg/parsley/lexer/lexer.go` - Added CHECK, STOP, SKIP tokens
- `pkg/parsley/ast/ast.go` - Added CheckStatement, StopStatement, SkipStatement nodes
- `pkg/parsley/parser/parser.go` - Added parsing functions
- `pkg/parsley/evaluator/evaluator.go` - Added signal types and evaluation logic
- `pkg/parsley/tests/control_flow_test.go` - New test file
- `docs/parsley/reference.md` - Added Control Flow section
- `docs/parsley/CHEATSHEET.md` - Updated with new features

### Test Files Modified
- `pkg/parsley/tests/markdown_test.go` - Changed variable name from `skip` to `escaped`
- `pkg/parsley/tests/datetime_literals_test.go` - Fixed test case using `not` keyword

Steps:
1. Add `parseCheckStatement()` function
2. Add `parseStopStatement()` function
3. Add `parseSkipStatement()` function
4. Register in `parseStatement()` switch

```go
func (p *Parser) parseCheckStatement() *ast.CheckStatement {
    stmt := &ast.CheckStatement{Token: p.curToken}
    
    p.nextToken() // move past 'check'
    stmt.Condition = p.parseExpression(LOWEST)
    
    if !p.expectPeek(lexer.ELSE) {
        p.addError("check requires 'else' clause", p.curToken.Line, p.curToken.Column)
        return nil
    }
    
    p.nextToken() // move past 'else'
    stmt.ElseValue = p.parseExpression(LOWEST)
    
    return stmt
}

func (p *Parser) parseStopStatement() *ast.StopStatement {
    return &ast.StopStatement{Token: p.curToken}
}

func (p *Parser) parseSkipStatement() *ast.SkipStatement {
    return &ast.SkipStatement{Token: p.curToken}
}
```

Tests:
- Parse `check x != null else error("msg")`
- Parse `check a > 0 else null`
- Parse `stop`
- Parse `skip`
- Error on `check x > 0` without else

---

### Task 4: Add Signal Types to Evaluator

**Files**: `pkg/parsley/evaluator/evaluator.go`  
**Estimated effort**: Small

Steps:
1. Add `STOP_SIGNAL` and `SKIP_SIGNAL` object types
2. Add `StopSignal` and `SkipSignal` structs
3. Add `CheckExit` struct (wraps early exit value)

```go
// Object types (add to const block)
STOP_SIGNAL_OBJ  = "STOP_SIGNAL"
SKIP_SIGNAL_OBJ  = "SKIP_SIGNAL"
CHECK_EXIT_OBJ   = "CHECK_EXIT"

// StopSignal signals loop termination
type StopSignal struct{}
func (s *StopSignal) Type() ObjectType { return STOP_SIGNAL_OBJ }
func (s *StopSignal) Inspect() string  { return "stop" }

// SkipSignal signals iteration skip
type SkipSignal struct{}
func (s *SkipSignal) Type() ObjectType { return SKIP_SIGNAL_OBJ }
func (s *SkipSignal) Inspect() string  { return "skip" }

// CheckExit wraps early exit value from check statement
type CheckExit struct {
    Value Object
}
func (c *CheckExit) Type() ObjectType { return CHECK_EXIT_OBJ }
func (c *CheckExit) Inspect() string  { return c.Value.Inspect() }
```

Tests:
- Signal types have correct Type() and Inspect()

---

### Task 5: Implement check Evaluation

**Files**: `pkg/parsley/evaluator/evaluator.go`  
**Estimated effort**: Medium

Steps:
1. Add case for `*ast.CheckStatement` in `Eval()`
2. Implement `evalCheckStatement()`
3. Check returns `CheckExit` when condition false

```go
case *ast.CheckStatement:
    return evalCheckStatement(node, env)

func evalCheckStatement(node *ast.CheckStatement, env *Environment) Object {
    condition := Eval(node.Condition, env)
    if isError(condition) {
        return condition
    }
    
    if isTruthy(condition) {
        return NULL  // Condition passed, continue execution
    }
    
    // Condition failed - evaluate else value and signal exit
    elseVal := Eval(node.ElseValue, env)
    if isError(elseVal) {
        return elseVal
    }
    
    // Handle special else values for loops
    switch elseVal.(type) {
    case *StopSignal, *SkipSignal:
        return elseVal
    default:
        return &CheckExit{Value: elseVal}
    }
}
```

Tests:
- `check true else null` returns NULL (continue)
- `check false else "error"` returns CheckExit("error")
- `check false else stop` returns StopSignal
- `check false else skip` returns SkipSignal

---

### Task 6: Implement stop/skip Evaluation

**Files**: `pkg/parsley/evaluator/evaluator.go`  
**Estimated effort**: Small

Steps:
1. Add cases for `*ast.StopStatement` and `*ast.SkipStatement` in `Eval()`
2. Return singleton signal objects

```go
case *ast.StopStatement:
    return &StopSignal{}

case *ast.SkipStatement:
    return &SkipSignal{}
```

Note: Context validation (must be in loop) done at eval time by checking if signal bubbles up incorrectly.

Tests:
- `stop` returns StopSignal
- `skip` returns SkipSignal

---

### Task 7: Handle Signals in For Loops

**Files**: `pkg/parsley/evaluator/evaluator.go`  
**Estimated effort**: Medium

Steps:
1. Modify `evalForExpression()` to handle StopSignal, SkipSignal, CheckExit
2. Modify `evalForDictExpression()` similarly
3. StopSignal: return accumulated results
4. SkipSignal: continue to next iteration
5. CheckExit with StopSignal: return accumulated
6. CheckExit with other value: return that value

```go
// In the for loop body evaluation section:
evaluated = evalStatement(stmt, extendedEnv)

// Check for control flow signals
switch sig := evaluated.(type) {
case *StopSignal:
    // Exit loop, return what we have
    return &Array{Elements: result}
case *SkipSignal:
    // Skip this iteration
    evaluated = NULL
    break  // break out of statement loop, continue for loop
case *CheckExit:
    // Check statement triggered exit
    switch inner := sig.Value.(type) {
    case *StopSignal:
        return &Array{Elements: result}
    case *SkipSignal:
        evaluated = NULL
        break
    default:
        // Return the else value directly
        return sig.Value
    }
case *ReturnValue:
    // ... existing return handling
}
```

Tests:
- For loop with `stop` returns partial array
- For loop with `skip` filters values
- For loop with `check X else stop` returns on failure
- For loop with `check X else skip` skips on failure
- For loop with `check X else []` returns empty on failure
- Nested loops: stop/skip affect innermost only

---

### Task 8: Handle Signals in Functions/If Blocks

**Files**: `pkg/parsley/evaluator/evaluator.go`  
**Estimated effort**: Medium

Steps:
1. Modify `evalBlockStatement()` to handle CheckExit
2. Modify function application to handle CheckExit
3. CheckExit should unwrap and become the block/function result
4. Stop/Skip outside loops should error

```go
// In evalBlockStatement, after evaluating statement:
if checkExit, ok := result.(*CheckExit); ok {
    return checkExit.Value  // Unwrap and return as block result
}

// Error if stop/skip appear in non-loop context
if _, ok := result.(*StopSignal); ok {
    return newError("'stop' can only be used inside a for loop")
}
if _, ok := result.(*SkipSignal); ok {
    return newError("'skip' can only be used inside a for loop")
}
```

Tests:
- Function with `check x else error("msg")` returns error early
- Function with `check x else null` returns null early
- If block with check returns early
- `stop` outside loop gives error
- `skip` outside loop gives error

---

### Task 9: Add Comprehensive Tests

**Files**: `pkg/parsley/tests/control_flow_test.go` (new)  
**Estimated effort**: Medium

Test categories:
1. check statement basics
2. check in functions
3. check in if blocks
4. check in for loops
5. stop in for loops
6. skip in for loops
7. Combined patterns
8. Error cases
9. Edge cases (nested loops, nested checks)

---

### Task 10: Update Documentation

**Files**: 
- `docs/parsley/reference.md`
- `docs/parsley/CHEATSHEET.md`

**Estimated effort**: Small

Steps:
1. Add Control Flow section with check/stop/skip
2. Add examples
3. Update Quick Reference tables

---

## Validation Checklist

- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] Documentation updated
- [ ] BACKLOG.md updated with deferrals (if any)

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| | Task 1: Lexer tokens | ⬜ Not started | |
| | Task 2: AST nodes | ⬜ Not started | |
| | Task 3: Parser | ⬜ Not started | |
| | Task 4: Signal types | ⬜ Not started | |
| | Task 5: check eval | ⬜ Not started | |
| | Task 6: stop/skip eval | ⬜ Not started | |
| | Task 7: For loop signals | ⬜ Not started | |
| | Task 8: Function/if signals | ⬜ Not started | |
| | Task 9: Tests | ⬜ Not started | |
| | Task 10: Documentation | ⬜ Not started | |

## Deferred Items

None anticipated.

## Implementation Order

Recommended order for incremental development:

1. **Phase 1: stop and skip** (simpler, no `else` clause)
   - Tasks 1-2 (partial), 4, 6, 7, 9 (partial)
   - Delivers immediate value for loop control

2. **Phase 2: check statement**
   - Tasks 1-2 (complete), 3, 5, 8, 9 (complete)
   - Adds precondition pattern

3. **Phase 3: Documentation**
   - Task 10

This allows testing and validation at each phase.
