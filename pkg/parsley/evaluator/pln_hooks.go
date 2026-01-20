package evaluator

import (
	"fmt"
)

// PLNSerializer is a function type for serializing objects to PLN strings.
// This allows the pln package to register its serializer without circular imports.
type PLNSerializer func(obj Object, env *Environment) (string, error)

// PLNDeserializer is a function type for deserializing PLN strings to objects.
type PLNDeserializer func(input string, resolver func(string) *DSLSchema, env *Environment) (Object, error)

var plnSerializer PLNSerializer
var plnDeserializer PLNDeserializer

// RegisterPLNFunctions registers the PLN serialize/deserialize functions.
// Called by the pln package during initialization.
func RegisterPLNFunctions(serializer PLNSerializer, deserializer PLNDeserializer) {
	plnSerializer = serializer
	plnDeserializer = deserializer
}

// SerializeToPLN converts a Parsley object to a PLN string.
// Used by the serialize() builtin function.
func SerializeToPLN(obj Object, env *Environment) Object {
	if plnSerializer == nil {
		return newInternalError("INTERNAL-0002", map[string]any{
			"Context": "PLN serializer not registered",
		})
	}

	result, err := plnSerializer(obj, env)
	if err != nil {
		return newPLNSerializationError("SERIALIZE-0001", obj.Type(), err)
	}
	return &String{Value: result}
}

// DeserializeFromPLN parses a PLN string and returns a Parsley object.
// Used by the deserialize() builtin function.
func DeserializeFromPLN(input string, env *Environment) Object {
	if plnDeserializer == nil {
		return newInternalError("INTERNAL-0002", map[string]any{
			"Context": "PLN deserializer not registered",
		})
	}

	// Create a schema resolver that looks up schemas in the environment
	resolver := func(name string) *DSLSchema {
		if obj, ok := env.Get(name); ok {
			if schema, ok := obj.(*DSLSchema); ok {
				return schema
			}
		}
		return nil
	}

	obj, err := plnDeserializer(input, resolver, env)
	if err != nil {
		return newParseError("DESERIALIZE-0001", "PLN", fmt.Errorf("%s", err.Error()))
	}
	return obj
}

// newPLNSerializationError creates a serialization error for PLN
func newPLNSerializationError(code string, objType ObjectType, err error) *Error {
	return &Error{
		Class:   "SERIALIZATION",
		Code:    code,
		Message: fmt.Sprintf("cannot serialize %s: %s", objType, err.Error()),
		Data: map[string]any{
			"Type":  string(objType),
			"Error": err.Error(),
		},
	}
}

// parsePLN parses a PLN string during file reading operations.
// Used by readFileContent when loading .pln files.
func parsePLN(content string, env *Environment) (Object, *Error) {
	if plnDeserializer == nil {
		return nil, newInternalError("INTERNAL-0002", map[string]any{
			"Context": "PLN deserializer not registered",
		})
	}

	// Create a schema resolver that looks up schemas in the environment
	resolver := func(name string) *DSLSchema {
		if obj, ok := env.Get(name); ok {
			if schema, ok := obj.(*DSLSchema); ok {
				return schema
			}
		}
		return nil
	}

	obj, err := plnDeserializer(content, resolver, env)
	if err != nil {
		return nil, newFormatError("FMT-0007", fmt.Errorf("invalid PLN: %s", err.Error()))
	}
	return obj, nil
}
