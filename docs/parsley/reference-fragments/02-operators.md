## 2. Operators

### 2.1 Arithmetic

| Operator | Description | Example |
|----------|-------------|---------|
| `+` | Addition | `5 + 3` → `8` |
| `-` | Subtraction | `10 - 4` → `6` |
| `*` | Multiplication | `6 * 7` → `42` |
| `/` | Division | `20 / 4` → `5` |
| `%` | Modulo | `17 % 5` → `2` |
| `-` | Negation (prefix) | `-42` |

#### String Operations

```parsley
"Hello" + " " + "World"         // Concatenation
"ab" * 3                        // "ababab" (repetition)
```

#### Array Operations

```parsley
let repeated = [1, 2] * 2       // [1, 2, 1, 2]
let chunked = [1,2,3,4,5,6] / 2 // [[1,2], [3,4], [5,6]]
```

---

### 2.2 Comparison

| Operator | Description |
|----------|-------------|
| `==` | Equal |
| `!=` | Not equal |
| `<` | Less than |
| `>` | Greater than |
| `<=` | Less than or equal |
| `>=` | Greater than or equal |

Standard boolean operators, e.g.:

```parsley
let age = 20
let status = if (age >= 18) "adult" else "minor"  // "adult"
```

---

### 2.3 Logical / Set Operations

| Operator | Description |
|----------|-------------|
| `&&` | Logical AND (short-circuit) |
| `\|\|` | Logical OR (short-circuit) |
| `!` | Logical NOT |
| `and` | Keyword alias for `&&` |
| `or` | Keyword alias for `\|\|` |

#### Boolean Logic

For boolean values, these operators work as expected:

```parsley
true && true                    // true
false || true                   // true
let notResult = !false          // true
true and true                   // true
false or true                   // true
```

#### Set Operations on Collections

The logical operators are overloaded to perform set operations when applied to arrays: AND corresponds to intersection; OR corresponds to union.

| Operator | Set Operation | Description |
|----------|---------------|-------------|
| `&&` | Intersection | Elements present in both arrays |
| `\|\|` or `\|` | Union | Elements present in either array (duplicates removed) |

```parsley
// Intersection: elements in BOTH arrays
([1, 2, 3] && [2, 3, 4]).toJSON()    // [2, 3]

// Union: elements in EITHER array (duplicates removed)
([1, 2, 3] || [3, 4, 5]).toJSON()    // [1, 2, 3, 4, 5]
([1, 2] | [2, 3]).toJSON()           // [1, 2, 3]
```

This is useful for filtering, merging lists, and finding common elements without writing explicit loops.

---

### 2.4 Membership

| Operator | Description |
|----------|-------------|
| `in` | Membership test |
| `not in` | Negated membership |

```parsley
2 in [1, 2, 3]                  // true
"name" in {name: "Sam"}         // true (key exists)
"ell" in "hello"                // true (substring)
"x" not in [1, 2, 3]            // true
```

---

### 2.5 Schema Checking

The `is` and `is not` operators check whether a value is a Record or Table bound to a specific schema.

| Operator | Description |
|----------|-------------|
| `is` | Schema identity check (returns boolean) |
| `is not` | Negated schema check (returns boolean) |

```parsley
@schema User { name: string }
@schema Product { sku: string }

let user = User({name: "Alice"})

user is User                    // true
user is Product                 // false
user is not Product             // true
```

**Identity comparison**: Schema checking uses pointer identity, not structural matching. Two schemas with identical fields are still different schemas:

```parsley
@schema UserA { name: string }
@schema UserB { name: string }  // Same fields, different schema

let record = UserA({name: "Bob"})
record is UserA                 // true
record is UserB                 // false (different schema)
```

**Works with Tables too**:

```parsley
@schema Point { x: int, y: int }
let points = @table(Point) [{x: 1, y: 2}]

points is Point                 // true
```

**Non-record values**: For values that aren't Records or Tables (strings, numbers, plain dicts, arrays, etc.), `is` safely returns `false`:

```parsley
"hello" is User                 // false
42 is User                      // false
{name: "Alice"} is User         // false (plain dict, not a Record)
```

**Error case**: The right-hand side must be a schema. Using a non-schema value produces a TypeError:

```parsley
user is 42                      // Error: 'is' requires a schema
user is "User"                  // Error: 'is' requires a schema
```

---

### 2.6 Pattern Matching

The `~` and `!~` operators perform regex matching, similar to Perl's pattern matching syntax.

| Operator | Description |
|----------|-------------|
| `~` | Regex match (returns first match or null) |
| `!~` | Regex not match (returns boolean) |

```parsley
"hello123" ~ /\d+/              // "123" (first match)
"hello" ~ /\d+/                 // null (no match)
"hello" !~ /\d+/                // true (does not match)
"abc123" !~ /\d+/               // false (does match)
```

---

### 2.7 Range

The range operator `..` creates an inclusive sequence of integers from start to end.

```parsley
let range = 1..5                // [1, 2, 3, 4, 5]
let countdown = 5..1            // [5, 4, 3, 2, 1] (descending)
```

**Eager Evaluation**: Ranges are evaluated immediately into arrays. The entire sequence is generated in memory when the expression is evaluated, not lazily on demand. For very large ranges, be mindful of memory usage.

```parsley
1..1000000                      // Creates array with 1 million elements
```

---

### 2.8 Concatenation

```parsley
let concat = [1, 2] ++ [3, 4]   // [1, 2, 3, 4]
let merged = {a: 1} ++ {b: 2}   // {a: 1, b: 2}
```

---

### 2.9 Null Coalescing

The `??` operator returns the right-hand value when the left-hand value is `null`. This provides a concise way to supply default values.

```parsley
null ?? "default"               // "default" (left is null, use right)
"value" ?? "default"            // "value" (left is not null, use left)
0 ?? "default"                  // 0 (0 is not null)
"" ?? "default"                 // "" (empty string is not null)
```

**Note**: Unlike truthiness checks, `??` only triggers on `null`, not on other falsy values like `0`, `""`, or `[]`.

#### Optional Index Access

Use `[?index]` syntax to safely access array or dictionary elements without errors when the index/key doesn't exist, or is out of range:

```parsley
let arr = [1, 2, 3]
arr[?99]                        // null (index out of bounds, no error)
arr[?0]                         // 1 (valid index)

let user = {name: "Alice"}
user[?"email"]                  // null (key doesn't exist, no error)
```

Without `?`, out-of-bounds access would produce an error.

---

### 2.10 DateTime Arithmetic

Parsley supports arithmetic operations on dates, times, and durations with sensible rules.

#### Valid Operations

| Operation | Result | Example |
|-----------|--------|--------|
| datetime + duration | datetime | `@now + @1d` → tomorrow |
| datetime - duration | datetime | `@now - @1w` → one week ago |
| datetime - datetime | duration | `@2024-12-25 - @2024-12-20` → 5 days |
| duration + duration | duration | `@1d + @2h` → 1 day 2 hours |
| duration - duration | duration | `@1w - @2d` → 5 days |
| duration * number | duration | `@1d * 3` → 3 days |
| date && time | datetime | `@2024-12-25 && @14:30` → datetime |

```parsley
@now + @1d                      // Tomorrow
@now - @1w                      // One week ago
@2024-12-25 - @2024-12-20       // 5 days (duration)
@2024-12-25 && @14:30           // 2024-12-25T14:30:00
@1d + @1d                       // 2 days
@1d * 3                         // 3 days
```

#### Invalid Operations

Some operations don't make sense and will produce errors:

```parsley
@2024-12-25 + @2024-12-20       // Error: can't add two dates
3 * @1d                         // Error: number must be on the right
```

**Tip**: Think of durations as time offsets. You can add/subtract offsets from dates, or multiply offsets by numbers, but adding two absolute dates together is meaningless.

---

### 2.11 Unit Arithmetic

Parsley supports arithmetic on measurement units with exact integer storage and cross-system conversion.

#### Valid Operations

| Operation | Result | Example |
|-----------|--------|---------|
| unit + unit (same family) | unit | `#5cm + #3mm` → `#5.3cm` |
| unit - unit (same family) | unit | `#1ft - #6in` → `#1/2ft` |
| unit * number | unit | `#5m * 3` → `#15m` |
| number * unit | unit | `3 * #5m` → `#15m` |
| unit / number | unit | `#10m / 3` → `#3.33m` |
| unit / unit (same family) | number | `#10m / #5m` → `2` |
| length * length | area | `#5m * #3m` → `#15m2` |
| area / length | length | `#15m2 / #3m` → `#5m` |
| -unit | unit | `-#6m` → `#-6m` |

```parsley
#5cm + #3mm                     // #5.3cm
#12.3m + #0.7m                  // #13m
#3/8in + #5/8in                 // #1in
#1ft - #6in                     // #1/2ft
#5m * 3                         // #15m
#10m / #5m                      // 2
#1/3cup + #1/3cup + #1/3cup     // #1cup (exact)
#1gal / #1qt                    // 4
#100m2 + #50m2                  // #150m2 (area)
#2kL * 3                        // #6kL (kilolitres)
```

#### Derived Unit Arithmetic (Length × Length → Area)

Multiplying two length values produces an area. Dividing an area by a length produces a length:

```parsley
// Length × Length → Area
#5m * #3m                       // #15m2
#2ft * #3ft                     // #6ft2
#100cm * #50cm                  // #5000cm2
#1km * #1km                     // #1km2
#12in * #12in                   // #144in2

// Area / Length → Length
#15m2 / #3m                     // #5m
#6ft2 / #2ft                    // #3ft
#144in2 / #12in                 // #12in

// Round-trip: (L × L) / L = L
(#5m * #3m) / #3m               // #5m
```

**Display hint rules:**
- **Multiplication**: The left operand determines the result's display hint. `#5m * #300cm` → `#15m2` (not cm²).
- **Division**: The divisor (right operand) determines the result's display hint. `#1km2 / #1000m` → `#1000m`.

**Cross-system restriction**: Length × length and area / length only work within the same system:

```parsley
#5m * #3ft                      // Error: Cannot multiply SI length by US length
#15m2 / #3ft                    // Error: Cannot divide SI area by US length
```

> Other derived units (speed, volume from length³, etc.) are not yet supported.

#### Temperature Arithmetic

Temperature addition and subtraction use "treat like numbers" semantics with offset-based formulas internally. Cross-scale operations convert the right operand to the left's scale:

```parsley
#20C + #10C                     // #30C
#100C - #37C                    // #63C
#212F - #32F                    // #180F
```

**Temperature comparisons** work across scales because all temperatures share the same internal sub-kelvin representation:

```parsley
#0C == #32F                     // true
#100C == #212F                  // true
#-40C == #-40F                  // true (scales cross at -40)
#0K == #-273.15C                // true (absolute zero)
#100C > #200F                   // true (#100C = #212F)
```

**Temperature multiply/divide is forbidden** — these operations are undefined for offset scales:

```parsley
#20C * 2                        // Error: Cannot multiply a temperature
#100F / 2                       // Error: Cannot divide a temperature
#100C / #50C                    // Error: Cannot divide temperature by temperature
```

> Negation (`-#20C`) is allowed — it negates the internal sub-kelvin value.

#### Cross-System Arithmetic

Units from the same family but different systems are converted automatically. The **left operand wins** — the result uses the left operand's system and display hint:

```parsley
#1cm + #1in                     // #3.54cm (SI result)
#1in + #1cm                     // #1.39...in (US result)
#1L + #1floz                    // result in litres (SI)
```

Rounding occurs only at the SI↔US boundary. Within-system arithmetic is always exact.

#### Invalid Operations

```parsley
#5m + #5kg                      // Error: cannot add length to mass
#5kg * #3kg                     // Error: cannot multiply mass by mass (only length × length → area)
#5m2 + #5kg                     // Error: cannot add area to mass
5 + #5m                         // Error: cannot add number to unit
10 / #5m                        // Error: cannot divide number by unit
#1L + #1C                       // Error: cannot add volume to temperature
#5m * #3ft                      // Error: cannot multiply SI length by US length
```

---

### 2.12 Precedence Table (Lowest to Highest)

| Level | Operators |
|-------|-----------|
| 1 | `??`, `\|\|`, `or` |
| 2 | `&&`, `and` |
| 3 | `==`, `!=`, `~`, `!~`, `in`, `not in`, `is`, `is not` |
| 4 | `<`, `>`, `<=`, `>=` |
| 5 | `+`, `-`, `..` |
| 6 | `++` |
| 7 | `*`, `/`, `%` |
| 8 | `-`, `!` (prefix) |
| 9 | `.`, `[]`, `()` (access/call) |
| — | `<==`, `==>`, `==>>` (file I/O, statement-level) |
| — | `<=/=` (fetch, statement or expression), `=/=>`, `=/=>>` (remote write, statement or expression) |

---

