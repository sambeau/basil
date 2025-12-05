package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// evalTryHelper evaluates code that may use try expressions
func evalTryHelper(input string) evaluator.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return &evaluator.Error{Message: strings.Join(p.Errors(), "\n")}
	}

	env := evaluator.NewEnvironment()
	return evaluator.Eval(program, env)
}

// TestTrySuccessCase tests that try returns {result: value, error: null} on success
func TestTrySuccessCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			name:     "simple_function_call",
			input:    `let add = fn(a, b) { a + b }; (try add(2, 3)).result`,
			expected: 5,
		},
		{
			name:     "nested_function_call",
			input:    `let double = fn(x) { x * 2 }; let triple = fn(x) { x * 3 }; (try double(triple(2))).result`,
			expected: 12,
		},
		{
			name:     "builtin_function",
			input:    `(try toString(42)).result`,
			expected: 0, // String, not int - we'll handle this differently
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "builtin_function" {
				// Special case for string result
				result := evalTryHelper(tt.input)
				str, ok := result.(*evaluator.String)
				if !ok {
					t.Fatalf("expected String, got %T: %s", result, result.Inspect())
				}
				if str.Value != "42" {
					t.Errorf("expected '42', got %q", str.Value)
				}
				return
			}

			result := evalTryHelper(tt.input)
			integer, ok := result.(*evaluator.Integer)
			if !ok {
				t.Fatalf("expected Integer, got %T: %s", result, result.Inspect())
			}
			if integer.Value != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, integer.Value)
			}
		})
	}
}

// TestTryErrorFieldOnSuccess tests that error is null on success
func TestTryErrorFieldOnSuccess(t *testing.T) {
	input := `let add = fn(a, b) { a + b }; (try add(2, 3)).error`
	result := evalTryHelper(input)

	if result != evaluator.NULL {
		t.Errorf("expected NULL, got %T: %s", result, result.Inspect())
	}
}

// TestTryCatchableFormatError tests that Format errors are caught
func TestTryCatchableFormatError(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedInError string // substring that should appear in error
	}{
		{
			name:            "invalid_url",
			input:           `(try url("not a valid url ::: <<<")).error`,
			expectedInError: "url", // should mention URL in error
		},
		{
			name:            "invalid_time_string",
			input:           `(try time("not-a-date")).error`,
			expectedInError: "parse", // should mention parsing issue
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalTryHelper(tt.input)

			// Result should NOT be null - should have an error string
			if result == evaluator.NULL {
				t.Fatal("expected error string, got NULL")
			}

			str, ok := result.(*evaluator.String)
			if !ok {
				t.Fatalf("expected String (error message), got %T: %s", result, result.Inspect())
			}

			if !strings.Contains(strings.ToLower(str.Value), strings.ToLower(tt.expectedInError)) {
				t.Errorf("expected error containing %q, got: %s", tt.expectedInError, str.Value)
			}
		})
	}
}

// TestTryResultNullOnError tests that result is null when error occurs
func TestTryResultNullOnError(t *testing.T) {
	input := `(try url("not a valid url ::: <<<")).result`
	result := evalTryHelper(input)

	if result != evaluator.NULL {
		t.Errorf("expected NULL on error, got %T: %s", result, result.Inspect())
	}
}

// TestTryNonCatchableTypeError tests that Type errors propagate (not caught)
func TestTryNonCatchableTypeError(t *testing.T) {
	// Type errors should NOT be caught by try
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "time_wrong_type",
			input: `try time([1, 2, 3])`, // array instead of string/int/dict
		},
		{
			name:  "url_wrong_type",
			input: `try url(123)`, // integer instead of string
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalTryHelper(tt.input)

			// Should be an Error object, not a dictionary
			errObj, ok := result.(*evaluator.Error)
			if !ok {
				t.Fatalf("expected Error to propagate, got %T: %s", result, result.Inspect())
			}

			// Error should be a Type error
			if errObj.Class != evaluator.ClassType {
				t.Errorf("expected ClassType error, got class: %s", errObj.Class)
			}
		})
	}
}

// TestTryNonCatchableArityError tests that Arity errors propagate (not caught)
func TestTryNonCatchableArityError(t *testing.T) {
	input := `try time()` // time requires 1-2 args
	result := evalTryHelper(input)

	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error to propagate, got %T: %s", result, result.Inspect())
	}

	if errObj.Class != evaluator.ClassArity {
		t.Errorf("expected ClassArity error, got class: %s", errObj.Class)
	}
}

// TestTryNonCatchableUndefinedError tests that Undefined errors propagate (not caught)
func TestTryNonCatchableUndefinedError(t *testing.T) {
	input := `try nonExistentFunction()`
	result := evalTryHelper(input)

	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error to propagate, got %T: %s", result, result.Inspect())
	}

	if errObj.Class != evaluator.ClassUndefined {
		t.Errorf("expected ClassUndefined error, got class: %s", errObj.Class)
	}
}

// TestTryDestructuring tests that try works with destructuring
func TestTryDestructuring(t *testing.T) {
	input := `let add = fn(a, b) { a + b }; let {result, error} = try add(10, 20); result`
	result := evalTryHelper(input)

	integer, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T: %s", result, result.Inspect())
	}
	if integer.Value != 30 {
		t.Errorf("expected 30, got %d", integer.Value)
	}
}

// TestTryDestructuringOnError tests destructuring with error
func TestTryDestructuringOnError(t *testing.T) {
	input := `let {result, error} = try url(":::invalid:::"); error != null`
	result := evalTryHelper(input)

	boolean, ok := result.(*evaluator.Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T: %s", result, result.Inspect())
	}
	if !boolean.Value {
		t.Error("expected error to not be null")
	}
}

// TestTryMethodCall tests try with method calls
func TestTryMethodCall(t *testing.T) {
	input := `let arr = [1, 2, 3]; (try arr.map(fn(x) { x * 2 })).result`
	result := evalTryHelper(input)

	arr, ok := result.(*evaluator.Array)
	if !ok {
		t.Fatalf("expected Array, got %T: %s", result, result.Inspect())
	}

	if len(arr.Elements) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(arr.Elements))
	}

	expected := []int64{2, 4, 6}
	for i, elem := range arr.Elements {
		integer, ok := elem.(*evaluator.Integer)
		if !ok {
			t.Fatalf("element %d: expected Integer, got %T", i, elem)
		}
		if integer.Value != expected[i] {
			t.Errorf("element %d: expected %d, got %d", i, expected[i], integer.Value)
		}
	}
}

// TestTryNullCoalescing tests try with null coalescing operator
func TestTryNullCoalescing(t *testing.T) {
	input := `(try url(":::invalid:::")).result ?? "default"`
	result := evalTryHelper(input)

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T: %s", result, result.Inspect())
	}

	if str.Value != "default" {
		t.Errorf("expected 'default', got %q", str.Value)
	}
}

// TestTryParserError tests that try requires a call expression
func TestTryParserError(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "try_on_literal",
			input: `try 5`,
		},
		{
			name:  "try_on_identifier",
			input: `let x = 5; try x`,
		},
		{
			name:  "try_on_binary_expression",
			input: `try 1 + 2`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalTryHelper(tt.input)

			// Should be an error (from parser)
			errObj, ok := result.(*evaluator.Error)
			if !ok {
				t.Fatalf("expected parser error, got %T: %s", result, result.Inspect())
			}

			// Error message should mention "call" or "function"
			msg := strings.ToLower(errObj.Message)
			if !strings.Contains(msg, "call") && !strings.Contains(msg, "function") {
				t.Errorf("expected error about call/function, got: %s", errObj.Message)
			}
		})
	}
}

// TestTryDictionaryStructure tests that try returns correct dictionary structure
func TestTryDictionaryStructure(t *testing.T) {
	input := `let add = fn(a, b) { a + b }; try add(1, 2)`
	result := evalTryHelper(input)

	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected Dictionary, got %T: %s", result, result.Inspect())
	}

	// Check that both keys exist
	keys := dict.Keys()
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d: %v", len(keys), keys)
	}

	hasResult := false
	hasError := false
	for _, k := range keys {
		if k == "result" {
			hasResult = true
		}
		if k == "error" {
			hasError = true
		}
	}

	if !hasResult {
		t.Error("missing 'result' key")
	}
	if !hasError {
		t.Error("missing 'error' key")
	}
}
