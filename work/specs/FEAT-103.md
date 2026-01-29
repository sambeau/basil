---
id: FEAT-103
title: "Unicode Identifier Support"
status: draft
priority: medium
created: 2026-01-29
author: "@human"
---

# FEAT-103: Unicode Identifier Support

## Summary

Enable Unicode characters in Parsley identifiers, allowing mathematical symbols (π, τ, Δ), non-Latin scripts (日本語, кириллица), and other Unicode letters. This aligns Parsley with its UTF-8 foundation and internationalization goals while maintaining performance through a hybrid byte/rune lexer approach.

## Motivation

Parsley claims to be:
- **UTF-8 native** — but identifiers are currently ASCII-only
- **Good at localization/i18n** — but variable names must be English
- **Fun to work with** — mathematical notation like `let π = 3.14` is more expressive

The current lexer has broken dead code that *appears* to support Unicode (`unicode.IsLetter()` in `isLetter()`), which is misleading. We should either fix it or remove it.

## User Story

As a Parsley developer, I want to use Unicode characters in variable and function names so that I can write more expressive mathematical code and use identifiers in my native language.

## Examples

```parsley
// Mathematical constants
let π = 3.14159
let τ = 2 * π
let Δt = t1 - t0

// Non-English identifiers
let 名前 = "Tanaka"
let счёт = 100
let prénom = "Marie"

// Greek letters for formulas
fn area(r) = π * r * r
fn distance(Δx, Δy) = sqrt(Δx*Δx + Δy*Δy)
```

## Acceptance Criteria

### Functional
- [ ] Unicode letters valid at identifier start (per Go's `unicode.IsLetter`)
- [ ] Unicode letters and digits valid in identifier body
- [ ] All existing ASCII identifiers continue to work
- [ ] Keywords remain ASCII-only (no `если` for `if`)
- [ ] Error messages show actual Unicode character, not corrupted bytes

### Performance (CRITICAL)
- [ ] Benchmark suite created and baseline established BEFORE implementation
- [ ] ASCII-only code: <5% lexer overhead vs baseline
- [ ] No increase in memory allocations
- [ ] Server startup: <10% regression on example apps
- [ ] If performance targets not met, feature is rejected

### Documentation
- [ ] Update `docs/parsley/reference.md` identifier grammar
- [ ] Update `docs/parsley/CHEATSHEET.md` with examples
- [ ] Add note about Unicode identifier support

## Design Decisions

- **Hybrid byte/rune approach**: Keep byte-based scanning for operators (all ASCII), decode runes only for identifier detection. This preserves O(1) operator lookahead while enabling Unicode identifiers.

- **Follow Go's identifier rules**: Use `unicode.IsLetter()` and `unicode.IsDigit()` categories, matching Go's identifier semantics. No special handling for emoji, ZWJ sequences, or combining characters.

- **Column numbers remain byte offsets**: Editors use byte offsets, so column numbers in error messages stay byte-based for accurate jump-to-location.

- **Performance-gated merge**: Implementation happens on a feature branch. If benchmarks show >5% regression on ASCII code, the branch is rejected and we revisit the approach.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components

- `pkg/parsley/lexer/lexer.go` — Core changes:
  - Add `chRune rune` and `chSize int` to `Lexer` struct
  - Modify `readChar()` with ASCII fast-path
  - Update `isLetter()` to use `chRune`
  - Update `readIdentifier()` for multi-byte accumulation
  - Update `LexerState` for save/restore

- `pkg/parsley/lexer/lexer_bench_test.go` — New benchmark suite

- `pkg/parsley/parser/parser.go` — No changes expected (receives tokens)

- `docs/parsley/reference.md` — Update identifier grammar

- `docs/parsley/CHEATSHEET.md` — Add Unicode examples

### Implementation Approach

```go
// Hybrid readChar - ASCII fast-path, rune decode for non-ASCII
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

// isLetter now checks the full rune
func (l *Lexer) isLetterRune() bool {
    return unicode.IsLetter(l.chRune) || l.chRune == '_'
}
```

### Workflow

1. **Create benchmark suite** — Establish performance baseline
2. **Create feature branch** — `feat/FEAT-103-unicode-identifiers`
3. **Implement hybrid lexer** — On feature branch
4. **Run benchmarks** — Compare against baseline
5. **Decision gate**:
   - If <5% regression: merge to main
   - If >5% regression: analyze, optimize, or reject
6. **Update documentation** — After successful merge

### Edge Cases & Constraints

1. **Malformed UTF-8** — `utf8.DecodeRuneInString` returns `RuneError` (U+FFFD). Treat as illegal character with clear error message.

2. **Combining characters** — `e` + `́` (combining acute) vs `é` (precomposed). Go's `unicode.IsLetter` handles both; identifiers may look identical but be different bytes. Document this as user's responsibility (use NFC normalization if needed).

3. **RTL scripts** — Hebrew, Arabic identifiers will work but may display oddly in editors. Not our problem to solve.

4. **Emoji** — Not letters per Unicode, so `let 🚀 = 1` remains invalid. Error message explains this (already implemented in commit `0f1be1d`).

5. **peekCharN() for non-ASCII** — Remains byte-based. If future syntax needs rune lookahead, will need redesign. Currently all multi-char operators are ASCII.

### Performance Analysis

See [work/reports/LEXER-UNICODE-ANALYSIS.md](../reports/LEXER-UNICODE-ANALYSIS.md) for detailed analysis.

**Key points:**
- Lexer overhead: estimated 1-3% for ASCII code
- Request handling: zero impact (AST is cached)
- Server startup: imperceptible (~0.1ms on typical app)

### Estimated Effort

| Task | Estimate |
|------|----------|
| Create benchmark suite | 1 hour |
| Establish baseline | 30 min |
| Modify Lexer struct and readChar() | 1-2 hours |
| Update isLetter(), readIdentifier() | 1-2 hours |
| Update LexerState for save/restore | 30 min |
| Fix column tracking for multi-byte | 1 hour |
| Tests for Unicode identifiers | 2 hours |
| Run benchmarks, analyze | 1 hour |
| Update docs | 1 hour |
| **Total** | **~1 day** |

## Related

- Plan: `work/plans/FEAT-103-plan.md`
- Analysis: `work/reports/LEXER-UNICODE-ANALYSIS.md`
- Backlog: #88 (Unicode identifier support)
- Commit `0f1be1d`: Fixed emoji error messages (preparatory work)
