---
id: FEAT-025
title: "Money Type"
status: implemented
priority: high
created: 2025-12-04
updated: 2025-12-05
author: "@human"
---

# FEAT-025: Money Type

## Summary

Add a money/currency type to Parsley with familiar currency symbols (`$`, `£`, `€`) and explicit `CODE#` syntax for all currencies. Mixing different currencies is a runtime error, ensuring safe money arithmetic.

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

## Design Philosophy

Money arithmetic is just integer arithmetic with a type tag and scale. The **only** things that make it special are:

1. **Type safety** — Can't mix currencies
2. **Scale** — Display as dollars, store as cents
3. **Formatting** — Locale-aware output

Everything else (sum, sort, filter, map) already works if we get the basics right. Arrays of money don't need special handling — if the basic operators work, composition is free.

## Acceptance Criteria

- [ ] Currency literals parse correctly (`$12.34`, `£99.99`, `EUR#50.00`)
- [ ] Arithmetic between same currencies works (`$10 + $5` → `$15`)
- [ ] Arithmetic between different currencies errors at runtime
- [ ] Arithmetic with floats/integers errors (explicit conversion required)
- [ ] Scalar multiplication/division works (`$10 * 3` → `$30`, `$15 / 3` → `$5`)
- [ ] Comparison operators work (`$10 > $5` → `true`)
- [ ] `.format()` method produces locale-aware output
- [ ] `.split(n)` distributes amount fairly across n parts
- [ ] `money(cents, "USD")` function for dynamic currency creation

## Minimum Viable Feature Set

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

### Operators

| Operation | Example | Notes |
|-----------|---------|-------|
| `+` `-` | `$10 + $5` → `$15` | Same currency only |
| `*` `/` | `$10 * 3` → `$30` | Scalar multiplication/division |
| `>` `<` `>=` `<=` `==` `!=` | `$10 > $5` → `true` | Same currency only |
| Unary `-` | `-$50` | Negation |

Division uses banker's rounding (round half to even) — standard for finance.

### Properties

| Property | Example | Result |
|----------|---------|--------|
| `.currency` | `$10.50.currency` | `"USD"` |
| `.amount` | `$10.50.amount` | `1050` (integer cents) |

### Methods

| Method | Example | Result |
|--------|---------|--------|
| `.format()` | `$1234.56.format()` | `"$1,234.56"` |
| `.format(locale)` | `$1234.56.format("de-DE")` | `"1.234,56 $"` |
| `.abs()` | `(-$50).abs()` | `$50` |
| `.split(n)` | `$100.split(3)` | `[$33.34, $33.33, $33.33]` |

The `.split(n)` method solves the "splitting bills" problem — it ensures the parts always sum to the whole by distributing the remainder across the first parts.

### Construction

```parsley
money(1234, "USD")          // $12.34 (amount in cents, scale=2 for known currency)
money(1000, "JPY")          // ¥1000 (scale=0 for JPY)
money(100000000, "BTC", 8)  // BTC#1.00000000 (explicit scale for unknown currency)
money(100, "PTS", 0)        // PTS#100 (custom currency with explicit scale)
```

For unknown currencies, the scale parameter is required in `money()` to avoid ambiguity.

### What We're NOT Adding

| Omitted | Rationale |
|---------|-----------|
| `.toFloat()` | Encourages bad patterns; use `.amount / 100` if needed |
| `.round()` | Division already rounds appropriately |
| Currency conversion | Out of scope (needs exchange rates) |
| Percentage type | `$100 * 0.15` works with scalar multiplication |
| `.usd()`, `.gbp()` methods | `money()` function is sufficient |

## Design Decisions

### Currency Codes

**Decision**: Accept any 3-letter uppercase code, not just ISO 4217.

```parsley
USD#12.34       // US Dollar (ISO 4217)
BTC#0.00001234  // Bitcoin (not ISO, but useful)
PTS#100         // Custom "points" currency
```

**Rationale**:
- Mixing errors catch typos quickly (`USd#10 + USD#5` → error)
- Enables legitimate use cases (crypto, loyalty points, game currencies)
- More "Parsley" — trust the user, fail fast when things don't match
- Can add strict mode or linting later if needed

### Scale (Decimal Places)

**Decision**: Infer scale from the literal; promote to higher precision on arithmetic.

```parsley
USD#12.34           // scale=2 (inferred from literal)
BTC#0.00001234      // scale=8 (inferred from literal)
JPY#1000            // scale=0 (inferred from literal)

// Arithmetic promotes to higher precision
USD#1.00 + USD#0.001    // USD#1.001 (scale=3)
BTC#1.00 + BTC#0.00001  // BTC#1.00001 (scale=5)
```

**Known currency validation**: For ISO 4217 currencies, warn if scale exceeds standard:
```parsley
USD#0.001   // WARNING: USD typically uses 2 decimal places
JPY#100.5   // ERROR: JPY uses 0 decimal places
```

### Currency Symbol Shortcuts

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

**Rationale**: These cover the most traded currencies. `$` = USD is unambiguous (other dollars get prefixes).

### Float/Integer Mixing is an Error

```parsley
$10 + 5         // ERROR: cannot add money and integer
$10 + 5.0       // ERROR: cannot add money and float
$10 * 2         // OK: scalar multiplication
$10 / 2         // OK: scalar division
```

**Rationale**: The whole point is to be safe with money. Implicit conversions defeat that purpose.

### Internal Representation

```go
type Money struct {
    Amount   int64  // e.g., 1234 for $12.34
    Currency string // "USD", "GBP", etc.
    Scale    int8   // decimal places (2 for USD, 0 for JPY)
}
```

- Integer arithmetic is exact
- Scale handles currencies with different decimal places (JPY has 0, KWD has 3)
- int64 handles values up to ~$92 quadrillion

## Examples

### Basic Arithmetic

```parsley
$10 + $5            // $15
$20 - $8            // $12
$10 * 3             // $30
$15 / 3             // $5
$17 / 3             // $5.67 (banker's rounding)

$10 + £5            // ERROR: cannot add USD and GBP
USD#10 + $5         // $15 (same currency, different syntax)
```

### Arrays (Composition Works Free)

```parsley
let prices = [$10, $20, $30]

sum(prices)                     // $60
prices | sort                   // [$10, $20, $30]
prices | map(p => p * 2)        // [$20, $40, $60]
prices | filter(p => p > $15)   // [$20, $30]
```

### Splitting Bills

```parsley
$100.split(3)       // [$33.34, $33.33, $33.33]
$10.split(4)        // [$2.50, $2.50, $2.50, $2.50]

// The parts always sum to the original
sum($100.split(3))  // $100.00 (exactly)
```

### Formatting

```parsley
let price = $1234.56

price.format()              // "$1,234.56"
price.format("de-DE")       // "1.234,56 $"
price.format("fr-FR")       // "1 234,56 $"
price.format("ja-JP")       // "$1,234.56" (or "US$1,234.56")
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

- `pkg/parsley/lexer/lexer.go` — Add token types for currency symbols and `CODE#` syntax
- `pkg/parsley/ast/ast.go` — Add `MoneyLiteral` AST node
- `pkg/parsley/parser/parser.go` — Parse money literals
- `pkg/parsley/evaluator/evaluator.go` — Money object type and arithmetic operators
- `pkg/parsley/evaluator/methods.go` — Money methods (`.format()`, `.abs()`, `.split()`)

### Formatting Implementation

The existing `golang.org/x/text/currency` package supports **154 ISO 4217 currencies** with full locale-aware formatting. This covers all major world currencies.

For `.format()`:
- **Known currencies (154)**: Use `golang.org/x/text/currency` for proper locale formatting
- **Unknown currencies (BTC, custom)**: Fall back to simple format: `CODE amount` (e.g., `BTC 1.00001234`)

Currencies NOT in the library (as of 2024):
- MRU (Mauritanian Ouguiya) — newer ISO code
- VES (Venezuelan Bolívar Soberano) — newer ISO code  
- BTC, ETH, XBT — crypto (not ISO 4217)

### Edge Cases & Constraints

1. **Division rounding** — Banker's rounding (round half to even)
2. **Scale mismatch** — Arithmetic promotes to higher precision (no precision loss)
3. **JPY has no decimals** — `¥100.50` is an error (known currency validation)
4. **Negative money** — Allowed: `-$50`
5. **Overflow** — int64 supports up to ~$92 trillion; error on overflow
6. **Unknown currency codes** — Any 3-letter uppercase code accepted
7. **Known currency scale warning** — `USD#0.001` warns about non-standard precision

## Open Questions

1. **Interpolation in amounts?** — Should `$(10 + 5)` work? (Probably not needed)
2. **Currency conversion?** — Out of scope, but should we provide hooks?

## Future Work

### The `#` Precise Number Family

The `CODE#number` syntax creates a foundation for future extensions:

```parsley
// Units (separate FEAT)
12.34#mm            // millimeters
5#kg                // kilograms
100#m / 10#s        // → 10#m/s (derived unit)

// Bare decimals (separate FEAT)
#12.34              // Precise decimal (no currency/unit)
```

## Related

- FEAT-024: Print function (type representation)
- Future: Unit types (`#` with suffix)
- Future: Rational numbers

## Notes

This feature was motivated by the common problem of floating-point money bugs in web applications. The design prioritizes:

- **Safety** over convenience (no implicit mixing)
- **Composability** — basic operators enable all array operations for free
- **Minimalism** — smallest feature set that solves 90% of use cases
