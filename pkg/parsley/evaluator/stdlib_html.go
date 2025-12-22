package evaluator

import (
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
)

// PreludeLoader is a function that loads a prelude AST by path.
// This is set by the server package to allow the evaluator to access prelude files.
var PreludeLoader func(path string) *ast.Program

// fileToComponentName converts a filename to a PascalCase component name.
// Examples:
//   - "text_field.pars" -> "TextField"
//   - "sr_only.pars" -> "SrOnly"
//   - "data_table.pars" -> "DataTable"
func fileToComponentName(filename string) string {
	// Remove .pars extension
	name := strings.TrimSuffix(filename, ".pars")

	// Split on underscores and capitalize each part
	parts := strings.Split(name, "_")
	var result strings.Builder
	for _, part := range parts {
		if len(part) > 0 {
			// Capitalize first letter
			result.WriteString(strings.ToUpper(part[:1]))
			if len(part) > 1 {
				result.WriteString(part[1:])
			}
		}
	}
	return result.String()
}

// componentFiles maps component filenames to their export names.
// This list defines which components are loaded from the prelude.
var componentFiles = []struct {
	file string
	name string // export name (defaults to PascalCase of filename)
}{
	// Form components
	{"text_field.pars", "TextField"},
	{"textarea_field.pars", "TextareaField"},
	{"select_field.pars", "SelectField"},
	{"radio_group.pars", "RadioGroup"},
	{"checkbox_group.pars", "CheckboxGroup"},
	{"checkbox.pars", "Checkbox"},
	{"button.pars", "Button"},
	{"form.pars", "Form"},

	// Navigation components
	{"nav.pars", "Nav"},
	{"breadcrumb.pars", "Breadcrumb"},
	{"skip_link.pars", "SkipLink"},

	// Media components
	{"img.pars", "Img"},
	{"iframe.pars", "Iframe"},
	{"figure.pars", "Figure"},
	{"blockquote.pars", "Blockquote"},

	// Utility components
	{"sr_only.pars", "SrOnly"},
	{"abbr.pars", "Abbr"},
	{"a.pars", "A"},
	{"icon.pars", "Icon"},

	// Time components
	{"time.pars", "Time"},
	{"local_time.pars", "LocalTime"},
	{"time_range.pars", "TimeRange"},
	{"relative_time.pars", "RelativeTime"},

	// Table components
	{"data_table.pars", "DataTable"},
}

// loadHTMLModule loads the HTML components module from prelude.
// Components are pre-parsed .pars files in the prelude/components/ directory.
func loadHTMLModule(env *Environment) Object {
	// Check if prelude loader is available
	if PreludeLoader == nil {
		return &Error{
			Class:   ClassImport,
			Code:    "HTML-0001",
			Message: "HTML components not available: prelude not initialized",
			Hints:   []string{"HTML components require the Basil server environment"},
		}
	}

	exports := make(map[string]Object)

	for _, comp := range componentFiles {
		// Load the component AST from prelude
		program := PreludeLoader("components/" + comp.file)
		if program == nil {
			// Component not found - skip it (allows gradual implementation)
			continue
		}

		// Create an isolated environment for the component
		compEnv := NewEnvironment()
		compEnv.Filename = "prelude/components/" + comp.file

		// Copy parent environment context for stdlib access
		compEnv.Security = env.Security
		compEnv.DevLog = env.DevLog
		compEnv.BasilCtx = env.BasilCtx
		compEnv.AssetRegistry = env.AssetRegistry
		compEnv.AssetBundle = env.AssetBundle

		// Evaluate the component file to populate exports
		result := Eval(program, compEnv)
		if isError(result) {
			// Log error but continue loading other components
			continue
		}

		// Extract the exported function (named after the component)
		if compEnv.IsExported(comp.name) {
			if fn, ok := compEnv.store[comp.name]; ok {
				exports[comp.name] = fn
			}
		}
	}

	return &StdlibModuleDict{
		Exports: exports,
	}
}
