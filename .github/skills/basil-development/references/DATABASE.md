# Database Operations

> **Complete API Documentation**: See `docs/basil/reference.md` for the full database reference including connection methods, schema binding, Query DSL, and error handling.

## Quick Reference

### Database Operators

```parsley
// Query one row (returns dict or null)
let userId = 123
let user = @DB <=?=> `SELECT * FROM users WHERE id = {userId}`

// Query many rows (returns array)
let active = true
let users = @DB <=??=> `SELECT * FROM users WHERE active = {active}`

// Execute mutation (returns {affected, lastId})
let name = "Alice"
let email = "alice@example.com"
let result = @DB <=!=> `INSERT INTO users (name, email) VALUES ('{name}', '{email}')`
```

### Connection Properties

```parsley
@DB.driver          // "sqlite", "postgres", or "mysql"
@DB.inTransaction   // true if in transaction
@DB.lastError       // Error message from last operation
```

### Connection Methods

```parsley
// Transactions
@DB.begin()
@DB.commit()
@DB.rollback()

// Status
@DB.ping()          // Test connection
@DB.close()         // Close (not allowed on managed @DB)
```

## Examples

### Basic Queries

```parsley
// Find one user
let email = "alice@example.com"
let user = @DB <=?=> `SELECT * FROM users WHERE email = '{email}'`

if (!user) {
  "User not found"
} else {
  `Welcome, {user.name}!`
}

// Get all active users
let users = @DB <=??=> "SELECT * FROM users WHERE active = 1 ORDER BY name"

for (user in users) {
  user.name
}
```

### Insert/Update/Delete

```parsley
// Insert
let name = "Alice"
let email = "alice@example.com"
let result = @DB <=!=> `INSERT INTO users (name, email) VALUES ('{name}', '{email}')`
result.affected  // Number of rows affected (1)
let userId = @DB.lastInsertId()  // ID of inserted row (SQLite only)

// Update
let userId = 123
let result = @DB <=!=> `UPDATE users SET active = 1 WHERE id = {userId}`
result.affected  // Number of rows updated

// Delete
let result = @DB <=!=> `DELETE FROM users WHERE id = {userId}`
result.affected  // Number of rows deleted
```

### Transactions

```parsley
@DB.begin()

let name = "Alice"
let result = @DB <=!=> `INSERT INTO users (name) VALUES ('{name}')`
let userId = @DB.lastInsertId()

let bio = "Developer"
let _ = @DB <=!=> `INSERT INTO profiles (user_id, bio) VALUES ({userId}, '{bio}')`

if (someCondition) {
  @DB.commit()
  "Success"
} else {
  @DB.rollback()
  "Rolled back"
}
```

### Error Handling

```parsley
// Check for null
let userId = 123
let user = @DB <=?=> `SELECT * FROM users WHERE id = {userId}`
if (!user) {
  "User not found"
}

// Check affected rows
let result = @DB <=!=> `DELETE FROM users WHERE id = {userId}`
if (result.affected == 0) {
  "No user found with that ID"
}

// Use try for error capture
let name = "Alice"
let {result, error} = try (@DB <=!=> `INSERT INTO users (name) VALUES ('{name}')`)
if (error) {
  `Database error: {error}`
}
```

## SQL Tags (Safe Parameterized Queries)

The `<SQL>` tag provides a safe way to write parameterized queries. SQL content is raw text (no quotes needed), and parameters come from attributes.

```parsley
// Basic SQL tag - no quotes needed around SQL
let users = @DB <=??=> <SQL>SELECT * FROM users WHERE active = 1</SQL>

// Parameterized query - parameters from attributes
let GetUser = fn(props) {
    <SQL id={props.id}>
        SELECT * FROM users WHERE id = ?
    </SQL>
}
let user = @DB <=?=> <GetUser id={42} />

// Multiple parameters
let InsertUser = fn(props) {
    <SQL name={props.name} email={props.email}>
        INSERT INTO users (name, email) VALUES (?, ?)
    </SQL>
}
let result = @DB <=!=> <InsertUser name="Alice" email="alice@example.com" />

// Multi-line queries
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

### Why Use SQL Tags?

**Safety**: `@{}` interpolation is blocked inside SQL tags to prevent SQL injection. All dynamic values must come through attributes, which are passed as prepared statement parameters.

```parsley
// ❌ ERROR - interpolation not allowed in SQL tags
<SQL>SELECT * FROM users WHERE id = @{id}</SQL>

// ✅ SAFE - use attributes for parameters
<SQL id={id}>SELECT * FROM users WHERE id = ?</SQL>
```

**Readability**: No need for quotes or escaping SQL strings.

**Components**: SQL tags work naturally in reusable query components.

## Connection Types

### 1. SQLite

```parsley
let db = @sqlite("./myapp.db")
let db = @sqlite(":memory:")  // In-memory database
```

### 2. PostgreSQL

```parsley
let db = @postgres("postgres://user:pass@host:5432/dbname")
```

### 3. MySQL

```parsley
let db = @mysql("user:pass@tcp(host:3306)/dbname")
```

### 4. Managed Database (@DB)

In Basil handlers, `@DB` is the configured database:

```yaml
# basil.yaml
sqlite: ./myapp.db
```

```parsley
// In handler - use @DB directly
let users = @DB <=??=> "SELECT * FROM users"
```

**Note**: Managed connections cannot be closed by scripts.

## Complete Reference

For comprehensive documentation including:
- Schema binding and table operations
- Query DSL for declarative queries
- Advanced transaction patterns
- Connection pooling
- Error codes and handling
- Type conversions
- Performance optimization

See `docs/basil/reference.md` - Section 2: Database Operations
