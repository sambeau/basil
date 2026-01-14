package evaluator

import (
	"fmt"
	"strings"
)

// Method dispatch operations: dispatchMethodCall, evalDBConnectionMethod
// Extracted from evaluator.go - Phase 5 Extraction 30

func evalDBConnectionMethod(conn *DBConnection, method string, args []Object, env *Environment) Object {
	switch method {
	case "begin":
		if len(args) != 0 {
			return newArityError("begin", len(args), 0)
		}
		if conn.InTransaction {
			return newDatabaseStateError("DB-0007")
		}
		conn.InTransaction = true
		return &Boolean{Value: true}

	case "commit":
		if len(args) != 0 {
			return newArityError("commit", len(args), 0)
		}
		if !conn.InTransaction {
			return newDatabaseStateError("DB-0006")
		}
		// For now, just mark transaction as complete
		// Real transaction support will be added with actual query execution
		conn.InTransaction = false
		return &Boolean{Value: true}

	case "rollback":
		if len(args) != 0 {
			return newArityError("rollback", len(args), 0)
		}
		if !conn.InTransaction {
			return newDatabaseStateError("DB-0006")
		}
		conn.InTransaction = false
		return &Boolean{Value: true}

	case "close":
		if len(args) != 0 {
			return newArityError("close", len(args), 0)
		}
		// Managed connections cannot be closed by Parsley scripts
		if conn.Managed {
			return newDatabaseStateError("DB-0009")
		}
		// Note: We don't remove from cache on explicit close, as the cache
		// handles TTL and cleanup automatically. Manual close just closes
		// the database connection.

		if err := conn.DB.Close(); err != nil {
			conn.LastError = err.Error()
			return newDatabaseError("DB-0010", err)
		}
		return NULL

	case "ping":
		if len(args) != 0 {
			return newArityError("ping", len(args), 0)
		}
		if err := conn.DB.Ping(); err != nil {
			conn.LastError = err.Error()
			return &Boolean{Value: false}
		}
		return &Boolean{Value: true}

	case "createTable":
		// db.createTable(schema) or db.createTable(schema, "table_name")
		// Creates a table from a schema if it doesn't already exist
		if len(args) < 1 || len(args) > 2 {
			return newArityError("createTable", len(args), 1)
		}

		schema, ok := args[0].(*DSLSchema)
		if !ok {
			return newTypeError("TYPE-0001", "db.createTable", "schema", args[0].Type())
		}

		// Get table name: either from second arg or use schema name (lowercase)
		tableName := strings.ToLower(schema.Name) + "s" // default: pluralize
		if len(args) == 2 {
			tableNameStr, ok := args[1].(*String)
			if !ok {
				return newTypeError("TYPE-0001", "db.createTable", "string (table name)", args[1].Type())
			}
			tableName = tableNameStr.Value
		}

		// Build CREATE TABLE IF NOT EXISTS SQL
		sql := buildCreateTableSQL(schema, tableName, conn.Driver)

		// Execute the SQL
		_, err := conn.DB.Exec(sql)
		if err != nil {
			conn.LastError = err.Error()
			return newDatabaseError("DB-0005", err)
		}

		return &Boolean{Value: true}

	case "lastInsertId":
		// Get the last inserted row ID (SQLite only)
		if len(args) != 0 {
			return newArityError("lastInsertId", len(args), 0)
		}
		if conn.Driver != "sqlite" {
			return newDatabaseErrorWithDriver("DB-0001", conn.Driver, fmt.Errorf("lastInsertId only supported for SQLite"))
		}

		var id int64
		err := conn.DB.QueryRow("SELECT last_insert_rowid()").Scan(&id)
		if err != nil {
			conn.LastError = err.Error()
			return newDatabaseError("DB-0005", err)
		}
		return &Integer{Value: id}

	case "bind":
		// db.bind(schema, "table_name") or db.bind(schema, "table_name", {soft_delete: "deleted_at"})
		if len(args) < 2 || len(args) > 3 {
			return newArityError("bind", len(args), 2)
		}

		// Get table name (second argument)
		tableName, ok := args[1].(*String)
		if !ok {
			return newTypeError("TYPE-0001", "db.bind", "string (table name)", args[1].Type())
		}
		name := strings.TrimSpace(tableName.Value)
		if name == "" || !identifierRegex.MatchString(name) {
			return newValidationError("VAL-0003", map[string]any{"Pattern": "identifier", "GoError": "invalid table name"})
		}

		// Parse options if provided
		var softDeleteColumn string
		if len(args) == 3 {
			optsDict, ok := args[2].(*Dictionary)
			if !ok {
				return newTypeError("TYPE-0001", "db.bind", "dictionary (options)", args[2].Type())
			}
			if sdExpr, ok := optsDict.Pairs["soft_delete"]; ok {
				sdVal := Eval(sdExpr, optsDict.Env)
				if sdStr, ok := sdVal.(*String); ok {
					softDeleteColumn = sdStr.Value
				} else {
					return newTypeError("TYPE-0001", "db.bind soft_delete option", "string", sdVal.Type())
				}
			}
		}

		// Handle both DSLSchema and Dictionary schemas
		switch schema := args[0].(type) {
		case *DSLSchema:
			return &TableBinding{
				DB:               conn,
				DSLSchema:        schema,
				TableName:        name,
				SoftDeleteColumn: softDeleteColumn,
			}
		case *Dictionary:
			return &TableBinding{
				DB:               conn,
				Schema:           schema,
				TableName:        name,
				SoftDeleteColumn: softDeleteColumn,
			}
		default:
			return newTypeError("TYPE-0001", "db.bind", "schema or dictionary", args[0].Type())
		}

	default:
		return newUndefinedMethodError(method, "database connection")
	}
}

// Network I/O operations (evalSFTPConnectionMethod, evalSFTPFileHandleMethod, evalFetchStatement) are in eval_network_io.go

func dispatchMethodCall(left Object, method string, args []Object, env *Environment) Object {
	// Universal .type() method - works on all objects
	if method == "type" {
		if len(args) != 0 {
			return newArityError("type", len(args), 0)
		}
		return &String{Value: getObjectTypeString(left)}
	}

	switch receiver := left.(type) {
	case *DevModule:
		return evalDevModuleMethod(receiver, method, args, env)
	case *TableModule:
		return evalTableModuleMethod(receiver, method, args, env)
	case *Table:
		return EvalTableMethod(receiver, method, args, env)
	case *TableBinding:
		return evalTableBindingMethod(receiver, method, args, env)
	case *MdDoc:
		return evalMdDocMethod(receiver, method, args, env)
	case *DBConnection:
		return evalDBConnectionMethod(receiver, method, args, env)
	case *SFTPConnection:
		return evalSFTPConnectionMethod(receiver, method, args, env)
	case *SFTPFileHandle:
		return evalSFTPFileHandleMethod(receiver, method, args, env)
	case *SessionModule:
		return evalSessionMethod(receiver, method, args, env)
	case *String:
		return evalStringMethod(receiver, method, args, env)
	case *Array:
		return evalArrayMethod(receiver, method, args, env)
	case *Integer:
		return evalIntegerMethod(receiver, method, args)
	case *Float:
		return evalFloatMethod(receiver, method, args)
	case *Boolean:
		return evalBooleanMethod(receiver, method, args)
	case *Null:
		return evalNullMethod(method, args)
	case *Money:
		return evalMoneyMethod(receiver, method, args)
	case *Dictionary:
		// Check for special dictionary types first
		if isDatetimeDict(receiver) {
			result := evalDatetimeMethod(receiver, method, args, env)
			if result != nil && !isError(result) {
				return result
			}
			// Fall through to check dictionary methods if datetime method failed
			if result != nil && isError(result) {
				// Check if it's "unknown method" error - try dictionary method
				if errObj, ok := result.(*Error); ok && errObj.Code == "UNDEF-0002" {
					// Try dictionary methods
					dictResult := evalDictionaryMethod(receiver, method, args, env)
					if dictResult != nil {
						return dictResult
					}
				}
				return result
			}
		}
		if isDurationDict(receiver) {
			result := evalDurationMethod(receiver, method, args, env)
			if result != nil {
				return result
			}
		}
		if isPathDict(receiver) {
			result := evalPathMethod(receiver, method, args, env)
			if result != nil && !isError(result) {
				return result
			}
			// If unknown method, fall through to dictionary methods
			if result != nil && isError(result) {
				if errObj, ok := result.(*Error); ok && errObj.Code == "UNDEF-0002" {
					dictResult := evalDictionaryMethod(receiver, method, args, env)
					if dictResult != nil {
						return dictResult
					}
				}
				return result
			}
		}
		if isUrlDict(receiver) {
			result := evalUrlMethod(receiver, method, args, env)
			if result != nil && !isError(result) {
				return result
			}
			// If unknown method, fall through to dictionary methods
			if result != nil && isError(result) {
				if errObj, ok := result.(*Error); ok && errObj.Code == "UNDEF-0002" {
					dictResult := evalDictionaryMethod(receiver, method, args, env)
					if dictResult != nil {
						return dictResult
					}
				}
				return result
			}
		}
		if isRegexDict(receiver) {
			result := evalRegexMethod(receiver, method, args, env)
			if result != nil && !isError(result) {
				return result
			}
			// If unknown method, fall through to dictionary methods
			if result != nil && isError(result) {
				if errObj, ok := result.(*Error); ok && errObj.Code == "UNDEF-0002" {
					dictResult := evalDictionaryMethod(receiver, method, args, env)
					if dictResult != nil {
						return dictResult
					}
				}
				return result
			}
		}
		if isFileDict(receiver) {
			result := evalFileMethod(receiver, method, args, env)
			if result != nil && !isError(result) {
				return result
			}
			// If unknown method, fall through to dictionary methods
			if result != nil && isError(result) {
				if errObj, ok := result.(*Error); ok && errObj.Code == "UNDEF-0002" {
					dictResult := evalDictionaryMethod(receiver, method, args, env)
					if dictResult != nil {
						return dictResult
					}
				}
				return result
			}
		}
		if isDirDict(receiver) {
			result := evalDirMethod(receiver, method, args, env)
			if result != nil && !isError(result) {
				return result
			}
			// If unknown method, fall through to dictionary methods
			if result != nil && isError(result) {
				if errObj, ok := result.(*Error); ok && errObj.Code == "UNDEF-0002" {
					dictResult := evalDictionaryMethod(receiver, method, args, env)
					if dictResult != nil {
						return dictResult
					}
				}
				return result
			}
		}
		if isRequestDict(receiver) {
			result := evalRequestMethod(receiver, method, args, env)
			if result != nil && !isError(result) {
				return result
			}
			// If unknown method, fall through to dictionary methods
			if result != nil && isError(result) {
				if errObj, ok := result.(*Error); ok && errObj.Code == "UNDEF-0002" {
					dictResult := evalDictionaryMethod(receiver, method, args, env)
					if dictResult != nil {
						return dictResult
					}
				}
				return result
			}
		}
		if isResponseDict(receiver) {
			result := evalResponseMethod(receiver, method, args, env)
			if result != nil && !isError(result) {
				return result
			}
			// If unknown method, fall through to dictionary methods
			if result != nil && isError(result) {
				if errObj, ok := result.(*Error); ok && errObj.Code == "UNDEF-0002" {
					dictResult := evalDictionaryMethod(receiver, method, args, env)
					if dictResult != nil {
						return dictResult
					}
				}
				return result
			}
		}
		// Regular dictionary methods (keys, values, has)
		result := evalDictionaryMethod(receiver, method, args, env)
		if result != nil {
			return result
		}
		// Check if the dictionary has a user-defined function at this key
		if fnExpr, ok := receiver.Pairs[method]; ok {
			fnObj := Eval(fnExpr, receiver.Env)
			if fn, ok := fnObj.(*Function); ok {
				// Call the function with 'this' bound to the dictionary
				return applyMethodWithThis(fn, args, receiver, env)
			}
			// Check if it's a StdlibBuiltin (e.g., from @SEARCH, @DB, etc.)
			if builtin, ok := fnObj.(*StdlibBuiltin); ok {
				return builtin.Fn(args, env)
			}
			// If it's not a function, return error
			if !isError(fnObj) {
				return newStructuredError("TYPE-0021", map[string]any{"Name": method})
			}
		}
		// Method not found - return error with available methods
		return unknownMethodError(method, "dictionary", dictionaryMethods)
	}
	// No specific handler for this type
	return nil
}
