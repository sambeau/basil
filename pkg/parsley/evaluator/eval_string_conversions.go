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
