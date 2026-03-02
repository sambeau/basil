// Package evaluator provides dictionary method implementations via declarative registry.
// This file implements dictionary methods migrating from switch-based dispatch (FEAT-137 Tier 1).
package evaluator

import (
	"encoding/json"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
)

// DictionaryMethodRegistry defines all methods available on dictionary values.
// This is the single source of truth for dictionary method dispatch and introspection.
var DictionaryMethodRegistry MethodRegistry

func init() {
	DictionaryMethodRegistry = MethodRegistry{
		"keys": {
			Fn:          dictKeys,
			Arity:       "0",
			Description: "Get all keys",
		},
		"values": {
			Fn:          dictValues,
			Arity:       "0",
			Description: "Get all values",
		},
		"entries": {
			Fn:          dictEntries,
			Arity:       "0+",
			Description: "Get entries as array of {key, value} dicts (keyName?, valueName?)",
		},
		"has": {
			Fn:          dictHas,
			Arity:       "1",
			Description: "Check if key exists",
		},
		"delete": {
			Fn:          dictDelete,
			Arity:       "1",
			Description: "Remove key",
		},
		"insertAfter": {
			Fn:          dictInsertAfter,
			Arity:       "3",
			Description: "Insert key-value pair after existing key",
		},
		"insertBefore": {
			Fn:          dictInsertBefore,
			Arity:       "3",
			Description: "Insert key-value pair before existing key",
		},
		"render": {
			Fn:          dictRender,
			Arity:       "1",
			Description: "Render template string with dictionary values",
		},
		"toJSON": {
			Fn:          dictToJSON,
			Arity:       "0",
			Description: "Convert to JSON string",
		},
		"toBox": {
			Fn:          dictToBoxMethod,
			Arity:       "0+",
			Description: "Render as box diagram",
		},
		"repr": {
			Fn:          dictRepr,
			Arity:       "0",
			Description: "Get representation string",
		},
		"toHTML": {
			Fn:          dictToHTMLMethod,
			Arity:       "0+",
			Description: "Convert to HTML",
		},
		"toMarkdown": {
			Fn:          dictToMarkdownMethod,
			Arity:       "0+",
			Description: "Convert to Markdown",
		},
		"as": {
			Fn:          dictAs,
			Arity:       "1",
			Description: "Cast to record using schema",
		},
		"reorder": {
			Fn:          dictReorderMethod,
			Arity:       "1+",
			Description: "Reorder and optionally rename keys",
		},
	}
	RegisterMethodRegistry("dictionary", DictionaryMethodRegistry)
}

// ============================================================================
// Dictionary method implementations
// ============================================================================

func dictKeys(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	orderedKeys := dict.Keys()
	keys := make([]Object, 0, len(orderedKeys))
	for _, k := range orderedKeys {
		if !strings.HasPrefix(k, "__") {
			keys = append(keys, &String{Value: k})
		}
	}
	return &Array{Elements: keys}
}

func dictValues(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	orderedKeys := dict.Keys()
	values := make([]Object, 0, len(orderedKeys))
	for _, k := range orderedKeys {
		if !strings.HasPrefix(k, "__") {
			val := Eval(dict.Pairs[k], dict.Env)
			values = append(values, val)
		}
	}
	return &Array{Elements: values}
}

func dictEntries(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)

	// entries() accepts 0 or exactly 2 args — validate manually since arity spec is "0+"
	if len(args) != 0 && len(args) != 2 {
		return newArityErrorExact("entries", len(args), 0, 2)
	}

	keyName := "key"
	valueName := "value"
	if len(args) == 2 {
		k, ok := args[0].(*String)
		if !ok {
			return newTypeError("TYPE-0005", "entries", "a string (key name)", args[0].Type())
		}
		v, ok := args[1].(*String)
		if !ok {
			return newTypeError("TYPE-0006", "entries", "a string (value name)", args[1].Type())
		}
		keyName = k.Value
		valueName = v.Value
	}

	orderedKeys := dict.Keys()
	entries := make([]Object, 0, len(orderedKeys))
	for _, k := range orderedKeys {
		if !strings.HasPrefix(k, "__") {
			val := Eval(dict.Pairs[k], dict.Env)
			entryPairs := map[string]ast.Expression{
				keyName:   objectToExpression(&String{Value: k}),
				valueName: objectToExpression(val),
			}
			entries = append(entries, &Dictionary{
				Pairs:    entryPairs,
				KeyOrder: []string{keyName, valueName},
				Env:      env,
			})
		}
	}
	return &Array{Elements: entries}
}

func dictHas(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	key, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0012", "has", "a string", args[0].Type())
	}
	_, exists := dict.Pairs[key.Value]
	return nativeBoolToParsBoolean(exists)
}

func dictDelete(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	key, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0012", "delete", "a string", args[0].Type())
	}
	dict.DeleteKey(key.Value)
	return NULL
}

func dictInsertAfter(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	existingKey, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0012", "insertAfter", "a string (existing key)", args[0].Type())
	}
	newKey, ok := args[1].(*String)
	if !ok {
		return newTypeError("TYPE-0012", "insertAfter", "a string (new key)", args[1].Type())
	}
	if _, exists := dict.Pairs[existingKey.Value]; !exists {
		return newIndexError("INDEX-0005", map[string]any{"Key": existingKey.Value})
	}
	if _, exists := dict.Pairs[newKey.Value]; exists {
		return newStructuredError("TYPE-0023", map[string]any{"Key": newKey.Value})
	}
	return insertDictKeyAfter(dict, existingKey.Value, newKey.Value, args[2], env)
}

func dictInsertBefore(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	existingKey, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0012", "insertBefore", "a string (existing key)", args[0].Type())
	}
	newKey, ok := args[1].(*String)
	if !ok {
		return newTypeError("TYPE-0012", "insertBefore", "a string (new key)", args[1].Type())
	}
	if _, exists := dict.Pairs[existingKey.Value]; !exists {
		return newIndexError("INDEX-0005", map[string]any{"Key": existingKey.Value})
	}
	if _, exists := dict.Pairs[newKey.Value]; exists {
		return newStructuredError("TYPE-0023", map[string]any{"Key": newKey.Value})
	}
	return insertDictKeyBefore(dict, existingKey.Value, newKey.Value, args[2], env)
}

func dictRender(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	templateStr, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0012", "render", "a string", args[0].Type())
	}
	renderEnv, errObj := buildRenderEnv(env, dict)
	if errObj != nil {
		return errObj
	}
	return interpolateRawString(templateStr.Value, renderEnv)
}

func dictToJSON(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	jsonBytes, err := json.Marshal(objectToGo(dict))
	if err != nil {
		return newFormatError("FMT-0005", err)
	}
	return &String{Value: string(jsonBytes)}
}

func dictToBoxMethod(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	return dictToBox(dict, args, env)
}

func dictRepr(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	return &String{Value: objectToReprString(dict)}
}

func dictToHTMLMethod(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	return dictToHTML(dict, args, env)
}

func dictToMarkdownMethod(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	return dictToMarkdown(dict, args, env)
}

func dictAs(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	schema, ok := args[0].(*DSLSchema)
	if !ok {
		return newTypeError("TYPE-0001", "as", "a schema", args[0].Type())
	}
	return CreateRecord(schema, dict, env)
}

func dictReorderMethod(receiver Object, args []Object, env *Environment) Object {
	dict := receiver.(*Dictionary)
	return dictReorder(dict, args, env)
}
