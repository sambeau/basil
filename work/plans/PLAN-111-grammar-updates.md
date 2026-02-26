---
id: PLAN-111
feature: FEAT-131
title: "Update contrib grammars and VS Code extension to match Parsley 1.0"
status: draft
created: 2026-02-26
---

# Implementation Plan: Grammar Updates for Parsley 1.0

## Overview

The four syntax/grammar packages — tree-sitter (standalone), highlight.js, VS Code extension (tmLanguage), and Zed extension — have drifted from the actual Parsley parser and evaluator. This plan brings them all into alignment with the lexer keywords, operators, and `BuiltinMetadata` as the single source of truth.

## Source of Truth

### Keywords (from `pkg/parsley/lexer/lexer.go` `keywords` map)

```
fn  function  let  var  const  for  in  as  true  false
if  else  return  export  try  import  check  stop  skip
via  is  computed  with  and  or  not
```

Notes:
- `const` is a reserved keyword (suggests using `let` instead) — should still be highlighted as a keyword.
- `var` is a real statement keyword (like `let`) — currently missing from all 4 grammars.
- `with` is used in scoped field access expressions — present in tree-sitter rules but missing from highlight keyword lists.
- `via` is used in schema relations — present in tree-sitter rules but missing from highlight keyword lists.
- `is` is used in schema checking and null checks — present in tree-sitter rules but missing from highlight keyword lists.
- `stop` and `skip` are control flow keywords — present in hljs but missing from tmLanguage and Zed highlights.

### Operators (from `OperatorMetadata` and lexer token types)

Real operators that should be highlighted:
- Arithmetic: `+` `-` `*` `/` `%`
- Comparison: `==` `!=` `<` `>` `<=` `>=`
- Logical: `&&` `||` `!` `and` `or` `not`
- Collection: `++` `in` `not in` `..`
- Regex: `~` `!~`
- Null: `??`
- Assignment: `=`
- Spread/rest: `...`
- File I/O: `<==` `<=/=` `==>` `==>>` `=/=>` `=/=>>`
- Database: `<=?=>` `<=??=>` `<=!=>` `<=#=>`
- Query DSL: `|<` `?->` `??->` `?!->` `??!->` `.->` `<-`

Phantom operators to **remove**:
- `|>` — not in lexer, not parseable. Present in hljs (`QUERY_OPERATORS` regex) and Zed `highlights.scm`.

### Builtins (from `BuiltinMetadata` in `introspect.go` + dynamic registration)

Complete canonical list:
```
JSON  YAML  PLN  CSV  MD  markdown  lines  text  raw  SVG  file  dir  fileList
date  time  datetime
url  path
toInt  toFloat  toNumber  toString  toArray  toDict
serialize  deserialize
inspect  describe  repr  builtins
log  logLine
fail
format  tag
regex  match
money  unit
asset
sqlite  postgres  mysql  sftp  shell
```

Items to **remove** from grammars (not real builtins):
- `print` — does not exist
- `println` — does not exist
- `printf` — does not exist
- `toDebug` — does not exist (may have been removed/renamed)
- `bytes` — this is a named unit constructor, not a standalone builtin; omit to avoid implying it's a general-purpose function
- `now` — this is the `@now` literal, not a callable builtin
- `import` — this is a keyword, not a builtin (already in keywords list)

Items to **add** to grammars:
- `PLN` `raw` `date` `datetime` `path` `serialize` `deserialize` `unit`

### SQL Tag

The standalone tree-sitter grammar has a `sql_tag` rule for `<SQL>...</SQL>` that the Zed copy is missing. The Zed grammar should be updated to match.

---

## Prerequisites

- [x] BUG-023 fixes applied (phantom operators removed from `OperatorMetadata`, builtin metadata corrected)
- [ ] Verify `pars describe all --json` output matches the canonical lists above

---

## Tasks

### Task 1: Update highlight.js grammar (`contrib/highlightjs/parsley.js`)

**Files**: `contrib/highlightjs/parsley.js`
**Estimated effort**: Small

#### 1a. Fix keyword list

Add missing keywords to the `keyword` array:
- Add: `var`, `const`, `with`, `via`, `is`

These are all real lexer keywords. `const` is reserved but should still highlight. `with`, `via`, and `is` are used in expressions/schema and should be recognized.

#### 1b. Fix built_in list

Remove phantom builtins:
- Remove: `print`, `println`, `printf`, `toDebug`, `bytes`, `now`

Add missing builtins:
- Add: `PLN`, `raw`, `date`, `datetime`, `path`, `serialize`, `deserialize`, `unit`

#### 1c. Fix QUERY_OPERATORS regex

Remove `|>` from the regex pattern:
```
// Before:
match: /\?\?->|\?->|\.->|\|<|\|>|<-/

// After:
match: /\?\?!?->|\?!?->|\.->|\|<|<-/
```

Also add the missing query DSL operators `?!->` and `??!->` (RETURN_ONE_EXPLICIT and RETURN_MANY_EXPLICIT).

#### 1d. Verify demo.html still renders correctly

Open `contrib/highlightjs/demo.html` in a browser and confirm highlighting looks right.

Tests:
- All lexer keywords highlight as keywords
- All `BuiltinMetadata` entries highlight as builtins when followed by `(`
- `print(...)` no longer highlights as a builtin
- `var x = 1` highlights `var` as keyword
- `|>` no longer highlighted as an operator
- `?!->` and `??!->` highlight as operators

---

### Task 2: Update VS Code tmLanguage (`syntaxes/parsley.tmLanguage.json`)

**Files**: `.vscode-extension/syntaxes/parsley.tmLanguage.json`
**Estimated effort**: Small

#### 2a. Fix keyword patterns

Add missing keywords to the `keyword.control.parsley` pattern:
- Add: `stop`, `skip` (control flow keywords, alongside `check`)

Add missing keywords to the `keyword.other.parsley` pattern:
- Add: `var`, `const`, `with`, `via`, `is`

Updated patterns:
```
keyword.control: \\b(if|else|return|for|in|check|stop|skip|try)\\b
keyword.other:   \\b(let|var|const|fn|function|as|export|import|and|or|not|computed|with|via|is)\\b
```

Remove the separate `keyword.other.computed.parsley` pattern (fold `computed` into `keyword.other` above) to simplify.

#### 2b. Fix builtins pattern

Remove phantom builtins from the match regex:
- Remove: `print`, `println`, `printf`, `toDebug`, `bytes`, `now`, `import` (already a keyword)

Add missing builtins:
- Add: `PLN`, `raw`, `date`, `datetime`, `path`, `serialize`, `deserialize`, `unit`

Updated match:
```
\\b(time|date|datetime|url|path|file|JSON|YAML|PLN|CSV|lines|text|raw|SVG|MD|markdown|dir|fileList|format|regex|match|tag|asset|repr|inspect|describe|toInt|toFloat|toNumber|toString|toArray|toDict|serialize|deserialize|log|logLine|fail|money|unit|builtins|sqlite|postgres|mysql|sftp|shell)\\b(?=\\s*\\()
```

#### 2c. Fix query DSL operators

Add missing operators `?!->` and `??!->` to the query operator pattern. Remove `|>` if present (check — currently the tmLanguage does include `\\|>` in the query operator pattern).

Updated pattern:
```
\\?\\?!?->|\\?!?->|\\.->|\\|<|<-
```

Tests:
- Run the VS Code extension test file `.vscode-extension/test/syntax-test.pars` (manual inspection)
- `var x = 1` — `var` highlights as keyword
- `print("hello")` — `print` no longer highlights as builtin
- `serialize(data)` — highlights as builtin
- `let x = with obj { a }` — `with` highlights as keyword

---

### Task 3: Update Zed extension highlights (`contrib/zed-extension/`)

**Files**:
- `contrib/zed-extension/grammars/parsley/grammar.js`
- `contrib/zed-extension/grammars/parsley/queries/highlights.scm`

**Estimated effort**: Medium

#### 3a. Sync grammar.js with standalone tree-sitter

The Zed grammar is a copy of the standalone tree-sitter grammar minus the `sql_tag` rule. Sync it:
- Add `sql_tag` rule (copy from `contrib/tree-sitter-parsley/grammar.js`)
- Diff and apply any other divergences

#### 3b. Fix highlights.scm keyword list

Add missing keywords:
- Add: `"var"`, `"const"`, `"stop"`, `"skip"`, `"with"`, `"via"`, `"is"` to the `@keyword` list

#### 3c. Fix highlights.scm operator list

Remove phantom operator:
- Remove: `"|>"` from the Query DSL operators section

Add missing operators:
- Verify `"?!->"` and `"??!->"` are present; add if missing

#### 3d. Regenerate parser

After grammar.js changes, regenerate the parser artifacts:
```bash
cd contrib/zed-extension/grammars/parsley
npx tree-sitter generate
```

This updates `src/grammar.json`, `src/node-types.json`, and `src/parser.c`.

Tests:
- Run `npx tree-sitter parse` on sample `.pars` files to verify parsing works
- Run `npx tree-sitter highlight` on sample files if highlight queries are testable
- `var x = 1` parses correctly
- `<SQL>SELECT 1</SQL>` parses as `sql_tag`

---

### Task 4: Update standalone tree-sitter grammar (`contrib/tree-sitter-parsley/`)

**Files**: `contrib/tree-sitter-parsley/grammar.js`, `contrib/tree-sitter-parsley/src/grammar.json`

**Estimated effort**: Small

The standalone tree-sitter grammar is already the most complete — it has the `sql_tag` and most constructs. Verify and fix any gaps:

#### 4a. Add `var` statement support

Add `"var"` alongside `"let"` in `let_statement` and `export_statement`:

```js
let_statement: ($) =>
  seq(
    choice("let", "var"),
    field("pattern", $._pattern),
    field("operator", choice("=", "<==", "<=/=")),
    field("value", $._expression),
  ),
```

Similarly in `export_statement`, change `"let"` to `choice("let", "var")`.

**Note on `const`**: `const` is NOT added to the tree-sitter grammar rules because the Parsley parser produces an error for it (suggesting `let` instead). Since tree-sitter only recognizes string terminals that appear in grammar rules, `const` cannot appear in `highlights.scm` either — it would cause a "Invalid node type" query error. `const` IS added to the hljs and tmLanguage keyword lists (which use regex matching, not AST node types).

**Note on `stop`/`skip`**: These are expression-level keywords (handled as expression statements in the parser), not grammar terminals in the tree-sitter grammar. They cannot appear in tree-sitter `highlights.scm` keyword lists. They ARE added to hljs and tmLanguage keyword lists.

#### 4b. Verify query DSL operators

Confirm `?!->` and `??!->` are in the grammar. They should be in `query_terminal` or equivalent. ✅ Already present.

#### 4c. Update highlights.scm and injections.scm

- Add `"var"`, `"with"`, `"via"` to keyword list in `highlights.scm`
- Add `(sql_tag) @tag` highlighting
- Add SQL injection rule in `injections.scm`

#### 4d. Add test cases and regenerate parser

Add var statement test cases to test corpus. Regenerate parser:
```bash
cd contrib/tree-sitter-parsley
tree-sitter generate && tree-sitter test
```

Tests:
- All 262 tree-sitter tests pass (258 original + 4 new var tests)
- No highlight query errors
- `var x = 1` parses as `let_statement`
- `export var x = 1` parses as `export_statement`

---

## Validation Checklist

- [x] All `BuiltinMetadata` entries are represented in all 4 grammars' builtin lists
- [x] No phantom builtins (`print`, `println`, `printf`, `toDebug`) remain in any grammar
- [x] No phantom operators (`|>`) remain in any grammar
- [x] All lexer keywords are highlighted in all 4 grammars (see notes below)
- [x] `var` statement works in both tree-sitter grammars
- [x] `sql_tag` present in both tree-sitter grammars
- [x] Tree-sitter test corpus passes: standalone 262/262, Zed pre-existing failures unchanged
- [x] Tree-sitter parsers regenerated after grammar.js changes
- [ ] hljs demo.html renders correctly (manual verification needed)
- [ ] VS Code extension test file highlights correctly (manual verification needed)
- [x] All Parsley tests pass: `go test ./pkg/parsley/...`
- [x] Build succeeds: `make check` (server failures pre-existing, unrelated)
- [ ] Changes committed

**Keyword coverage notes**: `const`, `stop`, and `skip` cannot be added to tree-sitter `highlights.scm` because they are not used as string terminals in any grammar rule (tree-sitter rejects them with "Invalid node type" errors). They ARE highlighted in hljs and tmLanguage which use regex-based matching. This is a tree-sitter limitation, not a gap.

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-02-26 | Task 1: highlight.js | ✅ Complete | Added var/const/with/via/is keywords; removed phantom builtins; added missing builtins; fixed query operator regex |
| 2026-02-26 | Task 2: tmLanguage | ✅ Complete | Added stop/skip/var/const/with/via/is keywords; removed phantom builtins; added missing builtins; fixed query operators |
| 2026-02-26 | Task 3: Zed extension | ✅ Complete | Added sql_tag to grammar + highlights; added var to grammar rules; fixed highlights.scm keywords; removed |> phantom; added SQL injection; added var test cases; regenerated parser |
| 2026-02-26 | Task 4: tree-sitter standalone | ✅ Complete | Added var to grammar rules; updated highlights.scm keywords; added sql_tag highlight + SQL injection; added 4 var test cases; regenerated parser; 262/262 tests pass |

## Deferred Items

Items to add to `work/BACKLOG.md` after implementation:
- Named unit constructors (`bytes`, `metres`, `kilograms`, etc.) could be added to grammar builtin lists for highlighting, but there are ~40+ of them and they are also valid identifiers. Defer until there's a mechanism to export them from the evaluator programmatically (related to backlog #112).
- Zed extension publishing — after grammar updates, a new version should be published to the Zed extension registry.
- VS Code extension packaging — after tmLanguage updates, bump version in `package.json` and repackage.
- Zed tree-sitter test corpus has 84 pre-existing failures (test expectations use `binary_expression` but grammar uses `infix_expression`). These predate this work and should be fixed separately.
- Tree-sitter `const`/`stop`/`skip` highlighting — these keywords can't be highlighted via tree-sitter queries because they aren't grammar terminals. Would require adding them to grammar rules (e.g., a `const_error` rule or `stop`/`skip` as expression forms).