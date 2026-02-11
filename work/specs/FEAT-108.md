---
id: FEAT-108
title: "Syntax Highlighter Updates: VS Code and highlight.js"
status: draft
priority: high
created: 2025-01-15
author: "@human"
blocking: true
---

# FEAT-108: Syntax Highlighter Updates: VS Code and highlight.js

## Summary
Update the existing VS Code TextMate grammar and highlight.js grammar to support recent Parsley language features. These grammars are out of date and missing support for schema declarations, the Query DSL, computed exports, and other recent additions.

## User Story
As a Parsley developer using VS Code, I want all language features to be syntax highlighted so that I can read and write code effectively.

As a documentation author or AI tool developer, I want highlight.js to support all Parsley syntax so that code examples render correctly.

## Acceptance Criteria

### VS Code Extension
- [ ] `@schema`, `@table`, `@query`, `@insert`, `@update`, `@delete`, `@transaction` are highlighted
- [ ] `@SEARCH`, `@env`, `@args`, `@params` are highlighted
- [ ] `computed` keyword (after `export`) is highlighted
- [ ] Query DSL operators (`|<`, `|>`, `?->`, `??->`, `.->`, `<-`) are highlighted
- [ ] Extension version bumped in `package.json`

### highlight.js Grammar
- [ ] Same additions as VS Code
- [ ] Works in `demo.html` test page
- [ ] Can be used in documentation sites and AI chat UIs

## Design Decisions

- **Scope limited to existing grammars**: Tree-sitter grammar is a separate, larger effort tracked in FEAT-109.

- **Pattern additions only**: No structural changes to the grammars — just add missing patterns to existing rule sets.

- **Test with real code**: Verify highlighting using actual Parsley files that use these features.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components

| Component | Location | Effort |
|-----------|----------|--------|
| VS Code Grammar | `.vscode-extension/syntaxes/parsley.tmLanguage.json` | 2-3 hours |
| VS Code Package | `.vscode-extension/package.json` | 5 minutes |
| highlight.js | `contrib/highlightjs/parsley.js` | 1-2 hours |

### Dependencies
- Depends on: None
- Blocks: None (but should be done before 1.0 Alpha)

### Missing Syntax Elements

From `pkg/parsley/lexer/lexer.go`, the following are **not** in current grammars:

**Literals:**
- `@schema` — Schema declarations
- `@table` — Table literals
- `@query` — Query DSL entry
- `@insert` — Insert operations
- `@update` — Update operations
- `@delete` — Delete operations
- `@transaction` — Transaction blocks
- `@SEARCH` — Search literal
- `@env` — Environment variables
- `@args` — Command-line arguments
- `@params` — URL parameters

**Keywords:**
- `computed` — In context of `export computed`

**Operators (Query DSL):**
- `|<` — Write operator (insert/update)
- `|>` — Read projection
- `?->` — Single row terminal
- `??->` — Multi row terminal
- `.->` — Count terminal
- `<-` — Correlated subquery source

---

## Implementation Plan

### VS Code Grammar Update

**File: `.vscode-extension/syntaxes/parsley.tmLanguage.json`**

Add to `special-literals` patterns:
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

Add to `keywords` patterns:
```json
{
  "name": "keyword.other.computed.parsley",
  "match": "\\b(computed)\\b"
}
```

Add to `operators-special` patterns:
```json
{
  "comment": "Query DSL operators",
  "name": "keyword.operator.query.parsley",
  "match": "\\?\\?->|\\?->|\\.->|\\|<|\\|>|<-"
}
```

**File: `.vscode-extension/package.json`**

Bump version number for marketplace update.

### highlight.js Grammar Update

**File: `contrib/highlightjs/parsley.js`**

Add to `AT_LITERAL` variants:
```javascript
// Schema and table literals
{ match: /@(schema|table)\b/ },
// Query DSL literals
{ match: /@(query|insert|update|delete|transaction)\b/ },
// Runtime context literals
{ match: /@(SEARCH|env|args|params)\b/ },
```

Add `computed` to keywords:
```javascript
keyword: [
  'fn', 'function', 'let', 'for', 'in', 'as',
  'if', 'else', 'return', 'export', 'try', 'import',
  'check', 'stop', 'skip', 'computed',  // Added
  'and', 'or', 'not'
],
```

Add Query DSL operators:
```javascript
{
  // Query DSL operators
  scope: 'operator',
  match: /\?\?->|\?->|\.->|\|<|\|>|<-/
}
```

---

## Test Plan

### VS Code Extension
| Test | Expected |
|------|----------|
| Open file with `@schema User { }` | `@schema` highlighted as constant |
| Open file with `@query(db)` | `@query` highlighted as constant |
| Open file with `@table [...]` | `@table` highlighted as constant |
| Open file with `@transaction { }` | `@transaction` highlighted as constant |
| Open file with `export computed x = 1` | `computed` highlighted as keyword |
| Open file with `data ?-> result` | `?->` highlighted as operator |
| Open file with `@env.HOME` | `@env` highlighted as constant |
| Open file with `@args` | `@args` highlighted as constant |

### highlight.js
| Test | Expected |
|------|----------|
| Render `@schema` in demo.html | Highlighted correctly |
| Render `@table` in demo.html | Highlighted correctly |
| Render `@query`, `@insert`, `@update`, `@delete` | Highlighted correctly |
| Render `@transaction { }` in demo.html | Highlighted correctly |
| Render Query DSL operators | Operators highlighted |
| Render `export computed` | `computed` highlighted as keyword |

---

## Implementation Notes
*To be added during implementation*

## Related
- Report: `work/reports/PARSLEY-1.0-ALPHA-READINESS.md` (Section 5, Appendix B)
- VS Code Extension: `.vscode-extension/`
- highlight.js: `contrib/highlightjs/`
- Lexer (source of truth): `pkg/parsley/lexer/lexer.go`
- Tree-sitter grammar: FEAT-109 (separate effort)