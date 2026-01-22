package evaluator

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/format"
	"github.com/sambeau/basil/pkg/parsley/lexer"
)

// objectToTemplateString converts an object to its string representation for template interpolation
// Arrays are concatenated without separators. Special dictionaries (path, url, etc.) use their
// toString representations. Regular dictionaries use Inspect().
func objectToTemplateString(obj Object) string {
	switch obj := obj.(type) {
	case *Integer:
		return strconv.FormatInt(obj.Value, 10)
	case *Float:
		return fmt.Sprintf("%g", obj.Value)
	case *Boolean:
		if obj.Value {
			return "true"
		}
		return "false"
	case *String:
		return obj.Value
	case *Array:
		// Arrays are printed without commas in templates
		var result strings.Builder
		for _, elem := range obj.Elements {
			result.WriteString(objectToTemplateString(elem))
		}
		return result.String()
	case *Dictionary:
		// Check for special dictionary types
		if isPathDict(obj) {
			return pathDictToString(obj)
		}
		if isUrlDict(obj) {
			return urlDictToString(obj)
		}
		if isTagDict(obj) {
			return tagDictToString(obj)
		}
		if isDatetimeDict(obj) {
			return datetimeDictToString(obj)
		}
		if isDurationDict(obj) {
			return durationDictToString(obj)
		}
		if isRegexDict(obj) {
			return regexDictToString(obj)
		}
		if isFileDict(obj) {
			return fileDictToString(obj)
		}
		if isDirDict(obj) {
			return dirDictToString(obj)
		}
		if isRequestDict(obj) {
			return requestDictToString(obj)
		}
		return obj.Inspect()
	case *Record:
		// Records are serialized as JSON objects for JavaScript compatibility.
		// If the record has a PLN secret, use PLN encoding for round-trip support.
		if obj.Env != nil && obj.Env.PLNSecret != "" && serializeToPLNForProps != nil {
			plnStr, err := serializeToPLNForProps(obj, obj.Env)
			if err == nil && signPLNProp != nil {
				signed := signPLNProp(plnStr, obj.Env.PLNSecret)
				// Return as JSON object with __pln marker
				jsonBytes, _ := json.Marshal(map[string]string{"__pln": signed})
				return string(jsonBytes)
			}
		}
		// Fallback: convert to JSON object (data only)
		dict := obj.ToDictionary()
		goVal := objectToGoValue(dict)
		jsonBytes, err := json.Marshal(goVal)
		if err != nil {
			return obj.Inspect() // Last resort fallback
		}
		return string(jsonBytes)
	case *Null:
		return ""
	default:
		return obj.Inspect()
	}
}

// evalDictionarySpread evaluates a dictionary and writes its key-value pairs
// as HTML attributes to the builder. It handles null/false omission and boolean attributes.
func evalDictionarySpread(dict *Dictionary, builder *strings.Builder, env *Environment) error {
	if dict == nil {
		return nil
	}

	// Collect and sort keys for deterministic output
	keys := make([]string, 0, len(dict.Pairs))
	for key := range dict.Pairs {
		keys = append(keys, key)
	}

	// Sort keys alphabetically
	sortKeys := func(keys []string) {
		for i := 0; i < len(keys); i++ {
			for j := i + 1; j < len(keys); j++ {
				if keys[i] > keys[j] {
					keys[i], keys[j] = keys[j], keys[i]
				}
			}
		}
	}
	sortKeys(keys)

	for _, key := range keys {
		expr := dict.Pairs[key]

		// Evaluate the expression
		value := Eval(expr, env)
		if isError(value) {
			return fmt.Errorf("error evaluating attribute %s: %s", key, value.Inspect())
		}

		// Skip null and false values
		switch v := value.(type) {
		case *Null:
			continue
		case *Boolean:
			if !v.Value {
				continue
			}
			// Boolean true: render as boolean attribute
			builder.WriteByte(' ')
			builder.WriteString(key)
			continue
		}

		// Render as regular attribute with value
		builder.WriteByte(' ')
		builder.WriteString(key)
		builder.WriteString("=\"")

		// Get string value and escape quotes
		strVal := objectToTemplateString(value)
		for _, c := range strVal {
			if c == '"' {
				builder.WriteString("&quot;")
			} else if c == '&' {
				builder.WriteString("&amp;")
			} else if c == '<' {
				builder.WriteString("&lt;")
			} else if c == '>' {
				builder.WriteString("&gt;")
			} else {
				builder.WriteRune(c)
			}
		}

		builder.WriteByte('"')
	}

	return nil
}

// objectToUserString converts an object to its user-facing string representation
// for print() and {interpolation}. Differs from objectToPrintString in that:
// - Arrays are rendered as concatenated elements (not JSON-style)
// - Dictionaries are rendered as Parsley-style {a: 1, b: 2}
// - Null is empty string (silent)
func objectToUserString(obj Object) string {
	if obj == nil {
		return ""
	}

	switch o := obj.(type) {
	case *Integer:
		return strconv.FormatInt(o.Value, 10)
	case *Float:
		return fmt.Sprintf("%g", o.Value)
	case *Boolean:
		if o.Value {
			return "true"
		}
		return "false"
	case *String:
		return o.Value // No quotes
	case *Null:
		return "" // Silent in output
	case *Array:
		// Concatenate elements (same as objectToPrintString)
		var result strings.Builder
		for _, elem := range o.Elements {
			result.WriteString(objectToUserString(elem))
		}
		return result.String()
	case *Dictionary:
		// Check for special dictionary types first
		if isPathDict(o) {
			return pathDictToString(o)
		}
		if isUrlDict(o) {
			return urlDictToString(o)
		}
		if isTagDict(o) {
			return tagDictToString(o)
		}
		if isDatetimeDict(o) {
			return datetimeDictToString(o)
		}
		if isDurationDict(o) {
			return durationDictToString(o)
		}
		if isRegexDict(o) {
			return regexDictToString(o)
		}
		if isFileDict(o) {
			return fileDictToString(o)
		}
		if isDirDict(o) {
			return dirDictToString(o)
		}
		if isRequestDict(o) {
			return requestDictToString(o)
		}
		// Regular dictionary: Parsley-style {a: 1, b: 2}
		var result strings.Builder
		result.WriteString("{")
		first := true
		for key, val := range o.Pairs {
			if !first {
				result.WriteString(", ")
			}
			first = false
			result.WriteString(key)
			result.WriteString(": ")
			// Evaluate the expression to get the value
			if o.Env != nil {
				evaluated := Eval(val, o.Env)
				result.WriteString(objectToUserString(evaluated))
			} else {
				result.WriteString(val.String())
			}
		}
		result.WriteString("}")
		return result.String()
	case *Table:
		// Summary format
		rowCount := len(o.Rows)
		colCount := len(o.Columns)
		return fmt.Sprintf("<Table: %d rows, %d cols>", rowCount, colCount)
	case *Error:
		// [ERR-001] message format
		if o.Code != "" {
			return fmt.Sprintf("[%s] %s", o.Code, o.Message)
		}
		return o.Message
	case *Function:
		return "<function>"
	case *Builtin:
		return "<builtin>"
	case *DBConnection:
		return "<DBConnection>"
	default:
		return o.Inspect()
	}
}

// objectToPrintString converts an object to its string representation for the print function
// Arrays are concatenated without separators. Special dictionaries use their toString
// representations. Regular dictionaries use Inspect().
func objectToPrintString(obj Object) string {
	if obj == nil {
		return ""
	}

	switch obj := obj.(type) {
	case *Integer:
		return strconv.FormatInt(obj.Value, 10)
	case *Float:
		return fmt.Sprintf("%g", obj.Value)
	case *Boolean:
		if obj.Value {
			return "true"
		}
		return "false"
	case *String:
		return obj.Value
	case *Array:
		// Arrays: recursively print each element without any separators
		var result strings.Builder
		for _, elem := range obj.Elements {
			result.WriteString(objectToPrintString(elem))
		}
		return result.String()
	case *Table:
		// Tables: recursively print each row without any separators
		var result strings.Builder
		for _, row := range obj.Rows {
			result.WriteString(objectToPrintString(row))
		}
		return result.String()
	case *Dictionary:
		// Check for special dictionary types
		if isPathDict(obj) {
			// Convert path dictionary back to string
			return pathDictToString(obj)
		}
		if isUrlDict(obj) {
			// Convert URL dictionary back to string
			return urlDictToString(obj)
		}
		if isTagDict(obj) {
			// Convert tag dictionary to HTML string
			return tagDictToString(obj)
		}
		if isDatetimeDict(obj) {
			// Convert datetime dictionary to ISO 8601 string
			return datetimeDictToString(obj)
		}
		if isDurationDict(obj) {
			// Convert duration dictionary to human-readable string
			return durationDictToString(obj)
		}
		if isRegexDict(obj) {
			// Convert regex dictionary to /pattern/flags format
			return regexDictToString(obj)
		}
		if isFileDict(obj) {
			// Convert file dictionary to path string
			return fileDictToString(obj)
		}
		if isDirDict(obj) {
			// Convert dir dictionary to path string with trailing slash
			return dirDictToString(obj)
		}
		if isRequestDict(obj) {
			// Convert request dictionary to METHOD URL format
			return requestDictToString(obj)
		}
		return obj.Inspect()
	case *Null:
		return ""
	default:
		return obj.Inspect()
	}
}

// ObjectToPrintString is the exported version for use outside the package
func ObjectToPrintString(obj Object) string {
	return objectToPrintString(obj)
}

// ObjectToReprString returns a Parsley literal representation that can be parsed back.
// This is the exported version for use by the REPL.
func ObjectToReprString(obj Object) string {
	return objectToReprString(obj)
}

// ObjectToFormattedReprString returns a pretty-printed Parsley literal representation.
// Unlike ObjectToReprString, this uses the format package to add multiline formatting
// when the output would exceed thresholds. Use this for REPL display.
func ObjectToFormattedReprString(obj Object) string {
	if obj == nil {
		return "null"
	}
	return objectToFormattedReprStringWithSeen(obj, make(map[Object]bool), 0)
}

// objectToFormattedReprStringWithSeen handles pretty-printing with cycle detection
func objectToFormattedReprStringWithSeen(obj Object, seen map[Object]bool, indent int) string {
	if obj == nil {
		return "null"
	}
	// Check for cycles in compound types
	switch obj := obj.(type) {
	case *Array, *Dictionary:
		if seen[obj] {
			return "<circular>"
		}
		seen[obj] = true
		defer delete(seen, obj)
	}

	switch obj := obj.(type) {
	case *Null:
		return "null"
	case *Boolean:
		if obj.Value {
			return "true"
		}
		return "false"
	case *Integer:
		return strconv.FormatInt(obj.Value, 10)
	case *Float:
		return fmt.Sprintf("%g", obj.Value)
	case *String:
		escaped := escapeStringForRepr(obj.Value)
		return fmt.Sprintf("\"%s\"", escaped)
	case *Array:
		return arrayToFormattedReprString(obj, seen, indent)
	case *Dictionary:
		return dictToFormattedReprString(obj, seen, indent)
	case *Money:
		return obj.Inspect()
	case *Function:
		return functionToFormattedReprString(obj, indent)
	case *Record:
		return recordToFormattedReprString(obj, seen, indent)
	case *Table:
		return tableToFormattedReprString(obj, seen, indent)
	case *Builtin:
		return "<builtin>"
	default:
		return fmt.Sprintf("<%s>", obj.Type())
	}
}

// arrayToFormattedReprString formats an array with multiline support
func arrayToFormattedReprString(arr *Array, seen map[Object]bool, indent int) string {
	if len(arr.Elements) == 0 {
		return "[]"
	}

	// Try inline first
	inline := arrayToInlineReprString(arr, seen)
	if len(inline) <= format.ArrayThreshold && !strings.Contains(inline, "\n") {
		return inline
	}

	// Multiline format
	var result strings.Builder
	result.WriteString("[\n")
	indentStr := strings.Repeat(format.IndentString, indent+1)
	for i, elem := range arr.Elements {
		result.WriteString(indentStr)
		result.WriteString(objectToFormattedReprStringWithSeen(elem, seen, indent+1))
		if format.TrailingCommaMultiline || i < len(arr.Elements)-1 {
			result.WriteString(",")
		}
		result.WriteString("\n")
	}
	result.WriteString(strings.Repeat(format.IndentString, indent))
	result.WriteString("]")
	return result.String()
}

// arrayToInlineReprString formats array inline (no newlines)
func arrayToInlineReprString(arr *Array, seen map[Object]bool) string {
	var result strings.Builder
	result.WriteString("[")
	for i, elem := range arr.Elements {
		if i > 0 {
			result.WriteString(", ")
		}
		result.WriteString(objectToReprStringWithSeen(elem, seen))
	}
	result.WriteString("]")
	return result.String()
}

// dictToFormattedReprString formats a dictionary with multiline support
func dictToFormattedReprString(dict *Dictionary, seen map[Object]bool, indent int) string {
	// Check for special pseudo-types first
	if isDatetimeDict(dict) {
		return datetimeToReprString(dict)
	}
	if isDurationDict(dict) {
		return durationToReprString(dict)
	}
	if isMoneyDict(dict) {
		return moneyDictToReprString(dict)
	}
	if isPathDict(dict) {
		return pathToReprString(dict)
	}
	if isUrlDict(dict) {
		return urlToReprString(dict)
	}
	if isRegexDict(dict) {
		return regexToReprString(dict)
	}
	if isFileDict(dict) {
		return fileDictToLiteral(dict)
	}
	if isDirDict(dict) {
		return dirDictToLiteral(dict)
	}

	keys := dict.Keys()
	if len(keys) == 0 {
		return "{}"
	}

	// Try inline first
	inline := dictToInlineReprString(dict, seen)
	if len(inline) <= format.DictThreshold && !strings.Contains(inline, "\n") {
		return inline
	}

	// Multiline format
	var result strings.Builder
	result.WriteString("{\n")
	indentStr := strings.Repeat(format.IndentString, indent+1)
	for i, key := range keys {
		valExpr, ok := dict.Pairs[key]
		if !ok {
			continue
		}
		result.WriteString(indentStr)
		// Key
		if needsQuotes(key) {
			result.WriteString(fmt.Sprintf("\"%s\"", escapeStringForRepr(key)))
		} else {
			result.WriteString(key)
		}
		result.WriteString(": ")
		// Value
		var val Object
		if v, ok := valExpr.(Object); ok {
			val = v
		} else {
			val = Eval(valExpr, dict.Env)
		}
		if val != nil {
			result.WriteString(objectToFormattedReprStringWithSeen(val, seen, indent+1))
		} else {
			result.WriteString("null")
		}
		if format.TrailingCommaMultiline || i < len(keys)-1 {
			result.WriteString(",")
		}
		result.WriteString("\n")
	}
	result.WriteString(strings.Repeat(format.IndentString, indent))
	result.WriteString("}")
	return result.String()
}

// dictToInlineReprString formats dictionary inline (no newlines)
func dictToInlineReprString(dict *Dictionary, seen map[Object]bool) string {
	var result strings.Builder
	result.WriteString("{")
	first := true
	for _, key := range dict.Keys() {
		valExpr, ok := dict.Pairs[key]
		if !ok {
			continue
		}
		if !first {
			result.WriteString(", ")
		}
		first = false
		if needsQuotes(key) {
			result.WriteString(fmt.Sprintf("\"%s\"", escapeStringForRepr(key)))
		} else {
			result.WriteString(key)
		}
		result.WriteString(": ")
		var val Object
		if v, ok := valExpr.(Object); ok {
			val = v
		} else {
			val = Eval(valExpr, dict.Env)
		}
		if val != nil {
			result.WriteString(objectToReprStringWithSeen(val, seen))
		} else {
			result.WriteString("null")
		}
	}
	result.WriteString("}")
	return result.String()
}

// functionToFormattedReprString formats a function using the AST formatter
func functionToFormattedReprString(fn *Function, indent int) string {
	// Construct a FunctionLiteral AST node from the Function object
	fnLit := &ast.FunctionLiteral{
		Token:  lexer.Token{Type: lexer.FUNCTION, Literal: "fn"},
		Params: fn.Params,
		Body:   fn.Body,
	}
	// Use the formatter to produce well-formatted output
	formatted := format.FormatNode(fnLit)

	// If we're nested (indent > 0) and the output is multiline,
	// we need to indent all lines after the first
	if indent > 0 && strings.Contains(formatted, "\n") {
		lines := strings.Split(formatted, "\n")
		indentStr := strings.Repeat(format.IndentString, indent)
		for i := 1; i < len(lines); i++ {
			lines[i] = indentStr + lines[i]
		}
		return strings.Join(lines, "\n")
	}
	return formatted
}

// recordToFormattedReprString formats a record with multiline support
func recordToFormattedReprString(rec *Record, seen map[Object]bool, indent int) string {
	schemaName := "?"
	if rec.Schema != nil {
		schemaName = rec.Schema.Name
	}

	keys := rec.KeyOrder
	if len(keys) == 0 && len(rec.Data) > 0 {
		keys = make([]string, 0, len(rec.Data))
		for key := range rec.Data {
			keys = append(keys, key)
		}
		sort.Strings(keys)
	}

	if len(keys) == 0 {
		return schemaName + "{}"
	}

	// Try inline first
	inline := recordToInlineReprString(rec, schemaName, keys, seen)
	if len(inline) <= format.DictThreshold && !strings.Contains(inline, "\n") {
		return inline
	}

	// Multiline format
	var result strings.Builder
	result.WriteString(schemaName)
	result.WriteString("{\n")
	indentStr := strings.Repeat(format.IndentString, indent+1)
	for i, key := range keys {
		valExpr, ok := rec.Data[key]
		if !ok {
			continue
		}
		result.WriteString(indentStr)
		if needsQuotes(key) {
			result.WriteString(fmt.Sprintf("\"%s\"", escapeStringForRepr(key)))
		} else {
			result.WriteString(key)
		}
		result.WriteString(": ")
		var val Object
		if v, ok := valExpr.(Object); ok {
			val = v
		} else {
			val = Eval(valExpr, nil)
		}
		if val != nil {
			result.WriteString(objectToFormattedReprStringWithSeen(val, seen, indent+1))
		} else {
			result.WriteString("null")
		}
		if format.TrailingCommaMultiline || i < len(keys)-1 {
			result.WriteString(",")
		}
		result.WriteString("\n")
	}
	result.WriteString(strings.Repeat(format.IndentString, indent))
	result.WriteString("}")
	return result.String()
}

// recordToInlineReprString formats a record inline
func recordToInlineReprString(rec *Record, schemaName string, keys []string, seen map[Object]bool) string {
	var result strings.Builder
	result.WriteString(schemaName)
	result.WriteString("{")
	for i, key := range keys {
		valExpr, ok := rec.Data[key]
		if !ok {
			continue
		}
		if i > 0 {
			result.WriteString(", ")
		}
		if needsQuotes(key) {
			result.WriteString(fmt.Sprintf("\"%s\"", escapeStringForRepr(key)))
		} else {
			result.WriteString(key)
		}
		result.WriteString(": ")
		var val Object
		if v, ok := valExpr.(Object); ok {
			val = v
		} else {
			val = Eval(valExpr, nil)
		}
		if val != nil {
			result.WriteString(objectToReprStringWithSeen(val, seen))
		} else {
			result.WriteString("null")
		}
	}
	result.WriteString("}")
	return result.String()
}

// tableToFormattedReprString formats a table with multiline support
func tableToFormattedReprString(tbl *Table, seen map[Object]bool, indent int) string {
	if len(tbl.Rows) == 0 {
		return "table([])"
	}

	// Convert rows to an array for formatting
	elements := make([]Object, len(tbl.Rows))
	for i, row := range tbl.Rows {
		elements[i] = row
	}
	arr := &Array{Elements: elements}

	// Format the array
	arrStr := arrayToFormattedReprString(arr, seen, indent)

	// Check if inline fits
	inline := "table(" + arrStr + ")"
	if len(inline) <= format.ArrayThreshold && !strings.Contains(arrStr, "\n") {
		return inline
	}

	// Multiline: table(\n  [...]\n)
	// We need to re-indent the array if it's multiline
	if strings.Contains(arrStr, "\n") {
		lines := strings.Split(arrStr, "\n")
		indentStr := strings.Repeat(format.IndentString, indent+1)
		for i := 0; i < len(lines); i++ {
			if i == 0 {
				lines[i] = indentStr + lines[i]
			} else if lines[i] != "" {
				lines[i] = indentStr + lines[i]
			}
		}
		arrStr = strings.Join(lines, "\n")
		return "table(\n" + arrStr + "\n" + strings.Repeat(format.IndentString, indent) + ")"
	}

	return inline
}

// objectToDebugString converts an object to its debug string representation
// Strings are wrapped in quotes, arrays use JSON-style brackets with separators.
func objectToDebugString(obj Object) string {
	switch obj := obj.(type) {
	case *Integer:
		return strconv.FormatInt(obj.Value, 10)
	case *Float:
		return fmt.Sprintf("%g", obj.Value)
	case *Boolean:
		if obj.Value {
			return "true"
		}
		return "false"
	case *String:
		// Strings are wrapped in quotes for debug output
		return fmt.Sprintf("\"%s\"", obj.Value)
	case *Array:
		// Arrays: recursively debug print each element with separators, wrapped in brackets
		var result strings.Builder
		result.WriteString("[")
		for i, elem := range obj.Elements {
			if i > 0 {
				result.WriteString(", ")
			}
			result.WriteString(objectToDebugString(elem))
		}
		result.WriteString("]")
		return result.String()
	case *Null:
		return "null"
	default:
		return obj.Inspect()
	}
}

// objectToReprString converts an object to its Parsley-parseable literal representation.
// This is the reverse of parsing - the output can be parsed back to recreate the value.
// Use this for roundtripping, serialization, and debugging where you need valid Parsley syntax.
func objectToReprString(obj Object) string {
	return objectToReprStringWithSeen(obj, make(map[Object]bool))
}

// objectToReprStringWithSeen handles cycle detection for recursive structures
func objectToReprStringWithSeen(obj Object, seen map[Object]bool) string {
	// Check for cycles in compound types
	switch obj := obj.(type) {
	case *Array, *Dictionary:
		if seen[obj] {
			return "<circular>"
		}
		seen[obj] = true
		defer delete(seen, obj)
	}

	switch obj := obj.(type) {
	case *Null:
		return "null"
	case *Boolean:
		if obj.Value {
			return "true"
		}
		return "false"
	case *Integer:
		return strconv.FormatInt(obj.Value, 10)
	case *Float:
		return fmt.Sprintf("%g", obj.Value)
	case *String:
		// Escape special characters and wrap in quotes
		escaped := escapeStringForRepr(obj.Value)
		return fmt.Sprintf("\"%s\"", escaped)
	case *Array:
		var result strings.Builder
		result.WriteString("[")
		for i, elem := range obj.Elements {
			if i > 0 {
				result.WriteString(", ")
			}
			result.WriteString(objectToReprStringWithSeen(elem, seen))
		}
		result.WriteString("]")
		return result.String()
	case *Dictionary:
		// Check for special pseudo-types
		if isDatetimeDict(obj) {
			return datetimeToReprString(obj)
		}
		if isDurationDict(obj) {
			return durationToReprString(obj)
		}
		if isMoneyDict(obj) {
			return moneyDictToReprString(obj)
		}
		if isPathDict(obj) {
			return pathToReprString(obj)
		}
		if isUrlDict(obj) {
			return urlToReprString(obj)
		}
		if isRegexDict(obj) {
			return regexToReprString(obj)
		}
		if isFileDict(obj) {
			return fileDictToLiteral(obj)
		}
		if isDirDict(obj) {
			return dirDictToLiteral(obj)
		}
		// Regular dictionary
		var result strings.Builder
		result.WriteString("{")
		first := true
		keys := obj.Keys() // Use ordered keys
		for _, key := range keys {
			valExpr, ok := obj.Pairs[key]
			if !ok {
				continue
			}
			if !first {
				result.WriteString(", ")
			}
			first = false
			// Key: use quotes if needed for non-identifier keys
			if needsQuotes(key) {
				result.WriteString(fmt.Sprintf("\"%s\"", escapeStringForRepr(key)))
			} else {
				result.WriteString(key)
			}
			result.WriteString(": ")
			// Value: evaluate the expression if needed
			var val Object
			if v, ok := valExpr.(Object); ok {
				val = v
			} else {
				// It's an AST expression, evaluate it
				val = Eval(valExpr, obj.Env)
			}
			if val != nil {
				result.WriteString(objectToReprStringWithSeen(val, seen))
			} else {
				result.WriteString("null")
			}
		}
		result.WriteString("}")
		return result.String()
	case *Money:
		return obj.Inspect() // Money.Inspect() already returns parseable form
	case *Function:
		return obj.Inspect() // Function.Inspect() returns fn(...) {...} form
	case *Builtin:
		return "<builtin>"
	default:
		// For unknown types, return a non-parseable marker
		return fmt.Sprintf("<%s>", obj.Type())
	}
}

// escapeStringForRepr escapes special characters for repr output
func escapeStringForRepr(s string) string {
	var result strings.Builder
	for _, r := range s {
		switch r {
		case '\\':
			result.WriteString("\\\\")
		case '"':
			result.WriteString("\\\"")
		case '\n':
			result.WriteString("\\n")
		case '\r':
			result.WriteString("\\r")
		case '\t':
			result.WriteString("\\t")
		default:
			result.WriteRune(r)
		}
	}
	return result.String()
}

// needsQuotes returns true if a dictionary key needs to be quoted
func needsQuotes(key string) bool {
	if len(key) == 0 {
		return true
	}
	// Check if it's a valid identifier
	for i, r := range key {
		if i == 0 {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_') {
				return true
			}
		} else {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
				return true
			}
		}
	}
	return false
}

// datetimeToReprString converts a datetime dictionary to its literal form
func datetimeToReprString(dict *Dictionary) string {
	// Get the ISO format representation
	iso := datetimeDictToString(dict)
	return "@" + iso
}

// durationToReprString converts a duration dictionary to its literal form (@2w, @1y6mo, etc)
func durationToReprString(dict *Dictionary) string {
	// Use the compact literal format
	return durationDictToLiteral(dict)
}

// moneyDictToReprString converts a money dictionary to its literal form
func moneyDictToReprString(dict *Dictionary) string {
	// This shouldn't normally be called since Money is its own type
	// but handle dict representation just in case
	if currency, ok := dict.Pairs["currency"].(Object); ok {
		if currStr, ok := currency.(*String); ok {
			if amount, ok := dict.Pairs["amount"].(Object); ok {
				symbol := currencyToSymbol(currStr.Value)
				if symbol != "" {
					return symbol + objectToReprStringWithSeen(amount, make(map[Object]bool))
				}
				return currStr.Value + "#" + objectToReprStringWithSeen(amount, make(map[Object]bool))
			}
		}
	}
	return "<money>"
}

// pathToReprString converts a path dictionary to its literal form
func pathToReprString(dict *Dictionary) string {
	path := pathDictToString(dict)
	return "@" + path
}

// urlToReprString converts a URL dictionary to its literal form
func urlToReprString(dict *Dictionary) string {
	url := urlDictToString(dict)
	return "@" + url
}

// regexToReprString converts a regex dictionary to its literal form
func regexToReprString(dict *Dictionary) string {
	return regexDictToString(dict) // Already returns /pattern/flags format
}

// isMoneyDict checks if a dictionary represents a Money value
func isMoneyDict(dict *Dictionary) bool {
	_, hasCurrency := dict.Pairs["currency"]
	_, hasAmount := dict.Pairs["amount"]
	_, hasScale := dict.Pairs["scale"]
	_, hasType := dict.Pairs["__type"]
	if hasType {
		if typeVal, ok := dict.Pairs["__type"].(Object); ok {
			if typeStr, ok := typeVal.(*String); ok {
				return typeStr.Value == "money"
			}
		}
	}
	return hasCurrency && hasAmount && hasScale
}
