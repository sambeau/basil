---
id: PLAN-092
feature: FEAT-114
title: "Tree-sitter Grammar Phase 2: External Scanner and Remaining Work"
status: complete
created: 2026-02-13
---

# Implementation Plan: FEAT-114 Phase 2

## Overview

Phase 2 addresses the remaining gaps in the Parsley tree-sitter grammar that require context-sensitive tokenization beyond what tree-sitter's pure JS grammar can express. The central piece is a C external scanner that handles tag disambiguation, raw text tags, and regex/division ambiguity. This plan also covers the query DSL sub-grammar, language injection queries, and Zed extension query files.

Phase 1 (PLAN-091) delivered a near 1:1 structural translation of the Go parser with 210/210 corpus tests passing. However, most real handler files (which use multiple sibling tags) produce parse errors without the external scanner.

## Prerequisites

- [x] Phase 1 grammar complete (PLAN-091 Tasks 1–11 done)
- [x] Phase 1 corpus tests passing (210/210)
- [ ] Node names reviewed and stabilized (before Task 6)
- [ ] tree-sitter CLI installed (`tree-sitter generate`, `tree-sitter test`)

## Source of Truth

| File | Role |
|------|------|
| `pkg/parsley/lexer/lexer.go` | Tag depth tracking, raw text mode, `shouldTreatAsRegex`, `nextTagContentToken` |
| `pkg/parsley/parser/parser.go` | `parseTagPair`, `parseTagContents`, `parseQueryExpression` |
| `contrib/tree-sitter-parsley/grammar.js` | Current Phase 1 grammar |

Key Go lexer functions to mirror:
- `nextTagContentToken` (L1926–2117) — tag content mode, raw text mode, `@{}` interpolation
- `shouldTreatAsRegex` (L2879–2896) — regex vs division context
- `readRawTagText` (L2125–2146) — raw text reading for style/script
- `parseQueryExpression` (L3611–3764) — query DSL parsing

## Tasks

### Task 1: External scanner scaffold

**Spec steps:** 38 (extended)
**Files:** `contrib/tree-sitter-parsley/src/scanner.c`, `grammar.js`
**Estimated effort:** Medium

Steps:
1. Create `src/scanner.c` implementing the tree-sitter external scanner interface (`tree_sitter_external_scanner_create`, `_destroy`, `_serialize`, `_deserialize`, `_scan`)
2. Define scanner state struct: tag stack (names + depth), raw text mode flag, previous token type for regex context
3. Add `externals` declaration to `grammar.js` with initial token types: `_tag_start_disambiguation`, `_close_tag_start`
4. Implement `serialize`/`deserialize` for the tag stack state (required for GLR backtracking)
5. Verify `tree-sitter generate` succeeds with the external scanner wired in
6. Verify existing corpus tests still pass (scanner declines all tokens initially)

Tests:
- `tree-sitter generate` succeeds
- `tree-sitter test` — all 210 existing tests still pass

---

### Task 2: Multi-sibling tag disambiguation

**Spec steps:** 38 (extended)
**Files:** `src/scanner.c`, `grammar.js`, `test/corpus/tags.txt`
**Estimated effort:** Large
**Depends on:** Task 1

This is the single most impactful Phase 2 item — without it, most real handler files produce parse errors.

Steps:
1. In the external scanner, when inside tag children (tag stack non-empty) and encountering `<`:
   - If followed by `/`, emit `_close_tag_start` token (close tag)
   - If followed by a letter, emit `_tag_start_disambiguation` token (new sibling/nested open tag)
   - Otherwise, decline (let internal lexer handle as less-than operator)
2. Push tag name onto stack when open tag is recognized; pop when matching close tag is found
3. Update `grammar.js` tag rules to use externally-scanned tokens for `<` disambiguation in tag content context
4. Un-skip the 2 tag corpus tests that were skipped in Phase 1 (multi-sibling tag cases)
5. Add corpus tests for deeply nested sibling tags, mixed tags and expressions
6. Test against real handler files: `examples/hello/handlers/page.pars`, `examples/auth/handlers/login.pars`

Tests:
- Multi-sibling: `<div><h1>"Title"</h1><p>"Body"</p></div>` — two sibling tags
- Deep nesting: `<html><head><title>"T"</title></head><body><p>"B"</p></body></html>`
- Mixed content: `<div>if cond { <p>"yes"</p> } else { <p>"no"</p> }</div>`
- Comparison in tag: `<div>if a < b { "less" }</div>` — `<` is less-than, not tag start
- `tree-sitter parse` on all `examples/**/*.pars` files — no ERROR nodes in tag-heavy files

---

### Task 3: Style/script raw text with `@{}` interpolation

**Spec steps:** 38
**Files:** `src/scanner.c`, `grammar.js`, `test/corpus/tags.txt`
**Estimated effort:** Large
**Depends on:** Task 2

Steps:
1. Add `raw_text` to `externals` in `grammar.js`
2. In the external scanner, when a `<style>` or `<script>` open tag is pushed onto the stack, enter raw text mode
3. In raw text mode, consume all characters as `raw_text` tokens, stopping at:
   - `@{` — yield `raw_text` for content so far, then decline so the internal lexer handles the interpolation
   - `</style>` or `</script>` — yield `raw_text` for content so far, exit raw text mode, then decline for close tag parsing
   - EOF — yield whatever was consumed
4. Add `raw_text` node to `grammar.js` and wire it into `_tag_child` for style/script contexts
5. Add `raw_interpolation` handling — after `raw_text`, the grammar expects `@{` expression `}` before the next `raw_text`
6. Test against real style blocks from bofdi/components/page.pars

Tests:
- `<style> .cls { color: red; } </style>` — single raw_text node
- `<style> .cls { color: @{theme.color}; } </style>` — raw_text, interpolation, raw_text
- `<script> console.log("hello"); </script>` — script raw text
- `<style> :root { --size: @{base}px; } .a { } .b { } </style>` — multiple CSS rules with interpolation
- Literal `{}` in raw text is NOT treated as Parsley blocks/dicts

---

### Task 4: Regex vs division scanner

**Spec steps:** N/A (extends existing regex rule)
**Files:** `src/scanner.c`, `grammar.js`, `test/corpus/expressions.txt`
**Estimated effort:** Medium
**Depends on:** Task 1

Steps:
1. Add `_regex_start` to `externals` in `grammar.js`
2. In the external scanner, track the previous token type (updated on each `_scan` call)
3. When encountering `/`, check previous token type against the same list as Go's `shouldTreatAsRegex`:
   - After operators, keywords, commas, open parens/brackets, start of input → regex
   - After identifiers, numbers, close parens/brackets → division
4. If regex: emit `_regex_start` token; grammar uses this to begin regex parsing
5. If division: decline; internal lexer handles `/` as operator
6. Update `regex` rule in `grammar.js` to use the external token instead of `token(seq("/", ...))`

Tests:
- `let x = /pattern/gi` — regex after `=`
- `let y = a / b / c` — division after identifiers
- `let z = (a + b) / c` — division after `)`
- `if (/test/gi ~ str)` — regex after `(`
- `[1, /pat/]` — regex after `,`
- `let r = fn(){ /re/ }` — regex after `{`

---

### Task 5: Query DSL sub-grammar

**Spec steps:** 37
**Files:** `grammar.js`, `test/corpus/query.txt`
**Estimated effort:** Large
**Depends on:** None (independent of external scanner)

Steps:
1. Study Go parser's `parseQueryExpression` (L3611–3764) for full query DSL syntax
2. Replace the balanced-paren stub in `query_expression` with real rules:
   - Source: table/identifier reference
   - Alias: optional `as` clause
   - Conditions: `| field op value` (where value can be `{expr}` for interpolation)
   - Modifiers: `| order field`, `| limit n`
   - Group by: `+ by field`
   - Terminal: `?->` (one), `??->` (maybe one), `*` (all), field projections
3. Keep `{expr}` interpolation handling — Parsley expressions embedded in query via `{}`
4. Ensure mutation expressions (`@insert`, `@update`, `@delete`, `@transaction`) remain unchanged (already correct)
5. Add comprehensive corpus tests from real query usage

Tests:
- `@query(People ??-> *)` — simple query, all fields
- `@query(People | Firstname == {name} ?-> *)` — condition with interpolation
- `@query(People | age > {minAge} | order Surname | limit 10 ??-> Firstname, Surname)` — conditions, modifiers, projections
- `@query(People + by department ??-> department, count)` — group by
- Mutations: `@insert(people, record)`, `@update(people, record)` — unchanged behavior

---

### Task 6: Zed extension queries — highlights, brackets, outline, indents

**Spec steps:** 39
**Files:** `contrib/zed-extension/languages/parsley/highlights.scm`, `brackets.scm`, `outline.scm`, `indents.scm`
**Estimated effort:** Medium
**Depends on:** Node name stabilization (all grammar tasks should be complete first)

Steps:
1. Rewrite `highlights.scm` for final node names:
   - Keywords: `fn`, `function`, `let`, `for`, `in`, `if`, `else`, `return`, `export`, `import`, `try`, `check`, `stop`, `skip`, `and`, `or`, `not`, `as`, `computed`, `is`, `via`
   - Operators: all infix/prefix operators
   - Literals: number, string, template_string, raw_string, regex, boolean, money, all @-literals
   - Constants: `"null"` identifier as `@constant.builtin`, `true`/`false` as `@constant.builtin.boolean`
   - Tags: tag names, attribute names, spread attributes
   - Types/builtins: schema types
2. Rewrite `brackets.scm` — matching pairs for `()`, `[]`, `{}`, `<>`/tag open/close
3. Rewrite `outline.scm` — let bindings, export statements, function definitions, schema declarations
4. Rewrite `indents.scm` — blocks, tags, arrays, dictionaries, parameter lists
5. Visual verification in Zed with sample Parsley files

Tests:
- `tree-sitter highlight` on sample files — correct token classification
- Visual verification in Zed with handler files, style blocks, query expressions
- Bracket matching works for all pair types
- Code outline shows let/export/fn definitions

---

### Task 7: Language injection queries

**Spec steps:** N/A (new — emerged from Phase 2 discussion)
**Files:** `contrib/zed-extension/languages/parsley/injections.scm`
**Estimated effort:** Small
**Depends on:** Task 3 (raw_text nodes must exist), Task 6

Steps:
1. Create `injections.scm` with injection rules:
   - `raw_text` inside `<style>` tags → parse with CSS tree-sitter grammar
   - `raw_text` inside `<script>` tags → parse with JavaScript tree-sitter grammar
2. `@{}` interpolation nodes remain as "holes" in the injected parse, keeping Parsley highlighting
3. Test in Zed with real style/script blocks

Tests:
- `<style> .cls { color: red; } </style>` — CSS highlighting inside style tag
- `<script> console.log("hello"); </script>` — JS highlighting inside script tag
- `<style> .cls { color: @{theme.color}; } </style>` — CSS highlighting with Parsley interpolation hole

---

### Task 8: Sample file and final validation

**Spec steps:** 40
**Files:** `contrib/zed-extension/test/sample.pars`
**Estimated effort:** Small
**Depends on:** Tasks 1–7

Steps:
1. Replace `sample.pars` with valid Parsley code exercising all major features including Phase 2 additions:
   - Multi-sibling tags, deeply nested tags
   - `<style>` with `@{}` interpolation
   - Regex literals in various contexts
   - `@query(...)` with real DSL syntax
   - Imports, let bindings, exports, functions, control flow, collections, @-literals
2. Run `tree-sitter parse sample.pars` — no ERROR nodes
3. Run `tree-sitter parse` on all example `.pars` files — no ERROR nodes
4. Final `tree-sitter test` — all corpus tests pass (including new Phase 2 tests)

Tests:
- Zero ERROR nodes on `sample.pars`
- Zero ERROR nodes on all `examples/**/*.pars` files
- Full corpus green
- Visual verification in Zed with all query files active

---

## Recommended Task Order

Tasks 1–3 should be done together — they build the same C scanner file and address related tag-parsing concerns. Task 4 is a small addition to the same scanner. Task 5 is independent and can be done in parallel. Tasks 6–7 depend on node name stabilization and the scanner being complete. Task 8 is the final validation step.

```
Task 1 (scaffold) ──→ Task 2 (tag disambiguation) ──→ Task 3 (raw text) ──→ Task 4 (regex)
                                                                                    │
                                                                                    ▼
Task 5 (query DSL) ─────────────────────────────────────────────────────→ Task 6 (Zed queries)
                                                                                    │
                                                                                    ▼
                                                                          Task 7 (injections)
                                                                                    │
                                                                                    ▼
                                                                          Task 8 (validation)
```

## Validation Checklist

- [x] `tree-sitter generate` succeeds with external scanner
- [x] `tree-sitter test` — all corpus tests pass (Phase 1 + Phase 2) — 253/253 pass
- [x] `tree-sitter parse` on all `examples/**/*.pars` — 120/125 (96%) pass; 5 failures due to known edge cases
- [x] `tree-sitter parse contrib/zed-extension/test/sample.pars` — no ERROR nodes
- [x] Multi-sibling tags parse correctly in real handler files
- [x] `<style>` blocks produce raw_text + interpolation nodes
- [x] Regex vs division is correct in basic contexts (complex `\/` escapes still fail)
- [x] Query DSL has structured parse tree — full sub-grammar implemented with 33 corpus tests
- [x] Highlights render correctly (tree-sitter highlight verified)
- [x] Language injection queries created for CSS in `<style>` and JS in `<script>`
- [x] work/BACKLOG.md updated with any deferrals (#98-#100 added)

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-02-13 | Task 1: Scanner scaffold | ✅ Done | Created `src/scanner.c` with external scanner interface, serialize/deserialize |
| 2026-02-13 | Task 2: Tag disambiguation | ✅ Done | Used `tag_start` token with high precedence to capture `<tagname` as single unit; 220/220 tests pass; multi-sibling tags now work |
| 2026-02-13 | Task 3: Raw text tags | ✅ Done | Scanner produces `raw_text` and `raw_text_interpolation_start` tokens; handles `</div>` inside JS strings correctly |
| 2026-02-13 | Task 4: Regex vs division | 🔶 Partial | Basic regex works; complex patterns with `\/` escapes fail (known limitation, needs scanner extension) |
| 2026-02-14 | Task 5: Query DSL | ✅ Done | Full sub-grammar: query_source, query_clause, query_condition, query_condition_group, query_modifier, query_computed_field, query_subquery, query_group_by, query_terminal, query_projection. 33 new corpus tests added. |
| 2026-02-14 | Task 6: Zed extension queries | ✅ Done | Updated highlights.scm with Query DSL nodes, updated brackets.scm and indents.scm. Fixed sample.pars (invalid `check` and `\|>` syntax). |
| 2026-02-14 | Task 7: Language injection | ✅ Done | Created `injections.scm` in both tree-sitter-parsley/queries and zed-extension for CSS/JS injection |
| 2026-02-14 | Task 8: Final validation | ✅ Done | 253/253 corpus tests pass; 120/125 example files parse (96%); sample.pars parses cleanly |

## Deferred Items

Items to add to work/BACKLOG.md after implementation:
- **`is not` disambiguation** — `value is not Schema` parses as `is_expression(value, prefix_expression(not, Schema))` instead of a dedicated `is not` form. Functionally equivalent for highlighting but could be improved.
- **CSS/JS error recovery in injected content** — if CSS or JS inside style/script tags has syntax errors, the injection parser will produce ERROR nodes. This is expected and acceptable; the Parsley parse tree is unaffected.
- **Complex regex patterns** — Regex literals with `\/` escapes (e.g., `/^https?:\/\/.+/`) fail because the token-level regex pattern stops at the first `/`. Full fix requires external scanner to track escape sequences.
- **XML comments** — `<!-- ... -->` inside tags are not parsed. Would need grammar rule or external scanner support.
- **HTML-like tags inside strings** — Edge case where `<tag>` appears visually inside a quoted string but Parsley's tag-in-tag-content semantics differ from tree-sitter's strict string parsing. Affects 2 example files.

## Additional Changes Made

During implementation, these fixes were also applied:
- **String interpolation removed from double-quoted strings** — Double-quoted strings `"..."` do not support `{...}` interpolation in Parsley; only template strings and raw strings do. Grammar updated.
- **Bare numbers/identifiers as attribute values** — Added support for `width=20` and `class=myClass` (not just `width="20"`)
- **Namespaced attributes** — Added `:` to attribute name pattern for `xlink:href` etc.
- **Style/script token precedence** — `<style` and `<script` use `PREC.TAG + 1` to win over generic `tag_start`