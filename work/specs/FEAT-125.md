---
id: FEAT-125
title: "Error Message Improvements"
status: planned
created: 2025-01-20
---

# FEAT-125: Error Message Improvements

## Summary

Improve Parsley's error messages for clarity, consistency, and completeness before 1.0 release. This includes adding error codes to uncataloged errors, converting parser errors to structured format, standardizing message formatting, adding helpful hints, and consolidating duplicate errors.

## Motivation

1. **CLI JSON output**: Uncataloged errors lack codes, breaking structured error output
2. **User experience**: Inconsistent formatting and technical language confuse users
3. **Documentation**: Errors without codes are harder to document and search for
4. **Debugging**: Missing hints leave users without guidance on how to fix issues

## Requirements

### Must Have

1. All runtime errors must have error codes (no `&Error{Message: "..."}` bypassing catalog)
2. All parser errors must use structured error format with codes
3. Consistent capitalization (all messages start with capital letter)
4. Consistent quote style (backticks for code, single quotes for values)
5. Duplicate error codes consolidated

### Should Have

1. Hints added to high-value errors (TYPE, INDEX, UNDEF errors)
2. Technical language simplified for user-facing messages
3. Dynamic hints for undefined method/property errors listing available options

## Non-Goals

- Internationalization of error messages (post-1.0)
- Error message customization API (post-1.0)
- HTML-specific error rendering changes

## Design

### New Error Code Categories

| Category | Purpose | Count |
|----------|---------|-------|
| BOX-0xxx | Box formatting errors | 5 |
| EXPORT-0xxx | Export statement errors | 1 |
| DICT-0xxx | Dictionary errors | 1 |
| ID-0xxx | ID generation errors | 2 |
| MDDOC-0xxx | Markdown doc errors | 1 |
| SCHEMA-0xxx | Schema validation errors | 3 |
| DSL-0xxx | Query DSL errors | 4 |
| PARSE-0012+ | Additional parse errors | 8 |
| DT-0xxx | Datetime validation | 3 |
| ENCODE-0xxx | Encoding errors | 3 |
| DUR-0xxx | Duration parsing | 3 |

### Formatting Conventions

- **Capitalization**: All messages start with capital letter
- **Code identifiers**: Wrapped in backticks (`` `functionName` ``)
- **Literal values**: Wrapped in single quotes (`'value'`)
- **Type names**: No quotes (`got array`)

### Hint Guidelines

Hints should:
- Provide actionable guidance
- Show correct syntax examples
- Suggest alternatives when available

## Testing

1. Verify all errors have codes via grep/static analysis
2. Test CLI JSON output includes code for all error types
3. Test specific error messages match expected format
4. Manual verification of hint usefulness

## Related

- Audit report: `work/reports/ERROR-MESSAGE-AUDIT.md`
- Implementation plan: `work/plans/PLAN-104.md`

## References

- Error catalog: `pkg/parsley/errors/errors.go`
- Parser errors: `pkg/parsley/parser/parser.go`
- Evaluator errors: `pkg/parsley/evaluator/eval_errors.go`
