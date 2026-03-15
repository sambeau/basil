# Design: DataTable Redesign

**Date:** 2025-01-14
**Updated:** 2026-06-15
**Status:** Approved
**Depends On:** `DESIGN-typed-value-formatting.md` (Part C complete; Part A partial — money only)
**Related:** 
- `work/reports/STANDARD-PRELUDE-REVIEW.md` §3, §9, §13
- `work/design/DESIGN-typed-value-formatting.md` — Part A (objectToString), Part C (columnProps)
- `work/specs/FEAT-144.md` — Implementation spec

---

## 1. Overview

The current `DataTable` component predates the Parsley `Table` type and ignores it. Users must decompose a `Table` into parallel arrays (`columns`, `rows`, `keys`) that the `Table` already carries. This redesign makes `DataTable` the primary accessible presentation layer for `Table` objects while adding commonly-needed features: empty states, type-aware cell formatting, custom cell rendering, and footer/summary rows.

### 1.1 Design Goals

1. **Accept `Table` directly** — `<DataTable data={users}/>` should just work
2. **Auto-format typed values** — `money`, `datetime`, `unit`, `duration` cells render human-readable by default
3. **Provide empty state** — Show a message when there are no rows
4. **Support custom cell rendering** — Links, badges, action buttons per column
5. **Maintain backward compatibility** — Existing `columns`/`rows`/`keys` API continues to work
6. **Keep concerns separated** — Sorting and pagination remain external (`Table.orderBy()`, `<Pagination/>`)

### 1.2 Non-Goals

- **Client-side sorting** — Complex JS; server-side sorting via `Table.orderBy()` is the Basil pattern
- **Built-in pagination** — Better as separate `<Pagination/>` component that composes alongside
- **Inline editing** — Out of scope for 1.0

---

## 2. Current State

### 2.1 Current API

```parsley
<DataTable 
    caption="User list" 
    columns={["Name", "Email", "Role"]} 
    rows={users} 
    keys={["name", "email", "role"]}
/>
```

### 2.2 Current Implementation

```parsley
// From server/prelude/components/data_table.pars
export DataTable = fn({caption, columns, rows, keys, id, class, sortable}) {
    <table id={id} class={"data-table" + if (class) " " + class else ""}>
        if (caption) {
            <caption>caption</caption>
        }
        <thead>
            <tr>
                for (idx, col in columns ?? []) {
                    <th scope="col">col</th>
                }
            </tr>
        </thead>
        <tbody>
            for (row in rows ?? []) {
                <tr>
                    for (idx, key in keys ?? []) {
                        // First column gets row scope for accessibility
                        if (idx == 0) {
                            <th scope="row">row[key]</th>
                        } else {
                            <td>row[key]</td>
                        }
                    }
                </tr>
            }
        </tbody>
    </table>
}
```

### 2.3 Problems

| Issue | Impact |
|-------|--------|
| Must manually specify `columns`, `rows`, `keys` even when data is a `Table` | Verbose, error-prone |
| No empty state — renders empty `<tbody>` | Poor UX |
| No type-aware formatting — outputs raw values | `money` shows as `{amount: 49.99, currency: "GBP"}` |
| No custom cell rendering | Can't add links, badges, action buttons |
| No footer/summary row | Can't show totals |
| `sortable` prop exists but does nothing | Confusing API |

---

## 3. Proposed API

### 3.1 Primary Usage (with `Table`)

```parsley
// Using TableBinding (recommended)
let Users = @DB.bind(UserSchema, "users")
let users = Users.all()

<DataTable data={users} caption="Users"/>

// Or using Query DSL
let users = @query(Users ??-> *)
<DataTable data={users} caption="Users"/>
```

The component derives columns and rows from the `Table` automatically.

### 3.2 Full Props Interface

```parsley
<DataTable
    // Data source (choose one)
    data={table}                          // Table object — preferred
    rows={[...]}                          // Array of dicts — fallback
    columns={["Name", "Email"]}           // Column headers (optional with Table)
    keys={["name", "email"]}              // Row keys (optional with Table)
    
    // Presentation
    caption="User List"                   // Table caption (accessibility)
    empty="No users found"                // Empty state message
    class="striped"                       // Additional CSS class
    id="users-table"                      // HTML id
    
    // Column configuration
    headers={{                            // Override auto-derived headers
        email: "Email Address",
        created_at: "Joined"
    }}
    align={{                              // Column alignment (auto-derived if not set)
        price: "right",
        name: "left"
    }}
    hide={["id", "password_hash"]}        // Columns to exclude
    
    // Cell rendering
    render={{                             // Per-column render functions
        email: fn(val, row) { <a href={"mailto:" + val}>{val}</a> },
        status: fn(val) { <span class={"badge badge-" + val}>{val}</span> }
    }}
    format={{                             // Override type-based formatting
        price: "currency",
        created_at: "date"
    }}
    
    // Footer
    footer={[                             // Footer row(s)
        {name: "Total", price: totalPrice}
    ]}
    
    // Accessibility
    rowHeader={0}                         // Which column index is the row header (default: 0)
    
    // Spread remaining props to <table>
    ...attrs
/>
```

### 3.3 Backward Compatibility

The existing API continues to work:

```parsley
// Still works — no changes needed for existing code
<DataTable 
    columns={["Name", "Email"]} 
    rows={users} 
    keys={["name", "email"]}
/>
```

When both `data` and `rows` are provided, `data` takes precedence.

---

## 4. Feature Details

### 4.1 Auto-Deriving from `Table`

When `data` is a `Table`:

| Derived From | Used For |
|--------------|----------|
| `table.columns` | Column headers (unless `headers` prop overrides) |
| `table.rows` | Row data |
| `table.schema` | Type information for formatting/alignment (if bound) |
| Column names | Keys for accessing row values |

```parsley
// Given:
let Products = @DB.bind(ProductSchema, "products")
let products = Products.all()

// This:
<DataTable data={products}/>

// Is equivalent to:
<DataTable 
    columns={["name", "price", "created_at"]}
    rows={products.rows}
    keys={["name", "price", "created_at"]}
/>
```

### 4.2 Empty State

When `rows` is empty or `data` has no rows:

```html
<table class="data-table">
    <caption>Products</caption>
    <thead>...</thead>
    <tbody>
        <tr class="data-table-empty">
            <td colspan="4">No products found</td>
        </tr>
    </tbody>
</table>
```

The `empty` prop controls the message:

```parsley
<DataTable data={products} empty="No products match your search"/>
<DataTable data={products} empty={false}/>  // Suppress empty state entirely
```

Default: `"No data"`

### 4.3 Type-Aware Cell Formatting

Cells are formatted based on their value type:

| Value Type | Default Formatting | Alignment | Implementation |
|------------|-------------------|-----------|----------------|
| `money` | "£ 4,999.00" | Right | Automatic via string coercion (FEAT-145 Part A) |
| `datetime` | "Mar 15, 2025" | Left | Explicit `.medium()` call required |
| `duration` | "2h 30m" | Right | Explicit `.medium()` call required |
| `unit` | "5.00 kg" | Right | Explicit `.medium()` call required |
| `integer`, `float` | As-is (string coercion) | Right | Automatic |
| `boolean` | "Yes" / "No" | Center | Explicit check in DataTable |
| `null` | "—" (em dash) | — | Explicit check in DataTable |
| `string`, other | As-is | Left | Automatic |

**FEAT-145 Implementation Status:**
- **Money**: Formats automatically via string coercion ✅
- **Datetime/Duration/Unit**: FEAT-145 Part A deferred these types because:
  - `datetime.medium()` doesn't respect date-only/time-only kinds
  - `duration.medium()` returns relative time ("tomorrow") instead of absolute
  - `unit.medium()` forces decimal places that may be unwanted
  
  DataTable must explicitly call `.medium()` for these types based on the `columnProps().format` hint.

The `format` prop overrides auto-detection:

```parsley
<DataTable 
    data={orders}
    format={{
        total: "currency",      // Force currency formatting
        shipped_at: "relative"  // Use relative time
    }}
/>
```

### 4.4 Custom Cell Rendering

The `render` prop provides per-column render functions:

```parsley
<DataTable 
    data={users}
    render={{
        email: fn(value, row) {
            <a href={"mailto:" + value}>{value}</a>
        },
        status: fn(value) {
            <span class={"badge badge-" + value}>{value}</span>
        },
        actions: fn(_, row) {
            <a href={"/users/" + row.id + "/edit"} class="btn btn-sm">"Edit"</a>
        }
    }}
/>
```

Render function signature: `fn(value, row) → content`

- `value` — The cell value for this column
- `row` — The entire row dictionary (for accessing other columns)

If a render function returns `null`, the cell is empty.

### 4.5 Column Headers

Headers are derived in this order:

1. `headers` prop (explicit override)
2. `table.schema().title(column)` (if Table has schema with titles)
3. Column name, title-cased (`created_at` → "Created At")

```parsley
<DataTable 
    data={users}
    headers={{
        email: "Email Address",
        created_at: "Member Since"
    }}
/>
```

### 4.6 Hiding Columns

The `hide` prop excludes columns from rendering:

```parsley
<DataTable 
    data={users}
    hide={["id", "password_hash", "internal_notes"]}
/>
```

### 4.7 Footer Rows

The `footer` prop adds footer rows for totals/summaries:

```parsley
<DataTable 
    data={orders}
    footer={[
        {product: "Subtotal", price: subtotal},
        {product: "Tax", price: tax},
        {product: "Total", price: total}
    ]}
/>
```

Output:

```html
<tfoot>
    <tr>
        <td>Subtotal</td>
        <td class="align-right">£100.00</td>
    </tr>
    <!-- ... -->
</tfoot>
```

Footer cells use the same formatting/render rules as body cells.

### 4.8 Alignment

Alignment is auto-derived from value types but can be overridden:

```parsley
<DataTable 
    data={products}
    align={{
        description: "left",
        price: "right",
        in_stock: "center"
    }}
/>
```

Auto-derived alignment:
- **Right:** `money`, `integer`, `float`, `duration`, `unit`
- **Center:** `boolean`
- **Left:** Everything else

---

## 5. Implementation

### 5.1 Proposed Implementation

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
    let schema = if (data) data.schema else null
    
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

### 5.2 CSS Requirements

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

---

## 6. Decisions Needed

### 6.1 DECISION: Row Header Column

**Question:** Should the first column always be a `<th scope="row">`, or should this be configurable?

**Options:**
- **A) Always first column** — Current behavior, simplest
- **B) Configurable via `rowHeader` prop** — `rowHeader={0}` (default), `rowHeader={false}` to disable, `rowHeader={2}` for third column
- **C) Auto-detect** — Use the column that looks like an identifier (name, title, id)

**Recommendation:** Option B — Configurable with sensible default. Some tables have no logical row header (e.g., log entries); others have the identifier in a non-first column.

**Status:** ✅ **DECIDED: Option B** — Configurable via `rowHeader` prop. Default is `0` (first column). Use `rowHeader={false}` to disable.

---

### 6.2 DECISION: Boolean Formatting

**Question:** How should boolean values be displayed?

**Options:**
- **A) "Yes" / "No"** — Human readable
- **B) "True" / "False"** — Programmer readable
- **C) ✓ / ✗** — Visual, compact
- **D) Checkbox icon** — `<input type="checkbox" disabled checked/>` (or unchecked)
- **E) Configurable default** — Pick one, allow `format: {active: "checkbox"}` to override

**Recommendation:** Option A with Option E — "Yes"/"No" is most accessible; allow override for specific columns.

**Status:** ✅ **DECIDED: Option A** — "Yes"/"No" for simplicity and accessibility. Users can use `render` functions for custom boolean display if needed. No override prop for MVP.

---

### 6.3 DECISION: Null Value Display

**Question:** How should `null` values be displayed?

**Options:**
- **A) Em dash "—"** — Typographically correct for missing data
- **B) Hyphen "-"** — Simple, common
- **C) Empty string** — Minimal, but may look like a bug
- **D) "N/A"** — Explicit but verbose
- **E) Configurable via `nullValue` prop**

**Recommendation:** Option A — Em dash is the typographically correct choice for tabular data indicating "no value". No need for a prop; users can use `render` functions for custom handling.

**Status:** ✅ **DECIDED: Option A** — Em dash "—" for null values. Standard typographic convention for "no data" in tables.

---

### 6.4 DECISION: Title Case Conversion

**Question:** How should column names be converted to headers?

**Examples:**
- `created_at` → "Created At" (title case, underscores to spaces)
- `firstName` → "First Name" (camelCase split)
- `userID` → "User ID" or "UserID"?

**Options:**
- **A) Simple replace + title case** — `col.replace("_", " ").toTitleCase()`
- **B) Smart conversion** — Handle camelCase, acronyms (ID, URL, API)
- **C) No conversion** — Display raw column name; rely on `headers` prop for nice names

**Recommendation:** Option A for MVP — Simple and predictable. Smart conversion is complex and can produce surprising results. Users who care about precise headers will use the `headers` prop.

**Status:** ✅ **DECIDED: Option A** — Simple conversion: underscores to spaces + title case. Handles snake_case (common for database columns). Users who want precise headers use the `headers` prop.

---

### 6.5 DECISION: Remove `sortable` Prop

**Question:** The current `sortable` prop does nothing. Should we remove it or implement it?

**Options:**
- **A) Remove it** — Sorting is a `Table.orderBy()` concern, not presentation
- **B) Implement client-side sorting** — Complex JS, scope creep
- **C) Implement server-side sorting helpers** — Add `sortHref` prop that generates header links

**Recommendation:** Option A — Remove the unused prop. Server-side sorting with `Table.orderBy()` is the Basil pattern. If we add sorting UI later (1.2+), it should be a separate component or enhancement, not baked into `DataTable`.

**Status:** ✅ **DECIDED: Option A** — Remove the unused `sortable` prop. Server-side sorting via `Table.orderBy()` is the Basil pattern. A well-designed sorting enhancement (clickable headers, URL params, direction indicators) is deferred to 1.1/1.2 as a separate feature. Users can implement custom sorting in ~45 lines (see `bofdi/components/unsafeTable.pars` for a working example pattern using session state and URL parameters).

---

## 7. Migration Guide

### 7.1 For Existing Users

No changes required. The existing API continues to work:

```parsley
// Before (still works)
<DataTable 
    columns={["Name", "Email"]} 
    rows={users} 
    keys={["name", "email"]}
/>

// After (simpler, if using Table)
<DataTable data={users}/>
```

### 7.2 Taking Advantage of New Features

```parsley
// Before: Manual decomposition
let Users = @DB.bind(UserSchema, "users")
let users = Users.all()
<DataTable 
    caption="Users"
    columns={["Name", "Email", "Role"]}
    rows={users.rows}
    keys={["name", "email", "role"]}
/>

// After: Direct Table usage with enhancements
let Users = @DB.bind(UserSchema, "users")
let users = Users.all()
<DataTable 
    data={users}
    caption="Users"
    empty="No users found"
    hide={["id", "password_hash"]}
    headers={{created_at: "Member Since"}}
    render={{
        email: fn(v, r) { <a href={"mailto:" + v}>v</a> }
    }}
/>
```

---

## 8. Relationship to `Table.toHTML()`

After this redesign:

| Method | Use Case | Features |
|--------|----------|----------|
| `table.toHTML()` | Quick output, CLI, debugging, dev tools | Basic `<table>`, footer support, no accessibility attributes |
| `<DataTable data={table}/>` | Production UI | Full accessibility (`scope`, `<caption>`), empty state, formatting, custom rendering |

They serve different purposes and should not converge. `Table.toHTML()` stays simple; `DataTable` provides the full-featured presentation layer.

---

## 9. Implementation Order

1. **Add `data` prop support** — Accept `Table`, derive columns/rows/keys
2. **Add empty state** — `empty` prop with colspan
3. **Add `hide` prop** — Filter columns
4. **Add `headers` prop** — Override auto-derived headers
5. **Add `align` prop** — With auto-detection from types
6. **Add `render` prop** — Custom cell rendering
7. **Add `footer` prop** — Footer rows
8. **Remove `sortable` prop** — After decision confirmed
9. **Fix class merging** — Use `+` or array join pattern
10. **Add prop spreading** — `...attrs` on `<table>`

**Estimated effort:** 2-3 hours

---

## 10. Testing Strategy

```go
func TestDataTableWithTable(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        contains []string
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
            contains: []string{"Alice"},
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
    }
    // ... test execution
}
```

---

## 11. Future Considerations

### Server-Side Sorting Enhancement (Tier 3 / Post-1.0)

A future enhancement could add server-side sorting helpers to `DataTable`:

- `sortable={true}` or `sortable={["name", "created_at"]}` to enable clickable headers
- Generates URL parameters: `?orderBy=col&orderDir=asc`
- Visual indicator for current sort column/direction
- Integrates with `Table.orderBy()` on the server

This pattern has been proven in user projects (see `bofdi/components/unsafeTable.pars`):
- Session persistence for sort state
- Link-based header clicks (works without JS)
- Font Awesome or similar icons for direction

**Not included in initial redesign** to keep scope manageable. The current broken `sortable` prop is removed; a proper implementation can be added in 1.1/1.2.

---

Not in scope for this redesign, but worth noting for later:

- **Responsive tables** — Horizontal scroll wrapper or card-based layout on mobile
- **Row selection** — Checkboxes for bulk actions
- **Expandable rows** — Click to show details
- **Column resizing** — Drag to resize
- **Sticky headers** — Keep header visible on scroll
- **Server-side sorting UI** — Clickable headers that update URL params
- **Export** — CSV/Excel download button