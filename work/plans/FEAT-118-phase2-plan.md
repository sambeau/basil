---
id: PLAN-096
feature: FEAT-118
title: "Implementation Plan for Measurement Units — Phase 2 (Temperature + Volume)"
status: complete
created: 2026-02-14
---

# Implementation Plan: FEAT-118 Phase 2 — Temperature + Volume

## Overview

Add two new unit families to the existing measurement unit infrastructure: **Temperature** (K, C, F) and **Volume** (mL, L, floz, cup, pt, qt, gal). Temperature introduces a new storage model (K × 900 sub-kelvins) with special arithmetic restrictions (no multiply/divide). Volume follows the established SI/US dual-system pattern from Phase 1.

Phase 1 delivered the core infrastructure (lexer, parser, AST, evaluator, PLN, display, methods, errors) for Length, Mass, and Data. Phase 2 extends this infrastructure — most changes are additive (new entries in tables, new branches in existing switch statements), with one significant new piece: temperature's offset-based arithmetic.

**Design doc:** `work/design/DESIGN-units-v3.md` (authoritative — see §3.3, §4.3, §4.5, §5.2, §9)

## Prerequisites

- [x] Phase 1 complete and merged (PLAN-095)
- [x] Design doc `DESIGN-units-v3.md` §3.3 (Temperature) and §4.5 (Volume) reviewed
- [x] Phase 1 PR merged to main (branch `feat/FEAT-118-measurement-units`)

## Architecture Notes

### Temperature Storage: K × 900

Temperature uses a **unified representation** — all three scales (K, C, F) store as a single int64 counting sub-kelvins, where 1 K = 900 sub-K. This makes the 9/5 Celsius-to-Fahrenheit ratio exact:

- 1°C = 900 sub-K
- 1°F = 500 sub-K
- All conversions are integer arithmetic with no rounding

**Conversion constants (exact integers):**

| Constant | Value | Derivation |
|----------|-------|------------|
| `TempOffsetC` | 245,835 | 273.15 × 900 — sub-K value at 0°C |
| `TempOffsetF` | 229,835 | 459.67 × 500 — sub-K value at 0°F |
| `TempScaleC` | 900 | sub-K per degree Celsius |
| `TempScaleF` | 500 | sub-K per degree Fahrenheit |
| `TempScaleK` | 900 | sub-K per kelvin |

**Encoding:** `#20C` → `(20 × 900) + 245,835 = 263,835 sub-K`
**Decoding:** `263,835 sub-K` → `(263,835 − 245,835) / 900 = 20°C`
**Cross-scale:** `#0C` = 245,835 sub-K; `#32F` = `(32 × 500) + 229,835 = 245,835 sub-K` → equal ✓

### Temperature Arithmetic: Offset Trick

Temperature addition/subtraction use "treat like numbers" semantics: `#20C + #10C = #30C`. This is NOT plain addition of sub-kelvins (that would double-count the offset). The formula for same-scale addition:

```
result_subK = left_subK + right_subK − offset(suffix)
```

Subtraction:

```
result_subK = left_subK − right_subK + offset(suffix)
```

Where `offset(K) = 0`, `offset(C) = 245,835`, `offset(F) = 229,835`.

**Verification:** `#20C + #10C`: `263,835 + 254,835 − 245,835 = 272,835` → `(272,835 − 245,835) / 900 = 30°C` ✓

For cross-scale addition (e.g., `#20C + #10F`), left side wins: convert the right operand's numeric value to the left's scale, then apply the same-scale formula.

### Temperature Multiply/Divide: Forbidden

Celsius and Fahrenheit are offset scales. `#20C == #68F`, but `#20C × 2 = #40C` while `#68F × 2 = #136F`, and `#40C ≠ #136F`. There is no algebraically consistent definition. Scalar multiplication, scalar division, and temperature-over-temperature ratio are all errors. The error message (UNIT-0011, already registered) explains *why*.

### Volume Storage: Standard Dual-System

Volume follows the same pattern as Length and Mass:

- **SI Volume:** sub-unit = microlitre (µL). 1 mL = 1,000 µL, 1 L = 1,000,000 µL.
- **US Volume:** sub-unit = sub-floz over HCN = 725,760. 1 floz = HCN. Fractions like `#1/3cup` are exact.

**Within-system ratios (US):** cup = 8 floz, pt = 16 floz, qt = 32 floz, gal = 128 floz.

**Cross-system bridge:** 1 gal = 3.785411784 L (exact, derived from 1 gal = 231 in³).

In sub-units: 1 gal = 128 × HCN = 92,897,280 sub-floz = 3,785,411,784 µL.

Bridge ratio: `3,785,411,784 µL / 92,897,280 sub-floz`. Simplify via GCD during implementation to get the integer numerator/denominator pair for the bridge constants.

### Integration with Existing `Unit` Struct

Both families use the existing `Unit` struct (`Amount int64`, `Family string`, `System string`, `DisplayHint string`). Temperature values use `Family: "temperature"` and `System: "SI"` for K/C, `"US"` for F. The arithmetic code branches on family to apply temperature-specific logic.

## Tasks

### Task 1: Unit Tables — Temperature Constants and Suffixes

**Files:** `pkg/parsley/evaluator/unit_tables.go`
**Estimated effort:** Small

Add temperature family, suffixes, conversion constants, and display configuration.

Steps:
1. Add `FamilyTemperature = "temperature"` and `FamilyVolume = "volume"` to family constants
2. Add temperature suffixes to `unitSuffixTable`: `K` (SI), `C` (SI), `F` (US)
3. Add temperature conversion constants:
   - `TempOffsetC int64 = 245_835` (273.15 × 900)
   - `TempOffsetF int64 = 229_835` (459.67 × 500)
   - `TempScaleC int64 = 900` (sub-K per °C)
   - `TempScaleF int64 = 500` (sub-K per °F)
   - `TempScaleK int64 = 900` (sub-K per K)
4. Add `TempOffset(suffix) int64` helper that returns the offset for a temperature suffix
5. Add `TempScale(suffix) int64` helper that returns the scale for a temperature suffix
6. Add temperature encoding/decoding helpers:
   - `EncodeTempToSubK(value float64, suffix string) int64` — for literal parsing
   - `DecodeTempFromSubK(subK int64, suffix string) float64` — for `.value` property
7. Add temperature display defaults to `SIDefaultDecimalPlaces`: C → 1, F → 1, K → 2
8. Add temperature constructor names to `UnitConstructorNames`: `celsius` → `C`, `fahrenheit` → `F`, `kelvins` → `K`

Tests:
- Encoding: `EncodeTempToSubK(0, "C")` = 245,835; `EncodeTempToSubK(100, "C")` = 335,835
- Decoding: `DecodeTempFromSubK(245835, "C")` = 0; `DecodeTempFromSubK(335835, "F")` = 212
- Round-trip: encode → decode returns original value for K, C, F
- `#0C` and `#32F` produce the same `Amount`

---

### Task 2: Unit Tables — Volume Constants and Suffixes

**Files:** `pkg/parsley/evaluator/unit_tables.go`
**Estimated effort:** Small

Add volume suffixes, sub-unit tables, and bridge constants.

Steps:
1. Add volume SI suffixes to `unitSuffixTable`: `mL` (SI), `L` (SI)
2. Add volume US suffixes to `unitSuffixTable`: `floz` (US), `cup` (US), `pt` (US), `qt` (US), `gal` (US)
3. Add SI volume sub-units to `siSubUnitsPerUnit`:
   - `mL`: 1,000 (1 mL = 1,000 µL)
   - `L`: 1,000,000 (1 L = 1,000,000 µL)
4. Add US volume sub-units to `usSubUnitsPerUnit`:
   - `floz`: HCN (725,760 sub-floz per floz)
   - `cup`: HCN × 8 (5,806,080)
   - `pt`: HCN × 16 (11,612,160)
   - `qt`: HCN × 32 (23,224,320)
   - `gal`: HCN × 128 (92,897,280)
5. Add volume cross-system bridge constants:
   - Compute from: 1 gal = 128 × HCN sub-floz = 3,785,411,784 µL
   - Find GCD and store as simplified numerator/denominator pair
   - Add `VolumeBridgeSINumerator`, `VolumeBridgeSIDenominator`, `VolumeBridgeUSNumerator`, `VolumeBridgeUSDenominator`
6. Extend `ConvertUSToSI` and `ConvertSIToUS` with `FamilyVolume` case
7. Add volume constructor names to `UnitConstructorNames`:
   - `millilitres` → `mL`, `milliliters` → `mL`
   - `litres` → `L`, `liters` → `L`
   - `fluidounces` → `floz`
   - `cups` → `cup`
   - `pints` → `pt`
   - `quarts` → `qt`
   - `gallons` → `gal`
8. Add display defaults: `mL` → 0, `L` → 2, `floz` → 0, `cup` → 0, `pt` → 0, `qt` → 0, `gal` → 2

Tests:
- US sub-units: `cup` = 8 × HCN; `gal` = 128 × HCN
- Bridge round-trip: 1 gal → SI → US → matches original (within 1 sub-unit)
- `#1/3cup + #1/3cup + #1/3cup == #1cup` (exact US fractions)

---

### Task 3: Lexer — Temperature and Volume Suffix Recognition

**Files:** `pkg/parsley/lexer/lexer.go`
**Estimated effort:** Small

The lexer's suffix table is already driven by `unitSuffixTable` (or a parallel lookup). Add the new suffixes to the lexer's longest-match table.

Steps:
1. Verify that the lexer suffix matching reads from (or is consistent with) `unitSuffixTable`
2. Add `K`, `C`, `F` to the suffix table — ensure longest-match handles `C` without conflict
3. Add `mL`, `L`, `floz`, `cup`, `pt`, `qt`, `gal` to the suffix table
4. Ensure `L` doesn't conflict with other suffixes (it shouldn't — no suffix starts with `L` in Phase 1 except binary `...iB` patterns)
5. Ensure `cup`, `pt`, `qt` don't conflict with anything (they won't — unique prefixes)
6. Verify that `gal` uses longest-match correctly (no prefix collision with `g` for grams — longest match will prefer `gal` over `g` when the full suffix is `gal`)

Tests:
- `#100C` → UNIT token, literal `#100C` (not `#100` + identifier `C`)
- `#212F` → UNIT token
- `#0K` → UNIT token
- `#37.5C` → UNIT token (decimal temperature)
- `#500mL` → UNIT token
- `#2L` → UNIT token
- `#8floz` → UNIT token
- `#1/3cup` → UNIT token (fraction)
- `#1gal` → UNIT token (not `#1g` + `al`)
- `#5g` → still UNIT token for grams (regression)

---

### Task 4: Parser/Evaluator — Temperature Literal Evaluation

**Files:** `pkg/parsley/evaluator/evaluator.go`
**Estimated effort:** Medium

Extend unit literal evaluation to handle temperature encoding (K × 900 with offsets).

Steps:
1. In the unit literal evaluation path, after parsing the numeric value and suffix, branch on family
2. For `FamilyTemperature`: use `EncodeTempToSubK(numericValue, suffix)` instead of the standard SI/US sub-unit multiplication
3. Temperature fractions (`#1/2C`): encode the fractional value via the temperature formula — note that fractional temperatures are always truncated to integer sub-kelvins (like SI fractions)
4. Temperature mixed numbers (`#98+3/5F`): parse as normal, encode the combined value
5. Set `System: "SI"` for K and C, `System: "US"` for F on the resulting `Unit`

Tests:
- `#0C` → Unit{Amount: 245835, Family: "temperature", System: "SI", DisplayHint: "C"}
- `#100C` → Unit{Amount: 335835, ...}
- `#212F` → Unit{Amount: 335835, ...} (same as #100C)
- `#0K` → Unit{Amount: 0, ...}
- `#-273.15C` → Unit{Amount: 0, ...} (absolute zero)
- `#-40C` and `#-40F` produce same Amount (the scales cross at -40)

---

### Task 5: Parser/Evaluator — Volume Literal Evaluation

**Files:** `pkg/parsley/evaluator/evaluator.go`
**Estimated effort:** Small

Volume uses the standard SI/US paths already built in Phase 1. Verify that the existing code correctly routes volume suffixes through the right sub-unit tables.

Steps:
1. Confirm that volume SI suffixes (`mL`, `L`) route through `siSubUnitsPerUnit` correctly
2. Confirm that volume US suffixes (`floz`, `cup`, `pt`, `qt`, `gal`) route through `usSubUnitsPerUnit`
3. No new code should be needed if Phase 1's literal evaluation is suffix-table-driven — just verify

Tests:
- `#500mL` → Unit with Amount = 500,000 (500 × 1,000 µL/mL)
- `#2L` → Unit with Amount = 2,000,000
- `#8floz` → Unit with Amount = 8 × HCN
- `#1/3cup` → exact US fraction
- `#1gal` → Unit with Amount = 128 × HCN

---

### Task 6: Temperature Arithmetic — Offset-Based Add/Subtract

**Files:** `pkg/parsley/evaluator/eval_unit_infix.go`
**Estimated effort:** Large

This is the core new logic. Temperature addition and subtraction use offset arithmetic instead of plain integer addition.

Steps:
1. In `evalUnitInfixExpression`, after the existing system-normalization step, add a branch for `FamilyTemperature`
2. For same-suffix temperature addition:
   ```
   result = left.Amount + right.Amount − TempOffset(left.DisplayHint)
   ```
3. For same-suffix temperature subtraction:
   ```
   result = left.Amount − right.Amount + TempOffset(left.DisplayHint)
   ```
4. For cross-suffix temperature (left side wins):
   - Decode the right operand's numeric value in its own scale
   - Convert that numeric value to the left operand's scale
   - Encode the converted value as sub-kelvins in the left's scale
   - Apply the same-suffix formula with the converted right amount
5. For temperature `==` and `!=`: comparison works directly on sub-kelvins (no offset needed — same Amount = same physical temperature)
6. For temperature `<`, `>`, `<=`, `>=`: also compare sub-kelvins directly

Key conversion helper for cross-suffix arithmetic:
```
// Convert a temperature's numeric value from one suffix to another
// e.g., 10°F → °C: decode 10°F, re-encode as °C
func convertTempValue(subK int64, fromSuffix, toSuffix string) int64 {
    // Decode from 'from' scale to get numeric value
    // Re-encode in 'to' scale
    // Return the new sub-K value
}
```

Tests:
- `#20C + #10C` = `#30C` (same scale addition)
- `#100C - #37C` = `#63C` (same scale subtraction)
- `#212F - #32F` = `#180F`
- `#0K + #273.15K` = `#273.15K`
- `#20C + #18F` → result in C (left side wins, cross-scale)
- `#0C == #32F` → true
- `#100C == #212F` → true
- `#-40C == #-40F` → true
- `#100C > #200F` → true (#100C = #212F > #200F)
- `#20C < #20F` → false (#20C = #68F > #20F)
- `#0K == #-273.15C` → true (absolute zero)

---

### Task 7: Temperature Arithmetic — Block Multiply/Divide

**Files:** `pkg/parsley/evaluator/eval_unit_infix.go`, `pkg/parsley/evaluator/eval_infix.go`
**Estimated effort:** Small

Reject scalar multiplication, scalar division, and temperature-over-temperature ratio with clear error messages. The error code UNIT-0011 is already registered in the catalog.

Steps:
1. In the unit × scalar and scalar × unit paths, check if the unit's family is temperature — if so, return UNIT-0011 error
2. In the unit / scalar path, check if the unit's family is temperature — if so, return a new error (UNIT-0012: "Cannot divide a temperature") with the same explanatory hint
3. In the unit / unit (ratio) path, check if both units are temperature — if so, return a new error (UNIT-0013: "Cannot divide temperature by temperature") with hint explaining offset-scale ratio is meaningless
4. Add UNIT-0012 and UNIT-0013 to the error catalog in `pkg/parsley/errors/errors.go`

Tests:
- `#20C * 2` → error UNIT-0011, message explains offset scales
- `2 * #20C` → error UNIT-0011
- `#100F / 2` → error UNIT-0012
- `#100C / #50C` → error UNIT-0013
- `-#20C` → ALLOWED (negation is not multiplication — verify this works correctly: negate sub-kelvins, which gives a sub-zero temperature)

---

### Task 8: Volume Arithmetic — Standard Paths

**Files:** `pkg/parsley/evaluator/eval_unit_infix.go`
**Estimated effort:** Small

Volume arithmetic follows the exact same patterns as Length and Mass. Verify that the existing arithmetic code handles volume correctly once the tables are populated.

Steps:
1. Verify same-system volume arithmetic works: `#500mL + #500mL` = `#1000mL`
2. Verify cross-system volume arithmetic uses the bridge: `#1L + #1floz` → result in SI
3. Verify volume × scalar: `#1cup * 3` = `#3cup`
4. Verify volume / scalar: `#1gal / 4` = result in gal
5. Verify volume / volume (ratio): `#1gal / #1qt` = `4`
6. Verify cross-family errors: `#1L + #1m` → error

Tests:
- `#500mL + #500mL` = `#1L` or `#1000mL` (display hint from left)
- `#1/3cup + #1/3cup + #1/3cup` = `#1cup` (exact fraction arithmetic)
- `#3/8cup + #5/8cup` = `#1cup`
- `#1gal / #1qt` = `4`
- `#1cup * 8` = `#8cup`
- `#1L + #1floz` → result in litres (cross-system, left wins)
- `#1gal == #3.785411784L` → true (or very close, verify)
- `#1L + #1kg` → error (different families)

---

### Task 9: Constructors — Temperature

**Files:** `pkg/parsley/evaluator/evaluator.go`
**Estimated effort:** Medium

Add named constructors for temperature. These must handle the temperature encoding and cross-scale conversion.

Steps:
1. Register named constructors in the builtins map:
   - `celsius(value)` — create or convert to Celsius
   - `fahrenheit(value)` — create or convert to Fahrenheit
   - `kelvins(value)` — create or convert to Kelvin
2. When the argument is a number: encode directly using the target suffix
   - `celsius(100)` → `#100C`
   - `fahrenheit(212)` → `#212F`
   - `kelvins(373.15)` → `#373.15K`
3. When the argument is a Unit with family "temperature": convert
   - `celsius(#212F)` → `#100C` (decode sub-K to C)
   - `fahrenheit(#100C)` → `#212F` (decode sub-K to F)
4. When the argument is a Unit with a different family: error (UNIT-0006)
5. When the argument is a non-numeric, non-unit type: error
6. Update the `unitNamedConstructor` helper to handle temperature encoding (it currently only handles SI/US sub-unit multiplication)

Tests:
- `celsius(100)` → `#100C`
- `fahrenheit(212)` → `#212F`
- `kelvins(0)` → `#0K`
- `celsius(#212F)` → `#100C`
- `fahrenheit(#0C)` → `#32F`
- `kelvins(#100C)` → `#373.15K`
- `celsius(#5kg)` → error (wrong family)
- `unit(100, "C")` → `#100C` (generic constructor)
- `unit(#212F, "C")` → `#100C` (generic conversion)

---

### Task 10: Constructors — Volume

**Files:** `pkg/parsley/evaluator/evaluator.go`
**Estimated effort:** Small

Add named constructors for volume. These follow the standard Phase 1 pattern (no special encoding logic needed).

Steps:
1. Register named constructors in the builtins map:
   - `millilitres()` / `milliliters()` → `mL`
   - `litres()` / `liters()` → `L`
   - `fluidounces()` → `floz`
   - `cups()` → `cup`
   - `pints()` → `pt`
   - `quarts()` → `qt`
   - `gallons()` → `gal`
2. Standard numeric → unit creation (multiply by sub-units)
3. Standard unit → unit conversion (same family, cross-system via bridge)
4. Error on wrong family

Tests:
- `litres(2)` → `#2L`
- `gallons(1)` → `#1gal`
- `cups(#1L)` → convert to cups
- `litres(#1gal)` → `#3.79L` (approximate display)
- `millilitres(#1floz)` → result in mL
- `cups(#5m)` → error (wrong family)

---

### Task 11: Properties and Methods — Temperature Extensions

**Files:** `pkg/parsley/evaluator/methods_unit.go`
**Estimated effort:** Medium

Extend existing unit methods to handle temperature correctly.

Steps:
1. `.value` property: for temperature, use `DecodeTempFromSubK(amount, suffix)` instead of the standard float-from-sub-units calculation
2. `.to(suffix)` method: for temperature, re-encode sub-kelvins in the target suffix
   - Verify target suffix is in the temperature family, else error
3. `.abs()` method: negate sub-kelvins if negative — note that negative sub-kelvins mean below absolute zero, which is unphysical but not an error (same as negative length)
4. `.format()` / `.format(precision)`: temperature uses decimal display with defaults from `SIDefaultDecimalPlaces` (C → 1dp, F → 1dp, K → 2dp)
5. `.repr()`: temperature literals display as `#100C`, `#-40F`, `#0K` — no fraction display for temperature (always decimal)
6. `.toDict()`: returns `{value, unit, family, system}` — works with temperature decode
7. `.inspect()`: include sub-kelvins in debug output
8. `.toFraction()`: return error or null for temperature (fractions don't apply)
9. `.system` property: `"SI"` for K/C, `"US"` for F

Tests:
- `#100C.value` → `100`
- `#100C.unit` → `"C"`
- `#100C.family` → `"temperature"`
- `#100C.system` → `"SI"`
- `#212F.system` → `"US"`
- `#100C.to("F")` → `#212F`
- `#32F.to("C")` → `#0C`
- `#100C.to("K")` → `#373.15K`
- `(#-40C).abs()` → `#40C`
- `#37.5C.format()` → `"37.5C"`
- `#37.5C.format(0)` → `"38C"` (rounded)
- `#100C.repr()` → `"#100C"`
- `#-40F.repr()` → `"#-40F"`
- `#100C.toDict()` → `{value: 100, unit: "C", family: "temperature", system: "SI"}`
- `#100C.toFraction()` → error or null (not a US fractional unit)

---

### Task 12: Display and Formatting — Temperature and Volume

**Files:** `pkg/parsley/evaluator/unit_display.go`
**Estimated effort:** Medium

Extend display logic for temperature and volume.

Steps:
1. Temperature display: always decimal, use `DecodeTempFromSubK` to get the numeric value, then format with default decimal places
2. Temperature PLN output: `#100C`, `#-40F`, `#0K` — same as `.repr()`
3. Temperature string interpolation: `100C`, `-40F`, `0K` (no `#` sigil, no space)
4. Volume SI display: decimal with default dp (mL → 0, L → 2) — uses existing SI display path
5. Volume US display: fraction via GCD reduction (uses existing US fraction display path)
6. Volume US fraction examples: `#1/3cup` displays as `#1/3cup`, `#1/2gal` as `#1/2gal`
7. Verify PLN serialization for temperature and volume units
8. Verify PLN deserialization (round-trip) for temperature and volume

Tests:
- `#37.5C` displays as `#37.5C` (PLN) / `37.5C` (interpolation)
- `#100C` displays as `#100C`
- `#-273.15C` displays as `#-273.15C`
- `#212F` displays as `#212F`
- `#98.6F` displays as `#98.6F`
- `#500mL` displays as `#500mL`
- `#2.5L` displays as `#2.5L`
- `#1/3cup` displays as `#1/3cup` (fraction)
- `#1+1/2gal` displays as `#1+1/2gal` (mixed number)
- String interpolation: `` `Temp: {#37.5C}` `` → `"Temp: 37.5C"`
- PLN round-trip: `#100C` → serialize → parse → `#100C`
- PLN round-trip: `#1/3cup` → serialize → parse → `#1/3cup`

---

### Task 13: PLN — Temperature and Volume Lexer/Parser

**Files:** `pkg/parsley/pln/lexer.go`, `pkg/parsley/pln/parser.go`
**Estimated effort:** Small

The PLN lexer and parser already handle unit literals from Phase 1. Extend the suffix tables to recognize the new suffixes.

Steps:
1. Add temperature suffixes (`K`, `C`, `F`) to PLN lexer suffix table
2. Add volume suffixes (`mL`, `L`, `floz`, `cup`, `pt`, `qt`, `gal`) to PLN lexer suffix table
3. Ensure PLN parser routes temperature literals through the temperature encoding path (not the standard SI/US path)
4. Verify round-trip: PLN serialize → PLN parse → equal value

Tests:
- PLN `#100C` → parse → Unit(Amount=335835) → serialize → `#100C`
- PLN `#-40F` → parse → same Amount as `#-40C` → serialize → `#-40F`
- PLN `#500mL` → round-trip
- PLN `#1/3cup` → round-trip (exact fraction preserved)
- PLN `#2.5L` → round-trip

---

### Task 14: Error Messages — New Temperature/Volume Errors

**Files:** `pkg/parsley/errors/errors.go`
**Estimated effort:** Small

Add any new error codes needed for Phase 2. UNIT-0011 (temperature multiply) is already registered.

Steps:
1. Add `UNIT-0012`: "Cannot divide a temperature"
   - Hints: `"temperature scales have arbitrary zero points, so division is undefined — use subtraction instead"`
2. Add `UNIT-0013`: "Cannot divide temperature by temperature"
   - Hints: `"the ratio of two offset-scale temperatures is meaningless — convert to kelvin if you need a ratio"`
3. Verify UNIT-0011 template and hints are still accurate
4. Verify UNIT-0001 (family mismatch) works with new family names ("temperature", "volume")
5. Update UNIT-0007 hint to include temperature and volume suffixes in the abbreviation list

Tests:
- `#20C * 2` → UNIT-0011
- `#20C / 2` → UNIT-0012
- `#100C / #50C` → UNIT-0013
- `#5m + #5C` → UNIT-0001 with families "length" and "temperature"
- `#1L + #1gal` → works (same family, cross-system — not an error)
- `#1L + #1in` → UNIT-0001 with families "volume" and "length"

---

### Task 15: Integration Tests

**Files:** `pkg/parsley/tests/unit_test.go` (extend existing file)
**Estimated effort:** Large

Comprehensive tests for temperature and volume, following the Phase 1 test patterns.

Steps:
1. **Temperature literal tests**: `#0C`, `#100C`, `#212F`, `#0K`, `#-273.15C`, `#-40C`, `#37.5C`, `#98.6F`
2. **Temperature arithmetic tests**: add, subtract (same scale and cross-scale)
3. **Temperature restriction tests**: multiply, divide all produce errors
4. **Temperature comparison tests**: `#0C == #32F`, `#100C == #212F`, `#-40C == #-40F`, `#0K == #-273.15C`
5. **Temperature constructor tests**: `celsius()`, `fahrenheit()`, `kelvins()`, cross-scale conversion
6. **Temperature property tests**: `.value`, `.unit`, `.family`, `.system`
7. **Temperature method tests**: `.to()`, `.abs()`, `.format()`, `.repr()`, `.toDict()`
8. **Temperature PLN round-trip tests**
9. **Volume literal tests**: `#500mL`, `#2L`, `#8floz`, `#1cup`, `#1/3cup`, `#1pt`, `#1qt`, `#1gal`, `#1+1/2gal`
10. **Volume arithmetic tests**: same-system, cross-system, scalar multiply/divide, ratio
11. **Volume fraction exactness**: `#1/3cup + #1/3cup + #1/3cup == #1cup`
12. **Volume comparison tests**: cross-system equality
13. **Volume constructor tests**: `litres()`, `gallons()`, etc.
14. **Volume property and method tests**
15. **Volume PLN round-trip tests**
16. **Regression tests**: all Phase 1 tests still pass, money tests unaffected

Key test vectors from the design doc:
- `#0C == #32F` → true
- `#100C == #212F` → true
- `#-40C == #-40F` → true (scales cross at -40)
- `#0K == #-273.15C` → true (absolute zero)
- `#98.6F` → celsius = `#37C` (verify conversion)
- `#1/3cup + #1/3cup + #1/3cup == #1cup` → true (exact US fractions)
- `#20C + #10C == #30C` → true
- `#20C * 2` → error (not `#40C`)
- Temperature negation: `-#20C` → `#-20C` (allowed, this is negation not multiplication)

---

### Task 16: Documentation

**Files:** `docs/parsley/reference.md`, `docs/parsley/CHEATSHEET.md`
**Estimated effort:** Small

Update documentation with temperature and volume.

Steps:
1. Update §1.16 Unit Literals in reference.md:
   - Add temperature and volume to supported suffixes table
   - Add temperature encoding note (K × 900)
   - Add temperature arithmetic restriction note
2. Update §2.11 Unit Arithmetic in reference.md:
   - Add temperature row (add/subtract only, no multiply/divide)
   - Add temperature error examples
3. Update §5.11 Unit Properties & Methods in reference.md:
   - Note temperature `.toFraction()` is not available
4. Update §6.6 Units in reference.md:
   - Add temperature and volume constructors to the table
5. Update Appendix A Type Summary:
   - Update unit row to mention temperature and volume families
6. Update CHEATSHEET.md:
   - Add temperature to the Unit section with gotchas (no multiply!)
   - Add volume to the Unit section
   - Update suffixes list
   - Add temperature gotcha about multiply/divide

---

## Task Dependencies

```
Task 1 (Temp tables) ──→ Task 3 (Lexer) ──→ Task 4 (Temp eval)
                                                    │
Task 2 (Vol tables) ───→ Task 3 (Lexer) ──→ Task 5 (Vol eval)
                                                    │
                         ┌──────────────────────────┤
                         ↓                          ↓
                    Task 6 (Temp arithmetic)    Task 8 (Vol arithmetic)
                         │
                         ↓
                    Task 7 (Temp restrictions)
                         │
            ┌────────────┤
            ↓            ↓
       Task 9 (Temp)  Task 10 (Vol constructors)
            │            │
            ↓            ↓
       Task 11 (Methods) ──→ Task 12 (Display)
                                    │
                                    ↓
                              Task 13 (PLN) ──→ Task 14 (Errors)
                                                      │
                                                      ↓
                                                Task 15 (Tests) ──→ Task 16 (Docs)
```

Suggested implementation order: 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 → 9 → 10 → 11 → 12 → 13 → 14 → 15 → 16

Tasks 1 and 2 can be done in parallel. Tasks 4/5 can be started once 3 is done. Tasks 6/7/8 can be partially parallelized.

---

## Validation Checklist

- [x] All tests pass: `go test ./...`
- [x] Build succeeds: `make build`
- [x] Linter passes: `golangci-lint run` (no new issues)
- [x] All Phase 2 acceptance criteria in FEAT-118 checked off
- [x] Phase 1 regression: all existing unit tests still pass
- [x] Money regression: all existing money tests still pass
- [x] Temperature conversions verified: `#0C == #32F`, `#100C == #212F`, `#-40C == #-40F`
- [x] Temperature arithmetic restrictions: multiply/divide produce clear errors
- [x] Volume fraction exactness: `#1/3cup + #1/3cup + #1/3cup == #1cup`
- [x] PLN round-trip for temperature and volume literals
- [x] Documentation updated (reference.md, CHEATSHEET.md)
- [x] `work/BACKLOG.md` updated with any new deferrals

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-02-14 | Task 1: Temperature constants and suffixes | ✅ Done | Added FamilyTemperature, TempScale/Offset constants, EncodeTempToSubK/DecodeTempFromSubK helpers |
| 2026-02-14 | Task 2: Volume constants and suffixes | ✅ Done | Added FamilyVolume, SI µL sub-units, US HCN sub-floz, volume bridge (GCD=168) |
| 2026-02-14 | Task 3: Lexer suffix recognition | ✅ Done | Added K, C, F, mL, L, floz, cup, pt, qt, gal to isValidUnitSuffix; longest-match verified (gal vs g) |
| 2026-02-14 | Task 4: Temperature literal evaluation | ✅ Done | Parser encodes via parserEncodeTempAmount with math.Round for float precision |
| 2026-02-14 | Task 5: Volume literal evaluation | ✅ Done | Standard SI/US paths worked automatically once tables populated |
| 2026-02-14 | Task 6: Temperature offset arithmetic | ✅ Done | add: left+right−offset(left), sub: left−right+offset(left); works for same-scale and cross-scale |
| 2026-02-14 | Task 7: Temperature multiply/divide blocked | ✅ Done | UNIT-0011 (multiply), UNIT-0012 (divide), UNIT-0013 (temp/temp); negation still allowed |
| 2026-02-14 | Task 8: Volume arithmetic | ✅ Done | Standard paths from Phase 1; verified fraction exactness and cross-system bridge |
| 2026-02-14 | Task 9: Temperature constructors | ✅ Done | celsius(), fahrenheit(), kelvins() + generic unit(n, "C"); conversion via sub-K pass-through |
| 2026-02-14 | Task 10: Volume constructors | ✅ Done | litres/liters, millilitres/milliliters, gallons, cups, pints, quarts, fluidounces |
| 2026-02-14 | Task 11: Temperature methods/properties | ✅ Done | .value uses DecodeTempFromSubK; .to() changes display hint; .abs() operates on decoded value |
| 2026-02-14 | Task 12: Display and formatting | ✅ Done | Temperature uses full-precision decimal (FormatFloat -1); volume uses existing SI/US display |
| 2026-02-14 | Task 13: PLN temperature/volume | ✅ Done | PLN lexer accepts any letter suffix; PLN parser routes temperature through EncodeTempToSubK |
| 2026-02-14 | Task 14: Error messages | ✅ Done | Added UNIT-0012, UNIT-0013 to error catalog |
| 2026-02-14 | Task 15: Integration tests | ✅ Done | ~60 new tests: literals, arithmetic, restrictions, comparisons, constructors, methods, PLN round-trip |
| 2026-02-14 | Task 16: Documentation | ✅ Done | Updated reference.md (§1.16, §2.11, §5.11, §6.6) and CHEATSHEET.md |

## Deferred Items

Items to add to `work/BACKLOG.md` after implementation:
- Kelvin-based escape hatch for scientific temperature scaling (e.g., `#20C.asKelvin() * 2`)
- Temperature interval type (distinguish "20 degrees" from "20°C reading") — only if users request it
- US customary dry volume units (dry pt, dry qt, bushel) — niche, add on demand
- Metric cooking units (dL — decilitre) — uncommon outside Scandinavia