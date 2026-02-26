package help

import (
	"strings"
	"testing"
)

// TestFormatMarkdownType tests Markdown output for a type result
func TestFormatMarkdownType(t *testing.T) {
	result, err := DescribeTopic("string")
	if err != nil {
		t.Fatalf("DescribeTopic('string') returned error: %v", err)
	}

	md := FormatMarkdown(result)

	// Should have a heading with the type name
	if !strings.Contains(md, "## String") {
		t.Error("expected '## String' heading")
	}

	// Should have a Methods section
	if !strings.Contains(md, "### Methods") {
		t.Error("expected '### Methods' section")
	}

	// Should have a table header
	if !strings.Contains(md, "| Method | Description |") {
		t.Error("expected method table header")
	}

	// Should contain a known method
	if !strings.Contains(md, "toUpper") {
		t.Error("expected 'toUpper' method in output")
	}
}

// TestFormatMarkdownTypeWithProperties tests Markdown for types that have properties
func TestFormatMarkdownTypeWithProperties(t *testing.T) {
	result, err := DescribeTopic("datetime")
	if err != nil {
		t.Fatalf("DescribeTopic('datetime') returned error: %v", err)
	}

	md := FormatMarkdown(result)

	if !strings.Contains(md, "## Datetime") {
		t.Error("expected '## Datetime' heading")
	}

	// Datetime should have properties
	if !strings.Contains(md, "### Properties") {
		t.Error("expected '### Properties' section for datetime")
	}

	if !strings.Contains(md, "| Property | Type | Description |") {
		t.Error("expected property table header")
	}
}

// TestFormatMarkdownModule tests Markdown output for a module result
func TestFormatMarkdownModule(t *testing.T) {
	result, err := DescribeTopic("@std/math")
	if err != nil {
		t.Fatalf("DescribeTopic('@std/math') returned error: %v", err)
	}

	md := FormatMarkdown(result)

	if !strings.Contains(md, "## @std/math") {
		t.Error("expected '## @std/math' heading")
	}

	// Math module should have constants (like PI)
	if !strings.Contains(md, "### Constants") {
		t.Error("expected '### Constants' section")
	}

	if !strings.Contains(md, "PI") {
		t.Error("expected 'PI' constant in output")
	}
}

// TestFormatMarkdownModuleWithFunctions tests module with function exports
func TestFormatMarkdownModuleWithFunctions(t *testing.T) {
	result, err := DescribeTopic("@std/math")
	if err != nil {
		t.Fatalf("DescribeTopic('@std/math') returned error: %v", err)
	}

	md := FormatMarkdown(result)

	if !strings.Contains(md, "### Functions") {
		t.Error("expected '### Functions' section")
	}

	if !strings.Contains(md, "| Function | Description |") {
		t.Error("expected function table header")
	}
}

// TestFormatMarkdownBuiltin tests Markdown output for a single builtin
func TestFormatMarkdownBuiltin(t *testing.T) {
	result, err := DescribeTopic("JSON")
	if err != nil {
		t.Fatalf("DescribeTopic('JSON') returned error: %v", err)
	}

	md := FormatMarkdown(result)

	if !strings.Contains(md, "### JSON") {
		t.Error("expected '### JSON' heading")
	}

	// Should have a code block with signature
	if !strings.Contains(md, "```parsley") {
		t.Error("expected parsley code block with signature")
	}

	if !strings.Contains(md, "JSON(") {
		t.Error("expected 'JSON(' signature")
	}

	// Should have metadata
	if !strings.Contains(md, "**Arity:**") {
		t.Error("expected '**Arity:**' field")
	}

	if !strings.Contains(md, "**Category:**") {
		t.Error("expected '**Category:**' field")
	}
}

// TestFormatMarkdownBuiltinList tests Markdown output for the builtins list
func TestFormatMarkdownBuiltinList(t *testing.T) {
	result, err := DescribeTopic("builtins")
	if err != nil {
		t.Fatalf("DescribeTopic('builtins') returned error: %v", err)
	}

	md := FormatMarkdown(result)

	if !strings.Contains(md, "## Builtin Functions") {
		t.Error("expected '## Builtin Functions' heading")
	}

	// Should have category headings
	if !strings.Contains(md, "### File/Data Loading") {
		t.Error("expected '### File/Data Loading' category heading")
	}

	// Should have table structures
	if !strings.Contains(md, "| Function | Description |") {
		t.Error("expected function table header")
	}

	// Should contain known builtins
	if !strings.Contains(md, "JSON(") {
		t.Error("expected 'JSON(' in output")
	}

	if !strings.Contains(md, "log(") {
		t.Error("expected 'log(' in output")
	}
}

// TestFormatMarkdownOperatorList tests Markdown output for the operators list
func TestFormatMarkdownOperatorList(t *testing.T) {
	result, err := DescribeTopic("operators")
	if err != nil {
		t.Fatalf("DescribeTopic('operators') returned error: %v", err)
	}

	md := FormatMarkdown(result)

	if !strings.Contains(md, "## Operators") {
		t.Error("expected '## Operators' heading")
	}

	// Should have category headings
	if !strings.Contains(md, "### Arithmetic") {
		t.Error("expected '### Arithmetic' category heading")
	}

	if !strings.Contains(md, "### Comparison") {
		t.Error("expected '### Comparison' category heading")
	}

	if !strings.Contains(md, "### Logical") {
		t.Error("expected '### Logical' category heading")
	}

	// Should have table structures
	if !strings.Contains(md, "| Operator | Description |") {
		t.Error("expected operator table header")
	}

	// Should contain known operators
	for _, op := range []string{"+", "-", "==", "!="} {
		if !strings.Contains(md, op) {
			t.Errorf("expected operator %q in output", op)
		}
	}
}

// TestFormatMarkdownTypeList tests Markdown output for the types list
func TestFormatMarkdownTypeList(t *testing.T) {
	result, err := DescribeTopic("types")
	if err != nil {
		t.Fatalf("DescribeTopic('types') returned error: %v", err)
	}

	md := FormatMarkdown(result)

	if !strings.Contains(md, "## Available Types") {
		t.Error("expected '## Available Types' heading")
	}

	// Should have category groupings
	if !strings.Contains(md, "### Primitives") {
		t.Error("expected '### Primitives' section")
	}

	if !strings.Contains(md, "### Collections") {
		t.Error("expected '### Collections' section")
	}

	// Should list known types
	for _, typeName := range []string{"string", "integer", "float", "array", "dictionary"} {
		if !strings.Contains(md, typeName) {
			t.Errorf("expected type %q in output", typeName)
		}
	}
}

// TestFormatMarkdownAll tests Markdown output for the "all" topic
func TestFormatMarkdownAll(t *testing.T) {
	result, err := DescribeTopic("all")
	if err != nil {
		t.Fatalf("DescribeTopic('all') returned error: %v", err)
	}

	md := FormatMarkdown(result)

	// Should have the main title
	if !strings.Contains(md, "# Parsley API Reference") {
		t.Error("expected '# Parsley API Reference' title")
	}

	// Should have auto-generated notice
	if !strings.Contains(md, "auto-generated") {
		t.Error("expected auto-generated notice")
	}

	// Should have table of contents
	if !strings.Contains(md, "## Table of Contents") {
		t.Error("expected '## Table of Contents'")
	}

	// Should have all major sections
	if !strings.Contains(md, "## Types") {
		t.Error("expected '## Types' section")
	}

	if !strings.Contains(md, "## Builtin Functions") {
		t.Error("expected '## Builtin Functions' section")
	}

	if !strings.Contains(md, "## Operators") {
		t.Error("expected '## Operators' section")
	}

	if !strings.Contains(md, "## Standard Library") {
		t.Error("expected '## Standard Library' section")
	}
}

// TestFormatMarkdownAllContainsTypes tests that the "all" output includes individual types
func TestFormatMarkdownAllContainsTypes(t *testing.T) {
	result, err := DescribeTopic("all")
	if err != nil {
		t.Fatalf("DescribeTopic('all') returned error: %v", err)
	}

	md := FormatMarkdown(result)

	// Should contain type-specific sections with methods
	for _, typeName := range []string{"String", "Array", "Dictionary", "Integer", "Float"} {
		if !strings.Contains(md, "### "+typeName) {
			t.Errorf("expected '### %s' type section in all output", typeName)
		}
	}
}

// TestFormatMarkdownWithFrontmatter tests YAML frontmatter generation
func TestFormatMarkdownWithFrontmatter(t *testing.T) {
	result, err := DescribeTopic("string")
	if err != nil {
		t.Fatalf("DescribeTopic('string') returned error: %v", err)
	}

	md := FormatMarkdownWithFrontmatter(result, "String Type Reference")

	// Should start with YAML frontmatter
	if !strings.HasPrefix(md, "---\n") {
		t.Error("expected output to start with YAML frontmatter '---'")
	}

	if !strings.Contains(md, `title: "String Type Reference"`) {
		t.Error("expected title in frontmatter")
	}

	if !strings.Contains(md, "generated:") {
		t.Error("expected generated timestamp in frontmatter")
	}

	// Should still contain the regular content after frontmatter
	if !strings.Contains(md, "## String") {
		t.Error("expected type content after frontmatter")
	}
}

// TestFormatMarkdownEscapesTableCells tests that pipe characters are escaped
func TestFormatMarkdownEscapesTableCells(t *testing.T) {
	// Test the helper directly
	escaped := escapeMarkdownTableCell("a | b")
	if escaped != `a \| b` {
		t.Errorf("escapeMarkdownTableCell(\"a | b\") = %q, want %q", escaped, `a \| b`)
	}

	escaped = escapeMarkdownTableCell("line1\nline2")
	if escaped != "line1 line2" {
		t.Errorf("escapeMarkdownTableCell with newline = %q, want %q", escaped, "line1 line2")
	}
}

// TestFormatMarkdownMultipleTypes tests that different types produce distinct output
func TestFormatMarkdownMultipleTypes(t *testing.T) {
	types := []string{"string", "array", "integer", "float", "dictionary"}

	outputs := make(map[string]string)
	for _, typeName := range types {
		result, err := DescribeTopic(typeName)
		if err != nil {
			t.Fatalf("DescribeTopic(%q) returned error: %v", typeName, err)
		}
		outputs[typeName] = FormatMarkdown(result)
	}

	// Each type should produce unique output
	for i, t1 := range types {
		for _, t2 := range types[i+1:] {
			if outputs[t1] == outputs[t2] {
				t.Errorf("FormatMarkdown for %q and %q produced identical output", t1, t2)
			}
		}
	}
}

// TestFormatMarkdownNonEmpty tests that no topic produces empty output
func TestFormatMarkdownNonEmpty(t *testing.T) {
	topics := []string{"string", "array", "builtins", "operators", "types", "@std/math", "JSON", "all"}

	for _, topic := range topics {
		t.Run(topic, func(t *testing.T) {
			result, err := DescribeTopic(topic)
			if err != nil {
				t.Fatalf("DescribeTopic(%q) returned error: %v", topic, err)
			}

			md := FormatMarkdown(result)
			if len(md) == 0 {
				t.Error("FormatMarkdown produced empty output")
			}

			// Should contain at least one heading
			if !strings.Contains(md, "#") {
				t.Error("FormatMarkdown output contains no headings")
			}
		})
	}
}

// TestFormatMarkdownModuleNoExports tests formatting for a module with no documented exports
func TestFormatMarkdownModuleNoExports(t *testing.T) {
	result := &TopicResult{
		Kind:        "module",
		Name:        "@test/empty",
		Description: "An empty module",
		Exports:     nil,
	}

	md := FormatMarkdown(result)

	if !strings.Contains(md, "## @test/empty") {
		t.Error("expected module heading")
	}

	if !strings.Contains(md, "An empty module") {
		t.Error("expected description")
	}

	if !strings.Contains(md, "No documented exports") {
		t.Error("expected 'No documented exports' message")
	}
}

// TestFormatMarkdownTypeNoMethods tests formatting for a type with no methods
func TestFormatMarkdownTypeNoMethods(t *testing.T) {
	result := &TopicResult{
		Kind:       "type",
		Name:       "empty",
		Methods:    nil,
		Properties: nil,
	}

	md := FormatMarkdown(result)

	if !strings.Contains(md, "## Empty") {
		t.Error("expected type heading")
	}

	if !strings.Contains(md, "No properties or methods") {
		t.Error("expected 'No properties or methods' message")
	}
}
