---
id: FEAT-096
title: "Computed Exports"
status: implemented
priority: medium
created: 2026-01-19
author: "@human"
---

# FEAT-096: Computed Exports

## Summary

Add `export computed` syntax to allow module authors to export values that are recalculated each time they are accessed. This provides a clean way to expose "live" data (like database queries, current timestamps, or derived calculations) as simple property access rather than function calls, while being transparent to consumers.

## User Story

As a module author, I want to export values that are computed on access so that consumers can use simple property syntax (`data.users`) instead of function calls (`data.users()`), making the API cleaner while ensuring they always get fresh data.

## Motivation

Currently, if a module needs to provide data that should be recalculated on each access, it must export a function:

```parsley
// data.pars (current approach)
export let getUsers = fn() {
    @DB.query("SELECT * FROM users")
}

// consumer.pars
import {getUsers} from "./data.pars"
let users = getUsers()  // Must call as function
```

This works but has drawbacks:
1. **API aesthetics**: `data.users` reads better than `data.getUsers()`
2. **Consumer confusion**: It's not obvious whether you need `()` or not
3. **Inconsistency**: Static exports use property access, dynamic ones use function calls

The `@basil/http` and `@basil/auth` modules already use internal `DynamicAccessor` to solve this for framework values. This feature exposes the same capability to user-defined modules.

## Acceptance Criteria

- [ ] `export computed name { body }` block syntax works
- [ ] `export computed name = expression` expression syntax works
- [ ] Computed exports resolve to the evaluated value (not a function)
- [ ] Computed exports recalculate on each access
- [ ] Consumers can cache values via `let cached = computedValue`
- [ ] Errors propagate normally (can be caught with `try`)
- [ ] Computed exports work with destructuring imports
- [ ] Works with module caching (module loads once, computed resolves fresh)

## Examples

### Block Form (multi-line computation)
```parsley
// data.pars
export computed users {
    let query = "SELECT * FROM users WHERE active = true"
    @DB.query(query)
}

export computed timestamp {
    @std/time.now()
}

// consumer.pars
import {users, timestamp} from "./data.pars"

print(users)      // Queries database
print(users)      // Queries database again (fresh data)
print(timestamp)  // Current time
```

### Expression Form (one-liner)
```parsley
// stats.pars
let items = [1, 2, 3, 4, 5]

export computed count = items.length()
export computed sum = items.reduce(fn(a, b) { a + b }, 0)
export computed average = sum / count

// consumer.pars
import {count, sum, average} from "./stats.pars"
print("Average: " + average)
```

### Consumer Caching
```parsley
import {users} from "./data.pars"

// Each access recalculates
for (user in users) { print(user.name) }  // Query 1
for (user in users) { print(user.email) } // Query 2

// Cache if you need the same data
let snapshot = users                       // Query 3
for (user in snapshot) { print(user.name) }  // Uses snapshot
for (user in snapshot) { print(user.email) } // Uses snapshot
```

### Error Handling
```parsley
import {users} from "./data.pars"

// If the database query fails, catch the error
let result = try users
if (result is error) {
    print("Failed to load users: " + result.message)
} else {
    for (user in result) { print(user.name) }
}
```

## Design Decisions

- **Always recalculates**: Computed exports never cache. This keeps semantics simple and predictable. Consumers who want caching can assign to a variable.

- **Transparent to consumers**: Importing a computed export looks identical to importing a static export. The consumer doesn't need to know (or care) whether it's computed.

- **No parameters**: Computed exports are zero-argument by definition. If you need parameters, use a function. This maintains the "looks like a property" semantics.

- **Both syntax forms**: Supporting both `= expression` and `{ body }` forms matches existing patterns for `let` and functions in Parsley.

- **Errors propagate naturally**: No special error handling. If the computation fails, the error propagates to the access site, where `try` can be used as with any expression.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Existing Infrastructure

The `DynamicAccessor` type already exists and is used for `@basil/http` and `@basil/auth`:

```go
// pkg/parsley/evaluator/stdlib_table.go
type DynamicAccessor struct {
    Name     string
    Resolver func(*Environment) Object
}
```

Property access on `StdlibModuleDict` already resolves `DynamicAccessor`:

```go
// pkg/parsley/evaluator/evaluator.go (evalDotExpression)
if accessor, ok := val.(*DynamicAccessor); ok {
    return accessor.Resolve(env)
}
```

### Affected Components

- `pkg/parsley/lexer/lexer.go` — Add `COMPUTED` keyword
- `pkg/parsley/parser/parser.go` — Parse `export computed` statements
- `pkg/parsley/ast/ast.go` — Add `ComputedExportStatement` node
- `pkg/parsley/evaluator/evaluator.go` — Evaluate computed exports, creating `DynamicAccessor`
- `pkg/parsley/evaluator/eval_expressions.go` — Ensure module imports resolve `DynamicAccessor` for user modules

### Implementation Approach

1. **Lexer**: Add `computed` as a keyword (only valid after `export`)

2. **Parser**: When seeing `export computed`:
   - Parse identifier name
   - If `=`, parse expression form
   - If `{`, parse block form
   - Create `ComputedExportStatement` AST node

3. **AST Node**:
   ```go
   type ComputedExportStatement struct {
       Token token.Token      // The 'export' token
       Name  *Identifier      // Export name
       Body  Expression       // Expression or BlockExpression
   }
   ```

4. **Evaluator**: When evaluating `ComputedExportStatement`:
   ```go
   case *ast.ComputedExportStatement:
       accessor := &DynamicAccessor{
           Name: node.Name.Value,
           Resolver: func(resolveEnv *Environment) Object {
               // Evaluate body in module env, but with current context
               evalEnv := mergeEnvContext(env, resolveEnv)
               return Eval(node.Body, evalEnv)
           },
       }
       env.SetExport(node.Name.Value, accessor)
       return NULL
   ```

5. **Module Resolution**: User modules need to return a structure that resolves `DynamicAccessor` on property access (similar to `StdlibModuleDict`). May need a `UserModuleDict` wrapper or extend existing dictionary handling.

### Edge Cases & Constraints

1. **Module caching** — Module is parsed/evaluated once, but `DynamicAccessor` resolves fresh on each access. This is the existing behavior for `@basil/http`.

2. **Circular dependencies** — If computed export A accesses computed export B in the same module, both resolve in the module's environment. No special handling needed.

3. **Destructuring imports** — `import {users} from "./data.pars"` should bind `users` to the `DynamicAccessor`, not the resolved value. This matches existing behavior for `@basil/http`.

4. **Re-export** — `export {users} from "./data.pars"` should preserve the computed nature. The re-exported value should remain a `DynamicAccessor`.

5. **Environment context** — The resolver needs access to both:
   - Module's lexical environment (for closures over module variables)
   - Current request context (`BasilCtx`) for `@basil/http` etc.
   
   This is already handled by `DynamicAccessor` resolution passing the access-site environment.

### Grammar

```
export_statement
    : 'export' 'let' identifier '=' expression
    | 'export' 'const' identifier '=' expression
    | 'export' 'computed' identifier '=' expression
    | 'export' 'computed' identifier block
    | 'export' identifier
    | 'export' '{' export_list '}' 'from' string
    ;
```

## Dependencies

- Depends on: None (uses existing `DynamicAccessor` infrastructure)
- Blocks: None

## Related

- BUG-014: DynamicAccessor for `@basil/http` stale values (implementation reference)
- `pkg/parsley/evaluator/stdlib_table.go`: `DynamicAccessor` type definition
- `pkg/parsley/tests/module_cache_test.go`: `TestDynamicAccessorInCachedModule` (test reference)
