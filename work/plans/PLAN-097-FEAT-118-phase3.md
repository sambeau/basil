---
id: PLAN-097
feature: FEAT-118
title: "Implementation Plan for Measurement Units — Phase 3 (Area + Volume Scale)"
status: in-progress
created: 2026-02-14
updated: 2026-02-14
---

# Implementation Plan: FEAT-118 Phase 3 — Area + Volume Scale

## Overview

Add the **Area** unit family to the existing measurement unit infrastructure: SI (`mm2`, `cm2`, `m2`, `km2`) and US Customary (`in2`, `ft2`, `yd2`, `ac`, `mi2`). Area follows the established dual-system pattern from Phases 1–2 with no special arithmetic rules (unlike Temperature).

Additionally, extend the existing **Volume** family with the `kL` (kilolitre) suffix, leveraging the new Scale infrastructure to handle large volumes (reservoirs, lakes, industrial quantities).

Phase 3 introduces two significant architectural changes:

1. **Decimal Scale** — A new `Scale int` field on the `Unit` struct that extends range beyond int64 for derived units. The true value in base sub-units = `Amount × 10^Scale`. For all existing families, Scale = 0 always (zero behaviour change). For area and large volumes, Scale activates only when values exceed int64 range, enabling representation of Earth's surface, large bodies of water, and beyond while preserving full precision for small values.

2. **PLN Parser Fix** — The PLN parser's suffix extraction must handle suffixes containing digits (`m2`, `in2`, `km2`, etc.).

Phase 1 delivered the core infrastructure (lexer, parser, evaluator, PLN, display, methods, errors) for Length, Mass, and Data. Phase 2 added Temperature and Volume. Phase 3 extends the infrastructure — most area-specific changes are additive (new entries in tables, new branches in existing switch statements). The Scale infrastructure is the main structural change, and once built it immediately benefits both area and volume.

**Design doc:** `work/design/DESIGN-units-v3.md` (authoritative — see §4.6, §9)

## Prerequisites

- [x] Phase 1 complete (PLAN-095)
- [x] Phase 2 complete (PLAN-096)
- [ ] Phase 2 PR merged to main (branch `feat/FEAT-118-measurement-units`)
- [x] Design doc `DESIGN-units-v3.md` §4.6 (Area) reviewed

## Architecture Notes

### The 2D Scaling Problem

For 1D units, int64 range is generous (e.g., length max ≈ 61 AU in µm). For 2D units, the exponents double, creating a tension between precision and range:

| Family | Base sub-unit | Largest suffix multiplier | Max in largest | Practical? |
|--------|--------------|--------------------------|----------------|------------|
| Length | µm | km = 10⁹ | 9.22 billion km | ✅ Way beyond anything |
| Volume | nL | L = 10⁹ | 9.22 billion L | ⚠️ Adequate for everyday, but not reservoirs/lakes |
| **Area** | **mm²** | **km² = 10¹²** | **9.2 million km²** | **❌ Can't hold Earth (510M km²)** |

The 12 orders of magnitude from mm² to km² consume most of int64's ~18.9 digit capacity, leaving only ~7 digits for the actual value. Meanwhile US area (in² base) reaches 2.3 billion mi² — a 648× disparity in equivalent terms.

### The Solution: Decimal Scale

Instead of coarsening the base sub-unit (which sacrifices small-area precision) or choosing different bases for SI vs US (which creates range disparity), we add a **decimal scale exponent** to the Unit struct:

```
type Unit struct {
    Amount      int64
    Family      string
    System      string
    DisplayHint string
    Scale       int    // decimal exponent: true value = Amount × 10^Scale base-sub-units
}
```

**True value in base sub-units = Amount × 10^Scale**

Key properties:

- **Scale = 0 for all existing families** — zero behaviour change for length, mass, data, volume, temperature. The fast path (`Scale == 0`) is the common case.
- **Scale ≥ 0 always** — we never generate negative Scale. The parser, constructors, and all operations clamp to Scale ≥ 0. (int is signed so arithmetic handles negatives safely, but no code path produces them.)
- **Scale activates only on overflow** — for area values up to ~9.2 million km², Scale stays 0 and behaviour is identical to existing families. Scale > 0 only occurs for larger values, exactly when mm²-level precision is meaningless anyway.
- **Applies to both SI and US** — cross-system conversion of large SI values can overflow on the US side too (e.g., Jupiter's surface in in²). Scale is system-agnostic.
- **Uses `int` type** — Go struct padding means int8 vs int64 costs the same memory (Unit struct goes from 56 to 64 bytes either way). Using `int` eliminates casting noise and int8 overflow traps in scale arithmetic.

### Scale Examples

| Literal | Amount | Scale | True value (mm²) | Notes |
|---------|--------|-------|-------------------|-------|
| `#5mm2` | 5 | 0 | 5 | Exact, no scale needed |
| `#5.3m2` | 5,300,000 | 0 | 5.3 × 10⁶ | Fits in int64 |
| `#100m2` | 100,000,000 | 0 | 10⁸ | Fits in int64 |
| `#9000000km2` | 9 × 10¹⁸ | 0 | 9 × 10¹⁸ | Just fits (< 9.22 × 10¹⁸) |
| `#510000000km2` | 5.1 × 10¹⁸ | 2 | 5.1 × 10²⁰ | Earth's surface ✅ |
| `#61400000000km2` | 6.14 × 10¹⁸ | 4 | 6.14 × 10²² | Jupiter's surface ✅ |

### Scale Arithmetic

**Same-scale (fast path — vast majority of cases):**
```
#5m2 + #3m2
Both Scale=0: result.Amount = 5,000,000 + 3,000,000 = 8,000,000. Scale=0.
```

**Mixed-scale (align to smaller scale, overflow falls back to larger):**
```
#510000000km2 + #3mm2
a = (Amount=5.1e18, Scale=2), b = (Amount=3, Scale=0)
Try aligning a to Scale=0: 5.1e18 × 10² → overflow.
Fall back: align b to Scale=2: 3 / 10² → 0 (truncated).
Result: (5.1e18, Scale=2).
The 3mm² is lost — but 3mm² relative to 510 million km² is below any meaningful precision.
```

**Same-scale addition that produces overflow:**
```
#6000000km2 + #5000000km2
a = (6e18, Scale=0), b = (5e18, Scale=0)
Sum = 11e18 → overflow int64.
Normalize: result = (11e17, Scale=1) or (11e16, Scale=2), etc.
Pick smallest Scale that fits: Amount=1.1e18, Scale=1.
```

**Scalar multiplication:**
```
#5m2 * 3 → Amount=15,000,000, Scale=0. No scale needed.
#5000000km2 * 3 → 5e18 × 3 = 15e18 → overflow. Result: (15e17, Scale=1).
```

**Cross-system conversion:**
```
#510000000km2 → convert to mi2
SI true = 5.1e20 mm². Bridge: us_in2 = si_mm2 × 25 / 16129.
Compute with scale: (5.1e18, Scale=2) → multiply Amount by 25 = 1.275e20 → overflow.
Handle: factor out scale, compute in stages, adjust Scale on result.
Result in in² → divide by mi2 sub-units for display.
```

**Comparison:**
```
#5m2 > #3km2
(5e6, Scale=0) vs (3e12, Scale=0). 5e6 < 3e12 → false. Simple.

#510000000km2 > #9000000km2
(5.1e18, Scale=2) vs (9e18, Scale=0). Align to Scale=0: 5.1e18 × 10² → overflow.
Align to Scale=2: 9e18 / 10² = 9e16. Compare: 5.1e18 > 9e16 → true. ✓
```

### Precision Characteristics

The decimal scale introduces **lossy arithmetic only when combining values that differ by more than ~18 orders of magnitude**. At that point, the small value is below any meaningful precision — this is the same behaviour as IEEE 754 floating point, but with integer exactness for all same-magnitude operations.

Within Scale=0 range (all values up to ~9.2M km² / ~2.3B mi²), arithmetic is fully exact, identical to existing families.

### SI Area Storage

Base sub-unit: **mm²** (square millimetre). Preserves full precision for small areas (arts, crafts, engineering).

| Suffix | Unit | Sub-units (mm²) |
|--------|------|------------------|
| `mm2` | square millimetre | 1 |
| `cm2` | square centimetre | 100 |
| `m2` | square metre | 1,000,000 |
| `km2` | square kilometre | 1,000,000,000,000 (10¹²) |

**Scale=0 range:** int64 max ≈ 9.22 × 10¹⁸ mm² ≈ 9.2 million km².
**With Scale:** effectively unlimited. Earth (510M km²), Jupiter (61.4B km²), the Sun (6.08T km²) all representable.

### US Area Storage

Area does **not** use HCN (725,760) as a fraction denominator. The HCN exists to make common fractions of inches, ounces, and fluid ounces exact — but fractions of square inches are essentially never used in practice. Using HCN for area would severely limit range (max ~3,158 mi² with in² base).

Instead, US Area uses **plain integer in²** as the sub-unit:

| Suffix | Unit | Sub-units (in²) |
|--------|------|------------------|
| `in2` | square inch | 1 |
| `ft2` | square foot | 144 |
| `yd2` | square yard | 1,296 |
| `ac` | acre | 6,272,640 |
| `mi2` | square mile | 4,014,489,600 |

**Scale=0 range:** int64 max ≈ 9.22 × 10¹⁸ in² ≈ 2.3 billion mi².
**With Scale:** effectively unlimited (though Scale > 0 will rarely occur on the US side — it only arises from cross-system conversion of very large SI values or extreme arithmetic).

**Fractions of large US area units** divide cleanly without HCN:
- `#1/3ac` = 6,272,640 / 3 = 2,090,880 in² (exact)
- `#1/3mi2` = 4,014,489,600 / 3 = 1,338,163,200 in² (exact)
- `#1/4mi2` = 4,014,489,600 / 4 = 1,003,622,400 in² (exact)

**Small fractions:** `#1/3in2` truncates to 0 in² (like SI fractions). This is consistent and documented. Area fractions at the in² level are not a practical use case.

### Cross-System Bridge

1 in² = 645.16 mm² = 16,129/25 mm² (exact by definition, from 1 in = 25.4 mm).

Bridge constants (integer ratio for multiply-then-divide):

```
US→SI: si_mm2 = us_in2 × 16,129 / 25     (max truncation error: 24/25 mm² ≈ 0.96 mm²)
SI→US: us_in2 = si_mm2 × 25 / 16,129     (max truncation error: 16,128/16,129 in² ≈ 1 in²)
```

**Verification:** 1 in² → 1 × 16,129 / 25 = 645 mm² (exact: 645.16, error: 0.16 mm², 0.025%). 1 m² → 1,000,000 × 25 / 16,129 = 1,550 in² (exact: 1,550.003, error: 0.003 in²). 1 ac → 6,272,640 × 16,129 / 25 = 4,046,856,422 mm² ≈ 4,046.86 m² (exact: 4,046.8564224 m²).

By keeping mm² as the SI base (rather than coarsening to cm²), we preserve the best bridge precision (0.025% worst case at 1 in²). With a cm² base the worst case would be 7% at 1 in².

**Scale-aware bridge:** When converting scaled values, the bridge multiplication (`Amount × 16,129`) may overflow. The bridge functions must detect overflow and shift Scale upward before multiplying. See Task 0 for the helper functions.

### US Area Storage Architecture

Because US Area uses integer in² (not HCN fractions), the code needs a small structural change. The existing `USSubUnitsPerUnit` table and `ConvertUSToSI`/`ConvertSIToUS` functions all assume HCN-based storage. For area, the bridge constants and sub-unit table have different semantics.

**Decision: Extend existing tables.** The existing `ConvertUSToSI` and `ConvertSIToUS` already switch on family. Adding `FamilyArea` branches with different bridge constants is minimal and consistent. The `usSubUnitsPerUnit` map already contains entries with different semantics per family (length base = sub-yard, mass base = sub-ounce, volume base = sub-floz). Area adds in² base — same pattern, different denominator. The evaluator code that reads from these tables doesn't assume HCN; it just uses the returned value as "sub-units per display unit".

### Display Conventions

- **SI area:** Decimal display, same as other SI families. `#12.5m2`, `#0.5km2`.
- **US area:** Decimal display (not fractions — area fractions are uncommon). `#1500ft2`, `#2.5ac`.
- **String interpolation:** No `#` sigil, no space: `"12.5m2"`, `"2.5ac"`.
- **Formatted output:** Suffix `2` is literal (no superscript in PLN/code). Future Phase 4 could add `.format()` options for `m²` Unicode rendering.
- **Scaled values:** Display divides `Amount × 10^Scale` by the suffix's sub-units-per-unit. For Scale=0, this is identical to existing display logic.

### PLN Parser Fix: Digit-Containing Suffixes

The PLN parser (`pln/parser.go`, `parseUnit`) extracts the suffix by scanning **backwards** from the end of the literal, stopping at the first digit. This breaks for suffixes like `m2`, `in2`, `km2` where the suffix itself contains a digit:

```
#5m2 → body "5m2" → backward scan stops at '2' (digit) → numStr="5m2", suffix="" → ERROR
```

**Fix:** Replace the backward character-class scan with a forward longest-match approach using `LookupUnitSuffix`. After separating the `#` sigil and optional sign, try all possible split points between numeric and suffix, checking each candidate suffix against the unit table. Pick the split that yields the longest valid suffix. This mirrors the main lexer's strategy.

## Tasks

### Task 0: Scale Infrastructure — Unit Struct and Helpers

**Files:** `pkg/parsley/evaluator/evaluator.go`, `pkg/parsley/evaluator/unit_scale.go` (new)
**Estimated effort:** Medium

Add the `Scale` field to the `Unit` struct and implement scale-aware arithmetic helpers. This task is the foundation for all subsequent area work and is designed to have zero impact on existing families.

**Changes:**

1. **Add `Scale int` to Unit struct:**
   ```
   type Unit struct {
       Amount      int64
       Family      string
       System      string
       DisplayHint string
       Scale       int  // decimal exponent: true value = Amount × 10^Scale base-sub-units. Normally 0.
   }
   ```
   All existing Unit construction sites produce Scale=0 (the zero value of int), so no changes are needed at call sites.

2. **Create `unit_scale.go`** with the following helpers:

   a. `scaleAlign(aAmount int64, aScale int, bAmount int64, bScale int) (int64, int64, int)`
      — Aligns two (Amount, Scale) pairs to a common Scale for addition/subtraction/comparison.
      — Tries to align to the smaller Scale first (maximum precision).
      — If aligning the larger-Scale value overflows, falls back to the larger Scale (truncating the smaller-Scale value).
      — Returns aligned amounts and the common Scale.

   b. `scaleNormalize(amount int64, scale int) (int64, int)`
      — After arithmetic, if Amount overflows int64, increases Scale and reduces Amount.
      — Used after addition, subtraction, and scalar multiplication.
      — Also reduces Scale if Amount has trailing zeros and Scale > 0 (optional optimization for cleaner PLN output).

   c. `scaleMulDiv(amount int64, scale int, mulNum int64, mulDen int64) (int64, int)`
      — Scale-aware multiply-then-divide for bridge conversions.
      — Detects overflow in `amount × mulNum` and shifts Scale up before multiplying.
      — Returns result (Amount, Scale).

   d. `scalePow10(n int) int64`
      — Returns 10^n as int64. Panics or returns max for n > 18. Used by alignment helpers.

   e. `scaleCmp(aAmount int64, aScale int, bAmount int64, bScale int) int`
      — Compare two scaled values. Returns -1, 0, +1.
      — Uses `scaleAlign` internally.

   f. `scaleTrueValue(amount int64, scale int) float64`
      — Returns `float64(amount) * math.Pow10(scale)`. Used by display and `.value` property.

3. **Update all arithmetic call sites** in `eval_unit_infix.go` to use scale-aware helpers:
   - Addition: `scaleAlign` → add → `scaleNormalize`
   - Subtraction: `scaleAlign` → subtract → `scaleNormalize`
   - Scalar multiply: multiply Amount → `scaleNormalize`
   - Scalar divide: divide Amount (Scale unchanged unless amount becomes 0 and Scale > 0)
   - Unit/unit ratio: `scaleAlign` → divide amounts
   - Comparison: `scaleCmp`

   **Critical: for Scale=0 (all existing families), these helpers must produce identical results to the current code.** The helpers should fast-path on `aScale == 0 && bScale == 0` to keep the common case trivial and verifiable.

4. **Update bridge conversion functions** (`ConvertUSToSI`, `ConvertSIToUS`) to use `scaleMulDiv` instead of direct `amount * num / den`. For Scale=0, the result must be identical to current behaviour.

5. **Update display functions** in `unit_display.go` to handle Scale:
   - `unitSIDisplay`: if Scale > 0, use `scaleTrueValue` to compute the float64 value for formatting.
   - `unitUSDisplay`: same approach.
   - `unitDisplayValue`: account for Scale.
   - For Scale=0, existing code paths are unchanged.

6. **Update comparison operators** in `eval_unit_infix.go` to use `scaleCmp`.

**Fast-path guarantee:** Every helper must check `if aScale == 0 && bScale == 0` (or just `scale == 0`) and take the existing code path. This ensures:
- Zero behaviour change for length, mass, data, volume, temperature.
- No performance regression.
- Easy verification: existing test suite passes unchanged.

**Validation:**
- All existing unit tests pass unchanged (Scale=0 throughout).
- `scaleAlign(5, 0, 3, 0)` → `(5, 3, 0)` — trivial fast path.
- `scaleAlign(5_100_000_000_000_000_000, 2, 3, 0)` → `(5_100_000_000_000_000_000, 0, 2)` — falls back to larger Scale.
- `scaleNormalize(11_000_000_000_000_000_000, 0)` → overflow → `(1_100_000_000_000_000_000, 1)`.
- `scaleMulDiv(5_100_000_000_000_000_000, 2, 16129, 25)` → handles overflow from ×16129.
- `scaleCmp(5_100_000_000_000_000_000, 2, 9_000_000_000_000_000_000, 0)` → +1 (5.1e20 > 9e18).

---

### Task 1: Unit Tables — Area Constants and Suffixes

**Files:** `pkg/parsley/evaluator/unit_tables.go`
**Estimated effort:** Small

Add area family constant, suffix entries, sub-unit tables, bridge constants, display config, and constructor names.

**Changes:**

1. Add `FamilyArea = "area"` to family constants.

2. Add area entries to `unitSuffixTable`:
   ```
   "mm2": {Suffix: "mm2", Family: FamilyArea, System: SystemSI},
   "cm2": {Suffix: "cm2", Family: FamilyArea, System: SystemSI},
   "m2":  {Suffix: "m2",  Family: FamilyArea, System: SystemSI},
   "km2": {Suffix: "km2", Family: FamilyArea, System: SystemSI},
   "in2": {Suffix: "in2", Family: FamilyArea, System: SystemUS},
   "ft2": {Suffix: "ft2", Family: FamilyArea, System: SystemUS},
   "yd2": {Suffix: "yd2", Family: FamilyArea, System: SystemUS},
   "ac":  {Suffix: "ac",  Family: FamilyArea, System: SystemUS},
   "mi2": {Suffix: "mi2", Family: FamilyArea, System: SystemUS},
   ```

3. Add SI area sub-units to `siSubUnitsPerUnit` (base = mm²):
   ```
   "mm2": 1,
   "cm2": 100,
   "m2":  1_000_000,
   "km2": 1_000_000_000_000,
   ```

4. Add US area sub-units to `usSubUnitsPerUnit` (base = in², NOT HCN-based):
   ```
   "in2": 1,
   "ft2": 144,
   "yd2": 1_296,
   "ac":  6_272_640,
   "mi2": 4_014_489_600,
   ```

5. Add area bridge constants:
   ```
   AreaBridgeSINumerator   int64 = 16_129  // mm² per in² (numerator)
   AreaBridgeSIDenominator int64 = 25      // mm² per in² (denominator)
   AreaBridgeUSNumerator   int64 = 25      // in² per mm² (numerator)
   AreaBridgeUSDenominator int64 = 16_129  // in² per mm² (denominator)
   ```

6. Add `FamilyArea` branches to `ConvertUSToSI` and `ConvertSIToUS` using `scaleMulDiv` from Task 0.

7. Add display defaults to `SIDefaultDecimalPlaces`:
   ```
   "mm2": 0,
   "cm2": 0,
   "m2":  2,
   "km2": 2,
   ```

8. Add constructor names to `UnitConstructorNames`:
   ```
   "squaremillimetres": "mm2",  "squaremillimeters": "mm2",
   "squarecentimetres": "cm2",  "squarecentimeters": "cm2",
   "squaremetres":      "m2",   "squaremeters":      "m2",
   "squarekilometres":  "km2",  "squarekilometers":  "km2",
   "squareinches":      "in2",
   "squarefeet":        "ft2",
   "squareyards":       "yd2",
   "acres":             "ac",
   "squaremiles":       "mi2",
   ```

**Validation:**
- `LookupUnitSuffix("m2")` returns `{Family: "area", System: "SI"}`.
- `SISubUnitsPerUnit("m2")` returns 1,000,000.
- `USSubUnitsPerUnit("ft2")` returns 144.
- `ConvertUSToSI(1, 0, FamilyArea)` returns (645, 0) — 1 in² ≈ 645 mm².
- `ConvertSIToUS(1_000_000, 0, FamilyArea)` returns (1550, 0) — 1 m² ≈ 1,550 in².

---

### Task 2: Lexer — Area Suffix Recognition

**Files:** `pkg/parsley/lexer/lexer.go`
**Estimated effort:** Small

Add area suffixes to `isValidUnitSuffix`. The lexer's `readUnitSuffix` already consumes letters and digits (`isLetter(l.ch) || isDigit(l.ch)`), then does longest-match, so no structural changes needed.

**Changes:**

1. Add case to `isValidUnitSuffix`:
   ```
   // Area — SI
   case "mm2", "cm2", "m2", "km2":
       return true
   // Area — US
   case "in2", "ft2", "yd2", "ac", "mi2":
       return true
   ```

**Disambiguation verification:**

The longest-match strategy in `readUnitSuffix` handles all area vs non-area conflicts:
- `#5m2`: candidate `m2`, tries length 2 first → `m2` matches → area. ✓
- `#5m`: candidate `m`, tries length 1 → `m` matches → length. ✓
- `#5mm2`: candidate `mm2`, tries length 3 → `mm2` matches → area. ✓
- `#5mm`: candidate `mm`, tries length 2 → `mm` matches → length. ✓
- `#5km2`: candidate `km2`, tries length 3 → `km2` matches → area. ✓
- `#5in2`: candidate `in2`, tries length 3 → `in2` matches → area. ✓
- `#5in`: candidate `in`, tries length 2 → `in` matches → length. ✓
- `#5mi2`: candidate `mi2`, tries length 3 → `mi2` matches → area. ✓
- `#5ac`: candidate `ac`, tries length 2 → `ac` matches → area. ✓

**Edge case — `#5m20` (unit literal followed by number `0`):** The suffix reader consumes `m20`, tries length 3 → `m20` invalid, tries length 2 → `m2` valid → match. Lexer rewinds to after `m2`. The `0` becomes a separate token. ✓

**Validation:**
- `#5m2` lexes as unit literal with suffix `m2`.
- `#5m` lexes as unit literal with suffix `m` (length, not area).
- `#3.5ac` lexes as unit literal with suffix `ac`.

---

### Task 3: Parser/Evaluator — Area Literal Evaluation (Scale-Aware)

**Files:** `pkg/parsley/parser/parser.go`, `pkg/parsley/evaluator/evaluator.go`
**Estimated effort:** Medium

Area literals follow the standard SI/US evaluation paths, but the parser must now detect int64 overflow and apply Scale when the computed sub-unit amount exceeds int64 range.

**Changes:**

1. **Update parser's `parseUnitAmount` to detect overflow:**

   When computing `whole * subPerUnit`, check for int64 overflow before multiplying. If overflow would occur:
   - Factor out powers of 10 from subPerUnit: e.g., km2 has subPerUnit = 10¹², so factor = 10¹², reduced = 1.
   - Try computing `whole * reduced`. If that fits, set Scale = log10(factor).
   - If still overflows, factor out powers of 10 from `whole` into Scale as well.

   For existing families (subPerUnit ≤ 10¹², values small enough), this never triggers — the overflow check passes and the fast path runs.

2. **Update parser's duplicate SI sub-unit table** (`parserSISubUnitsPerUnit`) with area entries:
   ```
   "mm2": 1, "cm2": 100, "m2": 1_000_000, "km2": 1_000_000_000_000,
   ```

3. **Update parser's duplicate US sub-unit table** (`parserUSSubUnitsPerUnit`) with area entries:
   ```
   "in2": 1, "ft2": 144, "yd2": 1_296, "ac": 6_272_640, "mi2": 4_014_489_600,
   ```

**Validation:**
- `#100m2` → Amount=100,000,000, Scale=0, Family="area", System="SI", DisplayHint="m2".
- `#1500ft2` → Amount=216,000, Scale=0, Family="area", System="US", DisplayHint="ft2".
- `#510000000km2` → overflow at 510,000,000 × 10¹² = 5.1 × 10²⁰. Parser applies Scale: Amount=510,000,000, Scale=12 (or equivalent normalized form). Represents Earth's surface.
- `#1/2m2` → Amount=500,000, Scale=0 (truncated from 500,000.0).
- `#1/3ac` → Amount=2,090,880, Scale=0 (exact: 6,272,640 / 3).
- `#1/3in2` → Amount=0, Scale=0 (truncated from 0.333...).
- All existing unit literal tests pass unchanged (Scale=0).

---

### Task 4: Area Arithmetic — Standard Paths (Scale-Aware)

**Files:** `pkg/parsley/evaluator/eval_unit_infix.go`
**Estimated effort:** Small

Area uses the standard arithmetic paths (same as length, mass, volume) but now routed through scale-aware helpers from Task 0. No area-specific branching is needed — the Scale helpers handle everything generically.

**Verification (Scale=0 cases — behave identically to existing families):**
- `#1m2 + #5000cm2` → (1,000,000 + 500,000, Scale=0) = `#1.5m2`
- `#1ft2 - #36in2` → (144 - 36, Scale=0) = 108 in² displayed as `#0.75ft2`
- `#1m2 * 3` → `#3m2`
- `#10ac / 5` → `#2ac`
- `#1m2 / #1m2` → `1` (integer ratio)
- `#1m2 + #1kg` → error (UNIT-0001, cross-family)

**Verification (Scale>0 cases):**
- `#510000000km2 + #500000000km2` → (5.1e18, Scale=2) + (5.0e18, Scale=2) = (1.01e19, Scale=2). If overflows at Scale=2, normalize to (1.01e18, Scale=3).
- `#510000000km2 + #100m2` → align scales, add. The 100m² (10⁸ mm²) is negligible at Scale=2 but should not error.
- `#510000000km2 * 2` → (5.1e18 × 2, Scale=2) = (1.02e19, Scale=2). Normalize if overflow.

No code changes beyond what Task 0 already provides — the existing dispatch handles area once the tables and bridges are populated.

---

### Task 5: US Area Display — Decimal Only

**Files:** `pkg/parsley/evaluator/unit_display.go`
**Estimated effort:** Small

US area should always use **decimal display**, not fraction display. The existing `unitUSDisplay` function uses GCD-based fraction formatting, which doesn't make sense for area (no HCN denominator, and area fractions are not idiomatic).

**Changes:**

1. In `unitUSDisplay` (or in a new check at the top), detect `FamilyArea` and route to decimal display instead of fraction display. The simplest approach: check if the family is area and call `unitUSDecimalFallback` directly.

2. Add area suffix entries to the decimal display precision logic if needed (default 2dp for `ac`, `mi2`; 0dp for `in2`, `ft2`, `yd2`).

3. Ensure Scale>0 values display correctly. For US area with Scale>0, use `scaleTrueValue` to compute the float64 for formatting.

**Validation:**
- `#1500ft2` displays as `#1500ft2` (not a fraction).
- `#2.5ac` displays as `#2.5ac`.
- `#0.5mi2` displays as `#0.5mi2`.

---

### Task 6: Constructors — Area (Scale-Aware)

**Files:** `pkg/parsley/evaluator/methods_unit.go`, `pkg/parsley/evaluator/evaluator.go`
**Estimated effort:** Small

Add named constructors for area units. These follow the existing pattern — Task 1 already adds the constructor name→suffix mappings. The `UnitFromConstructor` function uses `LookupUnitSuffix` and routes based on system, so it should work automatically once the tables are populated.

**Constructors to add:**
- `squaremillimetres(v)` / `squaremillimeters(v)` — creates `mm2` unit
- `squarecentimetres(v)` / `squarecentimeters(v)` — creates `cm2` unit
- `squaremetres(v)` / `squaremeters(v)` — creates `m2` unit
- `squarekilometres(v)` / `squarekilometers(v)` — creates `km2` unit
- `squareinches(v)` — creates `in2` unit
- `squarefeet(v)` — creates `ft2` unit
- `squareyards(v)` — creates `yd2` unit
- `acres(v)` — creates `ac` unit
- `squaremiles(v)` — creates `mi2` unit

**Changes:**

1. Register each constructor name in the evaluator's builtin table (same pattern as Phase 1/2 constructors).

2. The generic `unit(value, "m2")` constructor should also work via `GenericUnitConstructor` once the suffix table is populated.

3. **Scale-aware construction:** When constructing from a large integer or float value, the multiplication `value × subPerUnit` may overflow. The constructor must detect this and apply Scale, same as the parser (Task 3). Factor this logic into a shared helper used by both parser and constructors.

**Validation:**
- `squaremetres(100)` → `#100m2` (Scale=0)
- `acres(#1mi2)` → converts 1 mi² to acres → `#640ac`
- `squarefeet(#1m2)` → converts via bridge → `~#10.76ft2` (cross-system)
- `unit(500, "cm2")` → `#500cm2`
- `squarekilometres(510000000)` → Scale>0, represents Earth's surface

---

### Task 7: Properties and Methods — Area Extensions

**Files:** `pkg/parsley/evaluator/methods_unit.go`
**Estimated effort:** Small

All existing unit properties and methods should work for area with no code changes, since they dispatch based on system (SI/US) and family. Verify:

- `.value` — returns decoded float in display-hint units. Uses `unitDisplayValue`, which reads from `SISubUnitsPerUnit` / `USSubUnitsPerUnit`. Must account for Scale when Scale>0 (use `scaleTrueValue` divided by subPerUnit). ✓
- `.unit` — returns `"m2"`, `"ac"`, etc. ✓
- `.family` — returns `"area"`. ✓
- `.system` — returns `"SI"` or `"US"`. ✓
- `.to("ft2")` — conversion. Uses bridge functions which are now scale-aware (Task 0). ✓
- `.abs()` — absolute value. Scale unchanged. ✓
- `.format()` — formatting. ✓
- `.repr()` — parseable literal. Must produce a string that re-parses to the same (Amount, Scale). ✓
- `.toDict()` — dictionary. ✓
- `.inspect()` — debug output. ✓
- `.toFraction()` — for US area, returns a decimal string since there's no HCN fraction. Verify no error.

**Changes:**

1. Update the `exampleForFamily` function to return an area example for `FamilyArea` (used in error messages when wrong-family values are passed to constructors).

2. Update the introspection `TypeProperties` description to include "area" in the family list.

3. Ensure `.value` handles Scale>0 correctly (returns float64 computed from `Amount × 10^Scale / subPerUnit`).

**Validation:**
- `#100m2.value` → `100.0`
- `#100m2.unit` → `"m2"`
- `#100m2.family` → `"area"`
- `#100m2.to("ft2")` → converts via bridge
- `#640ac.to("mi2")` → `#1mi2`
- `#510000000km2.value` → `510000000.0` (large but representable as float64)

---

### Task 8: PLN Parser Fix — Digit-Containing Suffixes

**Files:** `pkg/parsley/pln/parser.go`
**Estimated effort:** Medium

**This is one of the most important tasks in Phase 3.** The PLN parser's `parseUnit` function extracts the suffix by scanning backwards from the end, stopping at the first digit character. This breaks for area suffixes like `m2`, `in2`, `km2` where the suffix itself ends with `2`.

**Current code (broken for area):**
```
suffixStart := len(body)
for suffixStart > 0 {
    ch := body[suffixStart-1]
    if ch >= '0' && ch <= '9' || ch == '.' || ch == '/' || ch == '+' || ch == '-' {
        break
    }
    suffixStart--
}
```

**Fix — Forward longest-match using the suffix table:**

Replace the backward scan with a forward approach that tries all valid split points and picks the one yielding the longest recognized suffix:

```
// Find the split between numeric part and suffix.
// Try all possible split points; pick the longest valid suffix.
numStr := ""
suffix := ""
for i := len(body); i > 0; i-- {
    candidate := body[i:]
    if _, ok := evaluator.LookupUnitSuffix(candidate); ok {
        if len(candidate) > len(suffix) {
            numStr = body[:i]
            suffix = candidate
        }
        break // longest-first, so first match is longest
    }
}
```

This iterates from the end of the body backwards, trying progressively longer suffix candidates. The first match found is the longest possible suffix (since we start with the full remaining string and shrink the numeric prefix).

**PLN parser must also handle Scale on re-parse:** When parsing `#510000000km2`, the numeric part is `510000000` and the suffix is `km2`. The multiplication `510000000 × 10¹²` overflows int64. The PLN parser must use the same overflow-detecting logic from Task 3 to apply Scale.

**Edge cases to verify:**
- `#5m2` → numStr `5`, suffix `m2`. ✓
- `#5m` → numStr `5`, suffix `m`. ✓
- `#3/8in2` → numStr `3/8`, suffix `in2`. ✓
- `#100C` → numStr `100`, suffix `C`. ✓
- `#92+5/8in` → numStr `92+5/8`, suffix `in`. ✓
- `#-273.15C` → numStr `-273.15`, suffix `C`. ✓
- `#5mm2` → numStr `5`, suffix `mm2`. ✓ (longest match)
- `#1.5km2` → numStr `1.5`, suffix `km2`. ✓
- `#510000000km2` → numStr `510000000`, suffix `km2`, Scale applied. ✓

**Validation:**
- PLN round-trip: serialize `#100m2`, parse back, compare. Must be identical.
- PLN round-trip: serialize `#2.5ac`, parse back, compare.
- PLN round-trip: serialize `#1500ft2`, parse back, compare.
- PLN round-trip: serialize `#510000000km2` (scaled), parse back, compare.
- All existing PLN unit tests must still pass (no regressions).

---

### Task 9: Integration Tests

**Files:** `pkg/parsley/tests/unit_test.go`
**Estimated effort:** Large

Add comprehensive tests for area units and Scale infrastructure.

**Test categories:**

#### 9a. Scale infrastructure (unit-level tests)
- `scaleAlign(5, 0, 3, 0)` → `(5, 3, 0)` — fast path
- `scaleAlign(5e18, 2, 3, 0)` → falls back to Scale=2, truncates 3
- `scaleNormalize(11e18, 0)` → overflow → normalized
- `scaleMulDiv(1, 0, 16129, 25)` → `(645, 0)` — bridge at Scale=0
- `scaleMulDiv(5e18, 2, 16129, 25)` → correct result with Scale adjustment
- `scaleCmp(5e18, 2, 9e18, 0)` → +1
- `scaleCmp(5, 0, 5, 0)` → 0

#### 9b. Literal parsing
- `#100m2` → correct Amount, Family, System, DisplayHint, Scale=0
- `#1500ft2` → correct values, Scale=0
- `#2.5ac` → decimal literal, Scale=0
- `#0.5km2` → decimal literal, Scale=0
- `#-100m2` → negative area, Scale=0
- `#3/8m2` → SI fraction (375,000 mm²), Scale=0
- `#1/2ft2` → US area: 1/2 × 144 = 72 in², Scale=0
- `#1/3ac` → 2,090,880 in², Scale=0 (exact)
- `#510000000km2` → Scale>0, Earth's surface
- `#5mm2` → Amount=5, Scale=0 (small area preserved)

#### 9c. Within-system arithmetic
- `#1m2 + #5000cm2 == #1.5m2`
- `#1000ft2 - #500ft2 == #500ft2`
- `#10ac * 3 == #30ac`
- `#1mi2 / 2 == #0.5mi2`
- `#2m2 / #1m2 == 2`

#### 9d. Scale arithmetic
- Large + large within Scale=0 that overflows → Scale>0 result
- Scaled + small Scale=0 → small value absorbed or truncated
- Scaled × scalar → correct Scale adjustment
- Scaled ÷ scalar → correct result

#### 9e. Cross-system arithmetic
- `#1ft2 + #144in2 == #2ft2` (within US, different display)
- `#1m2 + #1ft2` → result in m2 (left side wins, cross-system bridge)
- `#1ft2 + #1m2` → result in ft2 (left side wins)

#### 9f. Cross-system comparison
- `#1296in2 == #1yd2` → true (within US)
- `#144in2 == #1ft2` → true (within US)
- `#640ac == #1mi2` → true (within US)

#### 9g. Constructors
- `squaremetres(100)` → `#100m2`
- `acres(640)` → `#640ac`
- `squarefeet(#1yd2)` → `#9ft2`
- `unit(100, "m2")` → `#100m2`
- `squaremetres(#5kg)` → error (cross-family)
- `squarekilometres(510000000)` → Scale>0

#### 9h. Properties
- `#100m2.value == 100`
- `#100m2.unit == "m2"`
- `#100m2.family == "area"`
- `#100m2.system == "SI"`
- `#640ac.system == "US"`
- `#5mm2.value == 5` (small area)

#### 9i. Methods
- `#1mi2.to("ac")` → `#640ac`
- `#1yd2.to("ft2")` → `#9ft2`
- `#100m2.abs() == #100m2`
- `#-100m2.abs() == #100m2`

#### 9j. PLN round-trip
- `#100m2` → serialize → parse → `#100m2`
- `#2.5ac` → serialize → parse → `#2.5ac`
- `#1500ft2` → serialize → parse → `#1500ft2`
- `#0.5km2` → serialize → parse → `#0.5km2`
- `#510000000km2` → serialize → parse → `#510000000km2` (scaled round-trip)
- `#5mm2` → serialize → parse → `#5mm2`

#### 9k. Display
- `#1500ft2` displays as `#1500ft2` (decimal, not fraction)
- `#100m2` displays as `#100m2`
- `#2.5ac` displays as `#2.5ac`
- `#5mm2` displays as `#5mm2`

#### 9l. Errors
- `#5m2 + #5kg` → cross-family error
- `#5m2 * #5m2` → unit × unit error

#### 9m. Existing family regression
- Verify a representative sample of length, mass, data, volume, temperature operations still produce identical results (Scale=0 throughout).

---

## Phase 3b: Volume Scale Extension

With the Scale infrastructure from Task 0 in place, extending Volume to support larger values is a small incremental addition. The primary change is adding the `kL` (kilolitre) suffix. 1 kL = 1,000 L = 1 m³ — a natural and widely-used unit for pools, tanks, reservoirs, and industrial quantities.

**Current Volume limits (Scale=0):**
- SI: max 9.22 billion L (~9.22 million kL / m³). Can hold ~3,700 Olympic swimming pools.
- US: max ~99.3 billion gal. Adequate for most purposes.

**With kL suffix and Scale:**
- `kL` = 10¹² nL. At Scale=0, max = ~9,220 kL. Beyond that, Scale activates.
- A large reservoir (e.g., Sydney Harbour ≈ 500 million kL) would need Scale>0 — handled automatically.
- Existing `mL` and `L` values are completely unaffected (Scale=0 always).

**Why now:** The Scale infrastructure is being built for area. Volume benefits from it for free. Adding `kL` is the natural companion — it's the volume equivalent of `km2` (the suffix that makes Scale essential). Shipping both together means the educational story is complete: "Parsley handles everything from millilitres to reservoirs, from square millimetres to Jupiter's surface."

### Task 11: Volume Extension — kL Suffix and Constructor

**Files:** `pkg/parsley/evaluator/unit_tables.go`, `pkg/parsley/evaluator/evaluator.go`, `pkg/parsley/lexer/lexer.go`, `pkg/parsley/parser/parser.go`
**Estimated effort:** Small

Add the `kL` (kilolitre) suffix to the SI Volume family. This follows the exact same pattern as existing mL/L entries.

**Changes:**

1. Add `kL` to `unitSuffixTable`:
   ```
   "kL": {Suffix: "kL", Family: FamilyVolume, System: SystemSI},
   ```

2. Add `kL` to `siSubUnitsPerUnit`:
   ```
   "kL": 1_000_000_000_000,  // 1 kL = 10¹² nL
   ```

3. Add `kL` to `parserSISubUnitsPerUnit`:
   ```
   "kL": 1_000_000_000_000,
   ```

4. Add `kL` to lexer's `isValidUnitSuffix`:
   ```
   case "mL", "L", "kL":
       return true
   ```

5. Add display default for `kL` to `SIDefaultDecimalPlaces`:
   ```
   case "kL":
       return 2
   ```

6. Add constructors:
   - `kilolitres(v)` / `kiloliters(v)` → creates `kL` unit

7. Register constructor names in `UnitConstructorNames`:
   ```
   "kilolitres": "kL",
   "kiloliters": "kL",
   ```

8. Register constructors in the evaluator's builtin table:
   ```
   "kilolitres": {Fn: func(args ...Object) Object { return unitNamedConstructor("kilolitres", args) }},
   "kiloliters": {Fn: func(args ...Object) Object { return unitNamedConstructor("kiloliters", args) }},
   ```

**Validation:**
- `LookupUnitSuffix("kL")` returns `{Family: "volume", System: "SI"}`.
- `SISubUnitsPerUnit("kL")` returns 1,000,000,000,000.
- `#5kL` lexes and parses correctly (Amount=5,000,000,000,000, Scale=0).
- `#1kL == #1000L` → true.
- `kilolitres(1)` → `#1kL`.
- Existing mL and L tests unaffected.

---

### Task 12: Volume Scale — Large Value Handling

**Files:** No new files — leverages Task 0 infrastructure and Task 3 parser overflow detection.
**Estimated effort:** Small

Verify that the generic Scale overflow detection from Tasks 0 and 3 handles large volume values correctly. The parser's overflow detection is family-agnostic, so `#500000000kL` (Sydney Harbour) should automatically apply Scale.

**Verification (no code changes expected):**

1. **Parser overflow for large kL values:**
   - `#500000000kL` → 500,000,000 × 10¹² = 5 × 10²⁰ nL. Overflows int64.
   - Parser detects overflow, applies Scale: Amount=500,000,000, Scale=12 (or equivalent).

2. **Arithmetic with scaled volumes:**
   - `#500000000kL + #1000kL` → mixed-scale addition works via `scaleAlign`.
   - `#500000000kL * 2` → scaled multiplication works via `scaleNormalize`.

3. **Cross-system conversion of large volumes:**
   - `gallons(#500000000kL)` → bridge conversion handles Scale via `scaleMulDiv`.

4. **Display of scaled volumes:**
   - `#500000000kL` displays correctly using `scaleTrueValue` for the float64 computation.

5. **PLN round-trip:**
   - `#500000000kL` → serialize → parse → identical result.

6. **Existing volume values unaffected:**
   - All existing mL, L, floz, cup, pt, qt, gal tests still pass with Scale=0.

**Note:** If any volume-specific code path doesn't handle Scale generically (e.g., hard-coded assumptions about nL range), fix those paths. But Task 0 should have already made all paths generic.

---

### Task 13: Volume Scale Tests

**Files:** `pkg/parsley/tests/unit_test.go`
**Estimated effort:** Small

Add tests specifically for the kL suffix and volume Scale behaviour.

**Test categories:**

#### 13a. kL literal parsing
- `#1kL` → Amount=1,000,000,000,000, Scale=0, Family="volume", System="SI"
- `#5.5kL` → correct decimal handling
- `#1kL == #1000L` → true
- `#1kL == #1000000mL` → true

#### 13b. kL constructors
- `kilolitres(1)` → `#1kL`
- `kiloliters(1)` → `#1kL`
- `kilolitres(#1000L)` → `#1kL`
- `kilolitres(#1gal)` → correct cross-system conversion

#### 13c. kL arithmetic
- `#1kL + #500L` → `#1.5kL`
- `#10kL - #3kL` → `#7kL`
- `#2kL * 5` → `#10kL`
- `#10kL / 2` → `#5kL`

#### 13d. Large volume (Scale>0)
- `#500000000kL` → Scale>0, represents a large reservoir
- `#500000000kL + #1000kL` → correct Scale arithmetic
- PLN round-trip of `#500000000kL`

#### 13e. kL properties and methods
- `#5kL.value == 5`
- `#5kL.unit == "kL"`
- `#5kL.family == "volume"`
- `#5kL.to("L")` → `#5000L`
- `#1000L.to("kL")` → `#1kL`

#### 13f. Volume regression
- All existing mL, L, floz, cup, pt, qt, gal tests still pass unchanged.

---

## Phase 3c: Unit Limits — `.max` and `.min` Properties

### Task 14: Unit `.max` and `.min` Properties

**Files:** `pkg/parsley/evaluator/methods_unit.go`
**Estimated effort:** Medium

Add `.max` and `.min` read-only properties to all unit values, across all families. These let users discover the representable range for any unit suffix — valuable for education, validation, and defensive programming.

**API Design:**

Properties (not methods — no parens, consistent with `.value`, `.unit`, `.family`, `.system`):

```
#0m.max     // → maximum representable value in metres at full (Scale=0) precision
#0m.min     // → smallest representable positive value in metres (1 base sub-unit)
#0km.max    // → maximum in km (same underlying limit, different display)
#0ft2.max   // → maximum in ft2
#0C.max     // → maximum representable Celsius
#0C.min     // → minimum representable Celsius (approaches absolute zero)
```

Both return a Unit value with the same suffix as the receiver. The user can then inspect `.value` to get the numeric limit.

**Semantics:**

1. **`.max`** — The maximum positive value representable at full integer precision (Scale=0). This is `int64 max / subPerUnit` expressed in the receiver's suffix.
   - `#0m.max` → `int64_max / 1,000,000 µm per m` = 9,223,372,036,854 µm = `#9223372036.854m` (or however display rounds it)
   - `#0km.max` → `int64_max / 1,000,000,000 µm per km` = `#9223372036.854775807km` → displayed per km rules
   - `#0mm2.max` → `int64_max / 1 mm² per mm²` = `#9223372036854775807mm2`
   - `#0km2.max` → `int64_max / 10¹² mm² per km²` = `#9223372km2` (~9.2 million km²)
   - For US area: `#0mi2.max` → `int64_max / 4,014,489,600 in² per mi²` ≈ `#2297mi2` ... no, `9.22e18 / 4.01e9` ≈ `#2297000000mi2` (~2.3 billion mi²)

2. **`.min`** — The smallest representable positive value. This is `1 base sub-unit / subPerUnit` expressed in the receiver's suffix.
   - `#0m.min` → `1 µm / 1,000,000 µm per m` = `#0.000001m`
   - `#0km.min` → `1 µm / 1,000,000,000 µm per km` = `#0.000000001km`
   - `#0mm2.min` → `1 mm² / 1` = `#1mm2`
   - `#0in.min` → `1 sub-yard / 20,160 sub-yards per in` — display as smallest representable inch

3. **Temperature special case:**
   - `.max` for C, F, K: derived from the int64 max of the sub-kelvin representation.
   - `.min` for K: `#0K` (absolute zero) or the smallest positive Kelvin. Design choice: use smallest positive sub-kelvin increment.
   - `.min` for C: the minimum representable Celsius (from sub-kelvin floor), which is close to but not exactly -273.15°C depending on precision.
   - `.min` for F: similarly derived.

**Note on Scale:** `.max` reports the Scale=0 ceiling — the boundary where all arithmetic is fully exact. Values beyond this are representable via Scale but with reduced precision at extreme magnitudes. This is the educationally meaningful limit: "up to here, everything is exact."

**Changes:**

1. Add `"max"` and `"min"` cases to `evalUnitProperty` in `methods_unit.go`.

2. For non-temperature families:
   ```
   case "max":
       subPerUnit := subUnitsForUnit(unit)  // SI or US, by suffix
       maxAmount := math.MaxInt64 / subPerUnit * subPerUnit  // largest exact multiple
       return &Unit{Amount: maxAmount, Family: unit.Family, System: unit.System, DisplayHint: unit.DisplayHint}
   case "min":
       return &Unit{Amount: 1, Family: unit.Family, System: unit.System, DisplayHint: unit.DisplayHint}
   ```
   Where `subUnitsForUnit` dispatches to `SISubUnitsPerUnit` or `USSubUnitsPerUnit` based on system.

3. For temperature: compute from sub-kelvin limits, converting to the receiver's display hint.

4. Update `TypeProperties["unit"]` to include `max` and `min` in the introspection list.

5. Update the `unknownMethodError` property list to include `"max"`, `"min"`.

**Validation:**
- `#0m.max.value` → approximately 9,223,372,036.85 (varies by display precision)
- `#0m.min.value` → 0.000001 (1 µm in metres)
- `#0km2.max.value` → approximately 9,223,372 (the ~9.2M km² Scale=0 ceiling)
- `#0mm2.min.value` → 1.0
- `#0ft.max.value` → positive, large
- `#0C.min.value` → close to -273.15
- `#0K.min.value` → small positive (smallest sub-kelvin increment)
- `.max` and `.min` return Unit values (not floats), so they can be used in arithmetic: `if myArea > #0km2.max { ... }`
- All existing property tests unaffected (`.max` / `.min` are new keys).

---

### Task 10: Documentation

**Files:** `docs/parsley/reference.md`, `docs/parsley/CHEATSHEET.md`
**Estimated effort:** Small

Update documentation to cover area units, the kL volume suffix, and unit `.max`/`.min` properties.

**Changes:**

1. **reference.md:**
   - Add area to the unit families table (§1.16 or equivalent section).
   - Add area suffixes: `mm2`, `cm2`, `m2`, `km2`, `in2`, `ft2`, `yd2`, `ac`, `mi2`.
   - Add `kL` to the volume suffixes.
   - Add area constructors and `kilolitres`/`kiloliters` to the constructor table.
   - Note area-specific display rules (decimal only for US, no fractions).
   - Document Scale behaviour: values up to ~9.2 million km² (SI) / ~2.3 billion mi² (US) / ~9.22 billion L are exact at int64 precision. Beyond that, Scale provides extended range with graceful precision at extreme magnitudes.
   - Note that `#1/3ac` and `#1/3mi2` produce exact results, while `#1/3in2` truncates to 0.
   - Document `.max` and `.min` properties: what they return, semantics (Scale=0 exact ceiling for `.max`, smallest positive for `.min`), temperature special cases.

2. **CHEATSHEET.md:**
   - Add area suffixes to the quick-reference table.
   - Add `kL` to the volume quick-reference.
   - Note that US area uses decimal display (not fractions like other US units).
   - Note that area suffixes contain digits (`m2` not `m²`).
   - Note that area and large volumes support very large values (Earth's surface, reservoirs, etc.) via internal scaling.
   - Add `.max` / `.min` to the unit properties quick-reference.

---

## Task Dependencies

```
                         ┌─── Phase 3a: Area ────────────────────────────────────┐
                         │                                                       │
Task 0 (scale infra) ──→ Task 1 (tables) ──┬──→ Task 2 (lexer) ──→ Task 3 (parser/eval)
         │                                  │                              │
         │                                  │                              ▼
         │                                  ├──→ Task 4 (arithmetic) ──────┤
         │                                  │                              │
         │                                  ├──→ Task 5 (display) ─────────┤
         │                                  │                              │
         │                                  ├──→ Task 6 (constructors) ────┤
         │                                  │                              │
         │                                  ├──→ Task 7 (methods) ─────────┤
         │                                  │                              │
         │                                  └──→ Task 8 (PLN fix) ─────────┤
         │                                                                 │
         │               ┌─── Phase 3b: Volume Scale ──────┐               │
         │               │                                 │               │
         ├──→ Task 11 (kL suffix) ──→ Task 12 (verify) ──→ Task 13 (tests)
         │                                                                 │
         │               ┌─── Phase 3c: Unit Limits ───┐                   │
         │               │                             │                   │
         └──→ Task 14 (.max/.min properties) ──────────────────────────────┤
                                                                           │
                                                                           ▼
                                                                  Task 9 (all tests)
                                                                           │
                                                                           ▼
                                                                  Task 10 (docs)
```

**Task 0 comes first.** It modifies the Unit struct and all arithmetic/display/bridge code. It must be fully working and passing all existing tests before any area-specific work begins. This is the riskiest task — it touches the most code and must maintain perfect backward compatibility.

**Phase 3a (Area):** Task 1 is the area foundation. Tasks 2–8 can mostly be done in parallel after Task 1, though Tasks 3–7 are light verification once Task 1 is complete. Task 8 (PLN fix) is independent of Tasks 2–7.

**Phase 3b (Volume Scale):** Tasks 11–13 depend only on Task 0 (not on area tasks). They can be done in parallel with Phase 3a or sequentially after — whichever is more convenient.

**Phase 3c (Unit Limits):** Task 14 depends only on Task 0 (it needs Scale awareness for area/volume max reporting). It can be done in parallel with Phases 3a and 3b.

Task 9 (integration tests) and Task 10 (docs) come last and cover all three phases.

**Recommended order:** 0 → 1 → 8 → 11 → 2 → 3 → 12 → 5 → 6 → 4 → 7 → 14 → 13 → 9 → 10.

Task 0 is implemented first and validated against the full existing test suite before proceeding. Task 8 is prioritized early because it's the riskiest area-specific change (modifying existing parsing logic). Task 11 (kL suffix) slots in early since it's small and independent of area tasks. Task 14 (.max/.min) is placed after the area and volume tasks so it can be validated across all families.

## Validation Checklist

- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run` (no new issues)
- [ ] All Phase 3 acceptance criteria in FEAT-118 checked off
- [ ] Phase 1 regression: all existing unit tests still pass
- [ ] Phase 2 regression: all temperature and volume tests still pass
- [ ] Money regression: all existing money tests still pass
- [ ] Scale infrastructure: all Scale=0 paths produce identical results to pre-Scale code
- [ ] Scale infrastructure: Scale>0 arithmetic produces correct results
- [ ] kL suffix: `#1kL == #1000L` passes
- [ ] kL constructors: `kilolitres(1)` produces `#1kL`
- [ ] Large volume: `#500000000kL` representable via Scale
- [ ] Volume regression: all existing mL, L, floz, cup, pt, qt, gal tests pass unchanged
- [ ] `.max` property returns correct Scale=0 ceiling for all families
- [ ] `.min` property returns correct smallest positive value for all families
- [ ] `.max` and `.min` return Unit values usable in arithmetic and comparisons
- [ ] Temperature `.min` handles absolute zero correctly
- [ ] Area arithmetic verified: same-system exact, cross-system with documented rounding
- [ ] US area display: decimal only (no fraction formatting)
- [ ] PLN round-trip for all area suffixes including digit-containing ones (`m2`, `in2`, etc.)
- [ ] PLN round-trip for scaled values (`#510000000km2`)
- [ ] Existing PLN round-trips still work (no regressions from Task 8 parser fix)
- [ ] Small area values (`#5mm2`) preserve full precision
- [ ] Large area values (`#510000000km2`) representable via Scale
- [ ] Cross-system conversion of scaled values works correctly
- [ ] Documentation updated (reference.md, CHEATSHEET.md)
- [ ] `work/BACKLOG.md` updated with any new deferrals

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-02-14 | Task 0: Scale Infrastructure | ✅ Done | Unit.Scale field + unit_scale.go helpers (scaleAlign, scaleNormalize, scaleAdd/Sub, scaleMul/DivScalar, scaleMulDiv, scaleCmp, scaleRatio, scaleAmountForSuffix) |
| 2026-02-14 | Task 1: Unit Tables — Area | ✅ Done | SI area (mm2/cm2/m2/km2) and US area (in2/ft2/yd2/ac/mi2) tables, bridge constants |
| 2026-02-14 | Task 2: Lexer — Area Suffixes | ✅ Done | Added mm2, cm2, m2, km2, in2, ft2, yd2, ac, mi2, kL to isValidUnitSuffix |
| 2026-02-14 | Task 3: Parser/Evaluator — Scale-Aware | ✅ Done | Parser and PLN parser produce Scale from literals when overflow occurs |
| 2026-02-14 | Task 4: Area Arithmetic | ✅ Done | Scale-aware add/sub/mul/div in eval_unit_infix.go |
| 2026-02-14 | Task 5: US Area Display | ✅ Done | Decimal-only display for US area (no HCN fractions) |
| 2026-02-14 | Task 6: Constructors — Area | ✅ Done | squaremillimetres through squaremiles, scale-aware UnitFromConstructor/GenericUnitConstructor |
| 2026-02-14 | Task 7: Properties — Area | ✅ Done | .value, .unit, .family, .system all work for area units |
| 2026-02-14 | Task 8: PLN Parser Fix | ✅ Done | Longest-match suffix lookup for digit-containing suffixes (m2, km2, kL) |
| 2026-02-14 | Task 9: Integration Tests | ✅ Done | unit_phase3_test.go — scale infra, literals, arithmetic, cross-system, constructors, properties, methods, PLN round-trips, display, errors, regression |
| 2026-02-14 | Task 11: kL Suffix + Constructor | ✅ Done | kL added to SI volume table, kilolitres() constructor |
| 2026-02-14 | Task 14: .max and .min Properties | ✅ Done | Temperature-aware min/max, scale-aware for area |
| 2026-02-14 | Lint cleanup | ✅ Done | Removed unused scaleAlign wrapper, fixed named returns, paramTypeCombine, emptyStringTest, ifElseChain, offBy1 |
| 2026-02-14 | Test fixes | ✅ Done | Fixed test expectations: unquoted string properties, US area decimal display, fraction display for US length, error message substring, scientific notation |
| 2026-02-14 | Task 10: Documentation | ⬜ Todo | Update reference.md, CHEATSHEET.md with area, kL, Scale, .max/.min |

## Deferred Items

Items to add to `work/BACKLOG.md` after implementation:
- Unicode superscript display option: `.format({unicode: true})` → `100 m²` instead of `100m2`
- Hectare (`ha`) suffix: 1 ha = 10,000 m². Common in many countries but not in the design doc. Add on demand.
- `ML` (megalitre) suffix: 1 ML = 1,000 kL = 10⁶ L. Used in water industry. Add on demand.
- Scale trailing-zero normalization: optional optimization to keep Amount minimal and Scale maximal for cleaner PLN output. Not required for correctness.