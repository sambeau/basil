// Package evaluator provides numeric method implementations via declarative registry.
// This file implements integer and float methods for FEAT-111: Declarative Method Registry
package evaluator

import (
	"fmt"
	"math"
	"strconv"
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
		"format": {
			Fn:          integerFormat,
			Arity:       "0-1",
			Description: "Format with locale",
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
		"format": {
			Fn:          floatFormat,
			Arity:       "0-1",
			Description: "Format with locale",
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

func integerFormat(receiver Object, args []Object, env *Environment) Object {
	num := receiver.(*Integer)
	localeStr := "en-US"
	if len(args) == 1 {
		loc, ok := args[0].(*String)
		if !ok {
			return newTypeError("TYPE-0012", "format", "a string", args[0].Type())
		}
		localeStr = loc.Value
	}
	return formatNumberWithLocale(float64(num.Value), localeStr)
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

func floatFormat(receiver Object, args []Object, env *Environment) Object {
	num := receiver.(*Float)
	localeStr := "en-US"
	if len(args) >= 1 {
		loc, ok := args[0].(*String)
		if !ok {
			return newTypeError("TYPE-0012", "format", "a string", args[0].Type())
		}
		localeStr = loc.Value
	}
	return formatNumberWithLocale(num.Value, localeStr)
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
