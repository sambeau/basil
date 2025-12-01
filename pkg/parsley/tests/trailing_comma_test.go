package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func TestTrailingCommaArrays(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Basic trailing comma
		{"[1, 2, 3,]", "[1, 2, 3]"},
		{"[1,]", "[1]"},
		{"[1, 2,]", "[1, 2]"},
		
		// Multi-line style (single line for test but same syntax)
		{`["a", "b", "c",]`, "[a, b, c]"},
		
		// Empty array (no trailing comma possible)
		{"[]", "[]"},
		
		// Expressions with trailing comma
		{"[1+2, 3*4,]", "[3, 12]"},
		
		// Nested arrays with trailing commas
		{"[[1,], [2, 3,],]", "[[1], [2, 3]]"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			env := evaluator.NewEnvironment()

			program := p.ParseProgram()
			if len(p.Errors()) != 0 {
				t.Fatalf("Parser errors: %v", p.Errors())
			}

			result := evaluator.Eval(program, env)
			if result == nil {
				t.Fatalf("Eval returned nil")
			}

			if result.Inspect() != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result.Inspect())
			}
		})
	}
}

func TestTrailingCommaDictionaries(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Basic trailing comma
		{`{a: 1, b: 2,}`, `{a: 1, b: 2}`},
		{`{x: 10,}`, `{x: 10}`},
		
		// Empty dictionary (no trailing comma possible)
		{`{}`, `{}`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			env := evaluator.NewEnvironment()

			program := p.ParseProgram()
			if len(p.Errors()) != 0 {
				t.Fatalf("Parser errors: %v", p.Errors())
			}

			result := evaluator.Eval(program, env)
			if result == nil {
				t.Fatalf("Eval returned nil")
			}

			if result.Inspect() != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result.Inspect())
			}
		})
	}
}

func TestTrailingCommaFunctionCalls(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Define a function and call with trailing comma
		{`let add = fn(a, b) { a + b }; add(1, 2,)`, `3`},
		{`let sum3 = fn(a, b, c) { a + b + c }; sum3(1, 2, 3,)`, `6`},
		
		// Single argument with trailing comma
		{`let identity = fn(x) { x }; identity(42,)`, `42`},
		
		// Built-in function with trailing comma
		{`len([1, 2, 3],)`, `3`},
		{`len("hello",)`, `5`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			env := evaluator.NewEnvironment()

			program := p.ParseProgram()
			if len(p.Errors()) != 0 {
				t.Fatalf("Parser errors: %v", p.Errors())
			}

			var result evaluator.Object
			for _, stmt := range program.Statements {
				result = evaluator.Eval(stmt, env)
			}

			if result == nil {
				t.Fatalf("Eval returned nil")
			}

			if result.Inspect() != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result.Inspect())
			}
		})
	}
}

func TestTrailingCommaErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		// Multiple trailing commas should be errors
		{"double comma array", "[1, 2,,]"},
		{"leading comma array", "[,1, 2]"},
		{"double comma dict", "{a: 1,,}"},
		{"leading comma dict", "{,a: 1}"},
		{"double comma function call", "len(x,,)"},
		{"leading comma function call", "len(,x)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)

			p.ParseProgram()
			if len(p.Errors()) == 0 {
				t.Errorf("Expected parse error for %q, but got none", tt.input)
			}
		})
	}
}
