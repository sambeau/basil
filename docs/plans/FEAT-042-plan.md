# FEAT-042 Implementation Plan: Explicit `...rest` for Array Destructuring

## Overview
Change array destructuring to require explicit `...rest` syntax for collecting remaining elements, matching JavaScript/TypeScript and Parsley's dict destructuring.

## Current State
- Array: Last variable implicitly slurps remaining elements
- Dict: Requires explicit `...rest` syntax
- Inconsistent and surprising for users

## Target State  
- Array: Requires explicit `...rest` syntax (like JS/TS)
- Dict: Already has `...rest` (no change)
- Consistent behavior across both

## Implementation Tasks

### Task 1: Create ArrayDestructuringPattern AST node
**File:** `pkg/parsley/ast/ast.go`

Add new type similar to `DictDestructuringPattern`:
```go
// ArrayDestructuringPattern represents array destructuring like [a, b, ...rest]
type ArrayDestructuringPattern struct {
    Token  lexer.Token    // the '[' token
    Names  []*Identifier  // the identifiers to extract
    Rest   *Identifier    // optional rest identifier (for ...rest)
}
```

Update these structs to use new pattern type:
- `LetStatement.Names` → `LetStatement.ArrayPattern *ArrayDestructuringPattern`
- `AssignmentStatement.Names` → `AssignmentStatement.ArrayPattern *ArrayDestructuringPattern`
- `ReadStatement.Names` → `ReadStatement.ArrayPattern *ArrayDestructuringPattern`
- `FetchStatement.Names` → `FetchStatement.ArrayPattern *ArrayDestructuringPattern`
- `FunctionParameter.ArrayPattern []*Identifier` → `FunctionParameter.ArrayPattern *ArrayDestructuringPattern`

### Task 2: Update parser for array destructuring
**File:** `pkg/parsley/parser/parser.go`

Create `parseArrayDestructuringPattern()` function similar to `parseDictDestructuringPattern()`:
- Parse `[ident, ident, ...rest]` syntax
- Validate `...rest` is at end of pattern
- Handle empty pattern error
- Handle `_` placeholder

Update all places that parse array patterns:
- `parseLetStatement()` (line ~460)
- `parseAssignmentStatement()` 
- `parseFunctionParameter()` (line ~1680)
- Any other locations using `[a, b]` syntax

### Task 3: Update evaluator
**File:** `pkg/parsley/evaluator/evaluator.go`

Update `evalDestructuringAssignment()` (line ~9818):
- Remove implicit slurping behavior
- Only collect remaining if `pattern.Rest != nil`
- Assign empty array to Rest if no remaining elements

Update `evalArrayDestructuringForParam()` (line ~8681):
- Same changes as above

### Task 4: Update tests
**Files:** `pkg/parsley/tests/arrays_test.go`, `pkg/parsley/tests/function_destructuring_test.go`

Update existing tests:
- `"destructuring with rest collects extra elements"` should test explicit `...rest`
- Add new tests for:
  - Without rest: extras ignored
  - With rest at end: collects remaining
  - Rest with empty remaining: gets empty array
  - `[...rest]` alone: gets all elements
  - `[_, ...rest]`: skips first
  - Error: `[a, ...rest, b]` rest not at end

### Task 5: Update documentation
**Files:** 
- `docs/parsley/reference.md`
- `docs/parsley/CHEATSHEET.md`
- `docs/parsley/CHANGES.md`

Document:
- New `...rest` syntax for arrays
- Breaking change from implicit slurping
- Examples of new behavior

## Execution Order
1. Task 1 (AST) - foundation
2. Task 2 (Parser) - enable new syntax  
3. Task 3 (Evaluator) - implement new behavior
4. Task 4 (Tests) - verify correctness
5. Task 5 (Docs) - communicate changes

## Risks & Mitigations
- **Breaking change**: Accepted for pre-alpha. Document in CHANGES.md
- **Complexity**: Follow existing dict pattern closely to minimize errors
- **Test coverage**: Use comprehensive edge case tests

## Progress Log
- 2025-12-08: Task 1 complete - Created ArrayDestructuringPattern AST type, updated all statement types
- 2025-12-08: Task 2 complete - Added parseArrayDestructuringPattern(), updated parser call sites
- 2025-12-08: Task 3 complete - Added evalArrayPatternAssignment() and evalArrayPatternForParam()
- 2025-12-08: Task 4 complete - Fixed failing test, added new test cases
- 2025-12-08: Task 5 complete - Updated reference.md, CHEATSHEET.md, CHANGES.md
- 2025-12-08: Implementation complete, all tests pass (`make check`)
- [ ] Task 1: AST changes
- [ ] Task 2: Parser changes
- [ ] Task 3: Evaluator changes
- [ ] Task 4: Test updates
- [ ] Task 5: Documentation updates
- [ ] Final: `make check` passes
