package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func evalIn(input string) evaluator.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		return &evaluator.Error{Message: p.Errors()[0]}
	}

	env := evaluator.NewEnvironment()
	return evaluator.Eval(program, env)
}

// TestInArrayMembership tests the 'in' operator with arrays
func TestInArrayMembership(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		// Basic membership
		{"1 in [1, 2, 3]", true},
		{"2 in [1, 2, 3]", true},
		{"3 in [1, 2, 3]", true},
		{"4 in [1, 2, 3]", false},
		{"0 in [1, 2, 3]", false},

		// Empty array
		{"1 in []", false},

		// String elements
		{`"apple" in ["apple", "banana", "cherry"]`, true},
		{`"grape" in ["apple", "banana", "cherry"]`, false},

		// Mixed types
		{`1 in [1, "two", 3]`, true},
		{`"two" in [1, "two", 3]`, true},
		{`2 in [1, "two", 3]`, false},

		// Boolean values
		{"true in [true, false]", true},
		{"false in [true, false]", true},
		{"true in [false, false]", false},

		// Null
		{"null in [1, null, 3]", true},
		{"null in [1, 2, 3]", false},

		// Nested usage with variables
		{`let x = 2; x in [1, 2, 3]`, true},
		{`let arr = [1, 2, 3]; 2 in arr`, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := evalIn(tt.input)
			boolObj, ok := result.(*evaluator.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T (%s)", result, result.Inspect())
			}
			if boolObj.Value != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, boolObj.Value)
			}
		})
	}
}

// TestInDictionaryKey tests the 'in' operator with dictionary keys
func TestInDictionaryKey(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		// Basic key lookup
		{`"name" in {name: "Sam", age: 30}`, true},
		{`"age" in {name: "Sam", age: 30}`, true},
		{`"email" in {name: "Sam", age: 30}`, false},

		// Empty dictionary
		{`"key" in {}`, false},

		// Nested dictionaries (only checks top-level keys)
		{`"user" in {user: {name: "Sam"}}`, true},
		{`"name" in {user: {name: "Sam"}}`, false},

		// Variables
		{`let key = "name"; key in {name: "Sam"}`, true},
		{`let d = {a: 1, b: 2}; "a" in d`, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := evalIn(tt.input)
			boolObj, ok := result.(*evaluator.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T (%s)", result, result.Inspect())
			}
			if boolObj.Value != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, boolObj.Value)
			}
		})
	}
}

// TestInSubstring tests the 'in' operator with strings (substring check)
func TestInSubstring(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		// Basic substring
		{`"world" in "hello world"`, true},
		{`"hello" in "hello world"`, true},
		{`"lo wo" in "hello world"`, true},
		{`"foo" in "hello world"`, false},

		// Empty strings
		{`"" in "hello"`, true},
		{`"hello" in ""`, false},
		{`"" in ""`, true},

		// Case sensitivity
		{`"Hello" in "hello world"`, false},
		{`"WORLD" in "hello world"`, false},

		// Single character
		{`"h" in "hello"`, true},
		{`"z" in "hello"`, false},

		// Variables
		{`let s = "ell"; s in "hello"`, true},
		{`let str = "hello world"; "world" in str`, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := evalIn(tt.input)
			boolObj, ok := result.(*evaluator.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T (%s)", result, result.Inspect())
			}
			if boolObj.Value != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, boolObj.Value)
			}
		})
	}
}

// TestInOperatorPrecedence tests that 'in' has correct precedence
func TestInOperatorPrecedence(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		// Combined with logical operators
		{`1 in [1,2] and 3 in [3,4]`, true},
		{`1 in [1,2] and 5 in [3,4]`, false},
		{`1 in [1,2] or 5 in [3,4]`, true},
		{`5 in [1,2] or 6 in [3,4]`, false},

		// Negation
		{`!(1 in [1,2,3])`, false},
		{`!(4 in [1,2,3])`, true},

		// In conditionals
		{`if (2 in [1,2,3]) { true } else { false }`, true},
		{`if (5 in [1,2,3]) { true } else { false }`, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := evalIn(tt.input)
			boolObj, ok := result.(*evaluator.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T (%s)", result, result.Inspect())
			}
			if boolObj.Value != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, boolObj.Value)
			}
		})
	}
}

// TestInOperatorErrors tests error cases for 'in' operator
func TestInOperatorErrors(t *testing.T) {
	tests := []struct {
		input       string
		expectedErr string
	}{
		// Invalid right operand
		{`1 in 42`, "'in' operator requires array, dictionary, or string on right side, got integer"},
		{`1 in true`, "'in' operator requires array, dictionary, or string on right side, got boolean"},

		// Invalid dictionary key type
		{`1 in {a: 1, b: 2}`, "dictionary key must be a string, got integer"},

		// Invalid substring type
		{`1 in "hello"`, "substring must be a string, got integer"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := evalIn(tt.input)
			errObj, ok := result.(*evaluator.Error)
			if !ok {
				t.Fatalf("expected Error, got %T (%s)", result, result.Inspect())
			}
			if !strings.EqualFold(errObj.Message, tt.expectedErr) {
				t.Errorf("expected error %q, got %q", tt.expectedErr, errObj.Message)
			}
		})
	}
}

// TestIncludesMethodString tests the .includes() method on strings
func TestIncludesMethodString(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		// Basic substring
		{`"hello world".includes("world")`, true},
		{`"hello world".includes("foo")`, false},

		// Empty string
		{`"hello".includes("")`, true},
		{`"".includes("a")`, false},

		// Case sensitivity
		{`"Hello".includes("hello")`, false},

		// With variables
		{`let s = "hello world"; s.includes("world")`, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := evalIn(tt.input)
			boolObj, ok := result.(*evaluator.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T (%s)", result, result.Inspect())
			}
			if boolObj.Value != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, boolObj.Value)
			}
		})
	}
}

// TestInWithNull tests that 'x in null' returns false (null-safe membership)
func TestInWithNull(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		// Basic null cases
		{`1 in null`, false},
		{`"foo" in null`, false},
		{`null in null`, false},
		{`true in null`, false},

		// With variables
		{`let x = null; "key" in x`, false},
		{`let roles = null; "admin" in roles`, false},

		// In conditionals (common pattern)
		{`if ("admin" in null) { true } else { false }`, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := evalIn(tt.input)
			boolObj, ok := result.(*evaluator.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T (%s)", result, result.Inspect())
			}
			if boolObj.Value != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, boolObj.Value)
			}
		})
	}
}

// TestNotInOperator tests the 'not in' compound operator
func TestNotInOperator(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		// Array membership
		{`4 not in [1, 2, 3]`, true},
		{`2 not in [1, 2, 3]`, false},
		{`1 not in []`, true},

		// Dictionary key
		{`"foo" not in {name: "Sam", age: 30}`, true},
		{`"name" not in {name: "Sam", age: 30}`, false},
		{`"key" not in {}`, true},

		// Substring
		{`"xyz" not in "hello world"`, true},
		{`"world" not in "hello world"`, false},
		{`"a" not in ""`, true},

		// With null (null-safe)
		{`"admin" not in null`, true},
		{`1 not in null`, true},

		// With variables
		{`let arr = [1, 2, 3]; 5 not in arr`, true},
		{`let arr = [1, 2, 3]; 2 not in arr`, false},
		{`let d = {a: 1}; "b" not in d`, true},

		// Combined with logical operators
		{`4 not in [1, 2, 3] and 5 not in [4, 5, 6]`, false},
		{`4 not in [1, 2, 3] or 5 not in [4, 5, 6]`, true},

		// In conditionals
		{`if (5 not in [1, 2, 3]) { true } else { false }`, true},
		{`if (2 not in [1, 2, 3]) { true } else { false }`, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := evalIn(tt.input)
			boolObj, ok := result.(*evaluator.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T (%s)", result, result.Inspect())
			}
			if boolObj.Value != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, boolObj.Value)
			}
		})
	}
}
