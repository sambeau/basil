package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func testEvalDict(input string) evaluator.Object {
	l := lexer.NewWithFilename(input, "test.pars")
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return &evaluator.Error{Message: p.Errors()[0]}
	}
	env := evaluator.NewEnvironment()
	return evaluator.Eval(program, env)
}

// TestDictionaryInsertionOrder verifies that dictionaries preserve insertion order
func TestDictionaryInsertionOrder(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Basic insertion order
		{
			`{ a: 1, b: 2, c: 3 }.keys()`,
			`[a, b, c]`,
		},
		{
			`{ z: 1, y: 2, x: 3 }.keys()`,
			`[z, y, x]`,
		},
		{
			`{ c: 1, a: 2, b: 3 }.keys()`,
			`[c, a, b]`,
		},
		// values() preserves order
		{
			`{ z: 1, y: 2, x: 3 }.values()`,
			`[1, 2, 3]`,
		},
		// entries() preserves order
		{
			`{ z: 1, y: 2 }.entries().map(fn (e) { e.key })`,
			`[z, y]`,
		},
		// for-in iteration preserves order
		{
			`d = { c: 3, a: 1, b: 2 }; result = for (k, v in d) { k }; result`,
			`[c, a, b]`,
		},
		// builtin keys() preserves order
		{
			`keys({ third: 3, first: 1, second: 2 })`,
			`[third, first, second]`,
		},
		// builtin values() preserves order
		{
			`values({ z: "last", a: "first", m: "middle" })`,
			`[last, first, middle]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEvalDict(tt.input)
			if evaluated == nil {
				t.Fatalf("evaluation returned nil for: %s", tt.input)
			}
			if err, ok := evaluated.(*evaluator.Error); ok {
				t.Fatalf("evaluation error for %s: %s", tt.input, err.Message)
			}
			result := evaluated.Inspect()
			if result != tt.expected {
				t.Errorf("for %s\nexpected: %s\ngot:      %s", tt.input, tt.expected, result)
			}
		})
	}
}

// TestDictionaryInspectOrder verifies that Inspect() uses insertion order
func TestDictionaryInspectOrder(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			`{ z: 1, a: 2, m: 3 }`,
			`{z: 1, a: 2, m: 3}`,
		},
		{
			`{ third: 3, first: 1, second: 2 }`,
			`{third: 3, first: 1, second: 2}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEvalDict(tt.input)
			if evaluated == nil {
				t.Fatalf("evaluation returned nil for: %s", tt.input)
			}
			if err, ok := evaluated.(*evaluator.Error); ok {
				t.Fatalf("evaluation error for %s: %s", tt.input, err.Message)
			}
			result := evaluated.Inspect()
			if result != tt.expected {
				t.Errorf("for %s\nexpected: %s\ngot:      %s", tt.input, tt.expected, result)
			}
		})
	}
}

// TestDictionaryDeletePreservesOrder verifies that delete doesn't break order
func TestDictionaryDeletePreservesOrder(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Delete middle key, remaining keys stay in order
		{
			`d = { a: 1, b: 2, c: 3 }; d.delete("b"); d.keys()`,
			`[a, c]`,
		},
		// Delete first key
		{
			`d = { z: 1, y: 2, x: 3 }; d.delete("z"); d.keys()`,
			`[y, x]`,
		},
		// Delete last key
		{
			`d = { z: 1, y: 2, x: 3 }; d.delete("x"); d.keys()`,
			`[z, y]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEvalDict(tt.input)
			if evaluated == nil {
				t.Fatalf("evaluation returned nil for: %s", tt.input)
			}
			if err, ok := evaluated.(*evaluator.Error); ok {
				t.Fatalf("evaluation error for %s: %s", tt.input, err.Message)
			}
			result := evaluated.Inspect()
			if result != tt.expected {
				t.Errorf("for %s\nexpected: %s\ngot:      %s", tt.input, tt.expected, result)
			}
		})
	}
}

// TestToArrayToDict preserves order
func TestToArrayToDictPreservesOrder(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// toArray preserves order
		{
			`toArray({ z: 1, a: 2, m: 3 }).map(fn (p) { p[0] })`,
			`[z, a, m]`,
		},
		// toDict from array preserves order
		{
			`toDict([["z", 1], ["a", 2], ["m", 3]]).keys()`,
			`[z, a, m]`,
		},
		// Round-trip preserves order
		{
			`d = { third: 3, first: 1, second: 2 }; toDict(toArray(d)).keys()`,
			`[third, first, second]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEvalDict(tt.input)
			if evaluated == nil {
				t.Fatalf("evaluation returned nil for: %s", tt.input)
			}
			if err, ok := evaluated.(*evaluator.Error); ok {
				t.Fatalf("evaluation error for %s: %s", tt.input, err.Message)
			}
			result := evaluated.Inspect()
			if result != tt.expected {
				t.Errorf("for %s\nexpected: %s\ngot:      %s", tt.input, tt.expected, result)
			}
		})
	}
}
