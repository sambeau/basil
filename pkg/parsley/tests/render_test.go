package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
)

func expectString(t *testing.T, obj evaluator.Object) *evaluator.String {
	t.Helper()
	str, ok := obj.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T (%+v)", obj, obj)
	}
	return str
}

func expectError(t *testing.T, obj evaluator.Object) {
	t.Helper()
	if obj == nil {
		t.Fatalf("expected error object, got nil")
	}
	if obj.Type() != evaluator.ERROR_OBJ {
		t.Fatalf("expected error object, got %T (%+v)", obj, obj)
	}
}

func TestStringRenderUsesCurrentScope(t *testing.T) {
	code := `
let color = "blue"
let border = 2
".btn { color: @{color}; border: @{border}px; }".render()
`
	result := testEvalHelper(code)
	str := expectString(t, result)

	expected := ".btn { color: blue; border: 2px; }"
	if str.Value != expected {
		t.Fatalf("expected %q, got %q", expected, str.Value)
	}
}

func TestStringRenderWithDictionary(t *testing.T) {
	code := `
let tpl = "math @{width * 2} chain @{names[0].toUpper()} cond @{if (visible) \"on\" else \"off\"} keep {braces}"
let result = tpl.render({width: 7, names: ["red", "blue"], visible: false})
result
`
	result := testEvalHelper(code)
	str := expectString(t, result)

	expected := "math 14 chain RED cond off keep {braces}"
	if str.Value != expected {
		t.Fatalf("expected %q, got %q", expected, str.Value)
	}
}

func TestStringRenderEscapeAndNested(t *testing.T) {
	code := `
let tpl = "\\@{literal} @{ {a: {b: 3}}.a.b }"
let rendered = tpl.render()
rendered
`
	result := testEvalHelper(code)
	str := expectString(t, result)

	expected := "@{literal} 3"
	if str.Value != expected {
		t.Fatalf("expected %q, got %q", expected, str.Value)
	}
}

func TestDictionaryRenderAndPrintf(t *testing.T) {
	t.Run("dictionary render", func(t *testing.T) {
		code := `
let data = {name: "Ada", age: 30}
data.render("Hello, @{name}! You are @{age}.")
`
		result := testEvalHelper(code)
		str := expectString(t, result)

		expected := "Hello, Ada! You are 30."
		if str.Value != expected {
			t.Fatalf("expected %q, got %q", expected, str.Value)
		}
	})

	t.Run("printf with dictionary scope", func(t *testing.T) {
		code := `
let factor = 2
printf("size @{size}", {size: factor * 5})
`
		result := testEvalHelper(code)
		str := expectString(t, result)

		expected := "size 10"
		if str.Value != expected {
			t.Fatalf("expected %q, got %q", expected, str.Value)
		}
	})
}

func TestRenderErrors(t *testing.T) {
	tests := []string{
		`"oops".render(1, 2)`,
		`"oops".render(123)`,
		`printf("hi", "not-a-dict")`,
		`{a: 1}.render(5)`,
		`"@{missing".render()`,
	}

	for _, code := range tests {
		t.Run(code, func(t *testing.T) {
			result := testEvalHelper(code)
			expectError(t, result)
		})
	}
}
