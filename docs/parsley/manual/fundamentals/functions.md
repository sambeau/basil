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

The last expression in a block is the return value — no `return` needed:

```parsley
let double = fn(x) {
    x * 2                        // automatically returned
}
```

### When to Use `return`

In most Parsley code, **`return` is redundant**. The language is expression-oriented: functions, `if`/`else`, and `for` all produce values naturally. Prefer implicit returns for cleaner code.

**Don't reach for `return`** when:
- The function body is a single expression
- You can use `check...else` for guard clauses
- The last expression is already your result

```parsley
// ❌ Unnecessary return
let double = fn(x) { return x * 2 }

// ✅ Implicit return
let double = fn(x) { x * 2 }

// ❌ Return for guards
let process = fn(x) {
    if (!x) { return null }
    x.value
}

// ✅ check...else for guards
let process = fn(x) {
    check x else null
    x.value
}
```

**Do use `return`** for early exit from inside loops — this is its primary purpose:

```parsley
let contains = fn(arr, value) {
    for (item in arr) {
        if (item == value) {
            return true          // exits the function, not just the loop
        }
    }
    false
}
```

Note that `stop` exits the loop but continues the function; `return` exits the function entirely. When you need to bail out of a loop and return from the enclosing function, `return` is the right tool.

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
    {first: first, rest: rest}
}
process([10, 20, 30])            // {first: 10, rest: [20, 30]}
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

Parsley checks arity strictly. A function must be called with exactly as many arguments as it declares parameters — too many and too few are both errors, reported at the call site:

```parsley
let f = fn(a, b) { a + b }
f(1, 2)                          // 3
f(1, 2, 3)                       // error: `f` expects 2 arguments, got 3
f(1)                             // error: `f` expects 2 arguments, got 1
```

Parsley does not support default parameter values. Use `??` in the body when a value is optional:

```parsley
let greet = fn(name) { "Hello, " + (name ?? "world") }
greet(null)                      // "Hello, world"
```

A destructuring parameter counts as one parameter, however many names it binds — its *contents* are still lenient, so missing keys bind to `null`:

```parsley
let f = fn({a, b}) { a }
f({a: 1})                        // 1  (one argument; b is null)
```

Built-in functions, methods, and user-defined functions all enforce arity the same way, raising an `arity` error.

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
