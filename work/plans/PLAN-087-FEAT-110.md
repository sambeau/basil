---
id: PLAN-087
feature: FEAT-110
title: "Implementation Plan for Introspection Validation Tests"
status: complete
created: 2025-02-11
completed: 2025-02-11
---

# Implementation Plan: FEAT-110

## Overview
Create validation tests that verify the `TypeMethods` and `TypeProperties` maps in `introspect.go` accurately reflect the actual method implementations in `methods.go`. This catches drift between documentation and implementation.

## Prerequisites
- [x] Understanding of `TypeMethods` and `TypeProperties` structure in `introspect.go`
- [x] Understanding of method dispatch in `eval_method_dispatch.go`
- [x] Understanding of test patterns in the evaluator package

## Tasks

### Task 1: Create Test Value Factory Functions
**Files**: `pkg/parsley/evaluator/introspect_validation_test.go`
**Estimated effort**: Medium

Create helper functions to construct valid test values for each type that has methods or properties. Special attention needed for typed dictionaries (datetime, duration, path, url, regex, file, directory).

Steps:
1. Create `createTestValues()` function that returns `map[string]Object`
2. For primitive types (string, integer, float, boolean, array, dictionary), create simple values
3. For typed dictionaries (datetime, duration, path, url, regex, file, dir), create properly typed dicts with `__type` field
4. For special types (money, table, dbconnection, sftpconnection, session, dev, tablemodule), create appropriate instances or skip with documentation
5. Handle types that require external resources (dbconnection, sftpconnection) by noting them as "requires setup"

Tests:
- All types in `TypeMethods` have a corresponding test value
- Test values are properly typed (typed dicts recognized by `isDatetimeDict`, etc.)

---

### Task 2: Create Method Existence Tests
**Files**: `pkg/parsley/evaluator/introspect_validation_test.go`
**Estimated effort**: Medium

Verify every method listed in `TypeMethods` actually exists on the corresponding type by attempting to call it.

Steps:
1. Create `TestTypeMethods_AllMethodsExist(t *testing.T)`
2. Iterate over `TypeMethods` map
3. For each type, get or skip if test value not available
4. For each documented method, call `dispatchMethodCall()` with minimal args
5. Check if result is an "unknown method" error (code `UNDEF-0002`)
6. If unknown method error, fail test with clear message

Tests:
- Test passes when all documented methods exist
- Test fails clearly when a method is misspelled in `TypeMethods`
- Test fails clearly when a method is removed from implementation

---

### Task 3: Create Arity Validation Tests
**Files**: `pkg/parsley/evaluator/introspect_validation_test.go`
**Estimated effort**: Medium

Verify that documented method arities match actual implementation by testing minimum and maximum argument counts.

Steps:
1. Create `parseArityBounds(arity string) (min, max int, unbounded bool)` helper
   - "0" → (0, 0, false)
   - "1" → (1, 1, false)
   - "0-1" → (0, 1, false)
   - "1-2" → (1, 2, false)
   - "1+" → (1, -1, true)
2. Create `TestTypeMethods_ArityMatches(t *testing.T)`
3. For each method, call with (min - 1) args if min > 0 — expect arity error
4. For each method, call with min args — should NOT get arity error (may get type error, that's ok)
5. For bounded arities, call with (max + 1) args — expect arity error

Tests:
- Methods with arity "0" reject 1 argument
- Methods with arity "1" reject 0 arguments
- Methods with arity "0-1" accept both 0 and 1 args
- Methods with arity "1+" accept 1 and 10 args

---

### Task 4: Create Property Existence Tests
**Files**: `pkg/parsley/evaluator/introspect_validation_test.go`
**Estimated effort**: Small

Verify properties listed in `TypeProperties` actually exist on typed objects.

Steps:
1. Create `TestTypeProperties_AllPropertiesExist(t *testing.T)`
2. For each type with properties (datetime, duration, path, url, money, file, dir, table, regex)
3. Create test value with appropriate type
4. Use testEval to access property via `.propertyName` syntax
5. Verify no "unknown property" error

Tests:
- All datetime properties accessible (year, month, day, etc.)
- All path properties accessible (absolute, segments, extension, etc.)
- All money properties accessible (amount, currency, scale)

---

### Task 5: Create Helper Functions
**Files**: `pkg/parsley/evaluator/introspect_validation_test.go`
**Estimated effort**: Small

Create utility functions used by multiple tests.

Steps:
1. Create `makeArgs(count int) []Object` — returns slice of dummy String args
2. Create `isUnknownMethodError(obj Object) bool` — checks error code UNDEF-0002
3. Create `isArityError(obj Object) bool` — checks for arity-related error codes
4. Document which types are skipped and why (requires external setup)

Tests:
- Helper functions work correctly in isolation

---

### Task 6: Add Skipped Types Documentation
**Files**: `pkg/parsley/evaluator/introspect_validation_test.go`
**Estimated effort**: Small

Document types that cannot be easily tested and why.

Steps:
1. Add comment block at top explaining validation scope
2. List types skipped: dbconnection (requires DB), sftpconnection (requires SFTP server), session (requires server context)
3. For skipped types, add placeholder test that documents the skip reason

Tests:
- No test failures for types that require external resources

---

## Type Coverage Matrix

| Type | Methods | Properties | Test Value | Notes |
|------|---------|------------|------------|-------|
| string | ✓ | — | Simple | — |
| integer | ✓ | — | Simple | — |
| float | ✓ | — | Simple | — |
| boolean | ✓ | — | Simple | No methods documented |
| null | — | — | Simple | No methods documented |
| array | ✓ | — | Simple | — |
| dictionary | ✓ | Dynamic | Simple | Properties are dynamic keys |
| datetime | ✓ | ✓ | Typed dict | `__type: "datetime"` |
| duration | ✓ | ✓ | Typed dict | `__type: "duration"` |
| path | ✓ | ✓ | Typed dict | `__type: "path"` |
| url | ✓ | ✓ | Typed dict | `__type: "url"` |
| regex | ✓ | ✓ | Typed dict | `__type: "regex"` |
| file | ✓ | ✓ | Typed dict | `__type: "file"` |
| directory | ✓ | — | Typed dict | `__type: "dir"` |
| money | ✓ | ✓ | *Money | Direct struct |
| table | ✓ | ✓ | *Table | Empty table |
| dbconnection | ✓ | — | SKIP | Requires DB connection |
| sftpconnection | ✓ | — | SKIP | Requires SFTP server |
| session | ✓ | — | SKIP | Requires server context |
| dev | ✓ | — | SKIP | Requires dev module setup |
| tablemodule | ✓ | — | SKIP | Requires table module |
| function | — | — | — | No methods documented |

---

## Validation Checklist
- [x] All tests pass: `go test ./pkg/parsley/evaluator/... -run TestType` (6 expected failures for missing methods)
- [x] Build succeeds: `make build`
- [x] Linter passes: `golangci-lint run`
- [x] No import cycles introduced
- [x] Tests detect intentional drift (successfully detected 6 real drift issues)

## Test Execution

Run the validation tests:
```bash
go test ./pkg/parsley/evaluator/... -run TestTypeMethods -v
go test ./pkg/parsley/evaluator/... -run TestTypeProperties -v
```

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-02-11 | Task 1: Test Value Factory | ✅ Complete | Created `createTestValues()` with all testable types |
| 2025-02-11 | Task 2: Method Existence Tests | ✅ Complete | **FOUND ISSUES**: 6 missing methods |
| 2025-02-11 | Task 3: Arity Validation Tests | ✅ Complete | Tests verify min/max arity bounds |
| 2025-02-11 | Task 4: Property Existence Tests | ✅ Complete | All properties validated |
| 2025-02-11 | Task 5: Helper Functions | ✅ Complete | `parseArityBounds`, `makeArgs`, error checkers |
| 2025-02-11 | Task 6: Skip Documentation | ✅ Complete | 6 types documented as skipped |

## Test Results Summary

The validation tests **successfully detected drift** between `introspect.go` and `methods.go`:

### Missing Methods (documented but not implemented):
1. `integer.abs()` - Documented in TypeMethods, missing from evalIntegerMethod
2. `float.abs()` - Documented in TypeMethods, missing from evalFloatMethod  
3. `float.round()` - Documented in TypeMethods, missing from evalFloatMethod
4. `float.floor()` - Documented in TypeMethods, missing from evalFloatMethod
5. `float.ceil()` - Documented in TypeMethods, missing from evalFloatMethod
6. `money.negate()` - Documented in TypeMethods, missing from evalMoneyMethod

### Action Required:
These discrepancies must be resolved by either:
- **Option A**: Implementing the missing methods in `methods.go`
- **Option B**: Removing them from `TypeMethods` in `introspect.go` if not needed

This demonstrates the tests are working correctly and catching real drift issues.

## Deferred Items
Items to add to work/BACKLOG.md after implementation:
- Resolve 6 missing method implementations (see Test Results Summary above)
- FEAT-111 (Declarative Method Registry) will make these tests redundant by generating TypeMethods from actual implementations
- Consider adding reverse validation (methods exist but not documented) once FEAT-111 is in place