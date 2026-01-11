---
id: FEAT-019
title: "Basil Dev Tools"
status: complete
priority: high
created: 2025-12-03
author: "@human"
---

# FEAT-019: Basil Dev Tools

## Summary
A web-based developer tools suite for Basil accessible at `/__` routes, providing visibility into running Parsley applications. The primary feature is a structured logging system (`dev.log`) that developers can view in-browser, with logs persisted to SQLite for durability.

## User Story
As a **Parsley developer**, I want **in-browser access to structured logs and dev tools** so that **I can debug my applications without leaving the browser or parsing terminal output**.

As an **AI assistant**, I want **easy access to live server output** so that **I can help developers debug issues more effectively**.

## Acceptance Criteria

### Phase 1: Core Logging Infrastructure
- [x] `dev.log(value)` logs a value to the default log page
- [x] `dev.log(label, value)` logs with a label
- [x] Logs are persisted to SQLite database (per handler)
- [x] Logs include: filename, line number, datetime, log call representation, value
- [x] `/__/logs` displays logs in styled HTML matching error pages
- [x] `/__/logs?text` displays logs in plain text format
- [x] `/__/logs?clear` clears the log page
- [x] `dev.clearLog()` clears the current log route programmatically
- [x] Dev tools only available in dev mode
- [x] In production: `/__/*` routes return 404, `dev.*` functions are silent no-ops

### Phase 2: Log Routing
- [x] `dev.logPage(route, value)` logs to `/__/logs/{route}`
- [x] `dev.logPage(route, label, value)` logs with label to specific route
- [x] `dev.setLogRoute(route)` sets default route for subsequent `dev.log()` calls
- [x] `dev.clearLogPage(route)` clears a specific log route
- [x] `/__/logs/{route}` displays route-specific logs
- [x] `/__/logs/{route}?clear` clears that route's logs

### Phase 3: Log Levels & Enhancements
- [x] `dev.log(value, {level: "warn"})` logs with warning level
- [x] Warning logs displayed with amber/yellow styling and ⚠️ icon
- [x] Info logs (default) displayed with standard styling and ℹ️ icon
- [ ] `.json` modifier renders value as formatted JSON in log *(deferred)*
- [x] ~~Log page auto-scrolls to most recent entry~~ *(not needed: newest-first display)*

### Phase 4: Dev Tools Index & Config
- [x] `/__` index page listing available dev tools
- [x] `/__/env` shows environment information (non-sensitive)
- [x] Config option for SQLite database location/name (per handler)
- [x] Log truncation when database grows too large
- [x] Graceful handling of deleted/moved database

## Design Decisions

- **`/__` route prefix**: Matches Parsley's `__` conventions (like `__type`), quick to type, clearly system-level
- **SQLite storage**: Per-handler database; persists across requests, survives page refreshes, can be cleared
- **`dev.*` namespace**: Separates dev tools from `basil.*` namespace; clearly development-only
- **Production behavior**: Routes 404, functions no-op silently — safe to leave `dev.log()` in code
- **No real-time updates v1**: Manual refresh is acceptable; avoids WebSocket complexity
- **Two log levels only**: Info (default) and Warn; keeps it simple, covers 90% of use cases
- **Route-based log pages**: Developers self-organize; no authentication/sessions needed
- **Styling matches error pages**: Consistent Basil look and feel

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components

**New Files:**
- `server/devtools.go` — Dev tools HTTP handlers (`/__/*` routes)
- `server/devlog.go` — Log storage and retrieval (SQLite)
- `pkg/parsley/evaluator/stdlib_dev.go` — `dev.*` Parsley functions

**Modified Files:**
- `server/server.go` — Register `/__` routes in dev mode
- `server/handler.go` — Inject `dev` module into environment
- `config/config.go` — Add dev tools config options

### Dependencies
- Depends on: None (self-contained)
- Blocks: None

### Edge Cases & Constraints

1. **Dev mode only** — `dev.*` functions return errors in production mode
2. **Missing database** — Create new database file automatically
3. **Large logs** — Truncate oldest entries when exceeding size limit (configurable, default 10MB)
4. **Concurrent writes** — SQLite handles this, but use WAL mode for better concurrency
5. **Route validation** — Log routes must be URL-safe (alphanumeric, hyphens, underscores)
6. **Value serialization** — Use `repr()` for Parsley objects in log display

### Database Schema

```sql
CREATE TABLE logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    route TEXT NOT NULL DEFAULT '',
    level TEXT NOT NULL DEFAULT 'info',
    filename TEXT NOT NULL,
    line INTEGER NOT NULL,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    call_repr TEXT NOT NULL,      -- e.g., 'dev.log("users", users)'
    value_repr TEXT NOT NULL,     -- serialized value
    value_json TEXT               -- JSON if .json modifier used
);

CREATE INDEX idx_logs_route ON logs(route);
CREATE INDEX idx_logs_timestamp ON logs(timestamp);
```

### Log Page HTML Structure

```
┌─────────────────────────────────────────────┐
│  🌿 Basil Log: /route              [Clear]  │  <- Fixed header
├─────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────┐ │
│ │ 📁 handlers/users.pars:42              │ │
│ │ 🕐 2025-12-03 14:32:15                 │ │
│ │ 💻 dev.log("users", users)             │ │
│ ├─────────────────────────────────────────┤ │
│ │ [{name: "Alice", age: 30}, ...]        │ │
│ └─────────────────────────────────────────┘ │
│                                             │
│ ┌─────────────────────────────────────────┐ │
│ │ ⚠️ handlers/users.pars:58              │ │  <- Warning level
│ │ ...                                    │ │
│ └─────────────────────────────────────────┘ │
└─────────────────────────────────────────────┘
                                          ↑ Auto-scroll here
```

### Config Options

```yaml
dev:
  log_database: "data/dev_logs.db"        # SQLite path (default: dev_logs.db in project dir)
  log_max_size: 10485760                  # Max DB size in bytes (default: 10MB)
  log_truncate_percent: 25                # Delete oldest X% when truncating
```

## Implementation Notes

### Completed 2025-12-03

**Files created:**
- `server/devlog.go` - SQLite-backed log storage with auto-truncation
- `server/devtools.go` - HTTP handlers for `/__/*` routes
- `pkg/parsley/evaluator/stdlib_dev.go` - `dev.*` functions and `std/dev` stdlib module

**Files modified:**
- `server/server.go` - Dev tools initialization in dev mode
- `server/handler.go` - Inject `dev` module and `env.DevLog` into handler environment
- `pkg/parsley/evaluator/evaluator.go` - Added `DevLog` and `BasilCtx` fields to Environment
- `pkg/parsley/evaluator/stdlib_table.go` - Added `std/dev` and `std/basil` to stdlib registry
- `config/config.go` - Added `DevConfig` struct

**Key decisions during implementation:**
- Logs display newest-first (no auto-scroll needed)
- Fixed header with always-accessible Clear button
- Default database is `dev_logs.db` (persists across restarts)
- `std/dev` stdlib import allows use in modules: `let {dev} = import("std/dev")`
- DevLog read from environment at call-time (not import-time) for module support

**Deferred:**
- `.json` modifier for formatted JSON output (not MVP)

## Related
- Plan: [PLAN-011](../plans/PLAN-011.md)
