---
id: man-pars-data-model
title: Data Model
system: parsley
type: fundamentals
name: data-model
created: 2026-02-06
version: 0.2.0
author: Basil Team
keywords:
  - schema
  - record
  - table
  - data model
  - validation
  - database
  - form binding
  - is operator
  - table binding
---

# Data Model

Parsley's structured data system is built on three connected concepts: **Schema** defines the shape, **Record** holds validated data, and **Table** organizes rows. This page explains how the pieces fit together. See the individual reference pages for full API details.

## The Pipeline

```
Schema  →  Record  →  Table
(shape)    (data)     (rows)
```

1. A **Schema** declares fields, types, constraints, and metadata.
2. A **Record** binds a schema to actual data and tracks validation errors.
3. A **Table** is an ordered collection of rows — optionally typed by a schema.

Each layer builds on the previous one. You can use dictionaries and arrays for simple cases, but schemas, records, and tables add validation, type safety, and database integration.

## Schema

A schema defines the shape of your data — field names, types, constraints, and UI metadata:

```parsley
@schema User {
    id: integer
    name: string(min: 2, required)
    email: email(required, unique: true)
    role: enum["user", "admin"] = "user"
}
```

Schemas are values — you can assign them, pass them to functions, and use them at runtime. They drive:

- **Validation** — what values are acceptable
- **Database tables** — column types and constraints
- **Form generation** — input types, labels, placeholders, and autocomplete
- **Type checking** — the `is` operator tests schema conformance

See [Schemas](../builtins/schema.md) for the full field type and constraint reference.

## Record

A Record is a Schema + Data + Errors. Create one by calling the schema as a function:

```parsley
let user = User({name: "Alice", email: "alice@example.com"})
```

Records behave like dictionaries — you access fields with dot notation — but they carry their schema and validation state:

```parsley
user.name                        // "Alice"
user.role                        // "user" (default applied)
user.errors()                    // {} (no errors)
user.valid()                     // true
```

Invalid data is accepted but tracked:

```parsley
let bad = User({name: "", email: "not-an-email"})
bad.valid()                      // false
bad.errors()                     // {name: "...", email: "..."}
```

Records are **immutable** — updating returns a new record:

```parsley
let updated = user.set("name", "Bob")
updated.name                     // "Bob"
user.name                        // "Alice" (unchanged)
```

See [Records](../builtins/record.md) for methods, form binding, and serialization.

## Dictionary vs Record

| | Dictionary | Record |
|---|---|---|
| Schema | None | Required |
| Validation | None | Automatic |
| Errors | None | Tracked per field |
| Database | Manual | Schema-driven |
| Form binding | Manual | `@field` attributes |
| Type | `dictionary` | `record` |

Use dictionaries for ad-hoc data (config, API responses, temporary structures). Use records when you need validation, database mapping, or form binding.

## Table

A Table is an ordered collection of rows with named columns. Create one from a literal, CSV, or database query:

```parsley
// From a literal
let t = @table [
    {name: "Alice", age: 30},
    {name: "Bob", age: 25}
]

// From CSV
let sales <== CSV(@./sales.csv)

// From a database query
let users <== @DB.query("SELECT * FROM users")
```

Tables provide SQL-like query methods that return new tables (immutable chaining):

```parsley
let result = t
    .where(fn(r) { r.age > 20 })
    .orderBy("name")
    .select("name")
```

### Typed Tables

When a table has a schema, rows are Records instead of plain dictionaries:

```parsley
let users = User.table()         // table bound to User schema
let row = users[0]               // a Record, not a dictionary
row is User                      // true
```

See [Tables](../builtins/table.md) for query methods, aggregation, and output formats.

## Schema Identity — the `is` Operator

The `is` operator checks whether a record conforms to a specific schema:

```parsley
let user = User({name: "Alice", email: "alice@example.com"})
user is User                     // true
user is Product                  // false
```

This is a schema identity check, not a structural/duck-typing check. A plain dictionary with the same keys would not pass `is User` — it must be a Record created from that schema.

## Table Bindings

A table binding connects a schema to a database, enabling CRUD operations:

```parsley
let users = User.table()         // in-memory table
let dbUsers = @DB.table(User)    // database-backed table
```

Database-backed tables support:

- **Insert** — `record ==> dbUsers`
- **Query** — `dbUsers.where(...)`, `dbUsers.find(id)`
- **Update** — `updatedRecord ==> dbUsers`
- **Delete** — `dbUsers.delete(id)`

Records from database tables are auto-validated against their schema.

## The Lifecycle

A typical data flow in a Basil web application:

1. **Define** a schema: `@schema User { ... }`
2. **Create** a record from form input: `let user = User(formData)`
3. **Validate**: `user.valid()` — check before saving
4. **Persist**: `user ==> @DB.table(User)` — write to database
5. **Query**: `let users <== @DB.table(User).where(...)` — read back
6. **Render**: `<form @record={user}>` — bind to HTML form

Each step uses the schema as the single source of truth for field names, types, constraints, and UI metadata.

## Key Differences from Other Languages

- **Schema is a runtime value** — not a compile-time type annotation. You can pass schemas to functions, store them in variables, and introspect them at runtime.
- **Validation is built in** — no separate validation library. The schema defines constraints; the record tracks errors automatically.
- **Records are immutable** — `.set()` returns a new record. No in-place mutation of validated data.
- **Tables are query-able** — `.where()`, `.orderBy()`, `.select()`, `.groupBy()` work on any table, not just database results. CSV data gets the same query API as SQL results.
- **No ORM** — schemas map directly to database columns. There's no object-relational mapping layer, no migrations framework, and no lazy loading. The schema *is* the model.

## See Also

- [Schemas](../builtins/schema.md) — field types, constraints, metadata, and schema methods
- [Records](../builtins/record.md) — record methods, form binding, serialization
- [Tables](../builtins/table.md) — query methods, aggregation, output formats
- [Type System](types.md) — overview of all Parsley types
- [Tags](tags.md) — form binding with `@record` and `@field`
- [Database](../features/database.md) — database connections and operations