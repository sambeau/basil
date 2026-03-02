// Package evaluator provides datetime method implementations via declarative registry.
// This file implements datetime methods migrating from switch-based dispatch (FEAT-137 Tier 2).
package evaluator

import (
	"encoding/json"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
)

// DatetimeMethodRegistry defines all methods available on datetime values.
var DatetimeMethodRegistry MethodRegistry

func init() {
	DatetimeMethodRegistry = MethodRegistry{
		"toDict":    {Fn: datetimeToDict, Arity: "0", Description: "Convert to plain dictionary (without __type marker)"},
		"inspect":   {Fn: datetimeInspect, Arity: "0", Description: "Return full dictionary including __type for debugging"},
		"fmt":       {Fn: datetimeFmt, Arity: "0-2", Description: "Format with style and/or locale"},
		"format":    {Fn: datetimeFmt, Arity: "0-2", Description: "Format with style and/or locale (alias for fmt)"},
		"short":     {Fn: datetimeShort, Arity: "0-1", Description: "Compact date format"},
		"medium":    {Fn: datetimeMedium, Arity: "0-1", Description: "Standard date format"},
		"long":      {Fn: datetimeLong, Arity: "0-1", Description: "Verbose date format"},
		"full":      {Fn: datetimeFull, Arity: "0-1", Description: "Maximum context date format"},
		"repr":      {Fn: datetimeRepr, Arity: "0", Description: "Get PLN literal representation"},
		"dayOfYear": {Fn: datetimeDayOfYear, Arity: "0", Description: "Day of year (1-366)"},
		"week":      {Fn: datetimeWeek, Arity: "0", Description: "ISO week number"},
		"timestamp": {Fn: datetimeTimestamp, Arity: "0", Description: "Unix timestamp"},
		"toJSON":    {Fn: datetimeToJSON, Arity: "0", Description: "Convert to ISO 8601 JSON string"},
		"toBox":     {Fn: datetimeToBoxMethod, Arity: "0+", Description: "Render as box diagram"},
	}
	RegisterMethodRegistry("datetime", DatetimeMethodRegistry)
}

func datetimeToDict(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	cleanPairs := make(map[string]ast.Expression)
	for key, val := range dict.Pairs {
		if key != "__type" {
			cleanPairs[key] = val
		}
	}
	return &Dictionary{Pairs: cleanPairs, Env: dict.Env}
}

func datetimeInspect(receiver Object, args []Object, env *Environment) Object {
	return receiver
}

func datetimeFmt(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	style := "medium"
	localeStr := "en-US"
	if len(args) >= 1 {
		switch arg := args[0].(type) {
		case *String:
			style = arg.Value
		case *Dictionary:
			if v := getDictStringValue(arg, "style"); v != "" {
				style = v
			}
			if v := getDictStringValue(arg, "locale"); v != "" {
				localeStr = v
			}
		default:
			return newTypeError("TYPE-0005", "fmt", "a string (style) or dictionary", args[0].Type())
		}
	}
	if len(args) == 2 {
		loc, ok := args[1].(*String)
		if !ok {
			return newTypeError("TYPE-0006", "fmt", "a string (locale)", args[1].Type())
		}
		localeStr = loc.Value
	}
	return formatDateWithStyleAndLocale(dict, style, localeStr, env)
}

func datetimeLocaleArg(methodName string, args []Object) (string, Object) {
	localeStr := "en-US"
	if len(args) == 1 {
		switch arg := args[0].(type) {
		case *String:
			localeStr = arg.Value
		case *Dictionary:
			if v := getDictStringValue(arg, "locale"); v != "" {
				localeStr = v
			}
		default:
			return "", newTypeError("TYPE-0012", methodName, "a string (locale) or dictionary", args[0].Type())
		}
	}
	return localeStr, nil
}

func datetimeShort(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	loc, err := datetimeLocaleArg("short", args)
	if err != nil {
		return err
	}
	return formatDateWithStyleAndLocale(dict, "short", loc, env)
}

func datetimeMedium(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	loc, err := datetimeLocaleArg("medium", args)
	if err != nil {
		return err
	}
	return formatDateWithStyleAndLocale(dict, "medium", loc, env)
}

func datetimeLong(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	loc, err := datetimeLocaleArg("long", args)
	if err != nil {
		return err
	}
	return formatDateWithStyleAndLocale(dict, "long", loc, env)
}

func datetimeFull(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	loc, err := datetimeLocaleArg("full", args)
	if err != nil {
		return err
	}
	return formatDateWithStyleAndLocale(dict, "full", loc, env)
}

func datetimeRepr(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	return &String{Value: datetimeToReprString(dict)}
}

func datetimeDayOfYear(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	return evalDatetimeComputedProperty(dict, "dayOfYear", env)
}

func datetimeWeek(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	return evalDatetimeComputedProperty(dict, "week", env)
}

func datetimeTimestamp(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	return evalDatetimeComputedProperty(dict, "timestamp", env)
}

func datetimeToJSON(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	isoStr := strings.TrimPrefix(datetimeToReprString(dict), "@")
	jsonBytes, _ := json.Marshal(isoStr)
	return &String{Value: string(jsonBytes)}
}

func datetimeToBoxMethod(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	return datetimeToBox(dict, args, env)
}
