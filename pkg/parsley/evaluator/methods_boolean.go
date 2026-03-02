// Package evaluator provides boolean and null method implementations via declarative registry.
// This file implements boolean and null methods migrating from switch-based dispatch (FEAT-137 Tier 2).
package evaluator

import "github.com/sambeau/basil/pkg/parsley/ast"

// BooleanMethodRegistry defines all methods available on boolean values.
var BooleanMethodRegistry MethodRegistry

// NullMethodRegistry defines all methods available on null values.
var NullMethodRegistry MethodRegistry

func init() {
	BooleanMethodRegistry = MethodRegistry{
		"toBox": {
			Fn:          booleanToBox,
			Arity:       "0",
			Description: "Render as box diagram",
		},
		"repr": {
			Fn:          booleanRepr,
			Arity:       "0",
			Description: "Get representation string",
		},
		"toJSON": {
			Fn:          booleanToJSON,
			Arity:       "0",
			Description: "Convert to JSON string",
		},
		"inspect": {
			Fn:          booleanInspect,
			Arity:       "0",
			Description: "Get debug dictionary",
		},
	}
	RegisterMethodRegistry("boolean", BooleanMethodRegistry)

	NullMethodRegistry = MethodRegistry{
		"toBox": {
			Fn:          nullToBox,
			Arity:       "0",
			Description: "Render as box diagram",
		},
		"repr": {
			Fn:          nullRepr,
			Arity:       "0",
			Description: "Get representation string",
		},
		"toJSON": {
			Fn:          nullToJSON,
			Arity:       "0",
			Description: "Convert to JSON string",
		},
		"inspect": {
			Fn:          nullInspect,
			Arity:       "0",
			Description: "Get debug dictionary",
		},
	}
	RegisterMethodRegistry("null", NullMethodRegistry)
}

// ============================================================================
// Boolean method implementations
// ============================================================================

func booleanToBox(receiver Object, args []Object, env *Environment) Object {
	br := NewBoxRenderer()
	return &String{Value: br.RenderSingleValue(receiver.Inspect())}
}

func booleanRepr(receiver Object, args []Object, env *Environment) Object {
	b := receiver.(*Boolean)
	if b.Value {
		return &String{Value: "true"}
	}
	return &String{Value: "false"}
}

func booleanToJSON(receiver Object, args []Object, env *Environment) Object {
	b := receiver.(*Boolean)
	if b.Value {
		return &String{Value: "true"}
	}
	return &String{Value: "false"}
}

func booleanInspect(receiver Object, args []Object, env *Environment) Object {
	b := receiver.(*Boolean)
	return &Dictionary{
		Pairs: map[string]ast.Expression{
			"__type": createLiteralExpression(&String{Value: "boolean"}),
			"value":  createLiteralExpression(b),
		},
		Env: NewEnvironment(),
	}
}

// ============================================================================
// Null method implementations
// ============================================================================

func nullToBox(receiver Object, args []Object, env *Environment) Object {
	br := NewBoxRenderer()
	return &String{Value: br.RenderSingleValue("null")}
}

func nullRepr(receiver Object, args []Object, env *Environment) Object {
	return &String{Value: "null"}
}

func nullToJSON(receiver Object, args []Object, env *Environment) Object {
	return &String{Value: "null"}
}

func nullInspect(receiver Object, args []Object, env *Environment) Object {
	return &Dictionary{
		Pairs: map[string]ast.Expression{
			"__type": createLiteralExpression(&String{Value: "null"}),
		},
		Env: NewEnvironment(),
	}
}
