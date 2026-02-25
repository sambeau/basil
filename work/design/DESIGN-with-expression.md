# Design: `with` Expression for Scoped Field Access

## Status

**Stage:** Proposal / Discussion  
**Created:** 2025-01-21  
**Author:** @sam, @copilot  

This document explores adding a `with` expression to Parsley that expands a dictionary's fields into a temporary scope, reducing repetitive property access chains in templates.

---

## 1. Overview

### 1.1 Motivation

Parsley templates often access multiple properties from the same nested object, leading to repetitive code:

```parsley
// Current: Repetitive prefix chains
<dl>
  <dt>"User ID"</dt>
  <dd>auth.user.id</dd>
  
  <dt>"Name"</dt>
  <dd>auth.user.name</dd>
  
  <dt>"Email"</dt>
  <dd>auth.user.email ?? "(not set)"</dd>
  
  <dt>"Account Created"</dt>
  <dd>auth.user.created</dd>
</dl>
```

The `auth.user.` prefix is repeated 4 times. In larger templates with deeply nested data, this becomes noisy and error-prone.

### 1.2 Proposed Solution

A `with` expression that expands dictionary fields into a scoped block:

```parsley
// Proposed: Clean, scoped field access
with auth.user {
  <dl>
    <dt>"User ID"</dt>
    <dd>id</dd>
    
    <dt>"Name"</dt>
    <dd>name</dd>
    
    <dt>"Email"</dt>
    <dd>email ?? "(not set)"</dd>
    
    <dt>"Account Created"</dt>
    <dd>created</dd>
  </dl>
}
```

### 1.3 Design Goals

- **Scoped:** Variables don't leak beyond the block
- **Simple:** Minimal syntax, follows existing patterns
- **Safe:** Immutable bindings, predictable shadowing
- **Composable:** Nestable, works in any expression context
- **Consistent:** Works with both Dictionary and Record types

### 1.4 Prior Art

| Language | Construct | Notes |
|----------|-----------|-------|
| Pascal | `with record do` | Closest match — expands record fields into scope |
| D | `with (expr) { }` | Similar, but more sophisticated (supports classes, enums) |
| JavaScript | `with (obj) { }` | **Deprecated** — different semantics, modifies scope chain unsafely |
| C# | `with` expression | Different — creates modified copy of record |
| Swift | `with` | Different — parameter label for closures |
| Visual Basic | `With...End With` | Similar intent, member access via `.property` |

Pascal's `with` is the closest precedent. D's version is similar but includes features (alias resolution, class scope injection) that would overcomplicate Parsley's use case.

---

## 2. Syntax

### 2.1 Basic Form

```parsley
with EXPRESSION {
  BODY
}
```

Where:
- `EXPRESSION` evaluates to a Dictionary or Record
- `BODY` is a block where all fields of the dictionary are available as local variables
- The block returns its result (expression-based, like `if` and `for`)

### 2.2 Parentheses

Parentheses are optional (consistent with `for` and `if`):

```parsley
// Both valid:
with user { ... }
with (user) { ... }
```

### 2.3 Nesting

`with` blocks can nest. Inner scopes shadow outer scopes:

```parsley
with config {
  with database {
    // Both config fields and database fields available
    // database.host shadows config.host if both exist
    <span>host ":" port</span>
  }
}
```

---

## 3. Semantics

### 3.1 Scoping Rules

1. **New enclosed environment:** A child scope is created for the block
2. **Field injection:** All dictionary/record fields are bound as local variables
3. **Immutable bindings:** Injected variables cannot be reassigned (like `let`)
4. **Lexical shadowing:** Inner scope shadows outer scope (standard behavior)
5. **No leakage:** Variables don't exist after the block ends

### 3.2 Supported Types

| Type | Behavior |
|------|----------|
| Dictionary | All key-value pairs become local variables |
| Record | All fields become local variables |
| Other | Runtime error: "with requires a dictionary or record" |

### 3.3 `this` Binding

The outer `this` binding is preserved (not replaced). This keeps behavior predictable and avoids magical rebinding.

### 3.4 Expression Result

`with` is an expression that returns the result of its body block:

```parsley
let greeting = with user {
  if (premium) {
    `Welcome back, {name}!`
  } else {
    `Hello, {name}`
  }
}
```

---

## 4. Examples

### 4.1 Template Simplification

**Before:**
```parsley
<div class="product-card">
  <h3>products[i].details.name</h3>
  <p class="price">products[i].details.price.format()</p>
  <p class="stock">
    if (products[i].details.inStock) "In Stock" else "Out of Stock"
  </p>
  <p class="category">products[i].details.category</p>
</div>
```

**After:**
```parsley
<div class="product-card">
  with products[i].details {
    <h3>name</h3>
    <p class="price">price.format()</p>
    <p class="stock">if (inStock) "In Stock" else "Out of Stock"</p>
    <p class="category">category</p>
  }
</div>
```

### 4.2 API Response Handling

```parsley
let {data} <=/= JSON(@https://api.example.com/user/123)

with data {
  <article>
    <header>
      <h1>displayName</h1>
      <span class="username">"@" username</span>
    </header>
    <p class="bio">bio ?? "No bio provided"</p>
    <footer>
      <span>"Joined: " joinedAt.format("date")</span>
      <span>followers.length() " followers"</span>
    </footer>
  </article>
}
```

### 4.3 Nested With

```parsley
with order {
  <div class="order">
    <h2>"Order #" id</h2>
    <p>"Status: " status</p>
    
    with customer {
      <div class="customer">
        <p>name</p>
        <p>email</p>
      </div>
    }
    
    with shipping.address {
      <div class="address">
        <p>street</p>
        <p>city ", " state " " zip</p>
      </div>
    }
  </div>
}
```

### 4.4 In Loops

```parsley
for item in cart.items {
  with item {
    <tr>
      <td>name</td>
      <td>quantity</td>
      <td>price.format()</td>
      <td>(price * quantity).format()</td>
    </tr>
  }
}
```

---

## 5. Implementation

### 5.1 Grammar Changes

Add `WITH` to the keyword list:

```go
// lexer/lexer.go
var keywords = map[string]TokenType{
    // ... existing keywords ...
    "with": WITH,
}
```

### 5.2 AST Node

```go
// ast/ast.go
type WithExpression struct {
    Token  lexer.Token  // the 'with' token
    Target Expression   // expression that evaluates to dict/record
    Body   *BlockStatement
}

func (we *WithExpression) expressionNode()      {}
func (we *WithExpression) TokenLiteral() string { return we.Token.Literal }
func (we *WithExpression) String() string {
    return "with " + we.Target.String() + " " + we.Body.String()
}
```

### 5.3 Parser

```go
// parser/parser.go
func (p *Parser) parseWithExpression() ast.Expression {
    expr := &ast.WithExpression{Token: p.curToken}
    
    // Optional parentheses
    hasParens := p.peekTokenIs(lexer.LPAREN)
    if hasParens {
        p.nextToken() // consume '('
    }
    p.nextToken() // move to target expression
    
    // Parse target
    if hasParens {
        expr.Target = p.parseExpression(LOWEST)
        if !p.expectPeek(lexer.RPAREN) {
            return nil
        }
    } else {
        expr.Target = p.parseExpressionUntilBrace()
    }
    
    // Parse body block
    if !p.expectPeek(lexer.LBRACE) {
        return nil
    }
    expr.Body = p.parseBlockStatement()
    
    return expr
}
```

### 5.4 Evaluator

```go
// evaluator/eval_control_flow.go
func evalWithExpression(node *ast.WithExpression, env *Environment) Object {
    // 1. Evaluate target
    target := Eval(node.Target, env)
    if isError(target) {
        return target
    }
    
    // 2. Extract pairs and environment from dict/record
    var pairs map[string]ast.Expression
    var targetEnv *Environment
    
    switch t := target.(type) {
    case *Dictionary:
        pairs = t.Pairs
        targetEnv = t.Env
    case *Record:
        pairs = t.Data
        targetEnv = t.Env
    default:
        return newTypeError("TYPE-XXXX", map[string]any{
            "Expected": "dictionary or record",
            "Got":      strings.ToLower(string(target.Type())),
        })
    }
    
    // 3. Create enclosed environment
    withEnv := NewEnclosedEnvironment(env)
    
    // 4. Inject fields as immutable bindings (skip invalid identifiers)
    for key, expr := range pairs {
        // Skip keys that aren't valid identifiers
        if !isValidIdentifier(key) {
            continue
        }
        value := Eval(expr, targetEnv)
        if isError(value) {
            return value
        }
        withEnv.SetLet(key, value)
    }
    
    // 5. Evaluate body in the with scope
    return evalBlockStatement(node.Body, withEnv)
}

// isValidIdentifier checks if a string can be used as a variable name.
// Valid identifiers start with a letter or underscore, followed by
// letters, digits, or underscores. Also supports Unicode letters.
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

### 5.5 Effort Estimate

| Component | Lines | Complexity |
|-----------|-------|------------|
| Lexer: `WITH` keyword | ~2 | Trivial |
| AST: `WithExpression` | ~25 | Easy |
| Parser: `parseWithExpression` | ~35 | Easy |
| Evaluator: `evalWithExpression` | ~50 | Easy |
| Tests | ~100 | Medium |
| Documentation | ~50 | Easy |
| **Total** | **~260** | **Low** |

Estimated time: 1-2 hours for implementation, plus testing.

---

## 6. Edge Cases

### 6.1 Empty Dictionary

```parsley
with {} {
  "nothing injected"  // works fine, no variables added
}
```

### 6.2 Field Name Conflicts

```parsley
let name = "outer"
with {name: "inner"} {
  name  // "inner" — inner scope shadows outer
}
name  // "outer" — outer unchanged
```

### 6.3 Invalid Identifier Keys

Dictionary keys can be arbitrary strings, but variable names must be valid identifiers. Keys that cannot be valid identifiers are **skipped** (not injected):

```parsley
// These keys are valid identifiers — injected
let d1 = {name: "Alice", age: 30, _private: true}
with d1 {
  name    // "Alice" ✓
  age     // 30 ✓
  _private // true ✓
}

// These keys are NOT valid identifiers — skipped silently
let d2 = {
  "hello world": 1,    // contains space
  "123numeric": 2,     // starts with digit  
  "with-dashes": 3,    // contains hyphen
  "foo.bar": 4,        // contains dot
  "": 5                // empty string
}
with d2 {
  // No variables injected — all keys are invalid identifiers
  // Body executes but has no new bindings
}

// Mixed case: some valid, some invalid
let d3 = {"valid_key": 1, "invalid-key": 2, "another_valid": 3}
with d3 {
  valid_key      // 1 ✓
  another_valid  // 3 ✓
  // "invalid-key" is skipped — not accessible as a variable
  // Use d3["invalid-key"] if you need it
}
```

**Rationale:** Silently skipping invalid keys is safer than erroring. Data from external sources (JSON APIs, databases) commonly has keys like `"created_at"` (valid) alongside `"user-id"` (invalid). Erroring would make `with` unusable for real-world data.

**Alternative considered:** Warn on skipped keys. Rejected because it would be noisy for common cases like JSON data, and the behavior is predictable (only identifier-like keys work).

### 6.4 Reserved/Special Names

Fields named `this`, `null`, `true`, `false` should work but are unusual:

```parsley
with {true: 1, false: 0} {
  // 'true' and 'false' shadow the boolean literals — probably a mistake
  // Could warn, but not an error
}
```

### 6.5 Computed Keys

Dictionaries with computed keys work — all keys become variables:

```parsley
let key = "dynamic"
let dict = {[key]: 42}
with dict {
  dynamic  // 42
}
```

### 6.6 Error in Field Evaluation

If evaluating a field produces an error, the entire `with` expression errors:

```parsley
with {a: 1/0} {  // Error: division by zero
  a
}
```

---

## 7. Alternatives Considered

### 7.1 Ad-hoc Block Scope

Add bare `{ }` as a scoping mechanism, then use destructuring:

```parsley
{
  let {id, name, email} = auth.user
  <dd>id</dd>
  <dd>name</dd>
  <dd>email</dd>
}
```

**Assessment:** More explicit but verbose. Requires two new features (bare blocks + spread destructure). The `with` approach is more ergonomic for the template use case.

### 7.2 Alias Binding

Just use a short alias:

```parsley
let u = auth.user
<dd>u.id</dd>
<dd>u.name</dd>
```

**Assessment:** Already possible. Reduces but doesn't eliminate repetition. Doesn't help with deeply nested access.

### 7.3 Implicit `this` Rebinding

Make `with` rebind `this` to the target:

```parsley
with user {
  this.name  // instead of just 'name'
}
```

**Assessment:** Less ergonomic — still requires a prefix. Also more magical and could break existing code patterns.

### 7.4 Spread Destructure

Add `let {...} = dict` to destructure all fields:

```parsley
let {...} = auth.user
// All fields now in scope
```

**Assessment:** Dangerous — no scope boundary, could shadow variables unexpectedly. The `with` block's explicit scope is safer.

---

## 8. Tradeoffs

### 8.1 Pros

✅ **Simple implementation** — leverages existing Environment scoping  
✅ **Consistent** — same structure as `for`/`if` blocks  
✅ **Scoped** — variables don't leak, predictable lifetime  
✅ **Ergonomic** — significant reduction in template noise  
✅ **Safe** — immutable bindings, standard shadowing rules  
✅ **Works with Records** — typed dictionaries from schemas  

### 8.2 Cons

⚠️ **Implicit variable names** — reader must know dictionary structure  
⚠️ **New keyword** — one more thing to learn  
⚠️ **Potential confusion** — "where did `name` come from?"  

### 8.3 Mitigation

The cons are similar to destructuring (`let {a, b} = dict`), which Parsley already has. The scoped nature limits the "implicit variable" concern to a visible, bounded block.

For the "where did this come from?" concern, the `with` keyword at the block start clearly signals that variables are being injected.

---

## 9. Future Considerations

### 9.1 Selective Extraction

Could add syntax to extract only specific fields:

```parsley
with {name, email} from user {
  // only name and email available
}
```

**Status:** Not proposed for initial implementation. Standard destructuring already serves this use case.

### 9.2 Aliasing in With

Could allow aliasing within the with:

```parsley
with user {name as userName} {
  userName
}
```

**Status:** Not proposed. Adds complexity; use regular destructuring if aliasing is needed.

### 9.3 Method Injection

D's `with` can inject methods from classes. For Parsley, this could mean injecting computed properties:

```parsley
with someUrl {
  host     // property access
  .path()  // method call on the url?
}
```

**Status:** Not applicable — Parsley dictionaries don't have methods. The computed properties are already available as regular fields.

---

## 10. Recommendation

**Implement this feature.** The `with` expression:

1. Solves a real ergonomic problem in templates
2. Has clean, well-understood semantics (Pascal/D precedent)
3. Requires minimal implementation effort (~250 lines)
4. Fits Parsley's aesthetic of simplicity and composability
5. Has no runtime cost beyond the scoping overhead (negligible)

The scoped nature makes it safe — the worst case is shadowing within a small, visible block, which is already possible with `let`.

---

## 11. Open Questions

1. **Should we warn on shadowing built-in names?** (e.g., `{true: 1}`)
2. **Should empty `with` target be an error or no-op?** (Proposed: no-op)
3. **What error code for type mismatch?** (Need to allocate)

---

## Appendix: Grammar

```ebnf
with_expression = "with" [ "(" ] expression [ ")" ] block ;
block           = "{" { statement } "}" ;
```
