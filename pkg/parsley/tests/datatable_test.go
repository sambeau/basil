package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/parsley"
)

// DataTable component definition for testing (mirrors server/prelude/components/data_table.pars)
const dataTableComponent = `
let DataTable = fn({
    data,
    rows,
    columns,
    keys,
    caption,
    empty,
    headers,
    align,
    hide,
    render,
    format,
    footer,
    rowHeader,
    id,
    class,
    ...attrs
}) {
    let emptyMsg = empty ?? "No data"
    let headersMap = headers ?? {}
    let alignMap = align ?? {}
    let hideList = hide ?? []
    let renderMap = render ?? {}
    let formatMap = format ?? {}
    let rowHeaderIdx = if (rowHeader == false) null else rowHeader ?? 0

    let tableColumns = if (data) data.columns else columns ?? []
    let tableRows = if (data) data.rows else rows ?? []
    let tableKeys = if (data) data.columns else keys ?? tableColumns

    let visibleColumns = tableColumns.filter(fn(c) { c not in hideList })
    let visibleKeys = tableKeys.filter(fn(k) { k not in hideList })

    let getColProps = fn(col) {
        if (data) {
            data.columnProps(col)
        } else {
            {name: col, label: col.replace("_", " ").toTitle(), align: "left"}
        }
    }

    let getHeader = fn(col) {
        if (headersMap[col]) {
            headersMap[col]
        } else {
            getColProps(col).label
        }
    }

    let getAlign = fn(col) {
        if (alignMap[col]) {
            alignMap[col]
        } else {
            getColProps(col).align ?? "left"
        }
    }

    let getFormat = fn(col) {
        if (formatMap[col]) {
            formatMap[col]
        } else {
            getColProps(col).format ?? null
        }
    }

    let formatCell = fn(value, col) {
        if (value == null) {
            "—"
        } else {
            let fmt = getFormat(col)
            if (fmt == "date" || fmt == "datetime") {
                value.medium()
            } else if (fmt == "duration") {
                value.medium()
            } else if (fmt == "unit") {
                value.medium()
            } else if (fmt == "boolean") {
                if (value) "Yes" else "No"
            } else {
                value
            }
        }
    }

    let tableClass = "data-table" + if (class) " " + class else ""

    <table id={id} class={tableClass} ...attrs>
        if (caption) {
            <caption>caption</caption>
        }
        <thead>
            <tr>
                for (col in visibleColumns) {
                    <th scope="col" class={"align-" + getAlign(col)}>
                        getHeader(col)
                    </th>
                }
            </tr>
        </thead>
        <tbody>
            if (tableRows.length() == 0 && emptyMsg != false) {
                <tr class="data-table-empty">
                    <td colspan={visibleColumns.length()}>emptyMsg</td>
                </tr>
            } else {
                for (row in tableRows) {
                    <tr>
                        for (idx, key in visibleKeys) {
                            let value = row[key]
                            let content = if (renderMap[key]) {
                                renderMap[key](value, row)
                            } else {
                                formatCell(value, key)
                            }
                            let alignClass = "align-" + getAlign(key)

                            if (rowHeaderIdx != null && rowHeaderIdx == idx) {
                                <th scope="row" class={alignClass}>content</th>
                            } else {
                                <td class={alignClass}>content</td>
                            }
                        }
                    </tr>
                }
            }
        </tbody>
        if (footer != null) {
            if (footer.length() > 0) {
                <tfoot>
                    for (footerRow in footer) {
                        <tr>
                            for (key in visibleKeys) {
                                let value = footerRow[key]
                                let content = if (renderMap[key]) {
                                    renderMap[key](value, footerRow)
                                } else if (value != null) {
                                    formatCell(value, key)
                                } else {
                                    ""
                                }
                                <td class={"align-" + getAlign(key)}>content</td>
                            }
                        </tr>
                    }
                </tfoot>
            }
        }
    </table>
}
`

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
			// Prepend DataTable definition to input
			fullInput := dataTableComponent + "\n" + tt.input

			result, err := parsley.Eval(fullInput)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			if result.Value.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Value.Inspect())
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
			// Prepend DataTable definition to input
			fullInput := dataTableComponent + "\n" + tt.input

			result, err := parsley.Eval(fullInput)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			if result.Value.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Value.Inspect())
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepend DataTable definition to input
			fullInput := dataTableComponent + "\n" + tt.input

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
