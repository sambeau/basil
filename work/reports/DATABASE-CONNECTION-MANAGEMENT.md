# Database Connection Management Analysis

**Date:** 2025-01-27
**Scope:** Basil server (`server/`), Parsley evaluator (`pkg/parsley/evaluator/`), connection lifecycle and performance

## Executive Summary

Basil has two database connection management systems that serve different roles:

1. **Basil server-managed connections** — SQLite only. Opened once at startup, reused for the server's lifetime, injected into every request as a `Managed` connection that scripts cannot close. This is the gold-standard path.

2. **Parsley evaluator `dbCache`** — Used by `@sqlite()`, `@postgres()`, and `@mysql()` connection constructors. A generic cache with 30-minute TTL, LRU eviction, and per-retrieval health checks. This is the path used for Postgres and MySQL in Basil handlers, and for all databases in standalone Parsley scripts.

SQLite is well-served. Postgres and MySQL work correctly but carry unnecessary per-request overhead from a redundant health check that duplicates what Go's `database/sql` pool already provides.

## Architecture

### Connection Flow: SQLite via `@DB` (Basil Server)

```
Server startup
  └─ initSQLite()
       └─ sql.Open("sqlite", path)          ← opened ONCE
       └─ db.Ping()                         ← verified ONCE
       └─ SetMaxOpenConns(1)                ← tuned for SQLite
       └─ SetConnMaxLifetime(0)             ← never expires
       └─ s.db = db                         ← stored on Server struct

Per request
  └─ handler.ServeHTTP()
       └─ NewManagedDBConnection(s.db, "sqlite")   ← lightweight wrapper
       └─ env.ServerDB = conn                       ← pointer into environment
       └─ Parsley script uses @DB
            └─ resolveDBLiteral()
                 └─ return env.ServerDB              ← direct pointer, zero overhead
```

**Cost per request:** One struct allocation (~64 bytes). No I/O. No locking. No cache lookup.

### Connection Flow: Postgres/MySQL via `@postgres()`/`@mysql()` (Any Context)

```
First request
  └─ connectionBuiltins()["postgres"]
       └─ dbCache.get("postgres:" + dsn)    ← cache miss
       └─ sql.Open("postgres", dsn)         ← connection opened
       └─ db.Ping()                         ← verified
       └─ dbCache.put(cacheKey, db)         ← cached

Subsequent requests (within 30-minute TTL)
  └─ connectionBuiltins()["postgres"]
       └─ dbCache.get("postgres:" + dsn)
            └─ RLock, map lookup, RUnlock    ← lock acquisition
            └─ time.Now(), TTL check         ← clock read
            └─ db.Ping()                     ← NETWORK ROUND-TRIP
            └─ Lock, update lastUsed, Unlock ← second lock acquisition
       └─ return &DBConnection{DB: db, ...}  ← wrapper allocation

Every 30 minutes (regardless of activity)
  └─ TTL expires (checked on next get() or by cleanup goroutine)
       └─ db.Close()                         ← connection torn down
       └─ Next request reopens              ← sql.Open + Ping again
```

**Cost per request:** Two lock acquisitions, one `time.Now()` call, one `db.Ping()` network round-trip, one struct allocation.

## The `db.Ping()` Problem

The `connectionCache.get()` method performs a health check on every retrieval:

```go
// connection_cache.go lines 72-82
if c.healthCheck != nil {
    if err := c.healthCheck(cached.conn); err != nil {
        // evict and return not-found
    }
}
```

For the `dbCache`, this health check is `db.Ping()`:

```go
// connection_cache.go lines 200-202
func(db *sql.DB) error {
    return db.Ping()
},
```

For Postgres and MySQL, `db.Ping()` sends a protocol-level ping to the remote server and waits for a response. Typical costs:

| Scenario | Ping Latency |
|----------|-------------|
| Same host | ~0.1–0.3ms |
| Same data centre / VPC | ~0.5–1ms |
| Cross availability zone | ~1–3ms |
| Cross region | ~10–50ms |

This happens **before any actual query** on every request that touches the database. Under load, this is pure waste — if the connection were dead, the subsequent query would fail anyway and could be retried.

### Why This Is Redundant

Go's `database/sql` package already manages connection health internally. When you call `db.Query()` or `db.Exec()`:

1. **Connection checkout:** The pool retrieves an idle connection (or opens a new one).
2. **Staleness check:** If `ConnMaxIdleTime` or `ConnMaxLifetime` is set, expired connections are discarded transparently.
3. **Retry on failure:** If a connection is broken, `database/sql` automatically retries with a fresh connection from the pool (up to `maxBadConnRetries`, which is 2 by default).
4. **Lazy validation:** Since Go 1.15, `SetConnMaxIdleTime` causes the pool to close connections that have been idle too long, without any explicit ping.

This means that even if a Postgres connection goes stale between requests, the first `db.Query()` in the handler will:
- Get the stale connection from the pool
- Fail on the network write/read
- Automatically retry with a new connection
- Succeed transparently

The application code never sees the failure. The `db.Ping()` in our cache is doing the same work that `database/sql` would do lazily — but we're paying for it eagerly on every single request.

### What Go's Pool Needs From Us

For `database/sql` to manage health well on its own, it benefits from two settings that Basil currently does **not** configure for Postgres/MySQL connections:

| Setting | Purpose | Recommended |
|---------|---------|-------------|
| `SetConnMaxIdleTime` | Close connections idle for too long (prevents stale TCP) | 5–10 minutes |
| `SetConnMaxLifetime` | Rotate connections periodically (respects server-side limits) | 30–60 minutes |

These are never set for Postgres/MySQL (the `connectionBuiltins` code only handles `maxOpenConns` and `maxIdleConns` from the options dict). Without `ConnMaxIdleTime`, Go's pool will hold idle connections indefinitely — but our `dbCache` TTL partially compensates by closing the entire `*sql.DB` after 30 minutes.

## The TTL Eviction Model

The `connectionCache` evicts entries based on `createdAt`, not `lastUsed`:

```go
// connection_cache.go line 59
if now.Sub(cached.createdAt) > c.ttl {
```

This means a Postgres connection under constant heavy traffic is torn down and reopened every 30 minutes — the same fate as a completely idle connection. The `lastUsed` field is tracked but only used for LRU eviction when the cache is full (100 entries), not for TTL.

For Postgres/MySQL this causes:
- A brief latency spike every 30 minutes as the connection is re-established
- A new TLS handshake if the connection uses TLS
- Any server-side prepared statements are lost

This isn't catastrophic — Go's pool smooths it over — but it's unnecessary churn compared to letting `database/sql` manage lifetime via `SetConnMaxLifetime`.

## The `db.close()` Cache Inconsistency

When a Parsley script explicitly calls `db.close()` on a non-managed connection:

```go
// eval_method_dispatch.go lines 54-56
// Note: We don't remove from cache on explicit close, as the cache
// handles TTL and cleanup automatically. Manual close just closes
// the database connection.
```

The `*sql.DB` is closed, but the cache entry remains. The next request's `dbCache.get()` will find the entry, attempt `db.Ping()` on the closed connection, fail, evict it, and the caller will reopen. This works but adds one wasted ping attempt after a manual close.

In the Basil server context this is a non-issue — handlers use `@DB` (managed, can't close) or `@postgres()`/`@mysql()` (no reason to close mid-request). It's only relevant for standalone Parsley scripts, where the process exits anyway.

## Shutdown Gap

On server shutdown, Basil closes its managed SQLite connection:

```go
// server.go lines 953-957
if s.db != nil {
    defer func() {
        s.logInfo("closing database connection")
        s.db.Close()
    }()
}
```

But `evaluator.ClearDBConnections()` is never called. Any Postgres/MySQL connections opened via `@postgres()`/`@mysql()` in Parsley handlers remain in the package-level `dbCache` and are only cleaned up when the process exits. This is fine in practice — process exit closes all file descriptors — but it means the remote database server sees an unclean disconnect rather than a graceful close, which can leave temporary resources (prepared statements, advisory locks) lingering on the server until its own idle timeout fires.

## Recommendations

### R1: Remove Per-Retrieval `db.Ping()` from `dbCache` (Performance)

**Impact: Eliminates one network round-trip per database request for Postgres/MySQL.**

Pass `nil` for the health check function when constructing `dbCache`, matching what the managed SQLite path effectively does (no health check at all):

```go
var dbCache = newConnectionCache[*sql.DB](
    100,
    30*time.Minute,
    nil,  // no health check — database/sql handles retry internally
    func(db *sql.DB) error { return db.Close() },
    nil,
)
```

Go's `database/sql` will handle stale connections transparently via its internal retry mechanism. This is the single highest-value change.

**Risk:** Very low. Go's retry logic has been stable since Go 1.9. The only scenario where Ping-on-get catches something that query-retry doesn't is if the *entire server* is unreachable — but in that case the query itself would fail with the same error, just one network timeout later. The user gets an error either way.

### R2: Set `ConnMaxIdleTime` and `ConnMaxLifetime` on Postgres/MySQL (Correctness)

**Impact: Lets Go's pool manage connection health properly, replacing our TTL eviction.**

In `connectionBuiltins()`, after opening Postgres/MySQL connections:

```go
db.SetConnMaxIdleTime(5 * time.Minute)   // close connections idle > 5 min
db.SetConnMaxLifetime(30 * time.Minute)   // rotate connections every 30 min
```

This makes Go's pool aware of connection age and idleness, enabling it to proactively close stale connections in the background without any explicit health checks. These can also be exposed via the options dict for user override.

### R3: Switch TTL Check to `lastUsed` (Efficiency)

**Impact: Prevents unnecessary teardown of active connections.**

Change the TTL check from `createdAt` to `lastUsed`:

```go
// Before:
if now.Sub(cached.createdAt) > c.ttl {

// After:
if now.Sub(cached.lastUsed) > c.ttl {
```

This way, a Postgres connection under active use stays open indefinitely (with Go's `ConnMaxLifetime` handling rotation), while idle connections are cleaned up after 30 minutes of disuse.

Note: the SFTP cache uses the same `connectionCache` type, so this change would apply there too — which is actually desirable, since SFTP connections under active use shouldn't be torn down either.

### R4: Call `ClearDBConnections()` on Server Shutdown (Hygiene)

**Impact: Clean disconnect from remote databases on shutdown.**

Add to the shutdown defer chain in `Server.Run()`:

```go
defer evaluator.ClearDBConnections()
```

This ensures Postgres/MySQL servers see a clean `Connection: close` rather than a TCP RST, allowing them to release server-side resources immediately.

### Summary Table

| # | Change | Effort | Impact | Risk |
|---|--------|--------|--------|------|
| R1 | Remove `db.Ping()` from cache get | Trivial | High — eliminates per-request network round-trip | Very low |
| R2 | Set `ConnMaxIdleTime`/`ConnMaxLifetime` | Small | Medium — proper pool hygiene for remote DBs | Very low |
| R3 | TTL based on `lastUsed` not `createdAt` | Trivial | Low-medium — prevents unnecessary reconnection churn | Very low (also benefits SFTP) |
| R4 | `ClearDBConnections()` on shutdown | Trivial | Low — clean remote disconnects | None |

## Assessment

The current architecture is sound. SQLite gets first-class treatment through the managed connection path, exactly as intended. Postgres and MySQL get a workable caching layer that correctly avoids opening a new connection per request.

The main inefficiency is the per-retrieval `db.Ping()`, which is a design pattern from before Go's `database/sql` had robust built-in connection health management. Removing it and letting Go's pool do what it was designed to do is the single most impactful change — it eliminates a synchronous network round-trip from the hot path while actually *improving* reliability, since Go's pool retry is more sophisticated than a single ping check.

R1 and R3 together would bring Postgres/MySQL performance close to the managed SQLite path in terms of per-request overhead: a cache lookup with two lock acquisitions and a map read, no I/O. The remaining difference — that the `*sql.DB` pool is managed by Go rather than held as a single persistent connection — is exactly right for networked databases that benefit from connection pooling.