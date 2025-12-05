// Package evaluator provides method implementations for primitive types
// This file implements the method-call API for String, Array, Integer, Float types
package evaluator

import (
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/errors"
	"github.com/sambeau/basil/pkg/parsley/locale"
	"golang.org/x/text/currency"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// ============================================================================
// Available Methods for Fuzzy Matching
// ============================================================================

// stringMethods lists all methods available on string
var stringMethods = []string{
	"toUpper", "toLower", "trim", "split", "replace", "length", "includes",
}

// arrayMethods lists all methods available on array
var arrayMethods = []string{
	"length", "reverse", "push", "pop", "shift", "unshift", "slice", "concat",
	"includes", "indexOf", "join", "sort", "first", "last", "map", "filter",
	"reduce", "unique", "flatten", "find", "findIndex", "every", "some", "groupBy",
	"count", "countBy", "maxBy", "minBy", "sortBy", "take", "skip", "zip",
}

// integerMethods lists all methods available on integer
var integerMethods = []string{
	"abs", "format",
}

// floatMethods lists all methods available on float
var floatMethods = []string{
	"abs", "format", "round", "floor", "ceil",
}

// unknownMethodError creates an error for an unknown method with fuzzy matching hint
func unknownMethodError(method, typeName string, availableMethods []string) *Error {
	parsleyErr := errors.NewUndefinedMethod(method, typeName, availableMethods)
	return &Error{
		Message: parsleyErr.Message,
		Class:   parsleyErr.Class,
		Code:    parsleyErr.Code,
		Hints:   parsleyErr.Hints,
	}
}

// ============================================================================
// String Methods
// ============================================================================

// evalStringMethod evaluates a method call on a String
func evalStringMethod(str *String, method string, args []Object) Object {
	switch method {
	case "toUpper":
		if len(args) != 0 {
			return newArityError("toUpper", len(args), 0)
		}
		return &String{Value: strings.ToUpper(str.Value)}

	case "toLower":
		if len(args) != 0 {
			return newArityError("toLower", len(args), 0)
		}
		return &String{Value: strings.ToLower(str.Value)}

	case "trim":
		if len(args) != 0 {
			return newArityError("trim", len(args), 0)
		}
		return &String{Value: strings.TrimSpace(str.Value)}

	case "split":
		if len(args) != 1 {
			return newArityError("split", len(args), 1)
		}
		delim, ok := args[0].(*String)
		if !ok {
			return newTypeError("TYPE-0012", "split", "a string", args[0].Type())
		}
		parts := strings.Split(str.Value, delim.Value)
		elements := make([]Object, len(parts))
		for i, part := range parts {
			elements[i] = &String{Value: part}
		}
		return &Array{Elements: elements}

	case "replace":
		if len(args) != 2 {
			return newArityError("replace", len(args), 2)
		}
		old, ok := args[0].(*String)
		if !ok {
			return newTypeError("TYPE-0005", "replace", "a string", args[0].Type())
		}
		new, ok := args[1].(*String)
		if !ok {
			return newTypeError("TYPE-0006", "replace", "a string", args[1].Type())
		}
		return &String{Value: strings.ReplaceAll(str.Value, old.Value, new.Value)}

	case "length":
		if len(args) != 0 {
			return newArityError("length", len(args), 0)
		}
		// Return rune count for proper Unicode support
		return &Integer{Value: int64(len([]rune(str.Value)))}

	case "includes":
		// includes(substring) - returns true if string contains the substring
		if len(args) != 1 {
			return newArityError("includes", len(args), 1)
		}
		substr, ok := args[0].(*String)
		if !ok {
			return newTypeError("TYPE-0012", "includes", "a string", args[0].Type())
		}
		if strings.Contains(str.Value, substr.Value) {
			return TRUE
		}
		return FALSE

	default:
		return unknownMethodError(method, "string", stringMethods)
	}
}

// ============================================================================
// Array Methods
// ============================================================================

// evalArrayMethod evaluates a method call on an Array
func evalArrayMethod(arr *Array, method string, args []Object, env *Environment) Object {
	switch method {
	case "length":
		if len(args) != 0 {
			return newArityError("length", len(args), 0)
		}
		return &Integer{Value: int64(len(arr.Elements))}

	case "reverse":
		if len(args) != 0 {
			return newArityError("reverse", len(args), 0)
		}
		// Create a reversed copy
		length := len(arr.Elements)
		newElements := make([]Object, length)
		for i, elem := range arr.Elements {
			newElements[length-1-i] = elem
		}
		return &Array{Elements: newElements}

	case "sort":
		if len(args) != 0 {
			return newArityError("sort", len(args), 0)
		}
		return naturalSortArray(arr)

	case "sortBy":
		if len(args) != 1 {
			return newArityError("sortBy", len(args), 1)
		}
		fn, ok := args[0].(*Function)
		if !ok {
			return newTypeError("TYPE-0012", "sortBy", "a function", args[0].Type())
		}
		return sortArrayByFunction(arr, fn, env)

	case "map":
		if len(args) != 1 {
			return newArityError("map", len(args), 1)
		}
		fn, ok := args[0].(*Function)
		if !ok {
			return newTypeError("TYPE-0012", "map", "a function", args[0].Type())
		}
		return mapArrayWithFunction(arr, fn, env)

	case "filter":
		if len(args) != 1 {
			return newArityError("filter", len(args), 1)
		}
		fn, ok := args[0].(*Function)
		if !ok {
			return newTypeError("TYPE-0012", "filter", "a function", args[0].Type())
		}
		return filterArrayWithFunction(arr, fn, env)

	case "format":
		// format(style?, locale?)
		if len(args) > 2 {
			return newArityErrorRange("format", len(args), 0, 2)
		}

		// Convert array elements to strings
		items := make([]string, len(arr.Elements))
		for i, elem := range arr.Elements {
			items[i] = elem.Inspect()
		}

		// Get style (default to "and")
		style := locale.ListStyleAnd
		localeStr := "en-US"

		if len(args) >= 1 {
			styleStr, ok := args[0].(*String)
			if !ok {
				return newTypeError("TYPE-0005", "format", "a string (style)", args[0].Type())
			}
			switch styleStr.Value {
			case "and":
				style = locale.ListStyleAnd
			case "or":
				style = locale.ListStyleOr
			case "unit":
				style = locale.ListStyleUnit
			default:
				return newValidationError("VAL-0002", map[string]any{"Style": styleStr.Value, "Context": "format", "ValidOptions": "and, or, unit"})
			}
		}

		if len(args) == 2 {
			locStr, ok := args[1].(*String)
			if !ok {
				return newTypeError("TYPE-0006", "format", "a string (locale)", args[1].Type())
			}
			localeStr = locStr.Value
		}

		result := locale.FormatList(items, style, localeStr)
		return &String{Value: result}

	case "join":
		// join(separator?) - joins array elements into a string
		if len(args) > 1 {
			return newArityErrorRange("join", len(args), 0, 1)
		}

		separator := ""
		if len(args) == 1 {
			sepStr, ok := args[0].(*String)
			if !ok {
				return newTypeError("TYPE-0012", "join", "a string", args[0].Type())
			}
			separator = sepStr.Value
		}

		// Convert array elements to strings
		items := make([]string, len(arr.Elements))
		for i, elem := range arr.Elements {
			items[i] = objectToTemplateString(elem)
		}

		return &String{Value: strings.Join(items, separator)}

	case "shuffle":
		// shuffle() - returns a new array with elements in random order (Fisher-Yates)
		if len(args) != 0 {
			return newArityError("shuffle", len(args), 0)
		}
		length := len(arr.Elements)
		newElements := make([]Object, length)
		copy(newElements, arr.Elements)
		// Fisher-Yates shuffle
		for i := length - 1; i > 0; i-- {
			j := rand.Intn(i + 1)
			newElements[i], newElements[j] = newElements[j], newElements[i]
		}
		return &Array{Elements: newElements}

	case "pick":
		// pick() - returns a single random element (null if empty)
		// pick(n) - returns array of n random elements (with replacement, can exceed length)
		if len(args) > 1 {
			return newArityErrorRange("pick", len(args), 0, 1)
		}
		length := len(arr.Elements)

		// pick() - single element
		if len(args) == 0 {
			if length == 0 {
				return NULL
			}
			return arr.Elements[rand.Intn(length)]
		}

		// pick(n) - array of n elements with replacement
		n, ok := args[0].(*Integer)
		if !ok {
			return newTypeError("TYPE-0012", "pick", "an integer", args[0].Type())
		}
		if n.Value < 0 {
			return newValidationError("VAL-0004", map[string]any{"Method": "pick", "Got": n.Value})
		}
		if length == 0 && n.Value > 0 {
			return newValidationError("VAL-0005", map[string]any{"Method": "pick"})
		}

		result := make([]Object, n.Value)
		for i := int64(0); i < n.Value; i++ {
			result[i] = arr.Elements[rand.Intn(length)]
		}
		return &Array{Elements: result}

	case "take":
		// take(n) - returns array of n unique random elements (without replacement)
		if len(args) != 1 {
			return newArityError("take", len(args), 1)
		}
		n, ok := args[0].(*Integer)
		if !ok {
			return newTypeError("TYPE-0012", "take", "an integer", args[0].Type())
		}
		if n.Value < 0 {
			return newValidationError("VAL-0004", map[string]any{"Method": "take", "Got": n.Value})
		}
		length := len(arr.Elements)
		if int(n.Value) > length {
			return newValidationError("VAL-0006", map[string]any{"Requested": n.Value, "Length": length})
		}

		// Use Fisher-Yates partial shuffle to select n unique elements
		indices := make([]int, length)
		for i := range indices {
			indices[i] = i
		}
		result := make([]Object, n.Value)
		for i := int64(0); i < n.Value; i++ {
			j := int(i) + rand.Intn(length-int(i))
			indices[int(i)], indices[j] = indices[j], indices[int(i)]
			result[i] = arr.Elements[indices[int(i)]]
		}
		return &Array{Elements: result}

	case "includes":
		// includes(value) - returns true if array contains the value
		if len(args) != 1 {
			return newArityError("includes", len(args), 1)
		}
		for _, elem := range arr.Elements {
			if objectsEqual(args[0], elem) {
				return TRUE
			}
		}
		return FALSE

	default:
		return unknownMethodError(method, "array", arrayMethods)
	}
}

// naturalSortArray performs a natural sort on an array
func naturalSortArray(arr *Array) *Array {
	// Make a copy of elements
	elements := make([]Object, len(arr.Elements))
	copy(elements, arr.Elements)

	// Sort using natural comparison
	sort.SliceStable(elements, func(i, j int) bool {
		return compareObjects(elements[i], elements[j]) < 0
	})

	return &Array{Elements: elements}
}

// sortArrayByFunction sorts an array using a key function
func sortArrayByFunction(arr *Array, fn *Function, env *Environment) Object {
	// Make a copy of elements
	elements := make([]Object, len(arr.Elements))
	copy(elements, arr.Elements)

	// Compute keys for all elements
	keys := make([]Object, len(elements))
	for i, elem := range elements {
		extendedEnv := extendFunctionEnv(fn, []Object{elem})
		result := Eval(fn.Body, extendedEnv)
		if isError(result) {
			return result
		}
		if returnValue, ok := result.(*ReturnValue); ok {
			result = returnValue.Value
		}
		keys[i] = result
	}

	// Sort by keys
	sort.SliceStable(elements, func(i, j int) bool {
		return compareObjects(keys[i], keys[j]) < 0
	})

	return &Array{Elements: elements}
}

// mapArrayWithFunction applies a function to each element
func mapArrayWithFunction(arr *Array, fn *Function, env *Environment) Object {
	result := make([]Object, 0, len(arr.Elements))

	for _, elem := range arr.Elements {
		extendedEnv := extendFunctionEnv(fn, []Object{elem})

		var evaluated Object
		for _, stmt := range fn.Body.Statements {
			evaluated = evalStatement(stmt, extendedEnv)
			if returnValue, ok := evaluated.(*ReturnValue); ok {
				evaluated = returnValue.Value
				break
			}
			if isError(evaluated) {
				return evaluated
			}
		}

		// Skip null values (filter behavior)
		if evaluated != NULL {
			result = append(result, evaluated)
		}
	}

	return &Array{Elements: result}
}

// filterArrayWithFunction filters array elements based on a predicate function
func filterArrayWithFunction(arr *Array, fn *Function, env *Environment) Object {
	result := make([]Object, 0, len(arr.Elements))

	for _, elem := range arr.Elements {
		extendedEnv := extendFunctionEnv(fn, []Object{elem})

		var evaluated Object
		for _, stmt := range fn.Body.Statements {
			evaluated = evalStatement(stmt, extendedEnv)
			if returnValue, ok := evaluated.(*ReturnValue); ok {
				evaluated = returnValue.Value
				break
			}
			if isError(evaluated) {
				return evaluated
			}
		}

		// Include element if predicate returns truthy value
		if isTruthy(evaluated) {
			result = append(result, elem)
		}
	}

	return &Array{Elements: result}
}

// compareObjects compares two objects for sorting
func compareObjects(a, b Object) int {
	// Handle nil/NULL
	if a == nil || a == NULL {
		if b == nil || b == NULL {
			return 0
		}
		return -1
	}
	if b == nil || b == NULL {
		return 1
	}

	// Compare by type
	switch av := a.(type) {
	case *Integer:
		if bv, ok := b.(*Integer); ok {
			if av.Value < bv.Value {
				return -1
			} else if av.Value > bv.Value {
				return 1
			}
			return 0
		}
		if bv, ok := b.(*Float); ok {
			af := float64(av.Value)
			if af < bv.Value {
				return -1
			} else if af > bv.Value {
				return 1
			}
			return 0
		}
	case *Float:
		if bv, ok := b.(*Float); ok {
			if av.Value < bv.Value {
				return -1
			} else if av.Value > bv.Value {
				return 1
			}
			return 0
		}
		if bv, ok := b.(*Integer); ok {
			bf := float64(bv.Value)
			if av.Value < bf {
				return -1
			} else if av.Value > bf {
				return 1
			}
			return 0
		}
	case *String:
		if bv, ok := b.(*String); ok {
			return strings.Compare(av.Value, bv.Value)
		}
	case *Boolean:
		if bv, ok := b.(*Boolean); ok {
			if !av.Value && bv.Value {
				return -1
			} else if av.Value && !bv.Value {
				return 1
			}
			return 0
		}
	}

	// Fall back to string comparison
	return strings.Compare(a.Inspect(), b.Inspect())
}

// ============================================================================
// Dictionary Methods
// ============================================================================

// evalDictionaryMethod evaluates a method call on a Dictionary
func evalDictionaryMethod(dict *Dictionary, method string, args []Object, env *Environment) Object {
	switch method {
	case "keys":
		if len(args) != 0 {
			return newArityError("keys", len(args), 0)
		}
		orderedKeys := dict.Keys()
		keys := make([]Object, 0, len(orderedKeys))
		for _, k := range orderedKeys {
			// Skip internal fields
			if !strings.HasPrefix(k, "__") {
				keys = append(keys, &String{Value: k})
			}
		}
		return &Array{Elements: keys}

	case "values":
		if len(args) != 0 {
			return newArityError("values", len(args), 0)
		}
		orderedKeys := dict.Keys()
		values := make([]Object, 0, len(orderedKeys))
		for _, k := range orderedKeys {
			// Skip internal fields
			if !strings.HasPrefix(k, "__") {
				val := Eval(dict.Pairs[k], dict.Env)
				values = append(values, val)
			}
		}
		return &Array{Elements: values}

	case "entries":
		// entries() or entries(keyName, valueName)
		// Returns array of dictionaries with key/value pairs
		if len(args) != 0 && len(args) != 2 {
			return newArityErrorExact("entries", len(args), 0, 2)
		}

		keyName := "key"
		valueName := "value"
		if len(args) == 2 {
			k, ok := args[0].(*String)
			if !ok {
				return newTypeError("TYPE-0005", "entries", "a string (key name)", args[0].Type())
			}
			v, ok := args[1].(*String)
			if !ok {
				return newTypeError("TYPE-0006", "entries", "a string (value name)", args[1].Type())
			}
			keyName = k.Value
			valueName = v.Value
		}

		orderedKeys := dict.Keys()
		entries := make([]Object, 0, len(orderedKeys))
		for _, k := range orderedKeys {
			// Skip internal fields
			if !strings.HasPrefix(k, "__") {
				val := Eval(dict.Pairs[k], dict.Env)
				// Create a dictionary for each entry
				entryPairs := map[string]ast.Expression{
					keyName:   objectToExpression(&String{Value: k}),
					valueName: objectToExpression(val),
				}
				entries = append(entries, &Dictionary{
					Pairs:    entryPairs,
					KeyOrder: []string{keyName, valueName},
					Env:      env,
				})
			}
		}
		return &Array{Elements: entries}

	case "has":
		if len(args) != 1 {
			return newArityError("has", len(args), 1)
		}
		key, ok := args[0].(*String)
		if !ok {
			return newTypeError("TYPE-0012", "has", "a string", args[0].Type())
		}
		_, exists := dict.Pairs[key.Value]
		return nativeBoolToParsBoolean(exists)

	case "delete":
		if len(args) != 1 {
			return newArityError("delete", len(args), 1)
		}
		key, ok := args[0].(*String)
		if !ok {
			return newTypeError("TYPE-0012", "delete", "a string", args[0].Type())
		}
		dict.DeleteKey(key.Value)
		return NULL

	default:
		// Return nil for unknown methods to allow user-defined methods to be checked
		return nil
	}
}

// ============================================================================
// Number Methods (Integer and Float)
// ============================================================================

// evalIntegerMethod evaluates a method call on an Integer
func evalIntegerMethod(num *Integer, method string, args []Object) Object {
	switch method {
	case "format":
		// format(locale?)
		if len(args) > 1 {
			return newArityErrorRange("format", len(args), 0, 1)
		}
		localeStr := "en-US"
		if len(args) == 1 {
			loc, ok := args[0].(*String)
			if !ok {
				return newTypeError("TYPE-0012", "format", "a string", args[0].Type())
			}
			localeStr = loc.Value
		}
		return formatNumberWithLocale(float64(num.Value), localeStr)

	case "currency":
		// currency(code, locale?)
		if len(args) < 1 || len(args) > 2 {
			return newArityErrorRange("currency", len(args), 1, 2)
		}
		code, ok := args[0].(*String)
		if !ok {
			return newTypeError("TYPE-0005", "currency", "a string (currency code)", args[0].Type())
		}
		localeStr := "en-US"
		if len(args) == 2 {
			loc, ok := args[1].(*String)
			if !ok {
				return newTypeError("TYPE-0006", "currency", "a string (locale)", args[1].Type())
			}
			localeStr = loc.Value
		}
		return formatCurrencyWithLocale(float64(num.Value), code.Value, localeStr)

	case "percent":
		// percent(locale?)
		if len(args) > 1 {
			return newArityErrorRange("percent", len(args), 0, 1)
		}
		localeStr := "en-US"
		if len(args) == 1 {
			loc, ok := args[0].(*String)
			if !ok {
				return newTypeError("TYPE-0012", "percent", "a string", args[0].Type())
			}
			localeStr = loc.Value
		}
		return formatPercentWithLocale(float64(num.Value), localeStr)

	default:
		return unknownMethodError(method, "integer", integerMethods)
	}
}

// evalFloatMethod evaluates a method call on a Float
func evalFloatMethod(num *Float, method string, args []Object) Object {
	switch method {
	case "format":
		// format(locale?)
		if len(args) > 1 {
			return newArityErrorRange("format", len(args), 0, 1)
		}
		localeStr := "en-US"
		if len(args) == 1 {
			loc, ok := args[0].(*String)
			if !ok {
				return newTypeError("TYPE-0012", "format", "a string", args[0].Type())
			}
			localeStr = loc.Value
		}
		return formatNumberWithLocale(num.Value, localeStr)

	case "currency":
		// currency(code, locale?)
		if len(args) < 1 || len(args) > 2 {
			return newArityErrorRange("currency", len(args), 1, 2)
		}
		code, ok := args[0].(*String)
		if !ok {
			return newTypeError("TYPE-0005", "currency", "a string (currency code)", args[0].Type())
		}
		localeStr := "en-US"
		if len(args) == 2 {
			loc, ok := args[1].(*String)
			if !ok {
				return newTypeError("TYPE-0006", "currency", "a string (locale)", args[1].Type())
			}
			localeStr = loc.Value
		}
		return formatCurrencyWithLocale(num.Value, code.Value, localeStr)

	case "percent":
		// percent(locale?)
		if len(args) > 1 {
			return newArityErrorRange("percent", len(args), 0, 1)
		}
		localeStr := "en-US"
		if len(args) == 1 {
			loc, ok := args[0].(*String)
			if !ok {
				return newTypeError("TYPE-0012", "percent", "a string", args[0].Type())
			}
			localeStr = loc.Value
		}
		return formatPercentWithLocale(num.Value, localeStr)

	default:
		return unknownMethodError(method, "float", floatMethods)
	}
}

// ============================================================================
// Datetime Methods
// ============================================================================

// evalDatetimeMethod evaluates a method call on a datetime dictionary
func evalDatetimeMethod(dict *Dictionary, method string, args []Object, env *Environment) Object {
	switch method {
	case "toDict":
		// toDict() - returns the raw dictionary representation for debugging
		if len(args) != 0 {
			return newArityError("toDict", len(args), 0)
		}
		return dict

	case "format":
		// format(style?, locale?)
		if len(args) > 2 {
			return newArityErrorRange("format", len(args), 0, 2)
		}

		style := "long"
		localeStr := "en-US"

		if len(args) >= 1 {
			styleArg, ok := args[0].(*String)
			if !ok {
				return newTypeError("TYPE-0005", "format", "a string (style)", args[0].Type())
			}
			style = styleArg.Value
		}

		if len(args) == 2 {
			locArg, ok := args[1].(*String)
			if !ok {
				return newTypeError("TYPE-0006", "format", "a string (locale)", args[1].Type())
			}
			localeStr = locArg.Value
		}

		// Delegate to the formatDate builtin logic
		return formatDateWithStyleAndLocale(dict, style, localeStr, env)

	case "dayOfYear":
		if len(args) != 0 {
			return newArityError("dayOfYear", len(args), 0)
		}
		return evalDatetimeComputedProperty(dict, "dayOfYear", env)

	case "week":
		if len(args) != 0 {
			return newArityError("week", len(args), 0)
		}
		return evalDatetimeComputedProperty(dict, "week", env)

	case "timestamp":
		if len(args) != 0 {
			return newArityError("timestamp", len(args), 0)
		}
		return evalDatetimeComputedProperty(dict, "timestamp", env)

	default:
		return unknownMethodError(method, "datetime", []string{
			"format", "year", "month", "day", "hour", "minute", "second",
			"weekday", "week", "timestamp",
		})
	}
}

// ============================================================================
// Duration Methods
// ============================================================================

// evalDurationMethod evaluates a method call on a duration dictionary
func evalDurationMethod(dict *Dictionary, method string, args []Object, env *Environment) Object {
	switch method {
	case "toDict":
		// toDict() - returns the raw dictionary representation for debugging
		if len(args) != 0 {
			return newArityError("toDict", len(args), 0)
		}
		return dict

	case "format":
		// format(locale?)
		if len(args) > 1 {
			return newArityErrorRange("format", len(args), 0, 1)
		}

		// Extract months and seconds from duration
		months, seconds, err := getDurationComponents(dict, env)
		if err != nil {
			return newValidationError("VAL-0007", map[string]any{"GoError": err.Error()})
		}

		// Get locale (default to en-US)
		localeStr := "en-US"
		if len(args) == 1 {
			locStr, ok := args[0].(*String)
			if !ok {
				return newTypeError("TYPE-0012", "format", "a string", args[0].Type())
			}
			localeStr = locStr.Value
		}

		// Format the duration as relative time
		result := locale.DurationToRelativeTime(months, seconds, localeStr)
		return &String{Value: result}

	default:
		return unknownMethodError(method, "duration", []string{"format"})
	}
}

// ============================================================================
// Path Methods
// ============================================================================

// evalPathMethod evaluates a method call on a path dictionary
func evalPathMethod(dict *Dictionary, method string, args []Object, env *Environment) Object {
	switch method {
	case "toDict":
		// toDict() - returns the raw dictionary representation for debugging
		if len(args) != 0 {
			return newArityError("toDict", len(args), 0)
		}
		return dict

	case "isAbsolute":
		if len(args) != 0 {
			return newArityError("isAbsolute", len(args), 0)
		}
		// Get the absolute property
		if absExpr, ok := dict.Pairs["absolute"]; ok {
			result := Eval(absExpr, env)
			if b, ok := result.(*Boolean); ok {
				return b
			}
		}
		return FALSE

	case "isRelative":
		if len(args) != 0 {
			return newArityError("isRelative", len(args), 0)
		}
		// Get the absolute property and negate it
		if absExpr, ok := dict.Pairs["absolute"]; ok {
			result := Eval(absExpr, env)
			if b, ok := result.(*Boolean); ok {
				return nativeBoolToParsBoolean(!b.Value)
			}
		}
		return TRUE

	default:
		return unknownMethodError(method, "path", []string{
			"toString", "join", "parent", "isAbsolute", "isRelative",
		})
	}
}

// ============================================================================
// URL Methods
// ============================================================================

// evalUrlMethod evaluates a method call on a URL dictionary
func evalUrlMethod(dict *Dictionary, method string, args []Object, env *Environment) Object {
	switch method {
	case "toDict":
		// toDict() - returns the raw dictionary representation for debugging
		if len(args) != 0 {
			return newArityError("toDict", len(args), 0)
		}
		return dict

	case "origin":
		if len(args) != 0 {
			return newArityError("origin", len(args), 0)
		}
		// origin = scheme + "://" + host + (port ? ":" + port : "")
		scheme := ""
		host := ""
		port := ""

		if schemeExpr, ok := dict.Pairs["scheme"]; ok {
			if s := Eval(schemeExpr, env); s != nil {
				if str, ok := s.(*String); ok {
					scheme = str.Value
				}
			}
		}
		if hostExpr, ok := dict.Pairs["host"]; ok {
			if h := Eval(hostExpr, env); h != nil {
				if str, ok := h.(*String); ok {
					host = str.Value
				}
			}
		}
		if portExpr, ok := dict.Pairs["port"]; ok {
			if p := Eval(portExpr, env); p != nil {
				switch pv := p.(type) {
				case *Integer:
					if pv.Value > 0 {
						port = fmt.Sprintf(":%d", pv.Value)
					}
				case *String:
					if pv.Value != "" {
						port = ":" + pv.Value
					}
				}
			}
		}
		return &String{Value: scheme + "://" + host + port}

	case "pathname":
		if len(args) != 0 {
			return newArityError("pathname", len(args), 0)
		}
		// pathname = "/" + path components joined by "/"
		if pathExpr, ok := dict.Pairs["path"]; ok {
			if p := Eval(pathExpr, env); p != nil {
				if arr, ok := p.(*Array); ok {
					parts := make([]string, 0, len(arr.Elements))
					for _, elem := range arr.Elements {
						if s, ok := elem.(*String); ok && s.Value != "" {
							parts = append(parts, s.Value)
						}
					}
					return &String{Value: "/" + strings.Join(parts, "/")}
				}
			}
		}
		return &String{Value: "/"}

	case "search":
		if len(args) != 0 {
			return newArityError("search", len(args), 0)
		}
		// search = query string representation
		if queryExpr, ok := dict.Pairs["query"]; ok {
			if q := Eval(queryExpr, env); q != nil {
				if queryDict, ok := q.(*Dictionary); ok {
					if len(queryDict.Pairs) == 0 {
						return &String{Value: ""}
					}
					parts := make([]string, 0, len(queryDict.Pairs))
					for k, v := range queryDict.Pairs {
						if strings.HasPrefix(k, "__") {
							continue
						}
						val := Eval(v, env)
						parts = append(parts, k+"="+val.Inspect())
					}
					return &String{Value: "?" + strings.Join(parts, "&")}
				}
			}
		}
		return &String{Value: ""}

	case "href":
		if len(args) != 0 {
			return newArityError("href", len(args), 0)
		}
		// href = full URL string representation
		return &String{Value: urlDictToString(dict)}

	default:
		return unknownMethodError(method, "url", []string{
			"toDict", "toString", "query", "href",
		})
	}
}

// ============================================================================
// Regex Methods
// ============================================================================

// evalRegexMethod evaluates a method call on a regex dictionary
func evalRegexMethod(dict *Dictionary, method string, args []Object, env *Environment) Object {
	switch method {
	case "toDict":
		// toDict() - returns the raw dictionary representation for debugging
		if len(args) != 0 {
			return newArityError("toDict", len(args), 0)
		}
		return dict

	case "format":
		// format(style?)
		// Styles: "pattern" (just pattern), "literal" (with slashes/flags), "verbose" (pattern and flags separated)
		if len(args) > 1 {
			return newArityErrorRange("format", len(args), 0, 1)
		}

		// Get pattern and flags
		var pattern, flags string
		if patternExpr, ok := dict.Pairs["pattern"]; ok {
			if p := Eval(patternExpr, env); p != nil {
				if str, ok := p.(*String); ok {
					pattern = str.Value
				}
			}
		}
		if flagsExpr, ok := dict.Pairs["flags"]; ok {
			if f := Eval(flagsExpr, env); f != nil {
				if str, ok := f.(*String); ok {
					flags = str.Value
				}
			}
		}

		// Get style (default to "literal")
		style := "literal"
		if len(args) == 1 {
			styleArg, ok := args[0].(*String)
			if !ok {
				return newTypeError("TYPE-0012", "format", "a string (style)", args[0].Type())
			}
			style = styleArg.Value
		}

		switch style {
		case "pattern":
			return &String{Value: pattern}
		case "literal":
			return &String{Value: "/" + pattern + "/" + flags}
		case "verbose":
			if flags == "" {
				return &String{Value: "pattern: " + pattern}
			}
			return &String{Value: "pattern: " + pattern + ", flags: " + flags}
		default:
			return newValidationError("VAL-0002", map[string]any{"Style": style, "Context": "regex format", "ValidOptions": "pattern, literal, verbose"})
		}

	case "test":
		// test(string) - returns boolean if the regex matches the string
		if len(args) != 1 {
			return newArityError("test", len(args), 1)
		}
		str, ok := args[0].(*String)
		if !ok {
			return newTypeError("TYPE-0012", "test", "a string", args[0].Type())
		}

		// Get pattern and flags
		var pattern, flags string
		if patternExpr, ok := dict.Pairs["pattern"]; ok {
			if p := Eval(patternExpr, env); p != nil {
				if s, ok := p.(*String); ok {
					pattern = s.Value
				}
			}
		}
		if flagsExpr, ok := dict.Pairs["flags"]; ok {
			if f := Eval(flagsExpr, env); f != nil {
				if s, ok := f.(*String); ok {
					flags = s.Value
				}
			}
		}

		// Compile regex with flags
		re, err := compileRegex(pattern, flags)
		if err != nil {
			return newFormatError("FMT-0007", err)
		}

		return nativeBoolToParsBoolean(re.MatchString(str.Value))

	default:
		return unknownMethodError(method, "regex", []string{
			"toDict", "toString", "test", "exec", "execAll", "matches",
		})
	}
}

// ============================================================================
// File Methods
// ============================================================================

// evalFileMethod evaluates a method call on a file dictionary
func evalFileMethod(dict *Dictionary, method string, args []Object, env *Environment) Object {
	switch method {
	case "toDict":
		// toDict() - returns the raw dictionary representation for debugging
		if len(args) != 0 {
			return newArityError("toDict", len(args), 0)
		}
		return dict

	case "remove":
		// remove() - removes/deletes the file from filesystem
		if len(args) != 0 {
			return newArityError("remove", len(args), 0)
		}
		return evalFileRemove(dict, env)

	case "mkdir":
		// Create directory
		pathStr := getFilePathString(dict, env)
		if pathStr == "" {
			return newValidationError("VAL-0008", map[string]any{"Type": "file"})
		}

		absPath, pathErr := resolveModulePath(pathStr, env.Filename, env.RootPath)
		if pathErr != nil {
			return newIOError("IO-0007", pathStr, pathErr)
		}

		var recursive bool
		if len(args) > 0 {
			if optDict, ok := args[0].(*Dictionary); ok {
				if parentsExpr, ok := optDict.Pairs["parents"]; ok {
					if parentsVal := Eval(parentsExpr, optDict.Env); parentsVal != nil {
						if boolVal, ok := parentsVal.(*Boolean); ok {
							recursive = boolVal.Value
						}
					}
				}
			}
		}

		// Security check (treat as write operation)
		if err := env.checkPathAccess(absPath, "write"); err != nil {
			return newSecurityError("write", err)
		}

		var err error
		if recursive {
			err = os.MkdirAll(absPath, 0755)
		} else {
			err = os.Mkdir(absPath, 0755)
		}

		if err != nil {
			return newIOError("IO-0009", absPath, err)
		}
		return NULL

	case "rmdir":
		// Remove directory
		pathStr := getFilePathString(dict, env)
		if pathStr == "" {
			return newValidationError("VAL-0008", map[string]any{"Type": "file"})
		}

		absPath, pathErr := resolveModulePath(pathStr, env.Filename, env.RootPath)
		if pathErr != nil {
			return newIOError("IO-0007", pathStr, pathErr)
		}

		var recursive bool
		if len(args) > 0 {
			if optDict, ok := args[0].(*Dictionary); ok {
				if recursiveExpr, ok := optDict.Pairs["recursive"]; ok {
					if recursiveVal := Eval(recursiveExpr, optDict.Env); recursiveVal != nil {
						if boolVal, ok := recursiveVal.(*Boolean); ok {
							recursive = boolVal.Value
						}
					}
				}
			}
		}

		// Security check (treat as write operation)
		if err := env.checkPathAccess(absPath, "write"); err != nil {
			return newSecurityError("write", err)
		}

		var err error
		if recursive {
			err = os.RemoveAll(absPath)
		} else {
			err = os.Remove(absPath)
		}

		if err != nil {
			return newIOError("IO-0010", absPath, err)
		}
		return NULL

	default:
		return unknownMethodError(method, "file", []string{
			"toDict", "read", "write", "append", "delete",
		})
	}
}

// ============================================================================
// Dir Methods
// ============================================================================

// evalDirMethod evaluates a method call on a directory dictionary
func evalDirMethod(dict *Dictionary, method string, args []Object, env *Environment) Object {
	switch method {
	case "toDict":
		// toDict() - returns the raw dictionary representation for debugging
		if len(args) != 0 {
			return newArityError("toDict", len(args), 0)
		}
		return dict

	case "mkdir":
		// Create directory
		pathStr := getFilePathString(dict, env)
		if pathStr == "" {
			return newValidationError("VAL-0008", map[string]any{"Type": "directory"})
		}

		absPath, pathErr := resolveModulePath(pathStr, env.Filename, env.RootPath)
		if pathErr != nil {
			return newIOError("IO-0007", pathStr, pathErr)
		}

		var recursive bool
		if len(args) > 0 {
			if optDict, ok := args[0].(*Dictionary); ok {
				if parentsExpr, ok := optDict.Pairs["parents"]; ok {
					if parentsVal := Eval(parentsExpr, optDict.Env); parentsVal != nil {
						if boolVal, ok := parentsVal.(*Boolean); ok {
							recursive = boolVal.Value
						}
					}
				}
			}
		}

		// Security check (treat as write operation)
		if err := env.checkPathAccess(absPath, "write"); err != nil {
			return newSecurityError("write", err)
		}

		var err error
		if recursive {
			err = os.MkdirAll(absPath, 0755)
		} else {
			err = os.Mkdir(absPath, 0755)
		}

		if err != nil {
			return newIOError("IO-0009", absPath, err)
		}
		return NULL

	case "rmdir":
		// Remove directory
		pathStr := getFilePathString(dict, env)
		if pathStr == "" {
			return newValidationError("VAL-0008", map[string]any{"Type": "directory"})
		}

		absPath, pathErr := resolveModulePath(pathStr, env.Filename, env.RootPath)
		if pathErr != nil {
			return newIOError("IO-0007", pathStr, pathErr)
		}

		var recursive bool
		if len(args) > 0 {
			if optDict, ok := args[0].(*Dictionary); ok {
				if recursiveExpr, ok := optDict.Pairs["recursive"]; ok {
					if recursiveVal := Eval(recursiveExpr, optDict.Env); recursiveVal != nil {
						if boolVal, ok := recursiveVal.(*Boolean); ok {
							recursive = boolVal.Value
						}
					}
				}
			}
		}

		// Security check (treat as write operation)
		if err := env.checkPathAccess(absPath, "write"); err != nil {
			return newSecurityError("write", err)
		}

		var err error
		if recursive {
			err = os.RemoveAll(absPath)
		} else {
			err = os.Remove(absPath)
		}

		if err != nil {
			return newIOError("IO-0010", absPath, err)
		}
		return NULL

	default:
		return unknownMethodError(method, "dir", []string{
			"toDict", "create", "delete",
		})
	}
}

// ============================================================================
// Request Methods
// ============================================================================

// evalRequestMethod evaluates a method call on a request dictionary
func evalRequestMethod(dict *Dictionary, method string, args []Object, env *Environment) Object {
	switch method {
	case "toDict":
		// toDict() - returns the raw dictionary representation for debugging
		if len(args) != 0 {
			return newArityError("toDict", len(args), 0)
		}
		return dict

	default:
		return unknownMethodError(method, "request", []string{"toDict"})
	}
}

// ============================================================================
// Response Methods
// ============================================================================

// evalResponseMethod evaluates a method call on a response typed dictionary
func evalResponseMethod(dict *Dictionary, method string, args []Object, env *Environment) Object {
	switch method {
	case "response":
		// response() - returns the __response metadata dictionary
		if len(args) != 0 {
			return newArityError("response", len(args), 0)
		}
		if responseExpr, ok := dict.Pairs["__response"]; ok {
			return Eval(responseExpr, dict.Env)
		}
		return NULL

	case "format":
		// format() - returns the format string (json, text, etc.)
		if len(args) != 0 {
			return newArityError("format", len(args), 0)
		}
		if formatExpr, ok := dict.Pairs["__format"]; ok {
			return Eval(formatExpr, dict.Env)
		}
		return NULL

	case "data":
		// data() - returns the __data directly (for explicit access)
		if len(args) != 0 {
			return newArityError("data", len(args), 0)
		}
		if dataExpr, ok := dict.Pairs["__data"]; ok {
			return Eval(dataExpr, dict.Env)
		}
		return NULL

	case "toDict":
		// toDict() - returns the raw dictionary representation for debugging
		if len(args) != 0 {
			return newArityError("toDict", len(args), 0)
		}
		return dict

	default:
		return unknownMethodError(method, "response", []string{
			"ok", "error", "json", "text", "data", "toDict",
		})
	}
}

// ============================================================================
// Money Methods
// ============================================================================

// moneyMethods lists all methods available on money
var moneyMethods = []string{
	"format", "abs", "split",
}

// evalMoneyProperty handles property access on Money values
func evalMoneyProperty(money *Money, key string) Object {
	switch key {
	case "currency":
		return &String{Value: money.Currency}
	case "amount":
		return &Integer{Value: money.Amount}
	case "scale":
		return &Integer{Value: int64(money.Scale)}
	default:
		return unknownMethodError(key, "money", append([]string{"currency", "amount", "scale"}, moneyMethods...))
	}
}

// evalMoneyMethod evaluates a method call on a Money value
func evalMoneyMethod(money *Money, method string, args []Object) Object {
	switch method {
	case "format":
		// format() or format(locale)
		if len(args) > 1 {
			return newArityErrorRange("format", len(args), 0, 1)
		}

		localeStr := "en-US" // default locale
		if len(args) == 1 {
			localeArg, ok := args[0].(*String)
			if !ok {
				return newTypeError("TYPE-0012", "format", "a string", args[0].Type())
			}
			localeStr = localeArg.Value
		}

		return formatMoney(money, localeStr)

	case "abs":
		// abs() - returns absolute value
		if len(args) != 0 {
			return newArityError("abs", len(args), 0)
		}
		amount := money.Amount
		if amount < 0 {
			amount = -amount
		}
		return &Money{
			Amount:   amount,
			Currency: money.Currency,
			Scale:    money.Scale,
		}

	case "split":
		// split(n) - split into n parts that sum to original
		if len(args) != 1 {
			return newArityError("split", len(args), 1)
		}
		nArg, ok := args[0].(*Integer)
		if !ok {
			return newTypeError("TYPE-0012", "split", "an integer", args[0].Type())
		}
		n := nArg.Value
		if n <= 0 {
			return newStructuredError("VAL-0021", map[string]any{"Function": "split", "Expected": "a positive integer", "Got": n})
		}

		return splitMoney(money, n)

	default:
		return unknownMethodError(method, "money", moneyMethods)
	}
}

// formatMoney formats a Money value with locale-aware formatting
func formatMoney(money *Money, localeStr string) Object {
	// Try to use golang.org/x/text/currency for known currencies
	cur, err := currency.ParseISO(money.Currency)
	if err == nil {
		// Known currency - use proper locale formatting
		tag, err := language.Parse(localeStr)
		if err != nil {
			return newLocaleError(localeStr)
		}

		// Convert amount to float for formatting
		divisor := float64(1)
		for i := int8(0); i < money.Scale; i++ {
			divisor *= 10
		}
		value := float64(money.Amount) / divisor

		p := message.NewPrinter(tag)
		return &String{Value: p.Sprintf("%v", currency.Symbol(cur.Amount(value)))}
	}

	// Unknown currency (BTC, custom) - simple format: CODE amount
	return &String{Value: money.Currency + " " + money.formatAmount()}
}

// splitMoney splits a Money value into n parts that sum exactly to the original
func splitMoney(money *Money, n int64) Object {
	if n == 1 {
		return &Array{Elements: []Object{&Money{
			Amount:   money.Amount,
			Currency: money.Currency,
			Scale:    money.Scale,
		}}}
	}

	// Base amount for each part
	baseAmount := money.Amount / n
	// Remainder to distribute (can be negative if amount is negative)
	remainder := money.Amount - (baseAmount * n)

	elements := make([]Object, n)

	// Distribute: first |remainder| parts get +1 or -1 (depending on sign)
	for i := int64(0); i < n; i++ {
		amount := baseAmount
		if remainder > 0 {
			amount++
			remainder--
		} else if remainder < 0 {
			amount--
			remainder++
		}
		elements[i] = &Money{
			Amount:   amount,
			Currency: money.Currency,
			Scale:    money.Scale,
		}
	}

	return &Array{Elements: elements}
}
