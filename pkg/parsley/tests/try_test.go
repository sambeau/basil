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

// TestTryNestedTry tests that nested try is a syntax error
// (inner try returns dict, which is not a call expression)
func TestTryNestedTry(t *testing.T) {
	// try try func() is a syntax error because inner try returns a dictionary
	// and outer try requires a function call
	input := `try try url("test")`
	result := evalTryHelper(input)

	// Should be a parse error
	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected parse error, got %T: %s", result, result.Inspect())
	}

	// Error should mention "call" or "function"
	msg := strings.ToLower(errObj.Message)
	if !strings.Contains(msg, "call") && !strings.Contains(msg, "function") {
		t.Errorf("expected error about call/function, got: %s", errObj.Message)
	}
}

// TestTryMethodOnNull tests that calling method on null is NOT catchable (it's a bug)
func TestTryMethodOnNull(t *testing.T) {
	// In Parsley, method calls on null return null (null propagation)
	// So try x.foo() where x is null just returns {result: null, error: null}
	// This is the expected behavior - null propagation, not an error
	input := `let x = null; (try x.foo()).result`
	result := evalTryHelper(input)

	// Should be null (null propagation)
	if result != evaluator.NULL {
		t.Errorf("expected null propagation, got %T: %s", result, result.Inspect())
	}
}

// =============================================================================
// FAIL() FUNCTION TESTS (FEAT-030)
// =============================================================================

// TestFailBasic tests that fail() creates an error that terminates without try
func TestFailBasic(t *testing.T) {
	input := `fail("something went wrong")`
	result := evalTryHelper(input)

	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected error, got %T: %s", result, result.Inspect())
	}

	if errObj.Message != "something went wrong" {
		t.Errorf("expected message 'something went wrong', got '%s'", errObj.Message)
	}

	// Should be Value class (catchable)
	if errObj.Class != evaluator.ClassValue {
		t.Errorf("expected ClassValue, got %s", errObj.Class)
	}

	// Should have USER-0001 code
	if errObj.Code != "USER-0001" {
		t.Errorf("expected code 'USER-0001', got '%s'", errObj.Code)
	}
}

// TestFailWithTry tests that try catches fail() errors
func TestFailWithTry(t *testing.T) {
	input := `
let validate = fn(x) {
  if (x < 0) {
    fail("must be non-negative")
  }
  x * 2
}

let {result, error} = try validate(-5)
error
`
	result := evalTryHelper(input)

	strVal, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected string error, got %T: %s", result, result.Inspect())
	}

	if strVal.Value != "must be non-negative" {
		t.Errorf("expected 'must be non-negative', got '%s'", strVal.Value)
	}
}

// TestFailWithTryResult tests that result is null when fail() is caught
func TestFailWithTryResult(t *testing.T) {
	input := `
let validate = fn(x) {
  if (x < 0) {
    fail("must be non-negative")
  }
  x * 2
}

let {result, error} = try validate(-5)
result
`
	result := evalTryHelper(input)

	if result != evaluator.NULL {
		t.Errorf("expected null result, got %T: %s", result, result.Inspect())
	}
}

// TestFailArityError tests that fail() without argument produces Arity error
func TestFailArityError(t *testing.T) {
	input := `fail()`
	result := evalTryHelper(input)

	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected error, got %T: %s", result, result.Inspect())
	}

	// Should be an arity error (not catchable)
	if errObj.Class != evaluator.ClassArity {
		t.Errorf("expected ClassArity, got %s", errObj.Class)
	}
}

// TestFailTypeError tests that fail() with non-string produces Type error
func TestFailTypeError(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"integer", `fail(123)`},
		{"array", `fail([1, 2, 3])`},
		{"dictionary", `fail({a: 1})`},
		{"null", `fail(null)`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalTryHelper(tt.input)

			errObj, ok := result.(*evaluator.Error)
			if !ok {
				t.Fatalf("expected error, got %T: %s", result, result.Inspect())
			}

			// Should be a type error (not catchable)
			if errObj.Class != evaluator.ClassType {
				t.Errorf("expected ClassType, got %s", errObj.Class)
			}
		})
	}
}

// TestFailNestedPropagation tests that fail() propagates through call stack
func TestFailNestedPropagation(t *testing.T) {
	input := `
let inner = fn() { fail("deep error") }
let middle = fn() { inner() }
let outer = fn() { middle() }

let {result, error} = try outer()
error
`
	result := evalTryHelper(input)

	strVal, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected string error, got %T: %s", result, result.Inspect())
	}

	if strVal.Value != "deep error" {
		t.Errorf("expected 'deep error', got '%s'", strVal.Value)
	}
}

// TestFailInCallback tests that fail() works in callbacks like map
func TestFailInCallback(t *testing.T) {
	input := `
let items = [1, 2, 3, 4, 5]
let {result, error} = try items.map(fn(x) {
  if (x == 3) {
    fail("found 3")
  }
  x * 2
})
error
`
	result := evalTryHelper(input)

	strVal, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected string error, got %T: %s", result, result.Inspect())
	}

	if strVal.Value != "found 3" {
		t.Errorf("expected 'found 3', got '%s'", strVal.Value)
	}
}

// TestFailEmptyMessage tests that fail("") is valid
func TestFailEmptyMessage(t *testing.T) {
	input := `let {result, error} = try fail(""); error`
	result := evalTryHelper(input)

	strVal, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected string error, got %T: %s", result, result.Inspect())
	}

	if strVal.Value != "" {
		t.Errorf("expected empty string, got '%s'", strVal.Value)
	}
}
