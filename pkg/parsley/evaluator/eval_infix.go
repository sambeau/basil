package evaluator

import (
	"time"

	"github.com/sambeau/basil/pkg/parsley/ast"
	perrors "github.com/sambeau/basil/pkg/parsley/errors"
	"github.com/sambeau/basil/pkg/parsley/lexer"
)

func evalInfixExpression(tok lexer.Token, operator string, left, right Object) Object {
	switch {
	case operator == "&" || operator == "&&" || operator == "and":
		// Array intersection
		if left.Type() == ARRAY_OBJ && right.Type() == ARRAY_OBJ {
			return evalArrayIntersection(left.(*Array), right.(*Array))
		}
		// Datetime intersection (must come before general dictionary intersection)
		if left.Type() == DICTIONARY_OBJ && right.Type() == DICTIONARY_OBJ {
			leftDict := left.(*Dictionary)
			rightDict := right.(*Dictionary)
			if isDatetimeDict(leftDict) && isDatetimeDict(rightDict) {
				return evalDatetimeIntersection(tok, leftDict, rightDict, NewEnvironment())
			}
			// Regular dictionary intersection
			return evalDictionaryIntersection(leftDict, rightDict)
		}
		// Boolean and
		return nativeBoolToParsBoolean(isTruthy(left) && isTruthy(right))
	case operator == "|" || operator == "||" || operator == "or":
		// Array union
		if left.Type() == ARRAY_OBJ && right.Type() == ARRAY_OBJ {
			return evalArrayUnion(left.(*Array), right.(*Array))
		}
		// Boolean or
		return nativeBoolToParsBoolean(isTruthy(left) || isTruthy(right))
	case operator == "++":
		return evalConcatExpression(left, right)
	case operator == "in":
		return evalInExpression(tok, left, right)
	case operator == "not in":
		result := evalInExpression(tok, left, right)
		if err, ok := result.(*Error); ok {
			return err
		}
		if result == TRUE {
			return FALSE
		}
		return TRUE
	case operator == "..":
		return evalRangeExpression(tok, left, right)
	// Path and URL operators with strings (must come before general string concatenation)
	case left.Type() == DICTIONARY_OBJ && right.Type() == STRING_OBJ:
		if dict := left.(*Dictionary); isPathDict(dict) {
			return evalPathStringInfixExpression(tok, operator, dict, right.(*String))
		}
		if dict := left.(*Dictionary); isUrlDict(dict) {
			return evalUrlStringInfixExpression(tok, operator, dict, right.(*String))
		}
		// Fall through to string concatenation if not path/url
		if operator == "+" {
			return evalStringConcatExpression(left, right)
		}
		return newOperatorErrorWithPos(tok, "OP-0001", map[string]any{"LeftType": left.Type(), "Operator": operator, "RightType": right.Type()})
	case operator == "+" && (left.Type() == STRING_OBJ || right.Type() == STRING_OBJ):
		// String concatenation with automatic type conversion
		return evalStringConcatExpression(left, right)
	// Regex match operators
	case operator == "~" || operator == "!~":
		if left.Type() != STRING_OBJ {
			return newOperatorErrorWithPos(tok, "OP-0007", map[string]any{"Operator": operator, "Expected": "a string", "Got": left.Type()})
		}
		if right.Type() != DICTIONARY_OBJ {
			return newOperatorErrorWithPos(tok, "OP-0008", map[string]any{"Operator": operator, "Expected": "a regex", "Got": right.Type()})
		}
		rightDict := right.(*Dictionary)
		if !isRegexDict(rightDict) {
			return newOperatorErrorWithPos(tok, "OP-0008", map[string]any{"Operator": operator, "Expected": "a regex dictionary", "Got": "dictionary"})
		}
		result := evalMatchExpression(tok, left.(*String).Value, rightDict, NewEnvironment())
		if operator == "!~" {
			// !~ returns boolean: true if no match, false if match
			return nativeBoolToParsBoolean(result == NULL)
		}
		return result // ~ returns array or null
	// Datetime dictionary operations
	case left.Type() == DICTIONARY_OBJ && right.Type() == DICTIONARY_OBJ:
		leftDict := left.(*Dictionary)
		rightDict := right.(*Dictionary)
		if isDatetimeDict(leftDict) && isDatetimeDict(rightDict) {
			return evalDatetimeInfixExpression(tok, operator, leftDict, rightDict)
		}
		if isDurationDict(leftDict) && isDurationDict(rightDict) {
			return evalDurationInfixExpression(tok, operator, leftDict, rightDict)
		}
		if isDatetimeDict(leftDict) && isDurationDict(rightDict) {
			return evalDatetimeDurationInfixExpression(tok, operator, leftDict, rightDict)
		}
		if isDurationDict(leftDict) && isDatetimeDict(rightDict) {
			// For addition, duration + datetime is commutative (same as datetime + duration)
			if operator == "+" {
				return evalDatetimeDurationInfixExpression(tok, operator, rightDict, leftDict)
			}
			// duration - datetime doesn't make sense
			return newOperatorErrorWithPos(tok, "OP-0011", map[string]any{})
		}
		// Path dictionary operations
		if isPathDict(leftDict) && isPathDict(rightDict) {
			return evalPathInfixExpression(tok, operator, leftDict, rightDict)
		}
		// URL dictionary operations
		if isUrlDict(leftDict) && isUrlDict(rightDict) {
			return evalUrlInfixExpression(tok, operator, leftDict, rightDict)
		}
		// Dictionary subtraction for regular dicts
		if operator == "-" {
			return evalDictionarySubtraction(leftDict, rightDict)
		}
		// Fall through to default comparison for non-datetime dicts
		switch operator {
		case "==":
			return nativeBoolToParsBoolean(left == right)
		case "!=":
			return nativeBoolToParsBoolean(left != right)
		}
		return newOperatorErrorWithPos(tok, "OP-0001", map[string]any{"LeftType": left.Type(), "Operator": operator, "RightType": right.Type()})
	case left.Type() == DICTIONARY_OBJ && right.Type() == INTEGER_OBJ:
		if dict := left.(*Dictionary); isDatetimeDict(dict) {
			return evalDatetimeIntegerInfixExpression(tok, operator, dict, right.(*Integer))
		}
		if dict := left.(*Dictionary); isDurationDict(dict) {
			return evalDurationIntegerInfixExpression(tok, operator, dict, right.(*Integer))
		}
		return newOperatorErrorWithPos(tok, "OP-0001", map[string]any{"LeftType": left.Type(), "Operator": operator, "RightType": right.Type()})
	case left.Type() == INTEGER_OBJ && right.Type() == DICTIONARY_OBJ:
		if dict := right.(*Dictionary); isDatetimeDict(dict) {
			return evalIntegerDatetimeInfixExpression(tok, operator, left.(*Integer), dict)
		}
		if dict := right.(*Dictionary); isDurationDict(dict) {
			// For multiplication, just swap operands (commutative)
			if operator == "*" {
				return evalDurationIntegerInfixExpression(tok, operator, dict, left.(*Integer))
			}
			// integer / duration doesn't make sense, fall through to error
		}
		return newOperatorErrorWithPos(tok, "OP-0001", map[string]any{"LeftType": left.Type(), "Operator": operator, "RightType": right.Type()})
	// Array subtraction
	case operator == "-" && left.Type() == ARRAY_OBJ && right.Type() == ARRAY_OBJ:
		return evalArraySubtraction(left.(*Array), right.(*Array))
	// Array chunking
	case operator == "/" && left.Type() == ARRAY_OBJ && right.Type() == INTEGER_OBJ:
		return evalArrayChunking(tok, left.(*Array), right.(*Integer))
	// Money operations
	case left.Type() == MONEY_OBJ && right.Type() == MONEY_OBJ:
		return evalMoneyInfixExpression(tok, operator, left.(*Money), right.(*Money))
	case left.Type() == MONEY_OBJ && right.Type() == INTEGER_OBJ:
		return evalMoneyScalarExpression(tok, operator, left.(*Money), float64(right.(*Integer).Value))
	case left.Type() == MONEY_OBJ && right.Type() == FLOAT_OBJ:
		return evalMoneyScalarExpression(tok, operator, left.(*Money), right.(*Float).Value)
	case left.Type() == INTEGER_OBJ && right.Type() == MONEY_OBJ:
		return evalScalarMoneyExpression(tok, operator, float64(left.(*Integer).Value), right.(*Money))
	case left.Type() == FLOAT_OBJ && right.Type() == MONEY_OBJ:
		return evalScalarMoneyExpression(tok, operator, left.(*Float).Value, right.(*Money))
	// Unit operations
	case left.Type() == UNIT_OBJ && right.Type() == UNIT_OBJ:
		return evalUnitInfixExpression(tok, operator, left.(*Unit), right.(*Unit))
	case left.Type() == UNIT_OBJ && right.Type() == INTEGER_OBJ:
		return evalUnitScalarExpression(tok, operator, left.(*Unit), float64(right.(*Integer).Value))
	case left.Type() == UNIT_OBJ && right.Type() == FLOAT_OBJ:
		return evalUnitScalarExpression(tok, operator, left.(*Unit), right.(*Float).Value)
	case left.Type() == INTEGER_OBJ && right.Type() == UNIT_OBJ:
		return evalScalarUnitExpression(tok, operator, float64(left.(*Integer).Value), right.(*Unit))
	case left.Type() == FLOAT_OBJ && right.Type() == UNIT_OBJ:
		return evalScalarUnitExpression(tok, operator, left.(*Float).Value, right.(*Unit))
	// String repetition
	case operator == "*" && left.Type() == STRING_OBJ && right.Type() == INTEGER_OBJ:
		return evalStringRepetition(left.(*String), right.(*Integer))
	// Array repetition
	case operator == "*" && left.Type() == ARRAY_OBJ && right.Type() == INTEGER_OBJ:
		return evalArrayRepetition(left.(*Array), right.(*Integer))
	case left.Type() == INTEGER_OBJ && right.Type() == INTEGER_OBJ:
		return evalIntegerInfixExpression(tok, operator, left, right)
	case left.Type() == FLOAT_OBJ && right.Type() == FLOAT_OBJ:
		return evalFloatInfixExpression(tok, operator, left, right)
	case left.Type() == INTEGER_OBJ && right.Type() == FLOAT_OBJ:
		return evalMixedInfixExpression(tok, operator, left, right)
	case left.Type() == FLOAT_OBJ && right.Type() == INTEGER_OBJ:
		return evalMixedInfixExpression(tok, operator, left, right)
	case left.Type() == STRING_OBJ && right.Type() == STRING_OBJ:
		return evalStringInfixExpression(tok, operator, left, right)
	case operator == "==":
		return nativeBoolToParsBoolean(left == right)
	case operator == "!=":
		return nativeBoolToParsBoolean(left != right)
	case left.Type() != right.Type():
		return newOperatorErrorWithPos(tok, "OP-0009", map[string]any{"LeftType": left.Type(), "Operator": operator, "RightType": right.Type()})
	default:
		return newOperatorErrorWithPos(tok, "OP-0001", map[string]any{"LeftType": left.Type(), "Operator": operator, "RightType": right.Type()})
	}
}

func evalIntegerInfixExpression(tok lexer.Token, operator string, left, right Object) Object {
	leftVal := left.(*Integer).Value
	rightVal := right.(*Integer).Value

	switch operator {
	case "+":
		return &Integer{Value: leftVal + rightVal}
	case "-":
		return &Integer{Value: leftVal - rightVal}
	case "*":
		return &Integer{Value: leftVal * rightVal}
	case "/":
		if rightVal == 0 {
			return newOperatorErrorWithPos(tok, "OP-0002", map[string]any{})
		}
		return &Integer{Value: leftVal / rightVal}
	case "%":
		if rightVal == 0 {
			return newOperatorErrorWithPos(tok, "OP-0006", map[string]any{})
		}
		return &Integer{Value: leftVal % rightVal}
	case "<":
		return nativeBoolToParsBoolean(leftVal < rightVal)
	case ">":
		return nativeBoolToParsBoolean(leftVal > rightVal)
	case "<=":
		return nativeBoolToParsBoolean(leftVal <= rightVal)
	case ">=":
		return nativeBoolToParsBoolean(leftVal >= rightVal)
	case "==":
		return nativeBoolToParsBoolean(leftVal == rightVal)
	case "!=":
		return nativeBoolToParsBoolean(leftVal != rightVal)
	default:
		return newOperatorError("OP-0014", map[string]any{"Type": "integer", "Operator": operator})
	}
}

func evalFloatInfixExpression(tok lexer.Token, operator string, left, right Object) Object {
	leftVal := left.(*Float).Value
	rightVal := right.(*Float).Value

	switch operator {
	case "+":
		return &Float{Value: leftVal + rightVal}
	case "-":
		return &Float{Value: leftVal - rightVal}
	case "*":
		return &Float{Value: leftVal * rightVal}
	case "/":
		if rightVal == 0 {
			return newOperatorErrorWithPos(tok, "OP-0002", map[string]any{})
		}
		return &Float{Value: leftVal / rightVal}
	case "<":
		return nativeBoolToParsBoolean(leftVal < rightVal)
	case ">":
		return nativeBoolToParsBoolean(leftVal > rightVal)
	case "<=":
		return nativeBoolToParsBoolean(leftVal <= rightVal)
	case ">=":
		return nativeBoolToParsBoolean(leftVal >= rightVal)
	case "==":
		return nativeBoolToParsBoolean(leftVal == rightVal)
	case "!=":
		return nativeBoolToParsBoolean(leftVal != rightVal)
	default:
		return newOperatorError("OP-0014", map[string]any{"Type": "float", "Operator": operator})
	}
}

func evalMixedInfixExpression(tok lexer.Token, operator string, left, right Object) Object {
	var leftVal, rightVal float64

	// Convert both operands to float64
	switch left := left.(type) {
	case *Integer:
		leftVal = float64(left.Value)
	case *Float:
		leftVal = left.Value
	default:
		return newOperatorError("OP-0010", map[string]any{"Type": left.Type()})
	}

	switch right := right.(type) {
	case *Integer:
		rightVal = float64(right.Value)
	case *Float:
		rightVal = right.Value
	default:
		return newOperatorError("OP-0010", map[string]any{"Type": right.Type()})
	}

	switch operator {
	case "+":
		return &Float{Value: leftVal + rightVal}
	case "-":
		return &Float{Value: leftVal - rightVal}
	case "*":
		return &Float{Value: leftVal * rightVal}
	case "/":
		if rightVal == 0 {
			return newOperatorErrorWithPos(tok, "OP-0002", map[string]any{})
		}
		return &Float{Value: leftVal / rightVal}
	case "<":
		return nativeBoolToParsBoolean(leftVal < rightVal)
	case ">":
		return nativeBoolToParsBoolean(leftVal > rightVal)
	case "<=":
		return nativeBoolToParsBoolean(leftVal <= rightVal)
	case ">=":
		return nativeBoolToParsBoolean(leftVal >= rightVal)
	case "==":
		return nativeBoolToParsBoolean(leftVal == rightVal)
	case "!=":
		return nativeBoolToParsBoolean(leftVal != rightVal)
	default:
		return newOperatorError("OP-0014", map[string]any{"Type": "mixed numeric", "Operator": operator})
	}
}

func evalStringInfixExpression(tok lexer.Token, operator string, left, right Object) Object {
	leftVal := left.(*String).Value
	rightVal := right.(*String).Value

	switch operator {
	case "+":
		return &String{Value: leftVal + rightVal}
	case "==":
		return nativeBoolToParsBoolean(leftVal == rightVal)
	case "!=":
		return nativeBoolToParsBoolean(leftVal != rightVal)
	case "<":
		return nativeBoolToParsBoolean(NaturalCompare(leftVal, rightVal) < 0)
	case ">":
		return nativeBoolToParsBoolean(NaturalCompare(leftVal, rightVal) > 0)
	case "<=":
		return nativeBoolToParsBoolean(NaturalCompare(leftVal, rightVal) <= 0)
	case ">=":
		return nativeBoolToParsBoolean(NaturalCompare(leftVal, rightVal) >= 0)
	default:
		return newOperatorError("OP-0001", map[string]any{"LeftType": left.Type(), "Operator": operator, "RightType": right.Type()})
	}
}

// evalDatetimeInfixExpression handles operations between two datetime dictionaries
func evalDatetimeInfixExpression(tok lexer.Token, operator string, left, right *Dictionary) Object {
	env := NewEnvironment()

	// Handle && operator for combining date and time components
	if operator == "&" || operator == "&&" || operator == "and" {
		return evalDatetimeIntersection(tok, left, right, env)
	}

	leftUnix, err := getDatetimeUnix(left, env)
	if err != nil {
		return newFormatErrorWithPos("FMT-0004", tok, err)
	}
	rightUnix, err := getDatetimeUnix(right, env)
	if err != nil {
		return newFormatErrorWithPos("FMT-0004", tok, err)
	}

	switch operator {
	case "<":
		return nativeBoolToParsBoolean(leftUnix < rightUnix)
	case ">":
		return nativeBoolToParsBoolean(leftUnix > rightUnix)
	case "<=":
		return nativeBoolToParsBoolean(leftUnix <= rightUnix)
	case ">=":
		return nativeBoolToParsBoolean(leftUnix >= rightUnix)
	case "==":
		return nativeBoolToParsBoolean(leftUnix == rightUnix)
	case "!=":
		return nativeBoolToParsBoolean(leftUnix != rightUnix)
	case "-":
		// BREAKING CHANGE: Return Duration instead of Integer
		// Calculate difference in seconds
		diffSeconds := leftUnix - rightUnix
		// Return as duration (0 months, diffSeconds seconds)
		return durationToDict(0, diffSeconds, env)
	default:
		return newOperatorError("OP-0014", map[string]any{"Type": "datetime", "Operator": operator})
	}
}

// evalDatetimeIntersection combines date and time components using && operator
// Rules:
// - Date && Time -> DateTime (combine date from left, time from right)
// - Time && Date -> DateTime (combine time from left, date from right)
// - DateTime && Time -> DateTime (replace time component)
// - DateTime && Date -> DateTime (replace date component)
// - Date && Date -> Error (ambiguous)
// - Time && Time -> Error (ambiguous)
// - DateTime && DateTime -> Error (ambiguous)
func evalDatetimeIntersection(tok lexer.Token, left, right *Dictionary, env *Environment) Object {
	leftKind := getDatetimeKind(left, env)
	rightKind := getDatetimeKind(right, env)

	// Get components from both sides
	leftTime, err := dictToTime(left, env)
	if err != nil {
		return newFormatErrorWithPos("FMT-0004", tok, err)
	}
	rightTime, err := dictToTime(right, env)
	if err != nil {
		return newFormatErrorWithPos("FMT-0004", tok, err)
	}

	var resultTime time.Time

	switch {
	case leftKind == "date" && rightKind == "time":
		// Date && Time -> combine date from left, time from right
		resultTime = time.Date(
			leftTime.Year(), leftTime.Month(), leftTime.Day(),
			rightTime.Hour(), rightTime.Minute(), rightTime.Second(),
			0, time.UTC,
		)
	case leftKind == "time" && rightKind == "date":
		// Time && Date -> combine time from left, date from right
		resultTime = time.Date(
			rightTime.Year(), rightTime.Month(), rightTime.Day(),
			leftTime.Hour(), leftTime.Minute(), leftTime.Second(),
			0, time.UTC,
		)
	case leftKind == "datetime" && rightKind == "time":
		// DateTime && Time -> replace time component
		resultTime = time.Date(
			leftTime.Year(), leftTime.Month(), leftTime.Day(),
			rightTime.Hour(), rightTime.Minute(), rightTime.Second(),
			0, time.UTC,
		)
	case leftKind == "time" && rightKind == "datetime":
		// Time && DateTime -> replace time component of right
		resultTime = time.Date(
			rightTime.Year(), rightTime.Month(), rightTime.Day(),
			leftTime.Hour(), leftTime.Minute(), leftTime.Second(),
			0, time.UTC,
		)
	case leftKind == "datetime" && rightKind == "date":
		// DateTime && Date -> replace date component
		resultTime = time.Date(
			rightTime.Year(), rightTime.Month(), rightTime.Day(),
			leftTime.Hour(), leftTime.Minute(), leftTime.Second(),
			0, time.UTC,
		)
	case leftKind == "date" && rightKind == "datetime":
		// Date && DateTime -> replace date component of right
		resultTime = time.Date(
			leftTime.Year(), leftTime.Month(), leftTime.Day(),
			rightTime.Hour(), rightTime.Minute(), rightTime.Second(),
			0, time.UTC,
		)
	case leftKind == "date" && rightKind == "date":
		return newOperatorError("OP-0012", map[string]any{"Kind": "date", "Hint": "use date && time to combine"})
	case leftKind == "time" && rightKind == "time":
		return newOperatorError("OP-0012", map[string]any{"Kind": "time", "Hint": "use date && time to combine"})
	case leftKind == "datetime" && rightKind == "datetime":
		return newOperatorError("OP-0012", map[string]any{"Kind": "datetime", "Hint": "ambiguous which components to use"})
	default:
		return newOperatorError("OP-0001", map[string]any{"LeftType": leftKind, "Operator": "&&", "RightType": rightKind})
	}

	return timeToDictWithKind(resultTime, "datetime", env)
}

// evalDatetimeIntegerInfixExpression handles datetime + integer or datetime - integer
func evalDatetimeIntegerInfixExpression(tok lexer.Token, operator string, dt *Dictionary, seconds *Integer) Object {
	env := NewEnvironment()
	unixTime, err := getDatetimeUnix(dt, env)
	if err != nil {
		return newFormatErrorWithPos("FMT-0004", tok, err)
	}

	// Get the kind from the original datetime
	kind := getDatetimeKind(dt, env)

	switch operator {
	case "+":
		// Add seconds to datetime
		newTime := time.Unix(unixTime+seconds.Value, 0).UTC()
		return timeToDictWithKind(newTime, kind, env)
	case "-":
		// Subtract seconds from datetime
		newTime := time.Unix(unixTime-seconds.Value, 0).UTC()
		return timeToDictWithKind(newTime, kind, env)
	default:
		return newOperatorError("OP-0015", map[string]any{"LeftType": "datetime", "RightType": "integer", "Operator": operator, "Supported": "+, -"})
	}
}

// evalIntegerDatetimeInfixExpression handles integer + datetime
func evalIntegerDatetimeInfixExpression(tok lexer.Token, operator string, seconds *Integer, dt *Dictionary) Object {
	env := NewEnvironment()
	unixTime, err := getDatetimeUnix(dt, env)
	if err != nil {
		return newFormatErrorWithPos("FMT-0004", tok, err)
	}

	// Get the kind from the original datetime
	kind := getDatetimeKind(dt, env)

	switch operator {
	case "+":
		// Add seconds to datetime (commutative)
		newTime := time.Unix(unixTime+seconds.Value, 0).UTC()
		return timeToDictWithKind(newTime, kind, env)
	default:
		return newOperatorError("OP-0015", map[string]any{"LeftType": "integer", "RightType": "datetime", "Operator": operator, "Supported": "+"})
	}
}

// evalDurationInfixExpression handles duration + duration or duration - duration
func evalDurationInfixExpression(tok lexer.Token, operator string, left, right *Dictionary) Object {
	env := NewEnvironment()

	leftMonths, leftSeconds, err := getDurationComponents(left, env)
	if err != nil {
		return newFormatErrorWithPos("FMT-0009", tok, err)
	}

	rightMonths, rightSeconds, err := getDurationComponents(right, env)
	if err != nil {
		return newFormatErrorWithPos("FMT-0009", tok, err)
	}

	switch operator {
	case "+":
		return durationToDict(leftMonths+rightMonths, leftSeconds+rightSeconds, env)
	case "-":
		return durationToDict(leftMonths-rightMonths, leftSeconds-rightSeconds, env)
	case "/":
		// Division returns the ratio of two durations as a number
		// For durations with months, we use approximate conversion:
		// 1 month ≈ 30.4375 days (365.25 / 12) ≈ 2,629,746 seconds
		if rightSeconds == 0 && rightMonths == 0 {
			return newOperatorError("OP-0002", map[string]any{})
		}

		// Convert both durations to approximate total seconds
		const secondsPerMonth = 2629746 // 365.25 days / 12 months * 86400 seconds/day
		leftTotal := float64(leftSeconds) + float64(leftMonths)*secondsPerMonth
		rightTotal := float64(rightSeconds) + float64(rightMonths)*secondsPerMonth

		if rightTotal == 0 {
			return newOperatorError("OP-0002", map[string]any{})
		}

		// Return the ratio as a float
		return &Float{Value: leftTotal / rightTotal}
	case "<", ">", "<=", ">=", "==", "!=":
		// Comparison only allowed for pure-seconds durations (no months)
		if leftMonths != 0 || rightMonths != 0 {
			return newOperatorError("OP-0013", map[string]any{})
		}
		switch operator {
		case "<":
			return nativeBoolToParsBoolean(leftSeconds < rightSeconds)
		case ">":
			return nativeBoolToParsBoolean(leftSeconds > rightSeconds)
		case "<=":
			return nativeBoolToParsBoolean(leftSeconds <= rightSeconds)
		case ">=":
			return nativeBoolToParsBoolean(leftSeconds >= rightSeconds)
		case "==":
			return nativeBoolToParsBoolean(leftSeconds == rightSeconds && leftMonths == rightMonths)
		case "!=":
			return nativeBoolToParsBoolean(leftSeconds != rightSeconds || leftMonths != rightMonths)
		}
	}

	return newOperatorError("OP-0014", map[string]any{"Type": "duration", "Operator": operator})
}

// evalDurationIntegerInfixExpression handles duration * integer or duration / integer
func evalDurationIntegerInfixExpression(tok lexer.Token, operator string, dur *Dictionary, num *Integer) Object {
	env := NewEnvironment()

	months, seconds, err := getDurationComponents(dur, env)
	if err != nil {
		return newFormatErrorWithPos("FMT-0009", tok, err)
	}

	switch operator {
	case "*":
		return durationToDict(months*num.Value, seconds*num.Value, env)
	case "/":
		if num.Value == 0 {
			return newOperatorError("OP-0002", map[string]any{})
		}
		return durationToDict(months/num.Value, seconds/num.Value, env)
	default:
		return newOperatorError("OP-0015", map[string]any{"LeftType": "duration", "RightType": "integer", "Operator": operator, "Supported": "*, /"})
	}
}

// evalDatetimeDurationInfixExpression handles datetime + duration or datetime - duration
func evalDatetimeDurationInfixExpression(tok lexer.Token, operator string, dt, dur *Dictionary) Object {
	env := NewEnvironment()

	// Get datetime as time.Time
	t, err := dictToTime(dt, env)
	if err != nil {
		return newFormatErrorWithPos("FMT-0004", tok, err)
	}

	// Get duration components
	months, seconds, err := getDurationComponents(dur, env)
	if err != nil {
		return newFormatErrorWithPos("FMT-0009", tok, err)
	}

	// Get the kind from the original datetime
	kind := getDatetimeKind(dt, env)

	switch operator {
	case "+":
		// Add months first (using AddDate for proper month arithmetic)
		if months != 0 {
			t = t.AddDate(0, int(months), 0)
		}
		// Then add seconds
		if seconds != 0 {
			t = t.Add(time.Duration(seconds) * time.Second)
		}
		return timeToDictWithKind(t, kind, env)
	case "-":
		// Subtract months first
		if months != 0 {
			t = t.AddDate(0, -int(months), 0)
		}
		// Then subtract seconds
		if seconds != 0 {
			t = t.Add(-time.Duration(seconds) * time.Second)
		}
		return timeToDictWithKind(t, kind, env)
	default:
		return newOperatorError("OP-0015", map[string]any{"LeftType": "datetime", "RightType": "duration", "Operator": operator, "Supported": "+, -"})
	}
}

// evalPathInfixExpression handles operations between two path dictionaries
func evalPathInfixExpression(tok lexer.Token, operator string, left, right *Dictionary) Object {
	switch operator {
	case "==":
		// Compare paths by their filesystem string representation
		leftStr := pathDictToString(left)
		rightStr := pathDictToString(right)
		return nativeBoolToParsBoolean(leftStr == rightStr)
	case "!=":
		leftStr := pathDictToString(left)
		rightStr := pathDictToString(right)
		return nativeBoolToParsBoolean(leftStr != rightStr)
	default:
		return newOperatorError("OP-0015", map[string]any{"LeftType": "path", "RightType": "path", "Operator": operator, "Supported": "==, !="})
	}
}

// evalPathStringInfixExpression handles path + string or path / string
func evalPathStringInfixExpression(tok lexer.Token, operator string, path *Dictionary, str *String) Object {
	env := path.Env
	if env == nil {
		env = NewEnvironment()
	}

	switch operator {
	case "+", "/":
		// Join path with string segment
		// Get current components
		componentsExpr, ok := path.Pairs["segments"]
		if !ok {
			return newValidationError("VAL-0017", map[string]any{"Type": "path", "Field": "segments"})
		}
		componentsObj := Eval(componentsExpr, env)
		if componentsObj.Type() != ARRAY_OBJ {
			return newValidationError("VAL-0018", map[string]any{"Field": "path components", "Expected": "array", "Got": componentsObj.Type()})
		}
		componentsArr := componentsObj.(*Array)

		// Get absolute flag
		absoluteExpr, ok := path.Pairs["absolute"]
		if !ok {
			return newValidationError("VAL-0017", map[string]any{"Type": "path", "Field": "absolute"})
		}
		absoluteObj := Eval(absoluteExpr, env)
		if absoluteObj.Type() != BOOLEAN_OBJ {
			return newValidationError("VAL-0018", map[string]any{"Field": "path absolute", "Expected": "boolean", "Got": absoluteObj.Type()})
		}
		isAbsolute := absoluteObj.(*Boolean).Value

		// Parse the string to add as new path segments
		newSegments, _ := parsePathString(str.Value)

		// Combine components
		var newComponents []string
		for _, elem := range componentsArr.Elements {
			if strObj, ok := elem.(*String); ok {
				newComponents = append(newComponents, strObj.Value)
			}
		}

		// Append new segments (skip empty leading segment if present)
		for _, seg := range newSegments {
			if seg != "" || len(newComponents) == 0 {
				newComponents = append(newComponents, seg)
			}
		}

		return pathToDict(newComponents, isAbsolute, env)
	default:
		return newOperatorError("OP-0015", map[string]any{"LeftType": "path", "RightType": "string", "Operator": operator, "Supported": "+, /"})
	}
}

// evalUrlInfixExpression handles operations between two URL dictionaries
func evalUrlInfixExpression(tok lexer.Token, operator string, left, right *Dictionary) Object {
	switch operator {
	case "==":
		// Compare URLs by their string representation
		leftStr := urlDictToString(left)
		rightStr := urlDictToString(right)
		return nativeBoolToParsBoolean(leftStr == rightStr)
	case "!=":
		leftStr := urlDictToString(left)
		rightStr := urlDictToString(right)
		return nativeBoolToParsBoolean(leftStr != rightStr)
	default:
		return newOperatorError("OP-0015", map[string]any{"LeftType": "url", "RightType": "url", "Operator": operator, "Supported": "==, !="})
	}
}

// evalUrlStringInfixExpression handles url + string for path joining
func evalUrlStringInfixExpression(tok lexer.Token, operator string, urlDict *Dictionary, str *String) Object {
	env := urlDict.Env
	if env == nil {
		env = NewEnvironment()
	}

	switch operator {
	case "+":
		// Add string to URL path
		// Get current path array
		pathExpr, ok := urlDict.Pairs["path"]
		if !ok {
			return newValidationError("VAL-0017", map[string]any{"Type": "url", "Field": "path"})
		}
		pathObj := Eval(pathExpr, env)
		if pathObj.Type() != ARRAY_OBJ {
			return newValidationError("VAL-0018", map[string]any{"Field": "url path", "Expected": "array", "Got": pathObj.Type()})
		}
		pathArr := pathObj.(*Array)

		// Parse the string as a path to add
		newSegments, _ := parsePathString(str.Value)

		// Combine path segments
		var newPath []string
		for _, elem := range pathArr.Elements {
			if strObj, ok := elem.(*String); ok {
				newPath = append(newPath, strObj.Value)
			}
		}

		// Append new segments (skip empty leading segment)
		for _, seg := range newSegments {
			if seg != "" {
				newPath = append(newPath, seg)
			}
		}

		// Create new URL dict with updated path
		pairs := make(map[string]ast.Expression)
		for k, v := range urlDict.Pairs {
			if k == "path" {
				// Create new path array
				pathElements := make([]ast.Expression, len(newPath))
				for i, seg := range newPath {
					pathElements[i] = &ast.StringLiteral{Value: seg}
				}
				pairs[k] = &ast.ArrayLiteral{Elements: pathElements}
			} else {
				pairs[k] = v
			}
		}

		return &Dictionary{Pairs: pairs, Env: env}
	default:
		return newOperatorError("OP-0015", map[string]any{"LeftType": "url", "RightType": "string", "Operator": operator, "Supported": "+"})
	}
}

// evalStringConcatExpression handles string concatenation with automatic type conversion
func evalStringConcatExpression(left, right Object) Object {
	leftStr := objectToTemplateString(left)
	rightStr := objectToTemplateString(right)
	return &String{Value: leftStr + rightStr}
}

func evalIfExpression(ie *ast.IfExpression, env *Environment) Object {
	condition := Eval(ie.Condition, env)
	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		return Eval(ie.Consequence, env)
	} else if ie.Alternative != nil {
		return Eval(ie.Alternative, env)
	} else {
		return NULL
	}
}

// isTruthy moved to eval_helpers.go

func evalIdentifier(node *ast.Identifier, env *Environment) Object {
	// Special handling for '_' - always returns null
	if node.Value == "_" {
		return NULL
	}

	// Special handling for 'null' - returns null
	if node.Value == "null" {
		return NULL
	}

	// Special handling for '__null__' - internal null representation
	if node.Value == "__null__" {
		return NULL
	}

	val, ok := env.Get(node.Value)
	if !ok {
		if builtin, ok := getBuiltins()[node.Value]; ok {
			return builtin
		}

		// Special error for @params at module scope
		if node.Value == "@params" {
			return &Error{
				Message: "@params is not available at module scope",
				Class:   "UndefinedError",
				Code:    "UNDEF-0010",
				Hints: []string{
					"@params is request-scoped and only available inside functions",
					"Move this code inside an exported function, or use `let {params} = import @basil/http` inside the function",
				},
				Line:   node.Token.Line,
				Column: node.Token.Column,
				File:   env.Filename,
			}
		}

		// Create a structured error with fuzzy matching
		parsleyErr := perrors.NewUndefinedIdentifier(node.Value, env.AllIdentifiers())
		parsleyErr.Line = node.Token.Line
		parsleyErr.Column = node.Token.Column

		// Also check for common keywords that might be misspelled
		if len(parsleyErr.Hints) == 0 {
			if suggestion := perrors.FindClosestMatch(node.Value, perrors.ParsleyKeywords); suggestion != "" {
				parsleyErr.Hints = append(parsleyErr.Hints, "Did you mean `"+suggestion+"`?")
			}
		}

		return &Error{
			Message: parsleyErr.Message,
			Class:   parsleyErr.Class,
			Code:    parsleyErr.Code,
			Hints:   parsleyErr.Hints,
			Line:    parsleyErr.Line,
			Column:  parsleyErr.Column,
			File:    env.Filename,
		}
	}

	// Resolve DynamicAccessor to current value
	if accessor, ok := val.(*DynamicAccessor); ok {
		return accessor.Resolve(env)
	}

	return val
}

func evalExpressions(exps []ast.Expression, env *Environment) []Object {
	var result []Object

	for _, e := range exps {
		evaluated := Eval(e, env)
		if isError(evaluated) {
			return []Object{evaluated}
		}
		result = append(result, evaluated)
	}

	return result
}
func evalMoneyInfixExpression(tok lexer.Token, operator string, left, right *Money) Object {
	// Currency must match for all operations
	if left.Currency != right.Currency {
		return newOperatorError("OP-0019", map[string]any{
			"LeftCurrency":  left.Currency,
			"RightCurrency": right.Currency,
		})
	}

	// Promote to higher scale if needed
	scale := max(right.Scale, left.Scale)

	leftAmount := promoteMoneyScale(left.Amount, left.Scale, scale)
	rightAmount := promoteMoneyScale(right.Amount, right.Scale, scale)

	switch operator {
	case "+":
		return &Money{
			Amount:   leftAmount + rightAmount,
			Currency: left.Currency,
			Scale:    scale,
		}
	case "-":
		return &Money{
			Amount:   leftAmount - rightAmount,
			Currency: left.Currency,
			Scale:    scale,
		}
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
		return newOperatorError("OP-0020", map[string]any{"Operator": operator})
	}
}

// evalMoneyScalarExpression handles Money * scalar and Money / scalar
func evalMoneyScalarExpression(tok lexer.Token, operator string, money *Money, scalar float64) Object {
	switch operator {
	case "*":
		// Multiply amount by scalar, use banker's rounding
		result := float64(money.Amount) * scalar
		return &Money{
			Amount:   bankersRound(result),
			Currency: money.Currency,
			Scale:    money.Scale,
		}
	case "/":
		if scalar == 0 {
			return newOperatorError("OP-0002", map[string]any{})
		}
		// Divide amount by scalar, use banker's rounding
		result := float64(money.Amount) / scalar
		return &Money{
			Amount:   bankersRound(result),
			Currency: money.Currency,
			Scale:    money.Scale,
		}
	default:
		return newOperatorError("OP-0021", map[string]any{
			"Operator": operator,
		})
	}
}

// evalScalarMoneyExpression handles scalar * Money (commutative with *)
func evalScalarMoneyExpression(tok lexer.Token, operator string, scalar float64, money *Money) Object {
	switch operator {
	case "*":
		// Multiply is commutative
		result := float64(money.Amount) * scalar
		return &Money{
			Amount:   bankersRound(result),
			Currency: money.Currency,
			Scale:    money.Scale,
		}
	default:
		return newOperatorError("OP-0021", map[string]any{
			"Operator": operator,
		})
	}
}

// promoteMoneyScale promotes an amount to a higher scale
func promoteMoneyScale(amount int64, fromScale, toScale int8) int64 {
	if fromScale >= toScale {
		return amount
	}
	for i := fromScale; i < toScale; i++ {
		amount *= 10
	}
	return amount
}
