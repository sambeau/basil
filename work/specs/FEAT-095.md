---
id: FEAT-095
title: "Schema Checking: is / is not operators"
status: draft
priority: medium
created: 2026-01-18
author: "@human"
---

# FEAT-095: Schema Checking: is / is not operators

## Summary

Add `is` and `is not` operators for runtime schema checking of Records and Tables. This enables functions to validate that they received the expected record type, supports type-based branching, and allows filtering collections by schema.

## User Story

As a Parsley developer, I want to check if a record or table has a specific schema so that I can guard functions against incorrect input, branch on record types, and filter mixed collections.

## Acceptance Criteria

- [ ] `record is Schema` returns `true` if the record's schema matches
- [ ] `table is Schema` returns `true` if the table's schema matches
- [ ] `record is not Schema` returns `true` if the schema does NOT match
- [ ] `table is not Schema` returns `true` if the schema does NOT match
- [ ] Non-record values return `false` (no error): `null is User` → `false`
- [ ] Plain dicts return `false`: `{name: "x"} is User` → `false`
- [ ] Schema comparison is by identity, not structure
- [ ] Works with `check` guards: `check record is User else error(...)`
- [ ] `is` and `is not` are reserved keywords (or recognized in context)

## Design Decisions

- **`is not` instead of `is !`**: The `!` operator typically has high precedence, making `is !User` ambiguous—does it negate User or the result? `is not` reads naturally and matches Python's syntax.

- **Identity comparison, not structural**: Two schemas with identical fields are still different schemas. `UserCopy` with same fields as `User` does not match `is User`. This prevents accidental type confusion.

- **Safe on all values**: Rather than erroring on non-records, `is` returns `false`. This simplifies guard patterns and filtering without requiring pre-checks.

- **No static checking**: Static schema annotations (`fn(r: User)`) were considered but rejected—they require full type inference to be useful, and partial coverage creates false confidence. Runtime `is` covers all practical use cases.

## Examples

### Guard pattern with check

```parsley
fn saveUser(record) {
    check record is User else {error: "Expected User record, got " + record.schema().name}
    @insert(Users |< ...record .)
}
```

### Multiple guards

```parsley
fn processOrder(order, user) {
    check order is Order else "Expected Order record"
    check user is User else "Expected User record"
    check order.items.length() > 0 else "Empty order"
    
    submitOrder(order, user)
}
```

### Conditional branching

```parsley
fn process(record) {
    if (record is User) {
        processUser(record)
    } else if (record is Product) {
        processProduct(record)
    } else {
        error("Unknown record type: " + record.schema().name)
    }
}
```

### Filtering collections

```parsley
let items = [User({...}), Product({...}), User({...})]

let users = items.filter(fn(x) { x is User })
let products = items.filter(fn(x) { x is Product })
```

### Loop filtering

```parsley
for (item in items) {
    if (item is not User) skip
    processUser(item)
}
```

### Edge cases

```parsley
null is User                      // false
"hello" is User                   // false
42 is User                        // false
{name: "Alice"} is User           // false (plain dict)
table([...]) is User              // false (untyped table)
{name: "Alice"}.as(User) is User  // true
```

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Grammar

```
is_expr := expression "is" schema_ref
         | expression "is" "not" schema_ref

schema_ref := identifier
```

The `is` / `is not` operators have lower precedence than comparison operators but higher than logical `and`/`or`.

### Affected Components

- `pkg/parsley/lexer/lexer.go` — Add IS token, recognize `is` keyword
- `pkg/parsley/token/token.go` — Add IS token type
- `pkg/parsley/ast/ast.go` — Add `IsExpression` node with `Value`, `Schema`, `Negated` fields
- `pkg/parsley/parser/parser.go` — Parse `is` and `is not` expressions
- `pkg/parsley/evaluator/evaluator.go` — Evaluate `IsExpression`

### Implementation Strategy

#### 1. Token and Lexer

Add `IS` token. The lexer recognizes `is` as a keyword. `not` is already a keyword (used in `not in`).

#### 2. AST Node

```go
type IsExpression struct {
    Token   token.Token  // The 'is' token
    Value   Expression   // Left side (record/table/any value)
    Schema  Expression   // Right side (schema identifier)
    Negated bool         // true for "is not"
}
```

#### 3. Parser

Parse as infix operator with precedence between EQUALS and AND:

```go
// In parseExpression, after parsing comparison
if p.curTokenIs(token.IS) {
    return p.parseIsExpression(left)
}

func (p *Parser) parseIsExpression(left ast.Expression) ast.Expression {
    expr := &ast.IsExpression{Token: p.curToken, Value: left}
    p.nextToken()
    
    if p.curTokenIs(token.NOT) {
        expr.Negated = true
        p.nextToken()
    }
    
    expr.Schema = p.parseExpression(LOWEST)
    return expr
}
```

#### 4. Evaluator

```go
func evalIsExpression(node *ast.IsExpression, env *Environment) Object {
    value := Eval(node.Value, env)
    schemaObj := Eval(node.Schema, env)
    
    schema, ok := schemaObj.(*DSLSchema)
    if !ok {
        return newError("is operator requires a schema on right side")
    }
    
    var matches bool
    switch v := value.(type) {
    case *Record:
        matches = v.Schema == schema
    case *Table:
        matches = v.Schema == schema
    default:
        matches = false  // Non-record/table always false
    }
    
    if node.Negated {
        matches = !matches
    }
    
    return nativeBoolToBooleanObject(matches)
}
```

### Edge Cases & Constraints

1. **Schema on right side required** — `record is 42` is a runtime error (42 is not a schema)
2. **Left side can be anything** — Returns false for non-records, doesn't error
3. **Nil/null records** — `null is User` returns false
4. **Nested in expressions** — `(record is User) && record.isValid()` works correctly
5. **Precedence** — `a == b is User` parses as `a == (b is User)` — may want parentheses

### Dependencies

- Depends on: FEAT-002 (Record type with schema)
- Blocks: None

## Testing Strategy

### Unit Tests

1. **Basic is**: `User({...}) is User` → true
2. **Basic is not**: `User({...}) is not Product` → true
3. **Wrong schema**: `User({...}) is Product` → false
4. **Plain dict**: `{name: "x"} is User` → false
5. **Null**: `null is User` → false
6. **Non-record values**: strings, numbers, arrays → false
7. **Tables**: `User([...]) is User` → true
8. **Untyped table**: `table([...]) is User` → false
9. **After .as()**: `{...}.as(User) is User` → true
10. **Schema identity**: two schemas with same fields are different

### Integration Tests

1. Guard pattern with check
2. Filtering arrays with filter()
3. Loop with skip
4. Conditional branching

## Related

- Design: `work/design/DESIGN-record-type-v3.md` (Section 11: Schema Checking)
- Depends on: FEAT-002 (Record Type)
- Related: FEAT-077 (check/stop/skip control flow)
