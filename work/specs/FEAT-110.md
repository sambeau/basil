---
id: FEAT-110
title: "Introspection Validation Tests"
status: draft
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
- [ ] Test file `introspect_validation_test.go` exists
- [ ] Tests verify every method in `TypeMethods` actually exists on the corresponding type
- [ ] Tests verify method arity matches documented arity
- [ ] Tests fail with clear messages when methods are missing or mismatched
- [ ] Tests run as part of normal `go test` suite
- [ ] CI catches introspection drift on PRs

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
*To be added during implementation*

## Related
- Report: `work/reports/PARSLEY-1.0-ALPHA-READINESS.md` (Section 2)
- Source: `pkg/parsley/evaluator/introspect.go`
- Methods: `pkg/parsley/evaluator/methods.go`
- Next phase: FEAT-111 (Declarative Method Registry)