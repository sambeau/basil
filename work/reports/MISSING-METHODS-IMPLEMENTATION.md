---
id: MISSING-METHODS-IMPLEMENTATION
title: "Missing Methods Implementation - Completion Summary"
date: 2026-02-11
related: FEAT-110
status: complete
---

# Missing Methods Implementation Summary

## Overview

Successfully implemented all 6 missing methods that were detected by the introspection validation tests (FEAT-110). These methods were documented in `TypeMethods` but missing from the actual implementation.

## Background

The introspection validation tests created for FEAT-110 immediately detected drift between documentation and implementation:

- **Documentation**: Methods listed in `introspect.go` TypeMethods map
- **Reality**: Methods missing from `methods.go` implementation
- **Impact**: Users calling these methods got "unknown method" errors

## Methods Implemented

### 1. `integer.abs()` - Integer Absolute Value

**Location**: `pkg/parsley/evaluator/methods.go` (evalIntegerMethod)

**Signature**: `abs() -> Integer`

**Implementation**:
```go
case "abs":
    if len(args) != 0 {
        return newArityError("abs", len(args), 0)
    }
    value := num.Value
    if value < 0 {
        value = -value
    }
    return &Integer{Value: value}
```

**Examples**:
- `(42).abs()` → `42`
- `(-42).abs()` → `42`
- `(0).abs()` → `0`

---

### 2. `float.abs()` - Float Absolute Value

**Location**: `pkg/parsley/evaluator/methods.go` (evalFloatMethod)

**Signature**: `abs() -> Float`

**Implementation**:
```go
case "abs":
    if len(args) != 0 {
        return newArityError("abs", len(args), 0)
    }
    return &Float{Value: math.Abs(num.Value)}
```

**Examples**:
- `(3.14).abs()` → `3.14`
- `(-3.14).abs()` → `3.14`
- `(-0.5).abs()` → `0.5`

---

### 3. `float.round(decimals?)` - Round to Decimal Places

**Location**: `pkg/parsley/evaluator/methods.go` (evalFloatMethod)

**Signature**: `round(decimals?: Integer) -> Float`

**Implementation**:
```go
case "round":
    if len(args) > 1 {
        return newArityErrorRange("round", len(args), 0, 1)
    }
    decimals := 0
    if len(args) == 1 {
        d, ok := args[0].(*Integer)
        if !ok {
            return newTypeError("TYPE-0012", "round", "an integer", args[0].Type())
        }
        decimals = int(d.Value)
    }
    multiplier := math.Pow(10, float64(decimals))
    return &Float{Value: math.Round(num.Value*multiplier) / multiplier}
```

**Examples**:
- `(3.14159).round()` → `3.0`
- `(3.14159).round(2)` → `3.14`
- `(3.14159).round(4)` → `3.1416`
- `(3.5).round()` → `4.0` (round-half-away-from-zero)

**Note**: Uses Go's `math.Round` which implements round-half-away-from-zero (not banker's rounding).

---

### 4. `float.floor()` - Round Down

**Location**: `pkg/parsley/evaluator/methods.go` (evalFloatMethod)

**Signature**: `floor() -> Float`

**Implementation**:
```go
case "floor":
    if len(args) != 0 {
        return newArityError("floor", len(args), 0)
    }
    return &Float{Value: math.Floor(num.Value)}
```

**Examples**:
- `(3.14).floor()` → `3.0`
- `(3.99).floor()` → `3.0`
- `(-3.14).floor()` → `-4.0`
- `(-3.99).floor()` → `-4.0`

---

### 5. `float.ceil()` - Round Up

**Location**: `pkg/parsley/evaluator/methods.go` (evalFloatMethod)

**Signature**: `ceil() -> Float`

**Implementation**:
```go
case "ceil":
    if len(args) != 0 {
        return newArityError("ceil", len(args), 0)
    }
    return &Float{Value: math.Ceil(num.Value)}
```

**Examples**:
- `(3.14).ceil()` → `4.0`
- `(3.99).ceil()` → `4.0`
- `(-3.14).ceil()` → `-3.0`
- `(-3.99).ceil()` → `-3.0`

---

### 6. `money.negate()` - Negate Money Value

**Location**: `pkg/parsley/evaluator/methods.go` (evalMoneyMethod)

**Signature**: `negate() -> Money`

**Implementation**:
```go
case "negate":
    if len(args) != 0 {
        return newArityError("negate", len(args), 0)
    }
    return &Money{
        Amount:   -money.Amount,
        Currency: money.Currency,
        Scale:    money.Scale,
    }
```

**Examples**:
- `($50.00).negate()` → `$-50.00`
- `($-50.00).negate()` → `$50.00`
- `(€100.00).negate()` → `€-100.00`

---

## Test Coverage

### New Test File: `methods_missing_test.go`

**Location**: `pkg/parsley/evaluator/methods_missing_test.go`

**Size**: 260 lines

**Test Functions**:
1. `TestIntegerAbs` - Positive, negative, zero cases
2. `TestFloatAbs` - Various float values
3. `TestFloatRound` - Different decimal places
4. `TestFloatFloor` - Positive and negative cases
5. `TestFloatCeil` - Positive and negative cases
6. `TestMoneyNegate` - Different currencies
7. `TestIntegerAbsArity` - Arity validation
8. `TestFloatRoundArityAndType` - Arity and type errors
9. `TestFloatFloorArity` - Arity validation
10. `TestFloatCeilArity` - Arity validation
11. `TestMoneyNegateArity` - Arity validation
12. `TestMethodChaining` - Chaining with other methods

**Test Results**: ✅ All tests pass

---

## Validation Results

### Before Implementation

```
--- FAIL: TestTypeMethods_AllMethodsExist (0.00s)
    --- FAIL: TestTypeMethods_AllMethodsExist/integer.abs (0.00s)
    --- FAIL: TestTypeMethods_AllMethodsExist/float.abs (0.00s)
    --- FAIL: TestTypeMethods_AllMethodsExist/float.round (0.00s)
    --- FAIL: TestTypeMethods_AllMethodsExist/float.floor (0.00s)
    --- FAIL: TestTypeMethods_AllMethodsExist/float.ceil (0.00s)
    --- FAIL: TestTypeMethods_AllMethodsExist/money.negate (0.00s)
```

### After Implementation

```
--- PASS: TestTypeMethods_AllMethodsExist (0.00s)
    --- PASS: TestTypeMethods_AllMethodsExist/integer.abs (0.00s)
    --- PASS: TestTypeMethods_AllMethodsExist/float.abs (0.00s)
    --- PASS: TestTypeMethods_AllMethodsExist/float.round (0.00s)
    --- PASS: TestTypeMethods_AllMethodsExist/float.floor (0.00s)
    --- PASS: TestTypeMethods_AllMethodsExist/float.ceil (0.00s)
    --- PASS: TestTypeMethods_AllMethodsExist/money.negate (0.00s)
```

---

## CLI Testing

All methods verified working with `pars` CLI:

```bash
$ ./pars -e "(-42).abs()"
42

$ ./pars -e "(3.14159).round(2)"
3.14

$ ./pars -e "(3.7).floor()"
3

$ ./pars -e "(3.2).ceil()"
4

$ ./pars -e "(-3.14).abs()"
3.14

$ ./pars -e '($50.00).negate()'
$-50.00
```

---

## Quality Checks

### Build
✅ `make build` - Success

### Tests
✅ `go test ./pkg/parsley/evaluator/...` - All pass

### Linting
✅ `golangci-lint run ./pkg/parsley/evaluator/...` - 0 issues

### Validation Tests
✅ All introspection validation tests pass
✅ No drift between documentation and implementation

---

## Documentation Updates

### Updated Files

1. **`work/specs/FEAT-110.md`**
   - Added Phase 2: Missing Method Implementation
   - Documented all 6 methods with signatures and examples
   - Marked as complete

2. **`work/BACKLOG.md`**
   - Removed #98 from High Priority
   - Added to Completed (Archive) section with completion date

3. **`work/reports/MISSING-METHODS-IMPLEMENTATION.md`** (this file)
   - Comprehensive implementation summary

---

## Impact

### User Experience
- ✅ Methods that were documented now work
- ✅ No more "unknown method" errors for these 6 methods
- ✅ `describe()` builtin output is now accurate
- ✅ API is complete and consistent

### Code Quality
- ✅ Documentation matches implementation
- ✅ Comprehensive test coverage
- ✅ Error handling follows project conventions
- ✅ Method chaining works correctly

### Maintenance
- ✅ Validation tests will catch future drift
- ✅ Clear test cases serve as documentation
- ✅ Easy to maintain and extend

---

## Method Patterns

All implementations follow consistent patterns:

### Arity Validation
```go
if len(args) != expectedCount {
    return newArityError(methodName, len(args), expectedCount)
}
```

### Type Validation (for optional args)
```go
argObj, ok := args[0].(*ExpectedType)
if !ok {
    return newTypeError("TYPE-0012", methodName, "expected type", args[0].Type())
}
```

### Return New Instance
```go
return &ResultType{Value: computedValue}
```

This ensures:
- Consistent error messages
- Proper error codes
- Immutability (methods return new instances)

---

## Metrics

- **Implementation time**: ~1 hour
- **Lines added**: ~80 in methods.go
- **Test lines**: 260 in methods_missing_test.go
- **Test coverage**: 12 test functions covering all cases
- **Files modified**: 3
- **Files created**: 2
- **Bugs found**: 0
- **Regressions**: 0

---

## Conclusion

All 6 missing methods have been successfully implemented with:
- ✅ Complete implementations following project patterns
- ✅ Comprehensive test coverage
- ✅ Proper error handling
- ✅ Working method chaining
- ✅ CLI verification
- ✅ Documentation updates
- ✅ Zero linter issues
- ✅ All validation tests passing

The introspection system is now accurate and complete. The validation tests created in FEAT-110 successfully detected real issues and will continue to prevent future drift.

---

**Related Documents:**
- Feature: `work/specs/FEAT-110.md`
- Validation Tests: `pkg/parsley/evaluator/introspect_validation_test.go`
- Method Tests: `pkg/parsley/evaluator/methods_missing_test.go`
- Implementation: `pkg/parsley/evaluator/methods.go`
