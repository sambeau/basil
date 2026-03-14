// Package evaluator provides array method implementations via declarative registry.
// This file implements array methods migrating from switch-based dispatch (FEAT-137 Tier 1).
package evaluator

import (
	"encoding/json"
	"math/rand"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/locale"
)

// ArrayMethodRegistry defines all methods available on array values.
// This is the single source of truth for array method dispatch and introspection.
var ArrayMethodRegistry MethodRegistry

func init() {
	ArrayMethodRegistry = MethodRegistry{
		"length": {
			Fn:          arrayLength,
			Arity:       "0",
			Description: "Get element count",
		},
		"reverse": {
			Fn:          arrayReverse,
			Arity:       "0",
			Description: "Reverse order",
		},
		"sort": {
			Fn:          arraySort,
			Arity:       "0-1",
			Description: "Sort elements",
		},
		"sortBy": {
			Fn:          arraySortBy,
			Arity:       "1",
			Description: "Sort by key function",
		},
		"map": {
			Fn:          arrayMap,
			Arity:       "1",
			Description: "Transform each element",
		},
		"filter": {
			Fn:          arrayFilter,
			Arity:       "1",
			Description: "Filter by predicate",
		},
		"reduce": {
			Fn:          arrayReduce,
			Arity:       "2",
			Description: "Reduce to single value with accumulator function",
		},
		"fmt": {
			Fn:          arrayFmt,
			Arity:       "0-2",
			Description: "Format as list (and/or/unit, locale)",
		},
		"format": {
			Fn:          arrayFmt,
			Arity:       "0-2",
			Description: "Format as list (and/or/unit, locale) (alias for fmt)",
		},
		"join": {
			Fn:          arrayJoin,
			Arity:       "0-1",
			Description: "Join elements into string",
		},
		"toJSON": {
			Fn:          arrayToJSON,
			Arity:       "0",
			Description: "Convert to JSON string",
		},
		"toCSV": {
			Fn:          arrayToCSV,
			Arity:       "0-1",
			Description: "Convert to CSV string",
		},
		"shuffle": {
			Fn:          arrayShuffle,
			Arity:       "0",
			Description: "Randomly shuffle elements",
		},
		"pick": {
			Fn:          arrayPick,
			Arity:       "0-1",
			Description: "Pick random element(s)",
		},
		"take": {
			Fn:          arrayTake,
			Arity:       "1",
			Description: "Take n unique random elements",
		},
		"insert": {
			Fn:          arrayInsert,
			Arity:       "2",
			Description: "Insert at index",
		},
		"has": {
			Fn:          arrayHas,
			Arity:       "1",
			Description: "Check if element exists",
		},
		"hasAny": {
			Fn:          arrayHasAny,
			Arity:       "1",
			Description: "Check if any element from array exists",
		},
		"hasAll": {
			Fn:          arrayHasAll,
			Arity:       "1",
			Description: "Check if all elements from array exist",
		},
		"toBox": {
			Fn:          arrayToBoxMethod,
			Arity:       "0+",
			Description: "Render as box diagram",
		},
		"repr": {
			Fn:          arrayRepr,
			Arity:       "0",
			Description: "Get representation string",
		},
		"toHTML": {
			Fn:          arrayToHTMLMethod,
			Arity:       "0+",
			Description: "Convert to HTML",
		},
		"toMarkdown": {
			Fn:          arrayToMarkdownMethod,
			Arity:       "0+",
			Description: "Convert to Markdown",
		},
		"reorder": {
			Fn:          arrayReorderMethod,
			Arity:       "1+",
			Description: "Reorder/rename keys in each dictionary element",
		},
	}
	RegisterMethodRegistry("array", ArrayMethodRegistry)
}

// ============================================================================
// Array method implementations
// ============================================================================

func arrayLength(receiver Object, args []Object, env *Environment) Object {
	arr := receiver.(*Array)
	return &Integer{Value: int64(len(arr.Elements))}
}

func arrayReverse(receiver Object, args []Object, env *Environment) Object {
	arr := receiver.(*Array)
	length := len(arr.Elements)
	newElements := make([]Object, length)
	for i, elem := range arr.Elements {
		newElements[length-1-i] = elem
	}
	return &Array{Elements: newElements}
}

func arraySort(receiver Object, args []Object, env *Environment) Object {
	arr := receiver.(*Array)
	natural := true
	if len(args) == 1 {
		opts, ok := args[0].(*Dictionary)
		if !ok {
			return newTypeError("TYPE-0012", "sort", "a dictionary of options", args[0].Type())
		}
		if natExpr, hasNat := opts.Pairs["natural"]; hasNat {
			natVal := Eval(natExpr, env)
			if b, ok := natVal.(*Boolean); ok {
				natural = b.Value
			}
		}
	}
	return sortArrayWithOptions(arr, natural)
}

func arraySortBy(receiver Object, args []Object, env *Environment) Object {
	arr := receiver.(*Array)
	fn, ok := args[0].(*Function)
	if !ok {
		return newTypeError("TYPE-0012", "sortBy", "a function", args[0].Type())
	}
	return sortArrayByFunction(arr, fn, env)
}

func arrayMap(receiver Object, args []Object, env *Environment) Object {
	arr := receiver.(*Array)
	fn, ok := args[0].(*Function)
	if !ok {
		return newTypeError("TYPE-0012", "map", "a function", args[0].Type())
	}
	return mapArrayWithFunction(arr, fn, env)
}

func arrayFilter(receiver Object, args []Object, env *Environment) Object {
	arr := receiver.(*Array)
	fn, ok := args[0].(*Function)
	if !ok {
		return newTypeError("TYPE-0012", "filter", "a function", args[0].Type())
	}
	return filterArrayWithFunction(arr, fn, env)
}

func arrayReduce(receiver Object, args []Object, env *Environment) Object {
	arr := receiver.(*Array)
	fn, ok := args[0].(*Function)
	if !ok {
		return newTypeError("TYPE-0012", "reduce", "a function", args[0].Type())
	}

	accumulator := args[1]

	for _, elem := range arr.Elements {
		extendedEnv := extendFunctionEnv(fn, []Object{accumulator, elem})

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

		accumulator = evaluated
	}

	return accumulator
}

func arrayFmt(receiver Object, args []Object, env *Environment) Object {
	arr := receiver.(*Array)

	items := make([]string, len(arr.Elements))
	for i, elem := range arr.Elements {
		items[i] = elem.Inspect()
	}

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
}

func arrayJoin(receiver Object, args []Object, env *Environment) Object {
	arr := receiver.(*Array)

	separator := ""
	if len(args) == 1 {
		sepStr, ok := args[0].(*String)
		if !ok {
			return newTypeError("TYPE-0012", "join", "a string", args[0].Type())
		}
		separator = sepStr.Value
	}

	items := make([]string, 0, len(arr.Elements))
	for _, elem := range arr.Elements {
		if _, ok := elem.(*Null); ok {
			continue
		}
		s := objectToTemplateString(elem)
		if s == "" {
			continue
		}
		items = append(items, s)
	}

	return &String{Value: strings.Join(items, separator)}
}

func arrayToJSON(receiver Object, args []Object, env *Environment) Object {
	arr := receiver.(*Array)
	jsonBytes, err := json.Marshal(objectToGo(arr))
	if err != nil {
		return newFormatError("FMT-0005", err)
	}
	return &String{Value: string(jsonBytes)}
}

func arrayToCSV(receiver Object, args []Object, env *Environment) Object {
	arr := receiver.(*Array)
	hasHeader := true
	if len(args) == 1 {
		flag, ok := args[0].(*Boolean)
		if !ok {
			return newTypeError("TYPE-0004", "toCSV", "a boolean", args[0].Type())
		}
		hasHeader = flag.Value
	}
	csvBytes, err := encodeCSV(arr, hasHeader)
	if err != nil {
		return newFormatError("FMT-0007", err)
	}
	return &String{Value: string(csvBytes)}
}

func arrayShuffle(receiver Object, args []Object, env *Environment) Object {
	arr := receiver.(*Array)
	length := len(arr.Elements)
	newElements := make([]Object, length)
	copy(newElements, arr.Elements)
	for i := length - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		newElements[i], newElements[j] = newElements[j], newElements[i]
	}
	return &Array{Elements: newElements}
}

func arrayPick(receiver Object, args []Object, env *Environment) Object {
	arr := receiver.(*Array)
	length := len(arr.Elements)

	if len(args) == 0 {
		if length == 0 {
			return NULL
		}
		return arr.Elements[rand.Intn(length)]
	}

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
}

func arrayTake(receiver Object, args []Object, env *Environment) Object {
	arr := receiver.(*Array)
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
}

func arrayInsert(receiver Object, args []Object, env *Environment) Object {
	arr := receiver.(*Array)
	idxObj, ok := args[0].(*Integer)
	if !ok {
		return newTypeError("TYPE-0012", "insert", "an integer", args[0].Type())
	}
	idx := int(idxObj.Value)
	length := len(arr.Elements)

	if idx < 0 {
		idx = length + idx
	}

	if idx < 0 || idx > length {
		return newIndexError("INDEX-0001", map[string]any{"Index": idxObj.Value, "Length": length})
	}

	newElements := make([]Object, length+1)
	copy(newElements[:idx], arr.Elements[:idx])
	newElements[idx] = args[1]
	copy(newElements[idx+1:], arr.Elements[idx:])
	return &Array{Elements: newElements}
}

func arrayHas(receiver Object, args []Object, env *Environment) Object {
	arr := receiver.(*Array)
	searchItem := args[0]
	for _, elem := range arr.Elements {
		if compareObjects(elem, searchItem) == 0 {
			return TRUE
		}
	}
	return FALSE
}

func arrayHasAny(receiver Object, args []Object, env *Environment) Object {
	arr := receiver.(*Array)
	arr2, ok := args[0].(*Array)
	if !ok {
		return newTypeError("TYPE-0012", "hasAny", "an array", args[0].Type())
	}
	for _, searchItem := range arr2.Elements {
		for _, elem := range arr.Elements {
			if compareObjects(elem, searchItem) == 0 {
				return TRUE
			}
		}
	}
	return FALSE
}

func arrayHasAll(receiver Object, args []Object, env *Environment) Object {
	arr := receiver.(*Array)
	arr2, ok := args[0].(*Array)
	if !ok {
		return newTypeError("TYPE-0012", "hasAll", "an array", args[0].Type())
	}
	for _, searchItem := range arr2.Elements {
		found := false
		for _, elem := range arr.Elements {
			if compareObjects(elem, searchItem) == 0 {
				found = true
				break
			}
		}
		if !found {
			return FALSE
		}
	}
	return TRUE
}

func arrayToBoxMethod(receiver Object, args []Object, env *Environment) Object {
	arr := receiver.(*Array)
	return arrayToBox(arr, args, env)
}

func arrayRepr(receiver Object, args []Object, env *Environment) Object {
	arr := receiver.(*Array)
	return &String{Value: objectToReprString(arr)}
}

func arrayToHTMLMethod(receiver Object, args []Object, env *Environment) Object {
	arr := receiver.(*Array)
	return arrayToHTML(arr, args)
}

func arrayToMarkdownMethod(receiver Object, args []Object, env *Environment) Object {
	arr := receiver.(*Array)
	return arrayToMarkdown(arr, args)
}

func arrayReorderMethod(receiver Object, args []Object, env *Environment) Object {
	arr := receiver.(*Array)
	return arrayReorder(arr, args, env)
}
