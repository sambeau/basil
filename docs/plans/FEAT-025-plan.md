---
id: PLAN-015
feature: FEAT-025
title: "Implementation Plan for Money Type"
status: draft
created: 2025-12-04
---

# Implementation Plan: FEAT-025 Money Type

## Overview

Implement a money/currency type using the `#` precise number system. This includes:
- Lexer support for currency symbols (`$`, `£`, `€`, etc.) and `CODE#` syntax
- Parser for money literals
- Money object type with integer-based storage
- Arithmetic operators (same-currency only)
- Methods (`.format()`, `.toFloat()`, `.toDict()`)
- Builtin `money()` function

## Prerequisites

- [x] FEAT-025 spec approved
- [ ] Decide on Go decimal library (shopspring/decimal vs custom int64)

## Tasks

### Task 1: Define Money Object Type
**Files**: `pkg/parsley/object/object.go`
**Estimated effort**: Small

Steps:
1. Add `Money` struct with `Amount int64`, `Currency string`, `Scale int8`
2. Implement `Object` interface: `Type()`, `Inspect()`
3. Add `MONEY` constant to object types
4. Implement `Hashable` interface for use as dictionary keys

Tests:
- Money object creation
- Inspect() output format
- Type() returns "MONEY"

---

### Task 2: Add Token Types
**Files**: `pkg/parsley/token/token.go`
**Estimated effort**: Small

Steps:
1. Add `MONEY_LITERAL` token type
2. Add `CURRENCY_SYMBOL` token type (for `$`, `£`, `€`, etc.)
3. Document token patterns

Tests:
- Token type constants exist

---

### Task 3: Lexer - Currency Symbol Shortcuts
**Files**: `pkg/parsley/lexer/lexer.go`
**Estimated effort**: Medium

Steps:
1. Add detection for currency symbols: `$`, `£`, `€`, `¥`
2. Add detection for prefixed symbols: `CA$`, `AU$`, `HK$`, `S$`, `CN¥`
3. Map symbols to ISO codes: `$` → `USD`, `£` → `GBP`, etc.
4. Read number after symbol
5. Return `MONEY_LITERAL` token with currency and amount

Tests:
- `$12.34` → token with USD, 12.34
- `£99.99` → token with GBP, 99.99
- `€50` → token with EUR, 50.00
- `CA$25.00` → token with CAD, 25.00
- `¥1000` → token with JPY, 1000 (no decimals)

---

### Task 4: Lexer - CODE# Syntax
**Files**: `pkg/parsley/lexer/lexer.go`
**Estimated effort**: Medium

Steps:
1. Detect uppercase letters followed by `#`
2. Read currency code (2-3 uppercase letters)
3. Read `#` separator
4. Read number
5. Validate currency code against ISO 4217 list
6. Return `MONEY_LITERAL` token

Tests:
- `USD#12.34` → token with USD, 12.34
- `CHF#100.00` → token with CHF, 100.00
- `JPY#1000` → token with JPY, 1000
- `XXX#10` → error (invalid currency)
- `usd#10` → error (must be uppercase)

---

### Task 5: Parser - Money Literals
**Files**: `pkg/parsley/ast/ast.go`, `pkg/parsley/parser/parser.go`
**Estimated effort**: Small

Steps:
1. Add `MoneyLiteral` AST node with `Currency`, `Amount`, `Scale` fields
2. Register prefix parser for `MONEY_LITERAL`
3. Parse token into MoneyLiteral node

Tests:
- Parse `$12.34` into MoneyLiteral
- Parse `CHF#100` into MoneyLiteral
- MoneyLiteral.String() returns canonical form

---

### Task 6: Evaluator - Money Literal Evaluation
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small

Steps:
1. Add case for `*ast.MoneyLiteral` in `Eval()`
2. Convert to `*object.Money` with integer amount
3. Handle scale (JPY=0, most=2, KWD=3)

Tests:
- `$12.34` evaluates to Money{Amount: 1234, Currency: "USD", Scale: 2}
- `¥1000` evaluates to Money{Amount: 1000, Currency: "JPY", Scale: 0}

---

### Task 7: Evaluator - Money Arithmetic
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Medium

Steps:
1. Add money cases to `evalInfixExpression()`
2. Implement `+` for same-currency: add amounts
3. Implement `-` for same-currency: subtract amounts
4. Implement `*` with integer/float: scale amount (error if result loses precision)
5. Implement `/` with integer/float: divide with banker's rounding
6. Error on different currencies
7. Error on money + raw number (no implicit conversion)

Tests:
- `$10 + $5` → `$15`
- `$20 - $8` → `$12`
- `$10 * 3` → `$30`
- `$15 / 3` → `$5`
- `$17 / 3` → `$5.67` (banker's rounding)
- `$10 + £5` → error
- `$10 + 5` → error
- `$10 + 5.0` → error

---

### Task 8: Evaluator - Money Comparison
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Small

Steps:
1. Add money cases to comparison operators
2. `==`, `!=`: compare currency and amount
3. `<`, `>`, `<=`, `>=`: compare amount (same currency only)
4. Error on different currencies

Tests:
- `$10 == $10` → true
- `$10 == USD#10` → true
- `$10 > $5` → true
- `$10 < £5` → error

---

### Task 9: Money Methods
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Medium

Steps:
1. Add `Money` case to method dispatch
2. `.amount` property → integer amount
3. `.currency` property → currency code string
4. `.scale` property → scale integer
5. `.format()` method → locale-aware formatting (default locale)
6. `.format(locale)` method → specified locale formatting
7. `.toFloat()` method → convert to float (explicit)
8. `.toDict()` method → dictionary representation
9. `.abs()` method → absolute value

Tests:
- `$12.34.amount` → 1234
- `$12.34.currency` → "USD"
- `$12.34.scale` → 2
- `$1234.56.format()` → "$1,234.56"
- `$1234.56.format("de-DE")` → "1.234,56 $"
- `$12.34.toFloat()` → 12.34
- `(-$50).abs()` → $50

---

### Task 10: Builtin money() Function
**Files**: `pkg/parsley/evaluator/builtins.go`
**Estimated effort**: Small

Steps:
1. Add `money` builtin function
2. Signature: `money(amount, currency)` or `money(amount, currency, scale)`
3. Validate currency code
4. Convert float amount to integer with scale
5. Return Money object

Tests:
- `money(12.34, "USD")` → $12.34
- `money(1000, "JPY")` → ¥1000
- `money(12.34, "XXX")` → error

---

### Task 11: Convenience Methods on Numbers
**Files**: `pkg/parsley/evaluator/methods.go`
**Estimated effort**: Small

Steps:
1. Add `.usd()`, `.gbp()`, `.eur()` methods to Float/Integer
2. Each returns Money object with appropriate currency

Tests:
- `(12.34).usd()` → $12.34
- `(99.99).gbp()` → £99.99
- `(50).eur()` → €50.00

---

### Task 12: Currency Data
**Files**: `pkg/parsley/evaluator/currency.go` (new file)
**Estimated effort**: Small

Steps:
1. Create currency data file
2. Define ISO 4217 currency codes with scales
3. Define symbol → code mappings
4. Add validation functions

Data:
```go
var currencyScales = map[string]int8{
    "USD": 2, "EUR": 2, "GBP": 2, "JPY": 0, "CHF": 2,
    "CAD": 2, "AUD": 2, "CNY": 2, "HKD": 2, "SGD": 2,
    "KRW": 0, "INR": 2, "BRL": 2, "KWD": 3, // etc.
}

var symbolToCurrency = map[string]string{
    "$": "USD", "£": "GBP", "€": "EUR", "¥": "JPY",
    "CA$": "CAD", "AU$": "AUD", "HK$": "HKD", "S$": "SGD", "CN¥": "CNY",
}
```

Tests:
- Valid currency lookup
- Invalid currency error
- Symbol mapping

---

### Task 13: String Representation for print/interpolation
**Files**: `pkg/parsley/evaluator/evaluator.go` or `object/object.go`
**Estimated effort**: Small

Steps:
1. Money.Inspect() returns formatted string (e.g., "$12.34")
2. Ensure works with print() and string interpolation

Tests:
- `"{$12.34}"` → "$12.34"
- `print($99.99)` outputs "$99.99"

---

### Task 14: Error Messages
**Files**: `pkg/parsley/errors/errors.go`
**Estimated effort**: Small

Steps:
1. Add `MONEY-0001`: "cannot mix currencies: {left} and {right}"
2. Add `MONEY-0002`: "cannot perform arithmetic between money and {type}"
3. Add `MONEY-0003`: "unknown currency code: {code}"
4. Add `MONEY-0004`: "invalid decimal places for {currency}: expected {expected}, got {actual}"

Tests:
- Error messages include proper codes and details

---

## Validation Checklist

- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] Documentation updated (reference.md, CHEATSHEET.md)
- [ ] BACKLOG.md updated with deferrals

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| — | — | — | — |

## Deferred Items

Items to add to BACKLOG.md after implementation:
- **Units (`#12.34mm`)** — Separate FEAT, extends the `#` system
- **Rationals (`#22/7`)** — Separate FEAT
- **Bare decimals (`#12.34`)** — May implement with money or defer
- **Currency conversion hooks** — Out of scope for MVP
- **Percentage operations (`$100 * 15%`)** — Needs design

## Implementation Order

Recommended order for incremental progress:

1. **Foundation**: Tasks 1, 2, 12 (types, tokens, currency data)
2. **Lexing**: Tasks 3, 4 (symbol shortcuts, CODE# syntax)
3. **Parsing**: Task 5 (AST and parser)
4. **Evaluation**: Tasks 6, 13 (literal eval, string repr)
5. **Operators**: Tasks 7, 8 (arithmetic, comparison)
6. **Methods**: Tasks 9, 10, 11 (properties, format, builtins)
7. **Polish**: Task 14 (error messages)

Each phase can be tested independently before moving to the next.
