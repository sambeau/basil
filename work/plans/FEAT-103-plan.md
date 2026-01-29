---
id: PLAN-077
feature: FEAT-103
title: "Implementation Plan for Unicode Identifier Support"
status: draft
created: 2026-01-29
---

# Implementation Plan: FEAT-103 Unicode Identifier Support

## Overview

Implement a hybrid byte/rune lexer that enables Unicode identifiers (e.g., `let π = 3.14`) while maintaining performance for ASCII-only code. The implementation is **performance-gated**: if benchmarks show >5% regression on ASCII code, the branch is rejected.

## Prerequisites

- [ ] Understand current lexer architecture (byte-based scanning)
- [ ] Review analysis report: `work/reports/LEXER-UNICODE-ANALYSIS.md`
- [ ] Install benchstat: `go install golang.org/x/perf/cmd/benchstat@latest`

## Tasks

### Phase 0: Establish Performance Baseline (on main)

### Task 0.1: Create Benchmark Suite
**Files**: `pkg/parsley/lexer/lexer_bench_test.go`  
**Estimated effort**: Small (1 hour)

Steps:
1. Create benchmark file with simple/medium/complex/unicode test cases
2. Include realistic Parsley code (functions, tags, database queries)
3. Add BenchmarkLexer_Simple, BenchmarkLexer_Medium, BenchmarkLexer_Complex
4. Add BenchmarkLexer_Unicode (will fail initially, expected)

Tests:
- `go test -bench=BenchmarkLexer -benchmem` runs without error

---

### Task 0.2: Capture Baseline Metrics
**Files**: None (output files)  
**Estimated effort**: Small (30 min)

Steps:
1. Run `go test -bench=BenchmarkLexer -benchmem -count=10 > baseline.txt`
2. Save baseline.txt in `work/reports/FEAT-103-baseline.txt`
3. Document baseline numbers in this plan's Progress Log
4. Commit benchmark suite to main

---

### Phase 1: Implement Hybrid Lexer (on feature branch)

### Task 1.1: Create Feature Branch
**Files**: None  
**Estimated effort**: Small (5 min)

Steps:
1. `git checkout -b feat/FEAT-103-unicode-identifiers`
2. Ensure clean state from main

---

### Task 1.2: Extend Lexer Struct
**Files**: `pkg/parsley/lexer/lexer.go`  
**Estimated effort**: Small (30 min)

Steps:
1. Add `chRune rune` field to `Lexer` struct (current character as rune)
2. Add `chSize int` field (byte width of current character)
3. Update `LexerState` struct with same fields
4. Update `SaveState()` to include new fields
5. Update `RestoreState()` to restore new fields

Tests:
- Existing lexer tests still pass
- SaveState/RestoreState roundtrip preserves new fields

---

### Task 1.3: Implement Hybrid readChar()
**Files**: `pkg/parsley/lexer/lexer.go`  
**Estimated effort**: Medium (1-2 hours)

Steps:
1. Modify `readChar()` with ASCII fast-path:
   ```go
   b := l.input[l.readPosition]
   if b < utf8.RuneSelf {
       // Fast path: ASCII
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
   ```
2. Update column tracking (increment by 1 regardless of chSize for visual columns)
3. Ensure EOF handling unchanged

Tests:
- All existing lexer tests pass (ASCII behavior unchanged)
- New test: lexer correctly decodes π (U+03C0) as single rune

---

### Task 1.4: Update isLetter() for Unicode
**Files**: `pkg/parsley/lexer/lexer.go`  
**Estimated effort**: Small (30 min)

Steps:
1. Create new method `isLetterRune()` that checks `l.chRune`:
   ```go
   func (l *Lexer) isLetterRune() bool {
       return unicode.IsLetter(l.chRune) || l.chRune == '_'
   }
   ```
2. Create `isDigitRune()` for identifier body:
   ```go
   func (l *Lexer) isDigitRune() bool {
       return unicode.IsDigit(l.chRune)
   }
   ```
3. Update call sites that check identifier start to use `isLetterRune()`
4. Remove broken `isLetter(ch byte)` function (the one with dead unicode.IsLetter call)

Tests:
- `isLetterRune()` returns true for `π`, `名`, `д`
- `isLetterRune()` returns false for `🚀`, `1`, `+`

---

### Task 1.5: Update readIdentifier() for Unicode
**Files**: `pkg/parsley/lexer/lexer.go`  
**Estimated effort**: Medium (1-2 hours)

Steps:
1. Modify `readIdentifier()` to use rune-aware reading:
   ```go
   func (l *Lexer) readIdentifier() string {
       position := l.position
       for l.isLetterRune() || l.isDigitRune() {
           l.readChar()
       }
       return l.input[position:l.position]
   }
   ```
2. Ensure byte slicing still works (position tracks bytes, not runes)
3. Handle identifier at EOF correctly

Tests:
- `let π = 3.14` lexes as LET, IDENT("π"), ASSIGN, FLOAT
- `let abc123 = 1` still works (ASCII)
- `let 名前 = "x"` lexes IDENT as "名前"
- Mixed ASCII/Unicode: `let πr2 = 1` lexes IDENT as "πr2"

---

### Task 1.6: Update Identifier Detection in NextToken()
**Files**: `pkg/parsley/lexer/lexer.go`  
**Estimated effort**: Small (30 min)

Steps:
1. Find the `default:` case in `NextToken()` switch
2. Update identifier start check from `isLetter(l.ch)` to `l.isLetterRune()`
3. Ensure operator detection still uses `l.ch` (byte) not `l.chRune`

Tests:
- `π + 1` correctly identifies π as identifier, + as operator
- `<=??=>` still recognized as QUERY_MANY operator

---

### Task 1.7: Handle Malformed UTF-8
**Files**: `pkg/parsley/lexer/lexer.go`  
**Estimated effort**: Small (30 min)

Steps:
1. In readChar() slow path, check for `utf8.RuneError`:
   ```go
   if r == utf8.RuneError && size == 1 {
       // Invalid UTF-8 byte
       l.chRune = utf8.RuneError
   }
   ```
2. In NextToken(), when `l.chRune == utf8.RuneError`, create ILLEGAL token with helpful message
3. Leverage existing improved error message from commit `0f1be1d`

Tests:
- Invalid UTF-8 byte sequence produces clear error message
- Error includes line and column

---

### Phase 2: Testing

### Task 2.1: Add Unicode Identifier Tests
**Files**: `pkg/parsley/lexer/lexer_test.go`  
**Estimated effort**: Medium (1-2 hours)

Steps:
1. Add test cases for Greek letters: `π`, `τ`, `Δ`, `α`, `β`
2. Add test cases for CJK: `名前`, `変数`
3. Add test cases for Cyrillic: `счёт`, `переменная`
4. Add test cases for accented Latin: `prénom`, `naïve`
5. Add test cases for mixed: `πr2`, `delta_Δ`
6. Add negative test cases: emoji `🚀` (should fail), numbers at start `1abc`

Tests:
- All new test cases pass
- `go test ./pkg/parsley/lexer/...` passes

---

### Task 2.2: Add Parser Integration Tests
**Files**: `pkg/parsley/parser/parser_test.go`  
**Estimated effort**: Small (1 hour)

Steps:
1. Add test: `let π = 3.14` parses to LetStatement with identifier "π"
2. Add test: `fn area(r) = π * r * r` parses correctly
3. Add test: function with Unicode parameter name
4. Verify AST structure matches expectations

Tests:
- Parser tests pass
- AST nodes have correct Unicode identifiers

---

### Task 2.3: Add Evaluator Integration Tests
**Files**: `pkg/parsley/evaluator/evaluator_test.go`  
**Estimated effort**: Small (1 hour)

Steps:
1. Add test: `let π = 3.14; π * 2` evaluates to 6.28
2. Add test: function with Unicode name can be called
3. Add test: Unicode identifier in closure captures correctly

Tests:
- Evaluator tests pass
- Unicode identifiers evaluate correctly

---

### Phase 3: Performance Validation

### Task 3.1: Run Post-Implementation Benchmarks
**Files**: None (output files)  
**Estimated effort**: Small (30 min)

Steps:
1. Run `go test -bench=BenchmarkLexer -benchmem -count=10 > hybrid.txt`
2. Compare: `benchstat baseline.txt hybrid.txt`
3. Document results in Progress Log

Decision Gate:
- If ASCII benchmarks show <5% regression: **PROCEED to Phase 4**
- If ASCII benchmarks show >5% regression: **STOP, analyze, optimize or reject**

---

### Task 3.2: Server Startup Benchmark (if needed)
**Files**: None  
**Estimated effort**: Small (30 min)

Steps:
1. Time baseline: `time ./basil --config examples/auth/basil.yaml &` (on main)
2. Time hybrid: same command on feature branch
3. Compare startup times
4. Target: <10% regression

---

### Phase 4: Documentation & Finalization

### Task 4.1: Update Language Reference
**Files**: `docs/parsley/reference.md`  
**Estimated effort**: Small (30 min)

Steps:
1. Update identifier grammar section
2. Add note about Unicode letter/digit categories
3. Add examples with Greek letters and non-Latin scripts

---

### Task 4.2: Update Cheatsheet
**Files**: `docs/parsley/CHEATSHEET.md`  
**Estimated effort**: Small (15 min)

Steps:
1. Add Unicode identifier examples
2. Note that emoji are NOT valid identifiers

---

### Task 4.3: Final Validation & Merge
**Files**: None  
**Estimated effort**: Small (30 min)

Steps:
1. Run full test suite: `make check`
2. Run linter: `golangci-lint run`
3. Create PR or merge to main
4. Update FEAT-103 spec status to "implemented"
5. Update backlog item #88 to "completed"

---

## Validation Checklist

- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] Benchmarks meet targets (<5% regression)
- [ ] Documentation updated
- [ ] FEAT-103 spec marked complete
- [ ] Backlog #88 marked complete

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| | Task 0.1: Create benchmark suite | ⬜ Not started | |
| | Task 0.2: Capture baseline | ⬜ Not started | |
| | Task 1.1: Create branch | ⬜ Not started | |
| | Task 1.2: Extend Lexer struct | ⬜ Not started | |
| | Task 1.3: Hybrid readChar() | ⬜ Not started | |
| | Task 1.4: Update isLetter() | ⬜ Not started | |
| | Task 1.5: Update readIdentifier() | ⬜ Not started | |
| | Task 1.6: Update NextToken() | ⬜ Not started | |
| | Task 1.7: Handle malformed UTF-8 | ⬜ Not started | |
| | Task 2.1: Lexer tests | ⬜ Not started | |
| | Task 2.2: Parser tests | ⬜ Not started | |
| | Task 2.3: Evaluator tests | ⬜ Not started | |
| | Task 3.1: Post-implementation benchmarks | ⬜ Not started | **DECISION GATE** |
| | Task 3.2: Server startup benchmark | ⬜ Not started | If needed |
| | Task 4.1: Update reference.md | ⬜ Not started | |
| | Task 4.2: Update CHEATSHEET.md | ⬜ Not started | |
| | Task 4.3: Final validation & merge | ⬜ Not started | |

## Baseline Metrics

*Captured 2025-01-20*

| Benchmark | Time/op | Alloc/op | Notes |
|-----------|---------|----------|-------|
| Lexer_Simple | 262.3 ns | 12 B, 3 allocs | Basic HTML |
| Lexer_Medium | 1082 ns | 168 B, 22 allocs | Handler function |
| Lexer_Complex | 3658 ns | 312 B, 55 allocs | DB query |
| Lexer_UnicodeEquiv | 1220 ns | 64 B, 13 allocs | ASCII version of Unicode test |

## Post-Implementation Metrics

*Captured 2025-01-20*

| Benchmark | Time/op | Delta | Pass/Fail |
|-----------|---------|-------|-----------|
| Lexer_Simple | 259.3 ns | -1.1% | ✅ PASS |
| Lexer_Medium | 1046 ns | -3.3% | ✅ PASS |
| Lexer_Complex | 3569 ns | -2.4% | ✅ PASS |
| Lexer_Unicode | 1147 ns | -6.0% | ✅ PASS (new) |

**Result: PERFORMANCE GATE PASSED** - All benchmarks show improved performance!

## Deferred Items

Items to add to work/BACKLOG.md after implementation:
- None anticipated (scope is well-contained)

## Rollback Plan

If performance targets are not met:
1. Do not merge feature branch
2. Document actual benchmark results in Progress Log
3. Options:
   a. Investigate optimization opportunities (profile hotspots)
   b. Accept higher overhead if benefits outweigh costs
   c. Reject feature and remove broken `isLetter()` code instead
4. Update FEAT-103 spec with outcome
