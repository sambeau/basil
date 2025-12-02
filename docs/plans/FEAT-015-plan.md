---
id: PLAN-009
feature: FEAT-015
title: "Implementation Plan for Optional Chaining ?."
status: draft
created: 2025-12-02
---

# Implementation Plan: FEAT-015 - Optional Chaining `?.`

## Overview
Add optional chaining operator `?.` that short-circuits to `null` when the left side is null, instead of throwing an error. This enables safe property access chains: `user?.address?.city ?? "Unknown"`.

## Prerequisites
- [x] Understand current DotExpression implementation
- [x] Understand lexer token handling
- [ ] FEAT-014 (Optional Indexing) - recommended first for consistency

## Complexity Estimate: **Medium** (~2-3 hours)

More complex than `[?n]` because:
1. Needs new `QUESTIONDOT` token (two-character token)
2. May interact with method calls: `arr?.first()` 
3. Should work with computed properties too
4. Need to handle short-circuit evaluation in chains

---

## Tasks

### Task 1: Add QUESTIONDOT token to lexer
**Files**: `pkg/parsley/lexer/lexer.go`
**Estimated effort**: Small (15 min)

Steps:
1. Add `QUESTIONDOT` token type constant
2. Add case in `TokenTypeString()`
3. Modify `?` handling to check for `?.` sequence:

```go
case '?':
    if l.peekChar() == '?' {
        // NULLISH ??
        ch := l.ch
        l.readChar()
        tok = Token{Type: NULLISH, ...}
    } else if l.peekChar() == '.' {
        // QUESTIONDOT ?.
        ch := l.ch
        l.readChar()
        tok = Token{Type: QUESTIONDOT, Literal: "?.", ...}
    } else {
        tok = newToken(QUESTION, l.ch, l.line, l.column)
    }
```

Tests:
- Lex `?.` → `QUESTIONDOT`
- Lex `??` still → `NULLISH`
- Lex `?` alone → `QUESTION` (or ILLEGAL if FEAT-014 not done)

---

### Task 2: Add `Optional` field to DotExpression AST
**Files**: `pkg/parsley/ast/ast.go`
**Estimated effort**: Small (10 min)

Steps:
1. Find `DotExpression` struct
2. Add `Optional bool` field
3. Update `String()` method

```go
type DotExpression struct {
    Token    lexer.Token
    Left     Expression
    Key      string
    Optional bool  // NEW: true for ?. syntax
}
```

---

### Task 3: Register and parse `?.` as infix operator
**Files**: `pkg/parsley/parser/parser.go`
**Estimated effort**: Medium (30 min)

Steps:
1. Add precedence for `QUESTIONDOT` (same as `DOT`)
2. Register infix parser for `QUESTIONDOT`
3. Create `parseOptionalDotExpression()` or modify existing

```go
// In init
precedences[lexer.QUESTIONDOT] = CALL  // same as DOT

// Register
p.registerInfix(lexer.QUESTIONDOT, p.parseOptionalDotExpression)

func (p *Parser) parseOptionalDotExpression(left ast.Expression) ast.Expression {
    dotExpr := &ast.DotExpression{
        Token:    p.curToken,
        Left:     left,
        Optional: true,
    }
    if !p.expectPeek(lexer.IDENT) {
        return nil
    }
    dotExpr.Key = p.curToken.Literal
    return dotExpr
}
```

Tests:
- Parse `a?.b` produces DotExpression with Optional=true
- Parse `a.b` produces DotExpression with Optional=false
- Parse `a?.b?.c` produces nested optional DotExpressions

---

### Task 4: Handle optional chaining in evaluator
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Medium (40 min)

Steps:
1. Find where `DotExpression` is evaluated
2. Check `node.Optional` flag
3. If left evaluates to `null` and Optional is true, return `NULL` immediately
4. Handle dictionaries, computed properties, and method calls

```go
case *ast.DotExpression:
    left := Eval(node.Left, env)
    if isError(left) {
        return left
    }
    
    // Optional chaining: return null if left is null
    if node.Optional && left == NULL {
        return NULL
    }
    
    // ... existing property/method resolution ...
```

Considerations:
- Method calls: `arr?.first()` - need to handle CallExpression with optional base
- Computed properties: `path?.components` should work

Tests:
- `null?.foo` → `null`
- `{a: 1}?.a` → `1`
- `{a: {b: 2}}?.a?.b` → `2`
- `{a: null}?.a?.b` → `null`
- `null?.foo?.bar?.baz` → `null` (short circuits)
- `let x = null; x?.name ?? "default"` → `"default"`

---

### Task 5: Handle optional method calls (if applicable)
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Medium (30 min)

If the pattern `obj?.method()` should work, we need to:
1. Check if CallExpression's function is an optional DotExpression
2. If base is null, return null instead of calling

This may require changes to how method calls are resolved.

Tests:
- `[1,2,3]?.first()` → works (array not null)
- `null?.first()` → `null`
- `[].first()` → needs FEAT-014 or returns error

---

### Task 6: Create tests
**Files**: `pkg/parsley/tests/optional_chaining_test.go` (new)
**Estimated effort**: Small (25 min)

Comprehensive tests for:
- Simple optional access
- Chained optional access
- Mixed `.` and `?.` in chains
- Combination with `??`
- With dictionaries, computed properties
- Edge cases

---

### Task 7: Update documentation
**Files**: `docs/parsley/reference.md`, `docs/parsley/CHEATSHEET.md`
**Estimated effort**: Small (10 min)

- Add `?.` to operators section
- Show chaining examples
- Explain short-circuit behavior

---

## Validation Checklist
- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `make build`
- [ ] Manual test: `echo 'let x = null; log(x?.foo ?? "safe")' | ./pars` → `safe`

## Total Estimated Time: ~2.5-3 hours

## Dependencies
- Recommend implementing FEAT-014 (Optional Indexing) first for:
  - Shared `QUESTION` token handling
  - Consistent behavior patterns
  - Full chain support: `arr[?0]?.name`

## Comparison

| Feature | FEAT-014 `[?n]` | FEAT-015 `?.` |
|---------|-----------------|---------------|
| Complexity | Small-Medium | Medium |
| Time | ~1.5-2 hours | ~2.5-3 hours |
| Lexer changes | Add QUESTION | Add QUESTIONDOT |
| AST changes | Add Optional to IndexExpression | Add Optional to DotExpression |
| Evaluator | Conditional null return | Short-circuit evaluation |
| Dependencies | None | Better with FEAT-014 first |
