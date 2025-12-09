---
id: PLAN-035
feature: FEAT-057
title: "Implementation Plan for DevTools in Parsley"
status: draft
created: 2025-12-09
---

# Implementation Plan: FEAT-057 DevTools in Parsley

## Overview
Convert the existing DevTools UI (`/__/` routes) from Go HTML generation to Parsley files in the prelude. This enables faster iteration on DevTools UI without Go code changes.

## Prerequisites
- [x] FEAT-056: Prelude Infrastructure (completed)
- [x] FEAT-059: Error Pages in Prelude (completed)

## Tasks

### Task 1: Create DevTools Parsley Templates
**Files**: `server/prelude/devtools/*.pars`
**Estimated effort**: Medium

Create the Parsley templates for each DevTools page:

Steps:
1. Create `prelude/devtools/index.pars` - main dashboard with links
2. Create `prelude/devtools/logs.pars` - dev logs viewer
3. Create `prelude/devtools/db.pars` - database browser (table list)
4. Create `prelude/devtools/db_table.pars` - single table view
5. Create `prelude/devtools/env.pars` - environment/config viewer

Templates should:
- Use consistent styling (dark theme matching error pages)
- Include navigation between pages
- Be self-contained (inline styles)

Tests:
- Templates parse without errors
- Templates render with mock data

---

### Task 2: Update Prelude Embed and AST Loading
**Files**: `server/prelude.go`
**Estimated effort**: Small

Steps:
1. Update embed directive to include `prelude/devtools/*`
2. Ensure devtools templates are parsed at startup
3. Add `GetPreludeAST()` calls for devtools paths

Tests:
- DevTools ASTs are available via `GetPreludeAST("devtools/index.pars")`

---

### Task 3: Create DevTools Environment Builder
**Files**: `server/devtools.go`
**Estimated effort**: Medium

Create `createDevToolsEnv()` function that provides:

Steps:
1. Add `basil.version`, `basil.commit`, `basil.dev` metadata
2. Add `devtools.path` - current page path
3. Add `devtools.tables` - table list for db pages
4. Add `devtools.logs` - log entries for logs page
5. Add `devtools.config` - sanitized config for env page
6. Add navigation links array

Tests:
- Environment contains expected variables
- Table list populated when DB configured
- Config properly sanitized (no secrets)

---

### Task 4: Update DevTools Handler to Use Prelude
**Files**: `server/devtools.go`
**Estimated effort**: Medium

Steps:
1. Create `handleDevToolsWithPrelude()` function
2. Map URL paths to prelude files:
   - `/__/` → `devtools/index.pars`
   - `/__/logs` → `devtools/logs.pars`
   - `/__/db` → `devtools/db.pars`
   - `/__/db/{table}` → `devtools/db_table.pars`
   - `/__/env` → `devtools/env.pars`
3. Evaluate AST with devtools environment
4. Handle array results (like error pages)
5. Fall back to 404 for unknown paths

Tests:
- Each route renders correct template
- Unknown paths return 404
- Table view receives table name parameter

---

### Task 5: Convert Index Page
**Files**: `server/prelude/devtools/index.pars`
**Estimated effort**: Small

Steps:
1. Port existing index HTML to Parsley
2. Add version/commit display
3. Add navigation links to all DevTools pages
4. Style consistently with error pages

Tests:
- Page renders with navigation
- Version info displayed

---

### Task 6: Convert Logs Page
**Files**: `server/prelude/devtools/logs.pars`
**Estimated effort**: Medium

Steps:
1. Port existing logs HTML to Parsley
2. Display log entries from `devtools.logs`
3. Include filtering/clear functionality
4. Format timestamps and levels

Tests:
- Logs display correctly
- Empty state handled
- Timestamps formatted

---

### Task 7: Convert Database Pages
**Files**: `server/prelude/devtools/db.pars`, `server/prelude/devtools/db_table.pars`
**Estimated effort**: Medium

Steps:
1. Port table list view to `db.pars`
2. Port single table view to `db_table.pars`
3. Display table data with pagination
4. Include row count and schema info

Tests:
- Table list renders
- Table data displays in grid
- Handles empty tables

---

### Task 8: Convert Environment Page
**Files**: `server/prelude/devtools/env.pars`
**Estimated effort**: Small

Steps:
1. Port existing env HTML to Parsley
2. Display config sections
3. Mask sensitive values (passwords, keys)
4. Format nested config nicely

Tests:
- Config displays
- Secrets masked

---

### Task 9: Remove Legacy Go HTML Generation
**Files**: `server/devtools.go`
**Estimated effort**: Small

Steps:
1. Remove old HTML generation functions
2. Remove inline style constants
3. Update handler to use only prelude templates
4. Clean up unused code

Tests:
- All existing devtools tests pass
- No Go HTML generation remains

---

## Validation Checklist
- [ ] All tests pass: `make test`
- [ ] Build succeeds: `make build`
- [ ] Manual testing of all DevTools pages
- [ ] Styling consistent with error pages
- [ ] No regression in functionality

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| — | — | — | — |
