---
id: man-pars-std-table
title: "@std/table"
system: parsley
type: stdlib
name: table
created: 2026-02-06
version: 0.2.0
author: Basil Team
keywords:
  - table
  - data
  - query
  - aggregation
  - CSV
  - rows
  - columns
  - sort
  - filter
  - export
---

# @std/table

SQL-like data manipulation for arrays of dictionaries. Tables are immutable — all operations return new tables.

```parsley
let table = import @std/table
```

## Constructors

| Function | Args | Description |
|---|---|---|
| `table(arr)` | array of dictionaries | Create a table from an array |
| `table.fromDict(dict, keyCol?, valCol?)` | dictionary, string?, string? | Create a table from dictionary entries |

```parsley
let data = [
    {name: "Alice", age: 30, dept: "Eng"},
    {name: "Bob", age: 25, dept: "Sales"},
    {name: "Carol", age: 35, dept: "Eng"}
]

let t = table.table(data)
t.count()                        // 3

// From dictionary — each key-value pair becomes a row
let counts = {a: 1, b: 2, c: 3}
let t2 = table.table.fromDict(counts, "letter", "count")
// [{letter: "a", count: 1}, {letter: "b", count: 2}, {letter: "c", count: 3}]
```

## Query Methods

All query methods return a new Table, so they can be chained.

| Method | Args | Returns | Description |
|---|---|---|---|
| `.where(fn)` | function | Table | Filter rows matching predicate |
| `.orderBy(col, dir?)` | string, "asc" or "desc"? | Table | Sort by column (default: "asc") |
| `.select(cols)` | array of strings | Table | Select specific columns |
| `.limit(n)` | integer | Table | Take first `n` rows |
| `.offset(n)` | integer | Table | Skip first `n` rows |

```parsley
let engineers = t.where(fn(row) { row.dept == "Eng" })
engineers.count()                // 2

let sorted = t.orderBy("age", "asc")
sorted.column("name")[0]         // "Bob" (youngest first)

let subset = t.select(["name", "age"])
subset.columnCount()             // 2

let page = t.orderBy("name").offset(1).limit(1)
page.count()                     // 1
```

### Chaining

```parsley
let result = t
    .where(fn(row) { row.age > 25 })
    .orderBy("name", "asc")
    .select(["name", "dept"])
    .limit(5)
```

## Aggregation Methods

| Method | Args | Returns | Description |
|---|---|---|---|
| `.count()` | none | integer | Number of rows |
| `.sum(col)` | string | number | Sum of column values |
| `.avg(col)` | string | number | Average of column values |
| `.min(col)` | string | number | Minimum column value |
| `.max(col)` | string | number | Maximum column value |

```parsley
t.count()                        // 3
t.sum("age")                     // 90
t.avg("age")                     // 30
t.min("age")                     // 25
t.max("age")                     // 35
```

## Access Methods

| Method | Args | Returns | Description |
|---|---|---|---|
| `.rowCount()` | none | integer | Number of rows |
| `.columnCount()` | none | integer | Number of columns |
| `.column(name)` | string | array | Extract a column as an array of values |

```parsley
t.rowCount()                     // 3
t.columnCount()                  // 3
t.column("name")                 // ["Alice", "Bob", "Carol"]
```

Tables also support indexing and iteration:

```parsley
t[0]                             // {name: "Alice", age: 30, dept: "Eng"}

for (row in t) {
    row.name + ": " + row.dept
}
```

## Mutation Methods

All mutation methods return a new table (the original is unchanged).

| Method | Args | Returns | Description |
|---|---|---|---|
| `.appendRow(row)` | dictionary | Table | Add row at end |
| `.insertRowAt(index, row)` | integer, dictionary | Table | Insert row at position |
| `.appendCol(name, values)` | string, array | Table | Add column at end |
| `.insertColAfter(after, name, values)` | string, string, array | Table | Insert column after another |
| `.insertColBefore(before, name, values)` | string, string, array | Table | Insert column before another |
| `.renameCol(old, new)` | string, string | Table | Rename a column |
| `.dropCol(name)` | string | Table | Remove a column |

```parsley
let t2 = t.appendRow({name: "Dave", age: 28, dept: "Eng"})
t2.count()                       // 4

let t3 = t.appendCol("active", [true, true, false])
t3.columnCount()                 // 4

let t4 = t.renameCol("dept", "department")
t4.column("department")          // ["Eng", "Sales", "Eng"]

let t5 = t.dropCol("dept")
t5.columnCount()                 // 2
```

## Functional Methods

| Method | Args | Returns | Description |
|---|---|---|---|
| `.map(fn)` | function | Table | Transform each row |
| `.find(fn)` | function | dictionary or null | First row matching predicate |
| `.any(fn)` | function | boolean | Any row matches? |
| `.all(fn)` | function | boolean | All rows match? |
| `.unique(col)` | string | Table | Unique rows by column |
| `.groupBy(col)` | string | dictionary | Group rows by column value |

```parsley
let names = t.map(fn(row) { {name: row.name.toUpper()} })

let alice = t.find(fn(row) { row.name == "Alice" })
alice.age                        // 30

t.any(fn(row) { row.age > 30 })  // true
t.all(fn(row) { row.age > 20 })  // true

let byDept = t.groupBy("dept")
byDept.Eng.count()               // 2
```

## Export Methods

| Method | Args | Returns | Description |
|---|---|---|---|
| `.toCSV()` | none | string | CSV string with header row |
| `.toJSON()` | none | string | JSON array of objects |
| `.toMarkdown()` | none | string | Markdown table |
| `.toHTML()` | none | tag | HTML `<table>` element |
| `.toBox(opts?)` | dictionary? | string | ASCII box-drawing table |
| `.toArray()` | none | array | Convert to array of dictionaries |

```parsley
t.toCSV()
// "age,dept,name\n30,Eng,Alice\n25,Sales,Bob\n35,Eng,Carol\n"

t.toMarkdown()
// "| age | dept | name |\n|---|---|---|\n| 30 | Eng | Alice |..."

t.toJSON()
// '[{"age":30,"dept":"Eng","name":"Alice"},...]'
```

### Box Table

`toBox()` renders an ASCII table suitable for terminal output. Pass an options dictionary to customize:

```parsley
t.toBox()
// ┌─────┬───────┬───────┐
// │ age │ dept  │ name  │
// ├─────┼───────┼───────┤
// │ 30  │ Eng   │ Alice │
// │ 25  │ Sales │ Bob   │
// │ 35  │ Eng   │ Carol │
// └─────┴───────┴───────┘

t.toBox({title: "Team", style: "rounded"})
```

## Validation Methods

Tables created from schemas support validation:

| Method | Args | Returns | Description |
|---|---|---|---|
| `.as(schema)` | schema | Table | Attach a schema to the table |
| `.validate()` | none | ValidationResult | Validate all rows against schema |
| `.isValid()` | none | boolean | Are all rows valid? |
| `.errors()` | none | array | Validation errors |
| `.validRows()` | none | Table | Only valid rows |
| `.invalidRows()` | none | Table | Only invalid rows |

```parsley
@schema User {
    name: string(required)
    age: integer
}

let t = table.table([
    {name: "Alice", age: 30},
    {name: "", age: 25},
    {name: "Carol", age: 35}
])

let typed = t.as(User)
typed.isValid()                  // false (empty name on row 2)
typed.validRows().count()        // 2
typed.invalidRows().count()      // 1
```

## Key Differences from Other Languages

- **Immutable** — every operation returns a new table. The original is never modified.
- **SQL-like methods** — `.where()`, `.orderBy()`, `.select()`, `.groupBy()` mirror SQL semantics but use function predicates.
- **Multiple export formats** — tables can render to CSV, JSON, Markdown, HTML, or ASCII box drawings with a single method call.
- **Schema validation** — attach a schema with `.as()` and validate all rows at once.

## See Also

- [Data Formats](../features/data-formats.md) — CSV parsing returns tables
- [Data Model](../fundamentals/data-model.md) — schemas, records, and table types
- [Query DSL](../features/query-dsl.md) — database queries that return tables
- [@std/math](math.md) — statistical functions on arrays