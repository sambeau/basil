// Package server provides PLN signing hooks for Part props.
package server

import (
	"fmt"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/pln"
)

func init() {
	// Register PLN signing functions with the evaluator for Part props (FEAT-098)
	evaluator.RegisterPLNSigningFunctions(SignPLN, deserializePLNProp)
}

// deserializePLNProp verifies and deserializes a signed PLN string.
func deserializePLNProp(signed string, secret string, env *evaluator.Environment) (evaluator.Object, error) {
	// Verify HMAC signature
	plnStr, err := VerifyPLN(signed, secret)
	if err != nil {
		return nil, fmt.Errorf("PLN signature verification failed: %w", err)
	}

	// Deserialize PLN to object
	// Use schema resolver from environment if available
	var resolver pln.SchemaResolver
	if env != nil {
		resolver = func(name string) *evaluator.DSLSchema {
			// Look up schema in environment
			if schemaObj, ok := env.Get(name); ok {
				if schema, ok := schemaObj.(*evaluator.DSLSchema); ok {
					return schema
				}
			}
			return nil
		}
	}

	return pln.Deserialize(plnStr, resolver, env)
}
