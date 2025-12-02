---
id: FEAT-013
title: "Add 'in' membership operator"
status: implemented
priority: medium
created: 2025-12-02
author: "@human"
implemented: 2025-12-02
---

# FEAT-013: Add 'in' membership operator

## Summary
Add an `in` binary operator to test if a value exists in an array, if a key exists in a dictionary, or if a substring exists in a string.

## User Story
As a Parsley developer, I want to easily check if a value is in a collection so I can write cleaner conditional logic without using `.filter()` or `.find()`.

## Acceptance Criteria
- [x] `value in array` returns `true` if value exists in the array
- [x] `key in dict` returns `true` if key exists in the dictionary
- [x] `substring in string` returns `true` if substring exists in the string
- [x] `in` works with all value types (strings, numbers, booleans, null)
- [x] Does not conflict with `for (x in y)` syntax (lexer already distinguishes)
- [x] Add `.includes()` method on arrays for chaining scenarios
- [x] Add `.includes()` method on strings for chaining scenarios

## Examples

### Array membership
```parsley
let fruits = ["apple", "banana", "cherry"]

"apple" in fruits      // true
"grape" in fruits      // false
5 in [1, 2, 3, 4, 5]   // true
```

### Dictionary key check
```parsley
let user = {name: "Sam", role: "admin"}

"name" in user         // true
"email" in user        // false
```

### In conditionals
```parsley
if ("admin" in user.roles) {
  showAdminPanel()
}

if (statusCode in [200, 201, 204]) {
  log("Success!")
}
```

### Method form for chaining
```parsley
let hasWrite = permissions
  .filter(fn(p) { p.level > 3 })
  .includes("write")
```

## Design Decisions

### Operator Precedence
`in` should have lower precedence than comparison operators but higher than logical operators:
```parsley
x == 1 or x in [2, 3]  // parsed as: (x == 1) or (x in [2, 3])
```

### No conflict with for-loops
The parser distinguishes by context:
- `for (x in y)` - `in` is part of for-loop syntax
- `x in y` elsewhere - `in` is the membership operator

### String membership
For strings, `in` checks if a substring exists:
```parsley
"ell" in "hello"  // true
"xyz" in "hello"  // false
```

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Lexer Changes
- `IN` token type already existed in the lexer (used for for-loops)
- No lexer changes required

### Parser Changes
- Added `lexer.IN: EQUALS` to the precedences map
- Registered `p.registerInfix(lexer.IN, p.parseInfixExpression)` 

### Evaluator Changes
- Added case for `operator == "in"` in `evalInfixExpression`, calling new `evalInExpression` function
- `evalInExpression` handles: 
  - Array membership (uses `objectsEqual` for comparison)
  - Dictionary key check (key must be string)
  - String contains (substring must be string)
- Added `.includes()` method to `evalArrayMethod` in methods.go
- Added `.includes()` method to `evalStringMethod` in methods.go

### Files Modified
- `pkg/parsley/parser/parser.go` - Register infix operator, add precedence
- `pkg/parsley/evaluator/evaluator.go` - Implement `evalInExpression`
- `pkg/parsley/evaluator/methods.go` - Add `.includes()` methods
- `pkg/parsley/tests/in_operator_test.go` - Comprehensive tests

## Related
- FEAT-011: basil namespace (common use case: `"admin" in basil.auth.user.roles`)
