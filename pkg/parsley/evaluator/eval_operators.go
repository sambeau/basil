package evaluator

import (
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/lexer"
)

// evalConcatExpression handles the ++ operator for array and dictionary concatenation
func evalConcatExpression(left, right Object) Object {
	// Handle dictionary concatenation
	if left.Type() == DICTIONARY_OBJ && right.Type() == DICTIONARY_OBJ {
		leftDict := left.(*Dictionary)
		rightDict := right.(*Dictionary)

		// Create new dictionary with merged pairs
		merged := &Dictionary{
			Pairs: make(map[string]ast.Expression),
			Env:   leftDict.Env, // Use left dict's environment
		}

		// Copy left dictionary pairs
		for k, v := range leftDict.Pairs {
			merged.Pairs[k] = v
		}

		// Copy right dictionary pairs (overwrites left if keys match)
		for k, v := range rightDict.Pairs {
			merged.Pairs[k] = v
		}

		return merged
	}

	// Convert single values to arrays
	var leftElements, rightElements []Object

	switch l := left.(type) {
	case *Array:
		leftElements = l.Elements
	default:
		leftElements = []Object{left}
	}

	switch r := right.(type) {
	case *Array:
		rightElements = r.Elements
	default:
		rightElements = []Object{right}
	}

	// Concatenate the arrays
	result := make([]Object, 0, len(leftElements)+len(rightElements))
	result = append(result, leftElements...)
	result = append(result, rightElements...)

	return &Array{Elements: result}
}

// evalInExpression handles the 'in' membership operator
// Returns true if left is contained in right (array, dictionary key, or substring)
// Returns false if right is null (null-safe membership check)
func evalInExpression(tok lexer.Token, left, right Object) Object {
	// Null-safe: x in null is always false
	if right.Type() == NULL_OBJ {
		return FALSE
	}

	switch r := right.(type) {
	case *Array:
		// Check if left is an element of the array
		for _, elem := range r.Elements {
			if objectsEqual(left, elem) {
				return TRUE
			}
		}
		return FALSE
	case *Dictionary:
		// Check if left is a key in the dictionary
		if left.Type() != STRING_OBJ {
			return newOperatorError("OP-0017", map[string]any{"Got": left.Type()})
		}
		key := left.(*String).Value
		if _, ok := r.Pairs[key]; ok {
			return TRUE
		}
		return FALSE
	case *String:
		// Check if left is a substring of right
		if left.Type() != STRING_OBJ {
			return newOperatorError("OP-0018", map[string]any{"Got": left.Type()})
		}
		substring := left.(*String).Value
		if strings.Contains(r.Value, substring) {
			return TRUE
		}
		return FALSE
	default:
		return newOperatorError("OP-0016", map[string]any{"Got": right.Type()})
	}
}

// evalIndexExpression handles array and string indexing
// If optional is true, returns NULL instead of error for out-of-bounds access
func evalIndexExpression(tok lexer.Token, left, index Object, optional bool) Object {
	// Handle response typed dictionary - unwrap __data for indexing
	if dict, ok := left.(*Dictionary); ok && isResponseDict(dict) {
		if dataExpr, ok := dict.Pairs["__data"]; ok {
			left = Eval(dataExpr, dict.Env)
			if isError(left) {
				return left
			}
		}
	}

	switch {
	case left.Type() == ARRAY_OBJ && index.Type() == INTEGER_OBJ:
		return evalArrayIndexExpression(tok, left, index, optional)
	case left.Type() == STRING_OBJ && index.Type() == INTEGER_OBJ:
		return evalStringIndexExpression(tok, left, index, optional)
	case left.Type() == DICTIONARY_OBJ && index.Type() == STRING_OBJ:
		return evalDictionaryIndexExpression(left, index, optional)
	case left.Type() == TABLE_OBJ && index.Type() == INTEGER_OBJ:
		return evalTableIndexExpression(tok, left, index, optional)
	default:
		return newIndexTypeError(tok, left.Type(), index.Type())
	}
}

// evalArrayIndexExpression handles array indexing with support for negative indices
// If optional is true, returns NULL instead of error for out-of-bounds access
func evalArrayIndexExpression(tok lexer.Token, array, index Object, optional bool) Object {
	arrayObject := array.(*Array)
	idx := index.(*Integer).Value
	max := int64(len(arrayObject.Elements))

	// Handle negative indices
	if idx < 0 {
		idx = max + idx
	}

	if idx < 0 || idx >= max {
		if optional {
			return NULL
		}
		return newIndexErrorWithPos(tok, "INDEX-0001", map[string]any{"Index": index.(*Integer).Value, "Length": max})
	}

	return arrayObject.Elements[idx]
}

// evalStringIndexExpression handles string indexing with support for negative indices
// If optional is true, returns NULL instead of error for out-of-bounds access
func evalStringIndexExpression(tok lexer.Token, str, index Object, optional bool) Object {
	stringObject := str.(*String)
	idx := index.(*Integer).Value
	max := int64(len(stringObject.Value))

	// Handle negative indices
	if idx < 0 {
		idx = max + idx
	}

	if idx < 0 || idx >= max {
		if optional {
			return NULL
		}
		return newIndexErrorWithPos(tok, "INDEX-0001", map[string]any{"Index": index.(*Integer).Value, "Length": max})
	}

	return &String{Value: string(stringObject.Value[idx])}
}

// evalTableIndexExpression handles table row indexing with support for negative indices
// If optional is true, returns NULL instead of error for out-of-bounds access
func evalTableIndexExpression(tok lexer.Token, table, index Object, optional bool) Object {
	tableObject := table.(*Table)
	idx := index.(*Integer).Value
	max := int64(len(tableObject.Rows))

	// Handle negative indices
	if idx < 0 {
		idx = max + idx
	}

	if idx < 0 || idx >= max {
		if optional {
			return NULL
		}
		return newIndexErrorWithPos(tok, "INDEX-0001", map[string]any{"Index": index.(*Integer).Value, "Length": max})
	}

	return tableObject.Rows[idx]
}

// evalSliceExpression handles array and string slicing
func evalSliceExpression(left, start, end Object) Object {
	switch left.Type() {
	case ARRAY_OBJ:
		return evalArraySliceExpression(left, start, end)
	case STRING_OBJ:
		return evalStringSliceExpression(left, start, end)
	default:
		return newSliceTypeError(left.Type())
	}
}

// evalArraySliceExpression handles array slicing
func evalArraySliceExpression(array, start, end Object) Object {
	arrayObject := array.(*Array)
	max := int64(len(arrayObject.Elements))

	var startIdx, endIdx int64

	// Determine start index
	if start == nil {
		startIdx = 0
	} else if start.Type() == INTEGER_OBJ {
		startIdx = start.(*Integer).Value
		if startIdx < 0 {
			startIdx = max + startIdx
		}
	} else {
		return newSliceIndexTypeError("start", string(start.Type()))
	}

	// Determine end index
	if end == nil {
		endIdx = max
	} else if end.Type() == INTEGER_OBJ {
		endIdx = end.(*Integer).Value
		if endIdx < 0 {
			endIdx = max + endIdx
		}
	} else {
		return newSliceIndexTypeError("end", string(end.Type()))
	}

	// Validate and clamp indices
	if startIdx < 0 {
		return newIndexError("INDEX-0001", map[string]any{"Index": startIdx, "Length": max})
	}
	if endIdx < 0 {
		return newIndexError("INDEX-0001", map[string]any{"Index": endIdx, "Length": max})
	}
	if startIdx > endIdx {
		return newIndexError("INDEX-0003", map[string]any{"Start": startIdx, "End": endIdx})
	}

	// Clamp to array bounds (allow slicing beyond length)
	if startIdx > max {
		startIdx = max
	}
	if endIdx > max {
		endIdx = max
	}

	// Create the slice
	return &Array{Elements: arrayObject.Elements[startIdx:endIdx]}
}

// evalStringSliceExpression handles string slicing
func evalStringSliceExpression(str, start, end Object) Object {
	stringObject := str.(*String)
	max := int64(len(stringObject.Value))

	var startIdx, endIdx int64

	// Determine start index
	if start == nil {
		startIdx = 0
	} else if start.Type() == INTEGER_OBJ {
		startIdx = start.(*Integer).Value
		if startIdx < 0 {
			startIdx = max + startIdx
		}
	} else {
		return newSliceIndexTypeError("start", string(start.Type()))
	}

	// Determine end index
	if end == nil {
		endIdx = max
	} else if end.Type() == INTEGER_OBJ {
		endIdx = end.(*Integer).Value
		if endIdx < 0 {
			endIdx = max + endIdx
		}
	} else {
		return newSliceIndexTypeError("end", string(end.Type()))
	}

	// Validate and clamp indices
	if startIdx < 0 {
		return newIndexError("INDEX-0001", map[string]any{"Index": startIdx, "Length": max})
	}
	if endIdx < 0 {
		return newIndexError("INDEX-0001", map[string]any{"Index": endIdx, "Length": max})
	}
	if startIdx > endIdx {
		return newIndexError("INDEX-0003", map[string]any{"Start": startIdx, "End": endIdx})
	}

	// Clamp to string bounds (allow slicing beyond length)
	if startIdx > max {
		startIdx = max
	}
	if endIdx > max {
		endIdx = max
	}

	// Create the slice
	return &String{Value: stringObject.Value[startIdx:endIdx]}
}

// evalPrefixExpression handles prefix operators (!, not, -)
func evalPrefixExpression(tok lexer.Token, operator string, right Object) Object {
	switch operator {
	case "!":
		return evalBangOperatorExpression(right)
	case "not":
		return evalBangOperatorExpression(right)
	case "-":
		return evalMinusPrefixOperatorExpression(tok, right)
	default:
		return newOperatorError("OP-0005", map[string]any{"Operator": operator, "Type": right.Type()})
	}
}

// evalBangOperatorExpression handles the ! and 'not' operators
func evalBangOperatorExpression(right Object) Object {
	if isTruthy(right) {
		return FALSE
	}
	return TRUE
}

// evalMinusPrefixOperatorExpression handles the unary minus operator
func evalMinusPrefixOperatorExpression(tok lexer.Token, right Object) Object {
	switch right.Type() {
	case INTEGER_OBJ:
		value := right.(*Integer).Value
		return &Integer{Value: -value}
	case FLOAT_OBJ:
		value := right.(*Float).Value
		return &Float{Value: -value}
	case MONEY_OBJ:
		money := right.(*Money)
		return &Money{
			Amount:   -money.Amount,
			Currency: money.Currency,
			Scale:    money.Scale,
		}
	default:
		return newOperatorError("OP-0004", map[string]any{"Type": right.Type()})
	}
}

// evalDictionaryIndexExpression handles dictionary indexing by key
// If optional is true, returns NULL for missing keys instead of error
func evalDictionaryIndexExpression(dict, index Object, optional bool) Object {
	dictObject := dict.(*Dictionary)
	key := index.(*String).Value

	// Get the expression from the dictionary
	expr, ok := dictObject.Pairs[key]
	if !ok {
		return NULL
	}

	// Create a new environment with 'this' bound to the dictionary
	dictEnv := NewEnclosedEnvironment(dictObject.Env)
	dictEnv.Set("this", dictObject)

	// Evaluate the expression in the dictionary's environment
	return Eval(expr, dictEnv)
}
