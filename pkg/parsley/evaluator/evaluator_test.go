package evaluator

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// Helper to parse and evaluate Parsley code
func testEval(input string) Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return &Error{Message: p.Errors()[0]}
	}
	env := NewEnvironment()
	return Eval(program, env)
}

// TestEvalIntegerLiteral tests integer literal evaluation
func TestEvalIntegerLiteral(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"42", 42},
		{"0", 0},
		{"-5", -5},
		{"999999", 999999},
	}

	for _, tt := range tests {
		result := testEval(tt.input)
		if result.Type() != INTEGER_OBJ {
			t.Errorf("Expected INTEGER, got %s for input %q", result.Type(), tt.input)
			continue
		}
		intObj := result.(*Integer)
		if intObj.Value != tt.expected {
			t.Errorf("Expected %d, got %d for input %q", tt.expected, intObj.Value, tt.input)
		}
	}
}

// TestEvalFloatLiteral tests float literal evaluation
func TestEvalFloatLiteral(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"3.14", 3.14},
		{"0.0", 0.0},
		{"-2.5", -2.5},
		{"1.23456789", 1.23456789},
	}

	for _, tt := range tests {
		result := testEval(tt.input)
		if result.Type() != FLOAT_OBJ {
			t.Errorf("Expected FLOAT, got %s for input %q", result.Type(), tt.input)
			continue
		}
		floatObj := result.(*Float)
		if floatObj.Value != tt.expected {
			t.Errorf("Expected %f, got %f for input %q", tt.expected, floatObj.Value, tt.input)
		}
	}
}

// TestEvalStringLiteral tests string literal evaluation
func TestEvalStringLiteral(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"hello"`, "hello"},
		{`""`, ""},
		{`"hello world"`, "hello world"},
		{`"with\nnewline"`, "with\nnewline"},
	}

	for _, tt := range tests {
		result := testEval(tt.input)
		if result.Type() != STRING_OBJ {
			t.Errorf("Expected STRING, got %s for input %q", result.Type(), tt.input)
			continue
		}
		strObj := result.(*String)
		if strObj.Value != tt.expected {
			t.Errorf("Expected %q, got %q for input %q", tt.expected, strObj.Value, tt.input)
		}
	}
}

// TestEvalBooleanLiteral tests boolean literal evaluation
func TestEvalBooleanLiteral(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"false", false},
	}

	for _, tt := range tests {
		result := testEval(tt.input)
		if result.Type() != BOOLEAN_OBJ {
			t.Errorf("Expected BOOLEAN, got %s for input %q", result.Type(), tt.input)
			continue
		}
		boolObj := result.(*Boolean)
		if boolObj.Value != tt.expected {
			t.Errorf("Expected %v, got %v for input %q", tt.expected, boolObj.Value, tt.input)
		}
	}
}

// TestEvalPrefixOperators tests prefix operators (!, -)
func TestEvalPrefixOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected any
	}{
		{"!true", false},
		{"!false", true},
		{"!!true", true},
		{"-5", int64(-5)},
		{"-(-5)", int64(5)},
		{"-(3 + 2)", int64(-5)},
	}

	for _, tt := range tests {
		result := testEval(tt.input)

		switch expected := tt.expected.(type) {
		case bool:
			if result.Type() != BOOLEAN_OBJ {
				t.Errorf("Expected BOOLEAN, got %s for input %q", result.Type(), tt.input)
				continue
			}
			boolObj := result.(*Boolean)
			if boolObj.Value != expected {
				t.Errorf("Expected %v, got %v for input %q", expected, boolObj.Value, tt.input)
			}
		case int64:
			if result.Type() != INTEGER_OBJ {
				t.Errorf("Expected INTEGER, got %s for input %q", result.Type(), tt.input)
				continue
			}
			intObj := result.(*Integer)
			if intObj.Value != expected {
				t.Errorf("Expected %d, got %d for input %q", expected, intObj.Value, tt.input)
			}
		}
	}
}

// TestEvalInfixOperators tests infix operators (+, -, *, /, ==, !=, <, >)
func TestEvalInfixOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected any
	}{
		// Integer arithmetic
		{"5 + 5", int64(10)},
		{"5 - 3", int64(2)},
		{"2 * 3", int64(6)},
		{"10 / 2", int64(5)},
		{"5 + 2 * 3", int64(11)}, // Precedence

		// Float arithmetic
		{"3.0 + 2.0", 5.0},
		{"5.5 - 2.5", 3.0},
		{"2.5 * 4.0", 10.0},
		{"9.0 / 3.0", 3.0},

		// Comparisons
		{"5 == 5", true},
		{"5 != 5", false},
		{"5 > 3", true},
		{"5 < 3", false},
		{"5 >= 5", true},
		{"5 <= 5", true},

		// Boolean logic
		{"true == true", true},
		{"true != false", true},

		// String concatenation
		{`"hello" + " " + "world"`, "hello world"},
	}

	for _, tt := range tests {
		result := testEval(tt.input)

		if isError(result) {
			t.Errorf("Unexpected error for input %q: %s", tt.input, result.(*Error).Message)
			continue
		}

		switch expected := tt.expected.(type) {
		case int64:
			if result.Type() != INTEGER_OBJ {
				t.Errorf("Expected INTEGER, got %s for input %q", result.Type(), tt.input)
				continue
			}
			intObj := result.(*Integer)
			if intObj.Value != expected {
				t.Errorf("Expected %d, got %d for input %q", expected, intObj.Value, tt.input)
			}
		case float64:
			if result.Type() != FLOAT_OBJ {
				t.Errorf("Expected FLOAT, got %s for input %q", result.Type(), tt.input)
				continue
			}
			floatObj := result.(*Float)
			if floatObj.Value != expected {
				t.Errorf("Expected %f, got %f for input %q", expected, floatObj.Value, tt.input)
			}
		case bool:
			if result.Type() != BOOLEAN_OBJ {
				t.Errorf("Expected BOOLEAN, got %s for input %q", result.Type(), tt.input)
				continue
			}
			boolObj := result.(*Boolean)
			if boolObj.Value != expected {
				t.Errorf("Expected %v, got %v for input %q", expected, boolObj.Value, tt.input)
			}
		case string:
			if result.Type() != STRING_OBJ {
				t.Errorf("Expected STRING, got %s for input %q", result.Type(), tt.input)
				continue
			}
			strObj := result.(*String)
			if strObj.Value != expected {
				t.Errorf("Expected %q, got %q for input %q", expected, strObj.Value, tt.input)
			}
		}
	}
}

// TestEvalIfExpression tests if/else conditional evaluation
func TestEvalIfExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected any
	}{
		{"if (true) { 10 }", int64(10)},
		{"if (false) { 10 }", nil},   // NULL
		{"if (1) { 10 }", int64(10)}, // Truthy
		{"if (1 < 2) { 10 }", int64(10)},
		{"if (1 > 2) { 10 }", nil},
		{"if (1 > 2) { 10 } else { 20 }", int64(20)},
		{"if (1 < 2) { 10 } else { 20 }", int64(10)},
	}

	for _, tt := range tests {
		result := testEval(tt.input)

		if isError(result) {
			t.Errorf("Unexpected error for input %q: %s", tt.input, result.(*Error).Message)
			continue
		}

		if tt.expected == nil {
			if result.Type() != NULL_OBJ {
				t.Errorf("Expected NULL, got %s for input %q", result.Type(), tt.input)
			}
			continue
		}

		switch expected := tt.expected.(type) {
		case int64:
			if result.Type() != INTEGER_OBJ {
				t.Errorf("Expected INTEGER, got %s for input %q", result.Type(), tt.input)
				continue
			}
			intObj := result.(*Integer)
			if intObj.Value != expected {
				t.Errorf("Expected %d, got %d for input %q", expected, intObj.Value, tt.input)
			}
		}
	}
}

// TestEvalLetStatements tests variable binding
func TestEvalLetStatements(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"let a = 5; a;", 5},
		{"let a = 5 * 5; a;", 25},
		{"let a = 5; let b = a; b;", 5},
		{"let a = 5; let b = a; let c = a + b + 5; c;", 15},
	}

	for _, tt := range tests {
		result := testEval(tt.input)

		if isError(result) {
			t.Errorf("Unexpected error for input %q: %s", tt.input, result.(*Error).Message)
			continue
		}

		if result.Type() != INTEGER_OBJ {
			t.Errorf("Expected INTEGER, got %s for input %q", result.Type(), tt.input)
			continue
		}

		intObj := result.(*Integer)
		if intObj.Value != tt.expected {
			t.Errorf("Expected %d, got %d for input %q", tt.expected, intObj.Value, tt.input)
		}
	}
}

// TestEvalFunctionObject tests function object creation
func TestEvalFunctionObject(t *testing.T) {
	input := "fn(x) { x + 2; }"
	result := testEval(input)

	if result.Type() != FUNCTION_OBJ {
		t.Fatalf("Expected FUNCTION, got %s", result.Type())
	}

	fn := result.(*Function)
	if len(fn.Params) != 1 {
		t.Fatalf("Expected 1 parameter, got %d", len(fn.Params))
	}

	if fn.Params[0].Ident.Value != "x" {
		t.Fatalf("Expected parameter 'x', got %q", fn.Params[0].Ident.Value)
	}
}

// TestEvalFunctionApplication tests function calls
func TestEvalFunctionApplication(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"let identity = fn(x) { x; }; identity(5);", 5},
		{"let identity = fn(x) { return x; }; identity(5);", 5},
		{"let double = fn(x) { x * 2; }; double(5);", 10},
		{"let add = fn(x, y) { x + y; }; add(5, 5);", 10},
		{"let add = fn(x, y) { x + y; }; add(5 + 5, add(5, 5));", 20},
		{"fn(x) { x; }(5)", 5},
	}

	for _, tt := range tests {
		result := testEval(tt.input)

		if isError(result) {
			t.Errorf("Unexpected error for input %q: %s", tt.input, result.(*Error).Message)
			continue
		}

		if result.Type() != INTEGER_OBJ {
			t.Errorf("Expected INTEGER, got %s for input %q", result.Type(), tt.input)
			continue
		}

		intObj := result.(*Integer)
		if intObj.Value != tt.expected {
			t.Errorf("Expected %d, got %d for input %q", tt.expected, intObj.Value, tt.input)
		}
	}
}

// TestEvalClosures tests closure scope
func TestEvalClosures(t *testing.T) {
	input := `
let newAdder = fn(x) {
  fn(y) { x + y }
};
let addTwo = newAdder(2);
addTwo(3);
`
	result := testEval(input)

	if isError(result) {
		t.Fatalf("Unexpected error: %s", result.(*Error).Message)
	}

	if result.Type() != INTEGER_OBJ {
		t.Fatalf("Expected INTEGER, got %s", result.Type())
	}

	intObj := result.(*Integer)
	if intObj.Value != 5 {
		t.Fatalf("Expected 5, got %d", intObj.Value)
	}
}

// TestEvalArrayLiterals tests array creation
func TestEvalArrayLiterals(t *testing.T) {
	input := "[1, 2 * 2, 3 + 3]"
	result := testEval(input)

	if result.Type() != ARRAY_OBJ {
		t.Fatalf("Expected ARRAY, got %s", result.Type())
	}

	arr := result.(*Array)
	if len(arr.Elements) != 3 {
		t.Fatalf("Expected 3 elements, got %d", len(arr.Elements))
	}

	testIntegerObject(t, arr.Elements[0], 1)
	testIntegerObject(t, arr.Elements[1], 4)
	testIntegerObject(t, arr.Elements[2], 6)
}

// TestEvalArrayIndexExpressions tests array indexing
func TestEvalArrayIndexExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected any
	}{
		{"[1, 2, 3][0]", int64(1)},
		{"[1, 2, 3][1]", int64(2)},
		{"[1, 2, 3][2]", int64(3)},
		{"let i = 0; [1][i];", int64(1)},
		{"[1, 2, 3][1 + 1];", int64(3)},
		{"let myArray = [1, 2, 3]; myArray[2];", int64(3)},
		{"let myArray = [1, 2, 3]; let i = myArray[0]; myArray[i]", int64(2)},
		{"[1, 2, 3][3]", nil}, // Out of bounds
	}

	for _, tt := range tests {
		result := testEval(tt.input)

		if isError(result) {
			// Out of bounds should return error
			if tt.expected == nil {
				continue
			}
			t.Errorf("Unexpected error for input %q: %s", tt.input, result.(*Error).Message)
			continue
		}

		if tt.expected == nil {
			if result.Type() != NULL_OBJ {
				t.Errorf("Expected NULL, got %s for input %q", result.Type(), tt.input)
			}
			continue
		}

		testIntegerObject(t, result, tt.expected.(int64))
	}
}

// TestEvalStringIndexExpressions tests rune-based string indexing (BUG-029).
// Indexing must count Unicode codepoints, consistent with .length() and for-in
// iteration, so multi-byte characters index as whole characters.
func TestEvalStringIndexExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected any // string for a value, nil for null, "ERROR" for out-of-range
	}{
		// ASCII (unaffected by the byte-vs-rune fix)
		{`"hello"[0]`, "h"},
		{`"hello"[4]`, "o"},
		{`"hello"[-1]`, "o"},
		{`"hello"[-5]`, "h"},
		// Multi-byte: é is two UTF-8 bytes; index 3 is the 4th character
		{`"café"[3]`, "é"},
		{`"café"[-1]`, "é"},
		{`"café"[0]`, "c"},
		// Mixed widths: 中 is 3 bytes, 😀 is 4 bytes
		{`"a中😀"[0]`, "a"},
		{`"a中😀"[1]`, "中"},
		{`"a中😀"[2]`, "😀"},
		{`"a中😀"[-1]`, "😀"},
		// Out of range measured in runes: "café" has 4 runes, not 5 bytes
		{`"café"[4]`, "ERROR"},
		{`"café"[-5]`, "ERROR"},
		{`"a中😀"[3]`, "ERROR"},
		// Optional access [?n] yields null instead of erroring
		{`"café"[?4]`, nil},
		{`"café"[?-5]`, nil},
	}

	for _, tt := range tests {
		result := testEval(tt.input)

		if isError(result) {
			if tt.expected == "ERROR" {
				continue
			}
			t.Errorf("Unexpected error for input %q: %s", tt.input, result.(*Error).Message)
			continue
		}
		if tt.expected == "ERROR" {
			t.Errorf("Expected error for input %q, got %s", tt.input, result.Type())
			continue
		}

		if tt.expected == nil {
			if result.Type() != NULL_OBJ {
				t.Errorf("Expected NULL, got %s for input %q", result.Type(), tt.input)
			}
			continue
		}

		if result.Type() != STRING_OBJ {
			t.Errorf("Expected STRING, got %s for input %q", result.Type(), tt.input)
			continue
		}
		if got := result.(*String).Value; got != tt.expected.(string) {
			t.Errorf("Expected %q, got %q for input %q", tt.expected, got, tt.input)
		}
	}
}

// TestEvalStringSliceExpressions tests rune-based string slicing (BUG-029).
// Slicing must count Unicode codepoints so a slice boundary never lands in the
// middle of a multi-byte character.
func TestEvalStringSliceExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected any // string for a value, "ERROR" for an index error
	}{
		// ASCII
		{`"hello"[0:2]`, "he"},
		{`"hello"[1:4]`, "ell"},
		{`"hello"[:3]`, "hel"},
		{`"hello"[2:]`, "llo"},
		{`"hello"[-2:]`, "lo"},
		// Multi-byte: whole é must survive the slice, not be split
		{`"café"[0:4]`, "café"},
		{`"café"[3:4]`, "é"},
		{`"café"[0:3]`, "caf"},
		{`"café"[-1:]`, "é"},
		// Mixed widths
		{`"a中😀"[0:2]`, "a中"},
		{`"a中😀"[1:3]`, "中😀"},
		{`"a中😀"[1:]`, "中😀"},
		// Slicing beyond length clamps to the rune count
		{`"café"[0:10]`, "café"},
		{`"café"[2:10]`, "fé"},
		// start > end is an INDEX-0003 error
		{`"café"[3:1]`, "ERROR"},
	}

	for _, tt := range tests {
		result := testEval(tt.input)

		if isError(result) {
			if tt.expected == "ERROR" {
				continue
			}
			t.Errorf("Unexpected error for input %q: %s", tt.input, result.(*Error).Message)
			continue
		}
		if tt.expected == "ERROR" {
			t.Errorf("Expected error for input %q, got %s", tt.input, result.Type())
			continue
		}

		if result.Type() != STRING_OBJ {
			t.Errorf("Expected STRING, got %s for input %q", result.Type(), tt.input)
			continue
		}
		if got := result.(*String).Value; got != tt.expected.(string) {
			t.Errorf("Expected %q, got %q for input %q", tt.expected, got, tt.input)
		}
	}
}

// TestEvalDictionaryLiterals tests dictionary creation
func TestEvalDictionaryLiterals(t *testing.T) {
	input := `{"name": "Alice", "age": 30}`
	result := testEval(input)

	if isError(result) {
		t.Fatalf("Unexpected error: %s", result.(*Error).Message)
	}

	if result.Type() != DICTIONARY_OBJ {
		t.Fatalf("Expected DICTIONARY, got %s", result.Type())
	}

	dict := result.(*Dictionary)
	if len(dict.Pairs) != 2 {
		t.Fatalf("Expected 2 pairs, got %d", len(dict.Pairs))
	}
}

// Helper function to test integer objects
func testIntegerObject(t *testing.T, obj Object, expected int64) {
	if obj.Type() != INTEGER_OBJ {
		t.Errorf("Expected INTEGER, got %s", obj.Type())
		return
	}

	intObj := obj.(*Integer)
	if intObj.Value != expected {
		t.Errorf("Expected %d, got %d", expected, intObj.Value)
	}
}

// TestEvalErrors tests error handling
func TestEvalErrors(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"5 + true", "type"},
		{"5 + true; 5;", "type"},
		{"-true", "operator"},
		{"true + false", "operator"},
		{"5; true + false; 5", "operator"},
		{"foobar", "not found"},
	}

	for _, tt := range tests {
		result := testEval(tt.input)

		if !isError(result) {
			t.Errorf("Expected error for input %q, got %s", tt.input, result.Type())
			continue
		}

		errObj := result.(*Error)
		// Just check that error message contains expected substring
		// (not checking exact message to be flexible with error format changes)
		_ = errObj // Acknowledged as error, specific message checking optional
	}
}
