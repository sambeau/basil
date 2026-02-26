package evaluator

import (
	"github.com/sambeau/basil/pkg/parsley/ast"
)

var htmlModuleMeta = ModuleMeta{
	Description: "Pre-built HTML components (requires Basil server)",
	Exports: map[string]ExportMeta{
		// Layout components
		"Page": {Kind: "component", Description: "Page layout wrapper"},
		"Head": {Kind: "component", Description: "HTML head section"},
		// Form components
		"TextField":     {Kind: "component", Description: "Text input field"},
		"TextareaField": {Kind: "component", Description: "Textarea input field"},
		"SelectField":   {Kind: "component", Description: "Select dropdown field"},
		"RadioGroup":    {Kind: "component", Description: "Radio button group"},
		"CheckboxGroup": {Kind: "component", Description: "Checkbox group"},
		"Checkbox":      {Kind: "component", Description: "Single checkbox"},
		"Button":        {Kind: "component", Description: "Button element"},
		"Form":          {Kind: "component", Description: "Form wrapper"},
		// Navigation components
		"Nav":        {Kind: "component", Description: "Navigation wrapper"},
		"Breadcrumb": {Kind: "component", Description: "Breadcrumb navigation"},
		"SkipLink":   {Kind: "component", Description: "Skip to content link"},
		// Media components
		"Img":        {Kind: "component", Description: "Image element"},
		"Iframe":     {Kind: "component", Description: "Iframe element"},
		"Figure":     {Kind: "component", Description: "Figure with caption"},
		"Blockquote": {Kind: "component", Description: "Blockquote element"},
		// Utility components
		"SrOnly": {Kind: "component", Description: "Screen reader only text"},
		"Abbr":   {Kind: "component", Description: "Abbreviation element"},
		"A":      {Kind: "component", Description: "Anchor/link element"},
		"Icon":   {Kind: "component", Description: "Icon element"},
		// Time components
		"Time":         {Kind: "component", Description: "Time element"},
		"LocalTime":    {Kind: "component", Description: "Localized time display"},
		"TimeRange":    {Kind: "component", Description: "Time range display"},
		"RelativeTime": {Kind: "component", Description: "Relative time display"},
		// Table components
		"DataTable": {Kind: "component", Description: "Data table component"},
	},
}

// PreludeLoader is a function that loads a prelude AST by path.
// This is set by the server package to allow the evaluator to access prelude files.
var PreludeLoader func(path string) *ast.Program

// componentFiles maps component filenames to their export names.
// This list defines which components are loaded from the prelude.
var componentFiles = []struct {
	file string
	name string // export name (defaults to PascalCase of filename)
}{
	// Layout components
	{"page.pars", "Page"},
	{"head.pars", "Head"},

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
// Uses a two-pass approach so components can reference each other.
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

	// Pass 1: Load all component ASTs
	type componentAST struct {
		name    string
		program *ast.Program
	}
	var components []componentAST

	for _, comp := range componentFiles {
		// Load the component AST from prelude
		program := PreludeLoader("components/" + comp.file)
		if program == nil {
			// Component not found - skip it (allows gradual implementation)
			continue
		}
		components = append(components, componentAST{
			name:    comp.name,
			program: program,
		})
	}

	// Pass 2: Evaluate components with shared environment
	// This allows components to reference each other (e.g., Page uses SkipLink)
	sharedEnv := NewEnvironment()
	sharedEnv.Security = env.Security
	sharedEnv.DevLog = env.DevLog
	sharedEnv.BasilCtx = env.BasilCtx
	sharedEnv.AssetRegistry = env.AssetRegistry
	sharedEnv.AssetBundle = env.AssetBundle

	exports := make(map[string]Object)

	for _, comp := range components {
		// Evaluate in the shared environment
		sharedEnv.Filename = "prelude/components/" + comp.name
		result := Eval(comp.program, sharedEnv)
		if isError(result) {
			// Log error but continue loading other components
			continue
		}

		// Extract the exported function
		if sharedEnv.IsExported(comp.name) {
			if fn, ok := sharedEnv.store[comp.name]; ok {
				exports[comp.name] = fn
			}
		}
	}

	return &StdlibModuleDict{
		Exports: exports,
	}
}
