---
id: man-pars-data-formats
title: Data Formats
system: parsley
type: features
name: data-formats
created: 2026-02-06
version: 0.2.0
author: Basil Team
keywords:
  - markdown
  - CSV
  - JSON
  - parsing
  - serialization
  - frontmatter
  - data
  - table
  - format
---

# Data Formats

Parsley has built-in support for parsing and generating Markdown, CSV, and JSON. These work both as **string methods** (parse/encode in memory) and as **file handles** (read/write files directly). See [File I/O](file-io.md) for the file handle approach — this page focuses on the string methods and format-specific behavior.

## Markdown

### `.parseMarkdown(options?)`

Parses a Markdown string into a dictionary with `html`, `raw`, and `md` keys:

```parsley
let source = "# Hello\n\nSome **bold** text."
let result = source.parseMarkdown()
result.html                      // "<h1>Hello</h1>\n<p>Some <strong>bold</strong> text.</p>\n"
result.raw                       // "# Hello\n\nSome **bold** text."
result.md                        // {} (empty — no frontmatter)
```

| Key | Type | Description |
|---|---|---|
| `html` | string | Rendered HTML |
| `raw` | string | Original Markdown source (with frontmatter stripped) |
| `md` | dictionary | Parsed YAML frontmatter fields |

### Options

Pass a dictionary to control rendering:

```parsley
let html = source.parseMarkdown({ids: true})
// Headings get auto-generated id attributes: <h1 id="hello">Hello</h1>
```

| Option | Type | Default | Description |
|---|---|---|---|
| `ids` | boolean | `false` | Generate `id` attributes on headings |

### Frontmatter

If the Markdown starts with YAML frontmatter delimited by `---`, it is parsed into the `md` field:

```parsley
let doc = "---\ntitle: My Post\ndate: 2024-06-15\ntags:\n  - parsley\n  - guide\n---\n# Content\n\nBody text."

let result = doc.parseMarkdown()
result.md.title                  // "My Post"
result.md.date                   // "2024-06-15"
result.md.tags                   // ["parsley", "guide"]
result.raw                       // "# Content\n\nBody text."
```

Frontmatter variables are also available for `@{expr}` interpolation within the Markdown body during rendering.

### File Handles

Two file handles read Markdown from disk:

```parsley
// markdown() — parses frontmatter and renders HTML
let doc <== markdown(@./post.md)
doc.meta                         // frontmatter dictionary
doc.content                      // rendered HTML

// MD() — renders to HTML only (no frontmatter parsing)
let html <== MD(@./readme.md)
```

## CSV

### `.parseCSV(hasHeader?)`

Parses a CSV string. The `hasHeader` argument (default `true`) controls whether the first row is treated as column names:

```parsley
let csv = "name,age,active\nAlice,30,true\nBob,25,false"

let data = csv.parseCSV()
// Returns a Table with columns ["name", "age", "active"]
// Each row is a dictionary: {name: "Alice", age: 30, active: true}
```

With header (default):

```parsley
let data = csv.parseCSV(true)
data.count()                     // 2
data[0].name                     // "Alice"
data[0].age                      // 30 (integer, not string)
```

Without header:

```parsley
let raw = "Alice,30\nBob,25"
let data = raw.parseCSV(false)
// Returns an array of arrays: [["Alice", 30], ["Bob", 25]]
```

### Auto-Type Detection

CSV values are automatically converted from strings to typed values:

| CSV Value | Parsley Type | Example |
|---|---|---|
| `42` | integer | `42` |
| `3.14` | float | `3.14` |
| `true` / `false` | boolean | `true` |
| anything else | string | `"Alice"` |

### `.toCSV(hasHeader?)`

Converts an array of dictionaries (or array of arrays) back to a CSV string. Available on arrays and tables.

```parsley
let people = [
    {name: "Alice", age: 30},
    {name: "Bob", age: 25}
]
people.toCSV()
// "name,age\nAlice,30\nBob,25\n"
```

Without header:

```parsley
let rows = [["Alice", 30], ["Bob", 25]]
rows.toCSV(false)
// "Alice,30\nBob,25\n"
```

### File Handles

```parsley
// Read CSV file — returns a Table
let sales <== CSV(@./sales.csv)
sales.count()

// Write CSV
people.toCSV() ==> text(@./output.csv)
```

The `CSV()` file handle always parses with headers.

### Table Methods

Tables (from CSV or database queries) have their own serialization methods:

```parsley
let sales <== CSV(@./sales.csv)
sales.toCSV()                    // CSV string with header
sales.toJSON()                   // JSON array of objects
sales.toHTML()                   // HTML <table> element
sales.toMarkdown()               // Markdown table
sales.toBox()                    // ASCII box-drawing table
```

## JSON

### `.parseJSON()`

Parses a JSON string into Parsley values:

```parsley
let json = '{"name": "Alice", "age": 30, "tags": ["admin", "user"]}'
let data = json.parseJSON()
data.name                        // "Alice"
data.age                         // 30
data.tags[0]                     // "admin"
```

JSON types map to Parsley types:

| JSON | Parsley |
|---|---|
| object | dictionary |
| array | array |
| string | string |
| number (integer) | integer |
| number (float) | float |
| `true` / `false` | boolean |
| `null` | null |

### `.toJSON()`

Converts a value to a JSON string. Available on strings, integers, floats, arrays, dictionaries, tables, datetimes, and durations:

```parsley
{name: "Alice", age: 30}.toJSON()
// '{\n  "age": 30,\n  "name": "Alice"\n}'

[1, 2, 3].toJSON()               // "[1,2,3]"
42.toJSON()                      // "42"
"hello".toJSON()                 // '"hello"'
```

JSON output is pretty-printed with 2-space indentation for dictionaries.

### File Handles

```parsley
// Read JSON file
let config <== JSON(@./config.json)

// Write JSON file
{name: "Alice"} ==> JSON(@./output.json)
```

## PLN (Parsley Literal Notation)

Parsley has its own serialization format that round-trips all Parsley types, including dates, money, paths, and other types that JSON cannot represent:

```parsley
let data <== PLN(@./data.pln)
data ==> PLN(@./backup.pln)
```

PLN is useful for caching or storing Parsley values without losing type information.

## Common Patterns

### Read, Transform, Write

```parsley
// Read CSV, transform, write JSON
let sales <== CSV(@./sales.csv)
let summary = for (row in sales) {
    {name: row.product, total: row.price * row.quantity}
}
summary.toJSON() ==> text(@./summary.json)
```

### Parse API Response

```parsley
let response <== JSON(@https://api.example.com/users)
for (user in response) {
    user.name + ": " + user.email
}
```

### Markdown Blog Pipeline

```parsley
let post <== markdown(@./posts/hello.md)
let title = post.meta.title
let html = post.content

<article>
    <h1>title</h1>
    html
</article>
```

## Key Differences from Other Languages

- **Parsing returns typed values** — CSV auto-detects integers, floats, and booleans. You don't need to manually convert `"42"` to a number after parsing.
- **Tables, not arrays** — CSV with headers returns a Table (which supports `.count()`, `.where()`, `.orderBy()`, etc.), not a plain array of objects.
- **Markdown includes frontmatter** — `.parseMarkdown()` handles YAML frontmatter in one step, returning structured metadata alongside rendered HTML.
- **No streaming** — all parsing and encoding operates on complete strings or files. There are no streaming parsers.

## See Also

- [File I/O](file-io.md) — file handles and I/O operators
- [Strings](../builtins/strings.md) — string methods including parsing
- [Data Model](../fundamentals/data-model.md) — Table and Record types
- [Tags](../fundamentals/tags.md) — rendering HTML with Parsley tags
