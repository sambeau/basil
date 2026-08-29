package evaluator

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
	perrors "github.com/sambeau/basil/pkg/parsley/errors"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// Expression evaluation: function application, parameter handling, assignments, destructuring
// Extracted from evaluator.go - Phase 5 Extraction 31

// MaxCallDepth bounds how deeply Parsley functions may recurse within a single
// evaluation. It exists to convert unbounded recursion — which would otherwise
// overflow the Go goroutine stack and crash the whole process — into a catchable
// Parsley error. Embedders may raise or lower it before evaluating. The default is
// generous enough for legitimate recursion yet well below the depth at which the
// interpreter's own stack frames exhaust a default goroutine stack.
var MaxCallDepth = 5000

// enterCall increments the shared call-depth counter for the evaluation tree that
// callerEnv belongs to and reports whether the limit has been exceeded. The returned
// leave function must be called (typically via defer) to decrement the counter again.
// It tolerates a nil counter by lazily seeding one, so a function invoked with an
// environment that predates the counter is still guarded from that point on.
func enterCall(callerEnv *Environment) (exceeded bool, leave func()) {
	if callerEnv == nil {
		return false, func() {}
	}
	if callerEnv.callDepth == nil {
		d := 0
		callerEnv.callDepth = &d
	}
	counter := callerEnv.callDepth
	*counter++
	if *counter > MaxCallDepth {
		*counter--
		return true, func() {}
	}
	return false, func() { *counter-- }
}

func applyFunction(fn Object, args []Object) Object {
	switch fn := fn.(type) {
	case *Function:
		if err := checkCallArity(fn, len(args)); err != nil {
			return err
		}
		extendedEnv := extendFunctionEnv(fn, args)
		exceeded, leave := enterCall(extendedEnv)
		if exceeded {
			return newCallError("CALL-0007", map[string]any{"Limit": MaxCallDepth})
		}
		defer leave()
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
	// Returned bare (Line 0) so the caller stamps the call site's token onto it.
	if err := checkCallArity(fn, len(args)); err != nil {
		return err
	}
	extendedEnv := extendFunctionEnv(fn, args)
	extendedEnv.Set("this", thisObj)
	// Copy runtime context from calling environment (like ApplyFunctionWithEnv does)
	// This ensures features like <CSS/>, <Javascript/>, and request-scoped values work
	if env != nil {
		extendedEnv.callDepth = env.callDepth // unify the call-depth counter along the dynamic call chain
		extendedEnv.AssetBundle = env.AssetBundle
		extendedEnv.AssetRegistry = env.AssetRegistry
		extendedEnv.ImageRegistry = env.ImageRegistry
		extendedEnv.FragmentCache = env.FragmentCache
		extendedEnv.BasilCtx = env.BasilCtx
		extendedEnv.DataPath = env.DataPath
		extendedEnv.DevLog = env.DevLog
		extendedEnv.HandlerPath = env.HandlerPath
		extendedEnv.NoCache = env.NoCache
		extendedEnv.TrustModuleCache = env.TrustModuleCache
		extendedEnv.Security = env.Security
		extendedEnv.Logger = env.Logger
		// Copy @params from calling environment so methods can access request params
		if params, ok := env.Get("@params"); ok {
			extendedEnv.Set("@params", params)
		}
	}
	exceeded, leave := enterCall(extendedEnv)
	if exceeded {
		return enrichErrorWithPos(newCallError("CALL-0007", map[string]any{"Limit": MaxCallDepth}), lastTokenOf(env))
	}
	defer leave()
	evaluated := Eval(fn.Body, extendedEnv)
	return unwrapReturnValue(evaluated)
}

// lastTokenOf returns the last-seen token of env for error positioning, or nil.
func lastTokenOf(env *Environment) *lexer.Token {
	if env == nil {
		return nil
	}
	return env.LastToken
}

// ApplyFunctionWithEnv applies a function with the given arguments in the context of an environment.
// It handles parameter binding including destructuring patterns, and copies runtime context
// from the calling environment to ensure features like <CSS/>, <Javascript/> work in imported components.
func ApplyFunctionWithEnv(fn Object, args []Object, env *Environment) Object {
	switch fn := fn.(type) {
	case *Function:
		// Returned bare (Line 0) so the caller stamps the call site's token onto it.
		if err := checkCallArity(fn, len(args)); err != nil {
			return err
		}
		extendedEnv := extendFunctionEnv(fn, args)
		// Copy runtime context from calling environment to function environment
		// This ensures <Css/>, <Script/>, and other runtime features work in imported components
		if env != nil {
			extendedEnv.callDepth = env.callDepth // unify the call-depth counter along the dynamic call chain
			extendedEnv.AssetBundle = env.AssetBundle
			extendedEnv.AssetRegistry = env.AssetRegistry
			extendedEnv.ImageRegistry = env.ImageRegistry
			extendedEnv.FragmentCache = env.FragmentCache
			extendedEnv.BasilCtx = env.BasilCtx
			extendedEnv.DataPath = env.DataPath
			extendedEnv.DevLog = env.DevLog
			extendedEnv.HandlerPath = env.HandlerPath
			extendedEnv.NoCache = env.NoCache
			extendedEnv.TrustModuleCache = env.TrustModuleCache
			extendedEnv.Security = env.Security
			extendedEnv.Logger = env.Logger
			// Copy @params from calling environment so modules can access request params
			// via `let {params} = import @basil/http`
			if params, ok := env.Get("@params"); ok {
				extendedEnv.Set("@params", params)
			}
		}
		exceeded, leave := enterCall(extendedEnv)
		if exceeded {
			return enrichErrorWithPos(newCallError("CALL-0007", map[string]any{"Limit": MaxCallDepth}), lastTokenOf(env))
		}
		defer leave()
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

	case *MdDocModule:
		// MdDocModule is callable: mdDoc(text) or mdDoc(dict) creates an MdDoc
		result := evalMdDocModuleCall(args, env)
		if isError(result) {
			return enrichErrorWithPos(result, env.LastToken)
		}
		return result
	case *DSLSchema:
		// DSLSchema is callable: Schema({...}) creates a Record, Schema([...]) creates a Table
		result := evalSchemaCall(fn, args, env)
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
		// Enrich error with position information from the import statement
		return withPosition(module, node.Token, env)
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
	if after, ok := strings.CutPrefix(pathStr, "std/"); ok {
		moduleName := after
		return loadStdlibModule(moduleName, env)
	}

	// Check for basil namespace imports (basil/modulename)
	if after, ok := strings.CutPrefix(pathStr, "basil/"); ok {
		moduleName := after
		return loadBasilModule(moduleName, env)
	}

	// Resolve path relative to current file (or root path for ~/ paths)
	absPath, err := resolveModulePath(pathStr, env.Filename, env.RootPath)
	if err != nil {
		return newImportError("IMPORT-0004", map[string]any{"GoError": err.Error()})
	}

	// Security check - module imports are reads, not executes
	if err := env.checkPathAccess(absPath, "read"); err != nil {
		return newSecurityError("read", err)
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

	// Record this import against the module being evaluated, so that module
	// can record what it was built from. Done before the cache lookup: a
	// dependency is a dependency whether or not it was served from cache.
	env.moduleDeps.add(absPath)

	// A cached module is served only if it can still be shown to be current.
	// env.TrustModuleCache skips that check, and is set only where the sources
	// cannot change under the running evaluation - a production release, or
	// dev.cache. Everywhere else, including the pars CLI and the REPL, gets
	// the check by default.
	if cached, ok := moduleCache.lookup(absPath, !env.TrustModuleCache); ok {
		return cached
	}

	// Stamped before the read, never after - see ModuleCache.store. A file
	// that cannot be stat'd is left to the read below to report properly.
	stamp, stampErr := stampOf(absPath)

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
	moduleEnv.DataPath = env.DataPath
	// Copy security policy from parent environment
	moduleEnv.Security = env.Security
	// Carry the caching decisions into the module. Without this a module's own
	// imports, and any <basil.cache.Cache> at module scope, cache in dev mode
	// however the request was configured.
	moduleEnv.NoCache = env.NoCache
	moduleEnv.TrustModuleCache = env.TrustModuleCache
	// A fresh collector: what this module imports is what this module was
	// built from, not what its importer was.
	moduleEnv.moduleDeps = newModuleDepSet()
	// Copy DevLog and BasilCtx for stdlib imports (std/dev) and basil namespace modules
	moduleEnv.DevLog = env.DevLog
	moduleEnv.BasilCtx = env.BasilCtx
	// Copy ServerDB for module-scope database access (e.g., schema.table() at module level)
	moduleEnv.ServerDB = env.ServerDB
	// Copy AssetRegistry, ImageRegistry, and AssetBundle for Basil server context
	moduleEnv.AssetRegistry = env.AssetRegistry
	moduleEnv.ImageRegistry = env.ImageRegistry
	moduleEnv.AssetBundle = env.AssetBundle
	// Copy PLNSecret for Record serialization in Parts (FEAT-098)
	moduleEnv.PLNSecret = env.PLNSecret

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

	// Cache the result against the stamp taken before the read, together with
	// what this module imported. A module whose own file could not be stat'd
	// is not cached: there would be nothing to validate it against later.
	if stampErr == nil {
		moduleCache.store(absPath, stamp, moduleDict, moduleEnv.moduleDeps.snapshot())
	}

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
	fmt.Fprintf(&result, "%s:%d: ", filename, line)

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

// evalDictDestructuringAssignmentImmutable is a helper for function parameters
// It calls evalDictDestructuringAssignment with isLet=true, export=false, mutable=false
// to ensure all destructured bindings are immutable (function params cannot be reassigned)
func evalDictDestructuringAssignmentImmutable(pattern *ast.DictDestructuringPattern, val Object, env *Environment) Object {
	return evalDictDestructuringAssignment(pattern, val, env, true, false, false)
}

// checkCallArity enforces that a user-defined function is called with exactly as
// many positional arguments as it declares parameters (BUG-032). It returns
// nil when the call is well-formed.
//
// The check lives here — called from applyFunction, ApplyFunctionWithEnv and
// applyMethodWithThis, the three entry points for a *direct* call — rather than
// inside extendFunctionEnv, because the error must be raised before the body
// runs and must be positioned at the call site. Those callers all return the
// error to an evaluator that stamps the call expression's token onto it
// (withPosition / enrichErrorWithPos), so the caret lands on the call and not
// on the unbound parameter inside the callee.
//
// Internal callback dispatch (.map, .reduce, table methods, tag components,
// markdown walkers) does not go through these entry points blindly: each such
// site inspects fn.ParamCount() and passes a matching number of arguments, the
// way `for (x in …) fn` already does.
//
// Destructuring parameters (`fn({a, b})`, `fn([x, y])`) count as one parameter
// each; their inner leniency is by design and untouched here.
func checkCallArity(fn *Function, argc int) *Error {
	want := fn.ParamCount()
	if argc == want {
		return nil
	}
	return newUserArityError(fn.DisplayName(), argc, want)
}

func extendFunctionEnv(fn *Function, args []Object) *Environment {
	env := NewEnclosedEnvironment(fn.Env)

	// Use parameter list with destructuring support
	// All function parameters are immutable (cannot be reassigned within the function body)
	for paramIdx, param := range fn.Params {
		if paramIdx >= len(args) {
			break
		}
		// End any active table chain when passing as argument
		arg := endTableChain(args[paramIdx])

		// Handle different parameter types
		if param.DictPattern != nil {
			// Dictionary destructuring (in function params, never exported, immutable)
			evalDictDestructuringAssignmentImmutable(param.DictPattern, arg, env)
		} else if param.ArrayPattern != nil {
			// Array destructuring (immutable)
			evalArrayPatternForParam(param.ArrayPattern, arg, env)
		} else if param.Ident != nil {
			// Simple identifier - use SetLet to make immutable
			env.SetLet(param.Ident.Value, arg)
		}
	}

	return env
}

// evalArrayPatternForParam handles array destructuring in function parameters with explicit ...rest
// All bindings are immutable (function parameters cannot be reassigned)
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

	// Assign each named element to corresponding variable (immutable)
	for i, name := range pattern.Names {
		if i < len(elements) {
			if name.Value != "_" {
				env.SetLet(name.Value, elements[i])
			}
		} else {
			// No more elements, assign null
			if name.Value != "_" {
				env.SetLet(name.Value, NULL)
			}
		}
	}

	// Handle rest parameter if present - ONLY collect remaining if explicit ...rest (immutable)
	if pattern.Rest != nil && pattern.Rest.Value != "_" {
		var remaining *Array
		if len(elements) > len(pattern.Names) {
			remaining = &Array{Elements: elements[len(pattern.Names):]}
		} else {
			remaining = &Array{Elements: []Object{}}
		}
		env.SetLet(pattern.Rest.Value, remaining)
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

func evalArrayPatternAssignment(pattern *ast.ArrayDestructuringPattern, val Object, env *Environment, isLet bool, export bool, mutable ...bool) Object {
	// Check if mutable flag is passed (default to false for backwards compatibility)
	isMutable := len(mutable) > 0 && mutable[0]

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
		if isLet {
			if err := env.CheckRedeclare(name.Value); err != nil {
				return withPosition(err, pattern.Token, env)
			}
		}
		if i < len(elements) {
			// Direct assignment for elements within bounds
			if name.Value != "_" {
				if export && isLet {
					if isMutable {
						env.SetVarExport(name.Value, elements[i])
					} else {
						env.SetLetExport(name.Value, elements[i])
					}
				} else if export {
					env.SetExport(name.Value, elements[i])
				} else if isLet {
					if isMutable {
						env.SetVar(name.Value, elements[i])
					} else {
						env.SetLet(name.Value, elements[i])
					}
				} else {
					if err := env.Update(name.Value, elements[i]); isError(err) {
						return err
					}
				}
			}
		} else {
			// No more elements, assign null
			if name.Value != "_" {
				if export && isLet {
					if isMutable {
						env.SetVarExport(name.Value, NULL)
					} else {
						env.SetLetExport(name.Value, NULL)
					}
				} else if export {
					env.SetExport(name.Value, NULL)
				} else if isLet {
					if isMutable {
						env.SetVar(name.Value, NULL)
					} else {
						env.SetLet(name.Value, NULL)
					}
				} else {
					if err := env.Update(name.Value, NULL); isError(err) {
						return err
					}
				}
			}
		}
	}

	// Handle rest parameter if present - ONLY collect remaining if explicit ...rest
	if pattern.Rest != nil && pattern.Rest.Value != "_" {
		if isLet {
			if err := env.CheckRedeclare(pattern.Rest.Value); err != nil {
				return withPosition(err, pattern.Token, env)
			}
		}
		var remaining *Array
		if len(elements) > len(pattern.Names) {
			remaining = &Array{Elements: elements[len(pattern.Names):]}
		} else {
			remaining = &Array{Elements: []Object{}}
		}
		if export && isLet {
			if isMutable {
				env.SetVarExport(pattern.Rest.Value, remaining)
			} else {
				env.SetLetExport(pattern.Rest.Value, remaining)
			}
		} else if export {
			env.SetExport(pattern.Rest.Value, remaining)
		} else if isLet {
			if isMutable {
				env.SetVar(pattern.Rest.Value, remaining)
			} else {
				env.SetLet(pattern.Rest.Value, remaining)
			}
		} else {
			if err := env.Update(pattern.Rest.Value, remaining); isError(err) {
				return err
			}
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
		if isLet {
			if err := env.CheckRedeclare(name.Value); err != nil {
				return withPosition(err, name.Token, env)
			}
		}
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
					if err := env.Update(name.Value, elements[i]); isError(err) {
						return err
					}
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
					if err := env.Update(name.Value, NULL); isError(err) {
						return err
					}
				}
			}
		}
	}

	// Note: This legacy function does NOT support rest parameters
	// Extra elements are ignored (for consistency with the new behavior)

	// Destructuring assignments return NULL (excluded from block concatenation)
	return NULL
}

// evalDictDestructuringAssignment evaluates dictionary/record destructuring patterns
func evalDictDestructuringAssignment(pattern *ast.DictDestructuringPattern, val Object, env *Environment, isLet bool, export bool, mutable ...bool) Object {
	// Check if mutable flag is passed (default to false for backwards compatibility)
	isMutable := len(mutable) > 0 && mutable[0]
	// Handle StdlibModuleDict (from @std/ imports)
	if stdlibMod, ok := val.(*StdlibModuleDict); ok {
		return evalStdlibModuleDestructuring(pattern, stdlibMod, env, isLet, export)
	}

	// Type check: value must be a dictionary or record
	// Extract pairs and env from either type
	var pairs map[string]ast.Expression
	var keyOrder []string
	var valEnv *Environment

	switch v := val.(type) {
	case *Dictionary:
		pairs = v.Pairs
		keyOrder = v.KeyOrder
		valEnv = v.Env
	case *Record:
		pairs = v.Data
		keyOrder = v.KeyOrder
		valEnv = v.Env
	default:
		err := newDestructuringError("DEST-0001", val)
		err.Line = pattern.Token.Line
		err.Column = pattern.Token.Column
		if env != nil && env.Filename != "" {
			err.File = env.Filename
		}
		return err
	}

	// Track which keys we've extracted (for rest operator)
	extractedKeys := make(map[string]bool)

	// Process each key in the pattern
	for _, keyPattern := range pattern.Keys {
		keyName := keyPattern.Key.Value
		extractedKeys[keyName] = true

		// Get expression from dictionary/record and evaluate it
		var value Object
		if expr, exists := pairs[keyName]; exists {
			// Evaluate the expression in the value's environment
			value = Eval(expr, valEnv)
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
				result := evalDictDestructuringAssignment(nestedPattern, value, env, isLet, export, isMutable)
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

			if isLet {
				if err := env.CheckRedeclare(targetName); err != nil {
					return withPosition(err, keyPattern.Token, env)
				}
			}

			// Assign to environment
			if targetName != "_" {
				if export && isLet {
					if isMutable {
						env.SetVarExport(targetName, value)
					} else {
						env.SetLetExport(targetName, value)
					}
				} else if export {
					env.SetExport(targetName, value)
				} else if isLet {
					if isMutable {
						env.SetVar(targetName, value)
					} else {
						env.SetLet(targetName, value)
					}
				} else {
					if err := env.Update(targetName, value); isError(err) {
						return err
					}
				}
			}
		}
	}

	// Handle rest operator
	if pattern.Rest != nil {
		restPairs := make(map[string]ast.Expression)
		for key, expr := range pairs {
			if !extractedKeys[key] {
				restPairs[key] = expr
			}
		}

		// Carry over the source's insertion order, minus the extracted keys.
		restOrder := make([]string, 0, len(restPairs))
		ordered := make(map[string]bool, len(restPairs))
		for _, key := range keyOrder {
			if _, ok := restPairs[key]; ok && !ordered[key] {
				restOrder = append(restOrder, key)
				ordered[key] = true
			}
		}
		// A source built without a complete KeyOrder leaves stragglers; append
		// them sorted so the result is at least deterministic.
		if len(restOrder) < len(restPairs) {
			extras := make([]string, 0, len(restPairs)-len(restOrder))
			for key := range restPairs {
				if !ordered[key] {
					extras = append(extras, key)
				}
			}
			sort.Strings(extras)
			restOrder = append(restOrder, extras...)
		}

		restDict := &Dictionary{Pairs: restPairs, KeyOrder: restOrder, Env: valEnv}
		if pattern.Rest.Value != "_" {
			if isLet {
				if err := env.CheckRedeclare(pattern.Rest.Value); err != nil {
					return withPosition(err, pattern.Token, env)
				}
			}
			if export && isLet {
				if isMutable {
					env.SetVarExport(pattern.Rest.Value, restDict)
				} else {
					env.SetLetExport(pattern.Rest.Value, restDict)
				}
			} else if export {
				env.SetExport(pattern.Rest.Value, restDict)
			} else if isLet {
				if isMutable {
					env.SetVar(pattern.Rest.Value, restDict)
				} else {
					env.SetLet(pattern.Rest.Value, restDict)
				}
			} else {
				if err := env.Update(pattern.Rest.Value, restDict); isError(err) {
					return err
				}
			}
		}
	}

	// Destructuring assignments return NULL (excluded from block concatenation)
	return NULL
}
