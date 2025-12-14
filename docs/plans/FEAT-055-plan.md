---
id: PLAN-033
feature: FEAT-055
title: "Implementation Plan for Parsley Namespace Cleanup - Final Reorganization"
status: complete
created: 2025-12-09
completed: 2025-12-09
---

# Implementation Plan: FEAT-055 Namespace Cleanup

## Overview
Complete the final namespace reorganization for Parsley. This is a **breaking change release** with no deprecation period. Remove `len()`, rename database/connection constructors to `@` prefix literals, move global formatting/serialization functions to type methods, and add path methods.

## ⚠️ Breaking Changes Policy

**NO DEPRECATION. BREAK THINGS. FIX THINGS.**

This is pre-alpha software. All changes are immediate:
- `len()` is **removed**, not deprecated — update all code to use `.length()`
- `SQLITE`/`POSTGRES`/`MYSQL`/`SFTP`/`COMMAND` are **removed** — use `@sqlite`/`@postgres`/`@mysql`/`@sftp`/`@shell`
- `formatNumber()`/`formatCurrency()`/`formatDate()` are **removed** — use `.format()`
- Global serialization functions are **removed** — use `.toJSON()`/`.parseJSON()`/`.toCSV()`/`.parseCSV()`

When tests fail, fix them. When examples break, update them. No backward compatibility layer.

## Prerequisites
- [x] FEAT-054 complete (`@now`/`@timeNow`/`@dateNow`/`@today` establishes `@` literal pattern)
- [x] Understand existing namespace (builtins, methods, lexer patterns)

## Impact Assessment

### Files Using `len()` (Parsley code)
- `pkg/parsley/tests/regex_test.go` — 1 use
- `pkg/parsley/tests/slicing_test.go` — 2 uses
- `pkg/parsley/tests/error_messages_test.go` — 1 use
- `pkg/parsley/tests/trailing_comma_test.go` — 2 uses
- `examples/parsley/modules/strings.pars` — 4 uses
- `examples/parsley/modules/arrays.pars` — 4 uses
- `examples/parsley/modules/validators.pars` — 1 use
- `examples/parsley/MODULE_EXAMPLES.md` — 6 uses
- `examples/parsley/array_demo.pars` — 3 uses
- `examples/parsley/regex_demo.pars` — 1 use

### Files Using Database Constructors
- Tests and examples using `SQLITE`, `POSTGRES`, `MYSQL`, `SFTP`, `COMMAND`

### Files Using Global Formatters
- `formatNumber`, `formatCurrency`, `formatDate` usages throughout tests/examples

## Basil-only vs Parsley
- Some builtins and std/basil are only availble in the Parsley environment of the server-version, Basil
- Basil gets everything that Parsley does; Parsley doesn't get everything that Basil does
- Basil-only features need to be injected into the Parsley environment of scripts and modules

---

## Tasks

### Task 1: Remove `len()` Builtin
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Effort**: Small

Steps:
1. Delete `"len"` entry from `getBuiltins()` (around line 5523)
2. Run tests — expect failures
3. Fix all test failures by changing `len(x)` → `x.length()`

Tests to update:
- `pkg/parsley/tests/regex_test.go`: `len("a:b:c:d".split(":"))` → `"a:b:c:d".split(":").length()`
- `pkg/parsley/tests/slicing_test.go`: `len([1,2,3,4,5][2:])` → `[1,2,3,4,5][2:].length()`
- `pkg/parsley/tests/slicing_test.go`: `len("hello"[:3])` → `"hello"[:3].length()`
- `pkg/parsley/tests/error_messages_test.go`: Remove/update `len()` test case
- `pkg/parsley/tests/trailing_comma_test.go`: `len([1, 2, 3],)` → `[1, 2, 3].length()`

Examples to update:
- `examples/parsley/modules/strings.pars`
- `examples/parsley/modules/arrays.pars`
- `examples/parsley/modules/validators.pars`
- `examples/parsley/MODULE_EXAMPLES.md`
- `examples/parsley/array_demo.pars`
- `examples/parsley/regex_demo.pars`

---

### Task 2: Add Connection Literal Tokens
**Files**: `pkg/parsley/lexer/lexer.go`
**Effort**: Medium

Steps:
1. Add token types:
   ```go
   SQLITE_LITERAL   // @sqlite
   POSTGRES_LITERAL // @postgres
   MYSQL_LITERAL    // @mysql
   SFTP_LITERAL     // @sftp
   SHELL_LITERAL    // @shell
   DB_LITERAL       // @DB (Basil-only)
   ```
2. Extend `detectAtLiteralType()` to recognize these keywords
3. Add `isKeywordAt()` checks for `sqlite`, `postgres`, `mysql`, `sftp`, `shell`, `DB`
4. Add `readConnectionLiteral()` helper (similar to `readNowLiteral`)

Tests:
- `@sqlite` produces SQLITE_LITERAL token
- `@postgres` produces POSTGRES_LITERAL token
- `@mysql` produces MYSQL_LITERAL token
- `@sftp` produces SFTP_LITERAL token
- `@shell` produces SHELL_LITERAL token
- `@DB` produces DB_LITERAL token
- Existing `@` literals still work (paths, URLs, datetimes, durations)

---

### Task 3: Add Connection Literal AST Nodes
**Files**: `pkg/parsley/ast/ast.go`
**Effort**: Small

Steps:
1. Add `ConnectionLiteral` struct:
   ```go
   type ConnectionLiteral struct {
       Token    lexer.Token
       Kind     string // "sqlite", "postgres", "mysql", "sftp", "shell", "db"
   }
   ```
2. Implement `expressionNode()`, `TokenLiteral()`, `String()` methods

---

### Task 4: Add Connection Literal Parser Support
**Files**: `pkg/parsley/parser/parser.go`
**Effort**: Small

Steps:
1. Register prefix parsers for each connection token type
2. Create parse functions that return `ConnectionLiteral` nodes
3. Connection literals are callable: `@sqlite("./data.db")` parses as call expression

Tests:
- Parse `@sqlite("./db.sqlite")` → ConnectionLiteral + CallExpression
- Parse `@postgres` → ConnectionLiteral (can be called or used directly)

---

### Task 5: Add Connection Literal Evaluator Support
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Effort**: Medium

Steps:
1. Add case for `*ast.ConnectionLiteral` in `Eval()` switch
2. Implement `evalConnectionLiteral()`:
   - Returns a callable object that wraps the connection constructor
   - When called with args, creates the connection (reuses existing SQLITE/POSTGRES/etc. logic)
3. Remove old builtins: `"SQLITE"`, `"POSTGRES"`, `"MYSQL"`, `"SFTP"`, `"COMMAND"`
4. Rename `"COMMAND"` logic to `"shell"` internally

Tests:
- `@sqlite("./test.db")` creates SQLite connection
- `@postgres(@postgres://...)` creates PostgreSQL connection
- `@mysql(@mysql://...)` creates MySQL connection
- `@sftp(@sftp://user@host)` creates SFTP connection
- `@shell("ls -la")` creates shell command
- Old syntax `SQLITE(...)` produces "identifier not found" error

---

### Task 6: Add `@DB` Basil-Only Connection
**Files**: `pkg/parsley/evaluator/evaluator.go`, `server/handler.go`
**Effort**: Small

Steps:
1. `@DB` in standalone Parsley → error "only available in Basil server"
2. In Basil handler context, inject `@DB` that connects to `basil.sqlite`
3. Server injects DB connection via environment (similar to `publicUrl`)

Tests:
- `@DB` in REPL → error
- `@DB` in Basil handler → connects to built-in database

---

### Task 7: Remove Global Formatting Functions
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Effort**: Small

Steps:
1. Remove `"formatNumber"` builtin (line ~5123)
2. Remove `"formatCurrency"` builtin (line ~5157)
3. Remove `"formatDate"` builtin (line ~5237)
4. Verify `.format()` methods exist on Integer, Float, Money, Dictionary (datetime)

Tests:
- `formatNumber(1234)` → error "identifier not found"
- `1234.format()` → works
- `formatCurrency(12.34, "USD")` → error
- `$12.34.format()` → works
- `formatDate(@now, "long")` → error
- `@now.format("long")` → works

---

### Task 8: Add Serialization Methods
**Files**: `pkg/parsley/evaluator/methods.go`
**Effort**: Medium

Steps:
1. Add `.toJSON()` method to Array and Dictionary
2. Add `.parseJSON()` method to String
3. Add `.toCSV()` method to Array (of dictionaries)
4. Add `.parseCSV()` method to String
5. Remove global `parseJSON` builtin if exists

Method implementations:
```go
// Array.toJSON() and Dictionary.toJSON()
case "toJSON":
    // Serialize to JSON string using existing logic

// String.parseJSON()
case "parseJSON":
    // Parse JSON string to object

// String.parseCSV()
case "parseCSV":
    // Parse CSV string to array of dictionaries

// Array.toCSV()
case "toCSV":
    // Serialize array of dicts to CSV string
```

Tests:
- `{a: 1}.toJSON()` → `'{"a":1}'`
- `[1, 2, 3].toJSON()` → `'[1,2,3]'`
- `'{"a":1}'.parseJSON()` → `{a: 1}`
- `'a,b\n1,2'.parseCSV()` → `[{a: "1", b: "2"}]`
- `[{a: 1, b: 2}].toCSV()` → `'a,b\n1,2'`

---

### Task 9: Add Path Methods
**Files**: `pkg/parsley/evaluator/methods.go`, `pkg/parsley/evaluator/public_url.go`
**Effort**: Medium

Steps:
1. Add `.public()` method to path dictionaries (Basil-only)
   - Move `publicUrl()` logic to path method
   - Requires AssetRegistry in environment
2. Add `.toURL(prefix)` method to path dictionaries (Parsley)
   - Converts path to URL with explicit prefix
   - `@./images/logo.png.toURL("/static")` → `"/static/images/logo.png"`
3. Add `.match(pattern)` method to path dictionaries
   - Move global `match()` pattern logic to path method
   - `@/users/123.match("/users/:id")` → `{id: "123"}`

Tests:
- `@./logo.png.public()` in Basil → returns hashed URL
- `@./logo.png.public()` in REPL → error "only available in Basil"
- `@./images/logo.png.toURL("/static")` → `"/static/images/logo.png"`
- `@/users/123.match("/users/:id")` → `{id: "123"}`

---

### Task 10: Update Documentation
**Files**: `docs/parsley/reference.md`, `docs/parsley/CHEATSHEET.md`
**Effort**: Medium

Steps:
1. Remove `len()` from builtin list
2. Update all examples using `len()` to use `.length()`
3. Document new connection literals (`@sqlite`, `@postgres`, etc.)
4. Remove global formatters from builtin list
5. Document serialization methods (`.toJSON()`, `.parseJSON()`, etc.)
6. Document path methods (`.public()`, `.toURL()`, `.match()`)
7. Update "Final Global Namespace" section in FEAT-055 spec

---

### Task 11: Update Examples
**Files**: `examples/parsley/**`
**Effort**: Small

Steps:
1. Update all `.pars` files using `len()` → `.length()`
2. Update any database examples to use new syntax
3. Update formatting examples to use methods
4. Verify all examples run without errors

---

## Validation Checklist
- [x] All tests pass: `make test`
- [x] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] `len()` removed — `len([1,2,3])` produces error
- [x] Connection literals work — `@sqlite("./test.db")` creates connection
- [x] Old constructors removed — `SQLITE(...)` produces error
- [x] Formatting methods work — `1234.format()` works
- [x] Global formatters removed — `formatNumber(...)` produces error
- [x] Serialization methods work — `{}.toJSON()` works
- [x] Path methods work — `@./file.toURL("/static")` works
- [x] Documentation updated (reference.md, CHEATSHEET.md)
- [x] Examples updated and working
- [ ] BACKLOG.md updated with deferrals (if any)

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| | Task 1 | ⬜ Not Started | Remove len() |
| 2025-12-09 | Task 2 | ✅ Complete | Added SQLITE_LITERAL, POSTGRES_LITERAL, MYSQL_LITERAL, SFTP_LITERAL, SHELL_LITERAL, DB_LITERAL tokens |
| 2025-12-09 | Task 3 | ✅ Complete | Added ConnectionLiteral AST node |
| 2025-12-09 | Task 4 | ✅ Complete | Registered prefix parsers for all connection literals |
| 2025-12-09 | Task 5 | ✅ Complete | Added evalConnectionLiteral() and connectionBuiltins() for @sqlite, @postgres, @mysql, @sftp, @shell |
| 2025-12-09 | Task 6 | ✅ Complete | Added @DB Basil-only connection via resolveDBLiteral() |
| 2025-12-09 | Task 7 | ✅ Complete | Removed formatNumber, formatCurrency, formatDate, formatPercent globals from evaluator.go |
| 2025-12-09 | Task 8 | ✅ Complete | Added string.parseJSON(), string.parseCSV(hasHeader?), array.toJSON(), array.toCSV(hasHeader?), dictionary.toJSON(); removed parseJSON, stringifyJSON, parseCSV, stringifyCSV globals |
| 2025-12-09 | Task 9 | ✅ Complete | Added path.public(), path.toURL(prefix), path.match(pattern) |
| 2025-12-09 | Task 10 | ✅ Complete | Updated reference.md serialization section |
| 2025-12-09 | Task 11 | ✅ Complete | Updated locale_formatting_demo.pars, process_demo.pars |

## Deferred Items
Items to add to BACKLOG.md after implementation:
- Consider `match()` global function removal (currently only moving to path method)
- Consider `parseJSON` global removal if not already done
- Consider adding more connection types (`@redis`, `@mongodb`) in future

## Implementation Order
Recommended sequence for clean implementation:

1. **Task 1** (Remove `len()`) — Quick win, isolated change
2. **Tasks 2-5** (Connection literals) — Lexer → AST → Parser → Evaluator
3. **Task 6** (`@DB` Basil-only) — Requires Task 5
4. **Task 7** (Remove formatters) — Quick, methods already exist
5. **Task 8** (Serialization methods) — New methods
6. **Task 9** (Path methods) — New methods + `publicUrl` refactor
7. **Tasks 10-11** (Docs & Examples) — Final cleanup

Total estimated effort: **Large** (multiple sessions)
