---
id: PLAN-085
feature: FEAT-109
title: "Implementation Plan for Tree-sitter Grammar and Registry Submissions"
status: complete
created: 2026-02-10
---

# Implementation Plan: FEAT-109

## Overview
Create a Tree-sitter grammar for Parsley covering all language syntax, publish it as a standalone repository, and submit to the tree-sitter registry and GitHub linguist for `.pars` file recognition.

## Prerequisites
- [x] FEAT-108 complete (VS Code and highlight.js grammars updated — informs what syntax to support)
- [x] Lexer reviewed as source of truth (`pkg/parsley/lexer/lexer.go`)
- [ ] `tree-sitter` CLI installed (`npm install -g tree-sitter-cli` or `cargo install tree-sitter-cli`)
- [ ] Node.js available for grammar generation

## Tasks

### Task 1: Project Scaffolding
**Files**: `contrib/tree-sitter-parsley/` (new directory tree)
**Estimated effort**: Small

Steps:
1. Run `tree-sitter init` inside `contrib/tree-sitter-parsley/` to generate boilerplate
2. Verify generated structure includes `grammar.js`, `bindings/`, `package.json`, `Cargo.toml`, `binding.gyp`
3. Edit `package.json` with correct metadata (name: `tree-sitter-parsley`, repository URL, etc.)
4. Edit `Cargo.toml` with crate metadata
5. Create `queries/` directory for highlight queries
6. Verify `tree-sitter generate` runs without errors on the skeleton grammar

Tests:
- `tree-sitter generate` succeeds
- `tree-sitter test` runs (even if no corpus tests yet)

---

### Task 2: Core Grammar — Literals and Basic Tokens
**Files**: `contrib/tree-sitter-parsley/grammar.js`
**Estimated effort**: Medium

Implement rules for all literal types and basic tokens, derived from the lexer (`pkg/parsley/lexer/lexer.go`):

Steps:
1. Define `extras` (whitespace, comments)
2. Add `comment` rule: `//` line comments
3. Add `identifier` rule: `[a-zA-Z_][a-zA-Z0-9_]*`
4. Add number rules: integer and float
5. Add string rules with interpolation:
   - `string`: double-quoted with `{expr}` interpolation
   - `template_string`: backtick with `{expr}` interpolation
   - `raw_string`: single-quoted with `@{expr}` interpolation
   - `escape_sequence`: `\\.` sequences
6. Add `regex` rule: `/pattern/flags`
7. Add boolean (`true`, `false`) and `null` literals
8. Add `money` literal: `([$£€¥]|[A-Z]{3}#)\d+(\.\d{1,2})?`
9. Add all at-literal rules (from `detectAtLiteralType`):
   - `datetime_literal`: `@2024-01-15`, `@2024-01-15T10:30:00Z`, `@12:30:00`
   - `time_now_literal`: `@now`, `@today`, `@timeNow`, `@dateNow`
   - `duration_literal`: `@2h30m`, `@-7d`, `@1y6mo`
   - `connection_literal`: `@sqlite`, `@postgres`, `@mysql`, `@sftp`, `@shell`, `@DB`
   - `schema_literal`: `@schema`
   - `table_literal`: `@table`
   - `query_literal`: `@query`, `@insert`, `@update`, `@delete`, `@transaction`
   - `context_literal`: `@SEARCH`, `@env`, `@args`, `@params`
   - `stdlib_import`: `@std/module`, `@std`, `@basil`, `@basil/http`, `@basil/auth`
   - `path_literal`: `@./file`, `@../dir`, `@/usr/local`, `@~/home`, `@-`, `@stdin`, `@stdout`, `@stderr`, `@.config`
   - `url_literal`: `@https://...`, `@http://...`, `@ftp://...`, `@file://...`
   - `path_template`: `@(./path/{expr})`
   - `url_template`: `@(https://api.com/{expr})`
   - `datetime_template`: `@(2024-{month}-{day})`
10. Write corpus tests for each literal type

Tests:
- `tree-sitter generate` succeeds
- `tree-sitter test` — all literal corpus tests pass
- `tree-sitter parse` on a sample file with all literal types — no ERROR nodes

---

### Task 3: Core Grammar — Keywords and Statements
**Files**: `contrib/tree-sitter-parsley/grammar.js`
**Estimated effort**: Medium

Steps:
1. Define all keywords (from `var keywords` in lexer):
   - `fn`/`function`, `let`, `for`, `in`, `as`, `if`, `else`, `return`
   - `export`, `try`, `import`, `check`, `stop`, `skip`
   - `via`, `is`, `computed`
   - `true`, `false` (already done as literals)
   - `and`, `or`, `not` (keyword operators)
2. Implement statement rules:
   - `let_statement`: `let identifier = expression`
   - `function_definition`: `identifier = fn(params) expression`
   - `export_statement`: `export [computed] identifier = expression`
   - `import_statement`: `import expression [as identifier]`
   - `for_statement`: `for identifier in expression { body }` / `for expression`
   - `if_expression`: `if condition { body } [else { body }]` / ternary form
   - `try_expression`: `try expression`
   - `return_statement`: `return expression`
   - `check_statement`: `check expression [{ body }]`
   - `expression_statement`: bare expression
3. Implement `parameter_list` and `argument_list` rules
4. Implement destructuring patterns (array and dictionary)
5. Write corpus tests for each statement type

Tests:
- `tree-sitter test` — all statement corpus tests pass
- Parse sample `.pars` files with mixed statements — no ERROR nodes

---

### Task 4: Core Grammar — Operators and Expressions
**Files**: `contrib/tree-sitter-parsley/grammar.js`
**Estimated effort**: Medium

Steps:
1. Implement binary expression with correct precedence (lowest to highest):
   - Nullish coalescing: `??` (right-associative)
   - Logical: `and`/`or`/`&&`/`||`
   - Comparison: `==`, `!=`, `<`, `>`, `<=`, `>=`
   - Regex match: `~`, `!~`
   - Range: `..`
   - Addition: `+`, `-`
   - Multiplication: `*`, `/`, `%`
   - Concatenation: `++`
2. Implement unary expression: `!`, `-`, `not`
3. Implement I/O operators:
   - File I/O: `<==`, `<=/=`, `==>`, `==>>`, `=/=>`, `=/=>>`
   - Database: `<=?=>`, `<=??=>`, `<=!=>`, `<=#=>`
4. Implement Query DSL operators:
   - `|<` (pipe write), `|>` (read projection)
   - `?->` (return one), `??->` (return many)
   - `?!->` (return one explicit), `??!->` (return many explicit)
   - `.->` (exec count), `<-` (arrow pull / correlated subquery)
5. Implement other expressions:
   - `call_expression`: `identifier(args)`
   - `index_expression`: `expression[index]`
   - `member_expression`: `expression.identifier`
   - `array_literal`: `[expr, expr, ...]`
   - `dictionary_literal`: `{key: value, ...}`
   - `function_expression`: `fn(params) expression`
   - `parenthesized_expression`: `(expression)`
   - Spread: `...expression`
   - Assignment: `identifier = expression`
6. Write corpus tests for operator precedence and all expression types

Tests:
- `tree-sitter test` — all expression corpus tests pass
- Verify precedence: `1 + 2 * 3` parses as `1 + (2 * 3)`

---

### Task 5: Core Grammar — JSX-like Tags
**Files**: `contrib/tree-sitter-parsley/grammar.js`, possibly `contrib/tree-sitter-parsley/src/scanner.c`
**Estimated effort**: Medium-Large

This is the most complex part. Parsley tags are JSX-like with embedded expressions.

Steps:
1. Implement tag rules:
   - `self_closing_tag`: `<Name attr=value />`
   - `open_tag` + `close_tag`: `<Name>...</Name>`
   - `tag_name`: `[a-zA-Z][a-zA-Z0-9-]*`
   - `tag_attribute`: `name=value` or `name={expr}` or bare `name`
   - `tag_text`: raw text content between tags
   - `embedded_expression`: `{expression}` inside tag content
   - `spread_attribute`: `...identifier`
2. Handle tag content mode (text vs expressions):
   - Text between tags is literal until `{`, `<`, or `</`
   - May require an **external scanner** in C for context-sensitive lexing
3. Handle raw text tags (if applicable — e.g., `<script>`, `<style>`)
4. Handle nested tags
5. Write corpus tests for tags

Tests:
- Simple self-closing: `<br/>`
- Tag with attributes: `<div class="foo">`
- Nested tags: `<div><span>text</span></div>`
- Embedded expressions: `<p>{variable}</p>`
- Spread attributes: `<div ...props>`
- Mixed content: `<p>Hello {name}, welcome!</p>`

---

### Task 6: Highlight Queries
**Files**: `contrib/tree-sitter-parsley/queries/highlights.scm`
**Estimated effort**: Small

Steps:
1. Map all grammar nodes to highlight capture names:

```
; Keywords
["let" "fn" "function" "if" "else" "for" "in" "return" "export" "import"
 "try" "check" "stop" "skip" "as" "via" "is" "computed"] @keyword
["and" "or" "not"] @keyword.operator

; Literals
(number) @number
(money) @number
(boolean) @constant.builtin
(null) @constant.builtin
(string) @string
(template_string) @string
(raw_string) @string
(regex) @string.regexp
(escape_sequence) @string.escape

; At-literals
(datetime_literal) @number
(time_now_literal) @constant.builtin
(duration_literal) @number
(connection_literal) @function.builtin
(schema_literal) @type
(table_literal) @type
(query_literal) @function.builtin
(context_literal) @variable.builtin
(stdlib_import) @module
(path_literal) @string.special.path
(url_literal) @string.special.url

; Operators
["+", "-", "*", "/", "%"] @operator
["==" "!=" "<" ">" "<=" ">="] @operator
["~" "!~"] @operator
["++" "??" ".."] @operator
["=" "<==" "==>" "==>>" "<=/=" "=/=>" "=/=>>"] @operator
["<=?=>" "<=??=>" "<=!=>" "<=#=>"] @operator
["|>" "|<" "?->" "??->" "?!->" "??!->" ".->" "<-"] @operator
["&&" "||" "!" "..."] @operator

; Punctuation
["(" ")" "[" "]" "{" "}"] @punctuation.bracket
["," ";" ":" "."] @punctuation.delimiter

; Tags
(tag_name) @tag
(tag_attribute name: (_) @attribute)

; Functions
(function_definition name: (identifier) @function)
(call_expression function: (identifier) @function.call)

; Interpolation
(interpolation ["{" "}"] @punctuation.special)

; Comments
(comment) @comment

; Identifiers (lowest priority)
(identifier) @variable
```

2. Optionally create `queries/locals.scm` for variable scoping
3. Optionally create `queries/injections.scm` (probably not needed initially)

Tests:
- `tree-sitter highlight sample.pars` produces correct output
- Visual check in editors that support tree-sitter

---

### Task 7: Test Corpus — Comprehensive
**Files**: `contrib/tree-sitter-parsley/test/corpus/*.txt`
**Estimated effort**: Medium

Steps:
1. Create `test/corpus/literals.txt` — all literal types
2. Create `test/corpus/strings.txt` — strings, templates, interpolation, escapes
3. Create `test/corpus/statements.txt` — let, export, import, return, check, for, if
4. Create `test/corpus/expressions.txt` — binary, unary, call, index, member, precedence
5. Create `test/corpus/operators.txt` — I/O, database, Query DSL operators
6. Create `test/corpus/tags.txt` — JSX-like tags, attributes, nesting, text content
7. Create `test/corpus/at_literals.txt` — exhaustive at-literal coverage
8. Create `test/corpus/functions.txt` — fn definitions, closures, parameters

Tests:
- `tree-sitter test` — all corpus tests pass
- Parse real `.pars` files from the project — no ERROR nodes

---

### Task 8: Parse Real Parsley Files
**Files**: None (validation step)
**Estimated effort**: Small

Steps:
1. Collect sample `.pars` files from the project or create representative ones
2. Run `tree-sitter parse` on each file
3. Fix any ERROR nodes by adjusting grammar rules
4. Iterate until clean parses

Tests:
- Zero ERROR nodes on representative Parsley files
- `tree-sitter highlight` produces reasonable output

---

### Task 9: Standalone Repository
**Files**: New repo `github.com/sambeau/tree-sitter-parsley`
**Estimated effort**: Small

Steps:
1. Create `tree-sitter-parsley` GitHub repository
2. Copy contents from `contrib/tree-sitter-parsley/`
3. Set up CI (GitHub Actions) for:
   - `tree-sitter generate` (verify grammar compiles)
   - `tree-sitter test` (run corpus tests)
   - Build Node.js and Rust bindings
4. Add README with usage instructions for each editor
5. Tag initial release

Tests:
- CI passes on the standalone repo
- Can install from the repo URL in each editor

---

### Task 10: Registry Submissions
**Files**: External PRs
**Estimated effort**: Small (but async — review may take days/weeks)

Steps:
1. **Tree-sitter registry**: Add entry to the tree-sitter wiki list of parsers
2. **GitHub linguist**:
   a. Fork `github/linguist`
   b. Add `Parsley` to `lib/linguist/languages.yml`:
      - `type: programming`
      - `extensions: [".pars"]`
      - `tm_scope: source.parsley`
   c. Add sample `.pars` files to `samples/Parsley/`
   d. Reference tree-sitter grammar in `grammars.yml`
   e. Submit PR
3. Verify after merge: `.pars` files render with highlighting on github.com

Tests:
- Linguist PR passes their CI
- After merge, `.pars` files on GitHub show syntax highlighting

---

## Validation Checklist
- [x] `tree-sitter generate` succeeds without errors
- [x] `tree-sitter test` — all corpus tests pass (129/129)
- [x] `tree-sitter parse sample.pars` — no ERROR nodes on representative files
- [x] `tree-sitter highlight sample.pars` — correct highlighting output
- [ ] Grammar works in Zed editor (pending — repo now public)
- [ ] Grammar works in Neovim with tree-sitter plugin
- [ ] Grammar works in Helix editor
- [x] Standalone repo published (github.com/sambeau/tree-sitter-parsley)
- [x] Linguist PR submitted (awaiting review)
- [ ] Tree-sitter registry entry added
- [ ] work/BACKLOG.md updated with deferrals (if any)

## Key Design Notes

### Operator Precedence (lowest to highest)
Derived from `pkg/parsley/lexer/lexer.go` and parser:

| Level | Operators | Associativity |
|-------|-----------|---------------|
| 1 | `??` | Right |
| 2 | `or`, `\|\|` | Left |
| 3 | `and`, `&&` | Left |
| 4 | `==`, `!=`, `<`, `>`, `<=`, `>=` | Left |
| 5 | `~`, `!~` | Left |
| 6 | `..` | Left |
| 7 | `+`, `-` | Left |
| 8 | `*`, `/`, `%` | Left |
| 9 | `++` | Left |
| 10 | Unary `-`, `!`, `not` | Right (prefix) |

### Complete Token Inventory (from lexer)

**Keywords**: `fn`, `function`, `let`, `for`, `in`, `as`, `true`, `false`, `if`, `else`, `return`, `export`, `try`, `import`, `check`, `stop`, `skip`, `via`, `is`, `computed`, `and`, `or`, `not`

**At-literals**: `@sqlite`, `@postgres`, `@mysql`, `@sftp`, `@shell`, `@DB`, `@SEARCH`, `@env`, `@args`, `@params`, `@schema`, `@table`, `@query`, `@insert`, `@update`, `@delete`, `@transaction`, `@now`, `@today`, `@timeNow`, `@dateNow`, `@std`, `@std/...`, `@basil`, `@basil/...`

**I/O Operators**: `<==`, `<=/=`, `==>`, `==>>`, `=/=>`, `=/=>>`

**Database Operators**: `<=?=>`, `<=??=>`, `<=!=>`, `<=#=>`

**Query DSL Operators**: `|<`, `?->`, `??->`, `?!->`, `??!->`, `.->`, `<-`

**Standard Operators**: `+`, `-`, `*`, `/`, `%`, `==`, `!=`, `<`, `>`, `<=`, `>=`, `&&`, `||`, `!`, `~`, `!~`, `??`, `..`, `...`, `++`, `=`

### External Scanner Consideration
The JSX-like tag system likely requires an external scanner (`src/scanner.c`) to handle:
- Context-sensitive tag text (raw text between tags)
- Distinguishing `<` as less-than vs tag-open
- Nested tag matching
- Raw text tags

This is the same approach used by `tree-sitter-html` and `tree-sitter-jsx`.

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-02-10 | Task 1: Scaffolding | ✅ Complete | package.json, Cargo.toml, binding.gyp, tree-sitter.json |
| 2026-02-10 | Task 2: Literals & Tokens | ✅ Complete | All at-literals, strings, numbers, money, regex |
| 2026-02-10 | Task 3: Keywords & Statements | ✅ Complete | let, export, import, for, if, check, return, try |
| 2026-02-10 | Task 4: Operators & Expressions | ✅ Complete | All operators including I/O, DB, Query DSL |
| 2026-02-10 | Task 5: JSX-like Tags | ✅ Complete | No external scanner needed |
| 2026-02-10 | Task 6: Highlight Queries | ✅ Complete | queries/highlights.scm |
| 2026-02-10 | Task 7: Test Corpus | ✅ Complete | 128 tests, 100% passing |
| 2026-02-10 | Task 8: Parse Real Files | ✅ Complete | Verified with sample Parsley code |
| 2026-02-11 | Task 9: Standalone Repo | ✅ Complete | Published to github.com/sambeau/tree-sitter-parsley |
| 2026-02-11 | Task 10: Registry Submissions | 🔶 Partial | GitHub Linguist PR submitted, awaiting review |

## Deferred Items
- Advanced tree-sitter features (code navigation, folding, indentation queries) — separate follow-up
- Language server protocol (LSP) integration — much larger effort
- `queries/locals.scm` for full variable scoping — can add after initial release
- Tree-sitter registry wiki entry — can be added after linguist acceptance