# Parsley Cheat Sheet

Quick reference for beginners and AI agents developing Parsley. Focus on key differences from JavaScript, Python, Rust, and Go.

---

## 🚨 Major Gotchas (Common Mistakes)

### 1. Output Functions
```parsley
// ❌ WRONG (JavaScript/Python style)
print("hello")
println("hello")
console.log("hello")

// ✅ CORRECT
log("hello")           // Most common - concatenates args with spaces
logLine("hello")       // Includes line number - USE FOR DEBUGGING
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

### 8. String/Tag nterpolation Uses {var} Not ${var}
```parsley
// ❌ WRONG (JavaScript style)
<div class="user-${id}">

// ✅ CORRECT - no $ needed
<div class="user-{id}">
"Hello, {name}!"
```

### 9. Self-Closing Tags MUST Use />
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

### Data Types

| Type | JavaScript | Python | Parsley |
|------|-----------|--------|---------|
| Array | `[1, 2, 3]` | `[1, 2, 3]` | `[1, 2, 3]` |
| Dict | `{x: 1, y: 2}` | `{"x": 1, "y": 2}` | `{x: 1, y: 2}` |
| String | `` `Hi ${x}` `` | `f"Hi {x}"` | `` `Hi {x}` `` |
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

// In attributes (interpolation in expression values)
<div class="user-{id}">"Content"</div>
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
let db = SQLITE(@./data.db)

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
$12.34.amount                // 1234 (in cents)
$1234.56.format()            // "$ 1,234.56"
$100.00.split(3)             // [$33.34, $33.33, $33.33]
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

---

## 📚 Standard Library (@std)

### Module Quick Reference

```parsley
// Standard library
let {table} = import @std/table
let {dev} = import @std/dev
let {PI, sin, cos, floor} = import @std/math
let {email, minLen, url} = import @std/valid
let {string, object, validate} = import @std/schema
let {uuid, nanoid, new} = import @std/id
let {notFound, redirect} = import @std/api
let {md} = import @std/markdown
let {mdDoc} = import @std/mdDoc

// Basil context (available in handlers)
let {request, response, query, route, method} = import @basil/http
let {db, session, auth, user} = import @basil/auth
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

### Table Module (`@std/table`)
```parsley
let {table} = import @std/table

let data = [{name: "Alice", age: 30}, {name: "Bob", age: 25}]
let t = table(data)

// Filtering
t.where(fn(row) { row.age > 25 }).rows

// Sorting  
t.orderBy("age", "desc").rows

// Aggregation
t.count()                          // 2
t.sum("age")                       // 55
t.avg("age")                       // 27.5

// Output
t.toHTML()                         // HTML <table> string
t.toCSV()                          // CSV string
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
     object, array, define, validate} = import @std/schema

// Define a schema
let UserSchema = define({
    name: string().minLen(1),
    email: email(),
    age: integer().min(0).max(150),
    active: boolean()
})

// Validate data
let {valid, errors} = validate(UserSchema, {
    name: "Alice",
    email: "alice@example.com",
    age: 30,
    active: true
})
```

### ID Module (`@std/id`)
```parsley
let {new, uuid, uuidv7, nanoid, cuid} = import @std/id

new()                              // ULID-like (sortable, URL-safe)
uuid()                             // UUID v4
uuidv7()                           // UUID v7 (time-sortable)
nanoid()                           // NanoID
cuid()                             // CUID
```

### Markdown Module (`@std/markdown`)
```parsley
let {md} = import @std/markdown

let doc = md.parse("# Hello\n\nWorld")

// Query
doc.title()                        // "Hello"
doc.headings()                     // [{level: 1, text: "Hello"}]
doc.links()                        // Array of links
doc.codeBlocks()                   // Array of code blocks
doc.toc()                          // Table of contents
doc.wordCount()                    // Word count
doc.text()                         // Plain text

// Convert
doc.toHTML()                       // HTML string
doc.toMarkdown()                   // Markdown string
```

### Dev Module (`@std/dev`)
```parsley
let {dev} = import @std/dev

dev.log("Debug info")              // Log to dev console
dev.clearLog()                     // Clear dev log
dev.logPage()                      // Get log page HTML
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
let {db, session, auth, user} = import @basil/auth

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

// Database
db <=?=> "SELECT * FROM users WHERE id = ?" [userId]
```

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

// Auto-refresh every second
<Part src={@~/parts/clock.part} part-refresh={1000}/>

// Load immediately after page (for slow data)
<Part src={@~/parts/data.part} view="placeholder" part-load="loaded"/>

// Lazy load when scrolled into view
<Part src={@~/parts/content.part} view="placeholder" 
      part-lazy="loaded" part-lazy-threshold={150}/>
```

### Part Attributes
| Attribute | Element | Effect |
|-----------|---------|--------|
| `part-click="view"` | Any | Fetches view on click |
| `part-submit="view"` | `<form>` | Fetches view on submit |
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
| `.trim()` | Remove whitespace | `"  hi  ".trim()` → `"hi"` |
| `.split(delim)` | Split to array | `"a,b".split(",")` → `["a","b"]` |
| `.replace(old, new)` | Replace text | `"hi".replace("i", "o")` → `"ho"` |
| `.includes(substr)` | Contains check | `"hello".includes("ell")` → `true` |
| `.slug()` | URL-safe slug | `"Hello World!".slug()` → `"hello-world"` |
| `.digits()` | Extract digits | `"(555) 123".digits()` → `"555123"` |
| `.stripHtml()` | Remove HTML tags | `"<p>Hi</p>".stripHtml()` → `"Hi"` |
| `.highlight(term)` | Wrap matches | `"hi".highlight("h")` → `"<mark>h</mark>i"` |
| `.paragraphs()` | Text to HTML `<p>` | See reference |
| `.parseJSON()` | Parse JSON string | `'{"a":1}'.parseJSON()` → `{a: 1}` |
| `.parseCSV(header?)` | Parse CSV string | `"a,b\n1,2".parseCSV(true)` |

### Array Methods
| Method | Description | Example |
|--------|-------------|---------|
| `.length()` | Array length | `[1,2,3].length()` → `3` |
| `.sort()` | Sort ascending | `[3,1,2].sort()` → `[1,2,3]` |
| `.reverse()` | Reverse order | `[1,2,3].reverse()` → `[3,2,1]` |
| `.shuffle()` | Random order | `[1,2,3].shuffle()` |
| `.pick()` | Random element | `[1,2,3].pick()` → `2` |
| `.take(n)` | n random unique | `[1,2,3,4,5].take(3)` → `[4,1,3]` |
| `.map(fn)` | Transform each | `[1,2].map(fn(x){x*2})` → `[2,4]` |
| `.filter(fn)` | Keep matching | `[1,2,3].filter(fn(x){x>1})` → `[2,3]` |
| `.join(sep?)` | Join to string | `["a","b"].join(",")` → `"a,b"` |
| `.has(item)` | Contains check | `[1,2,3].has(2)` → `true` |
| `.toJSON()` | To JSON string | `[1,2].toJSON()` → `"[1,2]"` |
| `.toCSV(header?)` | To CSV string | See reference |

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

1. **Use `logLine()` in multi-line scripts** - shows line numbers
2. **Check types with `type()`** when confused
3. **Remember `for` returns an array** - don't expect side effects
4. **Use error capture** `{data, error}` for file/network ops
5. **Path literals need @** - `@./file` not `"./file"`
6. **Comments are //** not #
7. **Output is `log()`** not `print()` or `console.log()`
8. **Strings in HTML need quotes** - `<p>"text"</p>` not `<p>text</p>`
9. **Self-closing tags need />** - `<br/>` not `<br>`
10. **Code between tags does not need `{}` - ``<p>text</p>`` not ``<p>{text}</p>``

