---
id: FEAT-090
title: "Universal Builtin Interface"
status: draft
priority: medium
created: 2026-01-14
author: "@copilot"
---

# FEAT-090: Universal Builtin Interface

## Summary
Standardize the method interface across all Parsley builtin types to ensure consistent introspection, serialization, and debugging capabilities. This includes adding universal methods like `repr()`, removing dead code, and filling gaps in type converters.

## User Story
As a Parsley developer, I want all types to have consistent introspection and conversion methods so that debugging is predictable and I can easily serialize any value.

## Acceptance Criteria
- [ ] All types have `repr()` method returning a Parsley-parseable literal
- [ ] All types have `toJSON()` method
- [ ] All pseudo-types have `toBox(opts?)` method
- [ ] All pseudo-types have consistent `toDict()` (reconstructible) and `inspect()` (internal debug) methods
- [ ] Array and Dictionary have `toHTML(opts?)` and `toMarkdown(opts?)` methods
- [ ] Dead method declarations removed (Integer `abs`, Float `abs/round/floor/ceil`)
- [ ] `toDebug()` builtin removed (replaced by `repr()`)
- [ ] `path()` constructor added (backlog #59)
- [ ] Documentation updated for all changes

## Design Decisions

### Universal Interface

All Parsley values should support:

| Method | Returns | Purpose |
|--------|---------|---------|
| `type()` | String | Type name: `"string"`, `"datetime"`, etc. |
| `repr()` | String | Parsley-parseable literal for roundtripping |
| `toJSON()` | String | JSON representation |
| `toBox(opts?)` | String | Box-drawing output |

### Pseudo-type Methods (DateTime, Money, Duration, Path, URL, Regex)

| Method | Returns | Purpose |
|--------|---------|---------|
| `toDict()` | Dictionary | Clean data for reconstruction via constructor |
| `inspect()` | Dictionary | Internal structure with `__type` marker |

**toDict() Examples:**
```parsley
@2024-01-15T10:30:00Z.toDict()  // {year: 2024, month: 1, day: 15, ...}
$50.00.toDict()                 // {amount: 50.00, currency: "USD"}
@1d2h.toDict()                  // {days: 1, hours: 2}
```

**inspect() Examples:**
```parsley
@2024-01-15T10:30:00Z.inspect() // {__type: "datetime", year: 2024, ...}
$50.00.inspect()                // {__type: "money", amount: 50.00, ...}
```

### Collection Methods (Array, Dictionary, Table)

| Method | Returns | Purpose |
|--------|---------|---------|
| `toHTML(opts?)` | String | HTML list/table |
| `toMarkdown(opts?)` | String | Markdown list/table |
| `toCSV(opts?)` | String | CSV (Array, Table only) |

Options for `toHTML`/`toMarkdown`:
- `{ordered: true}` — numbered list (arrays)
- `{headers: true}` — include header row (tables)

### String Conversion Semantics (Unchanged)

| Context | Strings | Null | Arrays | Use |
|---------|---------|------|--------|-----|
| `{x}` interpolation | Unquoted | `""` | Concatenated | User display |
| `toString(x)` | Unquoted | `""` | Concatenated | Explicit conversion |
| `log(x)` | Quoted | `null` | `[a, b]` | Debug output |
| `x.repr()` | `"\"hello\""` | `"null"` | `"[1, 2]"` | Roundtrippable literal |

### Methods to Remove

1. **Integer `abs`** — Declared in methods.go but not implemented. Use `@std/math.abs()`
2. **Float `abs`, `round`, `floor`, `ceil`** — Declared but not implemented. Use `@std/math`
3. **`toDebug()` builtin** — Replaced by `repr()` method

### Gap Fixes

1. **`path()` constructor** (backlog #59) — Add `path(string)` to create Path from variable
2. **`toBool(x)` builtin** — Consider adding for completeness (currently rely on truthiness)

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components

**Phase 1: Cleanup**
- `pkg/parsley/evaluator/methods.go` — Remove dead method declarations
- `pkg/parsley/evaluator/evaluator.go` — Remove `toDebug` builtin if present

**Phase 2: repr() Implementation**
- `pkg/parsley/evaluator/methods.go` — Add `repr()` to all type method tables
- `pkg/parsley/evaluator/eval_string_conversions.go` — Implement `objectToReprString()`

**Phase 3: toJSON() Gap Fill**
- `pkg/parsley/evaluator/methods.go` — Add `toJSON()` to types missing it
- Already have: Table, Array (likely), Dictionary (likely)
- Need to verify: DateTime, Money, Duration, Path, URL, Regex

**Phase 4: toBox() for Pseudo-types**
- `pkg/parsley/evaluator/eval_box.go` — Add toBox handlers for DateTime, Money, Duration, Path, URL, Regex

**Phase 5: toDict()/inspect() Standardization**
- `pkg/parsley/evaluator/methods.go` — Ensure all pseudo-types have both methods
- `pkg/parsley/evaluator/stdlib_datetime.go` — DateTime toDict/inspect
- `pkg/parsley/evaluator/stdlib_money.go` — Money toDict/inspect
- `pkg/parsley/evaluator/eval_duration.go` — Duration toDict/inspect
- `pkg/parsley/evaluator/eval_path.go` — Path toDict/inspect
- `pkg/parsley/evaluator/eval_url.go` — URL toDict/inspect
- `pkg/parsley/evaluator/eval_regex.go` — Regex toDict/inspect

**Phase 6: toHTML()/toMarkdown()**
- `pkg/parsley/evaluator/methods_array.go` — Array.toHTML(), Array.toMarkdown()
- `pkg/parsley/evaluator/methods_dictionary.go` — Dictionary.toHTML(), Dictionary.toMarkdown()
- Table already has these (verify)

**Phase 7: path() Constructor**
- `pkg/parsley/evaluator/evaluator.go` — Add `path()` builtin
- Resolves backlog #59

### Dependencies
- Depends on: None
- Blocks: None
- Related: FEAT-089 (toBox Phase 2), backlog #59 (path constructor)

### Edge Cases & Constraints

1. **repr() for functions** — Return `<function name>` or similar non-parseable marker
2. **repr() for recursive structures** — Detect cycles, return `<circular reference>`
3. **toDict() vs inspect()** — Must be clearly documented; toDict is for data, inspect is for debugging
4. **Path security** — `path()` constructor should respect sandbox restrictions

## Implementation Notes
*Added during/after implementation*

## Related
- Backlog: #59 (path constructor)
- Backlog: #65 (toBox color - Phase 3)
- Prior art: FEAT-089 (toBox Phase 2)
