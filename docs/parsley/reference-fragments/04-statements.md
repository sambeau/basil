## 4. Statements

### 4.1 Variable Declarations (`let` and `var`)

Parsley uses Swift-style variable declarations:

- **`let`** — Creates an **immutable** binding (cannot be reassigned)
- **`var`** — Creates a **mutable** binding (can be reassigned)

```parsley
let x = 5                       // Immutable — cannot reassign x
var y = 10                      // Mutable — can reassign y

y = 20                          // OK
// x = 15                       // ERROR: cannot reassign immutable binding 'x'
```

**Important**: Explicit declaration is required. Bare assignments like `x = 5` without a prior `let` or `var` declaration will produce an error.

#### Choosing Between `let` and `var`

Use `let` by default. Only use `var` when you need to reassign the variable:

```parsley
let name = "Alice"              // Won't change — use let
let items = [1, 2, 3]           // Reference won't change — use let

var counter = 0                 // Will be incremented — use var
var result = null               // Will be assigned later — use var
```

#### Shallow Immutability

Immutability applies to the **binding**, not the contents. You cannot reassign a `let` variable, but you can mutate the object it refers to:

```parsley
let arr = [1, 2, 3]
arr[0] = 99                     // OK — mutating contents
// arr = [4, 5]                 // ERROR — reassigning binding

let obj = {x: 1}
obj.x = 2                       // OK — mutating property
obj.y = 3                       // OK — adding property
// obj = {y: 3}                 // ERROR — reassigning binding
```

This matches Swift and JavaScript's `const` behavior.

#### Destructuring Arrays

Both `let` and `var` support destructuring. All bindings from a destructuring pattern share the same mutability:

```parsley
let arr = [1, 2, 3]
let [a, b, c] = arr             // a, b, c are all immutable
let [first, ...rest] = [1, 2, 3, 4]  // first=1, rest=[2,3,4] (both immutable)

var [x, y, z] = [4, 5, 6]       // x, y, z are all mutable
x = 40                          // OK — var binding
y = 50                          // OK
```

#### Destructuring Dictionaries

```parsley
let person = {name: "Bob", age: 25, city: "NYC"}
let {name, age} = person        // name="Bob", age=25 (both immutable)
let {name, ...rest} = person    // name="Bob", rest={age: 25, city: "NYC"}

var {x, y} = {x: 1, y: 2}       // x, y are mutable
x = 10                          // OK
```

---

### 4.2 Assignment

Assignment (`=`) reassigns an existing variable. The variable must have been declared with `var`:

```parsley
var y = 10
y = 20                          // OK — reassign var binding

let x = 5
// x = 10                       // ERROR — cannot reassign let binding
```

#### Property and Index Assignment

Property and index assignments modify the contents of an object, not the binding itself. These work regardless of whether the variable was declared with `let` or `var`:

```parsley
let obj = {a: 1}
obj.b = 2                       // OK — property assignment (mutates contents)

let nums = [1, 2, 3]
nums[0] = 99                    // OK — index assignment (mutates contents)
```

#### Scope and Binding

Parsley uses **lexical scoping** with **closure semantics**:

1. **Variables are visible** in the scope where they're defined and all nested scopes
2. **Inner scopes can modify** outer `var` variables (closures capture by reference)
3. **Inner variables don't leak** to outer scopes

```parsley
var x = 5                       // Must be var if modified by closure
let f = fn() { 
    x = 10                      // Modifies outer x (requires var)
}
f()
x                               // 10 (modified by closure)

let g = fn() {
    let y = 20                  // Local to g
    y
}
g()                             // 20
// y                            // Error: y not defined in outer scope
```

#### Loop Variables and Function Parameters

Loop variables and function parameters are **implicitly immutable**:

```parsley
for (x in [1, 2, 3]) {
    // x = 99                   // ERROR: cannot reassign loop variable
    x * 2
}

let f = fn(a, b) {
    // a = 10                   // ERROR: cannot reassign parameter
    a + b
}
```

---

### 4.3 Return

The `return` keyword explicitly returns a value from a function.

```parsley
let multiply = fn(a, b) {
    return a * b
}
```

**Note**: In Parsley, `return` is usually **redundant**. Functions are expressions, and the last expression's value is automatically returned:

```parsley
let multiply = fn(a, b) {
    a * b                       // Automatically returned
}
```

For early returns (guard patterns), prefer `check...else` which is more idiomatic:

```parsley
// Less idiomatic:
let validate = fn(x) {
    if (x <= 0) { return "must be positive" }
    x * 2
}

// More idiomatic:
let validate = fn(x) {
    check x > 0 else "must be positive"
    x * 2
}
```

---

### 4.4 Export

The `export` keyword makes values available to other files that import the module.

```parsley
export let greeting = "Hello"
export PI = 3.14159
export double = fn(x) { x * 2 }
```

#### Computed Exports

Use `export computed` to create exports that recalculate on each access. This is useful for exposing "live" data like database queries or current timestamps.

**Expression form:**

```parsley
export computed timestamp = @now
export computed count = items.length()
```

**Block form:**

```parsley
export computed activeUsers {
    let query = "SELECT * FROM users WHERE active = true"
    @DB.query(query)
}
```

Computed exports:
- Recalculate on **every access** (never cached)
- Look like regular exports to consumers
- Cannot accept parameters (use functions for that)

**Consumer caching:**

```parsley
import {activeUsers} from @./data.pars

// Each access recalculates
for (user in activeUsers) { user.name }  // Query 1
for (user in activeUsers) { user.email } // Query 2

// Cache by assigning to a variable
let snapshot = activeUsers               // Query 3
for (user in snapshot) { user.name }     // Uses snapshot
for (user in snapshot) { user.email }    // Uses snapshot
```

#### Module System Overview

Parsley modules are simply `.pars` files. Any file can be imported by another, and only `export`ed values are visible to the importer. Non-exported values remain private.

**Example module** (`mathutils.pars`):

```parsley
// Private helper (not exported)
let internalHelper = fn(x) { x * x }

// Public API (exported)
export PI = 3.14159
export square = fn(x) { internalHelper(x) }
export cube = fn(x) { x * x * x }
```

---

### 4.5 Import

The `import` statement loads a module and makes its exports available.

#### Standard Library Imports

```parsley
import @std/math                      // Import as `math.floor()`, etc.
import @std/math as M                 // Import with alias as `M.floor()`
let {floor, ceil} = import @std/math  // Destructure specific exports
```

#### Custom Module Imports

Import your own `.pars` files using path literals:

```parsley
// Import the module from section 4.4
import @./mathutils.pars              // Relative to current file
import @./mathutils.pars as Utils     // With alias
let {PI, square} = import @./mathutils.pars  // Destructure exports

square(4)                             // 16
PI                                    // 3.14159
Utils.cube(3)                         // 27
```

#### Import Paths

| Path Type | Example | Description |
|-----------|---------|-------------|
| Standard lib | `@std/math` | Built-in standard library module |
| Relative | `@./utils.pars` | Relative to current file |
| Project root | `@~/lib/utils.pars` | Relative to project root |

---

