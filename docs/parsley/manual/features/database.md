---
id: man-pars-database
title: Database
system: parsley
type: features
name: database
created: 2026-02-06
version: 0.2.0
author: Basil Team
keywords:
  - database
  - sqlite
  - postgres
  - mysql
  - SQL
  - query
  - connection
  - transaction
  - table binding
  - schema
---

# Database

Parsley has built-in database support with dedicated connection literals, query operators, and a parameterized `<SQL>` tag. You connect to a database, run queries with arrow-style operators, and get dictionaries and tables back — no ORM, no driver imports.

## Connections

### Inline Connections

Create a connection by calling a driver literal with a DSN string:

```parsley
let db = @sqlite("./myapp.sqlite")       // SQLite file
let db = @sqlite(":memory:")              // SQLite in-memory
let db = @postgres("postgres://user:pass@localhost/mydb")
let db = @mysql("user:pass@tcp(localhost:3306)/mydb")
```

Each driver takes an optional second argument — an options dictionary:

```parsley
let db = @sqlite("./myapp.sqlite", {
    maxOpenConns: 10,
    maxIdleConns: 5
})
```

| Option | Type | Description |
|---|---|---|
| `maxOpenConns` | integer | Maximum open connections |
| `maxIdleConns` | integer | Maximum idle connections |

Connections are cached by DSN. Calling `@sqlite("./myapp.sqlite")` twice returns the same connection.

### Managed Connections (`@DB`)

Inside a Basil server handler, `@DB` returns the server's configured database connection:

```parsley
let user = @DB <=?=> <GetUser id={params.id} />
```

`@DB` is only available in server context. Using it in a standalone script produces a state error.

> ⚠️ Managed connections cannot be closed by Parsley code. Calling `.close()` on a managed connection raises `DB-0009`.

## Query Operators

Three operators handle all SQL execution. The left side is always a connection; the right side is a query string or `<SQL>` tag.

| Operator | Mnemonic | Returns | Use for |
|---|---|---|---|
| `<=?=>` | query-one | dictionary or `null` | SELECT expecting 0–1 rows |
| `<=??=>` | query-many | array | SELECT expecting multiple rows |
| `<=!=>` | execute | `{affected, lastId}` | INSERT, UPDATE, DELETE, DDL |

### Query One (`<=?=>`)

Returns a dictionary for the first matching row, or `null` if no rows match:

```parsley
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)"
let _ = db <=!=> "INSERT INTO users (name) VALUES ('Alice')"

let user = db <=?=> "SELECT * FROM users WHERE name = 'Alice'"
user.name                        // "Alice"

let nobody = db <=?=> "SELECT * FROM users WHERE name = 'Nobody'"
nobody                           // null
```

### Query Many (`<=??=>`)

Returns an array of dictionaries, one per row:

```parsley
let _ = db <=!=> "INSERT INTO users (name) VALUES ('Bob')"

let users = db <=??=> "SELECT * FROM users"
users.length()                   // 2
for (u in users) {
    u.name
}
```

### Execute (`<=!=>`)

Returns a dictionary with `affected` (rows changed) and `lastId` (last inserted row ID):

```parsley
let result = db <=!=> "INSERT INTO users (name) VALUES ('Carol')"
result.affected                  // 1
result.lastId                    // 3
```

Use execute for any statement that modifies data or schema — CREATE, INSERT, UPDATE, DELETE, DROP.

## The `<SQL>` Tag

For parameterized queries, use the `<SQL>` tag. It produces a dictionary with `sql` and `params` keys that the query operators understand.

```parsley
let name = "Alice"
let query = <SQL name={name}>
    SELECT * FROM users WHERE name = ?
</SQL>
let user = db <=?=> query
```

The content of a `<SQL>` tag is **raw text** — no quotes needed around the SQL. This works like `<style>` and `<script>` tags, where the tag boundaries define the content. Attributes are bound as query parameters in sorted key order, preventing SQL injection.

> ⚠️ **Interpolation is blocked inside `<SQL>` tags.** Unlike `<style>` and `<script>`, you cannot use `@{expr}` inside SQL content. All dynamic values must come through attributes. This is intentional — it enforces safe parameterized queries and prevents SQL injection.

```parsley
// ❌ ERROR — interpolation not allowed
<SQL>SELECT * FROM users WHERE name = '@{name}'</SQL>

// ✅ SAFE — use attributes for parameters
<SQL name={name}>SELECT * FROM users WHERE name = ?</SQL>
```

Leading and trailing whitespace is automatically trimmed from SQL content.

### SQL Components

Wrap `<SQL>` in a component function for reusable queries:

```parsley
let GetUser = fn(props) {
    <SQL id={props.id}>
        SELECT * FROM users WHERE id = ?
    </SQL>
}

let user = db <=?=> <GetUser id={42} />
```

```parsley
let InsertUser = fn(props) {
    <SQL name={props.name}>
        INSERT INTO users (name) VALUES (?)
    </SQL>
}

let result = db <=!=> <InsertUser name="Carol" />
```

Multi-line queries work naturally:

```parsley
let GetActiveUsers = fn(props) {
    <SQL status={props.status} limit={props.limit}>
        SELECT id, name, email
        FROM users
        WHERE status = ?
        ORDER BY created_at DESC
        LIMIT ?
    </SQL>
}
```

SQL comments are preserved:

```parsley
<SQL>
    -- Get all admin users
    SELECT * FROM users WHERE role = 'admin'
</SQL>
```

You can also pass a plain string directly — useful for DDL or simple queries where parameterization isn't needed:

```parsley
let _ = db <=!=> "DROP TABLE IF EXISTS temp"
```

## Connection Methods

| Method | Args | Returns | Description |
|---|---|---|---|
| `.ping()` | none | boolean | Test if connection is alive |
| `.close()` | none | null | Close the connection (not allowed on managed connections) |
| `.begin()` | none | boolean | Begin a manual transaction |
| `.commit()` | none | boolean | Commit a manual transaction |
| `.rollback()` | none | boolean | Roll back a manual transaction |
| `.createTable(schema)` | schema, name? | boolean | Create table from schema if not exists |
| `.bind(schema, name)` | schema, name, opts? | TableBinding | Bind a schema to a table |
| `.lastInsertId()` | none | integer | Last inserted row ID (SQLite only) |

### Manual Transactions

Use `.begin()`, `.commit()`, and `.rollback()` for explicit transaction control:

```parsley
let _ = db.begin()
let _ = db <=!=> <InsertUser name="Alice" />
let _ = db <=!=> <InsertUser name="Bob" />
db.commit()
```

Calling `.commit()` or `.rollback()` without a prior `.begin()` raises `DB-0006`. Calling `.begin()` when already in a transaction raises `DB-0007`.

### `@transaction` Blocks

For Query DSL operations, `@transaction` provides automatic commit/rollback:

```parsley
@transaction {
    @insert(Users |< name: "Alice" .)
    @insert(Users |< name: "Bob" .)
}
```

`@transaction` commits on success and rolls back if any statement produces an error. It returns the value of the last statement:

```parsley
let newUser = @transaction {
    let order = @insert(Orders |< status: "pending" ?-> *)
    order
}
```

> ⚠️ Nested transactions are not supported. `@transaction` finds the database connection by inspecting the DSL operations inside the block — at least one database operation must be present.

## Table Bindings

A TableBinding connects a schema to a database table, providing high-level CRUD methods. Create one with `db.bind()`:

```parsley
@schema User {
    id: id(auto)
    name: string(required)
    email: email
}

let db = @sqlite(":memory:")
db.createTable(User, "users")
let users = db.bind(User, "users")
```

`db.createTable(schema, name)` generates a `CREATE TABLE IF NOT EXISTS` statement from the schema. The second argument is the SQL table name.

### Binding Options

`db.bind()` accepts an optional third argument for configuration:

```parsley
let users = db.bind(User, "users", {soft_delete: "deleted_at"})
```

| Option | Type | Description |
|---|---|---|
| `soft_delete` | string | Column name for soft-delete timestamps |

### Query Methods

| Method | Returns | Description |
|---|---|---|
| `.all()` | Table | All rows |
| `.where(cond)` | TableBinding | Filter (chainable) |
| `.find(id)` | Record or null | Find by primary key |
| `.first()` | Record or null | First matching row |

```parsley
let allUsers = users.all()
let active = users.where({status: "active"}).all()
let user = users.find("abc-123")
```

### Mutation Methods

| Method | Single (Record) | Bulk (Table) |
|---|---|---|
| `.insert(data)` | Record (with generated ID) | `{inserted: N}` |
| `.update(data)` | Record | `{updated: N}` |
| `.save(data)` | Record (upsert) | `{inserted: N, updated: M}` |
| `.delete(data)` | `{deleted: 1}` | `{deleted: N}` |

```parsley
// Insert
let user = User({name: "Alice", email: "alice@example.com"})
let inserted = users.insert(user)
inserted.id                      // generated ID

// Update (requires id)
users.update(inserted.update({name: "Alice Smith"}))

// Save (upsert — inserts if no id, updates if id exists)
users.save(User({name: "New User", email: "new@example.com"}))

// Delete (requires id)
users.delete(inserted)
```

Update and delete require an `id` field on the record. Missing `id` raises `DB-0016` (update) or `DB-0017` (delete).

## Error Handling

Database errors are catchable with `try`:

```parsley
let result = try(fn() {
    db <=?=> "SELECT * FROM nonexistent_table"
})
if (result.error) {
    log("Query failed: " + result.error)
}
```

### Error Codes

| Code | Description |
|---|---|
| `DB-0002` | Query execution failed |
| `DB-0003` | Connection failed (driver-level) |
| `DB-0004` | Row scan failed |
| `DB-0005` | Connection ping or DDL execution failed |
| `DB-0006` | Commit/rollback without active transaction |
| `DB-0007` | Begin when already in transaction |
| `DB-0008` | Failed to read column metadata |
| `DB-0009` | Cannot close managed connection |
| `DB-0010` | Connection close failed |
| `DB-0011` | Execute (mutation) failed |
| `DB-0012` | Wrong type for connection operand |
| `DB-0013` | Nested transactions not supported |
| `DB-0014` | Failed to begin transaction |
| `DB-0015` | Transaction commit failed |

All database errors have class `database` and are catchable by `try`.

## SQL Security

Parsley validates all SQL identifiers (table and column names) against an allowlist pattern: alphanumeric characters and underscores only, maximum 64 characters. This prevents SQL injection through identifier manipulation.

Always use `<SQL>` tag attributes for user-provided values — never interpolate them into query strings:

```parsley
// SAFE — parameterized (attributes become bound parameters)
let user = db <=?=> <SQL name={input}>
    SELECT * FROM users WHERE name = ?
</SQL>

// UNSAFE — string interpolation bypasses parameterization
let user = db <=?=> `SELECT * FROM users WHERE name = '${input}'`
```

## Key Differences from Other Languages

- **Operators instead of method calls** — `<=?=>`, `<=??=>`, and `<=!=>` replace `.query()` and `.execute()`. The arrow syntax makes data flow direction and cardinality visible at a glance.
- **`<SQL>` tags** — parameterized queries use tag syntax with raw text content (no quotes needed). The tag integrates naturally with Parsley's component model. Queries are composable as components, and `@{}` interpolation is blocked to enforce safety.
- **Connection caching** — connections are cached by DSN. Repeated calls to `@sqlite()` with the same path return the same connection.
- **No driver imports** — SQLite, PostgreSQL, and MySQL are built in. You just use `@sqlite`, `@postgres`, or `@mysql`.
- **Schema-driven bindings** — `db.bind()` connects a schema to a table, giving you typed CRUD methods without writing SQL.

## See Also

- [Query DSL](query-dsl.md) — `@query`, `@insert`, `@update`, `@delete` expressions
- [Data Model](../fundamentals/data-model.md) — schemas, records, and tables
- [Error Handling](../fundamentals/errors.md) — `try` and catchable error classes
- [Tags](../fundamentals/tags.md) — tag syntax and components
- [Security Model](security.md) — file, SQL, and command security policies