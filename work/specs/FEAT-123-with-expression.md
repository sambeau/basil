---
id: FEAT-123
title: "`with` Expression for Scoped Field Access"
status: implemented
priority: medium
created: 2025-01-21
author: "@sam"
---

# FEAT-123: `with` Expression for Scoped Field Access

## Summary
Add a `with` expression that expands a dictionary's fields into a temporary scope, reducing repetitive property access chains in templates. This makes templates cleaner and easier to read by eliminating long, repeated prefixes like `auth.user.name`, `auth.user.email`, etc.

## User Story
As a template author, I want to access multiple fields from a nested dictionary without repeating the full path each time, so that my templates are cleaner and less error-prone.

## Acceptance Criteria
- [x] `with dict { ... }` syntax expands all dictionary fields into the block scope
- [x] Works with both Dictionary and Record types
- [x] Injected variables are immutable (like `let` bindings)
- [x] Variables don't leak outside the `with` block
- [x] Parentheses are optional: `with dict { }` and `with (dict) { }` both work
- [x] `with` is an expression that returns the block's result
- [x] Nested `with` blocks work correctly with standard shadowing
- [x] Appropriate error for non-dictionary/record targets
- [x] Documentation updated (reference, cheatsheet)
- [x] Tests cover all edge cases

## Design Decisions

- **Immutable bindings**: Injected variables cannot be reassigned within the block. This matches `let` semantics and prevents surprising mutations.

- **No selective extraction**: All fields are injected. For selective extraction, use standard destructuring (`let {a, b} = dict`). This keeps `with` simple and single-purpose.

- **Preserve outer `this`**: The `this` binding is not changed by `with`. This is more predictable and avoids magical rebinding.

- **Block form only**: `with` requires braces `{ }`. This makes scope boundaries explicit and consistent with `for`/`if` when they use block form.

- **Optional parentheses**: Like `for` and `if`, parentheses around the target expression are optional for consistency.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `pkg/parsley/lexer/lexer.go` — Add `WITH` token type and keyword
- `pkg/parsley/ast/ast.go` — Add `WithExpression` AST node
- `pkg/parsley/parser/parser.go` — Add `parseWithExpression()` function
- `pkg/parsley/evaluator/eval_control_flow.go` — Add `evalWithExpression()` function
- `pkg/parsley/evaluator/evaluator.go` — Add case for `*ast.WithExpression` in `Eval()`
- `docs/parsley/reference.md` — Document `with` expression
- `docs/parsley/CHEATSHEET.md` — Add `with` to syntax reference

### Dependencies
- Depends on: None
- Blocks: None

### Edge Cases & Constraints

1. **Empty dictionary** — No variables injected, body executes normally
2. **Field name shadows outer variable** — Inner scope wins (standard lexical shadowing)
3. **Field named like keyword** (`true`, `null`) — Works but unusual; no warning for v1
4. **Computed keys in dictionary** — All keys become variables regardless of how they were defined
5. **Error evaluating a field** — Entire `with` expression returns the error
6. **Non-dict/record target** — Runtime error: "with requires a dictionary or record"
7. **Invalid identifier keys** — Keys that aren't valid identifiers (e.g., `"hello world"`, `"123"`, `"a-b"`) are silently skipped. This allows `with` to work with real-world JSON data that may have mixed key formats.

### Implementation Sketch

**Lexer:**
```go
// Add to keywords map
"with": WITH,
```

**AST:**
```go
type WithExpression struct {
    Token  lexer.Token
    Target Expression
    Body   *BlockStatement
}
```

**Parser:**
```go
func (p *Parser) parseWithExpression() ast.Expression {
    // Parse: with [( ] expr [)] { body }
}
```

**Evaluator:**
```go
func evalWithExpression(node *ast.WithExpression, env *Environment) Object {
    // 1. Eval target → must be Dictionary or Record
    // 2. Create NewEnclosedEnvironment(env)
    // 3. Inject all fields as SetLet() bindings
    // 4. evalBlockStatement(node.Body, withEnv)
}
```

## Implementation Notes

### Implementation completed 2025-01-21

**Files modified:**
- `pkg/parsley/lexer/lexer.go` — Added `WITH` token type, keyword mapping, and String() case
- `pkg/parsley/ast/ast.go` — Added `WithExpression` struct with Target and Body fields
- `pkg/parsley/parser/parser.go` — Added `parseWithExpression()`, updated query DSL to handle WITH token
- `pkg/parsley/evaluator/evaluator.go` — Added `unicode` import, `isValidIdentifier()` helper, Eval case
- `pkg/parsley/evaluator/eval_control_flow.go` — Added `evalWithExpression()` function
- `pkg/parsley/tests/with_test.go` — New test file with 17 comprehensive test cases

**Query DSL compatibility:**
The `with` keyword conflicted with the query DSL's `| with relation` syntax for eager loading. Fixed by updating the query parser to check for both `lexer.IDENT` and `lexer.WITH` token types.

**Invalid identifier handling:**
Dictionary keys that are not valid identifiers (e.g., `"hello world"`, `"123"`, `"a-b"`) are silently skipped. This allows `with` to work with real-world JSON data that may have mixed key formats.

## Related
- Plan: `work/plans/PLAN-103-with-expression.md`
- Design doc: `work/design/DESIGN-with-expression.md`
- Prior art: Pascal `with`, D `with` statement