# Schema + Table Binding Design

**Status:** Design exploration  
**Date:** December 2025

---

## The Goal

Useful composition without hidden magic. Schema types bring validation/sanitization/docs. Table bindings provide query helpers. Everything is explicit and inspectable.

---

## Part 1: Types with Behavior

Types bring their own validation, sanitization, and documentation:

```javascript
let schema = import("std/schema")

// Types bring their own validation, sanitization, docs
let User = schema.define("User", {
    name: schema.string({required: true, min: 1, max: 100}),
    email: schema.email({required: true}),
    role: schema.enum({values: ["user", "admin"], default: "user"})
})

// Use for validation (no magic, explicit)
let {value, errors} = schema.validate(User, formData)
let clean = schema.sanitize(User, formData)
```

---

## Part 2: The Question—How to Fetch Data?

### Option 1: Raw SQL (what we have now)
```javascript
let users = basil.sqlite.query("SELECT * FROM users WHERE role = ?", [role])
```
Explicit, flexible, but no schema connection.

### Option 2: ORM-style magic
```javascript
let users = User.findAll({where: {role: "admin"}})
```
Convenient, but where did `findAll` come from? Magic.

### Option 3: Something in between?

---

## Part 3: Schema-Aware Table Binding

Bind a schema to a table—returns an object with query methods:

```javascript
let schema = import("std/schema")

let User = schema.define("User", {
    name: schema.string({required: true}),
    email: schema.email({required: true}),
    role: schema.enum({values: ["user", "admin"]})
})

// Bind schema to a table - returns query helpers
let Users = schema.table(User, basil.sqlite, "users")

// Users is now an object with query methods
// BUT they're just conveniences over basil.sqlite.query()

Users.all()                        // SELECT * FROM users
Users.find(id)                     // SELECT * FROM users WHERE id = ?
Users.where({role: "admin"})       // SELECT * FROM users WHERE role = ?
Users.insert({name: "Alice", ...}) // INSERT INTO users ...
Users.update(id, {name: "Bob"})    // UPDATE users SET name = ? WHERE id = ?
Users.delete(id)                   // DELETE FROM users WHERE id = ?
```

**The "magic" is visible:** `schema.table()` creates an object with methods. Those methods generate and execute SQL. You could write the SQL yourself—this is just a shortcut.

---

## Part 4: What schema.table() Actually Returns

```javascript
// schema.table() returns something like:
{
    all: fn() { 
        basil.sqlite.query("SELECT * FROM users") 
    },
    find: fn(id) { 
        basil.sqlite.query("SELECT * FROM users WHERE id = ?", [id])[0] 
    },
    where: fn(conditions) {
        let {sql, params} = buildWhere(conditions)
        basil.sqlite.query("SELECT * FROM users WHERE " + sql, params)
    },
    insert: fn(data) {
        let clean = schema.sanitize(User, data)
        let {errors} = schema.validate(User, clean)
        if errors { return {error: errors} }
        // ... build INSERT SQL ...
        basil.sqlite.exec(sql, params)
        {value: {...clean, id: lastInsertId()}}
    },
    // etc.
}
```

**It's just functions.** The schema provides validation/sanitization, the table binding provides SQL generation. No hidden runtime.

---

## Part 5: Composing Queries

Composition via object spread + custom functions:

```javascript
// Base table binding
let Users = schema.table(User, basil.sqlite, "users")

// Add custom queries by extending the object
let UserQueries = {
    ...Users,
    
    admins: fn() {
        Users.where({role: "admin"})
    },
    
    byEmail: fn(email) {
        Users.where({email: email})[0]
    },
    
    withOrders: fn(userId) {
        let user = Users.find(userId)
        if user {
            user.orders = Orders.where({userId: userId})
        }
        user
    },
    
    recentlyActive: fn(days) {
        basil.sqlite.query(
            "SELECT * FROM users WHERE last_login > datetime('now', ?)",
            ["-{days} days"]
        )
    }
}

// Use them
let admins = UserQueries.admins()
let user = UserQueries.withOrders("123")
```

**No inheritance, no class hierarchy, just objects and functions.**

---

## Part 6: Using in Handlers

```javascript
let schema = import("std/schema")

let User = schema.define("User", {
    name: schema.string({required: true}),
    email: schema.email({required: true}),
    role: schema.enum({values: ["user", "admin"]})
})

let Users = schema.table(User, basil.sqlite, "users")

get "/users" {
    let users = Users.all()
    
    <ul>
        {users.map(fn(u) { <li>{u.name} ({u.email})</li> })}
    </ul>
}

get "/users/:id" {
    let user = Users.find(params.id)
    
    if !user {
        <div class=error>User not found</div>
    } else {
        <div class=user>
            <h1>{user.name}</h1>
            <p>{user.email}</p>
        </div>
    }
}

post "/users" {
    let result = Users.insert(body)
    
    if result.error {
        <div class=error>
            {result.error.map(fn(e) { <p>{e.field}: {e.message}</p> })}
        </div>
    } else {
        redirect("/users/" + result.value.id)
    }
}
```

---

## Part 7: The Non-Magic Contract

| What | How | Magic Level |
|------|-----|-------------|
| Type validation | Schema functions return `{value, errors}` | None |
| Type sanitization | Schema functions transform input | None |
| SQL generation | Query builder creates SQL strings | Low - inspectable |
| Query execution | `basil.sqlite.query()` | None - explicit |
| Table binding | `schema.table()` returns object with query methods | Low - it's just functions |
| Custom queries | Add functions to the object | None |

---

## Part 8: External Database Support

```javascript
// Local SQLite (default)
let Users = schema.table(User, basil.sqlite, "users")

// External Postgres (future)
let pg = db.connect("postgres://...")
let Users = schema.table(User, pg, "users")

// The interface is the same - just different connection
```

---

## Part 9: The Repository Pattern

`schema.table()` creates a **repository object**—a thing that knows how to fetch/store a particular type of data:

```javascript
let Users = schema.table(User, basil.sqlite, "users")

// Users IS the object that knows how to grab data
// It combines:
// - Schema (validation, sanitization)
// - Connection (basil.sqlite)  
// - Table name ("users")
// - Generated query methods
```

---

## Summary

```javascript
// 1. Define schema (types bring validation, sanitization, docs)
let User = schema.define("User", {
    name: schema.string({required: true}),
    email: schema.email({required: true})
})

// 2. Bind to table (creates query helpers)
let Users = schema.table(User, basil.sqlite, "users")

// 3. Use in handlers (explicit data fetching)
get "/users" {
    let users = Users.all()
    <ul>{users.map(fn(u) { <li>{u.name}</li> })}</ul>
}

// 4. Extend with custom queries (composition)
let UserQueries = {
    ...Users,
    admins: fn() { Users.where({role: "admin"}) }
}
```

**No magic, just composition.** The schema provides validation. The table binding provides query methods. You call them explicitly. You can always drop down to raw SQL. You can extend with custom functions.

---

## Key Properties

1. **Explicit execution** — You call the query methods. Nothing runs automatically.

2. **Inspectable** — `schema.table()` returns a plain object. You can log it, extend it, wrap it.

3. **Composable** — Object spread + custom functions. No special syntax.

4. **Escapable** — Raw SQL is always available via `basil.sqlite.query()`.

5. **Schema-connected** — Validation and sanitization happen on insert/update.

6. **Database-agnostic** — Same interface for SQLite, Postgres, etc.
