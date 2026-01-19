# Parsley Cheat Sheet

Quick reference for beginners and AI agents developing Parsley. Focus on key differences from JavaScript, Python, Rust, and Go.

---

## 🚨 Major Gotchas (Common Mistakes)

### 1. Output Functions
```parsley
// ❌ WRONG (JavaScript/Python style)
console.log("hello")   // No console object exists

// ✅ CORRECT - Multiple options exist
"hello"         // Print without newline
"hello\n"       // Print with newline
log("hello")           // Log to stdout immediately
```

### 2. Comments
```parsley
// ✅ CORRECT - C-style single-line comments only
// This is a comment

/* Multi-line comments DON'T work */  // ❌ Parsed as regex, causes error

// ❌ WRONG - No Python/Shell style
# This will ERROR
```

### 3. For Loops Return Arrays (Like map)
```parsley
// ❌ WRONG (JavaScript thinking)
for (n in [1,2,3]) {
    console.log(n)  // Expecting side effects only
}

// ✅ CORRECT - For is expression-based, returns array
let doubled = for (n in [1,2,3]) { n * 2 }  // [2, 4, 6]

// Filter pattern - if returns null, omitted from result
let evens = for (n in [1,2,3,4]) {
    if (n % 2 == 0) { n }  // [2, 4]
}

// Loop control: stop and skip (not break/continue!)
let firstTwo = for (x in 1..100) {
    if (x > 2) stop  // Exit loop, return accumulated [1, 2]
    x
}

let noThrees = for (x in 1..5) {
    if (x == 3) skip  // Skip this iteration
    x
}  // [1, 2, 4, 5]
```

### 4. If  Parentheses are optional but recommended
```parsley
// ⚠️ CORRECT but could be ambiguous, especially for ternary
if age >= 18 { "adult" }

// ✅ CORRECT - Parentheses never ambiguous
if (age >= 18) { "adult" }

// If is an expression (returns value like ternary) needs parentheses
let status = if (age >= 18) "adult" else "minor"
```

### 5. Path Literals Use @
```parsley
// ✅ CORRECT
let path = @./config.json          // Relative to current file
let rootPath = @~/components/nav   // Relative to project root (Basil)
let url = @https://example.com
let date = @2024-11-29
let time = @14:30
let duration = @1d

// ❌ WRONG
let path = "./config.json"  // This is just a string
```

### 6. No Arrow Functions - Use fn() { }
```parsley
// ❌ WRONG (JavaScript arrow functions)
arr.map(x => x * 2)

// ✅ CORRECT - Use fn() { } syntax
arr.map(fn(x) { x * 2 })
arr.filter(fn(x) { x > 0 })

// Named functions
let double = fn(x) { x * 2 }
```

### 7. Strings in HTML Must Be Quoted
```parsley
// ❌ WRONG - bare text in tags
<h3>Welcome to Parsley</h3>

// ✅ CORRECT - strings need quotes
<h3>"Welcome to Parsley"</h3>
<h3>`Welcome to {name}`</h3>       // Template literal style also works
```

### 8. Tag Attributes: Strings vs Expressions
```parsley
// Tag attributes have THREE forms:

// 1. Double-quoted strings - literal, no interpolation
<button onclick="alert('hello')">
<a href="/about">

// 2. Single-quoted strings - RAW, for embedding JavaScript
<button onclick='Parts.refresh("editor", {id: 1})'>
// ^ Double quotes and braces stay literal - perfect for JS!
// Use @{} for dynamic values:
<button onclick='Parts.refresh("editor", {id: @{myId}})'>

// 3. Expression braces - Parsley code
<div class={`user-{id}`}>              // Template string for dynamic class
<button disabled={!isValid}>           // Boolean expression
<img width={width} height={height}>

// ❌ WRONG - interpolation in quoted strings
<div class="user-{id}">               // {id} is literal text

// ✅ CORRECT - use expression braces with template string
<div class={`user-{id}`}>
```

### 9. Single-Quoted Raw Strings (JavaScript Embedding)
```parsley
// Single quotes create raw strings - braces stay literal
let js = 'Parts.refresh("editor", {id: 1})'
let regex = '\d+\.\d+'              // Backslashes stay literal

// Use @{} for interpolation inside raw strings
let id = 42
let js = 'Parts.refresh("editor", {id: @{id}})'  // id interpolated

// Perfect for onclick handlers with dynamic values:
let myId = 5
<button onclick='Parts.refresh("editor", {id: @{myId}, view: "delete"})'/>

// Static JS (no interpolation needed):
<button onclick='Parts.refresh("editor", {id: 1, view: "delete"})'/>

// Escape @ with \@ if you need a literal @
'email: user\@domain.com'          // literal @
```

### 10. Self-Closing Tags MUST Use />
```parsley
// ❌ WRONG - not self-closing
<br>
<img src="photo.jpg">
<Part src={@./foo.part}>

// ✅ CORRECT - self-closing tags need />
<br/>
<img src="photo.jpg"/>
<Part src={@./foo.part}/>
```

### 11. Schema ID Types Require `auto` for Generation
```parsley
// ❌ WRONG - id: id without auto expects valid ULID format
@schema User {
    id: id                 // Requires valid ULID if provided!
    name: string
}
User({name: "Alice"})      // Missing id!
User({id: "my-id"})        // "my-id" is not a valid ULID format!

// ✅ CORRECT - use auto for server-generated IDs
@schema User {
    id: ulid(auto)         // Auto-generated ULID
    name: string
}
User({name: "Alice"})      // ID generated on insert

// id type is alias for ulid:
// id(auto) = ulid(auto)
// id       = ulid (validates format)

// For arbitrary string IDs:
@schema User {
    id: string             // No format validation
    name: string
}
User({id: "my-custom-id"}) // Any string works
```

---

## 📊 Syntax Quick Reference

### Variables & Functions

| Feature | JavaScript | Python | Parsley |
|---------|-----------|--------|---------|
| Variable | `let x = 5` | `x = 5` | `let x = 5` |
| Destructure | `const {x, y} = obj` | `x, y = obj` | `let {x, y} = obj` |
| Array Destruct | `const [a, b] = arr` | `a, b = arr` | `let [a, b] = arr` |
| Rest (array) | `const [a, ...rest] = arr` | `a, *rest = arr` | `let [a, ...rest] = arr` |
| Function | `(x) => x*2` | `lambda x: x*2` | `fn(x) { x*2 }` |
| Named func | `function f(x) {}` | `def f(x):` | `let f = fn(x) {}` |

### Control Flow

| Feature | JavaScript | Python | Parsley |
|---------|-----------|--------|---------|
| If expr | `x ? "yes" : "no"` | `"yes" if x else "no"` | `if (x) "yes" else "no"` |
| If block | `if (x) { } else { }` | `if x:\nelse:` | `if (x) {} else {}` |
| For loop | `for (let x of arr)` | `for x in arr:` | `for (x in arr) {}` |
| Map | `arr.map(x => x*2)` | `[x*2 for x in arr]` | `for (x in arr) { x*2 }` |
| Filter | `arr.filter(x => x>0)` | `[x for x in arr if x>0]` | `for (x in arr) { if (x>0) {x} }` |
| Index | `arr.forEach((x,i) => )` | `for i, x in enumerate(arr):` | `for (i, x in arr) {}` |
| Break | `break` | `break` | `stop` |
| Continue | `continue` | `continue` | `skip` |
| Guard | N/A | N/A | `check COND else VAL` |
| Type check | `x instanceof Class` | `isinstance(x, Class)` | `record is Schema` |

### Data Types

| Type | JavaScript | Python | Parsley |
|------|-----------|--------|---------|
| Array | `[1, 2, 3]` | `[1, 2, 3]` | `[1, 2, 3]` |
| Dict | `{x: 1, y: 2}` | `{"x": 1, "y": 2}` | `{x: 1, y: 2}` |
| String | `"text"` | `"text"` | `"text"` (escapes only) |
| Template | `` `Hi ${x}` `` | `f"Hi {x}"` | `` `Hi {x}` `` |
| Raw String | `'raw'` | `r'raw'` | `'raw @{x}'` |
| Regex | `/abc/i` | `re.compile(r"abc", re.I)` | `/abc/i` |
| Null | `null` | `None` | `null` |
| Money | N/A | N/A | `$12.34`, `EUR#50.00` |

---

## 🎯 Key Language Features

### 1. Everything Is an Expression
```parsley
// If returns value
let x = if (true) 10 else 20  // x = 10

// For returns array
let squares = for (n in 1..5) { n * n }  // [1, 4, 9, 16, 25]

// Tags return strings
let html = <p>"Hello"</p>  // "<p>Hello</p>"
```

### 2. String Interpolation
```parsley
// ✅ Interpolation ONLY in backtick strings
let name = "Alice"
let msg = `Hello, {name}!`      // "Hello, Alice!"
let calc = `2 + 2 = {2 + 2}`    // "2 + 2 = 4"

// ❌ Regular strings do NOT interpolate
let msg = "Hello, {name}!"      // {name} stays literal

// In tag attributes:
// - Quoted strings are ALWAYS literal (allows JavaScript)
// - Expression braces allow Parsley code
<div class="static-class">       // Literal string
<div class={dynamicClass}>       // Parsley expression  
<div class={`user-{id}`}>        // Template string interpolation
<button onclick="Parts.refresh('search', {query: this.value})">  // JS works!
```

### 3. HTML/XML as First-Class
```parsley
// Tags return strings
<p>"Hello"</p>                    // "<p>Hello</p>"

// Components are just functions
let Card = fn({title, contents}) { // contents, not body, not children
    <div class="card">
        <h3>title</h3> // not <h3>{title}</h3>
        <p>contents</p> // contents, not body, not children
    </div>
}

// Use 
<Card title="Welcome">"Hello world"</Card> // preferred
or
<Card title="Welcome" contents="Hello world"/>

```

### 4. Operators Are Overloaded
```parsley
// Arithmetic
5 + 3                        // 8
"Hello" + " World"           // "Hello World"
@/usr + "local"              // @/usr/local (path join)

// Multiplication
3 * 4                        // 12
"ab" * 3                     // "ababab"
[1, 2] * 3                   // [1, 2, 1, 2, 1, 2]

// Division
10 / 3                       // 3.333...
[1,2,3,4,5,6] / 2           // [[1,2], [3,4], [5,6]] (chunk)

// Logical become set operations on collections
[1,2,3] && [2,3,4]          // [2, 3] (intersection)
[1,2] || [2,3]              // [1, 2, 3] (union)
[1,2,3] - [2]               // [1, 3] (subtraction)

// Membership testing with 'in'
2 in [1, 2, 3]              // true (array membership)
"name" in {name: "Sam"}     // true (dict key exists)
"ell" in "hello"            // true (substring)

// Negated membership with 'not in'
5 not in [1, 2, 3]          // true
"foo" not in {name: "Sam"}  // true
"xyz" not in "hello"        // true

// Schema checking with 'is' / 'is not'
@schema User { name: string }
let user = User({name: "Alice"})
user is User                // true (record has User schema)
user is not User            // false
"hello" is User             // false (non-records safely return false)

// Null-safe: 'in' with null returns false
"admin" in null             // false (no error)
"key" not in null           // true

// Range
1..5                         // [1, 2, 3, 4, 5]

// Null coalescing
value ?? "default"           // Returns "default" only if value is null
```

### 5. Path Literals with @
```parsley
// Paths
@./relative/path           // Relative to current file
@~/components/header.pars  // Relative to project root (in Basil)
@/absolute/path            // Absolute filesystem path

// URLs
@https://example.com
@https://api.github.com/users

// Dates/Times
@2024-11-29                  // Date
@2024-11-29T14:30:00        // DateTime
@14:30                       // Time
@now                         // Current datetime
@today                       // Current date

// Single-digit months/days are accepted (forgiving parsing)
@2024-3-5                    // Same as @2024-03-05
@1990-3-18                   // Same as @1990-03-18

// Durations
@1d                          // 1 day
@2h30m                       // 2 hours 30 minutes
@-1w                         // Negative: 1 week ago

// DateTime operations
@2024-12-25 + @1d            // Add 1 day
@now - @2024-01-01           // Duration between dates
@2024-11-21 && @12:30        // Combine date + time → datetime

// Interpolated (dynamic)
let month = "11"
let date = @(2024-{month}-29)  // Builds from variables

// Forgiving interpolation (works with single-digit months/days)
let month = 3                  // No leading zero needed
let day = 5
let date = @(2024-{month}-{day})  // Works! Parses as 2024-03-05
```

---

## 📁 File I/O

### Factory Functions
```parsley
file(@path)      // Auto-detect format from extension
JSON(@path)      // Parse as JSON
CSV(@path)       // Parse as CSV
YAML(@path)      // Parse as YAML
MD(@path)  // Markdown with @{...} rendering + frontmatter
text(@path)      // Plain text
lines(@path)     // Array of lines
bytes(@path)     // Byte array
SVG(@path)       // SVG (strips prolog)
dir(@path)       // Directory listing

// note that there are builtin string equivalents
// e.g.

markdown("# hello").html // --> <h1>hello</h1>
```

### Read/Write Operators
```parsley
// Read
let config <== JSON(@./config.json)

// With error handling
let {data, error} <== JSON(@./data.json)
if (error) {
    log("Error:", error)
}

// Destructure from file
let {name, version} <== JSON(@./package.json)

// Write (overwrite)
data ==> JSON(@./output.json)

// Append
log_entry ==>> text(@./log.txt)
```

### Stdin/Stdout/Stderr
```parsley
let data <== JSON(@-)              // Read JSON from stdin
data ==> JSON(@-)                  // Write JSON to stdout

let input <== text(@stdin)
"output" ==> text(@stdout)
"error" ==> text(@stderr)
```

### HTTP Requests
```parsley
// Simple GET
let users <=/= JSON(@https://api.example.com/users)

// With error handling
let {data, error, status} <=/= JSON(@https://api.example.com/data)

// POST with body
let response <=/= JSON(@https://api.example.com/users, {
    method: "POST",
    body: {name: "Alice"},
    headers: {"Authorization": "Bearer token"}
})
```

### Database (SQLite)
```parsley
let db = @sqlite(@./data.db)

// Query single row (returns dict or null)
let user = db <=?=> "SELECT * FROM users WHERE id = 1"

// Query multiple rows (returns array)
let users = db <=??=> "SELECT * FROM users WHERE age > 25"

// Execute mutation (INSERT/UPDATE/DELETE)
let result = db <=!=> "INSERT INTO users (name) VALUES ('Alice')"
// result = {affected: 1, lastId: 1}
```

---

## 💰 Money Type

```parsley
// Currency literals
$12.34                       // USD
£99.99                       // GBP
€50.00                       // EUR
¥1000                        // JPY (no decimals)
CA$25.00                     // Canadian Dollar
USD#12.34                    // CODE# syntax

// Arithmetic (same currency only!)
$10.00 + $5.00               // $15.00
$10.00 * 2                   // $20.00
$10.00 / 3                   // $3.33 (banker's rounding)

// ❌ ERROR: Cannot mix currencies
$10.00 + £5.00               // Error!

// Properties and methods
$12.34.currency              // "USD"
$12.34.amount                // 1234 (in cents/minor units)
$12.34.scale                 // 2 (decimal places)
$1234.56.format()            // "$1,234.56"
$1234.56.format("de-DE")     // "1.234,56 $" (German locale)
$100.00.split(3)             // [$33.34, $33.33, $33.33]
$50.00.convert("EUR", 0.92)  // Convert with exchange rate
```

---

## 🔧 Common Patterns

### Error Handling

**For file/network ops, use capture pattern:**
```parsley
let {data, error} <== JSON(@./file.json)
if (error) {
    log("Failed:", error)
}
```

**For function calls, use `try` expression:**
```parsley
let {result, error} = try url("user-input")
if (error != null) {
    log("Invalid URL:", error)
}

// With null coalescing for defaults
let parsed = (try time("maybe-invalid")).result ?? @now
```

**User-defined errors with `fail()`:**
```parsley
let validate = fn(x) {
  if (x < 0) { fail("must be non-negative") }
  x * 2
}
let {result, error} = try validate(-5)
// error = "must be non-negative"
```

### Map/Filter
```parsley
// Map
let doubled = for (n in numbers) { n * 2 }

// Filter  
let evens = for (n in numbers) {
    if (n % 2 == 0) { n }
}

// Map + Filter
let processed = for (item in items) {
    if (item.active) {
        item.name.toUpper()
    }
}
```

### Components
```parsley
// Define component
let Button = fn({text, onClick}) {
    <button onclick="{onClick}">text</button> // not {text}
}

// Use component
<Button text="Click Me" onClick="handleClick()"/>

// With contents
let Card = fn({title, contents}) {
    <div class="card">
        <h3>title</h3>
        contents // not {children}
    </div>
}
```

### Modules
```parsley
// Export from module (utils.pars)
export let greet = fn(name) { "Hello, {name}!" }
export PI = 3.14159
export Logo = <img src="logo.png" alt="Logo"/>

// Computed exports (recalculate on each access)
export computed timestamp = @now
export computed users {
    @DB.query("SELECT * FROM users")
}

// Import with destructuring (recommended)
let {greet, PI} = import @./utils
let {floor, ceil} = import @std/math

// Import entire module
import @std/math            // binds to 'math'
math.floor(3.7)

// With alias
import @std/math as M
M.floor(3.7)
```

#### ⚠️ Computed Export Pitfall
```parsley
// Computed exports ALWAYS recalculate
export computed users = @DB.query("SELECT * FROM users")

// BAD: This queries the database twice!
for (user in users) { print(user.name) }
for (user in users) { print(user.email) }

// GOOD: Cache if you need to iterate multiple times
let snapshot = users
for (user in snapshot) { print(user.name) }
for (user in snapshot) { print(user.email) }
```

---

## 📚 Standard Library (@std)

### Module Quick Reference

```parsley
// Standard library
let {dev} = import @std/dev
let {PI, sin, cos, floor} = import @std/math
let {email, minLen, url} = import @std/valid
let {string, object, validate} = import @std/schema
let {uuid, nanoid, new} = import @std/id
let {notFound, redirect} = import @std/api
let {mdDoc} = import @std/mdDoc

// Tables - use @table literal instead of @std/table
let t = @table [{name: "Alice"}, {name: "Bob"}]

// Basil context (available in handlers)
let {request, response, query, route, method} = import @basil/http
let {session, auth, user} = import @basil/auth
```

### Math Module (`@std/math`)
```parsley
let {PI, E, TAU, floor, ceil, round, min, max, sum, avg,
     median, stddev, random, randomInt, sqrt, pow, sin,
     hypot, lerp} = import @std/math

// Constants
PI                     // 3.14159...
E                      // 2.71828...
TAU                    // 6.28318...

// Rounding
floor(3.7)             // 3
ceil(3.2)              // 4
round(3.5)             // 4

// Aggregation (2 args OR array)
min(5, 3)              // 3
max([1, 2, 3])         // 3
sum([1, 2, 3])         // 6
avg([1, 2, 3])         // 2.0

// Statistics
median([1, 2, 3, 4])   // 2.5
stddev([1, 2, 3])      // ~0.816

// Random
random()               // [0, 1)
randomInt(6)           // [0, 6] (die roll)

// Powers & Trig
sqrt(16)               // 4
pow(2, 10)             // 1024
sin(PI / 2)            // 1

// Geometry
hypot(3, 4)            // 5
lerp(0, 100, 0.5)      // 50
```

### Validation Module (`@std/valid`)
```parsley
let {string, number, integer, minLen, maxLen, alpha,
     alphanumeric, positive, between, email, url,
     phone, creditCard, date} = import @std/valid

// Type validators
string("hello")                    // true
number(3.14)                       // true
integer(42)                        // true

// String validators
minLen("hello", 3)                 // true
maxLen("hello", 10)                // true
alpha("Hello")                     // true (letters only)
alphanumeric("abc123")             // true

// Number validators
positive(5)                        // true
between(5, 1, 10)                  // true

// Format validators
email("test@example.com")          // true
url("https://example.com")         // true
phone("+1 (555) 123-4567")         // true
creditCard("4111111111111111")     // true (Luhn check)

// Date validators
date("2024-12-25")                 // true (ISO)
date("12/25/2024", "US")           // true
date("25/12/2024", "GB")           // true
```

### Tables

#### Creating Tables
```parsley
// @table literal (preferred)
let t = @table [
    {name: "Alice", age: 30},
    {name: "Bob", age: 25}
]

// From arrays (first row is header)
let t = @table [
    ["name", "age"],
    ["Alice", 30],
    ["Bob", 25]
]

// With schema for validation/defaults
@schema Person { name: string, age: integer = 0 }
let t = @table(Person) [
    {name: "Alice", age: 30},
    {name: "Bob"}              // age defaults to 0
]

// CSV returns Table directly
let t = "name,age\nAlice,30".parseCSV()
let t <== CSV(@./data.csv)

// Table() builtin
let t = Table([{name: "Alice"}, {name: "Bob"}])
```

#### Table Operations
```parsley
// Filtering
t.where(fn(row) { row.age > 25 }).rows

// Sorting  
t.orderBy("age", "desc").rows

// Aggregation
t.count()                          // 2
t.sum("age")                       // 55
t.avg("age")                       // 27.5

// Properties
t.rows                             // Array of dictionaries
t.columns                          // ["name", "age"]
t.length                           // 2

// Output
t.toHTML()                         // HTML <table> string
t.toCSV()                          // CSV string
t.toArray()                        // Array of dictionaries
```

### HTML Components (`@std/html`)
```parsley
let {Page, TextField, Button, Form} = import @std/html

<Page lang="en" title="Contact">
    <main>
        <h1>"Get in Touch"</h1>
        <Form action="/contact" method="POST">
            <TextField name="email" label="Email" type="email" required={true}/>
            <Button type="submit">"Send"</Button>
        </Form>
    </main>
</Page>
```

### API Helpers (`@std/api`)
```parsley
let {redirect, notFound, forbidden, badRequest,
     unauthorized, conflict, serverError} = import @std/api

redirect("/dashboard")              // 302 redirect
redirect("/new-page", 301)          // Permanent redirect
notFound("User not found")          // 404 error
forbidden("Access denied")          // 403 error
badRequest("Invalid input")         // 400 error
unauthorized("Login required")      // 401 error
```

### Schema Module (`@std/schema`)
```parsley
let {string, email, integer, number, boolean,
     object, array, define} = import @std/schema

// Define a schema with name and fields
let UserSchema = define("User", {
    name: string({minLen: 1}),
    email: email(),
    age: integer({min: 0, max: 150}),
    active: boolean()
})

// Validate data using schema method
let {valid, errors} = UserSchema.validate({
    name: "Alice",
    email: "alice@example.com",
    age: 30,
    active: true
})
```

### Form Binding

Form binding connects schema-validated records to HTML forms:

```parsley
@schema User {
    name: string(min: 2, required) | {title: "Full Name"}
    email: email(required)
    role: enum["user", "admin"]
}

let form = User({name: "Alice"})

// @record establishes form context, @field binds elements
<form @record={form} method="POST">
    <label @field="name"/>           // <label for="name">Full Name</label>
    <input @field="name"/>           // Sets name, value, type, constraints, ARIA
    <error @field="name"/>           // Shows validation error (or nothing)
    
    <select @field="role"/>          // Auto-generates <option> from enum
    
    <button type="submit">"Save"</button>
</form>
```

#### Form Binding Elements
| Element | Purpose |
|---------|---------|
| `<input @field="x"/>` | Bound input (auto-sets name, value, type, constraints, autocomplete) |
| `<label @field="x"/>` | Label from schema title metadata |
| `<error @field="x"/>` | Validation error (renders only if error exists) |
| `<select @field="x"/>` | Dropdown for enum fields |
| `<val @field="x" @key="help"/>` | Schema metadata value (help text, hints) |

Use `@tag` to change output element: `<error @field="name" @tag="div"/>`

#### Autocomplete Auto-Derivation
- **By type**: `email` → `autocomplete="email"`, `phone` → `"tel"`
- **By field name**: `firstName` → `"given-name"`, `password` → `"current-password"`
- **Override**: `street: string | {autocomplete: "shipping street-address"}`
- **Disable**: `captcha: string | {autocomplete: "off"}`

### ID Module (`@std/id`)
```parsley
let {new, uuid, uuidv4, uuidv7, nanoid, cuid} = import @std/id

new()                              // ULID-like (26 chars, sortable, URL-safe)
uuid()                             // UUID v4 (alias for uuidv4)
uuidv4()                           // UUID v4 (random)
uuidv7()                           // UUID v7 (time-sortable)
nanoid()                           // NanoID (default 21 chars)
nanoid(16)                         // NanoID with custom length
cuid()                             // CUID2-like (secure, collision-resistant)
```

### Dev Module (`@std/dev`)
```parsley
import @std/dev                    // Imports as 'dev'

dev.log("Debug info")              // Log to dev panel
dev.log("label", value)            // Log with label
dev.clearLog()                     // Clear dev log
dev.logPage("/route", value)       // Log to specific route
dev.setLogRoute("/api")            // Set default route
dev.clearLogPage("/route")         // Clear log for route
```

---

## 🌿 Basil Server

### Configuration (basil.yaml)
```yaml
server:
  host: localhost
  port: 8080

# Filesystem-based routing
site: ./site              # Files serve at their path

# OR explicit routes
routes:
  - path: /
    handler: ./handlers/index.pars
  - path: /api/*
    handler: ./handlers/api.pars

public_dir: ./public      # Static files

sqlite: ./data.db         # Database

session:
  secret: "32-char-secret"  # Required in production
  max_age: 24h

security:
  allow_write:
    - ./data              # Whitelist write directories
```

### HTTP Request/Response (`@basil/http`)
```parsley
let {request, response, query, route, method} = import @basil/http

// Shortcuts (most common)
query                              // URL query params {id: "123"}
route                              // Matched route subpath
method                             // "GET", "POST", etc.

// Full request object
request.method                     // "GET", "POST", etc.
request.path                       // "/users/123"
request.query                      // {id: "123"}
request.form                       // POST form data
request.cookies                    // {theme: "dark"}
request.headers                    // Request headers

// Set response
response.status = 404
response.headers["X-Custom"] = "value"

// Set cookies
response.cookies.theme = "dark"
response.cookies.session = {
    value: token,
    maxAge: @30d,
    httpOnly: true,
    secure: true
}
```

### Sessions & Auth (`@basil/auth`)
```parsley
let {session, auth, user} = import @basil/auth

// Session: store values
session.set("user_id", 123)
session.set("cart", ["item1", "item2"])

// Session: retrieve values
let userId = session.get("user_id")
let cart = session.get("cart", [])        // with default

// Session: check/delete
session.has("user_id")                    // true
session.delete("user_id")
session.clear()                           // logout

// Flash messages (show once then disappear)
session.flash("success", "Profile updated!")
// On next page:
let msg = session.getFlash("success")

// Auth context
user                                      // Current user (auth.user shortcut)
auth.user                                 // Same as above
auth.isLoggedIn                           // Boolean

// Database - use @DB magic variable instead
@DB <=?=> `SELECT * FROM users WHERE id = {userId}`
```

**Session notes:**
- Stored in encrypted cookies (AES-256-GCM)
- Dev mode: `secure=false` (works with HTTP localhost)
- Production: `secure=true` (requires HTTPS)
- Dev mode: random secret (sessions don't persist across restarts)
- Set `session.secret` in config for persistent dev sessions

### CSRF Protection
```parsley
let {request} = import @basil/http

// In forms with auth
<form method=POST action="/submit">
    <input type=hidden name=_csrf value={request.csrf}/>
    <button>"Submit"</button>
</form>

// For AJAX, use meta tag
<meta name=csrf-token content={request.csrf}/>
```

### Path Pattern Matching
```parsley
let {route} = import @basil/http

// route is the subpath after the matched route pattern
// e.g., if basil.yaml has "/api/*" and URL is "/api/users/123"
// then route = "users/123"

logLine("Route: " + route)

// Use .match() to extract parameters
let {id} = route.match("users/:id")       // {id: "123"}
let {rest} = route.match("files/*")       // {rest: "a/b/c"}

// Pattern matching returns null if no match
if (let params = route.match("users/:id")) {
    showUser(params.id)
}

// Chain for nested routes
let {rest} = route.match("api/*")
let {id} = rest.match("users/:id")
```


---

## 🧩 Parts (Interactive Components)

Parts are reloadable HTML fragments that update without page reloads.

### Creating a Part (.part file)
```parsley
// counter.part - ONLY export functions (not variables)
export default = fn(props) {
    let count = props.count
    <div>
        `Count: {count}`
        <button part-click="increment" part-count={count + 1}>"+"</button>
    </div>
}

export increment = fn(props) {
    let count = props.count
    <div>
        `Count: {count}`
        <button part-click="increment" part-count={count + 1}>"+"</button>
    </div>
}
```

### Using Parts
```parsley
// Basic usage
<Part src={@~/parts/counter.part} view="default" count={0}/>

// With id for cross-part targeting
<Part src={@~/parts/results.part} id="search-results"/>

// Auto-refresh every second
<Part src={@~/parts/clock.part} part-refresh={1000}/>

// Load immediately after page (for slow data)
<Part src={@~/parts/data.part} view="placeholder" part-load="loaded"/>

// Lazy load when scrolled into view
<Part src={@~/parts/content.part} view="placeholder" 
      part-lazy="loaded" part-lazy-threshold={150}/>
```

### Cross-Part Targeting
Target a Part from outside its boundaries (e.g., search box targeting results):
```parsley
// Form outside the Part targets it by id
<form part-target="search-results" part-submit="results">
    <input type="text" name="query"/>
    <button type="submit">"Search"</button>
</form>

// Results Part receives the query prop
<Part src={@~/parts/results.part} id="search-results"/>
```

### Parts JavaScript API
```javascript
// Refresh a Part programmatically (with debounce for live search)
Parts.refresh("search-results", {query: "hello"}, {debounce: 300});

// Get Part state
const state = Parts.get("search-results");
// → { id, view, props, element, loading }

// Listen for Part events
Parts.on("search-results", "afterRefresh", (detail) => {
    console.log("Part refreshed:", detail.props);
});
```

### Part Attributes
| Attribute | Element | Effect |
|-----------|---------|--------|
| `id` | `<Part/>` | ID for cross-part targeting |
| `part-click="view"` | Any | Fetches view on click |
| `part-submit="view"` | `<form>` | Fetches view on submit |
| `part-target="id"` | Any | Target Part by id (cross-part) |
| `part-*` | Any | Passed as props to view |
| `part-refresh={ms}` | `<Part/>` | Auto-refresh interval |
| `part-load="view"` | `<Part/>` | Fetch view immediately after page load |
| `part-lazy="view"` | `<Part/>` | Lazy-load when near viewport |
| `part-lazy-threshold={px}` | `<Part/>` | Pre-load distance in pixels |

### CSS for Lazy Parts
```css
/* Lazy Parts need dimensions for IntersectionObserver */
[data-part-src]:not([data-part-lazy]) {
    display: contents;    /* Eager parts are transparent */
}

[data-part-lazy] {
    display: block;       /* Lazy parts need a box model */
    min-height: 50px;     /* Give them dimensions */
}
```

---

## 🎨 Asset Bundles

Basil auto-bundles CSS/JS from your handlers directory.

```parsley
<html>
  <head>
    <CSS/>    <!-- Outputs: <link rel="stylesheet" href="/__site.css?v=abc123"> -->
  </head>
  <body>
    <h1>"Hello"</h1>
    <Javascript/>  <!-- Outputs: <script src="/__site.js?v=def456"></script> -->
  </body>
</html>
```

**publicUrl() for component assets:**
```parsley
// In modules/Button.pars
let icon = publicUrl(@./icon.svg)
<img src={icon}/>
// Output: <img src="/__p/a3f2b1c8.svg"/>
```

---

## 🔒 Security Flags (CLI)

```bash
# Development (allow writes and imports)
./pars -w -x script.pars

# Production (whitelist specific paths)
./pars --allow-write=./output --allow-execute=./lib script.pars

# Restrict reads
./pars --restrict-read=/etc script.pars
./pars --no-read < data.json  # stdin-only
```

---

## 📝 Method Reference

### String Methods
| Method | Description | Example |
|--------|-------------|---------|
| `.length()` | String length | `"hello".length()` → `5` |
| `.toUpper()` | Uppercase | `"hello".toUpper()` → `"HELLO"` |
| `.toLower()` | Lowercase | `"HELLO".toLower()` → `"hello"` |
| `.capitalize()` | Capitalize first | `"hello".capitalize()` → `"Hello"` |
| `.title()` | Title Case | `"hello world".title()` → `"Hello World"` |
| `.trim()` | Remove whitespace | `"  hi  ".trim()` → `"hi"` |
| `.trimStart()` | Trim left | `"  hi".trimStart()` → `"hi"` |
| `.trimEnd()` | Trim right | `"hi  ".trimEnd()` → `"hi"` |
| `.split(delim)` | Split to array | `"a,b".split(",")` → `["a","b"]` |
| `.replace(old, new)` | Replace first | `"hi hi".replace("i", "o")` → `"ho hi"` |
| `.replaceAll(old, new)` | Replace all | `"hi hi".replaceAll("i", "o")` → `"ho ho"` |
| `.includes(substr)` | Contains check | `"hello".includes("ell")` → `true` |
| `.startsWith(prefix)` | Starts with? | `"hello".startsWith("he")` → `true` |
| `.endsWith(suffix)` | Ends with? | `"hello".endsWith("lo")` → `true` |
| `.indexOf(substr)` | Find position | `"hello".indexOf("l")` → `2` |
| `.slug()` | URL-safe slug | `"Hello World!".slug()` → `"hello-world"` |
| `.digits()` | Extract digits | `"(555) 123".digits()` → `"555123"` |
| `.stripHtml()` | Remove HTML tags | `"<p>Hi</p>".stripHtml()` → `"Hi"` |
| `.escapeHtml()` | Escape for HTML | `"<b>".escapeHtml()` → `"&lt;b&gt;"` |
| `.highlight(term)` | Wrap matches | `"hi".highlight("h")` → `"<mark>h</mark>i"` |
| `.paragraphs()` | Text to HTML `<p>` | See reference |
| `.parseJSON()` | Parse JSON string | `'{"a":1}'.parseJSON()` → `{a: 1}` |
| `.parseCSV(header?)` | Parse CSV string | `"a,b\n1,2".parseCSV(true)` |
| `.pad(len, char?)` | Pad both sides | `"hi".pad(6)` → `"  hi  "` |
| `.padStart(len, char?)` | Pad left | `"5".padStart(3, "0")` → `"005"` |
| `.padEnd(len, char?)` | Pad right | `"hi".padEnd(5)` → `"hi   "` |

### Array Methods
| Method | Description | Example |
|--------|-------------|---------|
| `.length()` | Array length | `[1,2,3].length()` → `3` |
| `.first()` | First element | `[1,2,3].first()` → `1` |
| `.last()` | Last element | `[1,2,3].last()` → `3` |
| `.sort()` | Sort ascending | `[3,1,2].sort()` → `[1,2,3]` |
| `.sortBy(fn)` | Sort by key | `users.sortBy(fn(u){u.age})` |
| `.reverse()` | Reverse order | `[1,2,3].reverse()` → `[3,2,1]` |
| `.shuffle()` | Random order | `[1,2,3].shuffle()` |
| `.pick()` | Random element | `[1,2,3].pick()` → `2` |
| `.take(n)` | n random unique | `[1,2,3,4,5].take(3)` → `[4,1,3]` |
| `.map(fn)` | Transform each | `[1,2].map(fn(x){x*2})` → `[2,4]` |
| `.filter(fn)` | Keep matching | `[1,2,3].filter(fn(x){x>1})` → `[2,3]` |
| `.find(fn)` | Find first match | `[1,2,3].find(fn(x){x>1})` → `2` |
| `.findIndex(fn)` | Index of first match | `[1,2,3].findIndex(fn(x){x>1})` → `1` |
| `.every(fn)` | All match? | `[2,4,6].every(fn(x){x%2==0})` → `true` |
| `.some(fn)` | Any match? | `[1,2,3].some(fn(x){x>2})` → `true` |
| `.join(sep?)` | Join to string | `["a","b"].join(",")` → `"a,b"` |
| `.has(item)` | Contains check | `[1,2,3].has(2)` → `true` |
| `.flatten()` | Flatten nested | `[[1,2],[3]].flatten()` → `[1,2,3]` |
| `.unique()` | Remove dupes | `[1,1,2,2].unique()` → `[1,2]` |
| `.toJSON()` | To JSON string | `[1,2].toJSON()` → `"[1,2]"` |
| `.toCSV(header?)` | To CSV string | See reference |

### Dictionary Methods
| Method | Description | Example |
|--------|-------------|---------|
| `.keys()` | Get all keys | `{a:1, b:2}.keys()` → `["a", "b"]` |
| `.values()` | Get all values | `{a:1, b:2}.values()` → `[1, 2]` |
| `.has(key)` | Key exists? | `{a:1}.has("a")` → `true` |
| `.get(key, default?)` | Get with default | `{a:1}.get("b", 0)` → `0` |
| `.merge(other)` | Merge dicts | `{a:1}.merge({b:2})` → `{a:1, b:2}` |
| `.without(keys...)` | Remove keys | `{a:1, b:2}.without("b")` → `{a:1}` |
| `.pick(keys...)` | Keep only keys | `{a:1, b:2, c:3}.pick("a","c")` → `{a:1, c:3}` |
| `.toJSON()` | To JSON string | `{a:1}.toJSON()` → `"{\"a\":1}"` |
| `.type()` | Get type name | `{a:1}.type()` → `"dictionary"` |

### Number Methods
| Method | Description | Example |
|--------|-------------|---------|
| `.format(locale?)` | Locale format | `1234567.format()` → `"1,234,567"` |
| `.currency(code, locale?)` | Currency format | `99.currency("USD")` → `"$99.00"` |
| `.percent()` | Percentage | `0.125.percent()` → `"13%"` |
| `.humanize(locale?)` | Compact format | `1234567.humanize()` → `"1.2M"` |

### DateTime Methods
| Property | Description |
|----------|-------------|
| `.year`, `.month`, `.day` | Date components |
| `.hour`, `.minute`, `.second` | Time components |
| `.weekday` | Day name ("Monday") |
| `.unix` | Unix timestamp |
| `.format(style?, locale?)` | Format output |

---

## 🚀 Quick Examples

### Simple Script
```parsley
let data <== JSON(@./input.json)
let processed = for (item in data) {
    {
        name: item.name.toUpper(),
        score: item.score * 2
    }
}
processed ==> JSON(@./output.json)
log("Processed {processed.length()} items")
```

### HTML Generation
```parsley
let {Page} = import @~/components/page/page.pars

let users <== JSON(@./users.json)

<Page title="Users">
    <table>
        <tr><th>"Name"</th><th>"Role"</th></tr>
        for (user in users) {
            <tr>
                <td>user.name</td>
                <td>user.role</td>
            </tr>
        }
    </table>
</Page>
```

### Dictionary Spreading in HTML Tags
```parsley
// Spread dictionaries into HTML attributes using ...identifier
let attrs = {placeholder: "Name", maxlength: 50, disabled: true}
<input type="text" ...attrs/>
// → <input type="text" placeholder="Name" maxlength="50" disabled/>

// Boolean handling: true → attr, false/null → omitted
let flags = {required: true, disabled: false, readonly: null}
<input ...flags/>
// → <input required/>  (disabled and readonly omitted)

// Multiple spreads: last value wins
let base = {class: "input", type: "text"}
let override = {class: "input-lg"}
<input ...base ...override/>
// → <input type="text" class="input-lg"/>

// Works with rest destructuring for clean component APIs
let TextField = fn(props) {
    let {name, label, ...inputAttrs} = props
    <input name={name} ...inputAttrs/>
}

<TextField name="email" placeholder="Email" required={true}/>
// → <input name="email" placeholder="Email" required/>
```

### API Integration
```parsley
let {data, error} <=/= JSON(@https://jsonplaceholder.typicode.com/posts)

if (error) {
    log("API Error:", error)
} else {
    for (post in data) {
        log("Post {post.id}: {post.title}")
    }
}
```

---

## 📝 Testing/Debugging Tips

1. **Use `inspect(value)`** - returns introspection data as dictionary
2. **Use `describe(value)`** - human-readable description of any value
3. **Check types with `.type()`** method when confused - `x.type()` → `"string"`
4. **Remember `for` returns an array** - don't expect side effects
5. **Use error capture** `{data, error}` for file/network ops
6. **Path literals need @** - `@./file` not `"./file"`
7. **Comments are //** not #
8. **Strings in HTML need quotes** - `<p>"text"</p>` not `<p>text</p>`
9. **Self-closing tags need />** - `<br/>` not `<br>`
10. **Code between tags does not need `{}`** - `<p>text</p>` not `<p>{text}</p>`
11. **Use `repr(value)`** - returns code representation for debugging
12. **Use `builtins()`** - list all available builtin functions

