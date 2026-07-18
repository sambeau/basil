package help

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
)

// FormatMarkdown formats a TopicResult as Markdown
func FormatMarkdown(result *TopicResult) string {
	var sb strings.Builder

	switch result.Kind {
	case "type":
		formatTypeMarkdown(&sb, result)
	case "module":
		formatModuleMarkdown(&sb, result)
	case "builtin":
		formatBuiltinMarkdown(&sb, result)
	case "builtin-list":
		formatBuiltinListMarkdown(&sb, result)
	case "operator-list":
		formatOperatorListMarkdown(&sb, result)
	case "type-list":
		formatTypeListMarkdown(&sb, result)
	case "all":
		formatAllMarkdown(&sb, result)
	default:
		fmt.Fprintf(&sb, "Unknown result kind: %s\n", result.Kind)
	}

	return sb.String()
}

// FormatMarkdownWithFrontmatter adds YAML frontmatter to the output
func FormatMarkdownWithFrontmatter(result *TopicResult, title string) string {
	var sb strings.Builder

	// YAML frontmatter
	sb.WriteString("---\n")
	fmt.Fprintf(&sb, "title: %q\n", title)
	fmt.Fprintf(&sb, "generated: %s\n", time.Now().UTC().Format(time.RFC3339))
	sb.WriteString("---\n\n")

	sb.WriteString(FormatMarkdown(result))

	return sb.String()
}

// formatTypeMarkdown formats type help as Markdown
func formatTypeMarkdown(sb *strings.Builder, result *TopicResult) {
	fmt.Fprintf(sb, "## %s\n\n", capitalizeFirst(result.Name))

	// Properties table
	if len(result.Properties) > 0 {
		sb.WriteString("### Properties\n\n")
		sb.WriteString("| Property | Type | Description |\n")
		sb.WriteString("|----------|------|-------------|\n")

		for _, p := range result.Properties {
			desc := escapeMarkdownTableCell(p.Description)
			fmt.Fprintf(sb, "| `.%s` | %s | %s |\n", p.Name, p.Type, desc)
		}
		sb.WriteString("\n")
	}

	// Methods table
	if len(result.Methods) > 0 {
		sb.WriteString("### Methods\n\n")
		sb.WriteString("| Method | Description |\n")
		sb.WriteString("|--------|-------------|\n")

		for _, m := range result.Methods {
			sig := fmt.Sprintf(".%s(%s)", m.Name, arityToParams(m.Arity))
			desc := escapeMarkdownTableCell(m.Description)
			fmt.Fprintf(sb, "| `%s` | %s |\n", sig, desc)
		}
		sb.WriteString("\n")
	}

	if len(result.Properties) == 0 && len(result.Methods) == 0 {
		sb.WriteString("*No properties or methods.*\n\n")
	}
}

// formatModuleMarkdown formats module help as Markdown
func formatModuleMarkdown(sb *strings.Builder, result *TopicResult) {
	fmt.Fprintf(sb, "## %s\n\n", result.Name)

	if result.Description != "" {
		fmt.Fprintf(sb, "%s\n\n", result.Description)
	}

	if len(result.Exports) == 0 {
		sb.WriteString("*No documented exports.*\n\n")
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

	// Constants table
	if len(constants) > 0 {
		sb.WriteString("### Constants\n\n")
		sb.WriteString("| Name | Description |\n")
		sb.WriteString("|------|-------------|\n")

		for _, e := range constants {
			desc := escapeMarkdownTableCell(e.Description)
			fmt.Fprintf(sb, "| `%s` | %s |\n", e.Name, desc)
		}
		sb.WriteString("\n")
	}

	// Functions table
	if len(functions) > 0 {
		sb.WriteString("### Functions\n\n")
		sb.WriteString("| Function | Description |\n")
		sb.WriteString("|----------|-------------|\n")

		for _, e := range functions {
			sig := fmt.Sprintf("%s(%s)", e.Name, arityToParams(e.Arity))
			desc := escapeMarkdownTableCell(e.Description)
			fmt.Fprintf(sb, "| `%s` | %s |\n", sig, desc)
		}
		sb.WriteString("\n")
	}
}

// formatBuiltinMarkdown formats a single builtin as Markdown
func formatBuiltinMarkdown(sb *strings.Builder, result *TopicResult) {
	fmt.Fprintf(sb, "### %s\n\n", result.Name)

	// Signature
	fmt.Fprintf(sb, "```parsley\n%s(%s)\n```\n\n", result.Name, strings.Join(result.Params, ", "))

	// Description
	fmt.Fprintf(sb, "%s\n\n", result.Description)

	// Metadata
	fmt.Fprintf(sb, "- **Arity:** %s\n", result.Arity)
	fmt.Fprintf(sb, "- **Category:** %s\n", result.Category)

	// Deprecation warning
	if result.Deprecated != "" {
		fmt.Fprintf(sb, "\n> ⚠️ **Deprecated:** %s\n", result.Deprecated)
	}

	sb.WriteString("\n")
}

// formatBuiltinListMarkdown formats the builtins list as Markdown
func formatBuiltinListMarkdown(sb *strings.Builder, result *TopicResult) {
	sb.WriteString("## Builtin Functions\n\n")

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
		fmt.Fprintf(sb, "### %s\n\n", displayName)

		// Sort builtins in category by name
		sort.Slice(builtins, func(i, j int) bool {
			return builtins[i].Name < builtins[j].Name
		})

		// Table
		sb.WriteString("| Function | Description |\n")
		sb.WriteString("|----------|-------------|\n")

		for _, b := range builtins {
			sig := fmt.Sprintf("%s(%s)", b.Name, strings.Join(b.Params, ", "))
			desc := escapeMarkdownTableCell(b.Description)
			fmt.Fprintf(sb, "| `%s` | %s |\n", sig, desc)
		}
		sb.WriteString("\n")
	}
}

// formatOperatorListMarkdown formats the operators list as Markdown
func formatOperatorListMarkdown(sb *strings.Builder, result *TopicResult) {
	sb.WriteString("## Operators\n\n")

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
		fmt.Fprintf(sb, "### %s\n\n", displayName)

		// Sort operators in category by symbol
		sort.Slice(ops, func(i, j int) bool {
			return ops[i].Symbol < ops[j].Symbol
		})

		// Table
		sb.WriteString("| Operator | Description |\n")
		sb.WriteString("|----------|-------------|\n")

		for _, op := range ops {
			sym := escapeMarkdownTableCell(op.Symbol)
			desc := escapeMarkdownTableCell(op.Description)
			fmt.Fprintf(sb, "| `%s` | %s |\n", sym, desc)
		}
		sb.WriteString("\n")
	}

	// Any categories not in the predefined order
	for cat := range byCategory {
		found := slices.Contains(categoryOrder, cat)
		if !found {
			ops := byCategory[cat]
			fmt.Fprintf(sb, "### %s\n\n", strings.ToUpper(cat[:1])+cat[1:])

			sb.WriteString("| Operator | Description |\n")
			sb.WriteString("|----------|-------------|\n")

			for _, op := range ops {
				sym := escapeMarkdownTableCell(op.Symbol)
				desc := escapeMarkdownTableCell(op.Description)
				fmt.Fprintf(sb, "| `%s` | %s |\n", sym, desc)
			}
			sb.WriteString("\n")
		}
	}
}

// formatTypeListMarkdown formats the types list as Markdown
func formatTypeListMarkdown(sb *strings.Builder, result *TopicResult) {
	sb.WriteString("## Available Types\n\n")

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
		sb.WriteString("### Primitives\n\n")
		for _, name := range primitives {
			fmt.Fprintf(sb, "- `%s`\n", name)
		}
		sb.WriteString("\n")
	}

	if len(collections) > 0 {
		sb.WriteString("### Collections\n\n")
		for _, name := range collections {
			fmt.Fprintf(sb, "- `%s`\n", name)
		}
		sb.WriteString("\n")
	}

	if len(datetime) > 0 {
		sb.WriteString("### Date & Time\n\n")
		for _, name := range datetime {
			fmt.Fprintf(sb, "- `%s`\n", name)
		}
		sb.WriteString("\n")
	}

	if len(io) > 0 {
		sb.WriteString("### I/O & Resources\n\n")
		for _, name := range io {
			fmt.Fprintf(sb, "- `%s`\n", name)
		}
		sb.WriteString("\n")
	}

	if len(other) > 0 {
		sb.WriteString("### Other\n\n")
		for _, name := range other {
			fmt.Fprintf(sb, "- `%s`\n", name)
		}
		sb.WriteString("\n")
	}
}

// formatAllMarkdown formats the complete API schema as Markdown
func formatAllMarkdown(sb *strings.Builder, result *TopicResult) {
	sb.WriteString("# Parsley API Reference\n\n")
	sb.WriteString("> This document is auto-generated from the Parsley source code.\n")
	sb.WriteString("> Do not edit manually.\n\n")

	// Table of contents
	sb.WriteString("## Table of Contents\n\n")
	sb.WriteString("- [Types](#types)\n")
	sb.WriteString("- [Builtin Functions](#builtin-functions)\n")
	sb.WriteString("- [Operators](#operators)\n")
	sb.WriteString("- [Standard Library](#standard-library)\n\n")

	// Types section
	sb.WriteString("## Types\n\n")
	for _, t := range result.Types {
		formatTypeSchemaMarkdown(sb, t)
	}

	// Builtins section
	builtinResult := &TopicResult{
		Kind:     "builtin-list",
		Builtins: result.Builtins,
	}
	formatBuiltinListMarkdown(sb, builtinResult)

	// Operators section
	operatorResult := &TopicResult{
		Kind:      "operator-list",
		Operators: result.Operators,
	}
	formatOperatorListMarkdown(sb, operatorResult)

	// Modules section
	sb.WriteString("## Standard Library\n\n")
	for _, m := range result.Modules {
		formatModuleSchemaMarkdown(sb, m)
	}
}

// formatTypeSchemaMarkdown formats a TypeSchema as Markdown
func formatTypeSchemaMarkdown(sb *strings.Builder, t TypeSchema) {
	fmt.Fprintf(sb, "### %s\n\n", capitalizeFirst(t.Name))

	// Properties table
	if len(t.Properties) > 0 {
		sb.WriteString("#### Properties\n\n")
		sb.WriteString("| Property | Type | Description |\n")
		sb.WriteString("|----------|------|-------------|\n")

		for _, p := range t.Properties {
			desc := escapeMarkdownTableCell(p.Description)
			fmt.Fprintf(sb, "| `.%s` | %s | %s |\n", p.Name, p.Type, desc)
		}
		sb.WriteString("\n")
	}

	// Methods table
	if len(t.Methods) > 0 {
		sb.WriteString("#### Methods\n\n")
		sb.WriteString("| Method | Description |\n")
		sb.WriteString("|--------|-------------|\n")

		for _, m := range t.Methods {
			sig := fmt.Sprintf(".%s(%s)", m.Name, arityToParams(m.Arity))
			desc := escapeMarkdownTableCell(m.Description)
			fmt.Fprintf(sb, "| `%s` | %s |\n", sig, desc)
		}
		sb.WriteString("\n")
	}

	if len(t.Properties) == 0 && len(t.Methods) == 0 {
		sb.WriteString("*No properties or methods.*\n\n")
	}
}

// formatModuleSchemaMarkdown formats a ModuleSchema as Markdown
func formatModuleSchemaMarkdown(sb *strings.Builder, m ModuleSchema) {
	fmt.Fprintf(sb, "### %s\n\n", m.Name)

	if m.Description != "" {
		fmt.Fprintf(sb, "%s\n\n", m.Description)
	}

	if len(m.Exports) == 0 {
		sb.WriteString("*No documented exports.*\n\n")
		return
	}

	// Separate constants and functions
	var constants, functions []ExportEntry
	for _, e := range m.Exports {
		if e.Kind == "constant" {
			constants = append(constants, e)
		} else {
			functions = append(functions, e)
		}
	}

	// Constants
	if len(constants) > 0 {
		sb.WriteString("#### Constants\n\n")
		sb.WriteString("| Name | Description |\n")
		sb.WriteString("|------|-------------|\n")

		for _, e := range constants {
			desc := escapeMarkdownTableCell(e.Description)
			fmt.Fprintf(sb, "| `%s` | %s |\n", e.Name, desc)
		}
		sb.WriteString("\n")
	}

	// Functions
	if len(functions) > 0 {
		sb.WriteString("#### Functions\n\n")
		sb.WriteString("| Function | Description |\n")
		sb.WriteString("|----------|-------------|\n")

		for _, e := range functions {
			sig := fmt.Sprintf("%s(%s)", e.Name, arityToParams(e.Arity))
			desc := escapeMarkdownTableCell(e.Description)
			fmt.Fprintf(sb, "| `%s` | %s |\n", sig, desc)
		}
		sb.WriteString("\n")
	}
}

// escapeMarkdownTableCell escapes characters that would break markdown table cells
func escapeMarkdownTableCell(s string) string {
	// Replace pipe characters which break table cells
	s = strings.ReplaceAll(s, "|", "\\|")
	// Replace newlines which break table rows
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}
