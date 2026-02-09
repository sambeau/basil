package tests

import (
	"slices"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// Helper function to check if a slice contains a string
func containsColumn(slice []string, str string) bool {
	return slices.Contains(slice, str)
}

// TestTableMap tests the map() method with various scenarios
func TestTableMap(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		expectRows  int
		validate    func(*testing.T, evaluator.Object)
	}{
		{
			name: "map with simple transformation",
			input: `
				let data = table([{a: 1, b: 2}, {a: 3, b: 4}])
				data.map(fn(row) { {a: row.a * 2, b: row.b * 2} })
			`,
			expectRows: 2,
			validate: func(t *testing.T, result evaluator.Object) {
				table := result.(*evaluator.Table)
				if len(table.Rows) != 2 {
					t.Errorf("Expected 2 rows, got %d", len(table.Rows))
				}
				// Check first row
				firstRow := table.Rows[0]
				aExpr := firstRow.Pairs["a"]
				aVal := evaluator.Eval(aExpr, firstRow.Env)
				if intVal, ok := aVal.(*evaluator.Integer); ok {
					if intVal.Value != 2 {
						t.Errorf("Expected a=2, got %d", intVal.Value)
					}
				}
			},
		},
		{
			name: "map with column addition",
			input: `
				let data = table([{x: 5}, {x: 10}])
				data.map(fn(row) { {x: row.x, doubled: row.x * 2} })
			`,
			expectRows: 2,
			validate: func(t *testing.T, result evaluator.Object) {
				table := result.(*evaluator.Table)
				if !containsColumn(table.Columns, "doubled") {
					t.Errorf("Expected 'doubled' column")
				}
			},
		},
		{
			name: "map with filtering-like behavior",
			input: `
				let data = table([{val: 1}, {val: 2}, {val: 3}])
				data.map(fn(row) { {val: row.val, even: row.val % 2 == 0} })
			`,
			expectRows: 3,
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

			if tt.expectError {
				if result.Type() != evaluator.ERROR_OBJ {
					t.Errorf("Expected error, got %s", result.Type())
				}
				return
			}

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			if result.Type() != evaluator.TABLE_OBJ {
				t.Fatalf("Expected TABLE, got %s", result.Type())
			}

			table := result.(*evaluator.Table)
			if table == nil {
				t.Fatal("result is nil")
			}

			if tt.expectRows >= 0 && len(table.Rows) != tt.expectRows {
				t.Errorf("Expected %d rows, got %d", tt.expectRows, len(table.Rows))
			}

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

// TestTableFind tests the find() method
func TestTableFind(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectNull  bool
		validateVal func(*testing.T, evaluator.Object)
	}{
		{
			name: "find existing row",
			input: `
				let data = table([{id: 1, name: "Alice"}, {id: 2, name: "Bob"}])
				data.find(fn(row) { row.id == 2 })
			`,
			validateVal: func(t *testing.T, result evaluator.Object) {
				dict := result.(*evaluator.Dictionary)
				nameExpr := dict.Pairs["name"]
				nameVal := evaluator.Eval(nameExpr, dict.Env)
				if strVal, ok := nameVal.(*evaluator.String); ok {
					if strVal.Value != "Bob" {
						t.Errorf("Expected name='Bob', got %s", strVal.Value)
					}
				}
			},
		},
		{
			name: "find non-existing row",
			input: `
				let data = table([{id: 1}, {id: 2}])
				data.find(fn(row) { row.id == 99 })
			`,
			expectNull: true,
		},
		{
			name: "find first match",
			input: `
				let data = table([{val: 10}, {val: 20}, {val: 30}])
				data.find(fn(row) { row.val > 15 })
			`,
			validateVal: func(t *testing.T, result evaluator.Object) {
				dict := result.(*evaluator.Dictionary)
				valExpr := dict.Pairs["val"]
				valVal := evaluator.Eval(valExpr, dict.Env)
				if intVal, ok := valVal.(*evaluator.Integer); ok {
					if intVal.Value != 20 {
						t.Errorf("Expected val=20 (first match), got %d", intVal.Value)
					}
				}
			},
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
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			if tt.expectNull {
				if result.Type() != evaluator.NULL_OBJ {
					t.Errorf("Expected NULL, got %s", result.Type())
				}
				return
			}

			if result.Type() != evaluator.DICTIONARY_OBJ && result.Type() != evaluator.RECORD_OBJ {
				t.Fatalf("Expected DICTIONARY or RECORD, got %s", result.Type())
			}

			if tt.validateVal != nil {
				tt.validateVal(t, result)
			}
		})
	}
}

// TestTableAny tests the any() method
func TestTableAny(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect bool
	}{
		{
			name: "any true - match exists",
			input: `
				let data = table([{x: 1}, {x: 5}, {x: 10}])
				data.any(fn(row) { row.x > 8 })
			`,
			expect: true,
		},
		{
			name: "any false - no match",
			input: `
				let data = table([{x: 1}, {x: 2}, {x: 3}])
				data.any(fn(row) { row.x > 10 })
			`,
			expect: false,
		},
		{
			name: "any true - multiple matches",
			input: `
				let data = table([{even: true}, {even: false}, {even: true}])
				data.any(fn(row) { row.even })
			`,
			expect: true,
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
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			if result.Type() != evaluator.BOOLEAN_OBJ {
				t.Fatalf("Expected BOOLEAN, got %s", result.Type())
			}

			boolVal := result.(*evaluator.Boolean)
			if boolVal.Value != tt.expect {
				t.Errorf("Expected %v, got %v", tt.expect, boolVal.Value)
			}
		})
	}
}

// TestTableAll tests the all() method
func TestTableAll(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect bool
	}{
		{
			name: "all true - all match",
			input: `
				let data = table([{x: 10}, {x: 20}, {x: 30}])
				data.all(fn(row) { row.x > 5 })
			`,
			expect: true,
		},
		{
			name: "all false - one doesn't match",
			input: `
				let data = table([{x: 10}, {x: 2}, {x: 30}])
				data.all(fn(row) { row.x > 5 })
			`,
			expect: false,
		},
		{
			name: "all true - empty table",
			input: `
				let data = table([])
				data.all(fn(row) { row.x > 100 })
			`,
			expect: true,
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
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			if result.Type() != evaluator.BOOLEAN_OBJ {
				t.Fatalf("Expected BOOLEAN, got %s", result.Type())
			}

			boolVal := result.(*evaluator.Boolean)
			if boolVal.Value != tt.expect {
				t.Errorf("Expected %v, got %v", tt.expect, boolVal.Value)
			}
		})
	}
}

// TestTableUnique tests the unique() method
func TestTableUnique(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		expectRows int
	}{
		{
			name: "unique all columns",
			input: `
				let data = table([{a: 1, b: 2}, {a: 1, b: 2}, {a: 2, b: 3}])
				data.unique()
			`,
			expectRows: 2,
		},
		{
			name: "unique by single column",
			input: `
				let data = table([{id: 1, val: "x"}, {id: 2, val: "y"}, {id: 1, val: "z"}])
				data.unique("id")
			`,
			expectRows: 2,
		},
		{
			name: "unique by multiple columns",
			input: `
				let data = table([
					{a: 1, b: 1, c: "x"},
					{a: 1, b: 2, c: "y"},
					{a: 1, b: 1, c: "z"}
				])
				data.unique(["a", "b"])
			`,
			expectRows: 2,
		},
		{
			name: "unique no duplicates",
			input: `
				let data = table([{id: 1}, {id: 2}, {id: 3}])
				data.unique()
			`,
			expectRows: 3,
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
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			if result.Type() != evaluator.TABLE_OBJ {
				t.Fatalf("Expected TABLE, got %s", result.Type())
			}

			table := result.(*evaluator.Table)
			if len(table.Rows) != tt.expectRows {
				t.Errorf("Expected %d rows, got %d", tt.expectRows, len(table.Rows))
			}
		})
	}
}

// TestTableRenameCol tests the renameCol() method
func TestTableRenameCol(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectError  bool
		validateCols func(*testing.T, evaluator.Object)
	}{
		{
			name: "rename single column",
			input: `
				let data = table([{old_name: 1}, {old_name: 2}])
				data.renameCol("old_name", "new_name")
			`,
			validateCols: func(t *testing.T, result evaluator.Object) {
				table := result.(*evaluator.Table)
				if !containsColumn(table.Columns, "new_name") {
					t.Error("Expected 'new_name' in columns")
				}
				if containsColumn(table.Columns, "old_name") {
					t.Error("Did not expect 'old_name' in columns")
				}
			},
		},
		{
			name: "rename non-existent column",
			input: `
				let data = table([{a: 1}])
				data.renameCol("nonexistent", "new")
			`,
			expectError: true,
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

			if tt.expectError {
				if result.Type() != evaluator.ERROR_OBJ {
					t.Errorf("Expected error, got %s", result.Type())
				}
				return
			}

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			if result.Type() != evaluator.TABLE_OBJ {
				t.Fatalf("Expected TABLE, got %s", result.Type())
			}

			if tt.validateCols != nil {
				tt.validateCols(t, result)
			}
		})
	}
}

// TestTableDropCol tests the dropCol() method
func TestTableDropCol(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectCols    []string
		notExpectCols []string
	}{
		{
			name: "drop single column",
			input: `
				let data = table([{a: 1, b: 2, c: 3}])
				data.dropCol("b")
			`,
			expectCols:    []string{"a", "c"},
			notExpectCols: []string{"b"},
		},
		{
			name: "drop multiple columns",
			input: `
				let data = table([{a: 1, b: 2, c: 3, d: 4}])
				data.dropCol("b", "d")
			`,
			expectCols:    []string{"a", "c"},
			notExpectCols: []string{"b", "d"},
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
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			if result.Type() != evaluator.TABLE_OBJ {
				t.Fatalf("Expected TABLE, got %s", result.Type())
			}

			table := result.(*evaluator.Table)
			for _, col := range tt.expectCols {
				if !containsColumn(table.Columns, col) {
					t.Errorf("Expected column '%s' to exist", col)
				}
			}
			for _, col := range tt.notExpectCols {
				if containsColumn(table.Columns, col) {
					t.Errorf("Did not expect column '%s' to exist", col)
				}
			}
		})
	}
}

// TestTableGroupBy tests the groupBy() method
func TestTableGroupBy(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		expectRows int
		validate   func(*testing.T, evaluator.Object)
	}{
		{
			name: "groupBy single column without aggregation",
			input: `
				let data = table([
					{category: "A", value: 1},
					{category: "B", value: 2},
					{category: "A", value: 3}
				])
				data.groupBy("category")
			`,
			expectRows: 2,
			validate: func(t *testing.T, result evaluator.Object) {
				table := result.(*evaluator.Table)
				if !containsColumn(table.Columns, "category") {
					t.Error("Expected 'category' column")
				}
				if !containsColumn(table.Columns, "rows") {
					t.Error("Expected 'rows' column")
				}
			},
		},
		{
			name: "groupBy with aggregation",
			input: `
				let data = table([
					{category: "A", value: 10},
					{category: "B", value: 20},
					{category: "A", value: 15}
				])
				data.groupBy("category", fn(rows) {
					let count = 0
					for (row in rows) {
						count = count + 1
					}
					{count: count}
				})
			`,
			expectRows: 2,
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
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			if result.Type() != evaluator.TABLE_OBJ {
				t.Fatalf("Expected TABLE, got %s", result.Type())
			}

			table := result.(*evaluator.Table)
			if len(table.Rows) != tt.expectRows {
				t.Errorf("Expected %d rows, got %d", tt.expectRows, len(table.Rows))
			}

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

// TestTableMethodChaining tests that methods can be chained
func TestTableMethodChaining(t *testing.T) {
	input := `
		let data = table([
			{category: "A", value: 10},
			{category: "B", value: 20},
			{category: "A", value: 15},
			{category: "C", value: 5}
		])
		data.where(fn(row) { row.value > 8 }).unique("category")
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
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if result.Type() != evaluator.TABLE_OBJ {
		t.Fatalf("Expected TABLE, got %s", result.Type())
	}

	table := result.(*evaluator.Table)
	// After filtering (value > 8), we have A:10, B:20, A:15
	// After unique(category), we have A:10, B:20 (or A:15, B:20)
	if len(table.Rows) != 2 {
		t.Errorf("Expected 2 rows after chaining, got %d", len(table.Rows))
	}
}

// TestTypedTableCallbacksReceiveRecords tests that callbacks receive Records when table has schema
func TestTypedTableCallbacksReceiveRecords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "where callback receives Record",
			input: `
				@schema Item { name: string, price: int }
				let items = Item([{name: "A", price: 10}, {name: "B", price: 20}])
				// Row type should be record
				items.where(fn(row) { row.type() == "record" }).count()
			`,
			expected: "2", // All rows pass because they're all records
		},
		{
			name: "map callback receives Record",
			input: `
				@schema Item { name: string, price: int }
				let items = Item([{name: "A", price: 10}])
				// Access record to verify it's a Record
				items.map(fn(row) { {name: row.name, isRecord: row.type() == "record"} })[0].isRecord
			`,
			expected: "true",
		},
		{
			name: "find callback receives Record",
			input: `
				@schema Item { name: string, price: int }
				let items = Item([{name: "A", price: 10}, {name: "B", price: 20}])
				let found = items.find(fn(row) { row.type() == "record" and row.name == "B" })
				found.name
			`,
			expected: "B",
		},
		{
			name: "any callback receives Record",
			input: `
				@schema Item { name: string, price: int }
				let items = Item([{name: "A", price: 10}])
				items.any(fn(row) { row.type() == "record" })
			`,
			expected: "true",
		},
		{
			name: "all callback receives Record",
			input: `
				@schema Item { name: string, price: int }
				let items = Item([{name: "A", price: 10}, {name: "B", price: 20}])
				items.all(fn(row) { row.type() == "record" })
			`,
			expected: "true",
		},
		{
			name: "appendCol callback receives Record",
			input: `
				@schema Item { name: string, price: int }
				let items = Item([{name: "A", price: 10}])
				items.appendCol("isRecord", fn(row) { row.type() == "record" })[0].isRecord
			`,
			expected: "true",
		},
		{
			name: "untyped table callback receives Dictionary",
			input: `
				let items = table([{name: "A", price: 10}])
				// Untyped table should pass dictionaries (type is "dictionary")
				items.where(fn(row) { row.type() == "dictionary" }).count()
			`,
			expected: "1",
		},
		{
			name: "destructuring works with Record callbacks",
			input: `
				@schema Item { name: string, price: int }
				let items = Item([{name: "A", price: 10}, {name: "B", price: 20}])
				// Destructuring now works with Records!
				items.where(fn({price}) { price > 15 }).count()
			`,
			expected: "1",
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
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}
