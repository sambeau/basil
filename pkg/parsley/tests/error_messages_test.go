package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func evalForError(input string) evaluator.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return &evaluator.Error{Message: strings.Join(p.Errors(), "\n")}
	}

	env := evaluator.NewEnvironment()
	return evaluator.Eval(program, env)
}

// TestNotAFunctionErrorMessages tests that "not a function" errors have helpful hints
func TestNotAFunctionErrorMessages(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string   // Substring that should appear in error
		expectedHint  string   // Hint that should appear
	}{
		{
			name:          "call_null_as_function",
			input:         `let x = null; x()`,
			expectedError: "null",
			expectedHint:  "Hint",
		},
		{
			name:          "call_integer_as_function",
			input:         `let x = 5; x()`,
			expectedError: "cannot call",
			expectedHint:  "function",
		},
		{
			name:          "call_string_as_function",
			input:         `let x = "hello"; x()`,
			expectedError: "cannot call",
			expectedHint:  "function",
		},
		{
			name:          "call_boolean_as_function",
			input:         `let flag = true; flag()`,
			expectedError: "cannot call",
			expectedHint:  "function",
		},
		{
			name:          "call_array_as_function",
			input:         `let arr = [1, 2, 3]; arr()`,
			expectedError: "cannot call",
			expectedHint:  "function",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalForError(tt.input)

			errObj, ok := result.(*evaluator.Error)
			if !ok {
				t.Fatalf("expected error, got %T: %s", result, result.Inspect())
			}

			if !strings.Contains(errObj.Message, tt.expectedError) {
				t.Errorf("expected error containing %q, got: %s", tt.expectedError, errObj.Message)
			}

			if !strings.Contains(errObj.Message, tt.expectedHint) {
				t.Errorf("expected hint containing %q, got: %s", tt.expectedHint, errObj.Message)
			}
		})
	}
}

// TestNullFunctionCallFromImport tests that calling a non-existent export gives helpful error
func TestNullFunctionCallFromImport(t *testing.T) {
	// This tests the case where someone does: let {foo} = import(...) and foo doesn't exist
	// The error should mention that it may not be exported

	input := `
		let x = null
		x()
	`

	result := evalForError(input)

	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected error, got %T: %s", result, result.Inspect())
	}

	// Should mention that it's null
	if !strings.Contains(errObj.Message, "null") {
		t.Errorf("expected error to mention 'null', got: %s", errObj.Message)
	}

	// Should have a helpful hint
	if !strings.Contains(errObj.Message, "Hint") {
		t.Errorf("expected error to have a hint, got: %s", errObj.Message)
	}
}

// TestComponentNotFunctionError tests error when using non-function as component
func TestComponentNotFunctionError(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "number_as_component",
			input: `let Widget = 42; <Widget/>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalForError(tt.input)

			errObj, ok := result.(*evaluator.Error)
			if !ok {
				t.Fatalf("expected error, got %T: %s", result, result.Inspect())
			}

			// Should mention "is not a function"
			if !strings.Contains(errObj.Message, "is not a function") {
				t.Errorf("expected 'is not a function' in error, got: %s", errObj.Message)
			}

			// Should have a hint about components being functions
			if !strings.Contains(errObj.Message, "Hint") {
				t.Errorf("expected hint in error, got: %s", errObj.Message)
			}
		})
	}
}

// TestValidFunctionCallsWork tests that valid function calls still work
func TestValidFunctionCallsWork(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			name:     "call_fn",
			input:    `let f = fn(x) { x * 2 }; f(5)`,
			expected: 10,
		},
		{
			name:     "call_function",
			input:    `let g = function(x) { x + 1 }; g(5)`,
			expected: 6,
		},
		{
			name:     "call_from_dict",
			input:    `let obj = { double: fn(x) { x * 2 } }; obj.double(7)`,
			expected: 14,
		},
		{
			name:     "call_builtin",
			input:    `len([1, 2, 3, 4, 5])`,
			expected: 5,
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

			if errObj, ok := result.(*evaluator.Error); ok {
				t.Fatalf("unexpected error: %s", errObj.Message)
			}

			intResult, ok := result.(*evaluator.Integer)
			if !ok {
				t.Fatalf("expected Integer, got %T: %s", result, result.Inspect())
			}

			if intResult.Value != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, intResult.Value)
			}
		})
	}
}

// TestValidComponentCalls tests that valid component usage works
func TestValidComponentCalls(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "self_closing_component",
			input:    `let Icon = fn(props) { <i class={props.name}/> }; <Icon name="star"/>`,
			expected: `<i class=star />`,
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

			if errObj, ok := result.(*evaluator.Error); ok {
				t.Fatalf("unexpected error: %s", errObj.Message)
			}

			if result.Inspect() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.Inspect())
			}
		})
	}
}
