package evaluator

import (
	"regexp"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
)

// Regex patterns for schema validation
var (
	schemaEmailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	schemaURLRegex   = regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	schemaPhoneRegex = regexp.MustCompile(`^[\d\s\+\-\(\)\.]+$`)
	schemaDateRegex  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	schemaULIDRegex  = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)
	schemaUUIDRegex  = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
)

// schemaMethods lists the available methods on schema objects
var schemaMethods = []string{"validate"}

// IsSchemaDict checks if a dictionary is a schema (created by schema.define)
func IsSchemaDict(dict *Dictionary) bool {
	if dict == nil {
		return false
	}
	if schemaMarker, ok := dict.Pairs["__schema__"]; ok {
		markerObj := Eval(schemaMarker, dict.Env)
		if b, ok := markerObj.(*Boolean); ok && b.Value {
			return true
		}
	}
	return false
}

// evalSchemaMethod dispatches method calls on schema dictionaries.
// Returns nil for unknown methods to allow fallthrough to dictionary methods.
func evalSchemaMethod(schema *Dictionary, method string, args []Object, env *Environment) Object {
	switch method {
	case "validate":
		if len(args) != 1 {
			return newArityError("validate", len(args), 1)
		}
		// Reuse the existing schemaValidate logic
		return schemaValidate(schema, args[0])
	default:
		// Return nil to allow fallthrough to regular dictionary methods
		return nil
	}
}

// loadSchemaModule returns the schema module as a StdlibModuleDict
func loadSchemaModule(env *Environment) Object {
	return &StdlibModuleDict{
		Exports: map[string]Object{
			// Type factories
			"string":   &Builtin{Fn: schemaString},
			"email":    &Builtin{Fn: schemaEmail},
			"url":      &Builtin{Fn: schemaURL},
			"phone":    &Builtin{Fn: schemaPhone},
			"integer":  &Builtin{Fn: schemaInteger},
			"number":   &Builtin{Fn: schemaNumber},
			"boolean":  &Builtin{Fn: schemaBoolean},
			"enum":     &Builtin{Fn: schemaEnum},
			"date":     &Builtin{Fn: schemaDate},
			"datetime": &Builtin{Fn: schemaDatetime},
			"money":    &Builtin{Fn: schemaMoney},
			"array":    &Builtin{Fn: schemaArray},
			"object":   &Builtin{Fn: schemaObject},
			"id":       &Builtin{Fn: schemaID},

			// Schema operations
			"define": &Builtin{Fn: schemaDefine},
			"table":  &Builtin{Fn: schemaTable},
		},
	}
}

// =============================================================================
// Schema Type Factories
// =============================================================================

// schemaString creates a string type spec
func schemaString(args ...Object) Object {
	opts := extractSchemaOptions(args)
	return createTypeSpec("string", opts)
}

// schemaEmail creates an email type spec
func schemaEmail(args ...Object) Object {
	opts := extractSchemaOptions(args)
	return createTypeSpec("email", opts)
}

// schemaURL creates a URL type spec
func schemaURL(args ...Object) Object {
	opts := extractSchemaOptions(args)
	return createTypeSpec("url", opts)
}

// schemaPhone creates a phone type spec
func schemaPhone(args ...Object) Object {
	opts := extractSchemaOptions(args)
	return createTypeSpec("phone", opts)
}

// schemaInteger creates an integer type spec
func schemaInteger(args ...Object) Object {
	opts := extractSchemaOptions(args)
	return createTypeSpec("integer", opts)
}

// schemaNumber creates a number type spec
func schemaNumber(args ...Object) Object {
	opts := extractSchemaOptions(args)
	return createTypeSpec("number", opts)
}

// schemaBoolean creates a boolean type spec
func schemaBoolean(args ...Object) Object {
	opts := extractSchemaOptions(args)
	return createTypeSpec("boolean", opts)
}

// schemaEnum creates an enum type spec with allowed values
func schemaEnum(args ...Object) Object {
	if len(args) == 0 {
		return &Error{Message: "schema.enum requires at least one value"}
	}

	// First arg can be options dict or the values themselves
	var opts map[string]Object
	var values []Object

	if dict, ok := args[0].(*Dictionary); ok && len(args) == 1 {
		// Single dict argument with options
		opts = extractSchemaOptionsFromDict(dict)
	} else {
		// Arguments are the enum values
		opts = make(map[string]Object)
		values = args
	}

	// Store values in opts
	if len(values) > 0 {
		opts["values"] = &Array{Elements: values}
	}

	return createTypeSpec("enum", opts)
}

// schemaDate creates a date type spec
func schemaDate(args ...Object) Object {
	opts := extractSchemaOptions(args)
	return createTypeSpec("date", opts)
}

// schemaDatetime creates a datetime type spec
func schemaDatetime(args ...Object) Object {
	opts := extractSchemaOptions(args)
	return createTypeSpec("datetime", opts)
}

// schemaMoney creates a money type spec
func schemaMoney(args ...Object) Object {
	opts := extractSchemaOptions(args)
	return createTypeSpec("money", opts)
}

// schemaArray creates an array type spec
func schemaArray(args ...Object) Object {
	opts := extractSchemaOptions(args)
	return createTypeSpec("array", opts)
}

// schemaObject creates an object type spec
func schemaObject(args ...Object) Object {
	opts := extractSchemaOptions(args)
	return createTypeSpec("object", opts)
}

// schemaID creates an ID type spec
func schemaID(args ...Object) Object {
	opts := extractSchemaOptions(args)
	// Default format is ULID
	if _, ok := opts["format"]; !ok {
		opts["format"] = &String{Value: "ulid"}
	}
	return createTypeSpec("id", opts)
}

// =============================================================================
// Helper Functions
// =============================================================================

// extractSchemaOptions extracts options from arguments
func extractSchemaOptions(args []Object) map[string]Object {
	opts := make(map[string]Object)

	if len(args) > 0 {
		if dict, ok := args[0].(*Dictionary); ok {
			return extractSchemaOptionsFromDict(dict)
		}
	}

	return opts
}

// extractSchemaOptionsFromDict extracts options from a dictionary
func extractSchemaOptionsFromDict(dict *Dictionary) map[string]Object {
	opts := make(map[string]Object)

	for key, expr := range dict.Pairs {
		val := Eval(expr, dict.Env)
		if !isError(val) {
			opts[key] = val
		}
	}

	return opts
}

// createTypeSpec creates a type specification dictionary
func createTypeSpec(typeName string, opts map[string]Object) Object {
	pairs := make(map[string]ast.Expression)
	pairs["__type__"] = objectToExpression(&String{Value: typeName})
	pairs["type"] = objectToExpression(&String{Value: typeName})

	// Copy all options
	for k, v := range opts {
		pairs[k] = objectToExpression(v)
	}

	return &Dictionary{Pairs: pairs}
}

// =============================================================================
// Schema Operations
// =============================================================================

// schemaDefine creates a schema definition from name and fields
func schemaDefine(args ...Object) Object {
	if len(args) != 2 {
		return newArityError("schema.define", len(args), 2)
	}

	name, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "schema.define", "string", args[0].Type())
	}

	fields, ok := args[1].(*Dictionary)
	if !ok {
		return newTypeError("TYPE-0001", "schema.define", "dictionary", args[1].Type())
	}

	// Create schema object
	pairs := make(map[string]ast.Expression)
	pairs["__schema__"] = objectToExpression(TRUE)
	pairs["name"] = objectToExpression(&String{Value: name.Value})
	pairs["fields"] = objectToExpression(fields)

	return &Dictionary{Pairs: pairs}
}

// schemaValidate validates data against a schema
func schemaValidate(args ...Object) Object {
	if len(args) != 2 {
		return newArityError("schema.validate", len(args), 2)
	}

	schema, ok := args[0].(*Dictionary)
	if !ok {
		return newTypeError("TYPE-0001", "schema.validate", "dictionary (schema)", args[0].Type())
	}

	data, ok := args[1].(*Dictionary)
	if !ok {
		return newTypeError("TYPE-0001", "schema.validate", "dictionary (data)", args[1].Type())
	}

	// Get fields from schema
	fieldsExpr, ok := schema.Pairs["fields"]
	if !ok {
		return &Error{Message: "Schema has no fields defined"}
	}

	fieldsObj := Eval(fieldsExpr, schema.Env)
	fields, ok := fieldsObj.(*Dictionary)
	if !ok {
		return &Error{Message: "Schema fields must be a dictionary"}
	}

	// Validate each field
	var errors []Object

	for fieldName, specExpr := range fields.Pairs {
		specObj := Eval(specExpr, fields.Env)
		spec, ok := specObj.(*Dictionary)
		if !ok {
			continue
		}

		// Get value from data
		valueExpr, hasValue := data.Pairs[fieldName]
		var value Object = NULL
		if hasValue {
			value = Eval(valueExpr, data.Env)
		}

		// Check required
		if reqExpr, ok := spec.Pairs["required"]; ok {
			reqObj := Eval(reqExpr, spec.Env)
			if req, ok := reqObj.(*Boolean); ok && req.Value {
				if value == NULL || (value.Type() == STRING_OBJ && value.(*String).Value == "") {
					errors = append(errors, createValidationError(fieldName, "REQUIRED", "Field is required"))
					continue
				}
			}
		}

		// Skip further validation if value is null and not required
		if value == NULL {
			continue
		}

		// Get type and validate
		typeExpr, ok := spec.Pairs["type"]
		if !ok {
			continue
		}
		typeObj := Eval(typeExpr, spec.Env)
		typeStr, ok := typeObj.(*String)
		if !ok {
			continue
		}

		// Type-specific validation
		fieldErrors := validateFieldType(fieldName, typeStr.Value, value, spec)
		errors = append(errors, fieldErrors...)
	}

	// Build result
	resultPairs := make(map[string]ast.Expression)
	resultPairs["valid"] = objectToExpression(&Boolean{Value: len(errors) == 0})

	if len(errors) > 0 {
		resultPairs["errors"] = objectToExpression(&Array{Elements: errors})
	} else {
		resultPairs["errors"] = objectToExpression(&Array{Elements: []Object{}})
	}

	return &Dictionary{Pairs: resultPairs}
}

// schemaTable binds a schema to a database table and returns a TableBinding helper.
func schemaTable(args ...Object) Object {
	if len(args) != 3 {
		return newArityError("schema.table", len(args), 3)
	}

	schemaDict, ok := args[0].(*Dictionary)
	if !ok {
		return newTypeError("TYPE-0001", "schema.table", "dictionary (schema)", args[0].Type())
	}

	db, ok := args[1].(*DBConnection)
	if !ok {
		return newTypeError("TYPE-0001", "schema.table", "database connection", args[1].Type())
	}

	tableName, ok := args[2].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "schema.table", "string (table name)", args[2].Type())
	}

	name := strings.TrimSpace(tableName.Value)
	if name == "" || !identifierRegex.MatchString(name) {
		return newValidationError("VAL-0003", map[string]any{"Pattern": "identifier", "GoError": "invalid table name"})
	}

	return &TableBinding{DB: db, Schema: schemaDict, TableName: name}
}

// validateFieldType validates a value against a type specification
func validateFieldType(fieldName, typeName string, value Object, spec *Dictionary) []Object {
	var errors []Object

	switch typeName {
	case "string":
		if _, ok := value.(*String); !ok {
			errors = append(errors, createValidationError(fieldName, "TYPE", "Expected string"))
		} else {
			errors = append(errors, validateStringConstraints(fieldName, value.(*String), spec)...)
		}

	case "email":
		if str, ok := value.(*String); !ok {
			errors = append(errors, createValidationError(fieldName, "TYPE", "Expected string"))
		} else if !schemaEmailRegex.MatchString(str.Value) {
			errors = append(errors, createValidationError(fieldName, "FORMAT", "Invalid email format"))
		}

	case "url":
		if str, ok := value.(*String); !ok {
			errors = append(errors, createValidationError(fieldName, "TYPE", "Expected string"))
		} else if !schemaURLRegex.MatchString(str.Value) {
			errors = append(errors, createValidationError(fieldName, "FORMAT", "Invalid URL format"))
		}

	case "phone":
		if str, ok := value.(*String); !ok {
			errors = append(errors, createValidationError(fieldName, "TYPE", "Expected string"))
		} else if !schemaPhoneRegex.MatchString(str.Value) {
			errors = append(errors, createValidationError(fieldName, "FORMAT", "Invalid phone format"))
		}

	case "integer":
		if _, ok := value.(*Integer); !ok {
			errors = append(errors, createValidationError(fieldName, "TYPE", "Expected integer"))
		} else {
			errors = append(errors, validateIntegerConstraints(fieldName, value.(*Integer), spec)...)
		}

	case "number":
		switch value.(type) {
		case *Integer, *Float:
			// OK
		default:
			errors = append(errors, createValidationError(fieldName, "TYPE", "Expected number"))
		}

	case "boolean":
		if _, ok := value.(*Boolean); !ok {
			errors = append(errors, createValidationError(fieldName, "TYPE", "Expected boolean"))
		}

	case "enum":
		if valuesExpr, ok := spec.Pairs["values"]; ok {
			valuesObj := Eval(valuesExpr, spec.Env)
			if values, ok := valuesObj.(*Array); ok {
				found := false
				for _, v := range values.Elements {
					if objectsEqual(value, v) {
						found = true
						break
					}
				}
				if !found {
					errors = append(errors, createValidationError(fieldName, "ENUM", "Value not in allowed set"))
				}
			}
		}

	case "date":
		if str, ok := value.(*String); !ok {
			errors = append(errors, createValidationError(fieldName, "TYPE", "Expected date string"))
		} else if !schemaDateRegex.MatchString(str.Value) {
			errors = append(errors, createValidationError(fieldName, "FORMAT", "Invalid date format (expected YYYY-MM-DD)"))
		}

	case "id":
		if str, ok := value.(*String); ok {
			// Check format if specified
			if formatExpr, ok := spec.Pairs["format"]; ok {
				formatObj := Eval(formatExpr, spec.Env)
				if format, ok := formatObj.(*String); ok {
					switch format.Value {
					case "ulid":
						if !schemaULIDRegex.MatchString(str.Value) {
							errors = append(errors, createValidationError(fieldName, "FORMAT", "Invalid ULID format"))
						}
					case "uuid", "uuidv4", "uuidv7":
						if !schemaUUIDRegex.MatchString(str.Value) {
							errors = append(errors, createValidationError(fieldName, "FORMAT", "Invalid UUID format"))
						}
					}
				}
			}
		} else {
			errors = append(errors, createValidationError(fieldName, "TYPE", "Expected string"))
		}

	case "array":
		if _, ok := value.(*Array); !ok {
			errors = append(errors, createValidationError(fieldName, "TYPE", "Expected array"))
		}

	case "object":
		if _, ok := value.(*Dictionary); !ok {
			errors = append(errors, createValidationError(fieldName, "TYPE", "Expected object"))
		}
	}

	return errors
}

// validateStringConstraints validates string-specific constraints
func validateStringConstraints(fieldName string, value *String, spec *Dictionary) []Object {
	var errors []Object

	// Min length
	if minExpr, ok := spec.Pairs["min"]; ok {
		minObj := Eval(minExpr, spec.Env)
		if min, ok := minObj.(*Integer); ok {
			if int64(len(value.Value)) < min.Value {
				errors = append(errors, createValidationError(fieldName, "MIN_LENGTH", "String too short"))
			}
		}
	}

	// Max length
	if maxExpr, ok := spec.Pairs["max"]; ok {
		maxObj := Eval(maxExpr, spec.Env)
		if max, ok := maxObj.(*Integer); ok {
			if int64(len(value.Value)) > max.Value {
				errors = append(errors, createValidationError(fieldName, "MAX_LENGTH", "String too long"))
			}
		}
	}

	// Pattern
	if patternExpr, ok := spec.Pairs["pattern"]; ok {
		patternObj := Eval(patternExpr, spec.Env)
		if pattern, ok := patternObj.(*String); ok {
			re, err := regexp.Compile(pattern.Value)
			if err == nil && !re.MatchString(value.Value) {
				errors = append(errors, createValidationError(fieldName, "PATTERN", "Value does not match pattern"))
			}
		}
	}

	return errors
}

// validateIntegerConstraints validates integer-specific constraints
func validateIntegerConstraints(fieldName string, value *Integer, spec *Dictionary) []Object {
	var errors []Object

	// Min value
	if minExpr, ok := spec.Pairs["min"]; ok {
		minObj := Eval(minExpr, spec.Env)
		if min, ok := minObj.(*Integer); ok {
			if value.Value < min.Value {
				errors = append(errors, createValidationError(fieldName, "MIN_VALUE", "Value too small"))
			}
		}
	}

	// Max value
	if maxExpr, ok := spec.Pairs["max"]; ok {
		maxObj := Eval(maxExpr, spec.Env)
		if max, ok := maxObj.(*Integer); ok {
			if value.Value > max.Value {
				errors = append(errors, createValidationError(fieldName, "MAX_VALUE", "Value too large"))
			}
		}
	}

	return errors
}

// createValidationError creates a validation error object
func createValidationError(field, code, message string) Object {
	pairs := make(map[string]ast.Expression)
	pairs["field"] = objectToExpression(&String{Value: field})
	pairs["code"] = objectToExpression(&String{Value: code})
	pairs["message"] = objectToExpression(&String{Value: message})
	return &Dictionary{Pairs: pairs}
}
