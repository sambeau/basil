---
id: FEAT-003
title: "Parsley Library: WithDB() Option"
status: implemented
priority: high
created: 2025-11-30
implemented: 2025-11-30
author: "@sambeau"
target-repo: sambeau/parsley
blocks: FEAT-002 (Phase 2)
---

# FEAT-003: Parsley Library: WithDB() Option

## Summary
Add a `WithDB()` option to the Parsley library API (`pkg/parsley`) that allows host applications to inject a server-managed `*sql.DB` connection into Parsley scripts. This enables web servers and other embedding applications to control database connection lifecycle, pooling, and concurrency.

## User Story
As a developer embedding Parsley in a web server, I want to pass a server-managed database connection to Parsley scripts so that the server can control connection pooling, lifecycle, and ensure proper concurrency handling across requests.

## Acceptance Criteria

- [ ] Add `WithDB(name string, db *sql.DB, driver string)` option to `pkg/parsley`
- [ ] Injected connection is available to scripts as a `DBConnection` object
- [ ] Scripts can use standard database operators (`<=?=>`, `<=??=>`, `<=!=>`) with injected connection
- [ ] Injected connections are NOT added to Parsley's internal connection cache
- [ ] Injected connections are NOT closed by Parsley (host manages lifecycle)
- [ ] Multiple connections can be injected with different names
- [ ] Works alongside existing `SQLITE()` builtin (scripts can use both)
- [ ] Documentation updated in `pkg/parsley/README.md`

## Design

### API Addition

Add to `pkg/parsley/options.go`:

```go
// WithDB injects a database connection into the Parsley environment.
// The connection is available to scripts as a variable with the given name.
// The host application is responsible for managing the connection lifecycle
// (opening, closing, pooling). Parsley will NOT close this connection.
//
// Example:
//
//     db, _ := sql.Open("sqlite", "./app.db")
//     defer db.Close()
//     
//     result, err := parsley.EvalFile("handler.pars",
//         parsley.WithDB("db", db, "sqlite"),
//     )
//
// In Parsley script:
//
//     let user = db <=?=> "SELECT * FROM users WHERE id = 1"
//
func WithDB(name string, db *sql.DB, driver string) Option {
    return func(c *Config) {
        if c.DBConnections == nil {
            c.DBConnections = make(map[string]*DBConnectionConfig)
        }
        c.DBConnections[name] = &DBConnectionConfig{
            DB:     db,
            Driver: driver,
        }
    }
}
```

### Config Changes

Add to `pkg/parsley/options.go`:

```go
// DBConnectionConfig holds an injected database connection
type DBConnectionConfig struct {
    DB     *sql.DB
    Driver string // "sqlite", "postgres", "mysql"
}

// Config holds evaluation configuration
type Config struct {
    Env           *evaluator.Environment
    Security      *evaluator.SecurityPolicy
    Logger        evaluator.Logger
    Filename      string
    Vars          map[string]interface{}
    DBConnections map[string]*DBConnectionConfig  // NEW
}
```

### Integration with Evaluator

In `applyConfig()`, inject connections as `DBConnection` objects:

```go
// Apply database connections
for name, dbConfig := range c.DBConnections {
    conn := &evaluator.DBConnection{
        DB:            dbConfig.DB,
        Driver:        dbConfig.Driver,
        DSN:           "",  // Not applicable for injected connections
        InTransaction: false,
        LastError:     "",
        Managed:       true,  // NEW: Flag to prevent Parsley from closing
    }
    env.Set(name, conn)
}
```

### DBConnection Changes

Add `Managed` field to `pkg/evaluator/evaluator.go`:

```go
type DBConnection struct {
    DB            *sql.DB
    Driver        string
    DSN           string
    InTransaction bool
    LastError     string
    Managed       bool  // NEW: If true, Parsley won't close this connection
}
```

Update `close()` method to respect `Managed` flag:

```go
case "close":
    if conn.Managed {
        return newError("cannot close server-managed database connection")
    }
    // ... existing close logic ...
```

## Usage Example

### Host Application (Go)

```go
package main

import (
    "database/sql"
    "net/http"
    
    "github.com/sambeau/parsley/pkg/parsley"
    _ "modernc.org/sqlite"
)

func main() {
    // Server manages the database connection
    db, err := sql.Open("sqlite", "./app.db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // Configure connection pool
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        result, err := parsley.EvalFile("handler.pars",
            parsley.WithDB("db", db, "sqlite"),
            parsley.WithVar("request", map[string]interface{}{
                "path": r.URL.Path,
            }),
        )
        if err != nil {
            http.Error(w, err.Error(), 500)
            return
        }
        w.Write([]byte(result.String()))
    })
    
    http.ListenAndServe(":8080", nil)
}
```

### Parsley Script

```parsley
// handler.pars
// `db` is injected by the server - no need to call SQLITE()

let user = db <=?=> "SELECT * FROM users WHERE id = 1"

if (user) {
    <h1>Welcome, {user.name}!</h1>
} else {
    <h1>User not found</h1>
}
```

## Implementation Notes

1. **No caching**: Injected connections should NOT be added to `dbConnections` cache
2. **No closing**: Parsley must not close managed connections
3. **Thread safety**: `*sql.DB` is already safe for concurrent use
4. **Transaction scope**: Transactions on injected connections work per-request (host's responsibility to not share transaction state)

## Testing

1. Inject SQLite connection, execute queries
2. Verify `db.close()` returns error for managed connections
3. Verify injected connection works with all operators (`<=?=>`, `<=??=>`, `<=!=>`)
4. Verify injected connection works alongside `SQLITE()` builtin
5. Verify transactions work on injected connections
6. Concurrent request test (multiple goroutines using same injected `*sql.DB`)

## Out of Scope

- Connection pooling configuration (host's responsibility)
- Multiple database driver support beyond SQLite (existing limitation)
- Automatic reconnection (host's responsibility)

## Related

- Blocked by: None
- Blocks: FEAT-002 Phase 2 (Basil database support)
- Parsley library: `pkg/parsley/`
- Database implementation: `pkg/evaluator/evaluator.go` (DBConnection type)
