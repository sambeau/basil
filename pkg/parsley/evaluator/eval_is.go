// eval_is.go - Evaluation of 'is' and 'is not' schema checking expressions
//
// The 'is' operator provides runtime schema checking for Records and Tables:
//   record is User     // true if record's schema is User
//   record is not User // true if record's schema is NOT User
//   table is Product   // true if table's schema is Product
//
// Non-record/table values always return false (no error):
//   null is User       // false
//   "hello" is User    // false
//   {x: 1} is User     // false (plain dict, no schema)

package evaluator

import (
	"github.com/sambeau/basil/pkg/parsley/ast"
)

// evalIsExpression evaluates 'is' and 'is not' schema checking expressions.
// Returns true/false based on whether the value's schema matches the specified schema.
// Non-record/table values return false. Right side must be a schema.
func evalIsExpression(node *ast.IsExpression, env *Environment) Object {
	// Evaluate the left side (the value being checked)
	value := Eval(node.Value, env)
	if isError(value) {
		return value
	}

	// Evaluate the right side (the schema)
	schemaObj := Eval(node.Schema, env)
	if isError(schemaObj) {
		return schemaObj
	}

	// Right side must be a DSLSchema
	schema, ok := schemaObj.(*DSLSchema)
	if !ok {
		return newErrorWithClassAndPos(
			"TypeError",
			node.Token,
			"'is' operator requires a schema on the right side, got %s",
			schemaObj.Type(),
		)
	}

	// Check if value's schema matches
	var matches bool
	switch v := value.(type) {
	case *Record:
		matches = v.Schema == schema
	case *Table:
		matches = v.Schema == schema
	default:
		// Non-record/table values always return false (safe behavior)
		matches = false
	}

	// Handle negation for 'is not'
	if node.Negated {
		matches = !matches
	}

	return nativeBoolToParsBoolean(matches)
}
