# Parsley Reference

Complete reference for all Parsley types, methods, and operators.

## Table of Contents

- [Data Types](#data-types)
- [Operators](#operators)
- [String Methods](#string-methods)
- [Array Methods](#array-methods)
- [Dictionary Methods](#dictionary-methods)
- [Number Methods](#number-methods)
- [Datetime Methods](#datetime-methods)
- [Duration Methods](#duration-methods)
- [Money Type](#money-type)
- [Path Methods](#path-methods)
- [URL Methods](#url-methods)
- [File I/O](#file-io)
- [Process Execution](#process-execution)
- [Database](#database)
- [Regex](#regex)
- [Modules](#modules)
- [Tags](#tags)
- [Error Handling](#error-handling)
- [Utility Functions](#utility-functions)
- [Basil Server Functions](#basil-server-functions)
- [Go Library](#go-library)

---

## Data Types

| Type | Example | Description |
|------|---------|-------------|
| Integer | `42`, `-15` | Whole numbers |
| Float | `3.14`, `2.718` | Decimal numbers |
| String | `"hello"`, `"world"` | Text with `{interpolation}` |
| Boolean | `true`, `false` | Logical values |
| Null | `null` | Absence of value |
| Array | `[1, 2, 3]` | Ordered collections |
| Dictionary | `{x: 1, y: 2}` | Key-value pairs |
| Function | `fn(x) { x * 2 }` | First-class functions |
| Regex | `/pattern/flags` | Regular expressions |
| Date | `@2024-11-26` | Date only |
| DateTime | `@2024-11-26T15:30:00` | Date and time |
| Time | `@12:30`, `@12:30:45` | Time only (uses current date internally) |
| Duration | `@1d`, `@2h30m` | Time spans |
| Money | `$12.34`, `EUR#50.00` | Currency values with exact arithmetic |
| Path | `@./file.pars` | File system paths |
| URL | `@https://example.com` | Web addresses |
| File Handle | `JSON(@./config.json)` | File with format binding |
| Directory | `dir(@./folder)` | Directory handle |

---

## Operators

### Arithmetic

| Operator | Description | Example |
|----------|-------------|---------|
| `+` | Addition | `2 + 3` → `5` |
| `-` | Subtraction | `5 - 2` → `3` |
| `-` | Array subtraction | `[1,2,3] - [2]` → `[1, 3]` |
| `-` | Dictionary subtraction | `{a:1, b:2} - {b:0}` → `{a: 1}` |
| `*` | Multiplication | `4 * 3` → `12` |
| `*` | String repetition | `"ab" * 3` → `"ababab"` |
| `*` | Array repetition | `[1,2] * 3` → `[1, 2, 1, 2, 1, 2]` |
| `/` | Division | `10 / 4` → `2.5` |
| `/` | Array chunking | `[1,2,3,4] / 2` → `[[1, 2], [3, 4]]` |
| `%` | Modulo | `10 % 3` → `1` |
| `++` | Concatenation | `[1] ++ [2]` → `[1, 2]` |
| `++` | Scalar to array | `1 ++ [2,3]` → `[1, 2, 3]` |
| `++` | Array to scalar | `[1,2] ++ 3` → `[1, 2, 3]` |
| `..` | Range (inclusive) | `1..5` → `[1, 2, 3, 4, 5]` |

### Comparison

| Operator | Description |
|----------|-------------|
| `==` | Equal |
| `!=` | Not equal |
| `<` | Less than |
| `<=` | Less than or equal |
| `>` | Greater than |
| `>=` | Greater than or equal |

### Logical

| Operator | Description | Example |
|----------|-------------|---------|
| `&&` | Boolean AND | `true && false` → `false` |
| `&&` | Array intersection | `[1,2,3] && [2,3,4]` → `[2, 3]` |
| `&&` | Dictionary intersection | `{a:1, b:2} && {b:3, c:4}` → `{b: 2}` |
| `\|\|` | Boolean OR | `true \|\| false` → `true` |
| `\|\|` | Array union | `[1,2] \|\| [2,3]` → `[1, 2, 3]` |
| `!` | NOT | `!true` → `false` |

### Set Operations

**Array Intersection** (`&&`): Returns elements present in both arrays (deduplicated).

```parsley
[1, 2, 3] && [2, 3, 4]           // [2, 3]
[1, 2, 2, 3] && [2, 3, 3, 4]     // [2, 3] (duplicates removed)
[1, 2] && [3, 4]                 // [] (no common elements)
```

**Array Union** (`||`): Merges arrays, removing duplicates.

```parsley
[1, 2] || [2, 3]                 // [1, 2, 3]
[1, 1, 2] || [2, 3, 3]           // [1, 2, 3] (duplicates removed)
[1, 2] || []                     // [1, 2]
```

**Array Subtraction** (`-`): Removes elements from left array that exist in right.

```parsley
[1, 2, 3, 4] - [2, 4]            // [1, 3]
[1, 2, 2, 3] - [2]               // [1, 3] (all instances removed)
[1, 2, 3] - [4, 5]               // [1, 2, 3] (no change)
```

**Dictionary Intersection** (`&&`): Returns dictionary with keys present in both (left values kept).

```parsley
{a: 1, b: 2, c: 3} && {b: 99, c: 99, d: 4}  // {b: 2, c: 3}
{a: 1} && {b: 2}                             // {}
```

**Dictionary Subtraction** (`-`): Removes keys from left that exist in right (values in right don't matter).

```parsley
{a: 1, b: 2, c: 3} - {b: 0, d: 0}  // {a: 1, c: 3}
{a: 1, b: 2} - {c: 3}              // {a: 1, b: 2} (no change)
```

**Array Chunking** (`/`): Splits array into chunks of specified size.

```parsley
[1, 2, 3, 4, 5, 6] / 2    // [[1, 2], [3, 4], [5, 6]]
[1, 2, 3, 4, 5] / 2       // [[1, 2], [3, 4], [5]]
[1, 2] / 5                // [[1, 2]]
[1, 2, 3] / 0             // ERROR: chunk size must be positive
```

**String Repetition** (`*`): Repeats string N times.

```parsley
"abc" * 3                 // "abcabcabc"
"x" * 5                   // "xxxxx"
"test" * 0                // ""
"hi" * -1                 // "" (negative treated as 0)
```

**Array Repetition** (`*`): Repeats array contents N times.

```parsley
[1, 2] * 3                // [1, 2, 1, 2, 1, 2]
["a"] * 4                 // ["a", "a", "a", "a"]
[1, 2, 3] * 0             // []
```

**Scalar Concatenation** (`++`): Wraps scalars in arrays for concatenation.
```parsley
1 ++ [2, 3, 4]            // [1, 2, 3, 4]
[1, 2, 3] ++ 4            // [1, 2, 3, 4]
1 ++ 2 ++ 3               // [1, 2, 3]
"a" ++ ["b", "c"]         // ["a", "b", "c"]
```

### Range Operator

**Range** (`..`): Creates inclusive integer ranges from start to end.
```parsley
1..5                      // [1, 2, 3, 4, 5]
0..10                     // [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
5..1                      // [5, 4, 3, 2, 1] (reverse)
-2..2                     // [-2, -1, 0, 1, 2]
10..10                    // [10] (single element)
```

**Common Use Cases:**

```parsley
// Loop over a range
for (i in 1..10) { log(i) }

// Generate sequences
let evens = (1..10).filter(fn(x) { x % 2 == 0 })  // [2, 4, 6, 8, 10]
let squares = (1..5).map(fn(x) { x * x })         // [1, 4, 9, 16, 25]

// Array indexing
let first10 = data[0..9]
let countdown = (10..1).join(", ")  // "10, 9, 8, 7, 6, 5, 4, 3, 2, 1"

// With variables
let start = 5
let end = 15
let range = start..end
```

### Pattern Matching

| Operator | Description | Example |
|----------|-------------|---------|
| `~` | Regex match | `"test" ~ /\w+/` → `["test"]` |
| `!~` | Regex not-match | `"abc" !~ /\d/` → `true` |

### Nullish Coalescing

| Operator | Description | Example |
|----------|-------------|---------|
| `??` | Default if null | `null ?? "default"` → `"default"` |

```parsley
value ?? default           // Returns default only if value is null
null ?? "fallback"         // "fallback"
"hello" ?? "fallback"      // "hello"
0 ?? 42                    // 0 (not null)
a ?? b ?? c ?? "default"   // First non-null value
```

### File I/O

| Operator | Description | Example |
|----------|-------------|---------|
| `<==` | Read from file | `let data <== JSON(@./file.json)` |
| `==>` | Write to file | `data ==> JSON(@./out.json)` |
| `==>>` | Append to file | `line ==>> lines(@./log.txt)` |

### Process Execution

| Operator | Description | Example |
|----------|-------------|---------|
| `<=#=>` | Execute command with input | `let result = COMMAND("ls") <=#=> null` |

### Database

| Operator | Description | Example |
|----------|-------------|---------|
| `<=?=>` | Query single row | `let user = db <=?=> "SELECT * FROM users WHERE id = 1"` |
| `<=??=>` | Query multiple rows | `let users = db <=??=> "SELECT * FROM users"` |
| `<=!=>` | Execute mutation | `let result = db <=!=> "INSERT INTO users (name) VALUES ('Alice')"` |

### Other

| Operator | Description |
|----------|-------------|
| `=` | Assignment |
| `:` | Key-value separator |
| `.` | Property/method access |
| `[]` | Indexing and slicing |

---

## String Methods

| Method | Description | Example |
|--------|-------------|---------|
| `.length()` | String length | `"hello".length()` → `5` |
| `.toUpper()` | Uppercase | `"hello".toUpper()` → `"HELLO"` |
| `.toLower()` | Lowercase | `"HELLO".toLower()` → `"hello"` |
| `.trim()` | Remove whitespace | `"  hi  ".trim()` → `"hi"` |
| `.split(delim)` | Split to array | `"a,b,c".split(",")` → `["a","b","c"]` |
| `.replace(old, new)` | Replace text | `"hello".replace("l", "L")` → `"heLLo"` |
| `.includes(substr)` | Check if contains | `"hello".includes("ell")` → `true` |
| `.highlight(phrase)` | Highlight search matches | `"hello world".highlight("world")` → `"hello <mark>world</mark>"` |
| `.highlight(phrase, tag)` | With custom tag | `"hello".highlight("ell", "strong")` → `"h<strong>ell</strong>o"` |
| `.paragraphs()` | Text to HTML paragraphs | `"Para one.\n\nPara two.".paragraphs()` → `"<p>Para one.</p><p>Para two.</p>"` |

### Indexing and Slicing

```parsley
"hello"[0]      // "h"
"hello"[-1]     // "o" (last)
"hello"[1:4]    // "ell"
"hello"[2:]     // "llo"
"hello"[:3]     // "hel"

// Optional indexing - returns null instead of error
"hello"[?99]    // null
"hello"[?0]     // "h"
""[?0] ?? "?"   // "?" (safe empty string access)
```

### Interpolation

```parsley
let name = "World"
"Hello, {name}!"  // "Hello, World!"
```

---

## Array Methods

| Method | Description | Example |
|--------|-------------|---------|
| `.length()` | Array length | `[1,2,3].length()` → `3` |
| `.insert(i, v)` | Insert before index | `[1,3].insert(1, 2)` → `[1,2,3]` |
| `.sort()` | Sort ascending | `[3,1,2].sort()` → `[1,2,3]` |
| `.reverse()` | Reverse order | `[1,2,3].reverse()` → `[3,2,1]` |
| `.shuffle()` | Random order | `[1,2,3].shuffle()` → `[2,3,1]` |
| `.pick()` | Random element | `[1,2,3].pick()` → `2` |
| `.pick(n)` | n random elements (with replacement) | `[1,2,3].pick(5)` → `[1,3,1,2,1]` |
| `.take(n)` | n unique random elements | `[1,2,3,4,5].take(3)` → `[4,1,3]` |
| `.map(fn)` | Transform each | `[1,2].map(fn(x){x*2})` → `[2,4]` |
| `.filter(fn)` | Keep matching | `[1,2,3].filter(fn(x){x>1})` → `[2,3]` |
| `.join()` | Join to string | `["a","b","c"].join()` → `"abc"` |
| `.join(sep)` | Join with separator | `["a","b","c"].join(",")` → `"a,b,c"` |
| `.format()` | List as prose | `["a","b"].format()` → `"a and b"` |
| `.format("or")` | With conjunction | `["a","b"].format("or")` → `"a or b"` |

### Array Literals

Arrays are created using bracket syntax:

```parsley
let nums = [1, 2, 3]
let names = ["Alice", "Bob", "Carol"]
let mixed = [1, "two", true, null]
let nested = [[1, 2], [3, 4]]
let empty = []
```

### Array Destructuring

Extract values from arrays into variables using bracket syntax:

```parsley
let [a, b, c] = [1, 2, 3]    // a=1, b=2, c=3
let [first, second] = nums    // first=1, second=2
let [x, y] = [10, 20, 30]     // x=10, y=20 (extras ignored)
```

Use `...rest` to collect remaining elements (must be last):
```parsley
let [head, ...tail] = [1, 2, 3, 4]  // head=1, tail=[2, 3, 4]
let [a, b, ...rest] = [1, 2]        // a=1, b=2, rest=[]
let [...all] = [1, 2, 3]            // all=[1, 2, 3]
let [_, ...rest] = arr              // skip first, collect rest
```

Destructuring works in function parameters:
```parsley
let sum = fn([a, b]) { a + b }
sum([3, 4])  // 7

let head = fn([first, ...rest]) { first }
head([1, 2, 3])  // 1

let tail = fn([_, ...rest]) { rest }
tail([1, 2, 3])  // [2, 3]
```

### Indexing and Slicing

```parsley
nums[0]      // First element
nums[-1]     // Last element
nums[1:3]    // Elements 1 and 2
nums[2:]     // From index 2 to end
nums[:2]     // From start to index 2
```

### Optional Indexing

Use `[?n]` for safe access that returns `null` instead of an error when out of bounds:
```parsley
let arr = [1, 2, 3]
arr[0]       // 1
arr[99]      // ERROR: index out of range
arr[?99]     // null (no error)
arr[?0]      // 1 (same as arr[0] when in bounds)

// Combine with null coalesce for defaults
arr[?99] ?? "default"  // "default"
[][?0] ?? "empty"      // "empty"

// Works with negative indices too
arr[?-99]    // null
```

### Concatenation

```parsley
[1, 2] ++ [3, 4]  // [1, 2, 3, 4]
1 ++ [2, 3]       // [1, 2, 3] (scalar concatenation)
[1, 2] ++ 3       // [1, 2, 3]
```

### Set Operations

```parsley
[1, 2, 3] && [2, 3, 4]  // [2, 3] (intersection)
[1, 2] || [2, 3]        // [1, 2, 3] (union)
[1, 2, 3] - [2]         // [1, 3] (subtraction)
```

### Other Operations

```parsley
[1, 2, 3, 4] / 2  // [[1, 2], [3, 4]] (chunking)
[1, 2] * 3        // [1, 2, 1, 2, 1, 2] (repetition)
```

---

## Dictionary Methods

| Method | Description | Example |
|--------|-------------|---------|
| `.keys()` | All keys | `{a:1}.keys()` → `["a"]` |
| `.values()` | All values | `{a:1}.values()` → `[1]` |
| `.has(key)` | Key exists | `{a:1}.has("a")` → `true` |
| `.delete(key)` | Remove key | `d.delete("a")` → removes key `a` |
| `.insertAfter(k, newK, v)` | Insert after key | `{a:1,c:3}.insertAfter("a","b",2)` → `{a:1,b:2,c:3}` |
| `.insertBefore(k, newK, v)` | Insert before key | `{b:2,c:3}.insertBefore("b","a",1)` → `{a:1,b:2,c:3}` |

### Access

```parsley
dict.key        // Dot notation
dict["key"]     // Bracket notation
```

### Removing Keys

```parsley
let d = {a: 1, b: 2, c: 3}
d.delete("b")   // d is now {a: 1, c: 3}
d.delete("x")   // No error if key doesn't exist
```

### Self-Reference with `this`

```parsley
let config = {
    width: 100,
    height: 200,
    area: this.width * this.height  // Computed on access
}
```

### Merging

```parsley
{a: 1} ++ {b: 2}  // {a: 1, b: 2}
```

### Set Operations

```parsley
{a: 1, b: 2} && {b: 3, c: 4}  // {b: 2} (intersection, left values kept)
{a: 1, b: 2} - {b: 0}         // {a: 1} (subtract keys)
```

---

## Number Methods

| Method | Description | Example |
|--------|-------------|---------|
| `.format()` | Locale format | `1234567.format()` → `"1,234,567"` |
| `.format(locale)` | With locale | `1234.format("de-DE")` → `"1.234"` |
| `.currency(code)` | Currency format | `99.currency("USD")` → `"$99.00"` |
| `.currency(code, locale)` | With locale | `99.currency("EUR","de-DE")` → `"99,00 €"` |
| `.percent()` | Percentage | `0.125.percent()` → `"13%"` |
| `.humanize()` | Compact format | `1234567.humanize()` → `"1.2M"` |
| `.humanize(locale)` | With locale | `1234.humanize("de")` → `"1,2K"` |

### Math Functions

```parsley
sqrt(16)        // 4
round(3.7)      // 4
pow(2, 8)       // 256
pi()            // 3.14159...
sin(x), cos(x), tan(x)
asin(x), acos(x), atan(x)
```

---

## Datetime Methods

### Creation

```parsley
now()                                    // Current datetime
time("2024-11-26")                       // Parse ISO date
time("2024-11-26T15:30:00")              // With time
time(1732579200)                         // Unix timestamp
time({year: 2024, month: 12, day: 25})   // From components
```

### Literals

Parsley supports three kinds of datetime literals, each with its own display format:

```parsley
@2024-11-26           // Date only
@2024-11-26T15:30:00  // Full datetime
@12:30                // Time only (HH:MM)
@12:30:45             // Time only with seconds (HH:MM:SS)
```

### Literal Kinds

Each datetime literal tracks its kind, which determines how it displays when converted to a string:

| Literal | Kind | String Output |
|---------|------|---------------|
| `@2024-11-26` | `"date"` | `"2024-11-26"` |
| `@2024-11-26T15:30:00` | `"datetime"` | `"2024-11-26T15:30:00Z"` |
| `@12:30` | `"time"` | `"12:30"` |
| `@12:30:45` | `"time_seconds"` | `"12:30:45"` |

```parsley
// Access the kind
@2024-11-26.kind           // "date"
@2024-11-26T15:30:00.kind  // "datetime"
@12:30.kind                // "time"
@12:30:45.kind             // "time_seconds"

// String conversion respects kind
toString(@2024-11-26)           // "2024-11-26"
toString(@2024-11-26T15:30:00)  // "2024-11-26T15:30:00Z"
toString(@12:30)                // "12:30"
toString(@12:30:45)             // "12:30:45"
```

### Time-Only Literals

Time-only literals (`@HH:MM` or `@HH:MM:SS`) use the current UTC date internally but display as time only:

```parsley
let meeting = @14:30
meeting.hour     // 14
meeting.minute   // 30
meeting.kind     // "time"

// Internal date is today (UTC)
meeting.year     // Current year
meeting.month    // Current month
meeting.day      // Current day

// But string output shows time only
toString(meeting)  // "14:30"
```

### Kind Preservation

The kind is preserved through arithmetic operations:

```parsley
// Date arithmetic stays date
(@2024-12-25 + 86400).kind        // "date"
(@2024-12-25 + @1d).kind          // "date"

// Datetime arithmetic stays datetime
(@2024-12-25T14:30:00 + 3600).kind  // "datetime"

// Time arithmetic stays time
(@12:30 + 3600).kind              // "time"
(@12:30:45 + 60).kind             // "time_seconds"
```

### Interpolated Datetime Templates

Use `@(...)` syntax for datetime literals with embedded expressions:

```parsley
// Date interpolation
month = "06"
day = "15"
dt = @(2024-{month}-{day})
dt.year    // 2024
dt.month   // 6
dt.day     // 15
dt.kind    // "date"

// Full datetime interpolation
year = "2025"
hour = "14"
dt2 = @({year}-12-25T{hour}:30:00)
dt2.year   // 2025
dt2.hour   // 14
dt2.kind   // "datetime"

// Time-only interpolation
h = "09"
m = "15"
meeting = @({h}:{m})
meeting.hour    // 9
meeting.minute  // 15
meeting.kind    // "time"

// Expressions in interpolations
baseDay = 10
dt3 = @(2024-12-{baseDay + 5})
dt3.day    // 15

// Dictionary-based construction
date = { year: "2024", month: "07", day: "04" }
dt4 = @({date.year}-{date.month}-{date.day})
dt4.month  // 7
```

The kind is automatically determined:
- Date templates (YYYY-MM-DD) → `"date"`
- Full datetime templates → `"datetime"`
- Time templates (HH:MM) → `"time"`

Static datetime literals (`@2024-12-25`) remain unchanged and don't require parentheses.

### Properties

| Property | Description |
|----------|-------------|
| `.year` | Year number |
| `.month` | Month (1-12) |
| `.day` | Day of month |
| `.hour` | Hour (0-23) |
| `.minute` | Minute (0-59) |
| `.second` | Second (0-59) |
| `.weekday` | Day name ("Monday", etc.) |
| `.iso` | ISO 8601 string |
| `.unix` | Unix timestamp |
| `.kind` | Literal kind ("date", "datetime", "time", "time_seconds") |
| `.date` | Date only ("2024-11-26") |
| `.time` | Time only ("15:30") |
| `.dayOfYear` | Day number (1-366) |
| `.week` | ISO week number (1-53) |

### Methods

| Method | Description | Example |
|--------|-------------|---------|
| `.format()` | Default format | `dt.format()` → `"11/26/2024"` |
| `.format(style)` | Style format | `dt.format("long")` → `"November 26, 2024"` |
| `.format(style, locale)` | Localized | `dt.format("long","de-DE")` → `"26. November 2024"` |
| `.toDict()` | Dictionary form | `dt.toDict()` → `{year: 2024, month: 11, kind: "datetime", ...}` |

Style options: `"short"`, `"medium"`, `"long"`, `"full"`

### Comparisons

All datetime kinds can be compared:

```parsley
@12:30 < @14:00           // true
@2024-12-25 > @2024-12-24 // true
@12:30:45 == @12:30:45    // true
```

### Intersection Operator (`&&`)

Combine date and time components using the `&&` operator:

```parsley
// Combine date and time
@1968-11-21 && @12:30        // → @1968-11-21T12:30:00
@09:15 && @2024-03-15        // → @2024-03-15T09:15:00

// Replace time in a datetime
@1968-11-21T08:00:00 && @12:30  // → @1968-11-21T12:30:00

// Replace date in a datetime
@1968-11-21T08:00:00 && @2024-03-15  // → @2024-03-15T08:00:00
```

| Expression | Result |
|------------|--------|
| `Date && Time` | DateTime (combine) |
| `Time && Date` | DateTime (combine) |
| `DateTime && Time` | DateTime (replace time) |
| `DateTime && Date` | DateTime (replace date) |
| `Date && Date` | Error |
| `Time && Time` | Error |
| `DateTime && DateTime` | Error |

---

## Duration Methods

### Literals

```parsley
@1d          // 1 day
@2h          // 2 hours
@30m         // 30 minutes
@1d2h30m     // Combined
@-1d         // Negative (yesterday)
```

### Methods

| Method | Description | Example |
|--------|-------------|---------|
| `.format()` | Relative time | `@1d.format()` → `"tomorrow"` |
| `.format(locale)` | Localized | `@-1d.format("de-DE")` → `"gestern"` |
| `.toDict()` | Dictionary form | `@1d2h.toDict()` → `{__type: "duration", ...}` |

### String Conversion

Durations convert to human-readable strings in templates and print statements:

```parsley
let d = @1d2h30m
"{d}"              // "1 day, 2 hours, 30 minutes"
log(d)             // 1 day, 2 hours, 30 minutes
```

### Arithmetic

```parsley
let christmas = @2025-12-25
let daysUntil = christmas - now()
daysUntil.format()  // "in 4 weeks"
```

---

## Money Type

Money values provide exact arithmetic for financial calculations with currency type safety.

### Literals

```parsley
// Currency symbol syntax
$12.34       // USD
£99.99       // GBP
€50.00       // EUR
¥1000        // JPY (no decimals)

// Compound symbols
CA$25.00     // Canadian Dollar
AU$50.00     // Australian Dollar
HK$100.00    // Hong Kong Dollar
S$75.50      // Singapore Dollar
CN¥500       // Chinese Yuan

// CODE# syntax (any 3-letter currency)
USD#12.34    // Same as $12.34
GBP#99.99    // Same as £99.99
EUR#50.00    // Same as €50.00
BTC#1.00000000  // Bitcoin (custom scale)
```

### Constructor

```parsley
money(12.34, "USD")       // $12.34
money(1000, "JPY")        // ¥1000
money(1234, "USD", 2)     // $12.34 (amount in minor units with explicit scale)
```

### Arithmetic

Money arithmetic maintains exact precision:

```parsley
$10.00 + $5.00    // $15.00
$20.00 - $8.00    // $12.00
$10.00 * 3        // $30.00
$15.00 / 3        // $5.00 (banker's rounding)
-$50.00           // Negative amount
```

**Rules:**

- Money + Money: Same currency only
- Money - Money: Same currency only
- Money * scalar: Allowed
- scalar * Money: Allowed
- Money / scalar: Allowed (uses banker's rounding)
- Money / Money: Error (use `.amount` if you need a ratio)
- Money + scalar: Error (ambiguous)

### Comparison

```parsley
$10.00 > $5.00    // true
$10.00 == $10     // true (same value)
$10.00 != $5.00   // true
$10.00 < £5.00    // Error: cannot mix currencies
```

### Properties

| Property | Type | Description |
|----------|------|-------------|
| `.currency` | String | ISO currency code (`"USD"`, `"GBP"`) |
| `.amount` | Integer | Amount in smallest unit (cents) |
| `.scale` | Integer | Decimal places (2 for USD, 0 for JPY) |

```parsley
$12.34.currency   // "USD"
$12.34.amount     // 1234
$12.34.scale      // 2
¥1000.scale       // 0
```

### Methods

| Method | Description | Example |
|--------|-------------|---------|
| `.format()` | Locale-aware formatting | `$1234.56.format()` → `"$ 1,234.56"` |
| `.format(locale)` | Specific locale | `$1234.56.format("de-DE")` → formatted for German |
| `.abs()` | Absolute value | `(-$50.00).abs()` → `$50.00` |
| `.split(n)` | Split into n parts | `$100.00.split(3)` → `[$33.34, $33.33, $33.33]` |

### Split Method

The `.split(n)` method divides money fairly, distributing any remainder:

```parsley
$100.00.split(3)  // [$33.34, $33.33, $33.33]
$10.00.split(4)   // [$2.50, $2.50, $2.50, $2.50]
$0.01.split(3)    // [$0.01, $0.00, $0.00]
```

The sum of split parts always equals the original:
```parsley
let parts = $100.00.split(3)
// parts[0] + parts[1] + parts[2] == $100.00
```

### Currency Mismatch Errors

Mixing currencies is a runtime error:

```parsley
$10.00 + £5.00    // Error: cannot mix currencies: USD and GBP
$10.00 == €10.00  // Error: cannot mix currencies: USD and EUR
```

### Using Money in Templates

```parsley
let price = $29.99
let tax = price * 0.08
let total = price + tax

<div>
  <span>Price: {price}</span>
  <span>Tax: {tax}</span>
  <span>Total: {total}</span>
</div>
```

---

## Path Methods

### Creation

```parsley
@./config.json       // Relative path
@/usr/local/bin      // Absolute path
path("some/path")    // Dynamic path
```

### Path Cleaning

Paths are automatically cleaned when created, following [Rob Pike's cleanname algorithm](https://9p.io/sys/doc/lexnames.html):
- `.` (current directory) elements are eliminated
- `..` elements eliminate the preceding component
- `..` at the start of absolute paths is eliminated (`/../foo` → `/foo`)
- `..` at the start of relative paths is preserved (`../foo` stays as is)

```parsley
let p = @/foo/../bar
p.string  // "/bar"

let p = @./a/b/../../c
p.string  // "./c"
```

### Interpolated Path Templates

Use `@(...)` syntax for paths with embedded expressions:

```parsley
name = "config"
p = @(./data/{name}.json)
p.string  // "./data/config.json"

dir = "src"
file = "main"
p = @(./{dir}/{file}.go)
p.string  // "./src/main.go"

// Expressions in interpolations
n = 1
p = @(./file{n + 1}.txt)
p.string  // "./file2.txt"
```

Static path literals (`@./path`) remain unchanged and don't require parentheses.

### Properties

| Property | Description | Example |
|----------|-------------|---------|
| `.segments` | Array of path segments | `["src", "main.go"]` |
| `.absolute` | Whether path is absolute | `true` or `false` |
| `.basename` | Filename | `"config.json"` |
| `.ext` | Extension | `"json"` |
| `.stem` | Name without ext | `"config"` |
| `.dirname` | Parent directory | Path object |
| `.dir` | Parent directory as string | `"./data"` |
| `.string` | Full path as string | `"./data/config.json"` |

```parsley
let p = @/usr/local/bin
p.segments   // ["usr", "local", "bin"]
p.absolute   // true

let p2 = @./src/main.go
p2.segments  // [".", "src", "main.go"]
p2.absolute  // false
```

### Methods

| Method | Description |
|--------|-------------|
| `.isAbsolute()` | Is absolute path |
| `.isRelative()` | Is relative path |
| `.toDict()` | Dictionary form |

### String Conversion

Paths convert to their path string in templates:
```parsley
let p = @./src/main.go
"{p}"              // "./src/main.go"
log(p)             // ./src/main.go
```

---

## URL Methods

### Creation

```parsley
@https://api.example.com/users    // URL
url("https://example.com:8080")   // Dynamic URL
```

### Interpolated URL Templates

Use `@(...)` syntax for URLs with embedded expressions:
```parsley
version = "v2"
u = @(https://api.example.com/{version}/users)
u.string  // "https://api.example.com/v2/users"

host = "api.test.com"
u = @(https://{host}/data)
u.string  // "https://api.test.com/data"

// Port interpolation
port = 8080
u = @(http://localhost:{port}/api)
u.port    // "8080"

// Fragment interpolation
section = "intro"
u = @(https://docs.com/guide#{section})
u.fragment  // "intro"
```

Static URL literals (`@https://...`) remain unchanged and don't require parentheses.

### Properties

| Property | Description |
|----------|-------------|
| `.scheme` | Protocol ("https") |
| `.host` | Hostname |
| `.port` | Port number |
| `.path` | Path component |
| `.query` | Query parameters dict |
| `.string` | Full URL as string |

### Methods

| Method | Description |
|--------|-------------|
| `.origin()` | Scheme + host + port |
| `.pathname()` | Path only |
| `.search()` | Query string with `?` |
| `.href()` | Full URL string |
| `.toDict()` | Dictionary form |

```parsley
let u = @https://example.com?q=test&page=2
u.query.q      // "test"
u.query.page   // "2"
```

### String Conversion

URLs convert to their full URL string in templates:

```parsley
let u = @https://api.example.com/v1
"{u}"              // "https://api.example.com/v1"
log(u)             // https://api.example.com/v1
```

---

## File I/O

### File Handle Factories

| Factory | Format | Read Returns | Write Accepts |
|---------|--------|--------------|---------------|
| `file(path)` | Auto-detect | Depends on ext | String |
| `JSON(path)` | JSON | Dict or Array | Dict or Array |
| `CSV(path)` | CSV | Array of Dicts | Array of Dicts |
| `MD(path)` | Markdown | Dict (html + frontmatter) | String |
| `SVG(path)` | SVG | String (prolog stripped) | String |
| `lines(path)` | Lines | Array of Strings | Array of Strings |
| `text(path)` | Text | String | String |
| `bytes(path)` | Binary | Byte Array | Byte Array |

### File Handle Properties

| Property | Description |
|----------|-------------|
| `.exists` | File exists |
| `.size` | Size in bytes |
| `.modified` | Last modified datetime |
| `.isFile` | Is a file |
| `.isDir` | Is a directory |
| `.ext` | File extension |
| `.basename` | Filename |
| `.stem` | Name without extension |

### File Handle Methods

| Method | Description |
|--------|-------------|
| `.remove()` | Removes/deletes the file from the filesystem. Returns `null` on success, error on failure. |
| `.mkdir(options?)` | Creates a directory. Options: `{parents: true}` to create parent directories. |
| `.rmdir(options?)` | Removes a directory. Options: `{recursive: true}` to remove with contents. |

```parsley
// Remove a file
let f = file(@./temp.txt)
f.remove()  // Deletes the file

// Create directories
file(@./new-dir).mkdir()
file(@./parent/child).mkdir({parents: true})

// Remove directories
file(@./empty-dir).rmdir()
file(@./dir-tree).rmdir({recursive: true})

// With error handling
let result = f.remove()
if (result != null) {
    log("Error:", result)
}
```

### Reading (`<==`)

```parsley
let config <== JSON(@./config.json)
let rows <== CSV(@./data.csv)
let content <== text(@./readme.txt)

// Load SVG icons as reusable components
let Arrow <== SVG(@./icons/arrow.svg)
<button><Arrow/> Next</button>

// Load markdown with YAML frontmatter
let post <== MD(@./blog.md)
post.title       // From frontmatter
post.date        // Parsed as DateTime if ISO format
post.tags        // Array from frontmatter
post.html        // Rendered HTML
post.raw         // Original markdown body

// Destructure from file
let {name, version} <== JSON(@./package.json)

// Error capture pattern
let {data, error} <== JSON(@./config.json)
if (error) {
    log("Error:", error)
}

// Fallback
let config <== JSON(@./config.json) ?? {defaults: true}
```

### Writing (`==>`)

```parsley
myDict ==> JSON(@./output.json)
records ==> CSV(@./export.csv)
"Hello" ==> text(@./greeting.txt)
"<svg>...</svg>" ==> SVG(@./icon.svg)
```

### Appending (`==>>`)

```parsley
newLine ==>> lines(@./log.txt)
message ==>> text(@./debug.log)
```

### Stdin/Stdout/Stderr

Read from stdin and write to stdout/stderr for Unix pipeline integration.

**Syntax:**
- `@-` - Unix convention: stdin for reads, stdout for writes
- `@stdin` - Explicit stdin reference
- `@stdout` - Explicit stdout reference  
- `@stderr` - Explicit stderr reference

```parsley
// Read JSON from stdin
let data <== JSON(@-)

// Write JSON to stdout
data ==> JSON(@-)

// Using explicit aliases
let input <== text(@stdin)
"output" ==> text(@stdout)
"error" ==> text(@stderr)

// Works with all format factories
let lines <== lines(@-)
let csvData <== CSV(@stdin)
data ==> YAML(@stdout)

// Full pipeline example: filter active items
let input <== JSON(@-)
let active = for (item in input.items) {
    if (item.active) { item }
}
active ==> JSON(@-)
```

**Error Handling:**

```parsley
// Cannot read from stdout/stderr
let data <== text(@stdout)  // ERROR: cannot read from stdout

// Cannot write to stdin
"text" ==> text(@stdin)     // ERROR: cannot write to stdin
```

### Directory Operations

```parsley
let d = dir(@./images)
d.exists      // true
d.isDir       // true
d.count       // Number of entries
d.files       // Array of file handles

// Read directory contents
let files <== dir(@./images)
```

### File Globbing

Use `files()` to find files matching a glob pattern. Returns an array of file/directory handles.

```parsley
// Find all .pars files in current directory
let parsFiles = files(@./*.pars)

// Find all images in a directory
let images = files(@./images/*.jpg)

// Using string patterns
let logs = files("./logs/*.log")

// Home directory expansion
let configs = files("~/.config/*.json")
```

**Glob Pattern Syntax:**

| Pattern | Matches |
|---------|---------|
| `*` | Any sequence of characters (not including `/`) |
| `?` | Any single character |
| `[abc]` | Any character in the set |
| `[a-z]` | Any character in the range |

**Working with Results:**

```parsley
// List all markdown files
let docs = files(@./docs/*.md)
for(doc in docs) {
    log(doc.name)  // Print each filename
}

// Read and process all JSON configs
let configs = files(@./config/*.json)
for(config in configs) {
    let data <== config
    log(config.name, ":", data)
}

// Filter files by property
let bigFiles = filter(fn(f) { f.size > 1000000 }, files(@./data/*))
```

**Note:** Standard glob patterns work (`*`, `?`, `[...]`). For recursive directory traversal, use `dir()` with iteration instead of `**` patterns.

---

## SFTP (Network File Operations)

### SFTP Connection

Create SFTP connections for secure file transfer over SSH.

```parsley
// Password authentication
let conn = SFTP("sftp://user:password@example.com/")

// SSH key authentication (preferred)
let conn = SFTP("sftp://user@example.com/", {
    key: @~/.ssh/id_rsa,
    passphrase: "optional-key-passphrase"
})

// Custom port and timeout
let conn = SFTP("sftp://user:password@example.com:2222/", {
    timeout: @10s
})
```

**Connection Features:**

- Connection caching by `user@host:port` for efficiency
- `known_hosts` verification for security (~/.ssh/known_hosts)
- Supports SSH keys (recommended) and password authentication
- Automatic reconnection on network errors
- `.close()` method to free resources

### SFTP File Operations

SFTP uses the same network operators as HTTP/database operations:

| Operator | Purpose | Example |
|----------|---------|---------|
| `<=/=` | Read from SFTP | `data <=/= conn(@/file.json).json` |
| `=/=>` | Write to SFTP | `data =/=> conn(@/file.json).json` |
| `=/=>>` | Append to SFTP | `line =/=>> conn(@/log.txt).lines` |

**Callable Syntax:**

```parsley
let conn = SFTP("sftp://user:pass@host/")
let handle = conn(@/path/to/file.txt)  // Returns file handle
```

### Format Support

All file formats work over SFTP:

| Format | Accessor | Read Returns | Write Accepts |
|--------|----------|--------------|---------------|
| JSON | `.json` | Dict/Array | Dict/Array |
| Text | `.text` | String | String |
| CSV | `.csv` | Array | Array |
| Lines | `.lines` | Array[String] | Array[String] |
| Bytes | `.bytes` | Array[Int] | Array[Int] |
| Auto | `.file` | Auto-detect | String |
| Directory | `.dir` | Array[FileInfo] | N/A |

### Reading from SFTP

```parsley
let conn = SFTP("sftp://user:password@example.com/")

// Read JSON file
{data, error} <=/= conn(@/data/config.json).json
if (!error) {
    log("Config:", data.name, data.version)
}

// Read text file
{content, readErr} <=/= conn(@/logs/app.log).text
if (!readErr) {
    log("Log size:", content.length())
}

// Read CSV
{rows, csvErr} <=/= conn(@/reports/sales.csv).csv
if (!csvErr) {
    for (row in rows) {
        log("Row:", row)
    }
}

// Read as lines
{lines, linesErr} <=/= conn(@/data/list.txt).lines
if (!linesErr) {
    for (line in lines) {
        log(line)
    }
}

// Read binary file
{bytes, bytesErr} <=/= conn(@/images/icon.png).bytes
if (!bytesErr) {
    log("Image size:", bytes.length(), "bytes")
}

// Auto-detect format
{fileData, fileErr} <=/= conn(@/data/unknown.json).file
```

### Writing to SFTP

```parsley
let conn = SFTP("sftp://user:password@example.com/")

// Write JSON
let config = {app: "MyApp", version: "1.0.0"}
writeErr = config =/=> conn(@/config/app.json).json
if (!writeErr) {
    log("Config saved")
}

// Write text
textErr = "Hello, SFTP!" =/=> conn(@/messages/hello.txt).text

// Write lines
let logs = ["Line 1", "Line 2", "Line 3"]
linesErr = logs =/=> conn(@/logs/output.log).lines

// Write bytes
let data = [137, 80, 78, 71, 13, 10, 26, 10]  // PNG signature
bytesErr = data =/=> conn(@/images/test.png).bytes
```

### Appending to SFTP Files

```parsley
let conn = SFTP("sftp://user:password@example.com/")

// Append text
appendErr = "New log entry\n" =/=>> conn(@/logs/app.log).text
if (!appendErr) {
    log("Entry appended")
}

// Append line
lineErr = "Additional line" =/=>> conn(@/data/list.txt).lines
```

### Directory Operations

```parsley
let conn = SFTP("sftp://user:password@example.com/")

// List directory with file metadata
{files, dirErr} <=/= conn(@/uploads).dir
if (!dirErr) {
    for (file in files) {
        log(file.name, "-", file.size, "bytes")
        log("  Modified:", file.modified)
        log("  Is directory:", file.isDir)
    }
}

// Create directory
mkdirErr = conn(@/data/archive).mkdir()
if (!mkdirErr) {
    log("Directory created")
}

// Create directory with permissions
mkdirErr = conn(@/data/secure).mkdir({mode: 0700})

// Remove empty directory
rmdirErr = conn(@/data/temp).rmdir()

// Delete file
removeErr = conn(@/data/old.txt).remove()
```

### Error Handling

```parsley
let conn = SFTP("sftp://user:password@example.com/")

// Pattern 1: Check error after operation
error = data =/=> conn(@/file.txt).text
if (error) {
    log("Write failed:", error)
}

// Pattern 2: Destructuring with error capture
{result, err} <=/= conn(@/data.json).json
if (err) {
    log("Read failed:", err)
} else {
    log("Data loaded:", result)
}

// Pattern 3: Default value on error
{data, fetchErr} <=/= conn(@/config.json).json
let settings = if (fetchErr) { {defaults: true} } else { data }
```

### Connection Management

```parsley
let conn = SFTP("sftp://user:password@example.com/")

// Perform operations...
data <=/= conn(@/file.json).json

// Close when done
conn.close()

// Attempting to use closed connection returns error
{data, err} <=/= conn(@/file.txt).text
// err will be set (connection not connected)
```

### Complete Example

```parsley
// Connect to SFTP server
let conn = SFTP("sftp://user@example.com/", {
    key: @~/.ssh/id_rsa,
    timeout: @10s
})

// Read and process data
{users, readErr} <=/= conn(@/data/users.json).json
if (!readErr) {
    // Transform data
    let activeUsers = users.filter(fn(u) { u.active })
    let processed = activeUsers.map(fn(u) {
        {
            id: u.id,
            name: u.name.toUpper(),
            lastSeen: @now
        }
    })
    
    // Write back to server
    writeErr = processed =/=> conn(@/data/active-users.json).json
    if (!writeErr) {
        log("Processed", processed.length(), "users")
    }
}

// Clean up
conn.close()
```

---

## Regex

### Literals

```parsley
/pattern/       // Basic regex
/pattern/i      // Case insensitive
/pattern/g      // Global
```

### Dynamic Creation

```parsley
regex("\\d+", "i")
```

### Methods

| Method | Description | Example |
|--------|-------------|---------|
| `.test(string)` | Test if matches | `/\d+/.test("abc123")` → `true` |
| `.format()` | Pattern only | `/\d+/i.format()` → `\d+` |
| `.format("literal")` | Literal form | `/\d+/i.format("literal")` → `/\d+/i` |
| `.format("verbose")` | Detailed form | `/\d+/i.format("verbose")` → `regex(\d+, i)` |
| `.toDict()` | Dictionary form | `/\d+/i.toDict()` → `{pattern: "\\d+", flags: "i", ...}` |

### String Conversion

Regex patterns convert to literal notation in templates:

```parsley
let r = /[a-z]+/i
"{r}"              // "/[a-z]+/i"
log(r)             // /[a-z]+/i
```

### Matching

```parsley
"test@example.com" ~ /\w+@\w+\.\w+/  // ["test@example.com"]
"hello" ~ /\d+/                       // null (no match)
"hello" !~ /\d+/                      // true
```

### Capture Groups

```parsley
let match = "Phone: (555) 123-4567" ~ /\((\d{3})\) (\d{3})-(\d{4})/
match[0]  // Full match
match[1]  // "555"
match[2]  // "123"
match[3]  // "4567"
```

### Replace and Split

```parsley
"hello world".replace(/world/, "Parsley")  // "hello Parsley"
"a1b2c3".split(/\d+/)                      // ["a", "b", "c"]
```

---

## HTTP Requests

Fetch content from URLs using the `<=/=` operator with request handles.

### Fetch Operator

| Operator | Description | Example |
|----------|-------------|---------|
| `<=/=` | Fetch from URL | `let data <=/= JSON(@https://api.example.com)` |

### Request Handle Factories

| Factory | Format | Returns |
|---------|--------|---------|
| `JSON(url)` | JSON | Parsed JSON (dict/array) |
| `text(url)` | Plain text | String |
| `YAML(url)` | YAML | Parsed YAML |
| `lines(url)` | Lines | Array of strings |
| `bytes(url)` | Binary | Array of integers |

### Basic Usage

```parsley
// Fetch JSON data
let users <=/= JSON(@https://api.example.com/users)
log(users[0].name)

// Fetch text content
let html <=/= text(@https://example.com)

// Direct URL fetch (defaults to text)
let content <=/= @https://example.com
```

### Request Options

Pass a second argument to customize the request:

```parsley
// POST with JSON body
let response <=/= JSON(@https://api.example.com/users, {
    method: "POST",
    body: {name: "Alice", email: "alice@example.com"},
    headers: {"Authorization": "Bearer token123"}
})

// Custom timeout (milliseconds)
let data <=/= JSON(@https://slow-api.com/data, {
    timeout: 10000  // 10 seconds
})

// PUT request
let updated <=/= JSON(@https://api.example.com/users/1, {
    method: "PUT",
    body: {name: "Bob"},
    headers: {"Content-Type": "application/json"}
})
```

### Error Handling

Use destructuring to capture errors and response metadata:

```parsley
// Basic error capture
let {data, error} <=/= JSON(@https://api.example.com/data)
if (error != null) {
    log("Fetch failed:", error)
} else {
    log("Success:", data)
}

// Access HTTP status and headers
let {data, error, status, headers} <=/= JSON(@https://api.example.com/users)
log("Status code:", status)
log("Content-Type:", headers["Content-Type"])

// Handle errors gracefully
let {data, error} <=/= JSON(@https://unreliable-api.com/data)
let users = data ?? []  // Default to empty array on error
```

### HTTP Methods

Supported methods: GET (default), POST, PUT, PATCH, DELETE, HEAD, OPTIONS

```parsley
// GET (default)
let data <=/= JSON(@https://api.example.com/items)

// POST
let created <=/= JSON(@https://api.example.com/items, {
    method: "POST",
    body: {title: "New Item"}
})

// DELETE
let {data, status} <=/= JSON(@https://api.example.com/items/123, {
    method: "DELETE"
})

// PATCH
let updated <=/= JSON(@https://api.example.com/items/123, {
    method: "PATCH",
    body: {title: "Updated Title"}
})
```

### Request Headers

Customize headers for authentication, content negotiation, etc.

**Note**: Parsley dictionary syntax requires identifier keys (no hyphens or special characters). For HTTP headers with hyphens like "Content-Type" or "User-Agent", you may need to work around this limitation or use simple header names.

```parsley
// Simple headers without hyphens work fine
let data <=/= JSON(@https://api.example.com/data, {
    headers: {
        Authorization: "Bearer " + apiToken
    }
})

// For headers requiring hyphens, consider alternative approaches
// or wait for future Parsley enhancements
```

### Response Structure

When using error capture pattern `{data, error, status, headers}`:

| Field | Type | Description |
|-------|------|-------------|
| `data` | Varies | Parsed response body (based on format) |
| `error` | String/Null | Error message if request failed, `null` on success |
| `status` | Integer | HTTP status code (200, 404, 500, etc.) |
| `headers` | Dictionary | Response HTTP headers |

```parsley
let {data, error, status, headers} <=/= JSON(@https://api.example.com/data)

if (status == 200) {
    log("Success!")
} else if (status == 404) {
    log("Not found")
} else if (status >= 500) {
    log("Server error")
}
```

### Practical Examples

**API Integration:**
```parsley
// Fetch and process API data
let {data, error} <=/= JSON(@https://api.github.com/users/octocat)
if (error == null) {
    log("User: " + data.login)
    log("Repos: " + data.public_repos)
}
```

**Form Submission:**

```parsley
let formData = {
    username: "alice",
    password: "secret123"
}

let {data, error, status} <=/= JSON(@https://example.com/login, {
    method: "POST",
    body: formData,
    headers: {"Content-Type": "application/json"}
})

if (status == 200) {
    log("Login successful!")
} else {
    log("Login failed:", error)
}
```

**Download Text Content:**

```parsley
let {data, error} <=/= text(@https://raw.githubusercontent.com/user/repo/main/README.md)
if (error == null) {
    data ==> text(@./downloaded_readme.md)
}
```

**Multiple API Calls:**

```parsley
let users <=/= JSON(@https://api.example.com/users)
let posts <=/= JSON(@https://api.example.com/posts)

for (user in users) {
    let userPosts = posts.filter(fn(p) { p.userId == user.id })
    log(user.name + " has " + userPosts.length() + " posts")
}
```

### Best Practices

1. **Always handle errors** - Use `{data, error}` pattern for robust code
2. **Set reasonable timeouts** - Default is 30 seconds, adjust as needed
3. **Check status codes** - Don't assume 200 OK, verify response status
4. **Use appropriate formats** - JSON for APIs, text for HTML, bytes for binary
5. **Secure credentials** - Never hardcode API keys, use environment variables

```parsley
// Good: Error handling and timeout
let {data, error, status} <=/= JSON(@https://api.example.com/data, {
    timeout: 5000,
    headers: {"Authorization": "Bearer " + getToken()}
})

if (error != null) {
    log("Request failed:", error)
} else if (status >= 400) {
    log("HTTP error:", status)
} else {
    // Process data
    log("Success:", data)
}
```

---

## Database

Parsley provides first-class support for SQLite databases with clean, expressive operators.

### Database Operators

| Operator | Description | Returns |
|----------|-------------|---------|
| `<=?=>` | Query single row | Dictionary or `null` |
| `<=??=>` | Query multiple rows | Array of dictionaries |
| `<=!=>` | Execute mutation | `{affected, lastId}` |

### Connection Factory

```parsley
// SQLite (only supported driver currently)
let db = SQLITE(":memory:")           // In-memory database
let db = SQLITE(@./data.db)           // File-based database
let db = SQLITE("/path/to/data.db")   // String path also works
```

### Querying Data

#### Single Row Query (`<=?=>`)

Returns a dictionary if a row is found, or `null` if no match:

```parsley
let user = db <=?=> "SELECT * FROM users WHERE id = 1"
// Returns: {id: 1, name: "Alice", email: "alice@example.com"} or null

// Using with conditional
if (user) {
    log("Found user: {user.name}")
} else {
    log("User not found")
}

// With nullish coalescing
let user = db <=?=> "SELECT * FROM users WHERE id = 999" ?? {name: "Guest"}
```

#### Multiple Row Query (`<=??=>`)

Returns an array of dictionaries (empty array if no matches):

```parsley
let users = db <=??=> "SELECT * FROM users WHERE age > 25"
// Returns: [{id: 1, name: "Alice", age: 30}, {id: 2, name: "Bob", age: 28}]

// Iterate over results
for (user in users) {
    log("{user.name}: {user.email}")
}

// Get count
let count = len(users)
```

### Executing Mutations (`<=!=>`)

Execute INSERT, UPDATE, DELETE, or DDL statements:

```parsley
// CREATE TABLE
let _ = db <=!=> "CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE,
    age INTEGER
)"

// INSERT
let result = db <=!=> "INSERT INTO users (name, email, age) VALUES ('Alice', 'alice@example.com', 30)"
// Returns: {affected: 1, lastId: 1}

log("Inserted {result.affected} row(s), last ID: {result.lastId}")

// UPDATE
let result = db <=!=> "UPDATE users SET age = 31 WHERE id = 1"
// Returns: {affected: 1, lastId: 1}

// DELETE
let result = db <=!=> "DELETE FROM users WHERE id = 5"
// Returns: {affected: 1, lastId: 5}
```

### Transactions

```parsley
// Begin transaction
db.begin()

// Execute multiple statements
let _ = db <=!=> "INSERT INTO users (name) VALUES ('Alice')"
let _ = db <=!=> "INSERT INTO posts (user_id, title) VALUES (1, 'First Post')"

// Commit or rollback
if (someCondition) {
    db.commit()     // Returns true on success
} else {
    db.rollback()   // Returns true on success
}

// Check transaction status
if (db.inTransaction) {
    log("Still in transaction")
}
```

### Connection Methods

| Method | Returns | Description |
|--------|---------|-------------|
| `db.ping()` | Boolean | Test if connection is alive |
| `db.begin()` | Boolean | Start transaction |
| `db.commit()` | Boolean | Commit transaction |
| `db.rollback()` | Boolean | Rollback transaction |
| `db.close()` | Null | Close connection |

### Connection Properties

```parsley
db.type           // "sqlite"
db.connected      // true/false
db.inTransaction  // true/false
db.lastError      // Error message string or empty
```

### Data Type Mapping

SQLite types are automatically converted to Parsley types:

| SQLite Type | Parsley Type | Example |
|-------------|--------------|---------|
| INTEGER | Integer | `42` |
| REAL | Float | `3.14` |
| TEXT | String | `"hello"` |
| BLOB | String | (converted to string) |
| NULL | Null | `null` |

### Working with NULL Values

NULL database values are represented as `null` in Parsley:

```parsley
let user = db <=?=> "SELECT name, age FROM users WHERE id = 1"
// If age is NULL in database: {name: "Alice", age: null}

if (user.age == null) {
    log("Age not set")
}

// Use nullish coalescing for defaults
let age = user.age ?? 0
```

### Error Handling

Database errors are returned as Parsley errors:

```parsley
// Syntax error
let result = db <=!=> "INVALID SQL"
// Returns: ERROR: near "INVALID": syntax error

// Table doesn't exist
let users = db <=??=> "SELECT * FROM nonexistent"
// Returns: ERROR: no such table: nonexistent

// Check last error
if (db.lastError != "") {
    log("Database error: {db.lastError}")
}
```

### Complete Example

```parsley
// Create database
let db = SQLITE(@./app.db)

// Set up schema
let _ = db <=!=> "CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    created_at INTEGER DEFAULT (strftime('%s', 'now'))
)"

// Insert data
let result = db <=!=> "INSERT INTO users (name, email) VALUES ('Alice', 'alice@example.com')"
log("Created user with ID: {result.lastId}")

// Query single user
let user = db <=?=> "SELECT * FROM users WHERE email = 'alice@example.com'"

if (user) {
    log("Welcome back, {user.name}!")
    
    // Update
    let _ = db <=!=> "UPDATE users SET name = 'Alice Smith' WHERE id = {user.id}"
    
    // Query all users
    let allUsers = db <=??=> "SELECT name, email FROM users ORDER BY created_at DESC"
    
    for (u in allUsers) {
        log("{u.name} <{u.email}>")
    }
}

// Close when done
db.close()
```

### Best Practices

1. **Use single-operator syntax**: `let result = db <=!=> query` (not double-operator)
2. **Handle NULL values**: Always check for `null` in query results
3. **Use transactions for multiple operations**: Ensures data consistency
4. **Close connections**: Call `db.close()` when done (especially for file-based DBs)
5. **Check errors**: Use `db.lastError` or handle ERROR returns
6. **Avoid SQL injection**: Future versions will support parameterized queries

---

## Process Execution

Execute external commands and capture their output.

### Creating a Command

Use `COMMAND()` to create a command handle:

```parsley
// Simple command
let cmd = COMMAND("echo")

// Command with arguments
let cmd = COMMAND("ls", ["-la", "/tmp"])

// Command with options
let cmd = COMMAND("node", ["script.js"], {
    env: {NODE_ENV: "production"},
    dir: "/path/to/project",
    timeout: @30s
})
```

### Command Options

| Option | Type | Description |
|--------|------|-------------|
| `env` | Dictionary | Environment variables (merged with system env) |
| `dir` | String/Path | Working directory for command execution |
| `timeout` | Duration | Maximum execution time (process killed if exceeded) |

### Executing Commands

Use the `<=#=>` operator to execute a command:

```parsley
// Execute without input
let result = COMMAND("echo", ["hello"]) <=#=> null

// Command can also have input data (passed to stdin)
let result = COMMAND("cat") <=#=> "input data"
```

### Result Structure

Execution returns a dictionary with:

| Field | Type | Description |
|-------|------|-------------|
| `stdout` | String | Standard output from command |
| `stderr` | String | Standard error from command |
| `exitCode` | Integer | Exit code (0 for success) |
| `error` | String/Null | Error message if execution failed, `null` otherwise |

### Examples

```parsley
// Basic command
let result = COMMAND("date") <=#=> null
log("Current date:", result.stdout)
log("Exit code:", result.exitCode)

// Command with arguments
let result = COMMAND("ls", ["-la", "/tmp"]) <=#=> null
if (result.exitCode == 0) {
    log("Files:")
    log(result.stdout)
}

// Command with custom environment
let cmd = COMMAND("printenv", ["MY_VAR"], {
    env: {MY_VAR: "custom value"}
})
let result = cmd <=#=> null
log("Environment variable:", result.stdout)

// Command with working directory
let result = COMMAND("pwd", [], {dir: "/tmp"}) <=#=> null
log("Current directory:", result.stdout)

// Command with timeout
let result = COMMAND("sleep", ["60"], {timeout: @5s}) <=#=> null
if (result.error != null) {
    log("Command timed out or failed:", result.error)
}
```

### Security

Process execution requires explicit permission via command-line flags:

```bash
# Allow all process execution
./pars --allow-execute-all script.pars
./pars -x script.pars

# Allow execution from specific directories
./pars --allow-execute=/usr/bin,/bin script.pars
```

Without these flags, `COMMAND()` will return a security error.

### Error Handling

```parsley
// Command doesn't exist
let result = COMMAND("nonexistent_cmd") <=#=> null
if (result.error != null) {
    log("Error:", result.error)
}

// Non-zero exit code
let result = COMMAND("ls", ["/nonexistent"]) <=#=> null
if (result.exitCode != 0) {
    log("Command failed with code:", result.exitCode)
    log("Error output:", result.stderr)
}
```

---

## Modules

### Creating a Module

```parsley
// math.pars

// Exported values (visible when imported)
export let PI = 3.14159
export add = fn(a, b) { a + b }
export Logo = <img src="logo.png" alt="Logo"/>

// Private (no 'export')
helper = fn(x) { x * 2 }

// 'let' without 'export' is also exported (for backward compatibility)
let multiply = fn(a, b) { a * b }
```

### Importing

**New syntax (recommended):**

```parsley
// Import with auto-binding (binds to last path segment)
import @std/math       // binds to 'math'
import @./utils.pars   // binds to 'utils'

math.floor(3.7)        // 3
utils.helper()

// Import with alias
import @std/math as M
M.PI  // 3.14159

// Destructuring import
{floor, ceil} = import @std/math
floor(3.7)  // 3
ceil(3.2)   // 4

// Destructure with rename
{floor as f, PI} = import @std/math
f(3.7)  // 3
PI      // 3.14159
```

**Old syntax (still supported for backward compatibility):**

```parsley
let math = import("std/math")
let {add, PI} = import(@./math.pars)
```

### Standard Library Imports

```parsley
import @std/math        // Math functions (floor, ceil, PI, etc.)
import @std/strings     // String utilities (split, join, etc.)
import @std/table       // Table data structure
import @std/id          // ID generation (uuid, nanoid, etc.)
import @std/valid       // Validation functions
import @std/schema      // Schema validation
```

### Local Imports

```parsley
import @./components/Button     // Relative to current file
import @../shared/utils         // Parent directory
```

---

## Tags

### HTML/XML Tags

```parsley
<div class="container">
    <h1>{title}</h1>
    <p>{content}</p>
</div>
```

### Self-Closing

```parsley
<br/>
<img src="photo.jpg" />
<meta charset="utf-8" />
```

### Components

```parsley
let Card = fn({title, body}) {
    <div class="card">
        <h2>{title}</h2>
        <p>{body}</p>
    </div>
}

<Card title="Hello" body="World" />
```

### Fragments

```parsley
<>
    <p>First</p>
    <p>Second</p>
</>
```

### Raw Mode (Style/Script)

Inside `<style>` and `<script>` tags, use `@{}` for interpolation:
```parsley
let color = "blue"
<style>.class { color: @{color}; }</style>
```

### Programmatic Tags

```parsley
tag("div", {class: "box"}, "content")
// Creates tag dictionary, use toString() to render
```

---

## Error Handling

### The `try` Expression

The `try` expression catches runtime errors from function and method calls, returning a dictionary with `result` and `error` fields instead of halting execution.

**Grammar:**

```
try_expr ::= "try" call_expr
call_expr ::= identifier "(" arguments? ")"
            | expression "." identifier "(" arguments? ")"
```

**Syntax:**

```parsley
let response = try functionCall()
let {result, error} = try obj.method()
```

**Return Value:**

On success:
```parsley
{result: <return_value>, error: null}
```

On catchable error:
```parsley
{result: null, error: "Error message"}
```

**Example:**

```parsley
// Try a function that might fail
let {result, error} = try url("not a valid url")

if (error != null) {
    log("Failed to parse URL: {error}")
} else {
    log("Parsed URL: {result}")
}

// Using null coalescing for defaults
let parsed = (try time("maybe-invalid")).result ?? now()
```

### Catchable vs Non-Catchable Errors

Not all errors can be caught. The `try` expression only catches "user errors" - failures from external factors that cannot be validated in advance.

**Catchable Errors** (caught by `try`):
| Class | Description | Example |
|-------|-------------|---------|
| IO | File operations | File not found, permission denied |
| Network | HTTP, SFTP | Connection refused, timeout |
| Database | SQL operations | Query syntax error, connection lost |
| Format | Parsing/conversion | Invalid JSON, malformed URL |
| Value | Invalid values | Empty required field |
| Security | Access control | Path traversal attempt |

**Non-Catchable Errors** (always halt execution):
| Class | Description | Why Not Catchable |
|-------|-------------|-------------------|
| Type | Type mismatch | Validate with `typeof()` |
| Arity | Wrong argument count | Code structure issue |
| Undefined | Name not found | Spelling error |
| Index | Out of bounds | Check `array.length()` first |
| Operator | Invalid operation | Code logic error |
| Parse | Syntax error | Fix the code |
| Internal | Interpreter bug | Report as bug |

**Design Philosophy:**

> If you can check before calling, you should. If external factors can fail unexpectedly, `try` catches it.

**Examples:**

```parsley
// These CAN be caught (external failures)
let {result, error} = try url("user-provided-input")
let {result, error} = try time("user-date-string")
let {result, error} = try file.read()

// These CANNOT be caught (fix your code instead)
try url(123)        // Type error - url() expects string
try time()          // Arity error - missing argument
try undefined()     // Undefined error - function doesn't exist
```

### The `fail()` Function

The `fail()` function creates a user-defined catchable error. This allows your functions to participate in the `try` error handling system.

**Grammar:**

```
fail_call ::= "fail" "(" string_expr ")"
```

**Syntax:**

```parsley
fail("error message")
```

**Behavior:**

- Creates a `Value`-class error (catchable by `try`)
- If not caught by `try`, halts execution with the error message
- Error code is always `USER-0001`

**Example:**

```parsley
// Create a validation function that can fail
let validateEmail = fn(email) {
  if (!email.contains("@")) {
    fail("Invalid email: missing @ symbol")
  }
  if (!email.contains(".")) {
    fail("Invalid email: missing domain")
  }
  email
}

// Caller uses try to handle potential failure
let {result, error} = try validateEmail(userInput)
if (error != null) {
  log("Validation failed: {error}")
} else {
  log("Valid email: {result}")
}
```

**Automatic Propagation:**

Errors from `fail()` propagate through the call stack until caught by `try`:

```parsley
let step1 = fn(x) { if (x < 0) { fail("negative") }; x }
let step2 = fn(x) { step1(x) * 2 }
let step3 = fn(x) { step2(x) + 1 }

// Catches error from step1, even though we called step3
let {result, error} = try step3(-5)
// error = "negative"
```

**Type Requirements:**

`fail()` requires a string argument. Non-strings produce a Type error:

```parsley
fail("message")     // ✓ OK
fail(123)           // ✗ Type error
fail(null)          // ✗ Type error
```

---

## Utility Functions

### Type Conversion

| Function | Description |
|----------|-------------|
| `toInt(str)` | String to integer |
| `toFloat(str)` | String to float |
| `toNumber(str)` | Auto-detect int/float |
| `toString(value)` | Convert to string |

### Path Pattern Matching

The `match(path, pattern)` function extracts named parameters from URL paths using Express-style patterns.

| Function | Description |
|----------|-------------|
| `match(path, pattern)` | Match path against pattern, returns dict or `null` |

**Pattern Syntax:**
- `:name` — Captures a single segment as a string
- `*name` — Captures remaining segments as an array (glob)
- Literal segments must match exactly (case sensitive)

```parsley
// Basic parameter extraction
let params = match("/users/123", "/users/:id")
// → {id: "123"}

// Multiple parameters
let params = match("/users/42/posts/99", "/users/:userId/posts/:postId")
// → {userId: "42", postId: "99"}

// Glob capture (rest of path as array)
let params = match("/files/docs/2025/report.pdf", "/files/*path")
// → {path: ["docs", "2025", "report.pdf"]}

// No match returns null
let params = match("/posts/123", "/users/:id")
// → null
```

**Pattern Examples:**

| Pattern | Path | Result |
|---------|------|--------|
| `/users` | `/users` | `{}` |
| `/users/:id` | `/users/123` | `{id: "123"}` |
| `/users/:id/posts` | `/users/42/posts` | `{id: "42"}` |
| `/:a/:b/:c` | `/x/y/z` | `{a: "x", b: "y", c: "z"}` |
| `/files/*path` | `/files/a/b/c` | `{path: ["a", "b", "c"]}` |
| `/*all` | `/any/thing` | `{all: ["any", "thing"]}` |

**Route Dispatch Pattern:**

```parsley
let path = basil.http.request.path
let method = basil.http.request.method

if (let params = match(path, "/users/:id")) {
    if (method == "GET") { showUser(params.id) }
    else if (method == "DELETE") { deleteUser(params.id) }
}
else if (let params = match(path, "/users")) {
    if (method == "GET") { listUsers() }
    else if (method == "POST") { createUser() }
}
else if (let params = match(path, "/files/*path")) {
    serveFile(params.path)
}
else {
    let api = import("std/api")
    api.notFound("Page not found")
}
```

**Edge Cases:**
- Trailing slashes are normalized: `/users/123/` matches `/users/:id`
- Empty segments don't match: `/users/` doesn't match `/users/:id`
- Case sensitive: `/Users/123` doesn't match `/users/:id`
- Extra segments fail: `/users/123/extra` doesn't match `/users/:id`

### Debugging

| Function | Description |
|----------|-------------|
| `log(...)` | Output to stdout |
| `logLine(...)` | Output with file:line prefix |
| `toDebug(value)` | Debug representation |
| `repr(value)` | Dictionary representation of pseudo-types |

### The `repr()` Function

The `repr()` function returns a detailed dictionary representation of pseudo-types (datetime, duration, regex, path, url, file, dir, request). This is useful for debugging and introspection:

```parsley
let d = @1d2h30m
repr(d)    // {__type: "duration", days: 1, hours: 2, minutes: 30, ...}

let r = /\w+/i
repr(r)    // {__type: "regex", pattern: "\\w+", flags: "i"}

let p = @./src/main.go
repr(p)    // {__type: "path", path: "./src/main.go", basename: "main.go", ...}
```

For regular values, `repr()` returns them unchanged.

### The `toDict()` Method

All pseudo-types support a `.toDict()` method that returns their internal dictionary representation:

```parsley
@2024-12-25.toDict()    // {__type: "datetime", year: 2024, month: 12, day: 25, ...}
@1h30m.toDict()         // {__type: "duration", hours: 1, minutes: 30, ...}
/\d+/g.toDict()         // {__type: "regex", pattern: "\\d+", flags: "g"}
@./config.json.toDict() // {__type: "path", path: "./config.json", ...}
```

### Format Conversion Functions

#### JSON Functions

**`parseJSON(string)`**

Parse a JSON string into Parsley objects:

```parsley
let jsonStr = "{\"name\":\"Alice\",\"age\":30}"
let obj = parseJSON(jsonStr)
log(obj.name)  // Alice
log(obj.age)   // 30

// Arrays
let arr = parseJSON("[1, 2, 3]")
log(arr[0])    // 1

// Nested structures
let data = parseJSON("{\"users\":[{\"id\":1,\"name\":\"Bob\"}]}")
log(data.users[0].name)  // Bob
```

**`stringifyJSON(object)`**

Convert Parsley objects to JSON string:

```parsley
let obj = {name: "Alice", age: 30, active: true}
let json = stringifyJSON(obj)
log(json)  // {"active":true,"age":30,"name":"Alice"}

// Arrays
let arr = [1, 2, 3]
log(stringifyJSON(arr))  // [1,2,3]

// Nested objects
let data = {user: {id: 1, name: "Bob"}, tags: ["a", "b"]}
log(stringifyJSON(data))
```

Supported types: dictionaries, arrays, strings, integers, floats, booleans, null.

#### CSV Functions

**`parseCSV(string, options?)`**

Parse CSV string into array of arrays or dictionaries:

```parsley
// Basic parsing (array of arrays)
let csv = "a,b,c\n1,2,3\n4,5,6"
let rows = parseCSV(csv)
log(rows)  // [["a","b","c"], ["1","2","3"], ["4","5","6"]]

// Parse with header (array of dictionaries)
let csv = "name,age,city\nAlice,30,NYC\nBob,25,LA"
let people = parseCSV(csv, {header: true})
log(people[0].name)   // Alice
log(people[1].city)   // LA

for (person in people) {
    log("{person.name} is {person.age} years old")
}
```

**`stringifyCSV(array)`**

Convert array of arrays to CSV string:

```parsley
let data = [
    ["Name", "Age", "City"],
    ["Alice", "30", "NYC"],
    ["Bob", "25", "LA"]
]
let csv = stringifyCSV(data)
log(csv)
// Output:
// Name,Age,City
// Alice,30,NYC
// Bob,25,LA
```

#### Practical Examples

**JSON API Response Processing:**

```parsley
// Simulate fetching JSON from API
let response = parseJSON("{\"users\":[{\"id\":1,\"name\":\"Alice\"}]}")
for (user in response.users) {
    log("User #{user.id}: {user.name}")
}

// Create JSON for API request
let request = {
    method: "POST",
    data: {username: "alice", email: "alice@example.com"}
}
let jsonRequest = stringifyJSON(request)
```

**CSV Data Processing:**

```parsley
// Read CSV with header
let csvData = "product,price,quantity\nApple,1.50,100\nBanana,0.75,200"
let inventory = parseCSV(csvData, {header: true})

// Calculate total value
let total = 0
for (item in inventory) {
    let value = parseFloat(item.price) * parseInt(item.quantity)
    total = total + value
}
log("Total inventory value: ${total}")

// Export to CSV
let report = [
    ["Product", "Value"],
    ["Apple", "150.00"],
    ["Banana", "150.00"]
]
let csvOutput = stringifyCSV(report)
```

---

## Method Chaining

Methods return appropriate types, enabling fluent chains:

```parsley
"  hello world  ".trim().toUpper().split(" ")  // ["HELLO", "WORLD"]
[3, 1, 2].sort().reverse()                   // [3, 2, 1]
[1, 2, 3].map(fn(x) { x * 2 }).reverse()     // [6, 4, 2]
```

## Null Propagation

Methods called on null return null instead of erroring:

```parsley
let d = {a: 1}
d.b.toUpper()              // null (d.b is null)
d.b.split(",").reverse() // null (entire chain)
```

---

## Security

Parsley provides file system access control through command-line flags. By default, write and execute operations are restricted for security.

### Security Model

| Operation | Default Behavior | Override Flags |
|-----------|-----------------|----------------|
| **Read** | ✅ Allowed | `--restrict-read=PATHS`, `--no-read` |
| **Write** | ❌ Denied | `--allow-write=PATHS`, `-w` |
| **Execute** | ❌ Denied | `--allow-execute=PATHS`, `-x` |

### Command-Line Flags

#### Read Control

```bash
--restrict-read=PATHS    # Blacklist: deny reading from paths
--no-read                # Deny all file reads
```

**Examples:**

```bash
# Prevent reading sensitive directories
./pars --restrict-read=/etc,/var script.pars

# stdin-only processing (no file reads)
./pars --no-read < data.json
```

#### Write Control

```bash
--allow-write=PATHS      # Whitelist: allow writes to specific paths
--allow-write-all        # Allow unrestricted writes (old behavior)
-w                       # Shorthand for --allow-write-all
```

**Examples:**

```bash
# Allow writes only to output directory
./pars --allow-write=./output build.pars

# Allow writes to multiple directories
./pars --allow-write=./data,./cache process.pars

# Development mode: unrestricted writes
./pars -w dev-script.pars
```

#### Execute Control

```bash
--allow-execute=PATHS    # Whitelist: allow imports from specific paths
--allow-execute-all      # Allow unrestricted module imports
-x                       # Shorthand for --allow-execute-all
```

**Examples:**

```bash
# Allow importing only from lib directory
./pars --allow-execute=./lib app.pars

# Allow imports from multiple directories
./pars --allow-execute=./lib,./modules app.pars

# Development mode: unrestricted imports
./pars -x dev-script.pars
```

### Path Resolution

All paths in security flags are:
- Resolved to absolute paths at startup
- Cleaned using filepath.Clean
- Applied to the directory and all subdirectories
- Support `~` for home directory expansion

```bash
# These are equivalent
./pars --allow-write=./output script.pars
./pars --allow-write=$(pwd)/output script.pars

# Home directory expansion
./pars --allow-write=~/Documents/output script.pars
```

### Combined Flags

Mix and match security flags for precise control:

```bash
# Static site generator: read freely, write to public
./pars --allow-write=./public build.pars

# API processor: restrict sensitive reads, write results, import libs
./pars --restrict-read=/etc --allow-write=./output --allow-execute=./lib process.pars

# Development: unrestricted writes and imports
./pars -w -x dev-script.pars

# Paranoid: specific write path, no reads, no imports
./pars --no-read --allow-write=./output template.pars
```

### Security Errors

When access is denied, clear error messages indicate the issue:

```
Error: security: file write not allowed: ./output/result.json (use --allow-write or -w)
Error: security: file read restricted: /etc/passwd
Error: security: script execution not allowed: ../tools/module.pars (use --allow-execute or -x)
```

### Migration from v0.9.x

**Breaking Changes in v0.10.0:**

- **Write operations** now denied by default
- **Module imports** (execute) now denied by default
- **Read operations** remain unrestricted (no change)

**Quick Fix:**

```bash
# Old (v0.9.x) - everything allowed
./pars build.pars

# New (v0.10.0) - add -w for old behavior
./pars -w build.pars

# Or specify allowed paths
./pars --allow-write=./output build.pars
```

### Protected Operations

The following operations are subject to security checks:

| Operation | Security Check | Example |
|-----------|----------------|---------|
| File read | `read` | `content <== text("file.txt")` |
| File write | `write` | `"data" ==> text("file.txt")` |
| File delete | `write` | `file("temp.txt").remove()` |
| Directory list | `read` | `dir("./folder").files` |
| Module import | `execute` | `import("./module.pars")` |

### Best Practices

1. **Production**: Use specific allow-lists
   ```bash
   ./pars --allow-write=./output --allow-execute=./lib app.pars
   ```

2. **Development**: Use shorthands for convenience
   ```bash
   ./pars -w -x dev-script.pars
   ```

3. **CI/CD**: Minimal permissions
   ```bash
   ./pars --allow-write=./dist build.pars
   ```

4. **Untrusted scripts**: Maximum restrictions
   ```bash
   ./pars --no-read --allow-write=./sandbox untrusted.pars
   ```

---

## Interactive REPL

Parsley includes an enhanced Read-Eval-Print Loop (REPL) for interactive development and testing.

### Starting the REPL

```bash
./pars              # Start interactive REPL
```

### Features

| Feature | Description | Keys |
|---------|-------------|------|
| **Cursor Movement** | Move within current line | ← → |
| **Command History** | Navigate previous commands | ↑ ↓ |
| **History Persistence** | Saved across sessions | `~/.parsley_history` |
| **Tab Completion** | Auto-complete keywords/builtins | Tab |
| **Multi-line Input** | Automatic detection of incomplete expressions | (automatic) |
| **Line Editing** | Standard editing shortcuts | Ctrl+A/E (home/end), Ctrl+K (kill) |
| **Abort Line** | Cancel current input | Ctrl+C |
| **Exit** | Quit REPL | Ctrl+D or `exit` |

### Tab Completion

Press Tab to auto-complete keywords and built-in functions. Completion words include:

**Keywords:** `let`, `if`, `else`, `for`, `in`, `fn`, `return`, `export`, `import`

**I/O Functions:** `log`, `logLine`, `file`, `dir`, `JSON`, `CSV`, `MD`, `SVG`, `text`, `lines`, `bytes`, `SFTP`, `Fetch`, `SQL`

**Collections:** `len`, `keys`, `values`, `type`, `sort`, `reverse`, `join`

**Strings:** `split`, `trim`, `toUpper`, `toLower`, `contains`, `startsWith`, `endsWith`, `replace`, `match`, `test`

**Math:** `abs`, `floor`, `ceil`, `round`, `sqrt`, `pow`, `sin`, `cos`, `tan`, `min`, `max`, `sum`

**DateTime:** `now`, `date`, `time`, `duration`, `format`, `parse`

**Other:** `range`, `glob`, `toString`, `true`, `false`, `null`

### Multi-line Input

The REPL automatically detects incomplete expressions (unclosed braces, brackets, or parentheses) and prompts for continuation:

```
>> let data = {
..     name: "Alice",
..     age: 30
.. }
{age: 30, name: "Alice"}
```

The `..` prompt indicates continuation mode. Press Ctrl+C to abort multi-line input and return to the main `>>` prompt.

### Command History

- Use ↑ and ↓ to navigate through previous commands
- History is saved to `~/.parsley_history` and persists across sessions
- Multi-line commands are saved as complete units

### Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| **Ctrl+A** | Move to start of line |
| **Ctrl+E** | Move to end of line |
| **Ctrl+K** | Delete from cursor to end of line |
| **Ctrl+U** | Delete from cursor to start of line |
| **Ctrl+C** | Abort current line (clears multi-line buffer) |
| **Ctrl+D** | Exit REPL |

### Example Session

```
>> let name = "Parsley"
>> log("Hello, {name}!")
Hello, Parsley!

>> let add = fn(a, b) { a + b }
>> add(5, 3)
8

>> let nums = [1, 2, 3]
>> nums.map(fn(x) { x * 2 })
[2, 4, 6]

>> exit
Goodbye!
```

---

## Error Messages

Parsley provides clear, human-readable error messages. Type errors display Parsley type names:

| Type | Display |
|------|---------|
| String | `STRING` |
| Integer | `INTEGER` |
| Float | `FLOAT` |
| Boolean | `BOOLEAN` |
| Array | `ARRAY` |
| Dictionary | `DICTIONARY` |
| Function | `FUNCTION` |
| Null | `NULL` |

### Example Errors

```parsley
sin("hello")
// ERROR: argument to `sin` not supported, got STRING

pow("a", 2)
// ERROR: first argument to `pow` not supported, got STRING

SQLITE(123)
// ERROR: first argument to `SQLITE` must be a path, got INTEGER
```

---

## Standard Library

Parsley includes a standard library of modules that provide additional functionality. Import them using the `std/` prefix:

```parsley
let {table} = import("std/table")
```

> **Note:** The `@std/` path literal syntax is planned but not yet implemented. Use string imports for now.

### Table Module (`std/table`)

The Table module provides SQL-like operations on arrays of dictionaries.

#### Creating a Table

```parsley
let {table} = import("std/table")

let data = [
    {name: "Alice", age: 30, dept: "Engineering"},
    {name: "Bob", age: 25, dept: "Sales"},
    {name: "Carol", age: 35, dept: "Engineering"}
]

let t = table(data)
```

#### Properties

| Property | Description |
|----------|-------------|
| `.rows` | Returns the array of dictionaries |

#### Methods

| Method | Description |
|--------|-------------|
| `.where(fn)` | Filter rows where predicate returns truthy |
| `.orderBy(col)` | Sort by column (ascending) |
| `.orderBy(col, "desc")` | Sort by column (descending) |
| `.orderBy([col1, col2])` | Sort by multiple columns |
| `.select([cols])` | Keep only specified columns |
| `.limit(n)` | Take first n rows |
| `.limit(n, offset)` | Take n rows starting at offset |
| `.count()` | Return number of rows |
| `.sum(col)` | Sum of numeric values in column |
| `.avg(col)` | Average of numeric values in column |
| `.min(col)` | Minimum value in column |
| `.max(col)` | Maximum value in column |
| `.appendRow(row)` | Append row to end of table |
| `.insertRowAt(i, row)` | Insert row before index |
| `.appendCol(name, values\|fn)` | Append column with values or computed |
| `.insertColAfter(col, name, values\|fn)` | Insert column after existing |
| `.insertColBefore(col, name, values\|fn)` | Insert column before existing |
| `.toHTML()` | Render as HTML table string |
| `.toCSV()` | Render as CSV string (RFC 4180) |

#### Examples

**Filtering:**
```parsley
// Get engineering employees over 28
table(data)
    .where(fn(row) { row.dept == "Engineering" && row.age > 28 })
    .rows  // [{name: "Alice", ...}, {name: "Carol", ...}]
```

**Sorting:**
```parsley
// Sort by age descending
table(data).orderBy("age", "desc").rows

// Sort by department, then by age descending
table(data).orderBy([["dept", "asc"], ["age", "desc"]]).rows
```

**Projection:**
```parsley
// Keep only name and age
table(data).select(["name", "age"]).rows
```

**Aggregation:**
```parsley
table(data).count()         // 3
table(data).sum("age")      // 90
table(data).avg("age")      // 30.0
table(data).min("age")      // 25
table(data).max("age")      // 35
```

**Chaining:**
```parsley
// Active users over 25, sorted by name, first 10
table(users)
    .where(fn(u) { u.active && u.age > 25 })
    .orderBy("name")
    .limit(10)
    .select(["name", "email"])
    .toHTML()
```

**Output Formats:**
```parsley
// HTML table
table(data).toHTML()
// <table><thead><tr><th>age</th><th>dept</th><th>name</th></tr></thead>...

// CSV
table(data).toCSV()
// "age","dept","name"
// 30,"Engineering","Alice"
// ...
```

**Row Operations:**
```parsley
// Append a new row
let t2 = table(data).appendRow({name: "Dave", age: 28, dept: "HR"})

// Insert row at specific position
let t3 = table(data).insertRowAt(1, {name: "Eve", age: 32, dept: "Sales"})
```

**Column Operations:**
```parsley
// Append column with values array
let t = table([{name: "Alice"}, {name: "Bob"}])
let withAge = t.appendCol("age", [30, 25])

// Append computed column using function
let withInitials = t.appendCol("initial", fn(row) { row.name[0] })

// Insert column after existing column
let withMiddle = t.insertColAfter("name", "middle", ["M.", "R."])

// Insert column before existing column
let withId = t.insertColBefore("name", "id", [1, 2])
```

### Math Module (`std/math`)

The Math module provides mathematical functions and constants. Designed for educators, students, and creative coders.

#### Importing

```parsley
// Import specific functions
let {floor, ceil, sqrt, PI} = import("std/math")

// Import entire module
let math = import("std/math")
math.sqrt(16)  // 4
```

> **Note:** The built-in `log` function prints to console. Use `math.log()` to access the natural logarithm function.

#### Constants

| Constant | Value | Description |
|----------|-------|-------------|
| `PI` | 3.14159... | Ratio of circle circumference to diameter |
| `E` | 2.71828... | Euler's number (base of natural logarithm) |
| `TAU` | 6.28318... | 2π, the full circle constant |

```parsley
let {PI, E, TAU} = import("std/math")
PI           // 3.141592653589793
E            // 2.718281828459045
TAU          // 6.283185307179586
```

#### Rounding Functions

These functions return integers:

| Function | Description | Example |
|----------|-------------|---------|
| `floor(x)` | Round down to nearest integer | `floor(3.7)` → `3` |
| `ceil(x)` | Round up to nearest integer | `ceil(3.2)` → `4` |
| `round(x)` | Round to nearest integer | `round(3.5)` → `4` |
| `trunc(x)` | Truncate toward zero | `trunc(-3.7)` → `-3` |

```parsley
let {floor, ceil, round, trunc} = import("std/math")
floor(3.7)   // 3
ceil(3.2)    // 4
round(3.5)   // 4
trunc(-3.7)  // -3 (toward zero)
```

#### Comparison Functions

| Function | Description | Example |
|----------|-------------|---------|
| `abs(x)` | Absolute value | `abs(-5)` → `5` |
| `sign(x)` | Sign of number (-1, 0, or 1) | `sign(-5)` → `-1` |
| `clamp(x, min, max)` | Constrain value to range | `clamp(15, 0, 10)` → `10` |

```parsley
let {abs, sign, clamp} = import("std/math")
abs(-3.14)        // 3.14
sign(-42)         // -1
clamp(15, 0, 10)  // 10
clamp(5, 0, 10)   // 5
```

#### Aggregation Functions

These functions accept either two arguments OR a single array:

| Function | Description |
|----------|-------------|
| `min(a, b)` or `min(arr)` | Minimum value |
| `max(a, b)` or `max(arr)` | Maximum value |
| `sum(a, b)` or `sum(arr)` | Sum of values |
| `avg(a, b)` or `avg(arr)` | Average (mean) of values |
| `mean(...)` | Alias for `avg` |
| `product(a, b)` or `product(arr)` | Product of values |
| `count(arr)` | Number of elements in array |

```parsley
let {min, max, sum, avg, product, count} = import("std/math")

// Two argument form
min(5, 3)       // 3
max(5, 3)       // 5
sum(10, 20)     // 30
avg(10, 20)     // 15.0

// Array form
min([5, 3, 8, 1])     // 1
max([5, 3, 8, 1])     // 8
sum([1, 2, 3, 4])     // 10
avg([1, 2, 3, 4])     // 2.5
product([2, 3, 4])    // 24
count([1, 2, 3])      // 3
```

#### Statistics Functions

These functions require a non-empty array:

| Function | Description |
|----------|-------------|
| `median(arr)` | Middle value (or average of two middle values) |
| `mode(arr)` | Most frequent value (smallest on tie) |
| `stddev(arr)` | Population standard deviation |
| `variance(arr)` | Population variance |
| `range(arr)` | Difference between max and min |

```parsley
let {median, mode, stddev, variance, range} = import("std/math")

median([1, 2, 3])        // 2
median([1, 2, 3, 4])     // 2.5
mode([1, 2, 2, 3])       // 2
stddev([2, 4, 4, 4, 5, 5, 7, 9])  // 2.0
variance([2, 4, 4, 4, 5, 5, 7, 9]) // 4.0
range([1, 5, 3, 10, 2])  // 9
```

#### Random Functions

| Function | Description |
|----------|-------------|
| `random()` | Random float in [0, 1) |
| `random(max)` | Random float in [0, max) |
| `random(min, max)` | Random float in [min, max) |
| `randomInt(max)` | Random integer in [0, max] |
| `randomInt(min, max)` | Random integer in [min, max] |
| `seed(n)` | Seed the random generator for reproducibility |

```parsley
let {random, randomInt, seed} = import("std/math")

random()           // e.g., 0.7234...
random(10)         // e.g., 4.891... (0 to <10)
random(5, 10)      // e.g., 7.234... (5 to <10)
randomInt(6)       // e.g., 4 (0 to 6 inclusive, like a die)
randomInt(1, 6)    // e.g., 3 (1 to 6 inclusive)

// For reproducible results
seed(42)
random()           // Always same value with same seed
```

#### Powers & Logarithms

| Function | Description |
|----------|-------------|
| `sqrt(x)` | Square root (error if negative) |
| `pow(base, exp)` | Raise base to power |
| `exp(x)` | e raised to power x |
| `log(x)` | Natural logarithm (error if ≤ 0) |
| `log10(x)` | Base-10 logarithm (error if ≤ 0) |

```parsley
let math = import("std/math")

math.sqrt(16)          // 4
math.pow(2, 10)        // 1024
math.exp(1)            // 2.718281828459045 (e)
math.log(math.E)       // 1.0
math.log10(1000)       // 3.0
```

#### Trigonometry

All angles are in radians. Use `degrees()` and `radians()` to convert.

| Function | Description |
|----------|-------------|
| `sin(x)` | Sine |
| `cos(x)` | Cosine |
| `tan(x)` | Tangent |
| `asin(x)` | Arcsine (input in [-1, 1]) |
| `acos(x)` | Arccosine (input in [-1, 1]) |
| `atan(x)` | Arctangent |
| `atan2(y, x)` | Arctangent of y/x (handles quadrants) |

```parsley
let {sin, cos, tan, asin, PI} = import("std/math")

sin(0)            // 0
sin(PI / 2)       // 1
cos(0)            // 1
cos(PI)           // -1
asin(1)           // 1.5707... (PI/2)
```

#### Angular Conversion

| Function | Description |
|----------|-------------|
| `degrees(radians)` | Convert radians to degrees |
| `radians(degrees)` | Convert degrees to radians |

```parsley
let {degrees, radians, PI, sin} = import("std/math")

degrees(PI)       // 180
radians(180)      // 3.14159... (PI)

// Using degrees for trig
sin(radians(90))  // 1
```

#### Geometry & Interpolation

| Function | Description |
|----------|-------------|
| `hypot(x, y)` | Hypotenuse (√(x² + y²)) |
| `dist(x1, y1, x2, y2)` | Distance between two points |
| `lerp(a, b, t)` | Linear interpolation (t=0 gives a, t=1 gives b) |
| `map(value, inMin, inMax, outMin, outMax)` | Map value from one range to another |

```parsley
let {hypot, dist, lerp, map} = import("std/math")

// Pythagorean theorem
hypot(3, 4)                // 5

// Distance between points
dist(0, 0, 3, 4)           // 5
dist(1, 1, 4, 5)           // 5

// Linear interpolation
lerp(0, 100, 0)            // 0 (start)
lerp(0, 100, 1)            // 100 (end)
lerp(0, 100, 0.5)          // 50 (middle)
lerp(0, 100, 0.25)         // 25

// Range mapping
map(50, 0, 100, 0, 1)      // 0.5 (50% of input range = 50% of output)
map(32, 32, 212, 0, 100)   // 0 (32°F = 0°C)
map(212, 32, 212, 0, 100)  // 100 (212°F = 100°C)
```

### Validation Module (`std/valid`)

The Validation module provides functions for validating user input, form data, and common formats.

#### Importing

```parsley
// Import specific validators
let {email, minLen, positive} = import("std/valid")

// Import entire module
let valid = import("std/valid")
valid.email("test@example.com")  // true
```

#### Type Validators

| Function | Description | Example |
|----------|-------------|---------|
| `string(x)` | True if x is a string | `string("hello")` → `true` |
| `number(x)` | True if x is int or float | `number(3.14)` → `true` |
| `integer(x)` | True if x is an integer | `integer(42)` → `true` |
| `boolean(x)` | True if x is boolean | `boolean(true)` → `true` |
| `array(x)` | True if x is an array | `array([1,2])` → `true` |
| `dict(x)` | True if x is a dictionary | `dict({a:1})` → `true` |

```parsley
let valid = import("std/valid")

valid.string("hello")   // true
valid.string(123)       // false
valid.number(3.14)      // true
valid.number("3.14")    // false (string)
valid.integer(42)       // true
valid.integer(3.14)     // false
```

#### String Validators

| Function | Description | Example |
|----------|-------------|---------|
| `empty(x)` | True if string is empty or whitespace | `empty("  ")` → `true` |
| `minLen(x, n)` | True if length ≥ n | `minLen("hello", 3)` → `true` |
| `maxLen(x, n)` | True if length ≤ n | `maxLen("hi", 10)` → `true` |
| `length(x, min, max)` | True if min ≤ length ≤ max | `length("hello", 1, 10)` → `true` |
| `matches(x, regex)` | True if x matches regex | `matches("abc", "^[a-z]+$")` → `true` |
| `alpha(x)` | True if only letters a-z, A-Z | `alpha("Hello")` → `true` |
| `alphanumeric(x)` | True if only letters and digits | `alphanumeric("abc123")` → `true` |
| `numeric(x)` | True if parseable as number | `numeric("123.45")` → `true` |

```parsley
let valid = import("std/valid")

// Form validation example
let username = "alice123"
let password = "secret"

valid.minLen(username, 3)         // true - at least 3 chars
valid.maxLen(username, 20)        // true - at most 20 chars
valid.alphanumeric(username)      // true - only letters/digits
valid.length(password, 6, 100)    // true - 6-100 chars

// Unicode support (counts runes, not bytes)
valid.minLen("日本語", 3)           // true - 3 characters
```

#### Number Validators

| Function | Description | Example |
|----------|-------------|---------|
| `min(x, n)` | True if x ≥ n | `min(5, 1)` → `true` |
| `max(x, n)` | True if x ≤ n | `max(5, 10)` → `true` |
| `between(x, lo, hi)` | True if lo ≤ x ≤ hi | `between(5, 1, 10)` → `true` |
| `positive(x)` | True if x > 0 | `positive(5)` → `true` |
| `negative(x)` | True if x < 0 | `negative(-5)` → `true` |

```parsley
let valid = import("std/valid")

// Age validation
let age = 25
valid.positive(age)           // true
valid.between(age, 0, 120)    // true

// Quantity validation
let qty = 3
valid.min(qty, 1)             // true - at least 1
valid.max(qty, 100)           // true - at most 100
```

#### Format Validators

| Function | Description | Example |
|----------|-------------|---------|
| `email(x)` | Basic email format | `email("test@example.com")` → `true` |
| `url(x)` | Valid http/https URL | `url("https://example.com")` → `true` |
| `uuid(x)` | UUID format | `uuid("550e8400-e29b-...")` → `true` |
| `phone(x)` | Loose phone format | `phone("+1 (555) 123-4567")` → `true` |
| `creditCard(x)` | Luhn algorithm check | `creditCard("4111111111111111")` → `true` |
| `time(x)` | Time format HH:MM[:SS] | `time("14:30")` → `true` |

```parsley
let valid = import("std/valid")

// Contact form validation
valid.email("user@example.com")         // true
valid.phone("+1 (555) 123-4567")        // true
valid.url("https://example.com")        // true

// Invalid formats
valid.email("not-an-email")             // false
valid.url("example.com")                // false (no protocol)
valid.phone("123")                      // false (too short)
```

#### Date Validators

| Function | Description | Example |
|----------|-------------|---------|
| `date(x, locale?)` | Valid date (default ISO) | `date("2024-12-25")` → `true` |
| `parseDate(x, locale)` | Parse to ISO or null | `parseDate("12/25/2024", "US")` → `"2024-12-25"` |

**Supported locales:**
- `"ISO"` (default): `YYYY-MM-DD`
- `"US"`: `MM/DD/YYYY`
- `"GB"`: `DD/MM/YYYY`

```parsley
let valid = import("std/valid")

// ISO format (default)
valid.date("2024-12-25")                // true
valid.date("2024-02-30")                // false (Feb 30 doesn't exist)
valid.date("2024-02-29")                // true (2024 is leap year)

// US format (MM/DD/YYYY)
valid.date("12/25/2024", "US")          // true
valid.date("25/12/2024", "US")          // false (month 25 invalid)

// GB format (DD/MM/YYYY)
valid.date("25/12/2024", "GB")          // true

// Parse to ISO format
valid.parseDate("12/25/2024", "US")     // "2024-12-25"
valid.parseDate("25/12/2024", "GB")     // "2024-12-25"
valid.parseDate("invalid", "US")        // null
```

#### Postal Code Validator

| Function | Description | Example |
|----------|-------------|---------|
| `postalCode(x, locale)` | Valid postal code | `postalCode("90210", "US")` → `true` |

**Supported locales:**
- `"US"`: 5-digit or 9-digit (12345 or 12345-6789)
- `"GB"`: UK format (SW1A 1AA, M1 1AA)

```parsley
let valid = import("std/valid")

// US postal codes
valid.postalCode("90210", "US")         // true
valid.postalCode("90210-1234", "US")    // true
valid.postalCode("9021", "US")          // false (too short)

// UK postal codes
valid.postalCode("SW1A 1AA", "GB")      // true
valid.postalCode("M1 1AA", "GB")        // true
valid.postalCode("12345", "GB")         // false
```

#### Collection Validators

| Function | Description | Example |
|----------|-------------|---------|
| `contains(arr, x)` | True if array contains value | `contains([1,2,3], 2)` → `true` |
| `oneOf(x, options)` | True if x is in options | `oneOf("red", ["red","green"])` → `true` |

```parsley
let valid = import("std/valid")

// Check if value is in allowed list
let color = "red"
valid.oneOf(color, ["red", "green", "blue"])  // true

// Check array membership
let cart = ["apple", "banana", "orange"]
valid.contains(cart, "apple")                  // true
valid.contains(cart, "grape")                  // false
```

#### Complete Form Validation Example

```parsley
let valid = import("std/valid")

// Validate a registration form
let form = {
    username: "alice123",
    email: "alice@example.com",
    password: "secret123",
    age: 25,
    country: "US",
    zip: "90210"
}

let errors = []

if (!valid.alphanumeric(form.username)) {
    errors = errors ++ ["Username must be alphanumeric"]
}
if (!valid.length(form.username, 3, 20)) {
    errors = errors ++ ["Username must be 3-20 characters"]
}
if (!valid.email(form.email)) {
    errors = errors ++ ["Invalid email address"]
}
if (!valid.minLen(form.password, 8)) {
    errors = errors ++ ["Password must be at least 8 characters"]
}
if (!valid.between(form.age, 13, 120)) {
    errors = errors ++ ["Age must be between 13 and 120"]
}
if (!valid.postalCode(form.zip, form.country)) {
    errors = errors ++ ["Invalid postal code"]
}

if (len(errors) == 0) {
    log("Form is valid!")
} else {
    for (err in errors) {
        log("Error: {err}")
    }
}
```

---

## Basil Server Functions

These functions are only available when running Parsley scripts in Basil server handlers.

### The `basil` Namespace

When running in Basil, scripts have access to the `basil` global object which provides HTTP request/response handling, authentication context, and server utilities.

#### basil.http.request

Contains information about the incoming HTTP request.

| Property | Type | Description |
|----------|------|-------------|
| `method` | String | HTTP method (GET, POST, etc.) |
| `path` | String | URL path |
| `query` | Dict | Query string parameters |
| `headers` | Dict | HTTP headers |
| `cookies` | Dict | Request cookies (name → value) |
| `body` | String | Raw request body (POST/PUT/PATCH) |
| `form` | Dict | Parsed form data (POST/PUT/PATCH) |
| `files` | Dict | Uploaded files metadata |
| `host` | String | Request host |
| `remoteAddr` | String | Client IP address |
| `subpath` | Path | Remaining path in site mode |

**Reading cookies:**
```parsley
// Access all cookies as a dict
let cookies = basil.http.request.cookies
let theme = cookies.theme ?? "light"

// Direct access
let sessionId = basil.http.request.cookies.session_id
```

#### basil.http.response

Control the HTTP response status, headers, and cookies.

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `status` | Int | 200 | HTTP status code |
| `headers` | Dict | {} | Response headers |
| `cookies` | Dict | {} | Cookies to set |

**Setting response status and headers:**
```parsley
basil.http.response.status = 404
basil.http.response.headers["X-Custom"] = "value"
basil.http.response.headers["Content-Type"] = "application/json"
```

**Setting cookies:**

Cookies can be set as simple strings (using secure defaults) or as dicts with options:

```parsley
// Simple value (uses secure defaults)
basil.http.response.cookies.theme = "dark"

// With options
basil.http.response.cookies.remember_token = {
    value: token,
    maxAge: @30d,           // Duration literal for 30 days
    path: "/",
    httpOnly: true,
    secure: true,
    sameSite: "Strict"
}

// Delete a cookie (maxAge: 0)
basil.http.response.cookies.old_cookie = {value: "", maxAge: @0s}
```

**Cookie options:**

| Option | Type | Default (prod) | Default (dev) | Description |
|--------|------|----------------|---------------|-------------|
| `value` | String | required | required | Cookie value |
| `maxAge` | Duration/Int | session | session | Seconds until expiry |
| `expires` | DateTime | — | — | Absolute expiry time |
| `path` | String | `"/"` | `"/"` | URL path scope |
| `domain` | String | — | — | Domain scope |
| `secure` | Bool | `true` | `false` | HTTPS only |
| `httpOnly` | Bool | `true` | `true` | No JavaScript access |
| `sameSite` | String | `"Lax"` | `"Lax"` | `"Strict"`, `"Lax"`, or `"None"` |

**Security notes:**
- In production, `secure` defaults to `true` (HTTPS only)
- In dev mode, `secure` defaults to `false` for localhost testing
- `httpOnly` always defaults to `true` to prevent XSS attacks
- Setting `sameSite: "None"` automatically enables `secure: true`

#### basil.auth

Authentication context (when auth is enabled).

| Property | Type | Description |
|----------|------|-------------|
| `required` | Bool | Whether auth is required for this route |
| `user` | Dict/null | Authenticated user or null |

**User object (when authenticated):**
```parsley
if (basil.auth.user != null) {
    let user = basil.auth.user
    log("User: {user.name} ({user.email})")
    log("ID: {user.id}")
    log("Joined: {user.created}")
}
```

#### basil.csrf

CSRF (Cross-Site Request Forgery) protection context.

| Property | Type | Description |
|----------|------|-------------|
| `token` | String | CSRF token for form submissions |

**Usage in forms:**
```parsley
<form method=POST action="/submit">
    <input type=hidden name=_csrf value={basil.csrf.token}/>
    <input type=text name=email/>
    <button>Submit</button>
</form>
```

**Usage in meta tag (for AJAX):**
```parsley
<head>
    <meta name=csrf-token content={basil.csrf.token}/>
</head>
```

Then in JavaScript:
```javascript
fetch('/submit', {
    method: 'POST',
    headers: {
        'X-CSRF-Token': document.querySelector('meta[name=csrf-token]').content,
        'Content-Type': 'application/json'
    },
    body: JSON.stringify(data)
})
```

**How CSRF protection works:**
- The token is stored in a cookie (`_csrf`) and must be submitted with forms
- POST, PUT, PATCH, DELETE requests to auth routes are validated automatically
- API routes (`type: api`) skip CSRF validation (they use API keys)
- Invalid or missing tokens return 403 Forbidden

#### basil.context

A mutable dict for passing data between handlers and modules:

```parsley
basil.context.pageTitle = "Dashboard"
basil.context.breadcrumbs = ["Home", "Dashboard"]
```

#### basil.public_dir

The public directory for this route (if configured).

### API Module (`std/api`)

The API module provides helpers for building web APIs with proper HTTP responses, auth wrappers, and redirects.

#### Importing

```parsley
// Import specific functions
let {redirect, notFound, forbidden} = import("std/api")

// Import entire module
let api = import("std/api")
api.redirect("/dashboard")
```

#### Redirect Helper

Return a redirect response from a handler:

| Function | Description |
|----------|-------------|
| `redirect(url, status?)` | Returns a redirect response |

**Arguments:**
- `url` — Target URL (string or path literal)
- `status` — Optional HTTP status code (default: 302). Must be 3xx.

**Valid status codes:** 300-308 (3xx redirect codes only)

```parsley
let {redirect} = import("std/api")

// Basic redirect (302 Found)
redirect("/dashboard")

// Permanent redirect (301)
redirect("/new-page", 301)

// Redirect to external URL
redirect("https://example.com/page")

// Using path literal
redirect(@/users/profile)

// Post-login redirect
if (loggedIn) {
    redirect(basil.http.request.query.return ?? "/home")
}
```

**Common status codes:**
| Code | Name | Use Case |
|------|------|----------|
| 301 | Moved Permanently | Page permanently moved, search engines update |
| 302 | Found | Temporary redirect (default) |
| 303 | See Other | Redirect after POST (PRG pattern) |
| 307 | Temporary Redirect | Temporary, preserve HTTP method |
| 308 | Permanent Redirect | Permanent, preserve HTTP method |

#### Error Helpers

Return HTTP error responses from handlers:

| Function | Status | Default Message |
|----------|--------|-----------------|
| `badRequest(msg?)` | 400 | "Bad request" |
| `unauthorized(msg?)` | 401 | "Unauthorized" |
| `forbidden(msg?)` | 403 | "Forbidden" |
| `notFound(msg?)` | 404 | "Not found" |
| `conflict(msg?)` | 409 | "Conflict" |
| `gone(msg?)` | 410 | "Gone" |
| `unprocessable(msg?)` | 422 | "Unprocessable entity" |
| `tooMany(msg?)` | 429 | "Too many requests" |

```parsley
let {notFound, forbidden} = import("std/api")

// Return 404 with default message
notFound()

// Return 404 with custom message
notFound("User not found")

// Access control
if (todo.owner_id != user.id) {
    forbidden("Not your todo")
}
```

#### Auth Wrappers

Wrap handlers to control authentication requirements:

| Wrapper | Effect |
|---------|--------|
| `public(fn)` | No auth required |
| `auth(fn)` | Auth required (default behavior) |
| `adminOnly(fn)` | Auth required + admin role |
| `roles(roleList, fn)` | Auth required + specific roles |

```parsley
let {public, adminOnly, roles} = import("std/api")

// Public endpoint - no auth required
export get = public(fn(req) {
    // Anyone can access this
    getPublicData()
})

// Admin only
export delete = adminOnly(fn(req) {
    // Only admins can delete
    deleteRecord(req.params.id)
})

// Specific roles
export put = roles(["editor", "admin"], fn(req) {
    // Only editors and admins can update
    updateRecord(req.params.id, req.form)
})
```

---

### Site Mode (Filesystem-Based Routing)

Basil supports filesystem-based routing via the `site:` configuration option. Instead of explicit route definitions, requests are routed to `index.pars` files based on the URL path.

**Configuration:**
```yaml
site: ./site  # Directory containing index.pars files
# Note: site: and routes: are mutually exclusive
```

**How it works:**
1. Given a request path like `/reports/2025/Q4/`, Basil walks back from the deepest path toward the root
2. It finds the first directory containing an `index.pars` file
3. That handler receives the request with `basil.http.request.subpath` containing the remaining path segments

**Example directory structure:**
```
site/
  index.pars           # Handles /
  reports/
    index.pars         # Handles /reports/, /reports/2025/, /reports/2025/Q4/
  admin/
    index.pars         # Handles /admin/, /admin/settings/
    users/
      index.pars       # Handles /admin/users/, /admin/users/123/
```

**Accessing the subpath:**
```parsley
// In site/reports/index.pars, for request /reports/2025/Q4/
let subpath = basil.http.request.subpath

subpath.segments       // ["2025", "Q4"]
subpath.segments[0]    // "2025"
subpath.segments.length() // 2

// Empty subpath for exact match (request to /reports/)
// subpath.segments would be []
```

**Trailing slash redirect:**
Requests to directory-like paths without a trailing slash (e.g., `/reports`) are automatically redirected to `/reports/` with a 302 redirect.

**Security:**
- Path traversal attempts (`..`) are blocked (400 Bad Request)
- Dotfile/hidden file access (`.git`, `.env`) is blocked (404 Not Found)

---

### publicUrl()

Makes a private file (e.g., a component asset in `modules/`) accessible via a public URL.

```parsley
// In a component file (modules/Button.pars)
let icon = publicUrl(@./icon.svg)
<button>
  <img src={icon} alt=""/>
  Button
</button>
// Output: <button><img src="/__p/a3f2b1c8.svg" alt=""/>Button</button>
```

**Signature:**
```
publicUrl(path) -> string
```

**Arguments:**
- `path`: A path literal (`@./file.ext`) or string path to the file

**Returns:** A public URL string in the format `/__p/{hash}.{ext}`

**Features:**
- **Content-hashed URLs**: URLs include a hash of file contents for automatic cache-busting
- **Aggressive caching**: Assets are served with `Cache-Control: public, max-age=31536000, immutable`
- **No file copying**: Files remain in their original location
- **Lazy hashing**: Hash is computed once and cached until file changes

**Size Limits:**
- Files >10MB trigger a warning in dev mode
- Files >100MB return an error (use `public/` folder instead)

**Security:**
- Path must be within the handler's root directory
- Path traversal outside handler root is blocked

**Example: Component with co-located assets:**
```parsley
// modules/Card.pars
export Card = fn({title, image}) {
  let imageUrl = publicUrl(image)
  let styleUrl = publicUrl(@./card.css)
  
  <>
    <link rel="stylesheet" href={styleUrl}/>
    <div class="card">
      <img src={imageUrl} alt=""/>
      <h3>{title}</h3>
    </div>
  </>
}
```

---

## Go Library

The `pkg/parsley` package provides a public API for embedding Parsley in Go applications.

### Installation

```bash
go get github.com/sambeau/parsley/pkg/parsley
```

### Basic Usage

```go
import "github.com/sambeau/parsley/pkg/parsley"

// Simple evaluation
result, err := parsley.Eval(`1 + 2`)
fmt.Println(result.String()) // "3"

// With variables
result, err := parsley.Eval(`name ++ "!"`,
    parsley.WithVar("name", "Hello"),
)

// Evaluate a file
result, err := parsley.EvalFile("script.pars")
```

### Options

| Option | Description |
|--------|-------------|
| `WithVar(name, value)` | Pre-populate a variable |
| `WithEnv(env)` | Use a pre-configured environment |
| `WithSecurity(policy)` | Set file system security policy |
| `WithLogger(logger)` | Set logger for `log()`/`logLine()` |
| `WithFilename(name)` | Set filename for error messages |
| `WithDB(name, db, driver)` | Inject a server-managed database connection |

### Type Conversion

```go
// Go → Parsley
obj, err := parsley.ToParsley(42)        // Integer
obj, err := parsley.ToParsley("hello")   // String
obj, err := parsley.ToParsley([]int{1,2}) // Array

// Parsley → Go
val := parsley.FromParsley(obj)
```

### Loggers

```go
parsley.StdoutLogger()          // Write to stdout (default)
parsley.WriterLogger(w)         // Write to io.Writer
parsley.NewBufferedLogger()     // Capture for testing
parsley.NullLogger()            // Discard output
```

### Full Documentation

See `pkg/parsley/README.md` for complete API documentation and examples.
