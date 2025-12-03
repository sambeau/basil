---
id: PLAN-013
feature: FEAT-021
title: "Implementation Plan for SQLite Dev Tools"
status: draft
created: 2025-12-03
---

# Implementation Plan: FEAT-021 SQLite Dev Tools

## Overview
Add a web interface at `/__/db` for viewing database structure and managing tables via CSV import/export. Dev-mode only.

## Prerequisites
- [x] FEAT-020 (per-developer config) — completed
- [x] Existing devtools.go structure to extend

## Tasks

### Task 1: Database Info Functions
**Files**: `server/devtools_db.go` (new)
**Estimated effort**: Small

Create helper functions to query SQLite metadata:

```go
// TableInfo represents a database table
type TableInfo struct {
    Name     string
    Columns  []ColumnInfo
    RowCount int64
}

type ColumnInfo struct {
    Name    string
    Type    string
    NotNull bool
    PK      bool
}

// getTableList returns all user tables (excludes sqlite_* tables)
func getTableList(db *sql.DB) ([]string, error)

// getTableInfo returns structure and row count for a table  
func getTableInfo(db *sql.DB, tableName string) (*TableInfo, error)

// getTableColumns returns column info using PRAGMA table_info
func getTableColumns(db *sql.DB, tableName string) ([]ColumnInfo, error)
```

Tests:
- Test getTableList returns user tables, excludes sqlite_* tables
- Test getTableInfo returns correct structure and row count
- Test empty database returns empty list

---

### Task 2: CSV Export
**Files**: `server/devtools_db.go`
**Estimated effort**: Small

```go
// exportTableCSV writes table data as CSV to the writer
// Excludes BLOB columns
func exportTableCSV(db *sql.DB, tableName string, w io.Writer) error
```

Steps:
1. Get column info, filter out BLOB columns
2. Query all rows: `SELECT col1, col2, ... FROM tablename`
3. Write header row
4. Write data rows using encoding/csv

Tests:
- Test export produces valid CSV with headers
- Test empty table exports headers only
- Test BLOB columns are excluded
- Test special characters (commas, quotes, newlines) are escaped

---

### Task 3: CSV Import with Type Inference
**Files**: `server/devtools_db.go`
**Estimated effort**: Medium

```go
// InferredColumn represents a column with inferred type
type InferredColumn struct {
    Name string
    Type string // INTEGER, REAL, or TEXT
}

// inferColumnTypes scans all values and returns inferred types
func inferColumnTypes(headers []string, rows [][]string) ([]InferredColumn, error)

// replaceTableFromCSV drops existing table, creates new one from CSV
func replaceTableFromCSV(db *sql.DB, tableName string, r io.Reader) error
```

Type inference rules:
1. For each column, collect all non-empty values
2. Try parsing all as int64 → INTEGER
3. Else try parsing all as float64 → REAL
4. Else → TEXT
5. Empty strings become NULL

Steps:
1. Parse CSV using encoding/csv
2. Validate table name (alphanumeric + underscore)
3. Infer column types from data
4. Begin transaction
5. DROP TABLE IF EXISTS
6. CREATE TABLE with inferred types
7. INSERT all rows
8. Commit transaction

Tests:
- Test INTEGER inference (all ints)
- Test REAL inference (mixed int/float)
- Test TEXT inference (any non-numeric)
- Test empty values become NULL
- Test invalid table name rejected
- Test transaction rollback on error

---

### Task 4: Create New Table
**Files**: `server/devtools_db.go`
**Estimated effort**: Small

```go
// createEmptyTable creates a new table with id INTEGER column and one row
func createEmptyTable(db *sql.DB, tableName string) error
```

Steps:
1. Validate table name
2. CREATE TABLE tablename (id INTEGER)
3. INSERT INTO tablename (id) VALUES (0)

Tests:
- Test creates table with correct structure
- Test inserts initial row
- Test invalid name rejected
- Test duplicate name returns error

---

### Task 5: HTTP Handlers
**Files**: `server/devtools.go`
**Estimated effort**: Medium

Add route handling in ServeHTTP switch:

```go
case path == "/__/db" || path == "/__/db/":
    h.serveDB(w, r)
case strings.HasPrefix(path, "/__/db/download/"):
    tableName := strings.TrimPrefix(path, "/__/db/download/")
    h.serveDBDownload(w, r, tableName)
case strings.HasPrefix(path, "/__/db/upload/"):
    tableName := strings.TrimPrefix(path, "/__/db/upload/")
    h.serveDBUpload(w, r, tableName)
case path == "/__/db/create":
    h.serveDBCreate(w, r)
```

Handler implementations:
- `serveDB` — GET: render database overview page
- `serveDBDownload` — GET: stream CSV file
- `serveDBUpload` — POST: multipart form with CSV file
- `serveDBCreate` — POST: form with table name

Tests:
- Test /__/db returns HTML with table list
- Test download returns CSV with correct Content-Disposition
- Test upload replaces table
- Test create makes new table
- Test all routes 404 when not in dev mode

---

### Task 6: HTML Templates
**Files**: `server/devtools.go`
**Estimated effort**: Medium

Create HTML templates matching existing dev pages style:

```go
const devToolsDBHTML = `...` // Main database view

const devToolsDBTableRowHTML = `...` // Template for each table
```

Page structure:
- Header with "Basil Database" title
- Database info box (filename)
- "Create New Table" form (input + button)
- Table list, each showing:
  - Table name
  - Column structure (name, type)
  - Row count
  - Download button
  - Upload form (file input + button)

Style: Match existing dev pages (same CSS variables, fonts, colors)

---

### Task 7: Get Database Connection
**Files**: `server/devtools.go`, `server/server.go`
**Estimated effort**: Small

Need access to the app's SQLite database connection. Options:
1. Open new connection to database.path
2. Share existing connection from server

Recommendation: Open new connection in handler (simpler, isolated).

```go
func (h *devToolsHandler) openAppDB() (*sql.DB, error) {
    dbPath := h.server.config.Database.Path
    if dbPath == "" {
        return nil, fmt.Errorf("no database configured")
    }
    return sql.Open("sqlite3", dbPath)
}
```

---

## Validation Checklist
- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `go build -o basil .`
- [ ] Manual test: view /__/db, see tables
- [ ] Manual test: download CSV, open in spreadsheet
- [ ] Manual test: upload CSV, verify table replaced
- [ ] Manual test: create new table
- [ ] Documentation updated (if needed)
- [ ] BACKLOG.md updated with deferrals (if any)

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-12-03 | Task 1: Database Info | ✅ Complete | getTableList, getTableInfo, getTableColumns |
| 2025-12-03 | Task 2: CSV Export | ✅ Complete | exportTableCSV with BLOB exclusion |
| 2025-12-03 | Task 3: CSV Import | ✅ Complete | inferColumnTypes, replaceTableFromCSV |
| 2025-12-03 | Task 4: Create Table | ✅ Complete | createEmptyTable |
| 2025-12-03 | Task 5: HTTP Handlers | ✅ Complete | /__/db routes added |
| 2025-12-03 | Task 6: HTML Templates | ✅ Complete | Styled pages matching dev tools |
| 2025-12-03 | Task 7: DB Connection | ✅ Complete | openAppDB helper |

## Deferred Items
Items to add to BACKLOG.md after implementation:
- Data view for each table (browse rows)
- SQL console with query input
- Edit individual cells in browser
- Export/import entire database

## Estimated Total Effort
~4-5 hours
