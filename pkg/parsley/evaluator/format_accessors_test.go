package evaluator

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/lexer"
)

// Helper to create object literal expression from object
func objExpr(obj Object) ast.Expression {
	return &ast.ObjectLiteralExpression{Obj: obj}
}

func TestFormatInteger(t *testing.T) {
	obj := &Integer{Value: 42}
	result := FormatObject(obj)
	if result != "42" {
		t.Errorf("Expected '42', got '%s'", result)
	}
}

func TestFormatNegativeInteger(t *testing.T) {
	obj := &Integer{Value: -123}
	result := FormatObject(obj)
	if result != "-123" {
		t.Errorf("Expected '-123', got '%s'", result)
	}
}

func TestFormatFloat(t *testing.T) {
	obj := &Float{Value: 3.14}
	result := FormatObject(obj)
	if result != "3.14" {
		t.Errorf("Expected '3.14', got '%s'", result)
	}
}

func TestFormatBoolean(t *testing.T) {
	tests := []struct {
		value    bool
		expected string
	}{
		{true, "true"},
		{false, "false"},
	}
	for _, tt := range tests {
		obj := &Boolean{Value: tt.value}
		result := FormatObject(obj)
		if result != tt.expected {
			t.Errorf("Expected '%s', got '%s'", tt.expected, result)
		}
	}
}

func TestFormatString(t *testing.T) {
	obj := &String{Value: "hello"}
	result := FormatObject(obj)
	expected := `"hello"`
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestFormatStringWithQuotes(t *testing.T) {
	obj := &String{Value: `hello "world"`}
	result := FormatObject(obj)
	expected := `"hello \"world\""`
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestFormatNull(t *testing.T) {
	obj := &Null{}
	result := FormatObject(obj)
	if result != "null" {
		t.Errorf("Expected 'null', got '%s'", result)
	}
}

func TestFormatNilObject(t *testing.T) {
	result := FormatObject(nil)
	if result != "null" {
		t.Errorf("Expected 'null', got '%s'", result)
	}
}

func TestFormatEmptyArray(t *testing.T) {
	obj := &Array{Elements: []Object{}}
	result := FormatObject(obj)
	if result != "[]" {
		t.Errorf("Expected '[]', got '%s'", result)
	}
}

func TestFormatSimpleArray(t *testing.T) {
	obj := &Array{
		Elements: []Object{
			&Integer{Value: 1},
			&Integer{Value: 2},
			&Integer{Value: 3},
		},
	}
	result := FormatObject(obj)
	expected := "[1, 2, 3]"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestFormatArrayWithStrings(t *testing.T) {
	obj := &Array{
		Elements: []Object{
			&String{Value: "a"},
			&String{Value: "b"},
		},
	}
	result := FormatObject(obj)
	expected := `["a", "b"]`
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestFormatLongArrayMultiline(t *testing.T) {
	// Create array that exceeds threshold (60 chars)
	obj := &Array{
		Elements: []Object{
			&String{Value: "this is a fairly long string"},
			&String{Value: "and another fairly long string"},
		},
	}
	result := FormatObject(obj)
	// Should be multiline
	if !strings.Contains(result, "\n") {
		t.Errorf("Expected multiline output for long array, got: '%s'", result)
	}
	// Should have trailing comma
	if !strings.Contains(result, ",\n") {
		t.Errorf("Expected trailing comma in multiline array, got: '%s'", result)
	}
}

func TestFormatEmptyDictionary(t *testing.T) {
	obj := &Dictionary{
		Pairs:    map[string]ast.Expression{},
		KeyOrder: []string{},
	}
	result := FormatObject(obj)
	if result != "{}" {
		t.Errorf("Expected '{}', got '%s'", result)
	}
}

func TestFormatSimpleDictionary(t *testing.T) {
	obj := &Dictionary{
		Pairs: map[string]ast.Expression{
			"a": objExpr(&Integer{Value: 1}),
			"b": objExpr(&Integer{Value: 2}),
		},
		KeyOrder: []string{"a", "b"},
	}
	result := FormatObject(obj)
	expected := "{a: 1, b: 2}"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestFormatDictionaryWithStringValue(t *testing.T) {
	obj := &Dictionary{
		Pairs: map[string]ast.Expression{
			"name": objExpr(&String{Value: "Alice"}),
		},
		KeyOrder: []string{"name"},
	}
	result := FormatObject(obj)
	expected := `{name: "Alice"}`
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestFormatDictionaryWithQuotedKey(t *testing.T) {
	obj := &Dictionary{
		Pairs: map[string]ast.Expression{
			"foo-bar": objExpr(&Integer{Value: 1}),
		},
		KeyOrder: []string{"foo-bar"},
	}
	result := FormatObject(obj)
	expected := `{"foo-bar": 1}`
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestFormatNestedArray(t *testing.T) {
	inner := &Array{
		Elements: []Object{
			&Integer{Value: 1},
			&Integer{Value: 2},
		},
	}
	obj := &Array{
		Elements: []Object{
			inner,
			&Integer{Value: 3},
		},
	}
	result := FormatObject(obj)
	expected := "[[1, 2], 3]"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestFormatFunction(t *testing.T) {
	fn := &Function{
		Params: []*ast.FunctionParameter{
			{Ident: &ast.Identifier{Value: "x"}},
		},
		Body: &ast.BlockStatement{
			Statements: []ast.Statement{
				&ast.ExpressionStatement{
					Expression: &ast.InfixExpression{
						Left:     &ast.Identifier{Value: "x"},
						Operator: "*",
						Right:    &ast.IntegerLiteral{Value: 2, Token: lexer.Token{Literal: "2"}},
					},
				},
			},
		},
	}
	result := FormatObject(fn)
	// The body string representation will be "(x * 2)" from the AST
	if !strings.Contains(result, "fn(x)") {
		t.Errorf("Expected function format to contain 'fn(x)', got: '%s'", result)
	}
	if !strings.Contains(result, "*") || !strings.Contains(result, "2") {
		t.Errorf("Expected function format to contain body, got: '%s'", result)
	}
}

func TestFormatMultiParamFunction(t *testing.T) {
	fn := &Function{
		Params: []*ast.FunctionParameter{
			{Ident: &ast.Identifier{Value: "a"}},
			{Ident: &ast.Identifier{Value: "b"}},
			{Ident: &ast.Identifier{Value: "c"}},
		},
		Body: &ast.BlockStatement{
			Statements: []ast.Statement{
				&ast.ExpressionStatement{
					Expression: &ast.Identifier{Value: "a"},
				},
			},
		},
	}
	result := FormatObject(fn)
	if !strings.Contains(result, "fn(a, b, c)") {
		t.Errorf("Expected function format to contain 'fn(a, b, c)', got: '%s'", result)
	}
}
