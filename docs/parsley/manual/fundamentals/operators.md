---
id: man-pars-operators
title: Operators
system: parsley
type: fundamentals
name: operators
created: 2026-02-05
version: 0.2.0
author: Basil Team
keywords:
  - operators
  - arithmetic
  - comparison
  - logical
  - set operations
  - pattern matching
  - null coalescing
  - membership
  - range
  - precedence
---

# Operators

Parsley's operators mostly work as you'd expect from other languages, with a few notable additions: set operations on arrays and dictionaries, regex matching with `~`, null coalescing with `??`, inclusive ranges with `..`, and a family of I/O operators for files and databases.

## Arithmetic

```parsley
5 + 3               // 8
10 - 4              // 6
6 * 7               // 42
7 / 2               // 3       (integer division truncates)
7 / 2.0             // 3.5     (mixed → float)
7 % 3               // 1
-5                  // -5      (unary negation)
```

Mixed integer/float operations promote to float. Division by zero is a runtime error.

### String & Array Arithmetic

Several operators are overloaded for strings and arrays:

```parsley
"hello" + " world"  // "hello world"   (string concatenation)
"ha" * 3            // "hahaha"         (string repetition)
10 + "px"           // "10px"           (auto-coercion to string)

[1, 2] ++ [3, 4]    // [1, 2, 3, 4]    (array concatenation)
[1, 2] * 3          // [1, 2, 1, 2, 1, 2]
[1,2,3,4,5] / 2     // [[1, 2], [3, 4], [5]]  (chunking)
```

> ⚠️ `++` is the array/dictionary merge operator, not string concatenation. On non-arrays it wraps each side: `"a" ++ "b"` → `["a", "b"]`.

> ⚠️ `+` with a string on either side converts the other operand to a string and concatenates. `10 + "px"` is `"10px"`, not an error.

### Dictionary Merge

`++` merges dictionaries. Right-side values win on key conflicts:

```parsley
{a: 1, b: 2} ++ {b: 3, c: 4}  // {a: 1, b: 3, c: 4}
```

### Path & URL Arithmetic

`+` joins path and URL segments:

```parsley
@/usr/local + "bin"    // @/usr/local/bin
@./config + "app.json" // @./config/app.json
```

### Money Arithmetic

Money supports `+`, `-` (same currency only), and `*`, `/` with scalars. Uses banker's rounding. See [Money](../builtins/money.md).

### DateTime Arithmetic

DateTime values support `+` and `-` with integers (days) and durations. See [DateTime](../builtins/datetime.md).

## Comparison

```parsley
5 == 5               // true
5 != 3               // true
3 < 5                // true
5 > 3                // true
3 <= 3               // true
5 >= 5               // true
```

Works on numbers, strings, money, and datetimes. Equality (`==`, `!=`) works on all types.

> ⚠️ String comparison uses **natural sort order**: `"file2" < "file10"` is `true`. This means numeric substrings within strings are compared by value, not lexicographically.

## Logical

```parsley
!true                // false
not false            // true
true & false         // false
true | false         // true
true and false       // false
true or false        // true
true && false        // false   (alias for &)
true || false        // true    (alias for |)
```

All six forms (`&`/`&&`/`and`, `|`/`||`/`or`) are equivalent. `and` has higher precedence than `or`:

```parsley
true | false & false   // true   — parsed as: true | (false & false)
```

Logical operators use truthiness (see [Booleans](../builtins/booleans.md) for the full list of falsy values).

### Null Coalescing

`??` returns the left side unless it is `null`, in which case it evaluates and returns the right side. Short-circuits — the right side is not evaluated if the left is non-null.

```parsley
null ?? "default"    // "default"
false ?? "default"   // false      (not null, so left wins)
0 ?? "default"       // 0          (not null, so left wins)
```

> ⚠️ `??` checks for `null` only, not general falsiness. Use `||` if you want to fall through on any falsy value.

## Membership

`in` and `not in` test membership across arrays, dictionaries, and strings:

```parsley
2 in [1, 2, 3]           // true    (element in array)
"x" in {x: 1, y: 2}     // true    (key in dictionary)
"ell" in "hello"          // true    (substring in string)
"z" not in [1, 2, 3]     // true
5 in null                 // false   (null-safe, never errors)
```

## Set Operations on Arrays

`&`, `|`, and `-` perform set operations when both operands are arrays:

```parsley
[1,2,3] & [2,3,4]   // [2, 3]         (intersection)
[1,2,3] | [2,3,4]   // [1, 2, 3, 4]   (union, deduplicated)
[1,2,3] - [2,3]     // [1]             (difference)
```

These also work on dictionaries (matching by key):

```parsley
let d = {a: 1, b: 2, c: 3}
d & {b: 99, c: 99}  // {b: 2, c: 3}   (intersection, values from left)
d - {b: 0}          // {a: 1, c: 3}    (difference, values in right ignored)
```

> ⚠️ `&` and `|` on arrays perform set operations. On non-array values they are boolean `and`/`or`. This overloading is resolved by operand type at runtime.

## Pattern Matching

`~` matches a string against a regex and returns an array of matches (or `null` on no match). `!~` returns a boolean.

```parsley
"hello-123" ~ /([a-z]+)-(\d+)/
// ["hello-123", "hello", "123"]  — [0] is full match, [1..] are capture groups

"abc" ~ /z/          // null     (no match)
"abc" !~ /z/         // true     (no match → true)
"abc" !~ /b/         // false    (match → false)
```

## Schema Checking

`is` and `is not` test whether a record or table was created from a given schema:

```parsley
record is User       // true if record's schema is User
record is not User   // negated
```

Non-record/table values always return `false` (no error). See [Data Model](../fundamentals/data-model.md).

## Range

`..` creates an inclusive integer range. Supports ascending and descending:

```parsley
1..5                 // [1, 2, 3, 4, 5]
5..1                 // [5, 4, 3, 2, 1]
```

## Indexing & Slicing

```parsley
let a = [10, 20, 30]
a[0]                 // 10
a[-1]                // 30       (negative index from end)
a[0:2]               // [10, 20] (slice, end exclusive)

let s = "hello"
s[1]                 // "e"
s[-2:]               // "lo"
```

### Optional Access

`[?n]` returns `null` instead of erroring on out-of-bounds:

```parsley
let a = [1, 2, 3]
a[?10]               // null     (instead of index error)
a[?0]                // 1        (works normally when in bounds)
```

Dictionary access always returns `null` for missing keys — no optional form needed.

## Spread

`...` is used in two contexts:

**Rest in destructuring** — collects remaining elements:

```parsley
let [first, ...rest] = [1, 2, 3, 4]
// first = 1, rest = [2, 3, 4]

let {a, ...rest} = {a: 1, b: 2, c: 3}
// a = 1, rest = {b: 2, c: 3}
```

**Spread in tags** — expands a dictionary into tag attributes:

```parsley
let attrs = {class: "card", id: "main"}
<div ...attrs>"content"</div>
```

## I/O Operators

These operators are syntactic sugar for file and database operations. They are covered in detail in their respective manual pages.

### File I/O

| Operator | Meaning | Example |
|----------|---------|---------|
| `<==` | Read from file | `data <== @./file.txt` |
| `<=/=` | Fetch from URL | `data <=/= @https://api.example.com` |
| `==>` | Write to file | `data ==> @./output.txt` |
| `==>>` | Append to file | `line ==>> @./log.txt` |

See [File I/O](../features/file-io.md).

### Database

| Operator | Meaning | Example |
|----------|---------|---------|
| `<=?=>` | Query one row | `db <=?=> "SELECT ..."` |
| `<=??=>` | Query many rows | `db <=??=> "SELECT ..."` |
| `<=!=>` | Execute (INSERT, etc.) | `db <=!=> "INSERT ..."` |

See [Database](../features/database.md).

### Command Execution

| Operator | Meaning | Example |
|----------|---------|---------|
| `<=#=>` | Execute shell command | `result <=#=> "ls -la"` |

See [Shell Commands](../features/commands.md).

## Precedence Table

From lowest to highest:

| Precedence | Operators |
|------------|-----------|
| 1 (lowest) | `,` |
| 2 | `\|` `\|\|` `or` `??` |
| 3 | `&` `&&` `and` |
| 4 | `==` `!=` `~` `!~` `in` `not in` `is` `<=?=>` `<=??=>` `<=!=>` |
| 5 | `<` `>` `<=` `>=` |
| 6 | `+` `-` `..` |
| 7 | `++` |
| 8 | `*` `/` `%` |
| 9 | `-x` `!x` `not x` (prefix) |
| 10 | `[]` `.` (index/access) |
| 11 (highest) | `()` (call) |

## Key Differences from Other Languages

- **`++` merges arrays/dicts**, not increment. There is no increment operator.
- **`&` `|` are overloaded**: boolean logic on scalars, set operations on arrays/dicts.
- **`??` is null-only**, not falsy-coalescing. `false ?? x` returns `false`.
- **`~` returns match data** (array or null), not a boolean. Use `!~` for a boolean "does not match" test.
- **String `<` `>` use natural sort**: `"file2" < "file10"` is `true`.
- **Integer division truncates**: `7 / 2` is `3`. Mix with a float for decimal result: `7 / 2.0`.
- **No ternary operator**: use `if (cond) a else b` (it's an expression).

## See Also

- [Booleans & Null](../builtins/booleans.md) — truthiness rules, null coalescing details
- [Strings](../builtins/strings.md) — string methods and regex replace
- [Variables & Binding](variables.md) — destructuring with rest
- [Control Flow](control-flow.md) — `if`/`else` expressions, `for` loops
- [File I/O](../features/file-io.md) — `<==`, `==>` operators
- [Database](../features/database.md) — `<=?=>`, `<=??=>`, `<=!=>` operators