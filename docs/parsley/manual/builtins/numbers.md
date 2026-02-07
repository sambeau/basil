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

Parsley supports two numeric types: **Integers** (whole numbers like 42) and **Floats** (decimal numbers like 3.14). Both types support arithmetic operations, comparisons, formatting methods, and integration with the comprehensive `@std/math` module for advanced calculations. Numbers in Parsley automatically coerce to the appropriate type for operations, making mathematical expressions natural and intuitive.

## Literals

### Integers

Create integers using standard decimal notation:

```parsley
42
```

**Result:** `42`

Negative integers:

```parsley
-5
```

**Result:** `-5`

Zero:

```parsley
0
```

**Result:** `0`

### Floats

Create floats using decimal point notation:

```parsley
3.14
```

**Result:** `3.14`

Negative floats:

```parsley
-0.5
```

**Result:** `-0.5`

Very small decimals:

```parsley
0.001
```

**Result:** `0.001`

## Operators

### Arithmetic

The standard arithmetic operators work on all numbers:

```parsley
5 + 3
```

**Result:** `8`

Subtraction:

```parsley
10 - 4
```

**Result:** `6`

Multiplication:

```parsley
6 * 7
```

**Result:** `42`

Division:

```parsley
20 / 4
```

**Result:** `5`

Modulo (remainder):

```parsley
17 % 5
```

**Result:** `2`

Unary negation:

```parsley
-42
```

**Result:** `-42`

### Comparisons

Numbers support all comparison operators:

```parsley
42 == 42
```

**Result:** `true`

Less than:

```parsley
3.14 < 5
```

**Result:** `true`

Greater than:

```parsley
42 > 0
```

**Result:** `true`

Less than or equal:

```parsley
0 <= 0
```

**Result:** `true`

Greater than or equal:

```parsley
5 >= 5
```

**Result:** `true`

Not equal:

```parsley
1 != 2
```

**Result:** `true`

## Methods

### format()

Format a number with thousand separators and appropriate decimal places.

```parsley
42.format()
```

**Result:** `"42"`

Float formatting shows appropriate decimals:

```parsley
3.14.format()
```

**Result:** `"3.14"`

Large numbers get thousand separators:

```parsley
1000000.format()
```

**Result:** `"1,000,000"`

### currency(code, locale?)

Format a number as currency. The currency code (e.g., "USD", "EUR", "GBP") determines the symbol and precision.

**Note:** This method is available but currently has limited locale support. Use the @std/money module for more comprehensive currency handling.

```parsley
x = 99
x.currency("USD")
```

**Result:** `"$99.00"`

Euros:

```parsley
y = 99
y.currency("EUR")
```

**Result:** `"€99.00"`

### humanize()

Format a number in compact notation, useful for large numbers (e.g., "1.2M", "5K").

```parsley
3500000.humanize()
```

**Result:** `"3.5M"`

Thousands:

```parsley
1500.humanize()
```

**Result:** `"1.5K"`

Billions:

```parsley
2500000000.humanize()
```

**Result:** `"2.5B"`


## Math Module

The `@std/math` module provides comprehensive mathematical functions. Import it at the beginning of your code:

```parsley
import @std/math
```

### Constants

#### PI

The mathematical constant π (pi):

```parsley
import @std/math
math.PI
```

**Result:** `3.14159...`

#### E

Euler's number (base of natural logarithm):

```parsley
import @std/math
math.E
```

**Result:** `2.71828...`

#### TAU

The mathematical constant τ (tau), equal to 2π:

```parsley
import @std/math
math.TAU
```

**Result:** `6.28318...`

### Rounding Functions

#### ceil()

Round up to the nearest integer:

```parsley
import @std/math
math.ceil(3.2)
```

**Result:** `4`

Negative numbers:

```parsley
import @std/math
math.ceil(-3.7)
```

**Result:** `-3`

#### floor()

Round down to the nearest integer:

```parsley
import @std/math
math.floor(3.7)
```

**Result:** `3`

#### round()

Round to the nearest integer (0.5 rounds up):

```parsley
import @std/math
math.round(3.5)
```

**Result:** `4`

#### trunc()

Remove the decimal part (truncate toward zero):

```parsley
import @std/math
math.trunc(-3.7)
```

**Result:** `-3`

### Comparison Functions

#### abs()

Return the absolute value (distance from zero):

```parsley
import @std/math
math.abs(-42)
```

**Result:** `42`

Positive numbers:

```parsley
import @std/math
math.abs(42)
```

**Result:** `42`

#### sign()

Return -1 for negative, 0 for zero, and 1 for positive:

```parsley
import @std/math
math.sign(-5)
```

**Result:** `-1`

Zero:

```parsley
import @std/math
math.sign(0)
```

**Result:** `0`

Positive:

```parsley
import @std/math
math.sign(5)
```

**Result:** `1`

#### clamp()

Constrain a number between minimum and maximum values:

```parsley
import @std/math
math.clamp(5, 1, 10)
```

**Result:** `5`

Below minimum:

```parsley
import @std/math
math.clamp(-5, 1, 10)
```

**Result:** `1`

Above maximum:

```parsley
import @std/math
math.clamp(15, 1, 10)
```

**Result:** `10`

#### max()

Return the larger of two numbers:

```parsley
import @std/math
math.max(3, 7)
```

**Result:** `7`

#### min()

Return the smaller of two numbers:

```parsley
import @std/math
math.min(3, 7)
```

**Result:** `3`

### Aggregation Functions

#### avg()

Calculate the average of an array of numbers:

```parsley
import @std/math
math.avg([10, 20, 30])
```

**Result:** `20`

#### sum()

Sum all elements in an array:

```parsley
import @std/math
math.sum([1, 2, 3, 4, 5])
```

**Result:** `15`

#### product()

Multiply all elements in an array:

```parsley
import @std/math
math.product([2, 3, 4])
```

**Result:** `24`

### Power Functions

#### pow(base, exponent)

Raise a base to an exponent:

```parsley
import @std/math
math.pow(2, 8)
```

**Result:** `256`

#### sqrt()

Calculate the square root:

```parsley
import @std/math
math.sqrt(16)
```

**Result:** `4`

#### exp()

Calculate e raised to a power (exponential function):

```parsley
import @std/math
math.exp(1)
```

**Result:** `2.71828...`

### Logarithm Functions

#### log()

Natural logarithm (base e):

```parsley
import @std/math
math.log(2.71828)
```

**Result:** `1`

#### log10()

Logarithm base 10:

```parsley
import @std/math
math.log10(100)
```

**Result:** `2`

#### log2()

Logarithm base 2:

```parsley
import @std/math
math.log2(8)
```

**Result:** `3`

### Trigonometric Functions

#### sin(), cos(), tan()

Calculate sine, cosine, and tangent (angles in radians):

```parsley
import @std/math
math.sin(math.PI / 2)
```

**Result:** `1`

#### asin(), acos(), atan()

Calculate inverse trigonometric functions:

```parsley
import @std/math
math.asin(1)
```

**Result:** `1.5707...` (π/2)

### Random Functions

#### random()

Generate a random number between 0 (inclusive) and 1 (exclusive):

```parsley
import @std/math
math.random()
```

**Result:** `0.523...` (varies each call)

#### randomInt(max) or randomInt(min, max)

Generate a random integer:

```parsley
import @std/math
math.randomInt(10)
```

**Result:** `7` (random integer from 0 to 9)

With min and max:

```parsley
import @std/math
math.randomInt(1, 100)
```

**Result:** `42` (random integer from 1 to 99)

#### seed(value)

Set the random seed for reproducible random numbers:

```parsley
import @std/math
math.seed(42)
math.random()
```

**Result:** `0.374...` (same each time seed is set to 42)

### Statistics Functions

#### median()

Find the middle value in an array:

```parsley
import @std/math
math.median([1, 2, 3, 4, 5])
```

**Result:** `3`

#### mode()

Find the most frequently occurring value:

```parsley
import @std/math
math.mode([1, 2, 2, 3, 3, 3])
```

**Result:** `3`

#### stddev()

Calculate the standard deviation:

```parsley
import @std/math
math.stddev([1, 2, 3, 4, 5])
```

**Result:** `1.414...`

#### variance()

Calculate the variance:

```parsley
import @std/math
math.variance([1, 2, 3, 4, 5])
```

**Result:** `2`

#### range()

Get the difference between max and min values:

```parsley
import @std/math
math.range([1, 2, 3, 4, 5])
```

**Result:** `4`

### Interpolation

#### lerp(a, b, t)

Linear interpolation between two values. The `t` parameter should be between 0 and 1, where 0 returns `a` and 1 returns `b`:

```parsley
import @std/math
math.lerp(0, 100, 0.5)
```

**Result:** `50`

At the start:

```parsley
import @std/math
math.lerp(0, 100, 0)
```

**Result:** `0`

At the end:

```parsley
import @std/math
math.lerp(0, 100, 1)
```

**Result:** `100`

## Type Conversions

Convert a string to a number using the `number()` function:

```parsley
number("42")
```

**Result:** `42`

Float strings:

```parsley
number("3.14")
```

**Result:** `3.14`

Convert a number to a string:

```parsley
42.string()
```

**Result:** `"42"`

## Common Patterns

Calculate compound interest:

```parsley
import @std/math
principal = 1000
rate = 0.05
years = 10
amount = principal * math.pow(1 + rate, years)
// amount is now approximately 1628.89
amount.format()
```

**Result:** `"1,628.89"`

Find the hypotenuse using the Pythagorean theorem:

```parsley
import @std/math
a = 3
b = 4
hypotenuse = math.sqrt(math.pow(a, 2) + math.pow(b, 2))
```

**Result:** `5`

Calculate a moving average:

```parsley
import @std/math
prices = [10, 12, 11, 13, 15, 14]
moving_avg = math.avg(prices[0:3])
```

**Result:** `11`

Format large data sizes (as commonly seen in data science):

```parsley
bytes = 1500000000
gb_size = (bytes / 1000000000).humanize()
gb_size
```

**Result:** `"1.5B"`

## Key Differences from Other Languages

- **No scientific notation:** Use `math.pow()` instead (e.g., `math.pow(10, 3)` not `1e3`)
- **All comparisons use `==` and `!=`:** No `===` operator
- **Coercion is automatic:** Operations on mixed integer/float types work seamlessly
- **No increment operators:** Use `x = x + 1` instead of `x++`
- **Division always returns float:** `5 / 2` equals `2.5`, not `2`
- **Number methods on literals need careful parsing:** Direct method calls on number literals may need parentheses (e.g., `(3.14).format()` for floats)
- **Math module requires import:** Must `import @std/math` before using functions like `sqrt()`, `pow()`, `PI`, etc.
- **Rounding functions are in Math module:** Use `math.floor()`, `math.ceil()`, `math.round()`, `math.trunc()` (not direct methods on numbers)

## See Also

- [Operators](../fundamentals/operators.md) — arithmetic, comparison, and assignment operators
- [Money](money.md) — exact decimal arithmetic for currency values
- [Strings](strings.md) — `.toNumber()` for parsing, number interpolation
- [Types](../fundamentals/types.md) — integer and float in the type system
- [@std/math](../stdlib/math.md) — math functions, constants, statistics, and trigonometry
- [@std/valid](../stdlib/valid.md) — number validation predicates
