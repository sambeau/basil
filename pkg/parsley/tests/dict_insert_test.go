package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func evalDictInsert(input string) evaluator.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		return &evaluator.Error{Message: p.Errors()[0]}
	}

	env := evaluator.NewEnvironment()
	return evaluator.Eval(program, env)
}

// TestDictionaryInsertAfter tests the dictionary insertAfter method
func TestDictionaryInsertAfter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "insert after first key",
			input:    `{a: 1, b: 2, c: 3}.insertAfter("a", "a2", 1.5).keys()`,
			expected: "[a, a2, b, c]",
		},
		{
			name:     "insert after middle key",
			input:    `{a: 1, b: 2, c: 3}.insertAfter("b", "b2", 2.5).keys()`,
			expected: "[a, b, b2, c]",
		},
		{
			name:     "insert after last key",
			input:    `{a: 1, b: 2, c: 3}.insertAfter("c", "d", 4).keys()`,
			expected: "[a, b, c, d]",
		},
		{
			name:     "inserted value accessible",
			input:    `{a: 1, b: 2}.insertAfter("a", "x", 99).x`,
			expected: "99",
		},
		{
			name:     "original dict unchanged",
			input:    `let d = {a: 1, b: 2}; let _ = d.insertAfter("a", "x", 99); d.keys()`,
			expected: "[a, b]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalDictInsert(tt.input)
			if result.Inspect() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.Inspect())
			}
		})
	}
}

// TestDictionaryInsertBefore tests the dictionary insertBefore method
func TestDictionaryInsertBefore(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "insert before first key",
			input:    `{a: 1, b: 2, c: 3}.insertBefore("a", "z", 0).keys()`,
			expected: "[z, a, b, c]",
		},
		{
			name:     "insert before middle key",
			input:    `{a: 1, b: 2, c: 3}.insertBefore("b", "a2", 1.5).keys()`,
			expected: "[a, a2, b, c]",
		},
		{
			name:     "insert before last key",
			input:    `{a: 1, b: 2, c: 3}.insertBefore("c", "b2", 2.5).keys()`,
			expected: "[a, b, b2, c]",
		},
		{
			name:     "inserted value accessible",
			input:    `{a: 1, b: 2}.insertBefore("b", "x", 99).x`,
			expected: "99",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalDictInsert(tt.input)
			if result.Inspect() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.Inspect())
			}
		})
	}
}

// TestDictionaryInsertErrors tests error cases
func TestDictionaryInsertErrors(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name:          "insertAfter - key not found",
			input:         `{a: 1, b: 2}.insertAfter("x", "y", 3)`,
			expectedError: "not found",
		},
		{
			name:          "insertBefore - key not found",
			input:         `{a: 1, b: 2}.insertBefore("x", "y", 3)`,
			expectedError: "not found",
		},
		{
			name:          "insertAfter - duplicate key",
			input:         `{a: 1, b: 2}.insertAfter("a", "b", 3)`,
			expectedError: "already exists",
		},
		{
			name:          "insertBefore - duplicate key",
			input:         `{a: 1, b: 2}.insertBefore("b", "a", 3)`,
			expectedError: "already exists",
		},
		{
			name:          "insertAfter - wrong arg count",
			input:         `{a: 1}.insertAfter("a", "b")`,
			expectedError: "wrong number of arguments",
		},
		{
			name:          "insertBefore - wrong arg count",
			input:         `{a: 1}.insertBefore("a")`,
			expectedError: "wrong number of arguments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalDictInsert(tt.input)
			err, ok := result.(*evaluator.Error)
			if !ok {
				t.Fatalf("expected error, got %T: %s", result, result.Inspect())
			}
			if !strings.Contains(strings.ToLower(err.Message), strings.ToLower(tt.expectedError)) {
				t.Errorf("expected error containing %q, got %q", tt.expectedError, err.Message)
			}
		})
	}
}
