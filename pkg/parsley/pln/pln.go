// Package pln implements Parsley Literal Notation (PLN), a safe data serialization
// format for the Parsley programming language.
//
// PLN is a subset of Parsley syntax that supports only literal values:
//   - Primitives: integers, floats, strings, booleans, null
//   - Collections: arrays, dictionaries
//   - Special types: records (with validation errors), datetimes, paths, URLs
//
// PLN explicitly forbids executable code (expressions, function calls, variables)
// making it safe for data transfer between untrusted contexts.
//
// Example PLN:
//
//	@Person({
//	    name: "Alice",
//	    email: "alice@example.com",
//	    joined: @2024-01-15
//	}) @errors {
//	    email: "Invalid format"
//	}
package pln

import (
	"fmt"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
)

func init() {
	// Register PLN functions with the evaluator to avoid import cycles
	evaluator.RegisterPLNFunctions(serializeWrapper, deserializeWrapper)
	// Register serialize function for Part props (FEAT-098)
	evaluator.RegisterPLNPropFunctions(serializeWrapper)
}

// serializeWrapper wraps the Serialize function for registration with evaluator
func serializeWrapper(obj evaluator.Object, env *evaluator.Environment) (string, error) {
	s := NewSerializerWithEnv(env)
	return s.Serialize(obj)
}

// deserializeWrapper wraps the Deserialize function for registration with evaluator
func deserializeWrapper(input string, resolver func(string) *evaluator.DSLSchema, env *evaluator.Environment) (evaluator.Object, error) {
	return Deserialize(input, resolver, env)
}

// Serialize converts a Parsley object to a PLN string.
// Returns an error if the object contains non-serializable values
// (functions, builtins, database connections, etc.) or circular references.
func Serialize(obj evaluator.Object) (string, error) {
	s := NewSerializer()
	return s.Serialize(obj)
}

// SerializeWithEnv converts a Parsley object to a PLN string,
// using the provided environment to evaluate any lazy expressions.
func SerializeWithEnv(obj evaluator.Object, env *evaluator.Environment) (string, error) {
	s := NewSerializerWithEnv(env)
	return s.Serialize(obj)
}

// SerializePretty converts a Parsley object to a formatted PLN string.
// The indent string is used for each level of nesting.
func SerializePretty(obj evaluator.Object, indent string) (string, error) {
	s := NewPrettySerializer(indent)
	return s.Serialize(obj)
}

// Deserialize parses a PLN string and returns a Parsley object.
// If schemaResolver is nil, records with unknown schemas will be returned
// as dictionaries with a "__schema" field containing the schema name.
func Deserialize(input string, schemaResolver SchemaResolver, env *evaluator.Environment) (evaluator.Object, error) {
	var p *Parser
	if schemaResolver != nil || env != nil {
		p = NewParserWithResolver(input, schemaResolver, env)
	} else {
		p = NewParser(input)
	}

	obj, err := p.Parse()
	if err != nil {
		return nil, err
	}
	return obj, nil
}

// Parse is a convenience function that parses PLN without schema resolution.
// Records will be returned as dictionaries with a "__schema" field.
func Parse(input string) (evaluator.Object, error) {
	return Deserialize(input, nil, nil)
}

// MustParse parses PLN and panics on error. Useful for tests and initialization.
func MustParse(input string) evaluator.Object {
	obj, err := Parse(input)
	if err != nil {
		panic(fmt.Sprintf("pln.MustParse: %v", err))
	}
	return obj
}

// Validate checks if a string is valid PLN without fully parsing it.
// Returns nil if valid, or an error describing the problem.
func Validate(input string) error {
	p := NewParser(input)
	_, err := p.Parse()
	return err
}

// IsValidPLN returns true if the input is valid PLN syntax.
func IsValidPLN(input string) bool {
	return Validate(input) == nil
}
