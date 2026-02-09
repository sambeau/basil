package evaluator

import (
	"bytes"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strings"
	"time"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/lexer"
)

// Network I/O operations: evalSFTPConnectionMethod, evalSFTPFileHandleMethod, evalFetchStatement, evalFetchExpression + helpers
// Extracted from evaluator.go - Phase 5 Extraction 29

func evalSFTPConnectionMethod(conn *SFTPConnection, method string, args []Object, env *Environment) Object {
	switch method {
	case "close":
		if len(args) != 0 {
			return newArityError("close", len(args), 0)
		}

		// Note: We don't remove from cache on explicit close, as the cache
		// handles TTL and cleanup automatically. Manual close just marks
		// the connection as disconnected.

		// Close SFTP and SSH clients
		if conn.Client != nil {
			conn.Client.Close()
		}
		if conn.SSHClient != nil {
			conn.SSHClient.Close()
		}
		conn.Connected = false
		return NULL

	default:
		return newUndefinedMethodError(method, "SFTP connection")
	}
}

// evalSFTPFileHandleMethod handles method calls on SFTP file handles
func evalSFTPFileHandleMethod(handle *SFTPFileHandle, method string, args []Object, env *Environment) Object {
	switch method {
	case "mkdir":
		// Create directory
		var recursive bool
		if len(args) > 0 {
			if optDict, ok := args[0].(*Dictionary); ok {
				if parentsExpr, ok := optDict.Pairs["parents"]; ok {
					if parentsVal := Eval(parentsExpr, optDict.Env); parentsVal != nil {
						if boolVal, ok := parentsVal.(*Boolean); ok {
							recursive = boolVal.Value
						}
					}
				}
			}
		}

		var err error
		if recursive {
			err = handle.Connection.Client.MkdirAll(handle.Path)
		} else {
			err = handle.Connection.Client.Mkdir(handle.Path)
		}

		if err != nil {
			return newIOError("IO-0009", handle.Path, err)
		}
		return NULL

	case "rmdir":
		// Remove directory
		var recursive bool
		if len(args) > 0 {
			if optDict, ok := args[0].(*Dictionary); ok {
				if recursiveExpr, ok := optDict.Pairs["recursive"]; ok {
					if recursiveVal := Eval(recursiveExpr, optDict.Env); recursiveVal != nil {
						if boolVal, ok := recursiveVal.(*Boolean); ok {
							recursive = boolVal.Value
						}
					}
				}
			}
		}

		// Note: SFTP RemoveDirectory only removes empty directories.
		// The recursive option is parsed but not yet implemented.
		// TODO: implement recursive directory removal if needed.
		_ = recursive
		err := handle.Connection.Client.RemoveDirectory(handle.Path)

		if err != nil {
			return newIOError("IO-0010", handle.Path, err)
		}
		return NULL

	case "remove":
		// Remove file
		if len(args) != 0 {
			return newArityError("remove", len(args), 0)
		}

		if err := handle.Connection.Client.Remove(handle.Path); err != nil {
			return newIOError("IO-0005", handle.Path, err)
		}
		return NULL

	default:
		return newUndefinedMethodError(method, "SFTP file handle")
	}
}
func evalFetchStatement(node *ast.FetchStatement, env *Environment) Object {
	// Check if we're using dict pattern destructuring with error capture pattern
	useErrorCapture := node.DictPattern != nil && isErrorCapturePattern(node.DictPattern)

	// Evaluate the source expression (should be a request handle, URL, or SFTP file handle)
	source := Eval(node.Source, env)
	if isError(source) {
		if useErrorCapture {
			return evalDictDestructuringAssignment(node.DictPattern,
				makeFetchResponseDict(NULL, source.(*Error).Message, 0, nil, env), env, node.IsLet, false)
		}
		return source
	}

	// Check if it's an SFTP file handle
	if sftpHandle, ok := source.(*SFTPFileHandle); ok {
		content, err := evalSFTPRead(sftpHandle, env)
		if err != nil {
			if useErrorCapture {
				return evalDictDestructuringAssignment(node.DictPattern,
					makeSFTPResponseDict(NULL, err.(*Error).Message, env), env, node.IsLet, false)
			}
			return err
		}

		// Assign to the target variable(s)
		if node.DictPattern != nil {
			if useErrorCapture {
				// Wrap successful result in {data: ..., error: null} format
				return evalDictDestructuringAssignment(node.DictPattern,
					makeSFTPResponseDict(content, "", env), env, node.IsLet, false)
			}
			// Regular dict destructuring
			return evalDictDestructuringAssignment(node.DictPattern, content, env, node.IsLet, false)
		}

		// Simple assignment
		if node.ArrayPattern != nil {
			return evalArrayPatternAssignment(node.ArrayPattern, content, env, node.IsLet, false)
		}

		return content
	}

	// The source should be a request dictionary (from jsonFile(@url), etc.) or a URL dictionary
	sourceDict, ok := source.(*Dictionary)
	if !ok {
		if useErrorCapture {
			return evalDictDestructuringAssignment(node.DictPattern,
				makeFetchResponseDict(NULL, fmt.Sprintf("fetch operator <=/= requires a request or URL handle, got %s", strings.ToLower(string(source.Type()))), 0, nil, env), env, node.IsLet, false)
		}
		return newFileOpError("FILEOP-0007", map[string]any{"Operator": "fetch operator <=/>=", "Expected": "a request or URL handle", "Got": string(source.Type())})
	}

	var reqDict *Dictionary

	if isRequestDict(sourceDict) {
		reqDict = sourceDict
	} else if isUrlDict(sourceDict) {
		// Wrap URL in a request dictionary with default format (text)
		reqDict = urlToRequestDict(sourceDict, "text", nil, env)
	} else {
		if useErrorCapture {
			return evalDictDestructuringAssignment(node.DictPattern,
				makeFetchResponseDict(NULL, "fetch operator <=/= requires a request or URL handle, got dictionary", 0, nil, env), env, node.IsLet, false)
		}
		return newFileOpError("FILEOP-0007", map[string]any{"Operator": "fetch operator <=/>=", "Expected": "a request or URL handle", "Got": "dictionary"})
	}

	// Fetch URL content with full response info
	info := fetchUrlContentFull(reqDict, env)

	// Handle errors with legacy error capture pattern
	if info.Error != "" {
		if useErrorCapture {
			return evalDictDestructuringAssignment(node.DictPattern,
				makeFetchResponseDict(NULL, info.Error, info.StatusCode, info.Headers, env), env, node.IsLet, false)
		}
		return newHTTPErrorMessage("HTTP-0006", info.Error)
	}

	// Create response typed dictionary
	responseDict := makeResponseTypedDict(
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

	// Assign to the target variable(s)
	if node.DictPattern != nil {
		if useErrorCapture {
			// Wrap successful result in {data: ..., error: null, status: ..., headers: ...} format
			return evalDictDestructuringAssignment(node.DictPattern,
				makeFetchResponseDict(info.Content, "", info.StatusCode, info.Headers, env), env, node.IsLet, false)
		}
		// Normal dict destructuring - extract keys directly from __data
		return evalDictDestructuringAssignment(node.DictPattern, info.Content, env, node.IsLet, false)
	}

	if node.ArrayPattern != nil {
		return evalArrayPatternAssignment(node.ArrayPattern, responseDict, env, node.IsLet, false)
	}

	// Single assignment
	if node.Name != nil && node.Name.Value != "_" {
		if node.IsLet {
			env.SetLet(node.Name.Value, responseDict)
		} else {
			env.Update(node.Name.Value, responseDict)
		}
	}

	// Fetch statements return NULL (like let statements)
	return NULL
}

// isRequestDict and isResponseDict moved to eval_helpers.go

// setRequestMethod clones a request dict with a new HTTP method
func setRequestMethod(dict *Dictionary, method string, env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)

	// Copy all existing pairs
	maps.Copy(pairs, dict.Pairs)

	// Set the method
	pairs["method"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: method},
		Value: method,
	}

	return &Dictionary{Pairs: pairs, Env: env}
}

// parseURLToDict parses a URL string into a URL dictionary, returning nil on error

// makeResponseTypedDict creates a response typed dictionary with __type, __format, __data, __response
// This is the new response structure that auto-unwraps for iteration/indexing
func makeResponseTypedDict(data Object, format string, statusCode int64, statusText string, ok bool, urlStr string, headers *Dictionary, errorMsg string, env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)

	// Set __type
	pairs["__type"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: "response"},
		Value: "response",
	}

	// Set __format
	pairs["__format"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: format},
		Value: format,
	}

	// Set __data (the actual fetched data, or null on error)
	if data != nil {
		pairs["__data"] = &ast.ObjectLiteralExpression{Obj: data}
	} else {
		pairs["__data"] = &ast.ObjectLiteralExpression{Obj: NULL}
	}

	// Build __response dictionary
	responsePairs := make(map[string]ast.Expression)

	responsePairs["status"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: fmt.Sprintf("%d", statusCode)},
		Value: statusCode,
	}

	responsePairs["statusText"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: statusText},
		Value: statusText,
	}

	responsePairs["ok"] = &ast.ObjectLiteralExpression{Obj: &Boolean{Value: ok}}

	// URL as a URL dictionary
	if urlStr != "" {
		urlDict := parseURLToDict(urlStr, env)
		if urlDict != nil {
			responsePairs["url"] = &ast.ObjectLiteralExpression{Obj: urlDict}
		} else {
			responsePairs["url"] = &ast.StringLiteral{
				Token: lexer.Token{Type: lexer.STRING, Literal: urlStr},
				Value: urlStr,
			}
		}
	} else {
		responsePairs["url"] = &ast.ObjectLiteralExpression{Obj: NULL}
	}

	// Headers
	if headers != nil {
		responsePairs["headers"] = &ast.ObjectLiteralExpression{Obj: headers}
	} else {
		responsePairs["headers"] = &ast.DictionaryLiteral{
			Token: lexer.Token{Type: lexer.LBRACE, Literal: "{"},
			Pairs: make(map[string]ast.Expression),
		}
	}

	// Error
	if errorMsg == "" {
		responsePairs["error"] = &ast.ObjectLiteralExpression{Obj: NULL}
	} else {
		responsePairs["error"] = &ast.StringLiteral{
			Token: lexer.Token{Type: lexer.STRING, Literal: errorMsg},
			Value: errorMsg,
		}
	}

	pairs["__response"] = &ast.DictionaryLiteral{
		Token: lexer.Token{Type: lexer.LBRACE, Literal: "{"},
		Pairs: responsePairs,
	}

	return &Dictionary{Pairs: pairs, Env: env}
}

// makeFetchResponseDict creates a {data: ..., error: ..., status: ..., headers: ...} dictionary
// This is the legacy format for error capture pattern
func makeFetchResponseDict(data Object, errorMsg string, status int64, headers *Dictionary, env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)

	// Set data field
	pairs["data"] = &ast.ObjectLiteralExpression{Obj: data}

	// Set error field
	if errorMsg == "" {
		pairs["error"] = &ast.ObjectLiteralExpression{Obj: NULL}
	} else {
		pairs["error"] = &ast.ObjectLiteralExpression{Obj: &String{Value: errorMsg}}
	}

	// Set status field
	pairs["status"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: fmt.Sprintf("%d", status)},
		Value: status,
	}

	// Set headers field
	if headers != nil {
		pairs["headers"] = &ast.ObjectLiteralExpression{Obj: headers}
	} else {
		pairs["headers"] = &ast.DictionaryLiteral{
			Token: lexer.Token{Type: lexer.LBRACE, Literal: "{"},
			Pairs: make(map[string]ast.Expression),
		}
	}

	return &Dictionary{Pairs: pairs, Env: env}
}

// getRequestUrlString extracts the URL string from a request dictionary
func getRequestUrlString(dict *Dictionary, env *Environment) string {
	var result strings.Builder

	// Get scheme
	schemeExpr, ok := dict.Pairs["_url_scheme"]
	if !ok {
		return ""
	}
	schemeObj := Eval(schemeExpr, env)
	schemeStr, ok := schemeObj.(*String)
	if !ok {
		return ""
	}
	result.WriteString(schemeStr.Value)
	result.WriteString("://")

	// Get host
	hostExpr, ok := dict.Pairs["_url_host"]
	if !ok {
		return ""
	}
	hostObj := Eval(hostExpr, env)
	hostStr, ok := hostObj.(*String)
	if !ok {
		return ""
	}
	result.WriteString(hostStr.Value)

	// Get port (if non-zero)
	if portExpr, ok := dict.Pairs["_url_port"]; ok {
		portObj := Eval(portExpr, env)
		if portInt, ok := portObj.(*Integer); ok && portInt.Value != 0 {
			result.WriteString(fmt.Sprintf(":%d", portInt.Value))
		}
	}

	// Get path
	if pathExpr, ok := dict.Pairs["_url_path"]; ok {
		pathObj := Eval(pathExpr, env)
		if pathArr, ok := pathObj.(*Array); ok {
			for _, elem := range pathArr.Elements {
				result.WriteString("/")
				if str, ok := elem.(*String); ok {
					result.WriteString(str.Value)
				}
			}
		}
	}

	// Get query
	if queryExpr, ok := dict.Pairs["_url_query"]; ok {
		queryObj := Eval(queryExpr, env)
		if queryDict, ok := queryObj.(*Dictionary); ok && len(queryDict.Pairs) > 0 {
			result.WriteString("?")
			first := true
			for key, valExpr := range queryDict.Pairs {
				if !first {
					result.WriteString("&")
				}
				first = false
				valObj := Eval(valExpr, env)
				result.WriteString(key)
				result.WriteString("=")
				switch v := valObj.(type) {
				case *String:
					result.WriteString(v.Value)
				case *Integer:
					result.WriteString(fmt.Sprintf("%d", v.Value))
				default:
					result.WriteString(valObj.Inspect())
				}
			}
		}
	}

	return result.String()
}

// HTTPResponseInfo holds all information about an HTTP response
type HTTPResponseInfo struct {
	Content    Object
	StatusCode int64
	StatusText string
	OK         bool
	FinalURL   string
	Headers    *Dictionary
	Format     string
	Error      string
}

// fetchUrlContentFull fetches content from a URL and returns full response info
func fetchUrlContentFull(reqDict *Dictionary, env *Environment) *HTTPResponseInfo {
	info := &HTTPResponseInfo{}

	// Get the URL string
	urlStr := getRequestUrlString(reqDict, env)
	if urlStr == "" {
		info.Error = "request handle has no valid URL"
		return info
	}
	info.FinalURL = urlStr

	// Get method
	method := "GET"
	if methodExpr, ok := reqDict.Pairs["method"]; ok {
		methodObj := Eval(methodExpr, env)
		if methodStr, ok := methodObj.(*String); ok {
			method = strings.ToUpper(methodStr.Value)
		}
	}

	// Get format
	format := "text"
	if formatExpr, ok := reqDict.Pairs["format"]; ok {
		formatObj := Eval(formatExpr, env)
		if formatStr, ok := formatObj.(*String); ok {
			format = formatStr.Value
		}
	}
	info.Format = format

	// Get timeout (default 30 seconds)
	timeout := 30 * time.Second
	if timeoutExpr, ok := reqDict.Pairs["timeout"]; ok {
		timeoutObj := Eval(timeoutExpr, env)
		if timeoutInt, ok := timeoutObj.(*Integer); ok {
			timeout = time.Duration(timeoutInt.Value) * time.Millisecond
		}
	}

	// Prepare request body
	var bodyReader io.Reader
	if bodyExpr, ok := reqDict.Pairs["body"]; ok {
		bodyObj := Eval(bodyExpr, env)
		if bodyObj != nil && bodyObj != NULL {
			switch v := bodyObj.(type) {
			case *String:
				bodyReader = strings.NewReader(v.Value)
			case *Dictionary, *Array:
				jsonBytes, err := encodeJSON(bodyObj)
				if err != nil {
					info.Error = fmt.Sprintf("failed to encode request body: %s", err.Error())
					return info
				}
				bodyReader = bytes.NewReader(jsonBytes)
			default:
				bodyReader = strings.NewReader(bodyObj.Inspect())
			}
		}
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: timeout,
	}

	// Create request
	req, err := http.NewRequest(method, urlStr, bodyReader)
	if err != nil {
		info.Error = fmt.Sprintf("failed to create request: %s", err.Error())
		return info
	}

	// Set headers
	if headersExpr, ok := reqDict.Pairs["headers"]; ok {
		headersObj := Eval(headersExpr, env)
		if headersDict, ok := headersObj.(*Dictionary); ok {
			for key, valExpr := range headersDict.Pairs {
				valObj := Eval(valExpr, env)
				if valStr, ok := valObj.(*String); ok {
					req.Header.Set(key, valStr.Value)
				}
			}
		}
	}

	// Set default Content-Type for POST/PUT with body
	if bodyReader != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		info.Error = fmt.Sprintf("fetch failed: %s", err.Error())
		return info
	}
	defer resp.Body.Close()

	// Capture response info
	info.StatusCode = int64(resp.StatusCode)
	info.StatusText = resp.Status // e.g., "200 OK" or "404 Not Found"
	info.OK = resp.StatusCode >= 200 && resp.StatusCode < 300
	info.FinalURL = resp.Request.URL.String() // Final URL after redirects

	// Read response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		info.Error = fmt.Sprintf("failed to read response: %s", err.Error())
		return info
	}

	// Convert response headers to dictionary
	respHeaders := &Dictionary{Pairs: make(map[string]ast.Expression), Env: env}
	for key, values := range resp.Header {
		if len(values) > 0 {
			respHeaders.Pairs[strings.ToLower(key)] = &ast.StringLiteral{
				Token: lexer.Token{Type: lexer.STRING, Literal: values[0]},
				Value: values[0],
			}
		}
	}
	info.Headers = respHeaders

	// Decode based on format
	var content Object
	var parseErr *Error

	switch format {
	case "text":
		content = &String{Value: string(data)}

	case "json":
		content, parseErr = parseJSON(string(data))
		if parseErr != nil {
			info.Error = parseErr.Message
			return info
		}

	case "yaml":
		content, parseErr = parseYAML(string(data))
		if parseErr != nil {
			info.Error = parseErr.Message
			return info
		}

	case "lines":
		lines := strings.Split(string(data), "\n")
		elements := make([]Object, len(lines))
		for i, line := range lines {
			elements[i] = &String{Value: line}
		}
		content = &Array{Elements: elements}

	case "bytes":
		elements := make([]Object, len(data))
		for i, b := range data {
			elements[i] = &Integer{Value: int64(b)}
		}
		content = &Array{Elements: elements}

	default:
		content = &String{Value: string(data)}
	}

	info.Content = content
	return info
}

// isErrorCapturePattern checks if a dict destructuring pattern contains "data" or "error" keys
// which indicates the user wants to use the error capture pattern
// evalFetchExpression evaluates a bare <=/= expression (used in assignment capture like `let x = <=/= source`)
func evalFetchExpression(node *ast.FetchExpression, env *Environment) Object {
	source := Eval(node.Source, env)
	if isError(source) {
		return source
	}

	// SFTP file handle — return content directly
	if sftpHandle, ok := source.(*SFTPFileHandle); ok {
		content, err := evalSFTPRead(sftpHandle, env)
		if err != nil {
			return err
		}
		return content
	}

	// Must be a request or URL dictionary
	sourceDict, ok := source.(*Dictionary)
	if !ok {
		return newFileOpError("FILEOP-0007", map[string]any{"Operator": "fetch operator <=/>=", "Expected": "a request or URL handle", "Got": string(source.Type())})
	}

	var reqDict *Dictionary
	if isRequestDict(sourceDict) {
		reqDict = sourceDict
	} else if isUrlDict(sourceDict) {
		reqDict = urlToRequestDict(sourceDict, "text", nil, env)
	} else {
		return newFileOpError("FILEOP-0007", map[string]any{"Operator": "fetch operator <=/>=", "Expected": "a request or URL handle", "Got": "dictionary"})
	}

	info := fetchUrlContentFull(reqDict, env)

	if info.Error != "" {
		return newHTTPErrorMessage("HTTP-0006", info.Error)
	}

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

// isResponseTypedDict checks whether a value is a typed response dictionary (has __type = "response")
func isResponseTypedDict(obj Object) bool {
	dict, ok := obj.(*Dictionary)
	if !ok {
		return false
	}
	typeExpr, exists := dict.Pairs["__type"]
	if !exists {
		return false
	}
	if sl, ok := typeExpr.(*ast.StringLiteral); ok {
		return sl.Value == "response"
	}
	if ole, ok := typeExpr.(*ast.ObjectLiteralExpression); ok {
		if s, ok := ole.Obj.(*String); ok {
			return s.Value == "response"
		}
	}
	return false
}

// responseTypedDictToLegacy converts a typed response dict (__type, __data, __response)
// to the legacy {data, error, status, headers} shape expected by error-capture destructuring.
func responseTypedDictToLegacy(dict *Dictionary, env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)

	// Extract __data
	if dataExpr, ok := dict.Pairs["__data"]; ok {
		pairs["data"] = dataExpr
	} else {
		pairs["data"] = &ast.ObjectLiteralExpression{Obj: NULL}
	}

	// Extract fields from __response sub-dict
	var status int64
	var errorMsg string
	var headers *Dictionary

	if responseExpr, ok := dict.Pairs["__response"]; ok {
		responseObj := Eval(responseExpr, env)
		if responseDict, ok := responseObj.(*Dictionary); ok {
			// status
			if statusExpr, ok := responseDict.Pairs["status"]; ok {
				statusObj := Eval(statusExpr, env)
				if statusInt, ok := statusObj.(*Integer); ok {
					status = statusInt.Value
				}
			}
			// error
			if errorExpr, ok := responseDict.Pairs["error"]; ok {
				errorObj := Eval(errorExpr, env)
				if errorStr, ok := errorObj.(*String); ok {
					errorMsg = errorStr.Value
				}
			}
			// headers
			if headersExpr, ok := responseDict.Pairs["headers"]; ok {
				headersObj := Eval(headersExpr, env)
				if h, ok := headersObj.(*Dictionary); ok {
					headers = h
				}
			}
		}
	}

	return makeFetchResponseDict(Eval(pairs["data"], env), errorMsg, status, headers, env)
}

func isErrorCapturePattern(pattern *ast.DictDestructuringPattern) bool {
	for _, key := range pattern.Keys {
		if key.Key != nil {
			keyName := key.Key.Value
			if keyName == "data" || keyName == "error" {
				return true
			}
		}
	}
	return false
}

// makeDataErrorDict creates a {data: ..., error: ...} dictionary for error capture pattern
func makeDataErrorDict(data Object, errorMsg string, env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)

	// Set data field
	pairs["data"] = &ast.ObjectLiteralExpression{Obj: data}

	// Set error field
	if errorMsg == "" {
		pairs["error"] = &ast.ObjectLiteralExpression{Obj: NULL}
	} else {
		pairs["error"] = &ast.ObjectLiteralExpression{Obj: &String{Value: errorMsg}}
	}

	return &Dictionary{Pairs: pairs}
}

// evalRemoteWriteStatement handles =/=> (remote write) and =/=>> (remote append) operators
func evalRemoteWriteStatement(node *ast.RemoteWriteStatement, env *Environment) Object {
	value := Eval(node.Value, env)
	if isError(value) {
		return value
	}

	target := Eval(node.Target, env)
	if isError(target) {
		return target
	}

	op := "=/=>"
	if node.Append {
		op = "=/=>>"
	}

	// SFTP file handle
	if sftpHandle, ok := target.(*SFTPFileHandle); ok {
		err := evalSFTPWrite(sftpHandle, value, node.Append, env)
		if err != nil {
			return err
		}
		return NULL
	}

	// HTTP request dictionary
	if reqDict, ok := target.(*Dictionary); ok && isRequestDict(reqDict) {
		if node.Append {
			return newErrorWithClass(ClassOperator, "operator =/=>> (remote append) is not supported for HTTP — HTTP has no append semantic")
		}
		return evalHTTPWrite(reqDict, value, env)
	}

	// Reject local file handles with helpful message
	if fileDict, ok := target.(*Dictionary); ok && isFileDict(fileDict) {
		if node.Append {
			return newErrorWithClass(ClassOperator, "operator =/=>> is for remote appends; use ==>> for local file appends")
		}
		return newErrorWithClass(ClassOperator, "operator =/=> is for network writes; use ==> for local file writes")
	}

	return newErrorWithClass(ClassType, "operator %s requires an HTTP request handle or SFTP file handle, got %s", op, strings.ToLower(string(target.Type())))
}
