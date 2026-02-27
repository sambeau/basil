---
id: FEAT-133
title: "Database Connection Pool Optimisation"
status: draft
priority: medium
created: 2025-07-27
author: "@human"
---

# FEAT-133: Database Connection Pool Optimisation

## Summary

Basil's Parsley evaluator uses a `connectionCache` to manage `@postgres()` and `@mysql()` connections. This cache performs a synchronous `db.Ping()` health check on every retrieval — a network round-trip to the remote database server on every request before any actual query runs. This is redundant with Go's built-in `database/sql` pool, which already handles stale connection detection, retry, and rotation internally.

This spec removes the redundant health check, configures Go's pool to manage connection lifecycle properly, fixes the TTL eviction model, and adds clean shutdown of cached connections.

See: `work/reports/DATABASE-CONNECTION-MANAGEMENT.md` for the full analysis.

## Context

Basil has two database connection management paths:

1. **Managed (SQLite via `@DB`)** — Basil server opens a single `*sql.DB` at startup, injects it into every request as a `Managed: true` connection. Zero per-request overhead. This path is unaffected by this spec.

2. **Cached (`@postgres()`, `@mysql()`, standalone `@sqlite()`)** — The Parsley evaluator's package-level `dbCache` caches `*sql.DB` instances by DSN. On every cache hit, it acquires locks, checks TTL, runs `db.Ping()`, and updates `lastUsed`. The `db.Ping()` is the problem — it's a synchronous network round-trip that Go's `database/sql` makes unnecessary.

Go's `database/sql` pool (since Go 1.9+, improved in 1.15) already:
- Retries queries transparently when a pooled connection is dead (up to 2 retries)
- Closes connections that exceed `ConnMaxLifetime` or `ConnMaxIdleTime`
- Manages idle connection cleanup in a background goroutine

The `db.Ping()` on every cache retrieval duplicates this at higher cost.

## User Story

As a developer using Postgres or MySQL with Basil, I want database requests to have minimal overhead so that connection management doesn't add unnecessary latency to every request.

## Acceptance Criteria

### Phase 1: Remove Redundant Health Check

- [ ] `dbCache` is constructed with `nil` health check function instead of `db.Ping()`
- [ ] `connectionCache.get()` skips health check when the function is nil (already the case — just verify)
- [ ] Existing `connection_cache_test.go` tests still pass
- [ ] New test: verify that a cached Postgres/MySQL-style connection is returned without `Ping()` being called

### Phase 2: Configure Go Pool Lifetime Settings

- [ ] `connectionBuiltins()["postgres"]` sets `db.SetConnMaxIdleTime(5 * time.Minute)` after opening a new connection
- [ ] `connectionBuiltins()["postgres"]` sets `db.SetConnMaxLifetime(30 * time.Minute)` after opening a new connection
- [ ] `connectionBuiltins()["mysql"]` sets the same values
- [ ] `connectionBuiltins()["sqlite"]` is unchanged (SQLite connections are local, no network staleness)
- [ ] The options dictionary supports `connMaxIdleTime` and `connMaxLifetime` overrides (duration in seconds as integer)
- [ ] New test: verify default pool settings are applied to new Postgres/MySQL connections

### Phase 3: Fix TTL Eviction to Use `lastUsed`

- [ ] `connectionCache.get()` TTL check uses `cached.lastUsed` instead of `cached.createdAt`
- [ ] `connectionCache.evictStale()` TTL check uses `cached.lastUsed` instead of `cached.createdAt`
- [ ] Existing TTL test updated to reflect new behaviour
- [ ] New test: verify that a connection under active use is not evicted at the old `createdAt`-based TTL boundary
- [ ] SFTP cache (`sftpCache`) benefits from the same change automatically (uses the same `connectionCache` type)

### Phase 4: Clean Shutdown

- [ ] `Server.Run()` calls `evaluator.ClearDBConnections()` in its shutdown defer chain
- [ ] New test: verify `ClearDBConnections()` is called during graceful shutdown (or test the shutdown path includes it)

## Design Decisions

- **Remove Ping, don't replace it.** Go's `database/sql` handles stale connections internally via query-level retry. There is no need for an alternative health check mechanism. If the database server is completely unreachable, the query itself will fail with the same error that Ping would have returned — just one timeout later. The user gets an error either way; the difference is that we don't pay for a round-trip on every successful request.

- **Keep the `connectionCache` generic.** The `connectionCache[T]` type is shared between `dbCache` and `sftpCache`. SFTP has no equivalent of Go's `database/sql` retry logic, so `sftpCache` may still benefit from a health check. The cache already supports `nil` health check — we just need to pass `nil` for `dbCache` specifically.

- **`lastUsed` TTL is the right model for both caches.** A connection under active use should not be torn down. For `dbCache`, Go's `ConnMaxLifetime` handles rotation at the pool level. For `sftpCache`, idle timeout is the natural eviction trigger. Both benefit from `lastUsed`-based TTL.

- **Don't change the SFTP health check.** SFTP connections don't have Go's pool retry. The `sftpCache` health check (`conn.Client.Getwd()`) should remain. Only the `dbCache` health check is removed.

- **Pool settings are sensible defaults, not mandates.** `ConnMaxIdleTime(5m)` and `ConnMaxLifetime(30m)` are conservative defaults that work for most deployments. Users can override via the options dictionary. These values match common recommendations for Postgres and MySQL connection pools.

## Technical Context

### Affected Files

| File | Change |
|------|--------|
| `pkg/parsley/evaluator/connection_cache.go` | TTL check: `createdAt` → `lastUsed` (lines 59, in `evictStale`) |
| `pkg/parsley/evaluator/connection_cache.go` | `dbCache` init: health check `db.Ping()` → `nil` (line 200) |
| `pkg/parsley/evaluator/evaluator.go` | `connectionBuiltins()["postgres"]`: add `SetConnMaxIdleTime`, `SetConnMaxLifetime` |
| `pkg/parsley/evaluator/evaluator.go` | `connectionBuiltins()["mysql"]`: add `SetConnMaxIdleTime`, `SetConnMaxLifetime` |
| `pkg/parsley/evaluator/evaluator.go` | Both: support `connMaxIdleTime` and `connMaxLifetime` in options dict |
| `pkg/parsley/evaluator/connection_cache_test.go` | Update TTL test, add new tests |
| `server/server.go` | Add `evaluator.ClearDBConnections()` to shutdown defer chain |

### Per-Request Cost: Before and After

**Before (Postgres/MySQL):**
1. `RLock` → map lookup → `RUnlock`
2. `time.Now()` + TTL check against `createdAt`
3. `db.Ping()` — **network round-trip (0.5–3ms typical)**
4. `Lock` → update `lastUsed` → `Unlock`
5. Struct allocation

**After (Postgres/MySQL):**
1. `RLock` → map lookup → `RUnlock`
2. `time.Now()` + TTL check against `lastUsed`
3. Health check skipped (nil function)
4. `Lock` → update `lastUsed` → `Unlock`
5. Struct allocation

**Estimated saving:** 0.5–3ms per request (depends on network distance to database server).

### Risk Assessment

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Stale connection causes query failure | Very low | Go's `database/sql` retries automatically; `ConnMaxIdleTime` proactively closes idle connections |
| `lastUsed` TTL keeps zombie connections alive | Very low | `ConnMaxLifetime` rotates at the Go pool level; cache TTL is a second line of defence |
| SFTP cache behaviour changes unexpectedly | Low | SFTP health check is not changed; only the TTL base changes, which is actually desirable for SFTP too |

### Edge Cases

1. **Database server restarts** — After this change, the first query after a server restart will fail on the stale connection, Go retries with a fresh connection, query succeeds. No user-visible error. Before this change, `Ping()` would have caught it — but at the cost of pinging on every single request.

2. **Options dict interaction** — If a user passes `{connMaxLifetime: 3600}` (1 hour), this overrides the default 30-minute `ConnMaxLifetime` on the Go pool. The `dbCache` TTL (also 30 minutes, now based on `lastUsed`) is a separate layer. The connection could be evicted by either mechanism — whichever fires first. This is fine; both are cleanup mechanisms.

3. **`:memory:` SQLite in standalone Parsley** — Unchanged. These go through `connectionBuiltins()["sqlite"]`, which doesn't set `ConnMaxIdleTime`/`ConnMaxLifetime` (not relevant for in-process SQLite). The cache still holds them for process lifetime or until TTL.

## Implementation Notes

*To be filled in during implementation.*

## Related

- Report: `work/reports/DATABASE-CONNECTION-MANAGEMENT.md`
- `pkg/parsley/evaluator/connection_cache.go` — the generic cache implementation
- `pkg/parsley/evaluator/evaluator.go` — `connectionBuiltins()`, `dbCache` init, `ClearDBConnections()`
- `server/server.go` — `Server.Run()` shutdown path