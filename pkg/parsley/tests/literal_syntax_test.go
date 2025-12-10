package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func testEvalLiteral(input string) evaluator.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	env := evaluator.NewEnvironment()
	return evaluator.Eval(program, env)
}

// Test path literal syntax
func TestPathLiterals(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"let p = @/usr/local/bin; toString(p)", "/usr/local/bin"},
		{"let p = @./config.json; toString(p)", "./config.json"},
		{"let p = @~/documents; toString(p)", "~/documents"},
		{"let p = @/usr/local/bin; p.basename", "bin"},
		{"let p = @/usr/local/bin; toString(p.dirname)", "/usr/local"},
		{"let p = @./config.json; p.extension", "json"},
		{"let p = @./config.json; p.stem", "config"},
	}

	for _, tt := range tests {
		result := testEvalLiteral(tt.input)
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

// Test URL literal syntax
func TestUrlLiterals(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{"let u = @https://example.com/api; toString(u)", "https://example.com/api"},
		{"let u = @http://localhost:8080/test; toString(u)", "http://localhost:8080/test"},
		{"let u = @https://example.com/api; u.scheme", "https"},
		{"let u = @https://example.com:8080/api; u.host", "example.com"},
		{"let u = @https://example.com/api/v1; u.pathname", "/api/v1"},
		{"let u = @https://example.com:8080/api; u.port", int64(8080)},
	}

	for _, tt := range tests {
		result := testEvalLiteral(tt.input)
		switch expected := tt.expected.(type) {
		case string:
			str, ok := result.(*evaluator.String)
			if !ok {
				t.Errorf("For input '%s': expected String, got %T (%+v)", tt.input, result, result)
				continue
			}
			if str.Value != expected {
				t.Errorf("For input '%s': expected '%s', got '%s'", tt.input, expected, str.Value)
			}
		case int64:
			num, ok := result.(*evaluator.Integer)
			if !ok {
				t.Errorf("For input '%s': expected Integer, got %T (%+v)", tt.input, result, result)
				continue
			}
			if num.Value != expected {
				t.Errorf("For input '%s': expected %d, got %d", tt.input, expected, num.Value)
			}
		}
	}
}

// Test URL query parameters with literals
func TestUrlLiteralsWithQuery(t *testing.T) {
	input := `let u = @http://example.com/search?q=hello&lang=en; u.query.q`
	result := testEvalLiteral(input)
	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T (%+v)", result, result)
	}
	if str.Value != "hello" {
		t.Errorf("expected 'hello', got '%s'", str.Value)
	}

	input = `let u = @http://example.com/search?q=hello&lang=en; u.query.lang`
	result = testEvalLiteral(input)
	str, ok = result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T (%+v)", result, result)
	}
	if str.Value != "en" {
		t.Errorf("expected 'en', got '%s'", str.Value)
	}
}

// Test equivalence between literals and constructors
func TestLiteralConstructorEquivalence(t *testing.T) {
	tests := []struct {
		literal     string
		constructor string
	}{
		// path() builtin removed - use @({...}) for dynamic paths
		{"@/usr/local/bin", `@({"/usr/local/bin"})`},
		{"@./config.json", `@({"./config.json"})`},
		{"@https://example.com/api", `url("https://example.com/api")`},
		{"@http://localhost:8080/test", `url("http://localhost:8080/test")`},
	}

	for _, tt := range tests {
		input := "let a = " + tt.literal + "; let b = " + tt.constructor + "; toString(a) == toString(b)"
		result := testEvalLiteral(input)
		boolean, ok := result.(*evaluator.Boolean)
		if !ok {
			t.Errorf("For input '%s': expected Boolean, got %T (%+v)", input, result, result)
			continue
		}
		if !boolean.Value {
			t.Errorf("For input '%s': expected true, got false", input)
		}
	}
}

// Test path literals with computed properties
func TestPathLiteralsComputedProperties(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"let p = @/usr/local/bin/parsley; p.basename", "parsley"},
		{"let p = @/usr/local/bin/parsley; toString(p.dirname)", "/usr/local/bin"},
		{"let p = @/usr/local/bin/file.txt; p.extension", "txt"},
		{"let p = @/usr/local/bin/file.txt; p.stem", "file"},
	}

	for _, tt := range tests {
		result := testEvalLiteral(tt.input)
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

// Test URL literals with computed properties
func TestUrlLiteralsComputedProperties(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"let u = @https://example.com:8080/api; u.origin", "https://example.com:8080"},
		{"let u = @https://example.com/api/v1; u.pathname", "/api/v1"},
	}

	for _, tt := range tests {
		result := testEvalLiteral(tt.input)
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

// Test that @ prefix correctly distinguishes between different literal types
func TestAtPrefixDisambiguation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"datetime", `@2024-12-25`, "datetime"},
		{"duration", `@2h30m`, "duration"},
		{"path", `@/usr/local`, "path"},
		{"url", `@https://example.com`, "url"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalLiteral(tt.input)
			dict, ok := result.(*evaluator.Dictionary)
			if !ok {
				t.Errorf("expected Dictionary, got %T (%+v)", result, result)
				return
			}

			// Check __type field
			typeExpr, ok := dict.Pairs["__type"]
			if !ok {
				t.Errorf("dictionary missing __type field")
				return
			}

			typeVal := evaluator.Eval(typeExpr, dict.Env)
			typeStr, ok := typeVal.(*evaluator.String)
			if !ok {
				t.Errorf("__type is not a string, got %T", typeVal)
				return
			}

			if typeStr.Value != tt.expected {
				t.Errorf("expected __type=%s, got %s", tt.expected, typeStr.Value)
			}
		})
	}
}
