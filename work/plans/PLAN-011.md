# PLAN-011: Basil Dev Tools Implementation

**Spec:** [FEAT-019](../specs/FEAT-019.md)
**Status:** Complete
**Created:** 2025-12-03
**Target:** Next release

## Overview

Implement a web-based developer tools suite for Basil accessible at `/__` routes, with a structured logging system (`dev.*`) that persists to SQLite and displays in-browser.

## Prerequisites

- [x] Existing `/__livereload` route pattern in `server/server.go`
- [x] SQLite database setup pattern in `server/server.go` (initSQLite)
- [x] Parsley stdlib module pattern in `pkg/parsley/evaluator/stdlib_table.go`
- [x] Environment injection pattern in `server/handler.go` (buildBasilContext)

## Implementation Tasks

### Phase 1: Core Logging Infrastructure

#### Task 1.1: Dev Log Storage (SQLite)
**Files:** `server/devlog.go` (new)
**Effort:** Medium
**Steps:**
1. Create `DevLog` struct to manage dev log database
2. Implement SQLite schema creation (logs table with indexes)
3. Add `Log()` method for writing log entries
4. Add `GetLogs()` method for retrieving logs (with optional route filter)
5. Add `ClearLogs()` method for clearing logs (with optional route filter)
6. Use WAL mode for better concurrency
7. Add size-based truncation (delete oldest 25% when exceeding limit)

**Tests:**
- [x] TestDevLogCreate - database creation
- [x] TestDevLogWrite - writing log entries
- [x] TestDevLogRead - reading logs back
- [x] TestDevLogClear - clearing logs
- [x] TestDevLogTruncation - size-based truncation

#### Task 1.2: Dev Module for Parsley
**Files:** `pkg/parsley/evaluator/stdlib_dev.go` (new), `pkg/parsley/evaluator/stdlib_table.go` (modify)
**Effort:** Medium
**Steps:**
1. Create `DevModule` type (callable, has methods)
2. Implement `dev.log(value)` and `dev.log(label, value)`
3. Implement `dev.clearLog()`
4. Register in `getStdlibModules()` 
5. Make dev functions no-op when `DevLog` is nil (production mode)
6. Capture filename, line number from environment context

**Tests:**
- [x] TestDevLog - basic logging
- [x] TestDevLogWithLabel - labeled logging
- [x] TestDevClearLog - clearing logs
- [x] TestDevNoOpInProduction - silent no-ops when disabled

#### Task 1.3: Inject Dev Module into Handler
**Files:** `server/handler.go`, `server/server.go`
**Effort:** Small
**Steps:**
1. Add `devLog *DevLog` field to Server struct
2. Initialize devLog in `New()` only when `config.Server.Dev == true`
3. Add "dev" to `buildBasilContext()` using `ast.ObjectLiteralExpression` pattern
4. Pass devLog reference through to evaluator context

**Tests:**
- [x] TestDevModuleInjectedInDevMode
- [x] TestDevModuleNotInjectedInProduction

#### Task 1.4: Dev Tools HTTP Handlers
**Files:** `server/devtools.go` (new), `server/server.go`
**Effort:** Medium
**Steps:**
1. Create `devToolsHandler` for `/__/logs` route
2. Implement HTML log display (styled like error pages)
3. Implement `?text` query param for plain text output
4. Implement `?clear` query param for clearing logs
5. Register `/__/logs` route in dev mode only
6. Return 404 for `/__/*` routes in production

**Tests:**
- [x] TestDevToolsLogsHTML - HTML output
- [x] TestDevToolsLogsText - plain text output
- [x] TestDevToolsClear - clearing via query param
- [x] TestDevTools404InProduction - 404 in production mode

### Phase 2: Log Routing

#### Task 2.1: Route-Specific Logging
**Files:** `pkg/parsley/evaluator/stdlib_dev.go`, `server/devlog.go`
**Effort:** Medium
**Steps:**
1. Add `route` field to log entries
2. Implement `dev.logPage(route, value)` and `dev.logPage(route, label, value)`
3. Implement `dev.setLogRoute(route)` to set default route
4. Implement `dev.clearLogPage(route)` for route-specific clearing
5. Add route validation (alphanumeric, hyphens, underscores)

**Tests:**
- [ ] TestDevLogPage - route-specific logging
- [ ] TestDevSetLogRoute - default route setting
- [ ] TestDevClearLogPage - route-specific clearing
- [ ] TestDevRouteValidation - invalid routes rejected

#### Task 2.2: Route-Specific HTTP Endpoints
**Files:** `server/devtools.go`
**Effort:** Small
**Steps:**
1. Add `/__/logs/{route}` route pattern
2. Filter logs by route in display
3. Support `?clear` for route-specific clearing

**Tests:**
- [x] TestDevToolsRouteSpecificLogs (TestDevToolsLogsRoute)
- [x] TestDevToolsRouteClear (via ?clear param)

### Phase 3: Log Levels & Enhancements

#### Task 3.1: Log Levels
**Files:** `pkg/parsley/evaluator/stdlib_dev.go`, `server/devlog.go`, `server/devtools.go`
**Effort:** Small
**Steps:**
1. Add `level` field to log entries (default: "info")
2. Support options dict: `dev.log(value, {level: "warn"})`
3. Style warnings with amber/yellow and ⚠️ icon
4. Style info with standard styling and ℹ️ icon

**Tests:**
- [x] TestDevToolsWarnLevel - warn level styling
- [x] TestDevToolsWarnLevel - ⚠️ icon for warnings

#### Task 3.2: UI Enhancements
**Files:** `server/devtools.go`
**Effort:** Small
**Steps:**
1. ~~Add `.json` modifier for formatted JSON display~~ (deferred)
2. Auto-scroll to most recent entry ✅
3. Add clear button in header ✅
4. Show log count in header ✅

**Tests:**
- [x] TestDevToolsLogsClear - clear button works
- [x] TestDevToolsLogsHTML - count displayed

### Phase 4: Dev Tools Index & Config

#### Task 4.1: Dev Tools Index Page
**Files:** `server/devtools.go`
**Effort:** Small
**Steps:**
1. Create `/__` index page with links to available tools
2. List: Logs, Env (future)
3. Match Basil styling

**Tests:**
- [x] TestDevToolsIndex

#### Task 4.2: Environment Info Page
**Files:** `server/devtools.go`
**Effort:** Small
**Steps:**
1. Create `/__/env` page showing:
   - Basil version
   - Go version
   - Config file path
   - Handler count
   - Dev mode status
2. Do NOT show sensitive info (secrets, full paths)

**Tests:**
- [x] TestDevToolsEnv
- [x] TestDevToolsEnvNoSecrets

#### Task 4.3: Configuration Options
**Files:** `config/config.go`, `config/load.go`, `server/server.go`
**Effort:** Small
**Steps:**
1. Add `DevConfig` struct to config
2. Add `log_database` path option
3. Add `log_max_size` limit option (default 10MB)
4. Add `log_truncate_percent` option (default 25%)
5. Parse and validate in config loading

**Tests:**
- [x] TestDevConfigDefaults
- [x] TestParseSize - size string parsing

## Validation Checklist

Before marking complete:

- [ ] All tests pass (`make test`)
- [ ] Build succeeds (`make build`)
- [ ] Lint passes (`golangci-lint run`)
- [ ] Manual testing:
  - [ ] `dev.log()` writes to database
  - [ ] `/__/logs` displays logs correctly
  - [ ] `?text` returns plain text
  - [ ] `?clear` clears logs
  - [ ] Production mode: `/__/*` returns 404
  - [ ] Production mode: `dev.*` are silent no-ops
- [ ] Documentation updated (CHEATSHEET, guide)
- [ ] CHANGELOG updated
- [ ] Spec marked complete

## Progress Log

| Date | Phase | Notes |
|------|-------|-------|
| 2025-12-03 | Planning | Created implementation plan |
| 2025-12-03 | Phase 1 | Completed - Core logging infrastructure (Tasks 1.1-1.4) |
| 2025-12-03 | Phase 2 | Completed - Log routing (already in Phase 1) |
| 2025-12-03 | Phase 3 | Completed - Log levels, UI enhancements |
| 2025-12-03 | Phase 4 | Completed - Index page, env page, config options |

## Deferred Items

Items discovered during implementation that should be added to BACKLOG.md:

- (none yet)

## Technical Notes

### Database Location
Default: `dev_logs_{datetime}.db` in handler directory
Rationale: Per-handler, clearly dev-only, easy to clean up

### Log Entry Structure
```go
type LogEntry struct {
    ID        int64
    Route     string    // Log page route (empty = default)
    Level     string    // "info" or "warn"
    Filename  string    // Source file path
    Line      int       // Source line number
    Timestamp time.Time
    CallRepr  string    // e.g., 'dev.log("users", users)'
    ValueRepr string    // Serialized value
    ValueJSON string    // JSON if .json modifier used
}
```

### Production Safety
- `dev.*` functions check for `env.DevLog != nil`
- If nil, return NULL silently (no error, no output)
- `/__/*` routes not registered in production mode
- Any direct access returns 404

### HTML Template
Reuse existing error page styling from `server/handler.go` (`handleScriptError`).
