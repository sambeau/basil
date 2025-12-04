# FEAT-024: Print Function

**Status:** Proposed  
**Created:** 2025-12-04  
**Author:** AI Assistant  
**Depends on:** FEAT-023 (Structured Error Objects)

## Summary

Add a `print` function that outputs values to the result stream rather than the dev log, allowing string output in contexts where bare expressions aren't allowed by the grammar.

## Motivation

Currently:
- `log()` outputs to the development log (stderr/console), useful for debugging
- Bare strings in blocks work: `for x in arr { x }` returns the values
- But some contexts don't allow bare expressions, e.g., `for (1..10) { "hello" }` is parsed as a dictionary

Users need a way to explicitly add values to the output stream, particularly in:
- Loop bodies where the block syntax is ambiguous
- Conditional branches
- Function bodies that want to emit multiple values

## Proposed Design

### Function Signature

```parsley
print(value)       // Adds value to result stream, returns null
print(v1, v2, ...) // Adds multiple values, returns null
```

### Behavior

1. **Adds to result stream**: Unlike `log()`, `print()` contributes to the evaluated result
2. **Returns null**: The function itself returns null (side-effect only)
3. **Multiple arguments**: Each argument is added separately to the result stream
4. **Type handling**: All types are accepted; converted to their natural output form

### Examples

```parsley
// Current workaround (when it works)
for x in 1..3 { x }  // Returns [1, 2, 3]

// With print - explicit about intent
for _ in 1..3 { print("hello") }  // Returns ["hello", "hello", "hello"]

// Multiple values
print("a", "b", "c")  // Returns ["a", "b", "c"]

// In conditionals
if condition {
    print("yes")
} else {
    print("no")
}
```

### Difference from `log()`

| Function | Output Destination | Returns | Use Case |
|----------|-------------------|---------|----------|
| `log()` | Dev log (stderr) | null | Debugging, tracing |
| `print()` | Result stream | null | Building output |

## Implementation Notes

### Result Stream

The evaluator needs a mechanism to accumulate print output. Options:
1. **Environment-based**: Store print buffer in environment
2. **Return-based**: Special return type that accumulates
3. **Context-based**: Pass accumulator through evaluation context

### Integration Points

- `pkg/parsley/evaluator/builtins.go` - Add `print` builtin
- `pkg/parsley/evaluator/evaluator.go` - Result stream mechanism
- `pkg/parsley/object/object.go` - May need result accumulator type

## Open Questions

1. Should `print()` with no arguments be allowed (no-op)?
2. Should there be a `println()` variant that adds newlines?
3. How does `print()` interact with template/HTML output in Basil handlers?
4. Should `print()` return the printed value(s) instead of null for chaining?

## Related

- `log()` builtin - existing debug output function
- FEAT-023 - Structured errors (should be implemented first)
- Block expression semantics in parser

## Notes

This feature was identified during FEAT-023 error message design - improved error hints wanted to suggest `print` as an alternative to bare expressions in ambiguous block contexts.
