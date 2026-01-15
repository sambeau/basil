package evaluator

import (
	"fmt"
	"strings"
	"unicode"
)

// Validation error codes as defined in FEAT-091
const (
	ErrCodeRequired  = "REQUIRED"
	ErrCodeType      = "TYPE"
	ErrCodeFormat    = "FORMAT"
	ErrCodeEnum      = "ENUM"
	ErrCodeMinLength = "MIN_LENGTH"
	ErrCodeMaxLength = "MAX_LENGTH"
	ErrCodeMinValue  = "MIN_VALUE"
	ErrCodeMaxValue  = "MAX_VALUE"
	ErrCodeCustom    = "CUSTOM"
)

// Error message templates with placeholders
var errorTemplates = map[string]string{
	ErrCodeRequired:  "{title} is required",
	ErrCodeType:      "{title} must be a {type}",
	ErrCodeFormat:    "{title} is not a valid {type}",
	ErrCodeEnum:      "{title} must be one of: {values}",
	ErrCodeMinLength: "{title} must be at least {min} characters",
	ErrCodeMaxLength: "{title} must be at most {max} characters",
	ErrCodeMinValue:  "{title} must be at least {min}",
	ErrCodeMaxValue:  "{title} must be at most {max}",
}

// ValidateRecord validates a record against its schema and returns a new Record
// with validation errors populated. The original record is unchanged.
func ValidateRecord(record *Record, env *Environment) *Record {
	if record == nil || record.Schema == nil {
		return record
	}

	errors := make(map[string]*RecordError)

	// Validate each field defined in the schema
	for fieldName, field := range record.Schema.Fields {
		if err := validateField(record, fieldName, field, env); err != nil {
			errors[fieldName] = err
		}
	}

	// Create new record with validation results
	return &Record{
		Schema:    record.Schema,
		Data:      record.Data,
		KeyOrder:  record.KeyOrder,
		Errors:    errors,
		Validated: true,
		Env:       record.Env,
	}
}

// validateField validates a single field and returns an error if validation fails.
// Returns nil if the field is valid.
// Validation order: required → type → format → constraints → enum
func validateField(record *Record, fieldName string, field *DSLSchemaField, env *Environment) *RecordError {
	// Get the field value
	value := record.Get(fieldName, env)

	// Get title for error messages
	title := getFieldTitle(fieldName, field)

	// 1. Required check
	if field.Required {
		if isNullOrMissing(value) {
			return &RecordError{
				Code:    ErrCodeRequired,
				Message: formatErrorMessage(ErrCodeRequired, map[string]string{"title": title}),
			}
		}
	}

	// If value is null/missing and not required, skip further validation
	if isNullOrMissing(value) {
		return nil
	}

	// 2. Type check
	if err := validateType(value, field, title); err != nil {
		return err
	}

	// 3. Format check for validated string types
	if err := validateFormat(value, field, title); err != nil {
		return err
	}

	// 4. Constraint checks (min/max)
	if err := validateConstraints(value, field, title); err != nil {
		return err
	}

	// 5. Enum check
	if err := validateEnum(value, field, title); err != nil {
		return err
	}

	return nil
}

// isNullOrMissing checks if a value is null or missing.
// Note: Empty string "", zero 0, and false pass required check.
func isNullOrMissing(value Object) bool {
	if value == nil {
		return true
	}
	if _, ok := value.(*Null); ok {
		return true
	}
	return false
}

// validateType checks if the value matches the expected type.
func validateType(value Object, field *DSLSchemaField, title string) *RecordError {
	baseType := strings.TrimSuffix(strings.ToLower(field.Type), "?")

	switch baseType {
	case "int", "integer", "bigint":
		if _, ok := value.(*Integer); !ok {
			return &RecordError{
				Code:    ErrCodeType,
				Message: formatErrorMessage(ErrCodeType, map[string]string{"title": title, "type": "integer"}),
			}
		}
	case "float", "number":
		switch value.(type) {
		case *Float, *Integer: // Integer is acceptable for float fields
			// OK
		default:
			return &RecordError{
				Code:    ErrCodeType,
				Message: formatErrorMessage(ErrCodeType, map[string]string{"title": title, "type": "number"}),
			}
		}
	case "bool", "boolean":
		if _, ok := value.(*Boolean); !ok {
			return &RecordError{
				Code:    ErrCodeType,
				Message: formatErrorMessage(ErrCodeType, map[string]string{"title": title, "type": "boolean"}),
			}
		}
	case "string", "text", "email", "url", "phone", "slug", "uuid", "ulid":
		if _, ok := value.(*String); !ok {
			return &RecordError{
				Code:    ErrCodeType,
				Message: formatErrorMessage(ErrCodeType, map[string]string{"title": title, "type": "string"}),
			}
		}
	}
	// Other types: no type check (future: datetime, money, etc.)
	return nil
}

// validateFormat checks format for validated string types (email, url, phone, slug).
func validateFormat(value Object, field *DSLSchemaField, title string) *RecordError {
	strVal, ok := value.(*String)
	if !ok {
		return nil // Only validate strings
	}

	// Empty strings pass format validation (use required for non-empty)
	if strVal.Value == "" {
		return nil
	}

	validationType := field.ValidationType
	if validationType == "" {
		validationType = strings.ToLower(field.Type)
	}

	switch validationType {
	case "email":
		if !dslEmailRegex.MatchString(strVal.Value) {
			return &RecordError{
				Code:    ErrCodeFormat,
				Message: formatErrorMessage(ErrCodeFormat, map[string]string{"title": title, "type": "email address"}),
			}
		}
	case "url":
		if !dslURLRegex.MatchString(strVal.Value) {
			return &RecordError{
				Code:    ErrCodeFormat,
				Message: formatErrorMessage(ErrCodeFormat, map[string]string{"title": title, "type": "URL"}),
			}
		}
	case "phone":
		if !dslPhoneRegex.MatchString(strVal.Value) {
			return &RecordError{
				Code:    ErrCodeFormat,
				Message: formatErrorMessage(ErrCodeFormat, map[string]string{"title": title, "type": "phone number"}),
			}
		}
	case "slug":
		if !dslSlugRegex.MatchString(strVal.Value) {
			return &RecordError{
				Code:    ErrCodeFormat,
				Message: formatErrorMessage(ErrCodeFormat, map[string]string{"title": title, "type": "slug"}),
			}
		}
	case "uuid":
		if !isValidUUID(strVal.Value) {
			return &RecordError{
				Code:    ErrCodeFormat,
				Message: formatErrorMessage(ErrCodeFormat, map[string]string{"title": title, "type": "UUID"}),
			}
		}
	case "ulid":
		if !isValidULID(strVal.Value) {
			return &RecordError{
				Code:    ErrCodeFormat,
				Message: formatErrorMessage(ErrCodeFormat, map[string]string{"title": title, "type": "ULID"}),
			}
		}
	}

	return nil
}

// validateConstraints checks min/max constraints.
func validateConstraints(value Object, field *DSLSchemaField, title string) *RecordError {
	// String length constraints
	if strVal, ok := value.(*String); ok {
		length := len(strVal.Value)

		if field.MinLength != nil && length < *field.MinLength {
			return &RecordError{
				Code: ErrCodeMinLength,
				Message: formatErrorMessage(ErrCodeMinLength, map[string]string{
					"title": title,
					"min":   fmt.Sprintf("%d", *field.MinLength),
				}),
			}
		}

		if field.MaxLength != nil && length > *field.MaxLength {
			return &RecordError{
				Code: ErrCodeMaxLength,
				Message: formatErrorMessage(ErrCodeMaxLength, map[string]string{
					"title": title,
					"max":   fmt.Sprintf("%d", *field.MaxLength),
				}),
			}
		}
	}

	// Integer value constraints
	if intVal, ok := value.(*Integer); ok {
		if field.MinValue != nil && intVal.Value < *field.MinValue {
			return &RecordError{
				Code: ErrCodeMinValue,
				Message: formatErrorMessage(ErrCodeMinValue, map[string]string{
					"title": title,
					"min":   fmt.Sprintf("%d", *field.MinValue),
				}),
			}
		}

		if field.MaxValue != nil && intVal.Value > *field.MaxValue {
			return &RecordError{
				Code: ErrCodeMaxValue,
				Message: formatErrorMessage(ErrCodeMaxValue, map[string]string{
					"title": title,
					"max":   fmt.Sprintf("%d", *field.MaxValue),
				}),
			}
		}
	}

	// Float value constraints
	if floatVal, ok := value.(*Float); ok {
		if field.MinValue != nil && floatVal.Value < float64(*field.MinValue) {
			return &RecordError{
				Code: ErrCodeMinValue,
				Message: formatErrorMessage(ErrCodeMinValue, map[string]string{
					"title": title,
					"min":   fmt.Sprintf("%d", *field.MinValue),
				}),
			}
		}

		if field.MaxValue != nil && floatVal.Value > float64(*field.MaxValue) {
			return &RecordError{
				Code: ErrCodeMaxValue,
				Message: formatErrorMessage(ErrCodeMaxValue, map[string]string{
					"title": title,
					"max":   fmt.Sprintf("%d", *field.MaxValue),
				}),
			}
		}
	}

	return nil
}

// validateEnum checks if value is in the allowed enum values.
func validateEnum(value Object, field *DSLSchemaField, title string) *RecordError {
	if len(field.EnumValues) == 0 {
		return nil
	}

	strVal, ok := value.(*String)
	if !ok {
		return nil // Enum only applies to strings
	}

	for _, allowed := range field.EnumValues {
		if strVal.Value == allowed {
			return nil // Found a match
		}
	}

	return &RecordError{
		Code: ErrCodeEnum,
		Message: formatErrorMessage(ErrCodeEnum, map[string]string{
			"title":  title,
			"values": strings.Join(field.EnumValues, ", "),
		}),
	}
}

// getFieldTitle returns the display title for a field.
// Priority: explicit title in metadata → titlecase of field name
func getFieldTitle(fieldName string, field *DSLSchemaField) string {
	// TODO: Check field.Metadata for explicit title when metadata is implemented
	// For now, convert field name to title case
	return toTitleCase(fieldName)
}

// toTitleCase converts a camelCase or snake_case identifier to Title Case.
// Examples: "firstName" → "First Name", "email" → "Email", "user_name" → "User Name"
func toTitleCase(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder
	wordStart := true

	for i, r := range s {
		if r == '_' || r == '-' {
			result.WriteRune(' ')
			wordStart = true
			continue
		}

		// Check for camelCase boundary (lowercase followed by uppercase)
		if i > 0 && unicode.IsUpper(r) && !wordStart {
			result.WriteRune(' ')
		}

		if wordStart {
			result.WriteRune(unicode.ToUpper(r))
			wordStart = false
		} else {
			result.WriteRune(unicode.ToLower(r))
		}
	}

	return result.String()
}

// formatErrorMessage replaces placeholders in error template with values.
func formatErrorMessage(code string, values map[string]string) string {
	template, ok := errorTemplates[code]
	if !ok {
		return code // Fall back to code if no template
	}

	result := template
	for key, value := range values {
		result = strings.ReplaceAll(result, "{"+key+"}", value)
	}
	return result
}

// isValidUUID checks if a string is a valid UUID format.
func isValidUUID(s string) bool {
	// UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	if len(s) != 36 {
		return false
	}
	for i, r := range s {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return false
			}
		default:
			if !isHexDigit(r) {
				return false
			}
		}
	}
	return true
}

// isValidULID checks if a string is a valid ULID format.
func isValidULID(s string) bool {
	// ULID format: 26 characters from Crockford's base32 alphabet
	if len(s) != 26 {
		return false
	}
	for _, r := range s {
		if !isBase32Char(r) {
			return false
		}
	}
	return true
}

// isHexDigit checks if a rune is a valid hex digit.
func isHexDigit(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

// isBase32Char checks if a rune is valid in Crockford's base32 (for ULID).
func isBase32Char(r rune) bool {
	// Crockford's base32: 0-9, A-Z excluding I, L, O, U
	if r >= '0' && r <= '9' {
		return true
	}
	if r >= 'A' && r <= 'Z' && r != 'I' && r != 'L' && r != 'O' && r != 'U' {
		return true
	}
	if r >= 'a' && r <= 'z' && r != 'i' && r != 'l' && r != 'o' && r != 'u' {
		return true
	}
	return false
}
