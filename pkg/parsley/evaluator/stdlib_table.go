package evaluator

import (
	"fmt"
	"html"
	"sort"
	"strconv"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
)

// StdlibBuiltin represents a standard library builtin function that has access to environment
type StdlibBuiltin struct {
	Name string
	Fn   func(args []Object, env *Environment) Object
}

func (sb *StdlibBuiltin) Type() ObjectType { return BUILTIN_OBJ }
func (sb *StdlibBuiltin) Inspect() string  { return fmt.Sprintf("stdlib function: %s", sb.Name) }

// getStdlibModules returns the standard library module registry
// This is a function rather than a var to avoid initialization cycles
func getStdlibModules() map[string]func(*Environment) Object {
	return map[string]func(*Environment) Object{
		"table":  loadTableModule,
		"dev":    loadDevModule,
		"basil":  loadBasilModule,
		"math":   loadMathModule,
		"valid":  loadValidModule,
		"schema": loadSchemaModule,
		"id":     loadIDModule,
		"api":    loadAPIModule,
	}
}

// loadStdlibModule loads a standard library module by name
func loadStdlibModule(name string, env *Environment) Object {
	modules := getStdlibModules()
	loader, ok := modules[name]
	if !ok {
		return newUndefinedError("UNDEF-0005", map[string]any{"Module": name})
	}
	return loader(env)
}

// loadTableModule returns the Table module as a dictionary
func loadTableModule(env *Environment) Object {
	// Return stdlib module dict with table constructor
	// The table export is a TableModule which is both callable and has methods
	return &StdlibModuleDict{
		Exports: map[string]Object{
			"table": &TableModule{},
		},
	}
}

// loadBasilModule returns the basil server context module
// This provides access to request, response, db, auth, etc. in handlers and modules
func loadBasilModule(env *Environment) Object {
	// Get basil context from environment (set by server handler)
	if env.BasilCtx == nil {
		// Return empty module if not in handler context (e.g., CLI, tests)
		return &StdlibModuleDict{
			Exports: map[string]Object{
				"basil": &Dictionary{Pairs: make(map[string]ast.Expression)},
			},
		}
	}

	return &StdlibModuleDict{
		Exports: map[string]Object{
			"basil": env.BasilCtx,
		},
	}
}

// TableModule represents the table constructor with methods like fromDict
// It can be called directly as table(arr) or used as table.fromDict(dict, ...)
type TableModule struct{}

func (tm *TableModule) Type() ObjectType { return BUILTIN_OBJ }
func (tm *TableModule) Inspect() string  { return "table" }

// evalTableModuleMethod handles method calls on the table module (e.g., table.fromDict)
func evalTableModuleMethod(tm *TableModule, method string, args []Object, env *Environment) Object {
	switch method {
	case "fromDict":
		return TableFromDict(args, env)
	default:
		return unknownMethodError(method, "table module", []string{"fromDict"})
	}
}

// StdlibModuleDict represents a standard library module's exported values
type StdlibModuleDict struct {
	Exports map[string]Object
}

func (smd *StdlibModuleDict) Type() ObjectType { return DICTIONARY_OBJ }
func (smd *StdlibModuleDict) Inspect() string {
	keys := make([]string, 0, len(smd.Exports))
	for k := range smd.Exports {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return fmt.Sprintf("StdlibModule{%s}", strings.Join(keys, ", "))
}

// StdlibRoot represents the root of the standard library (import @std)
// It provides introspection for available modules
type StdlibRoot struct {
	Modules []string // List of available module names
}

func (sr *StdlibRoot) Type() ObjectType { return DICTIONARY_OBJ }
func (sr *StdlibRoot) Inspect() string {
	return fmt.Sprintf("@std{%s}", strings.Join(sr.Modules, ", "))
}

// loadStdlibRoot returns the stdlib root with module listing
func loadStdlibRoot() *StdlibRoot {
	modules := getStdlibModules()
	names := make([]string, 0, len(modules))
	for name := range modules {
		names = append(names, name)
	}
	sort.Strings(names)
	return &StdlibRoot{Modules: names}
}

// evalStdlibModuleDestructuring handles destructuring imports from stdlib modules
func evalStdlibModuleDestructuring(pattern *ast.DictDestructuringPattern, mod *StdlibModuleDict, env *Environment, isLet bool, export bool) Object {
	// Process each key in the pattern
	for _, keyPattern := range pattern.Keys {
		keyName := keyPattern.Key.Value

		// Get value from module exports
		var value Object
		if exportedVal, exists := mod.Exports[keyName]; exists {
			value = exportedVal
		} else {
			return newUndefinedError("UNDEF-0006", map[string]any{"Name": keyName})
		}

		// Determine the target variable name (alias or original key)
		targetName := keyName
		if keyPattern.Alias != nil {
			targetName = keyPattern.Alias.Value
		}

		// Assign to environment
		if targetName != "_" {
			if export && isLet {
				env.SetLetExport(targetName, value)
			} else if export {
				env.SetExport(targetName, value)
			} else if isLet {
				env.Set(targetName, value)
			} else {
				env.Update(targetName, value)
			}
		}
	}

	// Destructuring assignments return NULL (excluded from block concatenation)
	return NULL
}

// TableConstructor creates a new Table from an array of dictionaries
func TableConstructor(args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("Table", len(args), 1)
	}

	arr, ok := args[0].(*Array)
	if !ok {
		return newTypeError("TYPE-0012", "Table", "an array", args[0].Type())
	}

	// Handle empty array
	if len(arr.Elements) == 0 {
		return &Table{Rows: []*Dictionary{}, Columns: []string{}}
	}

	// Validate all elements are dictionaries and collect rows
	rows := make([]*Dictionary, 0, len(arr.Elements))
	var columns []string

	for i, elem := range arr.Elements {
		dict, ok := elem.(*Dictionary)
		if !ok {
			return newStructuredError("TYPE-0019", map[string]any{"Function": "Table", "Index": i, "Expected": "dictionary", "Got": elem.Type()})
		}
		rows = append(rows, dict)

		// Get columns from first row
		if i == 0 {
			columns = getDictKeys(dict, env)
		}
	}

	return &Table{Rows: rows, Columns: columns}
}

// TableFromDict creates a Table from a dictionary's entries
// Usage: fromDict(dict) or fromDict(dict, keyColumnName, valueColumnName)
func TableFromDict(args []Object, env *Environment) Object {
	if len(args) != 1 && len(args) != 3 {
		return newArityErrorExact("fromDict", len(args), 1, 3)
	}

	dict, ok := args[0].(*Dictionary)
	if !ok {
		return newTypeError("TYPE-0005", "fromDict", "a dictionary", args[0].Type())
	}

	keyName := "key"
	valueName := "value"
	if len(args) == 3 {
		k, ok := args[1].(*String)
		if !ok {
			return newTypeError("TYPE-0006", "fromDict", "a string (key column name)", args[1].Type())
		}
		v, ok := args[2].(*String)
		if !ok {
			return newTypeError("TYPE-0014", "fromDict", "a string (value column name)", args[2].Type())
		}
		keyName = k.Value
		valueName = v.Value
	}

	// Build rows from dictionary entries
	rows := make([]*Dictionary, 0, len(dict.Pairs))
	for k, expr := range dict.Pairs {
		// Skip internal fields
		if strings.HasPrefix(k, "__") {
			continue
		}
		val := Eval(expr, dict.Env)
		// Create a dictionary for each entry
		entryPairs := map[string]ast.Expression{
			keyName:   objectToExpression(&String{Value: k}),
			valueName: objectToExpression(val),
		}
		rows = append(rows, &Dictionary{Pairs: entryPairs, Env: env})
	}

	return &Table{Rows: rows, Columns: []string{keyName, valueName}}
}

// getDictKeys extracts keys from a dictionary in insertion order
// Falls back to sorted order if KeyOrder is not set
func getDictKeys(dict *Dictionary, env *Environment) []string {
	orderedKeys := dict.Keys()
	// Filter out internal keys like __type
	keys := make([]string, 0, len(orderedKeys))
	for _, k := range orderedKeys {
		if !strings.HasPrefix(k, "__") {
			keys = append(keys, k)
		}
	}
	return keys
}

// getDictValue evaluates and returns a value from a dictionary
func getDictValue(dict *Dictionary, key string) Object {
	expr, ok := dict.Pairs[key]
	if !ok {
		return NULL
	}
	return Eval(expr, dict.Env)
}

// tableWhere filters rows where predicate returns truthy
func tableWhere(t *Table, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("where", len(args), 1)
	}

	fn, ok := args[0].(*Function)
	if !ok {
		return newTypeError("TYPE-0012", "where", "a function", args[0].Type())
	}

	filteredRows := make([]*Dictionary, 0)

	for _, row := range t.Rows {
		// Use extendFunctionEnv to properly bind the row to the function parameter
		extendedEnv := extendFunctionEnv(fn, []Object{row})

		// Evaluate the function body
		var result Object
		for _, stmt := range fn.Body.Statements {
			result = evalStatement(stmt, extendedEnv)
			if returnValue, ok := result.(*ReturnValue); ok {
				result = returnValue.Value
				break
			}
			if isError(result) {
				return result
			}
		}

		// Check if truthy
		if isTruthy(result) {
			filteredRows = append(filteredRows, row)
		}
	}

	newColumns := make([]string, len(t.Columns))
	copy(newColumns, t.Columns)
	return &Table{Rows: filteredRows, Columns: newColumns}
}

// tableOrderBy sorts rows by column(s)
func tableOrderBy(t *Table, args []Object, env *Environment) Object {
	if len(args) < 1 || len(args) > 2 {
		return newArityErrorRange("orderBy", len(args), 1, 2)
	}

	// Parse arguments to determine sort columns and directions
	type sortSpec struct {
		column string
		desc   bool
	}
	var specs []sortSpec

	switch arg := args[0].(type) {
	case *String:
		// Single column: orderBy("name") or orderBy("name", "desc")
		spec := sortSpec{column: arg.Value, desc: false}
		if len(args) == 2 {
			if dir, ok := args[1].(*String); ok {
				spec.desc = strings.ToLower(dir.Value) == "desc"
			} else {
				return newTypeError("TYPE-0006", "orderBy", "a string (direction)", args[1].Type())
			}
		}
		specs = append(specs, spec)

	case *Array:
		// Multi-column: orderBy(["a", "b"]) or orderBy([["a", "asc"], ["b", "desc"]])
		for i, elem := range arg.Elements {
			switch e := elem.(type) {
			case *String:
				specs = append(specs, sortSpec{column: e.Value, desc: false})
			case *Array:
				if len(e.Elements) < 1 || len(e.Elements) > 2 {
					return newValidationError("VAL-0010", map[string]any{"Min": 1, "Max": 2, "Got": len(e.Elements)})
				}
				col, ok := e.Elements[0].(*String)
				if !ok {
					return newStructuredError("TYPE-0020", map[string]any{"Context": "orderBy column name", "Expected": "string", "Got": e.Elements[0].Type()})
				}
				spec := sortSpec{column: col.Value, desc: false}
				if len(e.Elements) == 2 {
					dir, ok := e.Elements[1].(*String)
					if !ok {
						return newStructuredError("TYPE-0020", map[string]any{"Context": "orderBy direction", "Expected": "string", "Got": e.Elements[1].Type()})
					}
					spec.desc = strings.ToLower(dir.Value) == "desc"
				}
				specs = append(specs, spec)
			default:
				return newStructuredError("TYPE-0019", map[string]any{"Function": "orderBy", "Index": i, "Expected": "string or array", "Got": elem.Type()})
			}
		}
	default:
		return newTypeError("TYPE-0005", "orderBy", "a string or array", args[0].Type())
	}

	if len(specs) == 0 {
		return newValidationError("VAL-0011", map[string]any{"Function": "orderBy"})
	}

	// Copy rows for sorting
	sortedRows := make([]*Dictionary, len(t.Rows))
	copy(sortedRows, t.Rows)

	// Sort using stable sort to preserve order of equal elements
	sort.SliceStable(sortedRows, func(i, j int) bool {
		for _, spec := range specs {
			valI := getDictValue(sortedRows[i], spec.column)
			valJ := getDictValue(sortedRows[j], spec.column)

			cmp := compareObjects(valI, valJ)
			if cmp != 0 {
				if spec.desc {
					return cmp > 0
				}
				return cmp < 0
			}
		}
		return false // Equal
	})

	newColumns := make([]string, len(t.Columns))
	copy(newColumns, t.Columns)
	return &Table{Rows: sortedRows, Columns: newColumns}
}

// tableSelect projects specific columns
func tableSelect(t *Table, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("select", len(args), 1)
	}

	columnsArr, ok := args[0].(*Array)
	if !ok {
		return newTypeError("TYPE-0012", "select", "an array of column names", args[0].Type())
	}

	// Extract column names
	columns := make([]string, 0, len(columnsArr.Elements))
	for i, elem := range columnsArr.Elements {
		str, ok := elem.(*String)
		if !ok {
			return newStructuredError("TYPE-0019", map[string]any{"Function": "select", "Index": i, "Expected": "string", "Got": elem.Type()})
		}
		columns = append(columns, str.Value)
	}

	// Project each row to only include selected columns
	projectedRows := make([]*Dictionary, 0, len(t.Rows))
	for _, row := range t.Rows {
		newPairs := make(map[string]ast.Expression)
		for _, col := range columns {
			if expr, ok := row.Pairs[col]; ok {
				newPairs[col] = expr
			} else {
				// Column doesn't exist - use an identifier that evaluates to null
				newPairs[col] = &ast.Identifier{Value: "null"}
			}
		}
		projectedRows = append(projectedRows, &Dictionary{Pairs: newPairs, Env: row.Env})
	}

	return &Table{Rows: projectedRows, Columns: columns}
}

// tableLimit limits the number of rows
func tableLimit(t *Table, args []Object, env *Environment) Object {
	if len(args) < 1 || len(args) > 2 {
		return newArityErrorRange("limit", len(args), 1, 2)
	}

	n, ok := args[0].(*Integer)
	if !ok {
		return newTypeError("TYPE-0005", "limit", "an integer", args[0].Type())
	}
	if n.Value < 0 {
		return newValidationError("VAL-0004", map[string]any{"Method": "limit (count)", "Got": n.Value})
	}

	offset := int64(0)
	if len(args) == 2 {
		off, ok := args[1].(*Integer)
		if !ok {
			return newTypeError("TYPE-0006", "limit", "an integer (offset)", args[1].Type())
		}
		if off.Value < 0 {
			return newValidationError("VAL-0004", map[string]any{"Method": "limit (offset)", "Got": off.Value})
		}
		offset = off.Value
	}

	// Calculate slice bounds
	start := int(offset)
	if start > len(t.Rows) {
		start = len(t.Rows)
	}
	end := start + int(n.Value)
	if end > len(t.Rows) {
		end = len(t.Rows)
	}

	limitedRows := make([]*Dictionary, end-start)
	copy(limitedRows, t.Rows[start:end])

	newColumns := make([]string, len(t.Columns))
	copy(newColumns, t.Columns)
	return &Table{Rows: limitedRows, Columns: newColumns}
}

// tableCount returns the number of rows
func tableCount(t *Table, args []Object, env *Environment) Object {
	if len(args) != 0 {
		return newArityError("count", len(args), 0)
	}
	return &Integer{Value: int64(len(t.Rows))}
}

// tableSum returns the sum of a numeric column
func tableSum(t *Table, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("sum", len(args), 1)
	}

	col, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0012", "sum", "a column name string", args[0].Type())
	}

	var sum float64
	hasFloat := false

	for _, row := range t.Rows {
		val := getDictValue(row, col.Value)
		switch v := val.(type) {
		case *Integer:
			sum += float64(v.Value)
		case *Float:
			sum += v.Value
			hasFloat = true
		case *String:
			// Try to parse string as number
			if f, err := strconv.ParseFloat(v.Value, 64); err == nil {
				sum += f
				if strings.Contains(v.Value, ".") {
					hasFloat = true
				}
			}
			// Skip non-numeric strings
		}
	}

	if hasFloat {
		return &Float{Value: sum}
	}
	return &Integer{Value: int64(sum)}
}

// tableAvg returns the average of a numeric column
func tableAvg(t *Table, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("max", len(args), 1)
	}

	col, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0012", "max", "a column name string", args[0].Type())
	}

	var sum float64
	count := 0

	for _, row := range t.Rows {
		val := getDictValue(row, col.Value)
		switch v := val.(type) {
		case *Integer:
			sum += float64(v.Value)
			count++
		case *Float:
			sum += v.Value
			count++
		case *String:
			// Try to parse string as number
			if f, err := strconv.ParseFloat(v.Value, 64); err == nil {
				sum += f
				count++
			}
			// Skip non-numeric strings
		}
	}

	if count == 0 {
		return NULL
	}

	return &Float{Value: sum / float64(count)}
}

// tableMin returns the minimum value of a column
func tableMin(t *Table, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("min", len(args), 1)
	}

	col, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0012", "min", "a column name string", args[0].Type())
	}

	if len(t.Rows) == 0 {
		return NULL
	}

	var minVal Object = nil
	for _, row := range t.Rows {
		val := getDictValue(row, col.Value)
		if val.Type() == NULL_OBJ {
			continue
		}
		// Try to coerce strings to numbers for comparison
		val = coerceToNumber(val)
		if minVal == nil || compareObjects(val, minVal) < 0 {
			minVal = val
		}
	}

	if minVal == nil {
		return NULL
	}
	return minVal
}

// tableMax returns the maximum value of a column
func tableMax(t *Table, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("max", len(args), 1)
	}

	col, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0012", "max", "a column name string", args[0].Type())
	}

	if len(t.Rows) == 0 {
		return NULL
	}

	var maxVal Object = nil
	for _, row := range t.Rows {
		val := getDictValue(row, col.Value)
		if val.Type() == NULL_OBJ {
			continue
		}
		// Try to coerce strings to numbers for comparison
		val = coerceToNumber(val)
		if maxVal == nil || compareObjects(val, maxVal) > 0 {
			maxVal = val
		}
	}

	if maxVal == nil {
		return NULL
	}
	return maxVal
}

// coerceToNumber attempts to convert a string to a number if possible
func coerceToNumber(obj Object) Object {
	if str, ok := obj.(*String); ok {
		// Try integer first
		if i, err := strconv.ParseInt(str.Value, 10, 64); err == nil {
			return &Integer{Value: i}
		}
		// Try float
		if f, err := strconv.ParseFloat(str.Value, 64); err == nil {
			return &Float{Value: f}
		}
	}
	return obj
}

// tableToHTML renders the table as an HTML table element
func tableToHTML(t *Table, args []Object, env *Environment) Object {
	if len(args) > 1 {
		return newArityErrorRange("toHTML", len(args), 0, 1)
	}

	var sb strings.Builder
	sb.WriteString("<table>\n")

	// Header
	if len(t.Columns) > 0 {
		sb.WriteString("  <thead>\n    <tr>")
		for _, col := range t.Columns {
			sb.WriteString("<th>")
			sb.WriteString(html.EscapeString(col))
			sb.WriteString("</th>")
		}
		sb.WriteString("</tr>\n  </thead>\n")
	}

	// Body
	sb.WriteString("  <tbody>\n")
	for _, row := range t.Rows {
		sb.WriteString("    <tr>")
		for _, col := range t.Columns {
			sb.WriteString("<td>")
			val := getDictValue(row, col)
			if val.Type() != NULL_OBJ {
				sb.WriteString(html.EscapeString(objectToString(val)))
			}
			sb.WriteString("</td>")
		}
		sb.WriteString("</tr>\n")
	}
	sb.WriteString("  </tbody>\n")

	// Footer (optional)
	if len(args) == 1 {
		footerContent, ok := args[0].(*String)
		if !ok {
			return newTypeError("TYPE-0012", "toHTML", "a string (footer content)", args[0].Type())
		}
		if footerContent.Value != "" {
			sb.WriteString("  <tfoot>\n    ")
			sb.WriteString(footerContent.Value)
			sb.WriteString("\n  </tfoot>\n")
		}
	}

	sb.WriteString("</table>")

	return &String{Value: sb.String()}
}

// objectToString converts an object to its string representation for display
func objectToString(obj Object) string {
	switch o := obj.(type) {
	case *String:
		return o.Value
	case *Integer:
		return fmt.Sprintf("%d", o.Value)
	case *Float:
		return fmt.Sprintf("%g", o.Value)
	case *Boolean:
		if o.Value {
			return "true"
		}
		return "false"
	case *Null:
		return ""
	default:
		return obj.Inspect()
	}
}

// tableToCSV renders the table as RFC 4180 compliant CSV
func tableToCSV(t *Table, args []Object, env *Environment) Object {
	if len(args) != 0 {
		return newArityError("toCSV", len(args), 0)
	}

	var sb strings.Builder

	// Header row
	for i, col := range t.Columns {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(csvEscape(col))
	}
	sb.WriteString("\r\n")

	// Data rows
	for _, row := range t.Rows {
		for i, col := range t.Columns {
			if i > 0 {
				sb.WriteString(",")
			}
			val := getDictValue(row, col)
			sb.WriteString(csvEscape(objectToString(val)))
		}
		sb.WriteString("\r\n")
	}

	return &String{Value: sb.String()}
}

// csvEscape escapes a value for CSV output per RFC 4180
func csvEscape(s string) string {
	needsQuoting := strings.ContainsAny(s, ",\"\r\n")
	if !needsQuoting {
		return s
	}
	// Escape quotes by doubling them
	escaped := strings.ReplaceAll(s, "\"", "\"\"")
	return "\"" + escaped + "\""
}

// tableToMarkdown renders the table as a GitHub Flavored Markdown table
func tableToMarkdown(t *Table, args []Object, env *Environment) Object {
	if len(args) != 0 {
		return newArityError("toMarkdown", len(args), 0)
	}

	if len(t.Columns) == 0 {
		return &String{Value: ""}
	}

	var sb strings.Builder

	// Header row
	sb.WriteString("|")
	for _, col := range t.Columns {
		sb.WriteString(" ")
		sb.WriteString(markdownEscape(col))
		sb.WriteString(" |")
	}
	sb.WriteString("\n")

	// Separator row
	sb.WriteString("|")
	for range t.Columns {
		sb.WriteString(" --- |")
	}
	sb.WriteString("\n")

	// Data rows
	for _, row := range t.Rows {
		sb.WriteString("|")
		for _, col := range t.Columns {
			sb.WriteString(" ")
			val := getDictValue(row, col)
			sb.WriteString(markdownEscape(objectToString(val)))
			sb.WriteString(" |")
		}
		sb.WriteString("\n")
	}

	return &String{Value: sb.String()}
}

// markdownEscape escapes special characters in Markdown table cells
func markdownEscape(s string) string {
	// Escape pipe characters which are table delimiters
	s = strings.ReplaceAll(s, "|", "\\|")
	// Escape newlines as they break table structure
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}

// tableRows returns the underlying array of dictionaries
func tableRows(t *Table) Object {
	elements := make([]Object, len(t.Rows))
	for i, row := range t.Rows {
		elements[i] = row
	}
	return &Array{Elements: elements}
}

// tableColumns returns the column names as an array
func tableColumns(t *Table) Object {
	elements := make([]Object, len(t.Columns))
	for i, col := range t.Columns {
		elements[i] = &String{Value: col}
	}
	return &Array{Elements: elements}
}

// tableRowCount returns the number of rows in the table
func tableRowCount(t *Table, args []Object, env *Environment) Object {
	if len(args) != 0 {
		return newArityError("rowCount", len(args), 0)
	}
	return &Integer{Value: int64(len(t.Rows))}
}

// tableColumnCount returns the number of columns in the table
func tableColumnCount(t *Table, args []Object, env *Environment) Object {
	if len(args) != 0 {
		return newArityError("columnCount", len(args), 0)
	}
	return &Integer{Value: int64(len(t.Columns))}
}

// ============================================================================
// Table Insert/Append Methods
// ============================================================================

// tableAppendRow appends a row to the table, returns new table
func tableAppendRow(t *Table, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("appendRow", len(args), 1)
	}

	row, ok := args[0].(*Dictionary)
	if !ok {
		return newTypeError("TYPE-0012", "appendRow", "a dictionary", args[0].Type())
	}

	// Validate row has required columns (for non-empty tables)
	if len(t.Columns) > 0 {
		rowKeys := row.Keys()
		if err := validateRowColumns(rowKeys, t.Columns, "appendRow"); err != nil {
			return err
		}
	}

	// Create new table with row appended
	newRows := make([]*Dictionary, len(t.Rows)+1)
	copy(newRows, t.Rows)
	newRows[len(t.Rows)] = row

	// Determine columns (from new row if table was empty)
	newColumns := t.Columns
	if len(newColumns) == 0 {
		newColumns = row.Keys()
	}

	return &Table{Rows: newRows, Columns: newColumns}
}

// tableInsertRowAt inserts a row at a specific index, returns new table
func tableInsertRowAt(t *Table, args []Object, env *Environment) Object {
	if len(args) != 2 {
		return newArityError("insertRowAt", len(args), 2)
	}

	idxObj, ok := args[0].(*Integer)
	if !ok {
		return newTypeError("TYPE-0012", "insertRowAt", "an integer", args[0].Type())
	}

	row, ok := args[1].(*Dictionary)
	if !ok {
		return newTypeError("TYPE-0012", "insertRowAt", "a dictionary", args[1].Type())
	}

	idx := int(idxObj.Value)
	length := len(t.Rows)

	// Handle negative indices
	if idx < 0 {
		idx = length + idx
	}

	// Bounds check: index must be in [0, length]
	if idx < 0 || idx > length {
		return newIndexError("INDEX-0001", map[string]any{"Index": idxObj.Value, "Length": length})
	}

	// Validate row has required columns (for non-empty tables)
	if len(t.Columns) > 0 {
		rowKeys := row.Keys()
		if err := validateRowColumns(rowKeys, t.Columns, "insertRowAt"); err != nil {
			return err
		}
	}

	// Create new table with row inserted
	newRows := make([]*Dictionary, length+1)
	copy(newRows[:idx], t.Rows[:idx])
	newRows[idx] = row
	copy(newRows[idx+1:], t.Rows[idx:])

	// Determine columns (from new row if table was empty)
	newColumns := t.Columns
	if len(newColumns) == 0 {
		newColumns = row.Keys()
	}

	return &Table{Rows: newRows, Columns: newColumns}
}

// tableAppendCol appends a column to the table, returns new table
// Accepts either values array or function: appendCol(name, values) or appendCol(name, fn)
func tableAppendCol(t *Table, args []Object, env *Environment) Object {
	if len(args) != 2 {
		return newArityError("appendCol", len(args), 2)
	}

	colName, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0012", "appendCol", "a string", args[0].Type())
	}

	// Check column doesn't already exist
	for _, col := range t.Columns {
		if col == colName.Value {
			return newStructuredError("TYPE-0023", map[string]any{"Key": colName.Value})
		}
	}

	// Get column values (either from array or by computing with function)
	values, err := getColumnValues(args[1], t, env, "appendCol")
	if err != nil {
		return err
	}

	// Create new table with column appended
	return createTableWithNewColumn(t, colName.Value, values, len(t.Columns), env)
}

// tableInsertColAfter inserts a column after an existing column, returns new table
func tableInsertColAfter(t *Table, args []Object, env *Environment) Object {
	if len(args) != 3 {
		return newArityError("insertColAfter", len(args), 3)
	}

	afterCol, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0012", "insertColAfter", "a string (existing column)", args[0].Type())
	}

	colName, ok := args[1].(*String)
	if !ok {
		return newTypeError("TYPE-0012", "insertColAfter", "a string (new column)", args[1].Type())
	}

	// Find position of existing column
	insertPos := -1
	for i, col := range t.Columns {
		if col == afterCol.Value {
			insertPos = i + 1 // Insert after this column
			break
		}
	}
	if insertPos == -1 {
		return newIndexError("INDEX-0005", map[string]any{"Key": afterCol.Value})
	}

	// Check new column doesn't already exist
	for _, col := range t.Columns {
		if col == colName.Value {
			return newStructuredError("TYPE-0023", map[string]any{"Key": colName.Value})
		}
	}

	// Get column values
	values, err := getColumnValues(args[2], t, env, "insertColAfter")
	if err != nil {
		return err
	}

	return createTableWithNewColumn(t, colName.Value, values, insertPos, env)
}

// tableInsertColBefore inserts a column before an existing column, returns new table
func tableInsertColBefore(t *Table, args []Object, env *Environment) Object {
	if len(args) != 3 {
		return newArityError("insertColBefore", len(args), 3)
	}

	beforeCol, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0012", "insertColBefore", "a string (existing column)", args[0].Type())
	}

	colName, ok := args[1].(*String)
	if !ok {
		return newTypeError("TYPE-0012", "insertColBefore", "a string (new column)", args[1].Type())
	}

	// Find position of existing column
	insertPos := -1
	for i, col := range t.Columns {
		if col == beforeCol.Value {
			insertPos = i // Insert at this position (before)
			break
		}
	}
	if insertPos == -1 {
		return newIndexError("INDEX-0005", map[string]any{"Key": beforeCol.Value})
	}

	// Check new column doesn't already exist
	for _, col := range t.Columns {
		if col == colName.Value {
			return newStructuredError("TYPE-0023", map[string]any{"Key": colName.Value})
		}
	}

	// Get column values
	values, err := getColumnValues(args[2], t, env, "insertColBefore")
	if err != nil {
		return err
	}

	return createTableWithNewColumn(t, colName.Value, values, insertPos, env)
}

// validateRowColumns checks that row keys match table columns
func validateRowColumns(rowKeys, tableColumns []string, methodName string) *Error {
	if len(rowKeys) != len(tableColumns) {
		return newStructuredError("TYPE-0020", map[string]any{
			"Context":  methodName + " row",
			"Expected": fmt.Sprintf("%d columns", len(tableColumns)),
			"Got":      fmt.Sprintf("%d columns", len(rowKeys)),
		})
	}

	// Check all required columns exist
	rowKeySet := make(map[string]bool)
	for _, k := range rowKeys {
		rowKeySet[k] = true
	}
	for _, col := range tableColumns {
		if !rowKeySet[col] {
			return newIndexError("INDEX-0005", map[string]any{"Key": col})
		}
	}

	return nil
}

// getColumnValues gets column values from either an array or a function
func getColumnValues(arg Object, t *Table, env *Environment, methodName string) ([]Object, *Error) {
	switch v := arg.(type) {
	case *Array:
		// Values array - must match row count
		if len(v.Elements) != len(t.Rows) {
			return nil, newStructuredError("TYPE-0020", map[string]any{
				"Context":  methodName + " values",
				"Expected": fmt.Sprintf("%d values", len(t.Rows)),
				"Got":      fmt.Sprintf("%d values", len(v.Elements)),
			})
		}
		return v.Elements, nil

	case *Function:
		// Compute values by calling function with each row
		values := make([]Object, len(t.Rows))
		for i, row := range t.Rows {
			extendedEnv := extendFunctionEnv(v, []Object{row})
			var result Object
			for _, stmt := range v.Body.Statements {
				result = evalStatement(stmt, extendedEnv)
				if returnValue, ok := result.(*ReturnValue); ok {
					result = returnValue.Value
					break
				}
				if isError(result) {
					return nil, result.(*Error)
				}
			}
			values[i] = result
		}
		return values, nil

	default:
		return nil, newTypeError("TYPE-0020", methodName, "an array or function", arg.Type())
	}
}

// createTableWithNewColumn creates a new table with a column inserted at the given position
func createTableWithNewColumn(t *Table, colName string, values []Object, insertPos int, env *Environment) *Table {
	// Create new column order
	newColumns := make([]string, len(t.Columns)+1)
	copy(newColumns[:insertPos], t.Columns[:insertPos])
	newColumns[insertPos] = colName
	copy(newColumns[insertPos+1:], t.Columns[insertPos:])

	// Create new rows with the new column
	newRows := make([]*Dictionary, len(t.Rows))
	for i, row := range t.Rows {
		// Copy existing pairs
		newPairs := make(map[string]ast.Expression, len(row.Pairs)+1)
		for k, v := range row.Pairs {
			newPairs[k] = v
		}
		// Add new column value
		newPairs[colName] = objectToExpression(values[i])

		// Create new key order with column inserted at correct position
		newKeyOrder := make([]string, len(newColumns))
		copy(newKeyOrder, newColumns)

		newRows[i] = &Dictionary{
			Pairs:    newPairs,
			KeyOrder: newKeyOrder,
			Env:      env,
		}
	}

	return &Table{Rows: newRows, Columns: newColumns}
}

// EvalTableMethod dispatches method calls on Table objects
func EvalTableMethod(t *Table, method string, args []Object, env *Environment) Object {
	switch method {
	case "where":
		return tableWhere(t, args, env)
	case "orderBy":
		return tableOrderBy(t, args, env)
	case "select":
		return tableSelect(t, args, env)
	case "limit":
		return tableLimit(t, args, env)
	case "count":
		return tableCount(t, args, env)
	case "sum":
		return tableSum(t, args, env)
	case "avg":
		return tableAvg(t, args, env)
	case "min":
		return tableMin(t, args, env)
	case "max":
		return tableMax(t, args, env)
	case "toHTML":
		return tableToHTML(t, args, env)
	case "toCSV":
		return tableToCSV(t, args, env)
	case "toMarkdown":
		return tableToMarkdown(t, args, env)
	case "appendRow":
		return tableAppendRow(t, args, env)
	case "insertRowAt":
		return tableInsertRowAt(t, args, env)
	case "appendCol":
		return tableAppendCol(t, args, env)
	case "insertColAfter":
		return tableInsertColAfter(t, args, env)
	case "insertColBefore":
		return tableInsertColBefore(t, args, env)
	case "rowCount":
		return tableRowCount(t, args, env)
	case "columnCount":
		return tableColumnCount(t, args, env)
	default:
		return unknownMethodError(method, "Table", []string{
			"where", "orderBy", "select", "limit", "count", "sum", "avg", "min", "max",
			"toHTML", "toCSV", "toMarkdown", "appendRow", "insertRowAt", "appendCol", "insertColAfter", "insertColBefore",
			"rowCount", "columnCount",
		})
	}
}

// EvalTableProperty handles property access on Table objects
func EvalTableProperty(t *Table, property string) Object {
	switch property {
	case "rows":
		return tableRows(t)
	case "columns":
		return tableColumns(t)
	default:
		return newUndefinedError("UNDEF-0004", map[string]any{"Property": property, "Type": "Table"})
	}
}
