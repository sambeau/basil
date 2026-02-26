---
id: PLAN-108
feature: FEAT-129
title: "Implementation Plan for Parsley Standard Library v1.0 Cleanup"
status: complete
created: 2025-02-26
completed: 2025-02-26
---

# Implementation Plan: FEAT-129

## Overview

Clean up the Parsley standard library for v1.0 release. This plan covers deprecations, refactoring `@std/valid`, adding `@std/hash`, adding string methods, moving server-specific modules to `@basil/`, and updating documentation.

## Prerequisites

- [ ] Audit report reviewed: `work/reports/STDLIB-AUDIT-2025.md`
- [ ] Feature spec approved: `work/specs/FEAT-129.md`

## Tasks

### Task 1: Deprecate @std/schema
**Files**: `pkg/parsley/evaluator/stdlib_schema.go`
**Estimated effort**: Small

Steps:
1. Modify `loadSchemaModule()` to return a deprecation error instead of loading the module
2. Error message should reference `@schema` DSL syntax and point to docs

Tests:
- Importing `@std/schema` returns error with migration message
- Error message includes reference to `@schema` DSL

---

### Task 2: Add ID validators to @std/valid
**Files**: `pkg/parsley/evaluator/stdlib_valid.go`
**Estimated effort**: Small

Steps:
1. Add `ulidRegex` pattern: `^[0-9A-HJKMNP-TV-Z]{26}$`
2. Add `nanoidRegex` pattern: `^[0-9A-Za-z_-]+$`
3. Add `cuidRegex` pattern: `^c[0-9a-z]{24}$`
4. Implement `validULID(args)` — validate ULID format
5. Implement `validNanoID(args)` — validate NanoID format with optional length (default 21)
6. Implement `validCUID(args)` — validate CUID2 format
7. Register functions in module exports

Tests:
- `valid.ulid("01ARZ3NDEKTSV4RRFFQ69G5FAV")` → true
- `valid.ulid("invalid")` → false
- `valid.ulid("01arz3ndektsv4rrffq69g5fav")` → false (lowercase invalid)
- `valid.nanoid("V1StGXR8_Z5jdHi6B-myT")` → true
- `valid.nanoid("abc", 3)` → true
- `valid.nanoid("ab", 3)` → false (wrong length)
- `valid.cuid("cjld2cjxh0000qzrmn831i7rn")` → true
- `valid.cuid("invalid")` → false

---

### Task 3: Remove redundant functions from @std/valid
**Files**: `pkg/parsley/evaluator/stdlib_valid.go`
**Estimated effort**: Medium

Steps:
1. Remove type validators: `string`, `number`, `integer`, `boolean`, `array`, `dict`
2. Remove constraint validators: `minLen`, `maxLen`, `length`, `min`, `max`, `between`, `positive`, `negative`
3. Remove format validators: `email`, `url`, `phone`
4. Remove string validators: `matches`, `alpha`, `alphanumeric`, `numeric`, `empty`
5. Remove collection validators: `contains`, `oneOf`
6. Remove date validators: `date`, `time`, `parseDate`
7. Update module metadata to reflect remaining functions
8. Clean up any helper functions no longer needed

Tests:
- Verify removed functions are no longer accessible
- Verify kept functions (`uuid`, `creditCard`, `luhn`, `postalCode`) still work
- Update/remove existing tests for removed functions

---

### Task 4: Create @std/hash module
**Files**: `pkg/parsley/evaluator/stdlib_hash.go` (new), `pkg/parsley/evaluator/stdlib_table.go`
**Estimated effort**: Small

Steps:
1. Create `stdlib_hash.go` with module metadata
2. Implement `hashMD5(args)` using `crypto/md5`
3. Implement `hashSHA1(args)` using `crypto/sha1`
4. Implement `hashSHA256(args)` using `crypto/sha256`
5. Implement `hashSHA512(args)` using `crypto/sha512`
6. Create `loadHashModule()` function
7. Register module in `loadStdlibModule()` switch statement

Tests:
- `hash.md5("hello")` → `"5d41402abc4b2a76b9719d911017c592"`
- `hash.sha1("hello")` → `"aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d"`
- `hash.sha256("hello")` → `"2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"`
- `hash.sha512("hello")` → (128-char hex string)
- Empty string input works correctly
- Non-string input returns error

---

### Task 5: Add Base64 string methods
**Files**: `pkg/parsley/evaluator/methods_string.go`
**Estimated effort**: Small

Steps:
1. Implement `stringToBase64()` using `encoding/base64.StdEncoding`
2. Implement `stringFromBase64()` with error handling for invalid input
3. Register methods in `StringMethodRegistry`

Tests:
- `"hello".toBase64()` → `"aGVsbG8="`
- `"aGVsbG8=".fromBase64()` → `"hello"`
- `"".toBase64()` → `""`
- `"".fromBase64()` → `""`
- `"invalid!!".fromBase64()` → error
- Unicode strings encode/decode correctly

---

### Task 6: Add case conversion string methods
**Files**: `pkg/parsley/evaluator/methods_string.go`
**Estimated effort**: Medium

Steps:
1. Create helper function to split string into words (handle snake_case, kebab-case, camelCase, PascalCase)
2. Implement `stringToCamel()` — first word lowercase, subsequent words capitalized
3. Implement `stringToPascal()` — all words capitalized
4. Implement `stringToSnake()` — lowercase with underscores
5. Implement `stringToKebab()` — lowercase with hyphens
6. Register methods in `StringMethodRegistry`

Tests:
- `"hello_world".toCamel()` → `"helloWorld"`
- `"hello_world".toPascal()` → `"HelloWorld"`
- `"HelloWorld".toSnake()` → `"hello_world"`
- `"HelloWorld".toKebab()` → `"hello-world"`
- `"hello-world".toCamel()` → `"helloWorld"`
- `"helloWorld".toSnake()` → `"hello_world"`
- `"XMLParser".toSnake()` → `"xml_parser"`
- `"already_snake".toSnake()` → `"already_snake"`
- Empty string returns empty string

---

### Task 7: Add truncate string method
**Files**: `pkg/parsley/evaluator/methods_string.go`
**Estimated effort**: Small

Steps:
1. Implement `stringTruncate(args)` with optional suffix parameter (default "...")
2. Use `unicode/utf8.RuneCountInString()` for proper Unicode handling
3. If string length ≤ target length, return unchanged
4. Otherwise, truncate to (length - suffix length) and append suffix
5. Register method in `StringMethodRegistry`

Tests:
- `"Hello world".truncate(8)` → `"Hello..."`
- `"Hello world".truncate(8, "…")` → `"Hello…"`
- `"Hi".truncate(8)` → `"Hi"` (no change)
- `"Hello".truncate(5)` → `"Hello"` (exact length)
- `"Hello".truncate(3)` → `"..."` (edge case: suffix fills entire space)
- Unicode: `"こんにちは".truncate(4)` → `"こ..."`

---

### Task 8: Move @std/api to @basil/api
**Files**: `pkg/parsley/evaluator/stdlib_api.go`, `pkg/parsley/evaluator/basil_modules.go`
**Estimated effort**: Small

Steps:
1. Create `loadBasilApiModule()` in basil modules
2. In `loadStdlibModule()`, make `"api"` emit deprecation warning and delegate to basil loader
3. Register module in `loadBasilModule()` under `"api"`
4. Update any internal references

Tests:
- `import @basil/api` works correctly
- `import @std/api` emits deprecation warning but still works
- All api functions work from both import paths

---

### Task 9: Move @std/dev to @basil/log
**Files**: `pkg/parsley/evaluator/stdlib_dev.go`, `pkg/parsley/evaluator/basil_modules.go`
**Estimated effort**: Small

Steps:
1. Create `loadBasilLogModule()` in basil modules (rename from dev)
2. In `loadStdlibModule()`, make `"dev"` emit deprecation warning and delegate to basil loader
3. Register module in `loadBasilModule()` under `"log"`
4. Update any internal references

Tests:
- `import @basil/log` works correctly
- `import @std/dev` emits deprecation warning but still works
- All log functions work from both import paths

---

### Task 10: Move @std/html to @basil/html
**Files**: `pkg/parsley/evaluator/stdlib_html.go`, `pkg/parsley/evaluator/basil_modules.go`
**Estimated effort**: Small

Steps:
1. Create `loadBasilHtmlModule()` in basil modules
2. In `loadStdlibModule()`, make `"html"` emit deprecation warning and delegate to basil loader
3. Register module in `loadBasilModule()` under `"html"`
4. Update any internal references

Tests:
- `import @basil/html` works correctly
- `import @std/html` emits deprecation warning but still works
- All html components work from both import paths

---

### Task 11: Update documentation
**Files**: `docs/parsley/reference.md`, `docs/parsley/manual/`
**Estimated effort**: Medium

Steps:
1. Update `@std/valid` section — remove docs for removed functions, add docs for new validators
2. Add `@std/hash` section with function documentation
3. Add string method documentation for all 7 new methods
4. Add `@basil/` namespace section documenting `api`, `log`, `html`
5. Add migration guide section for deprecated/moved modules
6. Update namespace overview to explain `@std/` vs `@basil/` distinction

Tests:
- All documented examples work when tested
- No references to removed functions

---

### Task 12: Add integration tests
**Files**: `pkg/parsley/tests/`
**Estimated effort**: Medium

Steps:
1. Add test file for `@std/hash` module
2. Add test file for new `@std/valid` functions
3. Add test file for new string methods
4. Add test for deprecation error on `@std/schema`
5. Add tests for deprecation aliases (`@std/api`, `@std/dev`, `@std/html`)

Tests:
- All test cases from individual tasks consolidated into integration tests
- Edge cases covered
- Error cases covered

---

## Validation Checklist

- [x] All tests pass: `go test ./pkg/parsley/...`
- [x] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [x] Documentation updated
- [ ] work/BACKLOG.md updated with deferrals (if any)
- [ ] FEAT-129 acceptance criteria all checked

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-02-26 | Task 1: Deprecate @std/schema | ✅ Complete | Warning (DEP-002), still works |
| 2025-02-26 | Task 2: Add ID validators | ✅ Complete | ulid, nanoid, cuid added |
| 2025-02-26 | Task 3: Remove redundant functions | ✅ Complete | 27 functions removed |
| 2025-02-26 | Task 4: Create @std/hash | ✅ Complete | md5, sha1, sha256, sha512 |
| 2025-02-26 | Task 5: Base64 methods | ✅ Complete | toBase64, fromBase64 |
| 2025-02-26 | Task 6: Case conversion methods | ✅ Complete | toCamel, toPascal, toSnake, toKebab |
| 2025-02-26 | Task 7: Truncate method | ✅ Complete | With Unicode support |
| 2025-02-26 | Task 8: Move @std/api | ✅ Complete | @basil/api with DEP-003 alias |
| 2025-02-26 | Task 9: Move @std/dev | ✅ Complete | @basil/log with DEP-004 alias |
| 2025-02-26 | Task 10: Move @std/html | ✅ Complete | @basil/html with DEP-005 alias |
| 2025-02-26 | Task 11: Documentation | ✅ Complete | reference.md fully updated |
| 2025-02-26 | Task 12: Integration tests | ✅ Complete | All new features tested |

## Deferred Items

Items to add to work/BACKLOG.md after implementation:
- Pluralization/singularization — Requires inflection database, English-specific
- Additional postal code locales — Add based on user demand
- HMAC signing — Niche use case, add if requested

## Implementation Order

Recommended order to minimize conflicts and enable incremental testing:

1. **Task 4** (hash module) — Independent, can be tested immediately
2. **Tasks 5-7** (string methods) — Independent, can be tested immediately
3. **Task 2** (add ID validators) — Add before removing to ensure tests pass
4. **Task 3** (remove redundant functions) — After validators added
5. **Task 1** (deprecate schema) — Simple, independent
6. **Tasks 8-10** (namespace moves) — Can be done in parallel
7. **Task 12** (integration tests) — After all functionality complete
8. **Task 11** (documentation) — Final step