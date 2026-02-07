---
id: man-pars-types
title: Type System
system: parsley
type: fundamentals
name: types
created: 2026-02-06
version: 0.2.0
author: Basil Team
keywords:
  - type
  - types
  - integer
  - float
  - string
  - boolean
  - null
  - array
  - dictionary
  - table
  - record
  - schema
  - money
  - datetime
  - duration
  - path
  - url
  - regex
  - function
  - coercion
  - is
  - truthiness
---

# Type System

Parsley is dynamically typed — variables don't have declared types and can hold any value. Types are checked at runtime when operations require specific types. There are no type annotations, interfaces, or generics.

## All Types

### Primitives

| Type | Examples | Notes |
|---|---|---|
| Integer | `0`, `42`, `-7` | 64-bit signed |
| Float | `3.14`, `-0.5`, `1.0` | 64-bit IEEE 754 |
| Boolean | `true`, `false` | |
| String | `"hello"`, `` `template` ``, `'raw'` | Three string kinds |
| Null | `null` | The absence of a value |

### Collections

| Type | Examples | Notes |
|---|---|---|
| Array | `[1, 2, 3]`, `[]` | Ordered, mixed types allowed |
| Dictionary | `{name: "Alice", age: 30}` | Ordered key-value pairs (string keys) |
| Table | `@table [{name: "Alice"}, {name: "Bob"}]` | Typed tabular data with columns |

### Structured

| Type | Description |
|---|---|
| Schema | Defines shape, types, constraints, and metadata for data |
| Record | Dictionary + Schema + validation errors — a validated data container |

### Specialized

| Type | Literal syntax | Description |
|---|---|---|
| Money | `$12.34`, `EUR#50.00` | Exact currency arithmetic with banker's rounding |
| DateTime | `@2026-02-06`, `@2026-02-06T15:30:00` | Date and datetime values |
| Duration | `@5m`, `@2h30m`, `@1d` | Time durations |
| Path | `@./config.json`, `@~/lib` | Filesystem paths |
| URL | `@https://example.com` | Web addresses |
| Regex | `/\d+/g` | Regular expressions |

### Callable

| Type | Description |
|---|---|
| Function | User-defined with `fn` — first-class, closures supported |
| Builtin | Built-in functions (`log`, `toString`, `fail`, etc.) |

### I/O & Connections

| Type | Description |
|---|---|
| File handle | Created by `JSON()`, `CSV()`, `text()`, etc. — used with `<==` / `==>` |
| Directory handle | Created by `dir()` — represents a directory for listing |
| DB connection | Database connection (SQLite, PostgreSQL, etc.) |
| SFTP connection | Remote file access over SSH |

## The Dictionary: Universal Composite Type

Dictionaries are Parsley's core composite type. Many "specialized" types are actually dictionaries with a `__type` metadata key:

- Paths are dictionaries with `__type: "path"` and `segments`, `absolute` keys
- URLs are dictionaries with `__type: "url"` and `scheme`, `host`, `path` keys
- Regex values are dictionaries with `__type: "regex"` and `pattern`, `flags` keys
- DateTime values are dictionaries with `__type: "datetime"` and date/time keys
- File handles are dictionaries with `__type: "file"` or `"dir"` and format/path keys

This means you can inspect any value's structure with standard dictionary access, and create values programmatically by constructing the right dictionary shape.

## Type Coercion

Parsley performs implicit type coercion in a few specific contexts:

### String concatenation

`+` converts the non-string operand to a string when one side is a string:

```parsley
"count: " + 5                    // "count: 5"
"pi: " + 3.14                   // "pi: 3.14"
"flag: " + true                  // "flag: true"
```

### Numeric promotion

Integer-to-float promotion in mixed arithmetic:

```parsley
1 + 2.5                          // 3.5 (integer promoted to float)
10 / 3                           // 3 (integer division stays integer)
10 / 3.0                         // 3.333... (float division)
```

### Truthiness

All values have a boolean interpretation used by `if`, `check`, and logical operators:

| Falsy values | Everything else is truthy |
|---|---|
| `false`, `null` | `true`, non-zero numbers |
| `0`, `0.0` | non-empty strings |
| `""` (empty string) | non-empty arrays and dictionaries |
| `[]` (empty array) | functions, file handles, etc. |
| `{}` (empty dictionary) | |

### No other implicit coercion

There is no implicit conversion between unrelated types. Adding an integer to an array, or comparing a string to a number (other than with `==`/`!=`), produces a type error:

```parsley
[1, 2] + 3                       // Error — use [1, 2] ++ [3]
```

## Type Checking

### typeof()

Returns a string identifying the value's type:

```parsley
typeof(42)                       // "integer"
typeof("hello")                  // "string"
typeof([1, 2])                   // "array"
typeof({a: 1})                   // "dictionary"
typeof(null)                     // "null"
typeof(fn() { })                 // "function"
```

### is (Schema Check)

The `is` operator checks whether a record conforms to a schema:

```parsley
let valid = record is UserSchema
```

This is not a general-purpose type check — it specifically tests schema conformance for Records. See [Data Model](data-model.md) for details.

## Everything Is an Expression

Every construct in Parsley produces a value. This means types flow naturally through control flow:

```parsley
let x = if (cond) 42 else "hello"     // x is integer or string
let items = for (n in 1..5) { n * n } // items is an array
let result = try riskyCall()           // result is a dictionary
```

There's no separate "statement" that produces no value — even `let` returns `null`. This expression-oriented design means you rarely need to think about types explicitly; values just flow through your code.

## Key Differences from Other Languages

- **Dynamic typing with no annotations** — no TypeScript-style type declarations, no Python type hints. Types are purely runtime.
- **Dictionary is the universal container** — paths, URLs, datetime, regex, and file handles are all dictionaries with metadata. There's no class or struct system.
- **No `instanceof` for general types** — `is` only works for schema checking. Use `typeof()` for general type inspection.
- **Integer division stays integer** — `10 / 3` is `3`, not `3.333`. Use a float operand (`10 / 3.0`) for float division.
- **Money is a distinct type** — not a float. `$10.00 + $5.00` uses exact arithmetic, not floating-point.
- **String concatenation coerces** — `"x" + 5` works and produces `"x5"`. Most other mixed-type operations are errors.

## See Also

- [Booleans & Null](../builtins/booleans.md) — truthiness rules
- [Strings](../builtins/strings.md) — the three string kinds
- [Operators](operators.md) — type-dependent operator behavior
- [Data Model](data-model.md) — Schema, Record, and Table system
- [Error Handling](errors.md) — type errors and how they're reported