# Parsley Cheat Sheet

Quick reference for beginners and AI agents developing Parsley. Focus on key differences from JavaScript, Python, Rust, and Go.

---

## 🤖 AI Agent Quick Start

**Before writing Parsley code, use these tools to get accurate information:**

| Task | Command |
|------|---------|
| Look up type methods | `pars describe string`, `pars describe array`, etc. |
| Look up a builtin | `pars describe serialize`, `pars describe JSON`, etc. |
| Get all API info as JSON | `pars describe all --json` |
| Verify code works | `pars -e "expression"` |
| Verify with raw output | `pars -r -e "<div>html</div>"` |

**Key rules:**
1. **No print/println** — values ARE output (expression-based)
2. **`let` is immutable**, `var` is mutable (Swift-style)
3. **Verify before outputting** — run `pars -e "code"` to confirm syntax
4. **Code is truth** — if docs conflict with `pars describe`, trust the command

**Quick verification example:**
```bash
# Check if a method exists
pars describe string | grep -i "trim"

# Test an expression
pars -e '"  hello  ".trim()'  # → "hello"

# Test HTML output
pars -r -e '<div class="test">"content"</div>'
```

---

## 🚨 Major Gotchas (Common Mistakes)

### 1. No `print()` Function — Expressions ARE Output
```parsley
// ❌ WRONG — These functions don't exist!
print("hello")         // Error: Unknown function 'print'
println("hello")       // Error: Unknown function 'println'
printf("hi {}", x)     // Error: Unknown function 'printf'
console.log("hello")   // No console object exists

// ✅ CORRECT — Values ARE the output (expression-based)
"hello"                // The string itself is the output
42                     // Numbers too
<div>"content"</div>   // HTML tags as well

// ✅ For debugging, use log()
log("debug:", someVar) // Writes to stdout immediately, returns null
```

Parsley uses **expression-based output**: the last expression in a block becomes its output. There's no need to "print" — just write the value.

### 2. `let` is Immutable, `var` is Mutable (Swift-Style)
```parsley
// ❌ WRONG — Cannot reassign a let binding
let x = 5
x = 10                 // Error: cannot reassign immutable binding 'x'

// ✅ CORRECT — Use var for mutable bindings
var x = 5
x = 10                 // OK — var can be reassigned

// ❌ WRONG — Implicit declarations are errors
y = 5                  // Error: cannot assign to undeclared variable 'y'

// ✅ CORRECT — Always declare with let or var
let y = 5              // Immutable (use by default)
var z = 10             // Mutable (when you need to reassign)
```

**Key points:**
- `let` = immutable binding (like Swift's `let`, Rust's `let`, JS's `const`)
- `var` = mutable binding (like Swift's `var`)
- Immutability is **shallow**: you can mutate contents of `let` arrays/dicts, just can't reassign the variable
- Loop variables and function parameters are implicitly immutable
- **One declaration per scope**: redeclaring a name with `let`/`var` in the same scope is an error — shadowing in an inner scope (function body, `if` block) is fine

### 3. Comments
```parsley
// ✅ CORRECT - C-style single-line comments only
// This is a comment

/* Multi-line comments DON'T work */  // ❌ Parsed as regex, causes error

// ❌ WRONG - No Python/Shell style
# This will ERROR
```

### 4. For Loops Return Arrays (Like map)
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

### 5. If  Parentheses are optional but recommended
```parsley
// ⚠️ CORRECT but could be ambiguous, especially for ternary
if age >= 18 { "adult" }

// ✅ CORRECT - Parentheses never ambiguous
if (age >= 18) { "adult" }

// If is an expression (returns value like ternary) needs parentheses
let status = if (age >= 18) "adult" else "minor"
```

### 6. Path Literals Use @
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

### 7. No Arrow Functions - Use fn() { }
```parsley
// ❌ WRONG (JavaScript arrow functions)
arr.map(x => x * 2)

// ✅ CORRECT - Use fn() { } syntax
arr.map(fn(x) { x * 2 })
arr.filter(fn(x) { x > 0 })

// Named functions
let double = fn(x) { x * 2 }
```

### 8. Strings in HTML Must Be Quoted
```parsley
// ❌ WRONG - bare text in tags
<h3>Welcome to Parsley</h3>

// ✅ CORRECT - strings need quotes
<h3>"Welcome to Parsley"</h3>
<h3>`Welcome to {name}`</h3>       // Template literal style also works
```

### 9. Tag Content Does NOT Need Braces (Not JSX!)
```parsley
// Parsley is NOT JSX/React! Expressions in tag content don't need {braces}

// ❌ WRONG - JSX style with braces
<td>{row[col]}</td>
<label>{schema.title(field)}</label>
<span>{user.name}</span>

// ✅ CORRECT - Parsley style, no braces
<td>row[col]</td>
<label>schema.title(field)</label>
<span>user.name</span>

// Method calls, index access, arithmetic - all work without braces
<p>items.length()</p>              // method call
<td>data[i]</td>                   // index access
<span>price * quantity</span>      // arithmetic

// Braces ARE used for:
// 1. Attribute expressions
<div class={dynamicClass}>         // ✅ braces for attribute
// 2. Template string interpolation
<p>`Hello {name}`</p>              // ✅ braces inside template string
```

### 10. Tag Attributes: Strings vs Expressions
```parsley
// Tag attributes have THREE forms:

// 1. Double-quoted strings - text, but {expr} IS interpolated
<button onclick="alert('hello')">
<a href="/about">

// 2. Single-quoted strings - RAW, for embedding JavaScript
<button onclick='fetch("/api/items", {method: "POST"})'>
// ^ Double quotes and braces stay literal - perfect for JS!
// Use @{} for dynamic values:
<button onclick='fetch("/api/items/@{myId}", {method: "DELETE"})'>

// 3. Expression braces - Parsley code
<div class={`user-{id}`}>              // Template string for dynamic class
<button disabled={!isValid}>           // Boolean expression
<img width={width} height={height}>

// ✅ CORRECT - {id} interpolates inside a double-quoted attribute
<div class="user-{id}">               // class="user-42"

// ✅ CORRECT - or use expression braces with a template string
<div class={`user-{id}`}>

// ❌ WRONG - braces in double-quoted JS break (parsed as interpolation)
<button onclick="fetch(url, {method: 'POST'})">   // Error: unexpected ':'

// ✅ CORRECT - single quotes keep braces literal
<button onclick='fetch(url, {method: "POST"})'>
```

### 11. Single-Quoted Raw Strings (JavaScript Embedding)
```parsley
// Single quotes create raw strings - braces stay literal
let js = 'fetch("/api/items", {method: "POST"})'
let regex = '\d+\.\d+'              // Backslashes stay literal

// Use @{} for interpolation inside raw strings
let id = 42
let deleteJs = 'fetch("/api/items/@{id}", {method: "DELETE"})'  // id interpolated

// Perfect for onclick handlers with dynamic values:
let myId = 5
<button onclick='fetch("/api/items/@{myId}", {method: "DELETE"})'/>

// Static JS (no interpolation needed):
<button onclick='document.getElementById("output").innerHTML = "done"'/>

// Escape @ with \@ if you need a literal @
'email: user\@domain.com'          // literal @
```

### 12. Local vs Network Write Operators
```parsley
// ❌ WRONG — ==> is for local files only
data ==> JSON(@https://api.example.com/users)
// ERROR: operator ==> is for local file writes; use =/=> for network writes

// ✅ CORRECT — use =/=> for network writes
data =/=> JSON(@https://api.example.com/users)

// ✅ CORRECT — ==> for local files
data ==> JSON(@./output.json)
```

### 13. Self-Closing Tags MUST Use />
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

### 14. Schema ID Types Require `auto` for Generation
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

### 15. `try` Error Is a Dictionary, Not a String
```parsley
// ❌ WRONG - error is no longer a plain string
let {result, error} = try riskyFn()
if (error == "not found") { ... }    // Won't match — error is a dict

// ✅ CORRECT - access .message for the string
let {result, error} = try riskyFn()
if (error) {
    error.message                    // "not found" (string)
    error.code                       // "USER-0001" or "HTTP-404" etc.
    error.status                     // 404 (present on API errors)
}

// String coercion works: "" + error yields error.message
"Failed: " + error                  // uses error.message automatically
```

### 16. Unit Literals Use `#` Sigil; Temperature Cannot Be Multiplied; Derived Units Have Restrictions
```parsley
// ❌ WRONG — $ is for money, not units
$12m                             // This is money!

// ✅ CORRECT — # is the unit sigil
#12m                             // 12 metres

// ⚠️ Fractions use + for mixed numbers (not -)
#92+5/8in                        // 92 and 5/8 inches
// #92-5/8in                     // WRONG — parsed as subtraction

// ⚠️ SI fractions truncate, US fractions are exact
#1/3m                            // 333,333 µm (truncated!)
#1/3in                           // exact

// ⚠️ scalar + unit is an error (no implicit promotion)
5 + #5m                          // Error! Write #5m + #5m
10 / #5m                         // Error! Write #10m / 5

// ⚠️ Data suffixes are case-sensitive
#64kB                            // 64 kilobytes (lowercase k)
#1KiB                            // 1 kibibyte (uppercase K, binary)
// #64KB                         // Error! Unknown suffix

// ⚠️⚠️ Temperature CANNOT be multiplied or divided!
#20C + #10C                      // OK: #30C
// #20C * 2                      // ERROR — offset scales make this undefined
// #100C / #50C                  // ERROR — ratio is meaningless
// Use addition instead: #20C + #20C

// ⚠️ #1gal is volume, not #1g + "al"
#1gal                            // 1 gallon (volume, longest suffix match)
#5g                              // 5 grams (mass, still works)

// ⚠️ Area suffixes end with 2 (m2 not m²)
#100m2                           // 100 square metres (not #100m²)
#5km2                            // 5 square kilometres
#640ac                           // 640 acres (= 1 mi²)

// ⚠️ US area uses decimal display (no fractions)
#1/3ac                           // displays as #0.333...ac (decimal, not fraction)
```

---

## 📊 Syntax Quick Reference

### Variables & Functions

| Feature | JavaScript | Python | Parsley |
|---------|-----------|--------|---------|
| Immutable var | `const x = 5` | N/A | `let x = 5` |
| Mutable var | `let x = 5` | `x = 5` | `var x = 5` |
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
| With scope | N/A | N/A | `with dict { fields... }` |
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
| Unit | N/A | N/A | `#12m`, `#3/8in`, `#100C`, `#1gal` |

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

// Fetch and remote write return response objects
let response = <=/= JSON(@https://api.example.com/users)
let result = payload =/=> JSON(@https://api.example.com/items)
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
// - Double-quoted strings DO interpolate {expr}
// - Single-quoted (raw) strings keep braces literal (use @{expr} instead)
// - Expression braces allow Parsley code
<div class="static-class">       // Literal string
<div class="user-{id}">          // Interpolated → class="user-42"
<div class={dynamicClass}>       // Parsley expression  
<div class={`user-{id}`}>        // Template string interpolation
<button onclick="fetch('/api/search?q=' + this.value)">  // JS works!
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
10 / 3                       // 3 (integer division truncates; 10.0 / 3 → 3.333...)
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

// Flexible parsing (100+ formats supported)
datetime("22 April 2005")    // Human-readable date
date("March 14, 2024")       // Returns date only
time("3:45 PM")              // Returns time only
datetime("22/04/2005", {locale: "en-GB"})  // Day-first parsing

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
PLN(@path)       // Parse as PLN (Parsley Literal Notation)
MD(@path)  // Markdown with @{...} rendering + frontmatter
text(@path)      // Plain text
lines(@path)     // Array of lines
raw(@path)       // Byte array
SVG(@path)       // SVG (strips prolog)
dir(@path)       // Directory listing

// note that there are builtin string equivalents
// e.g.

markdown("# hello").html // --> <h1>hello</h1>
```

### Serialization
```parsley
// Serialize Parsley values to PLN strings
serialize({name: "Alice", age: 30})  // '{name: "Alice", age: 30}'
serialize([1, 2, 3])                 // "[1, 2, 3]"

// Deserialize PLN strings back to values
deserialize('{name: "Bob"}')         // {name: "Bob"}

// PLN is safe - rejects expressions/code
deserialize("1 + 1")                 // Error! No expressions allowed
```

### Read/Write Operators
```parsley
// Read
let config <== JSON(@./config.json)

// With error handling
let {data, error} <== JSON(@./data.json)
if (error) {
    `Error: {error}`
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

// POST with body (via fetch options)
let response <=/= JSON(@https://api.example.com/users, {
    method: "POST",
    body: {name: "Alice"},
    headers: {"Authorization": "Bearer token"}
})

// Fetch as expression (capture response object)
// NOTE: these are methods, not plain keys — `response.status` returns null
let response = <=/= JSON(@https://api.example.com/users)
response.data()                  // parsed content
response.response().status       // 200
response.response().ok           // true

// Destructured expression form
let {data, error} = <=/= JSON(@https://api.example.com/users)

// POST with remote write operator
{name: "Alice"} =/=> JSON(@https://api.example.com/users)

// Remote write as expression (capture response)
let result = {name: "Alice"} =/=> JSON(@https://api.example.com/users)
if (!result.response().ok) { `Failed: {result.response().status}` }

// Destructured remote write
let {data, error} = payload =/=> JSON(@https://api.example.com/items)

// PUT
data =/=> JSON(@https://api.example.com/users/1).put

// PATCH
patch =/=> JSON(@https://api.example.com/users/1).patch
```

### Database (SQLite)
```parsley
let db = @sqlite("./data.db")      // DSN is a string, not a path literal

// Query single row (returns dict or null)
let user = db <=?=> <SQL id={1}>SELECT * FROM users WHERE id = :id</SQL>

// Query multiple rows (returns Table)
let users = db <=??=> <SQL min_age={25}>SELECT * FROM users WHERE age > :min_age</SQL>

// Execute mutation (INSERT/UPDATE/DELETE)
let result = db <=!=> <SQL name="Alice">INSERT INTO users (name) VALUES (:name)</SQL>
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
$1234.56.format()            // "$ 1,234.56"
$1234.56.format("de-DE")     // "$ 1.234,56" (German locale)
$100.00.split(3)             // [$33.34, $33.33, $33.33]
```

---

## 🎨 Unified Formatter API

All value types support a consistent formatting API with `.fmt()` and style sugar methods.

### The `.fmt()` Method

```parsley
// No args: default style (medium) and locale (en-US)
123456.fmt()                 // "123,456"
$1234.56.fmt()               // "$ 1,234.56"
@2024-12-25.fmt()            // "Dec 25, 2024"
#5m.fmt()                    // "5.00m"

// With style: "short", "medium", "long", "full"
1234567.fmt("short")         // "1.2M"
$1234.56.fmt("full")         // "1,234.56 US dollars"
@2024-12-25.fmt("long")      // "December 25, 2024"
#5m.fmt("full")              // "5.00 meters (16.4 ft)"

// With style and locale
@2024-12-25.fmt("long", "de-DE")  // "25. Dezember 2024"
1234.fmt("medium", "de-DE")       // "1.234"

// With options dictionary
1234.5678.fmt({precision: 2})     // "1,234.57"
@2024-12-25.fmt({style: "full", locale: "de-DE"})  // "Mittwoch, 25. Dezember 2024"
```

### Style Sugar Methods

```parsley
// Numbers
1234567.short()              // "1.2M"
1234567.medium()             // "1,234,567"
1234.5.long()                // "1,234.50"

// Money (supports full)
$1234.56.short()             // "$1.2K"
$1234.56.medium()            // "$ 1,234.56"
$1234.56.long()              // "$1,234.56"
$1234.56.full()              // "1,234.56 US dollars"

// DateTime (supports full)
@2024-12-25.short()          // "12/25/24"
@2024-12-25.medium()         // "Dec 25, 2024"
@2024-12-25.long()           // "December 25, 2024"
@2024-12-25.full()           // "Wednesday, December 25, 2024"

// Duration (NO full support)
@2h30m.short()               // "2h30m"
@2h30m.medium()              // "in 2 hours"
@2h30m.long()                // "2 hours 30 minutes"

// Units (supports full with conversion)
#5m.short()                  // "5m"
#5m.medium()                 // "5.00m"
#5m.long()                   // "5.00 meters"
#5m.full()                   // "5.00 meters (16.4 ft)"

// Style methods accept locale
@2024-12-25.long("de-DE")    // "25. Dezember 2024"
1234.short("de-DE")          // "1,2K"
```

### Array Conjunction Formatting

```parsley
["Alice", "Bob", "Charlie"].fmt("and")           // "Alice, Bob, and Charlie"
["coffee", "tea", "milk"].fmt("or")              // "coffee, tea, or milk"

// Locale-aware conjunctions
["A", "B", "C"].fmt("and", "de-DE")              // "A, B und C"
["A", "B", "C"].fmt("and", "fr-FR")              // "A, B et C"
["A", "B", "C"].fmt("or", "es-ES")               // "A, B o C"

// Edge cases
[].fmt("and")                                    // ""
["Alice"].fmt("and")                             // "Alice"
["Alice", "Bob"].fmt("and")                      // "Alice and Bob"
```

### Serialization Methods

All types support consistent serialization:

```parsley
// repr() - parseable literal representation
42.repr()                    // "42"
$50.repr()                   // "$50.00"
@2024-12-25.repr()           // "@2024-12-25"
#5m.repr()                   // "#5m"

// toJSON() - JSON representation
42.toJSON()                  // "42"
true.toJSON()                // "true"
#5m.toJSON()                 // "{\"value\":5,\"unit\":\"m\",...}"

// inspect() - debug dictionary with __type
42.inspect()                 // {__type: "integer", value: 42}
@2024-12-25.inspect()        // {__type: "datetime", kind: "date", ...}
```

---

## 📐 Measurement Units

```parsley
// Unit literals use # sigil
#12m                         // 12 metres
#5.5kg                       // 5.5 kilograms
#100cm                       // 100 centimetres
#64kB                        // 64 kilobytes (lowercase k!)
#1KiB                        // 1 kibibyte (binary)

// Temperature
#100C                        // 100 degrees Celsius
#212F                        // 212 degrees Fahrenheit
#0K                          // absolute zero (kelvin)
#37.5C                       // decimal temperature
#-40C                        // negative (same as #-40F)

// Volume
#500mL                       // 500 millilitres
#2.5L                        // 2.5 litres
#1kL                         // 1 kilolitre (= 1000 L)
#8floz                       // 8 fluid ounces
#1cup                        // 1 cup (= 8 floz)
#1/3cup                      // exact fraction
#1+1/2gal                    // mixed number: 1.5 gallons

// Area (suffixes end with 2, not ²)
#100m2                       // 100 square metres
#5.5km2                      // 5.5 square kilometres
#1500ft2                     // 1500 square feet
#640ac                       // 640 acres (= 1 mi²)
#1mi2                        // 1 square mile
#2.5ac                       // 2.5 acres (decimal display)

// ⚠️ Fractions: US exact, SI truncated
#3/8in                       // exact (US Customary)
#1/3cup                      // exact (US Customary)
#1/3m                        // truncated to 333,333 µm (SI)

// ⚠️ US area always uses decimal display (no fractions)
#1/3ac                       // #0.333...ac (decimal, not fraction)

// ⚠️ Mixed numbers use + (not -)
#92+5/8in                    // 92 and 5/8 inches
// #92-5/8in                 // WRONG — this is subtraction!

// Negative: sign before number
#-6m                         // negative 6 metres (canonical)
-#6m                         // also works (unary negation)

// Arithmetic (same family only!)
#5cm + #3mm                  // #5.3cm
#3/8in + #5/8in              // #1in
#5m * 3                      // #15m
#10m / #5m                   // 2 (dimensionless)
#1/3cup + #1/3cup + #1/3cup  // #1cup (exact!)
#1gal / #1qt                 // 4
#100m2 + #50m2               // #150m2 (area)
#2kL * 3                     // #6kL (kilolitres)

// Temperature arithmetic: add/subtract only!
#20C + #10C                  // #30C
#100C - #37C                 // #63C
#212F - #32F                 // #180F

// ⚠️⚠️ Temperature multiply/divide is FORBIDDEN
// #20C * 2                  // Error! Offset scales make this undefined
// #100F / 2                 // Error! Use subtraction instead
// #100C / #50C              // Error! Ratio is meaningless for offset scales

// Temperature comparison (works across scales)
#0C == #32F                  // true
#100C == #212F               // true
#-40C == #-40F               // true (scales cross at -40)
#0K == #-273.15C             // true (absolute zero)

// ⚠️ Left side wins for cross-system
#1cm + #1in                  // #3.54cm (SI result)
#1in + #1cm                  // result in inches (US)
#1L + #1floz                 // result in litres (SI)
#100m2 + #100ft2             // result in m² (SI)

// ❌ ERROR: Cannot mix families
#5m + #5kg                   // Error!
#1L + #1C                    // Error! (volume ≠ temperature)
#5m2 + #5kg                  // Error! (area ≠ mass)

// ❌ ERROR: No implicit number-to-unit promotion
5 + #5m                      // Error! Write #5m + #5m
10 / #5m                     // Error! Write #10m / 5

// ✅ Derived units: length × length → area
#5m * #3m                    // #15m2
#2ft * #3ft                  // #6ft2
#100cm * #50cm               // #5000cm2
#5m * #300cm                 // #15m2 (left wins display hint)

// ✅ Area / length → length
#15m2 / #3m                  // #5m
#6ft2 / #2ft                 // #3ft
(#5m * #3m) / #3m            // #5m (round-trip)

// ❌ Cross-system derived arithmetic is forbidden
#5m * #3ft                   // Error! Convert to same system first
#15m2 / #3ft                 // Error!

// ❌ Other unit × unit not supported
#5kg * #3kg                  // Error! (only length × length)

// Comparison (works across systems)
#1in == #25.4mm              // true
#1024B == #1KiB              // true

// Properties
#12.3m.value                 // 12.3
#12.3m.unit                  // "m"
#12.3m.family                // "length"
#12.3m.system                // "SI"
#100C.value                  // 100
#100C.family                 // "temperature"
#1gal.family                 // "volume"
#100m2.family                // "area"
#640ac.system                // "US"

// .max and .min properties
#0m.max                      // largest representable value in metres
#0m.min                      // smallest positive value in metres
#100m2 < #0m2.max            // true

// Methods
#1mi.to("km")                // #1.609344km
(#-6m).abs()                 // #6m
#12.3m.format()              // "12.3m"
#12.3m.format(2)             // "12.30m" (precision)
#12.3m.repr()                // "#12.3m"
#3/8in.toFraction()          // "3/8\""
#100C.to("F")                // #212F
#32F.to("C")                 // #0C
#100C.to("K")                // #373.15K
(#-40C).abs()                // #40C
#1gal.to("qt")               // 4 quarts
#1/3cup.toFraction()         // "1/3cup"
#640ac.to("mi2")             // #1mi2
#100m2.to("km2")             // #0.0001km2

// ✅ Compound formatting
#63in.format("ft-in")        // "5' 3\""
#63.375in.format("ft-in")    // "5' 3+3/8\""
#37oz.format("lb-oz")        // "2lb 5oz"
#13pt.format("gal-qt-pt")    // "1gal 2qt 1pt"
#1500mL.format("L-mL")       // "1L 500mL"
#63in.format("compound")     // "5' 3\"" (auto-detect)
#6in.format("compound")      // "6in" (< 1ft, no compound)

// Cross-system compound (auto-converts first)
#1.5m.format("ft-in")        // "4' 11.055\""
#1kg.format("lb-oz")         // "2lb 3.274oz"

// Constructors (plural names)
metres(100)                  // #100m
inches(#1cm)                 // convert cm to inches
unit(123, "m")               // #123m (generic)
unit(#12in, "m")             // convert to metres
celsius(100)                 // #100C
fahrenheit(#100C)            // #212F (convert)
kelvins(#100C)               // #373.15K
litres(2)                    // #2L
kilolitres(5)                // #5kL
gallons(1)                   // #1gal
cups(#1L)                    // convert litres to cups
metres2(100)                 // #100m2
acres(640)                   // #640ac
kilometres2(510000000)       // Earth's surface (uses Scale)

// String interpolation: no # sigil
let d = #1.83m
`Height: {d}`                // "Height: 1.83m"
let t = #37.5C
`Temp: {t}`                  // "Temp: 37.5C"

// bytes() constructor is available:
bytes(1024)                  // #1024B
bytes(#1KiB)                 // #1024B (convert)
unit(1024, "B")              // OK
```

**Suffixes**: `mm`, `cm`, `m`, `km` · `in`, `ft`, `yd`, `mi` · `mg`, `g`, `kg` · `oz`, `lb` · `B`, `kB`, `MB`, `GB`, `TB` · `KiB`, `MiB`, `GiB`, `TiB` · `K`, `C`, `F` · `mL`, `L`, `kL` · `floz`, `cup`, `pt`, `qt`, `gal` · `mm2`, `cm2`, `m2`, `km2` · `in2`, `ft2`, `yd2`, `ac`, `mi2`

**Compound Formats**: `"ft-in"` · `"lb-oz"` · `"gal-qt-pt"` · `"L-mL"` · `"compound"` (auto-detect)

### Unit Types in Schemas

```parsley
// Family names as schema types
@schema Product {
    weight: mass,            // any mass unit (#5kg, #2.2lb, etc.)
    height: length,          // any length unit (#1.8m, #6ft, etc.)
    storage: data,           // any data unit (#1GB, #500MB, etc.)
    temp: temperature,       // any temperature (#100C, #212F, etc.)
    capacity: volume,        // any volume (#2L, #1gal, etc.)
    floor: area              // any area (#100m2, #1000ft2, etc.)
}

// Specific unit constraint
@schema MetricProduct {
    weight: unit(suffix: "kg"),   // must be in kg
    height: unit(suffix: "cm")    // must be in cm
}

// Validation — every field is required unless nullable (`?`) or given a default
Product({weight: #5kg, height: #1.8m, storage: #1GB,
         temp: #100C, capacity: #2L, floor: #100m2}).validate().isValid()  // true
Product({weight: #5m, height: #1.8m, storage: #1GB,
         temp: #100C, capacity: #2L, floor: #100m2}).validate().isValid()  // false (length ≠ mass)
MetricProduct({weight: #5kg, height: #180cm}).validate().isValid()  // true
MetricProduct({weight: #5kg, height: #1.8m}).validate().isValid()   // false (m ≠ cm)
```

---

## 🔧 Common Patterns

### Error Handling

**For file/network ops, use capture pattern:**
```parsley
let {data, error} <== JSON(@./file.json)
if (error) {
    `Failed: {error}`
}
```

**For function calls, use `try` expression:**
```parsley
let {result, error} = try url("user-input")
if (error) {
    `Invalid URL: {error.message}`
    error.code                   // e.g. "FMT-0003"
}

// With null coalescing for defaults
let parsed = (try datetime("maybe-invalid")).result ?? @now
```

**User-defined errors with `fail()`:**
```parsley
// String form — wrapped in {message: ..., code: "USER-0001"}
let validate = fn(x) {
  if (x < 0) { fail("must be non-negative") }
  x * 2
}
let {result, error} = try validate(-5)
error.message                    // "must be non-negative"

// Dictionary form — structured errors with custom fields
fail({message: "Out of stock", code: "NO_STOCK", status: 400})
```

**⚠️ `error` from `try` is a dictionary, not a string.** Use `error.message` to get the message. String coercion works too: `"" + error` yields `error.message`.

**Validation bridge — `failIfInvalid()`:**
```parsley
// One-liner: validate and fail with structured error if invalid
let user = User(formData).validate().failIfInvalid()
// If invalid → catchable error: {status: 400, code: "VALIDATION", message: "Validation failed", fields: [...]}
// If valid → returns the record (chainable)
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

### Scoped Field Access (`with`)
```parsley
// Reduce repetitive property chains in templates
// Before:
<dd>auth.user.id</dd>
<dd>auth.user.name</dd>
<dd>auth.user.email</dd>

// After: with expands dict fields into scope
with auth.user {
  <dd>id</dd>
  <dd>name</dd>
  <dd>email</dd>
}

// Nested with blocks
with order {
  <h2>"Order #" id</h2>
  with shipping.address {
    <p>street</p>
    <p>city ", " state</p>
  }
}

// ⚠️ Invalid identifier keys are skipped silently
// Keys like "hello world", "123", "a-b" won't become variables
let d = {"valid": 1, "not-valid": 2}
with d { valid }  // works, "not-valid" skipped
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
export let greet = fn(name) { `Hello, {name}!` }
export PI = 3.14159
export Logo = <img src="logo.png" alt="Logo"/>

// Computed exports (recalculate on each access)
export computed timestamp = @now
export computed users {
    let db = @sqlite("./app.sqlite")
    let rows = db <=??=> "SELECT * FROM users"
    rows
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
let db = @sqlite("./app.sqlite")
export computed users = db <=??=> "SELECT * FROM users"

// BAD: This queries the database twice!
for (user in users) { user.name }
for (user in users) { user.email }

// GOOD: Cache if you need to iterate multiple times
let snapshot = users
for (user in snapshot) { user.name }
for (user in snapshot) { user.email }
```

---

## 📚 Standard Library (@std)

### Module Quick Reference

```parsley
// Standard library
let {PI, sin, cos, floor} = import @std/math
let {uuid, nanoid, cuid} = import @std/id         // ID generators
let {uuid, nanoid, ulid, cuid} = import @std/valid // ID validators
let {mdDoc} = import @std/mddoc
let {md5, sha256} = import @std/hash

// Tables - use @table literal
let t = @table [{name: "Alice"}, {name: "Bob"}]

// Basil-specific (server context)
let {request, response, params, route, method} = import @basil/http
let {session, auth, user} = import @basil/auth
let {notFound, redirect} = import @basil/api
let {dev} = import @basil/log
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
// Slimmed down for v1.0: type/string/number/format/date validators were removed.
// Use @schema types and constraints for those; see @std/valid docs.
let {uuid, ulid, nanoid, cuid,
     creditCard, luhn, postalCode} = import @std/valid

// ID validators
uuid("550e8400-e29b-41d4-a716-446655440000")  // true
ulid("01ARZ3NDEKTSV4RRFFQ69G5FAV")            // true
nanoid("V1StGXR8_Z5jdHi6B-myT")               // true (default 21 chars)
nanoid("abc", 3)                              // true (custom length)
cuid("cjld2cjxh0000qzrmn831i7rn")             // true

// Financial validators
creditCard("4111111111111111")     // true (Luhn check, 13–19 digits)
luhn("79927398713")                // true (Luhn, any length)

// Locale-aware validators ("US", "GB", "CA")
postalCode("90210", "US")          // true
postalCode("SW1A 1AA", "GB")       // true
postalCode("M5V 2T6", "CA")        // true
```

### Tables

#### Creating Tables
```parsley
// @table literal (preferred) — every row is a dictionary with the same keys
let t = @table [
    {name: "Alice", age: 30},
    {name: "Bob", age: 25}
]

// ❌ WRONG - a header row of arrays is not a @table form
// @table [["name", "age"], ["Alice", 30]]
// Parse error: @table row 1: expected dictionary literal, got LBRACKET

// With schema for validation/defaults
// (a schema default applies only when the field is absent from EVERY row)
@schema Person { name: string, age: integer = 0 }
let people = @table(Person) [
    {name: "Alice"},
    {name: "Bob"}              // age defaults to 0 in both rows
]

// CSV returns Table directly
let fromString = "name,age\nAlice,30".parseCSV()
let fromFile <== CSV(@./data.csv)

// table() builtin (lowercase)
let built = table([{name: "Alice"}, {name: "Bob"}])
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

### HTML Components (`@basil/html`)
```parsley
let {Page, TextField, Button, Form} = import @basil/html

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

### API Helpers (`@basil/api`)
```parsley
let {redirect, notFound, forbidden, badRequest,
     unauthorized, conflict, serverError} = import @basil/api

redirect("/dashboard")              // 302 redirect
redirect("/new-page", 301)          // Permanent redirect
notFound("User not found")          // 404 error
forbidden("Access denied")          // 403 error
badRequest("Invalid input")         // 400 error
unauthorized("Login required")      // 401 error
```

### Schema DSL (`@schema`)

> **Note:** The `@std/schema` module is deprecated. Use the `@schema` DSL instead.

```parsley
// Define schemas using @schema DSL (preferred)
@schema User {
    name: string(min: 1, required)
    email: email(required)
    age: integer(min: 0, max: 150)
    active: boolean
}

// Create and validate records
// A record is NOT validated at construction — call .validate() first
let user = User({name: "Alice", email: "alice@example.com", age: 30, active: true}).validate()
user.isValid()   // true if all fields pass validation
user.errors()    // dictionary of {field: {code, message}} (empty if valid)
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

#### Auto-inserted Hidden ID
When a record has a non-null `id`, `<form @record={...}>` auto-inserts:
```html
<input type="hidden" name="id" value="..."/>
```
This enables edit forms to identify which record to update.

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

### Logging Module (`@basil/log`)
```parsley
let {dev} = import @basil/log       // Exports a single `dev` object

dev.log("Debug info")              // Log to dev panel
dev.log("label", value)            // Log with label
dev.log("label", value, {level: "warn"})  // Log at warn level
dev.clearLog()                     // Clear dev log
dev.logPage("/route", value)       // Log to specific route
dev.setLogRoute("/api")            // Set default route
dev.clearLogPage("/route")         // Clear log for route
```

---



## 🔒 Security Flags (CLI)

```bash
# Development — reads/writes are allowed by default; -x also allows execution
./pars -x script.pars

# Production — deny writes, or deny only specific paths
./pars --no-write script.pars
./pars --restrict-write=/etc script.pars
./pars --allow-execute=./lib script.pars

# Restrict reads
./pars --restrict-read=/etc script.pars
./pars --no-read < data.json  # stdin-only
```

---

## 📝 Method Reference

> **For complete, up-to-date method lists:** Run `pars describe <type>`
> 
> The tables below are verified against the codebase. For the authoritative
> source, use `pars describe string`, `pars describe array`, etc.

### String Methods
| Method | Description | Example |
|--------|-------------|---------|
| `.length()` | Character count | `"hello".length()` → `5` |
| `.toUpper()` | Uppercase | `"hello".toUpper()` → `"HELLO"` |
| `.toLower()` | Lowercase | `"HELLO".toLower()` → `"hello"` |
| `.toTitle()` | Title Case | `"hello world".toTitle()` → `"Hello World"` |
| `.toCamel()` | camelCase | `"hello_world".toCamel()` → `"helloWorld"` |
| `.toPascal()` | PascalCase | `"hello_world".toPascal()` → `"HelloWorld"` |
| `.toSnake()` | snake_case | `"helloWorld".toSnake()` → `"hello_world"` |
| `.toKebab()` | kebab-case | `"helloWorld".toKebab()` → `"hello-world"` |
| `.trim()` | Remove whitespace | `"  hi  ".trim()` → `"hi"` |
| `.collapse()` | Collapse whitespace | `"a  b   c".collapse()` → `"a b c"` |
| `.normalizeSpace()` | Collapse and trim | `"  a  b  ".normalizeSpace()` → `"a b"` |
| `.stripSpace()` | Remove all whitespace | `"a b c".stripSpace()` → `"abc"` |
| `.split(delim)` | Split to array (string or regex) | `"a1b".split(/\d/)` → `["a","b"]` |
| `.replace(old, new)` | Replace all occurrences | `"hi hi".replace("i", "o")` → `"ho ho"` |
| `.includes(substr)` | Contains check | `"hello".includes("ell")` → `true` |
| `.slug()` | URL-safe slug | `"Hello World!".slug()` → `"hello-world"` |
| `.digits()` | Extract digits | `"(555) 123".digits()` → `"555123"` |
| `.truncate(len, suffix?)` | Truncate with suffix | `"hello world".truncate(8)` → `"hello..."` |
| `.indent(n)` | Add leading spaces | `"hi".indent(4)` → `"    hi"` |
| `.outdent()` | Remove common indent | Multi-line dedent |
| `.stripHtml()` | Remove HTML tags | `"<p>Hi</p>".stripHtml()` → `"Hi"` |
| `.htmlEncode()` | Escape for HTML | `"<b>".htmlEncode()` → `"&lt;b&gt;"` |
| `.htmlDecode()` | Decode HTML entities | `"&lt;b&gt;".htmlDecode()` → `"<b>"` |
| `.urlEncode()` | URL encode | `"a b".urlEncode()` → `"a+b"` |
| `.urlDecode()` | URL decode | `"a+b".urlDecode()` → `"a b"` |
| `.toBase64()` | Encode as Base64 | `"hello".toBase64()` → `"aGVsbG8="` |
| `.fromBase64()` | Decode from Base64 | `"aGVsbG8=".fromBase64()` → `"hello"` |
| `.highlight(term)` | Wrap matches | `"hi".highlight("h")` → `"<mark>h</mark>i"` |
| `.paragraphs()` | Text to HTML `<p>` | Blank lines become `<p>` |
| `.parseJSON()` | Parse JSON string | `'{"a":1}'.parseJSON()` → `{a: 1}` |
| `.parseCSV(header?)` | Parse CSV string | `"a,b\n1,2".parseCSV(true)` |
| `.parseMarkdown()` | Parse markdown | `"# Hi".parseMarkdown()` → `{html, raw, md}` |
| `.render(dict?)` | Template interpolation | Uses `\@{key}` syntax in raw strings |
| `.repr()` | Representation string | `"hi".repr()` → `"\"hi\""` |
| `.toJSON()` | JSON string | `"hi".toJSON()` → `"\"hi\""` |

### Array Methods
| Method | Description | Example |
|--------|-------------|---------|
| `.length()` | Array length | `[1,2,3].length()` → `3` |
| `.sort()` | Sort ascending | `[3,1,2].sort()` → `[1,2,3]` |
| `.sortBy(fn)` | Sort by key | `users.sortBy(fn(u){u.age})` |
| `.reverse()` | Reverse order | `[1,2,3].reverse()` → `[3,2,1]` |
| `.shuffle()` | Random order | `[1,2,3].shuffle()` |
| `.pick()` | Random element | `[1,2,3].pick()` → `2` |
| `.take(n)` | n random unique | `[1,2,3,4,5].take(3)` → `[4,1,3]` |
| `.map(fn)` | Transform each | `[1,2].map(fn(x){x*2})` → `[2,4]` |
| `.filter(fn)` | Keep matching | `[1,2,3].filter(fn(x){x>1})` → `[2,3]` |
| `.reduce(fn, init)` | Reduce to value | `[1,2,3].reduce(fn(a,x){a+x}, 0)` → `6` |
| `.join(sep?)` | Join to string | `["a","b"].join(",")` → `"a,b"` |
| `.insert(idx, val)` | Insert at index | `[1,3].insert(1, 2)` → `[1,2,3]` |
| `.format(conj, locale?)` | Conjunction list | `["A","B","C"].format("and")` → `"A, B, and C"` |
| `.toJSON()` | To JSON string | `[1,2].toJSON()` → `"[1,2]"` |
| `.toCSV(header?)` | To CSV string | See reference |

### Dictionary Methods
| Method | Description | Example |
|--------|-------------|---------|
| `.keys()` | Get all keys | `{a:1, b:2}.keys()` → `["a", "b"]` |
| `.values()` | Get all values | `{a:1, b:2}.values()` → `[1, 2]` |
| `.entries()` | Get key-value pairs | `{a:1}.entries()` → `[{key:"a", value:1}]` |
| `.has(key)` | Key exists? | `{a:1}.has("a")` → `true` |
| `.delete(key)` | Remove key (mutates) | `d.delete("b")` removes key from d |
| `.insertBefore(key, newKey, val)` | Insert before key | `{a:1, c:3}.insertBefore("c", "b", 2)` |
| `.insertAfter(key, newKey, val)` | Insert after key | `{a:1, c:3}.insertAfter("a", "b", 2)` |
| `.render(template)` | Template interpolation | Uses `\@{key}` syntax in raw strings |
| `.toJSON()` | To JSON string | `{a:1}.toJSON()` → `"{\"a\":1}"` |

### Number Methods (Integer & Float)
| Method | Description | Example |
|--------|-------------|---------|
| `.abs()` | Absolute value | `(-5).abs()` → `5` |
| `.fmt()` | Default format | `123456.fmt()` → `"123,456"` |
| `.fmt(precision)` | With precision (floats) | `3.14159.fmt(2)` → `"3.14"` |
| `.fmt(style)` | With style | `1234567.fmt("short")` → `"1.2M"` |
| `.fmt(style, locale)` | Style + locale | `1234.fmt("medium", "de-DE")` → `"1.234"` |
| `.short()` | Compact format | `1234567.short()` → `"1.2M"` |
| `.medium()` | Standard format | `123456.medium()` → `"123,456"` |
| `.long()` | Full precision | `1234.5.long()` → `"1,234.50"` |
| `.humanize()` | Human-readable | `1500.humanize()` → `"1.5K"` |
| `.currency(code)` | Currency format | `99.currency("USD")` → `"$ 99.00"` |
| `.percent()` | Percentage | `0.125.percent()` → `"12%"` |
| `.round(n?)` | Round (floats only) | `3.456.round(2)` → `3.46` |
| `.floor()` | Round down (floats) | `3.9.floor()` → `3` |
| `.ceil()` | Round up (floats) | `3.1.ceil()` → `4` |
| `.repr()` | Parseable literal | `42.repr()` → `"42"` |
| `.toJSON()` | JSON representation | `42.toJSON()` → `"42"` |
| `.inspect()` | Debug dictionary | `42.inspect()` → `{__type: "integer", ...}` |

### Money Methods
| Method | Description | Example |
|--------|-------------|---------|
| `.abs()` | Absolute value | `(-$50).abs()` → `$50.00` |
| `.negate()` | Negate amount | `$50.negate()` → `-$50.00` |
| `.fmt()` | Default format | `$1234.56.fmt()` → `"$ 1,234.56"` |
| `.fmt(style)` | With style | `$1234.56.fmt("short")` → `"$1.2K"` |
| `.short()` | Compact format | `$1234567.short()` → `"$1.2M"` |
| `.medium()` | Standard format | `$1234.56.medium()` → `"$ 1,234.56"` |
| `.long()` | Full precision | `$1234.56.long()` → `"$1,234.56"` |
| `.full()` | With currency name | `$1234.56.full()` → `"1,234.56 US dollars"` |
| `.split(n)` | Fair division | `$100.split(3)` → `[$33.34, $33.33, $33.33]` |
| `.repr()` | Parseable literal | `$50.repr()` → `"$50.00"` |
| `.toDict()` | To dictionary | `$50.toDict()` → `{amount: 50, currency: "USD"}` |
| `.toJSON()` | JSON string | `$50.toJSON()` → `"{...}"` |
| `.inspect()` | Debug dictionary | `$50.inspect()` → `{__type: "money", ...}` |

**Money Properties:** `.amount` (cents), `.currency` (code), `.scale` (decimals)

### DateTime Methods & Properties
| Property | Description | Example |
|----------|-------------|---------|
| `.year`, `.month`, `.day` | Date components | `@2024-12-25.year` → `2024` |
| `.hour`, `.minute`, `.second` | Time components | `@14:30.hour` → `14` |
| `.weekday` | Day name | `@2024-12-25.weekday` → `"Wednesday"` |
| `.unix`, `.timestamp` | Unix timestamp | `@2024-12-25.unix` → `1735084800` |
| `.date` | Date string | `@2024-12-25T14:30:00.date` → `"2024-12-25"` |
| `.time` | Time string | `@2024-12-25T14:30:00.time` → `"14:30"` |
| `.iso` | ISO 8601 string | `@2024-12-25.iso` → `"2024-12-25T00:00:00Z"` |
| `.week` | ISO week number | `@2024-12-25.week` → `52` |
| `.dayOfYear` | Day of year | `@2024-12-25.dayOfYear` → `360` |
| `.kind` | Datetime kind | `"date"`, `"datetime"`, `"time"` |

| Method | Description | Example |
|--------|-------------|---------|
| `.format(style?, locale?)` | Format with style | `@2024-12-25.format("full")` → `"Wednesday, December 25, 2024"` |
| `.toDict()` | To dictionary | `@2024-12-25.toDict()` → `{year: 2024, month: 12, ...}` |

### Duration Methods & Properties
| Property | Description | Example |
|----------|-------------|---------|
| `.months` | Month component | `@1y2mo.months` → `14` |
| `.seconds` | Seconds component | `@2h30m.seconds` → `9000` |
| `.days` | Total days (if no months) | `@2d.days` → `2` |
| `.hours` | Total hours (if no months) | `@2d.hours` → `48` |
| `.minutes` | Total minutes (if no months) | `@2h.minutes` → `120` |
| `.totalSeconds` | Total seconds (if no months) | `@2h.totalSeconds` → `7200` |

| Method | Description | Example |
|--------|-------------|---------|
| `.format(style?)` | Format duration | `@2h30m.format("short")` → `"2h30m"` |
| `.toDict()` | To dictionary | `@2h.toDict()` → `{months: 0, seconds: 7200, totalSeconds: 7200}` |

### Unit Properties & Methods
| Property | Type | Description |
|----------|------|-------------|
| `.value` | float | Decoded display value |
| `.unit` | string | Suffix (e.g., `"m"`, `"m2"`, `"ac"`) |
| `.family` | string | `"length"`, `"mass"`, `"data"`, `"temperature"`, `"volume"`, `"area"` |
| `.system` | string | `"SI"` or `"US"` |
| `.max` | unit | Largest representable value for this suffix |
| `.min` | unit | Smallest positive representable value |

| Method | Description | Example |
|--------|-------------|---------|
| `.abs()` | Absolute value | `(#-6m).abs()` → `#6m` |
| `.to(suffix)` | Convert to another unit | `#1mi.to("km")` → `#1.609344km` |
| `.fmt()` | Default format | `#5m.fmt()` → `"5.00m"` |
| `.fmt(precision)` | With precision | `#12.345m.fmt(2)` → `"12.35m"` |
| `.fmt(style)` | With style | `#5m.fmt("long")` → `"5.00 meters"` |
| `.short()` | Compact | `#5m.short()` → `"5m"` |
| `.medium()` | With precision | `#5m.medium()` → `"5.00m"` |
| `.long()` | Full unit name | `#5m.long()` → `"5.00 meters"` |
| `.full()` | With conversion | `#5m.full()` → `"5.00 meters (16.4 ft)"` |
| `.toFraction()` | Fraction string (US) | `#3/8in.toFraction()` → `"3/8\""` |
| `.repr()` | Parseable literal | `#3/8in.repr()` → `"#3/8in"` |
| `.toDict()` | To dictionary | `#12m.toDict()` → `{value: 12, unit: "m", ...}` |
| `.toJSON()` | JSON string | `#5m.toJSON()` → `"{...}"` |
| `.inspect()` | Debug dictionary | `#12m.inspect()` → internal details |

**Compound Formats:** `"ft-in"` (feet-inches), `"lb-oz"` (pounds-ounces), `"gal-qt-pt"`, `"L-mL"`, `"compound"` (auto-detect)

### Array Formatting
| Method | Description | Example |
|--------|-------------|---------|
| `.format("and")` | Conjunction list | `["A", "B", "C"].format("and")` → `"A, B, and C"` |
| `.format("or")` | Disjunction list | `["A", "B", "C"].format("or")` → `"A, B, or C"` |
| `.format("and", locale)` | Localized | `["A", "B", "C"].format("and", "de-DE")` → `"A, B und C"` |

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
`Processed {processed.length()} items`
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
    `API Error: {error}`
} else {
    for (post in data) {
        `Post {post.id}: {post.title}`
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
10. **Tag content does NOT need `{}`** - `<td>row[col]</td>` not `<td>{row[col]}</td>` (Parsley is not JSX!)
11. **Use `repr(value)`** - returns code representation for debugging
12. **Use `builtins()`** - list all available builtin functions

