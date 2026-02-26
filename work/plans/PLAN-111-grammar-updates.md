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

#### 4a. Verify `var` statement support

Check whether `var` is supported alongside `let` in `let_statement`. The lexer accepts `var` as a keyword but the tree-sitter grammar may only recognize `"let"`. If missing, add `"var"` as an alternative:

```js
let_statement: ($) =>
  seq(
    choice("let", "var"),
    field("pattern", $._pattern),
    field("operator", choice("=", "<==", "<=/=")),
    field("value", $._expression),
  ),
```

Similarly check `assignment_statement` — no change needed there since `var` only appears at declaration.

Also add `"const"` as a recognized (error-producing) alternative if the parser rejects it gracefully, or leave it to the lexer. Decision: add it so the tree-sitter grammar can parse `const` declarations even though the evaluator rejects them.

#### 4b. Verify query DSL operators

Confirm `?!->` and `??!->` are in the grammar. They should be in `query_terminal` or equivalent.

#### 4c. Regenerate parser

```bash
cd contrib/tree-sitter-parsley
npx tree-sitter generate
```

Tests:
- `npx tree-sitter parse` on test corpus files
- Run existing tree-sitter test corpus: `npx tree-sitter test`
- `var x = 1` parses as `let_statement`

---

## Validation Checklist

- [ ] All `BuiltinMetadata` entries are represented in all 4 grammars' builtin lists
- [ ] No phantom builtins (`print`, `println`, `printf`, `toDebug`) remain in any grammar
- [ ] No phantom operators (`|>`) remain in any grammar
- [ ] All lexer keywords are highlighted in all 4 grammars
- [ ] `var` statement works in both tree-sitter grammars
- [ ] `sql_tag` present in both tree-sitter grammars
- [ ] Tree-sitter test corpus passes: `npx tree-sitter test`
- [ ] Tree-sitter parsers regenerated after grammar.js changes
- [ ] hljs demo.html renders correctly
- [ ] VS Code extension test file highlights correctly
- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `make check`
- [ ] Changes committed

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| | Task 1: highlight.js | ⬜ Not started | |
| | Task 2: tmLanguage | ⬜ Not started | |
| | Task 3: Zed extension | ⬜ Not started | |
| | Task 4: tree-sitter standalone | ⬜ Not started | |

## Deferred Items

Items to add to `work/BACKLOG.md` after implementation:
- Named unit constructors (`bytes`, `metres`, `kilograms`, etc.) could be added to grammar builtin lists for highlighting, but there are ~40+ of them and they are also valid identifiers. Defer until there's a mechanism to export them from the evaluator programmatically (related to backlog #112).
- Zed extension publishing — after grammar updates, a new version should be published to the Zed extension registry.
- VS Code extension packaging — after tmLanguage updates, bump version in `package.json` and repackage.