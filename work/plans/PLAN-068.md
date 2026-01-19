---
id: PLAN-068
feature: FEAT-096
title: "Implementation Plan for Computed Exports"
status: complete
created: 2026-01-19
---

# Implementation Plan: FEAT-096 Computed Exports

## Overview

Implement `export computed` syntax allowing module authors to export values that recalculate on each access. Leverages existing `DynamicAccessor` infrastructure used by `@basil/http` and `@basil/auth`.

## Prerequisites

- [x] FEAT-096 spec reviewed and approved
- [x] `DynamicAccessor` infrastructure exists (BUG-014)
- [ ] Understand current module export/import flow

## Tasks

### Task 1: Add COMPUTED token to lexer
**Files**: `pkg/parsley/lexer/lexer.go`, `pkg/parsley/token/token.go`
**Estimated effort**: Small

Steps:
1. Add `COMPUTED` constant to token types in `token.go`
2. Add `"computed"` to keywords map in `lexer.go`
3. Verify token is recognized in isolation

Tests:
- Lexer tokenizes `computed` as COMPUTED token
- Lexer tokenizes `export computed` as EXPORT, COMPUTED sequence

---

### Task 2: Add ComputedExportStatement AST node
**Files**: `pkg/parsley/ast/ast.go`
**Estimated effort**: Small

Steps:
1. Define `ComputedExportStatement` struct:
   ```go
   type ComputedExportStatement struct {
       Token token.Token  // The 'export' token
       Name  *Identifier  // Export name
       Body  Expression   // Expression (for = form) or BlockExpression (for { } form)
   }
   ```
2. Implement `statementNode()` marker method
3. Implement `TokenLiteral()` method
4. Implement `String()` method for debugging

Tests:
- AST node String() output is readable

---

### Task 3: Parse `export computed` statements
**Files**: `pkg/parsley/parser/parser.go`
**Estimated effort**: Medium

Steps:
1. In `parseExportStatement()`, check for COMPUTED token after EXPORT
2. If COMPUTED:
   - Consume COMPUTED token
   - Parse identifier (the export name)
   - If next token is `=`:
     - Consume `=`
     - Parse expression
     - Create `ComputedExportStatement` with expression body
   - If next token is `{`:
     - Parse block expression
     - Create `ComputedExportStatement` with block body
   - Otherwise: error "expected '=' or '{' after computed export name"
3. Return the `ComputedExportStatement`

Tests:
- Parse `export computed foo = 1 + 2`
- Parse `export computed foo { let x = 1; x + 2 }`
- Parse error: `export computed foo` (missing body)
- Parse error: `export computed = 1` (missing name)
- Parse multiple computed exports in same file

---

### Task 4: Evaluate ComputedExportStatement
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Medium

Steps:
1. Add case for `*ast.ComputedExportStatement` in `Eval()`
2. Create `DynamicAccessor` with:
   - `Name`: the export name
   - `Resolver`: closure that captures module environment and evaluates body
3. Call `env.SetExport(name, accessor)` to register the export
4. Return `NULL`

Key consideration: The resolver closure must capture the module's environment so the body can access module-scoped variables, but resolution should use the access-site environment for `BasilCtx` etc.

Tests:
- Computed export evaluates to DynamicAccessor (not the value)
- Module exports table contains DynamicAccessor

---

### Task 5: Resolve DynamicAccessor on user module import
**Files**: `pkg/parsley/evaluator/eval_expressions.go`, `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Medium

Steps:
1. Find where module exports are accessed (likely `evalDotExpression` or import resolution)
2. When accessing a property on a user module's exports dict:
   - Check if value is `*DynamicAccessor`
   - If so, call `accessor.Resolve(env)` and return result
   - Otherwise, return value as-is
3. Ensure this works for both `import mod from "./file"` and `import {x} from "./file"`

Note: `StdlibModuleDict` already handles this for stdlib. Need to ensure user module dicts get same treatment.

Tests:
- `import {foo} from "./computed.pars"` where `foo` is computed → resolves on access
- `import mod from "./computed.pars"; mod.foo` → resolves on access
- Computed export accessed twice returns fresh values each time

---

### Task 6: Handle destructuring imports with DynamicAccessor
**Files**: `pkg/parsley/evaluator/evaluator.go` (import handling)
**Estimated effort**: Small

Steps:
1. When destructuring `import {x} from "..."`:
   - Bind `x` to the `DynamicAccessor` itself, not the resolved value
   - Resolution happens at access time, not import time
2. Verify this matches behavior for `@basil/http` imports

Tests:
- `import {computed_val} from "./mod.pars"` → `computed_val` is DynamicAccessor
- Accessing `computed_val` resolves it
- Two accesses return potentially different values

---

### Task 7: Integration tests
**Files**: `pkg/parsley/tests/computed_export_test.go` (new file)
**Estimated effort**: Medium

Steps:
1. Create test file with comprehensive scenarios
2. Test expression form with various expressions
3. Test block form with multi-statement bodies
4. Test computed export accessing module variables
5. Test computed export calling functions
6. Test error propagation (computed body throws)
7. Test consumer caching via assignment
8. Test re-export of computed values

Test cases:
- Basic expression: `export computed x = 42`
- Expression with closure: `let n = 1; export computed x = n + 1`
- Block form: `export computed x { let a = 1; a + 2 }`
- Multiple computed exports
- Computed export calling a function
- Computed export with side effects (counter)
- Error in computed body propagates
- `try` catches computed errors
- Re-export preserves computed nature
- Namespace import: `import * as mod from "./computed.pars"; mod.x`

---

### Task 8: Documentation
**Files**: `docs/parsley/reference.md`, `docs/parsley/CHEATSHEET.md`
**Estimated effort**: Small

Steps:
1. Add `export computed` to reference.md under Modules section
2. Add note to CHEATSHEET.md about computed exports (pitfall: they recalculate)
3. Add example to examples/parsley/ directory

Tests:
- Documentation examples are syntactically valid

---

## Validation Checklist

- [ ] All tests pass: `make test`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] Reference documentation updated
- [ ] Cheatsheet updated
- [ ] work/BACKLOG.md updated with deferrals (if any)

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-01-19 | Task 1: Lexer | ✅ Complete | Added COMPUTED token |
| 2026-01-19 | Task 2: AST | ✅ Complete | Added ComputedExportStatement |
| 2026-01-19 | Task 3: Parser | ✅ Complete | Parses both expression and block form |
| 2026-01-19 | Task 4: Evaluator | ✅ Complete | Creates DynamicAccessor |
| 2026-01-19 | Task 5: Import resolution | ✅ Complete | DynamicAccessor resolved in dict dot access |
| 2026-01-19 | Task 6: Destructuring | ✅ Complete | Works via evalIdentifier |
| 2026-01-19 | Task 7: Integration tests | ✅ Complete | 10 test cases (1 skipped) |
| 2026-01-19 | Task 8: Documentation | ✅ Complete | Updated reference.md and CHEATSHEET.md |

## Deferred Items

Items to add to work/BACKLOG.md after implementation:
- `export computed` with optional memoization (e.g., `export cached computed x = ...`) — Keep initial version simple
- IDE/editor support for computed exports — Needs syntax highlighting updates
- DevTools visibility into computed exports — Show which exports are computed vs static

## Implementation Notes

### Environment Handling

The trickiest part is getting the environment right for the resolver:

```go
Resolver: func(accessEnv *Environment) Object {
    // Need to evaluate body in module's lexical environment
    // but with access to accessEnv's BasilCtx for @basil/http etc.
    // 
    // Current DynamicAccessor for @basil/http already does this:
    // it gets BasilCtx from the access environment
}
```

Review how `@basil/http` DynamicAccessors work in `stdlib_basil.go` before implementing.

### Module Dict Handling

User modules return a `Dictionary` for their exports. Currently `evalDotExpression` has special handling for `StdlibModuleDict` to resolve DynamicAccessors. Need to either:

1. Add same handling for regular `Dictionary` (may have performance implications)
2. Wrap user module exports in a new `UserModuleDict` type
3. Mark dictionaries that may contain DynamicAccessors

Option 1 is simplest. The check `if accessor, ok := val.(*DynamicAccessor)` is cheap.

### Testing Strategy

Start with a simple end-to-end test to prove the concept works, then fill in edge cases. This ensures the full pipeline (lexer → parser → AST → evaluator → resolution) is wired up correctly before diving into edge cases.
