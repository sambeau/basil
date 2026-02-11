// Package evaluator provides string method implementations via declarative registry.
// This file implements string methods for FEAT-111: Declarative Method Registry
package evaluator

import (
	"encoding/json"
	"html"
	"net/url"
	"strings"
)

// StringMethodRegistry defines all methods available on string values.
// This is the single source of truth for string method dispatch and introspection.
// Initialized in init() to avoid initialization cycle.
var StringMethodRegistry MethodRegistry

func init() {
	StringMethodRegistry = MethodRegistry{
		"toUpper": {
			Fn:          stringToUpper,
			Arity:       "0",
			Description: "Convert to uppercase",
		},
		"toLower": {
			Fn:          stringToLower,
			Arity:       "0",
			Description: "Convert to lowercase",
		},
		"toTitle": {
			Fn:          stringToTitle,
			Arity:       "0",
			Description: "Convert to title case (capitalize first letter of each word)",
		},
		"trim": {
			Fn:          stringTrim,
			Arity:       "0",
			Description: "Remove leading/trailing whitespace",
		},
		"split": {
			Fn:          stringSplit,
			Arity:       "1",
			Description: "Split by delimiter into array",
		},
		"replace": {
			Fn:          stringReplace,
			Arity:       "2",
			Description: "Replace all occurrences",
		},
		"length": {
			Fn:          stringLength,
			Arity:       "0",
			Description: "Get character count",
		},
		"includes": {
			Fn:          stringIncludes,
			Arity:       "1",
			Description: "Check if contains substring",
		},
		"highlight": {
			Fn:          stringHighlight,
			Arity:       "1-2",
			Description: "Wrap matches in HTML tag",
		},
		"paragraphs": {
			Fn:          stringParagraphs,
			Arity:       "0",
			Description: "Convert blank lines to <p> tags",
		},
		"render": {
			Fn:          stringRender,
			Arity:       "0-1",
			Description: "Interpolate template with values",
		},
		"parseMarkdown": {
			Fn:          stringParseMarkdown,
			Arity:       "0-1",
			Description: "Parse markdown to {html, raw, md}",
		},
		"parseJSON": {
			Fn:          stringParseJSON,
			Arity:       "0",
			Description: "Parse as JSON",
		},
		"parseCSV": {
			Fn:          stringParseCSV,
			Arity:       "0-1",
			Description: "Parse as CSV",
		},
		"collapse": {
			Fn:          stringCollapse,
			Arity:       "0",
			Description: "Collapse whitespace to single spaces",
		},
		"normalizeSpace": {
			Fn:          stringNormalizeSpace,
			Arity:       "0",
			Description: "Collapse and trim whitespace",
		},
		"stripSpace": {
			Fn:          stringStripSpace,
			Arity:       "0",
			Description: "Remove all whitespace",
		},
		"stripHtml": {
			Fn:          stringStripHtml,
			Arity:       "0",
			Description: "Remove HTML tags",
		},
		"digits": {
			Fn:          stringDigits,
			Arity:       "0",
			Description: "Extract only digits",
		},
		"slug": {
			Fn:          stringSlug,
			Arity:       "0",
			Description: "Convert to URL-safe slug",
		},
		"htmlEncode": {
			Fn:          stringHtmlEncode,
			Arity:       "0",
			Description: "Encode HTML entities (<, >, &, etc.)",
		},
		"htmlDecode": {
			Fn:          stringHtmlDecode,
			Arity:       "0",
			Description: "Decode HTML entities",
		},
		"urlEncode": {
			Fn:          stringUrlEncode,
			Arity:       "0",
			Description: "URL encode (spaces become +)",
		},
		"urlDecode": {
			Fn:          stringUrlDecode,
			Arity:       "0",
			Description: "Decode URL-encoded string",
		},
		"urlPathEncode": {
			Fn:          stringUrlPathEncode,
			Arity:       "0",
			Description: "Encode URL path segment (/ becomes %2F)",
		},
		"urlQueryEncode": {
			Fn:          stringUrlQueryEncode,
			Arity:       "0",
			Description: "Encode URL query value (& and = encoded)",
		},
		"outdent": {
			Fn:          stringOutdent,
			Arity:       "0",
			Description: "Remove common leading whitespace from all lines",
		},
		"indent": {
			Fn:          stringIndent,
			Arity:       "1",
			Description: "Add spaces to beginning of all non-blank lines",
		},
		"toBox": {
			Fn:          stringToBox,
			Arity:       "0",
			Description: "Render as box diagram",
		},
		"repr": {
			Fn:          stringRepr,
			Arity:       "0",
			Description: "Get representation string",
		},
		"toJSON": {
			Fn:          stringToJSON,
			Arity:       "0",
			Description: "Convert to JSON string",
		},
	}
	RegisterMethodRegistry("string", StringMethodRegistry)
}

// String method implementations

func stringToUpper(receiver Object, args []Object, env *Environment) Object {
	str := receiver.(*String)
	return &String{Value: strings.ToUpper(str.Value)}
}

func stringToLower(receiver Object, args []Object, env *Environment) Object {
	str := receiver.(*String)
	return &String{Value: strings.ToLower(str.Value)}
}

func stringToTitle(receiver Object, args []Object, env *Environment) Object {
	str := receiver.(*String)
	return &String{Value: strings.Title(str.Value)}
}

func stringTrim(receiver Object, args []Object, env *Environment) Object {
	str := receiver.(*String)
	return &String{Value: strings.TrimSpace(str.Value)}
}

func stringSplit(receiver Object, args []Object, env *Environment) Object {
	str := receiver.(*String)
	delim, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0012", "split", "a string", args[0].Type())
	}
	parts := strings.Split(str.Value, delim.Value)
	elements := make([]Object, len(parts))
	for i, part := range parts {
		elements[i] = &String{Value: part}
	}
	return &Array{Elements: elements}
}

func stringReplace(receiver Object, args []Object, env *Environment) Object {
	str := receiver.(*String)

	switch search := args[0].(type) {
	case *String:
		// String search - replace all occurrences
		switch replacement := args[1].(type) {
		case *String:
			// Simple string replacement
			return &String{Value: strings.ReplaceAll(str.Value, search.Value, replacement.Value)}
		case *Function:
			// Functional replacement - call function for each match
			return stringReplaceWithFunction(str.Value, search.Value, replacement, env)
		default:
			return newTypeError("TYPE-0006", "replace", "a string or function", args[1].Type())
		}

	case *Dictionary:
		// Check if it's a regex
		if !isRegexDict(search) {
			return newTypeError("TYPE-0005", "replace", "a string or regex", "dictionary")
		}
		// Regex replacement
		return regexReplaceOnString(str.Value, search, args[1], env)

	default:
		return newTypeError("TYPE-0005", "replace", "a string or regex", args[0].Type())
	}
}

func stringLength(receiver Object, args []Object, env *Environment) Object {
	str := receiver.(*String)
	// Return rune count for proper Unicode support
	return &Integer{Value: int64(len([]rune(str.Value)))}
}

func stringIncludes(receiver Object, args []Object, env *Environment) Object {
	str := receiver.(*String)
	substr, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0012", "includes", "a string", args[0].Type())
	}
	if strings.Contains(str.Value, substr.Value) {
		return TRUE
	}
	return FALSE
}

func stringHighlight(receiver Object, args []Object, env *Environment) Object {
	str := receiver.(*String)
	phrase, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0012", "highlight", "a string", args[0].Type())
	}
	tag := "mark" // default tag
	if len(args) == 2 {
		tagArg, ok := args[1].(*String)
		if !ok {
			return newTypeError("TYPE-0013", "highlight", "a string", args[1].Type())
		}
		tag = tagArg.Value
	}
	return &String{Value: highlightString(str.Value, phrase.Value, tag)}
}

func stringParagraphs(receiver Object, args []Object, env *Environment) Object {
	str := receiver.(*String)
	return &String{Value: textToParagraphs(str.Value)}
}

func stringRender(receiver Object, args []Object, env *Environment) Object {
	str := receiver.(*String)

	if env == nil {
		env = NewEnvironment()
	}

	renderEnv := env
	if len(args) == 1 {
		dict, ok := args[0].(*Dictionary)
		if !ok {
			return newTypeError("TYPE-0012", "render", "a dictionary", args[0].Type())
		}

		var errObj Object
		renderEnv, errObj = buildRenderEnv(env, dict)
		if errObj != nil {
			return errObj
		}
	}

	return interpolateRawString(str.Value, renderEnv)
}

func stringParseMarkdown(receiver Object, args []Object, env *Environment) Object {
	str := receiver.(*String)

	var options *Dictionary
	if len(args) == 1 {
		optDict, ok := args[0].(*Dictionary)
		if !ok {
			return newTypeError("TYPE-0012", "parseMarkdown", "a dictionary", args[0].Type())
		}
		options = optDict
	}

	result, err := parseMarkdown(str.Value, options, env)
	if err != nil {
		return err
	}
	return result
}

func stringParseJSON(receiver Object, args []Object, env *Environment) Object {
	str := receiver.(*String)
	result, err := parseJSON(str.Value)
	if err != nil {
		return err
	}
	return result
}

func stringParseCSV(receiver Object, args []Object, env *Environment) Object {
	str := receiver.(*String)
	hasHeader := true
	if len(args) == 1 {
		flag, ok := args[0].(*Boolean)
		if !ok {
			return newTypeError("TYPE-0004", "parseCSV", "a boolean", args[0].Type())
		}
		hasHeader = flag.Value
	}

	result, err := parseCSV([]byte(str.Value), hasHeader)
	if err != nil {
		return err
	}
	return result
}

func stringCollapse(receiver Object, args []Object, env *Environment) Object {
	str := receiver.(*String)
	return &String{Value: whitespaceRegex.ReplaceAllString(str.Value, " ")}
}

func stringNormalizeSpace(receiver Object, args []Object, env *Environment) Object {
	str := receiver.(*String)
	collapsed := whitespaceRegex.ReplaceAllString(str.Value, " ")
	return &String{Value: strings.TrimSpace(collapsed)}
}

func stringStripSpace(receiver Object, args []Object, env *Environment) Object {
	str := receiver.(*String)
	return &String{Value: whitespaceRegex.ReplaceAllString(str.Value, "")}
}

func stringStripHtml(receiver Object, args []Object, env *Environment) Object {
	str := receiver.(*String)
	stripped := htmlTagRegex.ReplaceAllString(str.Value, "")
	return &String{Value: html.UnescapeString(stripped)}
}

func stringDigits(receiver Object, args []Object, env *Environment) Object {
	str := receiver.(*String)
	return &String{Value: nonDigitRegex.ReplaceAllString(str.Value, "")}
}

func stringSlug(receiver Object, args []Object, env *Environment) Object {
	str := receiver.(*String)
	lower := strings.ToLower(str.Value)
	return &String{Value: strings.Trim(nonSlugRegex.ReplaceAllString(lower, "-"), "-")}
}

func stringHtmlEncode(receiver Object, args []Object, env *Environment) Object {
	str := receiver.(*String)
	return &String{Value: html.EscapeString(str.Value)}
}

func stringHtmlDecode(receiver Object, args []Object, env *Environment) Object {
	str := receiver.(*String)
	return &String{Value: html.UnescapeString(str.Value)}
}

func stringUrlEncode(receiver Object, args []Object, env *Environment) Object {
	str := receiver.(*String)
	// QueryEscape uses + for spaces (application/x-www-form-urlencoded)
	return &String{Value: url.QueryEscape(str.Value)}
}

func stringUrlDecode(receiver Object, args []Object, env *Environment) Object {
	str := receiver.(*String)
	decoded, err := url.QueryUnescape(str.Value)
	if err != nil {
		return newFormatError("FMT-0011", err)
	}
	return &String{Value: decoded}
}

func stringUrlPathEncode(receiver Object, args []Object, env *Environment) Object {
	str := receiver.(*String)
	// PathEscape encodes path segments (including /)
	return &String{Value: url.PathEscape(str.Value)}
}

func stringUrlQueryEncode(receiver Object, args []Object, env *Environment) Object {
	str := receiver.(*String)
	// QueryEscape encodes query values (& and = are encoded)
	return &String{Value: url.QueryEscape(str.Value)}
}

func stringOutdent(receiver Object, args []Object, env *Environment) Object {
	str := receiver.(*String)
	return &String{Value: outdentString(str.Value)}
}

func stringIndent(receiver Object, args []Object, env *Environment) Object {
	str := receiver.(*String)
	spaces, ok := args[0].(*Integer)
	if !ok {
		return newTypeError("TYPE-0012", "indent", "an integer", args[0].Type())
	}
	return &String{Value: indentString(str.Value, int(spaces.Value))}
}

func stringToBox(receiver Object, args []Object, env *Environment) Object {
	str := receiver.(*String)
	br := NewBoxRenderer()
	return &String{Value: br.RenderSingleValue(str.Value)}
}

func stringRepr(receiver Object, args []Object, env *Environment) Object {
	str := receiver.(*String)
	return &String{Value: objectToReprString(str)}
}

func stringToJSON(receiver Object, args []Object, env *Environment) Object {
	str := receiver.(*String)
	// JSON encode the string
	jsonBytes, _ := json.Marshal(str.Value)
	return &String{Value: string(jsonBytes)}
}
