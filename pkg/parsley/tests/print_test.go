package tests

import (
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

// TestPrintBasic tests basic print functionality
func TestPrintBasic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "print single string",
			input:    `print("hello")`,
			expected: "hello",
		},
		{
			name:     "print single integer",
			input:    `print(42)`,
			expected: "42",
		},
		{
			name:     "print single float",
			input:    `print(3.14)`,
			expected: "3.14",
		},
		{
			name:     "print boolean true",
			input:    `print(true)`,
			expected: "true",
		},
		{
			name:     "print boolean false",
			input:    `print(false)`,
			expected: "false",
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
			str, ok := result.(*evaluator.String)
			if !ok {
				t.Fatalf("expected String, got %T (%s)", result, result.Inspect())
			}
			if str.Value != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, str.Value)
			}
		})
	}
}

// TestPrintMultipleArgs tests print with multiple arguments
func TestPrintMultipleArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "print two strings",
			input:    `print("a", "b")`,
			expected: []string{"a", "b"},
		},
		{
			name:     "print three values",
			input:    `print("x", 1, true)`,
			expected: []string{"x", "1", "true"},
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
			arr, ok := result.(*evaluator.Array)
			if !ok {
				t.Fatalf("expected Array, got %T (%s)", result, result.Inspect())
			}
			if len(arr.Elements) != len(tt.expected) {
				t.Fatalf("expected %d elements, got %d", len(tt.expected), len(arr.Elements))
			}
			for i, exp := range tt.expected {
				str, ok := arr.Elements[i].(*evaluator.String)
				if !ok {
					t.Errorf("element %d: expected String, got %T", i, arr.Elements[i])
					continue
				}
				if str.Value != exp {
					t.Errorf("element %d: expected %q, got %q", i, exp, str.Value)
				}
			}
		})
	}
}

// TestPrintln tests println functionality
func TestPrintln(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "println with value",
			input:    `println("hello")`,
			expected: []string{"hello", "\n"},
		},
		{
			name:     "println no args - bare newline",
			input:    `println()`,
			expected: []string{"\n"},
		},
		{
			name:     "println multiple values",
			input:    `println("a", "b")`,
			expected: []string{"a", "b", "\n"},
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

			// Single element case
			if len(tt.expected) == 1 {
				str, ok := result.(*evaluator.String)
				if !ok {
					t.Fatalf("expected String, got %T (%s)", result, result.Inspect())
				}
				if str.Value != tt.expected[0] {
					t.Errorf("expected %q, got %q", tt.expected[0], str.Value)
				}
				return
			}

			arr, ok := result.(*evaluator.Array)
			if !ok {
				t.Fatalf("expected Array, got %T (%s)", result, result.Inspect())
			}
			if len(arr.Elements) != len(tt.expected) {
				t.Fatalf("expected %d elements, got %d", len(tt.expected), len(arr.Elements))
			}
			for i, exp := range tt.expected {
				str, ok := arr.Elements[i].(*evaluator.String)
				if !ok {
					t.Errorf("element %d: expected String, got %T", i, arr.Elements[i])
					continue
				}
				if str.Value != exp {
					t.Errorf("element %d: expected %q, got %q", i, exp, str.Value)
				}
			}
		})
	}
}

// TestPrintNoArgs tests that print() with no arguments returns an error
func TestPrintNoArgs(t *testing.T) {
	result := evalPrint(`print()`)
	if result == nil {
		t.Fatalf("result is nil")
	}
	if result.Type() != evaluator.ERROR_OBJ {
		t.Fatalf("expected error, got %T (%s)", result, result.Inspect())
	}
}

// TestPrintNull tests that print(null) outputs empty string (excluded from results)
func TestPrintNull(t *testing.T) {
	// print(null) should produce empty string which gets excluded
	// Use if true to create a block context
	result := evalPrint(`if true {
		print("a")
		print(null)
		print("b")
	}`)
	if result == nil {
		t.Fatalf("result is nil")
	}
	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("unexpected error: %s", result.Inspect())
	}
	arr, ok := result.(*evaluator.Array)
	if !ok {
		t.Fatalf("expected Array, got %T (%s)", result, result.Inspect())
	}
	// Should be ["a", "b"] - null produces empty string which is skipped
	if len(arr.Elements) != 2 {
		t.Fatalf("expected 2 elements (null skipped), got %d: %s", len(arr.Elements), result.Inspect())
	}
}

// TestPrintInLoop tests print in a for loop
func TestPrintInLoop(t *testing.T) {
	result := evalPrint(`for i in 1..3 { print(i) }`)
	if result == nil {
		t.Fatalf("result is nil")
	}
	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("unexpected error: %s", result.Inspect())
	}
	arr, ok := result.(*evaluator.Array)
	if !ok {
		t.Fatalf("expected Array, got %T (%s)", result, result.Inspect())
	}
	expected := []string{"1", "2", "3"}
	if len(arr.Elements) != len(expected) {
		t.Fatalf("expected %d elements, got %d", len(expected), len(arr.Elements))
	}
	for i, exp := range expected {
		str, ok := arr.Elements[i].(*evaluator.String)
		if !ok {
			t.Errorf("element %d: expected String, got %T", i, arr.Elements[i])
			continue
		}
		if str.Value != exp {
			t.Errorf("element %d: expected %q, got %q", i, exp, str.Value)
		}
	}
}

// TestPrintInterleaved tests print interleaved with expressions
func TestPrintInterleaved(t *testing.T) {
	// Use if true to create a block context
	result := evalPrint(`if true {
		print("a")
		"b"
		print("c")
	}`)
	if result == nil {
		t.Fatalf("result is nil")
	}
	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("unexpected error: %s", result.Inspect())
	}
	arr, ok := result.(*evaluator.Array)
	if !ok {
		t.Fatalf("expected Array, got %T (%s)", result, result.Inspect())
	}
	// Should be ["a", "b", "c"] - interleaved in order
	expected := []string{"a", "b", "c"}
	if len(arr.Elements) != len(expected) {
		t.Fatalf("expected %d elements, got %d: %s", len(expected), len(arr.Elements), result.Inspect())
	}
	for i, exp := range expected {
		str, ok := arr.Elements[i].(*evaluator.String)
		if !ok {
			t.Errorf("element %d: expected String, got %T", i, arr.Elements[i])
			continue
		}
		if str.Value != exp {
			t.Errorf("element %d: expected %q, got %q", i, exp, str.Value)
		}
	}
}

// TestPrintInConditional tests print in if/else
func TestPrintInConditional(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "print in if true",
			input:    `if true { print("yes") }`,
			expected: "yes",
		},
		{
			name:     "print in if false else",
			input:    `if false { print("yes") } else { print("no") }`,
			expected: "no",
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
			str, ok := result.(*evaluator.String)
			if !ok {
				t.Fatalf("expected String, got %T (%s)", result, result.Inspect())
			}
			if str.Value != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, str.Value)
			}
		})
	}
}

// TestPrintArrayRepresentation tests that arrays are printed as concatenated values
func TestPrintArrayRepresentation(t *testing.T) {
	result := evalPrint(`print([1, 2, 3])`)
	if result == nil {
		t.Fatalf("result is nil")
	}
	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("unexpected error: %s", result.Inspect())
	}
	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T (%s)", result, result.Inspect())
	}
	expected := "123"
	if str.Value != expected {
		t.Errorf("expected %q, got %q", expected, str.Value)
	}
}

// TestPrintUTF8 tests UTF-8 handling
func TestPrintUTF8(t *testing.T) {
	result := evalPrint(`print("Hello, 世界! 🎉")`)
	if result == nil {
		t.Fatalf("result is nil")
	}
	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("unexpected error: %s", result.Inspect())
	}
	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T (%s)", result, result.Inspect())
	}
	expected := "Hello, 世界! 🎉"
	if str.Value != expected {
		t.Errorf("expected %q, got %q", expected, str.Value)
	}
}

// TestPrintInFunction tests print inside a function body
func TestPrintInFunction(t *testing.T) {
	result := evalPrint(`
let greet = fn(name) {
	print("Hello, ")
	print(name)
	print("!")
}
greet("World")
`)
	if result == nil {
		t.Fatalf("result is nil")
	}
	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("unexpected error: %s", result.Inspect())
	}
	arr, ok := result.(*evaluator.Array)
	if !ok {
		t.Fatalf("expected Array, got %T (%s)", result, result.Inspect())
	}
	expected := []string{"Hello, ", "World", "!"}
	if len(arr.Elements) != len(expected) {
		t.Fatalf("expected %d elements, got %d: %s", len(expected), len(arr.Elements), result.Inspect())
	}
	for i, exp := range expected {
		str, ok := arr.Elements[i].(*evaluator.String)
		if !ok {
			t.Errorf("element %d: expected String, got %T", i, arr.Elements[i])
			continue
		}
		if str.Value != exp {
			t.Errorf("element %d: expected %q, got %q", i, exp, str.Value)
		}
	}
}
