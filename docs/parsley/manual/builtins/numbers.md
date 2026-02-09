---
id: numbers
title: Numbers
system: parsley
type: builtin
name: numbers
created: 2025-12-16
version: 1.0.0
author: Basil Team
keywords:
  - number
  - integer
  - float
  - decimal
  - arithmetic
  - math
  - calculation
---

# Numbers

Parsley has two numeric types: **Integers** (`42`) and **Floats** (`3.14`). Arithmetic, comparisons, and mixed-type operations work as you'd expect from other languages — this page focuses on where Parsley differs.

## Literals

```parsley
42                  // integer
-5                  // negative integer
3.14                // float
-0.5                // negative float
```

> ⚠️ There is no scientific notation. Use `math.pow(10, 3)` instead of `1e3`.

## Operators

Standard arithmetic (`+`, `-`, `*`, `/`, `%`) and comparisons (`==`, `!=`, `<`, `>`, `<=`, `>=`) work on numbers. A few things to note:

```parsley
5 / 2               // 2.5 — division always returns a float
17 % 5              // 2
3 + 0.5             // 3.5 — int/float coercion is automatic
```

**No `++` or `--` operators.** Use `let x = x + 1`.

**No `===` operator.** All equality comparisons use `==` and `!=`.

## Methods

Numbers have built-in formatting methods — these are where Parsley adds value over raw arithmetic.

### format()

Adds thousand separators and preserves decimal places:

```parsley
1000000.format()    // "1,000,000"
3.14.format()       // "3.14"
```

### humanize()

Compact notation for large numbers:

```parsley
1500.humanize()          // "1.5K"
3500000.humanize()       // "3.5M"
2500000000.humanize()    // "2.5B"
```

### currency(code)

Formats with currency symbol and two decimal places:

```parsley
let x = 99
x.currency("USD")       // "$99.00"
x.currency("EUR")       // "€99.00"
x.currency("GBP")       // "£99.00"
```

> For precise currency arithmetic (avoiding floating-point rounding), use the [Money](money.md) type instead.

## Type Conversions

```parsley
number("42")        // 42
number("3.14")      // 3.14
42.string()         // "42"
```

Numbers interpolate naturally in template strings:

```parsley
let n = 42
`The answer is {n}`    // "The answer is 42"
```

## Math Module

Import `@std/math` for mathematical functions, constants, statistics, and random numbers. A quick taste:

```parsley
import @std/math

math.PI                         // 3.14159...
math.round(3.5)                 // 4
math.clamp(15, 1, 10)           // 10
math.avg([10, 20, 30])          // 20
math.sqrt(math.pow(3, 2) + math.pow(4, 2))  // 5
```

The module includes rounding (`ceil`, `floor`, `round`, `trunc`), comparison (`abs`, `sign`, `clamp`, `min`, `max`), aggregation (`sum`, `avg`, `product`), powers and logarithms, trigonometry, statistics (`median`, `mode`, `stddev`, `variance`), random numbers, and interpolation.

See [@std/math](../stdlib/math.md) for the full reference.

## Key Differences from Other Languages

| Gotcha | Parsley | Other languages |
|---|---|---|
| Division | `5 / 2` → `2.5` (always float) | Often integer division |
| Increment | `let x = x + 1` | `x++` |
| Equality | `==` only | `===` in JS |
| Scientific notation | Not supported | `1e3` |
| Rounding | `math.round(x)` (import required) | Often built-in |
| Method calls on literals | `(3.14).format()` — parentheses needed for floats | Varies |
| Int/float coercion | Automatic and seamless | Often explicit |

## See Also

- [Operators](../fundamentals/operators.md) — arithmetic, comparison, and assignment operators
- [Money](money.md) — exact decimal arithmetic for currency values
- [Strings](strings.md) — `.toNumber()` for parsing, number interpolation
- [Types](../fundamentals/types.md) — integer and float in the type system
- [@std/math](../stdlib/math.md) — math functions, constants, statistics, and trigonometry
- [@std/valid](../stdlib/valid.md) — number validation predicates