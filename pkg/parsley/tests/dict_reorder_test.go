package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func evalReorder(input string) evaluator.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		return &evaluator.Error{Message: p.Errors()[0]}
	}

	env := evaluator.NewEnvironment()
	return evaluator.Eval(program, env)
}

// TestDictionaryReorderWithArray tests dict.reorder() with an array of keys
func TestDictionaryReorderWithArray(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "reorder keys",
			input:    `{a: 1, b: 2, c: 3}.reorder(["c", "a", "b"]).keys()`,
			expected: "[c, a, b]",
		},
		{
			name:     "filter keys (subset)",
			input:    `{a: 1, b: 2, c: 3}.reorder(["b", "c"]).keys()`,
			expected: "[b, c]",
		},
		{
			name:     "filter keys (single)",
			input:    `{a: 1, b: 2, c: 3}.reorder(["b"]).keys()`,
			expected: "[b]",
		},
		{
			name:     "ignore non-existent keys",
			input:    `{a: 1, b: 2}.reorder(["z", "a", "x", "b"]).keys()`,
			expected: "[a, b]",
		},
		{
			name:     "preserve values",
			input:    `{a: 1, b: 2, c: 3}.reorder(["c", "a"]).c`,
			expected: "3",
		},
		{
			name:     "empty array returns empty dict",
			input:    `{a: 1, b: 2}.reorder([]).keys()`,
			expected: "[]",
		},
		{
			name:     "original dict unchanged",
			input:    `let d = {a: 1, b: 2}; let _ = d.reorder(["b", "a"]); d.keys()`,
			expected: "[a, b]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalReorder(tt.input)
			if result.Inspect() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.Inspect())
			}
		})
	}
}

// TestDictionaryReorderWithDict tests dict.reorder() with a mapping dictionary
func TestDictionaryReorderWithDict(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "rename keys",
			input:    `{a: 1, b: 2, c: 3}.reorder({x: "a", y: "b", z: "c"}).keys()`,
			expected: "[x, y, z]",
		},
		{
			name:     "reorder without renaming",
			input:    `{a: 1, b: 2, c: 3}.reorder({c: "c", a: "a", b: "b"}).keys()`,
			expected: "[c, a, b]",
		},
		{
			name:     "rename and filter",
			input:    `{a: 1, b: 2, c: 3}.reorder({first: "a", last: "c"}).keys()`,
			expected: "[first, last]",
		},
		{
			name:     "preserve values with rename",
			input:    `{name: "Alice", age: 30}.reorder({full_name: "name"}).full_name`,
			expected: "Alice",
		},
		{
			name:     "ignore non-existent keys in mapping",
			input:    `{a: 1, b: 2}.reorder({x: "a", y: "missing", z: "b"}).keys()`,
			expected: "[x, z]",
		},
		{
			name:     "empty mapping returns empty dict",
			input:    `{a: 1, b: 2}.reorder({}).keys()`,
			expected: "[]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalReorder(tt.input)
			if result.Inspect() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.Inspect())
			}
		})
	}
}

// TestArrayReorderWithArray tests array.reorder() with an array of keys
func TestArrayReorderWithArray(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "reorder all dictionaries in array",
			input:    `[{a: 1, b: 2}, {a: 3, b: 4}].reorder(["b", "a"])[0].keys()`,
			expected: "[b, a]",
		},
		{
			name:     "filter columns from table",
			input:    `[{a: 1, b: 2, c: 3}, {a: 4, b: 5, c: 6}].reorder(["a", "c"])[1].keys()`,
			expected: "[a, c]",
		},
		{
			name:     "preserve values",
			input:    `[{name: "Alice", age: 30}, {name: "Bob", age: 25}].reorder(["age", "name"])[0].age`,
			expected: "30",
		},
		{
			name:     "non-dict elements pass through",
			input:    `[{a: 1}, 42, {b: 2}].reorder(["a", "b"]).length()`,
			expected: "3",
		},
		{
			name:     "empty array stays empty",
			input:    `[].reorder(["a", "b"]).length()`,
			expected: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalReorder(tt.input)
			if result.Inspect() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.Inspect())
			}
		})
	}
}

// TestArrayReorderWithDict tests array.reorder() with a mapping dictionary
func TestArrayReorderWithDict(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "rename columns in table",
			input:    `[{a: 1, b: 2}, {a: 3, b: 4}].reorder({x: "a", y: "b"})[0].keys()`,
			expected: "[x, y]",
		},
		{
			name:     "preserve values with rename",
			input:    `[{name: "Alice"}, {name: "Bob"}].reorder({full_name: "name"})[1].full_name`,
			expected: "Bob",
		},
		{
			name:     "rename and reorder",
			input:    `[{first: "A", last: "B", age: 30}].reorder({surname: "last", given: "first"})[0].keys()`,
			expected: "[surname, given]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalReorder(tt.input)
			if result.Inspect() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.Inspect())
			}
		})
	}
}

// TestReorderErrors tests error cases for reorder
func TestReorderErrors(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "dict reorder with wrong type",
			input:    `{a: 1}.reorder(42)`,
			contains: "array or dictionary",
		},
		{
			name:     "dict reorder with no args",
			input:    `{a: 1}.reorder()`,
			contains: "want=1",
		},
		{
			name:     "dict reorder with too many args",
			input:    `{a: 1}.reorder(["a"], ["b"])`,
			contains: "want=1",
		},
		{
			name:     "array reorder with wrong type",
			input:    `[{a: 1}].reorder(42)`,
			contains: "array or dictionary",
		},
		{
			name:     "dict reorder array with non-string element",
			input:    `{a: 1}.reorder([1, 2, 3])`,
			contains: "string",
		},
		{
			name:     "dict reorder mapping with non-string value",
			input:    `{a: 1}.reorder({x: 123})`,
			contains: "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalReorder(tt.input)
			err, ok := result.(*evaluator.Error)
			if !ok {
				t.Fatalf("expected error, got %T: %s", result, result.Inspect())
			}
			if !containsIgnoreCase(err.Message, tt.contains) {
				t.Errorf("expected error containing %q, got %q", tt.contains, err.Message)
			}
		})
	}
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 &&
			(s[0] == substr[0] || s[0]+32 == substr[0] || s[0] == substr[0]+32) &&
			containsIgnoreCase(s[1:], substr[1:])) ||
		(len(s) > 0 && containsIgnoreCase(s[1:], substr)))
}
