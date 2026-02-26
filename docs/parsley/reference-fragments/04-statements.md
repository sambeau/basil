## 4. Statements

Statements are the building blocks of Parsley programs.

### 4.1 Variable Declarations

Parsley has two variable binding keywords:

- **`let`** — Immutable binding (cannot be reassigned)
- **`var`** — Mutable binding (can be reassigned)

```parsley
let name = "Alice"        // Immutable
var count = 0             // Mutable

count = count + 1         // OK — var can be reassigned
name = "Bob"              // ERROR — let cannot be reassigned
```

**Shallow immutability**: `let` prevents reassigning the variable, but you can still mutate the contents of arrays and dictionaries.

```parsley
let items = [1, 2, 3]
items[0] = 10             // OK — mutating contents
items = [4, 5, 6]         // ERROR — cannot reassign
```

### 4.2 Destructuring

```parsley
// Array destructuring
let [first, second, ...rest] = [1, 2, 3, 4, 5]
// first = 1, second = 2, rest = [3, 4, 5]

// Dictionary destructuring
let {name, age} = {name: "Alice", age: 30}

// With rename
let {name: userName} = person
```

### 4.3 Assignment

```parsley
var x = 10
x = 20                    // Simple assignment

// Property assignment
person.name = "Bob"

// Index assignment
items[0] = "first"
```

### 4.4 Return

```parsley
fn greet(name) {
    return `Hello, {name}!`
}

// Implicit return — last expression is returned
fn double(x) {
    x * 2
}
```

### 4.5 Export

Make values available to other modules:

```parsley
export PI = 3.14159
export fn square(x) { x * x }

// Computed exports (re-evaluated on each import)
export computed timestamp = now()
```

### 4.6 Import

```parsley
// Import from standard library
let {sin, cos} = import @std/math

// Import from local file
let {helper} = import @./utils.pars

// Import all exports
let math = import @std/math
math.sin(0)
```
