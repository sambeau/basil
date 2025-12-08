package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func evalTableInsert(input string) evaluator.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		return &evaluator.Error{Message: p.Errors()[0]}
	}

	env := evaluator.NewEnvironment()
	return evaluator.Eval(program, env)
}

// TestTableAppendRow tests the table appendRow method
func TestTableAppendRow(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "append row to table",
			input: `
				let {table} = import @std/table
				let t = table([{a: 1, b: 2}, {a: 3, b: 4}])
				t.appendRow({a: 5, b: 6}).count()
			`,
			expected: "3",
		},
		{
			name: "append row - verify data",
			input: `
				let {table} = import @std/table
				let t = table([{name: "Alice"}])
				let t2 = t.appendRow({name: "Bob"})
				t2.rows[1].name
			`,
			expected: "Bob",
		},
		{
			name: "append to empty table",
			input: `
				let {table} = import @std/table
				let t = table([])
				t.appendRow({x: 1, y: 2}).count()
			`,
			expected: "1",
		},
		{
			name: "original table unchanged",
			input: `
				let {table} = import @std/table
				let t = table([{a: 1}])
				let _ = t.appendRow({a: 2})
				t.count()
			`,
			expected: "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalTableInsert(tt.input)
			if result.Inspect() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.Inspect())
			}
		})
	}
}

// TestTableInsertRowAt tests the table insertRowAt method
func TestTableInsertRowAt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "insert at beginning",
			input: `
				let {table} = import @std/table
				let t = table([{name: "B"}, {name: "C"}])
				t.insertRowAt(0, {name: "A"}).rows[0].name
			`,
			expected: "A",
		},
		{
			name: "insert in middle",
			input: `
				let {table} = import @std/table
				let t = table([{name: "A"}, {name: "C"}])
				t.insertRowAt(1, {name: "B"}).rows[1].name
			`,
			expected: "B",
		},
		{
			name: "insert at end (same as append)",
			input: `
				let {table} = import @std/table
				let t = table([{name: "A"}, {name: "B"}])
				t.insertRowAt(2, {name: "C"}).rows[2].name
			`,
			expected: "C",
		},
		{
			name: "negative index",
			input: `
				let {table} = import @std/table
				let t = table([{name: "A"}, {name: "C"}])
				t.insertRowAt(-1, {name: "B"}).rows[1].name
			`,
			expected: "B",
		},
		{
			name: "verify count after insert",
			input: `
				let {table} = import @std/table
				let t = table([{a: 1}, {a: 2}])
				t.insertRowAt(1, {a: 99}).count()
			`,
			expected: "3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalTableInsert(tt.input)
			if result.Inspect() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.Inspect())
			}
		})
	}
}

// TestTableRowInsertErrors tests error cases for row operations
func TestTableRowInsertErrors(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name: "insertRowAt - index out of bounds",
			input: `
				let {table} = import @std/table
				table([{a: 1}]).insertRowAt(5, {a: 2})
			`,
			expectedError: "out of range",
		},
		{
			name: "appendRow - wrong type",
			input: `
				let {table} = import @std/table
				table([{a: 1}]).appendRow("not a dict")
			`,
			expectedError: "must be a dictionary",
		},
		{
			name: "appendRow - wrong column count",
			input: `
				let {table} = import @std/table
				table([{a: 1, b: 2}]).appendRow({a: 1})
			`,
			expectedError: "columns",
		},
		{
			name: "appendRow - missing column",
			input: `
				let {table} = import @std/table
				table([{a: 1, b: 2}]).appendRow({a: 1, c: 2})
			`,
			expectedError: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalTableInsert(tt.input)
			err, ok := result.(*evaluator.Error)
			if !ok {
				t.Fatalf("expected error, got %T: %s", result, result.Inspect())
			}
			if !strings.Contains(strings.ToLower(err.Message), strings.ToLower(tt.expectedError)) {
				t.Errorf("expected error containing %q, got %q", tt.expectedError, err.Message)
			}
		})
	}
}
