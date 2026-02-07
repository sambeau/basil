---
id: man-pars-std-math
title: "@std/math"
system: parsley
type: stdlib
name: math
created: 2026-02-06
version: 0.2.0
author: Basil Team
keywords:
  - math
  - stdlib
  - rounding
  - statistics
  - random
  - trigonometry
  - interpolation
  - geometry
---

# @std/math

Mathematical functions and constants.

```parsley
let math = import @std/math
```

## Constants

| Name | Value | Description |
|---|---|---|
| `PI` | 3.14159… | Pi (π) |
| `E` | 2.71828… | Euler's number |
| `TAU` | 6.28318… | Tau (2π) |

```parsley
math.PI                          // 3.141592653589793
math.TAU                         // 6.283185307179586
math.E                           // 2.718281828459045
```

## Rounding

| Function | Args | Description |
|---|---|---|
| `floor(n)` | number | Round down to integer |
| `ceil(n)` | number | Round up to integer |
| `round(n)` | number | Round to nearest integer |
| `trunc(n)` | number | Truncate toward zero |

```parsley
math.floor(3.7)                  // 3
math.ceil(3.2)                   // 4
math.round(3.5)                  // 4
math.trunc(-3.7)                 // -3
```

## Comparison & Clamping

| Function | Args | Description |
|---|---|---|
| `abs(n)` | number | Absolute value |
| `sign(n)` | number | Returns -1, 0, or 1 |
| `clamp(n, min, max)` | number, number, number | Clamp value to range |
| `min(a, b)` / `min(arr)` | two numbers or array | Minimum value |
| `max(a, b)` / `max(arr)` | two numbers or array | Maximum value |

```parsley
math.abs(-42)                    // 42
math.sign(-5)                    // -1
math.clamp(15, 0, 10)            // 10
math.min(3, 7)                   // 3
math.max([10, 20, 5])            // 20
```

## Aggregation

All aggregation functions accept either two arguments or an array.

| Function | Description |
|---|---|
| `sum(...)` | Sum of values |
| `avg(...)` / `mean(...)` | Average (`mean` is an alias) |
| `product(...)` | Product of values |
| `count(arr)` | Count elements |

```parsley
let nums = [1, 2, 3, 4, 5]
math.sum(nums)                   // 15
math.avg(nums)                   // 3
math.product(nums)               // 120
math.count(nums)                 // 5

math.sum(10, 20)                 // 30
```

## Statistics

These functions accept an array only.

| Function | Args | Description |
|---|---|---|
| `median(arr)` | array | Median value |
| `mode(arr)` | array | Most frequent value |
| `stddev(arr)` | array | Standard deviation |
| `variance(arr)` | array | Variance |
| `range(arr)` | array | max − min |

```parsley
math.median([1, 2, 3, 4, 100])  // 3
math.mode([1, 2, 2, 3])         // 2
math.stddev([1, 2, 3, 4, 5])    // ~1.41
math.variance([1, 2, 3, 4, 5])  // 2
math.range([10, 20, 5, 30])     // 25
```

## Random

| Function | Args | Description |
|---|---|---|
| `random()` | none | Random float 0.0–1.0 |
| `randomInt(max)` | integer | Random integer 0 to max−1 |
| `randomInt(min, max)` | integer, integer | Random integer min to max−1 |
| `seed(n)` | integer | Seed the random generator (for reproducibility) |

```parsley
math.random()                    // 0.314... (random)
math.randomInt(10)               // 0–9
math.randomInt(5, 10)            // 5–9

math.seed(42)
math.random()                    // deterministic after seeding
```

## Powers & Logarithms

| Function | Args | Description |
|---|---|---|
| `sqrt(n)` | number | Square root |
| `pow(base, exp)` | number, number | base^exp |
| `exp(n)` | number | e^n |
| `log(n)` | number | Natural logarithm (ln) |
| `log10(n)` | number | Base-10 logarithm |

```parsley
math.sqrt(16)                    // 4
math.pow(2, 10)                  // 1024
math.exp(1)                      // 2.718281828459045
math.log(math.E)                 // 1
math.log10(1000)                 // 3
```

## Trigonometry

All trigonometric functions use **radians**. Use `degrees()` and `radians()` for conversion.

| Function | Description |
|---|---|
| `sin(n)` | Sine |
| `cos(n)` | Cosine |
| `tan(n)` | Tangent |
| `asin(n)` | Arc sine |
| `acos(n)` | Arc cosine |
| `atan(n)` | Arc tangent |
| `atan2(y, x)` | Arc tangent of y/x (two-argument form) |

```parsley
math.sin(math.PI / 2)           // 1
math.cos(0)                     // 1
math.atan2(1, 1)                // 0.785... (π/4)
```

## Angular Conversion

| Function | Args | Description |
|---|---|---|
| `degrees(radians)` | number | Convert radians to degrees |
| `radians(degrees)` | number | Convert degrees to radians |

```parsley
math.degrees(math.PI)           // 180
math.radians(90)                // 1.5707... (π/2)
```

## Geometry & Interpolation

| Function | Args | Description |
|---|---|---|
| `hypot(a, b)` | number, number | Hypotenuse: √(a² + b²) |
| `dist(x1, y1, x2, y2)` | four numbers | Distance between two points |
| `lerp(a, b, t)` | number, number, number | Linear interpolation: a + (b−a)·t |
| `map(n, inMin, inMax, outMin, outMax)` | five numbers | Map value from one range to another |

```parsley
math.hypot(3, 4)                 // 5
math.dist(0, 0, 3, 4)           // 5
math.lerp(0, 100, 0.5)          // 50
math.map(5, 0, 10, 0, 100)      // 50
```

### `lerp` and `map`

`lerp(a, b, t)` returns the value at position `t` between `a` and `b`, where `t=0` gives `a` and `t=1` gives `b`. Values of `t` outside 0–1 extrapolate beyond the range.

`map(n, inMin, inMax, outMin, outMax)` rescales `n` from the input range to the output range. Equivalent to `lerp(outMin, outMax, (n - inMin) / (inMax - inMin))`.

## See Also

- [Operators](../fundamentals/operators.md) — arithmetic operators
- [Types](../fundamentals/types.md) — integer and float types
- [@std/valid](valid.md) — number validation functions