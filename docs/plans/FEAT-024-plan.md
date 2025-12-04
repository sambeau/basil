# PLAN-016: FEAT-024 Print Function Implementation

**Feature:** FEAT-024 Print Function  
**Created:** 2025-12-04  
**Status:** Not Started  
**Estimated Effort:** 2-4 hours

## Overview

Implement `print()` and `println()` builtins that output values to the result stream rather than the dev log.

## Architecture Decision

**Chosen approach:** Environment-based print buffer

The Environment will carry a `printBuffer []Object` that accumulates `print()` outputs. Block evaluation merges this buffer with expression results in interleaved order.

### Why Environment-based?

1. **Thread-safe**: Each evaluation has its own environment
2. **Scoped**: Print buffer naturally scopes to current execution
3. **Minimal change**: No new object types needed
4. **Interleaving**: Can track print positions for correct ordering

## Tasks

### Task 1: Add PrintBuffer to Environment
**File:** `pkg/parsley/evaluator/evaluator.go`  
**Effort:** 15 min

Add print buffer and position tracking to Environment:

```go
type Environment struct {
    // ... existing fields
    printBuffer    []Object  // Accumulated print() outputs
    printPositions []int     // Position indices for interleaving
    resultCount    int       // Counter for interleaving
}
```

Add helper methods:
- `AddPrint(obj Object)` - Add to print buffer with position
- `GetPrintBuffer() []Object` - Retrieve buffer
- `ClearPrintBuffer()` - Reset after block evaluation

### Task 2: Implement objectToUserString Function
**File:** `pkg/parsley/evaluator/evaluator.go`  
**Effort:** 30 min

Create user-facing string conversion (distinct from `objectToDebugString`):

```go
func objectToUserString(obj Object) string {
    switch o := obj.(type) {
    case *String:
        return o.Value  // No quotes
    case *Integer:
        return strconv.FormatInt(o.Value, 10)
    case *Float:
        return strconv.FormatFloat(o.Value, 'f', -1, 64)
    case *Boolean:
        if o.Value { return "true" }
        return "false"
    case *Null:
        return ""  // Silent!
    case *Array:
        // JSON-style: [1, 2, 3]
    case *Dictionary:
        // Parsley-style: {a: 1, b: 2}
    case *Table:
        return fmt.Sprintf("<Table: %d rows, %d cols>", ...)
    case *Error:
        return fmt.Sprintf("[%s] %s", o.Code, o.Message)
    // ... other types per spec
    }
}
```

### Task 3: Implement print Builtin
**File:** `pkg/parsley/evaluator/evaluator.go`  
**Effort:** 20 min

Add to builtins map. Note: Builtins don't have environment access by default, so need special handling.

**Option A:** Make print a special-cased builtin with env access
**Option B:** Return special PrintValue object that evalBlockStatement handles

Recommend **Option B** for cleaner separation:

```go
// New object type
type PrintValue struct {
    Values []Object
}

func (pv *PrintValue) Type() ObjectType { return "PRINT_VALUE" }
func (pv *PrintValue) Inspect() string { return "<print>" }

// In builtins
"print": {
    Fn: func(args ...Object) Object {
        if len(args) == 0 {
            return newArityError("print", 0, 1)
        }
        return &PrintValue{Values: args}
    },
},
```

### Task 4: Implement println Builtin
**File:** `pkg/parsley/evaluator/evaluator.go`  
**Effort:** 10 min

```go
"println": {
    Fn: func(args ...Object) Object {
        values := make([]Object, 0, len(args)+1)
        for _, arg := range args {
            values = append(values, arg)
        }
        values = append(values, &String{Value: "\n"})
        return &PrintValue{Values: values}
    },
},
```

### Task 5: Modify evalBlockStatement for Print Handling
**File:** `pkg/parsley/evaluator/evaluator.go`  
**Effort:** 45 min

Modify block evaluation to handle PrintValue objects:

```go
func evalBlockStatement(block *ast.BlockStatement, env *Environment) Object {
    var results []Object

    for _, statement := range block.Statements {
        result := Eval(statement, env)

        if result != nil {
            rt := result.Type()
            if rt == RETURN_OBJ || rt == ERROR_OBJ {
                return result
            }

            // Handle print values - expand into results
            if pv, ok := result.(*PrintValue); ok {
                for _, v := range pv.Values {
                    str := objectToUserString(v)
                    if str != "" {  // Skip empty (null)
                        results = append(results, &String{Value: str})
                    }
                }
                continue  // Don't add PrintValue itself
            }

            // Collect non-NULL results
            if rt != NULL_OBJ {
                results = append(results, result)
            }
        }
    }

    // Return based on number of results (unchanged)
    switch len(results) {
    case 0:
        return NULL
    case 1:
        return results[0]
    default:
        return &Array{Elements: results}
    }
}
```

### Task 6: Handle Print in evalProgram
**File:** `pkg/parsley/evaluator/evaluator.go`  
**Effort:** 15 min

Similar handling for top-level print statements.

### Task 7: Handle Print in evalInterpolationBlock
**File:** `pkg/parsley/evaluator/evaluator.go`  
**Effort:** 15 min

Same pattern for interpolation blocks.

### Task 8: Write Tests
**File:** `pkg/parsley/tests/print_test.go`  
**Effort:** 1 hour

Test cases:
1. Basic print with single value
2. Print with multiple values
3. println with value
4. println with no args (bare newline)
5. print with no args (error)
6. Print in for loop
7. Print in if/else
8. Print interleaved with expressions
9. Print null (empty output)
10. Print all type representations
11. Nested blocks with print
12. Print in function body

### Task 9: Update Spec with Implementation Notes
**File:** `docs/specs/FEAT-024.md`  
**Effort:** 15 min

Document chosen architecture and any deviations from original spec.

## Test Plan

```parsley
// Test 1: Basic print
print("hello")  // Returns "hello"

// Test 2: Multiple args
print("a", "b")  // Returns ["a", "b"]

// Test 3: println
println("hi")  // Returns "hi\n"

// Test 4: Empty println
println()  // Returns "\n"

// Test 5: Print no args - ERROR
print()  // Error: print requires at least 1 argument

// Test 6: In loop
for i in 1..3 { print(i) }  // Returns ["1", "2", "3"]

// Test 7: Interleaved
{
    print("a")
    "b"
    print("c")
}  // Returns ["a", "b", "c"]

// Test 8: Null handling
print(null)  // Returns "" (empty, excluded from array)

// Test 9: Type representations
print(42)           // "42"
print(3.14)         // "3.14"
print(true)         // "true"
print([1,2,3])      // "[1, 2, 3]"
print({x: 1})       // "{x: 1}"
```

## Dependencies

- None - can be implemented independently
- Uses existing block evaluation from FEAT-022

## Risks

1. **Performance**: Print buffer allocation in hot loops - mitigate with pre-allocated buffer
2. **Interleaving complexity**: May need to reconsider if edge cases emerge

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-12-04 | Plan created | ✅ Done | |
| | Task 1: Environment | Not started | |
| | Task 2: objectToUserString | Not started | |
| | Task 3: print builtin | Not started | |
| | Task 4: println builtin | Not started | |
| | Task 5: evalBlockStatement | Not started | |
| | Task 6: evalProgram | Not started | |
| | Task 7: evalInterpolationBlock | Not started | |
| | Task 8: Tests | Not started | |
| | Task 9: Update spec | Not started | |
