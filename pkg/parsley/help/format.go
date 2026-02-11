package help

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
)

// FormatText formats a TopicResult for terminal output with the given width
func FormatText(result *TopicResult, width int) string {
	if width <= 0 {
		width = 80
	}

	var sb strings.Builder

	switch result.Kind {
	case "type":
		formatTypeText(&sb, result, width)
	case "module":
		formatModuleText(&sb, result, width)
	case "builtin":
		formatBuiltinText(&sb, result)
	case "builtin-list":
		formatBuiltinListText(&sb, result, width)
	case "operator-list":
		formatOperatorListText(&sb, result, width)
	case "type-list":
		formatTypeListText(&sb, result, width)
	default:
		sb.WriteString(fmt.Sprintf("Unknown result kind: %s\n", result.Kind))
	}

	return sb.String()
}

// FormatJSON formats a TopicResult as JSON
func FormatJSON(result *TopicResult) ([]byte, error) {
	return json.MarshalIndent(result, "", "  ")
}

// formatTypeText formats type help output
func formatTypeText(sb *strings.Builder, result *TopicResult, width int) {
	fmt.Fprintf(sb, "Type: %s\n", result.Name)

	// Properties
	if len(result.Properties) > 0 {
		sb.WriteString("\nProperties:\n")

		// Find max property name length for alignment
		maxLen := 0
		for _, p := range result.Properties {
			display := fmt.Sprintf(".%s: %s", p.Name, p.Type)
			if len(display) > maxLen {
				maxLen = len(display)
			}
		}

		for _, p := range result.Properties {
			display := fmt.Sprintf(".%s: %s", p.Name, p.Type)
			padding := strings.Repeat(" ", maxLen-len(display)+2)
			fmt.Fprintf(sb, "  %s%s%s\n", display, padding, p.Description)
		}
	}

	// Methods
	if len(result.Methods) > 0 {
		sb.WriteString("\nMethods:\n")

		// Find max method signature length for alignment
		maxLen := 0
		for _, m := range result.Methods {
			display := fmt.Sprintf(".%s(%s)", m.Name, arityToParams(m.Arity))
			if len(display) > maxLen {
				maxLen = len(display)
			}
		}

		for _, m := range result.Methods {
			display := fmt.Sprintf(".%s(%s)", m.Name, arityToParams(m.Arity))
			padding := strings.Repeat(" ", maxLen-len(display)+2)
			fmt.Fprintf(sb, "  %s%s%s\n", display, padding, m.Description)
		}
	}

	if len(result.Properties) == 0 && len(result.Methods) == 0 {
		sb.WriteString("\n(no properties or methods)\n")
	}
}

// formatModuleText formats module help output
func formatModuleText(sb *strings.Builder, result *TopicResult, width int) {
	fmt.Fprintf(sb, "Module: %s\n", result.Name)
	if result.Description != "" {
		fmt.Fprintf(sb, "\n%s\n", result.Description)
	}

	if len(result.Exports) == 0 {
		sb.WriteString("\n(no documented exports)\n")
		return
	}

	// Separate constants and functions
	var constants, functions []ExportEntry
	for _, e := range result.Exports {
		if e.Kind == "constant" {
			constants = append(constants, e)
		} else {
			functions = append(functions, e)
		}
	}

	// Find max name length for alignment
	maxLen := 0
	for _, e := range result.Exports {
		var display string
		if e.Kind == "constant" {
			display = e.Name
		} else {
			display = fmt.Sprintf("%s(%s)", e.Name, arityToParams(e.Arity))
		}
		if len(display) > maxLen {
			maxLen = len(display)
		}
	}

	sb.WriteString("\nExports:\n")

	if len(constants) > 0 {
		sb.WriteString("  Constants:\n")
		for _, e := range constants {
			padding := strings.Repeat(" ", maxLen-len(e.Name)+2)
			if e.Description != "" {
				fmt.Fprintf(sb, "    %s%s%s\n", e.Name, padding, e.Description)
			} else {
				fmt.Fprintf(sb, "    %s\n", e.Name)
			}
		}
	}

	if len(functions) > 0 {
		if len(constants) > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("  Functions:\n")
		for _, e := range functions {
			display := fmt.Sprintf("%s(%s)", e.Name, arityToParams(e.Arity))
			padding := strings.Repeat(" ", maxLen-len(display)+2)
			if e.Description != "" {
				fmt.Fprintf(sb, "    %s%s%s\n", display, padding, e.Description)
			} else {
				fmt.Fprintf(sb, "    %s\n", display)
			}
		}
	}
}

// formatBuiltinText formats a single builtin's help output
func formatBuiltinText(sb *strings.Builder, result *TopicResult) {
	// Signature
	fmt.Fprintf(sb, "%s(%s)\n", result.Name, strings.Join(result.Params, ", "))
	sb.WriteString("\n")

	// Description
	fmt.Fprintf(sb, "%s\n", result.Description)
	sb.WriteString("\n")

	// Arity
	fmt.Fprintf(sb, "Arity: %s\n", result.Arity)

	// Category
	fmt.Fprintf(sb, "Category: %s\n", result.Category)

	// Deprecation warning
	if result.Deprecated != "" {
		fmt.Fprintf(sb, "\n⚠ DEPRECATED: %s\n", result.Deprecated)
	}
}

// formatBuiltinListText formats the builtins list output
func formatBuiltinListText(sb *strings.Builder, result *TopicResult, width int) {
	sb.WriteString("Builtin Functions\n")
	sb.WriteString("=================\n\n")

	// Group by category
	byCategory := make(map[string][]evaluator.BuiltinInfo)
	for _, b := range result.Builtins {
		byCategory[b.Category] = append(byCategory[b.Category], b)
	}

	// Sort categories
	categories := make([]string, 0, len(byCategory))
	for cat := range byCategory {
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	// Category display names
	categoryNames := map[string]string{
		"file":          "File/Data Loading",
		"time":          "Date & Time",
		"url":           "URLs",
		"path":          "Paths",
		"conversion":    "Type Conversion",
		"serialization": "Serialization",
		"introspection": "Introspection",
		"output":        "Output",
		"control":       "Control Flow",
		"format":        "Formatting",
		"regex":         "Regular Expressions",
		"money":         "Money",
		"asset":         "Assets",
		"connection":    "Connections",
	}

	for _, cat := range categories {
		builtins := byCategory[cat]

		// Display category name
		displayName := categoryNames[cat]
		if displayName == "" {
			displayName = strings.ToUpper(cat[:1]) + cat[1:]
		}
		fmt.Fprintf(sb, "%s:\n", displayName)

		// Find max name length for alignment
		maxLen := 0
		for _, b := range builtins {
			display := fmt.Sprintf("%s(%s)", b.Name, strings.Join(b.Params, ", "))
			if len(display) > maxLen {
				maxLen = len(display)
			}
		}

		// Sort builtins in category by name
		sort.Slice(builtins, func(i, j int) bool {
			return builtins[i].Name < builtins[j].Name
		})

		for _, b := range builtins {
			display := fmt.Sprintf("%s(%s)", b.Name, strings.Join(b.Params, ", "))
			padding := strings.Repeat(" ", maxLen-len(display)+2)
			fmt.Fprintf(sb, "  %s%s%s\n", display, padding, b.Description)
		}
		sb.WriteString("\n")
	}
}

// formatOperatorListText formats the operators list output
func formatOperatorListText(sb *strings.Builder, result *TopicResult, width int) {
	sb.WriteString("Operators\n")
	sb.WriteString("=========\n\n")

	// Group by category
	byCategory := make(map[string][]evaluator.OperatorInfo)
	for _, op := range result.Operators {
		byCategory[op.Category] = append(byCategory[op.Category], op)
	}

	// Define category order
	categoryOrder := []string{
		"arithmetic",
		"comparison",
		"logical",
		"collection",
		"regex",
		"pipe",
		"null",
		"control",
	}

	// Category display names
	categoryNames := map[string]string{
		"arithmetic": "Arithmetic",
		"comparison": "Comparison",
		"logical":    "Logical",
		"collection": "Collection",
		"regex":      "Regex",
		"pipe":       "Pipe",
		"null":       "Null Handling",
		"control":    "Control Flow",
	}

	for _, cat := range categoryOrder {
		ops, ok := byCategory[cat]
		if !ok || len(ops) == 0 {
			continue
		}

		// Display category name
		displayName := categoryNames[cat]
		if displayName == "" {
			displayName = strings.ToUpper(cat[:1]) + cat[1:]
		}
		fmt.Fprintf(sb, "%s:\n", displayName)

		// Find max symbol length for alignment
		maxLen := 0
		for _, op := range ops {
			if len(op.Symbol) > maxLen {
				maxLen = len(op.Symbol)
			}
		}
		if maxLen < 6 {
			maxLen = 6 // Minimum for alignment
		}

		// Sort operators in category by symbol
		sort.Slice(ops, func(i, j int) bool {
			return ops[i].Symbol < ops[j].Symbol
		})

		for _, op := range ops {
			padding := strings.Repeat(" ", maxLen-len(op.Symbol)+2)
			fmt.Fprintf(sb, "  %s%s%s\n", op.Symbol, padding, op.Description)
		}
		sb.WriteString("\n")
	}

	// Any categories not in the predefined order
	for cat := range byCategory {
		if !slices.Contains(categoryOrder, cat) {
			ops := byCategory[cat]
			fmt.Fprintf(sb, "%s:\n", strings.ToUpper(cat[:1])+cat[1:])
			for _, op := range ops {
				fmt.Fprintf(sb, "  %s  %s\n", op.Symbol, op.Description)
			}
			sb.WriteString("\n")
		}
	}
}

// formatTypeListText formats the types list output
func formatTypeListText(sb *strings.Builder, result *TopicResult, width int) {
	sb.WriteString("Available Types\n")
	sb.WriteString("===============\n\n")

	// Group types into categories
	primitives := []string{}
	collections := []string{}
	datetime := []string{}
	io := []string{}
	other := []string{}

	primitiveSet := map[string]bool{"string": true, "integer": true, "float": true, "boolean": true, "null": true, "money": true}
	collectionSet := map[string]bool{"array": true, "dictionary": true, "table": true}
	datetimeSet := map[string]bool{"datetime": true, "duration": true}
	ioSet := map[string]bool{"file": true, "directory": true, "path": true, "url": true, "regex": true, "dbconnection": true, "sftpconnection": true, "sftpfile": true}

	for _, name := range result.TypeNames {
		switch {
		case primitiveSet[name]:
			primitives = append(primitives, name)
		case collectionSet[name]:
			collections = append(collections, name)
		case datetimeSet[name]:
			datetime = append(datetime, name)
		case ioSet[name]:
			io = append(io, name)
		default:
			other = append(other, name)
		}
	}

	if len(primitives) > 0 {
		sb.WriteString("Primitives:\n")
		fmt.Fprintf(sb, "  %s\n\n", strings.Join(primitives, ", "))
	}

	if len(collections) > 0 {
		sb.WriteString("Collections:\n")
		fmt.Fprintf(sb, "  %s\n\n", strings.Join(collections, ", "))
	}

	if len(datetime) > 0 {
		sb.WriteString("Date & Time:\n")
		fmt.Fprintf(sb, "  %s\n\n", strings.Join(datetime, ", "))
	}

	if len(io) > 0 {
		sb.WriteString("I/O & Resources:\n")
		fmt.Fprintf(sb, "  %s\n\n", strings.Join(io, ", "))
	}

	if len(other) > 0 {
		sb.WriteString("Other:\n")
		fmt.Fprintf(sb, "  %s\n\n", strings.Join(other, ", "))
	}

	sb.WriteString("Use 'pars describe <type>' for details on a specific type.\n")
}

// arityToParams converts an arity string to a parameter representation
func arityToParams(arity string) string {
	switch arity {
	case "":
		return ""
	case "0":
		return ""
	case "1":
		return "arg"
	case "2":
		return "arg1, arg2"
	case "3":
		return "arg1, arg2, arg3"
	case "0-1":
		return "arg?"
	case "1-2":
		return "arg1, arg2?"
	case "0-2":
		return "arg1?, arg2?"
	case "1-3":
		return "arg1, arg2?, arg3?"
	case "2-3":
		return "arg1, arg2, arg3?"
	case "1+":
		return "arg, ..."
	case "0+":
		return "..."
	case "2+":
		return "arg1, arg2, ..."
	default:
		return "..."
	}
}
