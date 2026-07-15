---
id: man-bas-database
title: "Database"
system: basil
type: feature
name: database
created: 2026-07-12
version: 1.0.0-alpha.3
author: "@sam"
keywords:
  - database
  - sqlite
  - "@DB"
  - query
  - sql
  - inspector
  - postgres
  - mysql
---

# Database

Basil ships with an in-process SQLite database — no separate server to run. Configure a path, and every handler can query it as `@DB`.

```yaml
database:
  path: ./db/data.db
```

```parsley
let users = @DB <=??=> "SELECT * FROM users ORDER BY name"

<ul>
    for (user in users) {
        <li>user.name</li>
    }
</ul>
```

`@DB` is a managed connection: Basil opens it (WAL mode, sensible timeouts), shares it across handlers, and handlers cannot close it.

## Query Operators

The database operators visually distinguish reads from writes and one row from many — see the [Parsley database docs](../../parsley/manual/features/database.md) for the full set:

```parsley
let user  = @DB <=?=>  "SELECT * FROM users WHERE id = ?" <- [42]   // one row (or null)
let users = @DB <=??=> "SELECT * FROM users"                        // all rows (table)
@DB <=!=> "INSERT INTO users (name) VALUES (?)" <- ["Alice"]        // execute
```

Prefer the declarative [Query DSL](../../parsley/manual/features/query-dsl.md) for schema-bound tables:

```parsley
@schema User {
    id: int
    name: string
    status: string
}

let Users = @DB.bind(User, "users")
let active = @query(Users | status == "active" ??-> name, email)
```

## The Database Inspector

In [dev mode](dev-tools.md), Basil serves a web-based database inspector at `/__/db` — browse tables, run queries, and download or upload CSVs while you develop.

## Other Databases

`@DB` is the configured SQLite database, but handlers can open connections to anything:

```parsley
let pg = @postgres("postgres://user:pass@host:5432/dbname")
let my = @mysql("user:pass@tcp(host:3306)/dbname")
let mem = @sqlite(":memory:")
```

## See Also

- [Parsley: Database](../../parsley/manual/features/database.md) — connection literals, operators, transactions, connection methods, and the SQL tag in full
- [Parsley: Query DSL](../../parsley/manual/features/query-dsl.md) — `@query`, `@insert`, `@update`, `@delete`
