# Unified Error Model

**Status:** Approved
**Date:** 2026-02-08
**Purpose:** Design summary for unifying Parsley's error systems around a single `{result, error}` pattern

---

## Problem

Parsley has three independent error systems that can't talk to each other:

| System | Go Type | Produces | Caught by `try`? | Server-aware? |
|---|---|---|---|---|
| `fail("msg")` | `*Error` | String message | ✅ (message only) | ❌ |
| `api.notFound("msg")` | `*APIError` | HTTP error object | ❌ | ✅ |
| `record.validate()` | `*RecordError` | Per-field error data | N/A | ❌ |

Plus `check ... else value` — a control-flow guard that early-returns whatever you give it, inheriting the incompatibilities of whichever error type you choose.

### What Goes Wrong

**1. `try` destroys structure.** When `try` catches a `fail()`, it keeps only the message string. The error code, class, hints, and data are discarded.

**2. `try` can't catch API errors.** `APIError` is a separate Go type. `try` only checks for `*Error`, so API errors land in `result` with `error: null`.

**3. `fail()` has no error codes.** Every `fail()` produces the hardcoded code `"USER-0001"`. Callers can't distinguish different failure modes.

**4. `check` forces a context choice.** A function using `check ... else fail(...)` works with `try` but not the server. A function using `check ... else api.badRequest(...)` works with the server but not `try`. You can't write a reusable function that works in both contexts.

**5. No bridge from validation to response.** Converting `record.errors()` to an API error response requires manual string formatting, losing the structured field/code/message data.

---

## Design Principle

**`{result, error}` where one is always null.**

This pattern is already the core of Parsley's error handling. It's simple, consistent, and idiomatic. The proposal doesn't change this pattern — it refines what `error` contains when it's not null.

---

## Proposal

### `error` is always a dictionary

Today: `{result: value|null, error: string|null}`
Proposed: `{result: value|null, error: dict|null}`

When `error` is not null, it is a dictionary with at least a `message` field. It may contain additional fields depending on context.

The guaranteed shape:

```parsley
{
    message: "Human-readable error description"   // always present
    code: "ERROR_CODE"                            // optional
    status: 400                                   // optional (HTTP status)
    // ...any additional fields
}
```

### `fail()` accepts a string or dictionary

```parsley
// Simple — wraps in {message: "..."}
fail("something went wrong")

// Structured — passed through as-is (must include message)
fail({code: "NOT_FOUND", message: "User not found"})

// HTTP-aware — status field recognized by server
fail({code: "NOT_FOUND", message: "User not found", status: 404})
```

When `fail()` receives a string, it wraps it: `{message: "something went wrong"}`.
When it receives a dictionary, it uses it directly (must contain `message`).

### `try` preserves the full error dictionary

```parsley
let {result, error} = try riskyFn()
if (error) {
    error.message       // always works
    error.code          // available if the fail provided one
    error.status        // available if the fail provided one
}
```

The `if (error)` guard still works — dicts are truthy, null is falsy.

### `check` works everywhere

Because `fail()` now produces errors the server can handle (via `status`) and `try` can catch (as a dict), `check` no longer forces a context choice:

```parsley
// Works in handlers (server sees status: 400) AND with try (caller sees the dict)
let requireEmail = fn(data) {
    check data.email else fail({status: 400, message: "Email required"})
    data
}
```

### `api.*` helpers become sugar

The `api.*` helpers remain as convenient shortcuts but produce the same unified error type:

```parsley
api.notFound("User not found")
// equivalent to:
fail({code: "HTTP-404", message: "User not found", status: 404})
```

### Validation errors remain separate (with a bridge)

Schema validation errors are **data about fields**, not function failures. They stay as they are — `record.errors()`, `record.isValid()`, `record.errorList()`.

A bridge method converts validation state into a fail-able error when needed:

```parsley
let record = data.as(UserSchema).validate()
if (!record.isValid()) {
    fail({
        status: 400,
        code: "VALIDATION",
        message: "Validation failed",
        fields: record.errorList()
    })
}
```

Whether this warrants a convenience method (e.g., `record.failIfInvalid()`) is a separate decision.

---

## The Unified Pattern

Every error interaction uses the same shape:

```parsley
// Producing errors
fail("oops")                                          // {message: "oops"}
fail({code: "NO_STOCK", message: "Out of stock"})     // structured
fail({status: 404, message: "Not found"})              // HTTP-aware

// Guarding
check condition else fail("why it failed")
check condition else fail({status: 400, message: "why it failed"})

// Catching
let {result, error} = try anything()
if (error) {
    error.message       // always present
    error.code          // if provided
    error.status        // if provided
}
```

---

## Breaking Changes

One breaking change: code that concatenates `error` as a string.

```parsley
// Before (error is a string):
"Failed: " + error

// After (error is a dict):
"Failed: " + error.message
```

This is the only migration required. All other patterns are unchanged:
- `if (error)` — still works (dict is truthy)
- `{result, error}` destructuring — still works
- One-is-always-null invariant — still holds
- `fail("string")` — still works (wrapped automatically)

---

## Implementation Scope

### Changes Required

| Component | Change |
|---|---|
| `fail()` builtin | Accept string or dict. String wraps to `{message: str}`. Dict requires `message` field. |
| `evalTryExpression` | Put full error dict in `error` field instead of just `err.Message` string. |
| `try` + `APIError` | Catch `*APIError` the same as `*Error` — convert to dict with `{code, message, status}`. |
| Server response handler | Check `*Error` for a `status` field in Data and use it as HTTP status code. |
| `api.*` helpers | Optionally refactor to produce `*Error` with status instead of separate `*APIError` type. Or keep as-is and let `try` handle conversion. |

### What Doesn't Change

- `check` syntax and semantics (it's already agnostic)
- Schema validation (`record.validate()`, `.errors()`, `.isValid()`)
- Internal error classes (`ClassType`, `ClassIO`, etc.) and catchability rules
- Error catalog codes (`TYPE-0001`, `IO-0003`, etc.)

---

## Resolved Questions

1. **Should `api.*` helpers remain or become redundant?** **Keep them as sugar.** `api.notFound("msg")` is more readable than `fail({status: 404, code: "HTTP-404", message: "msg"})`. Internally they change from returning `*APIError` to delegating to `fail()` with pre-filled `status`/`code`/`message` fields. The maintenance cost is near-zero since they just wrap `fail()`. User-facing API is unchanged.

2. **Validation bridge convenience.** **Add `record.failIfInvalid()`.** The existing manual `validate()` → `isValid()` → `errorList()` pattern is unchanged and remains the way to do custom error handling. `failIfInvalid()` is additive sugar for the common case: it returns the record if valid (enabling chaining), or calls `fail({status: 400, code: "VALIDATION", message: "Validation failed", fields: record.errorList()})` if invalid.

3. **String coercion.** **Yes.** `"" + error` produces `error.message` via Parsley's string coercion rules. This reduces the migration burden for existing code that concatenates the error value to near-zero.

4. **Error dict immutability.** **Regular mutable dictionary.** Parsley does not currently have immutable types. The error dict is a plain dictionary — no special type needed.

---

## Summary

The `{result, error}` pattern doesn't change. The improvement is making `error` always a dictionary with at least `{message}`, so that:

- Structure is preserved, not discarded
- One function can work in both handler and library contexts
- Error codes are available when needed, optional when not
- The `if (error)` idiom works exactly as before