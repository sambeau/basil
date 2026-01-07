// eval_paths.go - Path manipulation and conversion functions for the Parsley evaluator
//
// This file contains functions for parsing, cleaning, and converting file paths between
// string and dictionary representations. Includes support for absolute/relative paths,
// path normalization (Rob Pike's cleanname algorithm), and stdio paths (stdin/stdout/stderr).

package evaluator

import (
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/lexer"
)

// cleanPathComponents implements Rob Pike's cleanname algorithm from Plan 9
// to canonicalize path components. This ensures paths always present clean file names.
// See: https://9p.io/sys/doc/lexnames.html
//
// Rules:
// 1. Reduce multiple slashes to a single slash (handled by parsePathString)
// 2. Eliminate . path name elements (the current directory)
// 3. Eliminate .. elements and the non-. non-.. element that precedes them
// 4. Eliminate .. elements that begin a rooted path (replace /.. by /)
// 5. Leave intact .. elements that begin a non-rooted path
//
// Note: For absolute paths, we prepend an empty string to represent the root.
// This is the traditional Unix convention: /usr/local → ["", "usr", "local"]
func cleanPathComponents(components []string, isAbsolute bool) []string {
	var result []string

	for _, comp := range components {
		switch comp {
		case "":
			// Skip empty components (multiple slashes already handled)
			continue
		case ".":
			// Rule 2: Eliminate . (current directory)
			continue
		case "..":
			if len(result) > 0 && result[len(result)-1] != ".." {
				// Rule 3: Eliminate .. and the preceding element
				result = result[:len(result)-1]
			} else if isAbsolute {
				// Rule 4: Eliminate .. at the beginning of rooted paths
				// (do nothing, effectively replacing /.. with /)
			} else {
				// Rule 5: Leave .. intact at the beginning of non-rooted paths
				result = append(result, comp)
			}
		default:
			result = append(result, comp)
		}
	}

	// If result is empty, return current directory for relative paths
	// For absolute paths with no components (just "/"), return empty slice
	// The absolute flag will be used during reconstruction to add leading "/"
	if len(result) == 0 && !isAbsolute {
		return []string{"."} // Current directory
	}

	return result
}

// parsePathString parses a file path string into components
// Returns components array and whether path is absolute
// The path is cleaned using Rob Pike's cleanname algorithm
func parsePathString(pathStr string) ([]string, bool) {
	if pathStr == "" {
		return []string{"."}, false
	}

	// Detect absolute vs relative
	isAbsolute := false
	hasLeadingDot := false
	if pathStr[0] == '/' {
		isAbsolute = true
	} else if len(pathStr) >= 2 && pathStr[1] == ':' {
		// Windows drive letter (C:, D:, etc.)
		isAbsolute = true
	} else if pathStr[0] == '.' && (len(pathStr) == 1 || pathStr[1] == '/') {
		// Starts with ./ - remember this for output
		hasLeadingDot = true
	} else if pathStr[0] == '~' {
		// Home directory reference - treat specially
		hasLeadingDot = false
	}

	// Split on forward slashes (handle both Unix and Windows)
	pathStr = strings.ReplaceAll(pathStr, "\\", "/")
	parts := strings.Split(pathStr, "/")

	// Collect raw components
	components := []string{}
	for _, part := range parts {
		if part != "" {
			components = append(components, part)
		}
	}

	// Clean the path components
	cleaned := cleanPathComponents(components, isAbsolute)

	// For relative paths that originally started with ./, preserve that style
	// unless the cleaned result already starts with . or ..
	if hasLeadingDot && len(cleaned) > 0 && cleaned[0] != "." && cleaned[0] != ".." {
		cleaned = append([]string{"."}, cleaned...)
	}

	return cleaned, isAbsolute
}

// pathToDict creates a path dictionary from components
func pathToDict(components []string, isAbsolute bool, env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)

	// Add __type field
	pairs["__type"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: "path"},
		Value: "path",
	}

	// Add components as array literal
	componentExprs := make([]ast.Expression, len(components))
	for i, comp := range components {
		componentExprs[i] = &ast.StringLiteral{
			Token: lexer.Token{Type: lexer.STRING, Literal: comp},
			Value: comp,
		}
	}
	pairs["segments"] = &ast.ArrayLiteral{
		Token:    lexer.Token{Type: lexer.LBRACKET, Literal: "["},
		Elements: componentExprs,
	}

	// Add absolute flag
	tokenType := lexer.FALSE
	tokenLiteral := "false"
	if isAbsolute {
		tokenType = lexer.TRUE
		tokenLiteral = "true"
	}
	pairs["absolute"] = &ast.Boolean{
		Token: lexer.Token{Type: tokenType, Literal: tokenLiteral},
		Value: isAbsolute,
	}

	return &Dictionary{Pairs: pairs, Env: env}
}

// stdioToDict creates a path dictionary for stdin/stdout/stderr
func stdioToDict(stream string, env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)

	// Add __type field
	pairs["__type"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: "path"},
		Value: "path",
	}

	// Add __stdio field to mark this as a stdio path
	pairs["__stdio"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: stream},
		Value: stream,
	}

	// Add components as array with just "-"
	pairs["segments"] = &ast.ArrayLiteral{
		Token: lexer.Token{Type: lexer.LBRACKET, Literal: "["},
		Elements: []ast.Expression{
			&ast.StringLiteral{
				Token: lexer.Token{Type: lexer.STRING, Literal: "-"},
				Value: "-",
			},
		},
	}

	// Not absolute
	pairs["absolute"] = &ast.Boolean{
		Token: lexer.Token{Type: lexer.FALSE, Literal: "false"},
		Value: false,
	}

	// Add path property as "-" for display
	pairs["path"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: "-"},
		Value: "-",
	}

	// Add name property
	pairs["name"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: stream},
		Value: stream,
	}

	return &Dictionary{Pairs: pairs, Env: env}
}

// fileToDict creates a file dictionary from a path and format
// format can be: "json", "csv", "lines", "text", "bytes", or "" for auto-detect
func fileToDict(pathDict *Dictionary, format string, options *Dictionary, env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)

	// Add __type field
	pairs["__type"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: "file"},
		Value: "file",
	}

	// Add path field (the original path dictionary)
	// Store the path components and absolute flag from the path dict
	if compExpr, ok := pathDict.Pairs["segments"]; ok {
		pairs["_pathComponents"] = compExpr
	}
	if absExpr, ok := pathDict.Pairs["absolute"]; ok {
		pairs["_pathAbsolute"] = absExpr
	}

	// Propagate __stdio marker from path dict (for stdin/stdout/stderr)
	if stdioExpr, ok := pathDict.Pairs["__stdio"]; ok {
		pairs["__stdio"] = stdioExpr
	}

	// Add format field
	pairs["format"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: format},
		Value: format,
	}

	// Add options field (if provided)
	if options != nil {
		// Copy options to ast expressions
		optPairs := make(map[string]ast.Expression)
		for k, v := range options.Pairs {
			optPairs[k] = v
		}
		pairs["options"] = &ast.DictionaryLiteral{
			Token: lexer.Token{Type: lexer.LBRACE, Literal: "{"},
			Pairs: optPairs,
		}
	} else {
		// Empty options
		pairs["options"] = &ast.DictionaryLiteral{
			Token: lexer.Token{Type: lexer.LBRACE, Literal: "{"},
			Pairs: make(map[string]ast.Expression),
		}
	}

	return &Dictionary{Pairs: pairs, Env: env}
}

// dirToDict creates a directory dictionary from a path dictionary
// Directory dictionaries have __type: "dir" and can be read to list contents
func dirToDict(pathDict *Dictionary, env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)

	// Add __type field
	pairs["__type"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: "dir"},
		Value: "dir",
	}

	// Store the path components and absolute flag from the path dict
	if compExpr, ok := pathDict.Pairs["segments"]; ok {
		pairs["_pathComponents"] = compExpr
	}
	if absExpr, ok := pathDict.Pairs["absolute"]; ok {
		pairs["_pathAbsolute"] = absExpr
	}

	return &Dictionary{Pairs: pairs, Env: env}
}
