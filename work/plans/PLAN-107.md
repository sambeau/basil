---
id: PLAN-107
feature: FEAT-128
title: "Implementation Plan for Removing Deprecated Parsley Features"
status: draft
created: 2025-02-26
---

# Implementation Plan: FEAT-128

## Overview

Remove all deprecated features from Parsley before the 1.0 release. This is intentionally breaking—we want to find and fix all code that depends on deprecated features.

## Prerequisites

- [x] Deprecation warnings added (FEAT-127)
- [x] Audit of deprecated features complete

## Task Order

The tasks are ordered to minimize churn:
1. First, update tests that use deprecated features
2. Then remove the features (tests will pass)
3. Finally, clean up deprecation infrastructure

---

## Tasks

### Task 1: Update Tests Using Deprecated Features

**Files**: `pkg/parsley/tests/*.go`
**Estimated effort**: Medium (1-2 hours)

Before removing features, update any tests that use them:

1. Search for `@std/table` in tests → convert to `@table` literal
2. Search for `format([` in tests → convert to `.format(` method
3. Search for `<Label`, `<Error`, `<Meta` in tests → convert to lowercase forms
4. Run tests to verify updates work

```bash
grep -r "@std/table" pkg/parsley/tests/
grep -r "format(\[" pkg/parsley/tests/
grep -rE "<(Label|Error|Meta)" pkg/parsley/tests/
```

---

### Task 2: Remove @std/table Module

**Files**: `pkg/parsley/evaluator/stdlib_table.go`
**Estimated effort**: Small (30 minutes)

Steps:
1. In `loadStdlibModule()`, change the `table` case to return an error instead of loading the module
2. Error message: `@std/table is no longer supported. Use @table literal syntax instead.`
3. Remove `"table"` from `getStdlibModules()` map
4. Keep the `TableConstructor`, `TableFromDict`, etc. functions (used by `@table` literal)
5. Remove `tableModuleMeta` and `loadTableModule` function
6. Remove `TableModule` type if no longer needed

Error approach: Use existing error infrastructure with inline message. No new error codes.
```go
return &Error{
    Message: "@std/table is no longer supported. Use @table literal syntax instead.",
    Hints:   []string{"@table [[\"name\", \"age\"], [\"Alice\", 30]]"},
}
```

Tests:
- Verify `import @std/table` returns helpful error
- Verify `@table` literal still works

---

### Task 3: Remove format(array, style) Global Function

**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small (30 minutes)

Steps:
1. Find `"format"` builtin in `getBuiltins()`
2. Remove the array handling branch (keep duration formatting)
3. When first arg is array, return error with migration hint
4. Update `introspect.go` to remove array from format description

Error approach: Return simple error with hint. No new error codes.
```go
if _, ok := args[0].(*Array); ok {
    return &Error{
        Message: "format(array, style) is no longer supported",
        Hints:   []string{"Use array.format(style) method instead"},
    }
}
```

Tests:
- Verify `format([1,2,3], "and")` returns helpful error
- Verify `format(@5d, "en-US")` still works (duration)
- Verify `[1,2,3].format("and")` works

---

### Task 4: Remove Uppercase Form Components

**Files**: `pkg/parsley/evaluator/eval_tags.go`, `pkg/parsley/evaluator/form_components.go`
**Estimated effort**: Small (30 minutes)

Steps:
1. Find handling for `<Label>` in `evalCustomTag` and `evalCustomTagPair`
2. Replace with error return: `<Label> is no longer supported. Use <label @field="..."> instead.`
3. Same for `<Error>` → `<error @field="...">`
4. Same for `<Meta>` → `<val @field="..." @key="help"/>`
5. Keep `evalLabelComponent`, `evalErrorComponent` functions (used by lowercase versions)

Tests:
- Verify `<Label @field="x"/>` returns helpful error
- Verify `<label @field="x"/>` works
- Same for Error and Meta

---

### Task 5: Remove Deprecated AST Fields

**Files**: `pkg/parsley/ast/ast.go`
**Estimated effort**: Medium (1-2 hours)

Remove these fields:
1. `TagLiteral.Spreads` - remove field
2. `TagPairExpression.Props` - remove field
3. `TagPairExpression.Spreads` - remove field
4. `QueryModifier.Fields` - remove field
5. `QueryModifier.Direction` - remove field

For each removal:
1. Remove the field from the struct
2. Search codebase for usages: `grep -r "\.Spreads" pkg/`
3. Update any code that references the removed field
4. Likely locations: parser, evaluator

```bash
grep -rn "\.Spreads" pkg/parsley/
grep -rn "\.Props" pkg/parsley/
grep -rn "QueryModifier.*Fields" pkg/parsley/
grep -rn "QueryModifier.*Direction" pkg/parsley/
```

---

### Task 6: Remove migrate-let-var Command

**Files**: `cmd/pars/main.go`
**Estimated effort**: Small (15 minutes)

Steps:
1. Remove `case "migrate-let-var":` from main() switch
2. Remove `migrateCommand()` function entirely
3. The command is already hidden from help (FEAT-127)

Tests:
- Verify `pars migrate-let-var` returns "unknown command" error

---

### Task 7: Remove Deprecation Warning Infrastructure

**Files**: `pkg/parsley/evaluator/deprecation.go`, various
**Estimated effort**: Small (30 minutes)

Steps:
1. Delete `pkg/parsley/evaluator/deprecation.go`
2. Remove all calls to `emitDeprecationWarning()`:
   - `stdlib_table.go` (already removed in Task 2)
   - `eval_tags.go` (already removed in Task 4)
   - `evaluator.go` (already removed in Task 3)
3. Remove `ResetDeprecationWarningsForTesting()` from test helpers
4. Update tests that check for deprecation warnings

```bash
grep -rn "emitDeprecationWarning" pkg/parsley/
grep -rn "ResetDeprecationWarningsForTesting" pkg/parsley/
grep -rn "DEPRECATION" pkg/parsley/tests/
```

---

### Task 8: Update Documentation

**Files**: `docs/parsley/*.md`, `docs/guide/*.md`
**Estimated effort**: Small (30 minutes)

1. Remove any references to `@std/table` → update to `@table`
2. Remove any references to `format(array, ...)` → update to method form
3. Remove any references to uppercase components
4. Update CHANGELOG.md with breaking changes section

---

### Task 9: Final Cleanup and Verification

**Estimated effort**: Small (30 minutes)

1. Run full test suite: `go test ./...`
2. Run linter: `golangci-lint run`
3. Search for any remaining deprecated references:
   ```bash
   grep -rn "deprecated" pkg/parsley/
   grep -rn "DEPRECATED" pkg/parsley/
   grep -rn "@std/table" .
   grep -rn "migrate-let-var" .
   ```
4. Build CLI and verify: `go build ./cmd/pars`
5. Manual smoke test of common operations

---

## Validation Checklist

- [ ] All tests pass: `go test ./pkg/parsley/...`
- [ ] `import @std/table` returns helpful error
- [ ] `format([...], style)` returns helpful error
- [ ] `<Label>`, `<Error>`, `<Meta>` return helpful errors
- [ ] `pars migrate-let-var` returns unknown command
- [ ] No deprecation warning code remains
- [ ] No references to removed AST fields
- [ ] Linter passes: `golangci-lint run`
- [ ] Documentation updated

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| | Task 1 | ⬚ Not Started | Update tests |
| | Task 2 | ⬚ Not Started | Remove @std/table |
| | Task 3 | ⬚ Not Started | Remove format(array) |
| | Task 4 | ⬚ Not Started | Remove uppercase components |
| | Task 5 | ⬚ Not Started | Remove AST fields |
| | Task 6 | ⬚ Not Started | Remove migrate-let-var |
| | Task 7 | ⬚ Not Started | Remove deprecation infra |
| | Task 8 | ⬚ Not Started | Update documentation |
| | Task 9 | ⬚ Not Started | Final verification |

## Breaking Changes Summary

For CHANGELOG.md:

### Removed

- **`@std/table` module**: Use `@table` literal syntax instead
- **`format(array, style)` function**: Use `array.format(style)` method instead
- **`<Label>` component**: Use `<label @field="...">` instead
- **`<Error>` component**: Use `<error @field="...">` instead
- **`<Meta>` component**: Use `<val @field="..." @key="help"/>` instead
- **`migrate-let-var` CLI command**: Let/var semantics are enforced; no migration tool needed

### Internal Breaking Changes (AST)

- Removed `TagLiteral.Spreads` field (use `Attributes`)
- Removed `TagPairExpression.Props` field (use `Attributes`)
- Removed `TagPairExpression.Spreads` field (use `Attributes`)
- Removed `QueryModifier.Fields` field (use `OrderFields`)
- Removed `QueryModifier.Direction` field (use `OrderFields`)