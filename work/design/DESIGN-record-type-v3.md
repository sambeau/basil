# Design: Record Type for Parsley

**Status:** Final Design  
**Date:** 2025-01-15  
**Supersedes:** DESIGN-record-type-v2.md, DESIGN-record-type.md  
**Related:** BACKLOG #21, FEAT-002

---

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
validated.errors()  // {} or {name: {code: "MIN_LENGTH", message: "..."}}
```

**Key insight:** A Record is Schema + Data + Errors. Nothing more.

---

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

### 2.6 Error Shape

| Context | Shape | Access |
|---------|-------|--------|
| Record | `{field: {code, message}}` | `record.errors()`, `record.error(field)`, `record.errorCode(field)` |
| Table | `[{row, field, code, message}]` | `table.errors()` |
| Convenience | `[{field, code, message}]` | `record.errorList()` |

### 2.7 Form Binding

| Decision | Choice |
|----------|--------|
| Form context | `<form @record={record}>` establishes context via AST |
| Field binding | `@field` attribute binds to schema field (not `@name`) |
| What gets bound | Values + validation attributes + errors + ARIA |
| Input rewriting | `<input @field="x"/>` → full input with value, attrs, ARIA |
| Accessibility | `aria-invalid`, `aria-describedby`, `aria-required` |
| Label display | `<Label @field="x"/>` → label with title from schema |
| Error display | `<Error @field="x"/>` → conditional error element with `role="alert"` |
| Metadata display | `<Meta @field="x" @key="help"/>` → any metadata value |
| Select display | `<Select @field="x"/>` → select with options from enum |
| Checkbox/radio | `type="checkbox"` binds `checked`, radio checks against `value` |
| Tag override | `@tag` prop to change element type on Label, Error, Meta |

### 2.8 Not Included in V1

| Feature | Reason |
|---------|--------|
| Changes tracking | Parsley's stateless model, adds complexity |
| Nested records | Too complicated for V1 |
| Computed fields | Belongs in code |
| Validation hooks/callbacks | Keep simple, use post-processing |
| Custom error messages in schema | Handle in code |

---

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
| `record.title(field)` | String | Shorthand for `record.schema().title(field)` |
| `record.placeholder(field)` | String | Shorthand for `record.meta(field, "placeholder")` |
| `record.meta(field, key)` | Any | Shorthand for `record.schema().meta(field, key)` |
| `record.enumValues(field)` | Array | Enum options for field (empty if not enum) |

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

---

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

---

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

---

## 6. Validation

### 6.1 Declarative Validation

Constraints are declared in the schema:

```parsley
let User = @schema {
    id: id(auto),                              // DB-generated, skipped in validation
    name: string(min: 2, max: 100, required),
    email: email(required),
    age: int(min: 0, max: 150),
    role: enum["user", "admin", "moderator"],
    website: url(),
    createdAt: datetime(auto)                  // Server-generated timestamp
}
```

### 6.1.1 The `auto` Constraint

The `auto` constraint marks fields whose values are generated by the database or server (e.g., auto-increment IDs, timestamps). This solves the "Catch-22" where a schema needs an `id` for completeness, but insert operations cannot provide one.

**Behavior:**
- Auto fields are **skipped during validation** — missing or null is not an error
- Auto fields are **immutable on update** — `record.update({id: x})` produces an error
- Auto fields **default to not-required** — they don't need `required: false`
- `auto` and `required` **cannot be combined** — they are contradictory
- `auto` and `default` **may be combined** — default applies before DB generation

```parsley
let user = User({name: "Alice", email: "a@b.com"})  // id and createdAt omitted
user.validate().isValid()   // true — auto fields skipped
user.update({id: "x"})      // Error: cannot update auto field 'id'
```
```

### 6.2 Validation Flow

```parsley
// Create record (no validation yet)
let record = User({name: "A", email: "bad"})
record.isValid()     // false (not validated)

// Validate
let validated = record.validate()
validated.isValid()  // false
validated.errors()   // {name: {code: "MIN_LENGTH", ...}, email: {code: "FORMAT", ...}}

// Or in one step
let form = User(props).validate()
```

### 6.3 Custom Validation

For edge cases, add errors via post-processing:

```parsley
let form = User(props).validate()

// Cross-field validation
if (form.password != props.confirmPassword) {
    form = form.withError("confirmPassword", "MISMATCH", "Passwords don't match")
}

// Business rule
if (isEmailTaken(form.email)) {
    form = form.withError("email", "DUPLICATE", "Already registered")
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

### 6.7 Error Codes

Standard error codes used by declarative validation:

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

---

## 7. Form Binding

### 7.1 Form Context

The `@record` attribute establishes form context:

```parsley
<form @record={form} method="POST">
    <Label @field="name"/>
    <input @field="name"/>
    <Error @field="name"/>
    
    <Label @field="email"/>
    <input @field="email" type="email"/>
    <Error @field="email"/>
    
    <button type="submit">"Save"</button>
</form>
```

### 7.2 What Gets Bound

The `@field` attribute binds to a schema field. It provides:
1. **Value:** Current data from record
2. **Validation attributes:** `required`, `minlength`, etc. from schema
3. **Errors:** Current validation errors
4. **Accessibility:** ARIA attributes for screen readers

**Why `@field` not `@name`?** Clearer that it binds to schema, avoids confusion with HTML `name` attribute.

### 7.3 Input Rewriting

```parsley
// You write:
<input @field="email"/>

// Becomes:
<input name="email" 
       value={form.email} 
       required 
       type="email"
       aria-invalid={form.hasError("email")}
       aria-describedby={form.hasError("email") ? "email-error" : null}/>
```

**Accessibility attributes:**

| Attribute | Value | Purpose |
|-----------|-------|---------|
| `aria-invalid` | `true`/`false` | Indicates validation state; commonly used for styling |
| `aria-describedby` | `"{field}-error"` | Links input to error message for screen readers |
| `aria-required` | `true` | Set when field is required (mirrors HTML `required`) |

**Note:** `aria-invalid="false"` is explicitly set (not omitted) so CSS selectors like `[aria-invalid="false"]` and `[aria-invalid="true"]` can be used for styling valid/invalid states.

**Checkbox binding** (boolean fields):

```parsley
// You write:
<input @field="active" type="checkbox"/>

// Becomes:
<input name="active" type="checkbox" checked={form.active} aria-invalid=.../>
```

**Radio button binding** (enum fields with value):

```parsley
// You write:
<input @field="color" type="radio" value="red"/>

// Becomes:
<input name="color" type="radio" value="red" checked={form.color == "red"} aria-invalid=.../>
```

### 7.4 Label Display

**Self-closing form** (generates `for` attribute):

```parsley
// You write:
<Label @field="email"/>

// Becomes:
<label for="email">{form.title("email")}</label>
```

**Tag-pair form** (wraps children, no `for` needed):

```parsley
// You write:
<Label @field="name">
    <input @field="name"/>
</Label>

// Becomes:
<label>
    Full Name
    <input name="name" value={form.name} .../>
</label>
```

The tag-pair form is useful when you want the input inside the label — a common accessibility pattern that makes the entire label clickable.

**Tag prop:** Use `@tag` to change the element type:

```parsley
<Label @field="email" @tag="span"/>   // <span>Email Address</span>
<Label @field="email"/>               // <label for="email">...</label> (default)
```

### 7.5 Error Display

```parsley
// You write:
<Error @field="email"/>

// Becomes (when error exists):
<span id="email-error" class="error" role="alert">form.error("email")</span>

// Or nothing (when no error)
```

**Tag prop:** Use `@tag` to change the element type:

```parsley
<Error @field="email" @tag="small"/>   // <small id="email-error" ...>
<Error @field="email" @tag="div"/>     // <div id="email-error" ...>
<Error @field="email"/>                // <span id="email-error" ...> (default)
```

**Accessibility attributes on Error:**

| Attribute | Value | Purpose |
|-----------|-------|---------|
| `id` | `"{field}-error"` | Target for `aria-describedby` on input |
| `role` | `"alert"` | Announces error to screen readers when it appears |

### 7.6 Metadata Display

For any schema metadata beyond title:

```parsley
// You write:
<Meta @field="email" @key="help"/>

// Becomes:
<span>form.meta("email", "help")</span>

// Or nothing (when metadata doesn't exist)
```

**Tag prop:** Use `@tag` to change the element type:

```parsley
<Meta @field="email" @key="help" @tag="small"/>  // <small>Help text here</small>
<Meta @field="email" @key="help" @tag="p"/>      // <p>Help text here</p>
```

### 7.7 Select Component

For enum fields, `<Select>` auto-generates options:

```parsley
// You write:
<Select @field="fruit"/>

// Becomes:
<select name="fruit" aria-invalid=... aria-describedby=...>
    <option value="">{form.placeholder("fruit")}</option>
    <option value="apple" selected={form.fruit == "apple"}>"apple"</option>
    <option value="banana" selected={form.fruit == "banana"}>"banana"</option>
    ...
</select>
```

**Custom placeholder:**

```parsley
<Select @field="fruit" placeholder="Choose a fruit..."/>
```

**Manual approach** for complex cases (optgroups, custom labels, disabled options):

```parsley
<select @field="fruit">
    <option value="">form.placeholder("fruit")</option>
    for(val in form.enumValues("fruit")) {
        <option value={val} selected={form.fruit == val}>val</option>
    }
</select>
```

### 7.8 Complete Form Example

```parsley
let User = @schema {
    name: string(min: 2, required) | {title: "Full Name", help: "Your legal name"},
    email: email(required) | {title: "Email", help: "We'll never share this"}
}

export default = fn(props) {
    let form = props.form ?? User({})
    
    <form @record={form} method="POST" action="save">
        <Label @field="name"/>
        <input @field="name"/>
        <Error @field="name"/>
        <Meta @field="name" @key="help" @tag="small"/>
        
        <Label @field="email"/>
        <input @field="email" type="email"/>
        <Error @field="email"/>
        <Meta @field="email" @key="help" @tag="small"/>
        
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

**Inline access:** For cases where you need metadata inline, use shorthand methods:

```parsley
// Shorthand (preferred)
form.title("name")          // "Full Name"
form.meta("name", "help")   // "Your legal name"

// Long form (also works)
form.schema().title("name")
form.schema().meta("name", "help")
```

---

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
user.isValid()      // true (auto-validated)
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

### 8.4 Query Return Type Behavior

**Auto-detect mode (`?->`, `??->`):**
- Returns Record if all projected columns are schema fields
- Falls back to Dict silently if any column isn't in schema

```parsley
@query(Users ?-> name, email)                    // Record (both in schema)
@query(Users ?-> name, UPPER(email) as upper)    // Dict ('upper' not in schema)
```

**Explicit mode (`?!->`, `??!->`):**
- Requires Record — error if any column isn't in schema
- Use when you need guarantees about the return type

```parsley
@query(Users ?!-> name, email)                   // Record (or error)
@query(Users ?!-> name, UPPER(email) as upper)   // ERROR: 'upper' not in User schema
```

### 8.5 Auto-Validation on Query

Records from queries are **auto-validated**:

```parsley
let user = @query(Users ?-> name, email)
user.isValid()  // true (data from DB is trusted)

// Mutations trigger re-validation:
let updated = user.update({name: "A"})  // too short
updated.isValid()  // false
```

**Rationale:** Data from DB already passed validation on insert — it's trusted. Auto-validation means `isValid()` is always meaningful with no "maybe validated" ambiguity.

**Validation on partial Records:**
- `required` not enforced on missing fields (data came from DB)
- Constraints (`min`, `max`, `format`) validated on present fields
- Metadata (titles, placeholders) available for present fields

### 8.6 Edit Flow

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

### 8.7 Relations

Records **do not include relations**. When using `with` for eager loading:

```parsley
@query(Posts | with author ??-> *)
```

The `author` field is a plain Dictionary — not validated as part of the Post record. If validation is needed, cast explicitly:

```parsley
let post = @query(Posts | id == {id} | with author ?-> *)
let author = post.author.as(User).validate()  // Explicit when needed
```

**Validating foreign keys** is done in code:

```parsley
let post = Post(props).validate()

if (post.isValid()) {
    let authorExists = @query(Users | id == {post.author_id} ?-> exists)
    if (!authorExists) {
        post = post.withError("author_id", "NOT_FOUND", "Author not found")
    }
}
```

### 8.8 Batch Insert Validation

Batch inserts validate all rows:

```parsley
// Pre-validated table — skips re-validation
let products = table(data).as(Product).validate()
@insert(Products |< ...products .)

// Unvalidated — validates each row during insert
@insert(Products |< ...table(data).as(Product) .)
```

### 8.9 `required` and NOT NULL

`required` in schema constraints maps to `NOT NULL` in database:

```parsley
@schema User {
    name: string(required)   // NOT NULL in DB, required in validation
    bio: string()            // NULL allowed in DB, optional in validation
}
```

---

## 9. Beyond Forms

### 9.1 API Validation

```parsley
let Request = @schema {
    page: int(min: 1, default: 1),
    limit: int(min: 1, max: 100, default: 20),
    sort: enum["name", "date", "relevance"]
}

export GET = fn(req) {
    let params = Request(req.query).validate()
    
    if (!params.isValid()) {
        return {status: 400, body: {errors: params.errorList()}}
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

---

## 10. Query DSL Compatibility

### 10.1 Schema Syntax is Unchanged

Records use the exact same `@schema` syntax as the Query DSL:

```parsley
@schema User {
    id: int
    name: string(min: 2, required)
    email: email(required)
    role: enum["user", "admin"]
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

### 10.3 Metadata and Query DSL

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
    <Label @field="name"/>       // "Full Name"
    <input @field="name"/>
</form>
```

---

## 11. Schema Checking (Runtime)

### 11.1 The Problem

When functions receive records as arguments, there's no compile-time guarantee they have the expected schema. A function expecting a `User` record might receive a `Product` record, leading to subtle bugs or crashes.

```parsley
fn saveUser(record) {
    @insert(Users |< ...record .)  // What if record isn't a User?
}

saveUser(Product({sku: "A001"}))   // Oops
```

### 11.2 The `is` Operator

The `is` operator provides runtime schema checking:

```parsley
record is User       // true if record's schema is User
table is Product     // true if table's schema is Product
record is not User   // negation
```

**Semantics:**

| Expression | Returns |
|------------|---------|
| `record is Schema` | `true` if `record.schema() == Schema` |
| `table is Schema` | `true` if `table.schema() == Schema` |
| `value is Schema` | `false` for non-Record/non-Table values |
| `null is Schema` | `false` |
| `{...} is Schema` | `false` (plain dict, no schema) |

### 11.3 Guard Pattern with `check`

Use `is` with Parsley's `check` statement for clean precondition guards:

```parsley
fn saveUser(record) {
    check record is User else {
        return {error: "Expected User record, got " + record.schema().name}
    }
    @insert(Users |< ...record .)
}
```

Multiple guards:

```parsley
fn processOrder(order, user) {
    check order is Order else error("Expected Order record")
    check user is User else error("Expected User record")
    check order.items.length() > 0 else error("Empty order")
    
    // Happy path...
    submitOrder(order, user)
}
```

### 11.4 Conditional Branching

Use `is` for type-based branching:

```parsley
fn process(record) {
    if (record is User) {
        processUser(record)
    } else if (record is Product) {
        processProduct(record)
    } else {
        error("Unknown record type: " + record.schema().name)
    }
}
```

### 11.5 Filtering Collections

Filter arrays of mixed records:

```parsley
let items = [User({...}), Product({...}), User({...})]

let users = items.filter(fn(x) { x is User })
let products = items.filter(fn(x) { x is Product })
```

Use with `for` loops:

```parsley
for (item in items) {
    if (item is not User) skip
    processUser(item)
}
```

### 11.6 Schema Identity vs Name

The `is` operator compares schema **identity**, not name strings:

```parsley
@schema User { name: string }
@schema UserCopy { name: string }  // Same fields, different schema

let u = User({name: "Alice"})
u is User       // true
u is UserCopy   // false (different schema identity)
```

For schema name access (e.g., error messages):

```parsley
record.schema().name    // "User" (string)
```

### 11.7 Edge Cases

```parsley
// Non-record values
null is User                      // false
"hello" is User                   // false
42 is User                        // false
{name: "Alice"} is User           // false (plain dict)

// Untyped collections
table([...]) is User              // false (no schema bound)
[] is User                        // false

// After .as() binding
{name: "Alice"}.as(User) is User  // true
table(data).as(User) is User      // true
```

### 11.8 Why Not Static Checking?

Static schema checking (type annotations on function parameters) was considered but rejected for V1:

```parsley
// Hypothetical — NOT implemented
fn process(record: User) { ... }
```

**Reasons:**

1. **Requires type inference**: Partial static checking without full inference creates false confidence
2. **Dynamic nature of Parsley**: Records can be constructed conditionally, returned from functions, stored in arrays
3. **Annotation without enforcement is confusing**: If `fn(r: User)` doesn't catch mistakes at compile time, why have it?
4. **Runtime `is` is sufficient**: Covers all practical use cases cleanly

The door remains open for future opt-in static checking via a linter or `--strict` mode.

### 11.9 Summary

| Need | Solution |
|------|----------|
| Guard function input | `check record is User else {...}` |
| Branch on schema | `if (record is User) {...}` |
| Filter collections | `items.filter(fn(x) { x is User })` |
| Skip in loops | `if (item is not User) skip` |
| Get schema name | `record.schema().name` |

---

## 12. Implementation Phases

### Phase 1: Core Record Type
- Record struct with schema, data, errors, validated flag
- `Schema({...})` → Record (unvalidated, with defaults applied)
- `record.validate()` → Record (validated)
- `record.update({...})` → Record (auto-revalidated)
- `record.isValid()`, `record.errors()`, `record.error(field)`, `record.errorCode(field)`, `record.errorList()`
- `record.withError(field, msg)`, `record.withError(field, code, msg)`
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
- `<input @field="..."/>` rewriting with ARIA
- Checkbox binding (`checked` for boolean fields)
- Radio binding (`checked` based on value match for enum fields)
- `<Label @field="..."/>` component (self-closing and tag-pair)
- `<Error @field="..."/>` component
- `<Meta @field="..." @key="..."/>` component
- `<Select @field="..."/>` component for enum fields
- `record.title()`, `record.placeholder()`, `record.meta()`, `record.enumValues()` shorthand methods

### Phase 5: Database Integration
- Queries return Records when table has schema
- Auto-validation on query return
- Projection auto-detect (Record if columns ⊆ schema)
- `?!->` / `??!->` explicit Record terminals
- Batch insert validation, Table type support

### Phase 6: Schema Checking
- `is` / `is not` operators for Records and Tables
- `record is Schema` → boolean
- `record is not Schema` → boolean
- `table is Schema` → boolean
- Integration with `check` guards

---

## Appendix A: Quick Reference

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
        <Label @field="name"/>
        <input @field="name"/>
        <Error @field="name"/>
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

### Query Return Types

| Query | Returns |
|-------|---------|
| `@query(Users ?-> *)` | Record |
| `@query(Users ??-> *)` | Table of Records |
| `@query(Users ?-> a, b)` | Record if a,b in schema, else Dict |
| `@query(Users ?!-> a, b)` | Record (error if not in schema) |

### Schema Checking

```parsley
// The is operator
record is User              // true if schema matches
record is not User          // negation
table is Product            // works on tables too

// Guard pattern
check record is User else error("Expected User")

// Branching
if (record is User) { ... }

// Filtering
items.filter(fn(x) { x is User })

// Schema name for errors
record.schema().name        // "User"
```
