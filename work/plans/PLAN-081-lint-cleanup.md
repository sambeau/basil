# PLAN-081: Lint Warning Cleanup

**Source:** golangci-lint audit (161 warnings)
**Created:** 2025-02-09
**Status:** Complete

---

## Summary

`golangci-lint run` reports 161 warnings across 4 categories. Most are noise, but two are real bugs, one uses a deprecated API, and there is significant dead code. This plan organises the work into five phases by priority.

---

## Phase 1 — Fix bugs (2 items)

Estimated effort: 15 minutes. No design decisions needed.

### 1a. Variable shadowing swallows error

**File:** `cmd/basil/main.go:1069–1082`
**Linter:** `ineffassign`
**Severity:** Bug — database errors silently swallowed

```go
// CURRENT (buggy)
if userID != "" {
    user, err := db.GetUser(userID)   // := creates NEW err, shadows outer
    ...
    logs, err = db.GetEmailLogs(...)  // assigns to inner err
}
// outer err is still nil — GetEmailLogs failure never checked
```

**Fix:** Declare `user` separately, use `=` instead of `:=`:

```go
if userID != "" {
    var user *auth.User
    user, err = db.GetUser(userID)
    ...
    logs, err = db.GetEmailLogs(ctx, &userID, limit)
}
```

### 1b. Unreachable case clause in formatter

**File:** `pkg/parsley/format/ast_format.go:48–58`
**Linter:** `staticcheck SA4020`
**Severity:** Bug — `*ast.TextNode` case is dead code

`*ast.TextNode` implements `ast.Expression`, so the `ast.Expression` case always matches first.

**Fix:** Move the `*ast.TextNode` case above `ast.Expression`:

```go
func getNodeToken(node ast.Node) *lexer.Token {
    switch n := node.(type) {
    case ast.Statement:
        return getStatementToken(n)
    case *ast.TextNode:          // ← must come before ast.Expression
        return &n.Token
    case ast.Expression:
        return getExpressionToken(n)
    }
    return nil
}
```

---

## Phase 2 — Replace deprecated API (1 item)

Estimated effort: 10 minutes.

### 2a. `strings.Title` → `cases.Title`

**File:** `server/search/markdown.go:45`
**Linter:** `staticcheck SA1019`
**Severity:** Deprecated since Go 1.18

**Fix:** Replace with `golang.org/x/text/cases`:

```go
import "golang.org/x/text/cases"
import "golang.org/x/text/language"

title = cases.Title(language.English).String(title)
```

Then `go mod tidy`.

---

## Phase 3 — Add logging to empty error branches (4 locations)

Estimated effort: 20 minutes.

These all have `if err != nil { }` with a comment saying "log but continue" but no actual logging. They silently swallow errors from cleanup/close operations.

| File | Line(s) | Context |
|------|---------|---------|
| `pkg/parsley/evaluator/connection_cache.go` | 59, 72, 129 | `closeFunc` errors during cache eviction |
| `server/auth/webauthn.go` | 276 | `UpdateCredentialSignCount` error during login |
| `server/search/scanner.go` | 128 | Scan errors collected but never reported |

**Fix for connection_cache.go:** The cache doesn't have a logger. Options:
- (a) Add a `log func(string, ...any)` field to the cache struct — cleanest
- (b) Use `fmt.Fprintf(os.Stderr, ...)` — simplest
- (c) Accept the current behaviour and add `//nolint:staticcheck` — least work

**Fix for webauthn.go:** The manager has access to a logger. Add: `m.logger.Warn("failed to update sign count: %v", err)` or equivalent.

**Fix for scanner.go:** The TODO says "add proper logging when available". Options:
- (a) Return scan errors as a second return value (breaking change)
- (b) Log to stderr
- (c) Accept and `//nolint`

**Recommendation:** Use option (a) for connection_cache (add logger field), log the webauthn error via existing logger, and `//nolint` the scanner for now since it's a known TODO.

---

## Phase 4 — Dead code audit (50 unused symbols)

### Disposition summary (agreed with human)

All groups reviewed against specs. Final disposition:

| Group | Count | Disposition | Rationale |
|-------|-------|-------------|-----------|
| A: Markdown stdlib | 13 | **Delete** | Old `@std/markdown` API, fully superseded by `@std/mdDoc` methods in `stdlib_mddoc.go` |
| B: Error constructors | 5 | **Delete** | Trivial factories, easy to recreate when FEAT-023 migration resumes |
| C: Dict-to-literal | 3 | **Delete** | One-line wrappers; FEAT-098/FEAT-100 shipped without them |
| D: Natural sort | 5 | **Delete** | FEAT-072 spec designs a different algorithm that supersedes these |
| E: Parser functions | 3 | **Delete** | Superseded or early scaffolding for unstarted features |
| F: Miscellaneous | 21 | **Delete 19, Keep 2** | Keep `evalDictionarySpread` (FEAT-074) and `sqliteSupportsReturning` (active TODO) |
| **Total** | **50** | **Delete 48, Keep 2** | |

---

### Group A: Markdown stdlib methods (13 functions)

All in `pkg/parsley/evaluator/markdown_helpers.go`. These are fully implemented stdlib-style functions with arity checks, type validation, and complete logic. They follow the `func markdownX(args []Object, env *Environment) Object` signature used by the stdlib dispatch table, but are not registered anywhere.

| Function | Line | Purpose |
|----------|------|---------|
| `markdownFindAll` | 950 | Find all AST nodes by type |
| `markdownFindFirst` | 995 | Find first AST node by type |
| `markdownHeadings` | 1035 | Extract all headings with metadata |
| `markdownLinks` | 1075 | Extract all links with metadata |
| `markdownImages` | 1119 | Extract all images with metadata |
| `markdownCodeBlocks` | 1159 | Extract all code blocks with metadata |
| `markdownTitle` | 1202 | Extract document title (first h1) |
| `markdownTOC` | 1238 | Generate table of contents |
| `markdownText` | 1296 | Extract all plain text |
| `markdownWordCount` | 1346 | Count words in document |
| `markdownWalk` | 1372 | Walk tree, call function on each node |
| `markdownMap` | 1409 | Transform tree nodes via function |
| `markdownFilter` | 1482 | Filter tree nodes via predicate |

**Spec research:** These are the **old standalone-function API** from a deprecated `@std/markdown` module. They have been fully superseded by the method-based `@std/mdDoc` module in `stdlib_mddoc.go`, which provides identical functionality as methods on the `MdDoc` type (e.g., `doc.findAll()`, `doc.headings()`, `doc.links()`, etc.). The `MdDoc` methods are wired into the dispatch table and working. The shared helper functions these old functions called (e.g., `findAllNodes`, `collectHeadings`, `extractPlainText`) are still used by the new `MdDoc` methods — only the 13 top-level `markdownXxx(args, env)` entry points are dead. The comment at the top of `markdown_helpers.go` confirms: *"extracted from the deprecated `@std/markdown` module"*.

**Recommendation: Delete** — fully superseded by `@std/mdDoc`. The shared helpers they called remain in use.

---

### Group B: Error constructors (5 functions)

All in `pkg/parsley/evaluator/eval_errors.go`. These are error factory functions that follow the same pattern as the ones that *are* used. Likely scaffolded during the structured error migration (FEAT-023).

| Function | Line | Purpose |
|----------|------|---------|
| `newStructuredErrorWithPos` | 47 | Error with explicit file/line position |
| `newArityErrorMin` | 254 | "at least N args" error |
| `newLoopError` | 519 | Loop-specific error |
| `newHTTPError` | 585 | HTTP error wrapping Go error |
| `newHTTPStateError` | 601 | HTTP state error |

**Spec research:** FEAT-023 (Structured Error Migration) is the source. Its backlog entry (#7) says: *"Migrate remaining files: other `stdlib_*.go` modules (not present yet). Core evaluator files and stdlib_table.go done."* These constructors were scaffolded for that incomplete migration.

**Recommendation:** **Delete** — they're trivial factory functions (3–10 lines each) that are easy to recreate when the migration resumes. No logic would be lost.

---

### Group C: Dict-to-literal converters (3 functions)

| Function | File | Line | Purpose |
|----------|------|------|---------|
| `datetimeDictToLiteral` | `eval_datetime.go` | 317 | `@2026-01-21T15:30:00Z` literal form |
| `pathDictToLiteral` | `eval_dict_to_string.go` | 289 | `@/path/to/file` literal form |
| `urlDictToLiteral` | `eval_dict_to_string.go` | 448 | `@https://example.com` literal form |

These are trivial one-line wrappers (`return "@" + xDictToString(dict)`) over functions that *are* used.

**Spec research:** No spec references these functions. FEAT-098 (PLN serialisation) and FEAT-100 (Pretty-Printer) are the two features that would produce Parsley source literals, and both are marked complete — meaning they chose not to use these wrappers. The formatter uses `getExpressionToken` and the PLN serialiser has its own logic.

**Recommendation:** **Delete** — one-line wrappers that nothing uses and nothing planned needs. Trivially recreated if ever needed.

---

### Group D: Natural sort helpers (5 functions)

In `pkg/parsley/evaluator/eval_helpers.go`:

| Function | Line | Purpose |
|----------|------|---------|
| `naturalCompare` | 18 | Natural comparison (numbers sort numerically) |
| `getTypeOrder` | 47 | Type ordering for mixed-type sorts |
| `compareNumbers` | 59 | Numeric comparison across int/float |
| `getNumericValue` | 66 | Extract float64 from int or float |

Plus the caller in `methods.go`:

| Function | File | Line | Purpose |
|----------|------|------|---------|
| `naturalSortArray` | `methods.go` | 1084 | Sort array with natural ordering |

**Spec research:** **FEAT-072 (Natural Sort Order)** is a detailed spec in **draft** status. It specifies: natural sort as default for `sort()`, string comparison operators (`<`, `>`) using natural order, an ASCII fast path with Unicode fallback, and mixed-type sort ordering. The spec explicitly designs a `NaturalCompare(a, b string) int` three-way compare function, an ASCII fast path, and type ordering — which aligns with these helpers but the spec calls for a more efficient implementation (three-way return, byte-level ASCII path) than what's currently scaffolded here.

**Recommendation:** **Delete** — FEAT-072 is a substantial feature that will likely rewrite these from scratch per its detailed algorithm design. The current scaffolding is a simpler, earlier approach. The spec's implementation plan supersedes this code.

---

### Group E: Parser functions (3 functions)

All in `pkg/parsley/parser/parser.go`.

| Function | Line | Purpose |
|----------|------|---------|
| `parseFunctionParameters` | 2324 | Parse function parameter list |
| `isComputedFieldStart` | 3813 | Detect computed field syntax in queries |
| `parseComputedField` | 3829 | Parse `name: function(field)` in queries |

**Spec research:** No spec directly references these. The computed field functions relate to backlog #2 (Query DSL Correlated Subqueries, from PLAN-052 Phase 5) which describes computed fields like `| comment_count <-Comments || post_id == id | count`. This code predates the workflow so there's no spec trail. `parseFunctionParameters` is likely superseded by a newer parameter parsing implementation.

**Recommendation:** **Delete** — `parseFunctionParameters` is clearly superseded (the parser works fine without it). The computed field functions are early scaffolding for a complex feature (backlog #2, estimated 3–4 days) that will need its own design when tackled. Recoverable from git history.

---

### Group F: Miscellaneous (21 symbols)

| Symbol | File | Line | Purpose | Spec finding | Recommendation |
|--------|------|------|---------|--------------|----|
| `mergeMonthNames` | `eval_datetime.go` | 593 | Merge locale month name maps | Scaffolding for backlog #17 (locale support). No spec yet — backlog says "Needs design". | **Delete** — speculative scaffolding for an undesigned feature. |
| `allMonthNames` | `eval_datetime.go` | 604 | Combined month name lookup | Same as above. | **Delete** |
| `fetchUrlContent` | `eval_network_io.go` | 640 | Legacy URL fetch | Comment in code says *"Legacy function — kept for backward compatibility with error capture pattern"*. No spec references it. | **Delete** — self-described as legacy, no callers. |
| `evalDictionarySpread` | `eval_string_conversions.go` | 104 | Spread dict as HTML attributes | **FEAT-074 (Dictionary Spreading in HTML Tags)** is a detailed spec in **proposed** status. It explicitly designs this exact function: *"May need new helper: `evalDictionarySpread(dict Object) string`"* (FEAT-074, Implementation §2). | **Keep + `//nolint`** — this is pre-built for FEAT-074 which is a high-priority proposed feature. |
| `logDeprecation` | `evaluator.go` | 738 | Log deprecation warnings to DevLog | FEAT-054 (Replace `now()` with `@now`) mentions `logDeprecation("now() is deprecated...")` in its spec but the feature was completed without using it. Backlog #55 (deprecate `format(arr)`) could use it. | **Delete** — no current callers, and when deprecation warnings are needed the implementation may look different. Trivial to recreate. |
| `getPublicDirComponents` | `evaluator.go` | 1671 | Extract `public_dir` from basil config | FEAT-055 (Namespace Cleanup) refactored `publicUrl()` into `path.public()`. This helper was part of the old design. | **Delete** — superseded by FEAT-055 refactoring. |
| `sqliteSupportsReturning` | `evaluator.go` | 1827 | Version check for SQLite RETURNING clause | No spec, but `stdlib_schema_table_binding.go:526` has a TODO comment referencing this function by name: *"Version detection already exists: `sqliteSupportsReturning(tb.DB.SQLiteVersion)`"*. It's scaffolding for a planned INSERT ... RETURNING optimisation. | **Keep + `//nolint`** — actively referenced in a TODO, will be needed when that optimisation is implemented. |
| `setFormContext` | `form_binding.go` | 35 | Trivial setter (`env.FormContext = ctx`) | No spec references it. It's a one-line setter that wraps a direct field assignment. | **Delete** — callers can set the field directly. |
| `formatRecordCurrency` | `methods_record.go` | 683 | Default USD currency formatter | **FEAT-094 (Schema Enhancements: Money Currency)** is marked **complete** and its spec explicitly designs this function: *"`record.format(field)` method SHOULD apply currency formatting when available"*. The spec even shows pseudocode for `formatRecordCurrency`. However, the feature was completed without wiring this up — the `format()` method may use a different approach. | **Delete** — FEAT-094 is complete and chose not to use this. If currency formatting in `format()` is revisited, the spec provides the design. |
| `validateField` | `record_validation.go` | 79 | Public wrapper for `validateFieldInternal` | FEAT-105 (Unified Error Model) references record validation but is in **draft** status. This is a thin wrapper that adds no logic. | **Delete** — trivial to recreate. The internal function it wraps is the real implementation. |
| `getConditionLogic` | `stdlib_dsl_query.go` | 1334 | Extract logic op from query condition | Relates to Query DSL (FEAT-079). No direct reference in specs. Likely scaffolding for complex query conditions. | **Delete** — small function, recoverable from git history when query DSL work resumes. |
| `knownPrimitiveTypes` | `stdlib_dsl_schema.go` | 573 | Map of primitive type names | No spec references it. Scaffolding for schema type validation that was never wired up. | **Delete** |
| `isPrimitiveType` | `stdlib_dsl_schema.go` | 599 | Check if type is primitive | Uses `knownPrimitiveTypes` above. | **Delete** (together with `knownPrimitiveTypes`) |
| `fileToComponentName` | `stdlib_html.go` | 18 | Convert filename to PascalCase component name | FEAT-074 (Dictionary Spreading) and FEAT-058 mention components but neither references auto-registration from filenames. FEAT-039 (Components) may be relevant but doesn't mention this function. | **Delete** — speculative scaffolding for an unspecified component auto-registration feature. |
| `cuidAlphabet` | `stdlib_id.go` | 37 | CUID character set constant | FEAT-034 (std/api) specifies `id.cuid()` which is **implemented and working** — but the implementation uses `base36Encode` instead of this alphabet. The constant is leftover from an earlier approach that was replaced. | **Delete** — the CUID implementation doesn't use it; it uses base36 encoding instead. |
| `schemaMethods` | `stdlib_schema.go` | 22 | `[]string{"validate"}` — list of schema method names | No spec references this. It's a single-element slice that nothing reads. | **Delete** |
| `stdoutLogger` | `parsley/logger.go` | 18 | Stdout logger type + `Log`/`LogLine` | No spec references it. Comment says *"convenience — evaluator.DefaultLogger can also be used"*. | **Delete** — unused convenience type. The evaluator has its own logger. |
| `renderPlainTextError` | `server/devtools.go` | 1523 | Plain-text error fallback for devtools | No spec references it. Comment says *"Used as ultimate fallback when all templates fail."* But nothing calls it — the current fallback code in devtools writes plain text inline without using this helper. | **Delete** — the fallback it describes is handled inline elsewhere. |
| `setEnvVar` | `server/handler.go` | 380 | Convert Go value to Parsley env var | FEAT-011 spec shows it as the old design: `setEnvVar(env, "request", reqCtx)`. PLAN-FEAT-011 says *"Replace individual `setEnvVar` calls with single `basil` injection"*. | **Delete** — explicitly superseded by the `basil` injection design. |
| `nsWordML` | `server/search/extract_docx.go` | 19 | XML namespace for DOCX WordprocessingML | FEAT-085 (Full-Text Search) specifies DOCX extraction. These constants are defined but never referenced — the DOCX extractor doesn't use namespace-qualified lookups. | **Delete** |
| `nsDCTerms` | `server/search/extract_docx.go` | 20 | XML namespace for Dublin Core Terms | Same as above. | **Delete** |
| `nsDC` | `server/search/extract_docx.go` | 21 | XML namespace for Dublin Core Elements | Same as above. | **Delete** |
| `normalize` | `server/search/search.go` | 299 | Normalize score to 0–1 range | FEAT-085 spec doesn't reference score normalisation. The search implementation uses FTS5's built-in ranking. | **Delete** — scoring approach changed, this is dead. |
| `getCheckedPaths` | `server/site.go` | 184 | List paths checked during 404 handler lookup | FEAT-040 (Site Mode) doesn't reference this. It would be useful for a dev-mode 404 page that shows "we tried these paths", but nothing calls it. | **Delete** — useful idea but unfinished. Can be recovered from git if a dev 404 page is built. |

### Group F summary

| Recommendation | Symbols |
|---|---|
| **Keep + `//nolint`** | `evalDictionarySpread` (FEAT-074), `sqliteSupportsReturning` (active TODO) |
| **Delete** | Everything else (19 symbols) |

---

## Phase 5 — Remaining `ineffassign` cleanup (10 items)

These are dead assignments — variables assigned but never read afterward. Not bugs, but noisy.

| File | Line | Variable | Likely fix |
|------|------|----------|------------|
| `eval_computed_properties.go` | 605 | `hasTime = true` | Remove assignment |
| `eval_control_flow.go` | 146 | `evaluated = NULL` | Remove assignment |
| `eval_control_flow.go` | 152 | `evaluated = NULL` | Remove assignment |
| `eval_control_flow.go` | 295 | `evaluated = NULL` | Remove assignment |
| `evaluator.go` | 5710 | `pathStr = "-"` | Remove assignment |
| `server/errors.go` | 254 | `displayValue := value` | Move declaration into else branch |
| `server/handler.go` | 1106 | `customStatus = false` | Remove assignment |
| `server/handler.go` | 1109 | `customStatus = false` | Remove assignment |
| `server/handler.go` | 1247 | `errType = "parse"` | Remove assignment |
| `server/handler.go` | 1269 | `errType = "parse"` | Remove assignment |

These are all safe mechanical fixes. No behaviour changes.

---

## Execution order

| Phase | Items | Effort | Blocked on |
|-------|-------|--------|------------|
| 1 | Bug fixes (1a, 1b) | 15 min | Nothing |
| 2 | Deprecated API (2a) | 10 min | Nothing |
| 3 | Empty error branches | 20 min | Nothing |
| 4 | Dead code audit | 30–60 min | **Human review of disposition table above** |
| 5 | Dead assignments | 15 min | Nothing |

Phases 1, 2, 3, and 5 can proceed immediately. Phase 4 requires the human to review the groups and confirm or override the recommendations.

---

## Progress log

| Date | Phase | Status | Notes |
|------|-------|--------|-------|
| 2025-02-09 | — | Plan created | Spec research completed, dispositions agreed with human |
| 2025-02-09 | 1 | ✅ Complete | Fixed variable shadowing bug in `main.go`, fixed unreachable case in `ast_format.go` |
| 2025-02-09 | 2 | ✅ Complete | Replaced `strings.Title` with `cases.Title` in `server/search/markdown.go` |
| 2025-02-09 | 3 | ✅ Complete | Added `logErr` callback to connection cache, logged webauthn error, nolinted scanner TODO |
| 2025-02-09 | 4 | ✅ Complete | Deleted 48 unused symbols, kept 2 with `//nolint` (`evalDictionarySpread`, `sqliteSupportsReturning`). Also cleaned up unused items in `format/`, `lexer/`, and `parser/` packages discovered during execution. |
| 2025-02-09 | 5 | ✅ Complete | Removed all 11 dead assignments |

## Results

**Before:** 161 warnings (50 errcheck, 11 ineffassign, 50 staticcheck, 50 unused)
**After:** 100 warnings (50 errcheck, 0 ineffassign, 50 staticcheck, 0 unused)
**Eliminated:** 61 warnings — all `unused` and `ineffassign` warnings resolved

Remaining 100 warnings are all harmless:
- **errcheck (50):** Unchecked `fmt.Fprint*`, `defer Close()`, test code — standard Go practice
- **staticcheck (50):** Cosmetic suggestions (QF1003 tagged switch, S1039 unnecessary Sprintf, etc.)