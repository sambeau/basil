---
id: FEAT-077
title: "Control Flow: check, stop, skip statements"
status: implemented
priority: medium
created: 2025-12-31
author: "@human"
implemented: 2025-12-31
---

# FEAT-077: Control Flow: check, stop, skip statements

## Summary

Add three new control flow statements to Parsley that provide clean early-exit patterns:
- `check CONDITION else VALUE` — precondition validation with early exit
- `stop` — exit a for loop, returning accumulated results
- `skip` — skip current iteration in a for loop (produce null)

These keywords fit Parsley's mental model of for loops as value generators rather than imperative iterators.

## User Story

As a Parsley developer, I want to write clean precondition checks and control loop execution so that I can avoid deeply nested conditionals and write more readable code.

## Acceptance Criteria

- [ ] `check COND else VALUE` exits function/block early when COND is false
- [ ] `check` requires the `else` clause (no optional shorthand)
- [ ] `stop` exits a for loop and returns accumulated results
- [ ] `skip` skips current iteration (equivalent to producing null)
- [ ] `stop` and `skip` are errors outside of for loops
- [ ] `check` works in functions, if blocks, and for loops
- [ ] All three are reserved keywords

## Design Decisions

- **`check` requires `else`**: Forces explicit handling of the failure case. No shorthand for `else null` to keep semantics clear and explicit.

- **`stop` has no value**: Keeps it simple—always returns accumulated results. Use `check ... else VALUE` to exit with a specific value.

- **`skip` vs implicit null**: While `if (cond) { null }` already skips values, `skip` is more readable and intentional. It signals "I'm deliberately skipping this" rather than "I forgot to return something."

- **Keyword names**: 
  - `check` — reads as a sentence, familiar concept
  - `stop` — "stop generating values" fits generator mental model
  - `skip` — "skip this value" is intuitive

- **Not using `break`/`continue`**: These imply imperative statement loops. Parsley's for loops are generators (like list comprehensions), so `stop`/`skip` better describe the action of controlling what values are produced.

## Examples

### check in functions
```parsley
let processOrder = fn(order, user) {
    check order != null else error("No order")
    check user.canPurchase else error("Permission denied")
    check order.items.length() > 0 else error("Empty cart")
    
    // Happy path continues...
    submitOrder(order, user)
}
```

### check in if blocks
```parsley
let result = if (action == "add") {
    check data != null else null
    check data.isValid else error("Invalid data")
    process(data)
}
```

### check in for loops
```parsley
for (item in items) {
    check item.isValid else stop    // exit loop, return accumulated
    check item.type != "skip" else skip  // skip this item
    transform(item)
}
```

### stop in for loops
```parsley
// Take first 5 valid items
let count = 0
for (item in items) {
    if (count >= 5) stop
    if (!item.isValid) skip
    count = count + 1
    transform(item)
}

// Process until sentinel
for (line in lines) {
    if (line == "END") stop
    process(line)
}
```

### skip in for loops
```parsley
// Filter and transform
for (x in xs) {
    if (x == null) skip
    if (x.type == "ignore") skip
    process(x)
}

// Equivalent to current pattern (but clearer intent):
for (x in xs) {
    if (x != null && x.type != "ignore") {
        process(x)
    }
}
```

### Combined example
```parsley
let processItems = fn(items, maxCount) {
    check items != null else []
    check maxCount > 0 else []
    
    let count = 0
    for (item in items) {
        if (count >= maxCount) stop
        if (item == null) skip
        check item.isValid else error("Invalid item: {item}")
        count = count + 1
        transform(item)
    }
}
```

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Grammar

```
check_stmt := "check" expression "else" expression
stop_stmt  := "stop"
skip_stmt  := "skip"
```

### Affected Components

- `pkg/parsley/lexer/lexer.go` — Add CHECK, STOP, SKIP tokens and keywords
- `pkg/parsley/ast/ast.go` — Add CheckStatement, StopStatement, SkipStatement nodes
- `pkg/parsley/parser/parser.go` — Parse check/stop/skip statements
- `pkg/parsley/evaluator/evaluator.go` — Evaluate check/stop/skip with proper control flow

### Implementation Strategy

#### 1. Control Flow via Special Return Values

Similar to how `return` works, use sentinel objects to signal control flow:

```go
// In evaluator
type StopSignal struct{}
func (s *StopSignal) Type() ObjectType { return STOP_SIGNAL }

type SkipSignal struct{}
func (s *SkipSignal) Type() ObjectType { return SKIP_SIGNAL }
```

#### 2. check Statement Evaluation

```go
func evalCheckStatement(node *ast.CheckStatement, env *Environment) Object {
    condition := Eval(node.Condition, env)
    if isTruthy(condition) {
        return nil  // condition passed, continue execution
    }
    // Condition failed - evaluate and return else value
    return Eval(node.ElseValue, env)
}
```

The caller (function body, if block, for loop) must check if result is the else value and exit appropriately.

#### 3. stop/skip in For Loops

```go
func evalForExpression(node *ast.ForExpression, env *Environment) Object {
    var results []Object
    for _, element := range iterable {
        // ... set up iteration env ...
        
        result := Eval(node.Body, loopEnv)
        
        switch result.(type) {
        case *StopSignal:
            return &Array{Elements: results}  // return accumulated
        case *SkipSignal:
            continue  // don't append, move to next iteration
        case *Error:
            return result
        default:
            if result != nil && result != NULL {
                results = append(results, result)
            }
        }
    }
    return &Array{Elements: results}
}
```

#### 4. Error Handling

- `stop` outside for loop → runtime error
- `skip` outside for loop → runtime error  
- `check` can appear anywhere (function, if, for)

### Edge Cases & Constraints

1. **Nested loops**: `stop` and `skip` affect only the innermost loop
2. **check in for loop with stop**: `check X else stop` exits loop with accumulated results
3. **check in for loop with skip**: `check X else skip` skips current iteration
4. **Multiple checks**: All must pass for execution to continue
5. **check with error()**: `check X else error("msg")` returns error, propagates up

### Dependencies

- None

### Blocks

- None

## Testing Strategy

### Unit Tests

1. **check in functions**: early return scenarios
2. **check in if blocks**: exit block scenarios
3. **check in for loops**: with stop, skip, and values
4. **stop**: basic usage, nested loops
5. **skip**: basic usage, filter patterns
6. **Error cases**: stop/skip outside loops

### Integration Tests

1. Complex function with multiple checks
2. For loop with check, stop, and skip combined
3. Nested structures

## Related

- Plan: `work/plans/FEAT-077-plan.md` (to be created)
