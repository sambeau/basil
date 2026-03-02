// Package evaluator provides path, url, and regex method implementations via declarative registry.
// This file implements these dict-subtype methods migrating from switch-based dispatch (FEAT-137 Tier 2).
package evaluator

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
)

// PathMethodRegistry defines all methods available on path values.
var PathMethodRegistry MethodRegistry

// UrlMethodRegistry defines all methods available on url values.
var UrlMethodRegistry MethodRegistry

// RegexMethodRegistry defines all methods available on regex values.
var RegexMethodRegistry MethodRegistry

func init() {
	PathMethodRegistry = MethodRegistry{
		"toDict":     {Fn: pathToDictMethod, Arity: "0", Description: "Convert to plain dictionary (without __type marker)"},
		"inspect":    {Fn: pathInspect, Arity: "0", Description: "Return full dictionary including __type for debugging"},
		"isAbsolute": {Fn: pathIsAbsolute, Arity: "0", Description: "Check if absolute path"},
		"isRelative": {Fn: pathIsRelative, Arity: "0", Description: "Check if relative path"},
		"public":     {Fn: pathPublic, Arity: "0", Description: "Get public URL for this path"},
		"toURL":      {Fn: pathToURL, Arity: "1", Description: "Convert to URL string with prefix"},
		"match":      {Fn: pathMatch, Arity: "1", Description: "Match against pattern, returns captures"},
		"toJSON":     {Fn: pathToJSON, Arity: "0", Description: "Convert to JSON string"},
		"toBox":      {Fn: pathToBoxMethod, Arity: "0+", Description: "Render as box diagram"},
		"repr":       {Fn: pathRepr, Arity: "0", Description: "Get PLN literal representation"},
	}
	RegisterMethodRegistry("path", PathMethodRegistry)

	UrlMethodRegistry = MethodRegistry{
		"toDict":   {Fn: urlToDictMethod, Arity: "0", Description: "Convert to plain dictionary (without __type marker)"},
		"inspect":  {Fn: urlInspect, Arity: "0", Description: "Return full dictionary including __type for debugging"},
		"origin":   {Fn: urlOrigin, Arity: "0", Description: "Get origin (scheme://host:port)"},
		"pathname": {Fn: urlPathname, Arity: "0", Description: "Get path component"},
		"search":   {Fn: urlSearch, Arity: "0", Description: "Get query string representation"},
		"href":     {Fn: urlHref, Arity: "0", Description: "Get full URL string"},
		"toJSON":   {Fn: urlToJSON, Arity: "0", Description: "Convert to JSON string"},
		"toBox":    {Fn: urlToBoxMethod, Arity: "0+", Description: "Render as box diagram"},
		"repr":     {Fn: urlRepr, Arity: "0", Description: "Get PLN literal representation"},
	}
	RegisterMethodRegistry("url", UrlMethodRegistry)

	RegexMethodRegistry = MethodRegistry{
		"toDict":  {Fn: regexToDict, Arity: "0", Description: "Convert to plain dictionary (without __type marker)"},
		"inspect": {Fn: regexInspect, Arity: "0", Description: "Return full dictionary including __type for debugging"},
		"format":  {Fn: regexFormat, Arity: "0-1", Description: "Format as string (pattern, literal, verbose)"},
		"test":    {Fn: regexTest, Arity: "1", Description: "Test if string matches pattern"},
		"replace": {Fn: regexReplace, Arity: "2", Description: "Replace matches in string"},
		"toJSON":  {Fn: regexToJSON, Arity: "0", Description: "Convert to JSON object with pattern and flags"},
		"toBox":   {Fn: regexToBoxMethod, Arity: "0+", Description: "Render as box diagram"},
	}
	RegisterMethodRegistry("regex", RegexMethodRegistry)
}

// ============================================================================
// Path method implementations
// ============================================================================

func pathToDictMethod(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	cleanPairs := make(map[string]ast.Expression)
	for key, val := range dict.Pairs {
		if key != "__type" {
			cleanPairs[key] = val
		}
	}
	return &Dictionary{Pairs: cleanPairs, Env: dict.Env}
}

func pathInspect(receiver Object, args []Object, env *Environment) Object {
	return receiver
}

func pathIsAbsolute(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	if absExpr, ok := dict.Pairs["absolute"]; ok {
		result := Eval(absExpr, env)
		if b, ok := result.(*Boolean); ok {
			return b
		}
	}
	return FALSE
}

func pathIsRelative(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	if absExpr, ok := dict.Pairs["absolute"]; ok {
		result := Eval(absExpr, env)
		if b, ok := result.(*Boolean); ok {
			return nativeBoolToParsBoolean(!b.Value)
		}
	}
	return TRUE
}

func pathPublic(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	return evalPublicURL([]Object{dict}, env)
}

func pathToURL(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	prefix, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0012", "toURL", "a string", args[0].Type())
	}
	pathStr := pathDictToString(dict)
	cleanPrefix := strings.TrimRight(prefix.Value, "/")
	relPath := pathStr
	if strings.HasPrefix(relPath, "./") {
		relPath = relPath[1:]
	}
	if !strings.HasPrefix(relPath, "/") {
		relPath = "/" + relPath
	}
	return &String{Value: cleanPrefix + relPath}
}

func pathMatch(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	pattern, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0006", "match", "a string", args[0].Type())
	}
	pathStr := pathDictToString(dict)
	result := matchPathPattern(pathStr, pattern.Value)
	if result == nil {
		return NULL
	}
	pairs := make(map[string]ast.Expression)
	for key, val := range result {
		switch v := val.(type) {
		case string:
			pairs[key] = createLiteralExpression(&String{Value: v})
		case []string:
			elements := make([]Object, len(v))
			for i, s := range v {
				elements[i] = &String{Value: s}
			}
			pairs[key] = createLiteralExpression(&Array{Elements: elements})
		}
	}
	return &Dictionary{Pairs: pairs, Env: NewEnvironment()}
}

func pathToJSON(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	pathStr := pathDictToString(dict)
	jsonBytes, _ := json.Marshal(pathStr)
	return &String{Value: string(jsonBytes)}
}

func pathToBoxMethod(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	return pathToBox(dict, args, env)
}

func pathRepr(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	return &String{Value: "@" + pathDictToString(dict)}
}

// ============================================================================
// URL method implementations
// ============================================================================

func urlToDictMethod(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	cleanPairs := make(map[string]ast.Expression)
	for key, val := range dict.Pairs {
		if key != "__type" {
			cleanPairs[key] = val
		}
	}
	return &Dictionary{Pairs: cleanPairs, Env: dict.Env}
}

func urlInspect(receiver Object, args []Object, env *Environment) Object {
	return receiver
}

func urlOrigin(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	scheme, host, port := "", "", ""
	if schemeExpr, ok := dict.Pairs["scheme"]; ok {
		if s := Eval(schemeExpr, env); s != nil {
			if str, ok := s.(*String); ok {
				scheme = str.Value
			}
		}
	}
	if hostExpr, ok := dict.Pairs["host"]; ok {
		if h := Eval(hostExpr, env); h != nil {
			if str, ok := h.(*String); ok {
				host = str.Value
			}
		}
	}
	if portExpr, ok := dict.Pairs["port"]; ok {
		if p := Eval(portExpr, env); p != nil {
			switch pv := p.(type) {
			case *Integer:
				if pv.Value > 0 {
					port = fmt.Sprintf(":%d", pv.Value)
				}
			case *String:
				if pv.Value != "" {
					port = ":" + pv.Value
				}
			}
		}
	}
	return &String{Value: scheme + "://" + host + port}
}

func urlPathname(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	if pathExpr, ok := dict.Pairs["path"]; ok {
		if p := Eval(pathExpr, env); p != nil {
			if arr, ok := p.(*Array); ok {
				parts := make([]string, 0, len(arr.Elements))
				for _, elem := range arr.Elements {
					if s, ok := elem.(*String); ok && s.Value != "" {
						parts = append(parts, s.Value)
					}
				}
				return &String{Value: "/" + strings.Join(parts, "/")}
			}
		}
	}
	return &String{Value: "/"}
}

func urlSearch(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	if queryExpr, ok := dict.Pairs["query"]; ok {
		if q := Eval(queryExpr, env); q != nil {
			if queryDict, ok := q.(*Dictionary); ok {
				if len(queryDict.Pairs) == 0 {
					return &String{Value: ""}
				}
				parts := make([]string, 0, len(queryDict.Pairs))
				for k, v := range queryDict.Pairs {
					if strings.HasPrefix(k, "__") {
						continue
					}
					val := Eval(v, env)
					parts = append(parts, k+"="+val.Inspect())
				}
				return &String{Value: "?" + strings.Join(parts, "&")}
			}
		}
	}
	return &String{Value: ""}
}

func urlHref(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	return &String{Value: urlDictToString(dict)}
}

func urlToJSON(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	urlStr := urlDictToString(dict)
	jsonBytes, _ := json.Marshal(urlStr)
	return &String{Value: string(jsonBytes)}
}

func urlToBoxMethod(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	return urlToBox(dict, args, env)
}

func urlRepr(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	return &String{Value: `@"` + urlDictToString(dict) + `"`}
}

// ============================================================================
// Regex method implementations
// ============================================================================

func regexToDict(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	cleanPairs := make(map[string]ast.Expression)
	for key, val := range dict.Pairs {
		if key != "__type" {
			cleanPairs[key] = val
		}
	}
	return &Dictionary{Pairs: cleanPairs, Env: dict.Env}
}

func regexInspect(receiver Object, args []Object, env *Environment) Object {
	return receiver
}

func regexFormat(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	var pattern, flags string
	if patternExpr, ok := dict.Pairs["pattern"]; ok {
		if p := Eval(patternExpr, env); p != nil {
			if str, ok := p.(*String); ok {
				pattern = str.Value
			}
		}
	}
	if flagsExpr, ok := dict.Pairs["flags"]; ok {
		if f := Eval(flagsExpr, env); f != nil {
			if str, ok := f.(*String); ok {
				flags = str.Value
			}
		}
	}
	style := "literal"
	if len(args) == 1 {
		styleArg, ok := args[0].(*String)
		if !ok {
			return newTypeError("TYPE-0012", "format", "a string (style)", args[0].Type())
		}
		style = styleArg.Value
	}
	switch style {
	case "pattern":
		return &String{Value: pattern}
	case "literal":
		return &String{Value: "/" + pattern + "/" + flags}
	case "verbose":
		if flags == "" {
			return &String{Value: "pattern: " + pattern}
		}
		return &String{Value: "pattern: " + pattern + ", flags: " + flags}
	default:
		return newValidationError("VAL-0002", map[string]any{"Style": style, "Context": "regex format", "ValidOptions": "pattern, literal, verbose"})
	}
}

func regexTest(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	str, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0012", "test", "a string", args[0].Type())
	}
	var pattern, flags string
	if patternExpr, ok := dict.Pairs["pattern"]; ok {
		if p := Eval(patternExpr, env); p != nil {
			if s, ok := p.(*String); ok {
				pattern = s.Value
			}
		}
	}
	if flagsExpr, ok := dict.Pairs["flags"]; ok {
		if f := Eval(flagsExpr, env); f != nil {
			if s, ok := f.(*String); ok {
				flags = s.Value
			}
		}
	}
	re, err := compileRegex(pattern, flags)
	if err != nil {
		return newFormatError("FMT-0007", err)
	}
	return nativeBoolToParsBoolean(re.MatchString(str.Value))
}

func regexReplace(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	str, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0012", "replace", "a string", args[0].Type())
	}
	return regexReplaceOnString(str.Value, dict, args[1], env)
}

func regexToJSON(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	var pattern, flags string
	if patternExpr, ok := dict.Pairs["pattern"]; ok {
		if p := Eval(patternExpr, env); p != nil {
			if s, ok := p.(*String); ok {
				pattern = s.Value
			}
		}
	}
	if flagsExpr, ok := dict.Pairs["flags"]; ok {
		if f := Eval(flagsExpr, env); f != nil {
			if s, ok := f.(*String); ok {
				flags = s.Value
			}
		}
	}
	patternJSON, _ := json.Marshal(pattern)
	flagsJSON, _ := json.Marshal(flags)
	return &String{Value: fmt.Sprintf(`{"pattern":%s,"flags":%s}`, patternJSON, flagsJSON)}
}

func regexToBoxMethod(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	return regexToBox(dict, args, env)
}
