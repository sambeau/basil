# Standard Library 1.0 Action Plan

**Date:** 2026-03-14
**Updated:** 2026-03-15
**Status:** In progress
**Companion reports:** 
- `STDLIB-1.0-RELEASE-REVIEW.md` — Module-level assessment
- `STANDARD-PRELUDE-REVIEW.md` — Component-level assessment (comprehensive)
**Purpose:** Concrete, prioritised work items to bring the standard library and prelude to release quality

---

## Overview

The standard library is substantially ready for 1.0. The major cleanup (FEAT-129) was completed 2025-02-26, and FEAT-143 (Parsley correctness) is complete. The DataTable redesign (FEAT-144) and typed value formatting (FEAT-145) are now complete. This plan consolidates remaining gaps from both the STDLIB-1.0-RELEASE-REVIEW and the STANDARD-PRELUDE-REVIEW.

**Total estimated effort:** 15–25 hours across all tiers.
**Remaining effort:** ~5–8 hours (Tier 1: ~3–4h, Tier 2: ~3–4h).

---

## Progress Summary

| Tier | Total Items | Complete | Remaining |
|------|-------------|----------|-----------|
| Tier 1 (Must do) | 10 | 6 | 4 |
| Tier 2 (Should do) | 10 | 5 | 5 |

### Recently Completed

- ✅ **FEAT-144: DataTable Redesign** (2026-03-15) — Accepts `Table` object, auto-formatting, empty state, custom cell rendering, footer rows, column hiding, row headers, caption. All 5 phases complete with 22+ tests.
- ✅ **FEAT-145: Typed Value Formatting** (2026-03-15) — Money formatting in string coercion, `record.fieldProps()`, `<field/>` tag, `table.columnProps()`. Datetime/duration/unit deferred pending upstream `.medium()` improvements.
- ✅ **BUG-025: Short-circuit && and ||** (2026-03-15) — Logical operators now short-circuit correctly, enabling guard patterns like `x != null && x.length() > 0`. DataTable footer null-guard simplified.
- ✅ **FEAT-142: Meta Component and Page Restructure** (2026-03-15) — `Meta` component created, `Page` outputs OG/Twitter metadata
- ✅ **FEAT-143: Prelude Component Styling** (complete) — Parsley syntax correctness, Pico CSS adoption
- ✅ **New prelude components** — `Pagination`, `Toast`, `Toasts`, `Dialog`, `Details`, `Accordion`, `ErrorSummary` all exist

---

## Tier 1: Must Do Before 1.0

These items would create permanent API warts, active user confusion, or accessibility failures if shipped as-is.

**Estimated effort: 3–4 hours remaining.**

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

### 1.3 Rename `@std/mdDoc` → `@std/mddoc`

| | |
|---|---|
| **Priority** | 🔴 Must fix |
| **Effort** | 1–2 hours |
| **Risk** | Low (mechanical rename with alias) |
| **Source** | STDLIB-1.0-RELEASE-REVIEW, 1.0-SHIP-REVIEW §4b |

**Problem:** `@std/mdDoc` is the only camelCase module name. All others are lowercase.

**Work required:**

1. In `stdlib_table.go` → `getStdlibModules()`: Add `"mddoc": loadMdDocModule`, keep `"mdDoc"` as deprecated alias
2. In `module_meta.go`: Register `"mddoc": &mdDocModuleMeta` (currently missing entirely — see also 2.9)
3. Update `docs/parsley/manual/stdlib/mddoc.md` import examples
4. Add test for new import path
5. Commit: `feat(stdlib): rename @std/mdDoc to @std/mddoc with deprecation alias`

---

### 1.4 Slim `@std/schema` Documentation

| | |
|---|---|
| **Priority** | 🟡 Should fix |
| **Effort** | 1 hour |
| **Risk** | None |
| **Source** | STDLIB-1.0-RELEASE-REVIEW |

**Problem:** `schema.md` is 1,524 lines with deprecation banner but full documentation — confusing.

**Work required:** Replace with ~100 line deprecation notice + migration table + link to `@schema` DSL docs.

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

### 1.6 Prelude Component Smoke Test

| | |
|---|---|
| **Priority** | 🟡 Should fix |
| **Effort** | 1–2 hours |
| **Risk** | May uncover rendering bugs |
| **Source** | STDLIB-1.0-RELEASE-REVIEW, 1.0-SHIP-REVIEW §12 |

**Problem:** Components have been code-reviewed but not verified against actual HTML rendering. DataTable has thorough tests, but the other 32 components have no rendering tests.

**Work required:** Create test file using every component, verify valid HTML output, accessibility attributes present.

---

### 1.7 Add `dir` Prop to `Page` for RTL Support

| | |
|---|---|
| **Priority** | 🔴 Must fix |
| **Effort** | 5 minutes |
| **Risk** | None |
| **Source** | STANDARD-PRELUDE-REVIEW.md §8, Tier 1 #3 |

**Problem:** No way to set `dir="rtl"` for right-to-left language support.

**Work required:** Add `dir` prop to `Page`, pass to `<html dir={dir}>`.

---

### ✅ 1.8 Typed Value Formatting in `objectToString`/`objectToPrintString` (MOSTLY COMPLETE)

| | |
|---|---|
| **Status** | ✅ **Mostly complete** (2026-03-15) |
| **Source** | STANDARD-PRELUDE-REVIEW.md §9.5 (DECIDED) |
| **Design** | `work/design/DESIGN-typed-value-formatting.md` |
| **Plan** | `work/plans/PLAN-124-FEAT-145.md` |

**Work completed:**
- ✅ Money formatting via `moneyMedium()` in `objectToTemplateString` and `objectToPrintString`
- ✅ `record.fieldProps(name)` method
- ✅ `<field/>` tag with type-aware rendering
- ✅ `table.columnProps(name)` method
- ✅ Documentation updates

**Deliberately deferred (needs upstream `.medium()` improvements):**
- Datetime: `.medium()` doesn't respect datetime kinds (date-only, time-only, full)
- Duration: `.medium()` returns relative time ("tomorrow") — not suitable for coercion
- Unit: `.medium()` adds unwanted decimal places ("12.00m" vs "12m")

**Known gap:** `objectToString` in `stdlib_table.go` (used by `Table.toHTML()`, `.toCSV()`, etc.) does not have a `*Money` case — money values fall through to `obj.Inspect()` instead of getting `.medium()` formatting. This should be fixed.

**Related:** `work/specs/FEAT-145.md` (status needs updating to `complete`)

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

**Estimated effort: 3–4 hours remaining.**

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

### 2.6 Add Canadian Postal Code to `@std/valid`

| | |
|---|---|
| **Priority** | 🟢 Nice to have |
| **Effort** | 15 minutes |
| **Source** | STDLIB-1.0-RELEASE-REVIEW, STDLIB-AUDIT-2025 |

**Problem:** Docs imply CA support but only US and GB are implemented.

**Work required:** Add CA regex (`^[A-Za-z]\d[A-Za-z]\s?\d[A-Za-z]\d$`), test cases, update docs and error message.

---

### 2.7 Fix `@basil/api` Module Meta Description

| | |
|---|---|
| **Priority** | 🟢 Nice to have |
| **Effort** | 5 minutes |
| **Source** | STDLIB-1.0-RELEASE-REVIEW |

**Problem:** Description in `stdlib_api.go` says "HTTP client for API requests" but module is API route helpers.

---

### 2.8 Rename `dev.md` → `log.md`

| | |
|---|---|
| **Priority** | 🟢 Nice to have |
| **Effort** | 10 minutes |
| **Source** | STDLIB-1.0-RELEASE-REVIEW |

**Problem:** Filename doesn't match module name `@basil/log`.

---

### 2.9 Module Metadata Completeness

| | |
|---|---|
| **Priority** | 🟢 Nice to have |
| **Effort** | 30 minutes |
| **Source** | STDLIB-1.0-RELEASE-REVIEW |

**Problem:** `mdDoc` not registered in metadata maps, causing gaps in `pars describe`. (`html` is registered.)

**Note:** Can be combined with 1.3 (mdDoc rename) — register as `mddoc` when doing the rename.

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

Recommended order for remaining items, accounting for dependencies and risk:

```
1. Tier 1.7 — Add dir prop to Page (5 min, trivial)
2. Tier 1.3 + 2.9 — Rename @std/mdDoc → @std/mddoc + register metadata (combine)
3. Tier 1.4 — Slim @std/schema docs
4. Tier 1.6 — Prelude smoke test (may uncover issues)
5. Tier 2.6-2.8 — Quick polish items (CA postal code, api description, dev.md rename)
6. Tier 2.10 — Documentation accuracy verification (time-consuming, do last)
```

---

## Checklist

### Tier 1 (Must do)

- [x] **FEAT-142**: Meta component created, Page updated with OG/Twitter output
- [x] **FEAT-143**: Parsley correctness (++, spread, for loops, ranges)
- [x] New prelude components (Pagination, Toast, Dialog, Details, Accordion, ErrorSummary)
- [ ] `@std/mdDoc` renamed to `@std/mddoc` with deprecation alias
- [ ] `@std/schema` docs slimmed to deprecation notice
- [x] **FEAT-144 DataTable redesign**: accept Table, auto-format, empty state, render, footer, row headers
- [ ] Prelude component smoke test
- [ ] Add `dir` prop to `Page`
- [x] **FEAT-145 Typed value formatting**: money in string coercion, fieldProps, field tag, columnProps
- [x] **BUG-025**: Short-circuit `&&` and `||` operators

### Tier 2 (Should do)

- [x] Prop spreading consistency (mostly complete)
- [x] Class merging fixed (`+` not `++`)
- [x] SkipLink uses external CSS
- [x] SkipLink points to `#main`
- [x] Page `id` default fixed
- [ ] Canadian postal code in `@std/valid`
- [ ] `@basil/api` module description corrected
- [ ] `dev.md` renamed to `log.md`
- [ ] Module metadata completeness (`mdDoc` missing — combine with 1.3)
- [ ] Documentation accuracy verification

### Gate: All Tier 1 items complete → ready for 1.0 release

**Current status: 6/10 Tier 1 complete. 4 items remaining (~3–4 hours).**

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| ~~DataTable redesign introduces bugs~~ | — | — | ✅ Complete with 22+ tests |
| ~~Typed value formatting breaks output~~ | — | — | ✅ Complete, datetime/duration/unit deferred |
| Smoke test reveals broken components | Medium | Medium | Fix before release |
| `mdDoc` rename breaks user code | Low | Low | Deprecation alias preserves old name |

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