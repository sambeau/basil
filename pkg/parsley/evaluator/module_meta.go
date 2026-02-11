package evaluator

import "sort"

// ExportMeta describes a single module export for help/introspection.
type ExportMeta struct {
	Kind        string `json:"kind"`
	Arity       string `json:"arity,omitempty"`
	Description string `json:"description,omitempty"`
}

// ModuleMeta describes a module and its exports for help/introspection.
// Each module file defines a package-level var (e.g. mathModuleMeta) that
// is referenced both by the loader (attached to StdlibModuleDict.Meta)
// and by the registry maps below (for CLI help without an Environment).
type ModuleMeta struct {
	Description string                `json:"description"`
	Exports     map[string]ExportMeta `json:"exports"`
}

// Registry maps — populated from per-module vars defined in each stdlib_*.go file.
// These are declared as vars so each module file can register via init() or
// direct reference; we use a function approach for clarity and testability.

var stdlibModuleMeta map[string]*ModuleMeta
var basilModuleMeta map[string]*ModuleMeta

func init() {
	stdlibModuleMeta = map[string]*ModuleMeta{
		"math":   &mathModuleMeta,
		"id":     &idModuleMeta,
		"valid":  &validModuleMeta,
		"schema": &schemaModuleMeta,
		"api":    &apiModuleMeta,
		"dev":    &devModuleMeta,
		"table":  &tableModuleMeta,
	}
	basilModuleMeta = map[string]*ModuleMeta{
		"http": &basilHTTPModuleMeta,
		"auth": &basilAuthModuleMeta,
	}
}

// GetStdlibModuleMeta returns metadata for a stdlib module, or nil if unknown.
func GetStdlibModuleMeta(name string) *ModuleMeta {
	return stdlibModuleMeta[name]
}

// GetBasilModuleMeta returns metadata for a basil module, or nil if unknown.
func GetBasilModuleMeta(name string) *ModuleMeta {
	return basilModuleMeta[name]
}

// GetStdlibModuleNames returns sorted names of all stdlib modules that have metadata.
func GetStdlibModuleNames() []string {
	names := make([]string, 0, len(stdlibModuleMeta))
	for name := range stdlibModuleMeta {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetBasilModuleNames returns sorted names of all basil modules that have metadata.
func GetBasilModuleNames() []string {
	names := make([]string, 0, len(basilModuleMeta))
	for name := range basilModuleMeta {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
