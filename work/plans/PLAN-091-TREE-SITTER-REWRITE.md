---
id: PLAN-091
feature: FEAT-114
title: "Implementation Plan: Tree-sitter Grammar — Direct Translation from Go Source"
status: draft
created: 2026-02-12
---

# Implementation Plan: FEAT-114

## Overview

Rewrite the Parsley tree-sitter grammar from scratch by directly translating the Go lexer (`pkg/parsley/lexer/lexer.go`) and parser (`pkg/parsley/parser/parser.go`) into tree-sitter's JavaScript DSL. Replace the existing grammar and test corpus entirely. Bottom-up: tokens → literals → expressions → statements → tags.

## Prerequisites

- [ ] tree-sitter CLI installed (`tree-sitter generate`, `tree-sitter test`, `tree-sitter parse`)
- [ ] Node.js available (tree-sitter grammar.js requires it)
- [ ] Access to `/Users/samphillips/Dev/bofdi/` for real-world .pars validation files
- [ ] Read FEAT-114 spec for precedence table, keyword list, and step-by-step translation notes

## Source of Truth

| File | Role |
|------|------|
| `pkg/parsley/lexer/lexer.go` | Token types, keywords, operator patterns, string/regex/money lexing |
| `pkg/parsley/parser/parser.go` | Grammar rules, precedence, all `parse*` functions |

When in doubt about any rule, read the Go function — it IS the spec.

---

## Tasks

### Task 1: Scaffold and clean slate

**Spec steps:** 1
**Files:** `contrib/tree-sitter-parsley/grammar.js`, `contrib/tree-sitter-parsley/test/corpus/*`
**Estimated effort:** Small

Steps:
1. Delete all existing corpus files in `test/corpus/`
2. Replace `grammar.js` with a minimal scaffold:
   - `source_file` → `repeat($._statement)`
   - `_statement` → `$.expression_statement` (placeholder, expanded later)
   - `expression_statement` → `$._expression`
   - `_expression` → `$.identifier` (placeholder, expanded later)
   - `identifier` → `/[a-zA-Z_][a-zA-Z0-9_]*/`
   - `comment` → `token(seq('//', /.*/))`
   - `word: ($) => $.identifier`
   - `extras: ($) => [/\s/, $.comment]`
   - `PREC` object with all 12 precedence levels from the spec
3. Run `tree-sitter generate`

Tests:
- Corpus: `identifiers.txt` — bare identifier parses as `(source_file (expression_statement (identifier)))`
- Corpus: `comments.txt` — `// comment` parses cleanly

---

### Task 2: Number and boolean literals

**Spec steps:** 2, 4
**Files:** `grammar.js`, `test/corpus/literals.txt`
**Estimated effort:** Small

Steps:
1. Add `number` → `/\d+(\.\d+)?/`
2. Add `boolean` → `choice('true', 'false')`
3. Wire into `_expression` via a `_literal` choice rule
4. Note: `null` is an identifier, NOT a keyword — handle in `highlights.scm` only

Tests:
- Corpus: integers (`42`), floats (`3.14`), booleans (`true`, `false`)

---

### Task 3: String literals

**Spec steps:** 3
**Files:** `grammar.js`, `test/corpus/strings.txt`
**Estimated effort:** Medium

Steps:
1. Add `string` with `"..."` — content includes escape sequences and `{expr}` interpolation
2. Add `template_string` with `` `...` `` — same interpolation rules
3. Add `raw_string` with `'...'` — uses `@{expr}` for interpolation instead of `{expr}`
4. Add shared rules: `escape_sequence` → `/\\./`, `interpolation` → `seq('{', $._expression, '}')`, `raw_interpolation` → `seq('@{', $._expression, '}')`
5. Wire all three into `_literal`

Tests:
- Plain strings, strings with escapes, strings with `{var}` interpolation
- Template strings with interpolation
- Raw strings with `@{var}` interpolation

---

### Task 4: Regex, money, and @-literals

**Spec steps:** 5, 6, 7, 8
**Files:** `grammar.js`, `test/corpus/at_literals.txt`, `test/corpus/regex.txt`
**Estimated effort:** Medium

Steps:
1. Add `regex` → `token(seq('/', /[^\/\n]+/, '/', optional(/[gimsuvy]+/)))` — placed where prefix is expected to avoid `/` ambiguity
2. Add `money` → `/([$£€¥]|[A-Z]{1,2}[$£€¥]|[A-Z]{3}#)\d+(\.\d{1,2})?/`
3. Add non-template @-literal rules — each as a distinct node:
   - `datetime_literal`, `duration_literal`, `connection_literal`, `context_literal`
   - `time_now_literal`, `stdio_literal`, `path_literal`, `url_literal`
   - `stdlib_import` (for `@std/...`, `@basil/...`)
   - `schema_keyword` (`@schema`), `table_keyword` (`@table`), `query_keyword` (`@query`)
   - `mutation_keyword` (`@insert`, `@update`, `@delete`, `@transaction`)
4. Add template @-literal rules:
   - `path_template`, `url_template`, `datetime_template` — all using `@(...)` with interpolation
5. Wire all into `_literal` / `_at_literal`

Tests:
- Regex: `/pattern/`, `/pattern/gi`
- Money: `$100`, `£50.99`, `CA$50`, `USD#100.00`
- @-literals: one per type (datetime, duration, connection, context, path, url, stdlib)
- @-templates: `@(./path/{name}/file)`, `@({year}-{month}-{day})`

---

### Task 5: Collection literals

**Spec steps:** 9, 10
**Files:** `grammar.js`, `test/corpus/collections.txt`
**Estimated effort:** Small

Steps:
1. Add `array_literal` → `seq('[', commaSep($._expression), ']')`
2. Add `dictionary_literal` → `seq('{', commaSep(choice($.pair, $.computed_property)), '}')`
3. Add `pair` → `seq(field('key', choice($.identifier, $.string)), ':', field('value', $._expression))`
4. Add `computed_property` → `seq('[', field('key', $._expression), ']', ':', field('value', $._expression))`
5. Define `commaSep` and `commaSep1` helpers at top of grammar.js
6. Wire into `_primary_expression`
7. Add `{` dict-vs-block to `conflicts` array (will be needed once blocks exist)

Tests:
- Empty and non-empty arrays, nested arrays
- Empty and non-empty dicts, string keys, computed keys

---

### Task 6: Expression framework — prefix, infix, postfix

**Spec steps:** 11, 12, 13, 14
**Files:** `grammar.js`, `test/corpus/expressions.txt`
**Estimated effort:** Large

Steps:
1. Wire `_expression` → `choice(_primary_expression, prefix_expression, infix_expression, ...)`
2. Wire `_primary_expression` → `choice(identifier, _literal, array_literal, dictionary_literal)`
3. Add `prefix_expression` → `prec(PREFIX, seq(choice('-', '!', 'not'), $._expression))`
4. Add `read_expression` → `seq('<==', $._expression)` and `fetch_expression` → `seq('<=!=', ...)`
5. Add `infix_expression` — one `prec.left()` per precedence level per the spec table:
   - PRODUCT: `*`, `/`, `%`
   - CONCAT: `++`
   - SUM: `+`, `-`, `..`
   - COMPARE: `<`, `>`, `<=`, `>=`
   - EQUALS: `==`, `!=`, `~`, `!~`, `in`, `is`
   - AND: `and`, `&&`, `&`
   - OR: `or`, `||`, `|`, `??`
   - COMMA: `,` operators — `==>`, `==>>`, `=/=>`, `=/=>>`
   - Database: `<=?=>`, `<=??=>`, `<=!=>`, `<=#=>`
6. Add special infixes:
   - `not_in_expression` → `prec.left(EQUALS, seq($._expression, 'not', 'in', $._expression))`
   - `is_not_expression` → `prec.left(EQUALS, seq($._expression, 'is', 'not', $._expression))`
7. Declare necessary `conflicts` for `not` prefix vs `not in` infix

Tests:
- One test per operator
- Precedence test: `1 + 2 * 3` groups as `1 + (2 * 3)`
- `x not in list`, `value is Schema`, `value is not Schema`

---

### Task 7: Call, index, member, and parenthesized expressions

**Spec steps:** 15, 16, 17, 18
**Files:** `grammar.js`, `test/corpus/access.txt`
**Estimated effort:** Medium

Steps:
1. Add `call_expression` → `prec(CALL, seq(field('function', $._expression), field('arguments', $.arguments)))`
2. Add `arguments` → `seq('(', commaSep($._expression), ')')`
3. Add `index_expression` → `prec(INDEX, seq(field('object', $._expression), '[', field('index', $._expression), ']'))`
4. Add `slice_expression` → `prec(INDEX, seq($._expression, '[', optional($._expression), ':', optional($._expression), ']'))`
5. Add `member_expression` → `prec(INDEX, seq(field('object', $._expression), '.', field('property', $.identifier)))`
6. Add `parenthesized_expression` → `seq('(', $._expression, ')')`
7. Wire all into `_expression`

Tests:
- `f()`, `f(1, 2)`, `obj.method()`, chained `a.b().c`
- `arr[0]`, `arr[1:3]`, `arr[:5]`, `arr[2:]`
- `obj.name`, `a.b.c`
- `(1 + 2) * 3`

---

### Task 8: Functions and blocks

**Spec steps:** 19, 20
**Files:** `grammar.js`, `test/corpus/functions.txt`
**Estimated effort:** Medium

Steps:
1. Add `block` → `seq('{', repeat($._statement), '}')`
2. Add `function_expression` → `seq(choice('fn', 'function'), optional(field('parameters', $.parameter_list)), field('body', $.block))`
3. Add `parameter_list` → `seq('(', commaSep(choice($.identifier, $._destructuring_param, $._default_param, $._rest_param)), ')')`
4. Add `_default_param` → `seq($.identifier, '=', $._expression)`
5. Add `_rest_param` → `seq('...', $.identifier)`
6. Add `_destructuring_param` → `choice($.array_pattern, $.dictionary_pattern)` (basic — patterns fully defined in Task 10)
7. Body is ALWAYS a block — no expression bodies
8. Wire `function_expression` and `block` into `_expression`
9. Resolve `{` dict-vs-block conflict in `conflicts` array

Tests:
- `fn(x) { x * 2 }`, `fn(a, b) { a + b }`
- `fn(x, y = 0) { x + y }`, `fn(...rest) { rest }`
- `fn {}` (no parameters)
- `fn({a, b}) { a + b }` (destructuring param)

---

### Task 9: Compound expressions — if, for, try, import

**Spec steps:** 21, 22, 23, 24
**Files:** `grammar.js`, `test/corpus/control_flow.txt`
**Estimated effort:** Large

Steps:
1. Add `if_expression`:
   - With parens: `seq('if', '(', field('condition', $._expression), ')', field('consequence', choice($.block, $._statement)), optional(seq('else', field('alternative', choice($.block, $.if_expression, $._statement)))))`
   - Without parens: `seq('if', field('condition', $._expression), field('consequence', $.block), optional(seq('else', field('alternative', choice($.block, $.if_expression)))))`
   - May need to combine into one rule with careful `choice`/`optional` usage
2. Add `for_expression`:
   - Iteration with parens: `seq('for', '(', optional(seq(field('key', $.identifier), ',')), field('value', $._pattern), 'in', field('iterable', $._expression), ')', field('body', $.block))`
   - Iteration without parens: `seq('for', optional(seq(field('key', $.identifier), ',')), field('value', $._pattern), 'in', field('iterable', $._expression), field('body', $.block))`
   - Mapping form: `seq('for', '(', field('iterable', $._expression), ')', field('mapper', $._expression))`
3. Add `try_expression` → `seq('try', $.call_expression)`
4. Add `import_expression` → `seq('import', field('source', $._expression), optional(seq('as', field('alias', $.identifier))))`
5. Wire all into `_expression`

Tests:
- If: all forms from spec (parens/no-parens, with/without else, chained else-if, compact ternary)
- For: iteration with/without parens, key-value, mapping form
- Try: `try fetchData()`, `try obj.load()`
- Import: `import @std/math`, `import @./utils.pars as utils`

---

### Task 10: Patterns and statements

**Spec steps:** 25, 26, 27, 28, 29, 30, 31
**Files:** `grammar.js`, `test/corpus/statements.txt`, `test/corpus/patterns.txt`
**Estimated effort:** Large

Steps:
1. Add destructuring patterns:
   - `_pattern` → `choice($.identifier, $.array_pattern, $.dictionary_pattern, '_')`
   - `array_pattern` → `seq('[', commaSep(choice($._pattern, seq('...', optional($.identifier)))), ']')`
   - `dictionary_pattern` → `seq('{', commaSep(choice($.identifier, seq(field('key', $.identifier), ':', field('value', $._pattern)), seq('...', optional($.identifier)))), '}')`
2. Add `let_statement` → `seq('let', field('pattern', $._pattern), '=', field('value', $._expression))`
3. Add `assignment_statement` → `seq(field('left', choice($.identifier, $.member_expression, $.index_expression, $.dictionary_pattern)), '=', field('right', $._expression))`
4. Add `export_statement` → `seq('export', optional('computed'), field('name', $.identifier), '=', field('value', $._expression))`
5. Add `return_statement` → `seq('return', optional($._expression))`
6. Add `check_statement` → `seq('check', field('condition', $._expression), 'else', field('fallback', $._expression))`
7. Add `stop_statement` → `'stop'`
8. Add `skip_statement` → `'skip'`
9. Expand `_statement` choice to include all statement types with correct priority order (matching `parseStatement` switch at L292)
10. Handle IDENT-followed-by-ASSIGN disambiguation — may need `conflicts` or `prec`

Tests:
- Let: `let x = 5`, `let [a, b] = arr`, `let {name} = person`
- Assignment: `x = 5`, `obj.name = "Alice"`, `arr[0] = 99`, `{a, b} = expr`
- Export: `export greeting = "Hello"`, `export computed total = a + b`
- Return/check/stop/skip: `return x + 1`, `check x > 0 else "negative"`, `stop`, `skip`
- Multi-statement program exercising each type

---

### Task 11: Tags

**Spec steps:** 32, 33, 34
**Files:** `grammar.js`, `test/corpus/tags.txt`
**Estimated effort:** Large

Steps:
1. Add `tag_expression` → `choice($.self_closing_tag, seq($.open_tag, repeat($._statement), $.close_tag))`
2. Add `self_closing_tag` → `seq('<', field('name', $.tag_name), repeat($.tag_attribute), '/>')`
3. Add `open_tag` → `seq('<', field('name', $.tag_name), repeat($.tag_attribute), '>')`
4. Add `close_tag` → `seq('</', field('name', $.tag_name), '>')`
5. Add `tag_name` → `/[a-zA-Z][a-zA-Z0-9-]*/` (also handle `<>...</>` grouping tags)
6. Add tag attributes:
   - `tag_attribute` → `choice(seq(field('name', $.attribute_name), '=', field('value', choice($.string, $.tag_expression_attribute))), field('name', $.attribute_name), $.tag_spread_attribute)`
   - `attribute_name` → `/[a-zA-Z@][a-zA-Z0-9_-]*/`
   - `tag_expression_attribute` → `seq('{', $._expression, '}')`
   - `tag_spread_attribute` → `seq('...', $.identifier)`
7. Wire `tag_expression` into `_expression`
8. Handle `<` tag-start vs less-than — tree-sitter should resolve structurally since tag shapes are distinct
9. Handle `{` in tag attributes (expression value) vs `{` in tag content (dictionary) — different parent contexts

Tests:
- Self-closing: `<br/>`, `<input @field="name"/>`
- Open/close with content: `<div>"Hello"</div>`, `<p>name</p>`
- Nested tags: `<div><span>"nested"</span></div>`
- Attributes: `class="container"`, `class={expr}`, `...props`
- Code content: `<ul> for (item in items) { <li>item</li> } </ul>`
- Real-world from bofdi: unsafeTable pattern with `for`, `if`, destructuring inside tags

---

### Task 12: Phase 2 — Schema, tables, queries

**Spec steps:** 35, 36, 37
**Files:** `grammar.js`, `test/corpus/schema.txt`, `test/corpus/query.txt`
**Estimated effort:** Large

Steps:
1. Add `schema_declaration` → `seq('@schema', field('name', $.identifier), '{', repeat($.schema_field), '}')`
2. Add `schema_field` with type, optional marker `?`, enum values, defaults, metadata `|`, and `via`
3. Add `table_literal` → `seq('@table', optional(seq('(', field('schema', $.identifier), ')')), '[', commaSep($.dictionary_literal), ']')`
4. Add query DSL rules:
   - `query_expression` → `seq('@query', '(', ...query clauses..., ')')`
   - Query clauses: source, optional `as` alias, conditions (`| field op value`), modifiers, terminal (`?->`, `??->`, `*`, projections)
5. Add mutation expressions: `@insert(...)`, `@update(...)`, `@delete(...)`, `@transaction(...)`
6. Wire `export @schema` and `export @table` into `export_statement`

Tests:
- Schema: from `bofdi/schema/person.pars`
- Table: `@table [{name: "Alice"}, {name: "Bob"}]`
- Query: from `bofdi/schema/birthday.pars` — `@query(People ??-> *)`, condition queries
- Mutations: `@insert(...)`, `@update(...)`

---

### Task 13: Phase 2 — External scanner for style/script

**Spec steps:** 38
**Files:** `grammar.js`, `contrib/tree-sitter-parsley/src/scanner.c`
**Estimated effort:** Large

Steps:
1. Create `src/scanner.c` implementing tree-sitter external scanner interface
2. Handle `<style>...</style>` and `<script>...</script>` as raw text mode
3. Parse `@{expr}` interpolation within raw text — switch back to normal parsing for the expression
4. Everything else between the tags is literal text (including `{` and `}`)
5. Register external scanner tokens in `grammar.js` via `externals`
6. Test against `bofdi/components/page.pars` which has real `<style>` blocks

Tests:
- `<style> :root { --color: red; } </style>` — raw text content
- `<style> .cls { color: @{theme.color}; } </style>` — with interpolation
- `<script> console.log("hello"); </script>` — script raw text

---

### Task 14: Zed extension queries

**Spec steps:** 39
**Files:** `contrib/zed-extension/languages/parsley/highlights.scm`, `brackets.scm`, `outline.scm`, `indents.scm`
**Estimated effort:** Medium

Steps:
1. Rewrite `highlights.scm` for new node names:
   - Keywords: `fn`, `function`, `let`, `for`, `in`, `if`, `else`, `return`, `export`, `import`, `try`, `check`, `stop`, `skip`, `and`, `or`, `not`, `as`, `computed`, `is`, `via`
   - Operators: all infix/prefix operators
   - Literals: number, string, template_string, raw_string, regex, boolean, money, @-literals
   - Highlight `"null"` identifier as `@constant.builtin`
   - Types/builtins: tag names, schema types
2. Rewrite `brackets.scm` — matching pairs for `()`, `[]`, `{}`, `<>`/tag pairs
3. Rewrite `outline.scm` — let bindings, export statements, function definitions, schema declarations
4. Rewrite `indents.scm` — blocks, tags, arrays, dictionaries, parameter lists

Tests:
- Visual verification in Zed with sample Parsley files
- `tree-sitter highlight` on sample files

---

### Task 15: Sample file and final validation

**Spec steps:** 40
**Files:** `contrib/zed-extension/test/sample.pars`
**Estimated effort:** Small

Steps:
1. Replace `sample.pars` with valid Parsley code exercising all major features:
   - Imports, let bindings, exports
   - Functions, control flow (if/for/try/check)
   - Tags with code content, attributes, nesting
   - String interpolation, @-literals, collections
   - Member access, calls, index/slice
2. Run `tree-sitter parse sample.pars` — no ERROR nodes
3. Run `tree-sitter parse` on all bofdi .pars files — no ERROR nodes
4. Final `tree-sitter test` — all corpus tests pass

Tests:
- Zero ERROR nodes on `sample.pars`
- Zero ERROR nodes on all `bofdi/**/*.pars` files
- Full corpus green

---

## Conflict & Ambiguity Strategy

These are known ambiguities that will require `conflicts` declarations or structural resolution. Address them as they arise during the relevant task:

| Ambiguity | Approach | Task |
|-----------|----------|------|
| `{` — dictionary vs block | `conflicts` declaration | Task 5/8 |
| `<` — tag start vs less-than | Structural (distinct shapes) | Task 11 |
| `/` — division vs regex | Structural (regex in prefix position) | Task 4/6 |
| `not` — prefix vs `not in` | `conflicts` or compound operator | Task 6 |
| Statement vs expression in tags | Natural (`repeat($._statement)`) | Task 11 |
| Dict destructuring assignment | `conflicts` for `{` at statement start | Task 10 |

---

## Validation Checklist

- [ ] `tree-sitter generate` succeeds
- [ ] `tree-sitter test` — all corpus tests pass
- [ ] `tree-sitter parse contrib/zed-extension/test/sample.pars` — no ERROR nodes
- [ ] `tree-sitter parse` on all bofdi .pars files — no ERROR nodes
- [ ] Highlights render correctly in Zed
- [ ] Grammar node names mirror Go parser function names where applicable
- [ ] Precedence levels match Go `var precedences` map exactly
- [ ] Keywords match Go `var keywords` map exactly
- [ ] work/BACKLOG.md updated with any deferrals

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-02-12 | Task 1: Scaffold | ✅ Done | Clean slate grammar.js, PREC map from parser.go |
| 2026-02-12 | Task 2: Numbers & booleans | ✅ Done | number, boolean rules; null is identifier |
| 2026-02-12 | Task 3: Strings | ✅ Done | string, template_string, raw_string with interpolation |
| 2026-02-12 | Task 4: Regex, money, @-literals | ✅ Done | All @-literals: datetime, duration, connection, context, stdlib, stdio, path, URL, path_template |
| 2026-02-12 | Task 5: Collections | ✅ Done | array_literal, dictionary_literal, pair, computed_property |
| 2026-02-12 | Task 6: Prefix & infix | ✅ Done | All operators mapped with correct precedence; `not in` and `is` expressions |
| 2026-02-12 | Task 7: Call, index, member | ✅ Done | call_expression, index_expression, slice_expression, member_expression (incl. `as` property) |
| 2026-02-12 | Task 8: Functions & blocks | ✅ Done | fn/function, block bodies, default/rest/destructuring params |
| 2026-02-12 | Task 9: If, for, try, import | ✅ Done | if (parens/no-parens), for (iteration/mapping with prec.dynamic), try, import with alias |
| 2026-02-12 | Task 10: Patterns & statements | ✅ Done | let, assignment, dict destructuring, export (simple/computed/bare), return, check; wildcard `_` aliased to identifier |
| 2026-02-12 | Task 11: Tags | ✅ Done | open/close, self-closing, grouping, attributes, spread, expression values; multi-sibling limitation noted |
| 2026-02-12 | Task 12: Schema, tables, queries | ✅ Stub | schema_declaration, table_expression, query_expression, mutation_expression — Phase 2 for full query DSL |
| | Task 13: External scanner | ⬜ Not started | Phase 2 — needed for multi-sibling tags and style/script raw text |
| | Task 14: Zed extension queries | ⬜ Not started | Blocked on grammar node name stabilization |
| 2026-02-12 | Task 15: Sample file & validation | 🔶 Partial | 210/210 corpus tests pass (100%); ~30/38 example files parse cleanly; handler files need Phase 2 external scanner |

## Deferred Items

Items to add to work/BACKLOG.md after implementation:
- **Multi-sibling tag children**: After `</tag>`, the `<` of the next sibling tag is consumed as a comparison operator. Requires an external scanner (`src/scanner.c`) to manage a tag stack and prioritize `</` over `<` in tag content context. Affects all real handler files.
- **`is not` disambiguation**: `value is not Schema` parses as `is_expression(value, prefix_expression(not, Schema))` instead of a dedicated `is not` form. Functionally equivalent for highlighting but could be improved with external scanner or token-level changes.
- **Style/script raw text tags**: `<style>` and `<script>` tags should treat content as raw text with `@{}` interpolation. Requires external scanner.
- **Query DSL sub-grammar**: `@query(...)` content is currently opaque (balanced parens). Full query DSL parsing is Phase 2.
- **Regex vs division**: Currently handled by token-level regex pattern; complex regex patterns (e.g., with `/` inside character classes) may fail. External scanner could improve this.