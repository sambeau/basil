---
id: PLAN-026
feature: FEAT-039
title: "Implementation Plan for Enhanced Import Syntax"
status: draft
created: 2025-12-07
---

# Implementation Plan: FEAT-039 Enhanced Import Syntax

## Overview
Change import syntax from `import("path")` to `import @path`, add aliasing (`as`), and destructuring. This is a core language change affecting lexer, parser, and evaluator.

## Prerequisites
- [ ] Current import implementation understood (function-call style)
- [ ] Syntax finalized: `import @path`, `import @(interpolated/{path})`

## Current Architecture

Import is currently a **builtin function** called like `import("std/math")`:
- Parser: `import(...)` parsed as CallExpression with `import` as identifier
- Evaluator: `evalImport()` handles the call in `evaluator.go:8340`

```go
// Current flow
CallExpression("import", [StringLiteral("std/math")])
  → evalImport(args, env)
  → resolves path, loads module
```

## New Syntax

```parsley
// Static imports
import @basil/auth
import @std/math
import @./local/file

// Aliased
import @basil/auth as Auth

// Destructured
{Login, Logout} = import @basil/auth

// Destructured with rename  
{Login as MyLogin} = import @basil/auth

// Dynamic (interpolated)
import @(./components/{name})
```

---

## Tasks

### Task 1: Lexer - Add `@` Token (if needed)
**Files**: `pkg/parsley/lexer/lexer.go`
**Estimated effort**: Small

Steps:
1. Check if `@` is already tokenized (used for `@./path`, `@2024-01-01`, etc.)
2. If not, add `AT` token type
3. Ensure `@` followed by identifier/path characters is handled

Current state: `@` likely already tokenized for path/date literals. Verify.

Tests:
- `@basil/auth` tokenizes correctly
- `@./local` tokenizes correctly
- `@(` starts dynamic expression

---

### Task 2: Parser - Add Import Statement
**Files**: `pkg/parsley/parser/parser.go`
**Estimated effort**: Large

This is the biggest change. Need to:

1. Recognize `import` as statement keyword (not just function name)
2. Parse module path after `import @`:
   ```
   ImportStmt = "import" "@" ModulePath [ "as" Identifier ]
   ModulePath = Identifier ( "/" Identifier )*
              | "." "/" RelativePath
              | "(" Expression ")"  // dynamic
   ```

3. Create new AST node `ImportStatement`:
   ```go
   type ImportStatement struct {
       Token     token.Token
       Path      Expression  // PathLiteral or dynamic Expression
       Alias     *Identifier // optional, for "as Alias"
       Dynamic   bool        // true if @(...)
   }
   ```

4. Handle destructuring as assignment:
   ```parsley
   {Login, Logout} = import @basil/auth
   ```
   This is already valid syntax if `import @basil/auth` returns an object.
   Parser sees: `DestructurePattern = Expression`

Steps:
1. In `parseStatement()`, check for `import` keyword
2. Add `parseImportStatement()`:
   ```go
   func (p *Parser) parseImportStatement() ast.Statement {
       stmt := &ast.ImportStatement{Token: p.curToken}
       
       p.nextToken() // consume 'import'
       
       if !p.expectPeek(token.AT) {
           return nil
       }
       
       if p.peekTokenIs(token.LPAREN) {
           // Dynamic: import @(expr)
           p.nextToken() // consume '('
           stmt.Path = p.parseExpression(LOWEST)
           stmt.Dynamic = true
           if !p.expectPeek(token.RPAREN) {
               return nil
           }
       } else {
           // Static: import @basil/auth
           stmt.Path = p.parseModulePath()
       }
       
       // Check for 'as Alias'
       if p.peekTokenIs(token.IDENT) && p.peekToken.Literal == "as" {
           p.nextToken() // consume 'as'
           p.nextToken() // move to alias
           stmt.Alias = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
       }
       
       return stmt
   }
   ```

3. Add `parseModulePath()` to handle `@basil/auth`, `@./local`, etc.
   - Consume tokens until whitespace, `as`, or newline
   - Build path string from identifier/slash sequence

Challenge: The `/` token is normally division. In import context, it's a path separator.

Tests:
- Parse `import @std/math` correctly
- Parse `import @basil/auth as Auth` correctly
- Parse `import @./local/file` correctly
- Parse `import @(./path/{var})` correctly
- Parse errors for malformed imports

---

### Task 3: AST - Add Import Node
**Files**: `pkg/parsley/ast/ast.go`
**Estimated effort**: Small

Steps:
1. Add `ImportStatement` struct:
   ```go
   type ImportStatement struct {
       Token   token.Token
       Path    Expression   // The path (static or dynamic)
       Alias   *Identifier  // Optional alias
   }
   
   func (is *ImportStatement) statementNode() {}
   func (is *ImportStatement) TokenLiteral() string { return is.Token.Literal }
   func (is *ImportStatement) String() string {
       var out bytes.Buffer
       out.WriteString("import @")
       out.WriteString(is.Path.String())
       if is.Alias != nil {
           out.WriteString(" as ")
           out.WriteString(is.Alias.Value)
       }
       return out.String()
   }
   ```

2. Add `ModulePath` expression type (or reuse existing path handling)

---

### Task 4: Evaluator - Handle Import Statement
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Medium

Steps:
1. Add case for `*ast.ImportStatement` in `Eval()`:
   ```go
   case *ast.ImportStatement:
       return evalImportStatement(node, env)
   ```

2. Implement `evalImportStatement()`:
   ```go
   func evalImportStatement(stmt *ast.ImportStatement, env *Environment) Object {
       // Evaluate path
       var pathStr string
       if stmt.Dynamic {
           pathObj := Eval(stmt.Path, env)
           pathStr = pathObj.(*String).Value
       } else {
           pathStr = stmt.Path.String() // or extract from PathLiteral
       }
       
       // Resolve and load module (reuse existing logic)
       module := loadModule(pathStr, env)
       
       // Bind to environment
       if stmt.Alias != nil {
           env.Set(stmt.Alias.Value, module)
       } else {
           // Use last path segment as name
           name := lastPathSegment(pathStr)
           env.Set(name, module)
       }
       
       return module
   }
   ```

3. Destructuring already works if `import @path` returns an object:
   ```parsley
   {Login} = import @basil/auth
   ```
   This is handled by existing destructure assignment logic.

Tests:
- `import @std/math` binds `math` to environment
- `import @std/math as M` binds `M` to environment
- `{floor} = import @std/math` binds `floor`

---

### Task 5: Update Module Path Resolution
**Files**: `pkg/parsley/evaluator/evaluator.go` (in `evalImport` or new function)
**Estimated effort**: Small

Steps:
1. Handle `@basil/` prefix → Basil runtime modules
2. Handle `@std/` prefix → Standard library (existing)
3. Handle `@./` and `@../` → Relative paths (existing logic)

```go
func resolveModulePath(pathStr string, env *Environment) (string, error) {
    switch {
    case strings.HasPrefix(pathStr, "basil/"):
        // Basil modules - handled specially
        return "basil/" + strings.TrimPrefix(pathStr, "basil/"), nil
    case strings.HasPrefix(pathStr, "std/"):
        // Standard library
        return pathStr, nil
    case strings.HasPrefix(pathStr, "./"), strings.HasPrefix(pathStr, "../"):
        // Relative to current file
        return resolveRelative(pathStr, env.Filename)
    default:
        return "", fmt.Errorf("unknown module path: %s", pathStr)
    }
}
```

---

### Task 6: Handle `as` Keyword
**Files**: `pkg/parsley/lexer/lexer.go`, `pkg/parsley/parser/parser.go`
**Estimated effort**: Small

Steps:
1. Decide: Is `as` a reserved keyword or contextual?
   - Recommend: **Contextual** (only after import or in destructure)
2. In parser, check for `as` by literal value, not token type:
   ```go
   if p.peekToken.Literal == "as" {
       // handle alias
   }
   ```

Tests:
- `let as = 5` still works (not reserved)
- `import @foo as Bar` parses alias correctly

---

### Task 7: Backward Compatibility / Migration
**Files**: Various
**Estimated effort**: Medium

The old syntax `import("path")` needs to either:
- A) Keep working (backward compat)
- B) Error with helpful message (clean break)

Recommend: **Option B** (pre-alpha, clean break)

Steps:
1. Remove or disable old `import` as builtin function
2. Add error message if old syntax detected:
   ```
   SyntaxError: import() syntax is deprecated. Use: import @path
   ```

Tests:
- Old syntax produces clear error
- All existing tests updated to new syntax

---

### Task 8: Update Existing Code
**Files**: `pkg/parsley/tests/*.go`, `examples/**/*.pars`, `docs/**`
**Estimated effort**: Large (many files)

Steps:
1. Search for `import("` across codebase
2. Replace with `import @` syntax:
   - `import("std/math")` → `import @std/math`
   - `import("./local")` → `import @./local`
3. Update destructuring:
   - `let {x} = import("std/y")` → `{x} = import @std/y`

This is mechanical but touches many files.

---

## Key Technical Challenges

### 1. `/` as Path Separator vs Division
In `import @basil/auth`, the `/` must be a path separator, not division operator.

Solution: Parser enters "path mode" after `import @`, treating `/` as separator until whitespace, `as`, or newline.

### 2. `@` Already Used for Literals
`@` starts paths (`@./file`), dates (`@2024-01-01`), durations (`@1h`).

Solution: After `import`, `@` starts a module path specifically. The lexer/parser context determines meaning.

### 3. Dynamic Paths with Interpolation
`import @(./components/{name})` requires parsing interpolated content.

Solution: Reuse existing `@(...)` interpolation logic for dates/paths.

### 4. Module Binding Names
`import @basil/auth` should bind to `auth` (last segment).
`import @./components/Button` should bind to `Button`.

Need logic to extract name from path.

---

## Validation Checklist
- [ ] All tests pass: `make check`
- [ ] New parser tests for import syntax
- [ ] All existing imports updated to new syntax
- [ ] Examples work with new syntax
- [ ] Error messages for old syntax are clear

## Risks
- Parser complexity: `/` disambiguation
- Many files to update
- May break user code (acceptable: pre-alpha)

## Order of Implementation
1. AST node (Task 3)
2. Lexer updates if needed (Task 1)
3. Parser (Task 2) - most complex
4. Evaluator (Task 4)
5. Path resolution (Task 5)
6. `as` handling (Task 6)
7. Remove old syntax (Task 7)
8. Update codebase (Task 8)

---

## Insight for FEAT Order

After writing this plan, the key insight is:

**FEAT-039 is independent but large**. It doesn't require FEAT-038 or FEAT-037, but it's the riskiest change (parser work, many file updates).

**FEAT-038 is small and isolated**. Just regex changes in one file.

**FEAT-037 requires evaluator changes** that partially overlap with what FEAT-039 needs (handling special tags, environment changes).

**Recommended order**:
1. **FEAT-038** first (smallest, proves namespace pattern)
2. **FEAT-039** second (establishes import syntax for cleaner examples)
3. **FEAT-037** third (uses both: namespace + potentially new import syntax in docs)

OR if you want to minimize risk:
1. **FEAT-038** (small, isolated)
2. **FEAT-037** (medium, evaluator work)
3. **FEAT-039** (large, risky parser work)

The second order saves the riskiest for last, when you have momentum.
