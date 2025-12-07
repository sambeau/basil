package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// TestNewImportSyntax tests the new import @path syntax
func TestNewImportSyntax(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			name:     "import stdlib with new syntax",
			input:    `import @std/math; math.PI`,
			expected: 3.141592653589793,
		},
		{
			name:     "import stdlib with alias",
			input:    `import @std/math as M; M.PI`,
			expected: 3.141592653589793,
		},
		{
			name:     "import stdlib function with auto-bind",
			input:    `import @std/math; math.floor(3.7)`,
			expected: int64(3),
		},
		{
			name:     "import with alias allows original name reuse",
			input:    `import @std/math as Mathematics; let math = 42; Mathematics.floor(math)`,
			expected: int64(42),
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

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			switch expected := tt.expected.(type) {
			case float64:
				if result.Type() != evaluator.FLOAT_OBJ {
					t.Fatalf("expected FLOAT, got %s (%s)", result.Type(), result.Inspect())
				}
				floatVal := result.(*evaluator.Float).Value
				if floatVal != expected {
					t.Errorf("expected %f, got %f", expected, floatVal)
				}
			case int64:
				if result.Type() != evaluator.INTEGER_OBJ {
					t.Fatalf("expected INTEGER, got %s (%s)", result.Type(), result.Inspect())
				}
				intVal := result.(*evaluator.Integer).Value
				if intVal != expected {
					t.Errorf("expected %d, got %d", expected, intVal)
				}
			}
		})
	}
}

// TestImportDestructuring tests destructuring with new import syntax
func TestImportDestructuring(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			name:     "destructure single export",
			input:    `{floor} = import @std/math; floor(3.9)`,
			expected: int64(3),
		},
		{
			name:     "destructure multiple exports",
			input:    `{floor, ceil} = import @std/math; floor(3.2) + ceil(3.2)`,
			expected: int64(7),
		},
		{
			name:     "destructure with rename",
			input:    `{floor as f} = import @std/math; f(9.9)`,
			expected: int64(9),
		},
		{
			name:     "mixed destructure with old syntax",
			input:    `{PI, E} = import("std/math"); PI > E`,
			expected: true,
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

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			switch expected := tt.expected.(type) {
			case int64:
				if result.Type() != evaluator.INTEGER_OBJ {
					t.Fatalf("expected INTEGER, got %s (%s)", result.Type(), result.Inspect())
				}
				if result.(*evaluator.Integer).Value != expected {
					t.Errorf("expected %d, got %d", expected, result.(*evaluator.Integer).Value)
				}
			case bool:
				if result.Type() != evaluator.BOOLEAN_OBJ {
					t.Fatalf("expected BOOLEAN, got %s (%s)", result.Type(), result.Inspect())
				}
				if result.(*evaluator.Boolean).Value != expected {
					t.Errorf("expected %v, got %v", expected, result.(*evaluator.Boolean).Value)
				}
			}
		})
	}
}

// TestImportBackwardCompat tests that old import() syntax still works
func TestImportBackwardCompat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float64
	}{
		{
			name:     "old syntax with string",
			input:    `let m = import("std/math"); m.PI`,
			expected: 3.141592653589793,
		},
		{
			name:     "old syntax with path literal",
			input:    `let m = import(@std/math); m.PI`,
			expected: 3.141592653589793,
		},
		{
			name:     "old syntax destructure",
			input:    `let {PI} = import("std/math"); PI`,
			expected: 3.141592653589793,
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

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			if result.Type() != evaluator.FLOAT_OBJ {
				t.Fatalf("expected FLOAT, got %s (%s)", result.Type(), result.Inspect())
			}

			if result.(*evaluator.Float).Value != tt.expected {
				t.Errorf("expected %f, got %f", tt.expected, result.(*evaluator.Float).Value)
			}
		})
	}
}

// TestImportASTString tests that ImportExpression.String() works correctly
func TestImportASTString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple import",
			input:    `import @std/math`,
			expected: "import @std/math",
		},
		{
			name:     "import with alias",
			input:    `import @std/math as M`,
			expected: "import @std/math as M",
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

			// The program should have one statement that when stringified matches expected
			if len(program.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(program.Statements))
			}

			got := program.Statements[0].String()
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}
