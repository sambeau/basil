# Parsley Cheat Sheet

Quick reference for Copilot AI agent developing Parsley. Focus on key differences from JavaScript, Python, Rust, and Go.

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
// ✅ CORRECT - C-style comments only
// This is a comment

/* Multi-line
   comments work too */

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

### 4. If Requires Parentheses (Like C/JavaScript)
```parsley
// ❌ WRONG (Go/Python style)
if age >= 18 { "adult" }

// ✅ CORRECT - Parentheses required around condition
if (age >= 18) { "adult" }

// If is an expression (returns value like ternary)
let status = if (age >= 18) "adult" else "minor"

// Can use in concatenation
let msg = "You are " + if (premium) "premium" else "regular"

// Block style
let result = if (x > 0) {
    "positive"
} else if (x < 0) {
    "negative" 
} else {
    "zero"
}
```

### 5. Path Literals Use @
```parsley
// ✅ CORRECT
let path = @./config.json
let url = @https://example.com
let date = @2024-11-29
let time = @14:30
let duration = @1d

// ❌ WRONG
let path = "./config.json"  // This is just a string
```

### 6. No Arrow Functions - Use fn() { }
Parsley uses `fn(x) { body }` syntax for functions. The arrow function syntax `x => body` is **NOT supported**.

```parsley
// ❌ WRONG (JavaScript arrow functions)
arr.map(x => x * 2)
arr.map((a, b) => a + b)
arr.filter(x => x > 0)

// ✅ CORRECT - Use fn() { } syntax
arr.map(fn(x) { x * 2 })
arr.filter(fn(x) { x > 0 })

// Named functions
let double = fn(x) { x * 2 }
let add = fn(a, b) { a + b }

// Multiline functions
let process = fn(items) {
    let filtered = items.filter(fn(x) { x > 0 })
    let doubled = filtered.map(fn(x) { x * 2 })
    doubled
}
```

---

## 📊 Most Used Features (from actual usage data)

### Core (use these constantly)
- `log()` - 710 uses in tests, 523 in examples - **#1 most used**
- `let` - 382 uses - variable declaration
- `if` - 72 uses - conditional expression
- `for` - 44 uses - iteration/mapping

### File I/O (very common)
- `file(@path)` - 46 test uses, 8 example uses
- `JSON(@path)` - 20 uses
- `dir(@path)` - 23 test uses, 15 example uses
- `text(@path)` - 27 test uses, 7 example uses

### String/Array (frequent)
- `len()` - 189 test uses, 14 example uses
- `split()` - 23 test uses, 6 example uses
- `sort()` - 7 uses

### DateTime (common)
- `time()` - 159 test uses, 22 example uses
- `now()` - 28 test uses, 8 example uses

---

## 🔤 Syntax Comparison

### Variables & Functions

| Feature | JavaScript | Python | Parsley |
|---------|-----------|--------|---------||
| Variable | `let x = 5` | `x = 5` | `let x = 5` |
| Destructure | `const {x, y} = obj` | `x, y = obj` | `let {x, y} = obj` |
| Array Destruct | `const [a, b] = arr` | `a, b = arr` | `let [a, b] = arr` |
| Rest (array) | `const [a, ...rest] = arr` | `a, *rest = arr` | `let [a, ...rest] = arr` |
| Rest (dict) | `const {a, ...rest} = obj` | N/A | `let {a, ...rest} = obj` |
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
| String | `` `Hi ${x}` `` | `f"Hi {x}"` | `"Hi {x}"` |
| Regex | `/abc/i` | `re.compile(r"abc", re.I)` | `/abc/i` |
| Null | `null` | `None` | `null` |
| Money | N/A | N/A | `$12.34`, `EUR#50.00` |

---

## 🎯 Key Language Features

### 1. Concatenative/Expression-Based
Everything is an expression that returns a value:
```parsley
// If returns value
let x = if (true) 10 else 20  // x = 10

// For returns array
let squares = for (n in 1..5) { n * n }  // [1, 4, 9, 16, 25]

// Tags return strings
let html = <p>Hello</p>  // "<p>Hello</p>"
```

### 2. String Interpolation (like template literals)
```parsley
let name = "Alice"
let msg = "Hello, {name}!"      // "Hello, Alice!"
let calc = "2 + 2 = {2 + 2}"    // "2 + 2 = 4"

// In attributes
<div class="user-{id}">Content</div>
```

### 3. HTML/XML as First-Class
```parsley
// Tags return strings
<p>Hello</p>                    // "<p>Hello</p>"

// Components are just functions
let Card = fn({title, body}) {
    <div class="card">
        <h3>{title}</h3>
        <p>{body}</p>
    </div>
}

// Use like JSX
<Card title="Welcome" body="Hello world"/>
```

### 4. Literal Syntax with @
```parsley
// Paths
@./relative/path           // Relative to current file
@~/components/header.pars  // Relative to handler root (in Basil)
@/absolute/path            // Absolute filesystem path

// URLs
@https://example.com
@https://api.github.com/users

// Dates/Times
@2024-11-29                  // Date
@2024-11-29T14:30:00        // DateTime
@14:30                       // Time
@14:30:45                    // Time with seconds

// Durations
@1d                          // 1 day
@2h30m                       // 2 hours 30 minutes
@-1w                         // Negative: 1 week ago

// Interpolated (dynamic)
let month = "11"
let day = "29"
let date = @(2024-{month}-{day})  // Builds from variables

// Dynamic imports
let name = "Button"
import @(./components/{name})      // Interpolated path literal
```

### 5. Operators Are Overloaded
```parsley
// Arithmetic
5 + 3                        // 8
"Hello" + " World"           // "Hello World"
@/usr + "local"              // @/usr/local (path join)
@https://api.com + "/v1"     // @https://api.com/v1

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
```

### 6. Money Type (Exact Financial Arithmetic)
```parsley
// Currency symbol literals
$12.34                       // USD
£99.99                       // GBP
€50.00                       // EUR
¥1000                        // JPY (no decimals)

// Compound symbols
CA$25.00                     // Canadian Dollar
AU$50.00                     // Australian Dollar

// CODE# syntax (any 3-letter currency)
USD#12.34                    // Same as $12.34
BTC#1.00000000              // Bitcoin (8 decimals)

// Constructor
money(12.34, "USD")          // $12.34
money(1000, "JPY")           // ¥1000

// Arithmetic (same currency only!)
$10.00 + $5.00               // $15.00
$10.00 * 2                   // $20.00
$10.00 / 3                   // $3.33 (banker's rounding)
-$50.00                      // Negative money

// ❌ ERROR: Cannot mix currencies
$10.00 + £5.00               // Error!

// Properties
$12.34.currency              // "USD"
$12.34.amount                // 1234 (in cents)
$12.34.scale                 // 2

// Methods
$1234.56.format()            // "$ 1,234.56"
(-$50.00).abs()              // $50.00
$100.00.split(3)             // [$33.34, $33.33, $33.33]
```

### 7. File I/O with Special Operators
```parsley
// Read operators
let data <== JSON(@./config.json)       // Read file
let {name, error} <== JSON(@./data.json) // With error capture

// Write operators  
data ==> JSON(@./output.json)           // Write/overwrite
data ==>> text(@./log.txt)              // Append

// Network operators (HTTP/SFTP)
let {response, error} <=/= Fetch(@https://api.example.com)
data =/=> conn(@/remote/file.json).json
```

### 8. Method Chaining
```parsley
// String methods
"hello".toUpper()              // "HELLO"
"  trim  ".trim()           // "trim"
"a,b,c".split(",")          // ["a", "b", "c"]
"hello".includes("ell")     // true (substring check)
"hello world".highlight("world")  // "hello <mark>world</mark>"
"Para 1\n\nPara 2".paragraphs()   // "<p>Para 1</p><p>Para 2</p>"

// Number methods
1234567.humanize()          // "1.2M"
1234.humanize("de")         // "1,2K"

// Array methods
[3,1,2].sort()              // [1, 2, 3]
[1,2,3].reverse()           // [3, 2, 1]
[1,2,3].join(",")           // "1,2,3"
[1,2,3].shuffle()           // [2, 3, 1] (random order)
[1,2,3].pick()              // 2 (random element)
[1,2,3].pick(5)             // [1, 3, 1, 2, 1] (random, allows duplicates)
[1,2,3,4,5].take(3)         // [4, 1, 3] (random, unique)
[1,2,3].includes(2)         // true (membership check)

// Array/string indexing
let arr = [1, 2, 3]
arr[0]                      // 1 (errors if out of bounds)
arr[-1]                     // 3 (negative indices from end)
arr[?99]                    // null (optional: no error if OOB)
arr[?0] ?? "default"        // 1 (combine with null coalesce)
[][?0] ?? "empty"           // "empty" (safe empty array access)

// String indexing same
"hello"[0]                  // "h"
"hello"[?99]                // null

// Path methods
@./file.txt.exists          // true/false
@./file.txt.basename        // "file.txt"
@./file.txt.ext             // "txt"

// Chaining
"  HELLO  ".trim().toLower()  // "hello"
```

---

## 📁 File I/O Patterns

### Factory Functions
```parsley
file(@path)      // Auto-detect format from extension
JSON(@path)      // Parse as JSON
CSV(@path)       // Parse as CSV
markdown(@path)        // Markdown with frontmatter
text(@path)      // Plain text (use for HTML files)
lines(@path)     // Array of lines
bytes(@path)     // Byte array
SVG(@path)       // SVG (strips prolog)
dir(@path)       // Directory listing
```

### Read Patterns
```parsley
// Simple read
let config <== JSON(@./config.json)

// With error handling
let {data, error} <== JSON(@./data.json)
if (error) {
    log("Error:", error)
}

// With fallback
let config <== JSON(@./config.json) ?? {default: true}
```

### Write Patterns
```parsley
// Overwrite
data ==> JSON(@./output.json)

// Append
log_entry ==>> text(@./log.txt)
```

### Stdin/Stdout/Stderr (NEW in v0.14.0)
```parsley
// @- is the Unix convention: stdin for reads, stdout for writes
let data <== JSON(@-)              // Read JSON from stdin
data ==> JSON(@-)                  // Write JSON to stdout

// Explicit aliases also available
let input <== text(@stdin)         // Read text from stdin
"output" ==> text(@stdout)         // Write to stdout
"error" ==> text(@stderr)          // Write to stderr

// Works with all format factories
let lines <== lines(@-)            // Read lines from stdin
let csvData <== CSV(@-)            // Parse CSV from stdin
data ==> YAML(@-)                  // Write YAML to stdout

// Full Unix pipeline example
let input <== JSON(@-)
let output = for (item in input.items) {
    if (item.active) { item }
}
output ==> JSON(@-)
```

### Directory Operations (NEW in v0.12.1)
```parsley
// Create directories
file(@./new-dir).mkdir()
file(@./parent/child).mkdir({parents: true})  // Recursive


// Remove directories
file(@./old-dir).rmdir()
file(@./tree).rmdir({recursive: true})        // With contents

// Works with dir() too
dir(@./test).mkdir()
dir(@./test).rmdir()
```

### File Globbing
```parsley
// Find files matching a pattern
let images = fileList(@./images/*.jpg)
let configs = fileList("~/.config/*.json")

// Iterate over matches
for(f in fileList(@./docs/*.md)) {
    log(f.name)
}

// Read all matching files
for(config in fileList(@./config/*.json)) {
    let data <== config
    log(data)
}
```

---

## 🌐 Network Operations

### HTTP (Fetch)
```parsley
// Simple GET
let {data, error} <=/= Fetch(@https://api.example.com)

// POST with body
let payload = {name: "Alice", age: 30}
let {response, error} =/=> Fetch(@https://api.example.com/users, {
    body: payload
})
```

### SFTP (NEW in v0.12.0)
```parsley
// Connect with SSH key
let conn = SFTP(@sftp://user@host, {
    keyFile: @~/.ssh/id_rsa,
    timeout: @10s
})

// Read remote file
let {config, error} <=/= conn(@/remote/config.json).json

// Write remote file
data =/=> conn(@/remote/output.json).json

// Directory operations
conn(@/remote/new-dir).mkdir()
conn(@/remote/old-dir).rmdir({recursive: true})

// Close connection
conn.close()
```

---

## 🔧 Common Patterns

### Error Handling

**For file operations, use capture pattern:**
```parsley
let {data, error} <== JSON(@./file.json)
if (error) {
    log("Failed:", error)
    return
}
log("Success:", data)
```

**For function/method calls, use `try` expression:**
```parsley
// try returns {result, error} dictionary
let {result, error} = try url("user-input")
if (error != null) {
    log("Invalid URL:", error)
    return
}
log("Parsed:", result)

// With null coalescing for defaults
let parsed = (try time("maybe-invalid")).result ?? now()
```

**Catchable errors (caught by `try`):**
- IO, Network, Database, Format, Value, Security

**Non-catchable errors (fix your code):**
- Type, Arity, Undefined, Index, Parse (these are bugs)

```parsley
// ❌ CANNOT catch type errors - validate first
try url(123)        // Type error propagates (crashes)

// ✅ CAN catch format errors - external input may fail  
try url(":::bad:::") // Format error caught in {error}
```

**User-defined errors with `fail()`:**
```parsley
// Create functions that can fail
let validate = fn(x) {
  if (x < 0) { fail("must be non-negative") }
  x * 2
}

// Caller uses try
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
    <button onclick="{onClick}">{text}</button>
}

// Use component
<Button text="Click Me" onClick="handleClick()"/>

// With children (pass as array)
let Card = fn({title, children}) {
    <div class="card">
        <h3>{title}</h3>
        {children}
    </div>
}

<Card title="Welcome" children={[
    <p>Body content</p>,
    <p>More content</p>
]}/>

// Or use a slot pattern
let Wrapper = fn({slot}) {
    <div class="wrapper">{slot}</div>
}

<Wrapper slot={<p>Content here</p>}/>
```

**Note:** Function rest parameters in simple position (`fn(a, ...rest)`) are not yet supported. Use array destructuring: `fn([a, ...rest])` or dict destructuring: `let {a, ...rest} = obj`.

### Modules
```parsley
// Export from module (utils.pars)
export let greet = fn(name) { "Hello, {name}!" }
export PI = 3.14159
export Logo = <img src="logo.png" alt="Logo"/>

// NEW SYNTAX (recommended) - auto-binds to last path segment
import @./utils             // binds to 'utils'
import @std/math            // binds to 'math'
utils.greet("Alice")
math.floor(3.7)

// With alias
import @std/math as M       // binds to 'M'
M.PI

// Destructuring
{greet, PI} = import @./utils
{floor, ceil} = import @std/math
```

### Standard Library

#### Table Module (`std/table`)
SQL-like operations on arrays of dictionaries:
```parsley
import @std/table
// Or: {table} = import @std/table

let data = [{name: "Alice", age: 30}, {name: "Bob", age: 25}]
let t = table(data)

// Filtering
t.where(fn(row) { row.age > 25 }).rows   // [{name: "Alice", age: 30}]

// Sorting  
t.orderBy("age", "desc").rows            // Sorted by age descending

// Projection
t.select(["name"]).rows                  // [{name: "Alice"}, {name: "Bob"}]

// Pagination
t.limit(10).rows                         // First 10 rows
t.limit(10, 20).rows                     // 10 rows starting at offset 20

// Aggregation
t.count()                                // 2
t.sum("age")                             // 55
t.avg("age")                             // 27.5
t.min("age")                             // 25
t.max("age")                             // 30

// Output
t.toHTML()                               // HTML <table> string
t.toCSV()                                // CSV string (RFC 4180)

// Chaining
table(users)
    .where(fn(u) { u.active })
    .orderBy("name")
    .limit(10)
    .toHTML()
```

#### Math Module (`std/math`)
Mathematical functions and constants. Note: Use `math.log()` for natural log since `log` is a builtin print function.
```parsley
import @std/math
// Or: {floor, sqrt, PI} = import @std/math

// Constants
math.PI                // 3.14159...
math.E                 // 2.71828...
math.TAU               // 6.28318... (2π)

// Rounding (all return integers)
math.floor(3.7)        // 3
math.ceil(3.2)         // 4
math.round(3.5)        // 4
math.trunc(-3.7)       // -3 (toward zero)

// Comparison
math.abs(-5)           // 5
math.sign(-42)         // -1
math.clamp(15, 0, 10)  // 10
// Aggregation (2 args OR array)
math.min(5, 3)         // 3
math.min([1, 2, 3])    // 1
math.max([1, 2, 3])    // 3
math.sum([1, 2, 3])    // 6
math.avg([1, 2, 3])    // 2.0
math.product([2, 3])   // 6

// Statistics (array only)
math.median([1, 2, 3, 4])     // 2.5
math.mode([1, 2, 2, 3])       // 2
math.stddev([1, 2, 3])        // ~0.816
math.variance([1, 2, 3])      // ~0.667
math.range([1, 5, 3])         // 4 (5 - 1)

// Random
math.random()          // [0, 1)
math.random(10)        // [0, 10)
math.random(5, 10)     // [5, 10)
math.randomInt(6)      // [0, 6] (die roll)
math.randomInt(1, 6)   // [1, 6]
math.seed(42)          // For reproducibility

// Powers & Logarithms
math.sqrt(16)          // 4
math.pow(2, 10)        // 1024
math.exp(1)            // e (~2.718)
math.log(math.E)       // 1.0 (natural log)
math.log10(1000)       // 3.0

// Trigonometry (radians)
math.sin(math.PI / 2)  // 1
math.cos(0)            // 1
math.tan(0)            // 0
math.asin(1)           // π/2
math.atan2(1, 1)       // π/4

// Angular conversion
math.degrees(math.PI)  // 180
math.radians(180)      // π

// Geometry & Interpolation
math.hypot(3, 4)       // 5
math.dist(0, 0, 3, 4)  // 5
math.lerp(0, 100, 0.5) // 50
math.map(50, 0, 100, 0, 1)  // 0.5
```

#### Validation Module (`std/valid`)
Validators for form input and data validation. All validators return `true` or `false`.
```parsley
import @std/valid as valid
// Or: {email, minLen, positive} = import @std/valid

// Type validators
valid.string("hello")              // true
valid.number(3.14)                 // true
valid.integer(42)                  // true
valid.boolean(true)                // true
valid.array([1,2,3])               // true
valid.dict({a: 1})                 // true

// String validators
valid.empty("   ")                 // true (whitespace only)
valid.minLen("hello", 3)           // true
valid.maxLen("hello", 10)          // true
valid.length("hello", 3, 10)       // true
valid.matches("abc123", "^[a-z0-9]+$")  // true
valid.alpha("Hello")               // true (letters only)
valid.alphanumeric("abc123")       // true
valid.numeric("123.45")            // true (parseable number)

// Number validators
valid.min(5, 1)                    // true (5 >= 1)
valid.max(5, 10)                   // true (5 <= 10)
valid.between(5, 1, 10)            // true
valid.positive(5)                  // true (> 0)
valid.negative(-5)                 // true (< 0)

// Format validators
valid.email("test@example.com")    // true
valid.url("https://example.com")   // true
valid.uuid("550e8400-e29b-41d4-...") // true
valid.phone("+1 (555) 123-4567")   // true
valid.creditCard("4111111111111111") // true (Luhn check)
valid.time("14:30")                // true

// Date validators (locale: "ISO", "US", "GB")
valid.date("2024-12-25")           // true (default ISO)
valid.date("12/25/2024", "US")     // true
valid.date("25/12/2024", "GB")     // true
valid.date("2024-02-30")           // false (Feb 30!)
valid.parseDate("12/25/2024", "US") // "2024-12-25" or null

// Postal codes (locale: "US", "GB")
valid.postalCode("90210", "US")    // true
valid.postalCode("SW1A 1AA", "GB") // true

// Collection validators
valid.contains([1,2,3], 2)         // true
valid.oneOf("red", ["red","green","blue"]) // true
```

---

## 🎨 String Formatting

### Numbers
```parsley
1234567.format()                  // "1,234,567"
99.99.currency("USD")             // "$99.99"
99.99.currency("EUR", "de-DE")    // "99,99 €"
0.15.percent()                    // "15%"
```

### Dates
```parsley
now().format("short")             // "11/29/24"
now().format("medium")            // "Nov 29, 2024"
now().format("long")              // "November 29, 2024"
now().format("long", "de-DE")     // "29. November 2024"

@2024-11-29.format("full")        // "Friday, November 29, 2024"
```

### Durations
```parsley
@1d.format()                      // "tomorrow"
@-1d.format()                     // "yesterday"
@2h30m.format()                   // "2 hours"
```

### Compact Numbers
```parsley
1234.humanize()                   // "1.2K"
1234567.humanize()                // "1.2M"
1234567890.humanize()             // "1.2B"
1234.humanize("de")               // "1,2K" (German locale)
```

---

## 📝 Text View Helpers

### Highlight Search Matches
```parsley
// Highlight search terms in text (case-insensitive)
text.highlight("search")          // wraps in <mark>
text.highlight("term", "strong")  // wraps in custom tag

// Example:
"Hello World".highlight("world")
// "Hello <mark>World</mark>"

"Search results".highlight("search", "em")
// "<em>Search</em> results"
```

### Convert Text to Paragraphs
```parsley
// Double newlines → <p>, single → <br/>
text.paragraphs()

// Example:
"First para.\n\nSecond para.".paragraphs()
// "<p>First para.</p><p>Second para.</p>"

"Line one\nLine two".paragraphs()
// "<p>Line one<br/>Line two</p>"
```

**Note:** Both methods HTML-escape their input for XSS safety.

---

## 🔍 Type Checking

```parsley
type(42)                          // "INTEGER"
type(3.14)                        // "FLOAT"
type("hi")                        // "STRING"
type([1,2])                       // "ARRAY"
type({x: 1})                      // "DICTIONARY"
type(fn() {})                     // "FUNCTION"
type(null)                        // "NULL"
type(true)                        // "BOOLEAN"
type(@2024-11-29)                 // "DATE"
type(@14:30)                      // "TIME"
type(@1d)                         // "DURATION"
```

---

## 🚀 Quick Examples

### Simple Script
```parsley
// Read, transform, write
let data <== JSON(@./input.json)
let processed = for (item in data) {
    {
        name: item.name.toUpper(),
        score: item.score * 2
    }
}
processed ==> JSON(@./output.json)
log("Processed {len(processed)} items")
```

### HTML Generation
```parsley
let users = [
    {name: "Alice", role: "Admin"},
    {name: "Bob", role: "User"}
]

let UserTable = fn(users) {
    <table>
        <tr><th>Name</th><th>Role</th></tr>
        {for (user in users) {
            <tr>
                <td>{user.name}</td>
                <td>{user.role}</td>
            </tr>
        }}
    </table>
}

<UserTable users={users}/>
```

### API Integration
```parsley
let {posts, error} <=/= Fetch(@https://jsonplaceholder.typicode.com/posts)

if (error) {
    log("API Error:", error)
} else {
    for (post in posts) {
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

---

## 🎯 When Writing Tests/Examples

### Most Common Test Pattern
```parsley
logLine("=== Test Description ===")
let input = [1, 2, 3]
let result = for (n in input) { n * 2 }
log("Result:", result)
log("Length:", len(result))
logLine()  // Blank line
```

### File Operation Pattern
```parsley
// Write test data
testData ==> JSON(@./test-file.json)

// Read it back
let {data, error} <== JSON(@./test-file.json)

// Verify
if (error) {
    log("ERROR:", error)
} else {
    log("SUCCESS:", data)
}

// Cleanup
file(@./test-file.json).remove()
```

### Component Testing Pattern
```parsley
let TestComponent = fn({title, items}) {
    <div>
        <h1>{title}</h1>
        <ul>
            {for (item in items) {
                <li>{item}</li>
            }}
        </ul>
    </div>
}

let result = <TestComponent 
    title="Test" 
    items={["a", "b", "c"]}
/>

log(result)
```

---

## Basil Server Functions

### Cookies

**Reading cookies:**
```parsley
let theme = basil.http.request.cookies.theme ?? "light"
let all = basil.http.request.cookies  // dict of all cookies
```

**Setting cookies:**
```parsley
// Simple value (secure defaults)
basil.http.response.cookies.theme = "dark"

// With options
basil.http.response.cookies.remember = {
    value: token,
    maxAge: @30d,           // Duration literal
    httpOnly: true,
    secure: true,
    sameSite: "Strict"
}

// Delete cookie
basil.http.response.cookies.old = {value: "", maxAge: @0s}
```

**Gotchas:**
- ❌ In dev mode, `secure` defaults to `false` (for localhost)
- ✅ In production, `secure` defaults to `true`
- ✅ `httpOnly: true` always (XSS protection)
- ❌ `sameSite: "None"` requires `secure: true` (auto-set)

---

### Sessions (basil.session)

**Basic session operations:**
```parsley
// Store values
basil.session.set("user_id", 123)
basil.session.set("cart", ["item1", "item2"])

// Retrieve values
let userId = basil.session.get("user_id")           // 123 or null
let cart = basil.session.get("cart", [])            // with default

// Check/delete
basil.session.has("user_id")                        // true
basil.session.delete("user_id")
basil.session.clear()                               // logout
```

**Flash messages (show once then disappear):**
```parsley
// Set flash on redirect
basil.session.flash("success", "Profile updated!")
redirect("/profile")

// Display flash on target page  
if (basil.session.hasFlash()) {
    let msg = basil.session.getFlash("success")
    if (msg != null) {
        <div class="alert">{msg}</div>
    }
}
```

**Configuration (basil.yaml):**
```yaml
session:
  secret: "32-char-secret-key"  # Required in production
  max_age: 24h                  # Session lifetime
```

**Gotchas:**
- ❌ In dev mode, random secret = sessions don't persist across restarts
- ✅ In production, must configure explicit `secret`
- ❌ ~4KB max data (stored in encrypted cookie)
- ✅ Use `flash()` for one-time messages (auto-deleted on read)

---

### CSRF Protection

**Forms with auth require CSRF token:**
```parsley
<form method=POST action="/submit">
    <input type=hidden name=_csrf value={basil.csrf.token}/>
    <input type=text name=email/>
    <button>Submit</button>
</form>
```

**For AJAX, use meta tag + header:**
```parsley
// In <head>
<meta name=csrf-token content={basil.csrf.token}/>
```
```javascript
// In JavaScript
fetch('/submit', {
    method: 'POST',
    headers: {
        'X-CSRF-Token': document.querySelector('meta[name=csrf-token]').content
    },
    body: JSON.stringify(data)
})
```

**When CSRF is validated:**
| Route Type | Method | CSRF Check |
|------------|--------|------------|
| `auth: required` | GET/HEAD/OPTIONS | ❌ Skip |
| `auth: required` | POST/PUT/PATCH/DELETE | ✅ Validate |
| `auth: optional` | POST/PUT/PATCH/DELETE | ✅ Validate |
| `type: api` | Any | ❌ Skip (uses API keys) |
| No auth | Any | ❌ Skip |

**Gotchas:**
- ❌ Missing or invalid token → 403 Forbidden
- ✅ Token stored in HttpOnly cookie (auto-managed)
- ❌ API routes don't need CSRF (use API keys instead)
- ✅ Same token works across tabs/back button (per-session)

---

### Redirects

**Return redirects from handlers:**
```parsley
import @std/api

redirect("/dashboard")              // 302 (default)
redirect("/new-page", 301)          // 301 permanent
redirect(@/users/profile)           // path literal
redirect("https://example.com")     // external URL
```

**Common status codes:**
| Code | Name | Use |
|------|------|-----|
| 301 | Moved Permanently | SEO-safe permanent move |
| 302 | Found | Temporary redirect (default) |
| 303 | See Other | After POST (PRG pattern) |
| 307 | Temporary Redirect | Preserve method |
| 308 | Permanent Redirect | Permanent + preserve method |

**Gotchas:**
- ❌ Status must be 3xx (300-308) or error
- ✅ `redirect()` returns immediately - handler exits
- ❌ No response body with redirects

---

### Path Pattern Matching

**Extract parameters from URL paths:**

```parsley
// Basic parameter
let params = match("/users/123", "/users/:id")
// → {id: "123"}

// Multiple parameters
let params = match(path, "/users/:userId/posts/:postId")
// → {userId: "42", postId: "99"}

// Glob capture (remaining segments as array)
let params = match("/files/a/b/c", "/files/*path")
// → {path: ["a", "b", "c"]}

// No match returns null
match("/posts/123", "/users/:id")  // → null
```

**Pattern syntax:**
| Pattern | Captures |
|---------|----------|
| `:name` | Single segment as string |
| `*name` | Rest of path as array |
| `literal` | Must match exactly |

**Route dispatch pattern:**
```parsley
let path = basil.http.request.path

if (let p = match(path, "/users/:id")) {
    showUser(p.id)
}
else if (let p = match(path, "/files/*rest")) {
    serveFile(p.rest.join("/"))
}
```

**Gotchas:**
- ✅ Trailing slashes normalized: `/users/123/` matches `/users/:id`
- ❌ Case sensitive: `/Users/123` doesn't match `/users/:id`
- ❌ Extra segments fail: `/users/123/extra` doesn't match `/users/:id`

---

### Site Mode - Filesystem-Based Routing

Configure in YAML:
```yaml
site: ./site  # Use instead of routes:
```

Walk-back routing finds `index.pars` files:
```
/reports/2025/Q4/  →  site/reports/index.pars (if site/reports/2025/Q4/index.pars doesn't exist)
```

**Access remaining path via subpath:**
```parsley
// In site/reports/index.pars for /reports/2025/Q4/
let segments = basil.http.request.subpath.segments  // ["2025", "Q4"]
let year = segments[0]  // "2025"
```

**Gotchas:**
- ❌ `site:` and `routes:` are mutually exclusive
- ✅ `/path` auto-redirects to `/path/` for directories
- ❌ Dotfiles (`.git/`) and path traversal (`..`) are blocked

---

### publicUrl() - Component Assets

Make private files (like SVGs in component folders) publicly accessible:

```parsley
// In modules/Button.pars
let icon = publicUrl(@./icon.svg)
<img src={icon}/>
// Output: <img src="/__p/a3f2b1c8.svg"/>
```

**Key Features:**
- Content-hashed URLs for automatic cache-busting
- Aggressive caching (`max-age=31536000`)
- File stays in place (no copying)
- Only in Basil handlers (not CLI)

**Gotchas:**
- ❌ Files >100MB fail - use `public/` folder
- ❌ Path must be within handler directory
- ✅ Works with `@./relative` paths from current file
