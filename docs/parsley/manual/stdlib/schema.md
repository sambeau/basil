---
id: man-pars-std-schema
title: "Std/Schema"
system: parsley
type: stdlib
name: "@std/schema"
created: 2026-01-01
version: 0.2.0
author: "@copilot"
keywords: schema, validation, types, forms, database, table, binding, constraints
---

## ⚠️ DEPRECATED — Use `@schema` DSL Instead

`@std/schema` is **deprecated**. Use the built-in `@schema { ... }` DSL syntax, which is shorter, declarative, and integrates directly with the query DSL.

**`@std/schema` still works** but will not receive new features. All new code should use `@schema`.

---

## Before / After

```parsley
// ❌ Old — @std/schema (deprecated)
let schema = import @std/schema

let UserSchema = schema.define("User", {
    id: schema.id(),
    name: schema.string({required: true, min: 1, max: 100}),
    email: schema.email({required: true}),
    age: schema.integer({min: 0, max: 150}),
    role: schema.enum("user", "admin"),
    active: schema.boolean({default: true})
})

let Users = schema.table(UserSchema, db, "users")
```

```parsley
// ✅ New — @schema DSL
@schema User {
    id: id(auto)
    name: string(required, min: 1, max: 100)
    email: email(required, unique: true)
    age: integer(min: 0, max: 150)
    role: enum["user", "admin"] = "user"
    active: boolean = true
}

let Users = db.bind(User, "users")
```

---

## Migration Reference

### Type Factories → DSL Types

| `@std/schema` (old)              | `@schema` DSL (new)          |
|----------------------------------|-------------------------------|
| `schema.string({...})`          | `string` / `string(...)`     |
| `schema.email({...})`           | `email` / `email(...)`       |
| `schema.url({...})`             | `url` / `url(...)`           |
| `schema.phone({...})`           | `phone` / `phone(...)`       |
| `schema.integer({...})`         | `integer` / `int` / `int(...)` |
| `schema.number({...})`          | `float` / `number`           |
| `schema.boolean({...})`         | `boolean` / `bool`           |
| `schema.enum("a", "b")`        | `enum["a", "b"]`             |
| `schema.date({...})`            | `date`                       |
| `schema.datetime({...})`        | `datetime`                   |
| `schema.money({...})`           | `money`                      |
| `schema.id()`                   | `id(auto)`                   |
| `schema.array({...})`           | *(use typed arrays)*         |
| `schema.object({...})`          | *(use nested schemas)*       |

### Operations → DSL Equivalents

| `@std/schema` (old)                         | `@schema` DSL (new)                     |
|----------------------------------------------|------------------------------------------|
| `schema.define("User", { ... })`            | `@schema User { ... }`                  |
| `schema.table(UserSchema, db, "users")`     | `db.bind(User, "users")`               |
| `UserSchema.validate(data)`                 | `User(data).validate()`                 |
| `{required: true}` option                   | `(required)` constraint or non-`?` type |
| `{default: value}` option                   | `= value` after the type               |

### New DSL Features (no @std/schema equivalent)

- **Nullable fields:** `name: string?`
- **Field metadata:** `name: string | {title: "Full Name", placeholder: "..."}`
- **Auto fields:** `id: id(auto)`, `createdAt: datetime(auto)`
- **Unit types:** `weight: mass`, `height: length`
- **`db.createTable(User, "users")`** — create tables directly from schemas

---

## Documentation

- **`@schema` DSL reference:** [Schema Literals](../../reference.md#115-schema-literals) in the Parsley Language Reference
- **Database integration:** [Query DSL Guide](../../../guide/query-dsl.md)

## See Also

- [@std/valid](valid.md) — Individual validation functions for custom logic
- [@std/html](html.md) — Form components with built-in validation display
- [@std/id](id.md) — ID generation functions