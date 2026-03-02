// Package evaluator provides file, dir, request, and response method implementations via declarative registry.
// This file implements these types migrating from switch-based dispatch (FEAT-137 Tier 2).
package evaluator

import (
	"os"

	"github.com/sambeau/basil/pkg/parsley/ast"
)

// FileMethodRegistry defines all methods available on file values.
var FileMethodRegistry MethodRegistry

// DirMethodRegistry defines all methods available on dir values.
var DirMethodRegistry MethodRegistry

// RequestMethodRegistry defines all methods available on request values.
var RequestMethodRegistry MethodRegistry

// ResponseMethodRegistry defines all methods available on response values.
var ResponseMethodRegistry MethodRegistry

func init() {
	FileMethodRegistry = MethodRegistry{
		"toDict":  {Fn: fileToDictMethod, Arity: "0", Description: "Convert to plain dictionary (without __type marker)"},
		"inspect": {Fn: fileInspect, Arity: "0", Description: "Return full dictionary including __type for debugging"},
		"remove":  {Fn: fileRemove, Arity: "0", Description: "Delete the file from filesystem"},
		"mkdir":   {Fn: fileMkdir, Arity: "0-1", Description: "Create directory at this path"},
		"rmdir":   {Fn: fileRmdir, Arity: "0-1", Description: "Remove directory at this path"},
	}
	RegisterMethodRegistry("file", FileMethodRegistry)

	DirMethodRegistry = MethodRegistry{
		"toDict":  {Fn: dirToDictMethod, Arity: "0", Description: "Convert to plain dictionary (without __type marker)"},
		"inspect": {Fn: dirInspect, Arity: "0", Description: "Return full dictionary including __type for debugging"},
		"mkdir":   {Fn: dirMkdir, Arity: "0-1", Description: "Create directory"},
		"rmdir":   {Fn: dirRmdir, Arity: "0-1", Description: "Remove directory"},
	}
	RegisterMethodRegistry("dir", DirMethodRegistry)

	RequestMethodRegistry = MethodRegistry{
		"toDict": {Fn: requestToDictMethod, Arity: "0", Description: "Convert to raw dictionary for debugging"},
	}
	RegisterMethodRegistry("request", RequestMethodRegistry)

	ResponseMethodRegistry = MethodRegistry{
		"response": {Fn: responseResponse, Arity: "0", Description: "Get __response metadata dictionary"},
		"format":   {Fn: responseFormat, Arity: "0", Description: "Get format string (json, text, etc.)"},
		"data":     {Fn: responseData, Arity: "0", Description: "Get __data value"},
		"toDict":   {Fn: responseToDict, Arity: "0", Description: "Convert to raw dictionary for debugging"},
	}
	RegisterMethodRegistry("response", ResponseMethodRegistry)
}

// ============================================================================
// File method implementations
// ============================================================================

func fileToDictMethod(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	cleanPairs := make(map[string]ast.Expression)
	for key, val := range dict.Pairs {
		if key != "__type" {
			cleanPairs[key] = val
		}
	}
	return &Dictionary{Pairs: cleanPairs, Env: dict.Env}
}

func fileInspect(receiver Object, args []Object, env *Environment) Object {
	return receiver
}

func fileRemove(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	return evalFileRemove(dict, env)
}

func fileMkdir(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	return mkdirFromDict(dict, args, env, "file")
}

func fileRmdir(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	return rmdirFromDict(dict, args, env, "file")
}

// ============================================================================
// Dir method implementations
// ============================================================================

func dirToDictMethod(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	cleanPairs := make(map[string]ast.Expression)
	for key, val := range dict.Pairs {
		if key != "__type" {
			cleanPairs[key] = val
		}
	}
	return &Dictionary{Pairs: cleanPairs, Env: dict.Env}
}

func dirInspect(receiver Object, args []Object, env *Environment) Object {
	return receiver
}

func dirMkdir(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	return mkdirFromDict(dict, args, env, "directory")
}

func dirRmdir(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	return rmdirFromDict(dict, args, env, "directory")
}

// ============================================================================
// Shared filesystem helpers
// ============================================================================

func mkdirFromDict(dict *Dictionary, args []Object, env *Environment, typeLabel string) Object {
	pathStr := getFilePathString(dict, env)
	if pathStr == "" {
		return newValidationError("VAL-0008", map[string]any{"Type": typeLabel})
	}
	absPath, pathErr := resolveModulePath(pathStr, env.Filename, env.RootPath)
	if pathErr != nil {
		return newIOError("IO-0007", pathStr, pathErr)
	}
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
	if err := env.checkPathAccess(absPath, "write"); err != nil {
		return newSecurityError("write", err)
	}
	var err error
	if recursive {
		err = os.MkdirAll(absPath, 0755)
	} else {
		err = os.Mkdir(absPath, 0755)
	}
	if err != nil {
		return newIOError("IO-0006", absPath, err)
	}
	return NULL
}

func rmdirFromDict(dict *Dictionary, args []Object, env *Environment, typeLabel string) Object {
	pathStr := getFilePathString(dict, env)
	if pathStr == "" {
		return newValidationError("VAL-0008", map[string]any{"Type": typeLabel})
	}
	absPath, pathErr := resolveModulePath(pathStr, env.Filename, env.RootPath)
	if pathErr != nil {
		return newIOError("IO-0007", pathStr, pathErr)
	}
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
	if err := env.checkPathAccess(absPath, "write"); err != nil {
		return newSecurityError("write", err)
	}
	var err error
	if recursive {
		err = os.RemoveAll(absPath)
	} else {
		err = os.Remove(absPath)
	}
	if err != nil {
		return newIOError("IO-0010", absPath, err)
	}
	return NULL
}

// ============================================================================
// Request method implementations
// ============================================================================

func requestToDictMethod(receiver Object, args []Object, env *Environment) Object {
	return receiver
}

// ============================================================================
// Response method implementations
// ============================================================================

func responseResponse(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	if responseExpr, ok := dict.Pairs["__response"]; ok {
		return Eval(responseExpr, dict.Env)
	}
	return NULL
}

func responseFormat(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	if formatExpr, ok := dict.Pairs["__format"]; ok {
		return Eval(formatExpr, dict.Env)
	}
	return NULL
}

func responseData(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	if dataExpr, ok := dict.Pairs["__data"]; ok {
		return Eval(dataExpr, dict.Env)
	}
	return NULL
}

func responseToDict(receiver Object, args []Object, env *Environment) Object {
	return receiver
}
