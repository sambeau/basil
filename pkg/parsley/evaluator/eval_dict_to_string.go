// eval_dict_to_string.go - Dictionary-to-string conversion functions for the Parsley evaluator
//
// This file contains functions that convert specialized dictionary types (regex, path, file,
// directory, URL, request, tag) back to their string representations.

package evaluator

import (
	"strconv"
	"strings"
)

// regexDictToString converts a regex dictionary to its literal form /pattern/flags
func regexDictToString(dict *Dictionary) string {
	var pattern, flags string

	if patternExpr, ok := dict.Pairs["pattern"]; ok {
		patternObj := Eval(patternExpr, dict.Env)
		if str, ok := patternObj.(*String); ok {
			pattern = str.Value
		}
	}

	if flagsExpr, ok := dict.Pairs["flags"]; ok {
		flagsObj := Eval(flagsExpr, dict.Env)
		if str, ok := flagsObj.(*String); ok {
			flags = str.Value
		}
	}

	return "/" + pattern + "/" + flags
}

// fileDictToString converts a file dictionary to its path string
func fileDictToString(dict *Dictionary) string {
	// Extract path components from the file dict
	var components []string
	var isAbsolute bool

	if compExpr, ok := dict.Pairs["_pathComponents"]; ok {
		compObj := Eval(compExpr, dict.Env)
		if arr, ok := compObj.(*Array); ok {
			for _, elem := range arr.Elements {
				if str, ok := elem.(*String); ok {
					components = append(components, str.Value)
				}
			}
		}
	}

	if absExpr, ok := dict.Pairs["_pathAbsolute"]; ok {
		absObj := Eval(absExpr, dict.Env)
		if b, ok := absObj.(*Boolean); ok {
			isAbsolute = b.Value
		}
	}

	// Build path string - use same logic as pathDictToString
	if len(components) == 0 {
		if isAbsolute {
			return "/"
		}
		return "."
	}

	result := strings.Join(components, "/")
	if isAbsolute {
		return "/" + result
	}
	return result
}

// dirDictToString converts a directory dictionary to its path string (with trailing slash)
func dirDictToString(dict *Dictionary) string {
	// Extract path components from the dir dict
	var components []string
	var isAbsolute bool

	if compExpr, ok := dict.Pairs["_pathComponents"]; ok {
		compObj := Eval(compExpr, dict.Env)
		if arr, ok := compObj.(*Array); ok {
			for _, elem := range arr.Elements {
				if str, ok := elem.(*String); ok {
					components = append(components, str.Value)
				}
			}
		}
	}

	if absExpr, ok := dict.Pairs["_pathAbsolute"]; ok {
		absObj := Eval(absExpr, dict.Env)
		if b, ok := absObj.(*Boolean); ok {
			isAbsolute = b.Value
		}
	}

	// Build path string - use same logic as pathDictToString
	var pathStr string
	if len(components) == 0 {
		if isAbsolute {
			pathStr = "/"
		} else {
			pathStr = "./"
		}
	} else {
		result := strings.Join(components, "/")
		if isAbsolute {
			pathStr = "/" + result
		} else {
			pathStr = result
		}
	}

	// Add trailing slash for directories
	if !strings.HasSuffix(pathStr, "/") {
		pathStr += "/"
	}

	return pathStr
}

// requestDictToString converts a request dictionary to METHOD URL format
func requestDictToString(dict *Dictionary) string {
	var method, urlStr string

	// Get method (default to GET)
	method = "GET"
	if methodExpr, ok := dict.Pairs["method"]; ok {
		methodObj := Eval(methodExpr, dict.Env)
		if str, ok := methodObj.(*String); ok {
			method = str.Value
		}
	}

	// Reconstruct URL from _url_* fields
	var result strings.Builder

	// Scheme
	if schemeExpr, ok := dict.Pairs["_url_scheme"]; ok {
		schemeObj := Eval(schemeExpr, dict.Env)
		if str, ok := schemeObj.(*String); ok {
			result.WriteString(str.Value)
			result.WriteString("://")
		}
	}

	// Host
	if hostExpr, ok := dict.Pairs["_url_host"]; ok {
		hostObj := Eval(hostExpr, dict.Env)
		if str, ok := hostObj.(*String); ok {
			result.WriteString(str.Value)
		}
	}

	// Port
	if portExpr, ok := dict.Pairs["_url_port"]; ok {
		portObj := Eval(portExpr, dict.Env)
		if i, ok := portObj.(*Integer); ok && i.Value != 0 {
			result.WriteString(":")
			result.WriteString(strconv.FormatInt(i.Value, 10))
		}
	}

	// Path
	if pathExpr, ok := dict.Pairs["_url_path"]; ok {
		pathObj := Eval(pathExpr, dict.Env)
		if arr, ok := pathObj.(*Array); ok && len(arr.Elements) > 0 {
			startIdx := 0
			if str, ok := arr.Elements[0].(*String); ok && str.Value == "" {
				result.WriteString("/")
				startIdx = 1
			} else if len(arr.Elements) > 0 {
				result.WriteString("/")
			}
			for i := startIdx; i < len(arr.Elements); i++ {
				if str, ok := arr.Elements[i].(*String); ok && str.Value != "" {
					if i > startIdx {
						result.WriteString("/")
					}
					result.WriteString(str.Value)
				}
			}
		}
	}

	// Query
	if queryExpr, ok := dict.Pairs["_url_query"]; ok {
		queryObj := Eval(queryExpr, dict.Env)
		if queryDict, ok := queryObj.(*Dictionary); ok && len(queryDict.Pairs) > 0 {
			result.WriteString("?")
			first := true
			for key, expr := range queryDict.Pairs {
				if !first {
					result.WriteString("&")
				}
				first = false
				result.WriteString(key)
				result.WriteString("=")
				valObj := Eval(expr, dict.Env)
				if str, ok := valObj.(*String); ok {
					result.WriteString(str.Value)
				}
			}
		}
	}

	urlStr = result.String()
	return method + " " + urlStr
}

// tagDictToString converts a tag dictionary back to an HTML string
func tagDictToString(dict *Dictionary) string {
	var result strings.Builder

	// Get tag name
	nameExpr, ok := dict.Pairs["name"]
	if !ok {
		return dict.Inspect() // Fallback if not a proper tag dict
	}
	nameObj := Eval(nameExpr, dict.Env)
	nameStr, ok := nameObj.(*String)
	if !ok {
		return dict.Inspect()
	}

	// Get contents
	contentsExpr, hasContents := dict.Pairs["contents"]
	var contentsObj Object
	if hasContents {
		contentsObj = Eval(contentsExpr, dict.Env)
	}

	// Get attributes
	attrsExpr, hasAttrs := dict.Pairs["attrs"]
	var attrsDict *Dictionary
	if hasAttrs {
		attrsObj := Eval(attrsExpr, dict.Env)
		if d, ok := attrsObj.(*Dictionary); ok {
			attrsDict = d
		}
	}

	// Check if self-closing (no contents)
	isSelfClosing := contentsObj == nil || contentsObj == NULL

	// Build the opening tag
	result.WriteByte('<')
	result.WriteString(nameStr.Value)

	// Add attributes
	if attrsDict != nil && len(attrsDict.Pairs) > 0 {
		for key, expr := range attrsDict.Pairs {
			result.WriteByte(' ')
			result.WriteString(key)
			result.WriteString(`="`)
			val := Eval(expr, attrsDict.Env)
			result.WriteString(objectToPrintString(val))
			result.WriteByte('"')
		}
	}

	if isSelfClosing {
		result.WriteString(" />")
	} else {
		result.WriteByte('>')

		// Add contents
		switch c := contentsObj.(type) {
		case *String:
			result.WriteString(c.Value)
		case *Array:
			for _, elem := range c.Elements {
				result.WriteString(objectToPrintString(elem))
			}
		default:
			result.WriteString(objectToPrintString(contentsObj))
		}

		// Closing tag
		result.WriteString("</")
		result.WriteString(nameStr.Value)
		result.WriteByte('>')
	}

	return result.String()
}

// pathDictToString converts a path dictionary back to a string
func pathDictToString(dict *Dictionary) string {
	// Get components array
	componentsExpr, ok := dict.Pairs["segments"]
	if !ok {
		return ""
	}

	// Evaluate the array expression
	componentsObj := Eval(componentsExpr, dict.Env)
	arr, ok := componentsObj.(*Array)
	if !ok {
		return ""
	}

	// Get absolute flag
	isAbsolute := false
	if absExpr, ok := dict.Pairs["absolute"]; ok {
		absObj := Eval(absExpr, dict.Env)
		if b, ok := absObj.(*Boolean); ok {
			isAbsolute = b.Value
		}
	}

	// Build path string from components
	var parts []string
	for _, elem := range arr.Elements {
		if str, ok := elem.(*String); ok {
			parts = append(parts, str.Value)
		}
	}

	if len(parts) == 0 {
		if isAbsolute {
			return "/"
		}
		return "."
	}

	// Join components and add leading / for absolute paths
	result := strings.Join(parts, "/")
	if isAbsolute {
		return "/" + result
	}
	return result
}

// urlDictToString converts a URL dictionary back to a string
func urlDictToString(dict *Dictionary) string {
	var result strings.Builder

	// Scheme
	if schemeExpr, ok := dict.Pairs["scheme"]; ok {
		schemeObj := Eval(schemeExpr, dict.Env)
		if str, ok := schemeObj.(*String); ok {
			result.WriteString(str.Value)
			result.WriteString("://")
		}
	}

	// Username and password
	if usernameExpr, ok := dict.Pairs["username"]; ok {
		usernameObj := Eval(usernameExpr, dict.Env)
		if str, ok := usernameObj.(*String); ok && str.Value != "" {
			result.WriteString(str.Value)

			if passwordExpr, ok := dict.Pairs["password"]; ok {
				passwordObj := Eval(passwordExpr, dict.Env)
				if pstr, ok := passwordObj.(*String); ok && pstr.Value != "" {
					result.WriteString(":")
					result.WriteString(pstr.Value)
				}
			}
			result.WriteString("@")
		}
	}

	// Host
	if hostExpr, ok := dict.Pairs["host"]; ok {
		hostObj := Eval(hostExpr, dict.Env)
		if str, ok := hostObj.(*String); ok {
			result.WriteString(str.Value)
		}
	}

	// Port (if non-zero)
	if portExpr, ok := dict.Pairs["port"]; ok {
		portObj := Eval(portExpr, dict.Env)
		if i, ok := portObj.(*Integer); ok && i.Value != 0 {
			result.WriteString(":")
			result.WriteString(strconv.FormatInt(i.Value, 10))
		}
	}

	// Path
	if pathExpr, ok := dict.Pairs["path"]; ok {
		pathObj := Eval(pathExpr, dict.Env)
		if arr, ok := pathObj.(*Array); ok && len(arr.Elements) > 0 {
			// Check if first element is empty string (indicates leading slash)
			startIdx := 0
			if str, ok := arr.Elements[0].(*String); ok && str.Value == "" {
				// Leading empty string means path starts with /
				result.WriteString("/")
				startIdx = 1
			} else if len(arr.Elements) > 0 {
				// No leading empty, but we have segments, so add /
				result.WriteString("/")
			}

			// Add remaining path segments
			for i := startIdx; i < len(arr.Elements); i++ {
				if str, ok := arr.Elements[i].(*String); ok && str.Value != "" {
					if i > startIdx {
						result.WriteString("/")
					}
					result.WriteString(str.Value)
				}
			}
		}
	}

	// Query
	if queryExpr, ok := dict.Pairs["query"]; ok {
		queryObj := Eval(queryExpr, dict.Env)
		if queryDict, ok := queryObj.(*Dictionary); ok && len(queryDict.Pairs) > 0 {
			result.WriteString("?")
			first := true
			for key, expr := range queryDict.Pairs {
				if !first {
					result.WriteString("&")
				}
				first = false
				result.WriteString(key)
				result.WriteString("=")
				valObj := Eval(expr, dict.Env)
				if str, ok := valObj.(*String); ok {
					result.WriteString(str.Value)
				}
			}
		}
	}

	// Fragment
	if fragmentExpr, ok := dict.Pairs["fragment"]; ok {
		fragmentObj := Eval(fragmentExpr, dict.Env)
		if str, ok := fragmentObj.(*String); ok && str.Value != "" {
			result.WriteString("#")
			result.WriteString(str.Value)
		}
	}

	return result.String()
}

// fileDictToLiteral converts a file dictionary to a Parsley literal (@/path/to/file)
func fileDictToLiteral(dict *Dictionary) string {
	return "@" + fileDictToString(dict)
}

// dirDictToLiteral converts a dir dictionary to a Parsley literal (@/path/to/dir)
func dirDictToLiteral(dict *Dictionary) string {
	return "@" + dirDictToString(dict)
}
