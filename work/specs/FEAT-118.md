---
id: FEAT-118
title: "Measurement Units"
status: draft
priority: high
created: 2026-02-14
author: "@human"
design: "work/design/DESIGN-units-v3.md"
---

# FEAT-118: Measurement Units

## Summary

Add built-in unit-of-measurement types to Parsley: integer storage, no floating-point drift, exact round-trips, and natural literal syntax. Three internal representations — SI (fixed sub-unit: µm, mg, B), US Customary (fixed-denominator fraction over HCN = 725,760), and Temperature (K × 900) — all use the same architecture: a single int64 counting fixed sub-units. They present a unified interface with the `#` literal sigil, standard arithmetic operators, and automatic cross-system conversion.

## User Story

As a Parsley developer building real-world applications, I want to work with physical measurements (lengths, weights, temperatures, file sizes) using natural syntax and exact arithmetic so that I can write correct calculations without worrying about floating-point drift, unit conversion bugs, or fraction rounding.

## Acceptance Criteria

### Phase 1 — MVP (Length, Mass, Digital Information)

#### Literal Syntax
- [ ] Basic unit literals: `#12.3m`, `#45oz`, `#64KB`
- [ ] Fraction unit literals: `#3/8in`, `#1/2km`
- [ ] Mixed number unit literals: `#92+5/8in`, `#2+3/8in`
- [ ] Negative unit literals: `#-6m`, `-#6m` (both produce the same value; `#-6m` is canonical)
- [ ] Lexer distinguishes `#50m` (unit) from `EUR#50` (Money)

#### Internal Representation — SI
- [ ] SI units stored as int64 count of fixed sub-units — no scale factor
- [ ] Sub-units: micrometre/µm for length (1 m = 1,000,000), milligram/mg for mass (1 g = 1,000), byte/B for data (1 B = 1)
- [ ] Fraction literals for SI are syntactic sugar for division: `#1/3m` → 333,333 µm (truncated to whole sub-units)

#### Internal Representation — US Customary
- [ ] US Customary units stored as int64 numerator over fixed HCN = 725,760
- [ ] Base units: yard (length), ounce (mass)
- [ ] Fraction literals for US Customary are stored exactly: `#1/3in` is exact

#### Unit Families — Length
- [ ] SI: `mm`, `cm`, `m`, `km`
- [ ] US: `in`, `ft`, `yd`, `mi`
- [ ] Cross-system bridge: 1 in = 0.0254 m (exact)

#### Unit Families — Mass
- [ ] SI: `mg`, `g`, `kg`
- [ ] US: `oz`, `lb`
- [ ] Cross-system bridge: 1 lb = 453.59237 g (exact)

#### Unit Families — Digital Information
- [ ] Decimal: `B`, `kB`, `MB`, `GB`, `TB`
- [ ] Binary: `KiB`, `MiB`, `GiB`, `TiB`
- [ ] Base unit: byte; all conversions are exact integer ratios

#### Arithmetic
- [ ] `unit + unit` (same family): result in left operand's system/display hint
- [ ] `unit - unit` (same family): result in left operand's system/display hint
- [ ] `unit * scalar` and `scalar * unit`: scales the value
- [ ] `unit / scalar`: scales the value (may truncate, same as Money)
- [ ] `unit / unit` (same family): returns dimensionless plain number (ratio)
- [ ] `-unit`: negation
- [ ] Within-system arithmetic is always exact (integer operations, no scale promotion)
- [ ] Cross-system arithmetic uses bridge ratios; rounding at SI↔US boundary only
- [ ] Precision: µm-level on both sides (~1 µm SI, ~1.26 µm US)
- [ ] `unit + unit` (different family): error
- [ ] `unit * unit`: error (derived units deferred)
- [ ] `scalar + unit`, `scalar - unit`, `scalar / unit`: error (no implicit promotion)

#### Comparison
- [ ] Equality normalises across systems: `#1in == #25.4mm` → true
- [ ] `#1024B == #1KiB` → true
- [ ] `#254cm == #2.54m` → true

#### Constructors and Conversion
- [ ] Named constructors (plural): `metres(123)`, `inches(#1cm)`, `kilograms(#2.2lb)`, etc.
- [ ] SI spelling primary (`metres`), US spelling alias (`meters`)
- [ ] Generic constructor: `unit(123, "m")`, `unit(#12in, "m")`
- [ ] `.to(unit)` method: `#1mi.to("km")`

#### Properties
- [ ] `.value` — decoded value as float in display-hint unit
- [ ] `.unit` — display-hint unit suffix as string
- [ ] `.family` — unit family as string (`"length"`, `"mass"`, `"data"`)
- [ ] `.system` — measurement system as string (`"SI"`, `"US"`)

#### Methods
- [ ] `.to(unit)` — convert to another unit
- [ ] `.abs()` — absolute value
- [ ] `.format()` / `.format(options)` — formatted string
- [ ] `.repr()` — parseable literal (`"#3/8in"`, `"#92+5/8in"`)
- [ ] `.toDict()` — `{value, unit, family, system}`
- [ ] `.inspect()` — debug output including internals
- [ ] `.toFraction()` — fraction string for US Customary values

#### Display and Formatting
- [ ] US Customary: fraction display via GCD reduction; decimal fallback for uncommon denominators
- [ ] SI: decimal display with context-aware defaults (m→2dp, cm→1dp, mm→0dp, kg→2dp, g→0dp)
- [ ] String interpolation: no `#` sigil, no space (`"1.83m"`)
- [ ] PLN output uses literal syntax: `#1.83m`
- [ ] `.repr()` always produces parseable literal

#### Error Messages

Parsley is known for concise, clear, human-focused error messages — the kind a casual programmer can read, understand, and act on without consulting documentation. Unit errors must continue this standard. Every error must use the structured error catalogue (`pkg/parsley/errors/errors.go`) with a `Template` and actionable `Hints`.

- [ ] Cross-family arithmetic: message names both families, hint shows what works
  - `#5m + #5kg` → `"Cannot add length to mass"` / hint: `"units must be the same family to add or subtract"`
- [ ] Scalar ± unit: message explains the type mismatch, hint shows the fix
  - `5 + #5m` → `"Cannot add number to unit"` / hint: `"write #5m + #5m, not 5 + #5m — numbers and units don't mix"`
  - `0 - #6C` → hint: `"write #0C - #6C to subtract units"`
- [ ] Unit × unit: message explains it's unsupported, hint sets expectations
  - `#5m * #3m` → `"Cannot multiply unit by unit"` / hint: `"derived units (area, etc.) are planned for a future release"`
- [ ] Scalar / unit: clear asymmetry explanation
  - `10 / #5m` → `"Cannot divide number by unit"` / hint: `"you can divide a unit by a number (#10m / 5) but not the other way around"`
- [ ] Wrong family in constructor: names both families, shows correct usage
  - `metres(#5kg)` → `"Cannot convert mass to length"` / hint: `"metres() accepts length values like #5in or #100cm"`
- [ ] Temperature multiply/divide (Phase 2): explains *why*, not just that it's forbidden
  - `#20C * 2` → `"Cannot multiply a temperature"` / hint: `"temperature scales have arbitrary zero points, so multiplication is undefined — use addition instead: #20C + #20C"`
- [ ] Unknown unit suffix: fuzzy-match suggestion when possible
  - `#5meter` → `"Unknown unit suffix 'meter'"` / hint: `"did you mean 'm'? — unit suffixes are abbreviations: m, cm, km, in, ft, etc."`
- [ ] Zero denominator in fraction literal
  - `#1/0in` → `"Fraction denominator cannot be zero"`
- [ ] Overflow: explains the practical limit
  - `"Unit value overflow"` / hint: `"the maximum representable distance is approximately 77 AU"`
- [ ] Negative mixed number clarity
  - Malformed `#2+-3/8in` → parse error with hint: `"for negative mixed numbers, write #-2+3/8in — the sign applies to the whole value"`

### Phase 2 — Temperature + Volume

- [ ] Temperature: `K`, `C`, `F` suffixes
- [ ] Temperature stored in unified K × 900 base (1°C = 900 sub-K, 1°F = 500 sub-K)
- [ ] Temperature add/subtract allowed (`#20C + #10C = #30C`)
- [ ] Temperature multiply/divide are errors (offset scale inconsistency)
- [ ] `#0C == #32F` → true
- [ ] Volume SI: `mL`, `L`
- [ ] Volume US: `floz`, `cup`, `pt`, `qt`, `gal`
- [ ] Volume cross-system bridge: 1 gal = 3.785411784 L (exact)
- [ ] Named constructors (plural): `celsius()`, `fahrenheit()`, `kelvins()`, `litres()`/`liters()`, `gallons()`, etc.

### Phase 3 — Area

- [ ] Area suffixes: `mm2`, `cm2`, `m2`, `km2`, `in2`, `ft2`, `yd2`, `ac`, `mi2`
- [ ] Area entered as own family (not derived from length × length)
- [ ] Area cross-system bridge: 1 in² = 0.00064516 m² (exact)

### Phase 4 — Polish

- [ ] Compound display formatting (`5' 3+1/4"`, `2lb 5oz`)
- [ ] Schema integration for unit types
- [ ] Derived unit arithmetic (`#5m * #3m` → area) if demanded
- [ ] Additional families (angle, energy) if demanded

## Design Decisions

- **HCN = 725,760 for US Customary storage**: Not a "nice highly-composite number" but the precise LCM of divisibility requirements (36 × 64, 36 × 9, 36 × 7, 36 × 5). Yields 20,160 sub-units per inch — every fraction from halves to sixty-fourths, plus thirds, fifths, sevenths, and ninths is an exact integer. Precision: ~1.26 µm per sub-yard, matching SI micrometre precision. See `DESIGN-units-v3.md` §3.2 for the full derivation.

- **K × 900 for temperature storage**: The Celsius-to-Fahrenheit ratio is 9/5. With 900 sub-units per kelvin, 1°C = 900 sub-K and 1°F = 500 sub-K — both exact integers. All temperature conversions are integer arithmetic with no rounding.

- **Temperature multiply/divide are errors**: Celsius and Fahrenheit are offset scales (arbitrary zero points). `#20C == #68F` but `#20C * 2 = #40C` while `#68F * 2 = #136F`, and `#40C ≠ #136F`. There is no algebraically consistent definition for temperature multiplication. Addition and subtraction are allowed under "treat like numbers" semantics.

- **Left side wins**: When combining units from the same family, the left operand determines the display hint and measurement system. `#1cm + #1in` → `#3.54cm`; `#1in + #1cm` → `~#1+3/8in`.

- **SI fractions are syntactic sugar**: `#1/3m` converts to 333,333 µm (truncated to whole sub-units). US Customary fractions like `#1/3cup` are stored exactly. This reflects the different numeric idioms of each system.

- **`+` as mixed-number separator**: `#92+5/8in` rather than `#92-5/8in` to avoid ambiguity with subtraction across locales.

- **`#` sigil for unit literals**: Unambiguous with Money's `CODE#amount` pattern. Lexer distinguishes by what precedes `#` (uppercase letters → Money, digit/sign follows → unit).

- **Byte as data base unit**: The target domain is file sizes and storage, not network throughput. `kB`/`MB`/`GB` use SI powers of 10; `KiB`/`MiB`/`GiB` use binary powers of 2.

- **Plural constructor names**: All constructors use plural forms — `metres()`, `feet()`, `inches()`, `bytes()`, `kilograms()` — because they read naturally at every value except 1 (e.g., `feet(12)` reads as "12 feet"). Singular forms like `foot(12)` read like compound adjectives ("a 12-foot board") rather than measurement expressions. No short-form constructors (`m()`, `g()`) — collision risk with variable names.

- **Fixed sub-unit base for SI**: All SI values are stored as an integer count of the smallest practical sub-unit (µm for length, mg for mass, B for data). This gives the same architecture as Imperial (int64 counting sub-yards) and Temperature (int64 counting sub-kelvins) — one int64, no scale factor, no promotion needed for within-system arithmetic. Precision is µm-level, matching Imperial's ~1.26 µm resolution.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components

| Component | Location | Change Type | Notes |
|-----------|----------|-------------|-------|
| Lexer | `pkg/parsley/lexer/lexer.go` | Modify | Unit literal tokenisation: `#` + number + suffix, fractions, mixed numbers |
| Token types | `pkg/parsley/lexer/token.go` | Add | New token types for unit literals |
| AST | `pkg/parsley/ast/ast.go` | Add | `UnitLiteral`, `FractionUnitLiteral`, `MixedUnitLiteral` nodes |
| Parser | `pkg/parsley/parser/parser.go` | Modify | Parse unit literal tokens into AST nodes |
| Evaluator | `pkg/parsley/evaluator/` | Add/Modify | `SIUnit`, `ImperialUnit`, `TempUnit` objects; arithmetic; conversion |
| Object system | `pkg/parsley/evaluator/object.go` | Add | Unit object types implementing the Object interface |
| Builtins | `pkg/parsley/evaluator/builtins.go` | Add | Named constructors (`metres()`, `inches()`, etc.), `unit()` constructor |
| Unit tables | `pkg/parsley/evaluator/unit_tables.go` | Add | Sub-unit bases, suffix→family maps, within-system ratios, cross-system bridges |
| Tests | `pkg/parsley/tests/` | Add | Unit literal, arithmetic, conversion, display, error tests |

### Dependencies
- Depends on: None (self-contained; Money precedent already established)
- Blocks: None

### Key Implementation Details

**Lexer strategy:** After seeing `#` followed by a digit or `-`, enter "unit literal mode". Consume the numeric part (integer, decimal, fraction `n/d`, or mixed `W+n/d`), then consume the longest-matching unit suffix. This mirrors Money's lexing but with a different trigger pattern.

**Suffix disambiguation:** `m` vs `m2`, `mi` vs `mm` — use longest-match. `kB` vs `KiB` — case-sensitive matching. `B` alone is byte.

**SI storage:** All values stored as integer count of sub-units. `#12.3m` = 12,300,000 µm. `#5cm` = 50,000 µm. `#3mm` = 3,000 µm. Arithmetic is direct integer addition: `#5cm + #3mm` = 50,000 + 3,000 = 53,000 µm. No scale promotion needed.

**US Customary storage:** All values stored as `Amount / HCN` in the base unit (yard for length, ounce for mass). To store `#3/8in`: one inch = HCN/36 = 20,160 sub-yards. 3/8 of an inch = 3 × 20,160 / 8 = 7,560. Amount = 7,560. Exact integer.

**Cross-system bridge implementation:** 1 in = 0.0254 m. In SI storage: 1 in = 25,400 µm. In US storage: 1 in = 20,160 sub-yards. Bridge ratio: 25,400 µm / 20,160 sub-yards (or use the exact ratio).

**GCD fraction display:** `GCD(amount, HCN)` reduces to the display fraction. If the reduced denominator is in the "common" set (2,3,4,5,6,7,8,10,12,16,32,64), display as fraction. Otherwise fall back to decimal.

### Edge Cases & Constraints

1. **Overflow** — SI: int64 max µm ≈ 9.2 × 10¹² m ≈ 61 AU. US: int64 max / 725,760 ≈ 77 AU. Overflow should be detected and produce a clear error, not silent wraparound.
2. **SI fraction truncation** — `#1/3m` truncates to 333,333 µm (not exactly 1/3 metre). Document this.
3. **Cross-system rounding** — SI → Imperial may round. Imperial → SI is often exact (bridge ratios are finite SI decimals). The max error is ~1.26 µm (one Imperial sub-unit).
4. **Uncommon fraction display** — If GCD reduction yields a denominator not in the common set, fall back to decimal rather than displaying `#17/315in`.
5. **Negative mixed numbers** — `#-2+3/8in` means −(2+3/8), not (−2)+(3/8). The sign applies to the whole value.
6. **Zero values** — `#0m`, `#0in`, `#0C` are all valid. `#0m == #0in` → true (same family, both zero).
7. **Temperature at absolute zero** — `#-273.15C`, `#-459.67F`, `#0K` are all the same value and all valid.

## Implementation Notes
*Added during/after implementation*

## Related
- Design doc: `work/design/DESIGN-units-v3.md` (authoritative, specification-ready)
- Predecessors: `work/design/DESIGN-units.md` (exploration), `work/design/DESIGN-units-v2.md` (decision memo)
- Plan: `work/plans/FEAT-118-plan.md` (to be created)