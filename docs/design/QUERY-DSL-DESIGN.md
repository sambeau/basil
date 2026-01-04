# Query DSL Design: @objects with Mini-Grammars

**Date:** 2026-01-03  
**Status:** Draft  
**Related:** FEAT-078 (TableBinding API), QUERY-BUILDER-INVESTIGATION.md

## Overview

This document proposes a domain-specific language (DSL) for database operations using Parsley's @object syntax. The key insight is that @objects already establish a pattern where content inside has *its own parsing rules* (e.g., `@date(2026-01-03)`, `@path(/users/{id})`). 

We extend this pattern to schemas and queries, creating a concise, readable syntax for the 90% of common database operations.

## Design Principles

1. **Simplicity** — Very few keywords, easy to remember
2. **Clarity** — Visual operators show data flow direction
3. **Not SQL** — Inspired by SQL but not a mirror of it
4. **Parsley-native** — Use Parsley syntax where code-like syntax is needed (`==` not `=`)
5. **90% coverage** — Handle common cases elegantly; escape to raw SQL for edge cases
6. **Safe by default** — All interpolation is parameterized, SQL-injection-safe

## Connection to Existing Features

### Existing SQL Operators

Parsley already has database operators:

```parsley
db <=?=>  "SELECT * FROM users WHERE id = {id}"    // Single row
db <=??=> "SELECT * FROM users"                    // Multiple rows
db <=!=>  "INSERT INTO users (name) VALUES ({n})"  // Execute (no result)
```

### The New DSL Mirrors These

| Raw SQL | DSL Operator | Returns |
|---------|--------------|---------|
| `<=?=>` | `?->` | Single row/value |
| `<=??=>` | `??->` | Multiple rows |
| `<=!=>` | `X` | Nothing (execute) |
| — | `X->` | Count (after execute) |

This creates a consistent mental model across raw SQL and the DSL.

### Relationship to FEAT-078

FEAT-078 defines the method-based TableBinding API:

```parsley
let Users = schema.table(UserSchema, db, "users")
Users.find(1)
Users.where({status: "active"})
Users.all({orderBy: "name", limit: 10})
```

The DSL proposed here is an **alternative syntax**, not a replacement. Both can coexist:

- **FEAT-078 methods**: Good for simple operations, programmatic building
- **@query DSL**: Good for complex queries, visual clarity, static analysis

They compile to the same underlying operations.

---

## Part 1: Schema Declarations (`@schema`)

### Current Syntax (Library Calls)

```parsley
let UserSchema = schema.define("User", {
  id: schema.int(),
  name: schema.string(),
  email: schema.string(),
})
```

### Proposed Syntax (Grammar)

```parsley
@schema User {
  id: int
  name: string
  email: string
}
```

### With Relations

For schemas with mutual references, forward declaration is implicit:

```parsley
@schema User {
  id: int
  name: string
  email: string
  posts: [Post] via user_id      // Has-many
  profile: Profile via user_id   // Has-one
}

@schema Post {
  id: int
  title: string
  body: string
  user_id: int
  author: User via user_id       // Belongs-to
  comments: [Comment] via post_id
}

@schema Comment {
  id: int
  body: string
  post_id: int
  user_id: int
  post: Post via post_id
  author: User via user_id
}
```

### Binding Schemas to Tables

The binding step connects schemas to a database:

```parsley
let Users = db.bind(User, "users")
let Posts = db.bind(Post, "posts")
let Comments = db.bind(Comment, "comments")
```

**Key insight:** Schemas are pure types (no database knowledge). Bindings add the database mapping. This separation allows:
- Same schema, different databases
- Same schema, different table names
- Schemas used for validation without any database

---

## Part 2: Query Operations

### Structure

All query operations follow a consistent structure:

```
@operation(
  Source
  | conditions
  | modifiers
  |< writes
  terminal
)
```

### Pipe Operators

| Operator | Purpose | Example |
|----------|---------|---------|
| `|` | Condition (filter) | `| status == "active"` |
| `|<` | Write (set values) | `|< name: "Alice"` |
| `?->` | Return single | `?-> *` or `?-> id, name` |
| `??->` | Return multiple | `??-> *` or `??-> id, name` |
| `X` | Execute, no return | `X` |
| `X->` | Execute, return count | `X-> count` |

### Visual Flow

```
@query(Posts | status == "published" | order created_at desc ??-> id, title)
       ───┬─   ─────────┬──────────   ─────────┬─────────── ────────┬──────
       Source       Condition              Modifier              Output
```

---

## Part 3: CRUD Operations

### Query (Read)

```parsley
// All published posts
@query(
  Posts
  | status == "published"
  | order created_at desc
  | limit 10
  ??-> id, title, created_at
)

// Single post by ID
@query(
  Posts
  | id == {postId}
  | with author, comments
  ?-> *
)

// Count
@query(
  Posts
  | status == "published"
  ?-> count
)

// Exists check
@query(
  Users
  | email == {email}
  ?-> exists
)
```

### Insert (Create)

```parsley
// Insert, no return
@insert(
  Posts
  |< title: {form.title},
  |< body: {form.body},
  |< user_id: {user.id},
  |< status: "draft"
  X
)

// Insert, return created row
let post = @insert(
  Posts
  |< title: {form.title},
  |< body: {form.body},
  |< user_id: {user.id}
  ?-> *
)

// Insert, return just ID
let id = @insert(
  Posts
  |< title: "Hello", body: "World"
  ?-> id
)
```

### Update (Modify)

```parsley
// Update, no return
@update(
  Posts
  | id == {postId}
  |< status: "published",
  |< published_at: {now}
  X
)

// Update, return count
let count = @update(
  Posts
  | status == "draft" and updated_at < {cutoff}
  |< status: "archived"
  X-> count
)

// Update, return modified rows
let updated = @update(
  Users
  | last_login < {oneYearAgo}
  |< status: "inactive"
  ??-> *
)

// Update single, return it
let user = @update(
  Users
  | id == {userId}
  |< last_login: {now}
  ?-> *
)
```

### Delete (Remove)

```parsley
// Delete, no return
@delete(
  Posts
  | id == {postId}
  X
)

// Delete, return count
let purged = @delete(
  Sessions
  | expires_at < {now}
  X-> count
)

// Delete, return removed rows (for audit)
let removed = @delete(
  Users
  | status == "banned"
  ??-> *
)
```

---

## Part 4: Condition Syntax

### Operators

Use Parsley operators, not SQL:

| DSL | SQL | Meaning |
|-----|-----|---------|
| `==` | `=` | Equal |
| `!=` | `<>` | Not equal |
| `>` | `>` | Greater than |
| `<` | `<` | Less than |
| `>=` | `>=` | Greater or equal |
| `<=` | `<=` | Less or equal |
| `in` | `IN` | In list |
| `not in` | `NOT IN` | Not in list |
| `like` | `LIKE` | Pattern match |
| `between X and Y` | `BETWEEN` | Range |
| `is null` | `IS NULL` | Null check |
| `is not null` | `IS NOT NULL` | Not null check |

### Logical Operators

```parsley
// AND
| status == "active" and age >= 18

// OR
| status == "featured" or views > 1000

// NOT
| not status == "draft"

// Grouping
| (status == "active" and age >= 18) or role == "admin"
```

### Examples

```parsley
// Simple equality
@query(Posts | status == "published" ??-> *)

// Multiple conditions (implicit AND with comma)
@query(Posts | status == "published", views > 100 ??-> *)

// Explicit AND
@query(Posts | status == "published" and views > 100 ??-> *)

// OR
@query(Posts | status == "featured" or views > 1000 ??-> *)

// IN list
@query(Posts | status in ["published", "featured"] ??-> *)

// NOT IN
@query(Posts | category not in ["spam", "archived"] ??-> *)

// LIKE
@query(Users | email like "%@example.com" ??-> *)

// BETWEEN
@query(Products | price between 10 and 50 ??-> *)

// NULL checks
@query(Posts | deleted_at is null ??-> *)
@query(Users | avatar_url is not null ??-> *)

// Complex
@query(
  Users
  | (status == "active" and age >= 18) or role == "admin"
  | created_at >= {lastMonth}
  ??-> *
)
```

---

## Part 5: Query Modifiers

### Ordering

```parsley
| order created_at desc
| order name asc
| order name                    // asc is default
| order category asc, created_at desc   // multiple
```

### Pagination

```parsley
| limit 10
| limit 10 offset 20
```

### Relations (Includes)

```parsley
// Simple include
| with author

// Multiple includes
| with author, comments

// Nested includes (dot notation)
| with author, comments.author

// Include with conditions
| with comments(approved == true)

// Include with ordering
| with comments(order created_at desc)

// Include with limit
| with comments(limit 5)

// Combined
| with comments(approved == true | order created_at desc | limit 5)
```

---

## Part 6: Output Projection

### Selecting Fields

```parsley
// All fields
??-> *

// Specific fields
??-> id, title, created_at

// With aliases (if needed)
??-> id, title, created_at as date

// Nested fields (from includes)
??-> id, title, author.name
```

### Special Returns

```parsley
// Count
?-> count

// Exists (boolean)
?-> exists

// First row
?-> *       // on @query, returns single

// All rows
??-> *      // on @query, returns array
```

---

## Part 7: Interpolation

### Syntax

Parsley expressions are interpolated with `{expression}`:

```parsley
@query(
  Posts
  | user_id == {currentUser.id}
  | created_at >= {date.subtract(date.now(), {days: 7})}
  ??-> *
)
```

### Safety

All interpolated values are parameterized, never string-concatenated:

```parsley
// This is SAFE - {name} becomes a parameter
@query(Users | name == {userInput} ?-> *)

// Compiles to: SELECT * FROM users WHERE name = $1
// With params: [$userInput]
```

### What Can Be Interpolated

| Context | Allowed |
|---------|---------|
| Values | `{expr}` — any Parsley expression |
| Lists | `{arrayExpr}` — Parsley array |
| Field names | No — must be static |
| Table names | No — must be static |

This restriction enables static analysis and prevents SQL injection.

---

## Part 8: Full Examples

### Blog Post Lifecycle

```parsley
// Create draft
let post = @insert(
  Posts
  |< title: {form.title},
  |< body: {form.body},
  |< user_id: {currentUser.id},
  |< status: "draft"
  ?-> *
)

// Read for editing
let draft = @query(
  Posts
  | id == {postId} and user_id == {currentUser.id}
  | with author
  ?-> *
)

// Update content
@update(
  Posts
  | id == {postId}
  |< title: {form.title},
  |< body: {form.body},
  |< updated_at: {now}
  X
)

// Publish
let published = @update(
  Posts
  | id == {postId}
  |< status: "published",
  |< published_at: {now}
  ?-> *
)

// List published posts
let posts = @query(
  Posts
  | status == "published"
  | order published_at desc
  | limit 10
  | with author
  ??-> id, title, excerpt, published_at, author.name
)

// Delete
@delete(
  Posts
  | id == {postId}
  X
)
```

### E-commerce Order

```parsley
// Create order
let order = @insert(
  Orders
  |< user_id: {user.id},
  |< status: "pending",
  |< total: {cart.total}
  ?-> *
)

// Add order items
for item in cart.items {
  @insert(
    OrderItems
    |< order_id: {order.id},
    |< product_id: {item.product_id},
    |< quantity: {item.quantity},
    |< price: {item.price}
    X
  )
}

// Update stock
@update(
  Products
  | id in {cart.productIds}
  |< quantity: quantity - 1
  X
)

// Process order
@update(
  Orders
  | id == {order.id}
  |< status: "processing",
  |< processed_at: {now}
  X
)
```

### Admin Cleanup Jobs

```parsley
// Archive old drafts
let archived = @update(
  Posts
  | status == "draft" and updated_at < {sixMonthsAgo}
  |< status: "archived"
  X-> count
)

// Purge expired sessions
let purged = @delete(
  Sessions
  | expires_at < {now}
  X-> count
)

// Deactivate inactive users
let deactivated = @update(
  Users
  | last_login < {oneYearAgo} and status == "active"
  |< status: "inactive",
  |< deactivated_at: {now}
  ??-> id, email
)
```

---

## Part 9: Grammar Summary

### @schema

```
@schema Name {
  field: type
  field: type
  relation: Type via foreign_key
  relation: [Type] via foreign_key
}
```

### @query

```
@query(
  Binding
  | conditions
  | order field (asc|desc)
  | limit N (offset M)?
  | with relations
  (?-> | ??->) projection
)
```

### @insert

```
@insert(
  Binding
  |< field: value,
  |< field: value
  (X | ?-> projection)
)
```

### @update

```
@update(
  Binding
  | conditions
  |< field: value,
  |< field: value
  (X | X-> count | ?-> projection | ??-> projection)
)
```

### @delete

```
@delete(
  Binding
  | conditions
  (X | X-> count | ??-> projection)
)
```

---

## Part 10: Comparison

### DSL vs FEAT-078 Methods

```parsley
// FEAT-078 method style
let posts = Posts.where({status: "published"}, {
  orderBy: "created_at",
  order: "desc",
  limit: 10,
  select: ["id", "title"],
  include: ["author"]
})

// DSL style
let posts = @query(
  Posts
  | status == "published"
  | order created_at desc
  | limit 10
  | with author
  ??-> id, title
)
```

### DSL vs Raw SQL

```parsley
// Raw SQL
let posts = db <=??=> "
  SELECT p.id, p.title, u.name as author_name
  FROM posts p
  JOIN users u ON p.user_id = u.id
  WHERE p.status = 'published'
  ORDER BY p.created_at DESC
  LIMIT 10
"

// DSL (no explicit JOIN needed)
let posts = @query(
  Posts
  | status == "published"
  | order created_at desc
  | limit 10
  | with author
  ??-> id, title, author.name
)
```

---

## Open Questions

1. **Transactions** — How to wrap multiple operations?
   ```parsley
   @transaction {
     @insert(...)
     @update(...)
   }
   ```

2. **Upsert** — Insert or update syntax?
   ```parsley
   @upsert(
     Users
     |< email: {email}, name: {name}
     | on email
     X
   )
   ```

3. **Aggregations** — GROUP BY, HAVING, aggregate functions?

4. **Subqueries** — How to express nested queries?

5. **Soft deletes** — Built-in support or convention?

---

## Implementation Notes

### Parsing Strategy

The @object content is parsed by a dedicated parser, not the main Parsley parser. This allows:
- Different keywords (`order`, `with`, `limit`)
- Different operators in different contexts
- Clear error messages specific to query syntax

### Compilation

DSL queries compile to the same intermediate representation as FEAT-078 methods, which then generate SQL. This ensures:
- Same security guarantees (parameterized queries)
- Same optimization opportunities
- Interoperability between styles

### Static Analysis

Because field and table names are static (not interpolated), tooling can:
- Validate field names against schema
- Check relation paths
- Warn about missing indexes
- Provide autocomplete

---

## Summary

| Feature | Syntax |
|---------|--------|
| Schema declaration | `@schema Name { field: type }` |
| Relation (belongs-to) | `author: User via user_id` |
| Relation (has-many) | `posts: [Post] via user_id` |
| Binding | `db.bind(Schema, "table")` |
| Condition | `| field == value` |
| Write | `|< field: value` |
| Return single | `?-> fields` |
| Return multiple | `??-> fields` |
| Execute | `X` |
| Execute + count | `X-> count` |
| Interpolation | `{parsley_expression}` |

The design prioritizes **readability** and **visual scanning** — you can glance at any query and immediately see what kind of operation it is, what conditions apply, what's being written, and what's being returned.
