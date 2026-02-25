// Package evaluator provides unit arithmetic operations for FEAT-118.
// This file implements unit+unit, unit-unit, unit*scalar, scalar*unit,
// unit/scalar, unit/unit (ratio), and comparison operators.
// Temperature arithmetic uses offset-based formulas (Phase 2).
// All non-temperature arithmetic is scale-aware (Phase 3).
// Derived unit arithmetic: length × length → area (Phase 4).
package evaluator

import (
	"math"

	"github.com/sambeau/basil/pkg/parsley/lexer"
)

// evalDerivedUnitMultiply handles unit × unit for derived units.
// Currently only supports length × length → area.
// Returns nil if the multiplication is not a supported derived operation.
func evalDerivedUnitMultiply(tok lexer.Token, left, right *Unit) Object {
	// Only length × length → area is supported
	if left.Family != FamilyLength || right.Family != FamilyLength {
		return nil
	}

	// Cross-system multiplication is not allowed
	if left.System != right.System {
		return newOperatorErrorWithPos(tok, "UNIT-0014", map[string]any{
			"LeftSystem":  left.System,
			"RightSystem": right.System,
		})
	}

	return multiplyLengthToArea(left, right)
}

// multiplyLengthToArea computes length × length → area.
// Both operands must be the same system (SI or US).
// The left operand's display hint determines the result's display hint.
func multiplyLengthToArea(left, right *Unit) Object {
	// Cross-system multiplication is not allowed
	if left.System != right.System {
		return &Error{
			Class:   ClassOperator,
			Code:    "UNIT-0014",
			Message: "Cannot multiply " + left.System + " length by " + right.System + " length",
			Hints:   []string{"convert both to the same system first — use .to(\"m\") or .to(\"ft\")"},
		}
	}

	if left.System == SystemSI {
		return multiplyLengthToAreaSI(left, right)
	}
	return multiplyLengthToAreaUS(left, right)
}

// multiplyLengthToAreaSI computes SI length × SI length → SI area.
// SI length is stored in µm, SI area is stored in mm².
// 1 mm² = 1,000,000 µm², so we divide by 1,000,000.
func multiplyLengthToAreaSI(left, right *Unit) *Unit {
	// Convert both to µm (base SI length sub-unit)
	// Amount is already in µm
	leftAmount := left.Amount
	leftScale := left.Scale
	rightAmount := right.Amount
	rightScale := right.Scale

	// Multiply: result in µm²
	// Then convert to mm²: divide by 1,000,000
	// Use float64 for the intermediate calculation to handle large values
	var resultAmount int64
	var resultScale int

	if leftScale == 0 && rightScale == 0 {
		// Fast path: both scale 0
		// result_mm2 = (left_µm × right_µm) / 1,000,000
		leftF := float64(leftAmount)
		rightF := float64(rightAmount)
		resultF := leftF * rightF / 1_000_000

		if resultF >= math.MinInt64 && resultF <= math.MaxInt64 {
			resultAmount = int64(math.Round(resultF))
			resultScale = 0
		} else {
			// Overflow: use scale
			resultAmount, resultScale = scaleAmountForSuffixFloat(resultF/1_000_000, 1)
			resultScale += 6 // Account for the division we didn't do
		}
	} else {
		// General case with scale
		leftTrue := scaleTrueValue(leftAmount, leftScale)
		rightTrue := scaleTrueValue(rightAmount, rightScale)
		resultF := leftTrue * rightTrue / 1_000_000

		if resultF >= math.MinInt64 && resultF <= math.MaxInt64 {
			resultAmount = int64(math.Round(resultF))
			resultScale = 0
		} else {
			// Use scale for large values
			resultAmount, resultScale = scaleAmountForSuffixFloat(resultF/1e12, 1)
			resultScale += 12
		}
	}

	// Determine display hint based on left operand
	displayHint := lengthToAreaDisplayHintSI(left.DisplayHint)

	return &Unit{
		Amount:      resultAmount,
		Family:      FamilyArea,
		System:      SystemSI,
		DisplayHint: displayHint,
		Scale:       resultScale,
	}
}

// multiplyLengthToAreaUS computes US length × US length → US area.
// US length is stored in sub-yards (HCN per yard), US area is stored in in².
// 1 yard = 36 inches, so 1 yd² = 1296 in².
// 1 inch = HCN/36 sub-yards, so 1 in² = (HCN/36)² sub-yards².
func multiplyLengthToAreaUS(left, right *Unit) *Unit {
	// Convert both operands to inches first
	// sub-yards per inch = HCN/36 = 20,160
	subYardsPerInch := float64(HCN / 36)

	var leftInches, rightInches float64
	if left.Scale > 0 {
		leftInches = scaleTrueValue(left.Amount, left.Scale) / subYardsPerInch
	} else {
		leftInches = float64(left.Amount) / subYardsPerInch
	}
	if right.Scale > 0 {
		rightInches = scaleTrueValue(right.Amount, right.Scale) / subYardsPerInch
	} else {
		rightInches = float64(right.Amount) / subYardsPerInch
	}

	// Result in square inches (US area base unit is in²)
	resultInches2 := leftInches * rightInches

	// Determine display hint based on left operand
	displayHint := lengthToAreaDisplayHintUS(left.DisplayHint)

	// Convert to display-unit sub-units for proper fraction display
	subPerDisplayUnit := USSubUnitsPerUnit(displayHint)
	if subPerDisplayUnit == 0 {
		subPerDisplayUnit = 1
	}

	// For area, we store the raw value in the base unit (in²)
	// The display function will handle conversion to the display hint
	var resultAmount int64
	var resultScale int

	if resultInches2 >= math.MinInt64 && resultInches2 <= math.MaxInt64 {
		resultAmount = int64(math.Round(resultInches2))
		resultScale = 0
	} else {
		// Use scale for large values
		resultAmount, resultScale = scaleAmountForSuffixFloat(resultInches2/1e12, 1)
		resultScale += 12
	}

	return &Unit{
		Amount:      resultAmount,
		Family:      FamilyArea,
		System:      SystemUS,
		DisplayHint: displayHint,
		Scale:       resultScale,
	}
}

// lengthToAreaDisplayHintSI maps a length display hint to an area display hint.
func lengthToAreaDisplayHintSI(lengthHint string) string {
	switch lengthHint {
	case "mm":
		return "mm2"
	case "cm":
		return "cm2"
	case "m":
		return "m2"
	case "km":
		return "km2"
	default:
		return "m2" // fallback
	}
}

// lengthToAreaDisplayHintUS maps a length display hint to an area display hint.
func lengthToAreaDisplayHintUS(lengthHint string) string {
	switch lengthHint {
	case "in":
		return "in2"
	case "ft":
		return "ft2"
	case "yd":
		return "yd2"
	case "mi":
		return "mi2"
	default:
		return "ft2" // fallback
	}
}

// evalDerivedUnitDivide handles area / length → length.
// Returns nil if the division is not a supported derived operation.
func evalDerivedUnitDivide(tok lexer.Token, left, right *Unit) Object {
	// Only area / length → length is supported
	if left.Family != FamilyArea || right.Family != FamilyLength {
		return nil
	}

	// Cross-system division is not allowed
	if left.System != right.System {
		return newOperatorErrorWithPos(tok, "UNIT-0015", map[string]any{
			"LeftSystem":  left.System,
			"RightSystem": right.System,
		})
	}

	return divideAreaByLength(left, right)
}

// divideAreaByLength computes area / length → length.
// Both operands must be the same system (SI or US).
// The right operand's (divisor's) display hint determines the result's display hint.
func divideAreaByLength(area, length *Unit) Object {
	if area.System == SystemSI {
		return divideAreaByLengthSI(area, length)
	}
	return divideAreaByLengthUS(area, length)
}

// divideAreaByLengthSI computes SI area / SI length → SI length.
// SI area is stored in mm², SI length is stored in µm.
// result_µm = area_mm² × 1,000,000 / length_µm × 1000
// (converting mm² to µm², then dividing by µm gives µm)
func divideAreaByLengthSI(area, length *Unit) Object {
	if length.Amount == 0 && length.Scale == 0 {
		return &Error{
			Class:   ClassOperator,
			Code:    "OP-0002",
			Message: "Division by zero",
		}
	}

	// Convert area to µm²: multiply by 1,000,000
	// Then divide by length in µm to get result in µm
	var areaInMicrons2 float64
	if area.Scale > 0 {
		areaInMicrons2 = scaleTrueValue(area.Amount, area.Scale) * 1_000_000
	} else {
		areaInMicrons2 = float64(area.Amount) * 1_000_000
	}

	var lengthInMicrons float64
	if length.Scale > 0 {
		lengthInMicrons = scaleTrueValue(length.Amount, length.Scale)
	} else {
		lengthInMicrons = float64(length.Amount)
	}

	resultMicrons := areaInMicrons2 / lengthInMicrons

	var resultAmount int64
	var resultScale int

	if resultMicrons >= math.MinInt64 && resultMicrons <= math.MaxInt64 {
		resultAmount = int64(math.Round(resultMicrons))
		resultScale = 0
	} else {
		resultAmount, resultScale = scaleAmountForSuffixFloat(resultMicrons/1e12, 1)
		resultScale += 12
	}

	// Result display hint comes from the divisor (length)
	return &Unit{
		Amount:      resultAmount,
		Family:      FamilyLength,
		System:      SystemSI,
		DisplayHint: length.DisplayHint,
		Scale:       resultScale,
	}
}

// divideAreaByLengthUS computes US area / US length → US length.
// US area is stored in in², US length is stored in sub-yards.
// 1 inch = HCN/36 sub-yards
func divideAreaByLengthUS(area, length *Unit) Object {
	if length.Amount == 0 && length.Scale == 0 {
		return &Error{
			Class:   ClassOperator,
			Code:    "OP-0002",
			Message: "Division by zero",
		}
	}

	subYardsPerInch := float64(HCN / 36)

	// Convert length to inches
	var lengthInches float64
	if length.Scale > 0 {
		lengthInches = scaleTrueValue(length.Amount, length.Scale) / subYardsPerInch
	} else {
		lengthInches = float64(length.Amount) / subYardsPerInch
	}

	// Get area in square inches
	var areaInches2 float64
	if area.Scale > 0 {
		areaInches2 = scaleTrueValue(area.Amount, area.Scale)
	} else {
		areaInches2 = float64(area.Amount)
	}

	// Divide: result in inches
	resultInches := areaInches2 / lengthInches

	// Convert back to sub-yards
	resultSubYards := resultInches * subYardsPerInch

	var resultAmount int64
	var resultScale int

	if resultSubYards >= math.MinInt64 && resultSubYards <= math.MaxInt64 {
		resultAmount = int64(math.Round(resultSubYards))
		resultScale = 0
	} else {
		resultAmount, resultScale = scaleAmountForSuffixFloat(resultSubYards/1e12, 1)
		resultScale += 12
	}

	// Result display hint comes from the divisor (length)
	return &Unit{
		Amount:      resultAmount,
		Family:      FamilyLength,
		System:      SystemUS,
		DisplayHint: length.DisplayHint,
		Scale:       resultScale,
	}
}

// evalUnitInfixExpression handles unit OP unit operations.
func evalUnitInfixExpression(tok lexer.Token, operator string, left, right *Unit) Object {
	// Check for derived unit operations (different families)
	if left.Family != right.Family {
		switch operator {
		case "==", "!=":
			// Different families are never equal
			if operator == "==" {
				return FALSE
			}
			return TRUE
		case "*":
			// Try derived multiplication (e.g., length × length → area)
			if result := evalDerivedUnitMultiply(tok, left, right); result != nil {
				return result
			}
			return newOperatorErrorWithPos(tok, "UNIT-0001", map[string]any{
				"LeftFamily":  left.Family,
				"RightFamily": right.Family,
				"Operator":    operator,
			})
		case "/":
			// Try derived division (e.g., area / length → length)
			if result := evalDerivedUnitDivide(tok, left, right); result != nil {
				return result
			}
			return newOperatorErrorWithPos(tok, "UNIT-0001", map[string]any{
				"LeftFamily":  left.Family,
				"RightFamily": right.Family,
				"Operator":    operator,
			})
		default:
			return newOperatorErrorWithPos(tok, "UNIT-0001", map[string]any{
				"LeftFamily":  left.Family,
				"RightFamily": right.Family,
				"Operator":    operator,
			})
		}
	}

	// Temperature has special arithmetic rules
	if IsTemperatureFamily(left.Family) {
		return evalTempInfixExpression(tok, operator, left, right)
	}

	// Normalise the right operand into the left's system if they differ
	leftAmount := left.Amount
	leftScale := left.Scale
	rightAmount := right.Amount
	rightScale := right.Scale

	if left.System != right.System {
		// Cross-system: convert right to left's system
		if left.System == SystemSI {
			// Right is US, convert to SI
			rightAmount, rightScale = ConvertUSToSIScaled(right.Amount, right.Scale, right.Family)
		} else {
			// Right is SI, convert to US
			rightAmount, rightScale = ConvertSIToUSScaled(right.Amount, right.Scale, right.Family)
		}
	}

	switch operator {
	case "+":
		resAmount, resScale := scaleAdd(leftAmount, leftScale, rightAmount, rightScale)
		return &Unit{
			Amount:      resAmount,
			Family:      left.Family,
			System:      left.System,
			DisplayHint: left.DisplayHint,
			Scale:       resScale,
		}
	case "-":
		resAmount, resScale := scaleSub(leftAmount, leftScale, rightAmount, rightScale)
		return &Unit{
			Amount:      resAmount,
			Family:      left.Family,
			System:      left.System,
			DisplayHint: left.DisplayHint,
			Scale:       resScale,
		}
	case "/":
		// unit / unit = dimensionless ratio (plain number)
		if rightAmount == 0 && rightScale == 0 {
			return newOperatorErrorWithPos(tok, "OP-0002", map[string]any{})
		}
		ratio := scaleRatio(leftAmount, leftScale, rightAmount, rightScale)
		// If it's an exact integer, return Integer
		if ratio == math.Trunc(ratio) && ratio >= math.MinInt64 && ratio <= math.MaxInt64 {
			return &Integer{Value: int64(ratio)}
		}
		return &Float{Value: ratio}
	case "*":
		// Same-family multiplication: only length × length → area is supported
		if left.Family == FamilyLength {
			// Same system check happens in multiplyLengthToArea
			return multiplyLengthToArea(left, right)
		}
		return newOperatorErrorWithPos(tok, "UNIT-0003", map[string]any{})
	case "<":
		return nativeBoolToParsBoolean(scaleCmp(leftAmount, leftScale, rightAmount, rightScale) < 0)
	case ">":
		return nativeBoolToParsBoolean(scaleCmp(leftAmount, leftScale, rightAmount, rightScale) > 0)
	case "<=":
		return nativeBoolToParsBoolean(scaleCmp(leftAmount, leftScale, rightAmount, rightScale) <= 0)
	case ">=":
		return nativeBoolToParsBoolean(scaleCmp(leftAmount, leftScale, rightAmount, rightScale) >= 0)
	case "==":
		return nativeBoolToParsBoolean(scaleCmp(leftAmount, leftScale, rightAmount, rightScale) == 0)
	case "!=":
		return nativeBoolToParsBoolean(scaleCmp(leftAmount, leftScale, rightAmount, rightScale) != 0)
	default:
		return newOperatorErrorWithPos(tok, "UNIT-0004", map[string]any{
			"Operator": operator,
		})
	}
}

// evalTempInfixExpression handles temperature-specific arithmetic.
// Add/subtract use offset formulas; multiply/divide between temperatures is forbidden.
// Comparisons work directly on sub-kelvins (absolute temperature).
func evalTempInfixExpression(tok lexer.Token, operator string, left, right *Unit) Object {
	// Temperature comparisons work directly on sub-kelvins
	switch operator {
	case "==":
		return nativeBoolToParsBoolean(left.Amount == right.Amount)
	case "!=":
		return nativeBoolToParsBoolean(left.Amount != right.Amount)
	case "<":
		return nativeBoolToParsBoolean(left.Amount < right.Amount)
	case ">":
		return nativeBoolToParsBoolean(left.Amount > right.Amount)
	case "<=":
		return nativeBoolToParsBoolean(left.Amount <= right.Amount)
	case ">=":
		return nativeBoolToParsBoolean(left.Amount >= right.Amount)
	case "*":
		return newOperatorErrorWithPos(tok, "UNIT-0003", map[string]any{})
	case "/":
		// temperature / temperature is forbidden
		return newOperatorErrorWithPos(tok, "UNIT-0013", map[string]any{})
	}

	// Addition and subtraction use offset-based arithmetic.
	// Formula (left side wins for display hint):
	//   add: result = left.Amount + right.Amount - TempOffset(left.DisplayHint)
	//   sub: result = left.Amount - right.Amount + TempOffset(left.DisplayHint)
	offset := TempOffset(left.DisplayHint)

	switch operator {
	case "+":
		return &Unit{
			Amount:      left.Amount + right.Amount - offset,
			Family:      FamilyTemperature,
			System:      left.System,
			DisplayHint: left.DisplayHint,
		}
	case "-":
		return &Unit{
			Amount:      left.Amount - right.Amount + offset,
			Family:      FamilyTemperature,
			System:      left.System,
			DisplayHint: left.DisplayHint,
		}
	default:
		return newOperatorErrorWithPos(tok, "UNIT-0004", map[string]any{
			"Operator": operator,
		})
	}
}

// evalUnitScalarExpression handles unit * scalar and unit / scalar.
func evalUnitScalarExpression(tok lexer.Token, operator string, unit *Unit, scalar float64) Object {
	// Temperature cannot be multiplied or divided by scalars
	if IsTemperatureFamily(unit.Family) {
		switch operator {
		case "*":
			return newOperatorErrorWithPos(tok, "UNIT-0011", map[string]any{})
		case "/":
			return newOperatorErrorWithPos(tok, "UNIT-0012", map[string]any{})
		}
	}

	switch operator {
	case "*":
		resAmount, resScale := scaleMulScalar(unit.Amount, unit.Scale, scalar)
		return &Unit{
			Amount:      resAmount,
			Family:      unit.Family,
			System:      unit.System,
			DisplayHint: unit.DisplayHint,
			Scale:       resScale,
		}
	case "/":
		if scalar == 0 {
			return newOperatorErrorWithPos(tok, "OP-0002", map[string]any{})
		}
		resAmount, resScale := scaleDivScalar(unit.Amount, unit.Scale, scalar)
		return &Unit{
			Amount:      resAmount,
			Family:      unit.Family,
			System:      unit.System,
			DisplayHint: unit.DisplayHint,
			Scale:       resScale,
		}
	case "+":
		// scalar + unit error: no implicit promotion
		return newOperatorErrorWithPos(tok, "UNIT-0002", map[string]any{
			"Operator": "+",
			"Left":     "unit",
			"Right":    "number",
		})
	case "-":
		return newOperatorErrorWithPos(tok, "UNIT-0002", map[string]any{
			"Operator": "-",
			"Left":     "unit",
			"Right":    "number",
		})
	default:
		return newOperatorErrorWithPos(tok, "UNIT-0004", map[string]any{
			"Operator": operator,
		})
	}
}

// evalScalarUnitExpression handles scalar * unit (commutative) and errors for scalar+unit, scalar-unit, scalar/unit.
func evalScalarUnitExpression(tok lexer.Token, operator string, scalar float64, unit *Unit) Object {
	// Temperature cannot be multiplied by scalars
	if IsTemperatureFamily(unit.Family) && operator == "*" {
		return newOperatorErrorWithPos(tok, "UNIT-0011", map[string]any{})
	}

	switch operator {
	case "*":
		// Multiplication is commutative
		resAmount, resScale := scaleMulScalar(unit.Amount, unit.Scale, scalar)
		return &Unit{
			Amount:      resAmount,
			Family:      unit.Family,
			System:      unit.System,
			DisplayHint: unit.DisplayHint,
			Scale:       resScale,
		}
	case "+":
		// number + unit is an error
		return newOperatorErrorWithPos(tok, "UNIT-0002", map[string]any{
			"Operator": "+",
			"Left":     "number",
			"Right":    "unit",
		})
	case "-":
		// number - unit is an error
		return newOperatorErrorWithPos(tok, "UNIT-0002", map[string]any{
			"Operator": "-",
			"Left":     "number",
			"Right":    "unit",
		})
	case "/":
		// number / unit is an error
		return newOperatorErrorWithPos(tok, "UNIT-0005", map[string]any{})
	default:
		return newOperatorErrorWithPos(tok, "UNIT-0004", map[string]any{
			"Operator": operator,
		})
	}
}
