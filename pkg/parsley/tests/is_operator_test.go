package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/parsley"
)

// evalIsOperatorTestWithError evaluates Parsley code and returns both result and error
func evalIsOperatorTestWithError(input string) (evaluator.Object, error) {
	result, err := parsley.Eval(input)
	if err != nil {
		return nil, err
	}
	return result.Value, nil
}

// =============================================================================
// 'is' Operator Tests - Schema Checking
// =============================================================================

func TestIsOperatorBasic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name: "record is its own schema",
			input: `
@schema User { name: string }
let r = User({name: "Alice"})
r is User`,
			expected: true,
		},
		{
			name: "record is not different schema",
			input: `
@schema User { name: string }
@schema Product { sku: string }
let r = User({name: "Alice"})
r is Product`,
			expected: false,
		},
		{
			name: "is not - record is not different schema",
			input: `
@schema User { name: string }
@schema Product { sku: string }
let r = User({name: "Alice"})
r is not Product`,
			expected: true,
		},
		{
			name: "is not - record is not its own schema",
			input: `
@schema User { name: string }
let r = User({name: "Alice"})
r is not User`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalRecordTest(t, tt.input)
			boolVal, ok := result.(*evaluator.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T", result)
			}
			if boolVal.Value != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, boolVal.Value)
			}
		})
	}
}

func TestIsOperatorWithTables(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name: "typed table is its schema",
			input: `
@schema User { name: string }
let t = User([{name: "Alice"}, {name: "Bob"}])
t is User`,
			expected: true,
		},
		{
			name: "typed table is not different schema",
			input: `
@schema User { name: string }
@schema Product { sku: string }
let t = User([{name: "Alice"}])
t is Product`,
			expected: false,
		},
		{
			name: "untyped table is not any schema",
			input: `
@schema User { name: string }
let t = table([{name: "Alice"}])
t is User`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalRecordTest(t, tt.input)
			boolVal, ok := result.(*evaluator.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T", result)
			}
			if boolVal.Value != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, boolVal.Value)
			}
		})
	}
}

func TestIsOperatorNonRecordValues(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name: "null is schema returns false",
			input: `
@schema User { name: string }
null is User`,
			expected: false,
		},
		{
			name: "string is schema returns false",
			input: `
@schema User { name: string }
"hello" is User`,
			expected: false,
		},
		{
			name: "integer is schema returns false",
			input: `
@schema User { name: string }
42 is User`,
			expected: false,
		},
		{
			name: "plain dict is schema returns false",
			input: `
@schema User { name: string }
{name: "Alice"} is User`,
			expected: false,
		},
		{
			name: "array is schema returns false",
			input: `
@schema User { name: string }
let arr = [1, 2, 3]
arr is User`,
			expected: false,
		},
		{
			name: "boolean is schema returns false",
			input: `
@schema User { name: string }
true is User`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalRecordTest(t, tt.input)
			boolVal, ok := result.(*evaluator.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T", result)
			}
			if boolVal.Value != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, boolVal.Value)
			}
		})
	}
}

func TestIsOperatorSchemaIdentity(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name: "two schemas with same fields are different",
			input: `
@schema User { name: string }
@schema UserCopy { name: string }
let r = User({name: "Alice"})
r is UserCopy`,
			expected: false,
		},
		{
			name: "schema identity is by reference not structure",
			input: `
@schema User { name: string }
@schema UserCopy { name: string }
let r = User({name: "Alice"})
let result = (r is User) && (r is not UserCopy)
result`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalRecordTest(t, tt.input)
			boolVal, ok := result.(*evaluator.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T", result)
			}
			if boolVal.Value != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, boolVal.Value)
			}
		})
	}
}

func TestIsOperatorWithAsMethod(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name: "dict converted with .as() passes is check",
			input: `
@schema User { name: string }
let r = {name: "Alice"}.as(User)
r is User`,
			expected: true,
		},
		{
			name: "table converted with .as() passes is check",
			input: `
@schema User { name: string }
let t = table([{name: "Alice"}]).as(User)
t is User`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalRecordTest(t, tt.input)
			boolVal, ok := result.(*evaluator.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T", result)
			}
			if boolVal.Value != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, boolVal.Value)
			}
		})
	}
}

func TestIsOperatorInExpressions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			name: "is in if condition",
			input: `
@schema User { name: string }
let r = User({name: "Alice"})
if (r is User) { "yes" } else { "no" }`,
			expected: "yes",
		},
		{
			name: "is not in if condition",
			input: `
@schema User { name: string }
@schema Product { sku: string }
let r = User({name: "Alice"})
if (r is not Product) { "correct" } else { "wrong" }`,
			expected: "correct",
		},
		{
			name: "is with && operator",
			input: `
@schema User { name: string }
let r = User({name: "Alice"})
let result = (r is User) && (r.name == "Alice")
result`,
			expected: true,
		},
		{
			name: "is with || operator",
			input: `
@schema User { name: string }
@schema Admin { name: string }
let r = User({name: "Alice"})
let result = (r is User) || (r is Admin)
result`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalRecordTest(t, tt.input)
			switch expected := tt.expected.(type) {
			case string:
				strVal, ok := result.(*evaluator.String)
				if !ok {
					t.Fatalf("expected String, got %T", result)
				}
				if strVal.Value != expected {
					t.Errorf("expected %q, got %q", expected, strVal.Value)
				}
			case bool:
				boolVal, ok := result.(*evaluator.Boolean)
				if !ok {
					t.Fatalf("expected Boolean, got %T", result)
				}
				if boolVal.Value != expected {
					t.Errorf("expected %v, got %v", expected, boolVal.Value)
				}
			}
		})
	}
}

func TestIsOperatorFiltering(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name: "filter array by schema",
			input: `
@schema User { name: string }
@schema Product { sku: string }
let items = [User({name: "Alice"}), Product({sku: "A001"}), User({name: "Bob"})]
items.filter(fn(x) { x is User }).length()`,
			expected: 2,
		},
		{
			name: "filter array by is not",
			input: `
@schema User { name: string }
@schema Product { sku: string }
let items = [User({name: "Alice"}), Product({sku: "A001"}), User({name: "Bob"})]
items.filter(fn(x) { x is not User }).length()`,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalRecordTest(t, tt.input)
			intVal, ok := result.(*evaluator.Integer)
			if !ok {
				t.Fatalf("expected Integer, got %T", result)
			}
			if intVal.Value != int64(tt.expected) {
				t.Errorf("expected %d, got %d", tt.expected, intVal.Value)
			}
		})
	}
}

func TestIsOperatorError(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorMsg    string
	}{
		{
			name: "is with non-schema on right side errors",
			input: `
@schema User { name: string }
let r = User({name: "Alice"})
r is 42`,
			expectError: true,
			errorMsg:    "requires a schema",
		},
		{
			name: "is with string on right side errors",
			input: `
@schema User { name: string }
let r = User({name: "Alice"})
r is "User"`,
			expectError: true,
			errorMsg:    "requires a schema",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evalIsOperatorTestWithError(tt.input)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error, got result: %v", result.Inspect())
				}
				if tt.errorMsg != "" && !isOperatorContains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// isOperatorContains is a helper function for string containment check
func isOperatorContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
