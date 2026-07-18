package evaluator

import (
	"fmt"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
)

// evalFieldTag handles <field name="email"/> — outputs a complete field structure.
// Requires @record form context. Outputs: div > label + input + error.
// Boolean fields use checkbox ordering: input before label.
func evalFieldTag(node *ast.TagLiteral, propsStr string, env *Environment) Object {
	// Parse name attribute (required)
	fieldName := parseAttrValue(propsStr, "name")
	if fieldName == "" {
		return &Error{
			Class:   ClassValue,
			Code:    "FORM-0010",
			Message: "<field> requires a name attribute",
			Hints:   []string{`Usage: <field name="email"/>`},
			Line:    node.Token.Line,
			Column:  node.Token.Column,
		}
	}

	// Get form context
	formCtx := getFormContext(env)
	if formCtx == nil {
		return &Error{
			Class:   ClassValue,
			Code:    "FORM-0002",
			Message: "<field> must be inside a <form @record={...}> context",
			Hints:   []string{`Wrap in a form: <form @record={myRecord}><field name="email"/></form>`},
			Line:    node.Token.Line,
			Column:  node.Token.Column,
		}
	}

	record := formCtx.Record
	if record == nil || record.Schema == nil {
		return &Error{
			Class:   ClassValue,
			Code:    "FORM-0003",
			Message: "Form context has no valid record or schema",
			Line:    node.Token.Line,
			Column:  node.Token.Column,
		}
	}

	field := record.Schema.Fields[fieldName]

	// Parse optional props
	overrideAs := parseAttrValue(propsStr, "as")
	overrideClass := parseAttrValue(propsStr, "class")
	overrideID := parseAttrValue(propsStr, "id")
	overrideLabel := parseAttrValue(propsStr, "label")
	overridePlaceholder := parseAttrValue(propsStr, "placeholder")
	helpText := parseAttrValue(propsStr, "help")

	// Determine input type
	inputType := ""
	if overrideAs != "" {
		inputType = overrideAs
	} else if field != nil {
		inputType = deriveHTMLInputType(field.Type)
	}

	// Determine if this is a checkbox/boolean
	isCheckbox := inputType == "checkbox"
	if !isCheckbox && field != nil {
		baseType := strings.TrimSuffix(strings.ToLower(field.Type), "?")
		isCheckbox = baseType == "boolean" || baseType == "bool"
		if isCheckbox && inputType == "" {
			inputType = "checkbox"
		}
	}

	// Field ID
	fieldID := fieldName
	if overrideID != "" {
		fieldID = overrideID
	}

	// Label text
	labelText := overrideLabel
	if labelText == "" {
		if field != nil {
			labelText = getFieldTitle(fieldName, field)
		} else {
			labelText = toTitleCase(fieldName)
		}
	}

	// Wrapper class
	wrapperClass := "field"
	if overrideClass != "" {
		wrapperClass = overrideClass
	}
	if isCheckbox {
		wrapperClass += " field--checkbox"
	}

	// Build the output
	var result strings.Builder

	// Wrapper div
	fmt.Fprintf(&result, `<div class="%s">`, escapeAttrValue(wrapperClass))

	// Build label HTML
	labelHTML := fmt.Sprintf(`<label for="%s">%s</label>`, escapeAttrValue(fieldID), escapeHTMLText(labelText))

	// Build input HTML
	var inputHTML strings.Builder
	if overrideAs == "textarea" {
		// Textarea
		inputHTML.WriteString("<textarea")
		fmt.Fprintf(&inputHTML, ` name="%s"`, escapeAttrValue(fieldName))
		fmt.Fprintf(&inputHTML, ` id="%s"`, escapeAttrValue(fieldID))
		if overridePlaceholder != "" {
			fmt.Fprintf(&inputHTML, ` placeholder="%s"`, escapeAttrValue(overridePlaceholder))
		} else if field != nil && field.Metadata != nil {
			if ph, ok := field.Metadata["placeholder"]; ok {
				if phStr, ok := ph.(*String); ok {
					fmt.Fprintf(&inputHTML, ` placeholder="%s"`, escapeAttrValue(phStr.Value))
				}
			}
		}
		if field != nil && field.Required {
			inputHTML.WriteString(` required aria-required="true"`)
		}
		// ARIA validation
		hasError := record.Errors != nil && record.Errors[fieldName] != nil
		if hasError {
			inputHTML.WriteString(` aria-invalid="true"`)
			fmt.Fprintf(&inputHTML, ` aria-describedby="%s-error"`, fieldName)
		} else if record.Validated {
			inputHTML.WriteString(` aria-invalid="false"`)
		}
		inputHTML.WriteString(">")
		// Textarea value goes inside
		value := record.Get(fieldName, record.Env)
		if value != nil && value != NULL {
			inputHTML.WriteString(escapeHTMLText(objectToTemplateString(value)))
		}
		inputHTML.WriteString("</textarea>")
	} else if overrideAs == "select" || (field != nil && len(field.EnumValues) > 0 && overrideAs == "") {
		// Select — delegate to existing evalSelectComponent with synthetic props
		selectProps := fmt.Sprintf(`@field="%s"`, fieldName)
		if overridePlaceholder != "" {
			selectProps += fmt.Sprintf(` placeholder="%s"`, overridePlaceholder)
		}
		selectResult := evalSelectComponent(selectProps, env)
		if selectStr, ok := selectResult.(*String); ok {
			inputHTML.WriteString(selectStr.Value)
		}
	} else {
		// Regular input
		inputHTML.WriteString("<input")
		if inputType != "" {
			fmt.Fprintf(&inputHTML, ` type="%s"`, escapeAttrValue(inputType))
		}
		fmt.Fprintf(&inputHTML, ` name="%s"`, escapeAttrValue(fieldName))
		fmt.Fprintf(&inputHTML, ` id="%s"`, escapeAttrValue(fieldID))

		// Value
		if isCheckbox {
			value := record.Get(fieldName, record.Env)
			if value != nil && value != NULL {
				if boolVal, ok := value.(*Boolean); ok && boolVal.Value {
					inputHTML.WriteString(` checked`)
				}
			}
		} else {
			value := record.Get(fieldName, record.Env)
			if value != nil && value != NULL {
				valueStr := objectToTemplateString(value)
				fmt.Fprintf(&inputHTML, ` value="%s"`, escapeAttrValue(valueStr))
			}
		}

		// Placeholder
		if overridePlaceholder != "" {
			fmt.Fprintf(&inputHTML, ` placeholder="%s"`, escapeAttrValue(overridePlaceholder))
		} else if field != nil && field.Metadata != nil {
			if ph, ok := field.Metadata["placeholder"]; ok {
				if phStr, ok := ph.(*String); ok {
					fmt.Fprintf(&inputHTML, ` placeholder="%s"`, escapeAttrValue(phStr.Value))
				}
			}
		}

		// Required
		if field != nil && field.Required {
			inputHTML.WriteString(` required aria-required="true"`)
		}

		// Constraints
		if field != nil {
			if field.MinLength != nil && inputType != "number" {
				fmt.Fprintf(&inputHTML, ` minlength="%d"`, *field.MinLength)
			}
			if field.MaxLength != nil && inputType != "number" {
				fmt.Fprintf(&inputHTML, ` maxlength="%d"`, *field.MaxLength)
			}
			if field.MinValue != nil {
				fmt.Fprintf(&inputHTML, ` min="%d"`, *field.MinValue)
			}
			if field.MaxValue != nil {
				fmt.Fprintf(&inputHTML, ` max="%d"`, *field.MaxValue)
			}
			if field.PatternSource != "" && inputType != "number" {
				jsPattern := convertGoRegexToJS(field.PatternSource)
				if jsPattern != "" {
					fmt.Fprintf(&inputHTML, ` pattern="%s"`, escapeAttrValue(jsPattern))
				}
			}
		}

		// Autocomplete
		if field != nil {
			if ac := getAutocomplete(fieldName, field.Type, field.Metadata); ac != "" {
				fmt.Fprintf(&inputHTML, ` autocomplete="%s"`, escapeAttrValue(ac))
			}
		}

		// ARIA validation
		hasError := record.Errors != nil && record.Errors[fieldName] != nil
		if hasError {
			inputHTML.WriteString(` aria-invalid="true"`)
			fmt.Fprintf(&inputHTML, ` aria-describedby="%s-error"`, fieldName)
		} else if record.Validated {
			inputHTML.WriteString(` aria-invalid="false"`)
		}

		inputHTML.WriteString("/>")
	}

	// Checkbox: input before label; otherwise: label before input
	if isCheckbox {
		result.WriteString(inputHTML.String())
		result.WriteString(labelHTML)
	} else {
		result.WriteString(labelHTML)
		result.WriteString(inputHTML.String())
	}

	// Help text
	if helpText != "" {
		fmt.Fprintf(&result, `<span class="help">%s</span>`, escapeHTMLText(helpText))
	}

	// Error (only if present)
	if record.Errors != nil && record.Errors[fieldName] != nil {
		errorMsg := record.Errors[fieldName].Message
		if errorMsg != "" {
			fmt.Fprintf(&result, `<span id="%s-error" class="error" role="alert">%s</span>`,
				escapeAttrValue(fieldName), escapeHTMLText(errorMsg))
		}
	}

	result.WriteString("</div>")
	return &String{Value: result.String()}
}

// evalLabelComponent handles <label @field="name"/> and <label @field="name">...</label>
// Returns HTML label element with field title and accessibility attributes.
func evalLabelComponent(props string, contents []Object, isSelfClosing bool, env *Environment) Object {
	// Get field name from @field attribute
	fieldName := parseFieldAttribute(props)
	if fieldName == "" {
		return &Error{
			Class:   ClassValue,
			Code:    "FORM-0001",
			Message: "label element requires @field attribute",
			Hints:   []string{`Add @field attribute: <label @field="fieldName"/>`},
		}
	}

	// Get form context
	formCtx := getFormContext(env)
	if formCtx == nil {
		return &Error{
			Class:   ClassValue,
			Code:    "FORM-0002",
			Message: "label element must be inside a <form @record={...}> context",
			Hints:   []string{`Wrap in a form with @record: <form @record={myRecord}><label @field="name"/></form>`},
		}
	}

	record := formCtx.Record
	if record == nil || record.Schema == nil {
		return &Error{
			Class:   ClassValue,
			Code:    "FORM-0003",
			Message: "Form context has no valid record or schema",
		}
	}

	// Get optional @tag override (default is "label")
	tagName := parseTagAttribute(props)
	if tagName == "" {
		tagName = "label"
	}

	// Get field title
	field := record.Schema.Fields[fieldName]
	title := getFieldTitle(fieldName, field)

	// Build the label
	var result strings.Builder
	result.WriteString("<")
	result.WriteString(tagName)

	// Add 'for' attribute (only for actual <label> tags)
	if tagName == "label" {
		fmt.Fprintf(&result, ` for="%s"`, fieldName)
	}

	// Add any additional props (excluding @field, @tag, @key)
	cleanedProps := removeComponentAttributes(props)
	if cleanedProps = strings.TrimSpace(cleanedProps); cleanedProps != "" {
		result.WriteString(" ")
		result.WriteString(cleanedProps)
	}

	result.WriteString(">")

	// Add title text
	result.WriteString(escapeHTMLText(title))

	// For tag pairs, add contents after title
	if !isSelfClosing && len(contents) > 0 {
		for _, content := range contents {
			if strContent, ok := content.(*String); ok {
				result.WriteString(strContent.Value)
			} else {
				result.WriteString(content.Inspect())
			}
		}
	}

	// Close tag
	result.WriteString("</")
	result.WriteString(tagName)
	result.WriteString(">")

	return &String{Value: result.String()}
}

// evalErrorComponent handles <error @field="name"/> and <error @field="name">...</error>
// Returns error message if field has error, null otherwise.
func evalErrorComponent(props string, contents []Object, isSelfClosing bool, env *Environment) Object {
	// Get field name from @field attribute
	fieldName := parseFieldAttribute(props)
	if fieldName == "" {
		return &Error{
			Class:   ClassValue,
			Code:    "FORM-0001",
			Message: "error element requires @field attribute",
			Hints:   []string{`Add @field attribute: <error @field="fieldName"/>`},
		}
	}

	// Get form context
	formCtx := getFormContext(env)
	if formCtx == nil {
		return &Error{
			Class:   ClassValue,
			Code:    "FORM-0002",
			Message: "error element must be inside a <form @record={...}> context",
			Hints:   []string{`Wrap in a form with @record: <form @record={myRecord}><error @field="name"/></form>`},
		}
	}

	record := formCtx.Record
	if record == nil {
		return &Error{
			Class:   ClassValue,
			Code:    "FORM-0003",
			Message: "Form context has no valid record",
		}
	}

	// Check if field has error
	if record.Errors == nil || record.Errors[fieldName] == nil {
		return NULL // No error - render nothing
	}

	// Check if error has a message (empty message = state-only error)
	errorMsg := record.Errors[fieldName].Message
	if errorMsg == "" {
		return NULL // Error state only, no message to display
	}

	// Get optional @tag override (default is "span")
	tagName := parseTagAttribute(props)
	if tagName == "" {
		tagName = "span"
	}

	// Build the error element
	var result strings.Builder
	result.WriteString("<")
	result.WriteString(tagName)
	fmt.Fprintf(&result, ` id="%s-error"`, fieldName)
	result.WriteString(` class="error"`)
	result.WriteString(` role="alert"`)

	// Add any additional props (excluding @field, @tag, @key)
	cleanedProps := removeComponentAttributes(props)
	if cleanedProps = strings.TrimSpace(cleanedProps); cleanedProps != "" {
		result.WriteString(" ")
		result.WriteString(cleanedProps)
	}

	result.WriteString(">")

	// Add error message (already retrieved above)
	result.WriteString(escapeHTMLText(errorMsg))

	// For tag pairs, add contents after error message
	if !isSelfClosing && len(contents) > 0 {
		for _, content := range contents {
			if strContent, ok := content.(*String); ok {
				result.WriteString(strContent.Value)
			} else {
				result.WriteString(content.Inspect())
			}
		}
	}

	// Close tag
	result.WriteString("</")
	result.WriteString(tagName)
	result.WriteString(">")

	return &String{Value: result.String()}
}

// evalValComponent handles <val @field="name" @key="help"/> and <val @field="name" @key="help">...</val>
// Returns metadata value if present, null otherwise.
func evalValComponent(props string, contents []Object, isSelfClosing bool, env *Environment) Object {
	// Get field name from @field attribute
	fieldName := parseFieldAttribute(props)
	if fieldName == "" {
		return &Error{
			Class:   ClassValue,
			Code:    "FORM-0001",
			Message: "val element requires @field attribute",
			Hints:   []string{`Add @field attribute: <val @field="fieldName" @key="help"/>`},
		}
	}

	// Get key from @key attribute
	keyName := parseKeyAttribute(props)
	if keyName == "" {
		return &Error{
			Class:   ClassValue,
			Code:    "FORM-0004",
			Message: "val element requires @key attribute",
			Hints:   []string{`Add @key attribute: <val @field="fieldName" @key="help"/>`},
		}
	}

	// Get form context
	formCtx := getFormContext(env)
	if formCtx == nil {
		return &Error{
			Class:   ClassValue,
			Code:    "FORM-0002",
			Message: "val element must be inside a <form @record={...}> context",
			Hints:   []string{`Wrap in a form with @record: <form @record={myRecord}><val @field="name" @key="help"/></form>`},
		}
	}

	record := formCtx.Record
	if record == nil || record.Schema == nil {
		return &Error{
			Class:   ClassValue,
			Code:    "FORM-0003",
			Message: "Form context has no valid record or schema",
		}
	}

	// Get field from schema
	field := record.Schema.Fields[fieldName]
	if field == nil || field.Metadata == nil {
		return NULL // No metadata - render nothing
	}

	// Get metadata value
	value, ok := field.Metadata[keyName]
	if !ok {
		return NULL // No such metadata key - render nothing
	}

	// Get optional @tag override (default is "span")
	tagName := parseTagAttribute(props)
	if tagName == "" {
		tagName = "span"
	}

	// Build the val element
	var result strings.Builder
	result.WriteString("<")
	result.WriteString(tagName)

	// Add any additional props (excluding @field, @tag, @key)
	cleanedProps := removeComponentAttributes(props)
	if cleanedProps = strings.TrimSpace(cleanedProps); cleanedProps != "" {
		result.WriteString(" ")
		result.WriteString(cleanedProps)
	}

	result.WriteString(">")

	// Add metadata value
	if strVal, ok := value.(*String); ok {
		result.WriteString(escapeHTMLText(strVal.Value))
	} else {
		result.WriteString(escapeHTMLText(objectToTemplateString(value)))
	}

	// For tag pairs, add contents after metadata value
	if !isSelfClosing && len(contents) > 0 {
		for _, content := range contents {
			if strContent, ok := content.(*String); ok {
				result.WriteString(strContent.Value)
			} else {
				result.WriteString(content.Inspect())
			}
		}
	}

	// Close tag
	result.WriteString("</")
	result.WriteString(tagName)
	result.WriteString(">")

	return &String{Value: result.String()}
}

// evalSelectComponent handles <select @field="name"/>
// Returns a <select> element with options from enum values.
func evalSelectComponent(props string, env *Environment) Object {
	// Get field name from @field attribute
	fieldName := parseFieldAttribute(props)
	if fieldName == "" {
		return &Error{
			Class:   ClassValue,
			Code:    "FORM-0001",
			Message: "select element requires @field attribute",
			Hints:   []string{`Add @field attribute: <select @field="fieldName"/>`},
		}
	}

	// Get form context
	formCtx := getFormContext(env)
	if formCtx == nil {
		return &Error{
			Class:   ClassValue,
			Code:    "FORM-0002",
			Message: "select element must be inside a <form @record={...}> context",
			Hints:   []string{`Wrap in a form with @record: <form @record={myRecord}><select @field="status"/></form>`},
		}
	}

	record := formCtx.Record
	if record == nil || record.Schema == nil {
		return &Error{
			Class:   ClassValue,
			Code:    "FORM-0003",
			Message: "Form context has no valid record or schema",
		}
	}

	// Get field from schema
	field := record.Schema.Fields[fieldName]

	// Get current value
	currentValue := record.Get(fieldName, record.Env)
	currentStr := ""
	if currentValue != nil && currentValue != NULL {
		currentStr = objectToTemplateString(currentValue)
	}

	// Get placeholder (from props, metadata, or empty)
	placeholder := ""
	if found := strings.Contains(props, "placeholder="); found {
		placeholder = parseAttrValue(props, "placeholder")
	} else if field != nil && field.Metadata != nil {
		if ph, ok := field.Metadata["placeholder"]; ok {
			if strPh, ok := ph.(*String); ok {
				placeholder = strPh.Value
			}
		}
	}

	// Build the select element
	var result strings.Builder
	result.WriteString("<select")
	fmt.Fprintf(&result, ` name="%s"`, fieldName)

	// Add required if field is required
	if field != nil && field.Required {
		result.WriteString(` required`)
		result.WriteString(` aria-required="true"`)
	}

	// Add ARIA validation attributes
	hasError := record.Errors != nil && record.Errors[fieldName] != nil
	if hasError {
		result.WriteString(` aria-invalid="true"`)
		fmt.Fprintf(&result, ` aria-describedby="%s-error"`, fieldName)
	} else if record.Validated {
		result.WriteString(` aria-invalid="false"`)
	}

	// Add autocomplete attribute (FEAT-097)
	if field != nil {
		if autocomplete := getAutocomplete(fieldName, field.Type, field.Metadata); autocomplete != "" {
			fmt.Fprintf(&result, ` autocomplete="%s"`, escapeAttrValue(autocomplete))
		}
	}

	// Add any additional props (excluding @field, @tag, @key, placeholder)
	cleanedProps := removeComponentAttributes(props)
	// Also remove placeholder
	cleanedProps = removeAttr(cleanedProps, "placeholder")
	if cleanedProps = strings.TrimSpace(cleanedProps); cleanedProps != "" {
		result.WriteString(" ")
		result.WriteString(cleanedProps)
	}

	result.WriteString(">")

	// Add placeholder option
	if placeholder != "" {
		result.WriteString(`<option value="">`)
		result.WriteString(escapeHTMLText(placeholder))
		result.WriteString("</option>")
	}

	// Add options from enum values
	if field != nil && len(field.EnumValues) > 0 {
		for _, enumVal := range field.EnumValues {
			result.WriteString(`<option value="`)
			result.WriteString(escapeAttrValue(enumVal))
			result.WriteString(`"`)
			if enumVal == currentStr {
				result.WriteString(` selected`)
			}
			result.WriteString(">")
			result.WriteString(escapeHTMLText(enumVal))
			result.WriteString("</option>")
		}
	}

	result.WriteString("</select>")

	return &String{Value: result.String()}
}

// parseAttrValue parses a specific attribute value from props string.
func parseAttrValue(props string, attrName string) string {
	idx := strings.Index(props, attrName+"=")
	if idx == -1 {
		return ""
	}

	valueStart := idx + len(attrName) + 1
	for valueStart < len(props) && props[valueStart] == ' ' {
		valueStart++
	}
	if valueStart >= len(props) {
		return ""
	}

	if props[valueStart] == '"' {
		valueEnd := valueStart + 1
		for valueEnd < len(props) && props[valueEnd] != '"' {
			if props[valueEnd] == '\\' {
				valueEnd++
			}
			valueEnd++
		}
		return props[valueStart+1 : valueEnd]
	}

	return ""
}

// removeAttr removes a specific attribute from props.
func removeAttr(props string, attrName string) string {
	// Pattern: attrName="..."
	idx := strings.Index(props, attrName+"=")
	if idx == -1 {
		return props
	}

	// Find end of attribute
	valueStart := idx + len(attrName) + 1
	for valueStart < len(props) && props[valueStart] == ' ' {
		valueStart++
	}
	if valueStart >= len(props) {
		return props[:idx]
	}

	if props[valueStart] == '"' {
		valueEnd := valueStart + 1
		for valueEnd < len(props) && props[valueEnd] != '"' {
			if props[valueEnd] == '\\' {
				valueEnd++
			}
			valueEnd++
		}
		if valueEnd < len(props) {
			valueEnd++ // Include closing quote
		}
		return strings.TrimSpace(props[:idx] + props[valueEnd:])
	}

	return props
}

// escapeHTMLText escapes text for HTML content (not attributes).
func escapeHTMLText(s string) string {
	var result strings.Builder
	for _, c := range s {
		switch c {
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
