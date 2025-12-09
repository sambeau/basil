---
id: PLAN-014
feature: FEAT-022
title: "Implementation Plan for Block Concatenation Investigation"
status: complete
created: 2025-12-04
completed: 2025-12-04
---

# Implementation Plan: FEAT-022 Block Concatenation

## Overview

Investigate whether Parsley can adopt concatenation semantics for all code blocks, where non-declaration expressions are collected and concatenated into the return value. This is an experimental investigation that may result in keeping or discarding the changes.

## Prerequisites

- [x] Feature spec created (FEAT-022)
- [ ] Create experimental branch

## Tasks

### Task 1: Create Experimental Branch
**Files**: N/A (git operation)
**Estimated effort**: Small

Steps:
1. Create branch `feat/FEAT-022-block-concatenation`
2. Ensure clean working state

---

### Task 2: Make Declarations Return NULL
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small

Steps:
1. Modify `case *ast.LetStatement` to return `NULL` instead of `val`
2. Modify `case *ast.AssignmentStatement` to return `NULL` instead of `val`
3. Check for other declaration-like constructs (import, etc.)

Current code (LetStatement ~line 6638):
```go
return val  // Change to: return NULL
```

Tests:
- Existing tests may break if they rely on assignment return values
- Document which tests break

---

### Task 3: Implement Block Concatenation
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Medium

Steps:
1. Modify `evalBlockStatement` to collect non-null results
2. Concatenate results at the end
3. Handle single-expression fast path
4. Ensure `return` statements still work

New implementation:
```go
func evalBlockStatement(block *ast.BlockStatement, env *Environment) Object {
    // Fast path: single statement
    if len(block.Statements) == 1 {
        return Eval(block.Statements[0], env)
    }
    
    var results []Object
    for _, statement := range block.Statements {
        result := Eval(statement, env)
        if result != nil {
            rt := result.Type()
            if rt == RETURN_OBJ || rt == ERROR_OBJ {
                return result
            }
            if result != NULL {
                results = append(results, result)
            }
        }
    }
    
    // Concatenate results
    return concatenateResults(results)
}
```

Tests:
- Test multiple expressions concatenate
- Test declarations are excluded
- Test return still works
- Test errors propagate

---

### Task 4: Implement concatenateResults Helper
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small

Steps:
1. Create helper function to concatenate results
2. Handle different types (strings, arrays, other)
3. Flatten nested arrays
4. Use strings.Builder for efficiency

```go
func concatenateResults(results []Object) Object {
    if len(results) == 0 {
        return NULL
    }
    if len(results) == 1 {
        return results[0]
    }
    
    var sb strings.Builder
    for _, r := range results {
        switch v := r.(type) {
        case *String:
            sb.WriteString(v.Value)
        case *Array:
            // Flatten and stringify
            for _, elem := range v.Elements {
                sb.WriteString(stringify(elem))
            }
        default:
            sb.WriteString(stringify(r))
        }
    }
    return &String{Value: sb.String()}
}
```

Tests:
- Empty results → NULL
- Single result → that result (no wrapping)
- Multiple strings → concatenated string
- Mixed types → all stringified

---

### Task 5: Update evalInterpolationBlock
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small

Steps:
1. Apply same concatenation logic to interpolation blocks
2. Ensure consistent behavior with regular blocks

---

### Task 6: Verify if/else and for Consistency
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small

Steps:
1. Check `evalIfExpression` — branches should use block semantics
2. Check `evalForExpression` — body already concatenates, verify consistency
3. Document any discrepancies

---

### Task 7: Run Test Suite and Document Breaks
**Files**: Various test files
**Estimated effort**: Medium

Steps:
1. Run `go test ./...`
2. Document each failing test
3. Categorize failures:
   - Expected semantic change (update test)
   - Actual bug (fix implementation)
   - Incompatible change (note for decision)

---

### Task 8: Performance Assessment
**Files**: N/A (benchmarking)
**Estimated effort**: Small

Steps:
1. Create benchmark for block evaluation
2. Compare before/after performance
3. Document memory allocation changes
4. Identify any hot paths that need optimization

---

### Task 9: Document Findings and Recommendation
**Files**: `docs/specs/FEAT-022.md`
**Estimated effort**: Small

Steps:
1. Update spec with investigation results
2. List breaking changes
3. List benefits
4. Make go/no-go recommendation
5. If go: create migration guide
6. If no-go: document learnings, consider alternatives

---

## Validation Checklist
- [ ] All tests pass (or failures documented as expected)
- [ ] Build succeeds: `go build -o basil ./cmd/basil`
- [ ] Performance acceptable
- [ ] Semantic changes documented
- [ ] Recommendation made

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-12-04 | Task 1 | ✅ Done | Branch created |
| 2025-12-04 | Task 2 | ✅ Done | LetStatement, AssignmentStatement, evalDestructuringAssignment, evalDictDestructuringAssignment now return NULL |
| 2025-12-04 | Task 3 | ✅ Done | evalBlockStatement now concatenates non-NULL results |
| 2025-12-04 | Task 4 | ⏭️ Skipped | Used `objectToTemplateString` directly instead of new helper |
| 2025-12-04 | Task 5 | ✅ Done | evalInterpolationBlock updated with same logic |
| 2025-12-04 | Task 7 | 🔄 In Progress | 38 test failures documented below |

## Test Failures Analysis

Running `go test ./...` shows 38 failing test cases:

### Category 1: Functions returning concatenated strings instead of typed values
These tests expect function calls to return integers/arrays/dicts but now get strings:
- `TestValidFunctionCallsWork/call_fn` - Expected Integer, got String "10"
- `TestValidFunctionCallsWork/call_function` - Expected Integer, got String "6"
- `TestValidFunctionCallsWork/call_from_dict` - Expected Integer, got String "14"
- `TestArrayDestructuringInFunctionParams/extract_rest_elements` - Expected [2,3,4,5], got "2345"
- `TestArrayDestructuringInFunctionParams/multiple_parameters` - Expected [20, 10], got "2010"
- `TestDictDestructuringInFunctionParams/with_rest_operator` - Expected dictionary, got String
- `TestHigherOrderFunctionsWithDestructuring/map-like_with_array_destructuring` - Expected [1,3,5], got "135"

### Category 2: Closures broken by stringification
- `TestDestructuringInClosures/closure_with_array_destructuring` - "cannot call STRING as a function"
- `TestDestructuringInClosures/closure_with_dict_destructuring` - "cannot call STRING as a function"

### Category 3: Assignments now return NULL (breaking tests that expect assignment results)
- `TestFunctions` - Multiple cases expecting function objects from `add = fn(...)`, get null
- `TestFunctionKeywordAlias` - 7 failing subcases
- `TestVariableAssignment` - Expects assignment to return value
- `TestAdvancedVariableUsage`

### Category 4: Module system affected
- `TestModuleDestructuring`
- `TestModuleAlias`
- `TestModuleFunction`
- `TestModuleClosures`

### Category 5: Database/SQL affected
- `TestSQLiteConnection/Create_SQLite_connection` - Expected DB_CONNECTION, got NULL
- `TestSQLTag/SQL_tag_with_params` - SQL syntax error

### Category 6: Other
- `TestInOperatorPrecedence` - if expressions affected
- `TestPathTemplateInExpressions/path_template_in_function`

## Decision Points

After investigation, we need to decide:

1. **Go**: Merge changes to main, update docs, release
2. **No-go but valuable**: Keep branch for future consideration
3. **No-go and discard**: Delete branch, document why

## Key Finding: Stringification Breaks Type Semantics

**The core problem:** By concatenating all results into a string, we lose type information.

Example: `fn(a, b) { a + b }` 
- Before: Returns Integer (the result of `a + b`)
- After: Returns String (stringified integer)

This breaks:
- Functions returning typed values (integers, arrays, dicts)
- Closures (can't call a string as a function)
- Any code that depends on return types

## Alternative Approaches

### Option A: Only concatenate in "template context"
Keep current semantics for regular blocks, only concatenate when the block is used in a template context (inside tag content, string interpolation).

This is essentially what we already do with `evalTagContents`.

### Option B: Return array instead of string
Instead of stringifying, return an array of non-null results. Let the consumer decide how to handle it.

```go
// Single result → that result
// Multiple results → array of results
// Zero results → NULL
```

This preserves types but changes semantics for multiple-expression blocks.

### Option C: Special "content block" syntax
Introduce new syntax `{| ... |}` for concatenation blocks, keep `{ ... }` for traditional blocks.

### Option D: Explicit concatenation operator
Require explicit concatenation with a new operator like `++` or similar:
```parsley
<div>
  { "Hello" ++ " " ++ name }
</div>
```

## Recommendation

**Option A** (status quo with documentation) is the safest choice:
- No breaking changes
- Already works for templates via `evalTagContents`
- Document clearly that multiple expressions in `{...}` interpolation concatenate

**If we want to explore further**: Option B (array return) is interesting but would require extensive test updates and semantic changes.

## Next Steps

1. Revert experimental changes
2. Update FEAT-022 spec with findings
3. Add to documentation: "In template interpolation, multiple expressions are concatenated"
4. Close this investigation as "valuable learning, no action needed"

## Rollback Plan

If investigation shows this is too breaking:
1. `git checkout main`
2. `git branch -D feat/FEAT-022-block-concatenation`
3. Update FEAT-022.md with findings
4. Consider alternative approaches (Option 1 or 2 from original discussion)

---

## Progress Log

### 2025-12-04: Investigation Complete - Array-Based Approach Implemented

**Decision**: Implemented Option B (array return) instead of string concatenation.

**Implementation**:
- Modified `evalBlockStatement` to collect non-NULL results
- Single result → return directly (preserves type)
- Multiple results → return as array
- Zero results → return NULL
- Declarations (`let`, assignments) return NULL (excluded from results)

**Commits**:
- `379eef5` - Initial investigation
- `38d7858` - Document investigation results
- `0a5fe2f` - Array-based block semantics implementation (all tests pass)
- `7a38c20` - Performance analysis (no measurable impact)
- `3d3ff41` - Final: array-based block semantics with optional parens

**Outcome**: Successfully implemented. Blocks now collect and return multiple expression results as arrays, preserving type information while maintaining consistent block behavior across all contexts (functions, if/else, for loops, top-level scripts).
