---
id: PLAN-095
feature: FEAT-118
title: "Implementation Plan for Measurement Units — Phase 1 (MVP)"
status: draft
created: 2026-02-14
---

# Implementation Plan: FEAT-118 Measurement Units — Phase 1

## Overview

Implement built-in unit-of-measurement types for Parsley covering Length, Mass, and Digital Information. All three representations use the same architecture: a single int64 counting fixed sub-units — SI uses µm/mg/B, US Customary uses sub-yards/sub-ounces (HCN = 725,760), and Temperature (Phase 2) uses sub-kelvins (K × 900). This adds literal syntax with the `#` sigil, arithmetic operators, cross-system conversion, and display formatting.

Phase 1 delivers the core infrastructure and three unit families. Temperature, Volume, Area, and polish are deferred to Phases 2–4 (separate plans).

**Design doc:** `work/design/DESIGN-units-v3.md` (authoritative)

## Prerequisites

- [ ] Design doc `DESIGN-units-v3.md` reviewed and approved
- [ ] No blocking dependencies — feature is self-contained
- [ ] Money implementation reviewed as reference (similar integer-storage approach, but units use fixed sub-unit bases rather than variable scale)

## Tasks

### Task 1: Token Type and Lexer — Basic Unit Literals

**Files:** `pkg/parsley/lexer/lexer.go`
**Estimated effort:** Large

Add a `UNIT` token type and teach the lexer to recognise `#` + number + suffix as a unit literal. This task handles basic decimal and integer literals only (fractions and mixed numbers in Task 2).

Steps:
1. Add `UNIT` token type constant (after `MONEY` in the `const` block)
2. Add unit literal detection in `NextToken()` — when `#` is followed by a digit or `-`, enter unit literal mode
3. Implement `readUnitLiteral()`:
   - Consume optional `-` sign
   - Consume integer or decimal number
   - Consume longest-matching unit suffix from a suffix table
   - Return `UNIT` token with `Literal` containing the full text (e.g., `#12.3m`)
4. Build the suffix lookup table (all Phase 1 suffixes):
   - Length SI: `mm`, `cm`, `m`, `km`
   - Length US: `in`, `ft`, `yd`, `mi`
   - Mass SI: `mg`, `g`, `kg`
   - Mass US: `oz`, `lb`
   - Data: `B`, `kB`, `MB`, `GB`, `TB`, `KiB`, `MiB`, `GiB`, `TiB`
5. Use longest-match for suffix disambiguation (`mi` vs `m`, `mg` vs `m`, `kB` vs `KiB`)
6. Ensure `#` followed by non-digit/non-minus is an `ILLEGAL` token (not confused with Money's `CODE#amount`)

Tests:
- `#12.3m` → UNIT token, literal `#12.3m`
- `#45oz` → UNIT token
- `#-6m` → UNIT token (negative)
- `#64KB` → UNIT token
- `#1KiB` → UNIT token (not `K` + `iB`)
- `#5mi` → UNIT token (not `m` + `i`)
- `EUR#50` → still MONEY token (regression)
- `#abc` → ILLEGAL (no digit after `#`)

---

### Task 2: Lexer — Fraction and Mixed Number Literals

**Files:** `pkg/parsley/lexer/lexer.go`
**Estimated effort:** Medium

Extend `readUnitLiteral()` to handle fraction (`#3/8in`) and mixed number (`#92+5/8in`) forms.

Steps:
1. After consuming the initial integer, check for `/` → fraction mode
   - Consume denominator integer
   - Then consume suffix
2. After consuming the initial integer, check for `+` followed by digit → mixed number mode
   - Consume numerator, `/`, denominator
   - Then consume suffix
3. Store parsed components in the token (use `Literal` for the full text; parsed values extracted by the parser)
4. Handle negative fractions: `#-3/8in` and negative mixed: `#-2+3/8in`

Tests:
- `#3/8in` → UNIT token, literal `#3/8in`
- `#1/3cup` → UNIT token (Phase 2 suffix — lexer should still tokenise; parser/evaluator rejects unknown suffix)
- `#92+5/8in` → UNIT token, literal `#92+5/8in`
- `#-3/8in` → UNIT token (negative fraction)
- `#-2+3/8in` → UNIT token (negative mixed)
- `#1/0in` → lexer produces token; evaluator handles zero-denominator error
- `#3/8` → ILLEGAL or error (no suffix)

---

### Task 3: AST Nodes

**Files:** `pkg/parsley/ast/ast.go`
**Estimated effort:** Small

Add AST node types for unit literals. Follow the `MoneyLiteral` pattern.

Steps:
1. Add `UnitLiteral` node:
   ```
   type UnitLiteral struct {
       Token       lexer.Token
       Value       int64   // amount in sub-units (µm, mg, B, or HCN-scaled)
       Suffix      string  // original unit suffix ("m", "in", "kg", etc.)
       System      string  // "SI" or "US"
       Family      string  // "length", "mass", "data"
       IsNegative  bool
   }
   ```
2. Add `FractionUnitLiteral` node:
   ```
   type FractionUnitLiteral struct {
       Token       lexer.Token
       Numerator   int64
       Denominator int64
       Suffix      string
       System      string
       Family      string
       IsNegative  bool
   }
   ```
3. Add `MixedUnitLiteral` node:
   ```
   type MixedUnitLiteral struct {
       Token       lexer.Token
       Whole       int64
       Numerator   int64
       Denominator int64
       Suffix      string
       System      string
       Family      string
       IsNegative  bool
   }
   ```
4. Implement `expressionNode()`, `TokenLiteral()`, `String()` for each

Tests:
- AST String() round-trips: `UnitLiteral.String()` → `#12.3m`
- FractionUnitLiteral.String() → `#3/8in`
- MixedUnitLiteral.String() → `#92+5/8in`

---

### Task 4: Parser — Unit Literal Parsing

**Files:** `pkg/parsley/parser/parser.go`
**Estimated effort:** Medium

Register a prefix parse function for the `UNIT` token type that produces the appropriate AST node.

Steps:
1. Register `parseUnitLiteral` as the prefix parser for `lexer.UNIT`
2. Implement `parseUnitLiteral()`:
   - Parse the token literal string to determine form (basic, fraction, mixed)
   - Look up suffix in the unit table to determine system and family
   - For SI fractions: compute the sub-unit amount at parse time (e.g., `#1/3m` → 1,000,000 / 3 = 333,333 µm, truncated to whole sub-units)
   - For US Customary fractions: preserve numerator and denominator
   - Return the appropriate AST node
3. Handle unary negation of unit literals (`-#6m`): the existing prefix `-` handler should work if `UnitLiteral` is a valid expression

Tests:
- Parse `#12.3m` → `UnitLiteral{Value: 12300000, Suffix: "m", System: "SI", Family: "length"}` (12,300,000 µm)
- Parse `#3/8in` → `FractionUnitLiteral{Numerator: 3, Denominator: 8, Suffix: "in", System: "US", Family: "length"}`
- Parse `#92+5/8in` → `MixedUnitLiteral{Whole: 92, Numerator: 5, Denominator: 8, ...}`
- Parse `#1/3m` → `UnitLiteral` (SI fraction converted to decimal)
- Unknown suffix → parse error

---

### Task 5: Object Types — SIUnit and ImperialUnit

**Files:** `pkg/parsley/evaluator/evaluator.go` (object type consts), new file `pkg/parsley/evaluator/unit_objects.go`
**Estimated effort:** Medium

Define the runtime object types. Both SI and Imperial use the same architecture: a single int64 counting fixed sub-units.

Steps:
1. Add object type constants: `SI_UNIT_OBJ = "SI_UNIT"`, `IMPERIAL_UNIT_OBJ = "IMPERIAL_UNIT"`
2. Create `pkg/parsley/evaluator/unit_objects.go` with:
   ```
   const HCN int64 = 725_760  // 2⁸ × 3⁴ × 5 × 7

   // Sub-unit bases per family (SI)
   // Length: 1 µm = 1,  so 1 m = 1,000,000
   // Mass:   1 mg = 1,  so 1 g = 1,000
   // Data:   1 B  = 1

   type SIUnit struct {
       Amount      int64   // count of sub-units (µm, mg, or B)
       Family      string
       DisplayHint string
   }

   type ImperialUnit struct {
       Amount      int64   // count of sub-units (HCN-scaled)
       Family      string
       DisplayHint string
   }
   ```
3. Implement `Type()`, `Inspect()` for both
4. `Inspect()` should produce the display form (e.g., `12.3m`, `3/8in`)
5. Add helper functions:
   - `newSIUnit(amount int64, family, hint string) *SIUnit`
   - `newImperialUnit(amount int64, family, hint string) *ImperialUnit`
   - `siSubUnitsPerUnit(suffix string) int64` — returns sub-units per display unit (e.g., `"m"` → 1,000,000, `"cm"` → 10,000, `"mm"` → 1,000)
   - `imperialToSubUnits(whole, numerator, denominator int64, suffix string) int64` — converts a fraction literal to HCN-scaled amount
   - `gcdInt64(a, b int64) int64` — for fraction display

Tests:
- `SIUnit{Amount: 12300000, ...}.Inspect()` → `"12.3m"` (12,300,000 µm ÷ 1,000,000 per m)
- `ImperialUnit` with amount for 3/8 inch → `Inspect()` → `"3/8in"`
- `siSubUnitsPerUnit("m")` → 1,000,000; `siSubUnitsPerUnit("cm")` → 10,000; `siSubUnitsPerUnit("mm")` → 1,000
- `imperialToSubUnits(0, 3, 8, "in")` → 7560 (3 × 20160 / 8)
- GCD reduction: `gcdInt64(7560, 725760)` produces correct reduced fraction

---

### Task 6: Unit Tables — Suffix Maps, Ratios, and Bridge Constants

**Files:** new file `pkg/parsley/evaluator/unit_tables.go`
**Estimated effort:** Medium

Centralise all unit metadata: suffix → family/system mapping, within-system ratios, cross-system bridge ratios, and display defaults.

Steps:
1. Define `UnitInfo` struct: `{Suffix, Family, System, BaseRatio}` where BaseRatio converts to the family's base unit
2. Build suffix lookup map for all Phase 1 units (see §4 of design doc)
3. SI sub-units-per-display-unit table (all exact integers):
   - Length: mm=1,000 µm, cm=10,000 µm, m=1,000,000 µm, km=1,000,000,000 µm
   - Mass: mg=1, g=1,000 mg, kg=1,000,000 mg
   - Data: B=1, kB=1,000, MB=1,000,000, GB=1,000,000,000, TB=1,000,000,000,000, KiB=1,024, MiB=1,048,576, GiB=1,073,741,824, TiB=1,099,511,627,776
4. US Customary within-system ratios (all exact integer multiples of HCN base):
   - Length: in=HCN/36, ft=HCN/3, yd=HCN, mi=HCN×1,760
   - Mass: oz=HCN, lb=HCN×16
5. Cross-system bridge ratios:
   - Length: 1 in = 0.0254 m → 1 in = 25,400 µm (SI). 1 in = HCN/36 = 20,160 sub-yards (US)
   - Mass: 1 lb = 453.59237 g → 1 lb = 453,592.37 mg ≈ 453,592 mg (SI, truncated). 1 lb = 16 oz in US
6. Display defaults table (decimal places per suffix — see §8.1)
7. Common denominator set for fraction display: `{2, 3, 4, 5, 6, 7, 8, 10, 12, 16, 32, 64}`

Tests:
- Suffix lookup: `"in"` → `{Family: "length", System: "US", ...}`
- `"m"` → `{Family: "length", System: "SI", ...}`
- `"KiB"` → `{Family: "data", System: "SI", ...}`
- All suffixes resolve to a valid UnitInfo

---

### Task 7: Evaluator — Unit Literal Evaluation

**Files:** `pkg/parsley/evaluator/evaluator.go` or new file `pkg/parsley/evaluator/eval_units.go`
**Estimated effort:** Medium

Evaluate AST unit nodes into runtime SIUnit/ImperialUnit objects.

Steps:
1. Add cases in `Eval()` for `*ast.UnitLiteral`, `*ast.FractionUnitLiteral`, `*ast.MixedUnitLiteral`
2. For `UnitLiteral` (SI): multiply parsed value by sub-units-per-display-unit to get Amount (e.g., `#12.3m` → 12.3 × 1,000,000 = 12,300,000 µm)
3. For `UnitLiteral` (US): convert to HCN-scaled amount using within-system ratios
4. For `FractionUnitLiteral` (US): `amount = (numerator * HCN_per_base_unit) / denominator` — must be exact integer; error if not
5. For `FractionUnitLiteral` (SI): compute `(numerator * sub_units_per_unit) / denominator`, truncate to integer
6. For `MixedUnitLiteral`: `amount = whole * sub_units_per_unit + (numerator * sub_units_per_unit) / denominator`
7. Handle unit suffix → base unit conversion using ratios from Task 6
8. Handle negation

Tests:
- `#12.3m` evaluates to `SIUnit{Amount: 12300000, Family: "length", DisplayHint: "m"}` (12,300,000 µm)
- `#5cm` evaluates to `SIUnit{Amount: 50000, ...}` (50,000 µm)
- `#3mm` evaluates to `SIUnit{Amount: 3000, ...}` (3,000 µm)
- `#3/8in` evaluates to `ImperialUnit{Amount: 7560, Family: "length", DisplayHint: "in"}`
- `#92+5/8in` evaluates correctly
- `#1/3m` evaluates to `SIUnit{Amount: 333333, ...}` (333,333 µm — truncated)
- `#-6m` evaluates to negative SIUnit (Amount: -6,000,000)
- `#1ft` → amount = 3 × 20160 = 60480 sub-yards (1 foot = 12 inches = 1/3 yard)
- `#1mi` → correct sub-yard amount
- `#1kg` → Amount: 1,000,000 (1 kg = 1,000 g = 1,000,000 mg)
- `#500g` → Amount: 500,000 mg
- `#1KiB` → Amount: 1,024 B

---

### Task 8: Arithmetic — Same-System Operations

**Files:** `pkg/parsley/evaluator/eval_infix.go`, `pkg/parsley/evaluator/eval_units.go`
**Estimated effort:** Large

Implement unit arithmetic in the infix evaluator, following the Money pattern. Same-system operations first (always exact).

Steps:
1. Add cases in `evalInfixExpression()` for SI_UNIT_OBJ and IMPERIAL_UNIT_OBJ combinations:
   - `SIUnit op SIUnit` → `evalSIUnitInfixExpression()`
   - `ImperialUnit op ImperialUnit` → `evalImperialUnitInfixExpression()`
   - `SIUnit op Integer/Float` and `Integer/Float op SIUnit` → scalar operations
   - `ImperialUnit op Integer/Float` and `Integer/Float op ImperialUnit` → scalar operations
2. Implement `evalSIUnitInfixExpression()`:
   - Family must match → error if different families
   - `+`, `-`: direct integer addition/subtraction (both already in same sub-unit base), left side wins display hint
   - `==`, `!=`, `<`, `>`, `<=`, `>=`: direct integer comparison (same sub-unit base)
   - `/` (unit/unit): return plain number (dimensionless ratio)
3. Implement `evalImperialUnitInfixExpression()`:
   - Family must match
   - `+`, `-`: both already in HCN base → just add/subtract amounts, left side wins display hint
   - Comparison: direct integer comparison (same base)
   - `/` (unit/unit): return dimensionless ratio
4. Implement scalar multiplication/division for both systems
5. Implement negation (`-unit`) in prefix evaluator
6. Error cases:
   - Different family: `"cannot add length to mass"`
   - `unit * unit`: `"unit multiplication not supported"`
   - `scalar + unit`, `scalar - unit`, `scalar / unit`: `"cannot add/subtract/divide scalar and unit"`

Tests:
- `#1/3cup + #1/3cup + #1/3cup == #1cup` — exact (Phase 2 suffix, but test the mechanism with `in` equivalents)
- `#3/8in + #5/8in` → `#1in` (exact)
- `#12.3m + #0.7m` → `#13m` (exact, 12,300,000 + 700,000 = 13,000,000 µm)
- `#2ft + #6in` → `#2+1/2ft` (left side wins, within US system)
- `#1m + #50cm` → `#1.5m` (left side wins, within SI; 1,000,000 + 500,000 = 1,500,000 µm)
- `#5cm + #3mm` → `#5.3cm` (50,000 + 3,000 = 53,000 µm, displayed in cm)
- `#10m * 3` → `#30m`
- `3 * #10m` → `#30m` (commutative)
- `#10m / 4` → `#2.5m`
- `#10m / #5m` → `2` (plain number)
- `-#5m` → `#-5m`
- `#5m + #5kg` → error (different families)
- `#5m * #5m` → error (unit × unit)
- `5 + #5m` → error (scalar + unit)

---

### Task 9: Arithmetic — Cross-System Operations

**Files:** `pkg/parsley/evaluator/eval_units.go`
**Estimated effort:** Large

When the left operand is SI and right is US (or vice versa), convert the right operand into the left's system using bridge ratios, then perform the operation.

Steps:
1. Add infix cases for cross-system pairs:
   - `SIUnit op ImperialUnit` → convert right to SI, then same-system SI arithmetic
   - `ImperialUnit op SIUnit` → convert right to US, then same-system US arithmetic
2. Implement `convertToSI(imperial *ImperialUnit) *SIUnit`:
   - Use bridge ratio (e.g., 1 in = 25,400 µm) to convert
   - First convert Imperial amount from sub-yards to inches (÷ 20,160)
   - Then multiply by 25,400 to get µm
3. Implement `convertToImperial(si *SIUnit) *ImperialUnit`:
   - Reverse bridge ratio: divide µm by 25,400 to get inches, multiply by 20,160 to get sub-yards
   - May involve rounding (SI µm precision is ~1 µm, Imperial sub-yard precision is ~1.26 µm)
4. Cross-system comparison: normalise both to SI (or both to US) for comparison

Tests:
- `#1cm + #1in` → `#3.54cm` (result in cm, SI)
- `#1in + #1cm` → approximately `#1+3/8in` (result in inches, US, may round)
- `#254cm + #1m` → `#354cm` (within SI, left side wins)
- `#1in == #25.4mm` → true
- `#1024B == #1KiB` → true
- `#1lb == #16oz` → true
- `#10cm + #1in` → `#12.54cm` (exact: 1in = 2.54cm)

---

### Task 10: Constructors — Named and Generic (Plural Forms)

**Files:** `pkg/parsley/evaluator/builtins.go` or new file `pkg/parsley/evaluator/builtins_units.go`
**Estimated effort:** Medium

Register named constructor functions and the generic `unit()` constructor.

Steps:
1. Register named constructors as builtins (arity 1) — **all plural forms**:
   - SI Length: `metres()`/`meters()`, `centimetres()`/`centimeters()`, `millimetres()`/`millimeters()`, `kilometres()`/`kilometers()`
   - US Length: `inches()`, `feet()`, `yards()`, `miles()`
   - SI Mass: `grams()`, `milligrams()`, `kilograms()`
   - US Mass: `ounces()`, `pounds()`
   - Data: `bytes()`, `kilobytes()`, `megabytes()`, `gigabytes()`, `terabytes()`, `kibibytes()`, `mebibytes()`, `gibibytes()`, `tebibytes()`
2. Each constructor accepts either:
   - A number (integer or float) → create a unit in that unit
   - A unit value (same family) → convert to the target unit
   - A unit value (different family) → error
3. Register `unit(value, suffix_string)` generic constructor (arity 2):
   - Look up suffix in unit table
   - If value is a number → create unit
   - If value is a unit → convert

Tests:
- `metres(123)` → `#123m`
- `metres(#12in)` → `#0.3048m` (cross-system conversion)
- `inches(#1m)` → approximately `#39+3/8in`
- `kilograms(#2.2lb)` → approximately `#0.998kg`
- `feet(12)` → `#12ft`
- `unit(123, "m")` → `#123m`
- `unit(#12in, "m")` → `#0.3048m`
- `metres("hello")` → error (wrong argument type)
- `metres(#5kg)` → error (different family)

---

### Task 11: Properties

**Files:** `pkg/parsley/evaluator/eval_computed_properties.go` or property dispatch
**Estimated effort:** Small

Implement `.value`, `.unit`, `.family`, `.system` properties on unit objects.

Steps:
1. Hook into property access for SI_UNIT_OBJ and IMPERIAL_UNIT_OBJ
2. `.value` → decode Amount to float in display-hint unit (e.g., `#3/8in.value` → `0.375`)
3. `.unit` → return DisplayHint as string
4. `.family` → return Family as string
5. `.system` → return `"SI"` or `"US"` as string

Tests:
- `#3/8in.value` → `0.375`
- `#3/8in.unit` → `"in"`
- `#3/8in.family` → `"length"`
- `#3/8in.system` → `"US"`
- `#12.3m.value` → `12.3`
- `#12.3m.system` → `"SI"`
- `#1KiB.value` → `1024` (value in bytes? or 1 in KiB?) — **Decision: value in the display-hint unit**, so `1`

---

### Task 12: Methods — Registry and Core Methods

**Files:** new file `pkg/parsley/evaluator/methods_units.go`
**Estimated effort:** Large

Implement methods via the declarative method registry (following `methods_money.go` pattern).

Steps:
1. Create `SIUnitMethodRegistry` and `ImperialUnitMethodRegistry` (or a shared `UnitMethodRegistry`)
2. Register in method dispatch (`eval_method_dispatch.go`)
3. Implement methods:
   - `.to(unit_string)` → convert to target unit, return new unit object
   - `.abs()` → absolute value
   - `.format()` / `.format(options)` → formatted string with locale defaults
   - `.repr()` → parseable literal string (`"#3/8in"`, `"#12.3m"`, `"#92+5/8in"`)
   - `.toDict()` → `{value, unit, family, system}` dictionary
   - `.inspect()` → debug dict including internal representation (Amount, sub-unit base)
   - `.toFraction()` → fraction string for US Customary (`"3/8\""`); error for SI

Tests:
- `#1mi.to("km")` → `#1.609km` (approximately)
- `#-5m.abs()` → `#5m`
- `#3/8in.repr()` → `"#3/8in"`
- `#92+5/8in.repr()` → `"#92+5/8in"`
- `#12.3m.repr()` → `"#12.3m"`
- `#12.3m.toDict()` → `{value: 12.3, unit: "m", family: "length", system: "SI"}`
- `#3/8in.toFraction()` → `"3/8\""`
- `#12.3m.toFraction()` → error (SI unit)

---

### Task 13: Display and Formatting

**Files:** `pkg/parsley/evaluator/unit_objects.go`, `pkg/parsley/evaluator/eval_units.go`, `pkg/parsley/evaluator/pln_hooks.go`
**Estimated effort:** Medium

Implement display formatting rules from §8 of the design doc.

Steps:
1. **US Customary fraction display** (in `Inspect()` / `format()`):
   - `g = GCD(amount, HCN)`
   - `num = amount / g`, `den = HCN / g` (adjusted for display-hint unit)
   - If `den == 1`: integer display
   - If `num > den`: mixed number with `+` separator
   - If `den` in common set: fraction display
   - Else: decimal fallback
2. **SI decimal display**:
   - Apply context-aware defaults from the display defaults table
   - Trailing zero handling per the defaults (e.g., `1.83m` not `1.830000m`)
3. **String interpolation**: output without `#` sigil, no space (e.g., `1.83m`)
4. **PLN output**: output with `#` sigil (e.g., `#1.83m`)
   - Hook into PLN serialisation in `pln_hooks.go`
5. **`.repr()` output**: always `#` + parseable literal

Tests:
- US display: amount for 3/8 inch → `"3/8in"`
- US display: amount for 1 inch → `"1in"` (integer, not `"1/1in"`)
- US display: amount for 1+3/8 inch → `"1+3/8in"` (mixed number)
- US display: uncommon denominator → decimal fallback
- SI display: `#1.83m` in interpolation → `"1.83m"`
- SI display: `#25mm` → `"25mm"` (0 decimal places for mm)
- PLN: `#1.83m` → `"#1.83m"`
- repr: `#92+5/8in` → `"#92+5/8in"`

---

### Task 14: Error Messages

**Files:** `pkg/parsley/errors/errors.go`, `pkg/parsley/evaluator/eval_errors.go`
**Estimated effort:** Medium

Parsley is known for concise, clear, human-focused error messages — the kind a casual programmer can read, understand, and act on without consulting documentation. Unit errors must continue this standard. Every error uses the structured error catalogue (`ErrorDef` with `Template` + `Hints`) and includes position information.

**Principles (matching existing Parsley style):**
- **Template** says *what went wrong* in plain English — no jargon, no internal type names
- **Hints** say *what to do instead* — concrete, showing corrected code where possible
- Use fuzzy matching (`FindClosestMatch`) for suffix typos, just as Parsley does for identifiers and methods
- Name the specific families/units involved, not generic "incompatible types"

Steps:
1. Reserve a block of error codes for units (e.g., `UNIT-0001` through `UNIT-00xx`) in `ErrorCatalog`
2. Add the following catalogue entries:

   **Arithmetic errors:**
   ```
   UNIT-0001: Cross-family arithmetic
     Template: "Cannot {{.Operator}} {{.LeftFamily}} and {{.RightFamily}}"
     Hints:    ["units must be the same family to add or subtract"]

   UNIT-0002: Scalar ± unit
     Template: "Cannot {{.Operator}} number and unit"
     Hints:    ["write {{.Example}} — numbers and units don't mix"]
     (where Example is e.g. "#5m + #5m, not 5 + #5m")

   UNIT-0003: Scalar / unit
     Template: "Cannot divide number by unit"
     Hints:    ["you can divide a unit by a number (#10m / 5) but not the other way around"]

   UNIT-0004: Unit × unit
     Template: "Cannot multiply unit by unit"
     Hints:    ["derived units (area, etc.) are planned for a future release"]
   ```

   **Conversion errors:**
   ```
   UNIT-0005: Wrong family in constructor
     Template: "Cannot convert {{.FromFamily}} to {{.ToFamily}}"
     Hints:    ["{{.Constructor}} accepts {{.ToFamily}} values like {{.Example}}"]
     (where Example is e.g. "#5in or #100cm" for metres())

   UNIT-0006: Wrong argument type in constructor
     Template: "{{.Constructor}} expects a number or unit, got {{.Got}}"
     Hints:    ["{{.Constructor}} creates or converts {{.Family}} values: {{.Example}}"]
   ```

   **Literal/parse errors:**
   ```
   UNIT-0007: Unknown unit suffix
     Template: "Unknown unit suffix '{{.Suffix}}'"
     Hints:    [fuzzy-match suggestion if available,
               "unit suffixes are abbreviations: m, cm, km, in, ft, yd, g, kg, oz, lb, B, MB, etc."]

   UNIT-0008: Zero denominator
     Template: "Fraction denominator cannot be zero"

   UNIT-0009: Malformed mixed number
     Template: "Invalid mixed number literal"
     Hints:    ["write #2+3/8in — whole number, then + then fraction, then suffix"]

   UNIT-0010: Negative mixed number sign placement
     Hints:    ["for negative mixed numbers, write #-2+3/8in — the sign applies to the whole value"]
   ```

   **Range errors:**
   ```
   UNIT-0011: Overflow
     Template: "Unit value overflow"
     Hints:    ["the maximum representable value is approximately {{.MaxHuman}}"]
     (where MaxHuman is e.g. "77 AU for length" or "9,200 tonnes for mass")
   ```

3. Add helper functions in `eval_errors.go`:
   - `newUnitError(code string, tok lexer.Token, data map[string]any) *Error`
   - `newUnitErrorWithSuggestion(code string, tok lexer.Token, suffix string, knownSuffixes []string) *Error` — uses `FindClosestMatch` for fuzzy suffix suggestions
4. Ensure all error paths from Tasks 7–12 use these catalogue entries (not ad-hoc strings)
5. Test that every error includes line/column position

Tests — each must verify the **exact message template** and **at least one hint**:
- `#5m + #5kg` → UNIT-0001: message contains "length" and "mass", hint mentions "same family"
- `5 + #5m` → UNIT-0002: hint shows the corrected form `#5m + #5m`
- `0 - #6C` → UNIT-0002: hint shows `#0C - #6C`
- `10 / #5m` → UNIT-0003: hint explains the asymmetry
- `#5m * #3m` → UNIT-0004: hint mentions "future release"
- `metres(#5kg)` → UNIT-0005: hint shows what `metres()` accepts
- `metres("hello")` → UNIT-0006: message says "got string"
- `#5meter` → UNIT-0007: hint says `did you mean 'm'?` (fuzzy match)
- `#5xyz` → UNIT-0007: hint lists valid suffixes (no close match)
- `#1/0in` → UNIT-0008
- `#2+-3/8in` → UNIT-0009 or UNIT-0010: hint shows correct form
- Overflow scenario → UNIT-0011: hint gives human-readable max
- All errors include correct line and column numbers

---

### Task 15: Integration Tests

**Files:** new file `pkg/parsley/tests/units_test.go`
**Estimated effort:** Large

Comprehensive test suite covering the full feature end-to-end.

Steps:
1. **Literal parsing tests**: all forms (basic, fraction, mixed, negative) for all Phase 1 suffixes
2. **Round-trip tests**: literal → parse → evaluate → display → matches original
3. **Within-system arithmetic tests**: exact integer results
4. **Cross-system arithmetic tests**: bridge ratio conversions, precision
5. **Comparison tests**: cross-system equality
6. **Constructor tests**: named and generic constructors, conversion
7. **Property tests**: `.value`, `.unit`, `.family`, `.system`
8. **Method tests**: `.to()`, `.abs()`, `.format()`, `.repr()`, `.toDict()`, `.toFraction()`
9. **Display tests**: fraction formatting, decimal formatting, interpolation, PLN
10. **Error tests**: all error conditions produce correct error messages
11. **Regression tests**: Money literals still work, `#` in other contexts unaffected

Key test vectors from the design doc:
- `#1/3cup + #1/3cup + #1/3cup == #1cup` (Phase 2, but test with inches: `#1/3yd + #1/3yd + #1/3yd == #1yd`)
- `#3/8in + #5/8in == #1in`
- `#12.3m + #0.7m == #13m`
- `#1in == #25.4mm`
- `#1024B == #1KiB`
- `#254cm == #2.54m`
- `#1cm + #1in` → result in cm
- `#1in + #1cm` → result in inches

---

### Task 16: Documentation

**Files:** `docs/parsley/reference.md`, `docs/parsley/CHEATSHEET.md`
**Estimated effort:** Small

Update Parsley documentation with unit types.

Steps:
1. Add Units section to `docs/parsley/reference.md`:
   - Literal syntax (all forms)
   - Supported suffixes (Phase 1)
   - Arithmetic rules
   - Constructors and conversion
   - Properties and methods
   - Display formatting rules
2. Add to `docs/parsley/CHEATSHEET.md`:
   - `#` sigil (not `$`!)
   - SI fractions are sugar (truncated decimal), US fractions are exact
   - `+` in mixed numbers (not `-`)
   - `scalar + unit` is an error
   - Left side wins for display

---

## Task Dependencies

```
Task 1 (Lexer basic) ──→ Task 2 (Lexer fractions)
                    └──→ Task 3 (AST) ──→ Task 4 (Parser)
                                                │
Task 5 (Objects) ──→ Task 6 (Tables) ──→ Task 7 (Evaluator)
                                                │
                         ┌──────────────────────┤
                         ↓                      ↓
                    Task 8 (Arithmetic)    Task 10 (Constructors)
                         │                      │
                         ↓                      ↓
                    Task 9 (Cross-system)  Task 11 (Properties)
                         │                      │
                         ↓                      ↓
                    Task 14 (Errors) ←── Task 12 (Methods)
                                                │
                                                ↓
                                          Task 13 (Display)
                                                │
                                                ↓
                                    Task 15 (Integration tests)
                                                │
                                                ↓
                                          Task 16 (Docs)
```

Suggested implementation order: 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 → 9 → 10 → 11 → 12 → 13 → 14 → 15 → 16

Tasks 5+6 can be started in parallel with Tasks 1–4 (no code dependency, just conceptual).

---

## Validation Checklist

- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `make build`
- [ ] Linter passes: `golangci-lint run`
- [ ] All Phase 1 acceptance criteria in FEAT-118 checked off
- [ ] Money regression: all existing money tests still pass
- [ ] Round-trip test: every unit literal parses and displays back to itself
- [ ] Cross-system equality: `#1in == #25.4mm` passes
- [ ] Documentation updated
- [ ] `work/BACKLOG.md` updated with deferrals

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| | | | |

## Deferred Items

Items to add to `work/BACKLOG.md` after implementation:
- Phase 2: Temperature (K × 900) + Volume — separate FEAT/PLAN
- Phase 3: Area (`unit2` suffixes) — separate FEAT/PLAN
- Phase 4: Compound display, derived units, schema integration — separate FEAT/PLAN
- Tree-sitter grammar updates for unit literal highlighting
- Locale-aware formatting (e.g., comma vs period decimal separator)
- Overflow detection and graceful error handling (verify int64 limits are checked)