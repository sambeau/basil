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

Interpolated strings using `{expression}` syntax. Any valid Parsley expression can be used inside the braces.

**Note**: Unlike JavaScript template literals, Parsley uses `{expr}` not `${expr}`.

```parsley
let name = "Alice"
`Hello, {name}!`        // "Hello, Alice!"
`2 + 2 = {2 + 2}`       // "2 + 2 = 4"
`{name.toUpper()}`      // "ALICE"
```

#### Raw Strings (`'...'`)

Backslashes are literal (no escape sequences). Interpolation only with `@{expression}`.

**Use for**: Regular expressions, file paths, SQL patterns.

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

#### Truthiness

Parsley has Python-style truthiness:

**Falsy values:**
- `false`
- `null`
- `0` (integer)
- `0.0` (float)
- `""` (empty string)
- `[]` (empty array)
- `{}` (empty dictionary)

**Everything else is truthy**, including:
- `true`
- Non-zero numbers
- Non-empty strings, arrays, and dictionaries

This matches the behavior of Python, PHP, and Perl, and avoids JavaScript's confusing inconsistency where empty arrays/objects are truthy. The design makes Parsley more intuitive for common web development patterns like form validation and collection checking:

```parsley
// Intuitive for form validation
if (username) { ... }      // fails for ""
if (items) { ... }         // fails for []
if (config) { ... }        // fails for {}
if (count) { ... }         // fails for 0
```

---

### 1.4 Arrays

Arrays are ordered, zero-indexed collections that can hold values of any type.

```parsley
[1, 2, 3]
let empty = []
```

#### Nested Arrays

Arrays can contain other arrays, useful for matrices and tabular data:

```parsley
let matrix = [[1, 2, 3], [4, 5, 6], [7, 8, 9]]
matrix[1][0]                    // 4 (second row, first column)
matrix[0]                       // [1, 2, 3] (first row)
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

Dictionaries are unordered key-value collections. Keys must be strings (quotes optional if valid identifiers).

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

Extract specific fields by name:

```parsley
let {name, age} = person        // Extract fields
```

Use `...rest` to collect remaining fields into a new dictionary:

```parsley
let person = {name: "Bob", age: 25, city: "NYC"}
let {name, ...rest} = person    // name="Bob", rest={age: 25, city: "NYC"}
```

---

### 1.6 Functions

Functions are defined with the `fn` keyword (or `function` as an alias).

**Note**: Arrow function syntax (`x => x * 2`) is **not supported**. Always use `fn(x) { x * 2 }`.

```parsley
let double = fn(x) { x * 2 }
double(5)                       // 10

let add = fn(a, b) { a + b }
add(3, 4)                       // 7

let constant = fn() { 42 }
constant()                      // 42
```

**Implicit return**: The last expression is returned automatically. Unlike JavaScript, you don't need `return` for single expressions.

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

Parsley has first-class support for monetary values with precise decimal handling.

#### Symbol Format

```parsley
$12.34                          // USD
$99.99
```

#### Unicode Currency Symbols

Parsley recognizes common currency symbols directly:

```parsley
€123.45                         // Euro (EUR)
£99.99                          // British Pound (GBP)
¥5000                           // Japanese Yen (JPY)
```

#### Compound Symbols

```parsley
CA$50.00                        // Canadian Dollar (CAD)
AU$75.00                        // Australian Dollar (AUD)
HK$100.00                       // Hong Kong Dollar (HKD)
S$88.00                         // Singapore Dollar (SGD)
CN¥200.00                       // Chinese Yuan (CNY)
```

#### CODE# Format

For currencies without symbols, use the ISO 4217 code followed by `#`:

```parsley
EUR#50.00                       // Euro
GBP#25.00                       // British Pound
CHF#100.00                      // Swiss Franc
INR#1000.00                     // Indian Rupee
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

### 1.14 Table Literals

Table literals create structured tabular data with named columns. Tables can be created from arrays of dictionaries, arrays of arrays (with header row), or with an optional schema.

#### Basic Syntax

```parsley
// From array of dictionaries (columns inferred from keys)
@table [
    {name: "Alice", age: 30},
    {name: "Bob", age: 25}
]

// From array of arrays (first row is header)
@table [
    ["name", "age"],
    ["Alice", 30],
    ["Bob", 25]
]
```

#### With Schema

Tables can reference a schema for validation and defaults:

```parsley
@schema Person { name: string, age: integer = 0 }

// Schema validates and applies defaults
@table(Person) [
    {name: "Alice", age: 30},
    {name: "Bob"}              // age defaults to 0
]
```

#### Empty Tables

```parsley
// Empty table with no columns
@table []

// Empty table with schema (has columns but no rows)
@table(Person) []
```

---

### 1.15 Schema Literals

Schemas define the structure of records and tables. They specify field names, types, validation rules, default values, and metadata. Schemas are used for database table bindings, form validation, and typed data structures.

#### Basic Syntax

```parsley
@schema Person {
    name: string
    age: integer
    email: email
}
```

#### Field Types

| Type | Description | SQL Type |
|------|-------------|----------|
| `string` | Text data | `TEXT` |
| `text` | Long text data | `TEXT` |
| `int`, `integer` | Whole numbers | `INTEGER` |
| `bigint` | Large integers | `BIGINT` |
| `float`, `number` | Decimal numbers | `REAL` |
| `bool`, `boolean` | True/false | `INTEGER` (0/1) |
| `datetime` | Date and time | `DATETIME` |
| `date` | Date only | `DATE` |
| `time` | Time only | `TIME` |
| `money` | Monetary values | `REAL` |
| `uuid` | UUID strings | `TEXT` |
| `ulid` | ULID strings | `TEXT` |
| `json` | JSON data | `TEXT` |
| `email` | Email (validated) | `TEXT` |
| `url` | URL (validated) | `TEXT` |
| `phone` | Phone number | `TEXT` |
| `slug` | URL slug (validated) | `TEXT` |
| `enum` | One of specified values | `TEXT` |

#### Nullable Fields

Append `?` to make a field nullable:

```parsley
@schema User {
    name: string           // Required
    nickname: string?      // Optional (nullable)
    email: email?          // Optional email
}
```

#### Default Values

Use `=` to specify default values:

```parsley
@schema Post {
    title: string
    status: string = "draft"
    views: integer = 0
    published: boolean = false
    createdAt: datetime = @now
}
```

#### Enum Types

Define allowed values inline:

```parsley
@schema Task {
    title: string
    priority: enum["low", "medium", "high"] = "medium"
    status: enum["todo", "in-progress", "done"]
}
```

#### Type Constraints

Add constraints using `(key: value)` syntax:

```parsley
@schema Profile {
    username: string(min: 3, max: 20, unique: true)
    age: integer(min: 0, max: 150)
    bio: text(max: 500)
}
```

| Constraint | Applies To | Description |
|------------|------------|-------------|
| `min` | string, integer | Minimum length or value |
| `max` | string, integer | Maximum length or value |
| `pattern` | string | Regex pattern for validation |
| `required` | any | Field must have a non-null value |
| `auto` | any | Database/server generates this value |
| `readOnly` | any | Field cannot be set from client/form input |
| `unique` | any | UNIQUE constraint in SQL |

#### The `auto` Constraint

The `auto` constraint marks fields whose values are generated by the database or server (e.g., auto-increment IDs, timestamps). Auto fields are skipped during validation and are immutable on updates.

```parsley
@schema User {
    id: id(auto)                     // Database generates on insert
    createdAt: datetime(auto)        // Server sets on insert
    updatedAt: datetime(auto)        // Server sets on insert/update
    name: string(required)
}

// Valid - id and timestamps are auto, don't need to be provided
let user = User({name: "Alice"})
user.validate().isValid()            // true

// Error - cannot update auto fields
user.update({id: "new-id"})          // Error: cannot update auto field 'id'
```

**Note:** `auto` and `required` cannot be combined on the same field.

#### Field Metadata (Pipe Syntax)

Add UI metadata using the pipe `|` syntax:

```parsley
@schema Contact {
    name: string | {title: "Full Name", placeholder: "Enter your name"}
    email: email | {title: "Email Address", hidden: false}
    notes: text | {title: "Notes", placeholder: "Optional notes...", hidden: true}
}
```

Common metadata keys:
- `title` — Display label for forms/tables
- `placeholder` — Input placeholder text
- `hidden` — Hide field in auto-generated UIs

#### Complete Example

```parsley
@schema User {
    id: id(auto)
    username: string(min: 3, max: 30, unique: true) | {title: "Username"}
    email: email(unique: true) | {title: "Email Address", placeholder: "user@example.com"}
    password: string | {hidden: true}
    role: enum["user", "admin", "moderator"] = "user" | {title: "Role"}
    bio: text? | {title: "Biography", placeholder: "Tell us about yourself..."}
    active: boolean = true
    createdAt: datetime(auto) | {title: "Created", hidden: true}
}
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

Standard boolean operators, e.g.:

```parsley
let age = 20
let status = if (age >= 18) "adult" else "minor"  // "adult"
```

---

### 2.3 Logical / Set Operations

| Operator | Description |
|----------|-------------|
| `&&` | Logical AND (short-circuit) |
| `\|\|` | Logical OR (short-circuit) |
| `!` | Logical NOT |
| `and` | Keyword alias for `&&` |
| `or` | Keyword alias for `\|\|` |

#### Boolean Logic

For boolean values, these operators work as expected:

```parsley
true && true                    // true
false || true                   // true
let notResult = !false          // true
true and true                   // true
false or true                   // true
```

#### Set Operations on Collections

The logical operators are overloaded to perform set operations when applied to arrays: AND corresponds to intersection; OR corresponds to union.

| Operator | Set Operation | Description |
|----------|---------------|-------------|
| `&&` | Intersection | Elements present in both arrays |
| `\|\|` or `\|` | Union | Elements present in either array (duplicates removed) |

```parsley
// Intersection: elements in BOTH arrays
([1, 2, 3] && [2, 3, 4]).toJSON()    // [2, 3]

// Union: elements in EITHER array (duplicates removed)
([1, 2, 3] || [3, 4, 5]).toJSON()    // [1, 2, 3, 4, 5]
([1, 2] | [2, 3]).toJSON()           // [1, 2, 3]
```

This is useful for filtering, merging lists, and finding common elements without writing explicit loops.

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

### 2.5 Schema Checking

The `is` and `is not` operators check whether a value is a Record or Table bound to a specific schema.

| Operator | Description |
|----------|-------------|
| `is` | Schema identity check (returns boolean) |
| `is not` | Negated schema check (returns boolean) |

```parsley
@schema User { name: string }
@schema Product { sku: string }

let user = User({name: "Alice"})

user is User                    // true
user is Product                 // false
user is not Product             // true
```

**Identity comparison**: Schema checking uses pointer identity, not structural matching. Two schemas with identical fields are still different schemas:

```parsley
@schema UserA { name: string }
@schema UserB { name: string }  // Same fields, different schema

let record = UserA({name: "Bob"})
record is UserA                 // true
record is UserB                 // false (different schema)
```

**Works with Tables too**:

```parsley
@schema Point { x: int, y: int }
let points = @table(Point) [{x: 1, y: 2}]

points is Point                 // true
```

**Non-record values**: For values that aren't Records or Tables (strings, numbers, plain dicts, arrays, etc.), `is` safely returns `false`:

```parsley
"hello" is User                 // false
42 is User                      // false
{name: "Alice"} is User         // false (plain dict, not a Record)
```

**Error case**: The right-hand side must be a schema. Using a non-schema value produces a TypeError:

```parsley
user is 42                      // Error: 'is' requires a schema
user is "User"                  // Error: 'is' requires a schema
```

---

### 2.6 Pattern Matching

The `~` and `!~` operators perform regex matching, similar to Perl's pattern matching syntax.

| Operator | Description |
|----------|-------------|
| `~` | Regex match (returns first match or null) |
| `!~` | Regex not match (returns boolean) |

```parsley
"hello123" ~ /\d+/              // "123" (first match)
"hello" ~ /\d+/                 // null (no match)
"hello" !~ /\d+/                // true (does not match)
"abc123" !~ /\d+/               // false (does match)
```

---

### 2.7 Range

The range operator `..` creates an inclusive sequence of integers from start to end.

```parsley
let range = 1..5                // [1, 2, 3, 4, 5]
let countdown = 5..1            // [5, 4, 3, 2, 1] (descending)
```

**Eager Evaluation**: Ranges are evaluated immediately into arrays. The entire sequence is generated in memory when the expression is evaluated, not lazily on demand. For very large ranges, be mindful of memory usage.

```parsley
1..1000000                      // Creates array with 1 million elements
```

---

### 2.8 Concatenation

```parsley
let concat = [1, 2] ++ [3, 4]   // [1, 2, 3, 4]
let merged = {a: 1} ++ {b: 2}   // {a: 1, b: 2}
```

---

### 2.9 Null Coalescing

The `??` operator returns the right-hand value when the left-hand value is `null`. This provides a concise way to supply default values.

```parsley
null ?? "default"               // "default" (left is null, use right)
"value" ?? "default"            // "value" (left is not null, use left)
0 ?? "default"                  // 0 (0 is not null)
"" ?? "default"                 // "" (empty string is not null)
```

**Note**: Unlike truthiness checks, `??` only triggers on `null`, not on other falsy values like `0`, `""`, or `[]`.

#### Optional Index Access

Use `[?index]` syntax to safely access array or dictionary elements without errors when the index/key doesn't exist, or is out of range:

```parsley
let arr = [1, 2, 3]
arr[?99]                        // null (index out of bounds, no error)
arr[?0]                         // 1 (valid index)

let user = {name: "Alice"}
user[?"email"]                  // null (key doesn't exist, no error)
```

Without `?`, out-of-bounds access would produce an error.

---

### 2.10 DateTime Arithmetic

Parsley supports arithmetic operations on dates, times, and durations with sensible rules.

#### Valid Operations

| Operation | Result | Example |
|-----------|--------|--------|
| datetime + duration | datetime | `@now + @1d` → tomorrow |
| datetime - duration | datetime | `@now - @1w` → one week ago |
| datetime - datetime | duration | `@2024-12-25 - @2024-12-20` → 5 days |
| duration + duration | duration | `@1d + @2h` → 1 day 2 hours |
| duration - duration | duration | `@1w - @2d` → 5 days |
| duration * number | duration | `@1d * 3` → 3 days |
| date && time | datetime | `@2024-12-25 && @14:30` → datetime |

```parsley
@now + @1d                      // Tomorrow
@now - @1w                      // One week ago
@2024-12-25 - @2024-12-20       // 5 days (duration)
@2024-12-25 && @14:30           // 2024-12-25T14:30:00
@1d + @1d                       // 2 days
@1d * 3                         // 3 days
```

#### Invalid Operations

Some operations don't make sense and will produce errors:

```parsley
@2024-12-25 + @2024-12-20       // Error: can't add two dates
3 * @1d                         // Error: number must be on the right
```

**Tip**: Think of durations as time offsets. You can add/subtract offsets from dates, or multiply offsets by numbers, but adding two absolute dates together is meaningless.

---

### 2.11 Precedence Table (Lowest to Highest)

| Level | Operators |
|-------|-----------|
| 1 | `??`, `\|\|`, `or` |
| 2 | `&&`, `and` |
| 3 | `==`, `!=`, `~`, `!~`, `in`, `not in`, `is`, `is not` |
| 4 | `<`, `>`, `<=`, `>=` |
| 5 | `+`, `-`, `..` |
| 6 | `++` |
| 7 | `*`, `/`, `%` |
| 8 | `-`, `!` (prefix) |
| 9 | `.`, `[]`, `()` (access/call) |

---

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

The `try` expression captures errors as values instead of stopping execution. It wraps the result in a dictionary with `result` and `error` fields.

```parsley
let safeDivide = fn(a, b) {
    check b != 0 else fail("division by zero")
    a / b
}

let result = try safeDivide(10, 0)
// {result: null, error: "division by zero"}

let result = try safeDivide(10, 2)
// {result: 5, error: null}
```

#### The `fail` Function

Use `fail(message)` to create a catchable error. Unlike runtime errors, `fail` produces a "value-class" error that can be captured by `try`. This is typically used with `check...else` for validation:

```parsley
let validate = fn(email) {
    check email.includes("@") else fail("invalid email")
    email
}

let result = try validate("bad-email")
// {result: null, error: "invalid email"}
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

## 4. Statements

### 4.1 Let Binding

The `let` keyword declares and initializes a variable.

```parsley
let x = 5
let name = "Alice"
```

**Note**: `let` is technically optional for simple assignments (`x = 5` works), but using it is recommended for clarity. The keyword is reserved for potential future features like block-scoping or immutability.

#### Destructuring Arrays

```parsley
let arr = [1, 2, 3]
let [a, b, c] = arr             // a=1, b=2, c=3
let [first, ...rest] = [1, 2, 3, 4]  // first=1, rest=[2,3,4]
```

#### Destructuring Dictionaries

```parsley
let person = {name: "Bob", age: 25, city: "NYC"}
let {name, age} = person        // name="Bob", age=25
let {name, ...rest} = person    // name="Bob", rest={age: 25, city: "NYC"}
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

#### Scope and Binding

Parsley uses **lexical scoping** with **closure semantics**:

1. **Variables are visible** in the scope where they're defined and all nested scopes
2. **Inner scopes can modify** outer variables (closures capture by reference)
3. **Inner variables don't leak** to outer scopes

```parsley
let x = 5
let f = fn() { 
    x = 10                      // Modifies outer x
}
f()
x                               // 10 (modified by closure)

let g = fn() {
    let y = 20                  // Local to g
    y
}
g()                             // 20
// y                            // Error: y not defined in outer scope
```

---

### 4.3 Return

The `return` keyword explicitly returns a value from a function.

```parsley
let multiply = fn(a, b) {
    return a * b
}
```

**Note**: In Parsley, `return` is usually **redundant**. Functions are expressions, and the last expression's value is automatically returned:

```parsley
let multiply = fn(a, b) {
    a * b                       // Automatically returned
}
```

For early returns (guard patterns), prefer `check...else` which is more idiomatic:

```parsley
// Less idiomatic:
let validate = fn(x) {
    if (x <= 0) { return "must be positive" }
    x * 2
}

// More idiomatic:
let validate = fn(x) {
    check x > 0 else "must be positive"
    x * 2
}
```

---

### 4.4 Export

The `export` keyword makes values available to other files that import the module.

```parsley
export let greeting = "Hello"
export PI = 3.14159
export double = fn(x) { x * 2 }
```

#### Computed Exports

Use `export computed` to create exports that recalculate on each access. This is useful for exposing "live" data like database queries or current timestamps.

**Expression form:**
```parsley
export computed timestamp = @now
export computed count = items.length()
```

**Block form:**
```parsley
export computed activeUsers {
    let query = "SELECT * FROM users WHERE active = true"
    @DB.query(query)
}
```

Computed exports:
- Recalculate on **every access** (never cached)
- Look like regular exports to consumers
- Cannot accept parameters (use functions for that)

**Consumer caching:**
```parsley
import {activeUsers} from @./data.pars

// Each access recalculates
for (user in activeUsers) { print(user.name) }  // Query 1
for (user in activeUsers) { print(user.email) } // Query 2

// Cache by assigning to a variable
let snapshot = activeUsers                       // Query 3
for (user in snapshot) { print(user.name) }     // Uses snapshot
for (user in snapshot) { print(user.email) }    // Uses snapshot
```

#### Module System Overview

Parsley modules are simply `.pars` files. Any file can be imported by another, and only `export`ed values are visible to the importer. Non-exported values remain private.

**Example module** (`mathutils.pars`):

```parsley
// Private helper (not exported)
let internalHelper = fn(x) { x * x }

// Public API (exported)
export PI = 3.14159
export square = fn(x) { internalHelper(x) }
export cube = fn(x) { x * x * x }
```

---

### 4.5 Import

The `import` statement loads a module and makes its exports available.

#### Standard Library Imports

```parsley
import @std/math                      // Import as `math.floor()`, etc.
import @std/math as M                 // Import with alias as `M.floor()`
let {floor, ceil} = import @std/math  // Destructure specific exports
```

#### Custom Module Imports

Import your own `.pars` files using path literals:

```parsley
// Import the module from section 4.4
import @./mathutils.pars              // Relative to current file
import @./mathutils.pars as Utils     // With alias
let {PI, square} = import @./mathutils.pars  // Destructure exports

square(4)                             // 16
PI                                    // 3.14159
Utils.cube(3)                         // 27
```

#### Import Paths

| Path Type | Example | Description |
|-----------|---------|-------------|
| Standard lib | `@std/math` | Built-in standard library module |
| Relative | `@./utils.pars` | Relative to current file |
| Project root | `@~/lib/utils.pars` | Relative to project root |

---

## 5. Type Methods

Methods are called on values using dot notation: `value.method(args)`.

**Return Value Convention**: Most methods return a new value and do not modify the original. You must assign the result to use it. Exception: `delete()` on dictionaries mutates in place.

```parsley
let name = "alice"
name.toUpper()                  // Returns "ALICE", but name is still "alice"
let upper = name.toUpper()      // Assign to use the result
```

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

#### Display

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.toBox(opts?)` | `opts?: {style, title, maxWidth, align}` | `string` | Render value in a box with box-drawing characters |

**toBox options:**
- `style`: `"single"` (default), `"double"`, `"ascii"`, `"rounded"` - box border style
- `title`: `string` - optional title row at top of box
- `maxWidth`: `integer` - truncate content to this width (adds `...`)
- `align`: `"left"` (default), `"right"`, `"center"` - text alignment

```parsley
"hello".toBox()                 // ┌───────┐
                                // │ hello │
                                // └───────┘

"hello".toBox({style: "double"})  // ╔═══════╗
                                  // ║ hello ║
                                  // ╚═══════╝

"hello".toBox({title: "Greeting"})  // ┌──────────┐
                                    // │ Greeting │
                                    // ├──────────┤
                                    // │  hello   │
                                    // └──────────┘
```

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
| `.toBox(opts?)` | `opts?: {direction, align, style, title, maxWidth}` | `string` | Render array in a box |
| `.toHTML()` | none | `string` | Convert to HTML unordered list |
| `.toMarkdown()` | none | `string` | Convert to Markdown list |

**Format styles**: `"and"` (default), `"or"`, or any custom conjunction string.

**Available locales**: `en`, `en-US`, `en-GB`, `de`, `fr`, `es`, `it`, `pt`, `nl`, `ru`, `ja`, `zh`, `ko`. Falls back to `en` for unrecognized locales.

**toBox options:**
- `direction`: `"vertical"` (default), `"horizontal"`, `"grid"` (auto for array-of-arrays)
- `align`: `"left"` (default), `"right"`, `"center"`
- `style`: `"single"` (default), `"double"`, `"ascii"`, `"rounded"` - box border style
- `title`: `string` - optional title row at top of box
- `maxWidth`: `integer` - truncate content to this width (adds `...`)

```parsley
let arr = [3, 1, 4, 1, 5]
arr.sort()                      // [1, 1, 3, 4, 5]
arr.map(fn(x) { x * 2 })        // [6, 2, 8, 2, 10]
arr.filter(fn(x) { x > 2 })     // [3, 4, 5]
arr.reduce(fn(acc, x) { acc + x }, 0)  // 14

let items = ["apple", "banana", "cherry"]
items.format()                  // "apple, banana, and cherry"
items.format("or")              // "apple, banana, or cherry"
items.format("and", "de")       // "apple, banana und cherry"
items.format("and", "fr")       // "apple, banana et cherry"
items.format("and", "ja")       // "apple、banana、cherry"
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
| `.toBox(opts?)` | `opts?: {align, keys, style, title, maxWidth}` | `string` | Render dictionary in a box |
| `.toHTML()` | none | `string` | Convert to HTML definition list |
| `.toMarkdown()` | none | `string` | Convert to Markdown table |

**Note**: `.delete()` is the only method that mutates the original. All others return new dictionaries.

**toBox options:**
- `align`: `"left"` (default), `"right"`, `"center"`
- `keys`: `boolean` - if true, renders only keys in a horizontal row
- `style`: `"single"` (default), `"double"`, `"ascii"`, `"rounded"` - box border style
- `title`: `string` - optional title row at top of box
- `maxWidth`: `integer` - truncate values to this width (adds `...`)

#### The `.render()` Method

The `render` method interprets a raw string template and substitutes `@{...}` placeholders with values from the dictionary. Unlike simple key substitution, the content inside `@{...}` is a **full Parsley expression** evaluated with the dictionary's keys available as variables.

```parsley
let data = {name: "Sam", bananas: 10}
data.render("@{name} has @{bananas + 2} bananas.")  // "Sam has 12 bananas."
```

This is the same interpolation syntax used in raw strings (`'...'`) and `<script>` tags—use `@{expr}` for substitution and `\@` to escape a literal `@`.

```parsley
let d = {name: "Alice", age: 30}
d.keys()                        // ["name", "age"]
d.values()                      // ["Alice", 30]
d.has("name")                   // true
d.entries()                     // [{key: "name", value: "Alice"}, {key: "age", value: 30}]
d.entries("k", "v")             // [{k: "name", v: "Alice"}, {k: "age", v: 30}]

// Template rendering with expressions
let person = {first: "Ada", last: "Lovelace", born: 1815}
person.render("@{first} @{last} was born in @{born}.")
// "Ada Lovelace was born in 1815."

// Ordered insertion
let record = {first: "Alice", last: "Smith"}
record.insertAfter("first", "middle", "Jane")
// {first: "Alice", middle: "Jane", last: "Smith"}
```

---

### 5.4 Number Methods

Integer and float types share formatting methods. For mathematical operations like `abs()`, `floor()`, etc., use `@std/math`.

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.format(locale?)` | `locale?: string` (default: `"en-US"`) | `string` | Locale-formatted number |
| `.currency(code, locale?)` | `code: string`, `locale?: string` | `string` | Currency format |
| `.percent(locale?)` | `locale?: string` | `string` | Percentage format |
| `.humanize(locale?)` | `locale?: string` | `string` | Compact format (1.2K, 3.4M) |
| `.toBox()` | none | `string` | Render number in a box |

**Note**: Numbers do not have `.abs()`, `.round()`, etc. as methods. Use `@std/math` functions instead: `math.abs(-5)`, `math.round(3.7)`.

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

DateTime values are dictionaries with special properties and methods. They are created from datetime literals (`@2024-12-25`), the special `@now` literal for the current moment, or the `time()` function.

```parsley
@now                            // Current datetime
@now.year                       // Current year
@now.format("full")             // e.g., "Tuesday, January 13, 2026"
```

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
| `.toDict()` | none | `dictionary` | Get clean dictionary for reconstruction |
| `.inspect()` | none | `dictionary` | Get full dictionary with `__type` for debugging |

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
| `.toDict()` | none | `dictionary` | Get clean dictionary for reconstruction |
| `.inspect()` | none | `dictionary` | Get full dictionary with `__type` for debugging |

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
| `.toDict()` | none | `dictionary` | Get clean dictionary for reconstruction |
| `.inspect()` | none | `dictionary` | Get full dictionary with `__type` for debugging |

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
| `.toDict()` | none | `dictionary` | Get clean dictionary for reconstruction |
| `.inspect()` | none | `dictionary` | Get full dictionary with `__type` for debugging |

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
| `.toDict()` | none | `dictionary` | Get clean dictionary for reconstruction |
| `.inspect()` | none | `dictionary` | Get full dictionary with `__type` for debugging |

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
| `.repr()` | none | `string` | Get parseable literal (e.g., `"$50.00"`) |
| `.toDict()` | none | `dictionary` | Get clean dictionary for reconstruction |
| `.inspect()` | none | `dictionary` | Get debug dictionary with `__type` and raw values |

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

Table values represent structured tabular data with named columns and typed rows. Tables provide SQL-like query, aggregation, and mutation methods that operate immutably (returning new tables).

#### Creating Tables

```parsley
// Table literal (preferred) - from array of dictionaries
let t = @table [
    {name: "Alice", age: 30},
    {name: "Bob", age: 25}
]

// Table literal - from array of arrays (first row is headers)
let t = @table [
    ["name", "age"],
    ["Alice", 30],
    ["Bob", 25]
]

// CSV parsing returns Table directly
let data = "name,age\nAlice,30\nBob,25".parseCSV()  // Returns Table
let rows <== CSV(@./sales.csv)                       // Returns Table

// Database queries return Table
let users <=??=> db <SQL>SELECT * FROM users</SQL>   // Returns Table

// table() builtin constructor
let t = table([{name: "Alice"}, {name: "Bob"}])

// From single dictionary (each key becomes a row)
let {table} = import @std/table
let config = table.fromDict({debug: true, port: 8080})
```

#### Properties

| Property | Type | Description |
|----------|------|-------------|
| `.row` | `dictionary\|null` | First row as dictionary, or `null` if empty |
| `.rows` | `array` | All rows as array of dictionaries |
| `.columns` | `array` | Column names as array of strings |
| `.length` | `integer` | Number of rows |
| `.schema` | `schema\|null` | Associated schema, or `null` |

#### Query Methods

All query methods return new tables (immutable operations):

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

All return new tables (immutable—original unchanged):

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.appendRow(row)` | `row: dictionary` | `table` | Add row at end |
| `.insertRowAt(i, row)` | `i: integer`, `row: dictionary` | `table` | Insert row at index |
| `.appendCol(name, vals)` | `name: string`, `vals: array` | `table` | Add column at end |
| `.insertColAfter(after, name, vals)` | `after, name: string`, `vals: array` | `table` | Insert column after another |
| `.insertColBefore(before, name, vals)` | `before, name: string`, `vals: array` | `table` | Insert column before another |
| `.renameCol(old, new)` | `old, new: string` | `table` | Rename a column |
| `.dropCol(col)` | `col: string` | `table` | Remove a column |

#### Collection Methods

Functional-style methods for transforming and querying table data:

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.map(fn)` | `fn: function(row)` | `table` | Transform each row |
| `.find(fn)` | `fn: function(row)` | `dictionary\|null` | First row matching predicate |
| `.any(fn)` | `fn: function(row)` | `boolean` | True if any row matches |
| `.all(fn)` | `fn: function(row)` | `boolean` | True if all rows match |
| `.reduce(fn, init)` | `fn: function(acc, row)`, `init: any` | `any` | Fold rows to accumulator |
| `.groupBy(fn\|col)` | `fn: function(row)` or `col: string` | `dictionary` | Group rows by key (values are Tables) |
| `.unique(col?)` | `col?: string` | `table` | Remove duplicate rows |
| `.sortBy(fn)` | `fn: function(row)` | `table` | Sort by computed key |

**Schema behavior with collection methods:**
- `map(fn)`: If fn returns Records of same schema → preserves schema. If different schema → adopts new. If plain dicts → clears schema.
- `groupBy`: Returns `{key: Table, ...}` where each group Table has same schema as source.
- `renameCol`, `dropCol`: Clear schema (structure changed).

```parsley
let users = @table [
    {name: "Alice", age: 30, dept: "Engineering"},
    {name: "Bob", age: 25, dept: "Sales"},
    {name: "Carol", age: 35, dept: "Engineering"}
]

// Transform rows
let withBonus = users.map(fn(r) { r ++ {bonus: r.age * 100} })

// Find first match
let alice = users.find(fn(r) { r.name == "Alice" })
// {name: "Alice", age: 30, dept: "Engineering"}

// Check conditions
users.any(fn(r) { r.age > 30 })     // true
users.all(fn(r) { r.age >= 25 })    // true

// Reduce to single value
let totalAge = users.reduce(fn(acc, r) { acc + r.age }, 0)  // 90

// Group by column
let byDept = users.groupBy("dept")
// {Engineering: Table[2 rows], Sales: Table[1 row]}

// Group by computed key
let byAgeGroup = users.groupBy(fn(r) { if (r.age >= 30) "senior" else "junior" })

// Remove duplicates
let uniqueDepts = users.unique("dept")  // 2 rows (Engineering, Sales)

// Sort by computed key
let byNameLength = users.sortBy(fn(r) { r.name.length() })
```

#### Inspection Methods

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.rowCount()` | none | `integer` | Number of rows (same as `.count()`) |
| `.columnCount()` | none | `integer` | Number of columns |

#### Output Methods

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.toHTML(footer?)` | `footer?: string\|dictionary` | `string` | Convert to HTML `<table>` |
| `.toCSV()` | none | `string` | Convert to CSV string |
| `.toMarkdown()` | none | `string` | Convert to Markdown table |
| `.toBox(opts?)` | `opts?: {style, title, maxWidth, align}` | `string` | Convert to box-drawing table (CLI style) |
| `.toJSON()` | none | `string` | Convert to JSON array |
| `.toArray()` | none | `array` | Convert to array of dictionaries |

**toBox options** (same as String/Array/Dictionary):
- `style`: `"single"` (default), `"double"`, `"ascii"`, `"rounded"` - box border style
- `title`: `string` - optional title row at top of box
- `maxWidth`: `integer` - truncate cell content to this width (adds `...`)
- `align`: `"left"` (default), `"right"`, `"center"` - text alignment

```parsley
// Load CSV (returns Table directly)
let data <== CSV(@./sales.csv)

// Query operations (chainable)
let filtered = data.where(fn(r) { r.amount > 100 })
let sorted = data.orderBy("date", "desc")
let projected = data.select("name", "total")
let page = data.limit(10, 20)       // 10 rows, skip 20

// Aggregates
data.count()                        // 150
data.sum("amount")                  // 12500.00
data.avg("price")                   // 49.99
data.column("name")                 // ["Alice", "Bob", ...]

// Method chaining
let result = data
    .where(fn(r) { r.region == "North" })
    .orderBy("sales", "desc")
    .limit(5)

// Output to HTML
<div>result.toHTML()</div>
```

**Note:** `parseCSV()` returns a Table when `hasHeader=true` (the default). To get a raw array of arrays, use `parseCSV(false)`.

```parsley
// With header (default) - returns Table
let t = csvString.parseCSV()
t.count()                    // Works: returns row count
t.rows[0].name               // Access first row's name column

// Without header - returns Array of Arrays
let arr = csvString.parseCSV(false)
arr[0]                       // First row as array: ["name", "age"]
```

---

### 5.12 Schema Properties & Methods

Schemas are first-class objects that can be introspected and queried at runtime.

#### Properties

| Property | Type | Description |
|----------|------|-------------|
| `.name` | `string` | Schema name |
| `.fields` | `dictionary` | Field definitions with type info |
| `.relations` | `dictionary` | Relation definitions |
| `.[fieldName]` | `string` | Direct access returns field type |

```parsley
@schema User {
    name: string
    age: integer
    email: email
}

User.name                       // "User"
User.age                        // "integer" (shorthand field access)
User.fields                     // {name: {type: "string", ...}, age: {...}, ...}
```

#### Methods

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.title(field)` | `field: string` | `string` | Get display title for field |
| `.placeholder(field)` | `field: string` | `string\|null` | Get placeholder text for field |
| `.meta(field, key)` | `field, key: string` | `any\|null` | Get metadata value for field |
| `.fields()` | none | `array<string>` | Get all field names |
| `.visibleFields()` | none | `array<string>` | Get non-hidden, non-auto field names |
| `.enumValues(field)` | `field: string` | `array<string>` | Get enum values for field |

#### title(field)

Returns the display title for a field. Uses metadata `title` if set, otherwise converts the field name to title case.

```parsley
@schema Contact {
    firstName: string | {title: "First Name"}
    lastName: string
}

Contact.title("firstName")      // "First Name" (from metadata)
Contact.title("lastName")       // "Last Name" (auto title-cased)
```

#### placeholder(field)

Returns the placeholder text for a field, or `null` if not set.

```parsley
@schema Login {
    email: email | {placeholder: "you@example.com"}
    password: string
}

Login.placeholder("email")      // "you@example.com"
Login.placeholder("password")   // null
```

#### meta(field, key)

Returns any metadata value for a field. Useful for custom metadata beyond title/placeholder/hidden.

```parsley
@schema Product {
    price: money | {currency: "USD", min: 0}
}

Product.meta("price", "currency")   // "USD"
Product.meta("price", "min")        // 0
Product.meta("price", "missing")    // null
```

#### fields()

Returns all field names as an array (alphabetically sorted).

```parsley
@schema Person { name: string, age: integer, city: string }

Person.fields()                 // ["age", "city", "name"]
```

#### visibleFields()

Returns field names excluding `hidden` metadata fields and `auto` constraint fields. Useful for auto-generating forms and tables where auto-generated IDs and hidden fields should not appear.

```parsley
@schema User {
    id: ulid(auto)                        // excluded (auto)
    name: string
    password: string | {hidden: true}     // excluded (hidden)
    email: email
}

User.visibleFields()            // ["email", "name"]
```

#### enumValues(field)

Returns the allowed values for an enum field, or an empty array if not an enum.

```parsley
@schema Task {
    status: enum["todo", "doing", "done"]
    title: string
}

Task.enumValues("status")       // ["todo", "doing", "done"]
Task.enumValues("title")        // []
```

#### Practical Example: Dynamic Form Generation

```parsley
@schema Contact {
    name: string | {title: "Full Name", placeholder: "John Doe"}
    email: email | {title: "Email", placeholder: "john@example.com"}
    phone: phone? | {title: "Phone", placeholder: "(555) 555-5555"}
    notes: text | {title: "Notes", hidden: true}
}

// Generate form fields for visible fields only
for (field in Contact.visibleFields()) {
    <div class="form-group">
        <label>{Contact.title(field)}</label>
        <input 
            name={field} 
            placeholder={Contact.placeholder(field) ?? ""}
        />
    </div>
}
```

#### Practical Example: Dynamic Table Headers

```parsley
@schema Product {
    sku: string | {title: "SKU", hidden: true}
    name: string | {title: "Product Name"}
    price: money | {title: "Price"}
    stock: integer | {title: "In Stock"}
}

let products = @table(Product) [...]

// Generate table with proper column headers
<table>
    <thead>
        <tr>
            for (col in Product.visibleFields()) {
                <th>{Product.title(col)}</th>
            }
        </tr>
    </thead>
    <tbody>
        for (row in products.rows) {
            <tr>
                for (col in Product.visibleFields()) {
                    <td>{row[col]}</td>
                }
            </tr>
        }
    </tbody>
</table>
```

---

### 5.13 TableBinding Properties & Methods

TableBinding represents a database table bound to a schema. It provides query and mutation methods for database operations.

#### Creating a TableBinding

```parsley
@schema User {
    id: integer
    name: string
    email: email
}

let db = @sqlite(":memory:")
db.createTable(User, "users")        // Create table from schema
let users = db.bind(User, "users")   // Bind schema to table
```

#### Properties

| Property | Type | Description |
|----------|------|-------------|
| `.schema` | `schema` | The bound schema |
| `.tableName` | `string` | Database table name |

#### Query Methods

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.all()` | none | `Table` | All rows as Table |
| `.where(cond)` | `cond: dictionary` | `TableBinding` | Filter by conditions |
| `.find(id)` | `id: any` | `Record\|null` | Find row by primary key |
| `.first()` | none | `Record\|null` | First matching row |

```parsley
let allUsers = users.all()
let active = users.where({status: "active"}).all()
let user = users.find("abc-123")
```

#### Schema-Driven Mutation Methods

TableBindings support method-based CRUD operations that accept Record or Table objects directly. The `id` field is used as the primary key for update, save, and delete operations.

| Method | Argument | Returns | Description |
|--------|----------|---------|-------------|
| `.insert(record)` | Record | Record | Insert single row, return with generated ID |
| `.insert(table)` | Table | `{inserted: N}` | Insert all rows, return count |
| `.update(record)` | Record | Record | Update row by ID |
| `.update(table)` | Table | `{updated: N}` | Update all rows by ID |
| `.save(record)` | Record | Record | Upsert single row (insert or update) |
| `.save(table)` | Table | `{inserted: N, updated: M}` | Upsert all rows |
| `.delete(record)` | Record | `{deleted: 1}` | Delete row by ID |
| `.delete(table)` | Table | `{deleted: N}` | Delete all rows by ID |

##### insert(record) / insert(table)

Insert a Record (id auto-generated) or all rows from a Table:

```parsley
// Insert single Record
let user = User({name: "Alice", email: "alice@example.com"})
let inserted = users.insert(user)
log(inserted.id)                     // Generated ID

// Insert Table (bulk)
let newUsers = table([
    {name: "Bob", email: "bob@example.com"},
    {name: "Carol", email: "carol@example.com"}
])
let result = users.insert(newUsers)  // {inserted: 2}
```

##### update(record) / update(table)

Update rows by their `id` field:

```parsley
// Update single Record
let user = users.find("abc-123")
users.update(user.update({name: "Alice Smith"}))

// Update Table (bulk)
let toUpdate = table([
    {id: "abc-123", name: "Alice Smith"},
    {id: "def-456", name: "Bob Jones"}
])
users.update(toUpdate)               // {updated: 2}
```

**Error**: If Record has no `id` field, raises `DB-0016`.

##### save(record) / save(table)

Upsert—inserts if `id` is missing or null, updates if `id` exists:

```parsley
// Save new record (no id)
let user = User({name: "Alice", email: "alice@example.com"})
let saved = users.save(user)         // Inserts, returns with generated ID

// Save existing record (has id)
let user = users.find("abc-123")
users.save(user.update({name: "Alice Smith"}))  // Updates

// Bulk save (mixed insert/update)
let mixed = table([
    {name: "New User", email: "new@example.com"},        // Will insert
    {id: "abc-123", name: "Updated", email: "a@b.com"}   // Will update
])
users.save(mixed)                    // {inserted: 1, updated: 1}
```

##### delete(record) / delete(table)

Delete rows by their `id` field:

```parsley
// Delete single Record
let user = users.find("abc-123")
users.delete(user)                   // {deleted: 1}

// Delete Table (bulk)
let inactive = users.where({status: "inactive"}).all()
users.delete(inactive)               // {deleted: N}
```

**Error**: If Record has no `id` field, raises `DB-0017`.

#### Schema Matching

When passing a Record or Table with an attached schema, it must match the binding's schema:

```parsley
@schema Product { id: integer, name: string, price: money }
let product = Product({name: "Widget", price: $9.99})

// Error VAL-0022: Schema mismatch
users.insert(product)
```

Plain dictionaries are always accepted (backward compatible).

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
| `printf(template, dict)` | `template: string`, `dict: dictionary` | `null` | Print template with `@{key}` placeholders |
| `log(vals...)` | `vals: any...` | `null` | Log values (first string unquoted) |
| `logLine(vals...)` | `vals: any...` | `null` | Log with newline |

**Note**: These write to stdout. In web context, output typically doesn't appear to users.

**`log()` behavior**: First argument is displayed without quotes if it's a string (as a label), subsequent values use debug format.

**`printf()` syntax**: Unlike C-style printf, Parsley's `printf` uses template interpolation with `@{key}` placeholders that are replaced with values from the dictionary argument.

```parsley
log("user", currentUser)        // "user {name: 'Alice', ...}"
log(42, "hello")                // "42, hello"

// printf uses @{key} placeholders
printf("Hello @{name}, you are @{age} years old", {name: "Alice", age: 30})
// Output: Hello Alice, you are 30 years old
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

### 6.7 Path

| Function | Arguments | Returns | Description |
|----------|-----------|---------|-------------|
| `path(str)` | `str: string` | `path` | Create path from string |

Create path values programmatically from strings. Prefer path literals (`@./file.txt`, `@~/config`) for static paths.

```parsley
let p = path("/home/user/file.txt")
p.isAbsolute                    // true
p.components                    // ["home", "user", "file.txt"]

// Use when path comes from dynamic source
let userPath = path(request.query.file)
```

---

### 6.8 Control Flow

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

### 6.9 DateTime

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

### 6.10 URL

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

### 6.11 Duration

| Function | Arguments | Returns | Description |
|----------|-----------|---------|-------------|
| `duration(str)` | `str: string` | `duration` | Parse duration string |
| `duration(dict)` | `dict: dictionary` | `duration` | Create duration from components |

**String format**: `[sign][components]` where components use: `y` (years), `mo` (months), `w` (weeks), `d` (days), `h` (hours), `m` (minutes), `s` (seconds)

```parsley
// From string
let d1 = duration("1d")              // 1 day
let d2 = duration("2h30m")           // 2 hours 30 minutes
let d3 = duration("1y6mo")           // 1 year 6 months
let d4 = duration("-1w")             // negative 1 week

// From dictionary
let d5 = duration({days: 1})
let d6 = duration({hours: 2, minutes: 30})
let d7 = duration({years: 1, months: 6})

// Access components
d2.seconds                           // 9000
d7.months                            // 18

// Arithmetic
let tomorrow = @now + duration("1d")
let nextYear = @now + duration({years: 1})
```

**Note**: Prefer duration literals (`@1d`, `@2h30m`) for static durations. Use `duration()` when parsing dynamic input.

---

### 6.12 File Operations

These functions create file handles for reading and writing.

| Function | Arguments | Returns | Description |
|----------|-----------|---------|-------------|
| `JSON(source, opts?)` | `source: path\|url`, `opts?: dict` | `file` | JSON file handle |
| `YAML(source, opts?)` | `source: path\|url`, `opts?: dict` | `file` | YAML file handle |
| `PLN(path, opts?)` | `path: path`, `opts?: dict` | `file` | PLN file handle (Parsley Literal Notation) |
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

### 6.12 Assets

| Function | Arguments | Returns | Description |
|----------|-----------|---------|-------------|
| `asset(path)` | `path: path` | `string` | Get asset path with cache-busting hash |

```parsley
<img src={asset(@./logo.png)} alt="Logo"/>
// Produces: <img src="/assets/logo-a1b2c3d4.png" alt="Logo"/>
```

**Note**: `asset()` is primarily useful in Basil server context for cache-busting.

---

### 6.14 Serialization

Functions for serializing and deserializing Parsley values to/from PLN (Parsley Literal Notation).

| Function | Arguments | Returns | Description |
|----------|-----------|---------|-------------|
| `serialize(value)` | `value: any` | `string` | Convert value to PLN string |
| `deserialize(pln)` | `pln: string` | `any` | Parse PLN string to value |

**PLN** is a safe data format that uses Parsley syntax but only allows values—no expressions or code execution.

```parsley
// Serialize values to PLN strings
serialize(42)                   // "42"
serialize("hello")              // "\"hello\""
serialize([1, 2, 3])            // "[1, 2, 3]"
serialize({name: "Alice"})      // '{name: "Alice"}'

// Deserialize PLN strings back to values
deserialize("42")               // 42
deserialize("[1, 2, 3]")        // [1, 2, 3]
deserialize('{name: "Alice"}')  // {name: "Alice"}

// Round-trip example
let original = {user: "Bob", active: true}
let pln = serialize(original)
let restored = deserialize(pln)
// restored.user == "Bob"

// Security: expressions are rejected
deserialize("1 + 1")            // Error
deserialize("print(42)")        // Error
```

**Non-serializable types** (will produce an error):
- Functions
- File handles
- Database connections
- Modules

See [PLN manual page](manual/pln.md) for complete syntax reference.

---

## 7. Standard Library

Import with `@std/` prefix:

```parsley
import @std/math
let {floor, ceil} = import @std/math
```

**Available modules:**

| Module | Description |
|--------|-------------|
| `@std/math` | Mathematical functions and constants |
| `@std/valid` | Validation predicates |
| `@std/id` | ID generation (ULID, UUID, NanoID) |
| `@std/table` | SQL-like table operations |
| `@std/schema` | Data validation schemas |
| `@std/api` | HTTP API utilities (auth wrappers, error helpers) |
| `@std/mdDoc` | Markdown document analysis |
| `@std/dev` | Development logging (server only) |
| `@std/html` | Pre-built HTML components (server only) |

---

### 7.1 @std/math

Mathematical functions and constants. All trigonometric functions use radians.

#### Constants

| Name | Value | Description |
|------|-------|-------------|
| `PI` | 3.14159... | Pi (π) |
| `E` | 2.71828... | Euler's number |
| `TAU` | 6.28318... | Tau (2π) |

#### Rounding Functions

| Function | Arguments | Description |
|----------|-----------|-------------|
| `floor(n)` | `n: number` | Round down to integer |
| `ceil(n)` | `n: number` | Round up to integer |
| `round(n, decimals?)` | `n: number`, `decimals?: integer` | Round to nearest (optional decimal places) |
| `trunc(n)` | `n: number` | Truncate toward zero |

#### Comparison & Clamping

| Function | Arguments | Description |
|----------|-----------|-------------|
| `abs(n)` | `n: number` | Absolute value |
| `sign(n)` | `n: number` | Returns -1, 0, or 1 |
| `clamp(n, min, max)` | `n, min, max: number` | Clamp value to range |
| `min(a, b)` / `min(arr)` | Two numbers or array | Minimum value |
| `max(a, b)` / `max(arr)` | Two numbers or array | Maximum value |

#### Aggregation

All accept two arguments or an array.

| Function | Description |
|----------|-------------|
| `sum(...)` | Sum of values |
| `avg(...)` / `mean(...)` | Average (mean is alias) |
| `product(...)` | Product of values |
| `count(arr)` | Count elements |

#### Statistics (Array Only)

| Function | Arguments | Description |
|----------|-----------|-------------|
| `median(arr)` | `arr: array` | Median value |
| `mode(arr)` | `arr: array` | Most frequent value |
| `stddev(arr)` | `arr: array` | Standard deviation |
| `variance(arr)` | `arr: array` | Variance |
| `range(arr)` | `arr: array` | max - min |

#### Random

| Function | Arguments | Description |
|----------|-----------|-------------|
| `random()` | none | Random float 0.0-1.0 |
| `randomInt(max)` | `max: integer` | Random integer 0 to max-1 |
| `randomInt(min, max)` | `min, max: integer` | Random integer min to max-1 |
| `seed(n)` | `n: integer` | Seed the random generator |

#### Powers & Logarithms

| Function | Arguments | Description |
|----------|-----------|-------------|
| `sqrt(n)` | `n: number` | Square root |
| `pow(base, exp)` | `base, exp: number` | Power (base^exp) |
| `exp(n)` | `n: number` | e^n |
| `log(n)` | `n: number` | Natural logarithm |
| `log10(n)` | `n: number` | Base-10 logarithm |

#### Trigonometry

All use radians. Use `degrees()` and `radians()` for conversion.

| Function | Description |
|----------|-------------|
| `sin(n)` | Sine |
| `cos(n)` | Cosine |
| `tan(n)` | Tangent |
| `asin(n)` | Arc sine |
| `acos(n)` | Arc cosine |
| `atan(n)` | Arc tangent |
| `atan2(y, x)` | Arc tangent of y/x |

#### Angular Conversion

| Function | Arguments | Description |
|----------|-----------|-------------|
| `degrees(radians)` | `radians: number` | Convert radians to degrees |
| `radians(degrees)` | `degrees: number` | Convert degrees to radians |

#### Geometry & Interpolation

| Function | Arguments | Description |
|----------|-----------|-------------|
| `hypot(a, b)` | `a, b: number` | Hypotenuse length: √(a² + b²) |
| `dist(x1, y1, x2, y2)` | Four numbers | Distance between points |
| `lerp(a, b, t)` | `a, b, t: number` | Linear interpolation: a + (b-a)*t |
| `map(n, inMin, inMax, outMin, outMax)` | Five numbers | Map value from one range to another |

```parsley
let math = import @std/math

// Rounding
math.floor(3.7)                 // 3
math.ceil(3.2)                  // 4
math.round(3.567, 2)            // 3.57
math.trunc(-3.7)                // -3

// Comparison
math.abs(-42)                   // 42
math.sign(-5)                   // -1
math.clamp(15, 0, 10)           // 10

// Aggregation
let nums = [1, 2, 3, 4, 5]
math.sum(nums)                  // 15
math.avg(nums)                  // 3
math.min(3, 7)                  // 3
math.max(nums)                  // 5

// Statistics
math.median([1, 2, 3, 4, 100])  // 3
math.stddev([1, 2, 3, 4, 5])    // ~1.41

// Random
math.random()                   // 0.314... (random)
math.randomInt(10)              // 0-9 (random)
math.randomInt(5, 10)           // 5-9 (random)

// Powers
math.sqrt(16)                   // 4
math.pow(2, 10)                 // 1024

// Trigonometry
math.sin(math.PI / 2)           // 1
math.degrees(math.PI)           // 180

// Interpolation
math.lerp(0, 100, 0.5)          // 50
math.map(5, 0, 10, 0, 100)      // 50
```

---

### 7.2 @std/valid

Validation functions that return `true` or `false`. All validators are pure functions with no side effects.

#### Type Validators

| Function | Arguments | Description |
|----------|-----------|-------------|
| `string(v)` | `v: any` | Is string? |
| `number(v)` | `v: any` | Is number (integer or float)? |
| `integer(v)` | `v: any` | Is integer? |
| `boolean(v)` | `v: any` | Is boolean? |
| `array(v)` | `v: any` | Is array? |
| `dict(v)` | `v: any` | Is dictionary? |

#### String Validators

| Function | Arguments | Description |
|----------|-----------|-------------|
| `empty(s)` | `s: string` | Is empty or whitespace only? |
| `minLen(s, n)` | `s: string`, `n: integer` | Has at least n characters? |
| `maxLen(s, n)` | `s: string`, `n: integer` | Has at most n characters? |
| `length(s, min, max)` | `s: string`, `min, max: integer` | Length in range? |
| `matches(s, pattern)` | `s: string`, `pattern: string\|regex` | Matches pattern? |
| `alpha(s)` | `s: string` | Only letters? |
| `alphanumeric(s)` | `s: string` | Only letters and numbers? |
| `numeric(s)` | `s: string` | Only digits? |

#### Number Validators

| Function | Arguments | Description |
|----------|-----------|-------------|
| `min(n, min)` | `n, min: number` | At least min? |
| `max(n, max)` | `n, max: number` | At most max? |
| `between(n, min, max)` | `n, min, max: number` | In range [min, max]? |
| `positive(n)` | `n: number` | Greater than 0? |
| `negative(n)` | `n: number` | Less than 0? |

#### Format Validators

| Function | Arguments | Description |
|----------|-----------|-------------|
| `email(s)` | `s: string` | Valid email format? |
| `url(s)` | `s: string` | Valid URL format? |
| `uuid(s)` | `s: string` | Valid UUID format? |
| `phone(s, locale?)` | `s: string`, `locale?: string` | Valid phone number? |
| `creditCard(s)` | `s: string` | Valid credit card (Luhn check)? |
| `date(s, format?)` | `s: string`, `format?: string` | Valid date? |
| `time(s)` | `s: string` | Valid time (HH:MM or HH:MM:SS)? |
| `postalCode(s, locale?)` | `s: string`, `locale?: string` | Valid postal code? |

#### Collection Validators

| Function | Arguments | Description |
|----------|-----------|-------------|
| `contains(arr, item)` | `arr: array`, `item: any` | Array contains item? |
| `oneOf(value, options)` | `value: any`, `options: array` | Value is one of options? |

```parsley
let valid = import @std/valid

// Type checking
valid.string("hello")           // true
valid.number(42)                // true
valid.integer(3.14)             // false

// String validation
valid.empty("   ")              // true
valid.minLen("hello", 3)        // true
valid.alpha("Hello")            // true
valid.alphanumeric("abc123")    // true

// Number validation
valid.positive(5)               // true
valid.between(10, 5, 15)        // true

// Format validation
valid.email("user@example.com") // true
valid.email("invalid")          // false
valid.uuid("550e8400-e29b-41d4-a716-446655440000")  // true
valid.phone("+1-555-123-4567")  // true

// Collection validation
valid.oneOf("red", ["red", "green", "blue"])  // true
```

---

### 7.3 @std/id

ID generation functions for creating unique identifiers. All functions return strings and are thread-safe.

| Function | Arguments | Description |
|----------|-----------|-------------|
| `new()` | none | ULID-like ID (26 chars, time-sortable, Crockford Base32) |
| `uuid()` | none | UUID v4 (random, 36 chars with dashes) |
| `uuidv4()` | none | Alias for `uuid()` |
| `uuidv7()` | none | UUID v7 (time-sortable, 36 chars with dashes) |
| `nanoid(length?)` | `length?: integer` (default: 21) | NanoID (URL-safe, compact) |
| `cuid()` | none | CUID2-like (collision-resistant) |

**When to use which:**
- `new()` / `uuidv7()`: When you need sortable IDs (databases, logs)
- `uuid()` / `uuidv4()`: Standard random UUID for interoperability
- `nanoid()`: Compact URLs, short codes
- `cuid()`: Horizontal scaling, distributed systems

```parsley
let id = import @std/id

id.new()                        // "01KEQAT4553AQS0P93DXYZ"
id.uuid()                       // "550e8400-e29b-41d4-a716-446655440000"
id.uuidv7()                     // "019baead-10a5-734c-8d7e-446655440000"
id.nanoid()                     // "V1StGXR8_Z5jdHi6B-myT"
id.nanoid(10)                   // "IRFa-VaY2b"
```

---

### 7.4 @std/table

The table module provides SQL-like data manipulation for arrays of dictionaries. Tables are immutable—all operations return new tables.

#### Constructors

| Function | Arguments | Description |
|----------|-----------|-------------|
| `table.table(arr)` | `arr: array` | Create table from array of dictionaries |
| `table.table.fromDict(dict, keyCol?, valCol?)` | `dict: dictionary` | Create table from dictionary entries |

#### Query Methods

All query methods return a new Table.

| Method | Arguments | Description |
|--------|-----------|-------------|
| `where(fn)` | `fn: (row) → boolean` | Filter rows matching predicate |
| `orderBy(col, dir?)` | `col: string`, `dir?: "asc"\|"desc"` | Sort by column (default: "asc") |
| `select(cols)` | `cols: array` | Select specific columns |
| `limit(n)` | `n: integer` | Take first n rows |

#### Aggregation Methods

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `count()` | none | integer | Number of rows |
| `sum(col)` | `col: string` | number | Sum of column values |
| `avg(col)` | `col: string` | number | Average of column values |
| `min(col)` | `col: string` | number | Minimum column value |
| `max(col)` | `col: string` | number | Maximum column value |

#### Access Methods

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `rowCount()` | none | integer | Number of rows |
| `columnCount()` | none | integer | Number of columns |
| `column(name)` | `name: string` | array | Extract column as array |

#### Mutation Methods

All mutation methods return a new Table.

| Method | Arguments | Description |
|--------|-----------|-------------|
| `appendRow(row)` | `row: dictionary` | Add row at end |
| `insertRowAt(index, row)` | `index: integer`, `row: dictionary` | Insert row at position |
| `appendCol(name, values)` | `name: string`, `values: array` | Add column at end |
| `insertColAfter(after, name, values)` | `after, name: string`, `values: array` | Insert column after another |
| `insertColBefore(before, name, values)` | `before, name: string`, `values: array` | Insert column before another |

#### Export Methods

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `toCSV()` | none | string | Export as CSV |
| `toJSON()` | none | string | Export as JSON array |
| `toMarkdown()` | none | string | Export as Markdown table |
| `toBox(opts?)` | `opts?: {style, title, maxWidth, align}` | string | Export as box-drawing table (CLI style) |
| `toHTML()` | none | tag | Export as HTML table |

```parsley
let table = import @std/table

let data = [
    {name: "Alice", age: 30, dept: "Eng"},
    {name: "Bob", age: 25, dept: "Sales"},
    {name: "Carol", age: 35, dept: "Eng"}
]

let t = table.table(data)

// Query
let engineers = t.where(fn(row) { row.dept == "Eng" })
engineers.count()               // 2

let sorted = t.orderBy("age", "asc")
sorted.column("name")[0]        // "Bob"

let subset = t.select(["name", "age"])
subset.columnCount()            // 2

// Aggregation
t.sum("age")                    // 90
t.avg("age")                    // 30
t.min("age")                    // 25
t.max("age")                    // 35

// Export
t.toCSV()                       // "name,age,dept\nAlice,30,Eng\n..."
t.toMarkdown()                  // "| name | age | dept |\n..."

// From dictionary
let counts = {a: 1, b: 2, c: 3}
let t2 = table.table.fromDict(counts, "letter", "count")
t2.toCSV()                      // "letter,count\na,1\nb,2\nc,3"
```

---

### 7.5 @std/schema

Schema definitions for data validation. Define reusable schemas and validate data against them.

#### Type Factories

| Function | Arguments | Description |
|----------|-----------|-------------|
| `string(opts?)` | `opts?: {minLength?, maxLength?}` | String type |
| `email(opts?)` | `opts?: dictionary` | Email format |
| `url(opts?)` | `opts?: dictionary` | URL format |
| `phone(opts?)` | `opts?: dictionary` | Phone number format |
| `integer(opts?)` | `opts?: {min?, max?}` | Integer type |
| `number(opts?)` | `opts?: {min?, max?}` | Number type (integer or float) |
| `boolean(opts?)` | `opts?: dictionary` | Boolean type |
| `enum(values...)` | `values: string...` | One of specified values |
| `date(opts?)` | `opts?: dictionary` | Date string (YYYY-MM-DD) |
| `datetime(opts?)` | `opts?: dictionary` | Datetime string (ISO 8601) |
| `money(opts?)` | `opts?: dictionary` | Monetary value |
| `array(itemType)` | `itemType: type` | Array of specified type |
| `object(schema)` | `schema: dictionary` | Nested object |
| `id(opts?)` | `opts?: dictionary` | ID string |

#### Schema Operations

| Function | Arguments | Returns | Description |
|----------|-----------|---------|-------------|
| `define(name, fields)` | `name: string`, `fields: dictionary` | Schema | Define a named schema |
| `table(schema)` | `schema: Schema` | Table | Create table with schema validation |

#### Schema Methods

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `.validate(data)` | `data: dictionary` | ValidationResult | Validate data against schema |
| `.name` | none | string | Schema name |

#### ValidationResult

| Property | Type | Description |
|----------|------|-------------|
| `valid` | boolean | Whether validation passed |
| `errors` | array | List of `{field, message}` errors |

```parsley
let schema = import @std/schema

let User = schema.define("User", {
    name: schema.string({minLength: 1, maxLength: 50}),
    email: schema.email(),
    age: schema.integer({min: 0, max: 150}),
    role: schema.enum("user", "admin", "guest"),
    active: schema.boolean()
})

// Valid data
let result = User.validate({
    name: "Alice",
    email: "alice@example.com",
    age: 30,
    role: "user",
    active: true
})
result.valid                    // true

// Invalid data
let bad = User.validate({
    email: "not-an-email",
    age: -5
})
bad.valid                       // false
bad.errors.length()             // 2
bad.errors[0].field             // "email"
bad.errors[0].message           // "User schema: Invalid email format"
```

---

### 7.6 @std/api

HTTP API utilities for Basil handlers. Provides auth wrappers and error helpers.

#### Auth Wrappers

Wrap handler functions to add authentication requirements.

| Function | Arguments | Description |
|----------|-----------|-------------|
| `public(fn)` | `fn: function` | No authentication required |
| `adminOnly(fn)` | `fn: function` | Requires admin role |
| `roles(roles, fn)` | `roles: array`, `fn: function` | Requires any of specified roles |
| `auth(fn)` | `fn: function` | Requires any authenticated user |

#### Error Helpers

Return special error objects that Basil converts to HTTP responses.

| Function | Arguments | HTTP Status | Description |
|----------|-----------|-------------|-------------|
| `notFound(msg?)` | `msg?: string` | 404 | Resource not found |
| `forbidden(msg?)` | `msg?: string` | 403 | Access denied |
| `badRequest(msg?)` | `msg?: string` | 400 | Invalid request |
| `unauthorized(msg?)` | `msg?: string` | 401 | Authentication required |
| `conflict(msg?)` | `msg?: string` | 409 | Resource conflict |
| `serverError(msg?)` | `msg?: string` | 500 | Internal server error |

#### Redirect Helper

| Function | Arguments | Description |
|----------|-----------|-------------|
| `redirect(url, status?)` | `url: string`, `status?: integer` | HTTP redirect (default: 302) |

```parsley
let api = import @std/api

// Auth wrappers (used in route definitions)
let getUsers = api.public(fn(req) {
    // Anyone can access
    return users
})

let deleteUser = api.adminOnly(fn(req) {
    // Only admins
    return {ok: true}
})

let editProfile = api.auth(fn(req) {
    // Any logged-in user
    return profile
})

// Error responses
fn getUser(req) {
    let user = findUser(req.params.id)
    if (user == null) {
        return api.notFound("User not found")
    }
    return user
}

// Redirects
fn handleLogin(req) {
    // ... authenticate ...
    return api.redirect("/dashboard")
}

fn handleOldUrl(req) {
    return api.redirect("/new-url", 301)  // Permanent redirect
}
```

---

### 7.7 @std/mdDoc

Markdown document analysis and manipulation. Parse markdown into a queryable document object.

#### Constructor

| Function | Arguments | Returns | Description |
|----------|-----------|---------|-------------|
| `mdDoc.mdDoc(markdown)` | `markdown: string` | MdDoc | Parse markdown string |

#### Rendering Methods

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `toMarkdown()` | none | string | Render back to markdown |
| `toHTML()` | none | string | Render to HTML |

#### Query Methods

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `findAll(type)` | `type: string\|array` | array | Find all nodes of type(s) |
| `findFirst(type)` | `type: string` | node\|null | Find first node of type |
| `headings()` | none | array | All headings with `{level, text, id}` |
| `links()` | none | array | All links with `{url, title, text}` |
| `images()` | none | array | All images with `{url, alt, title}` |
| `codeBlocks()` | none | array | All code blocks with `{language, code}` |

#### Convenience Methods

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `title()` | none | string\|null | First h1 text |
| `toc()` | none | array | Table of contents entries |
| `text()` | none | string | Plain text content |
| `wordCount()` | none | integer | Word count |

#### Transform Methods

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `walk(fn)` | `fn: (node) → void` | void | Visit each node |
| `map(fn)` | `fn: (node) → node` | MdDoc | Transform nodes |
| `filter(fn)` | `fn: (node) → boolean` | MdDoc | Filter nodes |

#### AST Access

| Method | Arguments | Returns | Description |
|--------|-----------|---------|-------------|
| `ast` | none | dictionary | Raw AST |

```parsley
let mdDoc = import @std/mdDoc

let markdown = `# Welcome to My Doc

This is a **test** with [a link](https://example.com).

## Section One

Some content here.

## Section Two

![Image](photo.png "A photo")
`

let doc = mdDoc.mdDoc(markdown)

// Basic info
doc.title()                     // "Welcome to My Doc"
doc.wordCount()                 // 17
doc.text()                      // "Welcome to My Doc This is a test..."

// Extract elements
doc.headings()                  // [{level: 1, text: "Welcome to My Doc", id: "..."}, ...]
doc.links()                     // [{url: "https://example.com", title: "", text: "a link"}]
doc.images()                    // [{url: "photo.png", alt: "Image", title: "A photo"}]

// Render
doc.toHTML()                    // "<h1 id=\"welcome...\">Welcome...</h1>..."
doc.toMarkdown()                // Original markdown (reformatted)
```

---

### 7.8 @std/dev

Development logging utilities. **Requires Basil server context**—not available in standalone Parsley scripts.

#### Methods

| Method | Arguments | Description |
|--------|-----------|-------------|
| `dev.log(label, value)` | `label: string`, `value: any` | Log a value with label |
| `dev.clearLog()` | none | Clear all log entries |
| `dev.logPage()` | none | Get log page HTML |
| `dev.setLogRoute(route)` | `route: string` | Set dev log route |
| `dev.clearLogPage()` | none | Clear and return log page |

```parsley
// In a Basil handler
let dev = import @std/dev

fn handleRequest(req) {
    dev.log("request", req.params)
    dev.log("user", currentUser)
    // ...
    return response
}
```

> **Note:** The dev module is for debugging during development. Log output is visible at the configured dev log route (typically `/_dev/log`).

---

### 7.9 @std/html

Pre-built HTML components. **Requires Basil server context**—not available in standalone Parsley scripts.

#### Layout Components

| Component | Description |
|-----------|-------------|
| `Page` | Full HTML page wrapper |
| `Head` | HTML `<head>` section |

#### Form Components

| Component | Description |
|-----------|-------------|
| `Form` | Form wrapper |
| `TextField` | Text input with label |
| `TextareaField` | Multi-line text input |
| `SelectField` | Dropdown select |
| `RadioGroup` | Radio button group |
| `CheckboxGroup` | Checkbox group |
| `Checkbox` | Single checkbox |
| `Button` | Button element |

#### Navigation Components

| Component | Description |
|-----------|-------------|
| `Nav` | Navigation wrapper |
| `Breadcrumb` | Breadcrumb navigation |
| `SkipLink` | Accessibility skip link |

#### Media Components

| Component | Description |
|-----------|-------------|
| `Img` | Image with accessibility |
| `Iframe` | Embedded iframe |
| `Figure` | Figure with caption |
| `Blockquote` | Block quotation |

#### Utility Components

| Component | Description |
|-----------|-------------|
| `A` | Anchor link |
| `Abbr` | Abbreviation |
| `Icon` | Icon element |
| `SrOnly` | Screen reader only text |

#### Time Components

| Component | Description |
|-----------|-------------|
| `Time` | Time element |
| `LocalTime` | Localized time |
| `TimeRange` | Time range display |
| `RelativeTime` | Relative time (e.g., "2 hours ago") |

#### Data Components

| Component | Description |
|-----------|-------------|
| `DataTable` | Data table with sorting/pagination |

```parsley
// In a Basil template
let html = import @std/html

<html.Page title="My App">
    <html.Head>
        <link rel="stylesheet" href="/styles.css"/>
    </html.Head>
    <main>
        <html.Form action="/submit" method="post">
            <html.TextField name="email" label="Email" type="email" required=true/>
            <html.TextareaField name="message" label="Message"/>
            <html.Button type="submit">"Send"</html.Button>
        </html.Form>
    </main>
</html.Page>
```

> **Note:** See the Basil HTML Components documentation for detailed component props and usage.

---

## 8. Tags (HTML/XML)

Tags are first-class values that render to HTML strings. Unlike JSX (React), Parsley tags do not require quotes around attribute values for simple strings, and string content inside tags must be quoted.

**Key differences from JSX/React:**
- Attribute values don't need `{...}` for simple strings: `class="container"` not `class={"container"}`
- String content must be quoted: `<p>"Hello"</p>` not `<p>Hello</p>`
- Self-closing tags MUST use `/>`: `<br/>` not `<br>`

### 8.1 Self-Closing Tags

**Must use `/>` syntax** (unlike HTML5 where `<br>` is valid):

```parsley
<br/>
<hr/>
<img src="photo.jpg" alt="A photo"/>
<input type="text" name="email"/>
```

---

### 8.2 Pair Tags

Text content must be quoted. Unquoted text is interpreted as variable references:

```parsley
<p>"Hello, World!"</p>         // Literal string
<h1>"Welcome"</h1>             // Literal string

let message = "Dynamic content"
<p>message</p>                  // Variable reference
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

### 8.8 Form Binding

Parsley provides special attributes to bind HTML form elements to schema-validated records.

#### Form Context with `@record`

The `@record` attribute establishes a form context:

```parsley
<form @record={userRecord} method="POST">
    // Form elements can now use @field binding
</form>
```

The `@record` attribute is removed from output — it's a compile-time directive.

#### Input Binding with `@field`

The `@field` attribute binds an input to a schema field:

```parsley
<form @record={form} method="POST">
    <input @field="name"/>
    <input @field="email"/>
</form>
```

This automatically sets: `name`, `value`, `type` (from schema), constraint attributes (`required`, `minlength`, etc.), accessibility attributes (`aria-invalid`, `aria-describedby`), and `autocomplete` (derived from type/field name or metadata).

#### Autocomplete Derivation

The `autocomplete` attribute is automatically derived:

- **By type**: `email` → `"email"`, `phone` → `"tel"`, `url` → `"url"`
- **By field name**: `firstName` → `"given-name"`, `password` → `"current-password"`, etc.
- **By metadata**: Override with `| {autocomplete: "shipping street-address"}`

#### Form Binding Elements

| Element | Purpose | Example |
|---------|---------|---------|
| `<input @field="name"/>` | Text input bound to field | Sets name, value, type, constraints |
| `<label @field="name"/>` | Label from field metadata | Renders `<label for="name">Full Name</label>` |
| `<error @field="name"/>` | Validation error (if any) | Renders `<span class="error">...</span>` or nothing |
| `<select @field="status"/>` | Dropdown for enum fields | Auto-generates `<option>` elements |
| `<val @field="name" @key="help"/>` | Metadata value | Renders help text, hints, etc. |

**Example:**

```parsley
<form @record={form} method="POST">
    <div class="field">
        <label @field="email"/>
        <input @field="email"/>
        <error @field="email"/>
        <val @field="email" @key="help" @tag="small"/>
    </div>
</form>
```

Use `@tag` to change the output element type:

```parsley
<label @field="email" @tag="span"/>   // Renders <span>Email</span>
<error @field="email" @tag="div"/>    // Renders <div class="error">...</div>
```

See the [Record manual page](manual/builtins/record.md#form-binding) for complete documentation.

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

Parsley provides structured error handling through the `try` expression and `fail` function.

### 10.1 The `try` Expression

The `try` expression catches certain errors and returns them as values instead of terminating execution. It wraps the result in a dictionary with `result` and `error` fields.

**Syntax**: `try` only accepts **function calls** or **method calls**, not arbitrary expressions or blocks.

```parsley
let result = try someFunction(args)
// Returns: {result: value, error: null} on success
// Returns: {result: null, error: "message"} on catchable error

let result2 = try obj.method(args)
// Same pattern for method calls
```

#### Success Case

When the function succeeds, `result` contains the return value and `error` is `null`:

```parsley
let add = fn(a, b) { a + b }
let res = try add(2, 3)
res.result                      // 5
res.error                       // null
```

#### Error Case

When a catchable error occurs, `result` is `null` and `error` contains the error message:

```parsley
let validate = fn(x) {
    check x > 0 else fail("must be positive")
    x * 2
}

let res = try validate(-5)
res.result                      // null
res.error                       // "must be positive"
```

#### Pattern: Check and Handle

Use destructuring with conditionals for clean error handling:

```parsley
let {result, error} = try riskyOperation()
if (error) {
    <div class="error">"Operation failed: " + error</div>
} else {
    <div class="success">"Result: " + toString(result)</div>
}
```

#### Pattern: Default with Null Coalescing

Use `??` to provide fallback values:

```parsley
let result = (try parseJSON(input)).result ?? {}
let data = (try loadConfig()).result ?? {default: true}
```

---

### 10.2 The `fail` Function

Use `fail(message)` to create a catchable error. This is the primary way to signal errors in validation and business logic.

```parsley
let validateEmail = fn(email) {
    check email.includes("@") else fail("Invalid email format")
    check email.length() > 3 else fail("Email too short")
    email
}

let result = try validateEmail("bad")
result.error                    // "Email too short"
```

**Important**: `fail()` creates **catchable** errors (class: "value"). They can be caught by `try` expressions.

---

### 10.3 Catchable vs Non-Catchable Errors

Not all errors can be caught by `try`. Parsley distinguishes between:

- **Catchable errors** — External/runtime errors that may occur despite correct code (network failures, invalid user input, file not found)
- **Non-catchable errors** — Developer errors that indicate bugs (type mismatches, wrong number of arguments, undefined variables)

#### Catchable Error Classes

These errors **can** be caught by `try`:

| Class | Examples |
|-------|----------|
| **Value** | Created by `fail()`, empty required fields |
| **Format** | Invalid URL, malformed JSON, bad date string |
| **IO** | File not found, permission denied |
| **Network** | HTTP request failure, timeout |
| **Database** | Connection failed, query error |
| **Security** | Access denied, authentication required |

```parsley
// These CAN be caught:
try url("not a valid url")      // Format error - invalid URL
try fail("custom error")         // Value error
try readFile(@./missing.txt)    // IO error - file not found
```

#### Non-Catchable Errors

These errors **cannot** be caught by `try` — they propagate and terminate execution:

| Class | Examples |
|-------|----------|
| **Type** | Wrong type passed to function or method |
| **Arity** | Wrong number of function arguments |
| **Undefined** | Variable, function, or method not found |
| **Index** | Array index out of bounds |
| **Operator** | Invalid operation (e.g., adding incompatible types) |
| **State** | Invalid state transition |

```parsley
// These CANNOT be caught - they propagate:
try unknownFunction()           // Undefined error - propagates
try "text".split(123)           // Type error - propagates
try someFunc()                  // Arity error if wrong args - propagates
```

**Why?** Non-catchable errors indicate bugs in your code. They should fail loudly during development so you fix them, not be silently caught at runtime.

---

### 10.4 Error Prevention

#### Check Guards

Use `check...else` for validation with early returns:

```parsley
let processOrder = fn(order) {
    check order else fail("Order required")
    check order.items else fail("Order must have items")
    check order.total > 0 else fail("Order total must be positive")
    // Process order...
}
```

#### Optional Index Access

Use `[?index]` to return `null` instead of erroring on missing indices:

```parsley
let arr = [1, 2, 3]
arr[?99]                        // null (no error)
arr[99]                         // Error: index out of bounds

let user = {name: "Alice"}
user[?"email"]                  // null (no error)
user.email                      // null (null propagation, no error)
```

#### Null Coalescing

Use `??` to provide default values:

```parsley
let name = user.name ?? "Anonymous"
let config = loadConfig() ?? {default: true}
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
| `money` | `$12.34`, `EUR#50` | `amount`, `currency`, `scale` | `format`, `split`, `abs`, `negate` |
| `path` | `@./file.txt` | `segments`, `extension`, etc. | `match`, `toURL` |
| `url` | `@https://...` | `scheme`, `host`, `query`, etc. | `origin`, `pathname` |
| `regex` | `/pattern/flags` | `pattern`, `flags` | `test`, `replace` |
| `table` | `CSV(@./data.csv)` | `row`, `rows`, `columns` | `where`, `orderBy`, `toHTML` |

**Note**: Numbers do not have math methods like `abs()`. Use `@std/math` for mathematical operations.

---

## Appendix B: Method Reference

### String Methods (27 methods)

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
| `toBox(opts?)` | 0-1 | Render in box |

### Array Methods (19 methods)

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
| `toBox(opts?)` | 0-1 | Render in box (direction, align, style, title, maxWidth) |
| `shuffle()` | 0 | Random order |
| `pick(n?)` | 0-1 | Random element(s) |
| `take(n)` | 1 | n unique random |
| `has(item)` | 1 | Contains item? |
| `hasAny(arr)` | 1 | Contains any? |
| `hasAll(arr)` | 1 | Contains all? |
| `insert(i, val)` | 2 | Insert at index |

### Dictionary Methods (11 methods)

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
| `toBox(opts?)` | 0-1 | Render in box (align, keys, style, title, maxWidth) |

### Number Methods (5 methods)

| Method | Arity | Description |
|--------|-------|-------------|
| `format(locale?)` | 0-1 | Locale format |
| `currency(code, locale?)` | 1-2 | Currency format |
| `percent(locale?)` | 0-1 | Percentage format |
| `humanize(locale?)` | 0-1 | Compact format (1.2K) |
| `toBox(opts?)` | 0-1 | Render in box |

### Boolean Methods (1 method)

| Method | Arity | Description |
|--------|-------|-------------|
| `toBox()` | 0 | Render in box |

### Null Methods (1 method)

| Method | Arity | Description |
|--------|-------|-------------|
| `toBox()` | 0 | Render in box |

### Table Methods (32 methods)

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
| `renameCol(old, new)` | 2 | Rename a column |
| `dropCol(col)` | 1 | Remove a column |
| `map(fn)` | 1 | Transform each row |
| `find(fn)` | 1 | First row matching predicate |
| `any(fn)` | 1 | True if any row matches |
| `all(fn)` | 1 | True if all rows match |
| `reduce(fn, init)` | 2 | Fold rows to accumulator |
| `groupBy(fn\|col)` | 1 | Group rows by key |
| `unique(col?)` | 0-1 | Remove duplicate rows |
| `sortBy(fn)` | 1 | Sort by computed key |
| `rowCount()` | 0 | Number of rows |
| `columnCount()` | 0 | Number of columns |
| `toHTML(footer?)` | 0-1 | Convert to HTML |
| `toCSV()` | 0 | Convert to CSV |
| `toMarkdown()` | 0 | Convert to Markdown |
| `toJSON()` | 0 | Convert to JSON |

### TableBinding Methods (8 methods)

| Method | Arity | Description |
|--------|-------|-------------|
| `all()` | 0 | Get all rows as Table |
| `where(cond)` | 1 | Filter by conditions |
| `find(id)` | 1 | Find row by primary key |
| `first()` | 0 | First matching row |
| `insert(record\|table)` | 1 | Insert Record or Table |
| `update(record\|table)` | 1 | Update by ID |
| `save(record\|table)` | 1 | Upsert (insert or update) |
| `delete(record\|table)` | 1 | Delete by ID |
