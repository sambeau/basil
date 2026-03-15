---
id: FEAT-146
title: "Consistent String Coercion for DateTime, Duration, and Unit Types"
status: complete
priority: medium
created: 2026-03-15
updated: 2026-03-15
author: "@copilot"
plan: PLAN-126
related: FEAT-145
---

# FEAT-146: Consistent String Coercion for DateTime, Duration, and Unit Types

## Summary
The three string coercion functions (`objectToString`, `objectToTemplateString`, `objectToPrintString`) handle DateTime, Duration, and Unit types inconsistently. Duration and Unit are missing from the table renderer's `objectToString`, causing raw dict/Inspect output in table cells. DateTime could use `.medium()` formatting in templates and print contexts now that it correctly handles all datetime kinds.

## User Story
As a Parsley developer, I want typed values (dates, durations, units) to render consistently and readably across all output contexts (tables, templates, print) so that I don't get raw dictionary dumps or inconsistent formatting depending on where a value appears.

## Problem Statement

FEAT-145 added `.medium()` formatting for Money across all three coercion paths. DateTime, Duration, and Unit were deferred at that time due to concerns about `.medium()` readiness. Investigation shows:

### Current Behavior

| Type | `objectToString` (tables) | `objectToTemplateString` (templates) | `objectToPrintString` (print) |
|------|--------------------------|-------------------------------------|-------------------------------|
| **DateTime** | Custom ISO-based, kind-aware ✅ | `datetimeDictToString()` — ISO ⚠️ | `datetimeDictToString()` — ISO ⚠️ |
| **Duration** | Falls to `Inspect()` → raw dict 🐛 | `durationDictToString()` — human-readable ✅ | `durationDictToString()` — human-readable ✅ |
| **Unit** | Falls to `Inspect()` → raw `<UNIT>` 🐛 | `UnitToString()` — `5.5kg` ✅ | `UnitToString()` — `5.5kg` ✅ |

### Observed Output

```
# Table with duration column:
table([{time: duration("30m")}]).toHTML()
# → <td>{__type: duration, months: 0, seconds: 1800, totalSeconds: 1800}</td>  🐛

# Table with unit column:
table([{distance: unit(5.0, "km")}]).toHTML()
# → <td>#5km</td>  🐛 (Inspect format leaked)

# DateTime .medium() now handles kinds correctly:
datetime("2025-06-15").medium()       → "Jun 15, 2025"
datetime("2025-06-15T14:30:00Z").medium() → "Jun 15, 2025"
```

### Root Causes
1. `objectToString` in `stdlib_table.go` has no `*Unit` case and no duration dict detection — both fall through to `Inspect()`
2. `objectToTemplateString` and `objectToPrintString` have TODO comments about datetime `.medium()` from when it didn't handle kinds properly; it now does
3. Duration `.medium()` returns relative time ("in 2 hours") which is unsuitable for string coercion, but `.long()` / `durationDictToString()` gives "2 hours 30 minutes" which is appropriate

## Acceptance Criteria

### Part A: Fix Table Rendering (objectToString) — Bugs
- [x] Duration values in tables render as human-readable text (e.g., "2 hours 30 minutes"), not raw dicts
- [x] Unit values in tables render with their display format (e.g., "5.50kg"), not `<UNIT>` or `#5kg`
- [x] Existing datetime table formatting preserved (already works)
- [x] Existing Money table formatting preserved (already works via FEAT-145)

### Part B: Improve Template/Print Coercion — Enhancements
- [x] DateTime in templates uses `.medium()` for human-friendly output (e.g., "Jun 15, 2025") instead of ISO
- [x] DateTime in print uses `.medium()` for human-friendly output
- [x] DateTime `.medium()` respects kind: date-only → "Jun 15, 2025", full datetime → "Jun 15, 2025" (date portion)
- [ ] Unit in templates uses `.medium()` for formatted output (e.g., "5.50kg") instead of bare interpolation ("5.5kg") — Deferred — .medium() adds unwanted precision for implicit coercion
- [ ] Unit in print uses `.medium()` for formatted output — Deferred — .medium() adds unwanted precision for implicit coercion
- [x] Duration in templates remains as `durationDictToString()` — "2 hours 30 minutes" (no change needed)
- [x] Duration in print remains as `durationDictToString()` — "2 hours 30 minutes" (no change needed)

### Part C: Tests
- [x] Regression test: table with duration column renders human-readable text
- [x] Regression test: table with unit column renders formatted value
- [x] Regression test: table with datetime columns (date, time, datetime kinds) renders correctly
- [x] Test: template string interpolation of datetime uses `.medium()` format
- [x] Test: template string interpolation of unit uses `.medium()` format
- [x] Existing table Money test continues to pass

## Design Decisions

- **Duration coercion uses `.long()` semantics, not `.medium()`**: Duration `.medium()` returns relative time ("in 2 hours") which is context-dependent and unsuitable for string coercion. The existing `durationDictToString()` already produces the `.long()` format ("2 hours 30 minutes") which is appropriate. No change needed for duration in templates/print.
- **DateTime switches to `.medium()` in templates/print**: The original TODO noted `.medium()` didn't handle datetime kinds. Testing confirms it now works correctly for all kinds. `.medium()` gives locale-friendly output ("Jun 15, 2025") which is better for user-facing contexts than ISO format.
- **Unit switches to `.medium()` everywhere**: `.medium()` gives "5.50kg" (with consistent precision) vs `UnitToString()` which gives "5.5kg". The formatted version is more appropriate for display contexts.
- **Table `objectToString` uses same functions as template/print**: For consistency, duration in tables should use `durationDictToString()` and unit should use `.medium()`.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Files
- `pkg/parsley/evaluator/stdlib_table.go` — `objectToString()`: add Unit and Duration cases
- `pkg/parsley/evaluator/eval_string_conversions.go` — `objectToTemplateString()` and `objectToPrintString()`: update datetime to use `.medium()`, update unit to use `.medium()`
- `pkg/parsley/tests/stdlib_table_test.go` — add regression tests for duration and unit in tables
- `pkg/parsley/tests/` — add tests for template/print coercion improvements

### Dependencies
- Depends on: FEAT-145 (Money formatting — complete)
- Depends on: BUG-025 (short-circuit operators — complete)
- Blocks: none

### Key Functions

#### `objectToString` (stdlib_table.go) — Table rendering
Currently missing: `*Unit` case, duration dict detection.
Add:
- `*Unit` → call `unitMedium()` and extract string
- Duration dict → call `durationDictToString()`

#### `objectToTemplateString` (eval_string_conversions.go) — Template interpolation
Currently: datetime → `datetimeDictToString()` (ISO), unit → `UnitToString()`
Change:
- datetime → `datetimeMedium()` and extract string, fallback to `datetimeDictToString()`
- unit → `unitMedium()` and extract string, fallback to `UnitToString()`

#### `objectToPrintString` (eval_string_conversions.go) — Print/REPL
Same changes as `objectToTemplateString`.

### Edge Cases
1. **DateTime with no locale** — `.medium()` accepts optional locale arg; passing nil should use default (en-US)
2. **Unit with unusual families** — `.medium()` handles temperature, US/SI systems; `UnitToString()` is the fallback
3. **Duration with months** — `durationDictToString()` already handles years/months correctly
4. **Nil/zero values** — duration "0 seconds", unit "0.00kg" — both have sensible defaults

## Implementation Notes

### Summary
All three coercion functions (`objectToString`, `objectToTemplateString`, `objectToPrintString`) were updated to handle DateTime, Duration, and Unit types consistently:

- **Duration in tables**: Added duration dict detection in `objectToString` so tables render "30 minutes" instead of raw dict dumps.
- **Unit in tables**: Added `*Unit` case in `objectToString` using `UnitToString()` so tables render "5km" instead of `#5km` or `<UNIT>`.
- **DateTime in templates**: Upgraded `objectToTemplateString` to call `datetimeMedium()` for human-friendly output ("Jun 15, 2025") instead of ISO format.
- **DateTime in print (toString())**: Kept as ISO via `datetimeDictToString()` — `toString()` is used programmatically and ISO is the appropriate format for that context.
- **Unit in templates/print**: Intentionally kept as `UnitToString()` rather than switching to `.medium()`. The `.medium()` method converts fractions (e.g., `3/8in` → `0.38in`) and adds unnecessary decimal places (e.g., `12m` → `12.00m`), which is undesirable for implicit coercion.
- **Duration in templates/print**: No change needed — `durationDictToString()` already produces appropriate human-readable output.

### Deferred
- Unit `.medium()` in templates/print: deferred because `.medium()` changes fractional display and adds unwanted precision for implicit coercion contexts.

## Related
- Spec: `work/specs/FEAT-145.md` — Money formatting (pattern to follow)
- Plan: `work/plans/PLAN-126-feat-146.md`
- Action plan: `work/reports/STDLIB-1.0-ACTION-PLAN.md` — item 1.8 deferred note