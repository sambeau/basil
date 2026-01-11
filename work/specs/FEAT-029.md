---
id: FEAT-029
title: "Try Expression for Catchable Errors"
status: implemented
priority: medium
created: 2025-12-05
author: "@human"
---

# FEAT-029: Try Expression for Catchable Errors

## Summary
Add a `try` expression that wraps function/method calls and catches "user errors" (I/O failures, invalid input, network issues) while still halting on "developer errors" (undefined variables, type mismatches, syntax errors). Returns a `{result, error}` dictionary consistent with the existing `<==` pattern.

## User Story
As a Parsley developer, I want to catch and handle runtime errors from user input and I/O operations so that my scripts can gracefully handle failures without crashing, while still getting immediate feedback on bugs in my code.

## Acceptance Criteria
- [x] `try` keyword parses before function/method call expressions
- [x] `try func()` returns `{result: <value>, error: null}` on success
- [x] `try func()` returns `{result: null, error: <message>}` on catchable error
- [x] Catchable errors: IO, Network, Database, Format, Value, Security classes
- [x] Non-catchable errors: Type, Arity, Undefined, Parse, Internal, Operator classes
- [x] Works with destructuring: `let {result, error} = try func()`
- [x] Nested calls are wrapped: `try outer(inner(x))` catches errors from both
- [x] `try` on non-call expression is a syntax error
- [x] Error message includes original error details

## Design Decisions

- **Go-style explicit errors**: Rather than try/catch blocks (JavaScript) or exceptions (Python), we use explicit return values. This is more explicit, composable, and fits Parsley's aesthetic.

- **Function calls only**: `try` only applies to function/method calls, not arbitrary expressions. This makes it clear exactly where errors can occur.

- **Error classification**: Errors are split into "developer errors" (bugs) that always halt, and "user errors" (environment/input) that can be caught. This distinction helps developers catch the right things.

- **Consistent with `<==`**: The `{result, error}` return dictionary matches the existing pattern from file read operations, so users don't need to learn a new pattern.

- **No unwrap methods (yet)**: Keeping it simple with just destructuring. Methods like `.unwrap()`, `.unwrapOr()` could be added later if needed.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Syntax

```parsley
// Basic usage
let {result, error} = try someFunction(arg1, arg2)

// Method calls
let {data, error} = try file.read()
let {rows, error} = try db.query(sql)

// Nested calls (catches errors from any call in the chain)
let {html, error} = try markdown(file.read(@./README.md))

// Can ignore result or error
let {error} = try riskyOperation()
let {result} = try maybeWorks()  // error discarded

// Use with null coalescing
let value = (try parseDate(input)).result ?? now()

// Syntax errors (try only works with calls)
let x = try 5           // ERROR: expected function call
let y = try a + b       // ERROR: expected function call
let z = try someVar     // ERROR: expected function call
```

### Error Classification

**Catchable by `try`** (user/environment errors):
| Class | Examples |
|-------|----------|
| IO | File not found, permission denied, read/write failure |
| Network | Connection refused, timeout, DNS failure |
| Database | Query failed, connection lost, constraint violation |
| Format | Invalid JSON, bad date format, malformed URL |
| Value | Invalid currency code, out of range, bad pattern |
| Security | Access denied (when security policy blocks operation) |

**Not catchable** (developer errors - always halt):
| Class | Examples |
|-------|----------|
| Parse | Syntax errors |
| Type | Type mismatch, wrong argument type |
| Arity | Wrong number of arguments |
| Undefined | Unknown identifier, unknown method |
| Operator | Invalid operator for types |
| Index | Index out of bounds |
| Internal | Interpreter bugs |

### Design Rationale: Why This Split?

The key question: **Can you validate before calling?**

| Situation | Validate First? | Use `try`? |
|-----------|-----------------|------------|
| Array index | ✅ `if (i < arr.length())` | No need |
| Dict key | ✅ `if (dict.has(key))` or `dict[key]?` | No need |
| File exists | ❌ Race condition, might fail anyway | ✅ Yes |
| Network request | ❌ Can't know if server is up | ✅ Yes |
| Parse user input | ❌ Can't know if valid without trying | ✅ Yes |
| DB query | ❌ Connection could drop mid-query | ✅ Yes |
| Date parsing | ❌ Complex validation, just try it | ✅ Yes |

**Catchable errors** are for things that can fail due to **external factors you can't check in advance**.

**Non-catchable errors** are for things where you have all the information needed to avoid the error. Index out of bounds? You know the array length. Check it. Type mismatch? That's a bug in your code.

This also means **wrapping a non-catchable error in a function doesn't make it catchable**:

```parsley
let safeGet = fn(arr, i) { arr[i] }
let {result, error} = try safeGet(myArray, 100)
// Still halts! IndexError is not catchable regardless of where it occurs.
```

If you want safe index access, validate explicitly:
```parsley
let safeGet = fn(arr, i) {
  if (i >= 0 && i < arr.length()) { arr[i] } else { null }
}
```

### Affected Components

- `pkg/parsley/lexer/lexer.go` — Add `TRY` token
- `pkg/parsley/parser/parser.go` — Parse `try` as prefix expression, validate RHS is call
- `pkg/parsley/ast/ast.go` — Add `TryExpression` node
- `pkg/parsley/evaluator/evaluator.go` — Evaluate try, catch appropriate error classes
- `pkg/parsley/errors/errors.go` — Add helper to check if error class is catchable

### Implementation Approach

1. **Lexer**: Add `TRY` keyword token

2. **Parser**: Parse `try` as prefix operator
3. 
   ```go
   case lexer.TRY:
       return p.parseTryExpression()
   ```
   Validate that the expression after `try` is a CallExpression.

3. **AST**: New node type
   ```go
   type TryExpression struct {
       Token token.Token
       Call  Expression  // Must be CallExpression
   }
   ```

4. **Evaluator**: Wrap evaluation in error check
   ```go
   case *ast.TryExpression:
       result := Eval(node.Call, env)
       if err, ok := result.(*Error); ok {
           if isCatchableError(err) {
               return &Dictionary{
                   Pairs: map[string]Object{
                       "result": NULL,
                       "error":  &String{Value: err.Message},
                   },
               }
           }
           // Non-catchable error - propagate
           return err
       }
       return &Dictionary{
           Pairs: map[string]Object{
               "result": result,
               "error":  NULL,
           },
       }
   ```

5. **Error classification helper**:
   ```go
   func isCatchableError(err *Error) bool {
       switch err.Class {
       case ClassIO, ClassNetwork, ClassDatabase, 
            ClassFormat, ClassValue, ClassSecurity:
           return true
       default:
           return false
       }
   }
   ```

### Edge Cases & Constraints

1. **Nested try**: `try try func()` — Should be syntax error (or just redundant?)
2. **Try in try**: Inner try catches first, outer try sees success
3. **Empty call**: `try ()` — Syntax error, no function to call
4. **Try on builtin**: Works the same as user functions
5. **Try on method of null**: `try null.foo()` — Should this be catchable? Probably not (it's a bug)

### Interaction with `<==`

The `<==` operator already returns `{data, error}`. With `try`, users have two equivalent patterns:

```parsley
// Pattern 1: <== (existing, for file reads)
let {data, error} <== JSON(@./config.json)

// Pattern 2: try (new, general purpose)
let {result, error} = try JSON(@./config.json).read()
```

Both are valid. `<==` remains as convenient sugar for file operations.

### Future Considerations

- **Result methods**: `.unwrap()`, `.unwrapOr(default)`, `.map(fn)` could be added later
- **Index errors**: Could make index-out-of-bounds catchable (currently halts)
- **Custom error types**: User-defined errors that are catchable
- **Error chaining**: Preserving error stack for debugging

## Dependencies
- Depends on: FEAT-023 (Structured Error Objects) — need error classes
- Blocks: None

## Related
- FEAT-023: Structured Error Objects (provides error classification)
- The `<==` operator (similar pattern for file reads)
