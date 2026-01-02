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

## Schema Library

The `@std/schema` module provides declarative data validation through schema definitions. Define your data shape once, then use it for form validation, API input checking, and database table bindings. Schemas specify field types, constraints, and whether fields are required—validation returns detailed error information for user feedback.

```parsley
let schema = import @std/schema

let UserSchema = schema.define("User", {
    email: schema.email({required: true}),
    name: schema.string({min: 1, max: 100}),
    age: schema.integer({min: 0, max: 150})
})

let result = UserSchema.validate({
    email: "alice@example.com",
    name: "Alice",
    age: 30
})

if (result.valid) {
    <p>"User data is valid!"</p>
} else {
    <ul>
        for (e in result.errors) {
            <li>e.message</li>
        }
    </ul>
}
```

## Philosophy

**"Define once, validate everywhere."**

The schema module follows these principles:

- **Declarative definitions** — Describe what valid data looks like, not how to check it
- **Rich type system** — Built-in types for common formats (email, URL, phone, dates)
- **Composable constraints** — Add validation rules through options
- **Actionable errors** — Each error includes field name, error code, and message
- **Database integration** — Bind schemas to tables for type-safe CRUD operations

### When to Use @std/schema vs @std/valid

| Use Case | Module |
|----------|--------|
| Validate form submission | `@std/schema` |
| Define data models | `@std/schema` |
| Bind to database tables | `@std/schema` |
| One-off format checks | `@std/valid` |
| Custom validation logic | `@std/valid` |

---

## Type Factories

Type factory functions create field specifications with optional constraints.

### string()

Text field with optional length and pattern constraints.

```parsley
let schema = import @std/schema

// Basic string
schema.string()

// With constraints
schema.string({required: true, min: 1, max: 255})

// With regex pattern
schema.string({pattern: "^[A-Z][a-z]+$"})
```

| Option | Type | Description |
|--------|------|-------------|
| `required` | boolean | Field must be present and non-empty |
| `min` | integer | Minimum character length |
| `max` | integer | Maximum character length |
| `pattern` | string | Regex pattern the value must match |

---

### email()

Email address with format validation.

```parsley
let schema = import @std/schema

schema.email()                           // Optional email
schema.email({required: true})           // Required email
```

Validates against standard email format: `user@domain.tld`

| Option | Type | Description |
|--------|------|-------------|
| `required` | boolean | Field must be present and non-empty |

---

### url()

URL with format validation. Requires `http://` or `https://` protocol.

```parsley
let schema = import @std/schema

schema.url()                             // Optional URL
schema.url({required: true})             // Required URL
```

| Option | Type | Description |
|--------|------|-------------|
| `required` | boolean | Field must be present and non-empty |

---

### phone()

Phone number with basic format validation. Accepts digits, spaces, plus signs, dashes, parentheses, and periods.

```parsley
let schema = import @std/schema

schema.phone()                           // Accepts "+1 (555) 123-4567"
schema.phone({required: true})
```

| Option | Type | Description |
|--------|------|-------------|
| `required` | boolean | Field must be present and non-empty |

---

### integer()

Whole number with optional range constraints.

```parsley
let schema = import @std/schema

schema.integer()                         // Any integer
schema.integer({min: 0})                 // Non-negative
schema.integer({min: 1, max: 100})       // Between 1 and 100
schema.integer({required: true, min: 0, max: 150})
```

| Option | Type | Description |
|--------|------|-------------|
| `required` | boolean | Field must be present |
| `min` | integer | Minimum value (inclusive) |
| `max` | integer | Maximum value (inclusive) |

---

### number()

Any numeric value (integer or float).

```parsley
let schema = import @std/schema

schema.number()
schema.number({required: true})
```

| Option | Type | Description |
|--------|------|-------------|
| `required` | boolean | Field must be present |

---

### boolean()

Boolean true/false value.

```parsley
let schema = import @std/schema

schema.boolean()
schema.boolean({default: false})
```

| Option | Type | Description |
|--------|------|-------------|
| `default` | boolean | Default value if not provided |

---

### enum()

Value must be one of a predefined set.

```parsley
let schema = import @std/schema

// Pass allowed values as arguments
schema.enum("draft", "published", "archived")

// Or as options dict with values array
schema.enum({values: ["small", "medium", "large"], required: true})
```

| Option | Type | Description |
|--------|------|-------------|
| `values` | array | Allowed values |
| `required` | boolean | Field must be present |
| `default` | any | Default value if not provided |

---

### date()

Date string in ISO format (YYYY-MM-DD).

```parsley
let schema = import @std/schema

schema.date()                            // Validates "2025-12-25"
schema.date({required: true})
```

| Option | Type | Description |
|--------|------|-------------|
| `required` | boolean | Field must be present |

---

### datetime()

Datetime value.

```parsley
let schema = import @std/schema

schema.datetime()
schema.datetime({required: true})
```

| Option | Type | Description |
|--------|------|-------------|
| `required` | boolean | Field must be present |

---

### money()

Monetary value with optional currency specification.

```parsley
let schema = import @std/schema

schema.money()
schema.money({currency: "USD", required: true})
```

| Option | Type | Description |
|--------|------|-------------|
| `required` | boolean | Field must be present |
| `currency` | string | Currency code (e.g., "USD", "EUR") |

---

### id()

Unique identifier with format validation. Defaults to ULID format.

```parsley
let schema = import @std/schema

schema.id()                              // ULID format (default)
schema.id({format: "uuid"})              // UUID v4 format
schema.id({format: "uuidv7"})            // UUID v7 format
```

Supported formats:
- `"ulid"` — 26-character sortable ID (default)
- `"uuid"` or `"uuidv4"` — Standard UUID v4
- `"uuidv7"` — Time-sortable UUID v7

| Option | Type | Description |
|--------|------|-------------|
| `format` | string | ID format: "ulid", "uuid", "uuidv4", "uuidv7" |

---

### array()

Array/list value with optional element type.

```parsley
let schema = import @std/schema

schema.array()
schema.array({of: schema.string(), min: 1})
```

| Option | Type | Description |
|--------|------|-------------|
| `of` | type spec | Type of array elements |
| `min` | integer | Minimum number of elements |
| `max` | integer | Maximum number of elements |

---

### object()

Nested object with its own field definitions.

```parsley
let schema = import @std/schema

schema.object({
    properties: {
        street: schema.string(),
        city: schema.string({required: true}),
        zip: schema.string({pattern: "^\\d{5}$"})
    }
})
```

| Option | Type | Description |
|--------|------|-------------|
| `properties` | dict | Field definitions for nested object |

---

## Schema Operations

### define()

Creates a named schema definition from field specifications.

#### Usage: define(name, fields)

```parsley
let schema = import @std/schema

let ContactSchema = schema.define("Contact", {
    id: schema.id(),
    email: schema.email({required: true}),
    name: schema.string({required: true, min: 1, max: 100}),
    phone: schema.phone(),
    role: schema.enum("user", "admin", "moderator"),
    active: schema.boolean({default: true}),
    createdAt: schema.datetime()
})
```

The returned schema object has:
- `name` — The schema name
- `fields` — The field definitions

---

### validate()

Validates data against the schema. Returns a result object with validation status and any errors.

#### Usage: Schema.validate(data)

```parsley
let schema = import @std/schema

let UserSchema = schema.define("User", {
    email: schema.email({required: true}),
    age: schema.integer({min: 0})
})

// Valid data
let result = UserSchema.validate({
    email: "test@example.com",
    age: 25
})
result.valid  // true
result.errors // []

// Invalid data
let badResult = UserSchema.validate({
    email: "not-an-email",
    age: -5
})
badResult.valid  // false
badResult.errors // Array of error objects
```

#### Validation Result

The result object contains:

| Field | Type | Description |
|-------|------|-------------|
| `valid` | boolean | `true` if all validations passed |
| `errors` | array | List of error objects (empty if valid) |

Each error object contains:

| Field | Type | Description |
|-------|------|-------------|
| `field` | string | Name of the field that failed |
| `code` | string | Error code (see below) |
| `message` | string | Human-readable error message |

#### Error Codes

| Code | Description |
|------|-------------|
| `REQUIRED` | Required field is missing or empty |
| `TYPE` | Value is not the expected type |
| `FORMAT` | Value doesn't match expected format (email, URL, etc.) |
| `MIN_LENGTH` | String is shorter than minimum |
| `MAX_LENGTH` | String is longer than maximum |
| `MIN_VALUE` | Number is less than minimum |
| `MAX_VALUE` | Number is greater than maximum |
| `PATTERN` | String doesn't match regex pattern |
| `ENUM` | Value is not in the allowed set |

---

### table()

Binds a schema to a database table for type-safe CRUD operations. Returns a TableBinding helper object that wraps common database operations with automatic validation, ID generation, and pagination.

#### Usage: table(schema, db, tableName)

```parsley
let schema = import @std/schema

let db = @sqlite("app.db")
let _ = db <=!=> "CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    age INTEGER
)"

let UserSchema = schema.define("User", {
    id: schema.id(),
    name: schema.string({required: true}),
    email: schema.email({required: true}),
    age: schema.integer({min: 0})
})

let Users = schema.table(UserSchema, db, "users")
```

The `table()` function takes three arguments:

| Argument | Type | Description |
|----------|------|-------------|
| `schema` | schema | A schema created with `schema.define()` |
| `db` | database | A database connection (`@sqlite`, `@DB`, etc.) |
| `tableName` | string | Name of the database table |

**Important:** The table must already exist in the database. The schema defines the validation rules, not the table structure.

---

## TableBinding In-Depth Guide

The TableBinding returned by `schema.table()` provides a Rails-like Active Record pattern for database operations. It combines schema validation with database CRUD operations, making it easy to build type-safe data access layers.

### Design Philosophy

TableBinding follows these principles:

- **Schema-first** — All data is validated against the schema before database operations
- **Convention over configuration** — Assumes `id` column for primary key, uses `?` placeholders
- **Fail fast** — Returns validation errors before attempting database writes
- **Transparent pagination** — Reads `limit`/`offset` from request query automatically
- **SQL injection prevention** — Column names are validated against identifier pattern

### Database Support

TableBinding currently supports **SQLite** databases only:

```parsley
// ✅ Supported
let db = @sqlite("app.db")
let Users = schema.table(UserSchema, db, "users")

// ✅ Also supported - Basil's built-in database
let db = @DB
let Users = schema.table(UserSchema, db, "users")

// ❌ Not yet supported
let db = @postgres("postgresql://...")
let db = @mysql("mysql://...")
```

### Using with @DB (Basil's Built-in Database)

`@DB` is Basil's built-in SQLite database, available only within Basil server handlers. It's the simplest way to add persistence to your Basil application:

```parsley
let schema = import @std/schema

// Define schema once (typically at module level)
let PostSchema = schema.define("Post", {
    id: schema.id(),
    title: schema.string({required: true, max: 200}),
    content: schema.string({required: true}),
    published: schema.boolean({default: false})
})

// Create table if needed (run once at startup or via a setup script)
// handlers/setup.pars
let _ = @DB <=!=> "CREATE TABLE IF NOT EXISTS posts (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    published INTEGER DEFAULT 0
)"
<p>"Database initialized"</p>
```

```parsley
// handlers/posts/index.pars - handles GET /posts
let schema = import @std/schema

let PostSchema = schema.define("Post", {
    id: schema.id(),
    title: schema.string({required: true, max: 200}),
    content: schema.string({required: true}),
    published: schema.boolean({default: false})
})

let Posts = schema.table(PostSchema, @DB, "posts")

if (basil.request.method == "POST") {
    // Handle POST /posts
    let post = Posts.insert({
        title: basil.body.title,
        content: basil.body.content
    })
    
    if (post.valid == false) {
        status(400)
        let err = post.errors | first
        <p class="error">err.message</p>
    } else {
        redirect("/posts/" ++ post.id)
    }
} else {
    // Handle GET /posts
    let posts = Posts.all()
    
    <ul>
        for (p in posts) {
            <li>p.title</li>
        }
    </ul>
}
```

**Note:** `@DB` is only available inside Basil handlers. It returns an error if used in standalone Parsley scripts.

### Using with @sqlite (External Database)

For external SQLite databases or standalone Parsley scripts, use `@sqlite`:

```parsley
let schema = import @std/schema

let db = @sqlite("./data/myapp.db")

let UserSchema = schema.define("User", {
    id: schema.id(),
    email: schema.email({required: true}),
    name: schema.string({required: true})
})

let Users = schema.table(UserSchema, db, "users")
```

---

## TableBinding Methods

### insert(data)

Validates and inserts a new record. Auto-generates `id` if the schema has an `id` field and none is provided.

```parsley
let user = Users.insert({
    name: "Alice",
    email: "alice@example.com",
    age: 30
})

// Returns the inserted record with generated id:
// {id: "01HGW...", name: "Alice", email: "alice@example.com", age: 30}
```

#### Validation Failures

If validation fails, `insert()` returns a validation result instead of the inserted record:

```parsley
let result = Users.insert({age: 30})  // missing required name and email

if (result.valid == false) {
    // Handle validation errors
    for (error in result.errors) {
        log(`{error.field}: {error.message}`)
    }
}
```

#### Auto-Generated IDs

When your schema includes an `id` field with `schema.id()`, the TableBinding automatically generates unique IDs:

```parsley
let UserSchema = schema.define("User", {
    id: schema.id(),                    // ULID by default
    // id: schema.id({format: "uuid"}), // Or UUID v4
    // id: schema.id({format: "uuidv7"}), // Or time-sortable UUID v7
    name: schema.string({required: true})
})

let user = Users.insert({name: "Alice"})
// id is auto-generated: "01HGW2N5KBPZ4QX8Y7MV3JD6TF"
```

You can also provide your own ID:

```parsley
let user = Users.insert({
    id: "custom-id-123",
    name: "Alice"
})
```

---

### find(id)

Retrieves a single record by ID. Returns `null` if not found.

```parsley
let user = Users.find("01HGW...")
// Returns: {id: "01HGW...", name: "Alice", email: "alice@example.com", age: 30}
// Or: null if not found
```

Common pattern - handle not found:

```parsley
// handlers/users/[id].pars - handles GET /users/:id
let user = Users.find(basil.params.id)
if (user == null) {
    status(404)
    <h1>"User not found"</h1>
} else {
    <h1>user.name</h1>
}
```

---

### all()

Retrieves all records with automatic pagination.

```parsley
let users = Users.all()
// Returns: [{id: ..., name: ..., ...}, ...]
```

#### Automatic Pagination

When used in a Basil handler, `all()` automatically reads pagination from the request query string:

| Query Parameter | Default | Max | Description |
|-----------------|---------|-----|-------------|
| `limit` | 20 | 100 | Number of records to return |
| `offset` | 0 | — | Number of records to skip |

```parsley
// handlers/users/index.pars
// Request: GET /users?limit=10&offset=20
let users = Users.all()  // Returns records 21-30
// ...
```

To disable pagination, set `limit=0` (or negative):

```parsley
// Request: GET /users?limit=0
// Returns ALL records (use with caution)
```

**Note:** `where()` does NOT apply automatic pagination—it returns all matching records.

---

### where(conditions)

Retrieves records matching the given conditions. All conditions are combined with AND.

```parsley
// Single condition
let admins = Users.where({role: "admin"})

// Multiple conditions (AND)
let activeAdmins = Users.where({role: "admin", active: true})

// Empty conditions returns all records
let everyone = Users.where({})
```

#### Security: SQL Injection Prevention

Column names are validated against a strict identifier pattern (`^[A-Za-z_][A-Za-z0-9_]*$`). Invalid column names return an error:

```parsley
// ❌ This will fail with an error
Users.where({"name; DROP TABLE users": "x"})
// Error: invalid column name

// ✅ Safe - values are parameterized
Users.where({name: "Robert'); DROP TABLE users;--"})
// Executes: SELECT * FROM users WHERE name = ?
// With parameter: "Robert'); DROP TABLE users;--"
```

---

### update(id, data)

Updates an existing record by ID. Validates the data before updating.

```parsley
let updated = Users.update("01HGW...", {
    name: "Alice Smith",
    age: 31
})

// Returns the updated record:
// {id: "01HGW...", name: "Alice Smith", email: "alice@example.com", age: 31}
```

#### Restrictions

- Cannot change the `id` field (returns an error if attempted)
- Must provide at least one field to update
- Validates data against schema before updating

```parsley
// ❌ Cannot change ID
let result = Users.update("01HGW...", {id: "new-id"})
// Error: cannot change id in update

// ❌ Validation failures return error result
let result = Users.update("01HGW...", {email: "not-an-email"})
if (result.valid == false) {
    // Handle validation error
}
```

---

### delete(id)

Deletes a record by ID. Returns an object with the number of affected rows.

```parsley
let result = Users.delete("01HGW...")
result.affected  // 1 if deleted, 0 if not found
```

Common pattern - handle DELETE request:

```parsley
// handlers/users/[id].pars - handles DELETE /users/:id
// (Configure route method in basil.yaml or check basil.request.method)
let result = Users.delete(basil.params.id)
if (result.affected > 0) {
    redirect("/users")
} else {
    status(404)
    <p>"User not found"</p>
}
```

---

## Extending and Customizing TableBinding

TableBinding provides common CRUD operations, but real applications often need custom queries. Here's how to extend its functionality.

### Custom Queries with Raw SQL

For complex queries, use the database connection directly alongside TableBinding:

```parsley
let schema = import @std/schema

let db = @sqlite("app.db")
let UserSchema = schema.define("User", {
    id: schema.id(),
    name: schema.string({required: true}),
    email: schema.email({required: true}),
    created_at: schema.datetime()
})

let Users = schema.table(UserSchema, db, "users")

// Standard CRUD through TableBinding
let user = Users.find(id)
let all = Users.all()

// Custom query using raw SQL
let recentUsers = db <=??=> {
    sql: "SELECT * FROM users WHERE created_at > ? ORDER BY created_at DESC LIMIT 10",
    params: [@today - @7d]
}

// Complex aggregation
let stats = db <=?=> {
    sql: "SELECT COUNT(*) as total, AVG(age) as avg_age FROM users"
}
```

### Wrapper Functions for Reusable Queries

Create wrapper functions for frequently used queries:

```parsley
let schema = import @std/schema

let db = @sqlite("app.db")
let PostSchema = schema.define("Post", {
    id: schema.id(),
    title: schema.string({required: true}),
    author_id: schema.string({required: true}),
    status: schema.enum("draft", "published", "archived"),
    published_at: schema.datetime()
})

let Posts = schema.table(PostSchema, db, "posts")

// Wrapper functions for common queries
let findPublished = fn() {
    Posts.where({status: "published"})
}

let findByAuthor = fn(authorId) {
    Posts.where({author_id: authorId})
}

let findRecent = fn(limit) {
    db <=??=> {
        sql: "SELECT * FROM posts WHERE status = 'published' ORDER BY published_at DESC LIMIT ?",
        params: [limit]
    }
}

let publish = fn(postId) {
    Posts.update(postId, {
        status: "published",
        published_at: @now
    })
}
```

Usage in handler files:

```parsley
// handlers/posts/index.pars - GET /posts
let posts = findPublished()
// ...
```

```parsley
// handlers/authors/[id]/posts.pars - GET /authors/:id/posts
let posts = findByAuthor(basil.params.id)
// ...
```

```parsley
// handlers/posts/[id]/publish.pars - POST /posts/:id/publish
let post = publish(basil.params.id)
redirect("/posts/" ++ post.id)
```

### Combining Multiple Tables

For relationships, combine multiple TableBindings:

```parsley
let schema = import @std/schema

let db = @sqlite("app.db")

// Schemas
let UserSchema = schema.define("User", {
    id: schema.id(),
    name: schema.string({required: true}),
    email: schema.email({required: true})
})

let PostSchema = schema.define("Post", {
    id: schema.id(),
    title: schema.string({required: true}),
    content: schema.string({required: true}),
    author_id: schema.string({required: true})
})

let CommentSchema = schema.define("Comment", {
    id: schema.id(),
    post_id: schema.string({required: true}),
    author_id: schema.string({required: true}),
    content: schema.string({required: true})
})

// TableBindings
let Users = schema.table(UserSchema, db, "users")
let Posts = schema.table(PostSchema, db, "posts")
let Comments = schema.table(CommentSchema, db, "comments")

// Load post with author
let loadPostWithAuthor = fn(postId) {
    let post = Posts.find(postId)
    if (post == null) { null }
    else {
        let author = Users.find(post.author_id)
        {
            ...post,
            author: author
        }
    }
}

// Load post with comments
let loadPostWithComments = fn(postId) {
    let post = Posts.find(postId)
    if (post == null) { null }
    else {
        let comments = Comments.where({post_id: postId})
        {
            ...post,
            comments: comments
        }
    }
}

// Usage in a handler file:
// handlers/posts/[id].pars - GET /posts/:id
let post = loadPostWithAuthor(basil.params.id)
if (post == null) {
    status(404)
    <h1>"Not found"</h1>
} else {
    <article>
        <h1>post.title</h1>
        <p class="author">`By {post.author.name}`</p>
        <div>post.content</div>
    </article>
}
```

### Overriding Insert Behavior

Add custom logic around insert operations:

```parsley
let schema = import @std/schema

let db = @sqlite("app.db")
let UserSchema = schema.define("User", {
    id: schema.id(),
    email: schema.email({required: true}),
    name: schema.string({required: true}),
    created_at: schema.datetime(),
    updated_at: schema.datetime()
})

let UsersTable = schema.table(UserSchema, db, "users")

// Custom insert with timestamps
let createUser = fn(data) {
    UsersTable.insert({
        ...data,
        created_at: @now,
        updated_at: @now
    })
}

// Custom update with timestamp
let updateUser = fn(id, data) {
    UsersTable.update(id, {
        ...data,
        updated_at: @now
    })
}

// Check uniqueness before insert
let createUniqueUser = fn(data) {
    let existing = UsersTable.where({email: data.email})
    if (existing.length() > 0) {
        {
            valid: false,
            errors: [{field: "email", code: "UNIQUE", message: "Email already exists"}]
        }
    } else {
        createUser(data)
    }
}

// Usage in a handler file:
// handlers/users/index.pars - POST /users
let result = createUniqueUser({
    email: basil.body.email,
    name: basil.body.name
})

if (result.valid == false) {
    status(400)
    let err = result.errors | first
    <p class="error">err.message</p>
} else {
    redirect("/users/" ++ result.id)
}
```

### Soft Deletes

Implement soft delete pattern:

```parsley
let schema = import @std/schema

let db = @sqlite("app.db")
let PostSchema = schema.define("Post", {
    id: schema.id(),
    title: schema.string({required: true}),
    deleted_at: schema.datetime()
})

let PostsTable = schema.table(PostSchema, db, "posts")

// Soft delete
let deletePost = fn(id) {
    PostsTable.update(id, {deleted_at: @now})
}

// Restore
let restorePost = fn(id) {
    // Note: need raw SQL to set null
    db <=!=> {
        sql: "UPDATE posts SET deleted_at = NULL WHERE id = ?",
        params: [id]
    }
}

// Query only non-deleted
let activePosts = fn() {
    db <=??=> {
        sql: "SELECT * FROM posts WHERE deleted_at IS NULL ORDER BY id DESC"
    }
}

// Query with deleted
let allPostsIncludingDeleted = fn() {
    PostsTable.all()
}
```

---

## Best Practices

### 1. Define Schemas at Module Level

```parsley
// ✅ Good: Define once at the top of your handler file
let schema = import @std/schema

let UserSchema = schema.define("User", {
    id: schema.id(),
    email: schema.email({required: true}),
    name: schema.string({required: true})
})

// Then use in the handler logic below...
```

### 2. Create TableBinding Inside Handlers (for @DB)

```parsley
// ✅ Good: @DB is available in Basil handler files
// handlers/users/index.pars
let Users = schema.table(UserSchema, @DB, "users")
let users = Users.all()
// ...
```

```parsley
// ❌ Bad: @DB not available in standalone Parsley scripts
// my-script.pars (run with `pars my-script.pars`)
let Users = schema.table(UserSchema, @DB, "users")  // Error!
```

**Why can't @DB be used at module level?**

`@DB` uses a **cached, shared database connection** — the same `*sql.DB` is reused across all requests for efficiency. However, `@DB` still requires handler context because:

1. **The `basil` context doesn't exist yet** — At module load time (before any request), Basil hasn't injected the `basil` object containing `basil.sqlite`
2. **`@DB` looks up `basil.sqlite`** — This lookup fails outside handlers because there's no `basil` context to look in

Think of it this way:
- The **database connection** is cached at server startup
- The **reference to that connection** (`@DB`) is only available inside handlers

```
Server starts
    └── Opens SQLite database (cached, shared *sql.DB)
    
Module loads
    └── @DB fails — no basil context exists yet
    
Request arrives
    └── Basil injects basil context (including basil.sqlite)
        └── Handler runs
            └── @DB succeeds — finds basil.sqlite in context
            └── Returns the SAME cached database connection
```

**Workaround for module-level TableBinding:**

Use `@sqlite` with an explicit path instead:

```parsley
// ✅ @sqlite works in standalone scripts (explicit connection)
let schema = import @std/schema
let db = @sqlite("./data/app.db")
let Users = schema.table(UserSchema, db, "users")

Users.all()  // Works!
```

### 3. Handle Validation Errors

```parsley
// ✅ Good: Always check for validation errors
let result = Users.insert(body)
if (result.valid == false) {
    // Handle error
} else {
    // Success - result is the inserted record
}
```

### 4. Use Type-Safe IDs

```parsley
// ✅ Good: Let schema generate IDs
let UserSchema = schema.define("User", {
    id: schema.id(),  // Auto-generated ULID
    // ...
})

// ✅ Good: Use sortable IDs for time-ordered data
let EventSchema = schema.define("Event", {
    id: schema.id({format: "uuidv7"}),  // Time-sortable
    // ...
})
```

### 5. Validate Column Names in Schema Match Table

```parsley
// Schema field names must match database column names
let UserSchema = schema.define("User", {
    id: schema.id(),
    email: schema.email(),      // Must match column "email"
    created_at: schema.datetime() // Must match column "created_at"
})

// Database table
// CREATE TABLE users (id TEXT, email TEXT, created_at TEXT)
```

---

## Complete Examples

### Form Validation

```parsley
let schema = import @std/schema
let {Form, TextField, Button} = import @std/html

let ContactSchema = schema.define("Contact", {
    name: schema.string({required: true, min: 1, max: 100}),
    email: schema.email({required: true}),
    message: schema.string({required: true, min: 10, max: 1000})
})

// handlers/contact.pars - handles POST /contact
let result = ContactSchema.validate(basil.body)

if (!result.valid) {
    <Form action="/contact" method="POST">
        for (e in result.errors) {
            <p class="error">`{e.field}: {e.message}`</p>
        }
        <TextField name="name" label="Name" value={basil.body.name}/>
        <TextField name="email" label="Email" type="email" value={basil.body.email}/>
        <TextField name="message" label="Message" value={basil.body.message}/>
        <Button type="submit">"Send"</Button>
    </Form>
} else {
    // Process valid submission...
    <p>"Thank you for your message!"</p>
}
```

### Database CRUD

First, set up your database schema (run once):

```parsley
// setup.pars - run with: pars setup.pars
let db = @sqlite("blog.db")
let _ = db <=!=> "CREATE TABLE IF NOT EXISTS posts (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    status TEXT DEFAULT 'draft',
    created_at TEXT
)"
```

Then create handler files for your routes:

```parsley
// handlers/posts/index.pars - handles /posts
let schema = import @std/schema

let PostSchema = schema.define("Post", {
    id: schema.id(),
    title: schema.string({required: true, min: 1, max: 200}),
    content: schema.string({required: true}),
    status: schema.enum("draft", "published", "archived"),
    created_at: schema.datetime()
})

let Posts = schema.table(PostSchema, @DB, "posts")

if (basil.request.method == "POST") {
    // Create new post
    let post = Posts.insert({
        title: basil.body.title,
        content: basil.body.content,
        status: "draft",
        created_at: @now
    })
    
    if (post.valid == false) {
        redirect("/posts/new?error=validation")
    } else {
        redirect("/posts/" ++ post.id)
    }
} else {
    // List with filtering
    let posts = if (basil.query.status) {
        Posts.where({status: basil.query.status})
    } else {
        Posts.all()
    }
    
    <ul>
        for (p in posts) {
            <li><a href={"/posts/" ++ p.id}>p.title</a></li>
        }
    </ul>
}
```

```parsley
// handlers/posts/[id].pars - handles /posts/:id
let schema = import @std/schema
// ... (same schema definition)
let Posts = schema.table(PostSchema, @DB, "posts")

if (basil.request.method == "DELETE") {
    // Delete post
    let result = Posts.delete(basil.params.id)
    if (result.affected > 0) {
        redirect("/posts")
    } else {
        status(404)
        <p>"Post not found"</p>
    }
} else {
    // Read single post
    let post = Posts.find(basil.params.id)
    if (post == null) {
        status(404)
        <h1>"Post not found"</h1>
    } else {
        <article>
            <h1>post.title</h1>
            <div>post.content</div>
        </article>
    }
}
```

### Nested Object Validation

```parsley
let schema = import @std/schema

let AddressSchema = schema.object({
    properties: {
        street: schema.string({required: true}),
        city: schema.string({required: true}),
        state: schema.string({min: 2, max: 2}),
        zip: schema.string({pattern: "^\\d{5}(-\\d{4})?$"})
    }
})

let OrderSchema = schema.define("Order", {
    id: schema.id(),
    customer_email: schema.email({required: true}),
    shipping_address: AddressSchema,
    items: schema.array({min: 1})
})
```

---

## See Also

- [@std/valid](valid.md) — Individual validation functions for custom logic
- [@std/html](html.md) — Form components with built-in validation display
- [@std/id](id.md) — ID generation functions
