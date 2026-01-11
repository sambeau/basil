---
id: FEAT-039
title: "Enhanced Import Syntax"
status: implemented
priority: medium
created: 2025-12-07
implemented: 2025-12-07
author: "@human"
---

# FEAT-039: Enhanced Import Syntax

## Summary
Extend Parsley's import syntax to support scoped modules (`@basil/`, `@std/`), aliasing (`as`), and destructuring imports. This provides flexible ways to import modules while keeping the global namespace clean.

## User Story
As a developer, I want flexible import syntax so that I can choose how to organize my code—whether namespaced, aliased, or with specific exports pulled into scope.

## Acceptance Criteria
- [x] Scoped imports: `import @basil/auth`, `import @std/strings`
- [x] Local imports: `import @./components/Button`
- [x] Aliased imports: `import @basil/auth as Auth`
- [x] Destructured imports: `{Login, Logout} = import @basil/auth`
- [x] Destructured with rename: `{Login as MyLogin} = import @basil/auth`
- [x] Dynamic paths with interpolation: `import @(./components/{name})`
- [x] Parentheses and quotes removed from current syntax (was `import("path")`, now `import @path`)
- [x] Clear error messages for invalid imports
- [x] Backward compatibility: old `import("path")` syntax still works

## Design Decisions
- **Unquoted `@path` syntax**: All imports use `@` prefix with unquoted paths: `import @basil/auth`, `import @std/math`, `import @./local`. Cleaner than quoted strings.
- **`@()` for dynamic paths**: Use `import @(./components/{name})` when path needs interpolation. Consistent with other Parsley literals like `@(2024-{month}-{day})`.
- **Path prefixes**: `@basil/` for Basil modules, `@std/` for stdlib, `@./` and `@../` for relative paths.
- **`as` for aliasing**: Familiar syntax from Python, JavaScript ES6.
- **Destructuring assignment syntax**: Uses existing `{...} = expr` pattern, applied to imports.
- **Module names are lowercase**: Following Go/JS conventions. Users can alias to PascalCase if preferred.

## Syntax Reference

### Scoped Module Import
```parsley
import @basil/auth
import @std/strings
import @std/math

// Access via last path segment
auth.Login
strings.split("a,b", ",")
```

### Aliased Import
```parsley
import @basil/auth as Auth
import @std/strings as S

// Access via alias
<Auth.Login/>
S.split("a,b", ",")
```

### Destructured Import
```parsley
{Login, Logout, Register} = import @basil/auth
{split, join} = import @std/strings

// Access directly (no namespace)
<Login/>
split("a,b", ",")
```

### Destructured with Rename
```parsley
{Login as AuthLogin} = import @basil/auth
{split as splitString} = import @std/strings

<AuthLogin/>
splitString("a,b", ",")
```

### Local Imports
```parsley
import @./components/Button
import @../shared/utils

<Button label="Click me"/>
```

### Dynamic Paths
```parsley
// Path literal with interpolation (preferred)
let name = "Button"
import @(./components/{name})

// Multiple variables
let version = "v2"
let endpoint = "users"
import @(./api/{version}/{endpoint})
```

Note: Dynamic paths use `@()` with an interpolated path inside—no quotes needed. This matches other Parsley literals like `@(2024-{month}-{day})`. Static paths use bare `@path`.

### Mixed Usage
```parsley
import @basil/cache                // Full module
{Login} = import @basil/auth       // Just Login
import @std/strings as str         // Aliased

<cache.Cache key="nav" maxAge={@1h}>
  <Login/>
</cache.Cache>

str.upper("hello")
```

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Syntax Change from Current

**Current syntax** (function-like with parentheses and quotes):
```parsley
let math = import("std/math")
let {Login} = import("std/auth")
```

**Proposed syntax** (unquoted `@path`, no parentheses):
```parsley
import @std/math
{Login} = import @basil/auth
import @./components/Button
```

**Dynamic paths** (interpolated):
```parsley
import @(./components/{name})
```

### Parser Changes Required

The parser must recognize `@` as the start of a module path after `import`:

```
import  @  basil  /  auth
        ^    ^    ^    ^
        |    |    |    identifier
        |    |    slash (path separator, not division)
        |    identifier  
        at-sign (starts module path)
```

After seeing `import @`, the parser enters "module path mode" where `/` is a path separator until whitespace, `as`, or newline.

### Path Syntax

| Form | Example | Use Case |
|------|---------|----------|
| Scoped | `import @basil/auth` | Basil modules |
| Stdlib | `import @std/math` | Standard library |
| Relative | `import @./foo` | Local files |
| Parent | `import @../shared/utils` | Parent directory |
| Dynamic | `import @(./components/{name})` | Interpolated paths |

The `@()` syntax with parentheses enables interpolation in the path, consistent with other Parsley literals like `@(2024-{month}-{day})`. No quotes needed inside.

### Affected Components
- `pkg/parsley/lexer/` — May need to tokenize `@` separately, or handle in parser
- `pkg/parsley/parser/` — Recognize `import @path` and `import @(expr)` forms
- `pkg/parsley/evaluator/` — Module resolution, aliasing, destructuring
- `pkg/parsley/ast/` — Update ImportExpression node (path can be static token sequence or dynamic expression)

### Module Resolution Order
1. `@basil/*` — Basil runtime modules (injected by server)
2. `@std/*` — Parsley standard library
3. `./path` or `../path` — Relative to current file
4. `path` — Relative to project root (or error?)

### Lexer Changes
New tokens:
- `@` — Scope prefix (or part of module path token)
- `as` — Keyword for aliasing (context-sensitive?)

### Parser Changes
```
ImportStmt = "import" ModulePath [ "as" Identifier ]
           | DestructurePattern "=" "import" ModulePath

ModulePath = "@" Scope "/" Path
           | RelativePath

DestructurePattern = "{" ImportItem ("," ImportItem)* "}"
ImportItem = Identifier [ "as" Identifier ]
```

### Evaluator Changes
- `import @scope/module` returns the module object
- `as Alias` binds module to alias name instead of module name
- Destructuring extracts named exports into current scope

### Dependencies
- Depends on: None (core language feature)
- Blocks: None (but enhances FEAT-038 experience)

### Edge Cases & Constraints
1. **Name conflicts** — Destructured import overwrites existing variable? Error?
2. **Missing exports** — `{Foo} = import @std/strings` where `Foo` doesn't exist
3. **Circular imports** — Already handled? Need to verify
4. **Runtime vs compile-time** — Imports resolved when? Affects caching
5. **`as` as identifier** — Is `as` a reserved word or contextual?

### Grammar Considerations
The `as` keyword could conflict with existing identifier usage:
```parsley
let as = 5  // Currently valid?
```

Options:
1. Make `as` a reserved word (breaking change if used as identifier)
2. Make `as` contextual (only keyword after `import` or in destructure)

Recommend: **Contextual keyword** (less breaking)

## Implementation Notes

### Implementation Date: 2025-12-07

**Key files changed:**
- `pkg/parsley/lexer/lexer.go` - Added `IMPORT` keyword token
- `pkg/parsley/ast/ast.go` - Added `ImportExpression` AST node
- `pkg/parsley/parser/parser.go` - Added `parseImportExpression()` with backward compat
- `pkg/parsley/evaluator/evaluator.go` - Added `evalImportExpression()` and refactored to `importModule()`

**Design decisions:**
1. **Backward compatibility**: Old `import("path")` syntax is preserved. When parser sees `import(`, it falls back to parsing as a function call.
2. **`import` is a keyword**: `import` is now a lexer keyword, enabling the new statement-like syntax.
3. **Auto-binding**: `import @std/math` automatically binds to `math` (last path segment). `import @std/math as M` binds to `M`.
4. **Shared logic**: Created `importModule()` function used by both old `evalImport()` and new `evalImportExpression()`.
5. **`as` remains contextual**: Checked by literal value in parser, not a separate token type. `let as = 5` still works.

**Tests:**
- `pkg/parsley/tests/import_syntax_test.go` - Tests for new syntax, aliases, destructuring, backward compat

## Related
- FEAT-038: Basil Namespace Cleanup (benefits from this)
- FEAT-037: Fragment Caching (can use `{Cache} = import @basil/cache`)
- `docs/parsley/design/modules.md` — Existing module design doc (if any)
