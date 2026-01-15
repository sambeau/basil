package evaluator

import (
	"fmt"
	"strings"
)

// evalLabelComponent handles <Label @field="name"/> and <Label @field="name">...</Label>
// Returns HTML label element with field title and accessibility attributes.
func evalLabelComponent(props string, contents []Object, isSelfClosing bool, env *Environment) Object {
	// Get field name from @field attribute
	fieldName := parseFieldAttribute(props)
	if fieldName == "" {
		return &Error{
			Class:   ClassValue,
			Code:    "FORM-0001",
			Message: "Label component requires @field attribute",
			Hints:   []string{`Add @field attribute: <Label @field="fieldName"/>`},
		}
	}

	// Get form context
	formCtx := getFormContext(env)
	if formCtx == nil {
		return &Error{
			Class:   ClassValue,
			Code:    "FORM-0002",
			Message: "Label component must be inside a <form @record={...}> context",
			Hints:   []string{`Wrap in a form with @record: <form @record={myRecord}><Label @field="name"/></form>`},
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
		result.WriteString(fmt.Sprintf(` for="%s"`, fieldName))
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

// evalErrorComponent handles <Error @field="name"/>
// Returns error message if field has error, null otherwise.
func evalErrorComponent(props string, env *Environment) Object {
	// Get field name from @field attribute
	fieldName := parseFieldAttribute(props)
	if fieldName == "" {
		return &Error{
			Class:   ClassValue,
			Code:    "FORM-0001",
			Message: "Error component requires @field attribute",
			Hints:   []string{`Add @field attribute: <Error @field="fieldName"/>`},
		}
	}

	// Get form context
	formCtx := getFormContext(env)
	if formCtx == nil {
		return &Error{
			Class:   ClassValue,
			Code:    "FORM-0002",
			Message: "Error component must be inside a <form @record={...}> context",
			Hints:   []string{`Wrap in a form with @record: <form @record={myRecord}><Error @field="name"/></form>`},
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

	// Get optional @tag override (default is "span")
	tagName := parseTagAttribute(props)
	if tagName == "" {
		tagName = "span"
	}

	// Build the error element
	var result strings.Builder
	result.WriteString("<")
	result.WriteString(tagName)
	result.WriteString(fmt.Sprintf(` id="%s-error"`, fieldName))
	result.WriteString(` class="error"`)
	result.WriteString(` role="alert"`)

	// Add any additional props (excluding @field, @tag, @key)
	cleanedProps := removeComponentAttributes(props)
	if cleanedProps = strings.TrimSpace(cleanedProps); cleanedProps != "" {
		result.WriteString(" ")
		result.WriteString(cleanedProps)
	}

	result.WriteString(">")

	// Add error message
	errorMsg := record.Errors[fieldName].Message
	result.WriteString(escapeHTMLText(errorMsg))

	// Close tag
	result.WriteString("</")
	result.WriteString(tagName)
	result.WriteString(">")

	return &String{Value: result.String()}
}

// evalMetaComponent handles <Meta @field="name" @key="help"/>
// Returns metadata value if present, null otherwise.
func evalMetaComponent(props string, env *Environment) Object {
	// Get field name from @field attribute
	fieldName := parseFieldAttribute(props)
	if fieldName == "" {
		return &Error{
			Class:   ClassValue,
			Code:    "FORM-0001",
			Message: "Meta component requires @field attribute",
			Hints:   []string{`Add @field attribute: <Meta @field="fieldName" @key="help"/>`},
		}
	}

	// Get key from @key attribute
	keyName := parseKeyAttribute(props)
	if keyName == "" {
		return &Error{
			Class:   ClassValue,
			Code:    "FORM-0004",
			Message: "Meta component requires @key attribute",
			Hints:   []string{`Add @key attribute: <Meta @field="fieldName" @key="help"/>`},
		}
	}

	// Get form context
	formCtx := getFormContext(env)
	if formCtx == nil {
		return &Error{
			Class:   ClassValue,
			Code:    "FORM-0002",
			Message: "Meta component must be inside a <form @record={...}> context",
			Hints:   []string{`Wrap in a form with @record: <form @record={myRecord}><Meta @field="name" @key="help"/></form>`},
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

	// Build the meta element
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

	// Close tag
	result.WriteString("</")
	result.WriteString(tagName)
	result.WriteString(">")

	return &String{Value: result.String()}
}

// evalSelectComponent handles <Select @field="name"/>
// Returns a <select> element with options from enum values.
func evalSelectComponent(props string, env *Environment) Object {
	// Get field name from @field attribute
	fieldName := parseFieldAttribute(props)
	if fieldName == "" {
		return &Error{
			Class:   ClassValue,
			Code:    "FORM-0001",
			Message: "Select component requires @field attribute",
			Hints:   []string{`Add @field attribute: <Select @field="fieldName"/>`},
		}
	}

	// Get form context
	formCtx := getFormContext(env)
	if formCtx == nil {
		return &Error{
			Class:   ClassValue,
			Code:    "FORM-0002",
			Message: "Select component must be inside a <form @record={...}> context",
			Hints:   []string{`Wrap in a form with @record: <form @record={myRecord}><Select @field="status"/></form>`},
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
	if idx := strings.Index(props, "placeholder="); idx != -1 {
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
	result.WriteString(fmt.Sprintf(` name="%s"`, fieldName))

	// Add required if field is required
	if field != nil && field.Required {
		result.WriteString(` required`)
		result.WriteString(` aria-required="true"`)
	}

	// Add ARIA validation attributes
	hasError := record.Errors != nil && record.Errors[fieldName] != nil
	if hasError {
		result.WriteString(` aria-invalid="true"`)
		result.WriteString(fmt.Sprintf(` aria-describedby="%s-error"`, fieldName))
	} else if record.Validated {
		result.WriteString(` aria-invalid="false"`)
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
