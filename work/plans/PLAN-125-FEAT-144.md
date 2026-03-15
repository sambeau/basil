---
id: PLAN-125
feature: FEAT-144
title: "Implementation Plan for DataTable Redesign"
status: active
created: 2026-06-15
---

# Implementation Plan: FEAT-144

## Overview

Redesign the `DataTable` component to accept Parsley `Table` objects directly, leveraging `table.columnProps()` for schema-aware column metadata. This plan covers five phases: core rewrite, formatting logic, styling, testing, and documentation.

## Prerequisites

- [x] FEAT-145 Part A (partial): Money values format via `.medium()` in string coercion
- [x] FEAT-145 Part C: `table.columnProps(col)` returns `{name, label, type, align, format}`
- [x] Design doc reviewed: `work/design/DESIGN-datatable-redesign.md`
- [x] Feature spec approved: `work/specs/FEAT-144.md`

## Phase 1: Core Component Rewrite (1-1.5 hours)

### Task 1.1: Scaffold new DataTable structure
**Files**: `server/prelude/components/data_table.pars`
**Estimated effort**: 20 min
**Status**: ✅ Complete

Steps:
1. Back up existing implementation (or rely on git)
2. Create new component with full props destructuring:
   - `data`, `rows`, `columns`, `keys` (data sources)
   - `caption`, `empty`, `headers`, `align`, `hide`, `render`, `format`, `footer`
   - `rowHeader`, `id`, `class`, `...attrs`
3. Add prop defaults:
   - `empty` → `"No data"`
   - `headers` → `{}`
   - `align` → `{}`
   - `hide` → `[]`
   - `render` → `{}`
   - `format` → `{}`
   - `rowHeader` → `0`
4. Remove `sortable` from destructuring (spread to `...attrs`)

Tests:
- Component parses without syntax errors: `pars --check server/prelude/components/data_table.pars`

---

### Task 1.2: Implement data derivation logic
**Files**: `server/prelude/components/data_table.pars`
**Estimated effort**: 15 min
**Status**: ✅ Complete

Steps:
1. Derive `tableColumns` from `data.columns` or `columns` prop
2. Derive `tableRows` from `data.rows` or `rows` prop
3. Derive `tableKeys` from `data.columns` or `keys` prop or `tableColumns`
4. Filter visible columns/keys using `hide` array
5. Implement `getColProps(col)` helper using `data.columnProps(col)` when available

Tests:
- Table-based: `<DataTable data={table([{a:1}])}/>` renders correctly
- Array-based: `<DataTable rows={[{a:1}]} columns={["A"]} keys={["a"]}/>` still works

---

### Task 1.3: Implement header derivation
**Files**: `server/prelude/components/data_table.pars`
**Estimated effort**: 10 min
**Status**: ✅ Complete

Steps:
1. Implement `getHeader(col)` helper:
   - Check `headers[col]` first (explicit override)
   - Fall back to `getColProps(col).label`
2. Render `<thead>` with derived headers

Tests:
- Auto-derived: `table([{created_at: "..."}])` → header "Created At"
- Schema title: schema with `{title: "Date Created"}` → uses title
- Override: `headers={{created_at: "When"}}` → uses override

---

### Task 1.4: Implement alignment logic
**Files**: `server/prelude/components/data_table.pars`
**Estimated effort**: 10 min
**Status**: ✅ Complete

Steps:
1. Implement `getAlign(col)` helper:
   - Check `align[col]` first (explicit override)
   - Fall back to `getColProps(col).align ?? "left"`
2. Apply alignment class to `<th>` and `<td>` elements

Tests:
- Schema-derived: money column → `align-right`
- Override: `align={{total: "center"}}` → uses override
- Default: unknown column → `align-left`

---

### Task 1.5: Implement body rows
**Files**: `server/prelude/components/data_table.pars`
**Estimated effort**: 15 min
**Status**: ✅ Complete

Steps:
1. Iterate over `tableRows`
2. For each row, iterate over `visibleKeys`
3. Apply `rowHeader` logic:
   - If `idx == rowHeaderIdx` → `<th scope="row">`
   - Otherwise → `<td>`
4. Apply alignment class to each cell

Tests:
- First column is row header by default
- `rowHeader={1}` makes second column the row header
- `rowHeader={false}` disables row headers

---

### Task 1.6: Implement empty state
**Files**: `server/prelude/components/data_table.pars`
**Estimated effort**: 10 min
**Status**: ✅ Complete

Steps:
1. Check if `tableRows.length() == 0`
2. If empty and `emptyMsg != false`:
   - Render single `<tr class="data-table-empty">`
   - Render `<td colspan={visibleColumns.length()}>emptyMsg</td>`
3. If `empty={false}`, render empty `<tbody>`

Tests:
- Default: empty table shows "No data"
- Custom: `empty="Nothing here"` shows custom message
- Suppressed: `empty={false}` shows empty tbody

---

## Phase 2: Cell Formatting (45 min - 1 hour)

### Task 2.1: Implement format hint helper
**Files**: `server/prelude/components/data_table.pars`
**Estimated effort**: 10 min
**Status**: ✅ Complete

Steps:
1. Implement `getFormat(col)` helper:
   - Check `format[col]` first (explicit override)
   - Fall back to `getColProps(col).format ?? null`

Tests:
- Schema-derived: money column → format "currency"
- Override: `format={{date: "relative"}}` → uses override

---

### Task 2.2: Implement formatCell helper
**Files**: `server/prelude/components/data_table.pars`
**Estimated effort**: 25 min
**Status**: ✅ Complete

Steps:
1. Create `formatCell(value, col)` function
2. Handle null: return "—" (em dash)
3. Handle render override: return null (let render function handle it)
4. Handle format hints:
   - `"date"` or `"datetime"` → `value.medium()`
   - `"duration"` → `value.medium()`
   - `"unit"` → `value.medium()`
   - `"boolean"` → `if (value) "Yes" else "No"`
5. Default: return `value` (string coercion handles money automatically)

Tests:
- Money: `£4999.00` → "£ 4,999.00" (automatic via string coercion)
- Datetime: with schema → "Mar 15, 2025"
- Boolean: `true` → "Yes", `false` → "No"
- Null: `null` → "—"
- String: "hello" → "hello"

---

### Task 2.3: Implement custom render functions
**Files**: `server/prelude/components/data_table.pars`
**Estimated effort**: 15 min
**Status**: ✅ Complete

Steps:
1. In row iteration, check if `render[key]` exists
2. If yes, call `render[key](value, row)`
3. If no, call `formatCell(value, key)`

Tests:
- Render function receives value and row
- Render function can return HTML: `fn(v, r) { <a href={"/user/" + r.id}>v</a> }`
- Render function overrides formatCell

---

### Task 2.4: Implement footer rows
**Files**: `server/prelude/components/data_table.pars`
**Estimated effort**: 15 min
**Status**: ✅ Complete

**Note**: Discovered that Parsley's `&&` operator does not short-circuit. Used nested `if` statements instead of `footer != null && footer.length() > 0`.

Steps:
1. Check if `footer && footer.length() > 0`
2. If yes, render `<tfoot>`
3. For each footer row, iterate over visible keys
4. Apply same formatting and alignment as body cells
5. Footer cells are always `<td>` (no row headers)

Tests:
- Footer renders when provided
- Footer uses same alignment as body
- Footer uses same render functions as body
- Footer not rendered when empty/null

---

## Phase 3: Styling and Props (30 min)

### Task 3.1: Implement prop spreading
**Files**: `server/prelude/components/data_table.pars`
**Estimated effort**: 10 min
**Status**: ✅ Complete

Steps:
1. Build `tableClass`: `"data-table" + if (class) " " + class else ""`
2. Apply `id={id}` to table
3. Apply `class={tableClass}` to table
4. Spread `...attrs` to table element

Tests:
- `class="striped"` → `class="data-table striped"`
- `id="users"` → `id="users"`
- `data-testid="table"` → spread to table element

---

### Task 3.2: Implement caption
**Files**: `server/prelude/components/data_table.pars`
**Estimated effort**: 5 min
**Status**: ✅ Complete

Steps:
1. If `caption` is provided, render `<caption>caption</caption>` as first child of table

Tests:
- Caption renders when provided
- Caption not rendered when null/missing

---

### Task 3.3: Update CSS
**Files**: `server/prelude/css/basil.css`
**Estimated effort**: 15 min
**Status**: ✅ Complete

Steps:
1. Add/update `.data-table` base styles
2. Add `.data-table th, .data-table td` cell styles
3. Add `.data-table thead th` header styles
4. Add `.data-table tbody tr:hover` hover styles
5. Add alignment classes: `.align-left`, `.align-right`, `.align-center`
6. Add `.data-table-empty td` empty state styles
7. Add `.data-table tfoot td` footer styles
8. Use CSS custom properties for theming

CSS to add:
```css
.data-table {
    width: 100%;
    border-collapse: collapse;
}

.data-table th,
.data-table td {
    padding: 0.5rem 0.75rem;
    text-align: left;
    border-bottom: 1px solid var(--border-color, #e5e7eb);
}

.data-table thead th {
    font-weight: 600;
    background: var(--table-header-bg, #f9fafb);
}

.data-table tbody tr:hover {
    background: var(--table-hover-bg, #f3f4f6);
}

.data-table .align-left { text-align: left; }
.data-table .align-right { text-align: right; }
.data-table .align-center { text-align: center; }

.data-table-empty td {
    text-align: center;
    padding: 2rem;
    color: var(--text-muted, #6b7280);
    font-style: italic;
}

.data-table tfoot td {
    font-weight: 600;
    border-top: 2px solid var(--border-color, #e5e7eb);
}
```

Tests:
- Visual inspection of rendered tables
- Alignment classes apply correct text-align

---

## Phase 4: Testing (1 hour)

### Task 4.1: Create DataTable test file
**Files**: `pkg/parsley/tests/datatable_test.go`
**Estimated effort**: 15 min
**Status**: ✅ Complete

Steps:
1. Create new test file with standard imports
2. Set up test helper for evaluating DataTable with prelude
3. Create table-driven test structure

---

### Task 4.2: Implement core functionality tests
**Files**: `pkg/parsley/tests/datatable_test.go`
**Estimated effort**: 25 min
**Status**: ✅ Complete

Test cases:
1. **Table input**: `<DataTable data={table([{name: "Alice"}])}/>` renders correctly
2. **Array input (backward compat)**: `columns`/`rows`/`keys` still work
3. **Auto-derived headers**: column names title-cased
4. **Schema headers**: schema titles used
5. **Header override**: `headers` prop overrides
6. **Column hiding**: `hide` excludes columns
7. **Row header default**: first column is `<th scope="row">`
8. **Row header configurable**: `rowHeader={1}` works
9. **Row header disabled**: `rowHeader={false}` works

---

### Task 4.3: Implement formatting tests
**Files**: `pkg/parsley/tests/datatable_test.go`
**Estimated effort**: 15 min
**Status**: ✅ Complete

Test cases:
1. **Money formatting**: renders with thousands separator
2. **Datetime formatting**: renders human-readable with schema
3. **Boolean formatting**: "Yes"/"No"
4. **Null formatting**: em dash "—"
5. **Custom render**: render function called with value and row

---

### Task 4.4: Implement edge case tests
**Files**: `pkg/parsley/tests/datatable_test.go`
**Estimated effort**: 15 min
**Status**: ✅ Complete

Test cases:
1. **Empty state default**: shows "No data"
2. **Empty state custom**: custom message
3. **Empty state suppressed**: `empty={false}`
4. **Footer rendering**: footer rows render in tfoot
5. **Alignment auto**: money → right, boolean → center
6. **Alignment override**: `align` prop works
7. **Caption**: renders when provided
8. **Class merging**: custom class added to data-table

---

## Phase 5: Documentation (30 min)

### Task 5.1: Update prelude component documentation
**Files**: `docs/basil/prelude.md` or equivalent
**Estimated effort**: 15 min
**Status**: ✅ Complete

Steps:
1. Update DataTable section with new API
2. Document all props with examples
3. Show Table-first usage pattern
4. Show backward-compatible array usage

---

### Task 5.2: Add migration notes
**Files**: `docs/parsley/CHANGES.md`
**Estimated effort**: 10 min
**Status**: ✅ Complete

Steps:
1. Document DataTable redesign
2. Note that `sortable` prop is removed (no-op)
3. Show migration from array-based to Table-based usage

---

### Task 5.3: Update CHANGELOG
**Files**: `CHANGELOG.md`
**Estimated effort**: 5 min
**Status**: ✅ Complete (in CHANGES.md)

Steps:
1. Add DataTable redesign to unreleased section
2. Note new features: empty state, formatting, render functions, footer

---

## Validation Checklist

- [x] `pars --check server/prelude/components/data_table.pars` passes
- [x] All tests pass: `go test ./pkg/parsley/...`
- [x] Build succeeds: `make build`
- [x] Benchmarks checked: `make bench-compare`
- [ ] Visual inspection of rendered tables
- [x] Backward compatibility verified
- [x] Documentation updated

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-06-15 | Task 1.1: Scaffold component | ✅ Complete | Full props destructuring |
| 2026-06-15 | Task 1.2: Data derivation | ✅ Complete | Table and array support |
| 2026-06-15 | Task 1.3: Header derivation | ✅ Complete | columnProps integration |
| 2026-06-15 | Task 1.4: Alignment logic | ✅ Complete | Schema-aware alignment |
| 2026-06-15 | Task 1.5: Body rows | ✅ Complete | rowHeader support |
| 2026-06-15 | Task 1.6: Empty state | ✅ Complete | Configurable message |
| 2026-06-15 | Task 2.1: Format helper | ✅ Complete | getFormat() |
| 2026-06-15 | Task 2.2: formatCell | ✅ Complete | Type-aware formatting |
| 2026-06-15 | Task 2.3: Render functions | ✅ Complete | Custom cell rendering |
| 2026-06-15 | Task 2.4: Footer rows | ✅ Complete | Nested if for null check |
| 2026-06-15 | Task 3.1: Prop spreading | ✅ Complete | ...attrs support |
| 2026-06-15 | Task 3.2: Caption | ✅ Complete | Accessibility |
| 2026-06-15 | Task 3.3: CSS | ✅ Complete | basil.css updated |
| 2026-06-15 | Task 4.1: Test file | ✅ Complete | datatable_test.go |
| 2026-06-15 | Task 4.2: Core tests | ✅ Complete | 22 test cases |
| 2026-06-15 | Task 4.3: Formatting tests | ✅ Complete | Schema integration |
| 2026-06-15 | Task 4.4: Edge case tests | ✅ Complete | Edge cases covered |
| 2026-06-15 | Task 5.1: Component docs | ✅ Complete | In CHANGES.md |
| 2026-06-15 | Task 5.2: Migration notes | ✅ Complete | DataTable section added |
| 2026-06-15 | Task 5.3: CHANGELOG | ✅ Complete | In CHANGES.md |

## Implementation Order

Recommended order to minimize conflicts and enable incremental testing:

1. **Phase 1** (Tasks 1.1-1.6) — Core rewrite, testable after each task
2. **Phase 2** (Tasks 2.1-2.4) — Formatting logic, builds on Phase 1
3. **Phase 3** (Tasks 3.1-3.3) — Styling and polish
4. **Phase 4** (Tasks 4.1-4.4) — Full test coverage
5. **Phase 5** (Tasks 5.1-5.3) — Documentation

## Effort Summary

| Phase | Tasks | Estimated Time |
|-------|-------|----------------|
| Phase 1: Core Rewrite | 1.1-1.6 | 1-1.5 hours |
| Phase 2: Formatting | 2.1-2.4 | 45 min - 1 hour |
| Phase 3: Styling | 3.1-3.3 | 30 min |
| Phase 4: Testing | 4.1-4.4 | 1 hour |
| Phase 5: Documentation | 5.1-5.3 | 30 min |
| **Total** | | **~4-5 hours** |

## Risk Factors

1. **Parsley syntax edge cases**: The component uses advanced features (spread, conditionals in loops). Verify with `pars --check` frequently.

2. **Prelude loading**: DataTable is loaded as part of prelude. Ensure changes don't break prelude initialization.

3. **Backward compatibility**: Existing code using `columns`/`rows`/`keys` must continue to work. Test thoroughly.

4. **Type detection without schema**: Tables without schemas won't have type hints. Ensure graceful fallback.

## Dependencies

- **FEAT-145 Part C**: `table.columnProps()` — COMPLETE ✅
- **FEAT-145 Part A**: Money formatting in string coercion — COMPLETE ✅