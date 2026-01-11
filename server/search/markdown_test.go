package search

import (
	"strings"
	"testing"
	"time"
)

func TestProcessMarkdown(t *testing.T) {
	t.Run("markdown with frontmatter", func(t *testing.T) {
		content := `---
title: Getting Started
tags: [guide, tutorial]
date: 2026-01-09
---

# Getting Started with Basil

## Installation

Install Basil using:

` + "```bash\ngo install github.com/sambeau/basil\n```" + `

## Usage

Run the server with:

` + "```bash\nbasil serve\n```" + `

That's it!`

		mtime := time.Now()
		doc, err := ProcessMarkdown(content, "./docs/getting-started.md", mtime)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if doc.URL != "/docs/getting-started" {
			t.Errorf("expected URL '/docs/getting-started', got %q", doc.URL)
		}

		if doc.Title != "Getting Started" {
			t.Errorf("expected title 'Getting Started', got %q", doc.Title)
		}

		if len(doc.Tags) != 2 {
			t.Errorf("expected 2 tags, got %d", len(doc.Tags))
		}

		if !strings.Contains(doc.Content, "Install Basil") {
			t.Error("content missing expected text")
		}

		if strings.Contains(doc.Content, "```") {
			t.Error("code blocks should be stripped from content")
		}

		headings := strings.Split(doc.Headings, "\n")
		if len(headings) < 2 {
			t.Errorf("expected at least 2 headings, got %d", len(headings))
		}
	})

	t.Run("markdown without frontmatter uses first H1", func(t *testing.T) {
		content := `# My Document Title

This is some content.

## Section 1

More content here.`

		doc, err := ProcessMarkdown(content, "./docs/test.md", time.Now())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if doc.Title != "My Document Title" {
			t.Errorf("expected title 'My Document Title', got %q", doc.Title)
		}

		if doc.URL != "/docs/test" {
			t.Errorf("expected URL '/docs/test', got %q", doc.URL)
		}
	})

	t.Run("markdown without headings uses filename", func(t *testing.T) {
		content := `Just some plain content without any headings.`

		doc, err := ProcessMarkdown(content, "./docs/my-test-file.md", time.Now())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if doc.Title != "My Test File" {
			t.Errorf("expected title 'My Test File', got %q", doc.Title)
		}
	})

	t.Run("headings are extracted correctly", func(t *testing.T) {
		content := `# Main Title

## Section 1

### Subsection 1.1

#### Deep section

## Section 2

Content here.`

		doc, err := ProcessMarkdown(content, "./test.md", time.Now())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		headings := strings.Split(doc.Headings, "\n")
		expected := []string{"Main Title", "Section 1", "Subsection 1.1", "Deep section", "Section 2"}

		if len(headings) != len(expected) {
			t.Errorf("expected %d headings, got %d", len(expected), len(headings))
		}

		for i, exp := range expected {
			if i < len(headings) && headings[i] != exp {
				t.Errorf("heading %d: expected %q, got %q", i, exp, headings[i])
			}
		}
	})

	t.Run("path with underscores converts to title", func(t *testing.T) {
		content := `Some content without a title.`

		doc, err := ProcessMarkdown(content, "./docs/my_test_file_name.md", time.Now())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if doc.Title != "My Test File Name" {
			t.Errorf("expected title 'My Test File Name', got %q", doc.Title)
		}
	})

	t.Run("nested directory path generates correct URL", func(t *testing.T) {
		content := `# Test`

		doc, err := ProcessMarkdown(content, "./docs/guide/advanced/custom-handlers.md", time.Now())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if doc.URL != "/docs/guide/advanced/custom-handlers" {
			t.Errorf("expected URL '/docs/guide/advanced/custom-handlers', got %q", doc.URL)
		}
	})
}

func TestGenerateURL(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"./docs/guide.md", "/docs/guide"},
		{"docs/getting-started.md", "/docs/getting-started"},
		{"./README.md", "/README"},
		{"../other/file.md", "/other/file"},
		{"/absolute/path/file.md", "/absolute/path/file"},
		{"./docs/guide/advanced.md", "/docs/guide/advanced"},
		{"index.html", "/index"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := GenerateURL(tt.path)
			if result != tt.expected {
				t.Errorf("GenerateURL(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestExtractHeadings(t *testing.T) {
	t.Run("extracts all heading levels", func(t *testing.T) {
		content := `# H1 Title
## H2 Section
### H3 Subsection
#### H4 Deep
##### H5 Deeper
###### H6 Deepest`

		headings := ExtractHeadings(content)
		if len(headings) != 6 {
			t.Errorf("expected 6 headings, got %d", len(headings))
		}

		expected := []string{"H1 Title", "H2 Section", "H3 Subsection", "H4 Deep", "H5 Deeper", "H6 Deepest"}
		for i, exp := range expected {
			if i < len(headings) && headings[i] != exp {
				t.Errorf("heading %d: expected %q, got %q", i, exp, headings[i])
			}
		}
	})

	t.Run("strips formatting from headings", func(t *testing.T) {
		content := `# **Bold** Heading
## *Italic* Section
### ` + "`Code`" + ` in Heading
#### ~~Strikethrough~~ Text`

		headings := ExtractHeadings(content)
		if len(headings) != 4 {
			t.Errorf("expected 4 headings, got %d", len(headings))
		}

		if strings.Contains(headings[0], "**") || strings.Contains(headings[0], "*") {
			t.Error("formatting not stripped from heading")
		}
	})

	t.Run("handles content without headings", func(t *testing.T) {
		content := `Just some regular content without any headings.`

		headings := ExtractHeadings(content)
		if len(headings) != 0 {
			t.Errorf("expected 0 headings, got %d", len(headings))
		}
	})
}

func TestStripMarkdownForIndexing(t *testing.T) {
	t.Run("removes code blocks", func(t *testing.T) {
		content := "Some text\n```go\ncode here\n```\nMore text"

		result := StripMarkdownForIndexing(content)
		if strings.Contains(result, "```") || strings.Contains(result, "code here") {
			t.Error("code block not removed")
		}
		if !strings.Contains(result, "Some text") {
			t.Error("regular text was removed")
		}
	})

	t.Run("removes inline code", func(t *testing.T) {
		content := "Use the `function()` to do something."

		result := StripMarkdownForIndexing(content)
		if strings.Contains(result, "`") {
			t.Error("inline code markers not removed")
		}
	})

	t.Run("collapses multiple spaces", func(t *testing.T) {
		content := "Text    with     many     spaces"

		result := StripMarkdownForIndexing(content)
		if strings.Contains(result, "  ") {
			t.Error("multiple spaces not collapsed")
		}
	})

	t.Run("strips markdown formatting", func(t *testing.T) {
		content := "**bold** and *italic* and [link](url)"

		result := StripMarkdownForIndexing(content)
		if strings.Contains(result, "**") || strings.Contains(result, "*") {
			t.Error("markdown formatting not stripped")
		}
		if strings.Contains(result, "[") || strings.Contains(result, "]") {
			t.Error("link syntax not stripped")
		}
	})
}
