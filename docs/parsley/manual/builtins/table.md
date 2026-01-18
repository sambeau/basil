---
id: man-pars-table
title: Tables
system: parsley
type: builtin
name: table
created: 2026-01-14
version: 0.16.0
author: Basil Team
keywords:
  - table
  - data
  - csv
  - database
  - query
  - sql
  - filter
  - sort
  - aggregate
---

# Tables

Tables are Parsley's first-class type for working with structured, rectangular data. Think of a Table as a spreadsheet or database result set—rows of data where each row has the same columns. Tables provide powerful SQL-like operations for filtering, sorting, selecting, and aggregating data, all with a clean method-chaining syntax.

```parsley
let sales = @table [
    {product: "Widget", region: "North", amount: $1200},
    {product: "Gadget", region: "South", amount: $800},
    {product: "Widget", region: "South", amount: $1500},
    {product: "Gadget", region: "North", amount: $950}
]

sales
    .where(fn(r) { r.region == "South" })
    .orderBy("amount", "desc")
    .select(["product", "amount"])
```

**Result:**

```
┌─────────┬────────┐
│ product │ amount │
├─────────┼────────┤
│ Widget  │ $1,500 │
│ Gadget  │ $800   │
└─────────┴────────┘
```

## Why Tables?

Parsley is designed for **web and data**. While arrays of dictionaries work for simple cases, Tables provide:

| Feature | Array of Dicts | Table |
|---------|---------------|-------|
| Column consistency | Not enforced | Guaranteed |
| SQL-like operations | Manual loops | Built-in methods |
| Export formats | DIY | `.toCSV()`, `.toHTML()`, `.toMarkdown()` |
| Aggregations | Manual | `.sum()`, `.avg()`, `.min()`, `.max()` |
| Schema validation | None | Optional `@schema` integration |
| Memory efficiency | N/A | Copy-on-chain optimization |

Tables are ideal for:
- **CSV file processing** — Import, transform, export
- **Database results** — Query, filter, format for display
- **Report generation** — Aggregate, format as HTML or Markdown
- **Data pipelines** — Chain transformations like SQL queries

---

## Creating Tables

### The `@table` Literal

The most direct way to create a Table is with the `@table` literal syntax:

```parsley
let users = @table [
    {name: "Alice", age: 30, active: true},
    {name: "Bob", age: 25, active: false},
    {name: "Carol", age: 35, active: true}
]
```

The `@table` literal provides **parse-time validation**:
- All rows must be dictionaries
- All rows must have the same keys (columns)
- Column names are inferred from the first row

```parsley
// This is a PARSE ERROR — "age" missing from second row:
let bad = @table [
    {name: "Alice", age: 30},
    {name: "Bob"}  // Error: missing column 'age'
]
```

### The `table()` Constructor

For dynamic data, use the `table()` constructor:

```parsley
let data = [{x: 1}, {x: 2}, {x: 3}]
let t = table(data)
```

The constructor validates at **runtime**:

```parsley
table([{a: 1}, {b: 2}])  // Error: column mismatch
table("not an array")    // Error: requires array
table([1, 2, 3])         // Error: elements must be dictionaries
```

### Empty Tables

Both syntaxes support empty tables:

```parsley
let empty1 = @table []
let empty2 = table([])

empty1.length    // 0
empty1.columns   // []
```

---

## Tables from External Sources

### CSV Files

The `CSV()` function and `.parseCSV()` method return Tables directly:

```parsley
// From a file
let sales <== CSV("sales.csv")

// From a string
let data = "name,age\nAlice,30\nBob,25".parseCSV()
```

CSV columns come from the header row. Values are automatically parsed as numbers, booleans, or strings:

```parsley
let csv = "product,price,in_stock\nWidget,9.99,true\nGadget,19.99,false"
let products = csv.parseCSV()

products[0].price     // 9.99 (number, not string)
products[0].in_stock  // true (boolean, not string)
```

### Database Queries

Database table bindings return Tables with full schema information:

```parsley
// Basil database binding
let users = Users.all()
let active = Users.where(fn(u) { u.active })

// Raw SQL also returns Table
let results <=?=> "SELECT * FROM orders WHERE total > 100"
```

Database-backed Tables carry schema metadata:

```parsley
users.schema  // Contains column types from database
```

### JSON APIs

JSON APIs return arrays by default. Convert to Table explicitly:

```parsley
let response <=/= fetch("https://api.example.com/users")
let users = table(response.json)
```

---

## Tables with Schemas

### Typed Tables with `@table(Schema)`

Combine `@table` with `@schema` for validated, typed tables with defaults:

```parsley
@schema Product {
    sku: string
    name: string
    price: money
    in_stock: bool = true
    discount: number = 0
}

// Schema defaults are applied when fields are missing from ALL rows
let products = @table(Product) [
    {sku: "A001", name: "Widget", price: $9.99},
    {sku: "A002", name: "Gadget", price: $19.99}
]

products[0].in_stock  // true (default applied)
products[0].discount  // 0 (default applied)
products[1].in_stock  // true (default applied)
```

> **Note:** All rows in a `@table` must have the same columns. To override a default, include the field in every row:

```parsley
let products = @table(Product) [
    {sku: "A001", name: "Widget", price: $9.99, in_stock: true, discount: 0},
    {sku: "A002", name: "Gadget", price: $19.99, in_stock: false, discount: 10}
]
```

### Nullable Fields

Use `?` suffix for optional fields. Nullable fields can hold `null` values:

```parsley
@schema Contact {
    name: string
    email: email
    phone: phone?  // nullable — can hold null
}

let contacts = @table(Contact) [
    {name: "Alice", email: "alice@example.com", phone: "555-1234"},
    {name: "Bob", email: "bob@example.com", phone: null}
]

contacts[0].phone  // "555-1234"
contacts[1].phone  // null
```

> **Note:** All rows must include all columns. Use `null` explicitly for missing nullable values.

### Required Field Validation

Non-nullable fields without defaults must be provided:

```parsley
@schema User {
    id: ulid
    email: email
}

// This is an ERROR — missing required "email":
let bad = @table(User) [
    {id: "01H..."}
]
// Error: Table row 1: missing required field 'email'
```

---

## Attributes

### columns

Returns an array of column names:

```parsley
let t = @table [{name: "Alice", age: 30}]
t.columns
```

**Result:** `["name", "age"]`

### length

Returns the number of rows:

```parsley
let t = @table [{x: 1}, {x: 2}, {x: 3}]
t.length
```

**Result:** `3`

### rows

Returns all rows as an array of dictionaries:

```parsley
let t = @table [{a: 1}, {a: 2}]
t.rows
```

**Result:** `[{a: 1}, {a: 2}]`

### schema

Returns the attached schema, or `null` if none:

```parsley
@schema Point { x: int, y: int }
let points = @table(Point) [{x: 1, y: 2}]
points.schema  // Point schema object
```

---

## Indexing and Iteration

### Row Access

Access rows by index (zero-based):

```parsley
let t = @table [{name: "Alice"}, {name: "Bob"}, {name: "Carol"}]

t[0]   // {name: "Alice"}
t[1]   // {name: "Bob"}
t[-1]  // {name: "Carol"} (last row)
```

### Iteration

Tables are iterable—use `for` to loop over rows:

```parsley
let users = @table [
    {name: "Alice", role: "admin"},
    {name: "Bob", role: "user"}
]

for (user in users) {
    <p>"{user.name} is a {user.role}"</p>
}
```

**Result:**
```html
<p>Alice is a admin</p>
<p>Bob is a user</p>
```

---

## SQL-Like Query Methods

Tables provide a fluent, chainable API inspired by SQL. Chain methods together to build expressive queries.

### where(fn)

Filter rows using a predicate function:

```parsley
let orders = @table [
    {id: 1, customer: "Alice", total: $150},
    {id: 2, customer: "Bob", total: $75},
    {id: 3, customer: "Alice", total: $200}
]

orders.where(fn(r) { r.total > $100 })
```

**Result:**

```
┌────┬──────────┬───────┐
│ id │ customer │ total │
├────┼──────────┼───────┤
│ 1  │ Alice    │ $150  │
│ 3  │ Alice    │ $200  │
└────┴──────────┴───────┘
```

Combine conditions with `&&` and `||`:

```parsley
orders.where(fn(r) { r.customer == "Alice" && r.total > $175 })
```

### orderBy(column) / orderBy(column, direction)

Sort rows by a column. Default is ascending:

```parsley
let products = @table [
    {name: "Banana", price: $1.50},
    {name: "Apple", price: $2.00},
    {name: "Cherry", price: $3.50}
]

products.orderBy("price")
```

**Result:**

# Tables

Tables are Parsley's rectangular data type: each row is a dictionary and every row has the same columns. They power CSV handling, database results, reporting, and any place you would normally reach for SQL-style transforms.

```parsley
let sales = @table [
    {product: "Widget", region: "North", amount: $1200},
    {product: "Gadget", region: "South", amount: $800},
    {product: "Widget", region: "South", amount: $1500},
    {product: "Gadget", region: "North", amount: $950}
]

sales
    .where(fn(r) { r.region == "South" })
    .orderBy("amount", "desc")
    .select(["product", "amount"])
```

---

## Ways to Create a Table

- **Literal `@table [...]`** — Parse-time rectangular validation. Every element must be a dictionary and every row must have exactly the first row's keys; missing/extra columns raise parse errors.
- **Typed literal `@table(Schema) [...]`** — Rows are cast to the schema immediately. Defaults are applied, unknown fields are dropped, and missing required fields raise `TABLE-0005` with the row/field. Column order uses sorted schema field names.
- **Constructor `table(array)`** — Runtime validation. `TABLE-0001` if the input is not an array; `TABLE-0002` if a row is not a dictionary; `TABLE-0003/0004` for missing/extra columns. `table()` with no args yields an empty table.
- **Schema call `Schema([...])`** — Calling a schema with an array returns a typed (unvalidated) table with defaults applied. Column order uses sorted schema field names (schema declaration order is not preserved in this call).
- **`parseCSV(hasHeader=true)`** — When `hasHeader` is true (default), returns a Table whose columns come from the header row and values are coerced to int/float/bool/string. With `hasHeader=false`, returns an array-of-arrays instead of a Table.
- **Database/query DSL** — `Users.all()`, `Users.where(...)`, and `@query` results are Tables with an attached schema and `FromDB=true`; indexing them returns Records marked validated (no revalidation is run on access).
- **Compat: `import @std/table`** — Provides the legacy `table` module and `table.fromDict(dict, keyName?, valueName?)` helper; prefer literals or `table()`.

Empty tables are allowed in all forms: `@table []` and `table([])` both produce `length = 0` and `columns = []`.

---

## Shape, Access, and Properties

- **Column order**
  - `@table [...]` and `table([...])`: from the first row's keys (insertion order, excluding keys starting with `__`).
- `@table(Schema) [...]`: sorted schema field names.
- `table(...).as(Schema)`: `schema.FieldOrder` if present, else sorted schema field names.
- `Schema([...])`: sorted schema field names (ignores schema declaration order).
- **Indexing**: `table[0]`, `table[-1]`, etc. Negative indices count from the end. Typed tables return a `Record` when indexed; database tables return validated Records.
- **Iteration**: `for (row in table) { ... }` iterates dictionaries or Records.
- **Properties**
  - `rows`: array of dictionaries (not a deep copy)
  - `row`: first row or `null` if empty
  - `columns`: array of column names
  - `length`: number of rows
  - `schema`: attached schema or `null`

---

## Copy-on-Chain

The first mutating-style method in a chain makes one copy; later calls in the same chain reuse it. The chain ends when you assign, pass as an argument, iterate, or access a property. Use `.copy()` for an explicit non-chain copy.

---

## Core Query Methods

- `where(fn(row))` — Keep rows where the predicate is truthy.
- `orderBy(col | [spec,...], dir?)` — Sort by a column, or by multiple specs (string or `[col, dir]`). `dir` is `"asc"` (default) or `"desc"`.
- `select([cols])` — Project to specific columns; missing columns become `null`.
- `limit(count, offset=0)` — Non-negative count/offset; slices rows.
- `offset(count)` — Skip rows; non-negative.

Example:

```parsley
let top = sales
    .where(fn(r) { r.region == "South" })
    .orderBy("amount", "desc")
    .limit(2)
    .select(["product", "amount"])
```

---

## Aggregations

- `count()` — Row count.
- `sum(col)` — Adds integers/floats; parses numeric strings; sums Money in a single currency. Mixing Money with other numeric types or multiple currencies returns an error.
- `avg(col)` — Average of numbers or Money (single currency). Returns `null` on empty input.
- `min(col)` / `max(col)` — Ignores `null`; string values are coerced to numbers when possible; returns `null` if nothing compares.

---

## Column and Group Helpers

- `column(name)` — Array of values; errors if the column is missing.
- `rowCount()` / `columnCount()` — Dimensions.
- `unique(colOrCols?)` — Removes duplicates; when columns are provided (string or array), uniqueness is based on those fields, otherwise all columns.
- `groupBy(colOrCols, fn(rows)? )`
  - Without `fn`: returns rows with the group keys plus a `rows` array.
  - With `fn`: `fn` receives an array of group rows and should return a dictionary (merged) or a single value stored under `value`.
  - Grouped tables do not preserve schemas.

Example with aggregation:

```parsley
let totals = sales.groupBy(["region", "product"], fn(rows) {
    let sum = 0
    for (r in rows) { sum = sum + r.amount }
    {total: sum}
})
```

---

## Functional Helpers

- `map(fn(row))` — Returns a new table built from the callback's dictionaries or Records. Schema is preserved only if every result is a Record with the same schema; otherwise the schema is cleared.
- `find(fn(row))` — First matching row (Record for typed tables) or `null`.
- `any(fn(row))` / `all(fn(row))` — Boolean checks (`all` is `true` for empty tables).

---

## Building and Editing Rows/Columns

- `appendRow(dict)` — Requires columns to match existing columns (or defines columns if the table was empty).
- `insertRowAt(index, dict)` — Zero-based; negative indices allowed; bounds checked; column validation like `appendRow`.
- `appendCol(name, valuesOrFn)` — `valuesOrFn` is an array matching row count **or** a function called per row. New column name must be unique.
- `insertColAfter(existing, name, valuesOrFn)` / `insertColBefore(existing, name, valuesOrFn)` — Insert at a position; validates existing column and unique new name.
- `renameCol(old, new)` — Errors if `old` is missing.
- `dropCol(name, ...)` — Remove one or more columns.

All of these return new tables (chain copies when part of a chain).

---

## Validation and Typed Tables

- `table.as(Schema)` — Attaches a schema, applying defaults and casting types (no validation yet). Column order follows schema.
- `validate()` — Validates every row of a typed table; returns a new typed table with `__errors__` stored per row.
- `isValid()` — `true` only if every row is valid; uses stored errors when present, otherwise revalidates.
- `errors()` — Array of dictionaries `{row, field, code, message}` (row indices are zero-based). Empty array when untyped.
- `validRows()` / `invalidRows()` — Filtered typed tables using stored errors when present, otherwise revalidating.

Example:

```parsley
@schema User { id: ulid, email: email }

let raw = table([
    {id: "01H7...", email: "alice@example.com"},
    {id: "01H8...", email: "not-an-email"}
]).as(User)

let checked = raw.validate()
checked.isValid()    // false
checked.errors()     // [{row: 1, field: "email", ...}]
checked.validRows()  // only the valid row
```

---

## Export and Presentation

- `toArray()` — Array of row dictionaries.
- `toJSON()` — JSON string with pretty newlines and indentation for readability.
- `toCSV()` — RFC 4180 CSV with a header row; uses CRLF line endings.
- `toMarkdown()` — GitHub-flavored table; returns `""` if there are no columns.
- `toHTML(footer?)` — HTML `<table>` with `<thead>` and `<tbody>`. Optional footer:
  - string footer: inserted raw inside `<tfoot>`
  - dictionary footer: values aligned to columns; consecutive empty cells are collapsed with `colspan`
- `toBox(opts?)` — Box-drawing table. Options: `style` (`single`/`double`/`ascii`/`rounded`), `align` (`left`/`right`/`center`), `title` (string), `maxWidth` (non-negative integer). Other `parseBoxOptions` fields are ignored for tables.

---

## Interop Notes

- Indexing a typed table (including database/query results) returns a `Record`; database-backed tables mark records as already validated when indexed.
- `parseCSV` with a header returns a Table directly; without a header you get an array-of-arrays, so wrap with `table()` if you need Table methods.
- The `table` module from `@std/table` remains for backward compatibility but is no longer required; prefer `@table [...]`, `table(...)`, and schema calls.
]
