package pln_test

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
	_ "github.com/sambeau/basil/pkg/parsley/pln" // Register PLN hooks
)

// evalParsley evaluates Parsley code and returns the result
func evalParsley(code string) evaluator.Object {
	l := lexer.New(code)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return &evaluator.Error{Message: p.Errors()[0]}
	}
	env := evaluator.NewEnvironment()
	return evaluator.Eval(program, env)
}

func TestSerializeBuiltin(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"integer", "serialize(42)", "42"},
		{"string", `serialize("hello")`, `"hello"`},
		{"array", "serialize([1, 2, 3])", "[1, 2, 3]"},
		{"dict", "serialize({a: 1})", "{a: 1}"},
		{"bool true", "serialize(true)", "true"},
		{"bool false", "serialize(false)", "false"},
		{"null", "serialize(null)", "null"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalParsley(tt.input)
			if errObj, ok := result.(*evaluator.Error); ok {
				t.Fatalf("unexpected error: %s", errObj.Message)
			}
			strObj, ok := result.(*evaluator.String)
			if !ok {
				t.Fatalf("expected String, got %T", result)
			}
			if strObj.Value != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, strObj.Value)
			}
		})
	}
}

func TestSerializeBuiltinError(t *testing.T) {
	// Serializing a function should fail
	result := evalParsley("serialize(fn(x) { x })")
	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error, got %T", result)
	}
	if !strings.Contains(errObj.Message, "serialize") || !strings.Contains(errObj.Message, "function") {
		t.Errorf("unexpected error message: %s", errObj.Message)
	}
}

func TestDeserializeBuiltin(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		checkFn func(evaluator.Object) bool
	}{
		{
			"integer",
			`deserialize("42")`,
			func(o evaluator.Object) bool {
				i, ok := o.(*evaluator.Integer)
				return ok && i.Value == 42
			},
		},
		{
			"string",
			`deserialize("\"hello\"")`,
			func(o evaluator.Object) bool {
				s, ok := o.(*evaluator.String)
				return ok && s.Value == "hello"
			},
		},
		{
			"array",
			`deserialize("[1, 2, 3]")`,
			func(o evaluator.Object) bool {
				a, ok := o.(*evaluator.Array)
				return ok && len(a.Elements) == 3
			},
		},
		{
			"dict",
			`deserialize("{a: 1}")`,
			func(o evaluator.Object) bool {
				d, ok := o.(*evaluator.Dictionary)
				return ok && len(d.Pairs) == 1
			},
		},
		{
			"bool true",
			`deserialize("true")`,
			func(o evaluator.Object) bool {
				b, ok := o.(*evaluator.Boolean)
				return ok && b.Value == true
			},
		},
		{
			"null",
			`deserialize("null")`,
			func(o evaluator.Object) bool {
				_, ok := o.(*evaluator.Null)
				return ok
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalParsley(tt.input)
			if errObj, ok := result.(*evaluator.Error); ok {
				t.Fatalf("unexpected error: %s", errObj.Message)
			}
			if !tt.checkFn(result) {
				t.Errorf("check failed for result: %v (%T)", result, result)
			}
		})
	}
}

func TestDeserializeBuiltinError(t *testing.T) {
	// Invalid PLN should fail
	result := evalParsley(`deserialize("{invalid")`)
	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error, got %T: %v", result, result)
	}
	// Error could be parse error or contain error code
	if !strings.Contains(errObj.Message, "parse") &&
		!strings.Contains(errObj.Message, "DESERIALIZE") &&
		!strings.Contains(errObj.Message, "expected") {
		t.Errorf("unexpected error message: %s", errObj.Message)
	}
}

func TestRoundTripBuiltins(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{"integer", "let x = 42; deserialize(serialize(x)) == x"},
		{"string", `let x = "hello"; deserialize(serialize(x)) == x`},
		{"bool true", `deserialize(serialize(true))`},   // Check value directly
		{"bool false", `deserialize(serialize(false))`}, // Check value directly
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalParsley(tt.code)
			if errObj, ok := result.(*evaluator.Error); ok {
				t.Fatalf("unexpected error: %s", errObj.Message)
			}

			// For bool tests, check the deserialized value directly
			if strings.HasPrefix(tt.name, "bool") {
				boolObj, ok := result.(*evaluator.Boolean)
				if !ok {
					t.Fatalf("expected Boolean, got %T: %v", result, result)
				}
				expectedValue := tt.name == "bool true"
				if boolObj.Value != expectedValue {
					t.Errorf("expected %v, got %v", expectedValue, boolObj.Value)
				}
				return
			}

			// For comparison tests, check for true result
			boolObj, ok := result.(*evaluator.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T: %v", result, result)
			}
			if !boolObj.Value {
				t.Error("round-trip comparison failed")
			}
		})
	}
}
