package evaluator

import (
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
)

// TableBinding represents a schema-bound table helper that provides CRUD methods backed by a DB connection.
type TableBinding struct {
	DB               *DBConnection
	Schema           *Dictionary // Old-style schema (Dictionary)
	DSLSchema        *DSLSchema  // New-style schema (@schema)
	TableName        string
	SoftDeleteColumn string // Column name for soft deletes, empty if disabled
}

func (tb *TableBinding) Type() ObjectType { return TABLE_BINDING_OBJ }
func (tb *TableBinding) Inspect() string {
	if tb.SoftDeleteColumn != "" {
		return fmt.Sprintf("TableBinding(%s, soft_delete: %s)", tb.TableName, tb.SoftDeleteColumn)
	}
	return fmt.Sprintf("TableBinding(%s)", tb.TableName)
}

// HasDSLSchema returns true if this binding uses a DSL schema
func (tb *TableBinding) HasDSLSchema() bool {
	return tb.DSLSchema != nil
}

// GetSchemaName returns the name of the bound schema
func (tb *TableBinding) GetSchemaName() string {
	if tb.DSLSchema != nil {
		return tb.DSLSchema.Name
	}
	if tb.Schema != nil {
		if nameExpr, ok := tb.Schema.Pairs["__name"]; ok {
			if nameStr, ok := nameExpr.(*ast.StringLiteral); ok {
				return nameStr.Value
			}
		}
	}
	return ""
}

// QueryOptions holds parsed query options for orderBy, select, limit/offset
type QueryOptions struct {
	OrderBy []OrderSpec // [{Column: "name", Dir: "ASC"}, ...]
	Select  []string    // ["id", "name"] or nil for *
	Limit   *int64      // nil = use default/no limit
	Offset  *int64      // nil = 0
	NoLimit bool        // explicit limit <= 0 means no limit
}

// OrderSpec specifies a column and direction for ORDER BY
type OrderSpec struct {
	Column string
	Dir    string // "ASC" or "DESC"
}

// evalTableBindingMethod dispatches method calls on TableBinding instances.
func evalTableBindingMethod(tb *TableBinding, method string, args []Object, env *Environment) Object {
	switch method {
	case "all":
		return tb.executeAll(args, env)
	case "find":
		return tb.executeFind(args, env)
	case "where":
		return tb.executeWhere(args, env)
	case "insert":
		return tb.executeInsert(args, env)
	case "update":
		return tb.executeUpdate(args, env)
	case "save":
		return tb.executeSave(args, env)
	case "delete":
		return tb.executeDelete(args, env)
	case "count":
		return tb.executeCount(args, env)
	case "sum":
		return tb.executeAggregate("SUM", args, env)
	case "avg":
		return tb.executeAggregate("AVG", args, env)
	case "min":
		return tb.executeAggregate("MIN", args, env)
	case "max":
		return tb.executeAggregate("MAX", args, env)
	case "first":
		return tb.executeFirst(args, env)
	case "last":
		return tb.executeLast(args, env)
	case "exists":
		return tb.executeExists(args, env)
	case "findBy":
		return tb.executeFindBy(args, env)
	case "toSQL":
		return tb.executeToSQL(args, env)
	default:
		return unknownMethodError(method, "TableBinding", []string{
			"all", "find", "where", "insert", "update", "save", "delete",
			"count", "sum", "avg", "min", "max",
			"first", "last", "exists", "findBy", "toSQL",
		})
	}
}

var identifierRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func (tb *TableBinding) ensureSQLite() *Error {
	if tb.DB == nil {
		return newDatabaseStateError("DB-0001")
	}
	if tb.DB.Driver != "sqlite" {
		return newDatabaseErrorWithDriver("DB-0001", tb.DB.Driver, fmt.Errorf("table binding only supports sqlite"))
	}
	return nil
}

// parseQueryOptions extracts QueryOptions from a dictionary argument.
func parseQueryOptions(dict *Dictionary) (*QueryOptions, *Error) {
	opts := &QueryOptions{}

	// Parse orderBy
	if orderByExpr, ok := dict.Pairs["orderBy"]; ok {
		orderByVal := Eval(orderByExpr, dict.Env)
		if isError(orderByVal) {
			return nil, orderByVal.(*Error)
		}

		switch v := orderByVal.(type) {
		case *String:
			// Simple string: {orderBy: "name"}
			if !identifierRegex.MatchString(v.Value) {
				return nil, newValidationError("VAL-0003", map[string]any{"Pattern": "identifier", "GoError": fmt.Sprintf("invalid column name in orderBy: %s", v.Value)})
			}
			dir := "ASC"
			if orderExpr, ok := dict.Pairs["order"]; ok {
				orderVal := Eval(orderExpr, dict.Env)
				if str, ok := orderVal.(*String); ok {
					upper := strings.ToUpper(str.Value)
					if upper != "ASC" && upper != "DESC" {
						return nil, newValidationError("VAL-0003", map[string]any{"Pattern": "order direction", "GoError": fmt.Sprintf("order must be 'asc' or 'desc', got: %s", str.Value)})
					}
					dir = upper
				}
			}
			opts.OrderBy = []OrderSpec{{Column: v.Value, Dir: dir}}

		case *Array:
			// Array of [col, dir] pairs: {orderBy: [["age", "desc"], ["name", "asc"]]}
			for _, elem := range v.Elements {
				pair, ok := elem.(*Array)
				if !ok || len(pair.Elements) != 2 {
					return nil, newValidationError("VAL-0003", map[string]any{"Pattern": "orderBy array", "GoError": "orderBy array elements must be [column, direction] pairs"})
				}
				colObj, ok := pair.Elements[0].(*String)
				if !ok {
					return nil, newValidationError("VAL-0003", map[string]any{"Pattern": "orderBy column", "GoError": "orderBy column must be a string"})
				}
				if !identifierRegex.MatchString(colObj.Value) {
					return nil, newValidationError("VAL-0003", map[string]any{"Pattern": "identifier", "GoError": fmt.Sprintf("invalid column name in orderBy: %s", colObj.Value)})
				}
				dirObj, ok := pair.Elements[1].(*String)
				if !ok {
					return nil, newValidationError("VAL-0003", map[string]any{"Pattern": "orderBy direction", "GoError": "orderBy direction must be a string"})
				}
				dir := strings.ToUpper(dirObj.Value)
				if dir != "ASC" && dir != "DESC" {
					return nil, newValidationError("VAL-0003", map[string]any{"Pattern": "order direction", "GoError": fmt.Sprintf("order must be 'asc' or 'desc', got: %s", dirObj.Value)})
				}
				opts.OrderBy = append(opts.OrderBy, OrderSpec{Column: colObj.Value, Dir: dir})
			}
		}
	}

	// Parse select
	if selectExpr, ok := dict.Pairs["select"]; ok {
		selectVal := Eval(selectExpr, dict.Env)
		if isError(selectVal) {
			return nil, selectVal.(*Error)
		}
		arr, ok := selectVal.(*Array)
		if !ok {
			return nil, newValidationError("VAL-0003", map[string]any{"Pattern": "select", "GoError": "select must be an array of column names"})
		}
		for _, elem := range arr.Elements {
			str, ok := elem.(*String)
			if !ok {
				return nil, newValidationError("VAL-0003", map[string]any{"Pattern": "select column", "GoError": "select columns must be strings"})
			}
			if !identifierRegex.MatchString(str.Value) {
				return nil, newValidationError("VAL-0003", map[string]any{"Pattern": "identifier", "GoError": fmt.Sprintf("invalid column name in select: %s", str.Value)})
			}
			opts.Select = append(opts.Select, str.Value)
		}
	}

	// Parse limit
	if limitExpr, ok := dict.Pairs["limit"]; ok {
		limitVal := Eval(limitExpr, dict.Env)
		if isError(limitVal) {
			return nil, limitVal.(*Error)
		}
		if intVal, ok := limitVal.(*Integer); ok {
			if intVal.Value <= 0 {
				opts.NoLimit = true
			} else {
				opts.Limit = &intVal.Value
			}
		}
	}

	// Parse offset
	if offsetExpr, ok := dict.Pairs["offset"]; ok {
		offsetVal := Eval(offsetExpr, dict.Env)
		if isError(offsetVal) {
			return nil, offsetVal.(*Error)
		}
		if intVal, ok := offsetVal.(*Integer); ok && intVal.Value >= 0 {
			opts.Offset = &intVal.Value
		}
	}

	return opts, nil
}

// buildOrderByClause generates the ORDER BY clause from QueryOptions.
func buildOrderByClause(opts *QueryOptions) string {
	if opts == nil || len(opts.OrderBy) == 0 {
		return ""
	}
	parts := make([]string, len(opts.OrderBy))
	for i, spec := range opts.OrderBy {
		parts[i] = fmt.Sprintf("%s %s", spec.Column, spec.Dir)
	}
	return " ORDER BY " + strings.Join(parts, ", ")
}

// buildSelectClause generates the SELECT columns from QueryOptions.
func buildSelectClause(opts *QueryOptions) string {
	if opts == nil || len(opts.Select) == 0 {
		return "*"
	}
	return strings.Join(opts.Select, ", ")
}

// executeAll selects all rows with optional pagination defaults.
func (tb *TableBinding) executeAll(args []Object, env *Environment) Object {
	if err := tb.ensureSQLite(); err != nil {
		return err
	}

	var opts *QueryOptions
	if len(args) > 0 {
		if dict, ok := args[0].(*Dictionary); ok {
			var parseErr *Error
			opts, parseErr = parseQueryOptions(dict)
			if parseErr != nil {
				return parseErr
			}
		}
	}

	// Build SELECT clause
	selectCols := buildSelectClause(opts)
	query := fmt.Sprintf("SELECT %s FROM %s", selectCols, tb.TableName)

	// Add ORDER BY if specified
	query += buildOrderByClause(opts)

	// Handle pagination
	var params []Object
	if opts != nil && opts.NoLimit {
		// Explicit no-limit requested
	} else if opts != nil && opts.Limit != nil {
		// Use explicit limit/offset
		offset := int64(0)
		if opts.Offset != nil {
			offset = *opts.Offset
		}
		query += " LIMIT ? OFFSET ?"
		params = append(params, &Integer{Value: *opts.Limit}, &Integer{Value: offset})
	} else {
		// Use auto-pagination from request
		limit, offset, useLimit := getPagination(env)
		if useLimit {
			query += " LIMIT ? OFFSET ?"
			params = append(params, &Integer{Value: limit}, &Integer{Value: offset})
		}
	}

	return tb.queryRows(query, params, env)
}

// executeFind selects a single row by id.
// Returns a Record if schema is bound, Dictionary otherwise.
// Implements SPEC-DB-001
func (tb *TableBinding) executeFind(args []Object, env *Environment) Object {
	if err := tb.ensureSQLite(); err != nil {
		return err
	}
	if len(args) != 1 {
		return newArityError("find", len(args), 1)
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE id = ? LIMIT 1", tb.TableName)
	return tb.querySingleRow(query, []Object{args[0]}, env)
}

// executeWhere selects rows matching equality conditions from a dictionary.
// Unlike all(), where() does not apply automatic pagination by default.
// Accepts optional second argument for options (orderBy, select, limit, offset).
func (tb *TableBinding) executeWhere(args []Object, env *Environment) Object {
	if err := tb.ensureSQLite(); err != nil {
		return err
	}
	if len(args) < 1 || len(args) > 2 {
		return newArityError("where", len(args), 1)
	}

	condDict, ok := args[0].(*Dictionary)
	if !ok {
		return newTypeError("TYPE-0005", "where", "a dictionary", args[0].Type())
	}

	// Parse options if provided
	var opts *QueryOptions
	if len(args) == 2 {
		if optsDict, ok := args[1].(*Dictionary); ok {
			var parseErr *Error
			opts, parseErr = parseQueryOptions(optsDict)
			if parseErr != nil {
				return parseErr
			}
		}
	}

	conditions, params, errObj := tb.buildWhereClause(condDict)
	if errObj != nil {
		return errObj
	}

	// Build SELECT clause
	selectCols := buildSelectClause(opts)
	query := fmt.Sprintf("SELECT %s FROM %s", selectCols, tb.TableName)
	if conditions != "" {
		query += " WHERE " + conditions
	}

	// Add ORDER BY if specified
	query += buildOrderByClause(opts)

	// Handle limit/offset if specified in options
	if opts != nil && opts.Limit != nil {
		offset := int64(0)
		if opts.Offset != nil {
			offset = *opts.Offset
		}
		query += " LIMIT ? OFFSET ?"
		params = append(params, &Integer{Value: *opts.Limit}, &Integer{Value: offset})
	}

	return tb.queryRows(query, params, env)
}

// executeInsert validates and inserts rows. Accepts Dictionary, Record, or Table.
func (tb *TableBinding) executeInsert(args []Object, env *Environment) Object {
	if err := tb.ensureSQLite(); err != nil {
		return err
	}
	if len(args) != 1 {
		return newArityError("insert", len(args), 1)
	}

	// Dispatch based on argument type
	switch arg := args[0].(type) {
	case *Record:
		return tb.executeInsertRecord(arg, env)
	case *Table:
		return tb.executeInsertTable(arg, env)
	case *Dictionary:
		return tb.executeInsertDictionary(arg, env)
	default:
		return newTypeError("TYPE-0005", "insert", "a dictionary, record, or table", args[0].Type())
	}
}

// executeInsertRecord inserts a single Record.
func (tb *TableBinding) executeInsertRecord(record *Record, env *Environment) Object {
	// Schema matching: if record has a schema, it must match
	if record.Schema != nil && tb.DSLSchema != nil {
		if record.Schema.Name != tb.DSLSchema.Name {
			return newValidationError("VAL-0022", map[string]any{
				"Expected": tb.DSLSchema.Name,
				"Got":      record.Schema.Name,
			})
		}
	}

	// Convert Record to Dictionary and delegate
	data := record.ToDictionary()
	return tb.executeInsertDictionary(data, env)
}

// executeInsertTable inserts multiple rows from a Table.
func (tb *TableBinding) executeInsertTable(table *Table, env *Environment) Object {
	// Schema matching: if table has a schema, it must match
	if table.Schema != nil && tb.DSLSchema != nil {
		if table.Schema.Name != tb.DSLSchema.Name {
			return newValidationError("VAL-0022", map[string]any{
				"Expected": tb.DSLSchema.Name,
				"Got":      table.Schema.Name,
			})
		}
	}

	// Insert each row
	insertedCount := int64(0)
	for _, row := range table.Rows {
		result := tb.executeInsertDictionary(row, env)
		if isError(result) {
			return result
		}
		// Result is either the inserted row or a validation error dict
		if dict, ok := result.(*Dictionary); ok {
			if validExpr, hasValid := dict.Pairs["valid"]; hasValid {
				validObj := Eval(validExpr, dict.Env)
				if validObj.Type() == BOOLEAN_OBJ && !validObj.(*Boolean).Value {
					// Validation failed - return the error
					return dict
				}
			}
		}
		insertedCount++
	}

	return &Dictionary{
		Pairs: map[string]ast.Expression{
			"inserted": objectToExpression(&Integer{Value: insertedCount}),
		},
		Env: env,
	}
}

// executeInsertDictionary validates and inserts a single dictionary row.
func (tb *TableBinding) executeInsertDictionary(data *Dictionary, env *Environment) Object {
	// Add id if missing or null
	needsID := true
	if idExpr, exists := data.Pairs["id"]; exists {
		idVal := Eval(idExpr, data.Env)
		if idVal != nil && idVal != NULL {
			needsID = false
		}
	}
	if needsID {
		if idObj := tb.generateID(env); idObj != nil {
			data.SetKey("id", objectToExpression(idObj))
		}
	}

	// Validate based on schema type
	if tb.DSLSchema != nil {
		// For DSL schemas, use Record-based validation
		record := CreateRecord(tb.DSLSchema, data, env)
		validated := ValidateRecord(record, env)
		if len(validated.Errors) > 0 {
			// Return validation result as dictionary for backward compatibility
			errorsDict := &Dictionary{Pairs: make(map[string]ast.Expression)}
			for field, err := range validated.Errors {
				errorsDict.Pairs[field] = objectToExpression(&String{Value: err.Message})
			}
			return &Dictionary{
				Pairs: map[string]ast.Expression{
					"valid":  objectToExpression(FALSE),
					"errors": objectToExpression(errorsDict),
				},
			}
		}
	} else if tb.Schema != nil {
		// For old-style dictionary schemas, use schemaValidate
		validation := schemaValidate(tb.Schema, data)
		if dict, ok := validation.(*Dictionary); ok {
			if validExpr, hasValid := dict.Pairs["valid"]; hasValid {
				if validObj := Eval(validExpr, dict.Env); validObj.Type() == BOOLEAN_OBJ && !validObj.(*Boolean).Value {
					return dict
				}
			}
		}
	}

	columns, params, errObj := tb.extractColumnsAndParams(data)
	if errObj != nil {
		return errObj
	}

	placeholders := make([]string, len(columns))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	// Build INSERT query
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", tb.TableName, strings.Join(columns, ", "), strings.Join(placeholders, ", "))

	// Version check implemented, RETURNING ready but not yet enabled
	//
	// SQLite 3.35.0+ supports RETURNING clause which would be more efficient:
	//   INSERT INTO users (...) VALUES (...) RETURNING *
	// This avoids the separate SELECT query below.
	//
	// Current implementation uses INSERT + SELECT fallback for reliability:
	// - Works with all SQLite versions (3.0.0+)
	// - Simpler implementation (reuses existing queryRows logic)
	// - Performance difference negligible for single inserts
	//
	// To enable RETURNING in future (requires additional work):
	//   1. Add queryRow method that returns single dictionary
	//   2. Handle RETURNING * column scanning (reuse queryRows internals)
	//   3. Add integration tests for SQLite 3.35.0+
	//   4. Fall back to SELECT for older versions
	//
	// Version detection already exists: sqliteSupportsReturning(tb.DB.SQLiteVersion)
	// See: evaluator.go lines 4714-4730

	// Execute INSERT
	if execErr := tb.executeMutation(query, params); execErr != nil {
		return execErr
	}

	// Fetch and return inserted row (works for all SQLite versions)
	if idExpr, ok := data.Pairs["id"]; ok {
		idVal := Eval(idExpr, data.Env)
		return tb.executeFind([]Object{idVal}, env)
	}
	return NULL
}

// executeUpdate validates and updates rows. Accepts (id, Dictionary) or Record/Table.
func (tb *TableBinding) executeUpdate(args []Object, env *Environment) Object {
	if err := tb.ensureSQLite(); err != nil {
		return err
	}

	// Handle single-arg form: update(record) or update(table)
	if len(args) == 1 {
		switch arg := args[0].(type) {
		case *Record:
			return tb.executeUpdateRecord(arg, env)
		case *Table:
			return tb.executeUpdateTable(arg, env)
		default:
			return newArityError("update", len(args), 2)
		}
	}

	// Handle two-arg form: update(id, data)
	if len(args) != 2 {
		return newArityError("update", len(args), 2)
	}

	idObj := args[0]
	data, ok := args[1].(*Dictionary)
	if !ok {
		return newTypeError("TYPE-0006", "update", "a dictionary (data)", args[1].Type())
	}

	return tb.executeUpdateByID(idObj, data, env)
}

// executeUpdateRecord updates a single Record using its primary key.
func (tb *TableBinding) executeUpdateRecord(record *Record, env *Environment) Object {
	// Schema matching
	if record.Schema != nil && tb.DSLSchema != nil {
		if record.Schema.Name != tb.DSLSchema.Name {
			return newValidationError("VAL-0022", map[string]any{
				"Expected": tb.DSLSchema.Name,
				"Got":      record.Schema.Name,
			})
		}
	}

	// Get primary key field name
	pkField := "id"
	if tb.DSLSchema != nil {
		if pk := tb.DSLSchema.PrimaryKey(); pk != "" {
			pkField = pk
		}
	}

	// Get primary key value from record
	pkValue := record.Get(pkField, env)
	if pkValue == nil || pkValue == NULL {
		return newDatabaseStateError("DB-0016") // Cannot update: record has no primary key value
	}

	// Convert to dictionary and remove primary key (don't update it)
	data := record.ToDictionary()
	delete(data.Pairs, pkField)

	return tb.executeUpdateByID(pkValue, data, env)
}

// executeUpdateTable updates multiple rows from a Table using primary keys.
func (tb *TableBinding) executeUpdateTable(table *Table, env *Environment) Object {
	// Schema matching
	if table.Schema != nil && tb.DSLSchema != nil {
		if table.Schema.Name != tb.DSLSchema.Name {
			return newValidationError("VAL-0022", map[string]any{
				"Expected": tb.DSLSchema.Name,
				"Got":      table.Schema.Name,
			})
		}
	}

	// Get primary key field name
	pkField := "id"
	if tb.DSLSchema != nil {
		if pk := tb.DSLSchema.PrimaryKey(); pk != "" {
			pkField = pk
		}
	}

	updatedCount := int64(0)
	for _, row := range table.Rows {
		// Get primary key value
		pkExpr, hasPK := row.Pairs[pkField]
		if !hasPK {
			return newDatabaseStateError("DB-0016")
		}
		pkValue := Eval(pkExpr, row.Env)
		if pkValue == nil || pkValue == NULL {
			return newDatabaseStateError("DB-0016")
		}

		// Create a copy without the primary key
		dataCopy := &Dictionary{
			Pairs: make(map[string]ast.Expression, len(row.Pairs)-1),
			Env:   row.Env,
		}
		for k, v := range row.Pairs {
			if k != pkField {
				dataCopy.Pairs[k] = v
			}
		}

		result := tb.executeUpdateByID(pkValue, dataCopy, env)
		if isError(result) {
			return result
		}
		// Check for validation errors
		if dict, ok := result.(*Dictionary); ok {
			if validExpr, hasValid := dict.Pairs["valid"]; hasValid {
				validObj := Eval(validExpr, dict.Env)
				if validObj.Type() == BOOLEAN_OBJ && !validObj.(*Boolean).Value {
					return dict
				}
			}
		}
		updatedCount++
	}

	return &Dictionary{
		Pairs: map[string]ast.Expression{
			"updated": objectToExpression(&Integer{Value: updatedCount}),
		},
		Env: env,
	}
}

// executeUpdateByID validates and updates a row by id (original implementation).
// Note: For partial updates, we skip "required" validation since only some fields
// are being updated. Type validation is still performed if a legacy Schema is present.
func (tb *TableBinding) executeUpdateByID(idObj Object, data *Dictionary, env *Environment) Object {
	// Disallow id changes for safety
	if _, hasID := data.Pairs["id"]; hasID {
		return newValidationError("VAL-0004", map[string]any{"Method": "update", "Got": "id"})
	}

	// For DSLSchema, skip validation since it would require all fields.
	// For partial updates, we only validate type correctness if legacy Schema exists.
	if tb.Schema != nil {
		validation := schemaValidate(tb.Schema, data)
		if dict, ok := validation.(*Dictionary); ok {
			if validExpr, hasValid := dict.Pairs["valid"]; hasValid {
				if validObj := Eval(validExpr, dict.Env); validObj.Type() == BOOLEAN_OBJ && !validObj.(*Boolean).Value {
					return dict
				}
			}
		}
	}

	columns, params, errObj := tb.extractColumnsAndParams(data)
	if errObj != nil {
		return errObj
	}
	if len(columns) == 0 {
		return newValidationError("VAL-0011", map[string]any{"Function": "update"})
	}

	setClauses := make([]string, len(columns))
	for i, col := range columns {
		setClauses[i] = fmt.Sprintf("%s = ?", col)
	}
	params = append(params, idObj)

	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", tb.TableName, strings.Join(setClauses, ", "))
	if execErr := tb.executeMutation(query, params); execErr != nil {
		return execErr
	}

	return tb.executeFind([]Object{idObj}, env)
}

// executeDelete deletes rows. Accepts id, Record, or Table.
func (tb *TableBinding) executeDelete(args []Object, env *Environment) Object {
	if err := tb.ensureSQLite(); err != nil {
		return err
	}
	if len(args) != 1 {
		return newArityError("delete", len(args), 1)
	}

	// Dispatch based on argument type
	switch arg := args[0].(type) {
	case *Record:
		return tb.executeDeleteRecord(arg, env)
	case *Table:
		return tb.executeDeleteTable(arg, env)
	default:
		// Assume it's an id value
		return tb.executeDeleteByID(arg, env)
	}
}

// executeDeleteRecord deletes a single Record using its primary key.
func (tb *TableBinding) executeDeleteRecord(record *Record, env *Environment) Object {
	// Schema matching
	if record.Schema != nil && tb.DSLSchema != nil {
		if record.Schema.Name != tb.DSLSchema.Name {
			return newValidationError("VAL-0022", map[string]any{
				"Expected": tb.DSLSchema.Name,
				"Got":      record.Schema.Name,
			})
		}
	}

	// Get primary key field name
	pkField := "id"
	if tb.DSLSchema != nil {
		if pk := tb.DSLSchema.PrimaryKey(); pk != "" {
			pkField = pk
		}
	}

	// Get primary key value
	pkValue := record.Get(pkField, env)
	if pkValue == nil || pkValue == NULL {
		return newDatabaseStateError("DB-0017") // Cannot delete: record has no primary key value
	}

	return tb.executeDeleteByID(pkValue, env)
}

// executeDeleteTable deletes multiple rows from a Table using primary keys.
func (tb *TableBinding) executeDeleteTable(table *Table, env *Environment) Object {
	// Schema matching
	if table.Schema != nil && tb.DSLSchema != nil {
		if table.Schema.Name != tb.DSLSchema.Name {
			return newValidationError("VAL-0022", map[string]any{
				"Expected": tb.DSLSchema.Name,
				"Got":      table.Schema.Name,
			})
		}
	}

	// Get primary key field name
	pkField := "id"
	if tb.DSLSchema != nil {
		if pk := tb.DSLSchema.PrimaryKey(); pk != "" {
			pkField = pk
		}
	}

	totalAffected := int64(0)
	for _, row := range table.Rows {
		// Get primary key value
		pkExpr, hasPK := row.Pairs[pkField]
		if !hasPK {
			return newDatabaseStateError("DB-0017")
		}
		pkValue := Eval(pkExpr, row.Env)
		if pkValue == nil || pkValue == NULL {
			return newDatabaseStateError("DB-0017")
		}

		result := tb.executeDeleteByID(pkValue, env)
		if isError(result) {
			return result
		}
		// Extract affected count
		if dict, ok := result.(*Dictionary); ok {
			if affExpr, hasAff := dict.Pairs["affected"]; hasAff {
				affObj := Eval(affExpr, dict.Env)
				if affInt, ok := affObj.(*Integer); ok {
					totalAffected += affInt.Value
				}
			}
		}
	}

	return &Dictionary{
		Pairs: map[string]ast.Expression{
			"affected": objectToExpression(&Integer{Value: totalAffected}),
		},
		Env: env,
	}
}

// executeDeleteByID deletes a row by id and returns affected count.
func (tb *TableBinding) executeDeleteByID(idObj Object, env *Environment) Object {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", tb.TableName)
	result, err := tb.exec(query, []Object{idObj})
	if err != nil {
		return err
	}

	affected, _ := result.RowsAffected()
	pairs := map[string]ast.Expression{
		"affected": objectToExpression(&Integer{Value: affected}),
	}
	return &Dictionary{Pairs: pairs, Env: env}
}

// executeSave performs an upsert (INSERT ... ON CONFLICT DO UPDATE).
// Accepts Record, Table, or Dictionary.
func (tb *TableBinding) executeSave(args []Object, env *Environment) Object {
	if err := tb.ensureSQLite(); err != nil {
		return err
	}
	if len(args) != 1 {
		return newArityError("save", len(args), 1)
	}

	// Dispatch based on argument type
	switch arg := args[0].(type) {
	case *Record:
		return tb.executeSaveRecord(arg, env)
	case *Table:
		return tb.executeSaveTable(arg, env)
	case *Dictionary:
		return tb.executeSaveDictionary(arg, env)
	default:
		return newTypeError("TYPE-0005", "save", "a dictionary, record, or table", args[0].Type())
	}
}

// executeSaveRecord upserts a single Record.
func (tb *TableBinding) executeSaveRecord(record *Record, env *Environment) Object {
	// Schema matching
	if record.Schema != nil && tb.DSLSchema != nil {
		if record.Schema.Name != tb.DSLSchema.Name {
			return newValidationError("VAL-0022", map[string]any{
				"Expected": tb.DSLSchema.Name,
				"Got":      record.Schema.Name,
			})
		}
	}

	// Convert Record to Dictionary and delegate
	data := record.ToDictionary()
	return tb.executeSaveDictionary(data, env)
}

// executeSaveTable upserts multiple rows from a Table.
func (tb *TableBinding) executeSaveTable(table *Table, env *Environment) Object {
	// Schema matching
	if table.Schema != nil && tb.DSLSchema != nil {
		if table.Schema.Name != tb.DSLSchema.Name {
			return newValidationError("VAL-0022", map[string]any{
				"Expected": tb.DSLSchema.Name,
				"Got":      table.Schema.Name,
			})
		}
	}

	insertedCount := int64(0)

	for _, row := range table.Rows {
		result := tb.executeSaveDictionary(row, env)
		if isError(result) {
			return result
		}
		// Check for validation errors or count successful saves
		// Get the underlying data map (handles both Dictionary and Record)
		var dataMap map[string]ast.Expression
		var dataEnv *Environment
		switch r := result.(type) {
		case *Dictionary:
			dataMap = r.Pairs
			dataEnv = r.Env
		case *Record:
			dataMap = r.Data
			dataEnv = r.Env
		}

		if dataMap != nil {
			if validExpr, hasValid := dataMap["valid"]; hasValid {
				validObj := Eval(validExpr, dataEnv)
				if validObj.Type() == BOOLEAN_OBJ && !validObj.(*Boolean).Value {
					// Return validation error as dictionary
					return &Dictionary{Pairs: dataMap, Env: dataEnv}
				}
			}
			// __saved__ indicates successful save (insert or update)
			if savedExpr, hasSaved := dataMap["__saved__"]; hasSaved {
				savedObj := Eval(savedExpr, dataEnv)
				if savedBool, ok := savedObj.(*Boolean); ok && savedBool.Value {
					// We count all saves; SQLite upsert doesn't distinguish insert vs update
					insertedCount++
				}
			}
		}
	}

	return &Dictionary{
		Pairs: map[string]ast.Expression{
			"saved": objectToExpression(&Integer{Value: insertedCount}),
			"total": objectToExpression(&Integer{Value: insertedCount}),
		},
		Env: env,
	}
}

// executeSaveDictionary upserts a single dictionary row.
func (tb *TableBinding) executeSaveDictionary(data *Dictionary, env *Environment) Object {
	// Get primary key field name
	pkField := "id"
	if tb.DSLSchema != nil {
		if pk := tb.DSLSchema.PrimaryKey(); pk != "" {
			pkField = pk
		}
	}

	// Add id if missing
	if _, exists := data.Pairs[pkField]; !exists {
		if idObj := tb.generateID(env); idObj != nil {
			data.SetKey(pkField, objectToExpression(idObj))
		}
	}

	// Validate based on schema type
	if tb.DSLSchema != nil {
		record := CreateRecord(tb.DSLSchema, data, env)
		validated := ValidateRecord(record, env)
		if len(validated.Errors) > 0 {
			errorsDict := &Dictionary{Pairs: make(map[string]ast.Expression)}
			for field, err := range validated.Errors {
				errorsDict.Pairs[field] = objectToExpression(&String{Value: err.Message})
			}
			return &Dictionary{
				Pairs: map[string]ast.Expression{
					"valid":  objectToExpression(FALSE),
					"errors": objectToExpression(errorsDict),
				},
			}
		}
	} else if tb.Schema != nil {
		validation := schemaValidate(tb.Schema, data)
		if dict, ok := validation.(*Dictionary); ok {
			if validExpr, hasValid := dict.Pairs["valid"]; hasValid {
				if validObj := Eval(validExpr, dict.Env); validObj.Type() == BOOLEAN_OBJ && !validObj.(*Boolean).Value {
					return dict
				}
			}
		}
	}

	columns, params, errObj := tb.extractColumnsAndParams(data)
	if errObj != nil {
		return errObj
	}

	placeholders := make([]string, len(columns))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	// Build UPDATE SET clause (exclude primary key)
	updateClauses := make([]string, 0, len(columns))
	for _, col := range columns {
		if col != pkField {
			updateClauses = append(updateClauses, fmt.Sprintf("%s = excluded.%s", col, col))
		}
	}

	// Build INSERT ... ON CONFLICT query
	// SQLite uses: INSERT INTO ... ON CONFLICT(pk) DO UPDATE SET ...
	var query string
	if len(updateClauses) > 0 {
		query = fmt.Sprintf(
			"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT(%s) DO UPDATE SET %s",
			tb.TableName,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "),
			pkField,
			strings.Join(updateClauses, ", "),
		)
	} else {
		// No non-PK columns to update - use INSERT OR REPLACE
		query = fmt.Sprintf(
			"INSERT OR REPLACE INTO %s (%s) VALUES (%s)",
			tb.TableName,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "),
		)
	}

	// Execute upsert
	result, err := tb.exec(query, params)
	if err != nil {
		return err
	}

	// Determine if it was insert or update
	// SQLite doesn't directly tell us, but we can check changes() and total_changes()
	// For simplicity, return the saved row
	if idExpr, ok := data.Pairs[pkField]; ok {
		idVal := Eval(idExpr, data.Env)
		row := tb.executeFind([]Object{idVal}, env)
		// Add metadata to indicate operation succeeded
		switch r := row.(type) {
		case *Dictionary:
			r.Pairs["__saved__"] = objectToExpression(TRUE)
		case *Record:
			r.Data["__saved__"] = objectToExpression(TRUE)
		}
		return row
	}

	// Check if rows were affected
	affected, _ := result.RowsAffected()
	return &Dictionary{
		Pairs: map[string]ast.Expression{
			"affected": objectToExpression(&Integer{Value: affected}),
		},
		Env: env,
	}
}

func (tb *TableBinding) extractColumnsAndParams(data *Dictionary) ([]string, []Object, *Error) {
	columns := make([]string, 0, len(data.Pairs))
	params := make([]Object, 0, len(data.Pairs))

	// Deterministic order
	keys := data.Keys()
	sort.Strings(keys)

	for _, key := range keys {
		if strings.HasPrefix(key, "__") {
			continue
		}
		if !identifierRegex.MatchString(key) {
			return nil, nil, newValidationError("VAL-0003", map[string]any{"Pattern": "identifier", "GoError": "invalid column name"})
		}

		expr := data.Pairs[key]
		val := Eval(expr, data.Env)
		if isError(val) {
			return nil, nil, val.(*Error)
		}
		columns = append(columns, key)
		params = append(params, val)
	}

	return columns, params, nil
}

func (tb *TableBinding) buildWhereClause(dict *Dictionary) (string, []Object, *Error) {
	keys := dict.Keys()
	if len(keys) == 0 {
		return "", nil, nil
	}

	sort.Strings(keys)
	clauses := make([]string, 0, len(keys))
	params := make([]Object, 0, len(keys))

	for _, key := range keys {
		if strings.HasPrefix(key, "__") {
			continue
		}
		if !identifierRegex.MatchString(key) {
			return "", nil, newValidationError("VAL-0003", map[string]any{"Pattern": "identifier", "GoError": "invalid column name"})
		}
		clauses = append(clauses, fmt.Sprintf("%s = ?", key))
		val := Eval(dict.Pairs[key], dict.Env)
		if isError(val) {
			return "", nil, val.(*Error)
		}
		params = append(params, val)
	}

	return strings.Join(clauses, " AND "), params, nil
}

func (tb *TableBinding) queryRows(query string, params []Object, env *Environment) Object {
	rows, err := tb.query(query, params)
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, colErr := rows.Columns()
	if colErr != nil {
		return newDatabaseError("DB-0008", colErr)
	}

	var results []*Dictionary
	for rows.Next() {
		values := make([]any, len(columns))
		ptrs := make([]any, len(columns))
		for i := range values {
			ptrs[i] = &values[i]
		}

		if scanErr := rows.Scan(ptrs...); scanErr != nil {
			return newDatabaseError("DB-0004", scanErr)
		}

		results = append(results, rowToDict(columns, values, env))
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return newDatabaseError("DB-0002", rowsErr)
	}

	// Return Table with schema from TableBinding
	// FromDB=true indicates data came from database (records are auto-validated)
	return &Table{Rows: results, Columns: columns, Schema: tb.DSLSchema, FromDB: true}
}

// rowDictToRecord converts a Dictionary row to a validated Record.
// Records from database queries are auto-validated (SPEC-DB-VAL-001/002/003).
// The fromDB flag indicates the data came from the database (trusted).
func (tb *TableBinding) rowDictToRecord(dict *Dictionary, env *Environment, fromDB bool) Object {
	if tb.DSLSchema == nil {
		return dict // No schema, return as dictionary
	}

	// Create a Record from the dictionary
	record := CreateRecord(tb.DSLSchema, dict, env)

	if fromDB {
		// Data from database is trusted - mark as validated with no errors
		// Implements SPEC-DB-VAL-001, SPEC-DB-VAL-002, SPEC-DB-VAL-003
		record.Validated = true
		record.Errors = nil // No errors for trusted DB data
	}

	return record
}

// querySingleRow executes a query and returns a single Record (or null).
// This is used by find(), first(), last(), findBy() methods.
// Implements SPEC-DB-001: Query ?-> * returns Record
func (tb *TableBinding) querySingleRow(query string, params []Object, env *Environment) Object {
	result := tb.queryRows(query, params, env)
	if tbl, ok := result.(*Table); ok {
		if len(tbl.Rows) == 0 {
			return NULL
		}
		// Convert first row to Record if schema is available
		return tb.rowDictToRecord(tbl.Rows[0], env, true)
	}
	return result
}

func (tb *TableBinding) executeMutation(query string, params []Object) *Error {
	if len(params) == 0 {
		return newValidationError("VAL-0011", map[string]any{"Function": "mutation"})
	}
	_, err := tb.exec(query, params)
	if err != nil {
		return err
	}
	return nil
}

func (tb *TableBinding) query(query string, params []Object) (*RowsWrapper, *Error) {
	goParams := make([]any, len(params))
	for i, p := range params {
		goParams[i] = objectToGoValue(p)
	}

	// Use transaction if active, otherwise use the DB connection
	var rows *sql.Rows
	var err error
	if tb.DB.Tx != nil {
		rows, err = tb.DB.Tx.Query(query, goParams...)
	} else {
		rows, err = tb.DB.DB.Query(query, goParams...)
	}
	if err != nil {
		tb.DB.LastError = err.Error()
		return nil, newDatabaseError("DB-0002", err)
	}
	return &RowsWrapper{Rows: rows}, nil
}

func (tb *TableBinding) exec(query string, params []Object) (ResultWrapper, *Error) {
	goParams := make([]any, len(params))
	for i, p := range params {
		goParams[i] = objectToGoValue(p)
	}

	// Use transaction if active, otherwise use the DB connection
	var res sql.Result
	var err error
	if tb.DB.Tx != nil {
		res, err = tb.DB.Tx.Exec(query, goParams...)
	} else {
		res, err = tb.DB.DB.Exec(query, goParams...)
	}
	if err != nil {
		tb.DB.LastError = err.Error()
		return nil, newDatabaseError("DB-0011", err)
	}
	return res, nil
}

// generateID creates an id based on the schema's id format. Defaults to ulid.
func (tb *TableBinding) generateID(env *Environment) Object {
	// For DSL schemas (@schema), determine ID format from field type
	if tb.DSLSchema != nil {
		idField, ok := tb.DSLSchema.Fields["id"]
		if !ok {
			return nil
		}

		// SPEC-ID-004: Only auto-generate ID if field has auto constraint
		if !idField.Auto {
			return nil
		}

		// Map DSL schema type to ID format
		format := idField.Type
		switch format {
		case "uuid", "uuidv4":
			return idUUID()
		case "uuidv7":
			return idUUIDv7()
		case "nanoid":
			return idNanoID()
		case "cuid":
			return idCUID()
		case "ulid", "id", "":
			return idNew()
		default:
			// For other types like "int", don't generate an ID
			return nil
		}
	}

	// For old-style dictionary schemas (schema.define)
	if tb.Schema == nil {
		return nil
	}

	fieldsExpr, ok := tb.Schema.Pairs["fields"]
	if !ok {
		return nil
	}
	fieldsObj := Eval(fieldsExpr, tb.Schema.Env)
	fields, ok := fieldsObj.(*Dictionary)
	if !ok {
		return nil
	}

	idSpecExpr, ok := fields.Pairs["id"]
	if !ok {
		return nil
	}
	idSpecObj := Eval(idSpecExpr, fields.Env)
	idSpec, ok := idSpecObj.(*Dictionary)
	if !ok {
		return nil
	}

	format := "ulid"
	if formatExpr, ok := idSpec.Pairs["format"]; ok {
		if formatObj := Eval(formatExpr, idSpec.Env); formatObj.Type() == STRING_OBJ {
			format = strings.ToLower(formatObj.(*String).Value)
		}
	}

	switch format {
	case "uuid", "uuidv4":
		return idUUID()
	case "uuidv7":
		return idUUIDv7()
	case "nanoid":
		return idNanoID()
	case "cuid":
		return idCUID()
	default:
		return idNew()
	}
}

// executeCount returns the count of rows, optionally filtered by conditions.
func (tb *TableBinding) executeCount(args []Object, env *Environment) Object {
	if err := tb.ensureSQLite(); err != nil {
		return err
	}

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tb.TableName)
	var params []Object

	if len(args) > 0 {
		if condDict, ok := args[0].(*Dictionary); ok {
			conditions, whereParams, errObj := tb.buildWhereClause(condDict)
			if errObj != nil {
				return errObj
			}
			if conditions != "" {
				query += " WHERE " + conditions
			}
			params = whereParams
		}
	}

	return tb.querySingleValue(query, params)
}

// executeAggregate handles SUM, AVG, MIN, MAX aggregations.
func (tb *TableBinding) executeAggregate(aggFunc string, args []Object, env *Environment) Object {
	if err := tb.ensureSQLite(); err != nil {
		return err
	}
	if len(args) < 1 {
		return newArityError(strings.ToLower(aggFunc), len(args), 1)
	}

	// First arg must be column name
	colStr, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0005", strings.ToLower(aggFunc), "a string (column name)", args[0].Type())
	}
	if !identifierRegex.MatchString(colStr.Value) {
		return newValidationError("VAL-0003", map[string]any{"Pattern": "identifier", "GoError": fmt.Sprintf("invalid column name: %s", colStr.Value)})
	}

	query := fmt.Sprintf("SELECT %s(%s) FROM %s", aggFunc, colStr.Value, tb.TableName)
	var params []Object

	// Optional second arg is conditions dict
	if len(args) > 1 {
		if condDict, ok := args[1].(*Dictionary); ok {
			conditions, whereParams, errObj := tb.buildWhereClause(condDict)
			if errObj != nil {
				return errObj
			}
			if conditions != "" {
				query += " WHERE " + conditions
			}
			params = whereParams
		}
	}

	return tb.querySingleValue(query, params)
}

// executeFirst returns the first record(s) ordered by id ASC.
// first() → single record or null
// first(n) → array of up to n records
// first({orderBy: ...}) → single record with custom order
// first(n, {orderBy: ...}) → array with custom order
// Returns Record(s) if schema is bound. Implements SPEC-DB-001.
func (tb *TableBinding) executeFirst(args []Object, env *Environment) Object {
	if err := tb.ensureSQLite(); err != nil {
		return err
	}

	limit := int64(1)
	returnSingle := true
	var opts *QueryOptions

	// Parse arguments
	for i, arg := range args {
		switch v := arg.(type) {
		case *Integer:
			if i == 0 {
				limit = v.Value
				returnSingle = false
			}
		case *Dictionary:
			var parseErr *Error
			opts, parseErr = parseQueryOptions(v)
			if parseErr != nil {
				return parseErr
			}
		}
	}

	// Default ORDER BY id ASC if not specified
	if opts == nil {
		opts = &QueryOptions{}
	}
	if len(opts.OrderBy) == 0 {
		opts.OrderBy = []OrderSpec{{Column: "id", Dir: "ASC"}}
	}

	selectCols := buildSelectClause(opts)
	query := fmt.Sprintf("SELECT %s FROM %s", selectCols, tb.TableName)
	query += buildOrderByClause(opts)
	query += " LIMIT ?"

	result := tb.queryRows(query, []Object{&Integer{Value: limit}}, env)
	// queryRows now returns Table
	if tbl, ok := result.(*Table); ok {
		if returnSingle {
			if len(tbl.Rows) == 0 {
				return NULL
			}
			// Return Record if schema is bound
			return tb.rowDictToRecord(tbl.Rows[0], env, true)
		}
	}
	return result
}

// executeLast returns the last record(s) ordered by id DESC.
// Same signature as first() but reverses order direction.
// Returns Record(s) if schema is bound. Implements SPEC-DB-001.
func (tb *TableBinding) executeLast(args []Object, env *Environment) Object {
	if err := tb.ensureSQLite(); err != nil {
		return err
	}

	limit := int64(1)
	returnSingle := true
	var opts *QueryOptions

	// Parse arguments
	for i, arg := range args {
		switch v := arg.(type) {
		case *Integer:
			if i == 0 {
				limit = v.Value
				returnSingle = false
			}
		case *Dictionary:
			var parseErr *Error
			opts, parseErr = parseQueryOptions(v)
			if parseErr != nil {
				return parseErr
			}
		}
	}

	// Default ORDER BY id DESC if not specified
	if opts == nil {
		opts = &QueryOptions{}
	}
	if len(opts.OrderBy) == 0 {
		opts.OrderBy = []OrderSpec{{Column: "id", Dir: "DESC"}}
	} else {
		// Reverse all directions for last()
		for i := range opts.OrderBy {
			if opts.OrderBy[i].Dir == "ASC" {
				opts.OrderBy[i].Dir = "DESC"
			} else {
				opts.OrderBy[i].Dir = "ASC"
			}
		}
	}

	selectCols := buildSelectClause(opts)
	query := fmt.Sprintf("SELECT %s FROM %s", selectCols, tb.TableName)
	query += buildOrderByClause(opts)
	query += " LIMIT ?"

	result := tb.queryRows(query, []Object{&Integer{Value: limit}}, env)
	// queryRows now returns Table
	if tbl, ok := result.(*Table); ok {
		if returnSingle {
			if len(tbl.Rows) == 0 {
				return NULL
			}
			// Return Record if schema is bound
			return tb.rowDictToRecord(tbl.Rows[0], env, true)
		}
	}
	return result
}

// executeExists checks if any matching record exists. Returns boolean.
func (tb *TableBinding) executeExists(args []Object, env *Environment) Object {
	if err := tb.ensureSQLite(); err != nil {
		return err
	}
	if len(args) != 1 {
		return newArityError("exists", len(args), 1)
	}

	condDict, ok := args[0].(*Dictionary)
	if !ok {
		return newTypeError("TYPE-0005", "exists", "a dictionary", args[0].Type())
	}

	conditions, params, errObj := tb.buildWhereClause(condDict)
	if errObj != nil {
		return errObj
	}

	query := fmt.Sprintf("SELECT 1 FROM %s", tb.TableName)
	if conditions != "" {
		query += " WHERE " + conditions
	}
	query += " LIMIT 1"

	rows, err := tb.query(query, params)
	if err != nil {
		return err
	}
	defer rows.Close()

	exists := rows.Next()
	return &Boolean{Value: exists}
}

// executeFindBy returns a single matching record or null.
// Like where() but returns first match, not an array.
// Returns Record if schema is bound. Implements SPEC-DB-001.
func (tb *TableBinding) executeFindBy(args []Object, env *Environment) Object {
	if err := tb.ensureSQLite(); err != nil {
		return err
	}
	if len(args) < 1 || len(args) > 2 {
		return newArityError("findBy", len(args), 1)
	}

	condDict, ok := args[0].(*Dictionary)
	if !ok {
		return newTypeError("TYPE-0005", "findBy", "a dictionary", args[0].Type())
	}

	// Parse options if provided
	var opts *QueryOptions
	if len(args) == 2 {
		if optsDict, ok := args[1].(*Dictionary); ok {
			var parseErr *Error
			opts, parseErr = parseQueryOptions(optsDict)
			if parseErr != nil {
				return parseErr
			}
		}
	}

	conditions, params, errObj := tb.buildWhereClause(condDict)
	if errObj != nil {
		return errObj
	}

	selectCols := buildSelectClause(opts)
	query := fmt.Sprintf("SELECT %s FROM %s", selectCols, tb.TableName)
	if conditions != "" {
		query += " WHERE " + conditions
	}
	query += buildOrderByClause(opts)
	query += " LIMIT 1"

	result := tb.queryRows(query, params, env)
	// queryRows now returns Table
	if tbl, ok := result.(*Table); ok {
		if len(tbl.Rows) == 0 {
			return NULL
		}
		// Return Record if schema is bound
		return tb.rowDictToRecord(tbl.Rows[0], env, true)
	}
	return result
}

// querySingleValue executes a query that returns a single scalar value.
func (tb *TableBinding) querySingleValue(query string, params []Object) Object {
	goParams := make([]any, len(params))
	for i, p := range params {
		goParams[i] = objectToGoValue(p)
	}

	var result any
	err := tb.DB.DB.QueryRow(query, goParams...).Scan(&result)
	if err != nil {
		if err == sql.ErrNoRows {
			return NULL
		}
		tb.DB.LastError = err.Error()
		return newDatabaseError("DB-0002", err)
	}

	if result == nil {
		return NULL
	}

	// Convert to appropriate Parsley type
	switch v := result.(type) {
	case int64:
		return &Integer{Value: v}
	case float64:
		return &Float{Value: v}
	case string:
		return &String{Value: v}
	default:
		// SQLite often returns int64 for COUNT, but let's handle other cases
		if i, ok := v.(int); ok {
			return &Integer{Value: int64(i)}
		}
		return &String{Value: fmt.Sprintf("%v", v)}
	}
}

// getPagination reads limit/offset from request query if present, applying defaults and caps.
func getPagination(env *Environment) (int64, int64, bool) {
	// Defaults
	limit := int64(20)
	offset := int64(0)
	useLimit := true

	query := getRequestQuery(env)

	if limStr, ok := query["limit"]; ok {
		if lim, err := strconv.ParseInt(limStr, 10, 64); err == nil {
			if lim <= 0 {
				useLimit = false
			} else {
				if lim > 100 {
					lim = 100
				}
				limit = lim
			}
		}
	}

	if offStr, ok := query["offset"]; ok {
		if off, err := strconv.ParseInt(offStr, 10, 64); err == nil && off >= 0 {
			offset = off
		}
	}

	return limit, offset, useLimit
}

// getRequestQuery extracts the request query map from basil.http.request.query if present.
func getRequestQuery(env *Environment) map[string]string {
	result := make(map[string]string)

	basilObj, ok := env.Get("basil")
	if !ok {
		return result
	}

	basilDict, ok := basilObj.(*Dictionary)
	if !ok {
		return result
	}

	httpExpr, ok := basilDict.Pairs["http"]
	if !ok {
		return result
	}
	httpObj := Eval(httpExpr, basilDict.Env)
	httpDict, ok := httpObj.(*Dictionary)
	if !ok {
		return result
	}

	reqExpr, ok := httpDict.Pairs["request"]
	if !ok {
		return result
	}
	reqObj := Eval(reqExpr, httpDict.Env)
	reqDict, ok := reqObj.(*Dictionary)
	if !ok {
		return result
	}

	queryExpr, ok := reqDict.Pairs["query"]
	if !ok {
		return result
	}
	queryObj := Eval(queryExpr, reqDict.Env)
	queryDict, ok := queryObj.(*Dictionary)
	if !ok {
		return result
	}

	for key, expr := range queryDict.Pairs {
		valObj := Eval(expr, queryDict.Env)
		if str, ok := valObj.(*String); ok {
			result[key] = str.Value
		}
	}

	return result
}

// reverseDirection reverses ASC <-> DESC for ORDER BY
func reverseDirection(dir string) string {
	if dir == "ASC" {
		return "DESC"
	}
	return "ASC"
}

// executeToSQL returns the SQL query and parameters that would be executed
// for the given query method, without actually executing it. Accepts method name
// as first argument, followed by method-specific arguments.
//
// Usage:
//
//	Users.toSQL("all", {orderBy: "name", limit: 10})
//	Users.toSQL("where", {status: "active"})
//	Users.toSQL("find", 42)
//
// Returns: {sql: "SELECT ...", params: [...]}
func (tb *TableBinding) executeToSQL(args []Object, env *Environment) Object {
	if len(args) < 1 {
		return newArityError("toSQL", len(args), 1)
	}

	methodName, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0005", "toSQL", "a string (method name)", args[0].Type())
	}

	methodArgs := args[1:]

	// Build SQL based on method type
	var sqlStr string
	var params []Object

	switch methodName.Value {
	case "all":
		// Build SELECT with optional orderBy, select, limit, offset
		var opts *QueryOptions
		if len(methodArgs) > 0 {
			if dict, ok := methodArgs[0].(*Dictionary); ok {
				var parseErr *Error
				opts, parseErr = parseQueryOptions(dict)
				if parseErr != nil {
					return parseErr
				}
			}
		}

		selectCols := buildSelectClause(opts)
		sqlStr = fmt.Sprintf("SELECT %s FROM %s", selectCols, tb.TableName)
		sqlStr += buildOrderByClause(opts)

		// Handle pagination
		if opts != nil && opts.NoLimit {
			// No limit
		} else if opts != nil && opts.Limit != nil {
			offset := int64(0)
			if opts.Offset != nil {
				offset = *opts.Offset
			}
			sqlStr += " LIMIT ? OFFSET ?"
			params = append(params, &Integer{Value: *opts.Limit}, &Integer{Value: offset})
		} else {
			// Would use auto-pagination from request in actual execution
			limit, offset, useLimit := getPagination(env)
			if useLimit {
				sqlStr += " LIMIT ? OFFSET ?"
				params = append(params, &Integer{Value: limit}, &Integer{Value: offset})
			}
		}

	case "where":
		// Build SELECT with WHERE clause
		if len(methodArgs) < 1 {
			return newArityError("where", len(methodArgs), 1)
		}

		condDict, ok := methodArgs[0].(*Dictionary)
		if !ok {
			return newTypeError("TYPE-0005", "where", "a dictionary", methodArgs[0].Type())
		}

		var opts *QueryOptions
		if len(methodArgs) == 2 {
			if optsDict, ok := methodArgs[1].(*Dictionary); ok {
				var parseErr *Error
				opts, parseErr = parseQueryOptions(optsDict)
				if parseErr != nil {
					return parseErr
				}
			}
		}

		conditions, whereParams, errObj := tb.buildWhereClause(condDict)
		if errObj != nil {
			return errObj
		}

		selectCols := buildSelectClause(opts)
		sqlStr = fmt.Sprintf("SELECT %s FROM %s", selectCols, tb.TableName)
		if conditions != "" {
			sqlStr += " WHERE " + conditions
		}
		sqlStr += buildOrderByClause(opts)

		params = whereParams

		if opts != nil && opts.Limit != nil {
			offset := int64(0)
			if opts.Offset != nil {
				offset = *opts.Offset
			}
			sqlStr += " LIMIT ? OFFSET ?"
			params = append(params, &Integer{Value: *opts.Limit}, &Integer{Value: offset})
		}

	case "find":
		// Build SELECT with id = ?
		if len(methodArgs) != 1 {
			return newArityError("find", len(methodArgs), 1)
		}
		sqlStr = fmt.Sprintf("SELECT * FROM %s WHERE id = ? LIMIT 1", tb.TableName)
		params = []Object{methodArgs[0]}

	case "count":
		// Build COUNT(*) with optional WHERE
		if len(methodArgs) == 0 {
			sqlStr = fmt.Sprintf("SELECT COUNT(*) FROM %s", tb.TableName)
		} else if len(methodArgs) == 1 {
			condDict, ok := methodArgs[0].(*Dictionary)
			if !ok {
				return newTypeError("TYPE-0005", "count", "a dictionary", methodArgs[0].Type())
			}
			conditions, whereParams, errObj := tb.buildWhereClause(condDict)
			if errObj != nil {
				return errObj
			}
			sqlStr = fmt.Sprintf("SELECT COUNT(*) FROM %s", tb.TableName)
			if conditions != "" {
				sqlStr += " WHERE " + conditions
			}
			params = whereParams
		} else {
			return newArityError("count", len(methodArgs), 1)
		}

	case "sum", "avg", "min", "max":
		// Build aggregate with optional WHERE
		if len(methodArgs) < 1 || len(methodArgs) > 2 {
			return newArityError(methodName.Value, len(methodArgs), 1)
		}

		colName, ok := methodArgs[0].(*String)
		if !ok {
			return newTypeError("TYPE-0005", methodName.Value, "a string (column name)", methodArgs[0].Type())
		}

		sqlStr = fmt.Sprintf("SELECT %s(%s) FROM %s", strings.ToUpper(methodName.Value), colName.Value, tb.TableName)

		if len(methodArgs) == 2 {
			condDict, ok := methodArgs[1].(*Dictionary)
			if !ok {
				return newTypeError("TYPE-0005", methodName.Value, "a dictionary", methodArgs[1].Type())
			}
			conditions, whereParams, errObj := tb.buildWhereClause(condDict)
			if errObj != nil {
				return errObj
			}
			if conditions != "" {
				sqlStr += " WHERE " + conditions
			}
			params = whereParams
		}

	case "first":
		// Build SELECT with ORDER BY and LIMIT 1
		var opts *QueryOptions
		if len(methodArgs) > 0 {
			if dict, ok := methodArgs[0].(*Dictionary); ok {
				var parseErr *Error
				opts, parseErr = parseQueryOptions(dict)
				if parseErr != nil {
					return parseErr
				}
			}
		}

		selectCols := buildSelectClause(opts)
		sqlStr = fmt.Sprintf("SELECT %s FROM %s", selectCols, tb.TableName)
		sqlStr += buildOrderByClause(opts)
		sqlStr += " LIMIT 1"

	case "last":
		// Build SELECT with ORDER BY DESC and LIMIT 1
		var opts *QueryOptions
		if len(methodArgs) > 0 {
			if dict, ok := methodArgs[0].(*Dictionary); ok {
				var parseErr *Error
				opts, parseErr = parseQueryOptions(dict)
				if parseErr != nil {
					return parseErr
				}
			}
		}

		selectCols := buildSelectClause(opts)
		sqlStr = fmt.Sprintf("SELECT %s FROM %s", selectCols, tb.TableName)
		// Reverse ORDER BY directions for last()
		if opts != nil && len(opts.OrderBy) > 0 {
			reversedSpecs := make([]OrderSpec, len(opts.OrderBy))
			for i, spec := range opts.OrderBy {
				reversedSpecs[i] = OrderSpec{
					Column: spec.Column,
					Dir:    reverseDirection(spec.Dir),
				}
			}
			opts.OrderBy = reversedSpecs
			sqlStr += buildOrderByClause(opts)
		}
		sqlStr += " LIMIT 1"

	case "exists":
		// Build SELECT 1 FROM ... LIMIT 1
		if len(methodArgs) != 1 {
			return newArityError("exists", len(methodArgs), 1)
		}

		condDict, ok := methodArgs[0].(*Dictionary)
		if !ok {
			return newTypeError("TYPE-0005", "exists", "a dictionary", methodArgs[0].Type())
		}

		conditions, whereParams, errObj := tb.buildWhereClause(condDict)
		if errObj != nil {
			return errObj
		}

		sqlStr = fmt.Sprintf("SELECT 1 FROM %s", tb.TableName)
		if conditions != "" {
			sqlStr += " WHERE " + conditions
		}
		sqlStr += " LIMIT 1"
		params = whereParams

	case "findBy":
		// Build SELECT with custom WHERE and LIMIT 1
		if len(methodArgs) != 1 {
			return newArityError("findBy", len(methodArgs), 1)
		}

		condDict, ok := methodArgs[0].(*Dictionary)
		if !ok {
			return newTypeError("TYPE-0005", "findBy", "a dictionary", methodArgs[0].Type())
		}

		conditions, whereParams, errObj := tb.buildWhereClause(condDict)
		if errObj != nil {
			return errObj
		}

		sqlStr = fmt.Sprintf("SELECT * FROM %s", tb.TableName)
		if conditions != "" {
			sqlStr += " WHERE " + conditions
		}
		sqlStr += " LIMIT 1"
		params = whereParams

	default:
		return &Error{
			Message: fmt.Sprintf("toSQL does not support method '%s'", methodName.Value),
			Class:   ClassType,
			Code:    "TYPE-0022",
			Hints:   []string{"Supported methods: all, where, find, count, sum, avg, min, max, first, last, exists, findBy"},
		}
	}

	// Build params array
	paramsArray := &Array{Elements: make([]Object, len(params))}
	for i, p := range params {
		// Convert Integer objects to their values for display
		if intObj, ok := p.(*Integer); ok {
			paramsArray.Elements[i] = intObj
		} else {
			paramsArray.Elements[i] = p
		}
	}

	// Return dictionary with sql and params
	result := &Dictionary{
		Pairs: make(map[string]ast.Expression),
		Env:   env,
	}
	result.SetKey("sql", objectToExpression(&String{Value: sqlStr}))
	result.SetKey("params", objectToExpression(paramsArray))
	return result
}

// RowsWrapper allows queryRows to defer closing while returning *sql.Rows-like interface.
type RowsWrapper struct {
	Rows *sql.Rows
}

func (rw *RowsWrapper) Columns() ([]string, error) { return rw.Rows.Columns() }
func (rw *RowsWrapper) Next() bool                 { return rw.Rows.Next() }
func (rw *RowsWrapper) Scan(dest ...any) error     { return rw.Rows.Scan(dest...) }
func (rw *RowsWrapper) Err() error                 { return rw.Rows.Err() }
func (rw *RowsWrapper) Close() error               { return rw.Rows.Close() }

// ResultWrapper mirrors sql.Result for testability and decoupling.
type ResultWrapper interface {
	RowsAffected() (int64, error)
	LastInsertId() (int64, error)
}
