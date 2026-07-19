package evaluator

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/sambeau/basil/pkg/parsley/ast"
)

// RecordMethodRegistry defines all methods available on Record objects.
var RecordMethodRegistry MethodRegistry

func init() {
	RecordMethodRegistry = MethodRegistry{
		"validate": {
			Fn:          recordMethodValidate,
			Arity:       "0",
			Description: "Validate the record against its schema",
		},
		"update": {
			Fn:          recordMethodUpdate,
			Arity:       "1",
			Description: "Merge fields and revalidate (dict)",
		},
		"errors": {
			Fn:          recordMethodErrors,
			Arity:       "0-1",
			Description: "Get validation errors (field?)",
		},
		"error": {
			Fn:          recordMethodError,
			Arity:       "1",
			Description: "Get error message for a field",
		},
		"errorCode": {
			Fn:          recordMethodErrorCode,
			Arity:       "1",
			Description: "Get error code for a field",
		},
		"errorList": {
			Fn:          recordMethodErrorList,
			Arity:       "0",
			Description: "Get all errors as an array",
		},
		"isValid": {
			Fn:          recordMethodIsValid,
			Arity:       "0",
			Description: "Check if the record is valid",
		},
		"hasError": {
			Fn:          recordMethodHasError,
			Arity:       "1",
			Description: "Check if a field has an error",
		},
		"schema": {
			Fn:          recordMethodSchema,
			Arity:       "0",
			Description: "Get the record's schema",
		},
		"data": {
			Fn:          recordMethodData,
			Arity:       "0",
			Description: "Get all record data as a dictionary",
		},
		"keys": {
			Fn:          recordMethodKeys,
			Arity:       "0",
			Description: "Get all field names as an array",
		},
		"withError": {
			Fn:          recordMethodWithError,
			Arity:       "1-3",
			Description: "Return a copy of the record with an added error (field, message?, code?)",
		},
		"title": {
			Fn:          recordMethodTitle,
			Arity:       "1",
			Description: "Get display title for a field",
		},
		"placeholder": {
			Fn:          recordMethodPlaceholder,
			Arity:       "1",
			Description: "Get placeholder text for a field",
		},
		"meta": {
			Fn:          recordMethodMeta,
			Arity:       "2",
			Description: "Get metadata value for a field (field, key)",
		},
		"enumValues": {
			Fn:          recordMethodEnumValues,
			Arity:       "1",
			Description: "Get enum options for a field",
		},
		"format": {
			Fn:          recordMethodFormat,
			Arity:       "1-2",
			Description: "Format a field value (field, options?)",
		},
		"toJSON": {
			Fn:          recordMethodToJSON,
			Arity:       "0",
			Description: "Serialize record data to a JSON string",
		},
		"failIfInvalid": {
			Fn:          recordMethodFailIfInvalid,
			Arity:       "0-1",
			Description: "Fail with an error if the record is invalid (message?)",
		},
		"fieldProps": {
			Fn:          recordMethodFieldProps,
			Arity:       "1-2",
			Description: "Get form field props for a field (field, overrides?)",
		},
	}
	RegisterMethodRegistry("record", RecordMethodRegistry)
}

// evalRecordMethod dispatches method calls on Record objects.
func evalRecordMethod(record *Record, method string, args []Object, env *Environment) Object {
	result := dispatchFromRegistry(RecordMethodRegistry, record, method, args, env)
	if result != nil {
		return result
	}
	return unknownMethodError(method, "record", RecordMethodRegistry.Names())
}

// ============================================================================
// Record method registry wrappers
// ============================================================================

func recordMethodValidate(receiver Object, args []Object, env *Environment) Object {
	return recordValidate(receiver.(*Record), args, env)
}

func recordMethodUpdate(receiver Object, args []Object, env *Environment) Object {
	return recordUpdate(receiver.(*Record), args, env)
}

func recordMethodErrors(receiver Object, args []Object, env *Environment) Object {
	return recordErrors(receiver.(*Record), args)
}

func recordMethodError(receiver Object, args []Object, env *Environment) Object {
	return recordError(receiver.(*Record), args, env)
}

func recordMethodErrorCode(receiver Object, args []Object, env *Environment) Object {
	return recordErrorCode(receiver.(*Record), args, env)
}

func recordMethodErrorList(receiver Object, args []Object, env *Environment) Object {
	return recordErrorList(receiver.(*Record), args)
}

func recordMethodIsValid(receiver Object, args []Object, env *Environment) Object {
	return recordIsValid(receiver.(*Record), args)
}

func recordMethodHasError(receiver Object, args []Object, env *Environment) Object {
	return recordHasError(receiver.(*Record), args, env)
}

func recordMethodSchema(receiver Object, args []Object, env *Environment) Object {
	return recordSchema(receiver.(*Record), args)
}

func recordMethodData(receiver Object, args []Object, env *Environment) Object {
	return recordData(receiver.(*Record), args)
}

func recordMethodKeys(receiver Object, args []Object, env *Environment) Object {
	return recordKeys(receiver.(*Record), args)
}

func recordMethodWithError(receiver Object, args []Object, env *Environment) Object {
	return recordWithError(receiver.(*Record), args)
}

func recordMethodTitle(receiver Object, args []Object, env *Environment) Object {
	return recordTitle(receiver.(*Record), args, env)
}

func recordMethodPlaceholder(receiver Object, args []Object, env *Environment) Object {
	return recordPlaceholder(receiver.(*Record), args, env)
}

func recordMethodMeta(receiver Object, args []Object, env *Environment) Object {
	return recordMeta(receiver.(*Record), args, env)
}

func recordMethodEnumValues(receiver Object, args []Object, env *Environment) Object {
	return recordEnumValues(receiver.(*Record), args, env)
}

func recordMethodFormat(receiver Object, args []Object, env *Environment) Object {
	return recordFormat(receiver.(*Record), args, env)
}

func recordMethodToJSON(receiver Object, args []Object, env *Environment) Object {
	return recordToJSON(receiver.(*Record), args)
}

func recordMethodFailIfInvalid(receiver Object, args []Object, env *Environment) Object {
	return recordFailIfInvalid(receiver.(*Record), args)
}

func recordMethodFieldProps(receiver Object, args []Object, env *Environment) Object {
	return recordFieldProps(receiver.(*Record), args, env)
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
// ReadOnly fields are silently filtered out; Auto fields produce an error.
func recordUpdate(record *Record, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("update", len(args), 1)
	}

	dict, ok := args[0].(*Dictionary)
	if !ok {
		return newTypeError("TYPE-0001", "Record.update", "dictionary", args[0].Type())
	}

	// SPEC-AUTO-003: Check for attempts to change auto fields (error)
	// ReadOnly fields are silently filtered below (not an error)
	for key := range dict.Pairs {
		if field, inSchema := record.Schema.Fields[key]; inSchema && field.Auto {
			return &Error{
				Class:   ClassType,
				Code:    "RECORD-0001",
				Message: fmt.Sprintf("cannot update auto field '%s'", key),
				Hints:   []string{"Auto fields are immutable - they are generated by the database/server"},
			}
		}
	}

	// Create new record with merged data
	newRecord := record.Clone()
	newRecord.Validated = false
	newRecord.Errors = nil

	// Merge fields from dict
	for key, expr := range dict.Pairs {
		// Only merge fields that are in the schema
		if field, inSchema := record.Schema.Fields[key]; inSchema {
			// Skip readOnly fields silently - they cannot be set from client input
			if field.ReadOnly {
				continue
			}
			value := Eval(expr, dict.Env)
			if !isError(value) {
				castedValue := castFieldValue(value, field)
				newRecord.Data[key] = &ast.ObjectLiteralExpression{Obj: castedValue}
				// Add to KeyOrder if not already present
				found := slices.Contains(newRecord.KeyOrder, key)
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
	return nativeBoolToParsBoolean(len(record.Errors) == 0)
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
	return nativeBoolToParsBoolean(exists)
}

// recordFailIfInvalid implements record.failIfInvalid(msg?) → Record | Error
// Returns the record if valid (or not yet validated), fails with structured error if invalid.
// Optional msg parameter allows customizing the error message.
func recordFailIfInvalid(record *Record, args []Object) Object {
	if len(args) > 1 {
		return newArityErrorRange("failIfInvalid", len(args), 0, 1)
	}

	// Get custom message or use default
	message := "Validation failed"
	if len(args) == 1 {
		msgStr, ok := args[0].(*String)
		if !ok {
			return newTypeError("TYPE-0005", "failIfInvalid", "a string", args[0].Type())
		}
		message = msgStr.Value
	}

	// If not validated or valid, return record for chaining
	if !record.Validated || len(record.Errors) == 0 {
		return record
	}

	// Build fields array from errorList
	fields := recordErrorList(record, nil).(*Array)

	// Build unified error dict
	pairs := make(map[string]ast.Expression)
	pairs["status"] = objectToExpression(&Integer{Value: 400})
	pairs["code"] = objectToExpression(&String{Value: "VALIDATION"})
	pairs["message"] = objectToExpression(&String{Value: message})
	pairs["fields"] = objectToExpression(fields)
	dict := &Dictionary{
		Pairs:    pairs,
		KeyOrder: []string{"status", "code", "message", "fields"},
	}

	return &Error{
		Class:    ClassValue,
		Code:     "VALIDATION",
		Message:  message,
		UserDict: dict,
	}
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

// recordToJSON implements record.toJSON() → String
// Encodes only the data fields (not schema, errors, or metadata)
// Implements SPEC-DC-003
func recordToJSON(record *Record, args []Object) Object {
	if len(args) != 0 {
		return newArityError("toJSON", len(args), 0)
	}

	// Use ToDictionary() which returns only data fields
	dict := record.ToDictionary()
	jsonBytes, err := json.Marshal(objectToGo(dict))
	if err != nil {
		return newFormatError("FMT-0005", err)
	}
	return &String{Value: string(jsonBytes)}
}

// recordWithError implements record.withError(field), record.withError(field, msg), or record.withError(field, code, msg)
// Adds a custom error without revalidation
// - withError(field) - flags field as having error state (no message displayed)
// - withError(field, msg) - adds error with message
// - withError(field, code, msg) - adds error with custom code and message
func recordWithError(record *Record, args []Object) Object {
	if len(args) < 1 || len(args) > 3 {
		return newArityError("withError", len(args), 1)
	}

	fieldName, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "Record.withError", "string (field)", args[0].Type())
	}

	var code, message string

	if len(args) == 1 {
		// withError(field) - flag as error state only (empty message)
		code = ErrCodeCustom
		message = ""
	} else if len(args) == 2 {
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

	// Check if it's a method name - provide helpful error
	if slices.Contains(RecordMethodRegistry.Names(), key) {
		return methodAsPropertyError(key, "Record")
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

	// Get format hint from schema metadata and check for money type with currency
	formatHint := ""
	currency := "USD" // Default currency
	if record.Schema != nil {
		if field, ok := record.Schema.Fields[fieldName.Value]; ok {
			// Check for money type with currency metadata (SPEC-CUR-007, SPEC-CUR-008)
			if field.Type == "money" && field.Metadata != nil {
				if curObj, ok := field.Metadata["currency"]; ok {
					if curStr, ok := curObj.(*String); ok {
						currency = curStr.Value
						// Auto-set format hint for money fields
						if formatHint == "" {
							formatHint = "currency"
						}
					}
				}
			}
			// Check explicit format metadata
			if field.Metadata != nil {
				if fmtObj, ok := field.Metadata["format"]; ok {
					if fmtStr, ok := fmtObj.(*String); ok {
						formatHint = fmtStr.Value
					}
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
		return formatRecordCurrencyWithCode(value, currency)
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
		// Extract time from datetime dictionary and format with time
		var t time.Time
		if unixExpr, ok := dict.Pairs["unix"]; ok {
			evalEnv := dict.Env
			if evalEnv == nil {
				evalEnv = env
			}
			unixObj := Eval(unixExpr, evalEnv)
			if unixInt, ok := unixObj.(*Integer); ok {
				t = time.Unix(unixInt.Value, 0).UTC()
			}
		}
		return &String{Value: t.Format("Jan 2, 2006 3:04 PM")}
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

// formatRecordCurrencyWithCode formats a numeric value as currency with specified currency code
// SPEC-CUR-007, SPEC-CUR-008: Money fields use currency metadata for formatting
func formatRecordCurrencyWithCode(value Object, currency string) Object {
	var num float64
	switch v := value.(type) {
	case *Integer:
		num = float64(v.Value)
	case *Float:
		num = v.Value
	case *Money:
		return &String{Value: v.Inspect()}
	default:
		return &String{Value: objectToString(value)}
	}

	return formatCurrencyWithLocale(num, currency, "en-US")
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
// recordFieldProps implements record.fieldProps(field, overrides?) → Dictionary
// Returns a dictionary of form field props derived from the schema.
// This bridges schema metadata to form components.
func recordFieldProps(record *Record, args []Object, env *Environment) Object {
	if len(args) < 1 || len(args) > 2 {
		return newArityErrorRange("fieldProps", len(args), 1, 2)
	}

	fieldName, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "Record.fieldProps", "string", args[0].Type())
	}

	result := make(map[string]ast.Expression)

	// name - the field name for HTML name attribute
	result["name"] = &ast.StringLiteral{Value: fieldName.Value}

	// Default label: titlecased field name
	label := toTitleCase(fieldName.Value)

	// Default type
	inputType := "text"
	var inputMode, autocomplete string

	// Get schema field if available
	var field *DSLSchemaField
	if record.Schema != nil {
		if f, exists := record.Schema.Fields[fieldName.Value]; exists {
			field = f

			// Label from schema title
			label = getFieldTitle(fieldName.Value, field)

			// Type mapping from schema type
			inputType, inputMode, autocomplete = inputTypeForSchemaType(field.Type)

			// Store original type
			result["type"] = &ast.StringLiteral{Value: inputType}

			// Required
			if field.Required {
				result["required"] = &ast.Boolean{Value: true}
			}

			// Enum options
			if len(field.EnumValues) > 0 {
				result["type"] = &ast.StringLiteral{Value: "select"}
				opts := make([]ast.Expression, len(field.EnumValues))
				for i, v := range field.EnumValues {
					opts[i] = &ast.StringLiteral{Value: v}
				}
				result["options"] = &ast.ArrayLiteral{Elements: opts}
			}

			// Placeholder from metadata
			if field.Metadata != nil {
				if ph, ok := field.Metadata["placeholder"]; ok {
					if phStr, ok := ph.(*String); ok {
						result["placeholder"] = &ast.StringLiteral{Value: phStr.Value}
					}
				}
			}
		}
	}

	result["label"] = &ast.StringLiteral{Value: label}

	if inputType != "" && result["type"] == nil {
		result["type"] = &ast.StringLiteral{Value: inputType}
	}
	if inputMode != "" {
		result["inputmode"] = &ast.StringLiteral{Value: inputMode}
	}
	if autocomplete != "" {
		result["autocomplete"] = &ast.StringLiteral{Value: autocomplete}
	}

	// Value - get current value from record, formatted for input
	if valExpr, exists := record.Data[fieldName.Value]; exists {
		val := Eval(valExpr, env)
		if val != nil && val != NULL {
			formattedVal := formatValueForInput(val, field)
			if formattedVal != nil {
				result["value"] = formattedVal
			}
		}
	}

	// Error - get error message if present
	if record.Errors != nil {
		if errObj, exists := record.Errors[fieldName.Value]; exists {
			if errObj.Message != "" {
				result["error"] = &ast.StringLiteral{Value: errObj.Message}
			}
		}
	}

	// Merge user overrides (second argument wins)
	if len(args) == 2 {
		if overrides, ok := args[1].(*Dictionary); ok {
			for key, valExpr := range overrides.Pairs {
				val := Eval(valExpr, env)
				if val != nil && val != NULL {
					switch v := val.(type) {
					case *String:
						result[key] = &ast.StringLiteral{Value: v.Value}
					case *Boolean:
						result[key] = &ast.Boolean{Value: v.Value}
					case *Integer:
						result[key] = &ast.IntegerLiteral{Value: v.Value}
					default:
						result[key] = valExpr
					}
				}
			}
		}
	}

	return &Dictionary{Pairs: result, Env: env}
}

// inputTypeForSchemaType returns HTML input type, inputmode, and autocomplete for a schema type.
func inputTypeForSchemaType(schemaType string) (inputType, inputMode, autocomplete string) {
	switch schemaType {
	case "email":
		return "email", "email", "email"
	case "url":
		return "url", "url", "url"
	case "phone":
		return "tel", "tel", "tel"
	case "integer", "int":
		return "number", "numeric", ""
	case "float", "number":
		return "text", "decimal", ""
	case "boolean", "bool":
		return "checkbox", "", ""
	case "money":
		return "text", "decimal", ""
	case "date":
		return "date", "", ""
	case "datetime":
		return "datetime-local", "", ""
	case "unit":
		return "text", "decimal", ""
	case "password":
		return "password", "", "current-password"
	default:
		return "text", "", ""
	}
}

// formatValueForInput formats a value for use in an HTML input element.
// Returns an AST expression suitable for inclusion in a Dictionary.
func formatValueForInput(val Object, field *DSLSchemaField) ast.Expression {
	switch v := val.(type) {
	case *String:
		return &ast.StringLiteral{Value: v.Value}
	case *Integer:
		return &ast.IntegerLiteral{Value: v.Value}
	case *Float:
		return &ast.FloatLiteral{Value: v.Value}
	case *Boolean:
		return &ast.Boolean{Value: v.Value}
	case *Money:
		// Format as decimal string for input (e.g., "49.99" not 4999)
		decimalValue := float64(v.Amount) / float64(pow10(int(v.Scale)))
		return &ast.StringLiteral{Value: fmt.Sprintf("%.*f", v.Scale, decimalValue)}
	case *Dictionary:
		// Handle datetime - format as ISO for input
		if isDatetimeDict(v) {
			isoStr := datetimeDictToString(v)
			// For datetime-local input, remove the Z suffix
			if len(isoStr) > 0 && isoStr[len(isoStr)-1] == 'Z' {
				isoStr = isoStr[:len(isoStr)-1]
			}
			return &ast.StringLiteral{Value: isoStr}
		}
		return nil
	case *Unit:
		// Just the numeric value for input
		decimalValue := float64(v.Amount) / float64(pow10(v.Scale))
		return &ast.StringLiteral{Value: fmt.Sprintf("%g", decimalValue)}
	default:
		return nil
	}
}

// pow10 returns 10^n for small non-negative n
func pow10(n int) int64 {
	result := int64(1)
	for range n {
		result *= 10
	}
	return result
}

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
