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

PLN (Parsley Literal Notation) is a data serialization format for Parsley that uses a safe subset of Parsley syntax to represent values—including schema-bound records and validation errors—without allowing code execution.

```parsley
// Serialize any Parsley value to a PLN string
let pln = serialize({name: "Alice", age: 30})
// Result: '{name: "Alice", age: 30}'

// Deserialize PLN back to Parsley values
let data = deserialize('{name: "Bob", active: true}')
log(data.name)  // "Bob"
```

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

- Throws if the value contains non-serializable types (functions, file handles, database connections)

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

- Throws if the PLN string is invalid or contains expressions/code

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

### Records (Coming Soon)

Records preserve schema association:

```pln
@Person({name: "Alice", age: 30})
```

Records with validation errors:

```pln
@Person({name: ""}) @errors {name: "Required"}
```

### Datetimes (Coming Soon)

```pln
@2024-01-20
@2024-01-20T10:30:00Z
@10:30:00
```

### Paths and URLs (Coming Soon)

```pln
@/path/to/file
@./relative/path
@https://example.com
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

| Type | PLN Representation |
|------|-------------------|
| Integer | `42` |
| Float | `3.14` |
| String | `"hello"` |
| Boolean | `true`, `false` |
| Null | `null` |
| Array | `[1, 2, 3]` |
| Dictionary | `{a: 1, b: 2}` |
| Record | `@Schema({...})` (planned) |
| DateTime | `@2024-01-20T10:30:00Z` (planned) |
| Path | `@/path/to/file` (planned) |
| URL | `@https://example.com` (planned) |

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

4. **Safe deserialization**: Unlike JSON with eval(), PLN cannot execute arbitrary code.

```parsley
// These will all fail:
deserialize("1 + 1")      // Error: expressions not allowed
deserialize("x")          // Error: identifiers not allowed
deserialize("print(42)")  // Error: function calls not allowed
```

---

## Use Cases

### Configuration Files

Store application configuration in `.pln` files:

```pln
// config.pln
{
  port: 8080,
  debug: true,
  database: {
    host: "localhost",
    name: "myapp"
  },
  features: ["auth", "logging"]
}
```

```parsley
let f = PLN(@./config.pln)
let config <== f
log("Running on port", config.port)
```

### Data Exchange

Serialize data for storage or transmission:

```parsley
let user = {name: "Alice", active: true}
let pln = serialize(user)
// Store `pln` in database or send over network
// Later: deserialize(pln) to restore the value
```

### API Responses

Return structured data from handlers:

```parsley
export default = fn(req) {
  let data = {status: "ok", items: getItems()}
  serialize(data)
}
```

---

## See Also

- [File I/O](features/file-io.md) — file handles and I/O operators
- [Records](builtins/record.md) — schema-bound records with validation
- [DateTime](builtins/datetime.md) — date and time handling
- [Security Model](features/security.md) — PLN safety and deserialization security
- [Data Formats](features/data-formats.md) — JSON, CSV, and Markdown parsing
