---
id: PLAN-080
feature: FEAT-105
title: "Implementation Plan for Unified Error Model"
status: complete
created: 2026-02-08
---

# Implementation Plan: FEAT-105 Unified Error Model

## Overview

Unify Parsley's three error systems (`fail()`, `api.*`, schema validation) around a single `{result, error}` pattern where `error` is always a dictionary with at least a `message` field. Implementation is ordered so each task builds on the previous one, with tests at every step.

## Prerequisites

- [x] Design doc approved (`work/design/unified-error-model.md`)
- [x] Feature spec complete (`work/specs/FEAT-105.md`)
- [ ] Feature branch created: `feat/FEAT-105-unified-error-model`

## Tasks

### Task 1: Add `UserDict` field to `Error` struct
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small

Add a `UserDict *Dictionary` field to the `Error` struct. This is a zero-impact structural change — no existing code sets or reads it yet.

Steps:
1. Add `UserDict *Dictionary` field to the `Error` struct (after `Data`)
2. Run `go build ./...` to confirm no compilation issues

Tests:
- `go test ./...` passes (no behavioral change)

---

### Task 2: Update `fail()` to accept string or dictionary
**Files**: `pkg/parsley/evaluator/evaluator.go`, `pkg/parsley/evaluator/introspect.go`
**Estimated effort**: Medium

Replace the current string-only `fail()` builtin with a type-switching implementation that accepts `*String` or `*Dictionary`.

Steps:
1. Replace `fail()` builtin function body with type switch:
   - `*String` → wrap in `{message: str}` dict, set as `UserDict`, keep `Message: str.Value`, `Code: "USER-0001"`
   - `*Dictionary` → validate `message` key exists and is a string, extract optional `code`, set dict as `UserDict`
   - Default → type error `TYPE-0005`
2. Update introspect entry for `fail` — change Params from `["message"]` to `["message_or_dict"]` and Description to reflect string|dict

Tests (new test file `pkg/parsley/tests/unified_error_test.go`):
- T1: `fail("oops")` produces `*Error` with `UserDict.message == "oops"` (backward compat)
- T3: `fail({code: "X"})` without `message` key → TYPE-0005 error
- T4: `fail(123)` → TYPE-0005 error
- T: `fail({message: "m", code: "C", status: 400})` → Error with Code="C", Message="m", UserDict has all three keys

---

### Task 3: Update `evalTryExpression` to preserve error dict
**Files**: `pkg/parsley/evaluator/eval_control_flow.go`
**Estimated effort**: Medium

Change `evalTryExpression` so the `error` slot in the returned `{result, error}` dictionary contains the full error dict instead of just a message string.

Steps:
1. In the `if err, ok := result.(*Error)` catchable branch:
   - If `err.UserDict != nil` → use `err.UserDict` as the error object
   - Else (internal catchable errors like IO/Network) → wrap `err.Message` and `err.Code` in a new dict `{message: ..., code: ...}`
2. Set `pairs["error"]` to the dict object instead of `&String{Value: err.Message}`

Tests (add to `unified_error_test.go`):
- T1: `let {result, error} = try fn() { fail("oops") }()` → `error.message == "oops"`, `result == null`
- T2: `try fn() { fail({code: "NO_STOCK", message: "Out of stock", status: 400}) }()` → error dict has all three fields
- T7: `if (error)` guard still works (dict is truthy, null is falsy)
- T14: Internal catchable error (e.g., IO) produces dict with `message` and `code`
- T15: Non-catchable error (Type) still propagates — not caught by try
- Backward compat: existing `try` tests in `pkg/parsley/tests/control_flow_test.go` updated if they assert `error` is a string (change to assert `error.message`)

---

### Task 4: Update `api.*` helpers to return unified `*Error`
**Files**: `pkg/parsley/evaluator/stdlib_api.go`
**Estimated effort**: Medium

Change all six `api.*` helper functions to return `*Error` with `UserDict` instead of `*APIError`.

Steps:
1. Add `apiFailError(code string, message string, status int) *Error` helper function that builds an `*Error` with a `UserDict` containing `{code, message, status}`
2. Update each of the six functions to call `apiFailError`:
   - `apiNotFound` → `apiFailError("HTTP-404", msg, 404)`
   - `apiBadRequest` → `apiFailError("HTTP-400", msg, 400)`
   - `apiForbidden` → `apiFailError("HTTP-403", msg, 403)`
   - `apiUnauthorized` → `apiFailError("HTTP-401", msg, 401)`
   - `apiConflict` → `apiFailError("HTTP-409", msg, 409)`
   - `apiServerError` → `apiFailError("HTTP-500", msg, 500)`
3. Keep `APIError` type and `writeAPIError` for now (Phase 1 — server still has direct `APIError` construction sites)

Tests (add to `unified_error_test.go`):
- T5: `try fn() { api.notFound("User not found") }()` → error dict with `code: "HTTP-404"`, `message: "User not found"`, `status: 404`
- T6: Same for all six helpers (badRequest/403, forbidden/403, unauthorized/401, conflict/409, serverError/500)
- Update existing tests in `pkg/parsley/tests/stdlib_api_test.go` — they currently assert `*APIError` type; change to assert `*Error` type with correct `UserDict` fields

---

### Task 5: Update server dispatch to handle unified errors
**Files**: `server/api.go`
**Estimated effort**: Medium

Update the server's handler dispatch to recognize `*Error` with `UserDict` and send appropriate HTTP responses.

Steps:
1. In `dispatchModule`: replace the `*APIError` check + `*Error` check with a single `*Error` check:
   - If `errObj.UserDict != nil`: extract `status` from dict (default 500), write JSON response `{error: <UserDict>}`
   - Else: log as runtime error, send 500 (existing behavior)
2. Keep the `*APIError` check as a fallback below the `*Error` check (for the direct-construction sites in auth/rate-limit until Task 6)
3. Add `wrapErrorDict` helper that wraps a `*Dictionary` in `{error: <dict>}` to match existing response shape
4. In `writeAPIResponse`, keep the `*APIError` case for now (removed in Task 6)

Tests:
- T16: Handler returning `fail({status: 404, message: "Not found", code: "HTTP-404"})` → HTTP 404 with JSON body
- T17: Handler returning `fail("internal oops")` → HTTP 500
- T18: Handler returning `api.notFound("msg")` → HTTP 404 (same as today)
- Existing server API tests still pass

---

### Task 6: Migrate server-internal `APIError` construction sites
**Files**: `server/api.go`
**Estimated effort**: Small

Replace the four places where the server directly constructs `&evaluator.APIError{...}` with calls to `evaluator.ApiFailError()` (exported version of `apiFailError`).

Steps:
1. Export `apiFailError` as `ApiFailError` in `stdlib_api.go` (or add an exported wrapper)
2. Replace in `enforceAuth` (3 sites):
   - `&evaluator.APIError{Code: "HTTP-401", ...}` → write error using the new `*Error` + `UserDict` path
   - Since these call `writeAPIError` directly (not returning to dispatch), create a `writeUnifiedError(w, *Error)` helper or inline the status extraction logic
3. Replace in `enforceRateLimit` (1 site): same pattern
4. Remove `writeAPIError` if no longer called; otherwise keep as dead-code cleanup for Phase 2

Tests:
- Auth enforcement returns correct HTTP status codes (401, 403)
- Rate limit enforcement returns HTTP 429
- Existing server auth tests pass

---

### Task 7: String coercion for dicts with `message` key
**Files**: `pkg/parsley/evaluator/eval_string_conversions.go`
**Estimated effort**: Small

Add string coercion so that `"" + errorDict` produces the `message` field value for plain dicts.

Steps:
1. In `objectToTemplateString`, within the `*Dictionary` case, add a check before the special-type checks:
   - If dict has a `message` key AND is not a special type (path, url, tag, datetime, duration, regex, file, dir, request) → eval the `message` expression, if it's a `*String`, return its value
2. Order matters: this check must come before `isPathDict` etc. so that special types aren't affected

Tests (add to `unified_error_test.go`):
- T8: `"Error: " + error` where error is `{message: "oops"}` → `"Error: oops"`
- T9: `"Error: " + error` where error is `{message: "bad input", status: 400}` → `"Error: bad input"`
- Negative: `"" + {name: "x"}` (no message key) → existing dict Inspect behavior unchanged
- Negative: path/url/datetime dicts with a hypothetical `message` key → still use their special coercion

---

### Task 8: Add `record.failIfInvalid()` method
**Files**: `pkg/parsley/evaluator/methods_record.go`
**Estimated effort**: Small

Add the validation bridge convenience method.

Steps:
1. Add `"failIfInvalid"` to the `recordMethods` slice
2. Add `case "failIfInvalid"` in `evalRecordMethod` → call `recordFailIfInvalid(record, args)`
3. Implement `recordFailIfInvalid`:
   - Arity check: 0 args
   - If not validated: return record (no-op)
   - If valid (no errors): return record (enables chaining)
   - If invalid: build error dict `{status: 400, code: "VALIDATION", message: "Validation failed", fields: record.errorList()}` and return `*Error` with `UserDict` set

Tests (add to `unified_error_test.go`):
- T11: Valid record → `.failIfInvalid()` returns the record, chaining works
- T12: Invalid record → `try` catches error with `code: "VALIDATION"`, `status: 400`, `fields` array
- T13: Existing validation API (`isValid`, `errorList`, `hasError`) still works unchanged
- T: `failIfInvalid()` on un-validated record returns record (no-op)

---

### Task 9: Update existing tests
**Files**: `pkg/parsley/tests/control_flow_test.go`, `pkg/parsley/tests/stdlib_api_test.go`, `server/api_test.go` (if exists)
**Estimated effort**: Medium

Update any existing tests that break due to `error` being a dict instead of a string.

Steps:
1. Search all test files for patterns that assert `error` is a `*String` from `try` expressions
2. Update assertions: `error` is now a `*Dictionary` — check `error.message` instead
3. Search for tests that assert `*APIError` return type from `api.*` helpers — update to assert `*Error`
4. Run `go test ./...` and fix any remaining failures

Tests:
- Full test suite passes: `go test ./...`

---

### Task 10: Phase 2 cleanup — remove `*APIError` type
**Files**: `pkg/parsley/evaluator/stdlib_api.go`, `pkg/parsley/evaluator/evaluator.go`, `server/api.go`
**Estimated effort**: Small

Remove the now-unused `APIError` type and associated code.

Steps:
1. Verify no remaining references to `*APIError` (grep codebase)
2. Remove `APIError` struct, `ToDict()`, `Type()`, `Inspect()` methods from `stdlib_api.go`
3. Remove `API_ERROR_OBJ` constant from `evaluator.go`
4. Remove `writeAPIError` from `server/api.go` (if not already removed in Task 6)
5. Remove `*APIError` case from `writeAPIResponse` in `server/api.go`
6. Clean up any orphaned imports

Tests:
- `go build ./...` compiles
- `go test ./...` passes

---

### Task 11: Documentation updates
**Files**: docs and reference files
**Estimated effort**: Medium

Steps:
1. Update `docs/parsley/reference.md` — document `fail(string|dict)` behavior, error dict shape
2. Update `docs/parsley/CHEATSHEET.md` — add error dict pitfall/difference
3. Update `docs/parsley/manual/builtins/` — update `fail` docs if a manual page exists
4. Update `docs/guide/faq.md` — add "how do I return structured errors" entry
5. Add `record.failIfInvalid()` to record method documentation
6. Note string coercion behavior for error dicts

Tests:
- Review docs for accuracy

---

## Task Dependency Graph

```
Task 1 (UserDict field)
  └─► Task 2 (fail() string|dict)
       └─► Task 3 (try preserves dict)
            ├─► Task 4 (api.* unified)
            │    └─► Task 5 (server dispatch)
            │         └─► Task 6 (server APIError sites)
            │              └─► Task 10 (remove APIError)
            ├─► Task 7 (string coercion)
            └─► Task 8 (failIfInvalid)
                 └─► Task 9 (update existing tests)
                      └─► Task 11 (docs)
```

Tasks 7 and 8 can run in parallel after Task 3.
Tasks 4–6 are sequential (each builds on the previous).
Task 9 should run after Tasks 4–8 are all complete.
Task 10 runs after Task 6 (all APIError sites migrated).
Task 11 runs last.

## Validation Checklist

- [x] All tests pass: `go test ./...`
- [x] Build succeeds: `go build ./...`
- [ ] Linter passes: `golangci-lint run`
- [x] Documentation updated
- [x] work/BACKLOG.md updated with deferrals (#90–#93)
- [x] No remaining references to `*APIError` (after Task 10)

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-02-08 | Task 1: UserDict field | ✅ done | Added `UserDict *Dictionary` to `Error` struct |
| 2026-02-08 | Task 2: fail() string\|dict | ✅ done | Type-switch in `fail()` builtin, introspect updated |
| 2026-02-08 | Task 3: try preserves dict | ✅ done | `evalTryExpression` returns dict in error slot |
| 2026-02-08 | Task 4: api.* unified | ✅ done | All 6 helpers return `*Error` via `apiFailError` |
| 2026-02-08 | Task 5: server dispatch | ✅ done | `dispatchModule` checks `UserDict`, `writeUnifiedError` added |
| 2026-02-08 | Task 6: server APIError sites | ✅ done | Auth + rate-limit use `ApiFailError`, `writeUnifiedError` |
| 2026-02-08 | Task 7: string coercion | ✅ done | Dict with `message` key coerces to message string |
| 2026-02-08 | Task 8: failIfInvalid | ✅ done | Record method returns `*Error` with validation fields |
| 2026-02-08 | Task 9: update existing tests | ✅ done | try_test.go, stdlib_api_test.go updated; unified_error_test.go added |
| 2026-02-08 | Task 10: remove APIError | ✅ done | `APIError` struct and `API_ERROR_OBJ` removed |
| 2026-02-08 | Task 11: documentation | ✅ done | errors.md, control-flow.md, reference.md, CHEATSHEET.md, schema.md, faq.md |

## Deferred Items

Items to add to work/BACKLOG.md after implementation:
- Custom `message` in `record.failIfInvalid(msg)` — allow overriding the default "Validation failed" message
- `record.toError()` — return the error dict without calling `fail()`, for cases where you want to inspect/modify before failing
- Error catalog entries for `VALIDATION` code — currently hardcoded, could be registered in the error catalog
- `fail()` with dict: enforce `message` is non-empty string (currently only checks key exists and type)