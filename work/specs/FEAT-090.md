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

**This spec serves dual purposes:**
1. **Implementation guide** — Defines the work to be done
2. **Verification checklist** — Used to verify implementation completeness
3. **Documentation source** — Drives updates to reference docs, manuals, and examples

## Philosophy: Break, Don't Deprecate

**We prefer breaking changes over deprecation.** When removing or changing methods:

- **Remove immediately** — Don't leave deprecated methods lingering
- **Fix all tests** — Update test files to use new APIs
- **Fix all documentation** — Update guides, examples, and reference docs
- **Fix all examples** — Ensure example code uses current APIs

Deprecation warnings encourage bad patterns to propagate through examples and user code. A clean break forces immediate migration and keeps the codebase consistent.

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
- [ ] All tests pass (including updates for removed methods)
- [ ] All documentation updated
- [ ] All examples updated

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

### Breaking Changes Checklist

When removing methods/builtins, audit and fix these locations:

**Tests** (must all pass after changes):
- `pkg/parsley/tests/*.go` — Unit tests for removed methods
- `pkg/parsley/evaluator/*_test.go` — Evaluator tests
- `server/*_test.go` — Server integration tests

**Documentation** (must be updated):
- `docs/parsley/reference.md` — Remove/update method signatures
- `docs/parsley/CHEATSHEET.md` — Update if removed methods mentioned
- `docs/parsley/manual/builtins/*.md` — Update type manual pages
- `docs/guide/*.md` — Check for usage in guides

**Examples** (must use current APIs):
- `examples/parsley/*.pars` — Standalone Parsley examples
- `examples/basil/**/*.pars` — Basil handler examples
- `examples/parts/**/*.pars` — Parts examples

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

### Test Work Required

Each phase requires corresponding test updates:

| Phase | Test Files | Work |
|-------|-----------|------|
| 1 (Cleanup) | `methods_test.go` | Remove tests for `abs`, `round`, `floor`, `ceil` on numeric types; remove `toDebug` tests |
| 2 (repr) | `repr_test.go` (new) | Comprehensive tests for all types: primitives, pseudo-types, collections, edge cases (cycles, functions) |
| 3 (toJSON) | `json_test.go` or per-type tests | Add tests for types that didn't have toJSON |
| 4 (toBox) | `tobox_test.go` | Add tests for DateTime, Money, Duration, Path, URL, Regex |
| 5 (toDict/inspect) | Per-type test files | Verify toDict returns reconstructible data, inspect includes `__type` |
| 6 (toHTML/toMarkdown) | `array_test.go`, `dictionary_test.go` | Test output format, options handling |
| 7 (path constructor) | `path_test.go` | Constructor with valid/invalid strings, sandbox restrictions |

**Test verification command:** `make test` must pass after each phase.

### Documentation Work Required

| Phase | Documentation Updates |
|-------|----------------------|
| 1 (Cleanup) | Remove `abs`, `round`, `floor`, `ceil` from Integer/Float in reference.md and manual pages; remove `toDebug` from builtins list; add note about `@std/math` for math operations |
| 2 (repr) | Add `repr()` to all type sections in reference.md; create `docs/parsley/manual/builtins/repr.md` or add to each type's manual page |
| 3 (toJSON) | Update type manual pages to show toJSON availability |
| 4 (toBox) | Update pseudo-type manual pages with toBox examples |
| 5 (toDict/inspect) | Document difference between toDict and inspect; update each pseudo-type manual page |
| 6 (toHTML/toMarkdown) | Add to Array and Dictionary manual pages with examples |
| 7 (path constructor) | Add `path()` to constructors section in reference.md; create manual page if warranted |

**Documentation locations:**
- `docs/parsley/reference.md` — Comprehensive reference (all changes)
- `docs/parsley/CHEATSHEET.md` — Quick reference (key changes only)
- `docs/parsley/manual/builtins/*.md` — Per-type detailed docs

### Example Updates Required

Search and update any examples using removed methods:

```bash
# Find uses of removed methods
grep -r "\.abs\(\)" examples/
grep -r "\.round\(\)" examples/
grep -r "\.floor\(\)" examples/
grep -r "\.ceil\(\)" examples/
grep -r "toDebug\(" examples/
```

Replace with `@std/math` equivalents or `repr()` as appropriate.

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

## Verification Checklist

Use this checklist to verify implementation is complete:

### Code Verification
- [ ] `grep -r "abs.*Integer\|Integer.*abs" pkg/` returns no method declarations
- [ ] `grep -r "round.*Float\|Float.*round" pkg/` returns no method declarations  
- [ ] `grep -r "toDebug" pkg/` returns no builtin definitions
- [ ] `grep -r "\.repr()" pkg/parsley/tests/` shows tests for all types
- [ ] `make test` passes
- [ ] `make check` passes (build + test + lint)

### Documentation Verification
- [ ] `grep -r "\.abs()" docs/` returns no hits (or only @std/math references)
- [ ] `grep -r "toDebug" docs/` returns no hits
- [ ] `docs/parsley/reference.md` includes `repr()` for all types
- [ ] `docs/parsley/reference.md` includes `path()` constructor
- [ ] Each pseudo-type manual page documents `toDict()` and `inspect()`

### Example Verification
- [ ] `grep -r "\.abs()\|\.round()\|\.floor()\|\.ceil()" examples/` returns no hits
- [ ] `grep -r "toDebug" examples/` returns no hits
- [ ] All example files in `examples/` execute without errors

## Related
- Backlog: #59 (path constructor)
- Backlog: #65 (toBox color - Phase 3)
- Prior art: FEAT-089 (toBox Phase 2)
