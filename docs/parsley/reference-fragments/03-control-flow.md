## 3. Control Flow

### 3.1 If Expression

`if` is an **expression** that returns a value, similar to the ternary operator (`? :`) in C-family languages. Unlike imperative if statements, Parsley's `if` always produces a result.

#### Compact Form (Ternary Style)

When you use parentheses around the condition, you can omit the braces for single expressions:

```parsley
let age = 20
let status = if (age >= 18) "adult" else "minor"
```

#### Block Form

When you omit parentheses, you must use braces:

```parsley
let status = if age >= 18 { "adult" } else { "minor" }
```

**Syntax Rule**: Either parentheses `()` around the condition OR braces `{}` around the body are required—the parser needs one to know where the condition ends.

```parsley
if (cond) expr else other       // OK: parens delimit condition
if cond { expr } else { other } // OK: braces delimit body
if cond expr else other         // ERROR: ambiguous
```

#### If-Else-If Chain

```parsley
let score = 75
if (score >= 90) {
    "A"
} else if (score >= 80) {
    "B"
} else if (score >= 70) {
    "C"
} else {
    "F"
}
```

---

### 3.2 For Expression

`for` is an **expression** that returns an array (like `map` in functional languages). This is fundamentally different from imperative for loops in most languages.

**Key behavior**: 
- Each iteration's result is collected into the output array
- `null` results are automatically filtered out (implicit filter)
- Use `stop` to break early, `skip` to skip an iteration

```parsley
let nums = [1, 2, 3, 4, 5]
for (n in nums) { n * 2 }       // [2, 4, 6, 8, 10]
```

#### Map Pattern

Transform every element:

```parsley
let names = ["alice", "bob", "charlie"]
for (name in names) { name.toUpper() }
// ["ALICE", "BOB", "CHARLIE"]
```

#### Filter Pattern

Return `null` (or use `skip`) to exclude elements. Because `for` automatically filters out `null` values, you can use an `if` without `else` to filter:

```parsley
let nums = [1, 2, 3, 4, 5, 6]
for (n in nums) { if (n % 2 == 0) n }   // [2, 4, 6] (odds return null, filtered out)
```

#### Map + Filter Combined

Transform and filter in one pass:

```parsley
let nums = [1, 2, 3, 4, 5, 6]
for (n in nums) { 
    if (n % 2 == 0) n * 10 
}
// [20, 40, 60] (filter evens, then multiply by 10)
```

#### With Index

Use two variables to get both index and value:

```parsley
for (i, n in nums) { `{i}: {n}` }
// ["0: 1", "1: 2", "2: 3", "3: 4", "4: 5"]
```

#### With Range

```parsley
for (x in 1..3) { x * x }       // [1, 4, 9]
```

#### Iterating Dictionaries

For dictionaries, the first variable is the key, second is the value:

```parsley
let person = {name: "Alice", age: 30}
for (k, v in person) { `{k}={v}` }  // ["name=Alice", "age=30"]
```

---

### 3.3 Loop Control

Parsley provides two keywords for controlling loop execution:

- **`stop`** — Exit the loop immediately (like `break` in C, Java, JavaScript, Python, Ruby)
- **`skip`** — Skip to the next iteration (like `continue` in those languages)

If you're familiar with C-family or Python loops, `stop` = `break` and `skip` = `continue`. Note that stop and skip are subtly different as ``for`` generates an array.

```parsley
// stop: exit loop early
let firstThree = for (x in 1..10) {
    if (x > 3) stop
    x
}
// [1, 2, 3]

// skip: skip this iteration
let evens = for (x in 1..6) {
    if (x % 2 != 0) skip
    x
}
// [2, 4, 6]
```

**Note**: When used with `if`, both `stop` and `skip` can be written without braces:

```parsley
for x in 1..10 {
    if (x > 5) stop     // No braces needed
    x
}\
// [1, 2, 3, 4, 5]
```

---

### 3.4 Try Expression

The `try` expression captures errors as values instead of stopping execution. It wraps the result in a dictionary with `result` and `error` fields. The `error` slot is a dictionary with at least `message` and `code` keys (or `null` on success).

```parsley
let safeDivide = fn(a, b) {
    check b != 0 else fail("division by zero")
    a / b
}

let result = try safeDivide(10, 0)
// {result: null, error: {message: "division by zero", code: "USER-0001"}}

let result = try safeDivide(10, 2)
// {result: 5, error: null}
```

#### The `fail` Function

Use `fail(message)` or `fail(dict)` to create a catchable error. Unlike runtime errors, `fail` produces a "value-class" error that can be captured by `try`. This is typically used with `check...else` for validation:

```parsley
// String form — wrapped in {message: ..., code: "USER-0001"}
let validate = fn(email) {
    check email.includes("@") else fail("invalid email")
    email
}

let result = try validate("bad-email")
// {result: null, error: {message: "invalid email", code: "USER-0001"}}

// Dictionary form — must have a string "message" key
fail({message: "Out of stock", code: "NO_STOCK", status: 400})
```

---

### 3.5 Check Guard

`check` is a guard statement for early returns. If the condition is false, the function immediately returns the `else` value instead of continuing execution. This is cleaner than nested `if` statements for validation.

```parsley
let validate = fn(x) {
    check x > 0 else "must be positive"
    check x < 100 else "must be less than 100"
    x * 2
}
validate(5)                     // 10
validate(-1)                    // "must be positive"
validate(200)                   // "must be less than 100"
```

#### How `else` Works

The `else` clause specifies what to return when the check fails:

- **Return a value**: `check x > 0 else "error message"` — returns the string
- **Return null**: `check x > 0 else null` — returns null
- **Throw error**: `check x > 0 else fail("error")` — creates a catchable error (use with `try`)

```parsley
let process = fn(data) {
    check data else null        // Early return null if data is falsy
    check data.valid else fail("invalid data")  // Throw catchable error
    data.value
}
```

**Tip**: Prefer `check...else` over `return` for guard patterns. It makes the intent clearer and reads as "check this condition, else return early."

---

### 3.6 With Expression

`with` creates a scoped context where dictionary fields become directly accessible as variables. This reduces repetition when working with structured data.

```parsley
let person = {name: "Alice", age: 30}
with person {
    `{name} is {age} years old`
}
// "Alice is 30 years old"
```

#### Syntax

Parentheses are optional:

```parsley
with {x: 10, y: 20} { x + y }     // 30
with ({x: 10, y: 20}) { x + y }   // 30 (parentheses optional)
```

#### Use Cases

**Reduce repetition in templates:**

```parsley
let data = {title: "Welcome", body: "Hello!"}
with data {
    <article>
        <h1>title</h1>
        <p>body</p>
    </article>
}
```

**Access nested structure cleanly:**

```parsley
let order = {customer: "Alice", total: 99.99}
with order {
    `Order for {customer}: ${total}`
}
// "Order for Alice: $99.99"
```

#### Scope

Dictionary keys shadow outer variables inside the block:

```parsley
let name = "Outer"
with {name: "Inner"} { name }   // "Inner"
name                             // "Outer" (unchanged)
```

> ⚠️ `with` only works with dictionaries. Passing a non-dictionary is a runtime error.

---

