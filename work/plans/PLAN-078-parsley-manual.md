---
id: PLAN-078
feature: N/A (Documentation)
title: "Parsley Language Manual — Complete Documentation"
status: complete
created: 2026-01-30
---

# Implementation Plan: Parsley Language Manual

## Overview

Write the complete Parsley programming language manual. The manual currently has 12 pages covering builtins and two stdlib modules, but is missing all language fundamentals, key features, most of the standard library, and navigational pages. This plan covers writing ~30 new manual pages organized into 5 phases, plus quality improvements to existing pages.

**Gap analysis:** `work/reports/MANUAL-GAP-ANALYSIS.md`
**Templates:** `.github/templates/DOC_MAN_BUILTIN.md`, `.github/templates/DOC_MAN_STD.md`
**Existing reference:** `docs/parsley/reference.md` (monolithic, 4141 lines — primary source of truth alongside Go source)

## Conventions

- All manual pages live under `docs/parsley/manual/`
- Pages use YAML frontmatter (see existing pages for format)
- Code examples must be valid Parsley — test with `pars` CLI when possible
- Each example should show a `**Result:**` annotation
- Use "See Also" links at the bottom of every page to cross-reference related pages
- Follow the template structure: overview → literals/syntax → operators → properties → methods → examples → see also
- Builtin type pages go in `manual/builtins/`
- Stdlib pages go in `manual/stdlib/`
- Language fundamental pages go in `manual/fundamentals/`
- Domain feature pages go in `manual/features/`

## Prerequisites

- [x] Gap analysis completed (MANUAL-GAP-ANALYSIS.md)
- [x] Templates reviewed (DOC_MAN_BUILTIN.md, DOC_MAN_STD.md)
- [x] Existing pages reviewed for quality baseline
- [x] Directory structure created (`fundamentals/`, `features/`)

## Phase 1: Language Fundamentals

These pages are the most critical — they're what someone needs to read first to learn Parsley. Without them, users must read the 4141-line `reference.md` monolith. Write in the order listed; each page builds on the previous.

---

### Task 1.1: Create directory structure
**Estimated effort**: Tiny

Create the new subdirectories:
- `docs/parsley/manual/fundamentals/`
- `docs/parsley/manual/features/`

---

### Task 1.2: Comments (`fundamentals/comments.md`)
**Estimated effort**: Small (half a page)
**Source**: `lexer.go` (COMMENT token), `reference.md` §9

Scope:
- Single-line comments (`//`)
- Inline comments
- No multi-line comments (explicit note — common gotcha from `/* */` languages)
- Comments in different contexts (top-level, in expressions, in tags)

---

### Task 1.3: Booleans & Null (`builtins/booleans.md`)
**Estimated effort**: Small
**Source**: `evaluator.go` L165-184 (Boolean, Null types), `eval_infix.go` (evalBangOperatorExpression), `reference.md` §1.3

Scope:
- `true`, `false`, `null` literals
- Truthiness rules — what is falsy: `false`, `null`, `0`, `0.0`, `""`
- What is truthy: everything else (including `[]`, `{}`)
- Logical operators `!`, `not`, `&`/`and`, `|`/`or`
- Null coalescing `??`
- Boolean methods (`.toString()`)
- Null methods (`.toString()`)
- Common patterns: `if (value) { ... }`, `value ?? default`

---

### Task 1.4: Strings (`builtins/strings.md`)
**Estimated effort**: Large (27 methods + 3 string types)
**Source**: `methods.go` L117-436 (evalStringMethod), `evaluator.go` L5113-5387 (template/raw template eval), `reference.md` §1.2, §5.1

Scope:
- Three string types:
  - Double-quoted (`"..."`) — standard strings with `\n`, `\t` escapes
  - Template strings (`` `...` ``) — interpolation with `{expr}`
  - Raw strings (`'...'`) — interpolation with `@{expr}`, no escape processing
- Escape sequences
- String operators: `+` (concat), `*` (repeat), `in` (substring), comparison
- Indexing and slicing (`str[0]`, `str[1:3]`, negative indices)
- All 27 string methods (alphabetically):
  `capitalize`, `chars`, `contains`, `endsWith`, `highlight`, `humanize`, `indent`, `isEmpty`, `join` (on split result), `length`, `lines`, `lower`/`toLower`, `match`, `outdent`, `pad`/`padStart`/`padEnd`, `paragraphs`, `render`, `repeat`, `replace`, `reverse`, `slug`, `split`, `startsWith`, `strip`/`stripTags`, `toBox`, `toCSV`, `toJSON`, `toNumber`, `trim`, `trimStart`, `trimEnd`, `upper`/`toUpper`, `urlEncode`/`urlDecode`, `words`
- Operator table for strings

---

### Task 1.5: Variables & Binding (`fundamentals/variables.md`)
**Estimated effort**: Medium
**Source**: `parser.go` L493-663 (parseLetStatement), L667-784 (parseAssignmentStatement, parseDictDestructuringAssignment), `evaluator.go` L641-905 (Environment), `reference.md` §4.1-4.2

Scope:
- `let` binding (immutable by convention, scoped)
- Bare assignment (reassignment of existing binding)
- `let` vs bare assignment — when to use which
- Scope rules (block scoping, closures capture by reference)
- Array destructuring: `let [a, b, c] = [1, 2, 3]`
- Dictionary destructuring: `let {name, age} = person`
- Discard with `_`: `let [_, second] = pair`
- Multi-assignment from expressions
- Protected bindings (builtins can't be overwritten)

---

### Task 1.6: Operators (`fundamentals/operators.md`)
**Estimated effort**: Large
**Source**: `eval_operators.go`, `eval_infix.go`, `eval_collections.go`, `lexer.go` L60-119 (operator tokens), `reference.md` §2

Scope:
- Arithmetic: `+`, `-`, `*`, `/`, `%` (with type-specific notes)
- Comparison: `==`, `!=`, `<`, `>`, `<=`, `>=`
- Logical: `&`/`and`, `|`/`or`, `!`/`not`
- Set operations on arrays/dicts: `&` (intersection), `|` (union), `-` (difference)
- Membership: `in`, `not in` (arrays, dicts, strings)
- Schema checking: `is`, `is not`
- Pattern matching: `~` (match), `!~` (not match)
- Range: `..` (inclusive integer range)
- Concatenation: `++` (array/dict merge)
- Null coalescing: `??`
- Optional access: `[?n]`, `.?key`
- Spread: `...` (in function calls, array/dict literals)
- String operations: `+` (concat), `*` (repeat)
- Array operations: `+` (concat), `*` (repeat), `/` (chunk)
- DateTime arithmetic: `+`, `-` with integers and durations
- Money arithmetic: `+`, `-`, `*`, `/` with safety rules
- Path/URL arithmetic: `+` (join)
- File I/O operators: `<==`, `<=/=`, `==>`, `==>>`
- Database operators: `<=?=>`, `<=??=>`, `<=!=>`
- Command execution: `<=#=>`
- Precedence table (lowest to highest)

---

### Task 1.7: Functions (`fundamentals/functions.md`)
**Estimated effort**: Medium
**Source**: `parser.go` L2224-2333 (parseFunctionLiteral, parseFunctionParameters), `eval_expressions.go` L17-176 (applyFunction, ApplyFunctionWithEnv), `reference.md` §1.6

Scope:
- `fn` syntax: `fn(x) { x * 2 }`, `fn(x, y) { x + y }`
- Named functions via `let`: `let double = fn(x) { x * 2 }`
- Default parameter values: `fn(x, y = 10) { x + y }`
- Rest parameters: `fn(...args) { args.length() }`
- Return values — implicit (last expression) and explicit (`return`)
- Closures — functions capture their environment
- `this` binding in dictionary methods
- Functions as values (first-class) — passing to `.map()`, `.filter()`, `.sortBy()`
- Immediately invoked: `fn() { 42 }()`
- Arity checking — Parsley validates argument counts

---

### Task 1.8: Control Flow (`fundamentals/control-flow.md`)
**Estimated effort**: Medium-Large
**Source**: `eval_control_flow.go` (entire file), `eval_infix.go` L784-797 (evalIfExpression), `parser.go` L2092-2444 (parseIfExpression, parseForExpression), `reference.md` §3

Scope:
- **if/else** — expression-based (returns a value)
  - Compact form (ternary style): `if (cond) value1 else value2`
  - Block form: `if (cond) { ... } else { ... }`
  - if-else-if chains
- **for** — expression-based (returns an array)
  - Map pattern: `for (x in items) { transform(x) }`
  - Filter pattern: `for (x in items) { if (cond) { x } }`
  - Map + filter combined
  - With index: `for (i, x in items) { ... }`
  - With range: `for (n in 1..10) { ... }`
  - Over dictionaries: `for (key, val in dict) { ... }`
  - Over strings (iterates characters)
  - Over tables (iterates rows)
- **Loop control**
  - `stop` — exit loop early (like `break`)
  - `skip` — skip iteration (like `continue`)
- **check** guard
  - Syntax: `check condition else fallback`
  - Early exit from blocks
  - Pattern: validation at top of function
- **try** expression
  - Wraps errors: `let result = try someCall()`
  - Returns `{result, error}` dictionary
  - Catchable vs non-catchable errors
- **fail** function
  - Creates user-level catchable errors: `fail("something went wrong")`

---

### Task 1.9: Error Handling (`fundamentals/errors.md`)
**Estimated effort**: Medium
**Source**: `eval_control_flow.go` (evalTryExpression), `evaluator.go` L224-254 (Error type), `eval_errors.go`, `reference.md` §10

Scope:
- Error model overview — errors are values with class, code, message, hints
- `try` expression in depth — success and error patterns
- `fail()` function — creating user errors
- Error result dictionaries: `{result: value, error: null}` or `{result: null, error: "message"}`
- Catchable error classes: Value, IO, Database, Network, Format, Index, Security
- Non-catchable errors: Parse, Type, Arity, Undefined, Operator, State, Import
- Error prevention patterns:
  - `check` guards for preconditions
  - Optional index access `[?n]` for safe array/dict access
  - Null coalescing `??` for defaults
  - `in` operator for membership testing before access
- Link to `docs/parsley/error-codes.md` for the full error code catalog

---

## Phase 2: Key Features

These cover Parsley's distinctive features — especially tags (HTML generation) and modules (code organization).

---

### Task 2.1: Modules (`fundamentals/modules.md`)
**Estimated effort**: Medium
**Source**: `eval_expressions.go` L189-426 (evalImportExpression, importModule, evalImport), `parser.go` L373-490 (parseExportStatement, parseComputedExportStatement), L1805-1865 (parseImportExpression), `reference.md` §4.4-4.5

Scope:
- `import` expression — returns a dictionary of exported values
- Relative imports: `import(@./utils.pars)`
- Standard library imports: `import @std/math`
- Destructuring imports: `let {floor, ceil} = import @std/math`
- Named imports: `let math = import @std/math`
- `export` statement — marking values for external use
- `export let` — combined declaration and export
- Computed exports: `computed export name = fn() { ... }`
- Module caching — modules are evaluated once and cached
- Module scope — each module has its own environment
- Circular import prevention

---

### Task 2.2: Tags (`fundamentals/tags.md`)
**Estimated effort**: Large
**Source**: `eval_tags.go` (entire file, 2586 lines), `parser.go` L1292-1766 (parseTagLiteral, parseTagPair, parseTagContents, parseTagAttributes), `reference.md` §8

Scope:
- Self-closing tags: `<br/>`, `<img src="photo.jpg"/>`
  - **Gotcha**: self-closing tags MUST use `/>` — `<br>` is invalid
- Pair tags: `<div>content</div>`
- Attributes:
  - String attributes: `<div class="container">`
  - Expression attributes: `<div class={dynamicClass}>`
  - Boolean attributes: `<input required={true}/>`
  - Spread attributes: `<div {...props}>`
  - Attributes don't need quotes around values (Parsley convention)
- Content:
  - Static text between tags
  - Variable interpolation as content
  - Parsley expressions as content (if, for, let blocks)
  - Method calls as content
- Void HTML elements (self-closing by default)
- Nested tags
- Components — using imported functions as custom tags
  - Tag call syntax: `<MyComponent prop={value}/>`
  - Tag pair syntax with children: `<Layout><Content/></Layout>`
- Special tags:
  - `<SQL>` — parameterized SQL queries
  - `<Cache>` — fragment caching
  - `<Part>` — partial/AJAX fragments
- Form binding:
  - `@record` attribute for form context
  - `@field` attribute for input binding
  - Autocomplete derivation
- `tag()` builtin for programmatic tag creation

---

### Task 2.3: File I/O (`features/file-io.md`)
**Estimated effort**: Medium
**Source**: `eval_file_io.go`, `evaluator.go` L2676-2780 (file handle builtins in getBuiltins), `reference.md` §6.12

Scope:
- Read operator `<==`: `let data <== fileHandle`
- Fetch operator `<=/=`: `let data <=/= urlHandle`
- Write operator `==>`: `data ==> fileHandle`
- Append operator `==>>`: `data ==>> fileHandle`
- File handle factories:
  - `file(@./data.json)` — auto-detect format from extension
  - `JSON(@./data.json)` — explicit JSON
  - `YAML(@./config.yaml)` — YAML
  - `CSV(@./data.csv)` — CSV (with `{header: true}` option)
  - `PLN(@./data.pln)` — Parsley Literal Notation
  - `SVG(@./icon.svg)` — SVG (strips XML prolog)
  - `MD(@./readme.md)` — Markdown (with frontmatter support)
  - `text(@./file.txt)` — plain text
  - `bytes(@./file.bin)` — raw bytes
  - `lines(@./file.txt)` — array of lines
- Directory operations: `dir(@./public)` — directory listing
- File listing: `fileList(@./public/*.jpg)` — glob patterns
- `asset()` builtin — convert paths to web URLs
- File methods (from `methods.go` evalFileMethod, evalDirMethod)

---

### Task 2.4: Regex (`builtins/regex.md`)
**Estimated effort**: Medium
**Source**: `eval_regex.go`, `methods.go` L2484-2637 (evalRegexMethod), `parser.go` L1047-1075 (parseRegexLiteral), `reference.md` §1.10, §5.9

Scope:
- Regex literals: `/pattern/flags`
- `regex()` builtin: `regex("pattern", "flags")`
- Flags: `i` (case-insensitive), `m` (multi-line), `s` (dotall), `g` (global — handled at operator level)
- Match operator `~`: `"hello" ~ /ell/` → truthy array of matches
- Not-match operator `!~`: `"hello" !~ /xyz/` → true
- Regex as dictionary: `{__type: "regex", pattern: "...", flags: "..."}`
- Regex properties: `pattern`, `flags`
- Regex methods: `test(str)`, `match(str)`, `matchAll(str)`, `replace(str, replacement)`, `split(str)`
- String methods that accept regex: `.match()`, `.replace()`, `.split()`
- Named capture groups
- Common patterns and recipes

---

### Task 2.5: Paths (`builtins/paths.md`)
**Estimated effort**: Medium
**Source**: `eval_paths.go`, `eval_computed_properties.go` L14-232, `methods.go` L2213-2338 (evalPathMethod), `evaluator.go` (evalPathLiteral, evalPathTemplateLiteral), `reference.md` §1.11, §5.7

Scope:
- Path literals: `@/usr/local/bin`, `@./config`, `@~/project`
- Interpolated paths: `@(./data/{filename}.json)`
- `path()` builtin: `path("./relative/path")`
- Path properties: `segments`, `name`, `ext`, `stem`, `dir`, `absolute`, `string`
- Path methods: `join()`, `parent()`, `resolve()`, `withExt()`, `withName()`, `exists()`, `isDir()`, `isFile()`
- Path arithmetic: `path + "subdir"` for joining
- Relative vs absolute paths
- `~/` means project root (not home directory — important gotcha)

---

### Task 2.6: URLs (`builtins/urls.md`)
**Estimated effort**: Medium
**Source**: `eval_urls.go`, `eval_computed_properties.go` L423-526, `methods.go` L2345-2477 (evalUrlMethod), `evaluator.go` (evalUrlLiteral, evalUrlTemplateLiteral), `reference.md` §1.12, §5.8

Scope:
- URL literals: `@https://example.com/api/v1`
- Interpolated URLs: `@(https://api.com/{version}/users/{id})`
- `url()` builtin: `url("https://example.com")`
- URL properties: `scheme`, `host`, `port`, `path`, `query`, `fragment`, `origin`, `string`
- URL methods: `withPath()`, `withQuery()`, `withFragment()`, `resolve()`, `withHost()`, `withScheme()`
- URL arithmetic: `url + "/path"` for joining
- URL as file handle source (for fetch operations)

---

### Task 2.7: Type System Overview (`fundamentals/types.md`)
**Estimated effort**: Medium
**Source**: `evaluator.go` L61-88 (ObjectType constants), `reference.md` Appendix A

Scope:
- Overview of all Parsley types
- Type hierarchy / relationships:
  - Primitives: Integer, Float, Boolean, String, Null
  - Collections: Array, Dictionary, Table
  - Structured: Record (Dictionary + Schema), Schema
  - Specialized: Money, DateTime, Duration, Path, URL, Regex
  - Callable: Function, Builtin
  - I/O: FileHandle, DirHandle, DBConnection, SFTPConnection
- Type coercion rules (when types auto-convert)
- Type checking: `is` operator for schema checks, truthiness for conditionals
- "Everything is an expression" philosophy
- Dictionary as the universal composite type

---

### Task 2.8: Data Model — Schemas, Records & Tables (`fundamentals/data-model.md`)
**Estimated effort**: Medium
**Source**: `evaluator.go` (Schema, Record, Table types), existing `builtins/schema.md`, existing `builtins/record.md`, existing `builtins/table.md`, `reference.md` §1.15, §5.11-5.12

This is a **conceptual overview** page — the detailed API reference already exists in the builtin pages. This page explains how the pieces fit together.

Scope:
- The data model at a glance: Schema → Record → Table pipeline
- Schema defines shape (fields, types, constraints, metadata)
- Record = Schema + Data + Errors (a validated dictionary)
- Table = ordered collection of rows, optionally typed by a schema
- Dictionary vs Record — when each is used and why
- The lifecycle: `Schema → Record(data) → .validate() → database / form`
- `is` operator for schema identity checks
- Table bindings (schema + database connection)
- How schemas drive form generation, validation, and database operations
- Cross-references to `builtins/schema.md`, `builtins/record.md`, `builtins/table.md` for full API details

---

## Phase 3: Database & I/O

---

### Task 3.1: Database (`features/database.md`)
**Estimated effort**: Large
**Source**: `eval_database.go`, `evaluator.go` L1229-1314 (evalConnectionLiteral, resolveDBLiteral), L1852-2362 (connectionBuiltins), `reference.md` (scattered)

Scope:
- Connection literals: `@sqlite`, `@postgres`, `@mysql`
- Connection with DSN: `let db = @sqlite "./mydb.sqlite"`
- Managed connections (from Basil config) vs inline connections
- Query operators:
  - `<=?=>` — query one row (returns dict or null)
  - `<=??=>` — query many rows (returns table)
  - `<=!=>` — execute mutation (returns `{affected, lastId}`)
- `<SQL>` tags for parameterized queries
- Transactions: `@transaction`
- Connection methods (from connectionBuiltins)
- Schema-driven table bindings
- Result types: dictionaries for rows, tables for result sets
- Error handling for database operations

---

### Task 3.2: Query DSL (`features/query-dsl.md`)
**Estimated effort**: Large
**Source**: `parser.go` L3561-4819 (parseQueryExpression through parseTransactionExpression), `stdlib_dsl_query.go`, `stdlib_dsl_schema.go`

Scope:
- `@query` expressions — declarative data queries
- `@insert` expressions — insert records
- `@update` expressions — update records
- `@delete` expressions — delete records
- Where conditions and operators
- Order, limit, offset modifiers
- Computed fields
- Relations and joins
- Subqueries
- Pipe operators (`|<`, `?->`, `??->`)
- Group by
- Integration with schemas and table bindings
- Generated SQL examples

---

### Task 3.3: Markdown & CSV (`features/data-formats.md`)
**Estimated effort**: Medium
**Source**: `eval_parsing.go` L445-593 (parseMarkdown, parseCSV), `eval_encoders.go` L138-222 (encodeCSV), `markdown_helpers.go`, `methods.go` (string methods: parseMarkdown, parseCSV, toCSV, toJSON), file handle factories in `eval_file_io.go` (MD, CSV)

Scope:
- **Markdown**:
  - `.parseMarkdown(opts?)` string method → `{html, md, raw}` dictionary
  - Options: `{ids: true}` for heading IDs
  - `MD()` file handle — read Markdown files with frontmatter support
  - Markdown AST via `parseMarkdownToAST()` (if exposed)
  - Integration with `@std/mdDoc` module (cross-reference)
- **CSV**:
  - `.parseCSV(hasHeader?)` string method → Table
  - `.toCSV(hasHeader?)` on arrays and tables → string
  - `CSV()` file handle — read/write CSV files with `{header: true}` option
  - Auto-type detection (integers, floats, booleans parsed from CSV values)
- **JSON** (brief, since methods exist on multiple types):
  - `.parseJSON()` string method → Parsley values
  - `.toJSON()` on strings, arrays, dictionaries, tables
  - `JSON()` file handle
- Common patterns: reading data files, transforming between formats

---

### Task 3.4: HTTP & Networking (`features/network.md`)
**Estimated effort**: Medium
**Source**: `eval_network_io.go`, `reference.md` §6.12

Scope:
- Fetch operator `<=/=` with URL handles
- HTTP methods via URL handle methods: GET (default), POST, PUT, DELETE
- Request configuration (headers, body, method)
- Response dictionaries: `{data, status, statusText, ok, url, headers, format}`
- URL-based file handle factories: `JSON(@https://api.com/data)`
- Error handling for network requests: `{data, error}` pattern
- SFTP connections and operations:
  - `@sftp` literal
  - Read/write via SFTP file handles
  - SFTP connection methods

---

### Task 3.5: Shell Commands (`features/commands.md`)
**Estimated effort**: Small-Medium
**Source**: `evaluator.go` L4000-4304 (createCommandHandle, executeCommand, applyCommandOptions, createResultDict)

Scope:
- `@shell` literal — create a command handle
- Execute operator `<=#=>` — run commands
- Command options (stdin, timeout, env vars)
- Result dictionaries: `{stdout, stderr, exitCode, ok}`
- Security model for command execution
- Error handling

---

### Task 3.6: Security Model (`features/security.md`)
**Estimated effort**: Small
**Source**: `evaluator.go` L598-607 (SecurityPolicy), `docs/parsley/security.md` (existing), `sql_security.go`, `command_security_test.go`

Scope:
- Overview of Parsley's security philosophy
- File path restrictions (RestrictRead, RestrictWrite, NoRead, NoWrite, AllowWrite)
- SQL injection prevention (parameterized queries)
- Command execution restrictions (AllowExecute, AllowExecuteAll)
- PLN safety (no code execution in deserialization)
- Integrate/link to existing `docs/parsley/security.md`

---

## Phase 4: Standard Library

Each stdlib page follows the `DOC_MAN_STD.md` template: overview → attributes → methods (alphabetical).

---

### Task 4.1: @std/math (`stdlib/math.md`)
**Estimated effort**: Large (many functions)
**Source**: `stdlib_math.go`

Scope:
- Constants: `PI`, `E`, `Inf`, `NaN`
- Rounding: `floor`, `ceil`, `round`, `trunc`
- Comparison: `abs`, `sign`, `clamp`, `min`, `max`
- Aggregation: `sum`, `avg`, `product`, `count`
- Statistics: `median`, `mode`, `stddev`, `variance`, `range`
- Random: `random`, `randomInt`, `seed`
- Powers & Logarithms: `sqrt`, `pow`, `exp`, `log`, `log10`
- Trigonometry: `sin`, `cos`, `tan`, `asin`, `acos`, `atan`, `atan2`
- Angular conversion: `degrees`, `radians`
- Geometry: `hypot`, `dist`, `lerp`, `map`

---

### Task 4.2: @std/valid (`stdlib/valid.md`)
**Estimated effort**: Medium
**Source**: `stdlib_valid.go`, `reference.md` §7.2

Scope: Type validators, string validators, number validators, format validators, collection validators

---

### Task 4.3: @std/id (`stdlib/id.md`)
**Estimated effort**: Small
**Source**: `stdlib_id.go`, `reference.md` §7.3

Scope: UUID and nanoid generation functions

---

### Task 4.4: @std/table (`stdlib/table.md`)
**Estimated effort**: Medium
**Source**: `stdlib_table.go`, `reference.md` §7.4

Scope: Table constructors, query methods, aggregation, access, mutation, export methods

---

### Task 4.5: @std/api (`stdlib/api.md`)
**Estimated effort**: Medium
**Source**: `stdlib_api.go`, `reference.md` §7.6

Scope: Auth wrappers, error helpers, redirect helpers

---

### Task 4.6: @std/mdDoc (`stdlib/mddoc.md`)
**Estimated effort**: Medium
**Source**: `stdlib_mddoc.go`, `reference.md` §7.7

Scope: Markdown document constructor, rendering methods, query methods, transform methods, AST access

---

### Task 4.7: @std/dev (`stdlib/dev.md`)
**Estimated effort**: Small
**Source**: `stdlib_dev.go`, `reference.md` §7.8

Scope: Development and debugging utilities

---

### Task 4.8: @std/session (`stdlib/session.md`)
**Estimated effort**: Medium
**Source**: `stdlib_session.go`

Scope:
- Session data: `get`, `set`, `delete`, `has`, `clear`, `all`
- Flash messages: `flash`, `getFlash`, `getAllFlash`, `hasFlash`
- Session regeneration
- Cookie session model

---

## Phase 5: Navigation, Tutorial & Polish

---

### Task 5.1: Manual Index (`index.md`)
**Estimated effort**: Medium
**Source**: All existing and newly created pages

Scope:
- Table of contents with logical chapter ordering
- Brief one-line description of each page
- Suggested reading order for beginners
- Quick-reference links by topic (e.g., "Working with data", "Building web pages", "Database access")

---

### Task 5.2: Getting Started Tutorial (`getting-started.md`)
**Estimated effort**: Large
**Source**: All fundamentals pages (write this LAST so it can reference them)

Scope:
- First Parsley program
- Variables and expressions
- Working with strings and templates
- Functions
- Building HTML with tags
- A simple web page handler
- Where to go next (links into the manual)

---

### Task 5.3: Cross-reference audit
**Estimated effort**: Medium

Go through all manual pages (existing + new) and add "See Also" sections linking to related pages. Specific connections to make:
- datetime ↔ duration (arithmetic)
- schema ↔ record ↔ table (data pipeline)
- dictionary → record (record extends dictionary)
- functions → control-flow (for loop callbacks)
- tags → modules (components are imported functions)
- file-io → paths, urls (handle sources)
- database → query-dsl, schema, table (query pipeline)
- errors → control-flow (try, check, fail)
- strings → regex (pattern matching methods)

---

### Task 5.4: Fix existing page issues
**Estimated effort**: Small

From the gap analysis:
- [ ] `array.md`: Fix typo "buty" → "but" in map section
- [ ] `array.md`: Fix natural sort example expected results (`1 apple` vs `9 apple`)
- [ ] `array.md`: Add static fallback results for live-rendered `⚡️@{repr(...)}` expressions
- [ ] `dictionary.md`: Add section on computed properties and `this` binding
- [ ] `schema.md`: Add cross-reference to table.md for TableBinding
- [ ] `record.md`: Expand or cross-reference form binding (`@record`, `@field`)
- [ ] All existing pages: Add "See Also" section (covered by Task 5.3)

---

## Writing Process for Each Page

For each manual page, follow this workflow:

1. **Read the source** — Open the Go files listed in the task. Read the implementation to understand exact behavior, edge cases, and error conditions.
2. **Check reference.md** — Read the corresponding section in `reference.md` for existing prose and examples.
3. **Check tests** — Look at test files (`*_test.go`) for edge cases and expected behavior.
4. **Draft the page** — Follow the template structure. Write clear examples with `**Result:**` annotations.
5. **Verify examples** — Run key examples through `pars` CLI to confirm they produce the stated results.
6. **Add cross-references** — Include "See Also" links to related pages.

---

## Validation Checklist

- [ ] All new pages have YAML frontmatter
- [ ] All code examples are valid Parsley
- [ ] All pages have "See Also" sections
- [ ] Directory structure matches proposed layout
- [ ] index.md links to all pages
- [ ] Existing page issues fixed
- [ ] Key examples verified with `pars` CLI

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-01-30 | Gap analysis | ✅ Complete | `work/reports/MANUAL-GAP-ANALYSIS.md` |
| 2026-01-30 | PLAN-078 created | ✅ Complete | This document |
| 2026-02-05 | Task 1.1: Directory structure | ✅ Complete | Created `fundamentals/` and `features/` dirs |
| 2026-02-05 | Task 1.2: Comments | ✅ Complete | `fundamentals/comments.md` — all examples verified with `pars` |
| 2026-02-05 | Task 1.3: Booleans & Null | ✅ Complete | `builtins/booleans.md` — all examples verified with `pars` |
| 2026-02-05 | Task 1.4: Strings | ✅ Complete | `builtins/strings.md` — all examples verified with `pars` |
| 2026-02-05 | Task 1.5: Variables & Binding | ✅ Complete | `fundamentals/variables.md` — all examples verified with `pars` |
| 2026-02-05 | Task 1.6: Operators | ✅ Complete | `fundamentals/operators.md` — all examples verified with `pars` |
| 2026-02-05 | Task 1.7: Functions | ✅ Complete | `fundamentals/functions.md` — all examples verified with `pars` |
| 2026-02-05 | Task 1.8: Control Flow | ✅ Complete | `fundamentals/control-flow.md` — all examples verified with `pars` |
| 2026-02-06 | Task 1.9: Error Handling | ✅ Complete | `fundamentals/errors.md` — all examples verified with `pars` |
| 2026-02-06 | Task 2.1: Modules | ✅ Complete | `fundamentals/modules.md` — all examples verified with `pars` |
| 2026-02-06 | Task 2.2: Tags | ✅ Complete | `fundamentals/tags.md` — all examples verified with `pars` |
| 2026-02-06 | Task 2.3: File I/O | ✅ Complete | `features/file-io.md` — covers operators, handles, read/write/append, error capture |
| 2026-02-06 | Task 2.4: Regex | ✅ Complete | `builtins/regex.md` — literals, operators, methods, patterns verified with `pars` |
| 2026-02-06 | Task 2.5: Paths | ✅ Complete | `builtins/paths.md` — literals, interpolation, properties, methods |
| 2026-02-06 | Task 2.6: URLs | ✅ Complete | `builtins/urls.md` — literals, interpolation, properties, methods |
| 2026-02-06 | Task 2.7: Type System Overview | ✅ Complete | `fundamentals/types.md` — all types, coercion rules, typeof, is |
| 2026-02-06 | Task 2.8: Data Model (Schemas/Records/Tables) | ✅ Complete | `fundamentals/data-model.md` — conceptual overview linking to existing API pages |
| 2026-02-06 | Task 3.1: Database | ✅ Complete | `features/database.md` — connections, query operators, SQL tag, transactions, table bindings, error codes |
| 2026-02-06 | Task 3.2: Query DSL | ✅ Complete | `features/query-dsl.md` — @query, @insert, @update, @delete, terminals, conditions, modifiers, group by, subqueries, CTEs, relations, batch, upsert, @transaction |
| 2026-02-06 | Task 3.3: Markdown & CSV | ✅ Complete | `features/data-formats.md` — parseMarkdown, parseCSV, toCSV, parseJSON, toJSON, file handles, auto-type detection |
| 2026-02-06 | Task 3.4: HTTP & Networking | ✅ Complete | `features/network.md` — fetch operator, URL handles, request/response shapes, HTTP methods, SFTP, error capture |
| 2026-02-06 | Task 3.5: Shell Commands | ✅ Complete | `features/commands.md` — @shell, execute operator, result dict, options, security |
| 2026-02-06 | Task 3.6: Security Model | ✅ Complete | `features/security.md` — security policy, file restrictions, SQL injection prevention, command execution, PLN safety |
| 2026-02-06 | Task 4.1: @std/math | ✅ Complete | `stdlib/math.md` — verified with `pars`; fixed doc: `round` takes 1 arg (no `decimals?`) |
| 2026-02-06 | Task 4.2: @std/valid | ✅ Complete | `stdlib/valid.md` — all examples verified with `pars` |
| 2026-02-06 | Task 4.3: @std/id | ✅ Complete | `stdlib/id.md` — all examples verified with `pars` |
| 2026-02-06 | Task 4.4: @std/table | ✅ Complete | `stdlib/table.md` — all examples verified with `pars` |
| 2026-02-06 | Task 4.5: @std/api | ✅ Complete | `stdlib/api.md` — source-verified (server-only, not testable via `pars`) |
| 2026-02-06 | Task 4.6: @std/mdDoc | ✅ Complete | `stdlib/mddoc.md` — all examples verified with `pars` |
| 2026-02-06 | Task 4.7: @std/dev | ✅ Complete | `stdlib/dev.md` — source-verified (server-only, no-ops in `pars`) |
| 2026-02-06 | Task 4.8: @std/session | ✅ Complete | `stdlib/session.md` — source-verified (server-only, not available in `pars`) |
| 2026-02-06 | Task 5.1: Manual Index | ✅ Complete | `index.md` — TOC, topic quick-reference, suggested reading order |
| 2026-02-06 | Task 5.2: Getting Started | ✅ Complete | `getting-started.md` — hands-on tutorial covering variables through components |
| 2026-02-06 | Task 5.3: Cross-reference audit | ✅ Complete | Fixed absolute links → relative; added See Also to 7 pre-existing pages (datetime, duration, numbers, money, array, record, table); added cross-refs per plan (datetime↔duration, strings→regex, file-io→paths/urls, tags↔modules, errors→@std/api, control-flow→arrays, etc.); fixed broken links in pln.md, schema.md, dictionary.md |
| 2026-02-06 | Task 5.4: Fix existing pages | ✅ Complete | array.md: fixed "buty" typo, fixed natural sort expected result, replaced 9 live-rendered `⚡️@{repr(...)}` with static fallbacks; dictionary.md: added `this` binding section, removed dead @stdlib json links; schema.md: fixed See Also links to correct manual pages; table.md (builtins): removed stray `]`; math.md: fixed `round` docs (1 arg, no `decimals?`) |

## Deferred Items

Items to add to `work/BACKLOG.md` if they arise during writing:
- Cookbook / recipes page (common patterns across features)
- Migration guide from other template languages
- Performance tips page
- Parsley grammar specification (formal BNF/EBNF)
- Interactive examples (requires tooling work)