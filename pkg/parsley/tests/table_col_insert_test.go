package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func evalTableColInsert(input string) evaluator.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		return &evaluator.Error{Message: p.Errors()[0]}
	}

	env := evaluator.NewEnvironment()
	return evaluator.Eval(program, env)
}

// TestTableAppendCol tests the table appendCol method with values array
func TestTableAppendColWithValues(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "append column with values",
			input: `
				let {table} = import(@std/table)
				let t = table([{name: "Alice"}, {name: "Bob"}])
				let t2 = t.appendCol("age", [30, 25])
				t2.rows[0].age
			`,
			expected: "30",
		},
		{
			name: "append column - verify second row",
			input: `
				let {table} = import(@std/table)
				let t = table([{name: "Alice"}, {name: "Bob"}])
				let t2 = t.appendCol("age", [30, 25])
				t2.rows[1].age
			`,
			expected: "25",
		},
		{
			name: "append column - original unchanged",
			input: `
				let {table} = import(@std/table)
				let t = table([{name: "Alice"}])
				let _ = t.appendCol("age", [30])
				t.rows[0].keys()
			`,
			expected: "[name]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalTableColInsert(tt.input)
			if result.Inspect() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.Inspect())
			}
		})
	}
}

// TestTableAppendColWithFunction tests the table appendCol method with function
func TestTableAppendColWithFunction(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "append computed column",
			input: `
				let {table} = import(@std/table)
				let t = table([{a: 10, b: 5}, {a: 20, b: 3}])
				let t2 = t.appendCol("sum", fn(row) { row.a + row.b })
				t2.rows[0].sum
			`,
			expected: "15",
		},
		{
			name: "append computed column - second row",
			input: `
				let {table} = import(@std/table)
				let t = table([{a: 10, b: 5}, {a: 20, b: 3}])
				let t2 = t.appendCol("sum", fn(row) { row.a + row.b })
				t2.rows[1].sum
			`,
			expected: "23",
		},
		{
			name: "append string computed column",
			input: `
				let {table} = import(@std/table)
				let t = table([{first: "John", last: "Doe"}])
				let t2 = t.appendCol("full", fn(row) { row.first + " " + row.last })
				t2.rows[0].full
			`,
			expected: "John Doe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalTableColInsert(tt.input)
			if result.Inspect() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.Inspect())
			}
		})
	}
}

// TestTableInsertColAfter tests inserting column after existing column
func TestTableInsertColAfter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "insert column after first",
			input: `
				let {table} = import(@std/table)
				let t = table([{a: 1, c: 3}])
				let t2 = t.insertColAfter("a", "b", [2])
				t2.rows[0].keys()
			`,
			expected: "[a, b, c]",
		},
		{
			name: "insert column with function",
			input: `
				let {table} = import(@std/table)
				let t = table([{x: 10, z: 30}])
				let t2 = t.insertColAfter("x", "y", fn(row) { row.x * 2 })
				t2.rows[0].y
			`,
			expected: "20",
		},
		{
			name: "insert after last column",
			input: `
				let {table} = import(@std/table)
				let t = table([{a: 1, b: 2}])
				let t2 = t.insertColAfter("b", "c", [3])
				t2.rows[0].keys()
			`,
			expected: "[a, b, c]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalTableColInsert(tt.input)
			if result.Inspect() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.Inspect())
			}
		})
	}
}

// TestTableInsertColBefore tests inserting column before existing column
func TestTableInsertColBefore(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "insert column before first",
			input: `
				let {table} = import(@std/table)
				let t = table([{b: 2, c: 3}])
				let t2 = t.insertColBefore("b", "a", [1])
				t2.rows[0].keys()
			`,
			expected: "[a, b, c]",
		},
		{
			name: "insert column before middle",
			input: `
				let {table} = import(@std/table)
				let t = table([{a: 1, c: 3}])
				let t2 = t.insertColBefore("c", "b", [2])
				t2.rows[0].keys()
			`,
			expected: "[a, b, c]",
		},
		{
			name: "insert with function",
			input: `
				let {table} = import(@std/table)
				let t = table([{y: 20, z: 30}])
				let t2 = t.insertColBefore("y", "x", fn(row) { row.y / 2 })
				t2.rows[0].x
			`,
			expected: "10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalTableColInsert(tt.input)
			if result.Inspect() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.Inspect())
			}
		})
	}
}

// TestTableColInsertErrors tests error cases for column operations
func TestTableColInsertErrors(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name: "appendCol - values length mismatch",
			input: `
				let {table} = import(@std/table)
				table([{a: 1}, {a: 2}]).appendCol("b", [1])
			`,
			expectedError: "values",
		},
		{
			name: "appendCol - duplicate column name",
			input: `
				let {table} = import(@std/table)
				table([{a: 1}]).appendCol("a", [2])
			`,
			expectedError: "already exists",
		},
		{
			name: "insertColAfter - column not found",
			input: `
				let {table} = import(@std/table)
				table([{a: 1}]).insertColAfter("x", "b", [2])
			`,
			expectedError: "not found",
		},
		{
			name: "insertColBefore - column not found",
			input: `
				let {table} = import(@std/table)
				table([{a: 1}]).insertColBefore("x", "b", [2])
			`,
			expectedError: "not found",
		},
		{
			name: "insertColAfter - duplicate new column",
			input: `
				let {table} = import(@std/table)
				table([{a: 1, b: 2}]).insertColAfter("a", "b", [3])
			`,
			expectedError: "already exists",
		},
		{
			name: "appendCol - invalid third arg type",
			input: `
				let {table} = import(@std/table)
				table([{a: 1}]).appendCol("b", "not array or function")
			`,
			expectedError: "array or function",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalTableColInsert(tt.input)
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
