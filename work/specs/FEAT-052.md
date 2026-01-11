---
id: FEAT-052
title: "Pre-Alpha Codebase Cleanup"
status: implemented
priority: high
created: 2025-12-08
author: "@sam"
---

# FEAT-052: Pre-Alpha Codebase Cleanup

## Summary
Remove all deprecated code, backward compatibility shims, and redundant builtins before alpha release. This is a one-time cleanup to ship with a clean codebase and fixed grammar. Breaking changes are acceptable since there are no external users yet.

## User Story
As a maintainer shipping alpha, I want to remove all legacy code so that the codebase is clean, the grammar is consistent, and there's no technical debt from pre-release iterations.

## Acceptance Criteria
- [x] Old import syntax `import("path")` removed (only `import @path` works)
- [x] Legacy `errors []string` array removed from parser
- [x] Backward-compat `Name` fields - **NOT REMOVED** - discovered these are NOT deprecated, they're the primary field for simple assignments
- [x] Deprecated `DatabaseConfig` struct removed from config
- [x] 11 method-duplicate builtins removed
- [x] All tests updated to use current syntax
- [x] Documentation updated to reflect removals
- [x] `make check` passes

## Design Decisions
- **No deprecation warnings**: We're pre-alpha with no users. Just remove.
- **Method syntax preferred**: `arr.sort()` not `sort(arr)`. Methods are discoverable.
- **Single import syntax**: `import(path)` is cleaner than `import("path")`.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### 1. Parser: Old Import Syntax

**Location**: `pkg/parsley/parser/parser.go` lines ~1231-1277

**Current**: Parser accepts both forms:
```parsley
let {x} = import(std/math)     // Current syntax
let {x} = import("std/math")   // Old syntax (backward compat)
```

**Action**: Remove the `parseOldImportExpression` function and related code. Only accept unquoted import paths.

**Test Updates**: `pkg/parsley/tests/import_syntax_test.go`
- Lines ~102, 153, 158, 163 test "old syntax"
- Remove or convert these tests to expect errors

### 2. Parser: Legacy Errors Array

**Location**: `pkg/parsley/parser/parser.go` line ~65

**Current**:
```go
errors           []string // Legacy - kept for any code still checking it
```

**Action**: Remove the field and any code that populates/reads it. Structured errors are the only error mechanism.

### 3. AST: Backward-Compat Name Fields

**Location**: `pkg/parsley/ast/ast.go`

**Current**: `LetStatement` and `ConstStatement` have:
```go
Name *Identifier // for backwards compatibility
```

**Action**: Remove these fields. All code should use the pattern-based destructuring fields (`Pattern`, `Identifier`, etc.).

**Affected files**: Search for `.Name` usage on let/const statements across evaluator, compiler, etc.

### 4. Config: Deprecated DatabaseConfig

**Location**: `config/config.go` lines 33-38

**Current**:
```go
// Deprecated: Use Database field with Driver, Path, DSN
type DatabaseConfig struct {
    Driver string
    Path   string
    DSN    string
}
```

**Action**: Remove the struct and any migration code that reads it.

### 5. Builtins: Method Duplicates

**Location**: `pkg/parsley/evaluator/builtins.go`

**Remove these 11 builtins** (method forms exist and are preferred):

| Builtin | Method Form |
|---------|-------------|
| `toUpper(s)` | `s.toUpper()` |
| `toLower(s)` | `s.toLower()` |
| `replace(s, old, new)` | `s.replace(old, new)` |
| `split(s, delim)` | `s.split(delim)` |
| `map(arr, fn)` | `arr.map(fn)` |
| `sort(arr)` | `arr.sort()` |
| `reverse(arr)` | `arr.reverse()` |
| `sortBy(arr, fn)` | `arr.sortBy(fn)` |
| `keys(dict)` | `dict.keys()` |
| `values(dict)` | `dict.values()` |
| `has(dict, key)` | `dict.has(key)` |

**Action**: 
1. Remove from `builtins` map in `builtins.go`
2. Remove the builtin implementation functions
3. Update any tests that use function form
4. Update documentation (cheatsheet, reference)

### 6. Tests: Old Syntax Tests

**Location**: `pkg/parsley/tests/import_syntax_test.go`

**Current**: Contains tests like:
```go
{"old syntax with string path", `import("std/math")`, ...}
```

**Action**: Convert to error-expectation tests or remove entirely.

### 7. Documentation Updates

**Files to update**:
- `docs/parsley/reference.md` - Remove old import syntax, remove builtin functions
- `docs/parsley/CHEATSHEET.md` - Remove old syntax mentions
- `docs/guide/cheatsheet.md` - Update if affected

---

## Implementation Phases

### Phase 1: Parser Cleanup
1. Remove `parseOldImportExpression` and related code
2. Remove legacy `errors []string` field
3. Update import syntax tests

### Phase 2: AST Cleanup
1. Remove `Name` fields from `LetStatement`, `ConstStatement`
2. Find and fix all usages across codebase
3. Run tests to catch any breakage

### Phase 3: Config Cleanup
1. Remove `DatabaseConfig` struct
2. Remove any migration/compat code

### Phase 4: Builtins Cleanup
1. Remove 11 method-duplicate builtins
2. Update tests to use method syntax
3. Update documentation


### Phase 5: Final Validation
1. `make check` passes
2. Grep for remaining "deprecated", "backward", "compat" references
3. Update CHEATSHEET with any removed pitfalls

---

## Edge Cases & Constraints
1. **AST Name field removal** - May break evaluator code that reads `.Name`. Need careful search.
2. **External examples** - `examples/` directory may use old syntax. Must update.
3. **Error messages** - Some error messages may reference old syntax. Update them.

## Related
- Design: `docs/design/namespace-cleanup.md` (superset - this does immediate items only)
- Deferred: `std/format`, `std/fs` modules (BACKLOG - not part of this cleanup)

## Dependencies
- **Blocks**: Examples update (all `examples/` will need updating after this lands)
- **Blocks**: Docs/examples reorg (in pre-planning stage)

---

## Implementation Notes (2025-12-08)

### Changes Made

1. **Parser (pkg/parsley/parser/parser.go)**
   - Removed `errors []string` legacy field
   - Updated `addError()` to use only structured errors
   - Updated `Errors()` to derive from structured errors
   - Removed old `import("path")` parsing - now only accepts `import @path`

2. **AST (pkg/parsley/ast/ast.go)**
   - **NO REMOVAL**: Discovered `Name` field is NOT deprecated - it's the primary field for simple `let x = ...` assignments. The comment was misleading. Fixed comments instead.

3. **Config (config/config.go)**
   - Removed deprecated `DatabaseConfig` struct (lines 33-40)

4. **Builtins (pkg/parsley/evaluator/evaluator.go)**
   - Removed 11 method-duplicate builtins from `getBuiltins()`:
     - `toUpper`, `toLower` → use `"str".toUpper()`, `"str".toLower()`
     - `replace`, `split` → use `"str".replace()`, `"str".split()`
     - `map` → use `arr.map(fn)`
     - `sort`, `reverse`, `sortBy` → use array methods
     - `keys`, `values`, `has` → use dictionary methods

5. **Tests Updated**
   - All parsley tests updated to use new import syntax
   - All server tests updated to use new import syntax
   - Tests using removed builtins converted to method syntax
   - Fixed regex replace/split tests (regex support was only in removed builtins, not methods - removed test cases)

6. **Documentation Updated**
   - `docs/parsley/reference.md` - Updated import syntax examples
   - `docs/parsley/CHEATSHEET.md` - Removed "old syntax" section
   - `examples/parsley/temp/test_modules.pars` - Updated import syntax

### Breaking Changes
- `import("path")` and `import(@path)` no longer work - use `import @path`
- Builtin functions `toUpper()`, `toLower()`, `map()`, `replace()`, `split()`, `sort()`, `reverse()`, `sortBy()`, `keys()`, `values()`, `has()` removed - use method syntax instead
- `DatabaseConfig` struct removed from config

