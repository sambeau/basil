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

The standard library is substantially ready for 1.0. The major cleanup (FEAT-129) was completed 2025-02-26, and FEAT-143 (Parsley correctness) is complete. This plan consolidates remaining gaps from both the STDLIB-1.0-RELEASE-REVIEW and the STANDARD-PRELUDE-REVIEW.

**Total estimated effort:** 15–25 hours across all tiers.

---

## Progress Summary

| Tier | Total Items | Complete | Remaining |
|------|-------------|----------|-----------|
| Tier 1 (Must do) | 8 | 3 | 5 |
| Tier 2 (Should do) | 10 | 5 | 5 |

### Recently Completed

- ✅ **FEAT-142: Meta Component and Page Restructure** (2026-03-15) — `Meta` component created, `Page` outputs OG/Twitter metadata
- ✅ **FEAT-143: Prelude Component Styling** (complete) — Parsley syntax correctness, Pico CSS adoption
- ✅ **New prelude components** — `Pagination`, `Toast`, `Toasts`, `Dialog`, `Details`, `Accordion`, `ErrorSummary` all exist

---

## Tier 1: Must Do Before 1.0

These items would create permanent API warts, active user confusion, or accessibility failures if shipped as-is.

**Estimated effort: 8–12 hours total.**

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
2. In `module_meta.go`: Register `"mddoc": &mdDocModuleMeta`
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

### 1.5 DataTable Redesign

| | |
|---|---|
| **Priority** | 🔴 Must fix |
| **Effort** | 3–4 hours |
| **Risk** | Low (backward compatible) |
| **Source** | STANDARD-PRELUDE-REVIEW.md §3, §9, §13 |
| **Design** | `work/design/DESIGN-datatable-redesign.md` |

**Problem:** `DataTable` predates the Parsley `Table` type. Users must manually decompose `Table` into parallel arrays. No type-aware formatting, empty state, or custom cell rendering. The `sortable` prop does nothing.

**Work required:**

1. Accept `data` prop (Table object) — auto-derive columns, rows, keys
2. Add empty state with customizable message
3. Auto-format typed values (`money`, `datetime`, `unit` via `.medium()`)
4. Remove unused `sortable` prop
5. Resolve 5 pending design decisions (see design doc §6)
6. Backward compatible: existing `columns`/`rows`/`keys` API continues to work

**Decisions needed:**
- Row header column behavior (always first vs configurable)
- Boolean formatting (Yes/No vs ✓/✗)
- Null value display (em dash vs empty)
- Title case conversion for column headers
- Confirm removal of `sortable` prop

---

### 1.6 Prelude Component Smoke Test

| | |
|---|---|
| **Priority** | 🟡 Should fix |
| **Effort** | 1–2 hours |
| **Risk** | May uncover rendering bugs |
| **Source** | STDLIB-1.0-RELEASE-REVIEW, 1.0-SHIP-REVIEW §12 |

**Problem:** Components have been code-reviewed but not verified against actual HTML rendering.

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

### 1.8 Typed Value Formatting in `objectToString`/`objectToPrintString`

| | |
|---|---|
| **Priority** | 🔴 Must fix |
| **Effort** | 30 min |
| **Risk** | Breaking change (acceptable pre-1.0) |
| **Source** | STANDARD-PRELUDE-REVIEW.md §9.5 (DECIDED) |
| **Design** | `work/design/DESIGN-typed-value-formatting.md` |

**Problem:** `money`, `datetime`, `unit`, `duration` values output as raw structures instead of human-readable text.

**Work required:** Change `objectToString()` and `objectToPrintString()` to call `.medium()` on these types.

---

## Tier 2: Should Do Before 1.0

These items improve polish and correctness but don't create permanent problems.

**Estimated effort: 5–8 hours total.**

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

**Problem:** Docs imply CA support but it's not implemented.

**Work required:** Add CA regex, test cases, update docs.

---

### 2.7 Fix `@basil/api` Module Meta Description

| | |
|---|---|
| **Priority** | 🟢 Nice to have |
| **Effort** | 5 minutes |
| **Source** | STDLIB-1.0-RELEASE-REVIEW |

**Problem:** Description says "HTTP client" but module is API route helpers.

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

**Problem:** `mddoc` and `html` not registered in metadata maps, may cause gaps in `pars describe`.

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

| # | Item | Effort | Source |
|---|------|--------|--------|
| 26 | DataTable per-column `format` override prop | 1–2h | §3 |
| 27 | DataTable custom cell render functions (`render` prop) | 2–3h | §3 |
| 28 | DataTable footer/summary row support | 1–2h | §3 |
| 29 | `FileField` component | 1h | §14 |
| 30 | `LocalNumber` / `LocalCurrency` components | 2h | §8 |
| 31 | `record.fieldProps()` method | 4–6h | §9 |
| 32 | Time components accept `datetime` objects directly | 1h | §13 |
| 33 | `record.toHTML()` with schema-aware formatting | 1–2h | §9 |
| 34 | Auto-detect external links in `Link` component | 15m | §13 |
| 35 | `request.language` server feature | 2h | §8 |

### From STDLIB Review

| Item | Effort |
|------|--------|
| Additional postal codes (DE, FR, AU, JP) | 1h |
| Date/number formatting locales (Backlog #17) | 4–8h |
| `truncateWords(n, suffix?)` string method | 1h |
| HTTP client module | 8–16h |

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

Recommended order accounting for dependencies and risk:

```
1. Tier 1.5 — DataTable redesign (resolve design decisions first)
2. Tier 1.8 — Typed value formatting (affects DataTable)
3. Tier 1.7 — Add dir prop to Page (5 min)
4. Tier 1.6 — Prelude smoke test (may uncover issues)
5. Tier 1.3 — Rename @std/mdDoc → @std/mddoc
6. Tier 1.4 — Slim @std/schema docs
7. Tier 2.6-2.10 — Polish items (can be parallelized)
```

---

## Checklist

### Tier 1 (Must do)

- [x] **FEAT-142**: Meta component created, Page updated with OG/Twitter output
- [x] **FEAT-143**: Parsley correctness (++, spread, for loops, ranges)
- [x] New prelude components (Pagination, Toast, Dialog, Details, Accordion, ErrorSummary)
- [ ] `@std/mdDoc` renamed to `@std/mddoc` with deprecation alias
- [ ] `@std/schema` docs slimmed to deprecation notice
- [ ] **DataTable redesign**: accept Table, auto-format, empty state
- [ ] Prelude component smoke test
- [ ] Add `dir` prop to `Page`
- [ ] Typed value formatting in `objectToString`/`objectToPrintString`

### Tier 2 (Should do)

- [x] Prop spreading consistency (mostly complete)
- [x] Class merging fixed (`+` not `++`)
- [x] SkipLink uses external CSS
- [x] SkipLink points to `#main`
- [x] Page `id` default fixed
- [ ] Canadian postal code in `@std/valid`
- [ ] `@basil/api` module description corrected
- [ ] `dev.md` renamed to `log.md`
- [ ] Module metadata completeness
- [ ] Documentation accuracy verification

### Gate: All Tier 1 items complete → ready for 1.0 release

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| DataTable redesign introduces bugs | Medium | Medium | Backward compatible API, extensive testing |
| Typed value formatting breaks output | Low | Medium | Pre-1.0 breaking change acceptable |
| Smoke test reveals broken components | Medium | Medium | Fix before release |
| `mdDoc` rename breaks user code | Low | Low | Deprecation alias preserves old name |

---

## Related Documents

| Document | Relevance |
|----------|-----------|
| `work/reports/STDLIB-1.0-RELEASE-REVIEW.md` | Module-level analysis |
| `work/reports/STANDARD-PRELUDE-REVIEW.md` | Component-level analysis (primary source) |
| `work/design/DESIGN-datatable-redesign.md` | DataTable redesign (Draft, needs decisions) |
| `work/design/DESIGN-typed-value-formatting.md` | Typed value formatting (Approved) |
| `work/specs/FEAT-142.md` | Meta component spec (Complete) |
| `work/specs/FEAT-143-prelude-component-styling.md` | Parsley correctness (Complete) |
| `work/specs/FEAT-129.md` | Stdlib cleanup (Complete) |
| `work/BACKLOG.md` | #17 (locales), #5 (email), #12 (form targets) |