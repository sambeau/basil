// Package evaluator provides method implementations for primitive types
// This file implements the method-call API for String, Array, Integer, Float types
package evaluator

import (
	"fmt"
	"html"
	"maps"
	"math"
	"regexp"
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
// Pre-compiled regex patterns for sanitizer methods
// ============================================================================

var (
	whitespaceRegex  = regexp.MustCompile(`\s+`)
	htmlTagRegex     = regexp.MustCompile(`<[^>]*>`)
	nonDigitRegex    = regexp.MustCompile(`[^0-9]`)
	nonSlugRegex     = regexp.MustCompile(`[^a-z0-9]+`)
	paragraphSplitRe = regexp.MustCompile(`\n\n+`) // blank-line paragraph separator
)

// ============================================================================
// Available Methods for Fuzzy Matching
// ============================================================================

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

// methodAsPropertyError creates an error when a method is accessed as a property
func methodAsPropertyError(method, typeName string) *Error {
	parsleyErr := errors.NewMethodAsProperty(method, typeName)
	return &Error{
		Message: parsleyErr.Message,
		Class:   parsleyErr.Class,
		Code:    parsleyErr.Code,
		Hints:   parsleyErr.Hints,
	}
}

// buildRenderEnv creates an environment seeded with the provided dictionary's evaluated values.
// Values are evaluated in the dictionary's own environment, then bound in a new environment that
// encloses the provided base environment so callers retain outer-scope access.
func buildRenderEnv(baseEnv *Environment, dict *Dictionary) (*Environment, Object) {
	renderEnv := NewEnclosedEnvironment(baseEnv)

	for key, valExpr := range dict.Pairs {
		val := Eval(valExpr, dict.Env)
		if isError(val) {
			return nil, val
		}
		renderEnv.Set(key, val)
	}

	return renderEnv, nil
}

// ============================================================================
// String Methods
// ============================================================================

// evalStringMethod evaluates a method call on a String using the registry.
func evalStringMethod(str *String, method string, args []Object, env *Environment) Object {
	result := dispatchFromRegistry(StringMethodRegistry, str, method, args, env)
	if result != nil {
		return result
	}
	return unknownMethodError(method, "string", StringMethodRegistry.Names())
}

// stringReplaceWithFunction replaces all occurrences of a string using a function
func stringReplaceWithFunction(input, search string, fn *Function, env *Environment) Object {
	// A literal search has no capture groups, so the callback gets exactly the
	// matched text and must declare one parameter for it (BUG-032).
	if fn.ParamCount() != 1 {
		return newCallbackArityError("replace", "1 parameter", fn.ParamCount())
	}
	if search == "" {
		return &String{Value: input}
	}

	var result strings.Builder
	remaining := input

	for {
		idx := strings.Index(remaining, search)
		if idx == -1 {
			result.WriteString(remaining)
			break
		}

		// Write everything before the match
		result.WriteString(remaining[:idx])

		// Call the function with the matched text
		extendedEnv := extendFunctionEnv(fn, []Object{&String{Value: search}})
		var replacement Object
		for _, stmt := range fn.Body.Statements {
			replacement = Eval(stmt, extendedEnv)
		}

		// Convert result to string
		if replacement != nil {
			if str, ok := replacement.(*String); ok {
				result.WriteString(str.Value)
			} else {
				result.WriteString(replacement.Inspect())
			}
		}

		// Move past the match
		remaining = remaining[idx+len(search):]
	}

	return &String{Value: result.String()}
}

// regexReplaceOnString performs regex replacement on a string
func regexReplaceOnString(input string, regexDict *Dictionary, replacement Object, env *Environment) Object {
	// Extract pattern and flags
	var pattern, flags string
	if patternExpr, ok := regexDict.Pairs["pattern"]; ok {
		if p := Eval(patternExpr, env); p != nil {
			if str, ok := p.(*String); ok {
				pattern = str.Value
			}
		}
	}
	if flagsExpr, ok := regexDict.Pairs["flags"]; ok {
		if f := Eval(flagsExpr, env); f != nil {
			if str, ok := f.(*String); ok {
				flags = str.Value
			}
		}
	}

	// Check for global flag
	global := strings.Contains(flags, "g")

	// Compile regex
	re, err := compileRegex(pattern, flags)
	if err != nil {
		return newFormatError("FMT-0007", err)
	}

	switch repl := replacement.(type) {
	case *String:
		// Simple string replacement
		if global {
			return &String{Value: re.ReplaceAllString(input, repl.Value)}
		}
		// Replace only first match
		loc := re.FindStringIndex(input)
		if loc == nil {
			return &String{Value: input}
		}
		return &String{Value: input[:loc[0]] + repl.Value + input[loc[1]:]}

	case *Function:
		// Functional replacement
		if global {
			result := re.ReplaceAllStringFunc(input, func(match string) string {
				// Get submatch info for capturing groups
				submatches := re.FindStringSubmatch(match)
				return callReplacementFunction(repl, submatches, env)
			})
			return &String{Value: result}
		}
		// Replace only first match
		loc := re.FindStringSubmatchIndex(input)
		if loc == nil {
			return &String{Value: input}
		}
		match := re.FindStringSubmatch(input)
		replacement := callReplacementFunction(repl, match, env)
		return &String{Value: input[:loc[0]] + replacement + input[loc[1]:]}

	default:
		return newTypeError("TYPE-0006", "replace", "a string or function", replacement.Type())
	}
}

// callReplacementFunction calls the replacement function with match info
func callReplacementFunction(fn *Function, submatches []string, env *Environment) string {
	// Build arguments: (match, ...groups). The site adapts to the callback's
	// declared arity — exactly ParamCount arguments are passed, drawn from the
	// full match followed by the capture groups. A group the regex does not
	// have (or that did not participate) arrives as an empty string, which is
	// what Go's submatches give for a non-participating group anyway (BUG-032).
	args := make([]Object, fn.ParamCount())
	for i := range args {
		if i < len(submatches) {
			args[i] = &String{Value: submatches[i]}
		} else {
			args[i] = &String{Value: ""}
		}
	}

	extendedEnv := extendFunctionEnv(fn, args)
	var result Object
	for _, stmt := range fn.Body.Statements {
		result = Eval(stmt, extendedEnv)
	}

	if result != nil {
		if str, ok := result.(*String); ok {
			return str.Value
		}
		return result.Inspect()
	}
	return ""
}

// outdentString removes common leading whitespace from all lines
// It ignores whitespace-only lines during measurement, finds the minimum
// indent among lines with text, and removes that amount from all text lines
func outdentString(s string) string {
	if s == "" {
		return s
	}

	lines := strings.Split(s, "\n")
	if len(lines) == 0 {
		return s
	}

	// Find minimum indent (excluding whitespace-only lines and lines with no leading whitespace)
	minIndent := -1
	for _, line := range lines {
		// Skip empty lines and whitespace-only lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Measure leading whitespace
		indent := 0
		for _, ch := range line {
			if ch == ' ' || ch == '\t' {
				indent++
			} else {
				break
			}
		}

		// Skip lines with no indent (already at column 0)
		if indent == 0 {
			continue
		}

		// Track minimum
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}

	// If no common indent found, return as-is
	if minIndent <= 0 {
		return s
	}

	// Remove common indent from all lines
	result := make([]string, len(lines))
	for i, line := range lines {
		// If line is whitespace-only, remove all whitespace
		if strings.TrimSpace(line) == "" {
			result[i] = ""
		} else if len(line) >= minIndent {
			// Remove the common indent (but only if the line has at least that much indent)
			hasIndent := true
			for j := 0; j < minIndent; j++ {
				if j >= len(line) || (line[j] != ' ' && line[j] != '\t') {
					hasIndent = false
					break
				}
			}
			if hasIndent {
				result[i] = line[minIndent:]
			} else {
				result[i] = line
			}
		} else {
			// Line is shorter than minIndent
			result[i] = line
		}
	}

	return strings.Join(result, "\n")
}

// indentString adds spaces to the beginning of all non-whitespace-only lines
func indentString(s string, spaces int) string {
	if s == "" || spaces <= 0 {
		return s
	}

	lines := strings.Split(s, "\n")
	indent := strings.Repeat(" ", spaces)
	result := make([]string, len(lines))

	for i, line := range lines {
		// If line is whitespace-only, keep it as-is
		if strings.TrimSpace(line) == "" {
			result[i] = line
		} else {
			// Add indent to lines with text
			result[i] = indent + line
		}
	}

	return strings.Join(result, "\n")
}

// ============================================================================
// Array Methods
// ============================================================================

// evalArrayMethod evaluates a method call on an Array
func evalArrayMethod(arr *Array, method string, args []Object, env *Environment) Object {
	result := dispatchFromRegistry(ArrayMethodRegistry, arr, method, args, env)
	if result != nil {
		return result
	}
	return unknownMethodError(method, "array", ArrayMethodRegistry.Names())
}

// sortArrayWithOptions performs a sort on an array with configurable options.
func sortArrayWithOptions(arr *Array, natural bool) *Array {
	// Make a copy of elements
	elements := make([]Object, len(arr.Elements))
	copy(elements, arr.Elements)

	// Sort using comparison with options
	sort.SliceStable(elements, func(i, j int) bool {
		return compareObjectsWithOptions(elements[i], elements[j], natural) < 0
	})

	return &Array{Elements: elements}
}

// sortArrayByFunction sorts an array using a key function
func sortArrayByFunction(arr *Array, fn *Function, env *Environment) Object {
	// The key function receives one element (BUG-032).
	if fn.ParamCount() != 1 {
		return newCallbackArityError("sort", "1 parameter", fn.ParamCount())
	}
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
	// The callback receives one element (BUG-032).
	if fn.ParamCount() != 1 {
		return newCallbackArityError("map", "1 parameter", fn.ParamCount())
	}
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
	// The predicate receives one element (BUG-032).
	if fn.ParamCount() != 1 {
		return newCallbackArityError("filter", "1 parameter", fn.ParamCount())
	}
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

// arrayReorder reorders and optionally renames keys in each dictionary element of an array
// With array: reorder(["key1", "key2"]) - reorders keys in each dict, ignoring non-existent keys
// With dictionary: reorder({newName: "oldName"}) - reorders and renames keys in each dict
func arrayReorder(arr *Array, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("reorder", len(args), 1)
	}

	result := make([]Object, 0, len(arr.Elements))

	for _, elem := range arr.Elements {
		dict, ok := elem.(*Dictionary)
		if !ok {
			// Non-dictionary elements pass through unchanged
			result = append(result, elem)
			continue
		}

		// Reorder the dictionary using the same logic as dict.reorder()
		reordered := dictReorder(dict, args, env)
		if isError(reordered) {
			return reordered
		}
		result = append(result, reordered)
	}

	return &Array{Elements: result}
}

// typeOrder returns the sort order for a type.
// Order: null < numbers < strings < booleans < dates < durations < money < arrays < dicts < other
func typeOrder(obj Object) int {
	if obj == nil || obj == NULL {
		return 0
	}
	switch obj := obj.(type) {
	case *Integer, *Float:
		return 1
	case *String:
		return 2
	case *Boolean:
		return 3
	case *Dictionary:
		// Check for special dictionary types
		if isDatetime(obj) {
			return 4
		}
		if isDuration(obj) {
			return 5
		}
		if isMoney(obj) {
			return 6
		}
		return 8
	case *Array:
		return 7
	default:
		return 9
	}
}

// compareObjects compares two objects for sorting using natural sort order.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func compareObjects(a, b Object) int {
	return compareObjectsWithOptions(a, b, true)
}

// compareObjectsWithOptions compares two objects with configurable natural sort.
func compareObjectsWithOptions(a, b Object, natural bool) int {
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

	// Check if types are different - compare by type order
	aOrder := typeOrder(a)
	bOrder := typeOrder(b)
	if aOrder != bOrder {
		if aOrder < bOrder {
			return -1
		}
		return 1
	}

	// Same type category - compare values
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
			if natural {
				return NaturalCompare(av.Value, bv.Value)
			}
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
	case *Dictionary:
		if bv, ok := b.(*Dictionary); ok {
			// Handle datetime comparison
			if isDatetime(av) && isDatetime(bv) {
				env := NewEnvironment()
				aUnix, _ := getDatetimeUnix(av, env)
				bUnix, _ := getDatetimeUnix(bv, env)
				if aUnix < bUnix {
					return -1
				} else if aUnix > bUnix {
					return 1
				}
				return 0
			}
			// Handle duration comparison
			if isDuration(av) && isDuration(bv) {
				aSec := getDurationSeconds(av)
				bSec := getDurationSeconds(bv)
				if aSec < bSec {
					return -1
				} else if aSec > bSec {
					return 1
				}
				return 0
			}
			// Handle money comparison (same currency only)
			if isMoney(av) && isMoney(bv) {
				aCur := getMoneyField(av, "currency")
				bCur := getMoneyField(bv, "currency")
				if aCur == bCur {
					aAmt := getMoneyAmount(av)
					bAmt := getMoneyAmount(bv)
					if aAmt < bAmt {
						return -1
					} else if aAmt > bAmt {
						return 1
					}
					return 0
				}
				// Different currencies - compare currency codes
				return strings.Compare(aCur, bCur)
			}
		}
	case *Array:
		if bv, ok := b.(*Array); ok {
			// Compare arrays element by element
			minLen := min(len(bv.Elements), len(av.Elements))
			for i := range minLen {
				cmp := compareObjectsWithOptions(av.Elements[i], bv.Elements[i], natural)
				if cmp != 0 {
					return cmp
				}
			}
			if len(av.Elements) < len(bv.Elements) {
				return -1
			} else if len(av.Elements) > len(bv.Elements) {
				return 1
			}
			return 0
		}
	}

	// Fall back to string comparison
	if natural {
		return NaturalCompare(a.Inspect(), b.Inspect())
	}
	return strings.Compare(a.Inspect(), b.Inspect())
}

// Helper functions for type detection and value extraction
func isDatetime(dict *Dictionary) bool {
	_, hasYear := dict.Pairs["year"]
	_, hasMonth := dict.Pairs["month"]
	_, hasDay := dict.Pairs["day"]
	return hasYear && hasMonth && hasDay
}

func isDuration(dict *Dictionary) bool {
	_, hasSeconds := dict.Pairs["seconds"]
	_, hasMinutes := dict.Pairs["minutes"]
	_, hasHours := dict.Pairs["hours"]
	return hasSeconds || hasMinutes || hasHours
}

func isMoney(dict *Dictionary) bool {
	_, hasCurrency := dict.Pairs["currency"]
	_, hasAmount := dict.Pairs["amount"]
	return hasCurrency && hasAmount
}

func getDurationSeconds(dict *Dictionary) float64 {
	total := 0.0
	if s, ok := dict.Pairs["seconds"]; ok {
		total += evalExprToFloat(s)
	}
	if m, ok := dict.Pairs["minutes"]; ok {
		total += evalExprToFloat(m) * 60
	}
	if h, ok := dict.Pairs["hours"]; ok {
		total += evalExprToFloat(h) * 3600
	}
	if d, ok := dict.Pairs["days"]; ok {
		total += evalExprToFloat(d) * 86400
	}
	return total
}

func getMoneyField(dict *Dictionary, field string) string {
	if expr, ok := dict.Pairs[field]; ok {
		env := NewEnvironment()
		result := Eval(expr, env)
		if s, ok := result.(*String); ok {
			return s.Value
		}
	}
	return ""
}

func getMoneyAmount(dict *Dictionary) float64 {
	if expr, ok := dict.Pairs["amount"]; ok {
		return evalExprToFloat(expr)
	}
	return 0
}

func evalExprToFloat(expr ast.Expression) float64 {
	env := NewEnvironment()
	result := Eval(expr, env)
	switch v := result.(type) {
	case *Integer:
		return float64(v.Value)
	case *Float:
		return v.Value
	}
	return 0
}

// ============================================================================
// Dictionary Methods
// ============================================================================

// evalDictionaryMethod evaluates a method call on a Dictionary
func evalDictionaryMethod(dict *Dictionary, method string, args []Object, env *Environment) Object {
	// Check if this is a schema dictionary first - dispatch to schema methods
	if IsSchemaDict(dict) {
		result := evalSchemaMethod(dict, method, args, env)
		// If schema method returns non-nil, use that result
		// nil means "method not found, try dictionary methods"
		if result != nil {
			return result
		}
	}

	result := dispatchFromRegistry(DictionaryMethodRegistry, dict, method, args, env)
	if result != nil {
		return result
	}
	// Return nil for unknown methods to allow user-defined methods to be checked
	return nil
}

// insertDictKeyAfter creates a new dictionary with a key-value pair inserted after an existing key
func insertDictKeyAfter(dict *Dictionary, afterKey, newKey string, value Object, env *Environment) *Dictionary {
	// Build new key order with newKey inserted after afterKey
	newKeyOrder := make([]string, 0, len(dict.KeyOrder)+1)
	for _, k := range dict.Keys() {
		newKeyOrder = append(newKeyOrder, k)
		if k == afterKey {
			newKeyOrder = append(newKeyOrder, newKey)
		}
	}

	// Copy pairs and add new pair
	newPairs := make(map[string]ast.Expression, len(dict.Pairs)+1)
	maps.Copy(newPairs, dict.Pairs)
	newPairs[newKey] = objectToExpression(value)

	return &Dictionary{
		Pairs:    newPairs,
		KeyOrder: newKeyOrder,
		Env:      env,
	}
}

// insertDictKeyBefore creates a new dictionary with a key-value pair inserted before an existing key
func insertDictKeyBefore(dict *Dictionary, beforeKey, newKey string, value Object, env *Environment) *Dictionary {
	// Build new key order with newKey inserted before beforeKey
	newKeyOrder := make([]string, 0, len(dict.KeyOrder)+1)
	for _, k := range dict.Keys() {
		if k == beforeKey {
			newKeyOrder = append(newKeyOrder, newKey)
		}
		newKeyOrder = append(newKeyOrder, k)
	}

	// Copy pairs and add new pair
	newPairs := make(map[string]ast.Expression, len(dict.Pairs)+1)
	maps.Copy(newPairs, dict.Pairs)
	newPairs[newKey] = objectToExpression(value)

	return &Dictionary{
		Pairs:    newPairs,
		KeyOrder: newKeyOrder,
		Env:      env,
	}
}

// dictReorder reorders and optionally renames dictionary keys
// With array: reorder(["key1", "key2"]) - reorders keys, ignoring non-existent keys (also filters)
// With dictionary: reorder({newName: "oldName"}) - reorders and renames, key order in dict is significant
func dictReorder(dict *Dictionary, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("reorder", len(args), 1)
	}

	switch arg := args[0].(type) {
	case *Array:
		// Array form: reorder(["key1", "key2", ...])
		// Creates new dict with only these keys in this order
		// Non-existent keys are silently ignored (acts as filter)
		newKeyOrder := make([]string, 0, len(arg.Elements))
		newPairs := make(map[string]ast.Expression)

		for _, elem := range arg.Elements {
			str, ok := elem.(*String)
			if !ok {
				return newTypeError("TYPE-0019", "reorder", "string", elem.Type())
			}
			// Only include keys that exist in the original dict
			if expr, exists := dict.Pairs[str.Value]; exists {
				newKeyOrder = append(newKeyOrder, str.Value)
				newPairs[str.Value] = expr
			}
		}

		return &Dictionary{
			Pairs:    newPairs,
			KeyOrder: newKeyOrder,
			Env:      env,
		}

	case *Dictionary:
		// Dictionary form: reorder({newName: "oldName", ...})
		// Creates new dict with renamed keys in the order of the mapping dict
		newKeyOrder := make([]string, 0, len(arg.Pairs))
		newPairs := make(map[string]ast.Expression)

		// Iterate in key order of the mapping dictionary
		for _, newKey := range arg.Keys() {
			// Get the old key name from the mapping
			oldKeyExpr := arg.Pairs[newKey]
			oldKeyObj := Eval(oldKeyExpr, env)
			oldKeyStr, ok := oldKeyObj.(*String)
			if !ok {
				return newTypeError("TYPE-0019", "reorder", "string value for key mapping", oldKeyObj.Type())
			}
			oldKey := oldKeyStr.Value

			// Only include if old key exists in the original dict
			if expr, exists := dict.Pairs[oldKey]; exists {
				newKeyOrder = append(newKeyOrder, newKey)
				newPairs[newKey] = expr
			}
		}

		return &Dictionary{
			Pairs:    newPairs,
			KeyOrder: newKeyOrder,
			Env:      env,
		}

	default:
		return newTypeError("TYPE-0012", "reorder", "an array or dictionary", arg.Type())
	}
}

// ============================================================================
// Boolean Methods
// ============================================================================

// evalBooleanMethod evaluates a method call on a Boolean using the registry.
func evalBooleanMethod(b *Boolean, method string, args []Object) Object {
	result := dispatchFromRegistry(BooleanMethodRegistry, b, method, args, nil)
	if result != nil {
		return result
	}
	return unknownMethodError(method, "boolean", BooleanMethodRegistry.Names())
}

// evalNullMethod evaluates a method call on Null using the registry.
func evalNullMethod(method string, args []Object) Object {
	result := dispatchFromRegistry(NullMethodRegistry, NULL, method, args, nil)
	if result != nil {
		return result
	}
	return unknownMethodError(method, "null", NullMethodRegistry.Names())
}

// ============================================================================
// Number Methods (Integer and Float)
// ============================================================================

// evalIntegerMethod evaluates a method call on an Integer using the registry.
func evalIntegerMethod(num *Integer, method string, args []Object) Object {
	result := dispatchFromRegistry(IntegerMethodRegistry, num, method, args, nil)
	if result != nil {
		return result
	}
	return unknownMethodError(method, "integer", IntegerMethodRegistry.Names())
}

// evalFloatMethod evaluates a method call on a Float using the registry.
func evalFloatMethod(num *Float, method string, args []Object) Object {
	result := dispatchFromRegistry(FloatMethodRegistry, num, method, args, nil)
	if result != nil {
		return result
	}
	return unknownMethodError(method, "float", FloatMethodRegistry.Names())
}

// ============================================================================
// Datetime Methods
// ============================================================================

// evalDatetimeMethod evaluates a method call on a datetime dictionary using the registry.
func evalDatetimeMethod(dict *Dictionary, method string, args []Object, env *Environment) Object {
	result := dispatchFromRegistry(DatetimeMethodRegistry, dict, method, args, env)
	if result != nil {
		return result
	}
	// Return an UNDEF-0002 error so dispatchMethodCall can fall through to dictionary methods.
	return unknownMethodError(method, "datetime", DatetimeMethodRegistry.Names())
}

// getDictStringValue extracts a string value from a dictionary by key.
func getDictStringValue(dict *Dictionary, key string) string {
	if expr, ok := dict.Pairs[key]; ok {
		val := Eval(expr, dict.Env)
		if s, ok := val.(*String); ok {
			return s.Value
		}
	}
	return ""
}

// ============================================================================
// Duration Methods
// ============================================================================

// evalDurationMethod evaluates a method call on a duration dictionary using the registry.
func evalDurationMethod(dict *Dictionary, method string, args []Object, env *Environment) Object {
	result := dispatchFromRegistry(DurationMethodRegistry, dict, method, args, env)
	if result != nil {
		return result
	}
	return unknownMethodError(method, "duration", DurationMethodRegistry.Names())
}

// formatDurationWithStyle formats a duration with the specified style.
func formatDurationWithStyle(dict *Dictionary, style, localeStr string, env *Environment) Object {
	months, seconds, err := getDurationComponents(dict, env)
	if err != nil {
		return newValidationError("VAL-0007", map[string]any{"GoError": err.Error()})
	}

	switch style {
	case "short":
		return &String{Value: formatDurationShort(months, seconds)}
	case "medium":
		return &String{Value: locale.DurationToRelativeTime(months, seconds, localeStr)}
	case "long":
		return &String{Value: formatDurationLong(months, seconds, localeStr)}
	default:
		return &String{Value: locale.DurationToRelativeTime(months, seconds, localeStr)}
	}
}

// formatDurationShort returns a compact duration string (2h, 3d, 1y)
func formatDurationShort(months, seconds int64) string {
	if months != 0 {
		years := months / 12
		months = months % 12
		if years > 0 {
			if months > 0 {
				return fmt.Sprintf("%dy%dmo", years, months)
			}
			return fmt.Sprintf("%dy", years)
		}
		return fmt.Sprintf("%dmo", months)
	}

	if seconds == 0 {
		return "0s"
	}

	negative := seconds < 0
	if negative {
		seconds = -seconds
	}

	days := seconds / (24 * 3600)
	seconds %= (24 * 3600)
	hours := seconds / 3600
	seconds %= 3600
	minutes := seconds / 60
	secs := seconds % 60

	var result string
	if days > 0 {
		result = fmt.Sprintf("%dd", days)
		if hours > 0 {
			result += fmt.Sprintf("%dh", hours)
		}
	} else if hours > 0 {
		result = fmt.Sprintf("%dh", hours)
		if minutes > 0 {
			result += fmt.Sprintf("%dm", minutes)
		}
	} else if minutes > 0 {
		result = fmt.Sprintf("%dm", minutes)
		if secs > 0 {
			result += fmt.Sprintf("%ds", secs)
		}
	} else {
		result = fmt.Sprintf("%ds", secs)
	}

	if negative {
		result = "-" + result
	}
	return result
}

// formatDurationLong returns a verbose duration string (2 hours 30 minutes)
func formatDurationLong(months, seconds int64, localeStr string) string {
	var parts []string

	if months != 0 {
		years := months / 12
		months = months % 12
		if years > 0 {
			if years == 1 {
				parts = append(parts, "1 year")
			} else {
				parts = append(parts, fmt.Sprintf("%d years", years))
			}
		}
		if months > 0 {
			if months == 1 {
				parts = append(parts, "1 month")
			} else {
				parts = append(parts, fmt.Sprintf("%d months", months))
			}
		}
	}

	if seconds != 0 || len(parts) == 0 {
		negative := seconds < 0
		if negative {
			seconds = -seconds
		}

		days := seconds / (24 * 3600)
		seconds %= (24 * 3600)
		hours := seconds / 3600
		seconds %= 3600
		minutes := seconds / 60
		secs := seconds % 60

		if days > 0 {
			if days == 1 {
				parts = append(parts, "1 day")
			} else {
				parts = append(parts, fmt.Sprintf("%d days", days))
			}
		}
		if hours > 0 {
			if hours == 1 {
				parts = append(parts, "1 hour")
			} else {
				parts = append(parts, fmt.Sprintf("%d hours", hours))
			}
		}
		if minutes > 0 {
			if minutes == 1 {
				parts = append(parts, "1 minute")
			} else {
				parts = append(parts, fmt.Sprintf("%d minutes", minutes))
			}
		}
		if secs > 0 || len(parts) == 0 {
			if secs == 1 {
				parts = append(parts, "1 second")
			} else {
				parts = append(parts, fmt.Sprintf("%d seconds", secs))
			}
		}

		if negative && len(parts) > 0 {
			// Append "ago" for negative durations in long format
			return strings.Join(parts, " ") + " ago"
		}
	}

	if len(parts) == 0 {
		return "0 seconds"
	}
	return strings.Join(parts, " ")
}

// formatDurationRepr returns a PLN literal representation of a duration.
func formatDurationRepr(dict *Dictionary, env *Environment) string {
	months, seconds, err := getDurationComponents(dict, env)
	if err != nil {
		return "@duration{}"
	}

	var parts []string

	if months != 0 {
		years := months / 12
		months = months % 12
		if years != 0 {
			parts = append(parts, fmt.Sprintf("years: %d", years))
		}
		if months != 0 {
			parts = append(parts, fmt.Sprintf("months: %d", months))
		}
	}

	if seconds != 0 {
		days := seconds / (24 * 3600)
		seconds %= (24 * 3600)
		hours := seconds / 3600
		seconds %= 3600
		minutes := seconds / 60
		secs := seconds % 60

		if days != 0 {
			parts = append(parts, fmt.Sprintf("days: %d", days))
		}
		if hours != 0 {
			parts = append(parts, fmt.Sprintf("hours: %d", hours))
		}
		if minutes != 0 {
			parts = append(parts, fmt.Sprintf("minutes: %d", minutes))
		}
		if secs != 0 {
			parts = append(parts, fmt.Sprintf("seconds: %d", secs))
		}
	}

	if len(parts) == 0 {
		return "@duration{seconds: 0}"
	}
	return "@duration{" + strings.Join(parts, ", ") + "}"
}

// ============================================================================
// Path Methods
// ============================================================================

// evalPathMethod evaluates a method call on a path dictionary using the registry.
func evalPathMethod(dict *Dictionary, method string, args []Object, env *Environment) Object {
	result := dispatchFromRegistry(PathMethodRegistry, dict, method, args, env)
	if result != nil {
		return result
	}
	return unknownMethodError(method, "path", PathMethodRegistry.Names())
}

// ============================================================================
// URL Methods
// ============================================================================

// evalUrlMethod evaluates a method call on a URL dictionary using the registry.
func evalUrlMethod(dict *Dictionary, method string, args []Object, env *Environment) Object {
	result := dispatchFromRegistry(UrlMethodRegistry, dict, method, args, env)
	if result != nil {
		return result
	}
	return unknownMethodError(method, "url", UrlMethodRegistry.Names())
}

// ============================================================================
// Regex Methods
// ============================================================================

// evalRegexMethod evaluates a method call on a regex dictionary using the registry.
func evalRegexMethod(dict *Dictionary, method string, args []Object, env *Environment) Object {
	result := dispatchFromRegistry(RegexMethodRegistry, dict, method, args, env)
	if result != nil {
		return result
	}
	return unknownMethodError(method, "regex", RegexMethodRegistry.Names())
}

// ============================================================================
// File Methods
// ============================================================================

// evalFileMethod evaluates a method call on a file dictionary using the registry.
func evalFileMethod(dict *Dictionary, method string, args []Object, env *Environment) Object {
	result := dispatchFromRegistry(FileMethodRegistry, dict, method, args, env)
	if result != nil {
		return result
	}
	return unknownMethodError(method, "file", FileMethodRegistry.Names())
}

// ============================================================================
// Dir Methods
// ============================================================================

// evalDirMethod evaluates a method call on a directory dictionary using the registry.
func evalDirMethod(dict *Dictionary, method string, args []Object, env *Environment) Object {
	result := dispatchFromRegistry(DirMethodRegistry, dict, method, args, env)
	if result != nil {
		return result
	}
	return unknownMethodError(method, "dir", DirMethodRegistry.Names())
}

// ============================================================================
// Request Methods
// ============================================================================

// evalRequestMethod evaluates a method call on a request dictionary using the registry.
func evalRequestMethod(dict *Dictionary, method string, args []Object, env *Environment) Object {
	result := dispatchFromRegistry(RequestMethodRegistry, dict, method, args, env)
	if result != nil {
		return result
	}
	return unknownMethodError(method, "request", RequestMethodRegistry.Names())
}

// ============================================================================
// Response Methods
// ============================================================================

// evalResponseMethod evaluates a method call on a response typed dictionary using the registry.
func evalResponseMethod(dict *Dictionary, method string, args []Object, env *Environment) Object {
	result := dispatchFromRegistry(ResponseMethodRegistry, dict, method, args, env)
	if result != nil {
		return result
	}
	// Return an UNDEF-0002 error so dispatchMethodCall can fall through to dictionary methods.
	return unknownMethodError(method, "response", ResponseMethodRegistry.Names())
}

// ============================================================================
// Money Methods
// ============================================================================

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
		// Check if it's a method name - provide helpful error
		methodNames := MoneyMethodRegistry.Names()
		if _, ok := MoneyMethodRegistry[key]; ok {
			return methodAsPropertyError(key, "Money")
		}
		return unknownMethodError(key, "money", append([]string{"currency", "amount", "scale"}, methodNames...))
	}
}

// evalMoneyMethod evaluates a method call on a Money value using the registry.
func evalMoneyMethod(money *Money, method string, args []Object) Object {
	result := dispatchFromRegistry(MoneyMethodRegistry, money, method, args, nil)
	if result != nil {
		return result
	}
	return unknownMethodError(method, "money", MoneyMethodRegistry.Names())
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
	for i := range n {
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

// ============================================================================
// Text View Helper Functions
// ============================================================================

// highlightString wraps all occurrences of phrase in the string with an HTML tag.
// The string is HTML-escaped first to prevent XSS. Matching is case-insensitive.
func highlightString(s, phrase, tag string) string {
	// Empty phrase or string - return escaped original
	if phrase == "" || s == "" {
		return html.EscapeString(s)
	}

	// Escape the source string first
	escaped := html.EscapeString(s)

	// Also escape the phrase for matching (in case it contains HTML chars)
	escapedPhrase := html.EscapeString(phrase)

	// If phrase is empty after escaping, return escaped string
	if escapedPhrase == "" {
		return escaped
	}

	// Build case-insensitive regex pattern for the escaped phrase
	// Escape regex special characters in the phrase
	quotedPhrase := regexp.QuoteMeta(escapedPhrase)
	pattern := regexp.MustCompile("(?i)" + quotedPhrase)

	// Replace all matches, preserving original case
	result := pattern.ReplaceAllStringFunc(escaped, func(match string) string {
		return "<" + tag + ">" + match + "</" + tag + ">"
	})

	return result
}

// textToParagraphs converts plain text with blank lines to HTML paragraphs.
// The text is HTML-escaped to prevent XSS. Single newlines become <br/>.
func textToParagraphs(s string) string {
	// Empty or whitespace-only input
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	// Normalize line endings
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	// Split on blank lines (two or more consecutive newlines)
	paragraphs := paragraphSplitRe.Split(s, -1)

	var result strings.Builder
	for _, para := range paragraphs {
		// Trim and skip empty paragraphs
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		// Escape HTML
		para = html.EscapeString(para)

		// Convert single newlines to <br/>
		para = strings.ReplaceAll(para, "\n", "<br/>")

		result.WriteString("<p>")
		result.WriteString(para)
		result.WriteString("</p>")
	}

	return result.String()
}

// humanizeNumber formats a number in compact form (e.g., 1.2M, 1K).
// Uses CLDR locale data for proper internationalization.
func humanizeNumber(value float64, localeStr string) string {
	// Handle special cases
	if math.IsNaN(value) {
		return "NaN"
	}
	if math.IsInf(value, 1) {
		return "∞"
	}
	if math.IsInf(value, -1) {
		return "-∞"
	}

	// Parse locale, fall back to en-US
	tag, err := language.Parse(localeStr)
	if err != nil {
		tag = language.AmericanEnglish
	}

	// For small numbers, just format normally
	absValue := math.Abs(value)
	if absValue < 1000 {
		p := message.NewPrinter(tag)
		// Format with up to 1 decimal place for small numbers
		if value == math.Trunc(value) {
			return p.Sprintf("%.0f", value)
		}
		return p.Sprintf("%.1f", value)
	}

	// Determine the appropriate suffix and divisor
	// Using short scale (US/modern): K, M, B, T
	type compactUnit struct {
		threshold float64
		divisor   float64
		suffix    string
	}

	// Different languages use different compact forms
	// For now, we'll use English-style suffixes and locale-aware number formatting
	units := []compactUnit{
		{1e15, 1e15, "Q"}, // Quadrillion
		{1e12, 1e12, "T"}, // Trillion
		{1e9, 1e9, "B"},   // Billion
		{1e6, 1e6, "M"},   // Million
		{1e3, 1e3, "K"},   // Thousand
	}

	var divisor float64 = 1
	var suffix = ""

	for _, u := range units {
		if absValue >= u.threshold {
			divisor = u.divisor
			suffix = u.suffix
			break
		}
	}

	scaledValue := value / divisor
	p := message.NewPrinter(tag)

	// Format with 1 decimal place if needed, otherwise whole number
	if scaledValue == math.Trunc(scaledValue) {
		return p.Sprintf("%.0f", scaledValue) + suffix
	}

	// Round to 1 decimal place
	rounded := math.Round(scaledValue*10) / 10
	if rounded == math.Trunc(rounded) {
		return p.Sprintf("%.0f", rounded) + suffix
	}
	return p.Sprintf("%.1f", rounded) + suffix
}

// ============================================================================
// Array/Dictionary HTML and Markdown Methods
// ============================================================================

// arrayToHTML converts an array to an HTML list
func arrayToHTML(arr *Array, args []Object) Object {
	if len(args) > 1 {
		return newArityErrorRange("toHTML", len(args), 0, 1)
	}

	ordered := false
	if len(args) == 1 {
		opts, ok := args[0].(*Dictionary)
		if !ok {
			return newTypeError("TYPE-0012", "toHTML", "a dictionary", args[0].Type())
		}
		if orderedExpr, exists := opts.Pairs["ordered"]; exists {
			// Need to evaluate the expression to get the actual value
			if orderedObj, ok := orderedExpr.(Object); ok {
				if orderedVal, ok := orderedObj.(*Boolean); ok {
					ordered = orderedVal.Value
				}
			}
		}
	}

	var result strings.Builder
	if ordered {
		result.WriteString("<ol>")
	} else {
		result.WriteString("<ul>")
	}

	for _, elem := range arr.Elements {
		result.WriteString("<li>")
		result.WriteString(html.EscapeString(objectToPrintString(elem)))
		result.WriteString("</li>")
	}

	if ordered {
		result.WriteString("</ol>")
	} else {
		result.WriteString("</ul>")
	}

	return &String{Value: result.String()}
}

// arrayToMarkdown converts an array to a Markdown list
func arrayToMarkdown(arr *Array, args []Object) Object {
	if len(args) > 1 {
		return newArityErrorRange("toMarkdown", len(args), 0, 1)
	}

	ordered := false
	if len(args) == 1 {
		opts, ok := args[0].(*Dictionary)
		if !ok {
			return newTypeError("TYPE-0012", "toMarkdown", "a dictionary", args[0].Type())
		}
		if orderedExpr, exists := opts.Pairs["ordered"]; exists {
			// Need to evaluate the expression to get the actual value
			if orderedObj, ok := orderedExpr.(Object); ok {
				if orderedVal, ok := orderedObj.(*Boolean); ok {
					ordered = orderedVal.Value
				}
			}
		}
	}

	var result strings.Builder
	for i, elem := range arr.Elements {
		if ordered {
			fmt.Fprintf(&result, "%d. ", i+1)
		} else {
			result.WriteString("- ")
		}
		result.WriteString(objectToPrintString(elem))
		result.WriteString("\n")
	}

	return &String{Value: strings.TrimSuffix(result.String(), "\n")}
}

// dictToHTML converts a dictionary to an HTML definition list or table
func dictToHTML(dict *Dictionary, args []Object, env *Environment) Object {
	if len(args) > 1 {
		return newArityErrorRange("toHTML", len(args), 0, 1)
	}

	useTable := false
	if len(args) == 1 {
		opts, ok := args[0].(*Dictionary)
		if !ok {
			return newTypeError("TYPE-0012", "toHTML", "a dictionary", args[0].Type())
		}
		if tableExpr, exists := opts.Pairs["table"]; exists {
			// Need to evaluate the expression to get the actual value
			if tableObj, ok := tableExpr.(Object); ok {
				if tableVal, ok := tableObj.(*Boolean); ok {
					useTable = tableVal.Value
				}
			}
		}
	}

	var result strings.Builder

	if useTable {
		result.WriteString("<table><tr><th>Key</th><th>Value</th></tr>")
		for _, key := range dict.Keys() {
			if valExpr, exists := dict.Pairs[key]; exists {
				val := Eval(valExpr, env)
				result.WriteString("<tr><td>")
				result.WriteString(html.EscapeString(key))
				result.WriteString("</td><td>")
				result.WriteString(html.EscapeString(objectToPrintString(val)))
				result.WriteString("</td></tr>")
			}
		}
		result.WriteString("</table>")
	} else {
		result.WriteString("<dl>")
		for _, key := range dict.Keys() {
			if valExpr, exists := dict.Pairs[key]; exists {
				val := Eval(valExpr, env)
				result.WriteString("<dt>")
				result.WriteString(html.EscapeString(key))
				result.WriteString("</dt><dd>")
				result.WriteString(html.EscapeString(objectToPrintString(val)))
				result.WriteString("</dd>")
			}
		}
		result.WriteString("</dl>")
	}

	return &String{Value: result.String()}
}

// dictToMarkdown converts a dictionary to a Markdown table
func dictToMarkdown(dict *Dictionary, args []Object, env *Environment) Object {
	if len(args) > 1 {
		return newArityErrorRange("toMarkdown", len(args), 0, 1)
	}

	var result strings.Builder
	result.WriteString("| Key | Value |\n")
	result.WriteString("|-----|-------|\n")

	for _, key := range dict.Keys() {
		if valExpr, exists := dict.Pairs[key]; exists {
			val := Eval(valExpr, env)
			// Escape pipe characters in markdown
			escapedKey := strings.ReplaceAll(key, "|", "\\|")
			escapedVal := strings.ReplaceAll(objectToPrintString(val), "|", "\\|")
			result.WriteString("| ")
			result.WriteString(escapedKey)
			result.WriteString(" | ")
			result.WriteString(escapedVal)
			result.WriteString(" |\n")
		}
	}

	return &String{Value: strings.TrimSuffix(result.String(), "\n")}
}
