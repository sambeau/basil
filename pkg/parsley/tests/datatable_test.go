package tests

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/parsley"
)

// dataTableSource loads the DataTable component source from the canonical .pars file.
// Cached after first load since the file doesn't change during a test run.
var (
	dataTableSource     string
	dataTableSourceOnce sync.Once
	dataTableSourceErr  error
)

func loadDataTableComponent(t *testing.T) string {
	t.Helper()
	dataTableSourceOnce.Do(func() {
		// Resolve path relative to this test file → ../../../server/prelude/components/data_table.pars
		_, thisFile, _, ok := runtime.Caller(0)
		if !ok {
			dataTableSourceErr = os.ErrNotExist
			return
		}
		componentPath := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "server", "prelude", "components", "data_table.pars")
		src, err := os.ReadFile(componentPath)
		if err != nil {
			dataTableSourceErr = err
			return
		}
		dataTableSource = string(src)
	})
	if dataTableSourceErr != nil {
		t.Fatalf("failed to load DataTable component: %v", dataTableSourceErr)
	}
	return dataTableSource
}

// evalDataTable evaluates the DataTable component source followed by the given test code.
func evalDataTable(t *testing.T, code string) string {
	t.Helper()
	src := loadDataTableComponent(t)
	fullInput := src + "\n" + code

	result, err := parsley.Eval(fullInput)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if result.Value.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Value.Inspect())
	}

	return result.Value.Inspect()
}

// TestDataTableWithTable tests the DataTable component with Table input
func TestDataTableWithTable(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		contains    []string
		notContains []string
	}{
		{
			name: "basic table input",
			input: `
				let t = table([{name: "Alice", age: 30}])
				<DataTable data={t}/>
			`,
			contains: []string{
				"<table",
				"class=\"data-table\"",
				"<thead>",
				"<tbody>",
				"Name",
				"Age",
				"Alice",
				"30",
			},
		},
		{
			name: "backward compatibility with arrays",
			input: `
				<DataTable
					columns={["Name", "Email"]}
					rows={[{name: "Bob", email: "bob@example.com"}]}
					keys={["name", "email"]}
				/>
			`,
			contains: []string{"Name", "Email", "Bob", "bob@example.com"},
		},
		{
			name: "auto-derived headers from column names",
			input: `
				let t = table([{created_at: "2025-01-01", user_name: "Alice"}])
				<DataTable data={t}/>
			`,
			contains: []string{"Created At", "User Name"},
		},
		{
			name: "header override",
			input: `
				let t = table([{created_at: "2025-01-01"}])
				<DataTable data={t} headers={{created_at: "Date Created"}}/>
			`,
			contains:    []string{"Date Created"},
			notContains: []string{">Created At<"},
		},
		{
			name: "column hiding",
			input: `
				let t = table([{name: "Alice", secret: "hidden", age: 30}])
				<DataTable data={t} hide={["secret"]}/>
			`,
			contains:    []string{"Name", "Age", "Alice", "30"},
			notContains: []string{"Secret", "hidden"},
		},
		{
			name: "empty state default message",
			input: `
				let t = table([])
				<DataTable data={t}/>
			`,
			contains: []string{"No data", "data-table-empty"},
		},
		{
			name: "empty state custom message",
			input: `
				let t = table([])
				<DataTable data={t} empty="Nothing to show"/>
			`,
			contains:    []string{"Nothing to show"},
			notContains: []string{"No data"},
		},
		{
			name: "empty state suppressed",
			input: `
				let t = table([])
				<DataTable data={t} empty={false}/>
			`,
			notContains: []string{"No data", "data-table-empty"},
		},
		{
			name: "row header default (first column)",
			input: `
				let t = table([{id: 1, name: "Alice"}])
				<DataTable data={t}/>
			`,
			contains: []string{"<th scope=\"row\""},
		},
		{
			name: "row header configurable",
			input: `
				let t = table([{id: 1, name: "Alice"}])
				<DataTable data={t} rowHeader={1}/>
			`,
			contains: []string{"<th scope=\"row\""},
		},
		{
			name: "row header disabled",
			input: `
				let t = table([{id: 1, name: "Alice"}])
				<DataTable data={t} rowHeader={false}/>
			`,
			notContains: []string{"<th scope=\"row\""},
		},
		{
			name: "alignment classes applied",
			input: `
				let t = table([{name: "Alice"}])
				<DataTable data={t}/>
			`,
			contains: []string{"align-"},
		},
		{
			name: "alignment override",
			input: `
				let t = table([{name: "Alice"}])
				<DataTable data={t} align={{name: "center"}}/>
			`,
			contains: []string{"align-center"},
		},
		{
			name: "caption rendered",
			input: `
				let t = table([{name: "Alice"}])
				<DataTable data={t} caption="User List"/>
			`,
			contains: []string{"<caption>User List</caption>"},
		},
		{
			name: "custom class merged",
			input: `
				let t = table([{name: "Alice"}])
				<DataTable data={t} class="striped"/>
			`,
			contains: []string{"class=\"data-table striped\""},
		},
		{
			name: "id attribute",
			input: `
				let t = table([{name: "Alice"}])
				<DataTable data={t} id="users-table"/>
			`,
			contains: []string{"id=\"users-table\""},
		},
		{
			name: "null value displays em dash",
			input: `
				let t = table([{name: "Alice", note: null}])
				<DataTable data={t}/>
			`,
			contains: []string{"—"},
		},
		{
			name: "custom render function",
			input: `
				let t = table([{name: "Alice", id: 42}])
				<DataTable data={t} render={{name: fn(v, row) { <strong>v</strong> }}}/>
			`,
			contains: []string{"<strong>Alice</strong>"},
		},
		{
			name: "render function receives row",
			input: `
				let t = table([{name: "Alice", id: 42}])
				<DataTable data={t} render={{name: fn(v, row) { <a href={"/user/" + row.id}>v</a> }}}/>
			`,
			contains: []string{"<a href=\"/user/42\">Alice</a>"},
		},
		{
			name: "footer rows",
			input: `
				let t = table([{item: "Widget", qty: 10}, {item: "Gadget", qty: 5}])
				<DataTable data={t} footer={[{item: "Total", qty: 15}]}/>
			`,
			contains: []string{"<tfoot>", "Total", "15"},
		},
		{
			name: "money formatting automatic",
			input: `
				let t = table([{product: "Widget", price: £4999.00}])
				<DataTable data={t}/>
			`,
			contains: []string{"£"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := evalDataTable(t, tt.input)

			for _, want := range tt.contains {
				if !strings.Contains(output, want) {
					t.Errorf("output should contain %q\nGot: %s", want, output)
				}
			}

			for _, notWant := range tt.notContains {
				if strings.Contains(output, notWant) {
					t.Errorf("output should NOT contain %q\nGot: %s", notWant, output)
				}
			}
		})
	}
}

// TestDataTableSchemaIntegration tests DataTable with schema-bound tables
func TestDataTableSchemaIntegration(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		contains    []string
		notContains []string
	}{
		{
			name: "schema title used for headers",
			input: `
				@schema Product {
					sku: string | {title: "Product SKU"}
					price: money | {title: "Unit Price"}
				}
				let t = table([{sku: "ABC-123", price: £29.99}]).as(Product)
				<DataTable data={t}/>
			`,
			contains: []string{"Product SKU", "Unit Price"},
		},
		{
			name: "money column aligned right",
			input: `
				@schema Product {
					name: string
					price: money
				}
				let t = table([{name: "Widget", price: £100.00}]).as(Product)
				<DataTable data={t}/>
			`,
			contains: []string{"align-right"},
		},
		{
			name: "boolean column aligned center",
			input: `
				@schema Item {
					name: string
					active: boolean
				}
				let t = table([{name: "Widget", active: true}]).as(Item)
				<DataTable data={t}/>
			`,
			contains: []string{"align-center", "Yes"},
		},
		{
			name: "boolean false shows No",
			input: `
				@schema Item {
					name: string
					active: boolean
				}
				let t = table([{name: "Widget", active: false}]).as(Item)
				<DataTable data={t}/>
			`,
			contains: []string{"No"},
		},
		{
			name: "datetime formatted with medium",
			input: `
				@schema Event {
					name: string
					date: date
				}
				let t = table([{name: "Meeting", date: datetime("2025-03-15")}]).as(Event)
				<DataTable data={t}/>
			`,
			contains: []string{"Mar"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := evalDataTable(t, tt.input)

			for _, want := range tt.contains {
				if !strings.Contains(output, want) {
					t.Errorf("output should contain %q\nGot: %s", want, output)
				}
			}

			for _, notWant := range tt.notContains {
				if strings.Contains(output, notWant) {
					t.Errorf("output should NOT contain %q\nGot: %s", notWant, output)
				}
			}
		})
	}
}

// TestDataTableEdgeCases tests edge cases and error handling
func TestDataTableEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		contains    []string
		notContains []string
		wantError   bool
	}{
		{
			name: "empty columns array",
			input: `
				<DataTable columns={[]} rows={[]} keys={[]}/>
			`,
			contains: []string{"<table", "<thead>", "<tbody>"},
		},
		{
			name: "data prop takes precedence over rows",
			input: `
				let t = table([{name: "FromTable"}])
				<DataTable data={t} rows={[{name: "FromRows"}]} columns={["Name"]} keys={["name"]}/>
			`,
			contains:    []string{"FromTable"},
			notContains: []string{"FromRows"},
		},
		{
			name: "spread attributes",
			input: `
				let t = table([{name: "Alice"}])
				<DataTable data={t} data-testid="my-table"/>
			`,
			contains: []string{"data-testid=\"my-table\""},
		},
		{
			name: "multiple footer rows",
			input: `
				let t = table([{col1: "a", col2: "b"}])
				<DataTable data={t} footer={[{col1: "Footer1", col2: "F1"}, {col1: "Footer2", col2: "F2"}]}/>
			`,
			contains: []string{"Footer1", "Footer2", "F1", "F2"},
		},
		{
			name: "footer with null values shows empty",
			input: `
				let t = table([{name: "Alice", total: 100}])
				<DataTable data={t} footer={[{name: "Sum", total: 100}]}/>
			`,
			contains: []string{"<tfoot>", "Sum", "100"},
		},
		{
			name: "legacy alignment override matches keys not columns",
			input: `
				<DataTable
					columns={["Full Name", "Score"]}
					rows={[{name: "Alice", score: 99}]}
					keys={["name", "score"]}
					align={{score: "right"}}
				/>
			`,
			contains: []string{"align-right"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := loadDataTableComponent(t)
			fullInput := src + "\n" + tt.input

			result, err := parsley.Eval(fullInput)
			if err != nil {
				if tt.wantError {
					return
				}
				t.Fatalf("parse error: %v", err)
			}

			if result.Value.Type() == evaluator.ERROR_OBJ {
				if tt.wantError {
					return
				}
				t.Fatalf("evaluation error: %s", result.Value.Inspect())
			}

			if tt.wantError {
				t.Fatal("expected error but got none")
			}

			output := result.Value.Inspect()

			for _, want := range tt.contains {
				if !strings.Contains(output, want) {
					t.Errorf("output should contain %q\nGot: %s", want, output)
				}
			}

			for _, notWant := range tt.notContains {
				if strings.Contains(output, notWant) {
					t.Errorf("output should NOT contain %q\nGot: %s", notWant, output)
				}
			}
		})
	}
}
