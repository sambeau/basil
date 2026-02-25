---
id: PLAN-099
feature: FEAT-118
title: "Implementation Plan for Measurement Units — Phase 4 (Polish)"
status: complete
created: 2025-01-14
---

# Implementation Plan: FEAT-118 Phase 4 — Polish

## Overview

Phase 4 completes the Measurement Units feature with three polish items:

1. **Derived Unit Arithmetic** — `#5m * #3m` → `#15m2` (length × length → area)
2. **Schema Integration** — Unit types in schemas: `weight: mass`, `height: unit("cm")`
3. **Compound Display Formatting** — `#63.375in.format("ft-in")` → `5' 3+3/8"`

These are "nice to have" features that enhance usability but aren't blocking. Each can be implemented independently.

**Design doc:** `work/design/DESIGN-units-v3.md` §10 (Phase 4), `work/design/DESIGN-units.md` §6.3, §10.5, §11

## Prerequisites

- [x] Phase 1 complete (Length, Mass, Data)
- [x] Phase 2 complete (Temperature, Volume)
- [x] Phase 3 complete (Area, Scale infrastructure)
- [ ] FEAT-118 merged to main

---

## Task 1: Derived Unit Arithmetic (Length × Length → Area)

**Goal:** Allow `#5m * #3m` to produce `#15m2` instead of an error.

### 1.1 Design Decisions

**Scope:** Only length × length → area in Phase 4. Other derived units (speed = length/time, volume = length³) are deferred.

**System rules:**
- SI × SI → SI area: `#5m * #3m` → `#15m2`
- US × US → US area: `#5ft * #3ft` → `#15ft2`
- SI × US or US × SI → Error (no implicit cross-system multiplication)

**Display hint mapping:**
| Left | Right | Result |
|------|-------|--------|
| mm × mm | → | mm2 |
| mm × cm | → | mm2 (left wins, but scaled) |
| cm × cm | → | cm2 |
| cm × m | → | cm2 |
| m × m | → | m2 |
| m × km | → | m2 |
| km × km | → | km2 |
| in × in | → | in2 |
| ft × ft | → | ft2 |
| yd × yd | → | yd2 |
| mi × mi | → | mi2 |

**"Left wins" rule:** The left operand determines the display hint for the result. `#5m * #300cm` → `#15m2` (not cm2).

**Reverse operation (area / length → length):**
- `#15m2 / #3m` → `#5m`
- Only valid when dividing area by length of same system
- Result display hint comes from the divisor

### 1.2 Implementation

**Files:** `pkg/parsley/evaluator/eval_unit_infix.go`

**Changes:**

1. In `evalUnitInfixExpression`, modify the `*` case for unit × unit:
   ```go
   case "*":
       if right.Type() == UNIT_OBJ {
           rightUnit := right.(*Unit)
           // Check if both are length family
           if left.Family == "length" && rightUnit.Family == "length" {
               // Check same system
               if left.System != rightUnit.System {
                   return newError("Cannot multiply %s length by %s length — convert to same system first",
                       left.System, rightUnit.System)
               }
               return multiplyLengthToArea(left, rightUnit)
           }
           // Existing error for other unit × unit
           return newError("Cannot multiply unit by unit...")
       }
   ```

2. Add `multiplyLengthToArea(left, right *Unit) Object`:
   - Convert both to base sub-units (µm for SI, sub-yards for US)
   - Multiply: `result = left_base × right_base`
   - Convert to area sub-units (mm² for SI, in² for US)
   - Apply Scale if overflow
   - Set display hint based on left operand's suffix

3. In the `/` case, add area / length → length:
   ```go
   if left.Family == "area" && rightUnit.Family == "length" {
       if left.System != rightUnit.System {
           return newError("Cannot divide %s area by %s length", left.System, rightUnit.System)
       }
       return divideAreaByLength(left, rightUnit)
   }
   ```

**Validation:**
- `#5m * #3m` → `#15m2`
- `#2ft * #3ft` → `#6ft2`
- `#100cm * #50cm` → `#5000cm2`
- `#5m * #3ft` → Error (cross-system)
- `#15m2 / #3m` → `#5m`
- `#15m2 / #3ft` → Error (cross-system)

### 1.3 Tests

**File:** `pkg/parsley/tests/unit_derived_test.go` (new file)

```go
func TestLengthTimesLength(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {`#5m * #3m`, `#15m2`},
        {`#2ft * #3ft`, `#6ft2`},
        {`#100cm * #50cm`, `#5000cm2`},
        {`#1km * #1km`, `#1km2`},
        {`#12in * #12in`, `#144in2`},  // = 1ft2
        {`#5m * #300cm`, `#15m2`},     // left wins display
    }
    // ...
}

func TestLengthTimesLengthCrossSystemError(t *testing.T) {
    // #5m * #3ft should error
}

func TestAreaDividedByLength(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {`#15m2 / #3m`, `#5m`},
        {`#144in2 / #12in`, `#12in`},
        {`#1km2 / #1000m`, `#1000m`},  // divisor determines display
    }
    // ...
}
```

---

## Task 2: Schema Integration for Unit Types

**Goal:** Allow schemas to declare fields as unit types with optional constraints.

### 2.1 Design Decisions

**Syntax options:**

```parsley
// Option A: Family name as type
@schema Product {
    weight: mass,           // any mass unit
    height: length,         // any length unit
    storage: data           // any data unit
}

// Option B: unit() type constructor
@schema Product {
    weight: unit("mass"),      // any mass unit
    height: unit("length"),    // any length unit
    width: unit("cm"),         // specific unit required
}

// Option C: Both (family names are shortcuts)
@schema Product {
    weight: mass,              // shortcut for unit("mass")
    height: unit("cm"),        // specific unit
}
```

**Recommendation:** Option C — family names (`mass`, `length`, `data`, `temperature`, `volume`, `area`) are recognized as schema types. For specific unit constraints, use `unit("suffix")`.

**Validation behavior:**
- `mass` type accepts any mass unit (`#5kg`, `#2.2lb`, `#500g`)
- `unit("kg")` type accepts only kg-denominated values, auto-converts others
- Validation on record creation, not on field access

**Storage:** Units are stored as-is (no normalization). The schema type is for validation, not storage conversion.

### 2.2 Implementation

**Files:**
- `pkg/parsley/evaluator/stdlib_dsl_schema.go` — Add unit type recognition
- `pkg/parsley/evaluator/record.go` — Add unit validation in CreateRecord

**Changes:**

1. Add unit families to recognized schema types:
   ```go
   var unitFamilyTypes = map[string]bool{
       "mass": true, "length": true, "data": true,
       "temperature": true, "volume": true, "area": true,
   }
   ```

2. In schema field parsing, recognize `unit("...")` syntax:
   ```go
   // In parseSchemaFieldType or equivalent
   if strings.HasPrefix(typeName, "unit(") {
       // Extract the unit constraint: "mass", "length", "kg", "cm", etc.
       constraint := extractUnitConstraint(typeName)
       return &DSLSchemaField{
           Type: "unit",
           UnitConstraint: constraint,  // new field
       }
   }
   if unitFamilyTypes[typeName] {
       return &DSLSchemaField{
           Type: "unit",
           UnitConstraint: typeName,  // family name
       }
   }
   ```

3. In `CreateRecord`, validate unit fields:
   ```go
   case "unit":
       unit, ok := value.(*Unit)
       if !ok {
           return newError("field %s expects a unit value, got %s", fieldName, value.Type())
       }
       constraint := field.UnitConstraint
       if unitFamilyTypes[constraint] {
           // Family constraint: check family matches
           if unit.Family != constraint {
               return newError("field %s expects %s unit, got %s", fieldName, constraint, unit.Family)
           }
       } else {
           // Specific unit constraint: check suffix matches
           if unit.DisplayHint != constraint {
               // Auto-convert? Or error?
               // For now: error
               return newError("field %s expects %s unit, got %s", fieldName, constraint, unit.DisplayHint)
           }
       }
   ```

### 2.3 Tests

**File:** `pkg/parsley/tests/schema_unit_test.go` (new file)

```go
func TestSchemaUnitFamilyType(t *testing.T) {
    input := `
        @schema Product {
            weight: mass,
            height: length
        }
        Product({weight: #5kg, height: #1.8m})
    `
    // Should succeed
}

func TestSchemaUnitFamilyTypeWrongFamily(t *testing.T) {
    input := `
        @schema Product { weight: mass }
        Product({weight: #5m})  // length, not mass
    `
    // Should error: "field weight expects mass unit, got length"
}

func TestSchemaUnitSpecificType(t *testing.T) {
    input := `
        @schema Metric {
            height: unit("cm")
        }
        Metric({height: #180cm})
    `
    // Should succeed
}

func TestSchemaUnitSpecificTypeWrongUnit(t *testing.T) {
    input := `
        @schema Metric {
            height: unit("cm")
        }
        Metric({height: #1.8m})  // metres, not cm
    `
    // Should error or auto-convert (decide based on design)
}
```

---

## Task 3: Compound Display Formatting

**Goal:** Format units in compound form like `5' 3+3/8"` or `2lb 5oz`.

### 3.1 Design Decisions

**Supported compound formats:**

| Format String | Input | Output | Notes |
|---------------|-------|--------|-------|
| `"ft-in"` | `#63.375in` | `5' 3+3/8"` | Feet and fractional inches |
| `"ft-in"` | `#1.5ft` | `1' 6"` | |
| `"lb-oz"` | `#37oz` | `2lb 5oz` | Pounds and ounces |
| `"gal-qt-pt"` | `#13pt` | `1gal 2qt 1pt` | Gallons, quarts, pints |
| `"L-mL"` | `#1500mL` | `1L 500mL` | SI compound (less common) |

**Method signature:**
```parsley
#63.375in.format("ft-in")     // "5' 3+3/8\""
#63.375in.format("compound")  // auto-detect: "5' 3+3/8\""
#63.375in.format()            // default: "63+3/8in" (existing behavior)
```

**Auto-detect rules for `"compound"`:**
- Length (US): feet-inches if >= 1ft, else inches only
- Mass (US): pounds-ounces if >= 1lb, else ounces only
- Volume (US): largest fitting unit

**Edge cases:**
- Zero remainder: `#5ft.format("ft-in")` → `5'` (no inches shown)
- Negative values: `#-63.375in.format("ft-in")` → `-5' 3+3/8"`
- SI values: `#1.5m.format("ft-in")` → convert first, then format

### 3.2 Implementation

**Files:**
- `pkg/parsley/evaluator/unit_format.go` (new file)
- `pkg/parsley/evaluator/methods.go` — Extend `.format()` method

**Changes:**

1. Create `unit_format.go` with compound formatting logic:
   ```go
   func formatUnitCompound(u *Unit, formatStr string) string {
       switch formatStr {
       case "ft-in":
           return formatFeetInches(u)
       case "lb-oz":
           return formatPoundsOunces(u)
       case "compound":
           return formatCompoundAuto(u)
       default:
           return formatUnitDefault(u)
       }
   }

   func formatFeetInches(u *Unit) string {
       // Convert to inches (US length base)
       totalInches := convertToInches(u)
       feet := totalInches / 12
       inches := totalInches % 12
       
       // Format feet part
       result := ""
       if feet != 0 {
           result = fmt.Sprintf("%d'", feet)
       }
       
       // Format inches part (with fractions)
       if inches != 0 || feet == 0 {
           inchStr := formatUSFraction(inches, "in")
           if feet != 0 {
               result += " "
           }
           result += inchStr + "\""
       }
       
       return result
   }
   ```

2. In `evalUnitMethod`, extend the `format` case:
   ```go
   case "format":
       if len(args) == 0 {
           return &String{Value: formatUnitDefault(u)}
       }
       formatStr, ok := args[0].(*String)
       if !ok {
           return newTypeError("format", "string", args[0].Type())
       }
       return &String{Value: formatUnitCompound(u, formatStr.Value)}
   ```

### 3.3 Tests

**File:** `pkg/parsley/tests/unit_format_test.go` (new file)

```go
func TestFormatFeetInches(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {`#63in.format("ft-in")`, `5' 3"`},
        {`#63.375in.format("ft-in")`, `5' 3+3/8"`},
        {`#12in.format("ft-in")`, `1'`},
        {`#6in.format("ft-in")`, `6"`},
        {`#0in.format("ft-in")`, `0"`},
        {`#1.5ft.format("ft-in")`, `1' 6"`},
        {`#-63in.format("ft-in")`, `-5' 3"`},
    }
    // ...
}

func TestFormatPoundsOunces(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {`#37oz.format("lb-oz")`, `2lb 5oz`},
        {`#16oz.format("lb-oz")`, `1lb`},
        {`#8oz.format("lb-oz")`, `8oz`},
        {`#2.5lb.format("lb-oz")`, `2lb 8oz`},
    }
    // ...
}

func TestFormatCompoundAuto(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {`#63in.format("compound")`, `5' 3"`},
        {`#6in.format("compound")`, `6in`},  // under 1ft, no compound
        {`#37oz.format("compound")`, `2lb 5oz`},
        {`#8oz.format("compound")`, `8oz`},  // under 1lb, no compound
    }
    // ...
}
```

---

## Task Summary

| Task | Complexity | Dependencies | Priority |
|------|------------|--------------|----------|
| 1. Derived Unit Arithmetic | Medium | None | Medium |
| 2. Schema Integration | Medium | None | Medium |
| 3. Compound Display | Medium | None | Low |

Tasks can be implemented in any order. Each is self-contained.

---

## Validation Checklist

- [x] All existing unit tests still pass
- [x] New tests for derived arithmetic pass
- [x] New tests for schema unit types pass
- [x] New tests for compound formatting pass
- [x] Documentation updated (reference.md, CHEATSHEET.md)
- [x] Error messages follow Parsley conventions (clear, actionable hints)

---

## Progress Log

| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2025-01-14 | Task 1: Derived Unit Arithmetic | Complete | length × length → area, area / length → length implemented in `eval_unit_infix.go`. Cross-system errors (UNIT-0014, UNIT-0015) added. Tests in `unit_derived_test.go`. |
| 2025-01-14 | Task 2: Schema Integration | Complete | Unit family types (`mass`, `length`, etc.) recognized in schemas. `unit(suffix: "kg")` and `unit(family: "mass")` options supported. Validation in `record_validation.go`. Tests in `schema_unit_test.go`. |
| 2025-01-14 | Task 3: Compound Display | Complete | `.format("ft-in")`, `.format("lb-oz")`, `.format("compound")` implemented in `methods_unit.go`. Tests in `unit_format_test.go`. |
| 2025-01-14 | Task 3: Compound Display (extended) | Complete | Added `"gal-qt-pt"` and `"L-mL"` formats. Added cross-system auto-convert (SI→US for `"ft-in"`, US→SI for `"L-mL"`). Volume compound auto-detect in `"compound"`. |
| 2025-01-14 | Documentation | Complete | Updated `reference.md`: derived arithmetic, schema unit types, compound formatting. Updated `CHEATSHEET.md`: all Phase 4 features with examples. |

---

## Deferred Items

- **Speed (length/time)** — Requires time/duration units first
- **Volume from length³** — `#5m * #3m * #2m` → volume
- **Other derived units** — Energy, force, pressure, etc.
- **Angle units** — degrees, radians
- **Unit normalization in schemas** — Auto-convert `#180cm` to `#1.8m` for `unit("m")` fields