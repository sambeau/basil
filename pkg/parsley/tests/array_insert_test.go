package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func evalInsert(input string) evaluator.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		return &evaluator.Error{Message: p.Errors()[0]}
	}

	env := evaluator.NewEnvironment()
	return evaluator.Eval(program, env)
}

// TestArrayInsert tests the array insert method
func TestArrayInsert(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Basic insertion
		{
			name:     "insert at beginning",
			input:    `[1, 2, 3].insert(0, "new")`,
			expected: "[new, 1, 2, 3]",
		},
		{
			name:     "insert in middle",
			input:    `[1, 2, 3].insert(1, "new")`,
			expected: "[1, new, 2, 3]",
		},
		{
			name:     "insert at end",
			input:    `[1, 2, 3].insert(3, "new")`,
			expected: "[1, 2, 3, new]",
		},
		{
			name:     "insert into empty array",
			input:    `[].insert(0, "first")`,
			expected: "[first]",
		},

		// Negative indices
		{
			name:     "negative index -1 (before last)",
			input:    `[1, 2, 3].insert(-1, "new")`,
			expected: "[1, 2, new, 3]",
		},
		{
			name:     "negative index -2",
			input:    `[1, 2, 3].insert(-2, "new")`,
			expected: "[1, new, 2, 3]",
		},
		{
			name:     "negative index equal to length",
			input:    `[1, 2, 3].insert(-3, "new")`,
			expected: "[new, 1, 2, 3]",
		},

		// Immutability - original unchanged
		{
			name:     "original array unchanged",
			input:    `let arr = [1, 2, 3]; let _ = arr.insert(1, "new"); arr`,
			expected: "[1, 2, 3]",
		},

		// Different types
		{
			name:     "insert dictionary",
			input:    `[1, 2].insert(1, {a: 1})`,
			expected: "[1, {a: 1}, 2]",
		},
		{
			name:     "insert array",
			input:    `[1, 2].insert(1, [3, 4])`,
			expected: "[1, [3, 4], 2]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInsert(tt.input)
			if result.Inspect() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.Inspect())
			}
		})
	}
}

// TestArrayInsertErrors tests error cases for array insert
func TestArrayInsertErrors(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name:          "index out of bounds (positive)",
			input:         `[1, 2, 3].insert(5, "x")`,
			expectedError: "index 5 out of range",
		},
		{
			name:          "index out of bounds (negative)",
			input:         `[1, 2, 3].insert(-5, "x")`,
			expectedError: "index -5 out of range",
		},
		{
			name:          "wrong number of args (too few)",
			input:         `[1, 2, 3].insert(1)`,
			expectedError: "wrong number of arguments",
		},
		{
			name:          "wrong number of args (too many)",
			input:         `[1, 2, 3].insert(1, 2, 3)`,
			expectedError: "wrong number of arguments",
		},
		{
			name:          "index not an integer",
			input:         `[1, 2, 3].insert("1", "x")`,
			expectedError: "must be an integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInsert(tt.input)
			err, ok := result.(*evaluator.Error)
			if !ok {
				t.Fatalf("expected error, got %T: %s", result, result.Inspect())
			}
			if !containsSubstring(err.Message, tt.expectedError) {
				t.Errorf("expected error containing %q, got %q", tt.expectedError, err.Message)
			}
		})
	}
}

func containsSubstring(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
