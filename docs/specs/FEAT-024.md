# FEAT-024: Print Function

**Status:** Proposed  
**Created:** 2025-12-04  
**Author:** AI Assistant  
**Depends on:** None

## Summary

Add `print` and `println` functions that output values to the result stream rather than the dev log, allowing string output in contexts where bare expressions aren't allowed by the grammar.

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

### Function Signatures

```parsley
print(value)       // Adds value to result stream, returns null
print(v1, v2, ...) // Adds multiple values, returns null

println(value)     // Adds value + newline to result stream, returns null
println()          // Adds just a newline, returns null
```

### Behavior

1. **Adds to result stream**: Unlike `log()`, `print()` contributes to the evaluated result
2. **Returns null**: The function itself returns null (side-effect only)
3. **Multiple arguments**: Each argument is added separately to the result stream
4. **Type handling**: All types accepted; converted via their default string representation
5. **UTF-8 throughout**: No escaping - raw UTF-8 output (Parsley is Go, so UTF-8 native)
6. **println**: Same as print but appends `\n` after all arguments

### Examples

```parsley
// Current workaround (when it works)
for x in 1..3 { x }  // Returns [1, 2, 3]

// With print - explicit about intent
for _ in 1..3 { print("hello") }  // Returns ["hello", "hello", "hello"]

// Multiple values
print("a", "b", "c")  // Adds "a", "b", "c" to result stream

// With newlines
println("Line 1")
println("Line 2")

// In conditionals
if condition {
    print("yes")
} else {
    print("no")
}

// UTF-8 just works
print("Hello, 世界! 🎉")
```

### Difference from `log()`

| Function | Output Destination | Returns | Use Case |
|----------|-------------------|---------|----------|
| `log()` | Dev log (stderr) | null | Debugging, tracing |
| `print()` | Result stream | null | Building output |
| `println()` | Result stream + newline | null | Building output with line breaks |

### No printf

Parsley already has string interpolation, so `printf` is unnecessary:

```parsley
let name = "world"
print("Hello, {name}!")  // Interpolation handles formatting
```

Format specifiers (like `%.2f`) are not provided. Instead, objects are responsible for their own string representation (see Future Work).

### Escaping

`print` does **no escaping**. Context-specific escaping is handled elsewhere:
- HTML escaping: template layer (when using `{value}` in HTML)
- URL encoding: `url.encode()` function
- JSON escaping: `json` module
- SQL escaping: `db` module parameterized queries

## Implementation Notes

### Result Stream

The evaluator needs a mechanism to accumulate print output. Options:
1. **Environment-based**: Store print buffer in environment
2. **Return-based**: Special return type that accumulates
3. **Context-based**: Pass accumulator through evaluation context

### Integration Points

- `pkg/parsley/evaluator/builtins.go` - Add `print` and `println` builtins
- `pkg/parsley/evaluator/evaluator.go` - Result stream mechanism
- `pkg/parsley/object/object.go` - May need result accumulator type

## Resolved Questions

1. **Should there be a `println()` variant?** → Yes, adds newline after output
2. **Should `print()` return the value for chaining?** → No, returns null (side-effect only)
3. **Do we need printf?** → No, use string interpolation instead
4. **Do we need escaping?** → No, UTF-8 raw output; escaping is context-specific elsewhere
5. **`print()` with no arguments?** → Error (use `println()` for bare newline)
6. **Interaction with Basil HTML handlers?** → `print` is raw, no magic; HTML escaping is template layer's job
7. **Complex types without toString?** → Output as dictionary representation

## Default String Representation by Type

All types have a defined string representation for `print()` and `{interpolation}`:

| Type | Default | Notes |
|------|---------|-------|
| **String** | `hello` | As-is, no quotes |
| **Integer** | `42` | Standard decimal |
| **Float** | `3.14` | Minimal precision (no trailing zeros) |
| **Boolean** | `true` / `false` | Lowercase |
| **Null** | *(empty string)* | Silent in output; use `value ?? "N/A"` for explicit fallback |
| **Array** | `[1, 2, 3]` | JSON-style |
| **Dictionary** | `{a: 1, b: 2}` | Parsley-style (unquoted keys) |
| **Function** | `<function name>` | Name only |
| **Builtin** | `<builtin name>` | Name only |
| **Table** | `<Table: 5 rows, 3 cols>` | Summary |
| **Error** | `[ERR-001] message` | Code + message |
| **Date** | `2025-12-04` | ISO 8601 |
| **Time** | `14:30:00` | ISO 8601 with seconds |
| **DateTime** | `2025-12-04T14:30:00` | ISO 8601 |
| **Duration** | `2h30m` | Human-readable |
| **Regex** | `/pattern/flags` | Literal form |
| **Range** | `1..10` | Literal form (not expanded) |
| **HTML Element** | `<div>...</div>` | Rendered HTML |
| **HTTP Response** | `<Response: 200 OK>` | Summary |
| **File Handle** | `<File: /path/to.txt>` | Path |
| **DB Result** | `<DBResult: 3 rows>` | Summary |
| **Module** | `<Module: std/table>` | Name |

### Null Handling

Null outputs as empty string in `print()` and `{interpolation}`. This matches user expectations in web/text contexts where "null" is meaningless to end users.

For explicit null representation:
```parsley
print(value ?? "N/A")     // Custom fallback
print(value ?? "missing") // Or any string
log(value)                // Shows "null" in dev log
```

This is an intentional design choice matching template language conventions (Jinja2, Handlebars, etc.).

## Future Work

### Object Formatting Protocol (separate FEAT)

Define standard methods for object string representation:

| Method | Purpose | Example for Number |
|--------|---------|-------------------|
| *(default)* | What goes in `{value}` and `print()` | `12.34` |
| `.repr()` | Debug/inspection | `Number(12.34)` |
| `.json()` | JSON serialization | `12.34` |
| `.short()` | Abbreviated | `12` |
| `.full()` | Complete/precise | `12.340000000000` |

This allows users to control formatting via methods rather than format specifiers:
```parsley
print("Price: {price.fixed(2)}")  // Method call, not %specifier
```

## Related

- `log()` builtin - existing debug output function
- Block expression semantics in parser

## Notes

This feature was identified during FEAT-023 error message design - improved error hints wanted to suggest `print` as an alternative to bare expressions in ambiguous block contexts.
