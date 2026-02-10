---
id: PLAN-083
feature: FEAT-107
title: "Implementation Plan: Database Driver Support (PostgreSQL & MySQL)"
status: draft
created: 2025-01-15
---

# Implementation Plan: FEAT-107

## Overview
Add PostgreSQL and MySQL driver dependencies to enable the existing `@postgres()` and `@mysql()` connection functions. The evaluator code is already complete—only driver registration and documentation are needed.

## Prerequisites
- [x] Evaluator code exists (`evaluator.go` lines 1877-2020)
- [x] Connection caching implemented (`connection_cache.go`)
- [x] SQLite driver pattern established

## Tasks

### Task 1: Add Driver Dependencies
**Files**: `go.mod`
**Estimated effort**: Small

Steps:
1. Add `github.com/lib/pq` (PostgreSQL driver)
2. Add `github.com/go-sql-driver/mysql` (MySQL driver)
3. Run `go mod tidy` to resolve dependencies

Tests:
- `go build ./...` succeeds

---

### Task 2: Create Driver Registration File
**Files**: `pkg/parsley/evaluator/drivers.go` (new file)
**Estimated effort**: Small

Steps:
1. Create `drivers.go` with blank imports for both drivers
2. Imports register drivers via `init()` side-effects

Content:
```go
package evaluator

// Database driver imports for side-effect registration with database/sql.
// These drivers are required for @postgres() and @mysql() to function.

import (
    _ "github.com/lib/pq"              // PostgreSQL driver
    _ "github.com/go-sql-driver/mysql" // MySQL driver
)
```

Tests:
- `go build ./...` succeeds
- Unit test verifies `sql.Drivers()` includes "postgres" and "mysql"

---

### Task 3: Add Driver Registration Test
**Files**: `pkg/parsley/evaluator/drivers_test.go` (new file)
**Estimated effort**: Small

Steps:
1. Create test file that verifies driver registration
2. Test that `sql.Drivers()` returns both "postgres" and "mysql"

Tests:
- `go test ./pkg/parsley/evaluator/... -run TestDriverRegistration`

---

### Task 4: Update Documentation
**Files**: `docs/parsley/reference.md`
**Estimated effort**: Medium

Steps:
1. Add new section "6.15 Database Connections" after Serialization (or find appropriate location)
2. Document `@sqlite()`, `@postgres()`, and `@mysql()` functions
3. Include DSN format examples for each driver
4. Document connection options (`maxOpenConns`, `maxIdleConns`)

Content to add:
- SQLite: `@sqlite("path/to/database.db")` or `@sqlite(":memory:")`
- PostgreSQL: `@postgres("postgres://user:password@localhost:5432/dbname?sslmode=disable")`
- MySQL: `@mysql("user:password@tcp(localhost:3306)/dbname")`
- Connection options: `@postgres(url, {maxOpenConns: 10, maxIdleConns: 5})`

---

### Task 5: Create Integration Test File (Optional/CI-Skipped)
**Files**: `pkg/parsley/tests/database_drivers_test.go` (new file)
**Estimated effort**: Medium

Steps:
1. Create integration test file with build tag `//go:build integration`
2. Add tests for PostgreSQL connectivity (skipped without `POSTGRES_DSN` env var)
3. Add tests for MySQL connectivity (skipped without `MYSQL_DSN` env var)
4. Test basic query execution for each driver

Tests:
- Tests skip gracefully when databases unavailable
- Tests pass when run with Docker Compose (`docker-compose.test.yml`)

---

### Task 6: Create Docker Compose Test Configuration (Optional)
**Files**: `docker-compose.test.yml` (new file at repo root)
**Estimated effort**: Small

Steps:
1. Create Docker Compose file with PostgreSQL 16 and MySQL 8 services
2. Configure test credentials matching test expectations

---

## Validation Checklist
- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] Documentation updated in `docs/parsley/reference.md`
- [ ] `work/BACKLOG.md` updated with deferrals (if any)
- [ ] `go mod tidy` run after adding dependencies

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-01-15 | Task 1: Add dependencies | ✅ Complete | Added mysql v1.8.1 and pq v1.10.9 |
| 2025-01-15 | Task 2: Create drivers.go | ✅ Complete | Created with blank imports |
| 2025-01-15 | Task 3: Add driver test | ✅ Complete | Test passes, linter clean |
| 2025-01-15 | Task 4: Update docs | ✅ Complete | Added section 6.13 to reference.md |
| | Task 5: Integration tests | ⬜ Deferred | Optional - see backlog |
| | Task 6: Docker Compose | ⬜ Deferred | Optional - see backlog |

## Deferred Items
Items to add to work/BACKLOG.md after implementation:
- Integration tests for PostgreSQL (Task 5) - requires running PostgreSQL server
- Integration tests for MySQL (Task 5) - requires running MySQL server
- Docker Compose test configuration (Task 6) - useful for local validation but not required for core functionality

## Implementation Notes

### Completed Work
- Added `github.com/lib/pq v1.10.9` to `go.mod`
- Added `github.com/go-sql-driver/mysql v1.8.1` to `go.mod`
- Created `pkg/parsley/evaluator/drivers.go` with blank imports for driver registration
- Created `pkg/parsley/evaluator/drivers_test.go` to verify driver registration
- Updated `docs/parsley/reference.md` with new section 6.13 "Database Connections"
- All tests pass, linter clean, build succeeds

### Files Changed
- `go.mod` - Added driver dependencies
- `go.sum` - Updated with driver checksums
- `pkg/parsley/evaluator/drivers.go` - New file (9 lines)
- `pkg/parsley/evaluator/drivers_test.go` - New file (26 lines)
- `docs/parsley/reference.md` - Added 86 lines documenting database connections

### Testing
- `TestDriverRegistration` verifies all three drivers are registered
- Existing database tests continue to pass
- Build and linter both clean

### Notes
- The existing `@postgres()` and `@mysql()` implementations in `evaluator.go` are **complete and correct**
- No changes needed to evaluator logic—only driver registration
- Integration tests are deferred as they require running database servers

## Related
- Spec: `work/specs/FEAT-107.md`
- Existing code: `pkg/parsley/evaluator/evaluator.go` (lines 1877-2020)
- Connection cache: `pkg/parsley/evaluator/connection_cache.go`
