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
	DB        *DBConnection
	Schema    *Dictionary
	TableName string
}

func (tb *TableBinding) Type() ObjectType { return TABLE_BINDING_OBJ }
func (tb *TableBinding) Inspect() string  { return fmt.Sprintf("TableBinding(%s)", tb.TableName) }

// evalTableBindingMethod dispatches method calls on TableBinding instances.
func evalTableBindingMethod(tb *TableBinding, method string, args []Object, env *Environment) Object {
	switch method {
	case "all":
		return tb.executeAll(env)
	case "find":
		return tb.executeFind(args, env)
	case "where":
		return tb.executeWhere(args, env)
	case "insert":
		return tb.executeInsert(args, env)
	case "update":
		return tb.executeUpdate(args, env)
	case "delete":
		return tb.executeDelete(args, env)
	default:
		return unknownMethodError(method, "TableBinding", []string{"all", "find", "where", "insert", "update", "delete"})
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

// executeAll selects all rows with optional pagination defaults.
func (tb *TableBinding) executeAll(env *Environment) Object {
	if err := tb.ensureSQLite(); err != nil {
		return err
	}

	limit, offset, useLimit := getPagination(env)
	query := fmt.Sprintf("SELECT * FROM %s", tb.TableName)
	var params []Object

	if useLimit {
		query = query + " LIMIT ? OFFSET ?"
		params = append(params, &Integer{Value: limit}, &Integer{Value: offset})
	}

	return tb.queryRows(query, params, env)
}

// executeFind selects a single row by id.
func (tb *TableBinding) executeFind(args []Object, env *Environment) Object {
	if err := tb.ensureSQLite(); err != nil {
		return err
	}
	if len(args) != 1 {
		return newArityError("find", len(args), 1)
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE id = ? LIMIT 1", tb.TableName)
	result := tb.queryRows(query, []Object{args[0]}, env)
	if arr, ok := result.(*Array); ok {
		if len(arr.Elements) == 0 {
			return NULL
		}
		return arr.Elements[0]
	}
	return result
}

// executeWhere selects rows matching equality conditions from a dictionary.
// Unlike all(), where() does not apply automatic pagination.
func (tb *TableBinding) executeWhere(args []Object, env *Environment) Object {
	if err := tb.ensureSQLite(); err != nil {
		return err
	}
	if len(args) != 1 {
		return newArityError("where", len(args), 1)
	}

	condDict, ok := args[0].(*Dictionary)
	if !ok {
		return newTypeError("TYPE-0005", "where", "a dictionary", args[0].Type())
	}

	conditions, params, errObj := tb.buildWhereClause(condDict)
	if errObj != nil {
		return errObj
	}

	query := fmt.Sprintf("SELECT * FROM %s", tb.TableName)
	if conditions != "" {
		query += " WHERE " + conditions
	}

	return tb.queryRows(query, params, env)
}

// executeInsert validates and inserts a row, auto-generating an id when needed.
func (tb *TableBinding) executeInsert(args []Object, env *Environment) Object {
	if err := tb.ensureSQLite(); err != nil {
		return err
	}
	if len(args) != 1 {
		return newArityError("insert", len(args), 1)
	}
	data, ok := args[0].(*Dictionary)
	if !ok {
		return newTypeError("TYPE-0005", "insert", "a dictionary", args[0].Type())
	}

	// Add id if missing
	if _, exists := data.Pairs["id"]; !exists {
		if idObj := tb.generateID(env); idObj != nil {
			data.SetKey("id", objectToExpression(idObj))
		}
	}

	// Validate
	validation := schemaValidate(tb.Schema, data)
	if dict, ok := validation.(*Dictionary); ok {
		if validExpr, hasValid := dict.Pairs["valid"]; hasValid {
			if validObj := Eval(validExpr, dict.Env); validObj.Type() == BOOLEAN_OBJ && !validObj.(*Boolean).Value {
				return dict
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
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", tb.TableName, strings.Join(columns, ", "), strings.Join(placeholders, ", "))

	if execErr := tb.executeMutation(query, params); execErr != nil {
		return execErr
	}

	// Return inserted row
	if idExpr, ok := data.Pairs["id"]; ok {
		idVal := Eval(idExpr, data.Env)
		return tb.executeFind([]Object{idVal}, env)
	}
	return NULL
}

// executeUpdate validates and updates a row by id.
func (tb *TableBinding) executeUpdate(args []Object, env *Environment) Object {
	if err := tb.ensureSQLite(); err != nil {
		return err
	}
	if len(args) != 2 {
		return newArityError("update", len(args), 2)
	}

	idObj := args[0]
	data, ok := args[1].(*Dictionary)
	if !ok {
		return newTypeError("TYPE-0006", "update", "a dictionary (data)", args[1].Type())
	}

	// Disallow id changes for safety
	if _, hasID := data.Pairs["id"]; hasID {
		return newValidationError("VAL-0004", map[string]any{"Method": "update", "Got": "id"})
	}

	validation := schemaValidate(tb.Schema, data)
	if dict, ok := validation.(*Dictionary); ok {
		if validExpr, hasValid := dict.Pairs["valid"]; hasValid {
			if validObj := Eval(validExpr, dict.Env); validObj.Type() == BOOLEAN_OBJ && !validObj.(*Boolean).Value {
				return dict
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

// executeDelete deletes a row by id and returns affected count.
func (tb *TableBinding) executeDelete(args []Object, env *Environment) Object {
	if err := tb.ensureSQLite(); err != nil {
		return err
	}
	if len(args) != 1 {
		return newArityError("delete", len(args), 1)
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", tb.TableName)
	result, err := tb.exec(query, []Object{args[0]})
	if err != nil {
		return err
	}

	affected, _ := result.RowsAffected()
	pairs := map[string]ast.Expression{
		"affected": objectToExpression(&Integer{Value: affected}),
	}
	return &Dictionary{Pairs: pairs, Env: env}
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

	var results []Object
	for rows.Next() {
		values := make([]interface{}, len(columns))
		ptrs := make([]interface{}, len(columns))
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

	return &Array{Elements: results}
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
	goParams := make([]interface{}, len(params))
	for i, p := range params {
		goParams[i] = objectToGoValue(p)
	}

	rows, err := tb.DB.DB.Query(query, goParams...)
	if err != nil {
		tb.DB.LastError = err.Error()
		return nil, newDatabaseError("DB-0002", err)
	}
	return &RowsWrapper{Rows: rows}, nil
}

func (tb *TableBinding) exec(query string, params []Object) (ResultWrapper, *Error) {
	goParams := make([]interface{}, len(params))
	for i, p := range params {
		goParams[i] = objectToGoValue(p)
	}

	res, err := tb.DB.DB.Exec(query, goParams...)
	if err != nil {
		tb.DB.LastError = err.Error()
		return nil, newDatabaseError("DB-0011", err)
	}
	return res, nil
}

// generateID creates an id based on the schema's id format. Defaults to ulid.
func (tb *TableBinding) generateID(env *Environment) Object {
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

// RowsWrapper allows queryRows to defer closing while returning *sql.Rows-like interface.
type RowsWrapper struct {
	Rows *sql.Rows
}

func (rw *RowsWrapper) Columns() ([]string, error)     { return rw.Rows.Columns() }
func (rw *RowsWrapper) Next() bool                     { return rw.Rows.Next() }
func (rw *RowsWrapper) Scan(dest ...interface{}) error { return rw.Rows.Scan(dest...) }
func (rw *RowsWrapper) Err() error                     { return rw.Rows.Err() }
func (rw *RowsWrapper) Close() error                   { return rw.Rows.Close() }

// ResultWrapper mirrors sql.Result for testability and decoupling.
type ResultWrapper interface {
	RowsAffected() (int64, error)
	LastInsertId() (int64, error)
}
