package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// TestTypeMethod tests the universal .type() method on all object types
func TestTypeMethod(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Primitive types
		{"string type", `"hello".type()`, "string"},
		{"integer type", `42.type()`, "integer"},
		{"float type", `3.14.type()`, "float"},
		{"boolean true type", `true.type()`, "boolean"},
		{"boolean false type", `false.type()`, "boolean"},
		{"array type", `[1, 2, 3].type()`, "array"},
		{"function type", `fn(x) { x + 1 }.type()`, "function"},
		{"null type", `null.type()`, "null"},
		
		// Dictionary types
		{"plain dictionary type", `{a: 1, b: 2}.type()`, "dictionary"},
		
		// Note: datetime, duration, path built-ins require fuller environment setup
		// These are tested in integration tests or skipped here
		
		// URL and regex work with basic environment
		{"url type", `url("https://example.com").type()`, "url"},
		{"regex type", `regex("test").type()`, "regex"},
		
		// Built-in function type - skip since `len` isn't in basic test env
		// {"builtin function type", `len.type()`, "builtin"},
		// {"builtin function type from var", `let f = len; f.type()`, "builtin"},
		
		// Variable references
		{"string var type", `let s = "test"; s.type()`, "string"},
		{"array var type", `let a = [1, 2]; a.type()`, "array"},
		{"dict var type", `let d = {x: 1}; d.type()`, "dictionary"},
		
		// Method chaining
		{"type after string method", `"hello".toUpper().type()`, "string"},
		{"type after array method", `[3, 1, 2].sort().type()`, "array"},
		{"type after dict method", `{a: 1, b: 2}.keys().type()`, "array"},
		
		// Expression results
		{"type of arithmetic result", `(10 + 5).type()`, "integer"},
		{"type of string concat result", `("hello" + " " + "world").type()`, "string"},
		{"type of comparison result", `(5 > 3).type()`, "boolean"},
		
		// Nested structures
		{"type of array element", `[1, "two", 3.0][1].type()`, "string"},
		{"type of dict value", `{name: "Alice", age: 30}["name"].type()`, "string"},
		
		// Functions returning different types
		{"type of function return value", `let f = fn() { "result" }; f().type()`, "string"},
		{"type of function with int return", `let f = fn() { 42 }; f().type()`, "integer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()
			
			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}
			
			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)
			
			if isError(result) {
				t.Fatalf("eval error: %s", result.Inspect())
			}
			
			str, ok := result.(*evaluator.String)
			if !ok {
				t.Fatalf("expected String result, got %T (%s)", result, result.Inspect())
			}
			
			if str.Value != tt.expected {
				t.Errorf("expected type %q, got %q", tt.expected, str.Value)
			}
		})
	}
}

// TestTypeMethodNoArgs tests that .type() requires zero arguments
func TestTypeMethodNoArgs(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"type with one arg", `"hello".type("extra")`},
		{"type with two args", `42.type(1, 2)`},
		{"type with array arg", `[1, 2].type([3])`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()
			
			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}
			
			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)
			
			if !isError(result) {
				t.Fatalf("expected error for .type() with arguments, got %T", result)
			}
			
			errObj := result.(*evaluator.Error)
			if errObj.Code != "ARITY-0001" {
				t.Errorf("expected arity error code ARITY-0001, got %s", errObj.Code)
			}
		})
	}
}

// TestTypeMethodConsistency tests that .type() is consistent with __type field
func TestTypeMethodConsistency(t *testing.T) {
	tests := []struct {
		name      string
		setupCode string
		typeCall  string
		typeField string
	}{
		// Note: datetime, duration, path need fuller environment setup
		// Only testing those that work with basic NewEnvironment()
		{
			name:      "url consistency",
			setupCode: `let u = url("https://example.com")`,
			typeCall:  `u.type()`,
			typeField: `u.__type`,
		},
		{
			name:      "regex consistency",
			setupCode: `let r = regex("test")`,
			typeCall:  `r.type()`,
			typeField: `r.__type`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get .type() result
			l := lexer.New(tt.setupCode + "; " + tt.typeCall)
			p := parser.New(l)
			program := p.ParseProgram()
			
			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}
			
			env := evaluator.NewEnvironment()
			typeResult := evaluator.Eval(program, env)
			
			if isError(typeResult) {
				t.Fatalf("eval error for .type(): %s", typeResult.Inspect())
			}
			
			typeStr, ok := typeResult.(*evaluator.String)
			if !ok {
				t.Fatalf("expected String from .type(), got %T", typeResult)
			}
			
			// Get __type field result
			l2 := lexer.New(tt.setupCode + "; " + tt.typeField)
			p2 := parser.New(l2)
			program2 := p2.ParseProgram()
			
			if len(p2.Errors()) > 0 {
				t.Fatalf("parser errors for __type: %v", p2.Errors())
			}
			
			env2 := evaluator.NewEnvironment()
			fieldResult := evaluator.Eval(program2, env2)
			
			if isError(fieldResult) {
				t.Fatalf("eval error for __type: %s", fieldResult.Inspect())
			}
			
			fieldStr, ok := fieldResult.(*evaluator.String)
			if !ok {
				t.Fatalf("expected String from __type, got %T", fieldResult)
			}
			
			// They should match
			if typeStr.Value != fieldStr.Value {
				t.Errorf(".type() returned %q but __type is %q", typeStr.Value, fieldStr.Value)
			}
		})
	}
}

// TestTypeMethodInConditionals tests using .type() in conditional logic
func TestTypeMethodInConditionals(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "if statement with type check",
			input: `
				let x = "hello"
				if x.type() == "string" {
					"is string"
				} else {
					"not string"
				}
			`,
			expected: "is string",
		},
		{
			name: "type-based routing",
			input: `
				let process = fn(val) {
					if val.type() == "integer" {
						"number"
					} else if val.type() == "string" {
						"text"
					} else {
						"other"
					}
				}
				process(42)
			`,
			expected: "number",
		},
		{
			name: "array element type check",
			input: `
				let arr = [1, "two", 3.14, true]
				let types = arr.map(fn(x) { x.type() })
				types[1]
			`,
			expected: "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()
			
			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}
			
			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)
			
			if isError(result) {
				t.Fatalf("eval error: %s", result.Inspect())
			}
			
			if result.Inspect() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.Inspect())
			}
		})
	}
}
