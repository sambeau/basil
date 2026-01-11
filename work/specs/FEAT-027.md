---
id: FEAT-027
title: "Collection Insert Methods"
status: complete
priority: medium
created: 2025-12-05
completed: 2025-12-05
author: "@sambeau"
---

# FEAT-027: Collection Insert Methods

## Summary
Add insert and append methods to Parsley's collection types (arrays, dictionaries, tables) enabling positional insertion of elements. This gives users fine-grained control over collection mutation, particularly important for tables which are opaque to users.

## User Story
As a Parsley developer, I want to insert elements at specific positions in collections so that I can build and manipulate data structures without complex slice operations.

## Acceptance Criteria

### Array Methods
- [ ] `insert(index, value)` — Insert value before the given index
- [ ] Returns a new array (immutable)
- [ ] Index 0 inserts at beginning, index equal to length appends at end
- [ ] Negative indices work from end (like Python)

### Dictionary Methods
- [ ] `insertAfter(existingKey, newKey, value)` — Insert after existing key
- [ ] `insertBefore(existingKey, newKey, value)` — Insert before existing key
- [ ] Returns a new dictionary (immutable)
- [ ] Error if existingKey doesn't exist

### Table Row Methods
- [ ] `appendRow(row)` — Append row to end of table
- [ ] `insertRowAt(index, row)` — Insert row before given index
- [ ] Returns a new table (immutable)
- [ ] Row must have matching column structure

### Table Column Methods
- [ ] `appendCol(name, values)` — Append column with values array
- [ ] `appendCol(name, fn)` — Append computed column using function
- [ ] `insertColAfter(existingCol, name, values|fn)` — Insert after existing column
- [ ] `insertColBefore(existingCol, name, values|fn)` — Insert before existing column
- [ ] Returns a new table (immutable)
- [ ] Function receives row as parameter: `fn(row) { row.firstName[0] }`
- [ ] Values array must match row count

## Design Decisions

- **Python-style insert for arrays**: `insert(index, value)` inserts *before* index, matching Python's `list.insert(i, x)` and Go's `slices.Insert()`. This is the dominant convention.

- **Key-based (not index-based) for dictionaries**: Use `insertAfter(key)` / `insertBefore(key)` rather than index-based insertion. Dictionary keys are the natural addressing mechanism, and index-based access would be confusing.

- **Immutable operations**: All methods return new collections rather than mutating in place. This matches existing Parsley semantics (e.g., `.where()`, `.select()`).

- **Append naming**: Use `append` (not `add`) to avoid confusion with mathematical addition and match Python/Go conventions.

- **Function support for columns**: Column methods accept either a values array OR a function. Functions enable computed columns: `appendCol("initials", fn(row) { row.first[0] + row.last[0] })`.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `pkg/parsley/evaluator/methods.go` — Add array and dictionary methods
- `pkg/parsley/evaluator/stdlib_table.go` — Add table row/column methods
- `pkg/parsley/tests/` — Unit tests for all new methods
- `docs/parsley/reference.md` — Document new methods
- `docs/parsley/CHEATSHEET.md` — Add gotchas if any

### Dependencies
- Depends on: FEAT-026 (Ordered Dictionaries) — Required for dictionary insert methods
- Blocks: None

### Edge Cases & Constraints

**Arrays:**
1. `insert(0, x)` — Equivalent to `unshift(x)`
2. `insert(len, x)` — Equivalent to `push(x)`
3. `insert(-1, x)` — Insert before last element
4. `insert(100, x)` on 3-element array — Should error or clamp? (Recommend: error)

**Dictionaries:**
1. `insertAfter("nonexistent", k, v)` — Error: key not found
2. `insertBefore` on first key — Valid, becomes new first key
3. `insertAfter` on last key — Valid, becomes new last key
4. Key already exists — Error: duplicate key

**Tables:**
1. Row with missing columns — Error
2. Row with extra columns — Error (or ignore extras?)
3. Values array length mismatch — Error
4. Function throws — Propagate error
5. Empty table — Should still work

### Method Signatures (Go Implementation)

```go
// Array
case "insert":
    // args: index (number), value (any)
    // returns: new array with value inserted before index

// Dictionary  
case "insertAfter":
    // args: existingKey (string), newKey (string), value (any)
    // returns: new dictionary with k/v inserted after existingKey

case "insertBefore":
    // args: existingKey (string), newKey (string), value (any)
    // returns: new dictionary with k/v inserted before existingKey

// Table (in stdlib_table.go builtins)
// appendRow(table, row)
// insertRowAt(table, index, row)
// appendCol(table, name, values|fn)
// insertColAfter(table, existingCol, name, values|fn)
// insertColBefore(table, existingCol, name, values|fn)
```

## Example Usage

### Array Insert
```parsley
let arr = [1, 2, 3]
let result = arr.insert(1, "new")  // [1, "new", 2, 3]
let atEnd = arr.insert(3, "end")   // [1, 2, 3, "end"]
let atStart = arr.insert(0, "first")  // ["first", 1, 2, 3]
```

### Dictionary Insert
```parsley
let dict = {a: 1, b: 2, c: 3}
let result = dict.insertAfter("a", "a2", 1.5)  // {a: 1, a2: 1.5, b: 2, c: 3}
let before = dict.insertBefore("b", "a2", 1.5) // {a: 1, a2: 1.5, b: 2, c: 3}
```

### Table Row Operations
```parsley
let users = [
    {name: "Alice", age: 30},
    {name: "Bob", age: 25}
]

let withNewUser = users.appendRow({name: "Charlie", age: 35})
let inserted = users.insertRowAt(1, {name: "Charlie", age: 35})
// Result: Alice at 0, Charlie at 1, Bob at 2
```

### Table Column Operations
```parsley
let users = [
    {first: "Alice", last: "Smith"},
    {first: "Bob", last: "Jones"}
]

// With values array
let withAge = users.appendCol("age", [30, 25])

// With computed function
let withInitials = users.appendCol("initials", fn(row) {
    row.first[0] + row.last[0]
})
// Result: [{first: "Alice", last: "Smith", initials: "AS"}, ...]

// Insert after specific column
let withMiddle = users.insertColAfter("first", "middle", ["M.", "R."])
```

## Implementation Notes
*Added during/after implementation*

## Related
- Plan: `docs/plans/FEAT-027-plan.md`
- Depends on: FEAT-026 (Ordered Dictionaries)
