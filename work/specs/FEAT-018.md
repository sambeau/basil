# FEAT-018: Standard Library Table Module

## Status
- **State:** Implemented
- **Priority:** High
- **Branch:** `feat/FEAT-018-stdlib-table`
- **Related:** PLAN-010
- **Completed:** 2025-01-03

## Summary
Add a Table module to the Parsley standard library that provides SQL-like operations on arrays of dictionaries. This establishes the foundation for the standard library pattern.

## Motivation
Working with tabular data is common in web handlers - database results, CSV imports, API responses. Currently users must write loops for filtering, sorting, aggregating. A Table module with fluent API makes this declarative and readable.

**Current approach:**
```parsley
result = []
for row in data {
    if row.status == "active" {
        result = result + [row]
    }
}
```

**With Table module:**
```parsley
result = Table(data).where({it.status == "active"}).rows
```

## Design

### Import Path
```parsley
{table} = import(@std/table)
```

The `@std/` prefix indicates standard library modules (built into the interpreter, not filesystem paths).

### Core Concept
`Table(array)` wraps an array of dictionaries and returns a Table object. All operations are **immutable** - they return new Table objects, never mutating the original.

```parsley
original = Table(data)
filtered = original.where({it.age > 18})  // original unchanged
```

### Fluent/Chainable API
Operations can be chained:
```parsley
result = Table(users)
    .where({it.active})
    .orderBy("lastName")
    .select(["firstName", "lastName", "email"])
    .limit(10)
    .rows
```

### MVP Methods

#### Constructor
- `Table(array)` - Wrap an array of dicts. Returns Table object.

#### Filtering
- `where(predicate)` - Filter rows where predicate returns truthy. Predicate receives each row as `it`.

#### Ordering
- `orderBy(column)` - Sort by column ascending
- `orderBy(column, "desc")` - Sort by column descending
- `orderBy([col1, col2])` - Multi-column sort (all ascending)
- `orderBy([[col1, "asc"], [col2, "desc"]])` - Multi-column with directions

#### Projection
- `select(columns)` - Keep only specified columns. `columns` is array of strings.
- `limit(n)` - Keep first n rows
- `limit(n, offset)` - Keep n rows starting at offset

#### Aggregations
All aggregations operate on the current rows and return a scalar value:
- `sum(column)` - Sum of numeric column
- `avg(column)` - Average of numeric column  
- `count()` - Number of rows
- `min(column)` - Minimum value
- `max(column)` - Maximum value

#### Output
- `rows` - Property that returns the underlying array of dicts
- `toHTML()` - Render as HTML `<table>` with `<thead>` and `<tbody>`
- `toCSV()` - Render as CSV string with header row

### Examples

**Database results:**
```parsley
{table} = import(@std/table)

users = query("SELECT * FROM users WHERE role = ?", [role])
table = Table(users)
    .where({it.verified})
    .orderBy("createdAt", "desc")
    .limit(20)

<div class=user-list>
    {table.toHTML()}
</div>
```

**API response transformation:**
```parsley
{table} = import(@std/table)

data = fetch("https://api.example.com/sales").json()
summary = Table(data)
    .where({it.region == selectedRegion})

<div>
    <p>Total: ${summary.sum("amount")}</p>
    <p>Count: {summary.count()}</p>
    <p>Average: ${summary.avg("amount")}</p>
</div>
```

**CSV export endpoint:**
```parsley
{table} = import(@std/table)

orders = query("SELECT * FROM orders WHERE date >= ?", [startDate])
csv = Table(orders)
    .select(["id", "customer", "total", "status"])
    .orderBy("id")
    .toCSV()

basil.response.headers["Content-Type"] = "text/csv"
basil.response.headers["Content-Disposition"] = "attachment; filename=orders.csv"
{csv}
```

**Loading and displaying a CSV file:**
```parsley
{table} = import(@std/table)

// Read CSV file and convert to table
data <== CSV(@./data/sales.csv)
sales = table(data)
    .where({it.amount > 100})
    .orderBy("date", "desc")

<h2>High Value Sales</h2>
{sales.toHTML()}
```

## Implementation Notes

### Architecture
- Implement core operations in Go for performance
- Table object is a new object type in the evaluator
- Standard library modules use `@std/` prefix, resolved by evaluator (not filesystem)

### toHTML() Output
```html
<table>
  <thead>
    <tr><th>column1</th><th>column2</th></tr>
  </thead>
  <tbody>
    <tr><td>value1</td><td>value2</td></tr>
  </tbody>
</table>
```
- Column order from first row's keys (or select order if used)
- Values HTML-escaped
- No styling classes (user wraps with styled container)

### toCSV() Output
- RFC 4180 compliant
- Header row from column names
- Values quoted if they contain commas, quotes, or newlines
- Newline: CRLF

### Error Handling
- `Table(non-array)` → error
- `where(non-function)` → error  
- `orderBy(missing-column)` → error
- `sum/avg(non-numeric-column)` → error or skip non-numeric values (TBD)
- `select(non-existent-column)` → include column with null values

## Deferred (Future Enhancements)
- `groupBy(column)` - Group rows, returns dict of Tables
- `join(otherTable, onColumn)` - SQL-like joins
- Column transforms: `transform(column, fn)` or `addColumn(name, fn)`
- `distinct()` - Remove duplicate rows
- `first()` / `last()` - Single row access
- `toJSON()` - JSON string output
- `fromCSV(string)` - Parse CSV into Table

## Test Cases
1. Basic Table creation from array
2. Empty array handling
3. where() with various predicates
4. orderBy() ascending and descending
5. orderBy() with multiple columns
6. select() column projection
7. limit() with and without offset
8. Aggregation functions with numeric data
9. Aggregation on empty table (count=0, sum/avg=0 or error)
10. Chain multiple operations
11. Immutability verification
12. toHTML() output structure
13. toCSV() with special characters (quotes, commas, newlines)
14. Error cases (non-array, invalid column, etc.)

## Acceptance Criteria
- [ ] `@std/table` import resolves to Table module
- [ ] Table() constructor wraps arrays
- [ ] All MVP methods implemented
- [ ] Operations are immutable
- [ ] toHTML() produces valid HTML table
- [ ] toCSV() produces RFC 4180 compliant output
- [ ] Error messages are clear
- [ ] All test cases pass
- [ ] Documentation in docs/parsley/reference.md
- [ ] Example in docs/guide/

## References
- Similar to: LINQ (C#), Pandas (Python), Lodash (JS)
- RFC 4180: CSV format specification
