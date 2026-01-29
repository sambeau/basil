package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// TestUnicodeIdentifiersEndToEnd tests that Unicode identifiers work through the full pipeline
func TestUnicodeIdentifiersEndToEnd(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Greek letter pi",
			input:    `let π = 3.14; π`,
			expected: "3.14",
		},
		{
			name:     "Greek letter alpha",
			input:    `let α = 1; α + 1`,
			expected: "2",
		},
		{
			name:     "Greek letters in expression",
			input:    `let α = 2; let β = 3; α * β`,
			expected: "6",
		},
		{
			name:     "Japanese identifier",
			input:    `let 結果 = 10; 結果`,
			expected: "10",
		},
		{
			name:     "Chinese function",
			input:    `let 加法 = fn(x, y) { x + y }; 加法(2, 3)`,
			expected: "5",
		},
		{
			name:     "Russian identifier",
			input:    `let привет = "hello"; привет`,
			expected: "hello",
		},
		{
			name:     "Mixed ASCII and Unicode",
			input:    `let total = 10; let Δ = 5; total + Δ`,
			expected: "15",
		},
		{
			name:     "Unicode in function parameter",
			input:    `let f = fn(π) { π * 2 }; f(3.14)`,
			expected: "6.28",
		},
		{
			name:     "Multiple Greek letters",
			input:    `let α = 1; let β = 2; let γ = 3; α + β + γ`,
			expected: "6",
		},
		{
			name:     "Unicode identifier with digit suffix",
			input:    `let α1 = 1; let α2 = 2; α1 + α2`,
			expected: "3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) != 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)

			if errObj, ok := result.(*evaluator.Error); ok {
				t.Fatalf("evaluation error: %s", errObj.Inspect())
			}

			if result.Inspect() != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, result.Inspect())
			}
		})
	}
}

// TestUnicodeIdentifiersInHTML tests Unicode identifiers in HTML context
// Note: Unicode characters in tag attribute interpolation need further work
// in the readTag function. For now, use ASCII variable names in tag attributes.
func TestUnicodeIdentifiersInHTML(t *testing.T) {
	// Use ASCII variable name in tag, but test that Unicode still works in code
	input := `let π = 3.14
let piValue = π
<div class="math" data-pi={piValue}/>`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	result := evaluator.Eval(program, env)

	if errObj, ok := result.(*evaluator.Error); ok {
		t.Fatalf("evaluation error: %s", errObj.Inspect())
	}

	expected := `<div class="math" data-pi="3.14" />`

	if result.Inspect() != expected {
		t.Fatalf("expected:\n%s\ngot:\n%s", expected, result.Inspect())
	}
}
