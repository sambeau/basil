---
id: FEAT-105
title: "Unified Error Model"
status: draft
priority: high
created: 2026-02-08
author: "@human"
---

# FEAT-105: Unified Error Model

## Summary

Unify Parsley's three independent error systems (`fail()`, `api.*`, schema validation) around a single `{result, error}` pattern where `error` is always a dictionary with at least a `message` field. This lets functions work in both handler and library contexts without forcing a choice between `fail()` (catchable by `try`, invisible to server) and `api.*` (visible to server, invisible to `try`).

## User Story

As a Parsley developer, I want `fail()` errors, API errors, and validation errors to share a common shape so that I can write reusable functions that work with both `try` and the HTTP server without choosing between incompatible error types.

## Acceptance Criteria

### Core: `fail()` accepts string or dict
- [ ] `fail("msg")` produces an `*Error` with `UserDict: {message: "msg"}`
- [ ] `fail({message: "msg"})` produces an `*Error` with `UserDict` set to the provided dict
- [ ] `fail({message: "msg", code: "X", status: 400})` preserves all fields in `UserDict`
- [ ] `fail({})` (no `message` key) produces a type error
- [ ] `fail(123)` (not string or dict) produces a type error

### Core: `try` preserves full error dict
- [ ] `let {result, error} = try fn() { fail("msg") }()` → `error` is `{message: "msg"}`
- [ ] `let {result, error} = try fn() { fail({code: "X", message: "msg", status: 400}) }()` → `error` is `{code: "X", message: "msg", status: 400}`
- [ ] `error.message` always works when `error` is not null
- [ ] `error.code` works when the original `fail()` included `code`
- [ ] `error.status` works when the original `fail()` included `status`
- [ ] `if (error)` still works (dict is truthy, null is falsy)

### Core: `try` catches API errors
- [ ] `let {result, error} = try fn() { api.notFound("msg") }()` → `error` is `{code: "HTTP-404", message: "msg", status: 404}`
- [ ] Same for `api.badRequest`, `api.forbidden`, `api.unauthorized`, `api.conflict`, `api.serverError`

### Core: Server recognizes `status` in error dicts
- [ ] Handler returning `fail({status: 404, message: "Not found"})` sends HTTP 404 with JSON error body
- [ ] Handler returning `fail({status: 400, message: "Bad input", fields: [...]})` sends HTTP 400 with full dict as JSON
- [ ] Handler returning `fail("msg")` (no status) sends HTTP 500 (backward-compatible fallback)

### Sugar: `api.*` helpers produce unified errors
- [ ] `api.notFound("msg")` internally produces the same `*Error` type as `fail({...})`
- [ ] `api.notFound("msg")` is equivalent to `fail({code: "HTTP-404", message: "msg", status: 404})`
- [ ] All six helpers: `notFound` (404), `badRequest` (400), `forbidden` (403), `unauthorized` (401), `conflict` (409), `serverError` (500)
- [ ] User-facing API is unchanged — same function names, same arguments

### Bridge: `record.failIfInvalid()`
- [ ] Valid record: `record.validate().failIfInvalid()` returns the record (enables chaining)
- [ ] Invalid record: `record.validate().failIfInvalid()` calls `fail({status: 400, code: "VALIDATION", message: "Validation failed", fields: record.errorList()})`
- [ ] Existing validation API unchanged: `validate()`, `isValid()`, `errors()`, `errorList()`, `hasError()`, `error()`, `errorCode()`, `withError()` all work exactly as before

### String coercion
- [ ] `"Error: " + error` produces `"Error: <message>"` when `error` is an error dict (dict with `message` key)
- [ ] General dictionary string coercion unchanged for dicts without a `message` key

### Backward compatibility
- [ ] `fail("msg")` still works (string wrapped to dict)
- [ ] `{result, error}` destructuring still works
- [ ] `if (error)` guard still works
- [ ] `check ... else fail(...)` still works in both contexts
- [ ] Non-catchable errors (Type, Arity, Undefined, etc.) propagate unchanged

## Design Decisions

- **`api.*` helpers remain as sugar.** `api.notFound("msg")` is more readable than `fail({status: 404, code: "HTTP-404", message: "msg"})`. Internally they delegate to `fail()`. Near-zero maintenance cost.
- **`record.failIfInvalid()` is additive.** The manual `validate()` → `isValid()` → `errorList()` pattern stays unchanged for custom error handling. `failIfInvalid()` is sugar for the common case.
- **String coercion: yes.** `"" + error` produces `error.message` for dicts that have a `message` key. Reduces migration burden to near-zero.
- **Error dict is a regular mutable dictionary.** Parsley doesn't have immutable types. No special type needed.
- **`*APIError` Go type is retired.** `api.*` helpers produce `*Error` with a `UserDict` field. Server dispatch switches on `*Error` and reads `status` from `UserDict`. The `*APIError` type can be removed after migration (or kept temporarily with a conversion path in `try`).
- **Error class stays `ClassValue`.** User-raised errors (both `fail()` and `api.*`) use `ClassValue`, which is catchable by `try`. Internal error classes (`ClassType`, `ClassIO`, etc.) are unchanged.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components

| File | Change |
|---|---|
| `pkg/parsley/evaluator/evaluator.go` | Add `UserDict *Dictionary` field to `Error` struct. Update `fail()` builtin to accept string or dict. |
| `pkg/parsley/evaluator/eval_control_flow.go` | Update `evalTryExpression` to put `err.UserDict` in the `error` slot instead of `&String{Value: err.Message}`. |
| `pkg/parsley/evaluator/stdlib_api.go` | Change `api.*` helpers to return `*Error` with `UserDict` instead of `*APIError`. Optionally deprecate/remove `APIError` type. |
| `pkg/parsley/evaluator/eval_string_conversions.go` | In `objectToTemplateString`, add early check for dicts with a `message` key — return the message string. |
| `pkg/parsley/evaluator/methods_record.go` | Add `failIfInvalid` to `recordMethods` list and implement `recordFailIfInvalid()`. |
| `server/api.go` | Update `dispatchModule` and `writeAPIResponse` to handle `*Error` with `UserDict.status` as HTTP status. Remove or reduce `*APIError` handling. |
| `pkg/parsley/evaluator/introspect.go` | Update `fail` introspection to reflect string\|dict parameter. |

### Error Struct Change

Current `Error` struct (`evaluator.go:224`):

```go
type Error struct {
    Message string
    Line    int
    Column  int
    Class   ErrorClass
    Code    string
    Hints   []string
    File    string
    Data    map[string]any  // template variables — NOT the user dict
}
```

Add one field:

```go
type Error struct {
    Message  string
    Line     int
    Column   int
    Class    ErrorClass
    Code     string
    Hints    []string
    File     string
    Data     map[string]any  // template variables (internal)
    UserDict *Dictionary     // NEW: structured error dict from fail()
}
```

`Data` (the existing `map[string]any`) is used internally for error catalog template rendering. `UserDict` is the Parsley-visible dictionary that `try` exposes.

### `fail()` Builtin Change

Current (`evaluator.go:3607`):

```go
"fail": {
    Fn: func(args ...Object) Object {
        if len(args) != 1 {
            return newArityError("fail", len(args), 1)
        }
        msg, ok := args[0].(*String)
        if !ok {
            return newTypeError("TYPE-0005", "fail", "a string", args[0].Type())
        }
        return &Error{
            Class:   ClassValue,
            Code:    "USER-0001",
            Message: msg.Value,
        }
    },
},
```

New:

```go
"fail": {
    Fn: func(args ...Object) Object {
        if len(args) != 1 {
            return newArityError("fail", len(args), 1)
        }
        switch arg := args[0].(type) {
        case *String:
            dict := &Dictionary{
                Pairs:    map[string]ast.Expression{
                    "message": &ast.ObjectLiteralExpression{Obj: arg},
                },
                KeyOrder: []string{"message"},
                Env:      nil,
            }
            return &Error{
                Class:    ClassValue,
                Code:     "USER-0001",
                Message:  arg.Value,
                UserDict: dict,
            }
        case *Dictionary:
            // Must contain "message" key
            msgExpr, ok := arg.Pairs["message"]
            if !ok {
                return newTypeError("TYPE-0005", "fail",
                    "a dictionary with a 'message' key", "dictionary without 'message'")
            }
            msg := Eval(msgExpr, arg.Env)
            msgStr, ok := msg.(*String)
            if !ok {
                return newTypeError("TYPE-0005", "fail",
                    "message to be a string", msg.Type())
            }
            // Extract optional code
            code := "USER-0001"
            if codeExpr, has := arg.Pairs["code"]; has {
                if codeObj, ok := Eval(codeExpr, arg.Env).(*String); ok {
                    code = codeObj.Value
                }
            }
            return &Error{
                Class:    ClassValue,
                Code:     code,
                Message:  msgStr.Value,
                UserDict: arg,
            }
        default:
            return newTypeError("TYPE-0005", "fail",
                "a string or dictionary", args[0].Type())
        }
    },
},
```

### `evalTryExpression` Change

Current (`eval_control_flow.go:358`):

```go
if err, ok := result.(*Error); ok {
    perrClass := perrors.ErrorClass(err.Class)
    if perrClass.IsCatchable() {
        pairs := make(map[string]ast.Expression)
        pairs["result"] = &ast.ObjectLiteralExpression{Obj: NULL}
        pairs["error"] = &ast.ObjectLiteralExpression{Obj: &String{Value: err.Message}}
        return &Dictionary{
            Pairs:    pairs,
            KeyOrder: []string{"result", "error"},
            Env:      env,
        }
    }
    return err
}
```

New:

```go
if err, ok := result.(*Error); ok {
    perrClass := perrors.ErrorClass(err.Class)
    if perrClass.IsCatchable() {
        // Use UserDict if present (structured fail), otherwise wrap message
        var errorObj Object
        if err.UserDict != nil {
            errorObj = err.UserDict
        } else {
            // Internal errors caught by try — wrap message in dict for consistency
            errorObj = &Dictionary{
                Pairs: map[string]ast.Expression{
                    "message": &ast.ObjectLiteralExpression{Obj: &String{Value: err.Message}},
                    "code":    &ast.ObjectLiteralExpression{Obj: &String{Value: err.Code}},
                },
                KeyOrder: []string{"message", "code"},
                Env:      env,
            }
        }
        pairs := make(map[string]ast.Expression)
        pairs["result"] = &ast.ObjectLiteralExpression{Obj: NULL}
        pairs["error"] = &ast.ObjectLiteralExpression{Obj: errorObj}
        return &Dictionary{
            Pairs:    pairs,
            KeyOrder: []string{"result", "error"},
            Env:      env,
        }
    }
    return err
}
```

### `api.*` Helper Change

Current (`stdlib_api.go:202`):

```go
func apiNotFound(args ...Object) Object {
    msg := "Not found"
    if len(args) == 1 {
        if str, ok := args[0].(*String); ok {
            msg = str.Value
        }
    }
    return &APIError{Code: "HTTP-404", Message: msg, Status: 404}
}
```

New:

```go
func apiNotFound(args ...Object) Object {
    msg := "Not found"
    if len(args) == 1 {
        if str, ok := args[0].(*String); ok {
            msg = str.Value
        }
    }
    return apiFailError("HTTP-404", msg, 404)
}

// apiFailError builds a unified *Error with UserDict containing code, message, status.
func apiFailError(code, message string, status int) *Error {
    dict := &Dictionary{
        Pairs: map[string]ast.Expression{
            "code":    &ast.ObjectLiteralExpression{Obj: &String{Value: code}},
            "message": &ast.ObjectLiteralExpression{Obj: &String{Value: message}},
            "status":  &ast.ObjectLiteralExpression{Obj: &Integer{Value: int64(status)}},
        },
        KeyOrder: []string{"code", "message", "status"},
    }
    return &Error{
        Class:    ClassValue,
        Code:     code,
        Message:  message,
        UserDict: dict,
    }
}
```

All six `api.*` helpers (`apiNotFound`, `apiBadRequest`, `apiForbidden`, `apiUnauthorized`, `apiConflict`, `apiServerError`) follow the same pattern — change the return to call `apiFailError(code, msg, status)`.

### Server Dispatch Change

Current (`server/api.go:160`):

```go
result := evaluator.CallWithEnv(handler, []evaluator.Object{reqObj}, module.Env)

// Auth wrappers can return APIError directly
if apiErr, ok := result.(*evaluator.APIError); ok {
    h.writeAPIError(w, apiErr)
    return
}

if errObj, ok := result.(*evaluator.Error); ok {
    h.server.logError("runtime error in %s: %s", h.scriptPath, errObj.Inspect())
    http.Error(w, "Internal Server Error", http.StatusInternalServerError)
    return
}
```

New:

```go
result := evaluator.CallWithEnv(handler, []evaluator.Object{reqObj}, module.Env)

if errObj, ok := result.(*evaluator.Error); ok {
    if errObj.UserDict != nil {
        // Structured error — check for status field
        status := http.StatusInternalServerError
        if statusExpr, ok := errObj.UserDict.Pairs["status"]; ok {
            if iv, ok := evaluator.Eval(statusExpr, errObj.UserDict.Env).(*evaluator.Integer); ok {
                status = int(iv.Value)
            }
        }
        w.Header().Set("Content-Type", "application/json; charset=utf-8")
        w.WriteHeader(status)
        // Write {error: <UserDict>} as JSON body
        h.writeJSONDict(w, wrapErrorDict(errObj.UserDict))
        return
    }
    // Unstructured *Error (internal runtime error) — 500
    h.server.logError("runtime error in %s: %s", h.scriptPath, errObj.Inspect())
    http.Error(w, "Internal Server Error", http.StatusInternalServerError)
    return
}
```

The `wrapErrorDict` helper wraps the user dict in `{error: <dict>}` to match the existing `APIError.ToDict()` response shape.

Internal server code that constructs `APIError` directly (auth enforcement, rate limiting) should be updated to use `apiFailError()` or construct `*Error` with `UserDict`.

### String Coercion Change

Current (`eval_string_conversions.go:39`):

```go
case *Dictionary:
    // Check for special dictionary types
    if isPathDict(obj) {
        return pathDictToString(obj)
    }
    // ...other special types...
    return obj.Inspect()
```

New — add one check at the top of the Dictionary case, before the special-type checks:

```go
case *Dictionary:
    // Error dicts coerce to their message
    if msgExpr, ok := obj.Pairs["message"]; ok {
        if !isPathDict(obj) && !isUrlDict(obj) && !isTagDict(obj) &&
           !isDatetimeDict(obj) && !isDurationDict(obj) && !isRegexDict(obj) &&
           !isFileDict(obj) && !isDirDict(obj) && !isRequestDict(obj) {
            msgObj := Eval(msgExpr, obj.Env)
            if msgStr, ok := msgObj.(*String); ok {
                return msgStr.Value
            }
        }
    }
    // Check for special dictionary types
    if isPathDict(obj) {
        // ...existing code unchanged...
```

This means any plain dictionary with a `message` key coerces to its message string. This is intentionally general — it's useful beyond just error dicts, and it's a natural convention.

### `record.failIfInvalid()` Implementation

Add to `recordMethods` slice (`methods_record.go:13`):

```go
var recordMethods = []string{
    "validate", "update", "errors", "error", "errorCode", "errorList",
    "isValid", "hasError", "schema", "data", "keys", "withError",
    "title", "placeholder", "meta", "enumValues", "format", "toJSON",
    "failIfInvalid",
}
```

Add case in `evalRecordMethod`:

```go
case "failIfInvalid":
    return recordFailIfInvalid(record, args)
```

Implementation:

```go
func recordFailIfInvalid(record *Record, args []Object) Object {
    if len(args) != 0 {
        return newArityError("failIfInvalid", len(args), 0)
    }

    // Not yet validated — return the record (no-op)
    if !record.Validated {
        return record
    }

    // Valid — return the record for chaining
    if len(record.Errors) == 0 {
        return record
    }

    // Invalid — build error list and fail
    errorList := recordErrorList(record, nil)
    dict := &Dictionary{
        Pairs: map[string]ast.Expression{
            "status":  &ast.ObjectLiteralExpression{Obj: &Integer{Value: 400}},
            "code":    &ast.ObjectLiteralExpression{Obj: &String{Value: "VALIDATION"}},
            "message": &ast.ObjectLiteralExpression{Obj: &String{Value: "Validation failed"}},
            "fields":  &ast.ObjectLiteralExpression{Obj: errorList},
        },
        KeyOrder: []string{"status", "code", "message", "fields"},
    }
    return &Error{
        Class:    ClassValue,
        Code:     "VALIDATION",
        Message:  "Validation failed",
        UserDict: dict,
    }
}
```

### `*APIError` Retirement Plan

Phase 1 (this feature): `api.*` helpers return `*Error` with `UserDict`. Server dispatch handles `*Error` with `UserDict.status`. Internal server code (auth, rate limit) updated to use `apiFailError()` or equivalent.

Phase 2 (cleanup): Remove `APIError` struct, `ToDict()`, `writeAPIError()`, and `API_ERROR_OBJ` constant. Update any remaining references.

During Phase 1, the `*APIError` type switch in `writeAPIResponse` can remain as a fallback for any code paths that still construct `APIError` directly. This makes migration safe.

### Dependencies

- Depends on: None
- Blocks: None

### Edge Cases & Constraints

1. **`fail()` with dict missing `message`** — Produces a type error (`TYPE-0005`). The `message` key is required.
2. **`fail()` with dict where `message` is not a string** — Produces a type error.
3. **`try` catching internal errors (IO, Network, etc.)** — These don't have `UserDict`. The `evalTryExpression` change wraps their `Message` and `Code` in a dict for consistency, so callers always get a dict in the `error` slot.
4. **Non-catchable errors** — `ClassType`, `ClassArity`, `ClassUndefined`, etc. still propagate unchanged. `try` doesn't catch them. No change.
5. **`check ... else fail({...})`** — Works naturally. `check` evaluates the `else` expression and wraps it in `CheckExit`. The `fail()` call produces an `*Error`, which propagates up as a function return value. The server or `try` then handles it.
6. **Handler returning `fail("msg")` (no status)** — Server sees `*Error` with `UserDict` containing only `{message: "msg"}` (no `status` key). Falls back to HTTP 500. This matches current behavior where `*Error` returns 500.
7. **String coercion on non-error dicts** — A dict like `{message: "hello", name: "world"}` will coerce to `"hello"` in string concatenation. This is intentional and useful, but should be documented. Special dict types (path, url, tag, datetime, etc.) are excluded — they keep their existing coercion.
8. **`record.failIfInvalid()` on un-validated record** — Returns the record unchanged (no-op). The record must be validated first.

## Test Plan

### T1: `fail()` with string (backward compat)

```parsley
let {result, error} = try fn() { fail("oops") }()
error.message   // "oops"
result          // null
```

### T2: `fail()` with dict

```parsley
let {result, error} = try fn() { fail({code: "NO_STOCK", message: "Out of stock", status: 400}) }()
error.message   // "Out of stock"
error.code      // "NO_STOCK"
error.status    // 400
result          // null
```

### T3: `fail()` with dict missing `message` — error

```parsley
fail({code: "X"})   // TYPE-0005 error
```

### T4: `fail()` with non-string/dict — error

```parsley
fail(123)   // TYPE-0005 error
```

### T5: `try` catches `api.notFound()`

```parsley
let api = import @std/api
let {result, error} = try fn() { api.notFound("User not found") }()
error.message   // "User not found"
error.code      // "HTTP-404"
error.status    // 404
```

### T6: `try` catches all `api.*` helpers

Test each: `badRequest` (400), `forbidden` (403), `unauthorized` (401), `conflict` (409), `serverError` (500).

### T7: `if (error)` guard still works

```parsley
let {result, error} = try fn() { fail("oops") }()
if (error) { "caught: " + error.message } else { "no error" }
// "caught: oops"
```

### T8: String coercion

```parsley
let {result, error} = try fn() { fail("oops") }()
"Error: " + error   // "Error: oops"
```

### T9: String coercion with structured error

```parsley
let {result, error} = try fn() { fail({message: "bad input", status: 400}) }()
"Error: " + error   // "Error: bad input"
```

### T10: `check` with unified error

```parsley
let requireEmail = fn(data) {
    check data.email else fail({status: 400, message: "Email required"})
    data
}
let {result, error} = try fn() { requireEmail({}) }()
error.message   // "Email required"
error.status    // 400
```

### T11: `record.failIfInvalid()` — valid record

```parsley
let Schema = schema {name: "string"}
let record = {name: "Sam"}.as(Schema).validate().failIfInvalid()
record.data().name   // "Sam"
```

### T12: `record.failIfInvalid()` — invalid record

```parsley
let Schema = schema {name: {type: "string", required: true}}
let {result, error} = try fn() {
    {}.as(Schema).validate().failIfInvalid()
}()
error.code      // "VALIDATION"
error.status    // 400
error.message   // "Validation failed"
error.fields    // [{field: "name", code: "...", message: "..."}]
```

### T13: `record.failIfInvalid()` — existing validation API unchanged

```parsley
let Schema = schema {name: {type: "string", required: true}}
let record = {}.as(Schema).validate()
record.isValid()        // false
record.errorList()      // [{field: "name", code: "...", message: "..."}]
record.hasError("name") // true
```

### T14: Internal catchable errors produce dicts in `try`

```parsley
// IO errors, network errors, etc. should also produce error dicts
let {result, error} = try fn() { import @./nonexistent }()
error.message   // contains error message
error.code      // contains error code (e.g., "IO-0002")
```

### T15: Non-catchable errors still propagate

```parsley
// Type errors should NOT be caught by try
let {result, error} = try fn() { 1 + "a" }()
// This should be a runtime error, not caught
```

### T16: Server handler test — structured error as HTTP response

Integration test: handler returns `fail({status: 404, message: "Not found", code: "HTTP-404"})`, server responds with HTTP 404 and JSON body `{"error": {"code": "HTTP-404", "message": "Not found", "status": 404}}`.

### T17: Server handler test — string fail as HTTP 500

Integration test: handler returns `fail("internal oops")`, server responds with HTTP 500.

### T18: `api.*` helpers in handler context (unchanged behavior)

Integration test: handler returns `api.notFound("msg")`, server responds with HTTP 404 — same as today.

## Migration Guide

### Breaking change

One breaking change: code that concatenates the `error` variable from `try` as a string.

**Before** (`error` was a string):
```parsley
let {result, error} = try riskyFn()
if (error) {
    log("Failed: " + error)
}
```

**After** (`error` is a dict, but string coercion extracts `message`):
```parsley
// This still works thanks to string coercion!
let {result, error} = try riskyFn()
if (error) {
    log("Failed: " + error)    // coerces to error.message
}

// But explicit .message is clearer and recommended:
if (error) {
    log("Failed: " + error.message)
}
```

Because string coercion is included in this feature, most existing code will continue to work without changes. The recommended migration is to use `error.message` explicitly for clarity, but it's not required.

### What doesn't need to change

- `if (error)` guards — still work
- `{result, error}` destructuring — still works
- `fail("string")` calls — still work
- `check ... else fail(...)` — still works
- `api.*` function calls — still work
- `record.validate()`, `.isValid()`, `.errors()`, `.errorList()` — all unchanged

## Implementation Notes

*Added during/after implementation*

## Related

- Design doc: `work/design/unified-error-model.md`
- Plan: `work/plans/PLAN-080-FEAT-105-unified-error-model.md`