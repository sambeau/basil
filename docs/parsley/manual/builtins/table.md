---
id: man-pars-table
title: Tables
system: parsley
type: builtin
name: table
created: 2026-01-14
version: 0.15.3
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

```
┌────────┬───────┐
│ name   │ price │
├────────┼───────┤
│ Banana │ $1.50 │
│ Apple  │ $2.00 │
│ Cherry │ $3.50 │
└────────┴───────┘
```

Sort descending:

```parsley
products.orderBy("price", "desc")
```

**Result:**

```
┌────────┬───────┐
│ name   │ price │
├────────┼───────┤
│ Cherry │ $3.50 │
│ Apple  │ $2.00 │
│ Banana │ $1.50 │
└────────┴───────┘
```

Sort by multiple columns using arrays:

```parsley
employees.orderBy(["department", "asc"], ["salary", "desc"])
```

### select(columns)

Pick specific columns (pass an array of column names):

```parsley
let users = @table [
    {id: 1, name: "Alice", email: "alice@example.com", role: "admin"},
    {id: 2, name: "Bob", email: "bob@example.com", role: "user"}
]

users.select(["name", "email"])
```

**Result:**

```
┌───────┬───────────────────┐
│ name  │ email             │
├───────┼───────────────────┤
│ Alice │ alice@example.com │
│ Bob   │ bob@example.com   │
└───────┴───────────────────┘
```

### limit(n) / offset(n)

Paginate results:

```parsley
let items = @table [
    {id: 1}, {id: 2}, {id: 3}, {id: 4}, {id: 5}
]

items.limit(3)          // First 3 rows
items.offset(2)         // Skip first 2 rows
items.offset(2).limit(2) // Rows 3 and 4 (pagination)
```

---

## Chaining: Building SQL-Like Queries

The real power of Tables comes from **method chaining**. Combine operations to build complex queries in a readable, declarative style:

```parsley
let orders = @table [
    {id: 1, customer: "Alice", product: "Widget", amount: $120, date: @2024-01-15},
    {id: 2, customer: "Bob", product: "Gadget", amount: $85, date: @2024-01-16},
    {id: 3, customer: "Alice", product: "Gadget", amount: $200, date: @2024-01-17},
    {id: 4, customer: "Carol", product: "Widget", amount: $150, date: @2024-01-18},
    {id: 5, customer: "Bob", product: "Widget", amount: $95, date: @2024-01-19}
]

// "Top 3 Widget orders by amount"
let topWidgets = orders
    .where(fn(r) { r.product == "Widget" })
    .orderBy("amount", "desc")
    .limit(3)
    .select(["customer", "amount"])
```

**Result:**

```
┌──────────┬────────┐
│ customer │ amount │
├──────────┼────────┤
│ Carol    │ $150   │
│ Alice    │ $120   │
│ Bob      │ $95    │
└──────────┴────────┘
```

### SQL Equivalence

Table method chains map directly to SQL:

| Parsley | SQL |
|---------|-----|
| `.where(fn(r) { r.x > 10 })` | `WHERE x > 10` |
| `.orderBy("name")` | `ORDER BY name ASC` |
| `.orderBy("name", "desc")` | `ORDER BY name DESC` |
| `.select(["a", "b"])` | `SELECT a, b` |
| `.limit(10)` | `LIMIT 10` |
| `.offset(20)` | `OFFSET 20` |
| `.count()` | `SELECT COUNT(*)` |
| `.sum("total")` | `SELECT SUM(total)` |

**Example: Complex query**

```parsley
// Parsley
let result = orders
    .where(fn(r) { r.date >= @2024-01-16 && r.amount > $100 })
    .orderBy("amount", "desc")
    .select(["customer", "product", "amount"])
    .limit(5)
```

**Equivalent SQL:**

```sql
SELECT customer, product, amount
FROM orders
WHERE date >= '2024-01-16' AND amount > 100
ORDER BY amount DESC
LIMIT 5
```

---

## Aggregation Methods

### count()

Count rows:

```parsley
let t = @table [{x: 1}, {x: 2}, {x: 3}]
t.count()
```

**Result:** `3`

Count filtered rows:

```parsley
orders.where(fn(r) { r.amount > $100 }).count()
```

### sum(column)

Sum a numeric column:

```parsley
let sales = @table [
    {product: "A", amount: $100},
    {product: "B", amount: $250},
    {product: "C", amount: $150}
]

sales.sum("amount")
```

**Result:** `$500.00`

### avg(column)

Calculate the average:

```parsley
sales.avg("amount")
```

**Result:** `$166.67`

### min(column) / max(column)

Find minimum and maximum values:

```parsley
sales.min("amount")  // $100.00
sales.max("amount")  // $250.00
```

### Filtered Aggregations

Combine with `.where()` for conditional aggregations:

```parsley
let orders = @table [
    {region: "North", sales: $1000},
    {region: "South", sales: $1500},
    {region: "North", sales: $800},
    {region: "South", sales: $2000}
]

// Total sales for South region
orders.where(fn(r) { r.region == "South" }).sum("sales")
```

**Result:** `$3,500.00`

---

## Column Operations

### column(name)

Extract a single column as an array:

```parsley
let users = @table [
    {name: "Alice", score: 85},
    {name: "Bob", score: 92},
    {name: "Carol", score: 78}
]

users.column("name")
```

**Result:** `["Alice", "Bob", "Carol"]`

Useful for further array operations:

```parsley
let avgScore = users.column("score").reduce(fn(a, b) { a + b }, 0) / users.length
```

### rowCount() / columnCount()

Get table dimensions:

```parsley
let t = @table [{a: 1, b: 2}, {a: 3, b: 4}]

t.rowCount()     // 2
t.columnCount()  // 2
```

---

## Row and Column Modification

### appendRow(dict)

Add a row to the end:

```parsley
let t = @table [{x: 1}]
t.appendRow({x: 2})
```

**Result:** Table with rows `[{x: 1}, {x: 2}]`

### insertRowAt(index, dict)

Insert a row at a specific position:

```parsley
let t = @table [{x: 1}, {x: 3}]
t.insertRowAt(1, {x: 2})
```

**Result:** Table with rows `[{x: 1}, {x: 2}, {x: 3}]`

### appendCol(name, fn)

Add a computed column:

```parsley
let products = @table [
    {name: "Widget", price: $10, qty: 5},
    {name: "Gadget", price: $20, qty: 3}
]

products.appendCol("total", fn(r) { r.price * r.qty })
```

**Result:**

```
┌────────┬───────┬─────┬───────┐
│ name   │ price │ qty │ total │
├────────┼───────┼─────┼───────┤
│ Widget │ $10   │ 5   │ $50   │
│ Gadget │ $20   │ 3   │ $60   │
└────────┴───────┴─────┴───────┘
```

### insertColAfter(afterCol, name, fn) / insertColBefore(beforeCol, name, fn)

Insert a column at a specific position:

```parsley
products.insertColAfter("price", "tax", fn(r) { r.price * 0.1 })
```

---

## Export Methods

### toArray()

Convert back to an array of dictionaries:

```parsley
let t = @table [{a: 1}, {a: 2}]
let arr = t.toArray()
// arr is [{a: 1}, {a: 2}]
```

### toJSON()

Export as a JSON string:

```parsley
let t = @table [{name: "Alice", age: 30}]
t.toJSON()
```

**Result:** `[{"name":"Alice","age":30}]`

### toCSV()

Export as CSV:

```parsley
let t = @table [
    {name: "Alice", age: 30},
    {name: "Bob", age: 25}
]
t.toCSV()
```

**Result:**

```
name,age
Alice,30
Bob,25
```

### toHTML()

Export as an HTML table:

```parsley
let t = @table [{name: "Alice"}, {name: "Bob"}]
t.toHTML()
```

**Result:**

```html
<table>
<thead><tr><th>name</th></tr></thead>
<tbody>
<tr><td>Alice</td></tr>
<tr><td>Bob</td></tr>
</tbody>
</table>
```

### toMarkdown()

Export as a Markdown table:

```parsley
let t = @table [{name: "Alice", age: 30}, {name: "Bob", age: 25}]
t.toMarkdown()
```

**Result:**

```
| name | age |
|------|-----|
| Alice | 30 |
| Bob | 25 |
```

### toBox()

Export as a box-drawing table (like SQL CLI output):

```parsley
let t = @table [
    {name: "Alice", age: 30, city: "London"},
    {name: "Bob", age: 25, city: "Paris"}
]
t.toBox()
```

**Result:**

```
┌───────┬─────┬────────┐
│ name  │ age │ city   │
├───────┼─────┼────────┤
│ Alice │ 30  │ London │
│ Bob   │ 25  │ Paris  │
└───────┴─────┴────────┘
```

---

## Efficiency: Copy-on-Chain

Tables use a **copy-on-chain** optimization for memory efficiency. When you chain methods, only one copy is made at the start of the chain—subsequent operations modify that same copy.

### How It Works

```parsley
let original = @table [{x: 1}, {x: 2}, {x: 3}, {x: 4}, {x: 5}]

// This chain creates ONE copy, not four:
let result = original
    .where(fn(r) { r.x > 1 })   // Copy made here
    .orderBy("x", "desc")        // Same copy modified
    .limit(3)                    // Same copy modified
    .select(["x"])               // Same copy modified
```

The original table is **never modified**:

```parsley
original.length  // Still 5
result.length    // 3
```

### Chain-Ending Operations

A chain ends when the result is:
- **Assigned to a variable**: `let x = table.where(...)`
- **Passed to a function**: `fn(table.where(...))`
- **Iterated**: `for (row in table.where(...))`
- **Accessed for a property**: `table.where(...).length`

### Multiple Independent Chains

Multiple chains from the same source create separate copies:

```parsley
let data = @table [{x: 1}, {x: 2}, {x: 3}]

let small = data.where(fn(r) { r.x < 2 })  // Copy 1
let large = data.where(fn(r) { r.x > 2 })  // Copy 2 (independent)
```

### Explicit Copying

Use `.copy()` when you need an explicit copy outside a chain:

```parsley
let backup = original.copy()
```

### Memory Considerations

| Pattern | Copies Created | Memory |
|---------|---------------|--------|
| `t.where(...).orderBy(...).limit(...)` | 1 | Efficient |
| `let a = t.where(...); let b = a.orderBy(...)` | 2 | Separate chains |
| `t.where(...).length` | 1 | Property ends chain |
| Multiple chains from same source | N | Independent copies |

---

## Integration with @query

Tables work seamlessly with Basil's `@query` DSL for database access:

```parsley
@query users {
    select: [name, email, created_at]
    where: active == true
    orderBy: created_at desc
    limit: 10
}

// Result is a Table
users.toHTML()
```

Database bindings return Tables with schema information:

```parsley
let users = Users.all()  // Returns Table

// Schema-aware column access
users.schema.columns  // ["id", "name", "email", "active", "created_at"]

// Further filtering (still a Table)
let recent = users.where(fn(u) { u.created_at > @now.addDays(-7) })
```

---

## Real-World Examples

### CSV Report Processing

```parsley
// Load sales data
let sales <== CSV("quarterly_sales.csv")

// Generate summary report
let summary = sales
    .where(fn(r) { r.region == "EMEA" })
    .orderBy("revenue", "desc")
    .limit(10)
    .select(["product", "revenue", "units_sold"])

// Export for stakeholders
<h2>"Top 10 EMEA Products"</h2>
summary.toHTML()
```

### Database Dashboard

```parsley
let orders = Orders
    .where(fn(o) { o.status == "pending" && o.created_at > @now.addDays(-7) })
    .orderBy("total", "desc")

<div class="dashboard">
    <h3>"Pending Orders This Week"</h3>
    <p>"Total: " orders.sum("total")</p>
    <p>"Count: " orders.count()</p>
    orders.select(["id", "customer_name", "total"]).toHTML()
</div>
```

### Data Transformation Pipeline

```parsley
// Transform raw API data into a report
let raw <=/= fetch("https://api.example.com/transactions")
let transactions = table(raw.json)

let report = transactions
    .where(fn(t) { t.amount > 0 })
    .appendCol("category", fn(t) { categorize(t.description) })
    .orderBy([["category", "asc"], ["amount", "desc"]])
    .select(["date", "category", "description", "amount"])

// Save as CSV
report.toCSV().writeFile("report.csv")
```

---

## Summary

Tables are Parsley's answer to structured data manipulation:

- **Create** with `@table [...]` literals or `table()` constructor
- **Import** from CSV files, database queries, or JSON APIs
- **Validate** with `@schema` for typed, defaulted, nullable fields
- **Query** using chainable SQL-like methods: `where`, `orderBy`, `select`, `limit`
- **Aggregate** with `sum`, `avg`, `min`, `max`, `count`
- **Export** to JSON, CSV, HTML, Markdown, or box-drawing format
- **Efficient** thanks to copy-on-chain optimization

Tables bridge the gap between raw data and polished output—whether you're building reports, dashboards, or data pipelines.
