package evaluator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/lexer"
)

// Regex patterns for DSL schema validation (same as @std/schema)
var (
	dslEmailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	dslURLRegex   = regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	dslPhoneRegex = regexp.MustCompile(`^[\d\s\+\-\(\)\.]+$`)
	dslSlugRegex  = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
)

// DSLSchema represents a schema declared with @schema
type DSLSchema struct {
	Name      string
	Fields    map[string]*DSLSchemaField
	Relations map[string]*DSLSchemaRelation
}

// DSLSchemaField represents a field in a DSL schema
type DSLSchemaField struct {
	Name           string
	Type           string // original type: "email", "url", "int", etc.
	Required       bool
	Nullable       bool     // true if type ends with ?
	DefaultValue   Object   // parsed default value, or nil
	DefaultExpr    string   // original expression (for SQL generation)
	ValidationType string   // "email", "url", "phone", "slug", "enum", or "" for no validation
	EnumValues     []string // for enum types: allowed values
	MinLength      *int     // for string length validation
	MaxLength      *int     // for string length validation
	MinValue       *int64   // for integer range validation
	MaxValue       *int64   // for integer range validation
	Unique         bool     // whether field has UNIQUE constraint
}

// DSLSchemaRelation represents a relation in a DSL schema
type DSLSchemaRelation struct {
	FieldName    string
	TargetSchema string
	ForeignKey   string
	IsMany       bool // true for has-many, false for belongs-to/has-one
}

// Type returns the object type
func (s *DSLSchema) Type() ObjectType { return "DSL_SCHEMA" }

// Inspect returns a string representation
func (s *DSLSchema) Inspect() string {
	var fields []string
	for name, f := range s.Fields {
		fields = append(fields, fmt.Sprintf("%s: %s", name, f.Type))
	}
	for name, r := range s.Relations {
		typeName := r.TargetSchema
		if r.IsMany {
			typeName = "[" + typeName + "]"
		}
		fields = append(fields, fmt.Sprintf("%s: %s via %s", name, typeName, r.ForeignKey))
	}
	return fmt.Sprintf("@schema %s { %s }", s.Name, strings.Join(fields, ", "))
}

// dslSchemaMethods lists available methods on DSL schema objects
var dslSchemaMethods = []string{}

// evalDSLSchemaMethod dispatches method calls on DSL schema objects
func evalDSLSchemaMethod(schema *DSLSchema, method string, args []Object, env *Environment) Object {
	switch method {
	default:
		return &Error{Message: fmt.Sprintf("unknown method '%s' for DSL_SCHEMA", method)}
	}
}

// evalDSLSchemaProperty evaluates property access on a DSLSchema
func evalDSLSchemaProperty(schema *DSLSchema, key string) Object {
	switch key {
	case "Name", "name":
		return &String{Value: schema.Name}
	case "Fields", "fields":
		// Return a dictionary of field definitions
		pairs := make(map[string]ast.Expression)
		for name, field := range schema.Fields {
			// Create a dict for each field with name, type, required, nullable, default
			fieldPairs := make(map[string]ast.Expression)
			fieldPairs["name"] = &ast.StringLiteral{Value: field.Name}
			fieldPairs["type"] = &ast.StringLiteral{Value: field.Type}
			fieldPairs["required"] = &ast.Boolean{Value: field.Required}
			fieldPairs["nullable"] = &ast.Boolean{Value: field.Nullable}
			if field.DefaultExpr != "" {
				fieldPairs["default"] = &ast.StringLiteral{Value: field.DefaultExpr}
			}
			pairs[name] = &ast.DictionaryLiteral{Pairs: fieldPairs}
		}
		return &Dictionary{Pairs: pairs}
	case "Relations", "relations":
		// Return a dictionary of relation definitions
		pairs := make(map[string]ast.Expression)
		for name, rel := range schema.Relations {
			relPairs := make(map[string]ast.Expression)
			relPairs["field"] = &ast.StringLiteral{Value: rel.FieldName}
			relPairs["target"] = &ast.StringLiteral{Value: rel.TargetSchema}
			relPairs["foreign_key"] = &ast.StringLiteral{Value: rel.ForeignKey}
			relPairs["is_many"] = &ast.Boolean{Value: rel.IsMany}
			pairs[name] = &ast.DictionaryLiteral{Pairs: relPairs}
		}
		return &Dictionary{Pairs: pairs}
	default:
		// Check if it's a field name
		if field, ok := schema.Fields[key]; ok {
			return &String{Value: field.Type}
		}
		// Check if it's a relation name
		if rel, ok := schema.Relations[key]; ok {
			return &String{Value: rel.TargetSchema}
		}
		return NULL
	}
}

// evalSchemaDeclaration evaluates a @schema declaration
func evalSchemaDeclaration(node *ast.SchemaDeclaration, env *Environment) Object {
	schema := &DSLSchema{
		Name:      node.Name.Value,
		Fields:    make(map[string]*DSLSchemaField),
		Relations: make(map[string]*DSLSchemaRelation),
	}

	// Process fields
	for _, field := range node.Fields {
		if field.ForeignKey != "" {
			// This is a relation
			schema.Relations[field.Name.Value] = &DSLSchemaRelation{
				FieldName:    field.Name.Value,
				TargetSchema: field.TypeName,
				ForeignKey:   field.ForeignKey,
				IsMany:       field.IsArray,
			}
		} else {
			// This is a regular field
			dslField := &DSLSchemaField{
				Name:           field.Name.Value,
				Type:           field.TypeName,
				Required:       !field.Nullable, // Required unless nullable
				Nullable:       field.Nullable,
				ValidationType: getValidationType(field.TypeName),
				EnumValues:     field.EnumValues,
			}

			// Evaluate default value if present
			if field.DefaultValue != nil {
				dslField.DefaultValue = Eval(field.DefaultValue, env)
				dslField.DefaultExpr = field.DefaultValue.String()
			}

			// Process type options (min, max, unique, etc.)
			if field.TypeOptions != nil {
				for key, valExpr := range field.TypeOptions {
					val := Eval(valExpr, env)
					switch key {
					case "min":
						if intVal, ok := val.(*Integer); ok {
							// For string types, min means minLength
							if isStringType(field.TypeName) {
								minLen := int(intVal.Value)
								dslField.MinLength = &minLen
							} else {
								minVal := intVal.Value
								dslField.MinValue = &minVal
							}
						}
					case "max":
						if intVal, ok := val.(*Integer); ok {
							// For string types, max means maxLength
							if isStringType(field.TypeName) {
								maxLen := int(intVal.Value)
								dslField.MaxLength = &maxLen
							} else {
								maxVal := intVal.Value
								dslField.MaxValue = &maxVal
							}
						}
					case "unique":
						if boolVal, ok := val.(*Boolean); ok {
							dslField.Unique = boolVal.Value
						}
					}
				}
			}

			schema.Fields[field.Name.Value] = dslField
		}
	}

	// Register schema in environment
	env.Set(node.Name.Value, schema)

	// Declarations return NULL (excluded from block concatenation)
	return NULL
}

// getValidationType returns the validation type for a schema field type
func getValidationType(typeName string) string {
	switch strings.ToLower(typeName) {
	case "email":
		return "email"
	case "url":
		return "url"
	case "phone":
		return "phone"
	case "slug":
		return "slug"
	case "enum":
		return "enum"
	default:
		return ""
	}
}

// isStringType returns true if the type stores as TEXT and can have length constraints
func isStringType(typeName string) bool {
	switch strings.ToLower(typeName) {
	case "string", "text", "email", "url", "phone", "slug":
		return true
	default:
		return false
	}
}

// Known primitive types for schema fields
var knownPrimitiveTypes = map[string]bool{
	"int":      true,
	"integer":  true,
	"bigint":   true,
	"string":   true,
	"bool":     true,
	"boolean":  true,
	"float":    true,
	"number":   true,
	"datetime": true,
	"date":     true,
	"time":     true,
	"money":    true,
	"uuid":     true,
	"ulid":     true,
	"text":     true,
	"json":     true,
	"email":    true,
	"url":      true,
	"phone":    true,
	"slug":     true,
	"enum":     true,
}

// isPrimitiveType checks if a type name is a known primitive type
func isPrimitiveType(typeName string) bool {
	return knownPrimitiveTypes[strings.ToLower(typeName)]
}

// buildCreateTableSQL generates CREATE TABLE IF NOT EXISTS SQL from a schema
func buildCreateTableSQL(schema *DSLSchema, tableName string, driver string) string {
	var columns []string

	// Map schema types to SQL types based on driver
	for name, field := range schema.Fields {
		sqlType := schemaTypeToSQL(field.Type, driver)
		var colParts []string
		colParts = append(colParts, name, sqlType)

		// id fields get special treatment
		if name == "id" {
			if driver == "sqlite" {
				columns = append(columns, "id INTEGER PRIMARY KEY")
				continue
			} else if driver == "postgres" {
				columns = append(columns, "id SERIAL PRIMARY KEY")
				continue
			} else if driver == "mysql" {
				columns = append(columns, "id INT AUTO_INCREMENT PRIMARY KEY")
				continue
			}
		}

		// Add UNIQUE constraint if specified
		if field.Unique {
			colParts = append(colParts, "UNIQUE")
		}

		// Add NOT NULL for required (non-nullable) fields
		if !field.Nullable {
			colParts = append(colParts, "NOT NULL")
		}

		// Add DEFAULT clause if present
		if field.DefaultExpr != "" {
			defaultSQL := objectToSQLDefault(field.DefaultValue)
			if defaultSQL != "" {
				colParts = append(colParts, "DEFAULT", defaultSQL)
			}
		}

		// Build CHECK constraints
		var checks []string

		// Enum CHECK constraint
		if len(field.EnumValues) > 0 {
			quoted := make([]string, len(field.EnumValues))
			for i, v := range field.EnumValues {
				// Escape single quotes in enum values
				quoted[i] = fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
			}
			checks = append(checks, fmt.Sprintf("%s IN (%s)", name, strings.Join(quoted, ", ")))
		}

		// Integer range CHECK constraint
		if field.MinValue != nil || field.MaxValue != nil {
			if field.MinValue != nil && field.MaxValue != nil {
				checks = append(checks, fmt.Sprintf("%s >= %d AND %s <= %d", name, *field.MinValue, name, *field.MaxValue))
			} else if field.MinValue != nil {
				checks = append(checks, fmt.Sprintf("%s >= %d", name, *field.MinValue))
			} else if field.MaxValue != nil {
				checks = append(checks, fmt.Sprintf("%s <= %d", name, *field.MaxValue))
			}
		}

		// String length CHECK constraint
		if field.MinLength != nil || field.MaxLength != nil {
			if field.MinLength != nil && field.MaxLength != nil {
				checks = append(checks, fmt.Sprintf("length(%s) >= %d AND length(%s) <= %d", name, *field.MinLength, name, *field.MaxLength))
			} else if field.MinLength != nil {
				checks = append(checks, fmt.Sprintf("length(%s) >= %d", name, *field.MinLength))
			} else if field.MaxLength != nil {
				checks = append(checks, fmt.Sprintf("length(%s) <= %d", name, *field.MaxLength))
			}
		}

		// Add CHECK constraints
		if len(checks) > 0 {
			colParts = append(colParts, fmt.Sprintf("CHECK(%s)", strings.Join(checks, " AND ")))
		}

		columns = append(columns, strings.Join(colParts, " "))
	}

	// Add foreign key columns for belongs-to relations (not has-many)
	for _, rel := range schema.Relations {
		if !rel.IsMany {
			// This is a belongs-to relation - the foreign key should be in this table
			// But we only add it if it's not already a field
			if _, exists := schema.Fields[rel.ForeignKey]; !exists {
				columns = append(columns, fmt.Sprintf("%s INTEGER", rel.ForeignKey))
			}
		}
	}

	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", tableName, strings.Join(columns, ", "))
}

// schemaTypeToSQL converts a schema field type to SQL type
func schemaTypeToSQL(schemaType string, driver string) string {
	switch strings.ToLower(schemaType) {
	case "int", "integer":
		return "INTEGER"
	case "bigint":
		if driver == "postgres" {
			return "BIGINT"
		}
		return "INTEGER" // SQLite integers are already 64-bit
	case "string":
		return "TEXT"
	case "text":
		return "TEXT"
	case "bool", "boolean":
		if driver == "sqlite" {
			return "INTEGER" // SQLite uses 0/1 for bools
		}
		return "BOOLEAN"
	case "float", "number":
		return "REAL"
	case "datetime":
		if driver == "sqlite" {
			return "TEXT" // SQLite stores datetimes as TEXT
		}
		return "TIMESTAMP"
	case "date":
		if driver == "sqlite" {
			return "TEXT"
		}
		return "DATE"
	case "time":
		if driver == "sqlite" {
			return "TEXT"
		}
		return "TIME"
	case "money":
		return "INTEGER" // Store as cents/smallest unit
	case "uuid", "ulid":
		return "TEXT"
	case "json":
		if driver == "postgres" {
			return "JSONB"
		}
		return "TEXT"
	// Validated string types - store as TEXT, validate in Parsley
	case "email", "url", "phone", "slug", "enum":
		return "TEXT"
	default:
		return "TEXT"
	}
}

// objectToSQLDefault converts a Parsley Object to a SQL DEFAULT value string
func objectToSQLDefault(obj Object) string {
	if obj == nil || obj == NULL {
		return "NULL"
	}
	switch v := obj.(type) {
	case *String:
		// Escape single quotes for SQL
		escaped := strings.ReplaceAll(v.Value, "'", "''")
		return fmt.Sprintf("'%s'", escaped)
	case *Integer:
		return fmt.Sprintf("%d", v.Value)
	case *Float:
		return fmt.Sprintf("%g", v.Value)
	case *Boolean:
		if v.Value {
			return "1" // SQLite-compatible
		}
		return "0"
	default:
		// Complex types can't be SQL defaults easily
		return ""
	}
}

// ============================================================================
// Schema Field Validation
// ============================================================================

// ValidationFieldError represents a single field validation error
type ValidationFieldError struct {
	Field   string
	Code    string
	Message string
}

// ValidateSchemaField validates a value against a schema field's constraints
// Returns nil if valid, or a ValidationFieldError if invalid
func ValidateSchemaField(fieldName string, value Object, field *DSLSchemaField) *ValidationFieldError {
	// Skip validation for NULL values (unless required is enforced elsewhere)
	if value == nil || value == NULL {
		return nil
	}

	// Validate based on validation type
	switch field.ValidationType {
	case "email":
		str, ok := value.(*String)
		if !ok {
			return &ValidationFieldError{
				Field:   fieldName,
				Code:    "TYPE",
				Message: "Expected string value for email field",
			}
		}
		if !dslEmailRegex.MatchString(str.Value) {
			return &ValidationFieldError{
				Field:   fieldName,
				Code:    "FORMAT",
				Message: "Invalid email format",
			}
		}

	case "url":
		str, ok := value.(*String)
		if !ok {
			return &ValidationFieldError{
				Field:   fieldName,
				Code:    "TYPE",
				Message: "Expected string value for url field",
			}
		}
		if !dslURLRegex.MatchString(str.Value) {
			return &ValidationFieldError{
				Field:   fieldName,
				Code:    "FORMAT",
				Message: "Invalid URL format (must start with http:// or https://)",
			}
		}

	case "phone":
		str, ok := value.(*String)
		if !ok {
			return &ValidationFieldError{
				Field:   fieldName,
				Code:    "TYPE",
				Message: "Expected string value for phone field",
			}
		}
		if !dslPhoneRegex.MatchString(str.Value) {
			return &ValidationFieldError{
				Field:   fieldName,
				Code:    "FORMAT",
				Message: "Invalid phone number format",
			}
		}

	case "slug":
		str, ok := value.(*String)
		if !ok {
			return &ValidationFieldError{
				Field:   fieldName,
				Code:    "TYPE",
				Message: "Expected string value for slug field",
			}
		}
		if !dslSlugRegex.MatchString(str.Value) {
			return &ValidationFieldError{
				Field:   fieldName,
				Code:    "FORMAT",
				Message: "Invalid slug format (must be lowercase alphanumeric with hyphens)",
			}
		}

	case "enum":
		str, ok := value.(*String)
		if !ok {
			return &ValidationFieldError{
				Field:   fieldName,
				Code:    "TYPE",
				Message: "Expected string value for enum field",
			}
		}
		if len(field.EnumValues) > 0 {
			found := false
			for _, v := range field.EnumValues {
				if v == str.Value {
					found = true
					break
				}
			}
			if !found {
				return &ValidationFieldError{
					Field:   fieldName,
					Code:    "ENUM",
					Message: fmt.Sprintf("Value must be one of: %s", strings.Join(field.EnumValues, ", ")),
				}
			}
		}
	}

	// Validate string length constraints
	if field.MinLength != nil || field.MaxLength != nil {
		str, ok := value.(*String)
		if ok {
			length := len(str.Value)
			if field.MinLength != nil && length < *field.MinLength {
				return &ValidationFieldError{
					Field:   fieldName,
					Code:    "MIN_LENGTH",
					Message: fmt.Sprintf("Must be at least %d characters", *field.MinLength),
				}
			}
			if field.MaxLength != nil && length > *field.MaxLength {
				return &ValidationFieldError{
					Field:   fieldName,
					Code:    "MAX_LENGTH",
					Message: fmt.Sprintf("Must be at most %d characters", *field.MaxLength),
				}
			}
		}
	}

	// Validate integer range constraints
	if field.MinValue != nil || field.MaxValue != nil {
		intVal, ok := value.(*Integer)
		if ok {
			if field.MinValue != nil && intVal.Value < *field.MinValue {
				return &ValidationFieldError{
					Field:   fieldName,
					Code:    "MIN_VALUE",
					Message: fmt.Sprintf("Must be at least %d", *field.MinValue),
				}
			}
			if field.MaxValue != nil && intVal.Value > *field.MaxValue {
				return &ValidationFieldError{
					Field:   fieldName,
					Code:    "MAX_VALUE",
					Message: fmt.Sprintf("Must be at most %d", *field.MaxValue),
				}
			}
		}
	}

	return nil
}

// ValidateSchemaFields validates multiple field values against a schema
// Returns a validation error object or nil if all valid
func ValidateSchemaFields(values map[string]Object, schema *DSLSchema) Object {
	var fieldErrors []ValidationFieldError

	for fieldName, value := range values {
		if field, ok := schema.Fields[fieldName]; ok {
			if err := ValidateSchemaField(fieldName, value, field); err != nil {
				fieldErrors = append(fieldErrors, *err)
			}
		}
	}

	if len(fieldErrors) > 0 {
		return buildValidationErrorObject(fieldErrors)
	}
	return nil
}

// buildValidationErrorObject creates a Dictionary representing validation errors
func buildValidationErrorObject(errors []ValidationFieldError) *Dictionary {
	// Build the main error object
	pairs := make(map[string]ast.Expression)
	pairs["error"] = makeStringLiteral("VALIDATION_ERROR")
	pairs["message"] = makeStringLiteral("Validation failed")
	pairs["fields"] = &ast.ArrayLiteral{Elements: makeFieldErrorElements(errors)}

	return &Dictionary{Pairs: pairs, Env: NewEnvironment()}
}

// makeFieldErrorElements creates an array of field error dictionaries
func makeFieldErrorElements(errors []ValidationFieldError) []ast.Expression {
	elements := make([]ast.Expression, len(errors))
	for i, err := range errors {
		pairs := make(map[string]ast.Expression)
		pairs["field"] = makeStringLiteral(err.Field)
		pairs["code"] = makeStringLiteral(err.Code)
		pairs["message"] = makeStringLiteral(err.Message)
		elements[i] = &ast.DictionaryLiteral{Pairs: pairs}
	}
	return elements
}

// makeStringLiteral creates a StringLiteral with proper Token.Literal for inspection
func makeStringLiteral(value string) *ast.StringLiteral {
	return &ast.StringLiteral{
		Value: value,
		Token: lexer.Token{
			Type:    lexer.STRING,
			Literal: value,
		},
	}
}
