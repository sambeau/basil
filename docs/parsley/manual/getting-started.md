---
id: man-pars-getting-started
title: "Getting Started with Parsley"
system: parsley
type: tutorial
name: getting-started
created: 2026-02-06
version: 0.2.0
author: Basil Team
keywords:
  - tutorial
  - getting started
  - beginner
  - introduction
  - first program
---

# Getting Started with Parsley

This tutorial walks you through Parsley's core concepts with hands-on examples. By the end, you'll be able to write variables, functions, loops, and HTML templates — everything you need to start building with Basil.

> **Prerequisites:** You need `pars` (the Parsley CLI) installed. Run `pars` to start an interactive session, or `pars myfile.pars` to run a script.

---

## Your First Program

Create a file called `hello.pars`:

```parsley
let name = "world"
log("Hello, " + name + "!")
```

Run it:

```
$ pars hello.pars
Hello, world!
```

`let` creates a variable. `log()` prints a value. The `+` operator concatenates strings.

---

## Variables and Expressions

Variables are declared with `let`. Parsley has numbers, strings, booleans, and `null`:

```parsley
let age = 30
let price = 9.99
let active = true
let missing = null
```

Arithmetic works as you'd expect:

```parsley
let x = 10
let y = 3
log(x + y)       // 13
log(x * y)       // 30
log(x / y)       // 3.333...
log(x % y)       // 1 (remainder)
```

**See:** [Variables & Binding](fundamentals/variables.md) · [Types](fundamentals/types.md) · [Operators](fundamentals/operators.md)

---

## Strings

Strings can be written with double quotes, single quotes, or backticks:

```parsley
let s1 = "double quoted"
let s2 = 'single quoted — @{1 + 1} interpolation'
let s3 = `backtick — {1 + 1} interpolation`
```

Backtick strings use `{expr}` for interpolation. Single-quoted strings use `@{expr}`:

```parsley
let user = "Alice"
let greeting = `Welcome, {user}!`
log(greeting)    // Welcome, Alice!
```

Strings have many useful methods:

```parsley
log("hello".toUpper())       // HELLO
log("  hi  ".trim())         // hi
log("hello world".split(" ")) // ["hello", "world"]
log("banana".includes("nan")) // true
```

**See:** [Strings](builtins/strings.md)

---

## Arrays

Arrays are ordered collections. Create them with square brackets:

```parsley
let fruits = ["apple", "banana", "cherry"]
log(fruits[0])          // apple
log(fruits[-1])         // cherry (last element)
log(fruits.length())    // 3
```

Use `.map()` to transform and `.filter()` to select:

```parsley
let nums = [1, 2, 3, 4, 5]

let doubled = nums.map(fn(n) { n * 2 })
log(doubled)    // [2, 4, 6, 8, 10]

let big = nums.filter(fn(n) { n > 3 })
log(big)        // [4, 5]
```

Concatenate arrays with `++`:

```parsley
log([1, 2] ++ [3, 4])  // [1, 2, 3, 4]
```

**See:** [Arrays](builtins/array.md)

---

## Dictionaries

Dictionaries are key-value pairs — Parsley's equivalent of objects or maps:

```parsley
let person = {name: "Alice", age: 30, city: "London"}
log(person.name)    // Alice
log(person.age)     // 30
```

Merge dictionaries with `++`:

```parsley
let defaults = {theme: "light", lang: "en"}
let prefs = {theme: "dark"}
log(defaults ++ prefs)  // {theme: "dark", lang: "en"}
```

Destructure to extract values:

```parsley
let {name, age} = {name: "Bob", age: 25, role: "admin"}
log(name)   // Bob
log(age)    // 25
```

**See:** [Dictionaries](builtins/dictionary.md)

---

## Functions

Functions are created with `fn`. The last expression is the return value:

```parsley
let double = fn(x) { x * 2 }
log(double(5))      // 10

let add = fn(a, b) { a + b }
log(add(3, 4))      // 7
```

Functions can have default parameters:

```parsley
let greet = fn(name, greeting = "Hello") {
    `{greeting}, {name}!`
}
log(greet("Alice"))            // Hello, Alice!
log(greet("Bob", "Hey"))       // Hey, Bob!
```

Functions are values — you can pass them to other functions:

```parsley
let apply = fn(f, x) { f(x) }
log(apply(double, 21))  // 42
```

**See:** [Functions](fundamentals/functions.md)

---

## Control Flow

### `if` is an Expression

`if` returns a value, so you can use it inline:

```parsley
let age = 20
let status = if (age >= 18) "adult" else "minor"
log(status)     // adult
```

Or with blocks:

```parsley
if (age >= 18) {
    log("Welcome!")
} else {
    log("Too young")
}
```

### `for` Returns an Array

`for` loops are expressions that return arrays — like `map` in other languages:

```parsley
let nums = [1, 2, 3, 4, 5]

let squares = for (n in nums) { n * n }
log(squares)    // [1, 4, 9, 16, 25]
```

Filter by returning values only from an `if`:

```parsley
let evens = for (n in nums) {
    if (n % 2 == 0) { n }
}
log(evens)      // [2, 4]
```

Iterate over dictionaries:

```parsley
let scores = {alice: 95, bob: 87}
for (name, score in scores) {
    `{name}: {score}`
}
// ["alice: 95", "bob: 87"]
```

**See:** [Control Flow](fundamentals/control-flow.md)

---

## Building HTML with Tags

Parsley has first-class HTML tag syntax. Tags are values, not strings:

```parsley
let heading = <h1>"Hello!"</h1>
log(heading)    // <h1>Hello!</h1>
```

> **Important:** Text content inside tags must be quoted. Tag attributes don't need quotes for simple values.

Embed expressions directly inside tags:

```parsley
let user = "Alice"
<p>"Welcome, " user "!"</p>
```

**Result:** `<p>Welcome, Alice!</p>`

Use `for` to generate lists:

```parsley
let items = ["Apples", "Bananas", "Cherries"]
<ul>
    for (item in items) {
        <li>item</li>
    }
</ul>
```

**Result:** `<ul><li>Apples</li><li>Bananas</li><li>Cherries</li></ul>`

> **Singleton tags MUST be self-closing:** Write `<br/>`, `<hr/>`, `<img src="photo.jpg"/>` — never `<br>` or `<img>`.

**See:** [Tags](fundamentals/tags.md)

---

## Components

Components are functions that return tags. Use them like custom HTML elements:

```parsley
let Card = fn({title, body}) {
    <div class=card>
        <h2>title</h2>
        <p>body</p>
    </div>
}

<Card title="Hello" body="Welcome to Parsley!"/>
```

**Result:** `<div class=card><h2>Hello</h2><p>Welcome to Parsley!</p></div>`

Components compose naturally:

```parsley
let Page = fn({title, contents}) {
    <html>
    <head><title>title</title></head>
    <body>contents</body>
    </html>
}

<Page title="Home">
    <h1>"Welcome!"</h1>
    <p>"This is my page."</p>
</Page>
```

**See:** [Tags](fundamentals/tags.md) · [Modules](fundamentals/modules.md)

---

## A Simple Web Page Handler

In Basil, each route is a `.pars` file that exports a handler. Here's a complete example:

```parsley
// components/Layout.pars
export Layout = fn({title, contents}) {
    <html>
    <head>
        <title>title</title>
        <link rel="stylesheet" href="/style.css"/>
    </head>
    <body>
        <nav><a href="/">"Home"</a></nav>
        <main>contents</main>
    </body>
    </html>
}
```

```parsley
// routes/index.pars
{Layout} = import(@./components/Layout.pars)

let items = ["Learn Parsley", "Build with Basil", "Ship it!"]

<Layout title="My App">
    <h1>"My To-Do List"</h1>
    <ul>
        for (item in items) {
            <li>item</li>
        }
    </ul>
</Layout>
```

The handler returns HTML automatically — Basil detects the leading `<` tag and sets the content type.

---

## Error Handling

`try` wraps a function call and returns `{result, error}` — there are no catch blocks:

```parsley
let risky = fn() { fail("oops") }
let {result, error} = try risky()
if (error) {
    log("Failed: " + error)
} else {
    log("Got: " + result)
}
```

Use `check` as a guard — it's like an assertion that returns early:

```parsley
let process = fn(input) {
    check input != null else "No input"
    check input.length() > 0 else "Empty input"
    `Processing: {input}`
}
```

**See:** [Error Handling](fundamentals/errors.md)

---

## Where to Go Next

You now know the fundamentals. Here are paths forward depending on what you're building:

| Goal | Read |
|------|------|
| Learn all the built-in types | [Types](fundamentals/types.md), then the [Built-in Types](index.md#built-in-types) section |
| Build web pages | [Tags](fundamentals/tags.md) → [Modules](fundamentals/modules.md) → [Strings](builtins/strings.md) |
| Work with databases | [Database](features/database.md) → [Query DSL](features/query-dsl.md) → [Schemas](builtins/schema.md) |
| Build REST APIs | [@std/api](stdlib/api.md) → [HTTP & Networking](features/network.md) → [@std/session](stdlib/session.md) |
| Process files and data | [File I/O](features/file-io.md) → [Data Formats](features/data-formats.md) → [Arrays](builtins/array.md) |
| Explore the full manual | [Manual Index](index.md) |

---

## Key Differences from Other Languages

If you're coming from JavaScript, Python, or similar languages, watch out for these:

| Concept | Parsley | Other languages |
|---------|---------|-----------------|
| Output | `log()` | `print()`, `console.log()` |
| Comments | `//` only | `#` in Python |
| String interpolation | `` `Hello {name}` `` | `${}` in JS, `f""` in Python |
| For loops | Return arrays (like `map`) | Statements (no return value) |
| If/else | Expression (returns a value) | Statement in most languages |
| Tags | First-class syntax: `<p>"hi"</p>` | Strings: `"<p>hi</p>"` |
| Self-closing tags | Required: `<br/>` | Optional in HTML5 |
| Paths | Literals: `@./file.txt` | Strings: `"./file.txt"` |

**See:** [Parsley Cheatsheet](../CHEATSHEET.md)