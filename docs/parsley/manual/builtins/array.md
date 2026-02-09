---
id: array
title: Arrays
system: parsley
type: builtin
name: array
created: 2025-12-16
version: 1.0.0
author: Basil Team
keywords:
  - array
  - list
  - collection
  - sequence
  - iteration
  - sort
---

# Arrays

Arrays are ordered, mixed-type collections and the workhorse of data processing in Parsley. They support indexing, slicing, functional operations, set algebra, random sampling, locale-aware formatting, and more.

```parsley
[1, "hello", true, null, £5.00]     // mixed types, including money
[[1, 2], [3, 4]]                     // nested arrays
[]                                   // empty (falsy)
```

## Operators

### Indexing & Slicing

```parsley
let cities = ["London", "Paris", "Tokyo"]
cities[0]                            // "London"
cities[-1]                           // "Tokyo" (negative = from end)
cities[?10]                          // null (optional access, no error)

let n = [10, 20, 30, 40, 50]
n[1:3]                               // [20, 30] (start inclusive, end exclusive)
n[:2]                                // [10, 20]
n[2:]                                // [30, 40, 50]
```

> ⚠️ **Optional indexing** `[?n]` returns `null` instead of an error on out-of-bounds access. Most languages don't have this.

### Concatenation & Repetition

```parsley
[1, 2] ++ [3, 4]                    // [1, 2, 3, 4]
[1, 2] * 3                          // [1, 2, 1, 2, 1, 2]
```

> ⚠️ **`++` concatenates arrays, `+` does not.** `"a" ++ "b"` produces `["a", "b"]` — the `++` operator always creates/extends arrays. Use `+` for string concatenation.

### Chunking with `/`

The division operator chunks an array into groups — unique to Parsley:

```parsley
[1, 2, 3, 4, 5, 6, 7, 8, 9] / 3    // [[1, 2, 3], [4, 5, 6], [7, 8, 9]]
[1, 2, 3, 4, 5] / 2                 // [[1, 2], [3, 4], [5]]
```

This is useful for pagination, grid layouts, and batch processing.

### Set Operations

Logical operators become set operations on arrays:

```parsley
[1, 2, 3] && [2, 3, 4]              // [2, 3] (intersection)
[1, 2] || [2, 3]                    // [1, 2, 3] (union)
[1, 2, 3] - [2]                     // [1, 3] (subtraction)
```

### Membership

```parsley
2 in [1, 2, 3]                      // true
5 not in [1, 2, 3]                   // true
"admin" in null                      // false (null-safe, no error)
```

### Ranges

The `..` operator creates arrays of sequential integers:

```parsley
1..5                                 // [1, 2, 3, 4, 5]
```

## `for` Loops Return Arrays

This is one of Parsley's most distinctive features: `for` is an **expression** that returns an array, making it a built-in `map` and `filter`:

```parsley
let doubled = for (n in [1, 2, 3]) { n * 2 }       // [2, 4, 6]

let evens = for (n in 1..10) { if (n % 2 == 0) { n } }  // [2, 4, 6, 8, 10]

// With index
let labeled = for (i, city in ["London", "Paris"]) {
    `{i + 1}. {city}`
}
// ["1. London", "2. Paris"]
```

> ⚠️ If the body returns `null` (e.g. an `if` with no `else`), that element is **omitted** from the result — this is how filtering works. Use `stop` and `skip` instead of `break` and `continue`.

Throughout the methods below, many examples show both `.method(fn)` and `for` style — use whichever reads better for your case.

## Methods

### filter

```parsley
[1, 2, 3, 4, 5].filter(fn(x) { x > 2 })            // [3, 4, 5]

// Equivalent with for:
for (x in [1, 2, 3, 4, 5]) { if (x > 2) { x } }    // [3, 4, 5]
```

### format

Format an array as a human-readable list, with locale support. This is unusual — most languages require a library for this:

```parsley
["Alice", "Bob", "Charlie"].format("and")             // "Alice, Bob, and Charlie"
["coffee", "tea", "milk"].format("or")                 // "coffee, tea, or milk"
[1, 2, 3].format("unit")                              // "1, 2, 3"

// Locale-aware:
["Alice", "Bob", "Charlie"].format("and", "Fr")        // "Alice, Bob et Charlie"
```

### insert

Insert an element at a specific index, returning a new array. Supports negative indices:

```parsley
["a", "c"].insert(1, "b")                             // ["a", "b", "c"]
["a", "b"].insert(-1, "x")                            // ["a", "x", "b"]
```

### join

```parsley
["Hello", "world"].join(" ")                           // "Hello world"
[1, 2, 3].join("-")                                    // "1-2-3"
```

### length

```parsley
[10, 20, 30].length()                                 // 3
```

### map

```parsley
[1, 2, 3].map(fn(x) { x * 2 })                       // [2, 4, 6]

// Equivalent with for:
for (x in [1, 2, 3]) { x * 2 }                        // [2, 4, 6]

// Extract fields from dictionaries:
let users = [{name: "Alice", age: 30}, {name: "Bob", age: 25}]
users.map(fn(u) { u.name })                            // ["Alice", "Bob"]
```

### pick

Select random elements **with replacement** (duplicates possible):

```parsley
["red", "green", "blue"].pick(2)                       // e.g. ["green", "red"]
```

### reduce

Accumulate a result across elements. Works naturally with Parsley's money type:

```parsley
[1, 2, 3, 4].reduce(fn(sum, x) { sum + x }, 0)       // 10

[£10.50, £5.25, £12.00].reduce(fn(total, p) { total + p }, £0.00)  // £27.75
```

### reorder

Reshape arrays of dictionaries — reorder, select, or rename keys in bulk. Useful for preparing database/API results for display or export:

**With an array argument** — select and reorder keys:

```parsley
let users = [{name: "Alice", age: 30, city: "London"}, {name: "Bob", age: 25, city: "Paris"}]
users.reorder(["city", "name"])
```

**Result:** `[{city: "London", name: "Alice"}, {city: "Paris", name: "Bob"}]`

**With a dictionary argument** — rename and reorder:

```parsley
let data = [{first_name: "Alice", last_name: "Smith"}, {first_name: "Bob", last_name: "Jones"}]
data.reorder({name: "first_name", surname: "last_name"})
```

**Result:** `[{name: "Alice", surname: "Smith"}, {name: "Bob", surname: "Jones"}]`

Non-dictionary elements in the array are left unchanged.

### reverse

```parsley
[1, 2, 3].reverse()                                   // [3, 2, 1]
```

### shuffle

Randomly reorder elements (Fisher-Yates):

```parsley
[1, 2, 3, 4, 5].shuffle()                             // e.g. [3, 5, 1, 4, 2]
```

### sort

Sort in natural order:

```parsley
[3, 1, 4, 1, 5].sort()                                // [1, 1, 3, 4, 5]
["banana", "apple", "cherry"].sort()                   // ["apple", "banana", "cherry"]
```

> ⚠️ **Natural sort order:** `["10 banana", "9 apple", "100 cherry"].sort()` → `["9 apple", "10 banana", "100 cherry"]`. Numbers embedded in strings are compared numerically, not lexicographically. This is almost always what you want but differs from most languages.

### sortBy

Sort by a derived key. Sort is stable — equal elements preserve their original order:

```parsley
let users = [{name: "Alice", age: 30}, {name: "Bob", age: 25}]
users.sortBy(fn(u) { u.age })
// [{name: "Bob", age: 25}, {name: "Alice", age: 30}]

["hello", "a", "goodbye"].sortBy(fn(s) { s.length() })
// ["a", "hello", "goodbye"]
```

### take

Select random elements **without replacement** (each picked at most once):

```parsley
["♠", "♥", "♦", "♣"].take(2)                          // e.g. ["♥", "♣"]
```

### toCSV

```parsley
[["Name", "Age"], ["Alice", 30], ["Bob", 25]].toCSV()
```

**Result:**
```/dev/null/output.csv#L1-3
Name,Age
Alice,30
Bob,25
```

### toJSON

```parsley
[{"id": 1, "name": "Alice"}, {"id": 2, "name": "Bob"}].toJSON()
```

**Result:** `[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}]`

### toBox

Render the array in a box with box-drawing characters. Useful for CLI output and debugging:

```parsley
["apple", "banana", "cherry"].toBox()
```

**Result:**

```/dev/null/output.txt#L1-7
┌────────┐
│ apple  │
├────────┤
│ banana │
├────────┤
│ cherry │
└────────┘
```

#### Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `direction` | string | `"vertical"` | Layout: `"vertical"`, `"horizontal"`, `"grid"` |
| `align` | string | `"left"` | Text alignment: `"left"`, `"right"`, `"center"` |
| `style` | string | `"single"` | Box border style: `"single"`, `"double"`, `"ascii"`, `"rounded"` |
| `title` | string | none | Title row centered at top of box |
| `maxWidth` | integer | none | Truncate content to this width (adds `...`) |

#### Direction Examples

```parsley
[1, 2, 3].toBox({direction: "horizontal"})
```

```/dev/null/output.txt#L1-3
┌───┬───┬───┐
│ 1 │ 2 │ 3 │
└───┴───┴───┘
```

Grid layout (auto-detected for array of arrays):

```parsley
[[1, 2, 3], [4, 5, 6]].toBox()
```

```/dev/null/output.txt#L1-5
┌───┬───┬───┐
│ 1 │ 2 │ 3 │
├───┼───┼───┤
│ 4 │ 5 │ 6 │
└───┴───┴───┘
```

#### Style Examples

```parsley
["A", "B", "C"].toBox({style: "double", direction: "horizontal"})
```

```/dev/null/output.txt#L1-3
╔═══╦═══╦═══╗
║ A ║ B ║ C ║
╚═══╩═══╩═══╝
```

#### Title Example

```parsley
[1, 2, 3].toBox({title: "Numbers"})
```

```/dev/null/output.txt#L1-9
┌─────────┐
│ Numbers │
├─────────┤
│    1    │
├─────────┤
│    2    │
├─────────┤
│    3    │
└─────────┘
```

## Examples

### Chaining Operations

```parsley
let people = [
  {name: "Alice", age: 30},
  {name: "Charlie", age: 22},
  {name: "Bob", age: 28}
]
let adults = people
  .filter(fn(p) { p.age > 25 })
  .map(fn(p) { p.name })
  .sort()
// ["Alice", "Bob"]
```

### Money Arithmetic

```parsley
let prices = [£10.00, £15.50, £8.25, £12.75]
let total = prices.reduce(fn(sum, p) { sum + p }, £0.00)
let average = total / prices.length()
// average = £11.63
```

### Random Selection

```parsley
let jedi = ["Yoda", "Luke", "Leia", "Obi-Wan", "Mace"]
let chosen = jedi.pick(1)                // with replacement: e.g. ["Obi-Wan"]

let deck = ["2♠", "3♠", "4♠", "5♠", "6♠", "7♠", "8♠", "9♠", "10♠", "J♠", "Q♠", "K♠", "A♠"]
let hand = deck.shuffle().take(5)        // without replacement: 5 distinct cards
```

### Locale-Aware Presentation

```parsley
let winners = ["Alice", "Bob", "Charlie"]
`Congratulations to {winners.format("and")}!`
// "Congratulations to Alice, Bob, and Charlie!"

// German:
`Herzlichen Glückwunsch {winners.format("and", "DE")}!`
```

### Chunking for Layouts

```parsley
let items = ["A", "B", "C", "D", "E", "F"]
let rows = items / 3
// [["A", "B", "C"], ["D", "E", "F"]]

// Render as a grid:
<table>
    for (row in rows) {
        <tr>for (cell in row) { <td>cell</td> }</tr>
    }
</table>
```

### Set Algebra

```parsley
let admins = ["alice", "bob", "carol"]
let editors = ["bob", "carol", "dave"]

let both = admins && editors             // ["bob", "carol"]
let either = admins || editors           // ["alice", "bob", "carol", "dave"]
let adminsOnly = admins - editors        // ["alice"]
```

## Key Differences from Other Languages

| Concept | Parsley | Other languages |
|---------|---------|-----------------|
| `for` loops | Return arrays (like `map`) | Statements (no return value) |
| `++` | Array concatenation | Not common (JS uses `.concat()`) |
| `/` on arrays | Chunking | Not available |
| `&&` `\|\|` `-` on arrays | Set intersection, union, subtraction | Not available |
| Sort order | Natural sort (`"file2" < "file10"`) | Lexicographic |
| `[?n]` | Optional access (returns `null`) | Throws error |
| `pick` / `take` | Built-in random sampling | Requires library |
| `format` | Locale-aware list formatting | Requires library |
| `stop` / `skip` | Loop control | `break` / `continue` |

## See Also

- [Dictionaries](dictionary.md) — key-value pairs; arrays of dictionaries form tables
- [Strings](strings.md) — `.split()` returns arrays; `.join()` concatenates array elements
- [Control Flow](../fundamentals/control-flow.md) — `for` loops return arrays
- [Operators](../fundamentals/operators.md) — `++` concatenation, `*` repetition, `/` chunking
- [Types](../fundamentals/types.md) — array in the type system
- [@std/table](../stdlib/table.md) — SQL-like operations on arrays of dictionaries
- [@std/math](../stdlib/math.md) — `sum`, `avg`, `median`, and other aggregation functions
- [Data Formats](../features/data-formats.md) — CSV parsing returns arrays/tables