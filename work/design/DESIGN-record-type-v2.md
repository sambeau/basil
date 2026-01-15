# Design: Record Type for Parsley

**Status:** Deprecated — See DESIGN-record-type-v3.md  
**Date:** 2025-01-15  
**Supersedes:** DESIGN-record-type.md  
**Superseded By:** DESIGN-record-type-v3.md  
**Related:** BACKLOG #21, FEAT-002

> ⚠️ **This document is deprecated.** It contains investigation notes and alternative designs explored during the design process. For the final agreed design, see [DESIGN-record-type-v3.md](DESIGN-record-type-v3.md).

## 1. Overview

This document defines the **Record type** for Parsley — a typed wrapper around data that carries its schema and validation state. The primary motivation is form handling, but records have broader applications.

### 1.1 Design Goals

Following Parsley's aesthetic:
- **Simplicity:** Easy to understand, minimal concepts
- **Minimalism:** No more features than necessary
- **Completeness:** Handles real-world use cases fully
- **Composability:** Works well with existing features

### 1.2 Core Concept

```parsley
// Define a schema (same syntax as Query DSL)
@schema User {
    name: string(min: 2, required)
    email: email(required)
}

// Create a record by calling the schema
let user = User({name: "Alice", email: "alice@example.com"})

user.name           // "Alice" (data access)
user.isValid()      // false (not yet validated)
user.errors()       // {} (empty, but not validated)

// Validate the record
let validated = user.validate()
validated.isValid() // true or false
validated.errors()  // {} or {name: "Must be at least 2 characters"}
```

**Key insight:** A Record is Schema + Data + Errors. Nothing more.

## 2. Decision Summary

### 2.1 Core Record Design

| Decision | Choice |
|----------|--------|
| What is a Record? | Schema + Data + Errors (no changes tracking) |
| Property access | Data fields direct (`record.name`), metadata via methods (`record.errors()`) |
| Mutability | Immutable — modifications return new records |
| Dict compatibility | Yes — Record IS-A Dictionary for data access, spreads work |

### 2.2 Creation API

| Syntax | Returns | Purpose |
|--------|---------|---------|
| `Schema({...})` | Record | Primary idiom — dict in |
| `Schema([...])` | Table | Primary idiom — array in |
| `{...}.as(Schema)` | Record | Chaining: `fetch().parse().as(Schema)` |
| `table(data).as(Schema)` | Table | Dynamic schema binding |
| `@table(Schema) [...]` | Table | Literal with compile-time binding (unchanged) |

### 2.3 Validation API

| Syntax | Returns | Description |
|--------|---------|-------------|
| `record.validate()` | Record | Single record with errors populated |
| `table.validate()` | Table | Bulk validate all rows |
| `table.validRows()` | Table | Rows that passed validation |
| `table.invalidRows()` | Table | Rows that failed (with error info) |

### 2.4 Schema & Metadata

| Decision | Choice |
|----------|--------|
| Metadata syntax | Pipe: `string(required) \| {title: "Name"}` |
| Metadata type | Open dictionary — any key-value pairs |
| Core V1 metadata | `title`, `placeholder`, `format`, `hidden`, `help` |
| i18n | Not in V1 — pipe syntax allows functions later |
| Metadata override | Layering: explicit override > schema metadata > titlecase field name |

### 2.5 Validation Behavior

| Decision | Choice |
|----------|--------|
| Initial state | Records start **unvalidated** — `isValid()` returns false |
| Auto-revalidate | Mutations (`update()`) auto-revalidate the record |
| Query return | Records from queries are **auto-validated** (data is trusted) |
| Default values | Applied on **creation**, not validation |
| Filtering | Implicit — schema fields act as whitelist |
| Type casting | Automatic based on field types |
| Declarative validation | In `@schema{}` — covers 90% of cases |
| Custom validation | Post-processing with `record.withError()` |
| Cross-field validation | Handle in code after `record.validate()` |
| DB constraints | Future phase |
| `@insert` validation | Always validates, even if record already validated |

### 2.6 Form Binding

| Decision | Choice |
|----------|--------|
| Form context | `<form @record={record}>` establishes context via AST |
| What gets bound | Values + validation attributes + errors |
| Input rewriting | `<input @name="x"/>` → full input with value, attrs, name |
| Error display | `<Error @name="x"/>` → conditional error span |

### 2.7 Not Included in V1

| Feature | Reason |
|---------|--------|
| Changes tracking | Parsley's stateless model, adds complexity |
| Nested records | Too complicated for V1 |
| Computed fields | Belongs in code |
| Validation hooks/callbacks | Keep simple, use post-processing |
| Custom error messages in schema | Handle in code |

## 3. Record API

### 3.1 Creating Records

**Schema as callable constructor:**

```parsley
// Canonical form (named declaration)
@schema User {
    name: string(min: 2, required),
    email: email(required)
}

// Also valid (assignment form)
let UserSchema = @schema {
    name: string(min: 2, required),
    email: email(required)
}

// Dict → Record
let user = User({name: "Alice", email: "alice@example.com"})

// Array → Table (of Records)
let users = User([
    {name: "Alice", email: "alice@example.com"},
    {name: "Bob", email: "bob@example.com"}
])
```

**The `.as(Schema)` method for chaining:**

```parsley
// On dicts — useful in pipelines
let user = fetchFromAPI().parse().as(User)
let row = csvRows[0].as(User)

// On tables — dynamic schema binding
let data = [{x: 1, y: 1}, {x: 2, y: 2}]
let points = table(data).as(Point)
```

**When to use which:**

| Syntax | Best for |
|--------|----------|
| `Schema({...})` | Primary idiom, literal data |
| `Schema([...])` | Primary idiom, literal array |
| `{...}.as(Schema)` | Chaining, pipelines |
| `table(data).as(Schema)` | Dynamic data, runtime binding |

### 3.2 Property Access

Data fields accessed directly. Metadata via methods:

```parsley
// Data access — direct
record.name           // "Alice"
record.email          // "alice@example.com"

// Metadata access — methods
record.errors()       // {name: {code: "MIN_LENGTH", message: "Too short"}}
record.isValid()      // true/false
record.schema()       // The schema
record.error("name")  // "Too short" (message only, or null)
```

**Why this design:**
- Data access is the common case → make it terse
- Metadata access is less frequent → methods are acceptable
- No reserved field names
- Clear visual distinction: `record.name` vs `record.errors()`

### 3.3 Record Methods

| Method | Returns | Description |
|--------|---------|-------------|
| `record.validate()` | Record | Validate and return with errors |
| `record.update({...})` | Record | Merge fields and auto-revalidate |
| `record.errors()` | Dictionary | All field errors: `{field: {code, message}}` |
| `record.error(field)` | String or null | Error message for specific field |
| `record.errorCode(field)` | String or null | Error code for specific field |
| `record.errorList()` | Array | Errors as array: `[{field, code, message}]` |
| `record.isValid()` | Boolean | True if validated AND no errors |
| `record.hasError(field)` | Boolean | True if field has error |
| `record.schema()` | Schema | The schema used |
| `record.data()` | Dictionary | Plain dict of all data |
| `record.keys()` | Array | Field names |
| `record.withError(field, msg)` | Record | Add custom error (no revalidation) |
| `record.withError(field, code, msg)` | Record | Add custom error with code |

### 3.4 Immutability and Auto-Revalidation

Records are immutable. Mutations return new records and **auto-revalidate**:

```parsley
let user = User({name: "Alice", email: "a@b.com"}).validate()
user.isValid()       // true

// Update returns new revalidated record
let updated = user.update({name: "A"})  // too short
updated.isValid()    // false (auto-revalidated)
updated.error("name") // "Must be at least 2 characters"

user.name            // "Alice" (original unchanged)
updated.name         // "A" (new record)

// Adding custom errors (no revalidation)
let withError = user.withError("email", "Already taken")
```

**Why auto-revalidate?** Like type-checking in a typed language — validation happens automatically so developers can't forget. Use `update()` for efficiency when changing multiple fields:

```parsley
// Efficient: validates once
let r = record.update({a: 1, b: 2, c: 3})
```

### 3.5 Dictionary Compatibility

Records work where dictionaries are expected:

```parsley
let user = User({name: "Alice"})

// Spread into dictionary
let merged = {...user, extraField: "value"}

// JSON encoding
json.encode(user)         // Encodes data fields only

// Pass to function expecting dictionary
someFunc(user)            // Works

// Explicit conversion
let dict = user.data()    // Get plain dictionary
```

## 4. Table API (with Schema)

### 4.1 Creating Typed Tables

```parsley
let Product = @schema {
    sku: string(required),
    name: string(required),
    price: money()
}

// Via callable schema
let products = Product([
    {sku: "A001", name: "Widget", price: $9.99},
    {sku: "A002", name: "Gadget", price: $19.99}
])

// Via table literal (compile-time binding)
let products = @table(Product) [
    {sku: "A001", name: "Widget", price: $9.99},
    {sku: "A002", name: "Gadget", price: $19.99}
]

// Via .as() for dynamic data
let products = table(csvData).as(Product)
```

### 4.2 Table Validation

```parsley
let data = loadCSV("products.csv")
let products = table(data).as(Product)
let validated = products.validate()

validated.isValid()      // true if ALL rows valid
validated.errors()       // [{row: 0, field: "sku", code: "REQUIRED", message: "..."}]
validated.validRows()    // Table of valid rows only
validated.invalidRows()  // Table of invalid rows with errors

// Individual rows are Records with the standard error shape:
validated[0].errors()    // {sku: {code: "REQUIRED", message: "..."}}
```

### 4.3 Typed Table Methods

| Method | Returns | Description |
|--------|---------|-------------|
| `table.validate()` | Table | Bulk validate all rows |
| `table.isValid()` | Boolean | True if all rows valid |
| `table.errors()` | Array | `[{row, field, code, message}]` |
| `table.validRows()` | Table | Rows that passed validation |
| `table.invalidRows()` | Table | Rows that failed, with errors |
| `table.schema()` | Schema | The schema used |

### 4.4 Records vs Tables

| Type | Created By | Use Case |
|------|------------|----------|
| **Record** | `Schema({...})` or `{...}.as(Schema)` | Single entity: form, API request, one DB row |
| **Table** | `Schema([...])` or `table(data).as(Schema)` | Multiple entities: CSV, query results, reports |

**For form validation:** Use Records.  
**For bulk import/pipelines:** Use Tables with `.validate()`.

## 5. Schema Definition

### 5.1 Basic Schema

Schemas use the same syntax as the Query DSL. Both bare types (`int`) and called types (`int()`) are valid:

```parsley
@schema User {
    id: int
    name: string(min: 2, max: 100, required)
    email: email(required)
    age: int(min: 0)
}
```

### 5.2 Field Metadata (Pipe Syntax)

Separate validation from display hints using `|`:

```parsley
@schema User {
    name: string(min: 2, required) | {title: "Full Name", placeholder: "Enter name"}
    email: email(required) | {title: "Email Address", placeholder: "you@example.com"}
    age: int(min: 0) | {title: "Age"}
    salary: decimal() | {title: "Salary", format: "currency"}
    createdAt: datetime() | {title: "Created", format: "date", hidden: true}
}
```

**Why pipe syntax:**
- Visual separation: validation on left, display hints on right
- Backward compatible: old syntax (no `|`) still works
- Extensible: metadata dict can hold anything

### 5.3 Metadata is Open

The metadata dictionary accepts **any** key-value pairs:

```parsley
@schema User {
    name: string(required) | {title: "Name", sortable: true, searchWeight: 2}
    avatar: string() | {title: "Avatar", widget: "image-upload", maxSize: "5mb"}
}

// Access any metadata
User.meta("name", "sortable")     // true
User.meta("avatar", "widget")     // "image-upload"
```

### 5.4 Core Metadata (V1)

These are conventions that built-in features look for:

| Metadata | Purpose | Used by |
|----------|---------|---------|
| `title` | Human-readable field name | Forms, tables, error messages |
| `placeholder` | Input placeholder text | Forms |
| `format` | Display format | Tables, read-only views |
| `hidden` | Exclude from default display | Tables, auto-generated forms |
| `help` | Help text below field | Forms |

### 5.5 Accessing Schema Metadata

```parsley
// From schema directly
User.title("name")           // "Full Name"
User.placeholder("email")    // "you@example.com"
User.fields()                // ["name", "email", "age", ...]
User.visibleFields()         // Excludes hidden fields

// From a record
let user = User({name: "Alice"})
user.schema().title("name")  // "Full Name"
```

### 5.6 Format Method

Records can format values using schema hints:

```parsley
let user = @query(Users | id == {id} ?-> *)

user.salary              // 52000 (raw number)
user.format("salary")    // "$52,000.00" (formatted per schema)
user.format("createdAt") // "Jan 15, 2025"
```

**Built-in formats:**

| Format | Example Input | Example Output |
|--------|---------------|----------------|
| `"date"` | `2025-01-15` | "Jan 15, 2025" |
| `"datetime"` | `2025-01-15T14:30:00Z` | "Jan 15, 2025 2:30 PM" |
| `"currency"` | `52000` | "$52,000.00" |
| `"percent"` | `0.15` | "15%" |
| `"number"` | `1234567` | "1,234,567" |

## 6. Validation

### 6.1 Declarative Validation

Constraints are declared in the schema:

```parsley
let User = @schema {
    name: string(min: 2, max: 100, required),
    email: email(required),
    age: int(min: 0, max: 150),
    role: enum("user", "admin", "moderator"),
    website: url()
}
```

### 6.2 Validation Flow

```parsley
// Create record (no validation yet)
let record = User({name: "A", email: "bad"})
record.isValid()     // true (no errors — not validated)

// Validate
let validated = record.validate()
validated.isValid()  // false
validated.errors()   // {name: "Min 2 chars", email: "Invalid email"}

// Or in one step
let form = User(props).validate()
```

### 6.3 Custom Validation

For edge cases, add errors via post-processing:

```parsley
let form = User(props).validate()

// Cross-field validation
if (form.password != props.confirmPassword) {
    form = form.withError("confirmPassword", "Passwords don't match")
}

// Business rule
if (isEmailTaken(form.email)) {
    form = form.withError("email", "Already registered")
}
```

### 6.4 Filtering (Whitelisting)

Schema fields act as the whitelist. Unknown fields are ignored:

```parsley
let User = @schema {
    name: string(),
    email: email()
}

// is_admin is silently ignored — not in schema
let record = User({name: "Alice", email: "a@b.com", is_admin: true})
record.data()  // {name: "Alice", email: "a@b.com"}
```

### 6.5 Type Casting

Schema types drive automatic casting:

```parsley
@schema User {
    age: int(),
    active: bool()
}

let record = User({age: "42", active: "true"})
record.age     // 42 (Integer, not "42" string)
record.active  // true (Boolean, not "true" string)
```

### 6.6 Default Values

Defaults are applied on **record creation**, not validation:

```parsley
@schema Request {
    page: int(default: 1),
    limit: int(default: 20),
    sort: string(default: "created_at")
}

let r = Request({})              // r.page = 1, r.limit = 20, r.sort = "created_at"
let r = Request({page: 5})       // r.page = 5, r.limit = 20, r.sort = "created_at"
let r = Request({page: null})    // r.page = 1 (null treated as missing)
```

**Rationale:** Defaults answer "what value should this have" — a creation concern. Validation answers "is this value acceptable" — a separate concern. Applying defaults at creation means fields always have usable values.

## 7. Form Binding

### 7.1 Form Context

The `@record` attribute establishes form context:

```parsley
<form @record={form} method="POST">
    <input @name="name"/>
    <Error @name="name"/>
    
    <input @name="email" type="email"/>
    <Error @name="email"/>
    
    <button type="submit">"Save"</button>
</form>
```

### 7.2 What Gets Bound

The `@name` attribute binds three things:
1. **Value:** Current data from record
2. **Validation attributes:** `required`, `minlength`, etc. from schema
3. **Errors:** Current validation errors

### 7.3 Input Rewriting

```parsley
// You write:
<input @name="email"/>

// Becomes:
<input name="email" 
       value={form.email} 
       required 
       type="email"
       class={form.hasError("email") ? "error" : ""}/>
```

### 7.4 Error Display

```parsley
// You write:
<Error @name="email"/>

// Becomes (when error exists):
<span class="error">{form.error("email")}</span>

// Or nothing (when no error)
```

### 7.5 Complete Form Example

```parsley
let User = @schema {
    name: string(min: 2, required) | {title: "Full Name"},
    email: email(required) | {title: "Email"}
}

export default = fn(props) {
    let form = props.form ?? User({})
    
    <form @record={form} method="POST" action="save">
        <label>{form.schema().title("name")}</label>
        <input @name="name"/>
        <Error @name="name"/>
        
        <label>{form.schema().title("email")}</label>
        <input @name="email" type="email"/>
        <Error @name="email"/>
        
        <button type="submit">"Save"</button>
    </form>
}

export save = fn(props) {
    let form = User(props).validate()
    
    if (form.isValid()) {
        @insert(Users |< ...form .)
        <div>"Saved!"</div>
    } else {
        default({form: form})
    }
}
```

## 8. Database Integration

### 8.1 Schema-Bound Database Tables

Schemas bind to database tables using `db.bind()` (same as Query DSL):

```parsley
@schema User {
    name: string(required)
    email: email()
}

let db = @sqlite("app.db")
let Users = db.bind(User, "users")
```

**Note:** `db.bind()` is for database tables. `@table(Schema) [...]` creates in-memory data tables.

### 8.2 Queries Return Records

When a table has a schema, queries return Records:

```parsley
// Single row → Record
let user = @query(Users | id == {id} ?-> *)
user.name           // Data access
user.isValid()      // Method available
user.schema()       // Returns User schema

// Multiple rows → Table of Records
let admins = @query(Users | role == "admin" ??-> *)
admins[0].schema()  // User schema
```

### 8.3 Return Type Summary

| Source | Syntax | Returns |
|--------|--------|---------|
| Schema + dict | `User({...})` | Record |
| Schema + array | `User([...])` | Table |
| Dict + schema | `{...}.as(User)` | Record |
| Table + schema | `table(data).as(User)` | Table |
| Table literal | `@table(User) [...]` | Table |
| DB single row | `@query(... ?-> *)` | Record |
| DB single row | `@query(... ?-> a, b)` | Record (if a, b in schema) or Dict |
| DB single row | `@query(... ?!-> a, b)` | Record (error if not subset) |
| DB multiple rows | `@query(... ??-> *)` | Table of Records |
| DB multiple rows | `@query(... ??!-> a, b)` | Table of Records (error if not subset) |

### 8.4 Edit Flow

```parsley
// Load existing user as Record
let user = @query(Users | id == {id} ?-> *)

// Merge new data and validate
let form = User({...user, ...newData}).validate()

// Save if valid
if (form.isValid()) {
    @update(Users | id == {id} |< ...form .)
}
```

## 9. Beyond Forms

### 9.1 API Validation

```parsley
let Request = @schema {
    page: int(min: 1, default: 1),
    limit: int(min: 1, max: 100, default: 20),
    sort: enum("name", "date", "relevance")
}

export GET = fn(req) {
    let params = Request(req.query).validate()
    
    if (!params.isValid()) {
        return {status: 400, body: {errors: params.errors()}}
    }
    
    let results = search(params.page, params.limit, params.sort)
    {status: 200, body: results}
}
```

### 9.2 Configuration Validation

```parsley
let Config = @schema {
    port: int(min: 1, max: 65535, required),
    host: string(default: "localhost"),
    maxConnections: int(min: 1, default: 100)
}

let config = Config(@env).validate()
if (!config.isValid()) {
    log("Invalid config: " + json.encode(config.errors()))
    exit(1)
}
```

### 9.3 Bulk Import

```parsley
let products = table(loadCSV("products.csv")).as(Product).validate()

if (products.isValid()) {
    @insert(Products |< ...products .)
    <div>"Imported " + products.length + " products"</div>
} else {
    let bad = products.invalidRows()
    <div>"Failed: " + bad.length + " rows with errors"</div>
    <table @records={bad}>...</table>
}
```

## 10. Query DSL Compatibility

This section clarifies how Records integrate with the existing Query DSL.

### 10.1 Schema Syntax is Unchanged

Records use the exact same `@schema` syntax as the Query DSL:

```parsley
@schema User {
    id: int
    name: string(min: 2, required)
    email: email(required)
    role: enum("user", "admin")
}
```

### 10.2 Database vs Data Tables

| Type | Purpose | Creation |
|------|---------|----------|
| **Database table** | Persistent storage | `db.bind(Schema, "tablename")` |
| **Data table** | In-memory data | `@table(Schema) [...]` or `table(data).as(Schema)` |

```parsley
// Database binding (Query DSL)
let db = @sqlite("app.db")
let Users = db.bind(User, "users")

// Data table (in-memory)
let products = @table(Product) [
    {sku: "A001", name: "Widget"}
]
```

### 10.3 Query Return Types

| Query | Returns | Notes |
|-------|---------|-------|
| `@query(Users ?-> *)` | Record | Full row with schema |
| `@query(Users ??-> *)` | Table of Records | Multiple rows |
| `@query(Users ?-> name, email)` | Record | Subset of schema fields |
| `@query(Users ?-> name, UPPER(x) as y)` | Dict | `y` not in schema — silent fallback |
| `@query(Users ?!-> name, email)` | Record | Explicit — error if not subset |
| `@query(Users ??!-> *)` | Table of Records | Explicit — error if not subset |
| `@query(... ?-> count)` | Integer | Aggregate |
| `@query(... + by ...)` | Table | Aggregate — no schema |

**Projection behavior:**
- **Default (`?->`, `??->`)**: Return Record if all columns are schema fields, otherwise Dict
- **Explicit (`?!->`, `??!->`)**: Require Record — error if any column isn't in schema

```parsley
// Auto-detect (90% case)
@query(Users ?-> name, email)                    // Record (both in schema)
@query(Users ?-> name, UPPER(email) as upper)    // Dict (silent, 'upper' not in schema)

// Explicit (when you need guarantees)
@query(Users ?!-> name, email)                   // Record (or error)
@query(Users ?!-> name, UPPER(email) as upper)   // ERROR: 'upper' not in User schema
```

**Validation on partial Records:**
- `required` not enforced on missing fields (data came from DB, already validated on insert)
- Constraints (`min`, `max`, `format`) validated on present fields
- Metadata (titles, placeholders) available for present fields

**Records from queries are auto-validated:**

```parsley
let user = @query(Users ?-> name, email)
user.isValid()  // true (data from DB is trusted, validation ran)

// Mutations trigger re-validation:
let updated = user.update({name: "A"})  // too short
updated.isValid()  // false
```

**Rationale:** Data from DB already passed validation on insert — it's trusted. Auto-validation on query means `isValid()` is always meaningful with no "maybe validated" ambiguity. If DB has stale data violating new schema constraints, validation surfaces this (useful for detecting schema/data drift).

### 10.4 Metadata and Query DSL

Schema metadata (pipe syntax) is **preserved but ignored** by the Query DSL:

```parsley
@schema User {
    name: string(required) | {title: "Full Name", placeholder: "Enter name"}
    email: email(required) | {title: "Email"}
}

// Query DSL ignores metadata — just validates constraints
@insert(Users |< name: "Alice" |< email: "a@b.com" .)

// Forms use metadata
<form @record={user}>
    <label>{user.schema().title("name")}</label>  // "Full Name"
    <input @name="name"/>
</form>
```

### 10.5 Relations

Records **do not include relations**. When using `with` for eager loading:

```parsley
@query(Posts | with author ??-> *)
```

The `author` field is a dictionary (or Record if User schema is bound), but it's a separate entity — not validated as part of the Post record.

**Validating foreign keys** (e.g., `author_id` exists) is done in code, not schema:

```parsley
let post = Post(props).validate()

// Check foreign key in code
if (post.isValid()) {
    let authorExists = @query(Users | id == {post.author_id} ?-> exists)
    if (!authorExists) {
        post = post.withError("author_id", "Author not found")
    }
}
```

### 10.6 `required` and NOT NULL

`required` in schema constraints maps to `NOT NULL` in database:

```parsley
@schema User {
    name: string(required)   // NOT NULL in DB, required in validation
    bio: string()            // NULL allowed in DB, optional in validation
}
```

## 11. Investigation Results and Recommendations

This section documents findings from investigating the Go implementation.

### 11.1 Batch Insert Behavior

**Investigation:** How does `@insert` handle arrays and data tables?

**Findings** (from `stdlib_dsl_query.go`):

| Input Type | Supported | Validation | Implementation |
|------------|-----------|------------|----------------|
| Single dict | ✅ Yes | ✅ Validates | `evalDSLInsert` |
| Array | ✅ Yes | ❌ **No validation** | `evalDSLInsertBatch` |
| Table | ❌ No | N/A | TYPE-0002 error |

Key code from `evalDSLInsertBatch`:
```go
arr, ok := collection.(*Array)
if !ok {
    return &Error{Message: fmt.Sprintf("batch insert collection must be an array, got %s", collection.Type())}
}
```

**Problem:** Batch inserts bypass validation entirely. Single inserts call `ValidateSchemaFields`, but batch inserts don't.

**Recommendation:**

1. **Add validation to batch inserts** — Call `ValidateSchemaFields` for each row
2. **Support Table type** — Cast table to array internally, preserving schema info
3. **Validation behavior:**
   - If table is **already validated** (all rows passed), skip re-validation
   - If table is **unvalidated** or has **invalid rows**, validate each row
   - Stop on first error (fail-fast) or collect all errors based on option

```parsley
// Current (array only, no validation)
@insert(Users * each people as person |< name: person.name .)

// Proposed (table with validation)
let products = table(data).as(Product).validate()
@insert(Products * each products as p |< ...p .)  // skips validation, already done

// Or: validate during insert
@insert(Products * each table(data).as(Product) as p |< ...p .)  // validates each row
```

### 11.2 Eager-Loaded Relations Type

**Investigation:** What type is returned for `with author`?

**Findings** (from `stdlib_dsl_query.go`):

```go
// loadBelongsToRelation returns:
return rowToDict(columns, values, env), nil  // Returns *Dictionary

// loadHasManyRelation returns:
return &Array{Elements: results}, nil  // Returns *Array of *Dictionary
```

| Relation Type | Returns | Schema Applied |
|---------------|---------|----------------|
| Belongs-to (`author: User via user_id`) | `*Dictionary` | ❌ No |
| Has-many (`posts: [Post] via author_id`) | `*Array` of `*Dictionary` | ❌ No |

**Key finding:** Schemas are NOT applied to eager-loaded data. The `rowToDict` function creates plain dictionaries without schema binding.

**Additional finding:** N+1 queries exist — each eager load does a separate `SELECT` per row.

**Recommendation:**

Option A: **Don't change Query DSL** — Keep relations as plain dicts
- Simpler implementation
- User can call `.as(Schema)` if they need validation
- Avoids performance overhead on every query

Option B: **Auto-apply schema to relations** — Make relations return Records
- More consistent mental model
- Requires looking up target schema at query time
- May have performance implications

**Recommended:** Option A for now. Relations are typically "read-only" views of related data. If user needs to validate/modify, they can explicitly cast:

```parsley
let post = @query(Posts | id == {id} | with author ?-> *)
let author = post.author.as(User).validate()  // Explicit when needed
```

### 11.3 Error Shape Unification

**Investigation:** What error shapes do the three systems produce?

**Findings:**

#### Query DSL Validation Errors (`stdlib_dsl_schema.go`)

```go
// buildValidationErrorObject creates:
{
    error: "VALIDATION_ERROR",
    message: "Validation failed",
    fields: [
        {field: "email", code: "FORMAT", message: "Invalid email format"},
        {field: "name", code: "MIN_LENGTH", message: "Must be at least 2 characters"}
    ]
}
```

#### @std/schema Validation (`stdlib_schema.go`)

```go
// schemaValidate returns:
{
    valid: false,
    errors: [
        {schema: "User", field: "email", code: "FORMAT", message: "User schema: Invalid email format"},
        {schema: "User", field: "name", code: "REQUIRED", message: "User schema: Field is required"}
    ]
}
```

#### Error Codes Used (both systems)

| Code | Meaning |
|------|---------|
| `REQUIRED` | Missing required field |
| `TYPE` | Wrong data type |
| `FORMAT` | Invalid format (email, URL, phone, slug, date, ULID, UUID) |
| `ENUM` | Value not in allowed set |
| `MIN_LENGTH` | String too short |
| `MAX_LENGTH` | String too long |
| `MIN_VALUE` | Number too small |
| `MAX_VALUE` | Number too large |

**Key insight:** Both systems already use the same error codes! The difference is structure, not semantics.

**Recommendation: Unified Error Shape**

Design a shape that works for:
- HTTP APIs (need field name + error code for client-side handling)
- Form binding (need field-keyed access for display)
- Simple code (can ignore codes if not needed)

```parsley
// Record errors() method returns:
{
    email: {code: "FORMAT", message: "Invalid email format"},
    name: {code: "MIN_LENGTH", message: "Must be at least 2 characters"}
}

// Access patterns:
record.errors()                    // Full dict
record.errors().email              // {code: "FORMAT", message: "..."}
record.errors().email.message      // "Invalid email format"
record.errors().email.code         // "FORMAT"
record.error("email")              // "Invalid email format" (convenience - message only)
record.errorCode("email")          // "FORMAT" (convenience - code only)
```

**Why field-keyed (not array)?**
- Faster lookup: `errors.email` vs iterating array
- Natural for form binding: `<span @if={errors.email}>{errors.email.message}</span>`
- Easy to check: `if (errors.email) { ... }`
- Spread works: `{...errors, email: null}` to clear one error

**HTTP API serialization:**
```parsley
// For REST/GraphQL response, convert to array if needed:
let fieldErrors = record.errors().entries().map(fn(e) {
    {field: e.key, code: e.value.code, message: e.value.message}
})

// Or provide convenience method:
record.errorList()  // [{field: "email", code: "FORMAT", message: "..."}]
```

**Migration path:**
1. **Phase 1 (Records):** Records use new shape: `{field: {code, message}}`
2. **Phase 1 (Records):** Add `record.errorList()` for array form
3. **Future release:** Query DSL adopts same shape (breaking change, semver major bump)
4. **No change:** @std/schema remains as-is (different purpose, explicit validation)

**Note:** Query DSL error shape migration is committed but deferred. Don't block Records on Query DSL changes.

### 11.4 Summary of Recommendations

| Area | Recommendation | Priority |
|------|----------------|----------|
| Batch insert validation | Add validation, support Table type | High |
| Eager-loaded relations | Keep as Dictionary, don't auto-apply schema | Medium |
| Error shape | Use `{field: {code, message}}` with convenience methods | High |
| N+1 queries | Out of scope for Record design (separate issue) | Low |

## 12. Implementation Phases

### Phase 1: Core Record Type
- Record struct with schema, data, errors, validated flag
- `Schema({...})` → Record (unvalidated, with defaults applied)
- `record.validate()` → Record (validated)
- `record.update({...})` → Record (auto-revalidated)
- `record.isValid()`, `record.errors()`, `record.error(field)`, `record.errorCode(field)`, `record.errorList()`
- Error shape: `{field: {code, message}}`
- Basic validation (required, min/max, types, enum, format)

### Phase 2: Schema Metadata
- Pipe syntax parsing
- Metadata access methods (`schema.title()`, `schema.meta()`)
- `record.format()` for display formatting

### Phase 3: Table Integration
- `Schema([...])` → Table
- `.as(Schema)` for dicts and tables
- `table.validate()` with `validRows()`/`invalidRows()`
- Table error shape: `[{row, field, code, message}]`

### Phase 4: Form Binding
- `<form @record={...}>` context
- `<input @name="..."/>` rewriting
- `<Error @name="..."/>` component

### Phase 5: Database Integration
- Queries return Records when table has schema
- Auto-validation on query return
- Projection auto-detect (Record if columns ⊆ schema)
- `?!->` / `??!->` explicit Record terminals
- Batch insert validation, Table type support

---

## Appendix: Quick Reference

### Creating Records and Tables

```parsley
// Records (both forms valid, named is canonical)
@schema User { ... }           // Named declaration (canonical)
let UserSchema = @schema {...} // Assignment form (also valid)

let r = User({...})
let r = {...}.as(User)

// Tables
let t = User([...])
let t = @table(User) [...]
let t = table(data).as(User)
```

### Validation

```parsley
// Record validation
let validated = record.validate()
validated.isValid()
validated.errors()           // {field: {code, message}, ...}
validated.error("field")     // "message" or null
validated.errorCode("field") // "CODE" or null
validated.errorList()        // [{field, code, message}, ...]

// Table validation
let validated = table.validate()
validated.isValid()          // true if ALL rows valid
validated.errors()           // [{row, field, code, message}, ...]
validated.validRows()        // Table of valid rows
validated.invalidRows()      // Table of invalid rows

// Query results are auto-validated
let user = @query(Users ?-> *)
user.isValid()               // true (data from DB is trusted)
```

### Default Values

```parsley
@schema Request {
    page: int(default: 1),
    limit: int(default: 20)
}

let r = Request({})           // page=1, limit=20 (defaults applied on creation)
let r = Request({page: 5})    // page=5, limit=20
```

### Error Codes

| Code | Meaning |
|------|---------|
| `REQUIRED` | Missing required field |
| `TYPE` | Wrong data type |
| `FORMAT` | Invalid format (email, URL, phone, slug, etc.) |
| `ENUM` | Value not in allowed set |
| `MIN_LENGTH` / `MAX_LENGTH` | String length constraint |
| `MIN_VALUE` / `MAX_VALUE` | Number range constraint |

### Schema Definition

```parsley
@schema User {
    name: string(min: 2, required) | {title: "Full Name"}
    email: email(required) | {title: "Email"}
}
```

### Form Pattern

```parsley
export default = fn(props) {
    let form = props.form ?? User({})
    <form @record={form} method="POST" action="save">
        <input @name="name"/>
        <Error @name="name"/>
        <button>"Save"</button>
    </form>
}

export save = fn(props) {
    let form = User(props).validate()
    if (form.isValid()) {
        @insert(Users |< ...form .)
        <div>"Success"</div>
    } else {
        default({form: form})
    }
}
```
