// Package evaluator provides DBConnection and SFTP method implementations via declarative registry.
// This file migrates database connection and SFTP methods from switch-based dispatch (FEAT-137 Tier 3).
package evaluator

// DBConnectionMethodRegistry defines all methods available on database connection objects.
var DBConnectionMethodRegistry MethodRegistry

// SFTPConnectionMethodRegistry defines all methods available on SFTP connection objects.
var SFTPConnectionMethodRegistry MethodRegistry

// SFTPFileHandleMethodRegistry defines all methods available on SFTP file handle objects.
var SFTPFileHandleMethodRegistry MethodRegistry

func init() {
	DBConnectionMethodRegistry = MethodRegistry{
		"begin": {
			Fn:          dbMethodBegin,
			Arity:       "0",
			Description: "Begin a transaction",
		},
		"commit": {
			Fn:          dbMethodCommit,
			Arity:       "0",
			Description: "Commit the current transaction",
		},
		"rollback": {
			Fn:          dbMethodRollback,
			Arity:       "0",
			Description: "Rollback the current transaction",
		},
		"close": {
			Fn:          dbMethodClose,
			Arity:       "0",
			Description: "Close the database connection",
		},
		"ping": {
			Fn:          dbMethodPing,
			Arity:       "0",
			Description: "Test the database connection",
		},
		"createTable": {
			Fn:          dbMethodCreateTable,
			Arity:       "1-2",
			Description: "Create a table from a schema (schema, tableName?)",
		},
		"lastInsertId": {
			Fn:          dbMethodLastInsertId,
			Arity:       "0",
			Description: "Get the last inserted row ID (SQLite only)",
		},
		"bind": {
			Fn:          dbMethodBind,
			Arity:       "2-3",
			Description: "Bind a schema to a table (schema, tableName, options?)",
		},
	}
	RegisterMethodRegistry("dbconnection", DBConnectionMethodRegistry)

	SFTPConnectionMethodRegistry = MethodRegistry{
		"close": {
			Fn:          sftpConnectionMethodClose,
			Arity:       "0",
			Description: "Close the SFTP connection",
		},
	}
	RegisterMethodRegistry("sftpconnection", SFTPConnectionMethodRegistry)

	SFTPFileHandleMethodRegistry = MethodRegistry{
		"mkdir": {
			Fn:          sftpFileMethodMkdir,
			Arity:       "0-1",
			Description: "Create a directory (options?)",
		},
		"rmdir": {
			Fn:          sftpFileMethodRmdir,
			Arity:       "0-1",
			Description: "Remove a directory (options?)",
		},
		"remove": {
			Fn:          sftpFileMethodRemove,
			Arity:       "0",
			Description: "Remove a file",
		},
	}
	RegisterMethodRegistry("sftpfile", SFTPFileHandleMethodRegistry)
}

// ============================================================================
// DBConnection method registry wrappers
// ============================================================================

func dbMethodBegin(receiver Object, args []Object, env *Environment) Object {
	return evalDBConnectionMethod(receiver.(*DBConnection), "begin", args, env)
}

func dbMethodCommit(receiver Object, args []Object, env *Environment) Object {
	return evalDBConnectionMethod(receiver.(*DBConnection), "commit", args, env)
}

func dbMethodRollback(receiver Object, args []Object, env *Environment) Object {
	return evalDBConnectionMethod(receiver.(*DBConnection), "rollback", args, env)
}

func dbMethodClose(receiver Object, args []Object, env *Environment) Object {
	return evalDBConnectionMethod(receiver.(*DBConnection), "close", args, env)
}

func dbMethodPing(receiver Object, args []Object, env *Environment) Object {
	return evalDBConnectionMethod(receiver.(*DBConnection), "ping", args, env)
}

func dbMethodCreateTable(receiver Object, args []Object, env *Environment) Object {
	return evalDBConnectionMethod(receiver.(*DBConnection), "createTable", args, env)
}

func dbMethodLastInsertId(receiver Object, args []Object, env *Environment) Object {
	return evalDBConnectionMethod(receiver.(*DBConnection), "lastInsertId", args, env)
}

func dbMethodBind(receiver Object, args []Object, env *Environment) Object {
	return evalDBConnectionMethod(receiver.(*DBConnection), "bind", args, env)
}

// ============================================================================
// SFTPConnection method registry wrappers
// ============================================================================

func sftpConnectionMethodClose(receiver Object, args []Object, env *Environment) Object {
	return evalSFTPConnectionMethod(receiver.(*SFTPConnection), "close", args, env)
}

// ============================================================================
// SFTPFileHandle method registry wrappers
// ============================================================================

func sftpFileMethodMkdir(receiver Object, args []Object, env *Environment) Object {
	return evalSFTPFileHandleMethod(receiver.(*SFTPFileHandle), "mkdir", args, env)
}

func sftpFileMethodRmdir(receiver Object, args []Object, env *Environment) Object {
	return evalSFTPFileHandleMethod(receiver.(*SFTPFileHandle), "rmdir", args, env)
}

func sftpFileMethodRemove(receiver Object, args []Object, env *Environment) Object {
	return evalSFTPFileHandleMethod(receiver.(*SFTPFileHandle), "remove", args, env)
}
