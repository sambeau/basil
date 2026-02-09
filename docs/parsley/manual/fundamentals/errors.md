---
id: man-pars-errors
title: Error Handling
system: parsley
type: fundamentals
name: errors
created: 2026-02-05
version: 0.3.0
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
  - error dict
  - failIfInvalid
  - structured errors
---

# Error Handling

Parsley divides errors into two camps: **catchable** errors from external factors (file not found, network timeout, bad input) and **non-catchable** errors from logic bugs (type mismatch, wrong argument count, undefined variable). Only catchable errors can be intercepted with `try` — logic bugs halt execution immediately.

There are no try/catch blocks. Instead, `try` wraps a single function or method call and returns a result dictionary.

## try

`try` calls a function or method and catches recoverable errors. It always returns a dictionary with `result` and `error` keys:

```parsley
let risky = fn() { fail("oops") }
try risky()                      // {result: null, error: {message: "oops", code: "USER-0001"}}

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
| Catchable error | `null` | Dictionary with at least `message` and `code` |
| Non-catchable error | *(never reached — error propagates)* | |

The `error` field is a dictionary with at least a `message` key (string). Most errors also include a `code` key. API errors include a `status` key with the HTTP status code.

To test whether an error occurred, use `if (error)` — dictionaries are truthy, `null` is falsy.

### Accessing Error Fields

```parsley
let {result, error} = try riskyOperation()
if (error) {
    error.message                // "Something went wrong" (always present)
    error.code                   // "USER-0001" (usually present)
    error.status                 // 404 (present on API errors)
}
```

### String Coercion

Error dictionaries (any dictionary with a `message` key) automatically coerce to the message string when used in string context:

```parsley
let {error} = try riskyOperation()
if (error) {
    log("Failed: " + error)     // uses error.message automatically
    // equivalent to:
    log("Failed: " + error.message)
}
```

This keeps code concise and provides backward compatibility. The coercion applies to any plain dictionary with a `message` key — it does not affect special typed dictionaries (paths, URLs, datetimes, etc.).

## fail

Creates a catchable error. Accepts either a string message or a dictionary with structured error data.

### String Form

The simplest form — pass a message string:

```parsley
fail("something went wrong")
```

This creates an error with class `value`, code `USER-0001`, and a `UserDict` of `{message: "something went wrong", code: "USER-0001"}`.

### Dictionary Form

Pass a dictionary for structured errors with custom fields. The dictionary **must** contain a `message` key with a string value:

```parsley
fail({
    message: "Out of stock",
    code: "NO_STOCK",
    status: 400,
    product: "Widget"
})
```

The `code` field is optional — if omitted, it defaults to `USER-0001`. Any additional keys are preserved and available when the error is caught by `try`.

### Validation Rules

- **String argument**: always valid
- **Dictionary argument**: must have a `message` key with a string value
- **Any other type** (integer, boolean, array, etc.): produces a TYPE-0005 error

```parsley
fail("ok")                       // ✅ Valid
fail({message: "ok"})            // ✅ Valid
fail({message: "ok", status: 400}) // ✅ Valid
fail({code: "X"})                // ❌ TYPE-0005 — missing message key
fail({message: 123})             // ❌ TYPE-0005 — message must be string
fail(123)                        // ❌ TYPE-0005 — must be string or dict
```

### Using fail with check

Combine with `check` for validation-style guards:

```parsley
let divide = fn(a, b) {
    check b != 0 else fail("division by zero")
    a / b
}

let {result, error} = try divide(10, 0)
error.message                    // "division by zero"

let {result, error} = try divide(10, 2)
result                           // 5
```

### Structured API Errors

For API handlers, include `status` and `code` for proper HTTP error responses:

```parsley
let getUser = fn(id) {
    let user = db.find(id)
    check user else fail({
        message: "User not found",
        code: "USER_NOT_FOUND",
        status: 404
    })
    user
}
```

When this error reaches the server dispatch, the `status` field determines the HTTP status code and the full error dictionary is returned as JSON in the `{error: {...}}` envelope.

## Error Classes

Every error belongs to a class. The class determines whether `try` can catch it.

### Catchable (external factors)

| Class | Typical cause |
|---|---|
| `io` | File not found, permission denied, read/write failure |
| `network` | HTTP error, connection refused, timeout |
| `database` | Query failed, connection lost |
| `format` | Invalid JSON/CSV/markdown, parse failure |
| `value` | Invalid value, `fail()` errors, `api.*` errors |
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

## API Error Helpers

The `@std/api` module provides helpers that create structured errors with appropriate HTTP status codes. These are catchable `value`-class errors:

```parsley
let api = import @std/api

api.notFound("User not found")      // {code: "HTTP-404", message: "User not found", status: 404}
api.badRequest("Invalid input")     // {code: "HTTP-400", message: "Invalid input", status: 400}
api.forbidden("Access denied")      // {code: "HTTP-403", message: "Access denied", status: 403}
api.unauthorized("Not logged in")   // {code: "HTTP-401", message: "Not logged in", status: 401}
api.conflict("Already exists")      // {code: "HTTP-409", message: "Already exists", status: 409}
api.serverError("Internal error")   // {code: "HTTP-500", message: "Internal error", status: 500}
```

Each helper accepts an optional message string. If omitted, a default message is used. API errors are catchable with `try`:

```parsley
let {result, error} = try fn() { api.notFound("User not found") }()
error.message                    // "User not found"
error.code                       // "HTTP-404"
error.status                     // 404
```

When an API error reaches the server dispatch (not caught by `try`), the server writes an HTTP response using the `status` field and wraps the error dict in `{error: {...}}` JSON.

## Validation Bridge — failIfInvalid()

The `failIfInvalid()` method on validated records converts validation errors into a structured catchable error, bridging schema validation with the unified error model.

```parsley
@schema User { name: string(required), email: email(required) }

let user = User({name: null, email: "bad"}).validate()

// Without failIfInvalid — manual checking:
if (!user.isValid()) {
    fail({message: "Validation failed", status: 400, fields: user.errorList()})
}

// With failIfInvalid — one-liner:
user.failIfInvalid()
```

### Behavior

| Record state | Return value |
|---|---|
| Not yet validated | The record (no-op) |
| Valid (no errors) | The record (enables chaining) |
| Invalid (has errors) | Catchable error with structured dict |

### Error Shape

When validation fails, `failIfInvalid()` returns an error with:

```parsley
{
    status: 400,
    code: "VALIDATION",
    message: "Validation failed",
    fields: [
        {field: "name", code: "REQUIRED", message: "Name is required"},
        {field: "email", code: "FORMAT", message: "Email is not a valid email"}
    ]
}
```

### Chaining

Because `failIfInvalid()` returns the record when valid, you can chain it into processing pipelines:

```parsley
let user = User(formData).validate().failIfInvalid()
// If we get here, user is valid — proceed with confidence
db.insert(user)
```

### Catching Validation Errors

```parsley
let {result, error} = try fn() {
    User(formData).validate().failIfInvalid()
}()

if (error) {
    error.code                   // "VALIDATION"
    error.status                 // 400
    error.fields                 // array of field errors
}
```

The existing validation methods (`isValid()`, `errorList()`, `hasError()`, `error()`, `errorCode()`) continue to work unchanged. `failIfInvalid()` is a convenience that composes them into a single catchable error.

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
    // ... if called via try, caller gets {result: null, error: {message: "empty list", ...}}
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
error.message                    // "name required"
```

### Structured API Error Handling

Return rich error data from API handlers:

```parsley
let api = import @std/api

export post = fn(req) {
    let user = User(req.body).validate().failIfInvalid()
    let saved = Users.insert(user)
    {user: saved}
}
// If validation fails → HTTP 400 with {error: {code: "VALIDATION", message: "...", fields: [...]}}
// If insert fails → error propagates to server dispatch
```

### Inspecting Error Details

When you need to branch on error type:

```parsley
let {result, error} = try fetchData()
if (error) {
    if (error.code == "HTTP-404") {
        // Handle not found
    } else if (error.status >= 500) {
        // Handle server error
    } else {
        // Handle other errors
        log("Unexpected: " + error.message)
    }
}
```

## Key Differences from Other Languages

- **No try/catch/finally blocks** — `try` is an expression that returns `{result, error}`.
- **Not all errors are catchable** — logic bugs (type, arity, undefined) always halt. Only external/runtime errors are catchable.
- **`fail()` is the only way to throw** — and it only creates `value`-class (catchable) errors.
- **Error is a dictionary** — the `error` slot from `try` is a dictionary with at least `message` and `code` keys, not a plain string. Use `error.message` to get the message.
- **String coercion** — `"" + error` yields `error.message`, so string concatenation works naturally.
- **Prevention over recovery** — optional access `[?]`, null coalescing `??`, `in` checks, and `check` guards handle most cases without `try`.

## See Also

- [Control Flow](control-flow.md) — `check`, `try`, and `fail` syntax overview
- [Booleans & Null](../builtins/booleans.md) — truthiness rules, `??` null coalescing
- [Operators](operators.md) — `??`, `in`, `[?]` optional access
- [@std/api](../stdlib/api.md) — HTTP error helpers (`notFound`, `badRequest`, etc.)
- [Schemas](../builtins/schema.md) — `failIfInvalid()` and record validation
- [Security Model](../features/security.md) — security errors and policy enforcement