---
id: PLAN-098
feature: FEAT-119
title: "Implementation Plan: Rename bytes() to raw(), add bytes() unit constructor"
status: complete
created: 2025-01-14
completed: 2025-01-14
---

# Implementation Plan: FEAT-119

## Overview
Rename the file I/O builtin `bytes()` to `raw()` and add `bytes()` as a named constructor for data units, completing the set of plural-form unit constructors from FEAT-118.

## Prerequisites
- [x] FEAT-118 implemented (unit system in place)
- [x] Impact analysis complete (minimal usage of bytes() for file I/O)

## Tasks

### Task 1: Rename bytes() to raw() in builtins
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small

Steps:
1. Find the `"bytes":` builtin definition in getBuiltins()
2. Rename it to `"raw":`
3. Update any internal references to the function name in error messages

Tests:
- Existing bytes() tests will fail (expected, fixed in Task 3)

---

### Task 2: Update BuiltinMetadata for introspection
**Files**: `pkg/parsley/evaluator/introspect.go`
**Estimated effort**: Small

Steps:
1. Find `"bytes":` entry in BuiltinMetadata
2. Rename to `"raw":` and update description to "Load file as raw byte array"

Tests:
- Introspection should show `raw` instead of `bytes`

---

### Task 3: Update tests for file I/O rename
**Files**: 
- `pkg/parsley/tests/filehandle_test.go`
- `pkg/parsley/tests/read_operator_test.go`
- `pkg/parsley/tests/write_operator_test.go`

**Estimated effort**: Small

Steps:
1. Replace all `bytes(` with `raw(` in test code strings
2. Update test function names if they reference "bytes"
3. Update expected format strings from "bytes" to "raw"

Tests:
- All renamed tests should pass

---

### Task 4: Add bytes() unit constructor
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small

Steps:
1. Find the data unit constructors section (near kilobytes, megabytes, etc.)
2. Add `"bytes": {Fn: func(args ...Object) Object { return unitNamedConstructor("bytes", args) }},`
3. Remove the "NOTE: bytes omitted" comment

Tests:
- `bytes(1024)` → `#1024B`
- `bytes(#1KiB)` → `#1024B`

---

### Task 5: Add bytes to UnitConstructorNames table
**Files**: `pkg/parsley/evaluator/unit_tables.go`
**Estimated effort**: Small

Steps:
1. Verify "bytes" → "B" mapping exists in UnitConstructorNames (it should already be there from FEAT-118)

Tests:
- Constructor lookup works correctly

---

### Task 6: Add tests for bytes() unit constructor
**Files**: `pkg/parsley/tests/unit_test.go`
**Estimated effort**: Small

Steps:
1. Add `{`bytes(1024)`, `#1024B`}` to TestUnitNamedConstructors
2. Add conversion test: `bytes(#1KiB)` → `#1024B`

Tests:
- bytes() constructor creates correct unit values
- bytes() converts from other data units

---

### Task 7: Update syntax example file
**Files**: `.vscode-extension/test/syntax-test.pars`
**Estimated effort**: Small

Steps:
1. Change `bytes(@./image.png)` to `raw(@./image.png)` in the commented example

Tests:
- N/A (comment only)

---

## Validation Checklist
- [x] All tests pass: `go test ./...`
- [x] Build succeeds: `make build`
- [x] Linter passes: `golangci-lint run`
- [x] bytes() unit constructor works: `./pars -e 'bytes(1024)'` → `#1024B`
- [x] raw() file I/O works: test manually with a file

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-01-14 | Task 1-7 | ✅ Complete | All tasks implemented and tested |

## Deferred Items
None.