package evaluator

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
)

// Record represents a schema-bound data container with validation state.
// A Record IS-A Dictionary for data access purposes, enabling compatibility
// with spread syntax, JSON encoding, and functions expecting dictionaries.
//
// Records are immutable - all mutation operations return new Record instances.
type Record struct {
	Schema    *DSLSchema                // The schema this record is bound to
	Data      map[string]ast.Expression // The data fields (like Dictionary.Pairs)
	KeyOrder  []string                  // Insertion order of keys
	Errors    map[string]*RecordError   // Validation errors by field name
	Validated bool                      // Whether validate() has been called
	Env       *Environment              // Environment for lazy evaluation
}

// RecordError represents a validation error for a single field.
type RecordError struct {
	Code    string // Error code (e.g., "REQUIRED", "MIN_LENGTH")
	Message string // Human-readable error message
}

// Type returns the object type for Record.
func (r *Record) Type() ObjectType { return RECORD_OBJ }

// Inspect returns a string representation of the Record.
func (r *Record) Inspect() string {
	var out strings.Builder
	pairs := []string{}

	// Use KeyOrder for consistent output
	keys := r.KeyOrder
	if len(keys) == 0 && len(r.Data) > 0 {
		keys = make([]string, 0, len(r.Data))
		for key := range r.Data {
			keys = append(keys, key)
		}
		sort.Strings(keys)
	}

	for _, key := range keys {
		expr, ok := r.Data[key]
		if !ok {
			continue
		}
		var valueStr string
		if strLit, isStrLit := expr.(*ast.StringLiteral); isStrLit && strLit.Value == "" {
			valueStr = `""`
		} else if objLit, isObjLit := expr.(*ast.ObjectLiteralExpression); isObjLit {
			if strObj, isStr := objLit.Obj.(*String); isStr && strObj.Value == "" {
				valueStr = `""`
			} else if objLit.Obj == nil {
				valueStr = "null"
			} else {
				valueStr = expr.String()
			}
		} else {
			valueStr = expr.String()
		}
		pairs = append(pairs, fmt.Sprintf("%s: %s", key, valueStr))
	}

	schemaName := "?"
	if r.Schema != nil {
		schemaName = r.Schema.Name
	}

	validStatus := "unvalidated"
	if r.Validated {
		if len(r.Errors) == 0 {
			validStatus = "valid"
		} else {
			validStatus = fmt.Sprintf("%d errors", len(r.Errors))
		}
	}

	out.WriteString(schemaName)
	out.WriteString("({")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString("}) [")
	out.WriteString(validStatus)
	out.WriteString("]")

	return out.String()
}

// Get retrieves a field value from the record, evaluating lazily if needed.
func (r *Record) Get(key string, env *Environment) Object {
	expr, ok := r.Data[key]
	if !ok {
		return NULL
	}
	// Use the record's environment for evaluation, falling back to provided env
	evalEnv := r.Env
	if evalEnv == nil {
		evalEnv = env
	}
	if evalEnv == nil {
		// Can't evaluate without an environment
		if strLit, ok := expr.(*ast.StringLiteral); ok {
			return &String{Value: strLit.Value}
		}
		if intLit, ok := expr.(*ast.IntegerLiteral); ok {
			return &Integer{Value: intLit.Value}
		}
		if boolLit, ok := expr.(*ast.Boolean); ok {
			return &Boolean{Value: boolLit.Value}
		}
		return NULL
	}
	return Eval(expr, evalEnv)
}

// Set returns a new Record with the field set to the given value.
// The original Record is unchanged (immutability).
func (r *Record) Set(key string, value Object) *Record {
	newData := make(map[string]ast.Expression, len(r.Data))
	for k, v := range r.Data {
		newData[k] = v
	}
	newData[key] = &ast.ObjectLiteralExpression{Obj: value}

	// Preserve or update key order
	newKeyOrder := make([]string, 0, len(r.KeyOrder)+1)
	found := false
	for _, k := range r.KeyOrder {
		newKeyOrder = append(newKeyOrder, k)
		if k == key {
			found = true
		}
	}
	if !found {
		newKeyOrder = append(newKeyOrder, key)
	}

	return &Record{
		Schema:    r.Schema,
		Data:      newData,
		KeyOrder:  newKeyOrder,
		Errors:    nil,   // Clear errors - needs revalidation
		Validated: false, // Mark as unvalidated
		Env:       r.Env,
	}
}

// Clone creates a shallow copy of the Record.
func (r *Record) Clone() *Record {
	newData := make(map[string]ast.Expression, len(r.Data))
	for k, v := range r.Data {
		newData[k] = v
	}

	newKeyOrder := make([]string, len(r.KeyOrder))
	copy(newKeyOrder, r.KeyOrder)

	newErrors := make(map[string]*RecordError, len(r.Errors))
	for k, v := range r.Errors {
		newErrors[k] = v
	}

	return &Record{
		Schema:    r.Schema,
		Data:      newData,
		KeyOrder:  newKeyOrder,
		Errors:    newErrors,
		Validated: r.Validated,
		Env:       r.Env,
	}
}

// ToDictionary converts the Record to a Dictionary (data only).
func (r *Record) ToDictionary() *Dictionary {
	pairs := make(map[string]ast.Expression, len(r.Data))
	for k, v := range r.Data {
		pairs[k] = v
	}

	keyOrder := make([]string, len(r.KeyOrder))
	copy(keyOrder, r.KeyOrder)

	return &Dictionary{
		Pairs:    pairs,
		KeyOrder: keyOrder,
		Env:      r.Env,
	}
}

// ToDictionaryWithErrors converts the Record to a Dictionary including validation errors.
// Used for storing validated rows in typed tables.
func (r *Record) ToDictionaryWithErrors() *Dictionary {
	pairs := make(map[string]ast.Expression, len(r.Data)+1)
	for k, v := range r.Data {
		pairs[k] = v
	}

	keyOrder := make([]string, len(r.KeyOrder))
	copy(keyOrder, r.KeyOrder)

	// Store errors if present
	if r.Validated && len(r.Errors) > 0 {
		errorPairs := make(map[string]ast.Expression)
		errorKeys := make([]string, 0, len(r.Errors))
		for field, err := range r.Errors {
			errorPairs[field] = &ast.ObjectLiteralExpression{
				Obj: &Dictionary{
					Pairs: map[string]ast.Expression{
						"code":    &ast.ObjectLiteralExpression{Obj: &String{Value: err.Code}},
						"message": &ast.ObjectLiteralExpression{Obj: &String{Value: err.Message}},
					},
					KeyOrder: []string{"code", "message"},
					Env:      r.Env,
				},
			}
			errorKeys = append(errorKeys, field)
		}
		pairs["__errors__"] = &ast.ObjectLiteralExpression{
			Obj: &Dictionary{
				Pairs:    errorPairs,
				KeyOrder: errorKeys,
				Env:      r.Env,
			},
		}
	}

	return &Dictionary{
		Pairs:    pairs,
		KeyOrder: keyOrder,
		Env:      r.Env,
	}
}

// CreateRecord creates a new Record from a schema and dictionary data.
// It applies defaults and filters unknown fields.
func CreateRecord(schema *DSLSchema, data *Dictionary, env *Environment) *Record {
	record := &Record{
		Schema:    schema,
		Data:      make(map[string]ast.Expression),
		KeyOrder:  []string{},
		Errors:    nil,
		Validated: false,
		Env:       env,
	}

	// Process each field defined in the schema
	for fieldName, field := range schema.Fields {
		// Check if data has this field
		if expr, ok := data.Pairs[fieldName]; ok {
			// Evaluate and cast the value
			value := Eval(expr, env)
			if !isError(value) {
				castedValue := castFieldValue(value, field)
				record.Data[fieldName] = &ast.ObjectLiteralExpression{Obj: castedValue}
				record.KeyOrder = append(record.KeyOrder, fieldName)
			}
		} else if field.DefaultValue != nil {
			// Apply default value
			record.Data[fieldName] = &ast.ObjectLiteralExpression{Obj: field.DefaultValue}
			record.KeyOrder = append(record.KeyOrder, fieldName)
		} else {
			// Field is missing and has no default - store null
			record.Data[fieldName] = &ast.ObjectLiteralExpression{Obj: NULL}
			record.KeyOrder = append(record.KeyOrder, fieldName)
		}
	}

	// Use schema field order for KeyOrder (preserves declaration order)
	if len(schema.FieldOrder) > 0 {
		// Use the schema's declared field order
		orderedKeys := make([]string, 0, len(schema.FieldOrder))
		for _, fieldName := range schema.FieldOrder {
			if _, exists := record.Data[fieldName]; exists {
				orderedKeys = append(orderedKeys, fieldName)
			}
		}
		record.KeyOrder = orderedKeys
	} else {
		// Fallback: sort alphabetically for backwards compatibility
		sortedKeys := make([]string, 0, len(schema.Fields))
		for fieldName := range schema.Fields {
			sortedKeys = append(sortedKeys, fieldName)
		}
		sort.Strings(sortedKeys)
		record.KeyOrder = sortedKeys
	}

	return record
}

// castFieldValue casts a value to the appropriate type based on schema field.
func castFieldValue(value Object, field *DSLSchemaField) Object {
	if value == nil || value == NULL {
		return NULL
	}

	// Get the base type (strip nullable marker)
	baseType := strings.TrimSuffix(strings.ToLower(field.Type), "?")

	switch baseType {
	case "int", "integer", "bigint":
		return castToInteger(value)
	case "float", "number":
		return castToFloat(value)
	case "bool", "boolean":
		return castToBoolean(value)
	case "string", "text", "email", "url", "phone", "slug", "uuid", "ulid":
		return castToString(value)
	default:
		// No casting needed
		return value
	}
}

// castToInteger casts a value to Integer.
func castToInteger(value Object) Object {
	switch v := value.(type) {
	case *Integer:
		return v
	case *Float:
		return &Integer{Value: int64(v.Value)}
	case *String:
		if i, err := parseInt(v.Value); err == nil {
			return &Integer{Value: i}
		}
		return value // Return original if parse fails
	case *Boolean:
		if v.Value {
			return &Integer{Value: 1}
		}
		return &Integer{Value: 0}
	default:
		return value
	}
}

// castToFloat casts a value to Float.
func castToFloat(value Object) Object {
	switch v := value.(type) {
	case *Float:
		return v
	case *Integer:
		return &Float{Value: float64(v.Value)}
	case *String:
		if f, err := parseFloat(v.Value); err == nil {
			return &Float{Value: f}
		}
		return value
	default:
		return value
	}
}

// castToBoolean casts a value to Boolean.
func castToBoolean(value Object) Object {
	switch v := value.(type) {
	case *Boolean:
		return v
	case *String:
		lower := strings.ToLower(v.Value)
		if lower == "true" || lower == "1" || lower == "yes" {
			return TRUE
		}
		if lower == "false" || lower == "0" || lower == "no" || lower == "" {
			return FALSE
		}
		return value
	case *Integer:
		return &Boolean{Value: v.Value != 0}
	default:
		return value
	}
}

// castToString casts a value to String.
func castToString(value Object) Object {
	switch v := value.(type) {
	case *String:
		return v
	case *Integer:
		return &String{Value: fmt.Sprintf("%d", v.Value)}
	case *Float:
		return &String{Value: fmt.Sprintf("%g", v.Value)}
	case *Boolean:
		if v.Value {
			return &String{Value: "true"}
		}
		return &String{Value: "false"}
	default:
		return value
	}
}

// Helper functions for parsing (use existing strconv if available)
func parseInt(s string) (int64, error) {
	s = strings.TrimSpace(s)
	var i int64
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}

func parseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

// evalSchemaCall handles calling a schema as a function: Schema({...}) or Schema([...])
func evalSchemaCall(schema *DSLSchema, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return &Error{
			Message: fmt.Sprintf("schema %s expects 1 argument, got %d", schema.Name, len(args)),
			Class:   ClassArity,
		}
	}

	arg := args[0]

	switch v := arg.(type) {
	case *Dictionary:
		// Dictionary → Record
		return CreateRecord(schema, v, env)
	case *Array:
		// Array → Table of Records
		return CreateTypedTable(schema, v, env)
	case *Record:
		// Record → Re-bind to this schema (with type casting)
		return CreateRecord(schema, v.ToDictionary(), env)
	case *Table:
		// Table → Typed Table (bind schema to existing table)
		return BindSchemaToTable(schema, v, env)
	default:
		return &Error{
			Message: fmt.Sprintf("schema %s expects a dictionary or array, got %s", schema.Name, v.Type()),
			Class:   ClassType,
		}
	}
}

// CreateTypedTable creates a Table from an array of dictionaries, binding a schema.
// Each element becomes a Record (unvalidated) in the table.
// Implements SPEC-TBL-001, SPEC-TBL-002, SPEC-TBL-003.
func CreateTypedTable(schema *DSLSchema, arr *Array, env *Environment) Object {
	rows := make([]*Dictionary, 0, len(arr.Elements))

	for i, elem := range arr.Elements {
		// Each element must be a dictionary
		dict, ok := elem.(*Dictionary)
		if !ok {
			return &Error{
				Message: fmt.Sprintf("schema %s array element at index %d must be a dictionary, got %s", schema.Name, i, elem.Type()),
				Class:   ClassType,
			}
		}

		// Create a Record for this row (unvalidated)
		// CreateRecord returns *Record directly (never errors during creation)
		rec := CreateRecord(schema, dict, env)
		rowDict := rec.ToDictionary()
		rowDict.Env = env
		rows = append(rows, rowDict)
	}

	// Determine columns from schema fields (sorted for consistent order)
	columns := make([]string, 0, len(schema.Fields))
	for name := range schema.Fields {
		columns = append(columns, name)
	}
	sort.Strings(columns)

	return &Table{
		Rows:    rows,
		Columns: columns,
		Schema:  schema,
	}
}

// BindSchemaToTable binds a schema to an existing table, converting rows to Records.
// Implements SPEC-TBL-005 for table(data).as(Schema) syntax.
func BindSchemaToTable(schema *DSLSchema, t *Table, env *Environment) Object {
	rows := make([]*Dictionary, 0, len(t.Rows))

	for _, row := range t.Rows {
		// Create a Record for this row (unvalidated)
		// CreateRecord returns *Record directly (never errors during creation)
		rec := CreateRecord(schema, row, env)
		rowDict := rec.ToDictionary()
		rowDict.Env = env
		rows = append(rows, rowDict)
	}

	// Use schema field order for columns (preserves declaration order)
	var columns []string
	if len(schema.FieldOrder) > 0 {
		columns = schema.FieldOrder
	} else {
		// Fallback for schemas without FieldOrder (backwards compatibility)
		columns = make([]string, 0, len(schema.Fields))
		for name := range schema.Fields {
			columns = append(columns, name)
		}
		sort.Strings(columns)
	}

	return &Table{
		Rows:    rows,
		Columns: columns,
		Schema:  schema,
	}
}
