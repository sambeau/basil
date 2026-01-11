---
id: PLAN-061
title: "Parsley Documentation Overhaul"
status: draft
created: 2026-01-11
---

# Implementation Plan: Parsley Documentation Overhaul

## Overview

The Parsley documentation has drifted from the implementation. This plan creates accurate, verified documentation by auditing the source code and building canonical references from scratch.

**Goals:**
1. Create authoritative feature inventory from code
2. Verify and fix CHEATSHEET.md (mostly accurate, smaller scope)
3. Build reference.md from scratch using inventory
4. Ensure all examples are executable and tested

## Phase 1: Feature Inventory

**Objective:** Audit source code to create complete inventory of all language features.

### Task 1.1: Lexer Audit
**Files**: `pkg/parsley/lexer/`
**Estimated effort**: Medium

Extract:
- All operators and their token types
- All literal formats (paths, dates, money, etc.)
- All keywords
- String types (regular, template, raw)

Deliverable: `work/parsley/INVENTORY-lexer.md`

---

### Task 1.2: Parser Audit
**Files**: `pkg/parsley/parser/`
**Estimated effort**: Medium

Extract:
- Grammar rules for all constructs
- Precedence table
- Special syntax (destructuring, spreading, etc.)
- Expression vs statement distinctions

Deliverable: `work/parsley/INVENTORY-parser.md`

---

### Task 1.3: Evaluator Methods Audit
**Files**: `pkg/parsley/evaluator/`
**Estimated effort**: Large

Extract:
- All methods by type (string, array, dict, number, datetime, money, etc.)
- Method signatures and return types
- Method behaviors

Deliverable: `work/parsley/INVENTORY-methods.md`

---

### Task 1.4: Builtins Audit
**Files**: `pkg/parsley/evaluator/builtins*.go`
**Estimated effort**: Medium

Extract:
- All builtin functions
- Signatures and return types
- Factory functions (JSON, CSV, YAML, etc.)

Deliverable: `work/parsley/INVENTORY-builtins.md`

---

### Task 1.5: Standard Library Audit
**Files**: `pkg/parsley/evaluator/stdlib_*.go`
**Estimated effort**: Large

Extract:
- All @std/* modules
- Exported functions per module
- Signatures and behaviors

Deliverable: `work/parsley/INVENTORY-stdlib.md`

---

### Task 1.6: Operators Audit
**Files**: `pkg/parsley/evaluator/eval_infix.go`, `pkg/parsley/evaluator/eval_prefix.go`
**Estimated effort**: Medium

Extract:
- All operators by category
- Type-specific behaviors (overloading)
- Precedence

Deliverable: `work/parsley/INVENTORY-operators.md`

---

## Phase 2: Verify CHEATSHEET.md

**Objective:** Test every example in CHEATSHEET.md against `./pars`, fix errors.

### Task 2.1: Section-by-Section Verification
**Files**: `docs/parsley/CHEATSHEET.md`
**Estimated effort**: Large

Process:
1. Extract each code example
2. Run in `./pars` or create test script
3. Verify output matches documentation
4. Fix discrepancies
5. Mark section as verified

Sections to verify:
- [ ] Quick Pitfalls (1-10)
- [ ] Syntax Quick Reference
- [ ] Key Language Features (1-5)
- [ ] File I/O
- [ ] Money Type
- [ ] Common Patterns
- [ ] Standard Library (@std)
- [ ] Basil Server sections
- [ ] Parts
- [ ] Method Reference
- [ ] Quick Examples

---

## Phase 3: Build reference.md from Scratch

**Objective:** Create comprehensive, accurate reference using inventory as source of truth.

### Task 3.1: Reference Structure
**Files**: `docs/parsley/reference.md`
**Estimated effort**: Small

Define sections:
1. Lexical Structure (from lexer inventory)
2. Expressions & Operators (from parser + evaluator)
3. Statements
4. Data Types & Methods
5. Builtins
6. Standard Library
7. I/O & File Operations
8. Error Handling

---

### Task 3.2: Write Reference Sections
**Files**: `docs/parsley/reference.md`
**Estimated effort**: Large

For each section:
1. Pull from inventory documents
2. Write clear grammar snippets
3. Add tested examples
4. Cross-reference related sections

---

### Task 3.3: Grammar Snippets
**Files**: `docs/parsley/reference.md`
**Estimated effort**: Medium

Ensure all grammar is accurate:
```
for_expr     := "for" "(" binding "in" expr ")" block
binding      := IDENT | index_binding | destructure
index_binding := IDENT "," IDENT
```

---

## Phase 4: CI Integration (Optional)

**Objective:** Prevent documentation drift with automated testing.

### Task 4.1: Example Extraction Tool
**Estimated effort**: Medium

Create tool to:
1. Extract code blocks from markdown
2. Run through `./pars`
3. Compare output to documented expectations
4. Report failures

---

### Task 4.2: CI Pipeline
**Estimated effort**: Small

Add GitHub Action to run example tests on:
- PRs touching `pkg/parsley/`
- PRs touching `docs/parsley/`

---

## Validation Checklist
- [ ] All inventory documents complete
- [ ] CHEATSHEET.md fully verified
- [ ] reference.md rebuilt from inventory
- [ ] All examples tested
- [ ] No outstanding TODOs in docs

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-01-11 | Plan created | ✅ Complete | — |

## Deferred Items
Items to add to work/BACKLOG.md after implementation:
- CI doc testing (if Phase 4 not completed)
- Video/tutorial content based on new docs

## Notes

**Source of Truth Hierarchy:**
1. Source code (lexer, parser, evaluator)
2. Inventory documents (derived from code)
3. reference.md (derived from inventory)
4. CHEATSHEET.md (verified against reference)
5. Guides/tutorials (derived from reference)

**Testing Strategy:**
- Every code example must be runnable
- Use `./pars` for standalone examples
- Use `./basil --dev` for Basil-specific examples
- Document any examples that require special context
