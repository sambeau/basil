package evaluator

import (
	"sort"
	"strconv"

	"github.com/sambeau/basil/pkg/parsley/ast"
	perrors "github.com/sambeau/basil/pkg/parsley/errors"
	"github.com/sambeau/basil/pkg/parsley/lexer"
)

// Database query operations and SQL execution

func evalQueryOneStatement(node *ast.QueryOneStatement, env *Environment) Object {
	// Evaluate the connection
	connObj := Eval(node.Connection, env)
	if isError(connObj) {
		return connObj
	}

	conn, ok := connObj.(*DBConnection)
	if !ok {
		perr := perrors.New("DB-0012", map[string]any{"Operator": "query operator <=?=>", "Got": string(connObj.Type())})
		return &Error{Class: ErrorClass(perr.Class), Code: perr.Code, Message: perr.Message, Hints: perr.Hints, Data: perr.Data}
	}

	// Evaluate the query expression (should return a tag with SQL and params)
	queryObj := Eval(node.Query, env)
	if isError(queryObj) {
		return queryObj
	}

	// Extract SQL and params from the query object
	sql, params, err := extractSQLAndParams(queryObj, env)
	if err != nil {
		return err
	}

	// Execute the query
	// For QueryRow, we need to get column info, so we use Query instead
	rows, queryErr := conn.DB.Query(sql, params...)
	if queryErr != nil {
		conn.LastError = queryErr.Error()
		return newDatabaseError("DB-0002", queryErr)
	}
	defer rows.Close()

	// Get column names
	columns, colErr := rows.Columns()
	if colErr != nil {
		conn.LastError = colErr.Error()
		return newDatabaseError("DB-0008", colErr)
	}

	// Check if there's a row
	if !rows.Next() {
		// No rows - return null
		return assignQueryResult(node.Names, NULL, env, node.IsLet)
	}

	// Scan the row into a map
	values := make([]any, len(columns))
	valuePtrs := make([]any, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	if scanErr := rows.Scan(valuePtrs...); scanErr != nil {
		conn.LastError = scanErr.Error()
		return newDatabaseError("DB-0004", scanErr)
	}

	// Convert to dictionary
	resultDict := rowToDict(columns, values, env)

	return assignQueryResult(node.Names, resultDict, env, node.IsLet)
}

// evalQueryManyStatement evaluates the <=??=> operator to query multiple rows
func evalQueryManyStatement(node *ast.QueryManyStatement, env *Environment) Object {
	// Evaluate the connection
	connObj := Eval(node.Connection, env)
	if isError(connObj) {
		return connObj
	}

	conn, ok := connObj.(*DBConnection)
	if !ok {
		perr := perrors.New("DB-0012", map[string]any{"Operator": "query operator <=??=>", "Got": string(connObj.Type())})
		return &Error{Class: ErrorClass(perr.Class), Code: perr.Code, Message: perr.Message, Hints: perr.Hints, Data: perr.Data}
	}

	// Evaluate the query expression
	queryObj := Eval(node.Query, env)
	if isError(queryObj) {
		return queryObj
	}

	// Extract SQL and params
	sql, params, err := extractSQLAndParams(queryObj, env)
	if err != nil {
		return err
	}

	// Execute the query
	rows, queryErr := conn.DB.Query(sql, params...)
	if queryErr != nil {
		conn.LastError = queryErr.Error()
		return newDatabaseError("DB-0002", queryErr)
	}
	defer rows.Close()

	// Get column names
	columns, colErr := rows.Columns()
	if colErr != nil {
		conn.LastError = colErr.Error()
		return newDatabaseError("DB-0008", colErr)
	}

	// Scan all rows
	var results []*Dictionary
	for rows.Next() {
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if scanErr := rows.Scan(valuePtrs...); scanErr != nil {
			conn.LastError = scanErr.Error()
			return newDatabaseError("DB-0004", scanErr)
		}

		resultDict := rowToDict(columns, values, env)
		results = append(results, resultDict)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		conn.LastError = rowsErr.Error()
		return newDatabaseError("DB-0002", rowsErr)
	}

	// Return a Table with column info
	resultTable := &Table{Rows: results, Columns: columns}
	return assignQueryResult(node.Names, resultTable, env, node.IsLet)
}

// evalExecuteStatement evaluates the <=!=> operator to execute mutations
func evalExecuteStatement(node *ast.ExecuteStatement, env *Environment) Object {
	// Evaluate the connection
	connObj := Eval(node.Connection, env)
	if isError(connObj) {
		return connObj
	}

	conn, ok := connObj.(*DBConnection)
	if !ok {
		perr := perrors.New("DB-0012", map[string]any{"Operator": "execute operator <=!=>", "Got": string(connObj.Type())})
		return &Error{Class: ErrorClass(perr.Class), Code: perr.Code, Message: perr.Message, Hints: perr.Hints, Data: perr.Data}
	}

	// Evaluate the query expression
	queryObj := Eval(node.Query, env)
	if isError(queryObj) {
		return queryObj
	}

	// Extract SQL and params
	sql, params, err := extractSQLAndParams(queryObj, env)
	if err != nil {
		return err
	}

	// Execute the statement
	result, execErr := conn.DB.Exec(sql, params...)
	if execErr != nil {
		conn.LastError = execErr.Error()
		return newDatabaseError("DB-0011", execErr)
	}

	// Get affected rows and last insert ID
	affected, _ := result.RowsAffected()
	lastId, _ := result.LastInsertId()

	// Return result as dictionary
	resultDict := &Dictionary{
		Pairs: map[string]ast.Expression{
			"affected": &ast.IntegerLiteral{
				Token: lexer.Token{Type: lexer.INT, Literal: strconv.FormatInt(affected, 10)},
				Value: affected,
			},
			"lastId": &ast.IntegerLiteral{
				Token: lexer.Token{Type: lexer.INT, Literal: strconv.FormatInt(lastId, 10)},
				Value: lastId,
			},
		},
		Env: env,
	}

	return assignQueryResult(node.Names, resultDict, env, node.IsLet)
}

// extractSQLAndParams extracts SQL string and parameters from a query object
func extractSQLAndParams(queryObj Object, env *Environment) (string, []any, *Error) {
	// If it's a string, use it directly with no params
	if str, ok := queryObj.(*String); ok {
		return str.Value, nil, nil
	}

	// If it's a dictionary (from <SQL> tag), extract sql and params
	if dict, ok := queryObj.(*Dictionary); ok {
		// Get SQL content
		sqlExpr, hasSql := dict.Pairs["sql"]
		if !hasSql {
			return "", nil, newSQLError("SQL-0002", nil)
		}
		sqlObj := Eval(sqlExpr, env)
		if isError(sqlObj) {
			return "", nil, sqlObj.(*Error)
		}
		sqlStr, ok := sqlObj.(*String)
		if !ok {
			return "", nil, newSQLError("SQL-0003", map[string]any{"Got": string(sqlObj.Type())})
		}

		// Get params if present
		var params []any
		if paramsExpr, hasParams := dict.Pairs["params"]; hasParams {
			paramsObj := Eval(paramsExpr, env)
			if isError(paramsObj) {
				return "", nil, paramsObj.(*Error)
			}
			if paramsDict, ok := paramsObj.(*Dictionary); ok {
				params = dictToNamedParams(paramsDict, env)
			}
		}

		return sqlStr.Value, params, nil
	}

	return "", nil, newSQLError("SQL-0004", map[string]any{"Got": string(queryObj.Type())})
}

// dictToNamedParams converts a dictionary to a slice of named parameters
func dictToNamedParams(dict *Dictionary, env *Environment) []any {
	params := make([]any, 0, len(dict.Pairs))

	// Sort keys for consistent order
	keys := make([]string, 0, len(dict.Pairs))
	for key := range dict.Pairs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		expr := dict.Pairs[key]
		val := Eval(expr, env)
		params = append(params, objectToGoValue(val))
	}

	return params
}

// rowToDict is in eval_conversions.go

// assignQueryResult assigns query result to variables
func assignQueryResult(names []*ast.Identifier, result Object, env *Environment, isLet bool) Object {
	if len(names) == 0 {
		// No assignment target - shouldn't happen in practice
		return NULL
	}

	if len(names) == 1 {
		name := names[0].Value
		if name != "_" {
			if isLet {
				env.SetLet(name, result)
			} else {
				env.Update(name, result)
			}
		}
		// Query statements return NULL (like let statements)
		return NULL
	}

	// Multiple names - destructure array or dict
	return evalDestructuringAssignment(names, result, env, isLet, false)
}

// evalDatabaseQueryOne evaluates database query for single row (infix expression version)
func evalDatabaseQueryOne(connObj Object, queryObj Object, env *Environment) Object {
	conn, ok := connObj.(*DBConnection)
	if !ok {
		perr := perrors.New("DB-0012", map[string]any{"Operator": "query operator <=?=>", "Got": string(connObj.Type())})
		return &Error{Class: ErrorClass(perr.Class), Code: perr.Code, Message: perr.Message, Hints: perr.Hints, Data: perr.Data}
	}

	// Extract SQL and params from the query object
	sql, params, err := extractSQLAndParams(queryObj, env)
	if err != nil {
		return err
	}

	// Execute the query
	rows, queryErr := conn.DB.Query(sql, params...)
	if queryErr != nil {
		conn.LastError = queryErr.Error()
		return newDatabaseError("DB-0002", queryErr)
	}
	defer rows.Close()

	// Get column names
	columns, colErr := rows.Columns()
	if colErr != nil {
		conn.LastError = colErr.Error()
		return newDatabaseError("DB-0008", colErr)
	}

	// Check if there's a row
	if !rows.Next() {
		// No rows - return null
		return NULL
	}

	// Scan the row into a map
	values := make([]any, len(columns))
	valuePtrs := make([]any, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	if scanErr := rows.Scan(valuePtrs...); scanErr != nil {
		conn.LastError = scanErr.Error()
		return newDatabaseError("DB-0004", scanErr)
	}

	// Convert to dictionary
	return rowToDict(columns, values, env)
}

// evalDatabaseQueryMany evaluates database query for multiple rows (infix expression version)
func evalDatabaseQueryMany(connObj Object, queryObj Object, env *Environment) Object {
	conn, ok := connObj.(*DBConnection)
	if !ok {
		perr := perrors.New("DB-0012", map[string]any{"Operator": "query operator <=??=>", "Got": string(connObj.Type())})
		return &Error{Class: ErrorClass(perr.Class), Code: perr.Code, Message: perr.Message, Hints: perr.Hints, Data: perr.Data}
	}

	// Extract SQL and params
	sql, params, err := extractSQLAndParams(queryObj, env)
	if err != nil {
		return err
	}

	// Execute the query
	rows, queryErr := conn.DB.Query(sql, params...)
	if queryErr != nil {
		conn.LastError = queryErr.Error()
		return newDatabaseError("DB-0002", queryErr)
	}
	defer rows.Close()

	// Get column names
	columns, colErr := rows.Columns()
	if colErr != nil {
		conn.LastError = colErr.Error()
		return newDatabaseError("DB-0008", colErr)
	}

	// Scan all rows
	var results []Object
	for rows.Next() {
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if scanErr := rows.Scan(valuePtrs...); scanErr != nil {
			conn.LastError = scanErr.Error()
			return newDatabaseError("DB-0004", scanErr)
		}

		resultDict := rowToDict(columns, values, env)
		results = append(results, resultDict)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		conn.LastError = rowsErr.Error()
		return newDatabaseError("DB-0002", rowsErr)
	}

	return &Array{Elements: results}
}

// evalDatabaseExecute evaluates database execute statement (infix expression version)
func evalDatabaseExecute(connObj Object, queryObj Object, env *Environment) Object {
	conn, ok := connObj.(*DBConnection)
	if !ok {
		perr := perrors.New("DB-0012", map[string]any{"Operator": "execute operator <=!=>", "Got": string(connObj.Type())})
		return &Error{Class: ErrorClass(perr.Class), Code: perr.Code, Message: perr.Message, Hints: perr.Hints, Data: perr.Data}
	}

	// Extract SQL and params
	sql, params, err := extractSQLAndParams(queryObj, env)
	if err != nil {
		return err
	}

	// Execute the statement
	result, execErr := conn.DB.Exec(sql, params...)
	if execErr != nil {
		conn.LastError = execErr.Error()
		return newDatabaseError("DB-0011", execErr)
	}

	// Get affected rows and last insert ID
	affected, _ := result.RowsAffected()
	lastId, _ := result.LastInsertId()

	// Return result as dictionary
	return &Dictionary{
		Pairs: map[string]ast.Expression{
			"affected": &ast.IntegerLiteral{
				Token: lexer.Token{Type: lexer.INT, Literal: strconv.FormatInt(affected, 10)},
				Value: affected,
			},
			"lastId": &ast.IntegerLiteral{
				Token: lexer.Token{Type: lexer.INT, Literal: strconv.FormatInt(lastId, 10)},
				Value: lastId,
			},
		},
		Env: env,
	}
}

// writeFileContent writes content to a file based on its format
