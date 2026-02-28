# Database Implementation Status

## Implementation Complete ✅

The database support has been successfully implemented with the following features:

### Core Features Implemented

1. **✅ Three Query Operators**
   - `<=?=>` - Query single row (returns dictionary or null)
   - `<=??=>` - Query many rows (returns **Table** with column metadata)
   - `<=!=>` - Execute mutations (returns {affected, lastId})

2. **✅ Connection Factories**
   - `SQLITE(path, options?)` - SQLite connections with connection caching
   - `POSTGRES(url, options?)` - PostgreSQL stub (driver not included)
   - `MYSQL(url, options?)` - MySQL stub (driver not included)

3. **✅ Connection Methods**
   - `begin()` - Start a real `sql.Tx` transaction (returns boolean)
   - `commit()` - Commit the active transaction (returns boolean)
   - `rollback()` - Roll back the active transaction (returns boolean)
   - `close()` - Close connection and remove from cache
   - `ping()` - Test connection (returns boolean)

4. **✅ `<SQL>` Component Tag**
   - Parses SQL content from tag body
   - Returns dictionary with `sql` and `params` properties
   - Used with query operators

5. **✅ Comprehensive Test Suite**
   - All tests passing (100% success rate)
   - In-memory SQLite database testing
   - No external dependencies required

## Syntax

Both statement form and infix expression form are supported:

```parsley
// Connection creation
let db = @sqlite(":memory:")

// Query single row — infix or statement form
let user = db <=?=> "SELECT * FROM users WHERE id = 1"
let user <=?=> db <=?=> "SELECT * FROM users WHERE id = 1"

// Query multiple rows — returns Table (not Array)
let users = db <=??=> "SELECT * FROM users"
users.count()          // number of rows
users.columns          // ["id", "name", ...]

// Execute mutation
let result = db <=!=> "INSERT INTO users (name) VALUES ('Alice')"

// Parameterized queries via <SQL> tag
let GetUser = fn(props) {
    <SQL id={props.id}>
        SELECT * FROM users WHERE id = :id
    </SQL>
}
let user = db <=?=> <GetUser id={1} />

// Manual transactions
let _ = db.begin()
let _ = db <=!=> "INSERT INTO users (name) VALUES ('Alice')"
let _ = db.commit()

// DSL transactions
@transaction {
    @insert(Users |< name: "Alice" .)
}
```

## Known Limitations

### 1. ✅ Named Parameters — Implemented (FEAT-135)

`:name` style named parameters are now fully supported. Users write `:name` in SQL text and Parsley matches each to the corresponding `<SQL>` tag attribute, rewriting to driver-native placeholders (`$N` for PostgreSQL/SQLite, `?` for MySQL) at execution time. Attribute declaration order is irrelevant.

```parsley
let InsertUser = fn(props) {
    <SQL name={props.name} age={props.age}>
        INSERT INTO users (age, name) VALUES (:age, :name)
    </SQL>
}
```

Bare `?` positional placeholders continue to work for backward compatibility. Mixing `?` and `:name` in the same query is an error (`SQL-0006`). A `:name` with no matching attribute produces `SQL-0005`.

See `work/specs/FEAT-135.md` for full specification.

### 2. ✅ Transactions — Fully Implemented (FEAT-136)

Both manual (`begin()`/`commit()`/`rollback()`) and DSL (`@transaction`) transaction approaches now work correctly:

- `begin()` opens a real `sql.Tx` on the connection
- All raw SQL operators (`<=?=>`, `<=??=>`, `<=!=>`) route through the active `sql.Tx` when present
- `commit()` commits and clears the transaction; `rollback()` rolls back and clears it
- `@transaction { }` auto-discovers the DB connection and provides commit/rollback semantics

See `work/specs/FEAT-136.md` for full specification.

### 3. ⚠️ Spread Operator in Tag Props Not Supported

**Design shows**: `params={...props}` spread syntax
```parsley
let CreateUser = fn(props) {
    <SQL params={...props}>
        INSERT INTO users (name, email) VALUES (:name, :email)
    </SQL>
}
```

**Current workaround**: Construct params dict manually
```parsley
let CreateUser = fn(props) {
    <SQL>
        INSERT INTO users (name, email) VALUES ('Alice', 'alice@example.com')
    </SQL>
}
```

**Impact**: Spread operator in tag attributes requires parser enhancement. For now, use literal values or manual param construction.

### 4. ⚠️ Connection Properties as Dictionary

**Design shows**: Connection as dictionary with properties
```parsley
db.type           // "sqlite"
db.inTransaction  // true/false
db.lastError      // "error message"
```

**Current implementation**: Custom DBConnection type
- Properties not directly accessible
- `db.lastError` stored internally but not exposed as property
- `db.inTransaction` tracked but not accessible

**Impact**: Cannot inspect connection state via properties. Would require converting DBConnection to Dictionary with computed properties.

### 5. ⚠️ Options Not Fully Implemented

**Design shows**: Various connection options
```parsley
let db = SQLITE(@./data.db, {timeout: @5s, readonly: true})
```

**Current implementation**: Accepts options dict but only processes:
- `maxOpenConns` - maximum open connections
- `maxIdleConns` - maximum idle connections

**Impact**: Advanced options like `timeout`, `readonly`, `journal_mode` are not implemented.

## PostgreSQL and MySQL Support

**Status**: Stub implementations only (drivers not included)

- `POSTGRES()` and `MYSQL()` functions exist
- Will attempt to open connections using Go's `database/sql`
- Requires external drivers to be installed:
  - PostgreSQL: `github.com/lib/pq` or similar
  - MySQL: `github.com/go-sql-driver/mysql` or similar

**To enable**:
1. Add driver import to `pkg/evaluator/evaluator.go`
2. Add driver dependency to `go.mod`
3. Test with real database instances

## Recommendations

### For Production Use

1. **✅ Use SQLite** - Fully functional with pure Go driver (no C dependencies)
2. **✅ Use `:name` named parameters** - `<SQL id={x}>SELECT * WHERE id = :id</SQL>`
3. **✅ Use `@transaction` for DSL operations** - Automatic commit/rollback
4. **✅ Use `begin()`/`commit()`/`rollback()` for raw SQL** - Now real transactions
5. **⚠️ Avoid spread in tag props** - Wait for parser enhancement

### For Future Enhancement

1. **Spread operator in tags** - Enhance tag props parser
2. **Connection as Dictionary** - Convert DBConnection to Dictionary with computed properties
3. **Advanced options** - Implement timeout, readonly, journal_mode, etc.
4. **PostgreSQL/MySQL drivers** - Add and test external database drivers

## Implementation History

| Feature | Spec | Status |
|---------|------|--------|
| Three query operators | — | ✅ Original implementation |
| `<SQL>` tag | — | ✅ Original implementation |
| Named parameters (`:name`) | FEAT-135 | ✅ Implemented 2026-02-28 |
| Transaction-aware operators | FEAT-136 | ✅ Implemented 2026-02-28 |
| Real `begin()`/`commit()`/`rollback()` | FEAT-136 | ✅ Implemented 2026-02-28 |
| `<=??=>` returns Table (both forms) | FEAT-136 | ✅ Implemented 2026-02-28 |
| `rows.Err()` checks in single-row queries | FEAT-136 | ✅ Implemented 2026-02-28 |
| Block comment EOF fix in SQL scanner | FEAT-136 | ✅ Implemented 2026-02-28 |

## Test Examples

See `pkg/parsley/tests/database_test.go` and `database_advanced_test.go` for working examples:
- Connection creation and management
- CRUD operations with all three operators
- Transaction handling (manual and `@transaction`)
- SQL component usage with named parameters
