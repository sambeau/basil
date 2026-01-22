---
id: PLAN-072
feature: FEAT-100
title: "Implementation Plan for Parsley Pretty-Printer"
status: draft
created: 2026-01-21
---

# Implementation Plan: FEAT-100 Parsley Pretty-Printer

## Overview
Implement a code formatter for Parsley that produces consistently formatted, readable code. Initial target: REPL output via `Inspect()` and `ObjectToReprString()`.

## Prerequisites
- [x] Design document complete (`work/design/DESIGN-PRETTY-PRINTER.md`)
- [x] Feature spec complete (`work/specs/FEAT-100.md`)
- [x] Formatting rules agreed (thresholds, trailing commas, etc.)

## Tasks

### Task 1: Create format package with constants
**Files**: `pkg/parsley/format/constants.go`
**Estimated effort**: Small

Steps:
1. Create `pkg/parsley/format/` directory
2. Create `constants.go` with all configurable thresholds
3. Group constants logically (line width, thresholds, structure)
4. Add comments explaining each constant's purpose

```go
package format

// Line width
const MaxLineWidth = 100

// Thresholds (as percentage of MaxLineWidth)
const (
    ThresholdSmallPercent  = 60
    ThresholdIfElsePercent = 50
)

// Computed thresholds
var (
    ArrayThreshold    = MaxLineWidth * ThresholdSmallPercent / 100  // 60
    DictThreshold     = MaxLineWidth * ThresholdSmallPercent / 100  // 60
    ChainThreshold    = MaxLineWidth * ThresholdSmallPercent / 100  // 60
    FuncArgsThreshold = MaxLineWidth * ThresholdSmallPercent / 100  // 60
    IfElseThreshold   = MaxLineWidth * ThresholdIfElsePercent / 100 // 50
)

// Query DSL
const (
    QueryMaxInlineClauses = 2
    QueryInlineThreshold  = 60
)

// Structure
const (
    IndentWidth           = 4
    IndentString          = "    "
    BlankLinesBetweenDefs = 2
)
```

Tests:
- Verify computed thresholds are correct
- Verify changing MaxLineWidth cascades to computed values

---

### Task 2: Create Printer struct and core methods
**Files**: `pkg/parsley/format/printer.go`
**Estimated effort**: Medium

Steps:
1. Create `Printer` struct with state (indent level, output buffer, line position)
2. Implement core methods: `write()`, `newline()`, `indent()`, `dedent()`
3. Implement `fitsOnLine(s string) bool` helper
4. Implement `currentLineWidth() int` helper

```go
type Printer struct {
    output    strings.Builder
    indent    int
    linePos   int  // current position in line
}

func NewPrinter() *Printer
func (p *Printer) String() string
func (p *Printer) write(s string)
func (p *Printer) writeln(s string)
func (p *Printer) newline()
func (p *Printer) indentInc()
func (p *Printer) indentDec()
func (p *Printer) writeIndent()
func (p *Printer) fitsOnLine(s string, threshold int) bool
```

Tests:
- `write()` updates linePos correctly
- `newline()` resets linePos, adds indent on next write
- `fitsOnLine()` respects threshold
- Indentation increases/decreases correctly

---

### Task 3: Implement literal formatting
**Files**: `pkg/parsley/format/literals.go`
**Estimated effort**: Medium

Steps:
1. Implement `formatString(s *object.String)` — preserve quote style
2. Implement `formatNumber(n *object.Number)`
3. Implement `formatBoolean(b *object.Boolean)`
4. Implement `formatNull()`
5. Implement `formatMoney(m *object.Money)`
6. Implement `formatDatetime(d *object.Datetime)`
7. Implement `formatDuration(d *object.Duration)`
8. Implement `formatRegex(r *object.Regex)`
9. Implement `formatPath(p *object.Path)`
10. Implement `formatURL(u *object.URL)`

Tests:
- Each literal type formats correctly
- Strings preserve original quote style
- Special values (null, true, false) format correctly

---

### Task 4: Implement array formatting
**Files**: `pkg/parsley/format/collections.go`
**Estimated effort**: Medium

Steps:
1. Calculate total array length if inline
2. If ≤ threshold → format inline `[a, b, c]`
3. If > threshold → format multiline with trailing comma
4. Handle nested arrays recursively
5. Handle empty arrays `[]`

```parsley
// Inline (≤60 chars)
[1, 2, 3]

// Multiline (>60 chars)
[
    "alice",
    "bob",
    "charlie",
]
```

Tests:
- Empty array → `[]`
- Short array → inline
- Long array → multiline with trailing comma
- Nested arrays respect thresholds recursively

---

### Task 5: Implement dictionary formatting
**Files**: `pkg/parsley/format/collections.go`
**Estimated effort**: Medium

Steps:
1. Calculate total dict length if inline
2. If ≤ threshold → format inline `{a: 1, b: 2}`
3. If > threshold → format multiline with trailing comma
4. Handle nested dicts recursively
5. Handle empty dicts `{}`
6. Handle spread `{...x}` — preserve position

```parsley
// Inline (≤60 chars)
{name: "Alice", age: 30}

// Multiline (>60 chars)
{
    name: "Alice",
    email: "alice@example.com",
    phone: "+1 555 1234",
}
```

Tests:
- Empty dict → `{}`
- Short dict → inline
- Long dict → multiline with trailing comma
- Spread preserved in position
- Keys format correctly (no quotes unless needed)

---

### Task 6: Implement function formatting
**Files**: `pkg/parsley/format/functions.go`
**Estimated effort**: Medium

Steps:
1. Format parameters (inline if ≤ threshold, else multiline)
2. Check if body is single expression
3. Single expr + short → `fn(x) { x * 2 }`
4. Multi-statement → multiline body
5. Handle closures (captured variables — future)

```parsley
// Simple inline
fn(x) { x * 2 }

// Multiline body
fn(x) {
    let y = x * 2
    y + 1
}

// Long parameters
fn(
    name,
    email,
    password,
) {
    // body
}
```

Tests:
- Simple function → inline
- Multi-statement → multiline
- Long params → one per line with trailing comma
- Parameters comma-separated correctly

---

### Task 7: Implement control flow formatting
**Files**: `pkg/parsley/format/control.go`
**Estimated effort**: Medium

Steps:
1. Implement `formatIf()` — inline if short, else multiline
2. Implement `formatFor()` — inline if short body
3. Implement `formatCheck()` — always single line

```parsley
// Short if → inline
let status = if (age >= 18) "adult" else "minor"

// Long if → multiline
let grade = if (score >= 90) {
    "A"
} else if (score >= 80) {
    "B"
} else {
    "F"
}

// For expression
for (n in nums) { n * 2 }

// Check guard
check x > 0 else "must be positive"
```

Tests:
- Short if/else → inline
- Long if/else → multiline
- If-else-if chains → multiline
- For with simple body → inline
- For with complex body → multiline
- Check statements format correctly

---

### Task 8: Implement method chain formatting
**Files**: `pkg/parsley/format/chains.go`
**Estimated effort**: Medium

Steps:
1. Calculate chain length if inline
2. If ≤ threshold → inline `a.b().c()`
3. If > threshold → break after each method
4. Handle method args that are themselves long (closures)

```parsley
// Short chain
name.trim().toUpper()

// Long chain
data
    .filter(fn(x) { x.active })
    .map(fn(x) { x.name })
    .sort()
```

Tests:
- Short chain → inline
- Long chain → one method per line
- Chains with closure args format correctly

---

### Task 9: Implement tag formatting
**Files**: `pkg/parsley/format/tags.go`
**Estimated effort**: Large

Steps:
1. Self-closing tags `<br/>`
2. Short tags with content → inline
3. Tags with many attributes → one per line
4. Tags with code children (no braces around code)
5. Handle boolean attributes (shorthands)
6. Handle spread attributes → preserve position

```parsley
// Self-closing
<br/>

// Short inline
<p>Hello</p>

// Multi-line attributes
<input
    type=email
    name=email
    placeholder="Enter email"
    required
/>

// Code children (no braces!)
<ul>
    for (item in items) {
        <li>item.name</li>
    }
</ul>
```

Tests:
- Self-closing tags format correctly
- Short tags stay inline
- Many attributes → one per line
- Boolean attrs at end
- Code children have no braces
- Spread attributes preserved

---

### Task 10: Implement schema formatting
**Files**: `pkg/parsley/format/schema.go`
**Estimated effort**: Medium

Steps:
1. Format `@schema Name { ... }`
2. One field per line
3. Field constraints inline if short
4. Metadata on same line if short, else next line
5. `id` field first if present

```parsley
@schema User {
    id: int(auto)
    name: string(min: 2, required)
    email: email(unique: true)
    role: enum["admin", "user"] = "user"
}
```

Tests:
- Schema declaration formats correctly
- Fields one per line
- Constraints format correctly
- Default values format correctly
- `id` field sorted to top

---

### Task 11: Implement Query DSL formatting
**Files**: `pkg/parsley/format/query.go`
**Estimated effort**: Large

Steps:
1. Check if query fits inline (≤2 clauses AND ≤60 chars)
2. Inline format: `@query(Users | status == "active" ??-> *)`
3. Multiline format: table on own line, clauses indented, closing paren on newline
4. Handle all operators: `|`, `|<`, `??->`, `?->`, `.`, `+ by`
5. Format @insert, @update, @delete similarly
6. Format @transaction blocks

```parsley
// Inline (≤2 clauses, ≤60 chars)
@query(Users | id == {userId} ?-> *)

// Multiline
@query(
    Users
    | status == "active"
    | role == "admin"
    ??-> *
)

// Insert
@insert(
    Users
    |< name: "Alice"
    |< email: "alice@test.com"
    ?-> *
)

// Transaction
@transaction {
    let user = @insert(Users |< name: "Alice" ?-> *)
    @insert(Profiles |< user_id: {user.id} .)
    user
}
```

Tests:
- Short query → inline
- Long query → multiline with table on own line
- All query operators format correctly
- Insert/update/delete format correctly
- Transactions format correctly

---

### Task 12: Implement table formatting
**Files**: `pkg/parsley/format/tables.go`
**Estimated effort**: Small

Steps:
1. Short table literal → inline
2. Long table → one row per line
3. Handle `@table(Schema)` prefix

```parsley
// Short
@table [{x: 1}, {x: 2}]

// Long
@table [
    {name: "Alice", age: 30},
    {name: "Bob", age: 25},
]
```

Tests:
- Short table → inline
- Long table → multiline
- Schema prefix formats correctly

---

### Task 13: Create main Format() entry point
**Files**: `pkg/parsley/format/format.go`
**Estimated effort**: Small

Steps:
1. Create `Format(node ast.Node) string` function
2. Create `FormatObject(obj object.Object) string` function
3. Dispatch to appropriate formatter based on type
4. Handle unknown types gracefully

Tests:
- All supported types dispatch correctly
- Unknown types return reasonable fallback

---

### Task 14: Integrate with REPL (Inspect methods)
**Files**: `pkg/parsley/object/object.go`, various object files
**Estimated effort**: Medium

Steps:
1. Update `Function.Inspect()` to use formatter
2. Update other `Inspect()` methods as needed
3. Ensure backward compatibility (existing tests pass)
4. Consider adding `InspectPretty()` vs `Inspect()` distinction

Tests:
- REPL output is formatted correctly
- Existing tests still pass
- Functions display with proper formatting

---

### Task 15: Comprehensive integration tests
**Files**: `pkg/parsley/format/format_test.go`
**Estimated effort**: Medium

Steps:
1. Create test cases from design doc examples
2. Test all constructs together (nested, complex)
3. Test edge cases (empty, very long, deeply nested)
4. Add golden file tests for complex examples

Tests:
- All design doc examples format as specified
- Round-trip: parse(format(parse(code))) == parse(code)
- Edge cases handled gracefully

---

## Validation Checklist
- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] REPL shows formatted output
- [ ] Design doc examples all work
- [ ] work/BACKLOG.md updated with deferrals

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-01-21 | Task 1: Constants | ✅ Complete | `pkg/parsley/format/constants.go` |
| 2026-01-21 | Task 2: Printer core | ✅ Complete | `pkg/parsley/format/printer.go` |
| 2026-01-21 | Task 3: Literals | ✅ Complete | INT, FLOAT, BOOL, STRING, NULL, MONEY |
| 2026-01-21 | Task 4: Arrays | ✅ Complete | Inline/multiline with threshold |
| 2026-01-21 | Task 5: Dicts | ✅ Complete | Inline/multiline, key quoting |
| 2026-01-21 | Task 6: Functions | ✅ Complete | Inline/multiline body |
| 2026-01-21 | Task 13: Entry point | ✅ Complete | `format.FormatObject()`, `evaluator.FormatObject()` |
| 2026-01-21 | Task 14: REPL integration | ✅ Complete | `ObjectToFormattedReprString()` in REPL |
| 2026-01-21 | Task 15: Integration tests | ✅ Complete | 46+ tests across format + evaluator packages |
| | Task 7: Control flow | 📋 Backlog #77 | Deferred - not needed for core REPL |
| | Task 8: Chains | 📋 Backlog #78 | Deferred - not needed for core REPL |
| | Task 9: Tags | 📋 Backlog #79 | Deferred - not needed for core REPL |
| | Task 10: Schemas | 📋 Backlog #80 | Deferred - not needed for core REPL |
| | Task 11: Query DSL | 📋 Backlog #81 | Deferred - not needed for core REPL |
| | Task 12: Tables | ⬜ Not started | Tables already have good Inspect() |

## Deferred Items
Items to add to work/BACKLOG.md after implementation:
- `pars fmt` CLI command — requires this formatter
- Semantic attribute ordering for tags — complex, needs SVG support
- Comment preservation/attachment — requires parser changes
- Editor integration (format on save) — requires LSP work
