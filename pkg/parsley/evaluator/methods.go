// Package evaluator provides method implementations for primitive types
// This file implements the method-call API for String, Array, Integer, Float types
package evaluator

import (
	"encoding/json"
	"fmt"
	"html"
	"maps"
	"math"
	"math/rand"
	"net/url"
	"os"
	"regexp"
	"slices"
	"sort"
	"strconv"
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
	whitespaceRegex = regexp.MustCompile(`\s+`)
	htmlTagRegex    = regexp.MustCompile(`<[^>]*>`)
	nonDigitRegex   = regexp.MustCompile(`[^0-9]`)
	nonSlugRegex    = regexp.MustCompile(`[^a-z0-9]+`)
)

// ============================================================================
// Available Methods for Fuzzy Matching
// ============================================================================

// stringMethods lists all methods available on string
var stringMethods = []string{
	"type", "toUpper", "toLower", "toTitle", "trim", "split", "replace", "length", "includes",
	"render", "highlight", "paragraphs", "parseJSON", "parseCSV",
	"collapse", "normalizeSpace", "stripSpace", "stripHtml", "digits", "slug",
	"htmlEncode", "htmlDecode", "urlEncode", "urlDecode", "urlPathEncode", "urlQueryEncode",
	"outdent", "indent", "toBox", "repr", "toJSON",
}

// arrayMethods lists all methods available on array
var arrayMethods = []string{
	"type", "length", "reverse", "sort", "sortBy", "map", "filter", "reduce", "format", "join",
	"toJSON", "toCSV", "shuffle", "pick", "take", "insert", "has", "hasAny", "hasAll", "toBox",
	"repr", "toHTML", "toMarkdown", "reorder",
}

// integerMethods lists all methods available on integer
var integerMethods = []string{
	"type", "format", "humanize", "toBox", "repr", "toJSON",
}

// floatMethods lists all methods available on float
var floatMethods = []string{
	"type", "format", "humanize", "toBox", "repr", "toJSON",
}

// dictionaryMethods lists all methods available on dictionary
var dictionaryMethods = []string{
	"type", "keys", "values", "entries", "has", "delete", "insertAfter", "insertBefore", "render", "toJSON", "toBox",
	"repr", "toHTML", "toMarkdown", "as", "reorder",
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

// evalStringMethod evaluates a method call on a String
func evalStringMethod(str *String, method string, args []Object, env *Environment) Object {
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

	case "toTitle":
		if len(args) != 0 {
			return newArityError("toTitle", len(args), 0)
		}
		return &String{Value: strings.Title(str.Value)}

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
		// replace(search, replacement) - replace all occurrences
		// search can be a string or regex
		// replacement can be a string or function
		if len(args) != 2 {
			return newArityError("replace", len(args), 2)
		}

		switch search := args[0].(type) {
		case *String:
			// String search - replace all occurrences
			switch replacement := args[1].(type) {
			case *String:
				// Simple string replacement
				return &String{Value: strings.ReplaceAll(str.Value, search.Value, replacement.Value)}
			case *Function:
				// Functional replacement - call function for each match
				return stringReplaceWithFunction(str.Value, search.Value, replacement, env)
			default:
				return newTypeError("TYPE-0006", "replace", "a string or function", args[1].Type())
			}

		case *Dictionary:
			// Check if it's a regex
			if !isRegexDict(search) {
				return newTypeError("TYPE-0005", "replace", "a string or regex", "dictionary")
			}
			// Regex replacement
			return regexReplaceOnString(str.Value, search, args[1], env)

		default:
			return newTypeError("TYPE-0005", "replace", "a string or regex", args[0].Type())
		}

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

	case "highlight":
		// highlight(phrase, tag?) - wrap search matches in HTML tag with XSS protection
		if len(args) < 1 || len(args) > 2 {
			return newArityErrorRange("highlight", len(args), 1, 2)
		}
		phrase, ok := args[0].(*String)
		if !ok {
			return newTypeError("TYPE-0012", "highlight", "a string", args[0].Type())
		}
		tag := "mark" // default tag
		if len(args) == 2 {
			tagArg, ok := args[1].(*String)
			if !ok {
				return newTypeError("TYPE-0013", "highlight", "a string", args[1].Type())
			}
			tag = tagArg.Value
		}
		return &String{Value: highlightString(str.Value, phrase.Value, tag)}

	case "paragraphs":
		// paragraphs() - convert plain text with blank lines to HTML paragraphs
		if len(args) != 0 {
			return newArityError("paragraphs", len(args), 0)
		}
		return &String{Value: textToParagraphs(str.Value)}

	case "render":
		if len(args) > 1 {
			return newArityErrorRange("render", len(args), 0, 1)
		}

		if env == nil {
			env = NewEnvironment()
		}

		renderEnv := env
		if len(args) == 1 {
			dict, ok := args[0].(*Dictionary)
			if !ok {
				return newTypeError("TYPE-0012", "render", "a dictionary", args[0].Type())
			}

			var errObj Object
			renderEnv, errObj = buildRenderEnv(env, dict)
			if errObj != nil {
				return errObj
			}
		}

		return interpolateRawString(str.Value, renderEnv)

	case "parseMarkdown":
		// parseMarkdown(options?) - parse markdown string to {html, raw, md}
		if len(args) > 1 {
			return newArityErrorRange("parseMarkdown", len(args), 0, 1)
		}

		var options *Dictionary
		if len(args) == 1 {
			optDict, ok := args[0].(*Dictionary)
			if !ok {
				return newTypeError("TYPE-0012", "parseMarkdown", "a dictionary", args[0].Type())
			}
			options = optDict
		}

		result, err := parseMarkdown(str.Value, options, env)
		if err != nil {
			return err
		}
		return result

	case "parseJSON":
		if len(args) != 0 {
			return newArityError("parseJSON", len(args), 0)
		}
		result, err := parseJSON(str.Value)
		if err != nil {
			return err
		}
		return result

	case "parseCSV":
		if len(args) > 1 {
			return newArityErrorRange("parseCSV", len(args), 0, 1)
		}
		hasHeader := true
		if len(args) == 1 {
			flag, ok := args[0].(*Boolean)
			if !ok {
				return newTypeError("TYPE-0004", "parseCSV", "a boolean", args[0].Type())
			}
			hasHeader = flag.Value
		}

		result, err := parseCSV([]byte(str.Value), hasHeader)
		if err != nil {
			return err
		}
		return result

	case "collapse":
		if len(args) != 0 {
			return newArityError("collapse", len(args), 0)
		}
		return &String{Value: whitespaceRegex.ReplaceAllString(str.Value, " ")}

	case "normalizeSpace":
		if len(args) != 0 {
			return newArityError("normalizeSpace", len(args), 0)
		}
		collapsed := whitespaceRegex.ReplaceAllString(str.Value, " ")
		return &String{Value: strings.TrimSpace(collapsed)}

	case "stripSpace":
		if len(args) != 0 {
			return newArityError("stripSpace", len(args), 0)
		}
		return &String{Value: whitespaceRegex.ReplaceAllString(str.Value, "")}

	case "stripHtml":
		if len(args) != 0 {
			return newArityError("stripHtml", len(args), 0)
		}
		stripped := htmlTagRegex.ReplaceAllString(str.Value, "")
		return &String{Value: html.UnescapeString(stripped)}

	case "digits":
		if len(args) != 0 {
			return newArityError("digits", len(args), 0)
		}
		return &String{Value: nonDigitRegex.ReplaceAllString(str.Value, "")}

	case "slug":
		if len(args) != 0 {
			return newArityError("slug", len(args), 0)
		}
		lower := strings.ToLower(str.Value)
		return &String{Value: strings.Trim(nonSlugRegex.ReplaceAllString(lower, "-"), "-")}

	case "htmlEncode":
		if len(args) != 0 {
			return newArityError("htmlEncode", len(args), 0)
		}
		return &String{Value: html.EscapeString(str.Value)}

	case "htmlDecode":
		if len(args) != 0 {
			return newArityError("htmlDecode", len(args), 0)
		}
		return &String{Value: html.UnescapeString(str.Value)}

	case "urlEncode":
		if len(args) != 0 {
			return newArityError("urlEncode", len(args), 0)
		}
		// QueryEscape uses + for spaces (application/x-www-form-urlencoded)
		return &String{Value: url.QueryEscape(str.Value)}

	case "urlDecode":
		if len(args) != 0 {
			return newArityError("urlDecode", len(args), 0)
		}
		decoded, err := url.QueryUnescape(str.Value)
		if err != nil {
			return newFormatError("FMT-0011", err)
		}
		return &String{Value: decoded}

	case "urlPathEncode":
		if len(args) != 0 {
			return newArityError("urlPathEncode", len(args), 0)
		}
		// PathEscape encodes path segments (including /)
		return &String{Value: url.PathEscape(str.Value)}

	case "urlQueryEncode":
		if len(args) != 0 {
			return newArityError("urlQueryEncode", len(args), 0)
		}
		// QueryEscape encodes query values (& and = are encoded)
		return &String{Value: url.QueryEscape(str.Value)}

	case "outdent":
		if len(args) != 0 {
			return newArityError("outdent", len(args), 0)
		}
		return &String{Value: outdentString(str.Value)}

	case "indent":
		if len(args) != 1 {
			return newArityError("indent", len(args), 1)
		}
		spaces, ok := args[0].(*Integer)
		if !ok {
			return newTypeError("TYPE-0012", "indent", "an integer", args[0].Type())
		}
		return &String{Value: indentString(str.Value, int(spaces.Value))}

	case "toBox":
		if len(args) != 0 {
			return newArityError("toBox", len(args), 0)
		}
		br := NewBoxRenderer()
		return &String{Value: br.RenderSingleValue(str.Value)}

	case "repr":
		if len(args) != 0 {
			return newArityError("repr", len(args), 0)
		}
		return &String{Value: objectToReprString(str)}

	case "toJSON":
		if len(args) != 0 {
			return newArityError("toJSON", len(args), 0)
		}
		// JSON encode the string
		jsonBytes, _ := json.Marshal(str.Value)
		return &String{Value: string(jsonBytes)}

	default:
		return unknownMethodError(method, "string", stringMethods)
	}
}

// stringReplaceWithFunction replaces all occurrences of a string using a function
func stringReplaceWithFunction(input, search string, fn *Function, env *Environment) Object {
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
	// Build arguments: (match, ...groups)
	// If function takes 1 arg, just pass match
	// If function takes more args, pass match and capture groups
	var args []Object

	if len(submatches) > 0 {
		// First element is the full match
		args = append(args, &String{Value: submatches[0]})

		// Additional elements are capture groups
		if len(fn.Params) > 1 && len(submatches) > 1 {
			for _, group := range submatches[1:] {
				args = append(args, &String{Value: group})
			}
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
		// sort() or sort({natural: false})
		if len(args) > 1 {
			return newArityErrorRange("sort", len(args), 0, 1)
		}
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

	case "reduce":
		// reduce(fn, initial) - reduces array to single value
		// fn takes (accumulator, element) and returns new accumulator
		if len(args) != 2 {
			return newArityError("reduce", len(args), 2)
		}
		fn, ok := args[0].(*Function)
		if !ok {
			return newTypeError("TYPE-0012", "reduce", "a function", args[0].Type())
		}

		// Start with initial value
		accumulator := args[1]

		// Apply function to each element
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

	case "toJSON":
		if len(args) != 0 {
			return newArityError("toJSON", len(args), 0)
		}
		jsonBytes, err := json.Marshal(objectToGo(arr))
		if err != nil {
			return newFormatError("FMT-0005", err)
		}
		return &String{Value: string(jsonBytes)}

	case "toCSV":
		if len(args) > 1 {
			return newArityErrorRange("toCSV", len(args), 0, 1)
		}
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

	case "insert":
		// insert(index, value) - returns new array with value inserted before index
		// Supports negative indices (e.g., -1 = before last element)
		if len(args) != 2 {
			return newArityError("insert", len(args), 2)
		}
		idxObj, ok := args[0].(*Integer)
		if !ok {
			return newTypeError("TYPE-0012", "insert", "an integer", args[0].Type())
		}
		idx := int(idxObj.Value)
		length := len(arr.Elements)

		// Handle negative indices
		if idx < 0 {
			idx = length + idx
		}

		// Bounds check: index must be in [0, length] (inclusive of length for append)
		if idx < 0 || idx > length {
			return newIndexError("INDEX-0001", map[string]any{"Index": idxObj.Value, "Length": length})
		}

		// Create new array with element inserted
		newElements := make([]Object, length+1)
		copy(newElements[:idx], arr.Elements[:idx])
		newElements[idx] = args[1]
		copy(newElements[idx+1:], arr.Elements[idx:])
		return &Array{Elements: newElements}

	case "has":
		// has(item) - returns true if item is in array
		if len(args) != 1 {
			return newArityError("has", len(args), 1)
		}
		searchItem := args[0]
		for _, elem := range arr.Elements {
			if compareObjects(elem, searchItem) == 0 {
				return TRUE
			}
		}
		return FALSE

	case "hasAny":
		// hasAny(array2) - returns true if any item in array2 is in this array
		if len(args) != 1 {
			return newArityError("hasAny", len(args), 1)
		}
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

	case "hasAll":
		// hasAll(array2) - returns true if all items in array2 are in this array
		if len(args) != 1 {
			return newArityError("hasAll", len(args), 1)
		}
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

	case "toBox":
		return arrayToBox(arr, args, env)

	case "repr":
		if len(args) != 0 {
			return newArityError("repr", len(args), 0)
		}
		return &String{Value: objectToReprString(arr)}

	case "toHTML":
		return arrayToHTML(arr, args)

	case "toMarkdown":
		return arrayToMarkdown(arr, args)

	case "reorder":
		return arrayReorder(arr, args, env)

	default:
		return unknownMethodError(method, "array", arrayMethods)
	}
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

	case "insertAfter":
		// insertAfter(existingKey, newKey, value) - returns new dictionary with k/v inserted after existingKey
		if len(args) != 3 {
			return newArityError("insertAfter", len(args), 3)
		}
		existingKey, ok := args[0].(*String)
		if !ok {
			return newTypeError("TYPE-0012", "insertAfter", "a string (existing key)", args[0].Type())
		}
		newKey, ok := args[1].(*String)
		if !ok {
			return newTypeError("TYPE-0012", "insertAfter", "a string (new key)", args[1].Type())
		}
		// Check existing key exists
		if _, exists := dict.Pairs[existingKey.Value]; !exists {
			return newIndexError("INDEX-0005", map[string]any{"Key": existingKey.Value})
		}
		// Check new key doesn't exist
		if _, exists := dict.Pairs[newKey.Value]; exists {
			return newStructuredError("TYPE-0023", map[string]any{"Key": newKey.Value})
		}
		return insertDictKeyAfter(dict, existingKey.Value, newKey.Value, args[2], env)

	case "insertBefore":
		// insertBefore(existingKey, newKey, value) - returns new dictionary with k/v inserted before existingKey
		if len(args) != 3 {
			return newArityError("insertBefore", len(args), 3)
		}
		existingKey, ok := args[0].(*String)
		if !ok {
			return newTypeError("TYPE-0012", "insertBefore", "a string (existing key)", args[0].Type())
		}
		newKey, ok := args[1].(*String)
		if !ok {
			return newTypeError("TYPE-0012", "insertBefore", "a string (new key)", args[1].Type())
		}
		// Check existing key exists
		if _, exists := dict.Pairs[existingKey.Value]; !exists {
			return newIndexError("INDEX-0005", map[string]any{"Key": existingKey.Value})
		}
		// Check new key doesn't exist
		if _, exists := dict.Pairs[newKey.Value]; exists {
			return newStructuredError("TYPE-0023", map[string]any{"Key": newKey.Value})
		}
		return insertDictKeyBefore(dict, existingKey.Value, newKey.Value, args[2], env)

	case "render":
		if len(args) != 1 {
			return newArityError("render", len(args), 1)
		}

		templateStr, ok := args[0].(*String)
		if !ok {
			return newTypeError("TYPE-0012", "render", "a string", args[0].Type())
		}

		renderEnv, errObj := buildRenderEnv(env, dict)
		if errObj != nil {
			return errObj
		}

		return interpolateRawString(templateStr.Value, renderEnv)

	case "toJSON":
		if len(args) != 0 {
			return newArityError("toJSON", len(args), 0)
		}
		jsonBytes, err := json.Marshal(objectToGo(dict))
		if err != nil {
			return newFormatError("FMT-0005", err)
		}
		return &String{Value: string(jsonBytes)}

	case "toBox":
		return dictToBox(dict, args, env)

	case "repr":
		if len(args) != 0 {
			return newArityError("repr", len(args), 0)
		}
		return &String{Value: objectToReprString(dict)}

	case "toHTML":
		return dictToHTML(dict, args, env)

	case "toMarkdown":
		return dictToMarkdown(dict, args, env)

	case "as":
		// dict.as(Schema) → Record
		// Implements SPEC-REC-006, SPEC-REC-007
		if len(args) != 1 {
			return newArityError("as", len(args), 1)
		}
		schema, ok := args[0].(*DSLSchema)
		if !ok {
			return newTypeError("TYPE-0001", "as", "a schema", args[0].Type())
		}
		return CreateRecord(schema, dict, env)

	case "reorder":
		return dictReorder(dict, args, env)

	default:
		// Return nil for unknown methods to allow user-defined methods to be checked
		return nil
	}
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

// booleanMethods lists all methods available on boolean
var booleanMethods = []string{"type", "toBox"}

// evalBooleanMethod evaluates a method call on a Boolean
func evalBooleanMethod(b *Boolean, method string, args []Object) Object {
	switch method {
	case "toBox":
		if len(args) != 0 {
			return newArityError("toBox", len(args), 0)
		}
		br := NewBoxRenderer()
		return &String{Value: br.RenderSingleValue(b.Inspect())}

	default:
		return unknownMethodError(method, "boolean", booleanMethods)
	}
}

// nullMethods lists all methods available on null
var nullMethods = []string{"type", "toBox"}

// evalNullMethod evaluates a method call on Null
func evalNullMethod(method string, args []Object) Object {
	switch method {
	case "toBox":
		if len(args) != 0 {
			return newArityError("toBox", len(args), 0)
		}
		br := NewBoxRenderer()
		return &String{Value: br.RenderSingleValue("null")}

	default:
		return unknownMethodError(method, "null", nullMethods)
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

	case "humanize":
		// humanize(locale?) - compact number format (1K, 1.2M, etc.)
		if len(args) > 1 {
			return newArityErrorRange("humanize", len(args), 0, 1)
		}
		localeStr := "en-US"
		if len(args) == 1 {
			loc, ok := args[0].(*String)
			if !ok {
				return newTypeError("TYPE-0012", "humanize", "a string", args[0].Type())
			}
			localeStr = loc.Value
		}
		return &String{Value: humanizeNumber(float64(num.Value), localeStr)}

	case "toBox":
		if len(args) != 0 {
			return newArityError("toBox", len(args), 0)
		}
		br := NewBoxRenderer()
		return &String{Value: br.RenderSingleValue(num.Inspect())}

	case "repr":
		if len(args) != 0 {
			return newArityError("repr", len(args), 0)
		}
		return &String{Value: objectToReprString(num)}

	case "toJSON":
		if len(args) != 0 {
			return newArityError("toJSON", len(args), 0)
		}
		return &String{Value: strconv.FormatInt(num.Value, 10)}

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

	case "humanize":
		// humanize(locale?) - compact number format (1K, 1.2M, etc.)
		if len(args) > 1 {
			return newArityErrorRange("humanize", len(args), 0, 1)
		}
		localeStr := "en-US"
		if len(args) == 1 {
			loc, ok := args[0].(*String)
			if !ok {
				return newTypeError("TYPE-0012", "humanize", "a string", args[0].Type())
			}
			localeStr = loc.Value
		}
		return &String{Value: humanizeNumber(num.Value, localeStr)}

	case "toBox":
		if len(args) != 0 {
			return newArityError("toBox", len(args), 0)
		}
		br := NewBoxRenderer()
		return &String{Value: br.RenderSingleValue(num.Inspect())}

	case "repr":
		if len(args) != 0 {
			return newArityError("repr", len(args), 0)
		}
		return &String{Value: objectToReprString(num)}

	case "toJSON":
		if len(args) != 0 {
			return newArityError("toJSON", len(args), 0)
		}
		return &String{Value: fmt.Sprintf("%g", num.Value)}

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
		// toDict() - returns clean dictionary for reconstruction (no __type)
		if len(args) != 0 {
			return newArityError("toDict", len(args), 0)
		}
		// Return dict without __type marker
		cleanPairs := make(map[string]ast.Expression)
		for key, val := range dict.Pairs {
			if key != "__type" {
				cleanPairs[key] = val
			}
		}
		return &Dictionary{Pairs: cleanPairs, Env: dict.Env}

	case "inspect":
		// inspect() - returns full dictionary with __type for debugging
		if len(args) != 0 {
			return newArityError("inspect", len(args), 0)
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

	case "toJSON":
		// toJSON() - returns ISO 8601 string in JSON format
		if len(args) != 0 {
			return newArityError("toJSON", len(args), 0)
		}
		// Get the ISO string representation and return as JSON string
		isoStr := datetimeToReprString(dict)
		// Remove the @ prefix
		isoStr = strings.TrimPrefix(isoStr, "@")
		jsonBytes, _ := json.Marshal(isoStr)
		return &String{Value: string(jsonBytes)}

	case "toBox":
		// toBox(opts?) - render datetime as ASCII box
		return datetimeToBox(dict, args, env)

	default:
		return unknownMethodError(method, "datetime", []string{
			"toDict", "inspect", "format", "year", "month", "day", "hour", "minute", "second",
			"weekday", "week", "timestamp", "toJSON", "toBox",
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
		// toDict() - returns clean dictionary for reconstruction (no __type)
		if len(args) != 0 {
			return newArityError("toDict", len(args), 0)
		}
		// Return dict without __type marker
		cleanPairs := make(map[string]ast.Expression)
		for key, val := range dict.Pairs {
			if key != "__type" {
				cleanPairs[key] = val
			}
		}
		return &Dictionary{Pairs: cleanPairs, Env: dict.Env}

	case "inspect":
		// inspect() - returns full dictionary with __type for debugging
		if len(args) != 0 {
			return newArityError("inspect", len(args), 0)
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

	case "toJSON":
		// toJSON() - returns duration as JSON object with components
		if len(args) != 0 {
			return newArityError("toJSON", len(args), 0)
		}
		// Build JSON object with duration components
		months, seconds, err := getDurationComponents(dict, env)
		if err != nil {
			return newValidationError("VAL-0007", map[string]any{"GoError": err.Error()})
		}
		// Return as object with years, months, days, hours, minutes, seconds
		years := months / 12
		months %= 12
		days := seconds / (24 * 3600)
		seconds %= (24 * 3600)
		hours := seconds / 3600
		seconds %= 3600
		minutes := seconds / 60
		seconds %= 60
		result := fmt.Sprintf(`{"years":%d,"months":%d,"days":%d,"hours":%d,"minutes":%d,"seconds":%d}`,
			years, months, days, hours, minutes, seconds)
		return &String{Value: result}

	case "toBox":
		// toBox(opts?) - render duration as ASCII box
		return durationToBox(dict, args, env)

	default:
		return unknownMethodError(method, "duration", []string{"toDict", "inspect", "format", "toJSON", "toBox"})
	}
}

// ============================================================================
// Path Methods
// ============================================================================

// evalPathMethod evaluates a method call on a path dictionary
func evalPathMethod(dict *Dictionary, method string, args []Object, env *Environment) Object {
	switch method {
	case "toDict":
		// toDict() - returns clean dictionary for reconstruction (no __type)
		if len(args) != 0 {
			return newArityError("toDict", len(args), 0)
		}
		// Return dict without __type marker
		cleanPairs := make(map[string]ast.Expression)
		for key, val := range dict.Pairs {
			if key != "__type" {
				cleanPairs[key] = val
			}
		}
		return &Dictionary{Pairs: cleanPairs, Env: dict.Env}

	case "inspect":
		// inspect() - returns full dictionary with __type for debugging
		if len(args) != 0 {
			return newArityError("inspect", len(args), 0)
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

	case "public":
		if len(args) != 0 {
			return newArityError("public", len(args), 0)
		}
		return evalPublicURL([]Object{dict}, env)

	case "toURL":
		if len(args) != 1 {
			return newArityError("toURL", len(args), 1)
		}
		prefix, ok := args[0].(*String)
		if !ok {
			return newTypeError("TYPE-0012", "toURL", "a string", args[0].Type())
		}

		pathStr := pathDictToString(dict)
		cleanPrefix := strings.TrimRight(prefix.Value, "/")
		relPath := pathStr
		if strings.HasPrefix(relPath, "./") {
			relPath = relPath[1:]
		}
		if !strings.HasPrefix(relPath, "/") {
			relPath = "/" + relPath
		}

		return &String{Value: cleanPrefix + relPath}

	case "match":
		if len(args) != 1 {
			return newArityError("match", len(args), 1)
		}
		pattern, ok := args[0].(*String)
		if !ok {
			return newTypeError("TYPE-0006", "match", "a string", args[0].Type())
		}

		pathStr := pathDictToString(dict)
		result := matchPathPattern(pathStr, pattern.Value)
		if result == nil {
			return NULL
		}

		pairs := make(map[string]ast.Expression)
		for key, val := range result {
			switch v := val.(type) {
			case string:
				pairs[key] = createLiteralExpression(&String{Value: v})
			case []string:
				elements := make([]Object, len(v))
				for i, s := range v {
					elements[i] = &String{Value: s}
				}
				pairs[key] = createLiteralExpression(&Array{Elements: elements})
			}
		}

		return &Dictionary{Pairs: pairs, Env: NewEnvironment()}

	case "toJSON":
		// toJSON() - returns path string as JSON string
		if len(args) != 0 {
			return newArityError("toJSON", len(args), 0)
		}
		pathStr := pathDictToString(dict)
		jsonBytes, _ := json.Marshal(pathStr)
		return &String{Value: string(jsonBytes)}

	case "toBox":
		// toBox(opts?) - render path as ASCII box
		return pathToBox(dict, args, env)

	default:
		return unknownMethodError(method, "path", []string{
			"toDict", "inspect", "toString", "join", "parent", "isAbsolute", "isRelative", "public", "toURL", "match", "toJSON", "toBox",
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
		// toDict() - returns clean dictionary for reconstruction (no __type)
		if len(args) != 0 {
			return newArityError("toDict", len(args), 0)
		}
		// Return dict without __type marker
		cleanPairs := make(map[string]ast.Expression)
		for key, val := range dict.Pairs {
			if key != "__type" {
				cleanPairs[key] = val
			}
		}
		return &Dictionary{Pairs: cleanPairs, Env: dict.Env}

	case "inspect":
		// inspect() - returns full dictionary with __type for debugging
		if len(args) != 0 {
			return newArityError("inspect", len(args), 0)
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

	case "toJSON":
		// toJSON() - returns URL string as JSON string
		if len(args) != 0 {
			return newArityError("toJSON", len(args), 0)
		}
		urlStr := urlDictToString(dict)
		jsonBytes, _ := json.Marshal(urlStr)
		return &String{Value: string(jsonBytes)}

	case "toBox":
		// toBox(opts?) - render URL as ASCII box
		return urlToBox(dict, args, env)

	default:
		return unknownMethodError(method, "url", []string{
			"toDict", "inspect", "toString", "query", "href", "toJSON", "toBox",
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
		// toDict() - returns clean dictionary for reconstruction (no __type)
		if len(args) != 0 {
			return newArityError("toDict", len(args), 0)
		}
		// Return dict without __type marker
		cleanPairs := make(map[string]ast.Expression)
		for key, val := range dict.Pairs {
			if key != "__type" {
				cleanPairs[key] = val
			}
		}
		return &Dictionary{Pairs: cleanPairs, Env: dict.Env}

	case "inspect":
		// inspect() - returns full dictionary with __type for debugging
		if len(args) != 0 {
			return newArityError("inspect", len(args), 0)
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

	case "replace":
		// replace(string, replacement) - replace matches in string
		// replacement can be a string or function
		if len(args) != 2 {
			return newArityError("replace", len(args), 2)
		}
		str, ok := args[0].(*String)
		if !ok {
			return newTypeError("TYPE-0012", "replace", "a string", args[0].Type())
		}
		return regexReplaceOnString(str.Value, dict, args[1], env)

	case "toJSON":
		// toJSON() - returns regex as JSON object with pattern and flags
		if len(args) != 0 {
			return newArityError("toJSON", len(args), 0)
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
		// Return as JSON object
		patternJSON, _ := json.Marshal(pattern)
		flagsJSON, _ := json.Marshal(flags)
		return &String{Value: fmt.Sprintf(`{"pattern":%s,"flags":%s}`, patternJSON, flagsJSON)}

	case "toBox":
		// toBox(opts?) - render regex as ASCII box
		return regexToBox(dict, args, env)

	default:
		return unknownMethodError(method, "regex", []string{
			"toDict", "inspect", "toString", "test", "exec", "execAll", "matches", "replace", "toJSON", "toBox",
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
		// toDict() - returns clean dictionary for reconstruction (no __type)
		if len(args) != 0 {
			return newArityError("toDict", len(args), 0)
		}
		// Return dict without __type marker
		cleanPairs := make(map[string]ast.Expression)
		for key, val := range dict.Pairs {
			if key != "__type" {
				cleanPairs[key] = val
			}
		}
		return &Dictionary{Pairs: cleanPairs, Env: dict.Env}

	case "inspect":
		// inspect() - returns full dictionary with __type for debugging
		if len(args) != 0 {
			return newArityError("inspect", len(args), 0)
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
		// toDict() - returns clean dictionary for reconstruction (no __type)
		if len(args) != 0 {
			return newArityError("toDict", len(args), 0)
		}
		// Return dict without __type marker
		cleanPairs := make(map[string]ast.Expression)
		for key, val := range dict.Pairs {
			if key != "__type" {
				cleanPairs[key] = val
			}
		}
		return &Dictionary{Pairs: cleanPairs, Env: dict.Env}

	case "inspect":
		// inspect() - returns full dictionary with __type for debugging
		if len(args) != 0 {
			return newArityError("inspect", len(args), 0)
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
	"format", "abs", "split", "toJSON", "toBox", "repr", "toDict", "inspect",
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
		// Check if it's a method name - provide helpful error
		if slices.Contains(moneyMethods, key) {
			return methodAsPropertyError(key, "Money")
		}
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

	case "toJSON":
		// toJSON() - returns money as JSON object with amount (as string to preserve precision) and currency
		if len(args) != 0 {
			return newArityError("toJSON", len(args), 0)
		}
		// Format the amount as a decimal string to preserve precision
		amountStr := money.formatAmount()
		currencyJSON, _ := json.Marshal(money.Currency)
		return &String{Value: fmt.Sprintf(`{"amount":"%s","currency":%s}`, amountStr, currencyJSON)}

	case "toBox":
		// toBox(opts?) - render money as ASCII box
		return moneyToBox(money, args)

	case "repr":
		// repr() - returns Parsley-parseable literal (e.g., "$50.00")
		if len(args) != 0 {
			return newArityError("repr", len(args), 0)
		}
		return &String{Value: money.Inspect()}

	case "toDict":
		// toDict() - returns clean dictionary for reconstruction via money(dict)
		if len(args) != 0 {
			return newArityError("toDict", len(args), 0)
		}
		// Calculate user-friendly amount (e.g., 50.00 not 5000)
		divisor := math.Pow10(int(money.Scale))
		amount := float64(money.Amount) / divisor
		return &Dictionary{
			Pairs: map[string]ast.Expression{
				"amount":   createLiteralExpression(&Float{Value: amount}),
				"currency": createLiteralExpression(&String{Value: money.Currency}),
			},
			Env: NewEnvironment(),
		}

	case "inspect":
		// inspect() - returns debug dictionary with __type and raw internal values
		if len(args) != 0 {
			return newArityError("inspect", len(args), 0)
		}
		return &Dictionary{
			Pairs: map[string]ast.Expression{
				"__type":   createLiteralExpression(&String{Value: "money"}),
				"amount":   createLiteralExpression(&Integer{Value: money.Amount}),
				"currency": createLiteralExpression(&String{Value: money.Currency}),
				"scale":    createLiteralExpression(&Integer{Value: int64(money.Scale)}),
			},
			Env: NewEnvironment(),
		}

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

	// Split on blank lines (one or more consecutive newlines)
	// \n\n+ means two or more newlines
	paragraphPattern := regexp.MustCompile(`\n\n+`)
	paragraphs := paragraphPattern.Split(s, -1)

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
			result.WriteString(fmt.Sprintf("%d. ", i+1))
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
