// eval_regex.go - Regex compilation and matching functions
//
// This file contains functions for:
// - Compiling regex patterns with flags
// - Evaluating regex match expressions
// - Converting regex dictionaries to strings (see eval_dict_to_string.go)

package evaluator

import (
	"regexp"

	"github.com/sambeau/basil/pkg/parsley/lexer"
)

// compileRegex compiles a regex pattern with optional flags
// Go's regexp doesn't support all Perl flags, so we map what we can
// Supported flags: i (case-insensitive), m (multi-line), s (dot matches newline)
// Note: 'g' (global) is handled by match operator, not compilation
func compileRegex(pattern, flags string) (*regexp.Regexp, error) {
	// Process flags - Go regexp supports (?flags) syntax
	prefix := ""
	for _, flag := range flags {
		switch flag {
		case 'i': // case-insensitive
			prefix += "(?i)"
		case 'm': // multi-line (^ and $ match line boundaries)
			prefix += "(?m)"
		case 's': // dot matches newline
			prefix += "(?s)"
			// 'g' (global) is handled by match operator, not compilation
			// Other flags like 'x' (verbose) could be added
		}
	}

	fullPattern := prefix + pattern
	return regexp.Compile(fullPattern)
}

// evalMatchExpression handles string ~ regex matching
// Returns an array of matches (with captures) or null if no match
func evalMatchExpression(tok lexer.Token, text string, regexDict *Dictionary, env *Environment) Object {
	// Extract pattern and flags from regex dictionary
	patternExpr, ok := regexDict.Pairs["pattern"]
	if !ok {
		return newValidationError("VAL-0015", map[string]any{})
	}
	patternObj := Eval(patternExpr, env)
	patternStr, ok := patternObj.(*String)
	if !ok {
		return newValidationError("VAL-0016", map[string]any{"Got": patternObj.Type()})
	}

	flagsExpr, ok := regexDict.Pairs["flags"]
	var flags string
	if ok {
		flagsObj := Eval(flagsExpr, env)
		if flagsStr, ok := flagsObj.(*String); ok {
			flags = flagsStr.Value
		}
	}

	// Compile the regex
	re, err := compileRegex(patternStr.Value, flags)
	if err != nil {
		return newFormatError("FMT-0002", err)
	}

	// Find matches
	matches := re.FindStringSubmatch(text)
	if matches == nil {
		return NULL // No match - returns null (falsy)
	}

	// Convert matches to array of strings
	elements := make([]Object, len(matches))
	for i, match := range matches {
		elements[i] = &String{Value: match}
	}

	return &Array{Elements: elements}
}
