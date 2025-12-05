---
id: PLAN-018
feature: FEAT-029
title: "Implementation Plan for Try Expression"
status: complete
created: 2025-12-05
---

# Implementation Plan: FEAT-029 Try Expression

## Overview
Implement the `try` keyword that wraps function/method calls and catches "user errors" (IO, Network, Database, Format, Value, Security) while propagating "developer errors" (Type, Arity, Undefined, Parse, Operator, Index, Internal). Returns `{result, error}` dictionary.

## Prerequisites
- [x] FEAT-023: Structured Error Objects (provides error classes)
- [x] Error classification system in place

## Tasks

### Task 1: Add TRY Token to Lexer
**Files**: `pkg/parsley/lexer/lexer.go`
**Estimated effort**: Small

Steps:
1. Add `TRY` constant to token types
2. Add "try" to keyword map

Tests:
- Lexer recognizes `try` as keyword token
- `try` not confused with identifiers starting with "try"

---

### Task 2: Add TryExpression AST Node
**Files**: `pkg/parsley/ast/ast.go`
**Estimated effort**: Small

Steps:
1. Define `TryExpression` struct with Token and Call fields
2. Implement `expressionNode()`, `TokenLiteral()`, `String()` methods

```go
type TryExpression struct {
    Token token.Token  // The 'try' token
    Call  Expression   // Must be a CallExpression
}
```

Tests:
- AST node stringifies correctly: `try func()` → `try func()`

---

### Task 3: Parse try Expression
**Files**: `pkg/parsley/parser/parser.go`
**Estimated effort**: Medium

Steps:
1. Register `TRY` as prefix parse function
2. Implement `parseTryExpression()`:
   - Consume `try` token
   - Parse next expression
   - Validate it's a `CallExpression` (function or method call)
   - Return error if not a call expression
3. Handle edge cases (try on literals, operators, identifiers)

```go
func (p *Parser) parseTryExpression() ast.Expression {
    expr := &ast.TryExpression{Token: p.curToken}
    p.nextToken()
    
    // Parse the expression after try
    expr.Call = p.parseExpression(PREFIX)
    
    // Validate it's a call expression
    if _, ok := expr.Call.(*ast.CallExpression); !ok {
        p.addError("try requires a function call", ...)
        return nil
    }
    
    return expr
}
```

Tests:
- `try func()` parses correctly
- `try obj.method()` parses correctly
- `try 5` gives syntax error
- `try a + b` gives syntax error
- `try someVar` gives syntax error
- `try func(nested())` parses (nested calls OK)

---

### Task 4: Add isCatchableError Helper
**Files**: `pkg/parsley/errors/errors.go` or `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small

Steps:
1. Create function to check if an error class is catchable
2. Catchable: ClassIO, ClassNetwork, ClassDatabase, ClassFormat, ClassValue, ClassSecurity
3. Not catchable: ClassParse, ClassType, ClassArity, ClassUndefined, ClassOperator, ClassIndex, ClassState, ClassInternal, ClassImport

```go
func IsCatchableError(class ErrorClass) bool {
    switch class {
    case ClassIO, ClassNetwork, ClassDatabase, 
         ClassFormat, ClassValue, ClassSecurity:
        return true
    default:
        return false
    }
}
```

Tests:
- IO errors are catchable
- Network errors are catchable
- Type errors are NOT catchable
- Index errors are NOT catchable

---

### Task 5: Evaluate TryExpression
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Medium

Steps:
1. Add case for `*ast.TryExpression` in Eval switch
2. Evaluate the call expression
3. If result is an error:
   - Check if error class is catchable
   - If catchable: return `{result: null, error: <message>}` dictionary
   - If not catchable: propagate error (return it unchanged)
4. If result is not an error:
   - Return `{result: <value>, error: null}` dictionary

```go
case *ast.TryExpression:
    result := Eval(node.Call, env)
    
    if err, ok := result.(*Error); ok {
        if IsCatchableError(err.Class) {
            // Wrap in result dictionary
            return &Dictionary{
                Pairs: map[string]Object{
                    "result": NULL,
                    "error":  &String{Value: err.Message},
                },
                Order: []string{"result", "error"},
            }
        }
        // Non-catchable - propagate
        return err
    }
    
    // Success - wrap in result dictionary
    return &Dictionary{
        Pairs: map[string]Object{
            "result": result,
            "error":  NULL,
        },
        Order: []string{"result", "error"},
    }
```

Tests:
- `try` on successful call returns `{result: value, error: null}`
- `try` on IO error returns `{result: null, error: "message"}`
- `try` on type error still halts (propagates error)
- `try` on arity error still halts
- Destructuring works: `let {result, error} = try func()`
- Nested calls: `try outer(inner())` catches errors from either

---

### Task 6: Integration Tests
**Files**: `pkg/parsley/tests/try_test.go` (new file)
**Estimated effort**: Medium

Tests to write:
1. Basic success case
2. Basic catchable error case (IO)
3. Non-catchable error propagates (Type, Arity, Undefined)
4. Method calls work
5. Nested calls work
6. Destructuring patterns
7. Null coalescing: `(try func()).result ?? default`
8. Error message preserved
9. With pseudo-types: `try time(badString)`, `try url(badUrl)`
10. With file operations: `try file.read()`
11. With database: `try db.query(badSql)`

---

### Task 7: Documentation
**Files**: `docs/parsley/reference.md`, `docs/parsley/CHEATSHEET.md`
**Estimated effort**: Small

Steps:
1. Add `try` to reference.md with grammar and examples
2. Add to CHEATSHEET.md as common pattern
3. Update error handling section if exists

---

## Validation Checklist
- [x] All tests pass: `go test ./...`
- [x] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [x] Documentation updated
- [x] FEAT-029 status updated to `implemented`
- [x] BACKLOG.md updated with deferrals (if any)

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-12-05 | Task 1: Lexer | ✅ Complete | Added TRY token and keyword |
| 2025-12-05 | Task 2: AST | ✅ Complete | Added TryExpression struct |
| 2025-12-05 | Task 3: Parser | ✅ Complete | Added parseTryExpression, validates call expr |
| 2025-12-05 | Task 4: isCatchableError | ✅ Complete | Added IsCatchable() method on ErrorClass |
| 2025-12-05 | Task 5: Evaluator | ✅ Complete | Added evalTryExpression |
| 2025-12-05 | Task 6: Tests | ✅ Complete | Created try_test.go with 14 tests |
| 2025-12-05 | Task 7: Documentation | ✅ Complete | Updated reference.md and CHEATSHEET.md |

## Notes

### Error Dictionary Structure
The `try` expression returns a dictionary with guaranteed keys:
- `result` - the return value on success, `null` on error
- `error` - `null` on success, error message string on error

This matches the existing `<==` pattern for file operations.

### Edge Cases to Consider
- `try try func()` - nested try: should work, inner try returns dict, outer sees success
- `try null.method()` - calling method on null: this is a Type error, should NOT be caught
- Empty error message - ensure error always has meaningful message
