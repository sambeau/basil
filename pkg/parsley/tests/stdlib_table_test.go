package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func TestStdlibTableImport(t *testing.T) {
	input := `let {table} = import @std/table
table`

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
	input := `let {table} = import @std/table
data = [{name: "Alice", age: 30}, {name: "Bob", age: 25}]
t = table(data)
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
	input := `let {table} = import @std/table
table([])`

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
		{`let {table} = import @std/table; table("not array")`, "must be an array"},
		{`let {table} = import @std/table; table([1, 2, 3])`, "must be dictionary"},
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
	input := `let {table} = import @std/table
data = [{a: 1}, {a: 2}]
table(data).rows`

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
	input := `let {table} = import @std/table
data = [{a: 1}, {a: 2}, {a: 3}]
table(data).count()`

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
	input := `let {table} = import @std/table
data = [{name: "Alice", age: 30}, {name: "Bob", age: 25}, {name: "Carol", age: 35}]
table(data).where(fn(row) { row.age > 25 }).count()`

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
	input := `let {table} = import @std/table
data = [{name: "Carol", val: 3}, {name: "Alice", val: 1}, {name: "Bob", val: 2}]
t = table(data).orderBy("name")
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
	input := `let {table} = import @std/table
data = [{name: "Carol", val: 3}, {name: "Alice", val: 1}, {name: "Bob", val: 2}]
t = table(data).orderBy("val", "desc")
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
	input := `let {table} = import @std/table
data = [{a: 1, b: 2, c: 3}, {a: 4, b: 5, c: 6}]
t = table(data).select(["a", "c"])
t.rows[0].b`

	result := evalTest(t, input)

	// b should not exist, accessing it should return null
	if result != evaluator.NULL {
		t.Errorf("expected NULL for missing column, got %s: %s", result.Type(), result.Inspect())
	}
}

func TestTableLimit(t *testing.T) {
	input := `let {table} = import @std/table
data = [{v: 1}, {v: 2}, {v: 3}, {v: 4}, {v: 5}]
table(data).limit(2).count()`

	result := evalTest(t, input)

	intVal := result.(*evaluator.Integer)
	if intVal.Value != 2 {
		t.Errorf("expected 2, got %d", intVal.Value)
	}
}

func TestTableLimitOffset(t *testing.T) {
	input := `let {table} = import @std/table
data = [{v: 1}, {v: 2}, {v: 3}, {v: 4}, {v: 5}]
t = table(data).limit(2, 2)
t.rows[0].v`

	result := evalTest(t, input)

	intVal := result.(*evaluator.Integer)
	if intVal.Value != 3 {
		t.Errorf("expected 3 (offset 2 = third element), got %d", intVal.Value)
	}
}

func TestTableSum(t *testing.T) {
	input := `let {table} = import @std/table
data = [{val: 10}, {val: 20}, {val: 30}]
table(data).sum("val")`

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
	input := `let {table} = import @std/table
data = [{val: 10}, {val: 20}, {val: 30}]
table(data).avg("val")`

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
	input := `let {table} = import @std/table
data = [{val: 10}, {val: 20}, {val: 5}]
tbl = table(data)
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
	input := `let {table} = import @std/table
data = [{name: "Alice", age: 30}]
table(data).toHTML()`

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
	input := `let {table} = import @std/table
data = [{name: "Alice", age: 30}, {name: "Bob", age: 25}]
table(data).toCSV()`

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
	input := `let {table} = import @std/table
data = [{text: "hello, world"}, {text: "has \"quotes\""}]
table(data).toCSV()`

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
	input := `let {table} = import @std/table
data = [{name: "Alice", age: 30, active: true}, 
        {name: "Bob", age: 25, active: true},
        {name: "Carol", age: 35, active: false},
        {name: "Dan", age: 28, active: true}]
table(data)
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
	input := `let {table} = import @std/table
data = [{val: 1}, {val: 2}, {val: 3}]
original = table(data)
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
	input := `import @std/nonexistent`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	env := evaluator.NewEnvironment()
	result := evaluator.Eval(program, env)

	if result.Type() != evaluator.ERROR_OBJ {
		t.Fatalf("expected error, got %s", result.Type())
	}

	errMsg := result.(*evaluator.Error).Message
	if !strings.Contains(strings.ToLower(errMsg), strings.ToLower("unknown standard library module")) {
		t.Errorf("expected 'unknown standard library module' error, got %q", errMsg)
	}
}

// TestTableSumWithStringNumbers tests that sum() coerces string numbers
func TestTableSumWithStringNumbers(t *testing.T) {
	// Create a table with string values (simulating legacy or mixed data)
	input := `let {table} = import @std/table
data = [{val: "10"}, {val: "20"}, {val: "30"}]
table(data).sum("val")`

	result := evalTest(t, input)

	intVal, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %s", result.Type())
	}

	if intVal.Value != 60 {
		t.Errorf("expected 60, got %d", intVal.Value)
	}
}

// TestTableAvgWithStringNumbers tests that avg() coerces string numbers
func TestTableAvgWithStringNumbers(t *testing.T) {
	input := `let {table} = import @std/table
data = [{val: "10"}, {val: "20"}, {val: "30"}]
table(data).avg("val")`

	result := evalTest(t, input)

	floatVal, ok := result.(*evaluator.Float)
	if !ok {
		t.Fatalf("expected Float, got %s", result.Type())
	}

	if floatVal.Value != 20.0 {
		t.Errorf("expected 20.0, got %f", floatVal.Value)
	}
}

// TestTableMinWithStringNumbers tests that min() coerces string numbers for proper numeric comparison
func TestTableMinWithStringNumbers(t *testing.T) {
	// Without coercion, "5" > "10" lexicographically, so min would wrongly be "10"
	input := `let {table} = import @std/table
data = [{val: "5"}, {val: "10"}, {val: "2"}]
table(data).min("val")`

	result := evalTest(t, input)

	intVal, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer (coerced), got %s", result.Type())
	}

	if intVal.Value != 2 {
		t.Errorf("expected 2 (numeric min), got %d", intVal.Value)
	}
}

// TestTableMaxWithStringNumbers tests that max() coerces string numbers for proper numeric comparison
func TestTableMaxWithStringNumbers(t *testing.T) {
	// Without coercion, "9" > "100" lexicographically, so max would wrongly be "9"
	input := `let {table} = import @std/table
data = [{val: "9"}, {val: "100"}, {val: "50"}]
table(data).max("val")`

	result := evalTest(t, input)

	intVal, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer (coerced), got %s", result.Type())
	}

	if intVal.Value != 100 {
		t.Errorf("expected 100 (numeric max), got %d", intVal.Value)
	}
}

// TestTableWhereWithCSVData tests where() with type-coerced CSV data
func TestTableWhereWithCSVData(t *testing.T) {
	input := `let {table} = import @std/table
let data = "name,value\na,10\nb,20\nc,5\nd,15".parseCSV()
let t = table(data)
t.where(fn(row) { row.value > 10 }).count()`

	result := evalTest(t, input)

	intVal, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %s", result.Type())
	}

	// Values > 10 are: 20, 15 (2 rows)
	if intVal.Value != 2 {
		t.Errorf("expected 2 rows with value > 10, got %d", intVal.Value)
	}
}

// TestTableOrderByWithCSVData tests orderBy() with type-coerced CSV data
func TestTableOrderByWithCSVData(t *testing.T) {
	input := `let {table} = import @std/table
let data = "name,value\na,10\nb,2\nc,100".parseCSV()
let t = table(data).orderBy("value")
t.rows[0].value`

	result := evalTest(t, input)

	// Should be sorted numerically: 2 is first (not "10" lexicographically)
	intVal, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %s", result.Type())
	}

	if intVal.Value != 2 {
		t.Errorf("expected first value to be 2 (numeric sort), got %d", intVal.Value)
	}
}

// TestTableAggregatesWithCSVData tests all aggregate functions with CSV data
func TestTableAggregatesWithCSVData(t *testing.T) {
	// Test sum
	input := `let {table} = import @std/table
let data = "value\n10\n20\n30\n40".parseCSV()
table(data).sum("value")`

	result := evalTest(t, input)
	sumVal := result.(*evaluator.Integer).Value
	if sumVal != 100 {
		t.Errorf("expected sum=100, got %d", sumVal)
	}

	// Test avg
	input = `let {table} = import @std/table
let data = "value\n10\n20\n30\n40".parseCSV()
table(data).avg("value")`

	result = evalTest(t, input)
	avgVal := result.(*evaluator.Float).Value
	if avgVal != 25.0 {
		t.Errorf("expected avg=25.0, got %f", avgVal)
	}

	// Test min
	input = `let {table} = import @std/table
let data = "value\n10\n20\n30\n40".parseCSV()
table(data).min("value")`

	result = evalTest(t, input)
	minVal := result.(*evaluator.Integer).Value
	if minVal != 10 {
		t.Errorf("expected min=10, got %d", minVal)
	}

	// Test max
	input = `let {table} = import @std/table
let data = "value\n10\n20\n30\n40".parseCSV()
table(data).max("value")`

	result = evalTest(t, input)
	maxVal := result.(*evaluator.Integer).Value
	if maxVal != 40 {
		t.Errorf("expected max=40, got %d", maxVal)
	}

	// Test count
	input = `let {table} = import @std/table
let data = "value\n10\n20\n30\n40".parseCSV()
table(data).count()`

	result = evalTest(t, input)
	countVal := result.(*evaluator.Integer).Value
	if countVal != 4 {
		t.Errorf("expected count=4, got %d", countVal)
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

func TestDictionaryEntries(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "entries with default names",
			input:    `let d = {a: 1, b: 2}; d.entries()`,
			expected: `[{key: a, value: 1}, {key: b, value: 2}]`,
		},
		{
			name:     "entries with custom names",
			input:    `let d = {x: 10, y: 20}; d.entries("Name", "Val")`,
			expected: `[{Name: x, Val: 10}, {Name: y, Val: 20}]`,
		},
		{
			name:     "entries with string keys",
			input:    `let d = {"Total": 100}; d.entries("Category", "Value")`,
			expected: `[{Category: Total, Value: 100}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalTest(t, tt.input)
			// Check that it's an array
			arr, ok := result.(*evaluator.Array)
			if !ok {
				t.Fatalf("expected Array, got %s", result.Type())
			}
			// Should have entries
			if len(arr.Elements) == 0 {
				t.Fatal("expected non-empty array")
			}
			// Each element should be a dictionary
			for _, elem := range arr.Elements {
				if elem.Type() != evaluator.DICTIONARY_OBJ {
					t.Errorf("expected DICTIONARY, got %s", elem.Type())
				}
			}
		})
	}
}

func TestTableFromDict(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		expectRows int
		expectCols []string
	}{
		{
			name: "fromDict with default column names",
			input: `let {table} = import @std/table
let d = {a: 1, b: 2}
table.fromDict(d)`,
			expectRows: 2,
			expectCols: []string{"key", "value"},
		},
		{
			name: "fromDict with custom column names",
			input: `let {table} = import @std/table
let d = {"Total": 100, "Active": 50}
table.fromDict(d, "Category", "Value")`,
			expectRows: 2,
			expectCols: []string{"Category", "Value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalTest(t, tt.input)
			tbl, ok := result.(*evaluator.Table)
			if !ok {
				t.Fatalf("expected Table, got %s", result.Type())
			}
			if len(tbl.Rows) != tt.expectRows {
				t.Errorf("expected %d rows, got %d", tt.expectRows, len(tbl.Rows))
			}
			if len(tbl.Columns) != len(tt.expectCols) {
				t.Errorf("expected columns %v, got %v", tt.expectCols, tbl.Columns)
			}
			for i, col := range tt.expectCols {
				if tbl.Columns[i] != col {
					t.Errorf("expected column %d to be %s, got %s", i, col, tbl.Columns[i])
				}
			}
		})
	}
}

func TestBasilStdlibImport(t *testing.T) {
	// Test that std/basil import works (returns empty dict when not in handler context)
	input := `
		let {basil} = import @std/basil
		basil
	`
	result := evalTest(t, input)

	// Should be a Dictionary (empty in test context)
	if result.Type() != evaluator.DICTIONARY_OBJ {
		t.Errorf("expected Dictionary, got %s", result.Type())
	}
}

func TestBasilStdlibImportWithContext(t *testing.T) {
	// Test that std/basil import returns the context when set
	env := evaluator.NewEnvironment()

	// Create a mock basil context
	mockBasil := &evaluator.Dictionary{
		Pairs: map[string]ast.Expression{},
	}
	env.BasilCtx = mockBasil

	l := lexer.New(`let {basil} = import @std/basil; basil`)
	p := parser.New(l)
	program := p.ParseProgram()

	result := evaluator.Eval(program, env)

	// Should be the same object we set
	if result != mockBasil {
		t.Errorf("expected basil context to be returned, got %s", result.Type())
	}
}
