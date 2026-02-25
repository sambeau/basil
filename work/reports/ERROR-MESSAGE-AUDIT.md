# Parsley Error Message Audit & Implementation Plan

**Date:** 2025-01-20  
**Target:** Parsley 1.0  
**Scope:** `pkg/parsley/errors/errors.go`, `pkg/parsley/parser/`, `pkg/parsley/evaluator/`

## Executive Summary

This document identifies issues with Parsley's error messages and provides an implementation plan to fix all issues before 1.0. The work is organized into 8 tasks that can be completed incrementally.

---

## Task 1: Add Error Codes to Uncataloged Evaluator Errors

**Effort:** Medium  
**Files:** `pkg/parsley/errors/errors.go`, various evaluator files

These errors bypass the catalog and lack error codes, breaking CLI JSON output and making errors harder to document.

### New Error Codes to Add

Add these to `ErrorCatalog` in `errors.go`:

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
    Template: "Failed to generate random bytes for {{.Function}}",
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

### Files to Update

| File | Line(s) | Current | Change To |
|------|---------|---------|-----------|
| `eval_box.go` | 862 | `&Error{Message: "toBox: invalid style..."}` | `newValidationError("BOX-0001", ...)` |
| `eval_box.go` | 867 | `&Error{Message: "toBox: style option..."}` | `newTypeError("BOX-0002", ...)` |
| `eval_box.go` | 877 | `&Error{Message: "toBox: title option..."}` | `newTypeError("BOX-0003", ...)` |
| `eval_box.go` | 885 | `&Error{Message: "toBox: maxWidth must be..."}` | `newValueError("BOX-0004", ...)` |
| `eval_box.go` | 891 | `&Error{Message: "toBox: maxWidth option..."}` | `newTypeError("BOX-0005", ...)` |
| `evaluator.go` | 943 | `&Error{Message: fmt.Sprintf("cannot reassign...")}` | Use existing `ASSIGN-0001` |
| `evaluator.go` | 4601 | `&Error{Message: fmt.Sprintf("undefined identifier...")}` | `newUndefinedError("EXPORT-0001", ...)` |
| `evaluator.go` | 5539 | `&Error{Message: fmt.Sprintf("computed dictionary key...")}` | `newStructuredError("DICT-0001", ...)` |
| `stdlib_id.go` | 77,106,138,171,218 | `&Error{Message: "Failed to generate..."}` | `newInternalError("ID-0001", ...)` |
| `stdlib_id.go` | 157 | `&Error{Message: "NanoID length must be..."}` | `newValueError("ID-0002", ...)` |
| `stdlib_mddoc.go` | 68 | `&Error{Message: "mdDoc: dictionary must..."}` | `newTypeError("MDDOC-0001", ...)` |
| `stdlib_schema.go` | 151 | `&Error{Message: "schema.enum requires..."}` | `newArityError or newStructuredError("SCHEMA-0001", ...)` |
| `stdlib_schema.go` | 316 | `&Error{Message: "Schema has no fields..."}` | `newStructuredError("SCHEMA-0002", ...)` |
| `stdlib_schema.go` | 322 | `&Error{Message: "Schema fields must be..."}` | `newStructuredError("SCHEMA-0003", ...)` |
| `stdlib_dsl_query.go` | 713 | `&Error{Message: "join subquery requires..."}` | `newStateError("DSL-0001")` |
| `stdlib_dsl_query.go` | 788 | `&Error{Message: "unknown condition node..."}` | `newStateError("DSL-0002")` |
| `stdlib_dsl_query.go` | 907 | `&Error{Message: "unknown condition node..."}` | `newStateError("DSL-0003")` |
| `stdlib_dsl_query.go` | 997 | `&Error{Message: "correlated subquery..."}` | `newStructuredError("DSL-0004", ...)` |

---

## Task 2: Convert Parser Errors to Structured Errors

**Effort:** Medium  
**Files:** `pkg/parsley/parser/parser.go`, `pkg/parsley/errors/errors.go`

Parser errors use `addError()` which creates errors without codes. Convert to `addStructuredError()`.

### New Error Codes to Add

```go
// Additional parse errors
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

### Conversions

| Location | Current | Change To |
|----------|---------|-----------|
| `parser.go:1097` | `addError(fmt.Sprintf("could not parse %q as integer"...))` | `addStructuredError("PARSE-0007", ...)` |
| `parser.go:1110` | `addError(fmt.Sprintf("could not parse %q as float"...))` | `addStructuredError("PARSE-0012", ...)` |
| `parser.go:1134` | `addError(fmt.Sprintf("invalid regex literal: %s"...))` | `addStructuredError("PARSE-0005", ...)` |
| `parser.go:1142` | `addError(fmt.Sprintf("unterminated regex literal: %s"...))` | `addStructuredError("PARSE-0017", ...)` |
| `parser.go:1230` | `addError(fmt.Sprintf("invalid money literal: %s"...))` | `addStructuredError("PARSE-0013", ...)` |
| `parser.go:1261` | `addError(fmt.Sprintf("invalid unit literal: %s"...))` | `addStructuredError("PARSE-0007", ...)` (reuse) |
| `parser.go:1272` | `addError(fmt.Sprintf("missing unit suffix in '%s'"...))` | `addStructuredError("PARSE-0018", ...)` |
| `parser.go:1279` | `addError(fmt.Sprintf("unknown unit suffix '%s' in '%s'"...))` | `addStructuredError("PARSE-0019", ...)` or `UNIT-0007` |
| `parser.go:1857` | `addError(fmt.Sprintf("expected closing tag </%s>..."...))` | `addStructuredError("PARSE-0014", ...)` |
| `parser.go:1866` | `addError(fmt.Sprintf("mismatched tags: opening..."...))` | `addStructuredError("PARSE-0015", ...)` |
| `parser.go:2483` | `addError(fmt.Sprintf("expected 'in' after 'not'..."...))` | `addStructuredError("PARSE-0016", ...)` |
| `parser.go:2584` | `addError(fmt.Sprintf("expected ')', got '%s'"...))` | `addStructuredError("PARSE-0001", ...)` |

---

## Task 3: Fix Capitalization Inconsistencies

**Effort:** Small  
**Files:** `pkg/parsley/errors/errors.go`

All error messages should start with a capital letter.

### Changes

| Code | Current | Fixed |
|------|---------|-------|
| `ASSIGN-0001` | `"cannot reassign immutable binding '{{.Name}}'"` | `"Cannot reassign immutable binding '{{.Name}}'"` |
| `ASSIGN-0002` | `"cannot reassign loop variable '{{.Name}}'"` | `"Cannot reassign loop variable '{{.Name}}'"` |
| `ASSIGN-0003` | `"cannot reassign function parameter '{{.Name}}'"` | `"Cannot reassign function parameter '{{.Name}}'"` |
| `ASSIGN-0004` | `"cannot assign to undeclared variable '{{.Name}}'"` | `"Cannot assign to undeclared variable '{{.Name}}'"` |

---

## Task 4: Add Hints to High-Value Errors

**Effort:** Medium  
**Files:** `pkg/parsley/errors/errors.go`

Add hints to errors where guidance would help users fix issues quickly.

### Hints to Add

| Code | Add Hints |
|------|-----------|
| `TYPE-0003` | `"Only functions can be called with ()"` |
| `TYPE-0008` | `"Arrays use integer indices (arr[0]); dictionaries use string keys (dict[\"key\"])"` |
| `TYPE-0009` | `"Sort callbacks should return true if a comes before b"` |
| `INDEX-0001` | `"Valid indices are 0 to length-1"` |
| `INDEX-0005` | `"Use .get(key, default) to provide a fallback value"` |
| `OP-0002` | `"Check if the divisor is zero before dividing"` |
| `DB-0002` | `"Check SQL syntax and ensure parameter count matches placeholders"` |
| `IO-0002` | `"Check that the file path exists and is spelled correctly"` |
| `UNDEF-0002` | Dynamic hint listing available methods (requires code change) |
| `UNDEF-0004` | Dynamic hint listing available properties (requires code change) |

### Dynamic Hints Implementation

For `UNDEF-0002` and `UNDEF-0004`, modify the error creation to include available options:

```go
// In newUndefinedMethodError or where UNDEF-0002 is created:
availableMethods := getMethodsForType(typeName) // implement this
hints := []string{fmt.Sprintf("Available methods for %s: %s", typeName, strings.Join(availableMethods, ", "))}
```

---

## Task 5: Standardize Quote and Backtick Usage

**Effort:** Medium  
**Files:** `pkg/parsley/errors/errors.go`

### Convention

- **Backticks** for code identifiers: function names, variable names, keywords
- **Single quotes** for literal values the user provided
- **No quotes** for type names

### Examples

| Current | Standardized |
|---------|--------------|
| `"{{.Function}} expected..."` | `` "`{{.Function}}` expected..." `` |
| `"Expected {{.Expected}}, got '{{.Got}}'"` | `"Expected {{.Expected}}, got '{{.Got}}'"` (OK - value) |
| `"Cannot call {{.Got}} as a function"` | `"Cannot call {{.Got}} as a function"` (OK - type) |
| `"Argument to {{.Function}} not supported"` | `` "Argument to `{{.Function}}` not supported" `` |

### Full Review Needed

Scan all templates in `ErrorCatalog` and apply the convention consistently. Estimate ~30 templates need backticks added around `{{.Function}}`, `{{.Method}}`, `{{.Name}}` etc.

---

## Task 6: Simplify Overly Technical Messages

**Effort:** Small  
**Files:** `pkg/parsley/errors/errors.go`

### Changes

| Code | Current | Improved |
|------|---------|----------|
| `INTERNAL-0001` | `"{{.Context}} requires environment context"` | `"Internal error in {{.Context}}"` + hint: `"This is a bug in Parsley. Please report it."` |
| `INTERNAL-0002` | `"Unknown node type: {{.Type}}"` | `"Internal error: unexpected syntax element"` + hint: `"This is a bug in Parsley. Please report it."` |
| `DEST-0002` | `"Unsupported nested destructuring pattern"` | `"Nested destructuring is not supported"` + hint: `"Destructure one level at a time"` |
| `SPREAD-0001` | `"Spread operator requires a dictionary, got {{.Got}}"` | `"Cannot spread {{.Got}} — only dictionaries can be spread with {...}"` |
| `LOOP-0003` | `"For expression missing function or body"` | `"For loop needs a body"` + hint: `"for x in arr { ... }"` |
| `FILEOP-0003` | `"File handle has no format specified"` | `"File format not specified"` + hint: `"Use file('path', {format: 'json'}) or file('path', {format: 'text'})"` |

---

## Task 7: Consolidate Duplicate Error Codes

**Effort:** Small  
**Files:** `pkg/parsley/errors/errors.go`, evaluator files that use these codes

### Duplicates to Resolve

| Keep | Remove | Reason |
|------|--------|--------|
| `IO-0006` | `IO-0009` | Both say "Failed to create directory" — keep one, update references |
| `OP-0001` | — | Keep as generic; ensure `OP-0009` has distinct use case |

### OP-0001 vs OP-0009 Clarification

- `OP-0001`: "Unknown operator" — the operator doesn't exist for these types at all
- `OP-0009`: "Type mismatch" — the operator exists but types are incompatible

Review usages and ensure they're used correctly, or consolidate if distinction isn't meaningful.

---

## Task 8: Convert fmt.Errorf to Structured Errors

**Effort:** Medium  
**Files:** Various evaluator files

These internal `fmt.Errorf` calls bubble up and lose structure.

### New Error Codes Needed

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
    Template: "Expected digit at position {{.Position}} in duration",
},
"DUR-0002": {
    Class:    ClassFormat,
    Template: "Missing unit after number in duration at position {{.Position}}",
},
"DUR-0003": {
    Class:    ClassFormat,
    Template: "Unknown duration unit: {{.Unit}}",
    Hints:    []string{"Valid units: s (seconds), m (minutes), h (hours), d (days)"},
},
```

### Files to Update

| File | Location | Change |
|------|----------|--------|
| `eval_datetime.go` | 90-119 | Use `DT-0001`, `DT-0002` for dict validation |
| `eval_datetime.go` | 193-236 | Use `DT-0001`, `DT-0002` for duration/datetime parsing |
| `eval_datetime.go` | 654 | Use `DT-0003` for unknown timezone |
| `eval_encoders.go` | 27-39 | Use `ENCODE-0001`, `ENCODE-0002`, `ENCODE-0003` |
| `eval_parsing.go` | 58-106 | Use `DUR-0001`, `DUR-0002`, `DUR-0003` |

---

## Implementation Order

Recommended order to minimize conflicts and allow incremental testing:

1. **Task 3: Fix Capitalization** — Small, isolated change to `errors.go`
2. **Task 7: Consolidate Duplicates** — Small, need to update references
3. **Task 6: Simplify Technical Messages** — Small, isolated to `errors.go`
4. **Task 5: Standardize Quotes** — Medium, isolated to `errors.go`
5. **Task 4: Add Hints** — Medium, mostly `errors.go` with some evaluator changes
6. **Task 1: Uncataloged Evaluator Errors** — Medium, touches many files
7. **Task 2: Parser Errors** — Medium, focused on `parser.go`
8. **Task 8: fmt.Errorf Conversion** — Medium, touches several evaluator files

---

## Testing Strategy

1. **After each task:** Run `go test ./pkg/parsley/...` to ensure no regressions
2. **Error message tests:** Add/update tests in `pkg/parsley/tests/` that verify:
   - Error codes are present
   - Error messages match expected text
   - Hints are included where expected
3. **CLI output test:** Verify `pars --format=json` produces valid structured errors
4. **Manual verification:** Test a few errors in the REPL to verify readability

---

## Appendix: Current Error Code Inventory

| Category | Code Range | Count | With Hints |
|----------|------------|-------|------------|
| PARSE | PARSE-0001 to PARSE-0011 | 11 | 3 |
| TYPE | TYPE-0001 to TYPE-0023 | 23 | 4 |
| ARITY | ARITY-0001 to ARITY-0006 | 6 | 0 |
| UNDEF | UNDEF-0001 to UNDEF-0020 | 10 | 0 |
| IO | IO-0001 to IO-0010 | 10 | 0 |
| DB | DB-0001 to DB-0017 | 14 | 2 |
| NET | NET-0001 to NET-0009 | 9 | 0 |
| SEC | SEC-0001 to SEC-0006 | 6 | 5 |
| INDEX | INDEX-0001 to INDEX-0005 | 5 | 0 |
| FMT | FMT-0001 to FMT-0012 | 12 | 0 |
| VALUE | VALUE-0001 to VALUE-0003 | 3 | 0 |
| OP | OP-0001 to OP-0021 | 21 | 4 |
| ASSIGN | ASSIGN-0001 to ASSIGN-0004 | 4 | 4 |
| STATE | STATE-0001 to STATE-0003 | 3 | 0 |
| IMPORT | IMPORT-0001 to IMPORT-0006 | 6 | 1 |
| UNIT | UNIT-0001 to UNIT-0015 | 15 | 11 |
| Other | Various | ~40 | ~10 |

**Current total:** ~198 cataloged errors, ~44 with hints (22%)

### After Implementation

New codes to add: ~25-30
Expected total: ~225 cataloged errors
Target hints coverage: ~40%