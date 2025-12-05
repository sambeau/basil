package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func TestFunctions(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Basic function assignment and calling
		// Note: assignments return null, so we test the function call instead
		{"add = fn(i, j) { i + j }; add(1, 2)", "3"},
		{"add(10, 20)", "30"},

		// Function with if-else expression
		{"gt = fn(i, j) { if (i > j) true else false }; gt(5, 3)", "true"},
		{"gt(2, 7)", "false"},
		{"gt(5, 5)", "false"},

		// Function with if-return and fallback expression
		{"positive = fn(x) { if (x >= 0) { return \"yes\" } \"no\" }; positive(1)", "yes"},
		{"positive(0)", "yes"},
		{"positive(-1)", "no"},

		// Function with comparison operators
		{"lte = fn(a, b) { if (a <= b) true else false }; lte(3, 5)", "true"},
		{"lte(5, 3)", "false"},
		{"lte(5, 5)", "true"},

		// Functions with floats
		{"multiply = fn(x, y) { x * y }; multiply(2.5, 4.0)", "10"},

		// Functions with trigonometry (using std/math)
		{"math = import(\"std/math\"); sinCos = fn(angle) { math.sin(angle) + math.cos(angle) }; sinCos(0)", "1"},

		// Nested function calls
		{"max = fn(a, b) { if (a > b) a else b }; max(max(1, 2), 3)", "3"},

		// Function returning function result
		{"double = fn(x) { x * 2 }; quadruple = fn(x) { double(double(x)) }; quadruple(5)", "20"},
	}

	env := evaluator.NewEnvironment()

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) != 0 {
			t.Fatalf("parser errors for %q: %v", tt.input, p.Errors())
		}

		result := evaluator.Eval(program, env)

		if result == nil {
			t.Fatalf("Eval returned nil for input: %s", tt.input)
		}

		if result.Type() == evaluator.ERROR_OBJ {
			t.Fatalf("evaluation error for %q: %s", tt.input, result.Inspect())
		}

		resultStr := strings.TrimSpace(result.Inspect())
		expectedStr := strings.TrimSpace(tt.expected)

		if resultStr != expectedStr {
			t.Errorf("Expected %s for input %q, got %s", expectedStr, tt.input, resultStr)
		}

		t.Logf("✓ Input: %s, Result: %s", tt.input, resultStr)
	}
}
