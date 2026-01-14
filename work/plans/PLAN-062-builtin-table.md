---
id: PLAN-062
feature: FEAT-087
title: "Implementation Plan for Builtin Table Type"
status: draft
created: 2026-01-13
---

# Implementation Plan: FEAT-087 Builtin Table Type

## Overview

Promote Parsley's `Table` type to a builtin with:
1. `Table()` constructor available without import
2. `@table` literal syntax with parse-time validation
3. `@table(Schema)` for typed tables
4. @schema extensions: nullable (`?`) and defaults (`= value`)
5. Copy-on-chain semantics
6. CSV/Database return Table instead of Array

## Prerequisites

- [x] FEAT-087 specification complete
- [x] Design document reviewed
- [ ] Feature branch created: `feat/FEAT-087-builtin-table`

## Implementation Phases

### Phase 1: Foundation (Table as Builtin)
Effort: Medium | Tests: Required

### Phase 2: @schema Extensions  
Effort: Medium | Tests: Required

### Phase 3: @table Literal Syntax
Effort: Large | Tests: Required

### Phase 4: Copy-on-Chain
Effort: Medium | Tests: Required

### Phase 5: CSV/Database Integration
Effort: Medium | Tests: Required

### Phase 6: Backward Compatibility & Docs
Effort: Small | Tests: Verification

---

## Tasks

### Task 1: Add Table struct fields for schema and chain tracking
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small

Steps:
1. Add `Schema *DSLSchema` field to Table struct
2. Add `isChainCopy bool` field to Table struct (internal)
3. Update `Table.Copy()` to include new fields
4. Update `Table.Inspect()` to show schema name if present

Tests:
- Existing table tests still pass
- Table with schema shows schema in Inspect()

---

### Task 2: Register Table() as builtin constructor
**Files**: `pkg/parsley/evaluator/builtins.go`, `pkg/parsley/evaluator/stdlib_table.go`
**Estimated effort**: Small

Steps:
1. Add `Table` to builtins map in `builtins.go`
2. Create `builtinTable()` function that wraps existing table construction
3. Validate input is array of dictionaries
4. Validate rectangular shape (all rows have same keys)
5. Return descriptive errors per E1 spec

Tests:
- `Table([{a:1},{a:2}])` works without import
- `Table([])` returns empty table
- `Table("string")` returns TABLE_NOT_ARRAY error
- `Table([{a:1},{b:2}])` returns TABLE_COLUMN_MISMATCH error

---

### Task 3: Add nullable (?) support to @schema ✓ COMPLETE
**Files**: `pkg/parsley/evaluator/stdlib_dsl_schema.go`, `pkg/parsley/ast/ast.go`, `pkg/parsley/parser/parser.go`
**Estimated effort**: Medium

Steps:
1. ✓ Add `Nullable bool` field to `ast.SchemaField` struct
2. ✓ Add `Nullable bool` field to `DSLSchemaField` struct
3. ✓ Update parser's `parseSchemaField()` to parse `type?` syntax
4. ✓ Update `evalSchemaDeclaration()` to set `Required: !Nullable`
5. ✓ Update SQL generation to include NOT NULL for non-nullable fields

Tests:
- ✓ `@schema { field: string? }` parses correctly
- ✓ Nullable field has `Required: false`, `Nullable: true`
- ✓ SQL generation: non-nullable → includes NOT NULL

---

### Task 4: Add default value support to @schema ✓ COMPLETE
**Files**: `pkg/parsley/evaluator/stdlib_dsl_schema.go`, `pkg/parsley/ast/ast.go`, `pkg/parsley/parser/parser.go`
**Estimated effort**: Medium

Steps:
1. ✓ Add `DefaultValue Expression` field to `ast.SchemaField`
2. ✓ Add `DefaultValue Object` and `DefaultExpr string` fields to `DSLSchemaField`
3. ✓ Update parser's `parseSchemaField()` to parse `= expression` syntax
4. ✓ Update `evalSchemaDeclaration()` to evaluate and store defaults
5. ✓ Update SQL generation to include DEFAULT clause
6. ✓ Update `evalDSLSchemaProperty` to expose nullable and default in .fields property

Tests:
- ✓ `@schema { name: string = "default" }` parses
- ✓ `type? = value` (combined nullable and default) works
- ✓ SQL generation includes DEFAULT clause
- ✓ Integer and boolean defaults work

---

### Task 5: Add @table token to lexer ✓ COMPLETE
**Files**: `pkg/parsley/lexer/lexer.go`
**Estimated effort**: Small

Steps:
1. ✓ Add `TABLE_LITERAL` token type (alongside `SCHEMA`, etc.)
2. ✓ Add to DSL keywords detection in `detectAtLiteralType()`
3. ✓ Add case handling for TABLE_LITERAL
4. ✓ Add token string representation

Tests:
- ✓ `@table` lexes as TABLE_LITERAL token

---

### Task 6: Add TableLiteral AST node ✓ COMPLETE
**Files**: `pkg/parsley/ast/ast.go`
**Estimated effort**: Small

Steps:
1. ✓ Define `TableLiteral` struct with Token, Schema, Rows, Columns fields
2. ✓ Implement `TokenLiteral()`, `String()` methods
3. ✓ Implement `expressionNode()` marker

Tests:
- ✓ AST node compiles and integrates

---

### Task 7: Parse @table literal in parser ✓ COMPLETE
**Files**: `pkg/parsley/parser/parser.go`
**Estimated effort**: Large

Steps:
1. ✓ Register prefix parser for TABLE_LITERAL
2. ✓ Add `parseTableLiteral()` function
3. ✓ Handle `@table [...]` — infer columns from first row
4. ✓ Handle `@table(SchemaName) [...]` — parse schema reference
5. ✓ Validate all elements are dictionary literals (parse error)
6. ✓ Validate all rows have same keys as first row (parse error)
7. ✓ Store column order from first row's keys

Tests:
- ✓ `@table [{a:1},{a:2}]` parses correctly
- ✓ `@table []` parses as empty table
- ✓ `@table [{a:1},{b:2}]` gives parse error with row number
- ✓ `@table(Schema) [...]` parses with schema reference

---

### Task 8: Evaluate @table literal ✓ COMPLETE
**Files**: `pkg/parsley/evaluator/evaluator.go`, `pkg/parsley/evaluator/stdlib_table.go`
**Estimated effort**: Medium

Steps:
1. ✓ Add case for `*ast.TableLiteral` in `Eval()`
2. ✓ Create `evalTableLiteral()` function
3. ✓ Evaluate each row dictionary
4. ✓ If schema specified, look it up and apply defaults
5. ✓ Construct Table with Rows, Columns, Schema

Tests:
- ✓ `@table [{a:1}]` evaluates to Table with 1 row
- ✓ `@table(Schema) [...]` applies defaults
- ✓ `@table(Unknown) [...]` returns error

---

### Task 9: Implement copy-on-chain in table methods ✓ COMPLETE
**Files**: `pkg/parsley/evaluator/stdlib_table.go`, `pkg/parsley/evaluator/evaluator.go`, `pkg/parsley/evaluator/eval_expressions.go`
**Estimated effort**: Medium

Steps:
1. ✓ Create `ensureChainCopy(table *Table) *Table` helper in evaluator.go
2. ✓ If `isChainCopy` is true, return same table
3. ✓ Otherwise, create deep copy with `isChainCopy = true`
4. ✓ Update all mutating methods to use `ensureChainCopy()`:
   - `.where()`, `.orderBy()`, `.select()`, `.limit()`
5. ✓ Create `endChain(table *Table)` helper (sets `isChainCopy = false`)
6. ✓ Create `endTableChain(obj Object)` helper for general use
7. ✓ Call `endTableChain()` when table is:
   - Assigned to variable (in evaluator.go LetStatement/AssignmentStatement)
   - Passed as function argument (in eval_expressions.go extendFunctionEnv)

Tests:
- ✓ Original table unchanged after chain
- ✓ Long chain preserves original
- ✓ Assignment breaks chain
- ✓ Two chains from same source are independent
- ✓ Function argument ends chain

---

### Task 10: Make CSV return Table
**Files**: `pkg/parsley/evaluator/eval_file_io.go`, `pkg/parsley/evaluator/builtins.go`
**Estimated effort**: Small

Steps:
1. Update `builtinCSV()` to return Table instead of Array
2. Update `parseCSV()` string method to return Table
3. Use CSV headers as Columns
4. Each parsed row becomes Dictionary in Rows

Tests:
- `CSV("file.csv")` returns Table
- `"a,b\n1,2".parseCSV()` returns Table
- `table.toArray()` still works for backward compat

---

### Task 11: Make database queries return Table
**Files**: `pkg/parsley/evaluator/eval_database.go`
**Estimated effort**: Medium

Steps:
1. Update `TableBinding.all()` to return Table
2. Update `TableBinding.where()` to return Table  
3. Keep `TableBinding.find()` returning Dictionary (single row)
4. Update raw SQL `<=?=>` to return Table
5. Attach schema to returned Table (from table binding)
6. Build schema from SQL column types for raw queries

Tests:
- `Users.all()` returns Table with schema
- `Users.where(...)` returns Table
- `Users.find(id)` returns Dictionary
- Raw SQL returns Table

---

### Task 12: Add .schema property and .toArray() method
**Files**: `pkg/parsley/evaluator/stdlib_table.go`
**Estimated effort**: Small

Steps:
1. Add `.schema` property access (returns schema or null)
2. Verify `.toArray()` method works correctly
3. Add `.copy()` method for explicit non-chain copy
4. Ensure `.columns` and `.length` properties work

Tests:
- `table.schema` returns schema object or null
- `table.toArray()` returns array of dictionaries
- `table.copy()` returns independent copy
- `table.columns` returns column names
- `table.length` returns row count

---

### Task 13: Update @std/table as alias
**Files**: `pkg/parsley/evaluator/stdlib_table.go`
**Estimated effort**: Small

Steps:
1. Keep `loadTableModule()` working
2. Make it return the builtin Table
3. Add deprecation note in code comments (no runtime warning yet)

Tests:
- `import @std/table` still works
- `let {table} = import @std/table` still works
- Existing tests using @std/table pass unchanged

---

### Task 14: Documentation updates
**Files**: `docs/parsley/reference.md`, `docs/parsley/CHEATSHEET.md`, `docs/guide/*.md`
**Estimated effort**: Medium

Steps:
1. Add `@table` to literal syntax in reference.md
2. Document Table type and methods
3. Document @schema nullable and default syntax
4. Update CSV examples to show Table return
5. Add "Working with Tables" section to guide
6. Update CHEATSHEET with @table syntax

Tests:
- Documentation builds/renders correctly
- Examples in docs are valid Parsley

---

## Validation Checklist

- [ ] All tests pass: `make test`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] New tests added for all acceptance criteria
- [ ] Backward compatibility verified
- [ ] Documentation updated
- [ ] work/BACKLOG.md updated with deferrals (if any)

## Risk Assessment

| Risk | Mitigation |
|------|------------|
| Breaking change: CSV returns Table not Array | `.toArray()` provides escape hatch; most code iterates anyway |
| Copy-on-chain complexity | Thorough testing; can fall back to always-copy if issues |
| Parser changes for @table | Follow existing @schema pattern; incremental testing |
| Database schema inference | Start simple (column names only); enhance in V2 |

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-01-13 | Plan created | ✅ Complete | — |
| 2026-01-13 | Task 1: Table struct fields | ✅ Complete | Added Schema, isChainCopy |
| 2026-01-13 | Task 2: Builtin constructor | ✅ Complete | Table() works without import |
| 2026-01-13 | Task 12: Properties/methods | ✅ Complete | Added .length, .schema |
| 2026-01-14 | Task 3: Nullable support | ✅ Complete | type? syntax, SQL NOT NULL |
| 2026-01-14 | Task 4: Default support | ✅ Complete | = value syntax, SQL DEFAULT |
| 2026-01-14 | Task 5: @table token | ✅ Complete | TABLE_LITERAL in lexer |
| 2026-01-14 | Task 6: TableLiteral AST | ✅ Complete | AST node with Schema, Rows, Columns |
| 2026-01-14 | Task 7: Parse @table | ✅ Complete | Parse-time column validation |
| 2026-01-14 | Task 8: Eval @table | ✅ Complete | Schema lookup, defaults applied |
| 2026-01-14 | Task 9: Copy-on-chain | ✅ Complete | ensureChainCopy/endChain helpers |
| | Task 10: CSV returns Table | ⬜ Not started | — |
| | Task 11: DB returns Table | ⬜ Not started | — |
| | Task 13: @std/table alias | ⬜ Not started | — |
| | Task 14: Documentation | ⬜ Not started | — |

## Suggested Implementation Order

```
Phase 1 (Foundation):    Task 1 → Task 2 → Task 12
Phase 2 (@schema):       Task 3 → Task 4
Phase 3 (@table):        Task 5 → Task 6 → Task 7 → Task 8
Phase 4 (Copy-on-chain): Task 9
Phase 5 (Integration):   Task 10 → Task 11 → Task 13
Phase 6 (Docs):          Task 14
```

**Recommended checkpoints:**
- After Phase 1: Basic `Table()` works, run tests
- After Phase 2: @schema extensions work, run tests
- After Phase 3: @table literals work, run tests
- After Phase 4: Copy-on-chain works, run tests
- After Phase 5: Full integration, run all tests
- After Phase 6: Documentation complete, ready for merge

## Deferred Items (V2)

Items to add to work/BACKLOG.md after implementation:
- Lazy evaluation for table chains
- Database query pushdown optimization
- Columnar internal representation
- @std/table deprecation warning

## Files Summary

| File | Changes |
|------|---------|
| `pkg/parsley/evaluator/evaluator.go` | Table struct fields |
| `pkg/parsley/evaluator/builtins.go` | Table() builtin |
| `pkg/parsley/evaluator/stdlib_dsl_schema.go` | Nullable, defaults |
| `pkg/parsley/lexer/lexer.go` | @table token |
| `pkg/parsley/ast/ast.go` | TableLiteral node |
| `pkg/parsley/parser/parser.go` | @table parsing |
| `pkg/parsley/evaluator/eval_expressions.go` | @table evaluation |
| `pkg/parsley/evaluator/stdlib_table.go` | Copy-on-chain, methods, alias |
| `pkg/parsley/evaluator/eval_file_io.go` | CSV returns Table |
| `pkg/parsley/evaluator/eval_database.go` | DB returns Table |
| `docs/parsley/reference.md` | @table syntax |
| `docs/parsley/CHEATSHEET.md` | Quick reference |
| `docs/guide/*.md` | Examples, guides |

## Estimated Total Effort

| Phase | Effort | Tasks |
|-------|--------|-------|
| Phase 1 | Small | 1, 2, 12 |
| Phase 2 | Medium | 3, 4 |
| Phase 3 | Large | 5, 6, 7, 8 |
| Phase 4 | Medium | 9 |
| Phase 5 | Medium | 10, 11, 13 |
| Phase 6 | Medium | 14 |
| **Total** | **Large** | 14 tasks |

Estimated time: 2-3 focused sessions
