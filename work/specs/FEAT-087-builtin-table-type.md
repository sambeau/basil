---
id: FEAT-087
title: "Builtin Table Type"
status: draft
priority: high
created: 2026-01-13
author: "@copilot"
version: "1.0"
---

# FEAT-087: Builtin Table Type

## Summary

Promote Parsley's `Table` type from a standard library (`@std/table`) to a builtin type with first-class language support. This includes a `@table` literal syntax, integration with `@schema` for typed tables, copy-on-chain semantics for memory efficiency, and automatic Table returns from CSV parsing and database queries.

## User Stories

**US-1:** As a Parsley developer, I want to create tables without importing a library, so that I can work with tabular data immediately.

**US-2:** As a Parsley developer, I want CSV files and database queries to return Tables directly, so that I can use table methods (`.where()`, `.orderBy()`, etc.) without manual conversion.

**US-3:** As a Parsley developer, I want `@table` literals with parse-time validation, so that column inconsistencies are caught before runtime.

**US-4:** As a Parsley developer, I want to attach schemas to tables, so that I get type validation, defaults, and nullable field support.

**US-5:** As a Parsley developer, I want efficient table operations, so that long method chains don't create unnecessary memory copies.

---

## Scope

### In Scope (V1)

1. **Builtin `Table()` constructor** — Available without import
2. **`@table` literal syntax** — With parse-time column validation
3. **`@table(Schema)` syntax** — Typed tables with validation and defaults
4. **@schema extensions** — Nullable (`?`) and default (`= value`) syntax
5. **Copy-on-chain semantics** — One copy per method chain
6. **CSV returns Table** — `CSV()` and `.parseCSV()` return Table
7. **Database returns Table** — Query methods return Table with schema
8. **Array compatibility** — `.toArray()`, iteration, JSON export

### Out of Scope (V2+)

- Lazy evaluation
- Database query pushdown
- Columnar internal representation
- `@std/table` removal (will remain as alias)

---

## Acceptance Criteria

### AC-1: Builtin Table Constructor
- [ ] `Table([{a:1},{a:2}])` works without any import
- [ ] `Table()` with no args returns empty table
- [ ] `Table(array)` validates rectangular shape at runtime
- [ ] `Table(nonArray)` returns descriptive error
- [ ] Existing `@std/table` import continues to work (alias)

### AC-2: @table Literal Syntax
- [ ] `@table [...]` parses as table literal
- [ ] Column names inferred from first row's keys
- [ ] Parse error if subsequent rows have different keys
- [ ] Parse error if rows are not dictionary literals
- [ ] Empty `@table []` creates empty table with no columns
- [ ] Single-row `@table [{a:1}]` works correctly

### AC-3: @table with Schema
- [ ] `@table(SchemaName) [...]` parses correctly
- [ ] Schema must be defined before use (parse error otherwise)
- [ ] Each row validated against schema at parse/construct time
- [ ] Missing required fields produce clear error with row number
- [ ] Default values applied to missing optional fields
- [ ] Nullable fields accept null/missing values
- [ ] `.schema` property returns attached schema (or null)

### AC-4: @schema Nullable Extension
- [ ] `fieldName: type?` parses as nullable field
- [ ] Nullable fields have `Required: false`
- [ ] Validation accepts missing/null for nullable fields
- [ ] Validation rejects missing/null for non-nullable fields
- [ ] Database generation: nullable → allows NULL
- [ ] Database generation: non-nullable → NOT NULL

### AC-5: @schema Default Value Extension
- [ ] `fieldName: type = value` parses default value
- [ ] Default applies when field is missing in validation
- [ ] Default applies when constructing `@table(Schema)` rows
- [ ] Default values can be literals: strings, numbers, booleans
- [ ] Default values can be `@now` for datetime
- [ ] Database generation: includes DEFAULT clause
- [ ] Combining: `type? = value` (nullable with default) works

### AC-6: Copy-on-Chain Semantics
- [ ] First method in chain creates copy, sets `isChainCopy`
- [ ] Subsequent chained methods mutate same copy
- [ ] Original table never modified
- [ ] Assignment ends chain (`isChainCopy` reset)
- [ ] Function argument ends chain
- [ ] Property access ends chain
- [ ] Iteration ends chain
- [ ] Multiple chains from same source create separate copies
- [ ] `.copy()` creates explicit non-chain copy

### AC-7: CSV Returns Table
- [ ] `CSV("path")` returns Table (not Array)
- [ ] `"csv,string".parseCSV()` returns Table
- [ ] Table columns match CSV header row
- [ ] Table rows are dictionaries with header keys
- [ ] Empty CSV returns empty Table with columns from header
- [ ] CSV with no header row: configurable behavior

### AC-8: Database Returns Table
- [ ] `TableBinding.all()` returns Table
- [ ] `TableBinding.where(...)` returns Table
- [ ] `TableBinding.find(id)` returns Dictionary (single row)
- [ ] Raw SQL `<=?=>` returns Table
- [ ] Returned Table has `.schema` from database schema
- [ ] Schema includes column types from SQL types

### AC-9: Array Compatibility
- [ ] `table.toArray()` returns array of dictionaries
- [ ] `table.toJSON()` returns JSON array string
- [ ] `for (row in table)` iterates dictionaries
- [ ] `table[0]` returns first row as dictionary
- [ ] `table[n]` returns nth row (0-indexed)
- [ ] `table[-1]` returns last row
- [ ] `table.length` returns row count
- [ ] `table.columns` returns column name array

### AC-10: Existing Table Methods Work
- [ ] `.where(fn)` — filter rows
- [ ] `.orderBy(col)` / `.orderBy(col, "desc")` — sort
- [ ] `.select(col1, col2, ...)` — pick columns
- [ ] `.limit(n)` — take first n rows
- [ ] `.offset(n)` — skip first n rows
- [ ] `.count()` — return row count
- [ ] `.sum(col)` / `.avg(col)` / `.min(col)` / `.max(col)` — aggregates
- [ ] `.col(name)` — return column as array
- [ ] `.toCSV()` / `.toHTML()` / `.toMarkdown()` — exports
- [ ] All methods work with copy-on-chain

---

## Specification Details

### S1: Table Type Definition

```go
type Table struct {
    Rows        []*Dictionary    // Row data
    Columns     []string         // Column order (from first row or schema)
    Schema      *DSLSchema       // Optional: attached schema
    isChainCopy bool             // Internal: copy-on-chain tracking
}

const TABLE_OBJ ObjectType = "TABLE"

func (t *Table) Type() ObjectType { return TABLE_OBJ }
func (t *Table) Inspect() string  { /* table representation */ }
func (t *Table) Copy() *Table     { /* deep copy */ }
```

### S2: @table Literal Grammar

```ebnf
TableLiteral     = "@table" [ "(" SchemaRef ")" ] "[" RowList "]" .
SchemaRef        = Identifier .
RowList          = [ DictLiteral { "," DictLiteral } [ "," ] ] .
```

**Parse-time validation:**
1. All elements must be dictionary literals
2. Extract keys from first dictionary → these are the columns
3. Every subsequent dictionary must have exactly these keys
4. If SchemaRef provided, validate against schema

**AST Node:**
```go
type TableLiteral struct {
    Token      lexer.Token       // @table token
    Schema     *Identifier       // optional schema reference
    Rows       []*DictionaryLiteral
    Columns    []string          // inferred from first row
}
```

### S3: @schema Extensions Grammar

```ebnf
SchemaField      = Identifier ":" TypeSpec [ "?" ] [ "=" DefaultValue ] .
TypeSpec         = TypeName [ "(" TypeOptions ")" ] .
TypeName         = "string" | "int" | "bool" | "email" | ... .
DefaultValue     = Literal | "@now" .
```

**Examples:**
```parsley
@schema Example {
    required_field: string              // required, no default
    nullable_field: string?             // nullable, no default
    default_field: string = "hello"     // required, has default
    both: string? = "world"             // nullable, has default
}
```

### S4: DSLSchemaField Extensions

```go
type DSLSchemaField struct {
    Name           string
    Type           string
    Required       bool       // false if type ends with ?
    Nullable       bool       // true if type ends with ?
    DefaultValue   Object     // parsed default, or nil
    DefaultExpr    string     // original expression (for SQL generation)
    ValidationType string
    EnumValues     []string
    MinLength      *int
    MaxLength      *int
    MinValue       *int64
    MaxValue       *int64
    Unique         bool
}
```

### S5: Copy-on-Chain Algorithm (Pseudocode)

```
FUNCTION tableMethod(table, operation):
    IF table.isChainCopy THEN
        # Already in chain - mutate in place
        PERFORM operation ON table
        RETURN table
    ELSE
        # Start of chain - create copy
        copy = deepCopy(table)
        copy.isChainCopy = TRUE
        PERFORM operation ON copy
        RETURN copy
    END IF

FUNCTION endChain(table):
    # Called when table is assigned, passed to function, etc.
    table.isChainCopy = FALSE
    RETURN table
```

**Chain-ending triggers:**
- `let x = table.method()` — assignment
- `fn(table.method())` — function argument
- `table.method().property` — property access (non-method)
- `for (x in table.method())` — iteration
- `return table.method()` — return statement

### S6: Table Construction from Array (Pseudocode)

```
FUNCTION Table(input):
    IF input is NULL or UNDEFINED THEN
        RETURN new Table(rows=[], columns=[])
    END IF
    
    IF input is not Array THEN
        RETURN Error("Table() requires an array")
    END IF
    
    IF input.length == 0 THEN
        RETURN new Table(rows=[], columns=[])
    END IF
    
    # Validate first element is dictionary
    IF input[0] is not Dictionary THEN
        RETURN Error("Table() requires array of dictionaries")
    END IF
    
    columns = keys(input[0])
    rows = []
    
    FOR i, element IN input:
        IF element is not Dictionary THEN
            RETURN Error("Row {i}: expected dictionary, got {type}")
        END IF
        IF keys(element) != columns THEN
            missing = columns - keys(element)
            extra = keys(element) - columns
            RETURN Error("Row {i}: column mismatch...")
        END IF
        rows.append(element)
    END FOR
    
    RETURN new Table(rows=rows, columns=columns)
```

### S7: Typed Table Construction (Pseudocode)

```
FUNCTION TableWithSchema(schema, rows):
    table = new Table(columns=schema.fieldNames())
    
    FOR i, row IN rows:
        validatedRow = {}
        
        FOR field IN schema.fields:
            value = row[field.name]
            
            IF value is MISSING or NULL THEN
                IF field.defaultValue EXISTS THEN
                    value = evaluate(field.defaultValue)
                ELSE IF field.nullable THEN
                    value = NULL
                ELSE
                    RETURN Error("Row {i}: missing required field '{field.name}'")
                END IF
            END IF
            
            IF value is not NULL THEN
                error = validateFieldType(value, field)
                IF error THEN
                    RETURN Error("Row {i}, field '{field.name}': {error}")
                END IF
            END IF
            
            validatedRow[field.name] = value
        END FOR
        
        table.rows.append(validatedRow)
    END FOR
    
    table.schema = schema
    RETURN table
```

### S8: CSV to Table (Pseudocode)

```
FUNCTION parseCSV(input):
    lines = splitLines(input)
    IF lines.length == 0 THEN
        RETURN new Table(rows=[], columns=[])
    END IF
    
    columns = parseCSVRow(lines[0])
    rows = []
    
    FOR i FROM 1 TO lines.length - 1:
        values = parseCSVRow(lines[i])
        IF values.length != columns.length THEN
            RETURN Error("Row {i}: expected {columns.length} values, got {values.length}")
        END IF
        
        row = {}
        FOR j FROM 0 TO columns.length - 1:
            row[columns[j]] = inferType(values[j])
        END FOR
        rows.append(row)
    END FOR
    
    RETURN new Table(rows=rows, columns=columns)
```

### S9: Database Query to Table (Pseudocode)

```
FUNCTION executeQuery(sql, params):
    resultSet = database.query(sql, params)
    
    columns = resultSet.columnNames()
    columnTypes = resultSet.columnTypes()
    
    rows = []
    FOR dbRow IN resultSet:
        row = {}
        FOR i, col IN columns:
            row[col] = convertDBValue(dbRow[i], columnTypes[i])
        END FOR
        rows.append(row)
    END FOR
    
    table = new Table(rows=rows, columns=columns)
    table.schema = buildSchemaFromSQL(columns, columnTypes)
    RETURN table
```

---

## Error Messages

### E1: Table Construction Errors

| Code | Message | Cause |
|------|---------|-------|
| `TABLE_NOT_ARRAY` | `Table() requires an array, got {type}` | Non-array passed to constructor |
| `TABLE_NOT_DICT` | `Table row {n}: expected dictionary, got {type}` | Non-dict in array |
| `TABLE_COLUMN_MISMATCH` | `Table row {n}: missing columns [{cols}]` | Row missing keys |
| `TABLE_EXTRA_COLUMNS` | `Table row {n}: unexpected columns [{cols}]` | Row has extra keys |

### E2: @table Literal Parse Errors

| Code | Message | Cause |
|------|---------|-------|
| `PARSE_TABLE_NOT_ARRAY` | `@table requires array literal, got {token}` | Not followed by `[` |
| `PARSE_TABLE_NOT_DICT` | `@table row {n}: expected dictionary literal` | Non-dict element |
| `PARSE_TABLE_COLUMNS` | `@table row {n}: missing column '{col}'` | Inconsistent keys |
| `PARSE_TABLE_SCHEMA` | `@table: schema '{name}' is not defined` | Unknown schema |

### E3: @schema Parse Errors

| Code | Message | Cause |
|------|---------|-------|
| `PARSE_SCHEMA_DEFAULT` | `@schema field '{f}': invalid default value` | Unparseable default |
| `PARSE_SCHEMA_DEFAULT_TYPE` | `@schema field '{f}': default type mismatch` | Default wrong type |

### E4: Validation Errors

| Code | Message | Cause |
|------|---------|-------|
| `VALIDATE_REQUIRED` | `Field '{f}' is required` | Missing non-nullable |
| `VALIDATE_TYPE` | `Field '{f}': expected {type}, got {actual}` | Wrong type |
| `VALIDATE_ENUM` | `Field '{f}': must be one of [{values}]` | Invalid enum |

---

## Validation Checklists

### Checklist 1: Parser Validation

```
□ @table literal parses without error
□ @table with schema reference parses
□ @table with invalid element shows parse error
□ @table with column mismatch shows parse error at correct row
□ @schema with nullable (?) parses
□ @schema with default (= value) parses
□ @schema with nullable + default parses
□ @schema default type mismatch shows error
```

### Checklist 2: Runtime Validation

```
□ Table([{a:1},{a:2}]) succeeds
□ Table([{a:1},{b:2}]) fails with column mismatch
□ Table([1,2,3]) fails with "not dictionary"
□ Table("string") fails with "not array"
□ Table() returns empty table
□ Table([]) returns empty table
□ @table(Schema) applies defaults
□ @table(Schema) validates types
□ @table(Schema) rejects missing required
□ @table(Schema) accepts missing nullable
```

### Checklist 3: Copy-on-Chain Validation

```
□ data.where(...) creates copy
□ data.where(...).orderBy(...) uses same copy
□ Original unchanged after chain
□ let x = data.where(...) ends chain (x.isChainCopy == false)
□ let a = data.where(...); let b = a.orderBy(...) creates 2 copies
□ Multiple chains from same source independent
□ .copy() returns non-chain copy
```

### Checklist 4: CSV Validation

```
□ CSV("file.csv") returns Table
□ "a,b\n1,2".parseCSV() returns Table
□ Table columns match CSV headers
□ Empty CSV returns empty Table
□ CSV with column count mismatch errors
□ Backward compat: can still get Array via .toArray()
```

### Checklist 5: Database Validation

```
□ Users.all() returns Table
□ Users.where(...) returns Table
□ Users.find(id) returns Dictionary (single row)
□ Raw SQL <=?=> returns Table
□ Table has .schema property
□ Schema reflects database column types
```

### Checklist 6: Method Validation

```
□ .where(fn) filters rows
□ .orderBy(col) sorts ascending
□ .orderBy(col, "desc") sorts descending
□ .select(cols...) picks columns, updates .columns
□ .limit(n) takes first n
□ .offset(n) skips first n
□ .count() returns integer
□ .sum(col) / .avg(col) / .min(col) / .max(col) work
□ .col(name) returns array
□ .toArray() returns array of dicts
□ .toJSON() returns JSON string
□ .toCSV() returns CSV string
□ .toHTML() returns HTML table string
□ All methods respect copy-on-chain
```

### Checklist 7: Schema Integration Validation

```
□ nullable field: DB generates without NOT NULL
□ non-nullable field: DB generates with NOT NULL
□ default field: DB generates with DEFAULT
□ Form validation: missing nullable OK
□ Form validation: missing required ERROR
□ Form validation: applies defaults
□ API validation: same behavior as form
```

### Checklist 8: Backward Compatibility

```
□ import @std/table still works
□ let {table} = import @std/table works
□ Existing table() function works (lowercase)
□ Existing code using Array from CSV still works with .toArray()
□ Existing code iterating database results still works
```

---

## Test Cases

*Test cases are written as Parsley code with expected outputs in comments. Each test should be run with `./pars` and output verified manually or via test harness.*

### T1: Basic Table Construction

```parsley
// T1.1: Constructor with valid array
let t = Table([{a: 1, b: 2}, {a: 3, b: 4}])
print(t.length)    // Expected: 2
print(t.columns)   // Expected: ["a", "b"]

// T1.2: Constructor with empty array
let empty = Table([])
print(empty.length)    // Expected: 0
print(empty.columns)   // Expected: []

// T1.3: Constructor validates shape (ERROR CASE)
// let bad = Table([{a: 1}, {b: 2}])
// Expected error: TABLE_COLUMN_MISMATCH

// T1.4: Constructor rejects non-array (ERROR CASE)
// let bad = Table("not array")
// Expected error: TABLE_NOT_ARRAY
```

### T2: @table Literal

```parsley
// T2.1: Basic literal
let t = @table [
    {name: "Alice", age: 30},
    {name: "Bob", age: 25}
]
print(t.length)    // Expected: 2
print(t.columns)   // Expected: ["name", "age"]

// T2.2: Empty literal
let empty = @table []
print(empty.length)  // Expected: 0

// T2.3: Single row
let single = @table [{x: 1}]
print(single.length)  // Expected: 1

// T2.4: Parse error on mismatch (PARSE ERROR CASE)
// let bad = @table [
//     {a: 1, b: 2},
//     {a: 1}  // missing b
// ]
// Expected parse error: PARSE_TABLE_COLUMNS at row 2
```

### T3: @table with Schema

```parsley
// T3.1: Schema with defaults
@schema Product {
    name: string
    price: money
    active: bool = true
}

let products = @table(Product) [
    {name: "Widget", price: $9.99},
    {name: "Gadget", price: $19.99, active: false}
]
print(products[0].active)  // Expected: true (default applied)
print(products[1].active)  // Expected: false (explicit value)

// T3.2: Schema with nullable
@schema User {
    name: string
    email: email?
}

let users = @table(User) [
    {name: "Alice", email: "alice@example.com"},
    {name: "Bob"}  // email omitted, OK
]
print(users[0].email)  // Expected: alice@example.com
print(users[1].email)  // Expected: null

// T3.3: Schema validation error (ERROR CASE)
// @schema Strict {
//     required: string
// }
// let bad = @table(Strict) [{other: "value"}]
// Expected error: VALIDATE_REQUIRED for 'required'
```

### T4: @schema Extensions

```parsley
// T4.1: Nullable field
@schema NullableTest {
    optional: string?
}
let result = NullableTest.validate({})
print(result.valid)  // Expected: true

// T4.2: Default value
@schema DefaultTest {
    name: string = "Anonymous"
}
let result2 = DefaultTest.validate({})
print(result2.valid)       // Expected: true
print(result2.value.name)  // Expected: Anonymous

// T4.3: Combined
@schema Combined {
    a: string              // required, no default
    b: string?             // nullable, no default
    c: string = "default"  // required, has default
    d: string? = "maybe"   // nullable, has default
}

let r1 = Combined.validate({a: "x"})
print(r1.valid)      // Expected: true
print(r1.value.b)    // Expected: null
print(r1.value.c)    // Expected: default
print(r1.value.d)    // Expected: maybe

let r2 = Combined.validate({a: "x", d: null})
print(r2.valid)      // Expected: true
print(r2.value.d)    // Expected: null (explicit null overrides default)
```

### T5: Copy-on-Chain

```parsley
// T5.1: Chain preserves original
let original = @table [{x: 1}, {x: 2}, {x: 3}]
let filtered = original.where(fn(r) { r.x > 1 })
print(original.length)  // Expected: 3 (unchanged)
print(filtered.length)  // Expected: 2

// T5.2: Long chain = 1 copy (verify original unchanged)
let data = @table [{x: 1}, {x: 2}, {x: 3}, {x: 4}, {x: 5}]
let result = data
    .where(fn(r) { r.x > 1 })
    .where(fn(r) { r.x < 5 })
    .orderBy("x", "desc")
    .limit(2)
print(data.length)     // Expected: 5 (original unchanged)
print(result.length)   // Expected: 2
print(result[0].x)     // Expected: 4

// T5.3: Breaking chain creates new copy
let step1 = data.where(fn(r) { r.x > 2 })
let step2 = step1.orderBy("x")
print(data.length)    // Expected: 5
print(step1.length)   // Expected: 3
print(step2.length)   // Expected: 3
```

### T6: CSV Returns Table

```parsley
// T6.1: String parseCSV
let data = "name,age\nAlice,30\nBob,25".parseCSV()
print(data.type)       // Expected: TABLE
print(data.length)     // Expected: 2
print(data[0].name)    // Expected: Alice
print(data.columns)    // Expected: ["name", "age"]

// T6.2: toArray for backward compat
let arr = data.toArray()
print(arr.type)        // Expected: ARRAY

// T6.3: File CSV (requires test file)
// let sales <== CSV("test.csv")
// print(sales.type)  // Expected: TABLE
```

### T7: Database Returns Table

```parsley
// T7.1: Setup (requires database)
// @schema User { id: ulid, name: string }
// let Users = schema.table(User, @DB, "users")

// T7.2: all() returns Table
// let all = Users.all()
// print(all.type)        // Expected: TABLE

// T7.3: Schema attached
// print(all.schema.name) // Expected: User

// T7.4: find() returns Dictionary
// let one = Users.find("01ABC...")
// print(one.type)        // Expected: DICTIONARY
```

---

## Performance Considerations

### P1: Copy-on-Chain Memory

| Scenario | Pure Immutable | Copy-on-Chain |
|----------|---------------|---------------|
| 5-step chain, 10K rows | 50K row objects | 10K row objects |
| 10-step chain, 100K rows | 1M row objects | 100K row objects |

### P2: Large Table Guidelines

- Tables > 10K rows: consider streaming/pagination
- Tables > 100K rows: use database-side filtering
- Copy-on-chain helps but doesn't eliminate memory use

---

## Migration Guide

### From @std/table

**Before:**
```parsley
let {table} = import @std/table
let data <== CSV("file.csv")
let t = table(data)
t.where(fn(r) { r.x > 0 })
```

**After:**
```parsley
let t <== CSV("file.csv")  // Already a Table
t.where(fn(r) { r.x > 0 })
```

### From Array Processing

**Before:**
```parsley
let data = Users.all()
let filtered = data.filter(fn(r) { r.active })
let sorted = filtered.sortBy(fn(r) { r.name })
```

**After:**
```parsley
let data = Users.all()  // Now a Table
let result = data
    .where(fn(r) { r.active })
    .orderBy("name")
```

---

## Documentation Requirements

### D1: Language Reference Updates

- [ ] Add `@table` to literal syntax section
- [ ] Add Table type to types section
- [ ] Document `Table()` constructor
- [ ] Document all table methods
- [ ] Document copy-on-chain behavior

### D2: @schema Reference Updates

- [ ] Document nullable (`?`) syntax
- [ ] Document default (`= value`) syntax
- [ ] Update validation behavior table
- [ ] Update database generation section

### D3: Tutorial/Guide Updates

- [ ] Add "Working with Tables" guide
- [ ] Update CSV examples
- [ ] Update database examples
- [ ] Add schema + table examples

### D4: API Reference

- [ ] Table type methods
- [ ] Table properties (.schema, .columns, .length)
- [ ] Schema validation methods with defaults

---

## Related Documents

- Design: [work/design/builtin-table-type.md](../design/builtin-table-type.md)
- Design: [work/design/schema-table-binding.md](../design/schema-table-binding.md)
- Backlog: #54 (Builtin Table type)
- Backlog: #27-31 (Table methods)

---

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-01-13 | @copilot | Initial specification |
