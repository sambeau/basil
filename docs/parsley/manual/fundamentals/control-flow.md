---
id: man-pars-control-flow
title: Control Flow
system: parsley
type: fundamentals
name: control-flow
created: 2026-02-05
version: 0.2.0
author: Basil Team
keywords:
  - if
  - else
  - for
  - loop
  - stop
  - skip
  - check
  - try
  - fail
  - control flow
  - iteration
  - guard
---

# Control Flow

Both `if` and `for` are **expressions** in Parsley — they return values. There are no `while` loops, no `switch`/`match` statements, and no ternary operator (because `if` already fills that role).

## if / else

`if` evaluates a condition using truthiness and returns the value of the taken branch. Parentheses around the condition are optional when using braces.

```parsley
// Compact form (parens required without braces)
let status = if (age >= 18) "adult" else "minor"

// Block form (parens optional)
if x > 0 {
    "positive"
} else {
    "non-positive"
}

// Chained
let label = if (x < 0) "negative" else if (x == 0) "zero" else "positive"
```

Without an `else`, a false condition returns `null`:

```parsley
let x = if (false) "yes"         // null
```

> ⚠️ There is no ternary operator (`? :`). Use `if (cond) a else b` — it's an expression that works anywhere a value is expected.

## for

`for` maps over an iterable and returns an array. Null results are automatically filtered out, which gives you map and filter in one construct.

### Map

```parsley
for (x in [1, 2, 3]) { x * 10 }           // [10, 20, 30]
for (n in 1..5) { n * n }                  // [1, 4, 9, 16, 25]
```

Parentheses are optional when using braces:

```parsley
for x in [1, 2, 3] { x * 10 }             // [10, 20, 30]
```

### Filter

Return a value to keep it; let the block evaluate to `null` (by not entering the `if`) to discard it:

```parsley
for (x in [1,2,3,4,5]) { if (x > 3) { x } }  // [4, 5]
```

### Map + Filter

Combine transformation and filtering in one loop:

```parsley
for (x in 1..10) {
    if (x % 2 == 0) { x * 100 }
}
// [200, 400, 600, 800, 1000]
```

### With Index

Two-variable form gives you the index (0-based) and the element:

```parsley
for (i, x in ["a", "b", "c"]) { i + ": " + x }
// ["0: a", "1: b", "2: c"]
```

### Over Dictionaries

Iterate over key-value pairs. One variable gets the value; two variables get the key and value:

```parsley
for (k, v in {a: 1, b: 2, c: 3}) { k + "=" + v }
// ["a=1", "b=2", "c=3"]
```

### Over Strings

Iterates over individual characters:

```parsley
for (ch in "abc") { ch }                   // ["a", "b", "c"]
```

### Over Tables

Iterates over rows (as dictionaries or records). See [Data Model](data-model.md).

## Loop Control

### stop

Exits the loop early (like `break` in other languages). Results collected so far are returned:

```parsley
for (x in [1,2,3,4,5]) {
    if (x == 3) { stop }
    x
}
// [1, 2]
```

### skip

Skips the current iteration (like `continue`). The current element produces no result:

```parsley
for (x in [1,2,3,4,5]) {
    if (x % 2 == 0) { skip }
    x
}
// [1, 3, 5]
```

> ⚠️ `stop` and `skip` can only be used inside `for` loops. Using them elsewhere is a runtime error.

## check

A guard expression for preconditions. If the condition is falsy, the `else` value is returned as the function's result (early exit):

```parsley
let validate = fn(x) {
    check x > 0 else "must be positive"
    check x < 100 else "must be under 100"
    "ok: " + x
}
validate(50)    // "ok: 50"
validate(-1)    // "must be positive"
validate(200)   // "must be under 100"
```

`check` is useful for validation at the top of a function — it avoids deeply nested `if/else` chains. The else value becomes the function's return value immediately.

## try

Wraps a function or method call and catches recoverable errors. Returns a dictionary with `result` and `error` keys:

```parsley
let risky = fn() { fail("oops") }

let outcome = try risky()
// {result: null, error: {message: "oops", code: "USER-0001"}}

let safe = fn() { 42 }
let outcome = try safe()
// {result: 42, error: null}
```

Destructure for clean error handling:

```parsley
let {result, error} = try risky()
if (error) {
    log("Failed: " + error.message)
} else {
    log("Got: " + result)
}
```

> ⚠️ `try` only wraps function and method calls — not arbitrary expressions. `try 1 + 2` is a parse error.

### Catchable vs Non-Catchable Errors

`try` only catches errors from external/runtime factors:

| Catchable | Non-catchable |
|-----------|---------------|
| IO, Network, Database | Type, Arity, Parse |
| Format, Value, Security | Undefined, Operator, State, Import |

Non-catchable errors (logic bugs) propagate through `try` and halt execution. This is intentional — a type mismatch shouldn't be silently swallowed.

## fail

Creates a catchable error. Accepts a string message or a dictionary with structured error data. Useful for signalling application-level error conditions that callers can handle with `try`:

```parsley
// String form — wraps in {message: ..., code: "USER-0001"}
fail("division by zero")

// Dictionary form — must have a string "message" key
fail({message: "Out of stock", code: "NO_STOCK", status: 400})
```

Example with `check` guard:

```parsley
let divide = fn(a, b) {
    check b != 0 else fail("division by zero")
    a / b
}

let {result, error} = try divide(10, 0)
error.message                               // "division by zero"
```

`fail` creates a `Value`-class error, which is always catchable by `try`. The `error` slot in the `try` result is a dictionary — use `error.message` to get the message string. String coercion also works: `"" + error` yields `error.message`.

## Key Differences from Other Languages

- **`if` and `for` are expressions** — they return values. No need for a ternary operator.
- **`for` is map + filter** — null results are automatically excluded. There is no separate `map`/`filter` syntax (though `.map()` and `.filter()` methods exist on arrays).
- **No `while` loop** — use `for` with a range or recursive functions.
- **No `switch`/`match`** — use `if`/`else if` chains.
- **`stop`/`skip` instead of `break`/`continue`** — different keywords, same semantics.
- **`check` replaces guard clauses** — cleaner than `if (!cond) return error`.
- **`try` returns a result dictionary** — not try/catch blocks. Only catches runtime errors, not logic bugs.

## See Also

- [Booleans & Null](../builtins/booleans.md) — truthiness rules used by `if` and `check`
- [Arrays](../builtins/array.md) — `for` loops return arrays; `.map()`, `.filter()` alternatives
- [Operators](operators.md) — `..` range operator, `??` null coalescing
- [Functions](functions.md) — `return`, closures, `check` in function bodies
- [Error Handling](errors.md) — error classes, `try`/`fail` in depth
- [Tags](tags.md) — `if` and `for` inside tag content for dynamic HTML