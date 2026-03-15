---
id: PLAN-126
feature: FEAT-146
title: "Implementation Plan for Consistent String Coercion"
status: complete
created: 2026-03-15
---

# Implementation Plan: FEAT-146

## Overview
Fix Duration and Unit rendering in tables (bugs) and upgrade DateTime/Unit formatting in templates and print to use `.medium()` (enhancements). Three coercion functions need alignment: `objectToString`, `objectToTemplateString`, `objectToPrintString`.

## Prerequisites
- [x] FEAT-145 complete (Money formatting pattern established)
- [x] DateTime `.medium()` handles all kinds correctly (verified via `pars -e`)
- [x] `durationDictToString()` already produces appropriate human-readable output

## Tasks

### Task 1: Fix Duration and Unit in Table `objectToString`
**Files**: `pkg/parsley/evaluator/stdlib_table.go`
**Estimated effort**: Small

These are bugs — Duration shows raw dict, Unit shows Inspect format in table cells.

Steps:
1. Add a `*Unit` case to `objectToString` that calls `unitMedium(o, nil, nil)` and extracts the string value, falling back to `UnitToString(o)`
2. Add duration dict detection inside the existing `*Dictionary` case (before the `__type` check or alongside it): call `isDurationDict(o)` → `durationDictToString(o)`
3. Verify existing datetime and Money cases are unchanged

Tests:
- Table with duration column renders "30 minutes" not raw dict
- Table with unit column renders "5.00km" not `#5km` or `<UNIT>`
- Existing `TestTableMoneyMediumFormatting` still passes
- Table with mixed typed columns (money, datetime, duration, unit) in one row

---

### Task 2: Upgrade DateTime in Template and Print Coercion
**Files**: `pkg/parsley/evaluator/eval_string_conversions.go`
**Estimated effort**: Small

Replace `datetimeDictToString(obj)` with `.medium()` call in both `objectToTemplateString` and `objectToPrintString`.

Steps:
1. In `objectToTemplateString`, replace the datetime branch:
   - Call `datetimeMedium(obj, nil, nil)` 
   - If result is `*String`, return its value
   - Fallback to `datetimeDictToString(obj)` if `.medium()` returns an error
2. Apply the same change in `objectToPrintString`
3. Remove the TODO comments about `.medium()` not handling kinds

Tests:
- Template interpolation of date-only datetime → "Jun 15, 2025" style output
- Template interpolation of full datetime → medium formatted date
- Print coercion matches template coercion for datetime values

---

### Task 3: Upgrade Unit in Template and Print Coercion
**Files**: `pkg/parsley/evaluator/eval_string_conversions.go`
**Estimated effort**: Small

Replace `UnitToString(obj)` with `.medium()` call in both functions.

Steps:
1. In `objectToTemplateString`, replace the `*Unit` case:
   - Call `unitMedium(obj, nil, nil)`
   - If result is `*String`, return its value
   - Fallback to `UnitToString(obj)`
2. Apply the same change in `objectToPrintString`
3. Remove the backward-compatibility comments (`.medium()` is the new standard)

Tests:
- Template interpolation of unit → "5.50kg" (with precision)
- Print coercion of unit → "5.50kg"
- Temperature units format correctly

---

### Task 4: Regression Tests
**Files**: `pkg/parsley/tests/stdlib_table_test.go`, `pkg/parsley/tests/` (new or existing test file)
**Estimated effort**: Small

Steps:
1. Add `TestTableDurationFormatting` — table with duration column
2. Add `TestTableUnitFormatting` — table with unit column  
3. Add `TestTableMixedTypedColumns` — row with money, datetime, duration, unit
4. Add tests for template string coercion of datetime and unit (if not covered by existing tests)
5. Verify all existing tests pass

---

## Execution Order

1. **Task 1** first — fixes the bugs (Duration/Unit in tables)
2. **Tasks 2 & 3** together — enhancement pass on template/print
3. **Task 4** — tests (some written alongside Tasks 1-3, consolidated here)

Total estimated effort: ~1 hour

## Validation Checklist
- [x] All tests pass: `go test ./...`
- [x] Build succeeds: `make dev`
- [x] Table: duration column renders human-readable text
- [x] Table: unit column renders formatted value
- [x] Template: datetime interpolates with `.medium()` format
- [x] Template: unit in templates/print kept as `UnitToString()` — deferred (see below)
- [x] Duration coercion unchanged in templates/print (already correct)
- [x] Existing Money formatting unaffected
- [x] `make bench-compare` shows no regression
- [x] Action plan updated

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-03-15 | Task 1: Fix Duration and Unit in Table `objectToString` | ✅ Complete | Duration renders human-readable, Unit renders formatted value |
| 2026-03-15 | Task 2: Upgrade DateTime in Template and Print Coercion | ✅ Complete | `.medium()` used in templates; `toString()` kept as ISO (programmatic use) |
| 2026-03-15 | Task 3: Upgrade Unit in Template and Print Coercion | ⏸️ Deferred | `.medium()` converts fractions and adds unwanted precision |
| 2026-03-15 | Task 4: Regression Tests | ✅ Complete | Table duration/unit/datetime tests, template coercion tests |

## Deferred Items
- Duration `.medium()` returns relative time — if a future spec changes this to absolute duration, reconsider using it for coercion
- DateTime `.medium()` currently returns date portion only for full datetimes — may want a "datetime medium" that includes time (e.g., "Jun 15, 2025 2:30 PM") in a future spec
- Unit `.medium()` in templates/print deferred because it changes fractional display (e.g., 3/8in → 0.38in) and adds unnecessary decimal places (e.g., 12m → 12.00m). Kept as `UnitToString()` which preserves fractional and minimal formatting.