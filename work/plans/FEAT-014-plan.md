---
id: PLAN-008
feature: FEAT-014
title: "Implementation Plan for Optional Indexing [?n]"
status: complete
created: 2025-12-02
completed: 2025-12-02
---

# Implementation Plan: FEAT-014 - Optional Indexing `[?n]`

## Overview
Add optional indexing syntax `arr[?n]` that returns `null` instead of an error when the index is out of bounds. This enables safe array access with null coalescing: `arr[?0] ?? "default"`.

## Prerequisites
- [x] Understand current IndexExpression implementation
- [x] Understand lexer token handling for `?`

## Complexity Estimate: **Small-Medium** (~1-2 hours)

The implementation is straightforward because:
1. Only need to add one boolean flag to `IndexExpression` AST node
2. Lexer already handles `?` (just needs to not mark it ILLEGAL in `[?` context)
3. Evaluator just needs a conditional null return instead of error

---

## Tasks

### Task 1: Add QUESTION token to lexer
**Files**: `pkg/parsley/lexer/lexer.go`
**Estimated effort**: Small (15 min)

Steps:
1. Add `QUESTION` token type constant (around line 52)
2. Add case in `TokenTypeString()` (around line 184)
3. Modify `?` handling in `NextToken()` to emit `QUESTION` when not followed by `?`

Currently `?` alone is `ILLEGAL`. Change to:
```go
case '?':
    if l.peekChar() == '?' {
        // existing NULLISH handling
    } else {
        tok = newToken(QUESTION, l.ch, l.line, l.column)
    }
```

Tests:
- Lex `[?0]` → `LBRACKET`, `QUESTION`, `INT`, `RBRACKET`
- Lex `??` still produces `NULLISH`

---

### Task 2: Add `Optional` field to IndexExpression AST
**Files**: `pkg/parsley/ast/ast.go`
**Estimated effort**: Small (10 min)

Steps:
1. Add `Optional bool` field to `IndexExpression` struct
2. Update `String()` method to show `[?...]` when Optional is true

```go
type IndexExpression struct {
    Token    lexer.Token
    Left     Expression
    Index    Expression
    Optional bool  // NEW: true for [?n] syntax
}
```

Tests:
- AST String() output shows `[?n]` vs `[n]`

---

### Task 3: Parse `[?expr]` syntax
**Files**: `pkg/parsley/parser/parser.go`
**Estimated effort**: Small (20 min)

Steps:
1. In `parseIndexOrSliceExpression()`, after consuming `[`:
   - Check if current token is `QUESTION`
   - If so, set `exp.Optional = true` and advance
2. Continue with normal index parsing

```go
func (p *Parser) parseIndexOrSliceExpression(left ast.Expression) ast.Expression {
    exp := &ast.IndexExpression{Token: p.curToken, Left: left}
    
    p.nextToken()
    
    // Check for optional index [?...]
    if p.curTokenIs(lexer.QUESTION) {
        exp.Optional = true
        p.nextToken()
    }
    
    // ... rest of existing parsing
}
```

Tests:
- Parse `arr[?0]` produces IndexExpression with Optional=true
- Parse `arr[0]` produces IndexExpression with Optional=false
- Parse `arr[?-1]` works (negative optional index)

---

### Task 4: Handle optional indexing in evaluator
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small (30 min)

Steps:
1. In `Eval()` case for `*ast.IndexExpression`, pass `node.Optional` to `evalIndexExpression()`
2. Update `evalIndexExpression()` signature to accept `optional bool`
3. Update `evalArrayIndexExpression()` to return `NULL` instead of error when optional and out of bounds
4. Update `evalStringIndexExpression()` similarly
5. Update `evalDictionaryIndexExpression()` to return `NULL` for missing keys when optional

```go
func evalArrayIndexExpression(tok lexer.Token, array, index Object, optional bool) Object {
    // ... existing bounds calculation ...
    
    if idx < 0 || idx >= int64(len(arrayObject.Elements)) {
        if optional {
            return NULL
        }
        return newError("index out of range: %d", idx)
    }
    // ... return element ...
}
```

Tests:
- `[1,2,3][?0]` → `1`
- `[1,2,3][?99]` → `null`
- `[1,2,3][?-99]` → `null`
- `[][?0]` → `null`
- `"hello"[?99]` → `null`
- `{a: 1}[?"b"]` → `null`
- `[1,2,3][?0] ?? "default"` → `1`
- `[][?0] ?? "default"` → `"default"`

---

### Task 5: Create tests
**Files**: `pkg/parsley/tests/optional_index_test.go` (new)
**Estimated effort**: Small (20 min)

Create comprehensive test file covering:
- Basic optional indexing on arrays
- Negative indices with optional
- Optional indexing on strings
- Optional indexing on dictionaries
- Combination with null coalesce `??`
- Edge cases: empty array, single element, boundary indices

---

### Task 6: Update documentation
**Files**: `docs/parsley/reference.md`, `docs/parsley/CHEATSHEET.md`
**Estimated effort**: Small (10 min)

- Add `[?n]` to array/string indexing section
- Add example to CHEATSHEET showing `arr[?0] ?? "default"` pattern

---

## Validation Checklist
- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `make build`
- [ ] Manual test: `echo 'log([1,2][?5] ?? "nope")' | ./pars` → `nope`

## Total Estimated Time: ~1.5-2 hours
