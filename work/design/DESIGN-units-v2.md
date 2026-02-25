# Design: Measurement Units for Parsley — V2 Summary

**Distilled from:** `DESIGN-units.md`
**Purpose:** Capture the decisions made, the key design, and the open questions — without the exploratory narrative.

---

## 1. What This Is

A built-in unit-of-measurement type for Parsley, following the Money precedent. Integer storage, no floating-point drift, exact round-trips, fractional Imperial arithmetic that actually works.

**Target audience:** Domestic and small-business scripts — carpet sellers, timber framers, home cooks, dressmakers, storage calculators.

---

## 2. Decisions Made

### 2.1 Literal Syntax: `#` Prefix

```parsley
#12.3m      #45oz      #3/8in      #92+5/8in      -#6C
```

- `#` + number + suffix for all units
- No collision with Money's `CODE#amount` pattern — lexer can disambiguate (`EUR#50` vs `#50m`)
- Negative values: `-#6C` or via arithmetic

### 2.2 Three Representations, One Interface

SI, Imperial, and Temperature have fundamentally different numeric idioms. Each gets the right internal representation, but all present the same interface to the user — same literals, operators, methods.

| System | Storage | Idiom | Base units |
|--------|---------|-------|------------|
| **SI** | int64 + decimal scale (like Money) | Decimal: `12.3m`, `0.5kg` | metre, gram, byte |
| **Imperial** | int64 / HCN (fixed denominator) | Fractional: `3/8in`, `1/3cup` | yard, ounce, quart |
| **Temperature** | int64 in K × 900 base units | Offset + ratio conversions | kelvin (×900) |

### 2.3 HCN = 725,760

Derived from the LCM of actual requirements, not picked from a table of Highly Composite Numbers:

```
Need HCN per yard divisible by 36 × q for each fraction 1/q of an inch:
  36 × 64 = 2,304    (64ths — machining, fine carpentry)
  36 × 9  = 324      (9ths)
  36 × 7  = 252      (7ths)
  36 × 5  = 180      (5ths)

LCM = 2⁸ × 3⁴ × 5 × 7 = 725,760
```

- **20,160 sub-units per inch** — every fraction from halves to sixty-fourths is an exact integer
- **Precision:** ~1.26 µm per sub-yard, matching SI micrometre precision
- **Range:** int64 holds ~77 AU of length, hundreds of millions of tons of mass

> **Why not 720,720** (a Superior Highly Composite Number)? 720,720 = 2⁴ × 3² × 5 × 7 × 11 × 13 — it has more distinct prime factors but not enough depth in 2 and 3. After dividing by 36 (inches per yard), it has **zero factors of 3** left, so thirds of an inch fail. It also lacks 2⁸, so 64ths fail. The 11 and 13 factors buy exact 11ths and 13ths — fractions nobody uses in Imperial measurement. "Most divisors" ≠ "right divisors."

### 2.4 Fractional Literals: Both Systems, Different Storage

Fraction syntax is allowed for both SI and Imperial units, but the storage differs:

- **Imperial:** Stored as exact HCN integers. `#1/3yd` remains `#1/3yd` — the fraction is preserved.
- **SI:** Immediately converted to decimal. `#1/3m` is syntactic sugar for `#1m / 3` → stored as `#0.333333m` (truncated at max scale). The fraction is a convenience, not a storage format.

```parsley
#3/8in      // Imperial: stored as 7,560 sub-inches (exact, fraction preserved)
#1/3cup     // Imperial: stored as 241,920 sub-quarts (exact, fraction preserved)
#1/3m       // SI: converted to #0.333333m (decimal, like #1m / 3)
#1/2km      // SI: converted to #0.5km (exact — "half a kilometre" is natural speech)
```

This reflects how people actually talk: "half a kilometre" is natural, and nobody who deals in km would be surprised to see it become `#0.5km`. Meanwhile `#1/3yd` stays as a fraction because that's how Imperial works.

**Equivalence:** `#1/3m == #1m / 3` — the fraction literal is purely syntactic sugar on the SI side.

### 2.5 Mixed Number Syntax: `+` Separator

```parsley
#92+5/8in   // Imperial: 92 + 5/8 inches (fraction preserved)
#2+3/8in    // Imperial: 2 + 3/8 inches (fraction preserved)
#1+1/2lb    // Imperial: 1.5 pounds (fraction preserved)
#1+1/2km    // SI: converted to #1.5km (decimal)
```

Uses `+` as the separator between whole part and fraction. Unambiguous inside a unit literal (after `#`, before suffix) since the lexer is already in "unit literal mode."

> **Note:** American construction documents conventionally use a hyphen (`92-5/8"`), but a hyphen reads as subtraction to non-American eyes. The `+` is universally unambiguous and semantically correct — the whole part and fraction are being *added*.

### 2.6 Display Hint (Stored, Not Identity)

Values carry the unit the user originally wrote (`cm`, `ft`, `oz`) as a display hint. Two values with different hints but the same underlying amount are **equal**.

```parsley
#254cm + #1m        // → #354cm (left hint "cm" wins)
#1m + #254cm        // → #3.54m (left hint "m" wins)
#254cm == #2.54m    // true
```

### 2.7 "Left Side Wins" Rule

The left operand determines the display unit (and measurement system) of the result:

```parsley
#1cm + #1in     // result in cm: #3.54cm
#2mi + #2km     // result in miles: #3.24mi
```

### 2.8 Cross-System Conversion

Rounding lives **only** at the SI↔Imperial boundary. Bridge ratios are international definitions:

| Bridge | Ratio | Direction |
|--------|-------|-----------|
| Length | 1 in = 0.0254 m (exact) | Imperial → SI is often exact |
| Mass | 1 lb = 453.59237 g (exact) | SI → Imperial may round |
| Volume | 1 gal = 3.785411784 L (exact) | |

With 20,160 sub-inches, rounding error on SI → Imperial is at most ~0.63 µm.

**Max scale for cross-system conversions: 6** (micrometre precision). This matches the Imperial side's ~1.26µm precision — no point storing more decimal places than the source system can distinguish.

### 2.9 Temperature: K × 900 Base with "Treat Like Numbers" Semantics

Temperature gets its own internal representation — a **third** type, distinct from both SI decimal and Imperial HCN:

- **Base unit:** K × 900 (900 sub-units per kelvin)
- **Why 900?** Because 1°C = 9/5°F. With 900 sub-K per kelvin, 1°F = 500 sub-K. Both are exact integers. The 5/9 conversion factor becomes integer multiply-then-divide with no remainder.

**Conversion formulas (all integer arithmetic):**

| Direction | Formula |
|-----------|---------|
| To base from °C | `base = (C + 273.15) × 900` |
| To base from °F | `base = (F + 459.67) × 500` |
| Base to °C | `C = (base − 245,835) / 900` |
| Base to °F | `F = (base − 229,835) / 500` |

**Verification:**

```
#100C → base = 373.15 × 900 = 335,835
  → °F: (335,835 − 229,835) / 500 = 106,000 / 500 = 212 ✓

#98.6F → base = 558.27 × 500 = 279,135
  → °C: (279,135 − 245,835) / 900 = 33,300 / 900 = 37 ✓
```

This makes temperature conversions **lossless in both directions** — unlike length/mass where SI↔Imperial can round at the boundary.

**Point vs. Interval:** Temperature values can conceptually be either a "point on the thermometer" (20°C outside) or a "difference" (rose by 5°C). Some unit libraries distinguish these at the type level, restricting which operations are valid on each. The decision here is: **don't distinguish.** `#20C + #10C = #30C` — just number addition. This is what the target audience expects, and what every mainstream unit library does. Not thermodynamically rigorous, but practically useful. Can revisit if demand exists.

### 2.10 Arithmetic Rules

| Operation | Result | Notes |
|-----------|--------|-------|
| unit + unit (same family) | unit | Left side wins display |
| unit − unit (same family) | unit | Left side wins display |
| unit × scalar | unit | |
| scalar × unit | unit | Commutative |
| unit / scalar | unit | May truncate (like Money) |
| unit / unit (same family) | **plain number** | Dimensionless ratio |
| unit + unit (different family) | **error** | Can't add kg to metres |
| unit × unit | **error** (Phase 1) | Derived units deferred |

### 2.11 Digital Information: SI Model, Byte Base

All conversions are exact integer ratios (powers of 10 for SI, powers of 2 for IEC). Stored in bytes as the base unit using the SI decimal representation.

**Why bytes, not bits?** The target domain is web scripts, storage calculators, and file handling — all byte-oriented. Bits are used for network *throughput* (Mbps), which is a *rate* (a different kind of thing, and out of scope). Sub-byte arithmetic (individual bits, nibbles) is a systems programming concern. If bit-level work ever becomes needed, bitwise operations are a better fit than the units system.

### 2.12 Constructors and Conversion

**Named constructors are the primary API.** Full unit name (singular), doubling as conversion functions:

```parsley
metre(123)          // create: #123m
metre(#12in)        // convert: #0.3048m
fahrenheit(#100C)   // convert: #212F
```

**Generic constructor** is the escape hatch for dynamic/programmatic use — when the target unit comes from a variable or user input:

```parsley
unit(123, "m")
unit(#12in, "m")

let target = getUserChoice()
distance.to(target)
```

**`.to()` method** for chaining with dynamic targets:

```parsley
#1mi.to("km")      // #1.609km
```

Named constructors are preferred for readability; `unit()` and `.to()` exist for the case where the unit is a runtime value.

**Constructor spelling:** SI convention (`-re`): `metre()`, `litre()`, `kilometre()`. US spelling available as aliases: `meter()`, `liter()`, `kilometer()`.

**No short-form constructors.** `m()` and `g()` collide with likely variable names. The full name is always used.

### 2.13 Equality Normalises Across Systems

```parsley
#1in == #25.4mm     // true
#0C == #32F         // true
#1024B == #1KiB     // true
```

### 2.14 Fraction Display via GCD

Deterministic, not fuzzy: `GCD(amount, HCN)` → reduced fraction. Falls back to decimal if the denominator isn't a "common" one (2, 3, 4, 5, 6, 7, 8, 10, 12, 16, 32, 64).

Complexity is pushed onto display/formatting to keep arithmetic simple and performant — the right trade-off since output is slow relative to arithmetic anyway.

**SI → Imperial non-fraction display:** Default to decimal (`#1.3937in`). Decimal is exact to the stored precision; guessing a "nearest common fraction" is a heuristic that adds error rather than removing it. If someone wants the nearest fraction, they can ask explicitly via `.format("fraction")`.

### 2.15 Display Precision Defaults

Defaults should match real-world everyday conventions, not scientific precision. Always overridable via `.format(precision)`.

| Context | Default | Rationale |
|---------|---------|-----------|
| Lengths in m | 2dp | `1.83m` (height), `2.54m` |
| Lengths in cm | 1dp | `12.7cm` |
| Lengths in mm | 0dp | `25mm` |
| Mass in kg | 2dp | `2.50kg` |
| Mass in g | 0dp | `500g` |
| Celsius | 1dp | `37.0C` (medical), `22.2C` |
| Fahrenheit | 1dp | `98.6F` (medical), `72.0F` |
| Imperial lengths | Fraction via GCD | `3/8in`, `92+5/8in` |
| Imperial mass | Fraction via GCD | `1/4lb` |
| Digital | 0dp (always whole bytes) | `1024B`, `2GB` |

### 2.16 String Interpolation

No `#` sigil, no space between value and unit:

```parsley
let height = #1.83m
`Height: {height}`      // "Height: 1.83m"
```

Consistent with Money, which doesn't include `$` in interpolation. The `.format()` method is available for more control.

### 2.17 Properties: `.value` Only

Internal representation (`.amount`, `.scale`, HCN constants) is **not exposed** as properties. The public API is `.value` (decoded float in the display-hint unit). This avoids the confusion of `.scale` meaning different things for SI (int8 exponent) and Imperial (HCN constant), and keeps the internal representation free to evolve.

Debug access is available via `.inspect()` for development use.

### 2.18 Implementation Phasing

| Phase | Scope |
|-------|-------|
| **1 — MVP** | Length, Mass, Digital Information. Tests all hard problems: dual representation, cross-system conversion, fraction literals, fraction display |
| **2** | Temperature (K×900 representation, offset conversions) + Volume (1/3 cup shines here) |
| **3** | Area (entered directly, not derived from length × length) |
| **4** | Mixed number polish, compound display (`5' 3+1/4"`), schema integration, derived units if demanded |

---

## 3. Object Shapes

```go
// SI — identical to Money's model
type SIUnit struct {
    Amount      int64   // value = Amount / 10^Scale, in base SI unit
    Scale       int8    // decimal places
    Family      string  // "length", "mass", "data"
    DisplayHint string  // "cm", "m", "km", etc.
}

// Imperial — HCN-scaled integer
const HCN = 725_760    // 2⁸ × 3⁴ × 5 × 7

type ImperialUnit struct {
    Amount      int64   // value = Amount / HCN, in base Imperial unit
    Family      string  // "length", "mass", "volume"
    DisplayHint string  // "in", "ft", "yd", "mi", etc.
}

// Temperature — K × 900 base, unified across systems
type TempUnit struct {
    Amount      int64   // value in sub-kelvins (K × 900)
    DisplayHint string  // "K", "C", or "F"
}
```

### Base Units

| Family | SI base | Imperial base | Notes |
|--------|---------|---------------|-------|
| Length | metre | yard | |
| Mass | gram | ounce | |
| Temperature | kelvin × 900 | kelvin × 900 | Unified — not split by system |
| Data | byte | — | Byte, not bit (see §2.11) |
| Volume (Phase 2) | litre | quart | |

---

## 4. Unit Suffix Table

### Length
`mm` `cm` `m` `km` · `in` `ft` `yd` `mi`

### Mass
`mg` `g` `kg` · `oz` `lb`

### Temperature
`K` `C` · `F`

### Digital Information
`B` `kB` `MB` `GB` `TB` · `KiB` `MiB` `GiB` `TiB`

### Volume (Phase 2)
`mL` `L` · `floz` `cup` `pt` `qt` `gal`

### Area (Phase 3)
`mm2` `cm2` `m2` `km2` · `in2` `ft2` `yd2` `ac` `mi2`

> Area uses the `unit2` suffix style (standard scientific/engineering notation, rendered as unit² in print). The lexer handles this via longest-match on the registered suffix table — `#5m2` unambiguously parses as number `5` + suffix `m2`, and `#52m2` as number `52` + suffix `m2`.

---

## 5. Properties and Methods

| Property/Method | Returns | Example |
|-----------------|---------|---------|
| `.value` | float — decoded value in display-hint unit | `#3/8in.value` → `0.375` |
| `.unit` | string — display-hint unit | `#3/8in.unit` → `"in"` |
| `.family` | string | `"length"` |
| `.system` | string | `"SI"` or `"US"` |
| `.to(unit)` | converted unit value | `#1mi.to("km")` → `#1.609km` |
| `.abs()` | absolute value | `(-#5m).abs()` → `#5m` |
| `.format()` | formatted string | `#1234.5m.format()` → `"1,234.5m"` |
| `.repr()` | parseable literal string | `#12.3m.repr()` → `"#12.3m"` |
| `.toDict()` | clean dictionary | `{value: 12.3, unit: "m"}` |
| `.inspect()` | debug dictionary with internals | |
| `.toFraction()` | fraction string (Imperial) | `#3/8in.toFraction()` → `"3/8\""` |

---

## 6. Open Questions

### Decided

| # | Question | Decision | Rationale |
|---|----------|----------|-----------|
| 1 | `#` as unit sigil? | **Yes** | Unambiguous with Money's `CODE#` pattern |
| 2 | US vs British spelling? | **SI default (`-re`), US aliases** | `metre()` primary, `meter()` alias |
| 3 | Short-form constructors? | **No** | `m` and `g` collide with likely variable names |
| 4 | HCN value? | **725,760** | LCM of required fractions at µm precision |
| 5 | Temperature arithmetic? | **Treat like numbers** | No point/interval distinction (see §2.9) |
| 6 | Fraction literals both systems? | **Yes — different storage** | SI: sugar for division (→ decimal). Imperial: exact HCN storage. See §2.4 |
| 7 | Mixed number separator? | **`+` sign** | `#92+5/8in`. Universally unambiguous; hyphen reads as subtraction to non-American eyes |
| 8 | Store display hint? | **Yes** | Enables "left wins" and natural default display |
| 9 | Max scale for cross-system? | **Scale 6** (µm precision) | Matches Imperial-side precision — no point storing more |
| 10 | SI → Imperial non-fraction display? | **Decimal fallback** | Heuristic fraction-guessing adds error; explicit `.format("fraction")` for those who want it |
| 11 | Display precision? | **Context-aware defaults** | Match real-world conventions (see §2.15) |
| 13 | String interpolation format? | **No space, no sigil** | `"1.83m"` — consistent with Money |
| 14 | `unit / unit` returns a number? | **Yes** | Dimensionless ratio |
| 15 | Generic `unit()` constructor? | **Yes** | Escape hatch alongside named constructors |
| 16 | Which Imperial system? | **US Customary only** | |
| 17 | Expose internal representation? | **No — `.value` only** | `.inspect()` for debug; internals free to evolve |
| 18 | Equality across systems? | **Yes** | `#1in == #25.4mm` is true |
| 19 | Digital units representation? | **SI decimal model** | Stored in bytes |

### Deferred

| # | Question | Deferred to |
|---|----------|-------------|
| 12 | Unit types in schemas (`mass`, `unit("kg")`) | Phase 4 |
| — | Derived unit arithmetic (`#5m * #3m` → area) | Phase 4+ |
| — | Compound display formatting (`5' 3+1/4"`) | Phase 4 |
| 20 | Generic `Precise`/Rational type as a standalone numeric type? | Post-units exploration |

---

## 7. Errata in V1

The following sections in `DESIGN-units.md` still reference the earlier HCN value of **5,040** and should be updated to **725,760** (with corresponding sub-unit values of 20,160 per inch instead of 5,040):

- §4.10 — Volume example: *"With HCN = 5040, 1/3 cup = 1/3 × 8 × 5040 = 13,440"*
- §9.2 — All examples use 5,040 sub-units per inch (should be 20,160)
- §9.4 — Fraction display table uses HCN=5040 column

Additionally, §4.8 describes Fahrenheit storage using HCN = 725,760, but §5.3 describes the K × 900 unified temperature representation. These should be reconciled — the K × 900 approach (§5.3) is the preferred design as it makes temperature conversions lossless in both directions.

---

## 8. Key Invariants

These are the properties the design guarantees:

1. **Within-system arithmetic is always exact.** No rounding, ever. Integer operations on integer storage.
2. **Round-trips are always exact.** `#3/8in` → store → display → `#3/8in`. No drift.
3. **Rounding only occurs at the SI↔Imperial boundary** (length, mass, volume). Sub-micrometre precision.
4. **Temperature conversions are lossless in both directions.** The K × 900 base makes the 5/9 ratio exact.
5. **Common fractions (halves through sixty-fourths, plus thirds, fifths, sevenths, ninths) are all exact integers** in the HCN representation.
6. **Range is not a constraint.** Both systems reach ~77 AU for length, hundreds of millions of tons for mass.
7. **Equality ignores display hints.** Two values representing the same physical quantity are equal regardless of which unit they're displayed in.