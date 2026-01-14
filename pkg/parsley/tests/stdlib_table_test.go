package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
	"github.com/sambeau/basil/pkg/parsley/parsley"
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

func assertNoNilExpressions(t *testing.T, dict *evaluator.Dictionary, prefix string) {
	t.Helper()
	if dict == nil {
		t.Fatalf("nil dictionary at %s", prefix)
	}
	for key, expr := range dict.Pairs {
		if expr == nil {
			t.Fatalf("nil expression for %s.%s", prefix, key)
		}
		if objLit, ok := expr.(*ast.ObjectLiteralExpression); ok {
			if nested, ok := objLit.Obj.(*evaluator.Dictionary); ok {
				next := key
				if prefix != "" {
					next = prefix + "." + key
				}
				assertNoNilExpressions(t, nested, next)
			}
		}
	}
}

// TestTableBuiltinConstructor tests the Table() builtin (no import needed)
func TestTableBuiltinConstructor(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectRows  int
		expectCols  []string
		expectError string
	}{
		{
			name:       "basic table",
			input:      `Table([{a: 1, b: 2}, {a: 3, b: 4}])`,
			expectRows: 2,
			expectCols: []string{"a", "b"},
		},
		{
			name:       "empty array",
			input:      `Table([])`,
			expectRows: 0,
			expectCols: []string{},
		},
		{
			name:       "no args",
			input:      `Table()`,
			expectRows: 0,
			expectCols: []string{},
		},
		{
			name:       "single row",
			input:      `Table([{x: 1}])`,
			expectRows: 1,
			expectCols: []string{"x"},
		},
		{
			name:        "not array",
			input:       `Table("string")`,
			expectError: "requires an array",
		},
		{
			name:        "not dictionary",
			input:       `Table([1, 2, 3])`,
			expectError: "expected dictionary",
		},
		{
			name:        "column mismatch - missing",
			input:       `Table([{a: 1}, {b: 2}])`,
			expectError: "missing columns",
		},
		{
			name:        "column mismatch - extra",
			input:       `Table([{a: 1}, {a: 2, b: 3}])`,
			expectError: "unexpected columns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()
			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}
			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)

			if tt.expectError != "" {
				if result.Type() != evaluator.ERROR_OBJ {
					t.Fatalf("expected error containing %q, got %s", tt.expectError, result.Type())
				}
				errMsg := result.(*evaluator.Error).Message
				if !strings.Contains(errMsg, tt.expectError) {
					t.Errorf("expected error containing %q, got %q", tt.expectError, errMsg)
				}
				return
			}

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("unexpected error: %s", result.Inspect())
			}

			if result.Type() != evaluator.TABLE_OBJ {
				t.Fatalf("expected TABLE, got %s: %s", result.Type(), result.Inspect())
			}

			table := result.(*evaluator.Table)
			if len(table.Rows) != tt.expectRows {
				t.Errorf("expected %d rows, got %d", tt.expectRows, len(table.Rows))
			}
			if len(table.Columns) != len(tt.expectCols) {
				t.Errorf("expected %d columns, got %d", len(tt.expectCols), len(table.Columns))
			}
		})
	}
}

// TestTableBuiltinProperties tests .length, .columns, .schema properties
func TestTableBuiltinProperties(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "length",
			input:    `Table([{a: 1}, {a: 2}]).length`,
			expected: "2",
		},
		{
			name:     "columns",
			input:    `Table([{x: 1, y: 2}]).columns`,
			expected: "[x, y]",
		},
		{
			name:     "schema is null",
			input:    `Table([{a: 1}]).schema`,
			expected: "null",
		},
		{
			name:     "empty table length",
			input:    `Table([]).length`,
			expected: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("unexpected error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
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
		{`let {table} = import @std/table; table("not array")`, "requires an array"},
		{`let {table} = import @std/table; table([1, 2, 3])`, "expected dictionary"},
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

func TestTableOrderByAsc(t *testing.T) {
	input := `let {table} = import @std/table
data = [{name: "Carol", val: 3}, {name: "Alice", val: 1}, {name: "Bob", val: 2}]
t = table(data).orderBy("val", "asc")
t.rows[0].val`

	result := evalTest(t, input)

	intVal, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %s: %s", result.Type(), result.Inspect())
	}

	if intVal.Value != 1 {
		t.Errorf("expected 1, got %d", intVal.Value)
	}
}

func TestTableOrderByDynamic(t *testing.T) {
	// Test programmatic control of sort direction
	input := `let {table} = import @std/table
data = [{name: "Carol", val: 3}, {name: "Alice", val: 1}, {name: "Bob", val: 2}]
sortDir = "desc"
t = table(data).orderBy("val", sortDir)
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
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
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
	// parseCSV() now returns Table directly, so no need to wrap in table()
	input := `let data = "name,value\na,10\nb,20\nc,5\nd,15".parseCSV()
data.where(fn(row) { row.value > 10 }).count()`

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
	// parseCSV() now returns Table directly, so no need to wrap in table()
	input := `let t = "name,value\na,10\nb,2\nc,100".parseCSV().orderBy("value")
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
	// parseCSV() now returns Table directly, so no need to wrap in table()
	// Test sum
	input := `"value\n10\n20\n30\n40".parseCSV().sum("value")`

	result := evalTest(t, input)
	sumVal := result.(*evaluator.Integer).Value
	if sumVal != 100 {
		t.Errorf("expected sum=100, got %d", sumVal)
	}

	// Test avg
	input = `"value\n10\n20\n30\n40".parseCSV().avg("value")`

	result = evalTest(t, input)
	avgVal := result.(*evaluator.Float).Value
	if avgVal != 25.0 {
		t.Errorf("expected avg=25.0, got %f", avgVal)
	}

	// Test min
	input = `"value\n10\n20\n30\n40".parseCSV().min("value")`

	result = evalTest(t, input)
	minVal := result.(*evaluator.Integer).Value
	if minVal != 10 {
		t.Errorf("expected min=10, got %d", minVal)
	}

	// Test max
	input = `"value\n10\n20\n30\n40".parseCSV().max("value")`

	result = evalTest(t, input)
	maxVal := result.(*evaluator.Integer).Value
	if maxVal != 40 {
		t.Errorf("expected max=40, got %d", maxVal)
	}

	// Test count
	input = `"value\n10\n20\n30\n40".parseCSV().count()`

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

func TestStdBasilImportFailsWithError(t *testing.T) {
	l := lexer.New(`let {basil} = import @std/basil; basil`)
	p := parser.New(l)
	program := p.ParseProgram()

	env := evaluator.NewEnvironment()
	result := evaluator.Eval(program, env)

	err, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected error, got %s", result.Type())
	}

	if err.Code != "IMPORT-0006" {
		t.Fatalf("expected IMPORT-0006, got %s (%s)", err.Code, err.Message)
	}

	if !strings.Contains(err.Message, "removed") {
		t.Fatalf("expected removal message, got %q", err.Message)
	}
	if !strings.Contains(err.Message, "@basil/http") {
		t.Fatalf("expected hint for @basil/http, got %q", err.Message)
	}
}

func TestBasilHttpModuleWithContext(t *testing.T) {
	env := evaluator.NewEnvironment()

	basilMap := map[string]interface{}{
		"http": map[string]interface{}{
			"request": map[string]interface{}{
				"method": "GET",
				"path":   "/users/42",
				"query":  map[string]interface{}{"flag": true},
				"route": map[string]interface{}{
					"__type":   "path",
					"absolute": false,
					"segments": []interface{}{"users", "42"},
				},
			},
			"response": map[string]interface{}{},
		},
		"auth": map[string]interface{}{
			"user": map[string]interface{}{"id": "u1"},
		},
	}

	basilObj, err := parsley.ToParsley(basilMap)
	if err != nil {
		t.Fatalf("failed to build basil context: %v", err)
	}
	if dict, ok := basilObj.(*evaluator.Dictionary); ok {
		assertNoNilExpressions(t, dict, "basil")
		env.BasilCtx = dict
	} else {
		t.Fatalf("expected basil context dictionary, got %T", basilObj)
	}

	httpParser := parser.New(lexer.New(`import @basil/http`))
	httpProgram := httpParser.ParseProgram()
	if len(httpParser.Errors()) > 0 {
		t.Fatalf("import parser errors: %v", httpParser.Errors())
	}
	httpResult := evaluator.Eval(httpProgram, env)
	if errObj, isErr := httpResult.(*evaluator.Error); isErr {
		t.Fatalf("import @basil/http failed: %s", errObj.Inspect())
	}

	testCases := []struct {
		name   string
		src    string
		assert func(t *testing.T, obj evaluator.Object)
	}{
		{
			name: "request.query access",
			src: `let {request} = import @basil/http
request.query.flag`,
			assert: func(t *testing.T, obj evaluator.Object) {
				b, ok := obj.(*evaluator.Boolean)
				if !ok || b.Value != true {
					t.Fatalf("expected request.query.flag true, got %s", obj.Inspect())
				}
			},
		},
		{
			name: "route match",
			src: `let {route} = import @basil/http
route.match("users/:id").id`,
			assert: func(t *testing.T, obj evaluator.Object) {
				s, ok := obj.(*evaluator.String)
				if !ok || s.Value != "42" {
					t.Fatalf("expected matched id 42, got %s", obj.Inspect())
				}
			},
		},
		{
			name: "method",
			src: `let {method} = import @basil/http
method`,
			assert: func(t *testing.T, obj evaluator.Object) {
				s, ok := obj.(*evaluator.String)
				if !ok || s.Value != "GET" {
					t.Fatalf("expected method GET, got %s", obj.Inspect())
				}
			},
		},
	}

	for _, tc := range testCases {
		l := lexer.New(tc.src)
		p := parser.New(l)
		program := p.ParseProgram()
		if len(p.Errors()) > 0 {
			t.Fatalf("%s parser errors: %v", tc.name, p.Errors())
		}

		result := evaluator.Eval(program, env)
		if errObj, isErr := result.(*evaluator.Error); isErr {
			t.Fatalf("%s eval error: %s", tc.name, errObj.Inspect())
		}

		tc.assert(t, result)
	}
}

func TestBasilAuthModuleWithContext(t *testing.T) {
	env := evaluator.NewEnvironment()

	basilMap := map[string]interface{}{
		"auth": map[string]interface{}{
			"user": map[string]interface{}{"email": "user@example.com"},
		},
		"sqlite":  "db-conn",
		"session": map[string]interface{}{"id": "sess-1"},
	}

	basilObj, err := parsley.ToParsley(basilMap)
	if err != nil {
		t.Fatalf("failed to build basil context: %v", err)
	}
	if dict, ok := basilObj.(*evaluator.Dictionary); ok {
		assertNoNilExpressions(t, dict, "basil")
		env.BasilCtx = dict
	} else {
		t.Fatalf("expected basil context dictionary, got %T", basilObj)
	}

	authParser := parser.New(lexer.New(`import @basil/auth`))
	authProgram := authParser.ParseProgram()
	if len(authParser.Errors()) > 0 {
		t.Fatalf("import parser errors: %v", authParser.Errors())
	}
	authResult := evaluator.Eval(authProgram, env)
	if errObj, isErr := authResult.(*evaluator.Error); isErr {
		t.Fatalf("import @basil/auth failed: %s", errObj.Inspect())
	}

	testCases := []struct {
		name   string
		src    string
		assert func(t *testing.T, obj evaluator.Object)
	}{
		{
			name: "session",
			src: `let {session} = import @basil/auth
session.id`,
			assert: func(t *testing.T, obj evaluator.Object) {
				s, ok := obj.(*evaluator.String)
				if !ok || s.Value != "sess-1" {
					t.Fatalf("expected session id, got %s", obj.Inspect())
				}
			},
		},
		{
			name: "auth user",
			src: `let {auth} = import @basil/auth
auth.user.email`,
			assert: func(t *testing.T, obj evaluator.Object) {
				s, ok := obj.(*evaluator.String)
				if !ok || s.Value != "user@example.com" {
					t.Fatalf("expected auth.user.email, got %s", obj.Inspect())
				}
			},
		},
		{
			name: "user shortcut",
			src: `let {user} = import @basil/auth
user.email`,
			assert: func(t *testing.T, obj evaluator.Object) {
				s, ok := obj.(*evaluator.String)
				if !ok || s.Value != "user@example.com" {
					t.Fatalf("expected user.email, got %s", obj.Inspect())
				}
			},
		},
	}

	for _, tc := range testCases {
		l := lexer.New(tc.src)
		p := parser.New(l)
		program := p.ParseProgram()
		if len(p.Errors()) > 0 {
			t.Fatalf("%s parser errors: %v", tc.name, p.Errors())
		}

		result := evaluator.Eval(program, env)
		if errObj, isErr := result.(*evaluator.Error); isErr {
			t.Fatalf("%s eval error: %s", tc.name, errObj.Inspect())
		}

		tc.assert(t, result)
	}
}

// TestTableLiteralSyntax tests the @table [...] literal syntax
func TestTableLiteralSyntax(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectRows  int
		expectCols  int
		expectError string
	}{
		{
			name:       "basic @table literal",
			input:      `@table [{a: 1, b: 2}, {a: 3, b: 4}]`,
			expectRows: 2,
			expectCols: 2,
		},
		{
			name:       "empty @table",
			input:      `@table []`,
			expectRows: 0,
			expectCols: 0,
		},
		{
			name:       "single row @table",
			input:      `@table [{x: 1, y: 2, z: 3}]`,
			expectRows: 1,
			expectCols: 3,
		},
		{
			name:       "@table length property",
			input:      `@table [{a: 1}, {a: 2}, {a: 3}].length`,
			expectRows: -1, // special case: checking property
		},
		{
			name:        "@table with missing column",
			input:       `@table [{a: 1}, {b: 2}]`,
			expectError: "missing columns",
		},
		{
			name:        "@table with extra column",
			input:       `@table [{a: 1}, {a: 2, b: 3}]`,
			expectError: "extra columns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			// Check for parse errors
			if len(p.Errors()) > 0 {
				if tt.expectError != "" {
					// Check if any error contains expected message
					for _, err := range p.Errors() {
						if strings.Contains(err, tt.expectError) {
							return // expected error found
						}
					}
				}
				t.Fatalf("parser errors: %v", p.Errors())
			}

			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)

			if tt.expectError != "" {
				if result.Type() != evaluator.ERROR_OBJ {
					t.Fatalf("expected error containing %q, got %s", tt.expectError, result.Type())
				}
				errMsg := result.(*evaluator.Error).Message
				if !strings.Contains(errMsg, tt.expectError) {
					t.Errorf("expected error containing %q, got %q", tt.expectError, errMsg)
				}
				return
			}

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("unexpected error: %s", result.Inspect())
			}

			// Special case for property access
			if tt.expectRows == -1 {
				if result.Type() != evaluator.INTEGER_OBJ {
					t.Fatalf("expected INTEGER, got %s", result.Type())
				}
				return
			}

			if result.Type() != evaluator.TABLE_OBJ {
				t.Fatalf("expected TABLE, got %s: %s", result.Type(), result.Inspect())
			}

			table := result.(*evaluator.Table)
			if len(table.Rows) != tt.expectRows {
				t.Errorf("expected %d rows, got %d", tt.expectRows, len(table.Rows))
			}
			if len(table.Columns) != tt.expectCols {
				t.Errorf("expected %d columns, got %d", tt.expectCols, len(table.Columns))
			}
		})
	}
}

// TestTableLiteralWithSchema tests @table(Schema) [...] syntax
func TestTableLiteralWithSchema(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectRows  int
		expectError string
	}{
		{
			name: "table with schema",
			input: `
@schema User { name: string, age: int }
@table(User) [{name: "Alice", age: 30}]
`,
			expectRows: 1,
		},
		{
			name: "table with schema applies defaults",
			input: `
@schema Config { name: string, enabled: bool = true }
let t = @table(Config) [{name: "test"}]
t.rows[0].enabled
`,
			expectRows: -1, // checking default was applied
		},
		{
			name: "table with undefined schema",
			input: `
@table(UnknownSchema) [{a: 1}]
`,
			expectError: "Identifier not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()
			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)

			if tt.expectError != "" {
				if result.Type() != evaluator.ERROR_OBJ {
					t.Fatalf("expected error containing %q, got %s: %s", tt.expectError, result.Type(), result.Inspect())
				}
				errMsg := result.(*evaluator.Error).Message
				if !strings.Contains(errMsg, tt.expectError) {
					t.Errorf("expected error containing %q, got %q", tt.expectError, errMsg)
				}
				return
			}

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("unexpected error: %s", result.Inspect())
			}

			// Special case for checking default value
			if tt.expectRows == -1 {
				if result.Type() != evaluator.BOOLEAN_OBJ {
					t.Fatalf("expected BOOLEAN (default value), got %s: %s", result.Type(), result.Inspect())
				}
				if !result.(*evaluator.Boolean).Value {
					t.Error("expected default value true to be applied")
				}
				return
			}

			if result.Type() != evaluator.TABLE_OBJ {
				t.Fatalf("expected TABLE, got %s: %s", result.Type(), result.Inspect())
			}

			table := result.(*evaluator.Table)
			if len(table.Rows) != tt.expectRows {
				t.Errorf("expected %d rows, got %d", tt.expectRows, len(table.Rows))
			}
		})
	}
}

// TestTableCopyOnChain tests that method chaining uses copy-on-chain semantics
func TestTableCopyOnChain(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			name: "original unchanged after where",
			input: `
				data = Table([{x: 1}, {x: 2}, {x: 3}])
				filtered = data.where(fn(r) { r.x > 1 })
				data.length
			`,
			expected: 3, // data unchanged
		},
		{
			name: "original unchanged after orderBy",
			input: `
				data = Table([{x: 3}, {x: 1}, {x: 2}])
				sorted = data.orderBy("x")
				data.rows[0].x
			`,
			expected: 3, // first row should still be x:3
		},
		{
			name: "original unchanged after select",
			input: `
				data = Table([{x: 1, y: 2}, {x: 3, y: 4}])
				projected = data.select(["x"])
				data.columns.length()
			`,
			expected: 2, // should still have 2 columns
		},
		{
			name: "original unchanged after limit",
			input: `
				data = Table([{x: 1}, {x: 2}, {x: 3}])
				limited = data.limit(1)
				data.length
			`,
			expected: 3, // data unchanged
		},
		{
			name: "chained operations work correctly",
			input: `
				data = Table([{x: 3, y: "c"}, {x: 1, y: "a"}, {x: 2, y: "b"}])
				data.where(fn(r) { r.x > 1 }).orderBy("x").limit(1).rows[0].x
			`,
			expected: 2, // x:2 is the first after filtering x>1 and ordering
		},
		{
			name: "long chain preserves original",
			input: `
				data = Table([{x: 1}, {x: 2}, {x: 3}, {x: 4}, {x: 5}])
				result = data.where(fn(r) { r.x > 1 }).where(fn(r) { r.x < 5 }).orderBy("x").limit(2)
				data.length
			`,
			expected: 5, // original unchanged
		},
		{
			name: "two independent chains from same source - original preserved",
			input: `
				data = Table([{x: 1}, {x: 2}, {x: 3}])
				a = data.where(fn(r) { r.x == 1 })
				b = data.where(fn(r) { r.x == 2 })
				data.length
			`,
			expected: 3, // original unchanged
		},
		{
			name: "two independent chains - first chain result",
			input: `
				data = Table([{x: 1}, {x: 2}, {x: 3}])
				a = data.where(fn(r) { r.x == 1 })
				b = data.where(fn(r) { r.x == 2 })
				a.length
			`,
			expected: 1, // a has 1 row
		},
		{
			name: "two independent chains - second chain result",
			input: `
				data = Table([{x: 1}, {x: 2}, {x: 3}])
				a = data.where(fn(r) { r.x == 1 })
				b = data.where(fn(r) { r.x == 2 })
				b.length
			`,
			expected: 1, // b has 1 row
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()
			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("unexpected error: %s", result.Inspect())
			}

			intResult, ok := result.(*evaluator.Integer)
			if !ok {
				t.Fatalf("expected Integer, got %s: %s", result.Type(), result.Inspect())
			}

			if intResult.Value != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, intResult.Value)
			}
		})
	}
}

// TestTableCopyOnChainAssignment tests that assignment ends a chain
func TestTableCopyOnChainAssignment(t *testing.T) {
	// Test that data is unchanged
	input1 := `
		data = Table([{x: 1}, {x: 2}, {x: 3}])
		filtered = data.where(fn(r) { r.x > 1 })
		sorted = filtered.orderBy("x", "desc")
		data.length
	`
	result1 := evalTest(t, input1)
	if intVal, ok := result1.(*evaluator.Integer); !ok || intVal.Value != 3 {
		t.Errorf("data should have 3 rows, got %s", result1.Inspect())
	}

	// Test that filtered has 2 rows
	input2 := `
		data = Table([{x: 1}, {x: 2}, {x: 3}])
		filtered = data.where(fn(r) { r.x > 1 })
		sorted = filtered.orderBy("x", "desc")
		filtered.length
	`
	result2 := evalTest(t, input2)
	if intVal, ok := result2.(*evaluator.Integer); !ok || intVal.Value != 2 {
		t.Errorf("filtered should have 2 rows, got %s", result2.Inspect())
	}

	// Test that sorted has correct ordering (first row should be x:3)
	input3 := `
		data = Table([{x: 1}, {x: 2}, {x: 3}])
		filtered = data.where(fn(r) { r.x > 1 })
		sorted = filtered.orderBy("x", "desc")
		sorted.rows[0].x
	`
	result3 := evalTest(t, input3)
	if intVal, ok := result3.(*evaluator.Integer); !ok || intVal.Value != 3 {
		t.Errorf("sorted first row x should be 3, got %s", result3.Inspect())
	}
}

// TestTableCopyOnChainFunctionArg tests that passing table as argument ends chain
func TestTableCopyOnChainFunctionArg(t *testing.T) {
	input := `
		getLength = fn(tbl) {
			tbl.length
		}
		
		data = Table([{x: 1}, {x: 2}, {x: 3}])
		filtered = data.where(fn(r) { r.x > 1 })
		
		// Pass to function - should end chain
		getLength(filtered)
	`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	result := evaluator.Eval(program, env)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("unexpected error: %s", result.Inspect())
	}

	intResult, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %s", result.Type())
	}

	if intResult.Value != 2 {
		t.Errorf("expected 2, got %d", intResult.Value)
	}
}

// TestTableIndexing tests table[n] indexing with positive and negative indices
func TestTableIndexing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			name:     "first element",
			input:    `Table([{a: 1}, {a: 2}, {a: 3}])[0].a`,
			expected: 1,
		},
		{
			name:     "second element",
			input:    `Table([{a: 1}, {a: 2}, {a: 3}])[1].a`,
			expected: 2,
		},
		{
			name:     "last element via negative index",
			input:    `Table([{a: 1}, {a: 2}, {a: 3}])[-1].a`,
			expected: 3,
		},
		{
			name:     "second to last",
			input:    `Table([{a: 1}, {a: 2}, {a: 3}])[-2].a`,
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalTest(t, tt.input)
			intVal, ok := result.(*evaluator.Integer)
			if !ok {
				t.Fatalf("expected Integer, got %s: %s", result.Type(), result.Inspect())
			}
			if intVal.Value != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, intVal.Value)
			}
		})
	}
}

// TestTableIndexingError tests bounds checking on table indexing
func TestTableIndexingError(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "out of bounds positive",
			input: `Table([{a: 1}])[5]`,
		},
		{
			name:  "out of bounds negative",
			input: `Table([{a: 1}])[-10]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()
			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}
			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)
			if result.Type() != evaluator.ERROR_OBJ {
				t.Fatalf("expected error, got %s: %s", result.Type(), result.Inspect())
			}
		})
	}
}

// TestTableIteration tests for (row in table) { ... } iteration
func TestTableIteration(t *testing.T) {
	// The for loop returns an array of body results
	input := `
let t = Table([{a: 1}, {a: 2}, {a: 3}])
let results = for (row in t) {
	row.a
}
results[2]  // Third element (0-indexed)
`
	result := evalTest(t, input)
	intVal, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %s: %s", result.Type(), result.Inspect())
	}
	if intVal.Value != 3 {
		t.Errorf("expected results[2] = 3, got %d", intVal.Value)
	}
}

// TestTableToArray tests .toArray() method
func TestTableToArray(t *testing.T) {
	input := `Table([{a: 1}, {a: 2}]).toArray()`
	result := evalTest(t, input)
	arr, ok := result.(*evaluator.Array)
	if !ok {
		t.Fatalf("expected Array, got %s: %s", result.Type(), result.Inspect())
	}
	if len(arr.Elements) != 2 {
		t.Errorf("expected 2 elements, got %d", len(arr.Elements))
	}
}

// TestTableCopyMethod tests .copy() method creates independent copy
func TestTableCopyMethod(t *testing.T) {
	input := `
let original = Table([{a: 1}, {a: 2}])
let copied = original.copy()
copied.length
`
	result := evalTest(t, input)
	intVal, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %s: %s", result.Type(), result.Inspect())
	}
	if intVal.Value != 2 {
		t.Errorf("expected 2, got %d", intVal.Value)
	}
}

// TestTableOffsetMethod tests .offset() standalone method
func TestTableOffsetMethod(t *testing.T) {
	input := `Table([{a: 1}, {a: 2}, {a: 3}]).offset(1).length`
	result := evalTest(t, input)
	intVal, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %s: %s", result.Type(), result.Inspect())
	}
	if intVal.Value != 2 {
		t.Errorf("expected 2 rows after offset(1), got %d", intVal.Value)
	}
}
