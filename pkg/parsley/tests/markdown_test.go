package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// Helper function for evaluating Parsley code with a filename context
func testEvalMDWithFilename(input string, filename string) evaluator.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	env := evaluator.NewEnvironment()
	env.Filename = filename
	// Reads are allowed by default, no special security needed
	return evaluator.Eval(program, env)
}

// TestMarkdownBasic tests basic markdown parsing
func TestMarkdownBasic(t *testing.T) {
	// Create a temp directory for test files
	tmpDir, err := os.MkdirTemp("", "parsley-md-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a simple markdown file without frontmatter
	mdContent := `# Hello World

This is a paragraph.

- Item 1
- Item 2
`
	mdPath := filepath.Join(tmpDir, "simple.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatalf("Failed to write markdown file: %v", err)
	}

	// Test file path for relative path resolution
	testFilePath := filepath.Join(tmpDir, "test.pars")

	// Test reading markdown
	code := `let post <== MD(@./simple.md); post.html`
	result := testEvalMDWithFilename(code, testFilePath)

	if result == nil {
		t.Fatalf("Result is nil")
	}

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("Evaluation error: %s", result.Inspect())
	}

	html := result.Inspect()
	if !strings.Contains(html, "<h1>Hello World</h1>") {
		t.Errorf("Expected h1 tag, got: %s", html)
	}
	if !strings.Contains(html, "<li>Item 1</li>") {
		t.Errorf("Expected list items, got: %s", html)
	}
}

// TestMarkdownWithFrontmatter tests markdown with YAML frontmatter
func TestMarkdownWithFrontmatter(t *testing.T) {
	// Create a temp directory for test files
	tmpDir, err := os.MkdirTemp("", "parsley-md-frontmatter-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a markdown file with frontmatter
	mdContent := `---
title: My Blog Post
author: John Doe
tags:
  - go
  - parsley
draft: false
---
# Content

This is the blog content.
`
	mdPath := filepath.Join(tmpDir, "blog.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatalf("Failed to write markdown file: %v", err)
	}

	testFilePath := filepath.Join(tmpDir, "test.pars")

	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "access title",
			code:     `let post <== MD(@./blog.md); post.md.title`,
			expected: "My Blog Post",
		},
		{
			name:     "access author",
			code:     `let post <== MD(@./blog.md); post.md.author`,
			expected: "John Doe",
		},
		{
			name:     "access draft",
			code:     `let post <== MD(@./blog.md); post.md.draft`,
			expected: "false",
		},
		{
			name:     "access tags array",
			code:     `let post <== MD(@./blog.md); post.md.tags[0]`,
			expected: "go",
		},
		{
			name:     "html contains content",
			code:     `let post <== MD(@./blog.md); post.html`,
			expected: "<h1>Content</h1>",
		},
		{
			name:     "raw contains markdown",
			code:     `let post <== MD(@./blog.md); post.raw`,
			expected: "# Content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalMDWithFilename(tt.code, testFilePath)

			if result == nil {
				t.Fatalf("Result is nil")
			}

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("Evaluation error: %s", result.Inspect())
			}

			if !strings.Contains(result.Inspect(), tt.expected) {
				t.Errorf("Expected to contain '%s', got: %s", tt.expected, result.Inspect())
			}
		})
	}
}

// TestMarkdownAsComponent tests using markdown in templates
func TestMarkdownAsComponent(t *testing.T) {
	// Create a temp directory for test files
	tmpDir, err := os.MkdirTemp("", "parsley-md-component-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a markdown file with frontmatter
	mdContent := `---
title: Hello World
---
This is **bold** text.
`
	mdPath := filepath.Join(tmpDir, "post.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatalf("Failed to write markdown file: %v", err)
	}

	testFilePath := filepath.Join(tmpDir, "test.pars")

	// Test using markdown in a template
	code := `let post <== MD(@./post.md)
<article>
  <h1>post.md.title</h1>
  <div class="content">post.html</div>
</article>`

	result := testEvalMDWithFilename(code, testFilePath)

	if result == nil {
		t.Fatalf("Result is nil")
	}

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("Evaluation error: %s", result.Inspect())
	}

	html := result.Inspect()
	if !strings.Contains(html, "<h1>Hello World</h1>") {
		t.Errorf("Expected title in h1, got: %s", html)
	}
	if !strings.Contains(html, "<strong>bold</strong>") {
		t.Errorf("Expected bold text, got: %s", html)
	}
}

func TestMarkdownInterpolatesRawTemplates(t *testing.T) {
	// Create a temp directory for test files
	tmpDir, err := os.MkdirTemp("", "parsley-md-interpolation-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mdContent := `# @{title}

Value: @{value}
Literal: \@{escaped}
`
	mdPath := filepath.Join(tmpDir, "dynamic.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatalf("Failed to write markdown file: %v", err)
	}

	testFilePath := filepath.Join(tmpDir, "test.pars")
	code := `let title = "Hello"; let value = 42; let escaped = "ignored"; let post <== MD(@./dynamic.md); post.html`
	result := testEvalMDWithFilename(code, testFilePath)

	if result == nil {
		t.Fatalf("Result is nil")
	}

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("Evaluation error: %s", result.Inspect())
	}

	html := result.Inspect()
	if !strings.Contains(html, "<h1>Hello</h1>") {
		t.Errorf("Expected interpolated title, got: %s", html)
	}
	if !strings.Contains(html, "Value: 42") {
		t.Errorf("Expected interpolated value, got: %s", html)
	}
	// \@{escaped} is escaped, so it should remain as literal @{escaped} in output (not interpolated)
	if !strings.Contains(html, "Literal: @{escaped}") {
		t.Errorf("Expected literal @{escaped}, got: %s", html)
	}
}

// TestMarkdownStringParsing tests parsing markdown from strings
func TestMarkdownStringParsing(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		contains []string
	}{
		{
			name:     "basic markdown",
			code:     `markdown("# Hello World\n\nParagraph").html`,
			contains: []string{"<h1>Hello World</h1>", "<p>Paragraph</p>"},
		},
		{
			name:     "markdown with frontmatter",
			code:     `markdown("---\ntitle: Test\n---\n# Body").md.title`,
			contains: []string{"Test"},
		},
		{
			name:     "raw field",
			code:     `markdown("# Hello").raw`,
			contains: []string{"# Hello"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.code)
			p := parser.New(l)
			program := p.ParseProgram()
			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)

			if result == nil {
				t.Fatalf("Result is nil")
			}

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("Evaluation error: %s", result.Inspect())
			}

			output := result.Inspect()
			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected to contain '%s', got: %s", expected, output)
				}
			}
		})
	}
}

// TestParseMarkdownMethod tests the string.parseMarkdown() method
func TestParseMarkdownMethod(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		contains []string
	}{
		{
			name:     "basic usage",
			code:     `"# Hello".parseMarkdown().html`,
			contains: []string{"<h1>Hello</h1>"},
		},
		{
			name:     "with variable",
			code:     `let md = "## Test\n\nBody"; md.parseMarkdown().html`,
			contains: []string{"<h2>Test</h2>", "<p>Body</p>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.code)
			p := parser.New(l)
			program := p.ParseProgram()
			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)

			if result == nil {
				t.Fatalf("Result is nil")
			}

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("Evaluation error: %s", result.Inspect())
			}

			output := result.Inspect()
			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected to contain '%s', got: %s", expected, output)
				}
			}
		})
	}
}

// TestMarkdownHeadingIDs tests the {ids: true} option
func TestMarkdownHeadingIDs(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		contains    []string
		notContains []string
	}{
		{
			name:     "basic heading ID",
			code:     `markdown("# Hello World", {ids: true}).html`,
			contains: []string{`id="hello-world"`},
		},
		{
			name:     "multiple headings",
			code:     `markdown("# Intro\n\n## Getting Started\n\n## Features", {ids: true}).html`,
			contains: []string{`id="intro"`, `id="getting-started"`, `id="features"`},
		},
		{
			name:     "duplicate headings",
			code:     `markdown("# Intro\n\n# Intro", {ids: true}).html`,
			contains: []string{`id="intro"`, `id="intro-1"`},
		},
		{
			name:     "special characters",
			code:     `markdown("# Hello, World!", {ids: true}).html`,
			contains: []string{`id="hello-world"`},
		},
		{
			name:     "punctuation edge cases - parentheses with operators",
			code:     `markdown("# Subtraction (-)\n\n## Addition (+)", {ids: true}).html`,
			contains: []string{`id="subtraction--"`, `id="addition-"`},
		},
		{
			name:        "default no IDs",
			code:        `markdown("# Hello").html`,
			notContains: []string{`id=`},
		},
		{
			name:     "parseMarkdown with IDs",
			code:     `"# Test".parseMarkdown({ids: true}).html`,
			contains: []string{`id="test"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.code)
			p := parser.New(l)
			program := p.ParseProgram()
			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)

			if result == nil {
				t.Fatalf("Result is nil")
			}

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("Evaluation error: %s", result.Inspect())
			}

			output := result.Inspect()
			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected to contain '%s', got: %s", expected, output)
				}
			}
			for _, notExpected := range tt.notContains {
				if strings.Contains(output, notExpected) {
					t.Errorf("Expected NOT to contain '%s', got: %s", notExpected, output)
				}
			}
		})
	}
}

// TestMarkdownFileWithHeadingIDs tests MD(@path, {ids: true}) with file loading
func TestMarkdownFileWithHeadingIDs(t *testing.T) {
	// Create a temp directory for test files
	tmpDir, err := os.MkdirTemp("", "parsley-md-file-ids-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a markdown file with multiple headings
	mdContent := `# Getting Started

## Installation

Follow these steps.

## Configuration

Set up your config.
`
	mdPath := filepath.Join(tmpDir, "guide.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatalf("Failed to write markdown file: %v", err)
	}

	testFilePath := filepath.Join(tmpDir, "test.pars")

	tests := []struct {
		name        string
		code        string
		contains    []string
		notContains []string
	}{
		{
			name:     "MD with ids option",
			code:     `let doc <== MD(@./guide.md, {ids: true}); doc.html`,
			contains: []string{`id="getting-started"`, `id="installation"`, `id="configuration"`},
		},
		{
			name:        "MD without ids option",
			code:        `let doc <== MD(@./guide.md); doc.html`,
			notContains: []string{`id=`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalMDWithFilename(tt.code, testFilePath)

			if result == nil {
				t.Fatalf("Result is nil")
			}

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("Evaluation error: %s", result.Inspect())
			}

			output := result.Inspect()
			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected to contain '%s', got: %s", expected, output)
				}
			}
			for _, notExpected := range tt.notContains {
				if strings.Contains(output, notExpected) {
					t.Errorf("Expected NOT to contain '%s', got: %s", notExpected, output)
				}
			}
		})
	}
}
