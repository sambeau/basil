---
id: man-pars-variables
title: Variables & Binding
system: parsley
type: fundamental
name: variables
created: 2026-02-05
version: 0.2.0
author: Basil Team
keywords:
  - variable
  - let
  - var
  - assignment
  - binding
  - destructuring
  - scope
  - closure
---

# Variables & Binding

Parsley has two variable declaration keywords:

- **`let`** — Creates an **immutable** binding (cannot be reassigned)
- **`var`** — Creates a **mutable** binding (can be reassigned)

Variables are lexically scoped, closures capture by reference, and destructuring works on both arrays and dictionaries.

```parsley
let name = "Alice"              // immutable — cannot reassign
var count = 0                   // mutable — can reassign
count = count + 1               // OK
// name = "Bob"                 // Error: cannot reassign immutable binding
let [x, y] = [1, 2]            // array destructuring
let {age} = {age: 30}          // dictionary destructuring
```

## `let` — Immutable Binding

The `let` keyword declares an **immutable** variable. Once assigned, the binding cannot be changed:

```parsley
let x = 5
let greeting = "hello"
let items = [1, 2, 3]

// x = 10                       // Error: cannot reassign immutable binding 'x'
```

`let` can be used again on the same name — this **shadows** (creates a new binding) rather than reassigning:

```parsley
let x = 5
let x = 10                     // shadows the previous x (new binding)
x                               // 10
```

## `var` — Mutable Binding

The `var` keyword declares a **mutable** variable that can be reassigned:

```parsley
var count = 0
count = count + 1               // OK — var allows reassignment
count = count + 1
count                           // 2
```

Use `var` when you need to update a value across multiple statements:

```parsley
var count = 0
count = count + 1
count = count + 1
count                           // 2
```

Or use `.reduce()` for accumulation (preferred over mutation):

```parsley
[1, 2, 3, 4, 5].reduce(fn(acc, n) { acc + n }, 0)  // 15
```

## Choosing Between `let` and `var`

- **Default to `let`** — immutability makes code easier to reason about
- **Use `var`** when you need to reassign — loops, counters, accumulators
- The compiler will tell you if you try to reassign a `let` binding

```parsley
// Prefer let for values that don't change
let name = "Alice"
let config = {debug: true, port: 8080}

// Use var when you need mutation
var attempts = 0
var buffer = []
```

### Property & Index Assignment

You can assign directly to dictionary keys and array indices:

```parsley
let obj = {a: 1}
obj.b = 2                       // {a: 1, b: 2}

let nums = [1, 2, 3]
nums[0] = 99                    // [99, 2, 3]
```

## Destructuring

### Arrays

Extract elements by position. Use `...rest` to capture remaining elements:

```parsley
let [a, b, c] = [1, 2, 3]      // a=1, b=2, c=3
let [first, ...rest] = [1, 2, 3, 4]  // first=1, rest=[2, 3, 4]
let [_, second] = [10, 20]     // discard first with _
```

### Dictionaries

Extract values by key name. Use `...rest` to capture remaining keys:

```parsley
let person = {name: "Bob", age: 25, city: "NYC"}
let {name, age} = person        // name="Bob", age=25
let {name, ...rest} = person    // name="Bob", rest={age: 25, city: "NYC"}
```

> ⚠️ Dictionary destructuring binds to the **key name** — there's no renaming syntax like JavaScript's `{name: alias}`. The variable name must match the key.

## Scope

Parsley uses **lexical scoping**. Variables are visible in the scope where they're defined and all nested scopes. Inner variables don't leak outward:

```parsley
let x = "outer"
if (true) {
    let y = "inner"
    x                           // "outer" (visible from parent scope)
    y                           // "inner"
}
x                               // "outer"
// y is not defined here
```

## Closures

Functions capture variables from their enclosing scope **by reference** — modifications to outer variables are visible to and from the closure:

```parsley
let makeCounter = fn() {
    let count = 0
    fn() {
        count = count + 1
        count
    }
}
let c = makeCounter()
c()                             // 1
c()                             // 2
c()                             // 3
```

A direct example of capture-by-reference:

```parsley
let x = 5
let f = fn() { x = 10 }
f()
x                               // 10 (modified by the closure)
```

## Key Differences from Other Languages

- **`let` IS immutable:** Unlike JavaScript, Parsley's `let` creates an immutable binding — use `var` for mutable bindings
- **`var` is mutable:** Like Swift, `var` allows reassignment while `let` does not
- **No `const` keyword:** Use `let` for constants — it's already immutable
- **Shadowing with `let`:** Using `let` on an existing name creates a new binding (shadowing), not reassignment
- **No rename in dict destructuring:** `let {name: alias} = obj` is a parse error — use a separate assignment if you need a different name
- **`_` is a discard:** In array destructuring, `_` signals that you're intentionally ignoring a value

## See Also

- [Functions](functions.md) — function definitions and closures
- [Control Flow](control-flow.md) — block scoping with `if` and `for`
- [Operators](operators.md) — spread operator (`...`) in detail
- [Modules](modules.md) — `export` and `import` for sharing bindings
- [Dictionaries](../builtins/dictionary.md) — destructuring dictionaries