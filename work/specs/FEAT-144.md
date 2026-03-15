---
id: FEAT-144
title: "DataTable Redesign"
status: approved
priority: high
created: 2026-06-15
updated: 2026-06-15
author: "@human"
depends-on: FEAT-145 (Part C only; Part A partial)
plan: PLAN-125
---

# FEAT-144: DataTable Redesign

## Summary

Redesign the `DataTable` component to accept Parsley `Table` objects directly, eliminating the need to manually decompose tables into parallel arrays. Leverage the new `Table.columnProps()` method (from DESIGN-typed-value-formatting Part C) to derive column metadata from schemas. Add commonly-needed features: empty states, type-aware cell formatting, custom cell rendering, column hiding, header overrides, alignment control, and footer/summary rows. Remove the non-functional `sortable` prop.

## User Story

As a developer building data-driven UIs, I want to pass a `Table` directly to `DataTable` and have it render with sensible defaults, so that I don't have to manually specify columns, rows, and keys that the Table already carries.

## Problem Statement

The current `DataTable` component predates the Parsley `Table` type and ignores it:

1. Users must decompose a `Table` into parallel arrays (`columns`, `rows`, `keys`) that the `Table` already carries
2. No empty state — renders empty `<tbody>` when there's no data
3. No type-aware formatting — typed values like `money` show as raw `{amount: 49.99, currency: "GBP"}`
4. No custom cell rendering — can't add links, badges, or action buttons
5. No footer/summary rows — can't show totals
6. The `sortable` prop exists but does nothing, creating a confusing API

## Acceptance Criteria

### Core: Accept Table Directly

- [ ] `DataTable` accepts `data` prop containing a `Table` object
- [ ] When `data` is provided, component derives columns from `table.columns`
- [ ] When `data` is provided, component derives rows from `table.rows`
- [ ] When `data` is provided, component derives keys from `table.columns`
- [ ] Uses `table.columnProps(col)` to derive per-column metadata (label, type, align, format)
- [ ] Backward compatibility: existing `columns`/`rows`/`keys` API continues to work
- [ ] When both `data` and `rows` are provided, `data` takes precedence

### Empty State

- [ ] When rows are empty, display empty state message in a single `<td>` with `colspan`
- [ ] Default empty message: `"No data"`
- [ ] `empty` prop overrides the default message
- [ ] `empty={false}` suppresses empty state entirely (shows empty `<tbody>`)
- [ ] Empty row has CSS class `data-table-empty`

### Column Headers

- [ ] Headers auto-derived via `columnProps().label`
- [ ] Fallback: underscores to spaces + title case
- [ ] `headers` prop (dict) overrides `columnProps()` labels for specific columns
- [ ] Schema titles flow through `columnProps()` automatically

### Column Hiding

- [ ] `hide` prop (array) excludes specified columns from rendering
- [ ] Hidden columns don't appear in `<thead>` or `<tbody>`
- [ ] Hidden columns don't appear in `<tfoot>`

### Cell Alignment

- [ ] `align` prop (dict) sets alignment per column: `"left"`, `"right"`, `"center"`
- [ ] Auto-derived alignment via `columnProps().align`:
  - Right: `money`, `integer`, `float`, `duration`, `unit`
  - Center: `boolean`
  - Left: everything else
- [ ] `align` prop overrides `columnProps()` alignment for specific columns
- [ ] Alignment classes: `align-left`, `align-right`, `align-center`

### Type-Aware Cell Formatting

- [ ] `money` values display automatically via string coercion → "£ 4,999.00" (FEAT-145 Part A)
- [ ] `datetime` values formatted explicitly via `.medium()` → "Mar 15, 2025"
- [ ] `duration` values formatted explicitly via `.medium()` → "2h 30m"
- [ ] `unit` values formatted explicitly via `.medium()` → "5.00 kg"
- [ ] `boolean` values displayed as "Yes" / "No"
- [ ] `null` values displayed as em dash "—"
- [ ] `integer`, `float`, `string` displayed as-is (string coercion)
- [ ] Formatting uses `columnProps().format` hint to determine which `.medium()` to call
- [ ] `format` prop (dict) overrides auto-detected format per column

**Note:** FEAT-145 Part A only implemented `.medium()` coercion for money. Datetime, duration, and unit require explicit `.medium()` calls in DataTable because their automatic coercion was deferred (datetime.medium() doesn't handle date-only kinds; duration.medium() returns relative time; unit.medium() forces decimal places).

### Custom Cell Rendering

- [ ] `render` prop (dict) provides per-column render functions
- [ ] Render function signature: `fn(value, row) → content`
- [ ] `value` is the cell value for that column
- [ ] `row` is the entire row dictionary (for accessing other columns)
- [ ] Render functions can return any valid Parsley content (strings, HTML, components)

### Footer Rows

- [ ] `footer` prop (array of dicts) adds `<tfoot>` with footer rows
- [ ] Footer cells use same formatting rules as body cells
- [ ] Footer cells use same render functions as body cells
- [ ] Footer cells use same alignment as body cells
- [ ] `<tfoot>` only rendered when `footer` is provided and non-empty

### Row Headers (Accessibility)

- [ ] `rowHeader` prop controls which column index is a `<th scope="row">`
- [ ] Default: `rowHeader={0}` (first column is row header)
- [ ] `rowHeader={false}` disables row headers (all cells are `<td>`)
- [ ] `rowHeader={n}` makes the nth column the row header

### Prop Spreading and Styling

- [ ] `id` prop sets table's HTML id
- [ ] `class` prop adds CSS classes (merged with `data-table`)
- [ ] `caption` prop adds accessible `<caption>` element
- [ ] Additional props spread to `<table>` element via `...attrs`

### Remove sortable Prop

- [ ] The non-functional `sortable` prop is removed from the component
- [ ] No breaking change — passing `sortable` is ignored (spread to table)

### Parsley Correctness

- [ ] All code uses `+` for string concatenation (not `++`)
- [ ] All single-expression conditionals use concise form
- [ ] Spread syntax uses `...attrs` (not `{...attrs}`)
- [ ] All files pass `pars --check`

### CSS

- [ ] Component outputs `class="data-table"` on root `<table>`
- [ ] CSS provided for alignment classes
- [ ] CSS provided for empty state styling
- [ ] CSS provided for footer styling
- [ ] CSS uses CSS custom properties for theming

## Design Decisions

1. **Table takes precedence**: When both `data` and `rows` are provided, `data` wins. This prevents confusion about which data source is used.

2. **Simple title case conversion**: Column names use simple `replace("_", " ").toTitle()` conversion. Smart camelCase handling adds complexity for minimal benefit — users who need precise headers use the `headers` prop.

3. **"Yes"/"No" for booleans**: Human-readable and accessible. Users can use `render` functions for checkmarks or other representations.

4. **Em dash for null**: Typographically correct for "no value" in tables. Standard convention in data tables.

5. **Row header is configurable**: Some tables have no logical row header; others have the identifier in a non-first column. Default to first column but allow override.

6. **Remove sortable, don't fix it**: Server-side sorting via `Table.orderBy()` is the Basil pattern. A proper sorting UI (clickable headers, URL params, indicators) is deferred to Tier 3 as a separate feature.

7. **Use `columnProps()` for schema integration**: Rather than ad-hoc schema lookups, DataTable uses `table.columnProps(col)` to get label, type, alignment, and format hints. This parallels how `record.fieldProps(field)` works for forms, creating a consistent pattern across form and display components.

8. **Explicit `.medium()` calls for non-money types**: FEAT-145 Part A only changed string coercion for `money` values. For `datetime`, `duration`, and `unit`, DataTable must explicitly call `.medium()` based on the `columnProps().format` hint. This is because:
   - `datetime.medium()` doesn't respect date-only/time-only kinds
   - `duration.medium()` returns relative time ("tomorrow") instead of absolute
   - `unit.medium()` forces decimal places that may be unwanted
   
   DataTable handles this internally — users get human-readable formatting without extra work.

9. **Depends on FEAT-145**: This feature requires:
   - **Part A (partial)**: Money values format automatically via string coercion
   - **Part C (complete)**: `Table.columnProps()` method for schema-aware column metadata

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Design Document

See `work/design/DESIGN-datatable-redesign.md` for full design rationale, implementation code, and CSS requirements.

### Affected Files

| File | Change |
|------|--------|
| `server/prelude/components/data_table.pars` | Rewrite — full redesign |
| `server/prelude/styles/components.css` | Update — add DataTable CSS |

### Dependencies

- **Depends on**: FEAT-145 (Typed Value Formatting and Field Abstraction)
  - Part A (partial): Money values format via `.medium()` in string coercion ✅
  - Part C (complete): `Table.columnProps()` method ✅
- Blocks: None
- Related: FEAT-051 (Standard Prelude), FEAT-058 (HTML Components in Prelude)

### Prerequisite Implementation — COMPLETE

FEAT-145 Parts A (partial) and C are implemented. DataTable can proceed:

1. **`objectToString()` for money** — Money values format via `.medium()` automatically ✅
2. **`Table.columnProps(col)` method** — Returns `{name, label, type, align, format}` from schema ✅
3. **Explicit formatting for datetime/duration/unit** — DataTable must call `.medium()` explicitly for these types (based on `format` hint from `columnProps()`)

### Props Interface

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `data` | Table | — | Table object (preferred) |
| `rows` | array | — | Array of dicts (fallback) |
| `columns` | array | — | Column headers (optional with Table) |
| `keys` | array | — | Row keys (optional with Table) |
| `caption` | string | — | Table caption (accessibility) |
| `empty` | string/false | `"No data"` | Empty state message |
| `headers` | dict | `{}` | Override auto-derived headers |
| `align` | dict | `{}` | Column alignment overrides |
| `hide` | array | `[]` | Columns to exclude |
| `render` | dict | `{}` | Per-column render functions |
| `format` | dict | `{}` | Per-column format overrides (e.g., `{shipped_at: "relative"}`) |
| `footer` | array | — | Footer row(s) |
| `rowHeader` | int/false | `0` | Row header column index |
| `id` | string | — | HTML id |
| `class` | string | — | Additional CSS class |
| `...attrs` | — | — | Spread to `<table>` |

### Implementation

```parsley
export DataTable = fn({
    data,
    rows,
    columns,
    keys,
    caption,
    empty,
    headers,
    align,
    hide,
    render,
    format,
    footer,
    rowHeader,
    id,
    class,
    ...attrs
}) {
    // Defaults
    let emptyMsg = empty ?? "No data"
    let headersMap = headers ?? {}
    let alignMap = align ?? {}
    let hideList = hide ?? []
    let renderMap = render ?? {}
    let formatMap = format ?? {}
    let rowHeaderIdx = rowHeader ?? 0
    
    // Derive from Table if provided
    let tableColumns = if (data) data.columns else columns ?? []
    let tableRows = if (data) data.rows else rows ?? []
    let tableKeys = if (data) data.columns else keys ?? tableColumns
    
    // Filter out hidden columns
    let visibleColumns = tableColumns.filter(fn(c) { c not in hideList })
    let visibleKeys = tableKeys.filter(fn(k) { k not in hideList })
    
    // Helper: get column props (from schema via columnProps, or defaults)
    let getColProps = fn(col) {
        if (data) {
            data.columnProps(col)
        } else {
            {name: col, label: col.replace("_", " ").toTitle(), align: "left"}
        }
    }
    
    // Helper: get header text for a column (headers prop overrides columnProps)
    let getHeader = fn(col) {
        if (headersMap[col]) {
            headersMap[col]
        } else {
            getColProps(col).label
        }
    }
    
    // Helper: get alignment for a column (align prop overrides columnProps)
    let getAlign = fn(col) {
        if (alignMap[col]) {
            alignMap[col]
        } else {
            getColProps(col).align ?? "left"
        }
    }
    
    // Helper: get format hint for a column (format prop overrides columnProps)
    let getFormat = fn(col) {
        if (formatMap[col]) {
            formatMap[col]
        } else {
            getColProps(col).format ?? null
        }
    }
    
    // Helper: format a cell value based on type
    // Money formats automatically via string coercion (FEAT-145 Part A)
    // Datetime, duration, unit need explicit .medium() calls
    let formatCell = fn(value, col) {
        if (renderMap[col]) {
            null  // Render function handles it
        } else if (value == null) {
            "—"
        } else {
            let fmt = getFormat(col)
            if (fmt == "date" || fmt == "datetime") {
                value.medium()
            } else if (fmt == "duration") {
                value.medium()
            } else if (fmt == "unit") {
                value.medium()
            } else if (fmt == "boolean") {
                if (value) "Yes" else "No"
            } else {
                // Money formats automatically; strings/numbers pass through
                value
            }
        }
    }
    
    let tableClass = "data-table" + if (class) " " + class else ""
    
    <table id={id} class={tableClass} ...attrs>
        if (caption) {
            <caption>caption</caption>
        }
        <thead>
            <tr>
                for (idx, col in visibleColumns) {
                    <th scope="col" class={"align-" + getAlign(visibleKeys[idx])}>
                        getHeader(col)
                    </th>
                }
            </tr>
        </thead>
        <tbody>
            if (tableRows.length() == 0 && emptyMsg != false) {
                <tr class="data-table-empty">
                    <td colspan={visibleColumns.length()}>emptyMsg</td>
                </tr>
            } else {
                for (row in tableRows) {
                    <tr>
                        for (idx, key in visibleKeys) {
                            let value = row[key]
                            let content = if (renderMap[key]) {
                                renderMap[key](value, row)
                            } else {
                                formatCell(value, key)
                            }
                            let alignClass = "align-" + getAlign(key)
                            
                            if (idx == rowHeaderIdx) {
                                <th scope="row" class={alignClass}>content</th>
                            } else {
                                <td class={alignClass}>content</td>
                            }
                        }
                    </tr>
                }
            }
        </tbody>
        if (footer && footer.length() > 0) {
            <tfoot>
                for (footerRow in footer) {
                    <tr>
                        for (idx, key in visibleKeys) {
                            let value = footerRow[key]
                            let content = if (renderMap[key]) {
                                renderMap[key](value, footerRow)
                            } else if (value != null) {
                                formatCell(value, key)
                            } else {
                                ""
                            }
                            <td class={"align-" + getAlign(key)}>content</td>
                        }
                    </tr>
                }
            </tfoot>
        }
    </table>
}
```

### CSS Requirements

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
    background: var(--table-hover-bg, #f9fafb);
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

### Usage Examples

**Simple usage with Table:**

```parsley
let Users = @DB.bind(UserSchema, "users")
let users = Users.all()

<DataTable data={users} caption="Users"/>
```

**Full-featured example:**

```parsley
let Orders = @DB.bind(OrderSchema, "orders")
let orders = Orders.where({status: "pending"})
let total = orders.sum("amount")

<DataTable 
    data={orders}
    caption="Pending Orders"
    empty="No pending orders"
    hide={["id", "internal_notes"]}
    headers={{
        created_at: "Order Date",
        customer_name: "Customer"
    }}
    align={{
        amount: "right"
    }}
    render={{
        customer_name: fn(v, row) { 
            <a href={"/customers/" + row.customer_id}>{v}</a> 
        },
        status: fn(v) { 
            <span class={"badge badge-" + v}>{v}</span> 
        }
    }}
    footer={[
        {customer_name: "Total", amount: total}
    ]}
/>
```

**Backward compatible (existing API):**

```parsley
<DataTable 
    columns={["Name", "Email", "Role"]} 
    rows={users} 
    keys={["name", "email", "role"]}
/>
```

### Test Cases

```go
func TestDataTableWithTable(t *testing.T) {
    tests := []struct {
        name        string
        input       string
        contains    []string
        notContains []string
    }{
        {
            name: "accepts Table directly",
            input: `
                let t = table([{name: "Alice", age: 30}])
                <DataTable data={t}/>
            `,
            contains: []string{
                "<th scope=\"col\">",
                "Alice",
                "30",
            },
        },
        {
            name: "shows empty state",
            input: `
                let t = table([])
                <DataTable data={t} empty="No records"/>
            `,
            contains: []string{
                "No records",
                "data-table-empty",
            },
        },
        {
            name: "hides columns",
            input: `
                let t = table([{id: 1, name: "Alice", secret: "xxx"}])
                <DataTable data={t} hide={["id", "secret"]}/>
            `,
            contains:    []string{"Alice"},
            notContains: []string{"secret", "xxx"},
        },
        {
            name: "custom render function",
            input: `
                let t = table([{email: "a@b.com"}])
                <DataTable data={t} render={{email: fn(v) { <a href={"mailto:" + v}>{v}</a> }}}/>
            `,
            contains: []string{
                "mailto:a@b.com",
            },
        },
        {
            name: "null values show em dash",
            input: `
                let t = table([{name: "Alice", email: null}])
                <DataTable data={t}/>
            `,
            contains: []string{"—"},
        },
        {
            name: "footer rows",
            input: `
                let t = table([{item: "Widget", price: 10}])
                <DataTable data={t} footer={[{item: "Total", price: 10}]}/>
            `,
            contains: []string{"<tfoot>", "Total"},
        },
        {
            name: "backward compatible API",
            input: `
                <DataTable 
                    columns={["Name"]} 
                    rows={[{name: "Bob"}]} 
                    keys={["name"]}
                />
            `,
            contains: []string{"Name", "Bob"},
        },
        {
            name: "row header configurable",
            input: `
                let t = table([{id: 1, name: "Alice"}])
                <DataTable data={t} rowHeader={1}/>
            `,
            contains: []string{"<th scope=\"row\">"},
        },
        {
            name: "empty state suppressed",
            input: `
                let t = table([])
                <DataTable data={t} empty={false}/>
            `,
            notContains: []string{"data-table-empty", "No data"},
        },
        {
            name: "header override",
            input: `
                let t = table([{created_at: "2025-01-01"}])
                <DataTable data={t} headers={{created_at: "Date Created"}}/>
            `,
            contains: []string{"Date Created"},
        },
    }
    // ... test execution
}
```

### Implementation Steps

1. **Verify prerequisites** — Confirm `objectToString()` and `columnProps()` are implemented
2. **Update `data_table.pars`** — Replace implementation with new version using `columnProps()`
3. **Add CSS** — Update `components.css` with DataTable styles
4. **Remove `sortable`** — Delete from props destructuring
5. **Verify syntax** — Run `pars --check server/prelude/components/data_table.pars`
6. **Add tests** — Implement test cases from testing strategy
7. **Run test suite** — `go test ./pkg/parsley/...`
8. **Check benchmarks** — `make bench-compare`
9. **Update documentation** — Prelude guide, component reference

### Effort Estimate

| Task | Effort |
|------|--------|
| Rewrite `data_table.pars` | 45 min |
| Add/update CSS | 15 min |
| Syntax verification | 10 min |
| Add tests | 1 hour |
| Update documentation | 30 min |
| **Total** | **~2.5 hours** |

**Note:** This assumes prerequisites (typed-value-formatting Parts A and C) are already implemented. Total effort including prerequisites is ~14-19 hours.

## Future Enhancements (Tier 3)

Server-side sorting UI is deferred to post-1.0:
- `sortable={true}` or `sortable={["name", "created_at"]}` for clickable headers
- URL parameter generation: `?orderBy=col&orderDir=asc`
- Visual indicators for sort direction
- Integration with `Table.orderBy()`

See `work/design/DESIGN-datatable-redesign.md` §11 for details.

## Related

- Design doc: `work/design/DESIGN-datatable-redesign.md`
- Standard Prelude Review: `work/reports/STANDARD-PRELUDE-REVIEW.md` §3, §9, §13
- **Prerequisite**: FEAT-145 (Typed Value Formatting and Field Abstraction)
  - Part A: `objectToString()` changes for typed value formatting
  - Part C: `Table.columnProps()` method for schema-aware column metadata
- Parent feature: FEAT-051 (Standard Prelude)
- Related: FEAT-058 (HTML Components in Prelude)
- Pattern parallel: `record.fieldProps()` for forms ↔ `table.columnProps()` for display