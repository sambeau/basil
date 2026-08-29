// eval_regex.go - Regex compilation and matching functions
//
// This file contains functions for:
// - Compiling regex patterns with flags
// - Evaluating regex match expressions
// - Converting regex dictionaries to strings (see eval_dict_to_string.go)

package evaluator

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/lexer"
)

// compileRegex compiles a regex pattern with optional flags
// Go's regexp doesn't support all Perl flags, so we map what we can
// Supported flags: i (case-insensitive), m (multi-line), s (dot matches newline)
// Note: 'g' (global) is handled by match operator, not compilation
func compileRegex(pattern, flags string) (*regexp.Regexp, error) {
	// Process flags - Go regexp supports (?flags) syntax
	var prefix strings.Builder
	for _, flag := range flags {
		switch flag {
		case 'i': // case-insensitive
			prefix.WriteString("(?i)")
		case 'm': // multi-line (^ and $ match line boundaries)
			prefix.WriteString("(?m)")
		case 's': // dot matches newline
			prefix.WriteString("(?s)")
			// 'g' (global) is handled by match operator, not compilation
			// Other flags like 'x' (verbose) could be added
		}
	}

	fullPattern := prefix.String() + pattern
	return regexp.Compile(fullPattern)
}

// regexDictPatternFlags pulls the pattern and flags strings out of a regex dictionary.
// Missing or non-string fields come back as "".
func regexDictPatternFlags(dict *Dictionary, env *Environment) (pattern, flags string) {
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
	return pattern, flags
}

// compileRegexDict compiles the regex described by a regex dictionary
func compileRegexDict(dict *Dictionary, env *Environment) (*regexp.Regexp, error) {
	pattern, flags := regexDictPatternFlags(dict, env)
	return compileRegex(pattern, flags)
}

// hasNamedGroups checks if the compiled regex has any named capture groups
func hasNamedGroups(re *regexp.Regexp) bool {
	names := re.SubexpNames()
	for i := 1; i < len(names); i++ { // Skip index 0 (full match, always empty string)
		if names[i] != "" {
			return true
		}
	}
	return false
}

// evalMatchExpression handles string ~ regex matching
// Returns an array of matches (with captures) or null if no match
// If the regex has named capture groups, returns a dictionary instead:
//   - Key "0" contains the full match
//   - Named groups use their name as key
//   - Unnamed groups use their numeric index as string key ("1", "2", etc.)
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

	// With the g flag, return every match instead of the first (BUG-027)
	if strings.Contains(flags, "g") {
		allMatches := re.FindAllStringSubmatch(text, -1)
		if allMatches == nil {
			return NULL // No match - returns null (falsy)
		}
		elements := make([]Object, len(allMatches))
		for i, m := range allMatches {
			if hasNamedGroups(re) {
				elements[i] = buildMatchDictionary(re, m)
			} else if len(m) > 1 {
				// Capture groups: one array per match
				groups := make([]Object, len(m))
				for j, g := range m {
					groups[j] = &String{Value: g}
				}
				elements[i] = &Array{Elements: groups}
			} else {
				elements[i] = &String{Value: m[0]}
			}
		}
		return &Array{Elements: elements}
	}

	// Find matches
	matches := re.FindStringSubmatch(text)
	if matches == nil {
		return NULL // No match - returns null (falsy)
	}

	// If regex has named groups, return a dictionary
	if hasNamedGroups(re) {
		return buildMatchDictionary(re, matches)
	}

	// Otherwise, return array for backward compatibility
	elements := make([]Object, len(matches))
	for i, match := range matches {
		elements[i] = &String{Value: match}
	}

	return &Array{Elements: elements}
}

// buildMatchDictionary creates a dictionary from regex matches with named groups
// Key "0" is always the full match
// Named groups get their name as key
// Unnamed groups get their index as string key
func buildMatchDictionary(re *regexp.Regexp, matches []string) *Dictionary {
	names := re.SubexpNames()
	pairs := make(map[string]ast.Expression)
	keyOrder := make([]string, 0, len(matches))

	for i, match := range matches {
		var key string
		switch {
		case i == 0:
			// Full match always gets key "0"
			key = "0"
		case names[i] != "":
			// Named group uses its name
			key = names[i]
		default:
			// Unnamed group uses numeric string key
			key = strconv.Itoa(i)
		}

		pairs[key] = objectToExpression(&String{Value: match})
		keyOrder = append(keyOrder, key)
	}

	return &Dictionary{
		Pairs:    pairs,
		KeyOrder: keyOrder,
	}
}
