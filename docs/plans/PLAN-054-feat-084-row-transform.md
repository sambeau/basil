# PLAN-054: Query Row Transform Implementation

**Feature:** FEAT-084
**Created:** 2026-01-05
**Status:** In Progress

## Overview
Implement post-query row transformation in the Query DSL, allowing `as binding { body }` syntax after projection to transform each result row through Parsley code.

## Implementation Steps

### Phase 1: AST Changes
**File:** `pkg/parsley/ast/ast.go`

1. Add `RowTransform` struct:
```go
type RowTransform struct {
    Token       lexer.Token
    Binding     *RowBinding
    Body        *BlockStatement
}

type RowBinding struct {
    Identifier  *Identifier               // simple: as row
    Destructure *DictDestructuringPattern // pattern: as {a, b, ...rest}
}
```

2. Add `RowTransform *RowTransform` field to `QueryExpression` struct

3. Update `QueryExpression.String()` to include row transform in output

### Phase 2: Parser Changes
**File:** `pkg/parsley/parser/parser.go`

1. In `parseQueryExpression()`, after parsing terminal and before checking for CTE continuation:
   - Check if next token is `AS`
   - If so, call new `parseRowTransform()` function

2. Implement `parseRowTransform()`:
   - Consume `AS` keyword
   - Check if next token is `LBRACE` (destructure) or `IDENT` (simple binding)
   - For destructure: reuse `parseDictDestructuringPattern()`
   - For identifier: parse identifier
   - Parse block statement (the transform body)

### Phase 3: Evaluator Changes
**File:** `pkg/parsley/evaluator/stdlib_dsl_query.go`

1. After query execution (at end of `evalQueryExpression`), check if `node.RowTransform != nil`

2. If transform exists:
   - For Array results: transform each element
   - For single row (Dictionary): transform that row
   - Create new scope for each row
   - Bind row to identifier or destructure into scope
   - Eval transform body
   - Collect results into new Array

3. Helper function `applyRowTransform(result Object, transform *ast.RowTransform, env *Environment) Object`

### Phase 4: Testing
**File:** `pkg/parsley/tests/dsl_query_test.go`

Test cases:
1. Simple binding: `as row { name: row.a + row.b }`
2. Destructure: `as {a, b} { name: a + b }`
3. Destructure with rest: `as {a, ...rest} { ...rest, computed: a }`
4. Block with statements: `as row { let x = ...; { ... } }`
5. Empty result set (should return empty array)
6. Error in transform body (should report row context)

## Files Modified
- `pkg/parsley/ast/ast.go` - Add RowTransform, RowBinding types
- `pkg/parsley/parser/parser.go` - Parse row transform syntax
- `pkg/parsley/evaluator/stdlib_dsl_query.go` - Apply transform to results
- `pkg/parsley/tests/dsl_query_test.go` - Add tests

## Progress Log
- [ ] Phase 1: AST changes
- [ ] Phase 2: Parser changes  
- [ ] Phase 3: Evaluator changes
- [ ] Phase 4: Tests
- [ ] Validation: `make check` passes
