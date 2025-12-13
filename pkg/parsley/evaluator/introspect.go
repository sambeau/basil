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
	"table": {
		{Name: "where", Arity: "1", Description: "Filter rows by predicate"},
		{Name: "orderBy", Arity: "1+", Description: "Sort rows by column(s)"},
		{Name: "select", Arity: "1+", Description: "Select specific columns"},
		{Name: "limit", Arity: "1-2", Description: "Limit rows (count, offset?)"},
		{Name: "count", Arity: "0", Description: "Count rows"},
		{Name: "sum", Arity: "1", Description: "Sum column values"},
		{Name: "avg", Arity: "1", Description: "Average column values"},
		{Name: "min", Arity: "1", Description: "Minimum column value"},
		{Name: "max", Arity: "1", Description: "Maximum column value"},
		{Name: "toHTML", Arity: "0-1", Description: "Convert to HTML table (footer?)"},
		{Name: "toCSV", Arity: "0", Description: "Convert to CSV string"},
		{Name: "toMarkdown", Arity: "0", Description: "Convert to Markdown table"},
		{Name: "appendRow", Arity: "1", Description: "Add row at end"},
		{Name: "insertRowAt", Arity: "2", Description: "Insert row at index"},
		{Name: "appendCol", Arity: "2", Description: "Add column at end"},
		{Name: "insertColAfter", Arity: "3", Description: "Insert column after another"},
		{Name: "insertColBefore", Arity: "3", Description: "Insert column before another"},
		{Name: "rowCount", Arity: "0", Description: "Get number of rows"},
		{Name: "columnCount", Arity: "0", Description: "Get number of columns"},
		{Name: "column", Arity: "1", Description: "Get array of values from column"},
	},
	"dbconnection": {
		{Name: "begin", Arity: "0", Description: "Begin transaction"},
		{Name: "commit", Arity: "0", Description: "Commit transaction"},
		{Name: "rollback", Arity: "0", Description: "Rollback transaction"},
		{Name: "close", Arity: "0", Description: "Close connection"},
		{Name: "ping", Arity: "0", Description: "Test connection"},
	},
	"sftpconnection": {
		{Name: "close", Arity: "0", Description: "Close connection"},
	},
	"sftpfile": {
		{Name: "mkdir", Arity: "0-1", Description: "Create directory"},
		{Name: "rmdir", Arity: "0-1", Description: "Remove directory"},
		{Name: "remove", Arity: "0", Description: "Remove file"},
	},
	"session": {
		{Name: "get", Arity: "1-2", Description: "Get session value (key, default?)"},
		{Name: "set", Arity: "2", Description: "Set session value (key, value)"},
		{Name: "delete", Arity: "1", Description: "Delete session key"},
		{Name: "has", Arity: "1", Description: "Check if key exists"},
		{Name: "clear", Arity: "0", Description: "Clear all session data"},
		{Name: "all", Arity: "0", Description: "Get all session data"},
		{Name: "flash", Arity: "2", Description: "Set flash message (key, value)"},
		{Name: "getFlash", Arity: "1", Description: "Get and clear flash message"},
		{Name: "getAllFlash", Arity: "0", Description: "Get all flash messages"},
		{Name: "hasFlash", Arity: "0", Description: "Check if flash messages exist"},
		{Name: "regenerate", Arity: "0", Description: "Regenerate session ID"},
	},
	"dev": {
		{Name: "log", Arity: "1-3", Description: "Log value to dev panel"},
		{Name: "clearLog", Arity: "0", Description: "Clear dev log"},
		{Name: "logPage", Arity: "0-1", Description: "Log page content"},
		{Name: "setLogRoute", Arity: "1", Description: "Set log route pattern"},
		{Name: "clearLogPage", Arity: "0", Description: "Clear page log"},
	},
	"tablemodule": {
		{Name: "fromDict", Arity: "1", Description: "Create table from dictionary"},
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
	case *Table:
		return "table", ""
	case *DBConnection:
		return "dbconnection", ""
	case *SFTPConnection:
		return "sftpconnection", ""
	case *SFTPFileHandle:
		return "sftpfile", ""
	case *SessionModule:
		return "session", ""
	case *DevModule:
		return "dev", ""
	case *TableModule:
		return "tablemodule", ""
	case *StdlibRoot:
		return "stdlib", ""
	case *StdlibModuleDict:
		return "module", ""
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

	// Special handling for StdlibRoot - show available modules
	if root, ok := obj.(*StdlibRoot); ok {
		return inspectStdlibRoot(root)
	}

	// Special handling for StdlibModuleDict - show exports
	if mod, ok := obj.(*StdlibModuleDict); ok {
		return inspectStdlibModule(mod)
	}

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

// inspectStdlibModule returns introspection data for a stdlib module
func inspectStdlibModule(mod *StdlibModuleDict) Object {
	// Build exports array - sorted list of {name, type, description?}
	keys := make([]string, 0, len(mod.Exports))
	for k := range mod.Exports {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	exports := make([]Object, len(keys))
	for i, name := range keys {
		obj := mod.Exports[name]
		exportType := strings.ToLower(string(obj.Type()))

		pairs := map[string]ast.Expression{
			"name": createLiteralExpression(&String{Value: name}),
			"type": createLiteralExpression(&String{Value: exportType}),
		}

		// Check if we have metadata for this export
		if info, ok := StdlibExports[name]; ok {
			pairs["arity"] = createLiteralExpression(&String{Value: info.Arity})
			pairs["description"] = createLiteralExpression(&String{Value: info.Description})
		}

		exports[i] = &Dictionary{Pairs: pairs, Env: NewEnvironment()}
	}

	// Build result dictionary
	pairs := map[string]ast.Expression{
		"type":    createLiteralExpression(&String{Value: "module"}),
		"exports": createLiteralExpression(&Array{Elements: exports}),
	}

	return &Dictionary{Pairs: pairs, Env: NewEnvironment()}
}

// inspectStdlibRoot returns introspection data for the stdlib root
func inspectStdlibRoot(root *StdlibRoot) Object {
	// Build modules array
	modules := make([]Object, len(root.Modules))
	for i, name := range root.Modules {
		info, hasInfo := StdlibModuleDescriptions[name]
		pairs := map[string]ast.Expression{
			"name": createLiteralExpression(&String{Value: name}),
		}
		if hasInfo {
			pairs["description"] = createLiteralExpression(&String{Value: info})
		}
		modules[i] = &Dictionary{Pairs: pairs, Env: NewEnvironment()}
	}

	// Build result dictionary
	pairs := map[string]ast.Expression{
		"type":    createLiteralExpression(&String{Value: "stdlib"}),
		"modules": createLiteralExpression(&Array{Elements: modules}),
	}

	return &Dictionary{Pairs: pairs, Env: NewEnvironment()}
}

// describeStdlibRoot pretty prints the stdlib module listing
func describeStdlibRoot(root *StdlibRoot) Object {
	var sb strings.Builder
	sb.WriteString("Parsley Standard Library (@std)\n\n")
	sb.WriteString("Available modules:\n")

	// Find max name length for alignment
	maxNameLen := 0
	for _, name := range root.Modules {
		if len(name) > maxNameLen {
			maxNameLen = len(name)
		}
	}

	for _, name := range root.Modules {
		padding := strings.Repeat(" ", maxNameLen-len(name)+2)
		if desc, ok := StdlibModuleDescriptions[name]; ok {
			sb.WriteString(fmt.Sprintf("  @std/%s%s- %s\n", name, padding, desc))
		} else {
			sb.WriteString(fmt.Sprintf("  @std/%s\n", name))
		}
	}

	sb.WriteString("\nUsage: import @std/<module>\n")
	sb.WriteString("Example: let { floor, ceil } = import @std/math\n")

	return &String{Value: sb.String()}
}

// StdlibModuleDescriptions contains descriptions for each stdlib module
var StdlibModuleDescriptions = map[string]string{
	"api":    "HTTP client for API requests",
	"basil":  "Basil server context (server-only)",
	"dev":    "Development tools (logging, debugging)",
	"id":     "ID generation (UUID, nanoid, etc.)",
	"math":   "Mathematical functions and constants",
	"schema": "Schema validation and type checking",
	"table":  "Table data structure with query methods",
	"valid":  "Validation functions for strings, numbers, formats",
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

	// Special handling for StdlibRoot - show available modules
	if root, ok := obj.(*StdlibRoot); ok {
		return describeStdlibRoot(root)
	}

	// Special handling for StdlibModuleDict - show exports
	if mod, ok := obj.(*StdlibModuleDict); ok {
		return describeStdlibModule(mod)
	}

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

// describeStdlibModule pretty prints a stdlib module's exports
func describeStdlibModule(mod *StdlibModuleDict) Object {
	var sb strings.Builder
	sb.WriteString("Type: module\n\nExports:\n")

	// Sort exports alphabetically
	keys := make([]string, 0, len(mod.Exports))
	for k := range mod.Exports {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Group by type (constants vs functions)
	var constants []string
	var functions []string
	for _, name := range keys {
		obj := mod.Exports[name]
		switch obj.(type) {
		case *Builtin, *Function:
			functions = append(functions, name)
		default:
			constants = append(constants, name)
		}
	}

	// Find max name length for alignment
	maxNameLen := 0
	for _, name := range keys {
		info, hasInfo := StdlibExports[name]
		var display string
		if hasInfo && info.Arity != "" {
			display = fmt.Sprintf("%s(%s)", name, arityToParams(info.Arity))
		} else {
			display = name
		}
		if len(display) > maxNameLen {
			maxNameLen = len(display)
		}
	}

	// Print constants first
	if len(constants) > 0 {
		sb.WriteString("  Constants:\n")
		for _, name := range constants {
			obj := mod.Exports[name]
			padding := strings.Repeat(" ", maxNameLen-len(name)+2)
			sb.WriteString(fmt.Sprintf("    %s%s= %s\n", name, padding, obj.Inspect()))
		}
		sb.WriteString("\n")
	}

	// Print functions
	if len(functions) > 0 {
		sb.WriteString("  Functions:\n")
		for _, name := range functions {
			info, hasInfo := StdlibExports[name]
			var display string
			var desc string
			if hasInfo {
				display = fmt.Sprintf("%s(%s)", name, arityToParams(info.Arity))
				desc = info.Description
			} else {
				display = name + "(...)"
				desc = ""
			}
			padding := strings.Repeat(" ", maxNameLen-len(display)+2)
			if desc != "" {
				sb.WriteString(fmt.Sprintf("    %s%s- %s\n", display, padding, desc))
			} else {
				sb.WriteString(fmt.Sprintf("    %s\n", display))
			}
		}
	}

	return &String{Value: sb.String()}
}

// ============================================================================
// Stdlib Export Metadata
// ============================================================================

// StdlibExports contains metadata for stdlib module exports
var StdlibExports = map[string]MethodInfo{
	// math module - Constants
	"PI":  {Arity: "", Description: "Pi (3.14159...)"},
	"E":   {Arity: "", Description: "Euler's number (2.71828...)"},
	"TAU": {Arity: "", Description: "Tau (2*Pi)"},

	// math module - Rounding
	"floor": {Arity: "1", Description: "Round down to integer"},
	"ceil":  {Arity: "1", Description: "Round up to integer"},
	"round": {Arity: "1-2", Description: "Round to nearest (decimals?)"},
	"trunc": {Arity: "1", Description: "Truncate to integer"},

	// math module - Comparison & Clamping
	"abs":   {Arity: "1", Description: "Absolute value"},
	"sign":  {Arity: "1", Description: "Sign (-1, 0, or 1)"},
	"clamp": {Arity: "3", Description: "Clamp value between min and max"},

	// math module - Aggregation
	"min":     {Arity: "1+", Description: "Minimum of values or array"},
	"max":     {Arity: "1+", Description: "Maximum of values or array"},
	"sum":     {Arity: "1+", Description: "Sum of values or array"},
	"avg":     {Arity: "1+", Description: "Average of values or array"},
	"mean":    {Arity: "1+", Description: "Mean (alias for avg)"},
	"product": {Arity: "1+", Description: "Product of values or array"},
	"count":   {Arity: "1", Description: "Count elements in array"},

	// math module - Statistics
	"median":   {Arity: "1", Description: "Median of array"},
	"mode":     {Arity: "1", Description: "Mode of array"},
	"stddev":   {Arity: "1", Description: "Standard deviation"},
	"variance": {Arity: "1", Description: "Variance"},
	"range":    {Arity: "1", Description: "Range (max - min)"},

	// math module - Random
	"random":    {Arity: "0", Description: "Random float 0-1"},
	"randomInt": {Arity: "1-2", Description: "Random int (max) or (min, max)"},
	"seed":      {Arity: "1", Description: "Seed random generator"},

	// math module - Powers & Logarithms
	"sqrt":  {Arity: "1", Description: "Square root"},
	"pow":   {Arity: "2", Description: "Power (base, exponent)"},
	"exp":   {Arity: "1", Description: "e^x"},
	"log":   {Arity: "1", Description: "Natural logarithm"},
	"log10": {Arity: "1", Description: "Base-10 logarithm"},

	// math module - Trigonometry
	"sin":   {Arity: "1", Description: "Sine (radians)"},
	"cos":   {Arity: "1", Description: "Cosine (radians)"},
	"tan":   {Arity: "1", Description: "Tangent (radians)"},
	"asin":  {Arity: "1", Description: "Arc sine"},
	"acos":  {Arity: "1", Description: "Arc cosine"},
	"atan":  {Arity: "1", Description: "Arc tangent"},
	"atan2": {Arity: "2", Description: "Arc tangent of y/x"},

	// math module - Angular Conversion
	"degrees": {Arity: "1", Description: "Radians to degrees"},
	"radians": {Arity: "1", Description: "Degrees to radians"},

	// math module - Geometry & Interpolation
	"hypot": {Arity: "2", Description: "Hypotenuse length"},
	"dist":  {Arity: "4", Description: "Distance between points"},
	"lerp":  {Arity: "3", Description: "Linear interpolation"},
	"map":   {Arity: "5", Description: "Map value from one range to another"},

	// valid module - Type validators
	"string":  {Arity: "1", Description: "Check if value is string"},
	"number":  {Arity: "1", Description: "Check if value is number"},
	"integer": {Arity: "1", Description: "Check if value is integer"},
	"boolean": {Arity: "1", Description: "Check if value is boolean"},
	"array":   {Arity: "1", Description: "Check if value is array"},
	"dict":    {Arity: "1", Description: "Check if value is dictionary"},

	// valid module - String validators
	"empty":        {Arity: "1", Description: "Check if string is empty"},
	"minLen":       {Arity: "2", Description: "Check minimum length"},
	"maxLen":       {Arity: "2", Description: "Check maximum length"},
	"length":       {Arity: "2-3", Description: "Check length (exact or range)"},
	"matches":      {Arity: "2", Description: "Check regex match"},
	"alpha":        {Arity: "1", Description: "Check if only letters"},
	"alphanumeric": {Arity: "1", Description: "Check if only letters/numbers"},
	"numeric":      {Arity: "1", Description: "Check if only digits"},

	// valid module - Number validators
	// "min" and "max" already defined in math
	"between":  {Arity: "3", Description: "Check if number in range"},
	"positive": {Arity: "1", Description: "Check if positive"},
	"negative": {Arity: "1", Description: "Check if negative"},

	// valid module - Format validators
	"email":      {Arity: "1", Description: "Check email format"},
	"url":        {Arity: "1", Description: "Check URL format"},
	"uuid":       {Arity: "1", Description: "Check UUID format"},
	"phone":      {Arity: "1-2", Description: "Check phone format (locale?)"},
	"creditCard": {Arity: "1", Description: "Check credit card format"},
	"date":       {Arity: "1-2", Description: "Check date format"},
	"time":       {Arity: "1", Description: "Check time format"},

	// valid module - Locale-aware validators
	"postalCode": {Arity: "1-2", Description: "Check postal code (locale?)"},
	"parseDate":  {Arity: "1-2", Description: "Parse date string (locale?)"},

	// valid module - Collection validators
	"contains": {Arity: "2", Description: "Check if array contains value"},
	"oneOf":    {Arity: "2", Description: "Check if value is one of array"},
}

