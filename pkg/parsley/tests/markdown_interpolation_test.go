package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
)

// TestMarkdownInterpolation tests @{expr} interpolation in markdown files
func TestMarkdownInterpolation(t *testing.T) {
	// Create a temp directory for test files
	tmpDir, err := os.MkdirTemp("", "parsley-md-interp-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name     string
		markdown string
		setup    string
		expected []string // Multiple strings that should appear in output
	}{
		{
			name:     "simple variable interpolation",
			markdown: "The answer is @{answer}",
			setup:    "let answer = 42",
			expected: []string{"<p>The answer is 42</p>"},
		},
		{
			name:     "expression interpolation",
			markdown: "Sum: @{10 + 20}",
			setup:    "",
			expected: []string{"<p>Sum: 30</p>"},
		},
		{
			name:     "function call interpolation",
			markdown: "# @{title.toUpper()}",
			setup:    `let title = "hello world"`,
			expected: []string{"<h1", "HELLO WORLD</h1>"},
		},
		{
			name: "multiple interpolations",
			markdown: `Count: @{count}
Value: @{value}`,
			setup: `let count = 5
let value = "test"`,
			expected: []string{"Count: 5", "Value: test"},
		},
		{
			name:     "nested braces",
			markdown: "Result: @{{a: 10}.a}",
			setup:    "",
			expected: []string{"<p>Result: 10</p>"},
		},
		{
			name:     "string method interpolation",
			markdown: "Upper: @{\"hello\".toUpper()}",
			setup:    "",
			expected: []string{"<p>Upper: HELLO</p>"},
		},
		{
			name:     "array interpolation",
			markdown: "First: @{items[0]}",
			setup:    `let items = [1, 2, 3]`,
			expected: []string{"<p>First: 1</p>"},
		},
		{
			name:     "dictionary interpolation",
			markdown: "Name: @{person.name}",
			setup:    `let person = {name: "Alice", age: 30}`,
			expected: []string{"<p>Name: Alice</p>"},
		},
		{
			name:     "interpolation in heading",
			markdown: "# Report for @{year}",
			setup:    `let year = 2024`,
			expected: []string{"<h1", ">Report for 2024</h1>"},
		},
		{
			name: "interpolation in list",
			markdown: `- Item @{1}
- Item @{2}
- Item @{3}`,
			setup:    "",
			expected: []string{"<li>Item 1</li>", "<li>Item 2</li>", "<li>Item 3</li>"},
		},
		{
			name:     "comparison operators",
			markdown: "Result: @{10 > 5}",
			setup:    "",
			expected: []string{"<p>Result: true</p>"},
		},
		{
			name:     "less than operator",
			markdown: "Check: @{3 < 10}",
			setup:    "",
			expected: []string{"<p>Check: true</p>"},
		},
		{
			name:     "complex comparison",
			markdown: "Value: @{if (x > 10) { \"big\" } else { \"small\" }}",
			setup:    "let x = 15",
			expected: []string{"<p>Value: big</p>"},
		},
		{
			name:     "HTML tags in expression",
			markdown: "HTML: @{\"<div>Hello</div>\"}",
			setup:    "",
			expected: []string{"<p>HTML: <div>Hello</div></p>"},
		},
		{
			name:     "nested braces with comparison",
			markdown: "Adults: @{{age: 25}.age > 18}",
			setup:    "",
			expected: []string{"<p>Adults: true</p>"},
		},
		{
			name:     "function with nested braces and tags",
			markdown: `Items: @{[1,2,3].map(fn(x) { "<li>" + x + "</li>" }).join("")}`,
			setup:    "",
			expected: []string{"<p>Items: <li>1</li><li>2</li><li>3</li></p>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create markdown file
			mdPath := filepath.Join(tmpDir, "test.md")
			if err := os.WriteFile(mdPath, []byte(tt.markdown), 0644); err != nil {
				t.Fatalf("Failed to write markdown file: %v", err)
			}

			// Create test file path for environment
			testFilePath := filepath.Join(tmpDir, "test.pars")

			// Build test code
			code := tt.setup
			if code != "" {
				code += "\n"
			}
			code += `let mdfile = markdown(@./test.md)
let m = null
m <== mdfile
m.html`

			result := testEvalMDWithFilename(code, testFilePath)

			if result == nil {
				t.Fatalf("Result is nil")
			}

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("Evaluation error: %s", result.Inspect())
			}

			html := result.Inspect()

			// Check all expected strings are in output
			for _, expected := range tt.expected {
				if !strings.Contains(html, expected) {
					t.Errorf("Expected HTML to contain:\n%s\n\nGot:\n%s", expected, html)
				}
			}
		})
	}
}

// TestMarkdownInterpolationWithFrontmatter tests that frontmatter data is accessible
func TestMarkdownInterpolationWithFrontmatter(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parsley-md-interp-fm-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	markdown := `---
title: My Report
year: 2024
total: 42000
---

# Report

Year: @{year}
Total: $@{total}
`

	mdPath := filepath.Join(tmpDir, "report.md")
	if err := os.WriteFile(mdPath, []byte(markdown), 0644); err != nil {
		t.Fatalf("Failed to write markdown file: %v", err)
	}

	testFilePath := filepath.Join(tmpDir, "test.pars")

	// Make frontmatter data available in environment
	code := `let year = 2024
let total = 42000
let mdfile = markdown(@./report.md)
let m = null
m <== mdfile
m.html`

	result := testEvalMDWithFilename(code, testFilePath)

	if result == nil {
		t.Fatalf("Result is nil")
	}

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("Evaluation error: %s", result.Inspect())
	}

	html := result.Inspect()

	expectedParts := []string{
		"Year: 2024",
		"Total: $42000",
	}

	for _, expected := range expectedParts {
		if !strings.Contains(html, expected) {
			t.Errorf("Expected HTML to contain:\n%s\n\nGot:\n%s", expected, html)
		}
	}
}

// TestMarkdownInterpolationInline tests that interpolation can be used inline with plain text
func TestMarkdownInterpolationInline(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parsley-md-interp-inline-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	markdown := `The sum of @{a} and @{b} is @{a + b}.`

	mdPath := filepath.Join(tmpDir, "math.md")
	if err := os.WriteFile(mdPath, []byte(markdown), 0644); err != nil {
		t.Fatalf("Failed to write markdown file: %v", err)
	}

	testFilePath := filepath.Join(tmpDir, "test.pars")

	code := `let a = 10
let b = 20
let mdfile = markdown(@./math.md)
let m = null
m <== mdfile
m.html`

	result := testEvalMDWithFilename(code, testFilePath)

	if result == nil {
		t.Fatalf("Result is nil")
	}

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("Evaluation error: %s", result.Inspect())
	}

	html := result.Inspect()

	expected := "<p>The sum of 10 and 20 is 30.</p>"
	if !strings.Contains(html, expected) {
		t.Errorf("Expected HTML to contain:\n%s\n\nGot:\n%s", expected, html)
	}
}
