---
id: FEAT-121
title: "Unified Formatter API for Parsley Builtins"
status: implemented
priority: high
created: 2025-01-20
implemented: 2025-02-24
author: "@ai"
---

# FEAT-121: Unified Formatter API for Parsley Builtins

## Summary

Implement a unified, terse, and composable formatting API for all Parsley builtin types. The design replaces the verbose `.format()` method with `.fmt()`, adds style sugar methods (`.short()`, `.long()`, etc.), and ensures all types have consistent serialization methods (`repr()`, `toJSON()`, `inspect()`, `toBox()`).

## User Story

As a Parsley developer, I want brief and consistent formatting methods so that I can easily format values in templates without verbose method chains, and so that all types behave predictably.

## Background

The current formatting API has several problems:
1. **Verbosity** — `.format("short")` is too long for frequent template use
2. **Inconsistency** — Different types have different method signatures
3. **Gaps** — Some types lack `repr()`, `toJSON()`, or locale support
4. **Discoverability** — No unified pattern across types

See `work/design/FORMATTER_DESIGN.md` for the approved design and `work/reports/FORMATTER_AUDIT.md` for the full audit.

## Acceptance Criteria

### Part 1: Core `.fmt()` Method
- [x] Add `.fmt()` method to all value types (Number, Money, DateTime, Duration, Unit)
- [x] Implement overloads:
  - `.fmt()` — medium style, default locale
  - `.fmt(n)` — integer precision (Number, Money, Unit only)
  - `.fmt("style")` — named style
  - `.fmt("style", "locale")` — style with locale
  - `.fmt({...})` — full options dictionary
- [x] Keep `.format()` as an alias for backward compatibility

### Part 2: Style Sugar Methods
- [x] Add `.short(opts?)` — compact representation
- [x] Add `.medium(opts?)` — balanced (default)
- [x] Add `.long(opts?)` — verbose/full precision
- [x] Add `.full(opts?)` — maximum context (Money, DateTime, Unit only)
- [x] Each method accepts optional: locale string OR options dictionary

### Part 3: Standardize Styles Across Types

| Type | short | medium (default) | long | full |
|------|-------|------------------|------|------|
| Number | `"1.2M"` | `"1,235"` | `"1,234.57"` | — |
| Money | `"$1K"` | `"$1,235"` | `"$1,234.56"` | `"1,234.56 US dollars"` |
| DateTime | `"12/25/24"` | `"Dec 25, 2024"` | `"December 25, 2024"` | `"Wednesday, December 25, 2024"` |
| Duration | `"2h"` | `"2 hours"` | `"2 hours 30 min"` | — |
| Unit | `"5m"` | `"5.00m"` | `"5 metres"` | `"5 metres (16.4 ft)"` |

### Part 4: Universal Serialization Methods
- [x] Add `repr()` to all types that lack it (DateTime, Duration, Bool, Null, URL, Path)
- [x] Add `toJSON()` to all types that lack it
- [x] Add `inspect()` to all value types (returns debug dictionary with `__type`)
- [x] Add `toBox()` to all types that lack it

### Part 5: Collection Formatting
- [x] Arrays: `.fmt("and")` → `"A, B, and C"`
- [x] Arrays: `.fmt("or")` → `"A, B, or C"`
- [x] Arrays: `.fmt("and", "locale")` → localized conjunction
- [x] Add `toMarkdown()` to Array and Dictionary (already existed)
- [x] Add `toHTML()` to Array and Dictionary (already existed)
- [x] Add `toCSV()` to Array (already existed)

### Part 6: Locale Support
- [x] Add locale support to Unit formatting (currently missing)
- [x] Ensure all types respect locale for:
  - Number separators and decimal marks
  - Currency symbol placement
  - Date/time formats and month/weekday names
  - Conjunction words for arrays
  - Unit name spelling (metre vs meter)

### Part 7: Options Dictionary
- [x] Support `style` key (string): `"short"`, `"medium"`, `"long"`, `"full"`
- [x] Support `locale` key (string): BCP 47 locale code
- [x] Support `precision` key (integer): decimal places (Number, Money, Unit)
- [x] Support `compound` key (boolean): compound format for Units

### Part 8: Documentation
- [ ] Update `docs/parsley/reference.md` with new API (deferred)
- [ ] Update `docs/parsley/CHEATSHEET.md` with formatting examples (deferred)
- [ ] Add manual pages for new methods in `docs/parsley/manual/` (deferred)
- [ ] Update examples in `examples/parsley/` (deferred)

### Part 9: Tests
- [x] Add tests for `.fmt()` overloads for each type
- [x] Add tests for style methods for each type
- [x] Add tests for locale variations
- [x] Add tests for options dictionary
- [x] Add tests for array conjunction formatting

## Design Decisions

- **`.fmt()` over `.format()`**: Brevity is critical for template interpolation. `.fmt()` is common in modern languages (Rust, Go).
- **Style methods as sugar**: `.short()` is cleaner than `.fmt("short")` in templates.
- **No printf-style format strings**: Named styles + options dict covers 90% of use cases with better discoverability.
- **Options dictionary accepts locale string shorthand**: `.short("de-DE")` is equivalent to `.short({locale: "de-DE"})`.
- **Backward compatibility**: `.format()` remains as an alias; deprecation deferred to 2.0.

## Interaction with FEAT-120 (Remove print/println)

FEAT-120 removes `print()`, `println()`, and `printf()`. This affects formatter implementation:

1. **`printf()` removal**: Users needing template rendering should use `.render()` on strings, not `printf()`. No formatter replacement needed.
2. **Output model**: Since Parsley uses expression-based output, formatted values are typically used directly in interpolation:
   ```parsley
   <span class="price">{price.short()}</span>
   ```
3. **No interaction with formatting API**: The formatter API is independent of output mechanism.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components

**Value type methods:**
- `pkg/parsley/evaluator/methods_numeric.go` — Number `.fmt()`, style methods
- `pkg/parsley/evaluator/methods_money.go` — Money `.fmt()`, style methods
- `pkg/parsley/evaluator/methods_datetime.go` — DateTime `.fmt()`, style methods
- `pkg/parsley/evaluator/methods_duration.go` — Duration `.fmt()`, style methods
- `pkg/parsley/evaluator/methods_unit.go` — Unit `.fmt()`, style methods, locale support

**Collection methods:**
- `pkg/parsley/evaluator/methods_array.go` — Array `.fmt()`, `toMarkdown()`, `toCSV()`, `toHTML()`
- `pkg/parsley/evaluator/methods_dict.go` — Dictionary `toMarkdown()`, `toHTML()`

**Core serialization:**
- `pkg/parsley/evaluator/methods_*.go` — Add `repr()`, `toJSON()`, `inspect()`, `toBox()` where missing

**Method registry:**
- `pkg/parsley/evaluator/introspect.go` — Update `BuiltinMetadata` for new methods

### Dependencies
- Depends on: None
- Blocks: None
- Related: FEAT-120 (print removal is independent but happening in parallel)

### Edge Cases & Constraints

1. **Precision on non-numeric types** — `.fmt(2)` on DateTime should error with helpful message
2. **Full style on types without it** — Number and Duration don't support `.full()`; should error clearly
3. **Locale fallback** — Unknown locales should fall back to `"en-US"` with no error
4. **Empty arrays** — `[].fmt("and")` should return `""`
5. **Single-element arrays** — `["Alice"].fmt("and")` should return `"Alice"` (no conjunction)
6. **Two-element arrays** — `["Alice", "Bob"].fmt("and")` should return `"Alice and Bob"` (no comma)
7. **Unit compound format** — Only certain unit types support compound (length, not temperature)

## Implementation Notes

### Method Signature Pattern

All style methods should follow this pattern:
```go
func (v *Value) short(args ...interface{}) (string, error) {
    opts := parseFormatOpts(args)
    opts.Style = "short"
    return v.formatWithOpts(opts)
}
```

### Options Parsing

```go
func parseFormatOpts(args []interface{}) FormatOpts {
    switch len(args) {
    case 0:
        return FormatOpts{Style: "medium", Locale: "en-US"}
    case 1:
        switch v := args[0].(type) {
        case int:
            return FormatOpts{Precision: v}
        case string:
            if isLocale(v) {
                return FormatOpts{Locale: v}
            }
            return FormatOpts{Style: v}
        case map[string]interface{}:
            return optsFromDict(v)
        }
    case 2:
        // style, locale
        return FormatOpts{Style: args[0].(string), Locale: args[1].(string)}
    }
}
```

### Array Conjunction Localization

Conjunction words by locale:
- `en-US`: "and" / "or"
- `de-DE`: "und" / "oder"
- `fr-FR`: "et" / "ou"
- `es-ES`: "y" / "o"

Oxford comma rules vary by locale (English uses it, most others don't).

## Implementation Priority

### P0 — Must Have (Core Functionality)
1. `.fmt()` method with all overloads
2. Style sugar methods (`.short()`, `.medium()`, `.long()`, `.full()`)
3. `repr()` and `toJSON()` for all types
4. Locale support for Units

### P1 — Should Have (Completeness)
5. `inspect()` for all value types
6. `toBox()` for remaining types
7. Array conjunction formatting
8. Collection serialization methods

### P2 — Nice to Have (Polish)
9. Extended locale support
10. Compound unit formatting
11. `toMarkdown()`, `toHTML()` for collections

## Test Plan

1. **Unit tests per type**: Each type gets tests for all `.fmt()` overloads
2. **Style consistency tests**: Verify style outputs match spec table
3. **Locale tests**: Test key locales (en-US, de-DE, fr-FR)
4. **Edge case tests**: Empty arrays, single elements, unknown locales
5. **Round-trip tests**: `repr()` output should be parseable back to equivalent value

## Related

- Design: `work/design/FORMATTER_DESIGN.md`
- Audit: `work/reports/FORMATTER_AUDIT.md`
- Related: FEAT-120 (Remove print/println)