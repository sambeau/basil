package evaluator

import (
"fmt"
"os"
"strings"

"github.com/sambeau/basil/pkg/parsley/ast"
perrors "github.com/sambeau/basil/pkg/parsley/errors"
"github.com/sambeau/basil/pkg/parsley/lexer"
"github.com/sambeau/basil/pkg/parsley/parser"
)

// Expression evaluation: function application, parameter handling, assignments, destructuring
// Extracted from evaluator.go - Phase 5 Extraction 31

func applyFunction(fn Object, args []Object) Object {
	switch fn := fn.(type) {
	case *Function:
		extendedEnv := extendFunctionEnv(fn, args)
		evaluated := Eval(fn.Body, extendedEnv)
		return unwrapReturnValue(evaluated)
	case *Builtin:
		if fn.FnWithEnv != nil {
			return fn.FnWithEnv(nil, args...)
		}
		return fn.Fn(args...)
	case *StdlibBuiltin:
		// StdlibBuiltin needs an environment but applyFunction doesn't have one
		// This shouldn't happen as StdlibBuiltin should be called via applyFunctionWithEnv
		perr := perrors.New("INTERNAL-0001", map[string]any{"Context": "stdlib function"})
		return &Error{Class: ErrorClass(perr.Class), Code: perr.Code, Message: perr.Message, Hints: perr.Hints, Data: perr.Data}
	default:
		if fn == NULL || fn == nil {
			return newCallError("CALL-0001", nil)
		}
		return newCallError("CALL-0002", map[string]any{"Type": string(fn.Type())})
	}
}

// applyMethodWithThis calls a function with 'this' bound to a dictionary.
// This enables object-oriented style method calls like user.greet() where
// the function can access the dictionary via 'this'.
// The calling environment (env) is used to copy runtime context (BasilCtx, etc.)
// to ensure request-scoped values like @basil/http.query work correctly.
func applyMethodWithThis(fn *Function, args []Object, thisObj *Dictionary, env *Environment) Object {
	extendedEnv := extendFunctionEnv(fn, args)
	extendedEnv.Set("this", thisObj)
	// Copy runtime context from calling environment (like ApplyFunctionWithEnv does)
	// This ensures features like <CSS/>, <Javascript/>, and request-scoped values work
	if env != nil {
		extendedEnv.AssetBundle = env.AssetBundle
		extendedEnv.AssetRegistry = env.AssetRegistry
		extendedEnv.FragmentCache = env.FragmentCache
		extendedEnv.BasilCtx = env.BasilCtx
		extendedEnv.DevLog = env.DevLog
		extendedEnv.HandlerPath = env.HandlerPath
		extendedEnv.DevMode = env.DevMode
		extendedEnv.Security = env.Security
		extendedEnv.Logger = env.Logger
	}
	evaluated := Eval(fn.Body, extendedEnv)
	return unwrapReturnValue(evaluated)
}

// ApplyFunctionWithEnv applies a function with the given arguments in the context of an environment.
// It handles parameter binding including destructuring patterns, and copies runtime context
// from the calling environment to ensure features like <CSS/>, <Javascript/> work in imported components.
func ApplyFunctionWithEnv(fn Object, args []Object, env *Environment) Object {
	switch fn := fn.(type) {
	case *Function:
		extendedEnv := extendFunctionEnv(fn, args)
		// Copy runtime context from calling environment to function environment
		// This ensures <Css/>, <Script/>, and other runtime features work in imported components
		if env != nil {
			extendedEnv.AssetBundle = env.AssetBundle
			extendedEnv.AssetRegistry = env.AssetRegistry
			extendedEnv.FragmentCache = env.FragmentCache
			extendedEnv.BasilCtx = env.BasilCtx
			extendedEnv.DevLog = env.DevLog
			extendedEnv.HandlerPath = env.HandlerPath
			extendedEnv.DevMode = env.DevMode
			extendedEnv.Security = env.Security
			extendedEnv.Logger = env.Logger
		}
		evaluated := Eval(fn.Body, extendedEnv)
		return unwrapReturnValue(evaluated)
	case *Builtin:
		var result Object
		if fn.FnWithEnv != nil {
			result = fn.FnWithEnv(env, args...)
		} else {
			result = fn.Fn(args...)
		}
		// Add position info to builtin errors for better debugging
		if isError(result) {
			return enrichErrorWithPos(result, env.LastToken)
		}
		return result
	case *StdlibBuiltin:
		result := fn.Fn(args, env)
		// Add position info to stdlib errors for better debugging
		if isError(result) {
			return enrichErrorWithPos(result, env.LastToken)
		}
		return result
	case *AuthWrappedFunction:
		// Delegate to the inner function
		return ApplyFunctionWithEnv(fn.Inner, args, env)
	case *TableModule:
		// TableModule is callable: table(arr) creates a Table from an array
		result := TableConstructor(args, env)
		if isError(result) {
			return enrichErrorWithPos(result, env.LastToken)
		}
		return result
	case *MdDocModule:
		// MdDocModule is callable: mdDoc(text) or mdDoc(dict) creates an MdDoc
		result := evalMdDocModuleCall(args, env)
		if isError(result) {
			return enrichErrorWithPos(result, env.LastToken)
		}
		return result
	case *DevModule:
		// DevModule is not directly callable, only used as a namespace
		return enrichErrorWithPos(newCallError("CALL-0003", nil), env.LastToken)
	case *SFTPConnection:
		// SFTP connection is callable: conn(@/path) returns SFTP file handle
		if len(args) != 1 {
			return newArityError("SFTP", len(args), 1)
		}

		// Extract path from argument
		var pathStr string
		switch arg := args[0].(type) {
		case *Dictionary:
			if !isPathDict(arg) {
				return newTypeError("TYPE-0012", "SFTP connection", "a path", DICTIONARY_OBJ)
			}
			pathStr = pathDictToString(arg)
		case *String:
			pathStr = arg.Value
		default:
			return newTypeError("TYPE-0012", "SFTP connection", "a path", arg.Type())
		}

		// Return SFTP file handle
		return &SFTPFileHandle{
			Connection: fn,
			Path:       pathStr,
			Format:     "", // Will default to "text"
			Options:    nil,
		}
	default:
		if fn == NULL || fn == nil {
			return enrichErrorWithPos(newCallError("CALL-0001", nil), env.LastToken)
		}
		return enrichErrorWithPos(newCallError("CALL-0002", map[string]any{"Type": string(fn.Type())}), env.LastToken)
	}
}

// CallWithEnv invokes a callable object within the provided environment.
// This is used by external packages (e.g., server layer) to execute exported handlers.
func CallWithEnv(fn Object, args []Object, env *Environment) Object {
	return ApplyFunctionWithEnv(fn, args, env)
}

// evalImportExpression implements the new import @path syntax.
// Unlike the old import("path") function call, this:
// - Takes path directly from AST (no string arg)
// - Auto-binds to environment when used as a statement
// - Supports "as Alias" syntax
func evalImportExpression(node *ast.ImportExpression, env *Environment) Object {
	// Evaluate the path expression to get the path string
	pathObj := Eval(node.Path, env)
	if isError(pathObj) {
		return pathObj
	}

	// Convert to string - handles StdlibPathLiteral, PathLiteral, PathTemplateLiteral, etc.
	var pathStr string
	switch p := pathObj.(type) {
	case *String:
		pathStr = p.Value
	case *Dictionary:
		// Handle path literal dictionary (@./path/to/file)
		if typeExpr, ok := p.Pairs["__type"]; ok {
			typeVal := Eval(typeExpr, p.Env)
			if typeStr, ok := typeVal.(*String); ok && typeStr.Value == "path" {
				pathStr = pathDictToString(p)
			} else {
				return newTypeError("TYPE-0012", "import", "a path", DICTIONARY_OBJ)
			}
		} else {
			return newTypeError("TYPE-0012", "import", "a path", DICTIONARY_OBJ)
		}
	default:
		return newTypeError("TYPE-0012", "import", "a path", pathObj.Type())
	}

	// Load the module using the shared import logic
	module := importModule(pathStr, env)
	if isError(module) {
		return module
	}

	// NOTE: Auto-binding to BindName is NOT done here.
	// When import is used as a standalone statement, evalStatement handles the auto-bind.
	// When import is used in a let/assignment (e.g., let {x} = import @std/foo),
	// only the destructured names should be bound, not the path-derived name.

	// Always return the module (for use in let statements and destructuring)
	return module
}

// importModule is the shared logic for loading a module by path string.
// Used by both evalImport (old syntax) and evalImportExpression (new syntax).
func importModule(pathStr string, env *Environment) Object {
	// Check for stdlib root import (just "std" without module name)
	if pathStr == "std" {
		return loadStdlibRoot()
	}

	// Check for basil root import (just "basil" without module name)
	if pathStr == "basil" {
		return loadBasilRoot()
	}

	// Check for standard library imports (std/modulename)
	if strings.HasPrefix(pathStr, "std/") {
		moduleName := strings.TrimPrefix(pathStr, "std/")
		return loadStdlibModule(moduleName, env)
	}

	// Check for basil namespace imports (basil/modulename)
	if strings.HasPrefix(pathStr, "basil/") {
		moduleName := strings.TrimPrefix(pathStr, "basil/")
		return loadBasilModule(moduleName, env)
	}

	// Resolve path relative to current file (or root path for ~/ paths)
	absPath, err := resolveModulePath(pathStr, env.Filename, env.RootPath)
	if err != nil {
		return newImportError("IMPORT-0004", map[string]any{"GoError": err.Error()})
	}

	// Security check
	if err := env.checkPathAccess(absPath, "execute"); err != nil {
		return newSecurityError("execute", err)
	}

	// Check if module is currently being loaded in THIS request (circular dependency)
	// Use the root environment's importStack to track across nested imports
	rootEnv := env
	for rootEnv.outer != nil {
		rootEnv = rootEnv.outer
	}
	if rootEnv.importStack[absPath] {
		return newImportError("IMPORT-0002", map[string]any{"Path": absPath})
	}

	// Check cache first (with lock for thread safety)
	moduleCache.mu.RLock()
	if cached, ok := moduleCache.modules[absPath]; ok {
		moduleCache.mu.RUnlock()
		return cached
	}
	moduleCache.mu.RUnlock()

	// Mark as loading in this request's import stack
	rootEnv.importStack[absPath] = true
	defer delete(rootEnv.importStack, absPath)

	// Read the file
	content, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return newIOError("IO-0002", absPath, err)
		}
		return newIOError("IO-0003", absPath, err)
	}

	// Parse the module (with filename for error reporting)
	l := lexer.NewWithFilename(string(content), absPath)
	p := parser.New(l)
	program := p.ParseProgram()

	// Check for parse errors using structured errors
	if errs := p.StructuredErrors(); len(errs) > 0 {
		// Return the first parse error with file info preserved
		perr := errs[0]
		parseErr := &Error{
			Class:   ClassParse,
			Code:    perr.Code,
			Message: perr.Message,
			Hints:   perr.Hints,
			Line:    perr.Line,
			Column:  perr.Column,
			File:    absPath,
			Data:    perr.Data,
		}
		return parseErr
	}

	// Create isolated environment for the module
	moduleEnv := NewEnvironment()
	moduleEnv.Filename = absPath
	// Copy root path from parent environment (preserved across imports for ~/ resolution)
	moduleEnv.RootPath = env.RootPath
	// Copy security policy from parent environment
	moduleEnv.Security = env.Security
	// Copy DevLog and BasilCtx for stdlib imports (std/dev) and basil namespace modules
	moduleEnv.DevLog = env.DevLog
	moduleEnv.BasilCtx = env.BasilCtx
	// Copy ServerDB for module-scope database access (e.g., schema.table() at module level)
	moduleEnv.ServerDB = env.ServerDB
	// Copy AssetRegistry and AssetBundle for Basil server context
	moduleEnv.AssetRegistry = env.AssetRegistry
	moduleEnv.AssetBundle = env.AssetBundle

	// Copy basil context to module environment (if present)
	// This allows modules to access basil.http, basil.auth, basil.sqlite etc.
	if basil, ok := env.Get("basil"); ok {
		moduleEnv.SetProtected("basil", basil)
	}

	// Evaluate the module
	result := Eval(program, moduleEnv)

	// Check for errors during module evaluation
	if isError(result) {
		errObj := result.(*Error)
		// Preserve file info from module error
		if errObj.File == "" {
			errObj.File = absPath
		}
		return errObj
	}

	// Convert environment to dictionary
	moduleDict := environmentToDict(moduleEnv)

	// Mark as Part module if file extension is .part
	if strings.HasSuffix(absPath, ".part") {
		// Add __type metadata to identify this as a Part module
		moduleDict.Pairs["__type"] = &ast.StringLiteral{
			Token: lexer.Token{Type: lexer.STRING, Literal: "part"},
			Value: "part",
		}

		// Verify all exports are functions (Part module contract)
		for name, expr := range moduleDict.Pairs {
			if name == "__type" {
				continue
			}
			// Evaluate the expression to check its type
			obj := Eval(expr, moduleDict.Env)
			if _, ok := obj.(*Function); !ok {
				return &Error{
					Class:   ClassType,
					Code:    "PART-0001",
					Message: fmt.Sprintf("Part module export '%s' must be a function, got %s", name, obj.Type()),
					Hints:   []string{"All exports in .part files must be view functions", "Example: export default = fn(props) { <div>...</div> }"},
					File:    absPath,
				}
			}
		}
	}

	// Cache the result
	moduleCache.mu.Lock()
	moduleCache.modules[absPath] = moduleDict
	moduleCache.mu.Unlock()

	return moduleDict
}

// evalImport implements the import(path) builtin (legacy syntax)
// Delegates to importModule after extracting the path string.
func evalImport(args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("import", len(args), 1)
	}

	// Extract path string from argument (handle both path dictionaries and strings)
	var pathStr string
	switch arg := args[0].(type) {
	case *Dictionary:
		// Handle path literal (@/path/to/file.pars)
		if typeExpr, ok := arg.Pairs["__type"]; ok {
			typeVal := Eval(typeExpr, arg.Env)
			if typeStr, ok := typeVal.(*String); ok && typeStr.Value == "path" {
				pathStr = pathDictToString(arg)
			} else {
				return newTypeError("TYPE-0012", "import", "a path or string", DICTIONARY_OBJ)
			}
		} else {
			return newTypeError("TYPE-0012", "import", "a path or string", DICTIONARY_OBJ)
		}
	case *String:
		pathStr = arg.Value
	default:
		return newTypeError("TYPE-0012", "import", "a path or string", arg.Type())
	}

	return importModule(pathStr, env)
}

// evalLogLine implements logLine with filename and line number
func evalLogLine(args []Object, env *Environment) Object {
	var result strings.Builder

	// Add filename and line number prefix
	filename := env.Filename
	if filename == "" {
		filename = "<unknown>"
	}
	line := 1
	if env.LastToken != nil {
		line = env.LastToken.Line
	}
	result.WriteString(fmt.Sprintf("%s:%d: ", filename, line))

	// Process arguments like log()
	for i, arg := range args {
		if i == 0 {
			// First argument: if it's a string, show without quotes
			if str, ok := arg.(*String); ok {
				result.WriteString(str.Value)
			} else {
				result.WriteString(objectToDebugString(arg))
			}
		} else {
			// Subsequent arguments: add separator and debug format
			if i == 1 {
				// After first string, no comma - just space
				if _, firstWasString := args[0].(*String); firstWasString {
					result.WriteString(" ")
				} else {
					result.WriteString(", ")
				}
			} else {
				result.WriteString(", ")
			}
			result.WriteString(objectToDebugString(arg))
		}
	}

	// Use the environment's logger
	if env.Logger != nil {
		env.Logger.LogLine(result.String())
	} else {
		fmt.Fprintln(os.Stdout, result.String())
	}

	// Return null
	return NULL
}

// evalLog implements log() using the environment's logger
func evalLog(args []Object, env *Environment) Object {
	var result strings.Builder

	for i, arg := range args {
		if i == 0 {
			// First argument: if it's a string, show without quotes
			if str, ok := arg.(*String); ok {
				result.WriteString(str.Value)
			} else {
				result.WriteString(objectToDebugString(arg))
			}
		} else {
			// Subsequent arguments: add separator and debug format
			if i == 1 {
				// After first string, no comma - just space
				if _, firstWasString := args[0].(*String); firstWasString {
					result.WriteString(" ")
				} else {
					result.WriteString(", ")
				}
			} else {
				result.WriteString(", ")
			}
			result.WriteString(objectToDebugString(arg))
		}
	}

	// Use the environment's logger
	if env.Logger != nil {
		env.Logger.LogLine(result.String())
	} else {
		fmt.Fprintln(os.Stdout, result.String())
	}

	// Return null
	return NULL
}

func extendFunctionEnv(fn *Function, args []Object) *Environment {
	env := NewEnclosedEnvironment(fn.Env)

	// Use parameter list with destructuring support
	for paramIdx, param := range fn.Params {
		if paramIdx >= len(args) {
			break
		}
		arg := args[paramIdx]

		// Handle different parameter types
		if param.DictPattern != nil {
			// Dictionary destructuring (in function params, never exported)
			evalDictDestructuringAssignment(param.DictPattern, arg, env, true, false)
		} else if param.ArrayPattern != nil {
			// Array destructuring
			evalArrayPatternForParam(param.ArrayPattern, arg, env)
		} else if param.Ident != nil {
			// Simple identifier
			env.Set(param.Ident.Value, arg)
		}
	}

	return env
}

// evalArrayPatternForParam handles array destructuring in function parameters with explicit ...rest
func evalArrayPatternForParam(pattern *ast.ArrayDestructuringPattern, val Object, env *Environment) {
	// Convert value to array if it isn't already
	var elements []Object

	switch v := val.(type) {
	case *Array:
		elements = v.Elements
	default:
		// Single value becomes single-element array
		elements = []Object{v}
	}

	// Assign each named element to corresponding variable
	for i, name := range pattern.Names {
		if i < len(elements) {
			if name.Value != "_" {
				env.Set(name.Value, elements[i])
			}
		} else {
			// No more elements, assign null
			if name.Value != "_" {
				env.Set(name.Value, NULL)
			}
		}
	}

	// Handle rest parameter if present - ONLY collect remaining if explicit ...rest
	if pattern.Rest != nil && pattern.Rest.Value != "_" {
		var remaining *Array
		if len(elements) > len(pattern.Names) {
			remaining = &Array{Elements: elements[len(pattern.Names):]}
		} else {
			remaining = &Array{Elements: []Object{}}
		}
		env.Set(pattern.Rest.Value, remaining)
	}
	// Without explicit ...rest, extra elements are simply ignored (like JS/TS)
}

func unwrapReturnValue(obj Object) Object {
	if returnValue, ok := obj.(*ReturnValue); ok {
		return returnValue.Value
	}
	// Unwrap CheckExit to its value (functions use it like return)
	if checkExit, ok := obj.(*CheckExit); ok {
		return checkExit.Value
	}
	// Stop/skip signals outside of for loops are errors
	if _, ok := obj.(*StopSignal); ok {
		return &Error{
			Class:   ClassType,
			Code:    "LOOP-0008",
			Message: "'stop' can only be used inside a for loop",
		}
	}
	if _, ok := obj.(*SkipSignal); ok {
		return &Error{
			Class:   ClassType,
			Code:    "LOOP-0009",
			Message: "'skip' can only be used inside a for loop",
		}
	}
	return obj
}

// Check statement and for loop evaluation (evalCheckStatement, evalForExpression,
// evalForDictExpression) are in eval_control_flow.go

// All error creation helpers moved to eval_errors.go

// enrichErrorWithPos adds position info to an error that doesn't have it.
// This is useful for wrapping errors from builtins at the call site.
func enrichErrorWithPos(obj Object, tok *lexer.Token) Object {
	if tok == nil {
		return obj
	}
	if errObj, ok := obj.(*Error); ok && errObj.Line == 0 {
		errObj.Line = tok.Line
		errObj.Column = tok.Column
	}
	return obj
}

func isError(obj Object) bool {
	if obj != nil {
		return obj.Type() == ERROR_OBJ
	}
	return false
}

// withPosition adds line/column position to an error if it doesn't already have one.
// Returns the object unchanged if it's not an error or already has position info.
func withPosition(obj Object, tok lexer.Token, env *Environment) Object {
	if err, ok := obj.(*Error); ok {
		if err.Line == 0 && err.Column == 0 {
			err.Line = tok.Line
			err.Column = tok.Column
			if err.File == "" && env != nil && env.Filename != "" {
				err.File = env.Filename
			}
		}
	}
	return obj
}

// dispatchMethodCall dispatches a method call to the appropriate type-specific handler.
// Returns nil if the type doesn't match any handler (falls through to property access).

func evalArrayPatternAssignment(pattern *ast.ArrayDestructuringPattern, val Object, env *Environment, isLet bool, export bool) Object {
	// Convert value to array if it isn't already
	var elements []Object

	switch v := val.(type) {
	case *Array:
		elements = v.Elements
	default:
		// Single value becomes single-element array
		elements = []Object{v}
	}

	// Assign each named element to corresponding variable
	for i, name := range pattern.Names {
		if i < len(elements) {
			// Direct assignment for elements within bounds
			if name.Value != "_" {
				if export && isLet {
					env.SetLetExport(name.Value, elements[i])
				} else if export {
					env.SetExport(name.Value, elements[i])
				} else if isLet {
					env.SetLet(name.Value, elements[i])
				} else {
					env.Update(name.Value, elements[i])
				}
			}
		} else {
			// No more elements, assign null
			if name.Value != "_" {
				if export && isLet {
					env.SetLetExport(name.Value, NULL)
				} else if export {
					env.SetExport(name.Value, NULL)
				} else if isLet {
					env.SetLet(name.Value, NULL)
				} else {
					env.Update(name.Value, NULL)
				}
			}
		}
	}

	// Handle rest parameter if present - ONLY collect remaining if explicit ...rest
	if pattern.Rest != nil && pattern.Rest.Value != "_" {
		var remaining *Array
		if len(elements) > len(pattern.Names) {
			remaining = &Array{Elements: elements[len(pattern.Names):]}
		} else {
			remaining = &Array{Elements: []Object{}}
		}
		if export && isLet {
			env.SetLetExport(pattern.Rest.Value, remaining)
		} else if export {
			env.SetExport(pattern.Rest.Value, remaining)
		} else if isLet {
			env.SetLet(pattern.Rest.Value, remaining)
		} else {
			env.Update(pattern.Rest.Value, remaining)
		}
	}
	// Without explicit ...rest, extra elements are simply ignored (like JS/TS)

	// Destructuring assignments return NULL (excluded from block concatenation)
	return NULL
}

// evalDestructuringAssignment handles simple array destructuring assignment (legacy, for db queries)
// This is kept for backwards compatibility with database query statements that use Names []*Identifier
func evalDestructuringAssignment(names []*ast.Identifier, val Object, env *Environment, isLet bool, export bool) Object {
	// Convert value to array if it isn't already
	var elements []Object

	switch v := val.(type) {
	case *Array:
		elements = v.Elements
	default:
		// Single value becomes single-element array
		elements = []Object{v}
	}

	// Assign each element to corresponding variable
	for i, name := range names {
		if i < len(elements) {
			// Direct assignment for elements within bounds
			if name.Value != "_" {
				if export && isLet {
					env.SetLetExport(name.Value, elements[i])
				} else if export {
					env.SetExport(name.Value, elements[i])
				} else if isLet {
					env.SetLet(name.Value, elements[i])
				} else {
					env.Update(name.Value, elements[i])
				}
			}
		} else {
			// No more elements, assign null
			if name.Value != "_" {
				if export && isLet {
					env.SetLetExport(name.Value, NULL)
				} else if export {
					env.SetExport(name.Value, NULL)
				} else if isLet {
					env.SetLet(name.Value, NULL)
				} else {
					env.Update(name.Value, NULL)
				}
			}
		}
	}

	// Note: This legacy function does NOT support rest parameters
	// Extra elements are ignored (for consistency with the new behavior)

	// Destructuring assignments return NULL (excluded from block concatenation)
	return NULL
}

// evalDictDestructuringAssignment evaluates dictionary destructuring patterns
func evalDictDestructuringAssignment(pattern *ast.DictDestructuringPattern, val Object, env *Environment, isLet bool, export bool) Object {
	// Handle StdlibModuleDict (from @std/ imports)
	if stdlibMod, ok := val.(*StdlibModuleDict); ok {
		return evalStdlibModuleDestructuring(pattern, stdlibMod, env, isLet, export)
	}

	// Type check: value must be a dictionary
	dict, ok := val.(*Dictionary)
	if !ok {
		return newDestructuringError("DEST-0001", val)
	}

	// Track which keys we've extracted (for rest operator)
	extractedKeys := make(map[string]bool)

	// Process each key in the pattern
	for _, keyPattern := range pattern.Keys {
		keyName := keyPattern.Key.Value
		extractedKeys[keyName] = true

		// Get expression from dictionary and evaluate it
		var value Object
		if expr, exists := dict.Pairs[keyName]; exists {
			// Evaluate the expression in the dictionary's environment
			value = Eval(expr, dict.Env)
			if isError(value) {
				return value
			}
		} else {
			// If key not found, assign null
			value = NULL
		}

		// Handle nested destructuring
		if keyPattern.Nested != nil {
			if nestedPattern, ok := keyPattern.Nested.(*ast.DictDestructuringPattern); ok {
				result := evalDictDestructuringAssignment(nestedPattern, value, env, isLet, export)
				if isError(result) {
					return result
				}
			} else {
				return newDestructuringError("DEST-0002", nil)
			}
		} else {
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
	}

	// Handle rest operator
	if pattern.Rest != nil {
		restPairs := make(map[string]ast.Expression)
		for key, expr := range dict.Pairs {
			if !extractedKeys[key] {
				restPairs[key] = expr
			}
		}

		restDict := &Dictionary{Pairs: restPairs, Env: dict.Env}
		if pattern.Rest.Value != "_" {
			if export && isLet {
				env.SetLetExport(pattern.Rest.Value, restDict)
			} else if export {
				env.SetExport(pattern.Rest.Value, restDict)
			} else if isLet {
				env.SetLet(pattern.Rest.Value, restDict)
			} else {
				env.Update(pattern.Rest.Value, restDict)
			}
		}
	}

	// Destructuring assignments return NULL (excluded from block concatenation)
	return NULL
}

