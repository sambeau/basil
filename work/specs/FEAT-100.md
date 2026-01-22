---
id: FEAT-100
title: "Parsley Pretty-Printer"
status: complete
priority: medium
created: 2026-01-21
author: "@human + AI"
---

# FEAT-100: Parsley Pretty-Printer

## Summary
A code formatter for Parsley that produces consistently formatted, readable code. Primary use cases: REPL output that can be copy-pasted, documentation examples, and eventual `pars fmt` command.

## User Story
As a Parsley developer, I want consistent code formatting so that REPL output is readable and copy-pasteable, documentation examples are uniform, and code diffs are meaningful.

## Acceptance Criteria
- [x] Functions display with proper formatting in REPL (`Inspect()` / `ObjectToReprString()`)
- [x] Core Parsley constructs format correctly (literals, arrays, dicts, functions, records)
- [x] Threshold-based line breaking (short → inline, long → multiline)
- [x] Trailing commas on multiline structures
- [x] Formatting constants are easily configurable for tuning
- [x] Output is valid, parseable Parsley code
- [x] REPL uses pretty-printed output

## Design Decisions

### Line Width & Thresholds (Rust-inspired)
- **Max line width**: 100 chars — Modern standard, fits most editors
- **Array/dict/chain threshold**: 60 chars (60% of max) — Rust's `use_small_heuristics` approach
- **Function call args**: 60 chars
- **Single-line if/else**: 50 chars
- **Rationale**: Different constructs have different "comfortable" widths; one threshold doesn't fit all

### Trailing Commas (Go-style)
- Required on last element of multiline structures
- **Rationale**: Cleaner diffs, easier reordering

### Minimal Semicolons (Go-style)
- Only where grammatically required
- **Rationale**: Less noise, Parsley has implicit semicolons

### K&R Brace Style
- Opening brace on same line as statement
- **Rationale**: Industry standard, compact

### Query DSL Multiline Format
- Table name on own indented line
- Closing paren on newline
- **Rationale**: Matches SQL-like visual structure

### Attribute Ordering
- Preserve user order (default)
- **Rationale**: Sorting is contentious; spreads have semantic meaning

### Empty Lines
- 2 blank lines between top-level definitions
- **Rationale**: Visual separation

### Import Grouping
- Std library first, then project imports, sorted within groups

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Configuration Constants

All thresholds should be defined as named constants for easy tuning:

```go
// pkg/parsley/format/constants.go

package format

// Line width
const (
    MaxLineWidth = 100
)

// Thresholds (percentage of MaxLineWidth)
const (
    ThresholdSmall      = 60  // 60% — arrays, dicts, chains, function args
    ThresholdIfElse     = 50  // 50% — single-line if/else
    ThresholdQueryInline = 60 // max chars for inline query
)

// Computed thresholds
var (
    ArrayThreshold    = MaxLineWidth * ThresholdSmall / 100    // 60
    DictThreshold     = MaxLineWidth * ThresholdSmall / 100    // 60
    ChainThreshold    = MaxLineWidth * ThresholdSmall / 100    // 60
    FuncArgsThreshold = MaxLineWidth * ThresholdSmall / 100    // 60
    IfElseThreshold   = MaxLineWidth * ThresholdIfElse / 100   // 50
)

// Query DSL
const (
    QueryMaxInlineClauses = 2  // max clauses for inline query
)

// Structure
const (
    IndentWidth          = 4
    IndentString         = "    "
    BlankLinesBetweenDefs = 2
)

// Trailing commas
const (
    TrailingCommaMultiline = true
)
```

### Affected Components

- `pkg/parsley/format/` — New package for pretty-printer
  - `constants.go` — Configurable thresholds
  - `printer.go` — Main formatter logic
  - `printer_test.go` — Unit tests
- `pkg/parsley/object/object.go` — Update `Inspect()` methods to use formatter
- `pkg/parsley/evaluator/` — `ObjectToReprString()` integration

### Key Types

```go
// Printer holds formatting state
type Printer struct {
    indent    int      // current indentation level
    output    strings.Builder
    lineWidth int      // current line position
}

// Format is the main entry point
func Format(node ast.Node) string

// Helper methods
func (p *Printer) write(s string)
func (p *Printer) newline()
func (p *Printer) indentInc()
func (p *Printer) indentDec()
func (p *Printer) fitsOnLine(s string) bool
```

### Formatting Rules by Construct

| Construct | Inline if... | Multiline format |
|-----------|--------------|------------------|
| Array | ≤60 chars | One element per line, trailing comma |
| Dict | ≤60 chars | One key-value per line, trailing comma |
| Function (simple) | Single expr body | Braces, indented body |
| Function params | ≤60 chars | One param per line, trailing comma |
| Method chain | ≤60 chars | Break after each `.method()` |
| If/else | ≤50 chars, simple bodies | Braces, standard indentation |
| For | Simple body | Braces, indented body |
| Tag | Few short attrs | One attr per line |
| Query | ≤2 clauses AND ≤60 chars | Table on own line, clause per line, closing paren on newline |
| Schema | — | One field per line |

### Edge Cases & Constraints

1. **Nested structures** — Recursively check if inner content fits before deciding inline vs multiline
2. **Strings with newlines** — Preserve internal newlines, don't count toward line width threshold
3. **Comments** — Preserve position relative to code (future: comment attachment)
4. **Tags with code children** — Code goes directly inside tags (no braces)
5. **Spread in dicts/tags** — Preserve position (semantic meaning)
6. **Query DSL operators** — `|`, `|<`, `??->`, `?->`, `.` have specific formatting rules

### Dependencies
- Depends on: Stable AST representation
- Blocks: `pars fmt` command, documentation auto-formatting

## Implementation Notes

### Phase 1 Complete (2026-01-21)

**Files created:**
- `pkg/parsley/format/constants.go` — Configurable thresholds
- `pkg/parsley/format/printer.go` — Core printer with indent/write/newline
- `pkg/parsley/format/format.go` — Type dispatch and formatting logic
- `pkg/parsley/format/format_test.go` — 16 unit tests with mock types
- `pkg/parsley/evaluator/format_accessors.go` — Adapter wrappers for evaluator→format bridge
- `pkg/parsley/evaluator/format_accessors_test.go` — 19 integration tests

**Implemented features:**
- INTEGER, FLOAT, BOOLEAN, STRING, NULL, MONEY literals
- Arrays (inline/multiline with 60-char threshold)
- Dictionaries (inline/multiline, key quoting for non-identifiers)
- Functions (inline/multiline body)
- Records (schema name prefix, field formatting)
- Nested structure support
- Trailing commas on multiline

**API:**
- `format.FormatObject(obj TypedObject) string` — Low-level format
- `format.FormatValue(v interface{}) string` — Flexible entry point
- `evaluator.FormatObject(obj Object) string` — High-level for REPL

### Phase 2 Complete (2026-01-21)

**Files modified:**
- `pkg/parsley/evaluator/eval_string_conversions.go` — Added `ObjectToFormattedReprString()` with multiline support
- `pkg/parsley/repl/repl.go` — Updated to use `ObjectToFormattedReprString()` for output
- `pkg/parsley/evaluator/formatted_repr_test.go` — 11 new tests for formatted output

**REPL now displays:**
- Short arrays inline: `[1, 2, 3]`
- Long arrays multiline with trailing commas
- Short dicts inline: `{name: "Alice", age: 30}`
- Long dicts multiline with trailing commas
- Nested structures with proper indentation
- Special types (datetime, duration, path, etc.) preserved

### Phase 3 Complete (2026-01-21)

**Files created:**
- `pkg/parsley/format/ast_format.go` — AST-based source code formatting (~1670 lines)
- `pkg/parsley/format/ast_format_test.go` — 30+ tests for AST formatting
- `cmd/pars/main.go` — Added `fmt` subcommand

**Implemented AST formatters:**
- **Statements:** let, assignment, return, expression, block, export, check
- **Expressions:** identifiers, literals (int, float, bool, string), arrays, dicts, functions
- **Control flow:** if/else (with threshold-based inline/multiline), for loops
- **Method chains:** with 60-char threshold for inline/multiline
- **Tags:** TagLiteral (self-closing), TagPairExpression (paired tags with content)
- **Schemas:** @schema declarations with field formatting
- **Tables:** @table and @table(Schema) with multiline row formatting
- **Operators:** infix, prefix, dot, index, slice, is/is not

**Query DSL formatting:**
- `@query()` — Inline for ≤2 clauses AND ≤60 chars; multiline with table name on own line, one clause per line, closing paren indented
- `@insert()` — Inline for short inserts; multiline with one field write per line
- `@update()` — Conditions and writes formatted separately
- `@delete()` — Conditions formatted with proper indentation
- `@transaction {}` — Statements indented inside braces
- String values properly quoted in all Query DSL expressions

**`pars fmt` CLI command:**
- `pars fmt file.pars` — Print formatted output to stdout
- `pars fmt -w file.pars` — Write result to source file
- `pars fmt -d file.pars` — Display diffs
- `pars fmt -l *.pars` — List files that differ from formatted

**Key features:**
- `FormatNode(node)` — Format any AST node
- `FormatProgram(prog)` — Format entire program with spacing
- Threshold-based formatting decisions
- Trailing commas on multiline structures
- Proper indentation at all levels

### Known Limitations

**Cannot Fix Without Architectural Changes:**

1. **Comment preservation** (BACKLOG #84)
   - Comments are skipped by the lexer (`skipComment()` in lexer.go)
   - Never tokenized or stored in AST
   - Formatting currently deletes all comments
   - Fix requires: lexer→AST→parser changes to track and attach comments
   - Significant architectural change (~3-5 days estimated)

2. **Tag attribute multiline breaking** (BACKLOG #85)
   - Tag attributes stored as raw strings (`TagLiteral.Raw`, `TagPairExpression.Props`)
   - Not parsed into individual attribute structures
   - Formatter preserves attributes as-is (multiline preserved, not reformatted)
   - Fix requires: parser changes to parse attributes into AST nodes

**Working as Intended:**

3. **String quote normalization**
   - Single-quoted strings WITHOUT `@{}` interpolation become double-quoted
   - This is correct: `"hello"` and `'hello'` are semantically equivalent (both `STRING` token type)
   - Lexer intentionally treats them the same (escape processing happens at lex time)
   - Quote-sensitive types ARE preserved:
     - Template literals `` `...` `` remain backticks
     - Raw templates `'...@{x}...'` remain single-quoted
   - Only simple strings normalize to double quotes (formatter convention)

### Future Enhancements (Backlog)
See `work/BACKLOG.md` for:
- #84: Comment preservation — requires lexer/parser changes to track comments
- #85: Tag attribute multiline — requires parser changes to parse attributes
- Editor integration (format on save) — requires LSP work

## Related
- Design doc: `work/design/DESIGN-PRETTY-PRINTER.md`
- Plan: `work/plans/PLAN-072-pretty-printer.md`
