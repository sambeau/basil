package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// TestCSVTypeCoercion tests that CSV parsing converts values to appropriate types
func TestCSVTypeCoercion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "integers are parsed as integers",
			input: `let data = "num\n42\n-7\n0".parseCSV()
data[0].num`,
			expected: "42",
		},
		{
			name: "floats are parsed as floats",
			input: `let data = "num\n3.14\n-2.5\n0.0".parseCSV()
data[0].num`,
			expected: "3.14",
		},
		{
			name: "booleans are parsed as booleans true",
			input: `let data = "flag\ntrue".parseCSV()
data[0].flag`,
			expected: "true",
		},
		{
			name: "booleans are parsed as booleans false",
			input: `let data = "flag\nfalse".parseCSV()
data[0].flag`,
			expected: "false",
		},
		{
			name: "strings stay as strings",
			input: `let data = "name\nhello\nworld".parseCSV()
data[0].name`,
			expected: "hello",
		},
		{
			name: "mixed types - name column",
			input: `let data = "name,age,score,active\nAlice,30,95.5,true".parseCSV()
data[0].name`,
			expected: `Alice`,
		},
		{
			name: "mixed types - age column",
			input: `let data = "name,age,score,active\nAlice,30,95.5,true".parseCSV()
data[0].age`,
			expected: `30`,
		},
		{
			name: "mixed types - score column",
			input: `let data = "name,age,score,active\nAlice,30,95.5,true".parseCSV()
data[0].score`,
			expected: `95.5`,
		},
		{
			name: "mixed types - active column",
			input: `let data = "name,age,score,active\nAlice,30,95.5,true".parseCSV()
data[0].active`,
			expected: `true`,
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

			if result == nil {
				t.Fatal("result is nil")
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

// TestCSVIntegerType verifies that CSV integers are actual INTEGER type
func TestCSVIntegerType(t *testing.T) {
	input := `let data = "num\n42".parseCSV()
data[0].num`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	env := evaluator.NewEnvironment()
	result := evaluator.Eval(program, env)

	if result.Type() != evaluator.INTEGER_OBJ {
		t.Errorf("expected INTEGER, got %s", result.Type())
	}
}

// TestCSVFloatType verifies that CSV floats are actual FLOAT type
func TestCSVFloatType(t *testing.T) {
	input := `let data = "num\n3.14".parseCSV()
data[0].num`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	env := evaluator.NewEnvironment()
	result := evaluator.Eval(program, env)

	if result.Type() != evaluator.FLOAT_OBJ {
		t.Errorf("expected FLOAT, got %s", result.Type())
	}
}

// TestCSVBooleanType verifies that CSV booleans are actual BOOLEAN type
func TestCSVBooleanType(t *testing.T) {
	input := `let data = "flag\ntrue".parseCSV()
data[0].flag`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	env := evaluator.NewEnvironment()
	result := evaluator.Eval(program, env)

	if result.Type() != evaluator.BOOLEAN_OBJ {
		t.Errorf("expected BOOLEAN, got %s", result.Type())
	}
}

// TestCSVComparisonWithIntegers tests that CSV integers can be compared with literal integers
func TestCSVComparisonWithIntegers(t *testing.T) {
	input := `let data = "value\n10\n20\n5".parseCSV()
data[0].value > 5`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	env := evaluator.NewEnvironment()
	result := evaluator.Eval(program, env)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("comparison failed: %s", result.Inspect())
	}

	if result != evaluator.TRUE {
		t.Errorf("expected true, got %s", result.Inspect())
	}
}

// TestCSVArithmetic tests that CSV numbers can be used in arithmetic
func TestCSVArithmetic(t *testing.T) {
	input := `let data = "a,b\n10,3".parseCSV()
data[0].a + data[0].b`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	env := evaluator.NewEnvironment()
	result := evaluator.Eval(program, env)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("arithmetic failed: %s", result.Inspect())
	}

	intVal, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %s", result.Type())
	}

	if intVal.Value != 13 {
		t.Errorf("expected 13, got %d", intVal.Value)
	}
}
