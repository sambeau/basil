---
id: FEAT-023
title: "Structured Error Objects"
status: complete
priority: high
created: 2025-12-04
updated: 2025-12-09
author: "@ai"
---

# FEAT-023: Structured Error Objects  

## Summary

Replace string-based error messages with structured error objects that include message, hints, line/column info, error codes, and template data. This enables richer error display, localization, and programmatic error handling.

## Motivation

Currently:
- **Parser errors** are `[]string` with line/column embedded in the message text
- **Runtime errors** are `*evaluator.Error` with `Message`, `Line`, `Column` fields
- Basil has to **regex-parse** error strings to extract line/column info
- No way for consumers to customize error presentation
- No support for localization
- Hints are embedded in message strings with `\n`

## Design Principles

### 1. Consistent Error Naming
- Use **"Parser error"** for syntax problems
- Use **"Runtime error"** for everything during execution (type errors, I/O, security, etc.)
- Retire "Woops!", "ERROR:", and other inconsistent prefixes

### 2. Always Show Location
- Line and column numbers should be shown wherever possible
- In REPL: show line/column
- In files: show filename (relative to handler root in Basil, as given in Parsley)

### 3. Show Correct Usage
- Wherever possible, show an example of correct usage
- Multiple alternatives should be shown as separate hints

### 4. Consistent Casing
- Use lowercase type names: `string`, `array`, `function` (not `STRING`, `ARRAY`)
- Use Parsley terminology: `fn` not `func`

### 5. File Path Formatting
For errors from script files (not REPL), use multi-line format for readability:
```
Parser error:
  in: handlers/api/users.pars
  at: line 15, column 8
  `for 1..10 string` is ambiguous without ()
  Use: for (1..10) fn
   or: for x in 1..10 { ... }
```

## Proposed Design

### Error Struct

```go
// ErrorClass categorizes errors for filtering and templating
type ErrorClass string

const (
    ClassParse     ErrorClass = "parse"      // Parser errors
    ClassType      ErrorClass = "type"       // Type mismatches
    ClassArity     ErrorClass = "arity"      // Wrong argument count
    ClassUndefined ErrorClass = "undefined"  // Not found/defined
    ClassIO        ErrorClass = "io"         // File operations
    ClassDatabase  ErrorClass = "database"   // DB operations
    ClassNetwork   ErrorClass = "network"    // HTTP, SSH, SFTP
    ClassSecurity  ErrorClass = "security"   // Access denied
    ClassIndex     ErrorClass = "index"      // Out of bounds
    ClassFormat    ErrorClass = "format"     // Invalid format/parse
    ClassOperator  ErrorClass = "operator"   // Invalid operations
    ClassState     ErrorClass = "state"      // Invalid state
    ClassImport    ErrorClass = "import"     // Module loading
)

// ParsleyError represents any error from parsing or evaluation
type ParsleyError struct {
    Class   ErrorClass     `json:"class"`             // Error category for filtering/templating
    Code    string         `json:"code,omitempty"`    // Error code (e.g., "E001", "FOR_EXPECTS_FUNCTION")
    Message string         `json:"message"`           // Human-readable message (always present)
    Hints   []string       `json:"hints,omitempty"`   // Suggestions for fixing the error
    Line    int            `json:"line"`              // 1-based line number (0 if unknown)
    Column  int            `json:"column"`            // 1-based column number (0 if unknown)
    File    string         `json:"file,omitempty"`    // File path (if known)
    Data    map[string]any `json:"data,omitempty"`    // Template variables for custom rendering
}
```

### Error Classes and Data Fields

Each error class has specific `Data` fields that consumers can use for custom templating.
Templates use Go `text/template` syntax:

| Class | Key Data Fields | Template Example |
|-------|-----------------|------------------|
| `parse` | `Expected`, `Got`, `Token` | `expected {{.Expected}}, got '{{.Got}}'` |
| `type` | `Expected`, `Got`, `Function` | `{{.Function}} expected {{.Expected}}, got {{.Got}}` |
| `arity` | `Function`, `Got`, `Want` | `{{.Function}} takes {{.Want}} argument(s), got {{.Got}}` |
| `undefined` | `Name`, `Type`, `Suggestion` | `{{.Type}} not found: {{.Name}}` |
| `io` | `Path`, `Operation`, `GoError` | `failed to {{.Operation}} '{{.Path}}': {{.GoError}}` |
| `database` | `Driver`, `Operation`, `GoError` | `{{.Driver}} {{.Operation}} failed: {{.GoError}}` |
| `network` | `URL`, `Operation`, `StatusCode`, `GoError` | `{{.Operation}} to {{.URL}} failed: {{.GoError}}` |
| `security` | `Operation`, `Path`, `Flag` | `security: {{.Operation}} access denied` |
| `index` | `Index`, `Length` | `index {{.Index}} out of range (length {{.Length}})` |
| `format` | `Input`, `Format`, `GoError` | `invalid {{.Format}}: {{.GoError}}` |
| `operator` | `Operator`, `LeftType`, `RightType` | `unknown operator: {{.LeftType}} {{.Operator}} {{.RightType}}` |
| `state` | `Resource`, `ExpectedState`, `ActualState` | `{{.Resource}} is {{.ActualState}}, expected {{.ExpectedState}}` |
| `import` | `ModulePath`, `NestedError` | `in module {{.ModulePath}}: {{.NestedError}}` |

### Example Errors

**Parser error:**
```json
{
  "code": "UNEXPECTED_TOKEN",
  "message": "expected '{', got 'x'",
  "hints": [
    "for (var in array) requires a { } block body",
    "Use: for x in array { ... }"
  ],
  "line": 1,
  "column": 17,
  "data": {
    "expected": "{",
    "got": "x"
  }
}
```

**Runtime error:**
```json
{
  "code": "TYPE_MISMATCH",
  "message": "for expects a function or builtin, got STRING",
  "hints": [
    "Pass a function: for (array) fn(x) { x * 2 }",
    "Or use for-in syntax: for x in array { x * 2 }"
  ],
  "line": 1,
  "column": 1,
  "data": {
    "expected": ["FUNCTION", "BUILTIN"],
    "got": "STRING",
    "gotValue": "\"hello\""
  }
}
```

**I/O error:**
```json
{
  "code": "FILE_NOT_FOUND",
  "message": "failed to read file '/nonexistent.txt': no such file or directory",
  "line": 1,
  "column": 1,
  "file": "/path/to/script.pars",
  "data": {
    "path": "/nonexistent.txt",
    "operation": "read"
  }
}
```

### API Changes

#### Parser

```go
// New method (keep Errors() for backward compatibility)
func (p *Parser) StructuredErrors() []*ParsleyError

// Or replace entirely
func (p *Parser) Errors() []*ParsleyError  // Breaking change
```

#### Evaluator

```go
// Update existing Error type or create unified type
type Error struct {
    // ... same fields as ParsleyError
}

// Add methods
func (e *Error) ToJSON() ([]byte, error)
func (e *Error) String() string  // Returns Message for simple use
```

#### High-level API (pkg/parsley/parsley)

```go
type Result struct {
    Value  Object
    Errors []*ParsleyError  // Parse or runtime errors
}

func Eval(code string) *Result
func EvalFile(path string) *Result
```

### String Formatting

For backward compatibility and simple consumers, errors should have a `String()` method:

```go
func (e *ParsleyError) String() string {
    var sb strings.Builder
    if e.Line > 0 {
        sb.WriteString(fmt.Sprintf("line %d, column %d: ", e.Line, e.Column))
    }
    sb.WriteString(e.Message)
    for _, hint := range e.Hints {
        sb.WriteString("\n  ")
        sb.WriteString(hint)
    }
    return sb.String()
}
```

### JSON Output

Errors should be easily serializable:

```go
func (e *ParsleyError) ToJSON() ([]byte, error) {
    return json.Marshal(e)
}

func (e *ParsleyError) MarshalJSON() ([]byte, error) {
    // Custom marshaling if needed
}
```

## Use Cases

### 1. Basil Error Pages
Basil can render rich error pages without regex parsing:
```go
err := result.Errors[0]
devErr := DevError{
    Type:    err.Code,
    File:    err.File,
    Line:    err.Line,
    Column:  err.Column,
    Message: err.Message,
    Hints:   err.Hints,
}
```

### 2. IDE/Editor Integration
Error codes enable precise problem matching:
```json
{
  "source": "parsley",
  "code": "UNDEFINED_VARIABLE",
  "message": "identifier not found: foo",
  "range": {"line": 5, "column": 10}
}
```

### 3. Localization
Error codes map to translated templates:
```go
templates := map[string]string{
    "en": "expected {expected}, got {got}",
    "es": "se esperaba {expected}, se obtuvo {got}",
}
msg := fmt.Sprintf(templates[lang], err.Data["expected"], err.Data["got"])
```

### 4. Programmatic Error Handling
```go
for _, err := range result.Errors {
    switch err.Code {
    case "UNDEFINED_VARIABLE":
        // Suggest imports
    case "TYPE_MISMATCH":
        // Suggest type conversion
    }
}
```

## Implementation Plan

### Phase 1: Core Struct
1. Create `ParsleyError` type in shared location
2. Add `String()` and `ToJSON()` methods
3. Add `StructuredErrors()` to parser (keep `Errors()` for now)

### Phase 2: Parser Migration
1. Update parser to create `ParsleyError` objects internally
2. Store hints separately from messages
3. Add error codes to common errors

### Phase 3: Evaluator Migration
1. Unify `evaluator.Error` with `ParsleyError` or make compatible
2. Add `Data` field for template variables
3. Add error codes to common runtime errors

### Phase 4: Basil Integration
1. Update Basil to use structured errors
2. Remove regex parsing from `errors.go`
3. Enhance error pages with hints and data

### Phase 5: Documentation
1. Document all error codes
2. Document `Data` fields for each error type
3. Add examples for custom error rendering

## Error Code Convention

Error codes are for debugging and i18n, **not for display to users**.

Suggested format: `CLASS-nnnn` where CLASS matches the error class:
- `PARSE-0001` - Parser errors
- `TYPE-0001` - Type mismatch errors  
- `ARITY-0001` - Argument count errors
- `UNDEF-0001` - Undefined identifier errors
- `IO-0001` - File/I/O errors
- `DB-0001` - Database errors
- `NET-0001` - Network errors
- `SEC-0001` - Security errors
- `INDEX-0001` - Index/range errors
- `FMT-0001` - Format/parse errors
- `OP-0001` - Operator errors
- `STATE-0001` - State errors
- `IMPORT-0001` - Import/module errors

This format:
- Is readable by developers for debugging
- Works well with i18n spreadsheets (human translators need identifiers)
- Combines class context with unique numbering

## Error Catalog

All error definitions should live in a single catalog for maintainability and i18n.

**Template format:** Use Go `text/template` syntax (`{{.Field}}`) for templates. This provides:
- Well-tested, fast template engine
- Rich formatting options (conditionals, loops)
- Custom template functions for pluralization, formatting
- Familiar syntax for Go developers

```go
type ErrorDef struct {
    Class    ErrorClass
    Template string     // Message template with {{.placeholders}}
    Hints    []string   // Hint templates
}

var ErrorCatalog = map[string]ErrorDef{
    "PARSE-0001": {
        Class:    ClassParse,
        Template: "`{{.Source}}` is ambiguous without ()",
        Hints:    []string{
            "for ({{.Array}}) fn",
            "for x in {{.Array}} { ... }",
        },
    },
    "UNDEF-0001": {
        Class:    ClassUndefined,
        Template: "identifier not found: {{.Name}}",
        Hints:    []string{"Did you mean `{{.Suggestion}}`?"},  // Only if suggestion available
    },
    // ...
}
```

This enables:
- Single source of truth for all error messages
- Easy i18n/l10n (translators work with the catalog)
- Consistent formatting across all errors
- Easy updates without hunting through code

## Open Questions

1. ~~Should `Code` be required or optional?~~ Required for catalog lookup
2. ~~Should we use numeric codes (E001) or descriptive (UNDEFINED_VARIABLE)?~~ Use `CLASS-nnnn` format
3. How to handle nested errors (e.g., error in imported module)?
4. Should `Data` values be restricted to JSON-serializable types?
5. ~~Should we add a `Source` field to capture the actual source text that caused the error?~~ No - only useful for small snippets, and supporting large expressions would break error display
6. ~~For "Did you mean?" suggestions, should fuzzy matching apply to identifiers only, or also methods and keywords?~~ All three - since execution has stopped, we can afford slower but more helpful fuzzy matching

## Related

- Basil's `server/errors.go` - current error parsing and display
- `pkg/parsley/evaluator/evaluator.go` - current `Error` type
- `pkg/parsley/parser/parser.go` - current `[]string` errors

## Error Class Analysis

Analysis of ~466 error sites in the evaluator revealed these distinct error patterns:

### 1. Type Errors (`type`) - Most Common
```
"argument to `sin` not supported, got STRING"
"first argument to `file` must be a path or string, got ARRAY"
"cannot call null as a function"
```
**Data fields:** `expected` (type or array of types), `got` (actual type), `gotValue` (actual value), `function` (function name)

### 2. Arity Errors (`arity`)
```
"wrong number of arguments to `len`. got=2, want=1"
"comparison function must take exactly 2 parameters, got 0"
```
**Data fields:** `function`, `got` (count), `want` (count or range like "1-2")

### 3. Undefined/Not Found Errors (`undefined`)
```
"identifier not found: foo"
"undefined component: MyComponent"
"unknown method 'xyz' for STRING"
```
**Data fields:** `name`, `type` (variable/component/method/function), `on` (for methods - the type it was called on)

### 4. File/I/O Errors (`io`)
```
"failed to read file '/path': no such file"
"failed to resolve path 'x': error"
"SFTP write failed: connection closed"
```
**Data fields:** `path`, `operation` (read/write/delete/mkdir/rmdir), `goError` (underlying Go error message)

### 5. Database Errors (`database`)
```
"failed to open SQLite database: error"
"query failed: syntax error"
"failed to scan row: type mismatch"
```
**Data fields:** `driver` (sqlite/postgres/mysql), `operation` (connect/query/scan/ping), `goError`, `query` (if applicable)

### 6. Network Errors (`network`)
```
"failed to connect to SSH server: timeout"
"HTTP request failed: connection refused"
```
**Data fields:** `url` or `host`, `operation`, `statusCode` (for HTTP), `goError`

### 7. Security Errors (`security`)
```
"security: read access denied"
"security: execute access denied (use --allow-execute)"
```
**Data fields:** `operation` (read/write/execute), `path`, `requiredFlag` (CLI flag needed to allow)

### 8. Index/Range Errors (`index`)
```
"index out of range: 5"
"slice start index 10 is greater than end index 5"
"cannot pick from empty array"
```
**Data fields:** `index`, `length`, `start`, `end`

### 9. Format/Parse Errors (`format`)
```
"invalid regex pattern: unclosed group"
"invalid URL: missing scheme"
"invalid datetime: cannot parse"
"invalid JSON: unexpected EOF"
```
**Data fields:** `input` (the invalid input), `format` (regex/url/datetime/json), `goError`

### 10. Operator Errors (`operator`)
```
"unknown operator: STRING + ARRAY"
"division by zero"
```
**Data fields:** `operator`, `leftType`, `rightType`

### 11. State Errors (`state`)
```
"no transaction in progress"
"connection is already in a transaction"
"SFTP connection is not connected"
```
**Data fields:** `resource`, `expectedState`, `actualState`

### 12. Import/Module Errors (`import`)
```
"module not found: /path/to/module.pars"
"circular dependency detected when importing: a.pars -> b.pars -> a.pars"
"in module ./path.pars: line 5, column 3: error message"
```
**Data fields:** `modulePath`, `chain` (for circular deps), `nestedError` (for errors within modules)

## Error Message Improvements

This section shows current error messages and their improved forms.

### Ambiguous Syntax Errors

#### `for 1..10 "hello"` (missing parentheses)

**Current:**
```
Woops! We ran into some parser errors:
	for(array) func form requires parentheses (ambiguous without them)
  Use: for (array) fn  OR  for x in array { ... }
```

**Problems:**
1. No line number
2. Shows `for(array)` not `for array`
3. Uses 'func' which isn't Parsley terminology
4. "hello" is a string not a fn

**Improved:**
```
Parser error: line 1, column 1
  `for 1..10 string` is ambiguous without ()
  Use: for (1..10) fn
   or: for x in 1..10 { ... }
```

#### `for(1..10) "hello"` (string instead of function)

**Current:**
```
ERROR: for expects a function or builtin, got STRING
```

**Problems:**
1. Different format from parser errors
2. STRING is in CAPITALS (should be `string`)
3. No line number

**Improved:**
```
Runtime error: line 1, column 1
  for expects a function, got string
  Use: for (1..10) fn(x) { ... }
   or: for x in 1..10 { ... }
```

#### `for (1..10){"hello"}` (block looks like dictionary)

**Current:**
```
Woops! We ran into some parser errors:
	line 1, column 18: expected ':', got '}'
	line 1, column 18: unexpected '}'
```

**Problems:**
1. Confusing error - parser thinks `{...}` is a dictionary
2. No context about what the user was trying to do

**Improved:**
```
Parser error: line 1, column 12
  `for (1..10) {string}` is ambiguous
  Use: for _ in 1..10 { "hello" }
   or: for (1..10) fn(_) { "hello" }
```

#### `for (x in 1..10) x` (expression instead of block)

**Current:**
```
Woops! We ran into some parser errors:
	for (var in array) requires a { } block body, not an expression
  Use: for x in array { ... }  (parentheses are optional)
```

**Improved:**
```
Parser error: line 1, column 18
  `for (x in array)` requires a { } block body
  Use: for x in 1..10 { x }
```

### Undefined Identifier with Suggestions

#### `mip` (typo for `map`)

**Current:**
```
line 1, column 1: identifier not found: mip
```

**Improved:**
```
Runtime error: line 1, column 1
  identifier not found: mip
  Did you mean `map`?
```

### I/O Errors

| Example | Current Error |
|---------|---------------|
| `import("nonexistent.pars")` | `ERROR: module not found: /tmp/nonexistent.pars` |
| `SQLITE("/bad/path/db.sqlite")` | `failed to ping SQLite database: unable to open database file` |
| `file("/nonexistent.txt").remove()` | `failed to delete file '/nonexistent.txt': remove /nonexistent.txt: no such file or directory` |
| `import("x.pars")` (without -x) | `ERROR: security: execute access denied (use --allow-execute or -x)` |

**Improved format:**
```
Runtime error: line 1, column 1
  failed to read file '/nonexistent.txt'
  Error: no such file or directory
```

### Template Flexibility

Different display contexts may need different formatting:

**REPL/Terminal:**
```
Parser error: line 1, column 1
  `for 1..10 string` is ambiguous
  Use: for (1..10) fn
   or: for x in 1..10 { ... }
```

**Web page (table format):**

| | |
|:--|:--|
| **Parser error** | line 1, column 1 |
| **Problem** | `for 1..10 string` is ambiguous |
| **Use** | `for (1..10) fn` |
| **or** | `for x in 1..10 { ... }` |

The structured error object enables both formats from the same data.

## Implementation Status

### Completed (2025-12-04)

**Phase 1: Core Struct** ✅
- Created `pkg/parsley/errors/` package with `ParsleyError` type
- Added `ErrorClass` enum for categorizing errors (13 classes)
- Added `ErrorCatalog` for centralized error definitions
- Implemented `String()`, `PrettyString()`, `ToJSON()` methods
- Added template rendering with Go `text/template`

**Phase 2: Parser Integration** ✅
- Added `structuredErrors` field to Parser
- Added `StructuredErrors()` method to Parser
- Added `addError()`, `addStructuredError()`, `addErrorWithHints()` helper methods
- Maintained backward compatibility with string errors via `Errors()`

**Phase 3: Evaluator Integration** ✅
- Extended `evaluator.Error` struct with new fields: Class, Code, Hints, File, Data
- Added `ToParsleyError()` conversion method
- Added `newErrorWithClass()`, `newStructuredError()` helper functions
- Re-exported error classes as type aliases

**Phase 4: Basil Integration** ✅
- Updated `DevError` to include `Hints` field
- Added `FromParsleyError()` conversion function
- Added `handleStructuredError()` method to handler
- Updated error page rendering to display hints
- Error pages now show structured hints when available

**Phase 5: Fuzzy Matching ("Did you mean?")** ✅ (2025-01-20)
- Implemented Levenshtein distance algorithm for edit distance calculation
- Added `FindClosestMatch()` for single best suggestion
- Added `FindTopMatches()` for multiple suggestions
- Added `NewUndefinedIdentifier()` with automatic fuzzy matching
- Added `NewUndefinedMethod()` with automatic fuzzy matching
- Added `AllIdentifiers()` to Environment to get all available identifiers + builtins
- Updated `evalIdentifier()` to use fuzzy matching with "Did you mean?" hints
- Updated all method error handlers (string, array, integer, float, datetime, duration, path, url, regex, file, dir, request, response) to use fuzzy matching
- Added `ParsleyKeywords` list for keyword typo detection
- Dynamic threshold: 1 edit for 1-3 chars, 2 edits for 4-6 chars, 3 edits for 7+ chars

**Phase 6: Error Catalog Expansion** ✅ (2025-01-20)
- Expanded ErrorCatalog from ~20 to 70+ error codes:
  - PARSE (8 codes): token errors, regex/string/number literals, singleton tags
  - TYPE (10 codes): function calls, iteration, indexing, callbacks, comparisons
  - ARITY (4 codes): argument count errors, ranges
  - UNDEF (6 codes): identifiers, methods, properties, modules, exports
  - IO (8 codes): file operations, SFTP
  - DB (7 codes): query, connection, transaction errors
  - NET (4 codes): HTTP, SSH
  - SEC (5 codes): read, write, execute, network access
  - INDEX (5 codes): out of range, empty collection, negative index, key not found
  - FMT (7 codes): regex, URL, datetime, JSON, YAML, CSV
  - OP (4 codes): unknown operator, division by zero, comparison, negation
  - STATE (3 codes): connection state, file handles
  - IMPORT (3 codes): nested errors, circular dependencies

**Phase 7: Documentation** (Pending)
- Document all error codes in reference documentation
- Add examples for custom error rendering

**Phase 8: Error Migration** ✅ In Progress (2025-12-04)
- Created helper functions for common error patterns:
  - `newSecurityError()` for SEC-xxxx errors
  - `newDatabaseError()`, `newDatabaseErrorWithDriver()`, `newDatabaseStateError()` for DB-xxxx
  - `newTypeError()` for TYPE-xxxx errors
  - `newArityError()`, `newArityErrorRange()`, `newArityErrorMin()` for ARITY-xxxx
  - `newIOError()` for IO-xxxx errors
  - `newFormatError()`, `newFormatErrorWithPos()` for FMT-xxxx
  - `newUndefinedMethodError()` for UNDEF-0002
  - `newLocaleError()` for FMT-0008
  - `newStateError()` for STATE-xxxx
  - `newParseError()` for PARSE-0009, PARSE-0010, PARSE-0011
  - `newConversionError()` for TYPE-0015, TYPE-0016, TYPE-0017
  - `newNetworkError()` for NET-xxxx
  - `newSliceIndexTypeError()` for TYPE-0018
  - `newIndexError()` for INDEX-xxxx
  - `newUndefinedComponentError()` for UNDEF-0003
- Migrated 459 of 554 error sites (82.8%):
  - Security errors (SEC-0001 through SEC-0006)
  - Database errors (DB-0001 through DB-0008)
  - Type errors (TYPE-0001 through TYPE-0018)
  - Arity errors (ARITY-0001 through ARITY-0005)
  - I/O errors (IO-0002 through IO-0007)
  - Format errors (FMT-0002 through FMT-0009)
  - Method errors (UNDEF-0002, UNDEF-0003)
  - Parse errors (PARSE-0009 through PARSE-0011)
  - Network errors (NET-0003, NET-0005 through NET-0009)
  - Index errors (INDEX-0001, INDEX-0003)
- Added new error codes:
  - ARITY-0005 (min args)
  - FMT-0008 (locale), FMT-0009 (duration)
  - TYPE-0015/0016/0017 (conversion), TYPE-0018 (slice index type)
  - PARSE-0009/0010/0011 (template syntax)
  - NET-0005 through NET-0009 (SFTP/SSH)
  - SEC-0006 (SFTP auth)
- 95 error sites remaining for migration

### Phase 5: Complete Migration (2025-12-04)

Completed full migration of all `newError()` calls in the evaluator package:

**Files migrated:**
- `methods.go`: 105 → 0 newError() calls
- `stdlib_dev.go`: 11 → 0 newError() calls  
- `stdlib_table.go`: 43 → 0 newError() calls
- `evaluator.go`: 58 → 0 newErrorWithPos() calls

**Deprecated functions removed:**
- `newError()` - removed entirely
- `newErrorWithPos()` - removed entirely

**New error codes added:**
- Operator errors: OP-0005 through OP-0018
- Validation errors: VAL-0004 through VAL-0018
- Type errors: TYPE-0019 through TYPE-0022
- Arity error: ARITY-0006

**New helper functions:**
- `newOperatorError()` - for operator-related errors
- `newUndefinedError()` - for undefined property/method errors
- `newArityErrorExact()` - for exact argument count errors

**Total migration:**
- All evaluator package `newError()` calls now use structured errors
- ~217 additional error sites migrated in this phase
- Error codes provide programmatic error handling capability

### Remaining Work

1. ~~**Complete error migration** - 95 remaining sites including:~~
   ~~- Command handle errors (~12 sites)~~
   ~~- For loop errors (~7 sites)~~
   ~~- Import/module errors (~6 sites)~~
   ~~- File/write operator errors (~25 sites)~~
   ~~- Database operator errors (~15 sites)~~
   ~~- Miscellaneous errors (~30 sites)~~
2. **Update error messages** - Apply improved message formats from the spec
3. **Documentation** - Document all error codes in reference ✅ (see docs/parsley/error-codes.md)

## Notes

Space for additional notes and ideas during implementation.

### Related Features (Out of Scope)

- **`print` function**: Some improved error hints suggest `print string` as an alternative to expressions in blocks. Currently `log()` exists but outputs to dev log. A `print` function that contributes to result output would be a separate feature. See FEAT-024.

### Implementation Notes

- Fuzzy matching uses Levenshtein distance for "Did you mean?" suggestions
- Applied to identifiers, method names, and keywords
- Performance is acceptable since execution has stopped when errors occur
- Templates use Go `text/template` - can add custom functions for pluralization etc.