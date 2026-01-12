# Parsley Query DSL Skill

## Description
Creating database schemas and queries using Parsley's Query DSL - a concise, type-safe syntax for database operations with @schema definitions and query operators.

## When to Use
- Defining database schemas with relations
- Querying databases with type-safe operations
- Performing CRUD operations (Create, Read, Update, Delete)
- Working with relations and eager loading
- Writing transactional database code

## Core Concepts

### Schema Definitions
Schemas define the shape of database tables with typed fields and relations:

```parsley
@schema User {
    id: int
    name: string
    email: email
    created_at: datetime
}

@schema Post {
    id: int
    title: string
    body: text
    user_id: int
    author: User via user_id        // belongs-to relation
    created_at: datetime
}
```

**Key Points:**
- Fields use type annotations (int, string, text, datetime, etc.)
- Relations use `Type via foreign_key` for belongs-to/has-one
- Relations use `[Type] via foreign_key` for has-many
- Forward references work automatically

### Database Connection & Binding
Connect to database and bind schemas to tables:

```parsley
// Connect to database
let db = @sqlite(`app.db`)

// Bind schemas to tables
let Users = db.bind(User, `users`)
let Posts = db.bind(Post, `posts`)
```

### Creating Tables
Generate tables from schemas automatically:

```parsley
db.createTable(User, `users`)
db.createTable(Post, `posts`)
```

Maps schema types to SQL types (INT, TEXT, etc.) with appropriate constraints.

## Query Operations

### Query Syntax Pattern
```
@query(Binding | conditions | modifiers terminal-> projection)
```

**Terminals:**
- `?->` - Single row/value
- `??->` - Multiple rows
- `.` - Execute without return

**Projections:**
- `*` - All columns
- `id, name, email` - Specific columns
- `count` - Count of rows
- `exists` - Boolean check

### Basic Queries

**Get all rows:**
```parsley
@query(Posts ??-> *)
```

**Get single row by ID:**
```parsley
let userId = 42
@query(Users | id == {userId} ?-> *)
```

**Get specific columns:**
```parsley
@query(Posts | status == `published` ??-> id, title, created_at)
```

**Count rows:**
```parsley
@query(Posts | status == `draft` ?-> count)
```

**Check existence:**
```parsley
@query(Users | email == {userEmail} ?-> exists)
```

### Conditions & Operators

**Comparison:**
```parsley
@query(Users | age >= 18 ??-> *)
@query(Posts | status != `draft` ??-> *)
```

**Pattern matching:**
```parsley
@query(Users | email like `%@example.com` ??-> *)
```

**Range:**
```parsley
@query(Products | price between 10 and 50 ??-> *)
```

**Set membership:**
```parsley
@query(Posts | status in [`published`, `featured`] ??-> *)
```

**NULL checks (use `is null`/`is not null`):**
```parsley
@query(Posts | deleted_at is null ??-> *)
```

**Logical operators:**
```parsley
@query(Users | status == `active` and age >= 18 ??-> *)
@query(Posts | (status == `published` or status == `featured`) ??-> *)
```

### Ordering & Pagination

**Order by:**
```parsley
@query(Posts | order created_at desc ??-> *)
@query(Users | order name asc, created_at desc ??-> *)
```

**Limit and offset:**
```parsley
@query(Posts | limit 10 ??-> *)
@query(Posts | offset 20 | limit 10 ??-> *)
```

### Eager Loading Relations

**Load single relation:**
```parsley
@query(Posts | with author ??-> *)
```

**Load multiple relations:**
```parsley
@query(Posts | with author, comments ??-> *)
```

**Nested relations:**
```parsley
@query(Posts | with author, comments.author ??-> *)
```

**Filtered relations:**
```parsley
@query(Posts | with comments(approved == true | order created_at desc) ??-> *)
```

## Insert Operations

### Basic Insert

**Insert without return:**
```parsley
@insert(Users |< name: `Alice` |< email: `alice@test.com` .)
```

**Insert and return created row:**
```parsley
let user = @insert(Users 
    |< name: `Bob` 
    |< email: `bob@test.com` 
    ?-> *)
```

**Insert and return ID:**
```parsley
let userId = @insert(Users 
    |< name: `Charlie` 
    |< email: `charlie@test.com` 
    ?-> id)
```

### Insert with Variables

```parsley
let userData = {name: `Diana`, email: `diana@test.com`}
@insert(Users 
    |< name: {userData.name} 
    |< email: {userData.email} 
    ?-> *)
```

### Upsert (Insert or Update)

```parsley
@insert(Settings 
    | update on key 
    |< key: `theme` 
    |< value: `dark` 
    .)
```

## Update Operations

**Update without return:**
```parsley
let userId = 42
@update(Users 
    | id == {userId} 
    |< status: `inactive` 
    .)
```

**Update and return count:**
```parsley
let count = @update(Users 
    | status == `pending` 
    |< status: `active` 
    .-> count)
```

**Update and return modified row:**
```parsley
let user = @update(Users 
    | id == {userId} 
    |< name: `New Name` 
    ?-> *)
```

## Delete Operations

**Delete without return:**
```parsley
let userId = 42
@delete(Users | id == {userId} .)
```

**Delete and return count:**
```parsley
let deleted = @delete(Users | status == `spam` .-> count)
```

## Transactions

Wrap multiple operations for atomic execution:

```parsley
@transaction {
    let user = @insert(Users 
        |< name: `Alice` 
        |< email: `alice@test.com` 
        ?-> *)
    
    let post = @insert(Posts 
        |< title: `Hello World` 
        |< user_id: {user.id} 
        ?-> *)
    
    post
}
```

**Error handling:**
```parsley
let result = @transaction {
    let author = @insert(Users |< name: `Alice` ?-> *)
    @insert(Posts |< title: `Hello`, user_id: {author.id} ?-> *)
}

if (result.error?) {
    <p>`Failed: {result.message}`</p>
} else {
    <p>`Created post: {result.title}`</p>
}
```

## Complete Working Example

```parsley
// Define schemas
@schema User {
    id: int
    name: string
    email: email
    role: enum(`admin`, `user`, `guest`)
    created_at: datetime
}

@schema Post {
    id: int
    title: string
    body: text
    status: enum(`draft`, `published`, `archived`)
    user_id: int
    author: User via user_id
    created_at: datetime
}

// Connect and bind
let db = @sqlite(`:memory:`)
db.createTable(User, `users`)
db.createTable(Post, `posts`)

let Users = db.bind(User, `users`)
let Posts = db.bind(Post, `posts`)

// Create a user
let user = @insert(Users
    |< name: `Alice`
    |< email: `alice@example.com`
    |< role: `admin`
    |< created_at: {@now}
    ?-> *)

// Create posts
@insert(Posts
    |< title: `Hello World`
    |< body: `My first post`
    |< status: `published`
    |< user_id: {user.id}
    |< created_at: {@now}
    .)

@insert(Posts
    |< title: `Second Post`
    |< body: `More content`
    |< status: `draft`
    |< user_id: {user.id}
    |< created_at: {@now}
    .)

// Query published posts with author
let publishedPosts = @query(Posts
    | status == `published`
    | with author
    | order created_at desc
    ??-> *)

// Update post status
@update(Posts
    | status == `draft`
    | user_id == {user.id}
    |< status: `published`
    .-> count)

// Get user's post count
let postCount = @query(Posts
    | user_id == {user.id}
    ?-> count)
```

## Common Patterns

### Pagination
```parsley
let page = @params.page or 1
let perPage = 10
let offset = (page - 1) * perPage

@query(Posts
    | status == `published`
    | order created_at desc
    | limit {perPage}
    | offset {offset}
    ??-> *)
```

### Search with LIKE
```parsley
let searchTerm = @params.q
@query(Posts
    | title like `%{searchTerm}%`
    | status == `published`
    | order created_at desc
    ??-> *)
```

### Get or Create
```parsley
let existing = @query(Users | email == {email} ?-> *)

if (existing) {
    existing
} else {
    @insert(Users |< email: {email} |< name: {name} ?-> *)
}
```

### Soft Deletes
```parsley
// Configure at binding
let Posts = db.bind(Post, `posts`, {soft_delete: `deleted_at`})

// Delete sets deleted_at instead of removing
@delete(Posts | id == {postId} .)

// Queries automatically filter out deleted rows
@query(Posts ??-> *)
```

## Important Rules

### Interpolation
- **Use backticks for string values:** `` `text` `` not `"text"`
- **Use `{}` for Parsley expressions:** `{variable}` not bare `variable`
- **Column names are bare:** `| status == value` (status is column)
- **Values need braces:** `| status == {value}` (value is Parsley variable)

### NULL Handling
- **Use `is null` / `is not null`:** Not `== null` / `!= null`
- SQL NULL semantics differ from Parsley null checking

### String Interpolation in Templates
When building SQL with values, use backticks with `{}`:
```parsley
@DB <=?=> `SELECT * FROM users WHERE id = {userId}`
```

## Anti-Patterns to Avoid

❌ **Don't use double quotes for strings:**
```parsley
@query(Users | status == "active" ??-> *)  // Wrong
```

✅ **Use backticks:**
```parsley
@query(Users | status == `active` ??-> *)  // Correct
```

❌ **Don't use `== null` for NULL checks:**
```parsley
@query(Posts | deleted_at == null ??-> *)  // Wrong
```

✅ **Use `is null`:**
```parsley
@query(Posts | deleted_at is null ??-> *)  // Correct
```

❌ **Don't forget braces for variables:**
```parsley
@query(Users | id == userId ??-> *)  // Wrong - treats userId as column
```

✅ **Use `{}` for variables:**
```parsley
@query(Users | id == {userId} ??-> *)  // Correct
```

## Type Reference

### Schema Types
- `int` - Integer
- `string` - Text (short)
- `text` - Text (long)
- `bool` - Boolean
- `float` - Floating point
- `datetime` - Timestamp
- `date` - Date only
- `time` - Time only
- `email` - Validated email
- `url` - Validated URL
- `phone` - Validated phone
- `slug` - URL-safe string
- `json` - JSON data
- `enum("a", "b")` - Enumerated values

### Query Operators
- `==` - Equal
- `!=` - Not equal
- `>`, `<`, `>=`, `<=` - Comparison
- `in` - Set membership
- `not in` - Not in set
- `like` - Pattern match
- `between X and Y` - Range
- `is null` / `is not null` - NULL checks
- `and`, `or`, `not` - Logical operators

## Additional Resources
- See [Query DSL Guide](../../docs/guide/query-dsl.md) for full reference
- See [FEAT-079](../../work/specs/FEAT-079.md) for design rationale
- See [FEAT-081](../../work/specs/FEAT-081.md) for rich schema types
