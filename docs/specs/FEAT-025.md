---
id: FEAT-025
title: "Money Type with Precise Numbers"
status: draft
priority: high
created: 2025-12-04
author: "@human"
---

# FEAT-025: Money Type with Precise Numbers

## Summary

Add a money/currency type to Parsley using a new "precise number" system with the `#` sigil. Currency literals use familiar symbols (`$`, `£`, `€`) as shortcuts for typed precise numbers (`USD#`, `GBP#`, `EUR#`). Mixing different currency types is a compile-time error, ensuring safe money arithmetic.

## User Story

As a web developer handling payments, I want to work with money values that are exact and type-safe, so that I don't introduce floating-point errors or accidentally mix currencies.

## Motivation

### The Problem with Floats

```javascript
// JavaScript
0.1 + 0.2  // 0.30000000000000004
```

This is unacceptable for financial calculations. Web applications deal with money constantly, and Parsley should handle it correctly by default.

### Goals

1. **Exact arithmetic**: `$0.10 + $0.20 = $0.30` (always)
2. **Type safety**: `$10 + £5` is an error (can't mix currencies)
3. **Familiar syntax**: `$12.34` looks like what users expect
4. **Locale-aware formatting**: `$1234.56.format("de-DE")` → `"1.234,56 $"`

## Acceptance Criteria

- [ ] Currency literals parse correctly (`$12.34`, `£99.99`, `EUR#50.00`)
- [ ] Arithmetic between same currencies works (`$10 + $5` → `$15`)
- [ ] Arithmetic between different currencies errors at runtime
- [ ] Arithmetic with floats/integers errors (explicit conversion required)
- [ ] Scalar multiplication works (`$10 * 3` → `$30`)
- [ ] Comparison operators work (`$10 > $5` → `true`)
- [ ] `.format()` method produces locale-aware output
- [ ] `.toFloat()` converts to float (explicit, lossy)
- [ ] `money(12.34, "USD")` function for dynamic currency creation

## Design Decisions

### The `#` Precise Number System

**Decision**: Introduce `#` as the "precise number" sigil, with an optional prefix indicating the type.

```parsley
PREFIX#number   // A typed precise number
USD#12.34       // US Dollars
GBP#99.99       // British Pounds
#12.34          // Bare precise number (decimal, for future use)
```

**Rationale**: 
- Creates a coherent family for future extensions (units, rationals)
- `#` suggests "number" (as in "No." or hashtag/pound sign)
- Prefix makes the type explicit and self-documenting

### Currency Symbol Shortcuts

**Decision**: Common currency symbols expand to their ISO 4217 code:

| Symbol | Expands To | Currency |
|--------|-----------|----------|
| `$12.34` | `USD#12.34` | US Dollar |
| `CA$12.34` | `CAD#12.34` | Canadian Dollar |
| `AU$12.34` | `AUD#12.34` | Australian Dollar |
| `£12.34` | `GBP#12.34` | British Pound |
| `€12.34` | `EUR#12.34` | Euro |
| `¥12.34` | `JPY#12.34` | Japanese Yen |
| `CN¥12.34` | `CNY#12.34` | Chinese Yuan |
| `HK$12.34` | `HKD#12.34` | Hong Kong Dollar |
| `S$12.34` | `SGD#12.34` | Singapore Dollar |

**Rationale**: 
- These cover the 10 most traded currencies (plus CHF via `CHF#`)
- `$` = USD is unambiguous (other dollars get prefixes)
- Familiar syntax for common cases

### Other Currencies

Currencies without shortcuts use the explicit `CODE#` syntax:

```parsley
CHF#100.00      // Swiss Franc
INR#500.00      // Indian Rupee
KRW#10000       // Korean Won
BRL#50.00       // Brazilian Real
```

### Float/Integer Mixing is an Error

**Decision**: Cannot mix money with raw numbers without explicit conversion.

```parsley
$10 + 5         // ERROR: cannot add money and integer
$10 + 5.0       // ERROR: cannot add money and float
$10 * 2         // OK: scalar multiplication
$10 / 2         // OK: scalar division → $5
$10.toFloat()   // OK: explicit conversion → 10.0
(10.0).usd()    // OK: explicit conversion → $10
money(10.0, "USD") // OK: function conversion → $10
```

**Rationale**: The whole point is to be safe with money. Implicit conversions defeat that purpose.

### Internal Representation

**Decision**: Store as integer cents (or smallest unit) + currency code + scale.

```go
type Money struct {
    Amount   int64  // e.g., 1234 for $12.34
    Currency string // "USD", "GBP", etc.
    Scale    int8   // decimal places (2 for USD, 0 for JPY)
}
```

**Rationale**:
- Integer arithmetic is exact
- Scale handles currencies with different decimal places (JPY has 0, KWD has 3)
- int64 handles values up to ~92 quadrillion cents

## Proposed Syntax

### Literals

```parsley
// Symbol shortcuts (common currencies)
$12.34              // USD
£99.99              // GBP
€50.00              // EUR
¥1000               // JPY (no decimal)
CA$25.00            // CAD
AU$30.00            // AUD

// Explicit form (all currencies)
USD#12.34           // US Dollar
GBP#99.99           // British Pound
CHF#100.00          // Swiss Franc
INR#500.00          // Indian Rupee
```

### Arithmetic

```parsley
$10 + $5            // $15
$20 - $8            // $12
$10 * 3             // $30
$15 / 3             // $5
$17 / 3             // $5.67 (rounds to currency precision)

$10 + £5            // ERROR: cannot mix USD and GBP
USD#10 + $5         // $15 (same currency, different syntax)
```

### Comparison

```parsley
$10 > $5            // true
$10 == USD#10       // true
$10 < £5            // ERROR: cannot compare USD and GBP
```

### Methods

```parsley
let price = $1234.56

price.amount        // 123456 (integer cents)
price.currency      // "USD"
price.scale         // 2

price.format()              // "$1,234.56"
price.format("de-DE")       // "1.234,56 $"
price.format("fr-FR")       // "1 234,56 $"

price.toFloat()             // 1234.56 (explicit, may lose precision)
price.toDict()              // {__type: "money", amount: 123456, currency: "USD", scale: 2}

price.abs()                 // $1234.56
(-$50).abs()                // $50
```

### Construction

```parsley
money(12.34, "USD")         // $12.34
money(1234, "JPY")          // ¥1234 (no decimals for JPY)
(12.34).usd()               // $12.34
(99.99).gbp()               // £99.99
```

### In Templates

```parsley
let total = $99.99
"Your total is {total}"           // "Your total is $99.99"
"Prix: {total.format('fr-FR')}"   // "Prix: 99,99 $"
```

---

## Technical Context

### Affected Components

- `pkg/parsley/lexer/lexer.go` — Add token types for currency symbols and `#` precise numbers
- `pkg/parsley/token/token.go` — Define `MONEY_LITERAL`, `PRECISE_NUMBER` tokens
- `pkg/parsley/ast/ast.go` — Add `MoneyLiteral` AST node
- `pkg/parsley/parser/parser.go` — Parse money literals
- `pkg/parsley/evaluator/evaluator.go` — Money arithmetic operators
- `pkg/parsley/evaluator/builtins.go` — `money()` function
- `pkg/parsley/object/object.go` — `Money` object type
- `pkg/parsley/evaluator/methods.go` — Money methods (`.format()`, `.toFloat()`, etc.)

### Dependencies

- Depends on: None
- Blocks: Future unit types (`12.34#mm`), rationals (`#22/7`)

### Edge Cases & Constraints

1. **Division rounding** — `$10 / 3` = `$3.33` (banker's rounding to currency precision)
2. **JPY has no decimals** — `¥100.50` is an error
3. **Negative money** — Allowed: `-$50`, `$(-50)`
4. **Overflow** — int64 supports up to ~$92 trillion; error on overflow
5. **Zero-scale currencies** — JPY, KRW use scale=0; others typically scale=2
6. **Unknown currency codes** — `XXX#12.34` is an error unless XXX is valid ISO 4217

## Open Questions

1. **Interpolation in amounts?** — Should `$(10 + 5)` work? (Probably not needed)
2. **Currency conversion?** — Out of scope, but should we provide hooks?
3. **Rounding mode** — Banker's rounding (round half to even) is standard for finance
4. **Percentage operations** — `$100 * 15%` → Should this work?

## Future Work

### Units (separate FEAT)

The `#` sigil extends naturally to SI units with suffix notation:

```parsley
#12.34mm            // millimeters
#5kg                // kilograms
#100m / #10s        // → #10m/s (derived unit)
#5km + #500m        // → #5.5km (compatible units)
#5kg + #5m          // ERROR: incompatible units
```

### Bare Decimals

```parsley
#12.34              // Precise decimal (no prefix or suffix)
```

## Related

- FEAT-024: Print function (type representation)
- Future: Unit types (`#` with suffix)
- Future: Rational numbers

## Notes

This feature was motivated by the common problem of floating-point money bugs in web applications. The design prioritizes safety (no implicit mixing) over convenience, which is appropriate for financial calculations.

The `#` precise number family provides a foundation for future numeric types that need exactness (units, rationals) while keeping a consistent syntax.
