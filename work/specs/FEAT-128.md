---
id: FEAT-128
title: "Remove Deprecated Parsley Features for 1.0"
status: draft
priority: high
created: 2025-02-26
author: "@ai"
---

# FEAT-128: Remove Deprecated Parsley Features for 1.0

## Summary

Remove all deprecated features, code paths, and APIs from Parsley before the 1.0 release. This is a breaking change release that cleans up technical debt and establishes a clean API surface for 1.0.

## User Story

As a Parsley maintainer, I want to remove deprecated code before 1.0 so that:
- The codebase is simpler and easier to maintain
- Users don't rely on deprecated features that will eventually break
- The 1.0 API surface is clean and intentional
- We identify any code still depending on deprecated features

## Philosophy

**"Break stuff so we have to fix stuff."**

This is intentionally a disruptive change. By removing deprecated features now:
1. We find all the places that depend on them
2. We fix or update those places
3. We ship a clean 1.0 with no legacy baggage

## Items to Remove

### 1. Deprecated Builtins & Modules

| Item | Replacement | Location |
|------|-------------|----------|
| `@std/table` module | `@table` literal | `pkg/parsley/evaluator/stdlib_table.go` |
| `format(array, style)` global | `array.format(style)` method | `pkg/parsley/evaluator/evaluator.go` |

### 2. Deprecated Form Components

| Component | Replacement | Location |
|-----------|-------------|----------|
| `<Label>` (uppercase) | `<label @field>` | `pkg/parsley/evaluator/eval_tags.go` |
| `<Error>` (uppercase) | `<error @field>` | `pkg/parsley/evaluator/eval_tags.go` |
| `<Meta @field>` | `<val @field @key="help"/>` | `pkg/parsley/evaluator/eval_tags.go` |

### 3. Deprecated AST Fields

| Field | Type | Replacement | Location |
|-------|------|-------------|----------|
| `TagLiteral.Spreads` | `[]*SpreadExpr` | `Attributes` | `pkg/parsley/ast/ast.go` |
| `TagPairExpression.Props` | `string` | `Attributes` | `pkg/parsley/ast/ast.go` |
| `TagPairExpression.Spreads` | `[]*SpreadExpr` | `Attributes` | `pkg/parsley/ast/ast.go` |
| `QueryModifier.Fields` | `[]string` | `OrderFields` | `pkg/parsley/ast/ast.go` |
| `QueryModifier.Direction` | `string` | `OrderFields` | `pkg/parsley/ast/ast.go` |

### 4. CLI Cleanup

| Item | Action | Location |
|------|--------|----------|
| `migrate-let-var` command | Remove entirely | `cmd/pars/main.go` |

### 5. Deprecation Warning Infrastructure

| Item | Action | Location |
|------|--------|----------|
| `deprecation.go` | Remove file | `pkg/parsley/evaluator/deprecation.go` |
| Deprecation warning calls | Remove all | Various files |

## Acceptance Criteria

- [ ] `@std/table` import returns error suggesting `@table` literal
- [ ] `format(array, style)` returns error suggesting `array.format(style)`
- [ ] `<Label>`, `<Error>`, `<Meta>` return errors with migration hints
- [ ] Deprecated AST fields are removed (breaking change for any external parsers)
- [ ] `migrate-let-var` command is completely removed
- [ ] All deprecation warning code is removed
- [ ] All tests pass after updating test files that use deprecated features

**Note:** Errors for removed features use inline messages, not new error catalog entries. We don't want to add permanent error codes for pre-1.0 features that no longer exist.

## Design Decisions

### Hard Errors vs Soft Warnings

**Decision:** Return hard errors, not warnings.

When users try to use deprecated features, they should get a clear error with a hint about what to use instead. This forces immediate migration rather than allowing continued use.

Example:
```
Error: @std/table is no longer supported
  hint: Use @table literal syntax instead: @table [[...], [...]]
```

### AST Field Removal

**Decision:** Remove deprecated AST fields entirely.

This is a breaking change for anyone who has written tools that parse Parsley AST directly. However:
- The AST is internal implementation detail
- No public API guarantees exist for AST structure
- Keeping dead fields adds confusion and maintenance burden

### Deprecation Infrastructure Removal

**Decision:** Remove `deprecation.go` and all warning calls.

Once deprecated features are removed, the deprecation warning system serves no purpose. Remove it to keep the codebase clean. It can be re-added if needed for future deprecations.

## Migration Guide

### For `@std/table` Users

**Before:**
```parsley
let {table} = import @std/table
let t = table([["name", "age"], ["Alice", 30]])
```

**After:**
```parsley
let t = @table [["name", "age"], ["Alice", 30]]
// Or from array of dicts:
let t = @table [{name: "Alice", age: 30}]
```

### For `format(array, style)` Users

**Before:**
```parsley
format(["apple", "banana", "cherry"], "and")
```

**After:**
```parsley
["apple", "banana", "cherry"].format("and")
```

### For Uppercase Component Users

**Before:**
```parsley
<form @record={user}>
  <Label @field="name"/>
  <Error @field="name"/>
  <Meta @field="name"/>
</form>
```

**After:**
```parsley
<form @record={user}>
  <label @field="name"/>
  <error @field="name"/>
  <val @field="name" @key="help"/>
</form>
```

## Testing Strategy

1. **Update existing tests** that use deprecated features to use new syntax
2. **Add error tests** verifying deprecated features return helpful errors
3. **Run full test suite** to find any missed usages
4. **Manual testing** of common workflows

## Risks

| Risk | Mitigation |
|------|------------|
| External tools depend on AST fields | AST is internal; document as breaking change |
| Users have code using deprecated features | Error messages include migration hints |
| Tests using deprecated features | Update tests as part of this work |

## Implementation Notes

*Added during/after implementation*

## Related

- Plan: `work/plans/PLAN-107.md`
- Audit: `work/reports/1.0-READINESS-AUDIT.md`
- Deprecation warnings: FEAT-127 (added warnings that we're now removing)