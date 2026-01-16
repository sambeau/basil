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
    role: enum("user", "admin") = "user"
    active: boolean = true
    createdAt: datetime | {hidden: true}
}

// Use with tables
let users = @table(User) [
    {id: 1, name: "Alice", email: "alice@example.com", createdAt: @now},
    {id: 2, name: "Bob", email: "bob@example.com", role: "admin", createdAt: @now}
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
| `uuid` | UUID strings | `TEXT` | UUID format |
| `ulid` | ULID strings | `TEXT` | ULID format |
| `json` | JSON data | `TEXT` | Valid JSON |
| `email` | Email addresses | `TEXT` | Email pattern |
| `url` | URLs | `TEXT` | URL pattern |
| `phone` | Phone numbers | `TEXT` | Phone pattern |
| `slug` | URL slugs | `TEXT` | Slug pattern |
| `enum(...)` | Enumerated values | `TEXT` | In list |

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

Define allowed values inline with `enum(...)`:

```parsley
@schema Task {
    title: string
    priority: enum("low", "medium", "high") = "medium"
    status: enum("todo", "in-progress", "blocked", "done")
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

Returns an array of field names where the `hidden` metadata is not `true`. This is the key method for auto-generating forms and tables that respect hidden fields.

```parsley
@schema User {
    id: integer | {hidden: true}
    name: string
    email: email
    passwordHash: string | {hidden: true}
    createdAt: datetime | {hidden: true}
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
    priority: enum("low", "medium", "high", "critical")
    status: enum("open", "in-progress", "resolved", "closed")
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
- [@std/schema](../std/schema.md) — Runtime schema validation
- [Database Bindings](../../guide/query-dsl.md) — Schema-driven database access
