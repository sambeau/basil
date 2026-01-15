package evaluator

import (
	"fmt"
	"sort"
	"time"

	"github.com/sambeau/basil/pkg/parsley/ast"
)

// Available methods on Record objects
var recordMethods = []string{
	"validate", "update", "errors", "error", "errorCode", "errorList",
	"isValid", "hasError", "schema", "data", "keys", "withError",
	"title", "placeholder", "meta", "enumValues", "format",
}

// evalRecordMethod dispatches method calls on Record objects.
func evalRecordMethod(record *Record, method string, args []Object, env *Environment) Object {
	switch method {
	case "validate":
		return recordValidate(record, args, env)
	case "update":
		return recordUpdate(record, args, env)
	case "errors":
		return recordErrors(record, args)
	case "error":
		return recordError(record, args, env)
	case "errorCode":
		return recordErrorCode(record, args, env)
	case "errorList":
		return recordErrorList(record, args)
	case "isValid":
		return recordIsValid(record, args)
	case "hasError":
		return recordHasError(record, args, env)
	case "schema":
		return recordSchema(record, args)
	case "data":
		return recordData(record, args)
	case "keys":
		return recordKeys(record, args)
	case "withError":
		return recordWithError(record, args)
	case "title":
		return recordTitle(record, args, env)
	case "placeholder":
		return recordPlaceholder(record, args, env)
	case "meta":
		return recordMeta(record, args, env)
	case "enumValues":
		return recordEnumValues(record, args, env)
	case "format":
		return recordFormat(record, args, env)
	default:
		// Check if it's a data field access via method syntax (shouldn't happen normally)
		return unknownMethodError(method, "Record", recordMethods)
	}
}

// recordValidate implements record.validate() → Record
func recordValidate(record *Record, args []Object, env *Environment) Object {
	if len(args) != 0 {
		return newArityError("validate", len(args), 0)
	}
	return ValidateRecord(record, env)
}

// recordUpdate implements record.update({...}) → Record
// Merges fields and auto-revalidates.
func recordUpdate(record *Record, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("update", len(args), 1)
	}

	dict, ok := args[0].(*Dictionary)
	if !ok {
		return newTypeError("TYPE-0001", "Record.update", "dictionary", args[0].Type())
	}

	// Create new record with merged data
	newRecord := record.Clone()
	newRecord.Validated = false
	newRecord.Errors = nil

	// Merge fields from dict
	for key, expr := range dict.Pairs {
		// Only merge fields that are in the schema
		if _, inSchema := record.Schema.Fields[key]; inSchema {
			value := Eval(expr, dict.Env)
			if !isError(value) {
				field := record.Schema.Fields[key]
				castedValue := castFieldValue(value, field)
				newRecord.Data[key] = &ast.ObjectLiteralExpression{Obj: castedValue}
				// Add to KeyOrder if not already present
				found := false
				for _, k := range newRecord.KeyOrder {
					if k == key {
						found = true
						break
					}
				}
				if !found {
					newRecord.KeyOrder = append(newRecord.KeyOrder, key)
				}
			}
		}
	}

	// Auto-revalidate
	return ValidateRecord(newRecord, env)
}

// recordErrors implements record.errors() → Dictionary
func recordErrors(record *Record, args []Object) Object {
	if len(args) != 0 {
		return newArityError("errors", len(args), 0)
	}

	// Convert RecordError map to Dictionary
	pairs := make(map[string]ast.Expression)
	keyOrder := []string{}

	for fieldName, err := range record.Errors {
		// Create error dict: {code: "...", message: "..."}
		errDict := make(map[string]ast.Expression)
		errDict["code"] = &ast.StringLiteral{Value: err.Code}
		errDict["message"] = &ast.StringLiteral{Value: err.Message}
		pairs[fieldName] = &ast.DictionaryLiteral{Pairs: errDict}
		keyOrder = append(keyOrder, fieldName)
	}

	sort.Strings(keyOrder) // Consistent ordering

	return &Dictionary{
		Pairs:    pairs,
		KeyOrder: keyOrder,
	}
}

// recordError implements record.error(field) → String or null
func recordError(record *Record, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("error", len(args), 1)
	}

	fieldName, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "Record.error", "string", args[0].Type())
	}

	if err, exists := record.Errors[fieldName.Value]; exists {
		return &String{Value: err.Message}
	}
	return NULL
}

// recordErrorCode implements record.errorCode(field) → String or null
func recordErrorCode(record *Record, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("errorCode", len(args), 1)
	}

	fieldName, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "Record.errorCode", "string", args[0].Type())
	}

	if err, exists := record.Errors[fieldName.Value]; exists {
		return &String{Value: err.Code}
	}
	return NULL
}

// recordErrorList implements record.errorList() → Array
// Returns [{field, code, message}, ...]
func recordErrorList(record *Record, args []Object) Object {
	if len(args) != 0 {
		return newArityError("errorList", len(args), 0)
	}

	elements := []Object{}

	// Sort field names for consistent ordering
	fields := make([]string, 0, len(record.Errors))
	for field := range record.Errors {
		fields = append(fields, field)
	}
	sort.Strings(fields)

	for _, field := range fields {
		err := record.Errors[field]
		errDict := &Dictionary{
			Pairs: map[string]ast.Expression{
				"field":   &ast.StringLiteral{Value: field},
				"code":    &ast.StringLiteral{Value: err.Code},
				"message": &ast.StringLiteral{Value: err.Message},
			},
			KeyOrder: []string{"field", "code", "message"},
		}
		elements = append(elements, errDict)
	}

	return &Array{Elements: elements}
}

// recordIsValid implements record.isValid() → Boolean
// Returns true if validated AND no errors
func recordIsValid(record *Record, args []Object) Object {
	if len(args) != 0 {
		return newArityError("isValid", len(args), 0)
	}

	if !record.Validated {
		return FALSE
	}
	return &Boolean{Value: len(record.Errors) == 0}
}

// recordHasError implements record.hasError(field) → Boolean
func recordHasError(record *Record, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("hasError", len(args), 1)
	}

	fieldName, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "Record.hasError", "string", args[0].Type())
	}

	_, exists := record.Errors[fieldName.Value]
	return &Boolean{Value: exists}
}

// recordSchema implements record.schema() → Schema
func recordSchema(record *Record, args []Object) Object {
	if len(args) != 0 {
		return newArityError("schema", len(args), 0)
	}

	if record.Schema == nil {
		return NULL
	}
	return record.Schema
}

// recordData implements record.data() → Dictionary
func recordData(record *Record, args []Object) Object {
	if len(args) != 0 {
		return newArityError("data", len(args), 0)
	}

	return record.ToDictionary()
}

// recordKeys implements record.keys() → Array
// Returns field names from schema (not just data keys)
func recordKeys(record *Record, args []Object) Object {
	if len(args) != 0 {
		return newArityError("keys", len(args), 0)
	}

	if record.Schema == nil {
		return &Array{Elements: []Object{}}
	}

	// Get field names from schema, sorted for consistency
	keys := make([]string, 0, len(record.Schema.Fields))
	for name := range record.Schema.Fields {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	elements := make([]Object, len(keys))
	for i, key := range keys {
		elements[i] = &String{Value: key}
	}

	return &Array{Elements: elements}
}

// recordWithError implements record.withError(field, msg) or record.withError(field, code, msg)
// Adds a custom error without revalidation
func recordWithError(record *Record, args []Object) Object {
	if len(args) < 2 || len(args) > 3 {
		return newArityError("withError", len(args), 2)
	}

	fieldName, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "Record.withError", "string (field)", args[0].Type())
	}

	var code, message string

	if len(args) == 2 {
		// withError(field, msg) - use CUSTOM code
		msg, ok := args[1].(*String)
		if !ok {
			return newTypeError("TYPE-0001", "Record.withError", "string (message)", args[1].Type())
		}
		code = ErrCodeCustom
		message = msg.Value
	} else {
		// withError(field, code, msg)
		codeArg, ok := args[1].(*String)
		if !ok {
			return newTypeError("TYPE-0001", "Record.withError", "string (code)", args[1].Type())
		}
		msgArg, ok := args[2].(*String)
		if !ok {
			return newTypeError("TYPE-0001", "Record.withError", "string (message)", args[2].Type())
		}
		code = codeArg.Value
		message = msgArg.Value
	}

	// Clone record and add error
	newRecord := record.Clone()
	if newRecord.Errors == nil {
		newRecord.Errors = make(map[string]*RecordError)
	}
	newRecord.Errors[fieldName.Value] = &RecordError{
		Code:    code,
		Message: message,
	}
	// Mark as validated since we're adding custom error
	newRecord.Validated = true

	return newRecord
}

// recordTitle implements record.title(field) → String
// Shorthand for record.schema().title(field)
func recordTitle(record *Record, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("title", len(args), 1)
	}

	fieldName, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "Record.title", "string", args[0].Type())
	}

	if record.Schema == nil {
		return &String{Value: toTitleCase(fieldName.Value)}
	}

	field, exists := record.Schema.Fields[fieldName.Value]
	if !exists {
		return &String{Value: toTitleCase(fieldName.Value)}
	}

	return &String{Value: getFieldTitle(fieldName.Value, field)}
}

// recordPlaceholder implements record.placeholder(field) → String or null
// Shorthand for record.meta(field, "placeholder")
func recordPlaceholder(record *Record, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("placeholder", len(args), 1)
	}

	fieldName, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "Record.placeholder", "string", args[0].Type())
	}

	if record.Schema == nil {
		return NULL
	}

	field, exists := record.Schema.Fields[fieldName.Value]
	if !exists {
		return NULL
	}

	if field.Metadata != nil {
		if placeholder, ok := field.Metadata["placeholder"]; ok {
			return placeholder
		}
	}

	return NULL
}

// recordMeta implements record.meta(field, key) → Any or null
// Shorthand for record.schema().meta(field, key)
func recordMeta(record *Record, args []Object, env *Environment) Object {
	if len(args) != 2 {
		return newArityError("meta", len(args), 2)
	}

	fieldName, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "Record.meta", "string (field)", args[0].Type())
	}

	key, ok := args[1].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "Record.meta", "string (key)", args[1].Type())
	}

	if record.Schema == nil {
		return NULL
	}

	field, exists := record.Schema.Fields[fieldName.Value]
	if !exists {
		return NULL
	}

	if field.Metadata != nil {
		if value, ok := field.Metadata[key.Value]; ok {
			return value
		}
	}

	return NULL
}

// recordEnumValues implements record.enumValues(field) → Array
// Returns enum options for a field (empty if not enum)
func recordEnumValues(record *Record, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("enumValues", len(args), 1)
	}

	fieldName, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "Record.enumValues", "string", args[0].Type())
	}

	if record.Schema == nil {
		return &Array{Elements: []Object{}}
	}

	field, exists := record.Schema.Fields[fieldName.Value]
	if !exists || len(field.EnumValues) == 0 {
		return &Array{Elements: []Object{}}
	}

	elements := make([]Object, len(field.EnumValues))
	for i, val := range field.EnumValues {
		elements[i] = &String{Value: val}
	}

	return &Array{Elements: elements}
}

// evalRecordProperty evaluates property access on a Record.
// Data fields are accessed directly, metadata via methods.
func evalRecordProperty(record *Record, key string, env *Environment) Object {
	// Check if it's a data field
	if expr, ok := record.Data[key]; ok {
		evalEnv := record.Env
		if evalEnv == nil {
			evalEnv = env
		}
		return Eval(expr, evalEnv)
	}

	// Not a data field - return null (per spec: direct property access for data only)
	return NULL
}

// recordFormat implements record.format(field) → String
// Formats a field value based on schema metadata "format" hint
// Built-in formats: date, datetime, currency, percent, number
func recordFormat(record *Record, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("format", len(args), 1)
	}

	fieldName, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "Record.format", "string", args[0].Type())
	}

	// Get the field value
	expr, exists := record.Data[fieldName.Value]
	if !exists {
		return NULL
	}

	evalEnv := record.Env
	if evalEnv == nil {
		evalEnv = env
	}
	value := Eval(expr, evalEnv)

	// Get format hint from schema metadata
	formatHint := ""
	if record.Schema != nil {
		if field, ok := record.Schema.Fields[fieldName.Value]; ok && field.Metadata != nil {
			if fmtObj, ok := field.Metadata["format"]; ok {
				if fmtStr, ok := fmtObj.(*String); ok {
					formatHint = fmtStr.Value
				}
			}
		}
	}

	// If no format hint, just return string representation
	if formatHint == "" {
		return &String{Value: objectToString(value)}
	}

	// Apply format based on hint
	switch formatHint {
	case "date":
		return formatRecordDate(value, env)
	case "datetime":
		return formatRecordDatetime(value, env)
	case "currency":
		return formatRecordCurrency(value)
	case "percent":
		return formatRecordPercent(value)
	case "number":
		return formatRecordNumber(value)
	default:
		// Unknown format, return string representation
		return &String{Value: objectToString(value)}
	}
}

// formatRecordDate formats a value as a date string "Jan 2, 2006"
func formatRecordDate(value Object, env *Environment) Object {
	// Handle datetime dictionary
	if dict, ok := value.(*Dictionary); ok && isDatetimeDict(dict) {
		return formatDateWithStyleAndLocale(dict, "long", "en-US", env)
	}

	// Handle ISO date string
	if str, ok := value.(*String); ok {
		// Parse ISO date string
		t, err := parseISODate(str.Value)
		if err != nil {
			return &String{Value: str.Value}
		}
		return &String{Value: t.Format("Jan 2, 2006")}
	}

	return &String{Value: objectToString(value)}
}

// formatRecordDatetime formats a value as a datetime string "Jan 2, 2006 3:04 PM"
func formatRecordDatetime(value Object, env *Environment) Object {
	// Handle datetime dictionary
	if dict, ok := value.(*Dictionary); ok && isDatetimeDict(dict) {
		return formatDateWithStyleAndLocale(dict, "long", "en-US", env)
	}

	// Handle ISO datetime string
	if str, ok := value.(*String); ok {
		t, err := parseISODate(str.Value)
		if err != nil {
			return &String{Value: str.Value}
		}
		return &String{Value: t.Format("Jan 2, 2006 3:04 PM")}
	}

	return &String{Value: objectToString(value)}
}

// formatRecordCurrency formats a numeric value as currency "$1,234.00"
func formatRecordCurrency(value Object) Object {
	var num float64
	switch v := value.(type) {
	case *Integer:
		num = float64(v.Value)
	case *Float:
		num = v.Value
	default:
		return &String{Value: objectToString(value)}
	}

	return formatCurrencyWithLocale(num, "USD", "en-US")
}

// formatRecordPercent formats a decimal as percentage "15%"
func formatRecordPercent(value Object) Object {
	var num float64
	switch v := value.(type) {
	case *Integer:
		num = float64(v.Value)
	case *Float:
		num = v.Value
	default:
		return &String{Value: objectToString(value)}
	}

	return formatPercentWithLocale(num, "en-US")
}

// formatRecordNumber formats a number with thousands separators "1,234,567"
func formatRecordNumber(value Object) Object {
	var num float64
	switch v := value.(type) {
	case *Integer:
		num = float64(v.Value)
	case *Float:
		num = v.Value
	default:
		return &String{Value: objectToString(value)}
	}

	return formatNumberWithLocale(num, "en-US")
}

// parseISODate attempts to parse an ISO date/datetime string
func parseISODate(s string) (time.Time, error) {
	// Try various ISO formats
	formats := []string{
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", s)
}
