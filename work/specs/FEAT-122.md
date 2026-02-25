# FEAT-122: Swift-Style Variable Declarations (`let`/`var`)

**Status:** Complete
**Created:** 2025-01-21  
**Author:** AI Assistant  

## Summary

Adopt Swift-style variable declaration semantics:
- `let` = immutable binding (constant)
- `var` = mutable binding (variable)

This is a breaking change that should be completed before v1.0.

## Motivation

### Current Problems

1. **Optional `let` is confusing** — Both `let x = 5` and `x = 5` work identically, making the language feel unfinished.

2. **No immutability support** — All bindings are mutable; no way to declare constants.

3. **`let` semantics are ambiguous** — In JavaScript `let` is mutable, but in mathematics, Swift, and Rust it implies immutability.

### Why Now?

Post-1.0, changing variable declaration syntax would require a breaking 2.0 release. The pre-1.0 window is the right time to get this right.

### Why Swift-Style?

| Approach | `let` means | `const`/`var` means | Used by |
|----------|-------------|---------------------|---------|
| JavaScript | mutable | immutable (`const`) | JS, TS |
| Swift | **immutable** | mutable (`var`) | Swift |
| Rust | immutable | mutable (`let mut`) | Rust |
| Kotlin | — | immutable (`val`), mutable (`var`) | Kotlin |

Swift-style was chosen because:
- **Semantically correct** — "let x = 5" in mathematics is an immutable assumption
- **Clean slate** — We can do it right; no legacy to maintain
- **Proven design** — Swift is a major modern language with this model
- **Clear keywords** — `let` (constant) and `var` (variable) are both explicit

## Specification

### New Syntax

```parsley
let x = 5           // immutable — cannot reassign x
var y = 10          // mutable — can reassign y

y = 20              // OK
x = 15              // ERROR: cannot reassign immutable binding 'x'
```

### Destructuring

```parsley
let [a, b] = [1, 2]         // a and b are immutable
var [x, y] = [1, 2]         // x and y are mutable

let {name, age} = person    // immutable bindings
var {name, age} = person    // mutable bindings
```

### Exports

```parsley
export PI = 3.14159         // immutable (shorthand for export let)
export let PI = 3.14159     // immutable (explicit)
export var counter = 0      // mutable (must be explicit)

export computed timestamp = @now    // immutable binding, re-evaluated on access
```

Bare `export x = ...` is sugar for `export let x = ...` (immutable by default).

### Shallow Immutability

Immutability applies to the **binding**, not the contents:

```parsley
let arr = [1, 2, 3]
arr[0] = 99         // OK — mutating contents
arr = [4, 5]        // ERROR — reassigning binding

let obj = {x: 1}
obj.x = 2           // OK — mutating property
obj = {y: 3}        // ERROR — reassigning binding
```

This matches Swift and JavaScript's `const` behavior.

### Explicit Declaration Required

Implicit declarations become errors:

```parsley
x = 5               // ERROR: use 'let x = 5' or 'var x = 5'
let x = 5           // OK
var x = 5           // OK
```

### Loop Variables

Loop variables are implicitly immutable within each iteration:

```parsley
for (x in [1, 2, 3]) {
    x = 99          // ERROR: cannot reassign loop variable
}

for (i, x in [1, 2, 3]) {
    i = 99          // ERROR: cannot reassign loop index
}
```

### Function Parameters

Function parameters are immutable by default:

```parsley
let f = fn(x) {
    x = 10          // ERROR: cannot reassign parameter
    x + 1
}
```

This is consistent with many languages and prevents accidental parameter mutation.

## Migration

### Phase 1: Add `var` (Non-Breaking)

- Add `VAR` token to lexer
- Parse `var` statements as mutable bindings
- `let` continues to work as mutable (backwards compatible)
- Add deprecation warning for implicit declarations
- Update documentation

### Phase 2: Flip `let` Semantics (Breaking)

- `let` becomes immutable
- Error on reassigning `let` bindings
- Error on implicit declarations
- Provide migration tool: `pars migrate-let-var`

### Migration Tool

The `pars migrate-let-var` command:

1. Parses all `.pars` files in a directory (use `-r` for recursive)
2. Identifies `let` bindings that are reassigned → converts to `var`
3. Identifies implicit declarations → adds `let` (or `var` if later reassigned)
4. Outputs a diff by default, or applies changes with `-w`/`--write`
5. Use `-l` to list files that need migration

Example:
```bash
pars migrate-let-var script.pars           # Show diff of changes
pars migrate-let-var -w script.pars        # Apply changes to file
pars migrate-let-var -l *.pars             # List files that need migration
pars migrate-let-var -r ./src              # Recursively check directory
pars migrate-let-var -r -w ./src           # Recursively migrate all files
```

## Error Messages

### Reassigning Immutable Binding

```
Error: cannot reassign immutable binding 'x'
  --> script.pars:5:1
   |
 3 | let x = 5
   |     - 'x' declared as immutable here
 5 | x = 10
   | ^^^^^^ cannot reassign
   |
   = help: use 'var x = 5' if you need to reassign this binding
```

### Implicit Declaration

```
Error: implicit variable declaration
  --> script.pars:3:1
   |
 3 | x = 5
   | ^^^^^
   |
   = help: use 'let x = 5' for a constant or 'var x = 5' for a variable
```

## Acceptance Criteria

- [x] `var` keyword added to lexer and parser
- [x] `let` bindings are immutable (cannot reassign)
- [x] `var` bindings are mutable (can reassign)
- [x] Implicit declarations produce error
- [x] Destructuring respects `let`/`var` mutability
- [x] `export x = ...` is immutable (sugar for `export let`)
- [x] `export var x = ...` is mutable
- [x] Loop variables are immutable
- [x] Function parameters are immutable
- [x] Clear error messages with hints
- [x] `const` reserved as keyword (error if used as identifier)
- [x] REPL enforces same semantics (no special exceptions)
- [x] Migration tool (`pars migrate-let-var`)
- [x] Documentation updated (reference, cheatsheet, FAQ)
- [x] All existing tests updated or migrated

## Impact Analysis

### Codebase Changes

| File | Changes |
|------|---------|
| `lexer/lexer.go` | Add `VAR` token, add `"var"` and `"const"` to keywords (reserved) |
| `ast/ast.go` | Add `Mutable bool` to `LetStatement` |
| `parser/parser.go` | Parse `var`, track mutability |
| `evaluator/evaluator.go` | Enforce immutability, error on reassign |
| `errors/errors.go` | Add immutability error codes |

### Test Files

- ~133 `.pars` files use `let`
- ~6 files have reassignment patterns needing `var`
- Migration tool can automate most changes

### Documentation

- `docs/parsley/reference.md` — Update §4.1 Let Binding
- `docs/parsley/CHEATSHEET.md` — Update syntax tables
- `docs/guide/faq.md` — Add migration guidance

## Resolved Questions

1. **Should `const` be reserved?** — **Yes.** Reserve `const` as a keyword to prevent user code from using it as an identifier, even though we use `let` for immutability. This avoids future conflicts.

2. **REPL behavior?** — **Keep consistent.** The REPL should enforce the same `let`/`var` semantics as scripts. No special exceptions for convenience.

## References

- [Analysis Report: let/const in Parsley](../reports/LET-CONST-ANALYSIS.md)
- [Swift Language Guide: Constants and Variables](https://docs.swift.org/swift-book/documentation/the-swift-programming-language/thebasics/#Constants-and-Variables)
- [Rust Book: Variables and Mutability](https://doc.rust-lang.org/book/ch03-01-variables-and-mutability.html)