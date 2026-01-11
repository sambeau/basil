# Parsley Language Reference

> Verified against source code (January 2026)

## Table of Contents

1. [Literals](#1-literals)
2. [Operators](#2-operators)
3. [Control Flow](#3-control-flow)
4. [Statements](#4-statements)
5. [Type Methods](#5-type-methods)
6. [Builtin Functions](#6-builtin-functions)
7. [Standard Library](#7-standard-library)
8. [Tags (HTML/XML)](#8-tags-htmlxml)
9. [Error Handling](#9-error-handling)
10. [Comments](#10-comments)
11. [Reserved Keywords](#11-reserved-keywords)

---

## 1. Literals

### 1.1 Numbers

#### Integers

```parsley
42
-15
1_000_000      // Underscores for readability
0x1F           // Hexadecimal
0b1010         // Binary
0o755          // Octal
```

#### Floats

```parsley
3.14159
-2.718
1.0e10         // Scientific notation
```

---

### 1.2 Strings

#### Double-Quoted Strings (`"..."`)

Standard strings with escape sequences. **No interpolation**.

```parsley
"Hello, World!"
"Line 1\nLine 2"       // Newline escape
"Tab\there"            // Tab escape
"Quote: \"hi\""        // Escaped quote
"Path: C:\\Users"      // Escaped backslash
```

| Escape | Meaning |
|--------|---------|
| `\n` | Newline |
| `\t` | Tab |
| `\r` | Carriage return |
| `\\` | Backslash |
| `\"` | Double quote |

#### Backtick Template Strings (`` `...` ``)

Interpolated strings with `{expression}` syntax.

```parsley
let name = "Alice"
let greeting = `Hello, {name}!`           // "Hello, Alice!"
let calc = `2 + 2 = {2 + 2}`              // "2 + 2 = 4"
let nested = `{user.name} is {user.age}`  // Expressions allowed
```

- Multi-line supported
- Use `\{` for literal brace

#### Single-Quoted Raw Strings (`'...'`)

Backslashes are literal. Interpolation only with `@{expression}`.

```parsley
'C:\Users\name'                    // Backslashes literal
'regex: \d+\.\d+'                  // No escaping needed
'Parts.refresh("editor", {id: 1})' // JS code with literal braces

// Interpolation with @{}
let id = 42
'Parts.refresh("editor", {id: @{id}})'  // id interpolated

// Escape @ with \@
'email: user\@example.com'         // Literal @
```

---

### 1.3 Booleans and Null

```parsley
true
false
null
```

**Truthy values**: Everything except `false`, `null`, `0`, `0.0`, `""`, `[]`

**Falsy values**: `false`, `null`, `0`, `0.0`, `""` (empty string), `[]` (empty array)

---

### 1.4 Arrays

```parsley
[1, 2, 3]
["a", "b", "c"]
[1, "mixed", true, null]
[[1, 2], [3, 4]]           // Nested
[]                         // Empty
```

#### Indexing

```parsley
arr[0]                     // First element
arr[-1]                    // Last element
arr[1:3]                   // Slice: elements 1 and 2
arr[2:]                    // From index 2 to end
arr[:2]                    // From start to index 2
arr[?99]                   // Optional: null if out of bounds
```

#### Destructuring

```parsley
let [a, b, c] = [1, 2, 3]          // a=1, b=2, c=3
let [first, ...rest] = [1, 2, 3]   // first=1, rest=[2, 3]
let [_, second] = arr              // Skip first element
```

---

### 1.5 Dictionaries

```parsley
{name: "Alice", age: 30}
{x: 1, y: 2}
{"key with spaces": value}         // Quoted keys
{}                                 // Empty
```

#### Access

```parsley
dict.name                          // Dot notation
dict["name"]                       // Bracket notation
dict.missing                       // Returns null (no error)
```

#### Destructuring

```parsley
let {name, age} = user             // Extract fields
let {name: n, age: a} = user       // Rename fields
let {name, ...rest} = user         // Rest pattern
```

#### Self-Reference

```parsley
let config = {
    width: 100,
    height: 200,
    area: this.width * this.height // Computed on access
}
```

---

### 1.6 Functions

```parsley
fn(x) { x * 2 }                    // Anonymous function
fn(x, y) { x + y }                 // Multiple parameters
fn() { "constant" }                // No parameters

let double = fn(x) { x * 2 }       // Named via let
let greet = fn(name) {
    `Hello, {name}!`
}
```

#### Implicit Return

The last expression is returned automatically:

```parsley
let add = fn(a, b) { a + b }       // Returns sum
let complex = fn(x) {
    let y = x * 2
    y + 1                          // Returns this
}
```

---

### 1.7 DateTime Literals

All datetime literals start with `@`:

```parsley
@2024-12-25                        // Date only
@2024-12-25T14:30:00               // DateTime
@2024-12-25T14:30:00Z              // DateTime UTC
@14:30                             // Time only
@14:30:45                          // Time with seconds
```

#### Special Values

```parsley
@now                               // Current datetime
@today                             // Current date (alias: @dateNow)
@timeNow                           // Current time
```

#### Interpolated DateTime

```parsley
let month = 12
let day = 25
@(2024-{month}-{day})              // Dynamic construction
```

---

### 1.8 Duration Literals

```parsley
@1d                                // 1 day
@2h30m                             // 2 hours 30 minutes
@1w                                // 1 week
@1y6mo                             // 1 year 6 months
@-1d                               // Negative: 1 day ago
```

| Unit | Meaning |
|------|---------|
| `y` | Year |
| `mo` | Month |
| `w` | Week |
| `d` | Day |
| `h` | Hour |
| `m` | Minute |
| `s` | Second |

---

### 1.9 Path Literals

```parsley
@./relative/path                   // Relative to current file
@~/from/root                       // Relative to project root
@/absolute/path                    // Absolute filesystem path
@-                                 // stdin/stdout
@stdin                             // Explicit stdin
@stdout                            // Explicit stdout
@stderr                            // Explicit stderr
```

#### Interpolated Paths

```parsley
let file = "config"
@(./data/{file}.json)              // Dynamic path
```

---

### 1.10 URL Literals

```parsley
@http://example.com
@https://api.github.com/users
@ftp://files.example.com
```

#### Interpolated URLs

```parsley
let id = 123
@(https://api.example.com/users/{id})
```

---

### 1.11 Money Literals

#### Symbol Formats

```parsley
$12.34                             // USD
£99.99                             // GBP
€50.00                             // EUR
¥1000                              // JPY (no decimals)
CA$25.00                           // Canadian Dollar
AU$25.00                           // Australian Dollar
```

#### CODE# Format

```parsley
USD#12.34                          // Any ISO currency
EUR#50.00
BTC#0.00100000                     // 8 decimal places
```

---

### 1.12 Regex Literals

```parsley
/pattern/                          // Basic regex
/\d+/                              // Digits
/hello/i                           // Case insensitive
/^start.*end$/m                    // Multiline
```

| Flag | Meaning |
|------|---------|
| `i` | Case insensitive |
| `m` | Multiline |
| `s` | Dotall (`.` matches newline) |
| `g` | Global (all matches) |

---

### 1.13 Connection Literals

```parsley
@sqlite(@./database.db)            // SQLite connection
@postgres(@postgres://...)         // PostgreSQL
@mysql(@mysql://...)               // MySQL
@sftp(@sftp://user@host)           // SFTP connection
@shell                             // Shell executor
```

---

## 2. Operators

### 2.1 Arithmetic

| Operator | Types | Result | Description |
|----------|-------|--------|-------------|
| `+` | int, int | int | Addition |
| `+` | float, float | float | Addition |
| `+` | string, any | string | Concatenation |
| `+` | path, string | path | Path join |
| `+` | datetime, duration | datetime | Add time |
| `+` | money, money | money | Add (same currency) |
| `-` | number, number | number | Subtraction |
| `-` | datetime, datetime | duration | Time difference |
| `-` | datetime, duration | datetime | Subtract time |
| `-` | array, array | array | Set difference |
| `*` | number, number | number | Multiplication |
| `*` | string, int | string | Repetition |
| `*` | array, int | array | Repetition |
| `*` | money, number | money | Scale |
| `/` | number, number | number | Division |
| `/` | array, int | array[] | Chunking |
| `%` | int, int | int | Modulo |

```parsley
5 + 3                              // 8
"ab" * 3                           // "ababab"
[1, 2] * 2                         // [1, 2, 1, 2]
[1,2,3,4,5,6] / 2                  // [[1,2], [3,4], [5,6]]
```

---

### 2.2 Comparison

| Operator | Description |
|----------|-------------|
| `==` | Equal |
| `!=` | Not equal |
| `<` | Less than |
| `>` | Greater than |
| `<=` | Less than or equal |
| `>=` | Greater than or equal |

String comparison uses **natural sort order**:
```parsley
"file2" < "file10"                 // true (not lexicographic)
```

---

### 2.3 Logical

| Operator | Description |
|----------|-------------|
| `&&` / `and` | Logical AND (short-circuit) |
| `\|\|` / `or` | Logical OR (short-circuit) |
| `!` / `not` | Logical NOT |

**Set operations on collections:**

```parsley
[1,2,3] && [2,3,4]                 // [2, 3] (intersection)
[1,2] || [2,3]                     // [1, 2, 3] (union)
{a:1, b:2} && {b:3, c:4}           // {b: 2} (intersection)
```

**DateTime intersection (`&&`):**

```parsley
@2024-12-25 && @14:30              // Combine date + time
```

---

### 2.4 Membership

| Operator | Description |
|----------|-------------|
| `in` | Membership test |
| `not in` | Negated membership |

```parsley
2 in [1, 2, 3]                     // true
"name" in {name: "Sam"}            // true (key exists)
"ell" in "hello"                   // true (substring)
"x" not in [1, 2, 3]               // true
```

---

### 2.5 Pattern Matching

| Operator | Left | Right | Result |
|----------|------|-------|--------|
| `~` | string | regex | array or null |
| `!~` | string | regex | boolean |

```parsley
"hello123" ~ /\d+/                 // ["123"]
"hello" !~ /\d+/                   // true
```

---

### 2.6 Range

```parsley
1..5                               // [1, 2, 3, 4, 5]
```

---

### 2.7 Concatenation

```parsley
[1, 2] ++ [3, 4]                   // [1, 2, 3, 4]
{a: 1} ++ {b: 2}                   // {a: 1, b: 2}
```

---

### 2.8 Null Coalescing

```parsley
value ?? "default"                 // Returns "default" if value is null
a ?? b ?? c                        // First non-null
```

#### Optional Index Access

Use `[?index]` syntax (question mark inside brackets):

```parsley
arr[?99]                           // null if out of bounds
dict[?"missing"]                   // null if key missing
```

**Note**: `?.` optional chaining is NOT supported. Use `[?key]` instead.

---

### 2.9 File I/O Operators

| Operator | Description |
|----------|-------------|
| `<==` | Read from file |
| `<=/=` | Fetch from URL |
| `==>` | Write to file |
| `==>>` | Append to file |

```parsley
let data <== JSON(@./config.json)
data ==> JSON(@./output.json)
line ==>> text(@./log.txt)
let response <=/= JSON(@https://api.example.com/data)
```

---

### 2.10 Database Operators

| Operator | Description | Returns |
|----------|-------------|---------|
| `<=?=>` | Query single row | dict or null |
| `<=??=>` | Query multiple rows | array |
| `<=!=>` | Execute mutation | result dict |

```parsley
let db = @sqlite(@./app.db)
let user = db <=?=> "SELECT * FROM users WHERE id = ?"(id)
let users = db <=??=> "SELECT * FROM users"
let result = db <=!=> "INSERT INTO users (name) VALUES (?)"(name)
```

---

### 2.11 Process Execution

| Operator | Description |
|----------|-------------|
| `<=#=>` | Execute command with input |

---

### 2.12 Precedence Table

From lowest to highest:

| Level | Operators |
|-------|-----------|
| 1 | `??`, `\|\|`, `or` |
| 2 | `&&`, `and` |
| 3 | `==`, `!=`, `~`, `!~`, `in`, `not in` |
| 4 | `<`, `>`, `<=`, `>=` |
| 5 | `+`, `-`, `..` |
| 6 | `++` |
| 7 | `*`, `/`, `%` |
| 8 | `-`, `!`, `not` (prefix) |
| 9 | `.`, `[]`, `()` (access/call) |

---

## 3. Control Flow

### 3.1 If Expressions

If is an **expression** that returns a value:

```parsley
let status = if (age >= 18) "adult" else "minor"
```

#### Block Form

```parsley
if (condition) {
    // body
} else if (other) {
    // body
} else {
    // body
}
```

#### Parentheses

Parentheses are optional but recommended:

```parsley
if age >= 18 { "adult" }           // Works
if (age >= 18) { "adult" }         // Recommended
```

---

### 3.2 For Expressions

For is an **expression** that returns an array (like `map`):

```parsley
let doubled = for (n in [1,2,3]) { n * 2 }  // [2, 4, 6]
```

#### With Index

```parsley
for (i, item in items) {
    `{i}: {item}`
}
```

#### Filter Pattern

Return nothing (or `null`) to omit from result:

```parsley
let evens = for (n in 1..10) {
    if (n % 2 == 0) { n }          // Only even numbers
}
// [2, 4, 6, 8, 10]
```

#### Loop Control

```parsley
stop                               // Exit loop (like break)
skip                               // Skip iteration (like continue)
```

```parsley
let firstFive = for (x in 1..100) {
    if (x > 5) stop
    x
}
// [1, 2, 3, 4, 5]
```

---

### 3.3 Try Expressions

Capture errors as values:

```parsley
let {result, error} = try riskyOperation()

if (error != null) {
    log("Failed:", error)
}
```

---

### 3.4 Check Guard

Early return if condition fails:

```parsley
check condition else fallbackValue
```

```parsley
let validate = fn(x) {
    check x > 0 else "must be positive"
    check x < 100 else "must be under 100"
    x * 2
}
```

---

## 4. Statements

### 4.1 Let Binding

```parsley
let x = 5
let name = "Alice"
let {a, b} = dict                  // Destructuring
let [first, ...rest] = arr         // Array destructuring
```

---

### 4.2 Assignment

```parsley
x = 10                             // Reassign
dict.key = value                   // Property assignment
arr[0] = value                     // Index assignment
```

---

### 4.3 Export

```parsley
export let greeting = "Hello"
export PI = 3.14159
export MyComponent = fn(props) { ... }
```

---

### 4.4 Import

```parsley
import @./module                   // Import as 'module'
import @./module as M              // Import with alias
let {func1, func2} = import @./module  // Destructure exports
```

#### Standard Library

```parsley
import @std/math
let {floor, ceil} = import @std/math
```

---

### 4.5 Return, Stop, Skip

```parsley
return value                       // Return from function
stop                               // Exit for loop
skip                               // Continue to next iteration
```

---

## 5. Type Methods

### 5.0 Universal Methods

All values have these methods:

| Method | Description |
|--------|-------------|
| `.type()` | Returns type name as string |

```parsley
"hello".type()                     // "string"
42.type()                          // "integer"
[1,2].type()                       // "array"
null.type()                        // "null"
```

---

### 5.1 String Methods

| Method | Description |
|--------|-------------|
| `.length()` | Character count |
| `.toUpper()` | Uppercase |
| `.toLower()` | Lowercase |
| `.toTitle()` | Title Case Each Word |
| `.trim()` | Remove surrounding whitespace |
| `.split(delim)` | Split to array |
| `.replace(old, new)` | Replace occurrences |
| `.includes(substr)` | Contains substring? |
| `.highlight(phrase, tag?)` | Wrap matches in HTML tag |
| `.paragraphs()` | Convert plain text to HTML paragraphs |
| `.render(dict?)` | Render `@{...}` interpolation |
| `.parseMarkdown(options?)` | Parse markdown to dict |
| `.parseJSON()` | Parse as JSON |
| `.parseCSV(header?)` | Parse as CSV |
| `.collapse()` | Collapse whitespace to single spaces |
| `.normalizeSpace()` | Collapse + trim |
| `.stripSpace()` | Remove all whitespace |
| `.stripHtml()` | Remove HTML tags |
| `.digits()` | Extract only digits |
| `.slug()` | URL-safe slug |
| `.htmlEncode()` | Escape for HTML |
| `.htmlDecode()` | Decode HTML entities |
| `.urlEncode()` | URL encode (query string) |
| `.urlDecode()` | URL decode |
| `.urlPathEncode()` | URL encode for path segments |
| `.urlQueryEncode()` | URL encode for query values |
| `.outdent()` | Remove common leading whitespace |
| `.indent(n)` | Add n spaces to line starts |

---

### 5.2 Array Methods

| Method | Description |
|--------|-------------|
| `.length()` | Element count |
| `.reverse()` | Reverse order |
| `.sort(options?)` | Sort (natural order by default) |
| `.sortBy(fn)` | Sort by key function |
| `.map(fn)` | Transform elements |
| `.filter(fn)` | Keep matching elements |
| `.reduce(fn, init)` | Reduce to single value |
| `.format(style?, locale?)` | Format as prose list |
| `.join(sep?)` | Join to string |
| `.toJSON()` | Convert to JSON string |
| `.toCSV(header?)` | Convert to CSV string |
| `.shuffle()` | Random order |
| `.pick()` | Random element |
| `.pick(n)` | n random elements (with replacement) |
| `.take(n)` | n unique random elements |
| `.insert(i, value)` | Insert at index |
| `.has(item)` | Contains item? |
| `.hasAny(arr)` | Contains any of items? |
| `.hasAll(arr)` | Contains all items? |

---

### 5.3 Dictionary Methods

| Method | Description |
|--------|-------------|
| `.keys()` | Array of keys |
| `.values()` | Array of values |
| `.entries(k?, v?)` | Array of {key, value} dicts |
| `.has(key)` | Key exists? |
| `.delete(key)` | Remove key (mutates) |
| `.insertAfter(after, key, val)` | Insert after existing key |
| `.insertBefore(before, key, val)` | Insert before existing key |
| `.render(template)` | Render template with values |
| `.toJSON()` | Convert to JSON string |

---

### 5.4 Number Methods

| Method | Description |
|--------|-------------|
| `.format(locale?)` | Locale-formatted string |
| `.currency(code, locale?)` | Currency format |
| `.percent(locale?)` | Percentage format |
| `.humanize(locale?)` | Compact format (1.2M) |

**Note**: For math operations like `abs()`, `round()`, `floor()`, `ceil()`, use `@std/math`.

---

### 5.5 DateTime Properties & Methods

DateTime values have these properties:

| Property | Description |
|----------|-------------|
| `.year` | Year component |
| `.month` | Month (1-12) |
| `.day` | Day of month |
| `.hour` | Hour (0-23) |
| `.minute` | Minute (0-59) |
| `.second` | Second (0-59) |
| `.weekday` | Day name ("Wednesday") |
| `.dayOfYear` | Day of year (1-366) |
| `.week` | ISO week number |
| `.unix` / `.timestamp` | Unix timestamp |
| `.iso` | ISO 8601 string |

| Method | Description |
|--------|-------------|
| `.format(style?, locale?)` | Format datetime |
| `.toDict()` | Return raw dictionary |

**DateTime arithmetic**: Use operators `+` and `-` with durations:
```parsley
@now + @1d                         // Tomorrow
@now - @1w                         // One week ago
```

---

### 5.6 Duration Properties & Methods

Duration values have these properties:

| Property | Description |
|----------|-------------|
| `.months` | Months component |
| `.seconds` | Seconds component |
| `.totalSeconds` | Total as seconds |
| `.totalMinutes` | Total as minutes |
| `.totalHours` | Total as hours |
| `.totalDays` | Total as days |

| Method | Description |
|--------|-------------|
| `.format(locale?)` | Human-readable format |
| `.toDict()` | Return raw dictionary |

---

### 5.7 Path Properties

Path values have these properties:

| Property | Description |
|----------|-------------|
| `.path` | Full path string |
| `.base` | Filename with extension |
| `.ext` | Extension (with dot) |
| `.dir` | Directory portion |
| `.absolute` | Whether path is absolute |

| Method | Description |
|--------|-------------|
| `.match(pattern)` | Match route pattern |
| `.toDict()` | Return raw dictionary |

---

### 5.8 URL Properties & Methods

URL values have these properties:

| Property | Description |
|----------|-------------|
| `.scheme` | Protocol (http, https) |
| `.host` | Hostname |
| `.port` | Port number |
| `.path` | Path segments (array) |
| `.query` | Query parameters (dict) |
| `.fragment` | Fragment/hash |
| `.username` | Username if present |
| `.password` | Password if present |

| Method | Description |
|--------|-------------|
| `.origin()` | Get origin (scheme://host:port) |
| `.pathname()` | Get path as string |
| `.search()` | Get query string (?key=val) |
| `.href()` | Get full URL string |
| `.toDict()` | Return raw dictionary |

---

### 5.9 Regex Methods

| Method | Description |
|--------|-------------|
| `.test(str)` | Returns boolean |
| `.replace(str, repl)` | Replace in string |
| `.format(style?)` | Format as string |
| `.toDict()` | Return raw dictionary |

---

### 5.10 Money Properties & Methods

Money values have these properties:

| Property | Description |
|----------|-------------|
| `.amount` | Amount in minor units (cents) |
| `.currency` | Currency code |
| `.scale` | Decimal places |

| Method | Description |
|--------|-------------|
| `.format(locale?)` | Formatted string |
| `.split(n)` | Split into n parts (penny-accurate) |
| `.abs()` | Absolute value |

---

## 6. Builtin Functions

### 6.1 File Loading

| Function | Description |
|----------|-------------|
| `file(path, options?)` | Auto-detect format |
| `JSON(path, options?)` | Load JSON |
| `YAML(path, options?)` | Load YAML |
| `CSV(path, options?)` | Load CSV |
| `MD(path, options?)` | Load Markdown |
| `SVG(path, options?)` | Load SVG |
| `text(path, options?)` | Load as text |
| `lines(path, options?)` | Load as line array |
| `bytes(path, options?)` | Load as bytes |
| `dir(path)` | Directory listing |
| `fileList(pattern)` | Glob file list |

---

### 6.2 Type Conversion

| Function | Description |
|----------|-------------|
| `toInt(value)` | Convert to integer |
| `toFloat(value)` | Convert to float |
| `toNumber(value)` | Convert to int or float |
| `toString(values...)` | Convert to string |
| `toArray(dict)` | Dict to [key, value] pairs |
| `toDict(pairs)` | [key, value] pairs to dict |

---

### 6.3 Output

| Function | Description |
|----------|-------------|
| `print(values...)` | Print without newline |
| `println(values...)` | Print with newline |
| `printf(template, dict)` | Formatted print |
| `log(values...)` | Log to stdout |

---

### 6.4 Introspection

| Function | Description |
|----------|-------------|
| `inspect(value)` | Introspection data as dict |
| `describe(value)` | Human-readable description |
| `repr(value)` | Code representation |
| `builtins(category?)` | List builtin functions |

---

### 6.5 Formatting

| Function | Description |
|----------|-------------|
| `format(value, style?, locale?)` | Format duration or list |
| `tag(name, attrs?, content?)` | Create tag programmatically |
| `markdown(string, options?)` | Parse markdown string |

---

### 6.6 Other Builtins

| Function | Description |
|----------|-------------|
| `time(input, delta?)` | Create datetime |
| `url(string)` | Parse URL |
| `regex(pattern, flags?)` | Create regex |
| `match(path, pattern)` | Match path pattern |
| `money(amount, currency, scale?)` | Create money value |
| `asset(path)` | Convert to web URL |
| `fail(message)` | Throw catchable error |

---

## 7. Standard Library

Import with `@std/` prefix:

```parsley
import @std/math
let {floor, ceil} = import @std/math
```

### 7.1 @std/math

#### Constants

| Name | Value |
|------|-------|
| `PI` | 3.14159... |
| `E` | 2.71828... |
| `TAU` | 6.28318... |

#### Functions

| Function | Description |
|----------|-------------|
| `floor(n)` | Round down |
| `ceil(n)` | Round up |
| `round(n)` | Round to nearest |
| `trunc(n)` | Truncate to integer |
| `abs(n)` | Absolute value |
| `sign(n)` | Sign (-1, 0, 1) |
| `clamp(n, min, max)` | Clamp to range |
| `min(a, b)` / `min(arr)` | Minimum |
| `max(a, b)` / `max(arr)` | Maximum |
| `sum(arr)` | Sum of array |
| `avg(arr)` / `mean(arr)` | Average |
| `median(arr)` | Median |
| `mode(arr)` | Mode |
| `stddev(arr)` | Standard deviation |
| `variance(arr)` | Variance |
| `product(arr)` | Product of array |
| `count(arr)` | Count of array |
| `range(arr)` | Range (max - min) |
| `random()` | Random 0-1 |
| `randomInt(max)` / `randomInt(min, max)` | Random integer |
| `seed(n)` | Seed RNG |
| `sqrt(n)` | Square root |
| `pow(base, exp)` | Power |
| `exp(n)` | e^n |
| `log(n)` | Natural log |
| `log10(n)` | Base-10 log |
| `sin(n)`, `cos(n)`, `tan(n)` | Trigonometry |
| `asin(n)`, `acos(n)`, `atan(n)` | Inverse trig |
| `atan2(y, x)` | 2-argument atan |
| `degrees(rad)` | Radians to degrees |
| `radians(deg)` | Degrees to radians |
| `hypot(a, b)` | Hypotenuse |
| `dist(x1, y1, x2, y2)` | Distance |
| `lerp(a, b, t)` | Linear interpolation |
| `map(v, inMin, inMax, outMin, outMax)` | Map range |

---

### 7.2 @std/valid

#### Type Validators

| Function | Description |
|----------|-------------|
| `string(v)` | Is string? |
| `number(v)` | Is number? |
| `integer(v)` | Is integer? |
| `boolean(v)` | Is boolean? |
| `array(v)` | Is array? |
| `dict(v)` | Is dictionary? |

#### String Validators

| Function | Description |
|----------|-------------|
| `empty(s)` | Is empty/whitespace? |
| `minLen(s, n)` | Minimum length? |
| `maxLen(s, n)` | Maximum length? |
| `length(s, min, max)` | Length in range? |
| `matches(s, regex)` | Matches pattern? |
| `alpha(s)` | Letters only? |
| `alphanumeric(s)` | Letters/numbers? |
| `numeric(s)` | Digits only? |

#### Number Validators

| Function | Description |
|----------|-------------|
| `min(n, min)` | At least min? |
| `max(n, max)` | At most max? |
| `between(n, min, max)` | In range? |
| `positive(n)` | Positive? |
| `negative(n)` | Negative? |

#### Format Validators

| Function | Description |
|----------|-------------|
| `email(s)` | Valid email? |
| `url(s)` | Valid URL? |
| `uuid(s)` | Valid UUID? |
| `phone(s)` | Valid phone? |
| `creditCard(s)` | Valid card (Luhn)? |
| `date(s, locale?)` | Valid date? |
| `time(s)` | Valid time? |
| `postalCode(s, country)` | Valid postal code? |

---

### 7.3 @std/id

| Function | Description |
|----------|-------------|
| `new()` | ULID-like (26 chars, sortable) |
| `uuid()` / `uuidv4()` | UUID v4 (random) |
| `uuidv7()` | UUID v7 (time-sortable) |
| `nanoid(length?)` | NanoID (default 21 chars) |
| `cuid()` | CUID2-like |

---

### 7.4 @std/schema

#### Type Factories

| Function | Description |
|----------|-------------|
| `string(opts?)` | String type |
| `email(opts?)` | Email type |
| `url(opts?)` | URL type |
| `phone(opts?)` | Phone type |
| `integer(opts?)` | Integer type |
| `number(opts?)` | Number type |
| `boolean(opts?)` | Boolean type |
| `enum(values...)` | Enum type |
| `date(opts?)` | Date type |
| `datetime(opts?)` | DateTime type |
| `money(opts?)` | Money type |
| `array(opts?)` | Array type |
| `object(opts?)` | Object type |
| `id(opts?)` | ID type (default ULID) |

#### Schema Operations

```parsley
let UserSchema = define("User", {
    name: string({minLen: 1}),
    email: email(),
    age: integer({min: 0, max: 150})
})

let {valid, errors} = UserSchema.validate(data)
```

---

### 7.5 @std/api

#### Auth Wrappers

| Function | Description |
|----------|-------------|
| `public(fn)` | Mark as public (no auth) |
| `adminOnly(fn)` | Require admin role |
| `roles(roles, fn)` | Require specific roles |
| `auth(fn)` | Require authentication |

#### Error Helpers

| Function | Description |
|----------|-------------|
| `notFound(msg?)` | 404 error |
| `forbidden(msg?)` | 403 error |
| `badRequest(msg?)` | 400 error |
| `unauthorized(msg?)` | 401 error |
| `conflict(msg?)` | 409 error |
| `serverError(msg?)` | 500 error |
| `redirect(url, status?)` | Redirect response |

---

### 7.6 @std/dev

```parsley
import @std/dev

dev.log(value)                     // Log to dev panel
dev.log("label", value)            // Log with label
dev.clearLog()                     // Clear dev log
dev.logPage("/route", value)       // Log to specific page route
dev.setLogRoute("/api")            // Set default log route
dev.clearLogPage("/route")         // Clear log for specific page
```

---

### 7.7 @std/table

```parsley
let {table} = import @std/table

let t = table([
    {name: "Alice", age: 30},
    {name: "Bob", age: 25}
])

t.where(fn(r) { r.age > 25 }).rows
t.orderBy("age", "desc").rows
t.select(["name"]).rows
t.limit(10).rows
t.count()
t.sum("age")
t.avg("age")
t.toHTML()
t.toCSV()
```

---

### 7.8 @std/html

Pre-built accessible HTML components:

| Component | Description |
|-----------|-------------|
| `Page` | Page wrapper |
| `Head` | HTML head |
| `TextField` | Text input |
| `TextareaField` | Textarea |
| `SelectField` | Select dropdown |
| `RadioGroup` | Radio buttons |
| `CheckboxGroup` | Checkboxes |
| `Checkbox` | Single checkbox |
| `Button` | Button |
| `Form` | Form wrapper |
| `Nav` | Navigation |
| `Breadcrumb` | Breadcrumb trail |
| `SkipLink` | Skip to content |
| `Img` | Image |
| `Figure` | Figure with caption |
| `A` | Anchor link |
| `Time` | Time element |
| `DataTable` | Data table |

---

## 8. Tags (HTML/XML)

Tags are first-class values that render to strings:

```parsley
<p>"Hello, World!"</p>             // "<p>Hello, World!</p>"
```

### Attributes

```parsley
// String attribute (literal, no interpolation)
<a href="/about">"About"</a>

// Expression attribute
<div class={className}>content</div>
<button disabled={!isValid}>"Submit"</button>

// Raw string for JavaScript (braces stay literal)
<button onclick='Parts.refresh("id", {key: 1})'>"Click"</button>

// Interpolation in raw strings with @{}
<button onclick='doThing(@{id})'>"Click"</button>
```

### Self-Closing Tags

**Must use `/>` syntax:**

```parsley
<br/>
<img src="photo.jpg"/>
<input type="text" name="email"/>
```

### Content

Text content must be quoted:

```parsley
<h1>"Welcome"</h1>                 // Correct
<h1>`Hello, {name}!`</h1>          // Template string OK
<h1>title</h1>                     // Variable reference OK
```

### Spreading Attributes

```parsley
let attrs = {class: "btn", disabled: true}
<button ...attrs>"Click"</button>
```

### Components

Components are functions:

```parsley
let Card = fn({title, contents}) {
    <div class="card">
        <h3>title</h3>
        <div class="body">contents</div>
    </div>
}

<Card title="Welcome">"Hello, World!"</Card>
```

---

## 9. Error Handling

### Try Expression

Capture errors as values:

```parsley
let {result, error} = try riskyOperation()

if (error != null) {
    log("Error:", error.message)
}
```

### File I/O Error Capture

```parsley
let {data, error} <== JSON(@./config.json)

if (error) {
    log("Failed to load config:", error)
}
```

### User-Defined Errors

```parsley
let validate = fn(x) {
    if (x < 0) {
        fail("must be non-negative")
    }
    x * 2
}

let {result, error} = try validate(-5)
// error = "must be non-negative"
```

### Check Guard

```parsley
let process = fn(data) {
    check data != null else "data required"
    check data.name else "name required"
    // continue processing...
}
```

---

## 10. Comments

```parsley
// Single-line comment only
// No block comments in Parsley
```

---

## 11. Reserved Keywords

```
fn, function, let, for, in, if, else, return, export, import,
try, check, stop, skip, true, false, null, and, or, not, as, via
```
