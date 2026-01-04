# Query DSL Design: @objects with Mini-Grammars

**Date:** 2026-01-04  
**Status:** Draft (v2)  
**Related:** FEAT-078 (TableBinding API)

---

## Overview

This document defines a domain-specific language (DSL) for database operations using Parsley's @object syntax. The key insight is that @objects already establish a pattern where content inside has *its own parsing rules* (e.g., `@date(2026-01-03)`, `@path(/users/{id})`).

We extend this pattern to schemas and queries, creating a concise, readable syntax for the 90% of common database operations.

---

## Design Principles

1. **Simplicity** — Very few keywords, easy to remember
2. **Clarity** — Visual operators show data flow direction
3. **Not SQL** — Inspired by SQL but not a mirror of it
4. **Parsley-native** — Use Parsley syntax where code-like syntax is needed (`==` not `=`, `and`/`or` not `&&`/`||`)
5. **90% coverage** — Handle common cases elegantly; escape to raw SQL for edge cases
6. **Safe by default** — All interpolation is parameterized, SQL-injection-safe

---

## Part 1: Connection to Existing Features

### Existing SQL Operators

Parsley already has database operators:

```parsley
db <=?=>  "SELECT * FROM users WHERE id = {id}"    // Single row
db <=??=> "SELECT * FROM users"                    // Multiple rows
db <=!=>  "INSERT INTO users (name) VALUES ({n})"  // Execute (no result)
```

### The DSL Mirrors These

| Raw SQL | DSL Terminal | Returns |
|---------|--------------|---------|
| `<=?=>` | `?->` | Single row/value |
| `<=??=>` | `??->` | Multiple rows |
| `<=!=>` | `.` or `•` | Nothing (execute) |
| — | `.-> count` | Count of affected rows |

This creates a consistent mental model across raw SQL and the DSL.

---

## Part 2: Schema Declarations

### Syntax

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

### Relations

| Syntax | Cardinality | Example |
|--------|-------------|---------|
| `field: Type via fk` | Belongs-to / Has-one | `author: User via user_id` |
| `field: [Type] via fk` | Has-many | `posts: [Post] via user_id` |

Forward references are implicit — schemas can reference each other without declaration order concerns.

---

## Part 3: Binding Schemas to Tables

### Basic Binding

```parsley
let Users = db.bind(User, "users")
let Posts = db.bind(Post, "posts")
let Comments = db.bind(Comment, "comments")
```

**Key insight:** Schemas are pure types (no database knowledge). Bindings add the database mapping. This separation allows:
- Same schema, different databases
- Same schema, different table names
- Schemas used for validation without any database

### Soft Deletes

Soft deletes are configured at binding time:

```parsley
// With soft deletes — auto-filters deleted_at IS NULL
let Posts = db.bind(Post, "posts", {soft_delete: "deleted_at"})

// Without soft deletes — sees all rows
let AllPosts = db.bind(Post, "posts")
```

When `soft_delete` is configured:
- `@query` automatically adds `WHERE deleted_at IS NULL`
- `@delete` sets `deleted_at = NOW()` instead of removing rows

To access deleted rows, bind without the `soft_delete` option:

```parsley
// Admin view: see deleted posts
@query(AllPosts | deleted_at is not null ??-> *)

// Hard delete for compliance
@delete(AllPosts | id == {id} .)
```

---

## Part 4: Query Structure

### General Form

```
@operation(
  Source
  | conditions
  | modifiers
  |< writes
  terminal
)
```

### Operators

| Operator | Purpose | Example |
|----------|---------|---------|
| `\|` | Condition/filter | `\| status == "active"` |
| `\|<` | Write (set value) | `\|< name: {name}` |
| `<-` | Subquery (pull from) | `\| author_id in <-Users` |
| `->` | Alias binding | `* each {items} -> item` |
| `+ by` | Group by | `+ by status` |
| `?->` | Return single | `?-> *` or `?-> id, name` |
| `??->` | Return multiple | `??-> *` or `??-> id, name` |
| `.` or `•` | Execute, no return | `.` |
| `.->` | Execute, return count | `.-> count` |

### Visual Flow

```
@query(Posts | status == "published" | order created_at desc ??-> id, title)
       ───┬─   ─────────┬──────────   ─────────┬─────────── ────────┬──────
       Source       Condition              Modifier              Output
```

---

## Part 5: CRUD Operations

### Query (Read)

```parsley
// Multiple rows
@query(
  Posts
  | status == "published"
  | order created_at desc
  | limit 10
  ??-> id, title, created_at
)
// → [
//     {id: 42, title: "Hello World", created_at: "2026-01-04T10:30:00Z"},
//     {id: 41, title: "Previous Post", created_at: "2026-01-03T14:20:00Z"},
//     ...
//   ]

// Single row by ID with relations
@query(
  Posts
  | id == {postId}
  | with author, comments
  ?-> *
)
// → {
//     id: 42,
//     title: "Hello World",
//     body: "...",
//     user_id: 7,
//     author: {id: 7, name: "Alice", email: "alice@example.com"},
//     comments: [
//       {id: 101, body: "Great post!", user_id: 3},
//       {id: 102, body: "Thanks for sharing", user_id: 5}
//     ]
//   }

// Count
@query(
  Posts
  | status == "published"
  ?-> count
)
// → 42

// Exists check
@query(
  Users
  | email == {email}
  ?-> exists
)
// → true
```

### Insert (Create)

```parsley
// Insert, no return
@insert(
  Posts
  |< title: {form.title}
  |< body: {form.body}
  |< user_id: {user.id}
  |< status: "draft"
  .
)

// Insert, return created row
let post = @insert(
  Posts
  |< title: {form.title}
  |< body: {form.body}
  |< user_id: {user.id}
  ?-> *
)

// Insert, return just ID
let id = @insert(
  Posts
  |< title: "Hello"
  |< body: "World"
  ?-> id
)
```

### Insert with Upsert

```parsley
// Insert or update on conflict
@insert(
  Users
  | update on email
  |< email: {email}
  |< name: {name}
  .
)

// With composite key
@insert(
  UserProfiles
  | update on user_id, profile_type
  |< user_id: {uid}
  |< profile_type: "primary"
  |< data: {data}
  .
)
```

The `| update on field` modifier transforms insert to upsert:
- If no row matches the key: INSERT
- If a row matches: UPDATE with the provided values

### Update (Modify)

```parsley
// Update, no return
@update(
  Posts
  | id == {postId}
  |< status: "published"
  |< published_at: {now}
  .
)

// Update, return count
let count = @update(
  Posts
  | status == "draft" and updated_at < {cutoff}
  |< status: "archived"
  .-> count
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
  .
)

// Delete, return count
let purged = @delete(
  Sessions
  | expires_at < {now}
  .-> count
)

// Delete, return removed rows (for audit)
let removed = @delete(
  Users
  | status == "banned"
  ??-> *
)
```

---

## Part 6: Conditions

### Comparison Operators

Use Parsley operators, not SQL:

| DSL | SQL | Meaning |
|-----|-----|---------|
| `==` | `=` | Equal |
| `!=` | `<>` | Not equal |
| `>` | `>` | Greater than |
| `<` | `<` | Less than |
| `>=` | `>=` | Greater or equal |
| `<=` | `<=` | Less or equal |
| `in` | `IN` | In list/subquery |
| `not in` | `NOT IN` | Not in list/subquery |
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

// IN list
@query(Posts | status in ["published", "featured"] ??-> *)

// LIKE pattern
@query(Users | email like "%@example.com" ??-> *)

// BETWEEN range
@query(Products | price between 10 and 50 ??-> *)

// NULL checks
@query(Posts | deleted_at is null ??-> *)

// Complex
@query(
  Users
  | (status == "active" and age >= 18) or role == "admin"
  | created_at >= {lastMonth}
  ??-> *
)
```

---

## Part 7: Modifiers

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

### Relations (Eager Loading)

```parsley
// Simple include
| with author

// Multiple includes
| with author, comments

// Nested includes
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

## Part 8: Aggregations

### Group By

Use `+ by` to group results:

```parsley
// Posts per status
@query(
  Posts
  + by status
  ??-> status, count
)
// → [{status: "published", count: 42}, {status: "draft", count: 7}]
```

### Aggregate Functions

| Function | Meaning |
|----------|---------|
| `count` | Number of rows |
| `sum(field)` | Sum of values |
| `avg(field)` | Average |
| `min(field)` | Minimum |
| `max(field)` | Maximum |

### Defining Computed Values

Use `name: expression` to define reusable values:

```parsley
// Total sales per customer
@query(
  Orders
  | status == "completed"
  + by customer_id
  | total: sum(total)
  ??-> customer_id, total
)

// High-value customers (HAVING equivalent)
@query(
  Orders
  | status == "completed"
  + by customer_id
  | total_spend: sum(total)
  | total_spend > 1000
  | order total_spend desc
  ??-> customer_id, total_spend
)
```

### Aggregates Without Grouping

```parsley
// Dashboard: orders this month
@query(
  Orders
  | created_at >= {monthStart}
  | revenue: sum(total)
  ?-> count, revenue
)
// → {count: 150, revenue: 12500}
```

### Real-World Examples

```parsley
// Leaderboard: top authors by post count
@query(
  Posts
  | status == "published"
  + by user_id
  | post_count: count
  | order post_count desc
  | limit 10
  ??-> user_id, post_count
)

// E-commerce: product stats
@query(
  OrderItems
  + by product_id
  | units_sold: sum(quantity)
  | revenue: sum(price * quantity)
  | order_count: count
  ??-> product_id, units_sold, revenue, order_count
)
```

---

## Part 9: Subqueries

### Inline Subqueries

Use `<-Table` to pull data from another table:

```parsley
// Posts by editors
@query(
  Posts
  | author_id in <-Users
  | | role == "editor"
  | | ?-> id
  | order created_at desc
  ??-> *
)
```

The `| |` indicates conditions on the subquery. The subquery's `?->` returns a single column for the `IN` clause.

### Nested Subqueries

```parsley
// Posts by senior editors
@query(
  Posts
  | author_id in <-Users
  | | role == "editor"
  | | id in <-Permissions
  | | | level == "senior"
  | | | ?-> user_id
  | | ?-> id
  | order created_at desc
  ??-> *
)
```

### CTE-Style (Named Subqueries)

For complex queries, name subqueries at the top:

```parsley
@query(
  Tags as food_tags
  | topic == "food"
  ??-> name

  Posts
  | status == "published"
  | tags in food_tags
  | order created_at desc
  ??-> *
)
```

### Correlated Subqueries

Name the parent scope to reference it in subqueries:

```parsley
// Posts with more than 5 comments
@query(
  Posts as post
  | comments <-Comments
  | | post_id == post.id
  | ?-> count
  | comments > 5
  ??-> *
)
```

### Scalar vs Join Subqueries

The return operator determines behavior:

| Return | Result | Use Case |
|--------|--------|----------|
| `?->` | Single value | Scalar lookup, no row expansion |
| `??->` | Multiple rows | Join-like, rows multiply |

```parsley
// Scalar lookup (one category per item)
@query(
  OrderItems as item
  | category <-Products
  | | id == item.product_id
  | ?-> category
  + by category
  ??-> category, count
)

// Join (multiple products per item would expand rows)
@query(
  Orders as o
  | items <-OrderItems
  | | order_id == o.id
  | ??-> *
  ??-> *
)
```

---

## Part 10: Batch Operations

### Iterating Over Collections

Use `* each {collection} -> alias` to insert multiple rows:

```parsley
@insert(
  OrderItems
  * each {cart.items} -> item
  |< order_id: {order.id}
  |< product_id: {item.product_id}
  |< quantity: {item.quantity}
  |< price: {item.price}
  .
)
```

This generates a single multi-row INSERT:

```sql
INSERT INTO order_items (order_id, product_id, quantity, price)
VALUES ($1, $2, $3, $4), ($1, $5, $6, $7), ...
```

### With Index

```parsley
* each {items} -> item, i    // item and index
```

---

## Part 11: Transactions

Wrap multiple operations in `@transaction`:

```parsley
@transaction {
  let order = @insert(
    Orders
    |< user_id: {user.id}
    |< status: "pending"
    |< total: {cart.total}
    ?-> *
  )

  @insert(
    OrderItems
    * each {cart.items} -> item
    |< order_id: {order.id}
    |< product_id: {item.product_id}
    |< quantity: {item.quantity}
    |< price: {item.price}
    .
  )

  @update(
    Products
    | id in {cart.productIds}
    |< stock: stock - 1
    .
  )
}
```

All operations succeed or all roll back.

---

## Part 12: Interpolation

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

### Why Interpolation Markers?

Without `{}`, there's ambiguity between columns and variables:

```parsley
// Is 'status' a column or a variable?
| status == published   // Ambiguous!

// Clear with {}
| status == {published}  // published is a Parsley variable
| status == "published"  // "published" is a literal
```

**Rule:** Bare identifiers are columns. `{...}` are Parsley expressions.

### Safety

All interpolated values are parameterized, never string-concatenated:

```parsley
// This is SAFE — {userInput} becomes a parameter
@query(Users | name == {userInput} ?-> *)

// Compiles to: SELECT * FROM users WHERE name = $1
// With params: [userInput]
```

Even malicious input is safe:
```
userInput = "'; DROP TABLE users; --"
→ Safely passed as parameter, no injection
```

### What Can Be Interpolated

| Context | Allowed |
|---------|---------|
| Values | `{expr}` — any Parsley expression |
| Lists | `{arrayExpr}` — Parsley array |
| Collections | `{collection}` — for batch operations |
| Field names | ❌ No — must be static |
| Table names | ❌ No — must be static |

This restriction enables static analysis and prevents SQL injection.

### Evaluation Timing

Interpolated expressions are:
1. **Captured** at parse time
2. **Evaluated** at query execution time
3. **Parameterized** (become `$1`, `$2`, etc.)

```parsley
@query(Posts | created_at >= {date.now()} ??-> *)
// date.now() called once when query executes
// Not re-evaluated per row
```

---

## Part 13: Full Examples

### Blog Post Lifecycle

```parsley
// Create draft
let post = @insert(
  Posts
  |< title: {form.title}
  |< body: {form.body}
  |< user_id: {currentUser.id}
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
  |< title: {form.title}
  |< body: {form.body}
  |< updated_at: {now}
  .
)

// Publish
let published = @update(
  Posts
  | id == {postId}
  |< status: "published"
  |< published_at: {now}
  ?-> *
)

// List published posts with author
let posts = @query(
  Posts
  | status == "published"
  | order published_at desc
  | limit 10
  | with author
  ??-> id, title, excerpt, published_at, author.name
)
// → [
//     {id: 42, title: "Hello", excerpt: "...", published_at: "...", author: {name: "Alice"}},
//     {id: 41, title: "World", excerpt: "...", published_at: "...", author: {name: "Bob"}},
//     ...
//   ]

// Delete
@delete(
  Posts
  | id == {postId}
  .
)
```

### E-commerce Order

```parsley
@transaction {
  // Create order
  let order = @insert(
    Orders
    |< user_id: {user.id}
    |< status: "pending"
    |< total: {cart.total}
    ?-> *
  )

  // Add order items (batch)
  @insert(
    OrderItems
    * each {cart.items} -> item
    |< order_id: {order.id}
    |< product_id: {item.product_id}
    |< quantity: {item.quantity}
    |< price: {item.price}
    .
  )

  // Update stock
  @update(
    Products
    | id in {cart.productIds}
    |< stock: stock - 1
    .
  )
}
```

### User Sync (Upsert)

```parsley
// Sync user from external provider
@insert(
  Users
  | update on external_id
  |< external_id: {profile.id}
  |< email: {profile.email}
  |< name: {profile.name}
  |< last_sync: {now}
  ?-> *
)
```

### Admin Cleanup Jobs

```parsley
// Archive old drafts
let archived = @update(
  Posts
  | status == "draft" and updated_at < {sixMonthsAgo}
  |< status: "archived"
  .-> count
)

// Purge expired sessions
let purged = @delete(
  Sessions
  | expires_at < {now}
  .-> count
)

// Deactivate inactive users
let deactivated = @update(
  Users
  | last_login < {oneYearAgo} and status == "active"
  |< status: "inactive"
  |< deactivated_at: {now}
  ??-> id, email
)
```

### Analytics Dashboard

```parsley
// Posts by status
let statusCounts = @query(
  Posts
  + by status
  ??-> status, count
)
// → [
//     {status: "published", count: 42},
//     {status: "draft", count: 7},
//     {status: "archived", count: 15}
//   ]

// Revenue by category
let revenueByCategory = @query(
  OrderItems as item
  | product <-Products
  | | id == item.product_id
  | ?-> category
  + by category
  | revenue: sum(item.price * item.quantity)
  | order revenue desc
  ??-> category, revenue
)
// → [
//     {category: "electronics", revenue: 45000},
//     {category: "clothing", revenue: 12500},
//     {category: "books", revenue: 3200}
//   ]

// Top authors
let topAuthors = @query(
  Posts
  | status == "published"
  + by user_id
  | post_count: count
  | order post_count desc
  | limit 10
  ??-> user_id, post_count
)
// → [
//     {user_id: 7, post_count: 23},
//     {user_id: 12, post_count: 18},
//     {user_id: 3, post_count: 15},
//     ...
//   ]
```

---

## Part 14: Grammar Summary

### Terminals

| Terminal | Returns | Use |
|----------|---------|-----|
| `.` or `•` | Nothing | Fire-and-forget |
| `.-> count` | Integer | Affected row count |
| `?-> fields` | Single row/value | One result expected |
| `??-> fields` | Array of rows | Multiple results |

### Operations

```
@query(Binding conditions modifiers terminal)
@insert(Binding writes terminal)
@insert(Binding | update on key writes terminal)
@update(Binding conditions writes terminal)
@delete(Binding conditions terminal)
@transaction { operations }
```

### @schema

```
@schema Name {
  field: type
  relation: Type via foreign_key
  relation: [Type] via foreign_key
}
```

### Binding

```
db.bind(Schema, "table_name")
db.bind(Schema, "table_name", {soft_delete: "deleted_at"})
```

### Conditions

```
| field == value
| field != value
| field > value
| field < value
| field >= value
| field <= value
| field in [values]
| field in <-Subquery
| field not in [values]
| field like "pattern"
| field between a and b
| field is null
| field is not null
| condition and condition
| condition or condition
| (grouped conditions)
```

### Modifiers

```
| order field (asc|desc)
| limit N
| limit N offset M
| with relations
+ by field              // GROUP BY
| name: aggregate       // define computed value
```

### Writes

```
|< field: value
|< field: {expression}
```

### Subqueries

```
| field in <-Table
| | subquery_conditions
| | ?-> column

Table as alias          // CTE-style naming
| name <-Table          // named subquery result
```

### Batch

```
* each {collection} -> alias
```

---

## Summary Table

| Feature | Syntax |
|---------|--------|
| Schema | `@schema Name { field: type }` |
| Relation (one) | `author: User via user_id` |
| Relation (many) | `posts: [Post] via user_id` |
| Binding | `db.bind(Schema, "table")` |
| Soft delete | `db.bind(Schema, "table", {soft_delete: "field"})` |
| Condition | `\| field == value` |
| Write | `\|< field: value` |
| Group | `+ by field` |
| Computed | `\| name: aggregate(field)` |
| Subquery | `\| field in <-Table` |
| Batch | `* each {collection} -> alias` |
| Return single | `?-> fields` |
| Return multiple | `??-> fields` |
| Execute | `.` or `•` |
| Execute + count | `.-> count` |
| Upsert | `\| update on key` |
| Transaction | `@transaction { ... }` |
| Interpolation | `{parsley_expression}` |

---

## Design Rationale

The DSL prioritizes:

1. **Visual scanning** — Operators are distinct: `|` filter, `|<` write, `?->` return, `.` execute
2. **Direction** — Arrows show data flow: `<-` pulls from, `->` assigns to, `?->` returns
3. **Familiarity** — Parsley operators (`==`, `and`, `or`), not SQL (`=`, `AND`, `OR`)
4. **Safety** — All interpolation parameterized, field/table names must be static
5. **Composability** — Same patterns work across operations

The 5 operations (`@query`, `@insert`, `@update`, `@delete`, `@transaction`) cover the vast majority of database needs while maintaining a minimal, memorable API.
