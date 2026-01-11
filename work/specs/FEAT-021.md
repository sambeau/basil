---
id: FEAT-021
title: "SQLite Dev Tools"
status: complete
priority: high
created: 2025-12-03
author: "@human"
---

# FEAT-021: SQLite Dev Tools

## Summary
Add a developer web interface at `/__/db` for viewing and managing SQLite databases in dev mode. Developers can view table structures, download table data as CSV, upload CSV to replace tables, and create new tables. This provides essential database management without building a full GUI.

## User Story
As a **developer**, I want **to view my database structure and import/export table data via CSV** so that **I can manage my development database without needing external tools**.

## Acceptance Criteria

### Database View (`/__/db`)
- [x] Shows database filename and basic info
- [x] Lists all tables with structure (columns, types) and row counts
- [x] Matches dev pages styling (consistent with `/__/logs`)
- [x] Only accessible in dev mode (`--dev` flag)

### Download Data
- [x] Each table has a "Download Data" button
- [x] Downloads CSV file with column headers and all rows
- [x] Standard CSV format (opens in Excel, Numbers, etc.)

### Replace Table
- [x] Each table has a "Replace Table" button
- [x] Upload CSV replaces both structure and data
- [x] Column types inferred from data:
  - All INTEGER → INTEGER
  - All REAL (or mixed INT/REAL) → REAL
  - Otherwise → TEXT
  - Empty strings → NULL (compatible with any type)
- [x] Validation: reject if any value doesn't match inferred column type
- [x] Clear error message on rejection

### New Table
- [x] Form on page to enter table name
- [x] Creates table with single column: `id INTEGER` with one row (value 0)
- [x] Table name validation (alphanumeric + underscore, no spaces)

## Design Decisions

- **Inferred types over explicit**: CSV headers stay clean, spreadsheet-compatible. Types determined by scanning all values in each column.
- **Replace table (not just data)**: Simpler mental model—upload always creates fresh table from CSV. More destructive but appropriate for dev environment.
- **Minimal new table**: Just `id INTEGER` with value 0. Enough to open in spreadsheet and reshape. Developer controls final structure via CSV upload.
- **Single database per port**: Each dev instance shows its own `database.path`. No multi-database UI needed.
- **Dev-only**: Same security model as `/__/logs`—requires `--dev` flag.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `server/devtools.go` — Add `/__/db` handler and supporting functions
- `server/server.go` — Register new dev route
- `server/csv.go` (new) — CSV parsing/generation with type inference

### Dependencies
- Depends on: FEAT-020 (per-developer config) for isolated dev databases
- Blocks: Future data view, SQL console features

### Edge Cases & Constraints
1. **Empty table** — Show structure, row count 0, download returns headers only
2. **Table with no rows** — Same as empty, CSV has headers but no data rows
3. **Large tables** — No pagination for now; CSV download streams all rows
4. **Reserved table names** — SQLite system tables (`sqlite_*`) should be hidden
5. **Binary/BLOB data** — Excluded from CSV export; column skipped entirely
6. **CSV parsing edge cases** — Quoted fields, embedded commas, newlines in values
7. **Transaction safety** — Replace table should be atomic (drop + create + insert in transaction)

### API Endpoints
| Method | Path | Description |
|--------|------|-------------|
| GET | `/__/db` | Database overview page |
| GET | `/__/db/download/{table}` | Download table as CSV |
| POST | `/__/db/upload/{table}` | Replace table from CSV |
| POST | `/__/db/create` | Create new table (form: `name`) |

### Type Inference Rules
```
For each column:
1. Collect all non-empty values
2. If all parse as int64 → INTEGER
3. Else if all parse as float64 → REAL
4. Else → TEXT
5. Empty string in any type column → NULL
```

### CSV Format
- Standard RFC 4180
- First row is headers (column names)
- Use Go's `encoding/csv` package
- Handle: quoted fields, escaped quotes, CRLF/LF line endings

## Implementation Notes
*Added during/after implementation*

## Related
- Plan: `work/plans/FEAT-021-plan.md`
- Similar: FEAT-019 (Dev Tools - Logs)

## Future Enhancements (not in scope)
- Data view for each table (browse rows)
- Edit individual cells
- SQL console with query input
- Export/import entire database
