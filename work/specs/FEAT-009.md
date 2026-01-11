---
id: FEAT-009
title: "Trailing Commas in Collections and Function Calls"
status: complete
priority: low
created: 2025-12-01
author: "@human"
---

# FEAT-009: Trailing Commas in Collections and Function Calls

## Summary
Allow an optional trailing comma after the last element in array literals, dictionary literals, and function call arguments. This makes copy-pasting lines easier and produces cleaner diffs when adding or removing items.

## User Story
As a Parsley developer, I want to be able to include a trailing comma after the last item in arrays, dictionaries, and function calls so that I can easily copy/paste lines and get cleaner version control diffs.

## Acceptance Criteria
- [x] Arrays allow trailing comma: `[1, 2, 3,]` parses as `[1, 2, 3]`
- [x] Dictionaries allow trailing comma: `{a: 1, b: 2,}` parses as `{a: 1, b: 2}`
- [x] Function calls allow trailing comma: `foo(1, 2, 3,)` parses as `foo(1, 2, 3)`
- [x] Works for both single-line and multi-line
- [x] Multiple trailing commas are an error: `[1, 2,,]` → parse error
- [x] Empty collections still work: `[]`, `{}`
- [x] Single element with trailing comma works: `[1,]`, `{a: 1,}`

## Design Decisions
- **Allow everywhere (not just multi-line)**: Consistent with JavaScript, Go, and Python. Simpler to implement and explain.
- **Multiple commas are an error**: Unlike JavaScript (which creates sparse arrays), we follow Go/Python and treat `[1,,]` as an error. This catches typos.
- **Function definitions excluded for now**: Focus on the common case (literals and calls). Function parameter definitions can be added later if needed.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `pkg/parsley/parser/parser.go` — Modify `parseArrayLiteral`, `parseHashLiteral`, `parseCallExpression` to accept trailing comma

### Dependencies
- Depends on: None
- Blocks: None

### Edge Cases & Constraints
1. `[,]` — Error (leading comma, no elements)
2. `[1,,]` — Error (multiple trailing commas)
3. `[1,]` — Valid, same as `[1]`
4. `[]` — Valid (empty array, no change needed)
5. `{,}` — Error (leading comma)
6. `{a:1,,}` — Error (multiple trailing commas)
7. `foo(,)` — Error (leading comma)
8. `foo(1,,)` — Error (multiple trailing commas)

## Implementation Notes
Modified three parser functions to check for trailing comma after consuming a comma:
- `parseSquareBracketArrayLiteral` - arrays
- `parseDictionaryLiteral` - dictionaries  
- `parseExpressionList` - function call arguments

The pattern is simple: after consuming a comma, check if the next token is the closing delimiter (], }, or )). If so, break out of the loop instead of trying to parse another element.

## Related
- Plan: N/A (simple implementation, no separate plan needed)
