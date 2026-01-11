---
id: FEAT-030
title: "fail() Function for User-Defined Catchable Errors"
status: implemented
priority: medium
created: 2025-12-05
author: "@copilot"
---

# FEAT-030: fail() Function for User-Defined Catchable Errors

## Summary
Add a `fail()` built-in function that allows user code to create catchable errors. This enables user-defined functions to participate in the `try` expression error handling system introduced in FEAT-029. Without this, `try` can only catch errors from built-in functions and system operations.

## User Story
As a Parsley developer, I want to create catchable errors in my functions so that callers can use `try` to handle expected failure conditions gracefully.

## Motivation

Currently, `try` only catches errors from built-in functions:
```parsley
// Works - url() can fail with Network/Format errors
let {result, error} = try url("https://api.example.com/data")

// Doesn't work - user functions can't signal catchable errors
let validate = fn(email) {
  if (!email.contains("@")) {
    // ??? How do we signal this is invalid?
  }
  email
}
let {result, error} = try validate("not-an-email")  // No way to catch
```

With `fail()`:
```parsley
let validate = fn(email) {
  if (!email.contains("@")) {
    fail("Invalid email format: missing @")
  }
  email
}
let {result, error} = try validate("not-an-email")
// error = "Invalid email format: missing @"
```

## Acceptance Criteria
- [x] `fail(message)` creates a catchable error
- [x] `try` catches errors created by `fail()`
- [x] Non-string arguments produce a Type error
- [x] `fail()` with no arguments produces Arity error
- [x] Errors propagate through call stack until caught by `try`
- [x] Uncaught `fail()` terminates with error message (like other errors)

## Design Decisions

### Error Class: Value
**Decision**: `fail()` creates errors with the `Value` error class.

**Rationale**: 
- Value errors are for "semantically invalid" data (wrong meaning, not wrong type)
- User validation errors fit this: the email string is valid data, but invalid for use as an email
- Value is catchable, which is the whole point
- Alternative `User` class adds complexity without benefit

### Function Name: `fail()`
**Decision**: Use `fail()` rather than `error()`, `throw()`, or `raise()`.

**Rationale**:
- **Avoids namespace clash**: `error()` would conflict with the established `{result, error}` destructuring pattern. Local variables shadow builtins in Parsley, so `let {result, error} = try func()` would make `error()` uncallable in that scope.
- **Natural pairing**: `try` and `fail` pair well in English — you try something, it might fail
- Parsley doesn't have exceptions, so `throw` is misleading
- `raise` is Python-specific terminology

### Signature Options

**Option A: Message only (recommended)**
```parsley
fail("Something went wrong")
```
Simple, covers 95% of use cases. Error class is always Value.

**Option B: Message + class**
```parsley
fail("File not found", "IO")
```
Allows specifying error class. More flexible but:
- Adds complexity
- Users might misuse (creating Type errors when they mean Value)
- Edge case: what if they specify a non-catchable class?

**Option C: Structured error**
```parsley
fail({message: "Invalid", code: "VAL-001", details: {...}})
```
Most flexible but complex. Probably overkill.

**Recommendation**: Start with Option A. Can extend later if needed.

### Why Not Just Return `{result, error}`?

An alternative to `fail()` is having functions manually return a `{result, error}` dictionary:

```parsley
// Manual approach
let validate = fn(email) {
  if (!email.contains("@")) {
    {result: null, error: "Invalid email format"}
  } else {
    {result: email, error: null}
  }
}
let {result, error} = validate("bad")  // No try needed
```

**Comparison:**

| Aspect | `fail()` | Manual `{result, error}` |
|--------|-----------|-------------------------|
| Call site | `try validate(x)` | `validate(x)` |
| Forgetting to handle | Uncaught error crashes | Silent `null` propagates |
| Propagation | Automatic through call stack | Must manually pass up |
| Return type | Normal type (string) | Always dict |
| Composition | Works with existing code | Caller must know pattern |

**The Big Win: Automatic Propagation**

```parsley
// With fail() - propagates automatically
let step1 = fn(x) { if (x < 0) { fail("negative") }; x }
let step2 = fn(x) { step1(x) * 2 }
let step3 = fn(x) { step2(x) + 1 }

let {result, error} = try step3(-5)  // Catches error from step1!
```

```parsley
// With manual return - tedious, error-prone
let step1 = fn(x) { 
  if (x < 0) { {result: null, error: "negative"} } 
  else { {result: x, error: null} }
}
let step2 = fn(x) { 
  let {result, error} = step1(x)
  if (error) { {result: null, error: error} }  // Must forward!
  else { {result: result * 2, error: null} }
}
// ... and so on for every layer
```

**When to use each:**
- **Use `fail()`**: Failure is exceptional but expected; you want automatic propagation; caller might forget to check
- **Use manual `{result, error}`**: Both outcomes are equally "normal" (e.g., lookup that may not find); returning multiple values anyway

The `fail()` approach is essentially Go's `if err != nil { return err }` pattern, but automatic.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `pkg/parsley/evaluator/builtins.go` — Add `fail` builtin function
- `pkg/parsley/errors/errors.go` — May need helper to create Value errors

### Dependencies
- Depends on: FEAT-029 (try expression), FEAT-023 (structured errors)
- Blocks: None

### Edge Cases & Constraints

1. **`fail()` without `try`** — Error propagates up and terminates program with message. Same as any other uncaught error.

2. **Nested function calls** — Error bubbles through stack:
   ```parsley
   let inner = fn() { fail("oops") }
   let outer = fn() { inner() }
   let {result, error} = try outer()  // Catches error from inner
   ```

3. **`fail()` in callbacks** — Works same as anywhere:
   ```parsley
   let items = [1, 2, 3]
   let result = try items.map(fn(x) {
     if (x == 2) { fail("don't like 2") }
     x
   })
   ```

4. **Empty message** — `fail("")` is valid, creates error with empty message.

5. **Null message** — `fail(null)` — Type error? Or coerce to "null"?

6. **Non-catchable by design** — `fail()` ALWAYS creates catchable errors. If you want a non-catchable error (programming mistake), just let it fail naturally.

### Implementation Sketch

```go
// In builtins.go
"fail": func(ctx EvalContext, args ...object.Object) object.Object {
    if len(args) != 1 {
        return newArityError(ctx, "fail", 1, len(args))
    }
    
    msg, ok := args[0].(*object.String)
    if !ok {
        return newTypeError(ctx, "fail() requires a string argument")
    }
    
    // Create a Value-class structured error
    return newStructuredError(
        ctx,
        errors.Value,      // Always Value class
        "USER-0001",       // Fixed code for all user errors
        msg.Value,
        nil,               // No extra details
    )
}
```

### Resolved Questions

1. **Error code**: Fixed `USER-0001` for all user errors. If more sophisticated error codes are needed in the future, we can extend the signature then.

2. **Stack trace**: No stack trace for MVP. Parsley errors don't currently include stack traces; adding them would be a broader effort.

3. **Type coercion**: Strict — `fail()` requires a string argument. Non-strings produce a Type error. Use `fail(string(value))` if coercion is needed.

## Related
- FEAT-029: Try expression (prerequisite)
- FEAT-023: Structured errors (provides error infrastructure)
- BACKLOG.md entry: "fail() function for user-defined catchable errors"
