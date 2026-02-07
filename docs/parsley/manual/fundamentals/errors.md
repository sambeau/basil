---
id: man-pars-errors
title: Error Handling
system: parsley
type: fundamentals
name: errors
created: 2026-02-05
version: 0.2.0
author: Basil Team
keywords:
  - error
  - try
  - fail
  - check
  - catchable
  - error class
  - error code
  - error handling
  - optional access
  - null coalescing
---

# Error Handling

Parsley divides errors into two camps: **catchable** errors from external factors (file not found, network timeout, bad input) and **non-catchable** errors from logic bugs (type mismatch, wrong argument count, undefined variable). Only catchable errors can be intercepted with `try` — logic bugs halt execution immediately.

There are no try/catch blocks. Instead, `try` wraps a single function or method call and returns a result dictionary.

## try

`try` calls a function or method and catches recoverable errors. It always returns a dictionary with `result` and `error` keys:

```parsley
let risky = fn() { fail("oops") }
try risky()                      // {result: null, error: "oops"}

let safe = fn() { 42 }
try safe()                       // {result: 42, error: null}
```

Destructure for clean handling:

```parsley
let {result, error} = try risky()
if (error) {
    log("Failed: " + error)
} else {
    log("Got: " + result)
}
```

> ⚠️ `try` only wraps function and method calls — not arbitrary expressions. `try 1 + 2` is a parse error. Write `try fn() { 1 + 2 }()` if you need to wrap an expression.

### What try Returns

| Call outcome | `result` | `error` |
|---|---|---|
| Success | The return value | `null` |
| Catchable error | `null` | Error message string |
| Non-catchable error | *(never reached — error propagates)* | |

The `error` field is a plain string (the error message), not an error object. This keeps destructured handling simple — test with `if (error)`.

## fail

Creates a catchable error with a message string. The error has class `value` and code `USER-0001`:

```parsley
fail("something went wrong")
```

`fail` takes exactly one argument, which must be a string. Combine with `check` for validation-style guards:

```parsley
let divide = fn(a, b) {
    check b != 0 else fail("division by zero")
    a / b
}

let {result, error} = try divide(10, 0)
error                            // "division by zero"

let {result, error} = try divide(10, 2)
result                           // 5
```

## Error Classes

Every error belongs to a class. The class determines whether `try` can catch it.

### Catchable (external factors)

| Class | Typical cause |
|---|---|
| `io` | File not found, permission denied, read/write failure |
| `network` | HTTP error, connection refused, timeout |
| `database` | Query failed, connection lost |
| `format` | Invalid JSON/CSV/markdown, parse failure |
| `value` | Invalid value, `fail()` errors |
| `security` | Access denied by security policy |

### Non-catchable (logic bugs)

| Class | Typical cause |
|---|---|
| `type` | Type mismatch in operation |
| `arity` | Wrong number of arguments to builtin/method |
| `undefined` | Variable or method not found |
| `index` | Array/string index out of bounds |
| `operator` | Invalid operator for given types |
| `parse` | Syntax error |
| `state` | Invalid state (e.g., using closed connection) |
| `import` | Module not found or load failure |

Non-catchable errors propagate straight through `try` and halt execution:

```parsley
let bad = fn() { notDefined }
// try bad() — still halts with "Identifier not found: notDefined"
```

The rationale: a type mismatch or undefined variable is a bug in your code, not a runtime condition you should silently recover from.

## Error Codes

Errors carry a code like `TYPE-0001` or `IO-0003` that identifies the specific error. Codes follow the pattern `PREFIX-NNNN`. See the [Error Codes Reference](../../error-codes.md) for the full catalog.

## Error Prevention

Rather than catching errors after the fact, Parsley provides several tools to prevent them.

### Optional Access — `[?n]`

Returns `null` instead of an index error when an index is out of bounds:

```parsley
let items = ["a", "b", "c"]
items[0]                         // "a"
items[?99]                       // null (no error)
items[99]                        // Error: index 99 out of bounds
```

Works on arrays, strings, and tables.

### Null Coalescing — `??`

Provides a default when a value is `null`:

```parsley
let name = null
name ?? "anonymous"              // "anonymous"
```

Combine with optional access for safe lookups with defaults:

```parsley
let items = ["a", "b"]
items[?5] ?? "missing"           // "missing"
```

> ⚠️ `??` only checks for `null`, not other falsy values. `0 ?? 10` returns `0`, not `10`.

### Membership Testing — `in`

Check before accessing:

```parsley
let d = {name: "Alice", age: 30}
if ("email" in d) { d["email"] } else { "no email" }
// "no email"
```

`in` is null-safe — `x in null` returns `false`.

### check Guards

Validate preconditions at the top of a function. Failing a `check` exits the function immediately with the `else` value:

```parsley
let process = fn(items) {
    check items.length() > 0 else "empty list"
    check items.length() < 1000 else "too many items"
    // ... process items ...
}
```

When combined with `fail`, the guard produces a catchable error instead of a plain return value:

```parsley
let process = fn(items) {
    check items.length() > 0 else fail("empty list")
    // ... if called via try, caller gets {result: null, error: "empty list"}
}
```

## Patterns

### Wrap-and-check

The most common pattern — call, destructure, branch:

```parsley
let {result, error} = try loadFile("config.json")
if (error) {
    log("Using defaults: " + error)
    let config = defaults
} else {
    let config = result
}
```

### Chain with ??

When you just need a fallback value:

```parsley
let {result} = try loadFile("config.json")
let config = result ?? defaults
```

### Validate-then-act

Use `check` + `fail` to validate inputs, `try` at the call site:

```parsley
let createUser = fn(name, email) {
    check name != "" else fail("name required")
    check email != "" else fail("email required")
    {name: name, email: email}
}

let {result, error} = try createUser("", "a@b.com")
error                            // "name required"
```

## Key Differences from Other Languages

- **No try/catch/finally blocks** — `try` is an expression that returns `{result, error}`.
- **Not all errors are catchable** — logic bugs (type, arity, undefined) always halt. Only external/runtime errors are catchable.
- **`fail()` is the only way to throw** — and it only creates `value`-class (catchable) errors.
- **Error result is a plain string** — not an error object. No stack traces or error hierarchies.
- **Prevention over recovery** — optional access `[?]`, null coalescing `??`, `in` checks, and `check` guards handle most cases without `try`.

## See Also

- [Control Flow](control-flow.md) — `check`, `try`, and `fail` syntax overview
- [Booleans & Null](../builtins/booleans.md) — truthiness rules, `??` null coalescing
- [Operators](operators.md) — `??`, `in`, `[?]` optional access
- [@std/api](../stdlib/api.md) — HTTP error helpers (`notFound`, `badRequest`, etc.)
- [Security Model](../features/security.md) — security errors and policy enforcement