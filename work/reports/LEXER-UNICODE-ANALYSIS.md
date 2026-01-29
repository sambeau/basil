# Lexer Unicode Support Analysis

**Date:** 2026-01-29  
**Status:** Analysis complete, enhancement proposed for backlog

## Summary

The Parsley lexer currently uses byte-based scanning, which prevents Unicode identifiers (e.g., `let π = 3.14`). This report documents the current design, its limitations, and proposes a hybrid approach for future enhancement.

## Current Design

### Byte-Based Scanning

The lexer stores the current character as a `byte`:

```go
// pkg/parsley/lexer/lexer.go line 429
ch byte // current char under examination
```

Key methods operate on bytes:
- `readChar()` — reads single byte, advances position
- `peekChar()` — returns next byte without advancing
- `peekCharN(n)` — returns byte n positions ahead (O(1) lookup)

### Why Bytes Work for Syntax

1. **All operators are ASCII** — `<=??=>`, `{`, `}`, `+`, etc.
2. **All keywords are ASCII** — `fn`, `let`, `for`, `if`, `else`, etc.
3. **UTF-8 in strings is preserved** — `readString()`, `readTemplate()`, `readTagText()` read raw bytes into Go strings, which naturally preserve valid UTF-8

### The Broken `isLetter()` Attempt

There's dead code that *appears* to support Unicode letters:

```go
func isLetter(ch byte) bool {
    r, _ := utf8.DecodeRune([]byte{ch})
    return unicode.IsLetter(r) || ch == '_'
}
```

**Why it doesn't work:**
- `ch` is a single byte, not a complete UTF-8 sequence
- For `π` (U+03C0, encoded as `0xCF 0x80`), the first byte `0xCF` alone decodes as `utf8.RuneError`
- For emoji (4 bytes starting with `0xF0`), same problem

## The Problem

When a user writes:

```parsley
let π = 3.14
```

The lexer:
1. Reads byte `0xCF` (first byte of π)
2. Doesn't recognize it as a valid token start
3. Creates `ILLEGAL` token with literal `string(0xCF)` → confusing output

Similarly for emojis outside strings — we recently fixed the error message (commit `0f1be1d`) to detect these corrupted multi-byte characters and show a helpful message instead of garbage.

## Performance Considerations

### Cost of Full Rune-Based Lexer

| Operation | Current (Bytes) | Rune-Based |
|-----------|-----------------|------------|
| `readChar()` | O(1), 1 memory access | O(1), but with branch + possible multi-byte decode |
| `peekChar()` | O(1) | O(1) similar cost |
| `peekCharN(n)` | O(1) | **O(n)** — must decode n runes |

`peekCharN()` is heavily used for multi-character operator detection (e.g., `<=??=>` peeks 5 chars ahead).

### Estimated Impact

| Scenario | Overhead |
|----------|----------|
| ASCII-only input | 5-15% slower lexing |
| Mixed UTF-8 input | 10-20% slower lexing |
| Overall parse time | 2-5% slower (lexing is ~20% of total) |

## Proposed Solution: Hybrid Approach

Keep byte-based operations for operators (all ASCII), but decode runes for identifier detection:

```go
// readChar with hybrid approach
func (l *Lexer) readChar() {
    if l.readPosition >= len(l.input) {
        l.ch = 0
        l.chRune = 0
        l.chSize = 0
        return
    }
    b := l.input[l.readPosition]
    if b < utf8.RuneSelf {
        // Fast path: ASCII (vast majority of code)
        l.ch = b
        l.chRune = rune(b)
        l.chSize = 1
        l.readPosition++
    } else {
        // Slow path: multi-byte UTF-8
        r, size := utf8.DecodeRuneInString(l.input[l.readPosition:])
        l.ch = b  // Keep first byte for operator matching
        l.chRune = r
        l.chSize = size
        l.readPosition += size
    }
    // ... line/column tracking
}
```

### Key Changes

1. Add `chRune rune` and `chSize int` to `Lexer` struct
2. Modify `readChar()` with ASCII fast-path
3. Update `isLetter()` to use `l.chRune` instead of `l.ch`
4. Update `readIdentifier()` to accumulate runes properly
5. Keep `peekChar()`/`peekCharN()` byte-based (operators are ASCII)
6. Update `LexerState` for save/restore

### Performance of Hybrid Approach

**Estimated overhead: 1-3%** for typical Parsley code

- ASCII fast-path: single `< 128` check (branch-predicted)
- Multi-byte slow-path: only triggered for Unicode identifiers
- Operator detection unchanged (byte-based)

## What This Enables

```parsley
// Mathematical constants
let π = 3.14159
let τ = 2 * π
let Δt = t1 - t0

// Non-English identifiers  
let 名前 = "Tanaka"
let счёт = 100

// Greek letters for formulas
fn area(r) = π * r * r
```

## Implementation Estimate

| Task | Estimate |
|------|----------|
| Modify Lexer struct and readChar() | 1-2 hours |
| Update isLetter(), readIdentifier() | 1-2 hours |
| Update LexerState for save/restore | 30 min |
| Fix column tracking for multi-byte | 1 hour |
| Tests for Unicode identifiers | 2 hours |
| Update docs (reference.md, CHEATSHEET.md) | 1 hour |
| **Total** | **~1 day** |

## Risks

1. **Column numbers** — Currently column = byte offset. With runes, need to decide: keep byte offset (matches editors) or use rune count (matches visual position).
2. **Edge cases** — Zero-width joiners, combining characters, emoji sequences. Recommend: follow Go's identifier rules (Unicode letter/digit categories).
3. **`peekCharN()` for non-ASCII** — If future syntax needs multi-byte lookahead, will need redesign. Currently not needed.

## Impact on Production Request Handling

**Zero impact on cached AST execution.**

The lexer only runs during parsing, not request handling:

```
Server startup:
  Load handlers → Lexer → Parser → AST Cache
                    ↑
              Overhead here only

Request flow:
  HTTP Request → Router → Cached AST → Evaluator → Response
                               ↑
                          No lexer here
```

The 1-3% overhead applies only to:
- **Server startup** — Parsing all handlers once
- **Dev mode reload** — When saving a file triggers hot reload
- **CLI execution** — `pars run script.pars`

For a typical Basil app (~20 handlers, ~2000 lines total), initial parse might increase from ~5ms to ~5.1ms. Imperceptible.

**Request latency is completely unaffected** — the evaluator works against the cached AST, never the source text.

## Measuring Performance Impact

To validate the estimated 1-3% overhead, benchmark before and after the refactor.

### Benchmark Suite

Create `pkg/parsley/lexer/lexer_bench_test.go`:

```go
package lexer

import (
    "testing"
)

// Realistic Parsley code samples of varying complexity
var (
    simpleCode = `let x = 1 + 2 * 3`
    
    mediumCode = `
fn greet(name) {
    let message = "Hello, " + name + "!"
    <div class="greeting">
        <h1>{message}</h1>
    </div>
}
`
    
    complexCode = `// Handler with database query
import @basil/http
import @std/table

let db = @sqlite("./app.db")

fn handler(request) {
    let users = db |< @query Users
        | status == "active"
        | age >= 18
        ??->
    
    <html>
        <body>
            <table>
                {for user in users}
                    <tr>
                        <td>{user.name}</td>
                        <td>{user.email}</td>
                    </tr>
                {/for}
            </table>
        </body>
    </html>
}

export handler
`
)

func BenchmarkLexer_Simple(b *testing.B) {
    for i := 0; i < b.N; i++ {
        l := New(simpleCode)
        for tok := l.NextToken(); tok.Type != EOF; tok = l.NextToken() {
        }
    }
}

func BenchmarkLexer_Medium(b *testing.B) {
    for i := 0; i < b.N; i++ {
        l := New(mediumCode)
        for tok := l.NextToken(); tok.Type != EOF; tok = l.NextToken() {
        }
    }
}

func BenchmarkLexer_Complex(b *testing.B) {
    for i := 0; i < b.N; i++ {
        l := New(complexCode)
        for tok := l.NextToken(); tok.Type != EOF; tok = l.NextToken() {
        }
    }
}

// Benchmark with Unicode identifiers (after implementation)
var unicodeCode = `
let π = 3.14159
let τ = 2 * π
let Δx = x2 - x1
let 名前 = "Tanaka"
fn area(r) = π * r * r
`

func BenchmarkLexer_Unicode(b *testing.B) {
    for i := 0; i < b.N; i++ {
        l := New(unicodeCode)
        for tok := l.NextToken(); tok.Type != EOF; tok = l.NextToken() {
        }
    }
}
```

### Running Benchmarks

```bash
# Before refactor - establish baseline
cd pkg/parsley/lexer
go test -bench=BenchmarkLexer -benchmem -count=10 > baseline.txt

# After refactor - measure impact
go test -bench=BenchmarkLexer -benchmem -count=10 > hybrid.txt

# Compare results
go install golang.org/x/perf/cmd/benchstat@latest
benchstat baseline.txt hybrid.txt
```

### Expected Output

```
name              old time/op    new time/op    delta
Lexer_Simple-8      1.23µs ± 2%    1.25µs ± 1%   +1.6%
Lexer_Medium-8      4.56µs ± 1%    4.63µs ± 2%   +1.5%
Lexer_Complex-8     12.3µs ± 2%    12.6µs ± 1%   +2.4%
Lexer_Unicode-8        N/A         5.12µs ± 1%     N/A

name              old alloc/op   new alloc/op   delta
Lexer_Simple-8        0B ± 0%        0B ± 0%    ~
Lexer_Medium-8        0B ± 0%        0B ± 0%    ~
Lexer_Complex-8       0B ± 0%        0B ± 0%    ~
```

### Metrics to Track

| Metric | Target | Notes |
|--------|--------|-------|
| Time/op (ASCII code) | <5% increase | Primary concern |
| Time/op (Unicode code) | N/A baseline | New capability |
| Allocations | No increase | Hybrid approach should be allocation-free |
| Bytes/op | No increase | No new heap allocations |

### Full Parse Benchmark

Also measure end-to-end parsing (lexer + parser):

```go
// In pkg/parsley/parser/parser_bench_test.go
func BenchmarkParse_Complex(b *testing.B) {
    for i := 0; i < b.N; i++ {
        l := lexer.New(complexCode)
        p := New(l)
        p.ParseProgram()
    }
}
```

### Real-World Validation

For final validation, time server startup with realistic handler set:

```bash
# Before
time ./basil --config examples/auth/basil.yaml &
PID=$!; sleep 2; kill $PID

# After (repeat and compare)
```

### Acceptance Criteria

- [ ] ASCII-only benchmarks: <5% regression
- [ ] No increase in allocations
- [ ] Unicode benchmarks: reasonable performance (establishes baseline)
- [ ] Server startup: <10% regression on example apps

## Recommendation

Add to backlog as medium-priority enhancement. The hybrid approach provides:
- Minimal performance impact (1-3%)
- Enables Unicode identifiers
- Preserves O(1) operator detection
- Fixes the misleading `isLetter()` code

## Related Work

- Commit `0f1be1d`: Fixed confusing error message for emojis outside strings
- The `unicode.IsLetter()` call in current `isLetter()` should be removed if we don't implement this — it's misleading dead code
