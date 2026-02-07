---
id: man-pars-functions
title: Functions
system: parsley
type: fundamentals
name: functions
created: 2026-02-05
version: 0.2.0
author: Basil Team
keywords:
  - function
  - fn
  - closure
  - callback
  - first-class
  - destructuring
  - this
  - return
  - component
---

# Functions

Functions in Parsley are first-class values created with `fn`. They are always anonymous — naming happens through `let` binding. The body is a block, and the last expression is the implicit return value.

## Basic Syntax

```parsley
let double = fn(x) { x * 2 }
let add = fn(a, b) { a + b }
let hello = fn() { "hello" }
let thunk = fn { 99 }           // parens optional when no parameters

double(5)                        // 10
add(3, 4)                        // 7
thunk()                          // 99
```

## Return Values

The last expression in a block is the return value. Use `return` for early exit:

```parsley
let abs = fn(x) {
    if (x < 0) { return -x }
    x
}
abs(-5)                          // 5
abs(3)                           // 3
```

## Parameter Destructuring

Function parameters can destructure dictionaries and arrays directly:

```parsley
// Dictionary destructuring
let greet = fn({name, age}) {
    name + " is " + age
}
greet({name: "Alice", age: 30})  // "Alice is 30"

// Array destructuring with rest
let process = fn([first, ...rest]) {
    log(first)                   // 10
    log(rest)                    // [20, 30]
}
process([10, 20, 30])
```

This is the standard pattern for components — a single dict parameter with named fields:

```parsley
let Card = fn({title, body}) {
    <div class="card">
        <h2>title</h2>
        <p>body</p>
    </div>
}
<Card title="Hello" body="World"/>
```

When a component tag has children, they arrive as `contents`:

```parsley
let Wrap = fn({contents}) {
    <div class="wrap">contents</div>
}
<Wrap><p>"inner"</p></Wrap>
```

## Closures

Functions capture their enclosing environment by reference:

```parsley
let make_counter = fn() {
    let count = 0
    fn() {
        count = count + 1
        count
    }
}
let c = make_counter()
c()                              // 1
c()                              // 2
c()                              // 3
```

## `this` Binding

When a function is stored as a dictionary value and called as a method, `this` is automatically bound to the dictionary:

```parsley
let user = {
    name: "Alice",
    greet: fn() { "Hello, " + this.name }
}
user.greet()                     // "Hello, Alice"
```

`this` is only available inside methods called via dot notation. Calling the function directly (not through the dict) won't bind `this`.

## First-Class Usage

Functions are values — pass them to methods, store them in arrays, return them from other functions:

```parsley
[1, 2, 3].map(fn(x) { x * 10 })          // [10, 20, 30]
[1, 2, 3, 4, 5].filter(fn(x) { x > 3 })  // [4, 5]
[1, 2, 3, 4, 5].reduce(fn(acc, x) { acc + x }, 0)  // 15
```

### Immediately Invoked

```parsley
fn() { 42 }()                    // 42
fn(x) { x * 2 }(5)              // 10
```

## Argument Handling

Parsley does not support default parameter values. Missing arguments leave the parameter unbound (using it will cause an "identifier not found" error). Extra arguments are silently ignored.

```parsley
let f = fn(a, b) { a }
f(1, 2, 3)                      // 1  (third arg ignored)
f(1)                             // 1  (b unbound but unused, so no error)
```

> ⚠️ Built-in functions and methods enforce arity strictly and will error on wrong argument counts. User-defined functions do not — they silently accept any number of arguments.

## Key Differences from Other Languages

- **No `function` keyword** — use `fn`.
- **No default parameters** — use `??` inside the body if you need defaults: `let x = arg ?? "default"`.
- **No arrow functions** — `fn(x) { x * 2 }` is the only syntax.
- **No named function declarations** — all functions are anonymous; naming is via `let` or `export`.
- **Implicit return** — the last expression is the return value. `return` is only needed for early exit.
- **`this` is dict-scoped** — not class-based. It's bound when calling a function through dot notation on a dictionary.

## See Also

- [Variables & Binding](variables.md) — `let`, destructuring, scope
- [Control Flow](control-flow.md) — `if`/`else`, `for`, `check`
- [Operators](operators.md) — spread in destructuring
- [Tags](../fundamentals/tags.md) — component pattern with `fn({contents})`
- [Modules](../fundamentals/modules.md) — `export` and `import`
