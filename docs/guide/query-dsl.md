# Query DSL Guide

The Query DSL provides a concise syntax for database operations in Parsley. It's designed to be readable, composable, and safe from SQL injection.

## Quick Start

```parsley
// Define a schema
@schema User {
    id: int
    name: string
    email: string
    status: string
}

// Connect and bind
let db = @sqlite(":memory:")
let Users = db.bind(User, "users")

// Query
@query(Users | status == "active" ??-> *)
```

## Core Concepts

### Schemas

Schemas define the shape of your data:

```parsley
@schema Post {
    id: int
    title: string
    author_id: int
    created_at: datetime
    author: User via author_id      // belongs-to relation
}

@schema User {
    id: int
    name: string
    posts: [Post] via author_id     // has-many relation
}
```

### Bindings

Bind schemas to database tables:

```parsley
let db = @sqlite("app.db")
let Users = db.bind(User, "users")
let Posts = db.bind(Post, "posts", {soft_delete: "deleted_at"})
```

### Creating Tables from Schemas

Create tables automatically from schema definitions:

```parsley
let db = @sqlite(":memory:")

// Create table with explicit name
db.createTable(User, "users")

// Create table with auto-generated name (schema name lowercase + "s")
db.createTable(Product)  // creates "products" table

// Safe to call multiple times (uses IF NOT EXISTS)
db.createTable(User, "users")
db.createTable(User, "users")  // no error

// Then bind and use
let Users = db.bind(User, "users")
```

The method maps schema types to SQL types:
- `int` → `INTEGER`
- `string`, `text` → `TEXT`
- `bool` → `INTEGER` (SQLite) / `BOOLEAN` (PostgreSQL)
- `float` → `REAL`
- `datetime` → `TEXT` (SQLite) / `TIMESTAMP` (PostgreSQL)
- `json` → `TEXT` (SQLite) / `JSONB` (PostgreSQL)
- `id` field → auto-incrementing primary key

### Rich Schema Types

Beyond basic types, schemas support validated types and constraints:

#### Validated String Types

These types store as TEXT but validate format on `@insert` and `@update`:

```parsley
@schema Contact {
    id: int
    email: email              // validates email format
    website: url              // validates URL format
    phone: phone              // validates phone format (digits, spaces, +, -, parens)
    username: slug            // validates URL-safe slug (lowercase, hyphens, numbers)
}
```

| Type | Stores As | Validates |
|------|-----------|-----------|
| `email` | TEXT | Email format (user@domain.tld) |
| `url` | TEXT | URL format (http/https) |
| `phone` | TEXT | Phone format (digits, +, -, spaces, parens) |
| `slug` | TEXT | URL-safe slug (lowercase letters, numbers, hyphens) |

#### Enum Types

Restrict a field to a set of allowed values:

```parsley
@schema User {
    id: int
    name: string
    role: enum("admin", "user", "guest")
    status: enum("active", "pending", "suspended")
}
```

Enums generate CHECK constraints in SQL:
```sql
role TEXT CHECK(role IN ('admin', 'user', 'guest'))
```

Validation ensures only allowed values can be inserted or updated.

#### Type Constraints

Add constraints to basic types:

```parsley
@schema Product {
    id: int
    name: string(min: 1, max: 100)       // length between 1 and 100
    sku: string(unique: true)            // UNIQUE constraint
    price: int(min: 0)                   // minimum value
    stock: int(min: 0, max: 10000)       // value range
    email: email(unique: true)           // validated type + unique
}
```

| Constraint | Applies To | Effect |
|------------|------------|--------|
| `min` | string | Minimum length |
| `max` | string | Maximum length |
| `min` | int | Minimum value |
| `max` | int | Maximum value |
| `unique` | any | UNIQUE SQL constraint |

Constraints generate CHECK constraints where applicable:
```sql
name TEXT CHECK(length(name) >= 1 AND length(name) <= 100)
price INTEGER CHECK(price >= 0)
sku TEXT UNIQUE
```

#### Additional Types

```parsley
@schema Stats {
    id: int
    big_count: bigint         // 64-bit integer (INTEGER in SQLite, BIGINT in PostgreSQL)
}
```

#### Validation Errors

When validation fails on `@insert` or `@update`, a validation error is returned:

```parsley
// Invalid email format
@insert(Contacts |< email: "not-an-email" .)
// Error: Field 'email' is not a valid email address

// Invalid enum value
@insert(Users |< role: "superuser" .)
// Error: Field 'role' must be one of: admin, user, guest

// Constraint violation
@insert(Products |< name: "" .)
// Error: Field 'name' must be at least 1 characters
```

#### Complete Example

```parsley
@schema User {
    id: int
    username: slug(unique: true)
    email: email(unique: true)
    phone: phone
    website: url
    role: enum("admin", "moderator", "user")
    bio: string(max: 500)
    age: int(min: 13, max: 150)
    created_at: datetime
}

let db = @sqlite(":memory:")
db.createTable(User, "users")
let Users = db.bind(User, "users")

// This will validate all fields before inserting
let user = @insert(Users
    |< username: "alice-smith"
    |< email: "alice@example.com"
    |< phone: "+1 (555) 123-4567"
    |< role: "user"
    |< bio: "Hello, world!"
    |< age: 25
    ?-> *)
```

## Queries

### Basic Syntax

```
@query(Binding | conditions | modifiers terminal-> projection)
```

### Terminals

| Terminal | Returns | Description |
|----------|---------|-------------|
| `??->` | Array | All matching rows |
| `?->` | Single/null | First matching row or null |
| `.` | null | Execute without returning |
| `.->` | Integer | Count of affected rows |

### Projections

```parsley
@query(Users ??-> *)              // all columns
@query(Users ??-> name, email)    // specific columns
@query(Users ?-> count)           // count of rows
@query(Users ?-> exists)          // boolean: any rows exist?
```

## Conditions

### Basic Comparisons

```parsley
@query(Users | status == "active" ??-> *)
@query(Users | age > 18 ??-> *)
@query(Users | age >= 21 ??-> *)
@query(Users | name != "admin" ??-> *)
```

### Interpolation Syntax

**Rule:** Bare identifiers are columns. `{expression}` are Parsley values.

```parsley
let minAge = 18
let targetStatus = "active"

// Variables must use {braces}
@query(Users | age >= {minAge} ??-> *)
@query(Users | status == {targetStatus} ??-> *)

// Expressions work too
@query(Users | age >= {minAge + 3} ??-> *)

// Column-to-column comparisons use bare identifiers
@query(Products | price > cost ??-> *)
```

### Operators

```parsley
// Equality and comparison
| status == "active"
| age > 18
| age >= 18
| age < 65
| age <= 65
| status != "banned"

// Pattern matching
| name like "A%"           // starts with A
| email like "%@test.com"  // ends with @test.com

// Range
| age between {18} and {65}

// Set membership
| status in ["active", "pending"]
| id in <-Admins | | ?-> id    // subquery

// Null checks (must use IS NULL / IS NOT NULL)
| deleted_at is null
| verified_at is not null

// IMPORTANT: SQL NULL semantics differ from Parsley
// In SQL, `column = NULL` always returns unknown (no rows)
// Use `is null` and `is not null` instead of `== null` / `!= null`

// Negation
| not status == "banned"
| not (deleted or archived)
```

### Logical Operators

```parsley
// AND (implicit between conditions)
@query(Users | status == "active" | role == "admin" ??-> *)

// OR with parentheses
@query(Users | (status == "active" or status == "pending") ??-> *)

// Complex combinations
@query(Users | (role == "admin" or role == "mod") | status == "active" ??-> *)

// NOT
@query(Users | not status == "banned" ??-> *)
@query(Users | not (deleted or archived) ??-> *)
```

## Modifiers

### Order By

```parsley
@query(Users | order name asc ??-> *)
@query(Users | order created_at desc ??-> *)
@query(Users | order status asc, name asc ??-> *)
```

### Limit and Offset

```parsley
@query(Users | limit 10 ??-> *)
@query(Users | offset 20 | limit 10 ??-> *)
```

### Eager Loading

```parsley
// Load related records
@query(Posts | with author ??-> *)
@query(Users | with posts ??-> *)

// Nested relations
@query(Posts | with author, comments.author ??-> *)

// Filtered relations
@query(Posts | with comments(approved == true | order created_at desc | limit 5) ??-> *)
```

## Mutations

### Insert

```parsley
// Insert without return
@insert(Users |< name: "Alice" |< email: "alice@test.com" .)

// Insert and return created row
@insert(Users |< name: "Bob" ?-> *)

// Insert with variables
let userData = {name: "Charlie", email: "charlie@test.com"}
@insert(Users |< name: {userData.name} |< email: {userData.email} ?-> *)

// Batch insert
@insert(Users * each userList as user |< name: user.name |< email: user.email .)

// Upsert (insert or update)
@insert(Settings | update on key |< key: "theme" |< value: "dark" .)
```

### Update

```parsley
// Update matching rows
@update(Users | id == {userId} |< status: "inactive" .)

// Update and return count
@update(Users | status == "pending" |< status: "active" .-> count)

// Update and return modified row
@update(Users | id == {userId} |< name: "New Name" ?-> *)
```

### Delete

```parsley
// Delete matching rows
@delete(Users | id == {userId} .)

// Delete and return count
@delete(Users | status == "spam" .-> count)

// Soft delete (if binding configured with soft_delete)
@delete(Posts | id == {postId} .)  // Sets deleted_at, doesn't remove row
```

## Advanced Features

### Aggregations

```parsley
// Group by with aggregates
@query(Orders 
    + by customer_id 
    | total: sum(amount) 
    | count: count 
    ??-> customer_id, total, count)

// Filter on aggregates (HAVING)
@query(Orders 
    + by customer_id 
    | total: sum(amount) 
    | total > {1000} 
    ??-> customer_id, total)
```

### Subqueries

```parsley
// IN subquery
@query(Posts 
    | author_id in <-Users | | role == "admin" | ?-> id 
    ??-> *)

// NOT IN subquery
@query(Posts 
    | author_id not in <-BannedUsers | | ?-> id 
    ??-> *)
```

### Correlated Subqueries

```parsley
// Computed field from subquery
@query(Posts as post
    | comment_count <-Comments | | post_id == post.id | ?-> count
    | comment_count > {5}
    ??-> title, comment_count)
```

### CTEs (Common Table Expressions)

```parsley
// Named subquery referenced in main query
@query(
    Tags as food_tags | topic == "food" ??-> name
    
    Posts | tags in food_tags ??-> *
)
```

### Transactions

```parsley
@transaction {
    let user = @insert(Users |< name: "Alice" ?-> *)
    @insert(Profiles |< user_id: {user.id} |< bio: "Hello" .)
    user
}
```

## Complete Example

```parsley
@schema User {
    id: int
    name: string
    email: string
    role: string
    created_at: datetime
    posts: [Post] via author_id
}

@schema Post {
    id: int
    title: string
    body: string
    author_id: int
    status: string
    created_at: datetime
    author: User via author_id
    comments: [Comment] via post_id
}

@schema Comment {
    id: int
    post_id: int
    author_id: int
    body: string
    approved: bool
    author: User via author_id
}

let db = @sqlite("blog.db")
let Users = db.bind(User, "users")
let Posts = db.bind(Post, "posts")
let Comments = db.bind(Comment, "comments")

// Get active admin users
let admins = @query(Users | role == "admin" | status == "active" ??-> *)

// Get published posts with authors and approved comments
let posts = @query(Posts 
    | status == "published" 
    | with author, comments(approved == true | order created_at desc)
    | order created_at desc 
    | limit 10 
    ??-> *)

// Get posts with more than 5 comments
let popular = @query(Posts as p
    | comment_count <-Comments | | post_id == p.id | ?-> count
    | comment_count > {5}
    | order comment_count desc
    ??-> title, comment_count)

// Create a new post
let newPost = @insert(Posts 
    |< title: "Hello World"
    |< body: "My first post"
    |< author_id: {currentUser.id}
    |< status: "draft"
    ?-> *)
```
