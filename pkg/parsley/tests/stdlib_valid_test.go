package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// evalValidTest is a helper for evaluating code that uses @std/valid
func evalValidTest(t *testing.T, input string) evaluator.Object {
	t.Helper()
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	env := evaluator.NewEnvironment()
	result := evaluator.Eval(program, env)
	return result
}

// =============================================================================
// Module Import Tests
// =============================================================================

func TestStdValidImport(t *testing.T) {
	input := `
		import @std/valid
		valid.uuid("550e8400-e29b-41d4-a716-446655440000")
	`
	result := evalValidTest(t, input)
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("unexpected error: %s", result.Inspect())
	}
	if result.Inspect() != "true" {
		t.Errorf("expected true, got %s", result.Inspect())
	}
}

func TestStdValidImportAll(t *testing.T) {
	input := `
		let { uuid, ulid, nanoid, cuid, creditCard, luhn, postalCode } = import @std/valid
		uuid("550e8400-e29b-41d4-a716-446655440000")
	`
	result := evalValidTest(t, input)
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("unexpected error: %s", result.Inspect())
	}
	if result.Inspect() != "true" {
		t.Errorf("expected true, got %s", result.Inspect())
	}
}

// =============================================================================
// ID Validators
// =============================================================================

func TestValidUUID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"valid_v4", `import @std/valid; valid.uuid("550e8400-e29b-41d4-a716-446655440000")`, "true"},
		{"valid_uppercase", `import @std/valid; valid.uuid("550E8400-E29B-41D4-A716-446655440000")`, "true"},
		{"valid_mixed", `import @std/valid; valid.uuid("550e8400-E29B-41d4-A716-446655440000")`, "true"},
		{"invalid_no_dashes", `import @std/valid; valid.uuid("550e8400e29b41d4a716446655440000")`, "false"},
		{"invalid_short", `import @std/valid; valid.uuid("550e8400-e29b-41d4-a716")`, "false"},
		{"invalid_chars", `import @std/valid; valid.uuid("550e8400-e29b-41d4-a716-44665544000g")`, "false"},
		{"empty", `import @std/valid; valid.uuid("")`, "false"},
		{"non_string", `import @std/valid; valid.uuid(123)`, "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

func TestValidULID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"valid", `import @std/valid; valid.ulid("01ARZ3NDEKTSV4RRFFQ69G5FAV")`, "true"},
		{"valid_all_zeros", `import @std/valid; valid.ulid("00000000000000000000000000")`, "true"},
		{"invalid_lowercase", `import @std/valid; valid.ulid("01arz3ndektsv4rrffq69g5fav")`, "false"},
		{"invalid_short", `import @std/valid; valid.ulid("01ARZ3NDEKTSV4RRFFQ69G5FA")`, "false"},
		{"invalid_long", `import @std/valid; valid.ulid("01ARZ3NDEKTSV4RRFFQ69G5FAVX")`, "false"},
		{"invalid_chars_i", `import @std/valid; valid.ulid("01ARZ3NDEKTSV4RRFFQ69G5FAI")`, "false"}, // I not allowed
		{"invalid_chars_l", `import @std/valid; valid.ulid("01ARZ3NDEKTSV4RRFFQ69G5FAL")`, "false"}, // L not allowed
		{"invalid_chars_o", `import @std/valid; valid.ulid("01ARZ3NDEKTSV4RRFFQ69G5FAO")`, "false"}, // O not allowed
		{"invalid_chars_u", `import @std/valid; valid.ulid("01ARZ3NDEKTSV4RRFFQ69G5FAU")`, "false"}, // U not allowed
		{"empty", `import @std/valid; valid.ulid("")`, "false"},
		{"non_string", `import @std/valid; valid.ulid(123)`, "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

func TestValidNanoID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"valid_default_len", `import @std/valid; valid.nanoid("V1StGXR8_Z5jdHi6B-myT")`, "true"},
		{"valid_custom_len", `import @std/valid; valid.nanoid("abc", 3)`, "true"},
		{"valid_with_underscore", `import @std/valid; valid.nanoid("abc_def-ghi1234567890", 21)`, "true"},
		{"invalid_wrong_len", `import @std/valid; valid.nanoid("ab", 3)`, "false"},
		{"invalid_default_len_short", `import @std/valid; valid.nanoid("V1StGXR8_Z5jdHi6B-my")`, "false"},
		{"invalid_chars", `import @std/valid; valid.nanoid("V1StGXR8_Z5jdHi6B-my!")`, "false"},
		{"empty", `import @std/valid; valid.nanoid("")`, "false"},
		{"non_string", `import @std/valid; valid.nanoid(123)`, "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

func TestValidCUID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"valid", `import @std/valid; valid.cuid("cjld2cjxh0000qzrmn831i7rn")`, "true"},
		{"valid_all_zeros", `import @std/valid; valid.cuid("c000000000000000000000000")`, "true"},
		{"invalid_no_c_prefix", `import @std/valid; valid.cuid("xjld2cjxh0000qzrmn831i7rn")`, "false"},
		{"invalid_uppercase", `import @std/valid; valid.cuid("CJLD2CJXH0000QZRMN831I7RN")`, "false"},
		{"invalid_short", `import @std/valid; valid.cuid("cjld2cjxh0000qzrmn831i7r")`, "false"},
		{"invalid_long", `import @std/valid; valid.cuid("cjld2cjxh0000qzrmn831i7rnx")`, "false"},
		{"empty", `import @std/valid; valid.cuid("")`, "false"},
		{"non_string", `import @std/valid; valid.cuid(123)`, "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

// =============================================================================
// Financial Validators
// =============================================================================

func TestValidCreditCard(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"valid_visa", `import @std/valid; valid.creditCard("4111111111111111")`, "true"},
		{"valid_mastercard", `import @std/valid; valid.creditCard("5500000000000004")`, "true"},
		{"valid_amex", `import @std/valid; valid.creditCard("340000000000009")`, "true"},
		{"valid_with_spaces", `import @std/valid; valid.creditCard("4111 1111 1111 1111")`, "true"},
		{"valid_with_dashes", `import @std/valid; valid.creditCard("4111-1111-1111-1111")`, "true"},
		{"invalid_luhn", `import @std/valid; valid.creditCard("4111111111111112")`, "false"},
		{"invalid_short", `import @std/valid; valid.creditCard("411111111111")`, "false"},
		{"invalid_long", `import @std/valid; valid.creditCard("41111111111111111111")`, "false"},
		{"empty", `import @std/valid; valid.creditCard("")`, "false"},
		{"non_string", `import @std/valid; valid.creditCard(123)`, "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

func TestValidLuhn(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"valid_simple", `import @std/valid; valid.luhn("79927398713")`, "true"},
		{"valid_single_digit", `import @std/valid; valid.luhn("0")`, "true"},
		{"valid_credit_card", `import @std/valid; valid.luhn("4111111111111111")`, "true"},
		{"invalid", `import @std/valid; valid.luhn("79927398710")`, "false"},
		{"empty", `import @std/valid; valid.luhn("")`, "false"},
		{"non_string", `import @std/valid; valid.luhn(123)`, "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

// =============================================================================
// Locale-Aware Validators
// =============================================================================

func TestValidPostalCodeUS(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"valid_5_digit", `import @std/valid; valid.postalCode("12345", "US")`, "true"},
		{"valid_9_digit", `import @std/valid; valid.postalCode("12345-6789", "US")`, "true"},
		{"invalid_4_digit", `import @std/valid; valid.postalCode("1234", "US")`, "false"},
		{"invalid_letters", `import @std/valid; valid.postalCode("1234A", "US")`, "false"},
		{"invalid_format", `import @std/valid; valid.postalCode("123456789", "US")`, "false"},
		{"empty", `import @std/valid; valid.postalCode("", "US")`, "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

func TestValidPostalCodeGB(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"valid_london", `import @std/valid; valid.postalCode("SW1A 1AA", "GB")`, "true"},
		{"valid_no_space", `import @std/valid; valid.postalCode("SW1A1AA", "GB")`, "true"},
		{"valid_edinburgh", `import @std/valid; valid.postalCode("EH1 1AA", "GB")`, "true"},
		{"valid_lowercase", `import @std/valid; valid.postalCode("sw1a 1aa", "GB")`, "true"},
		{"invalid_format", `import @std/valid; valid.postalCode("12345", "GB")`, "false"},
		{"empty", `import @std/valid; valid.postalCode("", "GB")`, "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

func TestValidPostalCodeUnsupportedLocale(t *testing.T) {
	input := `import @std/valid; valid.postalCode("12345", "XX")`
	result := evalValidTest(t, input)
	if result.Type() != evaluator.ERROR_OBJ {
		t.Fatalf("expected error for unsupported locale, got %s", result.Inspect())
	}
	if !strings.Contains(result.Inspect(), "unsupported") && !strings.Contains(result.Inspect(), "locale") {
		t.Errorf("expected error about unsupported locale, got: %s", result.Inspect())
	}
}

// =============================================================================
// Arity Errors
// =============================================================================

func TestValidArityErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"uuid_no_args", `import @std/valid; valid.uuid()`},
		{"uuid_too_many", `import @std/valid; valid.uuid("a", "b")`},
		{"ulid_no_args", `import @std/valid; valid.ulid()`},
		{"ulid_too_many", `import @std/valid; valid.ulid("a", "b")`},
		{"nanoid_no_args", `import @std/valid; valid.nanoid()`},
		{"nanoid_too_many", `import @std/valid; valid.nanoid("a", 21, "extra")`},
		{"cuid_no_args", `import @std/valid; valid.cuid()`},
		{"cuid_too_many", `import @std/valid; valid.cuid("a", "b")`},
		{"creditCard_no_args", `import @std/valid; valid.creditCard()`},
		{"luhn_no_args", `import @std/valid; valid.luhn()`},
		{"postalCode_no_args", `import @std/valid; valid.postalCode()`},
		{"postalCode_one_arg", `import @std/valid; valid.postalCode("12345")`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)
			if result.Type() != evaluator.ERROR_OBJ {
				t.Errorf("expected arity error, got %s", result.Inspect())
			}
		})
	}
}

// =============================================================================
// Type Errors
// =============================================================================

func TestValidTypeErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"nanoid_bad_length_type", `import @std/valid; valid.nanoid("abc", "not a number")`},
		{"postalCode_bad_locale_type", `import @std/valid; valid.postalCode("12345", 123)`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalValidTest(t, tt.input)
			if result.Type() != evaluator.ERROR_OBJ {
				t.Errorf("expected type error, got %s", result.Inspect())
			}
		})
	}
}
