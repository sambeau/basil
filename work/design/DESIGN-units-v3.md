# Measurement Units for Parsley — Design v3

**Status:** Final design — ready for specification
**Predecessors:** `DESIGN-units.md` (exploration), `DESIGN-units-v2.md` (decisions)

---

## 1. Overview

Parsley provides built-in unit-of-measurement types following the precedent set by Money: integer storage, no floating-point drift, exact round-trips, and natural syntax.

Three internal representations serve three numeric idioms:

| Representation | Storage | Idiom | Families |
|----------------|---------|-------|----------|
| **SI** | int64 count of fixed sub-units (µm, mg, B) | Decimal: `#12.3m`, `#0.5kg` | Length, Mass, Data |
| **US Customary** | int64 / HCN fixed denominator | Fractional: `#3/8in`, `#1/3cup` | Length, Mass, Volume |
| **Temperature** | int64 in K × 900 base units | Offset + ratio conversions | Temperature |

All three present the same interface: same literal syntax, same operators, same methods, same properties. The representation is purely internal.

---

## 2. Literal Syntax

### 2.1 Basic Literals

`#` + number + unit suffix:

```parsley
#12.3m          // 12.3 metres
#45oz           // 45 ounces
#100C           // 100 degrees Celsius
#64KB           // 64 kilobytes
#-6C            // negative 6 degrees Celsius
```

The `#` sigil is unambiguous with Money's `CODE#amount` pattern — the lexer distinguishes `EUR#50` (uppercase letters before `#`) from `#50m` (`#` before digits).

### 2.2 Fraction Literals

`#` + numerator `/` denominator + suffix:

```parsley
#3/8in          // three-eighths of an inch
#1/3cup         // one-third of a cup
#1/2km          // half a kilometre
```

Fractions are allowed for both SI and US Customary units. The storage differs by system:

- **US Customary:** The fraction is stored exactly as an HCN-scaled integer. `#1/3cup` preserves the fraction.
- **SI:** The fraction is syntactic sugar for division. `#1/3m` is equivalent to `#1m / 3` and is stored as `#0.333333m` (truncated to whole sub-units: 333,333 µm).

This means `#1/3m == #1m / 3`.

### 2.3 Mixed Number Literals

`#` + whole `+` numerator `/` denominator + suffix:

```parsley
#92+5/8in       // 92 and 5/8 inches
#2+3/8in        // 2 and 3/8 inches
#1+1/2lb        // 1 and 1/2 pounds
#1+1/2km        // 1.5 kilometres (SI: converted to decimal)
```

The `+` separator is used rather than a hyphen to avoid ambiguity with subtraction. The lexer recognises this within "unit literal mode" (after `#`, before suffix).

### 2.4 Negative Values

The negative sign goes inside the literal, keeping the unit as one self-contained token:

```parsley
#-6C            // negative 6 degrees Celsius (canonical literal form)
#0C - #6C       // equivalent, via arithmetic
-temperature    // unary negation of a unit variable
```

Both `#-6C` and `-#6C` produce the same value — the first is a negative literal, the second is unary negation of a positive literal. `#-6C` is the canonical form.

Note: `0 - #6C` is a **type error**, not negation. `0` is a plain number, and `scalar ± unit` is not a defined operation (see §5.1). To express "zero degrees minus six degrees," write `#0C - #6C`.

---

## 3. Internal Representation

### 3.1 SI Units

An integer count of fixed sub-units — the same "integer counting sub-units" model used by US Customary (HCN) and Temperature (K × 900). No scale factor; one int64, one base.

```go
type SIUnit struct {
    Amount      int64   // count of base sub-units (µm for length, mg for mass, B for data)
    Family      string  // "length", "mass", "data"
    DisplayHint string  // "cm", "m", "km", etc.
}
```

| Family | Sub-unit | Size | Rationale |
|--------|----------|------|-----------|
| Length | micrometre (µm) | 1 m = 1,000,000 | Matches Imperial's ~1.26 µm resolution; standard SI prefix |
| Mass | milligram (mg) | 1 g = 1,000 | Smallest common SI mass unit; `#500g` = Amount 500,000 |
| Data | byte (B) | 1 B = 1 | Already an integer; all file/storage sizes are byte-oriented |

**Range:** int64 max µm ≈ 9.2 × 10¹² m ≈ 61 AU. int64 max mg ≈ 9.2 × 10¹² kg ≈ 9,200 tonnes.

**Precision:** 1 µm — matching Imperial's ~1.26 µm per sub-yard. Cross-system conversions are precision-aligned.

### 3.2 US Customary Units

An integer numerator over a fixed Highly Composite Denominator (HCN).

```go
const HCN = 725_760    // 2⁸ × 3⁴ × 5 × 7

type ImperialUnit struct {
    Amount      int64   // value = Amount / HCN, in base US unit
    Family      string  // "length", "mass", "volume"
    DisplayHint string  // "in", "ft", "yd", "mi", etc.
}
```

| Family | Base unit | Rationale |
|--------|-----------|-----------|
| Length | yard | ~1m scale matches metre; 1 yd = 36 in gives 20,160 sub-units per inch |
| Mass | ounce | Smallest common US mass unit; 1 lb = 16 oz is exact |
| Volume | quart | ~1L scale matches litre; clean ratios to cups, pints, gallons |

#### Why 725,760

The HCN is the LCM of the divisibility requirements, not a number chosen for having "many divisors":

```
1 yard = 36 inches. For fraction 1/q of an inch to be exact, HCN must be divisible by 36 × q.

  36 × 64 = 2,304   →  2⁸ × 3²    (64ths: machining, fine carpentry)
  36 × 9  = 324     →  2² × 3⁴    (9ths)
  36 × 7  = 252     →  2² × 3² × 7 (7ths)
  36 × 5  = 180     →  2² × 3² × 5 (5ths)

  LCM = 2⁸ × 3⁴ × 5 × 7 = 725,760
```

This yields **20,160 sub-units per inch**. Every fraction from halves to sixty-fourths, plus thirds, fifths, sevenths, and ninths, is an exact integer.

**Precision:** 914,400 µm per yard / 725,760 = ~1.26 µm per sub-yard — matching SI's 1 µm sub-unit precision.

**Range:** int64 max / 725,760 ≈ 1.27 × 10¹³ yards ≈ 77 AU.

#### Fraction exactness table

| Fraction of inch | × 20,160 | Where it appears |
|------------------|----------|------------------|
| 1/2 | 10,080 | Everywhere |
| 1/3 | 6,720 | Cooking (1/3 cup), sewing |
| 1/4 | 5,040 | Cooking, carpentry |
| 1/5 | 4,032 | — |
| 1/6 | 3,360 | Recipe scaling |
| 1/7 | 2,880 | — |
| 1/8 | 2,520 | Cooking (1/8 tsp), carpentry (kerf) |
| 1/9 | 2,240 | — |
| 1/10 | 2,016 | — |
| 1/12 | 1,680 | Inches per foot |
| 1/16 | 1,260 | Ounces per pound, tape measures |
| 1/32 | 630 | Precision woodworking |
| 1/64 | 315 | Machining, fine carpentry |

All exact integers. All arithmetic on these values is integer addition — no rounding, ever.

### 3.3 Temperature Units

A unified representation for all temperature scales. Not split by measurement system.

```go
type TempUnit struct {
    Amount      int64   // value in sub-kelvins (K × 900)
    DisplayHint string  // "K", "C", or "F"
}
```

**Why K × 900:** The Celsius-to-Fahrenheit ratio is 9/5. With 900 sub-units per kelvin, 1°C = 900 sub-K and 1°F = 500 sub-K — both exact integers. All conversions are integer arithmetic with no rounding.

#### Conversion formulas

| Direction | Formula |
|-----------|---------|
| °C → base | `base = (C + 273.15) × 900` |
| °F → base | `base = (F + 459.67) × 500` |
| K → base | `base = K × 900` |
| base → °C | `C = (base − 245,835) / 900` |
| base → °F | `F = (base − 229,835) / 500` |
| base → K | `K = base / 900` |

**Verification:** `#100C` → 373.15 × 900 = 335,835 → (335,835 − 229,835) / 500 = 212°F ✓
**Verification:** `#98.6F` → 558.27 × 500 = 279,135 → (279,135 − 245,835) / 900 = 37°C ✓

Temperature addition and subtraction treat values as plain numbers — no point-vs-interval distinction. `#20C + #10C = #30C`. Scalar multiplication and division of temperature are errors (see §5.1).

### 3.4 Display Hint

Every unit value carries a **display hint** — the unit suffix the user originally wrote. The hint determines default display and is propagated by the "left side wins" rule (§5.2). It is **not** part of the value's identity: two values with different hints but the same underlying amount are equal.

---

## 4. Unit Families

### 4.1 Length

| Suffix | Unit | System | Ratio to base |
|--------|------|--------|---------------|
| `mm` | millimetre | SI | × 1/1000 m |
| `cm` | centimetre | SI | × 1/100 m |
| `m` | metre | SI | base |
| `km` | kilometre | SI | × 1000 m |
| `in` | inch | US | × 1 in (= 1/36 yd) |
| `ft` | foot | US | × 12 in |
| `yd` | yard | US | base (= 36 in) |
| `mi` | mile | US | × 63,360 in |

**Cross-system bridge:** 1 in = 0.0254 m (exact, by international definition).

### 4.2 Mass

| Suffix | Unit | System | Ratio to base |
|--------|------|--------|---------------|
| `mg` | milligram | SI | × 1/1000 g |
| `g` | gram | SI | base |
| `kg` | kilogram | SI | × 1000 g |
| `oz` | ounce | US | base |
| `lb` | pound | US | × 16 oz |

**Cross-system bridge:** 1 lb = 453.59237 g (exact, by international definition).

### 4.3 Temperature

| Suffix | Unit | System |
|--------|------|--------|
| `K` | kelvin | SI |
| `C` | degree Celsius | SI |
| `F` | degree Fahrenheit | US |

All stored in the unified K × 900 representation. See §3.3 for conversion formulas.

### 4.4 Digital Information

| Suffix | Unit | Multiplier |
|--------|------|------------|
| `B` | byte | base |
| `kB` | kilobyte | × 10³ |
| `MB` | megabyte | × 10⁶ |
| `GB` | gigabyte | × 10⁹ |
| `TB` | terabyte | × 10¹² |
| `KiB` | kibibyte | × 2¹⁰ |
| `MiB` | mebibyte | × 2²⁰ |
| `GiB` | gibibyte | × 2³⁰ |
| `TiB` | tebibyte | × 2⁴⁰ |

Stored using the SI decimal model with byte as the base unit. All conversions are exact integer ratios. The base unit is byte, not bit — the target domain is file sizes and storage, not network throughput.

### 4.5 Volume (Phase 2)

| Suffix | Unit | System | Ratio to base |
|--------|------|--------|---------------|
| `mL` | millilitre | SI | × 1/1000 L |
| `L` | litre | SI | base |
| `floz` | fluid ounce | US | base |
| `cup` | cup | US | × 8 floz |
| `pt` | pint | US | × 16 floz |
| `qt` | quart | US | × 32 floz |
| `gal` | gallon | US | × 128 floz |

**Cross-system bridge:** 1 gal = 3.785411784 L (exact, by definition: 231 in³).

### 4.6 Area (Phase 3)

| Suffix | Unit | System |
|--------|------|--------|
| `mm2` | square millimetre | SI |
| `cm2` | square centimetre | SI |
| `m2` | square metre | SI |
| `km2` | square kilometre | SI |
| `in2` | square inch | US |
| `ft2` | square foot | US |
| `yd2` | square yard | US |
| `ac` | acre | US |
| `mi2` | square mile | US |

Area suffixes use the `unit2` convention (rendered as unit² in formatted output). Area is entered as its own family — not derived from length × length.

**Cross-system bridge:** 1 in² = 0.00064516 m² (exact).

---

## 5. Arithmetic

### 5.1 Allowed Operations

| Operation | Result type | Notes |
|-----------|-------------|-------|
| unit + unit (same family) | unit | Left side wins display hint and system |
| unit − unit (same family) | unit | Left side wins display hint and system |
| unit × scalar | unit | **Not temperature** — see below |
| scalar × unit | unit | Commutative. **Not temperature** |
| unit / scalar | unit | May truncate (same as Money). **Not temperature** |
| unit / unit (same family) | plain number | Dimensionless ratio. **Not temperature** |
| −unit | unit | Negation |
| unit + unit (different family) | **error** | |
| unit × unit | **error** | Derived units deferred to Phase 4+ |
| scalar + unit | **error** | No implicit promotion of scalars to units |
| scalar − unit | **error** | Use `#0C - #6C`, not `0 - #6C` |
| scalar / unit | **error** | |
| temp × scalar | **error** | Offset scales make this inconsistent (see §5.2) |
| scalar × temp | **error** | |
| temp / scalar | **error** | |
| temp / temp | **error** | Ratio of offset-scale readings is meaningless |

### 5.2 Temperature Arithmetic Restrictions

Celsius and Fahrenheit are **offset scales** — their zero points are arbitrary, not absolute. This makes scalar multiplication inconsistent:

```parsley
#20C == #68F        // true (same physical temperature)
#20C * 2            // if allowed: #40C = #104F
#68F * 2            // if allowed: #136F = #57.78C
                    // #40C ≠ #136F — a == b but a*2 ≠ b*2
```

There is no way to define temperature multiplication that is both intuitive and algebraically consistent. Therefore, scalar multiplication, scalar division, and temperature-over-temperature division are all errors for the temperature family.

Temperature addition and subtraction are allowed under "treat like numbers" semantics: `#20C + #10C = #30C`, `#100C - #37C = #63C`.

> **Future:** If scientific users need to scale absolute temperatures, a kelvin-based escape hatch could be added — perform arithmetic in K where the offset is zero, then convert back.

### 5.3 Left Side Wins

When combining units from the same family, the left operand determines the display hint and measurement system of the result:

```parsley
#1cm + #1in         // → #3.54cm (result in cm)
#1in + #1cm         // → #1+3/8in (result in inches, approximate)
#254cm + #1m        // → #354cm
#1m + #254cm        // → #3.54m
```

### 5.4 Within-System Arithmetic

Always exact. Integer operations on integer storage. No rounding. All SI values are in the same sub-unit base (µm, mg, or B), so addition and subtraction are direct integer operations — no promotion or normalisation needed.

```parsley
#1/3cup + #1/3cup + #1/3cup    // = #1cup (exact)
#3/8in + #5/8in                // = #1in (exact)
#12.3m + #0.7m                 // = #13m (exact, 12300000 + 700000 = 13000000 µm)
#5cm + #3mm                    // = #5.3cm (exact, 50000 + 3000 = 53000 µm)
```

### 5.5 Cross-System Arithmetic

Rounding occurs only at the SI↔US boundary, using the bridge ratios from §4. The left operand determines which system the result is in; the right operand is converted.

Precision is micrometre-level on both sides (~1 µm for SI, ~1.26 µm for US Customary).

```parsley
#10cm + #1in        // Convert 1in → 2.54cm. Result: #12.54cm (exact)
#1in + #1cm         // Convert 1cm → sub-inches (rounds). Result: ~#1.3937in
```

Imperial → SI is often exact (bridge ratios are defined as finite SI decimals). SI → Imperial may round.

### 5.6 Comparison

Equality normalises across systems via bridge ratios:

```parsley
#1in == #25.4mm     // true
#0C == #32F         // true
#1024B == #1KiB     // true
#254cm == #2.54m    // true
```

---

## 6. Constructors and Conversion

### 6.1 Named Constructors

The primary API. Full unit name (plural). Each doubles as a conversion function:

```parsley
metres(123)          // create: #123m
metres(#12in)        // convert: #0.3048m
fahrenheit(#100C)    // convert: #212F
kilograms(#2.2lb)    // convert: ~#0.998kg
```

Constructor names are plural and use SI spelling (`-res`). US spelling is available as an alias:

| Primary | Alias |
|---------|-------|
| `metres()` | `meters()` |
| `centimetres()` | `centimeters()` |
| `kilometres()` | `kilometers()` |
| `litres()` | `liters()` |
| `millilitres()` | `milliliters()` |

No short-form constructors (`m()`, `g()`, etc.) — these collide with common variable names.

### 6.2 Generic Constructor

For dynamic use when the target unit is a runtime value:

```parsley
unit(123, "m")              // create: #123m
unit(#12in, "m")            // convert: #0.3048m

let target = getUserChoice()
unit(distance, target)      // convert to user-chosen unit
```

### 6.3 The `.to()` Method

For chaining and dynamic targets:

```parsley
#1mi.to("km")               // #1.609km
distance.to(targetUnit)     // convert to runtime-chosen unit
```

---

## 7. Properties and Methods

### 7.1 Properties

| Property | Type | Description | Example |
|----------|------|-------------|---------|
| `.value` | float | Decoded value in the display-hint unit | `#3/8in.value` → `0.375` |
| `.unit` | string | Display-hint unit suffix | `#3/8in.unit` → `"in"` |
| `.family` | string | Unit family | `#3/8in.family` → `"length"` |
| `.system` | string | Measurement system | `#3/8in.system` → `"US"` |

Internal representation is not exposed. This keeps the API stable as internals evolve.

### 7.2 Methods

| Method | Returns | Description |
|--------|---------|-------------|
| `.to(unit)` | unit value | Convert to another unit (see §6.3) |
| `.abs()` | unit value | Absolute value |
| `.format()` | string | Formatted with locale defaults |
| `.format(options)` | string | Formatted with explicit precision, style, etc. |
| `.repr()` | string | Parseable literal: `"#12.3m"` |
| `.toDict()` | dict | `{value: 12.3, unit: "m", family: "length", system: "SI"}` |
| `.inspect()` | dict | Debug output including internal representation |
| `.toFraction()` | string | Fraction string for US Customary values: `"3/8\""` |

---

## 8. Display and Formatting

### 8.1 Default Display

US Customary values display as fractions (via GCD); SI values display as decimals.

**US Customary fraction display algorithm:**

1. Compute `g = GCD(amount, HCN)`
2. `numerator = amount / g`, `denominator = HCN / g`
3. If `denominator == 1`: display as integer
4. If `numerator > denominator`: extract whole part, display as mixed number with `+`
5. If `denominator` is a common denominator (2, 3, 4, 5, 6, 7, 8, 10, 12, 16, 32, 64): display as fraction
6. Otherwise: fall back to decimal

**SI decimal display defaults** match everyday conventions:

| Context | Decimal places | Examples |
|---------|---------------|----------|
| Lengths in m | 2 | `1.83m`, `2.54m` |
| Lengths in cm | 1 | `12.7cm` |
| Lengths in mm | 0 | `25mm` |
| Mass in kg | 2 | `2.50kg` |
| Mass in g | 0 | `500g` |
| Celsius | 1 | `37.0C`, `22.2C` |
| Fahrenheit | 1 | `98.6F`, `72.0F` |
| Digital | 0 | `1024B`, `2GB` |

All defaults are overridable via `.format(precision)`.

### 8.2 String Interpolation

No `#` sigil, no space between value and unit — consistent with Money's interpolation behaviour:

```parsley
let height = #1.83m
`Height: {height}`          // "Height: 1.83m"
`Height: {height.format()}`  // "Height: 1.83m" (with locale formatting)
```

### 8.3 PLN (Parsley Literal Notation)

Unit values use the literal syntax in PLN output:

```parsley
{height: #1.83m, weight: #75kg, temperature: #37C}
```

### 8.4 `.repr()` Output

Always produces a parseable literal:

```parsley
#3/8in.repr()       // "#3/8in"
#12.3m.repr()       // "#12.3m"
#92+5/8in.repr()    // "#92+5/8in"
```

---

## 9. Cross-System Bridge Ratios

All ratios are exact by international definition. Rounding only occurs when converting the result into the target system's storage format.

| Family | Ratio | Source |
|--------|-------|--------|
| Length | 1 in = 0.0254 m | International yard and pound agreement (1959) |
| Mass | 1 lb = 453.59237 g | International yard and pound agreement (1959) |
| Volume | 1 gal = 3.785411784 L | Derived from 1 gal = 231 in³ |
| Area | 1 in² = 0.00064516 m² | Derived from length bridge |

### Within-System Ratios

All within-system ratios are exact integers or simple fractions. No rounding.

**SI Length:** mm = m/1000, cm = m/100, km = m × 1000
**US Length:** ft = 12 in, yd = 36 in, mi = 63,360 in
**US Mass:** lb = 16 oz
**US Volume:** cup = 8 floz, pt = 16 floz, qt = 32 floz, gal = 128 floz

---

## 10. Implementation Phases

### Phase 1 — MVP

**Families:** Length, Mass, Digital Information.

Covers all core challenges: dual SI/US representation, cross-system conversion, fraction literals and display, within-system exact arithmetic.

**Deliverables:**
- Lexer: `#` + number + suffix; `#` + fraction + suffix; `#` + mixed number + suffix
- Parser: UnitLiteral, FractionUnitLiteral, MixedUnitLiteral AST nodes
- Evaluator: SIUnit and ImperialUnit objects
- Arithmetic: same-system (exact), cross-system (bridge ratio), scalar
- Conversion tables: within-system ratios and cross-system bridge ratios
- Display: decimal for SI, fraction-via-GCD for US Customary, decimal fallback
- Constructors: named constructors (plural form) for all Phase 1 units, generic `unit()` constructor
- Properties: `.value`, `.unit`, `.family`, `.system`
- Methods: `.to()`, `.format()`, `.abs()`, `.repr()`, `.toDict()`, `.inspect()`, `.toFraction()`
- Error messages for cross-family arithmetic

### Phase 2 — Temperature + Volume

**Temperature** adds the K × 900 unified representation and offset conversions.
**Volume** adds a family where exact fractions matter most (1/3 cup, 1/4 tsp).

### Phase 3 — Area

Area as its own unit family with `unit2` suffixes. Entered directly, not derived from length × length.

### Phase 4 — Polish

- Compound display formatting (`5' 3+1/4"`, `2lb 5oz`)
- Schema integration for unit types
- Derived unit arithmetic (`#5m * #3m` → area) if demanded
- Additional families (angle, energy) if demanded

---

## 11. Design Guarantees

1. **Within-system arithmetic is always exact.** Integer operations on integer storage. No rounding. No scale promotion — all values in a family share the same fixed sub-unit base.
2. **Round-trips are always exact.** `#3/8in` → store → display → `#3/8in`. No drift.
3. **Rounding only occurs at the SI↔US boundary** (length, mass, volume). Maximum error: ~1.26 µm (one Imperial sub-unit).
4. **Temperature conversions are lossless in both directions.** The K × 900 base makes the 5/9 ratio exact integer arithmetic.
5. **Common fractions are exact integers.** Halves through sixty-fourths, plus thirds, fifths, sevenths, and ninths — all exact in the HCN representation.
6. **Range is not a constraint.** Both systems exceed 61 AU (SI) / 77 AU (US) for length and thousands of tonnes for mass within int64.
7. **Equality ignores display hints.** Two values representing the same physical quantity are equal regardless of the unit they are displayed in.
8. **Uniform architecture.** All three representations use the same model: a single int64 counting fixed sub-units. SI uses µm/mg/B, US Customary uses sub-yards/sub-ounces (HCN), Temperature uses sub-kelvins (K × 900). No scale factors, no auxiliary fields.
9. **Error messages are human-first.** Every unit error uses the structured error catalogue with a plain-English template (says what went wrong) and actionable hints (says what to do instead, with corrected code). Fuzzy matching suggests the closest valid suffix for typos. No jargon, no internal type names — a casual programmer should be able to read the error, understand it, and fix their code without consulting documentation.