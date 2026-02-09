---
id: man-pars-query-dsl
title: Query DSL
system: parsley
type: features
name: query-dsl
created: 2026-02-06
version: 0.2.0
author: Basil Team
keywords:
  - query
  - insert
  - update
  - delete
  - DSL
  - database
  - table binding
  - schema
  - transaction
  - subquery
  - CTE
  - relations
  - group by
---

# Query DSL

Parsley's Query DSL provides declarative syntax for database operations through table bindings. Instead of writing SQL strings, you compose queries using `@query`, `@insert`, `@update`, and `@delete` expressions with a pipe-based syntax that generates parameterized SQL under the hood. It is designed to be minimalist, graphical and to express the flow of data passing through multiple steps.

All DSL operations require a **TableBinding** — a schema bound to a database table via `db.bind()`. See [Database](database.md) for connection setup and binding creation.

## Setup

Every example on this page assumes this boilerplate:

```parsley
@schema User {
    id: int
    name: string
    email: string
    status: string
}

let db = @sqlite(":memory:")
db.createTable(User, "users")
let Users = db.bind(User, "users")
```

## Terminals

Every DSL expression ends with a **terminal** that controls what gets returned. Terminals appear at the end of the expression, before the closing `)`.

| Terminal | Name | Returns | Use for |
|---|---|---|---|
| `?-> *` | return one | Record or null | Single row with all columns |
| `?-> col1, col2` | return one (projection) | dictionary | Single row with named columns |
| `??-> *` | return many | array | All matching rows |
| `??-> col1, col2` | return many (projection) | array | All matching rows with named columns |
| `.` | execute | null | Fire-and-forget mutations |
| `.-> count` | execute count | integer | Number of affected rows |
| `?-> count` | count | integer | COUNT query |
| `?-> exists` | exists | boolean | Existence check |
| `?-> toSQL` | to SQL | dictionary | Generated SQL and params (debugging) |

## @query

Read data from a table binding. The general form is:

```
@query(Binding | conditions | modifiers terminal)
```

Or if separated on to multiple lines:

```
@query(
	Binding 
	| conditions 
	| modifiers 
	terminal
)
```

### Select All Rows

```parsley
@query(Users ??-> *)
```

### Select One Row

```parsley
@query(Users | id == 1 ?-> *)
```

Returns a record if found, or `null` if no rows match.

### Projection

Return only specific columns:

```parsley
@query(Users | status == "active" ??-> name, email)
```

### Count and Exists

```parsley
@query(Users ?-> count)                             // 3
@query(Users | id == 1 ?-> exists)                  // true
@query(Users | id == 999 ?-> exists)                // false
```

## Conditions

Conditions follow the binding name, each prefixed with `|`. They map to SQL `WHERE` clauses.

### Comparison Operators

```parsley
@query(Users | status == "active" ??-> *)           // equality
@query(Users | id != 3 ??-> *)                      // inequality
@query(Users | id > 1 ??-> *)                       // greater than
@query(Users | id >= 2 ??-> *)                      // greater or equal
@query(Users | id < 3 ??-> *)                       // less than
@query(Users | id <= 2 ??-> *)                      // less or equal
```

### Variable Interpolation

Use `{expression}` to inject Parsley values as parameterized values (safe from SQL injection):

```parsley
let targetId = 2
@query(Users | id == {targetId} ?-> *)
```

### Column-to-Column Comparison

Bare identifiers on both sides compare columns:

```parsley
@query(Products | price > cost ??-> *)
```

### Multiple Conditions

Multiple `|` clauses combine with AND:

```parsley
@query(Users | status == "active" | id > 1 ??-> *)
```

Once you have more than one clause, we recommend using a multi-line query:

```parsley
@query(
	Users
	| status == "active"
	| id > 1
	??-> *)
```

Which can re read as:

```parsley
FIND
	Users
	WHERE status == "active"
	AND id > 1
	AS AN ARRAY-> OF '*' (i.e. all columns)
```
### BETWEEN

```parsley
@query(Products | price between 40 and 110 ??-> *)
```

With variables:

```parsley
let lo = 15
let hi = 25
@query(Products | price between {lo} and {hi} ??-> *)
```

### LIKE

```parsley
@query(Users | email like "%gmail%" ??-> *)
```

### NOT

Prefix a condition group with `!` to negate:

```parsley
@query(Users | !(status == "banned") ??-> *)
```

### Grouped Conditions

Parentheses create OR groups:

```parsley
@query(
	Users 
	| (status == "active" | status == "pending") 
	??-> *)
```

Combine groups with other conditions:

```parsley
@query(
	Users 
	| (status == "active" | status == "pending") 
	| id > 5 
	??-> *)
```

## Modifiers

Modifiers control ordering, limits, and eager loading. Each is prefixed with `|`.

### Order By

```parsley
@query(Users | order name ??-> *)                   // ascending (default)
@query(Users | order name desc ??-> *)              // descending
@query(Users | order name asc, id desc ??-> *)      // multiple fields
```

### Limit and Offset

```parsley
@query(Users | order id asc | limit 10 ??-> *)

@query(
	Users
	| order id asc
	| limit 10
	| offset 20 
	??-> *)
```

### Eager Loading (with)

Load related records in a single query. Relations must be declared in the schema:

```parsley
@schema Author {
    id: int
    name: string
    posts: [Post] via author_id          // has-many
}

@schema Post {
    id: int
    title: string
    author_id: int
    author: Author via author_id         // belongs-to
}

let Authors = db.bind(Author, "authors")
let Posts = db.bind(Post, "posts")

// Eager-load the author for each post
@query(
	Posts 
	| id == 1 
	| with author 
	?-> *)

// Eager-load all posts for an author
@query(
	Authors 
	| id == 1 
	| with posts 
	?-> *)
```

Nested relations use dot notation:

```parsley
@query(Authors | with posts.comments ?-> *)
```

You can add conditions, ordering, and limits to eager-loaded relations:

```parsley
@query(
	Authors 
	| with posts 
	| status == "published" 
	| order created_at desc 
	| limit 5 
	?-> *)
```

## Group By and Aggregation

Use `+ by` to group rows. Computed fields define aggregations:

```parsley
@query(
	Orders 
	+ by status 
	| order_count: count 
	??-> status, order_count)
```

### Aggregate Functions

| Function | Description |
|---|---|
| `count` | Number of rows in each group |
| `sum(field)` | Sum of field values |
| `avg(field)` | Average of field values |
| `min(field)` | Minimum field value |
| `max(field)` | Maximum field value |

```parsley
@query(
	Orders 
	+ by customer_id 
	| total: sum(amount) 
	??-> customer_id, total)

@query(
	Orders 
	+ by customer_id 
	| average: avg(amount) 
	??-> customer_id, average)
```

Aggregates also work without `+ by` to compute over the entire table:

```parsley
@query(Orders | total: sum(amount) ?-> total)
```

## @insert

Insert rows into a table binding. Fields are written with `|<` (pipe-write):

```parsley
@insert(
	Users 
	|< name: "Alice" 
	|< email: "alice@test.com" 
	.
)
```

### Return the Inserted Row

```parsley
let user = @insert(Users |< name: "Bob" ?-> *)
user.id                                             // auto-generated ID
```

### Variable Values

```parsley
let userName = "Carol"
@insert(
	Users 
	|< name: {userName} 
	|< email: "carol@test.com" 
	.)
```

### Batch Insert

Insert from a collection using `* each`:

```parsley
let people = [
    {name: "Alice", age: 25},
    {name: "Bob", age: 30},
    {name: "Carol", age: 35}
]

@insert(
	Users 
	* each people as person 
	|< name: person.name 
	|< age: person.age 
	.)
```

### Upsert

Insert or update on conflict using `| update on`:

```parsley
@insert(
	Settings 
	| update on key 
	|< key: "theme" 
	|< value: "dark" 
	.)
```

If a row with the same `key` exists, it updates; otherwise it inserts.

## @update

Update rows matching conditions. Conditions come before `|<` writes:

```parsley
@update(
	Users 
	| status == "old" 
	|< status: "updated" 
	.)
```

### Return Affected Count

```parsley
@update(
	Users 
	| status == "old" 
	|< status: "updated" 
	.-> count)  // 2
```

### Return the Updated Row

```parsley
let user = @update(
	Users 
	| id == 1 
	|< score: 200 
	?-> *)
```

### Multiple Field Updates

```parsley
@update(
	Users 
	| id == 1 
	|< name: "Alice Smith" 
	|< email: "alice.smith@test.com" 
	.)
```

## @delete

Delete rows matching conditions:

```parsley
@delete(Users | id == 1 .)
```

### Return Deleted Count

```parsley
@delete(Users | status == "expired" .-> count)      // 2
```

### Soft Delete

When the table binding has `soft_delete` configured, `@delete` sets the timestamp column instead of removing the row. Subsequent `@query` calls automatically filter out soft-deleted rows:

```parsley
let Posts = db.bind(Post, "posts", {soft_delete: "deleted_at"})

@delete(Posts | id == 1 .)

// Post 1 is still in the database but won't appear in queries
@query(Posts ??-> *)
```

## Subqueries

*You’re probably not going to need subqueries. But Parsely’s Query DSL does support them:-*

Use `<-table_name` with double-pipe delimiters to embed a subquery as a condition value:

```parsley
// Posts by admins
@query(Posts | author_id in <-users | | role == "admin" | | ?-> id ??-> title)
```

The subquery `<-users | | role == "admin" | | ?-> id` generates a `SELECT id FROM users WHERE role = 'admin'` and uses it in an `IN` clause. Note the double `|` delimiters that bracket the subquery's own conditions.

The double-pipe makes more sense when you see it written across multiple lines:

```parsley
// Posts by admins
@query(
	Posts
	| author_id in <-users
	| | role == "admin"
	| | ?-> id 
	??-> title)
```

### NOT IN

```parsley
@query(
	Posts
	| author_id not in <-users 
	| | role == "admin" 
	| | ?-> id 
	??-> title)
```

## Correlated Subqueries

A correlated subquery computes a value for each row in the outer query. Use `as alias` on the outer query and `<-` with the alias reference:

```parsley
@query(
	Posts as post
	| comment_count <-comments 
	| | post_id == post.id 
	| ?-> count
	??-> *)
```

This adds a `comment_count` computed field to each post, containing the count of related comments.

### With Filters

```parsley
@query(Posts as post
    | recent_count <-comments 
    | | post_id == post.id 
    | | created_at > "2024-01-01" 
    | ?-> count
    ??-> *)
```

## CTEs (Common Table Expressions)

Chain multiple query blocks to build CTEs. Earlier blocks define named result sets that later blocks can reference:

```parsley
@query(
    Tags as food_tags
    | topic == "food"
    ??-> name

    Posts
    | status == "published"
    | tag_name in food_tags
    ??-> title
)
```

This generates SQL with a `WITH food_tags AS (SELECT name FROM tags WHERE topic = 'food')` clause.

Multiple CTEs:

```parsley
@query(
    Categories as active_cats
    | active == 1
    ??-> name

    Products
    | featured == 1
    | category_name in active_cats
    ??-> name
)
```

## Join-Like Expansion

Use a correlated subquery with `??->` (return many) to produce join-like row expansion:

```parsley
@query(
	Orders as o
    | items <-order_items 
    | | order_id == o.id 
    | ??-> *
    ??-> *)
```

This flattens the result — each order row is repeated for each matching item, similar to a SQL JOIN.

## @transaction

Wrap multiple DSL operations in an atomic transaction:

```parsley
@transaction {
    @insert(Users |< name: "Alice" .)
    @insert(Users |< name: "Bob" .)
}
```

The transaction commits on success. If any statement produces an error, all changes are rolled back.

### Return Values

`@transaction` returns the value of the last statement:

```parsley
let newUser = @transaction {
    let order = @insert(Orders |< status: "pending" ?-> *)
    order
}
```

### Let Bindings

Variables declared inside a transaction are scoped to the block:

```parsley
@transaction {
    let user = @insert(Users |< name: "Alice" ?-> *)
    @insert(Orders |< user_id: user.id |< status: "new" .)
}
```

> ⚠️ Nested transactions are not supported. `@transaction` discovers the database connection from the DSL operations inside the block — at least one must be present.

## Debugging with toSQL

Use `?-> toSQL` to see the generated SQL without executing the query:

```parsley
let info = @query(Users | status == "active" | order name ?-> toSQL)
info.sql                                            // the SQL string
info.params                                         // the bound parameters
```

## Schema Validation

The DSL validates inserted and updated values against the schema. Type-constrained fields (email, URL, slug, enum) are checked before the SQL is generated:

```parsley
@schema User {
    id: int
    email: email(required)
    role: string("admin", "user", "guest")
}

// Error: invalid email format
@insert(Users |< email: "not-an-email" .)

// Error: invalid enum value
@insert(Users |< role: "superadmin" .)
```

## Key Differences from Other Languages

- **No SQL strings** — the DSL generates parameterized SQL from a declarative syntax. You never concatenate values into query strings.
- **Pipe-based composition** — conditions (`|`), field writes (`|<`), and modifiers (`| order`, `| limit`) chain naturally. The syntax reads left to right; up to down.
- **Terminals control return shape** — `?->` for one, `??->` for many, `.` for fire-and-forget. The terminal is always the last thing before `)`.
- **Schema-aware** — the DSL validates values against the bound schema's type constraints before generating SQL.
- **Subqueries and CTEs** — complex multi-table queries compose within a single `@query()` expression rather than requiring raw SQL.

## See Also

- [Database](database.md) — connections, raw SQL operators, table bindings
- [Data Model](../fundamentals/data-model.md) — schemas, records, and tables
- [Error Handling](../fundamentals/errors.md) — `try` and catchable error classes
- [Security Model](security.md) — SQL injection prevention