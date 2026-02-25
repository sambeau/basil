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

### fmt()

The primary formatting method with multiple overloads.

#### Usage: fmt()

Format with default style (medium) and locale (en-US):

```parsley
123456.fmt()        // "123,456"
1234.5.fmt()        // "1,234.5"
```

#### Usage: fmt(precision)

Format floats with a specific number of decimal places:

```parsley
1234.5678.fmt(2)    // "1,234.57"
3.14159.fmt(4)      // "3.1416"
```

#### Usage: fmt(style)

Format with a named style:

```parsley
1234567.fmt("short")   // "1.2M"
1234567.fmt("medium")  // "1,234,567"
1234567.fmt("long")    // "1,234,567"
```

#### Usage: fmt(style, locale)

Format with style and locale:

```parsley
1234.fmt("medium", "de-DE")  // "1.234"
1234.fmt("medium", "fr-FR")  // "1 234"
```

#### Usage: fmt(options)

Format with an options dictionary:

```parsley
1234567.fmt({style: "short"})                    // "1.2M"
1234.5678.fmt({precision: 2})                    // "1,234.57"
1234.fmt({style: "medium", locale: "de-DE"})     // "1.234"
```

### short()

Compact notation for large numbers:

```parsley
1500.short()           // "1.5K"
1234567.short()        // "1.2M"
2500000000.short()     // "2.5B"

// With locale:
1234567.short("de-DE") // "1,2M"
```

### medium()

Standard format with thousand separators (default style):

```parsley
123456.medium()        // "123,456"
1234.5.medium()        // "1,234.5"

// With locale:
123456.medium("de-DE") // "123.456"
```

### long()

Full precision format:

```parsley
123456.long()          // "123,456"
1234.5.long()          // "1,234.50"
```

### format()

Alias for `fmt()`. Retained for backward compatibility:

```parsley
1000000.format()       // "1,000,000"
```

### humanize()

Alias for `short()`. Compact notation for large numbers:

```parsley
1500.humanize()        // "1.5K"
3500000.humanize()     // "3.5M"
```

### currency(code)

Formats with currency symbol and two decimal places:

```parsley
let x = 99
x.currency("USD")      // "$99.00"
x.currency("EUR")      // "€99.00"
x.currency("GBP")      // "£99.00"
```

> For precise currency arithmetic (avoiding floating-point rounding), use the [Money](money.md) type instead.

### percent()

Format as a percentage:

```parsley
0.125.percent()        // "13%"
0.5.percent()          // "50%"
```

### repr()

Returns a parseable literal representation:

```parsley
42.repr()              // "42"
3.14.repr()            // "3.14"
```

### toJSON()

Returns the JSON representation:

```parsley
42.toJSON()            // "42"
3.14.toJSON()          // "3.14"
```

### inspect()

Returns a debug dictionary with type information:

```parsley
42.inspect()           // {__type: "integer", value: 42}
3.14.inspect()         // {__type: "float", value: 3.14}
```

### toBox()

Renders the number in a box diagram:

```parsley
42.toBox()
```

```
┌────┐
│ 42 │
└────┘
```

## Float-Specific Methods

### abs()

Returns the absolute value:

```parsley
(-5).abs()             // 5
(-3.14).abs()          // 3.14
```

### round()

Round to the nearest integer or to n decimal places:

```parsley
3.7.round()            // 4
3.14159.round(2)       // 3.14
```

### floor()

Round down to the nearest integer:

```parsley
3.7.floor()            // 3
(-3.2).floor()         // -4
```

### ceil()

Round up to the nearest integer:

```parsley
3.2.ceil()             // 4
(-3.7).ceil()          // -3
```

## Type Conversions

```parsley
number("42")           // 42
number("3.14")         // 3.14
42.string()            // "42"
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

## Formatting Styles Summary

| Style | Integer | Float |
|-------|---------|-------|
| `short` | `"1.2M"` | `"1.2M"` |
| `medium` (default) | `"1,234,567"` | `"1,234.5"` |
| `long` | `"1,234,567"` | `"1,234.50"` |

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