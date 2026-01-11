---
id: FEAT-083
title: "Schema Migration: db.diff() and db.migrate()"
status: proposed
priority: medium
created: 2026-01-05
author: "@human"
---

# FEAT-083: Schema Migration: db.diff() and db.migrate()

## Summary

Add `db.diff(Schema, "table")` to compare a `@schema` definition against an existing database table, and `db.migrate(Schema, "table", options)` to safely apply additive changes. This enables iterative schema development without manual SQL migrations for common cases (adding columns).

## User Story

As a developer, I want to compare my `@schema` definitions against existing database tables and automatically add new columns so that I can evolve my schema during development without writing manual ALTER TABLE statements.

## Acceptance Criteria

### Phase 1: db.diff()
- [ ] `db.diff(Schema, "table")` returns a diff object comparing schema to table
- [ ] Diff identifies columns to add (in schema, not in table)
- [ ] Diff identifies columns to remove (in table, not in schema)
- [ ] Diff identifies columns with type changes
- [ ] Diff identifies constraint changes (UNIQUE, CHECK, NOT NULL)
- [ ] Diff includes `compatible: bool` indicating if safe migration is possible
- [ ] Works with SQLite
- [ ] Works with PostgreSQL

### Phase 2: db.migrate() - Safe Mode
- [ ] `db.migrate(Schema, "table")` applies safe (additive) changes only
- [ ] Adds new columns with `ALTER TABLE ADD COLUMN`
- [ ] New columns are nullable by default (safe for existing rows)
- [ ] Returns summary of changes applied
- [ ] Errors if breaking changes detected (doesn't apply partial changes)
- [ ] `{dryRun: true}` option shows what would change without applying

### Phase 3: db.migrate() - Full Mode (Optional)
- [ ] `{mode: "full"}` enables destructive operations
- [ ] `{allowDataLoss: true}` required for column removal
- [ ] SQLite: Uses table rebuild pattern for unsupported ALTER operations
- [ ] PostgreSQL: Uses native ALTER TABLE where possible
- [ ] Constraint changes handled appropriately per database

## Design Decisions

### Safe Mode by Default
- **Rationale**: Additive-only migrations cover 80% of development use cases (adding fields). Prevents accidental data loss. Explicit opt-in for destructive operations.

### Diff Before Migrate
- **Rationale**: `db.diff()` as separate function allows inspection before action. Useful for CI/CD checks, documentation, debugging. `db.migrate()` can use diff internally.

### Schema as Source of Truth
- **Rationale**: The `@schema` definition is the desired state. The database table is the current state. Migration brings current state to desired state.

### No Rename Detection
- **Rationale**: Detecting column renames (vs drop+add) requires heuristics or explicit hints. Out of scope for v1. Users can use raw SQL for renames.

### Nullable New Columns
- **Rationale**: Adding NOT NULL columns to tables with existing rows fails without defaults. Safe mode adds nullable columns; users can add constraints manually or use full mode.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Diff Object Structure

```parsley
{
    table: "users",
    schema: "User",
    
    add: [
        {name: "phone", type: "TEXT", nullable: true},
        {name: "role", type: "TEXT", check: "role IN ('admin', 'user')"}
    ],
    
    remove: [
        {name: "legacy_field", type: "TEXT"}
    ],
    
    change: [
        {name: "age", from: {type: "TEXT"}, to: {type: "INTEGER"}},
        {name: "email", from: {unique: false}, to: {unique: true}}
    ],
    
    compatible: true,      // true if safe migration possible
    safeChanges: ["add"],  // which change types are safe
    breakingChanges: []    // which changes require full mode
}
```

### Reading Table Schema

#### SQLite
```sql
-- Get column info
PRAGMA table_info('users');
-- Returns: cid, name, type, notnull, dflt_value, pk

-- Get indexes (for UNIQUE detection)
PRAGMA index_list('users');
PRAGMA index_info('index_name');

-- Get CHECK constraints (from table SQL)
SELECT sql FROM sqlite_master WHERE type='table' AND name='users';
-- Parse CREATE TABLE statement for CHECK constraints
```

#### PostgreSQL
```sql
-- Get column info
SELECT column_name, data_type, is_nullable, column_default
FROM information_schema.columns
WHERE table_name = 'users';

-- Get constraints
SELECT constraint_name, constraint_type
FROM information_schema.table_constraints
WHERE table_name = 'users';

-- Get CHECK constraint definitions
SELECT conname, pg_get_constraintdef(oid)
FROM pg_constraint
WHERE conrelid = 'users'::regclass;
```

### Affected Components

- `pkg/parsley/evaluator/stdlib_dsl_schema.go` — Add `diff()` and `migrate()` methods to DatabaseConnection
- `pkg/parsley/evaluator/stdlib_dsl_migrate.go` — New file for migration logic
- `pkg/parsley/tests/dsl_migrate_test.go` — Tests for migration functionality

### Migration SQL Generation

#### Adding Columns (SQLite & PostgreSQL)
```sql
ALTER TABLE users ADD COLUMN phone TEXT;
ALTER TABLE users ADD COLUMN role TEXT CHECK(role IN ('admin', 'user'));
```

#### SQLite Table Rebuild (for unsupported changes)
```sql
-- 1. Rename old table
ALTER TABLE users RENAME TO users_old;

-- 2. Create new table with desired schema
CREATE TABLE users (...);

-- 3. Copy data (mapping columns)
INSERT INTO users (id, name, email) 
SELECT id, name, email FROM users_old;

-- 4. Drop old table
DROP TABLE users_old;

-- 5. Recreate indexes
CREATE INDEX ...;
```

#### PostgreSQL ALTER Operations
```sql
-- Change type
ALTER TABLE users ALTER COLUMN age TYPE INTEGER USING age::INTEGER;

-- Add UNIQUE
ALTER TABLE users ADD CONSTRAINT users_email_unique UNIQUE (email);

-- Drop column
ALTER TABLE users DROP COLUMN legacy_field;

-- Add NOT NULL (with default)
ALTER TABLE users ALTER COLUMN status SET NOT NULL;
```

### Type Mapping for Comparison

| Schema Type | SQLite Type | PostgreSQL Type |
|-------------|-------------|-----------------|
| `int` | `INTEGER` | `integer` |
| `bigint` | `INTEGER` | `bigint` |
| `string`, `text` | `TEXT` | `text`, `character varying` |
| `bool` | `INTEGER` | `boolean` |
| `float` | `REAL` | `real`, `double precision` |
| `datetime` | `TEXT` | `timestamp` |
| `email`, `url`, `phone`, `slug` | `TEXT` | `text` |
| `enum(...)` | `TEXT` | `text` |
| `json` | `TEXT` | `jsonb` |

### Edge Cases & Constraints

1. **Table doesn't exist** — `db.diff()` returns diff with all columns as "add", `compatible: true`. `db.migrate()` creates table.

2. **Schema field is relation** — Skip relation fields (they don't map to columns). Only compare primitive fields.

3. **Primary key changes** — Never safe. Always requires `{mode: "full", allowDataLoss: true}`.

4. **Type widening vs narrowing** — `INTEGER` → `BIGINT` might be safe. `TEXT` → `INTEGER` is breaking.

5. **CHECK constraint changes** — Adding is safe (new data validated). Removing is safe. Changing requires validation of existing data.

6. **UNIQUE constraint on existing data** — Adding UNIQUE may fail if duplicates exist. Detect and warn.

7. **Foreign key references** — Out of scope for v1. Relations don't create FK constraints currently.

8. **Column order** — Ignore column order differences. SQL doesn't guarantee order.

9. **Case sensitivity** — Normalize column names for comparison (SQLite is case-insensitive, PostgreSQL preserves case).

10. **Reserved columns** — Skip comparing `id` column if it's auto-generated primary key.

### Dependencies

- Depends on: FEAT-081 (Rich Schema Types) — for type constraint metadata
- Depends on: FEAT-079 (Query DSL) — for schema and binding infrastructure

### API Examples

```parsley
@schema User {
    id: int
    name: string(min: 1, max: 100)
    email: email(unique: true)
    phone: phone                    // NEW FIELD
    role: enum("admin", "user")     // NEW FIELD
}

let db = @sqlite("app.db")

// Check what would change
let diff = db.diff(User, "users")
// {
//     add: [{name: "phone", type: "TEXT"}, {name: "role", type: "TEXT", check: "..."}],
//     remove: [],
//     change: [],
//     compatible: true
// }

// Apply safe changes
let result = db.migrate(User, "users")
// {applied: ["ADD COLUMN phone", "ADD COLUMN role"], skipped: []}

// Or dry run first
let preview = db.migrate(User, "users", {dryRun: true})
// {wouldApply: ["ADD COLUMN phone", "ADD COLUMN role"], wouldSkip: []}
```

### Error Handling

```parsley
// Breaking changes detected
let diff = db.diff(User, "users")
// {
//     change: [{name: "age", from: {type: "TEXT"}, to: {type: "INTEGER"}}],
//     compatible: false,
//     breakingChanges: ["age: type change requires full mode"]
// }

let result = db.migrate(User, "users")
// Error: Migration blocked - breaking changes detected:
//   - age: type change from TEXT to INTEGER
// Use db.migrate(User, "users", {mode: "full"}) to proceed
```

## Test Cases

### db.diff() Tests
- Diff identical schema and table returns empty diff
- Diff detects new columns in schema
- Diff detects removed columns (in table, not schema)
- Diff detects type changes
- Diff detects UNIQUE constraint added
- Diff detects UNIQUE constraint removed
- Diff detects CHECK constraint changes
- Diff handles enum types correctly
- Diff handles validated types (email, url, phone, slug)
- Diff skips relation fields
- Diff on non-existent table returns all-add diff
- Diff normalizes type names across databases

### db.migrate() Safe Mode Tests
- Migrate adds new columns
- Migrate with dryRun shows changes without applying
- Migrate refuses breaking changes in safe mode
- Migrate on non-existent table creates table
- Migrate is idempotent (running twice is safe)
- Migrate returns summary of applied changes

### db.migrate() Full Mode Tests
- Full mode allows type changes (PostgreSQL)
- Full mode with table rebuild (SQLite)
- Full mode requires allowDataLoss for column removal
- Full mode handles UNIQUE constraint addition with existing duplicates

### Integration Tests
- Diff + migrate workflow end-to-end
- Migrate preserves existing data
- Migrate handles tables with existing rows

## Implementation Notes

*To be added during implementation*

## Related

- FEAT-079: Query DSL (schema infrastructure)
- FEAT-081: Rich Schema Types (type constraints)
- `db.createTable()`: Initial table creation
