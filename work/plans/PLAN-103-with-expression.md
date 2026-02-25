---
id: PLAN-103
feature: FEAT-123
title: "Implementation Plan for `with` Expression"
status: draft
created: 2025-01-21
---

# Implementation Plan: FEAT-123 `with` Expression

## Overview
Implement the `with` expression that expands dictionary/record fields into a scoped block, reducing repetitive property access chains in templates.

**Syntax:** `with expr { body }`

**Example:**
```parsley
with auth.user {
  <span>name</span>    // instead of auth.user.name
  <span>email</span>   // instead of auth.user.email
}
```

## Prerequisites
- [x] Design document complete: `work/design/DESIGN-with-expression.md`
- [x] Feature spec complete: `work/specs/FEAT-123-with-expression.md`
- [x] Decision on invalid identifier keys: skip silently

## Tasks

### Task 1: Add `WITH` Token to Lexer
**Files:** `pkg/parsley/lexer/lexer.go`
**Estimated effort:** Small

Steps:
1. Add `WITH` constant to the `TokenType` constants (around line 100)
2. Add `"with": WITH` to the `keywords` map (around line 404)
3. Add case for `WITH` in `TokenType.String()` method (around line 312)

Tests:
- Verify `with` is recognized as keyword, not identifier
- Verify `WITH` in variable name like `withValue` still works as identifier

---

### Task 2: Add `WithExpression` AST Node
**Files:** `pkg/parsley/ast/ast.go`
**Estimated effort:** Small

Steps:
1. Add `WithExpression` struct after `ForExpression` (around line 832):
   ```go
   type WithExpression struct {
       Token  lexer.Token    // the 'with' token
       Target Expression     // expression evaluating to dict/record
       Body   *BlockStatement
   }
   ```
2. Implement `expressionNode()`, `TokenLiteral()`, `String()` methods

Tests:
- None directly (covered by parser/evaluator tests)

---

### Task 3: Add Parser Support
**Files:** `pkg/parsley/parser/parser.go`
**Estimated effort:** Medium

Steps:
1. Register prefix parse function in `New()` (around line 150):
   ```go
   p.registerPrefix(lexer.WITH, p.parseWithExpression)
   ```
2. Add `parseWithExpression()` function (after `parseForExpression`, around line 2955):
   - Handle optional parentheses (like `for`/`if`)
   - Parse target expression
   - Require and parse block body
   - Return `*ast.WithExpression`

Tests:
- `with dict { }` — basic form
- `with (dict) { }` — with parentheses
- `with a.b.c { }` — nested access as target
- `with dict[key] { }` — index expression as target
- `with { }` — error: missing target
- `with dict` — error: missing body block

---

### Task 4: Add `isValidIdentifier` Helper
**Files:** `pkg/parsley/evaluator/evaluator.go`
**Estimated effort:** Small

Steps:
1. Add helper function to check if a string is a valid Parsley identifier:
   ```go
   func isValidIdentifier(s string) bool {
       if len(s) == 0 {
           return false
       }
       for i, r := range s {
           if i == 0 {
               if !unicode.IsLetter(r) && r != '_' {
                   return false
               }
           } else {
               if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
                   return false
               }
           }
       }
       return true
   }
   ```
2. Ensure `unicode` is imported

Tests:
- `"valid"` → true
- `"_private"` → true
- `"CamelCase"` → true
- `"with_underscore"` → true
- `"π"` → true (Unicode letter)
- `"hello world"` → false (space)
- `"123numeric"` → false (starts with digit)
- `"with-dashes"` → false (hyphen)
- `""` → false (empty)

---

### Task 5: Add Evaluator Support
**Files:** `pkg/parsley/evaluator/eval_control_flow.go`, `pkg/parsley/evaluator/evaluator.go`
**Estimated effort:** Medium

Steps:
1. Add `evalWithExpression()` function to `eval_control_flow.go`:
   - Evaluate target expression
   - Type check: must be Dictionary or Record
   - Create enclosed environment
   - Loop through pairs, skip invalid identifier keys
   - Inject valid keys as immutable bindings (`SetLet`)
   - Evaluate body block in new environment
   - Return block result

2. Add case in `Eval()` switch (in `evaluator.go`, around line 4467):
   ```go
   case *ast.WithExpression:
       return evalWithExpression(node, env)
   ```

Tests:
- Basic dictionary field access
- Record field access
- Nested `with` blocks
- Shadowing outer variables
- Empty dictionary (no-op)
- Invalid identifier keys skipped
- Error propagation from target evaluation
- Error propagation from field evaluation
- Type error for non-dict/record target
- Expression result returned correctly

---

### Task 6: Add Error Code
**Files:** `pkg/parsley/evaluator/errors.go`, `docs/parsley/error-codes.md`
**Estimated effort:** Small

Steps:
1. Allocate error code for type mismatch (e.g., `TYPE-0020`)
2. Add error message template: "with requires a dictionary or record, got {Got}"
3. Update error codes documentation

Tests:
- `with 123 { }` — produces correct error
- `with "string" { }` — produces correct error
- `with [1,2,3] { }` — produces correct error

---

### Task 7: Add Comprehensive Tests
**Files:** `pkg/parsley/tests/with_test.go` (new file)
**Estimated effort:** Medium

Test cases to cover:
1. **Basic usage:**
   - Simple dictionary access
   - Record access
   - Multiple fields
   
2. **Scoping:**
   - Variables don't leak outside block
   - Inner scope shadows outer
   - Nested `with` blocks
   - Access to outer variables still works
   
3. **Expression result:**
   - `with` returns block result
   - Single expression result
   - Multiple expressions → array
   
4. **Edge cases:**
   - Empty dictionary
   - Invalid identifier keys skipped
   - Computed keys
   - Unicode field names
   - Field evaluation error
   
5. **Error cases:**
   - Non-dict/record target
   - Missing body block (parser error)
   - Target evaluation error

---

### Task 8: Update Documentation
**Files:** `docs/parsley/reference.md`, `docs/parsley/CHEATSHEET.md`
**Estimated effort:** Small

Steps:
1. Add `with` to reference.md:
   - Syntax
   - Semantics
   - Examples
   - Edge cases (invalid keys)

2. Add `with` to CHEATSHEET.md:
   - Quick syntax reference
   - Common gotcha about invalid identifier keys

---

## Validation Checklist
- [ ] All tests pass: `make test`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] `with` keyword syntax highlighted in VS Code extension
- [ ] Documentation updated
- [ ] work/BACKLOG.md updated with deferrals (if any)

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-01-21 | Task 1: Lexer | ✅ Complete | Added WITH token type and keyword |
| 2025-01-21 | Task 2: AST | ✅ Complete | Added WithExpression node |
| 2025-01-21 | Task 3: Parser | ✅ Complete | Added parseWithExpression, fixed query DSL compatibility |
| 2025-01-21 | Task 4: isValidIdentifier | ✅ Complete | Added helper function |
| 2025-01-21 | Task 5: Evaluator | ✅ Complete | Added evalWithExpression |
| 2025-01-21 | Task 6: Error code | ✅ Complete | Using TYPE-0020 |
| 2025-01-21 | Task 7: Tests | ✅ Complete | 17 test cases, all passing |
| 2025-01-21 | Task 8: Documentation | ✅ Complete | Added to CHEATSHEET.md |

## Deferred Items
Items to add to work/BACKLOG.md after implementation:
- Selective field extraction syntax (`with {a, b} from dict { }`) — adds complexity, use destructuring instead
- Warning for skipped invalid identifier keys — too noisy for JSON data
- Tree-sitter grammar update for `with` keyword — separate task for editor support

## Implementation Notes

### Query DSL Compatibility
The `with` keyword conflicted with the query DSL's `| with relation` syntax for eager loading. Fixed by updating the query parser to check for both `lexer.IDENT` and `lexer.WITH` token types in `parseQueryExpression` and `parseQuerySubquery`.

### Files Modified
- `pkg/parsley/lexer/lexer.go` — Added WITH token type, keyword, and String() case
- `pkg/parsley/ast/ast.go` — Added WithExpression struct
- `pkg/parsley/parser/parser.go` — Added parseWithExpression(), updated query DSL for WITH token
- `pkg/parsley/evaluator/evaluator.go` — Added unicode import, isValidIdentifier(), Eval case
- `pkg/parsley/evaluator/eval_control_flow.go` — Added evalWithExpression()
- `pkg/parsley/tests/with_test.go` — New test file with 17 test cases