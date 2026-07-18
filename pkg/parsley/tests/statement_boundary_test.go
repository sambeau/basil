package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// TestArrayLiteralAtLineStart tests that a '[' opening a new line begins an
// array-literal statement rather than indexing the previous expression
// (BUG-031). An index/slice bracket must open on the same line as the
// expression it indexes.
func TestArrayLiteralAtLineStart(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "multi-element literal after let",
			input:    "let a = 1\n[1, 2]",
			expected: "[1, 2]",
		},
		{
			name:     "single-element literal after let is not an index",
			input:    "let xs = [10, 20, 30]\n[0]",
			expected: "[0]",
		},
		{
			// Program output collects both expression statements; [0] shows up
			// as its own value and xs is untouched
			name:     "previous binding unchanged by line-start literal",
			input:    "let xs = [10, 20, 30]\n[0]\nxs",
			expected: "[[0], [10, 20, 30]]",
		},
		{
			name:     "literal after expression statement",
			input:    "\"output\"\n[1, 2, 3]",
			expected: "[output, [1, 2, 3]]",
		},
		{
			name:     "literal after method call",
			input:    "let v = \"abc\".length()\n[4, 5]",
			expected: "[4, 5]",
		},
		{
			name:     "same-line index still works",
			input:    "let xs = [10, 20, 30]\nxs[1]",
			expected: "20",
		},
		{
			name:     "same-line slice still works",
			input:    "let xs = [10, 20, 30]\nxs[1:3]",
			expected: "[20, 30]",
		},
		{
			name:     "same-line index on literal still works",
			input:    "[10, 20, 30][2]",
			expected: "30",
		},
		{
			name:     "chained same-line index still works",
			input:    "[[1, 2], [3, 4]][1][0]",
			expected: "3",
		},
		{
			name:     "semicolon-separated index still works",
			input:    "let xs = [10, 20]; xs[0]",
			expected: "10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) != 0 {
				t.Fatalf("Parser errors: %v", p.Errors())
			}

			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)

			if result == nil {
				t.Fatalf("Eval returned nil")
			}
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}
