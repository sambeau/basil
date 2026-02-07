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
  - assignment
  - binding
  - destructuring
  - scope
  - closure
---

# Variables & Binding

Parsley uses `let` to declare variables and bare assignment to update them. Variables are lexically scoped, closures capture by reference, and destructuring works on both arrays and dictionaries.

```parsley
let name = "Alice"              // declare with let
name = "Bob"                    // reassign with bare assignment
let [x, y] = [1, 2]            // array destructuring
let {age} = {age: 30}          // dictionary destructuring
```

## `let` Binding

The `let` keyword declares and initialises a variable:

```parsley
let x = 5
let greeting = "hello"
let items = [1, 2, 3]
```

`let` can be used again on the same name — this shadows (replaces) the previous binding:

```parsley
let x = 5
let x = 10                     // shadows the previous x
x                               // 10
```

## Bare Assignment

Omitting `let` reassigns an existing variable. If the name doesn't exist yet, it creates a new binding:

```parsley
let count = 0
count = count + 1               // reassign existing
total = 100                     // creates new binding (no prior let)
```

Both forms work, but `let` is recommended for initial declarations — it signals intent and is easier to spot when reading code.

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

- **`let` is not `const`:** `let` does not make a binding immutable — you can reassign with either `let` or bare assignment. There is no `const` keyword
- **No `var`/`const`/`let` distinction:** Parsley has only `let` and bare assignment — no hoisting, no temporal dead zone
- **Bare assignment creates bindings:** Unlike Python's similar behaviour, this is intentional — `x = 5` works without a prior declaration
- **No rename in dict destructuring:** `let {name: alias} = obj` is a parse error — use a separate assignment if you need a different name
- **`_` is a discard:** In array destructuring, `_` signals that you're intentionally ignoring a value

## See Also

- [Functions](functions.md) — function definitions and closures
- [Control Flow](control-flow.md) — block scoping with `if` and `for`
- [Operators](operators.md) — spread operator (`...`) in detail
- [Modules](modules.md) — `export` and `import` for sharing bindings
- [Dictionaries](../builtins/dictionary.md) — destructuring dictionaries