---
id: PLAN-084
feature: FEAT-108
title: "Implementation Plan for Syntax Highlighter Updates"
status: draft
created: 2025-01-15
---

# Implementation Plan: FEAT-108

## Overview
Update the VS Code TextMate grammar and highlight.js grammar to support recent Parsley language features including schema declarations, Query DSL, computed exports, and runtime context literals.

## Prerequisites
- [x] Existing grammars reviewed and understood
- [x] Missing syntax elements identified from FEAT-108 spec

## Tasks

### Task 1: Update VS Code TextMate Grammar
**Files**: `.vscode-extension/syntaxes/parsley.tmLanguage.json`
**Estimated effort**: Small

Steps:
1. Add schema and table literals (`@schema`, `@table`) to `special-literals` section
2. Add Query DSL literals (`@query`, `@insert`, `@update`, `@delete`, `@transaction`) to `special-literals` section
3. Add runtime context literals (`@SEARCH`, `@env`, `@args`, `@params`) to `special-literals` section
4. Add `computed` keyword to `keywords` section
5. Add Query DSL operators (`|<`, `|>`, `?->`, `??->`, `.->`, `<-`) to `operators-special` section

Tests:
- Open a Parsley file with `@schema User { }` — `@schema` should highlight as constant
- Open a Parsley file with `@table [...]` — `@table` should highlight as constant
- Open a Parsley file with `@query(db)` — `@query` should highlight as constant
- Open a Parsley file with `@insert`, `@update`, `@delete` — all should highlight as constants
- Open a Parsley file with `@transaction { }` — `@transaction` should highlight as constant
- Open a Parsley file with `@SEARCH` — should highlight as constant
- Open a Parsley file with `@env.HOME` — `@env` should highlight as constant
- Open a Parsley file with `@args` — should highlight as constant
- Open a Parsley file with `@params` — should highlight as constant
- Open a Parsley file with `export computed x = 1` — `computed` should highlight as keyword
- Open a Parsley file with `data ?-> result` — `?->` should highlight as operator
- Open a Parsley file with `data ??-> results` — `??->` should highlight as operator
- Open a Parsley file with `data .-> count` — `.->` should highlight as operator
- Open a Parsley file with `data |> projection` — `|>` should highlight as operator
- Open a Parsley file with `data |< insert` — `|<` should highlight as operator
- Open a Parsley file with `<- subquery` — `<-` should highlight as operator

---

### Task 2: Bump VS Code Extension Version
**Files**: `.vscode-extension/package.json`
**Estimated effort**: Small

Steps:
1. Update `version` field from `0.17.0` to `0.18.0`

Tests:
- Verify package.json is valid JSON after edit

---

### Task 3: Update highlight.js Grammar
**Files**: `contrib/highlightjs/parsley.js`
**Estimated effort**: Small

Steps:
1. Add schema and table literals to `AT_LITERAL` variants
2. Add Query DSL literals to `AT_LITERAL` variants
3. Add runtime context literals to `AT_LITERAL` variants
4. Add `computed` to the `keyword` array in `KEYWORDS`
5. Add Query DSL operators pattern to the `contains` array (before or after `SPECIAL_OPERATORS`)

Tests:
- Load `contrib/highlightjs/demo.html` in a browser
- Test `@schema`, `@table` highlighting
- Test `@query`, `@insert`, `@update`, `@delete`, `@transaction` highlighting
- Test `@SEARCH`, `@env`, `@args`, `@params` highlighting
- Test `computed` keyword highlighting
- Test Query DSL operators highlighting

---

### Task 4: Test with Real Parsley Code
**Files**: None (manual testing)
**Estimated effort**: Small

Steps:
1. Find or create sample Parsley files using the new syntax elements
2. Verify VS Code extension highlights them correctly
3. Verify highlight.js demo page renders them correctly

Tests:
- Visual inspection of syntax highlighting in VS Code
- Visual inspection of syntax highlighting in demo.html

---

## Validation Checklist
- [ ] VS Code grammar is valid JSON
- [ ] highlight.js grammar has no syntax errors
- [ ] VS Code extension version bumped
- [ ] All new @ literals highlight correctly in VS Code
- [ ] `computed` keyword highlights correctly in VS Code
- [ ] Query DSL operators highlight correctly in VS Code
- [ ] All new @ literals highlight correctly in highlight.js
- [ ] `computed` keyword highlights correctly in highlight.js
- [ ] Query DSL operators highlight correctly in highlight.js
- [ ] work/BACKLOG.md updated with deferrals (if any)

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-02-10 | Task 1: VS Code Grammar | ✅ Complete | Added special-literals, keywords, operators-special |
| 2026-02-10 | Task 2: VS Code Version | ✅ Complete | Bumped 0.17.0 → 0.18.0 |
| 2026-02-10 | Task 3: highlight.js | ✅ Complete | Added AT_LITERAL variants, keyword, QUERY_OPERATORS |
| | Task 4: Testing | ⬜ Not Started | — |

## Deferred Items
None identified yet.

## Implementation Notes

### VS Code Grammar Additions

**In `special-literals` patterns array, add:**
```json
{
  "comment": "Schema and table literals",
  "name": "support.constant.schema.parsley",
  "match": "@(schema|table)\\b"
},
{
  "comment": "Query DSL literals",
  "name": "support.constant.query.parsley",
  "match": "@(query|insert|update|delete|transaction)\\b"
},
{
  "comment": "Runtime context literals",
  "name": "support.constant.context.parsley",
  "match": "@(SEARCH|env|args|params)\\b"
}
```

**In `keywords` patterns array, add:**
```json
{
  "comment": "Computed export keyword",
  "name": "keyword.other.computed.parsley",
  "match": "\\b(computed)\\b"
}
```

**In `operators-special` patterns array, add:**
```json
{
  "comment": "Query DSL operators",
  "name": "keyword.operator.query.parsley",
  "match": "\\?\\?->|\\?->|\\.->|\\|<|\\|>|<-"
}
```

### highlight.js Grammar Additions

**In `AT_LITERAL` variants array, add:**
```javascript
// Schema and table literals
{ match: /@(schema|table)\b/ },
// Query DSL literals
{ match: /@(query|insert|update|delete|transaction)\b/ },
// Runtime context literals
{ match: /@(SEARCH|env|args|params)\b/ },
```

**In `KEYWORDS.keyword` array, add:**
```javascript
'computed'
```

**In main `contains` array, add (near `SPECIAL_OPERATORS`):**
```javascript
{
  // Query DSL operators
  scope: 'operator',
  match: /\?\?->|\?->|\.->|\|<|\|>|<-/
}
```
