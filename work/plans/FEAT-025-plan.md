---
id: PLAN-015
feature: FEAT-025
title: "Implementation Plan for Money Type"
status: complete
created: 2025-12-04
updated: 2025-12-05
---

# Implementation Plan: FEAT-025 Money Type

## Overview

Implement a money/currency type with exact arithmetic, type safety, and locale-aware formatting. This adds currency literals (`$12.34`, `EUR#50.00`), arithmetic operators, and methods (`.format()`, `.abs()`, `.split()`).

## Time Estimate

**Total: 4-6 hours**

| Phase | Estimate | Cumulative |
|-------|----------|------------|
| Task 1: Token types | 30 min | 30 min |
| Task 2: AST node | 15 min | 45 min |
| Task 3: Lexer | 45 min | 1h 30m |
| Task 4: Parser | 30 min | 2h |
| Task 5: Money object | 30 min | 2h 30m |
| Task 6: Arithmetic | 45 min | 3h 15m |
| Task 7: Comparison | 20 min | 3h 35m |
| Task 8: Methods | 45 min | 4h 20m |
| Task 9: money() function | 20 min | 4h 40m |
| Task 10: Integration tests | 30 min | 5h 10m |
| Buffer for edge cases | 30-50 min | ~6h |

## Prerequisites

- [x] Design decisions finalized (see FEAT-025.md)
- [ ] Add `golang.org/x/text` dependency (for currency formatting)

## Tasks

### Task 1: Add Token Types
**Files**: `pkg/parsley/token/token.go`
**Estimated effort**: Small (30 min)

Steps:
1. Add `MONEY` token type for parsed money literals
2. Add currency symbol tokens: `DOLLAR`, `POUND`, `EURO`, `YEN`
3. Add `CURRENCY_CODE` token for `USD#`, `GBP#`, etc.

Tests:
- Token string representations

---

### Task 2: Add AST Node
**Files**: `pkg/parsley/ast/ast.go`
**Estimated effort**: Small (15 min)

Steps:
1. Add `MoneyLiteral` struct:
   ```go
   type MoneyLiteral struct {
       Token    token.Token
       Currency string  // "USD", "GBP", etc.
       Amount   int64   // stored as smallest unit (cents)
       Scale    int8    // decimal places
   }
   ```
2. Implement `expressionNode()`, `TokenLiteral()`, `String()`

Tests:
- AST string representation

---

### Task 3: Lexer — Currency Symbols and CODE# Syntax
**Files**: `pkg/parsley/lexer/lexer.go`
**Estimated effort**: Medium (45 min)

Steps:
1. Recognize currency symbols: `$`, `£`, `€`, `¥`
2. Recognize compound symbols: `CA$`, `AU$`, `HK$`, `S$`, `CN¥`
3. Recognize `CODE#` pattern (3 uppercase letters + `#`)
4. After symbol, read the number (including decimals)
5. Validate: known currencies (JPY) can't have decimals

Symbol → Currency mapping:
| Symbol | Currency |
|--------|----------|
| `$` | USD |
| `CA$` | CAD |
| `AU$` | AUD |
| `HK$` | HKD |
| `S$` | SGD |
| `£` | GBP |
| `€` | EUR |
| `¥` | JPY |
| `CN¥` | CNY |

Tests:
- `$12.34` → MONEY token with USD, 1234, scale=2
- `£99.99` → MONEY token with GBP, 9999, scale=2
- `EUR#50.00` → MONEY token with EUR, 5000, scale=2
- `JPY#1000` → MONEY token with JPY, 1000, scale=0
- `BTC#0.00001234` → MONEY token with BTC, 1234, scale=8
- `CA$25.00` → MONEY token with CAD, 2500, scale=2
- Error: `¥100.50` (JPY has no decimals)

---

### Task 4: Parser — Money Literals
**Files**: `pkg/parsley/parser/parser.go`
**Estimated effort**: Small (30 min)

Steps:
1. Register prefix parser for `MONEY` token
2. Create `MoneyLiteral` AST node from token data

Tests:
- Parse `$12.34` → MoneyLiteral{Currency: "USD", Amount: 1234, Scale: 2}
- Parse `EUR#100` → MoneyLiteral{Currency: "EUR", Amount: 100, Scale: 0}

---

### Task 5: Money Object Type
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small (30 min)

Steps:
1. Add `Money` object type:
   ```go
   type Money struct {
       Amount   int64
       Currency string
       Scale    int8
   }
   ```
2. Implement `Type()` → `"money"`
3. Implement `Inspect()` → `"$12.34"` or `"EUR#50.00"`
4. Evaluate `MoneyLiteral` → `Money` object

Tests:
- `$12.34` evaluates to Money{1234, "USD", 2}
- Money.Inspect() returns appropriate string

---

### Task 6: Arithmetic Operators
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Medium (45 min)

Steps:
1. `+` / `-` between Money: same currency only, error otherwise
2. `*` / `/` with scalar (integer/float): allowed
3. `/` uses banker's rounding
4. Handle scale promotion on arithmetic
5. Unary `-` for negation

Arithmetic rules:
- `Money + Money` → same currency, promote scale
- `Money - Money` → same currency, promote scale  
- `Money * scalar` → allowed
- `scalar * Money` → allowed
- `Money / scalar` → allowed, banker's rounding
- `Money + integer` → ERROR
- `Money + float` → ERROR

Tests:
- `$10 + $5` → `$15`
- `$20 - $8` → `$12`
- `$10 * 3` → `$30`
- `$15 / 3` → `$5`
- `$17 / 3` → `$5.67` (banker's rounding)
- `-$50` → Money with negative amount
- `$10 + £5` → runtime error
- `$10 + 5` → runtime error
- `USD#1.00 + USD#0.001` → `USD#1.001` (scale promotion)

---

### Task 7: Comparison Operators
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small (20 min)

Steps:
1. Implement `>`, `<`, `>=`, `<=`, `==`, `!=` for Money
2. Same currency only; error otherwise

Tests:
- `$10 > $5` → `true`
- `$10 == $10` → `true`
- `$10 < £5` → runtime error

---

### Task 8: Money Methods
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Medium (45 min)

Steps:
1. `.currency` property → string
2. `.amount` property → integer (raw amount in smallest unit)
3. `.format()` method → locale-aware string
4. `.format(locale)` method → locale-specific formatting
5. `.abs()` method → absolute value
6. `.split(n)` method → array of Money

For `.format()`:
- Use `golang.org/x/text/currency` for known currencies (154 ISO 4217)
- Fall back to `"CODE amount"` for unknown currencies (BTC, custom)

Tests:
- `$10.50.currency` → `"USD"`
- `$10.50.amount` → `1050`
- `$1234.56.format()` → `"$1,234.56"`
- `$1234.56.format("de-DE")` → `"1.234,56 $"`
- `(-$50).abs()` → `$50`
- `$100.split(3)` → `[$33.34, $33.33, $33.33]`
- `sum($100.split(3))` → `$100` (exact)

---

### Task 9: money() Constructor Function
**Files**: `pkg/parsley/evaluator/builtins.go`
**Estimated effort**: Small (20 min)

Steps:
1. Add `money(amount, currency)` for known currencies
2. Add `money(amount, currency, scale)` for unknown currencies
3. Infer scale from currency if known (USD=2, JPY=0, etc.)

Tests:
- `money(1234, "USD")` → `$12.34`
- `money(1000, "JPY")` → `¥1000`
- `money(100000000, "BTC", 8)` → `BTC#1.00000000`

---

### Task 10: Integration Tests
**Files**: `pkg/parsley/tests/money_test.go`
**Estimated effort**: Small (30 min)

Steps:
1. Create comprehensive test file
2. Test all literals, operators, methods
3. Test error cases
4. Test array operations (sum, sort, filter, map)

Tests:
- Full integration tests covering spec examples
- Error message quality tests

---

## Validation Checklist
- [x] All tests pass: `make test`
- [x] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] Documentation updated (reference.md, CHEATSHEET.md)
- [ ] BACKLOG.md updated with deferrals (if any)

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-12-05 | Task 1: Token types | Complete | Added MONEY token to lexer |
| 2025-12-05 | Task 2: AST node | Complete | Added MoneyLiteral struct |
| 2025-12-05 | Task 3: Lexer | Complete | Currency symbols + CODE# syntax |
| 2025-12-05 | Task 4: Parser | Complete | MONEY prefix parser |
| 2025-12-05 | Task 5: Money object | Complete | Money struct in evaluator |
| 2025-12-05 | Task 6: Arithmetic | Complete | +, -, *, / with scale promotion |
| 2025-12-05 | Task 7: Comparison | Complete | ==, !=, <, >, <=, >= |
| 2025-12-05 | Task 8: Methods | Complete | format(), abs(), split() |
| 2025-12-05 | Task 9: money() function | Complete | money(amount, currency[, scale]) |
| 2025-12-05 | Task 10: Integration tests | Complete | pkg/parsley/tests/money_test.go |

## Deferred Items
Items to add to BACKLOG.md after implementation:
- Currency conversion API hooks
- Strict ISO-only mode (lint rule?)
- Unit types using `#` syntax (e.g., `12.34#mm`)
- Precise decimals (`#12.34` without currency)
- `.toFloat()` method (intentionally omitted — use `.amount / 100`)
- `.toDict()` method (not needed for MVP)
- Convenience methods on numbers (`.usd()`, `.gbp()`) — use `money()` instead

## Implementation Notes

### Banker's Rounding
Round half to even (standard for finance):
- 2.5 → 2
- 3.5 → 4
- 2.25 → 2.2
- 2.35 → 2.4

### Scale Promotion
When two Money values with different scales are combined:
```go
// USD#1.00 (scale=2) + USD#0.001 (scale=3)
// Promote USD#1.00 to scale=3: amount 1000
// Add: 1000 + 1 = 1001
// Result: USD#1.001 (scale=3)
```

### Symbol Lexing Strategy
1. When seeing `$`, check next chars for compound (`CA$`, `AU$`, `HK$`, `S$`)
2. When seeing `¥`, check for `CN¥`
3. Otherwise single-char symbol
4. When seeing uppercase letter, check for `XXX#` pattern
5. After symbol/code, read number and calculate scale from decimal places

### Currency Code Validation
Accept any 3-letter uppercase code (not just ISO 4217):
- Enables BTC, ETH, custom loyalty points
- Mixing errors catch typos quickly (`USd#10 + USD#5` → error)
- Can add strict mode later if needed
