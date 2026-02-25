---
id: PLAN-104
feature: FEAT-125
title: "Implementation Plan for Error Message Improvements"
status: complete
created: 2025-01-20
completed: 2025-01-20
---

# Implementation Plan: FEAT-125 Error Message Improvements

## Overview

Improve all Parsley error messages before 1.0 release by:
- Adding error codes to ~40 uncataloged errors
- Converting parser errors to structured format
- Standardizing formatting (capitalization, quotes)
- Adding hints to high-value errors
- Consolidating duplicate error codes

Reference: `work/reports/ERROR-MESSAGE-AUDIT.md`

## Prerequisites

- [ ] Review audit report for full scope
- [ ] Ensure test suite passes before starting

## Tasks

### Task 1: Fix Capitalization Inconsistencies

**Files**: `pkg/parsley/errors/errors.go`
**Estimated effort**: Small

Steps:
1. Find ASSIGN-0001 through ASSIGN-0004 in ErrorCatalog
2. Change lowercase "cannot" to "Cannot" in each template
3. Verify no other messages start with lowercase

Changes:
```go
"ASSIGN-0001": Template: "Cannot reassign immutable binding '{{.Name}}'"
"ASSIGN-0002": Template: "Cannot reassign loop variable '{{.Name}}'"
"ASSIGN-0003": Template: "Cannot reassign function parameter '{{.Name}}'"
"ASSIGN-0004": Template: "Cannot assign to undeclared variable '{{.Name}}'"
```

Tests:
- `go test ./pkg/parsley/...`
- Grep for `Template:.*"[a-z]` to find any remaining lowercase starts

---

### Task 2: Consolidate Duplicate Error Codes

**Files**: `pkg/parsley/errors/errors.go`, evaluator files using IO-0009
**Estimated effort**: Small

Steps:
1. Remove IO-0009 (duplicate of IO-0006 "Failed to create directory")
2. Search for IO-0009 usage and replace with IO-0006
3. Review OP-0001 vs OP-0009 usage — document distinction or consolidate

Tests:
- `go test ./pkg/parsley/...`
- Verify no references to removed code remain

---

### Task 3: Simplify Overly Technical Messages

**Files**: `pkg/parsley/errors/errors.go`
**Estimated effort**: Small

Steps:
1. Update the following error templates:

| Code | Current | New |
|------|---------|-----|
| INTERNAL-0001 | `"{{.Context}} requires environment context"` | `"Internal error in {{.Context}}"` |
| INTERNAL-0002 | `"Unknown node type: {{.Type}}"` | `"Internal error: unexpected syntax element"` |
| DEST-0002 | `"Unsupported nested destructuring pattern"` | `"Nested destructuring is not supported"` |
| SPREAD-0001 | `"Spread operator requires a dictionary, got {{.Got}}"` | `"Cannot spread {{.Got}} — only dictionaries can be spread with {...}"` |
| LOOP-0003 | `"For expression missing function or body"` | `"For loop needs a body"` |
| FILEOP-0003 | `"File handle has no format specified"` | `"File format not specified"` |

2. Add hints to internal errors:
```go
"INTERNAL-0001": Hints: []string{"This is a bug in Parsley — please report it"}
"INTERNAL-0002": Hints: []string{"This is a bug in Parsley — please report it"}
```

3. Add hints to simplified errors:
```go
"DEST-0002": Hints: []string{"Destructure one level at a time"}
"LOOP-0003": Hints: []string{"for x in arr { ... }"}
"FILEOP-0003": Hints: []string{"Specify format: file('path', {format: 'json'})"}
```

Tests:
- `go test ./pkg/parsley/...`

---

### Task 4: Standardize Quote and Backtick Usage

**Files**: `pkg/parsley/errors/errors.go`
**Estimated effort**: Medium

Convention:
- Backticks for code identifiers: function names, variable names, keywords
- Single quotes for literal values provided by user
- No quotes for type names

Steps:
1. Review all templates containing `{{.Function}}`, `{{.Method}}`, `{{.Name}}`
2. Ensure they're wrapped in backticks when referring to code
3. Ensure `{{.Got}}`, `{{.Type}}` etc. are NOT quoted (they're type names)

Examples of changes needed:
```go
// Before
"{{.Function}} expected {{.Expected}}, got {{.Got}}"
// After
"`{{.Function}}` expected {{.Expected}}, got {{.Got}}"

// Before  
"{{.Function}} callback must be a function"
// After
"`{{.Function}}` callback must be a function"
```

Tests:
- `go test ./pkg/parsley/...`
- Manual review of sample error output

---

### Task 5: Add Hints to High-Value Errors

**Files**: `pkg/parsley/errors/errors.go`
**Estimated effort**: Medium

Add hints to these errors:

```go
"TYPE-0003": {
    Hints: []string{"Only functions can be called with ()"},
},
"TYPE-0008": {
    Hints: []string{"Arrays use integer indices (arr[0]); dictionaries use string keys (dict[\"key\"])"},
},
"TYPE-0009": {
    Hints: []string{"Sort callbacks should return true if a comes before b"},
},
"INDEX-0001": {
    Hints: []string{"Valid indices are 0 to length-1"},
},
"INDEX-0005": {
    Hints: []string{"Use .get(key, default) to provide a fallback value"},
},
"OP-0002": {
    Hints: []string{"Check if the divisor is zero before dividing"},
},
"DB-0002": {
    Hints: []string{"Check SQL syntax and ensure parameter count matches placeholders"},
},
"IO-0002": {
    Hints: []string{"Check that the file path exists and is spelled correctly"},
},
```

Tests:
- `go test ./pkg/parsley/...`
- Test one error in REPL to verify hint displays

---

### Task 6: Add Error Codes for Uncataloged Evaluator Errors

**Files**: `pkg/parsley/errors/errors.go`, multiple evaluator files
**Estimated effort**: Medium

Steps:

1. Add new error codes to catalog:

```go
// Box formatting errors (BOX-0xxx)
"BOX-0001": {
    Class:    ClassFormat,
    Template: "`toBox`: invalid style '{{.Style}}', expected 'single', 'double', 'ascii', or 'rounded'",
},
"BOX-0002": {
    Class:    ClassType,
    Template: "`toBox`: style option must be a string, got {{.Got}}",
},
"BOX-0003": {
    Class:    ClassType,
    Template: "`toBox`: title option must be a string, got {{.Got}}",
},
"BOX-0004": {
    Class:    ClassValue,
    Template: "`toBox`: maxWidth must be a non-negative integer",
},
"BOX-0005": {
    Class:    ClassType,
    Template: "`toBox`: maxWidth option must be an integer, got {{.Got}}",
},

// Export errors (EXPORT-0xxx)
"EXPORT-0001": {
    Class:    ClassUndefined,
    Template: "Cannot export undefined identifier '{{.Name}}'",
    Hints:    []string{"Ensure '{{.Name}}' is defined before the export statement"},
},

// Dictionary errors (DICT-0xxx)
"DICT-0001": {
    Class:    ClassType,
    Template: "Computed dictionary key must be a string, integer, float, or boolean, got {{.Got}}",
},

// ID generation errors (ID-0xxx)
"ID-0001": {
    Class:    ClassState,
    Template: "Failed to generate random bytes for `{{.Function}}`",
    Hints:    []string{"This is usually a system-level issue with the random number generator"},
},
"ID-0002": {
    Class:    ClassValue,
    Template: "NanoID length must be between 1 and 256, got {{.Got}}",
},

// Markdown doc errors (MDDOC-0xxx)
"MDDOC-0001": {
    Class:    ClassType,
    Template: "`mdDoc`: dictionary must have a 'type' field to be a valid markdown AST",
},

// Schema errors (SCHEMA-0xxx)
"SCHEMA-0001": {
    Class:    ClassArity,
    Template: "`schema.enum` requires at least one value",
},
"SCHEMA-0002": {
    Class:    ClassType,
    Template: "Schema has no fields defined",
},
"SCHEMA-0003": {
    Class:    ClassType,
    Template: "Schema fields must be a dictionary, got {{.Got}}",
},

// DSL query errors (DSL-0xxx)
"DSL-0001": {
    Class:    ClassState,
    Template: "Join subquery requires a subquery definition",
},
"DSL-0002": {
    Class:    ClassState,
    Template: "Unknown condition node type in join subquery",
},
"DSL-0003": {
    Class:    ClassState,
    Template: "Unknown condition node type in correlated subquery",
},
"DSL-0004": {
    Class:    ClassType,
    Template: "Correlated subquery conditions must be simple conditions",
},
```

2. Update evaluator files to use new codes:

| File | Line | Change |
|------|------|--------|
| `eval_box.go` | ~862 | Use `newValidationError("BOX-0001", map[string]any{"Style": styleStr.Value})` |
| `eval_box.go` | ~867 | Use `newStructuredError("BOX-0002", map[string]any{"Got": styleVal.Type()})` |
| `eval_box.go` | ~877 | Use `newStructuredError("BOX-0003", map[string]any{"Got": titleVal.Type()})` |
| `eval_box.go` | ~885 | Use `newValueError("BOX-0004", nil)` |
| `eval_box.go` | ~891 | Use `newStructuredError("BOX-0005", map[string]any{"Got": maxWidthVal.Type()})` |
| `evaluator.go` | ~943 | Reuse `ASSIGN-0001` with appropriate data |
| `evaluator.go` | ~4601 | Use `newUndefinedError("EXPORT-0001", map[string]any{"Name": node.Name.Value})` |
| `evaluator.go` | ~5539 | Use `newStructuredError("DICT-0001", map[string]any{"Got": keyObj.Type()})` |
| `stdlib_id.go` | 77,106,138,171,218 | Use `newInternalError("ID-0001", map[string]any{"Function": "funcName"})` |
| `stdlib_id.go` | ~157 | Use `newValueError("ID-0002", map[string]any{"Got": length})` |
| `stdlib_mddoc.go` | ~68 | Use `newStructuredError("MDDOC-0001", nil)` |
| `stdlib_schema.go` | ~151 | Use `newStructuredError("SCHEMA-0001", nil)` |
| `stdlib_schema.go` | ~316 | Use `newStructuredError("SCHEMA-0002", nil)` |
| `stdlib_schema.go` | ~322 | Use `newStructuredError("SCHEMA-0003", map[string]any{"Got": fieldsObj.Type()})` |
| `stdlib_dsl_query.go` | ~713 | Use `newStateError("DSL-0001")` |
| `stdlib_dsl_query.go` | ~788 | Use `newStateError("DSL-0002")` |
| `stdlib_dsl_query.go` | ~907 | Use `newStateError("DSL-0003")` |
| `stdlib_dsl_query.go` | ~997 | Use `newStructuredError("DSL-0004", nil)` |

Tests:
- `go test ./pkg/parsley/...`
- Verify JSON output includes codes for these errors

---

### Task 7: Convert Parser Errors to Structured Format

**Files**: `pkg/parsley/parser/parser.go`, `pkg/parsley/errors/errors.go`
**Estimated effort**: Medium

Steps:

1. Add new parse error codes to catalog:

```go
"PARSE-0012": {
    Class:    ClassParse,
    Template: "Could not parse '{{.Literal}}' as float",
},
"PARSE-0013": {
    Class:    ClassParse,
    Template: "Invalid money literal: {{.Literal}}",
},
"PARSE-0014": {
    Class:    ClassParse,
    Template: "Expected closing tag </{{.Expected}}>, got {{.Got}}",
},
"PARSE-0015": {
    Class:    ClassParse,
    Template: "Mismatched tags: opening <{{.Opening}}> but closing </{{.Closing}}>",
},
"PARSE-0016": {
    Class:    ClassParse,
    Template: "Expected 'in' after 'not', got {{.Got}}",
    Hints:    []string{"Use 'not in' for negated containment: x not in arr"},
},
"PARSE-0017": {
    Class:    ClassParse,
    Template: "Unterminated regex literal: {{.Literal}}",
},
"PARSE-0018": {
    Class:    ClassParse,
    Template: "Missing unit suffix in '{{.Literal}}'",
    Hints:    []string{"Unit literals need a suffix like m, cm, kg, etc."},
},
"PARSE-0019": {
    Class:    ClassParse,
    Template: "Unknown unit suffix '{{.Suffix}}' in '{{.Literal}}'",
},
```

2. Convert parser `addError` calls to `addStructuredError`:

| Location | Current | Change To |
|----------|---------|-----------|
| ~1097 | `addError(fmt.Sprintf("could not parse %q as integer"...))` | `addStructuredError("PARSE-0007", line, col, map[string]any{"Literal": lit})` |
| ~1110 | `addError(fmt.Sprintf("could not parse %q as float"...))` | `addStructuredError("PARSE-0012", line, col, map[string]any{"Literal": lit})` |
| ~1134 | `addError(fmt.Sprintf("invalid regex literal: %s"...))` | `addStructuredError("PARSE-0005", line, col, map[string]any{"Literal": lit})` |
| ~1142 | `addError(fmt.Sprintf("unterminated regex literal: %s"...))` | `addStructuredError("PARSE-0017", line, col, map[string]any{"Literal": lit})` |
| ~1230 | `addError(fmt.Sprintf("invalid money literal: %s"...))` | `addStructuredError("PARSE-0013", line, col, map[string]any{"Literal": lit})` |
| ~1272 | `addError(fmt.Sprintf("missing unit suffix in '%s'"...))` | `addStructuredError("PARSE-0018", line, col, map[string]any{"Literal": lit})` |
| ~1279 | `addError(fmt.Sprintf("unknown unit suffix '%s'..."...))` | `addStructuredError("PARSE-0019", line, col, map[string]any{"Suffix": suffix, "Literal": lit})` |
| ~1857 | `addError(fmt.Sprintf("expected closing tag </%s>..."...))` | `addStructuredError("PARSE-0014", line, col, map[string]any{"Expected": name, "Got": got})` |
| ~1866 | `addError(fmt.Sprintf("mismatched tags..."...))` | `addStructuredError("PARSE-0015", line, col, map[string]any{"Opening": open, "Closing": close})` |
| ~2483 | `addError(fmt.Sprintf("expected 'in' after 'not'..."...))` | `addStructuredError("PARSE-0016", line, col, map[string]any{"Got": got})` |
| ~2584 | `addError(fmt.Sprintf("expected ')', got '%s'"...))` | `addStructuredError("PARSE-0001", line, col, map[string]any{"Expected": ")", "Got": got})` |

Tests:
- `go test ./pkg/parsley/...`
- Verify parser errors have codes in JSON output

---

### Task 8: Convert fmt.Errorf to Structured Errors

**Files**: `pkg/parsley/evaluator/eval_datetime.go`, `eval_encoders.go`, `eval_parsing.go`, `pkg/parsley/errors/errors.go`
**Estimated effort**: Medium

Steps:

1. Add new error codes to catalog:

```go
// Datetime validation (DT-0xxx)
"DT-0001": {
    Class:    ClassType,
    Template: "Datetime dictionary missing '{{.Field}}' field",
},
"DT-0002": {
    Class:    ClassType,
    Template: "'{{.Field}}' must be an integer, got {{.Got}}",
},
"DT-0003": {
    Class:    ClassFormat,
    Template: "Unknown timezone: {{.Timezone}}",
},

// Encoding errors (ENCODE-0xxx)
"ENCODE-0001": {
    Class:    ClassType,
    Template: "Bytes format requires an array, got {{.Got}}",
},
"ENCODE-0002": {
    Class:    ClassType,
    Template: "Bytes array must contain integers, got {{.Got}} at index {{.Index}}",
},
"ENCODE-0003": {
    Class:    ClassValue,
    Template: "Byte value out of range (0-255): {{.Value}} at index {{.Index}}",
},

// Duration parsing (DUR-0xxx)
"DUR-0001": {
    Class:    ClassFormat,
    Template: "Expected digit at position {{.Position}} in duration string",
},
"DUR-0002": {
    Class:    ClassFormat,
    Template: "Missing unit after number at position {{.Position}} in duration string",
},
"DUR-0003": {
    Class:    ClassFormat,
    Template: "Unknown duration unit: '{{.Unit}}'",
    Hints:    []string{"Valid units: s (seconds), m (minutes), h (hours), d (days)"},
},
```

2. Update evaluator files — this requires refactoring functions that return `error` to return `*Error` or wrapping the errors appropriately at call sites.

Key files:
- `eval_datetime.go`: `dictToTime`, `getDurationComponents`, `getDatetimeUnix`, `parseFlexibleDateTime`
- `eval_encoders.go`: `encodeBytes`
- `eval_parsing.go`: `parseDurationString`

Tests:
- `go test ./pkg/parsley/...`

---

## Validation Checklist

- [ ] All tests pass: `go test ./pkg/parsley/...`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] Grep for `&Error{Message:` returns no results in evaluator (all use catalog)
- [ ] Grep for `addError(` in parser returns minimal results (most use `addStructuredError`)
- [ ] Manual test: verify error in REPL shows code and hints
- [ ] Manual test: verify `pars --format=json` output includes error codes

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-01-20 | Task 1: Capitalization | ✅ Complete | ASSIGN-0001 to ASSIGN-0004 fixed |
| 2025-01-20 | Task 2: Duplicates | ✅ Complete | IO-0009 consolidated into IO-0006 |
| 2025-01-20 | Task 3: Technical Language | ✅ Complete | LOOP-0003, FILEOP-0003, DEST-0002, SPREAD-0001, INTERNAL-* improved |
| 2025-01-20 | Task 4: Quote Style | ✅ Complete | Added backticks to ~15 templates |
| 2025-01-20 | Task 5: Add Hints | ✅ Complete | TYPE-0003, TYPE-0008, TYPE-0009, INDEX-*, OP-0002, DB-0002, IO-0002 |
| 2025-01-20 | Task 6: Evaluator Codes | ✅ Complete | BOX-*, EXPORT-*, DICT-*, ID-*, MDDOC-*, SCHEMA-*, DSL-* added |
| 2025-01-20 | Task 7: Parser Codes | ✅ Complete | PARSE-0012 to PARSE-0019 added, 11 parser errors converted |
| 2025-01-20 | Task 8: fmt.Errorf | ✅ Complete | DT-*, ENCODE-*, DUR-* codes added; internal helpers deferred |

## Deferred Items

- **Internal helper function refactoring**: Functions like `dictToTime`, `encodeBytes`, `parseDurationString` still return Go `error` types which are wrapped by structured errors at call sites. Full refactoring to return `*Error` would require significant signature changes. Current behavior is acceptable as errors are wrapped with proper codes (FMT-0004, FILEOP-0006, etc.).