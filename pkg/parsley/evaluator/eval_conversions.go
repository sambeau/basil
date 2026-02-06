package evaluator

import (
	"fmt"
	"strconv"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/lexer"
)

// Data conversion functions for SQL rows, environments, and exports

func rowToDict(columns []string, values []interface{}, env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)

	for i, col := range columns {
		var expr ast.Expression

		switch v := values[i].(type) {
		case int64:
			literal := strconv.FormatInt(v, 10)
			expr = &ast.IntegerLiteral{
				Token: lexer.Token{Type: lexer.INT, Literal: literal},
				Value: v,
			}
		case float64:
			literal := strconv.FormatFloat(v, 'f', -1, 64)
			expr = &ast.FloatLiteral{
				Token: lexer.Token{Type: lexer.FLOAT, Literal: literal},
				Value: v,
			}
		case string:
			expr = &ast.StringLiteral{
				Token: lexer.Token{Type: lexer.STRING, Literal: v},
				Value: v,
			}
		case []byte:
			strVal := string(v)
			expr = &ast.StringLiteral{
				Token: lexer.Token{Type: lexer.STRING, Literal: strVal},
				Value: strVal,
			}
		case bool:
			var tokenType lexer.TokenType
			var literal string
			if v {
				tokenType = lexer.TRUE
				literal = "true"
			} else {
				tokenType = lexer.FALSE
				literal = "false"
			}
			expr = &ast.Boolean{
				Token: lexer.Token{Type: tokenType, Literal: literal},
				Value: v,
			}
		case nil:
			expr = &ast.Identifier{
				Token: lexer.Token{Type: lexer.IDENT, Literal: "null"},
				Value: "null",
			}
		default:
			// For unknown types, convert to string
			strVal := fmt.Sprintf("%v", v)
			expr = &ast.StringLiteral{
				Token: lexer.Token{Type: lexer.STRING, Literal: strVal},
				Value: strVal,
			}
		}

		pairs[col] = expr
	}

	// Preserve column order from SQL query
	return &Dictionary{Pairs: pairs, KeyOrder: columns, Env: env}
}

func environmentToDict(env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)

	// Only export variables that are explicitly exported or declared with 'let'
	for name, value := range env.store {
		if env.IsExported(name) {
			// Wrap the object as a literal expression
			pairs[name] = objectToExpression(value)
		}
	}

	// Create dictionary with the module's environment for evaluation
	return &Dictionary{Pairs: pairs, Env: env}
}

// ExportsToDict exposes the exported bindings of a module environment as a dictionary.
// Intended for host callers (e.g., Basil server) that need to access module exports directly.
func ExportsToDict(env *Environment) *Dictionary {
	return environmentToDict(env)
}

// objectToExpression wraps an Object as an AST expression
func objectToExpression(obj Object) ast.Expression {
	switch v := obj.(type) {
	case *Integer:
		return &ast.IntegerLiteral{
			Token: lexer.Token{Type: lexer.INT, Literal: fmt.Sprintf("%d", v.Value)},
			Value: v.Value,
		}
	case *Float:
		return &ast.FloatLiteral{
			Token: lexer.Token{Type: lexer.FLOAT, Literal: fmt.Sprintf("%g", v.Value)},
			Value: v.Value,
		}
	case *String:
		return &ast.StringLiteral{
			Token: lexer.Token{Type: lexer.STRING, Literal: v.Value},
			Value: v.Value,
		}
	case *Boolean:
		if v.Value {
			return &ast.Boolean{
				Token: lexer.Token{Type: lexer.TRUE, Literal: "true"},
				Value: v.Value,
			}
		}
		return &ast.Boolean{
			Token: lexer.Token{Type: lexer.FALSE, Literal: "false"},
			Value: v.Value,
		}
	default:
		// For complex types (functions, arrays, dictionaries, null), we create
		// an expression that returns the object directly when evaluated
		return &ast.ObjectLiteralExpression{Obj: obj}
	}
}
