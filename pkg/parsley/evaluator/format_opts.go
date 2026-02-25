// Package evaluator provides format options helpers for FEAT-121: Unified Formatter API.
// This file defines shared types and functions used by methods_numeric.go, methods_money.go,
// methods_unit.go, and methods.go for consistent formatting across all value types.
package evaluator

// FormatOpts contains formatting options for value types.
type FormatOpts struct {
	Style     string // "short", "medium", "long", "full"
	Locale    string // BCP 47 locale code (e.g., "en-US", "de-DE")
	Precision int    // Decimal places (-1 means use default)
	Compound  bool   // Use compound format for units (e.g., feet-inches)
}

// DefaultFormatOpts returns the default formatting options.
func DefaultFormatOpts() FormatOpts {
	return FormatOpts{
		Style:     "medium",
		Locale:    "en-US",
		Precision: -1,
		Compound:  false,
	}
}

// isStyleName returns true if the string is a valid style name.
func isStyleName(s string) bool {
	switch s {
	case "short", "medium", "long", "full":
		return true
	default:
		return false
	}
}

// parseFormatArgs parses arguments for .fmt() method calls.
// Supports:
//   - .fmt() - no args, use defaults
//   - .fmt(n) - integer precision
//   - .fmt("style") - named style
//   - .fmt("style", "locale") - style + locale
//   - .fmt({...}) - options dictionary
func parseFormatArgs(args []Object) (FormatOpts, Object) {
	opts := DefaultFormatOpts()

	switch len(args) {
	case 0:
		return opts, nil

	case 1:
		switch arg := args[0].(type) {
		case *Integer:
			opts.Precision = int(arg.Value)
			return opts, nil
		case *String:
			if isStyleName(arg.Value) {
				opts.Style = arg.Value
			} else {
				// Assume it's a locale
				opts.Locale = arg.Value
			}
			return opts, nil
		case *Dictionary:
			return parseFormatOptsFromDict(arg)
		default:
			return opts, newTypeError("TYPE-0012", "fmt", "an integer, string, or dictionary", args[0].Type())
		}

	case 2:
		// .fmt("style", "locale")
		style, ok := args[0].(*String)
		if !ok {
			return opts, newTypeError("TYPE-0005", "fmt", "a string (style)", args[0].Type())
		}
		locale, ok := args[1].(*String)
		if !ok {
			return opts, newTypeError("TYPE-0006", "fmt", "a string (locale)", args[1].Type())
		}
		opts.Style = style.Value
		opts.Locale = locale.Value
		return opts, nil

	default:
		return opts, newArityErrorRange("fmt", len(args), 0, 2)
	}
}

// parseStyleMethodArgs parses arguments for style methods like .short(), .long(), etc.
// Supports:
//   - .short() - no args
//   - .short("locale") - locale string
//   - .short({...}) - options dictionary
func parseStyleMethodArgs(methodName string, args []Object, style string) (FormatOpts, Object) {
	opts := DefaultFormatOpts()
	opts.Style = style

	switch len(args) {
	case 0:
		return opts, nil

	case 1:
		switch arg := args[0].(type) {
		case *String:
			opts.Locale = arg.Value
			return opts, nil
		case *Dictionary:
			parsed, err := parseFormatOptsFromDict(arg)
			if err != nil {
				return opts, err
			}
			// Style method overrides style from dict
			parsed.Style = style
			return parsed, nil
		default:
			return opts, newTypeError("TYPE-0012", methodName, "a string (locale) or dictionary", args[0].Type())
		}

	default:
		return opts, newArityErrorRange(methodName, len(args), 0, 1)
	}
}

// parseFormatOptsFromDict extracts FormatOpts from a dictionary.
// Dictionary stores AST expressions, so we need to evaluate them.
// Uses the existing evalDictValue helper from stdlib_table.go.
func parseFormatOptsFromDict(dict *Dictionary) (FormatOpts, Object) {
	opts := DefaultFormatOpts()

	// Use evalDictValue(dict, key, env) from stdlib_table.go
	if s, ok := evalDictValue(dict, "style", dict.Env).(*String); ok {
		opts.Style = s.Value
	}

	if s, ok := evalDictValue(dict, "locale", dict.Env).(*String); ok {
		opts.Locale = s.Value
	}

	if n, ok := evalDictValue(dict, "precision", dict.Env).(*Integer); ok {
		opts.Precision = int(n.Value)
	}

	if b, ok := evalDictValue(dict, "compound", dict.Env).(*Boolean); ok {
		opts.Compound = b.Value
	}

	return opts, nil
}
