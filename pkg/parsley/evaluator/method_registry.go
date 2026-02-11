// Package evaluator provides the method registry infrastructure for declarative method definitions.
// This file implements FEAT-111: Declarative Method Registry
package evaluator

import (
	"sort"
	"strconv"
	"strings"
)

// MethodFunc is the signature for all method implementations.
// The receiver is passed as an Object to allow uniform handling across types.
// Methods that don't need env can ignore that parameter.
type MethodFunc func(receiver Object, args []Object, env *Environment) Object

// MethodEntry defines a single method with its implementation and metadata.
// This serves as the single source of truth for both dispatch and introspection.
type MethodEntry struct {
	Fn          MethodFunc
	Arity       string // "0", "1", "0-1", "1+", "2", etc.
	Description string
}

// MethodRegistry maps method names to their entries for a type.
type MethodRegistry map[string]MethodEntry

// Names returns a sorted list of method names in this registry.
// Used for fuzzy matching in error messages.
func (r MethodRegistry) Names() []string {
	names := make([]string, 0, len(r))
	for name := range r {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Get returns the method entry for the given name, if it exists.
func (r MethodRegistry) Get(name string) (MethodEntry, bool) {
	entry, ok := r[name]
	return entry, ok
}

// ToMethodInfos converts the registry to a slice of MethodInfo for introspection.
// Results are sorted alphabetically by method name.
func (r MethodRegistry) ToMethodInfos() []MethodInfo {
	methods := make([]MethodInfo, 0, len(r))
	for name, entry := range r {
		methods = append(methods, MethodInfo{
			Name:        name,
			Arity:       entry.Arity,
			Description: entry.Description,
		})
	}
	sort.Slice(methods, func(i, j int) bool {
		return methods[i].Name < methods[j].Name
	})
	return methods
}

// typeRegistries maps type names to their method registries.
// This is the master registry used by introspection.
var typeRegistries = map[string]MethodRegistry{}

// RegisterMethodRegistry registers a method registry for a type.
// Called during init to populate the master registry.
func RegisterMethodRegistry(typeName string, registry MethodRegistry) {
	typeRegistries[typeName] = registry
}

// GetRegistryForType returns the method registry for a type, or nil if not found.
func GetRegistryForType(typeName string) MethodRegistry {
	return typeRegistries[typeName]
}

// GetMethodsForType returns method info for a type from its registry.
// Used by describe() for introspection output.
func GetMethodsForType(typeName string) []MethodInfo {
	registry := typeRegistries[typeName]
	if registry == nil {
		return nil
	}
	return registry.ToMethodInfos()
}

// checkArity validates that the argument count matches the arity specification.
// Arity specs: "0", "1", "2", "0-1", "1-2", "0-2", "1+", "0+", "2+", etc.
func checkArity(spec string, got int) bool {
	spec = strings.TrimSpace(spec)

	// Exact match: "0", "1", "2", etc.
	if exact, err := strconv.Atoi(spec); err == nil {
		return got == exact
	}

	// Range: "0-1", "1-2", "0-2", etc.
	if strings.Contains(spec, "-") {
		parts := strings.Split(spec, "-")
		if len(parts) == 2 {
			minVal, errMin := strconv.Atoi(parts[0])
			maxVal, errMax := strconv.Atoi(parts[1])
			if errMin == nil && errMax == nil {
				return got >= minVal && got <= maxVal
			}
		}
	}

	// Variadic: "1+", "0+", "2+", etc.
	if suffix, found := strings.CutSuffix(spec, "+"); found {
		minVal, err := strconv.Atoi(suffix)
		if err == nil {
			return got >= minVal
		}
	}

	// Unknown spec - be permissive
	return true
}

// newArityErrorFromSpec creates an arity error based on the spec string.
// This provides better error messages by interpreting the spec.
func newArityErrorFromSpec(method, spec string, got int) *Error {
	spec = strings.TrimSpace(spec)

	// Exact match: "0", "1", "2", etc.
	if exact, err := strconv.Atoi(spec); err == nil {
		return newArityError(method, got, exact)
	}

	// Range: "0-1", "1-2", "0-2", etc.
	if strings.Contains(spec, "-") {
		parts := strings.Split(spec, "-")
		if len(parts) == 2 {
			minVal, errMin := strconv.Atoi(parts[0])
			maxVal, errMax := strconv.Atoi(parts[1])
			if errMin == nil && errMax == nil {
				return newArityErrorRange(method, got, minVal, maxVal)
			}
		}
	}

	// Variadic: "1+", "0+", "2+", etc.
	if suffix, found := strings.CutSuffix(spec, "+"); found {
		minVal, err := strconv.Atoi(suffix)
		if err == nil {
			return newArityErrorMin(method, got, minVal)
		}
	}

	// Fallback - generic error
	return newArityError(method, got, 0)
}

// dispatchFromRegistry handles method dispatch using a registry.
// Returns nil if the method is not found (caller should handle unknown method error).
func dispatchFromRegistry(registry MethodRegistry, typeName string, receiver Object, method string, args []Object, env *Environment) Object {
	entry, ok := registry.Get(method)
	if !ok {
		return nil // Method not found - caller handles error
	}

	if !checkArity(entry.Arity, len(args)) {
		return newArityErrorFromSpec(method, entry.Arity, len(args))
	}

	return entry.Fn(receiver, args, env)
}
