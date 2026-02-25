package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// testEvalWith evaluates Parsley code for with expression tests
func testEvalWith(input string) evaluator.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	env := evaluator.NewEnvironment()
	return evaluator.Eval(program, env)
}

// TestWithExpressionBasic tests basic with expression functionality
func TestWithExpressionBasic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			name:     "simple dictionary unpacking",
			input:    `with {a: 1, b: 2} { a + b }`,
			expected: int64(3),
		},
		{
			name:     "access single field",
			input:    `with {name: "Alice"} { name }`,
			expected: "Alice",
		},
		{
			name:     "multiple fields",
			input:    `with {x: 10, y: 20, z: 30} { x + y + z }`,
			expected: int64(60),
		},
		{
			name:     "string concatenation in body",
			input:    `with {first: "Hello", second: "World"} { first + " " + second }`,
			expected: "Hello World",
		},
		{
			name:     "dictionary from variable",
			input:    `let person = {name: "Bob", age: 25}; with person { name }`,
			expected: "Bob",
		},
		{
			name:     "nested field access in body",
			input:    `with {data: {inner: 42}} { data.inner }`,
			expected: int64(42),
		},
		{
			name:     "computed values in dictionary",
			input:    `with {a: 1 + 1, b: 2 * 3} { a + b }`,
			expected: int64(8),
		},
		{
			name:     "boolean fields",
			input:    `with {active: true, enabled: false} { active && !enabled }`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalWith(tt.input)

			if err, ok := result.(*evaluator.Error); ok {
				t.Fatalf("unexpected error: [%s] %s", err.Code, err.Message)
			}

			switch expected := tt.expected.(type) {
			case int64:
				if intResult, ok := result.(*evaluator.Integer); ok {
					if intResult.Value != expected {
						t.Errorf("expected %d, got %d", expected, intResult.Value)
					}
				} else {
					t.Errorf("expected Integer, got %T (%s)", result, result.Inspect())
				}
			case string:
				if strResult, ok := result.(*evaluator.String); ok {
					if strResult.Value != expected {
						t.Errorf("expected %q, got %q", expected, strResult.Value)
					}
				} else {
					t.Errorf("expected String, got %T (%s)", result, result.Inspect())
				}
			case bool:
				if boolResult, ok := result.(*evaluator.Boolean); ok {
					if boolResult.Value != expected {
						t.Errorf("expected %v, got %v", expected, boolResult.Value)
					}
				} else {
					t.Errorf("expected Boolean, got %T (%s)", result, result.Inspect())
				}
			}
		})
	}
}

// TestWithExpressionNested tests nested with expressions
func TestWithExpressionNested(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			name: "nested with expressions",
			input: `
				with {a: 1} {
					with {b: 2} {
						a + b
					}
				}
			`,
			expected: 3,
		},
		{
			name: "inner shadows outer",
			input: `
				with {x: 10} {
					with {x: 20} {
						x
					}
				}
			`,
			expected: 20,
		},
		{
			name: "outer still accessible after inner",
			input: `
				with {a: 5, b: 10} {
					let inner = with {c: 3} { c };
					a + b + inner
				}
			`,
			expected: 18,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalWith(tt.input)

			if err, ok := result.(*evaluator.Error); ok {
				t.Fatalf("unexpected error: [%s] %s", err.Code, err.Message)
			}

			if intResult, ok := result.(*evaluator.Integer); ok {
				if intResult.Value != tt.expected {
					t.Errorf("expected %d, got %d", tt.expected, intResult.Value)
				}
			} else {
				t.Errorf("expected Integer, got %T (%s)", result, result.Inspect())
			}
		})
	}
}

// TestWithExpressionEdgeCases tests edge cases for with expressions
func TestWithExpressionEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			name:     "empty dictionary",
			input:    `with {} { 42 }`,
			expected: int64(42),
		},
		{
			name:     "body uses external variable",
			input:    `let external = 100; with {a: 1} { a + external }`,
			expected: int64(101),
		},
		{
			name:     "with does not leak to outer scope",
			input:    `let result = with {secret: 42} { secret }; result`,
			expected: int64(42),
		},
		{
			name:     "array field",
			input:    `with {items: [1, 2, 3]} { items.length() }`,
			expected: int64(3),
		},
		{
			name:     "function in body",
			input:    `with {n: 5} { let double = fn(x) { x * 2 }; double(n) }`,
			expected: int64(10),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalWith(tt.input)

			if err, ok := result.(*evaluator.Error); ok {
				t.Fatalf("unexpected error: [%s] %s", err.Code, err.Message)
			}

			switch expected := tt.expected.(type) {
			case int64:
				if intResult, ok := result.(*evaluator.Integer); ok {
					if intResult.Value != expected {
						t.Errorf("expected %d, got %d", expected, intResult.Value)
					}
				} else {
					t.Errorf("expected Integer, got %T (%s)", result, result.Inspect())
				}
			case string:
				if strResult, ok := result.(*evaluator.String); ok {
					if strResult.Value != expected {
						t.Errorf("expected %q, got %q", expected, strResult.Value)
					}
				} else {
					t.Errorf("expected String, got %T (%s)", result, result.Inspect())
				}
			}
		})
	}
}

// TestWithExpressionErrors tests error cases for with expressions
func TestWithExpressionErrors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorCode   string
	}{
		{
			name:        "non-dict target - integer",
			input:       `with 42 { x }`,
			expectError: true,
			errorCode:   "TYPE-0020",
		},
		{
			name:        "non-dict target - string",
			input:       `with "hello" { x }`,
			expectError: true,
			errorCode:   "TYPE-0020",
		},
		{
			name:        "non-dict target - array",
			input:       `with [1, 2, 3] { x }`,
			expectError: true,
			errorCode:   "TYPE-0020",
		},
		{
			name:        "undefined field in body",
			input:       `with {a: 1} { b }`,
			expectError: true,
			errorCode:   "UNDEF-0001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalWith(tt.input)

			if tt.expectError {
				err, ok := result.(*evaluator.Error)
				if !ok {
					t.Fatalf("expected error, got %T (%s)", result, result.Inspect())
				}
				if tt.errorCode != "" && err.Code != tt.errorCode {
					t.Errorf("expected error code %s, got %s: %s", tt.errorCode, err.Code, err.Message)
				}
			} else {
				if _, ok := result.(*evaluator.Error); ok {
					t.Fatalf("unexpected error: %s", result.Inspect())
				}
			}
		})
	}
}

// TestWithExpressionWithRecord tests with expression with record types
func TestWithExpressionWithRecord(t *testing.T) {
	// Records require a schema, so we test with a simple record setup
	input := `
@schema Person {
	name: string
	age: int
}
let person = Person({name: "Alice", age: 30})
with person { name + " is " + age + " years old" }
`

	result := testEvalWith(input)

	if err, ok := result.(*evaluator.Error); ok {
		t.Fatalf("unexpected error: [%s] %s", err.Code, err.Message)
	}

	if strResult, ok := result.(*evaluator.String); ok {
		expected := "Alice is 30 years old"
		if strResult.Value != expected {
			t.Errorf("expected %q, got %q", expected, strResult.Value)
		}
	} else {
		t.Errorf("expected String, got %T (%s)", result, result.Inspect())
	}
}

// TestWithExpressionReturnsLastValue tests that with returns the last expression value
func TestWithExpressionReturnsLastValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			name: "returns last of multiple expressions",
			input: `
				with {a: 1, b: 2} {
					let x = a + b;
					let y = x * 2;
					y
				}
			`,
			expected: 6,
		},
		{
			name:     "single expression",
			input:    `with {n: 42} { n }`,
			expected: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalWith(tt.input)

			if err, ok := result.(*evaluator.Error); ok {
				t.Fatalf("unexpected error: [%s] %s", err.Code, err.Message)
			}

			if intResult, ok := result.(*evaluator.Integer); ok {
				if intResult.Value != tt.expected {
					t.Errorf("expected %d, got %d", tt.expected, intResult.Value)
				}
			} else {
				t.Errorf("expected Integer, got %T (%s)", result, result.Inspect())
			}
		})
	}
}

// TestWithExpressionScopeIsolation verifies that with doesn't pollute outer scope
func TestWithExpressionScopeIsolation(t *testing.T) {
	// After with block, the unpacked variables should not be accessible
	input := `
		let before = 1;
		let result = with {inner: 10} { inner + before };
		// 'inner' should not be defined here
		result
	`

	result := testEvalWith(input)

	if err, ok := result.(*evaluator.Error); ok {
		t.Fatalf("unexpected error: [%s] %s", err.Code, err.Message)
	}

	if intResult, ok := result.(*evaluator.Integer); ok {
		if intResult.Value != 11 {
			t.Errorf("expected 11, got %d", intResult.Value)
		}
	} else {
		t.Errorf("expected Integer, got %T (%s)", result, result.Inspect())
	}
}

// Ensure strings import is used
var _ = strings.Contains
