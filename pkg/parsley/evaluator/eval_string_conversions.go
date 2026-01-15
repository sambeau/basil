package evaluator

import (
	"fmt"
	"strconv"
	"strings"
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
		// Regular dictionary
		var result strings.Builder
		result.WriteString("{")
		first := true
		for key, valExpr := range obj.Pairs {
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
			// Value: need to evaluate the expression first
			if val, ok := valExpr.(Object); ok {
				result.WriteString(objectToReprStringWithSeen(val, seen))
			} else {
				result.WriteString("<unevaluated>")
			}
		}
		result.WriteString("}")
		return result.String()
	case *Money:
		return obj.Inspect() // Money.Inspect() already returns parseable form
	case *Function:
		return "<function>"
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

// durationToReprString converts a duration dictionary to its literal form
func durationToReprString(dict *Dictionary) string {
	// Get the duration string representation
	dur := durationDictToString(dict)
	return "@" + dur
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
