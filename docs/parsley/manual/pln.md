---
id: man-pars-pln
title: "Parsley Literal Notation (PLN)"
system: parsley
type: feature
name: pln
created: 2026-01-20
version: 0.3.0
author: "@sam"
keywords: pln, serialization, data, format, serialize, deserialize
---

## Parsley Literal Notation (PLN)

PLN (Parsley Literal Notation) is a data serialization format for Parsley that uses a safe subset of Parsley syntax to represent values—including schema-bound records, dates, money, paths, and validation errors—without allowing code execution.

**Use PLN instead of JSON** for Parsley-to-Parsley data exchange. PLN preserves all Parsley types, while JSON loses type information (dates become strings, money becomes numbers).

```parsley
// Write Parsley data to a PLN file
{name: "Alice", joined: @2024-01-15, balance: $100.00} ==> PLN(@./user.pln)

// Read it back — types are preserved!
let user <== PLN(@./user.pln)
user.joined.year     // 2024 (datetime, not string)
user.balance + $50   // $150.00 (money, not number)
```

## File I/O

PLN integrates with Parsley's file I/O system. Use `==>` to write and `<==` to read:

```parsley
// Write PLN
let config = {
    port: 8080,
    launchDate: @2024-06-01,
    budget: $50000.00,
    dataPath: @./data/users.csv
}
config ==> PLN(@./config.pln)

// Read PLN
let loaded <== PLN(@./config.pln)
loaded.launchDate              // @2024-06-01 (datetime ✓)
loaded.budget                  // $50000.00 (money ✓)
loaded.dataPath                // @./data/users.csv (path ✓)
```

### Append Mode

Use `==>>` to append multiple values to a PLN file:

```parsley
{event: "login", time: @now} ==>> PLN(@./events.pln)
{event: "purchase", time: @now} ==>> PLN(@./events.pln)
```

### Auto-Detection

Files with `.pln` extension are automatically recognized:

```parsley
let data <== file(@./data.pln)   // auto-detects PLN format
```

---

## Functions

### serialize(value)

Converts a Parsley value to a PLN string representation.

#### Signature

```
serialize(value: Any) → String
```

#### Parameters

- `value` — Any serializable Parsley value

#### Returns

A string containing the PLN representation of the value.

#### Errors

Fails if the value contains non-serializable types (functions, file handles, database connections)

#### Example

```parsley
// Primitives
serialize(42)           // "42"
serialize("hello")      // "\"hello\""
serialize(true)         // "true"
serialize(null)         // "null"

// Collections
serialize([1, 2, 3])    // "[1, 2, 3]"
serialize({a: 1, b: 2}) // "{a: 1, b: 2}"

// Nested structures
let user = {
  name: "Alice",
  profile: {email: "alice@example.com"},
  tags: ["admin", "user"]
}
serialize(user)
// '{name: "Alice", profile: {email: "alice@example.com"}, tags: ["admin", "user"]}'
```

---

### deserialize(pln)

Parses a PLN string and returns the corresponding Parsley value.

#### Signature

```
deserialize(pln: String) → Any
```

#### Parameters

- `pln` — A valid PLN string

#### Returns

The Parsley value represented by the PLN string.

#### Errors

Fails if the PLN string is invalid or contains expressions/code

#### Example

```parsley
// Primitives
deserialize("42")           // 42
deserialize("\"hello\"")    // "hello"
deserialize("true")         // true
deserialize("null")         // null

// Collections
deserialize("[1, 2, 3]")    // [1, 2, 3]
deserialize("{a: 1, b: 2}") // {a: 1, b: 2}

// From a file
let f = PLN(@./data.pln)
let config <== f
```

---

### PLN(path)

Creates a file handle for loading PLN files. Used with the `<==` read operator.

#### Signature

```
PLN(path: Path | String) → FileHandle
```

#### Parameters

- `path` — Path to a `.pln` file

#### Returns

A file handle that can be read with `<==`.

#### Example

```parsley
let f = PLN(@./config.pln)
let config <== f
log(config.name)
```

---

## PLN Syntax

PLN is a safe subset of Parsley syntax. It supports values only—no expressions, variables, or function calls.

### Primitives

```pln
// Integers and floats
42
3.14
-17

// Strings (double-quoted with escapes)
"hello world"
"line1\nline2"

// Booleans
true
false

// Null
null
```

### Arrays

```pln
[]
[1, 2, 3]
["a", "b", "c"]
[1, 2, 3,]  // Trailing comma allowed
```

### Dictionaries

```pln
{}
{name: "Alice"}
{name: "Alice", age: 30}
{"quoted-key": "value"}  // Quoted keys allowed
{nested: {deep: true}}
```

### Records

Records preserve schema association:

```pln
@Person({name: "Alice", age: 30})
```

Records with validation errors:

```pln
@Person({name: ""}) @errors {name: "Required"}
```

### Money

Money uses `CODE#amount` literal notation:

```pln
USD#19.99
JPY#500
EUR#-10.50
GBP#1000.00
```

### Datetimes

```pln
@2024-01-20
@2024-01-20T10:30:00Z
@2024-01-20T10:30:00
```

### Paths and URLs

```pln
@/path/to/file
@./relative/path
@https://example.com/api
@http://localhost:8080/test
```

### Comments

```pln
// Single-line comments are supported
{
  name: "Alice",  // inline comment
  age: 30
}
```

---

## File Loading

PLN files can be loaded using the `file()` builtin (auto-detects format) or the explicit `PLN()` builtin.

### Auto-Detection

Files with `.pln` extension are automatically parsed as PLN:

```parsley
let f = file(@./data.pln)
let data <== f
```

### Explicit Loading

Use `PLN()` for explicit format control:

```parsley
let f = PLN(@./config.pln)
let config <== f
```

---

## Serializable Types

| Type | PLN Representation | Round-trips |
|------|-------------------|-------------|
| Integer | `42` | ✅ |
| Float | `3.14` | ✅ |
| String | `"hello"` | ✅ |
| Boolean | `true`, `false` | ✅ |
| Null | `null` | ✅ |
| Array | `[1, 2, 3]` | ✅ |
| Dictionary | `{a: 1, b: 2}` | ✅ |
| Money | `USD#19.99` | ✅ |
| Record | `@Schema({...})` | ✅ |
| DateTime | `@2024-01-20T10:30:00` | ✅ |
| Path | `@./path/to/file` | ✅ |
| URL | `@https://example.com` | ✅ |
| Table | `[{...}, {...}]` | ✅ |

### Non-Serializable Types

The following types cannot be serialized:

- Functions (`fn(x) { x }`)
- Builtins (`len`, `print`)
- File handles
- Database connections
- Modules

Attempting to serialize these will produce an error.

---

## Security

PLN is designed for safe data exchange:

1. **No code execution**: PLN uses a dedicated parser that only accepts values. Expressions like `1 + 1` are rejected.

2. **No variables**: References to identifiers (except keywords like `true`, `false`, `null`) are rejected.

3. **No function calls**: Syntax like `Schema({...})` is rejected; records must use `@Schema({...})`.

4. **Safe deserialization**: PLN cannot execute arbitrary code.

```parsley
// These will all fail:
deserialize("1 + 1")      // Error: expressions not allowed
deserialize("x")          // Error: identifiers not allowed
deserialize("print(42)")  // Error: function calls not allowed
```

---

## Use Cases

### Configuration Files

Store application configuration in `.pln` files—types are preserved:

```pln
// config.pln
{
  port: 8080,
  debug: true,
  launchDate: @2024-06-01,
  budget: USD#50000.00,
  dataPath: @./data/users.csv,
  apiEndpoint: @https://api.example.com/v1
}
```

```parsley
let config <== PLN(@./config.pln)
config.launchDate.year           // 2024 (datetime)
config.budget + USD#1000.00      // $51000.00 (money arithmetic works)
```

### Caching Parsley Data

Cache computed results without losing type information:

```parsley
// Compute and cache
let results = expensiveComputation()
results ==> PLN(@./cache/results.pln)

// Later: load from cache
let cached <== PLN(@./cache/results.pln)
// All types preserved — dates, money, paths, etc.
```

### Data Migration

Transform and save data between runs:

```parsley
let users <== CSV(@./users.csv)
let enriched = for (user in users) {
    user ++ {
        created: @now,
        balance: USD#0.00,
        configPath: @(./configs/{user.id}.pln)
    }
}
enriched ==> PLN(@./users.pln)
```

### Debugging

PLN output is valid Parsley syntax—useful for debugging:

```parsley
let data = {name: "Alice", joined: @2024-01-15, balance: USD#100.00}
log(serialize(data))
// {name: "Alice", joined: @2024-01-15, balance: USD#100.00}
// Can copy-paste this directly into Parsley code!
```

---

## PLN vs JSON

| Aspect | PLN | JSON |
|--------|-----|------|
| **Use for** | Parsley-to-Parsley data | External systems, APIs |
| **Dates** | `@2024-01-15` (preserved) | `"2024-01-15"` (string) |
| **Money** | `USD#19.99` (preserved) | `19.99` (number, loses currency) |
| **Paths** | `@./file.txt` (preserved) | `"./file.txt"` (string) |
| **URLs** | `@https://...` (preserved) | `"https://..."` (string) |
| **Records** | `@Person({...})` (preserved) | `{...}` (plain object) |
| **Syntax** | Parsley literal syntax | Standard JSON |

**Rule of thumb:** Use PLN for internal Parsley data. Use JSON for external interoperability.

---

## See Also

- [File I/O](features/file-io.md) — file handles and I/O operators
- [Data Formats](features/data-formats.md) — when to use PLN vs JSON
- [Records](builtins/record.md) — schema-bound records with validation
- [DateTime](builtins/datetime.md) — date and time handling
- [Money](builtins/money.md) — currency handling
