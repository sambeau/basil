# Design: Precise Measurement Units for Parsley

## Status

**Stage:** Discussion / Design  
**Created:** 2025-06-28  
**Author:** @sam, @copilot  

This document explores adding unit-of-measurement types to Parsley, following the precedent set by the Money type. The goal is to refine the idea before moving to a formal specification.

---

## 1. Overview

### 1.1 Motivation

Parsley's Money type demonstrates that building precision arithmetic into the language removes boilerplate and eliminates entire classes of bugs. The same opportunity exists for units of measurement.

Scripts that deal with length, weight, temperature, or data sizes currently require the developer to:

- Choose between floats (imprecise) or integers (awkward scaling)
- Manually track significant figures
- Hand-code conversion logic between unit scales (mm → m → km)
- Hand-code conversion between measurement systems (Imperial ↔ SI)
- Handle edge cases: fractions (1/4"), temperature offsets, IEC vs SI data units

This is tedious and error-prone. A built-in unit type would make domestic and small-business scripts dramatically simpler — selling carpet by the square metre, calculating firewood by the cord, sizing storage in GiB.

### 1.2 Design Goals

- **Precise:** Integer storage with defined scaling, no floating-point drift
- **Safe:** Prevent nonsensical operations (adding kilograms to metres)
- **Convertible:** Seamless, lossless conversion within a unit family
- **Ergonomic:** Literals, constructors, and operators that feel natural
- **Minimal:** Only the units people actually need, with room to grow
- **Consistent:** Follow the patterns established by Money

### 1.3 Prior Art

| System | Approach | Notes |
|--------|----------|-------|
| F# Units of Measure | Compile-time type checking | Gold standard for safety, requires static types |
| Frink | Language built around units | Powerful but units are the entire point |
| Pint (Python) | Runtime library, float storage | Popular API design, but imprecise |
| Boost.Units (C++) | Template-based, compile-time | Thorough but complex |
| CSS | Suffix syntax (`12px`, `1em`) | Closest to what Parsley would do |
| Java JSR 385 | Runtime library | Good API, verbose |

The most relevant precedents for Parsley are CSS (suffix syntax, runtime evaluation) and Pint (conversion methods, operator overloading), filtered through Money's integer-storage philosophy.

---

## 2. The Money Precedent

Understanding how Money works is key, because Units would follow the same pattern.

### 2.1 Money's Internal Shape

```parsley
// What the user writes
let price = $19.99

// What Parsley stores internally
{
    Amount:   1999    // int64, in cents (smallest unit)
    Currency: "USD"   // discriminator
    Scale:    2       // decimal places for display
}
```

### 2.2 What Money Provides

- **Literals:** `$12.34`, `£99.99`, `EUR#50.00`
- **Constructor:** `money(19.99, "USD")`
- **Operators:** `+`, `-` (same currency), `*`, `/` (with scalars)
- **Properties:** `.amount`, `.currency`, `.scale`
- **Methods:** `.abs()`, `.format()`, `.split()`, `.repr()`, `.toDict()`, `.inspect()`
- **Safety:** Cross-currency arithmetic is an error

A Unit type would provide analogous features, with the addition of cross-unit conversion within a family.

---

## 3. Literal Syntax

### 3.1 Proposal: `#` Prefix

The proposal suggests:

```
#23in       // 23 inches
#12.3m      // 12.3 metres
#45.6yd     // 45.6 yards
#100mi      // 100 miles
#10kg       // 10 kilograms
#64KB       // 64 kilobytes
```

### 3.2 Collision with Money's `CODE#` Format

Money already uses `#` in the `CODE#amount` pattern:

```parsley
EUR#50.00   // Money: code THEN hash THEN number
#50m        // Unit: hash THEN number THEN suffix
```

**Can the lexer disambiguate?** Yes. The patterns are structurally different:

| Pattern | Structure | Example |
|---------|-----------|---------|
| Money | `UPPER{2,3}#NUMBER` | `EUR#50.00` |
| Unit | `#NUMBER LOWER+` | `#50m` |

The lexer already handles context-dependent tokenization (e.g., `$` for money vs identifiers). Adding `#` + number → unit is comparable in complexity.

**Is it confusing for users?** Possibly. The `#` serves opposite roles: in money it separates a code from a value; in units it acts as a type sigil. But the visual pattern is distinct enough that in practice it should be clear. Something to validate with user testing.

### 3.3 Alternative Syntaxes Considered

| Syntax | Example | Pros | Cons |
|--------|---------|------|------|
| `#number+suffix` | `#12.3m` | Clear sigil, unambiguous parse | `#` collision with money |
| Bare suffix | `12.3m` | Clean, CSS-like | Ambiguous with identifiers (`m` as a variable) |
| Backtick suffix | `` 12.3`m` `` | Unambiguous | Ugly, unfamiliar |
| Tilde prefix | `~12.3m` | No collisions | `~` has no semantic connection to units |
| `@unit` prefix | `@12.3m` | Consistent with `@datetime` | `@` is already overloaded |

**Recommendation:** Keep `#number+suffix` as proposed. The lexer can handle it, and it establishes `#` as a "typed value" sigil (money uses it too, just differently). The visual distinction between `EUR#50` and `#50m` is clear enough.

### 3.4 Negative Values

Negative units follow the same pattern as negative numbers:

```parsley
-#6C        // negative 6 degrees Celsius
let drop = -#15m   // 15 metres below
```

Or with arithmetic:

```parsley
#0C - #6C   // -#6C
```

### 3.5 The Unit Suffix Table

These are the suffixes that would appear in literals:

#### Length
| Suffix | Unit | System |
|--------|------|--------|
| `mm` | millimetre | SI |
| `cm` | centimetre | SI |
| `m` | metre | SI |
| `km` | kilometre | SI |
| `in` | inch | US |
| `ft` | foot | US |
| `yd` | yard | US |
| `mi` | mile | US |

#### Mass
| Suffix | Unit | System |
|--------|------|--------|
| `mg` | milligram | SI |
| `g` | gram | SI |
| `kg` | kilogram | SI |
| `oz` | ounce | US |
| `lb` | pound | US |

#### Temperature
| Suffix | Unit | System |
|--------|------|--------|
| `K` | kelvin | SI |
| `C` | degree Celsius | SI |
| `F` | degree Fahrenheit | US |

#### Digital Information
| Suffix | Unit | System |
|--------|------|--------|
| `B` | byte | — |
| `kB` | kilobyte (1000) | SI |
| `MB` | megabyte (1000²) | SI |
| `GB` | gigabyte (1000³) | SI |
| `TB` | terabyte (1000⁴) | SI |
| `KiB` | kibibyte (1024) | IEC |
| `MiB` | mebibyte (1024²) | IEC |
| `GiB` | gibibyte (1024³) | IEC |
| `TiB` | tebibyte (1024⁴) | IEC |

#### Volume (Phase 2)
| Suffix | Unit | System |
|--------|------|--------|
| `mL` | millilitre | SI |
| `L` | litre | SI |
| `floz` | fluid ounce | US |
| `cup` | cup | US |
| `pt` | pint | US |
| `qt` | quart | US |
| `gal` | gallon | US |

#### Area (Phase 2)
| Suffix | Unit | System |
|--------|------|--------|
| `sqmm` | square millimetre | SI |
| `sqcm` | square centimetre | SI |
| `sqm` | square metre | SI |
| `sqkm` | square kilometre | SI |
| `sqin` | square inch | US |
| `sqft` | square foot | US |
| `sqyd` | square yard | US |
| `ac` | acre | US |
| `sqmi` | square mile | US |

> **Open question:** Should area use `sqm` or `m2`? The former is unambiguous to the lexer; the latter is more conventional but `2` after a unit suffix is unusual in Parsley.

---

## 4. Internal Representation

The biggest design question for this feature is how values are stored internally. Getting this right determines whether fractions work, whether round-trips are exact, and whether arithmetic is fast.

This section has gone through several iterations. The key insight driving the current design: **SI and Imperial units have fundamentally different numeric idioms, and they deserve different internal representations.**

### 4.1 The Core Problem

SI units are decimal. People write `12.3m`, `2.54cm`, `0.5kg`. Powers of 10 everywhere. A decimal integer + scale (like Money) is a natural fit.

Imperial units are fractional. People write `3/8"`, `1/3 cup`, `5/16"`. The denominators are small integers — 2, 3, 4, 5, 6, 7, 8, 9, 10, 12, 16 — that don't relate to powers of 10. A decimal representation forces these into repeating decimals (0.333..., 0.1666...), destroying exactness.

A single representation can't serve both idioms well:

- **Decimal scale** (like Money): Perfect for SI. Breaks Imperial fractions — `#1/3in + #1/3in + #1/3in ≠ #1in` because 0.333... × 3 ≠ 1.
- **Shared base unit** (e.g., µm): Perfect for SI. Breaks Imperial round-trips — 1 oz = 28,349,523.125 µg, so `#1oz` becomes `#0.999999996oz`.
- **Rational arithmetic** (numerator/denominator pairs): Exact for both systems but makes all arithmetic expensive (GCD on every operation, denominator explosion in chains of operations).

### 4.2 The Proposal: Two Representations, One Interface

Use the right tool for each system:

- **SI values** use **decimal scaling** (integer + power-of-10 scale), identical to Money
- **Imperial values** use **HCN scaling** (integer numerator over a fixed highly composite denominator), making common fractions exact as plain integers

Both present the same interface to the user: same literals, same operators, same methods. The difference is purely internal.

### 4.3 Why a Highly Composite Denominator? And How Big?

If we pick a fixed denominator that is divisible by every common fraction denominator, then every common fraction becomes an exact integer division. The question is: how big should that denominator be?

Since SI uses micrometre precision (10⁶ subdivisions per metre), and 1 yard ≈ 1 metre (0.9144 m exactly), we should aim for a comparable number of subdivisions per yard — roughly 914,400. This gives both systems similar precision and range, and makes the cross-system bridge ratio (0.9144) lose minimal relative precision.

But we can't just pick 914,400 — the HCN must have the right factors. Specifically, since 1 inch = 1/36 yard, for a fraction 1/q of an inch to be exact, the HCN per yard must be divisible by 36 × q.

**Deriving the HCN:**

For all fractions up to 64ths of an inch, plus thirds, fifths, sevenths, and ninths:

```
Need HCN per yard divisible by:
  36 × 64 = 2,304   (for 64ths of an inch)
  36 × 9  = 324     (for 9ths)
  36 × 7  = 252     (for 7ths)
  36 × 5  = 180     (for 5ths)

LCM(2304, 324, 252, 180):
  2304 = 2⁸ × 3²
  324  = 2² × 3⁴
  252  = 2² × 3² × 7
  180  = 2² × 3² × 5

LCM = 2⁸ × 3⁴ × 5 × 7 = 256 × 81 × 5 × 7 = 725,760
```

**725,760 sub-yards per yard.** This gives 725,760 / 36 = **20,160 sub-units per inch**.

| Fraction of inch | × 20,160 | Exact? | Where it appears |
|------------------|----------|--------|------------------|
| 1/2 | 10,080 | ✓ | everywhere |
| 1/3 | 6,720 | ✓ | cooking (1/3 cup), sewing |
| 1/4 | 5,040 | ✓ | cooking, carpentry |
| 1/5 | 4,032 | ✓ | — |
| 1/6 | 3,360 | ✓ | cooking (1/6 recipe scale) |
| 1/7 | 2,880 | ✓ | — |
| 1/8 | 2,520 | ✓ | cooking (1/8 tsp), carpentry (1/8" kerf) |
| 1/9 | 2,240 | ✓ | — |
| 1/10 | 2,016 | ✓ | — |
| 1/12 | 1,680 | ✓ | inches per foot |
| 1/16 | 1,260 | ✓ | ounces per pound, tape measures |
| 1/32 | 630 | ✓ | precision woodworking |
| 1/64 | 315 | ✓ | machining, fine carpentry |
| 3/8 | 7,560 | ✓ | sewing (3/8" seam), carpentry |
| 5/8 | 12,600 | ✓ | timber framing (92-5/8" stud) |
| 5/16 | 6,300 | ✓ | wrenches, drill bits |

Every common fraction from halves to sixty-fourths — all exact integers. No "5040 or 10,080?" question. By anchoring to micrometre-scale precision, we get all of them for free.

**Precision:** 914,400 µm / 725,760 = **1.26 µm per sub-yard** — approximately one micrometre, matching SI precision.

**Range:** int64 max / 725,760 ≈ 1.27 × 10¹³ yards ≈ **77 AU** (sun to well past Pluto). Enormous.

**How common are these fractions?** They are the standard idiom in US domestic and trades contexts:

- **Cooking/baking:** Nearly every US recipe uses fractional cups and spoons — 1/3 cup, 2/3 cup, 1/4 tsp, 3/4 cup. A recipe-scaling script that turns 1/3 cup into 0.333333 cup has failed its user.
- **Timber framing:** Standard pre-cut studs are **92-5/8 inches**. Standard stud spacing is **16 inches on center** (sometimes 15-1/4" for drywall). Wall plates, headers, jacks — all specified in fractional inches. A framing calculator needs these to be exact.
- **Sewing/dressmaking:** Seam allowances are 5/8" or 3/8". Pattern adjustments are in 1/8" increments.
- **General construction:** Plywood is 3/4", drywall is 1/2" or 5/8", pipe is 3/8" or 1/2".

These are not edge cases — they're the primary way Imperial measurements are expressed in practice. Supporting them exactly is a core requirement, not a nice-to-have.

#### Why yard as the base unit?

Yard parallels metre. Both represent a similar physical scale (~1 m), and both systems end up with roughly micrometre-level precision. This symmetry is elegant: the two representations differ in numeric idiom (decimal vs fractional) but agree in scale and precision. The cross-system bridge ratio (1 yard = 0.9144 metres) is close to 1, so conversions lose minimal relative precision.

The choice of yard vs inch as "base" doesn't change the per-inch resolution — 20,160 sub-units per inch either way. What it changes is the conceptual framing: "metres for SI, yards for Imperial" is a cleaner parallel than "metres for SI, inches for Imperial."

| Family | SI base | Imperial base | Bridge ratio | Closeness |
|--------|---------|---------------|--------------|-----------|
| Length | metre | yard | 1 yd = 0.9144 m | ~9% |
| Volume | litre | quart | 1 qt ≈ 0.9464 L | ~5% |
| Mass | gram | ounce | 1 oz = 28.35 g | not close* |

\* Mass doesn't have a clean scale parallel, but the HCN approach works identically regardless. The ounce is the natural base because it's the smallest commonly-used Imperial mass unit, and 1 lb = 16 oz is an exact integer ratio.

### 4.4 The Object Shapes

#### SI Unit: Decimal (like Money)

```go
type SIUnit struct {
    Amount int64  // value = Amount / 10^Scale, in the base SI unit
    Scale  int8   // decimal places
    Family string // "length", "mass", "temperature", "data"
}
```

The base SI unit is the primary unit for each family: **metre** for length, **gram** for mass, **kelvin** for temperature, **byte** for data.

Examples:

| Literal | Stored as | Meaning |
|---------|-----------|---------|
| `#12.3m` | {Amount: 123, Scale: 1} | 12.3 metres |
| `#254cm` | {Amount: 254, Scale: 2} | 2.54 metres (254/100) |
| `#2.5mm` | {Amount: 25, Scale: 4} | 0.0025 metres (25/10000) |
| `#0.5kg` | {Amount: 5, Scale: 1} | 500 grams? or 0.5 × 1000g? |

Wait — the base-unit question matters here. If base is **metre**, then `#254cm` = 2.54 m = {Amount: 254, Scale: 2}. If base is **centimetre**, then `#254cm` = {Amount: 254, Scale: 0} but `#12.3m` = 1230 cm = {Amount: 1230, Scale: 0}. The metre-as-base approach normalises everything to a common scale, which is cleaner for arithmetic.

For mass, the base could be gram or kilogram. Gram is more practical: `#500g` = {Amount: 500, Scale: 0} is simpler than `#500g` = {Amount: 5, Scale: 1} (0.5 kg).

| Family | Base SI unit |
|--------|-------------|
| Length | metre (m) |
| Mass | gram (g) |
| Temperature | kelvin (K) — see §4.8 |
| Data | byte (B) |

#### Imperial Unit: HCN-Scaled Integer

```go
const HCN = 725_760 // 2⁸ × 3⁴ × 5 × 7 — sub-yards per yard

type ImperialUnit struct {
    Amount int64  // value = Amount / HCN, in yards (length), ounces (mass), etc.
    Family string // "length", "mass", "temperature"
}
```

The base Imperial unit parallels SI where possible: **yard** for length (≈ metre), **ounce** for mass, **degree Fahrenheit** for temperature, **quart** for volume (≈ litre).

All use the same HCN (725,760). The per-unit resolution depends on the base:
- Per inch: 725,760 / 36 = 20,160 (since 36 inches per yard)
- Per foot: 725,760 / 3 = 241,920 (since 3 feet per yard)
- Per ounce: 725,760 (same — ounce IS the base)
- Per cup: 725,760 (same — quart subdivided as needed)

Examples (length, HCN = 725,760 per yard, shown as sub-inches for clarity):

| Literal | Sub-inches | Calculation | Meaning |
|---------|-----------|-------------|---------|
| `#1in` | 20,160 | 1 × 20,160 | 1 inch |
| `#3/8in` | 7,560 | 3/8 × 20,160 | 3/8 inch |
| `#92-5/8in` | 1,867,320 | (92 + 5/8) × 20,160 | 92-5/8 inch (standard stud) |
| `#1ft` | 241,920 | 12 × 20,160 | 1 foot |
| `#2.5ft` | 604,800 | 2.5 × 12 × 20,160 | 2.5 feet |
| `#1yd` | 725,760 | 36 × 20,160 | 1 yard |
| `#16in` | 322,560 | 16 × 20,160 | 16 inch (standard OC spacing) |
| `#1mi` | 1,277,337,600 | 63,360 × 20,160 | 1 mile |

(Internally these are all stored as sub-yards — the sub-inch column is just for intuition. `#3/8in` = 7,560 sub-inches = 7,560 sub-yards too, since the sub-yard IS 1/36 of the sub-inch... actually no: Amount is in sub-yards, so `#3/8in` = 3/8 × 1/36 yard × 725,760 = 7,560 sub-yards. The number is the same.)

Examples (mass, HCN = 725,760 per ounce):

| Literal | Amount | Calculation | Meaning |
|---------|--------|-------------|---------|
| `#1oz` | 725,760 | 1 × 725,760 | 1 ounce |
| `#1lb` | 11,612,160 | 16 × 725,760 | 1 pound |
| `#1/3cup` | 241,920 | 1/3 × 725,760 | 1/3 cup |
| `#2/3cup` | 483,840 | 2/3 × 725,760 | 2/3 cup |

**Round-trip is always exact.** `#3/8in` → 7,560 → GCD(7560, 20160) = 2520 → 3/8 → `#3/8in`.

**Fraction arithmetic is just integer addition:**

```
#1/3cup + #1/3cup + #1/3cup
= 241920 + 241920 + 241920
= 725760
= 725760/725760
= 1 cup ✓ (exactly — this is the test case)
```

```
#3/8in + #5/8in
= 7560 + 12600
= 20160
= 20160/20160
= 1 inch ✓
```

Timber framing: how much stud material for a wall?

```
#92-5/8in * 9
= 1867320 × 9
= 16805880 sub-inches
GCD(16805880, 20160) = 2520 → 6670/8 = 833 5/8 inches ✓
```

Mixed units:

```
#12ft + #3/8in
= 2903040 + 7560
= 2910600 sub-inches (= sub-yards, since base is yard)
// To display in ft: 2910600 / 241920 = 12.03125 ft
// Integer feet: 12, remainder 7560 sub-inches = 3/8 inch
// → "12' 3/8\"" ✓
```

Note: with 725,760 the numbers are larger than with 5,040, but they're still well within int64 range. A mile is only 1.28 billion sub-yards — int64 can hold 9.2 × 10¹⁸.

### 4.5 Do We Need to Store the Display Unit?

The previous design stored the specific unit the user typed (`cm`, `ft`, `oz`). With base-unit storage, we have a choice: store it or derive it.

**Arguments for NOT storing it:**

- The value is "a length", not "a length in centimetres." The display unit is a formatting choice, like locale.
- It simplifies the object — fewer fields, fewer code paths.
- It enables natural multi-unit display: 727,650 sub-inches can display as `12' 3/8"` or `148 3/8"` or `4.125 yd` — all are the same value.
- For SI, any value in metres can be formatted as mm, cm, m, or km by shifting the decimal point.

**Arguments for storing it:**

- If the user writes `#254cm`, they probably expect to see `cm` in output, not `2.54m`.
- The "left side wins" rule needs to know what unit the left operand was in.
- Without it, we need a default display rule (always show in base unit? auto-scale?).

**Proposed compromise:** Store a **display hint** — the unit the user originally wrote — but don't make it part of the value's identity. Two values with different display hints but the same underlying amount are equal. The display hint is used for default formatting and the "left side wins" rule, but it's not a fundamental part of the type.

```go
type SIUnit struct {
    Amount      int64
    Scale       int8
    Family      string
    DisplayHint string // "cm", "m", "km", etc. — for default display
}

type ImperialUnit struct {
    Amount      int64
    Family      string
    DisplayHint string // "in", "ft", "yd", "mi", etc.
}
```

The display hint is set at creation time and propagated by the "left side wins" rule. The user can override it with `.to("cm")` or `.format("ft-in")`.

This means:

```parsley
#254cm + #1m        // display hint "cm" wins → #354cm
#1m + #254cm        // display hint "m" wins → #3.54m
#254cm == #2.54m    // true — same underlying value
```

> **Open question:** Is the display hint worth the complexity? The alternative is simpler: always display in the base unit (metres, grams, inches, ounces) and let users explicitly format. This is "purer" but arguably worse UX for casual use.

### 4.6 Range Analysis

#### SI (decimal)

With int64 and the base unit as metre:
- Scale 6 (micrometre precision): max = 9.2 × 10¹⁸ / 10⁶ = 9.2 × 10¹² m ≈ 61 AU. Enormous.
- Scale 9 (nanometre precision): max = 9.2 × 10⁹ m ≈ 9.2 million km. Still huge.

With gram as base, Scale 6: max ≈ 9.2 × 10¹² g = 9,200 tonnes. Plenty.

#### Imperial (HCN = 725,760)

With yard as base:
- 1 mile = 1,760 × 725,760 = 1,277,337,600
- int64 max / 1,277,337,600 ≈ 7.2 × 10⁹ miles ≈ **77 AU**

With ounce as base:
- 1 ton (2000 lb) = 2000 × 16 × 725,760 = 23,224,320,000
- int64 max / 23,224,320,000 ≈ 3.96 × 10⁸ tons. Hundreds of millions of tons.

**Range is not a concern.** Both systems extend well beyond the solar system for length and well beyond practical mass for weight. The SI and Imperial ranges are comparable — a natural consequence of choosing similar-scale base units (metre ≈ yard) with similar precision (~1 µm).

| Family | SI range | Imperial range |
|--------|----------|----------------|
| Length | ~61 AU (at µm precision) | ~77 AU |
| Mass | ~9,200 tonnes (at µg precision) | ~396 million tons |

### 4.7 How Arithmetic Works

#### Same system, same family — integer addition

Both representations reduce same-system arithmetic to integer operations:

**SI:**

```parsley
#12.3m + #0.7m
// Both in metres. Promote to common scale:
// {Amount: 123, Scale: 1} + {Amount: 7, Scale: 1}
// = {Amount: 130, Scale: 1} → #13m ✓
```

**Imperial:**
```parsley
#3/8in + #5/8in
// Both in sub-inches: 7560 + 12600 = 20160 → #1in ✓

#1lb + #4oz
// Convert lb to sub-ounces: 1 × 16 × 725760 = 11612160
// 4oz = 4 × 725760 = 2903040
// 11612160 + 2903040 = 14515200 → 14515200/725760 = 20 oz = 1.25 lb ✓
```

Within each system, ALL arithmetic is exact. No rounding, ever.

#### Cross-system — convert at the boundary

When an SI value meets an Imperial value in an operation, one must be converted to the other. The left operand determines the result system.

The bridge ratios are the international definitions:
- 1 in = 0.0254 m (exact)
- 1 lb = 453.59237 g (exact)

**Imperial → SI** (converting Imperial to decimal):

```parsley
#10cm + #1in
// Convert #1in to metres: 1 × 0.0254 = 0.0254 m
// SI: {Amount: 254, Scale: 4} (0.0254 m)
// #10cm = {Amount: 1, Scale: 1} (0.1 m)
// Promote: {Amount: 1000, Scale: 4} + {Amount: 254, Scale: 4}
// = {Amount: 1254, Scale: 4} → 0.1254 m
// Display with hint "cm": #12.54cm ✓ (exact!)
```

**SI → Imperial** (converting decimal to HCN):

```parsley
#1in + #1cm
// Convert #1cm to sub-inches: 0.01m / 0.0254m × 20160 = 7937.00787...
// Must round to integer: 7937 sub-inches
// 20160 + 7937 = 28097 sub-inches
// GCD(28097, 20160) = 1 — no clean fraction
// Display: ~#1.3937in (decimal fallback)
```

This is where rounding lives — at the SI↔Imperial boundary. And it's the ONLY place.

Note the asymmetry: Imperial → SI is often exact (because the bridge ratios are defined as finite decimals in SI). SI → Imperial is where repeating decimals and rounding appear. This matches reality — the definitions go that direction.

With 725,760 sub-yards (20,160 sub-inches), the rounding error on SI → Imperial conversion is at most 0.5 sub-inches = 0.63 µm — sub-micrometre precision. The larger HCN means even the rounding residue is negligible.

> **Open question:** When an SI → Imperial conversion doesn't land on a clean fraction, should the display fall back to decimal (`#1.3937in`) or show the nearest common fraction (`≈ #1 3/8in`)? Decimal is exact (to the stored precision). Nearest-fraction is more natural for Imperial users but introduces a display-level approximation. Probably should offer both via formatting options.

#### Scalar operations

```parsley
#3/8in * 4      // 7560 * 4 = 30240 sub-inches = 1.5 in = #1-1/2in ✓
#12.3m / 2      // {Amount: 123, Scale: 1} → 123/2 = 61.5 → {Amount: 615, Scale: 2} → #6.15m ✓
#10kg / 3       // {Amount: 10000, Scale: 3} → ... 3333.33 → {Amount: 3333, Scale: 3} → #3.333kg
                // (truncation — same as Money)
```

For Imperial, scalar multiplication preserves exactness when the result is an integer number of sub-units (which it always is for integer scalars). Scalar division may require rounding if the result isn't an integer number of sub-units — but 725,760 is divisible by every integer from 1 to 10, plus 12, 14, 15, 16, 18, 20, 21, 24, 25, 27, 28, 32, 35, 36, 42, 45, 48, 56, 63, 64, and many more. In practice, virtually every common division is exact.

#### Unit-to-unit division — returns a plain number

```parsley
#10km / #2km    // 5 (dimensionless)
#1mi / #1km     // 1.609344
#3/8in / #1in   // 0.375 (or the fraction 3/8 as a float)
```

### 4.8 Temperature

Temperature is special in two ways: it has offset conversions (not just scaling), and the Fahrenheit ↔ Celsius ratio is 9/5 — which introduces repeating decimals.

**Imperial (Fahrenheit):** HCN = 725,760. Since 725,760 is divisible by both 5 and 9:
- 1°F = 725,760 sub-degrees
- 5/9 of a degree F = 725,760 × 5/9 = 403,200 sub-degrees (exact!)
- This means the "5/9" in the conversion formula stays exact in the HCN representation.

**SI (Celsius/Kelvin):** Decimal scale, as usual. The 5/9 ratio produces repeating decimals when going F→C, which is expected and handled by max scale truncation.

**Conversion:**
```parsley
fahrenheit(#100C)
// 100 × 9/5 + 32 = 212°F
// Imperial: 212 × 725,760 = 153,861,120 sub-degrees → #212F ✓

celsius(#72F)
// (72 − 32) × 5/9 = 22.222...°C
// SI: {Amount: 222222, Scale: 4} → #22.2222C
// (repeating decimal truncated — inherent to the conversion)
```

Storing Fahrenheit in HCN-scaled sub-degrees means that common fractional temperatures (oven at 350°F, thermostat at 68°F) are always exact, and the 5/9 conversion factor can be applied as integer multiplication before division, minimising precision loss. With 725,760 sub-degrees per °F, fractional temperatures (rare but possible — some candy-making recipes specify 1/2 degree) are also exact.

### 4.9 Digital Information

Digital units don't have an SI/Imperial split. All conversions are exact integer ratios (powers of 10 for SI prefixes, powers of 2 for IEC prefixes). They fit cleanly into the SI decimal model:

| From | To bytes | Ratio |
|------|----------|-------|
| B | 1 | × 1 |
| kB | 1,000 | × 10³ |
| MB | 1,000,000 | × 10⁶ |
| GB | 1,000,000,000 | × 10⁹ |
| TB | 1,000,000,000,000 | × 10¹² |
| KiB | 1,024 | × 2¹⁰ |
| MiB | 1,048,576 | × 2²⁰ |
| GiB | 1,073,741,824 | × 2³⁰ |
| TiB | 1,099,511,627,776 | × 2⁴⁰ |

These use the SI unit type internally. The IEC ratios (powers of 2) aren't powers of 10, but they're exact integers, so converting KiB → B is just multiplication — no rounding.

> **Open question:** Since IEC ratios aren't powers of 10, should digital units use their own representation? Or is it simpler to just store everything in bytes (base unit) with a decimal scale for the amount? E.g., `#2.5GB` = {Amount: 2500000000, Scale: 0, DisplayHint: "GB"}. The values are always whole bytes, so Scale would typically be 0. This is arguably the simplest approach.

### 4.10 Within-System Conversion Ratios

These are the exact ratios used for within-system arithmetic. All are integers or simple fractions — no rounding.

#### Length

| SI | → metres | | US | → inches |
|----|----------|---|----|----------|
| mm | × 1/1000 | | in | × 1 |
| cm | × 1/100 | | ft | × 12 |
| m | × 1 | | yd | × 36 |
| km | × 1000 | | mi | × 63,360 |

Cross-system bridge: **1 in = 0.0254 m** (exact, by international definition).

#### Mass

| SI | → grams | | US | → ounces |
|----|---------|---|----|----------|
| mg | × 1/1000 | | oz | × 1 |
| g | × 1 | | lb | × 16 |
| kg | × 1000 | | | |

Cross-system bridge: **1 lb = 453.59237 g** (exact, by international definition).

#### Volume (Phase 2)

| SI | → litres | | US | → fluid ounces |
|----|----------|---|----|----------------|
| mL | × 1/1000 | | fl oz | × 1 |
| L | × 1 | | cup | × 8 |
| | | | pt | × 16 |
| | | | qt | × 32 |
| | | | gal | × 128 |

Cross-system bridge: **1 gal = 3.785411784 L** (exact, by definition: 231 in³).

Note how clean the US volume ratios are — all powers of 2. With HCN = 5040, 1/3 cup = 1/3 × 8 × 5040 = 13,440 sub-fluid-ounces. Exact.

### 4.11 Summary

| Aspect | SI Units | Imperial Units |
|--------|----------|----------------|
| Storage | int64 + decimal scale | int64 / 725,760 |
| Idiom | Decimal (12.3, 0.254) | Fractional (3/8, 1/3, 5/16) |
| Precision | ~1 µm (at Scale 6) | ~1.26 µm (914,400/725,760) |
| Base unit | metre, gram, kelvin, byte | yard, ounce, °F, quart |
| Within-system arithmetic | Exact (powers of 10) | Exact (integer ratios + HCN) |
| Common fractions | Repeating decimals | Exact integers |
| Display | Decimal with unit suffix | Fractions or decimal, user's choice |
| Cross-system conversion | Rounding at boundary | Rounding at boundary |
| Range | ~61 AU / 9,200 tonnes | ~77 AU / 396M tons |
| Display hint | Stored for default formatting | Stored for default formatting |

---

## 5. The Temperature Problem

Temperature deserves its own section because it's fundamentally different from other units.

### 5.1 Point vs. Interval

Most units represent *quantities*: 5 kg is an amount, 10 m is a distance. Temperature can be either:

- **A point on a scale:** "It's 20°C outside" (absolute temperature)
- **A difference:** "The temperature rose by 5°C" (temperature interval)

This matters for arithmetic:

```parsley
// Points: what does this mean?
#20C + #10C     // 30°C? That's "adding" two thermometer readings
#20C * 2        // 40°C? Doubling a temperature isn't physically meaningful

// Intervals: this makes sense
#5C + #3C       // 8°C rise — sum of two changes
#5C * 2         // 10°C — double the change
```

### 5.2 Options

**Option A: Temperature is always a point.** Arithmetic is restricted:
- `#20C + #5C` → `#25C` (point + interval, but how do we know which is which?)
- `#20C * 2` → error (can't scale a temperature point)
- This requires distinguishing points from intervals, which adds complexity

**Option B: Temperature is always an interval (delta).** Users must use constructors for absolute temperatures:
- `#5C` means "5 degrees Celsius" as a difference
- `celsius(20)` means "20°C on the thermometer"
- This is precise but unintuitive for common use ("what's the boiling point? `celsius(100)` not `#100C`")

**Option C: Temperature is a point by default, intervals are derived.** 
- `#20C` means "20°C on the thermometer"
- `#20C - #10C` yields `#10C` (but is this a point or interval?)
- Addition of two points is an error; multiplication by a scalar is an error
- This is probably the most intuitive approach but has edge cases

**Option D: Don't overthink it — treat temperature like any other unit.**
- `#20C + #10C` → `#30C` (just add the numbers)
- `#20C * 2` → `#40C` (just multiply)
- Physically meaningless but practically useful for simple scripts
- This is what most unit libraries do

**Recommendation:** Option D for now. The target audience is building websites and domestic tools, not physics simulations. Document that temperature arithmetic is value-arithmetic, not thermodynamically correct. If demand exists for proper point/interval distinction, it can be added later as an opt-in.

### 5.3 Conversion

With the 1/900 K base unit, conversion is straightforward:

```parsley
let boiling = #100C
let boilingF = fahrenheit(boiling)  // #212F

let bodyTemp = #98.6F
let bodyTempC = celsius(bodyTemp)   // #37C
```

Internally:
- `#100C` → (273.15 + 100) × 900 = 335,835 base units
- Convert to °F: 335,835 / 500 − 459.67 → ... actually let's work in base units
- To display as °F: value_in_base / 500 − 459.67? No — the offset needs to be in base units too

Let's think about this more carefully:

- Store: base units = K × 900
- To Kelvin: K = base / 900
- To Celsius: C = base / 900 − 273.15 = (base − 245,835) / 900
- To Fahrenheit: F = base / 500 − 459.67 = (base − 229,835) / 500

Verification: #100C
- base = (100 + 273.15) × 900 = 373.15 × 900 = 335,835
- To °F: (335,835 − 229,835) / 500 = 106,000 / 500 = 212 ✓

Verification: #98.6F
- base = (98.6 + 459.67) × 500 = 558.27 × 500 = 279,135
- To °C: (279,135 − 245,835) / 900 = 33,300 / 900 = 37 ✓ (= 37°C, body temp)

All integer arithmetic, no floating point involved. The conversions are exact.

---

## 6. Arithmetic

### 6.1 Allowed Operations

Following Money's precedent, but with cross-unit conversion:

| Operation | Example | Result | Notes |
|-----------|---------|--------|-------|
| unit + unit (same family) | `#10cm + #1m` | `#110cm` | Left side determines display unit |
| unit − unit (same family) | `#1m - #10cm` | `#0.9m` | Left side determines display unit |
| unit × scalar | `#5m * 3` | `#15m` | |
| scalar × unit | `2 * #5kg` | `#10kg` | Commutative |
| unit / scalar | `#10m / 3` | `#3.333333m` | Banker's rounding at display |
| unit / unit (same family) | `#10km / #2km` | `5` | Returns a plain number (ratio) |
| −unit | `-#5m` | `#-5m` | Negation |
| unit + unit (different family) | `#10kg + #1m` | Error | Type error |
| unit × unit | `#5m * #3m` | Error (Phase 1) | See §6.3 |
| unit / unit (different family) | `#10kg / #1m` | Error (Phase 1) | See §6.3 |

### 6.2 The "Left Side Wins" Rule

When combining units from the same family but different scales or systems, the left operand determines the display unit of the result:

```parsley
#1cm + #1in     // result displayed in cm: #3.54cm
#2mi + #2km     // result displayed in miles: #3.24mi
#100C + #10F    // result displayed in °C: #105.56C (if we allow temperature addition)
```

Internally, both operands are converted to the base unit for arithmetic, then the result is tagged with the left operand's display unit.

**Comparison always normalises.** Two values are equal if they represent the same quantity, regardless of display unit:

```parsley
#1in == #25.4mm     // true
#0C == #32F         // true
#1024B == #1KiB     // true
```

### 6.3 Derived Units (Future)

Multiplying length × length to get area, or dividing distance by time to get speed, requires dimensional analysis. This is a significant complexity increase:

```parsley
// Future possibility
#10m * #5m      // #50sqm (area)
#100km / @1h    // #100km/h (speed)
```

**Recommendation:** Explicitly defer this to a future phase. In Phase 1, `unit * unit` is an error. Area and volume are entered directly as their own unit family, not derived from length arithmetic.

---

## 7. Constructors

### 7.1 Named Constructors

Each unit has a constructor function that creates a value in that unit:

```parsley
metre(123)          // #123m
inch(12)            // #12in
kilogram(5.5)       // #5.5kg
celsius(100)        // #100C
kilobyte(64)        // #64kB
```

These also serve as conversion functions when given a unit value:

```parsley
metre(#12in)        // #0.3048m
fahrenheit(#100C)   // #212F
gibibyte(#1TB)      // #931.32GiB
```

### 7.2 Constructor Naming

Use the full unit name (singular), in line with Parsley's preference for clarity:

| Unit | Constructor | Alternative considered |
|------|------------|----------------------|
| metre | `metre()` | `m()` — too terse, collides with identifiers |
| centimetre | `centimetre()` | `cm()` |
| inch | `inch()` | `in()` — reserved keyword in many languages |
| foot | `foot()` | `ft()` |
| kilogram | `kilogram()` | `kg()` |
| celsius | `celsius()` | |
| fahrenheit | `fahrenheit()` | |
| kelvin | `kelvin()` | |
| kilobyte | `kilobyte()` | `kB()` — weird casing for a function |
| kibibyte | `kibibyte()` | `KiB()` |

> **Open question:** US vs British spelling — `metre` vs `meter`, `litre` vs `liter`. The proposal doesn't specify. SI standard uses `-re`, US English uses `-er`. Since Imperial is only offered to support American users, and the primary audience is American, should we use `meter()`? Or follow SI convention since these are SI unit names? Could support both as aliases.

> **Open question:** Should we also provide short-form constructors? `m(123)` alongside `metre(123)`? It would be convenient but adds naming collisions. We could provide them only where the short form is unambiguous.

### 7.3 Generic Constructor

As an alternative to many named constructors, a single generic constructor:

```parsley
unit(123, "m")          // #123m
unit(#12in, "m")        // convert 12 inches to metres
```

This is consistent with `money(19.99, "USD")` but less readable for conversions. Could exist alongside named constructors:

```parsley
// Both equivalent
metre(#12in)
unit(#12in, "m")
```

**Recommendation:** Provide both. Named constructors for readability, `unit()` for dynamic/programmatic use where the target unit comes from a variable.

---

## 8. Properties and Methods

### 8.1 Properties

Following Money's `.amount`, `.currency`, `.scale`:

| Property | Type | Example (SI) | Example (Imperial) |
|----------|------|--------------|-------------------|
| `.value` | float | `#12.3m.value` → `12.3` | `#3/8in.value` → `0.375` |
| `.unit` | string | `#12.3m.unit` → `"m"` | `#3/8in.unit` → `"in"` |
| `.family` | string | `#12.3m.family` → `"length"` | `#3/8in.family` → `"length"` |
| `.system` | string | `#12.3m.system` → `"SI"` | `#3/8in.system` → `"US"` |
| `.amount` | int | `#12.3m.amount` → `123` | `#3/8in.amount` → `1890` |
| `.scale` | int/string | `#12.3m.scale` → `1` | `#3/8in.scale` → `5040` |

For SI units, `.amount` and `.scale` mirror Money: `amount / 10^scale` reconstructs the value.

For Imperial units, `.amount` is the HCN-scaled integer and `.scale` is the HCN divisor: `amount / scale` gives the value in the base unit. This exposes the internal representation for advanced use (database storage, serialisation, debugging).

The `.value` property always returns a float — the decoded value in the display-hint unit. This is the most commonly useful property.

> **Open question:** Should `.scale` return different types (int8 for SI, int64 for Imperial)? Or should we use different property names — e.g., `.divisor` for Imperial? Or just hide the internals entirely and only expose `.value`?

### 8.2 Methods

| Method | Description | Example | Result |
|--------|-------------|---------|--------|
| `.to(unit)` | Convert to another unit | `#1mi.to("km")` | `#1.609km` |
| `.format()` | Format with default locale | `#1234.5m.format()` | `"1,234.5 m"` |
| `.format(locale)` | Format with locale | `#1234.5m.format("de-DE")` | `"1.234,5 m"` |
| `.abs()` | Absolute value | `(-#5m).abs()` | `#5m` |
| `.repr()` | Parseable literal | `#12.3m.repr()` | `"#12.3m"` |
| `.toDict()` | Clean dictionary | `#12.3m.toDict()` | `{value: 12.3, unit: "m"}` |
| `.inspect()` | Debug dictionary | `#12.3m.inspect()` | `{__type: "unit", amount: 123, scale: 1, unit: "m", family: "length", system: "SI"}` |
| `.toFraction()` | Fraction string (Imperial) | `#3/8in.toFraction()` | `"3/8\""` |

### 8.3 The `.to()` Method vs. Named Constructors

Both achieve the same result:

```parsley
#12in.to("m")       // #0.3048m
metre(#12in)        // #0.3048m
```

`.to()` is better for chaining and dynamic targets:

```parsley
let targetUnit = "km"
distance.to(targetUnit)
```

Named constructors are better for clarity:

```parsley
let distanceInKm = kilometre(distance)
```

**Recommendation:** Support both.

---

## 9. Fractional Literals

### 9.1 The Proposal

The proposal suggests allowing fractions in literals:

```parsley
#1/3in      // one-third of an inch
#1/4mi      // one-quarter of a mile
#3/8in      // three-eighths of an inch
```

### 9.2 With HCN Storage, Fractions Are (Mostly) Solved

The HCN representation (§4) changes this analysis entirely. Fractions are no longer a display hack — they're the natural idiom for Imperial units.

**Storage:** `#3/8in` is stored as 3/8 × 5040 = 1890. An exact integer. No truncation, no rounding.

**Round-trip:** 1890 → GCD(1890, 5040) = 630 → 1890/630 = 3, 5040/630 = 8 → `#3/8in`. Exact.

**Arithmetic:**

```parsley
#1/3in + #1/3in + #1/3in
// 1680 + 1680 + 1680 = 5040 = 1 inch ✓ (exactly!)

#3/8in + #5/8in
// 1890 + 3150 = 5040 = 1 inch ✓

#1/3cup + #2/3cup
// 1680 + 3360 = 5040 = 1 cup ✓

#1lb - #4oz
// 80640 - 20160 = 60480 = 12 oz = 3/4 lb ✓
```

All exact. No special handling needed. It's just integer addition.

### 9.3 Lexer Impact

The lexer must recognise `#INTEGER/INTEGER+SUFFIX` as a fraction unit literal. This is context-dependent but unambiguous because:

- It starts with `#` (unit sigil)
- Followed by an integer, `/`, another integer — this is NOT the division operator because the left side (`#3`) is a unit-literal-in-progress, not a complete expression
- Followed by a unit suffix

The parser would produce a `FractionUnitLiteral` AST node with numerator, denominator, and unit suffix. The evaluator converts to HCN sub-units at creation time.

Mixed numbers are essential — they're how Imperial measurements are actually written in practice. A timber framer writes "92-5/8 inches", not "92.625 inches" or "92 inches plus 5/8 inch".

```parsley
#92-5/8in   // 92 + 5/8 = 92.625 inches = 466830 sub-inches (standard stud)
#2-3/8in    // 2 + 3/8 = 2.375 inches = 11970 sub-inches
#1-1/2lb    // 1.5 pounds = 24 ounces = 120960 sub-ounces
```

**Syntax options:**

| Syntax | Example | Pros | Cons |
|--------|---------|------|------|
| Hyphen | `#92-5/8in` | Visually clear, no ambiguity | Could be misread as subtraction |
| Space | `#92 5/8in` | Most natural written form | Lexer can't see past whitespace |
| Parenthesised | `#(92 5/8)in` | Unambiguous grouping | Verbose, unfamiliar |
| Addition | `#92in + #5/8in` | No new syntax needed | Verbose, loses the idiom |

**Recommendation:** Hyphen (`#92-5/8in`). It's visually distinct from subtraction because it's inside a unit literal (after `#`, before the suffix). The lexer is already in "unit literal mode" when it sees `#` followed by a digit — a hyphen between an integer and a fraction is unambiguous in that context. This is also how mixed numbers appear in construction documents (e.g., "92-5/8"" on architectural drawings).

### 9.4 Displaying Fractions

Given an HCN-scaled integer, displaying as a fraction is deterministic (not fuzzy):

1. Compute `g = GCD(amount, HCN)`
2. `numerator = amount / g`, `denominator = HCN / g`
3. If `denominator == 1`: display as integer
4. If `numerator > denominator`: extract whole part, display as mixed number
5. If `denominator` is a "common" denominator (2, 3, 4, 5, 6, 7, 8, 10, 12, 16): display as fraction
6. Otherwise: fall back to decimal

This is slower than decimal display but not meaningfully so — it's a single GCD computation (fast for small numbers) plus an integer division. The user is right: output is slow anyway compared to arithmetic.

**Examples:**

| Amount (HCN=5040) | GCD | Num/Den | Display |
|--------------------|-----|---------|---------|
| 5040 | 5040 | 1/1 | `#1in` |
| 1890 | 630 | 3/8 | `#3/8in` |
| 1680 | 1680 | 1/3 | `#1/3in` |
| 11970 | 630 | 19/8 | `#2 3/8in` |
| 80640 | 80640 | 1/1 | `#1lb` (= 16oz base, displayed in user's hint unit) |
| 4973 | 1 | 4973/5040 | `#0.9867...in` (no clean fraction — decimal fallback) |

Step 6 (the decimal fallback) is for values that don't reduce to a common fraction — typically the result of cross-system conversions. This is the right behaviour: if the number isn't a clean fraction, don't pretend it is.

### 9.5 What About Fractions in SI Units?

SI units use decimal storage, so `#1/3m` can't be stored exactly. Options:

**Option A: Fractions are Imperial-only.** `#1/3in` is valid, `#1/3m` is a syntax error. This is pragmatic — fractions in SI are rare in practice. People write `#0.333m`, not `#1/3m`.

**Option B: Allow fractions for SI but convert to decimal.** `#1/3m` is parsed, converted to 0.333333m (truncated to max scale), and stored as a decimal SI unit. Document that SI fractions are approximate. The user gets the convenience of the literal syntax with the caveat that it's sugar over truncation.

**Option C: Allow fractions everywhere; SI fractions create an Imperial-style object.** This is conceptually odd — "1/3 metre" stored with an HCN denominator. It works arithmetically but conflates measurement system with number representation.

**Recommendation:** Option A. Fractions are an Imperial idiom — that's literally why the HCN representation exists. SI users write decimals. This keeps the conceptual model clean: `#1/3in` → Imperial (HCN), `#0.333m` → SI (decimal). If a user wants a third of a metre, they write `#1m / 3` and get `#0.333333m`.

> **Open question:** Is Option A too restrictive? A dressmaker might want `#1/3m` for fabric. But they'd more likely use `#33.3cm` or `#333mm`. The fraction idiom is genuinely an Imperial thing — it exists *because* Imperial subdivisions are based on halves, thirds, and powers of two rather than powers of ten.

---

## 10. Formatting and Display

### 10.1 Default Display

SI units display as decimals. Imperial units display as fractions when the value reduces to a common denominator, otherwise as decimals:

```parsley
// SI — decimal display
#12.3m      // displays as: #12.3m
#2.54cm     // displays as: #2.54cm
#100C       // displays as: #100C

// Imperial — fraction display (via GCD)
#3/8in      // displays as: #3/8in
#1/3cup     // displays as: #1/3cup
#5ft        // displays as: #5ft (whole number — no fraction needed)
#1lb + #4oz // displays as: #1 1/4lb (if display hint is "lb")

// Imperial — decimal fallback (when GCD gives an uncommon denominator)
metre(#1cm).to("in")  // displays as: #0.393701in (no clean fraction)
```

### 10.2 In String Interpolation

```parsley
let height = #1.83m
`Height: {height}`          // "Height: 1.83m"  (no # prefix in interpolation)
`Height: {height.format()}`  // "Height: 1.83 m" (with space, formatted)
```

> **Open question:** Should interpolation include the `#` prefix? Money doesn't include the `$` in `{price}` interpolation — it formats naturally. Units should behave similarly: `{height}` → `"1.83m"` or `"1.83 m"`.

### 10.3 In PLN

Unit values in PLN (Parsley Literal Notation) would use the literal syntax:

```parsley
// PLN output
{
    height: #1.83m,
    weight: #75kg,
    temperature: #37C
}
```

### 10.4 SI Display: Precision and Rounding

SI units use decimal display. When converting between units, the result may have many decimal places:

```parsley
metre(#1in)     // #0.0254m — exact
metre(#1ft)     // #0.3048m — exact
celsius(#98.6F) // #37C — exact
celsius(#72F)   // #22.222222C — repeating decimal
```

How many decimal places should be displayed?

**Recommendation:** Show up to 6 significant figures by default, with `.format(precision)` for control. This balances precision with readability.

### 10.5 Imperial Display: Fractions

Imperial units display as fractions by default (see §9.4). The GCD-based algorithm produces clean fractions for any value that divides the HCN evenly, and falls back to decimal for values that don't.

```parsley
#3/8in              // displays as: #3/8in
#1/3cup             // displays as: #1/3cup
#1lb + #4oz         // displays as: #1 1/4lb (if mixed numbers supported)
metre(#1in).to("in") // displays as: #1in (round-trip exact)
```

For compound display (feet-and-inches, pounds-and-ounces), a `.format()` option:

```parsley
let board = #148 3/8in        // if mixed numbers supported, or #1in * 148 + #3/8in
board.format("ft-in")          // "12' 3/8\""
board.format("decimal")        // "148.375\""
```

---

## 11. Schema Integration

### 11.1 Unit Types in Schemas

Schemas should be able to declare fields as unit types:

```parsley
@schema Product {
    name: string,
    weight: mass,       // accepts any mass unit
    height: length,     // accepts any length unit
    size: data          // accepts any data unit
}
```

Or more specific:

```parsley
@schema Product {
    weight: unit("kg"),     // must be in kg
    height: unit("length")  // any length unit
}
```

> **Open question:** What's the right syntax for unit types in schemas? This can be deferred until the core unit type is implemented.

---

## 12. Comparison with Money

| Aspect | Money | SI Units | Imperial Units |
|--------|-------|----------|----------------|
| Families | ~180 currencies | ~5 families | ~5 families |
| Internal storage | int64 + decimal scale | int64 + decimal scale | int64 / HCN |
| Numeric idiom | Decimal | Decimal | Fractional |
| Round-trip fidelity | Always exact | Always exact | Always exact |
| Within-family arithmetic | Error (different currencies) | Exact (powers of 10) | Exact (integer ratios + HCN) |
| Cross-system arithmetic | N/A | Rounding at SI↔US boundary | Rounding at SI↔US boundary |
| Scalar operations | × and / only | × and / only | × and / only |
| Division by same type | Not supported | Returns plain number | Returns plain number |
| Constructor | `money(amount, code)` | Named: `metre(val)` | Named: `inch(val)` |
| Display | Symbol or CODE# format | Decimal with suffix | Fraction or decimal |

Money and SI units share the identical storage model: integer + decimal scale. Imperial units diverge to support fractions natively via HCN scaling. But all three guarantee exact round-trips and exact within-system arithmetic. Rounding only appears when crossing the SI↔US boundary, which parallels how Money forbids cross-currency arithmetic entirely (because exchange rates are external and volatile, unlike unit conversion ratios which are fixed by international definition).

---

## 13. Implementation Phasing

### Phase 1: Core (MVP)

**Length, Mass, Digital Information**

These three families test all the hard problems:
- Dual representation: decimal scale (SI) and HCN scaling (Imperial)
- Within-system arithmetic with exact integer ratios
- Cross-system conversion at the SI↔US boundary
- Fraction literals for Imperial units
- Fraction display via GCD
- Different system structures (SI powers-of-10, US integer ratios, digital)

Deliverables:
- Lexer: `#` + number + suffix token; `#` + fraction + suffix token
- Parser: UnitLiteral and FractionUnitLiteral AST nodes
- Evaluator: SIUnit and ImperialUnit objects
- Arithmetic: same-system (exact), cross-system (bridge ratio), scalar
- Conversion tables: within-system ratios and cross-system bridge ratios
- Display: decimal for SI, fraction-via-GCD for Imperial
- Constructors: Named constructors for all length/mass/data units
- Generic constructor: `unit(value, code)`
- Properties: `.value`, `.unit`, `.family`, `.system`
- Methods: `.to()`, `.format()`, `.abs()`, `.repr()`, `.toDict()`, `.inspect()`, `.toFraction()`
- Error messages: Clear errors for cross-family arithmetic

### Phase 2: Temperature + Volume

**Temperature** adds offset conversions (not just scaling), requiring formula-based conversion rather than simple ratios. Fahrenheit uses HCN storage; Celsius/Kelvin use decimal.  
**Volume** adds a family where the HCN approach shines — 1/3 cup, 1/4 tsp are exact.

Deliverables:
- Temperature unit family with K, °C (SI) and °F (Imperial/HCN)
- Volume unit family with mL, L (SI) and fl oz, cup, pt, qt, gal (Imperial/HCN)
- Arithmetic semantics for temperature (see §5)

### Phase 3: Area

**Area** adds a derived quantity. With dual representation, each system stores area in its own base:
- SI: square metre with decimal scale (range up to ~9.2 × 10¹² m² — millions of km²)
- Imperial: square inch with HCN scaling (range up to billions of square miles)

Deliverables:
- Area unit family with SI (mm², cm², m², km², hectare) and US (in², ft², yd², acre, mi²) units
- Within-system ratios (e.g., 144 in²/ft², 10,000 m²/hectare)
- Cross-system bridge ratio (1 in² = 0.00064516 m²)
- Area literals and constructors

### Phase 4: Polish and Expansion

- Mixed number literals (`#2 3/8in`) if lexer design is resolved
- Compound display formatting (e.g., `5' 3 1/4"`, `2 lb 5 oz`)
- Schema integration (see §11)
- Additional unit families if demanded (angle, energy, power)
- Derived unit arithmetic (`#5m * #3m` → area) if demanded

---

## 14. Open Questions Summary

These are the decisions that need to be made before moving to a specification:

| # | Question | Options | Leaning |
|---|----------|---------|---------|
| 1 | Is `#` the right sigil for unit literals? | `#`, bare suffix, other sigil | `#` (§3) |
| 2 | US vs British spelling for constructors? | `metre`/`meter`, `litre`/`liter` | Support both as aliases |
| 3 | Should we provide short-form constructors? | `m()` alongside `metre()` | No — collision risk |
| 4 | HCN value | 725,760 per yard (= 20,160 per inch). Derived from µm-scale precision, handles 64ths. | Settled (§4.3) |
| 5 | How to handle temperature arithmetic? | Point vs. interval distinction, or just "treat like numbers" | Treat like numbers (§5.2) |
| 6 | Are fraction literals Imperial-only? | Imperial-only, or allow (approximate) SI fractions too | Imperial-only (§9.5) |
| 7 | Mixed number literal syntax? | `#92-5/8in` (hyphen), `#92 5/8in` (space), require addition | Hyphen — matches construction docs (§9.3) |
| 8 | Store display hint? | Yes (for "left wins" and default display), no (format explicitly) | Yes (§4.5) |
| 9 | Max scale for cross-system conversions? | Per-unit max, global constant (e.g., 9) | Needs decision (§4.7) |
| 10 | SI → Imperial non-fraction display? | Decimal fallback, nearest-fraction, user's choice | Offer both via `.format()` (§4.7) |
| 11 | How many decimal places in SI display? | All significant, 6 sig figs, match input | 6 sig figs default (§10.4) |
| 12 | Unit types in schemas? | `mass`, `unit("kg")`, other | Defer to Phase 4 |
| 13 | String interpolation format? | `"1.83m"`, `"1.83 m"`, `"#1.83m"` | `"1.83 m"` (with space) |
| 14 | Should `unit / unit` return a number? | Yes (ratio), no (error) | Yes |
| 15 | Generic `unit()` constructor? | Yes (alongside named), no (named only) | Yes (§7.3) |
| 16 | Which Imperial system? | US Customary, British Imperial | US Customary only |
| 17 | Expose internal representation? | `.amount`/`.scale`, `.value` only, both | Needs thought (§8.1) |
| 18 | Equality across systems? | `#1in == #2.54cm` true, or false | True — normalise via bridge ratio |
| 19 | Digital units: own representation or SI? | Store in bytes (base), use SI decimal model | SI decimal model (§4.9) |
| 20 | Is there a generic "precise" type? | Reusable {int64, divisor} type, or units-specific | Explore (§9.6) |

---

## 15. Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Scope creep | High — many unit families, each with edge cases | Strict phasing, MVP with 3 families |
| Two internal representations | Medium — more code paths than Money | Clean interface boundary; table-driven conversion ratios |
| Suffix collisions | Medium — `m` (metre), `g` (gram) are common letters | Only valid after `#` + number, so no real collision |
| User confusion with `#` in money vs units | Low–Medium | Clear documentation, distinct visual patterns |
| Cross-system rounding surprises | Low — rounding only at SI↔US boundary | Document clearly; use generous max scale |
| GCD display performance | Low — GCD is fast for small numbers | HCN values are small (5040–20160); GCD is O(log n) |
| Fraction display edge cases | Medium — what if GCD gives 1/5040? | Decimal fallback for non-"common" denominators |
| Equality across systems | Medium — requires normalising through bridge ratio | Define canonical comparison; document precision limits |
| Temperature semantic debates | Low | "Treat like numbers" sidesteps the issue |
| Maintenance burden | Medium — every new unit needs lexer/parser/evaluator work | Table-driven design, not hard-coded per unit |

---

## 16. What Does Success Look Like?

A timber framer calculating stud material for a wall can write:

```parsley
let studHeight = #92-5/8in              // standard pre-cut stud
let wallLength = #14ft                  // wall to frame
let spacing = #16in                     // on-center spacing
let bays = wallLength / spacing         // 10.5 — need 11 bays
let studs = ceil(bays) + 1             // 12 studs (11 bays + end)

let totalTimber = studHeight * studs    // #1111-1/2in — exact
let timberFeet = foot(totalTimber)      // #92-5/8ft — exact (GCD gives 5/8)

// Or display as feet-and-inches:
totalTimber.format("ft-in")             // "92' 7-1/2\""
```

Every measurement is a fraction. Every calculation is integer arithmetic on HCN-scaled values. The framer's tape measure and the code speak the same language.

A woodworker calculating cuts from a board can write:

```parsley
let boardLength = #96in                 // 8-foot board
let cutA = #12-3/8in                    // first piece
let cutB = #8-5/16in                    // second piece
let kerf = #1/8in                       // blade width per cut

let used = cutA + cutB + kerf * 2       // #20-15/16in — exact
let remaining = boardLength - used      // #75-1/16in — exact
let fitsAnother = remaining / cutA      // 6.06... — six more pieces
```

Every fraction is maintained exactly. `#1/8in + #1/8in` is `#1/4in`, not `#0.24999999in`. The carpenter's tape measure and the code agree.

A home cook scaling a recipe can write:

```parsley
let flour = #2/3cup
let sugar = #1/3cup
let milk = #3/4cup

// Scale up by 1.5×
let bigFlour = flour * 3 / 2           // #1cup — exact (not #0.999999cup)
let bigSugar = sugar * 3 / 2           // #1/2cup — exact
let bigMilk = milk * 3 / 2            // #1-1/8cup — exact

// Convert for a European friend
let milkMl = millilitre(bigMilk)       // #266.16mL
```

Fractions stay as fractions. Thirds stay as thirds. The recipe scales without drift.

A carpet seller can write:

```parsley
let roomLength = #12ft
let roomWidth = #10ft
let area = #120sqft                 // Phase 3: entered directly (not derived)
let areaInSqM = unit(area, "sqm")  // conversion

let pricePerSqM = $24.99
let rolls = area / #50sqft         // 2.4 rolls needed
let rollsNeeded = ceil(rolls)      // 3 rolls

let total = pricePerSqM * areaInSqM.value  // $277.64
```

A dressmaker converting a pattern can write:

```parsley
let seamAllowance = #5/8in
let metricSeam = millimetre(seamAllowance)  // #15.875mm — exact (5/8 × 25.4)
let panels = 6
let totalAllowance = seamAllowance * panels // #3-3/4in — exact
```

A user working with temperature can write:

```parsley
let ovenTemp = #350F
let ovenTempC = celsius(ovenTemp)   // #176.67C

let outsideTemp = #22C
`It's {outsideTemp} outside ({fahrenheit(outsideTemp)})`
// "It's 22C outside (71.6F)"
```

A user sizing storage can write:

```parsley
let fileSize = #2.5GB
let diskSpace = #500GiB
let filesPerDisk = diskSpace / fileSize  // 214.7...
```

The code is readable, precise, and free of manual conversion logic. Fractions are fractions, decimals are decimals, and the two systems meet cleanly at their boundary. That's the goal.

---

## 17. Exploration: A Generic "Precise" Type?

This section captures an open line of thinking that emerged from the design discussion. It is not part of the unit proposal per se, but is worth recording.

### 17.1 The Observation

Both SI and Imperial representations are variations of the same structure: **an integer numerator over a known denominator**.

- SI: `Amount / 10^Scale` — the denominator is always a power of 10
- Imperial: `Amount / HCN` — the denominator is always 5040 (or 10080)
- Money: `Amount / 10^Scale` — identical to SI

Is there a generic type here? Something like:

```go
type Precise struct {
    Numerator   int64
    Denominator int64 // could be 10^n, HCN, or any positive integer
}
```

This is a rational number with a fixed denominator. Arithmetic between values with the **same denominator** is trivial integer arithmetic. Arithmetic between values with **different denominators** requires finding a common denominator — the standard rational number approach.

### 17.2 What Could It Enable?

Beyond units, a `Precise` type could represent:

- **Exact fractions as a numeric type:** `#1/3` as a first-class value, not a float
- **Recipe scaling:** multiply all ingredients by 2/3 without accumulating float errors
- **Probability:** `#1/6` (one die face) without 0.16666... drift
- **Music/rhythm:** 1/4 note, 3/8 time, dotted 1/2 — exact subdivisions

The `#` sigil could mean "precise literal":

```parsley
let third = #1/3       // Precise{1, 3}
let half = #1/2         // Precise{1, 2}
third + half            // Precise{5, 6} — exact
third * 3               // Precise{1, 1} = 1 — exact
```

### 17.3 Relationship to Units

Units would be a Precise value plus metadata (family, system, display hint). The Precise type provides the arithmetic; the Unit type adds dimensional semantics.

```
Unit = Precise + Family + DisplayHint
Money = Precise(denominator=10^scale) + Currency
```

This is appealing architecturally: a single numeric engine powering both features.

### 17.4 What About Irrationals?

The original suggestion mentioned `#1/2π`. A rational type can't represent π — it's irrational by definition. You'd need either:

- **Symbolic computation:** store `π` as a symbol, not a number. This is a fundamentally different kind of system (CAS), well beyond scope.
- **Approximation:** `#1/2 * PI` where `PI` is a float constant. The `#1/2` is exact, the multiplication produces a float. Not really "precise" for the irrational part.

Conclusion: a Precise type handles rationals exactly. Irrationals remain floats. This is still very useful — most practical precision problems involve rationals (money, fractions, unit conversions), not irrationals.

### 17.5 Feasibility and Scope

A generic Precise type is a larger undertaking than units alone:

- New numeric type in the evaluator (alongside Integer and Float)
- Operator overloading for all arithmetic operations
- GCD normalisation after every operation (to keep denominators small)
- Interaction with Integer, Float, Money, and Unit types
- Potential denominator overflow for long chains of operations

It's worth exploring but should NOT block the unit feature. The pragmatic path is:

1. Build units with the dual representation (decimal SI + HCN Imperial) as described in §4
2. If the pattern proves useful, extract a Precise type as a refactoring later
3. The internal implementation can evolve without changing the user-facing unit API

> **Open question:** Is a Precise/Rational type worth pursuing as a standalone feature, independent of units? It would be a distinctive Parsley capability — few scripting languages offer exact rational arithmetic as a built-in type.