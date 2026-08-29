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
result.md.date                   // 2024-06-15T00:00:00Z (a datetime, not a string)
result.md.tags                   // ["parsley", "guide"]
result.raw                       // "# Content\n\nBody text."
```

Frontmatter variables are also available for `@{expr}` interpolation within the Markdown body during rendering.

### File Handles

The `MD()` file handle reads Markdown from disk, parsing frontmatter and rendering HTML in one step:

```parsley
let doc <== MD(@./post.md)
doc.md                           // frontmatter dictionary
doc.html                         // rendered HTML
doc.raw                          // original Markdown (frontmatter stripped)
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
// '{"age":30,"name":"Alice"}'

[1, 2, 3].toJSON()               // "[1,2,3]"
42.toJSON()                      // "42"
"hello".toJSON()                 // '"hello"'
```

`.toJSON()` returns compact JSON, with dictionary keys sorted alphabetically. Writing to a `JSON()` file handle pretty-prints with 2-space indentation instead.

### File Handles

```parsley
// Read JSON file
let config <== JSON(@./config.json)

// Write JSON file
{name: "Alice"} ==> JSON(@./output.json)
```

## PLN (Parsley Literal Notation)

Parsley has its own serialization format that round-trips all Parsley types losslessly:

```parsley
let data <== PLN(@./data.pln)
data ==> PLN(@./backup.pln)
```

PLN uses literal notation for Parsley types:

| Type | PLN Literal | Example |
|------|-------------|---------|
| Money | `CODE#amount` | `USD#19.99`, `JPY#500` |
| Date | `@YYYY-MM-DD` | `@2024-01-15` |
| DateTime | `@YYYY-MM-DDTHH:MM:SS` | `@2024-01-15T10:30:00` |
| Path | `@path` | `@./config/app.pln` |
| URL | `@url` | `@https://example.com/api` |
| Record | `@Schema({...})` | `@Person({name: "Alice"})` |

### When to Use PLN vs JSON

**Use PLN for:**
- Configuration files that include dates, money, or paths
- Caching Parsley data structures between runs
- Data files read and written by Parsley scripts
- Serializing records with schemas
- Debugging (PLN output is valid Parsley syntax)

**Use JSON for:**
- API requests and responses
- Data exchange with non-Parsley systems (JavaScript, Python, etc.)
- When compatibility with JSON parsers is required

### Type Preservation

PLN preserves types that JSON cannot represent:

```parsley
// Using JSON (loses types)
let config = {
    launchDate: @2024-06-01,
    budget: $50000.00,
    dataPath: @./data/users.csv
}
config ==> JSON(@./config.json)
let loaded <== JSON(@./config.json)
loaded.launchDate                // {year: 2024, month: 6, day: 1, …} (plain dictionary!)
loaded.budget                    // "$50000.00" (string, lost the money type!)
loaded.dataPath                  // {segments: [".", "data", "users.csv"], …} (plain dictionary!)

// Using PLN (preserves types)
config ==> PLN(@./config.pln)
let reloaded <== PLN(@./config.pln)
reloaded.launchDate              // @2024-06-01 (date ✓)
reloaded.budget                  // $50000.00 (money ✓)
reloaded.dataPath                // @./data/users.csv (path ✓)
```

See [PLN](../pln.md) for the full specification.

## Common Patterns

### Read, Transform, Write

```parsley
// Read CSV, transform, write PLN (preserves Parsley types)
let sales <== CSV(@./sales.csv)
let summary = for (row in sales) {
    {name: row.product, total: row.price * row.quantity}
}
summary ==> PLN(@./summary.pln)

// Or write JSON (for external systems)
summary ==> JSON(@./summary.json)
```

### Parse API Response

Network reads use the fetch operator `<=/=` (not `<==`, which is for files):

```parsley
let response <=/= JSON(@https://api.example.com/users)
for (user in response) {
    user.name + ": " + user.email
}
```

### Markdown Blog Pipeline

```parsley
let post <== MD(@./posts/hello.md)
let title = post.md.title
let html = post.html

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
