package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func TestStdlibTableImport(t *testing.T) {
	input := `let {Table} = import("std/table")
Table`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	result := evaluator.Eval(program, env)

	if result == nil {
		t.Fatal("result is nil")
	}

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	// Table should be a StdlibBuiltin
	if result.Type() != evaluator.BUILTIN_OBJ {
		t.Errorf("expected BUILTIN, got %s", result.Type())
	}
}

func TestTableConstructor(t *testing.T) {
	input := `let {Table} = import("std/table")
data = [{name: "Alice", age: 30}, {name: "Bob", age: 25}]
t = Table(data)
t`

	result := evalTest(t, input)

	// Should be a Table
	if result.Type() != evaluator.TABLE_OBJ {
		t.Fatalf("expected TABLE, got %s: %s", result.Type(), result.Inspect())
	}

	table := result.(*evaluator.Table)
	if len(table.Rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(table.Rows))
	}
}

func TestTableEmptyArray(t *testing.T) {
	input := `let {Table} = import("std/table")
Table([])`

	result := evalTest(t, input)

	table := result.(*evaluator.Table)
	if len(table.Rows) != 0 {
		t.Errorf("expected 0 rows, got %d", len(table.Rows))
	}
}

func TestTableInvalidInput(t *testing.T) {
	tests := []struct {
		input       string
		errContains string
	}{
		{`let {Table} = import("std/table"); Table("not array")`, "must be an array"},
		{`let {Table} = import("std/table"); Table([1, 2, 3])`, "must be dictionaries"},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := parser.New(l)
		program := p.ParseProgram()
		env := evaluator.NewEnvironment()
		result := evaluator.Eval(program, env)

		if result.Type() != evaluator.ERROR_OBJ {
			t.Errorf("expected error for %s, got %s", tt.input, result.Type())
			continue
		}

		errMsg := result.(*evaluator.Error).Message
		if !strings.Contains(errMsg, tt.errContains) {
			t.Errorf("expected error containing %q, got %q", tt.errContains, errMsg)
		}
	}
}

func TestTableRows(t *testing.T) {
	input := `let {Table} = import("std/table")
data = [{a: 1}, {a: 2}]
Table(data).rows`

	result := evalTest(t, input)

	arr, ok := result.(*evaluator.Array)
	if !ok {
		t.Fatalf("expected Array, got %s", result.Type())
	}

	if len(arr.Elements) != 2 {
		t.Errorf("expected 2 elements, got %d", len(arr.Elements))
	}
}

func TestTableCount(t *testing.T) {
	input := `let {Table} = import("std/table")
data = [{a: 1}, {a: 2}, {a: 3}]
Table(data).count()`

	result := evalTest(t, input)

	intVal, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %s", result.Type())
	}

	if intVal.Value != 3 {
		t.Errorf("expected 3, got %d", intVal.Value)
	}
}

func TestTableWhere(t *testing.T) {
	input := `let {Table} = import("std/table")
data = [{name: "Alice", age: 30}, {name: "Bob", age: 25}, {name: "Carol", age: 35}]
Table(data).where(fn(row) { row.age > 25 }).count()`

	result := evalTest(t, input)

	intVal, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %s: %s", result.Type(), result.Inspect())
	}

	if intVal.Value != 2 {
		t.Errorf("expected 2, got %d", intVal.Value)
	}
}

func TestTableOrderBy(t *testing.T) {
	input := `let {Table} = import("std/table")
data = [{name: "Carol", val: 3}, {name: "Alice", val: 1}, {name: "Bob", val: 2}]
t = Table(data).orderBy("name")
t.rows[0].name`

	result := evalTest(t, input)

	strVal, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %s: %s", result.Type(), result.Inspect())
	}

	if strVal.Value != "Alice" {
		t.Errorf("expected Alice, got %s", strVal.Value)
	}
}

func TestTableOrderByDesc(t *testing.T) {
	input := `let {Table} = import("std/table")
data = [{name: "Carol", val: 3}, {name: "Alice", val: 1}, {name: "Bob", val: 2}]
t = Table(data).orderBy("val", "desc")
t.rows[0].val`

	result := evalTest(t, input)

	intVal, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %s: %s", result.Type(), result.Inspect())
	}

	if intVal.Value != 3 {
		t.Errorf("expected 3, got %d", intVal.Value)
	}
}

func TestTableSelect(t *testing.T) {
	input := `let {Table} = import("std/table")
data = [{a: 1, b: 2, c: 3}, {a: 4, b: 5, c: 6}]
t = Table(data).select(["a", "c"])
t.rows[0].b`

	result := evalTest(t, input)

	// b should not exist, accessing it should return null
	if result != evaluator.NULL {
		t.Errorf("expected NULL for missing column, got %s: %s", result.Type(), result.Inspect())
	}
}

func TestTableLimit(t *testing.T) {
	input := `let {Table} = import("std/table")
data = [{v: 1}, {v: 2}, {v: 3}, {v: 4}, {v: 5}]
Table(data).limit(2).count()`

	result := evalTest(t, input)

	intVal := result.(*evaluator.Integer)
	if intVal.Value != 2 {
		t.Errorf("expected 2, got %d", intVal.Value)
	}
}

func TestTableLimitOffset(t *testing.T) {
	input := `let {Table} = import("std/table")
data = [{v: 1}, {v: 2}, {v: 3}, {v: 4}, {v: 5}]
t = Table(data).limit(2, 2)
t.rows[0].v`

	result := evalTest(t, input)

	intVal := result.(*evaluator.Integer)
	if intVal.Value != 3 {
		t.Errorf("expected 3 (offset 2 = third element), got %d", intVal.Value)
	}
}

func TestTableSum(t *testing.T) {
	input := `let {Table} = import("std/table")
data = [{val: 10}, {val: 20}, {val: 30}]
Table(data).sum("val")`

	result := evalTest(t, input)

	intVal, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %s", result.Type())
	}

	if intVal.Value != 60 {
		t.Errorf("expected 60, got %d", intVal.Value)
	}
}

func TestTableAvg(t *testing.T) {
	input := `let {Table} = import("std/table")
data = [{val: 10}, {val: 20}, {val: 30}]
Table(data).avg("val")`

	result := evalTest(t, input)

	floatVal, ok := result.(*evaluator.Float)
	if !ok {
		t.Fatalf("expected Float, got %s", result.Type())
	}

	if floatVal.Value != 20.0 {
		t.Errorf("expected 20.0, got %f", floatVal.Value)
	}
}

func TestTableMinMax(t *testing.T) {
	input := `let {Table} = import("std/table")
data = [{val: 10}, {val: 20}, {val: 5}]
tbl = Table(data)
minVal = tbl.min("val"); maxVal = tbl.max("val"); [minVal, maxVal]`

	result := evalTest(t, input)

	arr := result.(*evaluator.Array)
	minVal := arr.Elements[0].(*evaluator.Integer).Value
	maxVal := arr.Elements[1].(*evaluator.Integer).Value

	if minVal != 5 {
		t.Errorf("expected min=5, got %d", minVal)
	}
	if maxVal != 20 {
		t.Errorf("expected max=20, got %d", maxVal)
	}
}

func TestTableToHTML(t *testing.T) {
	input := `let {Table} = import("std/table")
data = [{name: "Alice", age: 30}]
Table(data).toHTML()`

	result := evalTest(t, input)

	strVal := result.(*evaluator.String)
	html := strVal.Value

	if !strings.Contains(html, "<table>") {
		t.Error("expected <table> tag")
	}
	if !strings.Contains(html, "<thead>") {
		t.Error("expected <thead> tag")
	}
	if !strings.Contains(html, "<th>age</th>") && !strings.Contains(html, "<th>name</th>") {
		t.Error("expected header cells")
	}
	if !strings.Contains(html, "<td>Alice</td>") {
		t.Error("expected Alice data cell")
	}
	if !strings.Contains(html, "<td>30</td>") {
		t.Error("expected 30 data cell")
	}
}

func TestTableToCSV(t *testing.T) {
	input := `let {Table} = import("std/table")
data = [{name: "Alice", age: 30}, {name: "Bob", age: 25}]
Table(data).toCSV()`

	result := evalTest(t, input)

	strVal := result.(*evaluator.String)
	csv := strVal.Value

	// Should have header row
	if !strings.Contains(csv, "age") || !strings.Contains(csv, "name") {
		t.Error("expected header row with age and name")
	}
	// Should have CRLF line endings
	if !strings.Contains(csv, "\r\n") {
		t.Error("expected CRLF line endings")
	}
	// Should have data
	if !strings.Contains(csv, "Alice") || !strings.Contains(csv, "Bob") {
		t.Error("expected data rows")
	}
}

func TestTableCSVEscaping(t *testing.T) {
	input := `let {Table} = import("std/table")
data = [{text: "hello, world"}, {text: "has \"quotes\""}]
Table(data).toCSV()`

	result := evalTest(t, input)

	strVal := result.(*evaluator.String)
	csv := strVal.Value

	// Commas and quotes should be escaped
	if !strings.Contains(csv, `"hello, world"`) {
		t.Error("expected comma-containing value to be quoted")
	}
	if !strings.Contains(csv, `"has ""quotes"""`) {
		t.Error("expected quotes to be escaped by doubling")
	}
}

func TestTableChaining(t *testing.T) {
	input := `let {Table} = import("std/table")
data = [{name: "Alice", age: 30, active: true}, 
        {name: "Bob", age: 25, active: true},
        {name: "Carol", age: 35, active: false},
        {name: "Dan", age: 28, active: true}]
Table(data)
    .where(fn(row) { row.active })
    .orderBy("age", "desc")
    .select(["name", "age"])
    .limit(2)
    .rows[0].name`

	result := evalTest(t, input)

	strVal := result.(*evaluator.String)
	// Active users sorted by age desc: Alice(30), Dan(28), Bob(25)
	// Limited to 2: Alice, Dan
	// First one: Alice
	if strVal.Value != "Alice" {
		t.Errorf("expected Alice, got %s", strVal.Value)
	}
}

func TestTableImmutability(t *testing.T) {
	input := `let {Table} = import("std/table")
data = [{val: 1}, {val: 2}, {val: 3}]
original = Table(data)
filtered = original.where(fn(row) { row.val > 1 })
origCount = original.count(); filtCount = filtered.count(); [origCount, filtCount]`

	result := evalTest(t, input)

	arr := result.(*evaluator.Array)
	originalCount := arr.Elements[0].(*evaluator.Integer).Value
	filteredCount := arr.Elements[1].(*evaluator.Integer).Value

	if originalCount != 3 {
		t.Errorf("expected original count=3, got %d", originalCount)
	}
	if filteredCount != 2 {
		t.Errorf("expected filtered count=2, got %d", filteredCount)
	}
}

func TestUnknownStdlibModule(t *testing.T) {
	input := `import("std/nonexistent")`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	env := evaluator.NewEnvironment()
	result := evaluator.Eval(program, env)

	if result.Type() != evaluator.ERROR_OBJ {
		t.Fatalf("expected error, got %s", result.Type())
	}

	errMsg := result.(*evaluator.Error).Message
	if !strings.Contains(errMsg, "unknown standard library module") {
		t.Errorf("expected 'unknown standard library module' error, got %q", errMsg)
	}
}

// Helper function to evaluate test input
func evalTest(t *testing.T, input string) evaluator.Object {
	t.Helper()

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	result := evaluator.Eval(program, env)

	if result == nil {
		t.Fatal("result is nil")
	}

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	return result
}
