# Design Investigation: Builtin Table Type

**Status:** Investigation  
**Date:** 2026-01-13  
**Author:** AI (design exploration)

## Summary

Should Parsley's `table` type be promoted from a standard library (`@std/table`) to a builtin type, similar to Money, DateTime, Path, and Query?

## Current State

### What Exists Today

1. **Table is already a core type** (TABLE_OBJ in evaluator.go):
   ```go
   type Table struct {
       Rows    []*Dictionary  // Array of dictionaries (each row is a dict)
       Columns []string       // Column order (from first row or select())
   }
   ```

2. **Table constructor is in stdlib** (requires `import @std/table`):
   ```parsley
   let {table} = import @std/table
   let t = table(arrayOfDicts)
   ```

3. **CSV returns Array**, not Table:
   ```parsley
   let data <== CSV("data.csv")  // → Array of Dictionary
   let data = "a,b\n1,2".parseCSV()  // → Array of Dictionary
   ```

4. **Database queries return Array**, not Table:
   ```parsley
   let users = Users.all()  // → Array of Dictionary
   ```

### The Awkward Pattern

Users who want Table methods must do:
```parsley
let {table} = import @std/table
let data <== CSV("data.csv")
let t = table(data)  // now can use .where(), .orderBy(), etc.
```

## Analysis

### Arguments FOR Builtin Table

1. **Parsley is a data language.** The README says Parsley is for "web and data". Tables are THE fundamental data structure for:
   - CSV files
   - Database results
   - API responses (JSON arrays)
   - HTML tables
   - Spreadsheet data

2. **Natural integration points.** These could return Tables directly:
   - `CSV()` / `.parseCSV()` 
   - Database queries (`Users.all()`, raw SQL)
   - JSON arrays of objects from APIs
   - `import "data.csv"`

3. **Column validation.** A true Table type could enforce:
   - Rectangular shape (all rows have same columns)
   - Column type consistency (optional)
   - Schema validation against database schemas

4. **Efficient implementation.** Freed from array-of-dict representation:
   - Columnar storage: `map[string][]Object` instead of `[]*Dictionary`
   - Efficient column operations (sum an entire column without row iteration)
   - Memory efficiency for large datasets
   - Lazy column evaluation

5. **@query integration.** Query DSL could:
   - Return Table with type-safe column access
   - Preserve column metadata through transforms
   - Enable join typing

6. **Export remains natural.** Tables can still:
   - Export as array of dictionaries for JSON
   - Iterate row-by-row for templates
   - Convert to CSV, HTML, etc.

### Arguments AGAINST Builtin Table

1. **Standard libraries exist for a reason.** Not every Parsley user needs tables. Simple handlers may only deal with strings and dictionaries.

2. **Current approach is flexible.** Array of Dictionary is:
   - Easy to understand
   - Compatible with JSON natively
   - Doesn't require learning a new type

3. **Complexity cost.** Adding a builtin type means:
   - More core code to maintain
   - Another type to document
   - Type coercion rules to define

4. **Migration burden.** Existing code using `@std/table` would need updating (though we could keep the import working as an alias).

## What Could Change

### If Table Were Builtin

```parsley
// CSV returns Table directly
let sales <== CSV("sales.csv")
sales.where(fn(r) { r.amount > 100 }).sum("amount")

// Database queries return Table
let users = Users.where(fn(u) { u.active })
users.orderBy("name").limit(10)

// JSON API returns Table when appropriate
let {data} <=/= API("https://api.example.com/users").json
let t = Table(data)  // explicit conversion, or...
let t <== API("...").table  // table format hint

// Table literal syntax? (maybe not needed)
let t = @table [
    {name: "Alice", age: 30},
    {name: "Bob", age: 25}
]
```

### Possible Internal Representation

```go
type Table struct {
    // Option A: Columnar (efficient for analytics)
    Columns map[string]*Column
    RowCount int
    ColumnOrder []string
    
    // Option B: Row-based with metadata (current, with enhancements)
    Rows []*Dictionary
    Schema *TableSchema  // optional: column names, types, constraints
}

type Column struct {
    Name string
    Type ObjectType  // optional type enforcement
    Values []Object
}

type TableSchema struct {
    Columns []ColumnDef
    Name string  // e.g., from database table name
}
```

### What Would Generate Tables

| Source | Current Return | With Builtin Table |
|--------|---------------|-------------------|
| `CSV("file.csv")` | Array | Table |
| `"...".parseCSV()` | Array | Table |
| `Users.all()` | Array | Table |
| `Users.where(...)` | Array | Table |
| Raw SQL (`<=?=>`) | Array | Table |
| `Table(array)` | Table | Table (identity) |
| JSON API (array of objects) | Array | Array (opt-in Table) |

### What Would Consume Tables

| Consumer | Notes |
|----------|-------|
| `for (row in table)` | Iterate rows as dictionaries |
| `table.toJSON()` | Array of objects |
| `table.toCSV()` | CSV string |
| `table.toHTML()` | HTML table |
| `<table>` tag | Direct HTML rendering |
| `table[0]` | First row as dictionary |
| `table.col("name")` | Column as array |

## Comparison: Other Languages

### R data.frame
- **Builtin:** Yes, fundamental type
- **Creation:** `data.frame(x=c(1,2), y=c("a","b"))`
- **Used by:** Everything—stats, plotting, I/O
- **Philosophy:** Data analysis IS R's purpose

### Python pandas DataFrame
- **Builtin:** No, but ubiquitous library
- **Creation:** `pd.DataFrame({"x": [1,2], "y": ["a","b"]})`
- **Used by:** All data work in Python
- **Philosophy:** Python is general-purpose; pandas adds data

### SQL
- **Builtin:** Yes, tables are the only thing
- **Philosophy:** Tables ARE the data model

### JavaScript
- **Builtin:** No, uses Array of Object
- **Philosophy:** General-purpose; tabular data is one use case

### Parsley's Position
Parsley is closer to R/SQL than Python/JS—it's purpose-built for web and data. This argues for Table being builtin.

## Recommendation

**Promote Table to builtin status**, with these guidelines:

1. **Keep Table constructor available without import:**
   ```parsley
   let t = Table(arrayOfDicts)  // works without import
   ```

2. **CSV and database queries return Table:**
   ```parsley
   let data <== CSV("file.csv")  // Table, not Array
   let users = Users.all()       // Table, not Array
   ```

3. **Preserve Array compatibility:**
   ```parsley
   let arr = table.toArray()     // explicit conversion
   for (row in table) { ... }    // iteration still works
   table.toJSON()                // returns JSON array
   ```

4. **Deprecate @std/table gradually:**
   ```parsley
   let {table} = import @std/table  // works, but unnecessary
   // Warning: @std/table is deprecated; Table is now builtin
   ```

5. **Consider columnar representation internally** for efficiency, but maintain row-iteration semantics externally.

## Open Questions (Expanded)

### 1. Table Literal Syntax

**Question:** Should there be a `@table` literal, or is `Table([...])` sufficient?

**Proposal:** Yes, introduce `@table` literal syntax.

```parsley
// Option A: @table with array of dicts
let users = @table [
    {name: "Alice", age: 30, active: true},
    {name: "Bob", age: 25, active: false},
    {name: "Carol", age: 35, active: true}
]

// Option B: @table with explicit columns (more validatable)
let users = @table {
    columns: ["name", "age", "active"],
    rows: [
        ["Alice", 30, true],
        ["Bob", 25, false],
        ["Carol", 35, true]
    ]
}

// Option C: Hybrid—infer columns from first row
let users = @table [
    {name: "Alice", age: 30},
    {name: "Bob", age: 25}    // Parser validates: must have same keys
]
```

**Advantages of literal syntax:**

| Advantage | Explanation |
|-----------|-------------|
| **Parse-time validation** | Compiler can check column consistency before runtime |
| **Clear intent** | `@table` signals "this is tabular data" vs arbitrary array |
| **Optimization hints** | Parser knows structure, can pre-allocate |
| **Error messages** | "Row 3 missing column 'age'" vs runtime key error |

**Validation at parse time:**
```parsley
// This would be a parse ERROR, not a runtime surprise:
let bad = @table [
    {name: "Alice", age: 30},
    {name: "Bob"}              // ERROR: missing column 'age'
]
```

**Importing arrays as tables:**
```parsley
// Explicit conversion (validates at runtime)
let data = [{name: "Alice"}, {name: "Bob"}]
let t = Table(data)  // runtime validation

// vs literal (validates at parse time)
let t = @table [
    {name: "Alice"},
    {name: "Bob"}
]
```

**Recommendation:** Option C (hybrid)—`@table [...]` with array-of-dict syntax, but with parse-time column validation. This is familiar to users (looks like JSON) but adds safety.

---

### 2. Integration with Existing @schema

**Question:** Should `table.col("age")` return a typed column? Could `@table` use a schema?

**Current state:** Parsley already has `@schema` declarations:

```parsley
// Existing @schema syntax (from stdlib_dsl_schema.go)
@schema User {
    id: ulid
    name: string
    email: email
    age: int(min: 0, max: 150)
    role: enum("user", "admin")
}
```

The existing `DSLSchemaField` struct already supports:
- `Required` (bool) — currently defaults to `true`
- `ValidationType` — email, url, phone, slug, enum
- `EnumValues` — for enum types
- `MinLength`, `MaxLength` — string constraints
- `MinValue`, `MaxValue` — integer constraints  
- `Unique` — database unique constraint

**What's missing:** Nullable fields and default values.

---

#### Proposed Extensions to @schema

**Nullable fields** with `?` suffix:
```parsley
@schema User {
    id: ulid
    name: string              // required (default)
    email: email?             // nullable - can be null/missing
    phone: phone?             // nullable
}
```

**Default values** with `= value`:
```parsley
@schema User {
    id: ulid
    name: string
    role: enum("user", "admin") = "user"   // defaults to "user"
    active: bool = true                     // defaults to true
    created_at: datetime = @now             // defaults to current time
}
```

**Combined:**
```parsley
@schema Post {
    id: ulid
    title: string(min: 1, max: 200)
    body: text
    author_id: ulid
    status: enum("draft", "published", "archived") = "draft"
    published_at: datetime?                 // nullable, no default
    view_count: int = 0                     // not nullable, defaults to 0
}
```

---

#### Use Cases for Nullable and Defaults

**1. Database Schema Generation**

```parsley
@schema User {
    id: ulid
    email: email
    name: string?              // → TEXT (allows NULL)
    role: enum("user","admin") = "user"  // → TEXT DEFAULT 'user'
    created_at: datetime = @now           // → TIMESTAMP DEFAULT CURRENT_TIMESTAMP
}

// Generates SQL:
// CREATE TABLE users (
//     id TEXT PRIMARY KEY,
//     email TEXT NOT NULL,
//     name TEXT,                          -- allows NULL
//     role TEXT DEFAULT 'user' CHECK(role IN ('user', 'admin')),
//     created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
// )
```

**2. Form Validation**

```parsley
@schema ContactForm {
    name: string(min: 1, max: 100)
    email: email
    phone: phone?              // optional field
    message: text(min: 10)
    newsletter: bool = false   // unchecked checkbox = false
}

post "/contact" {
    let result = ContactForm.validate(body)
    
    if (!result.valid) {
        // Errors only for required fields that are missing
        // phone won't error if omitted
        <div class=errors>...</div>
    } else {
        // result.value has defaults applied:
        // newsletter = false even if not in form
        saveContact(result.value)
    }
}
```

**3. API Input Handling**

```parsley
@schema CreateUserRequest {
    email: email
    name: string
    role: enum("user", "admin") = "user"  // API consumers can omit
    preferences: json = {}                 // default empty object
}

post "/api/users" {
    let input = CreateUserRequest.validate(body.json)
    // input.value.role is "user" if not specified
    // input.value.preferences is {} if not specified
}
```

**4. Table Data with Defaults**

```parsley
@schema Product {
    sku: string
    name: string
    price: money
    in_stock: bool = true
    discount: number = 0
}

// Table validates against schema, applies defaults
let products = @table(Product) [
    {sku: "ABC", name: "Widget", price: $9.99},
    // in_stock: true, discount: 0 (defaults applied)
    {sku: "DEF", name: "Gadget", price: $19.99, in_stock: false}
]
```

---

#### Implementation Notes

The existing `DSLSchemaField` struct would need:

```go
type DSLSchemaField struct {
    Name           string
    Type           string
    Required       bool      // true by default, false if type ends with ?
    Nullable       bool      // NEW: true if type ends with ?
    DefaultValue   Object    // NEW: default value, or nil
    ValidationType string
    EnumValues     []string
    MinLength      *int
    MaxLength      *int
    MinValue       *int64
    MaxValue       *int64
    Unique         bool
}
```

**Parser changes:**
- `email?` → sets `Nullable: true`, `Required: false`
- `= "value"` → sets `DefaultValue` to the parsed expression

**Validation behavior:**

| Field Spec | Missing Value | Null Value | Validation |
|------------|---------------|------------|------------|
| `name: string` | ERROR: required | ERROR: required | Must be string |
| `name: string?` | OK (null) | OK (null) | If present, must be string |
| `name: string = "Anonymous"` | OK (default applied) | ERROR: not nullable | Must be string |
| `name: string? = "Anonymous"` | OK (default applied) | OK (null) | If present, must be string |

---

#### Tables with @schema

```parsley
@schema User {
    id: ulid
    name: string
    email: email?
    active: bool = true
}

// Option 1: @table with schema reference
let users = @table(User) [
    {id: "01H...", name: "Alice", email: "alice@example.com"},
    {id: "01H...", name: "Bob"}  // email: null, active: true
]

// Option 2: Inline, validated against shape
let users = @table [
    {id: "01H...", name: "Alice"},
    {id: "01H...", name: "Bob"}
]
// No schema = no defaults, but still validates column consistency

// Schema is attached to table
users.schema  // → User schema object

// Schema-aware operations
users.col("email")  // knows this is nullable
users.sum("active") // ERROR: can't sum boolean
```

---

#### Schema Integration Summary

| Feature | Current @schema | Proposed Extension |
|---------|-----------------|-------------------|
| Required fields | Yes (all fields) | Yes (default) |
| Nullable fields | No | `type?` syntax |
| Default values | No | `= value` syntax |
| Type validation | Yes | Yes |
| Constraints | Yes (min, max, etc.) | Yes |
| Database generation | Yes | Enhanced with defaults |
| Form validation | Yes | Enhanced with optionals |
| Table typing | No | `@table(Schema) [...]` |

---

### 3. Schema Integration with Databases

**Question:** Should Tables from database queries carry schema metadata?

**Proposal:** Yes—this is natural and powerful.

```parsley
// Database query returns Table with schema attached
let users = Users.all()

// Schema is available
users.schema
// → {id: Integer, name: String, email: String?, created_at: DateTime}

// Benefits:

// 1. Column validation
users.col("naem")  // ERROR: did you mean 'name'?

// 2. Type-aware operations
users.sum("id")      // works (Integer column)
users.sum("name")    // ERROR: can't sum String column

// 3. Join safety
let orders = Orders.all()
users.join(orders, "id", "user_id")  // validates key types match

// 4. IDE autocomplete
users.  // autocomplete shows: id, name, email, created_at
```

**Where schema comes from:**

| Source | Schema Origin |
|--------|---------------|
| Database query | SQL schema (automatic) |
| CSV import | Header row (column names only, types inferred) |
| JSON API | None (or optional user-provided) |
| `@table` literal | Explicit schema or inferred from data |

**Schema propagation through transforms:**
```parsley
let users = Users.all()
// schema: {id, name, email, active}

let active = users.where(fn(u) { u.active })
// schema: {id, name, email, active} (preserved)

let names = users.select("id", "name")
// schema: {id, name} (subset)

let enhanced = users.withColumn("displayName", fn(u) { u.name.upper() })
// schema: {id, name, email, active, displayName} (extended)
```

---

### 4. Lazy Evaluation (Future Consideration)

**Question:** Should `table.where(...).orderBy(...)` be lazy until materialized?

**Summary:** Lazy evaluation offers memory efficiency and query optimization but adds mental complexity. 

**Recommendation for V1:** Keep eager evaluation. It's predictable and simple.

**Deferred to V2:** Consider opt-in laziness via `.lazy()` method, particularly for database-backed tables where lazy chains could compile to SQL.

See Appendix A for detailed lazy evaluation analysis.

---

### 5. Immutability and Chain Copying

**Question:** Current Table methods return new Tables. Should this continue?

**Current behavior (immutable):**
```parsley
let t1 = @table [{x: 1}, {x: 2}, {x: 3}]
let t2 = t1.where(fn(r) { r.x > 1 })

// t1 is unchanged: [{x: 1}, {x: 2}, {x: 3}]
// t2 is new:       [{x: 2}, {x: 3}]
```

**The problem:** Long chains create many intermediate copies:
```parsley
let result = data
    .where(fn(r) { r.active })     // copy 1
    .orderBy("name")                // copy 2
    .select("id", "name", "email")  // copy 3
    .limit(100)                     // copy 4
// 4 intermediate Tables created, 3 thrown away
```

---

#### Three Approaches

**Approach A: Pure Immutability (current)**
- Every operation returns a new Table
- Original never modified
- Simple mental model
- Memory inefficient for chains

**Approach B: Copy-on-Write (structural sharing)**
- Tables share underlying data until modified
- Complex implementation
- Memory efficient but unpredictable timing

**Approach C: Copy-on-Chain (proposed)**
- A chain makes ONE copy at the start
- All operations in the chain mutate that single copy
- Original is never touched
- Predictable: exactly one copy per chain

---

#### Copy-on-Chain: How It Works

```parsley
let original = @table [{x: 1}, {x: 2}, {x: 3}, {x: 4}, {x: 5}]

// Starting a chain creates ONE copy
let result = original
    .where(fn(r) { r.x > 1 })     // copy made HERE, then filtered in-place
    .orderBy("x", "desc")          // sorts the copy in-place
    .limit(3)                      // truncates the copy in-place
// Total: 1 copy, regardless of chain length

// original is untouched: [{x: 1}, {x: 2}, {x: 3}, {x: 4}, {x: 5}]
// result is the copy:    [{x: 5}, {x: 4}, {x: 3}]
```

**Key insight:** The "chain" is determined by method chaining syntax. When you write `a.b().c().d()`, the runtime knows this is a chain and can optimize.

---

#### Implementation Strategy

The runtime tracks whether we're "in a chain":

```go
type Table struct {
    Rows      []*Dictionary
    Columns   []string
    Schema    *DSLSchema  // optional
    isChainCopy bool      // true if this table is a working copy in a chain
}

func (t *Table) where(predicate func) *Table {
    if t.isChainCopy {
        // We're already in a chain - mutate in place
        t.Rows = filter(t.Rows, predicate)
        return t
    } else {
        // Start of chain - make a copy
        copy := t.Copy()
        copy.isChainCopy = true
        copy.Rows = filter(copy.Rows, predicate)
        return copy
    }
}
```

**When does a chain end?**
- Assignment to a variable: `let x = table.where(...)`
- Passing to a function: `process(table.where(...))`
- Property access: `table.where(...).length`
- Iteration: `for (row in table.where(...))`

At chain end, `isChainCopy` is reset to `false`.

---

#### Comparison

| Approach | Copies for 5-step chain | Memory | Predictability | Complexity |
|----------|------------------------|--------|----------------|------------|
| Pure Immutable | 5 | High | High | Low |
| Copy-on-Write | 1-5 (varies) | Low | Low | High |
| Copy-on-Chain | 1 | Low | High | Medium |

---

#### Copy-on-Chain Examples

**Example 1: Simple chain**
```parsley
let filtered = data.where(fn(r) { r.x > 0 }).orderBy("x")
// 1 copy made at .where(), .orderBy() mutates same copy
```

**Example 2: Multiple chains**
```parsley
let a = data.where(fn(r) { r.x > 0 })   // chain 1: 1 copy
let b = data.where(fn(r) { r.x < 0 })   // chain 2: 1 copy
// 2 copies total, original unchanged
```

**Example 3: Breaking the chain**
```parsley
let step1 = data.where(fn(r) { r.x > 0 })  // chain 1: 1 copy
let step2 = step1.orderBy("x")              // chain 2: 1 copy (step1 is not a chain copy)
// 2 copies because assignment ended chain 1
```

**Example 4: Continuous chain**
```parsley
let result = data
    .where(fn(r) { r.x > 0 })
    .orderBy("x")
    .select("x", "y")
    .limit(10)
    .where(fn(r) { r.y != null })
// 1 copy, 5 in-place mutations
```

---

#### Edge Cases

**Forking a chain:**
```parsley
let base = data.where(fn(r) { r.active })
let pathA = base.orderBy("name")   // new chain from base
let pathB = base.orderBy("date")   // another new chain from base
// base: 1 copy (now frozen, isChainCopy = false)
// pathA: 1 copy of base
// pathB: 1 copy of base
// Total: 3 copies
```

**Explicit copy:**
```parsley
let copy = table.copy()  // explicit copy, not a chain
copy.isChainCopy = false
```

---

#### Advantages of Copy-on-Chain

| Advantage | Explanation |
|-----------|-------------|
| **Predictable** | Exactly one copy per chain, always |
| **Memory efficient** | Long chains don't multiply memory |
| **Simple mental model** | "Chains share one copy" is easy to understand |
| **Original safety** | Original table is never modified |
| **No timing surprises** | Copy happens at chain start, not lazily |

#### Disadvantages

| Disadvantage | Explanation |
|--------------|-------------|
| **Implementation complexity** | Runtime must track chain state |
| **Slightly magic** | Behavior depends on syntax (chained vs separate statements) |
| **Debugging** | Need to understand what constitutes a "chain" |

---

#### Recommendation

**Adopt Copy-on-Chain for V1.** It provides:
- Memory efficiency of mutation
- Safety of immutability  
- Predictability (unlike copy-on-write)
- Reasonable implementation complexity

The "magic" is minimal and explainable: *"When you chain table methods, one copy is made and reused."*

```parsley
// Users learn this simple rule:
let result = data.a().b().c()  // one copy
// vs
let step1 = data.a()  // one copy
let step2 = step1.b() // another copy
```

---

## Implementation Phases (if approved)

**Phase 1: Foundation**
- Make `Table()` constructor available without import
- Add `@table [...]` literal syntax with column validation
- Implement copy-on-chain semantics

**Phase 2: Schema Integration**  
- Add nullable (`type?`) syntax to @schema parser
- Add default values (`= value`) syntax to @schema parser
- Enable `@table(Schema) [...]` typed tables
- Apply defaults during table construction

**Phase 3: Data Source Integration**
- Change CSV parsing to return Table
- Change database queries to return Table with schema
- Add `.toArray()` for explicit conversion

**Phase 4: Cleanup**
- Deprecate `@std/table` import (alias to builtin)
- Update documentation

**Phase 5: Future (V2)**
- Optional lazy evaluation via `.lazy()`
- Database query pushdown
- Columnar storage optimization

## Related Backlog Items

- #54: Builtin Table type (this investigation expands on that item)
- #27-31: Table methods (groupBy, join, distinct, first/last)

---

## Appendix A: Lazy Evaluation (Deferred)

*This section captures lazy evaluation design for future reference. Deferred to V2 due to complexity.*

### What is Lazy Evaluation?

```parsley
// Eager (V1): each step creates a new Table (or mutates chain copy)
let result = users
    .where(fn(u) { u.active })     // executes now
    .orderBy("name")                // executes now
    .limit(10)                      // executes now

// Lazy (V2): operations build a plan, execute once
let result = users.lazy()
    .where(fn(u) { u.active })     // returns LazyTable (no work yet)
    .orderBy("name")                // extends plan (no work yet)
    .limit(10)                      // extends plan (no work yet)
// Materialization triggers execution
for (user in result) { ... }       // NOW executes optimized plan
```

### Advantages

| Advantage | Explanation |
|-----------|-------------|
| **Memory efficiency** | No intermediate Tables |
| **Query optimization** | Can reorder operations (e.g., limit before sort) |
| **Database pushdown** | Lazy chain can become SQL query |
| **Short-circuit** | `.first()` doesn't process all rows |

### Database Pushdown Example

```parsley
// With lazy evaluation, this:
let result = Users.lazy()
    .where(fn(u) { u.active })
    .orderBy("name")
    .limit(10)

// Could compile to SQL:
// SELECT * FROM users WHERE active = true ORDER BY name LIMIT 10
```

### Disadvantages

| Disadvantage | Explanation |
|--------------|-------------|
| **Debugging complexity** | "When did this actually run?" |
| **Side effect timing** | Functions in `.where()` run later than expected |
| **Mental model** | Users expect immediate execution |

### Side Effect Surprise

```parsley
let counter = 0
let result = users.lazy().where(fn(u) { 
    counter = counter + 1  // When does this run?
    u.active 
})
print(counter)  // 0! (lazy hasn't run yet)
for (u in result) { }
print(counter)  // NOW it's incremented
```

### Why Defer?

1. Copy-on-chain provides most memory benefits with less complexity
2. Lazy evaluation requires careful API design
3. Side effect timing is a common source of bugs
4. V1 should prioritize simplicity and predictability

### Future Direction

If lazy evaluation is added:
- Make it opt-in via `.lazy()` method
- Database-backed tables could be lazy by default
- Provide `.materialize()` for explicit evaluation
- Consider caching: lazy tables remember their result after first evaluation

---

*This document is an investigation, not a decision. Feedback welcome.*
