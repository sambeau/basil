---
id: FEAT-022
title: "Block Concatenation Semantics"
status: investigation
priority: high
created: 2025-12-04
author: "@human"
---

# FEAT-022: Block Concatenation Semantics

## Summary

Investigate changing Parsley's code block semantics so that all non-declaration expressions are concatenated and returned as a single result, rather than only returning the last expression. This would make Parsley's functional nature more explicit and reduce confusion for JavaScript developers who expect either explicit `return` statements or clear expression-based semantics.

## Problem Statement

Parsley looks like JavaScript (imperative, statement-based) but behaves like a functional language (expression-based, implicit return of last value). This creates cognitive dissonance:

1. No delimiter between declarations and the "return expression"
2. Values "fall off the bottom" without explicit return
3. Multiple expressions in a block silently discard all but the last
4. JavaScript developers expect `return` statements

## Proposed Solution

Make all code blocks behave like `for` loops: concatenate all non-null expression results into a single output. Declarations (`let`, assignment) would return `null` and be excluded from concatenation.

### Example - Current Behavior

```parsley
let x = 5
<p>First</p>
<p>Second</p>
// Returns: "<p>Second</p>" (only last value)
```

### Example - Proposed Behavior

```parsley
let x = 5        // → null (excluded)
<p>First</p>     // → "<p>First</p>"
<p>Second</p>    // → "<p>Second</p>"
// Returns: "<p>First</p><p>Second</p>" (concatenated)
```

## User Story

As a **Parsley developer**, I want **all expressions in a code block to contribute to the output** so that **I don't accidentally lose content and the language behaves consistently with its functional nature**.

## Acceptance Criteria (Investigation)

- [ ] Document what currently returns values vs. side-effects
- [ ] Identify which constructs should return `null` (be excluded)
- [ ] Create experimental branch with modified behavior
- [ ] Run test suite and document what breaks
- [ ] Assess performance implications
- [ ] Make go/no-go recommendation

## Design Decisions

### What Returns NULL (Excluded from Concatenation)
- `let` statements — declarations, not expressions
- Assignment statements (`x = 5`) — side-effect, not expression
- `import` statements — module loading
- `log()` calls — side-effect only (already returns null)

### What Gets Concatenated
- Literals (strings, numbers, booleans)
- Tag expressions (`<div>...</div>`)
- Function call results
- `if`/`else` expression results
- `for` loop results (already returns array)
- Ternary expressions
- Any other expression

### Consistent Block Behavior
All code blocks should behave identically:
- Function bodies
- `if`/`else` branches
- `for` loop bodies
- Top-level scripts

### String Concatenation
When concatenating results:
- Arrays are flattened
- All values are stringified
- Empty strings and `null` are excluded

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `pkg/parsley/evaluator/evaluator.go` — `evalBlockStatement`, `evalInterpolationBlock`
- `pkg/parsley/evaluator/evaluator.go` — `LetStatement`, `AssignmentStatement` return values
- `pkg/parsley/evaluator/evaluator.go` — `evalIfExpression` 
- `pkg/parsley/evaluator/evaluator.go` — `evalForExpression` (reference implementation)

### Current Behavior Analysis

**evalBlockStatement** (line ~7166):
```go
func evalBlockStatement(block *ast.BlockStatement, env *Environment) Object {
    var result Object
    for _, statement := range block.Statements {
        result = Eval(statement, env)
        // ... error/return handling
    }
    return result  // Only returns LAST result
}
```

**evalForExpression** (line ~8422):
```go
// Maps over elements, collects non-null results into array
result := []Object{}
for idx, elem := range elements {
    // ... evaluate body
    if evaluated != NULL {
        result = append(result, evaluated)
    }
}
return &Array{Elements: result}
```

### Performance Considerations

1. **Memory**: Collecting results into array before concatenation
   - Mitigation: Use `strings.Builder` for string-heavy blocks
   - Most blocks are small (< 10 expressions)

2. **Allocation**: Creating intermediate arrays
   - Mitigation: Pre-allocate based on statement count
   - Mitigation: Fast path for single-expression blocks

3. **Flattening**: Nested arrays from `for` loops
   - Mitigation: Flatten during collection, not after

### Edge Cases & Constraints

1. **Return statements** — Should bypass concatenation and return immediately
2. **Error propagation** — Errors should still short-circuit
3. **Empty blocks** — Should return `null` or empty string
4. **Single expression** — Fast path, no array allocation needed
5. **REPL behavior** — May need adjustment for interactive use

## Investigation Plan

1. Create branch `feat/FEAT-022-block-concatenation`
2. Modify `LetStatement` and `AssignmentStatement` to return `NULL`
3. Modify `evalBlockStatement` to collect and concatenate results
4. Run test suite, document failures
5. Fix obvious issues, note semantic changes
6. Performance benchmark before/after
7. Document findings and recommendation

## Related

- Plan: `docs/plans/FEAT-022-plan.md`
- Similar: `for` loop already implements concatenation semantics
