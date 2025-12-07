---
id: FEAT-042
title: "Require explicit ...rest syntax for array destructuring"
status: implemented
priority: high
created: 2025-12-08
author: "@human"
---

# FEAT-042: Require explicit `...rest` syntax for array destructuring

## Summary
Change array destructuring to require explicit `...rest` syntax to collect remaining elements, matching JavaScript/TypeScript behavior and Parsley's own dictionary destructuring. Currently, the last variable in an array destructuring pattern implicitly "slurps" all remaining elements, which is inconsistent and surprising.

## User Story
As a Parsley developer, I want array destructuring to work consistently with dictionary destructuring and other languages so that I don't encounter surprising implicit behaviors.

## Acceptance Criteria
- [x] `let [a, b] = [1, 2, 3, 4]` assigns `a=1`, `b=2` (extras ignored)
- [x] `let [a, ...rest] = [1, 2, 3, 4]` assigns `a=1`, `rest=[2, 3, 4]`
- [x] `let [a, b, ...rest] = [1, 2, 3]` assigns `a=1`, `b=2`, `rest=[3]`
- [x] `let [a, b, ...rest] = [1, 2]` assigns `a=1`, `b=2`, `rest=[]`
- [x] Rest parameter must be last in pattern (error otherwise)
- [x] Function parameters support same syntax: `fn([a, ...rest]) { ... }`
- [x] Parser produces clear error for `...rest` not at end
- [x] Existing tests updated for new behavior
- [x] Documentation updated (reference.md, CHEATSHEET.md, CHANGES.md)

## Design Decisions
- **Match JS/TS behavior**: The `...rest` syntax is familiar to JavaScript developers and consistent with Parsley's dictionary destructuring
- **Breaking change accepted**: Pre-alpha status means correctness over backward compatibility
- **Extras ignored without rest**: Like JS/TS, extra elements are silently ignored rather than causing an error (unlike Python)

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `pkg/parsley/ast/ast.go` — Added `ArrayDestructuringPattern` type with `Names` and `Rest` fields
- `pkg/parsley/parser/parser.go` — Added `parseArrayDestructuringPattern()` function
- `pkg/parsley/evaluator/evaluator.go` — Added `evalArrayPatternAssignment()` and `evalArrayPatternForParam()`

### Dependencies
- Depends on: None
- Blocks: None

### Edge Cases & Constraints
1. `let [...rest] = [1, 2, 3]` — Valid, `rest=[1, 2, 3]`
2. `let [a, ...rest, b] = arr` — Error: rest must be last
3. `let [a, ..._] = arr` — Valid, discards rest with placeholder
4. `let [_, ...rest] = arr` — Valid, skips first element
5. Empty array with rest: `let [a, ...rest] = []` → `a=null`, `rest=[]`

## Implementation Notes
Implemented 2025-12-08:
- Created `ArrayDestructuringPattern` AST node (similar to `DictDestructuringPattern`)
- Updated `LetStatement`, `AssignmentStatement`, `ReadStatement`, `FetchStatement`, `FunctionParameter` to use new pattern type
- Parser now handles `...rest` syntax at end of array patterns
- Evaluator only collects remaining elements when explicit `Rest` field is present
- Tests updated: `function_destructuring_test.go`, `arrays_test.go`
- Documentation updated: `reference.md`, `CHEATSHEET.md`, `CHANGES.md`
*To be added during implementation*

## Related
- Plan: `docs/plans/FEAT-042-plan.md` (to be created)
- Similar to: Dictionary destructuring `{a, b, ...rest}` implementation
