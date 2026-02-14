// Package evaluator provides unit arithmetic operations for FEAT-118.
// This file implements unit+unit, unit-unit, unit*scalar, scalar*unit,
// unit/scalar, unit/unit (ratio), and comparison operators.
// Temperature arithmetic uses offset-based formulas (Phase 2).
// All non-temperature arithmetic is scale-aware (Phase 3).
package evaluator

import (
	"math"

	"github.com/sambeau/basil/pkg/parsley/lexer"
)

// evalUnitInfixExpression handles unit OP unit operations.
func evalUnitInfixExpression(tok lexer.Token, operator string, left, right *Unit) Object {
	// Check family compatibility for all operations except ==, !=
	if left.Family != right.Family {
		switch operator {
		case "==", "!=":
			// Different families are never equal
			if operator == "==" {
				return FALSE
			}
			return TRUE
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
		// unit * unit is an error (derived units deferred)
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
