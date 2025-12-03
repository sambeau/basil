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

// Standard library module registry
var stdlibModules = map[string]func(*Environment) Object{
	"table": loadTableModule,
}

// loadStdlibModule loads a standard library module by name
func loadStdlibModule(name string, env *Environment) Object {
	loader, ok := stdlibModules[name]
	if !ok {
		return newError("unknown standard library module: @std/%s", name)
	}
	return loader(env)
}

// loadTableModule returns the Table module as a dictionary
func loadTableModule(env *Environment) Object {
	// Return stdlib module dict with table constructor
	return &StdlibModuleDict{
		Exports: map[string]Object{
			"table": &StdlibBuiltin{Name: "table", Fn: TableConstructor},
		},
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
			return newError("module does not export '%s'", keyName)
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

	return mod
}

// TableConstructor creates a new Table from an array of dictionaries
func TableConstructor(args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newError("wrong number of arguments to `Table`. got=%d, want=1", len(args))
	}

	arr, ok := args[0].(*Array)
	if !ok {
		return newError("argument to `Table` must be an array, got %s", args[0].Type())
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
			return newError("Table elements must be dictionaries, element %d is %s", i, elem.Type())
		}
		rows = append(rows, dict)

		// Get columns from first row
		if i == 0 {
			columns = getDictKeys(dict, env)
		}
	}

	return &Table{Rows: rows, Columns: columns}
}

// getDictKeys extracts keys from a dictionary in sorted order
func getDictKeys(dict *Dictionary, env *Environment) []string {
	keys := make([]string, 0, len(dict.Pairs))
	for k := range dict.Pairs {
		// Skip internal keys like __type
		if !strings.HasPrefix(k, "__") {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
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
		return newError("wrong number of arguments to `where`. got=%d, want=1", len(args))
	}

	fn, ok := args[0].(*Function)
	if !ok {
		return newError("argument to `where` must be a function, got %s", args[0].Type())
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
		return newError("wrong number of arguments to `orderBy`. got=%d, want=1 or 2", len(args))
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
				return newError("second argument to `orderBy` must be a string direction, got %s", args[1].Type())
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
					return newError("orderBy column spec must have 1 or 2 elements, got %d", len(e.Elements))
				}
				col, ok := e.Elements[0].(*String)
				if !ok {
					return newError("orderBy column name must be a string")
				}
				spec := sortSpec{column: col.Value, desc: false}
				if len(e.Elements) == 2 {
					dir, ok := e.Elements[1].(*String)
					if !ok {
						return newError("orderBy direction must be a string")
					}
					spec.desc = strings.ToLower(dir.Value) == "desc"
				}
				specs = append(specs, spec)
			default:
				return newError("orderBy array element %d must be a string or array", i)
			}
		}
	default:
		return newError("first argument to `orderBy` must be a string or array, got %s", args[0].Type())
	}

	if len(specs) == 0 {
		return newError("orderBy requires at least one column")
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
		return newError("wrong number of arguments to `select`. got=%d, want=1", len(args))
	}

	columnsArr, ok := args[0].(*Array)
	if !ok {
		return newError("argument to `select` must be an array of column names, got %s", args[0].Type())
	}

	// Extract column names
	columns := make([]string, 0, len(columnsArr.Elements))
	for i, elem := range columnsArr.Elements {
		str, ok := elem.(*String)
		if !ok {
			return newError("select column names must be strings, element %d is %s", i, elem.Type())
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
		return newError("wrong number of arguments to `limit`. got=%d, want=1 or 2", len(args))
	}

	n, ok := args[0].(*Integer)
	if !ok {
		return newError("first argument to `limit` must be an integer, got %s", args[0].Type())
	}
	if n.Value < 0 {
		return newError("limit count cannot be negative")
	}

	offset := int64(0)
	if len(args) == 2 {
		off, ok := args[1].(*Integer)
		if !ok {
			return newError("second argument to `limit` must be an integer, got %s", args[1].Type())
		}
		if off.Value < 0 {
			return newError("limit offset cannot be negative")
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
		return newError("wrong number of arguments to `count`. got=%d, want=0", len(args))
	}
	return &Integer{Value: int64(len(t.Rows))}
}

// tableSum returns the sum of a numeric column
func tableSum(t *Table, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newError("wrong number of arguments to `sum`. got=%d, want=1", len(args))
	}

	col, ok := args[0].(*String)
	if !ok {
		return newError("argument to `sum` must be a column name string, got %s", args[0].Type())
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
		return newError("wrong number of arguments to `avg`. got=%d, want=1", len(args))
	}

	col, ok := args[0].(*String)
	if !ok {
		return newError("argument to `avg` must be a column name string, got %s", args[0].Type())
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
		return newError("wrong number of arguments to `min`. got=%d, want=1", len(args))
	}

	col, ok := args[0].(*String)
	if !ok {
		return newError("argument to `min` must be a column name string, got %s", args[0].Type())
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
		return newError("wrong number of arguments to `max`. got=%d, want=1", len(args))
	}

	col, ok := args[0].(*String)
	if !ok {
		return newError("argument to `max` must be a column name string, got %s", args[0].Type())
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
	if len(args) != 0 {
		return newError("wrong number of arguments to `toHTML`. got=%d, want=0", len(args))
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
		return newError("wrong number of arguments to `toCSV`. got=%d, want=0", len(args))
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

// tableRows returns the underlying array of dictionaries
func tableRows(t *Table) Object {
	elements := make([]Object, len(t.Rows))
	for i, row := range t.Rows {
		elements[i] = row
	}
	return &Array{Elements: elements}
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
	default:
		return newError("unknown method '%s' for Table", method)
	}
}

// EvalTableProperty handles property access on Table objects
func EvalTableProperty(t *Table, property string) Object {
	switch property {
	case "rows":
		return tableRows(t)
	default:
		return newError("unknown property '%s' for Table", property)
	}
}
