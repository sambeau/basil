package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// evalValidTest helper that evaluates Parsley code and handles errors
func evalValidTest(t *testing.T, input string) evaluator.Object {
	t.Helper()
	l := lexer.New(input)
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

	return result
}

// =============================================================================
// Module Import Tests
// =============================================================================

func TestStdValidImport(t *testing.T) {
	input := `let {email} = import @std/valid
email`

	result := evalValidTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if result.Type() != evaluator.BUILTIN_OBJ {
		t.Errorf("expected BUILTIN, got %s", result.Type())
	}
}

func TestStdValidImportAll(t *testing.T) {
	input := `let valid = import @std/valid
valid.email`

	result := evalValidTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if result.Type() != evaluator.BUILTIN_OBJ {
		t.Errorf("expected BUILTIN, got %s", result.Type())
	}
}

// =============================================================================
// Type Validators Tests
// =============================================================================

func TestValidString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"string true", `let valid = import @std/valid; valid.string("hello")`, true},
		{"number false", `let valid = import @std/valid; valid.string(123)`, false},
		{"null false", `let valid = import @std/valid; valid.string(null)`, false},
		{"array false", `let valid = import @std/valid; valid.string([1,2,3])`, false},
		{"empty string true", `let valid = import @std/valid; valid.string("")`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

func TestValidNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"integer true", `let valid = import @std/valid; valid.number(123)`, true},
		{"float true", `let valid = import @std/valid; valid.number(3.14)`, true},
		{"string false", `let valid = import @std/valid; valid.number("123")`, false},
		{"null false", `let valid = import @std/valid; valid.number(null)`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

func TestValidInteger(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"integer true", `let valid = import @std/valid; valid.integer(123)`, true},
		{"float false", `let valid = import @std/valid; valid.integer(3.14)`, false},
		{"string false", `let valid = import @std/valid; valid.integer("123")`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

func TestValidBoolean(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"true true", `let valid = import @std/valid; valid.boolean(true)`, true},
		{"false true", `let valid = import @std/valid; valid.boolean(false)`, true},
		{"string false", `let valid = import @std/valid; valid.boolean("true")`, false},
		{"integer false", `let valid = import @std/valid; valid.boolean(1)`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

func TestValidArray(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"array true", `let valid = import @std/valid; valid.array([1,2,3])`, true},
		{"empty array true", `let valid = import @std/valid; valid.array([])`, true},
		{"string false", `let valid = import @std/valid; valid.array("hello")`, false},
		{"dict false", `let valid = import @std/valid; valid.array({a: 1})`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

func TestValidDict(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"dict true", `let valid = import @std/valid; valid.dict({a: 1})`, true},
		{"empty dict true", `let valid = import @std/valid; valid.dict({})`, true},
		{"array false", `let valid = import @std/valid; valid.dict([1,2,3])`, false},
		{"string false", `let valid = import @std/valid; valid.dict("hello")`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

// =============================================================================
// String Validators Tests
// =============================================================================

func TestValidEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"empty string", `let valid = import @std/valid; valid.empty("")`, true},
		{"whitespace only", `let valid = import @std/valid; valid.empty("   ")`, true},
		{"tab and newline", `let valid = import @std/valid; valid.empty("\t\n")`, true},
		{"non-empty", `let valid = import @std/valid; valid.empty("hello")`, false},
		{"whitespace with text", `let valid = import @std/valid; valid.empty("  a  ")`, false},
		{"non-string returns false", `let valid = import @std/valid; valid.empty(123)`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

func TestValidMinLen(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"at minimum", `let valid = import @std/valid; valid.minLen("hello", 5)`, true},
		{"above minimum", `let valid = import @std/valid; valid.minLen("hello", 3)`, true},
		{"below minimum", `let valid = import @std/valid; valid.minLen("hi", 5)`, false},
		{"empty string", `let valid = import @std/valid; valid.minLen("", 1)`, false},
		{"zero minimum", `let valid = import @std/valid; valid.minLen("", 0)`, true},
		{"unicode", `let valid = import @std/valid; valid.minLen("日本語", 3)`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

func TestValidMaxLen(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"at maximum", `let valid = import @std/valid; valid.maxLen("hello", 5)`, true},
		{"below maximum", `let valid = import @std/valid; valid.maxLen("hi", 5)`, true},
		{"above maximum", `let valid = import @std/valid; valid.maxLen("hello world", 5)`, false},
		{"empty string", `let valid = import @std/valid; valid.maxLen("", 0)`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

func TestValidLength(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"in range", `let valid = import @std/valid; valid.length("hello", 3, 10)`, true},
		{"at min", `let valid = import @std/valid; valid.length("abc", 3, 10)`, true},
		{"at max", `let valid = import @std/valid; valid.length("0123456789", 3, 10)`, true},
		{"below min", `let valid = import @std/valid; valid.length("ab", 3, 10)`, false},
		{"above max", `let valid = import @std/valid; valid.length("01234567890", 3, 10)`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

func TestValidMatches(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"matches", `let valid = import @std/valid; valid.matches("abc123", "^[a-z0-9]+$")`, true},
		{"no match", `let valid = import @std/valid; valid.matches("ABC", "^[a-z]+$")`, false},
		{"partial match", `let valid = import @std/valid; valid.matches("abc123", "[0-9]+")`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

func TestValidMatchesInvalidRegex(t *testing.T) {
	input := `let valid = import @std/valid; valid.matches("test", "[invalid")`

	result := evalValidTest(t, input)

	if result.Type() != evaluator.ERROR_OBJ {
		t.Fatalf("expected error for invalid regex, got %s", result.Type())
	}
}

func TestValidAlpha(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"lowercase", `let valid = import @std/valid; valid.alpha("hello")`, true},
		{"uppercase", `let valid = import @std/valid; valid.alpha("HELLO")`, true},
		{"mixed case", `let valid = import @std/valid; valid.alpha("Hello")`, true},
		{"with numbers", `let valid = import @std/valid; valid.alpha("hello1")`, false},
		{"with space", `let valid = import @std/valid; valid.alpha("hello world")`, false},
		{"empty", `let valid = import @std/valid; valid.alpha("")`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

func TestValidAlphanumeric(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"letters only", `let valid = import @std/valid; valid.alphanumeric("hello")`, true},
		{"numbers only", `let valid = import @std/valid; valid.alphanumeric("123")`, true},
		{"mixed", `let valid = import @std/valid; valid.alphanumeric("abc123")`, true},
		{"with space", `let valid = import @std/valid; valid.alphanumeric("abc 123")`, false},
		{"with special", `let valid = import @std/valid; valid.alphanumeric("abc@123")`, false},
		{"empty", `let valid = import @std/valid; valid.alphanumeric("")`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

func TestValidNumeric(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"integer string", `let valid = import @std/valid; valid.numeric("123")`, true},
		{"float string", `let valid = import @std/valid; valid.numeric("123.45")`, true},
		{"negative", `let valid = import @std/valid; valid.numeric("-123")`, true},
		{"scientific", `let valid = import @std/valid; valid.numeric("1e10")`, true},
		{"not numeric", `let valid = import @std/valid; valid.numeric("abc")`, false},
		{"mixed", `let valid = import @std/valid; valid.numeric("123abc")`, false},
		{"empty", `let valid = import @std/valid; valid.numeric("")`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

// =============================================================================
// Number Validators Tests
// =============================================================================

func TestValidMin(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"at min", `let valid = import @std/valid; valid.min(5, 5)`, true},
		{"above min", `let valid = import @std/valid; valid.min(10, 5)`, true},
		{"below min", `let valid = import @std/valid; valid.min(3, 5)`, false},
		{"negative", `let valid = import @std/valid; valid.min(-3, -5)`, true},
		{"float", `let valid = import @std/valid; valid.min(5.5, 5.0)`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

func TestValidMax(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"at max", `let valid = import @std/valid; valid.max(5, 5)`, true},
		{"below max", `let valid = import @std/valid; valid.max(3, 5)`, true},
		{"above max", `let valid = import @std/valid; valid.max(10, 5)`, false},
		{"negative", `let valid = import @std/valid; valid.max(-10, -5)`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

func TestValidBetween(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"in range", `let valid = import @std/valid; valid.between(5, 1, 10)`, true},
		{"at low", `let valid = import @std/valid; valid.between(1, 1, 10)`, true},
		{"at high", `let valid = import @std/valid; valid.between(10, 1, 10)`, true},
		{"below", `let valid = import @std/valid; valid.between(0, 1, 10)`, false},
		{"above", `let valid = import @std/valid; valid.between(11, 1, 10)`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

func TestValidPositive(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"positive", `let valid = import @std/valid; valid.positive(5)`, true},
		{"zero", `let valid = import @std/valid; valid.positive(0)`, false},
		{"negative", `let valid = import @std/valid; valid.positive(-5)`, false},
		{"positive float", `let valid = import @std/valid; valid.positive(0.001)`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

func TestValidNegative(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"negative", `let valid = import @std/valid; valid.negative(-5)`, true},
		{"zero", `let valid = import @std/valid; valid.negative(0)`, false},
		{"positive", `let valid = import @std/valid; valid.negative(5)`, false},
		{"negative float", `let valid = import @std/valid; valid.negative(-0.001)`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

// =============================================================================
// Format Validators Tests
// =============================================================================

func TestValidEmail(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid simple", `let valid = import @std/valid; valid.email("test@example.com")`, true},
		{"valid with dots", `let valid = import @std/valid; valid.email("test.user@example.com")`, true},
		{"valid with plus", `let valid = import @std/valid; valid.email("test+tag@example.com")`, true},
		{"missing @", `let valid = import @std/valid; valid.email("testexample.com")`, false},
		{"missing domain", `let valid = import @std/valid; valid.email("test@")`, false},
		{"missing local", `let valid = import @std/valid; valid.email("@example.com")`, false},
		{"invalid", `let valid = import @std/valid; valid.email("invalid")`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

func TestValidURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"https", `let valid = import @std/valid; valid.url("https://example.com")`, true},
		{"http", `let valid = import @std/valid; valid.url("http://example.com")`, true},
		{"with path", `let valid = import @std/valid; valid.url("https://example.com/path")`, true},
		{"no protocol", `let valid = import @std/valid; valid.url("example.com")`, false},
		{"ftp", `let valid = import @std/valid; valid.url("ftp://example.com")`, false},
		{"no domain", `let valid = import @std/valid; valid.url("https://")`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

func TestValidUUID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid uuid", `let valid = import @std/valid; valid.uuid("550e8400-e29b-41d4-a716-446655440000")`, true},
		{"valid uuid lowercase", `let valid = import @std/valid; valid.uuid("550e8400-e29b-41d4-a716-446655440000")`, true},
		{"valid uuid uppercase", `let valid = import @std/valid; valid.uuid("550E8400-E29B-41D4-A716-446655440000")`, true},
		{"no dashes", `let valid = import @std/valid; valid.uuid("550e8400e29b41d4a716446655440000")`, false},
		{"too short", `let valid = import @std/valid; valid.uuid("550e8400-e29b-41d4-a716")`, false},
		{"invalid chars", `let valid = import @std/valid; valid.uuid("550e8400-e29b-41d4-a716-44665544000g")`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

func TestValidPhone(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"US format", `let valid = import @std/valid; valid.phone("+1 (555) 123-4567")`, true},
		{"simple", `let valid = import @std/valid; valid.phone("555-123-4567")`, true},
		{"digits only", `let valid = import @std/valid; valid.phone("5551234567")`, true},
		{"international", `let valid = import @std/valid; valid.phone("+44 20 7946 0958")`, true},
		{"too short", `let valid = import @std/valid; valid.phone("123")`, false},
		{"letters", `let valid = import @std/valid; valid.phone("555-CALL-ME")`, false},
		{"empty", `let valid = import @std/valid; valid.phone("")`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

func TestValidCreditCard(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"visa test", `let valid = import @std/valid; valid.creditCard("4111111111111111")`, true},
		{"mastercard test", `let valid = import @std/valid; valid.creditCard("5500000000000004")`, true},
		{"with dashes", `let valid = import @std/valid; valid.creditCard("4111-1111-1111-1111")`, true},
		{"with spaces", `let valid = import @std/valid; valid.creditCard("4111 1111 1111 1111")`, true},
		{"invalid luhn", `let valid = import @std/valid; valid.creditCard("1234567890123456")`, false},
		{"too short", `let valid = import @std/valid; valid.creditCard("411111111111")`, false},
		{"too long", `let valid = import @std/valid; valid.creditCard("41111111111111111111")`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

func TestValidTime(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"24h short", `let valid = import @std/valid; valid.time("14:30")`, true},
		{"24h long", `let valid = import @std/valid; valid.time("14:30:00")`, true},
		{"midnight", `let valid = import @std/valid; valid.time("00:00")`, true},
		{"end of day", `let valid = import @std/valid; valid.time("23:59:59")`, true},
		{"invalid hour", `let valid = import @std/valid; valid.time("25:00")`, false},
		{"invalid minute", `let valid = import @std/valid; valid.time("12:60")`, false},
		{"no colon", `let valid = import @std/valid; valid.time("1430")`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

// =============================================================================
// Date Validators Tests
// =============================================================================

func TestValidDateISO(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid ISO", `let valid = import @std/valid; valid.date("2024-12-25")`, true},
		{"valid leap year", `let valid = import @std/valid; valid.date("2024-02-29")`, true},
		{"invalid Feb 30", `let valid = import @std/valid; valid.date("2024-02-30")`, false},
		{"invalid leap year", `let valid = import @std/valid; valid.date("2023-02-29")`, false},
		{"invalid month", `let valid = import @std/valid; valid.date("2024-13-01")`, false},
		{"invalid day", `let valid = import @std/valid; valid.date("2024-01-32")`, false},
		{"wrong format", `let valid = import @std/valid; valid.date("12/25/2024")`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

func TestValidDateUS(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid US", `let valid = import @std/valid; valid.date("12/25/2024", "US")`, true},
		{"valid US single digit", `let valid = import @std/valid; valid.date("1/5/2024", "US")`, true},
		{"invalid month 25", `let valid = import @std/valid; valid.date("25/12/2024", "US")`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

func TestValidDateGB(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid GB", `let valid = import @std/valid; valid.date("25/12/2024", "GB")`, true},
		{"valid GB single digit", `let valid = import @std/valid; valid.date("5/1/2024", "GB")`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

// =============================================================================
// parseDate Tests
// =============================================================================

func TestValidParseDate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"US to ISO", `let valid = import @std/valid; valid.parseDate("12/25/2024", "US")`, "2024-12-25"},
		{"GB to ISO", `let valid = import @std/valid; valid.parseDate("25/12/2024", "GB")`, "2024-12-25"},
		{"ISO to ISO", `let valid = import @std/valid; valid.parseDate("2024-12-25", "ISO")`, "2024-12-25"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			s := result.(*evaluator.String).Value
			if s != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, s)
			}
		})
	}
}

func TestValidParseDateNull(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"invalid format", `let valid = import @std/valid; valid.parseDate("invalid", "US")`},
		{"invalid date", `let valid = import @std/valid; valid.parseDate("02/30/2024", "US")`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() != evaluator.NULL_OBJ {
				t.Errorf("expected NULL, got %s", result.Type())
			}
		})
	}
}

// =============================================================================
// Postal Code Tests
// =============================================================================

func TestValidPostalCodeUS(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"5 digit", `let valid = import @std/valid; valid.postalCode("90210", "US")`, true},
		{"9 digit", `let valid = import @std/valid; valid.postalCode("90210-1234", "US")`, true},
		{"invalid", `let valid = import @std/valid; valid.postalCode("9021", "US")`, false},
		{"too long", `let valid = import @std/valid; valid.postalCode("902101234", "US")`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

func TestValidPostalCodeGB(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"london", `let valid = import @std/valid; valid.postalCode("SW1A 1AA", "GB")`, true},
		{"manchester", `let valid = import @std/valid; valid.postalCode("M1 1AA", "GB")`, true},
		{"no space", `let valid = import @std/valid; valid.postalCode("SW1A1AA", "GB")`, true},
		{"invalid", `let valid = import @std/valid; valid.postalCode("12345", "GB")`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

func TestValidPostalCodeUnsupportedLocale(t *testing.T) {
	input := `let valid = import @std/valid; valid.postalCode("12345", "DE")`

	result := evalValidTest(t, input)

	if result.Type() != evaluator.ERROR_OBJ {
		t.Fatalf("expected error for unsupported locale, got %s", result.Type())
	}

	errMsg := result.Inspect()
	if !strings.Contains(strings.ToLower(errMsg), "unsupported locale") {
		t.Errorf("expected error about unsupported locale, got: %s", errMsg)
	}
}

// =============================================================================
// Collection Validators Tests
// =============================================================================

func TestValidContains(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"int in array", `let valid = import @std/valid; valid.contains([1, 2, 3], 2)`, true},
		{"string in array", `let valid = import @std/valid; valid.contains(["a", "b", "c"], "b")`, true},
		{"not found", `let valid = import @std/valid; valid.contains([1, 2, 3], 5)`, false},
		{"empty array", `let valid = import @std/valid; valid.contains([], 1)`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

func TestValidOneOf(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"found", `let valid = import @std/valid; valid.oneOf("red", ["red", "green", "blue"])`, true},
		{"not found", `let valid = import @std/valid; valid.oneOf("yellow", ["red", "green", "blue"])`, false},
		{"number", `let valid = import @std/valid; valid.oneOf(2, [1, 2, 3])`, true},
		{"empty options", `let valid = import @std/valid; valid.oneOf("a", [])`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			b := result.(*evaluator.Boolean).Value
			if b != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b)
			}
		})
	}
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestValidArityErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"string no args", `let valid = import @std/valid; valid.string()`},
		{"minLen one arg", `let valid = import @std/valid; valid.minLen("hello")`},
		{"between two args", `let valid = import @std/valid; valid.between(5, 1)`},
		{"postalCode one arg", `let valid = import @std/valid; valid.postalCode("12345")`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() != evaluator.ERROR_OBJ {
				t.Errorf("expected error, got %s", result.Type())
			}
		})
	}
}

func TestValidTypeErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"minLen non-string", `let valid = import @std/valid; valid.minLen(123, 5)`},
		{"min non-number", `let valid = import @std/valid; valid.min("hello", 5)`},
		{"contains non-array", `let valid = import @std/valid; valid.contains("hello", "h")`},
		{"oneOf non-array options", `let valid = import @std/valid; valid.oneOf("a", "abc")`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)

			if result.Type() != evaluator.ERROR_OBJ {
				t.Errorf("expected error, got %s", result.Type())
			}
		})
	}
}
