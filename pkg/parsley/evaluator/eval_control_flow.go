package evaluator

import (
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
	perrors "github.com/sambeau/basil/pkg/parsley/errors"
)

// Control flow evaluation functions: check, for, try

func evalCheckStatement(node *ast.CheckStatement, env *Environment) Object {
	condition := Eval(node.Condition, env)
	if isError(condition) {
		return condition
	}

	if !isTruthy(condition) {
		// Condition failed, evaluate and return the else value as CheckExit
		elseValue := Eval(node.ElseValue, env)
		if isError(elseValue) {
			return elseValue
		}
		return &CheckExit{Value: elseValue}
	}

	// Condition passed, continue execution (return NULL to not affect result stream)
	return NULL
}

// evalForExpression evaluates for expressions
func evalForExpression(node *ast.ForExpression, env *Environment) Object {
	// Evaluate the array/dict expression
	iterableObj := Eval(node.Array, env)
	if isError(iterableObj) {
		return iterableObj
	}

	// Handle response typed dictionary - unwrap __data for iteration
	if dict, ok := iterableObj.(*Dictionary); ok && isResponseDict(dict) {
		if dataExpr, ok := dict.Pairs["__data"]; ok {
			iterableObj = Eval(dataExpr, dict.Env)
			if isError(iterableObj) {
				return iterableObj
			}
		}
	}

	// Handle dictionary iteration
	if dict, ok := iterableObj.(*Dictionary); ok {
		return evalForDictExpression(node, dict, env)
	}

	// Convert to array (handle strings as rune arrays)
	var elements []Object
	switch arr := iterableObj.(type) {
	case *Array:
		elements = arr.Elements
	case *String:
		// Convert string to array of single-character strings
		runes := []rune(arr.Value)
		elements = make([]Object, len(runes))
		for i, r := range runes {
			elements[i] = &String{Value: string(r)}
		}
	default:
		return newLoopErrorWithPos(node.Token, "LOOP-0001", map[string]any{"Type": strings.ToLower(string(iterableObj.Type()))})
	}

	// Determine which function to use
	var fn Object
	if node.Function != nil {
		// Simple form: for(array) func
		fn = Eval(node.Function, env)
		if isError(fn) {
			return fn
		}
		// Accept both functions and builtins
		switch fn.(type) {
		case *Function, *Builtin:
			// OK
		default:
			return newLoopErrorWithPos(node.Token, "LOOP-0002", map[string]any{"Type": strings.ToLower(string(fn.Type()))})
		}
	} else if node.Body != nil {
		// 'in' form: for(var in array) body
		// node.Body is already a FunctionLiteral with the variable as parameter
		fn = &Function{
			Params: node.Body.(*ast.FunctionLiteral).Params,
			Body:   node.Body.(*ast.FunctionLiteral).Body,
			Env:    env,
		}
	} else {
		return newLoopErrorWithPos(node.Token, "LOOP-0003", nil)
	}

	// Map function over array elements
	result := []Object{}
	for idx, elem := range elements {
		var evaluated Object
		var stopLoop bool
		var skipIteration bool

		switch f := fn.(type) {
		case *Builtin:
			// Call builtin with single element
			evaluated = f.Fn(elem)
		case *Function:
			// Call user function
			paramCount := f.ParamCount()
			if paramCount != 1 && paramCount != 2 {
				return newLoopErrorWithPos(node.Token, "LOOP-0004", map[string]any{"Got": paramCount})
			}

			// Prepare arguments based on parameter count
			var args []Object
			if paramCount == 2 {
				// Two parameters: index and element
				args = []Object{&Integer{Value: int64(idx)}, elem}
			} else {
				// One parameter: element only (backward compatible)
				args = []Object{elem}
			}

			// Create a new environment and bind the parameters
			extendedEnv := extendFunctionEnv(f, args)

			// Evaluate all statements in the body and collect results
			var bodyResults []Object
			for _, stmt := range f.Body.Statements {
				evaluated = evalStatement(stmt, extendedEnv)
				if returnValue, ok := evaluated.(*ReturnValue); ok {
					evaluated = returnValue.Value
					bodyResults = append(bodyResults, evaluated)
					break
				}
				// Handle stop signal - exit loop early
				if _, ok := evaluated.(*StopSignal); ok {
					stopLoop = true
					evaluated = NULL
					break
				}
				// Handle skip signal - skip this iteration
				if _, ok := evaluated.(*SkipSignal); ok {
					skipIteration = true
					evaluated = NULL
					break
				}
				// Handle check exit - use the exit value and exit the block
				if checkExit, ok := evaluated.(*CheckExit); ok {
					evaluated = checkExit.Value
					bodyResults = append(bodyResults, evaluated)
					break
				}
				if isError(evaluated) {
					return evaluated
				}
				// Handle PrintValue in for loop body
				if pv, ok := evaluated.(*PrintValue); ok {
					for _, v := range pv.Values {
						str := objectToUserString(v)
						if str != "" {
							bodyResults = append(bodyResults, &String{Value: str})
						}
					}
					continue
				}
				// Collect non-NULL results
				if evaluated != NULL {
					bodyResults = append(bodyResults, evaluated)
				}
			}

			// Use block result rules: single result or array
			switch len(bodyResults) {
			case 0:
				evaluated = NULL
			case 1:
				evaluated = bodyResults[0]
			default:
				evaluated = &Array{Elements: bodyResults}
			}
		}

		// Handle skip - don't add anything to result
		if skipIteration {
			continue
		}

		// Skip null values (filter behavior)
		if evaluated != NULL {
			result = append(result, evaluated)
		}

		// Handle stop - exit after collecting any non-null result
		if stopLoop {
			break
		}
	}

	return &Array{Elements: result}
}

// evalForDictExpression handles for loops over dictionaries
func evalForDictExpression(node *ast.ForExpression, dict *Dictionary, env *Environment) Object {
	// Create environment for evaluation with 'this'
	dictEnv := NewEnclosedEnvironment(dict.Env)
	dictEnv.Set("this", dict)

	// Determine which function to use
	var fn *Function
	if node.Body != nil {
		bodyFn := node.Body.(*ast.FunctionLiteral)
		if len(bodyFn.Params) > 0 {
			fn = &Function{
				Params: bodyFn.Params,
				Body:   bodyFn.Body,
				Env:    env,
			}
		}
	}

	// If no function or body provided, error
	if fn == nil && node.Function == nil {
		return newLoopErrorWithPos(node.Token, "LOOP-0003", nil)
	}

	// Get function if not from body
	if fn == nil {
		fnObj := Eval(node.Function, env)
		if isError(fnObj) {
			return fnObj
		}
		// Accept both functions and builtins
		switch f := fnObj.(type) {
		case *Function:
			fn = f
		case *Builtin:
			// Builtins not supported for dict iteration
			return newLoopErrorWithPos(node.Token, "LOOP-0005", map[string]any{"Type": strings.ToLower(string(fnObj.Type()))})
		default:
			return newLoopErrorWithPos(node.Token, "LOOP-0002", map[string]any{"Type": strings.ToLower(string(fnObj.Type()))})
		}
	}

	// Verify function has correct arity
	paramCount := fn.ParamCount()
	if paramCount != 1 && paramCount != 2 {
		return newLoopErrorWithPos(node.Token, "LOOP-0006", map[string]any{"Got": paramCount})
	}

	// Iterate over dictionary
	result := []Object{}
	for _, key := range dict.KeyOrder {
		expr := dict.Pairs[key]
		value := Eval(expr, dictEnv)
		if isError(value) {
			return value
		}

		// Prepare arguments based on parameter count
		var args []Object
		if paramCount == 2 {
			// Two parameters: key and value
			args = []Object{&String{Value: key}, value}
		} else {
			// One parameter: value only
			args = []Object{value}
		}

		// Create extended environment with parameters bound
		extendedEnv := extendFunctionEnv(fn, args)

		// Evaluate function body
		var bodyResults []Object
		var stopLoop bool
		var skipIteration bool
		var evaluated Object
		for _, stmt := range fn.Body.Statements {
			evaluated = evalStatement(stmt, extendedEnv)
			if returnValue, ok := evaluated.(*ReturnValue); ok {
				evaluated = returnValue.Value
				bodyResults = append(bodyResults, evaluated)
				break
			}
			// Handle stop signal - exit loop early
			if _, ok := evaluated.(*StopSignal); ok {
				stopLoop = true
				evaluated = NULL
				break
			}
			// Handle skip signal - skip this iteration
			if _, ok := evaluated.(*SkipSignal); ok {
				skipIteration = true
				evaluated = NULL
				break
			}
			// Handle check exit
			if checkExit, ok := evaluated.(*CheckExit); ok {
				evaluated = checkExit.Value
				bodyResults = append(bodyResults, evaluated)
				break
			}
			if isError(evaluated) {
				return evaluated
			}
			// Handle PrintValue in for loop body
			if pv, ok := evaluated.(*PrintValue); ok {
				for _, v := range pv.Values {
					str := objectToUserString(v)
					if str != "" {
						bodyResults = append(bodyResults, &String{Value: str})
					}
				}
				continue
			}
			// Collect non-NULL results
			if evaluated != NULL {
				bodyResults = append(bodyResults, evaluated)
			}
		}

		// Use block result rules: single result or array
		switch len(bodyResults) {
		case 0:
			evaluated = NULL
		case 1:
			evaluated = bodyResults[0]
		default:
			evaluated = &Array{Elements: bodyResults}
		}

		// Handle skip - don't add anything to result
		if skipIteration {
			continue
		}

		// Skip null values (filter behavior)
		if evaluated != NULL {
			result = append(result, evaluated)
		}

		// Handle stop - exit after collecting any non-null result
		if stopLoop {
			break
		}
	}

	return &Array{Elements: result}
}

func evalTryExpression(node *ast.TryExpression, env *Environment) Object {
	// Evaluate the call expression
	result := Eval(node.Call, env)

	// If it's an error, check if it's catchable
	if err, ok := result.(*Error); ok {
		// Convert evaluator.Error class to perrors.ErrorClass to check catchability
		perrClass := perrors.ErrorClass(err.Class)
		if perrClass.IsCatchable() {
			// Wrap in {result: null, error: <message>} dictionary
			pairs := make(map[string]ast.Expression)
			pairs["result"] = &ast.ObjectLiteralExpression{Obj: NULL}
			pairs["error"] = &ast.ObjectLiteralExpression{Obj: &String{Value: err.Message}}
			return &Dictionary{
				Pairs:    pairs,
				KeyOrder: []string{"result", "error"},
				Env:      env,
			}
		}
		// Non-catchable error - propagate unchanged
		return err
	}

	// Success - wrap in {result: <value>, error: null} dictionary
	pairs := make(map[string]ast.Expression)
	pairs["result"] = &ast.ObjectLiteralExpression{Obj: result}
	pairs["error"] = &ast.ObjectLiteralExpression{Obj: NULL}
	return &Dictionary{
		Pairs:    pairs,
		KeyOrder: []string{"result", "error"},
		Env:      env,
	}
}
