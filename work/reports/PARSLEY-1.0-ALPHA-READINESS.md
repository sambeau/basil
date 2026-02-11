# Parsley 1.0 Alpha Readiness Report

**Date:** 2026-02-08  
**Status:** Analysis complete, action items identified  
**Scope:** Parsley language and `pars` CLI/REPL (not Basil server)

## Executive Summary

This report assesses the readiness of Parsley and `pars` for a 1.0 Alpha declaration. Three areas require attention before release:

1. **Database Drivers** — MySQL and Postgres are documented but non-functional (drivers not installed)
2. **`describe()` Brittleness** — Introspection metadata is manually maintained and easily falls out of sync
3. **CLI Enhancements** — Missing common features expected by users of other languages

Additionally, several backlog items should be addressed for a polished Alpha release.

---

## 1. Database Drivers (MySQL/Postgres)

### Current Status: BLOCKING

The code for `@postgres()` and `@mysql()` exists in `evaluator.go` (lines 1877-2020), but the actual drivers are **not installed** in `go.mod`.

**What's in `go.mod`:**
```go
modernc.org/sqlite v1.40.1  // Only SQLite driver present
```

**What's missing:**
- `github.com/lib/pq` (PostgreSQL)
- `github.com/go-sql-driver/mysql` (MySQL)

**Impact:** Users attempting `@postgres()` or `@mysql()` will get runtime errors because Go's `database/sql` requires driver registration via import side-effects.

### Evidence

From `work/parsley/design/Database Implementation Status.md`:

```
## PostgreSQL and MySQL Support

Status: Stub implementations only (drivers not included)

- `POSTGRES()` and `MYSQL()` functions exist
- Will attempt to open connections using Go's `database/sql`
- Requires external drivers to be installed
```

### Recommended Action

**Create FEAT spec:** Add PostgreSQL and MySQL drivers to `go.mod` and verify functionality.

**Tasks:**
1. Add `github.com/lib/pq` to `go.mod`
2. Add `github.com/go-sql-driver/mysql` to `go.mod`
3. Add import side-effects in evaluator.go (or separate file)
4. Write integration tests (can use Docker containers or test databases)
5. Update documentation to reflect "supported" rather than "planned"

**Estimated effort:** 2-4 hours (drivers are standard, implementation already done)

---

## 2. `describe()` Brittleness

### Current Status: HIGH PRIORITY

The `describe()` and `inspect()` functions rely on two manually-maintained maps in `introspect.go`:

```go
// Lines 40-127
var TypeProperties = map[string][]PropertyInfo{...}

// Lines 128-340
var TypeMethods = map[string][]MethodInfo{...}
```

These maps are **not derived from actual method implementations** in `methods.go`. If someone adds, removes, or renames a method, they must manually update both files. There's no validation that the maps match reality.

### The Risk

- New methods added to `methods.go` won't appear in `describe()` output
- Removed methods remain listed in `describe()` output
- Typos in method names go undetected
- Users lose trust in introspection if it's inaccurate

### Options Analysis

| Option | Pros | Cons | Runtime Cost | Recommendation |
|--------|------|------|--------------|----------------|
| **(1) Validation tests** | Quick to implement, catches drift | Requires maintenance, doesn't fix root cause | None (test-time only) | **Must have** |
| **(2) Generate from source** | Guaranteed accuracy | Build complexity, parsing Go code | None (build-time only) | Worth exploring |
| **(3) Runtime derivation** | Always accurate, single source of truth | Requires some refactoring | See analysis below | See analysis below |
| **(4) Accept as documentation** | No work required | Unacceptable for 1.0 | None | **Rejected** |

### Runtime Cost Analysis

The concern about "runtime cost" depends on which sub-option of runtime derivation we choose:

| Sub-option | When cost is incurred | Cost magnitude | Acceptable? |
|------------|----------------------|----------------|-------------|
| **3a. Probe dispatch** | Every `describe()` call | High (N method calls per type) | ❌ No |
| **3b. Declarative registry** | Never (same as current) | Zero | ✅ Yes |
| **3c. Code generation** | Never (build-time only) | Zero | ✅ Yes |

**Current implementation cost:** The existing `TypeMethods` and `TypeProperties` maps are package-level variables initialized at program start. Looking them up is O(1) map access. This is negligible.

**Option 3b (declarative registry) cost:** Would be identical to current—package-level maps initialized at start, O(1) lookup. The only difference is that method dispatch would *also* read from these maps instead of using switch statements. This could actually be *faster* than switch for types with many methods (map lookup is O(1), switch is O(n) in worst case).

**Option 3a (probing) cost:** This would be expensive—calling each potential method name and catching errors. Not recommended for production `describe()`, but acceptable for validation tests that run only during development.

**Bottom line:** Options 3b and 3c have **zero runtime cost** compared to today. Option 3b may even provide a micro-optimization for method dispatch.

### Option 3: Runtime Derivation — Detailed Analysis

**Does this require building a reflection system?**

Not necessarily a full reflection system, but it does require *some* form of runtime introspection. Current options:

**3a. Probe method dispatch table**
The `evalMethod()` function in `methods.go` uses a large switch statement. We could:
- Create a test harness that attempts to call each method name on each type
- Methods that don't error with `unknown method` are valid
- Downside: Doesn't capture arity or descriptions
- **Runtime cost: HIGH** — Would require N method call attempts per `describe()` invocation

**3b. Register methods declaratively**
Refactor `methods.go` to register methods in a data structure:

```go
var StringMethods = MethodRegistry{
    "toUpper": {Fn: stringToUpper, Arity: "0", Desc: "Convert to uppercase"},
    "toLower": {Fn: stringToLower, Arity: "0", Desc: "Convert to lowercase"},
    ...
}
```

Then `describe()` reads from this same registry, and method dispatch looks up `StringMethods[methodName]`.
- Pros: Single source of truth, descriptions stay with code
- Cons: Significant refactoring of methods.go (~2000 lines)
- **Runtime cost: ZERO** — Maps are initialized once at startup, lookup is O(1)
- **Potential benefit:** Map dispatch may be faster than large switch statements for types with many methods

**3c. Code generation from method dispatch**
Parse `methods.go` at build time, extract method names from switch cases, generate `introspect_generated.go`.
- Pros: Accurate method names, no manual sync required
- Cons: Can't extract descriptions from code (would need to add comments or annotations), adds build complexity
- **Runtime cost: ZERO** — Generated code is identical to hand-written maps

### Recommended Approach

**Phase 1 (Pre-Alpha):** Implement Option 1 — validation tests
- Write a test that calls each documented method on a sample value
- Fail if method doesn't exist or signature doesn't match
- Quick to implement, immediately catches drift

**Phase 2 (Pre-Alpha):** Implement Option 3b — declarative method registry
- Refactor `methods.go` to use declarative registry
- Single source of truth for method dispatch AND introspection
- Makes adding methods self-documenting

**Rationale for Phase 2 being Pre-Alpha:**

Parsley is a new language. For adoption, users need reliable tools to discover and learn the language. An introspection system that can fall out of sync undermines trust and makes learning harder. The declarative registry approach:

1. **Guarantees accuracy** — `describe()` output is always correct because it reads from the same data that drives method dispatch
2. **Reduces maintenance burden** — Adding a method is a single change, not two files to keep in sync
3. **Enables richer tooling** — IDE plugins, documentation generators, and the help system can all read from the same registry
4. **Supports AI assistance** — See note below

### AI Discoverability Considerations

As AI coding assistants become more prevalent, we should consider how AIs discover and learn Parsley syntax. The declarative registry enables:

1. **Machine-readable method catalog** — A single authoritative source that tools (including AI agents) can query
2. **Structured output from help system** — Consider adding `--json` flag to `pars describe` for programmatic access
3. **Self-documenting language** — AI can use `:describe` in REPL or `pars describe` to learn available methods
4. **Consistent documentation** — Descriptions in registry flow to help system, docs, and AI training data

**Future consideration (not 1.0):** A `pars introspect --json` command that dumps the entire method/property/builtin catalog in machine-readable format. This would be valuable for:
- AI agent context windows (include available methods for relevant types)
- Documentation generation
- IDE/LSP integration
- Automated testing

### Tasks for Phase 1

1. Create `introspect_validation_test.go`
2. For each type in `TypeMethods`, create a sample value
3. Attempt to call each documented method with appropriate arity
4. Verify no "unknown method" errors
5. Optionally: verify return types match descriptions

**Estimated effort:** 4-6 hours

### Tasks for Phase 2

1. Design `MethodRegistry` struct with `Fn`, `Arity`, `Description` fields
2. Create registry maps for each type (e.g., `StringMethods`, `ArrayMethods`)
3. Refactor `evalMethod()` to dispatch via registry lookup instead of switch
4. Update `describe()`/help system to read from registries
5. Remove `TypeMethods` map (now redundant)
6. Update tests

**Estimated effort:** 8-12 hours (significant refactoring of methods.go ~2000 lines)

---

## 3. CLI Enhancements

### Current `pars` Features (Good!)

- ✅ REPL with history, tab completion, `:help` commands
- ✅ `pars fmt` code formatter with `-w`, `-d`, `-l` flags
- ✅ Security sandbox flags (`--no-write`, `--restrict-read`, etc.)
- ✅ Pretty-print HTML output (`-pp`)
- ✅ Structured error messages with line/column/hints
- ✅ Version flag (`-V`)

### Missing Features (Compared to Other Languages)

| Feature | Python | Ruby | Node | Perl | pars | Priority |
|---------|--------|------|------|------|------|----------|
| Eval flag (`-e`) | `python -c` | `ruby -e` | `node -e` | `perl -e` | ❌ | **High** |
| Syntax check | `python -m py_compile` | `ruby -c` | `node --check` | `perl -c` | ❌ | Medium |
| Help/docs | `python -h topic` | `ri` | ❌ | `perldoc` | ❌ | Medium |
| Debugger | `pdb` | `debug` | `--inspect` | `perl -d` | ❌ | Low (defer) |
| Profiler | `cProfile` | `ruby-prof` | `--prof` | `Devel::NYTProf` | ❌ | Low (defer) |

### 3.1. `pars -e "code"` — Inline Evaluation

**Priority: High**

Common use case for testing and one-liners:

```bash
pars -e '"Hello, World!"'
pars -e '@now.format("short")'
pars -e '[1,2,3].map(fn(x) { x * 2 })'
```

**Implementation:** Add `-e` flag to `cmd/pars/main.go`, treat argument as source code instead of filename.

**Estimated effort:** 1-2 hours

### 3.2. `pars --check` — Syntax Check

**Analysis:** We already have pretty-print (`-pp`) which requires parsing. A `--check` flag would:
- Parse the file
- Report any syntax errors
- Exit without evaluation

**Question:** Is this needed given `-pp`?

**Answer:** Yes, for different use cases:
- `-pp` is for output formatting
- `--check` is for CI pipelines, pre-commit hooks, editor integration
- `--check` should have exit code semantics (0 = OK, 1 = errors)

**Implementation:** Add `--check` flag, parse program, exit with appropriate code.

**Estimated effort:** 1 hour

### 3.3. Unified Help System — CLI and REPL

**Priority: High**

#### The Problem with Current `describe()`

The current `describe()` builtin conflates two different use cases:

1. **Type/topic help** — "What methods does a string have?" "What's in @std/math?"
2. **Object introspection** — "What is this specific value?" "What are myVar's properties?"

These are fundamentally different:
- Type help is **static documentation** — the answer is the same regardless of runtime state
- Object introspection is **runtime reflection** — examining a specific value's actual state

Currently, `describe(string)` in the REPL doesn't work as expected because `string` is parsed as an identifier, not a type name. Users must write `describe("hello")` which describes that specific string instance.

#### Proposed Design: Separate Help from Introspection

**Help system (1.0 Alpha scope):**
- Static documentation about types, modules, builtins, operators
- Available via CLI (`pars describe <topic>`) and REPL (`:describe <topic>`)
- Same syntax, identical output in both contexts
- Topic-based, not expression-based

**Object introspection (future, not 1.0):**
- Runtime reflection on actual values
- `inspect(expr)` or similar — evaluates expression, returns structured data
- Useful for debugging, but not essential for Alpha

#### Unified Help Interface

**CLI:**

```bash
$ pars describe string
$ pars describe @std/math
$ pars describe builtins
$ pars describe operators
```

**REPL:**

```
>> :describe string
>> :describe @std/math
>> :describe builtins
>> :describe operators
```

**Output (identical in both):**

```
$ pars describe string
Type: string

Properties: (none)

Methods:
  .toUpper()           - Convert to uppercase
  .toLower()           - Convert to lowercase
  .trim()              - Remove leading/trailing whitespace
  .split(delim)        - Split by delimiter into array
  ...

$ pars describe @std/math
Module: @std/math

Exports:
  pi          - Mathematical constant π (3.14159...)
  e           - Mathematical constant e (2.71828...)
  abs(x)      - Absolute value
  sqrt(x)     - Square root
  ...

$ pars describe builtins
Builtin Functions by Category:

  file        - JSON, YAML, CSV, text, lines, bytes, file, dir, ...
  time        - now, date, datetime, duration
  conversion  - toInt, toFloat, toString, toArray, toDict
  output      - print, println, printf, log, logLine
  ...

$ pars describe operators
Operators:

  Arithmetic: +, -, *, /, %, **
  Comparison: ==, !=, <, >, <=, >=
  Logical:    &&, ||, !
  String:     ++ (concatenation)
  File I/O:   <== (read), ==> (write), =/=> (fetch)
  Database:   <=?=> (query one), <=??=> (query many), <=!=> (execute)
  ...
```

#### Topics to Support

| Topic | Description | Example |
|-------|-------------|---------|
| Type names | Methods and properties for a type | `string`, `array`, `datetime`, `money` |
| Modules | Exports from stdlib/basil modules | `@std/math`, `@std/table`, `@basil/http` |
| `builtins` | All builtin functions by category | Lists JSON, CSV, toInt, etc. |
| `operators` | All operators with descriptions | Arithmetic, comparison, I/O, etc. |
| Builtin name | Specific builtin function | `JSON`, `datetime`, `regex` |

#### Implementation

1. **Core help engine** — New package or file (e.g., `pkg/parsley/help/help.go`)
   - `DescribeTopic(topic string) string` — returns formatted help text
   - Reads from `TypeMethods`, `TypeProperties`, `BuiltinMetadata`, module registries
   - Single source of truth for both CLI and REPL

2. **CLI integration** — `cmd/pars/main.go`
   - Add `describe` subcommand (like existing `fmt` subcommand)
   - `pars describe <topic>` calls `help.DescribeTopic(topic)`

3. **REPL integration** — `pkg/parsley/repl/repl.go`
   - Add `:describe <topic>` command in `handleReplCommand()`
   - Calls same `help.DescribeTopic(topic)`

4. **Deprecate/repurpose `describe()` builtin**
   - Option A: Remove `describe()` builtin entirely (breaking change)
   - Option B: Keep `describe(value)` for object introspection, document as "advanced"
   - Option C: Have `describe()` print a hint directing users to `:describe` or `pars describe`
   - **Recommendation:** Option C for 1.0 Alpha, evaluate A or B for 1.0 final

#### What This Replaces

The current `describe()` builtin in `introspect.go` would be superseded by the help system for type/topic queries. The `TypeMethods` and `TypeProperties` maps would still be used, but accessed through the new help engine rather than the `describe()` builtin.

Object introspection (`describe(@now)`, `describe(myVar)`) is a separate feature that could return structured data (a dictionary) for programmatic use. This is **out of scope for 1.0 Alpha** but could be added later as `inspect()` or similar.

**Estimated effort:** 6-8 hours (slightly more due to unified architecture)

---

## 4. Backlog Items for 1.0 Alpha

### High Priority

| ID | Item | Impact | Effort |
|----|------|--------|--------|
| #52 | Commutative duration multiplication | `3 * @1d` fails but `@1d * 3` works — surprising | 30 min |
| #58 | Local module imports shouldn't require `-x` | Confusing security model | 2-3 hours |

### Medium Priority (Nice to Have)

| ID | Item | Impact | Effort |
|----|------|--------|--------|
| #34 | Error code documentation/help system | Polish, discoverability | 4-6 hours |
| #55 | Deprecate `format(arr)` builtin | API cleanup | 1 hour |

### Defer to Post-Alpha

| ID | Item | Reason |
|----|------|--------|
| #35 | Full CLDR compact number formatting | Library limitation, K/M/B is acceptable |
| #51 | Function methods | Nice to have, not blocking |
| #65 | toBox() color option | Terminal complexity |

---

## 5. IDE Integrations and Syntax Highlighters

### Current Status: NEEDS UPDATE

Two syntax highlighters exist but are out of date:

| Asset | Location | Status |
|-------|----------|--------|
| VS Code Extension | `.vscode-extension/` | Out of date |
| highlight.js | `contrib/highlightjs/` | Out of date |

#### Missing Language Features in Grammars

Based on lexer analysis, these features are **not highlighted** in current grammars:

| Feature | Syntax | Added When |
|---------|--------|------------|
| Schema declarations | `@schema Name { fields }` | Recent |
| Table literals | `@table [...]`, `@table(Schema) [...]` | Recent |
| Query DSL | `@query(...)`, `@insert(...)`, `@update(...)`, `@delete(...)` | Recent |
| Transactions | `@transaction { }` | Recent |
| Computed exports | `export computed name = expr` | Recent |
| Search literal | `@SEARCH` | Recent |
| Environment/args | `@env`, `@args`, `@params` | Older |

#### What Should Be Updated for 1.0

1. **VS Code Extension** (`.vscode-extension/syntaxes/parsley.tmLanguage.json`)
   - Add `@schema`, `@table`, `@query`, `@insert`, `@update`, `@delete`, `@transaction`
   - Add `computed` keyword after `export`
   - Add `@env`, `@args`, `@params`, `@SEARCH`
   - Update version number in `package.json`

2. **highlight.js** (`contrib/highlightjs/parsley.js`)
   - Same additions as VS Code
   - Test with demo.html

**Estimated effort:** 2-4 hours for both grammars

### What Else Should We Consider for 1.0?

#### Tree-sitter Grammar

| Consideration | Analysis |
|---------------|----------|
| **What is it?** | Parser generator used by Neovim, Helix, Zed, Nova, GitHub code navigation |
| **Effort** | 2-4 days for basic grammar, 1-2 weeks for full coverage |
| **1.0 Alpha?** | **Yes** — essential for modern editor support |
| **1.0 Final?** | Yes |
| **Priority** | **High** — blocking for 1.0 Alpha |

**Rationale for 1.0 Alpha:**

Tree-sitter has become the standard for syntax highlighting in modern editors:

- **Zed** — Fast, modern editor built entirely on tree-sitter (no TextMate fallback)
- **Nova** — macOS-native editor uses tree-sitter for syntax highlighting
- **Neovim** — Default syntax highlighting engine since 0.5
- **Helix** — Terminal editor, tree-sitter only
- **GitHub** — Code navigation, syntax highlighting in repos and PRs

Without tree-sitter, Parsley code appears as plain text in Zed, Nova, Helix, and GitHub. This is a poor first impression for developers evaluating the language.

**Note:** Tree-sitter grammar can be submitted to the official [tree-sitter grammars registry](https://github.com/tree-sitter/tree-sitter/wiki/List-of-parsers) and to GitHub's [linguist](https://github.com/github/linguist) for automatic `.pars` file recognition.

#### Language Server Protocol (LSP)

| Consideration | Analysis |
|---------------|----------|
| **What is it?** | Protocol for IDE features: autocomplete, go-to-definition, hover docs, diagnostics |
| **Effort** | 2-4 weeks for basic server (diagnostics, hover), 1-2 months for full features |
| **1.0 Alpha?** | **No** — significant undertaking |
| **1.0 Final?** | **Maybe** — depends on adoption/demand |
| **Priority** | Low for Alpha, revisit based on user feedback |

**Minimum viable LSP would provide:**
- Real-time syntax error highlighting
- Hover documentation (leveraging `describe()` infrastructure)
- Basic autocomplete for keywords and builtins

**Full LSP would add:**
- Go-to-definition for functions and imports
- Find references
- Rename symbol
- Code actions (quick fixes)

#### Other Editors

| Editor | Mechanism | Priority |
|--------|-----------|----------|
| Zed | Tree-sitter (required) | **High** — covered by tree-sitter grammar |
| Nova | Tree-sitter | **High** — covered by tree-sitter grammar |
| Neovim | Tree-sitter | **High** — covered by tree-sitter grammar |
| Helix | Tree-sitter (required) | **High** — covered by tree-sitter grammar |
| Sublime Text | `.sublime-syntax` (YAML) | Low — can convert from tmLanguage |
| JetBrains (IntelliJ, WebStorm) | TextMate or custom plugin | Low — TextMate works |
| Vim (without tree-sitter) | `.vim` syntax file | Low — small user base for new languages |
| Emacs | `parsley-mode.el` | Low — small user base |

**Recommendation:** Tree-sitter grammar covers Zed, Nova, Neovim, Helix, and GitHub. Defer Sublime/JetBrains/Vim/Emacs to post-1.0 based on user requests.

### Recommendations for 1.0 Alpha

| Task | Priority | Effort | Blocking? |
|------|----------|--------|-----------|
| Update VS Code grammar | High | 2-3 hours | Yes |
| Update highlight.js grammar | High | 1-2 hours | Yes |
| Tree-sitter grammar | High | 2-4 days | **Yes** |
| Language Server | Low | 2-4 weeks | No (post-1.0) |

### AI Tooling Considerations

For AI coding assistants to work well with Parsley:

1. **Syntax highlighting in AI context** — highlight.js update enables AI chat UIs to render Parsley code properly
2. **Tree-sitter for AI code analysis** — Some AI tools use tree-sitter for code understanding; defer to post-Alpha
3. **LSP for AI integration** — Copilot and similar tools can leverage LSP for better suggestions; defer to post-1.0

**Note:** The help system (`pars describe`, `:describe`) is more immediately valuable for AI assistance than LSP, since AI agents can invoke CLI commands to learn the language.

---

## 6. Summary of Recommended Actions

### Pre-1.0 Alpha (Blocking)

| Task | Priority | Effort | Creates Spec? |
|------|----------|--------|---------------|
| Add PostgreSQL driver | High | 2 hours | FEAT-XXX |
| Add MySQL driver | High | 2 hours | (same spec) |
| Add `describe()` validation tests | High | 4-6 hours | FEAT-XXX |
| Refactor methods.go to declarative registry | High | 8-12 hours | (same spec) |
| Add `pars -e` flag | High | 1-2 hours | FEAT-XXX |
| Update VS Code grammar | High | 2-3 hours | FEAT-XXX |
| Update highlight.js grammar | High | 1-2 hours | (same spec) |
| Create Tree-sitter grammar | High | 2-4 days | (same spec) |

### Pre-1.0 Alpha (Recommended)

| Task | Priority | Effort | Creates Spec? |
|------|----------|--------|---------------|
| Add `pars describe <topic>` | Medium | 4-6 hours | FEAT-XXX |
| Add `pars --check` flag | Medium | 1 hour | (same spec as -e) |
| Fix #52 (duration multiplication) | Medium | 30 min | BUG-XXX |
| Fix #58 (import without -x) | Medium | 2-3 hours | BUG-XXX |

### Post-1.0 Alpha (Enhancement)

| Task | Priority | Notes |
|------|----------|-------|
| Language Server (LSP) | Low | Complex, revisit based on user demand |
| Debugger | Low | Complex, defer to 1.0 final |
| Profiler | Low | Complex, defer to 1.0 final |

---

## 7. Proposed Spec Structure

Based on this analysis, create the following specs:

1. **FEAT-XXX: Database Driver Support**
   - Add lib/pq and go-sql-driver/mysql
   - Integration tests
   - Documentation updates

2. **FEAT-XXX: Introspection & Help System**
   - Phase 1: Validation tests for TypeMethods/TypeProperties
   - Phase 2: Declarative method registry (refactor methods.go)
   - Phase 3: Unified help system (`pars describe`, `:describe`)

3. **FEAT-XXX: CLI Enhancements**
   - `pars -e "code"` inline evaluation
   - `pars --check` syntax validation
   - Exit code semantics

4. **BUG-XXX: Duration Multiplication Not Commutative**
   - `3 * @1d` should equal `@1d * 3`

5. **BUG-XXX: Local Imports Require `-x` Flag**
   - Parsley module imports should work without execute permission

6. **FEAT-XXX: Syntax Highlighter Updates**
   - Update VS Code extension grammar
   - Update highlight.js grammar
   - Create Tree-sitter grammar (enables Zed, Nova, Neovim, Helix, GitHub)
   - Add missing literals: @schema, @table, @query, @insert, @update, @delete, @transaction
   - Add computed exports keyword
   - Submit to tree-sitter registry and GitHub linguist

---

## Appendix A: Current `describe()` Implementation

For reference, the current implementation in `introspect.go`:

```go
func builtinDescribe(args ...Object) Object {
    // 1. Get type name
    typeName, subType := getObjectTypeName(obj, nil)
    
    // 2. Look up properties (manually maintained map)
    propertyInfos, hasProps := TypeProperties[methodKey]
    
    // 3. Look up methods (manually maintained map)
    methodInfos, ok := TypeMethods[methodKey]
    
    // 4. Format as string
    // ...
}
```

The `TypeMethods` map (lines 128-340) contains ~25 type entries with ~200 total method definitions. The `TypeProperties` map (lines 40-127) contains ~15 type entries with ~60 total property definitions.

All of these are hand-coded and must be manually synchronized with `methods.go` and property accessor implementations.

---

## Appendix B: Missing Syntax Highlighting Features

### Literals Not in Current Grammars

From `pkg/parsley/lexer/lexer.go`:

```
SCHEMA_LITERAL    // @schema
TABLE_LITERAL     // @table
QUERY_LITERAL     // @query
INSERT_LITERAL    // @insert
UPDATE_LITERAL    // @update
DELETE_LITERAL    // @delete
TRANSACTION_LIT   // @transaction
SEARCH_LITERAL    // @SEARCH
ENV_LITERAL       // @env
ARGS_LITERAL      // @args
PARAMS_LITERAL    // @params
```

### Keywords Not in Current Grammars

- `computed` (in context of `export computed`)

### Query DSL Operators Not in Current Grammars

- `|<` (write operator in @insert/@update)
- `|>` (read projection)
- `?->` (single row terminal)
- `??->` (multi row terminal)
- `.->` (count terminal)
- `<-` (correlated subquery source)