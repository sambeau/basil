// Package evaluator provides unit arithmetic operations for FEAT-118.
// This file implements unit+unit, unit-unit, unit*scalar, scalar*unit,
// unit/scalar, unit/unit (ratio), and comparison operators.
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

	// Normalise the right operand into the left's system if they differ
	leftAmount := left.Amount
	rightAmount := right.Amount

	if left.System != right.System {
		// Cross-system: convert right to left's system
		if left.System == SystemSI {
			// Right is US, convert to SI
			rightAmount = ConvertUSToSI(right.Amount, right.Family)
		} else {
			// Right is SI, convert to US
			rightAmount = ConvertSIToUS(right.Amount, right.Family)
		}
	}

	switch operator {
	case "+":
		return &Unit{
			Amount:      leftAmount + rightAmount,
			Family:      left.Family,
			System:      left.System,
			DisplayHint: left.DisplayHint,
		}
	case "-":
		return &Unit{
			Amount:      leftAmount - rightAmount,
			Family:      left.Family,
			System:      left.System,
			DisplayHint: left.DisplayHint,
		}
	case "/":
		// unit / unit = dimensionless ratio (plain number)
		if rightAmount == 0 {
			return newOperatorErrorWithPos(tok, "OP-0002", map[string]any{})
		}
		// Return as float for precision
		ratio := float64(leftAmount) / float64(rightAmount)
		// If it's an exact integer, return Integer
		if ratio == math.Trunc(ratio) && ratio >= math.MinInt64 && ratio <= math.MaxInt64 {
			return &Integer{Value: int64(ratio)}
		}
		return &Float{Value: ratio}
	case "*":
		// unit * unit is an error (derived units deferred)
		return newOperatorErrorWithPos(tok, "UNIT-0003", map[string]any{})
	case "<":
		return nativeBoolToParsBoolean(leftAmount < rightAmount)
	case ">":
		return nativeBoolToParsBoolean(leftAmount > rightAmount)
	case "<=":
		return nativeBoolToParsBoolean(leftAmount <= rightAmount)
	case ">=":
		return nativeBoolToParsBoolean(leftAmount >= rightAmount)
	case "==":
		return nativeBoolToParsBoolean(leftAmount == rightAmount)
	case "!=":
		return nativeBoolToParsBoolean(leftAmount != rightAmount)
	default:
		return newOperatorErrorWithPos(tok, "UNIT-0004", map[string]any{
			"Operator": operator,
		})
	}
}

// evalUnitScalarExpression handles unit * scalar and unit / scalar.
func evalUnitScalarExpression(tok lexer.Token, operator string, unit *Unit, scalar float64) Object {
	switch operator {
	case "*":
		result := float64(unit.Amount) * scalar
		return &Unit{
			Amount:      int64(math.Round(result)),
			Family:      unit.Family,
			System:      unit.System,
			DisplayHint: unit.DisplayHint,
		}
	case "/":
		if scalar == 0 {
			return newOperatorErrorWithPos(tok, "OP-0002", map[string]any{})
		}
		result := float64(unit.Amount) / scalar
		return &Unit{
			Amount:      int64(math.Round(result)),
			Family:      unit.Family,
			System:      unit.System,
			DisplayHint: unit.DisplayHint,
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
	switch operator {
	case "*":
		// Multiplication is commutative
		result := scalar * float64(unit.Amount)
		return &Unit{
			Amount:      int64(math.Round(result)),
			Family:      unit.Family,
			System:      unit.System,
			DisplayHint: unit.DisplayHint,
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
