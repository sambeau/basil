// eval_urls.go - URL parsing and conversion functions for the Parsley evaluator
//
// This file contains functions for parsing URLs and creating URL/request dictionaries.
// Supports parsing of scheme, host, port, path, query parameters, fragments, and authentication.

package evaluator

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/lexer"
)

// parseUrlString parses a URL string into components
// Supports: scheme://[user:pass@]host[:port]/path?query#fragment
func parseUrlString(urlStr string, env *Environment) (*Dictionary, error) {
	// Simple URL parsing (not using net/url to keep it simple)
	pairs := make(map[string]ast.Expression)

	// Add __type field
	pairs["__type"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: "url"},
		Value: "url",
	}

	// Parse scheme
	before, after, ok := strings.Cut(urlStr, "://")
	if !ok {
		return nil, fmt.Errorf("invalid URL: missing scheme (expected scheme://...)")
	}
	scheme := before
	rest := after

	pairs["scheme"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: scheme},
		Value: scheme,
	}

	// Parse fragment (if present)
	var fragment string
	if fragIdx := strings.Index(rest, "#"); fragIdx != -1 {
		fragment = rest[fragIdx+1:]
		rest = rest[:fragIdx]
	}

	// Parse query (if present)
	queryPairs := make(map[string]ast.Expression)
	if queryIdx := strings.Index(rest, "?"); queryIdx != -1 {
		queryStr := rest[queryIdx+1:]
		rest = rest[:queryIdx]

		// Parse query parameters
		for param := range strings.SplitSeq(queryStr, "&") {
			if param == "" {
				continue
			}
			parts := strings.SplitN(param, "=", 2)
			key := parts[0]
			value := ""
			if len(parts) > 1 {
				value = parts[1]
			}
			queryPairs[key] = &ast.StringLiteral{
				Token: lexer.Token{Type: lexer.STRING, Literal: value},
				Value: value,
			}
		}
	}
	pairs["query"] = &ast.DictionaryLiteral{
		Token: lexer.Token{Type: lexer.LBRACE, Literal: "{"},
		Pairs: queryPairs,
	}

	// Parse path (if present)
	pathComponents := []string{}
	var pathStr string
	if pathIdx := strings.Index(rest, "/"); pathIdx != -1 {
		pathStr = rest[pathIdx:]
		rest = rest[:pathIdx]
		pathComponents, _ = parsePathString(pathStr)
	}

	pathExprs := make([]ast.Expression, len(pathComponents))
	for i, comp := range pathComponents {
		pathExprs[i] = &ast.StringLiteral{
			Token: lexer.Token{Type: lexer.STRING, Literal: comp},
			Value: comp,
		}
	}
	pairs["path"] = &ast.ArrayLiteral{
		Token:    lexer.Token{Type: lexer.LBRACKET, Literal: "["},
		Elements: pathExprs,
	}

	// Parse authority (user:pass@host:port)
	var username, password, host string
	var port int64 = 0

	// Check for userinfo (user:pass@)
	if atIdx := strings.Index(rest, "@"); atIdx != -1 {
		userinfo := rest[:atIdx]
		rest = rest[atIdx+1:]

		if before, after, ok := strings.Cut(userinfo, ":"); ok {
			username = before
			password = after
		} else {
			username = userinfo
		}
	}

	// Parse host:port
	if before, after, ok := strings.Cut(rest, ":"); ok {
		host = before
		portStr := after
		if p, err := strconv.ParseInt(portStr, 10, 64); err == nil {
			port = p
		}
	} else {
		host = rest
	}

	pairs["host"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: host},
		Value: host,
	}

	pairs["port"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: fmt.Sprintf("%d", port)},
		Value: port,
	}

	if username != "" {
		pairs["username"] = &ast.StringLiteral{
			Token: lexer.Token{Type: lexer.STRING, Literal: username},
			Value: username,
		}
	} else {
		pairs["username"] = &ast.Identifier{
			Token: lexer.Token{Type: lexer.IDENT, Literal: "null"},
			Value: "null",
		}
	}

	if password != "" {
		pairs["password"] = &ast.StringLiteral{
			Token: lexer.Token{Type: lexer.STRING, Literal: password},
			Value: password,
		}
	} else {
		pairs["password"] = &ast.Identifier{
			Token: lexer.Token{Type: lexer.IDENT, Literal: "null"},
			Value: "null",
		}
	}

	if fragment != "" {
		pairs["fragment"] = &ast.StringLiteral{
			Token: lexer.Token{Type: lexer.STRING, Literal: fragment},
			Value: fragment,
		}
	} else {
		pairs["fragment"] = &ast.Identifier{
			Token: lexer.Token{Type: lexer.IDENT, Literal: "null"},
			Value: "null",
		}
	}

	return &Dictionary{Pairs: pairs, Env: env}, nil
}

// parseURLToDict wraps parseUrlString and returns nil on error (for convenience)
func parseURLToDict(urlStr string, env *Environment) *Dictionary {
	dict, err := parseUrlString(urlStr, env)
	if err != nil {
		return nil
	}
	return dict
}

// urlToRequestDict wraps a URL dictionary in a request dictionary
func urlToRequestDict(urlDict *Dictionary, format string, options *Dictionary, env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)

	pairs["__type"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: "request"},
		Value: "request",
	}

	// Copy URL fields
	for key, expr := range urlDict.Pairs {
		pairs["_url_"+key] = expr
	}

	pairs["method"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: "GET"},
		Value: "GET",
	}

	pairs["format"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: format},
		Value: format,
	}

	// Add empty headers dict
	pairs["headers"] = &ast.DictionaryLiteral{
		Token: lexer.Token{Type: lexer.LBRACE, Literal: "{"},
		Pairs: make(map[string]ast.Expression),
	}

	return &Dictionary{Pairs: pairs, Env: env}
}

// requestToDict creates a request dictionary from a URL dictionary with format and options
func requestToDict(urlDict *Dictionary, format string, options *Dictionary, env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)

	pairs["__type"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: "request"},
		Value: "request",
	}

	// Copy URL fields with prefix
	for key, expr := range urlDict.Pairs {
		pairs["_url_"+key] = expr
	}

	pairs["format"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: format},
		Value: format,
	}

	// Default method is GET
	method := "GET"
	if options != nil {
		if methodExpr, ok := options.Pairs["method"]; ok {
			methodObj := Eval(methodExpr, env)
			if methodStr, ok := methodObj.(*String); ok {
				method = strings.ToUpper(methodStr.Value)
			}
		}
	}
	pairs["method"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: method},
		Value: method,
	}

	// Copy headers from options
	if options != nil {
		if headersExpr, ok := options.Pairs["headers"]; ok {
			pairs["headers"] = headersExpr
		} else {
			pairs["headers"] = &ast.DictionaryLiteral{
				Token: lexer.Token{Type: lexer.LBRACE, Literal: "{"},
				Pairs: make(map[string]ast.Expression),
			}
		}
		// Copy body from options
		if bodyExpr, ok := options.Pairs["body"]; ok {
			pairs["body"] = bodyExpr
		}
		// Copy timeout from options
		if timeoutExpr, ok := options.Pairs["timeout"]; ok {
			pairs["timeout"] = timeoutExpr
		}
	} else {
		pairs["headers"] = &ast.DictionaryLiteral{
			Token: lexer.Token{Type: lexer.LBRACE, Literal: "{"},
			Pairs: make(map[string]ast.Expression),
		}
	}

	return &Dictionary{Pairs: pairs, Env: env}
}
