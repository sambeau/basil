---
id: FEAT-110
title: "Introspection Validation Tests"
status: complete
priority: high
created: 2025-01-15
author: "@human"
blocking: false
---

# FEAT-110: Introspection Validation Tests

## Summary
Create validation tests that verify the `TypeMethods` and `TypeProperties` maps in `introspect.go` accurately reflect the actual method implementations in `methods.go`. This catches drift between documentation and implementation, ensuring `describe()` output is accurate.

## User Story
As a Parsley maintainer, I want automated tests that fail when introspection metadata falls out of sync with actual method implementations so that users can trust `describe()` output.

## Acceptance Criteria
- [x] Test file `introspect_validation_test.go` exists
- [x] Tests verify every method in `TypeMethods` actually exists on the corresponding type
- [x] Tests verify method arity matches documented arity
- [x] Tests fail with clear messages when methods are missing or mismatched
- [x] Tests run as part of normal `go test` suite
- [x] CI catches introspection drift on PRs (tests successfully detected 6 real drift issues)

## Design Decisions

- **Test-time validation only**: No runtime cost — validation happens during `go test`, not during program execution.

- **Probe-based approach**: Create sample values of each type and attempt to call documented methods. If a method doesn't exist, the test fails.

- **Arity verification**: For each documented method, verify the expected number of arguments is accepted (and wrong arity is rejected).

- **No description verification**: Descriptions are documentation; we can't automatically verify they're accurate. Focus on existence and arity.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `pkg/parsley/evaluator/introspect_validation_test.go` — New test file

### Dependencies
- Depends on: None
- Blocks: None (but should be done before 1.0 Alpha)

### Relationship to Other Work
- This is Phase 1 of the introspection improvement plan
- FEAT-111 (Declarative Method Registry) will make these tests redundant by design
- Until FEAT-111 is complete, these tests are essential for catching drift

---

## Implementation Plan

### Test Structure

```go
// pkg/parsley/evaluator/introspect_validation_test.go
package evaluator

import (
    "testing"
)

// Sample values for each type that has methods
var testValues = map[string]Object{
    "string":     &String{Value: "test"},
    "integer":    &Integer{Value: 42},
    "float":      &Float{Value: 3.14},
    "boolean":    &Boolean{Value: true},
    "array":      &Array{Elements: []Object{&Integer{Value: 1}}},
    "dictionary": &Dictionary{Pairs: map[string]ast.Expression{}},
    "datetime":   createTestDatetime(),
    "duration":   &Duration{Value: time.Hour},
    "money":      &Money{Amount: 1000, Currency: "USD"},
    "regex":      &Regex{Pattern: "test"},
    "url":        createTestURL(),
    "path":       &Path{Value: "/test"},
    "DBConnection": createTestDBConnection(),
    // ... etc
}

func TestTypeMethods_AllMethodsExist(t *testing.T) {
    for typeName, methods := range TypeMethods {
        testVal, ok := testValues[typeName]
        if !ok {
            t.Errorf("No test value for type %q — add one to testValues", typeName)
            continue
        }
        
        for _, method := range methods {
            t.Run(typeName+"."+method.Name, func(t *testing.T) {
                // Attempt to call method with minimum arity
                args := makeMinArgs(method.Arity)
                result := evalMethod(testVal, method.Name, args, nil)
                
                // Check for "unknown method" error
                if err, ok := result.(*Error); ok {
                    if strings.Contains(err.Message, "unknown method") ||
                       strings.Contains(err.Message, "has no method") {
                        t.Errorf("Method %s.%s is documented but doesn't exist", 
                            typeName, method.Name)
                    }
                }
            })
        }
    }
}

func TestTypeMethods_ArityMatches(t *testing.T) {
    for typeName, methods := range TypeMethods {
        testVal, ok := testValues[typeName]
        if !ok {
            continue // Already reported in existence test
        }
        
        for _, method := range methods {
            t.Run(typeName+"."+method.Name+"_arity", func(t *testing.T) {
                // Test minimum arity is accepted
                minArgs := parseMinArity(method.Arity)
                args := make([]Object, minArgs)
                for i := range args {
                    args[i] = &String{Value: "test"} // Dummy arg
                }
                result := evalMethod(testVal, method.Name, args, nil)
                
                // Should not get arity error
                if err, ok := result.(*Error); ok {
                    if strings.Contains(err.Message, "takes") ||
                       strings.Contains(err.Message, "argument") {
                        // May be wrong arg type, not arity - that's ok
                        // Only fail if it says wrong NUMBER of args
                        if strings.Contains(err.Message, "0") && minArgs > 0 {
                            t.Errorf("Method %s.%s documented as arity %q but rejected %d args",
                                typeName, method.Name, method.Arity, minArgs)
                        }
                    }
                }
            })
        }
    }
}

func TestTypeProperties_AllPropertiesExist(t *testing.T) {
    for typeName, props := range TypeProperties {
        if len(props) == 0 {
            continue // No properties to test
        }
        
        testVal, ok := testValues[typeName]
        if !ok {
            t.Errorf("No test value for type %q with properties", typeName)
            continue
        }
        
        for _, prop := range props {
            t.Run(typeName+"."+prop.Name, func(t *testing.T) {
                // Attempt property access
                result := evalPropertyAccess(testVal, prop.Name, nil)
                
                if err, ok := result.(*Error); ok {
                    if strings.Contains(err.Message, "unknown property") ||
                       strings.Contains(err.Message, "has no property") {
                        t.Errorf("Property %s.%s is documented but doesn't exist",
                            typeName, prop.Name)
                    }
                }
            })
        }
    }
}

// Helper to parse minimum arity from strings like "0", "1", "0-1", "1+"
func parseMinArity(arity string) int {
    // "0" -> 0
    // "1" -> 1
    // "0-1" -> 0
    // "1-2" -> 1
    // "1+" -> 1
    // ... etc
}

// Helper to create minimum args for a given arity
func makeMinArgs(arity string) []Object {
    n := parseMinArity(arity)
    args := make([]Object, n)
    for i := range args {
        args[i] = &String{Value: "test"}
    }
    return args
}
```

### Edge Cases

1. **Methods that require specific arg types**: Some methods will error with type errors, not "unknown method". This is fine — we're testing existence, not full functionality.

2. **Methods with side effects**: Some methods (like file writes) have side effects. Use mock values or skip these in the basic existence test.

3. **Methods requiring environment**: Database methods need connections. Create minimal mocks or mark as requiring setup.

4. **Properties on dictionaries**: Dictionary properties are dynamic. Document this exception.

---

## Test Plan

| Test | Expected |
|------|----------|
| Add a method to `methods.go` without updating `TypeMethods` | Test should NOT fail (method exists but undocumented — can't catch this) |
| Remove a method from `methods.go` without updating `TypeMethods` | Test SHOULD fail (documented method doesn't exist) |
| Change method arity without updating `TypeMethods` | Test SHOULD fail (arity mismatch) |
| Typo in method name in `TypeMethods` | Test SHOULD fail (unknown method) |
| All methods correctly documented | All tests pass |

**Note:** These tests catch when `TypeMethods` is WRONG (lists non-existent methods), but cannot catch when it's INCOMPLETE (missing methods that exist). FEAT-111 (Declarative Registry) solves this by making the registry the source of truth.

---

## Implementation Notes

### Phase 1: Validation Tests - Complete: 2025-02-11

Created `pkg/parsley/evaluator/introspect_validation_test.go` with comprehensive validation tests.

**File Structure:**
- 484 lines total
- Test value factory for all testable types
- Method existence validation
- Arity bounds validation
- Property existence validation
- Helper functions for parsing and error checking

**Tests Successfully Detect Drift:**

The validation tests immediately caught 6 real discrepancies between documentation and implementation:

1. **`integer.abs()`** - Documented in TypeMethods but missing from evalIntegerMethod
2. **`float.abs()`** - Documented in TypeMethods but missing from evalFloatMethod
3. **`float.round()`** - Documented in TypeMethods but missing from evalFloatMethod
4. **`float.floor()`** - Documented in TypeMethods but missing from evalFloatMethod
5. **`float.ceil()`** - Documented in TypeMethods but missing from evalFloatMethod
6. **`money.negate()`** - Documented in TypeMethods but missing from evalMoneyMethod

**Validation Coverage:**
- ✅ All primitive types (string, integer, float, boolean, null)
- ✅ All collection types (array, dictionary, table)
- ✅ All typed dictionaries (datetime, duration, path, url, regex, file, directory)
- ✅ Direct struct types (money, table)
- ⏭️ Skipped types requiring external resources (dbconnection, sftpconnection, session, dev, tablemodule)

**Test Execution:**
```bash
go test ./pkg/parsley/evaluator/... -run TestTypeMethods -v
go test ./pkg/parsley/evaluator/... -run TestTypeProperties -v
```

### Phase 2: Missing Method Implementation - Complete: 2025-02-11

All 6 missing methods have been implemented in `pkg/parsley/evaluator/methods.go`:

**`integer.abs()`** - Returns absolute value of integer
- Arity: 0
- Returns: `Integer`
- Example: `(-42).abs()` → `42`

**`float.abs()`** - Returns absolute value of float
- Arity: 0
- Returns: `Float`
- Example: `(-3.14).abs()` → `3.14`

**`float.round(decimals?)`** - Rounds to specified decimal places
- Arity: 0-1
- Returns: `Float`
- Default: 0 decimal places
- Example: `(3.14159).round(2)` → `3.14`

**`float.floor()`** - Rounds down to nearest integer
- Arity: 0
- Returns: `Float`
- Example: `(3.7).floor()` → `3.0`

**`float.ceil()`** - Rounds up to nearest integer
- Arity: 0
- Returns: `Float`
- Example: `(3.2).ceil()` → `4.0`

**`money.negate()`** - Returns negated money value
- Arity: 0
- Returns: `Money`
- Example: `($50.00).negate()` → `$-50.00`

**Test Coverage:**
- Created `pkg/parsley/evaluator/methods_missing_test.go` (260 lines)
- Tests for all 6 methods with positive and negative cases
- Arity validation tests
- Type error tests
- Method chaining tests
- All tests pass ✅

**Validation:**
- All introspection validation tests now pass
- No drift between `TypeMethods` and implementation
- Methods work correctly in REPL and CLI
- Full test suite passes
- Linter passes with zero issues

## Related
- Report: `work/reports/PARSLEY-1.0-ALPHA-READINESS.md` (Section 2)
- Source: `pkg/parsley/evaluator/introspect.go`
- Methods: `pkg/parsley/evaluator/methods.go`
- Next phase: FEAT-111 (Declarative Method Registry)