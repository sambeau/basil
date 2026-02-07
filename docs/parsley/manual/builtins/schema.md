---
id: man-pars-schema
title: Schemas
system: parsley
type: builtin
name: schema
created: 2026-01-16
version: 0.15.3
author: Basil Team
keywords:
  - schema
  - type
  - validation
  - database
  - form
  - record
  - table
  - metadata
---

# Schemas

Schemas define the structure of records and tables in Parsley. They specify field names, types, validation rules, default values, and metadata for UI generation. Schemas are central to Parsley's approach to structured data—they drive database table creation, form validation, and auto-generated UIs.

```parsley
@schema User {
    id: integer
    name: string | {title: "Full Name"}
    email: email(unique: true) | {placeholder: "you@example.com"}
    role: enum["user", "admin"] = "user"
    active: boolean = true
    createdAt: datetime = @now | {hidden: true}
}

// Use with tables
let users = @table(User) [
    {id: 1, name: "Alice", email: "alice@example.com"},
    {id: 2, name: "Bob", email: "bob@example.com", role: "admin"}
]

// Access schema metadata
User.title("name")              // "Full Name"
User.visibleFields()            // ["id", "name", "email", "role", "active"]
```

## Why Schemas?

Schemas provide a single source of truth for your data structure:

| Feature | Without Schema | With Schema |
|---------|---------------|-------------|
| Type safety | None | Validated on creation |
| Default values | Manual per row | Automatic |
| Database tables | Write SQL manually | Auto-generated |
| Form labels | Hardcoded | `schema.title(field)` |
| Hidden fields | Track separately | `schema.visibleFields()` |
| Enum options | Hardcoded arrays | `schema.enumValues(field)` |

---

## Declaring Schemas

### Basic Declaration

Use `@schema` followed by a name and field definitions in braces:

```parsley
@schema Person {
    name: string
    age: integer
    email: email
}
```

### Field Types

Parsley supports these built-in types:

| Type | Description | SQL Type | Validation |
|------|-------------|----------|------------|
| `string` | Text data | `TEXT` | None |
| `text` | Long text | `TEXT` | None |
| `int`, `integer` | Whole numbers | `INTEGER` | Numeric |
| `bigint` | Large integers | `BIGINT` | Numeric |
| `float`, `number` | Decimal numbers | `REAL` | Numeric |
| `bool`, `boolean` | True/false | `INTEGER` | Boolean |
| `datetime` | Date and time | `DATETIME` | ISO format |
| `date` | Date only | `DATE` | ISO format |
| `time` | Time only | `TIME` | Time format |
| `money` | Monetary values | `REAL` | Currency |
| `id` | ID alias for `ulid` | `TEXT` | ULID format |
| `uuid` | UUID strings | `TEXT` | UUID format |
| `ulid` | ULID strings | `TEXT` | ULID format |
| `json` | JSON data | `TEXT` | Valid JSON |
| `email` | Email addresses | `TEXT` | Email pattern |
| `url` | URLs | `TEXT` | URL pattern |
| `phone` | Phone numbers | `TEXT` | Phone pattern |
| `slug` | URL slugs | `TEXT` | Slug pattern |
| `enum[...]` | Enumerated values | `TEXT` | In list |

### Nullable Fields

By default, fields are required. Append `?` to make a field nullable (optional):

```parsley
@schema Profile {
    name: string           // Required - cannot be null
    nickname: string?      // Optional - can be null
    bio: text?             // Optional
}
```

### Default Values

Use `=` after the type to specify a default value:

```parsley
@schema Article {
    title: string
    status: string = "draft"
    views: integer = 0
    featured: boolean = false
    createdAt: datetime = @now
}
```

When creating records or table rows without these fields, the defaults are applied automatically.

### Enum Types

Define allowed values inline with `enum[...]`:

```parsley
@schema Task {
    title: string
    priority: enum["low", "medium", "high"] = "medium"
    status: enum["todo", "in-progress", "blocked", "done"]
}

// Access enum values programmatically
Task.enumValues("priority")     // ["low", "medium", "high"]
Task.enumValues("status")       // ["todo", "in-progress", "blocked", "done"]
```

---

## Type Constraints

Add constraints using `(key: value)` syntax after the type:

```parsley
@schema Registration {
    username: string(min: 3, max: 20, unique: true)
    password: string(min: 8)
    age: integer(min: 13, max: 120)
}
```

| Constraint | Applies To | Description |
|------------|------------|-------------|
| `min` | `string` | Minimum string length |
| `min` | `integer`, `number` | Minimum numeric value |
| `max` | `string` | Maximum string length |
| `max` | `integer`, `number` | Maximum numeric value |
| `pattern` | `string` | Regex pattern for validation |
| `required` | Any | Field must have a non-null value |
| `auto` | Any | Database/server generates this value |
| `readOnly` | Any | Field cannot be set from client/form input |
| `unique` | Any | SQL UNIQUE constraint |

```parsley
// String length constraints
@schema Comment {
    body: text(min: 1, max: 10000)
}

// Numeric range constraints
@schema Product {
    price: money(min: 0)
    quantity: integer(min: 0, max: 999)
}

// Unique constraint for database
@schema User {
    email: email(unique: true)
}
```

### The `auto` Constraint

The `auto` constraint marks fields whose values are generated by the database or server, such as auto-increment IDs or timestamps. This solves the common problem where a schema needs an `id` field for completeness, but insert operations cannot provide one.

```parsley
@schema User {
    id: id(auto)                     // Database generates on insert
    createdAt: datetime(auto)        // Server sets on insert
    updatedAt: datetime(auto)        // Server sets on insert/update
    name: string(required)
    email: email(required)
}

// Valid - auto fields don't need to be provided
let user = User({name: "Alice", email: "alice@example.com"})
user.validate().isValid()            // true

// Auto fields are immutable - cannot be changed via update()
user.update({id: "new-id"})          // Error: cannot update auto field 'id'
```

**Behavior:**

| Context | `auto` field behavior |
|---------|----------------------|
| `Schema({...})` | Optional, defaults to `null` |
| `record.validate()` | Skipped (not an error if missing) |
| `@insert` | Database/server generates value |
| `record.update()` | Immutable (error if changed) |
| `@query` result | Always present |
| `visibleFields()` | Excluded (not in list) |
| `@field` form binding | Renders as `type="hidden"` with `readonly` |

**Note:** `auto` and `required` cannot be combined on the same field — they are contradictory (auto fields are generated, not provided).

### ID Types

Parsley provides several ID types for primary keys. The `id` type is an alias for `ulid`:

| Type | Format | Sortable | Use Case |
|------|--------|----------|----------|
| `id` | ULID (alias) | ✅ Time-based | Default, recommended |
| `ulid` | 26 chars, base32 | ✅ Time-based | Distributed systems |
| `uuid` | 36 chars, hex | ❌ Random | UUID compatibility |
| `int(auto)` | Integer | ✅ Sequential | Simple auto-increment |
| `bigint(auto)` | 64-bit integer | ✅ Sequential | Large tables |

**Why `id` = `ulid`?** ULIDs are time-sortable (better for database indexing), URL-safe, don't expose business information, and work in distributed systems without coordination.

```parsley
// Recommended: explicit ID type with auto
@schema User {
    id: ulid(auto)       // Generates ULID on insert
    name: string
}

// Alternatives
@schema Product {
    id: uuid(auto)       // Generates UUID v4 on insert
    name: string
}

@schema Counter {
    id: int(auto)        // Database auto-increment
    value: int
}

// The 'id' type is an alias for 'ulid'
@schema Item {
    id: id(auto)         // Same as ulid(auto)
    name: string
}
```

**Database Mapping:**

| Type | SQLite | PostgreSQL |
|------|--------|------------|
| `uuid(auto)` | `TEXT PRIMARY KEY` | `UUID PRIMARY KEY DEFAULT gen_random_uuid()` |
| `ulid(auto)` | `TEXT PRIMARY KEY` | `TEXT PRIMARY KEY` |
| `int(auto)` | `INTEGER PRIMARY KEY` | `SERIAL PRIMARY KEY` |
| `bigint(auto)` | `INTEGER PRIMARY KEY` | `BIGSERIAL PRIMARY KEY` |

**Validation:**

When `auto` is **not** specified, the field must contain a valid format:

```parsley
@schema Reference {
    id: ulid              // Not auto - must provide valid ULID
}

// Valid ULID format
Reference({id: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}).validate().isValid()  // true

// Invalid format
Reference({id: "not-a-ulid"}).validate().isValid()  // false (FORMAT error)
```

### The `readOnly` Constraint

The `readOnly` constraint marks fields that cannot be set from client/form input. These fields are silently filtered out during record creation and updates, preventing privilege escalation attacks (e.g., a user setting their own `role` to `"admin"`).

```parsley
@schema User {
    name: string(required)
    email: email(required)
    role: enum["user", "admin"](readOnly, default: "user")
    isVerified: boolean(readOnly, default: false)
}

// Client tries to set role to "admin" - silently filtered
let user = User({name: "Alice", email: "a@b.com", role: "admin"})
user.role                            // "user" (default applied, input ignored)

// Update also filters readOnly fields
user.update({role: "admin"})         // role unchanged, still "user"

// readOnly fields are still readable
user.role                            // "user"
```

**Behavior:**

| Context | `readOnly` field behavior |
|---------|--------------------------|
| `Schema({...})` | Input filtered, default or null applied |
| `record.update()` | Input filtered (silently ignored) |
| `record.fieldName` | Readable (no restriction) |
| Display/forms | Visible (use `hidden` metadata to hide) |

**Multi-Schema Pattern:**

The `readOnly` constraint is enforced at the schema level, not the database level. This enables different schemas for different security contexts:

```parsley
// Public schema - role cannot be set by users
@schema User {
    name: string
    role: enum["user", "admin"](readOnly, default: "user")
}

// Admin schema - role can be set (no readOnly)
@schema AdminUser {
    name: string
    role: enum["user", "admin"](default: "user")
}

// Same table, different bindings
let Users = db.bind(User, "users")         // Public endpoints
let AdminUsers = db.bind(AdminUser, "users") // Admin endpoints

// Public: role filtered
let user = User({name: "Alice", role: "admin"})
user.role  // "user"

// Admin: role accepted
let admin = AdminUser({name: "Alice", role: "admin"})
admin.role  // "admin"
```

**Use cases:**
- `role` — Prevent users from escalating their own privileges
- `isVerified` — Server sets after email verification
- `isBanned` — Server sets via admin action
- `createdBy` — Server sets from session user

> **⚠️ Important: `readOnly` and Delete/Update Operations**
>
> When you mark `id` as `readOnly`, it gets filtered to `null` when creating a Record from form data. This means you **cannot** use that record for delete or update operations (which require the primary key).
>
> ```parsley
> @schema Person {
>     id: int(auto, readOnly)  // readOnly filters form input!
>     name: string
> }
>
> // ❌ This will FAIL:
> let person = Person(formData)    // id is null (filtered)
> People.delete(person)            // Error: no primary key value
>
> // ✅ Solution 1: Pass the ID directly
> People.delete(formData.id)       // Works!
>
> // ✅ Solution 2: Load from database first
> let person = People.find(formData.id)
> People.delete(person)            // Works - has real ID from DB
> ```
>
> The `readOnly` constraint is designed to prevent clients from *setting* protected fields, not from *using* them. For delete/update operations, use the ID from the request parameters or load the record from the database.

---

### The `pattern` Constraint

The `pattern` constraint validates string fields against a regular expression:

```parsley
@schema User {
    name: string(pattern: /^[A-Za-z\s\-']+$/)
    username: string(min: 3, max: 20, pattern: /^[a-z][a-z0-9_]*$/)
    slug: string(pattern: /^[a-z0-9]+(?:-[a-z0-9]+)*$/)
}

// Valid
User({name: "Alice O'Brien", username: "alice_123", slug: "hello-world"})
    .validate().isValid()  // true

// Invalid - pattern mismatch
User({username: "Alice"}).validate().errorCode("username")  // "PATTERN"
```

**Important:** Empty strings **pass** pattern validation. Use `min: 1` or `required` for non-empty:

```parsley
@schema Profile {
    // Empty string passes pattern
    slug: string(pattern: /^[a-z0-9-]+$/)
    
    // Combine with min for non-empty + pattern
    username: string(min: 1, pattern: /^[a-z0-9_]+$/)
}

Profile({slug: ""}).validate().isValid()      // true (empty passes)
Profile({username: ""}).validate().isValid()  // false (fails MIN_LENGTH first)
```

**Behavior:**

| Input | Pattern | Result |
|-------|---------|--------|
| `""` | Any | ✅ Valid (empty passes) |
| `"Alice"` | `/^[A-Z][a-z]+$/` | ✅ Valid |
| `"alice"` | `/^[A-Z][a-z]+$/` | ❌ PATTERN error |
| `"Hello!"` | `/^[A-Za-z]+$/` | ❌ PATTERN error |

**HTML Form Integration:**

When using `@field` for form binding, the pattern is converted to a JavaScript-compatible regex and applied as an HTML `pattern` attribute:

```parsley
@schema Contact {
    phone: string(pattern: /^\+?[0-9\s\-]+$/)
}

@field phone: Contact.phone  // <input pattern="^\+?[0-9\s\-]+$" ...>
```

---

## Field Metadata (Pipe Syntax)

Add UI metadata using the pipe `|` syntax followed by a dictionary:

```parsley
@schema Contact {
    name: string | {title: "Full Name", placeholder: "Enter your name"}
    email: email | {title: "Email Address"}
    phone: phone? | {title: "Phone Number", placeholder: "(555) 555-5555"}
    notes: text? | {title: "Additional Notes", hidden: true}
}
```

### Common Metadata Keys

| Key | Type | Description |
|-----|------|-------------|
| `title` | `string` | Display label for forms and table headers |
| `placeholder` | `string` | Input placeholder text |
| `hidden` | `boolean` | Exclude from auto-generated UIs |
| `currency` | `string` | Currency code for `money` fields (e.g., "USD", "EUR") |

### Custom Metadata

You can add any metadata keys you need:

```parsley
@schema Product {
    price: money | {
        title: "Price",
        currency: "USD",
        step: 0.01,
        helpText: "Enter the retail price"
    }
}

// Access custom metadata
Product.meta("price", "currency")   // "USD"
Product.meta("price", "step")       // 0.01
Product.meta("price", "helpText")   // "Enter the retail price"
```

### Currency Metadata for Money Fields

When a `money` field has `currency` metadata, `record.format()` uses it for locale-aware formatting:

```parsley
@schema Product {
    price: money | {currency: "USD"}
    cost: money | {currency: "EUR"}
    fee: money | {currency: "JPY"}
}

let p = Product({price: 1999, cost: 1500, fee: 5000})

p.format("price")  // "$ 1,999.00" (USD)
p.format("cost")   // "€ 1,500.00" (EUR)
p.format("fee")    // "¥ 5,000" (JPY, no decimals)
```

Without `currency` metadata, `format()` uses the default locale currency symbol.

---

## Attributes

### name

Returns the schema's declared name.

```parsley
@schema Customer { name: string }

Customer.name                   // "Customer"
```

### fields

Returns a dictionary of all field definitions with their type information.

```parsley
@schema User {
    name: string
    age: integer = 0
}

User.fields
// {
//   name: {name: "name", type: "string", required: true, nullable: false},
//   age: {name: "age", type: "integer", required: true, nullable: false, default: "0"}
// }
```

### [fieldName] (Direct Field Access)

Access a field name directly to get its type:

```parsley
@schema Person {
    name: string
    age: integer
    email: email
}

Person.name                     // "string"
Person.age                      // "integer"
Person.email                    // "email"
```

---

## Methods

### title()

#### Usage: title(field)

Returns the display title for a field. If the field has a `title` in its metadata, that value is returned. Otherwise, the field name is converted to title case (e.g., `firstName` → `"First Name"`).

```parsley
@schema Contact {
    firstName: string | {title: "First Name"}
    lastName: string
    emailAddress: string
}

Contact.title("firstName")      // "First Name" (from metadata)
Contact.title("lastName")       // "Last Name" (auto title-cased)
Contact.title("emailAddress")   // "Email Address" (auto title-cased)
```

This is useful for generating form labels and table headers:

```parsley
for (field in schema.visibleFields()) {
    <label>{schema.title(field)}</label>
}
```

---

### placeholder()

#### Usage: placeholder(field)

Returns the placeholder text for a field, or `null` if not set in metadata.

```parsley
@schema Login {
    email: email | {placeholder: "you@example.com"}
    password: string | {placeholder: "••••••••"}
    remember: boolean
}

Login.placeholder("email")      // "you@example.com"
Login.placeholder("password")   // "••••••••"
Login.placeholder("remember")   // null (no placeholder set)
```

```parsley
// Use with null coalescing for default
<input placeholder={schema.placeholder(field) ?? ""}/>
```

---

### meta()

#### Usage: meta(field, key)

Returns any metadata value for a field, or `null` if the key doesn't exist. This allows access to custom metadata beyond the standard `title`, `placeholder`, and `hidden`.

```parsley
@schema Settings {
    theme: string | {
        title: "Theme",
        options: ["light", "dark", "auto"],
        default: "auto"
    }
    fontSize: integer | {
        title: "Font Size",
        min: 8,
        max: 32,
        unit: "px"
    }
}

Settings.meta("theme", "options")       // ["light", "dark", "auto"]
Settings.meta("fontSize", "unit")       // "px"
Settings.meta("fontSize", "min")        // 8
Settings.meta("theme", "nonexistent")   // null
```

---

### fields()

#### Usage: fields()

Returns an array of all field names in declaration order.

```parsley
@schema Person {
    name: string
    age: integer
    city: string
}

Person.fields()                 // ["name", "age", "city"]
```

---

### visibleFields()

#### Usage: visibleFields()

Returns an array of field names excluding:
- Fields with `hidden: true` metadata
- Fields with `auto` constraint (e.g., `id: ulid(auto)`)

This is the key method for auto-generating forms and tables that respect hidden and auto-generated fields.

```parsley
@schema User {
    id: ulid(auto)                          // excluded (auto)
    name: string
    email: email
    passwordHash: string | {hidden: true}   // excluded (hidden)
    createdAt: datetime | {hidden: true}    // excluded (hidden)
}

User.fields()                   // ["id", "name", "email", "passwordHash", "createdAt"]
User.visibleFields()            // ["name", "email"]
```

**Common use case**: Generate a table showing only user-facing columns:

```parsley
<table>
    <thead>
        <tr>
            for (col in User.visibleFields()) {
                <th>{User.title(col)}</th>
            }
        </tr>
    </thead>
    // ...
</table>
```

---

### enumValues()

#### Usage: enumValues(field)

Returns the allowed values for an enum field as an array, or an empty array if the field is not an enum type.

```parsley
@schema Issue {
    title: string
    priority: enum["low", "medium", "high", "critical"]
    status: enum["open", "in-progress", "resolved", "closed"]
    description: text
}

Issue.enumValues("priority")    // ["low", "medium", "high", "critical"]
Issue.enumValues("status")      // ["open", "in-progress", "resolved", "closed"]
Issue.enumValues("title")       // [] (not an enum)
Issue.enumValues("description") // [] (not an enum)
```

**Common use case**: Generate a `<select>` dropdown:

```parsley
<select name="priority">
    for (value in Issue.enumValues("priority")) {
        <option value={value}>{value.toTitleCase()}</option>
    }
</select>
```

---

## Using Schemas with Tables

### Creating Typed Tables

Use `@table(Schema)` to create a table bound to a schema:

```parsley
@schema Product {
    name: string
    price: money
    inStock: boolean = true
}

let products = @table(Product) [
    {name: "Widget", price: $9.99},           // inStock defaults to true
    {name: "Gadget", price: $19.99, inStock: false}
]
```

### Accessing Table Schema

Tables remember their schema via the `.schema` property:

```parsley
products.schema                 // The Product schema
products.schema.title("name")   // "Name"
products.schema.visibleFields() // ["name", "price", "inStock"]
```

### Schema-Aware Row Iteration

When you access `.rows` on a typed table, the dictionary preserves the schema's field order:

```parsley
@schema Person {
    firstName: string
    lastName: string
    age: integer
}

let people = @table(Person) [
    {firstName: "Alice", lastName: "Smith", age: 30}
]

// Keys iterate in declaration order, not alphabetically
for (key, value in people.rows[0]) {
    log(key)  // firstName, lastName, age (not age, firstName, lastName)
}
```

---

## Using Schemas with Database Bindings

Schemas power Parsley's database table bindings:

```parsley
@schema User {
    id: integer
    name: string(unique: true)
    email: email(unique: true)
    createdAt: datetime = @now
}

let db = @sqlite("./app.db")

// Bind schema to database table (creates table if needed)
let Users = db.bind(User, "users")

// Now use typed queries
let all = Users.all()
let alice = Users.where({name: "Alice"}).first()
let newUser = Users.create({name: "Bob", email: "bob@example.com"})
```

The schema:
- Generates `CREATE TABLE` SQL with proper types
- Applies defaults on insert
- Validates data types
- Adds constraints (UNIQUE, NOT NULL)

### Primary Key Convention

The field named `id` is automatically treated as the primary key. This convention enables schema-driven mutations (insert, update, save, delete with Record/Table arguments) to identify rows for update and delete operations.

```parsley
@schema Product {
    id: integer        // Automatically marked as primary key
    name: string
    price: money
}
```

### Schema-Driven Mutations

Bound tables support method-based CRUD operations that accept Record or Table objects directly:

```parsley
let db = @sqlite(":memory:")
db.createTable(User, "users")
let users = db.bind(User, "users")

// Insert a Record (id auto-generated)
let user = User({name: "Alice", email: "alice@example.com"})
let inserted = users.insert(user)
log(inserted.id)  // Generated ID

// Update using Record
let updated = users.update(user.update({name: "Alice Smith"}))

// Save (upsert) - inserts if new, updates if exists
let saved = users.save(User({id: inserted.id, name: "Alice Jones"}))

// Delete by Record
users.delete(user)
```

#### Bound Table Mutation Methods

| Method | Argument | Returns | Description |
|--------|----------|---------|-------------|
| `insert(record)` | Record | Record | Insert single row, return with generated ID |
| `insert(table)` | Table | `{inserted: N}` | Insert all rows, return count |
| `update(record)` | Record | Record | Update row by ID |
| `update(table)` | Table | `{updated: N}` | Update all rows by ID |
| `save(record)` | Record | Record | Upsert single row |
| `save(table)` | Table | `{inserted: N, updated: M}` | Upsert all rows |
| `delete(record)` | Record | `{deleted: 1}` | Delete row by ID |
| `delete(table)` | Table | `{deleted: N}` | Delete all rows by ID |
| `delete(id)` | String/Int | `{deleted: 1}` | Delete row by ID value |

> **Note:** The `delete(record)` form requires the record to have a non-null primary key (`id` field). If you're constructing records from form data where `id` is marked `readOnly`, the ID will be filtered to `null`. In this case, use `delete(id)` directly:
>
> ```parsley
> // When id is readOnly, pass ID directly instead of a record:
> People.delete(params.id)         // ✅ Works
> People.delete(Person(formData))  // ❌ Fails - id is null
> ```

#### Schema Matching

When a Record or Table has an attached schema, it must match the binding's schema:

```parsley
@schema Product { id: integer, name: string }
let product = Product({name: "Widget"})

// Error VAL-0022: Schema mismatch
users.insert(product)  // User binding can't accept Product
```

Plain dictionaries can always be inserted (backward compatible):

```parsley
users.insert({name: "Bob", email: "bob@example.com"})  // Works
```

---

## Practical Examples

### Dynamic Form Generation

```parsley
@schema ContactForm {
    name: string | {title: "Your Name", placeholder: "John Doe"}
    email: email | {title: "Email Address", placeholder: "john@example.com"}
    subject: enum("General", "Support", "Sales") | {title: "Subject"}
    message: text | {title: "Message", placeholder: "How can we help?"}
}

let FormField = fn(schema, field) {
    let type = schema[field]
    let isEnum = schema.enumValues(field).length() > 0
    
    <div class="form-group">
        <label for={field}>{schema.title(field)}</label>
        if (isEnum) {
            <select name={field} id={field}>
                for (opt in schema.enumValues(field)) {
                    <option value={opt}>{opt}</option>
                }
            </select>
        } else if (type == "text") {
            <textarea 
                name={field} 
                id={field}
                placeholder={schema.placeholder(field) ?? ""}
            />
        } else {
            <input 
                type={if (type == "email") "email" else "text"}
                name={field} 
                id={field}
                placeholder={schema.placeholder(field) ?? ""}
            />
        }
    </div>
}

<form method="POST">
    for (field in ContactForm.visibleFields()) {
        FormField(ContactForm, field)
    }
    <button type="submit">Send</button>
</form>
```

### Sortable Table Component

```parsley
@schema Employee {
    id: integer | {hidden: true}
    name: string | {title: "Employee Name"}
    department: string | {title: "Department"}
    salary: money | {title: "Salary"}
    hireDate: date | {title: "Hire Date"}
}

let SortableTable = fn(table) {
    let schema = table.schema
    check schema else "<p>Table has no schema</p>"
    
    <table class="sortable">
        <thead>
            <tr>
                for (col in schema.visibleFields()) {
                    <th data-sort={col}>{schema.title(col)}</th>
                }
            </tr>
        </thead>
        <tbody>
            for (row in table.rows) {
                <tr>
                    for (col in schema.visibleFields()) {
                        <td>{row[col]}</td>
                    }
                </tr>
            }
        </tbody>
    </table>
}

let employees = @table(Employee) [
    {id: 1, name: "Alice", department: "Engineering", salary: $95000, hireDate: @2020-03-15},
    {id: 2, name: "Bob", department: "Sales", salary: $75000, hireDate: @2021-07-01}
]

SortableTable(employees)
```

---

## See Also

- [Tables](./table.md) — Using schemas with tables
- [Records](./record.md) — Schema-bound dictionaries with validation
- [@std/table](../stdlib/table.md) — SQL-like data manipulation with schema validation
- [@std/valid](../stdlib/valid.md) — Validation predicates for values
- [Data Model](../fundamentals/data-model.md) — Schemas, records, and tables overview
- [Query DSL](../features/query-dsl.md) — Schema-driven database queries
- [Database](../features/database.md) — Database connections and table bindings
