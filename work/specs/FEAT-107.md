---
id: FEAT-107
title: "Database Driver Support: PostgreSQL and MySQL"
status: draft
priority: high
created: 2025-01-15
author: "@human"
blocking: true
---

# FEAT-107: Database Driver Support: PostgreSQL and MySQL

## Summary
Add PostgreSQL and MySQL driver dependencies to enable the existing `@postgres()` and `@mysql()` connection functions. The code implementations already exist in the evaluator, but the Go SQL drivers are not included in `go.mod`, causing runtime failures when users attempt to connect to these databases.

## User Story
As a Parsley developer, I want to connect to PostgreSQL and MySQL databases so that I can build applications that use industry-standard relational databases beyond SQLite.

## Acceptance Criteria
- [ ] `@postgres("connection_string")` successfully connects to a PostgreSQL database
- [ ] `@mysql("connection_string")` successfully connects to a MySQL database
- [ ] Both drivers support connection options (maxOpenConns, maxIdleConns)
- [ ] Error messages are clear when connection fails (wrong credentials, host unreachable, etc.)
- [ ] Documentation is updated to reflect "supported" status for PostgreSQL and MySQL
- [ ] Integration tests verify basic connectivity and query execution

## Design Decisions

- **Driver choice (PostgreSQL)**: Use `github.com/lib/pq` — the most widely used pure-Go PostgreSQL driver with excellent compatibility.

- **Driver choice (MySQL)**: Use `github.com/go-sql-driver/mysql` — the standard MySQL driver for Go, maintained by the Go-SQL-Driver team.

- **Import side-effects**: Drivers register with `database/sql` via init(). The imports will be added to a dedicated `drivers.go` file in the evaluator package for clarity.

- **No code changes to evaluator logic**: The existing `@postgres()` and `@mysql()` implementations in `evaluator.go` are complete and correct. Only driver registration is needed.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `go.mod` — Add driver dependencies
- `pkg/parsley/evaluator/drivers.go` — New file for driver imports (side-effects)
- `docs/parsley/reference.md` — Update database section to show PostgreSQL/MySQL as supported
- `pkg/parsley/evaluator/evaluator_db_test.go` — Add integration tests

### Dependencies
- Depends on: None
- Blocks: None (but blocking for 1.0 Alpha release)

### Edge Cases & Constraints

1. **DSN format differences** — PostgreSQL uses `postgres://user:pass@host/db` or key-value format; MySQL uses `user:pass@tcp(host:port)/db`. Document both formats clearly.

2. **TLS/SSL connections** — Both drivers support TLS. PostgreSQL via `sslmode=require`, MySQL via `tls=true`. No special handling needed; users pass appropriate DSN parameters.

3. **Connection pooling** — Already implemented via `maxOpenConns` and `maxIdleConns` options in existing code.

4. **Connection caching** — Already implemented via `dbCache` in evaluator. No changes needed.

5. **Binary size impact** — Adding drivers increases binary size. This is acceptable for a batteries-included approach.

### Required Go Modules

```go
// go.mod additions
require (
    github.com/lib/pq v1.10.9
    github.com/go-sql-driver/mysql v1.8.1
)
```

### Implementation Sketch

**New file: `pkg/parsley/evaluator/drivers.go`**

```go
package evaluator

// Database driver imports for side-effect registration with database/sql.
// These drivers are required for @postgres() and @mysql() to function.

import (
    _ "github.com/lib/pq"           // PostgreSQL driver
    _ "github.com/go-sql-driver/mysql" // MySQL driver
)
```

**That's it.** The existing code in `evaluator.go` (lines 1877-2020) handles everything else:
- Connection creation with `sql.Open("postgres", dsn)` and `sql.Open("mysql", dsn)`
- Connection pooling options
- Connection caching
- Error handling with driver-specific messages
- `DBConnection` object creation

## Test Plan

### Unit Tests (Mock-based)
| Test Case | Description | Expected |
|-----------|-------------|----------|
| Driver registration | Verify `sql.Drivers()` includes "postgres" and "mysql" | Both drivers listed |

### Integration Tests (Require running databases)

These tests should be skipped in CI unless database services are available (via environment variable or Docker).

| Test Case | Command | Expected |
|-----------|---------|----------|
| PostgreSQL connect | `@postgres("postgres://user:pass@localhost/testdb")` | Returns DBConnection |
| PostgreSQL query | `db.query("SELECT 1 as num")` | Returns `[{num: 1}]` |
| PostgreSQL bad credentials | `@postgres("postgres://bad:creds@localhost/db")` | Clear error message |
| MySQL connect | `@mysql("user:pass@tcp(localhost:3306)/testdb")` | Returns DBConnection |
| MySQL query | `db.query("SELECT 1 as num")` | Returns `[{num: 1}]` |
| MySQL bad credentials | `@mysql("bad:creds@tcp(localhost)/db")` | Clear error message |
| Connection options | `@postgres(url, {maxOpenConns: 10})` | Pool configured correctly |

### Docker Compose for Local Testing

```yaml
# docker-compose.test.yml
version: '3.8'
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: test
      POSTGRES_PASSWORD: test
      POSTGRES_DB: testdb
    ports:
      - "5432:5432"
  
  mysql:
    image: mysql:8
    environment:
      MYSQL_ROOT_PASSWORD: test
      MYSQL_DATABASE: testdb
      MYSQL_USER: test
      MYSQL_PASSWORD: test
    ports:
      - "3306:3306"
```

## Documentation Updates

### `docs/parsley/reference.md` — Database Section

Update the database connection section to show PostgreSQL and MySQL as fully supported:

```markdown
## Database Connections

Parsley supports three database drivers out of the box:

### SQLite
@sqlite("path/to/database.db")

### PostgreSQL  
@postgres("postgres://user:password@localhost:5432/dbname?sslmode=disable")

### MySQL
@mysql("user:password@tcp(localhost:3306)/dbname")
```

## Implementation Notes
*To be added during implementation*

## Related
- Report: `work/reports/PARSLEY-1.0-ALPHA-READINESS.md` (Section 1)
- Design: `work/parsley/design/Database Implementation Status.md`
- Existing code: `pkg/parsley/evaluator/evaluator.go` lines 1877-2020