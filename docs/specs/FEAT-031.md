---
id: FEAT-031
title: "std/math Standard Library Module"
status: implemented
created: 2025-12-05
implemented: 2025-12-05
---

# FEAT-031: std/math Standard Library Module

## Summary
Create a `std/math` standard library module containing mathematical functions, constants, and random number generation. Move existing math builtins to this module.

## Motivation
Currently, Parsley has math functions (sin, cos, tan, etc.) as builtins. Moving them to a standard library module:
- Keeps the global namespace clean
- Groups related functions logically
- Follows common language patterns (Python's `math`, Go's `math`)
- Allows users to import only what they need

## Target Audience
Web developers, office workers, educators, students, game developers. NOT data scientists or mathematicians.

### Functions by Audience
- **Educators/students**: Stats 101 toolkit (mean, median, mode, stddev, variance, range)
- **Office workers**: Aggregations that work on arrays (sum, avg, count, min, max)
- **Game designers**: lerp, clamp, dist, map, random with seeding
- **Web devs**: The basics they'd expect from any math library (trig, rounding, powers)

### Explicitly Out of Scope
- **Matrix/vector operations** - requires Vector/Matrix types first
- **Perlin noise** - complex implementation, niche use case
- **Quaternions, slerp** - 3D-specific, beyond target audience
- **Advanced statistics** - percentile, quartile, correlation, z-score (leave for data scientists)
- **Hyperbolic functions** - sinh, cosh, tanh (niche)
- **Special functions** - gamma, factorial (niche)

## Design Decisions

### 1. Move vs Deprecate Builtins
**Decision**: Move existing builtins to `std/math`. They are not widely used yet.

### 2. Return Types
**Decision**: Return `int` when result is a whole number (e.g., `floor(3.7)` → `3`), otherwise `float`.

### 3. Aggregation Functions (min, max, sum, avg)
**Decision**: Accept both two arguments OR a single array:
```parsley
math.min(1, 2)        // → 1
math.min([1, 2, 3])   // → 1
```

### 4. Random Number Seeding
**Decision**: Include `seed(n)` for reproducible random sequences (game dev use case).

## Specification

### Import
```parsley
// Import entire module
let math = import(@std/math)
math.sin(math.PI)

// Destructure specific exports
let {sin, cos, PI} = import(@std/math)
sin(PI / 2)
```

### Constants
| Constant | Value | Description |
|----------|-------|-------------|
| `math.PI` | 3.14159... | Pi |
| `math.E` | 2.71828... | Euler's number |
| `math.TAU` | 6.28318... | 2π |

### Functions

#### Rounding
| Function | Description | Example |
|----------|-------------|---------|
| `floor(x)` | Round down to integer | `floor(3.7)` → `3` |
| `ceil(x)` | Round up to integer | `ceil(3.2)` → `4` |
| `round(x)` | Round to nearest integer | `round(3.5)` → `4` |
| `trunc(x)` | Truncate toward zero | `trunc(-3.7)` → `-3` |

#### Comparison & Clamping
| Function | Description | Example |
|----------|-------------|---------|
| `abs(x)` | Absolute value | `abs(-5)` → `5` |
| `sign(x)` | Sign of number (-1, 0, 1) | `sign(-5)` → `-1` |
| `clamp(x, min, max)` | Constrain to range | `clamp(15, 0, 10)` → `10` |

#### Aggregation (accept 2 args OR array)
| Function | Description | Example |
|----------|-------------|---------|
| `min(a, b)` or `min(arr)` | Minimum value | `min([1,2,3])` → `1` |
| `max(a, b)` or `max(arr)` | Maximum value | `max([1,2,3])` → `3` |
| `sum(a, b)` or `sum(arr)` | Sum of values | `sum([1,2,3])` → `6` |
| `avg(a, b)` or `avg(arr)` | Average (mean) | `avg([1,2,3])` → `2` |
| `product(a, b)` or `product(arr)` | Product of values | `product([2,3,4])` → `24` |
| `count(arr)` | Count of values | `count([1,2,3])` → `3` |

#### Statistics (array only)
| Function | Description | Example |
|----------|-------------|---------|
| `median(arr)` | Middle value | `median([1,2,3,4,5])` → `3` |
| `mode(arr)` | Most frequent value | `mode([1,2,2,3])` → `2` |
| `stddev(arr)` | Standard deviation | `stddev([2,4,4,4,5,5,7,9])` → `2` |
| `variance(arr)` | Variance (stddev²) | `variance([2,4,4,4,5,5,7,9])` → `4` |
| `range(arr)` | max - min | `range([1,5,3])` → `4` |

#### Random
| Function | Description | Example |
|----------|-------------|---------|
| `random()` | Random float 0.0-1.0 | `random()` → `0.7234...` |
| `randomInt(min, max)` | Random int in range (inclusive) | `randomInt(1, 6)` → `4` |
| `seed(n)` | Seed RNG for reproducibility | `seed(42)` |

#### Powers & Logarithms
| Function | Description | Example |
|----------|-------------|---------|
| `sqrt(x)` | Square root | `sqrt(16)` → `4` |
| `pow(x, y)` | x raised to power y | `pow(2, 3)` → `8` |
| `exp(x)` | e raised to power x | `exp(1)` → `2.718...` |
| `log(x)` | Natural logarithm | `log(math.E)` → `1` |
| `log10(x)` | Base-10 logarithm | `log10(100)` → `2` |

#### Trigonometry
| Function | Description | Example |
|----------|-------------|---------|
| `sin(x)` | Sine (radians) | `sin(math.PI/2)` → `1` |
| `cos(x)` | Cosine (radians) | `cos(0)` → `1` |
| `tan(x)` | Tangent (radians) | `tan(0)` → `0` |
| `asin(x)` | Arc sine | `asin(1)` → `1.5707...` |
| `acos(x)` | Arc cosine | `acos(1)` → `0` |
| `atan(x)` | Arc tangent | `atan(1)` → `0.7853...` |
| `atan2(y, x)` | Arc tangent of y/x | `atan2(1, 1)` → `0.7853...` |

#### Angular Conversion
| Function | Description | Example |
|----------|-------------|---------|
| `degrees(rad)` | Radians to degrees | `degrees(math.PI)` → `180` |
| `radians(deg)` | Degrees to radians | `radians(180)` → `3.14159...` |

#### Geometry & Interpolation
| Function | Description | Example |
|----------|-------------|---------||
| `hypot(x, y)` | Hypotenuse √(x² + y²) | `hypot(3, 4)` → `5` |
| `dist(x1, y1, x2, y2)` | 2D distance between points | `dist(0, 0, 3, 4)` → `5` |
| `lerp(a, b, t)` | Linear interpolation | `lerp(0, 10, 0.5)` → `5` |
| `map(v, inMin, inMax, outMin, outMax)` | Re-map value between ranges | `map(5, 0, 10, 0, 100)` → `50` |

### Array Methods (unchanged)
Existing array methods continue to work:
```parsley
[1, 2, 3].min()   // → 1
[1, 2, 3].max()   // → 3
[1, 2, 3].sum()   // → 6
```

## Migration
The following math builtins have been removed: `sin`, `cos`, `tan`, `asin`, `acos`, `atan`, `sqrt`, `pow`, `round`, `pi`. Users must import `std/math` to access these functions.

**Before:**
```parsley
let angle = pi() / 4
let result = sin(angle)
```

**After:**
```parsley
let math = import("std/math")
let angle = math.PI / 4
let result = math.sin(angle)
```

## Out of Scope
- Advanced statistics (percentile, quartile, correlation, zscore)
- Hyperbolic functions (sinh, cosh, tanh)
- Special functions (gamma, factorial)
- Complex numbers
- Matrix operations

These may be added in future versions based on demand.

## References
- Python `math` module: https://docs.python.org/3/library/math.html
- Go `math` package: https://pkg.go.dev/math
