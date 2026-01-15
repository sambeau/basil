# Design Investigation: Record Type for Parsley

> **⚠️ DEPRECATED:** This document has been superseded by [DESIGN-record-type-v2.md](DESIGN-record-type-v2.md).
> This file is retained for historical reference only.

**Status:** Deprecated  
**Date:** 2025-01-15  
**Superseded By:** DESIGN-record-type-v2.md  
**Related:** DESIGN-schema-form-validation.md, BACKLOG #21, FEAT-002

---

## 1. Overview

This document explores adding a **Record type** to Parsley — a typed wrapper around data that carries its schema, validation state, and optionally tracks changes. The primary motivation is form handling, but records may have broader applications.

### 1.1 Design Goals

Following Parsley's aesthetic:
- **Simplicity:** Easy to understand, minimal concepts
- **Minimalism:** No more features than necessary
- **Completeness:** Handles real-world use cases fully
- **Composability:** Works well with existing features

### 1.2 The Core Idea

```parsley
// Define a schema
let User = @schema {
    name: string(min: 2, required),
    email: email(required)
}

// Create a record by calling the schema like a function
let user = User({name: "Alice", email: "alice@example.com"})

user.name           // "Alice" (data access)
user.isValid()      // true (not yet validated, no errors)
user.errors()       // {} (no errors)

// Validate the record
let validated = user.validate()
validated.isValid() // true or false
validated.errors()  // {} or {name: "Must be at least 2 characters"}

// Or create and validate in one step
let form = User(formData).validate()
```

### 1.3 Key Insight: Schema as Constructor

The schema acts as a record constructor — calling it like a function creates a record:

```parsley
let User = @schema { name: string(required), email: email() }

// Schema is callable — returns a Record
let record = User({name: "Alice"})

// Validation is a method on the record
let validated = record.validate()
```

This feels natural in Parsley:
- Schemas define structure → calling them constructs instances
- Validation is an operation on a record → it's a method
- Immutable: `validate()` returns a new record with errors populated

## 2. Prior Art Deep Dive

### 2.1 Phoenix Changesets

Phoenix changesets are the closest analog. A changeset contains:

```elixir
%Ecto.Changeset{
  data: %User{name: "Alice", ...},     # Original data
  changes: %{name: "Bob"},              # Pending changes
  errors: [name: {"too short", ...}],   # Validation errors
  valid?: false,                        # Overall validity
  action: :update                       # What operation triggered this
}
```

**Key behaviors:**
- `cast(user, params, [:name, :email])` — whitelist and apply changes
- `validate_required(changeset, [:name])` — add validation
- `changeset.changes` — only the modified fields
- `apply_changes(changeset)` — get data with changes applied
- `get_field(changeset, :name)` — gets changed value or falls back to original

**What "changes" enables:**
1. **Dirty tracking:** Know what actually changed
2. **Partial updates:** Only write changed fields to DB
3. **Audit logging:** Record what changed and when
4. **Optimistic locking:** Detect concurrent modifications
5. **Undo/rollback:** Discard changes, keep original
6. **Form diffing:** Show "modified" indicator on fields

### 2.2 Redux/Immer (JavaScript)

Redux uses immutable state + actions:

```javascript
// State is immutable, actions describe changes
const newState = reducer(oldState, {type: 'UPDATE_NAME', payload: 'Bob'})

// Immer makes this ergonomic
produce(state, draft => {
  draft.name = 'Bob'  // Looks mutable, produces new immutable value
})
```

**Key insight:** Separating "what changed" from "current state" enables time-travel debugging, undo/redo, and optimistic UI updates.

### 2.3 Django Forms

```python
form = UserForm(data=request.POST, instance=user)
form.has_changed()           # True if any field differs from instance
form.changed_data            # ['name', 'email'] - list of changed fields
form.cleaned_data            # Validated/typed data
form.errors                  # {'name': ['Too short']}
```

### 2.4 Comparison Table

| Feature | Phoenix | Django | Redux | Proposed |
|---------|---------|--------|-------|----------|
| Schema reference | ✓ | ✓ (form class) | ✗ | ✓ |
| Data | ✓ | ✓ (instance) | ✓ | ✓ |
| Changes | ✓ | ✓ | ✓ (via diff) | ? |
| Errors | ✓ | ✓ | ✗ | ✓ |
| Immutable | Mostly | ✗ | ✓ | ? |
| Validation | Pipeline | Declarative | External | Declarative |

## 3. Should Parsley Records Track Changes?

### 3.1 Arguments For

**Dirty tracking for UI:**

```parsley
<input @name="name" class={record.isDirty("name") ? "modified" : ""}/>
// Show visual indicator when field has unsaved changes
```

**Efficient database updates:**

```parsley
Users.update(record)  // Only writes changed fields
// vs
Users.update(record.data)  // Writes all fields
```

**Undo/cancel:**

```parsley
export cancel = fn(props) {
    let form = props.form
    let original = form.revert()  // Discard changes, return to original
    default({form: original})
}
```

**Audit/history:**

```parsley
// Log what changed
log(record.changes)  // {name: "Bob"} - only the diff
```

### 3.2 Arguments Against

**Complexity:** Another concept to understand

**Server-side context:** In HTTP request/response, there's no persistent state between requests. Each form submission is stateless — we compare against DB, not in-memory original.

**Parsley's model:** Parts are stateless views. State flows through props, not held in components. "Changes" implies holding state.

### 3.3 Recommendation

**Start without changes tracking.** Record = schema + data + errors.

Reasons:
1. Parsley's stateless model doesn't naturally hold "original" values
2. Dirty tracking can be done at the template level if needed
3. Database can compute diffs itself (compare record.data with existing row)
4. Keeps the Record type simpler

**Revisit if needed:** If real use cases emerge that require changes tracking (e.g., complex multi-step wizards with undo), add it later as `record.changes` and `record.original`.

## 4. The Four Pillars: Filtering, Casting, Validation, Constraints

Ecto changesets provide four core capabilities when handling external data. Let's examine what each means for Parsley:

### 4.1 Filtering (Whitelisting)

**The problem:** External data may contain fields you don't want to accept. A user submitting `{name: "Alice", is_admin: true}` shouldn't be able to promote themselves.

**Ecto's approach:**
```elixir
cast(user, params, [:name, :email])  # Only accept name and email
```

**Parsley's approach:**

Schemas already define what fields exist. We could add explicit `permit`:

```parsley
let record = User(params).permit(["name", "email"])
// or
let record = User(params).except(["is_admin", "role"])
```

**Minimal version:** Schema fields themselves act as the whitelist. Unknown fields are ignored.

```parsley
let User = @schema {
    name: string(),
    email: email()
}

let record = User(params)  // params.is_admin is silently ignored - not in schema
```

**Recommendation:** Start with implicit filtering (schema fields = whitelist). Add explicit `permit` option later if needed for partial updates.

### 4.2 Type Casting

**The problem:** HTML forms send everything as strings. `"42"` needs to become `42`, `"true"` needs to become `true`.

**Ecto's approach:**

```elixir
# Schema defines types, cast() converts strings to types
field :age, :integer
cast(user, %{"age" => "42"}, [:age])  # "42" → 42
```

**What Parsley already does:**

Parts already do this! The server auto-coerces form parameters:

```go
// Type coercion happens automatically
// "42" → Integer, "3.14" → Float, "true" → Boolean
```

**What records should do:**

Schema types should drive casting:

```parsley
let User = @schema {
    name: string(),
    age: int(),
    active: bool()
}

let record = User({name: "Alice", age: "42", active: "true"})
record.age     // 42 (Integer, not "42" string)
record.active  // true (Boolean, not "true" string)
```

**Recommendation:** This is mostly already solved. Ensure schema construction performs type casting based on field types.

### 4.3 Validation

**The problem:** Data may be syntactically correct but semantically invalid. Email without `@`, age of `-5`, etc.

**Ecto's approach:**

```elixir
user
|> cast(params, [:email, :age])
|> validate_required([:email])
|> validate_format(:email, ~r/@/)
|> validate_number(:age, greater_than: 0)
```

**What Parsley already has:**

```parsley
let User = @schema {
    name: string(min: 2, max: 100, required),
    email: email(required),
    age: int(min: 0)
}
```

Constraints are declarative in the schema. `record.validate()` checks all of them.

**Gap:** Ecto allows chaining validations, adding custom validators. Parsley schemas are declarative-only.

**Possible enhancement:**

```parsley
// Custom validation via post-processing
let record = User(data).validate()
if (record.password != props.confirmPassword) {
    record = record.withError("confirmPassword", "doesn't match password")
}
```

**Recommendation:** Declarative validation in schema covers 90% of cases. Use post-processing with `withError()` for edge cases.

### 4.4 Constraints (Database-Level)

**The problem:** Some validations can only happen at the database — uniqueness (is this email taken?), foreign keys (does this post exist?), check constraints.

**Ecto's approach:**

```elixir
user
|> cast(params, [:email])
|> unique_constraint(:email)  # Checked at DB insert time
```

If DB raises a constraint error, Ecto converts it to a changeset error.

**The challenge for Parsley:**

This requires database-level integration:
1. Attempt insert/update
2. Catch constraint violation
3. Convert to record error
4. Return record to form

**What this would look like:**

```parsley
export save = fn(props) {
    let record = User(props).validate()
    
    if (record.isValid()) {
        // Try insert — may fail on constraint
        let result = @insert(Users |< ...record ?-> *)
        
        if (result.isError()) {
            // Constraint error (e.g., duplicate email)
            let withError = record.withError("email", result.message)
            default({form: withError})
        } else {
            <div>"Saved!"</div>
        }
    } else {
        // Validation error
        default({form: record})
    }
}
```

**Or with a helper that wraps the pattern:**

```parsley
export save = fn(props) {
    let record = User(props).validate()
    let result = insertOrError(Users, record)
    
    if (result.isValid()) {
        <div>"Saved!"</div>
    } else {
        default({form: result})
    }
}
```

**Implementation:**

```go
func (t *Table) Insert(record *Record) *Record {
    if !record.IsValid() {
        return record  // Don't try to insert invalid record
    }
    
    err := t.db.Insert(record.Data())
    if err != nil {
        // Check if it's a constraint violation
        if constraintErr := parseConstraintError(err); constraintErr != nil {
            // Add error to record and return
            return record.WithError(constraintErr.Field, constraintErr.Message)
        }
        // Other error - return record with generic error
        return record.WithError("_base", err.Error())
    }
    return record  // Success
}
```

**Recommendation:** This is valuable but non-trivial. Add in a later phase:
- Phase 1: Validation only (client-side equivalent)
- Phase 2: Constraint integration with `table.insert(record)` returning record with errors

### 4.5 Summary: The Four Pillars in Parsley

| Capability | Ecto | Parsley Approach | Status |
|------------|------|------------------|--------|
| **Filtering** | `cast(params, [:fields])` | Schema fields = whitelist | Implicit (good) |
| **Casting** | Schema types + cast() | Schema types + validate() | Already works in Parts |
| **Validation** | Declarative + pipeline | Declarative in @schema | Exists, could add custom |
| **Constraints** | DB error → changeset | table.insert() → record | Future phase |

### 4.6 What's Genuinely Useful with Minimal Complexity?

**Immediate value (Phase 1):**
- Filtering: implicit via schema (zero effort)
- Casting: already happens in Parts, formalize in Records
- Validation: `record.validate()` returns Record with errors

```parsley
// This is genuinely useful and simple
let form = User(props).validate()

if (form.isValid()) {
    @insert(Users |< ...form .)
    <div>"Success!"</div>
} else {
    default({form: form})  // Re-render with errors
}
```

**Later value (Phase 2+):**
- Constraints: `@insert()` returns record with DB errors
- Custom validators: post-process with `record.withError()`
- Explicit permit: `record.permit(["name", "email"])` for partial updates

## 5. Property Access Design

### 5.1 The Conflict

Records have both data fields and metadata:

```parsley
user.name       // Data: "Alice"
user.errors     // Metadata: {name: "Too short"}
user.ok         // Metadata: true/false
user.schema     // Metadata: the schema itself
```

What if the schema has a field called `errors` or `ok`?

```parsley
let BadSchema = @schema {
    ok: bool(),
    errors: string()
}
let record = BadSchema({ok: true, errors: "none"})

record.ok      // The boolean data? Or the validity flag?
record.errors  // The string data? Or the error dictionary?
```

### 5.2 Option A: Reserved Names (Problematic)

Reserve `ok`, `errors`, `schema`, `valid`, etc. as metadata-only.

**Problem:** Arbitrary restrictions on field names. What if you're wrapping an API that uses `errors`?

### 5.3 Option B: Explicit Namespaces (Verbose)

```parsley
record.data.name      // Data access
record.meta.errors    // Metadata access
record.meta.ok        // Validity
```

**Problem:** Verbose for the common case (data access).

### 5.4 Option C: Data-First with Method Accessors (Recommended)

Data fields accessed directly. Metadata via methods:

```parsley
// Data access - direct
record.name           // "Alice"
record.email          // "alice@example.com"
record.ok             // The data field if it exists!

// Metadata access - methods
record.errors()       // {name: "Too short"}
record.isValid()      // true/false
record.schema()       // The schema
record.isDirty()      // If we add changes tracking later
record.error("name")  // "Too short" or null
```

**Rationale:**
- Data access is the common case → make it terse
- Metadata access is less frequent → methods are acceptable
- No reserved field names
- Clear visual distinction: `record.name` vs `record.errors()`

### 5.5 Method Summary

**Record methods:**

| Method | Returns | Description |
|--------|---------|-------------|
| `record.errors()` | Dictionary | All field errors |
| `record.error(field)` | String or null | Error for specific field |
| `record.isValid()` | Boolean | True if no errors |
| `record.schema()` | Schema | The schema used |
| `record.data()` | Dictionary | Plain dict of all data |
| `record.keys()` | Array | Field names |
| `record.hasError(field)` | Boolean | True if field has error |
| `record.validate()` | Record | Validate and return with errors |

**Typed table methods (table with schema):**

| Method | Returns | Description |
|--------|---------|-------------|
| `table.validate()` | Table | Bulk validate all rows |
| `table.isValid()` | Boolean | True if all rows valid |
| `table.errors()` | Array | `[{row: 0, field: "x", error: "..."}]` |
| `table.validRows()` | Table | Rows that passed validation |
| `table.invalidRows()` | Table | Rows that failed, with errors |
| `table.schema()` | Schema | The schema used |

## 6. Mutability Design

### 6.1 Immutable (Recommended)

Records are immutable. "Modifications" return new records:

```parsley
let user = User({name: "Alice"})
let updated = user.set("name", "Bob")

user.name       // "Alice" (unchanged)
updated.name    // "Bob" (new record)

// Bulk update
let updated = user.merge({name: "Bob", age: 31})
```

**Rationale:**
- Matches Parsley's functional style
- Predictable: no spooky action at a distance
- Safe to pass records around without defensive copying
- Works well with Parts' stateless model

### 6.2 Practical Pattern

```parsley
export save = fn(props) {
    let form = User(props).validate()
    
    if (form.isValid()) {
        @insert(Users |< ...form .)
        <div>"Saved!"</div>
    } else {
        // form already has errors attached
        default({form: form})
    }
}
```

No mutation needed — `validate()` returns a new record with errors.

## 7. Schema Field Metadata

### 7.1 Design Principle

Focus on the 90% use case. Most applications:
- Are single-language (or handle i18n at a different layer)
- Want human-readable field titles
- Need basic display hints (placeholder, format)

Don't over-engineer for i18n in V1. Build something useful now; extend later.

### 7.2 Pipe Syntax for Metadata

Separate validation from metadata using `|`:

```parsley
let UserSchema = @schema {
    name: string(min: 2, max: 100, required) | {title: "Full Name"},
    email: email(required) | {title: "Email Address", placeholder: "you@example.com"},
    age: int(min: 0) | {title: "Age"},
    createdAt: datetime() | {title: "Created", format: "date"}
}
```

**Why this syntax:**
- **Visual separation:** Validation rules on left, display hints on right
- **Backward compatible:** Old syntax (no `|`) still works
- **Extensible:** Metadata dict can hold anything
- **Familiar:** Pipe as "with" or "annotated by" reads naturally

**Old syntax remains valid:**

```parsley
// Still works — no metadata
let SimpleSchema = @schema {
    name: string(required),
    email: email()
}
```

### 7.3 Metadata is an Open Dictionary

The metadata after `|` is just a dictionary. Users can put **any** key-value pairs they want:

```parsley
let UserSchema = @schema {
    name: string(required) | {title: "Name", sortable: true, searchWeight: 2},
    avatar: string() | {title: "Avatar", widget: "image-upload", maxSize: "5mb"}
}

// Access any metadata
UserSchema.meta("name", "sortable")     // true
UserSchema.meta("avatar", "widget")     // "image-upload"
UserSchema.meta("avatar", "maxSize")    // "5mb"
```

This keeps the door open for:
- Custom form widgets (`widget: "rich-text"`, `widget: "color-picker"`)
- App-specific hints (`sortable: true`, `searchable: true`, `exportable: false`)
- Third-party integrations
- Domain-specific metadata

### 7.4 Core Metadata (Used by Built-ins)

These are the fields that **built-in features will look for**. They're not special — just documented conventions:

| Metadata | Purpose | Used by |
|----------|---------|---------|
| `title` | Human-readable field name | Forms, tables, error messages |
| `placeholder` | Input placeholder text | Forms |
| `format` | Display format | Tables, read-only views |
| `hidden` | Exclude from default display | Tables, auto-generated forms |
| `help` | Help text below field | Forms |

**Example with core + custom metadata:**

```parsley
let User = @schema {
    id: int() | {hidden: true},
    name: string(min: 2, required) | {title: "Full Name", placeholder: "Enter name"},
    email: email(required) | {title: "Email", placeholder: "you@example.com"},
    salary: decimal() | {title: "Salary", format: "currency"},
    startDate: date() | {title: "Start Date", format: "date"},
    notes: text() | {title: "Notes", help: "Any additional information"}
}
```

### 7.5 Accessing Metadata

**From a record:**

```parsley
let user = User({name: "Alice"})

user.schema().title("name")        // "Full Name"
user.schema().placeholder("email") // "you@example.com"
user.schema().format("salary")     // "currency"

// With fallback to field name
user.schema().title("unknownField") // "Unknown Field" (titlecase)
```

**From the schema directly:**

```parsley
User.title("name")                 // "Full Name"
User.fields()                      // ["id", "name", "email", ...]
User.visibleFields()               // ["name", "email", ...] (excludes hidden)
```

### 7.6 Format Method on Records

Records can format values using schema hints:

```parsley
let user = @query(Users | id == {id} ?-> *)

user.salary                  // 52000 (raw number)
user.format("salary")        // "$52,000.00" (formatted per schema)
user.format("startDate")     // "Jan 15, 2025" (formatted per schema)
```

**Built-in formats:**

| Format | Example Input | Example Output |
|--------|---------------|----------------|
| `"date"` | `2025-01-15` | "Jan 15, 2025" |
| `"datetime"` | `2025-01-15T14:30:00Z` | "Jan 15, 2025 2:30 PM" |
| `"currency"` | `52000` | "$52,000.00" |
| `"percent"` | `0.15` | "15%" |
| `"number"` | `1234567` | "1,234,567" |

### 7.7 Future: Functions as Metadata Values

The pipe syntax naturally extends to dynamic values:

```parsley
// V2: Functions for i18n
let User = @schema {
    name: string(required) | {title: fn() { @i18n("user.name") }},
    email: email() | {title: fn() { @i18n("user.email") }}
}

// Or if @i18n returns a value directly
let User = @schema {
    name: string(required) | {title: @i18n("user.name")},
    email: email() | {title: @i18n("user.email")}
}
```

This keeps the door open for i18n without complicating V1.

### 7.8 Metadata Inheritance and Override

Schema metadata can be overridden at point of use:

```parsley
// Schema has defaults
let User = @schema {
    name: string(required) | {title: "Full Name"}
}

// Override for specific context
<form @record={record} @titles={name: "Your Name"}>
    ...
</form>

// Or for tables with short headers
<table @records={users} @titles={name: "Name", email: "Email"}>
    ...
</table>
```

The layering: explicit override > schema metadata > field name titlecase

## 8. Database Integration

### 8.1 Current Model

```parsley
let User = @schema { name: string(required), email: email() }
let Users = @table(User, "users")  // or db.bind(User, "users")

let user = @query(Users | id == {id} ?-> *)  // Returns Dictionary
@insert(Users |< name: "Alice" .)             // Accepts Dictionary
@update(Users | id == {id} |< name: "Bob" .) // Accepts Dictionary
```

### 8.2 Record-Aware Model

When a table is bound with a schema, queries automatically return Records:

```parsley
let User = @schema { name: string(required), email: email() }
let Users = @table(User, "users")

// Query returns Record (with schema!)
let user = @query(Users | id == {id} ?-> *)
user.name           // Data access
user.schema()       // Returns User schema
user.isValid()      // Method available

// Insert accepts Record (already validated)
let record = User(props).validate()
if (record.isValid()) {
    @insert(Users |< ...record .)
}
```

### 8.3 What This Enables

**Type safety:**

```parsley
let user = @query(Users | id == {id} ?-> *)
user.name                       // Known to exist (schema defines it)
user.nonexistent                // Could warn/error at parse time?
```

**Validation on insert:**

```parsley
// Validate before insert
let record = User(props).validate()
if (!record.isValid()) {
    // Handle errors
}
@insert(Users |< ...record .)
```

**Seamless edit flow:**

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

### 8.4 Records and Tables

Table queries return Records when bound with a schema:

```parsley
let User = @schema { name: string(required), email: email() }
let Users = @table(User, "users")

// Single record
let user = @query(Users | id == {id} ?-> *)
user.schema() == User           // true

// Multiple records
let users = @query(Users | age >= 18 ??-> *)
users[0].schema() == User       // true

// Raw SQL query (no schema)
let rows = @DB <=?=> "SELECT * FROM users" ??-> *
rows[0].schema()                // null or error - no schema attached
```

### 8.5 Backward Compatibility with Dictionaries

Records should work where dictionaries are expected:

```parsley
let user = Users.find(1)  // Record

// Pass to function expecting dictionary
someFunc(user)            // Works - record provides data as dict

// Spread into dictionary
let merged = {...user, extraField: "value"}  // Works

// JSON encoding
json.encode(user)         // Encodes data fields only

// Explicit conversion
let dict = user.data()    // Get plain dictionary
```

**Principle:** A Record IS-A Dictionary for data access. The schema and methods are "extra."

## 9. Record Creation API

### 9.1 Schema as Constructor (Recommended)

Schemas are callable — invoking a schema with data creates a Record:

```parsley
let User = @schema {
    name: string(min: 2, required),
    email: email(required)
}

// Call schema like a function → creates Record
let record = User({name: "Alice", email: "alice@example.com"})

record.name        // "Alice"
record.email       // "alice@example.com"
record.isValid()   // true (no errors yet — validation not run)
```

**Why this design:**
- Natural syntax: `User(data)` reads as "a User from this data"
- Consistent: schemas define structure, calling them constructs instances
- No confusion: `User.new(data)` sounds like "create a new schema"

### 9.2 Validation as Record Method

Validation is an operation on a record, not a schema method:

```parsley
let record = User({name: "A", email: "bad"})
let validated = record.validate()

validated.isValid()      // false
validated.errors()       // {name: "...", email: "..."}
validated.error("name")  // "Must be at least 2 characters"
```

**Why this design:**
- Separates construction from validation
- Immutable: `validate()` returns a new record with errors
- Chainable: `User(data).validate()`
- Clear: the record is being validated, not the schema

### 9.3 Common Patterns

```parsley
// Create and validate in one step
let form = User(props).validate()

// Create empty record for new form
let blank = User({})

// Validate form submission
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

### 9.4 Schema as Callable: Dict or Array

Schemas are callable with either a dictionary (→ Record) or an array (→ Table):

```parsley
let Product = @schema {
    sku: string(required),
    name: string(required),
    price: money()
}

// Single dict → Record
let item = Product({sku: "A001", name: "Widget", price: $9.99})
item.isValid()      // Record methods available

// Array of dicts → Table  
let products = Product([
    {sku: "A001", name: "Widget", price: $9.99},
    {sku: "A002", name: "Gadget", price: $19.99}
])
products.length     // Table methods available
```

**Why both?**
- Natural: `Product(data)` reads as "Product(s) from this data"
- Type-driven: array in → table out; dict in → record out
- Consistent: schema is always the constructor, regardless of cardinality

**Table literals still work:**

```parsley
// Equivalent ways to create a typed table
let products = Product([...])
let products = @table(Product) [...]
```

The `@table(Schema)` literal is useful when you want explicit compile-time schema binding. `Schema([...])` is useful when data is dynamic or when chaining.

### 9.5 The `.as(Schema)` Method

For dynamic schema application, use `.as(Schema)` on dicts or tables:

```parsley
// On a dictionary → Record
let item = {sku: "A001", name: "Widget", price: $9.99}.as(Product)
item.isValid()      // Record methods available

// On a table → Typed Table
let data = [{x: 1, y: 1}, {x: 2, y: 1}, {x: 3, y: 4}]
let t = table(data).as(Point)
```

**When to use `.as()` vs callable:**

| Syntax | Best for |
|--------|----------|
| `Schema({...})` | Primary idiom, literal data |
| `Schema([...])` | Primary idiom, literal array |
| `{...}.as(Schema)` | Chaining: `fetch().parse().as(Schema)` |
| `table(data).as(Schema)` | Dynamic data, runtime schema binding |

**Chaining example:**

```parsley
// .as() chains naturally in pipelines
let product = fetchFromAPI().parse().as(Product)
let row = csvRows[0].as(Product)

// Callable requires intermediate variable or nested parens
let product = Product(fetchFromAPI().parse())  // Less fluent
```

### 9.6 Table Validation

Tables support bulk validation via `.validate()`:

```parsley
let data = loadCSV("products.csv")
let products = table(data).as(Product)
let validated = products.validate()

validated.isValid()      // true if ALL rows valid
validated.errors()       // [{row: 0, field: "sku", error: "Required"}, ...]
validated.validRows()    // Table of valid rows only
validated.invalidRows()  // Table of invalid rows with errors attached
```

**Use cases:**
- **CSV import:** Validate all rows, show errors for bad ones, insert good ones
- **Bulk API:** Validate batch before processing
- **Data migration:** Check entire dataset against new schema

**What a schema enables on a table:**

| Capability | Description |
|------------|-------------|
| `t.validate()` | Bulk validate all rows |
| `@query(t \| ... ?-> *)` | Returns Record (not Dict) |
| HTML export | Tables know column types for formatting |
| Insert validation | Could validate each row as inserted |

### 9.7 Records vs Tables: When to Use Which

| Type | Created By | Use Case |
|------|------------|----------|
| **Record** | `Schema({...})` or `{...}.as(Schema)` | Single entity: form data, API request, one DB row |
| **Table** | `Schema([...])` or `table(data).as(Schema)` | Multiple entities: CSV data, query results, reports |

**Key difference:** Records carry per-field validation errors. Tables can validate in bulk but track errors differently.

```parsley
// Record: per-field errors
let user = User({name: ""}).validate()
user.isValid()           // false
user.error("name")       // "Required"

// Table: bulk validation
let users = User([{name: ""}, {name: "Bob"}]).validate()
users.isValid()          // false (one invalid row)
users.validRows()        // Table with just {name: "Bob"}
users.invalidRows()      // Table with {name: ""} + error info
```

**For form validation:** Use Records.
**For bulk import/pipelines:** Use Tables with `.validate()`.

### 9.8 Database Queries Return Records

When a table is bound with a schema, queries return Records automatically:

```parsley
let User = @schema { name: string(required), email: email() }
let Users = @table(User, "users")

// Single row query returns Record
let user = @query(Users | id == {id} ?-> *)
user.name           // Data access
user.isValid()      // Method available
user.schema()       // Returns User schema

// Multiple row query returns Table of Records
let admins = @query(Users | role == "admin" ??-> *)
admins.length       // Table method
admins[0].name      // Each row is a Record
admins[0].isValid() // Record method available
```

**Summary of return types:**

| Source | Syntax | Returns |
|--------|--------|---------|
| Schema + dict | `User({...})` | Record |
| Schema + array | `User([...])` | Table |
| Dict + schema | `{...}.as(User)` | Record |
| Table + schema | `table(data).as(User)` | Table |
| Table literal | `@table(User) [...]` | Table |
| DB single row | `@query(... ?-> *)` | Record (if table has schema) |
| DB multiple rows | `@query(... ??-> *)` | Table of Records |
| Raw SQL single | `@DB <=?=> "..." ?-> *` | Dictionary |
| Raw SQL multiple | `@DB <=?=> "..." ??-> *` | Array of Dictionaries |

### 9.9 Summary: Creation Methods

| Syntax | Returns | Purpose |
|--------|---------|---------|
| `Schema({...})` | Record | Create record from single dict |
| `Schema([...])` | Table | Create table from array of dicts |
| `{...}.as(Schema)` | Record | Apply schema to existing dict (chaining) |
| `table(data).as(Schema)` | Table | Apply schema to existing table (dynamic) |
| `@table(Schema) [...]` | Table | Table literal with compile-time schema |
| `record.validate()` | Record | Validate record, return with errors |
| `table.validate()` | Table | Bulk validate, track valid/invalid rows |
| `@query(Table \| ... ?-> *)` | Record | Load single row from DB |
| `@query(Table \| ... ??-> *)` | Table | Load multiple rows from DB |

## 10. Beyond Forms: Other Uses for Records

### 10.1 API Request/Response Validation

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
    
    // params.page, params.limit, params.sort are validated and typed
    let results = search(params.page, params.limit, params.sort)
    {status: 200, body: results}
}
```

### 10.2 Configuration Validation

```parsley
let Config = @schema {
    port: int(min: 1, max: 65535, required),
    host: string(default: "localhost"),
    debug: bool(default: false),
    maxConnections: int(min: 1, default: 100)
}

let config = Config(@env).validate()
if (!config.isValid()) {
    log("Invalid config: " + json.encode(config.errors()))
    exit(1)
}

// config.port, config.host etc are validated
```

### 10.3 Event/Message Validation

```parsley
let UserCreatedEvent = @schema {
    type: enum("user.created"),
    userId: int(required),
    email: email(required),
    timestamp: datetime(required)
}

let event = UserCreatedEvent(incomingMessage).validate()
if (event.isValid()) {
    processUserCreated(event)
}
```
```

### 10.4 Domain Objects

Records can represent validated domain objects, not just form data:

```parsley
let Money = @schema {
    amount: decimal(min: 0, required),
    currency: enum("USD", "EUR", "GBP", required)
}

let Price = @schema {
    base: Money(required),
    tax: Money(required),
    total: Money(required)
}

let invoice = Price.validate(data)
```

### 10.5 DTO / Transfer Objects

```parsley
// API returns this shape
let UserDTO = @schema {
    id: int(required),
    name: string(required),
    email: email(required),
    createdAt: datetime(required)
}

let response = await fetch("/api/users/1")
let user = UserDTO.validate(response.json())

if (user.isValid()) {
    // user.id, user.name, etc are guaranteed to exist and have correct types
}
```

## 11. Implementation Considerations

### 11.1 Go Type Definition

```go
type Record struct {
    schema    *DSLSchema
    data      map[string]Object
    errors    map[string]string
}

func (r *Record) Type() ObjectType { return RECORD_OBJ }

// Data access - implements dictionary-like interface
func (r *Record) Get(key string) (Object, bool) {
    val, ok := r.data[key]
    return val, ok
}

// Method implementations
func (r *Record) Errors() *Dictionary { ... }
func (r *Record) Error(field string) Object { ... }
func (r *Record) IsValid() bool { return len(r.errors) == 0 }
func (r *Record) Schema() *DSLSchema { return r.schema }
func (r *Record) Data() *Dictionary { ... }

// Immutable updates
func (r *Record) Set(key string, value Object) *Record {
    newData := copyMap(r.data)
    newData[key] = value
    return &Record{schema: r.schema, data: newData, errors: r.errors}
}

func (r *Record) Merge(updates map[string]Object) *Record { ... }
```

### 11.2 Schema Methods

```go
// Schema.new(data?) → Record
func (s *DSLSchema) New(data *Dictionary) *Record {
    return &Record{
        schema: s,
        data:   dictToMap(data),
        errors: make(map[string]string),
    }
}

// Schema.validate(data) → Record (with errors)
func (s *DSLSchema) Validate(data *Dictionary) *Record {
    record := s.New(data)
    errors := validateAgainstSchema(s, data)
    record.errors = errors
    return record
}
```

### 11.3 Dictionary Interoperability

Records should be usable where dictionaries are expected:

```go
// In spread evaluation
case *Record:
    // Spread record's data fields
    for key, value := range record.data {
        result[key] = value
    }

// In property access
func evalIndexExpression(left, index Object) Object {
    switch obj := left.(type) {
    case *Record:
        return obj.Get(index.(*String).Value)
    // ...
    }
}
```

### 11.4 Form Object (Higher-Level)

The "form" is a Record plus display metadata:

```go
type Form struct {
    record       *Record
    labels       func(string) string
    placeholders func(string) string
    help         func(string) string
}

func (f *Form) Label(field string) string {
    if f.labels != nil {
        return f.labels(field)
    }
    return titleCase(field)  // Default
}

func (f *Form) Input(field string, opts map[string]Object) string {
    // Generate <input> with attrs from schema + value from record
}
```

## 12. Migration and Compatibility

### 12.1 Is This a Breaking Change?

**No.** Records are additive:

| Feature | Before | After |
|---------|--------|-------|
| `@schema{}` | Returns schema | Returns callable schema |
| `@table()` queries | Return dicts | Return Records |
| Dictionary spread | Works | Works on Records too |

### 12.2 Migration Path

**Phase 1:** Add Record type, schemas become callable

```parsley
// New: schema is callable, returns Record
let record = User(data)
let validated = record.validate()

if (!validated.isValid()) { ... }
```

**Phase 2:** Update `@table()` queries to return Records

```parsley
let user = @query(Users | id == {id} ?-> *)
user.name              // Works (data access)
user.isValid()         // Now available
```

**Phase 3:** Add Form binding with metadata

```parsley
<form @record={record}>
    <input @name="email"/>
</form>
```

### 12.3 Gradual Adoption

Users can adopt Records incrementally:
- Keep using dicts if preferred
- Use Records only where validation is needed
- Mix: `record.data()` converts back to dict

## 13. Complete Example

### 13.1 Schema Definition

```parsley
let User = @schema {
    name: string(min: 2, max: 100, required) | {title: "Full Name"},
    email: email(required) | {title: "Email Address", placeholder: "you@example.com"},
    age: int(min: 18) | {title: "Age"},
    role: enum("admin", "user", "guest", default: "user") | {title: "Role"}
}
```

### 13.2 Database Binding

```parsley
let Users = @table(User, "users")
```

### 13.3 Part with Form (Verbose)

The explicit approach — all wiring visible:

```parsley
// users/edit.part

export default = fn(props) {
    let record = props.record ?? User({})
    
    <form part-submit="save">
        <div class="field">
            <label>Name</label>
            <input name="name" value={record.name} required minlength=2 maxlength=100/>
            {record.hasError("name") && <span class="error">{record.error("name")}</span>}
        </div>
        
        <div class="field">
            <label>Email</label>
            <input name="email" type="email" value={record.email} required/>
            {record.hasError("email") && <span class="error">{record.error("email")}</span>}
        </div>
        
        <div class="field">
            <label>Age</label>
            <input name="age" type="number" value={record.age} min=18/>
            {record.hasError("age") && <span class="error">{record.error("age")}</span>}
        </div>
        
        <button type="submit">Save</button>
    </form>
}
```

### 13.4 Part with Form (Binding)

The binding approach — record provides context via AST rewriting:

```parsley
// users/edit.part

export default = fn(props) {
    let record = props.record ?? User({})
    
    <form @record={record} part-submit="save">
        <div class="field">
            <label>Name</label>
            <input @name="name"/>
            <Error @name="name"/>
        </div>
        
        <div class="field">
            <label>Email</label>
            <input @name="email"/>
            <Error @name="email"/>
        </div>
        
        <div class="field">
            <label>Age</label>
            <input @name="age"/>
            <Error @name="age"/>
        </div>
        
        <button type="submit">Save</button>
    </form>
}
```

**What the evaluator does:**

When `<form @record={record}>` is encountered, it establishes a form context. Child elements with `@name` are rewritten:

| Original | Rewritten (conceptually) |
|----------|-------------------------|
| `<input @name="name"/>` | `<input name="name" value={record.name} required minlength=2 maxlength=100/>` |
| `<Error @name="name"/>` | `{record.hasError("name") && <span class="error">{record.error("name")}</span>}` |

The schema provides validation attributes (`required`, `minlength`, etc.), the record provides the value, and errors come from validation state.

### 13.5 Part with Form (Maximally Terse)

If labels also come from metadata:

```parsley
// users/edit.part

export default = fn(props) {
    let record = props.record ?? UserSchema.new()
    
    <form @record={record} part-submit="save">
        <Field @name="name"/>
        <Field @name="email"/>
        <Field @name="age"/>
        <button type="submit">Save</button>
    </form>
}
```

Where `<Field @name="x"/>` expands to the full field structure (label + input + error). This is maximum terseness but less flexible for custom layouts.

### 13.6 Adding Metadata (Labels, Placeholders, Help)

Records carry schema + data + errors. But what about display metadata like labels?

**Option A: Metadata dictionary on form**

```parsley
export default = fn(props) {
    let record = props.record ?? UserSchema.new()
    let labels = {
        name: @i18n("user.name"),
        email: @i18n("user.email"),
        age: @i18n("user.age")
    }
    
    <form @record={record} @labels={labels} part-submit="save">
        <div class="field">
            <Label @name="name"/>   // Uses @labels context
            <input @name="name"/>
            <Error @name="name"/>
        </div>
        ...
    </form>
}
```

**Option B: Wrap record with metadata**

```parsley
export default = fn(props) {
    let record = props.record ?? UserSchema.new()
    let form = record.withMeta({
        labels: fn(field) { @i18n("user." + field) },
        placeholders: {name: "Enter your name", email: "you@example.com"}
    })
    
    <form @record={form} part-submit="save">
        <div class="field">
            <Label @name="name"/>   // Form carries labels
            <input @name="name"/>   // Form carries placeholders
            <Error @name="name"/>
        </div>
        ...
    </form>
}
```

**Option C: Keep labels explicit (simplest)**

Don't try to bind labels — they're usually static per template anyway:

```parsley
export default = fn(props) {
    let record = props.record ?? UserSchema.new()
    
    <form @record={record} part-submit="save">
        <div class="field">
            <label>Name</label>           // Just write it
            <input @name="name"/>
            <Error @name="name"/>
        </div>
        ...
    </form>
}
```

For i18n, use interpolation:

```parsley
<label>{@i18n("user.name")}</label>
```

**Recommendation:** Start with Option C (explicit labels). The binding magic is most valuable for:
- Values (must match field names exactly)
- Validation attributes (tedious to repeat from schema)
- Errors (conditional display logic)

Labels are usually written once and rarely change. The i18n case can use interpolation without special binding.

If real usage shows that label binding would help (e.g., generating admin forms from schema), add Option A or B later.

### 13.8 Handler (Save)

```parsley
export save = fn(props) {
    let record = UserSchema.validate(props)
    
    if record.isValid()
        Users.insert(record)
        <div class="success">User {record.name} saved!</div>
    else
        default({record: record})
}
```

### 13.9 Handler (Load)

```parsley
// handlers/users/[id].pars

let Users = @table(UserSchema, "users")
let user = Users.find(@params.id)

<html>
<head><title>Edit {user.name}</title></head>
<body>
    <h1>Edit User</h1>
    <Part src="./edit.part" record={user}/>
</body>
</html>
```

## 14. Design Questions Summary

| Question | Recommendation |
|----------|---------------|
| Track changes? | No (start simple, add later if needed) |
| Property access | Data direct, metadata via methods |
| Mutability | Immutable (new record on "change") |
| Metadata in schema | Yes, via pipe syntax `type() | {title: "..."}` |
| DB returns records | Yes (seamless validation/edit flow) |
| Dict compatibility | Yes (record IS-A dict for data access) |
| Literal syntax | No (use Schema.new() factory) |

## 15. Design Decisions (Resolved)

1. **Error message customization:** Where do custom error messages live?

   **Decision:** Handle in code, not schema. Default error messages come from the validation system. Custom messages require code — either in a custom validator function or by post-processing the record's errors. This keeps the schema declarative and simple.

2. **Nested records:** Can a schema field be another schema?

   **Decision:** Not in V1. Too complicated. If composition is needed later, consider mixins rather than nesting. Keep schemas flat for now.

3. **Computed/virtual fields:** Can a record have fields derived from others?

   **Decision:** No. Computed values belong in code, done manually:
   ```parsley
   let fullName = record.firstName + " " + record.lastName
   ```
   Records are data containers, not computed property systems.

4. **Validation dependencies:** Can one field's validation depend on another?

   **Decision:** Handle in code. For cases like "confirmPassword must match password", use post-processing after schema validation:
   
   ```parsley
   let record = User(props).validate()
   if (record.password != props.confirmPassword) {
       record = record.withError("confirmPassword", "Passwords don't match")
   }
   ```
   No pre/post hooks or declarative cross-field validation in V1. Keep it simple.

5. **Form context magic:** What should `<form @record={record}>` bind automatically?

   **Decision:** Bind everything the record provides:
   - **Values** from `record.fieldName`
   - **Validation attributes** from schema (required, min, max, type, etc.)
   - **Errors** from `record.errors()`
   
   The record carries data + schema + errors, so the form binding should use all three.

6. **Custom validation hooks:** Should we provide a way to run custom validation code?

   **Prior art:**
   
   | System | Approach |
   |--------|----------|
   | **Ecto/Phoenix** | Pipeline: `changeset |> validate_required() |> custom_validator()` — validators are functions that take/return changeset |
   | **Rails** | Callbacks: `validate :my_method` — methods on the model class |
   | **Django** | Methods: `clean()` and `clean_<field>()` — override in form subclass |
   | **Yup** | Chained: `.test('name', 'message', fn)` — inline test functions |
   | **Zod** | Refinements: `.refine(fn, opts)` and `.superRefine(fn)` — post-parse validation |

   **Common patterns:**
   - Per-field validation (validate one field with custom logic)
   - Cross-field validation (compare password/confirmPassword)
   - Post-validation hooks (run after all field validation)

   **Options for Parsley:**

   **Option A: Explicit post-processing (simplest)**
   
   ```parsley
   let record = User(props).validate()
   if (record.password != props.confirmPassword) {
       record = record.withError("confirmPassword", "Passwords don't match")
   }
   ```
   Pro: No new concepts. Con: Verbose for common patterns.

   **Option B: Validator function argument**
   
   ```parsley
   let record = User(props).validate(fn(r) {
       if r.password != props.confirmPassword
           r.withError("confirmPassword", "Passwords don't match")
       else
           r
   })
   ```
   Pro: Single expression. Con: New API surface.

   **Option C: Schema-level validator**
   
   ```parsley
   let User = @schema {
       password: string(required),
       confirmPassword: string(required)
   } | {
       validate: fn(r) {
           if (r.password != r.confirmPassword) {
               r.withError("confirmPassword", "Passwords don't match")
           } else {
               r
           }
       }
   }
   ```
   
   Pro: Keeps validation with schema. Con: Mixes concerns (schema metadata vs logic).

   **Decision:** Start with **Option A** (explicit post-processing). It requires no new concepts — just use `record.withError()` after `validate()`. This is clear, debuggable, and covers the 10% of cases that need custom validation.

   If patterns emerge where Option B would significantly reduce boilerplate, add it later. Option C feels like it mixes declarative schema with imperative code — avoid.

## 16. Implementation Phases

### Phase 1: Core Record Type
- Add Record type to evaluator
- Make schemas callable: `Schema(data)` → Record
- Add `record.validate()` → Record with errors
- Record methods: `errors()`, `error(field)`, `isValid()`, `schema()`, `data()`, `withError()`
- Record property access → data fields
- Record spreads as dictionary

### Phase 2: AST-Level Form Binding

The key insight: `<form @record={...}>` establishes context that child tags can access.

**Implementation approach:**

1. When evaluating `<form @record={record}>`:
   - Store record in environment context (like we do for `BasilCtx`)
   - Process child nodes with this context available

2. When evaluating `<input @name="field">`:
   - Check for record context in environment
   - If found, rewrite attributes from schema + record:
     - `name="field"` (from @name)
     - `value={record.field}` (from record data)
     - `required`, `minlength`, `type`, etc. (from schema)

3. Built-in pseudo-components:
   - `<Error @name="field"/>` → conditional error span
   - `<Label @name="field"/>` → label from metadata (optional)
   - `<Field @name="field"/>` → complete field wrapper (optional)

**AST rewriting in evalTagPair:**

```go
func evalTagPair(node *ast.TagPairExpression, env *Environment) Object {
    tagName := node.TagName
    
    // Check for @record context establishment
    if hasAttr(node, "@record") {
        record := evalAttr(node, "@record", env)
        childEnv := NewEnclosedEnvironment(env)
        childEnv.Set("__formRecord__", record)
        return evalFormTag(node, childEnv)
    }
    
    // Check for @name in context of a record
    if hasAttr(node, "@name") {
        if record := env.Get("__formRecord__"); record != nil {
            return evalBoundInput(node, record, env)
        }
    }
    
    // ... normal tag evaluation
}
```

### Phase 3: Database Integration
- `@table()` queries return Records when schema is bound
- Records spread into `@insert`/`@update` naturally
- Constraint errors can be mapped back to record via `withError()`

### Phase 4: Advanced Features (Future)
- Changes tracking (`record.changes()`, `record.original()`)
- Nested schema support
- `record.permit()` for explicit field whitelisting
- Computed fields

## 17. Next Steps

1. Review this design
2. If direction is approved, create FEAT-XXX for Phase 1
3. Prototype Record type in evaluator
4. Test with simple form examples
5. Iterate based on real usage
