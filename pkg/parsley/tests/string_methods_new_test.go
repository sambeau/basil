package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func evalStringMethodTest(t *testing.T, input string) evaluator.Object {
	t.Helper()
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	env := evaluator.NewEnvironment()
	return evaluator.Eval(program, env)
}

// =============================================================================
// Base64 Encoding/Decoding
// =============================================================================

func TestStringToBase64(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"hello", `"hello".toBase64()`, `aGVsbG8=`},
		{"empty", `"".toBase64()`, ``},
		{"unicode", `"日本語".toBase64()`, `5pel5pys6Kqe`},
		{"with_spaces", `"hello world".toBase64()`, `aGVsbG8gd29ybGQ=`},
		{"special_chars", `"<html>&</html>".toBase64()`, `PGh0bWw+JjwvaHRtbD4=`},
		{"newlines", `"line1\nline2".toBase64()`, `bGluZTEKbGluZTI=`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalStringMethodTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

func TestStringFromBase64(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"hello", `"aGVsbG8=".fromBase64()`, `hello`},
		{"empty", `"".fromBase64()`, ``},
		{"unicode", `"5pel5pys6Kqe".fromBase64()`, `日本語`},
		{"with_spaces", `"aGVsbG8gd29ybGQ=".fromBase64()`, `hello world`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalStringMethodTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

func TestStringFromBase64Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"invalid_chars", `"!!!invalid!!!".fromBase64()`},
		{"bad_padding", `"aGVsbG8".fromBase64()`}, // missing padding
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalStringMethodTest(t, tt.input)
			if result.Type() != evaluator.ERROR_OBJ {
				t.Errorf("expected error for invalid base64, got %s", result.Inspect())
			}
		})
	}
}

func TestBase64RoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", `"hello".toBase64().fromBase64()`, `hello`},
		{"unicode", `"日本語テスト".toBase64().fromBase64()`, `日本語テスト`},
		{"special", `"<script>alert('xss')</script>".toBase64().fromBase64()`, `<script>alert('xss')</script>`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalStringMethodTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("round-trip failed: expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

// =============================================================================
// Case Conversion
// =============================================================================

func TestStringToCamel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"snake_case", `"hello_world".toCamel()`, `helloWorld`},
		{"kebab_case", `"hello-world".toCamel()`, `helloWorld`},
		{"pascal_case", `"HelloWorld".toCamel()`, `helloWorld`},
		{"mixed", `"hello_world-test".toCamel()`, `helloWorldTest`},
		{"already_camel", `"helloWorld".toCamel()`, `helloWorld`},
		{"single_word", `"hello".toCamel()`, `hello`},
		{"uppercase", `"HELLO_WORLD".toCamel()`, `helloWorld`},
		{"empty", `"".toCamel()`, ``},
		{"with_spaces", `"hello world".toCamel()`, `helloWorld`},
		{"multiple_underscores", `"hello__world".toCamel()`, `helloWorld`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalStringMethodTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

func TestStringToPascal(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"snake_case", `"hello_world".toPascal()`, `HelloWorld`},
		{"kebab_case", `"hello-world".toPascal()`, `HelloWorld`},
		{"camel_case", `"helloWorld".toPascal()`, `HelloWorld`},
		{"already_pascal", `"HelloWorld".toPascal()`, `HelloWorld`},
		{"single_word", `"hello".toPascal()`, `Hello`},
		{"uppercase", `"HELLO_WORLD".toPascal()`, `HelloWorld`},
		{"empty", `"".toPascal()`, ``},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalStringMethodTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

func TestStringToSnake(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"camel_case", `"helloWorld".toSnake()`, `hello_world`},
		{"pascal_case", `"HelloWorld".toSnake()`, `hello_world`},
		{"kebab_case", `"hello-world".toSnake()`, `hello_world`},
		{"already_snake", `"hello_world".toSnake()`, `hello_world`},
		{"single_word", `"hello".toSnake()`, `hello`},
		{"uppercase", `"HELLO".toSnake()`, `hello`},
		{"empty", `"".toSnake()`, ``},
		{"with_spaces", `"Hello World".toSnake()`, `hello_world`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalStringMethodTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

func TestStringToKebab(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"camel_case", `"helloWorld".toKebab()`, `hello-world`},
		{"pascal_case", `"HelloWorld".toKebab()`, `hello-world`},
		{"snake_case", `"hello_world".toKebab()`, `hello-world`},
		{"already_kebab", `"hello-world".toKebab()`, `hello-world`},
		{"single_word", `"hello".toKebab()`, `hello`},
		{"uppercase", `"HELLO".toKebab()`, `hello`},
		{"empty", `"".toKebab()`, ``},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalStringMethodTest(t, tt.input)
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
// Truncation
// =============================================================================

func TestStringTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"basic", `"Hello world".truncate(8)`, `Hello...`},
		{"custom_suffix", `"Hello world".truncate(8, "…")`, `Hello w…`},
		{"no_change_shorter", `"Hi".truncate(8)`, `Hi`},
		{"no_change_exact", `"Hello".truncate(5)`, `Hello`},
		{"suffix_fills_space", `"Hello".truncate(3)`, `...`},
		{"empty_suffix", `"Hello world".truncate(8, "")`, `Hello wo`},
		{"empty_string", `"".truncate(5)`, ``},
		{"zero_length", `"Hello".truncate(0)`, ``},
		{"unicode_content", `"こんにちは世界".truncate(5)`, `こん...`},
		{"unicode_suffix", `"Hello world".truncate(7, "→")`, `Hello →`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalStringMethodTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

func TestStringTruncateTypeErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"string_length", `"hello".truncate("5")`},
		{"number_suffix", `"hello".truncate(3, 123)`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalStringMethodTest(t, tt.input)
			if result.Type() != evaluator.ERROR_OBJ {
				t.Errorf("expected type error, got %s", result.Inspect())
			}
		})
	}
}

// =============================================================================
// Case Conversion Integration
// =============================================================================

func TestCaseConversionChaining(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// snake -> camel -> snake should be idempotent (after first conversion)
		{"snake_camel_snake", `"hello_world".toCamel().toSnake()`, `hello_world`},
		// camel -> snake -> camel
		{"camel_snake_camel", `"helloWorld".toSnake().toCamel()`, `helloWorld`},
		// pascal -> kebab -> pascal
		{"pascal_kebab_pascal", `"HelloWorld".toKebab().toPascal()`, `HelloWorld`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalStringMethodTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}
