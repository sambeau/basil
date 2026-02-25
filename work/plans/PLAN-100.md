---
id: PLAN-100
feature: FEAT-121
title: "Implementation Plan: Unified Formatter API for Parsley Builtins"
status: complete
created: 2025-01-20
completed: 2025-02-24
---

# Implementation Plan: FEAT-121

## Overview

Implement a unified, terse, and composable formatting API for all Parsley builtin types. This adds `.fmt()` as the primary formatting method, style sugar methods (`.short()`, `.medium()`, `.long()`, `.full()`), and ensures all types have consistent serialization methods (`repr()`, `toJSON()`, `inspect()`, `toBox()`).

## Prerequisites

- [x] Design approved (`work/design/FORMATTER_DESIGN.md`)
- [x] Audit completed (`work/reports/FORMATTER_AUDIT.md`)
- [x] Method registry pattern established (FEAT-111)

## Architecture Notes

### Existing Patterns

The codebase uses two patterns for method dispatch:

1. **Registry-based** (Integer, Float, Money, Unit, String): Methods defined in `MethodRegistry` maps, dispatched via `dispatchFromRegistry()`. Files: `methods_numeric.go`, `methods_money.go`, `methods_unit.go`, `methods_string.go`.

2. **Switch-based** (DateTime, Duration, Array, Dictionary, Path, URL, Bool, Null): Methods defined in `evalXxxMethod()` functions with switch statements. File: `methods.go`.

This plan follows existing patterns—registry types get registry entries, switch types get switch cases.

### Common FormatOpts Structure

All formatting methods will share a common options parsing pattern:

```go
type FormatOpts struct {
    Style     string // "short", "medium", "long", "full"
    Locale    string // BCP 47 locale code
    Precision int    // decimal places (-1 = default)
    Compound  bool   // compound format for units
}
```

---

## Tasks

### Phase 1: Core Infrastructure

#### Task 1.1: Add FormatOpts and Parsing Helpers
**Files**: `pkg/parsley/evaluator/format_opts.go` (new)
**Estimated effort**: Small

Steps:
1. Create `FormatOpts` struct with Style, Locale, Precision, Compound fields
2. Implement `parseFormatArgs()` to handle all overload patterns:
   - `()` → medium style, en-US locale
   - `(n int)` → precision
   - `("style")` → named style
   - `("style", "locale")` → style + locale
   - `({...})` → options dictionary
3. Implement `parseStyleMethodArgs()` for style sugar methods:
   - `()` → default opts
   - `("locale")` → locale string
   - `({...})` → options dictionary
4. Add `isLocale()` helper to distinguish locale strings from style names

Tests:
- `parseFormatArgs` with all overload combinations
- `parseStyleMethodArgs` with locale and dict
- Edge cases: empty dict, unknown keys, invalid types

---

### Phase 2: Value Type `.fmt()` Methods

#### Task 2.1: Number `.fmt()` (Integer and Float)
**Files**: `pkg/parsley/evaluator/methods_numeric.go`
**Estimated effort**: Small

Steps:
1. Add `fmt` entry to `IntegerMethodRegistry` and `FloatMethodRegistry` with arity `"0-2"`
2. Implement `integerFmt()` and `floatFmt()` using `parseFormatArgs()`
3. Implement style formatting:
   - `short`: compact notation via `humanizeNumber()` (1.2M, 500K)
   - `medium`: locale-aware with thousand separators (existing `formatNumberWithLocale`)
   - `long`: full precision with separators
4. Add `format` as alias pointing to same implementation

Tests:
- `123456.fmt()` → `"123,456"`
- `1234567.fmt("short")` → `"1.2M"`
- `1234.5678.fmt(2)` → `"1,234.57"`
- `1234.5.fmt("medium", "de-DE")` → `"1.234,5"`
- `1234.fmt({style: "short", locale: "de-DE"})` → `"1,2K"`

---

#### Task 2.2: Money `.fmt()`
**Files**: `pkg/parsley/evaluator/methods_money.go`
**Estimated effort**: Small

Steps:
1. Update `format` entry to arity `"0-2"` and rename impl to `moneyFmt`
2. Add `fmt` as alias
3. Implement style formatting:
   - `short`: compact (`$1K`, `€500`)
   - `medium`: standard currency format (existing behavior)
   - `long`: full precision (`$1,234.56`)
   - `full`: spelled out currency name (`1,234.56 US dollars`)
4. Support precision override for decimal places

Tests:
- `$1234.56.fmt()` → `"$1,234.56"`
- `$1234.56.fmt("short")` → `"$1K"`
- `$1234.56.fmt("full")` → `"1,234.56 US dollars"`
- `$1234.56.fmt("full", "de-DE")` → `"1.234,56 US-Dollar"`
- `$1234.567.fmt(2)` → `"$1,234.57"`

---

#### Task 2.3: DateTime `.fmt()`
**Files**: `pkg/parsley/evaluator/methods.go` (evalDatetimeMethod)
**Estimated effort**: Small

Steps:
1. Add `fmt` case as alias to existing `format` logic
2. Verify existing style support: short, medium, long, full
3. Ensure locale parameter works correctly
4. Default style should be `medium` (currently `long` - update default)

Tests:
- `@2024-12-25.fmt()` → `"Dec 25, 2024"` (medium)
- `@2024-12-25.fmt("short")` → `"12/25/24"`
- `@2024-12-25.fmt("long")` → `"December 25, 2024"`
- `@2024-12-25.fmt("full")` → `"Wednesday, December 25, 2024"`
- `@2024-12-25.fmt("long", "de-DE")` → `"25. Dezember 2024"`

---

#### Task 2.4: Duration `.fmt()`
**Files**: `pkg/parsley/evaluator/methods.go` (evalDurationMethod)
**Estimated effort**: Medium

Steps:
1. Add `fmt` case as alias to existing `format` logic
2. Add style support (currently only locale):
   - `short`: compact (`2h`, `3d`)
   - `medium`: readable (`2 hours`, `3 days`) - existing behavior
   - `long`: components (`2 hours 30 minutes`)
3. Note: Duration does not support `full` style

Tests:
- `@duration{hours: 2}.fmt()` → `"2 hours"`
- `@duration{hours: 2}.fmt("short")` → `"2h"`
- `@duration{hours: 2, minutes: 30}.fmt("long")` → `"2 hours 30 minutes"`
- `@duration{hours: 2}.fmt("full")` → error (unsupported)

---

#### Task 2.5: Unit `.fmt()`
**Files**: `pkg/parsley/evaluator/methods_unit.go`
**Estimated effort**: Medium

Steps:
1. Update `format` entry to support style and locale arguments
2. Add `fmt` as alias
3. Implement style formatting:
   - `short`: symbol only (`5m`, `10kg`)
   - `medium`: with precision (existing `format` behavior)
   - `long`: spelled out (`5 metres`, `10 kilograms`)
   - `full`: with conversion (`5 metres (16.4 ft)`)
4. **Add locale support** (currently missing - flagged in audit)
   - Decimal mark: `.` (en-US) vs `,` (de-DE)
   - Unit spelling: `metres` (en-GB) vs `meters` (en-US)

Tests:
- `#5m.fmt()` → `"5.00m"`
- `#5m.fmt("short")` → `"5m"`
- `#5m.fmt("long")` → `"5 metres"`
- `#5m.fmt("full")` → `"5 metres (16.4 ft)"`
- `#5.5m.fmt(1)` → `"5.5m"`
- `#5m.fmt("long", "en-US")` → `"5 meters"`

---

### Phase 3: Style Sugar Methods

#### Task 3.1: Number Style Methods
**Files**: `pkg/parsley/evaluator/methods_numeric.go`
**Estimated effort**: Small

Steps:
1. Add `short`, `medium`, `long` to both registries with arity `"0-1"`
2. Each delegates to `integerFmt`/`floatFmt` with style preset
3. Accept optional locale string or options dict

Tests:
- `1234567.short()` → `"1.2M"`
- `1234567.medium()` → `"1,234,567"`
- `1234567.long()` → `"1,234,567.00"` (with decimal places)
- `1234567.short("de-DE")` → `"1,2 Mio."`

---

#### Task 3.2: Money Style Methods
**Files**: `pkg/parsley/evaluator/methods_money.go`
**Estimated effort**: Small

Steps:
1. Add `short`, `medium`, `long`, `full` to registry with arity `"0-1"`
2. Each delegates to `moneyFmt` with style preset

Tests:
- `$1234.short()` → `"$1K"`
- `$1234.medium()` → `"$1,234"`
- `$1234.56.long()` → `"$1,234.56"`
- `$1234.56.full()` → `"1,234.56 US dollars"`

---

#### Task 3.3: DateTime Style Methods
**Files**: `pkg/parsley/evaluator/methods.go` (evalDatetimeMethod)
**Estimated effort**: Small

Steps:
1. Add `short`, `medium`, `long`, `full` cases
2. Each delegates to format logic with style preset

Tests:
- `@2024-12-25.short()` → `"12/25/24"`
- `@2024-12-25.medium()` → `"Dec 25, 2024"`
- `@2024-12-25.long()` → `"December 25, 2024"`
- `@2024-12-25.full()` → `"Wednesday, December 25, 2024"`

---

#### Task 3.4: Duration Style Methods
**Files**: `pkg/parsley/evaluator/methods.go` (evalDurationMethod)
**Estimated effort**: Small

Steps:
1. Add `short`, `medium`, `long` cases (no `full`)
2. Each delegates to format logic with style preset
3. `full()` should return helpful error

Tests:
- `@duration{hours: 2}.short()` → `"2h"`
- `@duration{hours: 2}.medium()` → `"2 hours"`
- `@duration{hours: 2, minutes: 30}.long()` → `"2 hours 30 minutes"`

---

#### Task 3.5: Unit Style Methods
**Files**: `pkg/parsley/evaluator/methods_unit.go`
**Estimated effort**: Small

Steps:
1. Add `short`, `medium`, `long`, `full` to registry with arity `"0-1"`
2. Each delegates to `unitFmt` with style preset

Tests:
- `#5m.short()` → `"5m"`
- `#5m.medium()` → `"5.00m"`
- `#5m.long()` → `"5 metres"`
- `#5m.full()` → `"5 metres (16.4 ft)"`

---

### Phase 4: Universal Serialization Methods

#### Task 4.1: Add Missing `repr()` Methods
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Small

Steps:
1. DateTime: Add `repr` case returning `@YYYY-MM-DDTHH:MM:SS` format
2. Duration: Add `repr` case returning `@duration{...}` format
3. Bool: Add `repr` case returning `true`/`false`
4. Null: Add `repr` case returning `null`
5. URL: Add `repr` case returning `@"url"` format
6. Path: Add `repr` case returning `@/path/to/file` format

Tests:
- `@2024-12-25.repr()` → `"@2024-12-25"`
- `@duration{hours: 2}.repr()` → `"@duration{hours: 2}"`
- `true.repr()` → `"true"`
- `null.repr()` → `"null"`

---

#### Task 4.2: Add Missing `toJSON()` Methods
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Small

Steps:
1. Bool: Add `toJSON` returning `"true"` or `"false"`
2. Null: Add `toJSON` returning `"null"`
3. URL: Add `toJSON` returning JSON-encoded URL string
4. Path: Add `toJSON` returning JSON-encoded path string
5. Unit: Add `toJSON` to `methods_unit.go` returning `{"value": n, "unit": "..."}`
6. Duration already has `toJSON` - verify format

Tests:
- `true.toJSON()` → `"true"`
- `null.toJSON()` → `"null"`
- `#5m.toJSON()` → `'{"value":5,"unit":"m"}'`

---

#### Task 4.3: Add `inspect()` to All Value Types
**Files**: `pkg/parsley/evaluator/methods.go`, `methods_numeric.go`
**Estimated effort**: Small

Steps:
1. Integer: Add `inspect` returning `{__type: "integer", value: n}`
2. Float: Add `inspect` returning `{__type: "float", value: n}`
3. String: Add `inspect` returning `{__type: "string", value: "...", length: n}`
4. Bool: Add `inspect` returning `{__type: "boolean", value: true/false}`
5. Null: Add `inspect` returning `{__type: "null"}`
6. URL: Already has `inspect` - verify `__type` key
7. Path: Already has `inspect` - verify `__type` key

Tests:
- `42.inspect()` → `{__type: "integer", value: 42}`
- `"hello".inspect()` → `{__type: "string", value: "hello", length: 5}`
- `true.inspect()` → `{__type: "boolean", value: true}`

---

#### Task 4.4: Add Missing `toBox()` Methods
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Small

Steps:
1. DateTime: Add `toBox` case using `BoxRenderer`
2. Duration: Already has `toBox` - verify
3. URL: Add `toBox` case
4. Path: Add `toBox` case
5. Unit: Add `toBox` to registry

Tests:
- Each type renders an ASCII box diagram

---

### Phase 5: Array Conjunction Formatting

#### Task 5.1: Array `.fmt()` for Conjunctions
**Files**: `pkg/parsley/evaluator/methods.go` (evalArrayMethod)
**Estimated effort**: Medium

Steps:
1. Update existing `format` case to also accept `fmt` alias
2. Verify conjunction formatting works:
   - `.fmt("and")` → `"A, B, and C"`
   - `.fmt("or")` → `"A, B, or C"`
3. Add locale support for conjunctions:
   - `de-DE`: "und" / "oder"
   - `fr-FR`: "et" / "ou"
   - `es-ES`: "y" / "o"
4. Handle edge cases:
   - Empty array: `""`
   - Single element: `"A"` (no conjunction)
   - Two elements: `"A and B"` (no comma)
5. Oxford comma rules by locale (English uses it, others don't)

Tests:
- `[].fmt("and")` → `""`
- `["Alice"].fmt("and")` → `"Alice"`
- `["Alice", "Bob"].fmt("and")` → `"Alice and Bob"`
- `["A", "B", "C"].fmt("and")` → `"A, B, and C"`
- `["A", "B", "C"].fmt("or")` → `"A, B, or C"`
- `["A", "B", "C"].fmt("and", "de-DE")` → `"A, B und C"`

---

### Phase 6: Collection Serialization (P1)

#### Task 6.1: Verify Collection Serialization Methods
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Small

Steps:
1. Verify Array has: `repr`, `toJSON`, `toBox`, `toMarkdown`, `toHTML`, `toCSV`
2. Verify Dictionary has: `repr`, `toJSON`, `toBox`, `toMarkdown`, `toHTML`
3. Add any missing methods
4. Ensure consistent behavior

Tests:
- Existing tests should cover most cases
- Add tests for edge cases if missing

---

### Phase 7: Documentation

#### Task 7.1: Update Reference Documentation
**Files**: `docs/parsley/reference.md`
**Estimated effort**: Medium

Steps:
1. Add `.fmt()` documentation for all types
2. Add style method documentation
3. Add serialization method documentation
4. Include style output table from design doc

---

#### Task 7.2: Update Cheatsheet
**Files**: `docs/parsley/CHEATSHEET.md`
**Estimated effort**: Small

Steps:
1. Add formatting examples highlighting differences from other languages
2. Note: `.fmt()` not `.format()` for brevity
3. Note: style methods as sugar

---

#### Task 7.3: Add Manual Pages
**Files**: `docs/parsley/manual/` (new files)
**Estimated effort**: Medium

Steps:
1. Create `fmt.md` - main formatting reference
2. Update type-specific manual pages with new methods

---

### Phase 8: Method Registry Updates

#### Task 8.1: Update Introspection Metadata
**Files**: `pkg/parsley/evaluator/introspect.go`
**Estimated effort**: Small

Steps:
1. Ensure all new methods appear in `BuiltinMetadata`
2. Verify arity strings are correct
3. Add descriptions for new methods

Tests:
- Introspection validation tests should pass

---

## Validation Checklist

- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] Documentation updated
- [ ] `work/BACKLOG.md` updated with deferrals (if any)
- [ ] Method registry introspection validates

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-02-24 | Phase 1: Infrastructure | ✅ Complete | Created `format_opts.go` with `FormatOpts` struct and parsing helpers |
| 2025-02-24 | Phase 2: `.fmt()` methods | ✅ Complete | Added to Number, Money, DateTime, Duration, Unit |
| 2025-02-24 | Phase 3: Style sugar | ✅ Complete | Added `.short()`, `.medium()`, `.long()`, `.full()` methods |
| 2025-02-24 | Phase 4: Serialization | ✅ Complete | Added `repr()`, `toJSON()`, `inspect()`, `toBox()` to missing types |
| 2025-02-24 | Phase 5: Array conjunctions | ✅ Complete | `.fmt("and")` and `.fmt("or")` work with locale support |
| 2025-02-24 | Tests | ✅ Complete | All evaluator and parsley tests pass |

## Deferred Items

Items to add to `work/BACKLOG.md` after implementation:

- **Extended locale support**: Only implemented top locales (en-US, de-DE, fr-FR, es-ES). Full i18n deferred.
- **Compound unit formatting**: `#5ft.fmt({compound: true})` → `"5' 3"`. Already tracked as #104.
- **printf-style format strings**: Explicitly not implementing per design decision.
- **Documentation updates**: Reference docs and manual pages need updating (Phase 7-8 deferred).

## Implementation Notes

### Breaking Changes
- DateTime `.format()` default changed from "long" to "medium" (per FEAT-121 spec)
- Number `.format()` now accepts integer as precision argument
- Unknown locales now fall back gracefully to en-US instead of erroring

### Backward Compatibility
- Unit `.format()` preserved original behavior (no decimal padding by default)
- `.format()` remains as alias for `.fmt()` on all types
- All existing method signatures continue to work

### Files Added
- `pkg/parsley/evaluator/format_opts.go` - FormatOpts struct and parsing helpers
- `pkg/parsley/evaluator/format_unified_test.go` - Comprehensive tests for new API

### Files Modified
- `pkg/parsley/evaluator/methods_numeric.go` - Integer/Float `.fmt()` and style methods
- `pkg/parsley/evaluator/methods_money.go` - Money `.fmt()` and style methods  
- `pkg/parsley/evaluator/methods_unit.go` - Unit `.fmt()` and style methods
- `pkg/parsley/evaluator/methods.go` - DateTime, Duration, Boolean, Null, Path, URL methods
- `pkg/parsley/evaluator/eval_helpers.go` - Fixed isDurationDict type checking
- `pkg/parsley/tests/locale_test.go` - Updated for new default behaviors
- `pkg/parsley/tests/methods_test.go` - Updated for new default behaviors

## Test Files

New test file: `pkg/parsley/evaluator/format_unified_test.go`

Existing test files to update:
- `pkg/parsley/evaluator/methods_numeric_test.go` (if exists) or add tests inline
- `pkg/parsley/evaluator/methods_money_test.go` (if exists)
- `pkg/parsley/evaluator/methods_unit_test.go` (if exists)
- `pkg/parsley/tests/` - Add Parsley-level integration tests

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Breaking existing `.format()` behavior | Low | High | Keep `.format()` as alias, test existing behavior |
| Locale data gaps | Medium | Medium | Fall back to en-US for unknown locales |
| Performance regression from options parsing | Low | Low | Keep parsing simple, no reflection |

## Dependencies

- **None blocking**: This feature is independent
- **Related**: FEAT-120 (print removal) is parallel work, no interaction

## Estimated Total Effort

| Phase | Effort |
|-------|--------|
| Phase 1: Infrastructure | 0.5 day |
| Phase 2: `.fmt()` methods | 1.5 days |
| Phase 3: Style sugar | 1 day |
| Phase 4: Serialization | 0.5 day |
| Phase 5: Array conjunctions | 0.5 day |
| Phase 6: Collection verify | 0.25 day |
| Phase 7: Documentation | 1 day |
| Phase 8: Registry updates | 0.25 day |
| **Total** | **~5.5 days** |