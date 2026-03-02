// Package evaluator provides duration method implementations via declarative registry.
// This file implements duration methods migrating from switch-based dispatch (FEAT-137 Tier 2).
package evaluator

import (
	"fmt"

	"github.com/sambeau/basil/pkg/parsley/ast"
)

// DurationMethodRegistry defines all methods available on duration values.
var DurationMethodRegistry MethodRegistry

func init() {
	DurationMethodRegistry = MethodRegistry{
		"toDict":  {Fn: durationToDictMethod, Arity: "0", Description: "Convert to plain dictionary (without __type marker)"},
		"inspect": {Fn: durationInspect, Arity: "0", Description: "Return full dictionary including __type for debugging"},
		"fmt":     {Fn: durationFmt, Arity: "0-2", Description: "Format with style and/or locale"},
		"format":  {Fn: durationFmt, Arity: "0-2", Description: "Format with style and/or locale (alias for fmt)"},
		"short":   {Fn: durationShort, Arity: "0-1", Description: "Compact duration format (2h)"},
		"medium":  {Fn: durationMedium, Arity: "0-1", Description: "Standard duration format (2 hours)"},
		"long":    {Fn: durationLong, Arity: "0-1", Description: "Verbose duration format (2 hours 30 minutes)"},
		"full":    {Fn: durationFull, Arity: "0", Description: "Not supported for duration (returns error)"},
		"repr":    {Fn: durationRepr, Arity: "0", Description: "Get PLN literal representation"},
		"toJSON":  {Fn: durationToJSON, Arity: "0", Description: "Convert to JSON object with components"},
		"toBox":   {Fn: durationToBoxMethod, Arity: "0+", Description: "Render as box diagram"},
	}
	RegisterMethodRegistry("duration", DurationMethodRegistry)
}

func durationToDictMethod(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	cleanPairs := make(map[string]ast.Expression)
	for key, val := range dict.Pairs {
		if key != "__type" {
			cleanPairs[key] = val
		}
	}
	return &Dictionary{Pairs: cleanPairs, Env: dict.Env}
}

func durationInspect(receiver Object, args []Object, env *Environment) Object {
	return receiver
}

func durationFmt(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	style := "medium"
	localeStr := "en-US"
	if len(args) >= 1 {
		switch arg := args[0].(type) {
		case *String:
			if isStyleName(arg.Value) {
				style = arg.Value
			} else {
				localeStr = arg.Value
			}
		case *Dictionary:
			if v := getDictStringValue(arg, "style"); v != "" {
				style = v
			}
			if v := getDictStringValue(arg, "locale"); v != "" {
				localeStr = v
			}
		default:
			return newTypeError("TYPE-0012", "fmt", "a string or dictionary", args[0].Type())
		}
	}
	if len(args) == 2 {
		loc, ok := args[1].(*String)
		if !ok {
			return newTypeError("TYPE-0006", "fmt", "a string (locale)", args[1].Type())
		}
		localeStr = loc.Value
	}
	return formatDurationWithStyle(dict, style, localeStr, env)
}

func durationLocaleArg(methodName string, args []Object) (string, Object) {
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

func durationShort(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	loc, err := durationLocaleArg("short", args)
	if err != nil {
		return err
	}
	return formatDurationWithStyle(dict, "short", loc, env)
}

func durationMedium(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	loc, err := durationLocaleArg("medium", args)
	if err != nil {
		return err
	}
	return formatDurationWithStyle(dict, "medium", loc, env)
}

func durationLong(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	loc, err := durationLocaleArg("long", args)
	if err != nil {
		return err
	}
	return formatDurationWithStyle(dict, "long", loc, env)
}

func durationFull(receiver Object, args []Object, env *Environment) Object {
	return newStructuredError("VAL-0021", map[string]any{
		"Function": "full",
		"Expected": "Duration does not support 'full' style",
		"Got":      "full",
	})
}

func durationRepr(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	return &String{Value: formatDurationRepr(dict, env)}
}

func durationToJSON(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	months, seconds, err := getDurationComponents(dict, env)
	if err != nil {
		return newValidationError("VAL-0007", map[string]any{"GoError": err.Error()})
	}
	years := months / 12
	months %= 12
	days := seconds / (24 * 3600)
	seconds %= (24 * 3600)
	hours := seconds / 3600
	seconds %= 3600
	minutes := seconds / 60
	seconds %= 60
	result := fmt.Sprintf(`{"years":%d,"months":%d,"days":%d,"hours":%d,"minutes":%d,"seconds":%d}`,
		years, months, days, hours, minutes, seconds)
	return &String{Value: result}
}

func durationToBoxMethod(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	return durationToBox(dict, args, env)
}
