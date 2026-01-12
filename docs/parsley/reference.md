# Parsley Language Reference

> All syntax verified against `pars` v0.2.0 (January 2026)

## Table of Contents

1. [Literals](#1-literals)
2. [Operators](#2-operators)
3. [Control Flow](#3-control-flow)
4. [Statements](#4-statements)
5. [Type Methods](#5-type-methods)
6. [Builtin Functions](#6-builtin-functions)
7. [Standard Library](#7-standard-library)
8. [Tags (HTML/XML)](#8-tags-htmlxml)
9. [Comments](#9-comments)
10. [Error Handling](#10-error-handling)

**Appendices**:
- [A. Type Summary](#appendix-a-type-summary)
- [B. Method Reference](#appendix-b-method-reference)

---

## 1. Literals

### 1.1 Numbers

#### Integers

```parsley
42
-15
0
```

#### Floats

```parsley
3.14159
-2.718
0.5
```

---

### 1.2 Strings

#### Double-Quoted Strings (`"..."`)

Standard strings with escape sequences. **No interpolation**.

```parsley
"Hello, World!"
"Line1\nLine2"
"Tab\there"
"Quote: \"hi\""
"Backslash: \\"
```

| Escape | Meaning |
|--------|---------|
| `\n` | Newline |
| `\t` | Tab |
| `\r` | Carriage return |
| `\\` | Backslash |
| `\"` | Double quote |

#### Template Strings (`` `...` ``)

Interpolated strings using `{expression}` syntax.

```parsley
let name = "Alice"
`Hello, {name}!`        // "Hello, Alice!"
`2 + 2 = {2 + 2}`       // "2 + 2 = 4"
```

#### Raw Strings (`'...'`)

Backslashes are literal. Interpolation only with `@{expression}`.

```parsley
'C:\Users\name'                 // Backslashes literal
'regex: \d+\.\d+'               // No escaping needed
let id = 42
'id = @{id}'                    // "id = 42"
```

---

### 1.3 Booleans and Null

```parsley
true
false
null
```

**Truthy**: Everything except `false`, `null`, `0`, `0.0`, `""`, `[]`

**Falsy**: `false`, `null`, `0`, `0.0`, `""` (empty string), `[]` (empty array)

---

### 1.4 Arrays

```parsley
[1, 2, 3]
let empty = []
```

#### Indexing

```parsley
let arr = [10, 20, 30, 40, 50]
arr[0]                          // 10 (first element)
arr[-1]                         // 50 (last element)
arr[1:3]                        // [20, 30] (slice)
arr[:2]                         // [10, 20] (first 2)
arr[2:]                         // [30, 40, 50] (from index 2)
arr[?99]                        // null (optional, no error)
```

#### Destructuring

```parsley
let [a, b, c] = [1, 2, 3]       // a=1, b=2, c=3
let [first, ...rest] = [1, 2, 3, 4]  // first=1, rest=[2,3,4]
```

---

### 1.5 Dictionaries

```parsley
{name: "Alice", age: 30}
let emptyDict = {}
```

#### Access

```parsley
let person = {name: "Bob", age: 25}
person.name                     // "Bob"
person["age"]                   // 25
person.missing                  // null (no error)
```

#### Destructuring

```parsley
let {name, age} = person        // Extract fields
```

---

### 1.6 Functions

```parsley
let double = fn(x) { x * 2 }
double(5)                       // 10

let add = fn(a, b) { a + b }
add(3, 4)                       // 7

let constant = fn() { 42 }
constant()                      // 42
```

**Implicit return**: The last expression is returned automatically.

```parsley
let complex = fn(x) {
    let y = x * 2
    y + 1                       // Returns this
}
complex(10)                     // 21
```

---

### 1.7 DateTime Literals

All datetime literals start with `@`:

```parsley
@2024-12-25                     // Date only
@2024-12-25T14:30:00            // DateTime
@14:30                          // Time only
@14:30:45                       // Time with seconds
```

#### Special Values

```parsley
@now                            // Current datetime
@today                          // Current date
```

#### Interpolated DateTime

```parsley
let month = 12
let day = 25
@(2024-{month}-{day})           // Dynamic construction
```

---

### 1.8 Duration Literals

```parsley
@1d                             // 1 day
@2h30m                          // 2 hours 30 minutes
@1w                             // 1 week
@1y6mo                          // 1 year 6 months
@-1d                            // Negative: 1 day ago
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

### 1.9 Money Literals

#### Symbol Format

```parsley
$12.34                          // USD
$99.99
```

#### CODE# Format

```parsley
EUR#50.00                       // Euro
GBP#25.00                       // British Pound
```

---

### 1.10 Regex Literals

Regex literals must be assigned to a variable or used in an expression context.

```parsley
let r = /hello/
let digits = /\d+/
let caseInsensitive = /pattern/i
```

| Flag | Meaning |
|------|---------|
| `i` | Case insensitive |
| `m` | Multiline |
| `s` | Dotall (`.` matches newline) |
| `g` | Global (all matches) |

---

### 1.11 Path Literals

```parsley
@./relative/path                // Relative to current file
@~/from/root                    // Relative to project root
```

#### Interpolated Paths

```parsley
let file = "config"
@(./data/{file}.json)           // ./data/config.json
```

---

### 1.12 URL Literals

```parsley
@https://example.com
@http://localhost:3000
```

#### Interpolated URLs

```parsley
let id = 123
@(https://api.example.com/users/{id})
```

---

### 1.13 Standard Library Paths

```parsley
@std/math
@std/valid
@std/id
```

---

## 2. Operators

### 2.1 Arithmetic

| Operator | Description | Example |
|----------|-------------|---------|
| `+` | Addition | `5 + 3` → `8` |
| `-` | Subtraction | `10 - 4` → `6` |
| `*` | Multiplication | `6 * 7` → `42` |
| `/` | Division | `20 / 4` → `5` |
| `%` | Modulo | `17 % 5` → `2` |
| `-` | Negation (prefix) | `-42` |

#### String Operations

```parsley
"Hello" + " " + "World"         // Concatenation
"ab" * 3                        // "ababab" (repetition)
```

#### Array Operations

```parsley
let repeated = [1, 2] * 2       // [1, 2, 1, 2]
let chunked = [1,2,3,4,5,6] / 2 // [[1,2], [3,4], [5,6]]
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

---

### 2.3 Logical

| Operator | Description |
|----------|-------------|
| `&&` | Logical AND (short-circuit) |
| `\|\|` | Logical OR (short-circuit) |
| `!` | Logical NOT |
| `and` | Keyword alias for `&&` |
| `or` | Keyword alias for `\|\|` |

```parsley
true && true                    // true
false || true                   // true
let notResult = !false          // true
true and true                   // true
false or true                   // true
```

#### Set Operations on Collections

```parsley
([1,2,3] && [2,3,4]).toJSON()   // [2,3] (intersection)
let union = [1,2] | [2,3]       // [1,2,3] (union)
```

---

### 2.4 Membership

| Operator | Description |
|----------|-------------|
| `in` | Membership test |
| `not in` | Negated membership |

```parsley
2 in [1, 2, 3]                  // true
"name" in {name: "Sam"}         // true (key exists)
"ell" in "hello"                // true (substring)
"x" not in [1, 2, 3]            // true
```

---

### 2.5 Pattern Matching

| Operator | Description |
|----------|-------------|
| `~` | Regex match (returns match or null) |
| `!~` | Regex not match (returns boolean) |

```parsley
"hello123" ~ /\d+/              // "123"
"hello" !~ /\d+/                // true
```

---

### 2.6 Range

```parsley
let range = 1..5                // [1, 2, 3, 4, 5]
```

---

### 2.7 Concatenation

```parsley
let concat = [1, 2] ++ [3, 4]   // [1, 2, 3, 4]
let merged = {a: 1} ++ {b: 2}   // {a: 1, b: 2}
```

---

### 2.8 Null Coalescing

```parsley
null ?? "default"               // "default"
"value" ?? "default"            // "value"
```

#### Optional Index Access

Use `[?index]` syntax (question mark inside brackets):

```parsley
let arr = [1, 2, 3]
arr[?99]                        // null (no error)
arr[?0]                         // 1
```

---

### 2.9 DateTime Arithmetic

```parsley
@now + @1d                      // Tomorrow
@now - @1w                      // One week ago
@2024-12-25 - @2024-12-20       // 5 days (duration)
@2024-12-25 && @14:30           // Combine date + time
```

---

### 2.10 Precedence Table (Lowest to Highest)

| Level | Operators |
|-------|-----------|
| 1 | `??`, `\|\|`, `or` |
| 2 | `&&`, `and` |
| 3 | `==`, `!=`, `~`, `!~`, `in`, `not in` |
| 4 | `<`, `>`, `<=`, `>=` |
| 5 | `+`, `-`, `..` |
| 6 | `++` |
| 7 | `*`, `/`, `%` |
| 8 | `-`, `!` (prefix) |
| 9 | `.`, `[]`, `()` (access/call) |

---

## 3. Control Flow

### 3.1 If Expression

If is an **expression** that returns a value.

```parsley
let age = 20
if (age >= 18) { "adult" } else { "minor" }
```

Parentheses are optional:

```parsley
if age >= 18 { "adult" }
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

For is an **expression** that returns an array (like `map`).

```parsley
let nums = [1, 2, 3, 4, 5]
for (n in nums) { n * 2 }       // [2, 4, 6, 8, 10]
```

#### With Index

```parsley
for (i, n in nums) { `{i}: {n}` }
```

#### With Range

```parsley
for (x in 1..3) { x * x }       // [1, 4, 9]
```

---

### 3.3 Loop Control

| Keyword | Description |
|---------|-------------|
| `stop` | Exit loop (like `break`) |
| `skip` | Skip iteration (like `continue`) |

```parsley
let firstThree = for (x in 1..10) {
    if (x > 3) { stop }
    x
}
// [1, 2, 3]

let evens = for (x in 1..6) {
    if (x % 2 != 0) { skip }
    x
}
// [2, 4, 6]
```

---

### 3.4 Try Expression

Capture errors as values.

```parsley
let safeDivide = fn(a, b) {
    check b != 0 else fail("division by zero")
    a / b
}
let result = try safeDivide(10, 0)
// {result: null, error: "division by zero"}
```

---

### 3.5 Check Guard

Early return if condition fails.

```parsley
let validate = fn(x) {
    check x > 0 else "must be positive"
    x * 2
}
validate(5)                     // 10
validate(-1)                    // "must be positive"
```

---

## 4. Statements

### 4.1 Let Binding

```parsley
let x = 5
let name = "Alice"
```

#### Destructuring

```parsley
let arr = [1, 2, 3]
let [a, b, c] = arr

let person = {name: "Bob", age: 25}
let {name, age} = person

let [first, ...rest] = [1, 2, 3, 4]
```

---

### 4.2 Assignment

```parsley
let y = 10
y = 20                          // Reassign

let obj = {a: 1}
obj.b = 2                       // Property assignment

let nums = [1, 2, 3]
nums[0] = 99                    // Index assignment
```

---

### 4.3 Return

```parsley
let multiply = fn(a, b) {
    return a * b
}
```

---

### 4.4 Export

```parsley
export let greeting = "Hello"
export PI = 3.14159
```

---

### 4.5 Import

```parsley
import @std/math
import @std/math as M
let {floor, ceil} = import @std/math
```

---

## 5. Type Methods

Methods are called on values using dot notation: `value.method(args)`.

**Return Value Convention**: Most methods return a new value and do not modify the original. Exception: `delete()` on dictionaries mutates in place.

---

### 5.1 String Methods

#### Case Conversion

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.toUpper()` | none | `string` | Convert to uppercase |
| `.toLower()` | none | `string` | Convert to lowercase |
| `.toTitle()` | none | `string` | Capitalize first letter of each word |

#### Whitespace Handling

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.trim()` | none | `string` | Remove leading/trailing whitespace |
| `.collapse()` | none | `string` | Collapse whitespace to single spaces |
| `.normalizeSpace()` | none | `string` | Collapse and trim (combines both) |
| `.stripSpace()` | none | `string` | Remove all whitespace |
| `.outdent()` | none | `string` | Remove common leading indent from all lines |
| `.indent(n)` | `n: integer` | `string` | Add n spaces to start of each non-blank line |

#### Search & Transform

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.length()` | none | `integer` | Character count |
| `.split(delim)` | `delim: string` | `array` | Split by delimiter |
| `.replace(old, new)` | `old, new: string` | `string` | Replace all occurrences |
| `.includes(substr)` | `substr: string` | `boolean` | Check if contains substring |
| `.digits()` | none | `string` | Extract only digits |
| `.slug()` | none | `string` | Convert to URL-safe slug |

#### HTML Processing

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.htmlEncode()` | none | `string` | Escape `<`, `>`, `&`, `"` |
| `.htmlDecode()` | none | `string` | Decode HTML entities |
| `.stripHtml()` | none | `string` | Remove all HTML tags |
| `.paragraphs()` | none | `string` | Convert blank-line-separated text to `<p>` tags |
| `.highlight(pattern, tag?)` | `pattern: string\|regex`, `tag?: string` | `string` | Wrap matches in HTML tag (default: `<mark>`) |

#### URL Encoding

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.urlEncode()` | none | `string` | URL encode (spaces → `+`) |
| `.urlDecode()` | none | `string` | Decode URL-encoded string |
| `.urlPathEncode()` | none | `string` | Encode path segment (`/` → `%2F`) |
| `.urlQueryEncode()` | none | `string` | Encode query value (`&`, `=` encoded) |

#### Parsing

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.parseJSON()` | none | `any` | Parse string as JSON |
| `.parseCSV(hasHeader?)` | `hasHeader?: boolean` (default: `true`) | `table` | Parse string as CSV |

#### Templating

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.render(dict?)` | `dict?: dictionary` | `string` | Interpolate `@{key}` placeholders with dict values |

```parsley
"  Hello, World!  ".trim()      // "Hello, World!"
"hello world".toTitle()         // "Hello World"
"hello world".split(" ")        // ["hello", "world"]
"hello".replace("l", "L")       // "heLLo"
"<div>test</div>".htmlEncode()  // "&lt;div&gt;test&lt;/div&gt;"
"abc123def456".digits()         // "123456"
"Hello World!".slug()           // "hello-world"
"name = @{name}".render({name: "Alice"})  // "name = Alice"
```

---

### 5.2 Array Methods

#### Inspection

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.length()` | none | `integer` | Element count |
| `.has(item)` | `item: any` | `boolean` | Check if array contains item |
| `.hasAny(arr)` | `arr: array` | `boolean` | Check if any item from arr is in array |
| `.hasAll(arr)` | `arr: array` | `boolean` | Check if all items from arr are in array |

#### Ordering

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.reverse()` | none | `array` | New array in reverse order |
| `.sort()` | none | `array` | New array sorted (natural order) |
| `.sortBy(fn)` | `fn: function` | `array` | Sort by key function result |
| `.shuffle()` | none | `array` | New array with elements in random order |

#### Transformation

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.map(fn)` | `fn: function(elem)` | `array` | Transform each element; `null` results excluded |
| `.filter(fn)` | `fn: function(elem)` | `array` | Keep elements where fn returns truthy |
| `.reduce(fn, init)` | `fn: function(acc, elem)`, `init: any` | `any` | Reduce to single value |
| `.insert(index, val)` | `index: integer`, `val: any` | `array` | New array with val inserted at index |

#### Random Selection

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.pick()` | none | `any` | Single random element (or `null` if empty) |
| `.pick(n)` | `n: integer` | `array` | n random elements **with replacement** |
| `.take(n)` | `n: integer` | `array` | n unique random elements **without replacement** |

**Note**: `.pick(n)` can return duplicates; `.take(n)` cannot. `.take(n)` fails if n > array length.

#### Output

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.join(sep?)` | `sep?: string` (default: `""`) | `string` | Concatenate elements with separator |
| `.format(style?, locale?)` | `style?: string`, `locale?: string` | `string` | Format as prose list |
| `.toJSON()` | none | `string` | Convert to JSON string |
| `.toCSV(hasHeader?)` | `hasHeader?: boolean` (default: `true`) | `string` | Convert to CSV string |

**Format styles**: `"and"` (default), `"or"`, or any string like `"unit"`.

```parsley
let arr = [3, 1, 4, 1, 5]
arr.sort()                      // [1, 1, 3, 4, 5]
arr.map(fn(x) { x * 2 })        // [6, 2, 8, 2, 10]
arr.filter(fn(x) { x > 2 })     // [3, 4, 5]
arr.reduce(fn(acc, x) { acc + x }, 0)  // 14

let items = ["apple", "banana", "cherry"]
items.format()                  // "apple, banana, and cherry"
items.format("or")              // "apple, banana, or cherry"
items.join(", ")                // "apple, banana, cherry"

[1, 2, 3].has(2)                // true
[1, 2, 3].hasAny([2, 5, 6])     // true
[1, 2, 3].hasAll([1, 3])        // true
```

---

### 5.3 Dictionary Methods

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.keys()` | none | `array` | Array of all keys (preserves insertion order) |
| `.values()` | none | `array` | Array of all values |
| `.entries()` | none | `array` | Array of `{key, value}` dictionaries |
| `.entries(k, v)` | `k, v: string` | `array` | Array of `{k, v}` with custom names |
| `.has(key)` | `key: string` | `boolean` | Check if key exists |
| `.delete(key)` | `key: string` | `null` | Remove key (**mutates in place**) |
| `.insertAfter(after, key, val)` | `after, key: string`, `val: any` | `dictionary` | New dict with key inserted after `after` |
| `.insertBefore(before, key, val)` | `before, key: string`, `val: any` | `dictionary` | New dict with key inserted before `before` |
| `.render(template)` | `template: string` | `string` | Render template with `@{key}` placeholders |
| `.toJSON()` | none | `string` | Convert to JSON string |

**Note**: `.delete()` is the only method that mutates the original. All others return new dictionaries.

```parsley
let d = {name: "Alice", age: 30}
d.keys()                        // ["name", "age"]
d.values()                      // ["Alice", 30]
d.has("name")                   // true
d.entries()                     // [{key: "name", value: "Alice"}, {key: "age", value: 30}]
d.entries("k", "v")             // [{k: "name", v: "Alice"}, {k: "age", v: 30}]

let tmpl = "Hello, @{name}!"
d.render(tmpl)                  // "Hello, Alice!"

// Ordered insertion
let person = {first: "Alice", last: "Smith"}
person.insertAfter("first", "middle", "Jane")
// {first: "Alice", middle: "Jane", last: "Smith"}
```

---

### 5.4 Number Methods

Both `integer` and `float` types share these methods:

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.format(locale?)` | `locale?: string` (default: `"en-US"`) | `string` | Locale-formatted number |
| `.currency(code, locale?)` | `code: string`, `locale?: string` | `string` | Currency format |
| `.percent(locale?)` | `locale?: string` | `string` | Percentage format |
| `.humanize(locale?)` | `locale?: string` | `string` | Compact format (1.2K, 3.4M) |

```parsley
let n = 1234567
n.format()                      // "1,234,567"
n.format("de-DE")               // "1.234.567"
n.currency("USD")               // "$1,234,567.00"
n.currency("EUR", "de-DE")      // "1.234.567,00 €"
n.humanize()                    // "1.2M"

let pct = 0.1234
pct.percent()                   // "12%"
```

---

### 5.5 DateTime Properties & Methods

DateTime values are dictionaries with special properties and methods. They are created from datetime literals (`@2024-12-25`) or the `time()` function.

#### Properties

| Property | Type | Description |
|----------|------|-------------|
| `.year` | `integer` | Year (e.g., 2024) |
| `.month` | `integer` | Month (1-12) |
| `.day` | `integer` | Day of month (1-31) |
| `.hour` | `integer` | Hour (0-23) |
| `.minute` | `integer` | Minute (0-59) |
| `.second` | `integer` | Second (0-59) |
| `.weekday` | `string` | Day name ("Monday", "Tuesday", etc.) |
| `.dayOfYear` | `integer` | Day number within year (1-366) |
| `.week` | `integer` | ISO week number (1-53) |
| `.unix` | `integer` | Unix timestamp (seconds since 1970-01-01) |
| `.iso` | `string` | ISO 8601 datetime string |
| `.kind` | `string` | Type: `"date"`, `"datetime"`, `"time"`, or `"time_seconds"` |
| `.date` | `string` | Date portion (`"YYYY-MM-DD"`) |
| `.time` | `string` | Time portion (`"HH:MM"` or `"HH:MM:SS"`) |

#### Methods

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.format(style?, locale?)` | `style?: string`, `locale?: string` | `string` | Format datetime |
| `.toDict()` | none | `dictionary` | Get raw dictionary for debugging |

**Format styles**: `"short"`, `"medium"`, `"long"` (default), `"full"`, or a custom Go format string.

| Style | Example Output |
|-------|----------------|
| `"short"` | `"12/25/24"` |
| `"medium"` | `"Dec 25, 2024"` |
| `"long"` | `"December 25, 2024"` |
| `"full"` | `"Wednesday, December 25, 2024"` |

```parsley
let dt = @2024-12-25T14:30:00
dt.year                         // 2024
dt.weekday                      // "Wednesday"
dt.kind                         // "datetime"
dt.iso                          // "2024-12-25T14:30:00"
dt.format()                     // "December 25, 2024"
dt.format("short")              // "12/25/24"
dt.format("full", "de-DE")      // "Mittwoch, 25. Dezember 2024"
```

---

### 5.6 Duration Properties & Methods

Duration values represent time spans and are created from duration literals (`@1d`, `@2h30m`).

#### Properties

| Property | Type | Description |
|----------|------|-------------|
| `.months` | `integer` | Months component (years stored as 12×years) |
| `.seconds` | `integer` | Seconds component |
| `.totalSeconds` | `integer` | Total as seconds (only valid when months = 0) |
| `.days` | `integer` | Total days (`null` if months > 0) |
| `.hours` | `integer` | Total hours (`null` if months > 0) |
| `.minutes` | `integer` | Total minutes (`null` if months > 0) |

**Note**: Durations with month/year components cannot be converted to exact seconds/days because month lengths vary.

#### Methods

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.format(locale?)` | `locale?: string` | `string` | Human-readable relative time |
| `.toDict()` | none | `dictionary` | Get raw dictionary for debugging |

```parsley
let dur = @2h30m
dur.hours                       // 2
dur.minutes                     // 150 (total)
dur.totalSeconds                // 9000
dur.format()                    // "in 2 hours"

let longDur = @1y6mo
longDur.months                  // 18
longDur.totalSeconds            // null (can't compute exactly)
```

---

### 5.7 Path Properties & Methods

Path values represent filesystem paths and are created from path literals (`@./file.txt`, `@~/config.json`).

#### Properties

| Property | Type | Description |
|----------|------|-------------|
| `.absolute` | `boolean` | Whether path is absolute |
| `.segments` | `array` | Path segments as array of strings |
| `.extension` | `string` | File extension (without dot) |
| `.filename` | `string` | Last segment (file or directory name) |
| `.parent` | `path` | Parent directory path |

#### Methods

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.isAbsolute()` | none | `boolean` | Check if absolute path |
| `.isRelative()` | none | `boolean` | Check if relative path |
| `.match(pattern)` | `pattern: string` | `dictionary\|null` | Match against route pattern (returns captures or `null`) |
| `.toURL(prefix)` | `prefix: string` | `string` | Convert to URL with prefix |
| `.public()` | none | `string` | Get public URL (for web serving) |
| `.toDict()` | none | `dictionary` | Get raw dictionary for debugging |

**Pattern syntax**: Use `:param` for single segments, `*splat` for multiple.

```parsley
let p = @./users/123/profile.json
p.filename                      // "profile.json"
p.extension                     // "json"
p.segments                      // [".", "users", "123", "profile.json"]
p.parent.filename               // "123"

p.match("/users/:id/:file")     // {id: "123", file: "profile.json"}
p.match("/products/:id")        // null (no match)
p.toURL("https://example.com")  // "https://example.com/users/123/profile.json"
```

---

### 5.8 URL Properties & Methods

URL values represent web addresses and are created from URL literals (`@https://example.com`) or the `url()` function.

#### Properties

| Property | Type | Description |
|----------|------|-------------|
| `.scheme` | `string` | URL scheme (`"http"`, `"https"`, etc.) |
| `.host` | `string` | Hostname |
| `.port` | `integer` | Port number (0 if not specified) |
| `.path` | `array` | Path segments as array |
| `.query` | `dictionary` | Query parameters as dictionary |
| `.fragment` | `string` | Fragment identifier (after `#`) |

#### Methods

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.origin()` | none | `string` | Get `scheme://host:port` |
| `.pathname()` | none | `string` | Get path as string (`"/path/to/resource"`) |
| `.href()` | none | `string` | Get full URL string |
| `.search()` | none | `string` | Get query string (`"?key=value"` or `""`) |
| `.toDict()` | none | `dictionary` | Get raw dictionary for debugging |

```parsley
let u = @https://example.com:8080/api/users?page=1&limit=10#section
u.scheme                        // "https"
u.host                          // "example.com"
u.port                          // 8080
u.query                         // {page: "1", limit: "10"}
u.fragment                      // "section"
u.origin()                      // "https://example.com:8080"
u.pathname()                    // "/api/users"
u.href()                        // full URL string
```

---

### 5.9 Regex Properties & Methods

Regex values are created from regex literals (`/pattern/flags`) or the `regex()` function.

#### Properties

| Property | Type | Description |
|----------|------|-------------|
| `.pattern` | `string` | Regular expression pattern |
| `.flags` | `string` | Regex flags (`"i"`, `"m"`, `"s"`, `"g"`) |

#### Methods

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.test(str)` | `str: string` | `boolean` | Check if pattern matches anywhere in string |
| `.replace(str, repl)` | `str: string`, `repl: string\|function` | `string` | Replace matches in string |
| `.format(style?)` | `style?: string` | `string` | Format regex for display |
| `.toDict()` | none | `dictionary` | Get raw dictionary for debugging |

**Format styles**: `"pattern"` (pattern only), `"literal"` (default, with `/` delimiters), `"verbose"` (pattern and flags separately).

**Replacement**: When `repl` is a function, it receives the match and should return the replacement string.

```parsley
let r = /\d+/g
r.pattern                       // "\\d+"
r.flags                         // "g"
r.test("hello123")              // true
r.test("hello")                 // false
r.replace("abc123def456", "X")  // "abcXdefX"

// Function replacement
let upper = /[a-z]+/g
upper.replace("hello WORLD", fn(m) { m.toUpper() })  // "HELLO WORLD"
```

---

### 5.10 Money Properties & Methods

Money values represent currency amounts with arbitrary precision. They are created from money literals (`$12.34`, `EUR#50.00`) or the `money()` function.

#### Properties

| Property | Type | Description |
|----------|------|-------------|
| `.amount` | `integer` | Amount in smallest unit (e.g., cents for USD) |
| `.currency` | `string` | ISO 4217 currency code (e.g., `"USD"`, `"EUR"`) |
| `.scale` | `integer` | Number of decimal places for this currency |

#### Methods

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.format(locale?)` | `locale?: string` | `string` | Format with currency symbol |
| `.abs()` | none | `money` | Absolute value |
| `.negate()` | none | `money` | Negate amount |
| `.split(n)` | `n: integer` | `array` | Split into n parts (handles rounding) |
| `.toDict()` | none | `dictionary` | Get raw dictionary for debugging |

**Arithmetic**: Money supports `+`, `-` (same currency only), and `*`, `/` by numbers.

**Splitting**: `.split(n)` distributes rounding errors across parts so the total is exact.

```parsley
let m = $100.00
m.amount                        // 10000 (cents)
m.currency                      // "USD"
m.scale                         // 2
m.format()                      // "$100.00"
m.format("de-DE")               // "100,00 $"

$100.00 + $50.00                // $150.00
$100.00 * 3                     // $300.00
$100.00.split(3)                // [$33.34, $33.33, $33.33]
(-$50.00).abs()                 // $50.00
```

---

### 5.11 Table Properties & Methods

Table values represent structured tabular data with rows and columns. They are created from CSV files using the `CSV()` function or by parsing CSV strings with `.parseCSV()`.

#### Properties

| Property | Type | Description |
|----------|------|-------------|
| `.row` | `dictionary\|null` | First row as dictionary, or `null` if empty |
| `.rows` | `array` | All rows as array of dictionaries |
| `.columns` | `array` | Column names as array of strings |

#### Query Methods

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.where(fn)` | `fn: function(row)` | `table` | Filter rows by predicate |
| `.orderBy(col, dir?)` | `col: string`, `dir?: "asc"\|"desc"` | `table` | Sort by column |
| `.select(cols...)` | `cols: string...` | `table` | Select specific columns |
| `.limit(n, offset?)` | `n: integer`, `offset?: integer` | `table` | Limit rows with optional offset |

#### Aggregate Methods

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.count()` | none | `integer` | Number of rows |
| `.sum(col)` | `col: string` | `number` | Sum of column values |
| `.avg(col)` | `col: string` | `float` | Average of column values |
| `.min(col)` | `col: string` | `any` | Minimum column value |
| `.max(col)` | `col: string` | `any` | Maximum column value |
| `.column(col)` | `col: string` | `array` | All values from column as array |

#### Mutation Methods

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.appendRow(row)` | `row: dictionary` | `table` | Add row at end |
| `.insertRowAt(i, row)` | `i: integer`, `row: dictionary` | `table` | Insert row at index |
| `.appendCol(name, vals)` | `name: string`, `vals: array` | `table` | Add column at end |
| `.insertColAfter(after, name, vals)` | `after, name: string`, `vals: array` | `table` | Insert column after another |
| `.insertColBefore(before, name, vals)` | `before, name: string`, `vals: array` | `table` | Insert column before another |

#### Inspection Methods

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.rowCount()` | none | `integer` | Number of rows |
| `.columnCount()` | none | `integer` | Number of columns |

#### Output Methods

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.toHTML(footer?)` | `footer?: string\|dictionary` | `string` | Convert to HTML `<table>` |
| `.toCSV()` | none | `string` | Convert to CSV string |
| `.toMarkdown()` | none | `string` | Convert to Markdown table |
| `.toJSON()` | none | `string` | Convert to JSON array |

```parsley
let data <== CSV(@./sales.csv)

// Query operations
let filtered = data.where(fn(r) { r.amount > 100 })
let sorted = data.orderBy("date", "desc")
let projected = data.select("name", "total")
let page = data.limit(10, 20)       // 10 rows, skip 20

// Aggregates
data.count()                        // 150
data.sum("amount")                  // 12500.00
data.avg("price")                   // 49.99
data.column("name")                 // ["Alice", "Bob", ...]

// Chaining
let result = data
    .where(fn(r) { r.region == "North" })
    .orderBy("sales", "desc")
    .limit(5)

// Output
<div>result.toHTML()</div>
```

---

## 6. Builtin Functions

Builtin functions are globally available without import.

### 6.1 Type Conversion

| Function | Arguments | Returns | Description |
|----------|-----------|---------|-------------|
| `toInt(val)` | `val: any` | `integer` | Convert to integer |
| `toFloat(val)` | `val: any` | `float` | Convert to float |
| `toNumber(val)` | `val: any` | `integer\|float` | Convert to int or float (preserves type) |
| `toString(val)` | `val: any` | `string` | Convert to string |
| `toArray(dict)` | `dict: dictionary` | `array` | Convert to `[[key, value], ...]` pairs |
| `toDict(pairs)` | `pairs: array` | `dictionary` | Convert `[[key, value], ...]` to dict |

**Conversion rules**:
- `toInt("42")` → `42`
- `toInt(3.7)` → `3` (truncates)
- `toFloat("3.14")` → `3.14`
- `toNumber("42")` → `42` (integer)
- `toNumber("3.14")` → `3.14` (float)

```parsley
toInt("42")                     // 42
toFloat("3.14")                 // 3.14
toString(42)                    // "42"
toArray({a: 1, b: 2})           // [["a", 1], ["b", 2]]
toDict([["x", 10], ["y", 20]])  // {x: 10, y: 20}
```

---

### 6.2 Output

| Function | Arguments | Returns | Description |
|----------|-----------|---------|-------------|
| `print(vals...)` | `vals: any...` | `null` | Print without newline |
| `println(vals...)` | `vals: any...` | `null` | Print with newline |
| `printf(fmt, vals...)` | `fmt: string`, `vals: any...` | `null` | Print with format string |
| `log(vals...)` | `vals: any...` | `null` | Log values (first string unquoted) |
| `logLine(vals...)` | `vals: any...` | `null` | Log with newline |
| `toDebug(val)` | `val: any` | `string` | Convert value to debug string |

**Note**: These write to stdout. In web context, output typically doesn't appear to users.

**`log()` behavior**: First argument is displayed without quotes if it's a string (as a label), subsequent values use debug format.

```parsley
log("user", currentUser)        // "user {name: 'Alice', ...}"
log(42, "hello")                // "42, hello"
toDebug({a: 1, b: 2})           // "{a: 1, b: 2}"
```

---

### 6.3 Introspection

| Function | Arguments | Returns | Description |
|----------|-----------|---------|-------------|
| `describe(val)` | `val: any` | `string` | Human-readable description with type and methods |
| `repr(val)` | `val: any` | `string` | Code representation (can often be parsed back) |
| `inspect(val)` | `val: any` | `dictionary` | Detailed introspection data |
| `builtins(category?)` | `category?: string` | `dictionary` | List all builtin functions |

```parsley
describe(42)                    // "integer: 42\nMethods: format, currency, percent, humanize"
repr("hello")                   // "\"hello\""
repr([1, 2, 3])                 // "[1, 2, 3]"

let info = inspect(@now)
info.type                       // "datetime"
info.properties                 // ["year", "month", "day", ...]
info.methods                    // ["format", "toDict", ...]
```

---

### 6.4 Formatting

| Function | Arguments | Returns | Description |
|----------|-----------|---------|-------------|
| `format(arr, style?)` | `arr: array`, `style?: string` | `string` | Format array as prose list |
| `tag(name, attrs?, content?)` | `name: string`, `attrs?: dictionary`, `content?: any` | `tag` | Create HTML tag programmatically |

**Format styles**: `"and"` (default), `"or"`, or any conjunction string.

```parsley
format(["a", "b", "c"])         // "a, b, and c"
format(["a", "b", "c"], "or")   // "a, b, or c"
format(["a", "b"], "unit")      // "a and b" (no Oxford comma for 2 items)

tag("div", {class: "box"}, "Hello")  // <div class="box">Hello</div>
```

---

### 6.5 Regex

| Function | Arguments | Returns | Description |
|----------|-----------|---------|-------------|
| `regex(pattern, flags?)` | `pattern: string`, `flags?: string` | `regex` | Create regex from string |
| `match(str, pattern, flags?)` | `str: string`, `pattern: string`, `flags?: string` | `string\|null` | Find first match |

**Flags**: `i` (case-insensitive), `m` (multiline), `s` (dotall), `g` (global).

```parsley
regex("\\d+", "g")              // /\d+/g
match("hello123world", "\\d+")  // "123"
match("hello", "\\d+")          // null
```

---

### 6.6 Money

| Function | Arguments | Returns | Description |
|----------|-----------|---------|-------------|
| `money(amount, currency?, scale?)` | `amount: number`, `currency?: string`, `scale?: integer` | `money` | Create money value |

**Defaults**: `currency` defaults to `"USD"`, `scale` defaults to currency's standard (usually 2).

```parsley
money(1234, "USD")              // $12.34 (amount in cents)
money(5000, "JPY", 0)           // ¥5000 (JPY has no decimals)
money(1000, "EUR")              // €10.00
```

---

### 6.7 Control Flow

| Function | Arguments | Returns | Description |
|----------|-----------|---------|-------------|
| `fail(message)` | `message: string` | never | Throw a catchable error |

Use with `try` to handle errors gracefully:

```parsley
let safeDivide = fn(a, b) {
    check b != 0 else fail("division by zero")
    a / b
}

let result = try safeDivide(10, 0)
// {result: null, error: "division by zero"}

let good = try safeDivide(10, 2)
// {result: 5, error: null}
```

---

### 6.8 DateTime

| Function | Arguments | Returns | Description |
|----------|-----------|---------|-------------|
| `time(input, delta?)` | `input: string\|integer\|dict`, `delta?: dict` | `datetime` | Create datetime from various inputs |

**Input types**:
- `string` — ISO 8601 date/datetime string (`"2024-12-25"`, `"2024-12-25T14:30:00"`)
- `integer` — Unix timestamp (seconds since 1970-01-01)
- `dictionary` — Date components (`{year: 2024, month: 12, day: 25, ...}`)

**Delta**: Optional dictionary to add/subtract time (`{days: 1}`, `{hours: -2}`).

```parsley
time("2024-12-25")              // DateTime from string
time(1735142400)                // DateTime from Unix timestamp
time({year: 2024, month: 12, day: 25})  // DateTime from components
time("2024-12-25", {days: 1})   // Add 1 day
```

**Note**: Prefer datetime literals (`@2024-12-25`, `@now`) for static dates. Use `time()` when parsing dynamic input.

---

### 6.9 URL

| Function | Arguments | Returns | Description |
|----------|-----------|---------|-------------|
| `url(urlString)` | `urlString: string` | `url` | Parse URL string into components |

```parsley
let u = url("https://example.com:8080/api?key=value")
u.scheme                        // "https"
u.host                          // "example.com"
u.port                          // 8080
u.query.key                     // "value"
```

---

### 6.10 File Operations

These functions create file handles for reading and writing.

| Function | Arguments | Returns | Description |
|----------|-----------|---------|-------------|
| `JSON(source, opts?)` | `source: path\|url`, `opts?: dict` | `file` | JSON file handle |
| `YAML(source, opts?)` | `source: path\|url`, `opts?: dict` | `file` | YAML file handle |
| `CSV(source, opts?)` | `source: path\|url`, `opts?: dict` | `file` | CSV file handle (returns table) |
| `text(source, opts?)` | `source: path\|url`, `opts?: dict` | `file` | Plain text file handle |
| `lines(source, opts?)` | `source: path\|url`, `opts?: dict` | `file` | Lines file handle (returns array) |
| `bytes(source)` | `source: path` | `file` | Binary file handle (returns byte array) |
| `SVG(path, attrs?)` | `path: path`, `attrs?: dict` | `file` | SVG file handle with optional attributes |
| `MD(path, opts?)` | `path: path`, `opts?: dict` | `file` | Markdown file handle (renders to HTML) |
| `markdown(path, opts?)` | `path: path`, `opts?: dict` | `file` | Markdown with frontmatter (returns `{meta, content}`) |
| `file(path, opts?)` | `path: path`, `opts?: dict` | `file` | Auto-detect format from extension |
| `dir(path)` | `path: path` | `file` | Directory listing handle |
| `fileList(path, pattern?)` | `path: path`, `pattern?: string` | `array` | Recursive file listing |

**Usage with I/O operators**:

```parsley
// Reading
let config <== JSON(@./config.json)
let data <== CSV(@./data.csv)
let content <== text(@./readme.txt)
let files <== dir(@./uploads)

// Writing
{name: "Alice"} ==> JSON(@./output.json)
"Hello" ==> text(@./message.txt)
"More" ==>> text(@./log.txt)    // Append

// Markdown with frontmatter
let doc <== markdown(@./post.md)
doc.meta.title                  // Frontmatter field
doc.content                     // Rendered HTML

// File listing
let all = fileList(@./src, "*.pars")  // All .pars files recursively
```

---

### 6.11 Assets

| Function | Arguments | Returns | Description |
|----------|-----------|---------|-------------|
| `asset(path)` | `path: path` | `string` | Get asset path with cache-busting hash |

```parsley
<img src={asset(@./logo.png)} alt="Logo"/>
// Produces: <img src="/assets/logo-a1b2c3d4.png" alt="Logo"/>
```

**Note**: `asset()` is primarily useful in Basil server context for cache-busting.

---

## 7. Standard Library

Import with `@std/` prefix:

```parsley
import @std/math
let {floor, ceil} = import @std/math
```

---

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
| `abs(n)` | Absolute value |
| `min(a, b)` / `min(arr)` | Minimum |
| `max(a, b)` / `max(arr)` | Maximum |
| `sum(arr)` | Sum of array |
| `sqrt(n)` | Square root |
| `pow(base, exp)` | Power |
| `random()` | Random 0-1 |

```parsley
let {floor, ceil, round, abs, min, max, sum, sqrt, pow, random, PI} = import @std/math

floor(3.7)                      // 3
ceil(3.2)                       // 4
round(3.5)                      // 4
abs(-42)                        // 42
min(3, 7)                       // 3
max(3, 7)                       // 7

let nums = [1, 2, 3, 4, 5]
sum(nums)                       // 15
min(nums)                       // 1
max(nums)                       // 5

sqrt(16)                        // 4
pow(2, 10)                      // 1024
PI                              // 3.141592653589793
random()                        // 0.314... (random)
```

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

```parsley
let valid = import @std/valid

valid.string("hello")           // true
valid.number(42)                // true
valid.email("user@example.com") // true
valid.email("invalid")          // false
valid.positive(5)               // true
valid.between(10, 5, 15)        // true
```

---

### 7.3 @std/id

| Function | Description |
|----------|-------------|
| `new()` | ULID-like (26 chars, sortable) |
| `uuid()` | UUID v4 (random) |
| `uuidv4()` | UUID v4 (random) |
| `uuidv7()` | UUID v7 (time-sortable) |
| `nanoid()` | NanoID (21 chars) |
| `cuid()` | CUID2-like |

```parsley
let id = import @std/id

id.new()                        // "01KEQAT4553AQS0P93..."
id.uuid()                       // "5856b07-37fc-4881-..."
id.uuidv7()                     // "019baead-10a5-734c-..."
id.nanoid()                     // "7YoNclTbwXecv1mFxZp6t"
```

---

## 8. Tags (HTML/XML)

Tags are first-class values that render to HTML strings.

### 8.1 Self-Closing Tags

**Must use `/>` syntax:**

```parsley
<br/>
<hr/>
<img src="photo.jpg" alt="A photo"/>
```

---

### 8.2 Pair Tags

Text content must be quoted:

```parsley
<p>"Hello, World!"</p>
<h1>"Welcome"</h1>
```

---

### 8.3 Attributes

#### String Attributes

```parsley
<div class="container">"Content"</div>
<a href="/about">"About Us"</a>
```

#### Expression Attributes

```parsley
let className = "active"
<div class={className}>"Dynamic class"</div>

let isDisabled = true
<button disabled={isDisabled}>"Click"</button>
```

---

### 8.4 Content

#### Variable Content

```parsley
let message = "Hello from variable"
<p>message</p>
```

#### Method Calls

```parsley
let name = "alice"
<span>name.toTitle()</span>
```

#### Parsley Code as Content

All Parsley code works inside tags—`let` statements, `for` loops, `if` expressions, function calls, etc.:

```parsley
<table>
    <thead>
        <tr>
            for (k,_ in rows[0]){
                if (k not in hidden) {
                    let title = k.toTitle()
                    <th class={"th-"+k}>
                        <a href={"?orderBy=" + title}>
                            title
                        </a>
                    </th>
                }
            }
        </tr>
    </thead>
    <tbody>
        for (row in rows){
            <tr>
                for (k,v in row){
                    if (k not in hidden)
                        <td class={"td-"+k}>v</td>
                }
            </tr>
        }
    </tbody>
</table>
```

#### Expression Attributes

Expressions with operators work in attribute values using `{...}`:

```parsley
let count = 5
<div class={"item-" + toString(count)}>"test"</div>
<th class={"th-" + key}>key.toTitle()</th>
```

---

### 8.5 Nested Tags

```parsley
<div class="card">
    <h2>"Title"</h2>
    <p>"Body text"</p>
</div>
```

---

### 8.6 Spread Attributes

```parsley
let attrs = {class: "btn", id: "submit"}
<button ...attrs>"Submit"</button>
```

---

### 8.7 Components

Components are functions that return tags:

```parsley
let Card = fn(props) {
    let title = props.title
    let body = props.body
    <div class="card">
        <h3>title</h3>
        <p>body</p>
    </div>
}
<Card title="My Card" body="Card content"/>
```

#### Tag Pair Syntax with Contents

Components can also use tag pair syntax. Content is passed via `contents`:

```parsley
let Card = fn({title, contents}) {
    <div class="card">
        <h3>title</h3>
        <p>contents</p>
    </div>
}
<Card title="My Card">"Card content"</Card>
```


---

## 9. Comments

Parsley only supports single-line comments:

```parsley
// This is a comment
let x = 5  // End of line comment

// Multiple lines
// require multiple
// comment markers
```

---

## 10. Error Handling

Parsley uses structured errors with error codes for consistent handling.

### 10.1 Error Categories

| Category | Code Pattern | Description |
|----------|--------------|-------------|
| Parse | `PARSE-0xxx` | Syntax errors during parsing |
| Type | `TYPE-0xxx` | Type mismatch errors |
| Arity | `ARITY-0xxx` | Wrong number of arguments |
| Undefined | `UNDEF-0xxx` | Unknown identifier, method, or property |
| Index | `INDEX-0xxx` | Array/dictionary index errors |
| Format | `FMT-0xxx` | Formatting/parsing errors |
| Validation | `VAL-0xxx` | Value validation failures |
| I/O | `IO-0xxx` | File and network errors |

### 10.2 Common Errors

#### Type Errors (`TYPE-0xxx`)

```parsley
// TYPE-0001: Expected type mismatch
"hello".split(123)              // Expected string, got integer

// TYPE-0012: Argument type mismatch
[1, 2, 3].take("two")           // Argument must be an integer

// TYPE-0007: Cannot iterate
for (x in 42) { x }             // Cannot iterate over integer
```

#### Arity Errors (`ARITY-0xxx`)

```parsley
// ARITY-0001: Wrong number of arguments
"hello".split()                 // Missing required argument
"hello".split("l", "extra")     // Too many arguments

// ARITY-0004: Outside valid range
[1, 2, 3].format("and", "en", "extra")  // Expects 0-2 arguments
```

#### Undefined Errors (`UNDEF-0xxx`)

```parsley
// UNDEF-0001: Identifier not found
unknownVariable                 // Identifier not found: unknownVariable

// UNDEF-0002: Unknown method
"hello".unknownMethod()         // Unknown method 'unknownMethod' for string

// UNDEF-0004: Unknown property
@now.unknownProp                // Unknown property 'unknownProp' on datetime
```

#### Index Errors (`INDEX-0xxx`)

```parsley
// INDEX-0001: Out of bounds
let arr = [1, 2, 3]
arr[10]                         // Index 10 out of bounds

// INDEX-0005: Key not found
let d = {a: 1}
d.insertAfter("missing", "b", 2)  // Key 'missing' not found
```

### 10.3 Handling Errors with `try`

Use `try` to catch errors and return a result dictionary:

```parsley
let result = try {
    riskyOperation()
}
// Returns: {result: value, error: null} on success
// Returns: {result: null, error: "message"} on failure
```

**Pattern: Check and handle**

```parsley
let result = try parseJSON(userInput)
if (result.error) {
    <div class="error">"Invalid JSON: " + result.error</div>
} else {
    <pre>result.result.toJSON()</pre>
}
```

### 10.4 Throwing Errors with `fail`

Use `fail()` to throw catchable errors:

```parsley
let validateAge = fn(age) {
    check age >= 0 else fail("Age cannot be negative")
    check age < 150 else fail("Age is unrealistic")
    age
}

let result = try validateAge(-5)
result.error                    // "Age cannot be negative"
```

### 10.5 Error Prevention

#### Optional Index Access

Use `[?index]` to return `null` instead of error for missing indices:

```parsley
let arr = [1, 2, 3]
arr[?99]                        // null (no error)
arr[99]                         // Error: index out of bounds
```

#### Null Coalescing

Use `??` to provide default values:

```parsley
let name = user.name ?? "Anonymous"
let config = loadConfig() ?? {default: true}
```

#### Check Guards

Use `check` for early validation:

```parsley
let processUser = fn(user) {
    check user != null else "User required"
    check user.email else "Email required"
    // Continue with valid user...
}
```

---

## Reserved Keywords

```
fn, function, let, for, in, if, else, return, export, import,
try, check, stop, skip, true, false, null, and, or, as, via
```

---

## Appendix A: Type Summary

| Type | Literal | Properties | Key Methods |
|------|---------|------------|-------------|
| `integer` | `42`, `-15` | — | `format`, `currency`, `percent`, `humanize` |
| `float` | `3.14`, `-2.5` | — | `format`, `currency`, `percent`, `humanize` |
| `string` | `"text"`, `` `template` ``, `'raw'` | — | `toUpper`, `split`, `replace`, `trim`, `length` |
| `boolean` | `true`, `false` | — | — |
| `null` | `null` | — | — |
| `array` | `[1, 2, 3]` | — | `map`, `filter`, `reduce`, `sort`, `join` |
| `dictionary` | `{a: 1, b: 2}` | dynamic | `keys`, `values`, `has`, `render` |
| `function` | `fn(x) { x }` | — | — |
| `datetime` | `@2024-12-25` | `year`, `month`, `day`, etc. | `format` |
| `duration` | `@1d`, `@2h30m` | `months`, `seconds`, etc. | `format` |
| `money` | `$12.34`, `EUR#50` | `amount`, `currency`, `scale` | `format`, `split`, `abs` |
| `path` | `@./file.txt` | `segments`, `extension`, etc. | `match`, `toURL` |
| `url` | `@https://...` | `scheme`, `host`, `query`, etc. | `origin`, `pathname` |
| `regex` | `/pattern/flags` | `pattern`, `flags` | `test`, `replace` |
| `table` | `CSV(@./data.csv)` | `row`, `rows`, `columns` | `where`, `orderBy`, `toHTML` |

---

## Appendix B: Method Reference

### String Methods (26 methods)

| Method | Arity | Description |
|--------|-------|-------------|
| `toUpper()` | 0 | Convert to uppercase |
| `toLower()` | 0 | Convert to lowercase |
| `toTitle()` | 0 | Title case |
| `trim()` | 0 | Remove surrounding whitespace |
| `split(delim)` | 1 | Split by delimiter |
| `replace(old, new)` | 2 | Replace all occurrences |
| `length()` | 0 | Character count |
| `includes(substr)` | 1 | Contains substring? |
| `highlight(pattern, tag?)` | 1-2 | Wrap matches in HTML tag |
| `paragraphs()` | 0 | Convert to `<p>` tags |
| `render(dict?)` | 0-1 | Interpolate template |
| `parseJSON()` | 0 | Parse as JSON |
| `parseCSV(hasHeader?)` | 0-1 | Parse as CSV |
| `collapse()` | 0 | Collapse whitespace |
| `normalizeSpace()` | 0 | Collapse + trim |
| `stripSpace()` | 0 | Remove all whitespace |
| `stripHtml()` | 0 | Remove HTML tags |
| `digits()` | 0 | Extract only digits |
| `slug()` | 0 | URL-safe slug |
| `htmlEncode()` | 0 | Encode HTML entities |
| `htmlDecode()` | 0 | Decode HTML entities |
| `urlEncode()` | 0 | URL encode |
| `urlDecode()` | 0 | URL decode |
| `urlPathEncode()` | 0 | Encode path segment |
| `urlQueryEncode()` | 0 | Encode query value |
| `outdent()` | 0 | Remove common indent |
| `indent(n)` | 1 | Add n spaces to lines |

### Array Methods (18 methods)

| Method | Arity | Description |
|--------|-------|-------------|
| `length()` | 0 | Element count |
| `reverse()` | 0 | Reverse order |
| `sort()` | 0 | Natural sort |
| `sortBy(fn)` | 1 | Sort by key function |
| `map(fn)` | 1 | Transform elements |
| `filter(fn)` | 1 | Filter by predicate |
| `reduce(fn, init)` | 2 | Reduce to value |
| `format(style?, locale?)` | 0-2 | Format as list |
| `join(sep?)` | 0-1 | Join to string |
| `toJSON()` | 0 | Convert to JSON |
| `toCSV(hasHeader?)` | 0-1 | Convert to CSV |
| `shuffle()` | 0 | Random order |
| `pick(n?)` | 0-1 | Random element(s) |
| `take(n)` | 1 | n unique random |
| `has(item)` | 1 | Contains item? |
| `hasAny(arr)` | 1 | Contains any? |
| `hasAll(arr)` | 1 | Contains all? |
| `insert(i, val)` | 2 | Insert at index |

### Dictionary Methods (10 methods)

| Method | Arity | Description |
|--------|-------|-------------|
| `keys()` | 0 | Get all keys |
| `values()` | 0 | Get all values |
| `entries(k?, v?)` | 0 or 2 | Get key-value pairs |
| `has(key)` | 1 | Key exists? |
| `delete(key)` | 1 | Remove key (mutates) |
| `insertAfter(after, k, v)` | 3 | Insert after key |
| `insertBefore(before, k, v)` | 3 | Insert before key |
| `render(template)` | 1 | Render template |
| `toJSON()` | 0 | Convert to JSON |

### Number Methods (4 methods)

| Method | Arity | Description |
|--------|-------|-------------|
| `format(locale?)` | 0-1 | Locale format |
| `currency(code, locale?)` | 1-2 | Currency format |
| `percent(locale?)` | 0-1 | Percentage format |
| `humanize(locale?)` | 0-1 | Compact format (1.2K) |

### Table Methods (22 methods)

| Method | Arity | Description |
|--------|-------|-------------|
| `where(fn)` | 1 | Filter rows by predicate |
| `orderBy(col, dir?)` | 1-2 | Sort by column |
| `select(cols...)` | 1+ | Select columns |
| `limit(n, offset?)` | 1-2 | Limit rows |
| `count()` | 0 | Number of rows |
| `sum(col)` | 1 | Sum column |
| `avg(col)` | 1 | Average column |
| `min(col)` | 1 | Minimum value |
| `max(col)` | 1 | Maximum value |
| `column(col)` | 1 | Get column values |
| `appendRow(row)` | 1 | Add row at end |
| `insertRowAt(i, row)` | 2 | Insert row at index |
| `appendCol(name, vals)` | 2 | Add column at end |
| `insertColAfter(after, name, vals)` | 3 | Insert column after |
| `insertColBefore(before, name, vals)` | 3 | Insert column before |
| `rowCount()` | 0 | Number of rows |
| `columnCount()` | 0 | Number of columns |
| `toHTML(footer?)` | 0-1 | Convert to HTML |
| `toCSV()` | 0 | Convert to CSV |
| `toMarkdown()` | 0 | Convert to Markdown |
| `toJSON()` | 0 | Convert to JSON |
