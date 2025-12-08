package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// TestFunctionKeywordAlias tests that 'function' is accepted as an alias for 'fn'
func TestFunctionKeywordAlias(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			name:     "basic_function_keyword",
			input:    `let double = function(x) { x * 2 }; double(5)`,
			expected: 10,
		},
		{
			name:     "function_with_multiple_params",
			input:    `let add = function(a, b) { a + b }; add(3, 4)`,
			expected: 7,
		},
		{
			name:     "anonymous_function_call",
			input:    `(function(x) { x + 1 })(10)`,
			expected: 11,
		},
		{
			name:     "function_in_dictionary",
			input:    `let obj = { square: function(n) { n * n } }; obj.square(6)`,
			expected: 36,
		},
		{
			name:     "nested_function",
			input:    `let outer = function(x) { let inner = function(y) { y * 2 }; inner(x) + 1 }; outer(5)`,
			expected: 11,
		},
		{
			name:     "function_as_callback",
			input:    `let apply = fn(f, x) { f(x) }; apply(function(n) { n * 3 }, 7)`,
			expected: 21,
		},
		{
			name:     "fn_and_function_mixed",
			input:    `let a = fn(x) { x + 1 }; let b = function(x) { x * 2 }; a(b(5))`,
			expected: 11,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)

			if result == nil {
				t.Fatal("Eval returned nil")
			}

			if errObj, ok := result.(*evaluator.Error); ok {
				t.Fatalf("Eval returned error: %s", errObj.Message)
			}

			intResult, ok := result.(*evaluator.Integer)
			if !ok {
				t.Fatalf("expected Integer, got %T (%s)", result, result.Inspect())
			}

			if intResult.Value != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, intResult.Value)
			}
		})
	}
}

// TestStdlibPathLexer tests that @std/ is recognized as STDLIB_PATH token
func TestStdlibPathLexer(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedType  lexer.TokenType
		expectedValue string
	}{
		{
			name:          "stdlib_table",
			input:         "@std/table",
			expectedType:  lexer.STDLIB_PATH,
			expectedValue: "std/table", // Lexer strips the @ prefix
		},
		{
			name:          "stdlib_string",
			input:         "@std/string",
			expectedType:  lexer.STDLIB_PATH,
			expectedValue: "std/string",
		},
		{
			name:          "stdlib_with_path",
			input:         "@std/collections/list",
			expectedType:  lexer.STDLIB_PATH,
			expectedValue: "std/collections/list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			tok := l.NextToken()

			if tok.Type != tt.expectedType {
				t.Errorf("expected token type %s, got %s", tt.expectedType, tok.Type)
			}

			if tok.Literal != tt.expectedValue {
				t.Errorf("expected literal %q, got %q", tt.expectedValue, tok.Literal)
			}
		})
	}
}

// TestStdlibPathVsRegularPath tests that @std/ is distinguished from regular paths
func TestStdlibPathVsRegularPath(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedType lexer.TokenType
	}{
		{
			name:         "stdlib_path",
			input:        "@std/table",
			expectedType: lexer.STDLIB_PATH,
		},
		{
			name:         "relative_path",
			input:        "@./utils/helper.pars",
			expectedType: lexer.PATH_LITERAL,
		},
		{
			name:         "home_path",
			input:        "@~/config.pars",
			expectedType: lexer.PATH_LITERAL,
		},
		{
			name:         "absolute_path",
			input:        "@/usr/local/lib",
			expectedType: lexer.PATH_LITERAL,
		},
		{
			name:         "url_path",
			input:        "@https://example.com/api",
			expectedType: lexer.URL_LITERAL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			tok := l.NextToken()

			if tok.Type != tt.expectedType {
				t.Errorf("expected token type %s, got %s (literal: %q)", tt.expectedType, tok.Type, tok.Literal)
			}
		})
	}
}

// TestStdlibImportSyntax tests the full import @std/... syntax is parsed correctly
func TestStdlibImportSyntax(t *testing.T) {
	// Test that import @std/... parses without error
	// (Actual module loading would require stdlib modules to exist)
	inputs := []string{
		`import @std/table`,
		`let t = import @std/table`,
		`let {filter, map} = import @std/collections`,
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			l := lexer.New(input)
			p := parser.New(l)
			program := p.ParseProgram()

			// Should parse without errors (module not found is a runtime error)
			if len(p.Errors()) > 0 {
				for _, err := range p.Errors() {
					// "unexpected token" is a parse error, but "module not found" would be OK
					if err != "" {
						t.Errorf("parser error: %s", err)
					}
				}
			}

			if program == nil {
				t.Fatal("ParseProgram returned nil")
			}
		})
	}
}
