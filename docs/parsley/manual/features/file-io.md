---
id: man-pars-file-io
title: File I/O
system: parsley
type: features
name: file-io
created: 2026-02-06
version: 0.2.0
author: Basil Team
keywords:
  - file
  - read
  - write
  - append
  - JSON
  - YAML
  - CSV
  - PLN
  - markdown
  - text
  - directory
  - file handle
  - operators
---

# File I/O

Parsley uses **file handles** and **I/O operators** for reading and writing files. You create a handle that describes the file and its format, then use `<==` to read or `==>` to write. The handle determines how data is serialized and deserialized.

## I/O Operators

| Operator | Direction | Description |
|---|---|---|
| `<==` | Read | Read file contents into a variable |
| `==>` | Write | Write data to a file (overwrites) |
| `==>>` | Append | Append data to a file |
| `=/=>` | Remote Write | Write data to a network target (HTTP/SFTP) |
| `=/=>>` | Remote Append | Append data to a network target (SFTP) |

```parsley
let config <== JSON(@./config.json)       // read
{name: "Alice"} ==> JSON(@./output.json)  // write
"log entry\n" ==>> text(@./app.log)       // append

// Network targets use =/=> instead of ==>
{name: "Alice"} =/=> JSON(@https://api.example.com/users)  // remote write
```

> **Note**: For network targets (HTTP URLs and SFTP connections), use `=/=>` and `=/=>>` instead of `==>` and `==>>`. The `/` in the operator visually signals that data crosses a network boundary. Both fetch (`<=/=`) and remote write (`=/=>`, `=/=>>`) can also be used as expressions — see [HTTP & Networking](network.md) for details.

## File Handles

File handles are created by calling a format function with a path. They don't read or write anything on their own — the I/O operators do the actual work.

| Function | Format | Read type | Description |
|---|---|---|---|
| `JSON(path)` | JSON | dictionary/array | JSON file |
| `YAML(path)` | YAML | dictionary/array | YAML file |
| `CSV(path)` | CSV | table | CSV file (returns a table) |
| `PLN(path)` | PLN | any | Parsley Literal Notation |
| `text(path)` | Plain text | string | Raw text content |
| `lines(path)` | Lines | array | Array of strings (one per line) |
| `bytes(path)` | Binary | array | Raw byte array |
| `MD(path)` | Markdown | string | Rendered HTML from markdown |
| `markdown(path)` | Markdown | dictionary | `{meta, content}` with frontmatter |
| `SVG(path)` | SVG | string | SVG content (strips XML prolog) |
| `file(path)` | Auto | varies | Auto-detect format from extension |
| `dir(path)` | Directory | array | Directory listing |

All file handles accept a path literal as the first argument:

```parsley
let handle = JSON(@./data.json)
let data <== handle
```

Or inline:

```parsley
let data <== JSON(@./data.json)
```

## Reading Files

### JSON

```parsley
let config <== JSON(@./config.json)
config.database.host             // "localhost"
```

> **Tip:** For configuration files with Parsley-specific types (dates, money, paths), use PLN instead: `let config <== PLN(@./config.pln)`. PLN preserves all Parsley types—see [Data Formats](data-formats.md#pln-parsley-literal-notation).

### CSV

CSV reads return a table (not a raw array):

```parsley
let sales <== CSV(@./sales.csv)
sales.count()                    // number of rows
for (row in sales) {
    row.name + ": " + row.amount
}
```

### Plain Text

```parsley
let readme <== text(@./README.md)
readme.length()                  // character count
```

### Lines

```parsley
let items <== lines(@./todo.txt)
items.length()                   // number of lines
items[0]                         // first line
```

### Markdown with Frontmatter

`markdown()` parses YAML frontmatter and renders content to HTML:

```parsley
let doc <== markdown(@./post.md)
doc.meta.title                   // frontmatter field
doc.content                      // rendered HTML string
```

`MD()` just renders to HTML without frontmatter parsing:

```parsley
let html <== MD(@./readme.md)
// html is an HTML string
```

### Auto-detect

`file()` picks the format based on the file extension:

```parsley
let data <== file(@./config.json)  // reads as JSON
let text <== file(@./notes.txt)    // reads as text
```

## Writing Files

Use `==>` to write (overwrite) and `==>>` to append:

```parsley
// Write PLN (preserves Parsley types like dates, money, paths)
{name: "Alice", joined: @2024-01-15, balance: $100.00} ==> PLN(@./user.pln)

// Write JSON (for external systems)
{name: "Alice", age: 30} ==> JSON(@./user.json)

// Write plain text
"Hello, world!" ==> text(@./greeting.txt)

// Append to a log
"New entry\n" ==>> text(@./app.log)
```

The write operator serializes the data according to the file handle's format. Writing a dictionary to a JSON handle produces formatted JSON; writing a string to a text handle writes it verbatim.

> **When to use PLN vs JSON:** Use PLN for internal Parsley data (configs, caches, data files)—it preserves dates, money, paths, and other Parsley types. Use JSON for external interoperability (APIs, other languages). See [Data Formats](data-formats.md#when-to-use-pln-vs-json) for details.

## Directory Operations

### dir()

Create a directory handle and read its contents:

```parsley
let files <== dir(@./uploads)
for (f in files) {
    f.name                       // filename
}
```

### fileList()

Recursively list files matching a glob pattern:

```parsley
let sources = fileList(@./src, "*.pars")
sources.length()                 // number of matching files
```

## Error Handling with Read

Use destructured read with `{data, error}` for safe file operations:

```parsley
let {data, error} <== JSON(@./config.json)
if (error) {
    log("Failed to read config: " + error)
    let config = defaults
} else {
    let config = data
}
```

When using the `{data, error}` pattern, read errors are captured instead of halting execution. Without it, a missing file or parse error produces an IO-class error.

## Assets

The `asset()` builtin converts a file path to a web-accessible URL with cache-busting:

```parsley
<img src={asset(@./logo.png)} alt="Logo"/>
// Produces: <img src="/assets/logo-a1b2c3d4.png" alt="Logo" />
```

## Key Differences from Other Languages

- **Operators instead of functions** — `<==` and `==>` replace `readFile()`/`writeFile()`. The operator syntax makes the data flow direction visually clear.
- **Format-aware handles** — the handle knows the file format, so you don't manually parse JSON or serialize CSV. `let data <== CSV(@./file.csv)` returns a ready-to-use table.
- **PLN format** — Parsley has its own serialization format (Parsley Literal Notation) that round-trips all Parsley types, including dates, money, and paths.
- **Markdown is a first-class format** — `markdown()` handles frontmatter parsing and HTML rendering in one step.
- **No streams** — file I/O is synchronous and reads/writes the entire file at once.

## See Also

- [Paths](../builtins/paths.md) — path literals and path manipulation
- [URLs](../builtins/urls.md) — URL literals used as file handle sources
- [HTTP & Networking](network.md) — `<=/=` fetch, `=/=>` remote write, and `=/=>>` remote append operators
- [Error Handling](../fundamentals/errors.md) — `{data, error}` destructuring pattern
- [Data Formats](data-formats.md) — CSV and Markdown parsing details