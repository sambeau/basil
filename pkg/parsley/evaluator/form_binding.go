package evaluator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// FormContext tracks the current @record binding during form evaluation.
// This enables @field attributes to access the bound record's data and schema.
type FormContext struct {
	Record *Record // The record bound via @record attribute
}

// getFormContext returns the current form context from the environment chain.
// Returns nil if not inside a <form @record={...}> context.
func getFormContext(env *Environment) *FormContext {
	// Walk up the environment chain looking for FormContext
	current := env
	for current != nil {
		if current.FormContext != nil {
			return current.FormContext
		}
		current = current.outer
	}
	return nil
}

// setFormContext sets a form context in the environment.
// The context is stored on this environment level only.
func setFormContext(env *Environment, ctx *FormContext) {
	env.FormContext = ctx
}

// parseFormAtRecord parses the @record attribute from props string.
// Returns the expression if found, nil if not present.
// baseLine and baseCol are the position of the props string start for error reporting.
func parseFormAtRecord(propsStr string, env *Environment, baseLine, baseCol int) (ast.Expression, *Error) {
	// Look for @record= in props
	idx := strings.Index(propsStr, "@record=")
	if idx == -1 {
		idx = strings.Index(propsStr, "@record =")
	}
	if idx == -1 {
		return nil, nil // No @record attribute
	}

	// Find the start of the value
	valueStart := idx + len("@record=")
	for valueStart < len(propsStr) && propsStr[valueStart] == ' ' {
		valueStart++
	}
	if valueStart >= len(propsStr) {
		return nil, &Error{
			Class:   ClassParse,
			Code:    "FORM-0005",
			Message: "@record attribute has no value",
			Line:    baseLine,
			Column:  baseCol + idx,
			File:    env.Filename,
		}
	}

	// Parse the value - must be a brace expression {expr}
	if propsStr[valueStart] != '{' {
		return nil, &Error{
			Class:   ClassParse,
			Code:    "FORM-0005",
			Message: "@record attribute must use brace syntax: @record={expression}",
			Hints:   []string{"Use @record={myRecord} instead of @record=myRecord"},
			Line:    baseLine,
			Column:  baseCol + valueStart,
			File:    env.Filename,
		}
	}

	// Find matching closing brace
	braceDepth := 1
	valueEnd := valueStart + 1
	for valueEnd < len(propsStr) && braceDepth > 0 {
		switch propsStr[valueEnd] {
		case '{':
			braceDepth++
		case '}':
			braceDepth--
		}
		valueEnd++
	}

	if braceDepth != 0 {
		return nil, &Error{
			Class:   ClassParse,
			Code:    "PARSE-0009",
			Message: "@record attribute has unmatched braces",
			Line:    baseLine,
			Column:  baseCol + valueStart,
			File:    env.Filename,
		}
	}

	// Extract expression (without braces)
	exprStr := propsStr[valueStart+1 : valueEnd-1]
	exprOffset := valueStart + 1 // offset of expression content within propsStr

	// Parse the expression
	l := lexer.NewWithFilename(exprStr, env.Filename)
	p := parser.New(l)
	program := p.ParseProgram()

	if errs := p.StructuredErrors(); len(errs) > 0 {
		perr := errs[0]
		return nil, &Error{
			Class:   ClassParse,
			Code:    perr.Code,
			Message: "invalid @record expression: " + perr.Message,
			Hints:   perr.Hints,
			Line:    baseLine,
			Column:  baseCol + exprOffset + (perr.Column - 1),
			File:    env.Filename,
			Data:    perr.Data,
		}
	}

	if len(program.Statements) == 0 {
		return nil, &Error{
			Class:   ClassParse,
			Code:    "FORM-0005",
			Message: "@record expression is empty",
			Line:    baseLine,
			Column:  baseCol + exprOffset,
			File:    env.Filename,
		}
	}

	exprStmt, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		return nil, &Error{
			Class:   ClassParse,
			Code:    "FORM-0005",
			Message: "@record must be an expression",
			Line:    baseLine,
			Column:  baseCol + exprOffset,
			File:    env.Filename,
		}
	}

	return exprStmt.Expression, nil
}

// removeAtRecord removes the @record={...} attribute from props string.
// Returns the cleaned props string.
func removeAtRecord(propsStr string) string {
	// Pattern: @record={...} or @record = {...}
	re := regexp.MustCompile(`\s*@record\s*=\s*\{[^}]*\}`)
	result := re.ReplaceAllString(propsStr, "")
	// Clean up any double spaces left behind
	result = strings.TrimSpace(result)
	// Replace multiple spaces with single space
	for strings.Contains(result, "  ") {
		result = strings.ReplaceAll(result, "  ", " ")
	}
	return result
}

// parseFieldAttribute parses @field="name" from a tag's props.
// Returns the field name if found, empty string if not present.
func parseFieldAttribute(propsStr string) string {
	// Look for @field=
	idx := strings.Index(propsStr, "@field=")
	if idx == -1 {
		idx = strings.Index(propsStr, "@field =")
	}
	if idx == -1 {
		return ""
	}

	// Find start of value
	valueStart := idx + len("@field=")
	for valueStart < len(propsStr) && propsStr[valueStart] == ' ' {
		valueStart++
	}
	if valueStart >= len(propsStr) {
		return ""
	}

	// Must be quoted or braced
	if propsStr[valueStart] == '"' {
		// Find closing quote
		valueEnd := valueStart + 1
		for valueEnd < len(propsStr) && propsStr[valueEnd] != '"' {
			if propsStr[valueEnd] == '\\' {
				valueEnd++
			}
			valueEnd++
		}
		return propsStr[valueStart+1 : valueEnd]
	} else if propsStr[valueStart] == '{' {
		// Brace expression - find closing brace
		braceDepth := 1
		valueEnd := valueStart + 1
		for valueEnd < len(propsStr) && braceDepth > 0 {
			switch propsStr[valueEnd] {
			case '{':
				braceDepth++
			case '}':
				braceDepth--
			}
			valueEnd++
		}
		// Return expression without braces - caller will evaluate
		return propsStr[valueStart+1 : valueEnd-1]
	}

	return ""
}

// parseTagAttribute parses @tag="element" from props.
// Returns the override tag name if found, empty string otherwise.
func parseTagAttribute(propsStr string) string {
	idx := strings.Index(propsStr, "@tag=")
	if idx == -1 {
		idx = strings.Index(propsStr, "@tag =")
	}
	if idx == -1 {
		return ""
	}

	valueStart := idx + len("@tag=")
	for valueStart < len(propsStr) && propsStr[valueStart] == ' ' {
		valueStart++
	}
	if valueStart >= len(propsStr) {
		return ""
	}

	if propsStr[valueStart] == '"' {
		valueEnd := valueStart + 1
		for valueEnd < len(propsStr) && propsStr[valueEnd] != '"' {
			valueEnd++
		}
		return propsStr[valueStart+1 : valueEnd]
	}

	return ""
}

// parseKeyAttribute parses @key="metadata-key" from props.
// Returns the key name if found, empty string otherwise.
func parseKeyAttribute(propsStr string) string {
	idx := strings.Index(propsStr, "@key=")
	if idx == -1 {
		idx = strings.Index(propsStr, "@key =")
	}
	if idx == -1 {
		return ""
	}

	valueStart := idx + len("@key=")
	for valueStart < len(propsStr) && propsStr[valueStart] == ' ' {
		valueStart++
	}
	if valueStart >= len(propsStr) {
		return ""
	}

	if propsStr[valueStart] == '"' {
		valueEnd := valueStart + 1
		for valueEnd < len(propsStr) && propsStr[valueEnd] != '"' {
			valueEnd++
		}
		return propsStr[valueStart+1 : valueEnd]
	}

	return ""
}

// removeFieldAttribute removes @field="..." from props string.
func removeFieldAttribute(propsStr string) string {
	// Pattern: @field="..." or @field = "..."
	re := regexp.MustCompile(`\s*@field\s*=\s*("[^"]*"|\{[^}]*\})`)
	return re.ReplaceAllString(propsStr, "")
}

// removeComponentAttributes removes @field, @tag, @key from props.
func removeComponentAttributes(propsStr string) string {
	re := regexp.MustCompile(`\s*@(field|tag|key)\s*=\s*("[^"]*"|\{[^}]*\})`)
	return re.ReplaceAllString(propsStr, "")
}

// buildInputAttributes creates attributes for an <input> based on schema field.
// Returns the attribute string to append to the input tag.
func buildInputAttributes(record *Record, fieldName string, inputType string) string {
	var attrs strings.Builder

	// Add name attribute
	attrs.WriteString(fmt.Sprintf(` name="%s"`, fieldName))

	// Get field definition from schema
	field := record.Schema.Fields[fieldName]

	// SPEC-ID-007: Auto fields render as hidden or readonly inputs
	if field != nil && field.Auto {
		// Auto fields should be hidden - they're generated by DB/server
		if inputType == "" {
			attrs.WriteString(` type="hidden"`)
		}
		// Add value if present (e.g., for existing records being edited)
		value := record.Get(fieldName, record.Env)
		if value != nil && value != NULL {
			valueStr := objectToTemplateString(value)
			attrs.WriteString(fmt.Sprintf(` value="%s"`, escapeAttrValue(valueStr)))
		}
		attrs.WriteString(` readonly`)
		return attrs.String()
	}

	// Add value attribute (unless checkbox/radio - handled differently)
	if inputType != "checkbox" && inputType != "radio" {
		value := record.Get(fieldName, record.Env)
		if value != nil && value != NULL {
			valueStr := objectToTemplateString(value)
			// Escape HTML attribute value
			valueStr = escapeAttrValue(valueStr)
			attrs.WriteString(fmt.Sprintf(` value="%s"`, valueStr))
		}
	}

	if field == nil {
		return attrs.String()
	}

	// Add type attribute if not already specified and can be derived
	if inputType == "" {
		derivedType := deriveHTMLInputType(field.Type)
		if derivedType != "" {
			attrs.WriteString(fmt.Sprintf(` type="%s"`, derivedType))
		}
	}

	// Add required attribute
	if field.Required {
		attrs.WriteString(` required`)
		attrs.WriteString(` aria-required="true"`)
	}

	// Add length constraints (for text-like inputs)
	if field.MinLength != nil && inputType != "number" {
		attrs.WriteString(fmt.Sprintf(` minlength="%d"`, *field.MinLength))
	}
	if field.MaxLength != nil && inputType != "number" {
		attrs.WriteString(fmt.Sprintf(` maxlength="%d"`, *field.MaxLength))
	}

	// Add value range constraints (for number inputs)
	if field.MinValue != nil && (inputType == "number" || inputType == "") {
		attrs.WriteString(fmt.Sprintf(` min="%d"`, *field.MinValue))
	}
	if field.MaxValue != nil && (inputType == "number" || inputType == "") {
		attrs.WriteString(fmt.Sprintf(` max="%d"`, *field.MaxValue))
	}

	// Add pattern attribute for regex validation (SPEC-PAT-008, SPEC-PAT-009)
	// HTML5 pattern attribute uses JavaScript regex syntax
	if field.PatternSource != "" && inputType != "number" {
		// Convert Go regex to JS-compatible (best effort)
		jsPattern := convertGoRegexToJS(field.PatternSource)
		if jsPattern != "" {
			attrs.WriteString(fmt.Sprintf(` pattern="%s"`, escapeAttrValue(jsPattern)))
		}
	}

	// Add placeholder from metadata
	if field.Metadata != nil {
		if placeholder, ok := field.Metadata["placeholder"]; ok {
			if strVal, ok := placeholder.(*String); ok {
				attrs.WriteString(fmt.Sprintf(` placeholder="%s"`, escapeAttrValue(strVal.Value)))
			}
		}
	}

	// Add autocomplete attribute (FEAT-097)
	if autocomplete := getAutocomplete(fieldName, field.Type, field.Metadata); autocomplete != "" {
		attrs.WriteString(fmt.Sprintf(` autocomplete="%s"`, escapeAttrValue(autocomplete)))
	}

	// Add ARIA attributes for validation state
	hasError := record.Errors != nil && record.Errors[fieldName] != nil
	if hasError {
		attrs.WriteString(` aria-invalid="true"`)
		attrs.WriteString(fmt.Sprintf(` aria-describedby="%s-error"`, fieldName))
	} else if record.Validated {
		attrs.WriteString(` aria-invalid="false"`)
	}

	return attrs.String()
}

// convertGoRegexToJS converts a Go regex pattern to JavaScript-compatible syntax.
// Returns empty string if the pattern cannot be converted (Go-specific features).
// SPEC-PAT-009, SPEC-PAT-010
func convertGoRegexToJS(goPattern string) string {
	// Most basic Go regex patterns are JS-compatible.
	// Known incompatibilities:
	// - Go: (?P<name>...) named groups - JS uses (?<name>...)
	// - Go: (?:...) non-capturing groups - same in JS
	// - Go: doesn't support lookbehind in older versions

	// For now, return patterns as-is for simple cases
	// More complex conversion could be added later

	// Check for Go-specific syntax that doesn't translate
	if strings.Contains(goPattern, "(?P<") {
		// Named capture groups differ between Go and JS
		// Could convert, but skip for now
		return ""
	}

	return goPattern
}

// buildCheckboxAttributes creates attributes for checkbox inputs.
func buildCheckboxAttributes(record *Record, fieldName string) string {
	var attrs strings.Builder

	attrs.WriteString(fmt.Sprintf(` name="%s"`, fieldName))
	attrs.WriteString(` type="checkbox"`)

	// Check if field value is truthy
	value := record.Get(fieldName, record.Env)
	if value != nil && value != NULL {
		if isTruthy(value) {
			attrs.WriteString(` checked`)
		}
	}

	// ARIA attributes
	field := record.Schema.Fields[fieldName]
	if field != nil {
		if field.Required {
			attrs.WriteString(` aria-required="true"`)
		}
	}

	hasError := record.Errors != nil && record.Errors[fieldName] != nil
	if hasError {
		attrs.WriteString(` aria-invalid="true"`)
		attrs.WriteString(fmt.Sprintf(` aria-describedby="%s-error"`, fieldName))
	}

	return attrs.String()
}

// buildRadioAttributes creates attributes for radio inputs.
func buildRadioAttributes(record *Record, fieldName string, radioValue string) string {
	var attrs strings.Builder

	attrs.WriteString(fmt.Sprintf(` name="%s"`, fieldName))
	attrs.WriteString(` type="radio"`)
	attrs.WriteString(fmt.Sprintf(` value="%s"`, escapeAttrValue(radioValue)))

	// Check if current value matches radio value
	currentValue := record.Get(fieldName, record.Env)
	if currentValue != nil && currentValue != NULL {
		currentStr := objectToTemplateString(currentValue)
		if currentStr == radioValue {
			attrs.WriteString(` checked`)
		}
	}

	// ARIA attributes
	field := record.Schema.Fields[fieldName]
	if field != nil {
		if field.Required {
			attrs.WriteString(` aria-required="true"`)
		}
	}

	hasError := record.Errors != nil && record.Errors[fieldName] != nil
	if hasError {
		attrs.WriteString(` aria-invalid="true"`)
	}

	return attrs.String()
}

// deriveHTMLInputType derives the HTML input type from schema field type.
func deriveHTMLInputType(schemaType string) string {
	// Normalize type (strip nullable marker)
	baseType := strings.TrimSuffix(strings.ToLower(schemaType), "?")

	switch baseType {
	case "email":
		return "email"
	case "url":
		return "url"
	case "phone", "tel":
		return "tel"
	case "int", "integer", "number", "bigint":
		return "number"
	case "date":
		return "date"
	case "datetime":
		return "datetime-local"
	case "time":
		return "time"
	case "password":
		return "password"
	case "hidden":
		return "hidden"
	default:
		return "" // Default to text (browser default)
	}
}

// escapeAttrValue escapes a string for use in an HTML attribute value.
func escapeAttrValue(s string) string {
	var result strings.Builder
	for _, c := range s {
		switch c {
		case '"':
			result.WriteString("&quot;")
		case '&':
			result.WriteString("&amp;")
		case '<':
			result.WriteString("&lt;")
		case '>':
			result.WriteString("&gt;")
		default:
			result.WriteRune(c)
		}
	}
	return result.String()
}

// parseExistingType extracts type="..." from props if present.
func parseExistingType(propsStr string) string {
	// Look for type= in props
	idx := strings.Index(propsStr, "type=")
	if idx == -1 {
		return ""
	}

	valueStart := idx + len("type=")
	for valueStart < len(propsStr) && propsStr[valueStart] == ' ' {
		valueStart++
	}
	if valueStart >= len(propsStr) {
		return ""
	}

	if propsStr[valueStart] == '"' {
		valueEnd := valueStart + 1
		for valueEnd < len(propsStr) && propsStr[valueEnd] != '"' {
			valueEnd++
		}
		return propsStr[valueStart+1 : valueEnd]
	}

	return ""
}

// parseExistingValue extracts value="..." from props if present.
func parseExistingValue(propsStr string) string {
	// Look for value= in props
	idx := strings.Index(propsStr, "value=")
	if idx == -1 {
		return ""
	}

	valueStart := idx + len("value=")
	for valueStart < len(propsStr) && propsStr[valueStart] == ' ' {
		valueStart++
	}
	if valueStart >= len(propsStr) {
		return ""
	}

	if propsStr[valueStart] == '"' {
		valueEnd := valueStart + 1
		for valueEnd < len(propsStr) && propsStr[valueEnd] != '"' {
			if propsStr[valueEnd] == '\\' {
				valueEnd++
			}
			valueEnd++
		}
		return propsStr[valueStart+1 : valueEnd]
	}

	return ""
}
