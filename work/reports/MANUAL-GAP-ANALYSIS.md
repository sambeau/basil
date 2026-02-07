# Parsley Manual Gap Analysis

**Date:** 2026-01-30
**Author:** Copilot
**Purpose:** Identify missing manual pages, assess existing page quality, and propose a plan for completing the Parsley programming language manual.

---

## Executive Summary

The Parsley manual currently has **12 pages** across three categories (builtins, stdlib, features). Based on a thorough analysis of the lexer, parser, and evaluator source code, the language has approximately **45+ documentable topics**. This means the manual is roughly **25% complete**, with critical gaps in language fundamentals that would prevent a new user from learning Parsley without also reading the monolithic `reference.md`.

The existing pages are generally **high quality**—well-structured with frontmatter, practical examples, and result annotations. The main gap is coverage, not quality.

### Inspiration from Great Language Manuals

The best programming language manuals share common structural patterns:

| Language | Strength | Lesson for Parsley |
|----------|----------|--------------------|
| **Rust Book** | Progressive tutorial + reference hybrid | Consider a "getting started" tutorial track alongside the reference manual |
| **Python Docs** | Clear separation: Tutorial → Library Reference → Language Reference | Parsley should separate "how to learn" from "how to look things up" |
| **Go Tour / Spec** | Minimal, precise, runnable examples | Parsley's example-driven style already follows this well |
| **Elixir Guides** | Topic-based guides that flow naturally | Group pages into logical chapters, not just alphabetical lists |
| **Swift Book** | Each concept builds on the previous | Define a reading order for fundamentals |

---

## Current Manual Inventory

### Builtins (9 pages) — `manual/builtins/`

| Page | File | Quality | Notes |
|------|------|---------|-------|
| Arrays | `array.md` | ⭐⭐⭐⭐⭐ | Excellent. Comprehensive operators, all methods with examples, practical recipes. Gold standard for other pages. |
| DateTime | `datetime.md` | ⭐⭐⭐⭐ | Good. Covers literals, properties, methods. Could add more arithmetic examples. |
| Dictionary | `dictionary.md` | ⭐⭐⭐⭐ | Good. Covers literals, access patterns, methods. Well structured. |
| Duration | `duration.md` | ⭐⭐⭐⭐ | Good. Covers literals, units, component storage model. |
| Money | `money.md` | ⭐⭐⭐⭐ | Good. Covers symbol/code syntax, exact arithmetic, currency codes. |
| Numbers | `numbers.md` | ⭐⭐⭐⭐ | Good. Covers integer/float literals, arithmetic, methods. |
| Record | `record.md` | ⭐⭐⭐⭐ | Good. Explains Record = Schema + Data + Errors model clearly. |
| Schema | `schema.md` | ⭐⭐⭐⭐ | Good. Field types, constraints, metadata, UI generation. |
| Table | `table.md` | ⭐⭐⭐⭐ | Good. SQL-like operations, method chaining, typed tables. |

### Standard Library (2 pages) — `manual/stdlib/`

| Page | File | Quality | Notes |
|------|------|---------|-------|
| @std/html | `html.md` | ⭐⭐⭐⭐ | Good. Component philosophy, accessible HTML generation. |
| @std/schema | `schema.md` | ⭐⭐⭐ | Adequate. Covers define/validate pattern. Could expand constraint docs. |

### Feature Pages (1 page) — `manual/`

| Page | File | Quality | Notes |
|------|------|---------|-------|
| PLN | `pln.md` | ⭐⭐⭐⭐⭐ | Excellent. Thorough coverage of serialize/deserialize, syntax, security model, use cases. |

---

## Gap Analysis: Missing Pages

### 🔴 Priority 1 — Language Fundamentals (Critical)

These pages are essential for anyone learning Parsley. Without them, users must read the 4000+ line `reference.md` monolith.

| # | Topic | Proposed File | Source Files | Scope |
|---|-------|---------------|-------------|-------|
| 1 | **Strings** | `builtins/strings.md` | `eval_infix.go`, `methods.go` (L117-436), `evaluator.go` (templates) | Three string types (`"..."`, `` `...` ``, `'...'`), interpolation (`{var}` and `@{var}`), escape sequences, 27 string methods, string operators (+, *, in) |
| 2 | **Variables & Binding** | `fundamentals/variables.md` | `parser.go` (parseLetStatement, parseAssignmentStatement), `evaluator.go` (Environment) | `let` binding, assignment, `let` vs bare assignment, scope rules, destructuring (array `[a, b]` and dict `{a, b}`), `_` discard |
| 3 | **Functions** | `fundamentals/functions.md` | `parser.go` (parseFunctionLiteral, parseFunctionParameters), `eval_expressions.go` | `fn` syntax, parameters, default values, closures, `this` binding, rest params (`...args`), return values, functions as values |
| 4 | **Control Flow** | `fundamentals/control-flow.md` | `eval_control_flow.go`, `eval_infix.go` (evalIfExpression), `parser.go` (parseForExpression, parseIfExpression) | `if`/`else`/`else if`, `for` loops (map/filter pattern), `check` guards, `stop`/`skip`, expression-based design, range (`..`) iteration |
| 5 | **Operators** | `fundamentals/operators.md` | `eval_operators.go`, `eval_infix.go`, `eval_collections.go`, `lexer.go` | Full operator reference: arithmetic, comparison, logical (`&`/`and`, `\|`/`or`), set operations, membership (`in`, `not in`), schema checking (`is`, `is not`), pattern matching (`~`, `!~`), range (`..`), concatenation (`++`), null coalescing (`??`), optional access (`?`), spread (`...`), precedence table |
| 6 | **Booleans & Null** | `builtins/booleans.md` | `evaluator.go` (Boolean, Null types), `eval_infix.go` | `true`/`false`/`null` literals, truthiness rules (what is falsy: `false`, `null`, `0`, `0.0`, `""`), boolean methods, null-safe patterns |
| 7 | **Error Handling** | `fundamentals/errors.md` | `eval_control_flow.go` (evalTryExpression), `evaluator.go` (Error type), `eval_errors.go` | `try` expression, `fail()` function, error result dictionaries `{result, error}`, catchable vs non-catchable error classes, `check` guards as error prevention, optional access `[?n]` |

### 🟡 Priority 2 — Important Features

These cover key Parsley features that differentiate it from other languages.

| # | Topic | Proposed File | Source Files | Scope |
|---|-------|---------------|-------------|-------|
| 8 | **Modules** | `fundamentals/modules.md` | `eval_expressions.go` (evalImportExpression, importModule), `parser.go` (parseExportStatement) | `import` (relative paths, stdlib paths), `export`, computed exports, destructuring imports `let {a, b} = import(...)`, module caching |
| 9 | **Tags (HTML/XML)** | `fundamentals/tags.md` | `eval_tags.go`, `parser.go` (parseTagLiteral, parseTagPair, parseTagContents) | Self-closing tags (`<br/>`), pair tags (`<div>...</div>`), attributes (string, expression, spread), dynamic content, components, void elements, `<SQL>` tags, `<Cache>` and `<Part>` special tags, form binding (`@record`, `@field`) |
| 10 | **File I/O** | `fundamentals/file-io.md` | `eval_file_io.go`, `evaluator.go` (file handle builtins) | Read operator (`<==`), write operator (`==>`), append operator (`==>>`), fetch operator (`<=/=`), file handle factories (`file`, `JSON`, `YAML`, `CSV`, `PLN`, `SVG`, `MD`, `text`, `bytes`, `lines`), `dir()`, `fileList()`, format auto-detection |
| 11 | **Regex** | `builtins/regex.md` | `eval_regex.go`, `methods.go` (evalRegexMethod) | Regex literals (`/pattern/flags`), `regex()` builtin, match operator (`~`), not-match (`!~`), flags (i, m, s, g), regex properties (pattern, flags), regex methods (test, match, matchAll, replace, split), string regex methods |
| 12 | **Paths** | `builtins/paths.md` | `eval_paths.go`, `eval_computed_properties.go` (evalPathComputedProperty), `methods.go` (evalPathMethod) | Path literals (`@/...`, `@./...`), interpolated paths (`@(./path/{var}/file)`), path properties (segments, name, ext, stem, dir, absolute), path methods (join, parent, resolve, withExt, withName), path arithmetic (`+` for joining) |
| 13 | **URLs** | `builtins/urls.md` | `eval_urls.go`, `eval_computed_properties.go` (evalUrlComputedProperty), `methods.go` (evalUrlMethod) | URL literals (`@https://...`), interpolated URLs (`@(https://api.com/{id}/data)`), URL properties (scheme, host, port, path, query, fragment, origin), URL methods (withPath, withQuery, withFragment, resolve), `url()` builtin |
| 14 | **Comments** | `fundamentals/comments.md` | `lexer.go` | Single-line comments (`//`), inline comments, no multi-line comments (common gotcha), leading/trailing comment preservation |

### 🟢 Priority 3 — Database & I/O

| # | Topic | Proposed File | Source Files | Scope |
|---|-------|---------------|-------------|-------|
| 15 | **Database** | `features/database.md` | `eval_database.go`, `evaluator.go` (connectionBuiltins) | Connection literals (`@sqlite`, `@postgres`, `@mysql`), query one (`<=?=>`), query many (`<=??=>`), execute (`<=!=>`), transactions (`@transaction`), parameterized queries, connection methods |
| 16 | **Query DSL** | `features/query-dsl.md` | `parser.go` (parseQueryExpression, parseInsertExpression, etc.), `stdlib_dsl_query.go`, `stdlib_dsl_schema.go` | `@query`, `@insert`, `@update`, `@delete` expressions, where conditions, order/limit/offset, computed fields, relations, subqueries, pipe operators |
| 17 | **HTTP & Networking** | `features/network.md` | `eval_network_io.go` | Fetch operator (`<=/=`), HTTP methods (GET, POST, PUT, DELETE), response dictionaries, URL-based file handles, SFTP connections and operations |
| 18 | **Shell Commands** | `features/commands.md` | `evaluator.go` (createCommandHandle, executeCommand) | `@shell` literal, execute operator (`<=#=>`), command options, result dictionaries (stdout, stderr, exitCode) |

### 🔵 Priority 4 — Standard Library (Missing Pages)

| # | Topic | Proposed File | Source Files | Scope |
|---|-------|---------------|-------------|-------|
| 19 | **@std/math** | `stdlib/math.md` | `stdlib_math.go` | Constants (PI, E, etc.), rounding (floor, ceil, round, trunc), comparison (abs, sign, clamp, min, max), aggregation (sum, avg, product, count), statistics (median, mode, stddev, variance, range), random (random, randomInt, seed), powers/logs (sqrt, pow, exp, log, log10), trig, geometry (hypot, dist, lerp, map) |
| 20 | **@std/valid** | `stdlib/valid.md` | `stdlib_valid.go` | Type validators, string validators, number validators, format validators, collection validators |
| 21 | **@std/id** | `stdlib/id.md` | `stdlib_id.go` | ID generation (UUID, nanoid, etc.) |
| 22 | **@std/table** | `stdlib/table.md` | `stdlib_table.go` | Table constructors, query methods, aggregation, access, mutation, export |
| 23 | **@std/api** | `stdlib/api.md` | `stdlib_api.go` | Auth wrappers, error helpers, redirect helpers |
| 24 | **@std/mdDoc** | `stdlib/mddoc.md` | `stdlib_mddoc.go` | Markdown document processing, rendering, querying, AST access |
| 25 | **@std/dev** | `stdlib/dev.md` | `stdlib_dev.go` | Development/debugging utilities |
| 26 | **@std/session** | `stdlib/session.md` | `stdlib_session.go` | Session management (get, set, delete, has, clear, all), flash messages (flash, getFlash, getAllFlash, hasFlash), session regeneration |

### ⚪ Priority 5 — Meta & Navigation Pages

| # | Topic | Proposed File | Notes |
|---|-------|---------------|-------|
| 27 | **Manual Index** | `index.md` | Table of contents with logical chapter ordering, brief description of each page |
| 28 | **Getting Started** | `getting-started.md` | Tutorial-style introduction: first program, variables, functions, tags, a simple web page |
| 29 | **Type System Overview** | `fundamentals/types.md` | Overview of all types, type relationships (Dictionary → Record, Array → Table), type checking patterns, truthiness |
| 30 | **Security Model** | `features/security.md` | Link to/integrate existing `docs/parsley/security.md`; file path restrictions, SQL injection prevention, command execution security |

---

## Detailed Quality Notes on Existing Pages

### Strengths (Consistent Across Pages)

- **Frontmatter metadata** — All pages have proper YAML frontmatter with id, title, system, type, keywords
- **Example-driven** — Code examples with explicit `**Result:**` annotations
- **Practical recipes** — Pages end with real-world usage examples (e.g., array.md's card dealing, movie sorting)
- **Method documentation** — Methods are listed alphabetically with signatures and multiple examples
- **Consistent voice** — Clear, direct, no unnecessary jargon

### Areas for Improvement

| Page | Issue | Recommendation |
|------|-------|----------------|
| **array.md** | Some examples have live-rendered expressions (` ⚡️<code>@{repr(...)}</code>`) that only work inside Basil | Add static fallback results for standalone reading |
| **array.md** | Typo: "buty" should be "but" in map section | Fix typo |
| **array.md** | Natural sort example has wrong expected result (`1 apple` vs `9 apple`) | Verify and fix expected results |
| **datetime.md** | Missing cross-references to Duration for arithmetic | Add "See Also" section |
| **dictionary.md** | No mention of computed properties or `this` binding | Add section on computed values |
| **schema.md** | Missing `@table` / TableBinding relationship | Add cross-reference to table.md |
| **record.md** | Form binding context (`@record`, `@field`) is briefly mentioned but not deep enough | Expand or cross-reference to tags page |
| **All pages** | No "See Also" section linking to related pages | Add cross-references to build a connected manual |
| **All pages** | No consistent navigation (prev/next page) | Add when index.md is created |

---

## Proposed Manual Structure

Based on analysis of successful language manuals, here's a recommended organization:

```
manual/
├── index.md                          # Table of contents + reading guide
├── getting-started.md                # Tutorial: first Parsley program
│
├── fundamentals/                     # Language core (read in order)
│   ├── variables.md                  # let, assignment, scope, destructuring
│   ├── types.md                      # Type system overview
│   ├── operators.md                  # All operators + precedence
│   ├── control-flow.md              # if/else, for, check, stop, skip
│   ├── functions.md                  # fn, closures, default params
│   ├── modules.md                    # import, export
│   ├── tags.md                       # HTML/XML tag system
│   ├── errors.md                     # try, fail, error handling
│   └── comments.md                   # Comment syntax
│
├── builtins/                         # Type reference (alphabetical)
│   ├── array.md                      ✅ EXISTS
│   ├── booleans.md                   # true, false, null, truthiness
│   ├── datetime.md                   ✅ EXISTS
│   ├── dictionary.md                 ✅ EXISTS
│   ├── duration.md                   ✅ EXISTS
│   ├── money.md                      ✅ EXISTS
│   ├── numbers.md                    ✅ EXISTS
│   ├── paths.md                      # Path literals + methods
│   ├── record.md                     ✅ EXISTS
│   ├── regex.md                      # Regex literals + methods
│   ├── schema.md                     ✅ EXISTS
│   ├── strings.md                    # Three string types + methods
│   ├── table.md                      ✅ EXISTS
│   └── urls.md                       # URL literals + methods
│
├── features/                         # Domain-specific features
│   ├── database.md                   # Connections, queries, transactions
│   ├── query-dsl.md                  # @query, @insert, @update, @delete
│   ├── file-io.md                    # File read/write operators
│   ├── network.md                    # HTTP fetch, SFTP
│   ├── commands.md                   # Shell execution
│   └── security.md                   # Security model
│
├── stdlib/                           # Standard library reference
│   ├── api.md                        # @std/api
│   ├── dev.md                        # @std/dev
│   ├── html.md                       ✅ EXISTS
│   ├── id.md                         # @std/id
│   ├── math.md                       # @std/math
│   ├── mddoc.md                      # @std/mdDoc
│   ├── schema.md                     ✅ EXISTS
│   ├── session.md                    # @std/session
│   ├── table.md                      # @std/table
│   └── valid.md                      # @std/valid
│
└── pln.md                            ✅ EXISTS
```

---

## Implementation Recommendations

### Phase 1: Language Fundamentals (Unblocks Learning)
**Estimated pages: 9 | Recommended order:**

1. `fundamentals/comments.md` — Smallest page, quick win (half a page)
2. `builtins/booleans.md` — Small but essential (truthiness rules are a major gotcha)
3. `builtins/strings.md` — Large page (27 methods) but strings are the most-used type
4. `fundamentals/variables.md` — Core concept, builds on types
5. `fundamentals/operators.md` — Large reference page, unlocks understanding of expressions
6. `fundamentals/functions.md` — Core concept, needed for control flow examples
7. `fundamentals/control-flow.md` — for/if/check, depends on functions
8. `fundamentals/errors.md` — try/fail, depends on functions + control flow
9. `index.md` — Create the table of contents once fundamentals exist

### Phase 2: Key Features (Differentiators)
**Estimated pages: 6**

10. `fundamentals/tags.md` — Huge differentiator for Parsley as a web language
11. `fundamentals/modules.md` — import/export
12. `builtins/regex.md` — Pattern matching
13. `builtins/paths.md` — Path type
14. `builtins/urls.md` — URL type
15. `fundamentals/types.md` — Type system overview connecting all the pieces

### Phase 3: I/O & Database
**Estimated pages: 5**

16. `features/file-io.md`
17. `features/database.md`
18. `features/query-dsl.md`
19. `features/network.md`
20. `features/commands.md`

### Phase 4: Standard Library
**Estimated pages: 8**

21-28. Remaining stdlib pages (`math`, `valid`, `id`, `table`, `api`, `mddoc`, `dev`, `session`)

### Phase 5: Polish
**Estimated pages: 2 + edits**

29. `getting-started.md` — Tutorial (best written after all reference pages exist)
30. `features/security.md`
31. Cross-reference audit: add "See Also" sections to all pages
32. Fix identified issues in existing pages

---

## Source of Truth Mapping

For each missing page, here's where to find the authoritative implementation:

| Page | Primary Source | Secondary Source | Reference Section |
|------|---------------|-----------------|-------------------|
| Strings | `methods.go` L117-436 | `evaluator.go` (template eval) | reference.md §1.2, §5.1 |
| Variables | `parser.go` (parseLetStatement L493-663) | `evaluator.go` (Environment L641-905) | reference.md §4.1-4.2 |
| Functions | `parser.go` (parseFunctionLiteral L2224-2308) | `eval_expressions.go` (applyFunction) | reference.md §1.6 |
| Control Flow | `eval_control_flow.go` (entire file) | `parser.go` (parseIfExpression, parseForExpression) | reference.md §3 |
| Operators | `eval_operators.go` + `eval_infix.go` + `eval_collections.go` | `lexer.go` (token types L60-110) | reference.md §2 |
| Booleans & Null | `evaluator.go` (Boolean L165-170, Null L181-184) | `eval_infix.go` (evalBangOperatorExpression) | reference.md §1.3 |
| Errors | `eval_control_flow.go` (evalTryExpression) | `eval_errors.go`, `evaluator.go` (Error L224-254) | reference.md §10 |
| Modules | `eval_expressions.go` (evalImportExpression L189-231, importModule L235-395) | `parser.go` (parseExportStatement L373-444) | reference.md §4.4-4.5 |
| Tags | `eval_tags.go` (entire file, 2586 lines) | `parser.go` (parseTagLiteral L1292-1748) | reference.md §8 |
| File I/O | `eval_file_io.go` | `evaluator.go` (file handle builtins) | reference.md §6.12 |
| Regex | `eval_regex.go` | `methods.go` (evalRegexMethod L2484-2637) | reference.md §1.10, §5.9 |
| Paths | `eval_paths.go` | `eval_computed_properties.go` L14-232, `methods.go` L2213-2338 | reference.md §1.11, §5.7 |
| URLs | `eval_urls.go` | `eval_computed_properties.go` L423-526, `methods.go` L2345-2477 | reference.md §1.12, §5.8 |
| Database | `eval_database.go` | `evaluator.go` (connectionBuiltins L1852-2362) | reference.md (scattered) |
| Query DSL | `stdlib_dsl_query.go` + `stdlib_dsl_schema.go` | `parser.go` (parseQueryExpression L3561-4603) | reference.md §7.4 |
| Network | `eval_network_io.go` | — | reference.md §6.12 |
| Commands | `evaluator.go` (createCommandHandle L4000-4048, executeCommand L4123-4209) | — | — |

---

## Summary Statistics

| Category | Existing | Missing | Total | Coverage |
|----------|----------|---------|-------|----------|
| Fundamentals | 0 | 9 | 9 | 0% |
| Builtins | 9 | 5 | 14 | 64% |
| Features | 1 | 6 | 7 | 14% |
| Stdlib | 2 | 8 | 10 | 20% |
| Meta/Navigation | 0 | 2 | 2 | 0% |
| **Total** | **12** | **30** | **42** | **29%** |

The biggest gap is **fundamentals** — the pages someone needs to read first to learn the language don't exist yet.