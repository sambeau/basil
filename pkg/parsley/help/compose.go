package help

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ComposeReference generates a complete reference document by combining
// a template file with hand-written fragments and auto-generated API sections.
//
// Template syntax:
//
//	{{include "filename.md"}}  - Insert contents of a fragment file
//	{{generate "section"}}     - Insert auto-generated API section
//	{{template "name"}}        - Insert a predefined template (e.g., "toc")
//
// Generated sections:
//   - "type-methods"     - All type method tables
//   - "builtins"         - All builtin function tables
//   - "operators"        - Operator tables by category
//   - "modules"          - Standard library module tables
//   - "type-summary"     - Type summary list
//   - "method-reference" - Compact method reference tables
//   - "keywords"         - Reserved keywords list
func ComposeReference(templatePath string, fragmentsDir string) (string, error) {
	// Read template file
	tmplContent, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("reading template: %w", err)
	}

	// Get API data for generated sections
	allResult, err := DescribeTopic("all")
	if err != nil {
		return "", fmt.Errorf("getting API data: %w", err)
	}

	// Process template
	output, err := processTemplate(string(tmplContent), fragmentsDir, allResult)
	if err != nil {
		return "", err
	}

	return output, nil
}

// processTemplate processes a template string, expanding all directives
func processTemplate(tmpl string, fragmentsDir string, apiData *TopicResult) (string, error) {
	var result strings.Builder

	// Regex patterns for template directives
	includePattern := regexp.MustCompile(`\{\{include\s+"([^"]+)"\}\}`)
	generatePattern := regexp.MustCompile(`\{\{generate\s+"([^"]+)"\}\}`)
	templatePattern := regexp.MustCompile(`\{\{template\s+"([^"]+)"\}\}`)

	lines := strings.SplitSeq(tmpl, "\n")
	for line := range lines {
		// Check for {{include "filename"}}
		if match := includePattern.FindStringSubmatch(line); match != nil {
			filename := match[1]
			content, err := readFragment(fragmentsDir, filename)
			if err != nil {
				return "", fmt.Errorf("including %q: %w", filename, err)
			}
			result.WriteString(content)
			continue
		}

		// Check for {{generate "section"}}
		if match := generatePattern.FindStringSubmatch(line); match != nil {
			section := match[1]
			content, err := generateSection(section, apiData)
			if err != nil {
				return "", fmt.Errorf("generating %q: %w", section, err)
			}
			result.WriteString(content)
			continue
		}

		// Check for {{template "name"}}
		if match := templatePattern.FindStringSubmatch(line); match != nil {
			name := match[1]
			content := getBuiltinTemplate(name)
			result.WriteString(content)
			continue
		}

		// Regular line - pass through
		result.WriteString(line)
		result.WriteString("\n")
	}

	// Remove trailing newline added by loop
	output := strings.TrimSuffix(result.String(), "\n")

	return output, nil
}

// readFragment reads a fragment file from the fragments directory
func readFragment(fragmentsDir, filename string) (string, error) {
	path := filepath.Join(fragmentsDir, filename)
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// generateSection generates markdown for a specific API section
func generateSection(section string, apiData *TopicResult) (string, error) {
	var sb strings.Builder

	switch section {
	case "type-methods":
		generateTypeMethods(&sb, apiData)
	case "builtins":
		generateBuiltins(&sb, apiData)
	case "operators":
		generateOperators(&sb, apiData)
	case "modules":
		generateModules(&sb, apiData)
	case "type-summary":
		generateTypeSummary(&sb, apiData)
	case "method-reference":
		generateMethodReference(&sb, apiData)
	case "keywords":
		generateKeywords(&sb)
	default:
		return "", fmt.Errorf("unknown section: %s", section)
	}

	return sb.String(), nil
}

// generateTypeMethods generates method tables for all types
func generateTypeMethods(sb *strings.Builder, apiData *TopicResult) {
	for _, t := range apiData.Types {
		if len(t.Methods) == 0 && len(t.Properties) == 0 {
			continue
		}

		fmt.Fprintf(sb, "### %s\n\n", capitalizeFirst(t.Name))

		if len(t.Properties) > 0 {
			sb.WriteString("#### Properties\n\n")
			sb.WriteString("| Property | Type | Description |\n")
			sb.WriteString("|----------|------|-------------|\n")
			for _, p := range t.Properties {
				fmt.Fprintf(sb, "| `.%s` | `%s` | %s |\n",
					p.Name, p.Type, escapeMarkdownTableCell(p.Description))
			}
			sb.WriteString("\n")
		}

		if len(t.Methods) > 0 {
			sb.WriteString("#### Methods\n\n")
			sb.WriteString("| Method | Description |\n")
			sb.WriteString("|--------|-------------|\n")
			for _, m := range t.Methods {
				sig := fmt.Sprintf(".%s(%s)", m.Name, arityToParams(m.Arity))
				fmt.Fprintf(sb, "| `%s` | %s |\n",
					sig, escapeMarkdownTableCell(m.Description))
			}
			sb.WriteString("\n")
		}
	}
}

// generateBuiltins generates builtin function tables grouped by category
func generateBuiltins(sb *strings.Builder, apiData *TopicResult) {
	result := &TopicResult{
		Kind:     "builtin-list",
		Builtins: apiData.Builtins,
	}
	// Reuse existing formatter but strip the top-level heading
	md := FormatMarkdown(result)
	// Remove the "## Builtin Functions" heading since we're inside a section
	md = strings.TrimPrefix(md, "## Builtin Functions\n\n")
	sb.WriteString(md)
}

// generateOperators generates operator tables grouped by category
func generateOperators(sb *strings.Builder, apiData *TopicResult) {
	result := &TopicResult{
		Kind:      "operator-list",
		Operators: apiData.Operators,
	}
	md := FormatMarkdown(result)
	md = strings.TrimPrefix(md, "## Operators\n\n")
	sb.WriteString(md)
}

// generateModules generates standard library module tables
func generateModules(sb *strings.Builder, apiData *TopicResult) {
	for _, m := range apiData.Modules {
		fmt.Fprintf(sb, "### %s\n\n", m.Name)

		if m.Description != "" {
			fmt.Fprintf(sb, "%s\n\n", m.Description)
		}

		if len(m.Exports) == 0 {
			sb.WriteString("*No documented exports.*\n\n")
			continue
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

		if len(constants) > 0 {
			sb.WriteString("#### Constants\n\n")
			sb.WriteString("| Name | Description |\n")
			sb.WriteString("|------|-------------|\n")
			for _, e := range constants {
				fmt.Fprintf(sb, "| `%s` | %s |\n",
					e.Name, escapeMarkdownTableCell(e.Description))
			}
			sb.WriteString("\n")
		}

		if len(functions) > 0 {
			sb.WriteString("#### Functions\n\n")
			sb.WriteString("| Function | Description |\n")
			sb.WriteString("|----------|-------------|\n")
			for _, e := range functions {
				sig := fmt.Sprintf("%s(%s)", e.Name, arityToParams(e.Arity))
				fmt.Fprintf(sb, "| `%s` | %s |\n",
					sig, escapeMarkdownTableCell(e.Description))
			}
			sb.WriteString("\n")
		}
	}
}

// generateTypeSummary generates a summary list of all types
func generateTypeSummary(sb *strings.Builder, apiData *TopicResult) {
	sb.WriteString("| Type | Properties | Methods |\n")
	sb.WriteString("|------|------------|--------|\n")

	for _, t := range apiData.Types {
		fmt.Fprintf(sb, "| `%s` | %d | %d |\n",
			t.Name, len(t.Properties), len(t.Methods))
	}
	sb.WriteString("\n")
}

// generateMethodReference generates compact method reference tables
func generateMethodReference(sb *strings.Builder, apiData *TopicResult) {
	for _, t := range apiData.Types {
		if len(t.Methods) == 0 {
			continue
		}

		fmt.Fprintf(sb, "### %s Methods (%d methods)\n\n",
			capitalizeFirst(t.Name), len(t.Methods))

		sb.WriteString("| Method | Description |\n")
		sb.WriteString("|--------|-------------|\n")

		for _, m := range t.Methods {
			sig := fmt.Sprintf(".%s(%s)", m.Name, arityToParams(m.Arity))
			fmt.Fprintf(sb, "| `%s` | %s |\n",
				sig, escapeMarkdownTableCell(m.Description))
		}
		sb.WriteString("\n")
	}
}

// generateKeywords generates the reserved keywords list
func generateKeywords(sb *strings.Builder) {
	// These should ideally come from the lexer, but for now we hardcode
	// the known keywords to match reference.md
	keywords := []string{
		"let", "var", "const", "fn", "if", "else", "for", "in",
		"return", "true", "false", "null", "and", "or", "not",
		"is", "try", "check", "import", "export", "computed",
		"stop", "skip", "with",
	}

	sb.WriteString("```\n")
	for i, kw := range keywords {
		sb.WriteString(kw)
		if i < len(keywords)-1 {
			if (i+1)%8 == 0 {
				sb.WriteString("\n")
			} else {
				sb.WriteString("  ")
			}
		}
	}
	sb.WriteString("\n```\n")
}

// getBuiltinTemplate returns content for built-in template names
func getBuiltinTemplate(name string) string {
	switch name {
	case "toc":
		return `## Table of Contents

1. [Literals](#1-literals)
2. [Operators](#2-operators)
3. [Control Flow](#3-control-flow)
4. [Statements](#4-statements)
5. [Type Methods](#5-type-methods)
6. [Builtin Functions](#6-builtin-functions)
7. [Standard Library](#7-standard-library)
8. [Tags (HTML/XML)](#8-tags-htmlxml)
9. [Comments](#9-comments)
10. [Error Handling](#10-error-handling)

**Appendices**:
- [A. Type Summary](#appendix-a-type-summary)
- [B. Method Reference](#appendix-b-method-reference)
`
	default:
		return fmt.Sprintf("<!-- unknown template: %s -->\n", name)
	}
}

// capitalizeFirst capitalizes the first letter of a string
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
