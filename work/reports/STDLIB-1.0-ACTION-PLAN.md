# Standard Library 1.0 Action Plan

**Date:** 2026-03-14
**Updated:** 2026-03-15
**Status:** Tier 1 complete 🎉
**Companion reports:** 
- `STDLIB-1.0-RELEASE-REVIEW.md` — Module-level assessment
- `STANDARD-PRELUDE-REVIEW.md` — Component-level assessment (comprehensive)
**Purpose:** Concrete, prioritised work items to bring the standard library and prelude to release quality

---

## Overview

The standard library is substantially ready for 1.0. The major cleanup (FEAT-129) was completed 2025-02-26, and FEAT-143 (Parsley correctness) is complete. The DataTable redesign (FEAT-144) and typed value formatting (FEAT-145) are now complete. This plan consolidates remaining gaps from both the STDLIB-1.0-RELEASE-REVIEW and the STANDARD-PRELUDE-REVIEW.

**Total estimated effort:** 15–25 hours across all tiers.
**Remaining effort:** ~3–4 hours (Tier 2 only).

---

## Progress Summary

| Tier | Total Items | Complete | Remaining |
|------|-------------|----------|-----------|
| Tier 1 (Must do) | 10 | **10** | **0** ✅ |
| Tier 2 (Should do) | 10 | 5 | 5 |

### Recently Completed

- ✅ **Tier 1.3: Rename @std/mdDoc → @std/mddoc** (2026-03-15) — Canonical name `mddoc`, deprecated alias `mdDoc` preserved. Module metadata registered (was missing). Tests and docs updated.
- ✅ **Tier 1.4: Slim @std/schema docs** (2026-03-15) — Replaced 1,524-line doc with 104-line deprecation notice + migration table.
- ✅ **Tier 1.6: Prelude component smoke test** (2026-03-15) — 40 subtests covering all 33 components. 24 render successfully; 9 have known bugs documented with `wantErr` assertions (see below).
- ✅ **Tier 1.7: Add `dir` prop to Page** (2026-03-15) — RTL support via `<html dir={dir}>`.
- ✅ **Money formatting in Table rendering** (2026-03-15) — `Table.toHTML()`, `.toCSV()`, `.toMarkdown()`, `.toBox()` now use `.medium()` for money values.
- ✅ **FEAT-144: DataTable Redesign** (2026-03-15) — Accepts `Table` object, auto-formatting, empty state, custom cell rendering, footer rows, column hiding, row headers, caption. All 5 phases complete with 22+ tests.
- ✅ **FEAT-145: Typed Value Formatting** (2026-03-15) — Money formatting in string coercion, `record.fieldProps()`, `<field/>` tag, `table.columnProps()`. Datetime/duration/unit deferred pending upstream `.medium()` improvements.
- ✅ **BUG-025: Short-circuit && and ||** (2026-03-15) — Logical operators now short-circuit correctly, enabling guard patterns like `x != null && x.length() > 0`. DataTable footer null-guard simplified.
- ✅ **FEAT-142: Meta Component and Page Restructure** (2026-03-15) — `Meta` component created, `Page` outputs OG/Twitter metadata
- ✅ **FEAT-143: Prelude Component Styling** (complete) — Parsley syntax correctness, Pico CSS adoption
- ✅ **New prelude components** — `Pagination`, `Toast`, `Toasts`, `Dialog`, `Details`, `Accordion`, `ErrorSummary` all exist

---

## Tier 1: Must Do Before 1.0

These items would create permanent API warts, active user confusion, or accessibility failures if shipped as-is.

**All Tier 1 items complete.**

### ✅ 1.0 FEAT-142: Meta Component and Page Restructure (COMPLETE)

| | |
|---|---|
| **Status** | ✅ **Complete** (2026-03-15) |
| **Source** | STANDARD-PRELUDE-REVIEW.md §7.4 |

**Work completed:** Created `Meta` component, updated `Page` to output OG/Twitter metadata, `Head` kept as deprecated alias.

**Related:** `work/specs/FEAT-142.md`, `work/design/DESIGN-prelude-meta-component.md`

---

### ✅ 1.1 FEAT-143: Prelude Component Parsley Correctness (COMPLETE)

| | |
|---|---|
| **Status** | ✅ **Complete** |
| **Source** | STANDARD-PRELUDE-REVIEW.md §16 (Appendix D) |

**Work completed:** Fixed `++` operator misuse, tag spread syntax, `for` loop variable ordering, pagination range logic.

**Related:** `work/specs/FEAT-143-prelude-component-styling.md`

---

### ✅ 1.2 New Prelude Components (COMPLETE)

| | |
|---|---|
| **Status** | ✅ **Complete** |
| **Source** | STANDARD-PRELUDE-REVIEW.md §14 |

**Components created:** `Pagination`, `Toast`, `Toasts`, `Dialog`, `Details`, `Accordion`, `ErrorSummary`

---

### ✅ 1.3 Rename `@std/mdDoc` → `@std/mddoc` (COMPLETE)

| | |
|---|---|
| **Status** | ✅ **Complete** (2026-03-15) |
| **Source** | STDLIB-1.0-RELEASE-REVIEW, 1.0-SHIP-REVIEW §4b |

**Work completed:**
- `"mddoc"` registered as canonical module name in `stdlib_table.go`
- `"mdDoc"` preserved as deprecated alias
- Module metadata registered in `module_meta.go` (was previously missing entirely)
- Documentation updated in `docs/parsley/manual/stdlib/mddoc.md`
- Import tests added for both `@std/mddoc` and deprecated `@std/mdDoc`

---

### ✅ 1.4 Slim `@std/schema` Documentation (COMPLETE)

| | |
|---|---|
| **Status** | ✅ **Complete** (2026-03-15) |
| **Source** | STDLIB-1.0-RELEASE-REVIEW |

**Work completed:** Replaced 1,524-line doc with 104-line deprecation notice including before/after example, migration tables (type factories and operations), new DSL features summary, and links to `@schema` DSL reference.

---

### ✅ 1.5 DataTable Redesign (COMPLETE)

| | |
|---|---|
| **Status** | ✅ **Complete** (2026-03-15) |
| **Source** | STANDARD-PRELUDE-REVIEW.md §3, §9, §13 |
| **Design** | `work/design/DESIGN-datatable-redesign.md` |
| **Plan** | `work/plans/PLAN-125-FEAT-144.md` |

**Work completed (all 5 phases):**
- Accepts `data` prop (Table object) — auto-derives columns, rows, keys
- Empty state with customizable message (`empty` prop)
- Type-aware formatting: money, datetime, duration, unit, boolean, null (em dash)
- Custom cell rendering via `render` prop (dict of column → function)
- Column hiding (`hide`), alignment (`align`), header overrides (`headers`)
- Footer rows (`footer` prop → `<tfoot>`)
- Row headers with `<th scope="row">` (`rowHeader` prop)
- Caption support
- Prop spreading (`id`, `class`, `...attrs`)
- Removed unused `sortable` prop
- Backward compatible: legacy `columns`/`rows`/`keys` API still works
- 22+ test cases in `datatable_test.go`
- Footer null-guard simplified using BUG-025 short-circuit fix

**Related:** `work/specs/FEAT-144.md` (status needs updating to `complete`)

---

### ✅ 1.6 Prelude Component Smoke Test (COMPLETE)

| | |
|---|---|
| **Status** | ✅ **Complete** (2026-03-15) |
| **Source** | STDLIB-1.0-RELEASE-REVIEW, 1.0-SHIP-REVIEW §12 |

**Work completed:** Created `prelude_smoke_test.go` with 40 subtests covering all 33 components. 24 components render successfully with output validation. 9 subtests document known component bugs via `wantErr` assertions.

**Bugs discovered (candidates for follow-up):**

| Components | Bug | Fix |
|-----------|-----|-----|
| Checkbox, CheckboxGroup, RadioGroup | Use `.length` (property) on arrays — Parsley requires `.length()` (method) | Change `.length` → `.length()` in each component |
| LocalTime, RelativeTime, Time, TimeRange | Call `.format("iso")` — `"iso"` is not a valid format style | Use `.iso` property or `.toJSON()` instead |
| Pagination | Calls `.floor()` on integer division result — `.floor()` only defined for floats | Remove `.floor()` call or cast to float first |

---

### ✅ 1.7 Add `dir` Prop to `Page` for RTL Support (COMPLETE)

| | |
|---|---|
| **Status** | ✅ **Complete** (2026-03-15) |
| **Source** | STANDARD-PRELUDE-REVIEW.md §8, Tier 1 #3 |

**Work completed:** Added `dir` prop to `Page`, passed to `<html dir={dir}>`. When omitted, no `dir` attribute is rendered (browser default).

---

### ✅ 1.8 Typed Value Formatting in `objectToString`/`objectToPrintString` (COMPLETE)

| | |
|---|---|
| **Status** | ✅ **Complete** (2026-03-15) |
| **Source** | STANDARD-PRELUDE-REVIEW.md §9.5 (DECIDED) |
| **Design** | `work/design/DESIGN-typed-value-formatting.md` |
| **Plan** | `work/plans/PLAN-124-FEAT-145.md`, `work/plans/PLAN-126-feat-146.md` |

**Work completed:**
- ✅ Money formatting via `moneyMedium()` in `objectToTemplateString` and `objectToPrintString`
- ✅ `record.fieldProps(name)` method
- ✅ `<field/>` tag with type-aware rendering
- ✅ `table.columnProps(name)` method
- ✅ Documentation updates
- ✅ Duration and Unit in table `objectToString` fixed (FEAT-146)
- ✅ DateTime in templates uses `.medium()` for human-friendly output (FEAT-146)
- ✅ DateTime in `objectToPrintString` (toString()) kept as ISO — used programmatically (FEAT-146)
- ✅ Unit in templates uses `UnitToString()` — `.medium()` deferred (adds unwanted precision) (FEAT-146)
- ✅ Duration coercion unchanged in templates/print — already correct (FEAT-146)

**Deferred:**
- Unit `.medium()` in templates/print — deferred because `.medium()` converts fractions (3/8in → 0.38in) and adds unnecessary decimal places (12m → 12.00m)

**Related:** `work/specs/FEAT-145.md`, `work/specs/FEAT-146.md`

---

### ✅ 1.9 BUG-025: Short-Circuit Logical Operators (COMPLETE)

| | |
|---|---|
| **Status** | ✅ **Complete** (2026-03-15) |
| **Source** | BUG-025, FEAT-144 workaround |

**Work completed:** `&&`/`&`/`and` and `||`/`|`/`or` now short-circuit correctly. Guard patterns like `x != null && x.length() > 0` work as expected. Array/dictionary intersection and array union are unaffected (both operands still evaluated for collection types). 25 regression tests added. DataTable footer null-guard simplified.

**Related:** `work/bugs/BUG-025.md`

---

## Tier 2: Should Do Before 1.0

These items improve polish and correctness but don't create permanent problems.

**Estimated effort: ~3–4 hours remaining.**

### ✅ 2.1 Prop Spreading Consistency (MOSTLY COMPLETE)

| | |
|---|---|
| **Status** | ✅ Mostly complete |
| **Source** | STANDARD-PRELUDE-REVIEW.md §7.1, Tier 1 #4 |

10 of 33 components have prop spreading. The new components (Pagination, Toast, Dialog, etc.) use it.

**Remaining:** Verify all form components spread rest props to underlying HTML elements.

---

### ✅ 2.2 Class Merging Fixed (COMPLETE)

| | |
|---|---|
| **Status** | ✅ Complete |
| **Source** | STANDARD-PRELUDE-REVIEW.md §7.2, Tier 1 #15 |

**Work completed:** All components now use `+` for string concatenation instead of `++`.

---

### ✅ 2.3 SkipLink Uses External CSS (COMPLETE)

| | |
|---|---|
| **Status** | ✅ Complete |
| **Source** | STANDARD-PRELUDE-REVIEW.md §4, Tier 1 #7 |

**Work completed:** SkipLink no longer has inline `<style>`, uses `basil-supplement.css`.

---

### ✅ 2.4 SkipLink Points to `#main` (COMPLETE)

| | |
|---|---|
| **Status** | ✅ Complete |
| **Source** | STANDARD-PRELUDE-REVIEW.md §5, Tier 1 #2 |

**Verified:** SkipLink defaults to `href="#main"`.

---

### ✅ 2.5 Page `id` Default Fixed (COMPLETE)

| | |
|---|---|
| **Status** | ✅ Complete |
| **Source** | STANDARD-PRELUDE-REVIEW.md §13, Tier 1 #1 |

**Verified:** Page no longer defaults `id="main"` on body. Documentation instructs users to put `id="main"` on their `<main>` element.

---

### ✅ 2.6 Add Canadian Postal Code to `@std/valid` (COMPLETE)

| | |
|---|---|
| **Priority** | 🟢 Nice to have |
| **Effort** | 15 minutes |
| **Source** | STDLIB-1.0-RELEASE-REVIEW, STDLIB-AUDIT-2025 |

**Problem:** Docs imply CA support but only US and GB are implemented.

**Work required:** Add CA regex (`^[A-Za-z]\d[A-Za-z]\s?\d[A-Za-z]\d$`), test cases, update docs and error message.

**Resolution:** Added `caPostalRegex` to `stdlib_valid.go`, CA case in `validPostalCode` switch, updated error message to include CA. Added 8 test cases. Updated `valid.md` and `reference.md` docs.

---

### ✅ 2.7 Fix `@basil/api` Module Meta Description (COMPLETE)

| | |
|---|---|
| **Priority** | 🟢 Nice to have |
| **Effort** | 5 minutes |
| **Source** | STDLIB-1.0-RELEASE-REVIEW |

**Problem:** Description in `stdlib_api.go` says "HTTP client for API requests" but module is API route helpers.

**Resolution:** Updated description to "HTTP API route helpers" in `stdlib_api.go` and `api-reference.md`.

---

### ✅ 2.8 Rename `dev.md` → `log.md` (COMPLETE)

| | |
|---|---|
| **Priority** | 🟢 Nice to have |
| **Effort** | 10 minutes |
| **Source** | STDLIB-1.0-RELEASE-REVIEW |

**Problem:** Filename doesn't match module name `@basil/log`.

**Resolution:** Renamed `docs/parsley/manual/stdlib/dev.md` → `log.md`. Updated two links in `manual/index.md`.

---

### 2.9 Module Metadata Completeness

| | |
|---|---|
| **Priority** | 🟢 Nice to have |
| **Effort** | 30 minutes |
| **Source** | STDLIB-1.0-RELEASE-REVIEW |

**Problem:** `mdDoc` not registered in metadata maps, causing gaps in `pars describe`. (`html` is registered.)

**Note:** ✅ Resolved as part of 1.3 (mdDoc rename) — `mddoc` and `mdDoc` both registered in metadata maps.

---

### 2.10 Verify Documentation Accuracy Against `pars describe`

| | |
|---|---|
| **Priority** | 🟢 Nice to have |
| **Effort** | 2–3 hours |
| **Source** | STDLIB-1.0-RELEASE-REVIEW |

**Work required:** Compare manual pages against `pars describe` output for all modules.

---

## Tier 3: Post-1.0 (1.1, 1.2)

### From STANDARD-PRELUDE-REVIEW Tier 2/3

| # | Item | Effort | Source | Notes |
|---|------|--------|--------|-------|
| 26 | DataTable per-column `format` override prop | 1–2h | §3 | |
| ~~27~~ | ~~DataTable custom cell render functions (`render` prop)~~ | — | §3 | ✅ Delivered in FEAT-144 |
| ~~28~~ | ~~DataTable footer/summary row support~~ | — | §3 | ✅ Delivered in FEAT-144 |
| 28b | DataTable server-side sorting helpers (clickable headers, URL params, direction indicators) | 2–3h | §3, user example | |
| 29 | `FileField` component | 1h | §14 | |
| 30 | `LocalNumber` / `LocalCurrency` components | 2h | §8 | |
| ~~31~~ | ~~`record.fieldProps()` method~~ | — | §9 | ✅ Delivered in FEAT-145 |
| 32 | Time components accept `datetime` objects directly | 1h | §13 | |
| 33 | `record.toHTML()` with schema-aware formatting | 1–2h | §9 | |
| 34 | Auto-detect external links in `Link` component | 15m | §13 | |
| 35 | `request.language` server feature | 2h | §8 | |

### From STDLIB Review

| Item | Effort |
|------|--------|
| Additional postal codes (DE, FR, AU, JP) | 1h |
| Date/number formatting locales (Backlog #17) | 4–8h |
| `truncateWords(n, suffix?)` string method | 1h |
| HTTP client module | 8–16h |

### From Recent Work

| Item | Effort | Notes |
|------|--------|-------|
| Datetime `.medium()` respects kind (date-only, time-only) | 2–3h | Unblocks datetime string coercion (FEAT-145 deferred) |
| Duration `.medium()` returns absolute format | 1–2h | Unblocks duration string coercion (FEAT-145 deferred) |
| Unit `.medium()` intelligent decimal handling | 1h | Unblocks unit string coercion (FEAT-145 deferred) |
| Money case in `objectToString` (stdlib_table.go) | 15m | Table rendering methods use `.medium()` for money |

---

## Tier 4: Defer to 2.0 or Never

| Item | Reason to Defer |
|------|-----------------|
| Remove deprecated module aliases | Keep warnings in 1.x, hard-error in 2.0 |
| WYSIWYG editor | Too complex, external dependency |
| CommandPalette | Niche, high complexity |
| SortableList | External JS dependency |
| Full i18n framework | Massive scope |

---

## Execution Order

**Tier 1 is complete.** Remaining Tier 2 items in recommended order:

```
1. Tier 2.6 — Canadian postal code in @std/valid (15 min)
2. Tier 2.7 — Fix @basil/api module description (5 min)
3. Tier 2.8 — Rename dev.md → log.md (10 min)
4. Tier 2.10 — Documentation accuracy verification (2-3 hours, do last)
```

Note: Tier 2.9 (module metadata) was resolved as part of Tier 1.3.

### Follow-Up: Component Bug Fixes

The smoke test (1.6) uncovered 9 component bugs. These should be fixed before 1.0:

```
1. Fix .length → .length() in Checkbox, CheckboxGroup, RadioGroup
2. Fix .format("iso") → .iso in LocalTime, RelativeTime, Time, TimeRange
3. Fix .floor() on integer in Pagination
```

---

## Checklist

### Tier 1 (Must do)

- [x] **FEAT-142**: Meta component created, Page updated with OG/Twitter output
- [x] **FEAT-143**: Parsley correctness (++, spread, for loops, ranges)
- [x] New prelude components (Pagination, Toast, Dialog, Details, Accordion, ErrorSummary)
- [x] `@std/mdDoc` renamed to `@std/mddoc` with deprecation alias
- [x] `@std/schema` docs slimmed to deprecation notice
- [x] **FEAT-144 DataTable redesign**: accept Table, auto-format, empty state, render, footer, row headers
- [x] Prelude component smoke test (uncovered 9 component bugs — see follow-up)
- [x] Add `dir` prop to `Page`
- [x] **FEAT-145 Typed value formatting**: money in string coercion, fieldProps, field tag, columnProps
- [x] **BUG-025**: Short-circuit `&&` and `||` operators

### Tier 2 (Should do)

- [x] Prop spreading consistency (mostly complete)
- [x] Class merging fixed (`+` not `++`)
- [x] SkipLink uses external CSS
- [x] SkipLink points to `#main`
- [x] Page `id` default fixed
- [x] Canadian postal code in `@std/valid`
- [x] `@basil/api` module description corrected
- [x] `dev.md` renamed to `log.md`
- [x] Module metadata completeness (resolved with 1.3 — `mddoc` registered)
- [ ] Documentation accuracy verification

### Gate: All Tier 1 items complete → ready for 1.0 release ✅

**Status: 10/10 Tier 1 complete. Gate passed.**

**Note:** The smoke test uncovered 9 component bugs (`.length` vs `.length()`, `.format("iso")`, `.floor()` on integer). These should be fixed before release — see Follow-Up section in Execution Order.

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| ~~DataTable redesign introduces bugs~~ | — | — | ✅ Complete with 22+ tests |
| ~~Typed value formatting breaks output~~ | — | — | ✅ Complete (FEAT-146) |
| ~~Smoke test reveals broken components~~ | — | — | ✅ Found 9 bugs in 3 categories (fixable) |
| ~~`mdDoc` rename breaks user code~~ | — | — | ✅ Deprecated alias preserves old name |

---

## Related Documents

| Document | Relevance |
|----------|-----------|
| `work/reports/STDLIB-1.0-RELEASE-REVIEW.md` | Module-level analysis |
| `work/reports/STANDARD-PRELUDE-REVIEW.md` | Component-level analysis (primary source) |
| `work/design/DESIGN-datatable-redesign.md` | DataTable redesign (Complete) |
| `work/design/DESIGN-typed-value-formatting.md` | Typed value formatting (Complete) |
| `work/specs/FEAT-142.md` | Meta component spec (Complete) |
| `work/specs/FEAT-143-prelude-component-styling.md` | Parsley correctness (Complete) |
| `work/specs/FEAT-144.md` | DataTable redesign spec (Complete) |
| `work/plans/PLAN-125-FEAT-144.md` | DataTable implementation plan (Complete) |
| `work/specs/FEAT-145.md` | Typed value formatting spec (Complete) |
| `work/plans/PLAN-124-FEAT-145.md` | Typed value formatting plan (Complete) |
| `work/bugs/BUG-025.md` | Short-circuit logical operators (Fixed) |
| `work/specs/FEAT-129.md` | Stdlib cleanup (Complete) |
| `work/BACKLOG.md` | #17 (locales), #5 (email), #12 (form targets) |