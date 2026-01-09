package search

import (
	"strings"
	"testing"
	"time"
)

func TestParseFrontmatter(t *testing.T) {
	t.Run("valid frontmatter with all fields", func(t *testing.T) {
		content := `---
title: Test Document
tags: [test, example, markdown]
date: 2026-01-09
authors: [John Doe, Jane Smith]
draft: false
---

# Content starts here

This is the main content.`

		fm, remaining, err := ParseFrontmatter(content)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if fm.Title != "Test Document" {
			t.Errorf("expected title 'Test Document', got %q", fm.Title)
		}

		if len(fm.Tags) != 3 {
			t.Errorf("expected 3 tags, got %d", len(fm.Tags))
		}
		if fm.Tags[0] != "test" || fm.Tags[1] != "example" || fm.Tags[2] != "markdown" {
			t.Errorf("unexpected tags: %v", fm.Tags)
		}

		expectedDate := time.Date(2026, 1, 9, 0, 0, 0, 0, time.UTC)
		if !fm.Date.Equal(expectedDate) {
			t.Errorf("expected date %v, got %v", expectedDate, fm.Date)
		}

		if len(fm.Authors) != 2 {
			t.Errorf("expected 2 authors, got %d", len(fm.Authors))
		}

		if fm.Draft {
			t.Error("expected draft to be false")
		}

		if !strings.Contains(remaining, "# Content starts here") {
			t.Error("remaining content missing expected text")
		}
	})

	t.Run("frontmatter with comma-separated tags", func(t *testing.T) {
		content := `---
title: Test
tags: tag1, tag2, tag3
---
Content`

		fm, _, err := ParseFrontmatter(content)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(fm.Tags) != 3 {
			t.Errorf("expected 3 tags, got %d", len(fm.Tags))
		}
		if fm.Tags[0] != "tag1" || fm.Tags[1] != "tag2" || fm.Tags[2] != "tag3" {
			t.Errorf("unexpected tags: %v", fm.Tags)
		}
	})

	t.Run("frontmatter with single author field", func(t *testing.T) {
		content := `---
title: Test
author: John Doe
---
Content`

		fm, _, err := ParseFrontmatter(content)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(fm.Authors) != 1 {
			t.Errorf("expected 1 author, got %d", len(fm.Authors))
		}
		if fm.Authors[0] != "John Doe" {
			t.Errorf("expected author 'John Doe', got %q", fm.Authors[0])
		}
	})

	t.Run("frontmatter with RFC3339 date", func(t *testing.T) {
		content := `---
title: Test
date: 2026-01-09T15:04:05Z
---
Content`

		fm, _, err := ParseFrontmatter(content)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedDate := time.Date(2026, 1, 9, 15, 4, 5, 0, time.UTC)
		if !fm.Date.Equal(expectedDate) {
			t.Errorf("expected date %v, got %v", expectedDate, fm.Date)
		}
	})

	t.Run("frontmatter with datetime", func(t *testing.T) {
		content := `---
title: Test
date: 2026-01-09 15:04:05
---
Content`

		fm, _, err := ParseFrontmatter(content)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if fm.Date.IsZero() {
			t.Error("expected date to be parsed")
		}
	})

	t.Run("missing frontmatter", func(t *testing.T) {
		content := `# Just a heading

Regular content without frontmatter.`

		fm, remaining, err := ParseFrontmatter(content)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if fm.Title != "" {
			t.Error("expected empty title for missing frontmatter")
		}
		if len(fm.Tags) != 0 {
			t.Error("expected no tags for missing frontmatter")
		}

		if remaining != content {
			t.Error("remaining content should equal original content")
		}
	})

	t.Run("invalid YAML - malformed", func(t *testing.T) {
		content := `---
title: Test
tags: [missing closing bracket
date: 2026-01-09
---
Content`

		fm, remaining, err := ParseFrontmatter(content)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should return empty frontmatter on parse error
		if fm.Title != "" {
			t.Error("expected empty title for invalid YAML")
		}

		// Should return full content including invalid frontmatter
		if remaining != content {
			t.Error("should return original content for invalid YAML")
		}
	})

	t.Run("frontmatter without closing delimiter", func(t *testing.T) {
		content := `---
title: Test
tags: [test]

Content without closing delimiter`

		fm, remaining, err := ParseFrontmatter(content)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should treat as no frontmatter
		if fm.Title != "" {
			t.Error("expected empty title when closing delimiter missing")
		}

		if remaining != content {
			t.Error("should return original content")
		}
	})

	t.Run("minimal frontmatter - only title", func(t *testing.T) {
		content := `---
title: Minimal Document
---
Content here`

		fm, remaining, err := ParseFrontmatter(content)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if fm.Title != "Minimal Document" {
			t.Errorf("expected title 'Minimal Document', got %q", fm.Title)
		}

		if len(fm.Tags) != 0 {
			t.Error("expected no tags")
		}

		if !strings.Contains(remaining, "Content here") {
			t.Error("remaining content missing")
		}
	})

	t.Run("empty frontmatter block", func(t *testing.T) {
		content := `---
---
Content`

		fm, remaining, err := ParseFrontmatter(content)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if fm.Title != "" {
			t.Error("expected empty title")
		}

		if !strings.Contains(remaining, "Content") {
			t.Error("remaining content missing")
		}
	})

	t.Run("frontmatter with Windows line endings", func(t *testing.T) {
		content := "---\r\ntitle: Test\r\ntags: [a, b]\r\n---\r\nContent"

		fm, remaining, err := ParseFrontmatter(content)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if fm.Title != "Test" {
			t.Errorf("expected title 'Test', got %q", fm.Title)
		}

		if !strings.Contains(remaining, "Content") {
			t.Error("remaining content missing")
		}
	})
}
