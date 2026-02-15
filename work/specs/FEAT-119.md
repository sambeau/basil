---
id: FEAT-119
title: "Rename bytes() to raw(), add bytes() unit constructor"
status: complete
priority: low
created: 2025-01-14
completed: 2025-01-14
author: "@human"
---

# FEAT-119: Rename bytes() to raw(), add bytes() unit constructor

## Summary

Rename the file I/O builtin `bytes()` to `raw()` to free up the name for a unit constructor. Then add `bytes()` as a named constructor for data units, completing the set of plural-form unit constructors specified in FEAT-118.

## User Story

As a Parsley developer working with data units, I want to use `bytes(1024)` to create a unit value, consistent with other unit constructors like `kilobytes()`, `megabytes()`, etc.

## Background

FEAT-118 (Measurement Units) introduced named constructors for all unit families using plural forms: `metres()`, `feet()`, `kilograms()`, `kilobytes()`, etc. The `bytes()` constructor was intentionally omitted because it conflicted with an existing file I/O builtin.

The existing `bytes(path)` function loads a file as a raw byte array. Renaming it to `raw(path)` is arguably clearer (it returns raw/unprocessed data) and frees the `bytes` name for the unit constructor.

## Acceptance Criteria

### Part 1: Rename file I/O builtin
- [x] Rename `bytes()` to `raw()` in the builtins
- [x] Update `BuiltinMetadata` for introspection
- [x] `raw(path)` loads file as byte array (same behavior as current `bytes()`)
- [x] Update all tests that use `bytes()` for file I/O

### Part 2: Add unit constructor
- [x] Add `bytes()` as a named constructor for data units
- [x] `bytes(1024)` → `#1024B`
- [x] `bytes(#1KiB)` → `#1024B` (conversion)
- [x] Remove the "bytes omitted" comment from evaluator.go
- [x] Add tests for the new `bytes()` constructor

### Part 3: Documentation
- [x] Update any documentation referencing `bytes()` for file I/O

## Scope

### In Scope
- Renaming the file I/O function
- Adding the unit constructor
- Updating tests

### Out of Scope
- Adding a deprecation period (usage is minimal)
- Keeping `bytes()` as an alias for `raw()`

## Impact Analysis

### Breaking Change
This is a breaking change for any code using `bytes()` for file I/O. However, analysis shows minimal usage:
- No usage in production Parsley code (only 1 commented-out example in a syntax test file)
- A few test files that will be updated as part of this change

### Files to Modify
- `pkg/parsley/evaluator/evaluator.go` - rename builtin, add constructor
- `pkg/parsley/evaluator/introspect.go` - update metadata
- `pkg/parsley/tests/filehandle_test.go` - update tests
- `pkg/parsley/tests/read_operator_test.go` - update tests  
- `pkg/parsley/tests/write_operator_test.go` - update tests
- `pkg/parsley/tests/unit_test.go` - add bytes() constructor test
- `.vscode-extension/test/syntax-test.pars` - update example

## Related
- FEAT-118: Measurement Units (parent feature)