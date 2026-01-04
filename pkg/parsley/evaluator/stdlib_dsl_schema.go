package evaluator

import (
	"fmt"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
)

// DSLSchema represents a schema declared with @schema
type DSLSchema struct {
	Name      string
	Fields    map[string]*DSLSchemaField
	Relations map[string]*DSLSchemaRelation
}

// DSLSchemaField represents a field in a DSL schema
type DSLSchemaField struct {
	Name     string
	Type     string
	Required bool
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
			// Create a dict for each field with name, type, required
			fieldPairs := make(map[string]ast.Expression)
			fieldPairs["name"] = &ast.StringLiteral{Value: field.Name}
			fieldPairs["type"] = &ast.StringLiteral{Value: field.Type}
			fieldPairs["required"] = &ast.Boolean{Value: field.Required}
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
			schema.Fields[field.Name.Value] = &DSLSchemaField{
				Name:     field.Name.Value,
				Type:     field.TypeName,
				Required: true, // Default to required for now
			}
		}
	}

	// Register schema in environment
	env.Set(node.Name.Value, schema)

	// Declarations return NULL (excluded from block concatenation)
	return NULL
}

// Known primitive types for schema fields
var knownPrimitiveTypes = map[string]bool{
	"int":      true,
	"integer":  true,
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
}

// isPrimitiveType checks if a type name is a known primitive type
func isPrimitiveType(typeName string) bool {
	return knownPrimitiveTypes[strings.ToLower(typeName)]
}
