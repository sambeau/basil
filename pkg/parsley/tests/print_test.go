package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func evalPrint(input string) evaluator.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return &evaluator.Error{Message: p.Errors()[0]}
	}
	env := evaluator.NewEnvironment()
	return evaluator.Eval(program, env)
}

// TestRemovedPrintFunctions tests that print/println/printf return helpful errors
func TestRemovedPrintFunctions(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedCode   string
		expectedInHint string
	}{
		{
			name:           "print returns helpful error",
			input:          `print("hello")`,
			expectedCode:   "UNDEF-0020",
			expectedInHint: "expression-based output",
		},
		{
			name:           "println returns helpful error",
			input:          `println("hello")`,
			expectedCode:   "UNDEF-0020",
			expectedInHint: "expression-based output",
		},
		{
			name:           "printf returns helpful error",
			input:          `printf("template", {})`,
			expectedCode:   "UNDEF-0020",
			expectedInHint: "expression-based output",
		},
		{
			name:           "print error suggests log()",
			input:          `print("debug")`,
			expectedCode:   "UNDEF-0020",
			expectedInHint: "log(",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalPrint(tt.input)
			if result == nil {
				t.Fatalf("result is nil")
			}

			err, ok := result.(*evaluator.Error)
			if !ok {
				t.Fatalf("expected Error, got %T (%s)", result, result.Inspect())
			}

			if err.Code != tt.expectedCode {
				t.Errorf("expected error code %q, got %q", tt.expectedCode, err.Code)
			}

			// Check that the hints contain the expected guidance
			hintsJoined := strings.Join(err.Hints, " ")
			if !strings.Contains(hintsJoined, tt.expectedInHint) {
				t.Errorf("expected hints to contain %q, got: %v", tt.expectedInHint, err.Hints)
			}
		})
	}
}

// TestExpressionBasedOutput tests that the idiomatic expression-based output works
func TestExpressionBasedOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single string expression",
			input:    `"hello"`,
			expected: "hello",
		},
		{
			name:     "single integer expression",
			input:    `42`,
			expected: "42",
		},
		{
			name:     "expression in block",
			input:    `if true { "hello" }`,
			expected: "hello",
		},
		{
			name:     "conditional expression",
			input:    `if true { "yes" } else { "no" }`,
			expected: "yes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalPrint(tt.input)
			if result == nil {
				t.Fatalf("result is nil")
			}
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("unexpected error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.Inspect())
			}
		})
	}
}

// TestLogStillWorks tests that log() is still available for debugging
func TestLogStillWorks(t *testing.T) {
	// log() should return NULL and not error
	result := evalPrint(`log("debug message")`)
	if result == nil {
		t.Fatalf("result is nil")
	}
	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("log() should not error: %s", result.Inspect())
	}
	if result.Type() != evaluator.NULL_OBJ {
		t.Errorf("log() should return null, got %T (%s)", result, result.Inspect())
	}
}
