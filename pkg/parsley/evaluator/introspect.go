package evaluator

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
)

// ============================================================================
// Method Metadata
// ============================================================================

// MethodInfo holds metadata about a method
type MethodInfo struct {
	Name        string
	Arity       string // e.g., "0", "1", "0-1", "1+"
	Description string
}

// TypeMethods maps type names to their available methods
var TypeMethods = map[string][]MethodInfo{
	"string": {
		{Name: "toUpper", Arity: "0", Description: "Convert to uppercase"},
		{Name: "toLower", Arity: "0", Description: "Convert to lowercase"},
		{Name: "trim", Arity: "0", Description: "Remove leading/trailing whitespace"},
		{Name: "split", Arity: "1", Description: "Split by delimiter into array"},
		{Name: "replace", Arity: "2", Description: "Replace all occurrences"},
		{Name: "length", Arity: "0", Description: "Get character count"},
		{Name: "includes", Arity: "1", Description: "Check if contains substring"},
		{Name: "highlight", Arity: "1-2", Description: "Wrap matches in HTML tag"},
		{Name: "paragraphs", Arity: "0", Description: "Convert blank lines to <p> tags"},
		{Name: "render", Arity: "0-1", Description: "Interpolate template with values"},
		{Name: "parseJSON", Arity: "0", Description: "Parse as JSON"},
		{Name: "parseCSV", Arity: "0-1", Description: "Parse as CSV"},
		{Name: "collapse", Arity: "0", Description: "Collapse whitespace to single spaces"},
		{Name: "normalizeSpace", Arity: "0", Description: "Collapse and trim whitespace"},
		{Name: "stripSpace", Arity: "0", Description: "Remove all whitespace"},
		{Name: "stripHtml", Arity: "0", Description: "Remove HTML tags"},
		{Name: "digits", Arity: "0", Description: "Extract only digits"},
		{Name: "slug", Arity: "0", Description: "Convert to URL-safe slug"},
	},
	"array": {
		{Name: "length", Arity: "0", Description: "Get element count"},
		{Name: "reverse", Arity: "0", Description: "Reverse order"},
		{Name: "push", Arity: "1", Description: "Add element to end"},
		{Name: "pop", Arity: "0", Description: "Remove and return last element"},
		{Name: "shift", Arity: "0", Description: "Remove and return first element"},
		{Name: "unshift", Arity: "1", Description: "Add element to beginning"},
		{Name: "slice", Arity: "1-2", Description: "Extract a section"},
		{Name: "concat", Arity: "1", Description: "Concatenate arrays"},
		{Name: "includes", Arity: "1", Description: "Check if contains element"},
		{Name: "indexOf", Arity: "1", Description: "Find index of element"},
		{Name: "join", Arity: "0-1", Description: "Join elements into string"},
		{Name: "sort", Arity: "0", Description: "Sort elements"},
		{Name: "first", Arity: "0", Description: "Get first element"},
		{Name: "last", Arity: "0", Description: "Get last element"},
		{Name: "map", Arity: "1", Description: "Transform each element"},
		{Name: "filter", Arity: "1", Description: "Filter by predicate"},
		{Name: "reduce", Arity: "2", Description: "Reduce to single value"},
		{Name: "unique", Arity: "0", Description: "Remove duplicates"},
		{Name: "flatten", Arity: "0", Description: "Flatten nested arrays"},
		{Name: "find", Arity: "1", Description: "Find first matching element"},
		{Name: "findIndex", Arity: "1", Description: "Find index of first match"},
		{Name: "every", Arity: "1", Description: "Check if all match predicate"},
		{Name: "some", Arity: "1", Description: "Check if any match predicate"},
		{Name: "groupBy", Arity: "1", Description: "Group elements by key function"},
		{Name: "count", Arity: "0-1", Description: "Count elements or matches"},
		{Name: "countBy", Arity: "1", Description: "Count by key function"},
		{Name: "maxBy", Arity: "1", Description: "Find max by key function"},
		{Name: "minBy", Arity: "1", Description: "Find min by key function"},
		{Name: "sortBy", Arity: "1", Description: "Sort by key function"},
		{Name: "take", Arity: "1", Description: "Take first n elements"},
		{Name: "skip", Arity: "1", Description: "Skip first n elements"},
		{Name: "zip", Arity: "1+", Description: "Combine arrays element-wise"},
		{Name: "insert", Arity: "2", Description: "Insert at index"},
		{Name: "toJSON", Arity: "0", Description: "Convert to JSON string"},
		{Name: "toCSV", Arity: "0-1", Description: "Convert to CSV string"},
	},
	"integer": {
		{Name: "abs", Arity: "0", Description: "Absolute value"},
		{Name: "format", Arity: "0-1", Description: "Format with locale"},
		{Name: "humanize", Arity: "0", Description: "Human-readable format (1K, 1M)"},
	},
	"float": {
		{Name: "abs", Arity: "0", Description: "Absolute value"},
		{Name: "format", Arity: "0-2", Description: "Format with decimals and locale"},
		{Name: "round", Arity: "0-1", Description: "Round to n decimals"},
		{Name: "floor", Arity: "0", Description: "Round down"},
		{Name: "ceil", Arity: "0", Description: "Round up"},
		{Name: "humanize", Arity: "0", Description: "Human-readable format (1K, 1M)"},
	},
	"dictionary": {
		{Name: "keys", Arity: "0", Description: "Get all keys"},
		{Name: "values", Arity: "0", Description: "Get all values"},
		{Name: "entries", Arity: "0", Description: "Get [key, value] pairs"},
		{Name: "has", Arity: "1", Description: "Check if key exists"},
		{Name: "delete", Arity: "1", Description: "Remove key"},
		{Name: "insertAfter", Arity: "2", Description: "Insert after key"},
		{Name: "insertBefore", Arity: "2", Description: "Insert before key"},
		{Name: "render", Arity: "0-1", Description: "Render template with values"},
		{Name: "toJSON", Arity: "0", Description: "Convert to JSON string"},
	},
	"money": {
		{Name: "format", Arity: "0-1", Description: "Format with locale"},
		{Name: "abs", Arity: "0", Description: "Absolute value"},
		{Name: "negate", Arity: "0", Description: "Negate amount"},
		{Name: "toDict", Arity: "0", Description: "Convert to dictionary"},
	},
	"datetime": {
		{Name: "format", Arity: "0-2", Description: "Format with style and locale"},
		{Name: "dayOfYear", Arity: "0", Description: "Day of year (1-366)"},
		{Name: "week", Arity: "0", Description: "ISO week number"},
		{Name: "timestamp", Arity: "0", Description: "Unix timestamp"},
		{Name: "toDict", Arity: "0", Description: "Convert to dictionary"},
	},
	"duration": {
		{Name: "format", Arity: "0-1", Description: "Format as relative time"},
		{Name: "toDict", Arity: "0", Description: "Convert to dictionary"},
	},
	"path": {
		{Name: "toString", Arity: "0", Description: "Convert to string"},
		{Name: "join", Arity: "1+", Description: "Join path components"},
		{Name: "parent", Arity: "0", Description: "Get parent directory"},
		{Name: "isAbsolute", Arity: "0", Description: "Check if absolute path"},
		{Name: "isRelative", Arity: "0", Description: "Check if relative path"},
		{Name: "public", Arity: "0", Description: "Get public URL"},
		{Name: "toURL", Arity: "1", Description: "Convert to URL with prefix"},
		{Name: "match", Arity: "1", Description: "Match against pattern"},
		{Name: "toDict", Arity: "0", Description: "Convert to dictionary"},
	},
	"url": {
		{Name: "origin", Arity: "0", Description: "Get origin (scheme://host:port)"},
		{Name: "pathname", Arity: "0", Description: "Get path component"},
		{Name: "toString", Arity: "0", Description: "Convert to string"},
		{Name: "withPath", Arity: "1", Description: "Create URL with new path"},
		{Name: "withQuery", Arity: "1", Description: "Create URL with query params"},
		{Name: "toDict", Arity: "0", Description: "Convert to dictionary"},
	},
	"regex": {
		{Name: "test", Arity: "1", Description: "Test if string matches"},
		{Name: "match", Arity: "1", Description: "Find first match"},
		{Name: "matchAll", Arity: "1", Description: "Find all matches"},
		{Name: "replace", Arity: "2", Description: "Replace matches"},
		{Name: "split", Arity: "1", Description: "Split string by pattern"},
		{Name: "toDict", Arity: "0", Description: "Convert to dictionary"},
	},
	"file": {
		{Name: "exists", Arity: "0", Description: "Check if file exists"},
		{Name: "read", Arity: "0", Description: "Read file contents"},
		{Name: "stat", Arity: "0", Description: "Get file metadata"},
		{Name: "toDict", Arity: "0", Description: "Convert to dictionary"},
	},
	"directory": {
		{Name: "exists", Arity: "0", Description: "Check if directory exists"},
		{Name: "list", Arity: "0", Description: "List directory contents"},
		{Name: "toDict", Arity: "0", Description: "Convert to dictionary"},
	},
	"function": {
		// Functions have no methods but we include them for completeness
	},
	"boolean": {
		// Booleans have no methods
	},
	"null": {
		// Null has no methods
	},
}

// ============================================================================
// Type Detection
// ============================================================================

// getObjectTypeName returns the type name and subtype (for typed dicts) of an object
func getObjectTypeName(obj Object, env *Environment) (typeName string, subType string) {
	switch o := obj.(type) {
	case *String:
		return "string", ""
	case *Integer:
		return "integer", ""
	case *Float:
		return "float", ""
	case *Boolean:
		return "boolean", ""
	case *Array:
		return "array", ""
	case *Function:
		return "function", ""
	case *Builtin:
		return "builtin", ""
	case *Money:
		return "money", ""
	case *Dictionary:
		// Check for typed dictionaries
		if isDatetimeDict(o) {
			return "dictionary", "datetime"
		}
		if isDurationDict(o) {
			return "dictionary", "duration"
		}
		if isPathDict(o) {
			return "dictionary", "path"
		}
		if isUrlDict(o) {
			return "dictionary", "url"
		}
		if isRegexDict(o) {
			return "dictionary", "regex"
		}
		if isFileDict(o) {
			return "dictionary", "file"
		}
		if isDirDict(o) {
			return "dictionary", "directory"
		}
		if isRequestDict(o) {
			return "dictionary", "request"
		}
		if isResponseDict(o) {
			return "dictionary", "response"
		}
		return "dictionary", ""
	case *Null:
		return "null", ""
	case *Error:
		return "error", ""
	default:
		return strings.ToLower(string(obj.Type())), ""
	}
}

// ============================================================================
// Inspect Function
// ============================================================================

// builtinInspect returns introspection data as a dictionary
func builtinInspect(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("inspect", len(args), 1)
	}

	obj := args[0]
	typeName, subType := getObjectTypeName(obj, nil)

	// Determine which method list to use
	methodKey := typeName
	if subType != "" {
		methodKey = subType
	}

	// Build methods array
	methodInfos, ok := TypeMethods[methodKey]
	if !ok {
		methodInfos = []MethodInfo{}
	}

	// Sort methods alphabetically
	sortedMethods := make([]MethodInfo, len(methodInfos))
	copy(sortedMethods, methodInfos)
	sort.Slice(sortedMethods, func(i, j int) bool {
		return sortedMethods[i].Name < sortedMethods[j].Name
	})

	// Build method dictionaries
	methodDicts := make([]Object, len(sortedMethods))
	for i, m := range sortedMethods {
		pairs := map[string]ast.Expression{
			"name":        createLiteralExpression(&String{Value: m.Name}),
			"arity":       createLiteralExpression(&String{Value: m.Arity}),
			"description": createLiteralExpression(&String{Value: m.Description}),
		}
		methodDicts[i] = &Dictionary{Pairs: pairs, Env: NewEnvironment()}
	}

	// Build result dictionary
	pairs := map[string]ast.Expression{
		"type":    createLiteralExpression(&String{Value: typeName}),
		"methods": createLiteralExpression(&Array{Elements: methodDicts}),
	}

	// Add subtype if present
	if subType != "" {
		pairs["subtype"] = createLiteralExpression(&String{Value: subType})
	}

	// For functions, add parameter info
	if fn, ok := obj.(*Function); ok {
		params := make([]Object, len(fn.Params))
		for i, p := range fn.Params {
			params[i] = &String{Value: p.String()}
		}
		pairs["params"] = createLiteralExpression(&Array{Elements: params})
	}

	// For dictionaries, add keys
	if dict, ok := obj.(*Dictionary); ok && subType == "" {
		keys := dict.Keys()
		keyObjs := make([]Object, len(keys))
		for i, k := range keys {
			keyObjs[i] = &String{Value: k}
		}
		pairs["keys"] = createLiteralExpression(&Array{Elements: keyObjs})
	}

	return &Dictionary{Pairs: pairs, Env: NewEnvironment()}
}

// ============================================================================
// Describe Function (Pretty Print)
// ============================================================================

// builtinDescribe pretty prints introspection data
func builtinDescribe(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("describe", len(args), 1)
	}

	obj := args[0]
	typeName, subType := getObjectTypeName(obj, nil)

	var sb strings.Builder

	// Type header
	if subType != "" {
		sb.WriteString(fmt.Sprintf("Type: %s (%s)\n", subType, typeName))
	} else {
		sb.WriteString(fmt.Sprintf("Type: %s\n", typeName))
	}

	// For functions, show parameters
	if fn, ok := obj.(*Function); ok {
		sb.WriteString("Parameters: ")
		if len(fn.Params) == 0 {
			sb.WriteString("(none)")
		} else {
			params := make([]string, len(fn.Params))
			for i, p := range fn.Params {
				params[i] = p.String()
			}
			sb.WriteString(strings.Join(params, ", "))
		}
		sb.WriteString("\n")
	}

	// For dictionaries without subtype, show keys
	if dict, ok := obj.(*Dictionary); ok && subType == "" {
		keys := dict.Keys()
		sb.WriteString(fmt.Sprintf("Keys: %s\n", strings.Join(keys, ", ")))
	}

	// Methods
	methodKey := typeName
	if subType != "" {
		methodKey = subType
	}

	methodInfos, ok := TypeMethods[methodKey]
	if !ok || len(methodInfos) == 0 {
		sb.WriteString("Methods: (none)\n")
	} else {
		sb.WriteString("\nMethods:\n")

		// Sort methods alphabetically
		sortedMethods := make([]MethodInfo, len(methodInfos))
		copy(sortedMethods, methodInfos)
		sort.Slice(sortedMethods, func(i, j int) bool {
			return sortedMethods[i].Name < sortedMethods[j].Name
		})

		// Find max name length for alignment
		maxNameLen := 0
		for _, m := range sortedMethods {
			nameWithArity := fmt.Sprintf(".%s(%s)", m.Name, arityToParams(m.Arity))
			if len(nameWithArity) > maxNameLen {
				maxNameLen = len(nameWithArity)
			}
		}

		for _, m := range sortedMethods {
			nameWithArity := fmt.Sprintf(".%s(%s)", m.Name, arityToParams(m.Arity))
			padding := strings.Repeat(" ", maxNameLen-len(nameWithArity)+2)
			sb.WriteString(fmt.Sprintf("  %s%s- %s\n", nameWithArity, padding, m.Description))
		}
	}

	return &String{Value: sb.String()}
}

// arityToParams converts arity string to parameter placeholder
func arityToParams(arity string) string {
	switch arity {
	case "0":
		return ""
	case "1":
		return "arg"
	case "2":
		return "arg1, arg2"
	case "0-1":
		return "arg?"
	case "0-2":
		return "arg1?, arg2?"
	case "1-2":
		return "arg1, arg2?"
	case "1+":
		return "arg1, ..."
	default:
		return "..."
	}
}
