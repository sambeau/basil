package tests

import (
	"strings"
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
		expected any
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
		expected any
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

// TestDestructuringImportDoesNotShadow tests that destructuring import
// only binds the destructured names, not the path-derived name.
// This allows builtins and other variables with the same name to remain accessible.
func TestDestructuringImportDoesNotShadow(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected any
	}{
		{
			name:     "let destructure does not bind path name",
			input:    `let {floor} = import @std/math; let math = "my math"; math`,
			expected: "my math",
		},
		{
			name:     "bare destructure does not bind path name",
			input:    `{floor} = import @std/math; let math = "my math"; math`,
			expected: "my math",
		},
		{
			name:     "destructure with alias does not bind path name",
			input:    `let {floor as f} = import @std/math; let math = 99; math + f(1.5)`,
			expected: int64(100),
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
			case string:
				if result.Type() != evaluator.STRING_OBJ {
					t.Fatalf("expected STRING, got %s (%s)", result.Type(), result.Inspect())
				}
				if result.(*evaluator.String).Value != expected {
					t.Errorf("expected %q, got %q", expected, result.(*evaluator.String).Value)
				}
			case int64:
				if result.Type() != evaluator.INTEGER_OBJ {
					t.Fatalf("expected INTEGER, got %s (%s)", result.Type(), result.Inspect())
				}
				if result.(*evaluator.Integer).Value != expected {
					t.Errorf("expected %d, got %d", expected, result.(*evaluator.Integer).Value)
				}
			}
		})
	}
}

// TestDestructuringImportPathNameNotBound verifies that accessing the path-derived
// name after a destructuring import results in an "Identifier not found" error.
func TestDestructuringImportPathNameNotBound(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "path name not bound after let destructure",
			input: `let {floor} = import @std/math; math`,
		},
		{
			name:  "path name not bound after bare destructure",
			input: `{floor} = import @std/math; math`,
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

			// We expect an error because "math" should not be bound
			if result.Type() != evaluator.ERROR_OBJ {
				t.Fatalf("expected error for undefined 'math', got %s (%s)", result.Type(), result.Inspect())
			}

			errObj := result.(*evaluator.Error)
			if !strings.Contains(errObj.Message, "Identifier not found") {
				t.Errorf("expected 'Identifier not found' error, got: %s", errObj.Message)
			}
		})
	}
}

// TestOldImportSyntaxRejected tests that old import() syntax is now rejected
func TestOldImportSyntaxRejected(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "old syntax with string",
			input: `let m = import("std/math"); m.PI`,
		},
		{
			name:  "old syntax with path literal in parens",
			input: `let m = import(@std/math); m.PI`,
		},
		{
			name:  "old syntax destructure",
			input: `let {PI} = import("std/math"); PI`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			// Old syntax should now produce an error
			if len(p.Errors()) == 0 {
				t.Fatalf("expected parser error for old import syntax, got none. Program: %s", program.String())
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

// =============================================================================
// BUG-016: Import error position information
// =============================================================================

// TestImportErrorPositions verifies that import errors include correct line/column info
// instead of showing 0:0
func TestImportErrorPositions(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantError  string
		wantLine   int
		wantColumn int
	}{
		{
			name:       "missing export shows position",
			input:      "{nonexistent} = import @std/math",
			wantError:  "Module does not export 'nonexistent'",
			wantLine:   1,
			wantColumn: 2, // position of 'nonexistent' in the pattern
		},
		{
			name:       "missing export on line 2",
			input:      "let x = 1\n{badexport} = import @std/math",
			wantError:  "Module does not export 'badexport'",
			wantLine:   2,
			wantColumn: 2,
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

			if result.Type() != evaluator.ERROR_OBJ {
				t.Fatalf("expected error, got %s: %s", result.Type(), result.Inspect())
			}

			err := result.(*evaluator.Error)

			// Check error message
			if err.Message != tt.wantError {
				t.Errorf("wrong error message: got %q, want %q", err.Message, tt.wantError)
			}

			// Check line number
			if err.Line != tt.wantLine {
				t.Errorf("wrong line: got %d, want %d", err.Line, tt.wantLine)
			}

			// Check column number
			if err.Column != tt.wantColumn {
				t.Errorf("wrong column: got %d, want %d", err.Column, tt.wantColumn)
			}
		})
	}
}

// TestDestructuringTypeErrorPosition verifies that destructuring type mismatch errors
// include correct line/column info
func TestDestructuringTypeErrorPosition(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantError  string
		wantLine   int
		wantColumn int
	}{
		{
			name:       "destructure string shows position",
			input:      `{x} = "not a dict"`,
			wantError:  "Destructuring requires a dictionary or record, got string",
			wantLine:   1,
			wantColumn: 1, // position of '{' token
		},
		{
			name:       "destructure on line 2",
			input:      "let s = \"hello\"\n{x} = s",
			wantError:  "Destructuring requires a dictionary or record, got string",
			wantLine:   2,
			wantColumn: 1,
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

			if result.Type() != evaluator.ERROR_OBJ {
				t.Fatalf("expected error, got %s: %s", result.Type(), result.Inspect())
			}

			err := result.(*evaluator.Error)

			// Check error message
			if err.Message != tt.wantError {
				t.Errorf("wrong error message: got %q, want %q", err.Message, tt.wantError)
			}

			// Check line number
			if err.Line != tt.wantLine {
				t.Errorf("wrong line: got %d, want %d", err.Line, tt.wantLine)
			}

			// Check column number
			if err.Column != tt.wantColumn {
				t.Errorf("wrong column: got %d, want %d", err.Column, tt.wantColumn)
			}
		})
	}
}
