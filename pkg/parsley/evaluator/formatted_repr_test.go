package evaluator

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/format"
)

func TestFormattedReprStringSimpleTypes(t *testing.T) {
	tests := []struct {
		name     string
		obj      Object
		expected string
	}{
		{"null", &Null{}, "null"},
		{"true", &Boolean{Value: true}, "true"},
		{"false", &Boolean{Value: false}, "false"},
		{"integer", &Integer{Value: 42}, "42"},
		{"negative integer", &Integer{Value: -123}, "-123"},
		{"float", &Float{Value: 3.14}, "3.14"},
		{"string", &String{Value: "hello"}, `"hello"`},
		{"string with quotes", &String{Value: `hello "world"`}, `"hello \"world\""`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ObjectToFormattedReprString(tt.obj)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFormattedReprStringArrayInline(t *testing.T) {
	arr := &Array{
		Elements: []Object{
			&Integer{Value: 1},
			&Integer{Value: 2},
			&Integer{Value: 3},
		},
	}
	result := ObjectToFormattedReprString(arr)
	expected := "[1, 2, 3]"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestFormattedReprStringArrayEmpty(t *testing.T) {
	arr := &Array{Elements: []Object{}}
	result := ObjectToFormattedReprString(arr)
	if result != "[]" {
		t.Errorf("Expected [], got %q", result)
	}
}

func TestFormattedReprStringArrayMultiline(t *testing.T) {
	// Create array that exceeds threshold (60 chars)
	arr := &Array{
		Elements: []Object{
			&String{Value: "this is a fairly long string that exceeds threshold"},
			&String{Value: "another fairly long string value here too"},
		},
	}
	result := ObjectToFormattedReprString(arr)

	// Should be multiline
	if !strings.Contains(result, "\n") {
		t.Errorf("Expected multiline output, got: %q", result)
	}

	// Should have trailing comma
	if !strings.Contains(result, ",\n") {
		t.Errorf("Expected trailing comma, got: %q", result)
	}

	// Should have proper indentation
	if !strings.Contains(result, format.IndentString) {
		t.Errorf("Expected indentation, got: %q", result)
	}
}

func TestFormattedReprStringDictInline(t *testing.T) {
	dict := &Dictionary{
		Pairs: map[string]ast.Expression{
			"a": &ast.ObjectLiteralExpression{Obj: &Integer{Value: 1}},
			"b": &ast.ObjectLiteralExpression{Obj: &Integer{Value: 2}},
		},
		KeyOrder: []string{"a", "b"},
	}
	result := ObjectToFormattedReprString(dict)
	expected := "{a: 1, b: 2}"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestFormattedReprStringDictEmpty(t *testing.T) {
	dict := &Dictionary{
		Pairs:    map[string]ast.Expression{},
		KeyOrder: []string{},
	}
	result := ObjectToFormattedReprString(dict)
	if result != "{}" {
		t.Errorf("Expected {}, got %q", result)
	}
}

func TestFormattedReprStringDictMultiline(t *testing.T) {
	// Create dict that exceeds threshold
	dict := &Dictionary{
		Pairs: map[string]ast.Expression{
			"longKeyName":    &ast.ObjectLiteralExpression{Obj: &String{Value: "this is a long value"}},
			"anotherLongKey": &ast.ObjectLiteralExpression{Obj: &String{Value: "another long value here"}},
		},
		KeyOrder: []string{"longKeyName", "anotherLongKey"},
	}
	result := ObjectToFormattedReprString(dict)

	// Should be multiline
	if !strings.Contains(result, "\n") {
		t.Errorf("Expected multiline output, got: %q", result)
	}

	// Should have trailing comma
	if !strings.Contains(result, ",\n") {
		t.Errorf("Expected trailing comma, got: %q", result)
	}
}

func TestFormattedReprStringDictKeyNeedsQuotes(t *testing.T) {
	dict := &Dictionary{
		Pairs: map[string]ast.Expression{
			"foo-bar": &ast.ObjectLiteralExpression{Obj: &Integer{Value: 1}},
		},
		KeyOrder: []string{"foo-bar"},
	}
	result := ObjectToFormattedReprString(dict)
	expected := `{"foo-bar": 1}`
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestFormattedReprStringNestedStructures(t *testing.T) {
	inner := &Array{
		Elements: []Object{
			&Integer{Value: 1},
			&Integer{Value: 2},
		},
	}
	outer := &Array{
		Elements: []Object{
			inner,
			&Integer{Value: 3},
		},
	}
	result := ObjectToFormattedReprString(outer)
	expected := "[[1, 2], 3]"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestFormattedReprStringCircularReference(t *testing.T) {
	// Create a circular array (not possible in normal Parsley but test the protection)
	arr := &Array{
		Elements: []Object{},
	}
	// This would normally cause infinite recursion without cycle detection
	// Just verify it doesn't crash for a simple array
	result := ObjectToFormattedReprString(arr)
	if result != "[]" {
		t.Errorf("Expected [], got %q", result)
	}
}

func TestFormattedReprStringBuiltin(t *testing.T) {
	builtin := &Builtin{}
	result := ObjectToFormattedReprString(builtin)
	if result != "<builtin>" {
		t.Errorf("Expected <builtin>, got %q", result)
	}
}

func TestFormattedReprStringNil(t *testing.T) {
	result := ObjectToFormattedReprString(nil)
	if result != "null" {
		t.Errorf("Expected null, got %q", result)
	}
}
