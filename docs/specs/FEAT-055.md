---
id: FEAT-055
title: "Parsley Namespace Cleanup - Final Reorganization"
status: draft
priority: high
created: 2025-12-09
author: "@human"
---

# FEAT-055: Parsley Namespace Cleanup - Final Reorganization

## Summary
Complete the final phase of Parsley namespace reorganization. This includes removing `len()`, renaming database constructors to `@` prefix, moving formatting/serialization to type methods, and adding path methods. This is the last opportunity to make breaking changes before API stability.

## Background

This feature completes the namespace cleanup started in earlier phases:
- **Phase 1 (FEAT-052)**: Removed 11 method-duplicate builtins ✅
- **Phase 2**: Renamed file builtins to final names (`JSON`, `YAML`, `CSV`, etc.) ✅
- **FEAT-054**: Replacing `now()` with `@now`/`@timeNow`/`@dateNow`/`@today` literals (planned)
- **Phase 3 (this feature)**: Final namespace reorganization

## User Story
As a Parsley developer, I want a clean, consistent namespace where:
- Types have standard methods (`.format()`, `.toJSON()`, `.length()`)
- External connections use `@` prefix (`@DB`, `@sqlite`)
- Functions follow a clear principle: constructors are global, operations are methods
- I can write idiomatic code without memorizing inconsistent function names

## Guiding Principles

1. **No deprecation; break things, fix things** — This is pre-alpha. Make all breaking changes now.
2. **Type constructors stay global** — Functions that create types remain global (`time()`, `url()`, `file()`, `money()`)
3. **Methods replace function forms** — If `arr.sort()` exists, remove `sort(arr)`
4. **Core mission: websites from data** — File reading, tags, dates, databases stay global
5. **Formatting is a method** — All types have their own `.format()` method
6. **Serialization standard** — All core types have `.toJSON()`

## Acceptance Criteria

### Remove `len()` Builtin
- [x] Remove `len()` builtin function
- [x] Users must use `string.length()` and `array.length()` methods

### Database Constructor Renames
- [x] `basil.sqlite` → `@DB` (Basil's built-in database)
- [x] `SQLITE` → `@sqlite` (external SQLite)
- [x] `POSTGRES` → `@postgres` (PostgreSQL)
- [x] `MYSQL` → `@mysql` (MySQL)
- [x] `SFTP` → `@sftp` (SFTP file system)
- [x] `COMMAND` → `@shell` (shell command execution)

### Path Methods
- [x] `publicUrl()` / `asset()` → `path.public()` (Basil-only: converts path under `public_dir` to web URL)
- [x] Add `path.toURL(prefix)` method (Parsley: converts path to URL with explicit prefix)
- [x] `match(path, pattern)` → `path.match(pattern)` method

### Formatting → Type Methods
- [x] `formatNumber(n, ...)` → `n.format(...)`
- [x] `formatCurrency(money, ...)` → `money.format(...)`
- [x] `formatDate(d, ...)` → `d.format(...)` (if not already present)
- [x] Remove global formatting functions

### JSON/CSV Serialization → Type Methods
- [x] `stringifyJSON(obj)` → `obj.toJSON()` (arrays, dicts serialize themselves)
- [x] `parseJSON(s)` → `s.parseJSON()` (string parses to object)
- [x] `stringifyCSV(table)` → `table.toCSV()`
- [x] `parseCSV(s)` → `s.parseCSV()` (string parses to table)
- [x] Remove global serialization functions

### Documentation & Tests
- [x] Update all documentation (reference.md, CHEATSHEET.md)
- [x] Update all tests to use new syntax
- [x] Update examples throughout codebase

## Design Decisions

- **Remove `len()` instead of deprecating** — Only works on strings/arrays; both have `.length()` methods; reduces namespace clutter
- **`@` prefix for connections** — Visually distinguishes external resources; `@DB` vs `@sqlite` clarifies built-in vs external
- **`path.public()` vs global `publicUrl()`** — Parsley has no knowledge of `public_dir` (Basil config); method keeps Basil-specific logic in Basil
- **`path.toURL(prefix)` in Parsley** — General-purpose path→URL conversion with explicit control
- **Methods for formatting** — Each type knows how to format itself; standard `.format()` interface
- **Methods for serialization** — `.toJSON()` enables transparent serialization; standard interface across types

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `pkg/parsley/evaluator/evaluator.go` — Remove `len()`, formatting functions, serialization functions
- `pkg/parsley/evaluator/methods.go` — Add `.format()`, `.toJSON()`, `.parseJSON()`, `.parseCSV()`, `.toCSV()`, `.match()`, `.public()`, `.toURL()`
- `pkg/parsley/lexer/lexer.go` — Add `@DB`, `@sqlite`, `@postgres`, `@mysql`, `@sftp`, `@shell` tokens
- `pkg/parsley/parser/parser.go` — Parse new `@` connection literals
- `server/` — Move `publicUrl`/`asset` logic to path method
- `pkg/parsley/tests/` — Update all affected tests
- `docs/parsley/reference.md` — Update documentation
- `docs/parsley/CHEATSHEET.md` — Update cheatsheet

### Dependencies
- **FEAT-054**: `now()` → `@now` must be completed first (establishes `@` literal pattern for special values)
- **FEAT-053**: `string.render()` (independent, can be done in parallel)

### Related Features
- **FEAT-054**: `now()` → `@now`/`@timeNow`/`@dateNow`/`@today` — Part of same namespace cleanup effort, establishes `@` literal precedent

## Implementation Phases

### Sub-task 1: Remove `len()` Builtin
```go
// Remove from getBuiltins():
// "len": { ... }

// Users must use:
items.length()
name.length()
```

### Sub-task 2: Database Constructor Renames
New lexer tokens:
```go
// Tokens for connection literals
DB_LITERAL      // @DB
SQLITE_LITERAL  // @sqlite
POSTGRES_LITERAL // @postgres
MYSQL_LITERAL   // @mysql
SFTP_LITERAL    // @sftp
SHELL_LITERAL   // @shell
```

Usage:
```parsley
// Old syntax
import @std/basil
let db = basil.sqlite <=> { /* query */ }

// New syntax
let db = @DB <=> { /* query */ }           // Basil's built-in
let ext = @sqlite("./data.db") <=> { /* query */ }  // External
```

### Sub-task 3: Path Methods
```go
// In evalPathMethod():
case "public":
    // Basil-only: convert path under public_dir to web URL
    // Requires access to Basil config
    
case "toURL":
    // Parsley: convert path to URL with explicit prefix
    // path.toURL("/static") → "/static/images/logo.png"
    
case "match":
    // Move from global match() to method
    // @/users/123.match("/users/:id") → {id: "123"}
```

### Sub-task 4: Formatting Methods
Ensure all formattable types have `.format()`:
- Integer, Float → number formatting with locale
- Money → currency formatting
- Time/Date/Datetime → date/time formatting (already exists)

### Sub-task 5: Serialization Methods
```go
// Array and Dictionary methods:
case "toJSON":
    // Serialize to JSON string

// String methods:
case "parseJSON":
    // Parse JSON string to object (array or dictionary)

case "parseCSV":
    // Parse CSV string to table (array of dictionaries)

// Table methods:
case "toCSV":
    // Serialize table to CSV string
```

## Examples: Before and After

### Length
```parsley
// ❌ Before:
let count = len(items)

// ✅ After:
let count = items.length()
```

### Database Connections
```parsley
// ❌ Before:
import @std/basil
let results = basil.sqlite <=> {
    SELECT * FROM users
}

// ✅ After:
let results = @DB <=> {
    SELECT * FROM users
}

// External database
let ext = @sqlite("./data.db") <=> { SELECT * FROM items }
```

### Path Operations
```parsley
// ❌ Before:
let url = publicUrl(@./public/logo.png)
let params = match("/users/123", "/users/:id")

// ✅ After:
let url = @./public/logo.png.public()         // Basil-only
let url = @./logo.png.toURL("/static")        // Parsley with prefix
let params = @/users/123.match("/users/:id")
```

### Formatting
```parsley
// ❌ Before:
let formatted = formatNumber(price, {decimals: 2})
let date = formatDate(today, "long")

// ✅ After:
let formatted = price.format({decimals: 2})
let date = today.format("long")
```

### Serialization
```parsley
// ❌ Before:
let json = stringifyJSON(data)
let obj = parseJSON(jsonString)
let csv = stringifyCSV(table)

// ✅ After:
let json = data.toJSON()
let obj = jsonString.parseJSON()
let csv = table.toCSV()
```

## Final Global Namespace (Target)

After this feature completes, the global namespace will contain:

### Core Language
- `import`, `fail`, `log`, `logLine`, `print`, `println`, `repr`

### Type Constructors  
- `tag`, `time`, `url`, `file`, `dir`, `regex`, `money`

### Datetime Literals (FEAT-054)
- `@now`, `@timeNow`, `@dateNow`, `@today`

### File Reading (Core to data-driven sites)
- `fileList`, `JSON`, `YAML`, `CSV`, `lines`, `text`, `bytes`, `SVG`, `markdown`

### Database/External Connections
- `@DB` (Basil-only), `@sqlite`, `@postgres`, `@mysql`, `@sftp`, `@shell`

### Type Conversion
- `toInt`, `toFloat`, `toNumber`, `toString`, `toDebug`, `toArray`, `toDict`

**Total: ~35 global builtins** (down from 59, with better organization)

### Edge Cases & Constraints
1. **`path.public()` is Basil-only** — Requires access to Basil's `public_dir` config; not available in standalone Parsley
2. **`@DB` is Basil-only** — Built-in database only available in Basil context
3. **Backward compatibility** — No deprecation period; this is pre-alpha
4. **Method chaining** — Ensure all new methods return appropriate types for chaining

## Implementation Notes
*To be added during implementation*

## Related
- **FEAT-052**: Phase 1 - Removed method-duplicate builtins ✅
- **FEAT-053**: `string.render()` method (parallel work)
- **FEAT-054**: `now()` → `@now` datetime literals (prerequisite for `@` pattern)
- **Design Doc**: `docs/design/namespace-cleanup.md` (source document)
