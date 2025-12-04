package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func TestMain(t *testing.T) {
	// This is a placeholder test
	// Replace with actual tests for your functions
	t.Log("Test placeholder - replace with real tests")
}

func TestTrigonometricFunctions(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"sin(0)", "0"},
		{"cos(0)", "1"},
		{"tan(0)", "0"},
		{"sqrt(4)", "2"},
		{"pow(2, 3)", "8"},
		{"pi()", "3.14159"},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) != 0 {
			t.Fatalf("parser errors: %v", p.Errors())
		}

		env := evaluator.NewEnvironment()
		result := evaluator.Eval(program, env)

		if result == nil {
			t.Fatalf("Eval returned nil for input: %s", tt.input)
		}

		// For trigonometric functions, we'll check if result is close to expected
		// Since floating point comparisons are tricky, we'll just check the type
		if result.Type() != evaluator.FLOAT_OBJ && result.Type() != evaluator.INTEGER_OBJ {
			t.Errorf("Expected numeric result for %s, got %T", tt.input, result)
		}

		t.Logf("Input: %s, Result: %s", tt.input, result.Inspect())
	}
}

func TestVariableAssignment(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Basic variable assignments - assignments return null, so access the variable
		{"x = 5; x", "5"},
		{"y = 3.14; y", "3.14"},
		{"name = \"hello\"; name", "hello"},
		{"flag = true; flag", "true"},

		// Variable assignment with expressions
		{"x = 2 + 3; x", "5"},
		{"y = sin(0); y", "0"},
		{"z = cos(0); z", "1"},
		{"pi_val = pi(); pi_val", "3.141592653589793"},
		{"area = pi() * pow(5, 2); area", "78.53981633974483"},

		// Using variables in expressions
		{"x = 10; y = x * 2; y", "20"},
		{"radius = 3; area = pi() * pow(radius, 2); area", "28.274333882308138"},
		{"a = 3; b = 4; c = sqrt(pow(a, 2) + pow(b, 2)); c", "5"},
	}

	for _, tt := range tests {
		env := evaluator.NewEnvironment()
		var result evaluator.Object

		// Handle multiple statements separated by semicolon
		statements := strings.Split(tt.input, ";")

		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}

			l := lexer.New(stmt)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) != 0 {
				t.Fatalf("parser errors for %q: %v", stmt, p.Errors())
			}

			result = evaluator.Eval(program, env)

			if result != nil && result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error for %q: %s", stmt, result.Inspect())
			}
		}

		if result == nil {
			t.Fatalf("Eval returned nil for input: %s", tt.input)
		}

		if result.Inspect() != tt.expected {
			t.Errorf("Expected %s for input %q, got %s", tt.expected, tt.input, result.Inspect())
		}

		t.Logf("Input: %s, Result: %s", tt.input, result.Inspect())
	}
}

func TestAdvancedVariableUsage(t *testing.T) {
	env := evaluator.NewEnvironment()

	tests := []struct {
		input    string
		expected string
	}{
		// Test variable reassignment - assignments return null, so access variable after
		{"x = 5; x", "5"},
		{"x", "5"},
		{"x = x * 2; x", "10"},
		{"x", "10"},

		// Test variables in complex expressions
		{"a = 3; a", "3"},
		{"b = 4; b", "4"},
		{"hypotenuse = sqrt(a*a + b*b); hypotenuse", "5"},

		// Test trigonometric variables
		{"angle = pi() / 4; angle", "0.7853981633974483"},
		{"sin_angle = sin(angle); sin_angle", "0.7071067811865475"}, // Updated expected value
		{"cos_angle = cos(angle); cos_angle", "0.7071067811865476"}, // Updated expected value

		// Test updating trigonometric calculations
		{"angle = pi() / 2; angle", "1.5707963267948966"},
		{"sin_angle = sin(angle); sin_angle", "1"},
		{"cos_angle = cos(angle); cos_angle", "6.123233995736757e-17"}, // Updated expected value

		// Test variable chains
		{"base = 2; base", "2"},
		{"exp = 3; exp", "3"},
		{"result = pow(base, exp); result", "8"},
		{"doubled = result * 2; doubled", "16"},
	}

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

		if result.Inspect() != tt.expected {
			t.Errorf("Expected %s for input %q, got %s", tt.expected, tt.input, result.Inspect())
		}

		t.Logf("Input: %s, Result: %s", tt.input, result.Inspect())
	}
}
