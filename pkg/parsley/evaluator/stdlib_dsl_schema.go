package evaluator

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
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
	Name       string
	Fields     map[string]*DSLSchemaField
	FieldOrder []string // preserves declaration order
	Relations  map[string]*DSLSchemaRelation
}

// DSLSchemaField represents a field in a DSL schema
type DSLSchemaField struct {
	Name           string
	Type           string // original type: "email", "url", "int", etc.
	Required       bool
	Auto           bool              // true if field is auto-generated (e.g., id, createdAt)
	ReadOnly       bool              // true if field cannot be set from client/form input
	Nullable       bool              // true if type ends with ?
	DefaultValue   Object            // parsed default value, or nil
	DefaultExpr    string            // original expression (for SQL generation)
	ValidationType string            // "email", "url", "phone", "slug", "enum", or "" for no validation
	EnumValues     []string          // for enum types: allowed values
	MinLength      *int              // for string length validation
	MaxLength      *int              // for string length validation
	MinValue       *int64            // for integer range validation
	MaxValue       *int64            // for integer range validation
	Unique         bool              // whether field has UNIQUE constraint
	Primary        bool              // whether this field is the primary key
	Pattern        *regexp.Regexp    // compiled regex pattern for string validation
	PatternSource  string            // original pattern string (for HTML form attribute)
	Metadata       map[string]Object // metadata from pipe syntax: {title: "...", ...}
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

// PrimaryKey returns the name of the primary key field, or empty string if none.
func (s *DSLSchema) PrimaryKey() string {
	for name, field := range s.Fields {
		if field.Primary {
			return name
		}
	}
	return ""
}

// dslSchemaMethods lists available methods on DSL schema objects
var dslSchemaMethods = []string{
	"title", "placeholder", "meta", "fields", "visibleFields", "enumValues",
}

// evalDSLSchemaMethod dispatches method calls on DSL schema objects
func evalDSLSchemaMethod(schema *DSLSchema, method string, args []Object, env *Environment) Object {
	switch method {
	case "title":
		return schemaTitle(schema, args)
	case "placeholder":
		return schemaPlaceholder(schema, args)
	case "meta":
		return schemaMeta(schema, args)
	case "fields":
		return schemaFields(schema, args)
	case "visibleFields":
		return schemaVisibleFields(schema, args)
	case "enumValues":
		return schemaEnumValues(schema, args)
	default:
		return unknownMethodError(method, "Schema", dslSchemaMethods)
	}
}

// schemaTitle implements schema.title(field) → String
// Returns metadata.title or titlecase of field name
func schemaTitle(schema *DSLSchema, args []Object) Object {
	if len(args) != 1 {
		return newArityError("title", len(args), 1)
	}

	fieldName, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "Schema.title", "string", args[0].Type())
	}

	field, exists := schema.Fields[fieldName.Value]
	if !exists {
		return &String{Value: toTitleCase(fieldName.Value)}
	}

	// Check metadata for explicit title
	if field.Metadata != nil {
		if title, ok := field.Metadata["title"]; ok {
			if strTitle, ok := title.(*String); ok {
				return strTitle
			}
		}
	}

	// Fall back to titlecase of field name
	return &String{Value: toTitleCase(fieldName.Value)}
}

// schemaPlaceholder implements schema.placeholder(field) → String or null
func schemaPlaceholder(schema *DSLSchema, args []Object) Object {
	if len(args) != 1 {
		return newArityError("placeholder", len(args), 1)
	}

	fieldName, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "Schema.placeholder", "string", args[0].Type())
	}

	field, exists := schema.Fields[fieldName.Value]
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

// schemaMeta implements schema.meta(field, key) → Any or null
func schemaMeta(schema *DSLSchema, args []Object) Object {
	if len(args) != 2 {
		return newArityError("meta", len(args), 2)
	}

	fieldName, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "Schema.meta", "string (field)", args[0].Type())
	}

	key, ok := args[1].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "Schema.meta", "string (key)", args[1].Type())
	}

	field, exists := schema.Fields[fieldName.Value]
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

// schemaFields implements schema.fields() → Array<String>
func schemaFields(schema *DSLSchema, args []Object) Object {
	if len(args) != 0 {
		return newArityError("fields", len(args), 0)
	}

	// Use FieldOrder to preserve declaration order
	var names []string
	if len(schema.FieldOrder) > 0 {
		names = schema.FieldOrder
	} else {
		// Fallback for schemas without FieldOrder (backwards compatibility)
		names = make([]string, 0, len(schema.Fields))
		for name := range schema.Fields {
			names = append(names, name)
		}
		sort.Strings(names)
	}

	elements := make([]Object, len(names))
	for i, name := range names {
		elements[i] = &String{Value: name}
	}

	return &Array{Elements: elements}
}

// schemaVisibleFields implements schema.visibleFields() → Array<String>
// Returns fields where hidden != true AND auto != true (SPEC-ID-008)
func schemaVisibleFields(schema *DSLSchema, args []Object) Object {
	if len(args) != 0 {
		return newArityError("visibleFields", len(args), 0)
	}

	// Use FieldOrder if available, otherwise fall back to map iteration
	var orderedNames []string
	if len(schema.FieldOrder) > 0 {
		orderedNames = schema.FieldOrder
	} else {
		// Fallback for schemas without FieldOrder (backwards compatibility)
		orderedNames = make([]string, 0, len(schema.Fields))
		for name := range schema.Fields {
			orderedNames = append(orderedNames, name)
		}
		sort.Strings(orderedNames)
	}

	// Filter to visible fields, preserving order
	// SPEC-ID-008: Exclude fields with auto constraint (generated by DB/server)
	names := make([]string, 0, len(orderedNames))
	for _, name := range orderedNames {
		field := schema.Fields[name]

		// SPEC-ID-008: Auto fields are excluded from form field iterations
		if field.Auto {
			continue
		}

		// Check if field is hidden via metadata
		hidden := false
		if field.Metadata != nil {
			if hiddenVal, ok := field.Metadata["hidden"]; ok {
				if boolVal, ok := hiddenVal.(*Boolean); ok {
					hidden = boolVal.Value
				}
			}
		}
		if !hidden {
			names = append(names, name)
		}
	}

	elements := make([]Object, len(names))
	for i, name := range names {
		elements[i] = &String{Value: name}
	}

	return &Array{Elements: elements}
}

// schemaEnumValues implements schema.enumValues(field) → Array<String>
func schemaEnumValues(schema *DSLSchema, args []Object) Object {
	if len(args) != 1 {
		return newArityError("enumValues", len(args), 1)
	}

	fieldName, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "Schema.enumValues", "string", args[0].Type())
	}

	field, exists := schema.Fields[fieldName.Value]
	if !exists || len(field.EnumValues) == 0 {
		return &Array{Elements: []Object{}}
	}

	elements := make([]Object, len(field.EnumValues))
	for i, val := range field.EnumValues {
		elements[i] = &String{Value: val}
	}

	return &Array{Elements: elements}
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
			// Create a dict for each field with name, type, required, nullable, auto, readOnly, default
			fieldPairs := make(map[string]ast.Expression)
			fieldPairs["name"] = &ast.StringLiteral{Value: field.Name}
			fieldPairs["type"] = &ast.StringLiteral{Value: field.Type}
			fieldPairs["required"] = &ast.Boolean{Value: field.Required}
			fieldPairs["nullable"] = &ast.Boolean{Value: field.Nullable}
			fieldPairs["auto"] = &ast.Boolean{Value: field.Auto}
			fieldPairs["readOnly"] = &ast.Boolean{Value: field.ReadOnly}
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
		// Check if it's a method name - provide helpful error
		if slices.Contains(dslSchemaMethods, key) {
			return methodAsPropertyError(key, "Schema")
		}
		return NULL
	}
}

// evalSchemaDeclaration evaluates a @schema declaration
func evalSchemaDeclaration(node *ast.SchemaDeclaration, env *Environment) Object {
	schema := &DSLSchema{
		Name:       node.Name.Value,
		Fields:     make(map[string]*DSLSchemaField),
		FieldOrder: make([]string, 0, len(node.Fields)),
		Relations:  make(map[string]*DSLSchemaRelation),
	}

	// Process fields (preserving declaration order)
	for _, field := range node.Fields {
		schema.FieldOrder = append(schema.FieldOrder, field.Name.Value)
		if field.ForeignKey != "" {
			// This is a relation
			schema.Relations[field.Name.Value] = &DSLSchemaRelation{
				FieldName:    field.Name.Value,
				TargetSchema: field.TypeName,
				ForeignKey:   field.ForeignKey,
				IsMany:       field.IsArray,
			}
		} else {
			// Resolve type alias: id → ulid (SPEC-ID-002)
			typeName := field.TypeName
			if strings.ToLower(typeName) == "id" {
				typeName = "ulid"
			}

			// This is a regular field
			dslField := &DSLSchemaField{
				Name:           field.Name.Value,
				Type:           typeName,
				Nullable:       field.Nullable,
				ValidationType: getValidationType(typeName),
				EnumValues:     field.EnumValues,
				Primary:        field.Name.Value == "id", // Convention: "id" field is primary key
			}

			// Evaluate default value if present
			if field.DefaultValue != nil {
				dslField.DefaultValue = Eval(field.DefaultValue, env)
				dslField.DefaultExpr = field.DefaultValue.String()
			}

			// Process type options (min, max, unique, auto, required, etc.)
			// Need to process auto first to determine default required behavior
			if field.TypeOptions != nil {
				// First pass: check for auto
				if valExpr, ok := field.TypeOptions["auto"]; ok {
					val := Eval(valExpr, env)
					if boolVal, ok := val.(*Boolean); ok {
						dslField.Auto = boolVal.Value
					}
				}

				// Second pass: process other options
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
					case "auto":
						// Already processed above
					case "required":
						if boolVal, ok := val.(*Boolean); ok {
							dslField.Required = boolVal.Value
						}
					case "readOnly":
						if boolVal, ok := val.(*Boolean); ok {
							dslField.ReadOnly = boolVal.Value
						}
					case "pattern":
						// SPEC-PAT-001: pattern constraint accepts a regex literal
						// Regex is stored as Dictionary with __type: "regex"
						if regexDict, ok := val.(*Dictionary); ok && isRegexDict(regexDict) {
							// Extract pattern and flags from regex dictionary
							patternStr := ""
							flagsStr := ""
							if patternExpr, ok := regexDict.Pairs["pattern"]; ok {
								if patternObj := Eval(patternExpr, env); patternObj != nil {
									if s, ok := patternObj.(*String); ok {
										patternStr = s.Value
									}
								}
							}
							if flagsExpr, ok := regexDict.Pairs["flags"]; ok {
								if flagsObj := Eval(flagsExpr, env); flagsObj != nil {
									if s, ok := flagsObj.(*String); ok {
										flagsStr = s.Value
									}
								}
							}
							// Compile the regex
							compiled, err := compileRegex(patternStr, flagsStr)
							if err != nil {
								return &Error{
									Class:   ClassType,
									Code:    "SCHEMA-0002",
									Message: fmt.Sprintf("field '%s': invalid regex pattern: %s", field.Name.Value, err),
								}
							}
							dslField.Pattern = compiled
							dslField.PatternSource = patternStr
						} else {
							return &Error{
								Class:   ClassType,
								Code:    "SCHEMA-0002",
								Message: fmt.Sprintf("field '%s': pattern must be a regex literal, got %s", field.Name.Value, val.Type()),
								Hints:   []string{"Use pattern: /^[a-z]+$/ with regex literal syntax"},
							}
						}
					case "default":
						// default value from type options: type(default: value)
						dslField.DefaultValue = val
						dslField.DefaultExpr = valExpr.String()
					}
				}
			}

			// Set default Required based on Nullable and Auto:
			// - If explicitly set via "required" option, use that
			// - Auto fields default to NOT required (they're generated)
			// - Otherwise, required unless nullable
			if _, hasRequiredOption := field.TypeOptions["required"]; !hasRequiredOption {
				if dslField.Auto {
					dslField.Required = false // Auto fields are not required on insert
				} else {
					dslField.Required = !field.Nullable // Required unless nullable
				}
			}

			// SPEC-AUTO-004: auto and required MUST NOT be combined
			if dslField.Auto && dslField.Required {
				return &Error{
					Class:   ClassType,
					Code:    "SCHEMA-0001",
					Message: fmt.Sprintf("field '%s': auto and required cannot be combined", field.Name.Value),
					Hints:   []string{"Auto fields are generated by the database/server and cannot be required on insert"},
				}
			}

			// Process metadata from pipe syntax
			if field.Metadata != nil {
				dslField.Metadata = make(map[string]Object)
				for key, valExpr := range field.Metadata.Pairs {
					dslField.Metadata[key] = Eval(valExpr, env)
				}
			}

			schema.Fields[field.Name.Value] = dslField
		}
	}

	// Register schema in environment (with export if flagged)
	if node.Export {
		env.SetExport(node.Name.Value, schema)
	} else {
		env.Set(node.Name.Value, schema)
	}

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

// buildCreateTableSQL generates CREATE TABLE IF NOT EXISTS SQL from a schema
func buildCreateTableSQL(schema *DSLSchema, tableName string, driver string) string {
	var columns []string

	// Map schema types to SQL types based on driver
	for name, field := range schema.Fields {
		sqlType := schemaTypeToSQL(field.Type, driver)
		var colParts []string
		colParts = append(colParts, name, sqlType)

		// Handle auto fields with primary key (SPEC-ID-003 through SPEC-ID-005)
		if field.Auto && field.Primary {
			baseType := strings.ToLower(field.Type)
			switch baseType {
			case "int", "integer":
				// SPEC-ID-003: Integer primary keys use implicit autoincrement
				switch driver {
				case "sqlite":
					columns = append(columns, fmt.Sprintf("%s INTEGER PRIMARY KEY", name))
				case "postgres":
					columns = append(columns, fmt.Sprintf("%s SERIAL PRIMARY KEY", name))
				case "mysql":
					columns = append(columns, fmt.Sprintf("%s INT AUTO_INCREMENT PRIMARY KEY", name))
				}
				continue
			case "bigint":
				switch driver {
				case "sqlite":
					columns = append(columns, fmt.Sprintf("%s INTEGER PRIMARY KEY", name))
				case "postgres":
					columns = append(columns, fmt.Sprintf("%s BIGSERIAL PRIMARY KEY", name))
				case "mysql":
					columns = append(columns, fmt.Sprintf("%s BIGINT AUTO_INCREMENT PRIMARY KEY", name))
				}
				continue
			case "uuid":
				// UUID auto fields - server generates, TEXT storage
				if driver == "postgres" {
					columns = append(columns, fmt.Sprintf("%s UUID PRIMARY KEY DEFAULT gen_random_uuid()", name))
				} else {
					// SQLite and others: TEXT PRIMARY KEY, server generates value
					columns = append(columns, fmt.Sprintf("%s TEXT PRIMARY KEY", name))
				}
				continue
			case "ulid":
				// ULID auto fields - server generates, TEXT storage
				columns = append(columns, fmt.Sprintf("%s TEXT PRIMARY KEY", name))
				continue
			}
		}

		// Fallback: legacy id field handling (for backwards compatibility)
		if name == "id" && !field.Auto {
			// Non-auto id field: just a regular primary key
			if field.Primary {
				colParts = append(colParts, "PRIMARY KEY")
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
			found := slices.Contains(field.EnumValues, str.Value)
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
