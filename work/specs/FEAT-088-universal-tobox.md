---
id: FEAT-088
title: "Universal toBox() Method"
status: draft
priority: medium
created: 2026-01-14
author: "@human"
---

# FEAT-088: Universal toBox() Method

## Summary
Extend the `toBox()` method from Tables to all Parsley value types, enabling ASCII box rendering for arrays, dictionaries, and scalar values. This provides a consistent way to format data for CLI output and Markdown documentation.

## User Story
As a Parsley developer building CLI tools, I want to render any value as an ASCII box so that I can create readable terminal output without manual formatting.

## Acceptance Criteria

### Phase 1: Core Implementation
- [ ] Arrays render vertically by default with `toBox()`
- [ ] Arrays render horizontally with `toBox({direction: "horizontal"})`
- [ ] Arrays of arrays render as grids
- [ ] Dictionaries render as key-value rows with `toBox()`
- [ ] Dictionaries render keys only with `toBox({keys: true})`
- [ ] Scalar values (string, number, bool, null) render in a single box
- [ ] Nested complex values display as inline summaries (not recursive boxes)
- [ ] Alignment option: `{align: "left" | "right" | "center"}`

### Phase 2: Polish (Nice-to-Have)
- [ ] Style options: `{style: "single" | "double" | "ascii" | "rounded"}`
- [ ] Title option: `{title: "My Data"}`
- [ ] Width control: `{maxWidth: 40}` with ellipsis truncation

## Design Decisions

### Rendering Returns String
`toBox()` returns a string, not rendered output. This matches Table's existing behavior and works for CLI, piping, and Markdown embedding.

### Arrays Default to Vertical
Vertical layout is more readable for lists and matches typical CLI output expectations. Horizontal available via option.

### Nested Values Show Inline Summaries
Complex nested values (dict in array, array in dict) display as inline text like `{name: "Alice", age: 30}` or `[1, 2, 3]` rather than attempting recursive box rendering. Tables show as `table(N rows)`. This keeps output readable and avoids explosion of nested boxes.

### No Recursive Boxing
`toBox()` does not recursively box nested values. A boxed array containing dictionaries shows the dicts as inline text, not as nested boxes. This is intentional for readability.

### Dictionary Layout
Dictionaries render as a two-column table (key | value) by default, similar to how Table already works. The `{keys: true}` option renders just the keys in a single row.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `pkg/parsley/evaluator/methods.go` — Add `toBox` method dispatch for Array, Dictionary, scalar types
- `pkg/parsley/evaluator/eval_tobox.go` — New file for toBox rendering logic (or extend existing)
- `pkg/parsley/evaluator/stdlib_table.go` — May share box-drawing utilities with existing Table.toBox()

### Type-Specific Behavior

#### Arrays
```parsley
["apple", "banana", "cherry"].toBox()
```
```
┌────────┐
│ apple  │
├────────┤
│ banana │
├────────┤
│ cherry │
└────────┘
```

```parsley
["apple", "banana", "cherry"].toBox({direction: "horizontal"})
```
```
┌────────┬────────┬────────┐
│ apple  │ banana │ cherry │
└────────┴────────┴────────┘
```

#### Array of Arrays (Grid)
```parsley
[[1, 2, 3], [4, 5, 6], [7, 8, 9]].toBox()
```
```
┌───┬───┬───┐
│ 1 │ 2 │ 3 │
├───┼───┼───┤
│ 4 │ 5 │ 6 │
├───┼───┼───┤
│ 7 │ 8 │ 9 │
└───┴───┴───┘
```

#### Dictionaries
```parsley
{name: "Alice", age: 30, city: "NYC"}.toBox()
```
```
┌──────┬───────┐
│ name │ Alice │
├──────┼───────┤
│ age  │ 30    │
├──────┼───────┤
│ city │ NYC   │
└──────┴───────┘
```

```parsley
{name: "Alice", age: 30, city: "NYC"}.toBox({keys: true})
```
```
┌──────┬─────┬──────┐
│ name │ age │ city │
└──────┴─────┴──────┘
```

#### Scalars
```parsley
42.toBox()
```
```
┌────┐
│ 42 │
└────┘
```

```parsley
"hello world".toBox()
```
```
┌─────────────┐
│ hello world │
└─────────────┘
```

```parsley
true.toBox()
```
```
┌──────┐
│ true │
└──────┘
```

#### Nested Complex Values
```parsley
[{name: "Alice"}, {name: "Bob"}].toBox()
```
```
┌─────────────────────┐
│ {name: "Alice"}     │
├─────────────────────┤
│ {name: "Bob"}       │
└─────────────────────┘
```

```parsley
{users: [{name: "Alice"}], count: 1}.toBox()
```
```
┌───────┬─────────────────────┐
│ users │ [{name: "Alice"}]   │
├───────┼─────────────────────┤
│ count │ 1                   │
└───────┴─────────────────────┘
```

### Options Schema

```parsley
toBox({
    direction: "vertical" | "horizontal",  // Arrays only, default: "vertical"
    align: "left" | "right" | "center",    // Cell alignment, default: "left"
    keys: boolean,                          // Dicts only: show keys without values
    
    // Phase 2 options:
    style: "single" | "double" | "ascii" | "rounded",
    title: string,
    maxWidth: number
})
```

### Box-Drawing Characters

| Style | TL | TR | BL | BR | H | V | Cross | T-down | T-up | T-right | T-left |
|-------|----|----|----|----|---|---|-------|--------|------|---------|--------|
| single | ┌ | ┐ | └ | ┘ | ─ | │ | ┼ | ┬ | ┴ | ├ | ┤ |
| double | ╔ | ╗ | ╚ | ╝ | ═ | ║ | ╬ | ╦ | ╩ | ╠ | ╣ |
| rounded | ╭ | ╮ | ╰ | ╯ | ─ | │ | ┼ | ┬ | ┴ | ├ | ┤ |
| ascii | + | + | + | + | - | \| | + | + | + | + | + |

### Dependencies
- Depends on: None (Table.toBox() already exists, can share utilities)
- Blocks: None

### Edge Cases & Constraints

1. **Empty array** — Renders as empty box: `┌┐\n└┘` or `(empty)`
2. **Empty dictionary** — Renders as empty box or `(empty)`
3. **Very long strings** — Phase 1: no truncation. Phase 2: `maxWidth` with `...`
4. **Unicode in values** — Must handle correctly for width calculation
5. **null values** — Display as `null` text
6. **Jagged arrays** — Arrays of arrays with different lengths: pad shorter rows

## Deferred Ideas

These were discussed but deferred for potential future consideration:

### Color Support
ANSI terminal colors for type-based highlighting:
- Strings: cyan
- Numbers: yellow
- Booleans: green/red
- null: dim gray

Would add `{color: true}` option. Deferred because it adds complexity and doesn't work in all contexts (piped output, some terminals).

### Schema-Driven Formatting
Use a schema to control column widths, alignment, and value formatting:
```parsley
let schema = {
    name: {width: 20, align: "left"},
    price: {width: 10, align: "right", format: "money"}
}
users.toBox({schema: schema})
```
Powerful for admin interfaces but adds significant complexity.

### Interactive Boxes
Collapsible nested values in DevTools/REPL. Requires JavaScript and state management. Doesn't work in plain terminal. Not worth the complexity.

### Per-Column Alignment
Array of alignment values for tables/grids:
```parsley
table.toBox({align: ["left", "right", "center"]})
```
Could be added in Phase 2 if needed.

## Implementation Notes
*Added during/after implementation*

## Related
- Existing: Table.toBox() in `stdlib_table.go`
- Related: FEAT-087 Builtin Table Type (recently completed)
