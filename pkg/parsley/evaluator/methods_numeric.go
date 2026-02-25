// Package evaluator provides numeric method implementations via declarative registry.
// This file implements integer and float methods for FEAT-111: Declarative Method Registry
// Updated for FEAT-121: Unified Formatter API
package evaluator

import (
	"fmt"
	"math"
	"strconv"

	"github.com/sambeau/basil/pkg/parsley/ast"
)

// IntegerMethodRegistry defines all methods available on integer values.
// This is the single source of truth for integer method dispatch and introspection.
var IntegerMethodRegistry MethodRegistry

// FloatMethodRegistry defines all methods available on float values.
// This is the single source of truth for float method dispatch and introspection.
var FloatMethodRegistry MethodRegistry

func init() {
	IntegerMethodRegistry = MethodRegistry{
		"abs": {
			Fn:          integerAbs,
			Arity:       "0",
			Description: "Absolute value",
		},
		"fmt": {
			Fn:          integerFmt,
			Arity:       "0-2",
			Description: "Format with style and/or locale",
		},
		"format": {
			Fn:          integerFmt,
			Arity:       "0-2",
			Description: "Format with style and/or locale (alias for fmt)",
		},
		"short": {
			Fn:          integerShort,
			Arity:       "0-1",
			Description: "Compact format (1K, 1M)",
		},
		"medium": {
			Fn:          integerMedium,
			Arity:       "0-1",
			Description: "Standard format with separators",
		},
		"long": {
			Fn:          integerLong,
			Arity:       "0-1",
			Description: "Full precision format",
		},
		"currency": {
			Fn:          integerCurrency,
			Arity:       "1-2",
			Description: "Format as currency (code, locale?)",
		},
		"percent": {
			Fn:          integerPercent,
			Arity:       "0-1",
			Description: "Format as percentage",
		},
		"humanize": {
			Fn:          integerHumanize,
			Arity:       "0-1",
			Description: "Human-readable format (1K, 1M)",
		},
		"toBox": {
			Fn:          integerToBox,
			Arity:       "0",
			Description: "Render as box diagram",
		},
		"repr": {
			Fn:          integerRepr,
			Arity:       "0",
			Description: "Get representation string",
		},
		"toJSON": {
			Fn:          integerToJSON,
			Arity:       "0",
			Description: "Convert to JSON string",
		},
		"inspect": {
			Fn:          integerInspect,
			Arity:       "0",
			Description: "Get debug dictionary with type info",
		},
	}
	RegisterMethodRegistry("integer", IntegerMethodRegistry)

	FloatMethodRegistry = MethodRegistry{
		"abs": {
			Fn:          floatAbs,
			Arity:       "0",
			Description: "Absolute value",
		},
		"round": {
			Fn:          floatRound,
			Arity:       "0-1",
			Description: "Round to n decimals",
		},
		"floor": {
			Fn:          floatFloor,
			Arity:       "0",
			Description: "Round down",
		},
		"ceil": {
			Fn:          floatCeil,
			Arity:       "0",
			Description: "Round up",
		},
		"fmt": {
			Fn:          floatFmt,
			Arity:       "0-2",
			Description: "Format with style and/or locale",
		},
		"format": {
			Fn:          floatFmt,
			Arity:       "0-2",
			Description: "Format with style and/or locale (alias for fmt)",
		},
		"short": {
			Fn:          floatShort,
			Arity:       "0-1",
			Description: "Compact format (1K, 1M)",
		},
		"medium": {
			Fn:          floatMedium,
			Arity:       "0-1",
			Description: "Standard format with separators",
		},
		"long": {
			Fn:          floatLong,
			Arity:       "0-1",
			Description: "Full precision format",
		},
		"currency": {
			Fn:          floatCurrency,
			Arity:       "1-2",
			Description: "Format as currency (code, locale?)",
		},
		"percent": {
			Fn:          floatPercent,
			Arity:       "0-1",
			Description: "Format as percentage",
		},
		"humanize": {
			Fn:          floatHumanize,
			Arity:       "0-1",
			Description: "Human-readable format (1K, 1M)",
		},
		"toBox": {
			Fn:          floatToBox,
			Arity:       "0",
			Description: "Render as box diagram",
		},
		"repr": {
			Fn:          floatRepr,
			Arity:       "0",
			Description: "Get representation string",
		},
		"toJSON": {
			Fn:          floatToJSON,
			Arity:       "0",
			Description: "Convert to JSON string",
		},
		"inspect": {
			Fn:          floatInspect,
			Arity:       "0",
			Description: "Get debug dictionary with type info",
		},
	}
	RegisterMethodRegistry("float", FloatMethodRegistry)
}

// Integer method implementations

func integerAbs(receiver Object, args []Object, env *Environment) Object {
	num := receiver.(*Integer)
	value := num.Value
	if value < 0 {
		value = -value
	}
	return &Integer{Value: value}
}

// integerFmt formats an integer with style and locale options.
// Overloads:
//   - .fmt() - medium style, default locale
//   - .fmt(n) - precision (ignored for integers, accepted for API consistency)
//   - .fmt("style") - named style
//   - .fmt("style", "locale") - style + locale
//   - .fmt({...}) - options dictionary
func integerFmt(receiver Object, args []Object, env *Environment) Object {
	num := receiver.(*Integer)
	opts, err := parseFormatArgs(args)
	if err != nil {
		return err
	}
	return formatIntegerWithOpts(num.Value, opts)
}

func integerShort(receiver Object, args []Object, env *Environment) Object {
	num := receiver.(*Integer)
	opts, err := parseStyleMethodArgs("short", args, "short")
	if err != nil {
		return err
	}
	return formatIntegerWithOpts(num.Value, opts)
}

func integerMedium(receiver Object, args []Object, env *Environment) Object {
	num := receiver.(*Integer)
	opts, err := parseStyleMethodArgs("medium", args, "medium")
	if err != nil {
		return err
	}
	return formatIntegerWithOpts(num.Value, opts)
}

func integerLong(receiver Object, args []Object, env *Environment) Object {
	num := receiver.(*Integer)
	opts, err := parseStyleMethodArgs("long", args, "long")
	if err != nil {
		return err
	}
	return formatIntegerWithOpts(num.Value, opts)
}

// formatIntegerWithOpts formats an integer using FormatOpts.
func formatIntegerWithOpts(value int64, opts FormatOpts) Object {
	switch opts.Style {
	case "short":
		return &String{Value: humanizeNumber(float64(value), opts.Locale)}
	case "medium":
		return formatNumberWithLocale(float64(value), opts.Locale)
	case "long":
		// Long format: full number with locale separators
		return formatNumberWithLocale(float64(value), opts.Locale)
	default:
		// Default to medium
		return formatNumberWithLocale(float64(value), opts.Locale)
	}
}

func integerCurrency(receiver Object, args []Object, env *Environment) Object {
	num := receiver.(*Integer)
	code, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0005", "currency", "a string (currency code)", args[0].Type())
	}
	localeStr := "en-US"
	if len(args) == 2 {
		loc, ok := args[1].(*String)
		if !ok {
			return newTypeError("TYPE-0006", "currency", "a string (locale)", args[1].Type())
		}
		localeStr = loc.Value
	}
	return formatCurrencyWithLocale(float64(num.Value), code.Value, localeStr)
}

func integerPercent(receiver Object, args []Object, env *Environment) Object {
	num := receiver.(*Integer)
	localeStr := "en-US"
	if len(args) == 1 {
		loc, ok := args[0].(*String)
		if !ok {
			return newTypeError("TYPE-0012", "percent", "a string", args[0].Type())
		}
		localeStr = loc.Value
	}
	return formatPercentWithLocale(float64(num.Value), localeStr)
}

func integerHumanize(receiver Object, args []Object, env *Environment) Object {
	num := receiver.(*Integer)
	localeStr := "en-US"
	if len(args) == 1 {
		loc, ok := args[0].(*String)
		if !ok {
			return newTypeError("TYPE-0012", "humanize", "a string", args[0].Type())
		}
		localeStr = loc.Value
	}
	return &String{Value: humanizeNumber(float64(num.Value), localeStr)}
}

func integerToBox(receiver Object, args []Object, env *Environment) Object {
	num := receiver.(*Integer)
	br := NewBoxRenderer()
	return &String{Value: br.RenderSingleValue(num.Inspect())}
}

func integerRepr(receiver Object, args []Object, env *Environment) Object {
	num := receiver.(*Integer)
	return &String{Value: objectToReprString(num)}
}

func integerToJSON(receiver Object, args []Object, env *Environment) Object {
	num := receiver.(*Integer)
	return &String{Value: strconv.FormatInt(num.Value, 10)}
}

func integerInspect(receiver Object, args []Object, env *Environment) Object {
	num := receiver.(*Integer)
	return &Dictionary{
		Pairs: map[string]ast.Expression{
			"__type": createLiteralExpression(&String{Value: "integer"}),
			"value":  createLiteralExpression(&Integer{Value: num.Value}),
		},
		Env: NewEnvironment(),
	}
}

// Float method implementations

func floatAbs(receiver Object, args []Object, env *Environment) Object {
	num := receiver.(*Float)
	return &Float{Value: math.Abs(num.Value)}
}

func floatRound(receiver Object, args []Object, env *Environment) Object {
	num := receiver.(*Float)
	decimals := 0
	if len(args) == 1 {
		d, ok := args[0].(*Integer)
		if !ok {
			return newTypeError("TYPE-0012", "round", "an integer", args[0].Type())
		}
		decimals = int(d.Value)
	}
	multiplier := math.Pow(10, float64(decimals))
	return &Float{Value: math.Round(num.Value*multiplier) / multiplier}
}

func floatFloor(receiver Object, args []Object, env *Environment) Object {
	num := receiver.(*Float)
	return &Float{Value: math.Floor(num.Value)}
}

func floatCeil(receiver Object, args []Object, env *Environment) Object {
	num := receiver.(*Float)
	return &Float{Value: math.Ceil(num.Value)}
}

// floatFmt formats a float with style and locale options.
// Overloads:
//   - .fmt() - medium style, default locale
//   - .fmt(n) - precision (decimal places)
//   - .fmt("style") - named style
//   - .fmt("style", "locale") - style + locale
//   - .fmt({...}) - options dictionary
func floatFmt(receiver Object, args []Object, env *Environment) Object {
	num := receiver.(*Float)
	opts, err := parseFormatArgs(args)
	if err != nil {
		return err
	}
	return formatFloatWithOpts(num.Value, opts)
}

func floatShort(receiver Object, args []Object, env *Environment) Object {
	num := receiver.(*Float)
	opts, err := parseStyleMethodArgs("short", args, "short")
	if err != nil {
		return err
	}
	return formatFloatWithOpts(num.Value, opts)
}

func floatMedium(receiver Object, args []Object, env *Environment) Object {
	num := receiver.(*Float)
	opts, err := parseStyleMethodArgs("medium", args, "medium")
	if err != nil {
		return err
	}
	return formatFloatWithOpts(num.Value, opts)
}

func floatLong(receiver Object, args []Object, env *Environment) Object {
	num := receiver.(*Float)
	opts, err := parseStyleMethodArgs("long", args, "long")
	if err != nil {
		return err
	}
	return formatFloatWithOpts(num.Value, opts)
}

// formatFloatWithOpts formats a float using FormatOpts.
func formatFloatWithOpts(value float64, opts FormatOpts) Object {
	switch opts.Style {
	case "short":
		return &String{Value: humanizeNumber(value, opts.Locale)}
	case "medium":
		if opts.Precision >= 0 {
			return formatNumberWithPrecisionAndLocale(value, opts.Precision, opts.Locale)
		}
		return formatNumberWithLocale(value, opts.Locale)
	case "long":
		// Long format: show more decimal places
		precision := opts.Precision
		if precision < 0 {
			precision = 2 // Default to 2 decimal places for long
		}
		return formatNumberWithPrecisionAndLocale(value, precision, opts.Locale)
	default:
		if opts.Precision >= 0 {
			return formatNumberWithPrecisionAndLocale(value, opts.Precision, opts.Locale)
		}
		return formatNumberWithLocale(value, opts.Locale)
	}
}

// formatNumberWithPrecisionAndLocale formats a number with specific decimal places.
func formatNumberWithPrecisionAndLocale(value float64, precision int, locale string) Object {
	// Round to precision
	multiplier := math.Pow(10, float64(precision))
	rounded := math.Round(value*multiplier) / multiplier

	// Format with locale
	result := formatNumberWithLocale(rounded, locale)
	if s, ok := result.(*String); ok {
		// If precision specified, ensure we have exactly that many decimal places
		if precision > 0 {
			formatted := s.Value
			// Check if there's a decimal point
			decSep := "."
			if locale == "de-DE" || locale == "fr-FR" || locale == "es-ES" {
				decSep = ","
			}
			if idx := lastIndexOf(formatted, decSep); idx >= 0 {
				// Count decimal places
				decimals := len(formatted) - idx - 1
				if decimals < precision {
					// Pad with zeros
					for i := decimals; i < precision; i++ {
						formatted += "0"
					}
				}
			} else {
				// No decimal point, add one with zeros
				formatted += decSep
				for i := 0; i < precision; i++ {
					formatted += "0"
				}
			}
			return &String{Value: formatted}
		}
	}
	return result
}

func lastIndexOf(s, substr string) int {
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func floatCurrency(receiver Object, args []Object, env *Environment) Object {
	num := receiver.(*Float)
	code, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0005", "currency", "a string (currency code)", args[0].Type())
	}
	localeStr := "en-US"
	if len(args) == 2 {
		loc, ok := args[1].(*String)
		if !ok {
			return newTypeError("TYPE-0006", "currency", "a string (locale)", args[1].Type())
		}
		localeStr = loc.Value
	}
	return formatCurrencyWithLocale(num.Value, code.Value, localeStr)
}

func floatPercent(receiver Object, args []Object, env *Environment) Object {
	num := receiver.(*Float)
	localeStr := "en-US"
	if len(args) == 1 {
		loc, ok := args[0].(*String)
		if !ok {
			return newTypeError("TYPE-0012", "percent", "a string", args[0].Type())
		}
		localeStr = loc.Value
	}
	return formatPercentWithLocale(num.Value, localeStr)
}

func floatHumanize(receiver Object, args []Object, env *Environment) Object {
	num := receiver.(*Float)
	localeStr := "en-US"
	if len(args) == 1 {
		loc, ok := args[0].(*String)
		if !ok {
			return newTypeError("TYPE-0012", "humanize", "a string", args[0].Type())
		}
		localeStr = loc.Value
	}
	return &String{Value: humanizeNumber(num.Value, localeStr)}
}

func floatToBox(receiver Object, args []Object, env *Environment) Object {
	num := receiver.(*Float)
	br := NewBoxRenderer()
	return &String{Value: br.RenderSingleValue(num.Inspect())}
}

func floatRepr(receiver Object, args []Object, env *Environment) Object {
	num := receiver.(*Float)
	return &String{Value: objectToReprString(num)}
}

func floatToJSON(receiver Object, args []Object, env *Environment) Object {
	num := receiver.(*Float)
	return &String{Value: fmt.Sprintf("%g", num.Value)}
}

func floatInspect(receiver Object, args []Object, env *Environment) Object {
	num := receiver.(*Float)
	return &Dictionary{
		Pairs: map[string]ast.Expression{
			"__type": createLiteralExpression(&String{Value: "float"}),
			"value":  createLiteralExpression(&Float{Value: num.Value}),
		},
		Env: NewEnvironment(),
	}
}
