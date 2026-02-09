package evaluator

import (
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
	perrors "github.com/sambeau/basil/pkg/parsley/errors"
	"github.com/sambeau/basil/pkg/parsley/lexer"
)

// File I/O operations: evalReadStatement, evalReadExpression, evalWriteStatement,
// writeFileContent, evalFileRemove

func evalReadStatement(node *ast.ReadStatement, env *Environment) Object {
	// Check if we're using dict pattern destructuring with error capture pattern
	// Only use {data, error} wrapping if the pattern contains "data" or "error" keys
	useErrorCapture := node.DictPattern != nil && isErrorCapturePattern(node.DictPattern)

	// Evaluate the source expression (should be a file or dir handle)
	source := Eval(node.Source, env)
	if isError(source) {
		if useErrorCapture {
			// Wrap the error in {data: null, error: "message"} format
			return evalDictDestructuringAssignment(node.DictPattern,
				makeDataErrorDict(NULL, source.(*Error).Message, env), env, node.IsLet, false)
		}
		return source
	}

	// The source should be a file or directory dictionary
	sourceDict, ok := source.(*Dictionary)
	if !ok {
		errMsg := fmt.Sprintf("read operator <== requires a file or directory handle, got %s", strings.ToLower(string(source.Type())))
		if useErrorCapture {
			return evalDictDestructuringAssignment(node.DictPattern,
				makeDataErrorDict(NULL, errMsg, env), env, node.IsLet, false)
		}
		return newFileOpError("FILEOP-0007", map[string]any{"Operator": "read operator <==", "Expected": "a file or directory handle", "Got": string(source.Type())})
	}

	var content Object
	var readErr *Error

	if isDirDict(sourceDict) {
		// Read directory contents
		pathStr := getFilePathString(sourceDict, env)
		if pathStr == "" {
			errMsg := "directory handle has no valid path"
			if useErrorCapture {
				return evalDictDestructuringAssignment(node.DictPattern,
					makeDataErrorDict(NULL, errMsg, env), env, node.IsLet, false)
			}
			return newFileOpError("FILEOP-0008", nil)
		}
		content = readDirContents(pathStr, env)
		if isError(content) {
			if useErrorCapture {
				return evalDictDestructuringAssignment(node.DictPattern,
					makeDataErrorDict(NULL, content.(*Error).Message, env), env, node.IsLet, false)
			}
			return content
		}
	} else if isFileDict(sourceDict) {
		// Read file content based on format
		content, readErr = readFileContent(sourceDict, env)
		if readErr != nil {
			if useErrorCapture {
				return evalDictDestructuringAssignment(node.DictPattern,
					makeDataErrorDict(NULL, readErr.Message, env), env, node.IsLet, false)
			}
			return readErr
		}
	} else {
		errMsg := "read operator <== requires a file or directory handle, got dictionary"
		if useErrorCapture {
			return evalDictDestructuringAssignment(node.DictPattern,
				makeDataErrorDict(NULL, errMsg, env), env, node.IsLet, false)
		}
		return newFileOpError("FILEOP-0007", map[string]any{"Operator": "read operator <==", "Expected": "a file or directory handle", "Got": "dictionary"})
	}

	// Assign to the target variable(s)
	if node.DictPattern != nil {
		if useErrorCapture {
			// Wrap successful result in {data: ..., error: null} format
			return evalDictDestructuringAssignment(node.DictPattern,
				makeDataErrorDict(content, "", env), env, node.IsLet, false)
		}
		// Normal dict destructuring - extract keys directly from content
		return evalDictDestructuringAssignment(node.DictPattern, content, env, node.IsLet, false)
	}

	if node.ArrayPattern != nil {
		return evalArrayPatternAssignment(node.ArrayPattern, content, env, node.IsLet, false)
	}

	// Single assignment
	if node.Name != nil && node.Name.Value != "_" {
		if node.IsLet {
			env.SetLet(node.Name.Value, content)
		} else {
			env.Update(node.Name.Value, content)
		}
	}

	// Read statements return NULL (like let statements)
	return NULL
}

// evalReadExpression evaluates a bare <== expression and returns the read content
func evalReadExpression(node *ast.ReadExpression, env *Environment) Object {
	// Evaluate the source expression (should be a file or dir handle)
	source := Eval(node.Source, env)
	if isError(source) {
		return source
	}

	// The source should be a file or directory dictionary
	sourceDict, ok := source.(*Dictionary)
	if !ok {
		return newFileOpError("FILEOP-0007", map[string]any{"Operator": "read expression <==", "Expected": "a file or directory handle", "Got": string(source.Type())})
	}

	if isDirDict(sourceDict) {
		// Read directory contents
		pathStr := getFilePathString(sourceDict, env)
		if pathStr == "" {
			return newFileOpError("FILEOP-0008", nil)
		}
		content := readDirContents(pathStr, env)
		if isError(content) {
			return content
		}
		return content
	} else if isFileDict(sourceDict) {
		// Read file content based on format
		content, readErr := readFileContent(sourceDict, env)
		if readErr != nil {
			return readErr
		}
		return content
	}

	return newFileOpError("FILEOP-0007", map[string]any{"Operator": "read expression <==", "Expected": "a file or directory handle", "Got": "dictionary"})
}

// evalFetchStatement evaluates the <=/= operator to fetch URL content
func evalWriteStatement(node *ast.WriteStatement, env *Environment) Object {
	// Evaluate the value to write
	value := Eval(node.Value, env)
	if isError(value) {
		return value
	}

	// Evaluate the target expression (should be a file handle, SFTP file handle, or HTTP request)
	target := Eval(node.Target, env)
	if isError(target) {
		return target
	}

	// Reject HTTP request dictionaries — must use =/=>
	if reqDict, ok := target.(*Dictionary); ok && isRequestDict(reqDict) {
		if node.Append {
			return newErrorWithClass(ClassOperator, "operator ==>> is for local file appends; use =/=>> for remote appends")
		}
		return newErrorWithClass(ClassOperator, "operator ==> is for local file writes; use =/=> for network writes")
	}

	// Reject SFTP file handles — must use =/=> or =/=>>
	if _, ok := target.(*SFTPFileHandle); ok {
		if node.Append {
			return newErrorWithClass(ClassOperator, "operator ==>> is for local file appends; use =/=>> for remote appends")
		}
		return newErrorWithClass(ClassOperator, "operator ==> is for local file writes; use =/=> for network writes")
	}

	// The target should be a file dictionary
	fileDict, ok := target.(*Dictionary)
	if !ok || !isFileDict(fileDict) {
		return newFileOpError("FILEOP-0007", map[string]any{"Operator": "write operator ==>", "Expected": "a file handle", "Got": string(target.Type())})
	}

	// Write the file content based on format
	err := writeFileContent(fileDict, value, node.Append, env)
	if err != nil {
		return err
	}

	return NULL
}

// evalHTTPWrite performs an HTTP write operation (POST/PUT/PATCH)
func evalHTTPWrite(reqDict *Dictionary, value Object, env *Environment) Object {
	// Set the body to the value being written
	pairs := make(map[string]ast.Expression)
	maps.Copy(pairs, reqDict.Pairs)

	// Encode the value as the request body
	pairs["body"] = &ast.ObjectLiteralExpression{Obj: value}

	// Default method to POST if not already set to PUT or PATCH
	method := "POST"
	if methodExpr, ok := reqDict.Pairs["method"]; ok {
		methodObj := Eval(methodExpr, env)
		if methodStr, ok := methodObj.(*String); ok {
			upperMethod := strings.ToUpper(methodStr.Value)
			// Only keep PUT, PATCH - otherwise default to POST
			if upperMethod == "PUT" || upperMethod == "PATCH" {
				method = upperMethod
			}
		}
	}
	pairs["method"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: method},
		Value: method,
	}

	newReqDict := &Dictionary{Pairs: pairs, Env: env}

	// Fetch URL content with full response info
	info := fetchUrlContentFull(newReqDict, env)

	// Handle errors
	if info.Error != "" {
		return newHTTPErrorMessage("HTTP-0006", info.Error)
	}

	// Create and return response typed dictionary
	return makeResponseTypedDict(
		info.Content,
		info.Format,
		info.StatusCode,
		info.StatusText,
		info.OK,
		info.FinalURL,
		info.Headers,
		"",
		env,
	)
}

// makeSFTPResponseDict creates a response dictionary for SFTP operations with error capture
func makeSFTPResponseDict(data Object, errMsg string, env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)

	if errMsg != "" {
		pairs["data"] = &ast.ObjectLiteralExpression{Obj: NULL}
		pairs["error"] = &ast.ObjectLiteralExpression{Obj: &String{Value: errMsg}}
	} else {
		// Store data directly as an expression
		pairs["data"] = &ast.ObjectLiteralExpression{Obj: data}
		pairs["error"] = &ast.ObjectLiteralExpression{Obj: NULL}
	}

	return &Dictionary{Pairs: pairs, Env: env}
}

// evalSFTPRead reads content from an SFTP file handle
func evalSFTPRead(handle *SFTPFileHandle, env *Environment) (Object, Object) {
	if !handle.Connection.Connected {
		return nil, newStateError("STATE-0002")
	}

	// Handle directory listing
	if handle.Format == "dir" {
		entries, err := handle.Connection.Client.ReadDir(handle.Path)
		if err != nil {
			return nil, newSFTPError("SFTP-0006", err)
		}

		files := make([]Object, 0, len(entries))
		for _, entry := range entries {
			fileInfo := make(map[string]ast.Expression)
			fileInfo["name"] = &ast.StringLiteral{Value: entry.Name()}
			fileInfo["path"] = &ast.StringLiteral{Value: filepath.Join(handle.Path, entry.Name())}
			fileInfo["size"] = &ast.IntegerLiteral{Value: entry.Size()}
			fileInfo["isDir"] = &ast.ObjectLiteralExpression{Obj: &Boolean{Value: entry.IsDir()}}
			fileInfo["isFile"] = &ast.ObjectLiteralExpression{Obj: &Boolean{Value: !entry.IsDir()}}
			fileInfo["mode"] = &ast.StringLiteral{Value: entry.Mode().String()}
			fileInfo["modified"] = &ast.ObjectLiteralExpression{Obj: timeToDict(entry.ModTime(), env)}

			files = append(files, &Dictionary{Pairs: fileInfo, Env: env})
		}

		return &Array{Elements: files}, nil
	}

	// Open remote file
	file, err := handle.Connection.Client.Open(handle.Path)
	if err != nil {
		return nil, newSFTPError("SFTP-0007", err)
	}
	defer file.Close()

	// Read content
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, newSFTPError("SFTP-0007", err)
	}

	// Parse based on format
	format := handle.Format
	if format == "" {
		format = "text"
	}

	switch format {
	case "json":
		return parseJSON(string(data))
	case "text":
		return &String{Value: string(data)}, nil
	case "lines":
		lines := strings.Split(string(data), "\n")
		// Remove trailing empty line if present
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		elements := make([]Object, len(lines))
		for i, line := range lines {
			elements[i] = &String{Value: line}
		}
		return &Array{Elements: elements}, nil
	case "csv":
		return parseCSV(data, true) // Assume CSV has headers by default
	case "bytes":
		elements := make([]Object, len(data))
		for i, b := range data {
			elements[i] = &Integer{Value: int64(b)}
		}
		return &Array{Elements: elements}, nil
	case "file":
		// Auto-detect from extension
		ext := filepath.Ext(handle.Path)
		switch ext {
		case ".json":
			return parseJSON(string(data))
		case ".csv":
			return parseCSV(data, true)
		default:
			return &String{Value: string(data)}, nil
		}
	default:
		perr := perrors.New("SFTP-0004", map[string]any{"Format": format})
		return nil, &Error{Class: ErrorClass(perr.Class), Code: perr.Code, Message: perr.Message, Hints: perr.Hints, Data: perr.Data}
	}
}

// evalSFTPWrite writes content to an SFTP file handle
func evalSFTPWrite(handle *SFTPFileHandle, value Object, append bool, env *Environment) Object {
	if !handle.Connection.Connected {
		return newStateError("STATE-0002")
	}

	// Determine open flags
	flags := os.O_WRONLY | os.O_CREATE
	if append {
		flags |= os.O_APPEND // SSH_FXF_APPEND (0x00000004)
	} else {
		flags |= os.O_TRUNC
	}

	// Encode based on format
	format := handle.Format
	if format == "" {
		format = "text"
	}

	var content string
	switch format {
	case "json":
		jsonBytes, err := encodeJSON(value)
		if err != nil {
			handle.Connection.Client.Close()
			return makeSFTPResponseDict(NULL, fmt.Sprintf("JSON encoding failed: %s", err.Error()), env)
		}
		content = string(jsonBytes)
	case "text":
		if str, ok := value.(*String); ok {
			content = str.Value
		} else {
			perr := perrors.New("SFTP-0001", map[string]any{"Format": "text", "Expected": "string value", "Got": string(value.Type())})
			return &Error{Class: ErrorClass(perr.Class), Code: perr.Code, Message: perr.Message, Hints: perr.Hints, Data: perr.Data}
		}
	case "lines":
		if arr, ok := value.(*Array); ok {
			lines := make([]string, len(arr.Elements))
			for i, elem := range arr.Elements {
				if str, ok := elem.(*String); ok {
					lines[i] = str.Value
				} else {
					perr := perrors.New("SFTP-0002", map[string]any{"Format": "lines", "Expected": "strings", "Index": i, "Got": string(elem.Type())})
					return &Error{Class: ErrorClass(perr.Class), Code: perr.Code, Message: perr.Message, Hints: perr.Hints, Data: perr.Data}
				}
			}
			content = strings.Join(lines, "\n") + "\n"
		} else {
			perr := perrors.New("SFTP-0001", map[string]any{"Format": "lines", "Expected": "array", "Got": string(value.Type())})
			return &Error{Class: ErrorClass(perr.Class), Code: perr.Code, Message: perr.Message, Hints: perr.Hints, Data: perr.Data}
		}
	case "csv":
		perr := perrors.New("SFTP-0003", nil)
		return &Error{Class: ErrorClass(perr.Class), Code: perr.Code, Message: perr.Message, Hints: perr.Hints, Data: perr.Data}
	case "bytes":
		if arr, ok := value.(*Array); ok {
			bytes := make([]byte, len(arr.Elements))
			for i, elem := range arr.Elements {
				if intVal, ok := elem.(*Integer); ok {
					bytes[i] = byte(intVal.Value)
				} else {
					perr := perrors.New("SFTP-0002", map[string]any{"Format": "bytes", "Expected": "integers", "Index": i, "Got": string(elem.Type())})
					return &Error{Class: ErrorClass(perr.Class), Code: perr.Code, Message: perr.Message, Hints: perr.Hints, Data: perr.Data}
				}
			}
			content = string(bytes)
		} else {
			perr := perrors.New("SFTP-0001", map[string]any{"Format": "bytes", "Expected": "array", "Got": string(value.Type())})
			return &Error{Class: ErrorClass(perr.Class), Code: perr.Code, Message: perr.Message, Hints: perr.Hints, Data: perr.Data}
		}
	default:
		perr := perrors.New("SFTP-0004", map[string]any{"Format": format})
		return &Error{Class: ErrorClass(perr.Class), Code: perr.Code, Message: perr.Message, Hints: perr.Hints, Data: perr.Data}
	}

	// Open remote file via SFTP with appropriate flags
	file, err := handle.Connection.Client.OpenFile(handle.Path, flags)
	if err != nil {
		return newSFTPError("SFTP-0005", err)
	}
	defer file.Close()

	// Write content
	_, err = file.Write([]byte(content))
	if err != nil {
		return newSFTPError("SFTP-0005", err)
	}

	return NULL
}

// evalQueryOneStatement evaluates the <=?=> operator to query a single row

// Database query operations (evalQueryOneStatement, evalQueryManyStatement,
// evalExecuteStatement, extractSQLAndParams, dictToNamedParams,
// assignQueryResult, evalDatabaseQueryOne, evalDatabaseQueryMany,
// evalDatabaseExecute) are in eval_database.go
func writeFileContent(fileDict *Dictionary, value Object, appendMode bool, env *Environment) *Error {
	// Check if this is a stdio stream
	var isStdio bool
	var stdioStream string

	if stdioExpr, ok := fileDict.Pairs["__stdio"]; ok {
		stdioObj := Eval(stdioExpr, env)
		if stdioStr, ok := stdioObj.(*String); ok {
			switch stdioStr.Value {
			case "stdin":
				return newStdioError("STDIO-0001", nil)
			case "stdout", "stderr":
				isStdio = true
				stdioStream = stdioStr.Value
			case "stdio":
				// @- for writes means stdout
				isStdio = true
				stdioStream = "stdout"
			default:
				return newStdioError("STDIO-0002", map[string]any{"Name": stdioStr.Value})
			}
		}
	}

	var pathStr string
	if !isStdio {
		// Get the path from the file dictionary
		pathStr = getFilePathString(fileDict, env)
		if pathStr == "" {
			return newFileOpError("FILEOP-0002", nil)
		}

		// Resolve the path relative to the current file (or root path for ~/ paths)
		absPath, pathErr := resolveModulePath(pathStr, env.Filename, env.RootPath)
		if pathErr != nil {
			return newIOError("IO-0007", pathStr, pathErr)
		}
		pathStr = absPath

		// Security check
		if err := env.checkPathAccess(pathStr, "write"); err != nil {
			return newSecurityError("write", err)
		}
	}

	// Get the format
	formatExpr, hasFormat := fileDict.Pairs["format"]
	if !hasFormat {
		return newFileOpError("FILEOP-0003", nil)
	}
	formatObj := Eval(formatExpr, env)
	if isError(formatObj) {
		return formatObj.(*Error)
	}
	formatStr, ok := formatObj.(*String)
	if !ok {
		return newFileOpError("FILEOP-0004", map[string]any{"Got": string(formatObj.Type())})
	}

	// Encode the value based on format
	var data []byte
	var encodeErr error

	switch formatStr.Value {
	case "text":
		data, encodeErr = encodeText(value)

	case "bytes":
		data, encodeErr = encodeBytes(value)

	case "lines":
		data, encodeErr = encodeLines(value, appendMode)

	case "json":
		data, encodeErr = encodeJSON(value)

	case "csv", "csv-noheader":
		data, encodeErr = encodeCSV(value, formatStr.Value == "csv")

	case "svg":
		data, encodeErr = encodeSVG(value)

	case "yaml":
		data, encodeErr = encodeYAML(value)

	default:
		return newFileOpError("FILEOP-0005", map[string]any{"Operation": "writing", "Format": formatStr.Value})
	}

	if encodeErr != nil {
		return newFileOpError("FILEOP-0006", map[string]any{"GoError": encodeErr.Error()})
	}

	// Write to stdout/stderr or file
	var writeErr error
	if isStdio {
		// Write to stdout or stderr
		var w *os.File
		if stdioStream == "stdout" {
			w = os.Stdout
		} else {
			w = os.Stderr
		}
		_, writeErr = w.Write(data)
	} else if appendMode {
		f, err := os.OpenFile(pathStr, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return newIOError("IO-0004", pathStr, err)
		}
		defer f.Close()
		_, writeErr = f.Write(data)
	} else {
		writeErr = os.WriteFile(pathStr, data, 0644)
	}

	if writeErr != nil {
		if isStdio {
			return newIOError("IO-0004", stdioStream, writeErr)
		}
		return newIOError("IO-0004", pathStr, writeErr)
	}

	return nil
}

// File encoding functions (encodeText, encodeBytes, encodeLines, encodeJSON,
// objectToGo, encodeSVG, encodeYAML, encodeCSV) are in eval_encoders.go

// evalFileRemove removes/deletes a file from the filesystem
func evalFileRemove(fileDict *Dictionary, env *Environment) Object {
	// Get the path from the file dictionary
	pathStr := getFilePathString(fileDict, env)
	if pathStr == "" {
		return newFileOpError("FILEOP-0002", nil)
	}

	// Resolve the path relative to the current file (or root path for ~/ paths)
	absPath, pathErr := resolveModulePath(pathStr, env.Filename, env.RootPath)
	if pathErr != nil {
		return newIOError("IO-0007", pathStr, pathErr)
	}

	// Security check (treat as write operation)
	if err := env.checkPathAccess(absPath, "write"); err != nil {
		return newSecurityError("write", err)
	}

	// Delete the file
	err := os.Remove(absPath)
	if err != nil {
		return newIOError("IO-0005", absPath, err)
	}

	// Return a new null value instead of the global NULL
	return &Null{}
}
