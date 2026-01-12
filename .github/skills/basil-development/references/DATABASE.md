# Database Operations

> **Complete API Documentation**: See `docs/basil/reference.md` for the full database reference including connection methods, schema binding, Query DSL, and error handling.

## Quick Reference

### Database Operators

```parsley
let {db} = import @basil/auth

// Query one row (returns dict or null)
let user = db <=?=> "SELECT * FROM users WHERE id = ?" [userId]

// Query many rows (returns array)
let users = db <=??=> "SELECT * FROM users WHERE active = ?" [true]

// Execute mutation (returns {affected, lastId})
let result = db <=!=> "INSERT INTO users (name, email) VALUES (?, ?)" [name, email]
```

### Connection Properties

```parsley
let {db} = import @basil/auth

db.driver          // "sqlite", "postgres", or "mysql"
db.inTransaction   // true if in transaction
db.lastError       // Error message from last operation
```

### Connection Methods

```parsley
// Transactions
db.begin()
db.commit()
db.rollback()

// Status
db.ping()          // Test connection
db.close()         // Close (not allowed on managed @DB)
```

## Examples

### Basic Queries

```parsley
let {db} = import @basil/auth

// Find one user
let user = db <=?=> "SELECT * FROM users WHERE email = ?" ["alice@example.com"]

if (!user) {
  "User not found"
} else {
  `Welcome, {user.name}!`
}

// Get all active users
let users = db <=??=> "SELECT * FROM users WHERE active = 1 ORDER BY name"

for (user in users) {
  user.name
}
```

### Insert/Update/Delete

```parsley
let {db} = import @basil/auth

// Insert
let result = db <=!=> "INSERT INTO users (name, email) VALUES (?, ?)" ["Alice", "alice@example.com"]
result.lastId    // ID of inserted row
result.affected  // Number of rows affected (1)

// Update
let result = db <=!=> "UPDATE users SET active = 1 WHERE id = ?" [userId]
result.affected  // Number of rows updated

// Delete
let result = db <=!=> "DELETE FROM users WHERE id = ?" [userId]
result.affected  // Number of rows deleted
```

### Transactions

```parsley
let {db} = import @basil/auth

db.begin()

let _ = db <=!=> "INSERT INTO users (name) VALUES (?)" ["Alice"]
let userId = db.lastInsertId()

let _ = db <=!=> "INSERT INTO profiles (user_id, bio) VALUES (?, ?)" [userId, "Developer"]

if (someCondition) {
  db.commit()
  "Success"
} else {
  db.rollback()
  "Rolled back"
}
```

### Error Handling

```parsley
let {db} = import @basil/auth

// Check for null
let user = db <=?=> "SELECT * FROM users WHERE id = ?" [userId]
if (!user) {
  "User not found"
}

// Check affected rows
let result = db <=!=> "DELETE FROM users WHERE id = ?" [userId]
if (result.affected == 0) {
  "No user found with that ID"
}

// Use try for error capture
let {result, error} = try (db <=!=> "INSERT INTO users (name) VALUES (?)" [name])
if (error) {
  `Database error: {error}`
}
```

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
// In handler
let {db} = import @basil/auth  // db is @DB from config

// Or use directly
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
