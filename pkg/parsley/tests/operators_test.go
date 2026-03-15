package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func TestLogicalOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// AND operator with &
		{"true & true", "true"},
		{"true & false", "false"},
		{"false & true", "false"},
		{"false & false", "false"},

		// AND operator with 'and' keyword
		{"true and true", "true"},
		{"true and false", "false"},
		{"false and true", "false"},
		{"false and false", "false"},

		// OR operator with |
		{"true | true", "true"},
		{"true | false", "true"},
		{"false | true", "true"},
		{"false | false", "false"},

		// OR operator with 'or' keyword
		{"true or true", "true"},
		{"true or false", "true"},
		{"false or true", "true"},
		{"false or false", "false"},

		// NOT operator with !
		{"!true", "false"},
		{"!false", "true"},
		{"!!true", "true"},
		{"!!false", "false"},

		// NOT operator with 'not' keyword
		{"not true", "false"},
		{"not false", "true"},
		{"not not true", "true"},
		{"not not false", "false"},

		// Combined logical operators
		{"true & true & true", "true"},
		{"true & true & false", "false"},
		{"false | false | true", "true"},
		{"false | false | false", "false"},

		// Logical operators with comparisons
		{"5 > 3 & 2 < 4", "true"},
		{"5 > 3 and 2 < 4", "true"},
		{"5 > 3 & 2 > 4", "false"},
		{"5 < 3 | 2 < 4", "true"},
		{"5 > 3 or 2 > 4", "true"},
		{"5 < 3 or 2 > 4", "false"},

		// Precedence: AND has higher precedence than OR
		{"true | false & false", "true"}, // true | (false & false) = true | false = true
		{"false & true | true", "true"},  // (false & true) | true = false | true = true

		// NOT with other operators
		{"!true & true", "false"},
		{"!(true & false)", "true"},
		{"not (5 > 3)", "false"},
		{"not (5 < 3)", "true"},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) != 0 {
			t.Fatalf("parser errors for %q: %v", tt.input, p.Errors())
		}

		env := evaluator.NewEnvironment()
		result := evaluator.Eval(program, env)

		if result == nil {
			t.Fatalf("Eval returned nil for input: %s", tt.input)
		}

		if result.Type() == evaluator.ERROR_OBJ {
			t.Fatalf("evaluation error for %q: %s", tt.input, result.Inspect())
		}

		if result.Inspect() != tt.expected {
			t.Errorf("Expected %s for input %q, got %s", tt.expected, tt.input, result.Inspect())
		}

		t.Logf("✓ Input: %s, Result: %s", tt.input, result.Inspect())
	}
}

// TestLogicalShortCircuit tests that && and || short-circuit evaluation (BUG-025).
// Before the fix, the right-hand side was always evaluated, causing runtime errors
// when guard patterns like `x != null && x.length() > 0` were used.
func TestLogicalShortCircuit(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// AND short-circuit: right side must NOT be evaluated when left is falsy
		{`let x = null; if (x != null && x.length() > 0) "yes" else "no"`, `no`},
		{`let x = null; if (x != null & x.length() > 0) "yes" else "no"`, `no`},
		{`let x = null; if (x != null and x.length() > 0) "yes" else "no"`, `no`},

		// AND: right side IS evaluated when left is truthy
		{`let x = "hello"; x != null && x.length() > 0`, "true"},
		{`let x = "hello"; x != null and x.length() > 0`, "true"},

		// OR short-circuit: right side must NOT be evaluated when left is truthy
		{`let x = null; if (x == null || x.length() > 0) "yes" else "no"`, `yes`},
		{`let x = null; if (x == null | x.length() > 0) "yes" else "no"`, `yes`},
		{`let x = null; if (x == null or x.length() > 0) "yes" else "no"`, `yes`},

		// OR: right side IS evaluated when left is falsy
		{`let x = null; x != null || x == null`, "true"},

		// Falsy values short-circuit AND correctly
		{`false && 1/0`, "false"},
		{`0 && 1/0`, "false"},
		{`"" && 1/0`, "false"},

		// Truthy values short-circuit OR correctly
		{`true || 1/0`, "true"},
		{`1 || 1/0`, "true"},
		{`"hi" || 1/0`, "true"},

		// Array intersection still works (not short-circuited)
		{`[1,2,3] && [2,3,4]`, "[2, 3]"},
		{`[] && [1,2,3]`, "[]"},
		{`[1,2,3] & [2,3,4]`, "[2, 3]"},

		// Array union still works (not short-circuited)
		{`[1,2,3] || [3,4,5]`, "[1, 2, 3, 4, 5]"},
		{`[] || [1,2,3]`, "[1, 2, 3]"},
		{`[1,2,3] | [3,4,5]`, "[1, 2, 3, 4, 5]"},

		// Dictionary intersection still works (not short-circuited)
		{`{a: 1, b: 2} && {b: 20, c: 30}`, "{b: 2}"},
		{`{} && {a: 1}`, "{}"},

		// Chained short-circuit
		{`false && false && 1/0`, "false"},
		{`true || true || 1/0`, "true"},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) != 0 {
			t.Fatalf("parser errors for %q: %v", tt.input, p.Errors())
		}

		env := evaluator.NewEnvironment()
		result := evaluator.Eval(program, env)

		if result == nil {
			t.Fatalf("Eval returned nil for input: %s", tt.input)
		}

		if result.Type() == evaluator.ERROR_OBJ {
			t.Fatalf("evaluation error for %q: %s", tt.input, result.Inspect())
		}

		if result.Inspect() != tt.expected {
			t.Errorf("Expected %s for input %q, got %s", tt.expected, tt.input, result.Inspect())
		}
	}
}

func TestComparisonOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Not equal operator
		{"5 != 3", "true"},
		{"5 != 5", "false"},
		{"3 != 5", "true"},

		// Less than or equal
		{"3 <= 5", "true"},
		{"5 <= 5", "true"},
		{"5 <= 3", "false"},

		// Greater than or equal (already tested, but let's be thorough)
		{"5 >= 3", "true"},
		{"5 >= 5", "true"},
		{"3 >= 5", "false"},

		// Combined with other operators
		{"5 != 3 & 2 <= 4", "true"},
		{"5 != 5 | 2 <= 4", "true"},
		{"3 <= 5 and 5 >= 3", "true"},

		// String comparisons
		{"\"hello\" != \"world\"", "true"},
		{"\"hello\" != \"hello\"", "false"},

		// Float comparisons
		{"3.5 >= 3.0", "true"},
		{"3.0 <= 3.5", "true"},
		{"3.5 != 3.0", "true"},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) != 0 {
			t.Fatalf("parser errors for %q: %v", tt.input, p.Errors())
		}

		env := evaluator.NewEnvironment()
		result := evaluator.Eval(program, env)

		if result == nil {
			t.Fatalf("Eval returned nil for input: %s", tt.input)
		}

		if result.Type() == evaluator.ERROR_OBJ {
			t.Fatalf("evaluation error for %q: %s", tt.input, result.Inspect())
		}

		if result.Inspect() != tt.expected {
			t.Errorf("Expected %s for input %q, got %s", tt.expected, tt.input, result.Inspect())
		}

		t.Logf("✓ Input: %s, Result: %s", tt.input, result.Inspect())
	}
}
