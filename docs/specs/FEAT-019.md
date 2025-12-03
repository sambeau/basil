---
id: FEAT-019
title: "Basil Dev Tools"
status: draft
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
- [ ] `dev.log(value)` logs a value to the default log page
- [ ] `dev.log(label, value)` logs with a label
- [ ] Logs are persisted to SQLite database (per handler)
- [ ] Logs include: filename, line number, datetime, log call representation, value
- [ ] `/__/logs` displays logs in styled HTML matching error pages
- [ ] `/__/logs?text` displays logs in plain text format
- [ ] `/__/logs?clear` clears the log page
- [ ] `dev.clearLog()` clears the current log route programmatically
- [ ] Dev tools only available in dev mode
- [ ] In production: `/__/*` routes return 404, `dev.*` functions are silent no-ops

### Phase 2: Log Routing
- [ ] `dev.logPage(route, value)` logs to `/__/logs/{route}`
- [ ] `dev.logPage(route, label, value)` logs with label to specific route
- [ ] `dev.setLogRoute(route)` sets default route for subsequent `dev.log()` calls
- [ ] `dev.clearLogPage(route)` clears a specific log route
- [ ] `/__/logs/{route}` displays route-specific logs
- [ ] `/__/logs/{route}?clear` clears that route's logs

### Phase 3: Log Levels & Enhancements
- [ ] `dev.log(value, {level: "warn"})` logs with warning level
- [ ] Warning logs displayed with amber/yellow styling and ⚠️ icon
- [ ] Info logs (default) displayed with standard styling and ℹ️ icon
- [ ] `.json` modifier renders value as formatted JSON in log
- [ ] Log page auto-scrolls to most recent entry

### Phase 4: Dev Tools Index & Config
- [ ] `/__` index page listing available dev tools
- [ ] `/__/env` shows environment information (non-sensitive)
- [ ] Config option for SQLite database location/name (per handler)
- [ ] Log truncation when database grows too large
- [ ] Graceful handling of deleted/moved database

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
  enabled: true                           # Enable dev mode (default: false in prod)
  log_database: "data/dev_logs.db"        # SQLite path (default: dev_logs_{datetime}.db)
  log_max_size: 10485760                  # Max DB size in bytes (default: 10MB)
  log_truncate_percent: 25                # Delete oldest X% when truncating
```

## Implementation Notes
*Added during/after implementation*

## Related
- Plan: [PLAN-011](../plans/PLAN-011.md)
