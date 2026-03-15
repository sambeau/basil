# Design: DataTable Redesign

**Date:** 2025-01-14
**Status:** Draft
**Related:** 
- `work/reports/STANDARD-PRELUDE-REVIEW.md` §3, §9, §13
- `work/design/DESIGN-typed-value-formatting.md`

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
export DataTable = fn({caption, columns, rows, keys, id, class, sortable}) {
    <table id={id} class={"data-table" ++ if (class) { " " ++ class } else { "" }}>
        if (caption) {
            <caption>caption</caption>
        }
        <thead>
            <tr>
                for (col, idx in columns ?? []) {
                    <th scope="col">col</th>
                }
            </tr>
        </thead>
        <tbody>
            for (row in rows ?? []) {
                <tr>
                    for (key, idx in keys ?? []) {
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
| Class merging uses `++` (creates array) | Works by accident |

---

## 3. Proposed API

### 3.1 Primary Usage (with `Table`)

```parsley
let users = db.query("SELECT name, email, role FROM users")

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
| `table.columns()` | Column headers (unless `headers` prop overrides) |
| `table.rows()` | Row data |
| `table.schema()` | Type information for formatting/alignment |
| Column names | Keys for accessing row values |

```parsley
// Given:
let products = db.query("SELECT name, price, created_at FROM products")⚠️

// This:
<DataTable data={products}/>

// Is equivalent to:
<DataTable 
    columns={["name", "price", "created_at"]}
    rows={products.rows()}
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

| Value Type | Default Formatting | Alignment |
|------------|-------------------|-----------|
| `money` | `.medium()` → "£49.99" | Right |
| `datetime` | `.medium()` → "Mar 15, 2025" | Left |
| `duration` | `.medium()` → "2h 30m" | Right |
| `unit` | `.medium()` → "5.00kg" | Right |
| `integer`, `float` | As-is (string coercion) | Right |
| `boolean` | "Yes" / "No" | Center |
| `null` | "—" (em dash) | — |
| `string`, other | As-is | Left |

**Dependency:** This relies on the `objectToString()` changes from `DESIGN-typed-value-formatting.md`. Once those changes land, `DataTable` gets sensible formatting automatically.

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
    empty = "No data",
    headers = {},
    align = {},
    hide = [],
    render = {},
    format = {},
    footer,
    rowHeader = 0,
    id,
    class,
    ...attrs
}) {
    // Derive from Table if provided
    let tableColumns = if (data) { data.columns() } else { columns ?? [] }
    let tableRows = if (data) { data.rows() } else { rows ?? [] }
    let tableKeys = if (data) { data.columns() } else { keys ?? tableColumns }
    let schema = if (data) { data.schema() } else { null }
    
    // Filter out hidden columns
    let visibleColumns = tableColumns.filter(fn(c) { !hide.contains(c) })
    let visibleKeys = tableKeys.filter(fn(k) { !hide.contains(k) })
    
    // Helper: get header text for a column
    let getHeader = fn(col, idx) {
        if (headers[col]) {
            headers[col]
        } else if (schema and schema.title) {
            schema.title(col) ?? col.replace("_", " ").toTitleCase()
        } else {
            col.replace("_", " ").toTitleCase()
        }
    }
    
    // Helper: get alignment for a column
    let getAlign = fn(col) {
        if (align[col]) {
            align[col]
        } else {
            // Auto-derive from schema or first row
            "left"  // Simplified — full impl checks value types
        }
    }
    
    // Helper: format a cell value
    let formatCell = fn(value, col) {
        if (render[col]) {
            null  // Render function handles it
        } else if (value == null) {
            "—"
        } else {
            value  // objectToString() handles typed values
        }
    }
    
    let tableClass = ["data-table", class].filter(fn(c) { c != null }).join(" ")
    
    <table id={id} class={tableClass} {...attrs}>
        if (caption) {
            <caption>{caption}</caption>
        }
        <thead>
            <tr>
                for (col, idx in visibleColumns) {
                    <th scope="col" class={"align-" + getAlign(visibleKeys[idx])}>
                        {getHeader(col, idx)}
                    </th>
                }
            </tr>
        </thead>
        <tbody>
            if (tableRows.len() == 0 and empty != false) {
                <tr class="data-table-empty">
                    <td colspan={visibleColumns.len()}>{empty}</td>
                </tr>
            } else {
                for (row in tableRows) {
                    <tr>
                        for (key, idx in visibleKeys) {
                            let value = row[key]
                            let content = if (render[key]) {
                                render[key](value, row)
                            } else {
                                formatCell(value, key)
                            }
                            let alignClass = "align-" + getAlign(key)
                            
                            if (idx == rowHeader) {
                                <th scope="row" class={alignClass}>{content}</th>
                            } else {
                                <td class={alignClass}>{content}</td>
                            }
                        }
                    </tr>
                }
            }
        </tbody>
        if (footer and footer.len() > 0) {
            <tfoot>
                for (footerRow in footer) {
                    <tr>
                        for (key, idx in visibleKeys) {
                            let value = footerRow[key]
                            let content = if (render[key]) {
                                render[key](value, footerRow)
                            } else if (value != null) {
                                formatCell(value, key)
                            } else {
                                ""
                            }
                            <td class={"align-" + getAlign(key)}>{content}</td>
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

**Status:** ⏳ Needs decision

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

**Status:** ⏳ Needs decision

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

**Status:** ⏳ Needs decision

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

**Status:** ⏳ Needs decision

---

### 6.5 DECISION: Remove `sortable` Prop

**Question:** The current `sortable` prop does nothing. Should we remove it or implement it?

**Options:**
- **A) Remove it** — Sorting is a `Table.orderBy()` concern, not presentation
- **B) Implement client-side sorting** — Complex JS, scope creep
- **C) Implement server-side sorting helpers** — Add `sortHref` prop that generates header links

**Recommendation:** Option A — Remove the unused prop. Server-side sorting with `Table.orderBy()` is the Basil pattern. If we add sorting UI later (1.2+), it should be a separate component or enhancement, not baked into `DataTable`.

**Status:** ⏳ Needs decision

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
let users = db.query("SELECT * FROM users")
<DataTable 
    caption="Users"
    columns={["Name", "Email", "Role"]}
    rows={users}
    keys={["name", "email", "role"]}
/>

// After: Direct Table usage with enhancements
let users = db.query("SELECT * FROM users")
<DataTable 
    data={users}
    caption="Users"
    empty="No users found"
    hide={["id", "password_hash"]}
    headers={{created_at: "Member Since"}}
    render={{
        email: fn(v, r) { <a href={"mailto:" + v}>{v}</a> }
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

Not in scope for this redesign, but worth noting for later:

- **Responsive tables** — Horizontal scroll wrapper or card-based layout on mobile
- **Row selection** — Checkboxes for bulk actions
- **Expandable rows** — Click to show details
- **Column resizing** — Drag to resize
- **Sticky headers** — Keep header visible on scroll
- **Server-side sorting UI** — Clickable headers that update URL params
- **Export** — CSV/Excel download button