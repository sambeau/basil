// Package help provides a unified help system for Parsley documentation.
// It offers topic-based lookup for types, builtins, operators, and modules,
// accessible via CLI (`pars describe`) and REPL (`:describe`).
package help

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
)

// NOTE: Module metadata (descriptions, export lists) is owned by each module
// file via ModuleMeta vars. The help system reads it through the registry
// functions GetStdlibModuleMeta / GetBasilModuleMeta — no separate export
// lists are maintained here.

// TopicResult represents the help output for a topic
type TopicResult struct {
	Kind        string                   `json:"kind"`
	Name        string                   `json:"name"`
	Description string                   `json:"description,omitempty"`
	Methods     []evaluator.MethodInfo   `json:"methods,omitempty"`
	Properties  []evaluator.PropertyInfo `json:"properties,omitempty"`
	Builtins    []evaluator.BuiltinInfo  `json:"builtins,omitempty"`
	Operators   []evaluator.OperatorInfo `json:"operators,omitempty"`
	Exports     []ExportEntry            `json:"exports,omitempty"`
	TypeNames   []string                 `json:"type_names,omitempty"`
	Params      []string                 `json:"params,omitempty"`
	Arity       string                   `json:"arity,omitempty"`
	Category    string                   `json:"category,omitempty"`
	Deprecated  string                   `json:"deprecated,omitempty"`
}

// ExportEntry represents a module export for help output
type ExportEntry struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Arity       string `json:"arity,omitempty"`
	Description string `json:"description,omitempty"`
	Value       string `json:"value,omitempty"`
}

// DescribeTopic returns help information for the given topic.
// Topics can be: type names (string, array), module paths (@std/math),
// special keywords (builtins, operators, types), or builtin names (JSON, CSV).
func DescribeTopic(topic string) (*TopicResult, error) {
	if topic == "" {
		return nil, fmt.Errorf("no topic specified (try: types, builtins, operators, @std/math, string, array)")
	}

	topic = strings.TrimSpace(topic)

	// Check for type (registries first, then TypeMethods)
	if result := describeType(topic); result != nil {
		return result, nil
	}

	// Check for module paths
	if strings.HasPrefix(topic, "@std/") || strings.HasPrefix(topic, "@basil/") {
		return describeModule(topic)
	}

	// Check special keywords
	switch topic {
	case "builtins":
		return describeBuiltins(), nil
	case "operators":
		return describeOperators(), nil
	case "types":
		return describeTypes(), nil
	}

	// Check for specific builtin name
	if result := describeBuiltinByName(topic); result != nil {
		return result, nil
	}

	// Unknown topic - provide helpful error
	return nil, unknownTopicError(topic)
}

// describeType returns help for a type, or nil if not found
func describeType(typeName string) *TopicResult {
	// Normalize type name to lowercase for lookup
	normalizedName := strings.ToLower(typeName)

	// Check method registries first (migrated types: string, integer, float, money)
	methods := evaluator.GetMethodsForType(normalizedName)

	// Fall back to TypeMethods for unmigrated types
	if methods == nil {
		if typeMethods, ok := evaluator.TypeMethods[normalizedName]; ok {
			methods = typeMethods
		}
	}

	// If no methods found, check if it's a known type with properties
	properties := evaluator.TypeProperties[normalizedName]

	if methods == nil && properties == nil {
		return nil // Not a known type
	}

	// Sort methods alphabetically
	if methods != nil {
		sortedMethods := make([]evaluator.MethodInfo, len(methods))
		copy(sortedMethods, methods)
		sort.Slice(sortedMethods, func(i, j int) bool {
			return sortedMethods[i].Name < sortedMethods[j].Name
		})
		methods = sortedMethods
	}

	// Sort properties alphabetically
	if properties != nil {
		sortedProps := make([]evaluator.PropertyInfo, len(properties))
		copy(sortedProps, properties)
		sort.Slice(sortedProps, func(i, j int) bool {
			return sortedProps[i].Name < sortedProps[j].Name
		})
		properties = sortedProps
	}

	return &TopicResult{
		Kind:       "type",
		Name:       normalizedName,
		Methods:    methods,
		Properties: properties,
	}
}

// describeModule returns help for a stdlib or basil module
func describeModule(modulePath string) (*TopicResult, error) {
	// Extract module name from path and look up metadata from registry
	var meta *evaluator.ModuleMeta

	if name, found := strings.CutPrefix(modulePath, "@std/"); found {
		meta = evaluator.GetStdlibModuleMeta(name)
		if meta == nil {
			return nil, fmt.Errorf("unknown module: %s (available @std modules: %s)",
				modulePath, strings.Join(evaluator.GetStdlibModuleNames(), ", "))
		}
	} else if name, found := strings.CutPrefix(modulePath, "@basil/"); found {
		meta = evaluator.GetBasilModuleMeta(name)
		if meta == nil {
			return nil, fmt.Errorf("unknown module: %s (available @basil modules: %s)",
				modulePath, strings.Join(evaluator.GetBasilModuleNames(), ", "))
		}
	}

	// Build exports from module metadata
	exports := getModuleExports(meta)

	return &TopicResult{
		Kind:        "module",
		Name:        modulePath,
		Description: meta.Description,
		Exports:     exports,
	}, nil
}

// describeBuiltins returns a list of all builtins grouped by category
func describeBuiltins() *TopicResult {
	// Collect all builtins
	builtins := make([]evaluator.BuiltinInfo, 0, len(evaluator.BuiltinMetadata))
	for _, info := range evaluator.BuiltinMetadata {
		builtins = append(builtins, info)
	}

	// Sort by category, then by name
	sort.Slice(builtins, func(i, j int) bool {
		if builtins[i].Category != builtins[j].Category {
			return builtins[i].Category < builtins[j].Category
		}
		return builtins[i].Name < builtins[j].Name
	})

	return &TopicResult{
		Kind:     "builtin-list",
		Name:     "builtins",
		Builtins: builtins,
	}
}

// describeOperators returns a list of all operators grouped by category
func describeOperators() *TopicResult {
	// Collect all operators
	operators := make([]evaluator.OperatorInfo, 0, len(evaluator.OperatorMetadata))
	for _, info := range evaluator.OperatorMetadata {
		operators = append(operators, info)
	}

	// Sort by category, then by symbol
	sort.Slice(operators, func(i, j int) bool {
		if operators[i].Category != operators[j].Category {
			return operators[i].Category < operators[j].Category
		}
		return operators[i].Symbol < operators[j].Symbol
	})

	return &TopicResult{
		Kind:      "operator-list",
		Name:      "operators",
		Operators: operators,
	}
}

// describeTypes returns a list of all known types
func describeTypes() *TopicResult {
	typeNames := make(map[string]bool)

	// Collect from TypeMethods
	for name := range evaluator.TypeMethods {
		typeNames[name] = true
	}

	// Collect from TypeProperties
	for name := range evaluator.TypeProperties {
		typeNames[name] = true
	}

	// Known types that have registries (FEAT-111 migrated types)
	migratedTypes := []string{"string", "integer", "float", "money"}
	for _, name := range migratedTypes {
		typeNames[name] = true
	}

	// Convert to sorted slice
	names := make([]string, 0, len(typeNames))
	for name := range typeNames {
		names = append(names, name)
	}
	sort.Strings(names)

	return &TopicResult{
		Kind:      "type-list",
		Name:      "types",
		TypeNames: names,
	}
}

// describeBuiltinByName returns help for a specific builtin, or nil if not found
func describeBuiltinByName(name string) *TopicResult {
	info, ok := evaluator.BuiltinMetadata[name]
	if !ok {
		return nil
	}

	return &TopicResult{
		Kind:        "builtin",
		Name:        info.Name,
		Description: info.Description,
		Params:      info.Params,
		Arity:       info.Arity,
		Category:    info.Category,
		Deprecated:  info.Deprecated,
	}
}

// unknownTopicError generates a helpful error for unknown topics
func unknownTopicError(topic string) error {
	// Try to suggest similar topics
	suggestions := findSuggestions(topic)

	if len(suggestions) > 0 {
		return fmt.Errorf("unknown topic: %s\nDid you mean: %s?", topic, strings.Join(suggestions, ", "))
	}

	return fmt.Errorf("unknown topic: %s\nTry: types, builtins, operators, string, array, @std/math, JSON", topic)
}

// findSuggestions finds topics similar to the given unknown topic
func findSuggestions(topic string) []string {
	topic = strings.ToLower(topic)
	var suggestions []string

	// Check type names
	allTypes := describeTypes().TypeNames
	for _, name := range allTypes {
		if strings.Contains(name, topic) || strings.Contains(topic, name) {
			suggestions = append(suggestions, name)
		}
	}

	// Check builtin names
	for name := range evaluator.BuiltinMetadata {
		nameLower := strings.ToLower(name)
		if strings.Contains(nameLower, topic) || strings.Contains(topic, nameLower) {
			suggestions = append(suggestions, name)
		}
	}

	// Check module names
	for _, name := range evaluator.GetStdlibModuleNames() {
		if strings.Contains(name, topic) || strings.Contains(topic, name) {
			suggestions = append(suggestions, "@std/"+name)
		}
	}

	// Limit to 3 suggestions
	if len(suggestions) > 3 {
		suggestions = suggestions[:3]
	}

	return suggestions
}

// getModuleExports builds the export list from a module's own metadata.
func getModuleExports(meta *evaluator.ModuleMeta) []ExportEntry {
	if meta == nil || len(meta.Exports) == 0 {
		return nil
	}

	exports := make([]ExportEntry, 0, len(meta.Exports))
	for name, em := range meta.Exports {
		exports = append(exports, ExportEntry{
			Name:        name,
			Kind:        em.Kind,
			Arity:       em.Arity,
			Description: em.Description,
		})
	}

	// Sort: constants first, then functions, alphabetically within each
	sort.Slice(exports, func(i, j int) bool {
		if exports[i].Kind != exports[j].Kind {
			return exports[i].Kind == "constant" // constants first
		}
		return exports[i].Name < exports[j].Name
	})

	return exports
}
