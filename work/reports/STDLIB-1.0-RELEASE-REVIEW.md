# Standard Library 1.0 Release Review

**Date:** 2026-03-14
**Updated:** 2026-03-15
**Status:** Complete — Action plan in `STDLIB-1.0-ACTION-PLAN.md`
**Scope:** All `@std/` and `@basil/` modules, prelude components, documentation
**Prior work:** `STDLIB-AUDIT-2025.md`, `1.0-SHIP-REVIEW.md` §4/§12, `1.0-READINESS-AUDIT.md`, `FEAT-129`

---

## Executive Summary

The Basil standard library is **substantially ready for 1.0 release**. The major cleanup work from FEAT-129 (completed 2025-02-26) addressed most of the issues identified in the STDLIB-AUDIT-2025: redundant modules were removed, namespace boundaries were established, new modules were added, and documentation was written.

However, this review identifies **3 items that should be fixed before 1.0** and **6 items that should be addressed soon after**. Nothing is fundamentally broken — the remaining issues are naming inconsistencies, documentation hygiene, and a small number of missing features that would be expected by the target audience.

### Overall Assessment

| Area | Rating | Verdict |
|------|--------|---------|
| `@std/` modules (pure Parsley) | 8/10 | Ship with one rename |
| `@basil/` modules (server) | 7/10 | Ship as-is |
| Prelude components (`@basil/html`) | 7/10 | Ship after smoke test |
| Documentation | 7/10 | Ship after schema doc cleanup |
| Test coverage | 7/10 | Adequate for 1.0 |

---

## Table of Contents

1. [Prior Work and What Was Already Done](#1-prior-work-and-what-was-already-done)
2. [Module-by-Module Assessment](#2-module-by-module-assessment)
3. [Prelude Components Assessment](#3-prelude-components-assessment)
4. [MdDoc Assessment](#4-mddoc-assessment)
5. [Organisation and Naming](#5-organisation-and-naming)
6. [Documentation Assessment](#6-documentation-assessment)
7. [Gaps Analysis (Against Target Audience)](#7-gaps-analysis-against-target-audience)
8. [Consistency and Usability Review](#8-consistency-and-usability-review)
9. [Relevant Backlog and Spec Items](#9-relevant-backlog-and-spec-items)
10. [Summary of Findings](#10-summary-of-findings)

---

## 1. Prior Work and What Was Already Done

### Reports Consulted

| Report | Date | Relevance |
|--------|------|-----------|
| `STDLIB-AUDIT-2025.md` | 2025-02-26 | Comprehensive module audit with decisions. Most recommendations implemented. |
| `1.0-SHIP-REVIEW.md` §4 | 2026-02-28 | Namespace organisation issues. Partially addressed. |
| `1.0-SHIP-REVIEW.md` §12 | 2026-02-28 | Lists prelude components as uninvestigated. Still uninvestigated until now. |
| `1.0-READINESS-AUDIT.md` | 2025-02-26 | Broader readiness including stdlib completeness. Identified deprecations. |

### FEAT-129 Implementation Status

FEAT-129 (Parsley Standard Library v1.0 Cleanup) was completed 2025-02-26. All seven phases were marked complete:

| Phase | Scope | Status |
|-------|-------|--------|
| 1: Deprecations | `@std/schema` warning | ✅ Implemented |
| 2: `@std/valid` refactor | 27 functions removed, 3 added | ✅ Implemented |
| 3: `@std/hash` module | MD5, SHA1, SHA256, SHA512 | ✅ Implemented |
| 4: String methods | toBase64, fromBase64, toCamel, toPascal, toSnake, toKebab, truncate | ✅ Implemented |
| 5: Namespace moves | `@std/api` → `@basil/api`, `@std/dev` → `@basil/log`, `@std/html` → `@basil/html` | ✅ Implemented |
| 6: Documentation | Reference docs, manual pages, migration guide | ✅ Implemented |
| 7: Testing | New validators, hash, string method tests | ✅ Implemented |

### What Was Not Done

Despite FEAT-129 completion, the following recommended items from the STDLIB-AUDIT and 1.0-SHIP-REVIEW remain open:

| Item | Source | Status |
|------|--------|--------|
| Rename `@std/mdDoc` → `@std/mddoc` | 1.0-SHIP-REVIEW §4b | ❌ Not done — still camelCase in registry |
| Prelude component rendering verification | 1.0-SHIP-REVIEW §12 | ❌ Not done — flagged as "still to investigate" |
| Canadian postal code (`"CA"`) in `@std/valid` | STDLIB-AUDIT-2025 mentions "US, GB, CA" | ❌ Code only has US and GB |
| Schema docs slimmed for deprecated status | Implied by deprecation | ❌ Still 1,500+ lines |

---

## 2. Module-by-Module Assessment

### `@std/math` — ✅ Ship (9/10)

**Contents:** Constants (PI, E, TAU), rounding, aggregation (sum, mean, median, mode, stddev, variance), random, transcendental functions (sin, cos, tan, asin, acos, atan, atan2, log, log2, log10, exp, pow, sqrt), geometry/interpolation (lerp, clamp, map, hypot), abs, sign, gcd, lcm.

- Well-designed, follows standard library conventions across languages
- No overlap with builtins or Query DSL
- No overlap with other modules
- Good test coverage (`stdlib_math_test.go`)
- Well-documented (`docs/parsley/manual/stdlib/math.md`)
- Appropriate scope for target audience (educators, science students, hobbyists)

**No action required.**

### `@std/id` — ✅ Ship (9/10)

**Contents:** `new()` (ULID), `uuid()`/`uuidv4()`, `uuidv7()`, `nanoid(len?)`, `cuid()`

- Clean, focused API
- Every generated ID type has a corresponding validator in `@std/valid`
- Good test coverage (`stdlib_id_test.go`)
- Well-documented (`docs/parsley/manual/stdlib/id.md`)

**No action required.**

### `@std/valid` — ✅ Ship (8/10)

**Contents (post-FEAT-129):** `uuid`, `ulid`, `nanoid`, `cuid`, `creditCard`, `luhn`, `postalCode`

- Focused, non-redundant API
- ID validators cover all types generated by `@std/id`
- Pure predicates (return boolean, no side effects)
- Well-documented with migration guide from old API (`docs/parsley/manual/stdlib/valid.md`)
- Good test coverage (`stdlib_valid_test.go`)

**Minor issue:** The STDLIB-AUDIT-2025 states "Supported: US, GB, CA" for `postalCode`, but the implementation only has US and GB regex patterns. Canadian postal code support (`"CA"`) is missing.

**Recommendation:** Add CA postal code regex before 1.0 (estimated 15 minutes).

### `@std/hash` — ✅ Ship (8/10)

**Contents:** `md5(str)`, `sha1(str)`, `sha256(str)`, `sha512(str)`

- New module created in FEAT-129
- Clean implementation using Go's crypto packages
- Returns hex-encoded strings (standard convention)
- Correctly documented as non-security hashing (password hashing uses Basil auth)
- Good test coverage (`stdlib_hash_test.go`)
- Well-documented (`docs/parsley/manual/stdlib/hash.md`)

**No action required.**

### `@std/mdDoc` — ⚠️ Rename Needed (7/10)

**Contents:** Constructor `mdDoc(text)`, rendering (toMarkdown, toHTML), queries (findAll, findFirst, headings, links, images, codeBlocks), convenience (title, toc, text, wordCount), transforms (walk, map, filter), raw AST access.

See [§4 MdDoc Assessment](#4-mddoc-assessment) for detailed analysis.

**Blocking issue:** Only camelCase module name. Must be `@std/mddoc` before 1.0.

### `@std/schema` — ⚠️ Documentation Needs Cleanup (4/10)

**Status:** Deprecated. Emits DEP-002 warning on import, directing users to `@schema` DSL.

The module still functions (the warning doesn't prevent use), which is appropriate for a 1.0 transition. However, the documentation is problematic:

- `docs/parsley/manual/stdlib/schema.md` is **1,525 lines** with a deprecation banner at the top but full detailed documentation below
- This sends mixed signals: "don't use this" followed by comprehensive usage guide
- New users will be confused about whether to use `@std/schema` or `@schema` DSL
- The page documents `schema.table()` / TableBinding which is still functional — this documentation needs to be relocated or clearly marked

**Recommendation:** Slim to deprecation notice + migration table + link to `@schema` DSL docs. The TableBinding documentation should move to the schema DSL reference if it hasn't been covered there already.

### `@std/table` — ✅ Already Deprecated (Correct)

Returns a clear error pointing to `@table` literal syntax. No documentation confusion. Handled correctly.

**No action required.**

### `@basil/api` — ✅ Ship (7/10)

**Contents:** Auth wrappers (`public`, `adminOnly`, `roles`, `auth`), error helpers (`notFound`, `forbidden`, `badRequest`, `unauthorized`, `conflict`, `serverError`), redirect helper.

- Clean API for the most common API patterns
- Auth wrappers are well-designed (wrap function + attach metadata)
- Error helpers use consistent pattern with optional custom message
- Redirect supports paths and status codes with validation
- Available at both `@basil/api` (canonical) and `@std/api` (deprecated alias with warning)

**Minor note:** The module meta description says "HTTP client for API requests" but the module is actually "HTTP API route helpers" (auth wrappers + error helpers). This is a metadata inaccuracy.

**Recommendation:** Fix the `apiModuleMeta.Description` to accurately describe the module.

### `@basil/log` — ✅ Ship (7/10)

**Contents:** `dev.log(value)`, `dev.log(label, value)`, `dev.clearLog()`, `dev.logPage(route, value)`, `dev.setLogRoute(route)`, `dev.clearLogPage(route)`. Supports log levels via options dict.

- Correctly no-ops in CLI mode (reads `env.DevLog` at call time, not import time)
- Available at both `@basil/log` (canonical) and `@std/dev` (deprecated alias with warning)
- Good source-line reading for call representation
- Supports log levels (info, warn)

**Design note:** The module exports `dev` as a sub-object, so usage is `let {dev} = import @basil/log; dev.log("hi")`. This is slightly unusual — most modules export functions directly. However, it's consistent with the module's identity as a namespace.

**No action required.**

### `@basil/html` — ✅ Ship After Verification (7/10)

See [§3 Prelude Components Assessment](#3-prelude-components-assessment) for detailed analysis.

### `@basil/http`, `@basil/auth`, `@basil/session`, `@basil/csrf` — Out of Scope

These are core server modules, not stdlib modules. Not assessed in this review.

---

## 3. Prelude Components Assessment

### Overview

The `@basil/html` module loads 26 components from `.pars` files in `server/prelude/components/`. Components are evaluated in a shared environment so they can reference each other (e.g., Page uses SkipLink).

### Component Inventory

| Category | Components | Count |
|----------|-----------|-------|
| Layout | Page, Meta (Head is deprecated alias) | 2 |
| Form | TextField, TextareaField, SelectField, RadioGroup, CheckboxGroup, Checkbox, Button, Form | 8 |
| Navigation | Nav, Breadcrumb, SkipLink | 3 |
| Media | Img, Iframe, Figure, Blockquote | 4 |
| Utility | SrOnly, Abbr, A, Icon | 4 |
| Time | Time, LocalTime, TimeRange, RelativeTime | 4 |
| Data | DataTable | 1 |
| **Total** | | **26** |

> **Update (2026-03-15):** FEAT-142 completed. `Head` component replaced by `Meta` component. `Head` remains as a deprecated alias for backward compatibility. `Page` now outputs Open Graph and Twitter metadata automatically from its `title` and `description` props.

### Quality Assessment by Category

#### Form Components — High Quality ✅

All form components follow a consistent, well-designed pattern:

1. **Accessibility**: Proper `aria-describedby`, `aria-invalid`, `aria-required` attributes. Labels are programmatically associated with inputs via `for`/`id`. Fieldsets use `<legend>` for groups.
2. **Prop spreading**: Components extract known props and spread the rest to the underlying HTML element via `...inputAttrs` / `...selectAttrs` / etc. This is an excellent pattern that allows arbitrary HTML attributes.
3. **Error/hint pattern**: Consistent `error` and `hint` props across all form components, rendered as `<p>` elements with `role="alert"` for errors.
4. **Required field indication**: All form components support `required` prop with consistent visual indicator (`<span class="field-required">*</span>`) and proper `aria-required`.

Specific notes:

| Component | Notes |
|-----------|-------|
| `TextField` | Solid. Generates unique IDs for accessibility. Supports all HTML input types via `type` prop. |
| `TextareaField` | Consistent with TextField pattern. |
| `SelectField` | Supports both simple value arrays and object arrays with configurable `valueKey`/`labelKey`. Has `autosubmit` data attribute. Has `placeholder` option support. |
| `RadioGroup` | Uses `<fieldset>`/`<legend>`. Only first radio gets `required` (correct HTML behaviour). Supports `disabled`. |
| `CheckboxGroup` | Consistent with RadioGroup. Uses `name[]` convention for multiple values. Uses `.has()` for checked state. |
| `Checkbox` | Standalone single checkbox variant. |
| `Button` | Should verify `type` defaults. |
| `Form` | CSRF token auto-injected from `request.csrf`. Supports `confirm` dialog via `data-confirm` attribute. Defaults to POST method. |

#### Layout Components — Good Quality ✅

| Component | Notes |
|-----------|-------|
| `Page` | Generates complete HTML document (doctype, html, head, body). Integrates SkipLink, CSS bundles, JavaScript bundles, and optional BasilJS. Supports `noBasilJS` for pages not needing enhanced components. `lang` defaults to `"en"`. **Updated (FEAT-142):** Now validates `title` is provided, and automatically outputs `og:title`, `og:description`, `twitter:title`, `twitter:description` from its props. |
| `Meta` | **(New in FEAT-142)** SEO and social media metadata tags. Outputs meta/link tags only (no `<head>` wrapper) for composition inside `Page`'s `head` prop. Handles `image`, `url`, `type`, `author`, `published`, `modified`, `twitter`, favicons, `noIndex`. |
| `Head` | **(Deprecated)** Alias for `Meta`. Kept for backward compatibility. |

#### Navigation Components — Good Quality ✅

| Component | Notes |
|-----------|-------|
| `Nav` | Simple wrapper with `aria-label`. Correct and minimal. |
| `Breadcrumb` | Uses `<nav>` with `<ol>` — correct semantic structure per WAI-ARIA breadcrumb pattern. |
| `SkipLink` | Accessibility "skip to content" link. Correctly included in Page component. |

#### Media Components — Good Quality ✅

| Component | Notes |
|-----------|-------|
| `Img` | Image element. Should verify if `loading="lazy"` is default or opt-in. |
| `Iframe` | Iframe with proper attributes. |
| `Figure` | `<figure>` with `<figcaption>`. Correct semantics. |
| `Blockquote` | Wraps `<blockquote>` in `<figure>` with optional `<cite>` via `<figcaption>`. Good semantic markup. |

#### Time Components — Good Quality ✅

| Component | Notes |
|-----------|-------|
| `Time` | HTML `<time>` element with `datetime` attribute. |
| `LocalTime` | Client-side localised time display (requires BasilJS). |
| `TimeRange` | Displays time range (start–end). |
| `RelativeTime` | "3 hours ago" style display (requires BasilJS). |

#### Data Components — Good Quality ✅

| Component | Notes |
|-----------|-------|
| `DataTable` | Proper table semantics with `<caption>`, `<thead>`, `<tbody>`, `scope="col"` on header cells, and `scope="row"` on first data column. Accepts `columns`, `rows`, `keys` props. Has unused `sortable` prop (noted for future enhancement). |

#### Utility Components — Adequate ✅

| Component | Notes |
|-----------|-------|
| `SrOnly` | Screen-reader-only text (CSS visually hidden). |
| `Abbr` | `<abbr>` element with `title` attribute for abbreviations. |
| `A` | Anchor/link element. Adds `rel="noopener noreferrer"` for external links. |
| `Icon` | Renders icons. Uses `aria-hidden` to hide decorative icons from screen readers. |

### Rendering Verification Status

⚠️ **No rendering verification has been performed.** The 1.0-SHIP-REVIEW §12 flagged this as "still to investigate". While code review shows the components are well-structured, they have not been tested with actual HTML rendering in a browser.

**Recommendation:** Perform a smoke test of all 26 components to verify they render valid, well-formed HTML.

### Missing Components (Gap Analysis)

| Component | Priority | Rationale |
|-----------|----------|-----------|
| `FileField` | Medium | File upload is common for SMB sites (logos, documents, images). Currently requires raw HTML `<input type="file">`. |
| `NumberField` | Low | `TextField` with `type="number"` works, but a dedicated component with `min`/`max`/`step` props would be more ergonomic. |
| `Alert` / `Flash` | Medium | Flash messages / notification banners are extremely common in form-heavy web apps. Currently requires manual HTML. |
| `Pagination` | Medium | DataTable may have its own but no standalone component. Very common for list pages. |
| `HiddenField` | Low | Trivial with `<input type="hidden">`, but would complete the form component set. |
| `DateField` | Low | `TextField` with `type="date"` works. Dedicated component could add datepicker integration. |
| `Modal` / `Dialog` | Low | Requires JavaScript. May not fit server-rendered philosophy. Defer. |

**Verdict:** The existing 26 components cover the 80% case well. The gaps are things users can build with native HTML tags in Parsley. None are blocking for 1.0. FileField, Alert, and Pagination would be valuable additions in 1.1.

### Fit-for-Purpose Verdict

The component library is **fit for purpose** for the target audience:

- **Web developers making SMB sites**: Full form suite, proper accessibility, CSRF integration
- **Hobbyists**: Sufficient components to build a complete website without touching raw HTML
- **Educators**: Good examples of accessible HTML patterns
- **PHP/Rails developers**: Similar to Rails form helpers / Laravel Blade components

The library follows a "batteries included but not bloated" philosophy consistent with Go's stdlib approach.

---

## 4. MdDoc Assessment

### Functionality

The `@std/mdDoc` module provides a complete markdown analysis and manipulation toolkit:

| Category | Methods |
|----------|---------|
| **Construction** | `mdDoc(text)`, `mdDoc(dict)` |
| **Rendering** | `toMarkdown()`, `toHTML()` |
| **Querying** | `findAll(type)`, `findFirst(type)`, `headings()`, `links()`, `images()`, `codeBlocks()` |
| **Convenience** | `title()`, `toc(options?)`, `text()`, `wordCount()` |
| **Transforms** | `walk(fn)`, `map(fn)`, `filter(fn)` |
| **Raw access** | `.ast` |

### Implementation Quality

- Built on `goldmark` (the standard Go Markdown library, CommonMark compliant)
- 17 methods covering the complete document lifecycle (parse → query → transform → render)
- Transform methods return new `MdDoc` objects (immutable pattern)
- Good error handling (type checking on arguments)
- AST uses Parsley-native dictionaries, so users can inspect/manipulate with normal Parsley code

### Use Cases for Target Audience

| Audience | Use Case |
|----------|----------|
| Web developers | Blog engines, CMS content rendering, documentation sites |
| IT workers | Report generation from Markdown templates |
| Educators | Teaching Markdown structure, document analysis |
| Science students | Processing lab notes, generating reports |
| Hobbyists | Static site generators, personal wikis |

### Quality Rating: 7/10

**Strengths:**
- Complete API — covers parse, query, transform, render
- Follows Parsley conventions (returns Parsley types, uses method chaining)
- Well-documented (`docs/parsley/manual/stdlib/mddoc.md`)
- Good test coverage (`pkg/parsley/tests/stdlib_mddoc_test.go`)

**Concerns:**
1. **Naming**: `@std/mdDoc` is the only camelCase module name. Must be renamed.
2. **Constructor stutter**: After rename, usage is `mddoc.mddoc("text")`. This follows the module convention (`id.uuid()`, `hash.sha256()`) but the repetition is slightly awkward. Not a blocker — it's consistent.
3. **markdown_helpers.go is large**: 1,231 lines of helper functions. Functional but could benefit from splitting in future. Not a 1.0 concern.

### Completeness

The module covers the standard markdown analysis use cases. Compared to similar tools in other ecosystems:

| Feature | @std/mdDoc | Python markdown | Ruby kramdown |
|---------|-----------|-----------------|---------------|
| Parse to AST | ✅ | ✅ | ✅ |
| Render to HTML | ✅ | ✅ | ✅ |
| Render to Markdown | ✅ | ❌ | ✅ |
| Query headings | ✅ | Manual | Manual |
| Query links | ✅ | Manual | Manual |
| Table of contents | ✅ | Extension | Extension |
| Word count | ✅ | Manual | Manual |
| AST transforms | ✅ (walk/map/filter) | Manual | Plugins |

The module is actually more ergonomic than most Markdown libraries in other languages. The query methods (`headings()`, `links()`, `images()`, `codeBlocks()`) provide structured access that usually requires manual AST walking in Python/Ruby.

---

## 5. Organisation and Naming

### Namespace Structure (Current)

```
@std/                           @basil/
├── math      ✅                ├── http    (core)
├── id        ✅                ├── auth    (core)
├── valid     ✅                ├── session (core)
├── hash      ✅                ├── csrf    (core)
├── mdDoc     ⚠️ camelCase     ├── api     ✅
├── schema    ⚠️ deprecated    ├── log     ✅
│                               └── html    ✅
│
│  Deprecated aliases (emit warning, still work):
├── api    → @basil/api
├── dev    → @basil/log
└── html   → @basil/html
```

### Namespace Structure (Target for 1.0)

```
@std/                           @basil/
├── math                        ├── http
├── id                          ├── auth
├── valid                       ├── session
├── hash                        ├── csrf
├── mddoc    ← renamed         ├── api
│                               ├── log
│                               └── html
│
│  Deprecated aliases (emit warning):
├── mdDoc  → @std/mddoc  (NEW)
├── api    → @basil/api
├── dev    → @basil/log
└── html   → @basil/html
└── schema → error + migration guide
```

### Naming Issues

| Issue | Severity | Status |
|-------|----------|--------|
| `@std/mdDoc` is camelCase (all others lowercase) | Should fix | ❌ Open |
| `@std/schema` fully deprecated but docs remain | Should fix | ❌ Open |
| `apiModuleMeta.Description` says "HTTP client" (wrong) | Minor | ❌ Open |
| Module metadata not in `stdlibModuleMeta` for `html` | Minor | Needs verification |

### Namespace Principle Compliance

The namespace principle from STDLIB-AUDIT-2025 is:

> - `@std/` — Pure Parsley functionality (works without Basil server)
> - `@basil/` — Server-specific functionality (requires Basil runtime context)

**Current compliance:** ✅ Good. The three server-dependent modules (`api`, `dev`/`log`, `html`) are available in `@basil/` namespace. The old `@std/` paths emit deprecation warnings.

**Concern:** The old `@std/api`, `@std/dev`, `@std/html` aliases should remain as warnings through 1.0 to avoid breaking existing user code. They should become hard errors in 2.0.

---

## 6. Documentation Assessment

### Manual Pages

| Module | Manual Page | Status |
|--------|------------|--------|
| `@std/math` | `docs/parsley/manual/stdlib/math.md` | ✅ Present |
| `@std/id` | `docs/parsley/manual/stdlib/id.md` | ✅ Present |
| `@std/valid` | `docs/parsley/manual/stdlib/valid.md` | ✅ Present, includes migration guide |
| `@std/hash` | `docs/parsley/manual/stdlib/hash.md` | ✅ Present |
| `@std/mdDoc` | `docs/parsley/manual/stdlib/mddoc.md` | ✅ Present |
| `@std/schema` | `docs/parsley/manual/stdlib/schema.md` | ⚠️ Problematic — see below |
| `@basil/api` | `docs/parsley/manual/stdlib/api.md` | ✅ Present |
| `@basil/log` | `docs/parsley/manual/stdlib/dev.md` | ✅ Present (filename still "dev") |
| `@basil/html` | `docs/parsley/manual/stdlib/html.md` | ✅ Present, comprehensive |
| `@basil/session` | `docs/parsley/manual/stdlib/session.md` | ✅ Present |
| `@std/table` | `docs/parsley/manual/stdlib/table.md` | ⚠️ Deprecated — verify it says so |

### Schema Documentation Problem

`docs/parsley/manual/stdlib/schema.md` is 1,525 lines. It has a deprecation banner at the top but then provides full, detailed documentation for the deprecated API including:

- All type factories (string, email, url, phone, integer, number, boolean, enum, date, datetime, money, id, array, object)
- Schema operations (define, validate, table)
- TableBinding in-depth guide (475 lines)
- Complete examples
- Best practices

**Problem:** This sends mixed signals. A new user landing on this page sees "deprecated" but then 1,500 lines of documentation that looks current. They won't know whether to use `@std/schema` or `@schema` DSL.

**Resolution needed:** The page should be reduced to:
1. Deprecation notice
2. Migration table showing old → new syntax
3. Link to `@schema` DSL documentation
4. Note about TableBinding (if it's still the recommended way to do database binding, document it in the `@schema` DSL docs instead)

### Documentation Filename Issues

| File | Content | Issue |
|------|---------|-------|
| `stdlib/dev.md` | Documents `@basil/log` | Filename should be `log.md` to match new module name |
| `stdlib/mddoc.md` | Documents `@std/mdDoc` | Filename correct for target rename, content references `mdDoc` |

### Documentation Accuracy

Documentation accuracy was not verified against `pars describe` output as part of this review. This should be done as a Tier 2 action item — run `pars describe <type>` for each module and compare against manual page contents.

---

## 7. Gaps Analysis (Against Target Audience)

### Target Audience Recap

- Web developers making websites for small-to-medium businesses
- IT workers and sales/marketing people needing reports from medium datasets
- Hobbyists
- Educators
- Science students not dealing with big data
- People who would ordinarily reach for a spreadsheet

### What the Stdlib Provides vs What's Needed

| Need | Provided By | Status |
|------|------------|--------|
| Math/statistics | `@std/math` | ✅ Complete (mean, median, mode, stddev, variance, random) |
| String manipulation | Builtin string methods (50+) | ✅ Complete |
| Case conversion (camel, snake, kebab) | Builtin string methods (FEAT-129) | ✅ Complete |
| Base64 encoding | Builtin string methods (FEAT-129) | ✅ Complete |
| Truncation | Builtin string methods (FEAT-129) | ✅ Complete |
| ID generation | `@std/id` | ✅ Complete (UUID, ULID, NanoID, CUID) |
| ID validation | `@std/valid` | ✅ Complete |
| Hashing/checksums | `@std/hash` | ✅ Complete (MD5, SHA1, SHA256, SHA512) |
| Form validation | `@schema` DSL + `@std/valid` | ✅ Complete |
| HTML components | `@basil/html` (26 components) | ✅ Complete for 80% case |
| Markdown processing | `@std/mdDoc` | ✅ Complete |
| CSV processing | Builtin (`import "data.csv"`, `parseCSV`) | ✅ Complete |
| JSON processing | Builtin (`parseJSON`, `.toJSON()`) | ✅ Complete |
| Date/time handling | Builtin (`@now`, datetime, duration) | ✅ Complete |
| Money/currency | Builtin money type | ✅ Complete |
| Units/measurement | Builtin unit type | ✅ Complete |
| Database CRUD | Query DSL + schema table binding | ✅ Complete |
| HTTP API helpers | `@basil/api` | ✅ Complete |
| Authentication | `@basil/auth` | ✅ Complete |
| Session management | `@basil/session` | ✅ Complete |
| Regex | Builtin (match operator `~`) | ✅ Complete |
| File I/O | Builtin (read/write operators) | ✅ Complete |
| Debug logging | `@basil/log` | ✅ Complete |

### What's Missing

| Gap | Target Audience Impact | Priority | Notes |
|-----|----------------------|----------|-------|
| Locale coverage for date/number formatting | Medium — non-English SMB sites | Medium | Only US, GB, ISO. Missing common European/Asian locales. Backlog #17. |
| Canadian postal code validation | Low — CA users affected | Low | Mentioned in audit docs as supported but not implemented. |
| HTTP client for external API calls | Medium — API mashups | Medium | Would allow fetching from external APIs in handlers. No module exists. |
| Email sending | Medium — SMB sites need transactional email | Medium | Backlog #5, FEAT-084 Phase 3. Core verification works, developer-initiated emails don't. |
| File upload component | Medium — common for SMB sites | Medium | No `<FileField>` prelude component. Raw HTML works. |
| Alert/notification component | Medium — common for form workflows | Medium | No `<Alert>` or `<Flash>` component. Raw HTML works. |
| Pagination component | Medium — common for list pages | Medium | No standalone pagination. DataTable may handle internally. |

### What's Surplus

**Nothing.** The FEAT-129 cleanup already removed all redundancy. Every remaining module serves a clear, non-overlapping purpose.

### Comparison with Competitor Standard Libraries

| Feature Area | Basil/Parsley | PHP | Python | Ruby | Go |
|-------------|--------------|-----|--------|------|-----|
| Math/stats | ✅ `@std/math` | ✅ Built-in | ✅ `math`/`statistics` | ✅ `Math` | ✅ `math` |
| String manipulation | ✅ 50+ methods | ✅ Built-in | ✅ Built-in | ✅ Built-in | ✅ `strings` |
| Hashing | ✅ `@std/hash` | ✅ `hash()` | ✅ `hashlib` | ✅ `digest` | ✅ `crypto` |
| ID generation | ✅ `@std/id` | ❌ Package | ❌ Package | ❌ Gem | ❌ Package |
| Validation | ✅ `@std/valid` + `@schema` | ❌ Framework | ❌ Package | ❌ Gem | ❌ Package |
| HTML components | ✅ `@basil/html` (26) | ❌ Framework | ❌ Framework | ✅ Rails helpers | ❌ Template |
| Markdown | ✅ `@std/mdDoc` | ❌ Package | ❌ Package | ❌ Gem | ❌ Package |
| CSV | ✅ Built-in | ✅ `fgetcsv` | ✅ `csv` | ✅ `CSV` | ✅ `encoding/csv` |
| JSON | ✅ Built-in | ✅ `json_*` | ✅ `json` | ✅ `json` | ✅ `encoding/json` |
| Database | ✅ Query DSL | ❌ PDO | ❌ Package | ✅ ActiveRecord | ✅ `database/sql` |

Basil's stdlib is **more "batteries included" than Go, PHP, or Python** for web development, and **comparable to Ruby on Rails** for form/HTML helpers. This is appropriate for the target audience.

---

## 8. Consistency and Usability Review

### API Consistency

| Pattern | Consistency | Notes |
|---------|-------------|-------|
| Module naming | ⚠️ `mdDoc` is the only camelCase | All others lowercase |
| Error handling | ✅ Consistent | Type errors return errors, non-type args return `false` for validators |
| Return types | ✅ Consistent | Validators return boolean, hash returns string, queries return arrays |
| Argument patterns | ✅ Consistent | Required args first, optional args last, options dict where appropriate |
| Method naming | ✅ Consistent | camelCase methods (`toHTML`, `findAll`, `wordCount`) |
| Documentation style | ✅ Consistent | All manual pages use same frontmatter, same section structure |

### Usability for Newcomers from Other Languages

| Potential Confusion Point | Severity | Mitigation |
|--------------------------|----------|------------|
| `@std/` vs `@basil/` namespace split | Medium | Clear in docs, but someone from Python/PHP expects one namespace. Add a prominent "Module Organisation" section to getting-started guide. |
| `@schema` DSL vs `@std/schema` (deprecated) | High | Deprecated module docs still large. New users may try the old API. Slim the docs. |
| `mdDoc.mdDoc()` constructor stutter | Low | Consistent with other modules (`id.uuid()`, `hash.sha256()`). Rename to `mddoc.mddoc()` doesn't help the stutter but fixes the naming inconsistency. |
| Validators return boolean, not error | Low | Different from Joi/Zod/Cerberus. Well-documented. Schema DSL provides error-returning validation. |
| No `print()` function | High | Already addressed in Parsley docs and cheatsheet. The expression-based output model is Parsley's most unusual feature. Not a stdlib issue. |
| Import syntax `let valid = import @std/valid` | Medium | Unusual compared to `import valid from '@std/valid'`. Already documented. Not a stdlib issue. |

### Duplication Check

| Area | Finding |
|------|---------|
| Validation | ✅ No duplication. `@std/valid` handles predicates, `@schema` DSL handles structured validation. Clear separation. |
| Hashing | ✅ No duplication. Only in `@std/hash`. |
| HTML utilities | ✅ No duplication. `htmlEncode`/`htmlDecode`/`stripHtml` are string methods. `@basil/html` has components. No overlap. |
| Date/time | ✅ No duplication. Builtin types only. No stdlib module. |
| Math | ✅ No duplication. `@std/math` doesn't overlap with builtin numeric methods. |

---

## 9. Relevant Backlog and Spec Items

### Already Completed (Verified)

| ID | Item | Completed In |
|----|------|-------------|
| #57 | Rename `@std/dev` to `@basil/log` | FEAT-129 |
| #21 | Form validation/sanitization | FEAT-032 / `@std/valid` |
| #90 | Custom message for `failIfInvalid(msg?)` | FEAT-127 |
| #97 | Named capture groups return dictionaries | FEAT-127 |
| #11 | Remove `@std/basil` error | FEAT-071 |

### Open and Relevant to Stdlib

| ID | Item | Priority for 1.0 | Recommendation |
|----|------|------------------|----------------|
| #17 | Standardize locale support across stdlib | Post-1.0 | Document the gap. Only US/GB/ISO supported. Plan for 1.1. |
| #5 | Notification API (`basil.email.send`) | Post-1.0 | Significant scope. Not stdlib — would be `@basil/email`. |
| #12 | Form `target=` partial updates | Post-1.0 | Enhancement for Form component. |
| #34 | Error code documentation/help system | Post-1.0 | Not stdlib-specific. |
| #55 | Deprecate `format(arr, style?)` builtin | Already done | Deprecation warning added in FEAT-127. |
| #54 | Builtin Table type | Post-1.0 | Would affect stdlib if `@std/table` were un-deprecated. Currently deferred. |

### Unimplemented Feature Specs

No unimplemented feature specs were found that directly relate to stdlib modules. FEAT-129 (the cleanup spec) is complete. The remaining FEAT specs (FEAT-130 through FEAT-141) address other areas (benchmarks, testing infrastructure, performance, etc.).

---

## 10. Summary of Findings

### By Severity

| Severity | Count | Items |
|----------|-------|-------|
| 🔴 Must fix before 1.0 | 1 | `@std/mdDoc` rename to `@std/mddoc` |
| 🟡 Should fix before 1.0 | 2 | Schema docs cleanup; prelude component render verification |
| 🟢 Nice to have for 1.0 | 4 | CA postal code; api module meta description; dev.md filename; docs accuracy check |
| 📋 Post-1.0 | 7 | FileField, Alert, Pagination components; locale coverage; HTTP client; email; truncateWords |

### What Is High Quality

- `@std/math` — Excellent, focused, well-tested
- `@std/id` — Excellent, focused
- `@std/valid` — Clean after FEAT-129 refactor
- `@std/hash` — Simple and correct
- Prelude form components — Excellent accessibility, consistent patterns
- String methods — Comprehensive (50+ methods including new FEAT-129 additions)
- Documentation — Manual pages exist for all modules with consistent structure

### What Needs Reworking

- `@std/mdDoc` naming (must rename to `@std/mddoc`)
- `@std/schema` documentation (must slim to deprecation notice)

### What Is Surplus

- Nothing. FEAT-129 already removed redundancy.

### What Is Inconsistent

- `@std/mdDoc` camelCase naming vs all-lowercase convention
- `apiModuleMeta.Description` says "HTTP client" instead of "API route helpers"
- `docs/parsley/manual/stdlib/dev.md` filename doesn't match `@basil/log` rename

### Where Is There Duplication

- None found. Clear separation between modules.

### What Would Be Difficult for Newcomers

- `@std/` vs `@basil/` namespace split (medium — needs prominent docs)
- `@std/schema` deprecation with large docs still present (high — actively confusing)
- Expression-based output model (high — but not a stdlib issue)

---

## Appendix A: Complete Module Registry

As of this review, the module registries in `stdlib_table.go` are:

### `@std/` modules (`getStdlibModules()`)

```go
"dev":    loadDevModule,     // deprecated alias → @basil/log
"math":   loadMathModule,
"valid":  loadValidModule,
"schema": loadSchemaModule,  // deprecated → @schema DSL
"id":     loadIDModule,
"api":    loadAPIModule,     // deprecated alias → @basil/api
"mdDoc":  loadMdDocModule,   // ⚠️ should be "mddoc"
"html":   loadHTMLModule,    // deprecated alias → @basil/html
"hash":   loadHashModule,
```

### `@basil/` modules (`getBasilModules()`)

```go
"http": loadBasilHTTPModule,
"auth": loadBasilAuthModule,
"api":  loadAPIModule,
"log":  loadDevModule,
"html": loadHTMLModule,
```

### Module metadata registrations (`module_meta.go`)

```go
stdlibModuleMeta = map[string]*ModuleMeta{
    "math":   &mathModuleMeta,
    "id":     &idModuleMeta,
    "valid":  &validModuleMeta,
    "schema": &schemaModuleMeta,
    "api":    &apiModuleMeta,
    "dev":    &devModuleMeta,
    "hash":   &hashModuleMeta,
}
// Note: "mdDoc" and "html" not in stdlibModuleMeta
```

## Appendix B: Prelude Component File Inventory

All files in `server/prelude/components/` (updated 2026-03-15 after FEAT-142):

```
a.pars              breadcrumb.pars     checkbox.pars
checkbox_group.pars data_table.pars     figure.pars
form.pars           icon.pars           iframe.pars
img.pars            local_time.pars     meta.pars
nav.pars            page.pars           radio_group.pars
relative_time.pars  select_field.pars   skip_link.pars
sr_only.pars        text_field.pars     textarea_field.pars
time.pars           time_range.pars     button.pars
abbr.pars           blockquote.pars
```

Total: 26 component files.

> **Note:** `head.pars` was deleted in FEAT-142 and replaced by `meta.pars`. The `Meta` component exports both `Meta` and `Head` (deprecated alias).

## Appendix C: Test File Inventory for Stdlib

```
pkg/parsley/tests/stdlib_api_test.go
pkg/parsley/tests/stdlib_dev_test.go
pkg/parsley/tests/stdlib_hash_test.go
pkg/parsley/tests/stdlib_id_test.go
pkg/parsley/tests/stdlib_math_test.go
pkg/parsley/tests/stdlib_mddoc_test.go
pkg/parsley/tests/stdlib_schema_test.go
pkg/parsley/tests/stdlib_table_test.go
pkg/parsley/tests/stdlib_valid_test.go
```

All `@std/` modules have dedicated test files. `@basil/html` components are tested through server-level integration tests rather than standalone stdlib tests.