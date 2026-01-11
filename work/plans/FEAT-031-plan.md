---
id: PLAN-019
feature: FEAT-031
title: "Implementation Plan for std/math"
status: complete
created: 2025-12-05
completed: 2025-12-05
---

# Implementation Plan: FEAT-031 std/math

## Overview
Create the `std/math` standard library module with mathematical functions, constants, and random number generation. Move existing math builtins to this module.

**Total functions: 44 + 3 constants**

## Prerequisites
- [x] Design decisions finalized (see FEAT-031)
- [x] Understand existing stdlib loading mechanism (see `stdlib_table.go`)

## Existing Stdlib Pattern
The stdlib uses `loadStdlibModule()` in `stdlib_table.go` which dispatches to module loaders:

```go
// In getStdlibModules()
"math": loadMathModule,  // Add this

// New function
func loadMathModule(env *Environment) Object {
    return &StdlibModuleDict{
        Exports: map[string]Object{
            "PI":  &Float{Value: math.Pi},
            "E":   &Float{Value: math.E},
            "TAU": &Float{Value: math.Pi * 2},
            // ... functions as StdlibBuiltin or Builtin
        },
    }
}
```

Import syntax: `let {sin, cos, PI} = import(@std/math)` or `let math = import(@std/math)`

## Error Handling Requirements

All errors MUST use the structured error system from FEAT-006/FEAT-023:

### Existing Error Helpers (use these)
```go
newTypeError(code, function, expected, got)      // TYPE-0001, TYPE-0005, TYPE-0006
newArityError(function, got, want)               // ARITY-0001
newArityErrorRange(function, got, min, max)      // ARITY-0004
newArityErrorExact(function, got, choice1, choice2) // ARITY-0006
```

### New Error Codes Needed
Add to `pkg/parsley/errors/errors.go`:
```go
// VALUE-0001: Empty array for aggregation
"VALUE-0001": {
    Class:    ClassValue,
    Template: "`{{.Function}}` requires a non-empty array",
},
// VALUE-0002: Negative value where positive required
"VALUE-0002": {
    Class:    ClassValue,
    Template: "`{{.Function}}` requires a non-negative number, got {{.Got}}",
},
// VALUE-0003: Domain error (e.g., sqrt of negative, log of zero)
"VALUE-0003": {
    Class:    ClassValue,
    Template: "`{{.Function}}` domain error: {{.Reason}}",
},
```

### New Error Helper Needed
Add to `pkg/parsley/evaluator/evaluator.go`:
```go
func newValueError(code string, data map[string]any) *Error {
    perr := perrors.New(code, data)
    return &Error{
        Class:   ErrorClass(perr.Class),
        Code:    perr.Code,
        Message: perr.Message,
        Hints:   perr.Hints,
        Data:    perr.Data,
    }
}
```

### Example Usage
```go
// Empty array error
if len(arr.Elements) == 0 {
    return newValueError("VALUE-0001", map[string]any{"Function": "math.min"})
}

// Domain error
if x < 0 {
    return newValueError("VALUE-0003", map[string]any{
        "Function": "math.sqrt",
        "Reason":   "cannot take square root of negative number",
    })
}

// Type error (use existing)
if _, ok := args[0].(*Integer); !ok {
    return newTypeError("TYPE-0005", "math.floor", "a number", args[0].Type())
}
```

## Tasks

### Task 1: Add Error Codes and Helper
**Files**: `pkg/parsley/errors/errors.go`, `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small

Steps:
1. Add VALUE-0001, VALUE-0002, VALUE-0003 to ErrorCatalog
2. Add `newValueError()` helper function

Tests:
- Error codes produce correct messages
- ClassValue errors are catchable by `try`

---

### Task 2: Create std/math Module Structure
**Files**: `pkg/parsley/evaluator/stdlib_math.go` (new), `pkg/parsley/evaluator/stdlib_table.go`
**Estimated effort**: Medium

Steps:
1. Create `pkg/parsley/evaluator/stdlib_math.go`
2. Define `loadMathModule()` function returning `StdlibModuleDict`
3. Add constants (PI, E, TAU) to exports
4. Register "math" in `getStdlibModules()` in `stdlib_table.go`

Tests:
- Import `std/math` succeeds
- Constants accessible: `math.PI`, `math.E`, `math.TAU`

---

### Task 3: Implement Rounding Functions
**Files**: `pkg/parsley/evaluator/stdlib_math.go`
**Estimated effort**: Small

Steps:
1. Implement `floor(x)` - return int
2. Implement `ceil(x)` - return int
3. Implement `round(x)` - move from builtin, return int
4. Implement `trunc(x)` - return int

Tests:
- `math.floor(3.7)` → `3`
- `math.floor(-3.7)` → `-4`
- `math.ceil(3.2)` → `4`
- `math.ceil(-3.2)` → `-3`
- `math.round(3.5)` → `4`
- `math.round(3.4)` → `3`
- `math.trunc(-3.7)` → `-3`
- `math.trunc(3.7)` → `3`

---

### Task 4: Implement Comparison & Clamping Functions
**Files**: `pkg/parsley/evaluator/stdlib_math.go`
**Estimated effort**: Small

Steps:
1. Implement `abs(x)` - works on int or float
2. Implement `sign(x)` - returns -1, 0, or 1
3. Implement `clamp(x, min, max)`

Tests:
- `math.abs(-5)` → `5`
- `math.abs(-3.5)` → `3.5`
- `math.sign(-5)` → `-1`
- `math.sign(0)` → `0`
- `math.sign(5)` → `1`
- `math.clamp(15, 0, 10)` → `10`
- `math.clamp(-5, 0, 10)` → `0`
- `math.clamp(5, 0, 10)` → `5`

---

### Task 5: Implement Aggregation Functions (min, max, sum, avg, product, count)
**Files**: `pkg/parsley/evaluator/stdlib_math.go`
**Estimated effort**: Medium

Steps:
1. Implement `min(a, b)` and `min(array)` - detect array vs two args
2. Implement `max(a, b)` and `max(array)` - detect array vs two args
3. Implement `sum(a, b)` and `sum(array)` - detect array vs two args
4. Implement `avg(a, b)` and `avg(array)` - detect array vs two args
5. Implement `product(a, b)` and `product(array)` - detect array vs two args
6. Implement `count(array)` - count of elements

Tests:
- `math.min(5, 3)` → `3`
- `math.min([1, 2, 3])` → `1`
- `math.min([])` → error
- `math.max(5, 3)` → `5`
- `math.max([1, 2, 3])` → `3`
- `math.sum(5, 3)` → `8`
- `math.sum([1, 2, 3])` → `6`
- `math.avg(4, 6)` → `5`
- `math.avg([1, 2, 3])` → `2`
- `math.product(2, 3)` → `6`
- `math.product([2, 3, 4])` → `24`
- `math.count([1, 2, 3])` → `3`
- `math.count([])` → `0`

---

### Task 6: Implement Statistics Functions (median, mode, stddev, variance, range)
**Files**: `pkg/parsley/evaluator/stdlib_math.go`
**Estimated effort**: Medium

Steps:
1. Implement `median(array)` - middle value (sort, pick middle)
2. Implement `mode(array)` - most frequent value
3. Implement `stddev(array)` - population standard deviation
4. Implement `variance(array)` - population variance (stddev²)
5. Implement `range(array)` - max - min

Tests:
- `math.median([1, 2, 3, 4, 5])` → `3`
- `math.median([1, 2, 3, 4])` → `2.5` (average of two middle values)
- `math.median([5, 1, 3])` → `3` (sorts first)
- `math.median([])` → error
- `math.mode([1, 2, 2, 3])` → `2`
- `math.mode([1, 2, 3])` → `1` (all equal frequency, return first/smallest)
- `math.mode([])` → error
- `math.stddev([2, 4, 4, 4, 5, 5, 7, 9])` → `2`
- `math.stddev([1])` → `0`
- `math.stddev([])` → error
- `math.variance([2, 4, 4, 4, 5, 5, 7, 9])` → `4`
- `math.range([1, 5, 3])` → `4`
- `math.range([5])` → `0`
- `math.range([])` → error

---

### Task 7: Implement Random Functions
**Files**: `pkg/parsley/evaluator/stdlib_math.go`
**Estimated effort**: Medium

Steps:
1. Add RNG state to evaluator or module state
2. Implement `random()` - returns float 0.0-1.0
3. Implement `randomInt(min, max)` - returns int in range (inclusive)
4. Implement `seed(n)` - seeds RNG for reproducibility

Tests:
- `math.random()` returns float in [0, 1)
- `math.randomInt(1, 6)` returns int in [1, 6]
- `math.seed(42); math.random()` returns same value each time
- `math.randomInt(1, 1)` → `1`

---

### Task 8: Implement Powers & Logarithms
**Files**: `pkg/parsley/evaluator/stdlib_math.go`
**Estimated effort**: Small

Steps:
1. Move `sqrt(x)` from builtin
2. Move `pow(x, y)` from builtin
3. Implement `exp(x)`
4. Implement `log(x)` - natural log
5. Implement `log10(x)`

Tests:
- `math.sqrt(16)` → `4`
- `math.sqrt(2)` → `1.414...`
- `math.pow(2, 3)` → `8`
- `math.pow(2, 0.5)` → `1.414...`
- `math.exp(0)` → `1`
- `math.exp(1)` → `2.718...`
- `math.log(math.E)` → `1`
- `math.log10(100)` → `2`

---

### Task 9: Implement Trigonometry Functions
**Files**: `pkg/parsley/evaluator/stdlib_math.go`
**Estimated effort**: Small

Steps:
1. Move `sin(x)`, `cos(x)`, `tan(x)` from builtins
2. Move `asin(x)`, `acos(x)`, `atan(x)` from builtins
3. Implement `atan2(y, x)`

Tests:
- `math.sin(0)` → `0`
- `math.sin(math.PI / 2)` → `1`
- `math.cos(0)` → `1`
- `math.cos(math.PI)` → `-1`
- `math.tan(0)` → `0`
- `math.asin(0)` → `0`
- `math.acos(1)` → `0`
- `math.atan(0)` → `0`
- `math.atan2(1, 1)` → `0.7853...` (π/4)

---

### Task 10: Implement Angular Conversion
**Files**: `pkg/parsley/evaluator/stdlib_math.go`
**Estimated effort**: Small

Steps:
1. Implement `degrees(rad)` - radians to degrees
2. Implement `radians(deg)` - degrees to radians

Tests:
- `math.degrees(math.PI)` → `180`
- `math.degrees(math.PI / 2)` → `90`
- `math.radians(180)` → `3.14159...`
- `math.radians(90)` → `1.5707...`

---

### Task 11: Implement Geometry & Interpolation
**Files**: `pkg/parsley/evaluator/stdlib_math.go`
**Estimated effort**: Small

Steps:
1. Implement `hypot(x, y)` - hypotenuse √(x² + y²)
2. Implement `dist(x1, y1, x2, y2)` - 2D distance between points
3. Implement `lerp(a, b, t)` - linear interpolation
4. Implement `map(v, inMin, inMax, outMin, outMax)` - re-map value between ranges

Tests:
- `math.hypot(3, 4)` → `5`
- `math.hypot(0, 5)` → `5`
- `math.dist(0, 0, 3, 4)` → `5`
- `math.dist(1, 1, 4, 5)` → `5`
- `math.dist(0, 0, 0, 0)` → `0`
- `math.lerp(0, 10, 0.5)` → `5`
- `math.lerp(0, 10, 0)` → `0`
- `math.lerp(0, 10, 1)` → `10`
- `math.lerp(0, 10, 0.25)` → `2.5`
- `math.map(5, 0, 10, 0, 100)` → `50`
- `math.map(0.5, 0, 1, 0, 255)` → `127.5`
- `math.map(25, 0, 100, 0, 1)` → `0.25`

---

### Task 12: Remove Math Builtins
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small

Steps:
1. Remove `sin`, `cos`, `tan`, `asin`, `acos`, `atan` from builtins
2. Remove `sqrt`, `pow`, `round` from builtins
3. Remove `pi` from builtins
4. Update any tests that used builtins directly

Tests:
- Verify builtins no longer accessible
- All existing functionality works via `std/math`

---

### Task 13: Update Documentation
**Files**: `docs/parsley/reference.md`, `docs/parsley/CHEATSHEET.md`
**Estimated effort**: Small

Steps:
1. Add `std/math` section to reference.md
2. Update CHEATSHEET.md with import syntax
3. Remove math builtins from documentation

Tests:
- Documentation review

---

## Validation Checklist
- [x] All tests pass: `make test`
- [x] Build succeeds: `make build`
- [x] Linter passes: `golangci-lint run`
- [x] Documentation updated
- [x] BACKLOG.md updated with deferrals (if any)

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-12-05 | Task 1: Error Codes | Complete | Added VALUE-0001/0002/0003 and newValueError() |
| 2025-12-05 | Tasks 2-11: Implementation | Complete | All 44 functions + 3 constants in stdlib_math.go |
| 2025-12-05 | Tests | Complete | Comprehensive tests in stdlib_math_test.go |
| 2025-12-05 | Task 12: Builtins | Complete | Removed math builtins, updated affected tests |
| 2025-12-05 | Task 13: Documentation | Complete | Updated reference.md and CHEATSHEET.md |

## Notes
- Removed existing math builtins (`sin`, `cos`, `tan`, `asin`, `acos`, `atan`, `sqrt`, `pow`, `round`, `pi`) to require std/math import
- Added dot notation support for StdlibModuleDict to enable `math.PI` access
- The `log` function in std/math conflicts with builtin `log` (print); users should use `math.log()` or alias on import

## Deferred Items
Items to add to BACKLOG.md after implementation:
- Advanced statistics (percentile, quartile, correlation, zscore) - add based on demand
- Hyperbolic functions (sinh, cosh, tanh) - niche use case
- Special functions (gamma, factorial) - niche use case
