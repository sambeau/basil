package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func testEvalOptionalIndex(input string) evaluator.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	env := evaluator.NewEnvironment()
	return evaluator.Eval(program, env)
}

// TestOptionalArrayIndexing tests [?n] syntax on arrays
func TestOptionalArrayIndexing(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		// Basic optional indexing - returns value when in bounds
		{`[1, 2, 3][?0]`, 1},
		{`[1, 2, 3][?1]`, 2},
		{`[1, 2, 3][?2]`, 3},

		// Optional indexing - returns null when out of bounds
		{`[1, 2, 3][?99]`, nil},
		{`[1, 2, 3][?-99]`, nil},
		{`[][?0]`, nil},

		// Negative indices still work
		{`[1, 2, 3][?-1]`, 3},
		{`[1, 2, 3][?-2]`, 2},
		{`[1, 2, 3][?-3]`, 1},

		// With null coalesce
		{`[1, 2, 3][?0] ?? "default"`, 1},
		{`[1, 2, 3][?99] ?? "default"`, "default"},
		{`[][?0] ?? "default"`, "default"},

		// Variable index
		{`let arr = [1, 2, 3]; let i = 99; arr[?i]`, nil},
		{`let arr = [1, 2, 3]; let i = 1; arr[?i]`, 2},

		// Regular indexing still errors (unchanged behavior)
		// These would error, so we test that optional doesn't affect regular
		{`[1, 2, 3][0]`, 1},
		{`[1, 2, 3][-1]`, 3},
	}

	for _, tt := range tests {
		result := testEvalOptionalIndex(tt.input)

		switch expected := tt.expected.(type) {
		case nil:
			if result.Type() != evaluator.NULL_OBJ {
				t.Errorf("For input '%s': expected NULL, got %T (%+v)", tt.input, result, result)
			}
		case int:
			intObj, ok := result.(*evaluator.Integer)
			if !ok {
				t.Errorf("For input '%s': expected Integer, got %T (%+v)", tt.input, result, result)
				continue
			}
			if intObj.Value != int64(expected) {
				t.Errorf("For input '%s': expected %d, got %d", tt.input, expected, intObj.Value)
			}
		case string:
			str, ok := result.(*evaluator.String)
			if !ok {
				t.Errorf("For input '%s': expected String, got %T (%+v)", tt.input, result, result)
				continue
			}
			if str.Value != expected {
				t.Errorf("For input '%s': expected '%s', got '%s'", tt.input, expected, str.Value)
			}
		}
	}
}

// TestOptionalStringIndexing tests [?n] syntax on strings
func TestOptionalStringIndexing(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		// Basic optional indexing on strings
		{`"hello"[?0]`, "h"},
		{`"hello"[?4]`, "o"},

		// Out of bounds returns null
		{`"hello"[?99]`, nil},
		{`"hello"[?-99]`, nil},
		{`""[?0]`, nil},

		// Negative indices work
		{`"hello"[?-1]`, "o"},
		{`"hello"[?-5]`, "h"},

		// With null coalesce
		{`"hello"[?0] ?? "?"`, "h"},
		{`"hello"[?99] ?? "?"`, "?"},
		{`""[?0] ?? "empty"`, "empty"},
	}

	for _, tt := range tests {
		result := testEvalOptionalIndex(tt.input)

		switch expected := tt.expected.(type) {
		case nil:
			if result.Type() != evaluator.NULL_OBJ {
				t.Errorf("For input '%s': expected NULL, got %T (%+v)", tt.input, result, result)
			}
		case string:
			str, ok := result.(*evaluator.String)
			if !ok {
				t.Errorf("For input '%s': expected String, got %T (%+v)", tt.input, result, result)
				continue
			}
			if str.Value != expected {
				t.Errorf("For input '%s': expected '%s', got '%s'", tt.input, expected, str.Value)
			}
		}
	}
}

// TestOptionalDictionaryIndexing tests [?key] syntax on dictionaries
func TestOptionalDictionaryIndexing(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		// Dictionaries already return null for missing keys, but [?] should work too
		{`{a: 1, b: 2}[?"a"]`, 1},
		{`{a: 1, b: 2}[?"missing"]`, nil},
		{`{}[?"any"]`, nil},

		// With null coalesce
		{`{a: 1}[?"a"] ?? "default"`, 1},
		{`{a: 1}[?"missing"] ?? "default"`, "default"},
	}

	for _, tt := range tests {
		result := testEvalOptionalIndex(tt.input)

		switch expected := tt.expected.(type) {
		case nil:
			if result.Type() != evaluator.NULL_OBJ {
				t.Errorf("For input '%s': expected NULL, got %T (%+v)", tt.input, result, result)
			}
		case int:
			intObj, ok := result.(*evaluator.Integer)
			if !ok {
				t.Errorf("For input '%s': expected Integer, got %T (%+v)", tt.input, result, result)
				continue
			}
			if intObj.Value != int64(expected) {
				t.Errorf("For input '%s': expected %d, got %d", tt.input, expected, intObj.Value)
			}
		case string:
			str, ok := result.(*evaluator.String)
			if !ok {
				t.Errorf("For input '%s': expected String, got %T (%+v)", tt.input, result, result)
				continue
			}
			if str.Value != expected {
				t.Errorf("For input '%s': expected '%s', got '%s'", tt.input, expected, str.Value)
			}
		}
	}
}

// TestOptionalIndexingWithPathComponents tests the original use case
func TestOptionalIndexingWithPathComponents(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Root path - components is empty, [?0] returns null, coalesce to "home"
		{`let p = @/; p.components[?0] ?? "home"`, "home"},

		// Path with segment - [?0] returns the first component
		{`let p = @/about; p.components[?0] ?? "home"`, "about"},
		{`let p = @/foo/bar; p.components[?0] ?? "home"`, "foo"},

		// Accessing deeper components
		{`let p = @/foo/bar; p.components[?1] ?? "none"`, "bar"},
		{`let p = @/foo/bar; p.components[?2] ?? "none"`, "none"},
	}

	for _, tt := range tests {
		result := testEvalOptionalIndex(tt.input)

		str, ok := result.(*evaluator.String)
		if !ok {
			t.Errorf("For input '%s': expected String, got %T (%+v)", tt.input, result, result)
			continue
		}
		if str.Value != tt.expected {
			t.Errorf("For input '%s': expected '%s', got '%s'", tt.input, tt.expected, str.Value)
		}
	}
}

// TestRegularIndexingStillErrors verifies that normal indexing still throws errors
func TestRegularIndexingStillErrors(t *testing.T) {
	tests := []struct {
		input string
	}{
		{`[1, 2, 3][99]`},
		{`[1, 2, 3][-99]`},
		{`[][0]`},
		{`"hello"[99]`},
		{`""[0]`},
	}

	for _, tt := range tests {
		result := testEvalOptionalIndex(tt.input)

		if result.Type() != evaluator.ERROR_OBJ {
			t.Errorf("For input '%s': expected ERROR, got %T (%+v)", tt.input, result, result)
		}
	}
}
