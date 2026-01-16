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
		"math":   loadMathModule,
		"valid":  loadValidModule,
		"schema": loadSchemaModule,
		"id":     loadIDModule,
		"api":    loadAPIModule,
		"mdDoc":  loadMdDocModule,
		"html":   loadHTMLModule,
	}
}

// loadStdlibModule loads a standard library module by name
func loadStdlibModule(name string, env *Environment) Object {
	if name == "basil" {
		return newImportError("IMPORT-0006", map[string]any{
			"Module":      name,
			"Replacement": "Use @basil/http or @basil/auth instead.",
		})
	}

	modules := getStdlibModules()
	loader, ok := modules[name]
	if !ok {
		return newUndefinedError("UNDEF-0005", map[string]any{"Module": name})
	}
	return loader(env)
}

// loadTableModule returns the Table module as a dictionary
// DEPRECATED: Prefer using @table literal syntax directly, e.g.:
//
//	@table [["name", "age"], ["Alice", 30]]
//	@table [ {name: "Alice", age: 30} ]
//
// The @std/table import is maintained for backward compatibility.
func loadTableModule(env *Environment) Object {
	// Return stdlib module dict with table constructor
	// The table export is a TableModule which is both callable and has methods
	return &StdlibModuleDict{
		Exports: map[string]Object{
			"table": &TableModule{},
		},
	}
}

// TableModule represents the table constructor with methods like fromDict
// It can be called directly as table(arr) or used as table.fromDict(dict, ...)
// DEPRECATED: Prefer @table literal syntax. The module is kept for backward compatibility.
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

// DynamicAccessor is a value that resolves lazily from the current environment.
// Used for @basil/http and @basil/auth exports to ensure request/session-scoped
// values are fresh even when imported at module scope.
type DynamicAccessor struct {
	Name     string                    // Display name (e.g., "query", "session")
	Resolver func(*Environment) Object // Function to resolve the actual value
}

func (da *DynamicAccessor) Type() ObjectType { return "DYNAMIC_ACCESSOR" }
func (da *DynamicAccessor) Inspect() string  { return fmt.Sprintf("<dynamic:%s>", da.Name) }

// Resolve returns the current value by calling the resolver with the given environment.
func (da *DynamicAccessor) Resolve(env *Environment) Object {
	if da.Resolver == nil {
		return NULL
	}
	result := da.Resolver(env)
	if result == nil {
		return NULL
	}
	return result
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

// BasilRoot represents the root of the basil namespace (import @basil)
// It provides introspection for available basil modules
type BasilRoot struct {
	Modules []string // List of available module names
}

func (br *BasilRoot) Type() ObjectType { return DICTIONARY_OBJ }
func (br *BasilRoot) Inspect() string {
	return fmt.Sprintf("@basil{%s}", strings.Join(br.Modules, ", "))
}

// getBasilModules returns the basil namespace module registry
func getBasilModules() map[string]func(*Environment) Object {
	return map[string]func(*Environment) Object{
		"http": loadBasilHTTPModule,
		"auth": loadBasilAuthModule,
	}
}

// loadBasilModule loads a basil namespace module by name
func loadBasilModule(name string, env *Environment) Object {
	modules := getBasilModules()
	loader, ok := modules[name]
	if !ok {
		return newUndefinedError("UNDEF-0007", map[string]any{"Module": name})
	}
	return loader(env)
}

// loadBasilRoot returns the basil root with module listing
func loadBasilRoot() *BasilRoot {
	modules := getBasilModules()
	names := make([]string, 0, len(modules))
	for name := range modules {
		names = append(names, name)
	}
	sort.Strings(names)
	return &BasilRoot{Modules: names}
}

// getBasilCtxDict safely returns the basil context dictionary from the environment.
// It searches up the environment chain to find the first (closest) non-nil BasilCtx,
// which ensures that request-scoped values (from @basil/http) are always current.
// ApplyFunctionWithEnv sets BasilCtx on the extended environment from the caller's env,
// so we find the freshest context by looking at the closest environment first.
func getBasilCtxDict(env *Environment) *Dictionary {
	if env == nil {
		return nil
	}

	// Walk up the environment chain and return the FIRST (closest) non-nil BasilCtx
	// This ensures we get the caller's context (set by ApplyFunctionWithEnv) rather
	// than the stale context from a cached module's closure.
	current := env
	for current != nil {
		if current.BasilCtx != nil {
			if dict, ok := current.BasilCtx.(*Dictionary); ok {
				return dict
			}
		}
		current = current.outer
	}

	return nil
}

// evalDictValue evaluates a dictionary field in the dictionary's own environment if present.
func evalDictValue(dict *Dictionary, key string, env *Environment) Object {
	if dict == nil {
		return NULL
	}
	expr, ok := dict.Pairs[key]
	if !ok {
		return NULL
	}
	targetEnv := dict.Env
	if targetEnv == nil {
		targetEnv = env
	}
	val := Eval(expr, targetEnv)
	if val == nil {
		return NULL
	}
	return val
}

func ensureObject(val Object) Object {
	if val == nil {
		return NULL
	}
	return val
}

// loadBasilHTTPModule returns the HTTP-related basil module
// Exports: request, response, route, method
// All exports are DynamicAccessors to ensure fresh values per-request
// even when imported at module scope.
// Note: query has been removed; use @params instead.
func loadBasilHTTPModule(env *Environment) Object {
	return &StdlibModuleDict{
		Exports: map[string]Object{
			"request": &DynamicAccessor{
				Name: "request",
				Resolver: func(e *Environment) Object {
					basilDict := getBasilCtxDict(e)
					httpObj := evalDictValue(basilDict, "http", e)
					httpDict, _ := httpObj.(*Dictionary)
					return ensureObject(evalDictValue(httpDict, "request", e))
				},
			},
			"response": &DynamicAccessor{
				Name: "response",
				Resolver: func(e *Environment) Object {
					basilDict := getBasilCtxDict(e)
					httpObj := evalDictValue(basilDict, "http", e)
					httpDict, _ := httpObj.(*Dictionary)
					return ensureObject(evalDictValue(httpDict, "response", e))
				},
			},
			"route": &DynamicAccessor{
				Name: "route",
				Resolver: func(e *Environment) Object {
					basilDict := getBasilCtxDict(e)
					httpObj := evalDictValue(basilDict, "http", e)
					httpDict, _ := httpObj.(*Dictionary)
					requestObj := evalDictValue(httpDict, "request", e)
					if reqDict, ok := requestObj.(*Dictionary); ok {
						routeObj := evalDictValue(reqDict, "route", e)
						if routeObj == NULL {
							// Backwards compatibility
							routeObj = evalDictValue(reqDict, "subpath", e)
						}
						return ensureObject(routeObj)
					}
					return NULL
				},
			},
			"method": &DynamicAccessor{
				Name: "method",
				Resolver: func(e *Environment) Object {
					basilDict := getBasilCtxDict(e)
					httpObj := evalDictValue(basilDict, "http", e)
					httpDict, _ := httpObj.(*Dictionary)
					requestObj := evalDictValue(httpDict, "request", e)
					if reqDict, ok := requestObj.(*Dictionary); ok {
						return ensureObject(evalDictValue(reqDict, "method", e))
					}
					return NULL
				},
			},
		},
	}
}

// loadBasilAuthModule returns the auth/database/session basil module
// Exports: session, auth (auth context), user (auth.user shortcut)
// All exports are DynamicAccessors for per-request freshness.
// Note: Database access has been moved to @DB magic variable.
func loadBasilAuthModule(env *Environment) Object {
	return &StdlibModuleDict{
		Exports: map[string]Object{
			"session": &DynamicAccessor{
				Name: "session",
				Resolver: func(e *Environment) Object {
					basilDict := getBasilCtxDict(e)
					return ensureObject(evalDictValue(basilDict, "session", e))
				},
			},
			"auth": &DynamicAccessor{
				Name: "auth",
				Resolver: func(e *Environment) Object {
					basilDict := getBasilCtxDict(e)
					return ensureObject(evalDictValue(basilDict, "auth", e))
				},
			},
			"user": &DynamicAccessor{
				Name: "user",
				Resolver: func(e *Environment) Object {
					basilDict := getBasilCtxDict(e)
					authObj := evalDictValue(basilDict, "auth", e)
					if authDict, ok := authObj.(*Dictionary); ok {
						return ensureObject(evalDictValue(authDict, "user", e))
					}
					return NULL
				},
			},
		},
	}
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

// TableConstructor creates a new Table from an array of dictionaries.
// Validates that all rows have the same columns (rectangular shape).
func TableConstructor(args []Object, env *Environment) Object {
	// Handle 0 args (empty table)
	if len(args) == 0 {
		return &Table{Rows: []*Dictionary{}, Columns: []string{}}
	}

	if len(args) != 1 {
		return newArityErrorRange("Table", len(args), 0, 1)
	}

	arr, ok := args[0].(*Array)
	if !ok {
		return newStructuredError("TABLE-0001", map[string]any{
			"Got": string(args[0].Type()),
		})
	}

	// Handle empty array
	if len(arr.Elements) == 0 {
		return &Table{Rows: []*Dictionary{}, Columns: []string{}}
	}

	// Validate all elements are dictionaries and collect rows
	rows := make([]*Dictionary, 0, len(arr.Elements))
	var columns []string
	var columnSet map[string]bool

	for i, elem := range arr.Elements {
		dict, ok := elem.(*Dictionary)
		if !ok {
			return newStructuredError("TABLE-0002", map[string]any{
				"Row": i + 1,
				"Got": string(elem.Type()),
			})
		}

		// Get columns from first row
		if i == 0 {
			columns = getDictKeys(dict, env)
			columnSet = make(map[string]bool, len(columns))
			for _, col := range columns {
				columnSet[col] = true
			}
		} else {
			// Validate subsequent rows have same columns
			rowKeys := getDictKeys(dict, env)

			// Check for missing columns
			var missing []string
			for _, col := range columns {
				found := false
				for _, k := range rowKeys {
					if k == col {
						found = true
						break
					}
				}
				if !found {
					missing = append(missing, col)
				}
			}

			// Check for extra columns
			var extra []string
			for _, k := range rowKeys {
				if !columnSet[k] {
					extra = append(extra, k)
				}
			}

			if len(missing) > 0 {
				return newStructuredError("TABLE-0003", map[string]any{
					"Row":     i + 1,
					"Missing": strings.Join(missing, ", "),
				})
			}

			if len(extra) > 0 {
				return newStructuredError("TABLE-0004", map[string]any{
					"Row":   i + 1,
					"Extra": strings.Join(extra, ", "),
				})
			}
		}

		rows = append(rows, dict)
	}

	return &Table{Rows: rows, Columns: columns}
}

// evalTableLiteral evaluates @table [...] or @table(Schema) [...] literals
func evalTableLiteral(node *ast.TableLiteral, env *Environment) Object {
	// Handle empty table
	if len(node.Rows) == 0 {
		table := &Table{
			Rows:    []*Dictionary{},
			Columns: node.Columns, // May be empty for @table []
		}
		// If schema specified, attach it
		if node.Schema != nil {
			schemaObj, ok := env.Get(node.Schema.Value)
			if !ok {
				return newUndefinedError("UNDEF-0001", map[string]any{
					"Name": node.Schema.Value,
				})
			}
			schema, ok := schemaObj.(*DSLSchema)
			if !ok {
				return newTypeError("TYPE-0005", "@table", "a schema", schemaObj.Type())
			}
			table.Schema = schema
		}
		return table
	}

	// Evaluate each row
	rows := make([]*Dictionary, 0, len(node.Rows))
	var schema *DSLSchema

	// If schema specified, look it up
	if node.Schema != nil {
		schemaObj, ok := env.Get(node.Schema.Value)
		if !ok {
			return newUndefinedError("UNDEF-0001", map[string]any{
				"Name": node.Schema.Value,
			})
		}
		schema, ok = schemaObj.(*DSLSchema)
		if !ok {
			return newTypeError("TYPE-0005", "@table", "a schema", schemaObj.Type())
		}
	}

	for i, rowNode := range node.Rows {
		// Evaluate the dictionary literal
		rowObj := evalDictionaryLiteral(rowNode, env)
		if isError(rowObj) {
			return rowObj
		}

		dict, ok := rowObj.(*Dictionary)
		if !ok {
			return newStructuredError("TABLE-0002", map[string]any{
				"Row":  i + 1,
				"Type": rowObj.Type(),
			})
		}

		// If schema specified, create a Record from the dictionary
		// This applies defaults, filters unknown fields, and casts types
		if schema != nil {
			// CreateRecord returns *Record directly (never errors during creation)
			rec := CreateRecord(schema, dict, env)

			// For @table(Schema) literal, validate immediately and error on invalid
			// This preserves backward compatibility where missing required fields error
			validated := ValidateRecord(rec, env)
			if len(validated.Errors) > 0 {
				// Return first error for the row
				for field, fieldErr := range validated.Errors {
					if fieldErr != nil {
						return newStructuredError("TABLE-0005", map[string]any{
							"Row":   i,
							"Field": field,
						})
					}
				}
			}

			rowDict := rec.ToDictionary()
			rowDict.Env = env
			rows = append(rows, rowDict)
		} else {
			rows = append(rows, dict)
		}
	}

	// For typed tables, use sorted schema field names for columns
	columns := node.Columns
	if schema != nil && len(schema.Fields) > 0 {
		columns = make([]string, 0, len(schema.Fields))
		for name := range schema.Fields {
			columns = append(columns, name)
		}
		sort.Strings(columns)
	}

	return &Table{
		Rows:    rows,
		Columns: columns,
		Schema:  schema,
	}
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
// Uses copy-on-chain: first call in chain creates copy, subsequent calls reuse it
func tableWhere(t *Table, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("where", len(args), 1)
	}

	fn, ok := args[0].(*Function)
	if !ok {
		return newTypeError("TYPE-0012", "where", "a function", args[0].Type())
	}

	// Get chain copy (creates one if needed, reuses if already in chain)
	result := t.ensureChainCopy()
	filteredRows := make([]*Dictionary, 0)

	for _, row := range result.Rows {
		// Use extendFunctionEnv to properly bind the row to the function parameter
		extendedEnv := extendFunctionEnv(fn, []Object{row})

		// Evaluate the function body
		var evalResult Object
		for _, stmt := range fn.Body.Statements {
			evalResult = evalStatement(stmt, extendedEnv)
			if returnValue, ok := evalResult.(*ReturnValue); ok {
				evalResult = returnValue.Value
				break
			}
			if isError(evalResult) {
				return evalResult
			}
		}

		// Check if truthy
		if isTruthy(evalResult) {
			filteredRows = append(filteredRows, row)
		}
	}

	// Mutate the chain copy in place
	result.Rows = filteredRows
	return result
}

// tableOrderBy sorts rows by column(s)
// Uses copy-on-chain: first call in chain creates copy, subsequent calls reuse it
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

	// Get chain copy (creates one if needed, reuses if already in chain)
	result := t.ensureChainCopy()

	// Sort in place on the chain copy
	sort.SliceStable(result.Rows, func(i, j int) bool {
		for _, spec := range specs {
			valI := getDictValue(result.Rows[i], spec.column)
			valJ := getDictValue(result.Rows[j], spec.column)

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

	return result
}

// tableSelect projects specific columns
// Uses copy-on-chain: first call in chain creates copy, subsequent calls reuse it
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

	// Get chain copy (creates one if needed, reuses if already in chain)
	result := t.ensureChainCopy()

	// Project each row to only include selected columns
	projectedRows := make([]*Dictionary, 0, len(result.Rows))
	for _, row := range result.Rows {
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

	// Mutate the chain copy in place
	result.Rows = projectedRows
	result.Columns = columns
	return result
}

// tableLimit limits the number of rows
// Uses copy-on-chain: first call in chain creates copy, subsequent calls reuse it
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

	// Get chain copy (creates one if needed, reuses if already in chain)
	result := t.ensureChainCopy()

	// Calculate slice bounds
	start := int(offset)
	if start > len(result.Rows) {
		start = len(result.Rows)
	}
	end := start + int(n.Value)
	if end > len(result.Rows) {
		end = len(result.Rows)
	}

	// Mutate the chain copy in place
	result.Rows = result.Rows[start:end]
	return result
}

// tableOffset skips the first n rows
// Uses copy-on-chain: first call in chain creates copy, subsequent calls reuse it
func tableOffset(t *Table, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("offset", len(args), 1)
	}

	n, ok := args[0].(*Integer)
	if !ok {
		return newTypeError("TYPE-0005", "offset", "an integer", args[0].Type())
	}
	if n.Value < 0 {
		return newValidationError("VAL-0004", map[string]any{"Method": "offset", "Got": n.Value})
	}

	// Get chain copy (creates one if needed, reuses if already in chain)
	result := t.ensureChainCopy()

	// Calculate slice start
	start := int(n.Value)
	if start > len(result.Rows) {
		start = len(result.Rows)
	}

	// Mutate the chain copy in place
	result.Rows = result.Rows[start:]
	return result
}

// tableToArray returns the table rows as an array of dictionaries
func tableToArray(t *Table, args []Object, env *Environment) Object {
	if len(args) != 0 {
		return newArityError("toArray", len(args), 0)
	}

	elements := make([]Object, len(t.Rows))
	for i, row := range t.Rows {
		elements[i] = row
	}
	return &Array{Elements: elements}
}

// tableCopy returns an explicit deep copy of the table (not a chain copy)
func tableCopy(t *Table, args []Object, env *Environment) Object {
	if len(args) != 0 {
		return newArityError("copy", len(args), 0)
	}
	return t.Copy()
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
	var moneyCurrency string
	var moneyScale int8
	var moneySum int64
	hasMoney := false

	for _, row := range t.Rows {
		val := getDictValue(row, col.Value)
		switch v := val.(type) {
		case *Money:
			if !hasMoney {
				// First money value - set currency and scale
				moneyCurrency = v.Currency
				moneyScale = v.Scale
				hasMoney = true
			} else if v.Currency != moneyCurrency {
				// Mixed currencies - error
				return newStructuredError("CALC-0001", map[string]any{"Message": fmt.Sprintf("Cannot sum mixed currencies: %s and %s", moneyCurrency, v.Currency)})
			}
			moneySum += v.Amount
		case *Integer:
			if hasMoney {
				return newStructuredError("CALC-0001", map[string]any{"Message": "Cannot mix money and numeric types in sum"})
			}
			sum += float64(v.Value)
		case *Float:
			if hasMoney {
				return newStructuredError("CALC-0001", map[string]any{"Message": "Cannot mix money and numeric types in sum"})
			}
			sum += v.Value
			hasFloat = true
		case *String:
			if hasMoney {
				return newStructuredError("CALC-0001", map[string]any{"Message": "Cannot mix money and numeric types in sum"})
			}
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

	if hasMoney {
		return &Money{Amount: moneySum, Currency: moneyCurrency, Scale: moneyScale}
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
	var moneyCurrency string
	var moneyScale int8
	var moneySum int64
	hasMoney := false

	for _, row := range t.Rows {
		val := getDictValue(row, col.Value)
		switch v := val.(type) {
		case *Money:
			if !hasMoney {
				// First money value - set currency and scale
				moneyCurrency = v.Currency
				moneyScale = v.Scale
				hasMoney = true
			} else if v.Currency != moneyCurrency {
				// Mixed currencies - error
				return newStructuredError("CALC-0001", map[string]any{"Message": fmt.Sprintf("Cannot average mixed currencies: %s and %s", moneyCurrency, v.Currency)})
			}
			moneySum += v.Amount
			count++
		case *Integer:
			if hasMoney {
				return newStructuredError("CALC-0001", map[string]any{"Message": "Cannot mix money and numeric types in average"})
			}
			sum += float64(v.Value)
			count++
		case *Float:
			if hasMoney {
				return newStructuredError("CALC-0001", map[string]any{"Message": "Cannot mix money and numeric types in average"})
			}
			sum += v.Value
			count++
		case *String:
			if hasMoney {
				return newStructuredError("CALC-0001", map[string]any{"Message": "Cannot mix money and numeric types in average"})
			}
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

	if hasMoney {
		avgAmount := moneySum / int64(count)
		return &Money{Amount: avgAmount, Currency: moneyCurrency, Scale: moneyScale}
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
		// Check if it's a string (legacy format) or dictionary
		if footerStr, ok := args[0].(*String); ok {
			// String footer - just insert raw HTML
			if footerStr.Value != "" {
				sb.WriteString("  <tfoot>\n    ")
				sb.WriteString(footerStr.Value)
				sb.WriteString("\n  </tfoot>\n")
			}
		} else if footerDict, ok := args[0].(*Dictionary); ok {
			// Dictionary footer - generate row with values for specified columns
			sb.WriteString("  <tfoot>\n    <tr>")

			// Track consecutive empty cells for colspan
			emptyCount := 0

			for i, col := range t.Columns {
				val := getDictValue(footerDict, col)

				// Check if cell should be empty (NULL or Error for undefined property)
				isEmpty := val.Type() == NULL_OBJ || val.Type() == ERROR_OBJ

				if isEmpty {
					// Empty cell - increment counter
					emptyCount++

					// If this is the last column, flush the empty cells
					if i == len(t.Columns)-1 && emptyCount > 0 {
						if emptyCount == 1 {
							sb.WriteString("<td></td>")
						} else {
							sb.WriteString(fmt.Sprintf("<td colspan=\"%d\"></td>", emptyCount))
						}
					}
				} else {
					// Non-empty cell - flush any accumulated empty cells first
					if emptyCount > 0 {
						if emptyCount == 1 {
							sb.WriteString("<td></td>")
						} else {
							sb.WriteString(fmt.Sprintf("<td colspan=\"%d\"></td>", emptyCount))
						}
						emptyCount = 0
					}

					// Write the cell with value
					sb.WriteString("<td>")
					// For String values, treat as raw HTML (like string footer does)
					// For other types, escape for safety
					if strVal, ok := val.(*String); ok {
						sb.WriteString(strVal.Value)
					} else {
						sb.WriteString(html.EscapeString(objectToString(val)))
					}
					sb.WriteString("</td>")
				}
			}

			sb.WriteString("</tr>\n  </tfoot>\n")
		} else {
			return newTypeError("TYPE-0012", "toHTML", "a string or dictionary (footer content)", args[0].Type())
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
	case *Dictionary:
		// Check if it's a special object type
		if typeExpr, ok := o.Pairs["__type"]; ok {
			typeVal := Eval(typeExpr, o.Env)
			if typeStr, ok := typeVal.(*String); ok {
				switch typeStr.Value {
				case "path":
					// Convert path dictionary to path string
					return pathDictToString(o)
				case "datetime":
					// It's a datetime - format it nicely
					if isoExpr, ok := o.Pairs["iso"]; ok {
						isoVal := Eval(isoExpr, o.Env)
						if isoStr, ok := isoVal.(*String); ok {
							// Parse the ISO string to determine the format
							isoString := isoStr.Value

							// Check what kind of datetime it is based on the kind field
							kind := "datetime" // default
							if kindExpr, ok := o.Pairs["kind"]; ok {
								kindVal := Eval(kindExpr, o.Env)
								if kindStr, ok := kindVal.(*String); ok {
									kind = kindStr.Value
								}
							}

							// Format based on kind
							switch kind {
							case "date":
								// Date only: 2025-12-24
								if len(isoString) >= 10 {
									return isoString[:10]
								}
							case "time", "time_seconds":
								// Time only: 14:30:05 or 14:30
								if strings.Contains(isoString, "T") {
									parts := strings.Split(isoString, "T")
									if len(parts) == 2 {
										timePart := strings.TrimSuffix(parts[1], "Z")
										return timePart
									}
								}
							default:
								// Full datetime: 2025-12-24 14:30:05
								// Convert from ISO format (2025-12-24T14:30:05Z) to readable format
								isoString = strings.TrimSuffix(isoString, "Z")
								isoString = strings.Replace(isoString, "T", " ", 1)
								return isoString
							}
							return isoString
						}
					}
				}
			}
		}
		// Not a special type or couldn't format - fall through to default
		return obj.Inspect()
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

// tableToBox renders the table using box-drawing characters (like SQL CLI output)
func tableToBox(t *Table, args []Object, env *Environment) Object {
	// Parse options
	opts, optErr := parseBoxOptions(args)
	if optErr != nil {
		return optErr
	}

	if len(t.Columns) == 0 {
		return &String{Value: ""}
	}

	// Build rows as strings
	rows := make([][]string, len(t.Rows))
	for i, row := range t.Rows {
		rowData := make([]string, len(t.Columns))
		for j, col := range t.Columns {
			val := getDictValue(row, col)
			rowData[j] = objectToString(val)
		}
		rows[i] = rowData
	}

	// Use shared BoxRenderer with options
	br := NewBoxRenderer()
	br.Style = opts.Style
	br.Align = opts.Align
	br.Title = opts.Title
	br.MaxWidth = opts.MaxWidth
	return &String{Value: br.RenderTable(t.Columns, rows)}
}

// tableToJSON renders the table as a JSON array of objects
func tableToJSON(t *Table, args []Object, env *Environment) Object {
	if len(args) != 0 {
		return newArityError("toJSON", len(args), 0)
	}

	var sb strings.Builder
	sb.WriteString("[")

	for i, row := range t.Rows {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString("\n  {")

		for j, col := range t.Columns {
			if j > 0 {
				sb.WriteString(",")
			}
			sb.WriteString("\n    \"")
			sb.WriteString(jsonEscape(col))
			sb.WriteString("\": ")

			val := getDictValue(row, col)
			sb.WriteString(objectToJSON(val))
		}

		sb.WriteString("\n  }")
	}

	if len(t.Rows) > 0 {
		sb.WriteString("\n")
	}
	sb.WriteString("]")

	return &String{Value: sb.String()}
}

// objectToJSON converts an object to its JSON representation
func objectToJSON(obj Object) string {
	switch o := obj.(type) {
	case *String:
		return "\"" + jsonEscape(o.Value) + "\""
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
		return "null"
	case *Array:
		var sb strings.Builder
		sb.WriteString("[")
		for i, elem := range o.Elements {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(objectToJSON(elem))
		}
		sb.WriteString("]")
		return sb.String()
	case *Record:
		// Record encodes as its data fields (like Dictionary)
		return objectToJSON(o.ToDictionary())
	case *Dictionary:
		// Dictionary encodes as JSON object
		var sb strings.Builder
		sb.WriteString("{")
		first := true
		for _, key := range o.KeyOrder {
			if expr, ok := o.Pairs[key]; ok {
				if !first {
					sb.WriteString(", ")
				}
				first = false
				sb.WriteString("\"")
				sb.WriteString(jsonEscape(key))
				sb.WriteString("\": ")
				val := Eval(expr, o.Env)
				sb.WriteString(objectToJSON(val))
			}
		}
		sb.WriteString("}")
		return sb.String()
	default:
		// For other types, use string representation in quotes
		return "\"" + jsonEscape(obj.Inspect()) + "\""
	}
}

// jsonEscape escapes special characters for JSON strings
func jsonEscape(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
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

// tableColumn returns all values from a specific column as an array
func tableColumn(t *Table, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("column", len(args), 1)
	}

	colName, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0012", "column", "a string (column name)", args[0].Type())
	}

	// Check if column exists
	columnExists := false
	for _, col := range t.Columns {
		if col == colName.Value {
			columnExists = true
			break
		}
	}
	if !columnExists {
		return newIndexError("INDEX-0005", map[string]any{
			"Key": colName.Value,
		})
	}

	// Extract column values
	values := make([]Object, len(t.Rows))
	for i, row := range t.Rows {
		val := getDictValue(row, colName.Value)
		values[i] = val
	}

	return &Array{Elements: values}
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

// tableMap applies a function to each row and returns a new table
// Schema preservation: If fn returns Records with same schema → preserve; if different schema → adopt new; if plain dicts → clear schema
func tableMap(t *Table, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("map", len(args), 1)
	}

	fn, ok := args[0].(*Function)
	if !ok {
		return newTypeError("TYPE-0012", "map", "a function", args[0].Type())
	}

	// Get chain copy
	result := t.ensureChainCopy()
	mappedRows := make([]*Dictionary, 0, len(result.Rows))
	
	var detectedSchema *DSLSchema
	schemaConsistent := true

	for i, row := range result.Rows {
		extendedEnv := extendFunctionEnv(fn, []Object{row})

		// Evaluate the function body
		var evalResult Object
		for _, stmt := range fn.Body.Statements {
			evalResult = evalStatement(stmt, extendedEnv)
			if returnValue, ok := evalResult.(*ReturnValue); ok {
				evalResult = returnValue.Value
				break
			}
			if isError(evalResult) {
				return evalResult
			}
		}

		// Convert result to dictionary
		var rowDict *Dictionary
		if record, ok := evalResult.(*Record); ok {
			// It's a Record - check schema consistency
			if i == 0 {
				detectedSchema = record.Schema
			} else if detectedSchema != nil && record.Schema != detectedSchema {
				// Different schema detected
				schemaConsistent = false
				detectedSchema = record.Schema
			}
			// Convert Record.Data to Dictionary
			rowDict = &Dictionary{
				Pairs:    record.Data,
				KeyOrder: record.KeyOrder,
				Env:      record.Env,
			}
		} else if dict, ok := evalResult.(*Dictionary); ok {
			// Plain dictionary - clear schema
			rowDict = dict
			detectedSchema = nil
			schemaConsistent = false
		} else {
			return newTypeError("TYPE-0012", "map callback", "a dictionary or Record", evalResult.Type())
		}

		mappedRows = append(mappedRows, rowDict)
	}

	// Update columns from first row
	var newColumns []string
	if len(mappedRows) > 0 {
		newColumns = make([]string, 0, len(mappedRows[0].Pairs))
		for col := range mappedRows[0].Pairs {
			newColumns = append(newColumns, col)
		}
	}

	// Schema preservation logic
	var finalSchema *DSLSchema
	if schemaConsistent && detectedSchema != nil {
		finalSchema = detectedSchema
	}

	result.Rows = mappedRows
	result.Columns = newColumns
	result.Schema = finalSchema
	return result
}

// tableFind returns the first row that matches the predicate, or null
func tableFind(t *Table, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("find", len(args), 1)
	}

	fn, ok := args[0].(*Function)
	if !ok {
		return newTypeError("TYPE-0012", "find", "a function", args[0].Type())
	}

	for _, row := range t.Rows {
		extendedEnv := extendFunctionEnv(fn, []Object{row})

		// Evaluate the function body
		var evalResult Object
		for _, stmt := range fn.Body.Statements {
			evalResult = evalStatement(stmt, extendedEnv)
			if returnValue, ok := evalResult.(*ReturnValue); ok {
				evalResult = returnValue.Value
				break
			}
			if isError(evalResult) {
				return evalResult
			}
		}

		// Check if truthy
		if isTruthy(evalResult) {
			// Return as Record if table has schema
			if t.Schema != nil {
				return &Record{
					Schema:    t.Schema,
					Data:      row.Pairs,
					KeyOrder:  row.KeyOrder,
					Validated: t.FromDB,
					Errors:    make(map[string]*RecordError),
					Env:       row.Env,
				}
			}
			return row
		}
	}

	return NULL
}

// tableAny returns true if any row matches the predicate
func tableAny(t *Table, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("any", len(args), 1)
	}

	fn, ok := args[0].(*Function)
	if !ok {
		return newTypeError("TYPE-0012", "any", "a function", args[0].Type())
	}

	for _, row := range t.Rows {
		extendedEnv := extendFunctionEnv(fn, []Object{row})

		// Evaluate the function body
		var evalResult Object
		for _, stmt := range fn.Body.Statements {
			evalResult = evalStatement(stmt, extendedEnv)
			if returnValue, ok := evalResult.(*ReturnValue); ok {
				evalResult = returnValue.Value
				break
			}
			if isError(evalResult) {
				return evalResult
			}
		}

		// Check if truthy
		if isTruthy(evalResult) {
			return &Boolean{Value: true}
		}
	}

	return &Boolean{Value: false}
}

// tableAll returns true if all rows match the predicate
func tableAll(t *Table, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("all", len(args), 1)
	}

	fn, ok := args[0].(*Function)
	if !ok {
		return newTypeError("TYPE-0012", "all", "a function", args[0].Type())
	}

	for _, row := range t.Rows {
		extendedEnv := extendFunctionEnv(fn, []Object{row})

		// Evaluate the function body
		var evalResult Object
		for _, stmt := range fn.Body.Statements {
			evalResult = evalStatement(stmt, extendedEnv)
			if returnValue, ok := evalResult.(*ReturnValue); ok {
				evalResult = returnValue.Value
				break
			}
			if isError(evalResult) {
				return evalResult
			}
		}

		// Check if falsy
		if !isTruthy(evalResult) {
			return &Boolean{Value: false}
		}
	}

	return &Boolean{Value: true}
}

// tableUnique returns a table with duplicate rows removed
// If column names are provided, only those columns are used for uniqueness
func tableUnique(t *Table, args []Object, env *Environment) Object {
	if len(args) > 1 {
		return newArityErrorRange("unique", len(args), 0, 1)
	}

	// Get chain copy
	result := t.ensureChainCopy()

	// Determine which columns to use for uniqueness
	var keyCols []string
	if len(args) == 1 {
		// Column names provided
		switch arg := args[0].(type) {
		case *String:
			keyCols = []string{arg.Value}
		case *Array:
			keyCols = make([]string, 0, len(arg.Elements))
			for _, elem := range arg.Elements {
				str, ok := elem.(*String)
				if !ok {
					return newTypeError("TYPE-0012", "unique", "string column names", elem.Type())
				}
				keyCols = append(keyCols, str.Value)
			}
		default:
			return newTypeError("TYPE-0012", "unique", "a string or array of strings", args[0].Type())
		}
	} else {
		// No args - use all columns
		keyCols = result.Columns
	}

	// Track seen rows using a map of key string -> bool
	seen := make(map[string]bool)
	uniqueRows := make([]*Dictionary, 0)

	for _, row := range result.Rows {
		// Build key from specified columns
		var keyParts []string
		for _, col := range keyCols {
			expr, exists := row.Pairs[col]
			if exists && expr != nil {
				// Evaluate expression to get value
				obj := Eval(expr, row.Env)
				if obj != nil && obj.Type() != ERROR_OBJ {
					keyParts = append(keyParts, obj.Inspect())
				} else {
					keyParts = append(keyParts, "NULL")
				}
			} else {
				keyParts = append(keyParts, "NULL")
			}
		}
		key := strings.Join(keyParts, "\x00") // Use null byte as separator

		if !seen[key] {
			seen[key] = true
			uniqueRows = append(uniqueRows, row)
		}
	}

	result.Rows = uniqueRows
	return result
}

// tableRenameCol renames a column in the table
func tableRenameCol(t *Table, args []Object, env *Environment) Object {
	if len(args) != 2 {
		return newArityError("renameCol", len(args), 2)
	}

	oldName, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0012", "renameCol", "a string (old column name)", args[0].Type())
	}

	newName, ok := args[1].(*String)
	if !ok {
		return newTypeError("TYPE-0012", "renameCol", "a string (new column name)", args[1].Type())
	}

	// Check if old column exists
	found := false
	for _, col := range t.Columns {
		if col == oldName.Value {
			found = true
			break
		}
	}
	if !found {
		return newValidationError("VAL-0012", map[string]any{
			"Column": oldName.Value,
			"Table":  "table",
		})
	}

	// Get chain copy
	result := t.ensureChainCopy()

	// Rename in columns list
	newColumns := make([]string, len(result.Columns))
	for i, col := range result.Columns {
		if col == oldName.Value {
			newColumns[i] = newName.Value
		} else {
			newColumns[i] = col
		}
	}
	result.Columns = newColumns

	// Rename in each row
	newRows := make([]*Dictionary, len(result.Rows))
	for i, row := range result.Rows {
		newPairs := make(map[string]ast.Expression, len(row.Pairs))
		newKeyOrder := make([]string, len(row.KeyOrder))
		for j, k := range row.KeyOrder {
			if k == oldName.Value {
				newKeyOrder[j] = newName.Value
				newPairs[newName.Value] = row.Pairs[k]
			} else {
				newKeyOrder[j] = k
				newPairs[k] = row.Pairs[k]
			}
		}
		newRows[i] = &Dictionary{
			Pairs:    newPairs,
			KeyOrder: newKeyOrder,
			Env:      row.Env,
		}
	}
	result.Rows = newRows

	return result
}

// tableDropCol removes one or more columns from the table
func tableDropCol(t *Table, args []Object, env *Environment) Object {
	if len(args) == 0 {
		return newArityError("dropCol", len(args), 1)
	}

	// Collect column names to drop
	colsToDrop := make(map[string]bool)
	for _, arg := range args {
		str, ok := arg.(*String)
		if !ok {
			return newTypeError("TYPE-0012", "dropCol", "string column name(s)", arg.Type())
		}
		colsToDrop[str.Value] = true
	}

	// Get chain copy
	result := t.ensureChainCopy()

	// Filter columns list
	newColumns := make([]string, 0, len(result.Columns))
	for _, col := range result.Columns {
		if !colsToDrop[col] {
			newColumns = append(newColumns, col)
		}
	}
	result.Columns = newColumns

	// Filter each row
	newRows := make([]*Dictionary, len(result.Rows))
	for i, row := range result.Rows {
		newPairs := make(map[string]ast.Expression)
		newKeyOrder := make([]string, 0)
		for _, k := range row.KeyOrder {
			if !colsToDrop[k] {
				newKeyOrder = append(newKeyOrder, k)
				newPairs[k] = row.Pairs[k]
			}
		}
		newRows[i] = &Dictionary{
			Pairs:    newPairs,
			KeyOrder: newKeyOrder,
			Env:      row.Env,
		}
	}
	result.Rows = newRows

	return result
}

// tableGroupBy groups rows by column value(s) and returns a Table with group key and arrays
func tableGroupBy(t *Table, args []Object, env *Environment) Object {
	if len(args) < 1 || len(args) > 2 {
		return newArityErrorRange("groupBy", len(args), 1, 2)
	}

	// Parse group-by column(s)
	var groupCols []string
	switch arg := args[0].(type) {
	case *String:
		groupCols = []string{arg.Value}
	case *Array:
		groupCols = make([]string, 0, len(arg.Elements))
		for _, elem := range arg.Elements {
			str, ok := elem.(*String)
			if !ok {
				return newTypeError("TYPE-0012", "groupBy", "string column name(s)", elem.Type())
			}
			groupCols = append(groupCols, str.Value)
		}
	default:
		return newTypeError("TYPE-0012", "groupBy", "a string or array of strings", args[0].Type())
	}

	// Optional: aggregation function
	var aggFn *Function
	if len(args) == 2 {
		fn, ok := args[1].(*Function)
		if !ok {
			return newTypeError("TYPE-0012", "groupBy", "a function (aggregation)", args[1].Type())
		}
		aggFn = fn
	}

	// Group rows by key
	type groupKey struct {
		key    string
		values []Object  // Store actual values, not strings
	}
	groups := make(map[string]*groupKey)
	groupOrder := []string{} // Preserve insertion order
	groupedRows := make(map[string][]*Dictionary)

	for _, row := range t.Rows {
		// Build group key
		var keyParts []string
		var valueParts []Object
		for _, col := range groupCols {
			expr, exists := row.Pairs[col]
			if exists && expr != nil {
				// Evaluate expression to get value
				obj := Eval(expr, row.Env)
				if obj != nil && obj.Type() != ERROR_OBJ {
					keyParts = append(keyParts, obj.Inspect())
					valueParts = append(valueParts, obj)
				} else {
					keyParts = append(keyParts, "NULL")
					valueParts = append(valueParts, NULL)
				}
			} else {
				keyParts = append(keyParts, "NULL")
				valueParts = append(valueParts, NULL)
			}
		}
		key := strings.Join(keyParts, "\x00")

		if _, exists := groups[key]; !exists {
			groups[key] = &groupKey{key: key, values: valueParts}
			groupOrder = append(groupOrder, key)
			groupedRows[key] = make([]*Dictionary, 0)
		}
		groupedRows[key] = append(groupedRows[key], row)
	}

	// Build result table
	var resultRows []*Dictionary
	var resultColumns []string

	if aggFn != nil {
		// With aggregation function - call it for each group
		for _, key := range groupOrder {
			groupData := groupedRows[key]
			groupArray := &Array{Elements: make([]Object, len(groupData))}
			for i, row := range groupData {
				groupArray.Elements[i] = row
			}

			// Call aggregation function with the group array
			extendedEnv := extendFunctionEnv(aggFn, []Object{groupArray})
			var evalResult Object
			for _, stmt := range aggFn.Body.Statements {
				evalResult = evalStatement(stmt, extendedEnv)
				if returnValue, ok := evalResult.(*ReturnValue); ok {
					evalResult = returnValue.Value
					break
				}
				if isError(evalResult) {
					return evalResult
				}
			}

			// Result should be a dictionary or value
			resultPairs := make(map[string]ast.Expression)
			resultKeyOrder := make([]string, 0)
			
			// Add group key columns
			gk := groups[key]
			for i, col := range groupCols {
				if i < len(gk.values) {
					resultPairs[col] = &ast.ObjectLiteralExpression{Obj: gk.values[i]}
					resultKeyOrder = append(resultKeyOrder, col)
				}
			}

			// Add aggregation result
			if dict, ok := evalResult.(*Dictionary); ok {
				// Merge aggregation result
				for _, k := range dict.KeyOrder {
					resultPairs[k] = dict.Pairs[k]
					resultKeyOrder = append(resultKeyOrder, k)
				}
			} else {
				// Single value - use "value" as column name
				resultPairs["value"] = &ast.ObjectLiteralExpression{Obj: evalResult}
				resultKeyOrder = append(resultKeyOrder, "value")
			}

			resultRows = append(resultRows, &Dictionary{
				Pairs:    resultPairs,
				KeyOrder: resultKeyOrder,
				Env:      env,
			})
		}

		// Determine columns from first row
		if len(resultRows) > 0 {
			resultColumns = resultRows[0].KeyOrder
		}
	} else {
		// No aggregation - return group keys and rows array
		resultColumns = append(groupCols, "rows")

		for _, key := range groupOrder {
			groupData := groupedRows[key]
			gk := groups[key]
			
			resultPairs := make(map[string]ast.Expression)
			resultKeyOrder := make([]string, 0)
			
			// Add group key columns
			for i, col := range groupCols {
				if i < len(gk.values) {
					resultPairs[col] = &ast.ObjectLiteralExpression{Obj: gk.values[i]}
					resultKeyOrder = append(resultKeyOrder, col)
				}
			}

			// Add rows array
			rowsArray := &Array{Elements: make([]Object, len(groupData))}
			for i, row := range groupData {
				rowsArray.Elements[i] = row
			}
			resultPairs["rows"] = &ast.ObjectLiteralExpression{Obj: rowsArray}
			resultKeyOrder = append(resultKeyOrder, "rows")

			resultRows = append(resultRows, &Dictionary{
				Pairs:    resultPairs,
				KeyOrder: resultKeyOrder,
				Env:      env,
			})
		}
	}

	return &Table{
		Rows:        resultRows,
		Columns:     resultColumns,
		Schema:      nil, // Grouped tables don't preserve schema
		isChainCopy: false,
	}
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
	case "toBox":
		return tableToBox(t, args, env)
	case "toJSON":
		return tableToJSON(t, args, env)
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
	case "column":
		return tableColumn(t, args, env)
	case "toArray":
		return tableToArray(t, args, env)
	case "copy":
		return tableCopy(t, args, env)
	case "offset":
		return tableOffset(t, args, env)
	// Phase 3: Table validation methods for typed tables
	case "as":
		return tableAs(t, args, env)
	case "validate":
		return tableValidate(t, args, env)
	case "isValid":
		return tableIsValid(t, args, env)
	case "errors":
		return tableErrors(t, args, env)
	case "validRows":
		return tableValidRows(t, args, env)
	case "invalidRows":
		return tableInvalidRows(t, args, env)
	// Array-like methods
	case "map":
		return tableMap(t, args, env)
	case "find":
		return tableFind(t, args, env)
	case "any":
		return tableAny(t, args, env)
	case "all":
		return tableAll(t, args, env)
	// Data manipulation methods
	case "unique":
		return tableUnique(t, args, env)
	case "renameCol":
		return tableRenameCol(t, args, env)
	case "dropCol":
		return tableDropCol(t, args, env)
	case "groupBy":
		return tableGroupBy(t, args, env)
	default:
		return unknownMethodError(method, "Table", []string{
			"where", "orderBy", "select", "limit", "offset", "count", "sum", "avg", "min", "max",
			"toHTML", "toCSV", "toMarkdown", "toBox", "toJSON", "toArray", "copy",
			"appendRow", "insertRowAt", "appendCol", "insertColAfter", "insertColBefore",
			"rowCount", "columnCount", "column",
			"as", "validate", "isValid", "errors", "validRows", "invalidRows",
			"map", "find", "any", "all", "unique", "renameCol", "dropCol", "groupBy",
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
	case "row":
		return tableRow(t)
	case "length":
		return &Integer{Value: int64(len(t.Rows))}
	case "schema":
		if t.Schema != nil {
			return t.Schema
		}
		return NULL
	default:
		return newUndefinedError("UNDEF-0004", map[string]any{"Property": property, "Type": "Table"})
	}
}

// tableRow returns the first row of the table as a dictionary, or NULL if the table is empty
func tableRow(t *Table) Object {
	if len(t.Rows) == 0 {
		return NULL
	}
	return t.Rows[0]
}

// ============================================================================
// Phase 3: Table Validation Methods for Typed Tables
// ============================================================================

// tableAs implements table.as(Schema) → typed Table
// Implements SPEC-TBL-005
func tableAs(t *Table, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("as", len(args), 1)
	}
	schema, ok := args[0].(*DSLSchema)
	if !ok {
		return newTypeError("TYPE-0001", "as", "a schema", args[0].Type())
	}
	return BindSchemaToTable(schema, t, env)
}

// tableValidate implements table.validate() → Table
// Validates all rows and returns a new table with validation state.
// Implements SPEC-TBL-MTD-001
func tableValidate(t *Table, args []Object, env *Environment) Object {
	if len(args) != 0 {
		return newArityError("validate", len(args), 0)
	}
	if t.Schema == nil {
		return &Error{
			Message: "validate() requires a typed table (use Schema([...]) or table.as(Schema))",
			Class:   ClassType,
		}
	}

	// Create new table with validated rows
	newRows := make([]*Dictionary, 0, len(t.Rows))
	for _, row := range t.Rows {
		// Create a record from the row
		// CreateRecord and ValidateRecord return *Record directly
		rec := CreateRecord(t.Schema, row, env)
		validatedRec := ValidateRecord(rec, env)

		// Convert back to dictionary, storing validation state
		rowDict := validatedRec.ToDictionaryWithErrors()
		rowDict.Env = env
		newRows = append(newRows, rowDict)
	}

	return &Table{
		Rows:    newRows,
		Columns: t.Columns,
		Schema:  t.Schema,
	}
}

// tableIsValid implements table.isValid() → Boolean
// Returns true if ALL rows are valid.
// Implements SPEC-TBL-MTD-001
func tableIsValid(t *Table, args []Object, env *Environment) Object {
	if len(args) != 0 {
		return newArityError("isValid", len(args), 0)
	}
	if t.Schema == nil {
		return FALSE
	}

	// Check each row
	for _, row := range t.Rows {
		// Check if row has stored errors
		if errorsExpr, hasErrors := row.Pairs["__errors__"]; hasErrors {
			errorsObj := Eval(errorsExpr, row.Env)
			if errDict, ok := errorsObj.(*Dictionary); ok && len(errDict.Pairs) > 0 {
				return FALSE
			}
		} else {
			// If no stored errors, validate the row
			rec := CreateRecord(t.Schema, row, env)
			validatedRec := ValidateRecord(rec, env)
			if len(validatedRec.Errors) > 0 {
				return FALSE
			}
		}
	}

	return TRUE
}

// tableErrors implements table.errors() → [{row, field, code, message}]
// Returns all validation errors with row indices.
// Implements SPEC-TBL-ERR-001, SPEC-TBL-ERR-002
func tableErrors(t *Table, args []Object, env *Environment) Object {
	if len(args) != 0 {
		return newArityError("errors", len(args), 0)
	}
	if t.Schema == nil {
		return &Array{Elements: []Object{}}
	}

	var allErrors []Object
	for rowIdx, row := range t.Rows {
		// Get errors for this row
		var rowErrors map[string]*RecordError

		if errorsExpr, hasErrors := row.Pairs["__errors__"]; hasErrors {
			// Use stored errors
			errorsObj := Eval(errorsExpr, row.Env)
			if errDict, ok := errorsObj.(*Dictionary); ok {
				rowErrors = dictToRecordErrors(errDict, row.Env)
			}
		} else {
			// Validate the row to get errors
			rec := CreateRecord(t.Schema, row, env)
			validatedRec := ValidateRecord(rec, env)
			rowErrors = validatedRec.Errors
		}

		// Add each error with row index
		for field, err := range rowErrors {
			errorEntry := &Dictionary{
				Pairs: map[string]ast.Expression{
					"row":     objectToExpression(&Integer{Value: int64(rowIdx)}),
					"field":   objectToExpression(&String{Value: field}),
					"code":    objectToExpression(&String{Value: err.Code}),
					"message": objectToExpression(&String{Value: err.Message}),
				},
				KeyOrder: []string{"row", "field", "code", "message"},
				Env:      env,
			}
			allErrors = append(allErrors, errorEntry)
		}
	}

	return &Array{Elements: allErrors}
}

// tableValidRows implements table.validRows() → Table
// Returns a new table with only valid rows.
// Implements SPEC-TBL-MTD-001
func tableValidRows(t *Table, args []Object, env *Environment) Object {
	if len(args) != 0 {
		return newArityError("validRows", len(args), 0)
	}
	if t.Schema == nil {
		return t // No schema means no validation, return as-is
	}

	var validRows []*Dictionary
	for _, row := range t.Rows {
		isValid := true

		if errorsExpr, hasErrors := row.Pairs["__errors__"]; hasErrors {
			errorsObj := Eval(errorsExpr, row.Env)
			if errDict, ok := errorsObj.(*Dictionary); ok && len(errDict.Pairs) > 0 {
				isValid = false
			}
		} else {
			// Validate the row
			rec := CreateRecord(t.Schema, row, env)
			validatedRec := ValidateRecord(rec, env)
			if len(validatedRec.Errors) > 0 {
				isValid = false
			}
		}

		if isValid {
			validRows = append(validRows, row)
		}
	}

	return &Table{
		Rows:    validRows,
		Columns: t.Columns,
		Schema:  t.Schema,
	}
}

// tableInvalidRows implements table.invalidRows() → Table
// Returns a new table with only invalid rows.
// Implements SPEC-TBL-MTD-001
func tableInvalidRows(t *Table, args []Object, env *Environment) Object {
	if len(args) != 0 {
		return newArityError("invalidRows", len(args), 0)
	}
	if t.Schema == nil {
		return &Table{Rows: []*Dictionary{}, Columns: t.Columns, Schema: nil}
	}

	var invalidRows []*Dictionary
	for _, row := range t.Rows {
		isInvalid := false

		if errorsExpr, hasErrors := row.Pairs["__errors__"]; hasErrors {
			errorsObj := Eval(errorsExpr, row.Env)
			if errDict, ok := errorsObj.(*Dictionary); ok && len(errDict.Pairs) > 0 {
				isInvalid = true
			}
		} else {
			// Validate the row
			rec := CreateRecord(t.Schema, row, env)
			validatedRec := ValidateRecord(rec, env)
			if len(validatedRec.Errors) > 0 {
				isInvalid = true
			}
		}

		if isInvalid {
			invalidRows = append(invalidRows, row)
		}
	}

	return &Table{
		Rows:    invalidRows,
		Columns: t.Columns,
		Schema:  t.Schema,
	}
}

// dictToRecordErrors converts a dictionary of errors to RecordError map
func dictToRecordErrors(errDict *Dictionary, env *Environment) map[string]*RecordError {
	errors := make(map[string]*RecordError)
	for field, errExpr := range errDict.Pairs {
		errObj := Eval(errExpr, env)
		if errEntry, ok := errObj.(*Dictionary); ok {
			code := ""
			message := ""
			if codeExpr, hasCode := errEntry.Pairs["code"]; hasCode {
				if codeObj := Eval(codeExpr, errEntry.Env); codeObj != nil {
					if codeStr, ok := codeObj.(*String); ok {
						code = codeStr.Value
					}
				}
			}
			if msgExpr, hasMsg := errEntry.Pairs["message"]; hasMsg {
				if msgObj := Eval(msgExpr, errEntry.Env); msgObj != nil {
					if msgStr, ok := msgObj.(*String); ok {
						message = msgStr.Value
					}
				}
			}
			errors[field] = &RecordError{Code: code, Message: message}
		}
	}
	return errors
}
