# Standard Library 1.0 Action Plan

**Date:** 2026-03-14
**Status:** Ready for execution
**Companion report:** `STDLIB-1.0-RELEASE-REVIEW.md`
**Purpose:** Concrete, prioritised work items to bring the standard library to release quality

---

## Overview

The standard library is substantially ready for 1.0. The major cleanup (FEAT-129) was completed 2025-02-26. This plan addresses the remaining gaps identified in the companion review.

**Total estimated effort:** 8–14 hours across all tiers.

---

## Tier 1: Must Do Before 1.0

These items would create permanent API warts or active user confusion if shipped as-is. They are small, low-risk, and mechanical.

**Estimated effort: 3–5 hours total.**

### 1.1 Rename `@std/mdDoc` → `@std/mddoc`

| | |
|---|---|
| **Priority** | 🔴 Must fix |
| **Effort** | 1–2 hours |
| **Risk** | Low (mechanical rename with alias) |
| **Source** | 1.0-SHIP-REVIEW §4b |

**Problem:** `@std/mdDoc` is the only camelCase module name. All other modules are lowercase (`math`, `id`, `valid`, `hash`). This will be a permanent inconsistency if shipped in 1.0, and a breaking change to fix afterwards.

**Work required:**

1. In `pkg/parsley/evaluator/stdlib_table.go` → `getStdlibModules()`:
   - Add `"mddoc": loadMdDocModule`
   - Keep `"mdDoc"` as deprecated alias (emit DEP warning, delegate to `mddoc`)
2. In `pkg/parsley/evaluator/module_meta.go`:
   - Register `"mddoc": &mdDocModuleMeta` in `stdlibModuleMeta`
   - Add `mdDocModuleMeta` if it doesn't exist (currently missing from the metadata map)
3. Update `docs/parsley/manual/stdlib/mddoc.md`:
   - Change import example from `import @std/mdDoc` to `import @std/mddoc`
   - Add note: "`@std/mdDoc` is a deprecated alias for `@std/mddoc`"
4. Update `docs/parsley/reference.md` if it references `@std/mdDoc`
5. Add/update test in `pkg/parsley/tests/stdlib_mddoc_test.go` for the new import path
6. Run `go test ./...` — all tests must pass
7. Commit: `feat(stdlib): rename @std/mdDoc to @std/mddoc with deprecation alias`

### 1.2 Slim `@std/schema` Documentation

| | |
|---|---|
| **Priority** | 🟡 Should fix |
| **Effort** | 1 hour |
| **Risk** | None |
| **Source** | Review finding — 1,525 lines of docs for deprecated module |

**Problem:** `docs/parsley/manual/stdlib/schema.md` has a deprecation banner at the top but retains 1,525 lines of full documentation. New users will be confused about whether to use `@std/schema` or `@schema` DSL. The page actively undermines the deprecation message.

**Work required:**

1. Archive the current `schema.md` content (move to `docs/parsley/manual/stdlib/schema-legacy.md` or similar, if we want to preserve it for reference)
2. Replace `schema.md` with a slim page containing:
   - Deprecation notice
   - Migration table: old syntax → new `@schema` DSL syntax
   - Link to `@schema` DSL documentation in the reference
   - Note about TableBinding (direct users to appropriate docs)
   - "Why was this deprecated?" brief explanation
3. Target length: ~100 lines (vs current 1,525)
4. Commit: `docs(stdlib): slim @std/schema docs to deprecation notice`

### 1.3 Prelude Component Smoke Test

| | |
|---|---|
| **Priority** | 🟡 Should fix |
| **Effort** | 1–2 hours |
| **Risk** | May uncover rendering bugs |
| **Source** | 1.0-SHIP-REVIEW §12 ("still to investigate") |

**Problem:** The 26 prelude components have been code-reviewed but never verified against actual HTML rendering. The ship review flagged this as uninvestigated.

**Work required:**

1. Create a test Parsley file that uses every prelude component with representative props
2. Render it through Basil and inspect the HTML output for:
   - Valid HTML structure
   - Correct attribute rendering
   - Proper nesting
   - No error output
   - Accessibility attributes present
3. Specifically check:
   - `Page` generates valid `<!DOCTYPE html>` document
   - `Form` includes CSRF token
   - `TextField` generates correct `for`/`id` association
   - `SelectField` renders options from both simple arrays and object arrays
   - `RadioGroup` and `CheckboxGroup` render all options
   - `DataTable` renders `<thead>`/`<tbody>` correctly
   - `Breadcrumb` renders ordered list structure
   - `A` component adds `rel="noopener noreferrer"` for external links
4. Document any findings; fix any broken components
5. Commit findings (pass or fail)

---

## Tier 2: Should Do Before 1.0

These items improve polish and correctness. They don't create permanent problems but affect the quality impression at launch.

**Estimated effort: 3–5 hours total.**

### 2.1 Add Canadian Postal Code to `@std/valid`

| | |
|---|---|
| **Priority** | 🟢 Nice to have |
| **Effort** | 15 minutes |
| **Risk** | None |
| **Source** | STDLIB-AUDIT-2025 states "US, GB, CA" but CA is not implemented |

**Problem:** The audit documentation and the `valid.md` manual page imply Canadian postal code support, but the implementation in `stdlib_valid.go` only has US and GB regex patterns. The `postalCode("K1A 0B1", "CA")` call returns an "unsupported locale" error.

**Work required:**

1. Add CA regex to `stdlib_valid.go`: `^[ABCEGHJ-NPRSTVXY]\d[ABCEGHJ-NPRSTV-Z]\s?\d[ABCEGHJ-NPRSTV-Z]\d$`
2. Add `"CA"` case to `validPostalCode` switch
3. Add test cases to `stdlib_valid_test.go`
4. Update `valid.md` supported locales table to include CA
5. Commit: `feat(stdlib): add Canadian postal code validation to @std/valid`

### 2.2 Fix `@basil/api` Module Meta Description

| | |
|---|---|
| **Priority** | 🟢 Nice to have |
| **Effort** | 5 minutes |
| **Risk** | None |
| **Source** | Review finding |

**Problem:** `apiModuleMeta.Description` in `stdlib_api.go` says `"HTTP client for API requests"` but the module is actually API route helpers (auth wrappers + error helpers + redirect). This incorrect description shows up in `pars describe` output.

**Work required:**

1. Change `Description` to `"API route helpers (auth wrappers, error responses, redirects)"` or similar
2. Commit: `fix(stdlib): correct @basil/api module description`

### 2.3 Ensure Module Metadata Completeness

| | |
|---|---|
| **Priority** | 🟢 Nice to have |
| **Effort** | 30 minutes |
| **Risk** | None |
| **Source** | Review finding — some modules missing from metadata maps |

**Problem:** `module_meta.go` registers metadata for `math`, `id`, `valid`, `schema`, `api`, `dev`, `hash` in `stdlibModuleMeta`, but `mdDoc` (soon `mddoc`) and `html` are not registered. This may cause gaps in `pars describe` output.

**Work required:**

1. Verify which modules have `ModuleMeta` structs defined
2. Ensure all modules are registered in the appropriate metadata map:
   - `mddoc` → `stdlibModuleMeta` (it's a pure Parsley module)
   - `html` → `basilModuleMeta` (check if already there)
3. Create `ModuleMeta` for any module that lacks one
4. Test with `pars describe all --json` to verify completeness
5. Commit: `fix(stdlib): register all module metadata for pars describe`

### 2.4 Verify Documentation Accuracy Against `pars describe`

| | |
|---|---|
| **Priority** | 🟢 Nice to have |
| **Effort** | 2–3 hours |
| **Risk** | May uncover doc inaccuracies |
| **Source** | 1.0-SHIP-REVIEW §12 ("Documentation accuracy") |

**Work required:**

For each stdlib module:
1. Run `pars describe <module_type>` or `pars describe all --json`
2. Compare output against the corresponding manual page in `docs/parsley/manual/stdlib/`
3. Check:
   - All exported functions/methods are documented
   - Arities match
   - Descriptions are accurate
   - No documented functions that don't exist
   - No undocumented functions that do exist
4. Fix any discrepancies
5. Commit: `docs(stdlib): verify and fix manual pages against pars describe`

### 2.5 Rename `dev.md` Documentation File

| | |
|---|---|
| **Priority** | 🟢 Nice to have |
| **Effort** | 10 minutes |
| **Risk** | None |
| **Source** | Review finding |

**Problem:** `docs/parsley/manual/stdlib/dev.md` documents `@basil/log` but the filename still says "dev". Users browsing the docs directory will be confused.

**Work required:**

1. Rename `docs/parsley/manual/stdlib/dev.md` → `docs/parsley/manual/stdlib/log.md`
2. Update any internal links that reference `dev.md`
3. Optionally leave a redirect/note at the old path
4. Commit: `docs(stdlib): rename dev.md to log.md to match @basil/log`

---

## Tier 3: Post-1.0 Point Releases (1.1, 1.2)

These are genuine gaps or enhancements that would improve the stdlib but are not necessary for a credible 1.0 release.

### 3.1 New Prelude Components

| Component | Priority | Effort | Target |
|-----------|----------|--------|--------|
| `FileField` | Medium | 1–2h | 1.1 |
| `Alert` / `Flash` | Medium | 1–2h | 1.1 |
| `Pagination` | Medium | 2–3h | 1.1 |
| `NumberField` | Low | 1h | 1.2 |
| `HiddenField` | Low | 30m | 1.2 |

### 3.2 Expanded Locale Support

| Item | Priority | Effort | Target |
|------|----------|--------|--------|
| Additional postal codes (DE, FR, AU, JP) | Low | 1h | 1.1 |
| Date/number formatting locales (Backlog #17) | Medium | 4–8h | 1.1 or 1.2 |

### 3.3 String Method Additions

| Method | Priority | Effort | Rationale |
|--------|----------|--------|-----------|
| `truncateWords(n, suffix?)` | Low | 1h | Cut at word boundary instead of character. Common in CMS/blog contexts. |

### 3.4 HTTP Client Module

| | |
|---|---|
| **Priority** | Medium |
| **Effort** | 8–16 hours |
| **Target** | 1.1 or 1.2 |

An `@std/http` or `@basil/fetch` module for making outbound HTTP requests (GET, POST, etc.) to external APIs. Would enable API mashups, webhook integrations, and data fetching from third-party services. Significant scope — needs design spec.

---

## Tier 4: Defer to 2.0 or Never

These items should explicitly **not** be pursued for 1.0 or near-term point releases.

| Item | Reason to Defer |
|------|-----------------|
| Remove `@std/api`, `@std/dev`, `@std/html` deprecated aliases | Keep warnings in 1.x, hard-error in 2.0 |
| Remove `@std/schema` module entirely | Keep deprecation warning in 1.x, remove in 2.0 |
| Remove `@std/mdDoc` alias (after rename) | Keep through 1.x for migration |
| `@std/color` module | No demonstrated demand. Users can use hex strings. |
| Modal/Dialog component | Requires JavaScript, doesn't fit server-rendered philosophy |
| Enterprise features (RBAC, multi-tenancy, etc.) | Out of scope for target audience |

---

## Execution Order

The recommended order of execution, accounting for dependencies and risk:

```
1. Tier 1.3 — Prelude smoke test (do first to uncover surprises)
2. Tier 1.1 — Rename @std/mdDoc → @std/mddoc
3. Tier 1.2 — Slim @std/schema docs
4. Tier 2.2 — Fix api module meta description (trivial)
5. Tier 2.5 — Rename dev.md → log.md (trivial)
6. Tier 2.1 — Add CA postal code (small)
7. Tier 2.3 — Module metadata completeness
8. Tier 2.4 — Documentation accuracy verification (last, as other changes may affect docs)
```

Steps 4 and 5 can be done as a single commit if desired. Steps 2, 3, 6, and 7 should each be separate commits.

---

## Checklist

### Tier 1 (Must do)

- [ ] `@std/mdDoc` renamed to `@std/mddoc` with deprecation alias
- [ ] `@std/schema` docs slimmed to deprecation notice + migration guide
- [ ] All 26 prelude components smoke-tested for rendering

### Tier 2 (Should do)

- [ ] Canadian postal code (`"CA"`) added to `@std/valid`
- [ ] `apiModuleMeta.Description` corrected
- [ ] All modules registered in metadata maps
- [ ] Manual pages verified against `pars describe` output
- [ ] `dev.md` renamed to `log.md`

### Gate: All Tier 1 and Tier 2 items complete → stdlib is release-ready

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| `mdDoc` rename breaks user code | Low (pre-1.0) | Low (alias preserves old name) | Deprecation alias + warning |
| Prelude smoke test reveals broken components | Medium | Medium | Fix before release; components are small `.pars` files |
| Schema docs slim loses useful TableBinding info | Low | Medium | Relocate to `@schema` DSL docs, don't delete |
| Metadata changes break `pars describe` | Low | Low | Run `pars describe all --json` before and after |

---

## Related Documents

| Document | Relevance |
|----------|-----------|
| `work/reports/STDLIB-1.0-RELEASE-REVIEW.md` | Companion analysis report |
| `work/reports/STDLIB-AUDIT-2025.md` | Prior audit (most recommendations implemented) |
| `work/reports/1.0-SHIP-REVIEW.md` | Broader ship review (§4 and §12 relevant) |
| `work/reports/1.0-READINESS-AUDIT.md` | Prior readiness assessment |
| `work/specs/FEAT-129.md` | Stdlib cleanup spec (complete) |
| `work/BACKLOG.md` | #17 (locales), #5 (email), #12 (form targets) |