package format

import (
	"fmt"
	"strconv"
	"strings"
)

// Inspectable is a minimal interface for objects that can be inspected
// This matches the pattern in ast.Inspectable
type Inspectable interface {
	Inspect() string
}

// TypedObject extends Inspectable with type information
type TypedObject interface {
	Inspectable
	Type() string
}

// FormatValue formats any value for display.
// This is the main entry point - it accepts interface{} for flexibility.
func FormatValue(v interface{}) string {
	if v == nil {
		return "null"
	}

	// Check if it's a typed object we know how to format
	if obj, ok := v.(TypedObject); ok {
		return formatTypedObject(obj)
	}

	// Check if it's at least inspectable
	if obj, ok := v.(Inspectable); ok {
		return obj.Inspect()
	}

	// Fall back to fmt
	return fmt.Sprintf("%v", v)
}

// formatTypedObject dispatches based on object type
func formatTypedObject(obj TypedObject) string {
	p := NewPrinter()
	p.formatTypedObject(obj)
	return p.String()
}

// formatTypedObject formats a typed object using the printer
func (p *Printer) formatTypedObject(obj TypedObject) {
	if obj == nil {
		p.write("null")
		return
	}

	switch obj.Type() {
	case "INTEGER", "FLOAT", "BOOLEAN", "MONEY", "NULL":
		// Simple types - use Inspect directly
		p.write(obj.Inspect())
	case "STRING":
		p.formatStringObject(obj)
	case "ARRAY":
		p.formatArrayObject(obj)
	case "DICTIONARY":
		p.formatDictionaryObject(obj)
	case "FUNCTION":
		p.formatFunctionObject(obj)
	case "TABLE":
		p.write(obj.Inspect()) // Tables have a good Inspect already
	case "RECORD":
		p.formatRecordObject(obj)
	default:
		// Fall back to Inspect for unknown types
		p.write(obj.Inspect())
	}
}

// formatStringObject formats a string with proper quoting for code output
func (p *Printer) formatStringObject(obj TypedObject) {
	// For code output, strings should be quoted
	s := obj.Inspect()
	p.write(strconv.Quote(s))
}

// ArrayAccessor provides access to array elements
type ArrayAccessor interface {
	GetElements() []TypedObject
}

// formatArrayObject formats an array
func (p *Printer) formatArrayObject(obj TypedObject) {
	// Try to get array accessor
	arr, ok := obj.(ArrayAccessor)
	if !ok {
		p.write(obj.Inspect())
		return
	}

	elements := arr.GetElements()
	if len(elements) == 0 {
		p.write("[]")
		return
	}

	// Try inline format
	inline := formatArrayInline(elements)
	if fitsInThreshold(inline, ArrayThreshold) {
		p.write(inline)
		return
	}

	// Multiline format
	p.write("[")
	p.newline()
	p.indentInc()
	for i, elem := range elements {
		p.writeIndent()
		p.formatTypedObject(elem)
		if TrailingCommaMultiline || i < len(elements)-1 {
			p.write(",")
		}
		p.newline()
	}
	p.indentDec()
	p.writeIndent()
	p.write("]")
}

// formatArrayInline formats array elements as inline string
func formatArrayInline(elements []TypedObject) string {
	parts := make([]string, len(elements))
	for i, elem := range elements {
		parts[i] = formatTypedObject(elem)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// DictionaryAccessor provides access to dictionary data
type DictionaryAccessor interface {
	GetKeys() []string
	GetValueObject(key string) TypedObject
}

// formatDictionaryObject formats a dictionary
func (p *Printer) formatDictionaryObject(obj TypedObject) {
	dict, ok := obj.(DictionaryAccessor)
	if !ok {
		p.write(obj.Inspect())
		return
	}

	keys := dict.GetKeys()
	if len(keys) == 0 {
		p.write("{}")
		return
	}

	// Try inline format
	inline := formatDictInline(dict, keys)
	if fitsInThreshold(inline, DictThreshold) {
		p.write(inline)
		return
	}

	// Multiline format
	p.write("{")
	p.newline()
	p.indentInc()
	for i, key := range keys {
		p.writeIndent()
		p.write(formatDictKey(key))
		p.write(": ")
		val := dict.GetValueObject(key)
		if val != nil {
			p.formatTypedObject(val)
		} else {
			p.write("null")
		}
		if TrailingCommaMultiline || i < len(keys)-1 {
			p.write(",")
		}
		p.newline()
	}
	p.indentDec()
	p.writeIndent()
	p.write("}")
}

// formatDictInline formats dictionary as inline string
func formatDictInline(dict DictionaryAccessor, keys []string) string {
	parts := make([]string, len(keys))
	for i, key := range keys {
		val := dict.GetValueObject(key)
		valStr := "null"
		if val != nil {
			valStr = formatTypedObject(val)
		}
		parts[i] = formatDictKey(key) + ": " + valStr
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// formatDictKey formats a dictionary key, quoting if necessary
func formatDictKey(key string) string {
	// If key is a valid identifier, no quotes needed
	if isValidIdentifier(key) {
		return key
	}
	return strconv.Quote(key)
}

// isValidIdentifier checks if a string is a valid Parsley identifier
func isValidIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !isLetter(r) {
				return false
			}
		} else {
			if !isLetter(r) && !isDigit(r) {
				return false
			}
		}
	}
	return true
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_'
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// FunctionAccessor provides access to function data
type FunctionAccessor interface {
	GetParamStrings() []string
	GetBodyString() string
}

// formatFunctionObject formats a function
func (p *Printer) formatFunctionObject(obj TypedObject) {
	fn, ok := obj.(FunctionAccessor)
	if !ok {
		p.write(obj.Inspect())
		return
	}

	params := fn.GetParamStrings()
	body := fn.GetBodyString()

	paramStr := strings.Join(params, ", ")

	// Check if we can format inline
	// Strip braces if they're there
	bodyTrimmed := strings.TrimSpace(body)
	if strings.HasPrefix(bodyTrimmed, "{") && strings.HasSuffix(bodyTrimmed, "}") {
		bodyTrimmed = strings.TrimSpace(bodyTrimmed[1 : len(bodyTrimmed)-1])
	}

	inline := fmt.Sprintf("fn(%s) { %s }", paramStr, bodyTrimmed)
	if !strings.Contains(bodyTrimmed, "\n") && fitsInThreshold(inline, FuncArgsThreshold) {
		p.write(inline)
		return
	}

	// Multiline format
	p.write("fn(")

	// Check if params need multiline
	paramsLine := strings.Join(params, ", ")
	if len(params) > 3 || !fitsInThreshold("fn("+paramsLine+")", FuncArgsThreshold) {
		p.newline()
		p.indentInc()
		for i, param := range params {
			p.writeIndent()
			p.write(param)
			if TrailingCommaMultiline || i < len(params)-1 {
				p.write(",")
			}
			p.newline()
		}
		p.indentDec()
		p.writeIndent()
	} else {
		p.write(paramsLine)
	}

	p.write(") {")
	p.newline()
	p.indentInc()

	// Write body lines
	lines := strings.Split(bodyTrimmed, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			p.writeIndent()
			p.write(trimmed)
			p.newline()
		}
	}

	p.indentDec()
	p.writeIndent()
	p.write("}")
}

// RecordAccessor provides access to record data
type RecordAccessor interface {
	GetSchemaName() string
	GetFieldKeys() []string
	GetFieldObject(key string) TypedObject
}

// formatRecordObject formats a schema-bound record
func (p *Printer) formatRecordObject(obj TypedObject) {
	rec, ok := obj.(RecordAccessor)
	if !ok {
		p.write(obj.Inspect())
		return
	}

	schemaName := rec.GetSchemaName()
	keys := rec.GetFieldKeys()

	// Sort keys if needed
	if len(keys) == 0 {
		p.write(schemaName + "{}")
		return
	}

	// Try inline format first
	inline := formatRecordInline(rec, schemaName, keys)
	if fitsInThreshold(inline, DictThreshold) {
		p.write(inline)
		return
	}

	// Multiline format
	p.write(schemaName)
	p.write("{")
	p.newline()
	p.indentInc()
	for i, key := range keys {
		p.writeIndent()
		p.write(formatDictKey(key))
		p.write(": ")
		val := rec.GetFieldObject(key)
		if val != nil {
			p.formatTypedObject(val)
		} else {
			p.write("null")
		}
		if TrailingCommaMultiline || i < len(keys)-1 {
			p.write(",")
		}
		p.newline()
	}
	p.indentDec()
	p.writeIndent()
	p.write("}")
}

// formatRecordInline formats record as inline string
func formatRecordInline(rec RecordAccessor, schemaName string, keys []string) string {
	parts := make([]string, len(keys))
	for i, key := range keys {
		val := rec.GetFieldObject(key)
		valStr := "null"
		if val != nil {
			valStr = formatTypedObject(val)
		}
		parts[i] = formatDictKey(key) + ": " + valStr
	}
	return schemaName + "{" + strings.Join(parts, ", ") + "}"
}

// FormatInspectable formats any Inspectable object
// This is useful for simple cases where we just want formatted output
func FormatInspectable(obj Inspectable) string {
	if obj == nil {
		return "null"
	}
	if typed, ok := obj.(TypedObject); ok {
		return formatTypedObject(typed)
	}
	return obj.Inspect()
}

// FormatObject formats any object that implements Type() and Inspect().
// This is the primary entry point for formatting evaluator objects.
// The object should be wrapped using the evaluator's wrapper functions.
func FormatObject(obj TypedObject) string {
	return formatTypedObject(obj)
}
